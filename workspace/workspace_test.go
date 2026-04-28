package workspace

import (
	core "dappco.re/go"
	"io/fs"

	coreio "dappco.re/go/io"
)

func newTestWorkspace(t *core.T) (*Workspace, *coreio.MemoryMedium) {
	t.Helper()

	medium := coreio.NewMemoryMedium()
	workspaceService, err := NewWorkspace(medium, "workspaces")
	core.RequireNoError(t, err)
	return workspaceService, medium
}

func TestWorkspace_NewWorkspace_Good(t *core.T) {
	medium := coreio.NewMemoryMedium()

	workspaceService, err := NewWorkspace(medium, "root")
	core.RequireNoError(t, err)

	core.AssertNotNil(t, workspaceService)
	core.AssertTrue(t, medium.IsDir("root"))
}

func TestWorkspace_NewWorkspace_Bad_InvalidBase(t *core.T) {
	workspaceService, err := NewWorkspace(coreio.NewMemoryMedium(), "../escape")

	core.AssertError(t, err)
	core.AssertNil(t, workspaceService)
}

func TestWorkspace_NewWorkspace_Ugly_NilMediumRejected(t *core.T) {
	workspaceService, err := NewWorkspace(nil, "")

	core.AssertError(t, err)
	core.AssertNil(t, workspaceService)
}

func TestWorkspace_CreateWorkspace_Good(t *core.T) {
	workspaceService, medium := newTestWorkspace(t)

	scoped, err := workspaceService.CreateWorkspace("alpha")
	core.RequireNoError(t, err)
	core.AssertNotNil(t, scoped)

	core.AssertTrue(t, medium.IsDir("workspaces/alpha"))
	core.RequireNoError(t, scoped.Write("note.txt", "hello"))
	content, err := medium.Read("workspaces/alpha/note.txt")
	core.RequireNoError(t, err)
	core.AssertEqual(t, "hello", content)
}

func TestWorkspace_CreateWorkspace_Bad_Duplicate(t *core.T) {
	workspaceService, _ := newTestWorkspace(t)
	_, err := workspaceService.CreateWorkspace("alpha")
	core.RequireNoError(t, err)

	_, err = workspaceService.CreateWorkspace("alpha")

	core.AssertError(t, err)
}

func TestWorkspace_CreateWorkspace_Ugly_TraversalName(t *core.T) {
	workspaceService, medium := newTestWorkspace(t)

	_, err := workspaceService.CreateWorkspace("../escape")

	core.AssertError(t, err)
	core.AssertFalse(t, medium.IsDir("escape"))
}

func TestWorkspace_SwitchWorkspace_Good(t *core.T) {
	workspaceService, _ := newTestWorkspace(t)
	_, err := workspaceService.CreateWorkspace("alpha")
	core.RequireNoError(t, err)

	err = workspaceService.SwitchWorkspace("alpha")

	core.RequireNoError(t, err)
	core.AssertEqual(t, "alpha", workspaceService.CurrentWorkspace())
}

func TestWorkspace_SwitchWorkspace_Bad_Missing(t *core.T) {
	workspaceService, _ := newTestWorkspace(t)

	err := workspaceService.SwitchWorkspace("missing")

	core.AssertError(t, err)
	core.AssertEmpty(t, workspaceService.CurrentWorkspace())
}

func TestWorkspace_SwitchWorkspace_Ugly_TraversalName(t *core.T) {
	workspaceService, _ := newTestWorkspace(t)
	_, err := workspaceService.CreateWorkspace("alpha")
	core.RequireNoError(t, err)
	core.RequireNoError(t, workspaceService.SwitchWorkspace("alpha"))

	err = workspaceService.SwitchWorkspace("alpha/../beta")

	core.AssertError(t, err)
	core.AssertEqual(t, "alpha", workspaceService.CurrentWorkspace())
}

func TestWorkspace_ReadWorkspaceFile_Good(t *core.T) {
	workspaceService, _ := newTestWorkspace(t)
	_, err := workspaceService.CreateWorkspace("alpha")
	core.RequireNoError(t, err)
	core.RequireNoError(t, workspaceService.WriteWorkspaceFile("alpha", "notes/todo.txt", "hello"))

	content, err := workspaceService.ReadWorkspaceFile("alpha", "notes/todo.txt")

	core.RequireNoError(t, err)
	core.AssertEqual(t, "hello", content)
}

func TestWorkspace_ReadWorkspaceFile_Bad_MissingFile(t *core.T) {
	workspaceService, _ := newTestWorkspace(t)
	_, err := workspaceService.CreateWorkspace("alpha")
	core.RequireNoError(t, err)

	_, err = workspaceService.ReadWorkspaceFile("alpha", "missing.txt")

	core.AssertError(t, err)
}

