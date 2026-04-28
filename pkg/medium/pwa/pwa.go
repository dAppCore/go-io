package pwa

import (
	"errors"
	goio "io"
	"io/fs"
	"path"
	"strings"
	"sync"

	core "dappco.re/go"
	coreio "dappco.re/go/io"
	borgdatanode "forge.lthn.ai/Snider/Borg/pkg/datanode"
	borgpwa "forge.lthn.ai/Snider/Borg/pkg/pwa"
)

const (
	opEnsureDataNode = "pwa.ensureDataNode"
	opRead           = "pwa.Read"
	opList           = "pwa.List"
	opStat           = "pwa.Stat"
	opOpen           = "pwa.Open"
)

// ErrNotImplemented remains for callers that used the round-1 stub sentinel.
var ErrNotImplemented = errors.New("pwa medium is not implemented")

var errReadOnly = errors.New("pwa medium is read-only")

// Medium is a Borg PWA DataNode-backed implementation of coreio.Medium.
type Medium struct {
	url      string
	dataNode *borgdatanode.DataNode
	mu       sync.RWMutex
}

var _ coreio.Medium = (*Medium)(nil)
var _ fs.FS = (*Medium)(nil)

// Options configures a PWA Medium.
type Options struct {
	URL string
}

// New creates a PWA Medium that lazily downloads the target into a Borg DataNode.
func New(options Options) (*Medium, error) {
	return &Medium{url: options.URL}, nil
}

func cleanRelative(filePath string) string {
	clean := path.Clean("/" + strings.ReplaceAll(filePath, "\\", "/"))
	if clean == "/" {
		return ""
	}
	return strings.TrimPrefix(clean, "/")
}

func requiredPath(operation, filePath string) (string, error) {
	clean := cleanRelative(filePath)
	if clean == "" {
		return "", core.E(operation, "path is required", fs.ErrInvalid)
	}
	return clean, nil
}

func (medium *Medium) ensureDataNode() (*borgdatanode.DataNode, error) {
	medium.mu.RLock()
	dataNode := medium.dataNode
	medium.mu.RUnlock()
	if dataNode != nil {
		return dataNode, nil
	}

	medium.mu.Lock()
	defer medium.mu.Unlock()
	if medium.dataNode != nil {
		return medium.dataNode, nil
	}

	targetURL := strings.TrimSpace(medium.url)
	if targetURL == "" {
		return nil, core.E(opEnsureDataNode, "url is required", fs.ErrInvalid)
	}

	client := borgpwa.NewPWAClient()
	manifestURL, err := client.FindManifest(targetURL)
	if err != nil {
		return nil, core.E(opEnsureDataNode, "failed to find PWA manifest", err)
	}
	dataNode, err = client.DownloadAndPackagePWA(targetURL, manifestURL, nil)
	if err != nil {
		return nil, core.E(opEnsureDataNode, "failed to download PWA", err)
	}
	medium.dataNode = dataNode
	return dataNode, nil
}

func readOnly(operation string) error {
	return core.E(operation, "PWA medium is read-only", errReadOnly)
}

// Read reads a downloaded PWA asset from Borg's DataNode.
func (medium *Medium) Read(filePath string) (string, error) {
	clean, err := requiredPath(opRead, filePath)
	if err != nil {
		return "", err
	}
	dataNode, err := medium.ensureDataNode()
	if err != nil {
		return "", err
	}
	file, err := dataNode.Open(clean)
	if err != nil {
		return "", core.E(opRead, core.Concat("not found: ", clean), fs.ErrNotExist)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return "", core.E(opRead, core.Concat("stat failed: ", clean), err)
	}
	if info.IsDir() {
		return "", core.E(opRead, core.Concat("path is a directory: ", clean), fs.ErrInvalid)
	}
	data, err := goio.ReadAll(file)
	if err != nil {
		return "", core.E(opRead, core.Concat("read failed: ", clean), err)
	}
	return string(data), nil
}

// Write returns an error because downloaded PWA DataNodes are read-only.
func (medium *Medium) Write(filePath, content string) error {
	return readOnly("pwa.Write")
}

