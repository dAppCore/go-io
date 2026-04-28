package workspace

import (
	goio "io"
	"io/fs"

	core "dappco.re/go"
	coreio "dappco.re/go/io"
)

func newAX7Service(t *core.T) (*Service, *coreio.MemoryMedium) {
	t.Helper()

	medium := coreio.NewMemoryMedium()
	rootPath := core.Path(t.TempDir(), "workspaces")
	service, err := New(Options{
		KeyPairProvider: testKeyPairProvider{privateKey: "private-key"},
		RootPath:        rootPath,
		Medium:          medium,
	})
	core.RequireNoError(t, err)
	return service, medium
}

func newAX7ScopedMedium(t *core.T) (*scopedMedium, *coreio.MemoryMedium) {
	t.Helper()

	workspaceService, medium := newTestWorkspace(t)
	scoped, err := workspaceService.CreateWorkspace("alpha")
	core.RequireNoError(t, err)
	return scoped.(*scopedMedium), medium
}

func TestAX7_New_Good(t *core.T) {
	service, medium := newAX7Service(t)
	core.AssertNotNil(t, service)
	core.AssertTrue(t, medium.IsDir(service.rootPath))
}

func TestAX7_New_Bad(t *core.T) {
	service, err := New(Options{RootPath: "workspaces", Medium: coreio.NewMemoryMedium()})
	core.AssertError(t, err)
	core.AssertNil(t, service)
}

func TestAX7_New_Ugly(t *core.T) {
	c := core.New()
	service, _ := newAX7Service(t)
	result := service.HandleWorkspaceMessage(c, "unsupported")
	core.AssertFalse(t, result.OK)
}

func TestAX7_SHA256Hash_Write_Good(t *core.T) {
	hash := &workspaceSHA256Hash{}
	count, err := hash.Write([]byte("payload"))
	core.AssertNoError(t, err)
	core.AssertEqual(t, len("payload"), count)
}

func TestAX7_SHA256Hash_Write_Bad(t *core.T) {
	hash := &workspaceSHA256Hash{}
	count, err := hash.Write(nil)
	core.AssertNoError(t, err)
	core.AssertEqual(t, 0, count)
}

func TestAX7_SHA256Hash_Write_Ugly(t *core.T) {
	hash := &workspaceSHA256Hash{}
	_, err := hash.Write([]byte("a"))
	core.RequireNoError(t, err)
	_, err = hash.Write([]byte("b"))
	core.AssertNoError(t, err)
}

func TestAX7_SHA256Hash_Sum_Good(t *core.T) {
	hash := &workspaceSHA256Hash{}
	_, err := hash.Write([]byte("payload"))
	core.RequireNoError(t, err)
	sum := hash.Sum(nil)
	core.AssertLen(t, sum, 32)
}

func TestAX7_SHA256Hash_Sum_Bad(t *core.T) {
	hash := &workspaceSHA256Hash{}
	sum := hash.Sum([]byte("prefix"))
	core.AssertLen(t, sum, len("prefix")+32)
	core.AssertEqual(t, "prefix", string(sum[:len("prefix")]))
}

func TestAX7_SHA256Hash_Sum_Ugly(t *core.T) {
	hash := &workspaceSHA256Hash{}
	sum := hash.Sum(nil)
	core.AssertLen(t, sum, 32)
	core.AssertNotEmpty(t, sum)
}

func TestAX7_SHA256Hash_Reset_Good(t *core.T) {
	hash := &workspaceSHA256Hash{}
	_, err := hash.Write([]byte("payload"))
	core.RequireNoError(t, err)
	hash.Reset()
	core.AssertEmpty(t, hash.data)
}

func TestAX7_SHA256Hash_Reset_Bad(t *core.T) {
	hash := &workspaceSHA256Hash{}
	hash.Reset()
	core.AssertEmpty(t, hash.data)
	core.AssertLen(t, hash.Sum(nil), 32)
}

func TestAX7_SHA256Hash_Reset_Ugly(t *core.T) {
	hash := &workspaceSHA256Hash{data: []byte("payload")}
	hash.Reset()
	_, err := hash.Write([]byte("again"))
	core.AssertNoError(t, err)
	core.AssertEqual(t, []byte("again"), hash.data)
}

func TestAX7_SHA256Hash_Size_Good(t *core.T) {
	hash := &workspaceSHA256Hash{}
	got := hash.Size()
	core.AssertEqual(t, 32, got)
}

