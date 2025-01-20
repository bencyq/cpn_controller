package version2

import (
	"time"

	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/client-go/kubernetes"
)

/////////////////////////////////////////////
// 定义调度器接口发送的数据结构，以example.json为例
/////////////////////////////////////////////

// Monitor 根结构体，包含 DataCenterNums 和 DataCenterInfo
type Monitor struct {
	DataCenterNums int              `json:"DataCenterNums"`
	DataCenterInfo []DataCenterInfo `json:"DataCenterInfo"`
	JobPool        JobPool
}

// DataCenterInfo 数据中心信息结构体
type DataCenterInfo struct {
	DataCenterID       string        `json:"DataCenterID"`
	DataCenterLocation string        `json:"DataCenterLocation"`
	ClusterNums        int           `json:"ClusterNums"`
	ClusterInfo        []ClusterInfo `json:"ClusterInfo"`
}

// ClusterInfo 集群信息结构体
type ClusterInfo struct {
	ClusterID                 string     `json:"ClusterID"`
	ClusterIP                 string     `json:"ClusterIP"`
	NodeNums                  int        `json:"NodeNums"`
	NodeInfo                  []NodeInfo `json:"NodeInfo"`
	ClusterPromIpPort         string     `json:"ClusterPromIpPort"`
	ClusterKubeconfigFilePath string     `json:"ClusterKubeconfigFilePath"`
	ClusterClientSet          *kubernetes.Clientset

	// 以下通过prometheus获取
	// TODO:网络指标，待定
}

// Node 节点信息结构体
type NodeInfo struct {
	NodeID   string     `json:"NodeID"`
	NodeIP   string     `json:"NodeIP"`
	CPUInfo  CPUInfo    `json:"CPUInfo"`
	NodeType string     `json:"NodeType"`
	CardNums int        `json:"CardNums"`
	CardInfo []CardInfo `json:"CardInfo"`

	// 以下通过prometheus获取
	CPU_USAGE    float64
	TOTAL_MEMORY int64
	FREE_MEMORY  int64
}

func (node *NodeInfo) FindCard(cardID string) (card *CardInfo) {
	for idx := range node.CardInfo {
		if node.CardInfo[idx].CardID == cardID {
			return &node.CardInfo[idx]
		}
	}
	return nil
}

// CPUInfo CPU 信息结构体
type CPUInfo struct {
	CPUNums      int    `json:"CPUNums"`
	Architecture string `json:"Architecture"`
	CPUModel     string `json:"CPUModel"`
	CPUCore      int    `json:"CPUCore"`
}

// CardInfo 显卡信息结构体
type CardInfo struct {
	CardID          string `json:"CardID"`
	CardModel       string `json:"CardModel"`
	GPU_UTIL        int64
	GPU_MEMORY_FREE int64
	GPU_MEMORY_USED int64
}

// ////////////////////////////
// 以下为自定义数据结构，为算法所用
// ////////////////////////////
type Card struct {
	ID              int
	CARDMODEL       string
	GPU_UTIL        float64
	GPU_MEMORY_FREE int64
	GPU_MEMORY_USED int64
}

// 定义prometheus的返回格式，不一定准确
type Result struct {
	Metric map[string]interface{} `json:"metric"`
	Value  []interface{}          `json:"value"`
}
type Data struct {
	ResultType string   `json:"resultType"`
	Result     []Result `json:"result"`
}
type PromResponse struct {
	Status string `json:"status"`
	Data   Data
}

type JobPool struct {
	Job []Job
}

type Job struct {
	// 从yaml文件读取的详细信息
	JobSpec batchv1.Job

	// Job的属性信息
	YamlFilePath string    `json:"YamlFilePath"` // yaml配置文件位置
	Timestamp    time.Time `json:"timestamp"`    // 作业提交的时间
	ID           string    `json:"id"`           // 作业ID
	JobModelName string    `json:"jobmodelname"` // 作业模型名字，如yolo、resnet、llama3等
	JobType      string    `json:"jobtype"`      // 作业类型 (CPU密集型 或 GPU密集型)
	MemoryReq    float64   `json:"memoryreq"`    // 内存需求
	GPUMemoryReq float64   `json:"gpumemoryreq"` // 显存需求 (仅GPU作业)
	DataSize     float64   `json:"datasize"`     // 数据大小
	// PresumedTime float64   `json:"presumedTime"` // 预计完成需要时间
	// CPUPowerReq  float64   `json:"cpupowereq"`   // CPU需求量，以Intel(R) Xeon(R) CPU E5-2630 v4 @ 2.20GHz的100%算力为基准
	// GPUPowerReq  float64   `json:"gpupowereq"`   // GPU算力需求量，以A100的100%为基准

	// Job 的分配信息
	ClusterID string `json:"clusterid"`
	NodeID    string `json:"nodeid"`
	CardID    string `json:"cardid"`
}
