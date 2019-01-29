package fuse

const (
	defaultBlockSize = 512
)

func toBlockSize(size uint64) uint64 {
	return (size + defaultBlockSize - 1) / defaultBlockSize
}
