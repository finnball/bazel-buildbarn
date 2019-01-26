package fuse

import (
	"github.com/hanwen/go-fuse/fuse"
	"github.com/hanwen/go-fuse/fuse/nodefs"
)

type directoryFUSENode struct {
	defaultFUSENode
}

func (n *directoryFUSENode) Fallocate(file nodefs.File, off uint64, size uint64, mode uint32, context *fuse.Context) fuse.Status {
	return fuse.ENOSYS
}

func (n *directoryFUSENode) Open(flags uint32, context *fuse.Context) (nodefs.File, fuse.Status) {
	return nil, fuse.EISDIR
}

func (n *directoryFUSENode) Read(file nodefs.File, dest []byte, off int64, context *fuse.Context) (fuse.ReadResult, fuse.Status) {
	return nil, fuse.EISDIR
}

func (n *directoryFUSENode) Readlink(c *fuse.Context) ([]byte, fuse.Status) {
	return nil, fuse.EINVAL
}

func (n *directoryFUSENode) Truncate(file nodefs.File, size uint64, context *fuse.Context) fuse.Status {
	return fuse.EBADF
}

func (n *directoryFUSENode) Write(file nodefs.File, data []byte, off int64, context *fuse.Context) (uint32, fuse.Status) {
	return 0, fuse.EISDIR
}
