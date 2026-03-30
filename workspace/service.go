package workspace

import (
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"sync"

	core "dappco.re/go/core"

	"dappco.re/go/core/io"
)

// Example: service, _ := workspace.New(workspace.Options{Core: core.New(), CryptProvider: cryptProvider})
type Workspace interface {
	CreateWorkspace(identifier, password string) (string, error)
	SwitchWorkspace(workspaceID string) error
	WorkspaceFileGet(workspaceFilePath string) (string, error)
	WorkspaceFileSet(workspaceFilePath, content string) error
}

// CryptProvider generates the encrypted private key stored with each workspace.
type CryptProvider interface {
	CreateKeyPair(name, passphrase string) (string, error)
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

// Example: service, _ := workspace.New(workspace.Options{Core: core.New(), CryptProvider: cryptProvider})
type Options struct {
	Core          *core.Core
	CryptProvider CryptProvider
}

// Example: service, _ := workspace.New(workspace.Options{Core: core.New(), CryptProvider: cryptProvider})
type Service struct {
	cryptProvider     CryptProvider
	activeWorkspaceID string
	rootPath          string
	medium            io.Medium
	stateLock         sync.RWMutex
}

var _ Workspace = (*Service)(nil)

// Example: service, _ := workspace.New(workspace.Options{Core: core.New(), CryptProvider: cryptProvider})
// workspaceID, _ := service.CreateWorkspace("alice", "pass123")
func New(options Options) (*Service, error) {
	home := resolveWorkspaceHomeDirectory()
	if home == "" {
		return nil, core.E("workspace.New", "failed to determine home directory", fs.ErrNotExist)
	}
	rootPath := core.Path(home, ".core", "workspaces")

	if options.Core == nil {
		return nil, core.E("workspace.New", "core is required", fs.ErrInvalid)
	}
	if options.CryptProvider == nil {
		return nil, core.E("workspace.New", "crypt provider is required", fs.ErrInvalid)
	}

	service := &Service{
		cryptProvider: options.CryptProvider,
		rootPath:      rootPath,
		medium:        io.Local,
	}

	if err := service.medium.EnsureDir(rootPath); err != nil {
		return nil, core.E("workspace.New", "failed to ensure root directory", err)
	}

	return service, nil
}

// Example: workspaceID, _ := service.CreateWorkspace("alice", "pass123")
func (service *Service) CreateWorkspace(identifier, password string) (string, error) {
	service.stateLock.Lock()
	defer service.stateLock.Unlock()

	if service.cryptProvider == nil {
		return "", core.E("workspace.CreateWorkspace", "crypt provider not available", nil)
	}

	hash := sha256.Sum256([]byte(identifier))
	workspaceID := hex.EncodeToString(hash[:])
	workspaceDirectory, err := service.resolveWorkspaceDirectory("workspace.CreateWorkspace", workspaceID)
	if err != nil {
		return "", err
	}

	if service.medium.Exists(workspaceDirectory) {
		return "", core.E("workspace.CreateWorkspace", "workspace already exists", nil)
	}

	for _, d := range []string{"config", "log", "data", "files", "keys"} {
		if err := service.medium.EnsureDir(core.Path(workspaceDirectory, d)); err != nil {
			return "", core.E("workspace.CreateWorkspace", core.Concat("failed to create directory: ", d), err)
		}
	}

	privKey, err := service.cryptProvider.CreateKeyPair(identifier, password)
	if err != nil {
		return "", core.E("workspace.CreateWorkspace", "failed to generate keys", err)
	}

	if err := service.medium.WriteMode(core.Path(workspaceDirectory, "keys", "private.key"), privKey, 0600); err != nil {
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
		return core.E("workspace.SwitchWorkspace", core.Concat("workspace not found: ", workspaceID), nil)
	}

	service.activeWorkspaceID = core.PathBase(workspaceDirectory)
	return nil
}

func (service *Service) resolveActiveWorkspaceFilePath(operation, workspaceFilePath string) (string, error) {
	if service.activeWorkspaceID == "" {
		return "", core.E(operation, "no active workspace", nil)
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

// Example: content, _ := service.WorkspaceFileGet("notes/todo.txt")
func (service *Service) WorkspaceFileGet(workspaceFilePath string) (string, error) {
	service.stateLock.RLock()
	defer service.stateLock.RUnlock()

	filePath, err := service.resolveActiveWorkspaceFilePath("workspace.WorkspaceFileGet", workspaceFilePath)
	if err != nil {
		return "", err
	}
	return service.medium.Read(filePath)
}

// Example: _ = service.WorkspaceFileSet("notes/todo.txt", "ship it")
func (service *Service) WorkspaceFileSet(workspaceFilePath, content string) error {
	service.stateLock.Lock()
	defer service.stateLock.Unlock()

	filePath, err := service.resolveActiveWorkspaceFilePath("workspace.WorkspaceFileSet", workspaceFilePath)
	if err != nil {
		return err
	}
	return service.medium.Write(filePath, content)
}

// Example: result := service.HandleWorkspaceCommand(WorkspaceCommand{Action: WorkspaceCreateAction, Identifier: "alice", Password: "pass123"})
func (service *Service) HandleWorkspaceCommand(command WorkspaceCommand) core.Result {
	switch command.Action {
	case WorkspaceCreateAction:
		workspaceID, err := service.CreateWorkspace(command.Identifier, command.Password)
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
	return core.Result{OK: true}
}

// Example: result := service.HandleWorkspaceMessage(core.New(), WorkspaceCommand{Action: WorkspaceSwitchAction, WorkspaceID: "f3f0d7"})
// Example: legacy := service.HandleWorkspaceMessage(core.New(), map[string]any{"action": WorkspaceCreateAction, "identifier": "alice", "password": "pass123"})
func (service *Service) HandleWorkspaceMessage(_ *core.Core, message core.Message) core.Result {
	command, ok := workspaceCommandFromMessage(message)
	if !ok {
		return core.Result{OK: true}
	}
	return service.HandleWorkspaceCommand(command)
}

// Example: result := service.HandleIPCEvents(core.New(), WorkspaceCommand{Action: WorkspaceSwitchAction, WorkspaceID: "f3f0d7"})
func (service *Service) HandleIPCEvents(coreRuntime *core.Core, message core.Message) core.Result {
	return service.HandleWorkspaceMessage(coreRuntime, message)
}

func workspaceCommandFromMessage(message core.Message) (WorkspaceCommand, bool) {
	switch payload := message.(type) {
	case WorkspaceCommand:
		return payload, true
	case map[string]any:
		command := WorkspaceCommand{}
		command.Action, _ = payload["action"].(string)
		command.Identifier, _ = payload["identifier"].(string)
		command.Password, _ = payload["password"].(string)
		command.WorkspaceID, _ = payload["workspaceID"].(string)
		return command, true
	}
	return WorkspaceCommand{}, false
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
	sep := core.Env("DS")
	if candidate == root || core.HasPrefix(candidate, root+sep) {
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
