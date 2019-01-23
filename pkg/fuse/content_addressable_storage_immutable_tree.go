package fuse

import (
	"context"

	"github.com/EdSchouten/bazel-buildbarn/pkg/blobstore"
	"github.com/EdSchouten/bazel-buildbarn/pkg/cas"
	"github.com/EdSchouten/bazel-buildbarn/pkg/util"
	remoteexecution "github.com/bazelbuild/remote-apis/build/bazel/remote/execution/v2"
)

type contentAddressableStorageImmutableTree struct {
	context                   context.Context
	contentAddressableStorage cas.ContentAddressableStorage
	blobAccess                blobstore.RandomAccessBlobAccess
}

// NewContentAddressableStorageImmutableTree creates a read-only view of
// a file system that is backend by a remote execution Content
// Addressable Storage (CAS).
//
// Directories are loaded through a ContentAddressableStorage, for which
// there are adapters to provide in-memory caching of deserialized
// protocol buffers.
//
// File contents are loaded through a RandomAccessBlobAccess, which is
// needed due to the fact that the FUSE file system permits us to access
// files partially and at arbitrary offsets.
func NewContentAddressableStorageImmutableTree(context context.Context, contentAddressableStorage cas.ContentAddressableStorage, blobAccess blobstore.RandomAccessBlobAccess) ImmutableTree {
	return &contentAddressableStorageImmutableTree{
		context:                   context,
		contentAddressableStorage: contentAddressableStorage,
		blobAccess:                blobAccess,
	}
}

func (it *contentAddressableStorageImmutableTree) GetDirectory(digest *util.Digest) (*remoteexecution.Directory, error) {
	return it.contentAddressableStorage.GetDirectory(it.context, digest)
}

func (it *contentAddressableStorageImmutableTree) ReadFileAt(digest *util.Digest, b []byte, off int64) (int, error) {
	return it.blobAccess.GetAndReadAt(it.context, digest, b, off)
}
