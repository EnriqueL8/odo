package component

import (
	"fmt"
	"path/filepath"

	"github.com/openshift/odo/pkg/envinfo"
	projectCmd "github.com/openshift/odo/pkg/odo/cli/project"
	"github.com/openshift/odo/pkg/odo/genericclioptions"
	"github.com/openshift/odo/pkg/odo/util/completion"
	"github.com/openshift/odo/pkg/odo/util/experimental"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	odoutil "github.com/openshift/odo/pkg/odo/util"

	ktemplates "k8s.io/kubectl/pkg/util/templates"
)

// TODO: add CLI Reference doc
var deployDeleteCmdExample = ktemplates.Examples(`  # Deletes deployment from Kubernetes cluster
  `)

// DeployDeleteRecommendedCommandName is the recommended build command name
const DeployDeleteRecommendedCommandName = "delete"

// DeployDeleteOptions encapsulates options that deploy delete command uses
type DeployDeleteOptions struct {
	*CommonPushOptions

	// devfile path
	DevfilePath    string
	DockerfilePath string
	namespace      string
	tag            string
	ManifestSource []byte
}

// NewDeployDeleteOptions returns new instance of DeployDeleteOptions
// with "default" values for certain values, for example, show is "false"
func NewDeployDeleteOptions() *DeployDeleteOptions {
	return &DeployDeleteOptions{
		CommonPushOptions: NewCommonPushOptions(),
	}
}

// Complete completes push args
func (do *DeployDeleteOptions) Complete(name string, cmd *cobra.Command, args []string) (err error) {
	do.DevfilePath = filepath.Join(do.componentContext, do.DevfilePath)
	envInfo, err := envinfo.NewEnvSpecificInfo(do.componentContext)
	if err != nil {
		return errors.Wrap(err, "unable to retrieve configuration information")
	}
	do.EnvSpecificInfo = envInfo
	do.Context = genericclioptions.NewDevfileContext(cmd)

	return nil
}

// Validate validates the push parameters
func (do *DeployDeleteOptions) Validate() (err error) {
	// TODO: Validate the value of tag and any user parameteres.
	return
}

// Run has the logic to perform the required actions as part of command
func (do *DeployDeleteOptions) Run() (err error) {
	fmt.Println("£££££££ IN DEELETEEEEE - RUN %%%%%%%")

	// do.componentContext, .odo. manifest.yaml
	// TODO: Check manifest is actually there!!!
	err = do.DevfileDeployDelete()
	if err != nil {
		return err
	}

	return nil
}

// NewCmdDeploy implements the push odo command
func NewCmdDeployDelete(name, fullName string) *cobra.Command {
	do := NewDeployDeleteOptions()

	var deployDeleteCmd = &cobra.Command{
		Use:         fmt.Sprintf("%s [component name]", name),
		Short:       "Delete deployed component",
		Long:        "Delete deployed component",
		Example:     fmt.Sprintf(deployDeleteCmdExample, fullName),
		Args:        cobra.NoArgs,
		Annotations: map[string]string{"command": "deploy"},
		Run: func(cmd *cobra.Command, args []string) {
			genericclioptions.GenericRun(do, cmd, args)
		},
	}
	genericclioptions.AddContextFlag(deployDeleteCmd, &do.componentContext)

	// enable devfile flag if experimental mode is enabled
	if experimental.IsExperimentalModeEnabled() {
		deployDeleteCmd.Flags().StringVar(&do.DevfilePath, "devfile", "./devfile.yaml", "Path to a devfile.yaml")
	}

	//Adding `--project` flag
	projectCmd.AddProjectFlag(deployDeleteCmd)

	deployDeleteCmd.SetUsageTemplate(odoutil.CmdUsageTemplate)
	completion.RegisterCommandHandler(deployDeleteCmd, completion.ComponentNameCompletionHandler)
	completion.RegisterCommandFlagHandler(deployDeleteCmd, "context", completion.FileCompletionHandler)

	return deployDeleteCmd
}
