package main

import (
	"cpn-controller/pkg/affinity"
	"cpn-controller/pkg/controller"
	"log"
	"time"
)

func main() {
	startTime := time.Now()

	controller.JsonUrl = "example3.json"
	controller.NAMESPACE = "affinity"
	monitor := controller.NewMonitor()
	monitor.ReadModelBaseline()
	affinity.MonitorAssignedJob(monitor)

	log.Println("INFO: Consumed time", time.Since(startTime).Minutes())
}
