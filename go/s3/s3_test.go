package s3

import (
	"context"
	core "dappco.re/go"
	"github.com/aws/aws-sdk-go-v2/aws"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	goio "io"
	"io/fs"
	"sort"
	"sync" // Note: AX-6 — internal concurrency primitive; structural per RFC §5.1
	"time"
)

const (
	s3MissingPath    = "nonexistent.txt"
	s3FilePath       = "file.txt"
	s3DeletePath     = "to-delete.txt"
	s3DirFileOnePath = "dir/file1.txt"
	s3DirFileTwoPath = "dir/file2.txt"
	s3OldPath        = "old.txt"
	s3NewPath        = "new.txt"
	s3AppendPath     = "append.txt"
	s3DirFilePath    = "dir/file.txt"
)

type testS3Client struct {
	mu                 sync.RWMutex
	objects            map[string][]byte
	mtimes             map[string]time.Time
	deleteObjectErrors map[string]error
	deleteObjectsErrs  map[string]types.Error
}

func newTestS3Client() *testS3Client {
	return &testS3Client{
		objects:            make(map[string][]byte),
		mtimes:             make(map[string]time.Time),
		deleteObjectErrors: make(map[string]error),
		deleteObjectsErrs:  make(map[string]types.Error),
	}
}

func (client *testS3Client) GetObject(operationContext context.Context, params *awss3.GetObjectInput, optionFns ...func(*awss3.Options)) (*awss3.GetObjectOutput, error) {
	client.mu.RLock()
	defer client.mu.RUnlock()

	key := aws.ToString(params.Key)
	data, ok := client.objects[key]
	if !ok {
		return nil, core.E("s3test.testS3Client.GetObject", core.Sprintf("NoSuchKey: key %q not found", key), fs.ErrNotExist)
	}
	mtime := client.mtimes[key]
	return &awss3.GetObjectOutput{
		Body:          goio.NopCloser(core.NewReader(string(data))),
		ContentLength: aws.Int64(int64(len(data))),
		LastModified:  &mtime,
	}, nil
}

func (client *testS3Client) PutObject(operationContext context.Context, params *awss3.PutObjectInput, optionFns ...func(*awss3.Options)) (*awss3.PutObjectOutput, error) {
	client.mu.Lock()
	defer client.mu.Unlock()

	key := aws.ToString(params.Key)
	data, err := goio.ReadAll(params.Body)
	if err != nil {
		return nil, err
	}
	client.objects[key] = data
	client.mtimes[key] = time.Now()
	return &awss3.PutObjectOutput{}, nil
}

func (client *testS3Client) DeleteObject(operationContext context.Context, params *awss3.DeleteObjectInput, optionFns ...func(*awss3.Options)) (*awss3.DeleteObjectOutput, error) {
	client.mu.Lock()
	defer client.mu.Unlock()

	key := aws.ToString(params.Key)
	if err, ok := client.deleteObjectErrors[key]; ok {
		return nil, err
	}
	delete(client.objects, key)
	delete(client.mtimes, key)
	return &awss3.DeleteObjectOutput{}, nil
}

func (client *testS3Client) DeleteObjects(operationContext context.Context, params *awss3.DeleteObjectsInput, optionFns ...func(*awss3.Options)) (*awss3.DeleteObjectsOutput, error) {
	client.mu.Lock()
	defer client.mu.Unlock()

	var outErrs []types.Error
	for _, obj := range params.Delete.Objects {
		key := aws.ToString(obj.Key)
		if errInfo, ok := client.deleteObjectsErrs[key]; ok {
			outErrs = append(outErrs, errInfo)
			continue
		}
		delete(client.objects, key)
		delete(client.mtimes, key)
	}
	return &awss3.DeleteObjectsOutput{Errors: outErrs}, nil
}

func (client *testS3Client) HeadObject(operationContext context.Context, params *awss3.HeadObjectInput, optionFns ...func(*awss3.Options)) (*awss3.HeadObjectOutput, error) {
	client.mu.RLock()
	defer client.mu.RUnlock()

	key := aws.ToString(params.Key)
	data, ok := client.objects[key]
	if !ok {
		return nil, core.E("s3test.testS3Client.HeadObject", core.Sprintf("NotFound: key %q not found", key), fs.ErrNotExist)
	}
	mtime := client.mtimes[key]
	return &awss3.HeadObjectOutput{
		ContentLength: aws.Int64(int64(len(data))),
		LastModified:  &mtime,
	}, nil
}

func (client *testS3Client) ListObjectsV2(operationContext context.Context, params *awss3.ListObjectsV2Input, optionFns ...func(*awss3.Options)) (*awss3.ListObjectsV2Output, error) {
	client.mu.RLock()
	defer client.mu.RUnlock()

	prefix := aws.ToString(params.Prefix)
	delimiter := aws.ToString(params.Delimiter)
	maxKeys := int32(1000)
	if params.MaxKeys != nil {
		maxKeys = *params.MaxKeys
	}

	allKeys := client.matchingObjectKeys(prefix)
	contents, commonPrefixes, truncated, nextToken := client.listObjectPage(
		allKeys,
		prefix,
		delimiter,
		aws.ToString(params.ContinuationToken),
		maxKeys,
	)

	out := &awss3.ListObjectsV2Output{
		Contents:       contents,
		CommonPrefixes: commonPrefixSlice(commonPrefixes),
		IsTruncated:    aws.Bool(truncated),
	}
	if truncated {
		out.NextContinuationToken = aws.String(nextToken)
	}
	return out, nil
}

func (client *testS3Client) matchingObjectKeys(prefix string) []string {
	var allKeys []string
	for key := range client.objects {
		if core.HasPrefix(key, prefix) {
			allKeys = append(allKeys, key)
		}
	}
	sort.Strings(allKeys)
	return allKeys
}

func (client *testS3Client) listObjectPage(
	allKeys []string,
	prefix string,
	delimiter string,
	continuationToken string,
	maxKeys int32,
) ([]types.Object, map[string]bool, bool, string) {
	var contents []types.Object
	commonPrefixes := make(map[string]bool)
	pastContinuation := continuationToken == ""
	for _, key := range allKeys {
		if !pastContinuation {
			pastContinuation = key == continuationToken
			continue
		}
		if addCommonPrefix(key, prefix, delimiter, commonPrefixes) {
			continue
		}
		if int32(len(contents)) >= maxKeys {
			return contents, commonPrefixes, true, key
		}
		contents = append(contents, client.objectForKey(key))
	}
	return contents, commonPrefixes, false, ""
}

