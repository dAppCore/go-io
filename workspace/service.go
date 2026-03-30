package workspace

import (
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"sync"

	core "dappco.re/go/core"

	"dappco.re/go/core/io"
)

// Example: service, _ := workspace.New(workspace.Options{Core: core.New(), Crypt: cryptProvider})
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

// Example: service, _ := workspace.New(workspace.Options{Core: core.New(), Crypt: cryptProvider})
type Options struct {
	// Core is the Core runtime used by the service.
	Core *core.Core
	// Crypt is the PGP key generation dependency.
	Crypt CryptProvider
}

// Example: service, _ := workspace.New(workspace.Options{Core: core.New(), Crypt: cryptProvider})
type Service struct {
	core              *core.Core
	crypt             CryptProvider
	activeWorkspaceID string
	rootPath          string
	medium            io.Medium
	mu                sync.RWMutex
}

var _ Workspace = (*Service)(nil)

// Example: service, _ := workspace.New(workspace.Options{Core: core.New(), Crypt: cryptProvider})
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
func (s *Service) CreateWorkspace(identifier, password string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.crypt == nil {
		return "", core.E("workspace.CreateWorkspace", "crypt service not available", nil)
	}

	hash := sha256.Sum256([]byte(identifier))
	workspaceID := hex.EncodeToString(hash[:])
	workspaceDirectory, err := s.resolveWorkspaceDirectory("workspace.CreateWorkspace", workspaceID)
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
func (s *Service) SwitchWorkspace(workspaceID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	workspaceDirectory, err := s.resolveWorkspaceDirectory("workspace.SwitchWorkspace", workspaceID)
	if err != nil {
		return err
	}
	if !s.medium.IsDir(workspaceDirectory) {
		return core.E("workspace.SwitchWorkspace", core.Concat("workspace not found: ", workspaceID), nil)
	}

	s.activeWorkspaceID = core.PathBase(workspaceDirectory)
	return nil
}

// resolveActiveWorkspaceFilePath resolves a file path inside the active workspace files root.
// It rejects empty names and traversal outside the workspace root.
func (s *Service) resolveActiveWorkspaceFilePath(operation, workspaceFilePath string) (string, error) {
	if s.activeWorkspaceID == "" {
		return "", core.E(operation, "no active workspace", nil)
	}
	filesRoot := core.Path(s.rootPath, s.activeWorkspaceID, "files")
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
func (s *Service) WorkspaceFileGet(workspaceFilePath string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	filePath, err := s.resolveActiveWorkspaceFilePath("workspace.WorkspaceFileGet", workspaceFilePath)
	if err != nil {
		return "", err
	}
	return s.medium.Read(filePath)
}

// Example: _ = service.WorkspaceFileSet("notes/todo.txt", "ship it")
func (s *Service) WorkspaceFileSet(workspaceFilePath, content string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	filePath, err := s.resolveActiveWorkspaceFilePath("workspace.WorkspaceFileSet", workspaceFilePath)
	if err != nil {
		return err
	}
	return s.medium.Write(filePath, content)
}

// service, _ := workspace.New(workspace.Options{Core: core.New(), Crypt: myCryptProvider})
//
//	createResult := service.HandleIPCEvents(core.New(), map[string]any{
//		"action":     "workspace.create",
//		"identifier": "alice",
//		"password":   "pass123",
//	})
//
//	switchResult := service.HandleIPCEvents(core.New(), map[string]any{
//		"action":      "workspace.switch",
//		"workspaceID": "f3f0d7",
//	})
//
// _ = createResult.OK
// _ = switchResult.OK
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
			workspaceID, _ := payload["workspaceID"].(string)
			if err := s.SwitchWorkspace(workspaceID); err != nil {
				return core.Result{}.New(err)
			}
			return core.Result{OK: true}
		}
	}
	return core.Result{OK: true}
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

func (s *Service) resolveWorkspaceDirectory(operation, workspaceID string) (string, error) {
	if workspaceID == "" {
		return "", core.E(operation, "workspace id is required", fs.ErrInvalid)
	}
	workspaceDirectory, err := joinPathWithinRoot(s.rootPath, workspaceID)
	if err != nil {
		return "", core.E(operation, "workspace path escapes root", err)
	}
	if core.PathDir(workspaceDirectory) != s.rootPath {
		return "", core.E(operation, core.Concat("invalid workspace id: ", workspaceID), fs.ErrPermission)
	}
	return workspaceDirectory, nil
}
