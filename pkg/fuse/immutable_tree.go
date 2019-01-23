package fuse

import (
	"github.com/EdSchouten/bazel-buildbarn/pkg/util"
	remoteexecution "github.com/bazelbuild/remote-apis/build/bazel/remote/execution/v2"
)

// ImmutableTree keeps track of any global state that needs to be
// tracked by ImmutableDirectoryNode and ImmutableFileNode objects. More
// pragmatically stated, this interface is used by these types to
// actually access underlying storage. It is used to obtain directory
// listings and chunks of data files.
type ImmutableTree interface {
	GetDirectory(digest *util.Digest) (*remoteexecution.Directory, error)
	ReadFileAt(digest *util.Digest, b []byte, off int64) (int, error)
}
