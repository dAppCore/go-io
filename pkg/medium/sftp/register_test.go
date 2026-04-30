package sftp

import (
	"context"
	core "dappco.re/go"
)

func TestRegister_RegisterFactory_Good(t *core.T) {
	result := RegisterFactory("ax7-sftp-good", New)
	core.AssertTrue(t, result.OK)
	factory, ok := FactoryFor("ax7-sftp-good")
	core.AssertTrue(t, ok)
	core.AssertNotNil(t, factory)
}

func TestRegister_RegisterFactory_Bad(t *core.T) {
	result := RegisterFactory("ax7-sftp-bad", nil)
	core.AssertTrue(t, result.OK)
	factory, ok := FactoryFor("ax7-sftp-bad")
	core.AssertTrue(t, ok)
	core.AssertNil(t, factory)
}

func TestRegister_RegisterFactory_Ugly(t *core.T) {
	result := RegisterFactory("ax7-sftp-ugly", New)
	core.AssertTrue(t, result.OK)
	result = RegisterFactory("ax7-sftp-ugly", New)
	core.AssertTrue(t, result.OK)
}

func TestRegister_FactoryFor_Good(t *core.T) {
	result := RegisterFactory("ax7-sftp-factory", New)
	core.AssertTrue(t, result.OK)
	factory, ok := FactoryFor("ax7-sftp-factory")
	core.AssertTrue(t, ok)
	core.AssertNotNil(t, factory)
}

func TestRegister_FactoryFor_Bad(t *core.T) {
	factory, ok := FactoryFor("missing-sftp-factory")
	core.AssertFalse(t, ok)
	core.AssertNil(t, factory)
}

func TestRegister_FactoryFor_Ugly(t *core.T) {
	factory, ok := FactoryFor("")
	core.AssertFalse(t, ok)
	core.AssertNil(t, factory)
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
	result := c.Action(ActionRead).Run(context.Background(), core.NewOptions())
	core.AssertFalse(t, result.OK)
}
