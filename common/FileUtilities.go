package common

import (
	"fmt"
	"io"
	"net/http"
	"os"
)

func DownloadFile(url, filePath string) error {
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

func CopyFile(src, dst string) error {

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

func DoesPathExist(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

func RemoveIfExists(path string) {
	if DoesPathExist(path) {
		err := os.RemoveAll(path)

		if err != nil {
			return
		}

		println("Removed file: ", path)
	}
}

func SaveContentsToFile(filename, contents string) error {
	hashFile, err := os.Create(filename)
	if err != nil {
		return err
	}

	defer hashFile.Close()

	_, err = hashFile.WriteString(contents)
	return err
}

func ListFilesInDir(dir string) ([]string, error) {

	var files []string

	fileInfo, err := os.ReadDir(dir)
	if err != nil {
		return files, err
	}

	for _, file := range fileInfo {
		files = append(files, file.Name())
	}

	return files, nil
}
