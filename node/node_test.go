package node

import (
	"archive/tar"
	"bytes"
	"io"
	"io/fs"
	"sort"

	core "dappco.re/go"
	coreio "dappco.re/go/io"
)

func TestNode_New_Good(t *core.T) {
	nodeTree := New()
	core.AssertNotNil(t, nodeTree, "New() must not return nil")
	core.AssertNotNil(t, nodeTree.files, "New() must initialise the files map")
}

func TestNode_AddData_Good(t *core.T) {
	nodeTree := New()
	nodeTree.AddData("foo.txt", []byte("foo"))

	file, ok := nodeTree.files["foo.txt"]
	core.RequireTrue(t, ok, "file foo.txt should be present")
	core.AssertEqual(t, []byte("foo"), file.content)

	info, err := file.Stat()
	core.RequireNoError(t, err)
	core.AssertEqual(t, "foo.txt", info.Name())
}

func TestNode_AddData_Bad(t *core.T) {
	nodeTree := New()

	nodeTree.AddData("", []byte("data"))
	core.AssertEmpty(t, nodeTree.files, "empty name must not be stored")

	nodeTree.AddData("dir/", nil)
	core.AssertEmpty(t, nodeTree.files, "directory entry must not be stored")
}

func TestNode_AddData_EdgeCases_Good(t *core.T) {
	t.Run("Overwrite", func(t *core.T) {
		nodeTree := New()
		nodeTree.AddData("foo.txt", []byte("foo"))
		nodeTree.AddData("foo.txt", []byte("bar"))

		file := nodeTree.files["foo.txt"]
		core.AssertEqual(t, []byte("bar"), file.content, "second AddData should overwrite")
	})

	t.Run("LeadingSlash", func(t *core.T) {
		nodeTree := New()
		nodeTree.AddData("/hello.txt", []byte("hi"))
		_, ok := nodeTree.files["hello.txt"]
		core.AssertTrue(t, ok, "leading slash should be trimmed")
	})
}

func TestNode_Open_Good(t *core.T) {
	nodeTree := New()
	nodeTree.AddData("foo.txt", []byte("foo"))

	file, err := nodeTree.Open("foo.txt")
	core.RequireNoError(t, err)
	defer file.Close()

	readBuffer := make([]byte, 10)
	nr, err := file.Read(readBuffer)
	core.RequireTrue(t, nr > 0 || err == io.EOF)
	core.AssertEqual(t, "foo", string(readBuffer[:nr]))
}

func TestNode_Open_Bad(t *core.T) {
	nodeTree := New()
	_, err := nodeTree.Open("nonexistent.txt")
	core.AssertError(t, err)
	core.AssertErrorIs(t, err, fs.ErrNotExist)
}

func TestNode_Open_Directory_Good(t *core.T) {
	nodeTree := New()
	nodeTree.AddData("bar/baz.txt", []byte("baz"))

	file, err := nodeTree.Open("bar")
	core.RequireNoError(t, err)
	defer file.Close()

	_, err = file.Read(make([]byte, 1))
	core.AssertError(t, err)

	var pathError *fs.PathError
	core.RequireTrue(t, core.As(err, &pathError))
	core.AssertEqual(t, fs.ErrInvalid, pathError.Err)
}

func TestNode_Stat_Good(t *core.T) {
	nodeTree := New()
	nodeTree.AddData("foo.txt", []byte("foo"))
	nodeTree.AddData("bar/baz.txt", []byte("baz"))

	info, err := nodeTree.Stat("bar/baz.txt")
	core.RequireNoError(t, err)
	core.AssertEqual(t, "baz.txt", info.Name())
	core.AssertEqual(t, int64(3), info.Size())
	core.AssertFalse(t, info.IsDir())

	dirInfo, err := nodeTree.Stat("bar")
	core.RequireNoError(t, err)
	core.AssertTrue(t, dirInfo.IsDir())
	core.AssertEqual(t, "bar", dirInfo.Name())
}

func TestNode_Stat_Bad(t *core.T) {
	nodeTree := New()
	_, err := nodeTree.Stat("nonexistent")
	core.AssertError(t, err)
	core.AssertErrorIs(t, err, fs.ErrNotExist)
}

