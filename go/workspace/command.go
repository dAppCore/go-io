package workspace

// Action values for WorkspaceCommand.
const (
	WorkspaceCreateAction = "create"
	WorkspaceSwitchAction = "switch"
	WorkspaceReadAction   = "read"
	WorkspaceWriteAction  = "write"
	WorkspaceListAction   = "list"
)

const (
	legacyWorkspaceCreateAction = "workspace.create"
	legacyWorkspaceSwitchAction = "workspace.switch"
)

// WorkspaceCommand is the RFC §5 DTO for workspace command dispatch.
type WorkspaceCommand struct {
	Action    string
	Workspace string
	Path      string
	Content   string

	// Legacy fields are kept so the existing encrypted workspace service can
	// continue handling its current Core IPC messages while this package exposes
	// the RFC §5 command shape.
	Identifier  string
	Password    string
	WorkspaceID string
}

func (command WorkspaceCommand) workspaceName() string {
	if command.Workspace != "" {
		return command.Workspace
	}
	if command.WorkspaceID != "" {
		return command.WorkspaceID
	}
	return command.Identifier
}