func TestAX7_SHA256Hash_Size_Bad(t *core.T) {
	hash := &workspaceSHA256Hash{data: []byte("payload")}
	got := hash.Size()
	core.AssertEqual(t, 32, got)
}

func TestAX7_SHA256Hash_Size_Ugly(t *core.T) {
	hash := newWorkspaceSHA256Hash()
	got := hash.Size()
	core.AssertEqual(t, 32, got)
}

func TestAX7_SHA256Hash_BlockSize_Good(t *core.T) {
	hash := &workspaceSHA256Hash{}
	got := hash.BlockSize()
	core.AssertEqual(t, 64, got)
}

func TestAX7_SHA256Hash_BlockSize_Bad(t *core.T) {
	hash := &workspaceSHA256Hash{data: []byte("payload")}
	got := hash.BlockSize()
	core.AssertEqual(t, 64, got)
}

func TestAX7_SHA256Hash_BlockSize_Ugly(t *core.T) {
	hash := newWorkspaceSHA256Hash()
	got := hash.BlockSize()
	core.AssertEqual(t, 64, got)
}

func TestAX7_Service_CreateWorkspace_Good(t *core.T) {
	service, medium := newAX7Service(t)
	workspaceID, err := service.CreateWorkspace("alice", "pass")
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.IsDir(core.Path(service.rootPath, workspaceID, "files")))
}

func TestAX7_Service_CreateWorkspace_Bad(t *core.T) {
	service, _ := newAX7Service(t)
	_, err := service.CreateWorkspace("alice", "pass")
	core.RequireNoError(t, err)
	_, err = service.CreateWorkspace("alice", "pass")
	core.AssertError(t, err)
}

func TestAX7_Service_CreateWorkspace_Ugly(t *core.T) {
	service, _ := newAX7Service(t)
	workspaceID, err := service.CreateWorkspace(" spaced user ", "pass")
	core.AssertNoError(t, err)
	core.AssertNotEmpty(t, workspaceID)
}

func TestAX7_Service_SwitchWorkspace_Good(t *core.T) {
	service, _ := newAX7Service(t)
	workspaceID, err := service.CreateWorkspace("alice", "pass")
	core.RequireNoError(t, err)
	err = service.SwitchWorkspace(workspaceID)
	core.AssertNoError(t, err)
	core.AssertEqual(t, workspaceID, service.activeWorkspaceID)
}

func TestAX7_Service_SwitchWorkspace_Bad(t *core.T) {
	service, _ := newAX7Service(t)
	err := service.SwitchWorkspace("missing")
	core.AssertError(t, err)
	core.AssertEqual(t, "", service.activeWorkspaceID)
}

func TestAX7_Service_SwitchWorkspace_Ugly(t *core.T) {
	service, _ := newAX7Service(t)
	err := service.SwitchWorkspace("../escape")
	core.AssertError(t, err)
	core.AssertEqual(t, "", service.activeWorkspaceID)
}

func TestAX7_Service_ReadWorkspaceFile_Good(t *core.T) {
	service, _ := newAX7Service(t)
	workspaceID, err := service.CreateWorkspace("alice", "pass")
	core.RequireNoError(t, err)
	core.RequireNoError(t, service.SwitchWorkspace(workspaceID))
	core.RequireNoError(t, service.WriteWorkspaceFile("note.txt", "payload"))
	got, err := service.ReadWorkspaceFile("note.txt")
	core.AssertNoError(t, err)
	core.AssertEqual(t, "payload", got)
}

func TestAX7_Service_ReadWorkspaceFile_Bad(t *core.T) {
	service, _ := newAX7Service(t)
	got, err := service.ReadWorkspaceFile("note.txt")
	core.AssertError(t, err)
	core.AssertEqual(t, "", got)
}

func TestAX7_Service_ReadWorkspaceFile_Ugly(t *core.T) {
	service, _ := newAX7Service(t)
	workspaceID, err := service.CreateWorkspace("alice", "pass")
	core.RequireNoError(t, err)
	core.RequireNoError(t, service.SwitchWorkspace(workspaceID))
	got, err := service.ReadWorkspaceFile("../keys/private.key")
	core.AssertError(t, err)
	core.AssertEqual(t, "", got)
}

