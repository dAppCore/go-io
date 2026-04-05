package s3

import (
	"bytes"
	"context"
	goio "io"
	"io/fs"
	"sort"
	"sync"
	"testing"
	"time"

	core "dappco.re/go/core"
	"github.com/aws/aws-sdk-go-v2/aws"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		Body:          goio.NopCloser(bytes.NewReader(data)),
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

	var allKeys []string
	for k := range client.objects {
		if core.HasPrefix(k, prefix) {
			allKeys = append(allKeys, k)
		}
	}
	sort.Strings(allKeys)

	continuationToken := aws.ToString(params.ContinuationToken)

	var contents []types.Object
	commonPrefixes := make(map[string]bool)
	truncated := false
	var nextToken string

	past := continuationToken == ""
	for _, k := range allKeys {
		if !past {
			if k == continuationToken {
				past = true
			}
			continue
		}

		rest := core.TrimPrefix(k, prefix)

		if delimiter != "" {
			parts := core.SplitN(rest, delimiter, 2)
			if len(parts) == 2 {
				cp := core.Concat(prefix, parts[0], delimiter)
				commonPrefixes[cp] = true
				continue
			}
		}

		if int32(len(contents)) >= maxKeys {
			truncated = true
			nextToken = k
			break
		}

		data := client.objects[k]
		mtime := client.mtimes[k]
		contents = append(contents, types.Object{
			Key:          aws.String(k),
			Size:         aws.Int64(int64(len(data))),
			LastModified: &mtime,
		})
	}

	var cpSlice []types.CommonPrefix
	var cpKeys []string
	for cp := range commonPrefixes {
		cpKeys = append(cpKeys, cp)
	}
	sort.Strings(cpKeys)
	for _, cp := range cpKeys {
		cpSlice = append(cpSlice, types.CommonPrefix{Prefix: aws.String(cp)})
	}

	out := &awss3.ListObjectsV2Output{
		Contents:       contents,
		CommonPrefixes: cpSlice,
		IsTruncated:    aws.Bool(truncated),
	}
	if truncated {
		out.NextContinuationToken = aws.String(nextToken)
	}
	return out, nil
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

func newS3Medium(t *testing.T) (*Medium, *testS3Client) {
	t.Helper()
	testS3Client := newTestS3Client()
	s3Medium, err := New(Options{Bucket: "test-bucket", Client: testS3Client})
	require.NoError(t, err)
	return s3Medium, testS3Client
}

func TestS3_New_Good(t *testing.T) {
	testS3Client := newTestS3Client()
	s3Medium, err := New(Options{Bucket: "my-bucket", Client: testS3Client})
	require.NoError(t, err)
	assert.Equal(t, "my-bucket", s3Medium.bucket)
	assert.Equal(t, "", s3Medium.prefix)
}

func TestS3_New_NoBucket_Bad(t *testing.T) {
	_, err := New(Options{Client: newTestS3Client()})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "bucket name is required")
}

