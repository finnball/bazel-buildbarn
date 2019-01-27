package fuse

import (
	"encoding/binary"

	"github.com/EdSchouten/bazel-buildbarn/pkg/util"
	"github.com/hanwen/go-fuse/fuse"
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
		mode |= 0111
	}
	return fuse.DirEntry{
		Mode: mode,
		Ino:  binary.BigEndian.Uint64(l.digest.GetHashBytes()),
	}
}

func (l *immutableFile) GetFUSENode() FUSENode {
	return NewImmutableFileFUSENode(l.immutableTree, l.digest, l.isExecutable)
}

func (l *immutableFile) Unlink() {
}
