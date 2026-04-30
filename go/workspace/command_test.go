package workspace

import (
	core "dappco.re/go"
)

const workspaceLegacyID = "legacy-id"

func TestWorkspaceCommand_Good(t *core.T) {
	command := WorkspaceCommand{
		Action:    WorkspaceWriteAction,
		Workspace: "alpha",
		Path:      "notes/todo.txt",
		Content:   "ship it",
	}

	core.AssertEqual(t, "write", command.Action)
	core.AssertEqual(t, "alpha", command.workspaceName())
	core.AssertEqual(t, "notes/todo.txt", command.Path)
	core.AssertEqual(t, "ship it", command.Content)
}

func TestWorkspaceCommand_Bad_EmptyWorkspace(t *core.T) {
	command := WorkspaceCommand{Action: WorkspaceReadAction}

	core.AssertEqual(t, WorkspaceReadAction, command.Action)
	core.AssertEmpty(t, command.workspaceName())
	core.AssertEmpty(t, command.Path)
}

func TestWorkspaceCommand_Ugly_LegacyWorkspaceFields(t *core.T) {
	core.AssertEqual(t, workspaceLegacyID, WorkspaceCommand{WorkspaceID: workspaceLegacyID}.workspaceName())
	core.AssertEqual(t, "identifier", WorkspaceCommand{Identifier: "identifier"}.workspaceName())
	core.AssertEqual(t, "workspace", WorkspaceCommand{
		Workspace:   "workspace",
		WorkspaceID: workspaceLegacyID,
		Identifier:  "identifier",
	}.workspaceName())
}
