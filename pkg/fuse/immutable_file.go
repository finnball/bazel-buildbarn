package fuse

import (
	"encoding/binary"

	"github.com/EdSchouten/bazel-buildbarn/pkg/util"
	"github.com/hanwen/go-fuse/fuse"
	"github.com/hanwen/go-fuse/fuse/nodefs"
)

type immutableFile struct {
	immutableTree ImmutableTree
	digest        *util.Digest
	isExecutable  bool
}

func NewImmutableFile(immutableTree ImmutableTree, digest *util.Digest, isExecutable bool) Leaf {
	return &immutableFile{
		immutableTree: immutableTree,
		digest:        digest,
		isExecutable:  isExecutable,
	}
}

func (l *immutableFile) GetFUSEDirEntry() fuse.DirEntry {
	var mode uint32 = fuse.S_IFREG | 0444
	if l.isExecutable {
		mode = fuse.S_IFREG | 0555
	}
	return fuse.DirEntry{
		Mode: mode,
		Ino:  binary.BigEndian.Uint64(l.digest.GetHashBytes()),
	}
}

func (l *immutableFile) GetFUSENode() nodefs.Node {
	return NewImmutableFileFUSENode(l.immutableTree, l.digest, l.isExecutable)
}
