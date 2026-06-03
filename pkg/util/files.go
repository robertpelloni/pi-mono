package util

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

var (
	fileCache      []string
	lastCacheTime  int64
	cacheDuration  = int64(30) // seconds
)

// ListFilesRecursively returns a list of files in the given directory up to a maximum number.
func ListFilesRecursively(root string, max int) []string {
	now := time.Now().Unix()

	if len(fileCache) > 0 && now-lastCacheTime < cacheDuration {
		return fileCache
	}

	var files []string
	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if len(files) >= max {
			return filepath.SkipDir
		}
		if info.IsDir() {
			// Skip hidden directories like .git
			if info.Name() != "." && strings.HasPrefix(info.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}

		rel, err := filepath.Rel(root, path)
		if err == nil {
			files = append(files, rel)
		}
		return nil
	})

	fileCache = files
	lastCacheTime = now
	return files
}
