package fuse

import (
	"github.com/EdSchouten/bazel-buildbarn/pkg/util"
	"github.com/hanwen/go-fuse/fuse"
	"github.com/hanwen/go-fuse/fuse/nodefs"
)

type Directory interface {
	GetFUSEDirEntry() fuse.DirEntry
	GetFUSENode() nodefs.Node
	GetOrCreateDirectory(name string) (Directory, error)
	MergeImmutableTree(immutableTree ImmutableTree, digest *util.Digest) error
	IsEmpty() (bool, error)
}
