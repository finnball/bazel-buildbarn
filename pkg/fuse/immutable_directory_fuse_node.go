package fuse

import (
	"encoding/binary"
	"sort"

	"github.com/EdSchouten/bazel-buildbarn/pkg/util"
	"github.com/hanwen/go-fuse/fuse"
	"github.com/hanwen/go-fuse/fuse/nodefs"
)

type immutableDirectoryFUSENode struct {
	directoryFUSENode

	immutableTree ImmutableTree
	digest        *util.Digest
}

// NewImmutableDirectoryFUSENode creates a FUSE directory node that provides
// a read-only view of a directory blob stored in a remote execution
// Content Addressable Storage (CAS).
func NewImmutableDirectoryFUSENode(immutableTree ImmutableTree, digest *util.Digest) nodefs.Node {
	return &immutableDirectoryFUSENode{
		immutableTree: immutableTree,
		digest:        digest,
	}
}

func (n *immutableDirectoryFUSENode) Access(mode uint32, context *fuse.Context) fuse.Status {
	if mode &^ (fuse.R_OK|fuse.X_OK) != 0 {
		return fuse.EACCES
	}
	return fuse.OK
}

func (n *immutableDirectoryFUSENode) Chmod(file nodefs.File, perms uint32, context *fuse.Context) fuse.Status {
	return fuse.EPERM
}

func (n *immutableDirectoryFUSENode) Create(name string, flags uint32, mode uint32, context *fuse.Context) (nodefs.File, *nodefs.Inode, fuse.Status) {
	return nil, nil, fuse.EACCES
}

func (n *immutableDirectoryFUSENode) GetAttr(out *fuse.Attr, file nodefs.File, context *fuse.Context) fuse.Status {
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

func (n *immutableDirectoryFUSENode) Link(name string, existing nodefs.Node, context *fuse.Context) (*nodefs.Inode, fuse.Status) {
	return nil, fuse.EACCES
}

func (n *immutableDirectoryFUSENode) Lookup(out *fuse.Attr, name string, context *fuse.Context) (*nodefs.Inode, fuse.Status) {
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
			childNode := NewImmutableFileFUSENode(n.immutableTree, childDigest, fileEntry.IsExecutable)
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
			childNode := NewImmutableDirectoryFUSENode(n.immutableTree, childDigest)
			if s := childNode.GetAttr(out, nil, context); s != fuse.OK {
				return nil, s
			}
			return n.Inode().NewChild(name, true, childNode), fuse.OK
		}
	}
	for _, symlinkEntry := range d.Symlinks {
		if name == symlinkEntry.Name {
			childNode := NewSymlinkFUSENode(symlinkEntry.Target)
			if s := childNode.GetAttr(out, nil, context); s != fuse.OK {
				return nil, s
			}
			return n.Inode().NewChild(name, false, childNode), fuse.OK
		}
	}
	return nil, fuse.ENOENT
}

func (n *immutableDirectoryFUSENode) Mkdir(name string, mode uint32, context *fuse.Context) (*nodefs.Inode, fuse.Status) {
	return nil, fuse.EACCES
}

func (n *immutableDirectoryFUSENode) OpenDir(context *fuse.Context) ([]fuse.DirEntry, fuse.Status) {
	d, err := n.immutableTree.GetDirectory(n.digest)
	if err != nil {
		return nil, fuse.EIO
	}

	var entries dirEntryList
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
			mode |= 0111
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
	sort.Sort(entries)
	return entries, fuse.OK
}

func (n *immutableDirectoryFUSENode) Rename(oldName string, newParent nodefs.Node, newName string, context *fuse.Context) fuse.Status {
	return fuse.EACCES
}

func (n *immutableDirectoryFUSENode) Rmdir(name string, context *fuse.Context) fuse.Status {
	return fuse.EACCES
}

func (n *immutableDirectoryFUSENode) Symlink(name string, content string, context *fuse.Context) (*nodefs.Inode, fuse.Status) {
	return nil, fuse.EACCES
}

func (n *immutableDirectoryFUSENode) Unlink(name string, context *fuse.Context) fuse.Status {
	return fuse.EACCES
}
