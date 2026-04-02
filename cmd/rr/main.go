package main

import (
	"cpn-controller/pkg/controller"
	"cpn-controller/pkg/rr"
	"log"
	"time"
)

func main() {
	startTime := time.Now()

	controller.JsonUrl = "example3.json"
	controller.NAMESPACE = rr.NAMESPACE
	monitor := controller.NewMonitor()
	monitor.ReadModelBaseline()
	rr.MonitorAssignedJob(monitor)

	log.Println("INFO: Consumed time", time.Since(startTime).Minutes())
}