func TestS3_New_NoClient_Bad(t *testing.T) {
	_, err := New(Options{Bucket: "bucket"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "client is required")
}

func TestS3_New_Options_Good(t *testing.T) {
	testS3Client := newTestS3Client()
	s3Medium, err := New(Options{Bucket: "bucket", Client: testS3Client, Prefix: "data/"})
	require.NoError(t, err)
	assert.Equal(t, "data/", s3Medium.prefix)

	prefixedS3Medium, err := New(Options{Bucket: "bucket", Client: testS3Client, Prefix: "data"})
	require.NoError(t, err)
	assert.Equal(t, "data/", prefixedS3Medium.prefix)
}

func TestS3_ReadWrite_Good(t *testing.T) {
	s3Medium, _ := newS3Medium(t)

	err := s3Medium.Write("hello.txt", "world")
	require.NoError(t, err)

	content, err := s3Medium.Read("hello.txt")
	require.NoError(t, err)
	assert.Equal(t, "world", content)
}

func TestS3_ReadWrite_NotFound_Bad(t *testing.T) {
	s3Medium, _ := newS3Medium(t)

	_, err := s3Medium.Read("nonexistent.txt")
	assert.Error(t, err)
}

func TestS3_ReadWrite_EmptyPath_Bad(t *testing.T) {
	s3Medium, _ := newS3Medium(t)

	_, err := s3Medium.Read("")
	assert.Error(t, err)

	err = s3Medium.Write("", "content")
	assert.Error(t, err)
}

func TestS3_ReadWrite_Prefix_Good(t *testing.T) {
	testS3Client := newTestS3Client()
	s3Medium, err := New(Options{Bucket: "bucket", Client: testS3Client, Prefix: "pfx"})
	require.NoError(t, err)

	err = s3Medium.Write("file.txt", "data")
	require.NoError(t, err)

	_, ok := testS3Client.objects["pfx/file.txt"]
	assert.True(t, ok, "object should be stored with prefix")

	content, err := s3Medium.Read("file.txt")
	require.NoError(t, err)
	assert.Equal(t, "data", content)
}

func TestS3_EnsureDir_Good(t *testing.T) {
	medium, _ := newS3Medium(t)
	err := medium.EnsureDir("any/path")
	assert.NoError(t, err)
}

func TestS3_IsFile_Good(t *testing.T) {
	s3Medium, _ := newS3Medium(t)

	err := s3Medium.Write("file.txt", "content")
	require.NoError(t, err)

	assert.True(t, s3Medium.IsFile("file.txt"))
	assert.False(t, s3Medium.IsFile("nonexistent.txt"))
	assert.False(t, s3Medium.IsFile(""))
}

func TestS3_Delete_Good(t *testing.T) {
	s3Medium, _ := newS3Medium(t)

	err := s3Medium.Write("to-delete.txt", "content")
	require.NoError(t, err)
	assert.True(t, s3Medium.Exists("to-delete.txt"))

	err = s3Medium.Delete("to-delete.txt")
	require.NoError(t, err)
	assert.False(t, s3Medium.IsFile("to-delete.txt"))
}

func TestS3_Delete_EmptyPath_Bad(t *testing.T) {
	s3Medium, _ := newS3Medium(t)
	err := s3Medium.Delete("")
	assert.Error(t, err)
}

func TestS3_DeleteAll_Good(t *testing.T) {
	s3Medium, _ := newS3Medium(t)

	require.NoError(t, s3Medium.Write("dir/file1.txt", "a"))
	require.NoError(t, s3Medium.Write("dir/sub/file2.txt", "b"))
	require.NoError(t, s3Medium.Write("other.txt", "c"))

	err := s3Medium.DeleteAll("dir")
	require.NoError(t, err)

	assert.False(t, s3Medium.IsFile("dir/file1.txt"))
	assert.False(t, s3Medium.IsFile("dir/sub/file2.txt"))
	assert.True(t, s3Medium.IsFile("other.txt"))
}

func TestS3_DeleteAll_EmptyPath_Bad(t *testing.T) {
	s3Medium, _ := newS3Medium(t)
	err := s3Medium.DeleteAll("")
	assert.Error(t, err)
}

func TestS3_DeleteAll_DeleteObjectError_Bad(t *testing.T) {
	s3Medium, testS3Client := newS3Medium(t)
	testS3Client.deleteObjectErrors["dir"] = core.NewError("boom")

	err := s3Medium.DeleteAll("dir")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to delete object: dir")
}

func TestS3_DeleteAll_PartialDelete_Bad(t *testing.T) {
	s3Medium, testS3Client := newS3Medium(t)

	require.NoError(t, s3Medium.Write("dir/file1.txt", "a"))
	require.NoError(t, s3Medium.Write("dir/file2.txt", "b"))
	testS3Client.deleteObjectsErrs["dir/file2.txt"] = types.Error{
		Key:     aws.String("dir/file2.txt"),
		Code:    aws.String("AccessDenied"),
		Message: aws.String("blocked"),
	}

	err := s3Medium.DeleteAll("dir")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "partial delete failed")
	assert.Contains(t, err.Error(), "dir/file2.txt")
	assert.True(t, s3Medium.IsFile("dir/file2.txt"))
	assert.False(t, s3Medium.IsFile("dir/file1.txt"))
}

func TestS3_Rename_Good(t *testing.T) {
	s3Medium, _ := newS3Medium(t)

	require.NoError(t, s3Medium.Write("old.txt", "content"))
	assert.True(t, s3Medium.IsFile("old.txt"))

	err := s3Medium.Rename("old.txt", "new.txt")
	require.NoError(t, err)

	assert.False(t, s3Medium.IsFile("old.txt"))
	assert.True(t, s3Medium.IsFile("new.txt"))

	content, err := s3Medium.Read("new.txt")
	require.NoError(t, err)
	assert.Equal(t, "content", content)
}

