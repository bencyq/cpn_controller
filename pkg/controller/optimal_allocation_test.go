package controller

import "testing"

func TestAnalyze(t *testing.T) {
	monitor := NewMonitor()
	for _, job := range monitor.JobPool.OriginJob {
		monitor.OptimalAllocate(job)
	}
}
