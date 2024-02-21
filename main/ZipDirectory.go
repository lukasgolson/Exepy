package main

import (
	"archive/zip"
	"context"
	"fmt"
	"github.com/mholt/archiver/v4"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func zipDirectory(directoryPath, zipfilename string) error {
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

	// we can use the CompressedArchive type to gzip a tarball
	// (compression is not required; you could use Tar directly)
	format := archiver.CompressedArchive{
		Compression: archiver.Gz{},
		Archival:    archiver.Tar{},
	}

	// create the archive
	err = format.Archive(context.Background(), out, files)
	if err != nil {
		return err
	}

	return nil
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
