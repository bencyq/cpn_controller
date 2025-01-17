package version2

/////////////////////////////////////////////
// 定义调度器接口发送的数据结构，以example.json为例
/////////////////////////////////////////////

// Root 根结构体，包含 DataCenterNums 和 DataCenterInfo
type Root struct {
	DataCenterNums int              `json:"DataCenterNums"`
	DataCenterInfo []DataCenterInfo `json:"DataCenterInfo"`
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
