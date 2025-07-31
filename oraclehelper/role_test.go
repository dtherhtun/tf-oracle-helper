package oraclehelper

import (
	"testing"
)

func TestRoleService(t *testing.T) {
	c, cleanup := setupTestClient(t)
	defer cleanup()

	// This test can not be run against an container db
	if c.ConName == "CDB$ROOT" {
		return
	}
	dbRole := ResourceRole{
		Role: "TESTROLE",
	}
	c.RoleService.CreateRole(dbRole)

	role, err := c.RoleService.ReadRole(dbRole)
	if err != nil {
		t.Fatalf("failed to read role, errormsg: %v\n", err)
	}

	if "TESTROLE" != role.Role {
		t.Errorf("%v; want %v\n", role.Role, "TESTROLE")
	}

	c.RoleService.DropRole(dbRole)

}

func TestModifyRole(t *testing.T) {
	c, cleanup := setupTestClient(t)
	defer cleanup()

	// This test can not be run against an container db
	if c.ConName == "CDB$ROOT" {
		return
	}
	dbRole := ResourceRole{
		Role: "TESTMODIFYROLE",
	}

	err := c.RoleService.CreateRole(dbRole)
	if err != nil {
		t.Fatalf("failed to create role: %v", err)
	}

	err = c.RoleService.ModifyRole(dbRole)
	if err != nil {
		t.Errorf("ModifyRole failed with error: %v", err)
	}

	err = c.RoleService.DropRole(dbRole)
	if err != nil {
		t.Fatalf("failed to drop role: %v", err)
	}
}