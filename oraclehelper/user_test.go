package oraclehelper

import (
	"fmt"
	"testing"
)

func TestUserService(t *testing.T) {
	// This test can not be run against an container db
	if c.ConName == "CDB$ROOT" {
		return
	}
	quota := make(map[string]string)
	quota["USERS"] = "unlimited"
	quota["SYSTEM"] = "10m"
	quota["SYSAUX"] = "10G"
	err := c.UserService.CreateUser(ResourceUser{Username: "TEST01"})
	if err != nil {
		t.Fatalf("CreateUser failed with error: %v", err)
	}
	c.ProfileService.CreateProfile(ResourceProfile{Profile: "PP01"})
	user, _ := c.UserService.ReadUser(ResourceUser{Username: "TEST01"})
	if user.Profile != "DEFAULT" {
		t.Errorf("want: %s, gott: %s", "DEFAULT", user.Profile)
	}
	c.UserService.ModifyUser(ResourceUser{Username: "TEST01", DefaultTablespace: "SYSTEM", Quota: quota, Profile: "PP01"})

	user, _ = c.UserService.ReadUser(ResourceUser{Username: "TEST01"})
	if "SYSTEM" != user.DefaultTablespace {
		t.Errorf("%v; want %v\n", user.DefaultTablespace, "SYSTEM")
	}
	if user.Quota["SYSTEM"] != "10M" {
		t.Errorf("gott: %s; want:%s\n", user.Quota["SYSTEM"], "10M")
	}
	if user.Quota["USERS"] != "unlimited" {
		t.Errorf("%s; want %s\n", user.Quota["USERS"], "unlimited")
	}
	if user.Quota["SYSAUX"] != "10G" {
		t.Errorf("%s; want %s\n", user.Quota["SYSAUX"], "10G")
	}
	if user.Profile != "PP01" {
		t.Errorf("want: %s, gott: %s", "PP01", user.Profile)
	}
	if user.AccountStatus != "OPEN" {
		t.Errorf("want: %s, gott: %s", "OPEN", user.AccountStatus)
	}
	c.DBClient.Exec(fmt.Sprintf("alter user %s password expire", "TEST01"))
	user, _ = c.UserService.ReadUser(ResourceUser{Username: "TEST01"})
	if user.AccountStatus != "EXPIRED" {
		t.Errorf("want: %s, gott: %s", "EXPIRED", user.AccountStatus)
	}
	c.UserService.ModifyUser(ResourceUser{Username: "TEST01", AccountStatus: "LOCK"})
	user, _ = c.UserService.ReadUser(ResourceUser{Username: "TEST01"})
	if user.AccountStatus != "EXPIRED & LOCKED" {
		t.Errorf("want: %s, gott: %s", "EXPIRED & LOCKED", user.AccountStatus)
	}
	c.UserService.DropUser(ResourceUser{Username: "TEST01"})
	c.ProfileService.DeleteProfile(ResourceProfile{Profile: "PP01"})
}
