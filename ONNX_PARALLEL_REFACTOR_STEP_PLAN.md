# ONNX 并行调度改造分步计划

本文档用于主 Agent 按顺序把任务逐个下发给子 Agent。
约束是：一次只执行一个任务；上一个任务完成并验收通过后，主 Agent 才推进下一个任务。

## 执行规则

- 每个任务都必须只解决一个问题。
- 每个任务都必须能独立开始，也必须有明确完成状态。
- 每个任务完成后都要先做该任务自己的最小验收，再进入下一个任务。
- 默认禁止运行会连接真实集群的测试；优先新增纯单元测试、纯编译检查、纯脚本自检。
- 所有时间单位统一为秒。
- 集群元数据固定使用 `pkg/controller/example2.json`。
- HASE 仓库只读复用，不扩训练面，不新增 GPU 画像数据，不要求先做精度优化。
- ONNX 统一容器镜像固定为 `bencyq/nnmeter:202602122308`。
- ONNX 模型宿主机统一目录固定为 `/data/cyq/hase/model_zoo/models`。
- 容器内模型挂载目录也固定为 `/data/cyq/hase/model_zoo/models`，避免 controller 再做路径转换。
- 新的代码跟改在`/data/cyq/cpn_controller`里面做

## 任务 01 固化 ONNX 运行常量

- 目标: 在 `cpn_controller` 内新增单一来源的 ONNX 运行常量，先不接入任何业务逻辑。
- 起点: 镜像、模型目录、示例 JSON 路径没有统一常量定义。
- 终点: 存在明确常量，至少包含 ONNX 镜像、宿主机模型目录、容器内模型目录、固定集群元数据文件名。
- 修改范围: `pkg/controller/` 下新增一个轻量配置文件，例如 `onnx_runtime.go`。
- 验收: `go test ./pkg/controller -run '^$'` 能通过编译。

## 任务 02 扩展 Job 结构体承载 ONNX 作业契约

- 目标: 给 `Job` 增加 ONNX 作业所需字段，但不修改读取逻辑。
- 起点: `Job` 仍主要围绕 `model_name/data_size/epoch`。
- 终点: `Job` 至少包含 `ModelPath`、`LaunchScript`、`ContainerImage`、`GPUMemoryReq`、可选的 `PredictSingleEpochSec` 或等价缓存字段。
- 修改范围: `pkg/controller/config.go`。
- 验收: `go test ./pkg/controller -run '^$'` 能通过编译。

## 任务 03 提取纯函数解析 ONNX annotations

- 目标: 把 YAML annotations 到 `Job` 字段的解析逻辑抽成纯函数。
- 起点: `monitor.getJobWithFile()` 直接在函数体里读取 annotations。
- 终点: 存在一个纯函数，负责读取 `model_name`、`model_path`、`epoch`、`gpu_memory_req_mb`、`container_image`、`launch_script` 等字段。
- 修改范围: `pkg/controller/monitor.go`，可新增 `pkg/controller/job_contract.go`。
- 验收: 新增纯单元测试 `TestParseOnnxJobAnnotations`，命令 `go test ./pkg/controller -run '^TestParseOnnxJobAnnotations$'` 通过。

## 任务 04 提取纯函数校验 ONNX 作业契约

- 目标: 把 ONNX 作业必填项校验从读取流程中独立出来。
- 起点: annotations 即使缺字段也不会明确报错。
- 终点: 存在一个纯校验函数，至少校验字段齐全、`epoch > 0`、`gpu_memory_req_mb > 0`、`model_name` 与 `model_path` 基本一致。
- 修改范围: `pkg/controller/job_contract.go` 或等价文件。
- 验收: 新增纯单元测试 `TestValidateOnnxJobContract`，命令 `go test ./pkg/controller -run '^TestValidateOnnxJobContract$'` 通过。

## 任务 05 把 YAML 入队流程切到新解析和校验函数

- 目标: 让 `getJobWithFile()` 真正使用任务 03 和任务 04 的纯函数。
- 起点: YAML 入队逻辑仍直接写死旧字段解析。
- 终点: 非法 ONNX YAML 不会进入 `OriginJob`，合法 YAML 会完整填充 `Job`。
- 修改范围: `pkg/controller/monitor.go`。
- 验收: 新增纯单元测试 `TestGetJobWithFileRejectsInvalidOnnxYaml`，命令 `go test ./pkg/controller -run '^TestGetJobWithFileRejectsInvalidOnnxYaml$'` 通过。

## 任务 06 修正 example2 集群卡型字符串

