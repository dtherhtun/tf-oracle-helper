package oraclehelper

import (
	"log"
	"testing"
)

func TestRoleService(t *testing.T) {
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
		log.Fatalf("failed to read role, errormsg: %v\n", err)
	}

	if "TESTROLE" != role.Role {
		t.Errorf("%v; want %v\n", role.Role, "TESTROLE")
	}

	c.RoleService.DropRole(dbRole)

}
