package fuse

import (
	"github.com/EdSchouten/bazel-buildbarn/pkg/util"
	"github.com/hanwen/go-fuse/fuse"
	"github.com/hanwen/go-fuse/fuse/nodefs"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type immutableDirectory struct {
	immutableTree ImmutableTree
	digest        *util.Digest
}

func NewImmutableDirectory(immutableTree ImmutableTree, digest *util.Digest) Directory {
	return &immutableDirectory{
		immutableTree: immutableTree,
		digest:        digest,
	}
}

func (d *immutableDirectory) GetFUSEDirEntry() fuse.DirEntry {
	return fuse.DirEntry{
		Mode: fuse.S_IFDIR | 0555,
	}
}

func (d *immutableDirectory) GetFUSENode() nodefs.Node {
	return NewImmutableDirectoryFUSENode(d.immutableTree, d.digest)
}

func (d *immutableDirectory) GetOrCreateDirectory(name string) (Directory, error) {
	return nil, status.Error(codes.InvalidArgument, "Cannot write directories inside an immutable directory")
}

func (d *immutableDirectory) MergeImmutableTree(immutableTree ImmutableTree, digest *util.Digest) error {
	return status.Error(codes.InvalidArgument, "Cannot merge tree inside an immutable directory")
}

func (d *immutableDirectory) IsEmpty() (bool, error) {
	dir, err := d.immutableTree.GetDirectory(d.digest)
	if err != nil {
		return false, err
	}
	return len(dir.Directories) == 0, nil
}
