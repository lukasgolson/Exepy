package common

import "path/filepath"

const pythonFilename = "python"
const payloadFilename = "payload"
const wheelsFilename = "wheels"

const pipFilename = "pip.pyz"

func GetPythonEmbedName() string {
	return pythonFilename
}

func GetPayloadEmbedName() string {
	return payloadFilename
}

func GetConfigEmbedName() string {
	return "settings.json"
}

func GetPipName(extractDir string) string {
	return filepath.Join(extractDir, pipFilename)
}

func GetWheelsEmbedName() string {
	return wheelsFilename
}

func GetHashEmbedName() string {
	return "hashes"
}
