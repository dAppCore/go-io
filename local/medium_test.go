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

func TestLocal_New_ResolvesRoot_Good(t *testing.T) {
	root := t.TempDir()
	localMedium, err := New(root)
	assert.NoError(t, err)
	resolved, err := resolveSymlinksPath(root)
	require.NoError(t, err)
	assert.Equal(t, resolved, localMedium.filesystemRoot)
}

func TestLocal_Path_Sandboxed_Good(t *testing.T) {
	localMedium := &Medium{filesystemRoot: "/home/user"}

	assert.Equal(t, "/home/user/file.txt", localMedium.sandboxedPath("file.txt"))
	assert.Equal(t, "/home/user/dir/file.txt", localMedium.sandboxedPath("dir/file.txt"))

	assert.Equal(t, "/home/user", localMedium.sandboxedPath(""))

	assert.Equal(t, "/home/user/file.txt", localMedium.sandboxedPath("../file.txt"))
	assert.Equal(t, "/home/user/file.txt", localMedium.sandboxedPath("dir/../file.txt"))

	assert.Equal(t, "/home/user/etc/passwd", localMedium.sandboxedPath("/etc/passwd"))
}

func TestLocal_Path_RootFilesystem_Good(t *testing.T) {
	localMedium := &Medium{filesystemRoot: "/"}

	assert.Equal(t, "/etc/passwd", localMedium.sandboxedPath("/etc/passwd"))
	assert.Equal(t, "/home/user/file.txt", localMedium.sandboxedPath("/home/user/file.txt"))

	workingDirectory := currentWorkingDir()
	assert.Equal(t, core.Path(workingDirectory, "file.txt"), localMedium.sandboxedPath("file.txt"))
}

func TestLocal_ReadWrite_Basic_Good(t *testing.T) {
	root := t.TempDir()
	localMedium, _ := New(root)

	err := localMedium.Write("test.txt", "hello")
	assert.NoError(t, err)

	content, err := localMedium.Read("test.txt")
	assert.NoError(t, err)
	assert.Equal(t, "hello", content)

	err = localMedium.Write("a/b/c.txt", "nested")
	assert.NoError(t, err)

	content, err = localMedium.Read("a/b/c.txt")
	assert.NoError(t, err)
	assert.Equal(t, "nested", content)

	_, err = localMedium.Read("nope.txt")
	assert.Error(t, err)
}

