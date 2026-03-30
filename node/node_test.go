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

// ---------------------------------------------------------------------------
// New
// ---------------------------------------------------------------------------

func TestNode_New_Good(t *testing.T) {
	n := New()
	require.NotNil(t, n, "New() must not return nil")
	assert.NotNil(t, n.files, "New() must initialise the files map")
}

// ---------------------------------------------------------------------------
// AddData
// ---------------------------------------------------------------------------

func TestNode_AddData_Good(t *testing.T) {
	n := New()
	n.AddData("foo.txt", []byte("foo"))

	file, ok := n.files["foo.txt"]
	require.True(t, ok, "file foo.txt should be present")
	assert.Equal(t, []byte("foo"), file.content)

	info, err := file.Stat()
	require.NoError(t, err)
	assert.Equal(t, "foo.txt", info.Name())
}

func TestNode_AddData_Bad(t *testing.T) {
	n := New()

	// Empty name is silently ignored.
	n.AddData("", []byte("data"))
	assert.Empty(t, n.files, "empty name must not be stored")

	// Directory entry (trailing slash) is silently ignored.
	n.AddData("dir/", nil)
	assert.Empty(t, n.files, "directory entry must not be stored")
}

func TestNode_AddData_Ugly(t *testing.T) {
	t.Run("Overwrite", func(t *testing.T) {
		n := New()
		n.AddData("foo.txt", []byte("foo"))
		n.AddData("foo.txt", []byte("bar"))

		file := n.files["foo.txt"]
		assert.Equal(t, []byte("bar"), file.content, "second AddData should overwrite")
	})

	t.Run("LeadingSlash", func(t *testing.T) {
		n := New()
		n.AddData("/hello.txt", []byte("hi"))
		_, ok := n.files["hello.txt"]
		assert.True(t, ok, "leading slash should be trimmed")
	})
}

// ---------------------------------------------------------------------------
// Open
// ---------------------------------------------------------------------------

func TestNode_Open_Good(t *testing.T) {
	n := New()
	n.AddData("foo.txt", []byte("foo"))

	file, err := n.Open("foo.txt")
	require.NoError(t, err)
	defer file.Close()

	buf := make([]byte, 10)
	nr, err := file.Read(buf)
	require.True(t, nr > 0 || err == io.EOF)
	assert.Equal(t, "foo", string(buf[:nr]))
}

func TestNode_Open_Bad(t *testing.T) {
	n := New()
	_, err := n.Open("nonexistent.txt")
	require.Error(t, err)
	assert.ErrorIs(t, err, fs.ErrNotExist)
}

func TestNode_Open_Ugly(t *testing.T) {
	n := New()
	n.AddData("bar/baz.txt", []byte("baz"))

	// Opening a directory should succeed.
	file, err := n.Open("bar")
	require.NoError(t, err)
	defer file.Close()

	// Reading from a directory should fail.
	_, err = file.Read(make([]byte, 1))
	require.Error(t, err)

	var pathErr *fs.PathError
	require.True(t, core.As(err, &pathErr))
	assert.Equal(t, fs.ErrInvalid, pathErr.Err)
}

// ---------------------------------------------------------------------------
// Stat
// ---------------------------------------------------------------------------

func TestNode_Stat_Good(t *testing.T) {
	n := New()
	n.AddData("foo.txt", []byte("foo"))
	n.AddData("bar/baz.txt", []byte("baz"))

	// File stat.
	info, err := n.Stat("bar/baz.txt")
	require.NoError(t, err)
	assert.Equal(t, "baz.txt", info.Name())
	assert.Equal(t, int64(3), info.Size())
	assert.False(t, info.IsDir())

	// Directory stat.
	dirInfo, err := n.Stat("bar")
	require.NoError(t, err)
	assert.True(t, dirInfo.IsDir())
	assert.Equal(t, "bar", dirInfo.Name())
}

