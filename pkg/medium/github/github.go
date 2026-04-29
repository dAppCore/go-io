package github

import (
	"context"
	goio "io"
	"io/fs"
	"net/http"
	"net/url"
	"slices"
	"time"

	core "dappco.re/go"
	coreio "dappco.re/go/io"
	borgdatanode "forge.lthn.ai/Snider/Borg/pkg/datanode"
	borgvcs "forge.lthn.ai/Snider/Borg/pkg/vcs"
	gh "github.com/google/go-github/v75/github"
	"golang.org/x/oauth2"
)

// ErrReadOnly is returned by all mutating operations on a GitHub Medium.
var ErrReadOnly = core.NewError("github medium is read-only")

const (
	opNew   = "github.New"
	opRead  = "github.Read"
	opList  = "github.List"
	opStat  = "github.Stat"
	opClone = "github.Clone"

	errNotFound = "not found: "
)

// Medium is a GitHub REST API-backed implementation of coreio.Medium.
type Medium struct {
	client  *gh.Client
	owner   string
	repo    string
	ref     string
	baseURL string
}

var _ coreio.Medium = (*Medium)(nil)
var _ fs.FS = (*Medium)(nil)

// Options configures a GitHub Medium.
type Options struct {
	Client     *gh.Client
	HTTPClient *http.Client
	Owner      string
	Repo       string
	Ref        string
	Branch     string
	Token      string
	TokenFile  string
	BaseURL    string
}

// New creates a GitHub Medium.
func New(options Options) (*Medium, error) {
	owner := core.Trim(options.Owner)
	if owner == "" {
		return nil, core.E(opNew, "owner is required", fs.ErrInvalid)
	}
	repo := core.Trim(options.Repo)
	if repo == "" {
		return nil, core.E(opNew, "repo is required", fs.ErrInvalid)
	}

	client := options.Client
	if client == nil {
		token := options.Token
		if token == "" {
			token = tokenFromEnvironment(options.TokenFile)
		}
		httpClient := options.HTTPClient
		if token != "" {
			httpClient = oauthClient(httpClient, token)
		}
		client = gh.NewClient(httpClient)
	}
	if options.BaseURL != "" {
		if err := setClientBaseURL(client, options.BaseURL); err != nil {
			return nil, core.E(opNew, "base URL is invalid", err)
		}
	}

	ref := core.Trim(options.Ref)
	if ref == "" {
		ref = core.Trim(options.Branch)
	}

	return &Medium{
		client:  client,
		owner:   owner,
		repo:    repo,
		ref:     ref,
		baseURL: core.Trim(options.BaseURL),
	}, nil
}

func tokenFromEnvironment(tokenFile string) string {
	if token := core.Trim(core.Env("GITHUB_TOKEN")); token != "" {
		return token
	}
	if tokenFile == "" {
		home := core.Trim(core.Env("HOME"))
		if home == "" {
			home = core.Trim(core.Env("DIR_HOME"))
		}
		if home == "" {
			return ""
		}
		tokenFile = core.Path(home, ".config", "lthn", "github-token")
	}

	medium, relativePath, err := tokenFileMedium(tokenFile)
	if err != nil {
		return ""
	}
	data, err := medium.Read(relativePath)
	if err != nil {
		return ""
	}
	return core.Trim(data)
}

func tokenFileMedium(tokenFile string) (coreio.Medium, string, error) {
	if core.PathIsAbs(tokenFile) {
		root := core.PathDir(tokenFile)
		relativePath := core.PathBase(tokenFile)
		if root == "" || root == "." || relativePath == "" || relativePath == "." || relativePath == "/" {
			return nil, "", fs.ErrInvalid
		}
		medium, err := coreio.NewSandboxed(root)
		return medium, relativePath, err
	}
	medium, err := coreio.NewSandboxed(".")
	return medium, tokenFile, err
}

func compareNames(left, right string) int {
	if left < right {
		return -1
	}
	if left > right {
		return 1
	}
	return 0
}

func oauthClient(client *http.Client, token string) *http.Client {
	var clone http.Client
	if client != nil {
		clone = *client
	}
	transport := clone.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}
	clone.Transport = &oauth2.Transport{
		Source: oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token}),
		Base:   transport,
	}
	return &clone
}

func setClientBaseURL(client *gh.Client, baseURL string) error {
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return err
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return fs.ErrInvalid
	}
	if !core.HasSuffix(parsed.Path, "/") {
		parsed.Path += "/"
	}
	client.BaseURL = parsed
	return nil
}

