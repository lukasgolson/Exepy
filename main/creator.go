package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/maja42/ember/embedding"
	"io"
	"lukasolson.net/common"
	"os"
	"path"
	"windowsPE"
)

const settingsFileName = "exepy.json"

func createInstaller() error {

	settings, err := common.LoadOrSaveDefault(settingsFileName)
	if err != nil {
		fmt.Println("Error loading or saving settings file:", err.Error())
		return err
	}

	pythonScriptPath := path.Join(*settings.ScriptDir, *settings.MainScript)
	requirementsPath := path.Join(*settings.ScriptDir, *settings.RequirementsFile)

	// check if payload directory exists
	if !common.DoesPathExist(*settings.ScriptDir) {
		println("Scripts directory does not exist: ", settings.ScriptDir)
		return errors.New("scripts directory does not exist")
	}

	// check if payload directory has the main file
	if !common.DoesPathExist(pythonScriptPath) {
		println("Main file does not exist: ", pythonScriptPath)
		return errors.New("main file does not exist")
	}

	// if requirements file is listed, check that it exists
	if *settings.RequirementsFile != "" {
		if !common.DoesPathExist(requirementsPath) {
			println("Requirements file is listed in config but does not exist: ", requirementsPath)
			return errors.New("requirements file does not exist")
		}
	}

	file, err := os.CreateTemp("", "installer*.exe")
	if err != nil {
		return err
	}

	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			println("Error closing file")
		}

		// move the file to bootstrap.exe
		err = os.Rename(file.Name(), "installer.exe")
	}(file)

	pythonFile, wheelsFile, err := PreparePython(*settings)
	if err != nil {
		return err
	}

	// check to ensure each copy to root file exists
	for _, toCopy := range settings.FilesToCopyToRoot {
		if !common.DoesPathExist(toCopy) {
			println("File to copy to root does not exist: ", toCopy)
			return errors.New("file to copy to root does not exist")
		}
	}

	CopyToRoot, err := common.FileMapToStream(settings.FilesToCopyToRoot)

	if err != nil {
		return err
	}

	ignoredDirs := settings.IgnoredPathParts

	PayloadHashes, err := common.ComputeDirectoryHashes(*settings.ScriptDir, ignoredDirs)
	if err != nil {
		return err
	}

	// convert the hashes to a json string
	PayloadHashesJson, err := json.Marshal(PayloadHashes)
	if err != nil {
		return err
	}

	PayloadFile, err := common.DirToStream(*settings.ScriptDir, ignoredDirs)
	if err != nil {
		return err
	}

	SettingsFile, err := os.Open(settingsFileName)
	defer SettingsFile.Close()

	embedMap := createEmbedMap(pythonFile, PayloadFile, wheelsFile, SettingsFile, CopyToRoot, bytes.NewReader(PayloadHashesJson))

	if err := writeExecutable(file, embedMap); err != nil {
		return err
	}

	file.Close()

	outputExeHash, err := common.Md5SumFile(file.Name())

	if err != nil {
		return err
	}

	println("Output executable hash: ", outputExeHash, " saved to hash.txt")

	// save the hash to a file

	if err := common.SaveContentsToFile("hash.txt", outputExeHash); err != nil {
		println("Error saving hash to file")
	}

	println("Embedded payload")

	return nil

}

func createEmbedMap(PythonRS, PayloadRS, wheelsFile, SettingsFile, PayloadIntegrity, CopyToRootFile io.ReadSeeker) map[string]io.ReadSeeker {

	hashMap, hashBytes := CalculateHashMap(PythonRS, PayloadRS, wheelsFile, SettingsFile, PayloadIntegrity, CopyToRootFile)

	json.NewEncoder(hashBytes).Encode(hashMap)

	embedMap := make(map[string]io.ReadSeeker)

	embedMap[common.HashmapName] = bytes.NewReader(hashBytes.Bytes())
	embedMap[common.PythonFilename] = PythonRS
	embedMap[common.ScriptsFilename] = PayloadRS
	embedMap[common.ScriptIntegrityFilename] = PayloadIntegrity
	embedMap[common.WheelsFolderName] = wheelsFile
	embedMap[common.CopyToRootFilename] = CopyToRootFile
	embedMap[common.GetConfigEmbedName()] = SettingsFile

	return embedMap
}

