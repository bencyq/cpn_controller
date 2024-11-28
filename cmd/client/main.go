package main

import (
	// "context"
	"cpn-controller/pkg/cpnclient"
	"log"
	// apiv1 "k8s.io/api/core/v1"
	// metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func main() {
	// 创建客户端
	log.Println("Start cpn client...")
	client, err := cpnclient.NewClientSetOutOfCluster()
	if err != nil {
		log.Println(err)
	}

	// 保证集群内部存在Namespace cpn-job
	cpnclient.CheckNamespace(client)
	// _, err = client.CoreV1().Namespaces().Get(context.TODO(), "cpn-job", metav1.GetOptions{})
	// if err != nil {
	// 	namespaceClient := client.CoreV1().Namespaces()
	// 	namespace := &apiv1.Namespace{
	// 		ObjectMeta: metav1.ObjectMeta{
	// 			Name: "cpn-job",
	// 		},
	// 	}
	// 	result, err := namespaceClient.Create(context.TODO(), namespace, metav1.CreateOptions{})
	// 	if err != nil {
	// 		log.Println(err.Error())
	// 	} else {
	// 		log.Printf("Create namespace %s SuccessFul !", result.ObjectMeta.Name)
	// 	}
	// } else {
	// 	log.Println("Namespace cpn-job exist")
	// }

	// 启动APP1：create_job里的内容
	cpnclient.APP1()
	// go func() {
	// 	// 创建http服务端
	// 	cpnclient.NewHttpClient()
	// 	// 创建yaml文件的存储目录
	// 	os.MkdirAll("yaml_file", 0755)
	// 	// 处理http请求并创建job
	// 	for yaml := range cpnclient.JobQueue {
	// 		CpnJobID, err := cpnclient.NewJob(client, &yaml)
	// 		if err != nil {
	// 			log.Println("Skip this Job")
	// 			err = cpnclient.SendHttp("Task failed")
	// 			if err != nil {
	// 				log.Printf("Error: Lost connection with cpnserver ")
	// 				log.Printf("       making POST request: %v\n", err)
	// 				continue
	// 			}
	// 			continue
	// 		}
	// 		err = cpnclient.SendHttp(CpnJobID)
	// 		if err != nil {
	// 			log.Printf("Error: Lost connection with cpnserver ")
	// 			log.Printf("       making POST request: %v\n", err)
	// 			continue
	// 		}
	// 	}
	// }()

	// 创建监控器，实时检查cpn-job的namespaces下的job是否已经完成
	// 并连同prometheus的监控信息一起发送给cpnserver

	cpnclient.APP2(client)

}
