package common

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func ExtractZip(zipFile, extractDir string, skipLevels int) error {
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
