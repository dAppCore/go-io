package workspace

import (
	core "dappco.re/go"
	coreio "dappco.re/go/io"
	goio "io"
	"io/fs"
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

func TestWorkspace_NewWorkspace_Bad(t *core.T) {
	workspaceService, err := NewWorkspace(nil, "root")
	core.AssertError(t, err)
	core.AssertNil(t, workspaceService)
}

func TestWorkspace_NewWorkspace_Ugly(t *core.T) {
	workspaceService, err := NewWorkspace(coreio.NewMemoryMedium(), "../escape")
	core.AssertError(t, err)
	core.AssertNil(t, workspaceService)
}

func TestWorkspace_Workspace_CreateWorkspace_Good(t *core.T) {
	workspaceService, medium := newTestWorkspace(t)
	scoped, err := workspaceService.CreateWorkspace("alpha")
	core.AssertNoError(t, err)
	core.AssertNotNil(t, scoped)
	core.AssertTrue(t, medium.IsDir("workspaces/alpha"))
}

func TestWorkspace_Workspace_CreateWorkspace_Bad(t *core.T) {
	workspaceService, _ := newTestWorkspace(t)
	_, err := workspaceService.CreateWorkspace("")
	core.AssertError(t, err)
	core.AssertEqual(t, "", workspaceService.CurrentWorkspace())
}

func TestWorkspace_Workspace_CreateWorkspace_Ugly(t *core.T) {
	workspaceService, _ := newTestWorkspace(t)
	_, err := workspaceService.CreateWorkspace("../escape")
	core.AssertError(t, err)
	core.AssertEqual(t, "", workspaceService.CurrentWorkspace())
}

func TestWorkspace_Workspace_SwitchWorkspace_Good(t *core.T) {
	workspaceService, _ := newTestWorkspace(t)
	_, err := workspaceService.CreateWorkspace("alpha")
	core.RequireNoError(t, err)
	err = workspaceService.SwitchWorkspace("alpha")
	core.AssertNoError(t, err)
	core.AssertEqual(t, "alpha", workspaceService.CurrentWorkspace())
}

func TestWorkspace_Workspace_SwitchWorkspace_Bad(t *core.T) {
	workspaceService, _ := newTestWorkspace(t)
	err := workspaceService.SwitchWorkspace("missing")
	core.AssertError(t, err)
	core.AssertEqual(t, "", workspaceService.CurrentWorkspace())
}

func TestWorkspace_Workspace_SwitchWorkspace_Ugly(t *core.T) {
	workspaceService, _ := newTestWorkspace(t)
	err := workspaceService.SwitchWorkspace("../escape")
	core.AssertError(t, err)
	core.AssertEqual(t, "", workspaceService.CurrentWorkspace())
}

func TestWorkspace_Workspace_CurrentWorkspace_Good(t *core.T) {
	workspaceService, _ := newTestWorkspace(t)
	_, err := workspaceService.CreateWorkspace("alpha")
	core.RequireNoError(t, err)
	core.RequireNoError(t, workspaceService.SwitchWorkspace("alpha"))
	got := workspaceService.CurrentWorkspace()
	core.AssertEqual(t, "alpha", got)
}

func TestWorkspace_Workspace_CurrentWorkspace_Bad(t *core.T) {
	var workspaceService *Workspace
	got := workspaceService.CurrentWorkspace()
	core.AssertEqual(t, "", got)
}

func TestWorkspace_Workspace_CurrentWorkspace_Ugly(t *core.T) {
	workspaceService, _ := newTestWorkspace(t)
	got := workspaceService.CurrentWorkspace()
	core.AssertEqual(t, "", got)
}

func TestWorkspace_Workspace_ReadWorkspaceFile_Good(t *core.T) {
	workspaceService, _ := newTestWorkspace(t)
	_, err := workspaceService.CreateWorkspace("alpha")
	core.RequireNoError(t, err)
	core.RequireNoError(t, workspaceService.WriteWorkspaceFile("alpha", "note.txt", "payload"))
	got, err := workspaceService.ReadWorkspaceFile("alpha", "note.txt")
	core.AssertNoError(t, err)
	core.AssertEqual(t, "payload", got)
}

func TestWorkspace_Workspace_ReadWorkspaceFile_Bad(t *core.T) {
	workspaceService, _ := newTestWorkspace(t)
	got, err := workspaceService.ReadWorkspaceFile("missing", "note.txt")
	core.AssertError(t, err)
	core.AssertEqual(t, "", got)
}

func TestWorkspace_Workspace_ReadWorkspaceFile_Ugly(t *core.T) {
	workspaceService, _ := newTestWorkspace(t)
	_, err := workspaceService.CreateWorkspace("alpha")
	core.RequireNoError(t, err)
	got, err := workspaceService.ReadWorkspaceFile("alpha", "../escape")
	core.AssertError(t, err)
	core.AssertEqual(t, "", got)
}

func TestWorkspace_Workspace_WriteWorkspaceFile_Good(t *core.T) {
	workspaceService, medium := newTestWorkspace(t)
	_, err := workspaceService.CreateWorkspace("alpha")
	core.RequireNoError(t, err)
	err = workspaceService.WriteWorkspaceFile("alpha", "note.txt", "payload")
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.IsFile("workspaces/alpha/note.txt"))
}

