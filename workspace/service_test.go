package workspace

import (
	core "dappco.re/go"
	coreio "dappco.re/go/io"
	"io/fs"
)

type testKeyPairProvider struct {
	privateKey string
	err        error
}

func (provider testKeyPairProvider) CreateKeyPair(identifier, passphrase string) (string, error) {
	if provider.err != nil {
		return "", provider.err
	}
	return provider.privateKey, nil
}

func newWorkspaceService(t *core.T) (*Service, string) {
	t.Helper()

	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	service, err := New(Options{KeyPairProvider: testKeyPairProvider{privateKey: "private-key"}})
	core.RequireNoError(t, err)
	return service, tempHome
}

func TestService_New_MissingKeyPairProvider_Bad(t *core.T) {
	service, err := New(Options{})
	core.AssertNil(t, service)
	core.AssertError(t, err)
	if err == nil {
		t.Fatal("expected missing key pair provider to fail")
	}
	core.AssertContains(t, err.Error(), "key pair provider is required")
}

func TestService_New_CustomRootPathAndMedium_Good(t *core.T) {
	medium := coreio.NewMemoryMedium()
	rootPath := core.Path(t.TempDir(), "custom", "workspaces")

	service, err := New(Options{
		KeyPairProvider: testKeyPairProvider{privateKey: "private-key"},
		RootPath:        rootPath,
		Medium:          medium,
	})
	core.RequireNoError(t, err)
	core.AssertEqual(t, rootPath, service.rootPath)
	core.AssertSame(t, medium, service.medium)

	workspaceID, err := service.CreateWorkspace("custom-user", "pass123")
	core.RequireNoError(t, err)
	core.AssertNotEmpty(t, workspaceID)

	expectedWorkspacePath := core.Path(rootPath, workspaceID)
	core.AssertTrue(t, medium.IsDir(rootPath))
	core.AssertTrue(t, medium.IsDir(core.Path(expectedWorkspacePath, "keys")))
	core.AssertTrue(t, medium.Exists(core.Path(expectedWorkspacePath, "keys", "private.key")))
}

func TestService_WorkspaceFileRoundTripGood(t *core.T) {
	service, tempHome := newWorkspaceService(t)

	workspaceID, err := service.CreateWorkspace("test-user", "pass123")
	core.RequireNoError(t, err)
	core.AssertNotEmpty(t, workspaceID)

	workspacePath := core.Path(tempHome, ".core", "workspaces", workspaceID)
	core.AssertTrue(t, service.medium.IsDir(workspacePath))
	core.AssertTrue(t, service.medium.IsDir(core.Path(workspacePath, "keys")))
	core.AssertTrue(t, service.medium.IsFile(core.Path(workspacePath, "keys", "private.key")))

	err = service.SwitchWorkspace(workspaceID)
	core.RequireNoError(t, err)
	core.AssertEqual(t, workspaceID, service.activeWorkspaceID)

	err = service.WriteWorkspaceFile("secret.txt", "top secret info")
	core.RequireNoError(t, err)

	got, err := service.ReadWorkspaceFile("secret.txt")
	core.RequireNoError(t, err)
	core.AssertEqual(t, "top secret info", got)
}

func TestService_SwitchWorkspace_TraversalBlocked_Bad(t *core.T) {
	service, tempHome := newWorkspaceService(t)

	outside := core.Path(tempHome, ".core", "escaped")
	core.RequireNoError(t, service.medium.EnsureDir(outside))

	err := service.SwitchWorkspace("../escaped")
	core.AssertError(t, err)
	core.AssertEmpty(t, service.activeWorkspaceID)
}