func addCommonPrefix(key, prefix, delimiter string, commonPrefixes map[string]bool) bool {
	if delimiter == "" {
		return false
	}
	parts := core.SplitN(core.TrimPrefix(key, prefix), delimiter, 2)
	if len(parts) != 2 {
		return false
	}
	commonPrefixes[core.Concat(prefix, parts[0], delimiter)] = true
	return true
}

func (client *testS3Client) objectForKey(key string) types.Object {
	data := client.objects[key]
	mtime := client.mtimes[key]
	return types.Object{
		Key:          aws.String(key),
		Size:         aws.Int64(int64(len(data))),
		LastModified: &mtime,
	}
}

func commonPrefixSlice(commonPrefixes map[string]bool) []types.CommonPrefix {
	var keys []string
	for commonPrefix := range commonPrefixes {
		keys = append(keys, commonPrefix)
	}
	sort.Strings(keys)
	prefixes := make([]types.CommonPrefix, 0, len(keys))
	for _, commonPrefix := range keys {
		prefixes = append(prefixes, types.CommonPrefix{Prefix: aws.String(commonPrefix)})
	}
	return prefixes
}

func (client *testS3Client) CopyObject(operationContext context.Context, params *awss3.CopyObjectInput, optionFns ...func(*awss3.Options)) (*awss3.CopyObjectOutput, error) {
	client.mu.Lock()
	defer client.mu.Unlock()

	source := aws.ToString(params.CopySource)
	parts := core.SplitN(source, "/", 2)
	if len(parts) != 2 {
		return nil, core.E("s3test.testS3Client.CopyObject", core.Sprintf("invalid CopySource: %s", source), fs.ErrInvalid)
	}
	srcKey := parts[1]

	data, ok := client.objects[srcKey]
	if !ok {
		return nil, core.E("s3test.testS3Client.CopyObject", core.Sprintf("NoSuchKey: source key %q not found", srcKey), fs.ErrNotExist)
	}

	destKey := aws.ToString(params.Key)
	client.objects[destKey] = append([]byte{}, data...)
	client.mtimes[destKey] = time.Now()

	return &awss3.CopyObjectOutput{}, nil
}

func newS3Medium(t *core.T) (*Medium, *testS3Client) {
	t.Helper()
	testS3Client := newTestS3Client()
	s3Medium, err := New(Options{Bucket: "test-bucket", Client: testS3Client})
	core.RequireNoError(t, err)
	return s3Medium, testS3Client
}

func TestS3_New_Good(t *core.T) {
	testS3Client := newTestS3Client()
	s3Medium, err := New(Options{Bucket: "my-bucket", Client: testS3Client})
	core.RequireNoError(t, err)
	core.AssertEqual(t, "my-bucket", s3Medium.bucket)
	core.AssertEqual(t, "", s3Medium.prefix)
}

func TestS3_New_NoBucket_Bad(t *core.T) {
	_, err := New(Options{Client: newTestS3Client()})
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "bucket name is required")
}

func TestS3_New_NoClient_Bad(t *core.T) {
	_, err := New(Options{Bucket: "bucket"})
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "client is required")
}

func TestS3_New_Options_Good(t *core.T) {
	testS3Client := newTestS3Client()
	s3Medium, err := New(Options{Bucket: "bucket", Client: testS3Client, Prefix: "data/"})
	core.RequireNoError(t, err)
	core.AssertEqual(t, "data/", s3Medium.prefix)

	prefixedS3Medium, err := New(Options{Bucket: "bucket", Client: testS3Client, Prefix: "data"})
	core.RequireNoError(t, err)
	core.AssertEqual(t, "data/", prefixedS3Medium.prefix)
}

func TestS3_ReadWriteGood(t *core.T) {
	s3Medium, _ := newS3Medium(t)

	err := s3Medium.Write("hello.txt", "world")
	core.RequireNoError(t, err)

	content, err := s3Medium.Read("hello.txt")
	core.RequireNoError(t, err)
	core.AssertEqual(t, "world", content)
}

func TestS3_ReadWrite_NotFoundBad(t *core.T) {
	s3Medium, _ := newS3Medium(t)

	_, err := s3Medium.Read(s3MissingPath)
	core.AssertError(t, err)
}

func TestS3_ReadWrite_EmptyPathBad(t *core.T) {
	s3Medium, _ := newS3Medium(t)

	_, err := s3Medium.Read("")
	core.AssertError(t, err)

	err = s3Medium.Write("", "content")
	core.AssertError(t, err)
}

func TestS3_ReadWrite_Prefix_Good(t *core.T) {
	testS3Client := newTestS3Client()
	s3Medium, err := New(Options{Bucket: "bucket", Client: testS3Client, Prefix: "pfx"})
	core.RequireNoError(t, err)

	err = s3Medium.Write(s3FilePath, "data")
	core.RequireNoError(t, err)

	_, ok := testS3Client.objects["pfx/file.txt"]
	core.AssertTrue(t, ok, "object should be stored with prefix")

	content, err := s3Medium.Read(s3FilePath)
	core.RequireNoError(t, err)
	core.AssertEqual(t, "data", content)
}

func TestS3_EnsureDir_Good(t *core.T) {
	medium, _ := newS3Medium(t)
	err := medium.EnsureDir("any/path")
	core.AssertNoError(t, err)
}

func TestS3_IsFile_Good(t *core.T) {
	s3Medium, _ := newS3Medium(t)

	err := s3Medium.Write(s3FilePath, "content")
	core.RequireNoError(t, err)

	core.AssertTrue(t, s3Medium.IsFile(s3FilePath))
	core.AssertFalse(t, s3Medium.IsFile(s3MissingPath))
	core.AssertFalse(t, s3Medium.IsFile(""))
}

func TestS3_Delete_Good(t *core.T) {
	s3Medium, _ := newS3Medium(t)

	err := s3Medium.Write(s3DeletePath, "content")
	core.RequireNoError(t, err)
	core.AssertTrue(t, s3Medium.Exists(s3DeletePath))

	err = s3Medium.Delete(s3DeletePath)
	core.RequireNoError(t, err)
	core.AssertFalse(t, s3Medium.IsFile(s3DeletePath))
}