func TestWorkspace_Workspace_WriteWorkspaceFile_Bad(t *core.T) {
	workspaceService, _ := newTestWorkspace(t)
	err := workspaceService.WriteWorkspaceFile("missing", "note.txt", "payload")
	core.AssertError(t, err)
	core.AssertEqual(t, "", workspaceService.CurrentWorkspace())
}

func TestWorkspace_Workspace_WriteWorkspaceFile_Ugly(t *core.T) {
	workspaceService, medium := newTestWorkspace(t)
	_, err := workspaceService.CreateWorkspace("alpha")
	core.RequireNoError(t, err)
	err = workspaceService.WriteWorkspaceFile("alpha", "../escape.txt", "payload")
	core.AssertError(t, err)
	core.AssertFalse(t, medium.IsFile("workspaces/escape.txt"))
}

func TestWorkspace_Workspace_ListWorkspaceFiles_Good(t *core.T) {
	workspaceService, _ := newTestWorkspace(t)
	_, err := workspaceService.CreateWorkspace("alpha")
	core.RequireNoError(t, err)
	core.RequireNoError(t, workspaceService.WriteWorkspaceFile("alpha", "notes/todo.txt", "payload"))
	entries, err := workspaceService.ListWorkspaceFiles("alpha", "notes")
	core.AssertNoError(t, err)
	core.AssertLen(t, entries, 1)
}

func TestWorkspace_Workspace_ListWorkspaceFiles_Bad(t *core.T) {
	workspaceService, _ := newTestWorkspace(t)
	entries, err := workspaceService.ListWorkspaceFiles("missing", "")
	core.AssertError(t, err)
	core.AssertNil(t, entries)
}

func TestWorkspace_Workspace_ListWorkspaceFiles_Ugly(t *core.T) {
	workspaceService, _ := newTestWorkspace(t)
	_, err := workspaceService.CreateWorkspace("alpha")
	core.RequireNoError(t, err)
	entries, err := workspaceService.ListWorkspaceFiles("alpha", "")
	core.AssertNoError(t, err)
	core.AssertEmpty(t, entries)
}

func TestWorkspace_Workspace_HandleWorkspaceCommand_Good(t *core.T) {
	workspaceService, _ := newTestWorkspace(t)
	result := workspaceService.HandleWorkspaceCommand(WorkspaceCommand{Action: WorkspaceCreateAction, Workspace: "alpha"})
	core.AssertTrue(t, result.OK)
	core.AssertNotNil(t, result.Value)
}

