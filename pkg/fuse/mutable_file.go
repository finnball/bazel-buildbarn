package fuse

import (
	"io"
	"sync"

	"github.com/EdSchouten/bazel-buildbarn/pkg/filesystem"
	"github.com/hanwen/go-fuse/fuse"
	"github.com/hanwen/go-fuse/fuse/nodefs"
)

type mutableFile struct {
	file filesystem.File

	lock         sync.Mutex
	isExecutable bool
	size         uint64
}

func NewMutableFile(file filesystem.File, isExecutable bool) Leaf {
	return &mutableFile{
		file:         file,
		isExecutable: isExecutable,
	}
}

func (i *mutableFile) GetFUSEDirEntry() fuse.DirEntry {
	var mode uint32 = fuse.S_IFREG | 0666
	if i.isExecutable {
		mode = fuse.S_IFREG | 0777
	}
	return fuse.DirEntry{
		Mode: mode,
	}
}

func (i *mutableFile) GetFUSENode() nodefs.Node {
	return &mutableFileFUSENode{
		Node: nodefs.NewDefaultNode(),
		i:    i,
	}
}

type mutableFileFUSENode struct {
	nodefs.Node

	i *mutableFile
}

func (n *mutableFileFUSENode) Chmod(file nodefs.File, perms uint32, context *fuse.Context) fuse.Status {
	n.i.lock.Lock()
	n.i.isExecutable = (perms & 0111) != 0
	n.i.lock.Unlock()
	return fuse.OK
}

func (n *mutableFileFUSENode) GetAttr(out *fuse.Attr, file nodefs.File, context *fuse.Context) fuse.Status {
	n.i.lock.Lock()
	defer n.i.lock.Unlock()

	var mode uint32 = fuse.S_IFREG | 0666
	if n.i.isExecutable {
		mode = fuse.S_IFREG | 0777
	}
	*out = fuse.Attr{
		Size:  n.i.size,
		Mode:  mode,
		Nlink: 123, // TODO(edsch): Set this.
	}
	return fuse.OK
}

func (n *mutableFileFUSENode) Open(flags uint32, context *fuse.Context) (file nodefs.File, code fuse.Status) {
	return nil, fuse.OK
}

func (n *mutableFileFUSENode) Read(file nodefs.File, dest []byte, off int64, context *fuse.Context) (fuse.ReadResult, fuse.Status) {
	nRead, err := n.i.file.ReadAt(dest, off)
	if err != nil && err != io.EOF {
		return nil, fuse.EIO
	}
	return fuse.ReadResultData(dest[:nRead]), fuse.OK
}

func (n *mutableFileFUSENode) Truncate(file nodefs.File, size uint64, context *fuse.Context) fuse.Status {
	n.i.lock.Lock()
	defer n.i.lock.Unlock()

	if err := n.i.file.Truncate(int64(size)); err != nil {
		return fuse.EIO
	}
	n.i.size = size
	return fuse.OK
}

func (n *mutableFileFUSENode) Write(file nodefs.File, data []byte, off int64, context *fuse.Context) (uint32, fuse.Status) {
	n.i.lock.Lock()
	defer n.i.lock.Unlock()

	nWritten, err := n.i.file.WriteAt(data, off)
	if end := uint64(off) + uint64(nWritten); n.i.size < end {
		n.i.size = end
	}
	if err != nil {
		return uint32(nWritten), fuse.EIO
	}
	return uint32(nWritten), fuse.OK
}
