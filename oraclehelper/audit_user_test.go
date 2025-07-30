package oraclehelper

import (
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/acctest"
)

func TestAuditRead(t *testing.T) {
	// This test can not be run against an container db
	if c.ConName == "CDB$ROOT" {
		return
	}
	audit := ResourceAuditUser{
		UserName: "BB",
	}

	result, err := c.AuditUserService.ReadAudit(audit)
	if err != nil {
		t.Logf("error: %v", err)
	}
	t.Logf("result: %v", result)

}

func TestAuditSetAudit(t *testing.T) {
	// This test can not be run against an container db
	if c.ConName == "CDB$ROOT" {
		return
	}
	username := acctest.RandStringFromCharSet(10, "ABCDEFGHIJKLMNOPQRSTUVWXYZ")
	t.Logf("TestAuditSetAudit using username: %s for this test run", username)
	quota := make(map[string]string)
	quota["USERS"] = "unlimited"
	quota["SYSTEM"] = "10m"
	quota["SYSAUX"] = "10G"
	tt := []struct {
		name        string
		auditOption []AuditOption
		username    string
	}{
		{
			"SingeOptionBothByAccess",
			[]AuditOption{AuditOption{
				Option:  "ALTER SYSTEM",
				Success: "BY ACCESS",
				Failure: "BY ACCESS"},
			},
			acctest.RandStringFromCharSet(10, "ABCDEFGHIJKLMNOPQRSTUVWXYZ"),
		},
		{
			"MultipleOptionBothByAccess",
			[]AuditOption{
				AuditOption{
					Option:  "ALTER SYSTEM",
					Success: "BY ACCESS",
					Failure: "BY ACCESS",
				},
				AuditOption{
					Option:  "ROLE",
					Success: "BY ACCESS",
					Failure: "BY ACCESS",
				},
			},
			acctest.RandStringFromCharSet(10, "ABCDEFGHIJKLMNOPQRSTUVWXYZ"),
		},
		{
			"MultipleOptionBothByFailure",
			[]AuditOption{
				AuditOption{
					Option:  "ALTER SYSTEM",
					Success: auditNotSet,
					Failure: "BY ACCESS",
				},
				AuditOption{
					Option:  "ROLE",
					Success: auditNotSet,
					Failure: "BY ACCESS",
				},
			},
			acctest.RandStringFromCharSet(10, "ABCDEFGHIJKLMNOPQRSTUVWXYZ"),
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			c.UserService.CreateUser(ResourceUser{Username: tc.username})
			err := c.AuditUserService.Audit(ResourceAuditUser{
				UserName:    tc.username,
				AuditOption: tc.auditOption,
			})
			if err != nil {
				t.Fatalf("Failed set audit failed with error error: %v", err)
			}
			result, _ := c.AuditUserService.ReadAudit(ResourceAuditUser{
				UserName: tc.username,
			})
			if !reflect.DeepEqual(result.AuditOption, tc.auditOption) {
				t.Fatalf("audit option diff got: %v want: %v", result.AuditOption, tc.auditOption)
				c.UserService.DropUser(ResourceUser{Username: tc.username})
			}
			c.UserService.DropUser(ResourceUser{Username: tc.username})
		})
	}
}

func TestAuditRemoveAudit(t *testing.T) {
	// This test can not be run against an container db
	if c.ConName == "CDB$ROOT" {
		return
	}
	t.Logf("TestAuditSetAudit")
	quota := make(map[string]string)
	quota["USERS"] = "unlimited"
	quota["SYSTEM"] = "10m"
	quota["SYSAUX"] = "10G"
	tt := []struct {
		name            string
		setAuditOption  []AuditOption
		rmAuditOption   []AuditOption
		wantauditOption []AuditOption
		username        string
	}{
		{
			"SingeOptionBothByAccess",
			[]AuditOption{AuditOption{
				Option:  "ALTER SYSTEM",
				Success: "BY ACCESS",
				Failure: "BY ACCESS"},
			},
			[]AuditOption{AuditOption{
				Option:  "ALTER SYSTEM",
				Success: "BY ACCESS",
				Failure: "BY ACCESS"},
			},
			[]AuditOption{},
			acctest.RandStringFromCharSet(10, "ABCDEFGHIJKLMNOPQRSTUVWXYZ"),
		},
		{
			"MultipleOptionBothByAccess",
			[]AuditOption{
				AuditOption{
					Option:  "ALTER SYSTEM",
					Success: "BY ACCESS",
					Failure: "BY ACCESS",
				},
				AuditOption{
					Option:  "ROLE",
					Success: "BY ACCESS",
					Failure: "BY ACCESS",
				},
			},
			[]AuditOption{AuditOption{
				Option:  "ALTER SYSTEM",
				Success: "BY ACCESS",
				Failure: "BY ACCESS"},
			},
			[]AuditOption{AuditOption{
				Option:  "ROLE",
				Success: "BY ACCESS",
				Failure: "BY ACCESS"},
			},
			acctest.RandStringFromCharSet(10, "ABCDEFGHIJKLMNOPQRSTUVWXYZ"),
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			c.UserService.CreateUser(ResourceUser{Username: tc.username})
			err := c.AuditUserService.Audit(ResourceAuditUser{
				UserName:    tc.username,
				AuditOption: tc.setAuditOption,
			})
			if err != nil {
				t.Fatalf("Failed set audit failed with error error: %v", err)
			}
			err = c.AuditUserService.NoAudit(ResourceAuditUser{
				UserName:    tc.username,
				AuditOption: tc.rmAuditOption,
			})
			result, _ := c.AuditUserService.ReadAudit(ResourceAuditUser{
				UserName: tc.username,
			})
			if len(result.AuditOption) > 0 && !reflect.DeepEqual(result.AuditOption, tc.wantauditOption) {
				t.Fatalf("audit option diff got: %v want: %v", result.AuditOption, tc.wantauditOption)
				c.UserService.DropUser(ResourceUser{Username: tc.username})
			} else if len(tc.wantauditOption) == 0 && len(result.AuditOption) > 0 {
				t.Fatalf("audit option diff got: %v want: %v", result.AuditOption, tc.wantauditOption)
				c.UserService.DropUser(ResourceUser{Username: tc.username})
			}
			c.UserService.DropUser(ResourceUser{Username: tc.username})
		})
	}
}
