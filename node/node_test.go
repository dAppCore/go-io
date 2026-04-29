package node

import (
	"archive/tar"
	core "dappco.re/go"
	coreio "dappco.re/go/io"
	"io"
	"io/fs"
	"sort"
	"time"
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

func TestNode_MediumFacadeGood(t *core.T) {
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

	tarReader := tar.NewReader(core.NewReader(string(tarball)))
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
	buffer := core.NewBuffer()
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

func TestNode_TarRoundTripGood(t *core.T) {
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

func TestNode_FSInterfaceGood(t *core.T) {
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

func TestNode_New_Bad(t *core.T) {
	first := New()
	second := New()
	first.AddData("only-first.txt", []byte("payload"))
	core.AssertFalse(t, second.Exists("only-first.txt"))
}

func TestNode_New_Ugly(t *core.T) {
	nodeTree := New()
	nodeTree.AddData("/leading.txt", []byte("payload"))
	core.AssertTrue(t, nodeTree.Exists("leading.txt"))
}

func TestNode_ArchiveBuffer_Write_Good(t *core.T) {
	buffer := &nodeArchiveBuffer{}
	count, err := buffer.Write([]byte("payload"))
	core.AssertNoError(t, err)
	core.AssertEqual(t, len("payload"), count)
}

func TestNode_ArchiveBuffer_Write_Bad(t *core.T) {
	buffer := &nodeArchiveBuffer{}
	count, err := buffer.Write(nil)
	core.AssertNoError(t, err)
	core.AssertEqual(t, 0, count)
}

func TestNode_ArchiveBuffer_Write_Ugly(t *core.T) {
	buffer := &nodeArchiveBuffer{data: []byte("a")}
	count, err := buffer.Write([]byte("b"))
	core.AssertNoError(t, err)
	core.AssertEqual(t, 1, count)
}

func TestNode_Node_AddData_Good(t *core.T) {
	nodeTree := New()
	nodeTree.AddData("file.txt", []byte("payload"))
	core.AssertTrue(t, nodeTree.Exists("file.txt"))
}

func TestNode_Node_AddData_Bad(t *core.T) {
	nodeTree := New()
	nodeTree.AddData("", []byte("payload"))
	core.AssertTrue(t, nodeTree.Exists(""))
}

func TestNode_Node_AddData_Ugly(t *core.T) {
	nodeTree := New()
	nodeTree.AddData("dir/", []byte("payload"))
	core.AssertFalse(t, nodeTree.Exists("dir"))
}

func TestNode_Node_ToTar_Good(t *core.T) {
	nodeTree := New()
	nodeTree.AddData("file.txt", []byte("payload"))
	data, err := nodeTree.ToTar()
	core.AssertNoError(t, err)
	core.AssertNotEmpty(t, data)
}

func TestNode_Node_ToTar_Bad(t *core.T) {
	nodeTree := New()
	data, err := nodeTree.ToTar()
	core.AssertNoError(t, err)
	core.AssertNotEmpty(t, data)
}

func TestNode_Node_ToTar_Ugly(t *core.T) {
	nodeTree := New()
	nodeTree.AddData("nested/file.txt", []byte(""))
	data, err := nodeTree.ToTar()
	core.AssertNoError(t, err)
	core.AssertContains(t, string(data), "nested/file.txt")
}

func TestNode_FromTar_Ugly(t *core.T) {
	source := New()
	data, err := source.ToTar()
	core.RequireNoError(t, err)
	restored, err := FromTar(data)
	core.AssertNoError(t, err)
	core.AssertNotNil(t, restored)
}

func TestNode_Node_LoadTar_Good(t *core.T) {
	source := New()
	source.AddData("file.txt", []byte("payload"))
	data, err := source.ToTar()
	core.RequireNoError(t, err)
	nodeTree := New()
	err = nodeTree.LoadTar(data)
	core.AssertNoError(t, err)
	core.AssertTrue(t, nodeTree.Exists("file.txt"))
}

func TestNode_Node_LoadTar_Bad(t *core.T) {
	nodeTree := New()
	err := nodeTree.LoadTar([]byte("not tar"))
	core.AssertError(t, err)
	core.AssertEmpty(t, nodeTree.files)
}

func TestNode_Node_LoadTar_Ugly(t *core.T) {
	nodeTree := New()
	source := New()
	data, err := source.ToTar()
	core.RequireNoError(t, err)
	err = nodeTree.LoadTar(data)
	core.AssertNoError(t, err)
}

func TestNode_Node_Walk_Good(t *core.T) {
	nodeTree := New()
	nodeTree.AddData("dir/file.txt", []byte("payload"))
	var paths []string
	err := nodeTree.Walk(".", func(path string, _ fs.DirEntry, _ error) error {
		paths = append(paths, path)
		return nil
	}, WalkOptions{})
	core.AssertNoError(t, err)
	core.AssertContains(t, paths, "dir/file.txt")
}

func TestNode_Node_Walk_Bad(t *core.T) {
	nodeTree := New()
	err := nodeTree.Walk("missing", func(_ string, _ fs.DirEntry, err error) error {
		return err
	}, WalkOptions{})
	core.AssertError(t, err)
}

func TestNode_Node_Walk_Ugly(t *core.T) {
	nodeTree := New()
	err := nodeTree.Walk("missing", func(_ string, _ fs.DirEntry, _ error) error {
		return nil
	}, WalkOptions{SkipErrors: true})
	core.AssertNoError(t, err)
}

func TestNode_Node_ReadFile_Good(t *core.T) {
	nodeTree := New()
	nodeTree.AddData("file.txt", []byte("payload"))
	got, err := nodeTree.ReadFile("file.txt")
	core.AssertNoError(t, err)
	core.AssertEqual(t, []byte("payload"), got)
}

func TestNode_Node_ReadFile_Bad(t *core.T) {
	nodeTree := New()
	got, err := nodeTree.ReadFile("missing.txt")
	core.AssertError(t, err)
	core.AssertNil(t, got)
}

func TestNode_Node_ReadFile_Ugly(t *core.T) {
	nodeTree := New()
	nodeTree.AddData("/file.txt", []byte("payload"))
	got, err := nodeTree.ReadFile("/file.txt")
	core.AssertNoError(t, err)
	core.AssertEqual(t, []byte("payload"), got)
}

func TestNode_Node_ExportFile_Good(t *core.T) {
	nodeTree := New()
	nodeTree.AddData("file.txt", []byte("payload"))
	destination := core.Path(t.TempDir(), "file.txt")
	err := nodeTree.ExportFile("file.txt", destination, 0644)
	core.AssertNoError(t, err)
}

func TestNode_Node_ExportFile_Bad(t *core.T) {
	nodeTree := New()
	destination := core.Path(t.TempDir(), "missing.txt")
	err := nodeTree.ExportFile("missing.txt", destination, 0644)
	core.AssertError(t, err)
}

func TestNode_Node_ExportFile_Ugly(t *core.T) {
	nodeTree := New()
	nodeTree.AddData("file.txt", []byte("payload"))
	err := nodeTree.ExportFile("file.txt", core.Path(t.TempDir(), "missing", "file.txt"), 0644)
	core.AssertError(t, err)
}

func TestNode_Node_CopyTo_Good(t *core.T) {
	nodeTree := New()
	nodeTree.AddData("file.txt", []byte("payload"))
	target := coreio.NewMemoryMedium()
	err := nodeTree.CopyTo(target, "file.txt", "copy.txt")
	core.AssertNoError(t, err)
	core.AssertTrue(t, target.Exists("copy.txt"))
}

func TestNode_Node_CopyTo_Bad(t *core.T) {
	nodeTree := New()
	target := coreio.NewMemoryMedium()
	err := nodeTree.CopyTo(target, "missing.txt", "copy.txt")
	core.AssertError(t, err)
}

func TestNode_Node_CopyTo_Ugly(t *core.T) {
	nodeTree := New()
	nodeTree.AddData("dir/file.txt", []byte("payload"))
	target := coreio.NewMemoryMedium()
	err := nodeTree.CopyTo(target, "dir", "backup")
	core.AssertNoError(t, err)
	core.AssertTrue(t, target.Exists("backup/file.txt"))
}

func TestNode_Node_Open_Good(t *core.T) {
	nodeTree := New()
	nodeTree.AddData("file.txt", []byte("payload"))
	file, err := nodeTree.Open("file.txt")
	core.AssertNoError(t, err)
	core.AssertNotNil(t, file)
	core.RequireNoError(t, file.Close())
}

func TestNode_Node_Open_Bad(t *core.T) {
	nodeTree := New()
	file, err := nodeTree.Open("missing.txt")
	core.AssertError(t, err)
	core.AssertNil(t, file)
}

func TestNode_Node_Open_Ugly(t *core.T) {
	nodeTree := New()
	nodeTree.AddData("dir/file.txt", []byte("payload"))
	file, err := nodeTree.Open("dir")
	core.AssertNoError(t, err)
	core.AssertNotNil(t, file)
}

func TestNode_Node_Stat_Good(t *core.T) {
	nodeTree := New()
	nodeTree.AddData("file.txt", []byte("payload"))
	info, err := nodeTree.Stat("file.txt")
	core.AssertNoError(t, err)
	core.AssertEqual(t, "file.txt", info.Name())
}

func TestNode_Node_Stat_Bad(t *core.T) {
	nodeTree := New()
	info, err := nodeTree.Stat("missing.txt")
	core.AssertError(t, err)
	core.AssertNil(t, info)
}

func TestNode_Node_Stat_Ugly(t *core.T) {
	nodeTree := New()
	info, err := nodeTree.Stat(".")
	core.AssertNoError(t, err)
	core.AssertTrue(t, info.IsDir())
}

func TestNode_Node_ReadDir_Good(t *core.T) {
	nodeTree := New()
	nodeTree.AddData("dir/file.txt", []byte("payload"))
	entries, err := nodeTree.ReadDir("dir")
	core.AssertNoError(t, err)
	core.AssertLen(t, entries, 1)
}

func TestNode_Node_ReadDir_Bad(t *core.T) {
	nodeTree := New()
	entries, err := nodeTree.ReadDir("missing")
	core.AssertError(t, err)
	core.AssertNil(t, entries)
}

func TestNode_Node_ReadDir_Ugly(t *core.T) {
	nodeTree := New()
	nodeTree.AddData("file.txt", []byte("payload"))
	entries, err := nodeTree.ReadDir("file.txt")
	core.AssertError(t, err)
	core.AssertNil(t, entries)
}

func TestNode_Node_Read_Good(t *core.T) {
	nodeTree := New()
	nodeTree.AddData("file.txt", []byte("payload"))
	got, err := nodeTree.Read("file.txt")
	core.AssertNoError(t, err)
	core.AssertEqual(t, "payload", got)
}

func TestNode_Node_Read_Bad(t *core.T) {
	nodeTree := New()
	got, err := nodeTree.Read("missing.txt")
	core.AssertError(t, err)
	core.AssertEqual(t, "", got)
}

func TestNode_Node_Read_Ugly(t *core.T) {
	nodeTree := New()
	nodeTree.AddData("/file.txt", []byte("payload"))
	got, err := nodeTree.Read("/file.txt")
	core.AssertNoError(t, err)
	core.AssertEqual(t, "payload", got)
}

func TestNode_Node_Write_Good(t *core.T) {
	nodeTree := New()
	err := nodeTree.Write("file.txt", "payload")
	core.AssertNoError(t, err)
	core.AssertTrue(t, nodeTree.Exists("file.txt"))
}

func TestNode_Node_Write_Bad(t *core.T) {
	nodeTree := New()
	err := nodeTree.Write("", "payload")
	core.AssertError(t, err)
	core.AssertTrue(t, nodeTree.Exists(""))
}

func TestNode_Node_Write_Ugly(t *core.T) {
	nodeTree := New()
	err := nodeTree.Write("/file.txt", "")
	core.AssertNoError(t, err)
	core.AssertTrue(t, nodeTree.Exists("file.txt"))
}

func TestNode_Node_WriteMode_Good(t *core.T) {
	nodeTree := New()
	err := nodeTree.WriteMode("file.txt", "payload", 0600)
	core.AssertNoError(t, err)
	core.AssertTrue(t, nodeTree.Exists("file.txt"))
}

func TestNode_Node_WriteMode_Bad(t *core.T) {
	nodeTree := New()
	err := nodeTree.WriteMode("", "payload", 0600)
	core.AssertError(t, err)
	core.AssertTrue(t, nodeTree.Exists(""))
}

func TestNode_Node_WriteMode_Ugly(t *core.T) {
	nodeTree := New()
	err := nodeTree.WriteMode("file.txt", "payload", 0)
	core.AssertNoError(t, err)
	core.AssertTrue(t, nodeTree.Exists("file.txt"))
}

func TestNode_Node_EnsureDir_Good(t *core.T) {
	nodeTree := New()
	err := nodeTree.EnsureDir("dir")
	core.AssertNoError(t, err)
	core.AssertFalse(t, nodeTree.Exists("dir"))
}

func TestNode_Node_EnsureDir_Bad(t *core.T) {
	nodeTree := New()
	err := nodeTree.EnsureDir("")
	core.AssertNoError(t, err)
	core.AssertTrue(t, nodeTree.Exists(""))
}

func TestNode_Node_EnsureDir_Ugly(t *core.T) {
	nodeTree := New()
	err := nodeTree.EnsureDir("a/b/c")
	core.AssertNoError(t, err)
	core.AssertFalse(t, nodeTree.Exists("a/b/c"))
}

func TestNode_Node_Exists_Good(t *core.T) {
	nodeTree := New()
	nodeTree.AddData("file.txt", []byte("payload"))
	got := nodeTree.Exists("file.txt")
	core.AssertTrue(t, got)
}

func TestNode_Node_Exists_Bad(t *core.T) {
	nodeTree := New()
	got := nodeTree.Exists("missing.txt")
	core.AssertFalse(t, got)
}

func TestNode_Node_Exists_Ugly(t *core.T) {
	nodeTree := New()
	got := nodeTree.Exists("")
	core.AssertTrue(t, got)
}

func TestNode_Node_IsFile_Good(t *core.T) {
	nodeTree := New()
	nodeTree.AddData("file.txt", []byte("payload"))
	got := nodeTree.IsFile("file.txt")
	core.AssertTrue(t, got)
}

func TestNode_Node_IsFile_Bad(t *core.T) {
	nodeTree := New()
	got := nodeTree.IsFile("missing.txt")
	core.AssertFalse(t, got)
}

func TestNode_Node_IsFile_Ugly(t *core.T) {
	nodeTree := New()
	nodeTree.AddData("dir/file.txt", []byte("payload"))
	got := nodeTree.IsFile("dir")
	core.AssertFalse(t, got)
}

func TestNode_Node_IsDir_Good(t *core.T) {
	nodeTree := New()
	nodeTree.AddData("dir/file.txt", []byte("payload"))
	got := nodeTree.IsDir("dir")
	core.AssertTrue(t, got)
}

func TestNode_Node_IsDir_Bad(t *core.T) {
	nodeTree := New()
	got := nodeTree.IsDir("missing")
	core.AssertFalse(t, got)
}

func TestNode_Node_IsDir_Ugly(t *core.T) {
	nodeTree := New()
	nodeTree.AddData("file.txt", []byte("payload"))
	got := nodeTree.IsDir("file.txt")
	core.AssertFalse(t, got)
}

func TestNode_Node_Delete_Good(t *core.T) {
	nodeTree := New()
	nodeTree.AddData("file.txt", []byte("payload"))
	err := nodeTree.Delete("file.txt")
	core.AssertNoError(t, err)
	core.AssertFalse(t, nodeTree.Exists("file.txt"))
}

func TestNode_Node_Delete_Bad(t *core.T) {
	nodeTree := New()
	err := nodeTree.Delete("missing.txt")
	core.AssertError(t, err)
	core.AssertFalse(t, nodeTree.Exists("missing.txt"))
}

func TestNode_Node_Delete_Ugly(t *core.T) {
	nodeTree := New()
	err := nodeTree.Delete("")
	core.AssertError(t, err)
	core.AssertTrue(t, nodeTree.Exists(""))
}

func TestNode_Node_DeleteAll_Good(t *core.T) {
	nodeTree := New()
	nodeTree.AddData("dir/file.txt", []byte("payload"))
	err := nodeTree.DeleteAll("dir")
	core.AssertNoError(t, err)
	core.AssertFalse(t, nodeTree.Exists("dir/file.txt"))
}

func TestNode_Node_DeleteAll_Bad(t *core.T) {
	nodeTree := New()
	err := nodeTree.DeleteAll("missing")
	core.AssertError(t, err)
	core.AssertFalse(t, nodeTree.Exists("missing"))
}

func TestNode_Node_DeleteAll_Ugly(t *core.T) {
	nodeTree := New()
	err := nodeTree.DeleteAll("")
	core.AssertError(t, err)
	core.AssertTrue(t, nodeTree.Exists(""))
}

func TestNode_Node_Rename_Good(t *core.T) {
	nodeTree := New()
	nodeTree.AddData("old.txt", []byte("payload"))
	err := nodeTree.Rename("old.txt", "new.txt")
	core.AssertNoError(t, err)
	core.AssertTrue(t, nodeTree.Exists("new.txt"))
}

func TestNode_Node_Rename_Bad(t *core.T) {
	nodeTree := New()
	err := nodeTree.Rename("missing.txt", "new.txt")
	core.AssertError(t, err)
	core.AssertFalse(t, nodeTree.Exists("new.txt"))
}

func TestNode_Node_Rename_Ugly(t *core.T) {
	nodeTree := New()
	nodeTree.AddData("dir/old.txt", []byte("payload"))
	err := nodeTree.Rename("dir", "moved")
	core.AssertNoError(t, err)
	core.AssertTrue(t, nodeTree.Exists("moved/old.txt"))
}

func TestNode_Node_List_Good(t *core.T) {
	nodeTree := New()
	nodeTree.AddData("dir/file.txt", []byte("payload"))
	entries, err := nodeTree.List("dir")
	core.AssertNoError(t, err)
	core.AssertLen(t, entries, 1)
}

func TestNode_Node_List_Bad(t *core.T) {
	nodeTree := New()
	entries, err := nodeTree.List("missing")
	core.AssertError(t, err)
	core.AssertNil(t, entries)
}

func TestNode_Node_List_Ugly(t *core.T) {
	nodeTree := New()
	entries, err := nodeTree.List("")
	core.AssertNoError(t, err)
	core.AssertEmpty(t, entries)
}

func TestNode_Node_Create_Good(t *core.T) {
	nodeTree := New()
	writer, err := nodeTree.Create("file.txt")
	core.RequireNoError(t, err)
	_, writeErr := writer.Write([]byte("payload"))
	core.AssertNoError(t, writeErr)
	core.AssertNoError(t, writer.Close())
}

func TestNode_Node_Create_Bad(t *core.T) {
	nodeTree := New()
	writer, err := nodeTree.Create("")
	core.AssertNoError(t, err)
	core.AssertNotNil(t, writer)
}

func TestNode_Node_Create_Ugly(t *core.T) {
	nodeTree := New()
	writer, err := nodeTree.Create("/file.txt")
	core.RequireNoError(t, err)
	core.AssertNoError(t, writer.Close())
	core.AssertTrue(t, nodeTree.Exists("file.txt"))
}

func TestNode_Node_Append_Good(t *core.T) {
	nodeTree := New()
	core.RequireNoError(t, nodeTree.Write("file.txt", "a"))
	writer, err := nodeTree.Append("file.txt")
	core.RequireNoError(t, err)
	_, writeErr := writer.Write([]byte("b"))
	core.RequireNoError(t, writeErr)
	core.RequireNoError(t, writer.Close())
}

func TestNode_Node_Append_Bad(t *core.T) {
	nodeTree := New()
	writer, err := nodeTree.Append("")
	core.AssertNoError(t, err)
	core.AssertNotNil(t, writer)
}

func TestNode_Node_Append_Ugly(t *core.T) {
	nodeTree := New()
	writer, err := nodeTree.Append("new.txt")
	core.RequireNoError(t, err)
	_, writeErr := writer.Write([]byte("created"))
	core.RequireNoError(t, writeErr)
	core.RequireNoError(t, writer.Close())
}

func TestNode_Node_ReadStream_Good(t *core.T) {
	nodeTree := New()
	nodeTree.AddData("file.txt", []byte("payload"))
	reader, err := nodeTree.ReadStream("file.txt")
	core.RequireNoError(t, err)
	defer reader.Close()
	data, readErr := io.ReadAll(reader)
	core.AssertNoError(t, readErr)
	core.AssertEqual(t, "payload", string(data))
}

func TestNode_Node_ReadStream_Bad(t *core.T) {
	nodeTree := New()
	reader, err := nodeTree.ReadStream("missing.txt")
	core.AssertError(t, err)
	core.AssertNil(t, reader)
}

func TestNode_Node_ReadStream_Ugly(t *core.T) {
	nodeTree := New()
	nodeTree.AddData("dir/file.txt", []byte("payload"))
	reader, err := nodeTree.ReadStream("dir")
	core.AssertNoError(t, err)
	core.AssertNoError(t, reader.Close())
}

func TestNode_Node_WriteStream_Good(t *core.T) {
	nodeTree := New()
	writer, err := nodeTree.WriteStream("file.txt")
	core.RequireNoError(t, err)
	_, writeErr := writer.Write([]byte("payload"))
	core.AssertNoError(t, writeErr)
	core.AssertNoError(t, writer.Close())
}

func TestNode_Node_WriteStream_Bad(t *core.T) {
	nodeTree := New()
	writer, err := nodeTree.WriteStream("")
	core.AssertNoError(t, err)
	core.AssertNotNil(t, writer)
}

func TestNode_Node_WriteStream_Ugly(t *core.T) {
	nodeTree := New()
	writer, err := nodeTree.WriteStream("/file.txt")
	core.RequireNoError(t, err)
	core.AssertNoError(t, writer.Close())
	core.AssertTrue(t, nodeTree.Exists("file.txt"))
}

func TestNode_Writer_Write_Good(t *core.T) {
	writer := &nodeWriter{}
	count, err := writer.Write([]byte("payload"))
	core.AssertNoError(t, err)
	core.AssertEqual(t, len("payload"), count)
}

func TestNode_Writer_Write_Bad(t *core.T) {
	writer := &nodeWriter{}
	count, err := writer.Write(nil)
	core.AssertNoError(t, err)
	core.AssertEqual(t, 0, count)
}

func TestNode_Writer_Write_Ugly(t *core.T) {
	writer := &nodeWriter{buffer: []byte("a")}
	count, err := writer.Write([]byte("b"))
	core.AssertNoError(t, err)
	core.AssertEqual(t, 1, count)
}

func TestNode_Writer_Close_Good(t *core.T) {
	nodeTree := New()
	writer := &nodeWriter{node: nodeTree, path: "file.txt", buffer: []byte("payload")}
	err := writer.Close()
	core.AssertNoError(t, err)
	core.AssertTrue(t, nodeTree.Exists("file.txt"))
}

func TestNode_Writer_Close_Bad(t *core.T) {
	nodeTree := New()
	writer := &nodeWriter{node: nodeTree, path: "", buffer: []byte("payload")}
	err := writer.Close()
	core.AssertError(t, err)
}

func TestNode_Writer_Close_Ugly(t *core.T) {
	nodeTree := New()
	writer := &nodeWriter{node: nodeTree, path: "empty.txt"}
	err := writer.Close()
	core.AssertNoError(t, err)
	core.AssertTrue(t, nodeTree.Exists("empty.txt"))
}

func TestNode_File_Stat_Good(t *core.T) {
	file := &dataFile{name: "file.txt", content: []byte("payload")}
	info, err := file.Stat()
	core.AssertNoError(t, err)
	core.AssertEqual(t, "file.txt", info.Name())
}

func TestNode_File_Stat_Bad(t *core.T) {
	file := &dataFile{}
	info, err := file.Stat()
	core.AssertNoError(t, err)
	core.AssertEqual(t, ".", info.Name())
}

func TestNode_File_Stat_Ugly(t *core.T) {
	file := &dataFile{name: "empty.txt"}
	info, err := file.Stat()
	core.AssertNoError(t, err)
	core.AssertEqual(t, int64(0), info.Size())
}

func TestNode_File_Read_Good(t *core.T) {
	file := &dataFile{content: []byte("payload")}
	buffer := make([]byte, 1)
	count, err := file.Read(buffer)
	core.AssertErrorIs(t, err, io.EOF)
	core.AssertEqual(t, 0, count)
}

func TestNode_File_Read_Bad(t *core.T) {
	file := &dataFile{}
	buffer := make([]byte, 1)
	count, err := file.Read(buffer)
	core.AssertErrorIs(t, err, io.EOF)
	core.AssertEqual(t, 0, count)
}

func TestNode_File_Read_Ugly(t *core.T) {
	file := &dataFile{name: "file.txt"}
	count, err := file.Read(nil)
	core.AssertErrorIs(t, err, io.EOF)
	core.AssertEqual(t, 0, count)
}

func TestNode_File_Close_Good(t *core.T) {
	file := &dataFile{name: "file.txt"}
	err := file.Close()
	core.AssertNoError(t, err)
	core.AssertEqual(t, "file.txt", file.name)
}

func TestNode_File_Close_Bad(t *core.T) {
	file := &dataFile{}
	err := file.Close()
	core.AssertNoError(t, err)
	core.AssertEqual(t, "", file.name)
}

func TestNode_File_Close_Ugly(t *core.T) {
	file := &dataFile{modTime: time.Unix(1, 0)}
	err := file.Close()
	core.AssertNoError(t, err)
	core.AssertFalse(t, file.modTime.IsZero())
}

func TestNode_FileInfo_Name_Good(t *core.T) {
	info := &dataFileInfo{file: &dataFile{name: "dir/file.txt"}}
	got := info.Name()
	core.AssertEqual(t, "file.txt", got)
}

func TestNode_FileInfo_Name_Bad(t *core.T) {
	info := &dataFileInfo{file: &dataFile{}}
	got := info.Name()
	core.AssertEqual(t, ".", got)
}

func TestNode_FileInfo_Name_Ugly(t *core.T) {
	info := &dataFileInfo{file: &dataFile{name: "/leading.txt"}}
	got := info.Name()
	core.AssertEqual(t, "leading.txt", got)
}

func TestNode_FileInfo_Size_Good(t *core.T) {
	info := &dataFileInfo{file: &dataFile{content: []byte("payload")}}
	got := info.Size()
	core.AssertEqual(t, int64(7), got)
}

func TestNode_FileInfo_Size_Bad(t *core.T) {
	info := &dataFileInfo{file: &dataFile{}}
	got := info.Size()
	core.AssertEqual(t, int64(0), got)
}

func TestNode_FileInfo_Size_Ugly(t *core.T) {
	info := &dataFileInfo{file: &dataFile{content: nil}}
	got := info.Size()
	core.AssertEqual(t, int64(0), got)
}

func TestNode_FileInfo_Mode_Good(t *core.T) {
	info := &dataFileInfo{file: &dataFile{}}
	got := info.Mode()
	core.AssertEqual(t, fs.FileMode(0444), got)
}

func TestNode_FileInfo_Mode_Bad(t *core.T) {
	info := &dataFileInfo{file: &dataFile{name: "file.txt"}}
	got := info.Mode()
	core.AssertEqual(t, fs.FileMode(0444), got)
}

func TestNode_FileInfo_Mode_Ugly(t *core.T) {
	info := &dataFileInfo{file: &dataFile{content: []byte("payload")}}
	got := info.Mode()
	core.AssertFalse(t, got.IsDir())
}

func TestNode_FileInfo_ModTime_Good(t *core.T) {
	now := time.Unix(1, 0)
	info := &dataFileInfo{file: &dataFile{modTime: now}}
	got := info.ModTime()
	core.AssertTrue(t, got.Equal(now))
}

func TestNode_FileInfo_ModTime_Bad(t *core.T) {
	info := &dataFileInfo{file: &dataFile{}}
	got := info.ModTime()
	core.AssertTrue(t, got.IsZero())
}

func TestNode_FileInfo_ModTime_Ugly(t *core.T) {
	now := time.Unix(0, 1)
	info := &dataFileInfo{file: &dataFile{modTime: now}}
	got := info.ModTime()
	core.AssertEqual(t, 1, got.Nanosecond())
}

func TestNode_FileInfo_IsDir_Good(t *core.T) {
	info := &dataFileInfo{file: &dataFile{}}
	got := info.IsDir()
	core.AssertFalse(t, got)
}

func TestNode_FileInfo_IsDir_Bad(t *core.T) {
	info := &dataFileInfo{file: &dataFile{name: "dir"}}
	got := info.IsDir()
	core.AssertFalse(t, got)
}

func TestNode_FileInfo_IsDir_Ugly(t *core.T) {
	info := &dataFileInfo{file: &dataFile{content: []byte("payload")}}
	got := info.IsDir()
	core.AssertFalse(t, got)
}

func TestNode_FileInfo_Sys_Good(t *core.T) {
	info := &dataFileInfo{file: &dataFile{}}
	got := info.Sys()
	core.AssertNil(t, got)
}

func TestNode_FileInfo_Sys_Bad(t *core.T) {
	info := &dataFileInfo{file: &dataFile{name: "file.txt"}}
	got := info.Sys()
	core.AssertNil(t, got)
}

func TestNode_FileInfo_Sys_Ugly(t *core.T) {
	info := &dataFileInfo{file: &dataFile{content: []byte("payload")}}
	got := info.Sys()
	core.AssertNil(t, got)
}

func TestNode_FileReader_Stat_Good(t *core.T) {
	reader := &dataFileReader{file: &dataFile{name: "file.txt", content: []byte("payload")}}
	info, err := reader.Stat()
	core.AssertNoError(t, err)
	core.AssertEqual(t, "file.txt", info.Name())
}

func TestNode_FileReader_Stat_Bad(t *core.T) {
	reader := &dataFileReader{file: &dataFile{}}
	info, err := reader.Stat()
	core.AssertNoError(t, err)
	core.AssertEqual(t, ".", info.Name())
}

func TestNode_FileReader_Stat_Ugly(t *core.T) {
	reader := &dataFileReader{file: &dataFile{name: "empty.txt"}}
	info, err := reader.Stat()
	core.AssertNoError(t, err)
	core.AssertEqual(t, int64(0), info.Size())
}

func TestNode_FileReader_Read_Good(t *core.T) {
	reader := &dataFileReader{file: &dataFile{content: []byte("payload")}}
	buffer := make([]byte, 7)
	count, err := reader.Read(buffer)
	core.AssertNoError(t, err)
	core.AssertEqual(t, "payload", string(buffer[:count]))
}

func TestNode_FileReader_Read_Bad(t *core.T) {
	reader := &dataFileReader{file: &dataFile{content: []byte("x")}, offset: 1}
	buffer := make([]byte, 1)
	count, err := reader.Read(buffer)
	core.AssertErrorIs(t, err, io.EOF)
	core.AssertEqual(t, 0, count)
}

func TestNode_FileReader_Read_Ugly(t *core.T) {
	reader := &dataFileReader{file: &dataFile{content: []byte("payload")}}
	buffer := make([]byte, 3)
	count, err := reader.Read(buffer)
	core.AssertNoError(t, err)
	core.AssertEqual(t, "pay", string(buffer[:count]))
}

func TestNode_FileReader_Close_Good(t *core.T) {
	reader := &dataFileReader{file: &dataFile{name: "file.txt"}}
	err := reader.Close()
	core.AssertNoError(t, err)
	core.AssertEqual(t, "file.txt", reader.file.name)
}

func TestNode_FileReader_Close_Bad(t *core.T) {
	reader := &dataFileReader{file: &dataFile{}}
	err := reader.Close()
	core.AssertNoError(t, err)
	core.AssertEqual(t, "", reader.file.name)
}

func TestNode_FileReader_Close_Ugly(t *core.T) {
	reader := &dataFileReader{file: &dataFile{}, offset: 99}
	err := reader.Close()
	core.AssertNoError(t, err)
	core.AssertEqual(t, int64(99), reader.offset)
}

func TestNode_Info_Name_Good(t *core.T) {
	info := &dirInfo{name: "dir"}
	got := info.Name()
	core.AssertEqual(t, "dir", got)
}

func TestNode_Info_Name_Bad(t *core.T) {
	info := &dirInfo{}
	got := info.Name()
	core.AssertEqual(t, "", got)
}

func TestNode_Info_Name_Ugly(t *core.T) {
	info := &dirInfo{name: "."}
	got := info.Name()
	core.AssertEqual(t, ".", got)
}

func TestNode_Info_Size_Good(t *core.T) {
	info := &dirInfo{name: "dir"}
	got := info.Size()
	core.AssertEqual(t, int64(0), got)
}

func TestNode_Info_Size_Bad(t *core.T) {
	info := &dirInfo{}
	got := info.Size()
	core.AssertEqual(t, int64(0), got)
}

func TestNode_Info_Size_Ugly(t *core.T) {
	info := &dirInfo{name: "."}
	got := info.Size()
	core.AssertEqual(t, int64(0), got)
}

func TestNode_Info_Mode_Good(t *core.T) {
	info := &dirInfo{name: "dir"}
	got := info.Mode()
	core.AssertTrue(t, got.IsDir())
}

func TestNode_Info_Mode_Bad(t *core.T) {
	info := &dirInfo{}
	got := info.Mode()
	core.AssertTrue(t, got.IsDir())
}

func TestNode_Info_Mode_Ugly(t *core.T) {
	info := &dirInfo{name: "."}
	got := info.Mode()
	core.AssertEqual(t, fs.ModeDir|0555, got)
}

func TestNode_Info_ModTime_Good(t *core.T) {
	now := time.Unix(1, 0)
	info := &dirInfo{modTime: now}
	got := info.ModTime()
	core.AssertTrue(t, got.Equal(now))
}

func TestNode_Info_ModTime_Bad(t *core.T) {
	info := &dirInfo{}
	got := info.ModTime()
	core.AssertTrue(t, got.IsZero())
}

func TestNode_Info_ModTime_Ugly(t *core.T) {
	now := time.Unix(0, 1)
	info := &dirInfo{modTime: now}
	got := info.ModTime()
	core.AssertEqual(t, 1, got.Nanosecond())
}

func TestNode_Info_IsDir_Good(t *core.T) {
	info := &dirInfo{name: "dir"}
	got := info.IsDir()
	core.AssertTrue(t, got)
}

func TestNode_Info_IsDir_Bad(t *core.T) {
	info := &dirInfo{}
	got := info.IsDir()
	core.AssertTrue(t, got)
}

func TestNode_Info_IsDir_Ugly(t *core.T) {
	info := &dirInfo{name: "."}
	got := info.IsDir()
	core.AssertTrue(t, got)
}

func TestNode_Info_Sys_Good(t *core.T) {
	info := &dirInfo{name: "dir"}
	got := info.Sys()
	core.AssertNil(t, got)
}

func TestNode_Info_Sys_Bad(t *core.T) {
	info := &dirInfo{}
	got := info.Sys()
	core.AssertNil(t, got)
}

func TestNode_Info_Sys_Ugly(t *core.T) {
	info := &dirInfo{name: "."}
	got := info.Sys()
	core.AssertNil(t, got)
}
