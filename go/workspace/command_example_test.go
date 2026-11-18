package workspace

import core "github.com/dappcore/go"

func ExampleWorkspaceCommand() {
	command := WorkspaceCommand{
		Action:    WorkspaceWriteAction,
		Workspace: "alpha",
		Path:      "notes/todo.txt",
		Content:   "ship it",
	}
	core.Println(command.Action, command.workspaceName(), command.Path, command.Content)
	// Output: write alpha notes/todo.txt ship it
}