func TestAX7_Service_WriteWorkspaceFile_Good(t *core.T) {
	service, _ := newAX7Service(t)
	workspaceID, err := service.CreateWorkspace("alice", "pass")
	core.RequireNoError(t, err)
	core.RequireNoError(t, service.SwitchWorkspace(workspaceID))
	err = service.WriteWorkspaceFile("note.txt", "payload")
	core.AssertNoError(t, err)
}

func TestAX7_Service_WriteWorkspaceFile_Bad(t *core.T) {
	service, _ := newAX7Service(t)
	err := service.WriteWorkspaceFile("note.txt", "payload")
	core.AssertError(t, err)
	core.AssertEqual(t, "", service.activeWorkspaceID)
}

func TestAX7_Service_WriteWorkspaceFile_Ugly(t *core.T) {
	service, _ := newAX7Service(t)
	workspaceID, err := service.CreateWorkspace("alice", "pass")
	core.RequireNoError(t, err)
	core.RequireNoError(t, service.SwitchWorkspace(workspaceID))
	err = service.WriteWorkspaceFile("../escape.txt", "payload")
	core.AssertError(t, err)
}

func TestAX7_Service_HandleWorkspaceCommand_Good(t *core.T) {
	service, _ := newAX7Service(t)
	result := service.HandleWorkspaceCommand(WorkspaceCommand{Action: WorkspaceCreateAction, Identifier: "alice", Password: "pass"})
	core.AssertTrue(t, result.OK)
	core.AssertNotEmpty(t, result.Value)
}

func TestAX7_Service_HandleWorkspaceCommand_Bad(t *core.T) {
	service, _ := newAX7Service(t)
	result := service.HandleWorkspaceCommand(WorkspaceCommand{Action: "unknown"})
	core.AssertFalse(t, result.OK)
	core.AssertNotNil(t, result.Value)
}

func TestAX7_Service_HandleWorkspaceCommand_Ugly(t *core.T) {
	service, _ := newAX7Service(t)
	result := service.HandleWorkspaceCommand(WorkspaceCommand{Action: WorkspaceCreateAction})
	core.AssertFalse(t, result.OK)
	core.AssertNotNil(t, result.Value)
}

func TestAX7_Service_HandleWorkspaceMessage_Good(t *core.T) {
	service, _ := newAX7Service(t)
	result := service.HandleWorkspaceMessage(core.New(), WorkspaceCommand{Action: WorkspaceCreateAction, Identifier: "alice", Password: "pass"})
	core.AssertTrue(t, result.OK)
	core.AssertNotEmpty(t, result.Value)
}

func TestAX7_Service_HandleWorkspaceMessage_Bad(t *core.T) {
	service, _ := newAX7Service(t)
	result := service.HandleWorkspaceMessage(core.New(), "unsupported")
	core.AssertFalse(t, result.OK)
	core.AssertNotNil(t, result.Value)
}

func TestAX7_Service_HandleWorkspaceMessage_Ugly(t *core.T) {
	service, _ := newAX7Service(t)
	result := service.HandleWorkspaceMessage(nil, WorkspaceCommand{Action: "unknown"})
	core.AssertFalse(t, result.OK)
	core.AssertNotNil(t, result.Value)
}

func TestAX7_NewWorkspace_Good(t *core.T) {
	medium := coreio.NewMemoryMedium()
	workspaceService, err := NewWorkspace(medium, "root")
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.IsDir("root"))
	core.AssertNotNil(t, workspaceService)
}

func TestAX7_NewWorkspace_Bad(t *core.T) {
	workspaceService, err := NewWorkspace(nil, "root")
	core.AssertError(t, err)
	core.AssertNil(t, workspaceService)
}

func TestAX7_NewWorkspace_Ugly(t *core.T) {
	workspaceService, err := NewWorkspace(coreio.NewMemoryMedium(), "../escape")
	core.AssertError(t, err)
	core.AssertNil(t, workspaceService)
}

func TestAX7_Workspace_CreateWorkspace_Good(t *core.T) {
	workspaceService, medium := newTestWorkspace(t)
	scoped, err := workspaceService.CreateWorkspace("alpha")
	core.AssertNoError(t, err)
	core.AssertNotNil(t, scoped)
	core.AssertTrue(t, medium.IsDir("workspaces/alpha"))
}

