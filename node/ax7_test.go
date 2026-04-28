package node

import (
	goio "io"
	"io/fs"
	"time"

	core "dappco.re/go"
	coreio "dappco.re/go/io"
)

func TestAX7_New_Good(t *core.T) {
	nodeTree := New()
	core.AssertNotNil(t, nodeTree)
	core.AssertNotNil(t, nodeTree.files)
}

func TestAX7_New_Bad(t *core.T) {
	first := New()
	second := New()
	first.AddData("only-first.txt", []byte("payload"))
	core.AssertFalse(t, second.Exists("only-first.txt"))
}

func TestAX7_New_Ugly(t *core.T) {
	nodeTree := New()
	nodeTree.AddData("/leading.txt", []byte("payload"))
	core.AssertTrue(t, nodeTree.Exists("leading.txt"))
}

func TestAX7_ArchiveBuffer_Write_Good(t *core.T) {
	buffer := &nodeArchiveBuffer{}
	count, err := buffer.Write([]byte("payload"))
	core.AssertNoError(t, err)
	core.AssertEqual(t, len("payload"), count)
}

func TestAX7_ArchiveBuffer_Write_Bad(t *core.T) {
	buffer := &nodeArchiveBuffer{}
	count, err := buffer.Write(nil)
	core.AssertNoError(t, err)
	core.AssertEqual(t, 0, count)
}

func TestAX7_ArchiveBuffer_Write_Ugly(t *core.T) {
	buffer := &nodeArchiveBuffer{data: []byte("a")}
	count, err := buffer.Write([]byte("b"))
	core.AssertNoError(t, err)
	core.AssertEqual(t, 1, count)
}

func TestAX7_Node_AddData_Good(t *core.T) {
	nodeTree := New()
	nodeTree.AddData("file.txt", []byte("payload"))
	core.AssertTrue(t, nodeTree.Exists("file.txt"))
}

func TestAX7_Node_AddData_Bad(t *core.T) {
	nodeTree := New()
	nodeTree.AddData("", []byte("payload"))
	core.AssertTrue(t, nodeTree.Exists(""))
}

func TestAX7_Node_AddData_Ugly(t *core.T) {
	nodeTree := New()
	nodeTree.AddData("dir/", []byte("payload"))
	core.AssertFalse(t, nodeTree.Exists("dir"))
}

func TestAX7_Node_ToTar_Good(t *core.T) {
	nodeTree := New()
	nodeTree.AddData("file.txt", []byte("payload"))
	data, err := nodeTree.ToTar()
	core.AssertNoError(t, err)
	core.AssertNotEmpty(t, data)
}

func TestAX7_Node_ToTar_Bad(t *core.T) {
	nodeTree := New()
	data, err := nodeTree.ToTar()
	core.AssertNoError(t, err)
	core.AssertNotEmpty(t, data)
}

func TestAX7_Node_ToTar_Ugly(t *core.T) {
	nodeTree := New()
	nodeTree.AddData("nested/file.txt", []byte(""))
	data, err := nodeTree.ToTar()
	core.AssertNoError(t, err)
	core.AssertContains(t, string(data), "nested/file.txt")
}

func TestAX7_FromTar_Good(t *core.T) {
	source := New()
	source.AddData("file.txt", []byte("payload"))
	data, err := source.ToTar()
	core.RequireNoError(t, err)
	restored, err := FromTar(data)
	core.AssertNoError(t, err)
	core.AssertTrue(t, restored.Exists("file.txt"))
}

func TestAX7_FromTar_Bad(t *core.T) {
	restored, err := FromTar([]byte("not tar"))
	core.AssertError(t, err)
	core.AssertNil(t, restored)
}

func TestAX7_FromTar_Ugly(t *core.T) {
	source := New()
	data, err := source.ToTar()
	core.RequireNoError(t, err)
	restored, err := FromTar(data)
	core.AssertNoError(t, err)
	core.AssertNotNil(t, restored)
}

func TestAX7_Node_LoadTar_Good(t *core.T) {
	source := New()
	source.AddData("file.txt", []byte("payload"))
	data, err := source.ToTar()
	core.RequireNoError(t, err)
	nodeTree := New()
	err = nodeTree.LoadTar(data)
	core.AssertNoError(t, err)
	core.AssertTrue(t, nodeTree.Exists("file.txt"))
}

func TestAX7_Node_LoadTar_Bad(t *core.T) {
	nodeTree := New()
	err := nodeTree.LoadTar([]byte("not tar"))
	core.AssertError(t, err)
	core.AssertEmpty(t, nodeTree.files)
}

