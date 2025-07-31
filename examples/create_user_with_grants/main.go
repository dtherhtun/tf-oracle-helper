package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/helper/acctest"

	"github.com/dtherhtun/tf-oracle-helper/oraclehelper"
)

func main() {
	cfg := oraclehelper.Cfg{
		Username:  os.Getenv("ORACLE_USERNAME"),
		Password:  os.Getenv("ORACLE_PASSWORD"),
		DbHost:    os.Getenv("ORACLE_DB_HOST"),
		DbPort:    os.Getenv("ORACLE_DB_PORT"),
		DbService: os.Getenv("ORACLE_DB_SERVICE"),
		SysDBA:    true, // This example requires SYSDBA for creating users and tables
	}

	client, err := oraclehelper.NewClient(cfg)
	if err != nil {
		log.Fatalf("Failed to create Oracle client: %v", err)
	}
	defer client.Close()

	// Generate a random username
	username := "EX_USER_" + acctest.RandStringFromCharSet(8, "ABCDEFGHIJKLMNOPQRSTUVWXYZ")
	user := oraclehelper.ResourceUser{Username: username}

	// Define a dummy table for object privileges
	dummyTableName := "DUMMY_TABLE_" + acctest.RandStringFromCharSet(5, "ABCDEFGHIJKLMNOPQRSTUVWXYZ")

	fmt.Printf("Creating user: %s\n", username)
	err = client.UserService.CreateUser(user)
	if err != nil {
		log.Fatalf("Failed to create user %s: %v", username, err)
	}
	fmt.Printf("User %s created successfully.\n", username)

	// Create a dummy table for object privileges
	fmt.Printf("Creating dummy table: %s.DUMMY_TABLE\n", os.Getenv("ORACLE_USERNAME"))
	_, err = client.DBClient.Exec(fmt.Sprintf("CREATE TABLE %s.%s (id NUMBER)", os.Getenv("ORACLE_USERNAME"), dummyTableName))
	if err != nil {
		log.Fatalf("Failed to create dummy table: %v", err)
	}
	fmt.Printf("Dummy table %s.DUMMY_TABLE created successfully.\n", os.Getenv("ORACLE_USERNAME"))

	// Grant system privileges
	sysPrivs := []string{"CREATE SESSION", "SELECT ANY TABLE"}
	for _, priv := range sysPrivs {
		fmt.Printf("Granting system privilege '%s' to user %s\n", priv, username)
		err = client.GrantService.GrantSysPriv(oraclehelper.ResourceGrantSystemPrivilege{
			Grantee:   username,
			Privilege: priv,
		})
		if err != nil {
			log.Fatalf("Failed to grant system privilege %s to %s: %v", priv, username, err)
		}
		fmt.Printf("System privilege '%s' granted to user %s.\n", priv, username)
	}

	// Grant object privileges
	objectPrivs := []string{"UPDATE", "INSERT", "DELETE"}
	objectName := fmt.Sprintf("%s.%s", strings.ToUpper(os.Getenv("ORACLE_USERNAME")), dummyTableName)
	fmt.Printf("Granting object privileges %v on %s to user %s\n", objectPrivs, objectName, username)
	err = client.GrantService.GrantObjectPrivilege(oraclehelper.ResourceGrantObjectPrivilege{
		Grantee:    username,
		Privilege:  objectPrivs,
		Owner:      strings.ToUpper(os.Getenv("ORACLE_USERNAME")),
		ObjectName: dummyTableName,
	})
	if err != nil {
		log.Fatalf("Failed to grant object privileges on %s to %s: %v", objectName, username, err)
	}
	fmt.Printf("Object privileges granted on %s to user %s.\n", objectName, username)

	fmt.Printf("Cleaning up: Dropping user %s and table %s.DUMMY_TABLE\n", username, os.Getenv("ORACLE_USERNAME"))
	// Revoke object privileges before dropping the table (optional, but good practice)
	_ = client.GrantService.RevokeObjectPrivilege(oraclehelper.ResourceGrantObjectPrivilege{
		Grantee:    username,
		Privilege:  objectPrivs,
		Owner:      strings.ToUpper(os.Getenv("ORACLE_USERNAME")),
		ObjectName: dummyTableName,
	})

	// Drop the dummy table
	_, err = client.DBClient.Exec(fmt.Sprintf("DROP TABLE %s.%s", os.Getenv("ORACLE_USERNAME"), dummyTableName))
	if err != nil {
		log.Printf("Warning: Failed to drop dummy table %s.DUMMY_TABLE: %v", os.Getenv("ORACLE_USERNAME"), err)
	}

	// Drop the user
	err = client.UserService.DropUser(user)
	if err != nil {
		log.Fatalf("Failed to drop user %s: %v", username, err)
	}
	fmt.Printf("User %s and table %s.DUMMY_TABLE cleaned up successfully.\n", username, os.Getenv("ORACLE_USERNAME"))
}
