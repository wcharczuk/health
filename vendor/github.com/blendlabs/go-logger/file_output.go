package logger

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"sync"

	exception "github.com/blendlabs/go-exception"
)

const (
	isArchiveFileRegexpFormat           = `%s\.([0-9]+)?`
	isCompressedArchiveFileRegexpFormat = `%s\.([0-9]+)?\.gz`

	// Kilobyte represents the bytes in a kilobyte.
	Kilobyte int64 = 1 << 10
	// Megabyte represents the bytes in a megabyte.
	Megabyte int64 = Kilobyte << 10
	// Gigabyte represents the bytes in a gigabyte.
	Gigabyte int64 = Megabyte << 10
)

const (
	//FileOutputUnlimitedSize is a preset for the size of the files that can be written to be unlimited.
	FileOutputUnlimitedSize int64 = 0

	// FileOutputDefaultFileSize is the default file size (50mb).
	FileOutputDefaultFileSize int64 = 50 * Megabyte

	// FileOutputUnlimitedArchiveFiles is a preset for the number of archive files to be kept to be unlimited.
	FileOutputUnlimitedArchiveFiles int64 = 0

	// FileOutputDefaultMaxArchiveFiles is the default number of archive files (10).
	FileOutputDefaultMaxArchiveFiles int64 = 10
)

// NewFileOutput creates a new file writer.
func NewFileOutput(filePath string, shouldCompressArchivedFiles bool, fileMaxSizeBytes, fileMaxArchiveCount int64) (*FileOutput, error) {
	file, err := File.CreateOrOpen(filePath)
	if err != nil {
		return nil, err
	}

	var regex *regexp.Regexp
	if shouldCompressArchivedFiles {
		regex, err = createIsCompressedArchiveFileRegexp(filePath)
	} else {
		regex, err = createIsArchivedFileRegexp(filePath)
	}
	if err != nil {
		return nil, err
	}

	return &FileOutput{
		filePath:                    filePath,
		file:                        file,
		syncRoot:                    &sync.Mutex{},
		isArchiveFileRegexp:         regex,
		shouldCompressArchivedFiles: shouldCompressArchivedFiles,
		fileMaxSizeBytes:            fileMaxSizeBytes,
		fileMaxArchiveCount:         fileMaxArchiveCount,
	}, nil
}

// NewFileOutputFromEnvironment creates a new FileOutput from the given environment variable names.`
func NewFileOutputFromEnvironment(pathVar, shouldCompressVar, maxSizeVar, maxArchiveVar string) (*FileOutput, error) {
	filePath := os.Getenv(pathVar)
	if len(filePath) == 0 {
		return nil, fmt.Errorf("Environment Variable `%s` required", pathVar)
	}

	shouldCompress := envFlagIsSet(shouldCompressVar, false)
	maxFileSize := File.ParseSize(os.Getenv(maxSizeVar), FileOutputDefaultFileSize)
	maxArchive := envFlagInt64(maxArchiveVar, FileOutputDefaultMaxArchiveFiles)
	return NewFileOutput(filePath, shouldCompress, maxFileSize, maxArchive)
}

// NewFileOutputWithDefaults returns a new file writer with defaults.
func NewFileOutputWithDefaults(filePath string) (*FileOutput, error) {
	return NewFileOutput(filePath, true, FileOutputDefaultFileSize, FileOutputDefaultMaxArchiveFiles)
}

// FileOutput implements the file rotation settings from supervisor.
type FileOutput struct {
	filePath string
	file     *os.File

	syncRoot                    *sync.Mutex
	shouldCompressArchivedFiles bool

	fileMaxSizeBytes    int64
	fileMaxArchiveCount int64

	isArchiveFileRegexp *regexp.Regexp
}

// Write writes to the file.
func (fo *FileOutput) Write(buffer []byte) (int, error) {
	fo.syncRoot.Lock()
	defer fo.syncRoot.Unlock()

	if fo.fileMaxSizeBytes > 0 {
		stat, err := fo.file.Stat()
		if err != nil {
			return 0, exception.New(err)
		}

		if stat.Size() > fo.fileMaxSizeBytes {
			err = fo.rotateFile()
			if err != nil {
				return 0, exception.New(err)
			}
		}
	}

	written, err := fo.file.Write(buffer)
	return written, exception.Wrap(err)
}

// Close closes the stream.
func (fo *FileOutput) Close() error {
	if fo.file != nil {
		err := fo.file.Close()
		fo.file = nil
		return err
	}
	return nil
}

func (fo *FileOutput) makeArchiveFilePath(filePath string, index int64) string {
	return fmt.Sprintf("%s.%d", filePath, index)
}

