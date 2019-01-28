package fuse

import (
	"log"
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

func (i *mutableDirectory) attachDirectory(name string, child Directory) {
	if i.existsDirectory(name) || i.existsLeaf(name) {
		log.Fatalf("Attempted to overwrite node %#v", name)
	}
	i.directories[name] = child
}

func (i *mutableDirectory) attachLeaf(name string, child Leaf) {
	if i.existsDirectory(name) || i.existsLeaf(name) {
		log.Fatalf("Attempted to overwrite node %#v", name)
	}
	i.leaves[name] = child
}

func (i *mutableDirectory) existsDirectory(name string) bool {
	_, ok := i.directories[name]
	return ok
}

func (i *mutableDirectory) existsLeaf(name string) bool {
	_, ok := i.leaves[name]
	return ok
}

func (i *mutableDirectory) GetFUSEDirEntry() fuse.DirEntry {
	return fuse.DirEntry{
		Mode: fuse.S_IFDIR | 0777,
	}
}

func (i *mutableDirectory) GetFUSENode() FUSENode {
	return &mutableDirectoryFUSENode{
		i: i,
	}
}

func (i *mutableDirectory) GetOrCreateDirectory(name string) (Directory, error) {
	i.lock.Lock()
	defer i.lock.Unlock()

	if child, ok := i.directories[name]; ok {
		return child, nil
	}
	if i.existsLeaf(name) {
		return nil, status.Error(codes.AlreadyExists, "A leaf node with this name already exists")
	}
	child := NewMutableDirectory(i.mutableTree)
	i.attachDirectory(name, child)
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
		if i.existsDirectory(fileEntry.Name) || i.existsLeaf(fileEntry.Name) {
			return status.Errorf(codes.AlreadyExists, "A node with name %#v already exists", fileEntry.Name)
		}
		childDigest, err := digest.NewDerivedDigest(fileEntry.Digest)
		if err != nil {
			return err
		}
		i.attachLeaf(fileEntry.Name, NewImmutableFile(immutableTree, childDigest, fileEntry.IsExecutable))
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
		} else if i.existsLeaf(directoryEntry.Name) {
			return status.Errorf(codes.AlreadyExists, "A node with name %#v already exists", directoryEntry.Name)
		} else {
			i.attachDirectory(directoryEntry.Name, NewImmutableDirectory(immutableTree, childDigest))
		}
	}

	for _, symlinkEntry := range d.Symlinks {
		if i.existsDirectory(symlinkEntry.Name) || i.existsLeaf(symlinkEntry.Name) {
			return status.Errorf(codes.AlreadyExists, "A node with name %#v already exists", symlinkEntry.Name)
		}
		i.attachLeaf(symlinkEntry.Name, NewSymlink(symlinkEntry.Target))
	}

	return nil
}

type mutableDirectoryFUSENode struct {
	directoryFUSENode

	i *mutableDirectory
}

func (n *mutableDirectoryFUSENode) detachDirectory(name string) (Directory, bool) {
	child, ok := n.i.directories[name]
	if ok {
		n.Inode().RmChild(name)
		delete(n.i.directories, name)
	}
	return child, ok
}

func (n *mutableDirectoryFUSENode) detachLeaf(name string) (Leaf, bool) {
	child, ok := n.i.leaves[name]
	if ok {
		n.Inode().RmChild(name)
		delete(n.i.leaves, name)
	}
	return child, ok
}

func (n *mutableDirectoryFUSENode) Access(mode uint32, context *fuse.Context) fuse.Status {
	if mode&^(fuse.R_OK|fuse.W_OK|fuse.X_OK) != 0 {
		return fuse.EACCES
	}
	return fuse.OK
}

func (n *mutableDirectoryFUSENode) Chmod(file nodefs.File, perms uint32, context *fuse.Context) fuse.Status {
	return fuse.OK
}

func (n *mutableDirectoryFUSENode) Create(name string, flags uint32, mode uint32, context *fuse.Context) (nodefs.File, *nodefs.Inode, fuse.Status) {
	n.i.lock.Lock()
	if n.i.existsDirectory(name) || n.i.existsLeaf(name) {
		n.i.lock.Unlock()
		return nil, nil, fuse.Status(syscall.EEXIST)
	}
	file, err := n.i.mutableTree.NewFile()
	if err != nil {
		n.i.lock.Unlock()
		return nil, nil, fuse.EIO
	}
	child := NewMutableFile(file, (mode&0111) != 0)
	n.i.attachLeaf(name, child)
	n.i.lock.Unlock()

	childNode := child.GetFUSENode()
	f, s := childNode.Open(flags, context)
	if s != fuse.OK {
		return nil, nil, s
	}
	return f, n.Inode().NewChild(name, false, childNode), fuse.OK
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
	existingFUSENode, ok := existing.(FUSENode)
	if !ok {
		return nil, fuse.EXDEV
	}
	child, s := existingFUSENode.LinkNode()
	if s != fuse.OK {
		return nil, s
	}

	n.i.lock.Lock()
	defer n.i.lock.Unlock()

	if n.i.existsDirectory(name) || n.i.existsLeaf(name) {
		child.Unlink()
		return nil, fuse.Status(syscall.EEXIST)
	}
	n.i.attachLeaf(name, child)
	return existing.Inode(), fuse.OK
}

