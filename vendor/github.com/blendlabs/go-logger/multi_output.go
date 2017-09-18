package logger

import (
	"io"
	"os"
)

// NewMultiOutputFromEnvironment creates a new multiplexed stdout writer.
func NewMultiOutputFromEnvironment() io.Writer {
	primary := os.Stdout
	filePath := os.Getenv(EnvironmentVariableLogOutFile)
	if len(filePath) > 0 {
		secondary, err := NewFileOutputFromEnvironment(
			EnvironmentVariableLogOutFile,
			EnvironmentVariableLogOutArchiveCompress,
			EnvironmentVariableLogOutMaxSizeBytes,
			EnvironmentVariableLogOutMaxArchive,
		)
		if err != nil {
			panic(err)
		}
		return NewMultiOutput(primary, secondary)
	}
	return NewSyncOutput(primary)
}

// NewErrorMultiOutputFromEnvironment creates a new multiplexed stderr writer.
func NewErrorMultiOutputFromEnvironment() io.Writer {
	primary := os.Stderr
	filePath := os.Getenv(EnvironmentVariableLogErrFile)
	if len(filePath) > 0 {
		secondary, err := NewFileOutputFromEnvironment(
			EnvironmentVariableLogErrFile,
			EnvironmentVariableLogErrArchiveCompress,
			EnvironmentVariableLogErrMaxSizeBytes,
			EnvironmentVariableLogErrMaxArchive,
		)
		if err != nil {
			panic(err)
		}
		return NewMultiOutput(primary, secondary)
	}
	return NewSyncOutput(primary)
}

// NewMultiOutput creates a new MultiOutput that wraps an array of writers.
func NewMultiOutput(outputs ...io.Writer) *MultiOutput {
	return &MultiOutput{
		outputs: outputs,
	}
}

// MultiOutput writes to many writers at once.
type MultiOutput struct {
	outputs []io.Writer
}

func (mo MultiOutput) Write(buffer []byte) (int, error) {
	var written int
	var err error

	for x := 0; x < len(mo.outputs); x++ {
		if mo.outputs[x] != nil {
			written, err = mo.outputs[x].Write(buffer)
		}
	}
	return written, err
}

// Close closes all of the inner writers (if they are io.WriteClosers).
func (mo MultiOutput) Close() error {
	var err error
	var closeErr error
	for x := 0; x < len(mo.outputs); x++ {
		if typed, isTyped := mo.outputs[x].(io.Closer); isTyped {
			closeErr = typed.Close()
			if closeErr != nil {
				err = closeErr
			}
		}
	}
	return err
}