func TestWorkspace_Workspace_HandleWorkspaceCommand_Bad(t *core.T) {
	workspaceService, _ := newTestWorkspace(t)
	result := workspaceService.HandleWorkspaceCommand(WorkspaceCommand{Action: "unknown"})
	core.AssertFalse(t, result.OK)
	core.AssertNotNil(t, result.Value)
}

func TestWorkspace_Workspace_HandleWorkspaceCommand_Ugly(t *core.T) {
	workspaceService, _ := newTestWorkspace(t)
	result := workspaceService.HandleWorkspaceCommand(WorkspaceCommand{Action: WorkspaceCreateAction, Workspace: "../escape"})
	core.AssertFalse(t, result.OK)
	core.AssertNotNil(t, result.Value)
}

func TestWorkspace_Medium_Read_Good(t *core.T) {
	medium, _ := newScopedMediumFixture(t)
	core.RequireNoError(t, medium.Write("read.txt", "payload"))
	got, err := medium.Read("read.txt")
	core.AssertNoError(t, err)
	core.AssertEqual(t, "payload", got)
}

func TestWorkspace_Medium_Read_Bad(t *core.T) {
	medium, _ := newScopedMediumFixture(t)
	got, err := medium.Read("../escape")
	core.AssertError(t, err)
	core.AssertEqual(t, "", got)
}

func TestWorkspace_Medium_Read_Ugly(t *core.T) {
	medium, _ := newScopedMediumFixture(t)
	got, err := medium.Read("missing.txt")
	core.AssertError(t, err)
	core.AssertEqual(t, "", got)
}

func TestWorkspace_Medium_Write_Good(t *core.T) {
	medium, backing := newScopedMediumFixture(t)
	err := medium.Write("write.txt", "payload")
	core.AssertNoError(t, err)
	core.AssertTrue(t, backing.IsFile("workspaces/alpha/write.txt"))
}

func TestWorkspace_Medium_Write_Bad(t *core.T) {
	medium, _ := newScopedMediumFixture(t)
	err := medium.Write("../escape", "payload")
	core.AssertError(t, err)
	core.AssertFalse(t, medium.Exists("../escape"))
}

func TestWorkspace_Medium_Write_Ugly(t *core.T) {
	medium, _ := newScopedMediumFixture(t)
	err := medium.Write("empty.txt", "")
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.Exists("empty.txt"))
}

func TestWorkspace_Medium_WriteMode_Good(t *core.T) {
	medium, _ := newScopedMediumFixture(t)
	err := medium.WriteMode("mode.txt", "payload", 0600)
	info, statErr := medium.Stat("mode.txt")
	core.AssertNoError(t, err)
	core.AssertNoError(t, statErr)
	core.AssertEqual(t, fs.FileMode(0600), info.Mode())
}

func TestWorkspace_Medium_WriteMode_Bad(t *core.T) {
	medium, _ := newScopedMediumFixture(t)
	err := medium.WriteMode("../escape", "payload", 0600)
	core.AssertError(t, err)
	core.AssertFalse(t, medium.Exists("../escape"))
}

func TestWorkspace_Medium_WriteMode_Ugly(t *core.T) {
	medium, _ := newScopedMediumFixture(t)
	err := medium.WriteMode("zero-mode.txt", "payload", 0)
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.Exists("zero-mode.txt"))
}

func TestWorkspace_Medium_EnsureDir_Good(t *core.T) {
	medium, _ := newScopedMediumFixture(t)
	err := medium.EnsureDir("dir")
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.IsDir("dir"))
}

func TestWorkspace_Medium_EnsureDir_Bad(t *core.T) {
	medium, _ := newScopedMediumFixture(t)
	err := medium.EnsureDir("../escape")
	core.AssertError(t, err)
	core.AssertFalse(t, medium.IsDir("../escape"))
}

