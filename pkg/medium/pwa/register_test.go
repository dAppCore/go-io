package pwa

import (
	core "dappco.re/go"
)

func TestRegister_RegisterFactory_Good(t *core.T) {
	result := RegisterFactory("ax7-pwa", New)
	factory, ok := FactoryFor("ax7-pwa")
	core.AssertTrue(t, result.OK)
	core.AssertTrue(t, ok)
	core.AssertNotNil(t, factory)
}

func TestRegister_RegisterFactory_Bad(t *core.T) {
	result := RegisterFactory("ax7-pwa-nil", nil)
	factory, ok := FactoryFor("ax7-pwa-nil")
	core.AssertTrue(t, result.OK)
	core.AssertTrue(t, ok)
	core.AssertNil(t, factory)
}

func TestRegister_RegisterFactory_Ugly(t *core.T) {
	result := RegisterFactory("", New)
	factory, ok := FactoryFor("")
	core.AssertTrue(t, result.OK)
	core.AssertTrue(t, ok)
	core.AssertNotNil(t, factory)
}

func TestRegister_FactoryFor_Good(t *core.T) {
	factory, ok := FactoryFor(Scheme)
	core.AssertTrue(t, ok)
	core.AssertNotNil(t, factory)
}

func TestRegister_FactoryFor_Bad(t *core.T) {
	factory, ok := FactoryFor("missing-pwa")
	core.AssertFalse(t, ok)
	core.AssertNil(t, factory)
}

func TestRegister_FactoryFor_Ugly(t *core.T) {
	RegisterFactory("ax7-pwa-empty", New)
	factory, ok := FactoryFor("ax7-pwa-empty")
	core.AssertTrue(t, ok)
	core.AssertNotNil(t, factory)
}

func TestRegister_RegisterActions_Good(t *core.T) {
	c := core.New()
	RegisterActions(c)
	core.AssertTrue(t, c.Action(ActionRead).Exists())
	core.AssertTrue(t, c.Action(ActionWrite).Exists())
}

func TestRegister_RegisterActions_Bad(t *core.T) {
	core.AssertNotPanics(t, func() { RegisterActions(nil) })
	c := core.New()
	core.AssertFalse(t, c.Action(ActionRead).Exists())
}

func TestRegister_RegisterActions_Ugly(t *core.T) {
	c := core.New()
	RegisterActions(c)
	core.AssertNotPanics(t, func() { RegisterActions(c) })
}
