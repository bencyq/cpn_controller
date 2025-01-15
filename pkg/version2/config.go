package version2

/////////////////////////////////////////////
// 定义调度器接口发送的数据结构，以example.json为例
/////////////////////////////////////////////

// DataCenterInfo 数据中心信息结构体
type DataCenterInfo struct {
	DataCenterID   string        `json:"DataCenterID"`
	DataCenterName string        `json:"DataCenterName"`
	ClusterNums    int           `json:"ClusterNums"`
	ClusterInfo    []ClusterInfo `json:"ClusterInfo"`
}

// ClusterInfo 集群信息结构体
type ClusterInfo struct {
	ClusterID                 string `json:"ClusterID"`
	ClusterIP                 string `json:"ClusterIP"`
	ClusterLocation           string `json:"ClusterLocation"`
	Nodes                     []Node `json:"Nodes"`
	ClusterPromIpPort         string `json:"ClusterPromIpPort"`
	ClusterKubeconfigFilePath string `json:"ClusterKubeconfigFilePath"`

	// 以下通过prometheus获取
	// TODO:网络指标，待定
}

// Node 节点信息结构体
type Node struct {
	NodeID   string   `json:"NodeID"`
	NodeIP   string   `json:"NodeIP"`
	CPUInfo  CPUInfo  `json:"CPUInfo"`
	NodeType string   `json:"NodeType"`
	CardInfo CardInfo `json:"CardInfo"` // FIXME:修改成 []Card

	// 以下通过prometheus获取
	CPU_USAGE    float64
	TOTAL_MEMORY int64
	FREE_MEMORY  int64
}

// CPUInfo CPU 信息结构体
type CPUInfo struct {
	Architecture string `json:"Architecture"`
	CPUModel     string `json:"CPU Model"`
	CPUCore      int    `json:"CPU Core"`
}

// CardInfo 显卡信息结构体
type CardInfo struct {
	CardType  string `json:"CardType"`
	CardMount int    `json:"CardMount"`
}

// Root 根结构体，包含 DataCenterNums 和 DataCenterInfo
type Root struct {
	DataCenterNums int              `json:"DataCenterNums"`
	DataCenterInfo []DataCenterInfo `json:"DataCenterInfo"`
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
	Data
}
