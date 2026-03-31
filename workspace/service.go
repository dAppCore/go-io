package workspace

import (
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"sync"

	core "dappco.re/go/core"

	"dappco.re/go/core/io"
	"dappco.re/go/core/io/sigil"
)

// Example: service, _ := workspace.New(workspace.Options{
// Example:     KeyPairProvider: keyPairProvider,
// Example:     RootPath: "/srv/workspaces",
// Example:     Medium: io.NewMemoryMedium(),
// Example: })
type Workspace interface {
	CreateWorkspace(identifier, passphrase string) (string, error)
	SwitchWorkspace(workspaceID string) error
	ReadWorkspaceFile(workspaceFilePath string) (string, error)
	WriteWorkspaceFile(workspaceFilePath, content string) error
}

// Example: key, _ := keyPairProvider.CreateKeyPair("alice", "pass123")
type KeyPairProvider interface {
	CreateKeyPair(identifier, passphrase string) (string, error)
}

const (
	WorkspaceCreateAction = "workspace.create"
	WorkspaceSwitchAction = "workspace.switch"
)

// Example: command := WorkspaceCommand{Action: WorkspaceCreateAction, Identifier: "alice", Password: "pass123"}
type WorkspaceCommand struct {
	Action      string
	Identifier  string
	Password    string
	WorkspaceID string
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
	// Core is the optional Core instance. When set, the workspace service
	// auto-registers as an IPC listener for workspace.create and workspace.switch events.
	Core *core.Core
}

// Example: service, _ := workspace.New(workspace.Options{
// Example:     KeyPairProvider: keyPairProvider,
// Example:     RootPath: "/srv/workspaces",
// Example:     Medium: io.NewMemoryMedium(),
// Example: })
type Service struct {
	keyPairProvider   KeyPairProvider
	activeWorkspaceID string
	rootPath          string
	medium            io.Medium
	stateLock         sync.RWMutex
}

var _ Workspace = (*Service)(nil)

// Example: service, _ := workspace.New(workspace.Options{
// Example:     KeyPairProvider: keyPairProvider,
// Example:     RootPath: "/srv/workspaces",
// Example:     Medium: io.NewMemoryMedium(),
// Example: })
// Example: workspaceID, _ := service.CreateWorkspace("alice", "pass123")
func New(options Options) (*Service, error) {
	rootPath := options.RootPath
	if rootPath == "" {
		home := resolveWorkspaceHomeDirectory()
		if home == "" {
			return nil, core.E("workspace.New", "failed to determine home directory", fs.ErrNotExist)
		}
		rootPath = core.Path(home, ".core", "workspaces")
	}

	if options.KeyPairProvider == nil {
		return nil, core.E("workspace.New", "key pair provider is required", fs.ErrInvalid)
	}

	medium := options.Medium
	if medium == nil {
		medium = io.Local
	}
	if medium == nil {
		return nil, core.E("workspace.New", "storage medium is required", fs.ErrInvalid)
	}

	service := &Service{
		keyPairProvider: options.KeyPairProvider,
		rootPath:        rootPath,
		medium:          medium,
	}

	if err := service.medium.EnsureDir(rootPath); err != nil {
		return nil, core.E("workspace.New", "failed to ensure root directory", err)
	}

	if options.Core != nil {
		options.Core.RegisterAction(service.HandleWorkspaceMessage)
	}

	return service, nil
}

// Example: workspaceID, _ := service.CreateWorkspace("alice", "pass123")
func (service *Service) CreateWorkspace(identifier, passphrase string) (string, error) {
	service.stateLock.Lock()
	defer service.stateLock.Unlock()

	if service.keyPairProvider == nil {
		return "", core.E("workspace.CreateWorkspace", "key pair provider not available", fs.ErrInvalid)
	}

	hash := sha256.Sum256([]byte(identifier))
	workspaceID := hex.EncodeToString(hash[:])
	workspaceDirectory, err := service.resolveWorkspaceDirectory("workspace.CreateWorkspace", workspaceID)
	if err != nil {
		return "", err
	}

	if service.medium.Exists(workspaceDirectory) {
		return "", core.E("workspace.CreateWorkspace", "workspace already exists", fs.ErrExist)
	}

	for _, directoryName := range []string{"config", "log", "data", "files", "keys"} {
		if err := service.medium.EnsureDir(core.Path(workspaceDirectory, directoryName)); err != nil {
			return "", core.E("workspace.CreateWorkspace", core.Concat("failed to create directory: ", directoryName), err)
		}
	}

	privateKey, err := service.keyPairProvider.CreateKeyPair(identifier, passphrase)
	if err != nil {
		return "", core.E("workspace.CreateWorkspace", "failed to generate keys", err)
	}

	if err := service.medium.WriteMode(core.Path(workspaceDirectory, "keys", "private.key"), privateKey, 0600); err != nil {
		return "", core.E("workspace.CreateWorkspace", "failed to save private key", err)
	}

	return workspaceID, nil
}