func TestNode_Stat_RootDirectory_Good(t *core.T) {
	nodeTree := New()
	nodeTree.AddData("foo.txt", []byte("foo"))

	info, err := nodeTree.Stat(".")
	core.RequireNoError(t, err)
	core.AssertTrue(t, info.IsDir())
	core.AssertEqual(t, ".", info.Name())
}

func TestNode_ReadFile_Good(t *core.T) {
	nodeTree := New()
	nodeTree.AddData("hello.txt", []byte("hello world"))

	data, err := nodeTree.ReadFile("hello.txt")
	core.RequireNoError(t, err)
	core.AssertEqual(t, []byte("hello world"), data)
}

func TestNode_ReadFile_Bad(t *core.T) {
	nodeTree := New()
	_, err := nodeTree.ReadFile("missing.txt")
	core.AssertError(t, err)
	core.AssertErrorIs(t, err, fs.ErrNotExist)
}

func TestNode_ReadFile_ReturnsCopy_Good(t *core.T) {
	nodeTree := New()
	nodeTree.AddData("data.bin", []byte("original"))

	data, err := nodeTree.ReadFile("data.bin")
	core.RequireNoError(t, err)
	data[0] = 'X'

	data2, err := nodeTree.ReadFile("data.bin")
	core.RequireNoError(t, err)
	core.AssertEqual(t, []byte("original"), data2, "ReadFile must return an independent copy")
}

func TestNode_ReadDir_Good(t *core.T) {
	nodeTree := New()
	nodeTree.AddData("foo.txt", []byte("foo"))
	nodeTree.AddData("bar/baz.txt", []byte("baz"))
	nodeTree.AddData("bar/qux.txt", []byte("qux"))

	entries, err := nodeTree.ReadDir(".")
	core.RequireNoError(t, err)
	core.AssertEqual(t, []string{"bar", "foo.txt"}, sortedNames(entries))

	barEntries, err := nodeTree.ReadDir("bar")
	core.RequireNoError(t, err)
	core.AssertEqual(t, []string{"baz.txt", "qux.txt"}, sortedNames(barEntries))
}

func TestNode_ReadDir_Bad(t *core.T) {
	nodeTree := New()
	nodeTree.AddData("foo.txt", []byte("foo"))

	_, err := nodeTree.ReadDir("foo.txt")
	core.AssertError(t, err)
	var pathError *fs.PathError
	core.RequireTrue(t, core.As(err, &pathError))
	core.AssertEqual(t, fs.ErrInvalid, pathError.Err)
}

func TestNode_ReadDir_IgnoresEmptyEntry_Good(t *core.T) {
	nodeTree := New()
	nodeTree.AddData("bar/baz.txt", []byte("baz"))
	nodeTree.AddData("empty_dir/", nil)

	entries, err := nodeTree.ReadDir(".")
	core.RequireNoError(t, err)
	core.AssertEqual(t, []string{"bar"}, sortedNames(entries))
}

func TestNode_Exists_Good(t *core.T) {
	nodeTree := New()
	nodeTree.AddData("foo.txt", []byte("foo"))
	nodeTree.AddData("bar/baz.txt", []byte("baz"))

	core.AssertTrue(t, nodeTree.Exists("foo.txt"))
	core.AssertTrue(t, nodeTree.Exists("bar"))
}

func TestNode_Exists_Bad(t *core.T) {
	nodeTree := New()
	nodeTree.AddData("present.txt", []byte("data"))
	core.AssertFalse(t, nodeTree.Exists("nonexistent"))
	core.AssertFalse(t, nodeTree.Exists("present.txt/missing"))
	core.AssertTrue(t, nodeTree.Exists("present.txt"))
}

func TestNode_Exists_RootAndEmptyPath_Good(t *core.T) {
	nodeTree := New()
	nodeTree.AddData("dummy.txt", []byte("dummy"))

	core.AssertTrue(t, nodeTree.Exists("."), "root '.' must exist")
	core.AssertTrue(t, nodeTree.Exists(""), "empty path (root) must exist")
}

