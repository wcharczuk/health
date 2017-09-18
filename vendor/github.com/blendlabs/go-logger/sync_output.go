package logger

import (
	"io"
	"sync"
)

// NewSyncOutput returns a new interlocked writer.
func NewSyncOutput(output io.Writer) io.Writer {
	return &SyncOutput{
		output:   output,
		syncRoot: &sync.Mutex{},
	}
}

// SyncOutput is a writer that serializes access to the Write() method.
type SyncOutput struct {
	output   io.Writer
	syncRoot *sync.Mutex
}

// Write writes the given bytes to the inner writer.
func (so *SyncOutput) Write(buffer []byte) (int, error) {
	so.syncRoot.Lock()
	defer so.syncRoot.Unlock()

	return so.output.Write(buffer)
}

/* experimental; we cannot close stdout or stderr
otherwise the program crashes
// Close is a no-op.
func (so SyncOutput) Close() error {
	if closer, isCloser := so.output.(io.Closer); isCloser {
		so.syncRoot.Lock()
		defer so.syncRoot.Unlock()
		return closer.Close()
	}
	return nil
}*/
