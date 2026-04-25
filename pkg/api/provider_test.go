// SPDX-License-Identifier: EUPL-1.2

package api

import (
	"testing"

	coreprovider "dappco.re/go/api/pkg/provider"
	core "dappco.re/go/core"
	coreio "dappco.re/go/io"
)

func TestNewProvider_Good(t *testing.T) {
	c := core.New()
	provider := NewProvider(c)

	var _ coreprovider.Provider = provider
	var _ coreprovider.Describable = provider

	if provider.Name() != "io" {
		t.Fatalf("expected provider name io, got %q", provider.Name())
	}
	if provider.BasePath() != "/v1" {
		t.Fatalf("expected base path /v1, got %q", provider.BasePath())
	}
	if !c.Action(coreio.ActionMemoryRead).Exists() {
		t.Fatalf("expected %s to be registered", coreio.ActionMemoryRead)
	}
	if got := len(rfc15Actions); got != 18 {
		t.Fatalf("expected 18 RFC actions, got %d", got)
	}
}

func TestNewProvider_Bad(t *testing.T) {
	provider := NewProvider(nil)
	if provider == nil {
		t.Fatal("expected provider")
	}
	if provider.core == nil {
		t.Fatal("expected provider core registry")
	}
	if !provider.core.Action(coreio.ActionLocalRead).Exists() {
		t.Fatalf("expected %s to be registered on default core", coreio.ActionLocalRead)
	}
}

func TestNewProvider_Ugly(t *testing.T) {
	c := core.New()
	coreio.RegisterActions(c)

	provider := NewProvider(c)
	if provider == nil {
		t.Fatal("expected provider")
	}
	if !provider.core.Action(coreio.ActionCopy).Exists() {
		t.Fatalf("expected %s to remain registered after duplicate registration", coreio.ActionCopy)
	}
	if len(provider.Describe()) != 5 {
		t.Fatalf("expected 5 route descriptions, got %d", len(provider.Describe()))
	}
}
