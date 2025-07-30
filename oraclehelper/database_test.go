package oraclehelper

import (
	"log"
	"testing"
)

func TestDatabaseRead(t *testing.T) {

	db, err := c.DatabaseService.ReadDatabase()
	if err != nil {
		log.Fatalf("failed to read database, errormsg: %v\n", err)
	}

	if db.Name == "" {
		t.Errorf("Database name should not be nil\n")
	}
}

func TestDatabaseModify(t *testing.T) {
	var logMode string
	c.DBClient.QueryRow("SELECT log_mode FROM v$database").Scan(&logMode)
	if logMode == "NOARCHIVELOG" {
		return
	}
	db, err := c.DatabaseService.ReadDatabase()
	if err != nil {
		log.Fatalf("failed to read database, errormsg: %v\n", err)
	}

	if db.ForceLogging == "NO" {
		c.DatabaseService.ModifyLoggingDatabase(ResourceDatabase{ForceLogging: "YES"})
		db, _ = c.DatabaseService.ReadDatabase()
		if db.ForceLogging != "YES" {
			t.Errorf("wanted: %s, gott: %s", "YES", db.ForceLogging)
		}
	} else if db.ForceLogging == "YES" {
		c.DatabaseService.ModifyLoggingDatabase(ResourceDatabase{ForceLogging: "NO"})
		db, _ = c.DatabaseService.ReadDatabase()
		if db.ForceLogging != "NO" {
			t.Errorf("wanted: %s, gott: %s", "NO", db.ForceLogging)
		}
	}
	if db.FlashBackOn == "NO" {
		c.DatabaseService.ModifyFlashbackDatabase(ResourceDatabase{FlashBackOn: "ON"})
		db, _ = c.DatabaseService.ReadDatabase()
		if db.FlashBackOn != "YES" {
			t.Errorf("wanted: %s, gott: %s", "YES", db.FlashBackOn)
		}
	} else if db.FlashBackOn == "YES" {
		c.DatabaseService.ModifyFlashbackDatabase(ResourceDatabase{FlashBackOn: "OFF"})
		db, _ = c.DatabaseService.ReadDatabase()
		if db.FlashBackOn != "NO" {
			t.Errorf("wanted: %s, gott: %s", "NO", db.FlashBackOn)
		}
	}
}
