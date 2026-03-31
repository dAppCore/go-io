package workspace

import (
	"testing"

	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubKeyPairProvider struct {
	key string
	err error
}

func (provider stubKeyPairProvider) CreateKeyPair(_, _ string) (string, error) {
	if provider.err != nil {
		return "", provider.err
	}
	return provider.key, nil
}

func newTestService(t *testing.T) (*Service, string) {
	t.Helper()

	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	service, err := New(Options{KeyPairProvider: stubKeyPairProvider{key: "private-key"}})
	require.NoError(t, err)
	return service, tempHome
}

func TestService_New_MissingKeyPairProvider_Bad(t *testing.T) {
	_, err := New(Options{})
	require.Error(t, err)
}

func TestService_WorkspaceFileRoundTrip_Good(t *testing.T) {
	service, tempHome := newTestService(t)

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
	service, tempHome := newTestService(t)

	outside := core.Path(tempHome, ".core", "escaped")
	require.NoError(t, service.medium.EnsureDir(outside))

	err := service.SwitchWorkspace("../escaped")
	require.Error(t, err)
	assert.Empty(t, service.activeWorkspaceID)
}

func TestService_WriteWorkspaceFile_TraversalBlocked_Bad(t *testing.T) {
	service, tempHome := newTestService(t)

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

func TestService_HandleWorkspaceMessage_Good(t *testing.T) {
	service, _ := newTestService(t)

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

	legacyCreate := service.HandleWorkspaceMessage(core.New(), map[string]any{
		"action":     WorkspaceCreateAction,
		"identifier": "legacy-user",
		"password":   "pass123",
	})
	assert.True(t, legacyCreate.OK)

	legacyWorkspaceID, ok := legacyCreate.Value.(string)
	require.True(t, ok)
	require.NotEmpty(t, legacyWorkspaceID)

	legacySwitch := service.HandleWorkspaceMessage(core.New(), WorkspaceCommand{
		Action:      WorkspaceSwitchAction,
		WorkspaceID: legacyWorkspaceID,
	})
	assert.True(t, legacySwitch.OK)
	assert.Equal(t, legacyWorkspaceID, service.activeWorkspaceID)

	rejectedLegacySwitch := service.HandleWorkspaceMessage(core.New(), map[string]any{
		"action": WorkspaceSwitchAction,
		"name":   workspaceID,
	})
	assert.False(t, rejectedLegacySwitch.OK)
	assert.Equal(t, legacyWorkspaceID, service.activeWorkspaceID)

	failedSwitch := service.HandleWorkspaceMessage(core.New(), map[string]any{
		"action":      WorkspaceSwitchAction,
		"workspaceID": "missing",
	})
	assert.False(t, failedSwitch.OK)

	unknown := service.HandleWorkspaceMessage(core.New(), "noop")
	assert.True(t, unknown.OK)
}
