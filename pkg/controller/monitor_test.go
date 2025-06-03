package controller

import (
	"fmt"
	"testing"
)

func TestUnmarshalJson(t *testing.T) {
	var monitor Monitor
	monitor.unmarshalJson(getJsonWithFile("example.json"))
}

func TestGetMetric(t *testing.T) {
	var monitor Monitor
	// 初始化monitor
	monitor.unmarshalJson(getJsonWithFile("example2.json"))
	monitor.GetMetric(1)
	fmt.Println(monitor)
}

func TestGetJob(t *testing.T) {
	var monitor Monitor
	monitor.getJob()
}

func TestNewMonitor(t *testing.T) {
	NewMonitor()
}

func TestGetImagefs(t *testing.T) {
	JsonUrl = "example2.json"
	m := NewMonitor()
	GetImagefs(m.DataCenterInfo[0].ClusterInfo[0].ClusterClientSet, "node191")
}
