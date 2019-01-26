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

func (mt *directoryBackedMutableTree) NewFile() (filesystem.File, error) {
	mt.lock.Lock()
	id := mt.nextID
	mt.nextID++
	mt.lock.Unlock()
	return mt.directory.OpenFile(
		strconv.FormatUint(id, 10),
		os.O_CREATE|os.O_RDWR|os.O_TRUNC,
		0666)
}
