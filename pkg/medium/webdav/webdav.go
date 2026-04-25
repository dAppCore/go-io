package webdav

import (
	"bytes"
	"cmp"
	"encoding/xml"
	"errors"
	"fmt"
	goio "io"
	"io/fs"
	"net/http"
	"net/url"
	"path"
	"slices"
	"strconv"
	"strings"
	"time"

	core "dappco.re/go/core"
	coreio "dappco.re/go/io"
)

const propfindBody = `<?xml version="1.0" encoding="utf-8"?>
<propfind xmlns="DAV:">
  <prop>
    <displayname/>
    <getcontentlength/>
    <getlastmodified/>
    <resourcetype/>
  </prop>
</propfind>`

// Medium is a WebDAV-backed implementation of coreio.Medium.
type Medium struct {
	baseURL  *url.URL
	client   *http.Client
	username string
	password string
	headers  http.Header
}

var _ coreio.Medium = (*Medium)(nil)

// Options configures a WebDAV Medium.
type Options struct {
	BaseURL  string
	Client   *http.Client
	Username string
	Password string
	Header   http.Header
}

// New creates a WebDAV Medium.
func New(options Options) (*Medium, error) {
	if options.BaseURL == "" {
		return nil, core.E("webdav.New", "base URL is required", fs.ErrInvalid)
	}

	baseURL, err := url.Parse(options.BaseURL)
	if err != nil {
		return nil, core.E("webdav.New", "base URL is invalid", err)
	}
	if baseURL.Scheme == "" || baseURL.Host == "" {
		return nil, core.E("webdav.New", "base URL must include scheme and host", fs.ErrInvalid)
	}

	client := options.Client
	if client == nil {
		client = http.DefaultClient
	}

	return &Medium{
		baseURL:  baseURL,
		client:   client,
		username: options.Username,
		password: options.Password,
		headers:  options.Header.Clone(),
	}, nil
}

func cleanRelative(filePath string) string {
	clean := path.Clean("/" + strings.ReplaceAll(filePath, "\\", "/"))
	if clean == "/" {
		return ""
	}
	return strings.TrimPrefix(clean, "/")
}

func (medium *Medium) resourceURL(filePath string) string {
	u := *medium.baseURL
	basePath := strings.TrimSuffix(u.Path, "/")
	relativePath := cleanRelative(filePath)
	if relativePath == "" {
		if basePath == "" {
			u.Path = "/"
			u.RawPath = ""
		}
		return u.String()
	}
	if basePath == "" {
		u.Path = "/" + relativePath
		u.RawPath = ""
		return u.String()
	}
	u.Path = basePath + "/" + relativePath
	u.RawPath = ""
	return u.String()
}

func (medium *Medium) requiredResourceURL(operation, filePath string) (string, error) {
	if cleanRelative(filePath) == "" {
		return "", core.E(operation, "path is required", fs.ErrInvalid)
	}
	return medium.resourceURL(filePath), nil
}

func (medium *Medium) newRequest(method, filePath string, body goio.Reader) (*http.Request, error) {
	request, err := http.NewRequest(method, medium.resourceURL(filePath), body)
	if err != nil {
		return nil, err
	}
	for key, values := range medium.headers {
		for _, value := range values {
			request.Header.Add(key, value)
		}
	}
	if medium.username != "" || medium.password != "" {
		request.SetBasicAuth(medium.username, medium.password)
	}
	return request, nil
}

func (medium *Medium) do(method, filePath string, body goio.Reader) (*http.Response, error) {
	request, err := medium.newRequest(method, filePath, body)
	if err != nil {
		return nil, err
	}
	return medium.client.Do(request)
}

func statusError(operation, resource string, statusCode int) error {
	switch statusCode {
	case http.StatusNotFound:
		return core.E(operation, core.Concat("not found: ", resource), fs.ErrNotExist)
	case http.StatusConflict:
		return core.E(operation, core.Concat("conflict: ", resource), fs.ErrInvalid)
	case http.StatusMethodNotAllowed:
		return core.E(operation, core.Concat("method not allowed: ", resource), fs.ErrExist)
	default:
		return core.E(operation, fmt.Sprintf("unexpected HTTP status %d for %s", statusCode, resource), nil)
	}
}

func statusOK(statusCode int, allowed ...int) bool {
	for _, code := range allowed {
		if statusCode == code {
			return true
		}
	}
	return false
}

func (medium *Medium) putBytes(filePath string, data []byte) error {
	resource, err := medium.requiredResourceURL("webdav.WriteMode", filePath)
	if err != nil {
		return err
	}
	if err := medium.ensureParent(filePath); err != nil {
		return err
	}

	response, err := medium.do(http.MethodPut, filePath, bytes.NewReader(data))
	if err != nil {
		return core.E("webdav.WriteMode", core.Concat("PUT failed: ", resource), err)
	}
	defer response.Body.Close()
	if !statusOK(response.StatusCode, http.StatusOK, http.StatusCreated, http.StatusNoContent) {
		return statusError("webdav.WriteMode", resource, response.StatusCode)
	}
	return nil
}

