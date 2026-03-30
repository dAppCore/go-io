package workspace

import (
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"sync"

	core "dappco.re/go/core"

	"dappco.re/go/core/io"
)

// Workspace provides management for encrypted user workspaces.
type Workspace interface {
	CreateWorkspace(identifier, password string) (string, error)
	SwitchWorkspace(name string) error
	WorkspaceFileGet(filename string) (string, error)
	WorkspaceFileSet(filename, content string) error
}

// CryptProvider is the interface for PGP key generation.
type CryptProvider interface {
	CreateKeyPair(name, passphrase string) (string, error)
}

// Options configures the workspace service.
type Options struct {
	// Core is the Core runtime used by the service.
	Core *core.Core
	// Crypt is the PGP key generation dependency.
	Crypt CryptProvider
}

// Service implements the Workspace interface.
type Service struct {
	core            *core.Core
	crypt           CryptProvider
	activeWorkspace string
	rootPath        string
	medium          io.Medium
	mu              sync.RWMutex
}

var _ Workspace = (*Service)(nil)

// New creates a new Workspace service instance.
//
// Example usage:
//
//	service, _ := workspace.New(workspace.Options{Core: core.New(), Crypt: myCryptProvider})
func New(options Options) (*Service, error) {
	home := workspaceHome()
	if home == "" {
		return nil, core.E("workspace.New", "failed to determine home directory", fs.ErrNotExist)
	}
	rootPath := core.Path(home, ".core", "workspaces")

	if options.Core == nil {
		return nil, core.E("workspace.New", "core is required", fs.ErrInvalid)
	}

	s := &Service{
		core:     options.Core,
		rootPath: rootPath,
		medium:   io.Local,
	}

	if options.Crypt != nil {
		s.crypt = options.Crypt
	}

	if err := s.medium.EnsureDir(rootPath); err != nil {
		return nil, core.E("workspace.New", "failed to ensure root directory", err)
	}

	return s, nil
}

// Example: workspaceID, _ := service.CreateWorkspace("alice", "pass123")
// Identifier is hashed (SHA-256) to create the directory name.
// A PGP keypair is generated using the password.
func (s *Service) CreateWorkspace(identifier, password string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.crypt == nil {
		return "", core.E("workspace.CreateWorkspace", "crypt service not available", nil)
	}

	hash := sha256.Sum256([]byte(identifier))
	workspaceID := hex.EncodeToString(hash[:])
	workspaceDirectory, err := s.workspacePath("workspace.CreateWorkspace", workspaceID)
	if err != nil {
		return "", err
	}

	if s.medium.Exists(workspaceDirectory) {
		return "", core.E("workspace.CreateWorkspace", "workspace already exists", nil)
	}

	for _, d := range []string{"config", "log", "data", "files", "keys"} {
		if err := s.medium.EnsureDir(core.Path(workspaceDirectory, d)); err != nil {
			return "", core.E("workspace.CreateWorkspace", core.Concat("failed to create directory: ", d), err)
		}
	}

	privKey, err := s.crypt.CreateKeyPair(identifier, password)
	if err != nil {
		return "", core.E("workspace.CreateWorkspace", "failed to generate keys", err)
	}

	if err := s.medium.WriteMode(core.Path(workspaceDirectory, "keys", "private.key"), privKey, 0600); err != nil {
		return "", core.E("workspace.CreateWorkspace", "failed to save private key", err)
	}

	return workspaceID, nil
}

// Example: _ = service.SwitchWorkspace(workspaceID)
func (s *Service) SwitchWorkspace(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	workspaceDirectory, err := s.workspacePath("workspace.SwitchWorkspace", name)
	if err != nil {
		return err
	}
	if !s.medium.IsDir(workspaceDirectory) {
		return core.E("workspace.SwitchWorkspace", core.Concat("workspace not found: ", name), nil)
	}

	s.activeWorkspace = core.PathBase(workspaceDirectory)
	return nil
}

// activeFilePath returns the full path to a file in the active workspace,
// or an error if no workspace is active.
func (s *Service) activeFilePath(operation, filename string) (string, error) {
	if s.activeWorkspace == "" {
		return "", core.E(operation, "no active workspace", nil)
	}
	filesRoot := core.Path(s.rootPath, s.activeWorkspace, "files")
	filePath, err := joinWithinRoot(filesRoot, filename)
	if err != nil {
		return "", core.E(operation, "file path escapes workspace files", fs.ErrPermission)
	}
	if filePath == filesRoot {
		return "", core.E(operation, "filename is required", fs.ErrInvalid)
	}
	return filePath, nil
}

// Example: content, _ := service.WorkspaceFileGet("notes/todo.txt")
func (s *Service) WorkspaceFileGet(filename string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	filePath, err := s.activeFilePath("workspace.WorkspaceFileGet", filename)
	if err != nil {
		return "", err
	}
	return s.medium.Read(filePath)
}

// Example: _ = service.WorkspaceFileSet("notes/todo.txt", "ship it")
func (s *Service) WorkspaceFileSet(filename, content string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	filePath, err := s.activeFilePath("workspace.WorkspaceFileSet", filename)
	if err != nil {
		return err
	}
	return s.medium.Write(filePath, content)
}

// HandleIPCEvents handles workspace-related IPC messages.
//
//	service, _ := workspace.New(workspace.Options{Core: core.New(), Crypt: myCryptProvider})
//	ipcResult := service.HandleIPCEvents(core.New(), map[string]any{
//		"action":     "workspace.create",
//		"identifier": "alice",
//		"password":   "pass123",
//	})
//	_ = ipcResult.OK
func (s *Service) HandleIPCEvents(_ *core.Core, message core.Message) core.Result {
	switch payload := message.(type) {
	case map[string]any:
		action, _ := payload["action"].(string)
		switch action {
		case "workspace.create":
			identifier, _ := payload["identifier"].(string)
			password, _ := payload["password"].(string)
			workspaceID, err := s.CreateWorkspace(identifier, password)
			if err != nil {
				return core.Result{}.New(err)
			}
			return core.Result{Value: workspaceID, OK: true}
		case "workspace.switch":
			name, _ := payload["name"].(string)
			if err := s.SwitchWorkspace(name); err != nil {
				return core.Result{}.New(err)
			}
			return core.Result{OK: true}
		}
	}
	return core.Result{OK: true}
}

func workspaceHome() string {
	if home := core.Env("CORE_HOME"); home != "" {
		return home
	}
	if home := core.Env("HOME"); home != "" {
		return home
	}
	return core.Env("DIR_HOME")
}

func joinWithinRoot(root string, parts ...string) (string, error) {
	candidate := core.Path(append([]string{root}, parts...)...)
	sep := core.Env("DS")
	if candidate == root || core.HasPrefix(candidate, root+sep) {
		return candidate, nil
	}
	return "", fs.ErrPermission
}

func (s *Service) workspacePath(operation, workspaceName string) (string, error) {
	if workspaceName == "" {
		return "", core.E(operation, "workspace name is required", fs.ErrInvalid)
	}
	workspaceDirectory, err := joinWithinRoot(s.rootPath, workspaceName)
	if err != nil {
		return "", core.E(operation, "workspace path escapes root", err)
	}
	if core.PathDir(workspaceDirectory) != s.rootPath {
		return "", core.E(operation, core.Concat("invalid workspace name: ", workspaceName), fs.ErrPermission)
	}
	return workspaceDirectory, nil
}