// WriteMode returns an error because downloaded PWA DataNodes are read-only.
func (medium *Medium) WriteMode(filePath, content string, mode fs.FileMode) error {
	return readOnly("pwa.WriteMode")
}

// EnsureDir returns an error because downloaded PWA DataNodes are read-only.
func (medium *Medium) EnsureDir(filePath string) error {
	return readOnly("pwa.EnsureDir")
}

// IsFile reports whether filePath exists as a PWA asset.
func (medium *Medium) IsFile(filePath string) bool {
	clean := cleanRelative(filePath)
	if clean == "" {
		return false
	}
	info, err := medium.Stat(clean)
	return err == nil && !info.IsDir()
}

// Delete returns an error because downloaded PWA DataNodes are read-only.
func (medium *Medium) Delete(filePath string) error {
	return readOnly("pwa.Delete")
}

// DeleteAll returns an error because downloaded PWA DataNodes are read-only.
func (medium *Medium) DeleteAll(filePath string) error {
	return readOnly("pwa.DeleteAll")
}

// Rename returns an error because downloaded PWA DataNodes are read-only.
func (medium *Medium) Rename(oldPath, newPath string) error {
	return readOnly("pwa.Rename")
}

// List lists downloaded PWA assets from Borg's DataNode.
func (medium *Medium) List(filePath string) ([]fs.DirEntry, error) {
	dataNode, err := medium.ensureDataNode()
	if err != nil {
		return nil, err
	}
	clean := cleanRelative(filePath)
	entries, err := dataNode.ReadDir(clean)
	if err != nil {
		return nil, core.E(opList, core.Concat("not found: ", clean), fs.ErrNotExist)
	}
	return entries, nil
}

// Stat returns metadata for a downloaded PWA asset.
func (medium *Medium) Stat(filePath string) (fs.FileInfo, error) {
	dataNode, err := medium.ensureDataNode()
	if err != nil {
		return nil, err
	}
	clean := cleanRelative(filePath)
	info, err := dataNode.Stat(clean)
	if err != nil {
		return nil, core.E(opStat, core.Concat("not found: ", clean), fs.ErrNotExist)
	}
	return info, nil
}

// Open opens a downloaded PWA asset from Borg's DataNode.
func (medium *Medium) Open(filePath string) (fs.File, error) {
	clean, err := requiredPath(opOpen, filePath)
	if err != nil {
		return nil, err
	}
	dataNode, err := medium.ensureDataNode()
	if err != nil {
		return nil, err
	}
	file, err := dataNode.Open(clean)
	if err != nil {
		return nil, core.E(opOpen, core.Concat("not found: ", clean), fs.ErrNotExist)
	}
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, core.E(opOpen, core.Concat("stat failed: ", clean), err)
	}
	if info.IsDir() {
		file.Close()
		return nil, core.E(opOpen, core.Concat("path is a directory: ", clean), fs.ErrInvalid)
	}
	return file, nil
}

// Create returns an error because downloaded PWA DataNodes are read-only.
func (medium *Medium) Create(filePath string) (goio.WriteCloser, error) {
	return nil, readOnly("pwa.Create")
}

// Append returns an error because downloaded PWA DataNodes are read-only.
func (medium *Medium) Append(filePath string) (goio.WriteCloser, error) {
	return nil, readOnly("pwa.Append")
}

// ReadStream opens a downloaded PWA asset as a stream.
func (medium *Medium) ReadStream(filePath string) (goio.ReadCloser, error) {
	file, err := medium.Open(filePath)
	if err != nil {
		return nil, err
	}
	return file, nil
}

// WriteStream returns an error because downloaded PWA DataNodes are read-only.
func (medium *Medium) WriteStream(filePath string) (goio.WriteCloser, error) {
	return nil, readOnly("pwa.WriteStream")
}

// Exists reports whether filePath exists in the downloaded PWA DataNode.
func (medium *Medium) Exists(filePath string) bool {
	_, err := medium.Stat(filePath)
	return err == nil
}

// IsDir reports whether filePath exists as a directory in the downloaded PWA DataNode.
func (medium *Medium) IsDir(filePath string) bool {
	info, err := medium.Stat(filePath)
	return err == nil && info.IsDir()
}
