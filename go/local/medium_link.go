package local

import (
	"syscall"

	core "dappco.re/go"
)

type linkInfo struct {
	IsSymlink bool
	Mode      core.FileMode
	Size      int64
	ModTime   core.Time
}

func lstat(path string) (
	linkInfo,
	error,
) {
	result := core.Lstat(path)
	if !result.OK {
		if err, ok := result.Value.(error); ok {
			return linkInfo{}, err
		}
		return linkInfo{}, core.E("local.lstat", msgUnexpectedResultType, nil)
	}
	info, ok := result.Value.(core.FsFileInfo)
	if !ok {
		return linkInfo{}, core.E("local.lstat", msgUnexpectedResultType, nil)
	}
	mode := info.Mode()
	return linkInfo{
		IsSymlink: mode&core.ModeSymlink != 0,
		Mode:      mode,
		Size:      info.Size(),
		ModTime:   info.ModTime(),
	}, nil
}

func readlink(path string) (
	string,
	error,
) {
	size := 256
	for {
		linkBuffer := make([]byte, size)
		bytesRead, err := syscall.Readlink(path, linkBuffer)
		if err != nil {
			return "", err
		}
		if bytesRead < len(linkBuffer) {
			return string(linkBuffer[:bytesRead]), nil
		}
		size *= 2
	}
}