func cleanRelative(filePath string) string {
	clean := core.CleanPath("/"+core.Replace(filePath, "\\", "/"), "/")
	if clean == "/" {
		return ""
	}
	return core.TrimPrefix(clean, "/")
}

func requiredPath(operation, filePath string) (string, error) {
	clean := cleanRelative(filePath)
	if clean == "" {
		return "", core.E(operation, "path is required", fs.ErrInvalid)
	}
	return clean, nil
}

func (medium *Medium) contentOptions() *gh.RepositoryContentGetOptions {
	if medium.ref == "" {
		return nil
	}
	return &gh.RepositoryContentGetOptions{Ref: medium.ref}
}

func (medium *Medium) getContents(operation, filePath string) (*gh.RepositoryContent, []*gh.RepositoryContent, error) {
	fileContent, directoryContent, _, err := medium.client.Repositories.GetContents(
		context.Background(),
		medium.owner,
		medium.repo,
		filePath,
		medium.contentOptions(),
	)
	if err != nil {
		return nil, nil, wrapGitHubError(operation, filePath, err)
	}
	return fileContent, directoryContent, nil
}

func wrapGitHubError(operation, filePath string, err error) error {
	if err == nil {
		return nil
	}
	if core.Is(err, gh.ErrPathForbidden) {
		return core.E(operation, core.Concat("path is invalid: ", filePath), fs.ErrInvalid)
	}

	var responseError *gh.ErrorResponse
	if core.As(err, &responseError) && responseError.Response != nil {
		switch responseError.Response.StatusCode {
		case http.StatusNotFound:
			return core.E(operation, core.Concat(errNotFound, filePath), fs.ErrNotExist)
		case http.StatusUnauthorized, http.StatusForbidden:
			return core.E(operation, core.Concat("permission denied: ", filePath), fs.ErrPermission)
		case http.StatusUnprocessableEntity:
			return core.E(operation, core.Concat("invalid path: ", filePath), fs.ErrInvalid)
		}
	}
	return core.E(operation, core.Concat("GitHub contents request failed: ", filePath), err)
}

func readOnly(operation string) error {
	return core.E(operation, "GitHub medium is read-only", ErrReadOnly)
}

func fileInfoForContent(content *gh.RepositoryContent, name string) coreio.FileInfo {
	mode := fs.FileMode(0644)
	isDir := content.GetType() == "dir"
	if isDir {
		mode = fs.ModeDir | 0755
	}
	return coreio.NewFileInfo(name, int64(content.GetSize()), mode, time.Time{}, isDir)
}

func dirInfoForPath(filePath string) coreio.FileInfo {
	name := core.PathBase(filePath)
	if name == "." || name == "/" || name == "" {
		name = "."
	}
	return coreio.NewFileInfo(name, 0, fs.ModeDir|0755, time.Time{}, true)
}

// Read reads a repository file into a string.
func (medium *Medium) Read(filePath string) (string, error) {
	clean, err := requiredPath(opRead, filePath)
	if err != nil {
		return "", err
	}
	fileContent, directoryContent, err := medium.getContents(opRead, clean)
	if err != nil {
		return "", err
	}
	if directoryContent != nil || fileContent.GetType() == "dir" {
		return "", core.E(opRead, core.Concat("path is a directory: ", clean), fs.ErrInvalid)
	}
	if fileContent == nil {
		return "", core.E(opRead, core.Concat(errNotFound, clean), fs.ErrNotExist)
	}
	content, err := fileContent.GetContent()
	if err != nil {
		return "", core.E(opRead, core.Concat("decode content failed: ", clean), err)
	}
	return content, nil
}

// Write returns ErrReadOnly because GitHub Medium is read-only.
func (medium *Medium) Write(filePath, content string) error {
	return readOnly("github.Write")
}

// WriteMode returns ErrReadOnly because GitHub Medium is read-only.
func (medium *Medium) WriteMode(filePath, content string, mode fs.FileMode) error {
	return readOnly("github.WriteMode")
}

// EnsureDir returns ErrReadOnly because GitHub Medium is read-only.
func (medium *Medium) EnsureDir(filePath string) error {
	return readOnly("github.EnsureDir")
}

// IsFile reports whether filePath exists and is not a directory.
func (medium *Medium) IsFile(filePath string) bool {
	clean := cleanRelative(filePath)
	if clean == "" {
		return false
	}
	info, err := medium.Stat(clean)
	return err == nil && !info.IsDir()
}

