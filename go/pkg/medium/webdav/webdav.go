package webdav

import (
	"cmp"
	"encoding/xml"
	goio "io"
	"io/fs"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"time"

	core "dappco.re/go"
	coreio "dappco.re/go/io"
)

const (
	defaultHTTPTimeout = 30 * time.Second

	opNew        = "webdav.New"
	opRead       = "webdav.Read"
	opWriteMode  = "webdav.WriteMode"
	opEnsureDir  = "webdav.EnsureDir"
	opDelete     = "webdav.Delete"
	opDeleteAll  = "webdav.DeleteAll"
	opRename     = "webdav.Rename"
	opPropfind   = "webdav.propfind"
	opReadStream = "webdav.ReadStream"
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
var _ fs.FS = (*Medium)(nil)

// Options configures a WebDAV Medium.
type Options struct {
	BaseURL  string
	Client   *http.Client
	Username string
	Password string
	Header   http.Header
}

// New creates a WebDAV Medium.
func New(options Options) (
	*Medium,
	error,
) {
	if options.BaseURL == "" {
		return nil, core.E(opNew, "base URL is required", fs.ErrInvalid)
	}

	baseURL, err := url.Parse(options.BaseURL)
	if err != nil {
		return nil, core.E(opNew, "base URL is invalid", err)
	}
	if baseURL.Scheme == "" || baseURL.Host == "" {
		return nil, core.E(opNew, "base URL must include scheme and host", fs.ErrInvalid)
	}

	client := options.Client
	if client == nil {
		client = &http.Client{Timeout: defaultHTTPTimeout}
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
	clean := core.CleanPath("/"+core.Replace(filePath, "\\", "/"), "/")
	if clean == "/" {
		return ""
	}
	return core.TrimPrefix(clean, "/")
}

func (medium *Medium) resourceURL(filePath string) string {
	u := *medium.baseURL
	basePath := core.TrimSuffix(u.Path, "/")
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

func (medium *Medium) requiredResourceURL(operation, filePath string) (
	string,
	error,
) {
	if cleanRelative(filePath) == "" {
		return "", core.E(operation, "path is required", fs.ErrInvalid)
	}
	return medium.resourceURL(filePath), nil
}

func (medium *Medium) newRequest(method, filePath string, body goio.Reader) (
	*http.Request,
	error,
) {
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

func (medium *Medium) do(method, filePath string, body goio.Reader) (
	*http.Response,
	error,
) {
	request, err := medium.newRequest(method, filePath, body)
	if err != nil {
		return nil, err
	}
	return medium.client.Do(request)
}

func statusError(operation, resource string, statusCode int) error { // legacy error contract

	switch statusCode {
	case http.StatusNotFound:
		return core.E(operation, core.Concat("not found: ", resource), fs.ErrNotExist)
	case http.StatusConflict:
		return core.E(operation, core.Concat("conflict: ", resource), fs.ErrInvalid)
	case http.StatusMethodNotAllowed:
		return core.E(operation, core.Concat("method not allowed: ", resource), fs.ErrExist)
	default:
		return core.E(operation, core.Sprintf("unexpected HTTP status %d for %s", statusCode, resource), nil)
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

func closeWebDAVBody(closer goio.Closer) {
	if err := closer.Close(); err != nil {
		core.Warn("webdav response close failed", "err", err)
	}
}

func (medium *Medium) putBytes(filePath string, data []byte) error { // legacy error contract

	resource, err := medium.requiredResourceURL(opWriteMode, filePath)
	if err != nil {
		return err
	}
	if err := medium.ensureParent(filePath); err != nil {
		return err
	}

	response, err := medium.do(http.MethodPut, filePath, core.NewReader(string(data)))
	if err != nil {
		return core.E(opWriteMode, core.Concat("PUT failed: ", resource), err)
	}
	defer closeWebDAVBody(response.Body)
	if !statusOK(response.StatusCode, http.StatusOK, http.StatusCreated, http.StatusNoContent) {
		return statusError(opWriteMode, resource, response.StatusCode)
	}
	return nil
}

func (medium *Medium) ensureParent(filePath string) error { // legacy error contract

	relative := cleanRelative(filePath)
	parent := core.PathDir(relative)
	if parent == "." || parent == "" {
		return nil
	}
	return medium.EnsureDir(parent)
}

// Read reads a WebDAV resource into a string.
func (medium *Medium) Read(filePath string) (
	string,
	error,
) {
	resource, err := medium.requiredResourceURL(opRead, filePath)
	if err != nil {
		return "", err
	}
	response, err := medium.do(http.MethodGet, filePath, nil)
	if err != nil {
		return "", core.E(opRead, core.Concat("GET failed: ", resource), err)
	}
	defer closeWebDAVBody(response.Body)
	if !statusOK(response.StatusCode, http.StatusOK) {
		return "", statusError(opRead, resource, response.StatusCode)
	}
	data, err := goio.ReadAll(response.Body)
	if err != nil {
		return "", core.E(opRead, core.Concat("read body failed: ", resource), err)
	}
	return string(data), nil
}

// Write writes a WebDAV resource using the default file mode.
func (medium *Medium) Write(filePath, content string) error { // legacy error contract

	return medium.WriteMode(filePath, content, 0644)
}

// WriteMode writes a WebDAV resource. The mode is intentionally ignored
// because WebDAV has no portable POSIX permission model.
func (medium *Medium) WriteMode(filePath, content string, mode fs.FileMode) error { // legacy error contract

	return medium.putBytes(filePath, []byte(content))
}

// EnsureDir creates a WebDAV collection and any missing parent collections.
func (medium *Medium) EnsureDir(filePath string) error { // legacy error contract

	relative := cleanRelative(filePath)
	if relative == "" {
		return nil
	}

	current := ""
	for _, part := range core.Split(relative, "/") {
		if current == "" {
			current = part
		} else {
			current = core.PathJoin(current, part)
		}
		if err := medium.mkcol(current); err != nil {
			return err
		}
	}
	return nil
}

func (medium *Medium) mkcol(filePath string) error { // legacy error contract

	resource := medium.resourceURL(filePath)
	response, err := medium.do("MKCOL", filePath, nil)
	if err != nil {
		return core.E(opEnsureDir, core.Concat("MKCOL failed: ", resource), err)
	}
	defer closeWebDAVBody(response.Body)
	switch response.StatusCode {
	case http.StatusCreated, http.StatusOK, http.StatusNoContent:
		return nil
	case http.StatusMethodNotAllowed:
		if medium.IsDir(filePath) {
			return nil
		}
		return statusError(opEnsureDir, resource, response.StatusCode)
	default:
		return statusError(opEnsureDir, resource, response.StatusCode)
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
func (medium *Medium) Delete(filePath string) error { // legacy error contract

	resource, err := medium.requiredResourceURL(opDelete, filePath)
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
			return core.E(opDelete, core.Concat("collection not empty: ", resource), fs.ErrExist)
		}
	}

	response, err := medium.do(http.MethodDelete, filePath, nil)
	if err != nil {
		return core.E(opDelete, core.Concat("DELETE failed: ", resource), err)
	}
	defer closeWebDAVBody(response.Body)
	if !statusOK(response.StatusCode, http.StatusOK, http.StatusAccepted, http.StatusNoContent) {
		return statusError(opDelete, resource, response.StatusCode)
	}
	return nil
}

// DeleteAll removes a file or collection tree.
func (medium *Medium) DeleteAll(filePath string) error { // legacy error contract

	resource, err := medium.requiredResourceURL(opDeleteAll, filePath)
	if err != nil {
		return err
	}
	response, err := medium.do(http.MethodDelete, filePath, nil)
	if err != nil {
		return core.E(opDeleteAll, core.Concat("DELETE failed: ", resource), err)
	}
	defer closeWebDAVBody(response.Body)
	if !statusOK(response.StatusCode, http.StatusOK, http.StatusAccepted, http.StatusNoContent) {
		return statusError(opDeleteAll, resource, response.StatusCode)
	}
	return nil
}

// Rename moves a WebDAV resource to a new path.
func (medium *Medium) Rename(oldPath, newPath string) error { // legacy error contract

	source, err := medium.requiredResourceURL(opRename, oldPath)
	if err != nil {
		return err
	}
	destination, err := medium.requiredResourceURL(opRename, newPath)
	if err != nil {
		return err
	}
	if err := medium.ensureParent(newPath); err != nil {
		return err
	}

	request, err := medium.newRequest("MOVE", oldPath, nil)
	if err != nil {
		return core.E(opRename, "failed to build MOVE request", err)
	}
	request.Header.Set("Destination", destination)
	request.Header.Set("Overwrite", "T")

	response, err := medium.client.Do(request)
	if err != nil {
		return core.E(opRename, core.Concat("MOVE failed: ", source), err)
	}
	defer closeWebDAVBody(response.Body)
	if !statusOK(response.StatusCode, http.StatusCreated, http.StatusNoContent) {
		return statusError(opRename, source, response.StatusCode)
	}
	return nil
}

// List returns the immediate children under a WebDAV collection.
func (medium *Medium) List(filePath string) (
	[]fs.DirEntry,
	error,
) {
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
func (medium *Medium) Stat(filePath string) (
	fs.FileInfo,
	error,
) {
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

func (medium *Medium) propfind(filePath, depth string) (
	[]davResponse,
	string,
	error,
) {
	resource := medium.resourceURL(filePath)
	request, err := medium.newRequest("PROPFIND", filePath, core.NewReader(propfindBody))
	if err != nil {
		return nil, "", core.E(opPropfind, "failed to build PROPFIND request", err)
	}
	request.Header.Set("Depth", depth)
	request.Header.Set("Content-Type", "application/xml; charset=utf-8")

	response, err := medium.client.Do(request)
	if err != nil {
		return nil, "", core.E(opPropfind, core.Concat("PROPFIND failed: ", resource), err)
	}
	defer closeWebDAVBody(response.Body)
	if response.StatusCode != http.StatusMultiStatus {
		return nil, "", statusError(opPropfind, resource, response.StatusCode)
	}

	var multistatus davMultiStatus
	if err := xml.NewDecoder(response.Body).Decode(&multistatus); err != nil {
		return nil, "", core.E(opPropfind, core.Concat("decode failed: ", resource), err)
	}
	return multistatus.Responses, hrefToPath(resource), nil
}

// Open opens a WebDAV resource as an fs.File.
func (medium *Medium) Open(filePath string) (
	fs.File,
	error,
) {
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
func (medium *Medium) Create(filePath string) (
	goio.WriteCloser,
	error,
) {
	if _, err := medium.requiredResourceURL("webdav.Create", filePath); err != nil {
		return nil, err
	}
	return &webdavWriteCloser{medium: medium, path: filePath, mode: 0644}, nil
}

// Append opens a buffered WebDAV writer that appends locally then replaces the
// resource on close.
func (medium *Medium) Append(filePath string) (
	goio.WriteCloser,
	error,
) {
	if _, err := medium.requiredResourceURL("webdav.Append", filePath); err != nil {
		return nil, err
	}

	var existing []byte
	content, err := medium.Read(filePath)
	if err == nil {
		existing = []byte(content)
	} else if !core.Is(err, fs.ErrNotExist) {
		return nil, core.E("webdav.Append", core.Concat("read existing failed: ", filePath), err)
	}

	return &webdavWriteCloser{medium: medium, path: filePath, data: existing, mode: 0644}, nil
}

// ReadStream opens a WebDAV resource as an io.ReadCloser.
func (medium *Medium) ReadStream(filePath string) (
	goio.ReadCloser,
	error,
) {
	resource, err := medium.requiredResourceURL(opReadStream, filePath)
	if err != nil {
		return nil, err
	}
	response, err := medium.do(http.MethodGet, filePath, nil)
	if err != nil {
		return nil, core.E(opReadStream, core.Concat("GET failed: ", resource), err)
	}
	if !statusOK(response.StatusCode, http.StatusOK) {
		closeWebDAVBody(response.Body)
		return nil, statusError(opReadStream, resource, response.StatusCode)
	}
	return response.Body, nil
}

// WriteStream opens a buffered WebDAV writer that replaces the resource on close.
func (medium *Medium) WriteStream(filePath string) (
	goio.WriteCloser,
	error,
) {
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
		if propstat.Status == "" || core.Contains(propstat.Status, " 200 ") {
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
	size, _ := strconv.ParseInt(core.Trim(prop.ContentLength), 10, 64)
	modTime := time.Time{}
	if prop.LastModified != "" {
		if parsedTime, err := http.ParseTime(prop.LastModified); err == nil {
			modTime = parsedTime
		}
	}

	name := prop.DisplayName
	if name == "" {
		name = core.PathBase(core.TrimSuffix(fallbackPath, "/"))
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
	return core.CleanPath("/"+left, "/") == core.CleanPath("/"+right, "/")
}

type webdavFile struct {
	name    string
	content []byte
	offset  int64
	mode    fs.FileMode
	modTime time.Time
}

func (file *webdavFile) Stat() (
	fs.FileInfo,
	error,
) {
	return coreio.NewFileInfo(file.name, int64(len(file.content)), file.mode, file.modTime, false), nil
}

func (file *webdavFile) Read(buffer []byte) (
	int,
	error,
) {
	if file.offset >= int64(len(file.content)) {
		return 0, goio.EOF
	}
	bytesRead := copy(buffer, file.content[file.offset:])
	file.offset += int64(bytesRead)
	return bytesRead, nil
}

func (file *webdavFile) Close() error { // legacy error contract

	return nil
}

type webdavWriteCloser struct {
	medium *Medium
	path   string
	data   []byte
	mode   fs.FileMode
}

func (writer *webdavWriteCloser) Write(data []byte) (
	int,
	error,
) {
	writer.data = append(writer.data, data...)
	return len(data), nil
}

func (writer *webdavWriteCloser) Close() error { // legacy error contract

	return writer.medium.WriteMode(writer.path, string(writer.data), writer.mode)
}