- 目标: 让 `example2.json` 的 GPU 型号只反映 `A6000` 和 `A100-SXM4`。
- 起点: 当前 fixture 仍存在 `NVIDIA A100-PCIE-40GB` 字符串。
- 终点: `example2.json` 中只保留 `NVIDIA RTX A6000` 和 `NVIDIA A100-SXM4-40GB` 或团队确认后的唯一标准写法。
- 修改范围: `pkg/controller/example2.json`。
- 验收: `rg -n "A100-PCIE|P100" pkg/controller/example2.json` 无输出。

## 任务 07 从 AssignJobToNode 中提取纯 Job 变换 helper

- 目标: 把 “修改 Job 对象” 和 “调用 K8s Create” 拆开，为后续纯单测创造入口。
- 起点: `AssignJobToNode()` 同时做模板改写和真实提交。
- 终点: 存在 `PrepareJobForNode()` 或等价 helper，只负责把 Job 改造成待提交状态；`AssignJobToNode()` 只负责调用 helper 后提交。
- 修改范围: `pkg/controller/assignjob.go`。
- 验收: `go test ./pkg/controller -run '^$'` 能通过编译。

## 任务 08 在 PrepareJobForNode 中统一覆盖 ONNX 镜像

- 目标: 确保所有 ONNX 作业最终使用统一镜像。
- 起点: YAML 里的镜像可能还是旧的 `bencyq/infer`、`bencyq/yolo` 等。
- 终点: `PrepareJobForNode()` 对 ONNX 作业统一写入 `bencyq/nnmeter:202602122308`。
- 修改范围: `pkg/controller/assignjob.go`。
- 验收: 新增纯单元测试 `TestPrepareJobForNodeOverridesOnnxImage`，命令 `go test ./pkg/controller -run '^TestPrepareJobForNodeOverridesOnnxImage$'` 通过。

## 任务 09 在 PrepareJobForNode 中统一注入模型目录挂载

- 目标: 给所有 ONNX 作业注入统一 hostPath 挂载。
- 起点: controller 还没有模型目录的统一 Volume 和 VolumeMount。
- 终点: Job 模板内存在只读挂载，把宿主机 `/data/cyq/hase/model_zoo/models` 挂到容器同路径。
- 修改范围: `pkg/controller/assignjob.go`。
- 验收: 新增纯单元测试 `TestPrepareJobForNodeAddsModelZooMount`，命令 `go test ./pkg/controller -run '^TestPrepareJobForNodeAddsModelZooMount$'` 通过。

## 任务 10 在 PrepareJobForNode 中保留并校验脚本执行契约

- 目标: controller 不重写作业脚本逻辑，但要保证 `epoch`、`launch_script`、`model_path` 在模板中可用。
- 起点: 当前逻辑只会注入 `--epoch`，不校验脚本和模型路径。
- 终点: ONNX 作业进入提交前，至少能保证脚本路径、模型路径存在于 annotations 或容器 args/env 的预期位置；不满足时返回错误。
- 修改范围: `pkg/controller/assignjob.go`。
- 验收: 新增纯单元测试 `TestPrepareJobForNodeRejectsMissingScriptOrModelPath`，命令 `go test ./pkg/controller -run '^TestPrepareJobForNodeRejectsMissingScriptOrModelPath$'` 通过。

## 任务 11 为 PrepareJobForNode 建立纯单测基线

- 目标: 把 `AssignJob` 侧最关键的行为都用纯单测固定住。
- 起点: 现有 `assignjob_test.go` 会触达真实环境，不适合作为小步迭代的验收基础。
- 终点: 至少有 3 个纯单测覆盖镜像覆盖、模型目录挂载、现有 CUDA/MPS 注入不回归。
- 修改范围: `pkg/controller/assignjob_test.go`，必要时拆成新的纯测试文件。
- 验收: `go test ./pkg/controller -run '^TestPrepareJobForNode'` 通过。

## 任务 12 在 Go 侧引入预测器接口

- 目标: 为调度逻辑引入可替换的 predictor 抽象，便于后续 fake predictor 单测。
- 起点: `TotaltimePredict()` 和 `RandomForestPredict()` 与具体 Python 实现强耦合。
- 终点: 存在清晰的接口，例如 `PredictSingleEpochSeconds(ctx CandidateContext) (float64, error)`。
- 修改范围: `pkg/controller/config.go` 或新增 `pkg/controller/predictor_interface.go`。
- 验收: `go test ./pkg/controller -run '^$'` 能通过编译。

## 任务 13 在 Go 侧定义 predictor 请求上下文结构