func TestS3_Delete_EmptyPath_Bad(t *core.T) {
	s3Medium, _ := newS3Medium(t)
	err := s3Medium.Delete("")
	core.AssertError(t, err)
}

func TestS3_DeleteAll_Good(t *core.T) {
	s3Medium, _ := newS3Medium(t)

	core.RequireNoError(t, s3Medium.Write(s3DirFileOnePath, "a"))
	core.RequireNoError(t, s3Medium.Write("dir/sub/file2.txt", "b"))
	core.RequireNoError(t, s3Medium.Write("other.txt", "c"))

	err := s3Medium.DeleteAll("dir")
	core.RequireNoError(t, err)

	core.AssertFalse(t, s3Medium.IsFile(s3DirFileOnePath))
	core.AssertFalse(t, s3Medium.IsFile("dir/sub/file2.txt"))
	core.AssertTrue(t, s3Medium.IsFile("other.txt"))
}

func TestS3_DeleteAll_EmptyPath_Bad(t *core.T) {
	s3Medium, _ := newS3Medium(t)
	err := s3Medium.DeleteAll("")
	core.AssertError(t, err)
}

func TestS3_DeleteAll_DeleteObjectError_Bad(t *core.T) {
	s3Medium, testS3Client := newS3Medium(t)
	testS3Client.deleteObjectErrors["dir"] = core.NewError("boom")

	err := s3Medium.DeleteAll("dir")
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "failed to delete object: dir")
}

func TestS3_DeleteAll_PartialDelete_Bad(t *core.T) {
	s3Medium, testS3Client := newS3Medium(t)

	core.RequireNoError(t, s3Medium.Write(s3DirFileOnePath, "a"))
	core.RequireNoError(t, s3Medium.Write(s3DirFileTwoPath, "b"))
	testS3Client.deleteObjectsErrs[s3DirFileTwoPath] = types.Error{
		Key:     aws.String(s3DirFileTwoPath),
		Code:    aws.String("AccessDenied"),
		Message: aws.String("blocked"),
	}

	err := s3Medium.DeleteAll("dir")
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "partial delete failed")
	core.AssertContains(t, err.Error(), s3DirFileTwoPath)
	core.AssertTrue(t, s3Medium.IsFile(s3DirFileTwoPath))
	core.AssertFalse(t, s3Medium.IsFile(s3DirFileOnePath))
}

func TestS3_Rename_Good(t *core.T) {
	s3Medium, _ := newS3Medium(t)

	core.RequireNoError(t, s3Medium.Write(s3OldPath, "content"))
	core.AssertTrue(t, s3Medium.IsFile(s3OldPath))

	err := s3Medium.Rename(s3OldPath, s3NewPath)
	core.RequireNoError(t, err)

	core.AssertFalse(t, s3Medium.IsFile(s3OldPath))
	core.AssertTrue(t, s3Medium.IsFile(s3NewPath))

	content, err := s3Medium.Read(s3NewPath)
	core.RequireNoError(t, err)
	core.AssertEqual(t, "content", content)
}

func TestS3_Rename_EmptyPath_Bad(t *core.T) {
	s3Medium, _ := newS3Medium(t)
	err := s3Medium.Rename("", s3NewPath)
	core.AssertError(t, err)

	err = s3Medium.Rename(s3OldPath, "")
	core.AssertError(t, err)
}

func TestS3_Rename_SourceNotFound_Bad(t *core.T) {
	s3Medium, _ := newS3Medium(t)
	err := s3Medium.Rename(s3MissingPath, s3NewPath)
	core.AssertError(t, err)
}

func TestS3_List_Good(t *core.T) {
	s3Medium, _ := newS3Medium(t)

	core.RequireNoError(t, s3Medium.Write(s3DirFileOnePath, "a"))
	core.RequireNoError(t, s3Medium.Write(s3DirFileTwoPath, "b"))
	core.RequireNoError(t, s3Medium.Write("dir/sub/file3.txt", "c"))

	entries, err := s3Medium.List("dir")
	core.RequireNoError(t, err)

	names := make(map[string]bool)
	for _, entry := range entries {
		names[entry.Name()] = true
	}

	core.AssertTrue(t, names["file1.txt"], "should list file1.txt")
	core.AssertTrue(t, names["file2.txt"], "should list file2.txt")
	core.AssertTrue(t, names["sub"], "should list sub directory")
	core.AssertLen(t, entries, 3)

	for _, entry := range entries {
		if entry.Name() == "sub" {
			core.AssertTrue(t, entry.IsDir())
			info, err := entry.Info()
			core.RequireNoError(t, err)
			core.AssertTrue(t, info.IsDir())
		}
	}
}

func TestS3_List_Root_Good(t *core.T) {
	s3Medium, _ := newS3Medium(t)

	core.RequireNoError(t, s3Medium.Write("root.txt", "content"))
	core.RequireNoError(t, s3Medium.Write("dir/nested.txt", "nested"))

	entries, err := s3Medium.List("")
	core.RequireNoError(t, err)

	names := make(map[string]bool)
	for _, entry := range entries {
		names[entry.Name()] = true
	}

	core.AssertTrue(t, names["root.txt"])
	core.AssertTrue(t, names["dir"])
}

func TestS3_Stat_Good(t *core.T) {
	s3Medium, _ := newS3Medium(t)

	core.RequireNoError(t, s3Medium.Write(s3FilePath, "hello world"))

	info, err := s3Medium.Stat(s3FilePath)
	core.RequireNoError(t, err)
	core.AssertEqual(t, s3FilePath, info.Name())
	core.AssertEqual(t, int64(11), info.Size())
	core.AssertFalse(t, info.IsDir())
}

func TestS3_Stat_NotFound_Bad(t *core.T) {
	s3Medium, _ := newS3Medium(t)

	_, err := s3Medium.Stat(s3MissingPath)
	core.AssertError(t, err)
}

func TestS3_Stat_EmptyPath_Bad(t *core.T) {
	s3Medium, _ := newS3Medium(t)
	_, err := s3Medium.Stat("")
	core.AssertError(t, err)
}

