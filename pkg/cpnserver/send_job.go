package cpnserver

import (
	"bytes"
	"log"
	"net/http"
	"os"
	"time"

	"sigs.k8s.io/yaml"
)

func sendHttp(msg []byte, clientIp string) (err error) {
	resp, err := http.Post(clientIp, "application/x-yaml", bytes.NewBuffer(msg))
	if err != nil {
		log.Printf("Error: failed sending http to Client %v, %v\n", clientIp, err)
		return err
	}
	defer resp.Body.Close()
	return nil
}

func SendJob(file_path string, clientIp string) (err error) {
	msg, err := os.ReadFile(file_path)
	if err != nil {
		log.Println("Error: reading yaml file:", err)
		return err
	}

	// 加入CpnJobID
	var obj map[string]interface{}
	err = yaml.Unmarshal(msg, &obj)
	if err != nil {
		log.Println("Error: Unmarshal yaml failed", err)
		return err
	}
	obj["CpnJobID"] = time.Now().Format("200601021504")
	msg, _ = yaml.Marshal(obj)
	sendHttp(msg, clientIp)
	return nil
}
