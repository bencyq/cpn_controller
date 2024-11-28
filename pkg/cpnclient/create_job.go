package cpnclient

// 启动一个dynamic client，负责接收http请求，并启动Job
// 先使用out of cluster的配置，以便调试，后期改成in cluster配置

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"sigs.k8s.io/yaml"
)

// 以out of cluster 的方式来新建client
func NewDynamicClientOutOfCluster() (client *dynamic.DynamicClient, err error) {
	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		return nil, err
	}
	client, err = dynamic.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func NewClientSetOutOfCluster() (client *kubernetes.Clientset, err error) {
	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "admin.conf"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		return nil, err
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	log.Println("clientset succeed !")
	return clientset, nil
}

var JobQueue = make(chan []byte, 100) // 内存队列，用于存储 YAML 数据，缓存为100条

// 接收http请求
func handler(w http.ResponseWriter, r *http.Request) {
	// 只接受 POST 请求
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	// 读取请求体中的数据
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	JobQueue <- body

	fmt.Fprintln(w, "Message received")
}

// 创建一个http服务器
func newHttpClient() {
	http.HandleFunc("/", handler)
	go func() {
		err := http.ListenAndServe(ClientIP, nil)
		if err != nil {
			log.Fatal("Error starting server: ", err)
		}
	}()
	log.Printf("Listening on %v...", ClientIP)
	log.Println("HttpClient succeed!")
}

// 解析yaml中附带的CpnJobID，并删除这一行
func YamlProcess(msg *[]byte) (CpnJobID string, err error) {
	// 将 yaml 解析为go结构
	var obj map[string]interface{}
	err = yaml.Unmarshal(*msg, &obj)
	if err != nil {
		return "", err
	}
	if CpnJobID, ok := obj["CpnJobID"].(string); ok {
		delete(obj, "CpnJobID")
		*msg, err = yaml.Marshal(obj)
		if err != nil {
			return "", err
		}

		// 将处理好的yaml文件存放到本地
		err = os.WriteFile("yaml_file/"+CpnJobID+".yaml", *msg, 0644)
		if err != nil {
			return "", err
		}
		return CpnJobID, nil
	} else {
		log.Println("Error: yaml has no CpnJobID!")
		return "", errors.New("error yaml has no cpnjobid")
	}
}

// 负责新建Job
func NewJob(msg *[]byte) (CpnJobID string, err error) {

	// 查询CpnJobID，并处理好yaml文件
	CpnJobID, err2 := YamlProcess(msg)
	if err2 != nil {
		log.Printf("Error: parsing YAML: %v\n", err)
		return "", err2
	}

	// 直接调用kubectl apply -f 来执行
	cmd := exec.Command("kubectl", "create", "-f", "yaml_file/"+CpnJobID+".yaml")
	_, err = cmd.Output()
	if err != nil {
		log.Printf("Error CpnJobID %v : %v", CpnJobID, err)
		return "", err
	}
	log.Printf("CpnJobID %v applied successfully\n", CpnJobID)

	return CpnJobID, nil
}

// 负责完成Job创建后的http确认信息发送
func SendHttp(CpnJobID string) (err error) {
	// 创建 POST 请求
	resp, err := http.Post(CpnServerURL, "text/plain", strings.NewReader(CpnJobID))
	if err != nil {
		// log.Printf("Error: making POST request: %v", err)
		return err
	}
	defer resp.Body.Close()

	// 打印响应状态码
	log.Printf("Send CpnJobID %v Response HTTP Status: %v\n", CpnJobID, resp.Status)
	return nil
}

// 负责整个create_job的运行逻辑
// 用子进程启动，避免阻塞主进程
func APP1() {
	go func() {
		// 创建http服务端
		newHttpClient()
		// 创建yaml文件的存储目录
		os.MkdirAll("yaml_file", 0755)

		// 处理http请求并创建job
		for yaml := range JobQueue {
			CpnJobID, err := NewJob(&yaml)
			if err != nil {
				log.Println("Skip this Job")
				err = SendHttp("Task failed")
				if err != nil {
					log.Printf("Error: Lost connection with cpnserver ")
					log.Printf("       making POST request: %v\n", err)
					continue
				}
				continue
			}

			err = SendHttp(CpnJobID)
			if err != nil {
				log.Printf("Error: Lost connection with cpnserver ")
				log.Printf("       making POST request: %v\n", err)
				continue
			}
		}
	}()

}
