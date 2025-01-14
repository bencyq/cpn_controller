# cpn controller version1 设计思路1 C/S架构 暂时废弃
## 客户端方面（即被控的k8s集群）

1. 给集群安装一个广域调度器的**本地节点**，负责发送prometheus收集到的资源信息，接受广域调度器发送的pod创建配置，并向kube apiserver发送请求
2. ~~使用自定义的 plugin `cpnscheduling` ，屏蔽掉除**Bind环节**以外的所有环节，实现接口`framework.BindPlugin` ，直接按照接收的pod创建配置，将pod调度到配置中选择的节点上~~
    
    ~~或者可以使用extender直接重写整个调度器逻辑~~
    
    在job的配置文件，如下指定节点名字以及卡的编号
    
    ```
    nodeSelector:
      kubernetes.io/hostname: <hostname>
    ……
    env:
      - name: NVIDIA_VISIBLE_DEVICES
        value: "0"
    ```
    

# 实现步骤

- [x]  完成自定义的 plugin `cpnscheduling`的安装，能直接按照Job、Deployment等的配置文件中指定的node节点调度pod
- [ ]  在集群中安装一个广域调度器的本地节点，能实现发送prometheus信息以及接收pod配置文件，并向k8s apiserver提交请求，使用client-go编写
- [ ]  使用go实现一个广域调度器，能依据固定的作业池，综合考虑作业之间的亲和性以及工作节点之间的宽带与时延，安排最合理的作业并行策略

# ~~自定义的 plugin `cpnscheduling` 设计流程~~

1. 克隆 scheduler plugin 仓库https://github.com/kubernetes-sigs/scheduler-plugins，并新建自己的plugin文件
2. plugin实现`framework.BindPlugin`接口
3. plugin从配置文件中读取到选择的node节点信息，以及卡号编码

# 基于client-go编写的调度器本地节点controller

## part1 接收指令新建Job

## part2 收集信息发送给广域调度器
