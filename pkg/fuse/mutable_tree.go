package fuse

type MutableTree interface {
	NewFile() (RandomAccessFile, error)
}
