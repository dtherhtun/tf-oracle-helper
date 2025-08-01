package oraclehelper

import (
	"fmt"
	"log"
)

const (
	queryDatabase = `
SELECT
	name,
	flashback_on,
	force_logging
FROM v$database
`
)

type (
	// ResourceDatabase represents the Oracle database configuration.
	ResourceDatabase struct {
		Name         string
		ForceLogging string
		FlashBackOn  string
	}
	databaseService struct {
		client *Client
	}
)

// ReadDatabase reads the current database configuration.
func (d *databaseService) ReadDatabase() (*ResourceDatabase, error) {
	log.Printf("[DEBUG] ReadDatabase\n")
	ResourceDatabase := &ResourceDatabase{}
	err := d.client.DBClient.QueryRow(
		queryDatabase,
	).Scan(&ResourceDatabase.Name,
		&ResourceDatabase.FlashBackOn,
		&ResourceDatabase.ForceLogging,
	)
	if err != nil {
		return nil, err
	}
	return ResourceDatabase, nil
}

// ModifyDatabase modifies the database configuration.
func (d *databaseService) ModifyDatabase(tf ResourceDatabase) error {

	if tf.ForceLogging != "" {
		d.ModifyLoggingDatabase(tf)
	}
	if tf.FlashBackOn != "" {
		d.ModifyFlashbackDatabase(tf)
	}
	return nil
}

// ModifyLoggingDatabase modifies the database logging mode.
func (d *databaseService) ModifyLoggingDatabase(tf ResourceDatabase) error {
	sqlCommand := "alter database"

	if tf.ForceLogging == "YES" {
		sqlCommand += " force logging"
	} else if tf.ForceLogging == "NO" {
		sqlCommand += " no force logging"
	}
	log.Printf("[DEBUG] sqlcommand: %s", sqlCommand)

	_, err := d.client.DBClient.Exec(sqlCommand)
	if err != nil {
		return err
	}
	return nil
}

// ModifyFlashbackDatabase modifies the database flashback mode.
func (d *databaseService) ModifyFlashbackDatabase(tf ResourceDatabase) error {

	sqlCommand := fmt.Sprintf("alter database flashback %s", tf.FlashBackOn)
	log.Printf("[DEBUG] sqlcommand: %s", sqlCommand)

	_, err := d.client.DBClient.Exec(sqlCommand)
	if err != nil {
		return err
	}
	return nil
}