func TestAX7_Workspace_CreateWorkspace_Bad(t *core.T) {
	workspaceService, _ := newTestWorkspace(t)
	_, err := workspaceService.CreateWorkspace("")
	core.AssertError(t, err)
	core.AssertEqual(t, "", workspaceService.CurrentWorkspace())
}

func TestAX7_Workspace_CreateWorkspace_Ugly(t *core.T) {
	workspaceService, _ := newTestWorkspace(t)
	_, err := workspaceService.CreateWorkspace("../escape")
	core.AssertError(t, err)
	core.AssertEqual(t, "", workspaceService.CurrentWorkspace())
}

func TestAX7_Workspace_SwitchWorkspace_Good(t *core.T) {
	workspaceService, _ := newTestWorkspace(t)
	_, err := workspaceService.CreateWorkspace("alpha")
	core.RequireNoError(t, err)
	err = workspaceService.SwitchWorkspace("alpha")
	core.AssertNoError(t, err)
	core.AssertEqual(t, "alpha", workspaceService.CurrentWorkspace())
}

func TestAX7_Workspace_SwitchWorkspace_Bad(t *core.T) {
	workspaceService, _ := newTestWorkspace(t)
	err := workspaceService.SwitchWorkspace("missing")
	core.AssertError(t, err)
	core.AssertEqual(t, "", workspaceService.CurrentWorkspace())
}

func TestAX7_Workspace_SwitchWorkspace_Ugly(t *core.T) {
	workspaceService, _ := newTestWorkspace(t)
	err := workspaceService.SwitchWorkspace("../escape")
	core.AssertError(t, err)
	core.AssertEqual(t, "", workspaceService.CurrentWorkspace())
}

func TestAX7_Workspace_CurrentWorkspace_Good(t *core.T) {
	workspaceService, _ := newTestWorkspace(t)
	_, err := workspaceService.CreateWorkspace("alpha")
	core.RequireNoError(t, err)
	core.RequireNoError(t, workspaceService.SwitchWorkspace("alpha"))
	got := workspaceService.CurrentWorkspace()
	core.AssertEqual(t, "alpha", got)
}

func TestAX7_Workspace_CurrentWorkspace_Bad(t *core.T) {
	var workspaceService *Workspace
	got := workspaceService.CurrentWorkspace()
	core.AssertEqual(t, "", got)
}

func TestAX7_Workspace_CurrentWorkspace_Ugly(t *core.T) {
	workspaceService, _ := newTestWorkspace(t)
	got := workspaceService.CurrentWorkspace()
	core.AssertEqual(t, "", got)
}

func TestAX7_Workspace_ReadWorkspaceFile_Good(t *core.T) {
	workspaceService, _ := newTestWorkspace(t)
	_, err := workspaceService.CreateWorkspace("alpha")
	core.RequireNoError(t, err)
	core.RequireNoError(t, workspaceService.WriteWorkspaceFile("alpha", "note.txt", "payload"))
	got, err := workspaceService.ReadWorkspaceFile("alpha", "note.txt")
	core.AssertNoError(t, err)
	core.AssertEqual(t, "payload", got)
}

func TestAX7_Workspace_ReadWorkspaceFile_Bad(t *core.T) {
	workspaceService, _ := newTestWorkspace(t)
	got, err := workspaceService.ReadWorkspaceFile("missing", "note.txt")
	core.AssertError(t, err)
	core.AssertEqual(t, "", got)
}

func TestAX7_Workspace_ReadWorkspaceFile_Ugly(t *core.T) {
	workspaceService, _ := newTestWorkspace(t)
	_, err := workspaceService.CreateWorkspace("alpha")
	core.RequireNoError(t, err)
	got, err := workspaceService.ReadWorkspaceFile("alpha", "../escape")
	core.AssertError(t, err)
	core.AssertEqual(t, "", got)
}

func TestAX7_Workspace_WriteWorkspaceFile_Good(t *core.T) {
	workspaceService, medium := newTestWorkspace(t)
	_, err := workspaceService.CreateWorkspace("alpha")
	core.RequireNoError(t, err)
	err = workspaceService.WriteWorkspaceFile("alpha", "note.txt", "payload")
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.IsFile("workspaces/alpha/note.txt"))
}

func TestAX7_Workspace_WriteWorkspaceFile_Bad(t *core.T) {
	workspaceService, _ := newTestWorkspace(t)
	err := workspaceService.WriteWorkspaceFile("missing", "note.txt", "payload")
	core.AssertError(t, err)
	core.AssertEqual(t, "", workspaceService.CurrentWorkspace())
}