func TestS3_Rename_EmptyPath_Bad(t *testing.T) {
	s3Medium, _ := newS3Medium(t)
	err := s3Medium.Rename("", "new.txt")
	assert.Error(t, err)

	err = s3Medium.Rename("old.txt", "")
	assert.Error(t, err)
}

func TestS3_Rename_SourceNotFound_Bad(t *testing.T) {
	s3Medium, _ := newS3Medium(t)
	err := s3Medium.Rename("nonexistent.txt", "new.txt")
	assert.Error(t, err)
}

func TestS3_List_Good(t *testing.T) {
	s3Medium, _ := newS3Medium(t)

	require.NoError(t, s3Medium.Write("dir/file1.txt", "a"))
	require.NoError(t, s3Medium.Write("dir/file2.txt", "b"))
	require.NoError(t, s3Medium.Write("dir/sub/file3.txt", "c"))

	entries, err := s3Medium.List("dir")
	require.NoError(t, err)

	names := make(map[string]bool)
	for _, entry := range entries {
		names[entry.Name()] = true
	}

	assert.True(t, names["file1.txt"], "should list file1.txt")
	assert.True(t, names["file2.txt"], "should list file2.txt")
	assert.True(t, names["sub"], "should list sub directory")
	assert.Len(t, entries, 3)

	for _, entry := range entries {
		if entry.Name() == "sub" {
			assert.True(t, entry.IsDir())
			info, err := entry.Info()
			require.NoError(t, err)
			assert.True(t, info.IsDir())
		}
	}
}

func TestS3_List_Root_Good(t *testing.T) {
	s3Medium, _ := newS3Medium(t)

	require.NoError(t, s3Medium.Write("root.txt", "content"))
	require.NoError(t, s3Medium.Write("dir/nested.txt", "nested"))

	entries, err := s3Medium.List("")
	require.NoError(t, err)

	names := make(map[string]bool)
	for _, entry := range entries {
		names[entry.Name()] = true
	}

	assert.True(t, names["root.txt"])
	assert.True(t, names["dir"])
}

func TestS3_Stat_Good(t *testing.T) {
	s3Medium, _ := newS3Medium(t)

	require.NoError(t, s3Medium.Write("file.txt", "hello world"))

	info, err := s3Medium.Stat("file.txt")
	require.NoError(t, err)
	assert.Equal(t, "file.txt", info.Name())
	assert.Equal(t, int64(11), info.Size())
	assert.False(t, info.IsDir())
}

func TestS3_Stat_NotFound_Bad(t *testing.T) {
	s3Medium, _ := newS3Medium(t)

	_, err := s3Medium.Stat("nonexistent.txt")
	assert.Error(t, err)
}

func TestS3_Stat_EmptyPath_Bad(t *testing.T) {
	s3Medium, _ := newS3Medium(t)
	_, err := s3Medium.Stat("")
	assert.Error(t, err)
}

func TestS3_Open_Good(t *testing.T) {
	s3Medium, _ := newS3Medium(t)

	require.NoError(t, s3Medium.Write("file.txt", "open me"))

	file, err := s3Medium.Open("file.txt")
	require.NoError(t, err)
	defer file.Close()

	data, err := goio.ReadAll(file.(goio.Reader))
	require.NoError(t, err)
	assert.Equal(t, "open me", string(data))

	stat, err := file.Stat()
	require.NoError(t, err)
	assert.Equal(t, "file.txt", stat.Name())
}

func TestS3_Open_NotFound_Bad(t *testing.T) {
	s3Medium, _ := newS3Medium(t)

	_, err := s3Medium.Open("nonexistent.txt")
	assert.Error(t, err)
}

func TestS3_Create_Good(t *testing.T) {
	s3Medium, _ := newS3Medium(t)

	writer, err := s3Medium.Create("new.txt")
	require.NoError(t, err)

	bytesWritten, err := writer.Write([]byte("created"))
	require.NoError(t, err)
	assert.Equal(t, 7, bytesWritten)

	err = writer.Close()
	require.NoError(t, err)

	content, err := s3Medium.Read("new.txt")
	require.NoError(t, err)
	assert.Equal(t, "created", content)
}

