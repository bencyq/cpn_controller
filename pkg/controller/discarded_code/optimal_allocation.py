# 作业队列按照FIFO顺序排列
# 对于一个作业，模拟其在各个集群运行的完成时间，并进行分配

class Monitor:
    def __init__(self, DataCenterNums=0, DataCenterInfo=None, JobPool=None):
        self.DataCenterNums = DataCenterNums
        self.DataCenterInfo = DataCenterInfo if DataCenterInfo is not None else []
        self.JobPool = JobPool

class DataCenterInfo:
    def __init__(self, DataCenterID="", DataCenterLocation="", ClusterNums=0, ClusterInfo=None):
        self.DataCenterID = DataCenterID
        self.DataCenterLocation = DataCenterLocation
        self.ClusterNums = ClusterNums
        self.ClusterInfo = ClusterInfo if ClusterInfo is not None else []

class ClusterInfo:
    def __init__(self, ClusterID="", ClusterIP="", NodeNums=0, NodeInfo=None, ClusterPromIpPort="",
                 ClusterKubeconfigFilePath="", ClusterClientSet=None):
        self.ClusterID = ClusterID
        self.ClusterIP = ClusterIP
        self.NodeNums = NodeNums
        self.NodeInfo = NodeInfo if NodeInfo is not None else []
        self.ClusterPromIpPort = ClusterPromIpPort
        self.ClusterKubeconfigFilePath = ClusterKubeconfigFilePath
        self.ClusterClientSet = ClusterClientSet

class NodeInfo:
    def __init__(self, NodeID="", NodeIP="", CPUInfo=None, NodeType="", CardNums=0, CardInfo=None,
                 CPU_USAGE=0.0, TOTAL_MEMORY=0, FREE_MEMORY=0, BenchMark=None):
        self.NodeID = NodeID
        self.NodeIP = NodeIP
        self.CPUInfo = CPUInfo if CPUInfo is not None else CPUInfo()
        self.NodeType = NodeType
        self.CardNums = CardNums
        self.CardInfo = CardInfo if CardInfo is not None else []
        self.CPU_USAGE = CPU_USAGE
        self.TOTAL_MEMORY = TOTAL_MEMORY
        self.FREE_MEMORY = FREE_MEMORY
        self.BenchMark = BenchMark

    def FindCard(self, cardID):
        for card in self.CardInfo:
            if card.CardID == cardID:
                return card
        return None

class CPUInfo:
    def __init__(self, CPUNums=0, Architecture="", CPUModel="", CPUCore=0):
        self.CPUNums = CPUNums
        self.Architecture = Architecture
        self.CPUModel = CPUModel
        self.CPUCore = CPUCore

class CardInfo:
    def __init__(self, CardID="", CardModel="", GPU_UTIL=0, GPU_MEMORY_FREE=0, GPU_MEMORY_USED=0):
        self.CardID = CardID
        self.CardModel = CardModel
        self.GPU_UTIL = GPU_UTIL
        self.GPU_MEMORY_FREE = GPU_MEMORY_FREE
        self.GPU_MEMORY_USED = GPU_MEMORY_USED

class Card:
    def __init__(self, ID=0, CARDMODEL="", GPU_UTIL=0.0, GPU_MEMORY_FREE=0, GPU_MEMORY_USED=0):
        self.ID = ID
        self.CARDMODEL = CARDMODEL
        self.GPU_UTIL = GPU_UTIL
        self.GPU_MEMORY_FREE = GPU_MEMORY_FREE
        self.GPU_MEMORY_USED = GPU_MEMORY_USED