func TestNode_Stat_Bad(t *testing.T) {
	n := New()
	_, err := n.Stat("nonexistent")
	require.Error(t, err)
	assert.ErrorIs(t, err, fs.ErrNotExist)
}

func TestNode_Stat_Ugly(t *testing.T) {
	n := New()
	n.AddData("foo.txt", []byte("foo"))

	// Root directory.
	info, err := n.Stat(".")
	require.NoError(t, err)
	assert.True(t, info.IsDir())
	assert.Equal(t, ".", info.Name())
}

// ---------------------------------------------------------------------------
// ReadFile
// ---------------------------------------------------------------------------

func TestNode_ReadFile_Good(t *testing.T) {
	n := New()
	n.AddData("hello.txt", []byte("hello world"))

	data, err := n.ReadFile("hello.txt")
	require.NoError(t, err)
	assert.Equal(t, []byte("hello world"), data)
}

func TestNode_ReadFile_Bad(t *testing.T) {
	n := New()
	_, err := n.ReadFile("missing.txt")
	require.Error(t, err)
	assert.ErrorIs(t, err, fs.ErrNotExist)
}

func TestNode_ReadFile_Ugly(t *testing.T) {
	n := New()
	n.AddData("data.bin", []byte("original"))

	// Returned slice must be a copy — mutating it must not affect internal state.
	data, err := n.ReadFile("data.bin")
	require.NoError(t, err)
	data[0] = 'X'

	data2, err := n.ReadFile("data.bin")
	require.NoError(t, err)
	assert.Equal(t, []byte("original"), data2, "ReadFile must return an independent copy")
}

// ---------------------------------------------------------------------------
// ReadDir
// ---------------------------------------------------------------------------

func TestNode_ReadDir_Good(t *testing.T) {
	n := New()
	n.AddData("foo.txt", []byte("foo"))
	n.AddData("bar/baz.txt", []byte("baz"))
	n.AddData("bar/qux.txt", []byte("qux"))

	// Root.
	entries, err := n.ReadDir(".")
	require.NoError(t, err)
	assert.Equal(t, []string{"bar", "foo.txt"}, sortedNames(entries))

	// Subdirectory.
	barEntries, err := n.ReadDir("bar")
	require.NoError(t, err)
	assert.Equal(t, []string{"baz.txt", "qux.txt"}, sortedNames(barEntries))
}

func TestNode_ReadDir_Bad(t *testing.T) {
	n := New()
	n.AddData("foo.txt", []byte("foo"))

	// Reading a file as a directory should fail.
	_, err := n.ReadDir("foo.txt")
	require.Error(t, err)
	var pathErr *fs.PathError
	require.True(t, core.As(err, &pathErr))
	assert.Equal(t, fs.ErrInvalid, pathErr.Err)
}

func TestNode_ReadDir_Ugly(t *testing.T) {
	n := New()
	n.AddData("bar/baz.txt", []byte("baz"))
	n.AddData("empty_dir/", nil) // Ignored by AddData.

	entries, err := n.ReadDir(".")
	require.NoError(t, err)
	assert.Equal(t, []string{"bar"}, sortedNames(entries))
}

// ---------------------------------------------------------------------------
// Exists
// ---------------------------------------------------------------------------

func TestNode_Exists_Good(t *testing.T) {
	n := New()
	n.AddData("foo.txt", []byte("foo"))
	n.AddData("bar/baz.txt", []byte("baz"))

	assert.True(t, n.Exists("foo.txt"))
	assert.True(t, n.Exists("bar"))
}

func TestNode_Exists_Bad(t *testing.T) {
	n := New()
	assert.False(t, n.Exists("nonexistent"))
}

func TestNode_Exists_Ugly(t *testing.T) {
	n := New()
	n.AddData("dummy.txt", []byte("dummy"))

	assert.True(t, n.Exists("."), "root '.' must exist")
	assert.True(t, n.Exists(""), "empty path (root) must exist")
}

