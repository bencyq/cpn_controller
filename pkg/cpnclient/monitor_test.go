package cpnclient

import (
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
