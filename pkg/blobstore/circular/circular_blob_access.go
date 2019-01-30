package circular

import (
	"context"
	"errors"
	"io"
	"io/ioutil"
	"sync"

	"github.com/EdSchouten/bazel-buildbarn/pkg/blobstore"
	"github.com/EdSchouten/bazel-buildbarn/pkg/util"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// OffsetStore maps a digest to an offset within the data file. This is
// where the blob's contents may be found.
type OffsetStore interface {
	Get(digest *util.Digest, cursors Cursors) (uint64, int64, bool, error)
	Put(digest *util.Digest, offset uint64, length int64, cursors Cursors) error
}

// DataStore is where the data corresponding with a blob is stored. Data
// can be accessed by providing an offset within the data store and its
// length.
type DataStore interface {
	Put(r io.Reader, offset uint64) error
	Get(offset uint64, size int64) io.Reader
}

// StateStore is where global metadata of the circular storage backend
// is stored, namely the read/write cursors where data is currently
// being stored in the data file.
type StateStore interface {
	GetCursors() Cursors
	Allocate(sizeBytes int64) (uint64, error)
	Invalidate(offset uint64, sizeBytes int64) error
}

type circularBlobAccess struct {
	// Fields that are constant or lockless.
	dataStore DataStore

	// Fields protected by the lock.
	lock        sync.Mutex
	offsetStore OffsetStore
	stateStore  StateStore
}

// NewCircularBlobAccess creates a new circular storage backend. Instead
// of writing data to storage directly, all three storage files are
// injected through separate interfaces.
func NewCircularBlobAccess(offsetStore OffsetStore, dataStore DataStore, stateStore StateStore) blobstore.RandomAccessBlobAccess {
	return &circularBlobAccess{
		offsetStore: offsetStore,
		dataStore:   dataStore,
		stateStore:  stateStore,
	}
}

func (ba *circularBlobAccess) Get(ctx context.Context, digest *util.Digest) (int64, io.ReadCloser, error) {
	ba.lock.Lock()
	cursors := ba.stateStore.GetCursors()
	offset, length, ok, err := ba.offsetStore.Get(digest, cursors)
	ba.lock.Unlock()
	if err != nil {
		return 0, nil, err
	} else if ok {
		return length, ioutil.NopCloser(ba.dataStore.Get(offset, length)), nil
	}
	return 0, nil, status.Errorf(codes.NotFound, "Blob not found")
}

func (ba *circularBlobAccess) GetAndReadAt(ctx context.Context, digest *util.Digest, b []byte, off int64) (int, error) {
	if off < 0 {
		return 0, status.Errorf(codes.InvalidArgument, "Cannot read at negative offset")
	}

	ba.lock.Lock()
	defer ba.lock.Unlock()

	cursors := ba.stateStore.GetCursors()
	offset, length, ok, err := ba.offsetStore.Get(digest, cursors)
	if err != nil {
		return 0, err
	} else if ok {
		// Trim off the first part of the blob.
		if length < off {
			return 0, io.EOF
		}
		offset += uint64(off)
		length -= off
		n, err := io.ReadFull(ba.dataStore.Get(offset, length), b)
		if err == io.ErrUnexpectedEOF {
			err = io.EOF
		}
		return n, err
	}
	return 0, status.Errorf(codes.NotFound, "Blob not found")
}

func (ba *circularBlobAccess) Put(ctx context.Context, digest *util.Digest, sizeBytes int64, r io.ReadCloser) error {
	defer r.Close()

	// Allocate space in the data store.
	ba.lock.Lock()
	offset, err := ba.stateStore.Allocate(sizeBytes)
	ba.lock.Unlock()
	if err != nil {
		return err
	}

	// Write the data to storage.
	if err := ba.dataStore.Put(r, offset); err != nil {
		return err
	}

	ba.lock.Lock()
	cursors := ba.stateStore.GetCursors()
	if cursors.Contains(offset, sizeBytes) {
		err = ba.offsetStore.Put(digest, offset, sizeBytes, cursors)
	} else {
		err = errors.New("Data became stale before write completed")
	}
	ba.lock.Unlock()
	return err
}

func (ba *circularBlobAccess) Delete(ctx context.Context, digest *util.Digest) error {
	ba.lock.Lock()
	defer ba.lock.Unlock()

	cursors := ba.stateStore.GetCursors()
	if offset, length, ok, err := ba.offsetStore.Get(digest, cursors); err != nil {
		return err
	} else if ok {
		return ba.stateStore.Invalidate(offset, length)
	}
	return nil
}

func (ba *circularBlobAccess) FindMissing(ctx context.Context, digests []*util.Digest) ([]*util.Digest, error) {
	ba.lock.Lock()
	defer ba.lock.Unlock()

	cursors := ba.stateStore.GetCursors()
	var missingDigests []*util.Digest
	for _, digest := range digests {
		if _, _, ok, err := ba.offsetStore.Get(digest, cursors); err != nil {
			return nil, err
		} else if !ok {
			missingDigests = append(missingDigests, digest)
		}
	}
	return missingDigests, nil
}
