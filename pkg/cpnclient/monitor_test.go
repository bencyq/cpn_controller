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
	for _, metric := range PromMetrics {
		fmt.Println(GetMertric("10.98.211.221:80", metric))
	}
	// fmt.Println(GetMertric("10.98.211.221:80", `sum%28increase%28node_cpu_seconds_total%7Bmode%21%3D%22idle%22%2Cnode%3D%22node16%22%7D%5B2m%5D%29%29`))
}
