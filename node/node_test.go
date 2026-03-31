package node

import (
	"archive/tar"
	"bytes"
	"io"
	"io/fs"
	"sort"
	"testing"

	core "dappco.re/go/core"
	coreio "dappco.re/go/core/io"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNode_New_Good(t *testing.T) {
	nodeTree := New()
	require.NotNil(t, nodeTree, "New() must not return nil")
	assert.NotNil(t, nodeTree.files, "New() must initialise the files map")
}

func TestNode_AddData_Good(t *testing.T) {
	nodeTree := New()
	nodeTree.AddData("foo.txt", []byte("foo"))

	file, ok := nodeTree.files["foo.txt"]
	require.True(t, ok, "file foo.txt should be present")
	assert.Equal(t, []byte("foo"), file.content)

	info, err := file.Stat()
	require.NoError(t, err)
	assert.Equal(t, "foo.txt", info.Name())
}

func TestNode_AddData_Bad(t *testing.T) {
	nodeTree := New()

	nodeTree.AddData("", []byte("data"))
	assert.Empty(t, nodeTree.files, "empty name must not be stored")

	nodeTree.AddData("dir/", nil)
	assert.Empty(t, nodeTree.files, "directory entry must not be stored")
}

func TestNode_AddData_EdgeCases_Good(t *testing.T) {
	t.Run("Overwrite", func(t *testing.T) {
		nodeTree := New()
		nodeTree.AddData("foo.txt", []byte("foo"))
		nodeTree.AddData("foo.txt", []byte("bar"))

		file := nodeTree.files["foo.txt"]
		assert.Equal(t, []byte("bar"), file.content, "second AddData should overwrite")
	})

	t.Run("LeadingSlash", func(t *testing.T) {
		nodeTree := New()
		nodeTree.AddData("/hello.txt", []byte("hi"))
		_, ok := nodeTree.files["hello.txt"]
		assert.True(t, ok, "leading slash should be trimmed")
	})
}

func TestNode_Open_Good(t *testing.T) {
	nodeTree := New()
	nodeTree.AddData("foo.txt", []byte("foo"))

	file, err := nodeTree.Open("foo.txt")
	require.NoError(t, err)
	defer file.Close()

	buf := make([]byte, 10)
	nr, err := file.Read(buf)
	require.True(t, nr > 0 || err == io.EOF)
	assert.Equal(t, "foo", string(buf[:nr]))
}

func TestNode_Open_Bad(t *testing.T) {
	nodeTree := New()
	_, err := nodeTree.Open("nonexistent.txt")
	require.Error(t, err)
	assert.ErrorIs(t, err, fs.ErrNotExist)
}

func TestNode_Open_Directory_Good(t *testing.T) {
	nodeTree := New()
	nodeTree.AddData("bar/baz.txt", []byte("baz"))

	file, err := nodeTree.Open("bar")
	require.NoError(t, err)
	defer file.Close()

	_, err = file.Read(make([]byte, 1))
	require.Error(t, err)

	var pathErr *fs.PathError
	require.True(t, core.As(err, &pathErr))
	assert.Equal(t, fs.ErrInvalid, pathErr.Err)
}

func TestNode_Stat_Good(t *testing.T) {
	nodeTree := New()
	nodeTree.AddData("foo.txt", []byte("foo"))
	nodeTree.AddData("bar/baz.txt", []byte("baz"))

	info, err := nodeTree.Stat("bar/baz.txt")
	require.NoError(t, err)
	assert.Equal(t, "baz.txt", info.Name())
	assert.Equal(t, int64(3), info.Size())
	assert.False(t, info.IsDir())

	dirInfo, err := nodeTree.Stat("bar")
	require.NoError(t, err)
	assert.True(t, dirInfo.IsDir())
	assert.Equal(t, "bar", dirInfo.Name())
}

func TestNode_Stat_Bad(t *testing.T) {
	nodeTree := New()
	_, err := nodeTree.Stat("nonexistent")
	require.Error(t, err)
	assert.ErrorIs(t, err, fs.ErrNotExist)
}

func TestNode_Stat_RootDirectory_Good(t *testing.T) {
	nodeTree := New()
	nodeTree.AddData("foo.txt", []byte("foo"))

	info, err := nodeTree.Stat(".")
	require.NoError(t, err)
	assert.True(t, info.IsDir())
	assert.Equal(t, ".", info.Name())
}

func TestNode_ReadFile_Good(t *testing.T) {
	nodeTree := New()
	nodeTree.AddData("hello.txt", []byte("hello world"))

	data, err := nodeTree.ReadFile("hello.txt")
	require.NoError(t, err)
	assert.Equal(t, []byte("hello world"), data)
}

func TestNode_ReadFile_Bad(t *testing.T) {
	nodeTree := New()
	_, err := nodeTree.ReadFile("missing.txt")
	require.Error(t, err)
	assert.ErrorIs(t, err, fs.ErrNotExist)
}

func TestNode_ReadFile_ReturnsCopy_Good(t *testing.T) {
	nodeTree := New()
	nodeTree.AddData("data.bin", []byte("original"))

	data, err := nodeTree.ReadFile("data.bin")
	require.NoError(t, err)
	data[0] = 'X'

	data2, err := nodeTree.ReadFile("data.bin")
	require.NoError(t, err)
	assert.Equal(t, []byte("original"), data2, "ReadFile must return an independent copy")
}

func TestNode_ReadDir_Good(t *testing.T) {
	nodeTree := New()
	nodeTree.AddData("foo.txt", []byte("foo"))
	nodeTree.AddData("bar/baz.txt", []byte("baz"))
	nodeTree.AddData("bar/qux.txt", []byte("qux"))

	entries, err := nodeTree.ReadDir(".")
	require.NoError(t, err)
	assert.Equal(t, []string{"bar", "foo.txt"}, sortedNames(entries))

	barEntries, err := nodeTree.ReadDir("bar")
	require.NoError(t, err)
	assert.Equal(t, []string{"baz.txt", "qux.txt"}, sortedNames(barEntries))
}

func TestNode_ReadDir_Bad(t *testing.T) {
	nodeTree := New()
	nodeTree.AddData("foo.txt", []byte("foo"))

	_, err := nodeTree.ReadDir("foo.txt")
	require.Error(t, err)
	var pathErr *fs.PathError
	require.True(t, core.As(err, &pathErr))
	assert.Equal(t, fs.ErrInvalid, pathErr.Err)
}

func TestNode_ReadDir_IgnoresEmptyEntry_Good(t *testing.T) {
	nodeTree := New()
	nodeTree.AddData("bar/baz.txt", []byte("baz"))
	nodeTree.AddData("empty_dir/", nil)

	entries, err := nodeTree.ReadDir(".")
	require.NoError(t, err)
	assert.Equal(t, []string{"bar"}, sortedNames(entries))
}

func TestNode_Exists_Good(t *testing.T) {
	nodeTree := New()
	nodeTree.AddData("foo.txt", []byte("foo"))
	nodeTree.AddData("bar/baz.txt", []byte("baz"))

	assert.True(t, nodeTree.Exists("foo.txt"))
	assert.True(t, nodeTree.Exists("bar"))
}

func TestNode_Exists_Bad(t *testing.T) {
	nodeTree := New()
	assert.False(t, nodeTree.Exists("nonexistent"))
}

func TestNode_Exists_RootAndEmptyPath_Good(t *testing.T) {
	nodeTree := New()
	nodeTree.AddData("dummy.txt", []byte("dummy"))

	assert.True(t, nodeTree.Exists("."), "root '.' must exist")
	assert.True(t, nodeTree.Exists(""), "empty path (root) must exist")
}

func TestNode_Walk_Default_Good(t *testing.T) {
	nodeTree := New()
	nodeTree.AddData("foo.txt", []byte("foo"))
	nodeTree.AddData("bar/baz.txt", []byte("baz"))
	nodeTree.AddData("bar/qux.txt", []byte("qux"))

	var paths []string
	err := nodeTree.Walk(".", func(p string, d fs.DirEntry, err error) error {
		paths = append(paths, p)
		return nil
	}, WalkOptions{})
	require.NoError(t, err)

	sort.Strings(paths)
	assert.Equal(t, []string{".", "bar", "bar/baz.txt", "bar/qux.txt", "foo.txt"}, paths)
}

func TestNode_Walk_Default_Bad(t *testing.T) {
	nodeTree := New()

	var called bool
	err := nodeTree.Walk("nonexistent", func(p string, d fs.DirEntry, err error) error {
		called = true
		assert.Error(t, err)
		assert.ErrorIs(t, err, fs.ErrNotExist)
		return err
	}, WalkOptions{})
	assert.True(t, called, "walk function must be called for nonexistent root")
	assert.ErrorIs(t, err, fs.ErrNotExist)
}

func TestNode_Walk_CallbackError_Good(t *testing.T) {
	nodeTree := New()
	nodeTree.AddData("a/b.txt", []byte("b"))
	nodeTree.AddData("a/c.txt", []byte("c"))

	walkErr := core.NewError("stop walking")
	var paths []string
	err := nodeTree.Walk(".", func(p string, d fs.DirEntry, err error) error {
		if p == "a/b.txt" {
			return walkErr
		}
		paths = append(paths, p)
		return nil
	}, WalkOptions{})

	assert.Equal(t, walkErr, err, "Walk must propagate the callback error")
}

func TestNode_Walk_Good(t *testing.T) {
	nodeTree := New()
	nodeTree.AddData("root.txt", []byte("root"))
	nodeTree.AddData("a/a1.txt", []byte("a1"))
	nodeTree.AddData("a/b/b1.txt", []byte("b1"))
	nodeTree.AddData("c/c1.txt", []byte("c1"))

	t.Run("MaxDepth", func(t *testing.T) {
		var paths []string
		err := nodeTree.Walk(".", func(p string, d fs.DirEntry, err error) error {
			paths = append(paths, p)
			return nil
		}, WalkOptions{MaxDepth: 1})
		require.NoError(t, err)

		sort.Strings(paths)
		assert.Equal(t, []string{".", "a", "c", "root.txt"}, paths)
	})

	t.Run("Filter", func(t *testing.T) {
		var paths []string
		err := nodeTree.Walk(".", func(p string, d fs.DirEntry, err error) error {
			paths = append(paths, p)
			return nil
		}, WalkOptions{Filter: func(p string, d fs.DirEntry) bool {
			return !core.HasPrefix(p, "a")
		}})
		require.NoError(t, err)

		sort.Strings(paths)
		assert.Equal(t, []string{".", "c", "c/c1.txt", "root.txt"}, paths)
	})

	t.Run("SkipErrors", func(t *testing.T) {
		var called bool
		err := nodeTree.Walk("nonexistent", func(p string, d fs.DirEntry, err error) error {
			called = true
			return err
		}, WalkOptions{SkipErrors: true})

		assert.NoError(t, err, "SkipErrors should suppress the error")
		assert.False(t, called, "callback should not be called when error is skipped")
	})
}

func TestNode_CopyFile_Good(t *testing.T) {
	nodeTree := New()
	nodeTree.AddData("foo.txt", []byte("foo"))

	tmpfile := core.Path(t.TempDir(), "test.txt")
	err := nodeTree.CopyFile("foo.txt", tmpfile, 0644)
	require.NoError(t, err)

	content, err := coreio.Local.Read(tmpfile)
	require.NoError(t, err)
	assert.Equal(t, "foo", content)
}

func TestNode_CopyFile_Bad(t *testing.T) {
	nodeTree := New()
	tmpfile := core.Path(t.TempDir(), "test.txt")

	err := nodeTree.CopyFile("nonexistent.txt", tmpfile, 0644)
	assert.Error(t, err)

	nodeTree.AddData("foo.txt", []byte("foo"))
	err = nodeTree.CopyFile("foo.txt", "/nonexistent_dir/test.txt", 0644)
	assert.Error(t, err)
}

func TestNode_CopyFile_DirectorySource_Bad(t *testing.T) {
	nodeTree := New()
	nodeTree.AddData("bar/baz.txt", []byte("baz"))
	tmpfile := core.Path(t.TempDir(), "test.txt")

	err := nodeTree.CopyFile("bar", tmpfile, 0644)
	assert.Error(t, err)
}

func TestNode_CopyTo_Good(t *testing.T) {
	nodeTree := New()
	nodeTree.AddData("config/app.yaml", []byte("port: 8080"))
	nodeTree.AddData("config/env/app.env", []byte("MODE=test"))

	fileTarget := coreio.NewMemoryMedium()
	err := nodeTree.CopyTo(fileTarget, "config/app.yaml", "backup/app.yaml")
	require.NoError(t, err)
	content, err := fileTarget.Read("backup/app.yaml")
	require.NoError(t, err)
	assert.Equal(t, "port: 8080", content)

	dirTarget := coreio.NewMemoryMedium()
	err = nodeTree.CopyTo(dirTarget, "config", "backup/config")
	require.NoError(t, err)
	content, err = dirTarget.Read("backup/config/app.yaml")
	require.NoError(t, err)
	assert.Equal(t, "port: 8080", content)
	content, err = dirTarget.Read("backup/config/env/app.env")
	require.NoError(t, err)
	assert.Equal(t, "MODE=test", content)
}

func TestNode_CopyTo_Bad(t *testing.T) {
	nodeTree := New()
	err := nodeTree.CopyTo(coreio.NewMemoryMedium(), "missing", "backup/missing")
	assert.Error(t, err)
}

func TestNode_MediumFacade_Good(t *testing.T) {
	nodeTree := New()

	require.NoError(t, nodeTree.Write("docs/readme.txt", "hello"))
	require.NoError(t, nodeTree.WriteMode("docs/mode.txt", "mode", 0600))
	require.NoError(t, nodeTree.Write("docs/guide.txt", "guide"))
	require.NoError(t, nodeTree.EnsureDir("ignored"))

	value, err := nodeTree.Read("docs/readme.txt")
	require.NoError(t, err)
	assert.Equal(t, "hello", value)

	value, err = nodeTree.Read("docs/guide.txt")
	require.NoError(t, err)
	assert.Equal(t, "guide", value)

	assert.True(t, nodeTree.IsFile("docs/readme.txt"))
	assert.True(t, nodeTree.IsDir("docs"))

	entries, err := nodeTree.List("docs")
	require.NoError(t, err)
	assert.Equal(t, []string{"guide.txt", "mode.txt", "readme.txt"}, sortedNames(entries))

	file, err := nodeTree.Open("docs/readme.txt")
	require.NoError(t, err)
	info, err := file.Stat()
	require.NoError(t, err)
	assert.Equal(t, "readme.txt", info.Name())
	assert.Equal(t, fs.FileMode(0444), info.Mode())
	assert.False(t, info.IsDir())
	assert.Nil(t, info.Sys())
	require.NoError(t, file.Close())

	dir, err := nodeTree.Open("docs")
	require.NoError(t, err)
	dirInfo, err := dir.Stat()
	require.NoError(t, err)
	assert.Equal(t, "docs", dirInfo.Name())
	assert.True(t, dirInfo.IsDir())
	assert.Equal(t, fs.ModeDir|0555, dirInfo.Mode())
	assert.Nil(t, dirInfo.Sys())
	require.NoError(t, dir.Close())

	createWriter, err := nodeTree.Create("docs/generated.txt")
	require.NoError(t, err)
	_, err = createWriter.Write([]byte("generated"))
	require.NoError(t, err)
	require.NoError(t, createWriter.Close())

	appendWriter, err := nodeTree.Append("docs/generated.txt")
	require.NoError(t, err)
	_, err = appendWriter.Write([]byte(" content"))
	require.NoError(t, err)
	require.NoError(t, appendWriter.Close())

	streamReader, err := nodeTree.ReadStream("docs/generated.txt")
	require.NoError(t, err)
	streamData, err := io.ReadAll(streamReader)
	require.NoError(t, err)
	assert.Equal(t, "generated content", string(streamData))
	require.NoError(t, streamReader.Close())

	writeStream, err := nodeTree.WriteStream("docs/stream.txt")
	require.NoError(t, err)
	_, err = writeStream.Write([]byte("stream"))
	require.NoError(t, err)
	require.NoError(t, writeStream.Close())

	require.NoError(t, nodeTree.Rename("docs/stream.txt", "docs/stream-renamed.txt"))
	assert.True(t, nodeTree.Exists("docs/stream-renamed.txt"))

	require.NoError(t, nodeTree.Delete("docs/stream-renamed.txt"))
	assert.False(t, nodeTree.Exists("docs/stream-renamed.txt"))

	require.NoError(t, nodeTree.DeleteAll("docs"))
	assert.False(t, nodeTree.Exists("docs"))
}

func TestNode_ToTar_Good(t *testing.T) {
	nodeTree := New()
	nodeTree.AddData("foo.txt", []byte("foo"))
	nodeTree.AddData("bar/baz.txt", []byte("baz"))

	tarball, err := nodeTree.ToTar()
	require.NoError(t, err)
	require.NotEmpty(t, tarball)

	tr := tar.NewReader(bytes.NewReader(tarball))
	files := make(map[string]string)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		content, err := io.ReadAll(tr)
		require.NoError(t, err)
		files[header.Name] = string(content)
	}

	assert.Equal(t, "foo", files["foo.txt"])
	assert.Equal(t, "baz", files["bar/baz.txt"])
}

