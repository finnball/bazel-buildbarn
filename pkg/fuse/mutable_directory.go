package fuse

import (
	"sort"
	"sync"
	"syscall"

	"github.com/EdSchouten/bazel-buildbarn/pkg/util"
	"github.com/hanwen/go-fuse/fuse"
	"github.com/hanwen/go-fuse/fuse/nodefs"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type mutableDirectory struct {
	mutableTree MutableTree

	lock        sync.Mutex
	directories map[string]Directory
	leaves      map[string]Leaf
}

func NewMutableDirectory(mutableTree MutableTree) Directory {
	return &mutableDirectory{
		mutableTree: mutableTree,

		directories: map[string]Directory{},
		leaves:      map[string]Leaf{},
	}
}

func (i *mutableDirectory) GetFUSEDirEntry() fuse.DirEntry {
	return fuse.DirEntry{
		Mode: fuse.S_IFDIR | 0777,
	}
}

func (i *mutableDirectory) GetFUSENode() nodefs.Node {
	return &mutableDirectoryFUSENode{
		i:    i,
	}
}

func (i *mutableDirectory) GetOrCreateDirectory(name string) (Directory, error) {
	i.lock.Lock()
	if child, ok := i.directories[name]; ok {
		i.lock.Unlock()
		return child, nil
	}
	if _, ok := i.leaves[name]; ok {
		i.lock.Unlock()
		return nil, status.Error(codes.AlreadyExists, "A leaf node with this name already exists")
	}
	child := NewMutableDirectory(i.mutableTree)
	i.directories[name] = child
	i.lock.Unlock()
	return child, nil
}

func (i *mutableDirectory) IsEmpty() (bool, error) {
	i.lock.Lock()
	defer i.lock.Unlock()
	return len(i.directories) == 0 && len(i.leaves) == 0, nil
}

func (i *mutableDirectory) MergeImmutableTree(immutableTree ImmutableTree, digest *util.Digest) error {
	d, err := immutableTree.GetDirectory(digest)
	if err != nil {
		return err
	}

	i.lock.Lock()
	defer i.lock.Unlock()

	for _, fileEntry := range d.Files {
		if _, ok := i.directories[fileEntry.Name]; ok {
			return status.Errorf(codes.AlreadyExists, "A directory node with name %#v already exists", fileEntry.Name)
		}
		if _, ok := i.leaves[fileEntry.Name]; ok {
			return status.Errorf(codes.AlreadyExists, "A leaf node with name %#v already exists", fileEntry.Name)
		}
		childDigest, err := digest.NewDerivedDigest(fileEntry.Digest)
		if err != nil {
			return err
		}
		i.leaves[fileEntry.Name] = NewImmutableFile(immutableTree, childDigest, fileEntry.IsExecutable)
	}

	for _, directoryEntry := range d.Directories {
		childDigest, err := digest.NewDerivedDigest(directoryEntry.Digest)
		if err != nil {
			return err
		}
		if child, ok := i.directories[directoryEntry.Name]; ok {
			if err := child.MergeImmutableTree(immutableTree, childDigest); err != nil {
				return err
			}
		} else if _, ok = i.leaves[directoryEntry.Name]; ok {
			return status.Errorf(codes.AlreadyExists, "A leaf node with name %#v already exists", directoryEntry.Name)
		} else {
			i.directories[directoryEntry.Name] = NewImmutableDirectory(immutableTree, childDigest)
		}
	}

	for _, symlinkEntry := range d.Symlinks {
		if _, ok := i.directories[symlinkEntry.Name]; ok {
			return status.Errorf(codes.AlreadyExists, "A directory node with name %#v already exists", symlinkEntry.Name)
		}
		if _, ok := i.leaves[symlinkEntry.Name]; ok {
			return status.Errorf(codes.AlreadyExists, "A leaf node with name %#v already exists", symlinkEntry.Name)
		}
		i.leaves[symlinkEntry.Name] = NewSymlink(symlinkEntry.Target)
	}

	return nil
}

type mutableDirectoryFUSENode struct {
	directoryFUSENode

	i *mutableDirectory
}

func (n *mutableDirectoryFUSENode) Access(mode uint32, context *fuse.Context) fuse.Status {
	if mode &^ (fuse.R_OK|fuse.W_OK|fuse.X_OK) != 0 {
		return fuse.EACCES
	}
	return fuse.OK
}

func (n *mutableDirectoryFUSENode) Chmod(file nodefs.File, perms uint32, context *fuse.Context) fuse.Status {
	return fuse.OK
}

func (n *mutableDirectoryFUSENode) Create(name string, flags uint32, mode uint32, context *fuse.Context) (nodefs.File, *nodefs.Inode, fuse.Status) {
	n.i.lock.Lock()
	if _, ok := n.i.directories[name]; ok {
		n.i.lock.Unlock()
		return nil, nil, fuse.Status(syscall.EEXIST)
	}
	if _, ok := n.i.leaves[name]; ok {
		n.i.lock.Unlock()
		return nil, nil, fuse.Status(syscall.EEXIST)
	}
	file, err := n.i.mutableTree.NewFile()
	if err != nil {
		n.i.lock.Unlock()
		return nil, nil, fuse.EIO
	}
	child := NewMutableFile(file, (mode&0111) != 0)
	n.i.leaves[name] = child
	n.i.lock.Unlock()

	childNode := child.GetFUSENode()
	return NewStatelessFUSEFile(childNode, context), n.Inode().NewChild(name, false, childNode), fuse.OK
}

func (n *mutableDirectoryFUSENode) GetAttr(out *fuse.Attr, file nodefs.File, context *fuse.Context) fuse.Status {
	n.i.lock.Lock()
	nDirectories := len(n.i.directories)
	n.i.lock.Unlock()
	*out = fuse.Attr{
		Mode:  fuse.S_IFDIR | 0777,
		Nlink: uint32(nDirectories) + 2,
	}
	return fuse.OK
}

func (n *mutableDirectoryFUSENode) Link(name string, existing nodefs.Node, context *fuse.Context) (*nodefs.Inode, fuse.Status) {
	// TODO(edsch): Implement!
	return nil, fuse.ENOSYS
}

func (n *mutableDirectoryFUSENode) Lookup(out *fuse.Attr, name string, context *fuse.Context) (*nodefs.Inode, fuse.Status) {
	n.i.lock.Lock()
	if child, ok := n.i.directories[name]; ok {
		childNode := child.GetFUSENode()
		n.i.lock.Unlock()
		if s := childNode.GetAttr(out, nil, context); s != fuse.OK {
			return nil, s
		}
		return n.Inode().NewChild(name, true, childNode), fuse.OK
	}
	if child, ok := n.i.leaves[name]; ok {
		childNode := child.GetFUSENode()
		n.i.lock.Unlock()
		if s := childNode.GetAttr(out, nil, context); s != fuse.OK {
			return nil, s
		}
		return n.Inode().NewChild(name, false, childNode), fuse.OK
	}
	n.i.lock.Unlock()
	return nil, fuse.ENOENT
}

func (n mutableDirectoryFUSENode) Mkdir(name string, mode uint32, context *fuse.Context) (*nodefs.Inode, fuse.Status) {
	n.i.lock.Lock()
	if _, ok := n.i.directories[name]; ok {
		n.i.lock.Unlock()
		return nil, fuse.Status(syscall.EEXIST)
	}
	if _, ok := n.i.leaves[name]; ok {
		n.i.lock.Unlock()
		return nil, fuse.Status(syscall.EEXIST)
	}
	child := NewMutableDirectory(n.i.mutableTree)
	n.i.directories[name] = child
	n.i.lock.Unlock()

	childNode := child.GetFUSENode()
	return n.Inode().NewChild(name, true, childNode), fuse.OK
}

func (n *mutableDirectoryFUSENode) OpenDir(context *fuse.Context) ([]fuse.DirEntry, fuse.Status) {
	n.i.lock.Lock()
	var entries dirEntryList
	for name, child := range n.i.directories {
		directoryEntry := child.GetFUSEDirEntry()
		directoryEntry.Name = name
		entries = append(entries, directoryEntry)
	}
	for name, child := range n.i.leaves {
		directoryEntry := child.GetFUSEDirEntry()
		directoryEntry.Name = name
		entries = append(entries, directoryEntry)
	}
	n.i.lock.Unlock()

	sort.Sort(entries)
	return entries, fuse.OK
}

func (n *mutableDirectoryFUSENode) Rename(oldName string, newParent nodefs.Node, newName string, context *fuse.Context) fuse.Status {
	// TODO(edsch): Implement!
	return fuse.ENOSYS
}

func (n *mutableDirectoryFUSENode) Rmdir(name string, context *fuse.Context) fuse.Status {
	n.i.lock.Lock()
	defer n.i.lock.Unlock()
	if child, ok := n.i.directories[name]; ok {
		if isEmpty, err := child.IsEmpty(); err != nil {
			return fuse.EIO
		} else if !isEmpty {
			return fuse.Status(syscall.ENOTEMPTY)
		}
		delete(n.i.directories, name)
		return fuse.OK
	}
	if _, ok := n.i.leaves[name]; ok {
		return fuse.Status(syscall.ENOTEMPTY)
	}
	return fuse.ENOENT
}

func (n *mutableDirectoryFUSENode) Symlink(name string, content string, context *fuse.Context) (*nodefs.Inode, fuse.Status) {
	n.i.lock.Lock()
	if _, ok := n.i.directories[name]; ok {
		n.i.lock.Unlock()
		return nil, fuse.Status(syscall.EEXIST)
	}
	if _, ok := n.i.leaves[name]; ok {
		n.i.lock.Unlock()
		return nil, fuse.Status(syscall.EEXIST)
	}
	child := NewSymlink(content)
	n.i.leaves[name] = child
	n.i.lock.Unlock()

	childNode := child.GetFUSENode()
	return n.Inode().NewChild(name, false, childNode), fuse.OK
}

func (n *mutableDirectoryFUSENode) Unlink(name string, context *fuse.Context) fuse.Status {
	n.i.lock.Lock()
	defer n.i.lock.Unlock()
	if _, ok := n.i.directories[name]; ok {
		return fuse.EPERM
	}
	if _, ok := n.i.leaves[name]; ok {
		delete(n.i.leaves, name)
		return fuse.OK
	}
	return fuse.ENOENT
}