func TestAX7_Node_LoadTar_Ugly(t *core.T) {
	nodeTree := New()
	source := New()
	data, err := source.ToTar()
	core.RequireNoError(t, err)
	err = nodeTree.LoadTar(data)
	core.AssertNoError(t, err)
}

func TestAX7_Node_Walk_Good(t *core.T) {
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

func TestAX7_Node_Walk_Bad(t *core.T) {
	nodeTree := New()
	err := nodeTree.Walk("missing", func(_ string, _ fs.DirEntry, err error) error {
		return err
	}, WalkOptions{})
	core.AssertError(t, err)
}

func TestAX7_Node_Walk_Ugly(t *core.T) {
	nodeTree := New()
	err := nodeTree.Walk("missing", func(_ string, _ fs.DirEntry, _ error) error {
		return nil
	}, WalkOptions{SkipErrors: true})
	core.AssertNoError(t, err)
}

func TestAX7_Node_ReadFile_Good(t *core.T) {
	nodeTree := New()
	nodeTree.AddData("file.txt", []byte("payload"))
	got, err := nodeTree.ReadFile("file.txt")
	core.AssertNoError(t, err)
	core.AssertEqual(t, []byte("payload"), got)
}

func TestAX7_Node_ReadFile_Bad(t *core.T) {
	nodeTree := New()
	got, err := nodeTree.ReadFile("missing.txt")
	core.AssertError(t, err)
	core.AssertNil(t, got)
}

func TestAX7_Node_ReadFile_Ugly(t *core.T) {
	nodeTree := New()
	nodeTree.AddData("/file.txt", []byte("payload"))
	got, err := nodeTree.ReadFile("/file.txt")
	core.AssertNoError(t, err)
	core.AssertEqual(t, []byte("payload"), got)
}

func TestAX7_Node_ExportFile_Good(t *core.T) {
	nodeTree := New()
	nodeTree.AddData("file.txt", []byte("payload"))
	destination := core.Path(t.TempDir(), "file.txt")
	err := nodeTree.ExportFile("file.txt", destination, 0644)
	core.AssertNoError(t, err)
}

func TestAX7_Node_ExportFile_Bad(t *core.T) {
	nodeTree := New()
	destination := core.Path(t.TempDir(), "missing.txt")
	err := nodeTree.ExportFile("missing.txt", destination, 0644)
	core.AssertError(t, err)
}

func TestAX7_Node_ExportFile_Ugly(t *core.T) {
	nodeTree := New()
	nodeTree.AddData("file.txt", []byte("payload"))
	err := nodeTree.ExportFile("file.txt", core.Path(t.TempDir(), "missing", "file.txt"), 0644)
	core.AssertError(t, err)
}

func TestAX7_Node_CopyTo_Good(t *core.T) {
	nodeTree := New()
	nodeTree.AddData("file.txt", []byte("payload"))
	target := coreio.NewMemoryMedium()
	err := nodeTree.CopyTo(target, "file.txt", "copy.txt")
	core.AssertNoError(t, err)
	core.AssertTrue(t, target.Exists("copy.txt"))
}

func TestAX7_Node_CopyTo_Bad(t *core.T) {
	nodeTree := New()
	target := coreio.NewMemoryMedium()
	err := nodeTree.CopyTo(target, "missing.txt", "copy.txt")
	core.AssertError(t, err)
}

func TestAX7_Node_CopyTo_Ugly(t *core.T) {
	nodeTree := New()
	nodeTree.AddData("dir/file.txt", []byte("payload"))
	target := coreio.NewMemoryMedium()
	err := nodeTree.CopyTo(target, "dir", "backup")
	core.AssertNoError(t, err)
	core.AssertTrue(t, target.Exists("backup/file.txt"))
}

func TestAX7_Node_Open_Good(t *core.T) {
	nodeTree := New()
	nodeTree.AddData("file.txt", []byte("payload"))
	file, err := nodeTree.Open("file.txt")
	core.AssertNoError(t, err)
	core.AssertNotNil(t, file)
	core.RequireNoError(t, file.Close())
}

func TestAX7_Node_Open_Bad(t *core.T) {
	nodeTree := New()
	file, err := nodeTree.Open("missing.txt")
	core.AssertError(t, err)
	core.AssertNil(t, file)
}

func TestAX7_Node_Open_Ugly(t *core.T) {
	nodeTree := New()
	nodeTree.AddData("dir/file.txt", []byte("payload"))
	file, err := nodeTree.Open("dir")
	core.AssertNoError(t, err)
	core.AssertNotNil(t, file)
}

func TestAX7_Node_Stat_Good(t *core.T) {
	nodeTree := New()
	nodeTree.AddData("file.txt", []byte("payload"))
	info, err := nodeTree.Stat("file.txt")
	core.AssertNoError(t, err)
	core.AssertEqual(t, "file.txt", info.Name())
}

func TestAX7_Node_Stat_Bad(t *core.T) {
	nodeTree := New()
	info, err := nodeTree.Stat("missing.txt")
	core.AssertError(t, err)
	core.AssertNil(t, info)
}

func TestAX7_Node_Stat_Ugly(t *core.T) {
	nodeTree := New()
	info, err := nodeTree.Stat(".")
	core.AssertNoError(t, err)
	core.AssertTrue(t, info.IsDir())
}

func TestAX7_Node_ReadDir_Good(t *core.T) {
	nodeTree := New()
	nodeTree.AddData("dir/file.txt", []byte("payload"))
	entries, err := nodeTree.ReadDir("dir")
	core.AssertNoError(t, err)
	core.AssertLen(t, entries, 1)
}

func TestAX7_Node_ReadDir_Bad(t *core.T) {
	nodeTree := New()
	entries, err := nodeTree.ReadDir("missing")
	core.AssertError(t, err)
	core.AssertNil(t, entries)
}

func TestAX7_Node_ReadDir_Ugly(t *core.T) {
	nodeTree := New()
	nodeTree.AddData("file.txt", []byte("payload"))
	entries, err := nodeTree.ReadDir("file.txt")
	core.AssertError(t, err)
	core.AssertNil(t, entries)
}

func TestAX7_Node_Read_Good(t *core.T) {
	nodeTree := New()
	nodeTree.AddData("file.txt", []byte("payload"))
	got, err := nodeTree.Read("file.txt")
	core.AssertNoError(t, err)
	core.AssertEqual(t, "payload", got)
}

func TestAX7_Node_Read_Bad(t *core.T) {
	nodeTree := New()
	got, err := nodeTree.Read("missing.txt")
	core.AssertError(t, err)
	core.AssertEqual(t, "", got)
}

func TestAX7_Node_Read_Ugly(t *core.T) {
	nodeTree := New()
	nodeTree.AddData("/file.txt", []byte("payload"))
	got, err := nodeTree.Read("/file.txt")
	core.AssertNoError(t, err)
	core.AssertEqual(t, "payload", got)
}

func TestAX7_Node_Write_Good(t *core.T) {
	nodeTree := New()
	err := nodeTree.Write("file.txt", "payload")
	core.AssertNoError(t, err)
	core.AssertTrue(t, nodeTree.Exists("file.txt"))
}

func TestAX7_Node_Write_Bad(t *core.T) {
	nodeTree := New()
	err := nodeTree.Write("", "payload")
	core.AssertError(t, err)
	core.AssertTrue(t, nodeTree.Exists(""))
}

func TestAX7_Node_Write_Ugly(t *core.T) {
	nodeTree := New()
	err := nodeTree.Write("/file.txt", "")
	core.AssertNoError(t, err)
	core.AssertTrue(t, nodeTree.Exists("file.txt"))
}

func TestAX7_Node_WriteMode_Good(t *core.T) {
	nodeTree := New()
	err := nodeTree.WriteMode("file.txt", "payload", 0600)
	core.AssertNoError(t, err)
	core.AssertTrue(t, nodeTree.Exists("file.txt"))
}

func TestAX7_Node_WriteMode_Bad(t *core.T) {
	nodeTree := New()
	err := nodeTree.WriteMode("", "payload", 0600)
	core.AssertError(t, err)
	core.AssertTrue(t, nodeTree.Exists(""))
}

func TestAX7_Node_WriteMode_Ugly(t *core.T) {
	nodeTree := New()
	err := nodeTree.WriteMode("file.txt", "payload", 0)
	core.AssertNoError(t, err)
	core.AssertTrue(t, nodeTree.Exists("file.txt"))
}

func TestAX7_Node_EnsureDir_Good(t *core.T) {
	nodeTree := New()
	err := nodeTree.EnsureDir("dir")
	core.AssertNoError(t, err)
	core.AssertFalse(t, nodeTree.Exists("dir"))
}

func TestAX7_Node_EnsureDir_Bad(t *core.T) {
	nodeTree := New()
	err := nodeTree.EnsureDir("")
	core.AssertNoError(t, err)
	core.AssertTrue(t, nodeTree.Exists(""))
}

func TestAX7_Node_EnsureDir_Ugly(t *core.T) {
	nodeTree := New()
	err := nodeTree.EnsureDir("a/b/c")
	core.AssertNoError(t, err)
	core.AssertFalse(t, nodeTree.Exists("a/b/c"))
}

func TestAX7_Node_Exists_Good(t *core.T) {
	nodeTree := New()
	nodeTree.AddData("file.txt", []byte("payload"))
	got := nodeTree.Exists("file.txt")
	core.AssertTrue(t, got)
}

func TestAX7_Node_Exists_Bad(t *core.T) {
	nodeTree := New()
	got := nodeTree.Exists("missing.txt")
	core.AssertFalse(t, got)
}

func TestAX7_Node_Exists_Ugly(t *core.T) {
	nodeTree := New()
	got := nodeTree.Exists("")
	core.AssertTrue(t, got)
}

func TestAX7_Node_IsFile_Good(t *core.T) {
	nodeTree := New()
	nodeTree.AddData("file.txt", []byte("payload"))
	got := nodeTree.IsFile("file.txt")
	core.AssertTrue(t, got)
}

func TestAX7_Node_IsFile_Bad(t *core.T) {
	nodeTree := New()
	got := nodeTree.IsFile("missing.txt")
	core.AssertFalse(t, got)
}

func TestAX7_Node_IsFile_Ugly(t *core.T) {
	nodeTree := New()
	nodeTree.AddData("dir/file.txt", []byte("payload"))
	got := nodeTree.IsFile("dir")
	core.AssertFalse(t, got)
}

func TestAX7_Node_IsDir_Good(t *core.T) {
	nodeTree := New()
	nodeTree.AddData("dir/file.txt", []byte("payload"))
	got := nodeTree.IsDir("dir")
	core.AssertTrue(t, got)
}

func TestAX7_Node_IsDir_Bad(t *core.T) {
	nodeTree := New()
	got := nodeTree.IsDir("missing")
	core.AssertFalse(t, got)
}

func TestAX7_Node_IsDir_Ugly(t *core.T) {
	nodeTree := New()
	nodeTree.AddData("file.txt", []byte("payload"))
	got := nodeTree.IsDir("file.txt")
	core.AssertFalse(t, got)
}

func TestAX7_Node_Delete_Good(t *core.T) {
	nodeTree := New()
	nodeTree.AddData("file.txt", []byte("payload"))
	err := nodeTree.Delete("file.txt")
	core.AssertNoError(t, err)
	core.AssertFalse(t, nodeTree.Exists("file.txt"))
}

func TestAX7_Node_Delete_Bad(t *core.T) {
	nodeTree := New()
	err := nodeTree.Delete("missing.txt")
	core.AssertError(t, err)
	core.AssertFalse(t, nodeTree.Exists("missing.txt"))
}

func TestAX7_Node_Delete_Ugly(t *core.T) {
	nodeTree := New()
	err := nodeTree.Delete("")
	core.AssertError(t, err)
	core.AssertTrue(t, nodeTree.Exists(""))
}

func TestAX7_Node_DeleteAll_Good(t *core.T) {
	nodeTree := New()
	nodeTree.AddData("dir/file.txt", []byte("payload"))
	err := nodeTree.DeleteAll("dir")
	core.AssertNoError(t, err)
	core.AssertFalse(t, nodeTree.Exists("dir/file.txt"))
}

func TestAX7_Node_DeleteAll_Bad(t *core.T) {
	nodeTree := New()
	err := nodeTree.DeleteAll("missing")
	core.AssertError(t, err)
	core.AssertFalse(t, nodeTree.Exists("missing"))
}

func TestAX7_Node_DeleteAll_Ugly(t *core.T) {
	nodeTree := New()
	err := nodeTree.DeleteAll("")
	core.AssertError(t, err)
	core.AssertTrue(t, nodeTree.Exists(""))
}

func TestAX7_Node_Rename_Good(t *core.T) {
	nodeTree := New()
	nodeTree.AddData("old.txt", []byte("payload"))
	err := nodeTree.Rename("old.txt", "new.txt")
	core.AssertNoError(t, err)
	core.AssertTrue(t, nodeTree.Exists("new.txt"))
}

func TestAX7_Node_Rename_Bad(t *core.T) {
	nodeTree := New()
	err := nodeTree.Rename("missing.txt", "new.txt")
	core.AssertError(t, err)
	core.AssertFalse(t, nodeTree.Exists("new.txt"))
}

func TestAX7_Node_Rename_Ugly(t *core.T) {
	nodeTree := New()
	nodeTree.AddData("dir/old.txt", []byte("payload"))
	err := nodeTree.Rename("dir", "moved")
	core.AssertNoError(t, err)
	core.AssertTrue(t, nodeTree.Exists("moved/old.txt"))
}

func TestAX7_Node_List_Good(t *core.T) {
	nodeTree := New()
	nodeTree.AddData("dir/file.txt", []byte("payload"))
	entries, err := nodeTree.List("dir")
	core.AssertNoError(t, err)
	core.AssertLen(t, entries, 1)
}

func TestAX7_Node_List_Bad(t *core.T) {
	nodeTree := New()
	entries, err := nodeTree.List("missing")
	core.AssertError(t, err)
	core.AssertNil(t, entries)
}

func TestAX7_Node_List_Ugly(t *core.T) {
	nodeTree := New()
	entries, err := nodeTree.List("")
	core.AssertNoError(t, err)
	core.AssertEmpty(t, entries)
}

func TestAX7_Node_Create_Good(t *core.T) {
	nodeTree := New()
	writer, err := nodeTree.Create("file.txt")
	core.RequireNoError(t, err)
	_, writeErr := writer.Write([]byte("payload"))
	core.AssertNoError(t, writeErr)
	core.AssertNoError(t, writer.Close())
}

func TestAX7_Node_Create_Bad(t *core.T) {
	nodeTree := New()
	writer, err := nodeTree.Create("")
	core.AssertNoError(t, err)
	core.AssertNotNil(t, writer)
}

func TestAX7_Node_Create_Ugly(t *core.T) {
	nodeTree := New()
	writer, err := nodeTree.Create("/file.txt")
	core.RequireNoError(t, err)
	core.AssertNoError(t, writer.Close())
	core.AssertTrue(t, nodeTree.Exists("file.txt"))
}

func TestAX7_Node_Append_Good(t *core.T) {
	nodeTree := New()
	core.RequireNoError(t, nodeTree.Write("file.txt", "a"))
	writer, err := nodeTree.Append("file.txt")
	core.RequireNoError(t, err)
	_, writeErr := writer.Write([]byte("b"))
	core.RequireNoError(t, writeErr)
	core.RequireNoError(t, writer.Close())
}

func TestAX7_Node_Append_Bad(t *core.T) {
	nodeTree := New()
	writer, err := nodeTree.Append("")
	core.AssertNoError(t, err)
	core.AssertNotNil(t, writer)
}

func TestAX7_Node_Append_Ugly(t *core.T) {
	nodeTree := New()
	writer, err := nodeTree.Append("new.txt")
	core.RequireNoError(t, err)
	_, writeErr := writer.Write([]byte("created"))
	core.RequireNoError(t, writeErr)
	core.RequireNoError(t, writer.Close())
}

func TestAX7_Node_ReadStream_Good(t *core.T) {
	nodeTree := New()
	nodeTree.AddData("file.txt", []byte("payload"))
	reader, err := nodeTree.ReadStream("file.txt")
	core.RequireNoError(t, err)
	defer reader.Close()
	data, readErr := goio.ReadAll(reader)
	core.AssertNoError(t, readErr)
	core.AssertEqual(t, "payload", string(data))
}

func TestAX7_Node_ReadStream_Bad(t *core.T) {
	nodeTree := New()
	reader, err := nodeTree.ReadStream("missing.txt")
	core.AssertError(t, err)
	core.AssertNil(t, reader)
}

func TestAX7_Node_ReadStream_Ugly(t *core.T) {
	nodeTree := New()
	nodeTree.AddData("dir/file.txt", []byte("payload"))
	reader, err := nodeTree.ReadStream("dir")
	core.AssertNoError(t, err)
	core.AssertNoError(t, reader.Close())
}

func TestAX7_Node_WriteStream_Good(t *core.T) {
	nodeTree := New()
	writer, err := nodeTree.WriteStream("file.txt")
	core.RequireNoError(t, err)
	_, writeErr := writer.Write([]byte("payload"))
	core.AssertNoError(t, writeErr)
	core.AssertNoError(t, writer.Close())
}

func TestAX7_Node_WriteStream_Bad(t *core.T) {
	nodeTree := New()
	writer, err := nodeTree.WriteStream("")
	core.AssertNoError(t, err)
	core.AssertNotNil(t, writer)
}

func TestAX7_Node_WriteStream_Ugly(t *core.T) {
	nodeTree := New()
	writer, err := nodeTree.WriteStream("/file.txt")
	core.RequireNoError(t, err)
	core.AssertNoError(t, writer.Close())
	core.AssertTrue(t, nodeTree.Exists("file.txt"))
}

func TestAX7_Writer_Write_Good(t *core.T) {
	writer := &nodeWriter{}
	count, err := writer.Write([]byte("payload"))
	core.AssertNoError(t, err)
	core.AssertEqual(t, len("payload"), count)
}

func TestAX7_Writer_Write_Bad(t *core.T) {
	writer := &nodeWriter{}
	count, err := writer.Write(nil)
	core.AssertNoError(t, err)
	core.AssertEqual(t, 0, count)
}

func TestAX7_Writer_Write_Ugly(t *core.T) {
	writer := &nodeWriter{buffer: []byte("a")}
	count, err := writer.Write([]byte("b"))
	core.AssertNoError(t, err)
	core.AssertEqual(t, 1, count)
}

func TestAX7_Writer_Close_Good(t *core.T) {
	nodeTree := New()
	writer := &nodeWriter{node: nodeTree, path: "file.txt", buffer: []byte("payload")}
	err := writer.Close()
	core.AssertNoError(t, err)
	core.AssertTrue(t, nodeTree.Exists("file.txt"))
}

func TestAX7_Writer_Close_Bad(t *core.T) {
	nodeTree := New()
	writer := &nodeWriter{node: nodeTree, path: "", buffer: []byte("payload")}
	err := writer.Close()
	core.AssertError(t, err)
}

func TestAX7_Writer_Close_Ugly(t *core.T) {
	nodeTree := New()
	writer := &nodeWriter{node: nodeTree, path: "empty.txt"}
	err := writer.Close()
	core.AssertNoError(t, err)
	core.AssertTrue(t, nodeTree.Exists("empty.txt"))
}

func TestAX7_File_Stat_Good(t *core.T) {
	file := &dataFile{name: "file.txt", content: []byte("payload")}
	info, err := file.Stat()
	core.AssertNoError(t, err)
	core.AssertEqual(t, "file.txt", info.Name())
}

func TestAX7_File_Stat_Bad(t *core.T) {
	file := &dataFile{}
	info, err := file.Stat()
	core.AssertNoError(t, err)
	core.AssertEqual(t, ".", info.Name())
}

func TestAX7_File_Stat_Ugly(t *core.T) {
	file := &dataFile{name: "empty.txt"}
	info, err := file.Stat()
	core.AssertNoError(t, err)
	core.AssertEqual(t, int64(0), info.Size())
}

func TestAX7_File_Read_Good(t *core.T) {
	file := &dataFile{content: []byte("payload")}
	buffer := make([]byte, 1)
	count, err := file.Read(buffer)
	core.AssertErrorIs(t, err, goio.EOF)
	core.AssertEqual(t, 0, count)
}

func TestAX7_File_Read_Bad(t *core.T) {
	file := &dataFile{}
	buffer := make([]byte, 1)
	count, err := file.Read(buffer)
	core.AssertErrorIs(t, err, goio.EOF)
	core.AssertEqual(t, 0, count)
}

func TestAX7_File_Read_Ugly(t *core.T) {
	file := &dataFile{name: "file.txt"}
	count, err := file.Read(nil)
	core.AssertErrorIs(t, err, goio.EOF)
	core.AssertEqual(t, 0, count)
}

func TestAX7_File_Close_Good(t *core.T) {
	file := &dataFile{name: "file.txt"}
	err := file.Close()
	core.AssertNoError(t, err)
	core.AssertEqual(t, "file.txt", file.name)
}

func TestAX7_File_Close_Bad(t *core.T) {
	file := &dataFile{}
	err := file.Close()
	core.AssertNoError(t, err)
	core.AssertEqual(t, "", file.name)
}

func TestAX7_File_Close_Ugly(t *core.T) {
	file := &dataFile{modTime: time.Unix(1, 0)}
	err := file.Close()
	core.AssertNoError(t, err)
	core.AssertFalse(t, file.modTime.IsZero())
}

func TestAX7_FileInfo_Name_Good(t *core.T) {
	info := &dataFileInfo{file: &dataFile{name: "dir/file.txt"}}
	got := info.Name()
	core.AssertEqual(t, "file.txt", got)
}

func TestAX7_FileInfo_Name_Bad(t *core.T) {
	info := &dataFileInfo{file: &dataFile{}}
	got := info.Name()
	core.AssertEqual(t, ".", got)
}

func TestAX7_FileInfo_Name_Ugly(t *core.T) {
	info := &dataFileInfo{file: &dataFile{name: "/leading.txt"}}
	got := info.Name()
	core.AssertEqual(t, "leading.txt", got)
}

func TestAX7_FileInfo_Size_Good(t *core.T) {
	info := &dataFileInfo{file: &dataFile{content: []byte("payload")}}
	got := info.Size()
	core.AssertEqual(t, int64(7), got)
}

func TestAX7_FileInfo_Size_Bad(t *core.T) {
	info := &dataFileInfo{file: &dataFile{}}
	got := info.Size()
	core.AssertEqual(t, int64(0), got)
}

func TestAX7_FileInfo_Size_Ugly(t *core.T) {
	info := &dataFileInfo{file: &dataFile{content: nil}}
	got := info.Size()
	core.AssertEqual(t, int64(0), got)
}

func TestAX7_FileInfo_Mode_Good(t *core.T) {
	info := &dataFileInfo{file: &dataFile{}}
	got := info.Mode()
	core.AssertEqual(t, fs.FileMode(0444), got)
}

func TestAX7_FileInfo_Mode_Bad(t *core.T) {
	info := &dataFileInfo{file: &dataFile{name: "file.txt"}}
	got := info.Mode()
	core.AssertEqual(t, fs.FileMode(0444), got)
}

func TestAX7_FileInfo_Mode_Ugly(t *core.T) {
	info := &dataFileInfo{file: &dataFile{content: []byte("payload")}}
	got := info.Mode()
	core.AssertFalse(t, got.IsDir())
}

func TestAX7_FileInfo_ModTime_Good(t *core.T) {
	now := time.Unix(1, 0)
	info := &dataFileInfo{file: &dataFile{modTime: now}}
	got := info.ModTime()
	core.AssertTrue(t, got.Equal(now))
}

func TestAX7_FileInfo_ModTime_Bad(t *core.T) {
	info := &dataFileInfo{file: &dataFile{}}
	got := info.ModTime()
	core.AssertTrue(t, got.IsZero())
}

func TestAX7_FileInfo_ModTime_Ugly(t *core.T) {
	now := time.Unix(0, 1)
	info := &dataFileInfo{file: &dataFile{modTime: now}}
	got := info.ModTime()
	core.AssertEqual(t, 1, got.Nanosecond())
}

func TestAX7_FileInfo_IsDir_Good(t *core.T) {
	info := &dataFileInfo{file: &dataFile{}}
	got := info.IsDir()
	core.AssertFalse(t, got)
}

func TestAX7_FileInfo_IsDir_Bad(t *core.T) {
	info := &dataFileInfo{file: &dataFile{name: "dir"}}
	got := info.IsDir()
	core.AssertFalse(t, got)
}

func TestAX7_FileInfo_IsDir_Ugly(t *core.T) {
	info := &dataFileInfo{file: &dataFile{content: []byte("payload")}}
	got := info.IsDir()
	core.AssertFalse(t, got)
}

func TestAX7_FileInfo_Sys_Good(t *core.T) {
	info := &dataFileInfo{file: &dataFile{}}
	got := info.Sys()
	core.AssertNil(t, got)
}

func TestAX7_FileInfo_Sys_Bad(t *core.T) {
	info := &dataFileInfo{file: &dataFile{name: "file.txt"}}
	got := info.Sys()
	core.AssertNil(t, got)
}

func TestAX7_FileInfo_Sys_Ugly(t *core.T) {
	info := &dataFileInfo{file: &dataFile{content: []byte("payload")}}
	got := info.Sys()
	core.AssertNil(t, got)
}

func TestAX7_FileReader_Stat_Good(t *core.T) {
	reader := &dataFileReader{file: &dataFile{name: "file.txt", content: []byte("payload")}}
	info, err := reader.Stat()
	core.AssertNoError(t, err)
	core.AssertEqual(t, "file.txt", info.Name())
}

func TestAX7_FileReader_Stat_Bad(t *core.T) {
	reader := &dataFileReader{file: &dataFile{}}
	info, err := reader.Stat()
	core.AssertNoError(t, err)
	core.AssertEqual(t, ".", info.Name())
}

func TestAX7_FileReader_Stat_Ugly(t *core.T) {
	reader := &dataFileReader{file: &dataFile{name: "empty.txt"}}
	info, err := reader.Stat()
	core.AssertNoError(t, err)
	core.AssertEqual(t, int64(0), info.Size())
}

func TestAX7_FileReader_Read_Good(t *core.T) {
	reader := &dataFileReader{file: &dataFile{content: []byte("payload")}}
	buffer := make([]byte, 7)
	count, err := reader.Read(buffer)
	core.AssertNoError(t, err)
	core.AssertEqual(t, "payload", string(buffer[:count]))
}

func TestAX7_FileReader_Read_Bad(t *core.T) {
	reader := &dataFileReader{file: &dataFile{content: []byte("x")}, offset: 1}
	buffer := make([]byte, 1)
	count, err := reader.Read(buffer)
	core.AssertErrorIs(t, err, goio.EOF)
	core.AssertEqual(t, 0, count)
}

func TestAX7_FileReader_Read_Ugly(t *core.T) {
	reader := &dataFileReader{file: &dataFile{content: []byte("payload")}}
	buffer := make([]byte, 3)
	count, err := reader.Read(buffer)
	core.AssertNoError(t, err)
	core.AssertEqual(t, "pay", string(buffer[:count]))
}

func TestAX7_FileReader_Close_Good(t *core.T) {
	reader := &dataFileReader{file: &dataFile{name: "file.txt"}}
	err := reader.Close()
	core.AssertNoError(t, err)
	core.AssertEqual(t, "file.txt", reader.file.name)
}

func TestAX7_FileReader_Close_Bad(t *core.T) {
	reader := &dataFileReader{file: &dataFile{}}
	err := reader.Close()
	core.AssertNoError(t, err)
	core.AssertEqual(t, "", reader.file.name)
}

func TestAX7_FileReader_Close_Ugly(t *core.T) {
	reader := &dataFileReader{file: &dataFile{}, offset: 99}
	err := reader.Close()
	core.AssertNoError(t, err)
	core.AssertEqual(t, int64(99), reader.offset)
}

func TestAX7_Info_Name_Good(t *core.T) {
	info := &dirInfo{name: "dir"}
	got := info.Name()
	core.AssertEqual(t, "dir", got)
}

func TestAX7_Info_Name_Bad(t *core.T) {
	info := &dirInfo{}
	got := info.Name()
	core.AssertEqual(t, "", got)
}

func TestAX7_Info_Name_Ugly(t *core.T) {
	info := &dirInfo{name: "."}
	got := info.Name()
	core.AssertEqual(t, ".", got)
}

func TestAX7_Info_Size_Good(t *core.T) {
	info := &dirInfo{name: "dir"}
	got := info.Size()
	core.AssertEqual(t, int64(0), got)
}

func TestAX7_Info_Size_Bad(t *core.T) {
	info := &dirInfo{}
	got := info.Size()
	core.AssertEqual(t, int64(0), got)
}

func TestAX7_Info_Size_Ugly(t *core.T) {
	info := &dirInfo{name: "."}
	got := info.Size()
	core.AssertEqual(t, int64(0), got)
}

func TestAX7_Info_Mode_Good(t *core.T) {
	info := &dirInfo{name: "dir"}
	got := info.Mode()
	core.AssertTrue(t, got.IsDir())
}

func TestAX7_Info_Mode_Bad(t *core.T) {
	info := &dirInfo{}
	got := info.Mode()
	core.AssertTrue(t, got.IsDir())
}

func TestAX7_Info_Mode_Ugly(t *core.T) {
	info := &dirInfo{name: "."}
	got := info.Mode()
	core.AssertEqual(t, fs.ModeDir|0555, got)
}

func TestAX7_Info_ModTime_Good(t *core.T) {
	now := time.Unix(1, 0)
	info := &dirInfo{modTime: now}
	got := info.ModTime()
	core.AssertTrue(t, got.Equal(now))
}

func TestAX7_Info_ModTime_Bad(t *core.T) {
	info := &dirInfo{}
	got := info.ModTime()
	core.AssertTrue(t, got.IsZero())
}

func TestAX7_Info_ModTime_Ugly(t *core.T) {
	now := time.Unix(0, 1)
	info := &dirInfo{modTime: now}
	got := info.ModTime()
	core.AssertEqual(t, 1, got.Nanosecond())
}

func TestAX7_Info_IsDir_Good(t *core.T) {
	info := &dirInfo{name: "dir"}
	got := info.IsDir()
	core.AssertTrue(t, got)
}

func TestAX7_Info_IsDir_Bad(t *core.T) {
	info := &dirInfo{}
	got := info.IsDir()
	core.AssertTrue(t, got)
}

func TestAX7_Info_IsDir_Ugly(t *core.T) {
	info := &dirInfo{name: "."}
	got := info.IsDir()
	core.AssertTrue(t, got)
}

func TestAX7_Info_Sys_Good(t *core.T) {
	info := &dirInfo{name: "dir"}
	got := info.Sys()
	core.AssertNil(t, got)
}

func TestAX7_Info_Sys_Bad(t *core.T) {
	info := &dirInfo{}
	got := info.Sys()
	core.AssertNil(t, got)
}

func TestAX7_Info_Sys_Ugly(t *core.T) {
	info := &dirInfo{name: "."}
	got := info.Sys()
	core.AssertNil(t, got)
}
