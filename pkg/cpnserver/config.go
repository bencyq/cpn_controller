package cpnserver

// http服务器监听地址
const ServerIP = "0.0.0.0:23981"

type CardModel struct {
	ComputingPower float64 // 算力等级，以A100为基准1,后期需要依照数据集打分TODO:
}

var (
	NVIDIA_GeForce_RTX_4090 = CardModel{ComputingPower: 1.1}
	NVIDIA_RTX_A6000_ADA    = CardModel{ComputingPower: 0.8}
	NVIDIA_A100             = CardModel{ComputingPower: 1}
	NVIDIA_V100             = CardModel{ComputingPower: 0.8}
	NVIDIA_P100             = CardModel{ComputingPower: 0.6}
)

// TODO:现在先手动配置，后期增加自动信息初始化的功能
var cluster_one = Cluster{
	name:   "cluster-one",
	ipPort: "10.90.1.49:23980",
	node: func() []Node {
		nodes := []Node{}
		nodes = append(nodes, Node{
			name: "node16",
			card: func() []Card {
				cards := []Card{}
				for i := 0; i < 2; i++ {
					cards = append(cards, Card{id: i})
				}
				return cards
			}(),
			CARDMODEL: NVIDIA_GeForce_RTX_4090,
		})
		nodes = append(nodes, Node{
			name: "node235",
			card: func() []Card {
				cards := []Card{}
				for i := 0; i < 8; i++ {
					cards = append(cards, Card{id: i})
				}
				return cards
			}(),
		})
		return nodes
	}(),
	bandwidth: 100,
}