func TestWorkspace_Medium_EnsureDir_Ugly(t *core.T) {
	medium, _ := newScopedMediumFixture(t)
	err := medium.EnsureDir("")
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.IsDir(""))
}

func TestWorkspace_Medium_IsFile_Good(t *core.T) {
	medium, _ := newScopedMediumFixture(t)
	core.RequireNoError(t, medium.Write("file.txt", "payload"))
	got := medium.IsFile("file.txt")
	core.AssertTrue(t, got)
}

func TestWorkspace_Medium_IsFile_Bad(t *core.T) {
	medium, _ := newScopedMediumFixture(t)
	got := medium.IsFile("../escape")
	core.AssertFalse(t, got)
}

func TestWorkspace_Medium_IsFile_Ugly(t *core.T) {
	medium, _ := newScopedMediumFixture(t)
	core.RequireNoError(t, medium.EnsureDir("dir"))
	got := medium.IsFile("dir")
	core.AssertFalse(t, got)
}

func TestWorkspace_Medium_Delete_Good(t *core.T) {
	medium, _ := newScopedMediumFixture(t)
	core.RequireNoError(t, medium.Write("delete.txt", "payload"))
	err := medium.Delete("delete.txt")
	core.AssertNoError(t, err)
	core.AssertFalse(t, medium.Exists("delete.txt"))
}

func TestWorkspace_Medium_Delete_Bad(t *core.T) {
	medium, _ := newScopedMediumFixture(t)
	err := medium.Delete("../escape")
	core.AssertError(t, err)
	core.AssertFalse(t, medium.Exists("../escape"))
}

func TestWorkspace_Medium_Delete_Ugly(t *core.T) {
	medium, _ := newScopedMediumFixture(t)
	err := medium.Delete("missing.txt")
	core.AssertError(t, err)
	core.AssertFalse(t, medium.Exists("missing.txt"))
}

func TestWorkspace_Medium_DeleteAll_Good(t *core.T) {
	medium, _ := newScopedMediumFixture(t)
	core.RequireNoError(t, medium.Write("tree/file.txt", "payload"))
	err := medium.DeleteAll("tree")
	core.AssertNoError(t, err)
	core.AssertFalse(t, medium.Exists("tree/file.txt"))
}

func TestWorkspace_Medium_DeleteAll_Bad(t *core.T) {
	medium, _ := newScopedMediumFixture(t)
	err := medium.DeleteAll("../escape")
	core.AssertError(t, err)
	core.AssertFalse(t, medium.Exists("../escape"))
}

func TestWorkspace_Medium_DeleteAll_Ugly(t *core.T) {
	medium, _ := newScopedMediumFixture(t)
	err := medium.DeleteAll("missing")
	core.AssertError(t, err)
	core.AssertFalse(t, medium.Exists("missing"))
}

func TestWorkspace_Medium_Rename_Good(t *core.T) {
	medium, _ := newScopedMediumFixture(t)
	core.RequireNoError(t, medium.Write("old.txt", "payload"))
	err := medium.Rename("old.txt", "new.txt")
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.Exists("new.txt"))
}

func TestWorkspace_Medium_Rename_Bad(t *core.T) {
	medium, _ := newScopedMediumFixture(t)
	err := medium.Rename("../escape", "new.txt")
	core.AssertError(t, err)
	core.AssertFalse(t, medium.Exists("new.txt"))
}

func TestWorkspace_Medium_Rename_Ugly(t *core.T) {
	medium, _ := newScopedMediumFixture(t)
	core.RequireNoError(t, medium.Write("dir/old.txt", "payload"))
	err := medium.Rename("dir", "moved")
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.Exists("moved/old.txt"))
}

func TestWorkspace_Medium_List_Good(t *core.T) {
	medium, _ := newScopedMediumFixture(t)
	core.RequireNoError(t, medium.Write("dir/file.txt", "payload"))
	entries, err := medium.List("dir")
	core.AssertNoError(t, err)
	core.AssertLen(t, entries, 1)
}

