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

	// current working directory

	currentWorkingDir, err := os.Getwd()
	if err != nil {
		return err
	}

	CopyToRoot, err := common.FilesToStream(currentWorkingDir, settings.FilesToCopyToRoot, true)

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

	PayloadHashesReader := bytes.NewReader(PayloadHashesJson)

	var SettingsFile2 io.ReadSeeker = SettingsFile
	var PayloadIntegrity io.ReadSeeker = PayloadHashesReader

	embedMap := make(map[string]io.ReadSeeker)
	embedMap[common.PythonFilename] = pythonFile
	embedMap[common.ScriptsFilename] = PayloadFile
	embedMap[common.ScriptIntegrityFilename] = PayloadIntegrity
	embedMap[common.WheelsFolderName] = wheelsFile
	embedMap[common.CopyToRootFilename] = CopyToRoot
	embedMap[common.GetConfigEmbedName()] = SettingsFile2

	if common.ThemeMusicSupport {
		themeWavPath := path.Join(currentWorkingDir, "theme.wav")
		if common.DoesPathExist(themeWavPath) {
			println("Theme.wav found in working directory. Embedding as background music.")
			themeWavFile, err := os.Open(themeWavPath)
			if err != nil {
				return err
			}
			defer themeWavFile.Close()

			buffer := new(bytes.Buffer)
			if _, err := io.Copy(buffer, themeWavFile); err != nil {
				return err
			}
			embedMap["theme.wav"] = bytes.NewReader(buffer.Bytes())
		}
	}

	hashMap := calculateHashesFromMap(embedMap)
	var hashBytes = new(bytes.Buffer)
	json.NewEncoder(hashBytes).Encode(hashMap)

	embedMap[common.HashmapName] = bytes.NewReader(hashBytes.Bytes())

	if err := writeExecutable(file, embedMap); err != nil {
		return err
	}

	outputExeHash, err := common.Md5SumFile(file.Name())

	if err != nil {
		return err
	}

	println("Output executable hash: ", outputExeHash, " saved to hash.txt")

	// save the hash to a file

	err = file.Close()
	if err != nil {
		panic(err)
	}

	// move the file to bootstrap.exe
	err = os.Rename(file.Name(), "installer.exe")

	if err := common.SaveContentsToFile("hash.txt", outputExeHash); err != nil {
		println("Error saving hash to file")
	}

	println("Embedded payload")

	return nil

}

func calculateHashesFromMap(embedMap map[string]io.ReadSeeker) map[string]string {

	hashMap := make(map[string]string)

	for k, v := range embedMap {
		hash, err := common.HashReadSeeker(v)
		if err != nil {
			panic(err)
		}
		hashMap[k] = hash
	}

	for k, v := range hashMap {
		fmt.Println("Hash for", k, ":", v)
	}

	return hashMap
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