// Example: _ = service.SwitchWorkspace(workspaceID)
func (service *Service) SwitchWorkspace(workspaceID string) error {
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

func (service *Service) resolveActiveWorkspaceFilePath(operation, workspaceFilePath string) (string, error) {
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
func (service *Service) workspaceCipherSigil(operation string) (*sigil.ChaChaPolySigil, error) {
	if service.activeWorkspaceID == "" {
		return nil, core.E(operation, "no active workspace", fs.ErrNotExist)
	}
	keyPath := core.Path(service.rootPath, service.activeWorkspaceID, "keys", "private.key")
	rawKey, err := service.medium.Read(keyPath)
	if err != nil {
		return nil, core.E(operation, "failed to read workspace key", err)
	}
	derived := sha256.Sum256([]byte(rawKey))
	cipherSigil, err := sigil.NewChaChaPolySigil(derived[:], nil)
	if err != nil {
		return nil, core.E(operation, "failed to create cipher sigil", err)
	}
	return cipherSigil, nil
}

// Example: content, _ := service.ReadWorkspaceFile("notes/todo.txt")
func (service *Service) ReadWorkspaceFile(workspaceFilePath string) (string, error) {
	service.stateLock.RLock()
	defer service.stateLock.RUnlock()

	filePath, err := service.resolveActiveWorkspaceFilePath("workspace.ReadWorkspaceFile", workspaceFilePath)
	if err != nil {
		return "", err
	}
	cipherSigil, err := service.workspaceCipherSigil("workspace.ReadWorkspaceFile")
	if err != nil {
		return "", err
	}
	encoded, err := service.medium.Read(filePath)
	if err != nil {
		return "", err
	}
	plaintext, err := sigil.Untransmute([]byte(encoded), []sigil.Sigil{cipherSigil})
	if err != nil {
		return "", core.E("workspace.ReadWorkspaceFile", "failed to decrypt file content", err)
	}
	return string(plaintext), nil
}

// Example: _ = service.WriteWorkspaceFile("notes/todo.txt", "ship it")
func (service *Service) WriteWorkspaceFile(workspaceFilePath, content string) error {
	service.stateLock.Lock()
	defer service.stateLock.Unlock()

	filePath, err := service.resolveActiveWorkspaceFilePath("workspace.WriteWorkspaceFile", workspaceFilePath)
	if err != nil {
		return err
	}
	cipherSigil, err := service.workspaceCipherSigil("workspace.WriteWorkspaceFile")
	if err != nil {
		return err
	}
	ciphertext, err := sigil.Transmute([]byte(content), []sigil.Sigil{cipherSigil})
	if err != nil {
		return core.E("workspace.WriteWorkspaceFile", "failed to encrypt file content", err)
	}
	return service.medium.Write(filePath, string(ciphertext))
}

// Example: commandResult := service.HandleWorkspaceCommand(WorkspaceCommand{Action: WorkspaceCreateAction, Identifier: "alice", Password: "pass123"})
func (service *Service) HandleWorkspaceCommand(command WorkspaceCommand) core.Result {
	switch command.Action {
	case WorkspaceCreateAction:
		passphrase := command.Password
		workspaceID, err := service.CreateWorkspace(command.Identifier, passphrase)
		if err != nil {
			return core.Result{}.New(err)
		}
		return core.Result{Value: workspaceID, OK: true}
	case WorkspaceSwitchAction:
		if err := service.SwitchWorkspace(command.WorkspaceID); err != nil {
			return core.Result{}.New(err)
		}
		return core.Result{OK: true}
	}
	return core.Result{}.New(core.E("workspace.HandleWorkspaceCommand", core.Concat("unsupported action: ", command.Action), fs.ErrInvalid))
}

// Example: result := service.HandleWorkspaceMessage(core.New(), WorkspaceCommand{Action: WorkspaceSwitchAction, WorkspaceID: "f3f0d7"})
func (service *Service) HandleWorkspaceMessage(_ *core.Core, message core.Message) core.Result {
	switch command := message.(type) {
	case WorkspaceCommand:
		return service.HandleWorkspaceCommand(command)
	}
	return core.Result{}.New(core.E("workspace.HandleWorkspaceMessage", "unsupported message type", fs.ErrInvalid))
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

func joinPathWithinRoot(root string, parts ...string) (string, error) {
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

func (service *Service) resolveWorkspaceDirectory(operation, workspaceID string) (string, error) {
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
