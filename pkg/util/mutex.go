package util

import (
	"sync"
	"unsafe"
)

// LockMutexPair acquires a pair of mutexes using a deadlock avoidance
// algorithm. In case both arguments refer to the same lock, the lock is
// only acquired once.
func LockMutexPair(a *sync.Mutex, b *sync.Mutex) {
	ap := uintptr(unsafe.Pointer(a))
	bp := uintptr(unsafe.Pointer(b))
	if ap < bp {
		a.Lock()
		b.Lock()
	} else if ap > bp {
		b.Lock()
		a.Lock()
	} else {
		a.Lock()
	}
}

// UnlockMutexPair releases a pair of mutexes. In case both arguments
// refer to the same lock, the lock is only released once.
func UnlockMutexPair(a *sync.Mutex, b *sync.Mutex) {
	ap := uintptr(unsafe.Pointer(a))
	bp := uintptr(unsafe.Pointer(b))
	if ap == bp {
		a.Unlock()
	} else {
		a.Unlock()
		b.Unlock()
	}
}
