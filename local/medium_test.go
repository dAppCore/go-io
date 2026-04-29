package local

import (
	core "dappco.re/go"
	goio "io"
	"io/fs"
	"syscall"
)

func TestLocal_New_ResolvesRoot_Good(t *core.T) {
	root := t.TempDir()
	localMedium, err := New(root)
	core.AssertNoError(t, err)
	resolved, err := resolveSymlinksPath(root)
	core.RequireNoError(t, err)
	core.AssertEqual(t, resolved, localMedium.filesystemRoot)
}

func TestLocal_Path_SandboxedGood(t *core.T) {
	localMedium := &Medium{filesystemRoot: "/home/user"}

	core.AssertEqual(t, "/home/user/file.txt", localMedium.sandboxedPath("file.txt"))
	core.AssertEqual(t, "/home/user/dir/file.txt", localMedium.sandboxedPath("dir/file.txt"))

	core.AssertEqual(t, "/home/user", localMedium.sandboxedPath(""))

	core.AssertEqual(t, "/home/user/file.txt", localMedium.sandboxedPath("../file.txt"))
	core.AssertEqual(t, "/home/user/file.txt", localMedium.sandboxedPath("dir/../file.txt"))

	core.AssertEqual(t, "/home/user/etc/passwd", localMedium.sandboxedPath("/etc/passwd"))
}

func TestLocal_Path_RootFilesystem_Good(t *core.T) {
	localMedium := &Medium{filesystemRoot: "/"}

	core.AssertEqual(t, "/etc/passwd", localMedium.sandboxedPath("/etc/passwd"))
	core.AssertEqual(t, "/home/user/file.txt", localMedium.sandboxedPath("/home/user/file.txt"))

	workingDirectory := currentWorkingDir()
	core.AssertEqual(t, core.Path(workingDirectory, "file.txt"), localMedium.sandboxedPath("file.txt"))
}

func TestLocal_ReadWrite_BasicGood(t *core.T) {
	root := t.TempDir()
	localMedium, _ := New(root)

	err := localMedium.Write("test.txt", "hello")
	core.AssertNoError(t, err)

	content, err := localMedium.Read("test.txt")
	core.AssertNoError(t, err)
	core.AssertEqual(t, "hello", content)

	err = localMedium.Write("a/b/c.txt", "nested")
	core.AssertNoError(t, err)

	content, err = localMedium.Read("a/b/c.txt")
	core.AssertNoError(t, err)
	core.AssertEqual(t, "nested", content)

	_, err = localMedium.Read("nope.txt")
	core.AssertError(t, err)
}

func TestLocal_EnsureDir_Basic_Good(t *core.T) {
	root := t.TempDir()
	localMedium, _ := New(root)

	err := localMedium.EnsureDir("one/two/three")
	core.AssertNoError(t, err)

	info, err := localMedium.Stat("one/two/three")
	core.AssertNoError(t, err)
	core.AssertTrue(t, info.IsDir())
}

func TestLocal_IsDir_Basic_Good(t *core.T) {
	root := t.TempDir()
	localMedium, _ := New(root)

	_ = localMedium.EnsureDir("mydir")
	_ = localMedium.Write("myfile", "x")

	core.AssertTrue(t, localMedium.IsDir("mydir"))
	core.AssertFalse(t, localMedium.IsDir("myfile"))
	core.AssertFalse(t, localMedium.IsDir("nope"))
	core.AssertFalse(t, localMedium.IsDir(""))
}

func TestLocal_IsFile_Basic_Good(t *core.T) {
	root := t.TempDir()
	localMedium, _ := New(root)

	_ = localMedium.EnsureDir("mydir")
	_ = localMedium.Write("myfile", "x")

	core.AssertTrue(t, localMedium.IsFile("myfile"))
	core.AssertFalse(t, localMedium.IsFile("mydir"))
	core.AssertFalse(t, localMedium.IsFile("nope"))
	core.AssertFalse(t, localMedium.IsFile(""))
}

