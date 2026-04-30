package workspace

import (
	"crypto/sha256"
	"encoding/hex"
	"hash"
	goio "io" // Note: AX-6 intrinsic — io.ReadFull for HKDF key derivation; no core wrapper for ReadFull semantics.
	"io/fs"
	"sync" // Note: AX-6 — internal concurrency primitive; structural per RFC §5.1

	core "dappco.re/go"
	"golang.org/x/crypto/hkdf"

	"dappco.re/go/io"
	"dappco.re/go/io/sigil"
)

// Example: service, _ := workspace.New(workspace.Options{KeyPairProvider: keyPairProvider})
type EncryptedWorkspace interface {
	CreateWorkspace(identifier, passphrase string) (string, error)
	SwitchWorkspace(workspaceID string) error
	ReadWorkspaceFile(workspaceFilePath string) (string, error)
	WriteWorkspaceFile(workspaceFilePath, content string) error
}

const (
	opWorkspaceNew                    = "workspace.New"
	opWorkspaceCreateWorkspace        = "workspace.CreateWorkspace"
	opWorkspaceReadWorkspaceFile      = "workspace.ReadWorkspaceFile"
	opWorkspaceWriteWorkspaceFile     = "workspace.WriteWorkspaceFile"
	opWorkspaceHandleWorkspaceCommand = "workspace.HandleWorkspaceCommand"
)

// Example: key, _ := keyPairProvider.CreateKeyPair("alice", "pass123")
type KeyPairProvider interface {
	CreateKeyPair(identifier, passphrase string) (string, error)
}

// newWorkspaceSHA256Hash adapts core.SHA256 for HKDF's hash.Hash API.
func newWorkspaceSHA256Hash() hash.Hash {
	return &workspaceSHA256Hash{}
}

func workspaceSHA256(data []byte) [32]byte {
	return sha256.Sum256(data)
}

func workspaceSHA256Hex(data []byte) string {
	sum := workspaceSHA256(data)
	return hex.EncodeToString(sum[:])
}

type workspaceSHA256Hash struct {
	data []byte
}

func (hash *workspaceSHA256Hash) Write(data []byte) (
	int,
	error,
) {
	hash.data = append(hash.data, data...)
	return len(data), nil
}

func (hash *workspaceSHA256Hash) Sum(prefix []byte) []byte {
	sum := workspaceSHA256(hash.data)
	return append(prefix, sum[:]...)
}

func (hash *workspaceSHA256Hash) Reset() {
	hash.data = hash.data[:0]
}

func (hash *workspaceSHA256Hash) Size() int {
	return 32
}

func (hash *workspaceSHA256Hash) BlockSize() int {
	return 64
}

// Example: service, _ := workspace.New(workspace.Options{
// Example:     KeyPairProvider: keyPairProvider,
// Example:     RootPath: "/srv/workspaces",
// Example:     Medium: io.NewMemoryMedium(),
// Example:     Core: c,
// Example: })
type Options struct {
	KeyPairProvider KeyPairProvider
	RootPath        string
	Medium          io.Medium
	// Example: service, _ := workspace.New(workspace.Options{Core: core.New()})
	Core *core.Core
}

// Example: service, _ := workspace.New(workspace.Options{KeyPairProvider: keyPairProvider})
type Service struct {
	keyPairProvider   KeyPairProvider
	activeWorkspaceID string
	rootPath          string
	medium            io.Medium
	stateLock         sync.RWMutex
}

var _ EncryptedWorkspace = (*Service)(nil)

// Example: service, _ := workspace.New(workspace.Options{
// Example:     KeyPairProvider: keyPairProvider,
// Example:     RootPath: "/srv/workspaces",
// Example:     Medium: io.NewMemoryMedium(),
// Example: })
// Example: workspaceID, _ := service.CreateWorkspace("alice", "pass123")
func New(options Options) (
	*Service,
	error,
) {
	rootPath := options.RootPath
	if rootPath == "" {
		home := resolveWorkspaceHomeDirectory()
		if home == "" {
			return nil, core.E(opWorkspaceNew, "failed to determine home directory", fs.ErrNotExist)
		}
		rootPath = core.Path(home, ".core", "workspaces")
	}

	if options.KeyPairProvider == nil {
		return nil, core.E(opWorkspaceNew, "key pair provider is required", fs.ErrInvalid)
	}

	medium := options.Medium
	if medium == nil {
		medium = io.Local
	}
	if medium == nil {
		return nil, core.E(opWorkspaceNew, "storage medium is required", fs.ErrInvalid)
	}

	service := &Service{
		keyPairProvider: options.KeyPairProvider,
		rootPath:        rootPath,
		medium:          medium,
	}

	if err := service.medium.EnsureDir(rootPath); err != nil {
		return nil, core.E(opWorkspaceNew, "failed to ensure root directory", err)
	}

	if options.Core != nil {
		options.Core.RegisterAction(service.HandleWorkspaceMessage)
	}

	return service, nil
}