func (fo *FileOutput) makeCompressedArchiveFilePath(filePath string, index int64) string {
	return fmt.Sprintf("%s.%d.gz", filePath, index)
}

func (fo *FileOutput) makeTempArchiveFilePath(filePath string, index int64) string {
	return fmt.Sprintf("%s.%d.tmp", filePath, index)
}

func (fo *FileOutput) makeTempCompressedArchiveFilePath(filePath string, index int64) string {
	return fmt.Sprintf("%s.%d.gz.tmp", filePath, index)
}

func (fo *FileOutput) compressFile(inFilePath, outFilePath string) error {
	inFile, err := os.Open(inFilePath)
	if err != nil {
		return err
	}
	defer inFile.Close()

	outFile, err := os.Create(outFilePath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	gzw := gzip.NewWriter(outFile)
	defer gzw.Close()

	_, err = io.Copy(gzw, inFile)
	if err != nil {
		return err
	}
	return gzw.Flush()
}

func (fo *FileOutput) extractArchivedFileIndex(filePath string) (int64, error) {
	filePathBase := filepath.Base(filePath)
	values := fo.isArchiveFileRegexp.FindStringSubmatch(filePathBase)
	if len(values) > 1 {
		value, err := strconv.ParseInt(values[1], 10, 32)
		return value, exception.Wrap(err)
	}
	return 0, exception.Newf("Cannot extract file index from `%s`", filePathBase)
}

func (fo *FileOutput) isArchivedFile(filePath string) bool {
	return fo.isArchiveFileRegexp.MatchString(filePath)
}

func (fo *FileOutput) getArchivedFilePaths() ([]string, error) {
	return File.List(filepath.Dir(fo.filePath), fo.isArchiveFileRegexp)
}

func (fo *FileOutput) getMaxArchivedFileIndex(paths []string) (int64, error) {
	var err error
	var index int64
	max := int64(-1 << 63)
	for _, path := range paths {
		index, err = fo.extractArchivedFileIndex(path)
		if err != nil {
			return max, err
		}
		if index > max {
			max = index
		}
	}
	return max, err
}

func (fo *FileOutput) shiftArchivedFiles(paths []string) error {
	var index int64
	var err error

	intermediatePaths := make(map[string]string)
	var tempPath, finalPath string

	for _, path := range paths {
		index, err = fo.extractArchivedFileIndex(path)
		if err != nil {
			return err
		}
		if fo.shouldCompressArchivedFiles {
			tempPath = fo.makeTempCompressedArchiveFilePath(fo.filePath, index+1)
			finalPath = fo.makeCompressedArchiveFilePath(fo.filePath, index+1)
		} else {
			tempPath = fo.makeTempArchiveFilePath(fo.filePath, index+1)
			finalPath = fo.makeArchiveFilePath(fo.filePath, index+1)
		}

		if fo.fileMaxArchiveCount > 0 {
			if index+1 <= fo.fileMaxArchiveCount {
				err = os.Rename(path, tempPath)
				intermediatePaths[tempPath] = finalPath
			} else {
				err = os.Remove(path)
			}
		} else {
			err = os.Rename(path, tempPath)
			intermediatePaths[tempPath] = finalPath
		}
		if err != nil {
			return err
		}
	}

	for from, to := range intermediatePaths {
		err = os.Rename(from, to)
		if err != nil {
			return err
		}
	}

	return nil
}

func (fo *FileOutput) rotateFile() error {
	var err error

	paths, err := fo.getArchivedFilePaths()
	if err != nil {
		return err
	}

	err = fo.shiftArchivedFiles(paths)
	if err != nil {
		return err
	}

	err = fo.file.Close()
	if err != nil {
		return err
	}

	if fo.shouldCompressArchivedFiles {
		err = fo.compressFile(fo.filePath, fo.makeCompressedArchiveFilePath(fo.filePath, 1))
		if err != nil {
			return err
		}
		err = os.Remove(fo.filePath)
		if err != nil {
			return err
		}
	} else {
		err = os.Rename(fo.filePath, fo.makeArchiveFilePath(fo.filePath, 1))
	}

	file, err := os.Create(fo.filePath)
	if err != nil {
		return err
	}
	fo.file = file
	return nil
}

func createIsArchivedFileRegexp(filePath string) (*regexp.Regexp, error) {
	return regexp.Compile(fmt.Sprintf(isArchiveFileRegexpFormat, filepath.Base(filePath)))
}

func createIsCompressedArchiveFileRegexp(filePath string) (*regexp.Regexp, error) {
	return regexp.Compile(fmt.Sprintf(isCompressedArchiveFileRegexpFormat, filepath.Base(filePath)))
}
