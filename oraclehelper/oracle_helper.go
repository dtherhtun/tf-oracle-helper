package oraclehelper

import (
	"database/sql"
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
	port, _ := strconv.Atoi(cfg.DbPort)
	urlOptions := make(map[string]string)
	if cfg.SysDBA {
		urlOptions["auth as"] = "sysdba"
	}
	connStr := go_ora.BuildUrl(cfg.DbHost, port, cfg.DbService, cfg.Username, cfg.Password, urlOptions)
	db, err := sql.Open("oracle", connStr)
	var dBVersion string
	var conName string
	var conID uint

	err = db.Ping()
	if err != nil {
		log.Printf("[DEBUG] ping failed")
		return nil, err
	}
	err = db.QueryRow(queryDbVersion).Scan(&dBVersion)
	if err != nil {
		log.Fatalf("Query db version failed and return error: %v\n", err)
		return nil, err
	}
	err = db.QueryRow(queryConName).Scan(&conName)
	if err != nil {
		log.Fatalf("Query con name failed and return error: %v\n", err)
		return nil, err
	}
	err = db.QueryRow(queryConID).Scan(&conID)
	if err != nil {
		log.Fatalf("Query con id failed and return error: %v\n", err)
		return nil, err
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
	c.DBVersion, _ = version.NewVersion(dBVersion)
	c.ConName = conName
	if conID >= 1 {
		c.DBPluggable = true
	} else {
		c.DBPluggable = false
	}
	log.Printf("[DEBUG] dbversion: %v", c.DBVersion)

	return c, nil
}
