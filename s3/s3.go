// Package s3 stores io.Medium data in S3 objects.
//
// Example: client := awss3.NewFromConfig(aws.Config{Region: "us-east-1"})
// Example: medium, _ := s3.New(s3.Options{Bucket: "backups", Client: client, Prefix: "daily/"})
// Example: _ = medium.Write("reports/daily.txt", "done")
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
// Example: medium, _ := s3.New(s3.Options{Bucket: "backups", Client: client, Prefix: "daily/"})
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
// Example: _ = medium.Write("reports/daily.txt", "done")
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
	for _, errorItem := range errs {
		key := aws.ToString(errorItem.Key)
		code := aws.ToString(errorItem.Code)
		message := aws.ToString(errorItem.Message)
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
// Example: _ = medium.Write("reports/daily.txt", "done")
func New(options Options) (*Medium, error) {
	if options.Bucket == "" {
		return nil, core.E("s3.New", "bucket name is required", nil)
	}
	if options.Client == nil {
		return nil, core.E("s3.New", "client is required", nil)
	}
	medium := &Medium{
		client: options.Client,
		bucket: options.Bucket,
		prefix: normalisePrefix(options.Prefix),
	}
	return medium, nil
}

// objectKey maps a virtual path to the full S3 object key.
func (medium *Medium) objectKey(filePath string) string {
	// Clean the path using a leading "/" to sandbox traversal attempts,
	// then strip the "/" prefix. This ensures ".." can't escape.
	clean := path.Clean("/" + filePath)
	if clean == "/" {
		clean = ""
	}
	clean = core.TrimPrefix(clean, "/")

	if medium.prefix == "" {
		return clean
	}
	if clean == "" {
		return medium.prefix
	}
	return medium.prefix + clean
}

func (medium *Medium) Read(filePath string) (string, error) {
	key := medium.objectKey(filePath)
	if key == "" {
		return "", core.E("s3.Read", "path is required", fs.ErrInvalid)
	}

	out, err := medium.client.GetObject(context.Background(), &awss3.GetObjectInput{
		Bucket: aws.String(medium.bucket),
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

func (medium *Medium) Write(filePath, content string) error {
	key := medium.objectKey(filePath)
	if key == "" {
		return core.E("s3.Write", "path is required", fs.ErrInvalid)
	}

	_, err := medium.client.PutObject(context.Background(), &awss3.PutObjectInput{
		Bucket: aws.String(medium.bucket),
		Key:    aws.String(key),
		Body:   core.NewReader(content),
	})
	if err != nil {
		return core.E("s3.Write", core.Concat("failed to put object: ", key), err)
	}
	return nil
}

// Example: _ = medium.WriteMode("keys/private.key", key, 0600)
func (medium *Medium) WriteMode(filePath, content string, _ fs.FileMode) error {
	return medium.Write(filePath, content)
}

// Example: _ = medium.EnsureDir("reports/2026")
func (medium *Medium) EnsureDir(_ string) error {
	return nil
}

// Example: ok := medium.IsFile("reports/daily.txt")
func (medium *Medium) IsFile(filePath string) bool {
	key := medium.objectKey(filePath)
	if key == "" {
		return false
	}
	// A "file" in S3 is an object whose key does not end with "/"
	if core.HasSuffix(key, "/") {
		return false
	}
	_, err := medium.client.HeadObject(context.Background(), &awss3.HeadObjectInput{
		Bucket: aws.String(medium.bucket),
		Key:    aws.String(key),
	})
	return err == nil
}

func (medium *Medium) FileGet(filePath string) (string, error) {
	return medium.Read(filePath)
}

func (medium *Medium) FileSet(filePath, content string) error {
	return medium.Write(filePath, content)
}

func (medium *Medium) Delete(filePath string) error {
	key := medium.objectKey(filePath)
	if key == "" {
		return core.E("s3.Delete", "path is required", fs.ErrInvalid)
	}

	_, err := medium.client.DeleteObject(context.Background(), &awss3.DeleteObjectInput{
		Bucket: aws.String(medium.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return core.E("s3.Delete", core.Concat("failed to delete object: ", key), err)
	}
	return nil
}

// Example: _ = medium.DeleteAll("reports/2026")
func (medium *Medium) DeleteAll(filePath string) error {
	key := medium.objectKey(filePath)
	if key == "" {
		return core.E("s3.DeleteAll", "path is required", fs.ErrInvalid)
	}

	// First, try deleting the exact key
	_, err := medium.client.DeleteObject(context.Background(), &awss3.DeleteObjectInput{
		Bucket: aws.String(medium.bucket),
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

	continueListing := true
	var continuationToken *string

	for continueListing {
		listOutput, err := medium.client.ListObjectsV2(context.Background(), &awss3.ListObjectsV2Input{
			Bucket:            aws.String(medium.bucket),
			Prefix:            aws.String(prefix),
			ContinuationToken: continuationToken,
		})
		if err != nil {
			return core.E("s3.DeleteAll", core.Concat("failed to list objects: ", prefix), err)
		}

		if len(listOutput.Contents) == 0 {
			break
		}

		objects := make([]types.ObjectIdentifier, len(listOutput.Contents))
		for i, object := range listOutput.Contents {
			objects[i] = types.ObjectIdentifier{Key: object.Key}
		}

		deleteOut, err := medium.client.DeleteObjects(context.Background(), &awss3.DeleteObjectsInput{
			Bucket: aws.String(medium.bucket),
			Delete: &types.Delete{Objects: objects, Quiet: aws.Bool(true)},
		})
		if err != nil {
			return core.E("s3.DeleteAll", "failed to delete objects", err)
		}
		if err := deleteObjectsError(prefix, deleteOut.Errors); err != nil {
			return err
		}

		if listOutput.IsTruncated != nil && *listOutput.IsTruncated {
			continuationToken = listOutput.NextContinuationToken
		} else {
			continueListing = false
		}
	}

	return nil
}

// Example: _ = medium.Rename("drafts/todo.txt", "archive/todo.txt")
func (medium *Medium) Rename(oldPath, newPath string) error {
	oldKey := medium.objectKey(oldPath)
	newKey := medium.objectKey(newPath)
	if oldKey == "" || newKey == "" {
		return core.E("s3.Rename", "both old and new paths are required", fs.ErrInvalid)
	}

	copySource := medium.bucket + "/" + oldKey

	_, err := medium.client.CopyObject(context.Background(), &awss3.CopyObjectInput{
		Bucket:     aws.String(medium.bucket),
		CopySource: aws.String(copySource),
		Key:        aws.String(newKey),
	})
	if err != nil {
		return core.E("s3.Rename", core.Concat("failed to copy object: ", oldKey, " -> ", newKey), err)
	}

	_, err = medium.client.DeleteObject(context.Background(), &awss3.DeleteObjectInput{
		Bucket: aws.String(medium.bucket),
		Key:    aws.String(oldKey),
	})
	if err != nil {
		return core.E("s3.Rename", core.Concat("failed to delete source object: ", oldKey), err)
	}

	return nil
}

// Example: entries, _ := medium.List("reports")
func (medium *Medium) List(filePath string) ([]fs.DirEntry, error) {
	prefix := medium.objectKey(filePath)
	if prefix != "" && !core.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	var entries []fs.DirEntry

	listOutput, err := medium.client.ListObjectsV2(context.Background(), &awss3.ListObjectsV2Input{
		Bucket:    aws.String(medium.bucket),
		Prefix:    aws.String(prefix),
		Delimiter: aws.String("/"),
	})
	if err != nil {
		return nil, core.E("s3.List", core.Concat("failed to list objects: ", prefix), err)
	}

	// Common prefixes are "directories"
	for _, commonPrefix := range listOutput.CommonPrefixes {
		if commonPrefix.Prefix == nil {
			continue
		}
		name := core.TrimPrefix(*commonPrefix.Prefix, prefix)
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
	for _, object := range listOutput.Contents {
		if object.Key == nil {
			continue
		}
		name := core.TrimPrefix(*object.Key, prefix)
		if name == "" || core.Contains(name, "/") {
			continue
		}
		var size int64
		if object.Size != nil {
			size = *object.Size
		}
		var modTime time.Time
		if object.LastModified != nil {
			modTime = *object.LastModified
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
func (medium *Medium) Stat(filePath string) (fs.FileInfo, error) {
	key := medium.objectKey(filePath)
	if key == "" {
		return nil, core.E("s3.Stat", "path is required", fs.ErrInvalid)
	}

	out, err := medium.client.HeadObject(context.Background(), &awss3.HeadObjectInput{
		Bucket: aws.String(medium.bucket),
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

func (medium *Medium) Open(filePath string) (fs.File, error) {
	key := medium.objectKey(filePath)
	if key == "" {
		return nil, core.E("s3.Open", "path is required", fs.ErrInvalid)
	}

	out, err := medium.client.GetObject(context.Background(), &awss3.GetObjectInput{
		Bucket: aws.String(medium.bucket),
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
func (medium *Medium) Create(filePath string) (goio.WriteCloser, error) {
	key := medium.objectKey(filePath)
	if key == "" {
		return nil, core.E("s3.Create", "path is required", fs.ErrInvalid)
	}
	return &s3WriteCloser{
		medium: medium,
		key:    key,
	}, nil
}

// Example: writer, _ := medium.Append("reports/daily.txt")
func (medium *Medium) Append(filePath string) (goio.WriteCloser, error) {
	key := medium.objectKey(filePath)
	if key == "" {
		return nil, core.E("s3.Append", "path is required", fs.ErrInvalid)
	}

	var existing []byte
	out, err := medium.client.GetObject(context.Background(), &awss3.GetObjectInput{
		Bucket: aws.String(medium.bucket),
		Key:    aws.String(key),
	})
	if err == nil {
		existing, _ = goio.ReadAll(out.Body)
		out.Body.Close()
	}

	return &s3WriteCloser{
		medium: medium,
		key:    key,
		data:   existing,
	}, nil
}

func (medium *Medium) ReadStream(filePath string) (goio.ReadCloser, error) {
	key := medium.objectKey(filePath)
	if key == "" {
		return nil, core.E("s3.ReadStream", "path is required", fs.ErrInvalid)
	}

	out, err := medium.client.GetObject(context.Background(), &awss3.GetObjectInput{
		Bucket: aws.String(medium.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, core.E("s3.ReadStream", core.Concat("failed to get object: ", key), err)
	}
	return out.Body, nil
}

func (medium *Medium) WriteStream(filePath string) (goio.WriteCloser, error) {
	return medium.Create(filePath)
}

// Example: ok := medium.Exists("reports/daily.txt")
func (medium *Medium) Exists(filePath string) bool {
	key := medium.objectKey(filePath)
	if key == "" {
		return false
	}

	// Check as an exact object
	_, err := medium.client.HeadObject(context.Background(), &awss3.HeadObjectInput{
		Bucket: aws.String(medium.bucket),
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
	listOutput, err := medium.client.ListObjectsV2(context.Background(), &awss3.ListObjectsV2Input{
		Bucket:  aws.String(medium.bucket),
		Prefix:  aws.String(prefix),
		MaxKeys: aws.Int32(1),
	})
	if err != nil {
		return false
	}
	return len(listOutput.Contents) > 0 || len(listOutput.CommonPrefixes) > 0
}

// Example: ok := medium.IsDir("reports")
func (medium *Medium) IsDir(filePath string) bool {
	key := medium.objectKey(filePath)
	if key == "" {
		return false
	}

	prefix := key
	if !core.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	listOutput, err := medium.client.ListObjectsV2(context.Background(), &awss3.ListObjectsV2Input{
		Bucket:  aws.String(medium.bucket),
		Prefix:  aws.String(prefix),
		MaxKeys: aws.Int32(1),
	})
	if err != nil {
		return false
	}
	return len(listOutput.Contents) > 0 || len(listOutput.CommonPrefixes) > 0
}

// --- Internal types ---

type fileInfo struct {
	name    string
	size    int64
	mode    fs.FileMode
	modTime time.Time
	isDir   bool
}

func (info *fileInfo) Name() string { return info.name }

func (info *fileInfo) Size() int64 { return info.size }

func (info *fileInfo) Mode() fs.FileMode { return info.mode }

func (info *fileInfo) ModTime() time.Time { return info.modTime }

func (info *fileInfo) IsDir() bool { return info.isDir }

func (info *fileInfo) Sys() any { return nil }

type dirEntry struct {
	name  string
	isDir bool
	mode  fs.FileMode
	info  fs.FileInfo
}

func (entry *dirEntry) Name() string { return entry.name }

func (entry *dirEntry) IsDir() bool { return entry.isDir }

func (entry *dirEntry) Type() fs.FileMode { return entry.mode.Type() }

func (entry *dirEntry) Info() (fs.FileInfo, error) { return entry.info, nil }

type s3File struct {
	name    string
	content []byte
	offset  int64
	size    int64
	modTime time.Time
}

func (file *s3File) Stat() (fs.FileInfo, error) {
	return &fileInfo{
		name:    file.name,
		size:    int64(len(file.content)),
		mode:    0644,
		modTime: file.modTime,
	}, nil
}

func (file *s3File) Read(buffer []byte) (int, error) {
	if file.offset >= int64(len(file.content)) {
		return 0, goio.EOF
	}
	bytesRead := copy(buffer, file.content[file.offset:])
	file.offset += int64(bytesRead)
	return bytesRead, nil
}

func (file *s3File) Close() error {
	return nil
}

type s3WriteCloser struct {
	medium *Medium
	key    string
	data   []byte
}

func (writer *s3WriteCloser) Write(data []byte) (int, error) {
	writer.data = append(writer.data, data...)
	return len(data), nil
}

func (writer *s3WriteCloser) Close() error {
	_, err := writer.medium.client.PutObject(context.Background(), &awss3.PutObjectInput{
		Bucket: aws.String(writer.medium.bucket),
		Key:    aws.String(writer.key),
		Body:   bytes.NewReader(writer.data),
	})
	if err != nil {
		return core.E("s3.writeCloser.Close", "failed to upload on close", err)
	}
	return nil
}
