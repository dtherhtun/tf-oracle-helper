package oraclehelper

import (
	"fmt"
	"log"
	"testing"
)

func TestGrantServiceObjectGrants(t *testing.T) {
	// This test can not be run against an container db
	if c.ConName == "CDB$ROOT" {
		return
	}
	c.DBClient.Exec("drop table system.test")
	c.DBClient.Exec("create table system.test(col1 number)")
	objGrant := ResourceGrantObjectPrivilege{
		Grantee:    "GRANTTST01",
		Privilege:  []string{"SELECT"},
		Owner:      "SYSTEM",
		ObjectName: "TEST",
	}
	objGrant2 := ResourceGrantObjectPrivilege{
		Grantee:    "GRANTTST01",
		Privilege:  []string{"SELECT", "UPDATE"},
		Owner:      "SYSTEM",
		ObjectName: "TEST",
	}
	c.UserService.CreateUser(ResourceUser{Username: "GRANTTST01"})
	c.GrantService.GrantObjectPrivilege(objGrant)
	grants, err := c.GrantService.ReadGrantObjectPrivilege(objGrant)
	if err != nil {
		log.Fatalf("failed to read role, errormsg: %v\n", err)
	}

	if grants.Privileges[0] != "SELECT" {
		t.Errorf("Wanted: %s gott: %s", "SELECT", grants.Privileges[0])
	}
	c.GrantService.RevokeObjectPrivilege(objGrant)
	grants, err = c.GrantService.ReadGrantObjectPrivilege(objGrant)
	if err != nil {
		log.Fatalf("failed to read role, errormsg: %v\n", err)
	}
	if len(grants.Privileges) != 0 {
		t.Errorf("Wanted: %d gott: %d", 0, len(grants.Privileges))
	}

	//Test grant multiple privs
	c.GrantService.GrantObjectPrivilege(objGrant2)
	grants, err = c.GrantService.ReadGrantObjectPrivilege(objGrant2)
	if err != nil {
		log.Fatalf("failed to read role, errormsg: %v\n", err)
	}

	if len(grants.Privileges) != 2 {
		t.Errorf("Wanted: %d gott: %d", 2, len(grants.Privileges))
	}

	// Revoke insert from
	c.GrantService.RevokeObjectPrivilege(objGrant)
	grants, err = c.GrantService.ReadGrantObjectPrivilege(objGrant)
	if err != nil {
		log.Fatalf("failed to read role, errormsg: %v\n", err)
	}
	if len(grants.Privileges) != 1 {
		t.Errorf("Wanted: %d gott: %d", 1, len(grants.Privileges))
	}
	//Clean up
	c.UserService.DropUser(ResourceUser{Username: "GRANTTST01"})
}

func TestGrantServiceSysPrivsGrants(t *testing.T) {
	// This test can not be run against an container db
	if c.ConName == "CDB$ROOT" {
		return
	}
	username := "GRANTSYSPRIVTEST"
	sysPrivs := ResourceGrantSystemPrivilege{
		Grantee:   username,
		Privilege: "CREATE SESSION",
	}
	c.UserService.CreateUser(ResourceUser{Username: username})

	err := c.GrantService.GrantSysPriv(sysPrivs)
	if err != nil {
		t.Errorf("GrantSysPriv FAiled")
	}
	userSysPrivs, err := c.GrantService.ReadGrantSysPrivs(sysPrivs)
	if err != nil {
		t.Errorf("ReadGrantSysPrivs Failed")
	}
	if value, ok := userSysPrivs["CREATE SESSION"]; !ok {
		t.Errorf("CREATE SESSION not exists value: %v\n", value)
	}
	//Clean up
	c.UserService.DropUser(ResourceUser{Username: username})

}

