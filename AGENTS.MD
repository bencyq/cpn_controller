# cpn_controller Agent Guide

## 仓库定位

这是一个基于 Go 的多集群作业控制器，目标是在广域调度器提供的集群拓扑之上，结合 Prometheus 指标、模型基线和预测器结果，为训练/推理 Job 选择落点并通过 Kubernetes API 直接提交。仓库同时保留了一个 `FIFO` baseline，用于和主调度策略做对比实验。

主流程在代码中的落点如下：

1. `cmd/controller/main.go`
   设置 `controller.JsonUrl = "example2.json"`，创建 `Monitor`，启动 Python 随机森林预测器，然后进入持续调度循环。
2. `pkg/controller/monitor.go`
   `NewMonitor()` 会加载集群 JSON、创建每个集群的 `clientset`、确保 namespace 存在、抓取 Prometheus 指标、检查 benchmark、读取 `yaml_queue` 中的 Job。
3. `pkg/controller/predictor.go`
   读取模型基线 CSV，拉起 `pkg/python/socket_server.py`，通过 Unix socket 调用 Python 随机森林模型。
4. `pkg/controller/optimal_allocation.go`
   对每个候选卡位模拟总时间，选择最优落点；GPU 作业无位可放时走资源预留逻辑。
5. `pkg/controller/assignjob.go`
   将分配结果写回 Job，注入环境变量、资源限制、挂载和节点选择器，再通过 K8s API 提交并在后续循环里监控完成情况。

## 目录地图

- `cmd/controller`
  主调度器入口。运行它会真正连 Prometheus、连 K8s，并提交 Job。
- `cmd/fifo`
  FIFO baseline 入口。使用 `pkg/fifo` 的顺序放置策略。
- `pkg/controller`
  核心调度逻辑。
  `config.go`: 拓扑、节点、GPU 卡、Job、JobPool 等核心结构体，以及全局变量 `JsonUrl`、`NAMESPACE`。
  `monitor.go`: 初始化、Prometheus 抓取、K8s client 创建、yaml 作业读取。
  `predictor.go`: 模型基线读取、Python 预测器管理、总时间预测。
  `optimal_allocation.go`: 最优分配和资源预留。
  `assignjob.go`: Job 变换、提交、轮询、删除。
  `example*.json`: 样例拓扑配置。
  `model_baseline*.csv`: 模型显存/速度基线和真实数据表。
  `yaml_queue/`: 默认作业输入队列，当前仓库里约 493 个样例 YAML。
- `pkg/fifo`
  并行负载下的 FIFO baseline。
- `pkg/fifo_without_parallel`
  单卡单负载实验用 baseline，代码里也标了 `TODO/FIXME`。
- `pkg/python`
  Python 预测器和训练脚本。
  `socket_server.py`: Unix socket 服务端，供 Go 调用。
  `random_forest_predict.py`: 加载 `random_forest_weight.pt` 并做推理。
  `random_forest_train.py`: 训练并生成 `random_forest_weight.pt`。
  `datasets/`, `train.csv`, `test.csv`: 训练/分析数据。
- `pkg/utils`
  项目根目录发现、CSV 读取、随机生成 Job 队列。
- `test_logs`, `metrics`, `nohup.out`
  实验产物，不是核心源码。
- `vendor`
  已 vendored 的 Go 依赖，除非明确要升级依赖，否则不要改。

## 运行和环境假设

- Go 版本写在 `go.mod` 中：`go 1.23.1`，`toolchain go1.23.4`。
- 运行主调度器前，需要真实可达的：
  `example*.json` 里配置的 Prometheus 地址；
  `example*.json` 里配置的 kubeconfig 路径；
  `pkg/controller/yaml_queue/` 中的待调度 Job YAML。
- `NewMonitor()` 不是纯构造函数，有明显副作用：
  会建 K8s client、检查/创建 namespace、抓取实时指标、读入作业队列。
  在测试或脚本里调用它前，默认按“会接触外部环境”处理。
- Go 和 Python 两边都通过“向上查找 `go.mod`”来定位项目根目录，所以命令必须从仓库内执行，不能随意切到外部目录。
- `cmd/controller/main.go` 和 `cmd/fifo/main.go` 当前都把 `JsonUrl` 写死成 `example2.json`；`NAMESPACE` 也是全局变量，baseline 会覆盖它。
- `pkg/python/random_forest_predict.py` 依赖 `pkg/python/random_forest_weight.pt`。如果权重文件缺失，需要先运行训练脚本生成。

## Job 和调度约束

- Job 元数据主要依赖 YAML `annotations`：
  `model_name`
  `data_size`
  `epoch`
- `pkg/controller/predictor.go` 当前用模型名做静态分类：
  `llama3`、`glm4`、`qwen2.5` 视为 `GPU` 作业；
  其他模型默认按 `CPU` 作业处理。
- `AssignJobToNode()` 会原地修改 Job 模板，关键动作包括：
  设置 `CUDA_VISIBLE_DEVICES` / `NVIDIA_VISIBLE_DEVICES`
  注入 `k8s.amazonaws.com/vgpu: 1`
  打开 `HostIPC`
  挂载 `/tmp/nvidia-mps`
  写入目标节点 `NodeSelector`
  如果要改提交流程、资源模型或 Hami/MPS 兼容逻辑，优先看这里。
- `OptimalAllocate()` 会过滤高 CPU 占用节点、显存不足卡、P100 上的 LLM，并调用 `TotaltimePredict()` 做时间模拟。
- `ReserveAllocate()` 专门处理暂时放不下的 GPU 大作业，为某张卡预留未来时间窗。

## 测试和修改建议

- 这个仓库里的很多 `*_test.go` 不是纯单元测试，而是直接调用 `NewMonitor()`、连接 K8s、删除/提交 Job，甚至启动 Python 预测器。
- 不要默认执行 `go test ./...`。先确认是否会碰真实集群和实验环境。
- 更安全的检查顺序通常是：
  先读 `cmd/` 和 `pkg/controller/` 的调用链；
  再做编译级验证；
  最后才跑有外部副作用的测试。
- 如果只改调度算法，重点保持 `JobPool` 和各卡 `JobQueue` / `GPU_MEMORY_FREE` / `GPU_MEMORY_USED` 的一致性。
- 如果要增强可维护性，优先考虑：
  把 `JsonUrl`、`NAMESPACE`、socket 名称等全局常量改成显式配置；
  给 `NewMonitor()` 拆出无副作用初始化；
  把实验数据、日志和样例队列与核心逻辑进一步分离。

## 协作时优先阅读的文件

1. `README.md`
2. `cmd/controller/main.go`
3. `pkg/controller/config.go`
4. `pkg/controller/monitor.go`
5. `pkg/controller/predictor.go`
6. `pkg/controller/optimal_allocation.go`
7. `pkg/controller/assignjob.go`
