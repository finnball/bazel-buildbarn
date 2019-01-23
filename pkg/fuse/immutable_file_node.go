package fuse

import (
	"encoding/binary"
	"io"

	"github.com/EdSchouten/bazel-buildbarn/pkg/util"
	"github.com/hanwen/go-fuse/fuse"
	"github.com/hanwen/go-fuse/fuse/nodefs"
)

type immutableFileNode struct {
	nodefs.Node

	immutableTree ImmutableTree
	digest        *util.Digest
	isExecutable  bool
	deletable     bool
}

// NewImmutableFileNode creates a FUSE file node that provides a
// read-only view of a file blob stored in a remote execution Content
// Addressable Storage (CAS).
func NewImmutableFileNode(immutableTree ImmutableTree, digest *util.Digest, isExecutable bool, deletable bool) nodefs.Node {
	return &immutableFileNode{
		Node:          nodefs.NewDefaultNode(),
		immutableTree: immutableTree,
		digest:        digest,
		isExecutable:  isExecutable,
		deletable:     deletable,
	}
}

func (n *immutableFileNode) Deletable() bool {
	return n.deletable
}

func (n *immutableFileNode) GetAttr(out *fuse.Attr, file nodefs.File, context *fuse.Context) fuse.Status {
	var mode uint32 = fuse.S_IFREG | 0444
	if n.isExecutable {
		mode = fuse.S_IFREG | 0555
	}
	*out = fuse.Attr{
		Size: uint64(n.digest.GetSizeBytes()),
		Mode: mode,
		Ino:  binary.BigEndian.Uint64(n.digest.GetHashBytes()),
	}
	return fuse.OK
}

func (n *immutableFileNode) Open(flags uint32, context *fuse.Context) (file nodefs.File, code fuse.Status) {
	return nil, fuse.OK
}

func (n *immutableFileNode) Read(file nodefs.File, dest []byte, off int64, context *fuse.Context) (fuse.ReadResult, fuse.Status) {
	nRead, err := n.immutableTree.ReadFileAt(n.digest, dest, off)
	if err != nil && err != io.EOF {
		return nil, fuse.EIO
	}
	return fuse.ReadResultData(dest[:nRead]), fuse.OK
}