func TestNode_Walk_Default_Good(t *core.T) {
	nodeTree := New()
	nodeTree.AddData("foo.txt", []byte("foo"))
	nodeTree.AddData("bar/baz.txt", []byte("baz"))
	nodeTree.AddData("bar/qux.txt", []byte("qux"))

	var paths []string
	err := nodeTree.Walk(".", func(p string, d fs.DirEntry, err error) error {
		paths = append(paths, p)
		return nil
	}, WalkOptions{})
	core.RequireNoError(t, err)

	sort.Strings(paths)
	core.AssertEqual(t, []string{".", "bar", "bar/baz.txt", "bar/qux.txt", "foo.txt"}, paths)
}

func TestNode_Walk_Default_Bad(t *core.T) {
	nodeTree := New()

	var called bool
	err := nodeTree.Walk("nonexistent", func(p string, d fs.DirEntry, err error) error {
		called = true
		core.AssertError(t, err)
		core.AssertErrorIs(t, err, fs.ErrNotExist)
		return err
	}, WalkOptions{})
	core.AssertTrue(t, called, "walk function must be called for nonexistent root")
	core.AssertErrorIs(t, err, fs.ErrNotExist)
}

func TestNode_Walk_CallbackError_Good(t *core.T) {
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

	core.AssertEqual(t, walkErr, err, "Walk must propagate the callback error")
}

func TestNode_Walk_Good(t *core.T) {
	nodeTree := New()
	nodeTree.AddData("root.txt", []byte("root"))
	nodeTree.AddData("a/a1.txt", []byte("a1"))
	nodeTree.AddData("a/b/b1.txt", []byte("b1"))
	nodeTree.AddData("c/c1.txt", []byte("c1"))

	t.Run("MaxDepth", func(t *core.T) {
		var paths []string
		err := nodeTree.Walk(".", func(p string, d fs.DirEntry, err error) error {
			paths = append(paths, p)
			return nil
		}, WalkOptions{MaxDepth: 1})
		core.RequireNoError(t, err)

		sort.Strings(paths)
		core.AssertEqual(t, []string{".", "a", "c", "root.txt"}, paths)
	})

	t.Run("Filter", func(t *core.T) {
		var paths []string
		err := nodeTree.Walk(".", func(p string, d fs.DirEntry, err error) error {
			paths = append(paths, p)
			return nil
		}, WalkOptions{Filter: func(p string, d fs.DirEntry) bool {
			return !core.HasPrefix(p, "a")
		}})
		core.RequireNoError(t, err)

		sort.Strings(paths)
		core.AssertEqual(t, []string{".", "c", "c/c1.txt", "root.txt"}, paths)
	})

	t.Run("SkipErrors", func(t *core.T) {
		var called bool
		err := nodeTree.Walk("nonexistent", func(p string, d fs.DirEntry, err error) error {
			called = true
			return err
		}, WalkOptions{SkipErrors: true})

		core.AssertNoError(t, err, "SkipErrors should suppress the error")
		core.AssertFalse(t, called, "callback should not be called when error is skipped")
	})
}

func TestNode_ExportFile_Good(t *core.T) {
	nodeTree := New()
	nodeTree.AddData("foo.txt", []byte("foo"))

	destinationPath := core.Path(t.TempDir(), "test.txt")
	err := nodeTree.ExportFile("foo.txt", destinationPath, 0644)
	core.RequireNoError(t, err)

	content, err := coreio.Local.Read(destinationPath)
	core.RequireNoError(t, err)
	core.AssertEqual(t, "foo", content)
}

func TestNode_ExportFile_Bad(t *core.T) {
	nodeTree := New()
	destinationPath := core.Path(t.TempDir(), "test.txt")

	err := nodeTree.ExportFile("nonexistent.txt", destinationPath, 0644)
	core.AssertError(t, err)

	nodeTree.AddData("foo.txt", []byte("foo"))
	nonExistentParent := core.Path(t.TempDir(), "nonexistent_subdir", "test.txt")
	err = nodeTree.ExportFile("foo.txt", nonExistentParent, 0644)
	core.AssertError(t, err)
}

func TestNode_ExportFile_DirectorySource_Bad(t *core.T) {
	nodeTree := New()
	nodeTree.AddData("bar/baz.txt", []byte("baz"))
	destinationPath := core.Path(t.TempDir(), "test.txt")

	err := nodeTree.ExportFile("bar", destinationPath, 0644)
	core.AssertError(t, err)
}

