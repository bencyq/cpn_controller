# cpn controller
- 只在广域调度器所在的节点起一个服务程序，负责信息的收集和作业队列的提交
- 测试环境里的prometheus的ip段可以直接访问到，实际生产环境中，需要访问的svc的ip段会有调度器来实现转换，这个不必操心，也就不用另外设计广域调度器的客户端节点
- 使用k8s的api来发送作业，而不是kubectl apply -f
- 实现预测器
- 使用算法实现作业队列的安排
