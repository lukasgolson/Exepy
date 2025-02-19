package common

import "path/filepath"

const PythonFilename = "python"
const ScriptsFilename = "scripts"
const ScriptIntegrityFilename = "scripts_integrity"
const WheelsFolderName = "wheels"
const HashmapName = "hashmap"
const CopyToRootFilename = "copy_to_root"

const pipFilename = "pip.pyz"

func GetConfigEmbedName() string {
	return "settings.json"
}

func GetPipName(extractDir string) string {
	return filepath.Join(extractDir, pipFilename)
}