func TestNode_CopyTo_Good(t *core.T) {
	nodeTree := New()
	nodeTree.AddData("config/app.yaml", []byte("port: 8080"))
	nodeTree.AddData("config/env/app.env", []byte("MODE=test"))

	fileTarget := coreio.NewMemoryMedium()
	err := nodeTree.CopyTo(fileTarget, "config/app.yaml", "backup/app.yaml")
	core.RequireNoError(t, err)
	content, err := fileTarget.Read("backup/app.yaml")
	core.RequireNoError(t, err)
	core.AssertEqual(t, "port: 8080", content)

	dirTarget := coreio.NewMemoryMedium()
	err = nodeTree.CopyTo(dirTarget, "config", "backup/config")
	core.RequireNoError(t, err)
	content, err = dirTarget.Read("backup/config/app.yaml")
	core.RequireNoError(t, err)
	core.AssertEqual(t, "port: 8080", content)
	content, err = dirTarget.Read("backup/config/env/app.env")
	core.RequireNoError(t, err)
	core.AssertEqual(t, "MODE=test", content)
}

func TestNode_CopyTo_Bad(t *core.T) {
	nodeTree := New()
	err := nodeTree.CopyTo(coreio.NewMemoryMedium(), "missing", "backup/missing")
	core.AssertError(t, err)
}

func TestNode_MediumFacade_Good(t *core.T) {
	nodeTree := New()

	core.RequireNoError(t, nodeTree.Write("docs/readme.txt", "hello"))
	core.RequireNoError(t, nodeTree.WriteMode("docs/mode.txt", "mode", 0600))
	core.RequireNoError(t, nodeTree.Write("docs/guide.txt", "guide"))
	core.RequireNoError(t, nodeTree.EnsureDir("ignored"))

	value, err := nodeTree.Read("docs/readme.txt")
	core.RequireNoError(t, err)
	core.AssertEqual(t, "hello", value)

	value, err = nodeTree.Read("docs/guide.txt")
	core.RequireNoError(t, err)
	core.AssertEqual(t, "guide", value)

	core.AssertTrue(t, nodeTree.IsFile("docs/readme.txt"))
	core.AssertTrue(t, nodeTree.IsDir("docs"))

	entries, err := nodeTree.List("docs")
	core.RequireNoError(t, err)
	core.AssertEqual(t, []string{"guide.txt", "mode.txt", "readme.txt"}, sortedNames(entries))

	file, err := nodeTree.Open("docs/readme.txt")
	core.RequireNoError(t, err)
	info, err := file.Stat()
	core.RequireNoError(t, err)
	core.AssertEqual(t, "readme.txt", info.Name())
	core.AssertEqual(t, fs.FileMode(0444), info.Mode())
	core.AssertFalse(t, info.IsDir())
	core.AssertNil(t, info.Sys())
	core.RequireNoError(t, file.Close())

	dir, err := nodeTree.Open("docs")
	core.RequireNoError(t, err)
	dirInfo, err := dir.Stat()
	core.RequireNoError(t, err)
	core.AssertEqual(t, "docs", dirInfo.Name())
	core.AssertTrue(t, dirInfo.IsDir())
	core.AssertEqual(t, fs.ModeDir|0555, dirInfo.Mode())
	core.AssertNil(t, dirInfo.Sys())
	core.RequireNoError(t, dir.Close())

	createWriter, err := nodeTree.Create("docs/generated.txt")
	core.RequireNoError(t, err)
	_, err = createWriter.Write([]byte("generated"))
	core.RequireNoError(t, err)
	core.RequireNoError(t, createWriter.Close())

	appendWriter, err := nodeTree.Append("docs/generated.txt")
	core.RequireNoError(t, err)
	_, err = appendWriter.Write([]byte(" content"))
	core.RequireNoError(t, err)
	core.RequireNoError(t, appendWriter.Close())

	streamReader, err := nodeTree.ReadStream("docs/generated.txt")
	core.RequireNoError(t, err)
	streamData, err := io.ReadAll(streamReader)
	core.RequireNoError(t, err)
	core.AssertEqual(t, "generated content", string(streamData))
	core.RequireNoError(t, streamReader.Close())

	writeStream, err := nodeTree.WriteStream("docs/stream.txt")
	core.RequireNoError(t, err)
	_, err = writeStream.Write([]byte("stream"))
	core.RequireNoError(t, err)
	core.RequireNoError(t, writeStream.Close())

	core.RequireNoError(t, nodeTree.Rename("docs/stream.txt", "docs/stream-renamed.txt"))
	core.AssertTrue(t, nodeTree.Exists("docs/stream-renamed.txt"))

	core.RequireNoError(t, nodeTree.Delete("docs/stream-renamed.txt"))
	core.AssertFalse(t, nodeTree.Exists("docs/stream-renamed.txt"))

	core.RequireNoError(t, nodeTree.DeleteAll("docs"))
	core.AssertFalse(t, nodeTree.Exists("docs"))
}

