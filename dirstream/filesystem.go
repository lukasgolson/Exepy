package dirstream

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func sanitizePath(rootPath, filePath string) (string, error) {
	// Convert rootPath to an absolute path and clean it.
	absRootPath, err := filepath.Abs(rootPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve destination path: %w", err)
	}
	absRootPath = filepath.Clean(absRootPath)
	// Ensure trailing separator for proper prefix checking.
	if !strings.HasSuffix(absRootPath, string(filepath.Separator)) {
		absRootPath += string(filepath.Separator)
	}

	// If filePath is empty or ".", return absRootPath directly.
	if filePath == "" || filePath == "." {
		return absRootPath, nil
	}

	// Reject absolute file paths immediately.
	if filepath.IsAbs(filePath) {
		return "", fmt.Errorf("invalid path: absolute paths not allowed: %s", filePath)
	}

	// Join rootPath and filePath, then clean the result.
	joinedPath := filepath.Join(absRootPath, filePath)
	cleanPath := filepath.Clean(joinedPath)

	// Convert the cleaned path to an absolute path.
	absCleanPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve clean path: %w", err)
	}
	absCleanPath = filepath.Clean(absCleanPath)

	// Allow if the cleaned path equals the root path.
	if absCleanPath == absRootPath {
		return absCleanPath, nil
	}

	// Prevent directory traversal: ensure the absolute clean path starts with the absolute root path.
	if !strings.HasPrefix(absCleanPath, absRootPath) {
		return "", fmt.Errorf("invalid path: %s", filePath)
	}

	// Check each component of the path; reject if any are exactly "..".
	parts := strings.Split(absCleanPath, string(filepath.Separator))
	for _, part := range parts {
		if part == ".." {
			return "", fmt.Errorf("invalid path contains '..': %s", filePath)
		}
	}

	return absCleanPath, nil
}

func BuildRelativeFileList(rootPath string, excludes []string) ([]string, error) {
	var files []string

	err := filepath.WalkDir(rootPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Get the relative path from the root
		relPath, err := filepath.Rel(rootPath, path)
		if err != nil {
			return err
		}

		// Skip the root path itself.
		if relPath == "." {
			return nil
		}

		// Check if the current directory or file should be excluded.
		for _, exclude := range excludes {
			// Use filepath.Base to get the name of the file or directory.
			baseName := filepath.Base(path)

			// Only skip if the base name exactly matches the exclude pattern.
			if baseName == exclude {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		files = append(files, relPath)
		return nil
	})

	return files, err
}
