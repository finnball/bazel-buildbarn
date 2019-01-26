package fuse

import (
	"github.com/hanwen/go-fuse/fuse"
)

type dirEntryList []fuse.DirEntry

func (l dirEntryList) Len() int {
	return len(l)
}

func (l dirEntryList) Less(i int, j int) bool {
	return l[i].Name < l[j].Name
}

func (l dirEntryList) Swap(i int, j int) {
	t := l[i]
	l[i] = l[j]
	l[j] = t
}