- 目标: 把候选卡并行态打包成结构体，而不是在多个函数里散拼参数。
- 起点: 现有预测输入是 `jobModelNames + dc/cl/n/c`，无法表达 ONNX 场景。
- 终点: 存在清晰的 `PredictRequest`，至少包含 `model_name`、`model_path`、`epoch`、`gpu_type`、`node_name`、`card_id`、`parallel_jobs`、实时 GPU 指标。
- 修改范围: `pkg/controller/predictor_interface.go` 或等价文件。
- 验收: 新增纯单元测试 `TestBuildPredictRequest`，命令 `go test ./pkg/controller -run '^TestBuildPredictRequest$'` 通过。

## 任务 14 提供 Go 侧 fake predictor

- 目标: 为 `TotaltimePredict()` 和 `OptimalAllocate()` 的后续单测提供稳定假实现。
- 起点: 调度逻辑还没有可注入的 predictor 假对象。
- 终点: 测试里可以通过 fake predictor 精确控制某张卡返回多少 `single_epoch_sec`。
- 修改范围: `pkg/controller/` 下测试辅助文件。
- 验收: 新增纯单元测试 `TestFakePredictor`，命令 `go test ./pkg/controller -run '^TestFakePredictor$'` 通过。

## 任务 15 新建 Python predictor server 骨架

- 目标: 在 `cpn_controller/pkg/python` 下新增专用于 ONNX/HASE 的 server 骨架。
- 起点: 现有 `socket_server.py` 面向旧的多模型随机森林接口。
- 终点: 新 server 能接收请求、解析 JSON 或等价消息格式、返回固定正数 `single_epoch_sec`。
- 修改范围: `pkg/python/` 下新增新文件，不覆盖旧 server。
- 验收: 新增一个最小脚本自检命令，能返回固定正数并打印成功。

## 任务 16 新建 Go 侧 predictor client 封装

- 目标: 让 Go 侧能调用任务 15 的 server，但暂时不接入调度逻辑。
- 起点: 还没有针对新 predictor 的 client。
- 终点: 存在一个 client 封装，输入 `PredictRequest`，输出 `single_epoch_sec`。
- 修改范围: `pkg/controller/` 或 `pkg/python/` 下 Go client 文件。
- 验收: 新增纯单元测试 `TestPredictorClientDecodeResponse`，命令 `go test ./pkg/controller -run '^TestPredictorClientDecodeResponse$'` 通过。

## 任务 17 打通 client/server 最小回路

- 目标: 先证明 socket 链路和消息结构没问题，不要求真实预测。
- 起点: server 和 client 都存在，但还没有回路验证。
- 终点: 一个最小 smoke test 能完成 “Go 发请求 -> Python 回固定值”。
- 修改范围: `pkg/python/` 和 `pkg/controller/`，必要时新增小型脚本。
- 验收: 运行 smoke 命令可得到固定 `single_epoch_sec > 0`。

## 任务 18 在 Python predictor 中实现模型文件名解析

- 目标: 直接复用 HASE 的模型命名约定，从 `model_path` 解析 `model_name/batch/input_size`。
- 起点: predictor 还不知道如何从 ONNX 文件名得到运行时维度。
- 终点: 存在纯函数，能解析 `<model>_bs<batch>_<size>x<size>.onnx`，错误输入会显式报错。
- 修改范围: `pkg/python/onnx_predictor_server.py` 或拆出的公共模块。
- 验收: 新增 Python 纯单测或最小脚本自检，对合法和非法文件名分别得到正确结果。

## 任务 19 在 Python predictor 中实现 DAG 定位与自动生成

- 目标: 优先读 HASE 已有 DAG，缺失时再从 kernel JSON 自动生成。
- 起点: predictor 还不能定位 `graph_model/model_DAG` 和 `ort_analysis/ort_kernel_record`。
- 终点: 输入 `model_name/model_path` 后，能稳定拿到 DAG JSON 路径。
- 修改范围: `pkg/python/onnx_predictor_server.py`。
- 验收: 对至少一个现有模型，命令行自检能输出有效 DAG 路径。

## 任务 20 在 Python predictor 中实现 runtime kernel rows 构建

- 目标: 从 HASE 的 DAG 和 kernel JSON 构建 runtime kernel 特征行。
- 起点: predictor 还不能把模型展开为逐 kernel 的特征。
- 终点: 至少能为一个已有模型生成非空 kernel rows，包含 `OpType/kernel_group/batch/channel/height/width/kernel_h/kernel_w`。
- 修改范围: `pkg/python/onnx_predictor_server.py`。
- 验收: 对至少一个现有模型，命令行自检打印出非空 kernel rows 数量。

