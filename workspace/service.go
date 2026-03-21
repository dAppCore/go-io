package workspace

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"sync"

	core "dappco.re/go/core"
	coreerr "forge.lthn.ai/core/go-log"

	"forge.lthn.ai/core/go-io"
)

// Workspace provides management for encrypted user workspaces.
type Workspace interface {
	CreateWorkspace(identifier, password string) (string, error)
	SwitchWorkspace(name string) error
	WorkspaceFileGet(filename string) (string, error)
	WorkspaceFileSet(filename, content string) error
}

// cryptProvider is the interface for PGP key generation.
type cryptProvider interface {
	CreateKeyPair(name, passphrase string) (string, error)
}

// Service implements the Workspace interface.
type Service struct {
	core            *core.Core
	crypt           cryptProvider
	activeWorkspace string
	rootPath        string
	medium          io.Medium
	mu              sync.RWMutex
}

// New creates a new Workspace service instance.
// An optional cryptProvider can be passed to supply PGP key generation.
func New(c *core.Core, crypt ...cryptProvider) (any, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, coreerr.E("workspace.New", "failed to determine home directory", err)
	}
	rootPath := filepath.Join(home, ".core", "workspaces")

	s := &Service{
		core:     c,
		rootPath: rootPath,
		medium:   io.Local,
	}

	if len(crypt) > 0 && crypt[0] != nil {
		s.crypt = crypt[0]
	}

	if err := s.medium.EnsureDir(rootPath); err != nil {
		return nil, coreerr.E("workspace.New", "failed to ensure root directory", err)
	}

	return s, nil
}

// CreateWorkspace creates a new encrypted workspace.
// Identifier is hashed (SHA-256) to create the directory name.
// A PGP keypair is generated using the password.
func (s *Service) CreateWorkspace(identifier, password string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.crypt == nil {
		return "", coreerr.E("workspace.CreateWorkspace", "crypt service not available", nil)
	}

	hash := sha256.Sum256([]byte(identifier))
	wsID := hex.EncodeToString(hash[:])
	wsPath := filepath.Join(s.rootPath, wsID)

	if s.medium.Exists(wsPath) {
		return "", coreerr.E("workspace.CreateWorkspace", "workspace already exists", nil)
	}

	for _, d := range []string{"config", "log", "data", "files", "keys"} {
		if err := s.medium.EnsureDir(filepath.Join(wsPath, d)); err != nil {
			return "", coreerr.E("workspace.CreateWorkspace", "failed to create directory: "+d, err)
		}
	}

	privKey, err := s.crypt.CreateKeyPair(identifier, password)
	if err != nil {
		return "", coreerr.E("workspace.CreateWorkspace", "failed to generate keys", err)
	}

	if err := s.medium.WriteMode(filepath.Join(wsPath, "keys", "private.key"), privKey, 0600); err != nil {
		return "", coreerr.E("workspace.CreateWorkspace", "failed to save private key", err)
	}

	return wsID, nil
}

// SwitchWorkspace changes the active workspace.
func (s *Service) SwitchWorkspace(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	wsPath := filepath.Join(s.rootPath, name)
	if !s.medium.IsDir(wsPath) {
		return coreerr.E("workspace.SwitchWorkspace", "workspace not found: "+name, nil)
	}

	s.activeWorkspace = name
	return nil
}

// activeFilePath returns the full path to a file in the active workspace,
// or an error if no workspace is active.
func (s *Service) activeFilePath(op, filename string) (string, error) {
	if s.activeWorkspace == "" {
		return "", coreerr.E(op, "no active workspace", nil)
	}
	return filepath.Join(s.rootPath, s.activeWorkspace, "files", filename), nil
}

// WorkspaceFileGet retrieves the content of a file from the active workspace.
func (s *Service) WorkspaceFileGet(filename string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	path, err := s.activeFilePath("workspace.WorkspaceFileGet", filename)
	if err != nil {
		return "", err
	}
	return s.medium.Read(path)
}

// WorkspaceFileSet saves content to a file in the active workspace.
func (s *Service) WorkspaceFileSet(filename, content string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	path, err := s.activeFilePath("workspace.WorkspaceFileSet", filename)
	if err != nil {
		return err
	}
	return s.medium.Write(path, content)
}

// HandleIPCEvents handles workspace-related IPC messages.
func (s *Service) HandleIPCEvents(c *core.Core, msg core.Message) core.Result {
	switch m := msg.(type) {
	case map[string]any:
		action, _ := m["action"].(string)
		switch action {
		case "workspace.create":
			id, _ := m["identifier"].(string)
			pass, _ := m["password"].(string)
			wsID, err := s.CreateWorkspace(id, pass)
			if err != nil {
				return core.Result{}
			}
			return core.Result{Value: wsID, OK: true}
		case "workspace.switch":
			name, _ := m["name"].(string)
			if err := s.SwitchWorkspace(name); err != nil {
				return core.Result{}
			}
			return core.Result{OK: true}
		}
	}
	return core.Result{OK: true}
}

// Ensure Service implements Workspace.
var _ Workspace = (*Service)(nil)
