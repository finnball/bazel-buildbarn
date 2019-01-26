package fuse

import (
	"github.com/hanwen/go-fuse/fuse"
	"github.com/hanwen/go-fuse/fuse/nodefs"
)

type symlinkFUSENode struct {
	nodefs.Node
	target string
}

// NewSymlinkFUSENode creates a FUSE symlink node.
func NewSymlinkFUSENode(target string) nodefs.Node {
	return &symlinkFUSENode{
		Node:   nodefs.NewDefaultNode(),
		target: target,
	}
}

func (n *symlinkFUSENode) GetAttr(out *fuse.Attr, file nodefs.File, context *fuse.Context) fuse.Status {
	*out = fuse.Attr{
		Size: uint64(len(n.target)),
		Mode: fuse.S_IFLNK | 0777,
	}
	return fuse.OK
}

func (n *symlinkFUSENode) Readlink(c *fuse.Context) ([]byte, fuse.Status) {
	return []byte(n.target), fuse.OK
}