func TestS3_Open_Good(t *core.T) {
	s3Medium, _ := newS3Medium(t)

	core.RequireNoError(t, s3Medium.Write(s3FilePath, "open me"))

	file, err := s3Medium.Open(s3FilePath)
	core.RequireNoError(t, err)
	defer func() { _ = file.Close() }()

	data, err := goio.ReadAll(file.(goio.Reader))
	core.RequireNoError(t, err)
	core.AssertEqual(t, "open me", string(data))

	stat, err := file.Stat()
	core.RequireNoError(t, err)
	core.AssertEqual(t, s3FilePath, stat.Name())
}

func TestS3_Open_NotFound_Bad(t *core.T) {
	s3Medium, _ := newS3Medium(t)

	_, err := s3Medium.Open(s3MissingPath)
	core.AssertError(t, err)
}

func TestS3_Create_Good(t *core.T) {
	s3Medium, _ := newS3Medium(t)

	writer, err := s3Medium.Create(s3NewPath)
	core.RequireNoError(t, err)

	bytesWritten, err := writer.Write([]byte("created"))
	core.RequireNoError(t, err)
	core.AssertEqual(t, 7, bytesWritten)

	err = writer.Close()
	core.RequireNoError(t, err)

	content, err := s3Medium.Read(s3NewPath)
	core.RequireNoError(t, err)
	core.AssertEqual(t, "created", content)
}

func TestS3_Append_Good(t *core.T) {
	s3Medium, _ := newS3Medium(t)

	core.RequireNoError(t, s3Medium.Write(s3AppendPath, "hello"))

	writer, err := s3Medium.Append(s3AppendPath)
	core.RequireNoError(t, err)

	_, err = writer.Write([]byte(" world"))
	core.RequireNoError(t, err)
	err = writer.Close()
	core.RequireNoError(t, err)

	content, err := s3Medium.Read(s3AppendPath)
	core.RequireNoError(t, err)
	core.AssertEqual(t, "hello world", content)
}

func TestS3_Append_NewFile_Good(t *core.T) {
	s3Medium, _ := newS3Medium(t)

	writer, err := s3Medium.Append(s3NewPath)
	core.RequireNoError(t, err)

	_, err = writer.Write([]byte("fresh"))
	core.RequireNoError(t, err)
	err = writer.Close()
	core.RequireNoError(t, err)

	content, err := s3Medium.Read(s3NewPath)
	core.RequireNoError(t, err)
	core.AssertEqual(t, "fresh", content)
}

func TestS3_ReadStream_Good(t *core.T) {
	s3Medium, _ := newS3Medium(t)

	core.RequireNoError(t, s3Medium.Write("stream.txt", "streaming content"))

	reader, err := s3Medium.ReadStream("stream.txt")
	core.RequireNoError(t, err)
	defer func() { _ = reader.Close() }()

	data, err := goio.ReadAll(reader)
	core.RequireNoError(t, err)
	core.AssertEqual(t, "streaming content", string(data))
}

func TestS3_ReadStream_NotFound_Bad(t *core.T) {
	s3Medium, _ := newS3Medium(t)
	_, err := s3Medium.ReadStream(s3MissingPath)
	core.AssertError(t, err)
}

func TestS3_WriteStream_Good(t *core.T) {
	s3Medium, _ := newS3Medium(t)

	writer, err := s3Medium.WriteStream("output.txt")
	core.RequireNoError(t, err)

	_, err = goio.Copy(writer, core.NewReader("piped data"))
	core.RequireNoError(t, err)
	err = writer.Close()
	core.RequireNoError(t, err)

	content, err := s3Medium.Read("output.txt")
	core.RequireNoError(t, err)
	core.AssertEqual(t, "piped data", content)
}

func TestS3_Exists_Good(t *core.T) {
	s3Medium, _ := newS3Medium(t)

	core.AssertFalse(t, s3Medium.Exists(s3MissingPath))

	core.RequireNoError(t, s3Medium.Write(s3FilePath, "content"))
	core.AssertTrue(t, s3Medium.Exists(s3FilePath))
}

func TestS3_Exists_DirectoryPrefix_Good(t *core.T) {
	s3Medium, _ := newS3Medium(t)

	core.RequireNoError(t, s3Medium.Write(s3DirFilePath, "content"))
	core.AssertTrue(t, s3Medium.Exists("dir"))
}

func TestS3_IsDir_Good(t *core.T) {
	s3Medium, _ := newS3Medium(t)

	core.RequireNoError(t, s3Medium.Write(s3DirFilePath, "content"))

	core.AssertTrue(t, s3Medium.IsDir("dir"))
	core.AssertFalse(t, s3Medium.IsDir(s3DirFilePath))
	core.AssertFalse(t, s3Medium.IsDir("nonexistent"))
	core.AssertFalse(t, s3Medium.IsDir(""))
}

func TestS3_ObjectKeyGood(t *core.T) {
	testS3Client := newTestS3Client()

	s3Medium, _ := New(Options{Bucket: "bucket", Client: testS3Client})
	core.AssertEqual(t, s3FilePath, s3Medium.objectKey(s3FilePath))
	core.AssertEqual(t, s3DirFilePath, s3Medium.objectKey(s3DirFilePath))
	core.AssertEqual(t, "", s3Medium.objectKey(""))
	core.AssertEqual(t, s3FilePath, s3Medium.objectKey("/file.txt"))
	core.AssertEqual(t, s3FilePath, s3Medium.objectKey("../file.txt"))

	prefixedS3Medium, _ := New(Options{Bucket: "bucket", Client: testS3Client, Prefix: "pfx"})
	core.AssertEqual(t, "pfx/file.txt", prefixedS3Medium.objectKey(s3FilePath))
	core.AssertEqual(t, "pfx/dir/file.txt", prefixedS3Medium.objectKey(s3DirFilePath))
	core.AssertEqual(t, "pfx/", prefixedS3Medium.objectKey(""))
}

