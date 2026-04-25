// SPDX-License-Identifier: EUPL-1.2

package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	goio "io"
	"io/fs"
	"net/http"
	"strconv"
	"strings"

	goapi "dappco.re/go/api"
	core "dappco.re/go/core"
	coreio "dappco.re/go/io"
	"github.com/gin-gonic/gin"
)

type rfc15Action struct {
	Name      string
	Medium    string
	Operation string
	Wired     bool
}

var rfc15Actions = []rfc15Action{
	{Name: coreio.ActionLocalRead, Medium: "local", Operation: "read", Wired: true},
	{Name: coreio.ActionLocalWrite, Medium: "local", Operation: "write", Wired: true},
	{Name: coreio.ActionLocalList, Medium: "local", Operation: "list", Wired: true},
	{Name: coreio.ActionLocalDelete, Medium: "local", Operation: "delete", Wired: true},
	{Name: coreio.ActionMemoryRead, Medium: "memory", Operation: "read", Wired: true},
	{Name: coreio.ActionMemoryWrite, Medium: "memory", Operation: "write", Wired: true},
	{Name: "core.io.github.clone", Medium: "github", Operation: "clone"},
	{Name: "core.io.github.read", Medium: "github", Operation: "read"},
	{Name: "core.io.pwa.scrape", Medium: "pwa", Operation: "scrape"},
	{Name: "core.io.sftp.read", Medium: "sftp", Operation: "read"},
	{Name: "core.io.sftp.write", Medium: "sftp", Operation: "write"},
	{Name: "core.io.s3.read", Medium: "s3", Operation: "read"},
	{Name: "core.io.s3.write", Medium: "s3", Operation: "write"},
	{Name: "core.io.cube.read", Medium: "cube", Operation: "read"},
	{Name: "core.io.cube.write", Medium: "cube", Operation: "write"},
	{Name: "core.io.cube.pack", Medium: "cube", Operation: "pack"},
	{Name: "core.io.cube.unpack", Medium: "cube", Operation: "unpack"},
	{Name: coreio.ActionCopy, Medium: "any", Operation: "copy", Wired: true},
}

var errUnsupportedMediumOperation = errors.New("unsupported medium operation")

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
	if strings.TrimSpace(stringValue(payload, "identifier")) == "" {
		c.JSON(http.StatusBadRequest, goapi.Fail("invalid_request", "identifier is required"))
		return
	}
	if stringValue(payload, "password", "passphrase") == "" {
		c.JSON(http.StatusBadRequest, goapi.Fail("invalid_request", "password is required"))
		return
	}

	// TODO(#631): delegate to workspace.Service.CreateWorkspace once Workspace is wired in actions.go.
	notImplemented(c, "workspace CreateWorkspace is not wired")
}

func (p *IOProvider) switchWorkspace(c *gin.Context) {
	workspaceID := strings.TrimSpace(c.Param("id"))
	if workspaceID == "" {
		c.JSON(http.StatusBadRequest, goapi.Fail("invalid_request", "workspace id is required"))
		return
	}

	// TODO(#631): delegate to workspace.Service.SwitchWorkspace once Workspace is wired in actions.go.
	notImplemented(c, "workspace SwitchWorkspace is not wired")
}

func (p *IOProvider) handleWorkspaceCommand(c *gin.Context) {
	workspaceID := strings.TrimSpace(c.Param("id"))
	if workspaceID == "" {
		c.JSON(http.StatusBadRequest, goapi.Fail("invalid_request", "workspace id is required"))
		return
	}
	payload, ok := bindPayload(c)
	if !ok {
		return
	}
	if strings.TrimSpace(stringValue(payload, "action")) == "" {
		c.JSON(http.StatusBadRequest, goapi.Fail("invalid_request", "action is required"))
		return
	}

	// TODO(#631): delegate to workspace.Service.HandleWorkspaceCommand once Workspace is wired in actions.go.
	notImplemented(c, "workspace HandleWorkspaceCommand is not wired")
}

