package main

import (
	"fmt"
	"log"
	"os"

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

	param, err := client.ParameterService.Read(oraclehelper.ResourceParameter{Name: "undo_retention"})
	if err != nil {
		log.Fatalf("Failed to read parameter: %v", err)
	}
	fmt.Printf("Undo Retention: %s\n", param.Value)
}
