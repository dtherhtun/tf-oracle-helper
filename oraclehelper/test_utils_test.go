package oraclehelper

import (
	"testing"
)

func setupTestClient(t *testing.T) (*Client, func()) {
	cfg := Cfg{
		DbHost:    "localhost",
		DbPort:    "1521",
		Username:  "system",
		Password:  "MyPassword123",
		DbService: "orclpdb1",
	}
	c, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	return c, func() {
		c.Close()
	}
}