func (p *IOProvider) dispatchAction(c *gin.Context) {
	actionName := strings.TrimSpace(c.Param("action"))
	action, ok := findRFC15Action(actionName)
	if !ok {
		c.JSON(http.StatusNotFound, goapi.Fail("unknown_action", "RFC §15 action is not registered"))
		return
	}
	if !action.Wired {
		// TODO(#632): wire the remaining RFC §15 named actions into go-io actions.go.
		notImplemented(c, fmt.Sprintf("%s is not wired", action.Name))
		return
	}
	payload, ok := bindPayload(c)
	if !ok {
		return
	}
	if p == nil || p.core == nil {
		c.JSON(http.StatusServiceUnavailable, goapi.Fail("service_unavailable", "core action registry is not configured"))
		return
	}

	result := p.core.Action(action.Name).Run(c.Request.Context(), optionsFromPayload(payload))
	if !result.OK {
		c.JSON(http.StatusInternalServerError, goapi.Fail("action_failed", resultErrorMessage(result)))
		return
	}
	c.JSON(http.StatusOK, goapi.OK(mediumResponse{OK: true, Action: action.Name, Value: result.Value}))
}

func (p *IOProvider) dispatchMedium(c *gin.Context) {
	mediumType := strings.TrimSpace(c.Param("type"))
	op := strings.TrimSpace(c.Param("op"))
	if mediumType == "" {
		c.JSON(http.StatusBadRequest, goapi.Fail("invalid_request", "medium type is required"))
		return
	}
	if op == "" {
		c.JSON(http.StatusBadRequest, goapi.Fail("invalid_request", "medium operation is required"))
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
		if errors.Is(err, errUnsupportedMediumOperation) {
			notImplemented(c, err.Error())
			return
		}
		c.JSON(http.StatusInternalServerError, goapi.Fail("medium_failed", err.Error()))
		return
	}
	resp.Medium = mediumType
	resp.Op = op
	c.JSON(http.StatusOK, goapi.OK(resp))
}

func (p *IOProvider) resolveMedium(c *gin.Context, mediumType string, req mediumRequest) (coreio.Medium, bool) {
	switch strings.ToLower(mediumType) {
	case "memory":
		if p == nil || p.memory == nil {
			c.JSON(http.StatusServiceUnavailable, goapi.Fail("service_unavailable", "memory medium is not configured"))
			return nil, false
		}
		return p.memory, true
	case "local":
		if strings.TrimSpace(req.Root) == "" {
			c.JSON(http.StatusBadRequest, goapi.Fail("invalid_request", "root is required for local medium"))
			return nil, false
		}
		medium, err := coreio.NewSandboxed(req.Root)
		if err != nil {
			c.JSON(http.StatusBadRequest, goapi.Fail("invalid_request", err.Error()))
			return nil, false
		}
		return medium, true
	case "github", "pwa":
		// TODO(#633): delegate once GitHub and PWA Medium backends are wired for HTTP construction.
		notImplemented(c, fmt.Sprintf("%s medium is not wired", mediumType))
		return nil, false
	case "sftp", "webdav":
		// TODO(#634): delegate once SFTP and WebDAV Medium backends are wired for HTTP construction.
		notImplemented(c, fmt.Sprintf("%s medium is not wired", mediumType))
		return nil, false
	default:
		notImplemented(c, fmt.Sprintf("%s medium is not configured", mediumType))
		return nil, false
	}
}

