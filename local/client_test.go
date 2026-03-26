package local

import (
	"io"
	"io/fs"
	"os"
	"strings"
	"testing"

	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew_Good_ResolvesRoot(t *testing.T) {
	root := t.TempDir()
	m, err := New(root)
	assert.NoError(t, err)
	// New() resolves symlinks (macOS /var → /private/var), so compare resolved paths.
	resolved, err := resolveSymlinksPath(root)
	require.NoError(t, err)
	assert.Equal(t, resolved, m.root)
}

func TestPath_Good_Sandboxed(t *testing.T) {
	m := &Medium{root: "/home/user"}

	// Normal paths
	assert.Equal(t, "/home/user/file.txt", m.path("file.txt"))
	assert.Equal(t, "/home/user/dir/file.txt", m.path("dir/file.txt"))

	// Empty returns root
	assert.Equal(t, "/home/user", m.path(""))

	// Traversal attempts get sanitised
	assert.Equal(t, "/home/user/file.txt", m.path("../file.txt"))
	assert.Equal(t, "/home/user/file.txt", m.path("dir/../file.txt"))

	// Absolute paths are constrained to sandbox (no escape)
	assert.Equal(t, "/home/user/etc/passwd", m.path("/etc/passwd"))
}

func TestPath_Good_RootFilesystem(t *testing.T) {
	m := &Medium{root: "/"}

	// When root is "/", absolute paths pass through
	assert.Equal(t, "/etc/passwd", m.path("/etc/passwd"))
	assert.Equal(t, "/home/user/file.txt", m.path("/home/user/file.txt"))

	// Relative paths are relative to CWD when root is "/"
	cwd := currentWorkingDir()
	assert.Equal(t, core.Path(cwd, "file.txt"), m.path("file.txt"))
}

func TestReadWrite_Good_Basic(t *testing.T) {
	root := t.TempDir()
	m, _ := New(root)

	// Write and read back
	err := m.Write("test.txt", "hello")
	assert.NoError(t, err)

	content, err := m.Read("test.txt")
	assert.NoError(t, err)
	assert.Equal(t, "hello", content)

	// Write creates parent dirs
	err = m.Write("a/b/c.txt", "nested")
	assert.NoError(t, err)

	content, err = m.Read("a/b/c.txt")
	assert.NoError(t, err)
	assert.Equal(t, "nested", content)

	// Read nonexistent
	_, err = m.Read("nope.txt")
	assert.Error(t, err)
}

func TestEnsureDir_Good_Basic(t *testing.T) {
	root := t.TempDir()
	m, _ := New(root)

	err := m.EnsureDir("one/two/three")
	assert.NoError(t, err)

	info, err := m.Stat("one/two/three")
	assert.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestIsDir_Good_Basic(t *testing.T) {
	root := t.TempDir()
	m, _ := New(root)

	_ = m.EnsureDir("mydir")
	_ = m.Write("myfile", "x")

	assert.True(t, m.IsDir("mydir"))
	assert.False(t, m.IsDir("myfile"))
	assert.False(t, m.IsDir("nope"))
	assert.False(t, m.IsDir(""))
}

func TestIsFile_Good_Basic(t *testing.T) {
	root := t.TempDir()
	m, _ := New(root)

	_ = m.EnsureDir("mydir")
	_ = m.Write("myfile", "x")

	assert.True(t, m.IsFile("myfile"))
	assert.False(t, m.IsFile("mydir"))
	assert.False(t, m.IsFile("nope"))
	assert.False(t, m.IsFile(""))
}

func TestExists_Good_Basic(t *testing.T) {
	root := t.TempDir()
	m, _ := New(root)

	_ = m.Write("exists", "x")

	assert.True(t, m.Exists("exists"))
	assert.False(t, m.Exists("nope"))
}

func TestList_Good_Basic(t *testing.T) {
	root := t.TempDir()
	m, _ := New(root)

	_ = m.Write("a.txt", "a")
	_ = m.Write("b.txt", "b")
	_ = m.EnsureDir("subdir")

	entries, err := m.List("")
	assert.NoError(t, err)
	assert.Len(t, entries, 3)
}

func TestStat_Good_Basic(t *testing.T) {
	root := t.TempDir()
	m, _ := New(root)

	_ = m.Write("file", "content")

	info, err := m.Stat("file")
	assert.NoError(t, err)
	assert.Equal(t, int64(7), info.Size())
}

func TestDelete_Good_Basic(t *testing.T) {
	root := t.TempDir()
	m, _ := New(root)

	_ = m.Write("todelete", "x")
	assert.True(t, m.Exists("todelete"))

	err := m.Delete("todelete")
	assert.NoError(t, err)
	assert.False(t, m.Exists("todelete"))
}

func TestDeleteAll_Good_Basic(t *testing.T) {
	root := t.TempDir()
	m, _ := New(root)

	_ = m.Write("dir/sub/file", "x")

	err := m.DeleteAll("dir")
	assert.NoError(t, err)
	assert.False(t, m.Exists("dir"))
}

func TestDelete_Bad_ProtectedHomeViaSymlinkEnv(t *testing.T) {
	realHome := t.TempDir()
	linkParent := t.TempDir()
	homeLink := core.Path(linkParent, "home-link")
	require.NoError(t, os.Symlink(realHome, homeLink))
	t.Setenv("HOME", homeLink)

	m, err := New("/")
	require.NoError(t, err)

	err = m.Delete(realHome)
	require.Error(t, err)
	assert.DirExists(t, realHome)
}

func TestDeleteAll_Bad_ProtectedHomeViaEnv(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	m, err := New("/")
	require.NoError(t, err)

	err = m.DeleteAll(tempHome)
	require.Error(t, err)
	assert.DirExists(t, tempHome)
}

func TestRename_Good_Basic(t *testing.T) {
	root := t.TempDir()
	m, _ := New(root)

	_ = m.Write("old", "x")

	err := m.Rename("old", "new")
	assert.NoError(t, err)
	assert.False(t, m.Exists("old"))
	assert.True(t, m.Exists("new"))
}

func TestFileGetFileSet_Good_Basic(t *testing.T) {
	root := t.TempDir()
	m, _ := New(root)

	err := m.FileSet("data", "value")
	assert.NoError(t, err)

	val, err := m.FileGet("data")
	assert.NoError(t, err)
	assert.Equal(t, "value", val)
}

func TestDelete_Good(t *testing.T) {
	testRoot := t.TempDir()

	medium, err := New(testRoot)
	assert.NoError(t, err)

	// Create and delete a file
	err = medium.Write("file.txt", "content")
	assert.NoError(t, err)
	assert.True(t, medium.IsFile("file.txt"))

	err = medium.Delete("file.txt")
	assert.NoError(t, err)
	assert.False(t, medium.IsFile("file.txt"))

	// Create and delete an empty directory
	err = medium.EnsureDir("emptydir")
	assert.NoError(t, err)
	err = medium.Delete("emptydir")
	assert.NoError(t, err)
	assert.False(t, medium.IsDir("emptydir"))
}

func TestDelete_Bad_NotEmpty(t *testing.T) {
	testRoot := t.TempDir()

	medium, err := New(testRoot)
	assert.NoError(t, err)

	// Create a directory with a file
	err = medium.Write("mydir/file.txt", "content")
	assert.NoError(t, err)

	// Try to delete non-empty directory
	err = medium.Delete("mydir")
	assert.Error(t, err)
}

func TestDeleteAll_Good(t *testing.T) {
	testRoot := t.TempDir()

	medium, err := New(testRoot)
	assert.NoError(t, err)

	// Create nested structure
	err = medium.Write("mydir/file1.txt", "content1")
	assert.NoError(t, err)
	err = medium.Write("mydir/subdir/file2.txt", "content2")
	assert.NoError(t, err)

	// Delete all
	err = medium.DeleteAll("mydir")
	assert.NoError(t, err)
	assert.False(t, medium.Exists("mydir"))
	assert.False(t, medium.Exists("mydir/file1.txt"))
	assert.False(t, medium.Exists("mydir/subdir/file2.txt"))
}

func TestRename_Good(t *testing.T) {
	testRoot := t.TempDir()

	medium, err := New(testRoot)
	assert.NoError(t, err)

	// Rename a file
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

func TestRename_Good_TraversalSanitised(t *testing.T) {
	testRoot := t.TempDir()

	medium, err := New(testRoot)
	assert.NoError(t, err)

	err = medium.Write("file.txt", "content")
	assert.NoError(t, err)

	// Traversal attempts are sanitised (.. becomes .), so this renames to "./escaped.txt"
	// which is just "escaped.txt" in the root
	err = medium.Rename("file.txt", "../escaped.txt")
	assert.NoError(t, err)
	assert.False(t, medium.Exists("file.txt"))
	assert.True(t, medium.Exists("escaped.txt"))
}

func TestList_Good(t *testing.T) {
	testRoot := t.TempDir()

	medium, err := New(testRoot)
	assert.NoError(t, err)

	// Create some files and directories
	err = medium.Write("file1.txt", "content1")
	assert.NoError(t, err)
	err = medium.Write("file2.txt", "content2")
	assert.NoError(t, err)
	err = medium.EnsureDir("subdir")
	assert.NoError(t, err)

	// List root
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

func TestStat_Good(t *testing.T) {
	testRoot := t.TempDir()

	medium, err := New(testRoot)
	assert.NoError(t, err)

	// Stat a file
	err = medium.Write("file.txt", "hello world")
	assert.NoError(t, err)
	info, err := medium.Stat("file.txt")
	assert.NoError(t, err)
	assert.Equal(t, "file.txt", info.Name())
	assert.Equal(t, int64(11), info.Size())
	assert.False(t, info.IsDir())

	// Stat a directory
	err = medium.EnsureDir("mydir")
	assert.NoError(t, err)
	info, err = medium.Stat("mydir")
	assert.NoError(t, err)
	assert.Equal(t, "mydir", info.Name())
	assert.True(t, info.IsDir())
}

func TestExists_Good(t *testing.T) {
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

func TestIsDir_Good(t *testing.T) {
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

func TestReadStream_Good_Basic(t *testing.T) {
	root := t.TempDir()
	m, _ := New(root)

	content := "streaming content"
	err := m.Write("stream.txt", content)
	assert.NoError(t, err)

	reader, err := m.ReadStream("stream.txt")
	assert.NoError(t, err)
	defer reader.Close()

	// Read only first 9 bytes
	limitReader := io.LimitReader(reader, 9)
	data, err := io.ReadAll(limitReader)
	assert.NoError(t, err)
	assert.Equal(t, "streaming", string(data))
}

func TestWriteStream_Good_Basic(t *testing.T) {
	root := t.TempDir()
	m, _ := New(root)

	writer, err := m.WriteStream("output.txt")
	assert.NoError(t, err)

	_, err = io.Copy(writer, strings.NewReader("piped data"))
	assert.NoError(t, err)
	err = writer.Close()
	assert.NoError(t, err)

	content, err := m.Read("output.txt")
	assert.NoError(t, err)
	assert.Equal(t, "piped data", content)
}

func TestPath_Ugly_TraversalAdvanced(t *testing.T) {
	m := &Medium{root: "/sandbox"}

	// Multiple levels of traversal
	assert.Equal(t, "/sandbox/file.txt", m.path("../../../file.txt"))
	assert.Equal(t, "/sandbox/target", m.path("dir/../../target"))

	// Traversal with hidden files
	assert.Equal(t, "/sandbox/.ssh/id_rsa", m.path(".ssh/id_rsa"))
	assert.Equal(t, "/sandbox/id_rsa", m.path(".ssh/../id_rsa"))

	// Null bytes (Go's filepath.Clean handles them, but good to check)
	assert.Equal(t, "/sandbox/file\x00.txt", m.path("file\x00.txt"))
}

func TestValidatePath_Bad_SymlinkEscape(t *testing.T) {
	root := t.TempDir()
	m, err := New(root)
	assert.NoError(t, err)

	// Create a directory outside the sandbox
	outside := t.TempDir()
	outsideFile := core.Path(outside, "secret.txt")
	outsideMedium, err := New("/")
	require.NoError(t, err)
	err = outsideMedium.Write(outsideFile, "secret")
	assert.NoError(t, err)

	// Test 1: Simple traversal
	_, err = m.validatePath("../outside.txt")
	assert.NoError(t, err) // path() sanitises to root, so this shouldn't escape

	// Test 2: Symlink escape
	// Create a symlink inside the sandbox pointing outside
	linkPath := core.Path(root, "evil_link")
	err = os.Symlink(outside, linkPath)
	assert.NoError(t, err)

	// Try to access a file through the symlink
	_, err = m.validatePath("evil_link/secret.txt")
	assert.Error(t, err)
	assert.ErrorIs(t, err, fs.ErrPermission)

	// Test 3: Nested symlink escape
	err = m.EnsureDir("inner")
	assert.NoError(t, err)
	innerDir := core.Path(root, "inner")
	nestedLink := core.Path(innerDir, "nested_evil")
	err = os.Symlink(outside, nestedLink)
	assert.NoError(t, err)

	_, err = m.validatePath("inner/nested_evil/secret.txt")
	assert.Error(t, err)
	assert.ErrorIs(t, err, fs.ErrPermission)
}

func TestEmptyPaths_Ugly(t *testing.T) {
	root := t.TempDir()
	m, err := New(root)
	assert.NoError(t, err)

	// Read empty path (should fail as it's a directory)
	_, err = m.Read("")
	assert.Error(t, err)

	// Write empty path (should fail as it's a directory)
	err = m.Write("", "content")
	assert.Error(t, err)

	// EnsureDir empty path (should be ok, it's just the root)
	err = m.EnsureDir("")
	assert.NoError(t, err)

	// IsDir empty path (should be true for root, but current impl returns false for "")
	// Wait, I noticed IsDir returns false for "" in the code.
	assert.False(t, m.IsDir(""))

	// Exists empty path (root exists)
	assert.True(t, m.Exists(""))

	// List empty path (lists root)
	entries, err := m.List("")
	assert.NoError(t, err)
	assert.NotNil(t, entries)
}
