# cpn controller
- 只在广域调度器所在的节点起一个服务程序，负责信息的收集和作业队列的提交
- 测试环境里的prometheus的ip段可以直接访问到，实际生产环境中，需要访问的svc的ip段会有调度器来实现转换，这个不必操心，也就不用另外设计广域调度器的客户端节点
- 使用k8s的api来发送作业，而不是kubectl apply -f
- 实现预测器
- 使用算法实现作业队列的安排

## 流程设计
1. 初始化调度策略模块，从调度器接口获取到集群的详细信息
2. 测试每个集群的prometheus是否能成功获取到需要的metric，并定期收集
3. 测试每个集群的Job、Namespace等信息能能否成功获取到，并缓存 TODO: 这部分功能先不开发，先完成静态的调度
4. 设计接口接受调度器的作业提交，解析yaml文件，并缓存作业
5. 在每个集群的每台服务器上运行基准测试程序，获得评价指标（暂定resnet50、yolov8m、llama3，每个各10mins）TODO: 先用静态配置的方法，后续接入自动化功能
6. 实现预测器的功能（能够根据提供的模型信息，给出指标）
7. 设计算法生成调度队列（考虑作业的运行时间和作业的传输时间）
8. 向调度器发送迁移镜像/模型的命令，并向集群发送作业
9. 收集作业的日志和完成时间等数据，以便后续更新算法（对预测器没有见过的模型，可以考虑第一次运行时给它独占资源来收集数据）

## 注意
1. `pkg/version2/socket_client.go`和`pkg/version2/socket_server.py`里面定义的socket路径有可能会出问题
2. yaml文件里的annotations，格式为
    ```yaml
    annotations: 
        model_name: "densenet121"
        data_size: 20
        epoch: 100
    ```