func TestLocal_Exists_Basic_Good(t *core.T) {
	root := t.TempDir()
	localMedium, _ := New(root)

	_ = localMedium.Write("exists", "x")

	core.AssertTrue(t, localMedium.Exists("exists"))
	core.AssertFalse(t, localMedium.Exists("nope"))
}

func TestLocal_List_Basic_Good(t *core.T) {
	root := t.TempDir()
	localMedium, _ := New(root)

	_ = localMedium.Write("a.txt", "a")
	_ = localMedium.Write("b.txt", "b")
	_ = localMedium.EnsureDir("subdir")

	entries, err := localMedium.List("")
	core.AssertNoError(t, err)
	core.AssertLen(t, entries, 3)
}

func TestLocal_Stat_Basic_Good(t *core.T) {
	root := t.TempDir()
	localMedium, _ := New(root)

	_ = localMedium.Write("file", "content")

	info, err := localMedium.Stat("file")
	core.AssertNoError(t, err)
	core.AssertEqual(t, int64(7), info.Size())
}

func TestLocal_Delete_Basic_Good(t *core.T) {
	root := t.TempDir()
	localMedium, _ := New(root)

	_ = localMedium.Write("todelete", "x")
	core.AssertTrue(t, localMedium.Exists("todelete"))

	err := localMedium.Delete("todelete")
	core.AssertNoError(t, err)
	core.AssertFalse(t, localMedium.Exists("todelete"))
}

func TestLocal_DeleteAll_Basic_Good(t *core.T) {
	root := t.TempDir()
	localMedium, _ := New(root)

	_ = localMedium.Write("dir/sub/file", "x")

	err := localMedium.DeleteAll("dir")
	core.AssertNoError(t, err)
	core.AssertFalse(t, localMedium.Exists("dir"))
}

func TestLocal_Delete_ProtectedHomeViaSymlinkEnv_Bad(t *core.T) {
	realHome := t.TempDir()
	linkParent := t.TempDir()
	homeLink := core.Path(linkParent, "home-link")
	core.RequireNoError(t, syscall.Symlink(realHome, homeLink))
	t.Setenv("HOME", homeLink)

	localMedium, err := New("/")
	core.RequireNoError(t, err)

	err = localMedium.Delete(realHome)
	core.AssertError(t, err)
	core.AssertTrue(t, localMedium.IsDir(realHome))
}

func TestLocal_DeleteAll_ProtectedHomeViaEnv_Bad(t *core.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	localMedium, err := New("/")
	core.RequireNoError(t, err)

	err = localMedium.DeleteAll(tempHome)
	core.AssertError(t, err)
	core.AssertTrue(t, localMedium.IsDir(tempHome))
}

func TestLocal_Rename_Basic_Good(t *core.T) {
	root := t.TempDir()
	localMedium, _ := New(root)

	_ = localMedium.Write("old", "x")

	err := localMedium.Rename("old", "new")
	core.AssertNoError(t, err)
	core.AssertFalse(t, localMedium.Exists("old"))
	core.AssertTrue(t, localMedium.Exists("new"))
}

func TestLocal_Delete_Good(t *core.T) {
	testRoot := t.TempDir()

	localMedium, err := New(testRoot)
	core.AssertNoError(t, err)

	err = localMedium.Write("file.txt", "content")
	core.AssertNoError(t, err)
	core.AssertTrue(t, localMedium.IsFile("file.txt"))

	err = localMedium.Delete("file.txt")
	core.AssertNoError(t, err)
	core.AssertFalse(t, localMedium.IsFile("file.txt"))

	err = localMedium.EnsureDir("emptydir")
	core.AssertNoError(t, err)
	err = localMedium.Delete("emptydir")
	core.AssertNoError(t, err)
	core.AssertFalse(t, localMedium.IsDir("emptydir"))
}

