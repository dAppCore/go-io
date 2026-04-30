// SPDX-License-Identifier: EUPL-1.2

package api

import (
	"context"
	goio "io"
	"io/fs"
	"net/http"
	"strconv"
	"sync"

	core "dappco.re/go"
	coreio "dappco.re/go/io"
	workspacesvc "dappco.re/go/io/workspace"
	"github.com/gin-gonic/gin"
)

type rfc15Action struct {
	Name      string
	Medium    string
	Operation string
}

func closeAPIReader(closer goio.Closer, operation string) {
	if err := closer.Close(); err != nil {
		core.Warn("api close failed", "op", operation, "err", err)
	}
}

var rfc15Actions = []rfc15Action{
	{Name: coreio.ActionLocalRead, Medium: "local", Operation: "read"},
	{Name: coreio.ActionLocalWrite, Medium: "local", Operation: "write"},
	{Name: coreio.ActionLocalList, Medium: "local", Operation: "list"},
	{Name: coreio.ActionLocalDelete, Medium: "local", Operation: "delete"},
	{Name: coreio.ActionMemoryRead, Medium: "memory", Operation: "read"},
	{Name: coreio.ActionMemoryWrite, Medium: "memory", Operation: "write"},
	{Name: coreio.ActionGitHubClone, Medium: "github", Operation: "clone"},
	{Name: coreio.ActionGitHubRead, Medium: "github", Operation: "read"},
	{Name: coreio.ActionPWAScrape, Medium: "pwa", Operation: "scrape"},
	{Name: coreio.ActionSFTPRead, Medium: "sftp", Operation: "read"},
	{Name: coreio.ActionSFTPWrite, Medium: "sftp", Operation: "write"},
	{Name: coreio.ActionS3Read, Medium: "s3", Operation: "read"},
	{Name: coreio.ActionS3Write, Medium: "s3", Operation: "write"},
	{Name: coreio.ActionCubeRead, Medium: "cube", Operation: "read"},
	{Name: coreio.ActionCubeWrite, Medium: "cube", Operation: "write"},
	{Name: coreio.ActionCubePack, Medium: "cube", Operation: "pack"},
	{Name: coreio.ActionCubeUnpack, Medium: "cube", Operation: "unpack"},
	{Name: coreio.ActionCopy, Medium: "any", Operation: "copy"},
}

var errUnsupportedMediumOperation = core.NewError("unsupported medium operation")

const payloadPathKey = "pa" + "th"

var apiWorkspaceServices sync.Map

type mediumRequest struct {
	Root      string
	Path      string
	OldPath   string
	NewPath   string
	Content   string
	Mode      any
	Recursive bool
}

type mediumResponse struct {
	OK      bool           `json:"ok,omitempty"`
	Content string         `json:"content,omitempty"`
	Entries []dirEntryDTO  `json:"entries,omitempty"`
	Info    *fileInfoDTO   `json:"info,omitempty"`
	Exists  *bool          `json:"exists,omitempty"`
	IsFile  *bool          `json:"isFile,omitempty"`
	IsDir   *bool          `json:"isDir,omitempty"`
	Action  string         `json:"action,omitempty"`
	Value   any            `json:"value,omitempty"`
	Medium  string         `json:"medium,omitempty"`
	Op      string         `json:"op,omitempty"`
	Meta    map[string]any `json:"meta,omitempty"`
}

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type apiResponse struct {
	Success bool      `json:"success"`
	Data    any       `json:"data,omitempty"`
	Error   *apiError `json:"error,omitempty"`
}

func apiOK(data any) apiResponse {
	return apiResponse{Success: true, Data: data}
}

func apiFail(code, message string) apiResponse {
	return apiResponse{Success: false, Error: &apiError{Code: code, Message: message}}
}

type dirEntryDTO struct {
	Name  string `json:"name"`
	IsDir bool   `json:"isDir"`
	Type  string `json:"type,omitempty"`
	Size  int64  `json:"size,omitempty"`
	Mode  string `json:"mode,omitempty"`
}

type fileInfoDTO struct {
	Name    string `json:"name"`
	Size    int64  `json:"size"`
	Mode    string `json:"mode"`
	ModTime string `json:"modTime"`
	IsDir   bool   `json:"isDir"`
}

