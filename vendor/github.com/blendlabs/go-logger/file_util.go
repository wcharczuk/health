package logger

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// File contains helper functions for files.
var File = &fileUtil{}

type fileUtil struct{}

// CreateOrOpen creates or opens a file.
func (fu fileUtil) CreateOrOpen(filePath string) (*os.File, error) {
	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	if os.IsNotExist(err) {
		return os.Create(filePath)
	}
	return f, err
}

// CreateAndClose creates and closes a file.
func (fu fileUtil) CreateAndClose(filePath string) error {
	f, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer f.Close()
	return nil
}

// RemoveMany removes an array of files.
func (fu fileUtil) RemoveMany(filePaths ...string) error {
	var err error
	for _, path := range filePaths {
		err = os.Remove(path)
		if err != nil {
			return err
		}
	}
	return err
}

func (fu fileUtil) List(path string, expr *regexp.Regexp) ([]string, error) {
	var files []string
	err := filepath.Walk(path, func(fullFilePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if expr == nil {
			files = append(files, fullFilePath)
		} else if expr.MatchString(info.Name()) {
			files = append(files, fullFilePath)
		}
		return nil
	})
	return files, err
}

func (fu fileUtil) ParseSize(fileSizeValue string, defaultFileSize int64) int64 {
	if len(fileSizeValue) == 0 {
		return defaultFileSize
	}

	if len(fileSizeValue) < 2 {
		val, err := strconv.Atoi(fileSizeValue)
		if err != nil {
			return defaultFileSize
		}
		return int64(val)
	}

	units := strings.ToLower(fileSizeValue[len(fileSizeValue)-2:])
	value, err := strconv.ParseInt(fileSizeValue[:len(fileSizeValue)-2], 10, 64)
	if err != nil {
		return defaultFileSize
	}
	switch units {
	case "gb":
		return value * Gigabyte
	case "mb":
		return value * Megabyte
	case "kb":
		return value * Kilobyte
	}
	fullValue, err := strconv.ParseInt(fileSizeValue, 10, 64)
	if err != nil {
		return defaultFileSize
	}
	return fullValue
}

// FormatFileSize returns a string representation of a file size in bytes.
func (fu fileUtil) FormatSize(sizeBytes int) string {
	if sizeBytes >= 1<<30 {
		return strconv.Itoa(sizeBytes/(1<<30)) + "gb"
	} else if sizeBytes >= 1<<20 {
		return strconv.Itoa(sizeBytes/(1<<20)) + "mb"
	} else if sizeBytes >= 1<<10 {
		return strconv.Itoa(sizeBytes/(1<<10)) + "kb"
	}
	return strconv.Itoa(sizeBytes)
}