## 任务 21 在 Python predictor 中实现“可用优先”的 fallback regressor

- 目标: 在没有正式权重时也能稳定返回正数，不阻塞整体流程。
- 起点: predictor 还没有回归器 fallback 机制。
- 终点: 先尝试加载默认权重；失败时回退到固定随机种子的伪回归器或常数回归器，保证同输入输出稳定。
- 修改范围: `pkg/python/onnx_predictor_server.py`。
- 验收: 删除或不提供权重文件时，predictor 仍能返回 `single_epoch_sec > 0`。

## 任务 22 在 Python predictor 中实现逐 kernel 累积得到单 epoch 秒数

- 目标: 把每个 kernel 的预测值累计成新作业的 `single_epoch_sec`。
- 起点: predictor 只有零散的特征和回归器，不返回整模型时长。
- 终点: predictor 对一个 ONNX 模型请求返回单 epoch 秒数，并且单位是秒。
- 修改范围: `pkg/python/onnx_predictor_server.py`。
- 验收: 对至少一个现有 ONNX 模型，predictor 返回一个稳定正数秒值。

## 任务 23 把 TotaltimePredict 改成 “调用 predictor 一次再乘 epoch”

- 目标: 用新 predictor 完全替代旧的多作业离散仿真逻辑。
- 起点: `TotaltimePredict()` 仍在旧逻辑里循环模拟多作业完成顺序。
- 终点: `TotaltimePredict()` 只负责组请求、调 predictor、计算 `single_epoch_sec * epoch`、返回秒值。
- 修改范围: `pkg/controller/predictor.go`。
- 验收: 新增纯单元测试 `TestTotaltimePredictUsesSingleEpochTimesEpoch`，命令 `go test ./pkg/controller -run '^TestTotaltimePredictUsesSingleEpochTimesEpoch$'` 通过。

## 任务 24 让 ReserveAllocate 与新时间语义保持一致

- 目标: 消除 reserve 分支里旧的 `RandomForestPredict()*Epoch` 逻辑。
- 起点: `ReserveAllocate()` 仍使用旧 predictor 入口和旧时间语义。
- 终点: reserve 分支要么改用同一个 predictor helper，要么明确短路禁用，并且代码注释写清楚原因。
- 修改范围: `pkg/controller/optimal_allocation.go`。
- 验收: `go test ./pkg/controller -run '^$'` 能通过编译；若保留 reserve，则新增 `TestReserveAllocateUsesPredictor`。

## 任务 25 移除 “同卡最多 3 作业” 限制并保留并行卡候选资格

- 目标: 让忙卡仍可参与预测比较，只做 CPU 和显存粗筛。
- 起点: `OptimalAllocate()` 中存在 `len(cardInfo.JobQueue) >= 3` 的硬限制。
- 终点: 候选卡只因 CPU、节点内存、显存、显式禁用条件被过滤，不因已有并行作业数量直接出局。
- 修改范围: `pkg/controller/optimal_allocation.go`。
- 验收: 新增纯单元测试 `TestOptimalAllocateKeepsBusyCardsAsCandidates`，命令 `go test ./pkg/controller -run '^TestOptimalAllocateKeepsBusyCardsAsCandidates$'` 通过。

## 任务 26 把显存粗筛切换到 YAML 提供的 ONNX 显存需求

- 目标: 用作业 YAML 中的 `gpu_memory_req_mb` 作为 OOM 粗筛依据。
- 起点: 显存需求仍主要来自旧 baseline 逻辑。
- 终点: `OptimalAllocate()` 的显存判断直接依赖新 `Job.GPUMemoryReq`，并保留固定安全边际。
- 修改范围: `pkg/controller/optimal_allocation.go`，必要时涉及 `JobAnalyze()`。
- 验收: 新增纯单元测试 `TestOptimalAllocateRejectsGpuOomCandidate`，命令 `go test ./pkg/controller -run '^TestOptimalAllocateRejectsGpuOomCandidate$'` 通过。

## 任务 27 为 OptimalAllocate 增加“选最小总完成时间”单测

- 目标: 用 fake predictor 锁住核心选址行为。
- 起点: 还没有纯单测验证 “两张忙卡都可选时选预测总时间最小者”。
- 终点: 单测中可构造两张卡都有并行任务，但新作业最终稳定选到总时间更小的卡。
- 修改范围: `pkg/controller/optimal_allocation_test.go` 或新测试文件。
- 验收: `go test ./pkg/controller -run '^TestOptimalAllocateChoosesFastestBusyCard$'` 通过。