func TestAX7_Workspace_WriteWorkspaceFile_Ugly(t *core.T) {
	workspaceService, medium := newTestWorkspace(t)
	_, err := workspaceService.CreateWorkspace("alpha")
	core.RequireNoError(t, err)
	err = workspaceService.WriteWorkspaceFile("alpha", "../escape.txt", "payload")
	core.AssertError(t, err)
	core.AssertFalse(t, medium.IsFile("workspaces/escape.txt"))
}

func TestAX7_Workspace_ListWorkspaceFiles_Good(t *core.T) {
	workspaceService, _ := newTestWorkspace(t)
	_, err := workspaceService.CreateWorkspace("alpha")
	core.RequireNoError(t, err)
	core.RequireNoError(t, workspaceService.WriteWorkspaceFile("alpha", "notes/todo.txt", "payload"))
	entries, err := workspaceService.ListWorkspaceFiles("alpha", "notes")
	core.AssertNoError(t, err)
	core.AssertLen(t, entries, 1)
}

func TestAX7_Workspace_ListWorkspaceFiles_Bad(t *core.T) {
	workspaceService, _ := newTestWorkspace(t)
	entries, err := workspaceService.ListWorkspaceFiles("missing", "")
	core.AssertError(t, err)
	core.AssertNil(t, entries)
}

func TestAX7_Workspace_ListWorkspaceFiles_Ugly(t *core.T) {
	workspaceService, _ := newTestWorkspace(t)
	_, err := workspaceService.CreateWorkspace("alpha")
	core.RequireNoError(t, err)
	entries, err := workspaceService.ListWorkspaceFiles("alpha", "")
	core.AssertNoError(t, err)
	core.AssertEmpty(t, entries)
}

func TestAX7_Workspace_HandleWorkspaceCommand_Good(t *core.T) {
	workspaceService, _ := newTestWorkspace(t)
	result := workspaceService.HandleWorkspaceCommand(WorkspaceCommand{Action: WorkspaceCreateAction, Workspace: "alpha"})
	core.AssertTrue(t, result.OK)
	core.AssertNotNil(t, result.Value)
}

func TestAX7_Workspace_HandleWorkspaceCommand_Bad(t *core.T) {
	workspaceService, _ := newTestWorkspace(t)
	result := workspaceService.HandleWorkspaceCommand(WorkspaceCommand{Action: "unknown"})
	core.AssertFalse(t, result.OK)
	core.AssertNotNil(t, result.Value)
}

func TestAX7_Workspace_HandleWorkspaceCommand_Ugly(t *core.T) {
	workspaceService, _ := newTestWorkspace(t)
	result := workspaceService.HandleWorkspaceCommand(WorkspaceCommand{Action: WorkspaceCreateAction, Workspace: "../escape"})
	core.AssertFalse(t, result.OK)
	core.AssertNotNil(t, result.Value)
}

func TestAX7_Medium_Read_Good(t *core.T) {
	medium, _ := newAX7ScopedMedium(t)
	core.RequireNoError(t, medium.Write("read.txt", "payload"))
	got, err := medium.Read("read.txt")
	core.AssertNoError(t, err)
	core.AssertEqual(t, "payload", got)
}

func TestAX7_Medium_Read_Bad(t *core.T) {
	medium, _ := newAX7ScopedMedium(t)
	got, err := medium.Read("../escape")
	core.AssertError(t, err)
	core.AssertEqual(t, "", got)
}

func TestAX7_Medium_Read_Ugly(t *core.T) {
	medium, _ := newAX7ScopedMedium(t)
	got, err := medium.Read("missing.txt")
	core.AssertError(t, err)
	core.AssertEqual(t, "", got)
}

func TestAX7_Medium_Write_Good(t *core.T) {
	medium, backing := newAX7ScopedMedium(t)
	err := medium.Write("write.txt", "payload")
	core.AssertNoError(t, err)
	core.AssertTrue(t, backing.IsFile("workspaces/alpha/write.txt"))
}

func TestAX7_Medium_Write_Bad(t *core.T) {
	medium, _ := newAX7ScopedMedium(t)
	err := medium.Write("../escape", "payload")
	core.AssertError(t, err)
	core.AssertFalse(t, medium.Exists("../escape"))
}

