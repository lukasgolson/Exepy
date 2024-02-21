package common

import (
	"bytes"
	"context"
	"fmt"
	"github.com/mholt/archiver/v4"
	"io"
	"os"
	"path/filepath"
)

func getFormat() archiver.CompressedArchive {
	format := archiver.CompressedArchive{
		Compression: archiver.Gz{},
		Archival:    archiver.Tar{},
	}
	return format
}

func CompressDirToStream(directoryPath string) (io.ReadSeeker, error) {
	// Get the list of files and directories in the specified folder
	FromDiskOptions := &archiver.FromDiskOptions{
		FollowSymlinks:  true,
		ClearAttributes: true,
	}

	// map the files to the archive
	pathMap, err := mapFilesAndDirectories(directoryPath)
	if err != nil {
		return nil, err
	}

	// Create a new zip archive
	files, err := archiver.FilesFromDisk(FromDiskOptions, pathMap)
	if err != nil {
		return nil, err
	}

	// create a buffer to hold the compressed data
	buf := new(bytes.Buffer)

	format := getFormat()

	// create the archive
	err = format.Archive(context.Background(), buf, files)
	if err != nil {
		return nil, err
	}

	// convert the buffer to an io.ReadSeeker
	readSeeker := bytes.NewReader(buf.Bytes())

	return readSeeker, nil
}

func DecompressIOStream(IOReader io.Reader, outputDir string) error {

	format := getFormat()

	handler := func(ctx context.Context, archivedFile archiver.File) error {

		outPath := filepath.Join(outputDir, archivedFile.FileInfo.Name())

		if archivedFile.FileInfo.IsDir() {
			err := os.MkdirAll(outPath, os.ModePerm)
			if err != nil {
				return err
			}

			fmt.Println("Created leaf directory:", outPath)

			return nil
		} else {
			dir := filepath.Dir(outPath)
			err := os.MkdirAll(dir, os.ModePerm)

			fmt.Println("Created directory:", dir)

			if err != nil {
				return err
			}
		}

		// Create the outputFileStream
		outputFileStream, err := os.Create(outPath)
		if err != nil {
			return err
		}

		defer outputFileStream.Close()

		archivedFileStream, err := archivedFile.Open()
		if err != nil {
			return err
		}
		defer archivedFileStream.Close()

		// Write the outputFileStream
		_, err = io.Copy(outputFileStream, archivedFileStream)

		fmt.Println("Created file:", outPath)

		if err != nil {
			return err
		}

		return nil
	}

	ctx := context.Background()

	err := format.Extract(ctx, IOReader, nil, handler)
	if err != nil {
		return err
	}

	fmt.Println("Extraction complete to:", outputDir)

	return nil
}

func mapFilesAndDirectories(directoryPath string) (map[string]string, error) {
	// Initialize a map to store file names and their corresponding paths
	fileMap := make(map[string]string)

	// Walk through the directory
	err := filepath.WalkDir(directoryPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Get the relative directory path without the file name
		dirPath := filepath.Dir(path)
		relativeDirPath, err := filepath.Rel(directoryPath, dirPath)
		if err != nil {
			return err
		}

		if d.IsDir() {
			if dirPath == directoryPath {
				return nil
			}

			isEmpty, err := isDirEmpty(path)
			if err != nil {
				return err
			}

			if isEmpty {
				fileMap[path] = relativeDirPath + "/"
			}

			return nil
		}

		// Use os.PathSeparator for the key and "/" for the value
		fileMap[path] = relativeDirPath + "/"

		return nil
	})

	// Check for errors during the walk
	if err != nil {
		return nil, err
	}

	// loop through the map and print the key-value pairs
	for key, value := range fileMap {
		fmt.Println("Disk Path:", key, "Archive Path:", value)
	}

	return fileMap, nil
}

func isDirEmpty(dirPath string) (bool, error) {
	dir, err := os.Open(dirPath)
	if err != nil {
		return false, err
	}
	defer dir.Close()

	_, err = dir.Readdirnames(1)
	if err == nil {
		// Directory is not empty
		return false, nil
	}

	// Directory is empty or an error occurred
	return true, nil
}
