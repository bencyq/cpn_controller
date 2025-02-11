import random
from typing import List, Dict, Tuple
import numpy as np

class GPU:
    def __init__(self, gpu_id):
        self.gpu_id = gpu_id
        self.timeline = []  # list of (start, end, jobs)

class Node:
    def __init__(self, node_id, cpu_capacity):
        self.node_id = node_id
        self.cpu_capacity = cpu_capacity  # 70%
        self.gpus = []  # list of GPU objects
        self.cpu_timeline = []  # list of (start, end, usage)

class Cluster:
    def __init__(self, cluster_id, bandwidth):
        self.cluster_id = cluster_id
        self.bandwidth = bandwidth
        self.nodes = []

class Job:
    def __init__(self, job_id, job_type, data_size, cpu_demand, single_time):
        self.job_id = job_id
        self.job_type = job_type  # 'cpu' or 'gpu'
        self.data_size = data_size
        self.cpu_demand = cpu_demand
        self.single_time = single_time  # time when run alone

# Example setup
num_clusters = 2
clusters = []
for c in range(num_clusters):
    cluster = Cluster(c, bandwidth=100 + 50*c)  # Varying bandwidth
    num_nodes = 3
    for n in range(num_nodes):
        node = Node(n, cpu_capacity=70)
        num_gpus = 4
        for g in range(num_gpus):
            node.gpus.append(GPU(g))
        cluster.nodes.append(node)
    clusters.append(cluster)

# Example jobs
jobs = [
    Job(0, 'cpu', 500, 30, 10),
    Job(1, 'gpu', 1000, 10, 20),
    Job(2, 'cpu', 300, 25, 15),
    Job(3, 'gpu', 800, 15, 25),
    Job(4, 'cpu', 400, 20, 12)
]

# List of all possible GPUs (cluster_id, node_id, gpu_id)
all_gpus = []
for cluster in clusters:
    for node in cluster.nodes:
        for gpu in node.gpus:
            all_gpus.append( (cluster.cluster_id, node.node_id, gpu.gpu_id) )
num_gpus = len(all_gpus)

def create_individual():
    return [random.randint(0, num_gpus - 1) for _ in jobs]

def calculate_fitness(individual):
    node_cpu_timelines = { (c.cluster_id, n.node_id): [] for c in clusters for n in c.nodes }
    gpu_timelines = { (c.cluster_id, n.node_id, g.gpu_id): [] for c in clusters for n in c.nodes for g in n.gpus }
    transmission_end = 0  # Track cumulative transmission time

    for job_idx, job in enumerate(jobs):
        gpu_idx = individual[job_idx]
        cluster_id, node_id, gpu_id = all_gpus[gpu_idx]
        cluster = clusters[cluster_id]
        node = next(n for n in cluster.nodes if n.node_id == node_id)
        gpu = node.gpus[gpu_id]

        # Calculate transmission time
        transmission_time = job.data_size / cluster.bandwidth
        transmission_end += transmission_time

        # Find available time slot on GPU
        timeline = gpu_timelines[(cluster_id, node_id, gpu_id)]
        start_time = transmission_end
        if timeline:
            start_time = max(start_time, timeline[-1][1])

        # Check GPU parallel constraints
        overlapping = [slot for slot in timeline if slot[0] < start_time < slot[1]]
        current_gpu_jobs = [j for slot in overlapping for j in slot[2]]
        gpu_count = sum(1 for j in current_gpu_jobs if j.job_type == 'gpu')
        cpu_count = sum(1 for j in current_gpu_jobs if j.job_type == 'cpu')

        if job.job_type == 'gpu':
            if gpu_count >= 1:
                start_time = max(start_time, max(slot[1] for slot in overlapping))
        if len(current_gpu_jobs) >= 3:
            start_time = max(start_time, max(slot[1] for slot in overlapping))

        # Adjust for overlapping after start_time
        new_overlapping = [slot for slot in timeline if slot[0] < start_time < slot[1]]
        while new_overlapping:
            slot = new_overlapping[0]
            start_time = slot[1]
            new_overlapping = [slot for slot in timeline if slot[0] < start_time < slot[1]]

        # Calculate runtime based on parallel jobs
        overlapping_during_run = [slot for slot in timeline if slot[0] < start_time + job.single_time and slot[1] > start_time]
        parallel_jobs = [j for slot in overlapping_during_run for j in slot[2]]
        parallel_cpu = sum(1 for j in parallel_jobs if j.job_type == 'cpu')
        parallel_gpu = sum(1 for j in parallel_jobs if j.job_type == 'gpu')
        if job.job_type == 'cpu':
            runtime = job.single_time / (parallel_cpu + 1)
        else:
            runtime = job.single_time * (parallel_gpu + 1) / (parallel_cpu + 1)
        end_time = start_time + runtime

        # Check CPU usage
        cpu_overlap = []
        for period in node_cpu_timelines[(cluster_id, node_id)]:
            if period[0] < end_time and period[1] > start_time:
                cpu_overlap.append(period)
        current_usage = sum(period[2] for period in cpu_overlap)
        if current_usage + job.cpu_demand > node.cpu_capacity:
            return float('inf')

        # Update GPU timeline
        gpu_timelines[(cluster_id, node_id, gpu_id)].append( (start_time, end_time, parallel_jobs + [job]) )
        # Update CPU timeline
        node_cpu_timelines[(cluster_id, node_id)].append( (start_time, end_time, job.cpu_demand) )

    # Calculate makespan
    makespan = 0
    for timeline in gpu_timelines.values():
        if timeline:
            makespan = max(makespan, max(slot[1] for slot in timeline))
    return makespan

def tournament_selection(population, fitness, tournament_size=3):
    selected = []
    for _ in range(len(population)):
        contestants = random.sample(range(len(population)), tournament_size)
        winner = min(contestants, key=lambda x: fitness[x])
        selected.append(population[winner])
    return selected

def crossover(parent1, parent2):
    point = random.randint(1, len(parent1) - 1)
    child1 = parent1[:point] + parent2[point:]
    child2 = parent2[:point] + parent1[point:]
    return child1, child2

def mutate(individual, mutation_rate=0.1):
    for i in range(len(individual)):
        if random.random() < mutation_rate:
            individual[i] = random.randint(0, num_gpus - 1)
    return individual

population_size = 50
generations = 100
population = [create_individual() for _ in range(population_size)]

for generation in range(generations):
    fitness = [calculate_fitness(ind) for ind in population]
    selected = tournament_selection(population, fitness)
    next_population = []
    for i in range(0, population_size, 2):
        parent1 = selected[i]
        parent2 = selected[i+1] if i+1 < len(selected) else selected[i]
        child1, child2 = crossover(parent1, parent2)
        next_population.extend([mutate(child1), mutate(child2)])
    population = next_population[:population_size]

best_idx = np.argmin([calculate_fitness(ind) for ind in population])
best_individual = population[best_idx]
print(f"Best makespan: {calculate_fitness(best_individual)}")