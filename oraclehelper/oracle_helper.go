package oraclehelper

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"

	"github.com/hashicorp/go-version"
	"github.com/sijms/go-ora/v2"
)

type (
	Cfg struct {
		Username  string
		Password  string
		DbHost    string
		DbPort    string
		DbService string
		SysDBA    bool
	}

	Client struct {
		DBClient                   *sql.DB
		DBVersion                  *version.Version
		DBPluggable                bool
		ConName                    string
		ParameterService           *parameterService
		ProfileService             *profileService
		UserService                *userService
		RoleService                *roleService
		GrantService               *grantService
		StatsService               *statsService
		SchedulerWindowService     *schedulerWindowService
		AutoTaskService            *autoTaskService
		DatabaseService            *databaseService
		BlockChangeTrackingService *blockChangeTrackingService
		AuditUserService           *auditUserService
	}
)

func (c *Client) Close() error {
	return c.DBClient.Close()
}

const (
	queryDbVersion = `
SELECT
	version
FROM v$instance
`
	queryConName = `
SELECT
	SYS_CONTEXT('USERENV', 'CON_NAME') AS CON_NAME
FROM   dual
`
	queryConID = `
SELECT
	SYS_CONTEXT('USERENV', 'CON_ID') AS CON_ID
FROM   dual
`
)

func NewClient(cfg Cfg) (*Client, error) {
	port, err := strconv.Atoi(cfg.DbPort)
	if err != nil {
		return nil, fmt.Errorf("invalid DbPort: %w", err)
	}
	urlOptions := make(map[string]string)
	if cfg.SysDBA {
		urlOptions["auth as"] = "sysdba"
	}
	connStr := go_ora.BuildUrl(cfg.DbHost, port, cfg.DbService, cfg.Username, cfg.Password, urlOptions)
	db, err := sql.Open("oracle", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	var dBVersion string
	var conName string
	var conID uint

	err = db.Ping()
	if err != nil {
		log.Printf("[DEBUG] ping failed: %v", err)
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}
	err = db.QueryRow(queryDbVersion).Scan(&dBVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to query database version: %w", err)
	}
	err = db.QueryRow(queryConName).Scan(&conName)
	if err != nil {
		return nil, fmt.Errorf("failed to query container name: %w", err)
	}
	err = db.QueryRow(queryConID).Scan(&conID)
	if err != nil {
		return nil, fmt.Errorf("failed to query container ID: %w", err)
	}

	c := &Client{DBClient: db}
	c.BlockChangeTrackingService = &blockChangeTrackingService{client: c}
	c.ParameterService = &parameterService{client: c}
	c.ProfileService = &profileService{client: c}
	c.UserService = &userService{client: c}
	c.RoleService = &roleService{client: c}
	c.GrantService = &grantService{client: c}
	c.StatsService = &statsService{client: c}
	c.SchedulerWindowService = &schedulerWindowService{client: c}
	c.AutoTaskService = &autoTaskService{client: c}
	c.DatabaseService = &databaseService{client: c}
	c.AuditUserService = &auditUserService{client: c}

	dbVersionParsed, err := version.NewVersion(dBVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database version '%s': %w", dBVersion, err)
	}
	c.DBVersion = dbVersionParsed
	c.ConName = conName
	c.DBPluggable = conID >= 1
	log.Printf("[DEBUG] dbversion: %v", c.DBVersion)

	return c, nil
}
