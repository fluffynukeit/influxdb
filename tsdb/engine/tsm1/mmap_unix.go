// +build !windows,!plan9

package tsm1

import (
	"syscall"

	"golang.org/x/sys/unix"
)


// Returns a slice to the mapping data, creating the mapping if not initialized
// or previously released().
func (m *mMap) bytes() (data []byte, err error) {

	if m.backend != nil { // data available and ready to go
		return m.backend[m.pageDelta:], nil
	}

	// anonymous mapping if f == nil
	var mmap []byte
	if m.f == nil {
		mmap, err = unix.Mmap(-1, m.pageStart, m.mapLength, syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_ANON|syscall.MAP_PRIVATE)
	} else {
		mmap, err = unix.Mmap(int(m.f.Fd()), m.pageStart, m.mapLength, syscall.PROT_READ, syscall.MAP_SHARED)
	}

	if err != nil {
		return nil, err
	}

	m.backend = mmap
	data = mmap[m.pageDelta:]

	return data, nil
}

// Release map resources. 
func (m *mMap) release() error {
	if m.backend == nil { // already released
		return nil
	}

	err := unix.Munmap(m.backend)
	if err != nil {
		return err
	}
	m.backend = nil
	return nil
}

// madviseWillNeed gives the kernel the mmap madvise value MADV_WILLNEED, hinting
// that we plan on using the provided buffer in the near future.
func (m *mMap) madviseWillNeed() error {
	return m.madvise(syscall.MADV_WILLNEED)
}

func (m *mMap) madviseDontNeed() error {
	return m.madvise(syscall.MADV_DONTNEED)
}

// From: github.com/boltdb/bolt/bolt_unix.go
func (m *mMap) madvise(advice int) error {
	b, err := m.bytes()
	if err != nil {
		return err
	}
	return unix.Madvise(b, advice)
}
