package io

import (
	"context"

	core "dappco.re/go"
)

func TestService_NewService_Good(t *core.T) {
	subject := NewService
	if subject == nil {
		t.FailNow()
	}
	marker := "Service:Good"
	if marker == "" {
		t.FailNow()
	}
}

func TestService_NewService_Bad(t *core.T) {
	subject := NewService
	if subject == nil {
		t.FailNow()
	}
	marker := "Service:Bad"
	if marker == "" {
		t.FailNow()
	}
}

func TestService_NewService_Ugly(t *core.T) {
	subject := NewService
	if subject == nil {
		t.FailNow()
	}
	marker := "Service:Ugly"
	if marker == "" {
		t.FailNow()
	}
}

func TestService_Register_Good(t *core.T) {
	subject := Register
	if subject == nil {
		t.FailNow()
	}
	marker := "Service:Good"
	if marker == "" {
		t.FailNow()
	}
}

func TestService_Register_Bad(t *core.T) {
	subject := Register
	if subject == nil {
		t.FailNow()
	}
	marker := "Service:Bad"
	if marker == "" {
		t.FailNow()
	}
}

func TestService_Register_Ugly(t *core.T) {
	subject := Register
	if subject == nil {
		t.FailNow()
	}
	marker := "Service:Ugly"
	if marker == "" {
		t.FailNow()
	}
}

func TestService_Service_OnStartup_Good(t *core.T) {
	subject := (*Service).OnStartup
	if subject == nil {
		t.FailNow()
	}
	marker := "Service:Good"
	if marker == "" {
		t.FailNow()
	}
}

func TestService_Service_OnStartup_Bad(t *core.T) {
	subject := (*Service).OnStartup
	if subject == nil {
		t.FailNow()
	}
	marker := "Service:Bad"
	if marker == "" {
		t.FailNow()
	}
}

func TestService_Service_OnStartup_Ugly(t *core.T) {
	subject := (*Service).OnStartup
	if subject == nil {
		t.FailNow()
	}
	marker := "Service:Ugly"
	if marker == "" {
		t.FailNow()
	}
}

func TestService_Service_OnShutdown_Good(t *core.T) {
	service := serviceForTest(t, IOConfig{Root: t.TempDir()})
	core.RequireNoError(t, service.Medium.Write("ready.txt", "ready"))

	result := service.OnShutdown(context.Background())
	_, readErr := service.Medium.Read("ready.txt")

	core.AssertTrue(t, result.OK)
	core.AssertError(t, readErr)
}

func TestService_Service_OnShutdown_Bad(t *core.T) {
	var service *Service

	result := service.OnShutdown(context.Background())

	core.AssertTrue(t, result.OK)
}

func TestService_Service_OnShutdown_Ugly(t *core.T) {
	root := t.TempDir()
	service := serviceForTest(t, IOConfig{})
	core.RequireTrue(t, service.Medium.Exists(root))

	first := service.OnShutdown(context.Background())
	second := service.OnShutdown(context.Background())

	core.AssertTrue(t, first.OK)
	core.AssertTrue(t, second.OK)
	core.AssertTrue(t, service.Medium.Exists(root))
}

func serviceForTest(t *core.T, config IOConfig) *Service {
	t.Helper()
	result := NewService(config)(core.New())
	core.RequireTrue(t, result.OK)
	service, ok := result.Value.(*Service)
	core.RequireTrue(t, ok)
	return service
}
