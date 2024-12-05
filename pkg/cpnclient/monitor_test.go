package cpnclient

import (
	"fmt"
	"log"
	"testing"
)

func TestAPP2(t *testing.T) {
	client, err := NewClientSetOutOfCluster()
	if err != nil {
		log.Println(err)
	}
	APP2(client)
}

func TestGetPrometheusSvcIPAndPort(t *testing.T) {
	client, err := NewClientSetOutOfCluster()
	if err != nil {
		log.Println(err)
	}
	fmt.Println(GetPrometheusSvcIPAndPort(client))
}

func TestGetMertric(t *testing.T) {
	for _, nodeName := range WorkerNodeName {
		for _, metric := range PromMetrics[nodeName] {
			fmt.Println(GetMertric("10.98.211.221:80", metric))
		}
	}
}