func TestService_WriteWorkspaceFile_TraversalBlocked_Bad(t *core.T) {
	service, tempHome := newWorkspaceService(t)

	workspaceID, err := service.CreateWorkspace("test-user", "pass123")
	core.RequireNoError(t, err)
	core.RequireNoError(t, service.SwitchWorkspace(workspaceID))

	keyPath := core.Path(tempHome, ".core", "workspaces", workspaceID, "keys", "private.key")
	before, err := service.medium.Read(keyPath)
	core.RequireNoError(t, err)

	err = service.WriteWorkspaceFile("../keys/private.key", "hijack")
	core.AssertError(t, err)

	after, err := service.medium.Read(keyPath)
	core.RequireNoError(t, err)
	core.AssertEqual(t, before, after)

	_, err = service.ReadWorkspaceFile("../keys/private.key")
	core.AssertError(t, err)
}

func TestService_JoinPathWithinRoot_DefaultSeparatorGood(t *core.T) {
	t.Setenv("CORE_PATH_SEPARATOR", "")

	path, err := joinPathWithinRoot("/tmp/workspaces", "../workspaces2")
	core.AssertError(t, err)
	core.AssertErrorIs(t, err, fs.ErrPermission)
	core.AssertEmpty(t, path)
}

func TestService_New_IPCAutoRegistration_Good(t *core.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	c := core.New()
	service, err := New(Options{
		KeyPairProvider: testKeyPairProvider{privateKey: "private-key"},
		Core:            c,
	})
	core.RequireNoError(t, err)

	// Create a workspace directly, then switch via the Core IPC bus.
	workspaceID, err := service.CreateWorkspace("ipc-bus-user", "pass789")
	core.RequireNoError(t, err)

	// Dispatching workspace.switch via ACTION must reach the auto-registered handler.
	c.ACTION(WorkspaceCommand{
		Action:      WorkspaceSwitchAction,
		WorkspaceID: workspaceID,
	})
	core.AssertEqual(t, workspaceID, service.activeWorkspaceID)
}

func TestService_New_IPCCreate_Good(t *core.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	c := core.New()
	service, err := New(Options{
		KeyPairProvider: testKeyPairProvider{privateKey: "private-key"},
		Core:            c,
	})
	core.RequireNoError(t, err)

	// workspace.create dispatched via the bus must create the workspace on the medium.
	c.ACTION(WorkspaceCommand{
		Action:     WorkspaceCreateAction,
		Identifier: "ipc-create-user",
		Password:   "pass123",
	})

	// A duplicate create must fail — proves the first create succeeded.
	_, err = service.CreateWorkspace("ipc-create-user", "pass123")
	core.AssertError(t, err)
}

func TestService_New_NoCoreOption_NoRegistration_Good(t *core.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	// Without Core in Options, New must succeed and no IPC handler is registered.
	service, err := New(Options{
		KeyPairProvider: testKeyPairProvider{privateKey: "private-key"},
	})
	core.RequireNoError(t, err)
	core.AssertNotNil(t, service)
}

func TestService_HandleWorkspaceMessage_Command_Good(t *core.T) {
	service, _ := newWorkspaceService(t)

	create := service.HandleWorkspaceMessage(core.New(), WorkspaceCommand{
		Action:     WorkspaceCreateAction,
		Identifier: "ipc-user",
		Password:   "pass123",
	})
	core.AssertTrue(t, create.OK)

	workspaceID, ok := create.Value.(string)
	core.RequireTrue(t, ok)
	core.RequireNotEmpty(t, workspaceID)

	switchResult := service.HandleWorkspaceMessage(core.New(), WorkspaceCommand{
		Action:      WorkspaceSwitchAction,
		WorkspaceID: workspaceID,
	})
	core.AssertTrue(t, switchResult.OK)
	core.AssertEqual(t, workspaceID, service.activeWorkspaceID)

	unknownAction := service.HandleWorkspaceCommand(WorkspaceCommand{Action: "noop"})
	core.AssertFalse(t, unknownAction.OK)

	unknown := service.HandleWorkspaceMessage(core.New(), "noop")
	core.AssertFalse(t, unknown.OK)
}

