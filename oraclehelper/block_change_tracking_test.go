package oraclehelper

import (
	"fmt"
	"log"
	"testing"
)

func TestBlackChangeTracking(t *testing.T) {
	if c.DBPluggable {
		return
	}
	var err error
	blockChangTracking, err := c.BlockChangeTrackingService.ReadBlockChangeTracking()
	if err != nil {
		log.Fatalf("failed to read bloch change tracking, errormsg: %v\n", err)
	}
	fmt.Printf("v: %v", blockChangTracking)
	if blockChangTracking.Status == "ENABLED" {
		err = c.BlockChangeTrackingService.DisableBlockChangeTracking()
	} else {
		err = c.BlockChangeTrackingService.EnableBlockChangeTracking(ResourceBlockChangeTracking{FileName: "change_tracking_file"})
	}
	if err != nil {
		t.Errorf("Enable/Disabled failed with: %v", err)
	}

}
