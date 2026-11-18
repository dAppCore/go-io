package local

import (
	goio "io"
	"io/fs"
	"syscall"

	core "github.com/dappcore/go"
)

func TestLocal_Read_ComponentSwapCannotEscape_Ugly(t *core.T) {
	root := t.TempDir()
	outside := t.TempDir()
	core.RequireTrue(t, core.WriteFile(
		core.Path(outside, "secret.txt"),
		[]byte("outside"),
		0600,
	).OK)
	core.RequireTrue(t, core.MkdirAll(core.Path(root, "pivot"), 0700).OK)
	core.RequireTrue(t, core.WriteFile(
		core.Path(root, "pivot", "secret.txt"),
		[]byte("inside"),
		0600,
	).OK)
	medium, err := New(root)
	core.RequireNoError(t, err)
	medium.beforeRootOperation = func() {
		core.RequireNoError(t, syscall.Rename(
			core.Path(root, "pivot"),
			core.Path(root, "held"),
		))
		core.RequireNoError(t, syscall.Symlink(
			outside,
			core.Path(root, "pivot"),
		))
	}

	content, err := medium.Read("pivot/secret.txt")

	core.AssertError(t, err)
	core.AssertNotEqual(t, "outside", content)
}

func TestLocal_InternalRelativeSymlink_Good(t *core.T) {
	root := t.TempDir()
	core.RequireTrue(t, core.MkdirAll(core.Path(root, "inside"), 0700).OK)
	core.RequireTrue(t, core.WriteFile(
		core.Path(root, "inside", "secret.txt"),
		[]byte("inside"),
		0600,
	).OK)
	core.RequireNoError(t, syscall.Symlink(
		"inside",
		core.Path(root, "current"),
	))
	medium, err := New(root)
	core.RequireNoError(t, err)
	t.Cleanup(func() {
		core.AssertNoError(t, medium.Close())
	})

	content, err := medium.Read("current/secret.txt")

	core.AssertNoError(t, err)
	core.AssertEqual(t, "inside", content)
}

func TestLocal_EscapingSymlink_Bad(t *core.T) {
	root := t.TempDir()
	outside := t.TempDir()
	core.RequireTrue(t, core.WriteFile(
		core.Path(outside, "secret.txt"),
		[]byte("outside"),
		0600,
	).OK)
	core.RequireNoError(t, syscall.Symlink(
		outside,
		core.Path(root, "escape"),
	))
	medium, err := New(root)
	core.RequireNoError(t, err)
	t.Cleanup(func() {
		core.AssertNoError(t, medium.Close())
	})

	content, err := medium.Read("escape/secret.txt")

	core.AssertError(t, err)
	core.AssertNotEqual(t, "outside", content)
}

func TestLocal_ComponentSwapOperationsCannotEscape_Ugly(t *core.T) {
	operations := []struct {
		name string
		run  func(*Medium) error
	}{
		{
			name: "Open",
			run: func(medium *Medium) error {
				file, err := medium.Open("pivot/secret.txt")
				if file != nil {
					_ = file.Close()
				}
				return err
			},
		},
		{
			name: "Create",
			run: func(medium *Medium) error {
				writer, err := medium.Create("pivot/created.txt")
				if writer != nil {
					_ = writer.Close()
				}
				return err
			},
		},
		{
			name: "Append",
			run: func(medium *Medium) error {
				writer, err := medium.Append("pivot/secret.txt")
				if writer != nil {
					_, _ = writer.Write([]byte("-changed"))
					_ = writer.Close()
				}
				return err
			},
		},
		{
			name: "List",
			run: func(medium *Medium) error {
				_, err := medium.List("pivot")
				return err
			},
		},
		{
			name: "Stat",
			run: func(medium *Medium) error {
				_, err := medium.Stat("pivot/secret.txt")
				return err
			},
		},
		{
			name: "Delete",
			run: func(medium *Medium) error {
				return medium.Delete("pivot/secret.txt")
			},
		},
		{
			name: "DeleteAll",
			run: func(medium *Medium) error {
				return medium.DeleteAll("pivot/tree")
			},
		},
		{
			name: "Rename",
			run: func(medium *Medium) error {
				return medium.Rename("pivot/secret.txt", "moved.txt")
			},
		},
	}

	for _, operation := range operations {
		t.Run(operation.name, func(t *core.T) {
			root, outside, medium := componentSwapFixture(t)
			medium.beforeRootOperation = func() {
				core.RequireNoError(t, syscall.Rename(
					core.Path(root, "pivot"),
					core.Path(root, "held"),
				))
				core.RequireNoError(t, syscall.Symlink(
					outside,
					core.Path(root, "pivot"),
				))
			}

			err := operation.run(medium)

			core.AssertError(t, err)
			outsideContent := hostFileContent(
				t,
				core.Path(outside, "secret.txt"),
			)
			core.AssertEqual(t, "outside", outsideContent)
			treeContent := hostFileContent(
				t,
				core.Path(outside, "tree", "secret.txt"),
			)
			core.AssertEqual(t, "outside-tree", treeContent)
			core.AssertFalse(
				t,
				hostFileExists(core.Path(outside, "created.txt")),
			)
		})
	}
}

