package tsm1

import (
	"errors"
	"os"
	"reflect"
	"sync"
	"syscall"
	"unsafe"
)

// mmap implementation for Windows
// Based on: https://github.com/edsrzf/mmap-go
// Based on: https://github.com/boltdb/bolt/bolt_windows.go
// Ref: https://groups.google.com/forum/#!topic/golang-nuts/g0nLwQI9www

// We keep this map so that we can get back the original handle from the memory address.
var handleLock sync.Mutex
var handleMap = map[uintptr]syscall.Handle{}
var fileMap = map[uintptr]*os.File{}

func openSharedFile(f *os.File) (file *os.File, err error) {

	var access, createmode, sharemode uint32
	var sa *syscall.SecurityAttributes

	access = syscall.GENERIC_READ
	sharemode = uint32(syscall.FILE_SHARE_READ | syscall.FILE_SHARE_WRITE | syscall.FILE_SHARE_DELETE)
	createmode = syscall.OPEN_EXISTING
	fileName := f.Name()

	pathp, err := syscall.UTF16PtrFromString(fileName)
	if err != nil {
		return nil, err
	}

	h, e := syscall.CreateFile(pathp, access, sharemode, sa, createmode, syscall.FILE_ATTRIBUTE_NORMAL, 0)

	if e != nil {
		return nil, e
	}
	//NewFile does not add finalizer, need to close this manually
	return os.NewFile(uintptr(h), fileName), nil
}

func (m *mMap) bytes() (data []byte, err error) {

	if m.backend != nil { // Already open and mapped
		if m.f == nil { // Anonymous mapping, no pageDelta
			return m.backend, nil
		}
		return m.backend[m.pageDelta:], nil
	}

	// TODO: Add support for anonymous mapping on windows
	if f == nil {
		m.backend = make([]byte, m.length)
		return m.backend, nil
	}

	// Open a file mapping handle.
	sizelo := uint32(m.mapLength >> 32)
	sizehi := uint32(m.mapLength) & 0xffffffff

	sharedHandle, errno := openSharedFile(m.f)
	if errno != nil {
		return os.NewSyscallError("CreateFile", errno)
	}

	h, errno := syscall.CreateFileMapping(syscall.Handle(sharedHandle.Fd()), nil, syscall.PAGE_READONLY, sizelo, sizehi, nil)
	if h == 0 {
		return os.NewSyscallError("CreateFileMapping", errno)
	}

	// Create the memory map.
	pageStartlo := uint32(m.pageStart >> 32)
	pageStarthi := uint32(m.pageStart) & 0xffffffff
	addr, errno := syscall.MapViewOfFile(h, syscall.FILE_MAP_READ, pageStartlo, pageStarthi, uintptr(m.mapLength))
	if addr == 0 {
		return os.NewSyscallError("MapViewOfFile", errno)
	}

	handleLock.Lock()
	handleMap[addr] = h
	fileMap[addr] = sharedHandle
	handleLock.Unlock()

	// Convert to a byte array.
	hdr := (*reflect.SliceHeader)(unsafe.Pointer(&out))
	hdr.Data = uintptr(unsafe.Pointer(addr))
	hdr.Len = m.mapLength
	hdr.Cap = m.mapLength

	m := &mMap{}
	m.backend = out

	return out[m.pageDelta:], nil
}

// munmap Windows implementation
// Based on: https://github.com/edsrzf/mmap-go
// Based on: https://github.com/boltdb/bolt/bolt_windows.go
func (m *mMap) close() (err error) {
	handleLock.Lock()
	defer handleLock.Unlock()

	if m.backend == nil {
		return nil
	}

	if m.f == nil { // "anonymous" mapping.  Let GC handle it.
		return nil
	}

	addr := (uintptr)(unsafe.Pointer(&m.backend[0]))
	if err := syscall.UnmapViewOfFile(addr); err != nil {
		return os.NewSyscallError("UnmapViewOfFile", err)
	}

	handle, ok := handleMap[addr]
	if !ok {
		// should be impossible; we would've seen the error above
		return errors.New("unknown base address")
	}
	delete(handleMap, addr)

	e := syscall.CloseHandle(syscall.Handle(handle))
	if e != nil {
		return os.NewSyscallError("CloseHandle", e)
	}

	file, ok := fileMap[addr]
	if !ok {
		// should be impossible; we would've seen the error above
		return errors.New("unknown base address")
	}
	delete(fileMap, addr)

	e = file.Close()
	if e != nil {
		return errors.New("close file" + e.Error())
	}
	m.backend = nil
	return nil
}

// madviseWillNeed is unsupported on Windows.
func (m *mMap) madviseWillNeed() error { return nil }

// madviseDontNeed is unsupported on Windows.
func (m *mMap) madviseDontNeed() error { return nil }

func (m *mMap) madvise(advice int) error {
	// Not implemented
	return nil
}
