package logger

import (
	"bytes"
	"sync"
)

// NewBufferPool returns a new BufferPool.
func NewBufferPool(bufferSize int) *BufferPool {
	return &BufferPool{
		Pool: sync.Pool{New: func() interface{} {
			b := bytes.NewBuffer(make([]byte, bufferSize))
			b.Reset()
			return b
		}},
	}
}

// BufferPool is a sync.Pool of bytes.Buffer.
type BufferPool struct {
	sync.Pool
}

// Get returns a pooled bytes.Buffer instance.
func (bp *BufferPool) Get() *bytes.Buffer {
	return bp.Pool.Get().(*bytes.Buffer)
}

// Put returns the pooled instance.
func (bp *BufferPool) Put(b *bytes.Buffer) {
	b.Reset()
	bp.Pool.Put(b)
}
