package controller

import (
	"fmt"
	"testing"
)

func TestUnmarshalJson(t *testing.T) {
	var monitor Monitor
	monitor.unmarshalJson(getJsonWithFile("example.json"))
}

func TestUnmarshalJsonLoadsUUID(t *testing.T) {
	var monitor Monitor
	monitor.unmarshalJson(getJsonWithFile("example2.json"))

	node200Card0 := monitor.DataCenterInfo[0].ClusterInfo[0].NodeInfo[1].CardInfo[0]
	if node200Card0.UUID != "GPU-fa5532d9-11d7-428d-6da3-0beb06a4c9f6" {
		t.Fatalf("unexpected node200 card0 uuid: %q", node200Card0.UUID)
	}

	node191Card5 := monitor.DataCenterInfo[0].ClusterInfo[0].NodeInfo[2].CardInfo[5]
	if node191Card5.UUID != "GPU-1b2ac972-5ab9-845c-1951-3589fe7380d0" {
		t.Fatalf("unexpected node191 card5 uuid: %q", node191Card5.UUID)
	}
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
