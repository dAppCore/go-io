package workspace

import (
	goio "io"
	"io/fs"
	"path"
	"strings"
	"sync"

	core "dappco.re/go"
	coreio "dappco.re/go/io"
)

const (
	opNewWorkspace    = "workspace.NewWorkspace"
	opCreateWorkspace = "workspace.CreateWorkspace"
	opSwitchWorkspace = "workspace.SwitchWorkspace"

	errWorkspaceServiceNotConfigured = "workspace service is not configured"
)

// Workspace is the RFC §5 medium-backed workspace service.
type Workspace struct {
	medium  coreio.Medium
	base    string
	current string
	mu      sync.RWMutex
}

// NewWorkspace creates a workspace service backed by medium under baseSubpath.
func NewWorkspace(medium coreio.Medium, baseSubpath string) (*Workspace, error) {
	if medium == nil {
		return nil, core.E(opNewWorkspace, "storage medium is required", fs.ErrInvalid)
	}
	base, err := cleanMediumSubpath(opNewWorkspace, baseSubpath, true)
	if err != nil {
		return nil, err
	}
	if base != "" {
		if err := medium.EnsureDir(base); err != nil {
			return nil, core.E(opNewWorkspace, "failed to ensure base workspace directory", err)
		}
	}
	return &Workspace{
		medium: medium,
		base:   base,
	}, nil
}

// CreateWorkspace creates a named workspace directory and returns a medium scoped to it.
func (workspace *Workspace) CreateWorkspace(name string) (coreio.Medium, error) {
	if workspace == nil {
		return nil, core.E(opCreateWorkspace, errWorkspaceServiceNotConfigured, fs.ErrInvalid)
	}
	workspace.mu.Lock()
	defer workspace.mu.Unlock()

	workspacePath, err := workspace.workspacePath(opCreateWorkspace, name)
	if err != nil {
		return nil, err
	}
	if workspace.medium.IsDir(workspacePath) {
		return nil, core.E(opCreateWorkspace, core.Concat("workspace already exists: ", name), fs.ErrExist)
	}
	if workspace.medium.Exists(workspacePath) {
		return nil, core.E(opCreateWorkspace, core.Concat("workspace path is not a directory: ", name), fs.ErrExist)
	}
	if err := workspace.medium.EnsureDir(workspacePath); err != nil {
		return nil, core.E(opCreateWorkspace, "failed to create workspace directory", err)
	}
	return workspace.scopedMedium(workspacePath), nil
}

// SwitchWorkspace records the named workspace as the current workspace.
func (workspace *Workspace) SwitchWorkspace(name string) error {
	if workspace == nil {
		return core.E(opSwitchWorkspace, errWorkspaceServiceNotConfigured, fs.ErrInvalid)
	}
	workspace.mu.Lock()
	defer workspace.mu.Unlock()

	workspaceName, workspacePath, err := workspace.workspaceNameAndPath(opSwitchWorkspace, name)
	if err != nil {
		return err
	}
	if !workspace.medium.IsDir(workspacePath) {
		return core.E(opSwitchWorkspace, core.Concat("workspace not found: ", workspaceName), fs.ErrNotExist)
	}
	workspace.current = workspaceName
	return nil
}

// CurrentWorkspace returns the workspace selected by SwitchWorkspace.
func (workspace *Workspace) CurrentWorkspace() string {
	if workspace == nil {
		return ""
	}
	workspace.mu.RLock()
	defer workspace.mu.RUnlock()
	return workspace.current
}

// ReadWorkspaceFile reads a file from the named workspace.
func (workspace *Workspace) ReadWorkspaceFile(name, filePath string) (string, error) {
	if workspace == nil {
		return "", core.E("workspace.ReadWorkspaceFile", errWorkspaceServiceNotConfigured, fs.ErrInvalid)
	}
	workspace.mu.RLock()
	defer workspace.mu.RUnlock()

	mediumPath, err := workspace.workspaceFilePath("workspace.ReadWorkspaceFile", name, filePath, false)
	if err != nil {
		return "", err
	}
	return workspace.medium.Read(mediumPath)
}

// WriteWorkspaceFile writes a file into the named workspace.
func (workspace *Workspace) WriteWorkspaceFile(name, filePath, content string) error {
	if workspace == nil {
		return core.E("workspace.WriteWorkspaceFile", errWorkspaceServiceNotConfigured, fs.ErrInvalid)
	}
	workspace.mu.Lock()
	defer workspace.mu.Unlock()

	mediumPath, err := workspace.workspaceFilePath("workspace.WriteWorkspaceFile", name, filePath, false)
	if err != nil {
		return err
	}
	return workspace.medium.Write(mediumPath, content)
}

// ListWorkspaceFiles lists entries under a workspace-relative directory.
func (workspace *Workspace) ListWorkspaceFiles(name, directoryPath string) ([]fs.DirEntry, error) {
	if workspace == nil {
		return nil, core.E("workspace.ListWorkspaceFiles", errWorkspaceServiceNotConfigured, fs.ErrInvalid)
	}
	workspace.mu.RLock()
	defer workspace.mu.RUnlock()

	mediumPath, err := workspace.workspaceFilePath("workspace.ListWorkspaceFiles", name, directoryPath, true)
	if err != nil {
		return nil, err
	}
	return workspace.medium.List(mediumPath)
}

