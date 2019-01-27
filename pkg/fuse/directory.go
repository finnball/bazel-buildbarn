package fuse

import (
	"github.com/EdSchouten/bazel-buildbarn/pkg/util"
	"github.com/hanwen/go-fuse/fuse"
)

type Directory interface {
	GetFUSEDirEntry() fuse.DirEntry
	GetFUSENode() FUSENode
	GetOrCreateDirectory(name string) (Directory, error)
	MergeImmutableTree(immutableTree ImmutableTree, digest *util.Digest) error
	IsEmpty() (bool, error)
}