func TestGrantServiceRolePrivs(t *testing.T) {
	// This test can not be run against an container db
	if c.ConName == "CDB$ROOT" {
		return
	}
	username := "GRANTROLEPRIVTEST"
	resourceUsername := ResourceUser{Username: username}
	dbRole := ResourceRole{
		Role: "TESTROLE",
	}
	rolePrivs := ResourceGrantRolePrivilege{
		Grantee: username,
		Role:    dbRole.Role,
	}
	c.UserService.CreateUser(resourceUsername)
	c.RoleService.CreateRole(dbRole)

	err := c.GrantService.GrantRolePriv(rolePrivs)
	if err != nil {
		t.Errorf("GrantRolePriv Failed")
	}

	roles, err := c.GrantService.ReadGrantRolePrivs(rolePrivs)
	if err != nil {
		t.Errorf("ReadGrantRolePrivs Failed")
	}
	if value, ok := roles[dbRole.Role]; !ok {
		t.Errorf("%s not exists value: %v\n", dbRole.Role, value)
	}

	c.RoleService.DropRole(dbRole)
	c.UserService.DropUser(resourceUsername)
}

func TestGrantServiceWholeSchemaToUser(t *testing.T) {
	// This test can not be run against an container db
	if c.ConName == "CDB$ROOT" {
		return
	}
	username := "TSTSCHEMA"
	want := "d49cb717d004498a4ea71d1742ff5755a0e295360515e96835ba43547a0059c9"
	c.DBClient.Exec(fmt.Sprintf("DROP USER %s CASCADE", username))
	resourceUsername := ResourceUser{Username: username}
	objGrant := ResourceGrantObjectPrivilege{
		Owner: username,
	}
	usrErr := c.UserService.CreateUser(resourceUsername)
	if usrErr != nil {
		t.Errorf("error: %v\n", usrErr)
	}

	c.DBClient.Exec(fmt.Sprintf("CREATE TABLE %s.tbl1(col1 varchar2(30))", username))
	c.DBClient.Exec(fmt.Sprintf("CREATE TABLE %s.tbl2(col1 varchar2(30))", username))
	got, err := c.GrantService.GetHashSchemaAllTables(objGrant)
	if err != nil {
		t.Errorf("error: %v\n", err)
	}

	if want != got {
		t.Errorf("want: %s got:%s \n", want, got)
	}

	c.DBClient.Exec(fmt.Sprintf("DROP USER %s CASCADE", username))

}
func TestGrantServiceGetHashSchemaPrivsToUser(t *testing.T) {
	// This test can not be run against an container db
	if c.ConName == "CDB$ROOT" {
		return
	}
	username1 := "TSTSCHEMA1"
	username2 := "TSTSCHEMA2"
	want := "cec6418210ba8b444c741c06a91390888287d5dae6bdb9eeff4ac8e081d13690"
	c.DBClient.Exec(fmt.Sprintf("DROP USER %s CASCADE", username1))
	c.DBClient.Exec(fmt.Sprintf("DROP USER %s CASCADE", username2))
	resourceUsername := ResourceUser{Username: username1}
	objGrant := ResourceGrantObjectPrivilege{
		Owner:      username1,
		Grantee:    username2,
		ObjectName: "TBL1",
		Privilege:  []string{"SELECT"},
	}
	usrErr := c.UserService.CreateUser(resourceUsername)
	if usrErr != nil {
		t.Errorf("error: %v\n", usrErr)
	}
	resourceUsername = ResourceUser{Username: username2}
	usrErr = c.UserService.CreateUser(resourceUsername)
	if usrErr != nil {
		t.Errorf("error: %v\n", usrErr)
	}
	c.DBClient.Exec(fmt.Sprintf("CREATE TABLE %s.tbl1(col1 varchar2(30))", username1))
	c.GrantService.GrantObjectPrivilege(objGrant)
	got, err := c.GrantService.GetHashSchemaPrivsToUser(objGrant)
	if err != nil {
		t.Errorf("error: %v\n", err)
	}

	if want != got {
		t.Errorf("want: %s got:%s \n", want, got)
	}

	c.DBClient.Exec(fmt.Sprintf("DROP USER %s CASCADE", username1))
	c.DBClient.Exec(fmt.Sprintf("DROP USER %s CASCADE", username2))

}