// HandleWorkspaceCommand dispatches an RFC §5 WorkspaceCommand.
func (workspace *Workspace) HandleWorkspaceCommand(command WorkspaceCommand) core.Result {
	switch strings.ToLower(strings.TrimSpace(command.Action)) {
	case WorkspaceCreateAction, legacyWorkspaceCreateAction:
		medium, err := workspace.CreateWorkspace(command.workspaceName())
		if err != nil {
			return core.Fail(err)
		}
		return core.Ok(medium)
	case WorkspaceSwitchAction, legacyWorkspaceSwitchAction:
		if err := workspace.SwitchWorkspace(command.workspaceName()); err != nil {
			return core.Fail(err)
		}
		return core.Ok(nil)
	case WorkspaceReadAction:
		content, err := workspace.ReadWorkspaceFile(command.workspaceName(), command.Path)
		if err != nil {
			return core.Fail(err)
		}
		return core.Ok(content)
	case WorkspaceWriteAction:
		if err := workspace.WriteWorkspaceFile(command.workspaceName(), command.Path, command.Content); err != nil {
			return core.Fail(err)
		}
		return core.Ok(nil)
	case WorkspaceListAction:
		entries, err := workspace.ListWorkspaceFiles(command.workspaceName(), command.Path)
		if err != nil {
			return core.Fail(err)
		}
		return core.Ok(entries)
	default:
		return core.Fail(core.E("workspace.HandleWorkspaceCommand", core.Concat("unsupported action: ", command.Action), fs.ErrInvalid))
	}
}

func (workspace *Workspace) workspaceFilePath(operation, name, filePath string, allowEmptyPath bool) (string, error) {
	_, workspacePath, err := workspace.workspaceNameAndPath(operation, name)
	if err != nil {
		return "", err
	}
	if !workspace.medium.IsDir(workspacePath) {
		return "", core.E(operation, core.Concat("workspace not found: ", name), fs.ErrNotExist)
	}
	cleanFilePath, err := cleanMediumSubpath(operation, filePath, allowEmptyPath)
	if err != nil {
		return "", err
	}
	return joinMediumSubpaths(workspacePath, cleanFilePath), nil
}

func (workspace *Workspace) workspacePath(operation, name string) (string, error) {
	_, workspacePath, err := workspace.workspaceNameAndPath(operation, name)
	return workspacePath, err
}

func (workspace *Workspace) workspaceNameAndPath(operation, name string) (string, string, error) {
	if workspace == nil || workspace.medium == nil {
		return "", "", core.E(operation, errWorkspaceServiceNotConfigured, fs.ErrInvalid)
	}
	workspaceName, err := cleanWorkspaceName(operation, name)
	if err != nil {
		return "", "", err
	}
	return workspaceName, joinMediumSubpaths(workspace.base, workspaceName), nil
}

func (workspace *Workspace) scopedMedium(root string) coreio.Medium {
	return &scopedMedium{
		medium: workspace.medium,
		root:   root,
	}
}

func cleanWorkspaceName(operation, name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" || name == "." || name == ".." {
		return "", core.E(operation, "workspace name is required", fs.ErrInvalid)
	}
	if strings.Contains(name, "/") || strings.Contains(name, "\\") {
		return "", core.E(operation, core.Concat("workspace name contains path separator: ", name), fs.ErrPermission)
	}
	return name, nil
}

func cleanMediumSubpath(operation, subpath string, allowEmpty bool) (string, error) {
	subpath = strings.TrimSpace(strings.ReplaceAll(subpath, "\\", "/"))
	if subpath == "" || subpath == "." {
		if allowEmpty {
			return "", nil
		}
		return "", core.E(operation, "path is required", fs.ErrInvalid)
	}
	if strings.HasPrefix(subpath, "/") {
		return "", core.E(operation, core.Concat("absolute path rejected: ", subpath), fs.ErrPermission)
	}
	for _, part := range strings.Split(subpath, "/") {
		if part == ".." {
			return "", core.E(operation, core.Concat("path traversal rejected: ", subpath), fs.ErrPermission)
		}
	}
	cleaned := path.Clean(subpath)
	if cleaned == "." {
		if allowEmpty {
			return "", nil
		}
		return "", core.E(operation, "path is required", fs.ErrInvalid)
	}
	return cleaned, nil
}

func joinMediumSubpaths(parts ...string) string {
	filtered := make([]string, 0, len(parts))
	for _, part := range parts {
		if part != "" {
			filtered = append(filtered, part)
		}
	}
	if len(filtered) == 0 {
		return ""
	}
	return path.Join(filtered...)
}

type scopedMedium struct {
	medium coreio.Medium
	root   string
}

var _ coreio.Medium = (*scopedMedium)(nil)
var _ fs.FS = (*scopedMedium)(nil)