func TestMedium_Medium_Close_Good(t *core.T) {
	medium, err := New(t.TempDir())
	core.RequireNoError(t, err)
	core.RequireNoError(t, medium.Write("ready.txt", "ready"))

	err = medium.Close()
	_, readErr := medium.Read("ready.txt")

	core.AssertNoError(t, err)
	core.AssertError(t, readErr)
}

func TestMedium_Medium_Close_Bad(t *core.T) {
	var medium *Medium

	err := medium.Close()

	core.AssertError(t, err)
}

func TestMedium_Medium_Close_Ugly(t *core.T) {
	medium, err := New(t.TempDir())
	core.RequireNoError(t, err)

	first := medium.Close()
	second := medium.Close()

	core.AssertNoError(t, first)
	core.AssertNoError(t, second)
}

func TestLocal_DeleteRoot_Bad(t *core.T) {
	root := t.TempDir()
	medium, err := New(root)
	core.RequireNoError(t, err)
	t.Cleanup(func() {
		core.AssertNoError(t, medium.Close())
	})

	deleteErr := medium.Delete("")
	deleteAllErr := medium.DeleteAll("")

	core.AssertError(t, deleteErr)
	core.AssertError(t, deleteAllErr)
	core.AssertTrue(t, hostFileExists(root))
}

func componentSwapFixture(t *core.T) (string, string, *Medium) {
	t.Helper()
	root := t.TempDir()
	outside := t.TempDir()
	core.RequireTrue(t, core.MkdirAll(
		core.Path(root, "pivot", "tree"),
		0700,
	).OK)
	core.RequireTrue(t, core.MkdirAll(
		core.Path(outside, "tree"),
		0700,
	).OK)
	for hostPath, content := range map[string]string{
		core.Path(root, "pivot", "secret.txt"):         "inside",
		core.Path(root, "pivot", "tree", "secret.txt"): "inside-tree",
		core.Path(outside, "secret.txt"):               "outside",
		core.Path(outside, "tree", "secret.txt"):       "outside-tree",
	} {
		core.RequireTrue(t, core.WriteFile(
			hostPath,
			[]byte(content),
			0600,
		).OK)
	}
	medium, err := New(root)
	core.RequireNoError(t, err)
	t.Cleanup(func() {
		core.AssertNoError(t, medium.Close())
	})
	return root, outside, medium
}

func hostFileContent(t *core.T, hostPath string) string {
	t.Helper()
	result := core.ReadFile(hostPath)
	core.RequireTrue(t, result.OK)
	payload, ok := result.Value.([]byte)
	core.RequireTrue(t, ok)
	return string(payload)
}

func hostFileExists(hostPath string) bool {
	return core.Stat(hostPath).OK
}

var _ goio.Closer = (*Medium)(nil)
var _ fs.FS = (*Medium)(nil)
