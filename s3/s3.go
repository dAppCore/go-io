// Example: client := awss3.NewFromConfig(aws.Config{Region: "us-east-1"})
// Example: medium, _ := s3.New(s3.Options{Bucket: "backups", Client: client, Prefix: "daily/"})
// Example: _ = medium.Write("reports/daily.txt", "done")
package s3

import (
	"context" // AX-6-exception: AWS SDK transport APIs require context.Context.
	goio "io" // AX-6-exception: io interface types have no core equivalent; io.EOF preserves stream semantics.
	"io/fs"   // AX-6-exception: fs interface types have no core equivalent.
	"time"    // AX-6-exception: S3 object metadata timestamps have no core equivalent.

	"github.com/aws/aws-sdk-go-v2/aws"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"

	core "dappco.re/go"
	coreio "dappco.re/go/io"
)

const (
	opS3Read      = "s3.Read"
	opS3DeleteAll = "s3.DeleteAll"
	opS3Rename    = "s3.Rename"
	opS3Open      = "s3.Open"

	msgS3PathRequired    = "path is required"
	msgS3GetObjectFailed = "failed to get object: "
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
var _ fs.FS = (*Medium)(nil)

// Example: medium, _ := s3.New(s3.Options{Bucket: "backups", Client: client, Prefix: "daily/"})
type Options struct {
	Bucket string
	Client Client
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
	return core.E(opS3DeleteAll, core.Concat("partial delete failed under ", prefix, ": ", core.Join("; ", details...)), nil)
}

func readAllString(reader goio.ReadCloser) (string, error) {
	defer closeS3Reader(reader)

	result := core.ReadAll(reader)
	if !result.OK {
		if err, ok := result.Value.(error); ok {
			return "", err
		}
		return "", fs.ErrInvalid
	}
	content, ok := result.Value.(string)
	if !ok {
		return "", fs.ErrInvalid
	}
	return content, nil
}

func closeS3Reader(reader goio.ReadCloser) {
	if err := reader.Close(); err != nil {
		core.Warn("s3 reader close failed", "err", err)
	}
}

func normalisePrefix(prefix string) string {
	if prefix == "" {
		return ""
	}
	clean := core.CleanPath("/"+prefix, "/")
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
		return nil, core.E("s3.New", "bucket name is required", fs.ErrInvalid)
	}
	if options.Client == nil {
		return nil, core.E("s3.New", "client is required", fs.ErrInvalid)
	}
	medium := &Medium{
		client: options.Client,
		bucket: options.Bucket,
		prefix: normalisePrefix(options.Prefix),
	}
	return medium, nil
}

