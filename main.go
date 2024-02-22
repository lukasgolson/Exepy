package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"github.com/maja42/ember/embedding"
	"io"
	"lukasolson.net/common"
	"lukasolson.net/pythonPreparer"
	"os"
	"path"
)

const settingsFileName = "settings.json"

func main() {

	settings, err := common.LoadOrSaveDefault(settingsFileName)
	if err != nil {
		return
	}

	pythonScriptPath := path.Join(settings.PayloadDir, settings.PayloadScript)
	requirementsPath := path.Join(settings.PayloadDir, settings.RequirementsFile)

	// check if payload directory exists
	if !common.DoesPathExist(settings.PayloadDir) {
		println("Payload directory does not exist: ", settings.PayloadDir)
		return
	}

	// check if payload directory has the main file
	if !common.DoesPathExist(pythonScriptPath) {
		println("Main file does not exist: ", pythonScriptPath)
		return
	}

	// if requirements file is listed, check that it exists
	if settings.RequirementsFile != "" {
		if !common.DoesPathExist(requirementsPath) {
			println("Requirements file is listed in config but does not exist: ", requirementsPath)
			return
		}
	}

	file, err := os.Create("launch.exe")
	if err != nil {
		panic(err)
	}

	defer file.Close()

	pythonFile, wheelsFile, err := pythonPreparer.PreparePython(*settings)
	if err != nil {
		panic(err)
	}

	PayloadFile, err := common.CompressDirToStream(settings.PayloadDir)
	if err != nil {
		panic(err)
	}

	SettingsFile, err := os.Open(settingsFileName)
	defer SettingsFile.Close()

	embedMap := createEmbedMap(pythonFile, PayloadFile, wheelsFile, SettingsFile)

	embedPayload(file, embedMap)

	file.Close()

	outputExeHash, err := common.Md5SumFile(file.Name())

	if err != nil {
		panic(err)
	}

	println("Output executable hash: ", outputExeHash, " saved to hash.txt")

	// save the hash to a file

	if err := SaveContentsToFile("hash.txt", outputExeHash); err != nil {
		println("Error saving hash to file")
	}

	println("Embedded payload")

}

func SaveContentsToFile(filename, contents string) error {
	hashFile, err := os.Create(filename)
	if err != nil {
		return err
	}

	defer hashFile.Close()

	_, err = hashFile.WriteString(contents)
	return err
}

func createEmbedMap(PythonRS, PayloadRS, wheelsFile, SettingsFile io.ReadSeeker) map[string]io.ReadSeeker {

	hashMap, hashBytes := HashFiles(PythonRS, PayloadRS, wheelsFile, SettingsFile)

	json.NewEncoder(hashBytes).Encode(hashMap)

	embedMap := make(map[string]io.ReadSeeker)

	embedMap[common.GetHashEmbedName()] = bytes.NewReader(hashBytes.Bytes())
	embedMap[common.GetPythonEmbedName()] = PythonRS
	embedMap[common.GetPayloadEmbedName()] = PayloadRS
	embedMap[common.GetWheelsEmbedName()] = wheelsFile
	embedMap[common.GetConfigEmbedName()] = SettingsFile

	return embedMap
}

func HashFiles(PythonRS io.ReadSeeker, PayloadRS io.ReadSeeker, wheelsFile io.ReadSeeker, SettingsFile io.ReadSeeker) (map[string]string, *bytes.Buffer) {
	PythonHash, err := common.HashReadSeeker(PythonRS)
	if err != nil {
		panic(err)
	}

	PayloadHash, err := common.HashReadSeeker(PayloadRS)
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

	hashMap, hashBytes := make(map[string]string), new(bytes.Buffer)
	hashMap[common.GetPythonEmbedName()] = PythonHash
	hashMap[common.GetPayloadEmbedName()] = PayloadHash
	hashMap[common.GetWheelsEmbedName()] = wheelsFileHash
	hashMap[common.GetConfigEmbedName()] = SettingsFileHash

	// print the hashes
	for k, v := range hashMap {
		fmt.Println("Hash for", k, ":", v)
	}

	return hashMap, hashBytes
}

//go:embed bootstrap.exe
var bootstrapExe []byte

func embedPayload(writer io.Writer, attachments map[string]io.ReadSeeker) {

	copiedByteArray := make([]byte, len(bootstrapExe))
	copy(copiedByteArray, bootstrapExe)

	reader := bytes.NewReader(copiedByteArray)

	err := embedding.Embed(writer, reader, attachments, nil)
	if err != nil {
		return
	}
}
