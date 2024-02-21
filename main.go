package main

import (
	"bytes"
	_ "embed"
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
		panic(err)
	}

	pythonScriptPath := path.Join(settings.PayloadDir, settings.PayloadScript)
	requirementsPath := path.Join(settings.PayloadDir, settings.RequirementsFile)

	// check if payload directory exists
	if common.DoesPathExist(settings.PayloadDir) {
		println("Payload directory does not exist: ", settings.PayloadDir)
		return
	}

	// check if payload directory has the main file
	if common.DoesPathExist(pythonScriptPath) {
		println("Main file does not exist: ", pythonScriptPath)
		return
	}

	// if requirements file is listed, check that it exists
	if settings.RequirementsFile != "" {
		if common.DoesPathExist(requirementsPath) {
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

	println("Embedded payload")

}

func createEmbedMap(PythonRS, PayloadRS, wheelsFile, SettingsFile io.ReadSeeker) map[string]io.ReadSeeker {

	embedMap := make(map[string]io.ReadSeeker)

	embedMap[common.GetPythonEmbedName()] = PythonRS
	embedMap[common.GetPayloadEmbedName()] = PayloadRS
	embedMap[common.GetWheelsEmbedName()] = wheelsFile
	embedMap[common.GetConfigEmbedName()] = SettingsFile

	return embedMap
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
