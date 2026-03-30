// Package workspace provides encrypted user workspaces backed by io.Medium.
//
//	service, _ := workspace.New(workspace.Options{Core: core.New(), Crypt: cryptProvider})
//	workspaceID, _ := service.CreateWorkspace("alice", "pass123")
//	_ = service.SwitchWorkspace(workspaceID)
//	_ = service.WorkspaceFileSet("notes/todo.txt", "ship it")
//
// Workspaces are rooted under the caller's configured home directory and keep
// file access constrained to the active workspace.
package workspace
