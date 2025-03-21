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
7. 模拟分析newJob在某个卡上的运行时间 TODO:randon forest 需要优化
8. 设计算法生成调度队列（考虑作业的运行时间和作业的传输时间）
9. 进行迁移镜像/模型，并向集群发送作业
10. 收集作业的日志和完成时间等数据，以便后续更新算法（对预测器没有见过的模型，可以考虑第一次运行时给它独占资源来收集数据）

## 目前进度以及TODO
- 上述设计流程的大部分功能已经实现
- 预测器功能实现完成，精度较高
- 测试AssignJob能否正常发送作业
- 准备好能运行的负载镜像
- 对作业预留资源的功能已经完成
- optimal_allocation算法完成，效果还可以，等待大批量的Job队列测试
- 自行实现提交作业的功能
- 浪潮的调度器不涉及到网络参数部分，准备自行模拟
- 目前的问题是，出现多个GPU作业在同一张卡上的情况（已经解决）
- 正常规模的作业队列测试已通过，准备分析日志来确认算法工作情况，以及是否存在漏洞（已完成）
- 实现了FIFO的baseline对比，FIFO下存在大量的OOM错误，甚至会发生死锁，导致作业队列无法结束
- 准备对比国网的默认调度策略
- 准备测试单卡单负载情况下的FIFO
- 长度为40的作业队列测试完毕，没问题；长度为100的作业队列测试中，出现了未给作业预留资源、SchduleFailedJob中的作业未重新分配的问题
  且在分配到040之后，出现了作业被提交到多个节点上的情况，且作业提交没有走正常的流程
  (暂时怀疑是实验环境有问题)
- 可做消融实验，目前CPU阈值限制为0.7，试试0.8、0.9
- 快速重启服务可能导致异常，因为Prometheus的抓取时间为1mins一次，即使进程停了，Prometheus也没更新
- 增加了预测器xgboost的对比，xgboost的效果很差

## 后续优化方向
1. 目前的策略，会导致资源需求量高的作业一直等待，需要优化；准备对GPU密集型作业进行预留策略，避免长时等待；这部分已经设计了资源预留的策略解决，等待测试
2. 设计作业完成后的自动触发机制，避免定期遍历带来的资源浪费（TODO:现在是一分钟遍历一次AssignedJob）
3. 设计OriginJob获取的动态机制
4. 设计monitor信息的动态扩展（现在很多引用都是用monitor.DataCentorInfo[idx]这种形式进行的）
5. 考虑用别的算法来对作业队列进行分配
6. 重构代码结构，把部分predictor里的代码迁移到别处 TODO:
7. 预测器的输入结果偶尔会有问题，需要调整

## 注意
1. `pkg/python/socket_client.go`和`pkg/python/socket_server.py`里面定义的socket路径有可能会出问题
2. yaml文件里的annotations，格式为
    ```yaml
    annotations: 
        model_name: "densenet121"
        data_size: 20
        epoch: 100
    ```
3. github仓库里没有模型权重文件，运行pkg/python/random_forest_train.py获取
4. kubeconfig文件里，修改目标ip
5. 国网环境下，作业需要配置hami scheduler
   ```
   spec:
        backoffLimit: 4
        template:
            metadata:
            creationTimestamp: null
            annotations:
                **hami.io/resource-pool: "poc"**
    ```
    且需要如下配置，才能正常使用mps
    ```
    resources:
        limits:
            nvidia.com/gpu: <物理GPU的个数>
    ```
    目前的方案是，直接不配置nvidia.com/gpu，不使用hami scheduler
6. 作业队列的随机生成功能在pkg/utils/utils.go下
7. 目前版本的代码可能存在问题，需要进一步排查；完成normalscale测试的那次commit的代码应该可以复现