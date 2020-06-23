package component

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"

	"k8s.io/client-go/dynamic"

	"github.com/fatih/color"
	"github.com/pkg/errors"
	"k8s.io/klog"

	"github.com/openshift/odo/pkg/component"
	"github.com/openshift/odo/pkg/config"
	"github.com/openshift/odo/pkg/devfile/adapters/common"
	adaptersCommon "github.com/openshift/odo/pkg/devfile/adapters/common"
	"github.com/openshift/odo/pkg/devfile/adapters/kubernetes/storage"
	"github.com/openshift/odo/pkg/devfile/adapters/kubernetes/utils"
	versionsCommon "github.com/openshift/odo/pkg/devfile/parser/data/common"
	"github.com/openshift/odo/pkg/envinfo"
	"github.com/openshift/odo/pkg/exec"
	"github.com/openshift/odo/pkg/kclient"
	"github.com/openshift/odo/pkg/log"
	odoutil "github.com/openshift/odo/pkg/odo/util"
	"github.com/openshift/odo/pkg/sync"
)

// New instantiantes a component adapter
func New(adapterContext common.AdapterContext, client kclient.Client) Adapter {
	return Adapter{
		Client:         client,
		AdapterContext: adapterContext,
	}
}

// Adapter is a component adapter implementation for Kubernetes
type Adapter struct {
	Client kclient.Client
	common.AdapterContext
	devfileInitCmd  string
	devfileBuildCmd string
	devfileRunCmd   string
}

func (a Adapter) generateBuildContainer(containerName, imageTag string) corev1.Container {
	buildImage := "quay.io/buildah/stable:latest"

	// TODO(Optional): Init container before the buildah bud to copy over the files.
	//command := []string{"buildah"}
	//commandArgs := []string{"bud"}
	command := []string{"tail"}
	commandArgs := []string{"-f", "/dev/null"}

	// TODO: Edit dockerfile env value if mounting it sometwhere else
	envVars := []corev1.EnvVar{
		{Name: "Tag", Value: imageTag},
	}

	isPrivileged := true
	resourceReqs := corev1.ResourceRequirements{}

	container := kclient.GenerateContainer(containerName, buildImage, isPrivileged, command, commandArgs, envVars, resourceReqs, nil)

	container.VolumeMounts = []corev1.VolumeMount{
		{Name: "varlibcontainers", MountPath: "/var/lib/containers"},
		{Name: kclient.OdoSourceVolume, MountPath: kclient.OdoSourceVolumeMount},
	}

	return *container
}

func (a Adapter) createBuildDeployment(labels map[string]string, container corev1.Container) (err error) {

	objectMeta := kclient.CreateObjectMeta(a.ComponentName, a.Client.Namespace, labels, nil)
	podTemplateSpec := kclient.GeneratePodTemplateSpec(objectMeta, []corev1.Container{container})

	// TODO: For openshift, need to specify a service account that allows priviledged containers
	saEnv := os.Getenv("BUILD_SERVICE_ACCOUNT")
	if saEnv != "" {
		podTemplateSpec.Spec.ServiceAccountName = saEnv
	}

	libContainersVolume := corev1.Volume{
		Name: "varlibcontainers",
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	}

	podTemplateSpec.Spec.Volumes = append(podTemplateSpec.Spec.Volumes, libContainersVolume)

	deploymentSpec := kclient.GenerateDeploymentSpec(*podTemplateSpec)
	klog.V(3).Infof("Creating deployment %v", deploymentSpec.Template.GetName())

	_, err = a.Client.CreateDeployment(*deploymentSpec)
	if err != nil {
		return err
	}
	klog.V(3).Infof("Successfully created component %v", deploymentSpec.Template.GetName())

	return nil
}

