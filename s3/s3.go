// Package s3 stores io.Medium data in S3 objects.
//
//	client := awss3.NewFromConfig(aws.Config{Region: "us-east-1"})
//	medium, _ := s3.New(s3.Options{Bucket: "backups", Client: client, Prefix: "daily/"})
//	_ = medium.Write("reports/daily.txt", "done")
package s3

import (
	"bytes"
	"context"
	goio "io"
	"io/fs"
	"path"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"

	core "dappco.re/go/core"
	coreio "dappco.re/go/core/io"
)

// Example: client := awss3.NewFromConfig(aws.Config{Region: "us-east-1"})
// medium, _ := s3.New(s3.Options{Bucket: "backups", Client: client, Prefix: "daily/"})
type Client interface {
	GetObject(ctx context.Context, params *awss3.GetObjectInput, optFns ...func(*awss3.Options)) (*awss3.GetObjectOutput, error)
	PutObject(ctx context.Context, params *awss3.PutObjectInput, optFns ...func(*awss3.Options)) (*awss3.PutObjectOutput, error)
	DeleteObject(ctx context.Context, params *awss3.DeleteObjectInput, optFns ...func(*awss3.Options)) (*awss3.DeleteObjectOutput, error)
	DeleteObjects(ctx context.Context, params *awss3.DeleteObjectsInput, optFns ...func(*awss3.Options)) (*awss3.DeleteObjectsOutput, error)
	HeadObject(ctx context.Context, params *awss3.HeadObjectInput, optFns ...func(*awss3.Options)) (*awss3.HeadObjectOutput, error)
	ListObjectsV2(ctx context.Context, params *awss3.ListObjectsV2Input, optFns ...func(*awss3.Options)) (*awss3.ListObjectsV2Output, error)
	CopyObject(ctx context.Context, params *awss3.CopyObjectInput, optFns ...func(*awss3.Options)) (*awss3.CopyObjectOutput, error)
}

// Example: medium, _ := s3.New(s3.Options{Bucket: "backups", Client: client, Prefix: "daily/"})
// _ = medium.Write("reports/daily.txt", "done")
type Medium struct {
	client Client
	bucket string
	prefix string
}

var _ coreio.Medium = (*Medium)(nil)

// Example: medium, _ := s3.New(s3.Options{Bucket: "backups", Client: client, Prefix: "daily/"})
type Options struct {
	// Bucket is the target S3 bucket name.
	Bucket string
	// Client is the AWS S3 client or test double used for requests.
	Client Client
	// Prefix is prepended to every object key.
	Prefix string
}

func deleteObjectsError(prefix string, errs []types.Error) error {
	if len(errs) == 0 {
		return nil
	}
	details := make([]string, 0, len(errs))
	for _, item := range errs {
		key := aws.ToString(item.Key)
		code := aws.ToString(item.Code)
		message := aws.ToString(item.Message)
		switch {
		case code != "" && message != "":
			details = append(details, core.Concat(key, ": ", code, " ", message))
		case code != "":
			details = append(details, core.Concat(key, ": ", code))
		case message != "":
			details = append(details, core.Concat(key, ": ", message))
		default:
			details = append(details, key)
		}
	}
	return core.E("s3.DeleteAll", core.Concat("partial delete failed under ", prefix, ": ", core.Join("; ", details...)), nil)
}

func normalisePrefix(prefix string) string {
	if prefix == "" {
		return ""
	}
	clean := path.Clean("/" + prefix)
	if clean == "/" {
		return ""
	}
	clean = core.TrimPrefix(clean, "/")
	if clean != "" && !core.HasSuffix(clean, "/") {
		clean += "/"
	}
	return clean
}