func TestS3_InterfaceComplianceGood(t *core.T) {
	testS3Client := newTestS3Client()
	s3Medium, err := New(Options{Bucket: "bucket", Client: testS3Client})
	core.RequireNoError(t, err)

	var _ interface {
		Read(string) (string, error)
		Write(string, string) error
		EnsureDir(string) error
		IsFile(string) bool
		Delete(string) error
		DeleteAll(string) error
		Rename(string, string) error
		List(string) ([]fs.DirEntry, error)
		Stat(string) (fs.FileInfo, error)
		Open(string) (fs.File, error)
		Create(string) (goio.WriteCloser, error)
		Append(string) (goio.WriteCloser, error)
		ReadStream(string) (goio.ReadCloser, error)
		WriteStream(string) (goio.WriteCloser, error)
		Exists(string) bool
		IsDir(string) bool
	} = s3Medium
}

func newS3MediumFixture(t *core.T) *Medium {
	t.Helper()
	medium, _ := newS3Medium(t)
	return medium
}

func TestS3_New_Bad(t *core.T) {
	medium, err := New(Options{Client: newTestS3Client()})
	core.AssertError(t, err)
	core.AssertNil(t, medium)
}

func TestS3_New_Ugly(t *core.T) {
	client := newTestS3Client()
	medium, err := New(Options{Bucket: "bucket", Client: client, Prefix: "//nested//"})
	core.AssertNoError(t, err)
	core.AssertEqual(t, "nested/", medium.prefix)
}

func TestS3_Medium_Read_Good(t *core.T) {
	medium := newS3MediumFixture(t)
	core.RequireNoError(t, medium.Write("read.txt", "payload"))
	got, err := medium.Read("read.txt")
	core.AssertNoError(t, err)
	core.AssertEqual(t, "payload", got)
}

func TestS3_Medium_Read_Bad(t *core.T) {
	medium := newS3MediumFixture(t)
	got, err := medium.Read("missing.txt")
	core.AssertError(t, err)
	core.AssertEqual(t, "", got)
}

func TestS3_Medium_Read_Ugly(t *core.T) {
	medium := newS3MediumFixture(t)
	got, err := medium.Read("")
	core.AssertError(t, err)
	core.AssertEqual(t, "", got)
}

func TestS3_Medium_Write_Good(t *core.T) {
	medium := newS3MediumFixture(t)
	err := medium.Write("write.txt", "payload")
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.IsFile("write.txt"))
}

func TestS3_Medium_Write_Bad(t *core.T) {
	medium := newS3MediumFixture(t)
	err := medium.Write("", "payload")
	core.AssertError(t, err)
	core.AssertFalse(t, medium.IsFile(""))
}

func TestS3_Medium_Write_Ugly(t *core.T) {
	medium := newS3MediumFixture(t)
	err := medium.Write("nested/write.txt", "payload")
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.IsFile("nested/write.txt"))
}

func TestS3_Medium_WriteMode_Good(t *core.T) {
	medium := newS3MediumFixture(t)
	err := medium.WriteMode("mode.txt", "payload", 0600)
	info, statErr := medium.Stat("mode.txt")
	core.AssertNoError(t, err)
	core.AssertNoError(t, statErr)
	core.AssertEqual(t, fs.FileMode(0644), info.Mode().Perm())
}

func TestS3_Medium_WriteMode_Bad(t *core.T) {
	medium := newS3MediumFixture(t)
	err := medium.WriteMode("", "payload", 0600)
	core.AssertError(t, err)
	core.AssertFalse(t, medium.IsFile(""))
}

func TestS3_Medium_WriteMode_Ugly(t *core.T) {
	medium := newS3MediumFixture(t)
	err := medium.WriteMode("zero-mode.txt", "payload", 0)
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.IsFile("zero-mode.txt"))
}

func TestS3_Medium_EnsureDir_Good(t *core.T) {
	medium := newS3MediumFixture(t)
	err := medium.EnsureDir("dir")
	core.AssertNoError(t, err)
	core.AssertFalse(t, medium.IsDir("dir"))
}

func TestS3_Medium_EnsureDir_Bad(t *core.T) {
	medium := newS3MediumFixture(t)
	core.RequireNoError(t, medium.Write("file", "payload"))
	err := medium.EnsureDir("file")
	core.AssertNoError(t, err)
}

func TestS3_Medium_EnsureDir_Ugly(t *core.T) {
	medium := newS3MediumFixture(t)
	err := medium.EnsureDir("a/b/c")
	core.AssertNoError(t, err)
	core.AssertFalse(t, medium.IsDir("a/b/c"))
}

func TestS3_Medium_IsFile_Good(t *core.T) {
	medium := newS3MediumFixture(t)
	core.RequireNoError(t, medium.Write(s3FilePath, "payload"))
	got := medium.IsFile(s3FilePath)
	core.AssertTrue(t, got)
}

func TestS3_Medium_IsFile_Bad(t *core.T) {
	medium := newS3MediumFixture(t)
	got := medium.IsFile("missing.txt")
	core.AssertFalse(t, got)
}

func TestS3_Medium_IsFile_Ugly(t *core.T) {
	medium := newS3MediumFixture(t)
	core.RequireNoError(t, medium.EnsureDir("dir"))
	got := medium.IsFile("dir")
	core.AssertFalse(t, got)
}

func TestS3_Medium_Delete_Good(t *core.T) {
	medium := newS3MediumFixture(t)
	core.RequireNoError(t, medium.Write("delete.txt", "payload"))
	err := medium.Delete("delete.txt")
	core.AssertNoError(t, err)
	core.AssertFalse(t, medium.Exists("delete.txt"))
}

func TestS3_Medium_Delete_Bad(t *core.T) {
	medium := newS3MediumFixture(t)
	err := medium.Delete("missing.txt")
	core.AssertNoError(t, err)
	core.AssertFalse(t, medium.Exists("missing.txt"))
}

func TestS3_Medium_Delete_Ugly(t *core.T) {
	medium := newS3MediumFixture(t)
	err := medium.Delete("")
	core.AssertError(t, err)
	core.AssertFalse(t, medium.Exists(""))
}

func TestS3_Medium_DeleteAll_Good(t *core.T) {
	medium := newS3MediumFixture(t)
	core.RequireNoError(t, medium.Write("tree/file.txt", "payload"))
	err := medium.DeleteAll("tree")
	core.AssertNoError(t, err)
	core.AssertFalse(t, medium.Exists("tree/file.txt"))
}

func TestS3_Medium_DeleteAll_Bad(t *core.T) {
	medium := newS3MediumFixture(t)
	err := medium.DeleteAll("missing")
	core.AssertNoError(t, err)
	core.AssertFalse(t, medium.Exists("missing"))
}

