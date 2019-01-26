package fuse

import (
	"time"

	"github.com/hanwen/go-fuse/fuse"
	"github.com/hanwen/go-fuse/fuse/nodefs"
)

type statelessFUSEFile struct {
	node    nodefs.Node
	context *fuse.Context
}

// NewStatelessFUSEFile creates a FUSE file handle that forwards all
// calls to the underlying FUSE node.
//
// Node.Open() is permitted to return a nil File object, causing all
// file operations to be applied directly to the Node. For some odd
// reason, the same does not apply to Node.Create(). This function must
// return a valid File object. Details:
//
//     https://github.com/hanwen/go-fuse/issues/255
func NewStatelessFUSEFile(node nodefs.Node, context *fuse.Context) nodefs.File {
	return &statelessFUSEFile{
		node:    node,
		context: context,
	}
}

func (f *statelessFUSEFile) Allocate(off uint64, size uint64, mode uint32) fuse.Status {
	return f.node.Fallocate(nil, off, size, mode, f.context)
}

func (f *statelessFUSEFile) Chown(uid uint32, gid uint32) fuse.Status {
	return f.node.Chown(nil, uid, gid, f.context)
}

func (f *statelessFUSEFile) Chmod(perms uint32) fuse.Status {
	return f.node.Chmod(nil, perms, f.context)
}

func (f *statelessFUSEFile) Flush() fuse.Status {
	return fuse.OK
}

func (f *statelessFUSEFile) Fsync(flags int) fuse.Status {
	return fuse.ENOSYS
}

func (f *statelessFUSEFile) GetAttr(out *fuse.Attr) fuse.Status {
	return f.node.GetAttr(out, nil, f.context)
}

func (f *statelessFUSEFile) GetLk(owner uint64, lk *fuse.FileLock, flags uint32, out *fuse.FileLock) fuse.Status {
	return f.node.GetLk(nil, owner, lk, flags, out, f.context)
}

func (f *statelessFUSEFile) InnerFile() nodefs.File {
	return nil
}

func (f *statelessFUSEFile) Read(dest []byte, off int64) (fuse.ReadResult, fuse.Status) {
	return f.node.Read(nil, dest, off, f.context)
}

func (f *statelessFUSEFile) Release() {
}

func (f *statelessFUSEFile) SetInode(inode *nodefs.Inode) {
}

func (f *statelessFUSEFile) SetLk(owner uint64, lk *fuse.FileLock, flags uint32) fuse.Status {
	return f.node.SetLk(nil, owner, lk, flags, f.context)
}

func (f *statelessFUSEFile) SetLkw(owner uint64, lk *fuse.FileLock, flags uint32) fuse.Status {
	return f.node.SetLkw(nil, owner, lk, flags, f.context)
}

func (f *statelessFUSEFile) String() string {
	return "StatelessFUSEFile"
}

func (f *statelessFUSEFile) Truncate(size uint64) fuse.Status {
	return f.node.Truncate(nil, size, f.context)
}

func (f *statelessFUSEFile) Utimens(atime *time.Time, mtime *time.Time) fuse.Status {
	return f.node.Utimens(nil, atime, mtime, f.context)
}

func (f *statelessFUSEFile) Write(data []byte, off int64) (uint32, fuse.Status) {
	return f.node.Write(nil, data, off, f.context)
}
