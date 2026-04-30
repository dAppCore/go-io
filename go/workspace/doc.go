// Example: service, _ := workspace.New(workspace.Options{
// Example:     KeyPairProvider: keyPairProvider,
// Example:     RootPath: "/srv/workspaces",
// Example:     Medium: io.NewMemoryMedium(),
// Example: })
// Example: workspaceID, _ := service.CreateWorkspace("alice", "pass123")
// Example: _ = service.SwitchWorkspace(workspaceID)
// Example: _ = service.WriteWorkspaceFile("notes/todo.txt", "ship it")
package workspace
