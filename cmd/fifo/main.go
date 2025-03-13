package main

import (
	"cpn-controller/pkg/controller"
	"cpn-controller/pkg/fifo"
	"log"
	"time"
)

func main() {
	startTime := time.Now()
	controller.NAMESPACE = "fifo"
	monitor := controller.NewMonitor()
	fifo.FifoSchedule(monitor)
	fifo.MonitorAssignedJob(monitor)
	log.Println("INFO: Consumed time", time.Since(startTime).Minutes())
}
