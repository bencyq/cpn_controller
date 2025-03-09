package controller

import "testing"

func TestAssignJobWithinController(t *testing.T) {
	monitor := NewMonitor()
	testJob := monitor.JobPool.OriginJob[1]
	testJob.DataCenterIDX, testJob.ClusterIDX, testJob.NodeIDX, testJob.CardIDX = 0, 1, 1, 0
	monitor.AssignJobWithinController(testJob)
}

func TestAssignJobToNode(t *testing.T) {
	monitor := NewMonitor()
	job := monitor.JobPool.OriginJob[0]
	job.DataCenterIDX, job.ClusterIDX, job.NodeIDX, job.CardIDX = 0, 2, 0, 4
	monitor.AssignJobToNode(monitor.DataCenterInfo[0].ClusterInfo[2].ClusterClientSet, job, "aigpuserver", NAMESPACE)
}
