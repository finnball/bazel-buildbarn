package fuse

import (
	"github.com/hanwen/go-fuse/fuse"
)

type Leaf interface {
	GetFUSEDirEntry() fuse.DirEntry
	GetFUSENode() FUSENode

	Unlink()
}
