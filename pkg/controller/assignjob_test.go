package controller

import (
	"testing"
)

func TestAssignJobWithinController(t *testing.T) {
	monitor := NewMonitor()
	testJob := monitor.JobPool.OriginJob[1]
	testJob.DataCenterIDX, testJob.ClusterIDX, testJob.NodeIDX, testJob.CardIDX = 0, 1, 1, 0
	monitor.AssignJobWithinController(testJob)
}

func TestAssignJobToNode(t *testing.T) {
	JsonUrl = "example2.json"
	monitor := NewMonitor()
	job := monitor.JobPool.OriginJob[2]
	job.DataCenterIDX, job.ClusterIDX, job.NodeIDX, job.CardIDX = 0, 0, 3, 4
	monitor.AssignJobToNode(monitor.DataCenterInfo[0].ClusterInfo[0].ClusterClientSet, job, "node191", NAMESPACE)
}

func TestDeleteJobFromNode(t *testing.T) {
	NAMESPACE = `fifo`
	monitor := NewMonitor()
	job := monitor.JobPool.OriginJob[6]
	monitor.DeleteJobFromNode(monitor.DataCenterInfo[0].ClusterInfo[0].ClusterClientSet, job, NAMESPACE)
}
