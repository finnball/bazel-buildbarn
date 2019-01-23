package fuse

import (
	"encoding/binary"

	"github.com/EdSchouten/bazel-buildbarn/pkg/util"
	"github.com/hanwen/go-fuse/fuse"
	"github.com/hanwen/go-fuse/fuse/nodefs"
)

type immutableDirectoryNode struct {
	nodefs.Node

	immutableTree ImmutableTree
	digest        *util.Digest
	deletable     bool
}

// NewImmutableDirectoryNode creates a FUSE directory node that provides
// a read-only view of a directory blob stored in a remote execution
// Content Addressable Storage (CAS).
func NewImmutableDirectoryNode(immutableTree ImmutableTree, digest *util.Digest, deletable bool) nodefs.Node {
	return &immutableDirectoryNode{
		Node:          nodefs.NewDefaultNode(),
		immutableTree: immutableTree,
		digest:        digest,
		deletable:     deletable,
	}
}

func (n *immutableDirectoryNode) Deletable() bool {
	return n.deletable
}

func (n *immutableDirectoryNode) GetAttr(out *fuse.Attr, file nodefs.File, context *fuse.Context) fuse.Status {
	d, err := n.immutableTree.GetDirectory(n.digest)
	if err != nil {
		return fuse.EIO
	}
	*out = fuse.Attr{
		Ino:   binary.BigEndian.Uint64(n.digest.GetHashBytes()),
		Size:  uint64(n.digest.GetSizeBytes()),
		Mode:  fuse.S_IFDIR | 0555,
		Nlink: uint32(len(d.Directories)) + 2,
	}
	return fuse.OK
}

func (n *immutableDirectoryNode) Lookup(out *fuse.Attr, name string, context *fuse.Context) (*nodefs.Inode, fuse.Status) {
	d, err := n.immutableTree.GetDirectory(n.digest)
	if err != nil {
		return nil, fuse.EIO
	}

	for _, fileEntry := range d.Files {
		if name == fileEntry.Name {
			childDigest, err := n.digest.NewDerivedDigest(fileEntry.Digest)
			if err != nil {
				return nil, fuse.EIO
			}
			childNode := NewImmutableFileNode(n.immutableTree, childDigest, fileEntry.IsExecutable, true)
			if s := childNode.GetAttr(out, nil, context); s != fuse.OK {
				return nil, s
			}
			return n.Inode().NewChild(name, false, childNode), fuse.OK
		}
	}
	for _, directoryEntry := range d.Directories {
		if name == directoryEntry.Name {
			childDigest, err := n.digest.NewDerivedDigest(directoryEntry.Digest)
			if err != nil {
				return nil, fuse.EIO
			}
			childNode := NewImmutableDirectoryNode(n.immutableTree, childDigest, true)
			if s := childNode.GetAttr(out, nil, context); s != fuse.OK {
				return nil, s
			}
			return n.Inode().NewChild(name, true, childNode), fuse.OK
		}
	}
	for _, symlinkEntry := range d.Symlinks {
		if name == symlinkEntry.Name {
			childNode := NewImmutableSymlinkNode(symlinkEntry.Target, true)
			if s := childNode.GetAttr(out, nil, context); s != fuse.OK {
				return nil, s
			}
			return n.Inode().NewChild(name, false, childNode), fuse.OK
		}
	}
	return nil, fuse.ENOENT
}

func (n *immutableDirectoryNode) OpenDir(context *fuse.Context) ([]fuse.DirEntry, fuse.Status) {
	d, err := n.immutableTree.GetDirectory(n.digest)
	if err != nil {
		return nil, fuse.EIO
	}

	var entries []fuse.DirEntry
	for _, fileEntry := range d.Files {
		childDigest, err := n.digest.NewDerivedDigest(fileEntry.Digest)
		if err != nil {
			return nil, fuse.EIO
		}
		// TODO(edsch): Remove duplication. Move code for
		// generating fuse.Attr/fuse.DirEntry to some central
		// place.
		var mode uint32 = fuse.S_IFREG | 0444
		if fileEntry.IsExecutable {
			mode = fuse.S_IFREG | 0555
		}
		entries = append(entries, fuse.DirEntry{
			Mode: mode,
			Name: fileEntry.Name,
			Ino:  binary.BigEndian.Uint64(childDigest.GetHashBytes()),
		})
	}
	for _, directoryEntry := range d.Directories {
		entries = append(entries, fuse.DirEntry{
			Mode: fuse.S_IFDIR | 0555,
			Name: directoryEntry.Name,
		})
	}
	for _, symlinkEntry := range d.Symlinks {
		entries = append(entries, fuse.DirEntry{
			Mode: fuse.S_IFLNK | 0777,
			Name: symlinkEntry.Name,
		})
	}
	return entries, fuse.OK
}