func (medium *Medium) objectKey(filePath string) string {
	clean := core.CleanPath("/"+filePath, "/")
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

// Example: content, _ := medium.Read("reports/daily.txt")
func (medium *Medium) Read(filePath string) (string, error) {
	key := medium.objectKey(filePath)
	if key == "" {
		return "", core.E(opS3Read, msgS3PathRequired, fs.ErrInvalid)
	}

	out, err := medium.client.GetObject(context.Background(), &awss3.GetObjectInput{
		Bucket: aws.String(medium.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return "", core.E(opS3Read, core.Concat(msgS3GetObjectFailed, key), err)
	}
	data, err := readAllString(out.Body)
	if err != nil {
		return "", core.E(opS3Read, core.Concat("failed to read body: ", key), err)
	}
	return data, nil
}

// Example: _ = medium.Write("reports/daily.txt", "done")
func (medium *Medium) Write(filePath, content string) error {
	key := medium.objectKey(filePath)
	if key == "" {
		return core.E("s3.Write", msgS3PathRequired, fs.ErrInvalid)
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
// Note: mode is intentionally ignored — S3 has no POSIX permission model.
// Use S3 bucket policies and IAM for access control.
func (medium *Medium) WriteMode(filePath, content string, mode fs.FileMode) error {
	return medium.Write(filePath, content)
}

// Example: _ = medium.EnsureDir("reports/2026")
func (medium *Medium) EnsureDir(directoryPath string) error {
	return nil
}

// Example: isFile := medium.IsFile("reports/daily.txt")
func (medium *Medium) IsFile(filePath string) bool {
	key := medium.objectKey(filePath)
	if key == "" {
		return false
	}
	if core.HasSuffix(key, "/") {
		return false
	}
	_, err := medium.client.HeadObject(context.Background(), &awss3.HeadObjectInput{
		Bucket: aws.String(medium.bucket),
		Key:    aws.String(key),
	})
	return err == nil
}

// Example: _ = medium.Delete("reports/daily.txt")
func (medium *Medium) Delete(filePath string) error {
	key := medium.objectKey(filePath)
	if key == "" {
		return core.E("s3.Delete", msgS3PathRequired, fs.ErrInvalid)
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
		return core.E(opS3DeleteAll, msgS3PathRequired, fs.ErrInvalid)
	}

	_, err := medium.client.DeleteObject(context.Background(), &awss3.DeleteObjectInput{
		Bucket: aws.String(medium.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return core.E(opS3DeleteAll, core.Concat("failed to delete object: ", key), err)
	}

	prefix := key
	if !core.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	var continuationToken *string
	for {
		nextToken, keepGoing, err := medium.deleteObjectBatch(prefix, continuationToken)
		if err != nil {
			return err
		}
		if !keepGoing {
			break
		}
		continuationToken = nextToken
	}

	return nil
}

func (medium *Medium) deleteObjectBatch(prefix string, continuationToken *string) (*string, bool, error) {
	listOutput, err := medium.client.ListObjectsV2(context.Background(), &awss3.ListObjectsV2Input{
		Bucket:            aws.String(medium.bucket),
		Prefix:            aws.String(prefix),
		ContinuationToken: continuationToken,
	})
	if err != nil {
		return nil, false, core.E(opS3DeleteAll, core.Concat("failed to list objects: ", prefix), err)
	}
	if len(listOutput.Contents) == 0 {
		return nil, false, nil
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
		return nil, false, core.E(opS3DeleteAll, "failed to delete objects", err)
	}
	if err := deleteObjectsError(prefix, deleteOut.Errors); err != nil {
		return nil, false, err
	}
	if listOutput.IsTruncated != nil && *listOutput.IsTruncated {
		return listOutput.NextContinuationToken, true, nil
	}
	return nil, false, nil
}

// Example: _ = medium.Rename("drafts/todo.txt", "archive/todo.txt")
func (medium *Medium) Rename(oldPath, newPath string) error {
	oldKey := medium.objectKey(oldPath)
	newKey := medium.objectKey(newPath)
	if oldKey == "" || newKey == "" {
		return core.E(opS3Rename, "both old and new paths are required", fs.ErrInvalid)
	}

	copySource := medium.bucket + "/" + oldKey

	_, err := medium.client.CopyObject(context.Background(), &awss3.CopyObjectInput{
		Bucket:     aws.String(medium.bucket),
		CopySource: aws.String(copySource),
		Key:        aws.String(newKey),
	})
	if err != nil {
		return core.E(opS3Rename, core.Concat("failed to copy object: ", oldKey, " -> ", newKey), err)
	}

	_, err = medium.client.DeleteObject(context.Background(), &awss3.DeleteObjectInput{
		Bucket: aws.String(medium.bucket),
		Key:    aws.String(oldKey),
	})
	if err != nil {
		return core.E(opS3Rename, core.Concat("failed to delete source object: ", oldKey), err)
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

	appendCommonPrefixEntries(prefix, listOutput.CommonPrefixes, &entries)
	appendObjectEntries(prefix, listOutput.Contents, &entries)

	return entries, nil
}

func appendCommonPrefixEntries(prefix string, commonPrefixes []types.CommonPrefix, entries *[]fs.DirEntry) {
	for _, commonPrefix := range commonPrefixes {
		if commonPrefix.Prefix == nil {
			continue
		}
		name := core.TrimPrefix(*commonPrefix.Prefix, prefix)
		name = core.TrimSuffix(name, "/")
		if name == "" {
			continue
		}
		*entries = append(*entries, s3DirectoryEntry(name))
	}
}

func appendObjectEntries(prefix string, objects []types.Object, entries *[]fs.DirEntry) {
	for _, object := range objects {
		if object.Key == nil {
			continue
		}
		name := core.TrimPrefix(*object.Key, prefix)
		if name == "" || core.Contains(name, "/") {
			continue
		}
		*entries = append(*entries, s3ObjectEntry(name, object))
	}
}

func s3DirectoryEntry(name string) fs.DirEntry {
	return &dirEntry{
		name:  name,
		isDir: true,
		mode:  fs.ModeDir | 0755,
		info: &fileInfo{
			name:  name,
			isDir: true,
			mode:  fs.ModeDir | 0755,
		},
	}
}

func s3ObjectEntry(name string, object types.Object) fs.DirEntry {
	var size int64
	if object.Size != nil {
		size = *object.Size
	}
	var modTime time.Time
	if object.LastModified != nil {
		modTime = *object.LastModified
	}
	return &dirEntry{
		name:  name,
		isDir: false,
		mode:  0644,
		info: &fileInfo{
			name:    name,
			size:    size,
			mode:    0644,
			modTime: modTime,
		},
	}
}

// Example: info, _ := medium.Stat("reports/daily.txt")
func (medium *Medium) Stat(filePath string) (fs.FileInfo, error) {
	key := medium.objectKey(filePath)
	if key == "" {
		return nil, core.E("s3.Stat", msgS3PathRequired, fs.ErrInvalid)
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

	name := core.PathBase(key)
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
		return nil, core.E(opS3Open, msgS3PathRequired, fs.ErrInvalid)
	}

	out, err := medium.client.GetObject(context.Background(), &awss3.GetObjectInput{
		Bucket: aws.String(medium.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, core.E(opS3Open, core.Concat(msgS3GetObjectFailed, key), err)
	}

	data, err := readAllString(out.Body)
	if err != nil {
		return nil, core.E(opS3Open, core.Concat("failed to read body: ", key), err)
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
		name:    core.PathBase(key),
		content: []byte(data),
		size:    size,
		modTime: modTime,
	}, nil
}

// Example: writer, _ := medium.Create("reports/daily.txt")
func (medium *Medium) Create(filePath string) (goio.WriteCloser, error) {
	key := medium.objectKey(filePath)
	if key == "" {
		return nil, core.E("s3.Create", msgS3PathRequired, fs.ErrInvalid)
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
		return nil, core.E("s3.Append", msgS3PathRequired, fs.ErrInvalid)
	}

	var existing []byte
	out, err := medium.client.GetObject(context.Background(), &awss3.GetObjectInput{
		Bucket: aws.String(medium.bucket),
		Key:    aws.String(key),
	})
	if err == nil {
		content, readErr := readAllString(out.Body)
		if readErr != nil {
			return nil, core.E("s3.Append", core.Concat("failed to read existing object: ", key), readErr)
		}
		existing = []byte(content)
	}

	return &s3WriteCloser{
		medium: medium,
		key:    key,
		data:   existing,
	}, nil
}

// Example: reader, _ := medium.ReadStream("reports/daily.txt")
func (medium *Medium) ReadStream(filePath string) (goio.ReadCloser, error) {
	key := medium.objectKey(filePath)
	if key == "" {
		return nil, core.E("s3.ReadStream", msgS3PathRequired, fs.ErrInvalid)
	}

	out, err := medium.client.GetObject(context.Background(), &awss3.GetObjectInput{
		Bucket: aws.String(medium.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, core.E("s3.ReadStream", core.Concat(msgS3GetObjectFailed, key), err)
	}
	return out.Body, nil
}

// Example: writer, _ := medium.WriteStream("reports/daily.txt")
func (medium *Medium) WriteStream(filePath string) (goio.WriteCloser, error) {
	return medium.Create(filePath)
}

// Example: exists := medium.Exists("reports/daily.txt")
func (medium *Medium) Exists(filePath string) bool {
	key := medium.objectKey(filePath)
	if key == "" {
		return false
	}

	_, err := medium.client.HeadObject(context.Background(), &awss3.HeadObjectInput{
		Bucket: aws.String(medium.bucket),
		Key:    aws.String(key),
	})
	if err == nil {
		return true
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

// Example: isDirectory := medium.IsDir("reports")
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
		Body:   core.NewReader(string(writer.data)),
	})
	if err != nil {
		return core.E("s3.writeCloser.Close", "failed to upload on close", err)
	}
	return nil
}