func TestS3_Medium_DeleteAll_Ugly(t *core.T) {
	medium := newS3MediumFixture(t)
	err := medium.DeleteAll("")
	core.AssertError(t, err)
	core.AssertFalse(t, medium.Exists(""))
}

func TestS3_Medium_Rename_Good(t *core.T) {
	medium := newS3MediumFixture(t)
	core.RequireNoError(t, medium.Write(s3OldPath, "payload"))
	err := medium.Rename(s3OldPath, s3NewPath)
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.IsFile(s3NewPath))
}

func TestS3_Medium_Rename_Bad(t *core.T) {
	medium := newS3MediumFixture(t)
	err := medium.Rename("missing.txt", s3NewPath)
	core.AssertError(t, err)
	core.AssertFalse(t, medium.Exists(s3NewPath))
}

func TestS3_Medium_Rename_Ugly(t *core.T) {
	medium := newS3MediumFixture(t)
	core.RequireNoError(t, medium.Write(s3OldPath, "payload"))
	err := medium.Rename(s3OldPath, "nested/new.txt")
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.IsFile("nested/new.txt"))
}

func TestS3_Medium_List_Good(t *core.T) {
	medium := newS3MediumFixture(t)
	core.RequireNoError(t, medium.Write("dir/a.txt", "payload"))
	entries, err := medium.List("dir")
	core.AssertNoError(t, err)
	core.AssertLen(t, entries, 1)
}

func TestS3_Medium_List_Bad(t *core.T) {
	medium := newS3MediumFixture(t)
	entries, err := medium.List("missing")
	core.AssertNoError(t, err)
	core.AssertNil(t, entries)
}

func TestS3_Medium_List_Ugly(t *core.T) {
	medium := newS3MediumFixture(t)
	entries, err := medium.List("")
	core.AssertNoError(t, err)
	core.AssertNil(t, entries)
}

func TestS3_Medium_Stat_Good(t *core.T) {
	medium := newS3MediumFixture(t)
	core.RequireNoError(t, medium.Write("stat.txt", "payload"))
	info, err := medium.Stat("stat.txt")
	core.AssertNoError(t, err)
	core.AssertEqual(t, "stat.txt", info.Name())
}

func TestS3_Medium_Stat_Bad(t *core.T) {
	medium := newS3MediumFixture(t)
	info, err := medium.Stat("missing.txt")
	core.AssertError(t, err)
	core.AssertNil(t, info)
}

func TestS3_Medium_Stat_Ugly(t *core.T) {
	medium := newS3MediumFixture(t)
	info, err := medium.Stat("")
	core.AssertError(t, err)
	core.AssertNil(t, info)
}

func TestS3_Medium_Open_Good(t *core.T) {
	medium := newS3MediumFixture(t)
	core.RequireNoError(t, medium.Write("open.txt", "payload"))
	file, err := medium.Open("open.txt")
	core.AssertNoError(t, err)
	core.AssertNotNil(t, file)
	core.RequireNoError(t, file.Close())
}

func TestS3_Medium_Open_Bad(t *core.T) {
	medium := newS3MediumFixture(t)
	file, err := medium.Open("missing.txt")
	core.AssertError(t, err)
	core.AssertNil(t, file)
}

func TestS3_Medium_Open_Ugly(t *core.T) {
	medium := newS3MediumFixture(t)
	file, err := medium.Open("")
	core.AssertError(t, err)
	core.AssertNil(t, file)
}

func TestS3_Medium_Create_Good(t *core.T) {
	medium := newS3MediumFixture(t)
	writer, err := medium.Create("create.txt")
	core.RequireNoError(t, err)
	_, writeErr := writer.Write([]byte("payload"))
	core.AssertNoError(t, writeErr)
	core.AssertNoError(t, writer.Close())
}

func TestS3_Medium_Create_Bad(t *core.T) {
	medium := newS3MediumFixture(t)
	writer, err := medium.Create("")
	core.AssertError(t, err)
	core.AssertNil(t, writer)
}

func TestS3_Medium_Create_Ugly(t *core.T) {
	medium := newS3MediumFixture(t)
	writer, err := medium.Create("nested/create.txt")
	core.RequireNoError(t, err)
	_, writeErr := writer.Write([]byte("payload"))
	core.AssertNoError(t, writeErr)
	core.AssertNoError(t, writer.Close())
}

func TestS3_Medium_Append_Good(t *core.T) {
	medium := newS3MediumFixture(t)
	core.RequireNoError(t, medium.Write(s3AppendPath, "a"))
	writer, err := medium.Append(s3AppendPath)
	core.RequireNoError(t, err)
	_, writeErr := writer.Write([]byte("b"))
	core.AssertNoError(t, writeErr)
	core.AssertNoError(t, writer.Close())
}

func TestS3_Medium_Append_Bad(t *core.T) {
	medium := newS3MediumFixture(t)
	writer, err := medium.Append("")
	core.AssertError(t, err)
	core.AssertNil(t, writer)
}

func TestS3_Medium_Append_Ugly(t *core.T) {
	medium := newS3MediumFixture(t)
	writer, err := medium.Append(s3NewPath)
	core.RequireNoError(t, err)
	_, writeErr := writer.Write([]byte("new"))
	core.AssertNoError(t, writeErr)
	core.AssertNoError(t, writer.Close())
}

func TestS3_Medium_ReadStream_Good(t *core.T) {
	medium := newS3MediumFixture(t)
	core.RequireNoError(t, medium.Write("stream.txt", "payload"))
	reader, err := medium.ReadStream("stream.txt")
	core.RequireNoError(t, err)
	data, readErr := goio.ReadAll(reader)
	core.AssertNoError(t, readErr)
	core.AssertEqual(t, "payload", string(data))
}

func TestS3_Medium_ReadStream_Bad(t *core.T) {
	medium := newS3MediumFixture(t)
	reader, err := medium.ReadStream("missing.txt")
	core.AssertError(t, err)
	core.AssertNil(t, reader)
}

func TestS3_Medium_ReadStream_Ugly(t *core.T) {
	medium := newS3MediumFixture(t)
	reader, err := medium.ReadStream("")
	core.AssertError(t, err)
	core.AssertNil(t, reader)
}