func (a Adapter) executeBuildAndPush(syncFolder string, imageTag string, compInfo common.ComponentInfo) (err error) {
	// Running buildah bud and buildah push
	buildahBud := "buildah bud -f ./Dockerfile -t $Tag ."
	command := []string{adaptersCommon.ShellExecutable, "-c", "cd " + syncFolder + " && " + buildahBud}

	// TODO: Add spinner
	err = exec.ExecuteCommand(&a.Client, compInfo, command, false)
	if err != nil {
		return errors.Wrapf(err, "failed to build image for component with name: %s", a.ComponentName)
	}

	values := strings.Split(imageTag, "/")
	tag := imageTag
	buildahPush := "buildah push "

	// Need to change this IF to be more robust
	if len(values) == 3 && strings.Contains(values[0], "openshift") {
		// This needs a valid service account: e.g builder for openshift
		// --creds flag arg has the format username:password
		// we want to use serviceaccount:token
		buildahPush += "--creds "
		saEnv := os.Getenv("BUILD_SERVICE_ACCOUNT")
		if saEnv != "" {
			buildahPush += saEnv
		} else {
			buildahBud += "dummy-username"
		}
		buildahPush += ":$(cat /var/run/secrets/kubernetes.io/serviceaccount/token) "
	}

	//TODO: handle dockerhub case and creds!!
	buildahPush += "--tls-verify=false " + tag + " docker://" + tag
	command = []string{adaptersCommon.ShellExecutable, "-c", buildahPush}

	//TODO: Add Spinner
	err = exec.ExecuteCommand(&a.Client, compInfo, command, false)
	if err != nil {
		return errors.Wrapf(err, "failed to push build image to the registry for component with name: %s", a.ComponentName)
	}

	return nil
}

// Build image for devfile project
func (a Adapter) Build(parameters common.BuildParameters) (err error) {
	containerName := a.ComponentName + "-container"
	buildContainer := a.generateBuildContainer(containerName, parameters.Tag)
	labels := map[string]string{
		"component": a.ComponentName,
	}

	err = a.createBuildDeployment(labels, buildContainer)
	if err != nil {
		return errors.Wrap(err, "error while creating buildah deployment")
	}

	// Delete deployment
	defer func() {
		derr := a.Delete(labels)
		if err == nil {
			err = errors.Wrapf(derr, "failed to delete build step for component with name: %s", a.ComponentName)
		}

	}()

	_, err = a.Client.WaitForDeploymentRollout(a.ComponentName)
	if err != nil {
		return errors.Wrap(err, "error while waiting for deployment rollout")
	}

	// Wait for Pod to be in running state otherwise we can't sync data or exec commands to it.
	pod, err := a.waitAndGetComponentPod(false)
	if err != nil {
		return errors.Wrapf(err, "unable to get pod for component %s", a.ComponentName)
	}

	// Need to wait for container to start
	time.Sleep(10 * time.Second)

	// Sync files to volume
	log.Infof("\nSyncing to component %s", a.ComponentName)
	// Get a sync adapter. Check if project files have changed and sync accordingly
	syncAdapter := sync.New(a.AdapterContext, &a.Client)
	compInfo := common.ComponentInfo{
		ContainerName: containerName,
		PodName:       pod.GetName(),
	}

	syncFolder, err := syncAdapter.SyncFilesBuild(parameters, compInfo)

	if err != nil {
		return errors.Wrapf(err, "failed to sync to component with name %s", a.ComponentName)
	}

	err = a.executeBuildAndPush(syncFolder, parameters.Tag, compInfo)
	if err != nil {
		return err
	}

	return
}

func determinePort(envSpecificInfo envinfo.EnvSpecificInfo) string {
	// TODO: Determine port to use (from env.yaml or other location!!)
	deploymentPort := ""
	for _, localURL := range envSpecificInfo.GetURL() {
		if localURL.Kind != envinfo.DOCKER {
			deploymentPort = strconv.Itoa(localURL.Port)
			break
		}
	}
	return deploymentPort
}

