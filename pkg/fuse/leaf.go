package fuse

import (
	"github.com/hanwen/go-fuse/fuse"
	"github.com/hanwen/go-fuse/fuse/nodefs"
)

type Leaf interface {
	GetFUSEDirEntry() fuse.DirEntry
	GetFUSENode() nodefs.Node
}