func TestWorkspace_Medium_List_Bad(t *core.T) {
	medium, _ := newScopedMediumFixture(t)
	entries, err := medium.List("../escape")
	core.AssertError(t, err)
	core.AssertNil(t, entries)
}

func TestWorkspace_Medium_List_Ugly(t *core.T) {
	medium, _ := newScopedMediumFixture(t)
	entries, err := medium.List("")
	core.AssertNoError(t, err)
	core.AssertEmpty(t, entries)
}

func TestWorkspace_Medium_Stat_Good(t *core.T) {
	medium, _ := newScopedMediumFixture(t)
	core.RequireNoError(t, medium.Write("stat.txt", "payload"))
	info, err := medium.Stat("stat.txt")
	core.AssertNoError(t, err)
	core.AssertEqual(t, "stat.txt", info.Name())
}

func TestWorkspace_Medium_Stat_Bad(t *core.T) {
	medium, _ := newScopedMediumFixture(t)
	info, err := medium.Stat("../escape")
	core.AssertError(t, err)
	core.AssertNil(t, info)
}

func TestWorkspace_Medium_Stat_Ugly(t *core.T) {
	medium, _ := newScopedMediumFixture(t)
	info, err := medium.Stat("")
	core.AssertNoError(t, err)
	core.AssertTrue(t, info.IsDir())
}

func TestWorkspace_Medium_Open_Good(t *core.T) {
	medium, _ := newScopedMediumFixture(t)
	core.RequireNoError(t, medium.Write("open.txt", "payload"))
	file, err := medium.Open("open.txt")
	core.AssertNoError(t, err)
	core.AssertNotNil(t, file)
	core.RequireNoError(t, file.Close())
}

func TestWorkspace_Medium_Open_Bad(t *core.T) {
	medium, _ := newScopedMediumFixture(t)
	file, err := medium.Open("../escape")
	core.AssertError(t, err)
	core.AssertNil(t, file)
}

func TestWorkspace_Medium_Open_Ugly(t *core.T) {
	medium, _ := newScopedMediumFixture(t)
	file, err := medium.Open("missing.txt")
	core.AssertError(t, err)
	core.AssertNil(t, file)
}

func TestWorkspace_Medium_Create_Good(t *core.T) {
	medium, _ := newScopedMediumFixture(t)
	writer, err := medium.Create("create.txt")
	core.RequireNoError(t, err)
	_, writeErr := writer.Write([]byte("payload"))
	core.AssertNoError(t, writeErr)
	core.AssertNoError(t, writer.Close())
}

func TestWorkspace_Medium_Create_Bad(t *core.T) {
	medium, _ := newScopedMediumFixture(t)
	writer, err := medium.Create("../escape")
	core.AssertError(t, err)
	core.AssertNil(t, writer)
}

func TestWorkspace_Medium_Create_Ugly(t *core.T) {
	medium, _ := newScopedMediumFixture(t)
	writer, err := medium.Create("nested/create.txt")
	core.RequireNoError(t, err)
	core.AssertNoError(t, writer.Close())
	core.AssertTrue(t, medium.Exists("nested/create.txt"))
}

func TestWorkspace_Medium_Append_Good(t *core.T) {
	medium, _ := newScopedMediumFixture(t)
	core.RequireNoError(t, medium.Write("append.txt", "a"))
	writer, err := medium.Append("append.txt")
	core.RequireNoError(t, err)
	_, writeErr := writer.Write([]byte("b"))
	core.RequireNoError(t, writeErr)
	core.RequireNoError(t, writer.Close())
}

func TestWorkspace_Medium_Append_Bad(t *core.T) {
	medium, _ := newScopedMediumFixture(t)
	writer, err := medium.Append("../escape")
	core.AssertError(t, err)
	core.AssertNil(t, writer)
}

