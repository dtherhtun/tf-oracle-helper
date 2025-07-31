package main

import (
	"fmt"
	"log"
	"os"

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
		SysDBA:    false,
	}

	client, err := oraclehelper.NewClient(cfg)
	if err != nil {
		log.Fatalf("Failed to create Oracle client: %v", err)
	}
	defer client.Close()

	username := "TESTUSER_" + acctest.RandStringFromCharSet(8, "ABCDEFGHIJKLMNOPQRSTUVWXYZ")
	user := oraclehelper.ResourceUser{Username: username}

	fmt.Printf("Creating user: %s\n", username)
	err = client.UserService.CreateUser(user)
	if err != nil {
		log.Fatalf("Failed to create user: %v", err)
	}
	fmt.Printf("User %s created successfully.\n", username)

	fmt.Printf("Dropping user: %s\n", username)
	err = client.UserService.DropUser(user)
	if err != nil {
		log.Fatalf("Failed to drop user: %v", err)
	}
	fmt.Printf("User %s dropped successfully.\n", username)
}
