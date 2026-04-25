package workspace

import (
	"io/fs"
	"testing"

	coreio "dappco.re/go/io"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestWorkspace(t *testing.T) (*Workspace, *coreio.MemoryMedium) {
	t.Helper()

	medium := coreio.NewMemoryMedium()
	workspaceService, err := NewWorkspace(medium, "workspaces")
	require.NoError(t, err)
	return workspaceService, medium
}

func TestWorkspace_NewWorkspace_Good(t *testing.T) {
	medium := coreio.NewMemoryMedium()

	workspaceService, err := NewWorkspace(medium, "root")
	require.NoError(t, err)

	assert.NotNil(t, workspaceService)
	assert.True(t, medium.IsDir("root"))
}

func TestWorkspace_NewWorkspace_Bad_InvalidBase(t *testing.T) {
	workspaceService, err := NewWorkspace(coreio.NewMemoryMedium(), "../escape")

	require.Error(t, err)
	assert.Nil(t, workspaceService)
}

func TestWorkspace_NewWorkspace_Ugly_NilMediumDefaults(t *testing.T) {
	workspaceService, err := NewWorkspace(nil, "")

	require.NoError(t, err)
	assert.NotNil(t, workspaceService)
}

func TestWorkspace_CreateWorkspace_Good(t *testing.T) {
	workspaceService, medium := newTestWorkspace(t)

	scoped, err := workspaceService.CreateWorkspace("alpha")
	require.NoError(t, err)
	require.NotNil(t, scoped)

	assert.True(t, medium.IsDir("workspaces/alpha"))
	require.NoError(t, scoped.Write("note.txt", "hello"))
	content, err := medium.Read("workspaces/alpha/note.txt")
	require.NoError(t, err)
	assert.Equal(t, "hello", content)
}

func TestWorkspace_CreateWorkspace_Bad_Duplicate(t *testing.T) {
	workspaceService, _ := newTestWorkspace(t)
	_, err := workspaceService.CreateWorkspace("alpha")
	require.NoError(t, err)

	_, err = workspaceService.CreateWorkspace("alpha")

	require.Error(t, err)
}

func TestWorkspace_CreateWorkspace_Ugly_TraversalName(t *testing.T) {
	workspaceService, medium := newTestWorkspace(t)

	_, err := workspaceService.CreateWorkspace("../escape")

	require.Error(t, err)
	assert.False(t, medium.IsDir("escape"))
}

func TestWorkspace_SwitchWorkspace_Good(t *testing.T) {
	workspaceService, _ := newTestWorkspace(t)
	_, err := workspaceService.CreateWorkspace("alpha")
	require.NoError(t, err)

	err = workspaceService.SwitchWorkspace("alpha")

	require.NoError(t, err)
	assert.Equal(t, "alpha", workspaceService.CurrentWorkspace())
}

func TestWorkspace_SwitchWorkspace_Bad_Missing(t *testing.T) {
	workspaceService, _ := newTestWorkspace(t)

	err := workspaceService.SwitchWorkspace("missing")

	require.Error(t, err)
	assert.Empty(t, workspaceService.CurrentWorkspace())
}

func TestWorkspace_SwitchWorkspace_Ugly_TraversalName(t *testing.T) {
	workspaceService, _ := newTestWorkspace(t)
	_, err := workspaceService.CreateWorkspace("alpha")
	require.NoError(t, err)
	require.NoError(t, workspaceService.SwitchWorkspace("alpha"))

	err = workspaceService.SwitchWorkspace("alpha/../beta")

	require.Error(t, err)
	assert.Equal(t, "alpha", workspaceService.CurrentWorkspace())
}

func TestWorkspace_ReadWorkspaceFile_Good(t *testing.T) {
	workspaceService, _ := newTestWorkspace(t)
	_, err := workspaceService.CreateWorkspace("alpha")
	require.NoError(t, err)
	require.NoError(t, workspaceService.WriteWorkspaceFile("alpha", "notes/todo.txt", "hello"))

	content, err := workspaceService.ReadWorkspaceFile("alpha", "notes/todo.txt")

	require.NoError(t, err)
	assert.Equal(t, "hello", content)
}

func TestWorkspace_ReadWorkspaceFile_Bad_MissingFile(t *testing.T) {
	workspaceService, _ := newTestWorkspace(t)
	_, err := workspaceService.CreateWorkspace("alpha")
	require.NoError(t, err)

	_, err = workspaceService.ReadWorkspaceFile("alpha", "missing.txt")

	require.Error(t, err)
}

func TestWorkspace_ReadWorkspaceFile_Ugly_TraversalPath(t *testing.T) {
	workspaceService, _ := newTestWorkspace(t)
	_, err := workspaceService.CreateWorkspace("alpha")
	require.NoError(t, err)

	_, err = workspaceService.ReadWorkspaceFile("alpha", "../alpha/secret.txt")

	require.Error(t, err)
}

func TestWorkspace_WriteWorkspaceFile_Good(t *testing.T) {
	workspaceService, medium := newTestWorkspace(t)
	_, err := workspaceService.CreateWorkspace("alpha")
	require.NoError(t, err)

	err = workspaceService.WriteWorkspaceFile("alpha", "notes/todo.txt", "hello")

	require.NoError(t, err)
	content, err := medium.Read("workspaces/alpha/notes/todo.txt")
	require.NoError(t, err)
	assert.Equal(t, "hello", content)
}

func TestWorkspace_WriteWorkspaceFile_Bad_MissingWorkspace(t *testing.T) {
	workspaceService, _ := newTestWorkspace(t)

	err := workspaceService.WriteWorkspaceFile("missing", "notes/todo.txt", "hello")

	require.Error(t, err)
}

func TestWorkspace_WriteWorkspaceFile_Ugly_TraversalPath(t *testing.T) {
	workspaceService, medium := newTestWorkspace(t)
	_, err := workspaceService.CreateWorkspace("alpha")
	require.NoError(t, err)

	err = workspaceService.WriteWorkspaceFile("alpha", "../outside.txt", "secret")

	require.Error(t, err)
	assert.False(t, medium.IsFile("workspaces/outside.txt"))
}

func TestWorkspace_HandleWorkspaceCommand_Good(t *testing.T) {
	workspaceService, _ := newTestWorkspace(t)

	create := workspaceService.HandleWorkspaceCommand(WorkspaceCommand{
		Action:    WorkspaceCreateAction,
		Workspace: "alpha",
	})
	require.True(t, create.OK)
	_, ok := create.Value.(coreio.Medium)
	require.True(t, ok)

	write := workspaceService.HandleWorkspaceCommand(WorkspaceCommand{
		Action:    WorkspaceWriteAction,
		Workspace: "alpha",
		Path:      "notes/todo.txt",
		Content:   "hello",
	})
	require.True(t, write.OK)

	read := workspaceService.HandleWorkspaceCommand(WorkspaceCommand{
		Action:    WorkspaceReadAction,
		Workspace: "alpha",
		Path:      "notes/todo.txt",
	})
	require.True(t, read.OK)
	assert.Equal(t, "hello", read.Value)

	list := workspaceService.HandleWorkspaceCommand(WorkspaceCommand{
		Action:    WorkspaceListAction,
		Workspace: "alpha",
		Path:      "notes",
	})
	require.True(t, list.OK)
	entries, ok := list.Value.([]fs.DirEntry)
	require.True(t, ok)
	require.Len(t, entries, 1)
	assert.Equal(t, "todo.txt", entries[0].Name())

	switchResult := workspaceService.HandleWorkspaceCommand(WorkspaceCommand{
		Action:    WorkspaceSwitchAction,
		Workspace: "alpha",
	})
	require.True(t, switchResult.OK)
	assert.Equal(t, "alpha", workspaceService.CurrentWorkspace())
}

func TestWorkspace_HandleWorkspaceCommand_Bad_UnknownAction(t *testing.T) {
	workspaceService, _ := newTestWorkspace(t)

	result := workspaceService.HandleWorkspaceCommand(WorkspaceCommand{Action: "noop"})

	assert.False(t, result.OK)
}

func TestWorkspace_HandleWorkspaceCommand_Ugly_TraversalPath(t *testing.T) {
	workspaceService, medium := newTestWorkspace(t)
	require.True(t, workspaceService.HandleWorkspaceCommand(WorkspaceCommand{
		Action:    WorkspaceCreateAction,
		Workspace: "alpha",
	}).OK)

	result := workspaceService.HandleWorkspaceCommand(WorkspaceCommand{
		Action:    WorkspaceWriteAction,
		Workspace: "alpha",
		Path:      "../outside.txt",
		Content:   "secret",
	})

	assert.False(t, result.OK)
	assert.False(t, medium.IsFile("workspaces/outside.txt"))
}