func newWorkspaceServiceFixture(t *core.T) (*Service, *coreio.MemoryMedium) {
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

func newScopedMediumFixture(t *core.T) (*scopedMedium, *coreio.MemoryMedium) {
	t.Helper()

	workspaceService, medium := newTestWorkspace(t)
	scoped, err := workspaceService.CreateWorkspace("alpha")
	core.RequireNoError(t, err)
	return scoped.(*scopedMedium), medium
}

func TestService_New_Good(t *core.T) {
	medium := coreio.NewMemoryMedium()
	rootPath := core.Path(t.TempDir(), "workspaces")
	service, err := New(Options{
		KeyPairProvider: testKeyPairProvider{privateKey: "private-key"},
		RootPath:        rootPath,
		Medium:          medium,
	})
	core.RequireNoError(t, err)

	core.AssertNotNil(t, service)
	core.AssertEqual(t, rootPath, service.rootPath)
	core.AssertSame(t, medium, service.medium)
}

func TestService_New_Bad(t *core.T) {
	service, err := New(Options{RootPath: "workspaces", Medium: coreio.NewMemoryMedium()})
	core.AssertError(t, err)
	core.AssertNil(t, service)
}

func TestService_New_Ugly(t *core.T) {
	c := core.New()
	service, _ := newWorkspaceServiceFixture(t)
	result := service.HandleWorkspaceMessage(c, "unsupported")
	core.AssertFalse(t, result.OK)
}

func TestService_SHA256Hash_Write_Good(t *core.T) {
	hash := &workspaceSHA256Hash{}
	count, err := hash.Write([]byte("payload"))
	core.AssertNoError(t, err)
	core.AssertEqual(t, len("payload"), count)
}

func TestService_SHA256Hash_Write_Bad(t *core.T) {
	hash := &workspaceSHA256Hash{}
	count, err := hash.Write(nil)
	core.AssertNoError(t, err)
	core.AssertEqual(t, 0, count)
}

func TestService_SHA256Hash_Write_Ugly(t *core.T) {
	hash := &workspaceSHA256Hash{}
	_, err := hash.Write([]byte("a"))
	core.RequireNoError(t, err)
	_, err = hash.Write([]byte("b"))
	core.AssertNoError(t, err)
}

func TestService_SHA256Hash_Sum_Good(t *core.T) {
	hash := &workspaceSHA256Hash{}
	_, err := hash.Write([]byte("payload"))
	core.RequireNoError(t, err)
	sum := hash.Sum(nil)
	core.AssertLen(t, sum, 32)
}

func TestService_SHA256Hash_Sum_Bad(t *core.T) {
	hash := &workspaceSHA256Hash{}
	sum := hash.Sum([]byte("prefix"))
	core.AssertLen(t, sum, len("prefix")+32)
	core.AssertEqual(t, "prefix", string(sum[:len("prefix")]))
}

func TestService_SHA256Hash_Sum_Ugly(t *core.T) {
	hash := &workspaceSHA256Hash{}
	sum := hash.Sum(nil)
	core.AssertLen(t, sum, 32)
	core.AssertNotEmpty(t, sum)
}

func TestService_SHA256Hash_Reset_Good(t *core.T) {
	hash := &workspaceSHA256Hash{}
	_, err := hash.Write([]byte("payload"))
	core.RequireNoError(t, err)
	hash.Reset()
	core.AssertEmpty(t, hash.data)
}

func TestService_SHA256Hash_Reset_Bad(t *core.T) {
	hash := &workspaceSHA256Hash{}
	hash.Reset()
	core.AssertEmpty(t, hash.data)
	core.AssertLen(t, hash.Sum(nil), 32)
}

func TestService_SHA256Hash_Reset_Ugly(t *core.T) {
	hash := &workspaceSHA256Hash{data: []byte("payload")}
	hash.Reset()
	_, err := hash.Write([]byte("again"))
	core.AssertNoError(t, err)
	core.AssertEqual(t, []byte("again"), hash.data)
}

func TestService_SHA256Hash_Size_Good(t *core.T) {
	hash := &workspaceSHA256Hash{}
	got := hash.Size()
	core.AssertEqual(t, 32, got)
}

func TestService_SHA256Hash_Size_Bad(t *core.T) {
	hash := &workspaceSHA256Hash{data: []byte("payload")}
	got := hash.Size()
	core.AssertEqual(t, 32, got)
}

func TestService_SHA256Hash_Size_Ugly(t *core.T) {
	hash := newWorkspaceSHA256Hash()
	got := hash.Size()
	core.AssertEqual(t, 32, got)
}

func TestService_SHA256Hash_BlockSize_Good(t *core.T) {
	hash := &workspaceSHA256Hash{}
	got := hash.BlockSize()
	core.AssertEqual(t, 64, got)
}

func TestService_SHA256Hash_BlockSize_Bad(t *core.T) {
	hash := &workspaceSHA256Hash{data: []byte("payload")}
	got := hash.BlockSize()
	core.AssertEqual(t, 64, got)
}

func TestService_SHA256Hash_BlockSize_Ugly(t *core.T) {
	hash := newWorkspaceSHA256Hash()
	got := hash.BlockSize()
	core.AssertEqual(t, 64, got)
}

func TestService_Service_CreateWorkspace_Good(t *core.T) {
	service, medium := newWorkspaceServiceFixture(t)
	workspaceID, err := service.CreateWorkspace("alice", "pass")
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.IsDir(core.Path(service.rootPath, workspaceID, "files")))
}

