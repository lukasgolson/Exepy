package main

import (
	"encoding/json"
	"fmt"
	"github.com/maja42/ember"
	"io"
	"lukasolson.net/common"
	"os"
	"path"
	"path/filepath"
)

func main() {

	attachments, err := ember.Open()
	if err != nil {
		fmt.Println("Error opening attachments:", err)
		return
	}
	defer attachments.Close()

	PythonReader := attachments.Reader(common.GetPythonEmbedName())

	if PythonReader == nil {
		fmt.Println("Error reading Python. Ensure it is embedded in the binary.")
		return
	}

	PayloadReader := attachments.Reader(common.GetPayloadEmbedName())

	if PayloadReader == nil {
		fmt.Println("Error reading payload. Ensure it is embedded in the binary.")
		return
	}

	// EXTRACT THE WHEELS ZIP FILE
	wheelsReader := attachments.Reader(common.GetWheelsEmbedName())
	if wheelsReader == nil {
		fmt.Println("Error reading wheels. Ensure it is embedded in the binary.")
		return
	}

	ConfigReader := attachments.Reader(common.GetConfigEmbedName())

	if ConfigReader == nil {
		fmt.Println("Error reading config. Ensure it is embedded in the binary.")
		return
	}
	config, err := io.ReadAll(ConfigReader)
	var settings common.PythonSetupSettings
	err = json.Unmarshal(config, &settings)

	// check if the bootstrap has already been run
	if _, err := os.Stat("bootstrapped"); os.IsNotExist(err) {
		// if the bootstrap has not been run, extract the Python and program files

		fmt.Println("Performing first time setup...")

		// EXTRACT THE PYTHON ZIP FILE
		err = common.DecompressIOStream(PythonReader, settings.PythonExtractDir)
		if err != nil {
			fmt.Println("Error extracting Python zip file:", err)
			return
		}

		// EXTRACT THE PIPELINE ZIP FILE
		err = common.DecompressIOStream(PayloadReader, "")
		if err != nil {
			fmt.Println("Error extracting payload zip file:", err)
			return
		}

		wheelsDir := path.Join(settings.PythonExtractDir, common.GetWheelsEmbedName())

		// EXTRACT THE WHEELS ZIP FILE
		err = common.DecompressIOStream(wheelsReader, wheelsDir)
		if err != nil {
			fmt.Println("Error extracting wheels zip file:", err)
			return
		}

		pythonPath := filepath.Join(settings.PythonExtractDir, "python.exe")

		if err := common.RunCommand(pythonPath, []string{common.GetPipName(settings.PythonExtractDir), "install", "pip", "setuptools", "wheel"}); err != nil {
			fmt.Println("Error building wheels:", err)
			return
		}

		// if requirements.txt exists, install the requirements
		if _, err := os.Stat(settings.RequirementsFile); err == nil {
			if err := common.RunCommand(pythonPath, []string{common.GetPipName(settings.PythonExtractDir), "install", "--find-links=" + "/" + path.Join(wheelsDir) + "/", "--no-index", "-r", settings.RequirementsFile}); err != nil {
				fmt.Println("Error installing requirements:", err)
				return
			}
		}

		// run the setup.py file if configured

		if settings.SetupScript != "" {
			if err := common.RunCommand(pythonPath, []string{settings.SetupScript}); err != nil {
				fmt.Println("Error running "+settings.SetupScript+":", err)
				return
			}
		}

		// save a text file to the current directory to indicate that the bootstrap has been run
		if err := os.WriteFile("bootstrapped", []byte("Bootstrap has been run"), os.ModePerm); err != nil {
			fmt.Println("Error saving bootstrap text file:", err)
			return
		}
	}

	// run the payload script

	fmt.Println("Running script...")

	appendedArguments := append([]string{settings.PayloadScript}, os.Args[1:]...)

	if err := common.RunCommand(filepath.Join(settings.PythonExtractDir, "python.exe"), appendedArguments); err != nil {
		fmt.Println("Error running Python script:", err)
		return
	}

}