func (medium *Medium) ensureParent(filePath string) error {
	relative := cleanRelative(filePath)
	parent := path.Dir(relative)
	if parent == "." || parent == "" {
		return nil
	}
	return medium.EnsureDir(parent)
}

// Read reads a WebDAV resource into a string.
func (medium *Medium) Read(filePath string) (string, error) {
	resource, err := medium.requiredResourceURL("webdav.Read", filePath)
	if err != nil {
		return "", err
	}
	response, err := medium.do(http.MethodGet, filePath, nil)
	if err != nil {
		return "", core.E("webdav.Read", core.Concat("GET failed: ", resource), err)
	}
	defer response.Body.Close()
	if !statusOK(response.StatusCode, http.StatusOK) {
		return "", statusError("webdav.Read", resource, response.StatusCode)
	}
	data, err := goio.ReadAll(response.Body)
	if err != nil {
		return "", core.E("webdav.Read", core.Concat("read body failed: ", resource), err)
	}
	return string(data), nil
}

// Write writes a WebDAV resource using the default file mode.
func (medium *Medium) Write(filePath, content string) error {
	return medium.WriteMode(filePath, content, 0644)
}

// WriteMode writes a WebDAV resource. The mode is intentionally ignored
// because WebDAV has no portable POSIX permission model.
func (medium *Medium) WriteMode(filePath, content string, mode fs.FileMode) error {
	return medium.putBytes(filePath, []byte(content))
}

// EnsureDir creates a WebDAV collection and any missing parent collections.
func (medium *Medium) EnsureDir(filePath string) error {
	relative := cleanRelative(filePath)
	if relative == "" {
		return nil
	}

	current := ""
	for _, part := range strings.Split(relative, "/") {
		if current == "" {
			current = part
		} else {
			current = path.Join(current, part)
		}
		if err := medium.mkcol(current); err != nil {
			return err
		}
	}
	return nil
}

func (medium *Medium) mkcol(filePath string) error {
	resource := medium.resourceURL(filePath)
	response, err := medium.do("MKCOL", filePath, nil)
	if err != nil {
		return core.E("webdav.EnsureDir", core.Concat("MKCOL failed: ", resource), err)
	}
	defer response.Body.Close()
	switch response.StatusCode {
	case http.StatusCreated, http.StatusOK, http.StatusNoContent:
		return nil
	case http.StatusMethodNotAllowed:
		if medium.IsDir(filePath) {
			return nil
		}
		return statusError("webdav.EnsureDir", resource, response.StatusCode)
	default:
		return statusError("webdav.EnsureDir", resource, response.StatusCode)
	}
}

// IsFile reports whether filePath exists and is not a collection.
func (medium *Medium) IsFile(filePath string) bool {
	if cleanRelative(filePath) == "" {
		return false
	}
	info, err := medium.Stat(filePath)
	return err == nil && !info.IsDir()
}

// Delete removes a file or empty collection.
func (medium *Medium) Delete(filePath string) error {
	resource, err := medium.requiredResourceURL("webdav.Delete", filePath)
	if err != nil {
		return err
	}
	info, err := medium.Stat(filePath)
	if err != nil {
		return err
	}
	if info.IsDir() {
		entries, err := medium.List(filePath)
		if err != nil {
			return err
		}
		if len(entries) > 0 {
			return core.E("webdav.Delete", core.Concat("collection not empty: ", resource), fs.ErrExist)
		}
	}

	response, err := medium.do(http.MethodDelete, filePath, nil)
	if err != nil {
		return core.E("webdav.Delete", core.Concat("DELETE failed: ", resource), err)
	}
	defer response.Body.Close()
	if !statusOK(response.StatusCode, http.StatusOK, http.StatusAccepted, http.StatusNoContent) {
		return statusError("webdav.Delete", resource, response.StatusCode)
	}
	return nil
}

// DeleteAll removes a file or collection tree.
func (medium *Medium) DeleteAll(filePath string) error {
	resource, err := medium.requiredResourceURL("webdav.DeleteAll", filePath)
	if err != nil {
		return err
	}
	response, err := medium.do(http.MethodDelete, filePath, nil)
	if err != nil {
		return core.E("webdav.DeleteAll", core.Concat("DELETE failed: ", resource), err)
	}
	defer response.Body.Close()
	if !statusOK(response.StatusCode, http.StatusOK, http.StatusAccepted, http.StatusNoContent) {
		return statusError("webdav.DeleteAll", resource, response.StatusCode)
	}
	return nil
}

