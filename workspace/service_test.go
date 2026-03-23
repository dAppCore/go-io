package workspace

import (
	"os"
	"testing"

	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubCrypt struct {
	key string
	err error
}

func (s stubCrypt) CreateKeyPair(_, _ string) (string, error) {
	if s.err != nil {
		return "", s.err
	}
	return s.key, nil
}

func newTestService(t *testing.T) (*Service, string) {
	t.Helper()

	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	svc, err := New(core.New(), stubCrypt{key: "private-key"})
	require.NoError(t, err)
	return svc.(*Service), tempHome
}

func TestWorkspace(t *testing.T) {
	s, tempHome := newTestService(t)

	id, err := s.CreateWorkspace("test-user", "pass123")
	require.NoError(t, err)
	assert.NotEmpty(t, id)

	wsPath := core.Path(tempHome, ".core", "workspaces", id)
	assert.DirExists(t, wsPath)
	assert.DirExists(t, core.Path(wsPath, "keys"))
	assert.FileExists(t, core.Path(wsPath, "keys", "private.key"))

	err = s.SwitchWorkspace(id)
	require.NoError(t, err)
	assert.Equal(t, id, s.activeWorkspace)

	err = s.WorkspaceFileSet("secret.txt", "top secret info")
	require.NoError(t, err)

	got, err := s.WorkspaceFileGet("secret.txt")
	require.NoError(t, err)
	assert.Equal(t, "top secret info", got)
}

func TestSwitchWorkspace_TraversalBlocked(t *testing.T) {
	s, tempHome := newTestService(t)

	outside := core.Path(tempHome, ".core", "escaped")
	require.NoError(t, os.MkdirAll(outside, 0755))

	err := s.SwitchWorkspace("../escaped")
	require.Error(t, err)
	assert.Empty(t, s.activeWorkspace)
}

func TestWorkspaceFileSet_TraversalBlocked(t *testing.T) {
	s, tempHome := newTestService(t)

	id, err := s.CreateWorkspace("test-user", "pass123")
	require.NoError(t, err)
	require.NoError(t, s.SwitchWorkspace(id))

	keyPath := core.Path(tempHome, ".core", "workspaces", id, "keys", "private.key")
	before, err := os.ReadFile(keyPath)
	require.NoError(t, err)

	err = s.WorkspaceFileSet("../keys/private.key", "hijack")
	require.Error(t, err)

	after, err := os.ReadFile(keyPath)
	require.NoError(t, err)
	assert.Equal(t, string(before), string(after))

	_, err = s.WorkspaceFileGet("../keys/private.key")
	require.Error(t, err)
}
