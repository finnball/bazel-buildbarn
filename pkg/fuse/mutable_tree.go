package fuse

import (
	"github.com/EdSchouten/bazel-buildbarn/pkg/filesystem"
)

type MutableTree interface {
	NewFile() (filesystem.File, error)
}
