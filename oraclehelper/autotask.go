package oraclehelper

import (
	"log"
)

const (
	queryAutoTask = `
SELECT 
	client_name,
	status 
FROM dba_autotask_client
WHERE client_name = :1
`
	execDisableAutoTask = `
BEGIN
	dbms_auto_task_admin.disable(
		client_name => :1,
		operation   => NULL,
		window_name => NULL);
END;
`
	execEnableAutoTask = `
BEGIN
	dbms_auto_task_admin.enable(
	client_name => :1,
	operation   => NULL,
	window_name => NULL);
END;
`
)

type (
	// ResourceAutoTask represents an Oracle autotask client.
	ResourceAutoTask struct {
		ClientName string
		Status     string
	}
	autoTaskService struct {
		client *Client
	}
)

// DisableAutoTask disables an Oracle autotask client.
func (a *autoTaskService) DisableAutoTask(tf ResourceAutoTask) error {
	log.Printf("[DEBUG] DisableAutoTask clientName: %s\n", tf.ClientName)
	_, err := a.client.DBClient.Exec(execDisableAutoTask, tf.ClientName)
	if err != nil {
		return err
	}
	return nil
}

// EnableAutoTask enables an Oracle autotask client.
func (a *autoTaskService) EnableAutoTask(tf ResourceAutoTask) error {
	log.Printf("[DEBUG] EnableAutoTask clientName: %s\n", tf.ClientName)
	_, err := a.client.DBClient.Exec(execEnableAutoTask, tf.ClientName)
	if err != nil {
		return err
	}
	return nil
}

// ReadAutoTask reads the status of an Oracle autotask client.
func (a *autoTaskService) ReadAutoTask(tf ResourceAutoTask) (*ResourceAutoTask, error) {
	log.Printf("[DEBUG] ReadAutoTask clientName: %s\n", tf.ClientName)
	resourceAutoTask := &ResourceAutoTask{}
	err := a.client.DBClient.QueryRow(
		queryAutoTask,
		tf.ClientName,
	).Scan(&resourceAutoTask.ClientName,
		&resourceAutoTask.Status,
	)
	if err != nil {
		return nil,
			err
	}
	return resourceAutoTask, nil
}
