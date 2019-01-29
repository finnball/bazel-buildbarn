package fuse

import (
	"encoding/binary"
	"io"

	"github.com/EdSchouten/bazel-buildbarn/pkg/util"
	"github.com/hanwen/go-fuse/fuse"
	"github.com/hanwen/go-fuse/fuse/nodefs"
)

type immutableFileFUSENode struct {
	leafFUSENode

	immutableTree ImmutableTree
	digest        *util.Digest
	isExecutable  bool
}

// NewImmutableFileFUSENode creates a FUSE file node that provides a
// read-only view of a file blob stored in a remote execution Content
// Addressable Storage (CAS).
func NewImmutableFileFUSENode(immutableTree ImmutableTree, digest *util.Digest, isExecutable bool) FUSENode {
	return &immutableFileFUSENode{
		immutableTree: immutableTree,
		digest:        digest,
		isExecutable:  isExecutable,
	}
}

func (n *immutableFileFUSENode) Access(mode uint32, context *fuse.Context) fuse.Status {
	var permitted uint32 = fuse.R_OK
	if n.isExecutable {
		permitted |= fuse.X_OK
	}
	if mode&^permitted != 0 {
		return fuse.EACCES
	}
	return fuse.OK
}

func (n *immutableFileFUSENode) Chmod(file nodefs.File, perms uint32, context *fuse.Context) fuse.Status {
	return fuse.EPERM
}

func (n *immutableFileFUSENode) Fallocate(file nodefs.File, off uint64, size uint64, mode uint32, context *fuse.Context) fuse.Status {
	return fuse.EBADF
}

func (n *immutableFileFUSENode) GetAttr(out *fuse.Attr, file nodefs.File, context *fuse.Context) fuse.Status {
	var mode uint32 = fuse.S_IFREG | 0444
	if n.isExecutable {
		mode |= 0111
	}
	*out = fuse.Attr{
		Ino:     binary.BigEndian.Uint64(n.digest.GetHashBytes()),
		Size:    uint64(n.digest.GetSizeBytes()),
		Blocks:  toBlockSize(uint64(n.digest.GetSizeBytes())),
		Mode:    mode,
		Blksize: defaultBlockSize,
	}
	return fuse.OK
}

func (n *immutableFileFUSENode) LinkNode() (Leaf, fuse.Status) {
	// TODO(edsch): This is copying, instead of linking. Is this valid?
	return NewImmutableFile(n.immutableTree, n.digest, n.isExecutable), fuse.OK
}

func (n *immutableFileFUSENode) Open(flags uint32, context *fuse.Context) (file nodefs.File, code fuse.Status) {
	if flags&fuse.O_ANYWRITE != 0 {
		return nil, fuse.EACCES
	}
	return nil, fuse.OK
}

func (n *immutableFileFUSENode) Read(file nodefs.File, dest []byte, off int64, context *fuse.Context) (fuse.ReadResult, fuse.Status) {
	nRead, err := n.immutableTree.ReadFileAt(n.digest, dest, off)
	if err != nil && err != io.EOF {
		return nil, fuse.EIO
	}
	return fuse.ReadResultData(dest[:nRead]), fuse.OK
}

func (n *immutableFileFUSENode) Readlink(c *fuse.Context) ([]byte, fuse.Status) {
	return nil, fuse.EINVAL
}

func (n *immutableFileFUSENode) Truncate(file nodefs.File, size uint64, context *fuse.Context) fuse.Status {
	return fuse.EBADF
}

func (n *immutableFileFUSENode) Write(file nodefs.File, data []byte, off int64, context *fuse.Context) (uint32, fuse.Status) {
	return 0, fuse.EBADF
}
