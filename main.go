package main

import (
	"bytes"
	_ "embed"
	"github.com/maja42/ember/embedding"
	"io"
	"lukasolson.net/common"
	"lukasolson.net/pythonPreparer"
	"os"
)

const settingsFileName = "settings.json"

func main() {

	settings, err := common.LoadOrSaveDefault(settingsFileName)
	if err != nil {
		panic(err)
	}

	// check if payload directory exists
	if _, err := os.Stat(settings.PayloadDir); os.IsNotExist(err) {
		println("Payload directory does not exist: ", settings.PayloadDir)
		return
	}

	file, err := os.Create("launch.exe")
	if err != nil {
		panic(err)
	}

	defer file.Close()

	pythonFile, _, err := pythonPreparer.PreparePython(*settings)
	if err != nil {
		panic(err)
	}

	PayloadFile, err := common.CompressDirToStream(settings.PayloadDir)
	if err != nil {
		panic(err)
	}

	SettingsFile, err := os.Open(settingsFileName)
	defer SettingsFile.Close()

	embedMap := createEmbedMap(pythonFile, PayloadFile, SettingsFile)

	embedPayload(file, embedMap)

	println("Embedded payload")

}

func createEmbedMap(PythonRS, PayloadRS io.ReadSeeker, SettingsFile io.ReadSeeker) map[string]io.ReadSeeker {

	embedMap := make(map[string]io.ReadSeeker)

	embedMap[common.GetPythonEmbedName()] = PythonRS
	embedMap[common.GetPayloadEmbedName()] = PayloadRS
	embedMap[common.GetConfigEmbedName()] = SettingsFile

	return embedMap
}

//go:embed bootstrap.exe
var bootstrapExe []byte

func embedPayload(writer io.Writer, attachments map[string]io.ReadSeeker) {

	copiedByteArray := make([]byte, len(bootstrapExe))
	copy(copiedByteArray, bootstrapExe)

	reader := bytes.NewReader(copiedByteArray)

	embedding.Embed(writer, reader, attachments, nil)
}
