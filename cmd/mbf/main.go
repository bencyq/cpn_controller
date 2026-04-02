package main

import (
	"cpn-controller/pkg/controller"
	"cpn-controller/pkg/mbf"
	"log"
	"time"
)

func main() {
	startTime := time.Now()

	controller.JsonUrl = "example3.json"
	controller.NAMESPACE = "mbf"
	monitor := controller.NewMonitor()
	monitor.ReadModelBaseline()
	mbf.MonitorAssignedJob(monitor)

	log.Println("INFO: Consumed time", time.Since(startTime).Minutes())
}
