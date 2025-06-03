package main

import (
	"cpn-controller/pkg/controller"
	"cpn-controller/pkg/fifo"
	"log"
	"time"
)

func main() {
	startTime := time.Now()
	controller.JsonUrl = "example2.json"
	controller.NAMESPACE = fifo.NAMESPACE
	monitor := controller.NewMonitor()
	monitor.ReadModelBaseline()
	fifo.MonitorAssignedJob(monitor)
	log.Println("INFO: Consumed time", time.Since(startTime).Minutes())
}