func dispatchMediumOperation(ctx context.Context, medium coreio.Medium, op string, req mediumRequest) (mediumResponse, error) {
	_ = ctx
	switch strings.ToLower(op) {
	case "read":
		content, err := medium.Read(req.Path)
		if err != nil {
			return mediumResponse{}, err
		}
		return mediumResponse{OK: true, Content: content}, nil
	case "write":
		if err := medium.Write(req.Path, req.Content); err != nil {
			return mediumResponse{}, err
		}
		return mediumResponse{OK: true}, nil
	case "writemode":
		mode, err := fileModeValue(req.Mode, 0644)
		if err != nil {
			return mediumResponse{}, err
		}
		if err := medium.WriteMode(req.Path, req.Content, mode); err != nil {
			return mediumResponse{}, err
		}
		return mediumResponse{OK: true}, nil
	case "ensuredir", "mkdir":
		if err := medium.EnsureDir(req.Path); err != nil {
			return mediumResponse{}, err
		}
		return mediumResponse{OK: true}, nil
	case "isfile":
		ok := medium.IsFile(req.Path)
		return mediumResponse{OK: true, IsFile: &ok}, nil
	case "delete":
		var err error
		if req.Recursive {
			err = medium.DeleteAll(req.Path)
		} else {
			err = medium.Delete(req.Path)
		}
		if err != nil {
			return mediumResponse{}, err
		}
		return mediumResponse{OK: true}, nil
	case "deleteall":
		if err := medium.DeleteAll(req.Path); err != nil {
			return mediumResponse{}, err
		}
		return mediumResponse{OK: true}, nil
	case "rename":
		if err := medium.Rename(req.OldPath, req.NewPath); err != nil {
			return mediumResponse{}, err
		}
		return mediumResponse{OK: true}, nil
	case "list":
		entries, err := medium.List(req.Path)
		if err != nil {
			return mediumResponse{}, err
		}
		return mediumResponse{OK: true, Entries: dirEntryDTOs(entries)}, nil
	case "stat":
		info, err := medium.Stat(req.Path)
		if err != nil {
			return mediumResponse{}, err
		}
		return mediumResponse{OK: true, Info: fileInfoDTOFromInfo(info)}, nil
	case "open":
		file, err := medium.Open(req.Path)
		if err != nil {
			return mediumResponse{}, err
		}
		defer file.Close()
		content, err := goio.ReadAll(file)
		if err != nil {
			return mediumResponse{}, err
		}
		return mediumResponse{OK: true, Content: string(content)}, nil
	case "create":
		writer, err := medium.Create(req.Path)
		if err != nil {
			return mediumResponse{}, err
		}
		if err := writeAndClose(writer, req.Content); err != nil {
			return mediumResponse{}, err
		}
		return mediumResponse{OK: true}, nil
	case "append":
		writer, err := medium.Append(req.Path)
		if err != nil {
			return mediumResponse{}, err
		}
		if err := writeAndClose(writer, req.Content); err != nil {
			return mediumResponse{}, err
		}
		return mediumResponse{OK: true}, nil
	case "readstream":
		reader, err := medium.ReadStream(req.Path)
		if err != nil {
			return mediumResponse{}, err
		}
		defer reader.Close()
		content, err := goio.ReadAll(reader)
		if err != nil {
			return mediumResponse{}, err
		}
		return mediumResponse{OK: true, Content: string(content)}, nil
	case "writestream":
		writer, err := medium.WriteStream(req.Path)
		if err != nil {
			return mediumResponse{}, err
		}
		if err := writeAndClose(writer, req.Content); err != nil {
			return mediumResponse{}, err
		}
		return mediumResponse{OK: true}, nil
	case "exists":
		ok := medium.Exists(req.Path)
		return mediumResponse{OK: true, Exists: &ok}, nil
	case "isdir":
		ok := medium.IsDir(req.Path)
		return mediumResponse{OK: true, IsDir: &ok}, nil
	default:
		return mediumResponse{}, fmt.Errorf("%w: %s", errUnsupportedMediumOperation, op)
	}
}

func bindPayload(c *gin.Context) (map[string]any, bool) {
	payload := map[string]any{}
	if c.Request == nil || c.Request.Body == nil || c.Request.ContentLength == 0 {
		return payload, true
	}
	decoder := json.NewDecoder(c.Request.Body)
	decoder.UseNumber()
	if err := decoder.Decode(&payload); err != nil {
		c.JSON(http.StatusBadRequest, goapi.Fail("invalid_request", err.Error()))
		return nil, false
	}
	return payload, true
}

func mediumRequestFromPayload(payload map[string]any) mediumRequest {
	return mediumRequest{
		Root:      stringValue(payload, "root"),
		Path:      stringValue(payload, "path"),
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
	c.JSON(http.StatusNotImplemented, goapi.Fail("not_implemented", message))
}

func resultErrorMessage(result core.Result) string {
	if err, ok := result.Value.(error); ok && err != nil {
		return err.Error()
	}
	if result.Value != nil {
		return fmt.Sprint(result.Value)
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
	case json.Number:
		return typed.String()
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
	case json.Number:
		if i, err := typed.Int64(); err == nil {
			return int(i)
		}
		if f, err := typed.Float64(); err == nil {
			return f
		}
		return typed.String()
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
	case json.Number:
		parsed, err := strconv.ParseInt(typed.String(), 0, 64)
		if err != nil {
			return 0, err
		}
		return fs.FileMode(parsed), nil
	case string:
		parsed, err := strconv.ParseInt(typed, 0, 64)
		if err != nil {
			return 0, err
		}
		return fs.FileMode(parsed), nil
	default:
		return 0, fmt.Errorf("unsupported file mode type %T", value)
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
		_ = writer.Close()
		return err
	}
	return writer.Close()
}
