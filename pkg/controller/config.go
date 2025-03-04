package controller

import (
	"fmt"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/client-go/kubernetes"
)

/////////////////////////////////////////////
// 定义调度器接口发送的数据结构，以example.json为例
/////////////////////////////////////////////

// Monitor 根结构体，包含 DataCenterNums 和 DataCenterInfo
type Monitor struct {
	DataCenterNums int               `json:"DataCenterNums"`
	DataCenterInfo []*DataCenterInfo `json:"DataCenterInfo"`
	JobPool        JobPool
	ModelBaseline  map[string][]string `json:"-"`
	ModelBaseline2 [][]string          `json:"-"`
}

// DataCenterInfo 数据中心信息结构体
type DataCenterInfo struct {
	DataCenterID       string         `json:"DataCenterID"`
	DataCenterLocation string         `json:"DataCenterLocation"`
	ClusterNums        int            `json:"ClusterNums"`
	ClusterInfo        []*ClusterInfo `json:"ClusterInfo"`
}

// ClusterInfo 集群信息结构体
type ClusterInfo struct {
	ClusterID                 string                `json:"ClusterID"`
	ClusterIP                 string                `json:"ClusterIP"`
	NodeNums                  int                   `json:"NodeNums"`
	NodeInfo                  []*NodeInfo           `json:"NodeInfo"`
	ClusterPromIpPort         string                `json:"ClusterPromIpPort"`
	ClusterKubeconfigFilePath string                `json:"ClusterKubeconfigFilePath"`
	ClusterClientSet          *kubernetes.Clientset `json:"-"`

	// 以下通过prometheus获取
	// TODO:网络指标，待定
}

// Node 节点信息结构体
type NodeInfo struct {
	NodeID   string      `json:"NodeID"`
	NodeIP   string      `json:"NodeIP"`
	CPUInfo  CPUInfo     `json:"CPUInfo"`
	NodeType string      `json:"NodeType"`
	CardNums int         `json:"CardNums"`
	CardInfo []*CardInfo `json:"CardInfo"`

	// 以下通过prometheus获取
	CPU_USAGE    float64
	TOTAL_MEMORY int64 // 单位为MB
	FREE_MEMORY  int64 // 单位为MB

	Bandwidth int64 `json:"Bandwidth"` // 单位为 MB/s

}

func (node *NodeInfo) FindCard(cardID string) (card *CardInfo) {
	for idx := range node.CardInfo {
		if node.CardInfo[idx].CardID == cardID {
			return node.CardInfo[idx]
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

	// 分配到该卡上的作业
	JobQueue JobQueue

	// 预留的Job和时间
	ReservedTime int64
	ReservedJob  *Job

	// 基准测试程序获得的分数
	BenchMark BenchMark
}

// ////////////////////////////
// 以下为自定义数据结构，为算法所用
// ////////////////////////////

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
	OriginJob    JobQueue // 初始作业队列，按照FIFO排序
	ScheduledJob JobQueue // 由调度器返回，将PreJobID为null的排列在最前，只要PreJobID为null，则直接发送作业到指定位置
	AssignedJob  JobQueue // 已经提交的作业，等待作业完成
	FinishedJob  JobQueue
	ReservedJob  JobQueue // 预留资源的作业
}

type JobQueue []*Job

func (jq JobQueue) GetID() []string {
	IDs := []string{}
	for _, job := range jq {
		IDs = append(IDs, job.ID)
	}
	return IDs
}
func (jq JobQueue) List() {
	for _, job := range jq {
		fmt.Printf("ID:%v JobModelName:%v Allocation: %v,%v,%v,%v\n", job.ID, job.JobModelName, job.DataCenterIDX, job.ClusterIDX, job.NodeIDX, job.CardIDX)
	}
}

func (jq *JobQueue) RemoveJob(jobID string) bool {
	for i := range *jq {
		if (*jq)[i].ID == jobID {
			*jq = append((*jq)[:i], (*jq)[i+1:]...)
			return true
		}
	}
	return true
}

type Job struct {
	// 从yaml文件读取的详细信息
	Batchv1Job batchv1.Job `json:"-"`

	// Job的属性信息
	YamlFilePath  string    `json:"YamlFilePath"` // yaml配置文件位置
	Timestamp     time.Time `json:"timestamp"`    // 作业提交的时间
	ID            string    `json:"id"`           // 作业ID
	JobModelName  string    `json:"jobmodelname"` // 作业模型名字，如yolov8n、resnet50、llama3等
	JobType       string    `json:"jobtype"`      // 作业类型 (CPU密集型 或 GPU密集型), 写为 CPU GPU
	MemoryReq     int64     `json:"memoryreq"`    // 内存需求，单位为MB
	GPUMemoryReq  int64     `json:"gpumemoryreq"` // 显存需求，单位为MB (仅GPU作业)
	DataSize      int64     `json:"datasize"`     // 数据大小，单位为GB
	Epoch         int64
	BaselineSpeed float64 // 单epoch的推理时间
	// PresumedTime float64   `json:"presumedTime"` // 预计完成需要时间
	// CPUPowerReq  float64   `json:"cpupowereq"`   // CPU需求量，以Intel(R) Xeon(R) CPU E5-2630 v4 @ 2.20GHz的100%算力为基准
	// GPUPowerReq  float64   `json:"gpupowereq"`   // GPU算力需求量，以A100的100%为基准

	// Job 的分配信息
	DataCenterIDX int
	ClusterIDX    int
	NodeIDX       int
	CardIDX       int
	PreJobIDX     int       // 作业队列信息
	TransferTime  int64     // 以秒为单位
	AssignedTime  time.Time // 调度器提交作业的时间

	// Job的预留信息
	IsReserved           bool  //是否为预留类型的作业
	ReservedTime         int64 // 预计的等待时间
	ReservationStartTime time.Time
}

// 基准测试，收集三个类型的模型的单位epoch运行时间
type BenchMark struct {
	Model1AVGRunTime float64
	Model2AVGRunTime float64
	Model3AVGRunTime float64
}

const NAMESPACE = `cpn-controller`