// Delete returns ErrReadOnly because GitHub Medium is read-only.
func (medium *Medium) Delete(filePath string) error {
	return readOnly("github.Delete")
}

// DeleteAll returns ErrReadOnly because GitHub Medium is read-only.
func (medium *Medium) DeleteAll(filePath string) error {
	return readOnly("github.DeleteAll")
}

// Rename returns ErrReadOnly because GitHub Medium is read-only.
func (medium *Medium) Rename(oldPath, newPath string) error {
	return readOnly("github.Rename")
}

// List returns a recursive listing under a repository directory.
func (medium *Medium) List(filePath string) ([]fs.DirEntry, error) {
	clean := cleanRelative(filePath)
	entries, err := medium.listRecursive(clean)
	if err != nil {
		return nil, err
	}
	slices.SortFunc(entries, func(a, b fs.DirEntry) int {
		return compareNames(a.Name(), b.Name())
	})
	return entries, nil
}

func (medium *Medium) listRecursive(filePath string) ([]fs.DirEntry, error) {
	fileContent, directoryContent, err := medium.getContents(opList, filePath)
	if err != nil {
		return nil, err
	}
	if fileContent == nil && directoryContent == nil {
		return nil, core.E(opList, core.Concat(errNotFound, filePath), fs.ErrNotExist)
	}
	if fileContent != nil && fileContent.GetType() != "dir" {
		return nil, core.E(opList, core.Concat("path is not a directory: ", filePath), fs.ErrInvalid)
	}
	if directoryContent == nil {
		return nil, core.E(opList, core.Concat("path is not a directory: ", filePath), fs.ErrInvalid)
	}

	var entries []fs.DirEntry
	for _, content := range directoryContent {
		name := cleanRelative(content.GetPath())
		if name == "" {
			name = content.GetName()
		}
		info := fileInfoForContent(content, name)
		entries = append(entries, coreio.NewDirEntry(name, info.IsDir(), info.Mode(), info))
		if content.GetType() == "dir" {
			childEntries, err := medium.listRecursive(content.GetPath())
			if err != nil {
				return nil, err
			}
			entries = append(entries, childEntries...)
		}
	}
	return entries, nil
}

// Stat returns metadata for a repository path.
func (medium *Medium) Stat(filePath string) (fs.FileInfo, error) {
	clean, err := requiredPath(opStat, filePath)
	if err != nil {
		return nil, err
	}
	fileContent, directoryContent, err := medium.getContents(opStat, clean)
	if err != nil {
		return nil, err
	}
	if fileContent == nil && directoryContent == nil {
		return nil, core.E(opStat, core.Concat(errNotFound, clean), fs.ErrNotExist)
	}
	if directoryContent != nil || fileContent.GetType() == "dir" {
		return dirInfoForPath(clean), nil
	}
	return fileInfoForContent(fileContent, core.PathBase(clean)), nil
}

// Open opens a repository file for reading.
func (medium *Medium) Open(filePath string) (fs.File, error) {
	content, err := medium.Read(filePath)
	if err != nil {
		return nil, err
	}
	info, err := medium.Stat(filePath)
	if err != nil {
		return nil, err
	}
	return &githubFile{
		name:    info.Name(),
		content: []byte(content),
		mode:    info.Mode(),
		modTime: info.ModTime(),
	}, nil
}

// Create returns ErrReadOnly because GitHub Medium is read-only.
func (medium *Medium) Create(filePath string) (goio.WriteCloser, error) {
	return nil, readOnly("github.Create")
}

// Append returns ErrReadOnly because GitHub Medium is read-only.
func (medium *Medium) Append(filePath string) (goio.WriteCloser, error) {
	return nil, readOnly("github.Append")
}

// ReadStream opens a repository file as an io.ReadCloser.
func (medium *Medium) ReadStream(filePath string) (goio.ReadCloser, error) {
	content, err := medium.Read(filePath)
	if err != nil {
		return nil, err
	}
	return goio.NopCloser(core.NewReader(content)), nil
}

// WriteStream returns ErrReadOnly because GitHub Medium is read-only.
func (medium *Medium) WriteStream(filePath string) (goio.WriteCloser, error) {
	return nil, readOnly("github.WriteStream")
}

// Exists reports whether a repository path exists.
func (medium *Medium) Exists(filePath string) bool {
	clean := cleanRelative(filePath)
	if clean == "" {
		return false
	}
	_, err := medium.Stat(clean)
	return err == nil
}