func TestLocal_EnsureDir_Basic_Good(t *testing.T) {
	root := t.TempDir()
	localMedium, _ := New(root)

	err := localMedium.EnsureDir("one/two/three")
	assert.NoError(t, err)

	info, err := localMedium.Stat("one/two/three")
	assert.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestLocal_IsDir_Basic_Good(t *testing.T) {
	root := t.TempDir()
	localMedium, _ := New(root)

	_ = localMedium.EnsureDir("mydir")
	_ = localMedium.Write("myfile", "x")

	assert.True(t, localMedium.IsDir("mydir"))
	assert.False(t, localMedium.IsDir("myfile"))
	assert.False(t, localMedium.IsDir("nope"))
	assert.False(t, localMedium.IsDir(""))
}

func TestLocal_IsFile_Basic_Good(t *testing.T) {
	root := t.TempDir()
	localMedium, _ := New(root)

	_ = localMedium.EnsureDir("mydir")
	_ = localMedium.Write("myfile", "x")

	assert.True(t, localMedium.IsFile("myfile"))
	assert.False(t, localMedium.IsFile("mydir"))
	assert.False(t, localMedium.IsFile("nope"))
	assert.False(t, localMedium.IsFile(""))
}

func TestLocal_Exists_Basic_Good(t *testing.T) {
	root := t.TempDir()
	localMedium, _ := New(root)

	_ = localMedium.Write("exists", "x")

	assert.True(t, localMedium.Exists("exists"))
	assert.False(t, localMedium.Exists("nope"))
}

func TestLocal_List_Basic_Good(t *testing.T) {
	root := t.TempDir()
	localMedium, _ := New(root)

	_ = localMedium.Write("a.txt", "a")
	_ = localMedium.Write("b.txt", "b")
	_ = localMedium.EnsureDir("subdir")

	entries, err := localMedium.List("")
	assert.NoError(t, err)
	assert.Len(t, entries, 3)
}

func TestLocal_Stat_Basic_Good(t *testing.T) {
	root := t.TempDir()
	localMedium, _ := New(root)

	_ = localMedium.Write("file", "content")

	info, err := localMedium.Stat("file")
	assert.NoError(t, err)
	assert.Equal(t, int64(7), info.Size())
}

func TestLocal_Delete_Basic_Good(t *testing.T) {
	root := t.TempDir()
	localMedium, _ := New(root)

	_ = localMedium.Write("todelete", "x")
	assert.True(t, localMedium.Exists("todelete"))

	err := localMedium.Delete("todelete")
	assert.NoError(t, err)
	assert.False(t, localMedium.Exists("todelete"))
}

func TestLocal_DeleteAll_Basic_Good(t *testing.T) {
	root := t.TempDir()
	localMedium, _ := New(root)

	_ = localMedium.Write("dir/sub/file", "x")

	err := localMedium.DeleteAll("dir")
	assert.NoError(t, err)
	assert.False(t, localMedium.Exists("dir"))
}

func TestLocal_Delete_ProtectedHomeViaSymlinkEnv_Bad(t *testing.T) {
	realHome := t.TempDir()
	linkParent := t.TempDir()
	homeLink := core.Path(linkParent, "home-link")
	require.NoError(t, syscall.Symlink(realHome, homeLink))
	t.Setenv("HOME", homeLink)

	localMedium, err := New("/")
	require.NoError(t, err)

	err = localMedium.Delete(realHome)
	require.Error(t, err)
	assert.DirExists(t, realHome)
}

func TestLocal_DeleteAll_ProtectedHomeViaEnv_Bad(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	localMedium, err := New("/")
	require.NoError(t, err)

	err = localMedium.DeleteAll(tempHome)
	require.Error(t, err)
	assert.DirExists(t, tempHome)
}

func TestLocal_Rename_Basic_Good(t *testing.T) {
	root := t.TempDir()
	localMedium, _ := New(root)

	_ = localMedium.Write("old", "x")

	err := localMedium.Rename("old", "new")
	assert.NoError(t, err)
	assert.False(t, localMedium.Exists("old"))
	assert.True(t, localMedium.Exists("new"))
}

func TestLocal_Delete_Good(t *testing.T) {
	testRoot := t.TempDir()

	localMedium, err := New(testRoot)
	assert.NoError(t, err)

	err = localMedium.Write("file.txt", "content")
	assert.NoError(t, err)
	assert.True(t, localMedium.IsFile("file.txt"))

	err = localMedium.Delete("file.txt")
	assert.NoError(t, err)
	assert.False(t, localMedium.IsFile("file.txt"))

	err = localMedium.EnsureDir("emptydir")
	assert.NoError(t, err)
	err = localMedium.Delete("emptydir")
	assert.NoError(t, err)
	assert.False(t, localMedium.IsDir("emptydir"))
}

func TestLocal_Delete_NotEmpty_Bad(t *testing.T) {
	testRoot := t.TempDir()

	localMedium, err := New(testRoot)
	assert.NoError(t, err)

	err = localMedium.Write("mydir/file.txt", "content")
	assert.NoError(t, err)

	err = localMedium.Delete("mydir")
	assert.Error(t, err)
}

func TestLocal_DeleteAll_Good(t *testing.T) {
	testRoot := t.TempDir()

	localMedium, err := New(testRoot)
	assert.NoError(t, err)

	err = localMedium.Write("mydir/file1.txt", "content1")
	assert.NoError(t, err)
	err = localMedium.Write("mydir/subdir/file2.txt", "content2")
	assert.NoError(t, err)

	err = localMedium.DeleteAll("mydir")
	assert.NoError(t, err)
	assert.False(t, localMedium.Exists("mydir"))
	assert.False(t, localMedium.Exists("mydir/file1.txt"))
	assert.False(t, localMedium.Exists("mydir/subdir/file2.txt"))
}

func TestLocal_Rename_Good(t *testing.T) {
	testRoot := t.TempDir()

	localMedium, err := New(testRoot)
	assert.NoError(t, err)

	err = localMedium.Write("old.txt", "content")
	assert.NoError(t, err)
	err = localMedium.Rename("old.txt", "new.txt")
	assert.NoError(t, err)
	assert.False(t, localMedium.IsFile("old.txt"))
	assert.True(t, localMedium.IsFile("new.txt"))

	content, err := localMedium.Read("new.txt")
	assert.NoError(t, err)
	assert.Equal(t, "content", content)
}

func TestLocal_Rename_TraversalSanitised_Good(t *testing.T) {
	testRoot := t.TempDir()

	localMedium, err := New(testRoot)
	assert.NoError(t, err)

	err = localMedium.Write("file.txt", "content")
	assert.NoError(t, err)

	err = localMedium.Rename("file.txt", "../escaped.txt")
	assert.NoError(t, err)
	assert.False(t, localMedium.Exists("file.txt"))
	assert.True(t, localMedium.Exists("escaped.txt"))
}

func TestLocal_List_Good(t *testing.T) {
	testRoot := t.TempDir()

	localMedium, err := New(testRoot)
	assert.NoError(t, err)

	err = localMedium.Write("file1.txt", "content1")
	assert.NoError(t, err)
	err = localMedium.Write("file2.txt", "content2")
	assert.NoError(t, err)
	err = localMedium.EnsureDir("subdir")
	assert.NoError(t, err)

	entries, err := localMedium.List(".")
	assert.NoError(t, err)
	assert.Len(t, entries, 3)

	names := make(map[string]bool)
	for _, entry := range entries {
		names[entry.Name()] = true
	}
	assert.True(t, names["file1.txt"])
	assert.True(t, names["file2.txt"])
	assert.True(t, names["subdir"])
}

func TestLocal_Stat_Good(t *testing.T) {
	testRoot := t.TempDir()

	localMedium, err := New(testRoot)
	assert.NoError(t, err)

	err = localMedium.Write("file.txt", "hello world")
	assert.NoError(t, err)
	info, err := localMedium.Stat("file.txt")
	assert.NoError(t, err)
	assert.Equal(t, "file.txt", info.Name())
	assert.Equal(t, int64(11), info.Size())
	assert.False(t, info.IsDir())

	err = localMedium.EnsureDir("mydir")
	assert.NoError(t, err)
	info, err = localMedium.Stat("mydir")
	assert.NoError(t, err)
	assert.Equal(t, "mydir", info.Name())
	assert.True(t, info.IsDir())
}

func TestLocal_Exists_Good(t *testing.T) {
	testRoot := t.TempDir()

	localMedium, err := New(testRoot)
	assert.NoError(t, err)

	assert.False(t, localMedium.Exists("nonexistent"))

	err = localMedium.Write("file.txt", "content")
	assert.NoError(t, err)
	assert.True(t, localMedium.Exists("file.txt"))

	err = localMedium.EnsureDir("mydir")
	assert.NoError(t, err)
	assert.True(t, localMedium.Exists("mydir"))
}

func TestLocal_IsDir_Good(t *testing.T) {
	testRoot := t.TempDir()

	localMedium, err := New(testRoot)
	assert.NoError(t, err)

	err = localMedium.Write("file.txt", "content")
	assert.NoError(t, err)
	assert.False(t, localMedium.IsDir("file.txt"))

	err = localMedium.EnsureDir("mydir")
	assert.NoError(t, err)
	assert.True(t, localMedium.IsDir("mydir"))

	assert.False(t, localMedium.IsDir("nonexistent"))
}

func TestLocal_ReadStream_Basic_Good(t *testing.T) {
	root := t.TempDir()
	localMedium, _ := New(root)

	content := "streaming content"
	err := localMedium.Write("stream.txt", content)
	assert.NoError(t, err)

	reader, err := localMedium.ReadStream("stream.txt")
	assert.NoError(t, err)
	defer reader.Close()

	limitReader := io.LimitReader(reader, 9)
	data, err := io.ReadAll(limitReader)
	assert.NoError(t, err)
	assert.Equal(t, "streaming", string(data))
}

func TestLocal_WriteStream_Basic_Good(t *testing.T) {
	root := t.TempDir()
	localMedium, _ := New(root)

	writer, err := localMedium.WriteStream("output.txt")
	assert.NoError(t, err)

	_, err = io.Copy(writer, core.NewReader("piped data"))
	assert.NoError(t, err)
	err = writer.Close()
	assert.NoError(t, err)

	content, err := localMedium.Read("output.txt")
	assert.NoError(t, err)
	assert.Equal(t, "piped data", content)
}

func TestLocal_Path_TraversalSandbox_Good(t *testing.T) {
	localMedium := &Medium{filesystemRoot: "/sandbox"}

	assert.Equal(t, "/sandbox/file.txt", localMedium.sandboxedPath("../../../file.txt"))
	assert.Equal(t, "/sandbox/target", localMedium.sandboxedPath("dir/../../target"))

	assert.Equal(t, "/sandbox/.ssh/id_rsa", localMedium.sandboxedPath(".ssh/id_rsa"))
	assert.Equal(t, "/sandbox/id_rsa", localMedium.sandboxedPath(".ssh/../id_rsa"))

	assert.Equal(t, "/sandbox/file\x00.txt", localMedium.sandboxedPath("file\x00.txt"))
}

func TestLocal_ValidatePath_SymlinkEscape_Bad(t *testing.T) {
	root := t.TempDir()
	localMedium, err := New(root)
	assert.NoError(t, err)

	outside := t.TempDir()
	outsideFile := core.Path(outside, "secret.txt")
	outsideMedium, err := New("/")
	require.NoError(t, err)
	err = outsideMedium.Write(outsideFile, "secret")
	assert.NoError(t, err)

	_, err = localMedium.validatePath("../outside.txt")
	assert.NoError(t, err)

	linkPath := core.Path(root, "evil_link")
	err = syscall.Symlink(outside, linkPath)
	assert.NoError(t, err)

	_, err = localMedium.validatePath("evil_link/secret.txt")
	assert.Error(t, err)
	assert.ErrorIs(t, err, fs.ErrPermission)

	err = localMedium.EnsureDir("inner")
	assert.NoError(t, err)
	innerDir := core.Path(root, "inner")
	nestedLink := core.Path(innerDir, "nested_evil")
	err = syscall.Symlink(outside, nestedLink)
	assert.NoError(t, err)

	_, err = localMedium.validatePath("inner/nested_evil/secret.txt")
	assert.Error(t, err)
	assert.ErrorIs(t, err, fs.ErrPermission)
}

func TestLocal_EmptyPaths_Good(t *testing.T) {
	root := t.TempDir()
	localMedium, err := New(root)
	assert.NoError(t, err)

	_, err = localMedium.Read("")
	assert.Error(t, err)

	err = localMedium.Write("", "content")
	assert.Error(t, err)

	err = localMedium.EnsureDir("")
	assert.NoError(t, err)

	assert.False(t, localMedium.IsDir(""))

	assert.True(t, localMedium.Exists(""))

	entries, err := localMedium.List("")
	assert.NoError(t, err)
	assert.NotNil(t, entries)
}
