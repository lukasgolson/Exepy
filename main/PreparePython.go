package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

const (
	resultZip    = "python-embed.gzip"
	settingsFile = "settings.json"
)

func main() {

	settings, err := loadOrSaveDefault(settingsFile)
	if err != nil {
		return
	}

	cleanDirectory(settings)

	removeIfExists(resultZip)

	defer cleanDirectory(settings) // Clean up the directory after the program finishes

	// CREATE THE EXTRACTION DIRECTORY
	if _, err := os.Stat(settings.PythonExtractDir); os.IsNotExist(err) {
		if err := os.Mkdir(settings.PythonExtractDir, os.ModePerm); err != nil {
			fmt.Println("Error creating extraction directory:", err)
			return
		}
	}

	// DOWNLOAD PYTHON ZIP FILE
	if err := downloadFile(settings.PythonDownloadURL, settings.PythonEmbedZip); err != nil {
		fmt.Println("Error downloading Python zip file:", err)
		return
	}

	// EXTRACT THE EMBEDDED PYTHON ZIP FILE
	if err := extractZip(settings.PythonEmbedZip, settings.PythonExtractDir, 0); err != nil {
		fmt.Println("Error extracting Python zip file:", err)
		return
	}

	// EXTRACT THE EMBEDDED PYTHON INTERIOR ZIP FILE
	if err := extractZip(filepath.Join(settings.PythonExtractDir, settings.PythonInteriorZip), settings.PythonExtractDir, 0); err != nil {
		fmt.Println("Error extracting the interiorPython zip file:", err)
		return
	}

	// CLEAN UP THE EXTRACTED FILES
	removeIfExists(settings.PythonEmbedZip)
	removeIfExists(filepath.Join(settings.PythonExtractDir, settings.PythonInteriorZip))
	removeIfExists(filepath.Join(settings.PythonExtractDir, settings.PthFile))

	// write to ._pth file
	pthFile, err := os.Create(filepath.Join(settings.PythonExtractDir, settings.PthFile))
	if err != nil {
		fmt.Println("Error creating ._pth file:", err)
		return
	}

	// change python311 to pythonExtractDir
	_, err = pthFile.WriteString(".\\" + settings.PythonExtractDir + "\n.\\Scripts\n.\n# importing site will run sitecustomize.py\nimport site")
	if err != nil {
		fmt.Println("Error writing to ._pth file:", err)
		return
	}

	// write to sitecustomize.py file
	sitecustomizeFile, err := os.Create(filepath.Join(settings.PythonExtractDir, "sitecustomize.py"))
	_, err = sitecustomizeFile.WriteString("import sys\nsys.path.append('.')")

	// make empty DLLs folder
	if err := os.Mkdir(filepath.Join(settings.PythonExtractDir, "DLLs"), os.ModePerm); err != nil {
		fmt.Println("Error creating DLLs folder:", err)
		return
	}

	if err := zipDirectory(settings.PythonExtractDir, resultZip); err != nil {
		fmt.Println("Error zipping Python directory:", err)
		return
	}

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

func cleanDirectory(settings *pythonSetupSettings) {
	removeIfExists(settings.PythonExtractDir)
	removeIfExists(settings.PythonEmbedZip)

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
