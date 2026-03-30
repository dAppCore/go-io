// Package workspace creates encrypted workspaces on top of io.Medium.
//
//	service, _ := workspace.New(workspace.Options{Core: core.New(), CryptProvider: cryptProvider})
//	workspaceID, _ := service.CreateWorkspace("alice", "pass123")
//	_ = service.SwitchWorkspace(workspaceID)
//	_ = service.WorkspaceFileSet("notes/todo.txt", "ship it")
package workspace
