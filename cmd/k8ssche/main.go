package main

import (
	"cpn-controller/pkg/controller"
	"cpn-controller/pkg/k8ssche"
	"log"
	"time"
)

func main() {
	startTime := time.Now()

	controller.JsonUrl = "example3.json"
	controller.NAMESPACE = "k8s-sche"
	monitor := controller.NewMonitor()
	monitor.ReadModelBaseline()
	k8ssche.MonitorAssignedJob(monitor)

	log.Println("INFO: Consumed time", time.Since(startTime).Minutes())
}