func TestNode_ToTar_Good(t *core.T) {
	nodeTree := New()
	nodeTree.AddData("foo.txt", []byte("foo"))
	nodeTree.AddData("bar/baz.txt", []byte("baz"))

	tarball, err := nodeTree.ToTar()
	core.RequireNoError(t, err)
	core.RequireNotEmpty(t, tarball)

	tarReader := tar.NewReader(bytes.NewReader(tarball))
	files := make(map[string]string)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		core.RequireNoError(t, err)
		content, err := io.ReadAll(tarReader)
		core.RequireNoError(t, err)
		files[header.Name] = string(content)
	}

	core.AssertEqual(t, "foo", files["foo.txt"])
	core.AssertEqual(t, "baz", files["bar/baz.txt"])
}

func TestNode_FromTar_Good(t *core.T) {
	buffer := new(bytes.Buffer)
	tarWriter := tar.NewWriter(buffer)

	for _, file := range []struct{ Name, Body string }{
		{"foo.txt", "foo"},
		{"bar/baz.txt", "baz"},
	} {
		hdr := &tar.Header{
			Name:     file.Name,
			Mode:     0600,
			Size:     int64(len(file.Body)),
			Typeflag: tar.TypeReg,
		}
		core.RequireNoError(t, tarWriter.WriteHeader(hdr))
		_, err := tarWriter.Write([]byte(file.Body))
		core.RequireNoError(t, err)
	}
	core.RequireNoError(t, tarWriter.Close())

	nodeTree, err := FromTar(buffer.Bytes())
	core.RequireNoError(t, err)

	core.AssertTrue(t, nodeTree.Exists("foo.txt"), "foo.txt should exist")
	core.AssertTrue(t, nodeTree.Exists("bar/baz.txt"), "bar/baz.txt should exist")
}

func TestNode_FromTar_Bad(t *core.T) {
	truncated := make([]byte, 100)
	_, err := FromTar(truncated)
	core.AssertError(t, err)
}

func TestNode_TarRoundTrip_Good(t *core.T) {
	nodeTree1 := New()
	nodeTree1.AddData("a.txt", []byte("alpha"))
	nodeTree1.AddData("b/c.txt", []byte("charlie"))

	tarball, err := nodeTree1.ToTar()
	core.RequireNoError(t, err)

	nodeTree2, err := FromTar(tarball)
	core.RequireNoError(t, err)

	data, err := nodeTree2.ReadFile("a.txt")
	core.RequireNoError(t, err)
	core.AssertEqual(t, []byte("alpha"), data)

	data, err = nodeTree2.ReadFile("b/c.txt")
	core.RequireNoError(t, err)
	core.AssertEqual(t, []byte("charlie"), data)
}

func TestNode_FSInterface_Good(t *core.T) {
	nodeTree := New()
	nodeTree.AddData("hello.txt", []byte("world"))

	var fsys fs.FS = nodeTree
	file, err := fsys.Open("hello.txt")
	core.RequireNoError(t, err)
	defer file.Close()

	var statFS fs.StatFS = nodeTree
	info, err := statFS.Stat("hello.txt")
	core.RequireNoError(t, err)
	core.AssertEqual(t, "hello.txt", info.Name())
	core.AssertEqual(t, int64(5), info.Size())

	var readFS fs.ReadFileFS = nodeTree
	data, err := readFS.ReadFile("hello.txt")
	core.RequireNoError(t, err)
	core.AssertEqual(t, []byte("world"), data)
}

func sortedNames(entries []fs.DirEntry) []string {
	var names []string
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	sort.Strings(names)
	return names
}