func TestNode_FromTar_Good(t *testing.T) {
	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)

	for _, f := range []struct{ Name, Body string }{
		{"foo.txt", "foo"},
		{"bar/baz.txt", "baz"},
	} {
		hdr := &tar.Header{
			Name:     f.Name,
			Mode:     0600,
			Size:     int64(len(f.Body)),
			Typeflag: tar.TypeReg,
		}
		require.NoError(t, tw.WriteHeader(hdr))
		_, err := tw.Write([]byte(f.Body))
		require.NoError(t, err)
	}
	require.NoError(t, tw.Close())

	nodeTree, err := FromTar(buf.Bytes())
	require.NoError(t, err)

	assert.True(t, nodeTree.Exists("foo.txt"), "foo.txt should exist")
	assert.True(t, nodeTree.Exists("bar/baz.txt"), "bar/baz.txt should exist")
}

func TestNode_FromTar_Bad(t *testing.T) {
	truncated := make([]byte, 100)
	_, err := FromTar(truncated)
	assert.Error(t, err, "truncated data should produce an error")
}

func TestNode_TarRoundTrip_Good(t *testing.T) {
	nodeTree1 := New()
	nodeTree1.AddData("a.txt", []byte("alpha"))
	nodeTree1.AddData("b/c.txt", []byte("charlie"))

	tarball, err := nodeTree1.ToTar()
	require.NoError(t, err)

	nodeTree2, err := FromTar(tarball)
	require.NoError(t, err)

	data, err := nodeTree2.ReadFile("a.txt")
	require.NoError(t, err)
	assert.Equal(t, []byte("alpha"), data)

	data, err = nodeTree2.ReadFile("b/c.txt")
	require.NoError(t, err)
	assert.Equal(t, []byte("charlie"), data)
}

func TestNode_FSInterface_Good(t *testing.T) {
	nodeTree := New()
	nodeTree.AddData("hello.txt", []byte("world"))

	var fsys fs.FS = nodeTree
	file, err := fsys.Open("hello.txt")
	require.NoError(t, err)
	defer file.Close()

	var statFS fs.StatFS = nodeTree
	info, err := statFS.Stat("hello.txt")
	require.NoError(t, err)
	assert.Equal(t, "hello.txt", info.Name())
	assert.Equal(t, int64(5), info.Size())

	var readFS fs.ReadFileFS = nodeTree
	data, err := readFS.ReadFile("hello.txt")
	require.NoError(t, err)
	assert.Equal(t, []byte("world"), data)
}

func sortedNames(entries []fs.DirEntry) []string {
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	sort.Strings(names)
	return names
}
