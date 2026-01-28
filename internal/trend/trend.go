package trend

import (
	"context"
	"encoding/csv"
	"fmt"
	"k8stools/pkg/config"
	"net/http"
	"os"
	"time"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

func GetTrend(c *config.Config) error {
	if err := ValidateConfig(c); err != nil {
		return fmt.Errorf("配置验证失败: %w", err)
	}
	
	err := AnalyzeResourceTrends(c.Prometheus, c.NameSpace)
	if err != nil {
		return fmt.Errorf("趋势分析失败: %w", err)
	}
	
	fmt.Println("✅ 资源趋势已保存到 resource_trend.csv")
	return nil
}

func ValidateConfig(c *config.Config) error {
	if c.Prometheus == "" {
		return fmt.Errorf("Prometheus地址不能为空")
	}
	if len(c.NameSpace) == 0 {
		return fmt.Errorf("命名空间列表不能为空")
	}
	return nil
}

func AnalyzeResourceTrends(promAddress string, namespaces []string) error {
	// 创建 Prometheus API client
	client, err := api.NewClient(api.Config{
		Address: promAddress,
		RoundTripper: &http.Transport{
			MaxIdleConns:    10,
			IdleConnTimeout: 30 * time.Second,
		},
	})
	if err != nil {
		return fmt.Errorf("创建 Prometheus 客户端失败: %w", err)
	}

	api := v1.NewAPI(client)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 构建 namespace 正则
	if len(namespaces) == 0 {
		return fmt.Errorf("命名空间列表不能为空")
	}

	nsFilter := ""
	for i, ns := range namespaces {
		if i > 0 {
			nsFilter += "|"
		}
		nsFilter += ns
	}
	filter := fmt.Sprintf(`namespace=~"%s"`, nsFilter)

	// 查询表达式：可自定义
	cpuQuery := fmt.Sprintf(`sum(rate(container_cpu_usage_seconds_total{%s,image!="",container!="POD"}[5m])) by (namespace, pod, container)`, filter)
	memQuery := fmt.Sprintf(`avg(container_memory_usage_bytes{%s,image!="",container!="POD"}) by (namespace, pod, container)`, filter)

	// 执行 Prometheus 查询
	cpuResult, cpuWarnings, err := api.QueryRange(ctx, cpuQuery, v1.Range{
		Start: time.Now().Add(-7 * 24 * time.Hour),
		End:   time.Now(),
		Step:  time.Hour,
	})
	if err != nil {
		return fmt.Errorf("查询 CPU 失败: %w", err)
	}
	if len(cpuWarnings) > 0 {
		fmt.Printf("CPU 查询警告: %v\n", cpuWarnings)
	}

	memResult, memWarnings, err := api.QueryRange(ctx, memQuery, v1.Range{
		Start: time.Now().Add(-7 * 24 * time.Hour),
		End:   time.Now(),
		Step:  time.Hour,
	})
	if err != nil {
		return fmt.Errorf("查询内存失败: %w", err)
	}
	if len(memWarnings) > 0 {
		fmt.Printf("内存查询警告: %v\n", memWarnings)
	}

	// 构建指标映射
	cpuData := parseMatrix(cpuResult, false) // CPU: 不需要转换单位，只乘以 1000 转成 milli
	memData := parseMatrix(memResult, true)  // 内存: 需要从 Bytes 转成 MiB

	// 创建 CSV 文件
	file, err := os.Create("resource_trend.csv")
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// 写入表头
	head := []string{
		"Namespace", "Deployment", "Container", "趋势标签", "趋势斜率",
		"推荐CPU Requests(m)", "推荐CPU Limits(m)", "推荐Memory Requests(Mi)", "推荐Memory Limits(Mi)",
		"日期", "平均CPU(m)", "最大CPU(m)", "平均内存(Mi)", "最大内存(Mi)",
	}
	writer.Write(head)

	// 遍历数据并写入 CSV
	for key, cpuSeries := range cpuData {
		memSeries := memData[key]

		avgCPU, maxCPU := calcAvgMax(cpuSeries)
		avgMem, maxMem := calcAvgMax(memSeries)

		// 推荐值计算
		recommendCPUReq := int(avgCPU * 1.2)
		recommendCPULim := int(maxCPU * 1.5)
		recommendMemReq := int(avgMem * 1.2)
		recommendMemLim := int(maxMem * 1.5)

		// 获取其他信息
		ns := key.namespace
		deploy := extractDeployment(key.pod)
		trend, trendSlope := analyzeTrend(cpuSeries) // 趋势标签和斜率
		
		// 基于趋势调整推荐值
		if trend == "上升趋势" {
			recommendCPUReq = int(float64(recommendCPUReq) * 1.1)
			recommendMemReq = int(float64(recommendMemReq) * 1.1)
		} else if trend == "下降趋势" {
			recommendCPUReq = int(float64(recommendCPUReq) * 0.9)
			recommendMemReq = int(float64(recommendMemReq) * 0.9)
		}
		
		row := []string{
			ns, deploy, key.container, trend,
			fmt.Sprintf("%.2f", trendSlope),
			fmt.Sprintf("%d", recommendCPUReq), fmt.Sprintf("%d", recommendCPULim),
			fmt.Sprintf("%d", recommendMemReq), fmt.Sprintf("%d", recommendMemLim),
			time.Now().Format("2006-01-02"),
			fmt.Sprintf("%.0f", avgCPU), fmt.Sprintf("%.0f", maxCPU),
			fmt.Sprintf("%.0f", avgMem), fmt.Sprintf("%.0f", maxMem),
		}
		writer.Write(row)
	}

	return nil
}

type metricKey struct {
	namespace string
	pod       string
	container string
}

// 解析 Prometheus 查询结果
func parseMatrix(val model.Value, convertToMi bool) map[metricKey][]float64 {
	res := make(map[metricKey][]float64)
	matrix, ok := val.(model.Matrix)
	if !ok {
		return res
	}
	for _, stream := range matrix {
		key := metricKey{
			namespace: string(stream.Metric["namespace"]),
			pod:       string(stream.Metric["pod"]),
			container: string(stream.Metric["container"]),
		}
		for _, v := range stream.Values {
			value := float64(v.Value)
			if convertToMi {
				value = value / (1024 * 1024) // Bytes -> Mi
			} else {
				value = value * 1000 // CPU: cores -> milli-cores
			}
			res[key] = append(res[key], value)
		}
	}
	return res
}

// 计算平均值和最大值
func calcAvgMax(data []float64) (avg, max float64) {
	if len(data) == 0 {
		return 0, 0
	}
	total := 0.0
	max = data[0]
	for _, v := range data {
		total += v
		if v > max {
			max = v
		}
	}
	avg = total / float64(len(data))
	return
}

// 分析趋势
func analyzeTrend(data []float64) (string, float64) {
	if len(data) < 10 {
		return "数据不足", 0
	}
	
	// 使用线性回归分析趋势
	var sumX, sumY, sumXY, sumXX float64
	for i, v := range data {
		x := float64(i)
		sumX += x
		sumY += v
		sumXY += x * v
		sumXX += x * x
	}
	
	n := float64(len(data))
	slope := (n*sumXY - sumX*sumY) / (n*sumXX - sumX*sumX)
	
	// 计算置信度
	meanY := sumY / n
	var sumSqDiff float64
	for _, v := range data {
		sumSqDiff += (v - meanY) * (v - meanY)
	}
	variance := sumSqDiff / n
	
	// 判断趋势
	if slope > 0.1 && variance > 10 {
		return "上升趋势", slope
	} else if slope < -0.1 && variance > 10 {
		return "下降趋势", slope
	}
	return "稳定", slope
}

// 提取 Deployment 名称
func extractDeployment(pod string) string {
	// 假设 deployment 名为 pod-name 的前缀
	// 比如 xxx-7f9cd5b477-abc12 => xxx
	if i := len(pod); i > 0 {
		if dash := lastIndex(pod, "-"); dash > 0 {
			if second := lastIndex(pod[:dash], "-"); second > 0 {
				return pod[:second]
			}
		}
	}
	return pod
}

// 查找分隔符位置
func lastIndex(s string, sep string) int {
	for i := len(s) - 1; i >= 0; i-- {
		if string(s[i]) == sep {
			return i
		}
	}
	return -1
}
