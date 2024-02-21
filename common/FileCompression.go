package common

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"github.com/mholt/archiver/v4"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func getFormat() archiver.CompressedArchive {
	format := archiver.CompressedArchive{
		Compression: archiver.Gz{},
		Archival:    archiver.Tar{},
	}
	return format
}

func CompressDir(directoryPath, zipfilename string) error {
	// Get the list of files and directories in the specified folder

	FromDiskOptions := &archiver.FromDiskOptions{
		FollowSymlinks:  true,
		ClearAttributes: true,
	}

	// map the files to the archive

	pathMap, err := mapFilesAndDirectories(directoryPath)
	if err != nil {
		return err
	}

	// Create a new zip archive
	files, err := archiver.FilesFromDisk(FromDiskOptions, pathMap)

	if err != nil {
		return err
	}

	// create the output file we'll write to
	out, err := os.Create(zipfilename)
	if err != nil {
		return err
	}
	defer func(out *os.File) {
		err := out.Close()
		if err != nil {
			fmt.Println("Error closing file:", err)
		}
	}(out)

	format := getFormat()

	// create the archive
	err = format.Archive(context.Background(), out, files)
	if err != nil {
		return err
	}

	return nil
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

func DecompressIOStream(IOReader io.Reader, directoryPath string) error {

	format := getFormat()

	handler := func(ctx context.Context, f archiver.File) error {

		// Create parent directories
		err := os.MkdirAll(filepath.Join(directoryPath, filepath.Dir(f.Name())), os.ModePerm)

		if err != nil {
			fmt.Println("Error creating parent directories:", err)
		}

		if !f.FileInfo.IsDir() {

			// write bytes to file
			outFile, err := os.Create(filepath.Join(directoryPath, f.Name()))

			reader, _ := f.Open()

			// gross but it works
			fileContents, err := io.ReadAll(reader)

			defer func(reader io.ReadCloser) {
				err := reader.Close()
				if err != nil {
					fmt.Println("Error closing reader:", err)
				}
			}(reader)

			_, err = io.Copy(outFile, bytes.NewReader(fileContents))
			if err != nil {
				fmt.Println("Error copying file", err)
			}
		}

		return nil
	}

	ctx := context.Background()

	err := format.Extract(ctx, IOReader, nil, handler)
	if err != nil {
		return err
	}

	return nil
}

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

func mapFilesAndDirectories(root string) (map[string]string, error) {
	paths := make(map[string]string)

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip the root directory
		if path == root {
			return nil
		}

		// Calculate the relative path
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}

		// Add the path and its relative path to the map
		paths[path] = rel

		return nil
	})

	if err != nil {
		return nil, err
	}

	return paths, nil
}
