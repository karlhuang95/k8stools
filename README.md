# K8stools

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://golang.org/)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
[![GitHub](https://img.shields.io/badge/GitHub-k8stools-purple)](https://github.com/karl/k8stools)

> Kubernetes 日常运维辅助工具集 - 数据驱动的资源优化平台

---

## 📋 目录

- [简介](#简介)
- [核心特性](#核心特性)
- [功能模块](#功能模块)
- [快速开始](#快速开始)
- [详细文档](#详细文档)
- [架构图](#架构图)
- [贡献指南](#贡献指南)
- [许可证](#许可证)

---

## 简介

**K8stools** 是一个专为 Kubernetes 运维设计的智能化工具集，通过整合 Metrics Server、Prometheus、Traefik 等监控数据源，提供数据驱动的资源分析与优化建议。

**适用场景：**
- 🔍 **资源分析** - 深入了解集群资源使用状况
- 📈 **趋势预测** - 基于历史数据的资源增长预测
- 💰 **成本优化** - 精确计算资源成本，优化预算
- 🚀 **性能调优** - 基于流量指标的服务配置建议
- 🛡️ **运行时监控** - 非入侵式容器行为采集

---

## 核心特性

| 特性 | 说明 |
|------|------|
| 📊 **数据驱动** | 基于 Metrics Server、Prometheus、Traefik 等真实指标 |
| 🧠 **智能分析** | 线性回归、多级决策模型等算法提供精准建议 |
| 🔒 **非入侵式** | 只读采集数据，不影响生产环境运行 |
| 🎯 **模块化设计** | 各模块独立运行，易于扩展和集成 |
| 📝 **多格式输出** | CSV 格式输出，便于进一步处理和分析 |

---

## 功能模块

### 📊 CPU 使用情况统计

统计 Deployment 级别的 CPU 使用情况，区分主容器和 Sidecar 容器，用于资源评估和优化分析。

**数据来源：** Kubernetes Metrics Server + Kubernetes API

**输出信息：**
- CPU 实际使用量（主容器 vs Sidecar）
- CPU Requests/Limits 配置
- HPA 副本数范围

```bash
./k8stools cpu -f config.yaml
```

---

### 🔍 容器运行时行为采集

非入侵式采集运行中 Pod 容器的详细运行信息，用于故障排查和运行时分析。

**采集内容：**
- 📋 进程列表 (`ps aux`)
- 🔌 监听端口 (`ss -tulnp`)
- 🌍 环境变量 (`printenv`)

**安全机制：**
- ✅ 命令白名单（只允许预定义的安全命令）
- ⚡ 并发控制（最多 5 个容器同时执行）
- ⏱️ 超时保护（30 秒超时）

```bash
./k8stools runtimeInspect -f config.yaml
```

---

### 📈 资源趋势分析

基于 Prometheus 历史数据（默认 7 天）分析容器资源使用趋势，并给出资源配置建议。

**核心算法：**
- **线性回归** - 分析资源使用趋势（上升/下降/稳定）
- **趋势系数调整** - 上升趋势 ×1.1，下降趋势 ×0.9
- **推荐值计算** - CPU Requests = ceil(平均 CPU × 1.2 × 系数)

**查询指标：**
- CPU 使用率：`sum(rate(container_cpu_usage_seconds_total{...}[5m]))`
- 内存使用量：`avg(container_memory_usage_bytes{...})`

```bash
./k8stools trend -f config.yaml
```

---

### 🎯 服务资源顾问

基于 Traefik 流量指标分析服务负载，提供 Pod 资源和副本数配置建议，支持 HPA 决策参考。

**核心算法：**

**1. 加权 RPS 计算**
```
总加权 RPS = Σ(各方法 RPS × 方法权重)
方法权重：GET=1, POST/PUT/PATCH/DELETE=2
```

**2. 资源需求计算**
```
CPU Request  = ceil(100 + RPS × 0.1 × 系数)
CPU Limit    = ceil(CPU Request × 系数)
Memory Req   = ceil(128 + RPS × 2 × 系数)
Memory Limit = ceil(Memory Req × 系数)
```

**3. 五级决策模型**

| 场景 | 条件 | 推荐副本数 | 决策 | 风险 |
|------|------|-----------|------|------|
| 无流量 | RPS = 0 | minReplicas | NO_TRAFFIC | LOW |
| 高负载 | 延迟>1000ms 或 RPS>200 | ceil(RPS/15×系数) | SCALE_OUT | HIGH |
| 中等负载 | 延迟>500ms 或 RPS>100 | ceil(RPS/20×系数) | SCALE_OUT | MEDIUM |
| 低负载 | RPS<10 且 延迟<100ms | max(1, ceil(minReplicas×0.8)) | DOWNSIZE_SAFE | LOW |
| 正常 | 其他情况 | max(minReplicas, ceil(RPS/25×系数)) | KEEP | LOW |

```bash
./k8stools resourceAdvisor -f config.yaml
```

---

### 💰 成本估算

根据容器资源请求（CPU Requests）计算每月成本，支持成本分析和资源优化决策。

**计算模型：**
```
每毫核价格 = 单价 / (CPU 核数 × 1000)
容器月费用 = CPU Request (m) × 每毫核价格 × 24 × 30
```

**示例：**
- 机器单价 = 4000 元/月
- CPU 核数 = 16 核
- 容器 CPU Request = 500m
- **月费用** = 0.25 × 500 × 24 × 30 = **90,000 元**

```bash
./k8stools costEstimator -f config.yaml
```

---

## 快速开始

### 1. 配置文件

创建 `config.yaml`：

```yaml
kubeconfig: /root/.kube/config

namespace:
  - {namespace}-prod

prometheus: https://prometheus.demo.com/

# 成本估算配置
cost:
  cpuPrice: 4000   # 单台机器价格（元/月）
  totalCpu: 16     # 单台机器 CPU 核数

# 资源建议配置
resourceAdvisor:
  cpuRequestFactor: 1.0      # CPU 请求系数
  cpuLimitFactor: 2.0       # CPU 限制系数
  memRequestFactor: 1.0     # 内存请求系数
  memLimitFactor: 2.0       # 内存限制系数
  podRedundancyFactor: 1.5  # 副本冗余系数
```

### 2. 编译安装

```bash
# 使用 Makefile 编译（推荐）
make build

# 或直接使用 Go 编译
go build -o k8stools main.go

# 查看所有可用命令
./k8stools --help
```

### 3. 运行模块

```bash
# CPU 使用情况统计
./k8stools cpu -f config.yaml

# 容器运行时行为采集
./k8stools runtimeInspect -f config.yaml

# 资源趋势分析
./k8stools trend -f config.yaml

# 服务资源顾问
./k8stools resourceAdvisor -f config.yaml

# 成本估算
./k8stools costEstimator -f config.yaml
```

---

## 详细文档

### 输出文件说明

各模块会生成带时间戳的 CSV 文件，便于追溯和对比：

| 文件 | 说明 | 生成命令 |
|------|------|----------|
| `deployment_cpu_info.csv` | CPU 使用情况统计 | `cpu` |
| `runtime_snapshot_*.csv` | 容器运行时快照 | `runtimeInspect` |
| `resource_trend.csv` | 资源趋势分析 | `trend` |
| `resource_advice_*.csv` | 服务资源建议 | `resourceAdvisor` |
| `cost_estimate_*.csv` | 成本估算 | `costEstimator` |

### 算法详解

#### 📈 线性回归趋势分析

**斜率计算公式：**
```
slope = (n·Σxy - Σx·Σy) / (n·Σx² - (Σx)²)
```

**趋势判断规则：**
```
- 斜率 > 0.1 且 方差 > 10  → 上升趋势
- 斜率 < -0.1 且 方差 > 10 → 下降趋势
- 其他                     → 稳定
```

#### 🎯 五级决策模型

根据 RPS 和 P95 延迟的组合判断服务状态，动态调整副本数：
- **NO_TRAFFIC** - 无流量，保持最小副本
- **SCALE_OUT** - 高负载，自动扩容
- **DOWNSIZE_SAFE** - 低负载，安全缩容
- **KEEP** - 正常负载，保持配置

---

## 架构图

```
┌─────────────────────────────────────────────────────────────┐
│                        K8stools                              │
├─────────────────────────────────────────────────────────────┤
│                                                               │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │   cpu 模块    │  │runtimeInspect │  │  trend 模块   │      │
│  │              │  │    模块       │  │              │      │
│  │ - Metrics    │  │ - 进程采集    │  │ - 趋势分析    │      │
│  │   Server     │  │ - 端口扫描    │  │ - 线性回归    │      │
│  │ - HPA 配置   │  │ - 环境变量    │  │ - 推荐计算    │      │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘      │
│         │                 │                  │              │
│  ┌──────┴───────┐  ┌──────┴───────┐  ┌──────┴───────┐      │
│  │resourceAdvisor│  │ costEstimator │  │   数据采集层   │      │
│  │              │  │              │  │              │      │
│  │ - RPS 计算   │  │ - 成本估算    │  │ - K8s API    │      │
│  │ - P95 延迟   │  │ - 预算分析    │  │ - Prometheus  │      │
│  │ - 决策模型   │  │ - 成本优化    │  │ - Traefik     │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
│                                                               │
└─────────────────────────────────────────────────────────────┘

数据源：
┌─────────────┐  ┌─────────────┐  ┌─────────────┐
│ Metrics     │  │ Prometheus  │  │ Traefik     │
│ Server      │  │             │  │             │
│ - CPU/Mem   │  │ - 历史数据   │  │ - 流量指标   │
│ - 实时指标  │  │ - 趋势分析   │  │ - 延迟数据   │
└─────────────┘  └─────────────┘  └─────────────┘
```

---

## 贡献指南

欢迎贡献代码、报告问题或提出改进建议！

### 开发环境设置

```bash
# 克隆仓库
git clone https://github.com/karl/k8stools.git
cd k8stools

# 安装依赖
go mod download

# 运行测试
go test ./...

# 代码格式化
go fmt ./...
```

### 提交 Pull Request

1. Fork 本仓库
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 开启 Pull Request

---

## TODO

- [ ] 🚨 增加 `poderrors` 异常 Pod 检查模块
- [ ] 📊 增加 `paradise` 理想资源建议模块
- [ ] 📈 支持 JSON/Table 等多种输出格式
- [ ] 🎨 增加图表可视化输出
- [ ] 🌐 支持 Web UI 界面
- [ ] 🔔 增加告警通知功能
- [ ] 📚 增加完整的使用文档和示例

---

## 许可证

本项目采用 Apache License 2.0 开源许可证。详见 [LICENSE](LICENSE) 文件。

---

## 联系方式

- **作者**: Karl
- **Email**: karlhuang93@gmail.com
- **GitHub**: https://github.com/karl/k8stools

如有任何问题或建议，欢迎提交 Issue 或 Pull Request！

---

<div align="center">

**⭐ 如果这个项目对你有帮助，请给一个 Star！**

Made with ❤️ by K8stools Team

</div>
