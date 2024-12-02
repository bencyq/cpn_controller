package cpnserver

// http服务器监听地址
const ServerIP = "0.0.0.0:23981"

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
}
