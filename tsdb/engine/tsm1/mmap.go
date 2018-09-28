
package tsm1

type Mmap struct {
	backend []byte // Slice representing the full mmapped space
	bytes []byte   // Slice representing the mmapped space starting at offset
}
