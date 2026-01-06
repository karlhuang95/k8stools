# 🧰K8stools 使用说明文档

版本：v1.0  
更新时间：2025-05-08  
作者：karl

# 📌简介

K8stools 是一个 Kubernetes 日常运维辅助工具集，旨在提升运维效率，辅助平台治理与资源优化。功能涵盖资源分析、趋势评估、异常检测、行为采集、成本估算等常见场景，适用于 DevOps、SRE、平台工程团队。

# 🎯功能模块

| 模块名称 | 功能说明 |
|------|------|
|costEstimator|成本估算|
|cpu|获取k8s的cpu使用情况|
|paradise|k8s理想情况分配|
|poderrors|异常检查|
|runtimeInspect|采集运行中的 Pod 容器行为信息（进程、端口、环境变量）|
|trend|基于 Prometheus 的资源使用趋势分析与建议|

# 📄 配置文件说明（config.yaml）

```yaml
kubeconfig: /root/.kube/config
namespace:
  - {namespace}-prod
prometheus: http://prom.example.net

cost:
  cpuPrice: 4000   # 单台机器价格（单位元）
  totalCpu: 16     # 单台机器 CPU 核数
```

# 🧠 各子工具设计逻辑

---

# 📊 paradise - 理想资源建议工具

**CPU 建议规则：**

| 类型     | 设计逻辑                                                             |
|----------|----------------------------------------------------------------------|
| Requests | 当前使用量的 50%~80%，保证调度时有资源可用                         |
| Limits   | 当前使用量的 100%~150%，防止异常飙高占满节点                        |
| 特殊处理 | 使用低于 50m 的容器 → 默认给 50m，避免调度失败；高于 1000m 提示设置上限 |

**内存建议规则：**

| 类型     | 设计逻辑                                                             |
|----------|----------------------------------------------------------------------|
| Requests | 当前使用量的 70%~100%，确保稳定调度                                 |
| Limits   | 当前使用量的 150%~200%，保留 buffer 防止 OOM                        |
| 特殊处理 | 对 sidecar / agent 等轻量容器，给最小起点值如 64Mi                  |

---

# 📈 trend - 资源趋势分析工具

**数据来源：Prometheus 查询**

| 指标 | 查询方式 |
|------|----------|
| 平均 CPU 使用量 | `avg_over_time(container_cpu_usage_seconds_total[1w])`（单位：m） |
| 最大 CPU 使用量 | `max_over_time(container_cpu_usage_seconds_total[1w])`（单位：m） |
| 平均/最大内存使用量 | 类似用 `container_memory_usage_bytes` 查询，并换算成 MiB |

**推荐策略（保守 & 稳健）**

| 类型             | 推荐计算公式                         |
|------------------|--------------------------------------|
| CPU Requests     | `ceil(平均 CPU 使用量 × 1.2)`        |
| CPU Limits       | `ceil(最大 CPU 使用量 × 1.5)`        |
| 内存 Requests    | `ceil(平均内存使用量 × 1.2)`         |
| 内存 Limits      | `ceil(最大内存使用量 × 1.5)`         |

---

# 🚨 poderrors - 异常 Pod 检查工具

**功能说明：**

- 遍历所有或指定命名空间下的 Pod
- 检查所有处于异常状态的容器（如 CrashLoop、ImagePullBackOff、OOMKilled 等）
- 输出字段：

| Namespace | Pod     | Container | Reason            | Message                                | Restart Count | Age |
|-----------|---------|-----------|-------------------|----------------------------------------|---------------|-----|
| default   | api-xxx | app       | CrashLoopBackOff  | Back-off restarting failed container   | 5             | 3m  |

---

# 📊 resourceAdvisor - Kubernetes 服务资源顾问工具

**数据来源：Traefik + Prometheus 指标**

| 指标             | 查询方式                         |
|------------------|--------------------------------------|
|加权 RPS（每秒请求数）| sum by(method)(rate(traefik_service_requests_total{namespace="traefik", exported_service="<服务名>"}[1m])) × 方法权重|
|P95 延迟（ms）| histogram_quantile(0.95, sum by(le)(rate(traefik_service_request_duration_seconds_bucket{namespace="traefik", exported_service="<服务名>"}[5m]))) * 1000|
|请求/限制 CPU| 固定值或根据 RPS 计算（m）|
|请求/限制内存|固定值或根据 RPS 计算（Mi）|

