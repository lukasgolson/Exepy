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

func main() {

	pythonPreparer.PreparePython()

	file, err := os.Create("launch.exe")
	if err != nil {
		panic(err)
	}

	defer file.Close()

	pythonFile, err := os.Open("python.tar.GZ")
	if err != nil {
		panic(err)
	}

	defer pythonFile.Close()

	PayloadFile, err := os.Open("python.tar.GZ")
	if err != nil {
		panic(err)
	}

	defer pythonFile.Close()

	embedMap := createEmbedMap(pythonFile, PayloadFile)

	embedPayload(file, embedMap)

	println("Embedded payload")

}

func createEmbedMap(PythonRS, PayloadRS io.ReadSeeker) map[string]io.ReadSeeker {

	embedMap := make(map[string]io.ReadSeeker)

	embedMap[common.GetPythonEmbedName()] = PythonRS

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