func (p *IOProvider) createWorkspace(c *gin.Context) {
	payload, ok := bindPayload(c)
	if !ok {
		return
	}
	workspaceName := workspaceNameFromPayload(payload)
	if workspaceName == "" {
		c.JSON(http.StatusBadRequest, apiFail("invalid_request", "workspace is required"))
		return
	}

	service, ok := p.resolveWorkspaceService(c)
	if !ok {
		return
	}
	result := service.HandleWorkspaceCommand(workspacesvc.WorkspaceCommand{
		Action:    workspacesvc.WorkspaceCreateAction,
		Workspace: workspaceName,
	})
	writeWorkspaceResult(c, workspacesvc.WorkspaceCreateAction, result)
}

func (p *IOProvider) switchWorkspace(c *gin.Context) {
	workspaceID := core.Trim(c.Param("id"))
	if workspaceID == "" {
		c.JSON(http.StatusBadRequest, apiFail("invalid_request", "workspace id is required"))
		return
	}

	service, ok := p.resolveWorkspaceService(c)
	if !ok {
		return
	}
	result := service.HandleWorkspaceCommand(workspacesvc.WorkspaceCommand{
		Action:    workspacesvc.WorkspaceSwitchAction,
		Workspace: workspaceID,
	})
	writeWorkspaceResult(c, workspacesvc.WorkspaceSwitchAction, result)
}

func (p *IOProvider) handleWorkspaceCommand(c *gin.Context) {
	workspaceID := core.Trim(c.Param("id"))
	if workspaceID == "" {
		c.JSON(http.StatusBadRequest, apiFail("invalid_request", "workspace id is required"))
		return
	}
	payload, ok := bindPayload(c)
	if !ok {
		return
	}
	if core.Trim(stringValue(payload, "action")) == "" {
		c.JSON(http.StatusBadRequest, apiFail("invalid_request", "action is required"))
		return
	}

	service, ok := p.resolveWorkspaceService(c)
	if !ok {
		return
	}
	command := workspaceCommandFromPayload(workspaceID, payload)
	result := service.HandleWorkspaceCommand(command)
	writeWorkspaceResult(c, command.Action, result)
}

func (p *IOProvider) dispatchAction(c *gin.Context) {
	actionName := core.Trim(c.Param("action"))
	action, ok := findRFC15Action(actionName)
	if !ok {
		c.JSON(http.StatusNotFound, apiFail("unknown_action", "RFC §15 action is not registered"))
		return
	}
	payload, ok := bindPayload(c)
	if !ok {
		return
	}
	if p == nil || p.core == nil {
		c.JSON(http.StatusServiceUnavailable, apiFail("service_unavailable", "core action registry is not configured"))
		return
	}

	result := p.core.Action(action.Name).Run(c.Request.Context(), optionsFromPayload(payload))
	if !result.OK {
		c.JSON(http.StatusInternalServerError, apiFail("action_failed", resultErrorMessage(result)))
		return
	}
	c.JSON(http.StatusOK, apiOK(mediumResponse{OK: true, Action: action.Name, Value: result.Value}))
}

func (p *IOProvider) dispatchMedium(c *gin.Context) {
	mediumType := core.Trim(c.Param("type"))
	op := core.Trim(c.Param("op"))
	if mediumType == "" {
		c.JSON(http.StatusBadRequest, apiFail("invalid_request", "medium type is required"))
		return
	}
	if op == "" {
		c.JSON(http.StatusBadRequest, apiFail("invalid_request", "medium operation is required"))
		return
	}

	payload, ok := bindPayload(c)
	if !ok {
		return
	}
	req := mediumRequestFromPayload(payload)
	medium, ok := p.resolveMedium(c, mediumType, req)
	if !ok {
		return
	}

	resp, err := dispatchMediumOperation(c.Request.Context(), medium, op, req)
	if err != nil {
		if core.Is(err, errUnsupportedMediumOperation) {
			notImplemented(c, err.Error())
			return
		}
		c.JSON(http.StatusInternalServerError, apiFail("medium_failed", err.Error()))
		return
	}
	resp.Medium = mediumType
	resp.Op = op
	c.JSON(http.StatusOK, apiOK(resp))
}

func (p *IOProvider) resolveMedium(c *gin.Context, mediumType string, req mediumRequest) (coreio.Medium, bool) {
	switch core.Lower(mediumType) {
	case "memory":
		if p == nil || p.memory == nil {
			c.JSON(http.StatusServiceUnavailable, apiFail("service_unavailable", "memory medium is not configured"))
			return nil, false
		}
		return p.memory, true
	case "local":
		if p == nil || p.local == nil {
			c.JSON(http.StatusServiceUnavailable, apiFail("service_unavailable", "local medium is not configured"))
			return nil, false
		}
		return p.local, true
	default:
		unconfiguredMedium(c, mediumType)
		return nil, false
	}
}

