package main

import (
	"fmt"
	"io"
	"lukasolson.net/common"
	"os"
	"path/filepath"
)

func PreparePython(settings common.PythonSetupSettings) (io.ReadSeeker, io.ReadSeeker, error) {

	cleanDirectory(&settings)

	defer cleanDirectory(&settings)

	// CREATE THE EXTRACTION DIRECTORY
	common.RemoveIfExists(*settings.PythonExtractDir)
	if err := os.Mkdir(*settings.PythonExtractDir, os.ModePerm); err != nil {
		fmt.Println("Error creating extraction directory:", err)
		return nil, nil, err
	}

	// DOWNLOAD PYTHON ZIP FILE
	if err := common.DownloadFile(*settings.PythonDownloadURL, *settings.PythonDownloadZip); err != nil {
		fmt.Println("Error downloading Python zip file:", err)
		return nil, nil, err
	}

	// DOWNLOAD PIP FILE
	if err := common.DownloadFile(*settings.PipDownloadURL, common.GetPipName(*settings.PythonExtractDir)); err != nil {
		fmt.Println("Error downloading pip module:", err)
		return nil, nil, err
	}

	if err := createBasePythonInstallation(&settings, *settings.PythonDownloadZip); err != nil {
		fmt.Println("Error creating base Python installation:", err)
		return nil, nil, err
	}

	common.RemoveIfExists(*settings.PythonDownloadZip)

	pythonStream, err := common.DirToStream(*settings.PythonExtractDir, []string{})

	if err != nil {
		fmt.Println("Error zipping Python directory:", err)
		return nil, nil, err
	}

	wheelsPath := filepath.Join(*settings.PythonExtractDir, common.WheelsFolderName)

	os.Mkdir(wheelsPath, os.ModePerm)

	if *settings.InstallerRequirements != "" {
		if common.DoesPathExist(*settings.InstallerRequirements) {
			fmt.Println("Installer requirements file found:", *settings.InstallerRequirements)
			if err := buildRequirementWheels(*settings.PythonExtractDir, *settings.InstallerRequirements, wheelsPath); err != nil {
				return nil, nil, err
			}

		} else {
			fmt.Println("Installer requirements file not found but is specified in configuration:", settings.InstallerRequirements)
		}
	}

	wheelsStream, _ := common.DirToStream(wheelsPath, []string{})

	return pythonStream, wheelsStream, nil
}

func createBasePythonInstallation(settings *common.PythonSetupSettings, pythonZip string) error {
	// EXTRACT THE Python ZIP FILE
	if err := common.ExtractZip(pythonZip, *settings.PythonExtractDir, 0); err != nil {
		fmt.Println("Error extracting Python zip file:", err)
		return err
	}

	if err := extractInteriorPythonArchive(settings); err != nil {
		return err
	}

	if err := updatePTHFile(settings); err != nil {
		return err
	}

	// write to sitecustomize.py file
	if err := createSiteCustomFile(settings); err != nil {
		return err
	}

	// make empty DLLs folder
	if err := os.Mkdir(filepath.Join(*settings.PythonExtractDir, "DLLs"), os.ModePerm); err != nil {
		fmt.Println("Error creating DLLs folder:", err)
		return err
	}

	return nil
}

func extractInteriorPythonArchive(settings *common.PythonSetupSettings) error {
	// EXTRACT THE EMBEDDED PYTHON INTERIOR ZIP FILE

	if err := common.ExtractZip(filepath.Join(*settings.PythonExtractDir, *settings.PythonInteriorZip), *settings.PythonExtractDir, 0); err != nil {
		fmt.Println("Error extracting the interiorPython zip file:", err)
		return err
	}

	common.RemoveIfExists(filepath.Join(*settings.PythonExtractDir, *settings.PythonInteriorZip))
	return nil
}

func createSiteCustomFile(settings *common.PythonSetupSettings) error {
	sitecustomizeFile, err := os.Create(filepath.Join(*settings.PythonExtractDir, "sitecustomize.py"))

	if err != nil {
		return err
	}

	_, err = sitecustomizeFile.WriteString("import sys\nsys.path.append('.')")

	if err != nil {
		return err
	}

	return nil
}

func updatePTHFile(settings *common.PythonSetupSettings) error {
	common.RemoveIfExists(filepath.Join(*settings.PythonExtractDir, *settings.PthFile))

	// write to ._pth file
	pthFile, err := os.Create(filepath.Join(*settings.PythonExtractDir, *settings.PthFile))
	if err != nil {
		fmt.Println("Error creating ._pth file:", err)
		return nil
	}

	// change python311 to pythonExtractDir
	_, err = pthFile.WriteString(".\\" + *settings.PythonExtractDir + "\n.\\Scripts\n.\n.\\Lib\\site-packages\nimport site")
	if err != nil {
		fmt.Println("Error writing to ._pth file:", err)
		return nil
	}
	return err
}

func buildRequirementWheels(extractDir, requirementsFile, wheelsPath string) error {

	pythonPath := filepath.Join(extractDir, "python.exe")
	pipPath := common.GetPipName(extractDir) // Get pip path once

	// Install pip, setuptools, and wheel (if not already installed)
	if err := common.RunCommand(pythonPath, []string{pipPath, "install", "--upgrade", "pip", "setuptools", "wheel"}); err != nil {
		fmt.Println("Error installing/upgrading pip, setuptools, wheel:", err)
		return err
	}

	requiredWheelsDir := filepath.Join(wheelsPath, "required")
	setupWheelsDir := filepath.Join(wheelsPath, "setup")

	// Create wheel directory if it doesn't exist.  Important to avoid errors later.
	if err := os.MkdirAll(requiredWheelsDir, os.ModePerm); err != nil {
		fmt.Println("Error creating required wheel directory:", err)
		return err
	}

	if err := os.MkdirAll(setupWheelsDir, os.ModePerm); err != nil {
		fmt.Println("Error creating setup wheel directory:", err)
		return err
	}

	// Build wheels for requirements
	if err := common.RunCommand(pythonPath, []string{pipPath, "wheel", "-w", requiredWheelsDir, "-r", requirementsFile}); err != nil {
		fmt.Println("Error building requirement wheels:", err)
		return err
	}

	// Build wheel for setuptools.
	if err := common.RunCommand(pythonPath, []string{pipPath, "wheel", "-w", setupWheelsDir, "setuptools", "pip"}); err != nil {
		fmt.Println("Error building setuptools/wheel wheels:", err)
		return err
	}

	return nil
}

func cleanDirectory(settings *common.PythonSetupSettings) {
	common.RemoveIfExists(*settings.PythonExtractDir)
	common.RemoveIfExists(*settings.PythonDownloadZip)

	println("Directory cleaned")
}
