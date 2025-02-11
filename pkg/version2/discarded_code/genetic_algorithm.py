import random
from deap import base, creator, tools, algorithms

# 问题参数
ClusterNums = 3  # 集群数量
NodeNums = [2, 1, 2]  # 每个集群的节点数量
CPUBaseline = [100, 110, 90]  # 自定义Baseline，以xxx型号为基准做基准测试 TODO:
MemAvailable = [512, 512, 1024]  # 单位为GB
CardNums = [[4, 2], [4], [2, 2]]  # 每个节点的 GPU 卡数量
CardBaseline = [[[0, 1, 2, 0], [1, 2]],  # 每个 GPU 卡的性能分数 TODO:
     [[1, 2, 0], [2, 0], [1, 2]],
     [[0, 1], [2, 0]]]
CardMem = [[[40, 40, 40, 40], [24, 24]], 
     [[48, 48, 48, 48]],
     [[40, 40], [40, 40]]]  # 每个GPU卡的显存大小
jobs = [
    {"type": "CPU", "data_size": 100, "cpu": 2, "memory": 4, "gpu": 1, "vram": 2, "run_time": 10},
    {"type": "GPU", "data_size": 200, "cpu": 1, "memory": 8, "gpu": 1, "vram": 4, "run_time": 20},
    # 添加更多作业...
]
bandwidth = [100, 200, 50]  # 调度器与每个集群的带宽，以MB/s为单位

# 定义个体和适应度
creator.create("FitnessMin", base.Fitness, weights=(-1.0,))  # 最小化目标
creator.create("Individual", list, fitness=creator.FitnessMin)

# 初始化工具
toolbox = base.Toolbox()

# 定义个体生成函数
def create_individual():
    individual = []
    for _ in range(len(jobs)):
        cluster = random.randint(0, ClusterNums - 1)
        node = random.randint(0, NodeNums[cluster] - 1)
        card = random.randint(0, CardNums[cluster][node] - 1)
        individual.append((cluster, node, card))  # 分配到集群、节点、GPU 卡
    order = list(range(len(jobs)))
    random.shuffle(order)
    return individual+order

toolbox.register("individual", tools.initIterate, creator.Individual, create_individual)
toolbox.register("population", tools.initRepeat, list, toolbox.individual)

# 定义适应度函数
def evaluate(individual):
    total_time = 0
    gpu_usage = {}  # 记录每个 GPU 卡上的作业

    for job_idx, (cluster, node, card) in enumerate(individual):
        job = jobs[job_idx]
        transfer_time = job["data_size"] / bandwidth[cluster]

        # 检查显存约束
        vram_used = sum(jobs[k]["vram"] for k, (c, n, g) in enumerate(individual) 
                        if (c, n, g) == (cluster, node, card))
        if vram_used > CardMem[cluster][node][card]: 
            return (1e9,)

        # 多个 GPU 密集型作业不能并行运行
        if job["type"] == "GPU":
            if any(jobs[k]["type"] == "GPU" for k, (c, n, g) in enumerate(individual) if (c, n, g) == (cluster, node, card) and k != job_idx):
                return (1e9,) 
            
        # 检查 CPU 资源剩余 TODO:
        
        # 检查 内存 资源剩余 TODO:

        # 计算运行时间 TODO:
        cardBaseline = CardBaseline[cluster][node][card]
        # run_time = TODO:
        total_time += transfer_time + run_time

    return (total_time,)

toolbox.register("evaluate", evaluate)
toolbox.register("mate", tools.cxTwoPoint)  # 两点交叉
toolbox.register("mutate", tools.mutUniformInt, low=0, up=sum(NodeNums)-1, indpb=0.1)  # 均匀变异
toolbox.register("select", tools.selTournament, tournsize=3)  # 锦标赛选择

# 遗传算法参数
population_size = 50
num_generations = 100
crossover_prob = 0.7
mutation_prob = 0.2

# 初始化种群
population = toolbox.population(n=population_size)

# 运行遗传算法
algorithms.eaSimple(
    population,
    toolbox,
    cxpb=crossover_prob,
    mutpb=mutation_prob,
    ngen=num_generations,
    verbose=True,
)

# 输出最优解
best_individual = tools.selBest(population, k=1)[0]
print("Best individual:", best_individual)      
print("Best fitness:", evaluate(best_individual))   