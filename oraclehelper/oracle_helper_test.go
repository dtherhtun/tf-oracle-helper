package oraclehelper

import (
	"log"
	"testing"
)

var (
	cfg = Cfg{}

	c = &Client{}
)

func init() {
	cfg.DbHost = "localhost"
	cfg.DbPort = "1521"
	cfg.Username = "system"
	cfg.Password = "MyPassword123"
	cfg.DbService = "orclpdb1"
	var err error
	c, err = NewClient(cfg)
	if err != nil {
		log.Fatalf("failed to parse config: %v\n", err)
	}
}
func TestDBConnection(t *testing.T) {
	var got string
	want := "foo"

	rows, err := c.DBClient.Query("select 'foo' as foo from dual")
	if err != nil {
		t.Errorf("error: %g", err)
	}
	defer rows.Close()
	for rows.Next() {
		rows.Scan(&got)
	}
	if want != got {
		t.Errorf("%v; want %v\n", got, want)
	}

}
