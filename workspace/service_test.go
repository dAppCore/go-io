package workspace

import (
	"testing"

	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubCryptProvider struct {
	key string
	err error
}

func (s stubCryptProvider) CreateKeyPair(_, _ string) (string, error) {
	if s.err != nil {
		return "", s.err
	}
	return s.key, nil
}

func newTestService(t *testing.T) (*Service, string) {
	t.Helper()

	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	svc, err := New(Options{Core: core.New(), CryptProvider: stubCryptProvider{key: "private-key"}})
	require.NoError(t, err)
	return svc, tempHome
}

func TestService_New_MissingCryptProvider_Bad(t *testing.T) {
	_, err := New(Options{Core: core.New()})
	require.Error(t, err)
}

func TestService_Workspace_RoundTrip_Good(t *testing.T) {
	s, tempHome := newTestService(t)

	workspaceID, err := s.CreateWorkspace("test-user", "pass123")
	require.NoError(t, err)
	assert.NotEmpty(t, workspaceID)

	workspacePath := core.Path(tempHome, ".core", "workspaces", workspaceID)
	assert.DirExists(t, workspacePath)
	assert.DirExists(t, core.Path(workspacePath, "keys"))
	assert.FileExists(t, core.Path(workspacePath, "keys", "private.key"))

	err = s.SwitchWorkspace(workspaceID)
	require.NoError(t, err)
	assert.Equal(t, workspaceID, s.activeWorkspaceID)

	err = s.WorkspaceFileSet("secret.txt", "top secret info")
	require.NoError(t, err)

	got, err := s.WorkspaceFileGet("secret.txt")
	require.NoError(t, err)
	assert.Equal(t, "top secret info", got)
}

func TestService_SwitchWorkspace_TraversalBlocked_Bad(t *testing.T) {
	s, tempHome := newTestService(t)

	outside := core.Path(tempHome, ".core", "escaped")
	require.NoError(t, s.medium.EnsureDir(outside))

	err := s.SwitchWorkspace("../escaped")
	require.Error(t, err)
	assert.Empty(t, s.activeWorkspaceID)
}

func TestService_WorkspaceFileSet_TraversalBlocked_Bad(t *testing.T) {
	s, tempHome := newTestService(t)

	workspaceID, err := s.CreateWorkspace("test-user", "pass123")
	require.NoError(t, err)
	require.NoError(t, s.SwitchWorkspace(workspaceID))

	keyPath := core.Path(tempHome, ".core", "workspaces", workspaceID, "keys", "private.key")
	before, err := s.medium.Read(keyPath)
	require.NoError(t, err)

	err = s.WorkspaceFileSet("../keys/private.key", "hijack")
	require.Error(t, err)

	after, err := s.medium.Read(keyPath)
	require.NoError(t, err)
	assert.Equal(t, before, after)

	_, err = s.WorkspaceFileGet("../keys/private.key")
	require.Error(t, err)
}

func TestService_HandleWorkspaceMessage_Good(t *testing.T) {
	s, _ := newTestService(t)

	create := s.HandleWorkspaceMessage(core.New(), WorkspaceCommand{
		Action:     WorkspaceCreateAction,
		Identifier: "ipc-user",
		Password:   "pass123",
	})
	assert.True(t, create.OK)

	workspaceID, ok := create.Value.(string)
	require.True(t, ok)
	require.NotEmpty(t, workspaceID)

	switchResult := s.HandleWorkspaceMessage(core.New(), WorkspaceCommand{
		Action:      WorkspaceSwitchAction,
		WorkspaceID: workspaceID,
	})
	assert.True(t, switchResult.OK)
	assert.Equal(t, workspaceID, s.activeWorkspaceID)

	legacyCreate := s.HandleWorkspaceMessage(core.New(), map[string]any{
		"action":     WorkspaceCreateAction,
		"identifier": "legacy-user",
		"password":   "pass123",
	})
	assert.True(t, legacyCreate.OK)

	legacyWorkspaceID, ok := legacyCreate.Value.(string)
	require.True(t, ok)
	require.NotEmpty(t, legacyWorkspaceID)

	legacySwitch := s.HandleWorkspaceMessage(core.New(), WorkspaceCommand{
		Action:      WorkspaceSwitchAction,
		WorkspaceID: legacyWorkspaceID,
	})
	assert.True(t, legacySwitch.OK)
	assert.Equal(t, legacyWorkspaceID, s.activeWorkspaceID)

	rejectedLegacySwitch := s.HandleWorkspaceMessage(core.New(), map[string]any{
		"action": WorkspaceSwitchAction,
		"name":   workspaceID,
	})
	assert.False(t, rejectedLegacySwitch.OK)
	assert.Equal(t, legacyWorkspaceID, s.activeWorkspaceID)

	failedSwitch := s.HandleWorkspaceMessage(core.New(), map[string]any{
		"action":      WorkspaceSwitchAction,
		"workspaceID": "missing",
	})
	assert.False(t, failedSwitch.OK)

	unknown := s.HandleWorkspaceMessage(core.New(), "noop")
	assert.True(t, unknown.OK)
}

func TestService_HandleIPCEvents_Compatibility_Good(t *testing.T) {
	s, _ := newTestService(t)

	result := s.HandleIPCEvents(core.New(), WorkspaceCommand{
		Action:     WorkspaceCreateAction,
		Identifier: "compat-user",
		Password:   "pass123",
	})

	assert.True(t, result.OK)
	workspaceID, ok := result.Value.(string)
	require.True(t, ok)
	require.NotEmpty(t, workspaceID)
}