// Rename moves a WebDAV resource to a new path.
func (medium *Medium) Rename(oldPath, newPath string) error {
	source, err := medium.requiredResourceURL("webdav.Rename", oldPath)
	if err != nil {
		return err
	}
	destination, err := medium.requiredResourceURL("webdav.Rename", newPath)
	if err != nil {
		return err
	}
	if err := medium.ensureParent(newPath); err != nil {
		return err
	}

	request, err := medium.newRequest("MOVE", oldPath, nil)
	if err != nil {
		return core.E("webdav.Rename", "failed to build MOVE request", err)
	}
	request.Header.Set("Destination", destination)
	request.Header.Set("Overwrite", "T")

	response, err := medium.client.Do(request)
	if err != nil {
		return core.E("webdav.Rename", core.Concat("MOVE failed: ", source), err)
	}
	defer response.Body.Close()
	if !statusOK(response.StatusCode, http.StatusCreated, http.StatusNoContent) {
		return statusError("webdav.Rename", source, response.StatusCode)
	}
	return nil
}

// List returns the immediate children under a WebDAV collection.
func (medium *Medium) List(filePath string) ([]fs.DirEntry, error) {
	responses, requestPath, err := medium.propfind(filePath, "1")
	if err != nil {
		return nil, err
	}

	var entries []fs.DirEntry
	for _, response := range responses {
		hrefPath := hrefToPath(response.Href)
		if sameURLPath(hrefPath, requestPath) {
			continue
		}
		info := response.fileInfo(hrefPath)
		entries = append(entries, fs.FileInfoToDirEntry(info))
	}
	slices.SortFunc(entries, func(a, b fs.DirEntry) int {
		return cmp.Compare(a.Name(), b.Name())
	})
	return entries, nil
}

// Stat returns metadata for a WebDAV resource.
func (medium *Medium) Stat(filePath string) (fs.FileInfo, error) {
	resource, err := medium.requiredResourceURL("webdav.Stat", filePath)
	if err != nil {
		return nil, err
	}
	responses, requestPath, err := medium.propfind(filePath, "0")
	if err != nil {
		return nil, err
	}
	if len(responses) == 0 {
		return nil, core.E("webdav.Stat", core.Concat("not found: ", resource), fs.ErrNotExist)
	}
	return responses[0].fileInfo(requestPath), nil
}

