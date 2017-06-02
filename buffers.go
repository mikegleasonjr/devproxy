package devproxy

import (
	"sync"
)

const (
	buffersSize = 256 * 1024
)

type bufferPool struct {
	*sync.Pool
}

var buffers = &bufferPool{&sync.Pool{
	New: func() interface{} { return make([]byte, buffersSize) },
}}

func (p *bufferPool) Get() []byte {
	return p.Pool.Get().([]byte)
}

func (p *bufferPool) Put(b []byte) {
	p.Pool.Put(b)
}