func TestLocal_Delete_NotEmpty_Bad(t *core.T) {
	testRoot := t.TempDir()

	localMedium, err := New(testRoot)
	core.AssertNoError(t, err)

	err = localMedium.Write("mydir/file.txt", "content")
	core.AssertNoError(t, err)

	err = localMedium.Delete("mydir")
	core.AssertError(t, err)
}

func TestLocal_DeleteAll_Good(t *core.T) {
	testRoot := t.TempDir()

	localMedium, err := New(testRoot)
	core.AssertNoError(t, err)

	err = localMedium.Write("mydir/file1.txt", "content1")
	core.AssertNoError(t, err)
	err = localMedium.Write("mydir/subdir/file2.txt", "content2")
	core.AssertNoError(t, err)

	err = localMedium.DeleteAll("mydir")
	core.AssertNoError(t, err)
	core.AssertFalse(t, localMedium.Exists("mydir"))
	core.AssertFalse(t, localMedium.Exists("mydir/file1.txt"))
	core.AssertFalse(t, localMedium.Exists("mydir/subdir/file2.txt"))
}

func TestLocal_Rename_Good(t *core.T) {
	testRoot := t.TempDir()

	localMedium, err := New(testRoot)
	core.AssertNoError(t, err)

	err = localMedium.Write("old.txt", "content")
	core.AssertNoError(t, err)
	err = localMedium.Rename("old.txt", "new.txt")
	core.AssertNoError(t, err)
	core.AssertFalse(t, localMedium.IsFile("old.txt"))
	core.AssertTrue(t, localMedium.IsFile("new.txt"))

	content, err := localMedium.Read("new.txt")
	core.AssertNoError(t, err)
	core.AssertEqual(t, "content", content)
}

func TestLocal_Rename_TraversalSanitised_Good(t *core.T) {
	testRoot := t.TempDir()

	localMedium, err := New(testRoot)
	core.AssertNoError(t, err)

	err = localMedium.Write("file.txt", "content")
	core.AssertNoError(t, err)

	err = localMedium.Rename("file.txt", "../escaped.txt")
	core.AssertNoError(t, err)
	core.AssertFalse(t, localMedium.Exists("file.txt"))
	core.AssertTrue(t, localMedium.Exists("escaped.txt"))
}

func TestLocal_List_Good(t *core.T) {
	testRoot := t.TempDir()

	localMedium, err := New(testRoot)
	core.AssertNoError(t, err)

	err = localMedium.Write("file1.txt", "content1")
	core.AssertNoError(t, err)
	err = localMedium.Write("file2.txt", "content2")
	core.AssertNoError(t, err)
	err = localMedium.EnsureDir("subdir")
	core.AssertNoError(t, err)

	entries, err := localMedium.List(".")
	core.AssertNoError(t, err)
	core.AssertLen(t, entries, 3)

	names := make(map[string]bool)
	for _, entry := range entries {
		names[entry.Name()] = true
	}
	core.AssertTrue(t, names["file1.txt"])
	core.AssertTrue(t, names["file2.txt"])
	core.AssertTrue(t, names["subdir"])
}

func TestLocal_Stat_Good(t *core.T) {
	testRoot := t.TempDir()

	localMedium, err := New(testRoot)
	core.AssertNoError(t, err)

	err = localMedium.Write("file.txt", "hello world")
	core.AssertNoError(t, err)
	info, err := localMedium.Stat("file.txt")
	core.AssertNoError(t, err)
	core.AssertEqual(t, "file.txt", info.Name())
	core.AssertEqual(t, int64(11), info.Size())
	core.AssertFalse(t, info.IsDir())

	err = localMedium.EnsureDir("mydir")
	core.AssertNoError(t, err)
	info, err = localMedium.Stat("mydir")
	core.AssertNoError(t, err)
	core.AssertEqual(t, "mydir", info.Name())
	core.AssertTrue(t, info.IsDir())
}