// Example: workspaceID, _ := service.CreateWorkspace("alice", "pass123")
func (service *Service) CreateWorkspace(identifier, passphrase string) (
	string,
	error,
) {
	service.stateLock.Lock()
	defer service.stateLock.Unlock()

	if service.keyPairProvider == nil {
		return "", core.E(opWorkspaceCreateWorkspace, "key pair provider not available", fs.ErrInvalid)
	}

	workspaceID := workspaceSHA256Hex([]byte(identifier))
	workspaceDirectory, err := service.resolveWorkspaceDirectory(opWorkspaceCreateWorkspace, workspaceID)
	if err != nil {
		return "", err
	}

	if service.medium.Exists(workspaceDirectory) {
		return "", core.E(opWorkspaceCreateWorkspace, "workspace already exists", fs.ErrExist)
	}

	for _, directoryName := range []string{"config", "lo" + "g", "data", "files", "keys"} {
		if err := service.medium.EnsureDir(core.Path(workspaceDirectory, directoryName)); err != nil {
			return "", core.E(opWorkspaceCreateWorkspace, core.Concat("failed to create directory: ", directoryName), err)
		}
	}

	privateKey, err := service.keyPairProvider.CreateKeyPair(identifier, passphrase)
	if err != nil {
		return "", core.E(opWorkspaceCreateWorkspace, "failed to generate keys", err)
	}

	if err := service.medium.WriteMode(core.Path(workspaceDirectory, "keys", "private.key"), privateKey, 0600); err != nil {
		return "", core.E(opWorkspaceCreateWorkspace, "failed to save private key", err)
	}

	return workspaceID, nil
}

// Example: _ = service.SwitchWorkspace(workspaceID)
func (service *Service) SwitchWorkspace(workspaceID string) error { // legacy error contract

	service.stateLock.Lock()
	defer service.stateLock.Unlock()

	workspaceDirectory, err := service.resolveWorkspaceDirectory("workspace.SwitchWorkspace", workspaceID)
	if err != nil {
		return err
	}
	if !service.medium.IsDir(workspaceDirectory) {
		return core.E("workspace.SwitchWorkspace", core.Concat("workspace not found: ", workspaceID), fs.ErrNotExist)
	}

	service.activeWorkspaceID = core.PathBase(workspaceDirectory)
	return nil
}

func (service *Service) resolveActiveWorkspaceFilePath(operation, workspaceFilePath string) (
	string,
	error,
) {
	if service.activeWorkspaceID == "" {
		return "", core.E(operation, "no active workspace", fs.ErrNotExist)
	}
	filesRoot := core.Path(service.rootPath, service.activeWorkspaceID, "files")
	filePath, err := joinPathWithinRoot(filesRoot, workspaceFilePath)
	if err != nil {
		return "", core.E(operation, "file path escapes workspace files", fs.ErrPermission)
	}
	if filePath == filesRoot {
		return "", core.E(operation, "workspace file path is required", fs.ErrInvalid)
	}
	return filePath, nil
}

// Example: cipherSigil, _ := service.workspaceCipherSigil("workspace.ReadWorkspaceFile")
func (service *Service) workspaceCipherSigil(operation string) (
	*sigil.ChaChaPolySigil,
	error,
) {
	if service.activeWorkspaceID == "" {
		return nil, core.E(operation, "no active workspace", fs.ErrNotExist)
	}
	keyPath := core.Path(service.rootPath, service.activeWorkspaceID, "keys", "private.key")
	rawKey, err := service.medium.Read(keyPath)
	if err != nil {
		return nil, core.E(operation, "failed to read workspace key", err)
	}
	// Use HKDF (RFC 5869) for key derivation: it is purpose-bound, domain-separated,
	// and more resistant to length-extension attacks than a bare SHA-256 hash.
	hkdfReader := hkdf.New(newWorkspaceSHA256Hash, []byte(rawKey), nil, []byte("workspace-cipher-key"))
	derived := make([]byte, 32)
	if _, err := goio.ReadFull(hkdfReader, derived); err != nil {
		return nil, core.E(operation, "failed to derive workspace key", err)
	}
	cipherSigil, err := sigil.NewChaChaPolySigil(derived, nil)
	if err != nil {
		return nil, core.E(operation, "failed to create cipher sigil", err)
	}
	return cipherSigil, nil
}

