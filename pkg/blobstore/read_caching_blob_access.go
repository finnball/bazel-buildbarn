package blobstore

import (
	"context"
	"io"

	"github.com/EdSchouten/bazel-buildbarn/pkg/util"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type readCachingBlobAccess struct {
	slow BlobAccess
	fast RandomAccessBlobAccess
}

// NewReadCachingBlobAccess turns a fast data store into a read cache
// for a slow data store. All writes are performed against the slow data
// store directly. The slow data store is only accessed for reading in
// case the fast data store does not contain the blob. The blob is then
// streamed into the fast data store.
//
// In addition to improving read performance, this adapter may be used
// to add support for random access reads to a data store that can only
// stream blobs.
func NewReadCachingBlobAccess(slow BlobAccess, fast RandomAccessBlobAccess) RandomAccessBlobAccess {
	return &readCachingBlobAccess{
		slow: slow,
		fast: fast,
	}
}

func (ba *readCachingBlobAccess) sync(ctx context.Context, digest *util.Digest) error {
	sizeBytes, r, err := ba.slow.Get(ctx, digest)
	if err != nil {
		return err
	}
	return ba.fast.Put(ctx, digest, sizeBytes, r)
}

func (ba *readCachingBlobAccess) Get(ctx context.Context, digest *util.Digest) (int64, io.ReadCloser, error) {
	// TODO(edsch): Should there be a maximum number of loop iterations?
	for {
		sizeBytes, r, err := ba.fast.Get(ctx, digest)
		if s := status.Convert(err); s.Code() != codes.NotFound {
			return sizeBytes, r, err
		}
		if err := ba.sync(ctx, digest); err != nil {
			return 0, nil, err
		}
	}
}

func (ba *readCachingBlobAccess) GetAndReadAt(ctx context.Context, digest *util.Digest, b []byte, off int64) (int, error) {
	// TODO(edsch): Should there be a maximum number of loop iterations?
	for {
		n, err := ba.fast.GetAndReadAt(ctx, digest, b, off)
		if s := status.Convert(err); s.Code() != codes.NotFound {
			return n, err
		}
		if err := ba.sync(ctx, digest); err != nil {
			return 0, err
		}
	}
}

func (ba *readCachingBlobAccess) Put(ctx context.Context, digest *util.Digest, sizeBytes int64, r io.ReadCloser) error {
	return ba.slow.Put(ctx, digest, sizeBytes, r)
}

func (ba *readCachingBlobAccess) Delete(ctx context.Context, digest *util.Digest) error {
	fastErrChan := make(chan error, 1)
	go func() {
		fastErrChan <- ba.fast.Delete(ctx, digest)
	}()
	slowErr := ba.slow.Delete(ctx, digest)
	fastErr := <-fastErrChan
	if slowErr != nil {
		return slowErr
	}
	return fastErr
}

func (ba *readCachingBlobAccess) FindMissing(ctx context.Context, digests []*util.Digest) ([]*util.Digest, error) {
	return ba.slow.FindMissing(ctx, digests)
}