func substitueYamlVariables(baseYaml []byte, yamlSubstitutions map[string]string) []byte {
	// TODO: Provide a better way to do the substitution in the manifest file(s)
	for key, value := range yamlSubstitutions {
		if value != "" && bytes.Contains(baseYaml, []byte(key)) {
			klog.V(3).Infof("Replacing %s with %s", key, value)
			tempYaml := bytes.ReplaceAll(baseYaml, []byte(key), []byte(value))
			baseYaml = tempYaml
		}
	}
	return baseYaml
}

// Build image for devfile project
func (a Adapter) Deploy(parameters common.DeployParameters) (err error) {
	namespace := a.Client.Namespace
	applicationName := a.ComponentName + "-deploy"
	deploymentManifest := &unstructured.Unstructured{}

	log.Info("\nDeploying manifest")
	// TODO: Work out how to correctly handle spinners
	s := log.Spinner("Deploying the manifest")

	// Specify the substitution keys and values
	yamlSubstitutions := map[string]string{
		// TODO: this tag is not passed to delete, do we need to template
		"CONTAINER_IMAGE": parameters.Tag,
		"PROJECT_NAME":    applicationName,
		"PORT":            determinePort(parameters.EnvSpecificInfo),
	}

	// Substitute the values in the manifest file
	deployYaml := substitueYamlVariables(parameters.ManifestSource, yamlSubstitutions)

	// Build a yaml decoder with the unstructured Scheme
	yamlDecoder := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)

	_, gvk, err := yamlDecoder.Decode([]byte(deployYaml), nil, deploymentManifest)
	if err != nil {
		return err
	}

	klog.V(3).Infof("Deploy manifest:\n\n%s", deploymentManifest)
	gvr := schema.GroupVersionResource{Group: gvk.Group, Version: gvk.Version, Resource: strings.ToLower(gvk.Kind + "s")}
	klog.V(3).Infof("Manifest type: %s", gvr.String())

	labels := map[string]string{
		"component": applicationName,
	}

	manifestLabels := deploymentManifest.GetLabels()
	if manifestLabels != nil {
		for key, value := range labels {
			manifestLabels[key] = value
		}
		deploymentManifest.SetLabels(manifestLabels)
	} else {
		deploymentManifest.SetLabels(labels)
	}

	// TODO: Determine why using a.Client.DynamicClient doesnt work
	// Need to create my own client in order to get the dynamic parts working
	myclient, err := dynamic.NewForConfig(a.Client.KubeClientConfig)
	if err != nil {
		return err
	}

	// Check to see whether deployed resource already exists. If not, create else update
	// Get?
	instanceFound := false
	list, err := myclient.Resource(gvr).Namespace(namespace).List(metav1.ListOptions{})
	if list != nil && len(list.Items) > 0 {
		for _, item := range list.Items {
			klog.V(3).Infof("Found %s %s with resourceVersion: %s.\n", gvk.Kind, item.GetName(), item.GetResourceVersion())
			if item.GetName() == applicationName {
				deploymentManifest.SetResourceVersion(item.GetResourceVersion())
				instanceFound = true
			}
		}
	}

	result := &unstructured.Unstructured{}
	if !instanceFound {
		// Create Deployment
		log.Infof("Creating %s...", gvk.Kind)
		result, err = myclient.Resource(gvr).Namespace(namespace).Create(deploymentManifest, metav1.CreateOptions{})
		//	result, err := a.Client.DynamicClient.Resource(gvr).Namespace(namespace).Create(deploymentManifest, metav1.CreateOptions{})
	} else {
		// Update Deployment
		log.Infof("Updating %s...", gvk.Kind)
		result, err = myclient.Resource(gvr).Namespace(namespace).Update(deploymentManifest, metav1.UpdateOptions{})
	}

	if err != nil {
		s.End(false)
		return errors.Wrap(err, "failed to deploy "+gvk.Kind)
	}

	s.End(true)
	log.Infof("Deployed %s %s.\n", gvk.Kind, result.GetName())

	// This will override if manifest.yaml is present
	manifestFile, err := os.Create(filepath.Join(a.Context, ".odo", "manifest.yaml"))
	if err != nil {
		err = manifestFile.Close()
		return err
	}
	err = yamlDecoder.Encode(result, manifestFile)

	if err != nil {
		err = manifestFile.Close()
		return err
	}

	err = manifestFile.Close()
	return err
}

