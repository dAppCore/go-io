// SPDX-License-Identifier: EUPL-1.2

package local

import core "github.com/dappcore/go"

func TestLocal_driveOf_Good(t *core.T) {
	core.AssertEqual(t, "C:", driveOf("C:/Users/snide"))
	core.AssertEqual(t, "d:", driveOf("d:/data"))
}

func TestLocal_driveOf_Bad(t *core.T) {
	core.AssertEqual(t, "", driveOf("/srv/app"))
	core.AssertEqual(t, "", driveOf("relative/path"))
}

func TestLocal_driveOf_Ugly(t *core.T) {
	core.AssertEqual(t, "", driveOf("C"))    // too short
	core.AssertEqual(t, "", driveOf("1:/x")) // not a letter
	core.AssertEqual(t, "C:", driveOf("C:")) // the bare drive
}
