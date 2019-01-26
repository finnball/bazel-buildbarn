package fuse

import (
	"github.com/hanwen/go-fuse/fuse"
	"github.com/hanwen/go-fuse/fuse/nodefs"
)

type leafFUSENode struct {
	defaultFUSENode
}

func (n *leafFUSENode) Create(name string, flags uint32, mode uint32, context *fuse.Context) (nodefs.File, *nodefs.Inode, fuse.Status) {
	return nil, nil, fuse.ENOSYS
}

func (n *leafFUSENode) Link(name string, existing nodefs.Node, context *fuse.Context) (*nodefs.Inode, fuse.Status) {
	return nil, fuse.ENOSYS
}

func (n *leafFUSENode) Lookup(out *fuse.Attr, name string, context *fuse.Context) (*nodefs.Inode, fuse.Status) {
	return nil, fuse.ENOSYS
}

func (n *leafFUSENode) Mkdir(name string, mode uint32, context *fuse.Context) (*nodefs.Inode, fuse.Status) {
	return nil, fuse.ENOSYS
}

func (n *leafFUSENode) OpenDir(context *fuse.Context) ([]fuse.DirEntry, fuse.Status) {
	return nil, fuse.ENOTDIR
}

func (n *leafFUSENode) Rename(oldName string, newParent nodefs.Node, newName string, context *fuse.Context) fuse.Status {
	return fuse.ENOSYS
}

func (n *leafFUSENode) Rmdir(name string, context *fuse.Context) fuse.Status {
	return fuse.ENOSYS
}

func (n *leafFUSENode) Symlink(name string, content string, context *fuse.Context) (*nodefs.Inode, fuse.Status) {
	return nil, fuse.ENOSYS
}

func (n *leafFUSENode) Unlink(name string, context *fuse.Context) fuse.Status {
	return fuse.ENOSYS
}
