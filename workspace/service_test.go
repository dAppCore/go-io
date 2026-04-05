package workspace

import (
	"io/fs"
	"testing"

	core "dappco.re/go/core"
	coreio "dappco.re/go/core/io"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func newWorkspaceService(t *testing.T) (*Service, string) {
	t.Helper()

	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	service, err := New(Options{KeyPairProvider: testKeyPairProvider{privateKey: "private-key"}})
	require.NoError(t, err)
	return service, tempHome
}

func TestService_New_MissingKeyPairProvider_Bad(t *testing.T) {
	_, err := New(Options{})
	require.Error(t, err)
}

func TestService_New_CustomRootPathAndMedium_Good(t *testing.T) {
	medium := coreio.NewMemoryMedium()
	rootPath := core.Path(t.TempDir(), "custom", "workspaces")

	service, err := New(Options{
		KeyPairProvider: testKeyPairProvider{privateKey: "private-key"},
		RootPath:        rootPath,
		Medium:          medium,
	})
	require.NoError(t, err)
	assert.Equal(t, rootPath, service.rootPath)
	assert.Same(t, medium, service.medium)

	workspaceID, err := service.CreateWorkspace("custom-user", "pass123")
	require.NoError(t, err)
	assert.NotEmpty(t, workspaceID)

	expectedWorkspacePath := core.Path(rootPath, workspaceID)
	assert.True(t, medium.IsDir(rootPath))
	assert.True(t, medium.IsDir(core.Path(expectedWorkspacePath, "keys")))
	assert.True(t, medium.Exists(core.Path(expectedWorkspacePath, "keys", "private.key")))
}

func TestService_WorkspaceFileRoundTrip_Good(t *testing.T) {
	service, tempHome := newWorkspaceService(t)

	workspaceID, err := service.CreateWorkspace("test-user", "pass123")
	require.NoError(t, err)
	assert.NotEmpty(t, workspaceID)

	workspacePath := core.Path(tempHome, ".core", "workspaces", workspaceID)
	assert.DirExists(t, workspacePath)
	assert.DirExists(t, core.Path(workspacePath, "keys"))
	assert.FileExists(t, core.Path(workspacePath, "keys", "private.key"))

	err = service.SwitchWorkspace(workspaceID)
	require.NoError(t, err)
	assert.Equal(t, workspaceID, service.activeWorkspaceID)

	err = service.WriteWorkspaceFile("secret.txt", "top secret info")
	require.NoError(t, err)

	got, err := service.ReadWorkspaceFile("secret.txt")
	require.NoError(t, err)
	assert.Equal(t, "top secret info", got)
}

func TestService_SwitchWorkspace_TraversalBlocked_Bad(t *testing.T) {
	service, tempHome := newWorkspaceService(t)

	outside := core.Path(tempHome, ".core", "escaped")
	require.NoError(t, service.medium.EnsureDir(outside))

	err := service.SwitchWorkspace("../escaped")
	require.Error(t, err)
	assert.Empty(t, service.activeWorkspaceID)
}

func TestService_WriteWorkspaceFile_TraversalBlocked_Bad(t *testing.T) {
	service, tempHome := newWorkspaceService(t)

	workspaceID, err := service.CreateWorkspace("test-user", "pass123")
	require.NoError(t, err)
	require.NoError(t, service.SwitchWorkspace(workspaceID))

	keyPath := core.Path(tempHome, ".core", "workspaces", workspaceID, "keys", "private.key")
	before, err := service.medium.Read(keyPath)
	require.NoError(t, err)

	err = service.WriteWorkspaceFile("../keys/private.key", "hijack")
	require.Error(t, err)

	after, err := service.medium.Read(keyPath)
	require.NoError(t, err)
	assert.Equal(t, before, after)

	_, err = service.ReadWorkspaceFile("../keys/private.key")
	require.Error(t, err)
}

func TestService_JoinPathWithinRoot_DefaultSeparator_Good(t *testing.T) {
	t.Setenv("CORE_PATH_SEPARATOR", "")

	path, err := joinPathWithinRoot("/tmp/workspaces", "../workspaces2")
	require.Error(t, err)
	assert.ErrorIs(t, err, fs.ErrPermission)
	assert.Empty(t, path)
}

func TestService_New_IPCAutoRegistration_Good(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	c := core.New()
	service, err := New(Options{
		KeyPairProvider: testKeyPairProvider{privateKey: "private-key"},
		Core:            c,
	})
	require.NoError(t, err)

	// Create a workspace directly, then switch via the Core IPC bus.
	workspaceID, err := service.CreateWorkspace("ipc-bus-user", "pass789")
	require.NoError(t, err)

	// Dispatching workspace.switch via ACTION must reach the auto-registered handler.
	c.ACTION(WorkspaceCommand{
		Action:      WorkspaceSwitchAction,
		WorkspaceID: workspaceID,
	})
	assert.Equal(t, workspaceID, service.activeWorkspaceID)
}

func TestService_New_IPCCreate_Good(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	c := core.New()
	service, err := New(Options{
		KeyPairProvider: testKeyPairProvider{privateKey: "private-key"},
		Core:            c,
	})
	require.NoError(t, err)

	// workspace.create dispatched via the bus must create the workspace on the medium.
	c.ACTION(WorkspaceCommand{
		Action:     WorkspaceCreateAction,
		Identifier: "ipc-create-user",
		Password:   "pass123",
	})

	// A duplicate create must fail — proves the first create succeeded.
	_, err = service.CreateWorkspace("ipc-create-user", "pass123")
	require.Error(t, err)
}

func TestService_New_NoCoreOption_NoRegistration_Good(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	// Without Core in Options, New must succeed and no IPC handler is registered.
	service, err := New(Options{
		KeyPairProvider: testKeyPairProvider{privateKey: "private-key"},
	})
	require.NoError(t, err)
	assert.NotNil(t, service)
}

func TestService_HandleWorkspaceMessage_Command_Good(t *testing.T) {
	service, _ := newWorkspaceService(t)

	create := service.HandleWorkspaceMessage(core.New(), WorkspaceCommand{
		Action:     WorkspaceCreateAction,
		Identifier: "ipc-user",
		Password:   "pass123",
	})
	assert.True(t, create.OK)

	workspaceID, ok := create.Value.(string)
	require.True(t, ok)
	require.NotEmpty(t, workspaceID)

	switchResult := service.HandleWorkspaceMessage(core.New(), WorkspaceCommand{
		Action:      WorkspaceSwitchAction,
		WorkspaceID: workspaceID,
	})
	assert.True(t, switchResult.OK)
	assert.Equal(t, workspaceID, service.activeWorkspaceID)

	unknownAction := service.HandleWorkspaceCommand(WorkspaceCommand{Action: "noop"})
	assert.False(t, unknownAction.OK)

	unknown := service.HandleWorkspaceMessage(core.New(), "noop")
	assert.False(t, unknown.OK)
}
