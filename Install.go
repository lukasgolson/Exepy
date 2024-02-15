package main

import (
	"archive/zip"
	_ "embed"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

//go:embed pipeline.zip
var pipelineZip []byte

const (
	pythonDownloadURL = "https://www.python.org/ftp/python/3.11.7/python-3.11.7-embed-amd64.zip"
	pythonEmbedZip    = "python-3.11.7-embed-amd64.zip"
	pythonExtractDir  = "python-embed"
	pthFile           = "python311._pth"
	pythonInteriorZip = "python311.zip"
)

func main() {

	// CHECK IF PYTHON IS INSTALLED IN LOCAL DIRECTORY
	if _, err := os.Stat(pythonExtractDir); os.IsNotExist(err) {

		// CREATE THE EXTRACTION DIRECTORY
		if err := os.Mkdir(pythonExtractDir, os.ModePerm); err != nil {
			fmt.Println("Error creating extraction directory:", err)
			return
		}

		// DOWNLOAD PYTHON ZIP FILE
		if err := downloadFile(pythonDownloadURL, pythonEmbedZip); err != nil {
			fmt.Println("Error downloading Python zip file:", err)
			return
		}

		// EXTRACT THE EMBEDDED PYTHON ZIP FILE
		if err := extractZip(pythonEmbedZip, pythonExtractDir, 0); err != nil {
			fmt.Println("Error extracting Python zip file:", err)
			return
		}

		// EXTRACT THE EMBEDDED PYTHON INTERIOR ZIP FILE
		if err := extractZip(filepath.Join(pythonExtractDir, pythonInteriorZip), pythonExtractDir, 0); err != nil {
			fmt.Println("Error extracting the interiorPython zip file:", err)
			return
		}

		// CLEAN UP THE EXTRACTED FILES
		if err := os.Remove(pythonEmbedZip); err != nil {
			fmt.Println("Error deleting downloaded zip file:", err)
		}

		if err := os.Remove(filepath.Join(pythonExtractDir, pythonInteriorZip)); err != nil {
			fmt.Println("Error deleting interior zip file:", err)
		}

		// Remove python39._pth file to avoid "import site" error
		if err := os.Remove(filepath.Join(pythonExtractDir, pthFile)); err != nil {
			fmt.Println("Error deleting pth file:", err)
		}

		pthFile, err := os.Create(filepath.Join(pythonExtractDir, pthFile))
		if err != nil {
			fmt.Println("Error creating ._pth file:", err)
			return
		}

		// change python311 to pythonExtractDir
		_, err = pthFile.WriteString(".\\" + pythonExtractDir + "\n.\\Scripts\n.\n# importing site will run sitecustomize.py\nimport site")
		if err != nil {
			fmt.Println("Error writing to ._pth file:", err)
			return
		}

		// write to sitecustomize.py file
		sitecustomizeFile, err := os.Create(filepath.Join(pythonExtractDir, "sitecustomize.py"))

		_, err = sitecustomizeFile.WriteString("import sys\nsys.path.append('.')")

		// make empty DLLs folder
		if err := os.Mkdir(filepath.Join(pythonExtractDir, "DLLs"), os.ModePerm); err != nil {
			fmt.Println("Error creating DLLs folder:", err)
			return
		}
	}

	// EXTRACT THE PIPELINE ZIP FILE

	// save the pipeline zip file to the current directory
	if err := os.WriteFile("pipeline.zip", pipelineZip, os.ModePerm); err != nil {
		fmt.Println("Error saving pipeline zip file:", err)
		return
	}

	if err := extractZip("pipeline.zip", "", 1); err != nil {
		fmt.Println("Error extracting pipeline zip file:", err)
		return
	}

	// delete the pipeline zip file
	if err := os.Remove("pipeline.zip"); err != nil {
		fmt.Println("Error deleting pipeline zip file:", err)
	}

	if err := runCommand(filepath.Join(pythonExtractDir, "python.exe"), "setup.py"); err != nil {
		fmt.Println("Error running setup.py:", err)
		return
	}

	if err := runCommand(filepath.Join(pythonExtractDir, "python.exe"), "videoToPointcloud.py"); err != nil {
		fmt.Println("Error running Python script:", err)
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

func extractZip(zipFile, extractDir string, skipLevels int) error {
	r, err := zip.OpenReader(zipFile)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, file := range r.File {
		// Split the file's path into components
		components := strings.Split(file.Name, "/")

		// Skip the first n levels
		if len(components) > skipLevels {
			relativePath := strings.Join(components[skipLevels:], "/")
			path := filepath.Join(extractDir, relativePath)

			if file.FileInfo().IsDir() {
				os.MkdirAll(path, os.ModePerm)
				continue
			}

			if err := os.MkdirAll(filepath.Dir(path), os.ModePerm); err != nil {
				return err
			}

			outFile, err := os.Create(path)
			if err != nil {
				return err
			}
			defer outFile.Close()

			rc, err := file.Open()
			if err != nil {
				return err
			}
			defer rc.Close()

			_, err = io.Copy(outFile, rc)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func runCommand(command string, args ...string) error {
	cmd := exec.Command(command, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func runPythonScript(script string) error {
	if err := runCommand(filepath.Join(pythonExtractDir, "python.exe"), script); err != nil {
		fmt.Println("Error running setup.py:", err)

		return nil

	} else {
		return err
	}
}
