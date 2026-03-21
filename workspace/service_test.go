package workspace

import (
	"path/filepath"
	"testing"

	core "dappco.re/go/core"
	"forge.lthn.ai/core/go-crypt/crypt/openpgp"
	"github.com/stretchr/testify/assert"
)

func TestWorkspace(t *testing.T) {
	c := core.New()
	pgpSvc, err := openpgp.New(nil)
	assert.NoError(t, err)

	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	svc, err := New(c, pgpSvc.(cryptProvider))
	assert.NoError(t, err)
	s := svc.(*Service)

	// Test CreateWorkspace
	id, err := s.CreateWorkspace("test-user", "pass123")
	assert.NoError(t, err)
	assert.NotEmpty(t, id)

	wsPath := filepath.Join(tempHome, ".core", "workspaces", id)
	assert.DirExists(t, wsPath)
	assert.DirExists(t, filepath.Join(wsPath, "keys"))
	assert.FileExists(t, filepath.Join(wsPath, "keys", "private.key"))

	// Test SwitchWorkspace
	err = s.SwitchWorkspace(id)
	assert.NoError(t, err)
	assert.Equal(t, id, s.activeWorkspace)

	// Test File operations
	filename := "secret.txt"
	content := "top secret info"
	err = s.WorkspaceFileSet(filename, content)
	assert.NoError(t, err)

	got, err := s.WorkspaceFileGet(filename)
	assert.NoError(t, err)
	assert.Equal(t, content, got)
}