func TestWorkspace_Medium_Append_Ugly(t *core.T) {
	medium, _ := newScopedMediumFixture(t)
	writer, err := medium.Append("new.txt")
	core.RequireNoError(t, err)
	_, writeErr := writer.Write([]byte("created"))
	core.RequireNoError(t, writeErr)
	core.RequireNoError(t, writer.Close())
}

func TestWorkspace_Medium_ReadStream_Good(t *core.T) {
	medium, _ := newScopedMediumFixture(t)
	core.RequireNoError(t, medium.Write("stream.txt", "payload"))
	reader, err := medium.ReadStream("stream.txt")
	core.RequireNoError(t, err)
	defer reader.Close()
	data, readErr := goio.ReadAll(reader)
	core.AssertNoError(t, readErr)
	core.AssertEqual(t, "payload", string(data))
}

func TestWorkspace_Medium_ReadStream_Bad(t *core.T) {
	medium, _ := newScopedMediumFixture(t)
	reader, err := medium.ReadStream("../escape")
	core.AssertError(t, err)
	core.AssertNil(t, reader)
}

func TestWorkspace_Medium_ReadStream_Ugly(t *core.T) {
	medium, _ := newScopedMediumFixture(t)
	reader, err := medium.ReadStream("missing.txt")
	core.AssertError(t, err)
	core.AssertNil(t, reader)
}

func TestWorkspace_Medium_WriteStream_Good(t *core.T) {
	medium, _ := newScopedMediumFixture(t)
	writer, err := medium.WriteStream("stream.txt")
	core.RequireNoError(t, err)
	_, writeErr := writer.Write([]byte("payload"))
	core.AssertNoError(t, writeErr)
	core.AssertNoError(t, writer.Close())
}

func TestWorkspace_Medium_WriteStream_Bad(t *core.T) {
	medium, _ := newScopedMediumFixture(t)
	writer, err := medium.WriteStream("../escape")
	core.AssertError(t, err)
	core.AssertNil(t, writer)
}

func TestWorkspace_Medium_WriteStream_Ugly(t *core.T) {
	medium, _ := newScopedMediumFixture(t)
	writer, err := medium.WriteStream("nested/stream.txt")
	core.RequireNoError(t, err)
	core.AssertNoError(t, writer.Close())
	core.AssertTrue(t, medium.Exists("nested/stream.txt"))
}

func TestWorkspace_Medium_Exists_Good(t *core.T) {
	medium, _ := newScopedMediumFixture(t)
	core.RequireNoError(t, medium.Write("exists.txt", "payload"))
	got := medium.Exists("exists.txt")
	core.AssertTrue(t, got)
}

func TestWorkspace_Medium_Exists_Bad(t *core.T) {
	medium, _ := newScopedMediumFixture(t)
	got := medium.Exists("../escape")
	core.AssertFalse(t, got)
}

func TestWorkspace_Medium_Exists_Ugly(t *core.T) {
	medium, _ := newScopedMediumFixture(t)
	got := medium.Exists("")
	core.AssertTrue(t, got)
}

func TestWorkspace_Medium_IsDir_Good(t *core.T) {
	medium, _ := newScopedMediumFixture(t)
	core.RequireNoError(t, medium.EnsureDir("dir"))
	got := medium.IsDir("dir")
	core.AssertTrue(t, got)
}

func TestWorkspace_Medium_IsDir_Bad(t *core.T) {
	medium, _ := newScopedMediumFixture(t)
	got := medium.IsDir("../escape")
	core.AssertFalse(t, got)
}

func TestWorkspace_Medium_IsDir_Ugly(t *core.T) {
	medium, _ := newScopedMediumFixture(t)
	core.RequireNoError(t, medium.Write("file.txt", "payload"))
	got := medium.IsDir("file.txt")
	core.AssertFalse(t, got)
}
