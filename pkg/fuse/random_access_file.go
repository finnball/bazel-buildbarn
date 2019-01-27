package fuse

import (
	"io"
)

type RandomAccessFile interface {
	io.Closer
	io.ReaderAt
	io.WriterAt

	Truncate(size int64) error
}