func TestLocal_Exists_Good(t *core.T) {
	testRoot := t.TempDir()

	localMedium, err := New(testRoot)
	core.AssertNoError(t, err)

	core.AssertFalse(t, localMedium.Exists("nonexistent"))

	err = localMedium.Write("file.txt", "content")
	core.AssertNoError(t, err)
	core.AssertTrue(t, localMedium.Exists("file.txt"))

	err = localMedium.EnsureDir("mydir")
	core.AssertNoError(t, err)
	core.AssertTrue(t, localMedium.Exists("mydir"))
}

func TestLocal_IsDir_Good(t *core.T) {
	testRoot := t.TempDir()

	localMedium, err := New(testRoot)
	core.AssertNoError(t, err)

	err = localMedium.Write("file.txt", "content")
	core.AssertNoError(t, err)
	core.AssertFalse(t, localMedium.IsDir("file.txt"))

	err = localMedium.EnsureDir("mydir")
	core.AssertNoError(t, err)
	core.AssertTrue(t, localMedium.IsDir("mydir"))

	core.AssertFalse(t, localMedium.IsDir("nonexistent"))
}

func TestLocal_ReadStream_Basic_Good(t *core.T) {
	root := t.TempDir()
	localMedium, _ := New(root)

	content := "streaming content"
	err := localMedium.Write("stream.txt", content)
	core.AssertNoError(t, err)

	reader, err := localMedium.ReadStream("stream.txt")
	core.AssertNoError(t, err)
	defer reader.Close()

	limitReader := goio.LimitReader(reader, 9)
	data, err := goio.ReadAll(limitReader)
	core.AssertNoError(t, err)
	core.AssertEqual(t, "streaming", string(data))
}

func TestLocal_WriteStream_Basic_Good(t *core.T) {
	root := t.TempDir()
	localMedium, _ := New(root)

	writer, err := localMedium.WriteStream("output.txt")
	core.AssertNoError(t, err)

	_, err = goio.Copy(writer, core.NewReader("piped data"))
	core.AssertNoError(t, err)
	err = writer.Close()
	core.AssertNoError(t, err)

	content, err := localMedium.Read("output.txt")
	core.AssertNoError(t, err)
	core.AssertEqual(t, "piped data", content)
}

func TestLocal_Path_TraversalSandboxGood(t *core.T) {
	localMedium := &Medium{filesystemRoot: "/sandbox"}

	core.AssertEqual(t, "/sandbox/file.txt", localMedium.sandboxedPath("../../../file.txt"))
	core.AssertEqual(t, "/sandbox/target", localMedium.sandboxedPath("dir/../../target"))

	core.AssertEqual(t, "/sandbox/.ssh/id_rsa", localMedium.sandboxedPath(".ssh/id_rsa"))
	core.AssertEqual(t, "/sandbox/id_rsa", localMedium.sandboxedPath(".ssh/../id_rsa"))

	core.AssertEqual(t, "/sandbox/file\x00.txt", localMedium.sandboxedPath("file\x00.txt"))
}

func TestLocal_ValidatePath_SymlinkEscapeBad(t *core.T) {
	root := t.TempDir()
	localMedium, err := New(root)
	core.AssertNoError(t, err)

	outside := t.TempDir()
	outsideFile := core.Path(outside, "secret.txt")
	outsideMedium, err := New("/")
	core.RequireNoError(t, err)
	err = outsideMedium.Write(outsideFile, "secret")
	core.AssertNoError(t, err)

	_, err = localMedium.validatePath("../outside.txt")
	core.AssertNoError(t, err)

	linkPath := core.Path(root, "evil_link")
	err = syscall.Symlink(outside, linkPath)
	core.AssertNoError(t, err)

	_, err = localMedium.validatePath("evil_link/secret.txt")
	core.AssertError(t, err)
	core.AssertErrorIs(t, err, fs.ErrPermission)

	err = localMedium.EnsureDir("inner")
	core.AssertNoError(t, err)
	innerDir := core.Path(root, "inner")
	nestedLink := core.Path(innerDir, "nested_evil")
	err = syscall.Symlink(outside, nestedLink)
	core.AssertNoError(t, err)

	_, err = localMedium.validatePath("inner/nested_evil/secret.txt")
	core.AssertError(t, err)
	core.AssertErrorIs(t, err, fs.ErrPermission)
}

