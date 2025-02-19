package common

import (
	"bytes"
	"compress/gzip"
	"dirstream"
	"fmt"
	"io"
)

func DirToStream(directoryPath string, ignoredDirs []string) (io.ReadSeeker, error) {
	files, err := dirstream.BuildRelativeFileList(directoryPath, ignoredDirs)
	if err != nil {
		return nil, fmt.Errorf("failed to build file list: %w", err)
	}

	return FilesToStream(directoryPath, files, false)

}

// FilesToStream compresses the files in the given directory and returns a stream of the compressed data.
// If flatPaths is true, the files will be stored in the archive without their directory structure.
func FilesToStream(directoryPath string, files []string, flatPaths bool) (io.ReadSeeker, error) {
	encoder := dirstream.NewEncoder(directoryPath, dirstream.DefaultChunkSize)
	encoderStream, err := encoder.Encode(files, flatPaths)
	if err != nil {
		return nil, fmt.Errorf("failed to encode directory: %w", err)
	}

	var buf bytes.Buffer

	gzipWriter := gzip.NewWriter(&buf)

	if _, err := io.Copy(gzipWriter, encoderStream); err != nil {
		gzipWriter.Close() // ensure we close on error
		return nil, fmt.Errorf("failed to compress data: %w", err)
	}

	// Close the gzip writer to flush all data into the buffer.
	if err := gzipWriter.Close(); err != nil {
		return nil, fmt.Errorf("failed to close gzip writer: %w", err)
	}

	return bytes.NewReader(buf.Bytes()), nil
}

func StreamToDir(IOReader io.Reader, outputDir string) error {
	// Create a gzip reader from the input stream.
	gzipReader, err := gzip.NewReader(IOReader)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzipReader.Close()

	// Pass the decompressed stream to the dirstream decoder.

	decoder, err := dirstream.NewDecoder(outputDir, false, dirstream.DefaultChunkSize)
	if err != nil {
		return fmt.Errorf("failed to create decoder: %w", err)
	}

	if err := decoder.Decode(gzipReader); err != nil {
		return fmt.Errorf("failed to decode stream: %w", err)
	}

	return nil
}
