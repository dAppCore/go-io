package sftp

import (
	"cmp"
	goio "io"
	"io/fs"
	"os"
	"path"
	"slices"
	"strings"

	core "dappco.re/go"
	coreio "dappco.re/go/io"
	pkgsftp "github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

const (
	opNew       = "sftp.New"
	opRead      = "sftp.Read"
	opWriteMode = "sftp.WriteMode"
	opRename    = "sftp.Rename"
	opCreate    = "sftp.Create"
	opAppend    = "sftp.Append"
	opOpen      = "sftp.Open"

	errOpenFailed         = "open failed: "
	errCreateParentFailed = "create parent failed: "
)

// Medium is an SFTP-backed implementation of coreio.Medium.
type Medium struct {
	client      *pkgsftp.Client
	sshClient   *ssh.Client
	root        string
	ownsClient  bool
	ownsSSHConn bool
}

var _ coreio.Medium = (*Medium)(nil)

// Options configures an SFTP Medium.
type Options struct {
	Client *pkgsftp.Client

	SSHClient  *ssh.Client
	Address    string
	User       string
	Password   string
	PrivateKey []byte

	Config          *ssh.ClientConfig
	HostKeyCallback ssh.HostKeyCallback
	Root            string
}

// New creates an SFTP Medium. Tests and callers that already manage transport
// state can inject Client directly; otherwise New dials Address using SSH.
func New(options Options) (*Medium, error) {
	root := normaliseRoot(options.Root)
	if options.Client != nil {
		return &Medium{client: options.Client, root: root}, nil
	}

	if options.SSHClient != nil {
		client, err := pkgsftp.NewClient(options.SSHClient)
		if err != nil {
			return nil, core.E(opNew, "failed to create SFTP client", err)
		}
		return &Medium{client: client, sshClient: options.SSHClient, root: root, ownsClient: true}, nil
	}

	config, err := sshConfig(options)
	if err != nil {
		return nil, err
	}
	if options.Address == "" {
		return nil, core.E(opNew, "address is required", fs.ErrInvalid)
	}

	sshClient, err := ssh.Dial("tcp", options.Address, config)
	if err != nil {
		return nil, core.E(opNew, "failed to dial SSH server", err)
	}

	client, err := pkgsftp.NewClient(sshClient)
	if err != nil {
		sshClient.Close()
		return nil, core.E(opNew, "failed to create SFTP client", err)
	}

	return &Medium{
		client:      client,
		sshClient:   sshClient,
		root:        root,
		ownsClient:  true,
		ownsSSHConn: true,
	}, nil
}

func sshConfig(options Options) (*ssh.ClientConfig, error) {
	if options.Config != nil {
		return options.Config, nil
	}
	if options.User == "" {
		return nil, core.E(opNew, "user is required", fs.ErrInvalid)
	}
	if options.HostKeyCallback == nil {
		return nil, core.E(opNew, "host key callback is required", fs.ErrInvalid)
	}

	var auth []ssh.AuthMethod
	if options.Password != "" {
		auth = append(auth, ssh.Password(options.Password))
	}
	if len(options.PrivateKey) > 0 {
		signer, err := ssh.ParsePrivateKey(options.PrivateKey)
		if err != nil {
			return nil, core.E(opNew, "failed to parse private key", err)
		}
		auth = append(auth, ssh.PublicKeys(signer))
	}
	if len(auth) == 0 {
		return nil, core.E(opNew, "password or private key is required", fs.ErrInvalid)
	}

	return &ssh.ClientConfig{
		User:            options.User,
		Auth:            auth,
		HostKeyCallback: options.HostKeyCallback,
	}, nil
}

func normaliseRoot(root string) string {
	clean := path.Clean("/" + root)
	if clean == "." || clean == "" {
		return "/"
	}
	return clean
}

func cleanRelative(filePath string) string {
	clean := path.Clean("/" + strings.ReplaceAll(filePath, "\\", "/"))
	if clean == "/" {
		return ""
	}
	return strings.TrimPrefix(clean, "/")
}

func (medium *Medium) remotePath(filePath string) string {
	relative := cleanRelative(filePath)
	if relative == "" {
		return medium.root
	}
	if medium.root == "/" {
		return "/" + relative
	}
	return path.Join(medium.root, relative)
}

func (medium *Medium) requiredRemotePath(operation, filePath string) (string, error) {
	if cleanRelative(filePath) == "" {
		return "", core.E(operation, "path is required", fs.ErrInvalid)
	}
	return medium.remotePath(filePath), nil
}

func (medium *Medium) ensureParent(remotePath string) error {
	parent := path.Dir(remotePath)
	if parent == "." || parent == "/" {
		return nil
	}
	return medium.client.MkdirAll(parent)
}

// Close closes clients created by New. Injected clients remain caller-owned.
func (medium *Medium) Close() error {
	var err error
	if medium.ownsClient && medium.client != nil {
		err = medium.client.Close()
	}
	if medium.ownsSSHConn && medium.sshClient != nil {
		if closeErr := medium.sshClient.Close(); err == nil {
			err = closeErr
		}
	}
	return err
}

// Read reads a remote file into a string.
func (medium *Medium) Read(filePath string) (string, error) {
	remotePath, err := medium.requiredRemotePath(opRead, filePath)
	if err != nil {
		return "", err
	}
	file, err := medium.client.Open(remotePath)
	if err != nil {
		return "", core.E(opRead, core.Concat(errOpenFailed, remotePath), err)
	}
	defer file.Close()

	data, err := goio.ReadAll(file)
	if err != nil {
		return "", core.E(opRead, core.Concat("read failed: ", remotePath), err)
	}
	return string(data), nil
}

// Write writes a remote file using the default file mode.
func (medium *Medium) Write(filePath, content string) error {
	return medium.WriteMode(filePath, content, 0644)
}

// WriteMode writes a remote file and applies POSIX permissions when supported
// by the SFTP server.
func (medium *Medium) WriteMode(filePath, content string, mode fs.FileMode) error {
	remotePath, err := medium.requiredRemotePath(opWriteMode, filePath)
	if err != nil {
		return err
	}
	if err := medium.ensureParent(remotePath); err != nil {
		return core.E(opWriteMode, core.Concat(errCreateParentFailed, remotePath), err)
	}

	file, err := medium.client.OpenFile(remotePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC)
	if err != nil {
		return core.E(opWriteMode, core.Concat(errOpenFailed, remotePath), err)
	}
	if _, err := file.Write([]byte(content)); err != nil {
		file.Close()
		return core.E(opWriteMode, core.Concat("write failed: ", remotePath), err)
	}
	if closeErr := file.Close(); closeErr != nil {
		return core.E(opWriteMode, core.Concat("close failed: ", remotePath), closeErr)
	}
	if mode != 0 {
		if err := medium.client.Chmod(remotePath, os.FileMode(mode)); err != nil {
			return core.E(opWriteMode, core.Concat("chmod failed: ", remotePath), err)
		}
	}
	return nil
}

// EnsureDir creates a remote directory and any missing parents.
func (medium *Medium) EnsureDir(filePath string) error {
	remotePath := medium.remotePath(filePath)
	if remotePath == medium.root {
		return nil
	}
	if err := medium.client.MkdirAll(remotePath); err != nil {
		return core.E("sftp.EnsureDir", core.Concat("mkdir failed: ", remotePath), err)
	}
	return nil
}

// IsFile reports whether filePath exists and is not a directory.
func (medium *Medium) IsFile(filePath string) bool {
	if cleanRelative(filePath) == "" {
		return false
	}
	info, err := medium.client.Stat(medium.remotePath(filePath))
	return err == nil && !info.IsDir()
}

// Delete removes a remote file or empty directory.
func (medium *Medium) Delete(filePath string) error {
	remotePath, err := medium.requiredRemotePath("sftp.Delete", filePath)
	if err != nil {
		return err
	}
	if err := medium.client.Remove(remotePath); err != nil {
		return core.E("sftp.Delete", core.Concat("remove failed: ", remotePath), err)
	}
	return nil
}

// DeleteAll removes a remote file or directory tree.
func (medium *Medium) DeleteAll(filePath string) error {
	remotePath, err := medium.requiredRemotePath("sftp.DeleteAll", filePath)
	if err != nil {
		return err
	}
	if err := medium.client.RemoveAll(remotePath); err != nil {
		return core.E("sftp.DeleteAll", core.Concat("remove all failed: ", remotePath), err)
	}
	return nil
}

// Rename renames a remote path.
func (medium *Medium) Rename(oldPath, newPath string) error {
	oldRemotePath, err := medium.requiredRemotePath(opRename, oldPath)
	if err != nil {
		return err
	}
	newRemotePath, err := medium.requiredRemotePath(opRename, newPath)
	if err != nil {
		return err
	}
	if err := medium.ensureParent(newRemotePath); err != nil {
		return core.E(opRename, core.Concat(errCreateParentFailed, newRemotePath), err)
	}
	if err := medium.client.Rename(oldRemotePath, newRemotePath); err != nil {
		return core.E(opRename, core.Concat("rename failed: ", oldRemotePath), err)
	}
	return nil
}

// List returns the immediate children under a remote directory.
func (medium *Medium) List(filePath string) ([]fs.DirEntry, error) {
	remotePath := medium.remotePath(filePath)
	infos, err := medium.client.ReadDir(remotePath)
	if err != nil {
		return nil, core.E("sftp.List", core.Concat("read dir failed: ", remotePath), err)
	}

	entries := make([]fs.DirEntry, 0, len(infos))
	for _, info := range infos {
		entries = append(entries, fs.FileInfoToDirEntry(info))
	}
	slices.SortFunc(entries, func(a, b fs.DirEntry) int {
		return cmp.Compare(a.Name(), b.Name())
	})
	return entries, nil
}

// Stat returns metadata for a remote path.
func (medium *Medium) Stat(filePath string) (fs.FileInfo, error) {
	remotePath, err := medium.requiredRemotePath("sftp.Stat", filePath)
	if err != nil {
		return nil, err
	}
	info, err := medium.client.Stat(remotePath)
	if err != nil {
		return nil, core.E("sftp.Stat", core.Concat("stat failed: ", remotePath), err)
	}
	return info, nil
}

// Open opens a remote file for reading.
func (medium *Medium) Open(filePath string) (fs.File, error) {
	remotePath, err := medium.requiredRemotePath(opOpen, filePath)
	if err != nil {
		return nil, err
	}
	file, err := medium.client.Open(remotePath)
	if err != nil {
		return nil, core.E(opOpen, core.Concat(errOpenFailed, remotePath), err)
	}
	return file, nil
}

// Create opens a remote file for replacement.
func (medium *Medium) Create(filePath string) (goio.WriteCloser, error) {
	remotePath, err := medium.requiredRemotePath(opCreate, filePath)
	if err != nil {
		return nil, err
	}
	if err := medium.ensureParent(remotePath); err != nil {
		return nil, core.E(opCreate, core.Concat(errCreateParentFailed, remotePath), err)
	}
	file, err := medium.client.OpenFile(remotePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC)
	if err != nil {
		return nil, core.E(opCreate, core.Concat(errOpenFailed, remotePath), err)
	}
	return file, nil
}

// Append opens a remote file for appending, creating it when missing.
func (medium *Medium) Append(filePath string) (goio.WriteCloser, error) {
	remotePath, err := medium.requiredRemotePath(opAppend, filePath)
	if err != nil {
		return nil, err
	}
	if err := medium.ensureParent(remotePath); err != nil {
		return nil, core.E(opAppend, core.Concat(errCreateParentFailed, remotePath), err)
	}
	file, err := medium.client.OpenFile(remotePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND)
	if err != nil {
		return nil, core.E(opAppend, core.Concat(errOpenFailed, remotePath), err)
	}
	return file, nil
}

// ReadStream opens a remote file as an io.ReadCloser.
func (medium *Medium) ReadStream(filePath string) (goio.ReadCloser, error) {
	file, err := medium.Open(filePath)
	if err != nil {
		return nil, err
	}
	return file, nil
}

// WriteStream opens a remote file as an io.WriteCloser.
func (medium *Medium) WriteStream(filePath string) (goio.WriteCloser, error) {
	return medium.Create(filePath)
}

// Exists reports whether a remote path exists.
func (medium *Medium) Exists(filePath string) bool {
	if cleanRelative(filePath) == "" {
		return false
	}
	_, err := medium.client.Stat(medium.remotePath(filePath))
	return err == nil
}

// IsDir reports whether a remote path exists and is a directory.
func (medium *Medium) IsDir(filePath string) bool {
	if cleanRelative(filePath) == "" {
		return false
	}
	info, err := medium.client.Stat(medium.remotePath(filePath))
	return err == nil && info.IsDir()
}
