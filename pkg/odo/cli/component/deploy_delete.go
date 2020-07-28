package component

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/openshift/odo/pkg/envinfo"
	projectCmd "github.com/openshift/odo/pkg/odo/cli/project"
	"github.com/openshift/odo/pkg/odo/genericclioptions"
	"github.com/openshift/odo/pkg/odo/util/completion"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	odoutil "github.com/openshift/odo/pkg/odo/util"

	ktemplates "k8s.io/kubectl/pkg/util/templates"
)

// Constant for manifest
const manifestFile = ".odo/manifest.yaml"

// TODO: add CLI Reference doc
var deployDeleteCmdExample = ktemplates.Examples(`  # Delete deployed component 
%[1]s
  `)

// DeployDeleteRecommendedCommandName is the recommended build command name
const DeployDeleteRecommendedCommandName = "delete"

// DeployDeleteOptions encapsulates options that deploy delete command uses
type DeployDeleteOptions struct {
	componentContext string
	EnvSpecificInfo  *envinfo.EnvSpecificInfo

	DevfilePath    string
	namespace      string
	ManifestPath   string
	ManifestSource []byte

	*genericclioptions.Context
}

// NewDeployDeleteOptions returns new instance of DeployDeleteOptions
// with "default" values for certain values, for example, show is "false"
func NewDeployDeleteOptions() *DeployDeleteOptions {
	return &DeployDeleteOptions{}
}

// CompleteDevfilePath completes the devfile path from context
func (ddo *DeployDeleteOptions) CompleteDevfilePath() {
	if len(ddo.DevfilePath) > 0 {
		ddo.DevfilePath = filepath.Join(ddo.componentContext, ddo.DevfilePath)
	} else {
		ddo.DevfilePath = filepath.Join(ddo.componentContext, "devfile.yaml")
	}
}

// Complete completes push args
func (ddo *DeployDeleteOptions) Complete(name string, cmd *cobra.Command, args []string) (err error) {
	ddo.CompleteDevfilePath()
	envInfo, err := envinfo.NewEnvSpecificInfo(ddo.componentContext)
	if err != nil {
		return errors.Wrap(err, "unable to retrieve configuration information")
	}
	ddo.EnvSpecificInfo = envInfo
	ddo.Context = genericclioptions.NewDevfileContext(cmd)

	return nil
}

// Validate validates the push parameters
func (ddo *DeployDeleteOptions) Validate() (err error) {
	// ddo.componentContext, .odo, manifest.yaml
	// TODO: Check manifest is actually there!!!
	// read bytes into deployDeleteOptions
	if _, err := os.Stat(manifestFile); os.IsNotExist(err) {
		return errors.Wrap(err, "manifest file at "+manifestFile+" does not exist")
	}

	ddo.ManifestSource, err = ioutil.ReadFile(manifestFile)
	if err != nil {
		return err
	}
	ddo.ManifestPath = manifestFile
	return
}

// Run has the logic to perform the required actions as part of command
func (ddo *DeployDeleteOptions) Run() (err error) {
	err = ddo.DevfileDeployDelete()
	if err != nil {
		return err
	}

	return nil
}

// NewCmdDeploy implements the push odo command
func NewCmdDeployDelete(name, fullName string) *cobra.Command {
	ddo := NewDeployDeleteOptions()

	var deployDeleteCmd = &cobra.Command{
		Use:     name,
		Short:   "Delete deployed component",
		Long:    "Delete deployed component",
		Example: fmt.Sprintf(deployDeleteCmdExample, fullName),
		Args:    cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			genericclioptions.GenericRun(ddo, cmd, args)
		},
	}
	genericclioptions.AddContextFlag(deployDeleteCmd, &ddo.componentContext)

	//Adding `--project` flag
	projectCmd.AddProjectFlag(deployDeleteCmd)

	deployDeleteCmd.SetUsageTemplate(odoutil.CmdUsageTemplate)
	completion.RegisterCommandHandler(deployDeleteCmd, completion.ComponentNameCompletionHandler)
	completion.RegisterCommandFlagHandler(deployDeleteCmd, "context", completion.FileCompletionHandler)

	return deployDeleteCmd
}