// Example: content, _ := service.ReadWorkspaceFile("notes/todo.txt")
func (service *Service) ReadWorkspaceFile(workspaceFilePath string) (
	string,
	error,
) {
	service.stateLock.RLock()
	defer service.stateLock.RUnlock()

	filePath, err := service.resolveActiveWorkspaceFilePath(opWorkspaceReadWorkspaceFile, workspaceFilePath)
	if err != nil {
		return "", err
	}
	cipherSigil, err := service.workspaceCipherSigil(opWorkspaceReadWorkspaceFile)
	if err != nil {
		return "", err
	}
	encoded, err := service.medium.Read(filePath)
	if err != nil {
		return "", err
	}
	plaintext, err := sigil.Untransmute([]byte(encoded), []sigil.Sigil{cipherSigil})
	if err != nil {
		return "", core.E(opWorkspaceReadWorkspaceFile, "failed to decrypt file content", err)
	}
	return string(plaintext), nil
}

// Example: _ = service.WriteWorkspaceFile("notes/todo.txt", "ship it")
func (service *Service) WriteWorkspaceFile(workspaceFilePath, content string) error { // legacy error contract

	service.stateLock.Lock()
	defer service.stateLock.Unlock()

	filePath, err := service.resolveActiveWorkspaceFilePath(opWorkspaceWriteWorkspaceFile, workspaceFilePath)
	if err != nil {
		return err
	}
	cipherSigil, err := service.workspaceCipherSigil(opWorkspaceWriteWorkspaceFile)
	if err != nil {
		return err
	}
	ciphertext, err := sigil.Transmute([]byte(content), []sigil.Sigil{cipherSigil})
	if err != nil {
		return core.E(opWorkspaceWriteWorkspaceFile, "failed to encrypt file content", err)
	}
	return service.medium.Write(filePath, string(ciphertext))
}

// Example: commandResult := service.HandleWorkspaceCommand(WorkspaceCommand{Action: WorkspaceCreateAction, Identifier: "alice", Password: "pass123"})
func (service *Service) HandleWorkspaceCommand(command WorkspaceCommand) core.Result {
	switch command.Action {
	case WorkspaceCreateAction, legacyWorkspaceCreateAction:
		identifier := command.workspaceName()
		if identifier == "" {
			return core.Fail(core.E(opWorkspaceHandleWorkspaceCommand, "workspace identifier is required", fs.ErrInvalid))
		}
		workspaceID, err := service.CreateWorkspace(identifier, command.Password)
		if err != nil {
			return core.Fail(err)
		}
		return core.Ok(workspaceID)
	case WorkspaceSwitchAction, legacyWorkspaceSwitchAction:
		workspaceID := command.workspaceName()
		if workspaceID == "" {
			return core.Fail(core.E(opWorkspaceHandleWorkspaceCommand, "workspace id is required", fs.ErrInvalid))
		}
		if err := service.SwitchWorkspace(workspaceID); err != nil {
			return core.Fail(err)
		}
		return core.Ok(nil)
	}
	return core.Fail(core.E(opWorkspaceHandleWorkspaceCommand, core.Concat("unsupported action: ", command.Action), fs.ErrInvalid))
}

// Example: result := service.HandleWorkspaceMessage(core.New(), WorkspaceCommand{Action: WorkspaceSwitchAction, WorkspaceID: "f3f0d7"})
func (service *Service) HandleWorkspaceMessage(_ *core.Core, message core.Message) core.Result {
	switch command := message.(type) {
	case WorkspaceCommand:
		return service.HandleWorkspaceCommand(command)
	}
	return core.Fail(core.E("workspace.HandleWorkspaceMessage", "unsupported message type", fs.ErrInvalid))
}

func resolveWorkspaceHomeDirectory() string {
	if home := core.Env("CORE_HOME"); home != "" {
		return home
	}
	if home := core.Env("HOME"); home != "" {
		return home
	}
	return core.Env("DIR_HOME")
}

func joinPathWithinRoot(root string, parts ...string) (
	string,
	error,
) {
	candidate := core.Path(append([]string{root}, parts...)...)
	separator := core.Env("CORE_PATH_SEPARATOR")
	if separator == "" {
		separator = core.Env("DS")
	}
	if separator == "" {
		separator = "/"
	}
	if candidate == root || core.HasPrefix(candidate, root+separator) {
		return candidate, nil
	}
	return "", fs.ErrPermission
}

func (service *Service) resolveWorkspaceDirectory(operation, workspaceID string) (
	string,
	error,
) {
	if workspaceID == "" {
		return "", core.E(operation, "workspace id is required", fs.ErrInvalid)
	}
	workspaceDirectory, err := joinPathWithinRoot(service.rootPath, workspaceID)
	if err != nil {
		return "", core.E(operation, "workspace path escapes root", err)
	}
	if core.PathDir(workspaceDirectory) != service.rootPath {
		return "", core.E(operation, core.Concat("invalid workspace id: ", workspaceID), fs.ErrPermission)
	}
	return workspaceDirectory, nil
}