func TestAX7_Medium_Write_Ugly(t *core.T) {
	medium, _ := newAX7ScopedMedium(t)
	err := medium.Write("empty.txt", "")
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.Exists("empty.txt"))
}

func TestAX7_Medium_WriteMode_Good(t *core.T) {
	medium, _ := newAX7ScopedMedium(t)
	err := medium.WriteMode("mode.txt", "payload", 0600)
	info, statErr := medium.Stat("mode.txt")
	core.AssertNoError(t, err)
	core.AssertNoError(t, statErr)
	core.AssertEqual(t, fs.FileMode(0600), info.Mode())
}

func TestAX7_Medium_WriteMode_Bad(t *core.T) {
	medium, _ := newAX7ScopedMedium(t)
	err := medium.WriteMode("../escape", "payload", 0600)
	core.AssertError(t, err)
	core.AssertFalse(t, medium.Exists("../escape"))
}

func TestAX7_Medium_WriteMode_Ugly(t *core.T) {
	medium, _ := newAX7ScopedMedium(t)
	err := medium.WriteMode("zero-mode.txt", "payload", 0)
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.Exists("zero-mode.txt"))
}

func TestAX7_Medium_EnsureDir_Good(t *core.T) {
	medium, _ := newAX7ScopedMedium(t)
	err := medium.EnsureDir("dir")
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.IsDir("dir"))
}

func TestAX7_Medium_EnsureDir_Bad(t *core.T) {
	medium, _ := newAX7ScopedMedium(t)
	err := medium.EnsureDir("../escape")
	core.AssertError(t, err)
	core.AssertFalse(t, medium.IsDir("../escape"))
}

func TestAX7_Medium_EnsureDir_Ugly(t *core.T) {
	medium, _ := newAX7ScopedMedium(t)
	err := medium.EnsureDir("")
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.IsDir(""))
}

func TestAX7_Medium_IsFile_Good(t *core.T) {
	medium, _ := newAX7ScopedMedium(t)
	core.RequireNoError(t, medium.Write("file.txt", "payload"))
	got := medium.IsFile("file.txt")
	core.AssertTrue(t, got)
}

func TestAX7_Medium_IsFile_Bad(t *core.T) {
	medium, _ := newAX7ScopedMedium(t)
	got := medium.IsFile("../escape")
	core.AssertFalse(t, got)
}

func TestAX7_Medium_IsFile_Ugly(t *core.T) {
	medium, _ := newAX7ScopedMedium(t)
	core.RequireNoError(t, medium.EnsureDir("dir"))
	got := medium.IsFile("dir")
	core.AssertFalse(t, got)
}

func TestAX7_Medium_Delete_Good(t *core.T) {
	medium, _ := newAX7ScopedMedium(t)
	core.RequireNoError(t, medium.Write("delete.txt", "payload"))
	err := medium.Delete("delete.txt")
	core.AssertNoError(t, err)
	core.AssertFalse(t, medium.Exists("delete.txt"))
}

func TestAX7_Medium_Delete_Bad(t *core.T) {
	medium, _ := newAX7ScopedMedium(t)
	err := medium.Delete("../escape")
	core.AssertError(t, err)
	core.AssertFalse(t, medium.Exists("../escape"))
}

func TestAX7_Medium_Delete_Ugly(t *core.T) {
	medium, _ := newAX7ScopedMedium(t)
	err := medium.Delete("missing.txt")
	core.AssertError(t, err)
	core.AssertFalse(t, medium.Exists("missing.txt"))
}

func TestAX7_Medium_DeleteAll_Good(t *core.T) {
	medium, _ := newAX7ScopedMedium(t)
	core.RequireNoError(t, medium.Write("tree/file.txt", "payload"))
	err := medium.DeleteAll("tree")
	core.AssertNoError(t, err)
	core.AssertFalse(t, medium.Exists("tree/file.txt"))
}

func TestAX7_Medium_DeleteAll_Bad(t *core.T) {
	medium, _ := newAX7ScopedMedium(t)
	err := medium.DeleteAll("../escape")
	core.AssertError(t, err)
	core.AssertFalse(t, medium.Exists("../escape"))
}

func TestAX7_Medium_DeleteAll_Ugly(t *core.T) {
	medium, _ := newAX7ScopedMedium(t)
	err := medium.DeleteAll("missing")
	core.AssertError(t, err)
	core.AssertFalse(t, medium.Exists("missing"))
}

