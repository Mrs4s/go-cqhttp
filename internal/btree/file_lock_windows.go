package btree

import "syscall"

type windowsFileLock struct {
	fd syscall.Handle
}

func (fl *windowsFileLock) release() error {
	return syscall.Close(fl.fd)
}

func newFileLock(path string) (fileLock, error) {
	pathp, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return nil, err
	}

	const access uint32 = syscall.GENERIC_READ | syscall.GENERIC_WRITE
	fd, err := syscall.CreateFile(pathp, access, 0, nil, syscall.OPEN_EXISTING, syscall.FILE_ATTRIBUTE_NORMAL, 0)
	if err == syscall.ERROR_FILE_NOT_FOUND {
		fd, err = syscall.CreateFile(pathp, access, 0, nil, syscall.OPEN_ALWAYS, syscall.FILE_ATTRIBUTE_NORMAL, 0)
	}
	if err != nil {
		return nil, err
	}
	return &windowsFileLock{fd: fd}, nil
}
