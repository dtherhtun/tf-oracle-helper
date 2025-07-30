package oraclehelper

import (
	"fmt"
	"log"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/acctest"
)

var (
	granularity = []string{"ALL", "AUTO", "DEFAULT", "PARTITION", "GLOBAL", "GLOBAL AND PARTITION", "SUBPARTITION"}
)

func TestStatsServiceTable(t *testing.T) {
	tableName := acctest.RandStringFromCharSet(10, "ABCDEFGHIJKLMNOPQRSTUVWXYZ")
	c.DBClient.Exec(fmt.Sprintf("drop table system.%s", tableName))
	c.DBClient.Exec(fmt.Sprintf("create table system.%s(col number)", tableName))

	v := granularity[acctest.RandIntRange(0, 6)]
	globalGranularity := granularity[acctest.RandIntRange(0, 6)]
	for globalGranularity == v {
		globalGranularity = granularity[acctest.RandIntRange(0, 6)]
	}
	t.Logf("got granularity: %s for table_name: %s and global: %s\n", v, tableName, globalGranularity)

	c.StatsService.SetGlobalPre(ResourceStats{
		Pname: "GRANULARITY",
		Pvalu: globalGranularity,
	})
	resourceStats := ResourceStats{
		Pname:   "GRANULARITY",
		OwnName: "SYSTEM",
		TaBName: tableName,
		Pvalu:   v,
	}
	err := c.StatsService.SetTabPre(resourceStats)

	if err != nil {
		log.Fatalf("failed to stats, errormsg: %v\n", err)
	}

	tableGranularity, err := c.StatsService.ReadTabPref(resourceStats)
	if err != nil {
		log.Fatalf("failed to stats, errormsg: %v\n", err)
	}

	if resourceStats.Pvalu != tableGranularity.Pvalu {
		t.Errorf("got %s; want %s\n", tableGranularity.Pvalu, resourceStats.Pvalu)
	}
	c.DBClient.Exec(fmt.Sprintf("drop table system.%s", tableName))
}

func TestStatsServiceGlobal(t *testing.T) {

	v := granularity[acctest.RandIntRange(0, 6)]

	resourceStats := ResourceStats{
		Pname: "GRANULARITY",
		Pvalu: v,
	}
	err := c.StatsService.SetGlobalPre(resourceStats)
	if err != nil {
		log.Fatalf("failed to stats, errormsg: %v\n", err)
	}

	globalGranularity, err := c.StatsService.ReadGlobalPre(resourceStats)
	if err != nil {
		log.Fatalf("failed to stats, errormsg: %v\n", err)
	}

	if resourceStats.Pvalu != globalGranularity.Pvalu {
		t.Errorf("got %s; want %s\n", globalGranularity.Pvalu, resourceStats.Pvalu)
	}

}