func TestAX7_Medium_Rename_Good(t *core.T) {
	medium, _ := newAX7ScopedMedium(t)
	core.RequireNoError(t, medium.Write("old.txt", "payload"))
	err := medium.Rename("old.txt", "new.txt")
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.Exists("new.txt"))
}

func TestAX7_Medium_Rename_Bad(t *core.T) {
	medium, _ := newAX7ScopedMedium(t)
	err := medium.Rename("../escape", "new.txt")
	core.AssertError(t, err)
	core.AssertFalse(t, medium.Exists("new.txt"))
}

func TestAX7_Medium_Rename_Ugly(t *core.T) {
	medium, _ := newAX7ScopedMedium(t)
	core.RequireNoError(t, medium.Write("dir/old.txt", "payload"))
	err := medium.Rename("dir", "moved")
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.Exists("moved/old.txt"))
}

func TestAX7_Medium_List_Good(t *core.T) {
	medium, _ := newAX7ScopedMedium(t)
	core.RequireNoError(t, medium.Write("dir/file.txt", "payload"))
	entries, err := medium.List("dir")
	core.AssertNoError(t, err)
	core.AssertLen(t, entries, 1)
}

func TestAX7_Medium_List_Bad(t *core.T) {
	medium, _ := newAX7ScopedMedium(t)
	entries, err := medium.List("../escape")
	core.AssertError(t, err)
	core.AssertNil(t, entries)
}

func TestAX7_Medium_List_Ugly(t *core.T) {
	medium, _ := newAX7ScopedMedium(t)
	entries, err := medium.List("")
	core.AssertNoError(t, err)
	core.AssertEmpty(t, entries)
}

func TestAX7_Medium_Stat_Good(t *core.T) {
	medium, _ := newAX7ScopedMedium(t)
	core.RequireNoError(t, medium.Write("stat.txt", "payload"))
	info, err := medium.Stat("stat.txt")
	core.AssertNoError(t, err)
	core.AssertEqual(t, "stat.txt", info.Name())
}

func TestAX7_Medium_Stat_Bad(t *core.T) {
	medium, _ := newAX7ScopedMedium(t)
	info, err := medium.Stat("../escape")
	core.AssertError(t, err)
	core.AssertNil(t, info)
}

func TestAX7_Medium_Stat_Ugly(t *core.T) {
	medium, _ := newAX7ScopedMedium(t)
	info, err := medium.Stat("")
	core.AssertNoError(t, err)
	core.AssertTrue(t, info.IsDir())
}

func TestAX7_Medium_Open_Good(t *core.T) {
	medium, _ := newAX7ScopedMedium(t)
	core.RequireNoError(t, medium.Write("open.txt", "payload"))
	file, err := medium.Open("open.txt")
	core.AssertNoError(t, err)
	core.AssertNotNil(t, file)
	core.RequireNoError(t, file.Close())
}

func TestAX7_Medium_Open_Bad(t *core.T) {
	medium, _ := newAX7ScopedMedium(t)
	file, err := medium.Open("../escape")
	core.AssertError(t, err)
	core.AssertNil(t, file)
}

func TestAX7_Medium_Open_Ugly(t *core.T) {
	medium, _ := newAX7ScopedMedium(t)
	file, err := medium.Open("missing.txt")
	core.AssertError(t, err)
	core.AssertNil(t, file)
}

func TestAX7_Medium_Create_Good(t *core.T) {
	medium, _ := newAX7ScopedMedium(t)
	writer, err := medium.Create("create.txt")
	core.RequireNoError(t, err)
	_, writeErr := writer.Write([]byte("payload"))
	core.AssertNoError(t, writeErr)
	core.AssertNoError(t, writer.Close())
}

func TestAX7_Medium_Create_Bad(t *core.T) {
	medium, _ := newAX7ScopedMedium(t)
	writer, err := medium.Create("../escape")
	core.AssertError(t, err)
	core.AssertNil(t, writer)
}

func TestAX7_Medium_Create_Ugly(t *core.T) {
	medium, _ := newAX7ScopedMedium(t)
	writer, err := medium.Create("nested/create.txt")
	core.RequireNoError(t, err)
	core.AssertNoError(t, writer.Close())
	core.AssertTrue(t, medium.Exists("nested/create.txt"))
}