// IsDir reports whether a repository path exists and is a directory.
func (medium *Medium) IsDir(filePath string) bool {
	clean := cleanRelative(filePath)
	if clean == "" {
		return false
	}
	info, err := medium.Stat(clean)
	return err == nil && info.IsDir()
}

// Clone returns all file contents under filePath, keyed by repository path.
// Default full-repo clones use Borg's Git cloner and DataNode substrate; custom
// GitHub API URLs and explicit refs keep the REST contents path for compatibility.
func (medium *Medium) Clone(filePath string) (map[string]string, error) {
	if medium.baseURL == "" && medium.ref == "" {
		return medium.cloneWithBorg(filePath)
	}
	return medium.cloneWithContentsAPI(filePath)
}

func (medium *Medium) cloneWithBorg(filePath string) (map[string]string, error) {
	dataNode, err := borgvcs.NewGitCloner().CloneGitRepository(medium.borgCloneURL(), nil)
	if err != nil {
		return nil, core.E(opClone, "Borg clone failed", err)
	}
	contents, err := collectBorgDataNodeContents(dataNode, cleanRelative(filePath))
	if err != nil {
		return nil, core.E(opClone, "failed to read Borg DataNode clone", err)
	}
	return contents, nil
}

func (medium *Medium) borgCloneURL() string {
	return core.Sprintf("https://github.com/%s/%s.git", medium.owner, medium.repo)
}

func collectBorgDataNodeContents(dataNode *borgdatanode.DataNode, clean string) (map[string]string, error) {
	if dataNode == nil {
		return nil, fs.ErrInvalid
	}
	contents := make(map[string]string)
	if clean != "" {
		info, err := dataNode.Stat(clean)
		if err != nil {
			return nil, fs.ErrNotExist
		}
		if !info.IsDir() {
			content, err := readBorgDataNodeFile(dataNode, clean)
			if err != nil {
				return nil, err
			}
			contents[clean] = content
			return contents, nil
		}
	}
	if err := collectBorgDataNodeDir(dataNode, clean, contents); err != nil {
		return nil, err
	}
	return contents, nil
}

func collectBorgDataNodeDir(dataNode *borgdatanode.DataNode, dirPath string, contents map[string]string) error {
	entries, err := dataNode.ReadDir(dirPath)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		childPath := entry.Name()
		if dirPath != "" {
			childPath = core.PathJoin(dirPath, childPath)
		}
		if entry.IsDir() {
			if err := collectBorgDataNodeDir(dataNode, childPath, contents); err != nil {
				return err
			}
			continue
		}
		content, err := readBorgDataNodeFile(dataNode, childPath)
		if err != nil {
			return err
		}
		contents[childPath] = content
	}
	return nil
}

func readBorgDataNodeFile(dataNode *borgdatanode.DataNode, filePath string) (string, error) {
	file, err := dataNode.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	data, err := goio.ReadAll(file)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (medium *Medium) cloneWithContentsAPI(filePath string) (map[string]string, error) {
	clean := cleanRelative(filePath)
	if clean != "" {
		info, err := medium.Stat(clean)
		if err != nil {
			return nil, err
		}
		if !info.IsDir() {
			content, err := medium.Read(clean)
			if err != nil {
				return nil, err
			}
			return map[string]string{clean: content}, nil
		}
	}

	entries, err := medium.List(clean)
	if err != nil {
		return nil, err
	}
	contents := make(map[string]string)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		content, err := medium.Read(entry.Name())
		if err != nil {
			return nil, err
		}
		contents[entry.Name()] = content
	}
	return contents, nil
}

type githubFile struct {
	name    string
	content []byte
	mode    fs.FileMode
	modTime time.Time
	offset  int64
	closed  bool
}

var _ fs.File = (*githubFile)(nil)

func (file *githubFile) Stat() (fs.FileInfo, error) {
	return coreio.NewFileInfo(file.name, int64(len(file.content)), file.mode, file.modTime, false), nil
}

func (file *githubFile) Read(data []byte) (int, error) {
	if file.closed {
		return 0, fs.ErrClosed
	}
	reader := core.NewReader(string(file.content))
	if _, err := reader.Seek(file.offset, goio.SeekStart); err != nil {
		return 0, err
	}
	n, err := reader.Read(data)
	file.offset += int64(n)
	return n, err
}

func (file *githubFile) Close() error {
	file.closed = true
	return nil
}

func (file *githubFile) String() string {
	return core.Sprintf("githubFile(%s)", file.name)
}
