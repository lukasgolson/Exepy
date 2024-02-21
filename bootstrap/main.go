package main

import (
	"encoding/json"
	"fmt"
	"github.com/maja42/ember"
	"io"
	"lukasolson.net/common"
	"os"
	"os/exec"
	"path/filepath"
)

func main() {

	attachments, err := ember.Open()
	if err != nil {
		fmt.Println("Error opening attachments:", err)
		return
	}
	defer attachments.Close()

	contents := attachments.List()

	for _, file := range contents {
		fmt.Println(file)

	}

	PythonReader := attachments.Reader(common.GetPythonEmbedName())

	if PythonReader == nil {
		fmt.Println("Error reading python. Ensure it is embedded in the binary.")
		return
	}

	PayloadReader := attachments.Reader(common.GetPayloadEmbedName())

	if PayloadReader == nil {
		fmt.Println("Error reading payload. Ensure it is embedded in the binary.")
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

		fmt.Println("Extracting Python and program files...")

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

		pythonPath := filepath.Join(settings.PythonExtractDir, "python.exe")

		// install the requirements
		if err := runCommand(pythonPath, []string{"-m", "pip", "install", "--find-links=" + settings.PythonExtractDir + "/wheels/", "-r", "requirements.txt"}); err != nil {
			fmt.Println("Error installing requirements:", err)
			return
		}

		// run the setup.py file if configured

		if settings.SetupScript != "" {
			if err := runCommand(pythonPath, []string{settings.SetupScript}); err != nil {
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

	appendedArguments := append([]string{settings.PayloadScript}, os.Args[1:]...)

	if err := runCommand(filepath.Join(settings.PythonExtractDir, "python.exe"), appendedArguments); err != nil {
		fmt.Println("Error running Python script:", err)
		return
	}

}

func runCommand(command string, args []string) error {
	cmd := exec.Command(command, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