// Example: medium, _ := s3.New(s3.Options{Bucket: "backups", Client: client, Prefix: "daily/"})
// _ = medium.Write("reports/daily.txt", "done")
func New(options Options) (*Medium, error) {
	if options.Bucket == "" {
		return nil, core.E("s3.New", "bucket name is required", nil)
	}
	if options.Client == nil {
		return nil, core.E("s3.New", "client is required", nil)
	}
	m := &Medium{
		client: options.Client,
		bucket: options.Bucket,
		prefix: normalisePrefix(options.Prefix),
	}
	return m, nil
}

// objectKey maps a virtual path to the full S3 object key.
func (m *Medium) objectKey(filePath string) string {
	// Clean the path using a leading "/" to sandbox traversal attempts,
	// then strip the "/" prefix. This ensures ".." can't escape.
	clean := path.Clean("/" + filePath)
	if clean == "/" {
		clean = ""
	}
	clean = core.TrimPrefix(clean, "/")

	if m.prefix == "" {
		return clean
	}
	if clean == "" {
		return m.prefix
	}
	return m.prefix + clean
}

func (m *Medium) Read(filePath string) (string, error) {
	key := m.objectKey(filePath)
	if key == "" {
		return "", core.E("s3.Read", "path is required", fs.ErrInvalid)
	}

	out, err := m.client.GetObject(context.Background(), &awss3.GetObjectInput{
		Bucket: aws.String(m.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return "", core.E("s3.Read", core.Concat("failed to get object: ", key), err)
	}
	defer out.Body.Close()

	data, err := goio.ReadAll(out.Body)
	if err != nil {
		return "", core.E("s3.Read", core.Concat("failed to read body: ", key), err)
	}
	return string(data), nil
}

func (m *Medium) Write(filePath, content string) error {
	key := m.objectKey(filePath)
	if key == "" {
		return core.E("s3.Write", "path is required", fs.ErrInvalid)
	}

	_, err := m.client.PutObject(context.Background(), &awss3.PutObjectInput{
		Bucket: aws.String(m.bucket),
		Key:    aws.String(key),
		Body:   core.NewReader(content),
	})
	if err != nil {
		return core.E("s3.Write", core.Concat("failed to put object: ", key), err)
	}
	return nil
}

// Example: _ = medium.WriteMode("keys/private.key", key, 0600)
func (m *Medium) WriteMode(filePath, content string, _ fs.FileMode) error {
	return m.Write(filePath, content)
}

// Example: _ = medium.EnsureDir("reports/2026")
func (m *Medium) EnsureDir(_ string) error {
	return nil
}

// Example: ok := medium.IsFile("reports/daily.txt")
func (m *Medium) IsFile(filePath string) bool {
	key := m.objectKey(filePath)
	if key == "" {
		return false
	}
	// A "file" in S3 is an object whose key does not end with "/"
	if core.HasSuffix(key, "/") {
		return false
	}
	_, err := m.client.HeadObject(context.Background(), &awss3.HeadObjectInput{
		Bucket: aws.String(m.bucket),
		Key:    aws.String(key),
	})
	return err == nil
}

func (m *Medium) FileGet(filePath string) (string, error) {
	return m.Read(filePath)
}

func (m *Medium) FileSet(filePath, content string) error {
	return m.Write(filePath, content)
}

func (m *Medium) Delete(filePath string) error {
	key := m.objectKey(filePath)
	if key == "" {
		return core.E("s3.Delete", "path is required", fs.ErrInvalid)
	}

	_, err := m.client.DeleteObject(context.Background(), &awss3.DeleteObjectInput{
		Bucket: aws.String(m.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return core.E("s3.Delete", core.Concat("failed to delete object: ", key), err)
	}
	return nil
}

// Example: _ = medium.DeleteAll("reports/2026")
func (m *Medium) DeleteAll(filePath string) error {
	key := m.objectKey(filePath)
	if key == "" {
		return core.E("s3.DeleteAll", "path is required", fs.ErrInvalid)
	}

	// First, try deleting the exact key
	_, err := m.client.DeleteObject(context.Background(), &awss3.DeleteObjectInput{
		Bucket: aws.String(m.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return core.E("s3.DeleteAll", core.Concat("failed to delete object: ", key), err)
	}

	// Then delete all objects under the prefix
	prefix := key
	if !core.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	paginator := true
	var continuationToken *string

	for paginator {
		listOut, err := m.client.ListObjectsV2(context.Background(), &awss3.ListObjectsV2Input{
			Bucket:            aws.String(m.bucket),
			Prefix:            aws.String(prefix),
			ContinuationToken: continuationToken,
		})
		if err != nil {
			return core.E("s3.DeleteAll", core.Concat("failed to list objects: ", prefix), err)
		}

		if len(listOut.Contents) == 0 {
			break
		}

		objects := make([]types.ObjectIdentifier, len(listOut.Contents))
		for i, obj := range listOut.Contents {
			objects[i] = types.ObjectIdentifier{Key: obj.Key}
		}

		deleteOut, err := m.client.DeleteObjects(context.Background(), &awss3.DeleteObjectsInput{
			Bucket: aws.String(m.bucket),
			Delete: &types.Delete{Objects: objects, Quiet: aws.Bool(true)},
		})
		if err != nil {
			return core.E("s3.DeleteAll", "failed to delete objects", err)
		}
		if err := deleteObjectsError(prefix, deleteOut.Errors); err != nil {
			return err
		}

		if listOut.IsTruncated != nil && *listOut.IsTruncated {
			continuationToken = listOut.NextContinuationToken
		} else {
			paginator = false
		}
	}

	return nil
}

// Example: _ = medium.Rename("drafts/todo.txt", "archive/todo.txt")
func (m *Medium) Rename(oldPath, newPath string) error {
	oldKey := m.objectKey(oldPath)
	newKey := m.objectKey(newPath)
	if oldKey == "" || newKey == "" {
		return core.E("s3.Rename", "both old and new paths are required", fs.ErrInvalid)
	}

	copySource := m.bucket + "/" + oldKey

	_, err := m.client.CopyObject(context.Background(), &awss3.CopyObjectInput{
		Bucket:     aws.String(m.bucket),
		CopySource: aws.String(copySource),
		Key:        aws.String(newKey),
	})
	if err != nil {
		return core.E("s3.Rename", core.Concat("failed to copy object: ", oldKey, " -> ", newKey), err)
	}

	_, err = m.client.DeleteObject(context.Background(), &awss3.DeleteObjectInput{
		Bucket: aws.String(m.bucket),
		Key:    aws.String(oldKey),
	})
	if err != nil {
		return core.E("s3.Rename", core.Concat("failed to delete source object: ", oldKey), err)
	}

	return nil
}

// Example: entries, _ := medium.List("reports")
func (m *Medium) List(filePath string) ([]fs.DirEntry, error) {
	prefix := m.objectKey(filePath)
	if prefix != "" && !core.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	var entries []fs.DirEntry

	listOut, err := m.client.ListObjectsV2(context.Background(), &awss3.ListObjectsV2Input{
		Bucket:    aws.String(m.bucket),
		Prefix:    aws.String(prefix),
		Delimiter: aws.String("/"),
	})
	if err != nil {
		return nil, core.E("s3.List", core.Concat("failed to list objects: ", prefix), err)
	}

	// Common prefixes are "directories"
	for _, cp := range listOut.CommonPrefixes {
		if cp.Prefix == nil {
			continue
		}
		name := core.TrimPrefix(*cp.Prefix, prefix)
		name = core.TrimSuffix(name, "/")
		if name == "" {
			continue
		}
		entries = append(entries, &dirEntry{
			name:  name,
			isDir: true,
			mode:  fs.ModeDir | 0755,
			info: &fileInfo{
				name:  name,
				isDir: true,
				mode:  fs.ModeDir | 0755,
			},
		})
	}

	// Contents are "files" (excluding the prefix itself)
	for _, obj := range listOut.Contents {
		if obj.Key == nil {
			continue
		}
		name := core.TrimPrefix(*obj.Key, prefix)
		if name == "" || core.Contains(name, "/") {
			continue
		}
		var size int64
		if obj.Size != nil {
			size = *obj.Size
		}
		var modTime time.Time
		if obj.LastModified != nil {
			modTime = *obj.LastModified
		}
		entries = append(entries, &dirEntry{
			name:  name,
			isDir: false,
			mode:  0644,
			info: &fileInfo{
				name:    name,
				size:    size,
				mode:    0644,
				modTime: modTime,
			},
		})
	}

	return entries, nil
}

// Example: info, _ := medium.Stat("reports/daily.txt")
func (m *Medium) Stat(filePath string) (fs.FileInfo, error) {
	key := m.objectKey(filePath)
	if key == "" {
		return nil, core.E("s3.Stat", "path is required", fs.ErrInvalid)
	}

	out, err := m.client.HeadObject(context.Background(), &awss3.HeadObjectInput{
		Bucket: aws.String(m.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, core.E("s3.Stat", core.Concat("failed to head object: ", key), err)
	}

	var size int64
	if out.ContentLength != nil {
		size = *out.ContentLength
	}
	var modTime time.Time
	if out.LastModified != nil {
		modTime = *out.LastModified
	}

	name := path.Base(key)
	return &fileInfo{
		name:    name,
		size:    size,
		mode:    0644,
		modTime: modTime,
	}, nil
}

func (m *Medium) Open(filePath string) (fs.File, error) {
	key := m.objectKey(filePath)
	if key == "" {
		return nil, core.E("s3.Open", "path is required", fs.ErrInvalid)
	}

	out, err := m.client.GetObject(context.Background(), &awss3.GetObjectInput{
		Bucket: aws.String(m.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, core.E("s3.Open", core.Concat("failed to get object: ", key), err)
	}

	data, err := goio.ReadAll(out.Body)
	out.Body.Close()
	if err != nil {
		return nil, core.E("s3.Open", core.Concat("failed to read body: ", key), err)
	}

	var size int64
	if out.ContentLength != nil {
		size = *out.ContentLength
	}
	var modTime time.Time
	if out.LastModified != nil {
		modTime = *out.LastModified
	}

	return &s3File{
		name:    path.Base(key),
		content: data,
		size:    size,
		modTime: modTime,
	}, nil
}

// Example: writer, _ := medium.Create("reports/daily.txt")
func (m *Medium) Create(filePath string) (goio.WriteCloser, error) {
	key := m.objectKey(filePath)
	if key == "" {
		return nil, core.E("s3.Create", "path is required", fs.ErrInvalid)
	}
	return &s3WriteCloser{
		medium: m,
		key:    key,
	}, nil
}

// Example: writer, _ := medium.Append("reports/daily.txt")
func (m *Medium) Append(filePath string) (goio.WriteCloser, error) {
	key := m.objectKey(filePath)
	if key == "" {
		return nil, core.E("s3.Append", "path is required", fs.ErrInvalid)
	}

	var existing []byte
	out, err := m.client.GetObject(context.Background(), &awss3.GetObjectInput{
		Bucket: aws.String(m.bucket),
		Key:    aws.String(key),
	})
	if err == nil {
		existing, _ = goio.ReadAll(out.Body)
		out.Body.Close()
	}

	return &s3WriteCloser{
		medium: m,
		key:    key,
		data:   existing,
	}, nil
}

func (m *Medium) ReadStream(filePath string) (goio.ReadCloser, error) {
	key := m.objectKey(filePath)
	if key == "" {
		return nil, core.E("s3.ReadStream", "path is required", fs.ErrInvalid)
	}

	out, err := m.client.GetObject(context.Background(), &awss3.GetObjectInput{
		Bucket: aws.String(m.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, core.E("s3.ReadStream", core.Concat("failed to get object: ", key), err)
	}
	return out.Body, nil
}

func (m *Medium) WriteStream(filePath string) (goio.WriteCloser, error) {
	return m.Create(filePath)
}

// Example: ok := medium.Exists("reports/daily.txt")
func (m *Medium) Exists(filePath string) bool {
	key := m.objectKey(filePath)
	if key == "" {
		return false
	}

	// Check as an exact object
	_, err := m.client.HeadObject(context.Background(), &awss3.HeadObjectInput{
		Bucket: aws.String(m.bucket),
		Key:    aws.String(key),
	})
	if err == nil {
		return true
	}

	// Check as a "directory" prefix
	prefix := key
	if !core.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	listOut, err := m.client.ListObjectsV2(context.Background(), &awss3.ListObjectsV2Input{
		Bucket:  aws.String(m.bucket),
		Prefix:  aws.String(prefix),
		MaxKeys: aws.Int32(1),
	})
	if err != nil {
		return false
	}
	return len(listOut.Contents) > 0 || len(listOut.CommonPrefixes) > 0
}

// Example: ok := medium.IsDir("reports")
func (m *Medium) IsDir(filePath string) bool {
	key := m.objectKey(filePath)
	if key == "" {
		return false
	}

	prefix := key
	if !core.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	listOut, err := m.client.ListObjectsV2(context.Background(), &awss3.ListObjectsV2Input{
		Bucket:  aws.String(m.bucket),
		Prefix:  aws.String(prefix),
		MaxKeys: aws.Int32(1),
	})
	if err != nil {
		return false
	}
	return len(listOut.Contents) > 0 || len(listOut.CommonPrefixes) > 0
}

// --- Internal types ---

// fileInfo implements fs.FileInfo for S3 objects.
type fileInfo struct {
	name    string
	size    int64
	mode    fs.FileMode
	modTime time.Time
	isDir   bool
}

func (fi *fileInfo) Name() string { return fi.name }

func (fi *fileInfo) Size() int64 { return fi.size }

func (fi *fileInfo) Mode() fs.FileMode { return fi.mode }

func (fi *fileInfo) ModTime() time.Time { return fi.modTime }

func (fi *fileInfo) IsDir() bool { return fi.isDir }

func (fi *fileInfo) Sys() any { return nil }

// dirEntry implements fs.DirEntry for S3 listings.
type dirEntry struct {
	name  string
	isDir bool
	mode  fs.FileMode
	info  fs.FileInfo
}

func (de *dirEntry) Name() string { return de.name }

func (de *dirEntry) IsDir() bool { return de.isDir }

func (de *dirEntry) Type() fs.FileMode { return de.mode.Type() }

func (de *dirEntry) Info() (fs.FileInfo, error) { return de.info, nil }

// s3File implements fs.File for S3 objects.
type s3File struct {
	name    string
	content []byte
	offset  int64
	size    int64
	modTime time.Time
}

func (f *s3File) Stat() (fs.FileInfo, error) {
	return &fileInfo{
		name:    f.name,
		size:    int64(len(f.content)),
		mode:    0644,
		modTime: f.modTime,
	}, nil
}

func (f *s3File) Read(b []byte) (int, error) {
	if f.offset >= int64(len(f.content)) {
		return 0, goio.EOF
	}
	n := copy(b, f.content[f.offset:])
	f.offset += int64(n)
	return n, nil
}

func (f *s3File) Close() error {
	return nil
}

// s3WriteCloser buffers writes and uploads to S3 on Close.
type s3WriteCloser struct {
	medium *Medium
	key    string
	data   []byte
}

func (w *s3WriteCloser) Write(p []byte) (int, error) {
	w.data = append(w.data, p...)
	return len(p), nil
}

func (w *s3WriteCloser) Close() error {
	_, err := w.medium.client.PutObject(context.Background(), &awss3.PutObjectInput{
		Bucket: aws.String(w.medium.bucket),
		Key:    aws.String(w.key),
		Body:   bytes.NewReader(w.data),
	})
	if err != nil {
		return core.E("s3.writeCloser.Close", "failed to upload on close", err)
	}
	return nil
}
