package common

const PythonFileName = "python.tar.GZ"
const PayloadFileName = "payload.tar.GZ"
const getPipScriptName = "get-pip.py"

func GetPythonEmbedName() string {
	return PythonFileName
}

func GetPayloadEmbedName() string {
	return PayloadFileName
}

func GetConfigEmbedName() string {
	return "settings.json"
}

func GetPipScriptName() string {
	return getPipScriptName
}