func TestS3_Append_Good(t *testing.T) {
	s3Medium, _ := newS3Medium(t)

	require.NoError(t, s3Medium.Write("append.txt", "hello"))

	writer, err := s3Medium.Append("append.txt")
	require.NoError(t, err)

	_, err = writer.Write([]byte(" world"))
	require.NoError(t, err)
	err = writer.Close()
	require.NoError(t, err)

	content, err := s3Medium.Read("append.txt")
	require.NoError(t, err)
	assert.Equal(t, "hello world", content)
}

func TestS3_Append_NewFile_Good(t *testing.T) {
	s3Medium, _ := newS3Medium(t)

	writer, err := s3Medium.Append("new.txt")
	require.NoError(t, err)

	_, err = writer.Write([]byte("fresh"))
	require.NoError(t, err)
	err = writer.Close()
	require.NoError(t, err)

	content, err := s3Medium.Read("new.txt")
	require.NoError(t, err)
	assert.Equal(t, "fresh", content)
}

func TestS3_ReadStream_Good(t *testing.T) {
	s3Medium, _ := newS3Medium(t)

	require.NoError(t, s3Medium.Write("stream.txt", "streaming content"))

	reader, err := s3Medium.ReadStream("stream.txt")
	require.NoError(t, err)
	defer reader.Close()

	data, err := goio.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, "streaming content", string(data))
}

func TestS3_ReadStream_NotFound_Bad(t *testing.T) {
	s3Medium, _ := newS3Medium(t)
	_, err := s3Medium.ReadStream("nonexistent.txt")
	assert.Error(t, err)
}

func TestS3_WriteStream_Good(t *testing.T) {
	s3Medium, _ := newS3Medium(t)

	writer, err := s3Medium.WriteStream("output.txt")
	require.NoError(t, err)

	_, err = goio.Copy(writer, core.NewReader("piped data"))
	require.NoError(t, err)
	err = writer.Close()
	require.NoError(t, err)

	content, err := s3Medium.Read("output.txt")
	require.NoError(t, err)
	assert.Equal(t, "piped data", content)
}

func TestS3_Exists_Good(t *testing.T) {
	s3Medium, _ := newS3Medium(t)

	assert.False(t, s3Medium.Exists("nonexistent.txt"))

	require.NoError(t, s3Medium.Write("file.txt", "content"))
	assert.True(t, s3Medium.Exists("file.txt"))
}

func TestS3_Exists_DirectoryPrefix_Good(t *testing.T) {
	s3Medium, _ := newS3Medium(t)

	require.NoError(t, s3Medium.Write("dir/file.txt", "content"))
	assert.True(t, s3Medium.Exists("dir"))
}

func TestS3_IsDir_Good(t *testing.T) {
	s3Medium, _ := newS3Medium(t)

	require.NoError(t, s3Medium.Write("dir/file.txt", "content"))

	assert.True(t, s3Medium.IsDir("dir"))
	assert.False(t, s3Medium.IsDir("dir/file.txt"))
	assert.False(t, s3Medium.IsDir("nonexistent"))
	assert.False(t, s3Medium.IsDir(""))
}

func TestS3_ObjectKey_Good(t *testing.T) {
	testS3Client := newTestS3Client()

	s3Medium, _ := New(Options{Bucket: "bucket", Client: testS3Client})
	assert.Equal(t, "file.txt", s3Medium.objectKey("file.txt"))
	assert.Equal(t, "dir/file.txt", s3Medium.objectKey("dir/file.txt"))
	assert.Equal(t, "", s3Medium.objectKey(""))
	assert.Equal(t, "file.txt", s3Medium.objectKey("/file.txt"))
	assert.Equal(t, "file.txt", s3Medium.objectKey("../file.txt"))

	prefixedS3Medium, _ := New(Options{Bucket: "bucket", Client: testS3Client, Prefix: "pfx"})
	assert.Equal(t, "pfx/file.txt", prefixedS3Medium.objectKey("file.txt"))
	assert.Equal(t, "pfx/dir/file.txt", prefixedS3Medium.objectKey("dir/file.txt"))
	assert.Equal(t, "pfx/", prefixedS3Medium.objectKey(""))
}

func TestS3_InterfaceCompliance_Good(t *testing.T) {
	testS3Client := newTestS3Client()
	s3Medium, err := New(Options{Bucket: "bucket", Client: testS3Client})
	require.NoError(t, err)

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