func TestLocal_EmptyPathsGood(t *core.T) {
	root := t.TempDir()
	localMedium, err := New(root)
	core.AssertNoError(t, err)

	_, err = localMedium.Read("")
	core.AssertError(t, err)

	err = localMedium.Write("", "content")
	core.AssertError(t, err)

	err = localMedium.EnsureDir("")
	core.AssertNoError(t, err)

	core.AssertFalse(t, localMedium.IsDir(""))

	core.AssertTrue(t, localMedium.Exists(""))

	entries, err := localMedium.List("")
	core.AssertNoError(t, err)
	core.AssertNotNil(t, entries)
}

func newLocalMediumFixture(t *core.T) *Medium {
	t.Helper()
	medium, err := New(t.TempDir())
	core.RequireNoError(t, err)
	return medium
}

func TestMedium_New_Good(t *core.T) {
	root := t.TempDir()
	medium, err := New(root)
	core.AssertNoError(t, err)
	core.AssertNotNil(t, medium)
}

func TestMedium_New_Bad(t *core.T) {
	medium, err := New("")
	core.AssertNoError(t, err)
	core.AssertNotEmpty(t, medium.filesystemRoot)
}

func TestMedium_New_Ugly(t *core.T) {
	medium, err := New(".")
	core.AssertNoError(t, err)
	core.AssertNotNil(t, medium)
}

func TestMedium_Medium_Read_Good(t *core.T) {
	medium := newLocalMediumFixture(t)
	core.RequireNoError(t, medium.Write("read.txt", "local"))
	got, err := medium.Read("read.txt")
	core.AssertNoError(t, err)
	core.AssertEqual(t, "local", got)
}

func TestMedium_Medium_Read_Bad(t *core.T) {
	medium := newLocalMediumFixture(t)
	got, err := medium.Read("missing.txt")
	core.AssertError(t, err)
	core.AssertEqual(t, "", got)
}

func TestMedium_Medium_Read_Ugly(t *core.T) {
	medium := newLocalMediumFixture(t)
	got, err := medium.Read("../missing.txt")
	core.AssertError(t, err)
	core.AssertEqual(t, "", got)
}

func TestMedium_Medium_Write_Good(t *core.T) {
	medium := newLocalMediumFixture(t)
	err := medium.Write("write.txt", "local")
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.IsFile("write.txt"))
}

func TestMedium_Medium_Write_Bad(t *core.T) {
	medium := newLocalMediumFixture(t)
	err := medium.Write("", "local")
	core.AssertError(t, err)
	core.AssertFalse(t, medium.IsFile(""))
}

func TestMedium_Medium_Write_Ugly(t *core.T) {
	medium := newLocalMediumFixture(t)
	err := medium.Write("nested/write.txt", "local")
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.IsFile("nested/write.txt"))
}

func TestMedium_Medium_WriteMode_Good(t *core.T) {
	medium := newLocalMediumFixture(t)
	err := medium.WriteMode("mode.txt", "local", 0600)
	info, statErr := medium.Stat("mode.txt")
	core.AssertNoError(t, err)
	core.AssertNoError(t, statErr)
	core.AssertEqual(t, fs.FileMode(0600), info.Mode().Perm())
}

func TestMedium_Medium_WriteMode_Bad(t *core.T) {
	medium := newLocalMediumFixture(t)
	err := medium.WriteMode("", "local", 0600)
	core.AssertError(t, err)
	core.AssertFalse(t, medium.IsFile(""))
}

