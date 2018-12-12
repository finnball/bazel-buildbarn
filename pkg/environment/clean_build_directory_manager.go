package environment

import (
	"github.com/EdSchouten/bazel-buildbarn/pkg/util"
)

type cleanBuildDirectoryManager struct {
	base Manager
}

// NewCleanBuildDirectoryManager is an adapter for Manager that upon
// acquistion empties out the build directory. This ensures that the
// build action is executed in a clean environment.
func NewCleanBuildDirectoryManager(base Manager) Manager {
	return &cleanBuildDirectoryManager{
		base: base,
	}
}

func (em *cleanBuildDirectoryManager) Acquire(actionDigest *util.Digest, platformProperties map[string]string) (Environment, error) {
	// Allocate underlying environment.
	environment, err := em.base.Acquire(actionDigest, platformProperties)
	if err != nil {
		return nil, err
	}

	// Remove all contents prior to use.
	if err := environment.GetBuildDirectory().RemoveAllChildren(); err != nil {
		environment.Release()
		return nil, util.StatusWrap(err, "Failed to clean build directory prior to build")
	}
	return environment, nil
}
