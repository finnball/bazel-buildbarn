package fuse

import (
	"encoding/binary"
	"io"

	"github.com/EdSchouten/bazel-buildbarn/pkg/util"
	"github.com/hanwen/go-fuse/fuse"
	"github.com/hanwen/go-fuse/fuse/nodefs"
)

type immutableFileFUSENode struct {
	nodefs.Node

	immutableTree ImmutableTree
	digest        *util.Digest
	isExecutable  bool
}

// NewImmutableFileFUSENode creates a FUSE file node that provides a
// read-only view of a file blob stored in a remote execution Content
// Addressable Storage (CAS).
func NewImmutableFileFUSENode(immutableTree ImmutableTree, digest *util.Digest, isExecutable bool) nodefs.Node {
	return &immutableFileFUSENode{
		Node:          nodefs.NewDefaultNode(),
		immutableTree: immutableTree,
		digest:        digest,
		isExecutable:  isExecutable,
	}
}

func (n *immutableFileFUSENode) GetAttr(out *fuse.Attr, file nodefs.File, context *fuse.Context) fuse.Status {
	var mode uint32 = fuse.S_IFREG | 0444
	if n.isExecutable {
		mode = fuse.S_IFREG | 0555
	}
	*out = fuse.Attr{
		Ino:  binary.BigEndian.Uint64(n.digest.GetHashBytes()),
		Size: uint64(n.digest.GetSizeBytes()),
		Mode: mode,
	}
	return fuse.OK
}

func (n *immutableFileFUSENode) Open(flags uint32, context *fuse.Context) (file nodefs.File, code fuse.Status) {
	return nil, fuse.OK
}

func (n *immutableFileFUSENode) Read(file nodefs.File, dest []byte, off int64, context *fuse.Context) (fuse.ReadResult, fuse.Status) {
	nRead, err := n.immutableTree.ReadFileAt(n.digest, dest, off)
	if err != nil && err != io.EOF {
		return nil, fuse.EIO
	}
	return fuse.ReadResultData(dest[:nRead]), fuse.OK
}