func TestS3_Medium_WriteStream_Good(t *core.T) {
	medium := newS3MediumFixture(t)
	writer, err := medium.WriteStream("stream.txt")
	core.RequireNoError(t, err)
	_, writeErr := writer.Write([]byte("payload"))
	core.AssertNoError(t, writeErr)
	core.AssertNoError(t, writer.Close())
}

func TestS3_Medium_WriteStream_Bad(t *core.T) {
	medium := newS3MediumFixture(t)
	writer, err := medium.WriteStream("")
	core.AssertError(t, err)
	core.AssertNil(t, writer)
}

func TestS3_Medium_WriteStream_Ugly(t *core.T) {
	medium := newS3MediumFixture(t)
	writer, err := medium.WriteStream("nested/stream.txt")
	core.RequireNoError(t, err)
	_, writeErr := writer.Write([]byte("payload"))
	core.AssertNoError(t, writeErr)
	core.AssertNoError(t, writer.Close())
}

func TestS3_Medium_Exists_Good(t *core.T) {
	medium := newS3MediumFixture(t)
	core.RequireNoError(t, medium.Write("exists.txt", "payload"))
	got := medium.Exists("exists.txt")
	core.AssertTrue(t, got)
}

func TestS3_Medium_Exists_Bad(t *core.T) {
	medium := newS3MediumFixture(t)
	got := medium.Exists("missing.txt")
	core.AssertFalse(t, got)
}

func TestS3_Medium_Exists_Ugly(t *core.T) {
	medium := newS3MediumFixture(t)
	got := medium.Exists("")
	core.AssertFalse(t, got)
}

func TestS3_Medium_IsDir_Good(t *core.T) {
	medium := newS3MediumFixture(t)
	core.RequireNoError(t, medium.EnsureDir("dir"))
	got := medium.IsDir("dir")
	core.AssertFalse(t, got)
}

func TestS3_Medium_IsDir_Bad(t *core.T) {
	medium := newS3MediumFixture(t)
	got := medium.IsDir("missing")
	core.AssertFalse(t, got)
}

func TestS3_Medium_IsDir_Ugly(t *core.T) {
	medium := newS3MediumFixture(t)
	got := medium.IsDir("")
	core.AssertFalse(t, got)
}

func TestS3_Info_Name_Good(t *core.T) {
	medium := newS3MediumFixture(t)
	core.RequireNoError(t, medium.Write("info.txt", "abc"))
	info, err := medium.Stat("info.txt")
	core.RequireNoError(t, err)
	core.AssertEqual(t, "info.txt", info.Name())
}

func TestS3_Info_Name_Bad(t *core.T) {
	info := &fileInfo{}
	got := info.Name()
	core.AssertEqual(t, "", got)
}

func TestS3_Info_Name_Ugly(t *core.T) {
	info := &fileInfo{name: ""}
	got := info.Name()
	core.AssertEqual(t, "", got)
}

func TestS3_Info_Size_Good(t *core.T) {
	medium := newS3MediumFixture(t)
	core.RequireNoError(t, medium.Write("size.txt", "abcd"))
	info, err := medium.Stat("size.txt")
	core.RequireNoError(t, err)
	core.AssertEqual(t, int64(4), info.Size())
}

func TestS3_Info_Size_Bad(t *core.T) {
	info := &fileInfo{}
	got := info.Size()
	core.AssertEqual(t, int64(0), got)
}

func TestS3_Info_Size_Ugly(t *core.T) {
	info := &fileInfo{size: -1}
	got := info.Size()
	core.AssertEqual(t, int64(-1), got)
}

func TestS3_Info_Mode_Good(t *core.T) {
	medium := newS3MediumFixture(t)
	core.RequireNoError(t, medium.WriteMode("mode.txt", "abc", 0600))
	info, err := medium.Stat("mode.txt")
	core.RequireNoError(t, err)
	core.AssertEqual(t, fs.FileMode(0644), info.Mode().Perm())
}

func TestS3_Info_Mode_Bad(t *core.T) {
	info := &fileInfo{}
	got := info.Mode()
	core.AssertEqual(t, fs.FileMode(0), got)
}

func TestS3_Info_Mode_Ugly(t *core.T) {
	info := &fileInfo{mode: fs.ModeDir | 0755}
	got := info.Mode()
	core.AssertTrue(t, got.IsDir())
}

func TestS3_Info_ModTime_Good(t *core.T) {
	medium := newS3MediumFixture(t)
	core.RequireNoError(t, medium.Write("time.txt", "abc"))
	info, err := medium.Stat("time.txt")
	core.RequireNoError(t, err)
	core.AssertFalse(t, info.ModTime().IsZero())
}

func TestS3_Info_ModTime_Bad(t *core.T) {
	info := &fileInfo{}
	got := info.ModTime()
	core.AssertTrue(t, got.IsZero())
}

func TestS3_Info_ModTime_Ugly(t *core.T) {
	stamp := core.Now()
	info := &fileInfo{modTime: stamp}
	core.AssertEqual(t, stamp, info.ModTime())
}

func TestS3_Info_IsDir_Good(t *core.T) {
	info := &fileInfo{name: "dir", isDir: true, mode: fs.ModeDir | 0755}
	got := info.IsDir()
	core.AssertTrue(t, got)
}

func TestS3_Info_IsDir_Bad(t *core.T) {
	info := &fileInfo{}
	got := info.IsDir()
	core.AssertFalse(t, got)
}

func TestS3_Info_IsDir_Ugly(t *core.T) {
	info := &fileInfo{isDir: true}
	got := info.IsDir()
	core.AssertTrue(t, got)
}

func TestS3_Info_Sys_Good(t *core.T) {
	info := &fileInfo{name: "sys"}
	got := info.Sys()
	core.AssertNil(t, got)
}

func TestS3_Info_Sys_Bad(t *core.T) {
	info := &fileInfo{}
	got := info.Sys()
	core.AssertNil(t, got)
}

func TestS3_Info_Sys_Ugly(t *core.T) {
	var info *fileInfo
	got := info.Sys()
	core.AssertNil(t, got)
}

func TestS3_Entry_Name_Good(t *core.T) {
	medium := newS3MediumFixture(t)
	core.RequireNoError(t, medium.Write("dir/entry.txt", "abc"))
	entries, err := medium.List("dir")
	core.RequireNoError(t, err)
	core.AssertEqual(t, "entry.txt", entries[0].Name())
}

