package integration

import (
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/openshift/odo/pkg/log"
	"github.com/openshift/odo/tests/helper"
)

var _ = Describe("odo devfile create command tests", func() {
	var project string
	var context string
	var currentWorkingDirectory string

	// This is run after every Spec (It)
	var _ = BeforeEach(func() {
		SetDefaultEventuallyTimeout(10 * time.Minute)
		project = helper.CreateRandProject()
		context = helper.CreateNewDevfileContext()
		currentWorkingDirectory = helper.Getwd()
		helper.Chdir(context)
		os.Setenv("GLOBALODOCONFIG", filepath.Join(context, "config.yaml"))
	})

	// This is run after every Spec (It)
	var _ = AfterEach(func() {
		helper.DeleteProject(project)
		helper.Chdir(currentWorkingDirectory)
		//helper.DeleteDir(context)
		os.Unsetenv("GLOBALODOCONFIG")
	})

	Context("When executing odo create with devfile component type argument", func() {
		It("should successfully create the devfile component", func() {
			helper.CmdShouldPass("odo", "preference", "set", "Experimental", "true")
			helper.CmdShouldPass("odo", "create", "openLiberty")
		})
	})

	Context("When executing odo create with devfile component type and component name arguments", func() {
		It("should successfully create the devfile component", func() {
			helper.CmdShouldPass("odo", "preference", "set", "Experimental", "true")
			componentName := helper.RandString(6)
			helper.CmdShouldPass("odo", "create", "openLiberty", componentName)
		})
	})

	Context("When executing odo create with devfile component type argument and --project flag", func() {
		It("should successfully create the devfile component", func() {
			helper.CmdShouldPass("odo", "preference", "set", "Experimental", "true")
			componentNamespace := helper.RandString(6)
			helper.CmdShouldPass("odo", "create", "openLiberty", "--project", componentNamespace)
		})
	})

	Context("When executing odo create with devfile component name that contains unsupported character", func() {
		It("should failed with component name is not valid and prompt supported character", func() {
			helper.CmdShouldPass("odo", "preference", "set", "Experimental", "true")
			componentName := "BAD@123"
			output := helper.CmdShouldFail("odo", "create", "openLiberty", componentName)
			helper.MatchAllInOutput(output, []string{"Contain only lowercase alphanumeric characters or ‘-’"})
		})
	})

	Context("When executing odo create with devfile component name that contains all numeric values", func() {
		It("should failed with component name is not valid and prompt container name must not contain all numeric values", func() {
			helper.CmdShouldPass("odo", "preference", "set", "Experimental", "true")
			componentName := "123456"
			output := helper.CmdShouldFail("odo", "create", "openLiberty", componentName)
			helper.MatchAllInOutput(output, []string{"Must not contain all numeric values"})
		})
	})

	Context("When executing odo create with devfile component name that contains more than 63 characters ", func() {
		It("should failed with component name is not valid and prompt container name contains at most 63 characters", func() {
			helper.CmdShouldPass("odo", "preference", "set", "Experimental", "true")
			componentName := helper.RandString(64)
			output := helper.CmdShouldFail("odo", "create", "openLiberty", componentName)
			helper.MatchAllInOutput(output, []string{"Contain at most 63 characters"})
		})
	})

	Context("When executing odo create with devfile component and --downloadSource flag", func() {
		It("should succesfully create the compoment and download the source", func() {
			helper.CmdShouldPass("odo", "preference", "set", "Experimental", "true")
			devfile := "devfile.yaml"
			helper.CopyExampleDevFile(filepath.Join("source", "devfiles", "nodejs", devfile), filepath.Join(context, devfile))
			// TODO: Check for devfile in non-default location
			helper.CmdShouldPass("odo", "create", "nodejs", "--downloadSource", "--devfile", devfile)
			expectedFiles := []string{"package.json", "package-lock.json", "README.MD", devfile}
			Expect(helper.VerifyFilesExist(context, expectedFiles)).To(Equal(true))
		})
	})

	Context("When executing odo create with shh clone location", func() {
		It("should succesfully create the compoment and download the source", func() {
			helper.CmdShouldPass("odo", "preference", "set", "Experimental", "true")
			devfile := "devfile.yaml"
			helper.CopyExampleDevFile(filepath.Join("source", "devfiles", "nodejs", devfile), filepath.Join(context, devfile))
			err := helper.ReplaceDevfileField("devfile.yaml", "location", "\"git@github.com:che-samples/web-nodejs-sample.git\"")
			if err != nil {
				log.Info("Could not replace the entry in the devfile: " + err.Error())
			}
			helper.CmdShouldPass("odo", "create", "nodejs", "--downloadSource", "--devfile", devfile)
			expectedFiles := []string{"package.json", "package-lock.json", "README.MD", devfile}
			Expect(helper.VerifyFilesExist(context, expectedFiles)).To(Equal(true))
		})
	})

	Context("When executing odo create with http location (rather than https)", func() {
		It("shouldn't create the compoment and download the source", func() {
			helper.CmdShouldPass("odo", "preference", "set", "Experimental", "true")
			devfile := "devfile.yaml"
			helper.CopyExampleDevFile(filepath.Join("source", "devfiles", "nodejs", devfile), filepath.Join(context, devfile))
			err := helper.ReplaceDevfileField("devfile.yaml", "location", "http://github.com/che-samples/web-nodejs-sample.git")
			if err != nil {
				log.Info("Could not replace the entry in the devfile: " + err.Error())
			}
			// check return code for expected output
			helper.CmdShouldPass("odo", "create", "nodejs", "--downloadSource", "--devfile", devfile)
			expectedFiles := []string{"package.json", "package-lock.json", "README.MD", devfile}
			Expect(helper.VerifyFilesExist(context, expectedFiles)).To(Equal(false))
		})
	})

	Context("When executing odo create with missing github owner", func() {
		It("shouldn't create the compoment and download the source", func() {
			helper.CmdShouldPass("odo", "preference", "set", "Experimental", "true")
			devfile := "devfile.yaml"
			helper.CopyExampleDevFile(filepath.Join("source", "devfiles", "nodejs", devfile), filepath.Join(context, devfile))
			err := helper.ReplaceDevfileField("devfile.yaml", "location", "https://github.com//web-nodejs-sample.git")
			if err != nil {
				log.Info("Could not replace the entry in the devfile: " + err.Error())
			}
			// check return code for expected output
			helper.CmdShouldFail("odo", "create", "nodejs", "--downloadSource", "--devfile", devfile)
			expectedFiles := []string{"package.json", "package-lock.json", "README.MD", "devfile.yaml"}
			Expect(helper.VerifyFilesExist(context, expectedFiles)).To(Equal(false))
		})
	})

	Context("When executing odo create with type zip", func() {
		It("should create the compoment and download the source", func() {
			helper.CmdShouldPass("odo", "preference", "set", "Experimental", "true")
			projectFolder := filepath.Join(context, "project")
			devfile := "devfile.yaml"
			err := os.Mkdir(projectFolder, os.FileMode(644))
			if err != nil {
				log.Info("Could not replace the entry in the devfile: " + err.Error())
			}
			helper.CopyExampleDevFile(filepath.Join("source", "devfiles", "nodejs", devfile), filepath.Join(projectFolder, devfile))
			err = helper.ReplaceDevfileField("devfile.yaml", "location", "https://github.com/che-samples/web-nodejs-sample/archive/master.zip")
			if err != nil {
				log.Info("Could not replace the entry in the devfile: " + err.Error())
			}
			err = helper.ReplaceDevfileField("devfile.yaml", "type", "zip")
			if err != nil {
				log.Info("Could not replace the entry in the devfile: " + err.Error())
			}
			helper.CmdShouldPass("odo", "create", "nodejs", "--downloadSource", "--devfile", filepath.Join(projectFolder, devfile), "--context", context)
			expectedFiles := []string{"package.json", "package-lock.json", "README.MD", devfile}
			Expect(helper.VerifyFilesExist(projectFolder, expectedFiles)).To(Equal(true))
		})
	})

	//file:// tests
	//git type --> github urlValue
	//git type --> not github url value
	//folder not empty
	//
})
