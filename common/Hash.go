package common

import (
	"crypto/md5"
	"encoding/hex"
	"io"
	"os"
)

// https://stackoverflow.com/a/40436529 CC BY-SA 4.0
func md5sum(filePath string) (string, error) {
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
