package blobstore

import (
	"context"

	"github.com/EdSchouten/bazel-buildbarn/pkg/util"
)

// RandomAccessBlobAccess is an abstraction for a data store that in
// addition to the operations of BlobAccess, permits read operations at
// random offsets without explicit handles to blobs.
//
// This interface should typically not be implemented for remote network
// protocols, for the reason that it cannot be used in combination with
// MerkleBlobAccess to validate the integrity of data transferred.
type RandomAccessBlobAccess interface {
	BlobAccess

	GetAndReadAt(ctx context.Context, digest *util.Digest, b []byte, off int64) (int, error)
}
