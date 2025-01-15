package version2

import (
	"testing"
)

func TestUnmarshalJson(t *testing.T) {
	jsonStr := `{
		"DataCenterNums": 3,
		"DataCenterInfo": [
			{
				"DataCenterID": "xxx",
				"DataCenterName": "xxx",
				"ClusterNums": 4,
				"ClusterInfo": [
					{
						"ClusterID": "xxxx",
						"ClusterIP": "27.154.1.18",
						"ClusterLocation": "北京",
						"Nodes": [
							{
								"NodeID": "4090xxx",
								"NodeIP": "27.154.1.18",
								"CPUInfo": {
									"Architecture": "x86_64",
									"CPU Model": "Intel(R) Xeon(R) Platinum 8358P * 2",
									"CPU Core": 64
								},
								"NodeType": "GPU",
								"CardInfo": {
									"CardType": "A100",
									"CardMount": 8
								}
							}
						],
						"ClusterPromIpPort": "27.154.1.18:31000",
						"ClusterKubeconfigFilePath": "/directory/file"
					}
				]
			}
		]
	}`
	unmarshalJson([]byte(jsonStr))
}
