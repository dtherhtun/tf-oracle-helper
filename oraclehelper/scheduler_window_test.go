package oraclehelper

import (
	"testing"
)

func TestSchedulerWindowService(t *testing.T) {
	// This test can not be run against an container db
	if c.ConName == "CDB$ROOT" {
		return
	}
	tstWindow := ResourceSchedulerWindow{
		WindowName:     "TEST01",
		ResourcePlan:   "INTERNAL_PLAN",
		Duration:       "+000 01:00:00.000000",
		RepeatInterval: "freq=daily;byday=SAT;byhour=6;byminute=0; bysecond=0",
		WindowPriority: "LOW",
		Comments:       "test01 commments",
	}
	c.SchedulerWindowService.CreateSchedulerWindow(tstWindow)
	windows, err := c.SchedulerWindowService.ReadSchedulerWindow(ResourceSchedulerWindow{Owner: "SYS", WindowName: "TEST01"})
	if err != nil {
		t.Error("Failed to read window")
	}
	if windows.WindowName != tstWindow.WindowName {
		t.Error("window name not equal")

	}
	c.SchedulerWindowService.ModifySchedulerWindow(
		ResourceSchedulerWindow{
			Owner:      "SYS",
			WindowName: "TEST01",
			Duration:   "+000 00:30:00",
		})

	windows, err = c.SchedulerWindowService.ReadSchedulerWindow(ResourceSchedulerWindow{Owner: "SYS", WindowName: "TEST01"})
	if err != nil {
		t.Error("Failed to read window")
	}
	if windows.Duration != "+000 00:30:00" {
		t.Errorf("wanted: +000 00:30:00 got: %s\n", windows.Duration)
	}
	c.SchedulerWindowService.DropSchedulerWindow(ResourceSchedulerWindow{WindowName: "TEST01"})
}
