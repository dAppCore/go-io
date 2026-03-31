package local

import (
	"io"
	"io/fs"
	"syscall"
	"testing"

	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_New_ResolvesRoot_Good(t *testing.T) {
	root := t.TempDir()
	m, err := New(root)
	assert.NoError(t, err)
	resolved, err := resolveSymlinksPath(root)
	require.NoError(t, err)
	assert.Equal(t, resolved, m.filesystemRoot)
}

func TestClient_Path_Sandboxed_Good(t *testing.T) {
	m := &Medium{filesystemRoot: "/home/user"}

	assert.Equal(t, "/home/user/file.txt", m.sandboxedPath("file.txt"))
	assert.Equal(t, "/home/user/dir/file.txt", m.sandboxedPath("dir/file.txt"))

	assert.Equal(t, "/home/user", m.sandboxedPath(""))

	assert.Equal(t, "/home/user/file.txt", m.sandboxedPath("../file.txt"))
	assert.Equal(t, "/home/user/file.txt", m.sandboxedPath("dir/../file.txt"))

	assert.Equal(t, "/home/user/etc/passwd", m.sandboxedPath("/etc/passwd"))
}

func TestClient_Path_RootFilesystem_Good(t *testing.T) {
	m := &Medium{filesystemRoot: "/"}

	assert.Equal(t, "/etc/passwd", m.sandboxedPath("/etc/passwd"))
	assert.Equal(t, "/home/user/file.txt", m.sandboxedPath("/home/user/file.txt"))

	cwd := currentWorkingDir()
	assert.Equal(t, core.Path(cwd, "file.txt"), m.sandboxedPath("file.txt"))
}

func TestClient_ReadWrite_Basic_Good(t *testing.T) {
	root := t.TempDir()
	m, _ := New(root)

	err := m.Write("test.txt", "hello")
	assert.NoError(t, err)

	content, err := m.Read("test.txt")
	assert.NoError(t, err)
	assert.Equal(t, "hello", content)

	err = m.Write("a/b/c.txt", "nested")
	assert.NoError(t, err)

	content, err = m.Read("a/b/c.txt")
	assert.NoError(t, err)
	assert.Equal(t, "nested", content)

	_, err = m.Read("nope.txt")
	assert.Error(t, err)
}

func TestClient_EnsureDir_Basic_Good(t *testing.T) {
	root := t.TempDir()
	m, _ := New(root)

	err := m.EnsureDir("one/two/three")
	assert.NoError(t, err)

	info, err := m.Stat("one/two/three")
	assert.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestClient_IsDir_Basic_Good(t *testing.T) {
	root := t.TempDir()
	m, _ := New(root)

	_ = m.EnsureDir("mydir")
	_ = m.Write("myfile", "x")

	assert.True(t, m.IsDir("mydir"))
	assert.False(t, m.IsDir("myfile"))
	assert.False(t, m.IsDir("nope"))
	assert.False(t, m.IsDir(""))
}

func TestClient_IsFile_Basic_Good(t *testing.T) {
	root := t.TempDir()
	m, _ := New(root)

	_ = m.EnsureDir("mydir")
	_ = m.Write("myfile", "x")

	assert.True(t, m.IsFile("myfile"))
	assert.False(t, m.IsFile("mydir"))
	assert.False(t, m.IsFile("nope"))
	assert.False(t, m.IsFile(""))
}

func TestClient_Exists_Basic_Good(t *testing.T) {
	root := t.TempDir()
	m, _ := New(root)

	_ = m.Write("exists", "x")

	assert.True(t, m.Exists("exists"))
	assert.False(t, m.Exists("nope"))
}

func TestClient_List_Basic_Good(t *testing.T) {
	root := t.TempDir()
	m, _ := New(root)

	_ = m.Write("a.txt", "a")
	_ = m.Write("b.txt", "b")
	_ = m.EnsureDir("subdir")

	entries, err := m.List("")
	assert.NoError(t, err)
	assert.Len(t, entries, 3)
}

func TestClient_Stat_Basic_Good(t *testing.T) {
	root := t.TempDir()
	m, _ := New(root)

	_ = m.Write("file", "content")

	info, err := m.Stat("file")
	assert.NoError(t, err)
	assert.Equal(t, int64(7), info.Size())
}

func TestClient_Delete_Basic_Good(t *testing.T) {
	root := t.TempDir()
	m, _ := New(root)

	_ = m.Write("todelete", "x")
	assert.True(t, m.Exists("todelete"))

	err := m.Delete("todelete")
	assert.NoError(t, err)
	assert.False(t, m.Exists("todelete"))
}

func TestClient_DeleteAll_Basic_Good(t *testing.T) {
	root := t.TempDir()
	m, _ := New(root)

	_ = m.Write("dir/sub/file", "x")

	err := m.DeleteAll("dir")
	assert.NoError(t, err)
	assert.False(t, m.Exists("dir"))
}

func TestClient_Delete_ProtectedHomeViaSymlinkEnv_Bad(t *testing.T) {
	realHome := t.TempDir()
	linkParent := t.TempDir()
	homeLink := core.Path(linkParent, "home-link")
	require.NoError(t, syscall.Symlink(realHome, homeLink))
	t.Setenv("HOME", homeLink)

	m, err := New("/")
	require.NoError(t, err)

	err = m.Delete(realHome)
	require.Error(t, err)
	assert.DirExists(t, realHome)
}

func TestClient_DeleteAll_ProtectedHomeViaEnv_Bad(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	m, err := New("/")
	require.NoError(t, err)

	err = m.DeleteAll(tempHome)
	require.Error(t, err)
	assert.DirExists(t, tempHome)
}

func TestClient_Rename_Basic_Good(t *testing.T) {
	root := t.TempDir()
	m, _ := New(root)

	_ = m.Write("old", "x")

	err := m.Rename("old", "new")
	assert.NoError(t, err)
	assert.False(t, m.Exists("old"))
	assert.True(t, m.Exists("new"))
}

func TestClient_FileGetFileSet_Basic_Good(t *testing.T) {
	root := t.TempDir()
	m, _ := New(root)

	err := m.FileSet("data", "value")
	assert.NoError(t, err)

	val, err := m.FileGet("data")
	assert.NoError(t, err)
	assert.Equal(t, "value", val)
}

func TestClient_Delete_Good(t *testing.T) {
	testRoot := t.TempDir()

	medium, err := New(testRoot)
	assert.NoError(t, err)

	err = medium.Write("file.txt", "content")
	assert.NoError(t, err)
	assert.True(t, medium.IsFile("file.txt"))

	err = medium.Delete("file.txt")
	assert.NoError(t, err)
	assert.False(t, medium.IsFile("file.txt"))

	err = medium.EnsureDir("emptydir")
	assert.NoError(t, err)
	err = medium.Delete("emptydir")
	assert.NoError(t, err)
	assert.False(t, medium.IsDir("emptydir"))
}

func TestClient_Delete_NotEmpty_Bad(t *testing.T) {
	testRoot := t.TempDir()

	medium, err := New(testRoot)
	assert.NoError(t, err)

	err = medium.Write("mydir/file.txt", "content")
	assert.NoError(t, err)

	err = medium.Delete("mydir")
	assert.Error(t, err)
}

func TestClient_DeleteAll_Good(t *testing.T) {
	testRoot := t.TempDir()

	medium, err := New(testRoot)
	assert.NoError(t, err)

	err = medium.Write("mydir/file1.txt", "content1")
	assert.NoError(t, err)
	err = medium.Write("mydir/subdir/file2.txt", "content2")
	assert.NoError(t, err)

	err = medium.DeleteAll("mydir")
	assert.NoError(t, err)
	assert.False(t, medium.Exists("mydir"))
	assert.False(t, medium.Exists("mydir/file1.txt"))
	assert.False(t, medium.Exists("mydir/subdir/file2.txt"))
}

func TestClient_Rename_Good(t *testing.T) {
	testRoot := t.TempDir()

	medium, err := New(testRoot)
	assert.NoError(t, err)

	err = medium.Write("old.txt", "content")
	assert.NoError(t, err)
	err = medium.Rename("old.txt", "new.txt")
	assert.NoError(t, err)
	assert.False(t, medium.IsFile("old.txt"))
	assert.True(t, medium.IsFile("new.txt"))

	content, err := medium.Read("new.txt")
	assert.NoError(t, err)
	assert.Equal(t, "content", content)
}

func TestClient_Rename_TraversalSanitised_Good(t *testing.T) {
	testRoot := t.TempDir()

	medium, err := New(testRoot)
	assert.NoError(t, err)

	err = medium.Write("file.txt", "content")
	assert.NoError(t, err)

	err = medium.Rename("file.txt", "../escaped.txt")
	assert.NoError(t, err)
	assert.False(t, medium.Exists("file.txt"))
	assert.True(t, medium.Exists("escaped.txt"))
}

func TestClient_List_Good(t *testing.T) {
	testRoot := t.TempDir()

	medium, err := New(testRoot)
	assert.NoError(t, err)

	err = medium.Write("file1.txt", "content1")
	assert.NoError(t, err)
	err = medium.Write("file2.txt", "content2")
	assert.NoError(t, err)
	err = medium.EnsureDir("subdir")
	assert.NoError(t, err)

	entries, err := medium.List(".")
	assert.NoError(t, err)
	assert.Len(t, entries, 3)

	names := make(map[string]bool)
	for _, e := range entries {
		names[e.Name()] = true
	}
	assert.True(t, names["file1.txt"])
	assert.True(t, names["file2.txt"])
	assert.True(t, names["subdir"])
}

func TestClient_Stat_Good(t *testing.T) {
	testRoot := t.TempDir()

	medium, err := New(testRoot)
	assert.NoError(t, err)

	err = medium.Write("file.txt", "hello world")
	assert.NoError(t, err)
	info, err := medium.Stat("file.txt")
	assert.NoError(t, err)
	assert.Equal(t, "file.txt", info.Name())
	assert.Equal(t, int64(11), info.Size())
	assert.False(t, info.IsDir())

	err = medium.EnsureDir("mydir")
	assert.NoError(t, err)
	info, err = medium.Stat("mydir")
	assert.NoError(t, err)
	assert.Equal(t, "mydir", info.Name())
	assert.True(t, info.IsDir())
}

func TestClient_Exists_Good(t *testing.T) {
	testRoot := t.TempDir()

	medium, err := New(testRoot)
	assert.NoError(t, err)

	assert.False(t, medium.Exists("nonexistent"))

	err = medium.Write("file.txt", "content")
	assert.NoError(t, err)
	assert.True(t, medium.Exists("file.txt"))

	err = medium.EnsureDir("mydir")
	assert.NoError(t, err)
	assert.True(t, medium.Exists("mydir"))
}

func TestClient_IsDir_Good(t *testing.T) {
	testRoot := t.TempDir()

	medium, err := New(testRoot)
	assert.NoError(t, err)

	err = medium.Write("file.txt", "content")
	assert.NoError(t, err)
	assert.False(t, medium.IsDir("file.txt"))

	err = medium.EnsureDir("mydir")
	assert.NoError(t, err)
	assert.True(t, medium.IsDir("mydir"))

	assert.False(t, medium.IsDir("nonexistent"))
}

func TestClient_ReadStream_Basic_Good(t *testing.T) {
	root := t.TempDir()
	m, _ := New(root)

	content := "streaming content"
	err := m.Write("stream.txt", content)
	assert.NoError(t, err)

	reader, err := m.ReadStream("stream.txt")
	assert.NoError(t, err)
	defer reader.Close()

	limitReader := io.LimitReader(reader, 9)
	data, err := io.ReadAll(limitReader)
	assert.NoError(t, err)
	assert.Equal(t, "streaming", string(data))
}

func TestClient_WriteStream_Basic_Good(t *testing.T) {
	root := t.TempDir()
	m, _ := New(root)

	writer, err := m.WriteStream("output.txt")
	assert.NoError(t, err)

	_, err = io.Copy(writer, core.NewReader("piped data"))
	assert.NoError(t, err)
	err = writer.Close()
	assert.NoError(t, err)

	content, err := m.Read("output.txt")
	assert.NoError(t, err)
	assert.Equal(t, "piped data", content)
}

func TestClient_Path_TraversalSandbox_Good(t *testing.T) {
	m := &Medium{filesystemRoot: "/sandbox"}

	assert.Equal(t, "/sandbox/file.txt", m.sandboxedPath("../../../file.txt"))
	assert.Equal(t, "/sandbox/target", m.sandboxedPath("dir/../../target"))

	assert.Equal(t, "/sandbox/.ssh/id_rsa", m.sandboxedPath(".ssh/id_rsa"))
	assert.Equal(t, "/sandbox/id_rsa", m.sandboxedPath(".ssh/../id_rsa"))

	assert.Equal(t, "/sandbox/file\x00.txt", m.sandboxedPath("file\x00.txt"))
}

func TestClient_ValidatePath_SymlinkEscape_Bad(t *testing.T) {
	root := t.TempDir()
	m, err := New(root)
	assert.NoError(t, err)

	outside := t.TempDir()
	outsideFile := core.Path(outside, "secret.txt")
	outsideMedium, err := New("/")
	require.NoError(t, err)
	err = outsideMedium.Write(outsideFile, "secret")
	assert.NoError(t, err)

	_, err = m.validatePath("../outside.txt")
	assert.NoError(t, err)

	linkPath := core.Path(root, "evil_link")
	err = syscall.Symlink(outside, linkPath)
	assert.NoError(t, err)

	_, err = m.validatePath("evil_link/secret.txt")
	assert.Error(t, err)
	assert.ErrorIs(t, err, fs.ErrPermission)

	err = m.EnsureDir("inner")
	assert.NoError(t, err)
	innerDir := core.Path(root, "inner")
	nestedLink := core.Path(innerDir, "nested_evil")
	err = syscall.Symlink(outside, nestedLink)
	assert.NoError(t, err)

	_, err = m.validatePath("inner/nested_evil/secret.txt")
	assert.Error(t, err)
	assert.ErrorIs(t, err, fs.ErrPermission)
}

func TestClient_EmptyPaths_Good(t *testing.T) {
	root := t.TempDir()
	m, err := New(root)
	assert.NoError(t, err)

	_, err = m.Read("")
	assert.Error(t, err)

	err = m.Write("", "content")
	assert.Error(t, err)

	err = m.EnsureDir("")
	assert.NoError(t, err)

	assert.False(t, m.IsDir(""))

	assert.True(t, m.Exists(""))

	entries, err := m.List("")
	assert.NoError(t, err)
	assert.NotNil(t, entries)
}
