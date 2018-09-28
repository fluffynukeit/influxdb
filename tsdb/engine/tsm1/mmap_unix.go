// +build !windows,!plan9

package tsm1

import (
	"os"
	"syscall"

	"golang.org/x/sys/unix"
)

func NewMmap(f *os.File, offset int64, length int) (m *Mmap, err error) {
	// Offset must be a multiple of the page size, typically 4096. We'll
	// extend the mmap range at the beginning so that it starts at the 
	// page boundary, then return a slice starting at the requested offset.
	pageSize := int64(os.Getpagesize())
	pageStart := int64((offset / pageSize) * pageSize) // offset to prev boundary
	pageDelta := offset - pageStart // shift from boundary to requested location
	mapLength := length + int(pageDelta)

	// anonymous mapping
	var mmap []byte
	if f == nil {
		mmap, err = unix.Mmap(-1, pageStart, mapLength, syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_ANON|syscall.MAP_PRIVATE)
	} else {
		mmap, err = unix.Mmap(int(f.Fd()), pageStart, mapLength, syscall.PROT_READ, syscall.MAP_SHARED)
	}

	if err != nil {
		return nil, err
	}

	m = &Mmap{}
	m.backend = mmap
	m.bytes = mmap[pageDelta:]

	return m, nil
}

func (m *Mmap) close() error {
	if m.backend == nil {
		return nil
	}

	err := unix.Munmap(m.backend)
	if err != nil {
		return err
	}
	m.backend = nil
	m.bytes = nil
	return nil
}

// madviseWillNeed gives the kernel the mmap madvise value MADV_WILLNEED, hinting
// that we plan on using the provided buffer in the near future.
func madviseWillNeed(b []byte) error {
	return madvise(b, syscall.MADV_WILLNEED)
}

func madviseDontNeed(b []byte) error {
	return madvise(b, syscall.MADV_DONTNEED)
}

// From: github.com/boltdb/bolt/bolt_unix.go
func madvise(b []byte, advice int) (err error) {
	return unix.Madvise(b, advice)
}
