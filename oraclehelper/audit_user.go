package oraclehelper

import (
	"fmt"
	"log"
)

const (
	queryUserAudit = `
SELECT
    sao.user_name,
    sao.proxy_name,
    sao.audit_option,
    sao.success,
    sao.failure
FROM DBA_STMT_AUDIT_OPTS sao
WHERE sao.user_name = UPPER(:1)
UNION
SELECT
    pao.user_name,
    pao.proxy_name,
    pao.privilege,
    pao.success,
    pao.failure
FROM DBA_PRIV_AUDIT_OPTS pao
WHERE pao.user_name = UPPER(:1)
`
)

type (
	auditUserService struct {
		client *Client
	}
	//AuditOption ..
	AuditOption struct {
		Option  string
		Success string
		Failure string
	}
	//ResourceAuditUser ..
	ResourceAuditUser struct {
		UserName    string
		AuditOption []AuditOption
		Proxy       string
	}
)

const (
	auditBySucess = "BY ACCESS"
	auditNotSet   = "NOT SET"
)

func (a *auditUserService) Audit(r ResourceAuditUser) error {
	log.Println("[DEBUG] Audit")

	for _, v := range r.AuditOption {
		sqlCommand := fmt.Sprintf("audit %s by %s", v.Option, r.UserName)
		if v.Success == auditBySucess && v.Failure == auditNotSet {
			sqlCommand += " by access whenever successful"
		} else if v.Success == auditNotSet && v.Failure == auditBySucess {
			sqlCommand += " by access whenever not successful"
		} else if v.Success == auditBySucess && v.Failure == auditBySucess {
			sqlCommand += " by access"
		}
		log.Printf("[DEBUG] sqlcommand: %s", sqlCommand)
		_, err := a.client.DBClient.Exec(sqlCommand)
		if err != nil {
			return err
		}
	}
	return nil
}

func (a *auditUserService) NoAudit(r ResourceAuditUser) error {

	log.Println("[DEBUG] Audit")

	for _, v := range r.AuditOption {
		sqlCommand := fmt.Sprintf("noaudit %s by %s", v.Option, r.UserName)
		log.Printf("[DEBUG] sqlcommand: %s", sqlCommand)
		_, err := a.client.DBClient.Exec(sqlCommand)
		if err != nil {
			return err
		}
	}
	return nil
}

func (a *auditUserService) ReadAudit(r ResourceAuditUser) (ResourceAuditUser, error) {
	log.Printf("[DEBUG] Running ReadAudit function")
	var auditOption []AuditOption
	rows, err := a.client.DBClient.Query(queryUserAudit, r.UserName)
	if err != nil {
		return ResourceAuditUser{}, err
	}
	defer rows.Close()
	cols, _ := rows.Columns()
	for rows.Next() {
		log.Printf("[DEBUG] Start Featching rows")
		columns := make([]interface{}, len(cols))
		columnPointers := make([]interface{}, len(cols))
		for i := range columns {
			columnPointers[i] = &columns[i]
		}
		// Scan the result into the column pointers...
		if err := rows.Scan(columnPointers...); err != nil {
			return ResourceAuditUser{}, err
		}

		m := make(map[string]interface{})
		for i, colName := range cols {
			val := columnPointers[i].(*interface{})
			m[colName] = *val
		}
		var a AuditOption

		if val, ok := m["AUDIT_OPTION"].(string); ok {
			a.Option = val
		}
		if val, ok := m["SUCCESS"].(string); ok {
			a.Success = val
		}
		if val, ok := m["FAILURE"].(string); ok {
			a.Failure = val
		}
		auditOption = append(auditOption, a)
	}

	return ResourceAuditUser{
		AuditOption: auditOption,
	}, nil
}