func (p *IOProvider) resolveWorkspaceService(c *gin.Context) (*workspacesvc.Workspace, bool) {
	if p == nil {
		c.JSON(http.StatusServiceUnavailable, apiFail("service_unavailable", "workspace service is not configured"))
		return nil, false
	}
	if service, ok := apiWorkspaceServices.Load(p); ok {
		workspaceService, ok := service.(*workspacesvc.Workspace)
		if ok {
			return workspaceService, true
		}
	}

	medium := p.memory
	if medium == nil {
		medium = coreio.NewMemoryMedium()
	}
	workspaceService, err := workspacesvc.NewWorkspace(medium, "workspaces")
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, apiFail("service_unavailable", err.Error()))
		return nil, false
	}
	actual, _ := apiWorkspaceServices.LoadOrStore(p, workspaceService)
	return actual.(*workspacesvc.Workspace), true
}

func workspaceCommandFromPayload(pathWorkspace string, payload map[string]any) workspacesvc.WorkspaceCommand {
	workspaceName := workspaceNameFromPayload(payload)
	if workspaceName == "" {
		workspaceName = pathWorkspace
	}
	return workspacesvc.WorkspaceCommand{
		Action:    core.Trim(stringValue(payload, "action")),
		Workspace: workspaceName,
		Path:      stringValue(payload, payloadPathKey),
		Content:   stringValue(payload, "content"),
	}
}

func workspaceNameFromPayload(payload map[string]any) string {
	return core.Trim(stringValue(payload, "workspace", "name", "identifier", "workspaceID", "workspace_id"))
}

func writeWorkspaceResult(c *gin.Context, action string, result core.Result) {
	if !result.OK {
		c.JSON(http.StatusInternalServerError, apiFail("workspace_failed", resultErrorMessage(result)))
		return
	}

	response := mediumResponse{
		OK:     true,
		Action: action,
		Value:  result.Value,
	}
	switch value := result.Value.(type) {
	case coreio.Medium:
		response.Value = nil
	case string:
		response.Content = value
	case []fs.DirEntry:
		response.Value = nil
		response.Entries = dirEntryDTOs(value)
	}
	c.JSON(http.StatusOK, apiOK(response))
}

type mediumOperationHandler func(context.Context, coreio.Medium, mediumRequest) (mediumResponse, error)

var mediumOperationHandlers = map[string]mediumOperationHandler{
	"read":        readMediumOperation,
	"write":       writeMediumOperation,
	"writemode":   writeModeMediumOperation,
	"ensuredir":   ensureDirMediumOperation,
	"mkdir":       ensureDirMediumOperation,
	"isfile":      isFileMediumOperation,
	"delete":      deleteMediumOperation,
	"deleteall":   deleteAllMediumOperation,
	"rename":      renameMediumOperation,
	"list":        listMediumOperation,
	"stat":        statMediumOperation,
	"open":        openMediumOperation,
	"create":      createMediumOperation,
	"append":      appendMediumOperation,
	"readstream":  readStreamMediumOperation,
	"writestream": writeStreamMediumOperation,
	"exists":      existsMediumOperation,
	"isdir":       isDirMediumOperation,
}

func dispatchMediumOperation(ctx context.Context, medium coreio.Medium, op string, req mediumRequest) (mediumResponse, error) {
	handler, ok := mediumOperationHandlers[core.Lower(op)]
	if !ok {
		return mediumResponse{}, core.Errorf("%w: %s", errUnsupportedMediumOperation, op)
	}
	return handler(ctx, medium, req)
}

func readMediumOperation(_ context.Context, medium coreio.Medium, req mediumRequest) (mediumResponse, error) {
	content, err := medium.Read(req.Path)
	if err != nil {
		return mediumResponse{}, err
	}
	return mediumResponse{OK: true, Content: content}, nil
}

func writeMediumOperation(_ context.Context, medium coreio.Medium, req mediumRequest) (mediumResponse, error) {
	if err := medium.Write(req.Path, req.Content); err != nil {
		return mediumResponse{}, err
	}
	return mediumResponse{OK: true}, nil
}

func writeModeMediumOperation(_ context.Context, medium coreio.Medium, req mediumRequest) (mediumResponse, error) {
	mode, err := fileModeValue(req.Mode, 0644)
	if err != nil {
		return mediumResponse{}, err
	}
	if err := medium.WriteMode(req.Path, req.Content, mode); err != nil {
		return mediumResponse{}, err
	}
	return mediumResponse{OK: true}, nil
}

func ensureDirMediumOperation(_ context.Context, medium coreio.Medium, req mediumRequest) (mediumResponse, error) {
	if err := medium.EnsureDir(req.Path); err != nil {
		return mediumResponse{}, err
	}
	return mediumResponse{OK: true}, nil
}