## 任务 28 新增一个规范的 ONNX Job YAML 模板

- 目标: 提供最小可运行模板，作为后续队列迁移的标准输入。
- 起点: `pkg/yaml_template` 仍是旧 PyTorch/YOLO/LLM 模板。
- 终点: 存在一个新的 ONNX 模板样例，字段齐全，镜像和模型路径符合新契约。
- 修改范围: `pkg/yaml_template/` 下新增或改造一个模板文件。
- 验收: 该 YAML 能被任务 03 和任务 04 的解析/校验逻辑接受。

## 任务 29 编写 legacy YAML -> ONNX YAML 的迁移脚本

- 目标: 先做自动化迁移工具，再批量改 `yaml_queue`。
- 起点: `pkg/controller/yaml_queue` 里仍是旧镜像、旧 annotations、旧脚本路径。
- 终点: 存在一个迁移脚本，能根据 `model_name` 生成 `model_path/container_image/launch_script/gpu_memory_req_mb` 等新字段。
- 修改范围: `pkg/utils/` 或 `scripts/` 下新增迁移脚本。
- 验收: 对单个样例 YAML 运行迁移后，输出符合新契约。

## 任务 30 对 yaml_queue 执行批量迁移

- 目标: 让 controller 真实读取到的新队列就是 ONNX 作业。
- 起点: 队列目录还是旧格式。
- 终点: `pkg/controller/yaml_queue` 中所有投递文件都满足新 ONNX 契约，且不再使用旧镜像。
- 修改范围: `pkg/controller/yaml_queue/`。
- 验收: `rg -n "bencyq/infer|bencyq/yolo|bencyq/llama3|bencyq/qwen2.5|bencyq/glm4" pkg/controller/yaml_queue` 无输出。

## 任务 31 为队列迁移结果增加抽样校验

- 目标: 不是只看批量替换结果，还要抽样检查字段完整性。
- 起点: YAML 批量迁移后还未验证每个文件都包含关键字段。
- 终点: 至少抽查 3 个队列文件，确认有 `model_name/model_path/epoch/gpu_memory_req_mb/container_image/launch_script`。
- 修改范围: 可新增一个小型检查脚本或纯测试。
- 验收: 抽样检查命令输出全部通过。

## 任务 32 给 controller 主链路接入新 predictor

- 目标: 把 `main.go` 指向的新 predictor 初始化链路真正接上。
- 起点: 新 predictor 还只是局部能力，没有进主流程。
- 终点: `cmd/controller/main.go` 启动后，controller 使用新 predictor 完成调度，不再依赖旧随机森林 socket。
- 修改范围: `cmd/controller/main.go`，`pkg/controller/predictor.go`。
- 验收: 在不连接真实集群提交的前提下，启动日志能走到 predictor 初始化成功和调度入口。

## 任务 33 增加一个“只算不提”的调度 smoke 模式

- 目标: 给主 Agent 一个安全验收入口，不触达真实集群创建 Job。
- 起点: 现有主流程默认会走到 K8s Create。
- 终点: 存在一个 dry-run 开关或独立 helper，可读取 `example2.json` 和 `yaml_queue`，完成候选卡比较并打印选址结果，但不提交。
- 修改范围: `cmd/controller/` 或 `pkg/controller/assignjob.go`。
- 验收: 运行 dry-run 命令后，能输出每个新作业的候选卡预测秒数和最终选址。

## 任务 34 对主流程做最终定向验收

- 目标: 在不触达真实训练和真实集群提交的前提下，完成这一轮改造的最终确认。
- 起点: 各子任务已完成，但还没有统一收口。
- 终点: 下列检查全部通过。
- 修改范围: 无新增功能，主要是运行验收命令和整理结果。
- 验收: `go test ./pkg/controller -run 'TestParseOnnxJobAnnotations|TestValidateOnnxJobContract|TestPrepareJobForNode|TestTotaltimePredictUsesSingleEpochTimesEpoch|TestOptimalAllocate'` 通过。
- 验收: predictor smoke 命令对至少一个模型返回正数秒值。
- 验收: `example2.json` 中 GPU 节点只反映 `node191` 和 `node200` 的当前环境。
- 验收: `yaml_queue` 中不再出现旧镜像。

## 主 Agent 交接要求

- 子 Agent 每完成一个任务，必须只汇报该任务的改动、测试结果、风险点。
- 主 Agent 在推进下一任务前，必须确认上一任务的验收命令已经通过。
- 如果某任务需要扩大范围，主 Agent 不能让子 Agent 自行扩张，必须先重写后续计划再继续。