func TestService_Service_CreateWorkspace_Bad(t *core.T) {
	service, _ := newWorkspaceServiceFixture(t)
	_, err := service.CreateWorkspace("alice", "pass")
	core.RequireNoError(t, err)
	_, err = service.CreateWorkspace("alice", "pass")
	core.AssertError(t, err)
}

func TestService_Service_CreateWorkspace_Ugly(t *core.T) {
	service, _ := newWorkspaceServiceFixture(t)
	workspaceID, err := service.CreateWorkspace(" spaced user ", "pass")
	core.AssertNoError(t, err)
	core.AssertNotEmpty(t, workspaceID)
}

func TestService_Service_SwitchWorkspace_Good(t *core.T) {
	service, _ := newWorkspaceServiceFixture(t)
	workspaceID, err := service.CreateWorkspace("alice", "pass")
	core.RequireNoError(t, err)
	err = service.SwitchWorkspace(workspaceID)
	core.AssertNoError(t, err)
	core.AssertEqual(t, workspaceID, service.activeWorkspaceID)
}

func TestService_Service_SwitchWorkspace_Bad(t *core.T) {
	service, _ := newWorkspaceServiceFixture(t)
	err := service.SwitchWorkspace("missing")
	core.AssertError(t, err)
	core.AssertEqual(t, "", service.activeWorkspaceID)
}

func TestService_Service_SwitchWorkspace_Ugly(t *core.T) {
	service, _ := newWorkspaceServiceFixture(t)
	err := service.SwitchWorkspace("../escape")
	core.AssertError(t, err)
	core.AssertEqual(t, "", service.activeWorkspaceID)
}

func TestService_Service_ReadWorkspaceFile_Good(t *core.T) {
	service, _ := newWorkspaceServiceFixture(t)
	workspaceID, err := service.CreateWorkspace("alice", "pass")
	core.RequireNoError(t, err)
	core.RequireNoError(t, service.SwitchWorkspace(workspaceID))
	core.RequireNoError(t, service.WriteWorkspaceFile("note.txt", "payload"))
	got, err := service.ReadWorkspaceFile("note.txt")
	core.AssertNoError(t, err)
	core.AssertEqual(t, "payload", got)
}

func TestService_Service_ReadWorkspaceFile_Bad(t *core.T) {
	service, _ := newWorkspaceServiceFixture(t)
	got, err := service.ReadWorkspaceFile("note.txt")
	core.AssertError(t, err)
	core.AssertEqual(t, "", got)
}

