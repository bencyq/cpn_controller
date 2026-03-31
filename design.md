# 总体设计
  调度逻辑分成两层。第一层是粗筛，仍放在 cpn_controller/pkg/controller/optimal_allocation.go:13 里，继续保留“CPU 阈值过滤、节点内存过滤、显存 OOM 过滤”，但
  去掉当前 len(cardInfo.JobQueue) >= 3 的硬编码上限，因为你要分析的是并行状态，不是禁止并行。显存过滤的数据源不再依赖 cpn_controller/pkg/controller/
  predictor.go:84 里的静态 model_baseline.csv，而是直接读取 YAML 里的 gpu_memory_req_mb，并保留 1GiB 左右安全边际，避免在高并发下卡边缘 OOM。

  第二层是候选卡评分。TotaltimePredict() 不再迭代模拟卡上已有作业什么时候结束，而是只做一件事：读取候选卡当前并行态快照，调用外置预测器得到“这个新作业在这
  张卡、当前负载下的单 epoch 秒数 single_epoch_sec”，然后返回 single_epoch_sec * epoch。这样你要的“卡 1 和卡 2 都已有任务时，比较新作业落在哪张卡更快”就完
  全落在预测器输入里，而不是落在 controller 侧的离散事件仿真里。所有时间单位统一为秒，epoch 是整数轮次，TotaltimePredict()、ReservedTime、日志输出都统一用
  秒。

  外置预测器设计
  外置预测器仍建议沿用 cpn_controller 现有 Go + Python socket 模式，只是把 cpn_controller/pkg/controller/predictor.go:177 现在的 RandomForestPredict() 换成
  一个 ONNX/HASE 适配器。实现上不改 hase 仓库，新增 cpn_controller/pkg/python/onnx_predictor_server.py 和一个轻量 client 即可。这个 server 接收候选位置信
  息：model_name、model_path、epoch、gpu_type、node_name、card_id、当前 parallel_jobs 模型名列表、当前 GPU/DCGM 指标。内部逻辑直接对齐 hase/inference/
  predict_model_latency.py:265 的 I/O：先用 model_path 找 DAG JSON；如果 hase/graph_model/model_DAG/resnet18_dag.json:1 没有，就按 hase/inference/
  predict_model_latency.py:157 从 hase/ort_analysis/ort_kernel_record/resnet50.json:1 自动生成；再按 kernel shape 构造运行时特征，调用 RandomForest 回归器
  得到每个 kernel 的 latency_ms，最后累计成单 epoch 总时延。

  精度不是这一阶段目标，所以回归器部分必须“永不阻塞整体流程”。建议做双层回退：优先加载默认权重；如果权重缺失或不可用，就用固定随机种子的伪随机回归器或常数
  回归器按 kernel_group + shape + gpu_type 产出正数预测值，保证每次同输入输出稳定。也就是说，这个 predictor 先保证“能出一个合理正值”，再逐步追精度。

  提交路径设计
  所有 ONNX 作业的容器统一标准化到 bencyq/nnmeter:202602122308。这件事最好在 cpn_controller/pkg/controller/assignjob.go:18 里做最终兜底，而不是依赖每个
  YAML 都写对。AssignJobToNode() 保留当前的 CUDA_VISIBLE_DEVICES、NVIDIA_VISIBLE_DEVICES=all、vgpu、HostIPC、/tmp/nvidia-mps 逻辑不动，只新增一个只读
  hostPath 卷，把宿主机 /data/cyq/hase/model_zoo/models 挂进容器同路径。这样 node191 和 node200 只要提前把模型放到同一目录，YAML 里的 model_path 就不用在
  controller 里重写。脚本路径仍由 YAML 传入，controller 只做校验和透传，不负责脚本内容。

  单元规划与单元验收标准

  1. Job 合同单元：改 cpn_controller/pkg/controller/config.go:165 和 cpn_controller/pkg/controller/monitor.go:260，新增 ModelPath、ContainerImage、
  LaunchScript、GPUMemoryReq 解析。验收标准：缺任何必填字段立即拒绝入队；model_name 与 model_path 文件名前缀不一致时拒绝；epoch <= 0 或 gpu_memory_req_mb
  <= 0 时拒绝。
  2. 提交改造单元：改 cpn_controller/pkg/controller/assignjob.go:18，统一镜像、统一模型目录挂载。验收标准：生成的 Job 里 image 固定为 bencyq/
  nnmeter:202602122308；存在只读模型挂载；保留现有 CUDA/MPS/NodeSelector 注入；脚本命令和 args 未被破坏。
  3. 粗筛单元：改 cpn_controller/pkg/controller/optimal_allocation.go:13，去掉 >=3 上限，保留 CPU/内存/显存过滤。验收标准：CPU 超阈值节点被排除；显存不足卡
  被排除；卡上已有作业但显存足够的候选卡仍会进入预测阶段。
  4. 预测器适配单元：新增 cpn_controller 内 Python server，按 HASE 输入输出适配，不改 hase 源码。验收标准：给一个存在的 ONNX 模型和对应 kernel JSON，返回
  single_epoch_sec > 0；缺 DAG 但有 kernel JSON 时可自动补 DAG；权重缺失时仍能返回稳定正数，不会让 controller 卡死。
  5. 总时长评估单元：重写 cpn_controller/pkg/controller/predictor.go:100 的 TotaltimePredict()，语义改为“单次 predictor 调用结果乘 epoch”。验收标准：mock
  predictor 返回 2.5 sec/epoch、epoch=100 时，TotaltimePredict() 必须返回 250 sec；单位全部为秒，不允许毫秒秒混用。
  6. 选址决策单元：保留 cpn_controller/pkg/controller/optimal_allocation.go:41 的“遍历候选卡、选最小总时长”主干。验收标准：mock 卡 0 预测 300 秒、卡 1 预测
  240 秒时必选卡 1；并列时按固定顺序打破平局，避免调度抖动。
  7. fixture 与启动单元：继续使用 cpn_controller/cmd/controller/main.go:16 指向的 example2.json。验收标准：启动时只看到 node200、node191 两个 GPU 节点；
  controller 不再依赖真实训练权重、也不要求扩 HASE 训练面才可启动。
  8. 安全测试单元：现有 cpn_controller/pkg/controller/assignjob_test.go:7 这类测试会连真实环境，不适合作为单元验收。验收标准：新增纯单元测试必须使用 fake
  clientset 或拆 helper 测 Job 对象，不允许 NewMonitor() 直连集群作为默认测试路径。

  上线前的通过条件

  - 两台节点 node191、node200 上都已存在 /data/cyq/hase/model_zoo/models，且目标 ONNX 文件可读。
  - Controller 在没有新增训练、没有新增 GPU 画像、只有默认/随机权重的前提下，能完成“读 YAML -> 预测 -> 选卡 -> 提交”全流程。
  - 对同一个新作业，若卡 1、卡 2 当前并行态不同，日志里能清楚打印两边的 single_epoch_sec 和 total_sec，最终选择最小时延的卡。
  - 整体时间单位在代码、日志、测试里全部是秒。