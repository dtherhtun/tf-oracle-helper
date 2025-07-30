package oraclehelper

import (
	"log"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/acctest"
)

var (
	autoTasks = [3]string{"sql tuning advisor", "auto optimizer stats collection", "auto space advisor"}
)

func TestAutoTaskRead(t *testing.T) {
	resourceAutoTask := ResourceAutoTask{ClientName: autoTasks[acctest.RandIntRange(0, 2)]}

	autoTask, err := c.AutoTaskService.ReadAutoTask(resourceAutoTask)
	if err != nil {
		log.Fatalf("failed to read autotask, errormsg: %v\n", err)
	}

	if autoTask.ClientName != resourceAutoTask.ClientName {
		t.Errorf("Wanted: %s gott: %s", resourceAutoTask.ClientName, autoTask.ClientName)
	}
}

func TestAutoTaskEnableDisable(t *testing.T) {

	resourceAutoTask := ResourceAutoTask{ClientName: autoTasks[acctest.RandIntRange(0, 2)]}

	autoTask, err := c.AutoTaskService.ReadAutoTask(resourceAutoTask)
	if err != nil {
		log.Fatalf("failed to read autotask, errormsg: %v\n", err)
	}

	if autoTask.Status == "ENABLED" {
		err = c.AutoTaskService.DisableAutoTask(resourceAutoTask)
		if err != nil {
			log.Fatalf("failed to Disable autotask, errormsg: %v\n", err)
		}
		autoTask, _ = c.AutoTaskService.ReadAutoTask(resourceAutoTask)
		if autoTask.Status != "DISABLED" {
			t.Errorf("Test Enabled autotask Wanted: %s gott: %s", "DISABLED", autoTask.Status)
		}
	} else {
		err = c.AutoTaskService.EnableAutoTask(resourceAutoTask)
		if err != nil {
			log.Fatalf("failed to Enable autotask, errormsg: %v\n", err)
		}
		autoTask, _ = c.AutoTaskService.ReadAutoTask(resourceAutoTask)
		if autoTask.Status != "ENABLED" {
			t.Errorf("Test Enabled autotask Wanted: %s gott: %s", "ENABLED", autoTask.Status)
		}
	}

}