// ---------------------------------------------------------------------------
// Walk
// ---------------------------------------------------------------------------

func TestNode_Walk_Good(t *testing.T) {
	n := New()
	n.AddData("foo.txt", []byte("foo"))
	n.AddData("bar/baz.txt", []byte("baz"))
	n.AddData("bar/qux.txt", []byte("qux"))

	var paths []string
	err := n.Walk(".", func(p string, d fs.DirEntry, err error) error {
		paths = append(paths, p)
		return nil
	})
	require.NoError(t, err)

	sort.Strings(paths)
	assert.Equal(t, []string{".", "bar", "bar/baz.txt", "bar/qux.txt", "foo.txt"}, paths)
}

func TestNode_Walk_Bad(t *testing.T) {
	n := New()

	var called bool
	err := n.Walk("nonexistent", func(p string, d fs.DirEntry, err error) error {
		called = true
		assert.Error(t, err)
		assert.ErrorIs(t, err, fs.ErrNotExist)
		return err
	})
	assert.True(t, called, "walk function must be called for nonexistent root")
	assert.ErrorIs(t, err, fs.ErrNotExist)
}

func TestNode_Walk_Ugly(t *testing.T) {
	n := New()
	n.AddData("a/b.txt", []byte("b"))
	n.AddData("a/c.txt", []byte("c"))

	// Stop walk early with a custom error.
	walkErr := core.NewError("stop walking")
	var paths []string
	err := n.Walk(".", func(p string, d fs.DirEntry, err error) error {
		if p == "a/b.txt" {
			return walkErr
		}
		paths = append(paths, p)
		return nil
	})

	assert.Equal(t, walkErr, err, "Walk must propagate the callback error")
}

