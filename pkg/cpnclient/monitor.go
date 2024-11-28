package cpnclient

// 负责定时向server发送prometheus收集到的信息，以及namespace ”cpn-job“下的所有pod、job、deployment等的情况

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func CheckNamespace(client *kubernetes.Clientset) {
	_, err := client.CoreV1().Namespaces().Get(context.TODO(), "cpn-job", metav1.GetOptions{})
	if err != nil {
		namespaceClient := client.CoreV1().Namespaces()
		namespace := &apiv1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "cpn-job",
			},
		}
		result, err := namespaceClient.Create(context.TODO(), namespace, metav1.CreateOptions{})
		if err != nil {
			log.Println(err.Error())
		} else {
			log.Printf("Create namespace %s SuccessFul !", result.ObjectMeta.Name)
		}
	} else {
		log.Println("Namespace cpn-job exist")
	}
}

func podList(client *kubernetes.Clientset) (podlist *apiv1.PodList, err error) {
	podlist, err = client.CoreV1().Pods("cpn-job").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Println("Error: cannot list pods", err)
		return nil, err
	}
	return podlist, nil
}

func jobList(client *kubernetes.Clientset) (joblist *batchv1.JobList, err error) {
	joblist, err = client.BatchV1().Jobs("cpn-job").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Println("Error: cannot list jobs", err)
		return nil, err
	}
	return joblist, nil
}

func nodeList(client *kubernetes.Clientset) (nodelist *apiv1.NodeList, err error) {
	nodelist, err = client.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Println("Error: cannot list nodes", err)
		return nil, err
	}
	return nodelist, nil
}

func SendJson(msg []byte) (err error) {
	resp, err := http.Post(CpnServerURL, "application/json", bytes.NewBuffer(msg))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

// 负责整个monitor的启动逻辑
func APP2(client *kubernetes.Clientset) (err error) {
	log.Println("CpnServerURL:", CpnServerURL)
	// 定时发送
	for {
		var podMap, jobMap, nodeMap, mergedMap map[string]interface{}
		mergedMap = make(map[string]interface{})

		podlist, err := podList(client)
		if err == nil {
			podData, _ := json.Marshal(podlist)
			_ = json.Unmarshal(podData, &podMap)
		}

		joblist, err := jobList(client)
		if err == nil {
			jobData, _ := json.Marshal(joblist)
			_ = json.Unmarshal(jobData, &jobMap)
		}

		nodelist, err := nodeList(client)
		if err == nil {
			nodeData, _ := json.Marshal(nodelist)
			_ = json.Unmarshal(nodeData, &nodeMap)
		}

		mergedMap["client-name"] = ClientName
		mergedMap["pod"] = podMap
		mergedMap["job"] = jobMap
		mergedMap["node"] = nodeMap

		// TODO:加入prometheus的监控信息

		msg, err := json.Marshal(mergedMap)
		if err != nil {
			log.Println("Error: merging information failed", err)
		}

		err = SendJson(msg)
		if err != nil {
			log.Print("Error: monitor failed sending message", err)
		} else {
			log.Print("monitor messgage sent")
		}

		time.Sleep(TimeInterval)
	}
}