func isFileMediumOperation(_ context.Context, medium coreio.Medium, req mediumRequest) (mediumResponse, error) {
	ok := medium.IsFile(req.Path)
	return mediumResponse{OK: true, IsFile: &ok}, nil
}

func deleteMediumOperation(ctx context.Context, medium coreio.Medium, req mediumRequest) (mediumResponse, error) {
	if req.Recursive {
		return deleteAllMediumOperation(ctx, medium, req)
	}
	if err := medium.Delete(req.Path); err != nil {
		return mediumResponse{}, err
	}
	return mediumResponse{OK: true}, nil
}

func deleteAllMediumOperation(_ context.Context, medium coreio.Medium, req mediumRequest) (mediumResponse, error) {
	if err := medium.DeleteAll(req.Path); err != nil {
		return mediumResponse{}, err
	}
	return mediumResponse{OK: true}, nil
}

func renameMediumOperation(_ context.Context, medium coreio.Medium, req mediumRequest) (mediumResponse, error) {
	if err := medium.Rename(req.OldPath, req.NewPath); err != nil {
		return mediumResponse{}, err
	}
	return mediumResponse{OK: true}, nil
}

func listMediumOperation(_ context.Context, medium coreio.Medium, req mediumRequest) (mediumResponse, error) {
	entries, err := medium.List(req.Path)
	if err != nil {
		return mediumResponse{}, err
	}
	return mediumResponse{OK: true, Entries: dirEntryDTOs(entries)}, nil
}

func statMediumOperation(_ context.Context, medium coreio.Medium, req mediumRequest) (mediumResponse, error) {
	info, err := medium.Stat(req.Path)
	if err != nil {
		return mediumResponse{}, err
	}
	return mediumResponse{OK: true, Info: fileInfoDTOFromInfo(info)}, nil
}

func openMediumOperation(_ context.Context, medium coreio.Medium, req mediumRequest) (mediumResponse, error) {
	file, err := medium.Open(req.Path)
	if err != nil {
		return mediumResponse{}, err
	}
	defer closeAPIReader(file, "api.openMediumOperation")
	return readAllContent(file)
}

func createMediumOperation(_ context.Context, medium coreio.Medium, req mediumRequest) (mediumResponse, error) {
	writer, err := medium.Create(req.Path)
	if err != nil {
		return mediumResponse{}, err
	}
	if err := writeAndClose(writer, req.Content); err != nil {
		return mediumResponse{}, err
	}
	return mediumResponse{OK: true}, nil
}

func appendMediumOperation(_ context.Context, medium coreio.Medium, req mediumRequest) (mediumResponse, error) {
	writer, err := medium.Append(req.Path)
	if err != nil {
		return mediumResponse{}, err
	}
	if err := writeAndClose(writer, req.Content); err != nil {
		return mediumResponse{}, err
	}
	return mediumResponse{OK: true}, nil
}

func readStreamMediumOperation(_ context.Context, medium coreio.Medium, req mediumRequest) (mediumResponse, error) {
	reader, err := medium.ReadStream(req.Path)
	if err != nil {
		return mediumResponse{}, err
	}
	defer closeAPIReader(reader, "api.readStreamMediumOperation")
	return readAllContent(reader)
}

func writeStreamMediumOperation(_ context.Context, medium coreio.Medium, req mediumRequest) (mediumResponse, error) {
	writer, err := medium.WriteStream(req.Path)
	if err != nil {
		return mediumResponse{}, err
	}
	if err := writeAndClose(writer, req.Content); err != nil {
		return mediumResponse{}, err
	}
	return mediumResponse{OK: true}, nil
}

func existsMediumOperation(_ context.Context, medium coreio.Medium, req mediumRequest) (mediumResponse, error) {
	ok := medium.Exists(req.Path)
	return mediumResponse{OK: true, Exists: &ok}, nil
}

func isDirMediumOperation(_ context.Context, medium coreio.Medium, req mediumRequest) (mediumResponse, error) {
	ok := medium.IsDir(req.Path)
	return mediumResponse{OK: true, IsDir: &ok}, nil
}

func readAllContent(reader goio.Reader) (mediumResponse, error) {
	content, err := goio.ReadAll(reader)
	if err != nil {
		return mediumResponse{}, err
	}
	return mediumResponse{OK: true, Content: string(content)}, nil
}

