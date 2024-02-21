package main

import (
	"fmt"
	"github.com/maja42/ember"
	"lukasolson.net/common"
	"os"
	"os/exec"
	"path/filepath"
)

const pythonExtractDir = "python"

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

	// EXTRACT THE PYTHON ZIP FILE
	err = common.DecompressDir(PythonReader, "python")
	if err != nil {
		fmt.Println("Error extracting Python zip file:", err)
		return
	}

	// EXTRACT THE PIPELINE ZIP FILE
	err = common.DecompressDir(PayloadReader, "")
	if err != nil {
		fmt.Println("Error extracting payload zip file:", err)
		return
	}

	pythonPath := filepath.Join(pythonExtractDir, "python.exe")
	if err := runCommand(pythonPath, []string{"setup.py"}); err != nil {
		fmt.Println("Error running setup.py:", err)
		return
	}

	// save a text file to the current directory to indicate that the bootstrap has been run
	if err := os.WriteFile("bootstrapped", []byte("Bootstrap has been run"), os.ModePerm); err != nil {
		fmt.Println("Error saving bootstrap text file:", err)
		return
	}

	appendedArguments := append([]string{"videoToPointcloud.py"}, os.Args[1:]...)

	if err := runCommand(filepath.Join(pythonExtractDir, "python code.exe"), appendedArguments); err != nil {
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
