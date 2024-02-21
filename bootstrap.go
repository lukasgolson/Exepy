package main

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

//go:embed payload.zip
var payloadZip []byte

//go:embed python.zip
var pythonZip []byte

func main() {

	// EXTRACT THE PIPELINE ZIP FILE

	// save the pipeline zip file to the current directory
	if _, err := os.Stat("bootstrapped"); os.IsNotExist(err) {
		if err := os.WriteFile("pipeline.zip", payloadZip, os.ModePerm); err != nil {
			fmt.Println("Error saving pipeline zip file:", err)
			return
		}

		if err := extractZip("pipeline.zip", "", 0); err != nil {
			fmt.Println("Error extracting pipeline zip file:", err)
			return
		}

		// delete the pipeline zip file
		if err := os.Remove("pipeline.zip"); err != nil {
			fmt.Println("Error deleting pipeline zip file:", err)
		}

		if err := runCommand(filepath.Join(pythonExtractDir, "python.exe"), []string{"setup.py"}); err != nil {
			fmt.Println("Error running setup.py:", err)
			return
		}

		// save a text file to the current directory to indicate that the bootstrap has been run
		if err := os.WriteFile("bootstrapped", []byte("Bootstrap has been run"), os.ModePerm); err != nil {
			fmt.Println("Error saving bootstrap text file:", err)
			return
		}
	}

	appendedArguments := append([]string{"videoToPointcloud.py"}, os.Args[1:]...)

	if err := runCommand(filepath.Join(pythonExtractDir, "python.exe"), appendedArguments); err != nil {
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