func TestS3_Entry_Name_Bad(t *core.T) {
	entry := &dirEntry{}
	got := entry.Name()
	core.AssertEqual(t, "", got)
}

func TestS3_Entry_Name_Ugly(t *core.T) {
	entry := &dirEntry{name: ""}
	got := entry.Name()
	core.AssertEqual(t, "", got)
}

func TestS3_Entry_IsDir_Good(t *core.T) {
	entry := &dirEntry{name: "dir", isDir: true}
	got := entry.IsDir()
	core.AssertTrue(t, got)
}

func TestS3_Entry_IsDir_Bad(t *core.T) {
	entry := &dirEntry{name: "file"}
	got := entry.IsDir()
	core.AssertFalse(t, got)
}

func TestS3_Entry_IsDir_Ugly(t *core.T) {
	entry := &dirEntry{}
	got := entry.IsDir()
	core.AssertFalse(t, got)
}

func TestS3_Entry_Type_Good(t *core.T) {
	entry := &dirEntry{name: "dir", isDir: true, mode: fs.ModeDir | 0755}
	got := entry.Type()
	core.AssertTrue(t, got.IsDir())
}

func TestS3_Entry_Type_Bad(t *core.T) {
	entry := &dirEntry{name: "file"}
	got := entry.Type()
	core.AssertEqual(t, fs.FileMode(0), got)
}

func TestS3_Entry_Type_Ugly(t *core.T) {
	entry := &dirEntry{}
	got := entry.Type()
	core.AssertEqual(t, fs.FileMode(0), got)
}

func TestS3_Entry_Info_Good(t *core.T) {
	entry := &dirEntry{name: "file", info: &fileInfo{name: "file"}}
	info, err := entry.Info()
	core.AssertNoError(t, err)
	core.AssertEqual(t, "file", info.Name())
}

func TestS3_Entry_Info_Bad(t *core.T) {
	entry := &dirEntry{}
	info, err := entry.Info()
	core.AssertNoError(t, err)
	core.AssertNil(t, info)
}

func TestS3_Entry_Info_Ugly(t *core.T) {
	entry := &dirEntry{info: &fileInfo{name: ""}}
	info, err := entry.Info()
	core.AssertNoError(t, err)
	core.AssertEqual(t, "", info.Name())
}

func TestS3_File_Read_Good(t *core.T) {
	medium := newS3MediumFixture(t)
	core.RequireNoError(t, medium.Write(s3FilePath, "abc"))
	file, err := medium.Open(s3FilePath)
	core.RequireNoError(t, err)
	buf := make([]byte, 3)
	n, readErr := file.Read(buf)
	core.AssertNoError(t, readErr)
	core.AssertEqual(t, 3, n)
}

func TestS3_File_Read_Bad(t *core.T) {
	file := &s3File{}
	buf := make([]byte, 1)
	n, err := file.Read(buf)
	core.AssertErrorIs(t, err, goio.EOF)
	core.AssertEqual(t, 0, n)
}

func TestS3_File_Read_Ugly(t *core.T) {
	file := &s3File{content: []byte("abc"), offset: 2}
	buf := make([]byte, 4)
	n, err := file.Read(buf)
	core.AssertNoError(t, err)
	core.AssertEqual(t, 1, n)
}

func TestS3_File_Stat_Good(t *core.T) {
	file := &s3File{name: s3FilePath, content: []byte("abc")}
	info, err := file.Stat()
	core.AssertNoError(t, err)
	core.AssertEqual(t, s3FilePath, info.Name())
}

func TestS3_File_Stat_Bad(t *core.T) {
	file := &s3File{}
	info, err := file.Stat()
	core.AssertNoError(t, err)
	core.AssertEqual(t, "", info.Name())
}

func TestS3_File_Stat_Ugly(t *core.T) {
	file := &s3File{name: "empty", content: nil}
	info, err := file.Stat()
	core.AssertNoError(t, err)
	core.AssertEqual(t, int64(0), info.Size())
}

func TestS3_File_Close_Good(t *core.T) {
	file := &s3File{name: s3FilePath}
	err := file.Close()
	core.AssertNoError(t, err)
}

func TestS3_File_Close_Bad(t *core.T) {
	file := &s3File{}
	err := file.Close()
	core.AssertNoError(t, err)
}

func TestS3_File_Close_Ugly(t *core.T) {
	file := &s3File{}
	core.AssertNoError(t, file.Close())
	core.AssertNoError(t, file.Close())
}

func TestS3_WriteCloser_Write_Good(t *core.T) {
	medium := newS3MediumFixture(t)
	writer, err := medium.Create("writer.txt")
	core.RequireNoError(t, err)
	n, writeErr := writer.Write([]byte("abc"))
	core.AssertNoError(t, writeErr)
	core.AssertEqual(t, 3, n)
}

func TestS3_WriteCloser_Write_Bad(t *core.T) {
	writer := &s3WriteCloser{}
	n, err := writer.Write(nil)
	core.AssertNoError(t, err)
	core.AssertEqual(t, 0, n)
}

func TestS3_WriteCloser_Write_Ugly(t *core.T) {
	writer := &s3WriteCloser{}
	n, err := writer.Write([]byte{})
	core.AssertNoError(t, err)
	core.AssertEqual(t, 0, n)
}

func TestS3_WriteCloser_Close_Good(t *core.T) {
	medium := newS3MediumFixture(t)
	writer, err := medium.Create("writer-close.txt")
	core.RequireNoError(t, err)
	_, writeErr := writer.Write([]byte("abc"))
	core.RequireNoError(t, writeErr)
	core.AssertNoError(t, writer.Close())
}

func TestS3_WriteCloser_Close_Bad(t *core.T) {
	writer := &s3WriteCloser{}
	core.AssertPanics(t, func() {
		_ = writer.Close()
	})
}

func TestS3_WriteCloser_Close_Ugly(t *core.T) {
	medium := newS3MediumFixture(t)
	writer, err := medium.Create("empty.txt")
	core.RequireNoError(t, err)
	_, writeErr := writer.Write([]byte("x"))
	core.RequireNoError(t, writeErr)
	core.AssertNoError(t, writer.Close())
}