func TestGrantServiceCheckGrantSchemaDiffLogic(t *testing.T) {
	// This test can not be run against an container db
	if c.ConName == "CDB$ROOT" {
		return
	}
	username1 := "TSTSCHEMA1"
	username2 := "TSTSCHEMA2"

	c.DBClient.Exec(fmt.Sprintf("DROP USER %s CASCADE", username1))
	c.DBClient.Exec(fmt.Sprintf("DROP USER %s CASCADE", username2))

	resourceUsername := ResourceUser{Username: username1}
	usrErr := c.UserService.CreateUser(resourceUsername)
	if usrErr != nil {
		t.Errorf("error: %v\n", usrErr)
	}
	resourceUsername = ResourceUser{Username: username2}
	usrErr = c.UserService.CreateUser(resourceUsername)

	c.DBClient.Exec(fmt.Sprintf("CREATE TABLE %s.tbl1(col1 varchar2(30))", username1))
	objGrant := ResourceGrantObjectPrivilege{
		Owner:      username1,
		Grantee:    username2,
		ObjectName: "TBL1",
		Privilege:  []string{"SELECT"},
	}
	c.GrantService.GrantObjectPrivilege(objGrant)
	grantHash, _ := c.GrantService.GetHashSchemaPrivsToUser(objGrant)
	allTableHash, _ := c.GrantService.GetHashSchemaAllTables(objGrant)

	if grantHash != allTableHash {
		t.Errorf("allTableHash: %s grantHash:%s \n", allTableHash, grantHash)
	}

	c.DBClient.Exec(fmt.Sprintf("CREATE TABLE %s.tbl2(col1 varchar2(30))", username1))
	grantHash, _ = c.GrantService.GetHashSchemaPrivsToUser(objGrant)
	allTableHash, _ = c.GrantService.GetHashSchemaAllTables(objGrant)
	if grantHash == allTableHash {
		t.Errorf("allTableHash: %s grantHash:%s \n", allTableHash, grantHash)
	}

	c.GrantService.GrantTableSchemaToUser(objGrant)
	grantHash, _ = c.GrantService.GetHashSchemaPrivsToUser(objGrant)
	allTableHash, _ = c.GrantService.GetHashSchemaAllTables(objGrant)
	if grantHash != allTableHash {
		t.Errorf("allTableHash: %s grantHash:%s \n", allTableHash, grantHash)
	}
	c.DBClient.Exec(fmt.Sprintf("DROP USER %s CASCADE", username1))
	c.DBClient.Exec(fmt.Sprintf("DROP USER %s CASCADE", username2))
}

func TestGrantServiceCRevokeSchemaFromUser(t *testing.T) {
	// This test can not be run against an container db
	if c.ConName == "CDB$ROOT" {
		return
	}
	username1 := "TSTSCHEMA1"
	username2 := "TSTSCHEMA2"

	c.DBClient.Exec(fmt.Sprintf("DROP USER %s CASCADE", username1))
	c.DBClient.Exec(fmt.Sprintf("DROP USER %s CASCADE", username2))

	resourceUsername := ResourceUser{Username: username1}
	usrErr := c.UserService.CreateUser(resourceUsername)
	if usrErr != nil {
		t.Errorf("error: %v\n", usrErr)
	}
	resourceUsername = ResourceUser{Username: username2}
	usrErr = c.UserService.CreateUser(resourceUsername)

	c.DBClient.Exec(fmt.Sprintf("CREATE TABLE %s.tbl1(col1 varchar2(30))", username1))
	c.DBClient.Exec(fmt.Sprintf("CREATE TABLE %s.tbl2(col1 varchar2(30))", username1))
	objGrant := ResourceGrantObjectPrivilege{
		Owner:     username1,
		Grantee:   username2,
		Privilege: []string{"SELECT", "UPDATE"},
	}
	c.GrantService.GrantTableSchemaToUser(objGrant)
	var result int
	c.DBClient.QueryRow("select count(*) as antal from dba_tab_privs where grantee = 'TSTSCHEMA2' and owner = 'TSTSCHEMA1'").Scan(&result)

	c.GrantService.RevokeTableSchemaFromUser(objGrant)

	c.DBClient.QueryRow("select count(*) as antal from dba_tab_privs where grantee = 'TSTSCHEMA2' and owner = 'TSTSCHEMA1'").Scan(&result)

	if result != 0 {
		t.Error("Result should be 0 after revoking all privileges")
	}
	c.DBClient.Exec(fmt.Sprintf("DROP USER %s CASCADE", username1))
	c.DBClient.Exec(fmt.Sprintf("DROP USER %s CASCADE", username2))
}