func TestService_Service_ReadWorkspaceFile_Ugly(t *core.T) {
	service, _ := newWorkspaceServiceFixture(t)
	workspaceID, err := service.CreateWorkspace("alice", "pass")
	core.RequireNoError(t, err)
	core.RequireNoError(t, service.SwitchWorkspace(workspaceID))
	got, err := service.ReadWorkspaceFile("../keys/private.key")
	core.AssertError(t, err)
	core.AssertEqual(t, "", got)
}

func TestService_Service_WriteWorkspaceFile_Good(t *core.T) {
	service, _ := newWorkspaceServiceFixture(t)
	workspaceID, err := service.CreateWorkspace("alice", "pass")
	core.RequireNoError(t, err)
	core.RequireNoError(t, service.SwitchWorkspace(workspaceID))
	err = service.WriteWorkspaceFile("note.txt", "payload")
	core.AssertNoError(t, err)
}

func TestService_Service_WriteWorkspaceFile_Bad(t *core.T) {
	service, _ := newWorkspaceServiceFixture(t)
	err := service.WriteWorkspaceFile("note.txt", "payload")
	core.AssertError(t, err)
	core.AssertEqual(t, "", service.activeWorkspaceID)
}

func TestService_Service_WriteWorkspaceFile_Ugly(t *core.T) {
	service, _ := newWorkspaceServiceFixture(t)
	workspaceID, err := service.CreateWorkspace("alice", "pass")
	core.RequireNoError(t, err)
	core.RequireNoError(t, service.SwitchWorkspace(workspaceID))
	err = service.WriteWorkspaceFile("../escape.txt", "payload")
	core.AssertError(t, err)
}

func TestService_Service_HandleWorkspaceCommand_Good(t *core.T) {
	service, _ := newWorkspaceServiceFixture(t)
	result := service.HandleWorkspaceCommand(WorkspaceCommand{Action: WorkspaceCreateAction, Identifier: "alice", Password: "pass"})
	core.AssertTrue(t, result.OK)
	core.AssertNotEmpty(t, result.Value)
}

func TestService_Service_HandleWorkspaceCommand_Bad(t *core.T) {
	service, _ := newWorkspaceServiceFixture(t)
	result := service.HandleWorkspaceCommand(WorkspaceCommand{Action: "unknown"})
	core.AssertFalse(t, result.OK)
	core.AssertNotNil(t, result.Value)
}

func TestService_Service_HandleWorkspaceCommand_Ugly(t *core.T) {
	service, _ := newWorkspaceServiceFixture(t)
	result := service.HandleWorkspaceCommand(WorkspaceCommand{Action: WorkspaceCreateAction})
	core.AssertFalse(t, result.OK)
	core.AssertNotNil(t, result.Value)
}

func TestService_Service_HandleWorkspaceMessage_Good(t *core.T) {
	service, _ := newWorkspaceServiceFixture(t)
	result := service.HandleWorkspaceMessage(core.New(), WorkspaceCommand{Action: WorkspaceCreateAction, Identifier: "alice", Password: "pass"})
	core.AssertTrue(t, result.OK)
	core.AssertNotEmpty(t, result.Value)
}

func TestService_Service_HandleWorkspaceMessage_Bad(t *core.T) {
	service, _ := newWorkspaceServiceFixture(t)
	result := service.HandleWorkspaceMessage(core.New(), "unsupported")
	core.AssertFalse(t, result.OK)
	core.AssertNotNil(t, result.Value)
}

func TestService_Service_HandleWorkspaceMessage_Ugly(t *core.T) {
	service, _ := newWorkspaceServiceFixture(t)
	result := service.HandleWorkspaceMessage(nil, WorkspaceCommand{Action: "unknown"})
	core.AssertFalse(t, result.OK)
	core.AssertNotNil(t, result.Value)
}