func (n *mutableDirectoryFUSENode) Lookup(out *fuse.Attr, name string, context *fuse.Context) (*nodefs.Inode, fuse.Status) {
	n.i.lock.Lock()
	defer n.i.lock.Unlock()

	if child, ok := n.i.directories[name]; ok {
		childNode := child.GetFUSENode()
		if s := childNode.GetAttr(out, nil, context); s != fuse.OK {
			return nil, s
		}
		return n.Inode().NewChild(name, true, childNode), fuse.OK
	}
	if child, ok := n.i.leaves[name]; ok {
		childNode := child.GetFUSENode()
		if s := childNode.GetAttr(out, nil, context); s != fuse.OK {
			return nil, s
		}
		return n.Inode().NewChild(name, false, childNode), fuse.OK
	}
	return nil, fuse.ENOENT
}

func (n mutableDirectoryFUSENode) Mkdir(name string, mode uint32, context *fuse.Context) (*nodefs.Inode, fuse.Status) {
	n.i.lock.Lock()
	if n.i.existsDirectory(name) || n.i.existsLeaf(name) {
		n.i.lock.Unlock()
		return nil, fuse.Status(syscall.EEXIST)
	}
	child := NewMutableDirectory(n.i.mutableTree)
	n.i.attachDirectory(name, child)
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

func (nOld *mutableDirectoryFUSENode) Rename(oldName string, newParent nodefs.Node, newName string, context *fuse.Context) fuse.Status {
	nNew, ok := newParent.(*mutableDirectoryFUSENode)
	if !ok {
		return fuse.EXDEV
	}

	util.LockMutexPair(&nOld.i.lock, &nNew.i.lock)
	defer util.UnlockMutexPair(&nOld.i.lock, &nNew.i.lock)

	// Renaming a directory.
	if child, ok := nOld.detachDirectory(oldName); ok {
		if nNew.i.existsLeaf(newName) {
			nOld.i.attachDirectory(oldName, child)
			return fuse.ENOTDIR
		}
		if existingChild, ok := nNew.detachDirectory(newName); ok {
			if isEmpty, err := existingChild.IsEmpty(); err != nil {
				nOld.i.attachDirectory(oldName, child)
				nNew.i.attachDirectory(newName, existingChild)
				return fuse.EIO
			} else if !isEmpty {
				nOld.i.attachDirectory(oldName, child)
				nNew.i.attachDirectory(newName, existingChild)
				return fuse.Status(syscall.ENOTEMPTY)
			}
		}
		nNew.i.attachDirectory(newName, child)
		return fuse.OK
	}

	// Renaming a file or symlink.
	if child, ok := nOld.detachLeaf(oldName); ok {
		if nNew.i.existsDirectory(newName) {
			nOld.i.attachLeaf(oldName, child)
			return fuse.EISDIR
		}
		if existingChild, ok := nNew.detachLeaf(newName); ok {
			existingChild.Unlink()
		}
		nNew.i.attachLeaf(newName, child)
		return fuse.OK
	}

	return fuse.ENOENT
}

func (n *mutableDirectoryFUSENode) Rmdir(name string, context *fuse.Context) fuse.Status {
	n.i.lock.Lock()
	defer n.i.lock.Unlock()

	if child, ok := n.detachDirectory(name); ok {
		if isEmpty, err := child.IsEmpty(); err != nil {
			n.i.attachDirectory(name, child)
			return fuse.EIO
		} else if !isEmpty {
			n.i.attachDirectory(name, child)
			return fuse.Status(syscall.ENOTEMPTY)
		}
		return fuse.OK
	}
	if n.i.existsLeaf(name) {
		return fuse.ENOTDIR
	}
	return fuse.ENOENT
}

func (n *mutableDirectoryFUSENode) Symlink(name string, content string, context *fuse.Context) (*nodefs.Inode, fuse.Status) {
	n.i.lock.Lock()
	if n.i.existsDirectory(name) || n.i.existsLeaf(name) {
		n.i.lock.Unlock()
		return nil, fuse.Status(syscall.EEXIST)
	}
	child := NewSymlink(content)
	n.i.attachLeaf(name, child)
	n.i.lock.Unlock()

	childNode := child.GetFUSENode()
	return n.Inode().NewChild(name, false, childNode), fuse.OK
}

func (n *mutableDirectoryFUSENode) Unlink(name string, context *fuse.Context) fuse.Status {
	n.i.lock.Lock()
	defer n.i.lock.Unlock()

	if child, ok := n.detachLeaf(name); ok {
		child.Unlink()
		return fuse.OK
	}
	if n.i.existsDirectory(name) {
		return fuse.EPERM
	}
	return fuse.ENOENT
}
