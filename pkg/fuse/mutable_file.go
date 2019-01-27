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
	nlink        uint32
}

func NewMutableFile(file filesystem.File, isExecutable bool) Leaf {
	return &mutableFile{
		file:         file,
		isExecutable: isExecutable,
		nlink:        1,
	}
}

func (i *mutableFile) GetFUSEDirEntry() fuse.DirEntry {
	var mode uint32 = fuse.S_IFREG | 0666
	if i.isExecutable {
		mode |= 0111
	}
	return fuse.DirEntry{
		Mode: mode,
	}
}

func (i *mutableFile) GetFUSENode() FUSENode {
	return &mutableFileFUSENode{
		i: i,
	}
}

func (i *mutableFile) Unlink() {
	i.lock.Lock()
	defer i.lock.Unlock()
	i.nlink--
	// TODO(edsch): Open file descriptors should cause this to be delayed.
	if i.nlink == 0 {
		i.file.Close()
	}
}

type mutableFileFUSENode struct {
	leafFUSENode

	i *mutableFile
}

func (n *mutableFileFUSENode) Access(mode uint32, context *fuse.Context) fuse.Status {
	var permitted uint32 = fuse.R_OK | fuse.W_OK
	n.i.lock.Lock()
	if n.i.isExecutable {
		permitted |= fuse.X_OK
	}
	n.i.lock.Unlock()
	if mode&^permitted != 0 {
		return fuse.EACCES
	}
	return fuse.OK
}

func (n *mutableFileFUSENode) Chmod(file nodefs.File, perms uint32, context *fuse.Context) fuse.Status {
	n.i.lock.Lock()
	n.i.isExecutable = (perms & 0111) != 0
	n.i.lock.Unlock()
	return fuse.OK
}

func (n *mutableFileFUSENode) Fallocate(file nodefs.File, off uint64, size uint64, mode uint32, context *fuse.Context) fuse.Status {
	n.i.lock.Lock()
	defer n.i.lock.Unlock()

	if end := uint64(off) + uint64(size); n.i.size < end {
		if err := n.i.file.Truncate(int64(end)); err != nil {
			return fuse.EIO
		}
		n.i.size = end
	}
	return fuse.OK
}

func (n *mutableFileFUSENode) GetAttr(out *fuse.Attr, file nodefs.File, context *fuse.Context) fuse.Status {
	n.i.lock.Lock()
	defer n.i.lock.Unlock()

	var mode uint32 = fuse.S_IFREG | 0666
	if n.i.isExecutable {
		mode |= 0111
	}
	*out = fuse.Attr{
		Size:  n.i.size,
		Mode:  mode,
		Nlink: n.i.nlink,
	}
	return fuse.OK
}

func (n *mutableFileFUSENode) LinkNode() (Leaf, fuse.Status) {
	n.i.lock.Lock()
	defer n.i.lock.Unlock()

	n.i.nlink++
	return n.i, fuse.OK
}

func (n *mutableFileFUSENode) Open(flags uint32, context *fuse.Context) (file nodefs.File, code fuse.Status) {
	return nil, fuse.OK
}

func (n *mutableFileFUSENode) Read(file nodefs.File, dest []byte, off int64, context *fuse.Context) (fuse.ReadResult, fuse.Status) {
	n.i.lock.Lock()
	defer n.i.lock.Unlock()

	nRead, err := n.i.file.ReadAt(dest, off)
	if err != nil && err != io.EOF {
		return nil, fuse.EIO
	}
	return fuse.ReadResultData(dest[:nRead]), fuse.OK
}

func (n *mutableFileFUSENode) Readlink(c *fuse.Context) ([]byte, fuse.Status) {
	return nil, fuse.EINVAL
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