func TestAX7_Medium_Append_Good(t *core.T) {
	medium, _ := newAX7ScopedMedium(t)
	core.RequireNoError(t, medium.Write("append.txt", "a"))
	writer, err := medium.Append("append.txt")
	core.RequireNoError(t, err)
	_, writeErr := writer.Write([]byte("b"))
	core.RequireNoError(t, writeErr)
	core.RequireNoError(t, writer.Close())
}

func TestAX7_Medium_Append_Bad(t *core.T) {
	medium, _ := newAX7ScopedMedium(t)
	writer, err := medium.Append("../escape")
	core.AssertError(t, err)
	core.AssertNil(t, writer)
}

func TestAX7_Medium_Append_Ugly(t *core.T) {
	medium, _ := newAX7ScopedMedium(t)
	writer, err := medium.Append("new.txt")
	core.RequireNoError(t, err)
	_, writeErr := writer.Write([]byte("created"))
	core.RequireNoError(t, writeErr)
	core.RequireNoError(t, writer.Close())
}

func TestAX7_Medium_ReadStream_Good(t *core.T) {
	medium, _ := newAX7ScopedMedium(t)
	core.RequireNoError(t, medium.Write("stream.txt", "payload"))
	reader, err := medium.ReadStream("stream.txt")
	core.RequireNoError(t, err)
	defer reader.Close()
	data, readErr := goio.ReadAll(reader)
	core.AssertNoError(t, readErr)
	core.AssertEqual(t, "payload", string(data))
}

func TestAX7_Medium_ReadStream_Bad(t *core.T) {
	medium, _ := newAX7ScopedMedium(t)
	reader, err := medium.ReadStream("../escape")
	core.AssertError(t, err)
	core.AssertNil(t, reader)
}

func TestAX7_Medium_ReadStream_Ugly(t *core.T) {
	medium, _ := newAX7ScopedMedium(t)
	reader, err := medium.ReadStream("missing.txt")
	core.AssertError(t, err)
	core.AssertNil(t, reader)
}

func TestAX7_Medium_WriteStream_Good(t *core.T) {
	medium, _ := newAX7ScopedMedium(t)
	writer, err := medium.WriteStream("stream.txt")
	core.RequireNoError(t, err)
	_, writeErr := writer.Write([]byte("payload"))
	core.AssertNoError(t, writeErr)
	core.AssertNoError(t, writer.Close())
}

func TestAX7_Medium_WriteStream_Bad(t *core.T) {
	medium, _ := newAX7ScopedMedium(t)
	writer, err := medium.WriteStream("../escape")
	core.AssertError(t, err)
	core.AssertNil(t, writer)
}

func TestAX7_Medium_WriteStream_Ugly(t *core.T) {
	medium, _ := newAX7ScopedMedium(t)
	writer, err := medium.WriteStream("nested/stream.txt")
	core.RequireNoError(t, err)
	core.AssertNoError(t, writer.Close())
	core.AssertTrue(t, medium.Exists("nested/stream.txt"))
}

func TestAX7_Medium_Exists_Good(t *core.T) {
	medium, _ := newAX7ScopedMedium(t)
	core.RequireNoError(t, medium.Write("exists.txt", "payload"))
	got := medium.Exists("exists.txt")
	core.AssertTrue(t, got)
}

func TestAX7_Medium_Exists_Bad(t *core.T) {
	medium, _ := newAX7ScopedMedium(t)
	got := medium.Exists("../escape")
	core.AssertFalse(t, got)
}

func TestAX7_Medium_Exists_Ugly(t *core.T) {
	medium, _ := newAX7ScopedMedium(t)
	got := medium.Exists("")
	core.AssertTrue(t, got)
}

func TestAX7_Medium_IsDir_Good(t *core.T) {
	medium, _ := newAX7ScopedMedium(t)
	core.RequireNoError(t, medium.EnsureDir("dir"))
	got := medium.IsDir("dir")
	core.AssertTrue(t, got)
}

func TestAX7_Medium_IsDir_Bad(t *core.T) {
	medium, _ := newAX7ScopedMedium(t)
	got := medium.IsDir("../escape")
	core.AssertFalse(t, got)
}

func TestAX7_Medium_IsDir_Ugly(t *core.T) {
	medium, _ := newAX7ScopedMedium(t)
	core.RequireNoError(t, medium.Write("file.txt", "payload"))
	got := medium.IsDir("file.txt")
	core.AssertFalse(t, got)
}