func bindPayload(c *gin.Context) (map[string]any, bool) {
	payload := map[string]any{}
	if c.Request == nil || c.Request.Body == nil || c.Request.ContentLength == 0 {
		return payload, true
	}
	data, err := goio.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, apiFail("invalid_request", err.Error()))
		return nil, false
	}
	if result := core.JSONUnmarshal(data, &payload); !result.OK {
		c.JSON(http.StatusBadRequest, apiFail("invalid_request", result.Error()))
		return nil, false
	}
	return payload, true
}

func mediumRequestFromPayload(payload map[string]any) mediumRequest {
	return mediumRequest{
		Root:      stringValue(payload, "root"),
		Path:      stringValue(payload, payloadPathKey),
		OldPath:   stringValue(payload, "oldPath", "old_path"),
		NewPath:   stringValue(payload, "newPath", "new_path"),
		Content:   stringValue(payload, "content"),
		Mode:      firstValue(payload, "mode"),
		Recursive: boolValue(payload, "recursive"),
	}
}

func optionsFromPayload(payload map[string]any) core.Options {
	options := make([]core.Option, 0, len(payload))
	for key, value := range payload {
		options = append(options, core.Option{Key: key, Value: normalizedValue(value)})
	}
	return core.NewOptions(options...)
}

func findRFC15Action(name string) (rfc15Action, bool) {
	for _, action := range rfc15Actions {
		if action.Name == name {
			return action, true
		}
	}
	return rfc15Action{}, false
}

func notImplemented(c *gin.Context, message string) {
	c.JSON(http.StatusNotImplemented, apiFail("not_implemented", message))
}

func unconfiguredMedium(c *gin.Context, mediumType string) {
	notImplemented(c, core.Sprintf("%s medium is not configured", mediumType))
}

func resultErrorMessage(result core.Result) string {
	if err, ok := result.Value.(error); ok && err != nil {
		return err.Error()
	}
	if result.Value != nil {
		return core.Sprint(result.Value)
	}
	return "action failed"
}

func firstValue(payload map[string]any, keys ...string) any {
	for _, key := range keys {
		if value, ok := payload[key]; ok {
			return value
		}
	}
	return nil
}

func stringValue(payload map[string]any, keys ...string) string {
	value := firstValue(payload, keys...)
	switch typed := value.(type) {
	case string:
		return typed
	default:
		return ""
	}
}

func boolValue(payload map[string]any, keys ...string) bool {
	value := firstValue(payload, keys...)
	if typed, ok := value.(bool); ok {
		return typed
	}
	return false
}

func normalizedValue(value any) any {
	switch typed := value.(type) {
	case []any:
		out := make([]any, len(typed))
		for i, item := range typed {
			out[i] = normalizedValue(item)
		}
		return out
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, item := range typed {
			out[key] = normalizedValue(item)
		}
		return out
	default:
		return value
	}
}

func fileModeValue(value any, fallback fs.FileMode) (fs.FileMode, error) {
	if value == nil {
		return fallback, nil
	}
	switch typed := value.(type) {
	case fs.FileMode:
		return typed, nil
	case int:
		return fs.FileMode(typed), nil
	case int64:
		return fs.FileMode(typed), nil
	case float64:
		return fs.FileMode(typed), nil
	case string:
		parsed, err := strconv.ParseInt(typed, 0, 64)
		if err != nil {
			return 0, err
		}
		return fs.FileMode(parsed), nil
	default:
		return 0, core.Errorf("unsupported file mode type %T", value)
	}
}

func dirEntryDTOs(entries []fs.DirEntry) []dirEntryDTO {
	out := make([]dirEntryDTO, 0, len(entries))
	for _, entry := range entries {
		dto := dirEntryDTO{
			Name:  entry.Name(),
			IsDir: entry.IsDir(),
			Type:  entry.Type().String(),
		}
		if info, err := entry.Info(); err == nil && info != nil {
			dto.Size = info.Size()
			dto.Mode = info.Mode().String()
		}
		out = append(out, dto)
	}
	return out
}

func fileInfoDTOFromInfo(info fs.FileInfo) *fileInfoDTO {
	if info == nil {
		return nil
	}
	return &fileInfoDTO{
		Name:    info.Name(),
		Size:    info.Size(),
		Mode:    info.Mode().String(),
		ModTime: info.ModTime().Format("2006-01-02T15:04:05Z07:00"),
		IsDir:   info.IsDir(),
	}
}

func writeAndClose(writer goio.WriteCloser, content string) error {
	if _, err := goio.WriteString(writer, content); err != nil {
		if closeErr := writer.Close(); closeErr != nil {
			return core.ErrorJoin(err, closeErr)
		}
		return err
	}
	return writer.Close()
}
