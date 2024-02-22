package common

import (
	"bytes"
	"context"
	"github.com/mholt/archiver/v4"
	"io"
	"os"
	"path/filepath"
)

func getFormat() archiver.CompressedArchive {
	format := archiver.CompressedArchive{
		Compression: archiver.Xz{},
		Archival:    archiver.Tar{},
	}
	return format
}

func CompressDirToStream(directoryPath string) (io.ReadSeeker, error) {
	// Get the list of files and directories in the specified folder
	FromDiskOptions := &archiver.FromDiskOptions{
		FollowSymlinks:  false,
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

		outPath := filepath.Join(outputDir, archivedFile.NameInArchive)

		if archivedFile.FileInfo.IsDir() {
			err := os.MkdirAll(outPath, os.ModePerm)
			if err != nil {
				return err
			}

			return nil
		} else {
			dir := filepath.Dir(outPath)
			err := os.MkdirAll(dir, os.ModePerm)

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

	return nil
}

func mapFilesAndDirectories(directoryPath string) (map[string]string, error) {

	pathSeperator := string(os.PathSeparator)

	// Initialize a map to store file names and their corresponding paths
	fileMap := make(map[string]string)

	// Walk through the directory
	err := filepath.WalkDir(directoryPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		relativeDirPath, err := filepath.Rel(directoryPath, path)
		if err != nil {
			return err
		}

		if d.IsDir() {

			// Skip the root directory
			if relativeDirPath == directoryPath {
				return nil
			}

			// Check if the directory is empty
			isEmpty, err := isDirEmpty(path)
			if err != nil {
				return err
			}

			// Use os.PathSeparator for the key and "/" for the value
			if isEmpty {
				fileMap[path] = filepath.ToSlash(relativeDirPath + pathSeperator)
			}

			return nil
		}

		fileMap[path] = filepath.ToSlash(relativeDirPath)

		return nil
	})

	// Check for errors during the walk
	if err != nil {
		return nil, err
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
