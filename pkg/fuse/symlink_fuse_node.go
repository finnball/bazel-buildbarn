package fuse

import (
	"github.com/hanwen/go-fuse/fuse"
	"github.com/hanwen/go-fuse/fuse/nodefs"
)

type symlinkFUSENode struct {
	leafFUSENode

	target string
}

// NewSymlinkFUSENode creates a FUSE symlink node.
func NewSymlinkFUSENode(target string) nodefs.Node {
	return &symlinkFUSENode{
		target: target,
	}
}

func (n *symlinkFUSENode) Access(mode uint32, context *fuse.Context) fuse.Status {
	if mode&^(fuse.R_OK|fuse.W_OK|fuse.X_OK) != 0 {
		return fuse.EACCES
	}
	return fuse.OK
}

func (n *symlinkFUSENode) Chmod(file nodefs.File, perms uint32, context *fuse.Context) fuse.Status {
	return fuse.OK
}

func (n *symlinkFUSENode) Fallocate(file nodefs.File, off uint64, size uint64, mode uint32, context *fuse.Context) fuse.Status {
	return fuse.ENOSYS
}

func (n *symlinkFUSENode) GetAttr(out *fuse.Attr, file nodefs.File, context *fuse.Context) fuse.Status {
	*out = fuse.Attr{
		Size: uint64(len(n.target)),
		Mode: fuse.S_IFLNK | 0777,
	}
	return fuse.OK
}

func (n *symlinkFUSENode) Open(flags uint32, context *fuse.Context) (nodefs.File, fuse.Status) {
	return nil, fuse.ENOSYS
}

func (n *symlinkFUSENode) Read(file nodefs.File, dest []byte, off int64, context *fuse.Context) (fuse.ReadResult, fuse.Status) {
	return nil, fuse.ENOSYS
}

func (n *symlinkFUSENode) Readlink(c *fuse.Context) ([]byte, fuse.Status) {
	return []byte(n.target), fuse.OK
}

func (n *symlinkFUSENode) Truncate(file nodefs.File, size uint64, context *fuse.Context) fuse.Status {
	return fuse.ENOSYS
}

func (n *symlinkFUSENode) Write(file nodefs.File, data []byte, off int64, context *fuse.Context) (uint32, fuse.Status) {
	return 0, fuse.ENOSYS
}
