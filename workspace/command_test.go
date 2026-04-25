package workspace

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWorkspaceCommand_Good(t *testing.T) {
	command := WorkspaceCommand{
		Action:    WorkspaceWriteAction,
		Workspace: "alpha",
		Path:      "notes/todo.txt",
		Content:   "ship it",
	}

	assert.Equal(t, "write", command.Action)
	assert.Equal(t, "alpha", command.workspaceName())
	assert.Equal(t, "notes/todo.txt", command.Path)
	assert.Equal(t, "ship it", command.Content)
}

func TestWorkspaceCommand_Bad_EmptyWorkspace(t *testing.T) {
	command := WorkspaceCommand{Action: WorkspaceReadAction}

	assert.Empty(t, command.workspaceName())
}

func TestWorkspaceCommand_Ugly_LegacyWorkspaceFields(t *testing.T) {
	assert.Equal(t, "legacy-id", WorkspaceCommand{WorkspaceID: "legacy-id"}.workspaceName())
	assert.Equal(t, "identifier", WorkspaceCommand{Identifier: "identifier"}.workspaceName())
	assert.Equal(t, "workspace", WorkspaceCommand{
		Workspace:   "workspace",
		WorkspaceID: "legacy-id",
		Identifier:  "identifier",
	}.workspaceName())
}
