package main

import (
	"encoding/json"
	"io/ioutil"
)

const (
	pythonDownloadURL  = "https://www.python.org/ftp/python/3.11.7/python-3.11.7-embed-amd64.zip"
	pipDownloadURL     = "https://bootstrap.pypa.io/pip/get-pip.py"
	pythonDownloadFile = "python code-3.11.7-embed-amd64.zip"
	pythonExtractDir   = "python-embed"
	pthFile            = "python311._pth"
	pythonInteriorZip  = "python311.zip"
)

type pythonSetupSettings struct {
	PythonDownloadURL string `json:"pythonDownloadURL"`
	PipDownloadURL    string `json:"pipDownloadURL"`
	PythonDownloadZip string `json:"pythonDownloadFile"`
	PythonExtractDir  string `json:"pythonExtractDir"`
	PthFile           string `json:"pthFile"`
	PythonInteriorZip string `json:"pythonInteriorZip"`
	RequirementsFile  string `json:"requirementsFile"`
}

func loadSettings(filename string) (*pythonSetupSettings, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var settings pythonSetupSettings
	err = json.Unmarshal(data, &settings)
	if err != nil {
		return nil, err
	}

	return &settings, nil
}

func saveSettings(filename string, settings *pythonSetupSettings) error {
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(filename, data, 0644)
	if err != nil {
		return err
	}

	return nil
}

func loadOrSaveDefault(filename string) (*pythonSetupSettings, error) {
	settings, err := loadSettings(settingsFile)
	if err != nil {
		settings = &pythonSetupSettings{
			PythonDownloadURL: pythonDownloadURL,
			PipDownloadURL:    pipDownloadURL,
			PythonDownloadZip: pythonDownloadFile,
			PythonExtractDir:  pythonExtractDir,
			PthFile:           pthFile,
			PythonInteriorZip: pythonInteriorZip,
			RequirementsFile:  "requirements.txt",
		}
		err = saveSettings(settingsFile, settings)
		if err != nil {
			return nil, err
		}
	}

	return settings, nil
}
