package tsm1

import (
	"os"
)

type mMap struct {
	backend   []byte   // Slice representing the full mmapped space
	pageStart int64    // Start location of mmap range
	pageDelta int64    // Start location of Bytes relative to pageStart
	mapLength int      // Length of map range
	length    int      // length of the desired Bytes slice
	f         *os.File // File to be mapped
}

func NewMMap(f *os.File, offset int64, length int) *mMap {
	m := &mMap{}
	// Offset must be a multiple of the page size, typically 4096. We'll
	// extend the mmap range at the beginning so that it starts at the
	// page boundary, then return a slice starting at the requested offset.
	pageSize := int64(os.Getpagesize())
	m.pageStart = int64((offset / pageSize) * pageSize) // offset to prev boundary
	m.pageDelta = offset - m.pageStart                  // shift from boundary to requested location
	m.mapLength = length + int(m.pageDelta)
	m.f = f
	m.length = length

	return m
}