func CalculateHashMap(PythonRS, PayloadRS, wheelsFile, SettingsFile, PayloadHashes, CopyToRootFile io.ReadSeeker) (map[string]string, *bytes.Buffer) {
	PythonHash, err := common.HashReadSeeker(PythonRS)
	if err != nil {
		panic(err)
	}

	PayloadHash, err := common.HashReadSeeker(PayloadRS)
	if err != nil {
		panic(err)
	}

	PayloadIntegrityHash, err := common.HashReadSeeker(PayloadHashes)
	if err != nil {
		panic(err)
	}

	wheelsFileHash, err := common.HashReadSeeker(wheelsFile)
	if err != nil {
		panic(err)
	}

	SettingsFileHash, err := common.HashReadSeeker(SettingsFile)
	if err != nil {
		panic(err)
	}

	CopyToRootFileHash, err := common.HashReadSeeker(CopyToRootFile)
	if err != nil {
		panic(err)
	}

	hashMap, hashBytes := make(map[string]string), new(bytes.Buffer)
	hashMap[common.PythonFilename] = PythonHash
	hashMap[common.ScriptsFilename] = PayloadHash
	hashMap[common.ScriptIntegrityFilename] = PayloadIntegrityHash
	hashMap[common.WheelsFolderName] = wheelsFileHash
	hashMap[common.CopyToRootFilename] = CopyToRootFileHash
	hashMap[common.GetConfigEmbedName()] = SettingsFileHash

	// print the hashes
	for k, v := range hashMap {
		fmt.Println("Hash for", k, ":", v)
	}

	return hashMap, hashBytes
}

// writeExecutable is a function that embeds attachments into a Python executable.
// It takes two parameters:
// - writer: an io.Writer where the resulting executable will be written.
// - attachments: a map where the key is the name of the attachment and the value is an io.ReadSeeker that reads the attachment's content.
func writeExecutable(writer io.Writer, attachments map[string]io.ReadSeeker) error {
	// Load the executable file of the current running program
	executableBytes, err := loadSelf()
	// If an error occurred while loading the executable, return
	if err != nil {
		return err
	}

	// Clean the executable file from any previous signature or attachments
	exeWithoutSignature, err := windowsPE.RemoveSignature(executableBytes)

	if err != nil {
		return err
	}

	exeWithoutEmbeddings, err := removeEmbedding(exeWithoutSignature)

	if err != nil {
		return err
	}

	// Create a new reader for the executable bytes
	reader := bytes.NewReader(exeWithoutEmbeddings)

	// Embed the attachments into the executable
	err = embedding.Embed(writer, reader, attachments, nil)
	// If an error occurred while embedding the attachments, return
	if err != nil {
		return err
	}

	return nil
}

// loadSelf is a function that retrieves the executable file of the current running program.
// It returns the file content as a byte slice and an error if any occurred during the process.
func loadSelf() ([]byte, error) {
	// Get the path of the executable file
	selfPath, err := os.Executable()
	// If an error occurred while getting the path, return the error
	if err != nil {
		return nil, err
	}

	// Open the executable file
	file, err := os.Open(selfPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Create a new buffer to hold the file content
	memSlice := new(bytes.Buffer)

	// Copy the file content into the buffer
	_, err = io.Copy(memSlice, file)
	if err != nil {
		return nil, err
	}

	// Return the file content as a byte slice and any error that might have occurred
	return memSlice.Bytes(), err
}

func removeEmbedding(file []byte) ([]byte, error) {

	out := new(bytes.Buffer)

	reader := bytes.NewReader(file)

	err := embedding.RemoveEmbedding(out, reader, nil)

	if errors.Is(err, embedding.ErrNothingEmbedded) {
		return file, nil
	}

	return out.Bytes(), err
}