func (medium *Medium) propfind(filePath, depth string) ([]davResponse, string, error) {
	resource := medium.resourceURL(filePath)
	request, err := medium.newRequest("PROPFIND", filePath, strings.NewReader(propfindBody))
	if err != nil {
		return nil, "", core.E("webdav.propfind", "failed to build PROPFIND request", err)
	}
	request.Header.Set("Depth", depth)
	request.Header.Set("Content-Type", "application/xml; charset=utf-8")

	response, err := medium.client.Do(request)
	if err != nil {
		return nil, "", core.E("webdav.propfind", core.Concat("PROPFIND failed: ", resource), err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusMultiStatus {
		return nil, "", statusError("webdav.propfind", resource, response.StatusCode)
	}

	var multistatus davMultiStatus
	if err := xml.NewDecoder(response.Body).Decode(&multistatus); err != nil {
		return nil, "", core.E("webdav.propfind", core.Concat("decode failed: ", resource), err)
	}
	return multistatus.Responses, hrefToPath(resource), nil
}

// Open opens a WebDAV resource as an fs.File.
func (medium *Medium) Open(filePath string) (fs.File, error) {
	content, err := medium.Read(filePath)
	if err != nil {
		return nil, err
	}
	info, err := medium.Stat(filePath)
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return nil, core.E("webdav.Open", core.Concat("path is a collection: ", filePath), fs.ErrInvalid)
	}
	return &webdavFile{
		name:    info.Name(),
		content: []byte(content),
		mode:    info.Mode(),
		modTime: info.ModTime(),
	}, nil
}

// Create opens a buffered WebDAV writer that replaces the resource on close.
func (medium *Medium) Create(filePath string) (goio.WriteCloser, error) {
	if _, err := medium.requiredResourceURL("webdav.Create", filePath); err != nil {
		return nil, err
	}
	return &webdavWriteCloser{medium: medium, path: filePath, mode: 0644}, nil
}

// Append opens a buffered WebDAV writer that appends locally then replaces the
// resource on close.
func (medium *Medium) Append(filePath string) (goio.WriteCloser, error) {
	if _, err := medium.requiredResourceURL("webdav.Append", filePath); err != nil {
		return nil, err
	}

	var existing []byte
	content, err := medium.Read(filePath)
	if err == nil {
		existing = []byte(content)
	} else if !errors.Is(err, fs.ErrNotExist) {
		return nil, core.E("webdav.Append", core.Concat("read existing failed: ", filePath), err)
	}

	return &webdavWriteCloser{medium: medium, path: filePath, data: existing, mode: 0644}, nil
}

// ReadStream opens a WebDAV resource as an io.ReadCloser.
func (medium *Medium) ReadStream(filePath string) (goio.ReadCloser, error) {
	resource, err := medium.requiredResourceURL("webdav.ReadStream", filePath)
	if err != nil {
		return nil, err
	}
	response, err := medium.do(http.MethodGet, filePath, nil)
	if err != nil {
		return nil, core.E("webdav.ReadStream", core.Concat("GET failed: ", resource), err)
	}
	if !statusOK(response.StatusCode, http.StatusOK) {
		response.Body.Close()
		return nil, statusError("webdav.ReadStream", resource, response.StatusCode)
	}
	return response.Body, nil
}

// WriteStream opens a buffered WebDAV writer that replaces the resource on close.
func (medium *Medium) WriteStream(filePath string) (goio.WriteCloser, error) {
	return medium.Create(filePath)
}

// Exists reports whether a WebDAV resource exists.
func (medium *Medium) Exists(filePath string) bool {
	if cleanRelative(filePath) == "" {
		return false
	}
	_, err := medium.Stat(filePath)
	return err == nil
}

// IsDir reports whether a WebDAV resource exists and is a collection.
func (medium *Medium) IsDir(filePath string) bool {
	if cleanRelative(filePath) == "" {
		return false
	}
	info, err := medium.Stat(filePath)
	return err == nil && info.IsDir()
}

type davMultiStatus struct {
	Responses []davResponse `xml:"response"`
}

type davResponse struct {
	Href      string        `xml:"href"`
	PropStats []davPropStat `xml:"propstat"`
}

type davPropStat struct {
	Prop   davProp `xml:"prop"`
	Status string  `xml:"status"`
}

type davProp struct {
	DisplayName   string          `xml:"displayname"`
	ContentLength string          `xml:"getcontentlength"`
	LastModified  string          `xml:"getlastmodified"`
	ResourceType  davResourceType `xml:"resourcetype"`
}

type davResourceType struct {
	Collection *struct{} `xml:"collection"`
}

func (response davResponse) prop() davProp {
	for _, propstat := range response.PropStats {
		if propstat.Status == "" || strings.Contains(propstat.Status, " 200 ") {
			return propstat.Prop
		}
	}
	if len(response.PropStats) > 0 {
		return response.PropStats[0].Prop
	}
	return davProp{}
}

func (response davResponse) fileInfo(fallbackPath string) fs.FileInfo {
	prop := response.prop()
	isDir := prop.ResourceType.Collection != nil
	size, _ := strconv.ParseInt(strings.TrimSpace(prop.ContentLength), 10, 64)
	modTime := time.Time{}
	if prop.LastModified != "" {
		if parsedTime, err := http.ParseTime(prop.LastModified); err == nil {
			modTime = parsedTime
		}
	}

	name := prop.DisplayName
	if name == "" {
		name = path.Base(strings.TrimSuffix(fallbackPath, "/"))
	}
	if name == "." || name == "/" {
		name = ""
	}

	mode := fs.FileMode(0644)
	if isDir {
		mode = fs.ModeDir | 0755
		size = 0
	}

	return coreio.NewFileInfo(name, size, mode, modTime, isDir)
}

func hrefToPath(href string) string {
	parsedURL, err := url.Parse(href)
	if err == nil && parsedURL.Path != "" {
		if unescaped, err := url.PathUnescape(parsedURL.Path); err == nil {
			return unescaped
		}
		return parsedURL.Path
	}
	if unescaped, err := url.PathUnescape(href); err == nil {
		return unescaped
	}
	return href
}

func sameURLPath(left, right string) bool {
	return path.Clean("/"+left) == path.Clean("/"+right)
}

type webdavFile struct {
	name    string
	content []byte
	offset  int64
	mode    fs.FileMode
	modTime time.Time
}

func (file *webdavFile) Stat() (fs.FileInfo, error) {
	return coreio.NewFileInfo(file.name, int64(len(file.content)), file.mode, file.modTime, false), nil
}

func (file *webdavFile) Read(buffer []byte) (int, error) {
	if file.offset >= int64(len(file.content)) {
		return 0, goio.EOF
	}
	bytesRead := copy(buffer, file.content[file.offset:])
	file.offset += int64(bytesRead)
	return bytesRead, nil
}

func (file *webdavFile) Close() error {
	return nil
}

type webdavWriteCloser struct {
	medium *Medium
	path   string
	data   []byte
	mode   fs.FileMode
}

func (writer *webdavWriteCloser) Write(data []byte) (int, error) {
	writer.data = append(writer.data, data...)
	return len(data), nil
}

func (writer *webdavWriteCloser) Close() error {
	return writer.medium.WriteMode(writer.path, string(writer.data), writer.mode)
}
