package fuse

import (
	"github.com/hanwen/go-fuse/fuse"
	"github.com/hanwen/go-fuse/fuse/nodefs"
)

type FUSENode interface {
	nodefs.Node

	LinkNode() (Leaf, fuse.Status)
}