**决策逻辑**

| 决策类型             | 条件说明                         |
|------------------|--------------------------------------|
|SCALE_OUT|加权 RPS 或 P95 延迟偏高，需要扩容 Pod|
|DOWNSIZE_SAFE|流量低且 CPU/内存长期低利用率，可安全缩容|
|NO_TRAFFIC|未观测到业务流量，无需操作|
|KEEP|指标正常，保持现有副本数|

**输出字段说明（中文表头）**

| 字段             | 含义                         |
|------------------|--------------------------------------|
|命名空间|	Kubernetes Namespace|
|服务	|exported_service 名称|
|RPS（加权）|	method 加权后的每秒请求数|
|P95延迟（ms）|	请求延迟的 95 百分位数|
|请求CPU（m）|	Pod 请求 CPU|
|限制CPU（m）|	Pod 限制 CPU|
|请求内存（Mi）|	Pod 请求内存|
|限制内存（Mi）|	Pod 限制内存|
|最小副本数|	最小副本数（minReplicas）|
|推荐副本数|	根据指标计算的推荐副本数|
|决策	|SCALE_OUT / DOWNSIZE_SAFE / NO_TRAFFIC / KEEP|
|风险等级|	LOW / MEDIUM / HIGH|
|置信度	|LOW / MEDIUM / HIGH|
|原因	|决策依据|
|指标窗口|	用于统计 RPS / 延迟的时间窗口|
|生成时间|	CSV 生成时间|

**方法加权说明**

| 方法             | 权重                        |
|------------------|-------------------------------------|
|GET	|1|
|POST	|2|
|DELETE|2|

使用说明
1.	配置 Prometheus 地址与命名空间列表。
2.	调用 ResourceAdvisor 接口生成 CSV 文件。
3.	CSV 文件可用于 Pod 资源调整或 HPA 决策参考。

# 🔍 runtimeInspect - 容器行为采集工具

**功能说明：**

- 采集运行中 Pod 的详细信息，包括：
  - 容器内进程列表
  - 监听端口信息
  - 环境变量

**使用场景：**
- 排查线上故障时快速查看容器内部运行情况
- 无需进入容器即可采集运行行为（非入侵式）

---

# 💰 costEstimator - 成本估算工具

**计算模型：**

- 基于你提供的每台机器：
  - CPU 核数（如：16）
  - 单价（如：4000元）

**计算逻辑：**

| 步骤             | 说明                                                             |
|------------------|------------------------------------------------------------------|
| 每核价格         | `单价 / 总 CPU 数`                                               |
| 容器请求费用     | `CPU Request (m) × 每毫核价格`                                   |
| 每月总费用       | `容器费用 × 24 × 30`（按 30 天、全天运行估算）                  |

**输出字段：**

| Namespace | Pod   | Container | CPU Request (m) | CPU Cost (元) | Total Cost (元/月) |
|-----------|-------|-----------|-----------------|----------------|---------------------|

---

# 📦 示例命令

```bash
k8stools paradise        -f config.yaml   # 理想资源建议
k8stools trend           -f config.yaml   # 资源趋势分析
k8stools poderrors       -f config.yaml   # 异常 Pod 检查
k8stools runtimeInspect  -f config.yaml   # 容器运行时行为采集
k8stools costEstimator   -f config.yaml   # 成本估算
```

```bash
# 计算资源使用情况的理想配置建议
k8stools paradise -f config.yaml

# 基于 Prometheus 历史数据的趋势分析
k8stools trend -f config.yaml

# 检查所有命名空间下异常状态的 Pod
k8stools poderrors -f config.yaml

# 查看容器运行时信息（进程、端口、环境变量）
k8stools runtimeInspect -f config.yaml

# 根据配置中机器单价和总 CPU 数进行成本估算
k8stools costEstimator -f config.yaml
```
---

# 📁 建议输出目录结构

统一将输出放到 `output/` 目录，并添加时间戳，便于追溯与比较：

```
output/
├── cpu_info_2025-04-21.csv
├── cost_estimate_2025-04-21.csv
└── resource_trend_2025-04-21.csv
```

---

# ✨ TODO（建议）

- [ ] 增加 `diagnose` 一键巡检工具
- [ ] 支持 JSON/table 等输出格式
---

# 📬 联系与反馈

如有建议或需求，欢迎反馈或提交 PR，一起打磨出更适合生产的 K8s 工具链！




