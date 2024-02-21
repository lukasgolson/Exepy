package pythonPreparer

import (
	"fmt"
	"io"
	common "lukasolson.net/common"
	"net/http"
	"os"
	"path/filepath"
)

const (
	outputZip    = "python.tar.GZ"
	settingsFile = "settings.json"
)

func PreparePython() {

	settings, err := common.LoadOrSaveDefault(settingsFile)
	if err != nil {
		return
	}

	cleanDirectory(settings)
	removeIfExists(outputZip)

	// CREATE THE EXTRACTION DIRECTORY
	if _, err := os.Stat(settings.PythonExtractDir); os.IsNotExist(err) {
		if err := os.Mkdir(settings.PythonExtractDir, os.ModePerm); err != nil {
			fmt.Println("Error creating extraction directory:", err)
			return
		}
	}

	// DOWNLOAD PYTHON ZIP FILE
	if err := downloadFile(settings.PythonDownloadURL, settings.PythonDownloadZip); err != nil {
		fmt.Println("Error downloading Python zip file:", err)
		return
	}

	if err := createBasePythonInstallation(settings, settings.PythonDownloadZip); err != nil {
		fmt.Println("Error creating base Python installation:", err)
		return
	}

	removeIfExists(settings.PythonDownloadZip)

	if settings.RequirementsFile != "" {

		if _, err := os.Stat(settings.RequirementsFile); !os.IsNotExist(err) {

			if err := setupRequirements(settings.PythonExtractDir, settings.RequirementsFile); err != nil {
				return
			}

		} else {
			fmt.Println("Requirements file not found but is specified in configuration:", settings.RequirementsFile)
		}
	}

	if err := common.CompressDir(settings.PythonExtractDir, outputZip); err != nil {
		fmt.Println("Error zipping Python directory:", err)
		return
	}

}

func createBasePythonInstallation(settings *common.PythonSetupSettings, pythonZip string) error {
	// EXTRACT THE Python ZIP FILE
	if err := common.ExtractZip(pythonZip, settings.PythonExtractDir, 0); err != nil {
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
	if err := os.Mkdir(filepath.Join(settings.PythonExtractDir, "DLLs"), os.ModePerm); err != nil {
		fmt.Println("Error creating DLLs folder:", err)
		return err
	}

	// DOWNLOAD PIP FILE
	if err := downloadFile(settings.PipDownloadURL, settings.PythonExtractDir+"/get-pip.py"); err != nil {
		fmt.Println("Error downloading pip module:", err)
		return err
	}

	return nil
}

func extractInteriorPythonArchive(settings *common.PythonSetupSettings) error {
	// EXTRACT THE EMBEDDED PYTHON INTERIOR ZIP FILE

	if err := common.ExtractZip(filepath.Join(settings.PythonExtractDir, settings.PythonInteriorZip), settings.PythonExtractDir, 0); err != nil {
		fmt.Println("Error extracting the interiorPython zip file:", err)
		return err
	}

	removeIfExists(filepath.Join(settings.PythonExtractDir, settings.PythonInteriorZip))
	return nil
}

func createSiteCustomFile(settings *common.PythonSetupSettings) error {
	sitecustomizeFile, err := os.Create(filepath.Join(settings.PythonExtractDir, "sitecustomize.py"))

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
	removeIfExists(filepath.Join(settings.PythonExtractDir, settings.PthFile))

	// write to ._pth file
	pthFile, err := os.Create(filepath.Join(settings.PythonExtractDir, settings.PthFile))
	if err != nil {
		fmt.Println("Error creating ._pth file:", err)
		return nil
	}

	// change python311 to pythonExtractDir
	_, err = pthFile.WriteString(".\\" + settings.PythonExtractDir + "\n.\\Scripts\n.\n# importing site will run sitecustomize.py\nimport site")
	if err != nil {
		fmt.Println("Error writing to ._pth file:", err)
		return nil
	}
	return err
}

func setupRequirements(extractDir, requirementsFile string) error {

	pythonPath := filepath.Join(extractDir, "python.exe")

	if err := runCommand(pythonPath, []string{filepath.Join(extractDir, "get-pip.py")}); err != nil {
		fmt.Println("Error running get-pip.py:", err)
		return err
	}

	// copy the requirements file to the python code directory using io.copy
	installRequirementsPath := filepath.Join(extractDir, requirementsFile)

	if err := copyFile(requirementsFile, installRequirementsPath); err != nil {
		fmt.Println("Error copying requirements file:", err)
		return err
	}

	if err := runCommand(pythonPath, []string{"-m", "pip", "wheel", "-w", filepath.Join(extractDir, "wheels"), "-r", requirementsFile}); err != nil {
		fmt.Println("Error building wheels:", err)
		return err
	}

	defer func(command string, args []string) {
		err := runCommand(command, args)
		if err != nil {
			fmt.Println("Error running command:", err)
		}
	}(pythonPath, []string{"-m", "pip", "cache", "purge"}) // Clean up the pip cache after the program finishes
	return nil
}

func downloadFile(url, filePath string) error {
	response, err := http.Get(url)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, response.Body)
	return err
}

func cleanDirectory(settings *common.PythonSetupSettings) {
	removeIfExists(settings.PythonExtractDir)
	removeIfExists(settings.PythonDownloadZip)

	println("Directory cleaned")
}

func removeIfExists(path string) {
	// Check if the path exists
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return
	}

	// Remove file or directory
	err = os.RemoveAll(path)
	if err != nil {
		fmt.Println("Error deleting:", err)
		return
	}

	fmt.Println("Successfully deleted:", path)
}

func copyFile(src, dst string) error {

	from, err := os.Open(src)
	if err != nil {
		fmt.Println("Error opening requirements file:", err)
		return err
	}

	to, err := os.Create(dst)
	if err != nil {
		fmt.Println("Error creating requirements file:", err)
		return err
	}

	_, err = io.Copy(to, from)
	if err != nil {
		fmt.Println("Error copying requirements file:", err)
		return err
	}

	return nil
}
