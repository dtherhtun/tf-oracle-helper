package oraclehelper

import (
	"testing"
)



func TestDBConnection(t *testing.T) {
	c, cleanup := setupTestClient(t)
	defer cleanup()

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

func TestNewClientErrorHandling(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Cfg
		wantErr bool
	}{
		{
			name: "Invalid Port",
			cfg: Cfg{
				DbHost:    "localhost",
				DbPort:    "invalid", // Invalid port
				Username:  "system",
				Password:  "MyPassword123",
				DbService: "orclpdb1",
			},
			wantErr: true,
		},
		{
			name: "Invalid Host (Ping Failure)",
			cfg: Cfg{
				DbHost:    "nonexistenthost", // Invalid host
				DbPort:    "1521",
				Username:  "system",
				Password:  "MyPassword123",
				DbService: "orclpdb1",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := NewClient(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil && c != nil {
				c.Close()
			}
		})
	}
}
