package utils

import (
	"io"
	"sync"
)

var copyBufPool = sync.Pool{
	New: func() any {
		return make([]byte, 4096)
	},
}

func CopyZeroAlloc(w io.Writer, r io.Reader) (int64, error) {
	vbuf := copyBufPool.Get()

	buf, ok := vbuf.([]byte)
	if !ok {
		buf = make([]byte, 4096)
	}

	n, err := io.CopyBuffer(w, r, buf)

	copyBufPool.Put(vbuf)

	return n, err
}
