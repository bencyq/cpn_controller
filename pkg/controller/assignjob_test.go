package controller

import "testing"

func TestAssignJobWithinController(t *testing.T) {
	monitor := NewMonitor()
	testJob := monitor.JobPool.OriginJob[0]
	testJob.DataCenterIDX, testJob.ClusterIDX, testJob.NodeIDX, testJob.CardIDX = 0, 2, 0, 4
	monitor.AssignJobWithinController(testJob)
}
