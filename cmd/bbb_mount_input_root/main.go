package main

import (
	"context"
	"flag"
	"log"
	"os"
	"strconv"

	"github.com/EdSchouten/bazel-buildbarn/pkg/blobstore"
	"github.com/EdSchouten/bazel-buildbarn/pkg/blobstore/circular"
	"github.com/EdSchouten/bazel-buildbarn/pkg/blobstore/configuration"
	"github.com/EdSchouten/bazel-buildbarn/pkg/cas"
	"github.com/EdSchouten/bazel-buildbarn/pkg/filesystem"
	fuse "github.com/EdSchouten/bazel-buildbarn/pkg/fuse"
	"github.com/EdSchouten/bazel-buildbarn/pkg/util"
	remoteexecution "github.com/bazelbuild/remote-apis/build/bazel/remote/execution/v2"
	"github.com/hanwen/go-fuse/fuse/nodefs"
)

func openCache(path string) (blobstore.RandomAccessBlobAccess, error) {
	var offsetFileSizeBytes uint64 = 1024 * 1024
	var dataFileSizeBytes uint64 = 1024 * 1024 * 1024
	var dataAllocationChunkSizeBytes uint64 = 1024 * 1024

	circularDirectory, err := filesystem.NewLocalDirectory(path)
	if err != nil {
		return nil, err
	}
	defer circularDirectory.Close()
	dataFile, err := circularDirectory.OpenFile("data", os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}
	stateFile, err := circularDirectory.OpenFile("state", os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}
	offsetFile, err := circularDirectory.OpenFile("offset", os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}

	stateStore, err := circular.NewFileStateStore(stateFile, dataFileSizeBytes)
	if err != nil {
		return nil, err
	}
	return circular.NewCircularBlobAccess(
		circular.NewFileOffsetStore(offsetFile, offsetFileSizeBytes),
		circular.NewFileDataStore(dataFile, dataFileSizeBytes),
		circular.NewPositiveSizedBlobStateStore(
			circular.NewBulkAllocatingStateStore(
				stateStore,
				dataAllocationChunkSizeBytes))), nil
}

func main() {
	var (
		blobstoreConfig = flag.String("blobstore-config", "/config/blobstore.conf", "Configuration for blob storage")
	)
	flag.Parse()
	if flag.NArg() != 4 {
		log.Fatal("Usage: bbb_mount_input_root instance hash size_bytes mountpoint")
	}

	// Storage access.
	contentAddressableStorageBlobAccess, _, err := configuration.CreateBlobAccessObjectsFromConfig(*blobstoreConfig)
	if err != nil {
		log.Fatal("Failed to create blob access: ", err)
	}

	// Access to Content Addressable Storage.
	contentAddressableStorage := cas.NewDirectoryCachingContentAddressableStorage(
		cas.NewBlobAccessContentAddressableStorage(contentAddressableStorageBlobAccess),
		util.DigestKeyWithoutInstance, 1000)

	// Cached access for files.
	circularBlobAccess, err := openCache("foo")
	if err != nil {
		log.Fatal("Failed to open cache: ", err)
	}
	readCachingBlobAccess := blobstore.NewReadCachingBlobAccess(
		contentAddressableStorageBlobAccess,
		circularBlobAccess)

	immutableTree := fuse.NewContentAddressableStorageImmutableTree(
		context.Background(),
		contentAddressableStorage,
		readCachingBlobAccess)

	// Input root digest.
	sizeBytes, err := strconv.ParseInt(flag.Arg(2), 10, 64)
	if err != nil {
		log.Fatal("Failed to parse digest size: ", err)
	}
	digest, err := util.NewDigest(
		flag.Arg(0),
		&remoteexecution.Digest{
			Hash:      flag.Arg(1),
			SizeBytes: sizeBytes,
		})
	if err != nil {
		log.Fatal("Failed to parse digest: ", err)
	}

	dataDir, err := filesystem.NewLocalDirectory("local-files")
	if err != nil {
		log.Fatal("Failed to open directory for mutable tree: ", err)
	}
	mutableTree := fuse.NewDirectoryBackedMutableTree(dataDir)
	root := fuse.NewMutableDirectory(mutableTree)
	root.GetOrCreateDirectory("bazel-out")
	if err := root.MergeImmutableTree(immutableTree, digest); err != nil {
		log.Fatal("Failed merge immutable tree: ", err)
	}
	rootNode := root.GetFUSENode()

	// FUSE mount.
	server, _, err := nodefs.MountRoot(
		flag.Arg(3),
		rootNode,
		nil)
	if err != nil {
		log.Fatal("Failed to mount: ", err)
	}
	log.Print("Mount succeeded")
	server.Serve()
}
