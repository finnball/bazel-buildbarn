package fuse

import (
	"github.com/hanwen/go-fuse/fuse"
	"github.com/hanwen/go-fuse/fuse/nodefs"
)

type symlinkNode struct {
	nodefs.Node
	target string
}

// NewSymlinkNode creates a FUSE symlink node.
func NewSymlinkNode(target string) nodefs.Node {
	return &symlinkNode{
		Node:   nodefs.NewDefaultNode(),
		target: target,
	}
}

func (n *symlinkNode) GetAttr(out *fuse.Attr, file nodefs.File, context *fuse.Context) fuse.Status {
	*out = fuse.Attr{
		Size: uint64(len(n.target)),
		Mode: fuse.S_IFLNK | 0777,
	}
	return fuse.OK
}

func (n *symlinkNode) Readlink(c *fuse.Context) ([]byte, fuse.Status) {
	return []byte(n.target), fuse.OK
}
