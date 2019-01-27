package fuse

import (
	"os"
	"strconv"
	"sync"

	"github.com/EdSchouten/bazel-buildbarn/pkg/filesystem"
)

type directoryBackedMutableTree struct {
	directory filesystem.Directory

	lock   sync.Mutex
	nextID uint64
}

func NewDirectoryBackedMutableTree(directory filesystem.Directory) MutableTree {
	return &directoryBackedMutableTree{
		directory: directory,
	}
}

func (mt *directoryBackedMutableTree) NewFile() (RandomAccessFile, error) {
	mt.lock.Lock()
	id := mt.nextID
	mt.nextID++
	mt.lock.Unlock()
	return &lazyOpeningSelfDeletingFile{
		directory: mt.directory,
		name:      strconv.FormatUint(id, 10),
	}, nil
}

type lazyOpeningSelfDeletingFile struct {
	directory filesystem.Directory
	name      string
}

func (f *lazyOpeningSelfDeletingFile) open(accmode int) (filesystem.File, error) {
	return f.directory.OpenFile(f.name, os.O_CREATE|accmode, 0600)
}

func (f *lazyOpeningSelfDeletingFile) Close() error {
	return f.directory.Remove(f.name)
}

func (f *lazyOpeningSelfDeletingFile) ReadAt(p []byte, off int64) (int, error) {
	fh, err := f.open(os.O_RDONLY)
	if err != nil {
		return 0, err
	}
	defer fh.Close()
	return fh.ReadAt(p, off)
}

func (f *lazyOpeningSelfDeletingFile) Truncate(size int64) error {
	fh, err := f.open(os.O_WRONLY)
	if err != nil {
		return err
	}
	defer fh.Close()
	return fh.Truncate(size)
}

func (f *lazyOpeningSelfDeletingFile) WriteAt(p []byte, off int64) (int, error) {
	fh, err := f.open(os.O_WRONLY)
	if err != nil {
		return 0, err
	}
	defer fh.Close()
	return fh.WriteAt(p, off)
}