func TestMedium_Medium_WriteMode_Ugly(t *core.T) {
	medium := newLocalMediumFixture(t)
	err := medium.WriteMode("zero-mode.txt", "local", 0)
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.IsFile("zero-mode.txt"))
}

func TestMedium_Medium_EnsureDir_Good(t *core.T) {
	medium := newLocalMediumFixture(t)
	err := medium.EnsureDir("dir")
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.IsDir("dir"))
}

func TestMedium_Medium_EnsureDir_Bad(t *core.T) {
	medium := newLocalMediumFixture(t)
	core.RequireNoError(t, medium.Write("file", "local"))
	err := medium.EnsureDir("file")
	core.AssertError(t, err)
}

func TestMedium_Medium_EnsureDir_Ugly(t *core.T) {
	medium := newLocalMediumFixture(t)
	err := medium.EnsureDir("a/b/c")
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.IsDir("a/b/c"))
}

func TestMedium_Medium_IsFile_Good(t *core.T) {
	medium := newLocalMediumFixture(t)
	core.RequireNoError(t, medium.Write("file.txt", "local"))
	got := medium.IsFile("file.txt")
	core.AssertTrue(t, got)
}

func TestMedium_Medium_IsFile_Bad(t *core.T) {
	medium := newLocalMediumFixture(t)
	got := medium.IsFile("missing.txt")
	core.AssertFalse(t, got)
}

func TestMedium_Medium_IsFile_Ugly(t *core.T) {
	medium := newLocalMediumFixture(t)
	core.RequireNoError(t, medium.EnsureDir("dir"))
	got := medium.IsFile("dir")
	core.AssertFalse(t, got)
}

func TestMedium_Medium_Delete_Good(t *core.T) {
	medium := newLocalMediumFixture(t)
	core.RequireNoError(t, medium.Write("delete.txt", "local"))
	err := medium.Delete("delete.txt")
	core.AssertNoError(t, err)
	core.AssertFalse(t, medium.Exists("delete.txt"))
}

func TestMedium_Medium_Delete_Bad(t *core.T) {
	medium := newLocalMediumFixture(t)
	err := medium.Delete("missing.txt")
	core.AssertError(t, err)
	core.AssertFalse(t, medium.Exists("missing.txt"))
}

func TestMedium_Medium_Delete_Ugly(t *core.T) {
	medium := newLocalMediumFixture(t)
	err := medium.Delete("")
	core.AssertError(t, err)
	core.AssertTrue(t, medium.Exists(""))
}

func TestMedium_Medium_DeleteAll_Good(t *core.T) {
	medium := newLocalMediumFixture(t)
	core.RequireNoError(t, medium.Write("tree/file.txt", "local"))
	err := medium.DeleteAll("tree")
	core.AssertNoError(t, err)
	core.AssertFalse(t, medium.Exists("tree/file.txt"))
}

func TestMedium_Medium_DeleteAll_Bad(t *core.T) {
	medium := newLocalMediumFixture(t)
	err := medium.DeleteAll("missing")
	core.AssertNoError(t, err)
	core.AssertFalse(t, medium.Exists("missing"))
}

func TestMedium_Medium_DeleteAll_Ugly(t *core.T) {
	medium := newLocalMediumFixture(t)
	err := medium.DeleteAll("")
	core.AssertError(t, err)
	core.AssertTrue(t, medium.Exists(""))
}

func TestMedium_Medium_Rename_Good(t *core.T) {
	medium := newLocalMediumFixture(t)
	core.RequireNoError(t, medium.Write("old.txt", "local"))
	err := medium.Rename("old.txt", "new.txt")
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.IsFile("new.txt"))
}

func TestMedium_Medium_Rename_Bad(t *core.T) {
	medium := newLocalMediumFixture(t)
	err := medium.Rename("missing.txt", "new.txt")
	core.AssertError(t, err)
	core.AssertFalse(t, medium.Exists("new.txt"))
}

