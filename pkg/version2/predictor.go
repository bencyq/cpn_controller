package version2

import "log"

// 5. 在每个集群的每台服务器上运行基准测试程序，获得评价指标（暂定resnet50、yolov8m、llama3，每个各10mins）
// 6. 实现预测器的功能（能够根据提供的模型信息，给出指标），预测其在A100上的平均运行时间

// 检测每个节点是否已经跑过benchmark
func (monitor *Monitor) checkBenchMark() {
	for _, datacenter := range monitor.DataCenterInfo {
		for _, cluster := range datacenter.ClusterInfo {
			for _, node := range cluster.NodeInfo {
				if node.BenchMark.Model1AVGRunTime == 0.0 {
					monitor.runBenchMark(datacenter.DataCenterID, cluster.ClusterID, node.NodeID)
					log.Printf("INFO: No BenchMark in DataCenter: %v\tClusterID: %v\tNodeID: %v\t", datacenter.DataCenterID, cluster.ClusterID, node.NodeID)
				}
			}
		}
	}
}

// 运行基准测试程序，获得评价指标 TODO: 先在json文件里手动配置，后续增加功能
func (monitor *Monitor) runBenchMark(DataCenterID string, ClusterID string, NodeID string) {

}

// 预测器，调用python程序（考虑以容器注册的形式）TODO:
func Predict(job *Job) {

}
