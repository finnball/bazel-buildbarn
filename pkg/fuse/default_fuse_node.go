package fuse

import (
	"time"

	"github.com/hanwen/go-fuse/fuse"
	"github.com/hanwen/go-fuse/fuse/nodefs"
)

type defaultFUSENode struct {
	inode *nodefs.Inode
}

func (n *defaultFUSENode) Chown(file nodefs.File, uid uint32, gid uint32, context *fuse.Context) fuse.Status {
	return fuse.EPERM
}

func (n *defaultFUSENode) Deletable() bool {
	return true
}

func (n *defaultFUSENode) GetLk(file nodefs.File, owner uint64, lk *fuse.FileLock, flags uint32, out *fuse.FileLock, context *fuse.Context) fuse.Status {
	return fuse.ENOSYS
}

func (n *defaultFUSENode) GetXAttr(attribute string, context *fuse.Context) ([]byte, fuse.Status) {
	return nil, fuse.ENOATTR
}

func (n *defaultFUSENode) Inode() *nodefs.Inode {
	return n.inode
}

func (n *defaultFUSENode) ListXAttr(context *fuse.Context) (attrs []string, code fuse.Status) {
	return nil, fuse.ENOSYS
}

func (n *defaultFUSENode) Mknod(name string, mode uint32, dev uint32, context *fuse.Context) (*nodefs.Inode, fuse.Status) {
	return nil, fuse.ENOSYS
}

func (n *defaultFUSENode) OnForget() {
}

func (n defaultFUSENode) OnMount(conn *nodefs.FileSystemConnector) {
}

func (n *defaultFUSENode) OnUnmount() {
}

func (n *defaultFUSENode) RemoveXAttr(attr string, context *fuse.Context) fuse.Status {
	return fuse.ENOSYS
}

func (n *defaultFUSENode) SetInode(node *nodefs.Inode) {
	n.inode = node
}

func (n *defaultFUSENode) SetLk(file nodefs.File, owner uint64, lk *fuse.FileLock, flags uint32, context *fuse.Context) fuse.Status {
	return fuse.ENOSYS
}

func (n *defaultFUSENode) SetLkw(file nodefs.File, owner uint64, lk *fuse.FileLock, flags uint32, context *fuse.Context) fuse.Status {
	return fuse.ENOSYS
}

func (n *defaultFUSENode) SetXAttr(attr string, data []byte, flags int, context *fuse.Context) fuse.Status {
	return fuse.ENOSYS
}

func (n *defaultFUSENode) StatFs() *fuse.StatfsOut {
	return nil
}

func (n *defaultFUSENode) Utimens(file nodefs.File, atime *time.Time, mtime *time.Time, context *fuse.Context) fuse.Status {
	return fuse.OK
}