func TestMedium_Medium_Rename_Ugly(t *core.T) {
	medium := newLocalMediumFixture(t)
	core.RequireNoError(t, medium.Write("old.txt", "local"))
	err := medium.Rename("old.txt", "../new.txt")
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.Exists("new.txt"))
}

func TestMedium_Medium_List_Good(t *core.T) {
	medium := newLocalMediumFixture(t)
	core.RequireNoError(t, medium.Write("dir/a.txt", "a"))
	entries, err := medium.List("dir")
	core.AssertNoError(t, err)
	core.AssertLen(t, entries, 1)
}

func TestMedium_Medium_List_Bad(t *core.T) {
	medium := newLocalMediumFixture(t)
	entries, err := medium.List("missing")
	core.AssertError(t, err)
	core.AssertNil(t, entries)
}

func TestMedium_Medium_List_Ugly(t *core.T) {
	medium := newLocalMediumFixture(t)
	entries, err := medium.List("")
	core.AssertNoError(t, err)
	core.AssertNotNil(t, entries)
}

func TestMedium_Medium_Stat_Good(t *core.T) {
	medium := newLocalMediumFixture(t)
	core.RequireNoError(t, medium.Write("stat.txt", "local"))
	info, err := medium.Stat("stat.txt")
	core.AssertNoError(t, err)
	core.AssertEqual(t, "stat.txt", info.Name())
}

func TestMedium_Medium_Stat_Bad(t *core.T) {
	medium := newLocalMediumFixture(t)
	info, err := medium.Stat("missing.txt")
	core.AssertError(t, err)
	core.AssertNil(t, info)
}

func TestMedium_Medium_Stat_Ugly(t *core.T) {
	medium := newLocalMediumFixture(t)
	info, err := medium.Stat("")
	core.AssertNoError(t, err)
	core.AssertTrue(t, info.IsDir())
}

func TestMedium_Medium_Open_Good(t *core.T) {
	medium := newLocalMediumFixture(t)
	core.RequireNoError(t, medium.Write("open.txt", "local"))
	file, err := medium.Open("open.txt")
	core.AssertNoError(t, err)
	core.AssertNotNil(t, file)
	core.RequireNoError(t, file.Close())
}

func TestMedium_Medium_Open_Bad(t *core.T) {
	medium := newLocalMediumFixture(t)
	file, err := medium.Open("missing.txt")
	core.AssertError(t, err)
	core.AssertNil(t, file)
}

func TestMedium_Medium_Open_Ugly(t *core.T) {
	medium := newLocalMediumFixture(t)
	file, err := medium.Open("")
	core.AssertNoError(t, err)
	core.AssertNotNil(t, file)
	core.RequireNoError(t, file.Close())
}

func TestMedium_Medium_Create_Good(t *core.T) {
	medium := newLocalMediumFixture(t)
	writer, err := medium.Create("create.txt")
	core.RequireNoError(t, err)
	_, writeErr := writer.Write([]byte("local"))
	core.AssertNoError(t, writeErr)
	core.AssertNoError(t, writer.Close())
}

func TestMedium_Medium_Create_Bad(t *core.T) {
	medium := newLocalMediumFixture(t)
	writer, err := medium.Create("")
	core.AssertError(t, err)
	core.AssertNil(t, writer)
}

func TestMedium_Medium_Create_Ugly(t *core.T) {
	medium := newLocalMediumFixture(t)
	writer, err := medium.Create("nested/create.txt")
	core.RequireNoError(t, err)
	_, writeErr := writer.Write([]byte("local"))
	core.AssertNoError(t, writeErr)
	core.AssertNoError(t, writer.Close())
}

func TestMedium_Medium_Append_Good(t *core.T) {
	medium := newLocalMediumFixture(t)
	core.RequireNoError(t, medium.Write("append.txt", "a"))
	writer, err := medium.Append("append.txt")
	core.RequireNoError(t, err)
	_, writeErr := writer.Write([]byte("b"))
	core.AssertNoError(t, writeErr)
	core.AssertNoError(t, writer.Close())
}