func (medium *scopedMedium) scopedPath(operation, entryPath string, allowRoot bool) (string, error) {
	cleanPath, err := cleanMediumSubpath(operation, entryPath, allowRoot)
	if err != nil {
		return "", err
	}
	return joinMediumSubpaths(medium.root, cleanPath), nil
}

func (medium *scopedMedium) Read(entryPath string) (string, error) {
	scopedPath, err := medium.scopedPath("workspace.scopedMedium.Read", entryPath, false)
	if err != nil {
		return "", err
	}
	return medium.medium.Read(scopedPath)
}

func (medium *scopedMedium) Write(entryPath, content string) error {
	return medium.WriteMode(entryPath, content, 0644)
}

func (medium *scopedMedium) WriteMode(entryPath, content string, mode fs.FileMode) error {
	scopedPath, err := medium.scopedPath("workspace.scopedMedium.WriteMode", entryPath, false)
	if err != nil {
		return err
	}
	return medium.medium.WriteMode(scopedPath, content, mode)
}

func (medium *scopedMedium) EnsureDir(entryPath string) error {
	scopedPath, err := medium.scopedPath("workspace.scopedMedium.EnsureDir", entryPath, true)
	if err != nil {
		return err
	}
	return medium.medium.EnsureDir(scopedPath)
}

func (medium *scopedMedium) IsFile(entryPath string) bool {
	scopedPath, err := medium.scopedPath("workspace.scopedMedium.IsFile", entryPath, false)
	if err != nil {
		return false
	}
	return medium.medium.IsFile(scopedPath)
}

func (medium *scopedMedium) Delete(entryPath string) error {
	scopedPath, err := medium.scopedPath("workspace.scopedMedium.Delete", entryPath, false)
	if err != nil {
		return err
	}
	return medium.medium.Delete(scopedPath)
}

func (medium *scopedMedium) DeleteAll(entryPath string) error {
	scopedPath, err := medium.scopedPath("workspace.scopedMedium.DeleteAll", entryPath, false)
	if err != nil {
		return err
	}
	return medium.medium.DeleteAll(scopedPath)
}

func (medium *scopedMedium) Rename(oldPath, newPath string) error {
	scopedOldPath, err := medium.scopedPath("workspace.scopedMedium.Rename", oldPath, false)
	if err != nil {
		return err
	}
	scopedNewPath, err := medium.scopedPath("workspace.scopedMedium.Rename", newPath, false)
	if err != nil {
		return err
	}
	return medium.medium.Rename(scopedOldPath, scopedNewPath)
}

func (medium *scopedMedium) List(entryPath string) ([]fs.DirEntry, error) {
	scopedPath, err := medium.scopedPath("workspace.scopedMedium.List", entryPath, true)
	if err != nil {
		return nil, err
	}
	return medium.medium.List(scopedPath)
}

func (medium *scopedMedium) Stat(entryPath string) (fs.FileInfo, error) {
	scopedPath, err := medium.scopedPath("workspace.scopedMedium.Stat", entryPath, true)
	if err != nil {
		return nil, err
	}
	return medium.medium.Stat(scopedPath)
}

func (medium *scopedMedium) Open(entryPath string) (fs.File, error) {
	scopedPath, err := medium.scopedPath("workspace.scopedMedium.Open", entryPath, false)
	if err != nil {
		return nil, err
	}
	return medium.medium.Open(scopedPath)
}

func (medium *scopedMedium) Create(entryPath string) (goio.WriteCloser, error) {
	scopedPath, err := medium.scopedPath("workspace.scopedMedium.Create", entryPath, false)
	if err != nil {
		return nil, err
	}
	return medium.medium.Create(scopedPath)
}

func (medium *scopedMedium) Append(entryPath string) (goio.WriteCloser, error) {
	scopedPath, err := medium.scopedPath("workspace.scopedMedium.Append", entryPath, false)
	if err != nil {
		return nil, err
	}
	return medium.medium.Append(scopedPath)
}

func (medium *scopedMedium) ReadStream(entryPath string) (goio.ReadCloser, error) {
	scopedPath, err := medium.scopedPath("workspace.scopedMedium.ReadStream", entryPath, false)
	if err != nil {
		return nil, err
	}
	return medium.medium.ReadStream(scopedPath)
}

func (medium *scopedMedium) WriteStream(entryPath string) (goio.WriteCloser, error) {
	scopedPath, err := medium.scopedPath("workspace.scopedMedium.WriteStream", entryPath, false)
	if err != nil {
		return nil, err
	}
	return medium.medium.WriteStream(scopedPath)
}

func (medium *scopedMedium) Exists(entryPath string) bool {
	scopedPath, err := medium.scopedPath("workspace.scopedMedium.Exists", entryPath, true)
	if err != nil {
		return false
	}
	return medium.medium.Exists(scopedPath)
}

func (medium *scopedMedium) IsDir(entryPath string) bool {
	scopedPath, err := medium.scopedPath("workspace.scopedMedium.IsDir", entryPath, true)
	if err != nil {
		return false
	}
	return medium.medium.IsDir(scopedPath)
}
