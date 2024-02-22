package common

import (
	"crypto/md5"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// https://stackoverflow.com/a/40436529 CC BY-SA 4.0
func Md5SumFile(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func Md5sumDirectory(dirPath string) (string, error) {
	var hashes []string

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()

			hash := md5.New()
			if _, err := io.Copy(hash, file); err != nil {
				return err
			}

			hashes = append(hashes, hex.EncodeToString(hash.Sum(nil)))
		}

		return nil
	})

	if err != nil {
		return "", err
	}

	// Sort the hashes to ensure consistent results
	sort.Strings(hashes)

	// Combine the hashes into a single string
	combined := strings.Join(hashes, "")

	// Hash the combined string
	finalHash := md5.Sum([]byte(combined))

	return hex.EncodeToString(finalHash[:]), nil
}

func HashReadSeeker(rs io.ReadSeeker) (string, error) {
	// Save the current position
	startPos, err := rs.Seek(0, io.SeekCurrent)
	if err != nil {
		return "", err
	}

	hash := md5.New()
	if _, err := io.Copy(hash, rs); err != nil {
		return "", err
	}

	// Restore the position
	_, err = rs.Seek(startPos, io.SeekStart)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}