func TestNode_WalkWithOptions_Good(t *testing.T) {
	n := New()
	n.AddData("root.txt", []byte("root"))
	n.AddData("a/a1.txt", []byte("a1"))
	n.AddData("a/b/b1.txt", []byte("b1"))
	n.AddData("c/c1.txt", []byte("c1"))

	t.Run("MaxDepth", func(t *testing.T) {
		var paths []string
		err := n.WalkWithOptions(".", func(p string, d fs.DirEntry, err error) error {
			paths = append(paths, p)
			return nil
		}, WalkOptions{MaxDepth: 1})
		require.NoError(t, err)

		sort.Strings(paths)
		assert.Equal(t, []string{".", "a", "c", "root.txt"}, paths)
	})

	t.Run("Filter", func(t *testing.T) {
		var paths []string
		err := n.WalkWithOptions(".", func(p string, d fs.DirEntry, err error) error {
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
		err := n.WalkWithOptions("nonexistent", func(p string, d fs.DirEntry, err error) error {
			called = true
			return err
		}, WalkOptions{SkipErrors: true})

		assert.NoError(t, err, "SkipErrors should suppress the error")
		assert.False(t, called, "callback should not be called when error is skipped")
	})
}

func TestNode_WalkNode_Good(t *testing.T) {
	n := New()
	n.AddData("alpha.txt", []byte("alpha"))
	n.AddData("nested/beta.txt", []byte("beta"))

	var paths []string
	err := n.WalkNode(".", func(p string, d fs.DirEntry, err error) error {
		require.NoError(t, err)
		paths = append(paths, p)
		return nil
	})
	require.NoError(t, err)

	sort.Strings(paths)
	assert.Equal(t, []string{".", "alpha.txt", "nested", "nested/beta.txt"}, paths)
}

// ---------------------------------------------------------------------------
// CopyFile
// ---------------------------------------------------------------------------

func TestNode_CopyFile_Good(t *testing.T) {
	n := New()
	n.AddData("foo.txt", []byte("foo"))

	tmpfile := core.Path(t.TempDir(), "test.txt")
	err := n.CopyFile("foo.txt", tmpfile, 0644)
	require.NoError(t, err)

	content, err := coreio.Local.Read(tmpfile)
	require.NoError(t, err)
	assert.Equal(t, "foo", content)
}

func TestNode_CopyFile_Bad(t *testing.T) {
	n := New()
	tmpfile := core.Path(t.TempDir(), "test.txt")

	// Source does not exist.
	err := n.CopyFile("nonexistent.txt", tmpfile, 0644)
	assert.Error(t, err)

	// Destination not writable.
	n.AddData("foo.txt", []byte("foo"))
	err = n.CopyFile("foo.txt", "/nonexistent_dir/test.txt", 0644)
	assert.Error(t, err)
}

func TestNode_CopyFile_Ugly(t *testing.T) {
	n := New()
	n.AddData("bar/baz.txt", []byte("baz"))
	tmpfile := core.Path(t.TempDir(), "test.txt")

	// Attempting to copy a directory should fail.
	err := n.CopyFile("bar", tmpfile, 0644)
	assert.Error(t, err)
}

func TestNode_CopyTo_Good(t *testing.T) {
	n := New()
	n.AddData("config/app.yaml", []byte("port: 8080"))
	n.AddData("config/env/app.env", []byte("MODE=test"))

	fileTarget := coreio.NewMockMedium()
	err := n.CopyTo(fileTarget, "config/app.yaml", "backup/app.yaml")
	require.NoError(t, err)
	assert.Equal(t, "port: 8080", fileTarget.Files["backup/app.yaml"])

	dirTarget := coreio.NewMockMedium()
	err = n.CopyTo(dirTarget, "config", "backup/config")
	require.NoError(t, err)
	assert.Equal(t, "port: 8080", dirTarget.Files["backup/config/app.yaml"])
	assert.Equal(t, "MODE=test", dirTarget.Files["backup/config/env/app.env"])
}

func TestNode_CopyTo_Bad(t *testing.T) {
	n := New()
	err := n.CopyTo(coreio.NewMockMedium(), "missing", "backup/missing")
	assert.Error(t, err)
}

func TestNode_MediumFacade_Good(t *testing.T) {
	n := New()

	require.NoError(t, n.Write("docs/readme.txt", "hello"))
	require.NoError(t, n.WriteMode("docs/mode.txt", "mode", 0600))
	require.NoError(t, n.FileSet("docs/guide.txt", "guide"))
	require.NoError(t, n.EnsureDir("ignored"))

	value, err := n.Read("docs/readme.txt")
	require.NoError(t, err)
	assert.Equal(t, "hello", value)

	value, err = n.FileGet("docs/guide.txt")
	require.NoError(t, err)
	assert.Equal(t, "guide", value)

	assert.True(t, n.IsFile("docs/readme.txt"))
	assert.True(t, n.IsDir("docs"))

	entries, err := n.List("docs")
	require.NoError(t, err)
	assert.Equal(t, []string{"guide.txt", "mode.txt", "readme.txt"}, sortedNames(entries))

	file, err := n.Open("docs/readme.txt")
	require.NoError(t, err)
	info, err := file.Stat()
	require.NoError(t, err)
	assert.Equal(t, "readme.txt", info.Name())
	assert.Equal(t, fs.FileMode(0444), info.Mode())
	assert.False(t, info.IsDir())
	assert.Nil(t, info.Sys())
	require.NoError(t, file.Close())

	dir, err := n.Open("docs")
	require.NoError(t, err)
	dirInfo, err := dir.Stat()
	require.NoError(t, err)
	assert.Equal(t, "docs", dirInfo.Name())
	assert.True(t, dirInfo.IsDir())
	assert.Equal(t, fs.ModeDir|0555, dirInfo.Mode())
	assert.Nil(t, dirInfo.Sys())
	require.NoError(t, dir.Close())

	createWriter, err := n.Create("docs/generated.txt")
	require.NoError(t, err)
	_, err = createWriter.Write([]byte("generated"))
	require.NoError(t, err)
	require.NoError(t, createWriter.Close())

	appendWriter, err := n.Append("docs/generated.txt")
	require.NoError(t, err)
	_, err = appendWriter.Write([]byte(" content"))
	require.NoError(t, err)
	require.NoError(t, appendWriter.Close())

	streamReader, err := n.ReadStream("docs/generated.txt")
	require.NoError(t, err)
	streamData, err := io.ReadAll(streamReader)
	require.NoError(t, err)
	assert.Equal(t, "generated content", string(streamData))
	require.NoError(t, streamReader.Close())

	writeStream, err := n.WriteStream("docs/stream.txt")
	require.NoError(t, err)
	_, err = writeStream.Write([]byte("stream"))
	require.NoError(t, err)
	require.NoError(t, writeStream.Close())

	require.NoError(t, n.Rename("docs/stream.txt", "docs/stream-renamed.txt"))
	assert.True(t, n.Exists("docs/stream-renamed.txt"))

	require.NoError(t, n.Delete("docs/stream-renamed.txt"))
	assert.False(t, n.Exists("docs/stream-renamed.txt"))

	require.NoError(t, n.DeleteAll("docs"))
	assert.False(t, n.Exists("docs"))
}

// ---------------------------------------------------------------------------
// ToTar / FromTar
// ---------------------------------------------------------------------------

func TestNode_ToTar_Good(t *testing.T) {
	n := New()
	n.AddData("foo.txt", []byte("foo"))
	n.AddData("bar/baz.txt", []byte("baz"))

	tarball, err := n.ToTar()
	require.NoError(t, err)
	require.NotEmpty(t, tarball)

	// Verify tar content.
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

	n, err := FromTar(buf.Bytes())
	require.NoError(t, err)

	assert.True(t, n.Exists("foo.txt"), "foo.txt should exist")
	assert.True(t, n.Exists("bar/baz.txt"), "bar/baz.txt should exist")
}

func TestNode_FromTar_Bad(t *testing.T) {
	// Truncated data that cannot be a valid tar.
	truncated := make([]byte, 100)
	_, err := FromTar(truncated)
	assert.Error(t, err, "truncated data should produce an error")
}

func TestNode_TarRoundTrip_Good(t *testing.T) {
	n1 := New()
	n1.AddData("a.txt", []byte("alpha"))
	n1.AddData("b/c.txt", []byte("charlie"))

	tarball, err := n1.ToTar()
	require.NoError(t, err)

	n2, err := FromTar(tarball)
	require.NoError(t, err)

	// Verify n2 matches n1.
	data, err := n2.ReadFile("a.txt")
	require.NoError(t, err)
	assert.Equal(t, []byte("alpha"), data)

	data, err = n2.ReadFile("b/c.txt")
	require.NoError(t, err)
	assert.Equal(t, []byte("charlie"), data)
}

// ---------------------------------------------------------------------------
// fs.FS interface compliance
// ---------------------------------------------------------------------------

func TestNode_FSInterface_Good(t *testing.T) {
	n := New()
	n.AddData("hello.txt", []byte("world"))

	// fs.FS
	var fsys fs.FS = n
	file, err := fsys.Open("hello.txt")
	require.NoError(t, err)
	defer file.Close()

	// fs.StatFS
	var statFS fs.StatFS = n
	info, err := statFS.Stat("hello.txt")
	require.NoError(t, err)
	assert.Equal(t, "hello.txt", info.Name())
	assert.Equal(t, int64(5), info.Size())

	// fs.ReadFileFS
	var readFS fs.ReadFileFS = n
	data, err := readFS.ReadFile("hello.txt")
	require.NoError(t, err)
	assert.Equal(t, []byte("world"), data)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func sortedNames(entries []fs.DirEntry) []string {
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	sort.Strings(names)
	return names
}
