package oraclehelper

import (
	"fmt"
	"log"
)

const (
	queryRole = `
SELECT
	r.role,
	r.password_required,
	r.authentication_type,
	r.common,
	r.oracle_maintained
FROM
	dba_roles r
WHERE r.role = UPPER(:1)
`
)

// Role represents an Oracle database role.
type (
	// ResourceRole represents a role resource.
	ResourceRole struct {
		Role string
	}
	// Role represents the detailed information of an Oracle role.
	Role struct {
		Role               string
		PasswordRequired   string
		AuthenticationType string
		Common             string
		OracleMaintained   string
	}
	roleService struct {
		client *Client
	}
)

// ReadRole reads the details of an Oracle database role.
func (r *roleService) ReadRole(tf ResourceRole) (*Role, error) {
	log.Printf("[DEBUG] ReadUser username: %s\n", tf.Role)
	roleType := &Role{}

	err := r.client.DBClient.QueryRow(queryRole, tf.Role).Scan(&roleType.Role,
		&roleType.PasswordRequired,
		&roleType.AuthenticationType,
		&roleType.Common,
		&roleType.OracleMaintained,
	)
	if err != nil {
		return nil, err
	}

	return roleType, nil
}

// CreateRole creates a new Oracle database role.
func (r *roleService) CreateRole(tf ResourceRole) error {
	log.Println("[DEBUG] CreateRole")
	sqlCommand := fmt.Sprintf("create role %s", tf.Role)

	log.Printf("[DEBUG] sqlcommand: %s", sqlCommand)

	_, err := r.client.DBClient.Exec(sqlCommand)
	if err != nil {
		return err
	}

	return nil
}

// ModifyRole modifies an Oracle database role. (Note: This function currently does nothing beyond logging and executing a generic ALTER USER command, which is likely incorrect for roles.)
func (r *roleService) ModifyRole(tf ResourceRole) error {
	log.Printf("[DEBUG] ModifyRole: No modifiable attributes provided for role %s. Returning nil.", tf.Role)
	return nil
}

// DropRole drops an Oracle database role.
func (r *roleService) DropRole(tf ResourceRole) error {
	log.Println("[DEBUG] DropRole")
	sqlCommand := fmt.Sprintf("drop role %s", tf.Role)
	log.Printf("[DEBUG] sqlcommand: %s", sqlCommand)

	_, err := r.client.DBClient.Exec(sqlCommand)
	if err != nil {
		log.Printf("[DEBUG] drop role err: %s\n", err)
		return err
	}

	return nil
}