func TestWorkspace_ReadWorkspaceFile_Ugly_TraversalPath(t *core.T) {
	workspaceService, _ := newTestWorkspace(t)
	_, err := workspaceService.CreateWorkspace("alpha")
	core.RequireNoError(t, err)

	_, err = workspaceService.ReadWorkspaceFile("alpha", "../alpha/secret.txt")

	core.AssertError(t, err)
}

func TestWorkspace_WriteWorkspaceFile_Good(t *core.T) {
	workspaceService, medium := newTestWorkspace(t)
	_, err := workspaceService.CreateWorkspace("alpha")
	core.RequireNoError(t, err)

	err = workspaceService.WriteWorkspaceFile("alpha", "notes/todo.txt", "hello")

	core.RequireNoError(t, err)
	content, err := medium.Read("workspaces/alpha/notes/todo.txt")
	core.RequireNoError(t, err)
	core.AssertEqual(t, "hello", content)
}

func TestWorkspace_WriteWorkspaceFile_Bad_MissingWorkspace(t *core.T) {
	workspaceService, _ := newTestWorkspace(t)

	err := workspaceService.WriteWorkspaceFile("missing", "notes/todo.txt", "hello")

	core.AssertError(t, err)
}

func TestWorkspace_WriteWorkspaceFile_Ugly_TraversalPath(t *core.T) {
	workspaceService, medium := newTestWorkspace(t)
	_, err := workspaceService.CreateWorkspace("alpha")
	core.RequireNoError(t, err)

	err = workspaceService.WriteWorkspaceFile("alpha", "../outside.txt", "secret")

	core.AssertError(t, err)
	core.AssertFalse(t, medium.IsFile("workspaces/outside.txt"))
}

func TestWorkspace_HandleWorkspaceCommand_Good(t *core.T) {
	workspaceService, _ := newTestWorkspace(t)

	create := workspaceService.HandleWorkspaceCommand(WorkspaceCommand{
		Action:    WorkspaceCreateAction,
		Workspace: "alpha",
	})
	core.RequireTrue(t, create.OK)
	_, ok := create.Value.(coreio.Medium)
	core.RequireTrue(t, ok)

	write := workspaceService.HandleWorkspaceCommand(WorkspaceCommand{
		Action:    WorkspaceWriteAction,
		Workspace: "alpha",
		Path:      "notes/todo.txt",
		Content:   "hello",
	})
	core.RequireTrue(t, write.OK)

	read := workspaceService.HandleWorkspaceCommand(WorkspaceCommand{
		Action:    WorkspaceReadAction,
		Workspace: "alpha",
		Path:      "notes/todo.txt",
	})
	core.RequireTrue(t, read.OK)
	core.AssertEqual(t, "hello", read.Value)

	list := workspaceService.HandleWorkspaceCommand(WorkspaceCommand{
		Action:    WorkspaceListAction,
		Workspace: "alpha",
		Path:      "notes",
	})
	core.RequireTrue(t, list.OK)
	entries, ok := list.Value.([]fs.DirEntry)
	core.RequireTrue(t, ok)
	core.AssertLen(t, entries, 1)
	core.AssertEqual(t, "todo.txt", entries[0].Name())

	switchResult := workspaceService.HandleWorkspaceCommand(WorkspaceCommand{
		Action:    WorkspaceSwitchAction,
		Workspace: "alpha",
	})
	core.RequireTrue(t, switchResult.OK)
	core.AssertEqual(t, "alpha", workspaceService.CurrentWorkspace())
}

func TestWorkspace_HandleWorkspaceCommand_Bad_UnknownAction(t *core.T) {
	workspaceService, _ := newTestWorkspace(t)

	result := workspaceService.HandleWorkspaceCommand(WorkspaceCommand{Action: "noop"})

	core.AssertFalse(t, result.OK)
}

func TestWorkspace_HandleWorkspaceCommand_Ugly_TraversalPath(t *core.T) {
	workspaceService, medium := newTestWorkspace(t)
	core.RequireTrue(t, workspaceService.HandleWorkspaceCommand(WorkspaceCommand{
		Action:    WorkspaceCreateAction,
		Workspace: "alpha",
	}).OK)

	result := workspaceService.HandleWorkspaceCommand(WorkspaceCommand{
		Action:    WorkspaceWriteAction,
		Workspace: "alpha",
		Path:      "../outside.txt",
		Content:   "secret",
	})

	core.AssertFalse(t, result.OK)
	core.AssertFalse(t, medium.IsFile("workspaces/outside.txt"))
}