func (a Adapter) DeployDelete(manifest []byte) (err error) {
	deploymentManifest := &unstructured.Unstructured{}
	// Build a yaml decoder with the unstructured Scheme
	yamlDecoder := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
	manifests := bytes.Split(manifest, []byte("---"))
	for _, splitManifest := range manifests {
		if len(manifest) > 0 {
			_, gvk, err := yamlDecoder.Decode([]byte(splitManifest), nil, deploymentManifest)
			if err != nil {
				return err
			}
			klog.V(3).Infof("Deploy manifest:\n\n%s", deploymentManifest)
			gvr := schema.GroupVersionResource{Group: gvk.Group, Version: gvk.Version, Resource: strings.ToLower(gvk.Kind + "s")}
			klog.V(3).Infof("Manifest type: %s", gvr.String())
			// TODO: Determine why using a.Client.DynamicClient doesnt work
			// Need to create my own client in order to get the dynamic parts working
			myclient, err := dynamic.NewForConfig(a.Client.KubeClientConfig)
			if err != nil {
				return err
			}
			// TODO: Check if resource is running in the cluster before attempting to delete Warning
			deployment, err := myclient.Resource(gvr).Namespace(a.Client.Namespace).Get(deploymentManifest.GetName(), metav1.GetOptions{})
			if err != nil {
				return err
			}
			if deployment == nil {
				log.Warningf("Could not delete deployment %v, as deployment was not found", deploymentManifest.GetName())
			}
			err = myclient.Resource(gvr).Namespace(a.Client.Namespace).Delete(deploymentManifest.GetName(), &metav1.DeleteOptions{})
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// Push updates the component if a matching component exists or creates one if it doesn't exist
// Once the component has started, it will sync the source code to it.
func (a Adapter) Push(parameters common.PushParameters) (err error) {
	componentExists := utils.ComponentExists(a.Client, a.ComponentName)

	a.devfileInitCmd = parameters.DevfileInitCmd
	a.devfileBuildCmd = parameters.DevfileBuildCmd
	a.devfileRunCmd = parameters.DevfileRunCmd

	podChanged := false
	var podName string

	// If the component already exists, retrieve the pod's name before it's potentially updated
	if componentExists {
		pod, err := a.waitAndGetComponentPod(true)
		if err != nil {
			return errors.Wrapf(err, "unable to get pod for component %s", a.ComponentName)
		}
		podName = pod.GetName()
	}

	// Validate the devfile build and run commands
	log.Info("\nValidation")
	s := log.Spinner("Validating the devfile")
	pushDevfileCommands, err := common.ValidateAndGetPushDevfileCommands(a.Devfile.Data, a.devfileInitCmd, a.devfileBuildCmd, a.devfileRunCmd)
	if err != nil {
		s.End(false)
		return errors.Wrap(err, "failed to validate devfile build and run commands")
	}
	s.End(true)

	log.Infof("\nCreating Kubernetes resources for component %s", a.ComponentName)
	err = a.createOrUpdateComponent(componentExists)
	if err != nil {
		return errors.Wrap(err, "unable to create or update component")
	}

	_, err = a.Client.WaitForDeploymentRollout(a.ComponentName)
	if err != nil {
		return errors.Wrap(err, "error while waiting for deployment rollout")
	}

	// Wait for Pod to be in running state otherwise we can't sync data or exec commands to it.
	pod, err := a.waitAndGetComponentPod(true)
	if err != nil {
		return errors.Wrapf(err, "unable to get pod for component %s", a.ComponentName)
	}

	err = component.ApplyConfig(nil, &a.Client, config.LocalConfigInfo{}, parameters.EnvSpecificInfo, color.Output, componentExists)
	if err != nil {
		odoutil.LogErrorAndExit(err, "Failed to update config to component deployed.")
	}

	// Compare the name of the pod with the one before the rollout. If they differ, it means there's a new pod and a force push is required
	if componentExists && podName != pod.GetName() {
		podChanged = true
	}

	// Find at least one pod with the source volume mounted, error out if none can be found
	containerName, err := getFirstContainerWithSourceVolume(pod.Spec.Containers)
	if err != nil {
		return errors.Wrapf(err, "error while retrieving container from pod %s with a mounted project volume", podName)
	}

	log.Infof("\nSyncing to component %s", a.ComponentName)
	// Get a sync adapter. Check if project files have changed and sync accordingly
	syncAdapter := sync.New(a.AdapterContext, &a.Client)
	compInfo := common.ComponentInfo{
		ContainerName: containerName,
		PodName:       pod.GetName(),
	}
	syncParams := adaptersCommon.SyncParameters{
		PushParams:      parameters,
		CompInfo:        compInfo,
		ComponentExists: componentExists,
		PodChanged:      podChanged,
	}
	execRequired, err := syncAdapter.SyncFiles(syncParams)
	if err != nil {
		return errors.Wrapf(err, "Failed to sync to component with name %s", a.ComponentName)
	}

	if execRequired {
		log.Infof("\nExecuting devfile commands for component %s", a.ComponentName)
		err = a.execDevfile(pushDevfileCommands, componentExists, parameters.Show, pod.GetName(), pod.Spec.Containers)
		if err != nil {
			return err
		}
	}

	return nil
}

// DoesComponentExist returns true if a component with the specified name exists, false otherwise
func (a Adapter) DoesComponentExist(cmpName string) bool {
	return utils.ComponentExists(a.Client, cmpName)
}

func (a Adapter) createOrUpdateComponent(componentExists bool) (err error) {
	componentName := a.ComponentName

	labels := map[string]string{
		"component": componentName,
	}

	containers, err := utils.GetContainers(a.Devfile)
	if err != nil {
		return err
	}

	if len(containers) == 0 {
		return fmt.Errorf("No valid components found in the devfile")
	}

	containers, err = utils.UpdateContainersWithSupervisord(a.Devfile, containers, a.devfileRunCmd)
	if err != nil {
		return err
	}

	objectMeta := kclient.CreateObjectMeta(componentName, a.Client.Namespace, labels, nil)
	podTemplateSpec := kclient.GeneratePodTemplateSpec(objectMeta, containers)

	kclient.AddBootstrapSupervisordInitContainer(podTemplateSpec)

	componentAliasToVolumes := adaptersCommon.GetVolumes(a.Devfile)

	var uniqueStorages []common.Storage
	volumeNameToPVCName := make(map[string]string)
	processedVolumes := make(map[string]bool)

	// Get a list of all the unique volume names and generate their PVC names
	for _, volumes := range componentAliasToVolumes {
		for _, vol := range volumes {
			if _, ok := processedVolumes[vol.Name]; !ok {
				processedVolumes[vol.Name] = true

				// Generate the PVC Names
				klog.V(3).Infof("Generating PVC name for %v", vol.Name)
				generatedPVCName, err := storage.GeneratePVCNameFromDevfileVol(vol.Name, componentName)
				if err != nil {
					return err
				}

				// Check if we have an existing PVC with the labels, overwrite the generated name with the existing name if present
				existingPVCName, err := storage.GetExistingPVC(&a.Client, vol.Name, componentName)
				if err != nil {
					return err
				}
				if len(existingPVCName) > 0 {
					klog.V(3).Infof("Found an existing PVC for %v, PVC %v will be re-used", vol.Name, existingPVCName)
					generatedPVCName = existingPVCName
				}

				pvc := common.Storage{
					Name:   generatedPVCName,
					Volume: vol,
				}
				uniqueStorages = append(uniqueStorages, pvc)
				volumeNameToPVCName[vol.Name] = generatedPVCName
			}
		}
	}

	// Add PVC and Volume Mounts to the podTemplateSpec
	err = kclient.AddPVCAndVolumeMount(podTemplateSpec, volumeNameToPVCName, componentAliasToVolumes)
	if err != nil {
		return err
	}

	deploymentSpec := kclient.GenerateDeploymentSpec(*podTemplateSpec)
	var containerPorts []corev1.ContainerPort
	for _, c := range deploymentSpec.Template.Spec.Containers {
		if len(containerPorts) == 0 {
			containerPorts = c.Ports
		} else {
			containerPorts = append(containerPorts, c.Ports...)
		}
	}
	serviceSpec := kclient.GenerateServiceSpec(objectMeta.Name, containerPorts)
	klog.V(3).Infof("Creating deployment %v", deploymentSpec.Template.GetName())
	klog.V(3).Infof("The component name is %v", componentName)

	if utils.ComponentExists(a.Client, componentName) {
		// If the component already exists, get the resource version of the deploy before updating
		klog.V(3).Info("The component already exists, attempting to update it")
		deployment, err := a.Client.UpdateDeployment(*deploymentSpec)
		if err != nil {
			return err
		}
		klog.V(3).Infof("Successfully updated component %v", componentName)
		oldSvc, err := a.Client.KubeClient.CoreV1().Services(a.Client.Namespace).Get(componentName, metav1.GetOptions{})
		objectMetaTemp := objectMeta
		ownerReference := kclient.GenerateOwnerReference(deployment)
		objectMetaTemp.OwnerReferences = append(objectMeta.OwnerReferences, ownerReference)
		if err != nil {
			// no old service was found, create a new one
			if len(serviceSpec.Ports) > 0 {
				_, err = a.Client.CreateService(objectMetaTemp, *serviceSpec)
				if err != nil {
					return err
				}
				klog.V(3).Infof("Successfully created Service for component %s", componentName)
			}
		} else {
			if len(serviceSpec.Ports) > 0 {
				serviceSpec.ClusterIP = oldSvc.Spec.ClusterIP
				objectMetaTemp.ResourceVersion = oldSvc.GetResourceVersion()
				_, err = a.Client.UpdateService(objectMetaTemp, *serviceSpec)
				if err != nil {
					return err
				}
				klog.V(3).Infof("Successfully update Service for component %s", componentName)
			} else {
				err = a.Client.KubeClient.CoreV1().Services(a.Client.Namespace).Delete(componentName, &metav1.DeleteOptions{})
				if err != nil {
					return err
				}
			}
		}
	} else {
		deployment, err := a.Client.CreateDeployment(*deploymentSpec)
		if err != nil {
			return err
		}
		klog.V(3).Infof("Successfully created component %v", componentName)
		ownerReference := kclient.GenerateOwnerReference(deployment)
		objectMetaTemp := objectMeta
		objectMetaTemp.OwnerReferences = append(objectMeta.OwnerReferences, ownerReference)
		if len(serviceSpec.Ports) > 0 {
			_, err = a.Client.CreateService(objectMetaTemp, *serviceSpec)
			if err != nil {
				return err
			}
			klog.V(3).Infof("Successfully created Service for component %s", componentName)
		}

	}

	// Get the storage adapter and create the volumes if it does not exist
	stoAdapter := storage.New(a.AdapterContext, a.Client)
	err = stoAdapter.Create(uniqueStorages)
	if err != nil {
		return err
	}

	return nil
}

func (a Adapter) waitAndGetComponentPod(hideSpinner bool) (*corev1.Pod, error) {
	podSelector := fmt.Sprintf("component=%s", a.ComponentName)
	watchOptions := metav1.ListOptions{
		LabelSelector: podSelector,
	}
	// Wait for Pod to be in running state otherwise we can't sync data to it.
	pod, err := a.Client.WaitAndGetPod(watchOptions, corev1.PodRunning, "Waiting for component to start", hideSpinner)
	if err != nil {
		return nil, errors.Wrapf(err, "error while waiting for pod %s", podSelector)
	}
	return pod, nil
}

// Executes all the commands from the devfile in order: init and build - which are both optional, and a compulsary run.
// Init only runs once when the component is created.
func (a Adapter) execDevfile(commandsMap common.PushCommandsMap, componentExists, show bool, podName string, containers []corev1.Container) (err error) {
	// If nothing has been passed, then the devfile is missing the required run command
	if len(commandsMap) == 0 {
		return errors.New(fmt.Sprint("error executing devfile commands - there should be at least 1 command"))
	}

	compInfo := common.ComponentInfo{
		PodName: podName,
	}

	// only execute Init command, if it is first run of container.
	if !componentExists {
		// Get Init Command
		command, ok := commandsMap[versionsCommon.InitCommandGroupType]
		if ok {
			compInfo.ContainerName = command.Exec.Component
			err = exec.ExecuteDevfileBuildAction(&a.Client, *command.Exec, command.Exec.Id, compInfo, show)
			if err != nil {
				return err
			}

		}

	}

	// Get Build Command
	command, ok := commandsMap[versionsCommon.BuildCommandGroupType]
	if ok {
		compInfo.ContainerName = command.Exec.Component
		err = exec.ExecuteDevfileBuildAction(&a.Client, *command.Exec, command.Exec.Id, compInfo, show)
		if err != nil {
			return err
		}
	}

	// Get Run Command
	command, ok = commandsMap[versionsCommon.RunCommandGroupType]
	if ok {
		klog.V(4).Infof("Executing devfile command %v", command.Exec.Id)
		compInfo.ContainerName = command.Exec.Component

		// Check if the devfile run component containers have supervisord as the entrypoint.
		// Start the supervisord if the odo component does not exist
		if !componentExists {
			err = a.InitRunContainerSupervisord(command.Exec.Component, podName, containers)
			if err != nil {
				return
			}
		}

		if componentExists && !common.IsRestartRequired(command) {
			klog.V(4).Infof("restart:false, Not restarting DevRun Command")
			err = exec.ExecuteDevfileRunActionWithoutRestart(&a.Client, *command.Exec, command.Exec.Id, compInfo, show)
			return
		}
		err = exec.ExecuteDevfileRunAction(&a.Client, *command.Exec, command.Exec.Id, compInfo, show)

	}

	return
}

// InitRunContainerSupervisord initializes the supervisord in the container if
// the container has entrypoint that is not supervisord
func (a Adapter) InitRunContainerSupervisord(containerName, podName string, containers []corev1.Container) (err error) {
	for _, container := range containers {
		if container.Name == containerName && !reflect.DeepEqual(container.Command, []string{common.SupervisordBinaryPath}) {
			command := []string{common.SupervisordBinaryPath, "-c", common.SupervisordConfFile, "-d"}
			compInfo := common.ComponentInfo{
				ContainerName: containerName,
				PodName:       podName,
			}
			err = exec.ExecuteCommand(&a.Client, compInfo, command, true)
		}
	}

	return
}

// getFirstContainerWithSourceVolume returns the first container that set mountSources: true
// Because the source volume is shared across all components that need it, we only need to sync once,
// so we only need to find one container. If no container was found, that means there's no
// container to sync to, so return an error
func getFirstContainerWithSourceVolume(containers []corev1.Container) (string, error) {
	for _, c := range containers {
		for _, vol := range c.VolumeMounts {
			if vol.Name == kclient.OdoSourceVolume {
				return c.Name, nil
			}
		}
	}

	return "", fmt.Errorf("In order to sync files, odo requires at least one component in a devfile to set 'mountSources: true'")
}

// Delete deletes the component
func (a Adapter) Delete(labels map[string]string) error {
	if !utils.ComponentExists(a.Client, a.ComponentName) {
		return errors.Errorf("the component %s doesn't exist on the cluster", a.ComponentName)
	}

	return a.Client.DeleteDeployment(labels)
}