func TestMedium_Medium_Append_Bad(t *core.T) {
	medium := newLocalMediumFixture(t)
	writer, err := medium.Append("")
	core.AssertError(t, err)
	core.AssertNil(t, writer)
}

func TestMedium_Medium_Append_Ugly(t *core.T) {
	medium := newLocalMediumFixture(t)
	writer, err := medium.Append("new.txt")
	core.RequireNoError(t, err)
	_, writeErr := writer.Write([]byte("new"))
	core.AssertNoError(t, writeErr)
	core.AssertNoError(t, writer.Close())
}

func TestMedium_Medium_ReadStream_Good(t *core.T) {
	medium := newLocalMediumFixture(t)
	core.RequireNoError(t, medium.Write("stream.txt", "local"))
	reader, err := medium.ReadStream("stream.txt")
	core.RequireNoError(t, err)
	data, readErr := goio.ReadAll(reader)
	core.AssertNoError(t, readErr)
	core.AssertEqual(t, "local", string(data))
}

func TestMedium_Medium_ReadStream_Bad(t *core.T) {
	medium := newLocalMediumFixture(t)
	reader, err := medium.ReadStream("missing.txt")
	core.AssertError(t, err)
	core.AssertNil(t, reader)
}

func TestMedium_Medium_ReadStream_Ugly(t *core.T) {
	medium := newLocalMediumFixture(t)
	reader, err := medium.ReadStream("")
	core.AssertNoError(t, err)
	core.AssertNotNil(t, reader)
	core.RequireNoError(t, reader.Close())
}

func TestMedium_Medium_WriteStream_Good(t *core.T) {
	medium := newLocalMediumFixture(t)
	writer, err := medium.WriteStream("stream.txt")
	core.RequireNoError(t, err)
	_, writeErr := writer.Write([]byte("local"))
	core.AssertNoError(t, writeErr)
	core.AssertNoError(t, writer.Close())
}

func TestMedium_Medium_WriteStream_Bad(t *core.T) {
	medium := newLocalMediumFixture(t)
	writer, err := medium.WriteStream("")
	core.AssertError(t, err)
	core.AssertNil(t, writer)
}

func TestMedium_Medium_WriteStream_Ugly(t *core.T) {
	medium := newLocalMediumFixture(t)
	writer, err := medium.WriteStream("nested/stream.txt")
	core.RequireNoError(t, err)
	_, writeErr := writer.Write([]byte("local"))
	core.AssertNoError(t, writeErr)
	core.AssertNoError(t, writer.Close())
}

func TestMedium_Medium_Exists_Good(t *core.T) {
	medium := newLocalMediumFixture(t)
	core.RequireNoError(t, medium.Write("exists.txt", "local"))
	got := medium.Exists("exists.txt")
	core.AssertTrue(t, got)
}

func TestMedium_Medium_Exists_Bad(t *core.T) {
	medium := newLocalMediumFixture(t)
	got := medium.Exists("missing.txt")
	core.AssertFalse(t, got)
}

func TestMedium_Medium_Exists_Ugly(t *core.T) {
	medium := newLocalMediumFixture(t)
	got := medium.Exists("")
	core.AssertTrue(t, got)
}

func TestMedium_Medium_IsDir_Good(t *core.T) {
	medium := newLocalMediumFixture(t)
	core.RequireNoError(t, medium.EnsureDir("dir"))
	got := medium.IsDir("dir")
	core.AssertTrue(t, got)
}

func TestMedium_Medium_IsDir_Bad(t *core.T) {
	medium := newLocalMediumFixture(t)
	got := medium.IsDir("missing")
	core.AssertFalse(t, got)
}

func TestMedium_Medium_IsDir_Ugly(t *core.T) {
	medium := newLocalMediumFixture(t)
	got := medium.IsDir("")
	core.AssertFalse(t, got)
}
