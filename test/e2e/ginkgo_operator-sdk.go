package e2e

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/shipwright-io/build/pkg/apis"
	operator "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	v1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"

	. "github.com/onsi/ginkgo"
)

// Logf logs data
func Logf(format string, args ...interface{}) {
	currentTime := time.Now().UTC().Format(time.RFC3339)

	fmt.Fprintf(GinkgoWriter, currentTime+" "+strconv.Itoa(getGinkgoNode())+" "+format+"\n", args...)
}

// configureOperatorSDKTestFramework configures the operator-sdk test framework that expects a set of command
// line arguments, see https://github.com/operator-framework/operator-sdk/blob/2f772d1dc2340dd19bdc3ec8c2dc9f0f77cc8297/doc/test-framework/writing-e2e-tests.md#local
func configureOperatorSDKTestFramework() error {
	ginkgoNode := getGinkgoNode()

	// determine the root directory
	rootDir, err := getRootDir()
	if err != nil {
		return err
	}

	// determine the ginkgo seed
	ginkgoSeed, err := getGinkoSeed()
	if err != nil {
		return err
	}

	// specify the file name of the global resource manifest
	globalManFile := fmt.Sprintf("%s/global-manifest-%s.yaml", os.TempDir(), ginkgoSeed)
	if ginkgoNode == 1 {
		// determine the file content
		content := []byte{}
		if os.Getenv(EnvVarCreateGlobalObjects) == "true" {
			files, err := ioutil.ReadDir(rootDir + "/deploy/crds")
			if err != nil {
				Logf("Failed to read directory %s %s", rootDir+"/deploy/crds", err.Error())
				return err
			}

			for _, file := range files {
				bytes, err := ioutil.ReadFile(rootDir + "/deploy/crds/" + file.Name())
				if err != nil {
					Logf("Failed to read file %s %s", rootDir+"/deploy/crds/"+file.Name(), err.Error())
					return err
				}
				content = append(content, bytes...)
			}
		}

		// write the file
		err = ioutil.WriteFile(globalManFile, content, 0644)
		if err != nil {
			Logf("Failed to write file %s %s", globalManFile, err.Error())
			return err
		}
	}

	// specify the file name for namespace resource manifest
	namespacedManFile := fmt.Sprintf("%s/namespace-manifest-%s.yaml", os.TempDir(), ginkgoSeed)
	if ginkgoNode == 1 {
		// write the file (empty)
		err = ioutil.WriteFile(namespacedManFile, []byte{}, 0644)
		if err != nil {
			Logf("Failed to write file %s %s", namespacedManFile, err.Error())
			return err
		}
	} else {
		// ensure the file exists
		for _, err = os.Stat(namespacedManFile); err != nil && os.IsNotExist(err); {
			time.Sleep(100 * time.Millisecond)
			_, err = os.Stat(namespacedManFile)
		}
	}

	// specify the args
	os.Args = append(os.Args, "-root", rootDir, "-globalMan", globalManFile, "-namespacedMan", namespacedManFile)

	return nil
}

// cleanupOperatorSDKTestFramework deletes the temporary files created for the operator-sdk test framework
func cleanupOperatorSDKTestFramework() error {
	// determine the ginkgo seed
	ginkgoSeed, err := getGinkoSeed()
	if err != nil {
		return err
	}

	globalManFile := fmt.Sprintf("%s/global-manifest-%s.yaml", os.TempDir(), ginkgoSeed)
	err = os.Remove(globalManFile)
	if err != nil {
		return err
	}

	namespacedManFile := fmt.Sprintf("%s/namespace-manifest-%s.yaml", os.TempDir(), ginkgoSeed)
	err = os.Remove(namespacedManFile)
	if err != nil {
		return err
	}

	return nil
}

func getArg(argName string) (bool, string) {
	for i, arg := range os.Args {
		if arg == argName {
			return true, os.Args[i+1]
		} else if strings.HasPrefix(arg, argName+"=") {
			argAndValue := strings.SplitN(arg, "=", 2)
			return true, argAndValue[1]
		}
	}
	return false, ""
}

func getGinkgoNode() int {
	defined, ginkgoNodeString := getArg("--ginkgo.parallel.node")
	if !defined {
		return 1
	}
	ginkgoNode, err := strconv.Atoi(ginkgoNodeString)
	if err != nil {
		fmt.Printf("Error: %s", err.Error())
		return 0
	}
	return ginkgoNode
}

func getGinkoSeed() (string, error) {
	defined, ginkgoSeed := getArg("--ginkgo.seed")
	if !defined {
		return "", errors.New("unable to find --ginko.seed argument")
	}
	return ginkgoSeed, nil
}

// getRootDir returns the root directory of the project no matter if it is the current working directory
// or not. It will go up and search for a parent directory that contains a LICENSE file. This is to
// workaround the current working directory to be the suite's directory if the -r flag is used.
// See https://github.com/onsi/ginkgo/issues/432
func getRootDir() (string, error) {
	rootDir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for info, err := os.Stat(rootDir + "/LICENSE"); os.IsNotExist(err) || (err == nil && info.IsDir()); {
		rootDir = filepath.Dir(rootDir)
		if rootDir == "/" {
			return "", errors.New("failed to determine root directory")
		}
		info, err = os.Stat(rootDir + "/LICENSE")
	}

	return rootDir, nil
}

func populateOperatorSDKTestFrameworkScheme() error {
	err := framework.AddToFrameworkScheme(apis.AddToScheme, &operator.BuildList{})
	if err != nil {
		Logf("Failed to add BuildList to schema %s", err.Error())
		return err
	}

	if os.Getenv(EnvVarCreateGlobalObjects) == "true" {
		err = framework.AddToFrameworkScheme(apis.AddToScheme, &operator.BuildStrategyList{})
		if err != nil {
			Logf("Failed to add BuildStrategyList to schema %s", err.Error())
			return err
		}

		err = framework.AddToFrameworkScheme(apis.AddToScheme, &operator.ClusterBuildStrategyList{})
		if err != nil {
			Logf("Failed to add ClusterBuildStrategyList to schema %s", err.Error())
			return err
		}
	}

	if os.Getenv(EnvVarVerifyTektonObjects) == "true" {
		err = framework.AddToFrameworkScheme(v1beta1.AddToScheme, &v1beta1.TaskList{})
		if err != nil {
			Logf("Failed to add TaskList to schema %s", err.Error())
			return err
		}

		err = framework.AddToFrameworkScheme(v1beta1.AddToScheme, &v1beta1.TaskRunList{})
		if err != nil {
			Logf("Failed to add TaskRunList to schema %s", err.Error())
			return err
		}
	}

	return nil
}

func startLocalOperator() (*exec.Cmd, error) {
	rootDir, err := getRootDir()
	if err != nil {
		return nil, err
	}

	// define the operator command
	operatorCmd := exec.Command("operator-sdk", "run", "--local", "--verbose", "--watch-namespace", os.Getenv("TEST_WATCH_NAMESPACE"))
	operatorCmd.Dir = rootDir
	outBuf := &bytes.Buffer{}
	operatorCmd.Stdout = outBuf
	operatorCmd.Stderr = outBuf

	// start the local operator
	err = operatorCmd.Start()
	if err != nil {
		return nil, err
	}

	// make sure the Operator SDK test framework knows that the operator is local
	framework.Global.LocalOperator = true

	return operatorCmd, nil
}
