package resourceAdvisor

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"k8stools/pkg/config"
)

/*
=====================
配置 & 常量
=====================
*/

var methodWeights = map[string]float64{
	"GET":    1,
	"POST":   2,
	"PUT":    2,
	"DELETE": 2,
	"PATCH":  2,
}

const (
	cpuRequestBase = 500  // m
	cpuLimitBase   = 1000 // m
	memRequestBase = 512  // Mi
	memLimitBase   = 1024 // Mi

	replicaMin = 3
)

/*
=====================
数据结构
=====================
*/

type AdviceRecord struct {
	Namespace           string
	Service             string
	WeightedRPS         float64
	P95LatencyMs        float64
	CPURequest          int
	CPULimit            int
	MemRequest          int
	MemLimit            int
	MinReplicas         int
	RecommendedReplicas int
	Decision            string
	Risk                string
	Confidence          string
	Reason              string
	MetricsWindow       string
	GeneratedAt         string
}

/*
=====================
入口
=====================
*/

func ResourceAdvisor(c *config.Config) error {
	// 验证配置
	if err := validateResourceAdvisorConfig(c); err != nil {
		return fmt.Errorf("配置验证失败: %w", err)
	}

	// 创建带时间戳的输出文件
	timestamp := time.Now().Format("2006-01-02_150405")
	filename := fmt.Sprintf("resource_advice_%s.csv", timestamp)
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("创建输出文件失败: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// 中文表头
	writer.Write([]string{
		"命名空间",
		"服务",
		"RPS（加权）",
		"P95延迟（ms）",
		"请求CPU（m）",
		"限制CPU（m）",
		"请求内存（Mi）",
		"限制内存（Mi）",
		"最小副本数",
		"推荐副本数",
		"决策",
		"风险等级",
		"置信度",
		"原因",
		"指标窗口",
		"生成时间",
	})

	totalRecords := 0
	successfulRecords := 0

	for _, ns := range c.NameSpace {
		records, err := runAdvisorForNamespace(c, ns)
		if err != nil {
			fmt.Printf("❌ 处理命名空间 %s 失败: %v\n", ns, err)
			continue
		}
		
		totalRecords += len(records)
		for _, r := range records {
			writer.Write([]string{
				r.Namespace,
				r.Service,
				fmt.Sprintf("%.2f", r.WeightedRPS),
				fmt.Sprintf("%.1f", r.P95LatencyMs),
				strconv.Itoa(r.CPURequest),
				strconv.Itoa(r.CPULimit),
				strconv.Itoa(r.MemRequest),
				strconv.Itoa(r.MemLimit),
				strconv.Itoa(r.MinReplicas),
				strconv.Itoa(r.RecommendedReplicas),
				r.Decision,
				r.Risk,
				r.Confidence,
				r.Reason,
				r.MetricsWindow,
				r.GeneratedAt,
			})
			successfulRecords++
		}
	}

	fmt.Printf("✅ resourceAdvisor分析完成，已生成 %s (成功处理 %d/%d 条记录)\n", 
		filename, successfulRecords, totalRecords)
	return nil
}

func validateResourceAdvisorConfig(c *config.Config) error {
	if c.Prometheus == "" {
		return fmt.Errorf("Prometheus地址不能为空")
	}
	if len(c.NameSpace) == 0 {
		return fmt.Errorf("命名空间列表不能为空")
	}
	return nil
}

/*
=====================
Namespace 维度
=====================
*/

func runAdvisorForNamespace(c *config.Config, ns string) ([]AdviceRecord, error) {
	var records []AdviceRecord

	services, err := getExportedServices(c.Prometheus)
	if err != nil {
		return nil, fmt.Errorf("获取服务列表失败: %w", err)
	}
	
	if len(services) == 0 {
		return nil, fmt.Errorf("命名空间 %s 中没有发现服务", ns)
	}

	for _, es := range services {
		if !strings.HasSuffix(es, "@kubernetescrd") {
			continue
		}
		r, err := runAdvisorForService(c, ns, es)
		if err != nil {
			fmt.Printf("⚠️ 处理服务 %s 失败: %v\n", es, err)
			continue
		}
		records = append(records, r)
	}

	return records, nil
}

/*
=====================
Service 维度
=====================
*/

func runAdvisorForService(c *config.Config, ns, es string) (AdviceRecord, error) {
	rps, err := queryDailyWeightedRPS(c.Prometheus, es)
	if err != nil {
		return AdviceRecord{}, fmt.Errorf("查询RPS失败: %w", err)
	}
	
	lat, err := queryDailyP95Latency(c.Prometheus, es)
	if err != nil {
		return AdviceRecord{}, fmt.Errorf("查询延迟失败: %w", err)
	}

	// 基于配置的系数计算资源需求
	cpuReq := calculateCPURequest(rps, c.ResourceAdvisor.CPURequestFactor)
	cpuLim := calculateCPULimit(rps, c.ResourceAdvisor.CPULimitFactor)
	memReq := calculateMemoryRequest(rps, c.ResourceAdvisor.MemRequestFactor)
	memLim := calculateMemoryLimit(rps, c.ResourceAdvisor.MemLimitFactor)

	rec := AdviceRecord{
		Namespace:     ns,
		Service:       es,
		WeightedRPS:   rps,
		P95LatencyMs:  lat,
		CPURequest:    cpuReq,
		CPULimit:      cpuLim,
		MemRequest:    memReq,
		MemLimit:      memLim,
		MinReplicas:   replicaMin,
		MetricsWindow: "1d",
		GeneratedAt:   time.Now().Format(time.RFC3339),
	}

	// 改进的决策逻辑
	rec.Decision, rec.RecommendedReplicas, rec.Risk, rec.Confidence, rec.Reason = 
		makeDecision(rps, lat, c.ResourceAdvisor.PodRedundancyFactor)

	return rec, nil
}

func calculateCPURequest(rps float64, factor float64) int {
	base := 100.0 // 基础CPU需求
	if factor == 0 {
		factor = 1.0
	}
	return int(math.Ceil(base + rps*0.1*factor))
}

func calculateCPULimit(rps float64, factor float64) int {
	if factor == 0 {
		factor = 2.0
	}
	return int(math.Ceil(float64(calculateCPURequest(rps, 1.0)) * factor))
}

func calculateMemoryRequest(rps float64, factor float64) int {
	base := 128.0 // 基础内存需求
	if factor == 0 {
		factor = 1.0
	}
	return int(math.Ceil(base + rps*2*factor))
}

func calculateMemoryLimit(rps float64, factor float64) int {
	if factor == 0 {
		factor = 2.0
	}
	return int(math.Ceil(float64(calculateMemoryRequest(rps, 1.0)) * factor))
}

func makeDecision(rps, latency, redundancyFactor float64) (decision string, replicas int, risk, confidence, reason string) {
	if redundancyFactor == 0 {
		redundancyFactor = 1.5
	}

	switch {
	case rps == 0:
		return "NO_TRAFFIC", replicaMin, "LOW", "HIGH", "连续24小时未观测到业务流量"
	
	case latency > 1000 || rps > 200:
		replicas = int(math.Ceil(rps / 15.0 * redundancyFactor))
		return "SCALE_OUT", replicas, "HIGH", "HIGH", "高负载或高延迟，需要扩容"
	
	case latency > 500 || rps > 100:
		replicas = int(math.Ceil(rps / 20.0 * redundancyFactor))
		return "SCALE_OUT", replicas, "MEDIUM", "HIGH", "负载较高，建议扩容"
	
	case rps < 10 && latency < 100:
		replicas = int(math.Ceil(float64(replicaMin) * 0.8))
		if replicas < 1 {
			replicas = 1
		}
		return "DOWNSIZE_SAFE", replicas, "LOW", "MEDIUM", "低负载，可安全缩容"
	
	default:
		replicas = int(math.Ceil(rps / 25.0 * redundancyFactor))
		if replicas < replicaMin {
			replicas = replicaMin
		}
		return "KEEP", replicas, "LOW", "HIGH", "指标稳定，保持当前配置"
	}
}

/*
=====================
Prometheus 查询
=====================
*/

func queryDailyWeightedRPS(promAddr, es string) (float64, error) {
	query := fmt.Sprintf(`avg_over_time(sum by(method)(rate(traefik_service_requests_total{namespace="traefik",exported_service="%s"}[5m]))[1d:1h])`, es)

	result, err := queryProm(promAddr, query)
	if err != nil {
		return 0, fmt.Errorf("查询失败: %w", err)
	}

	var total float64
	for _, r := range result {
		v, err := strconv.ParseFloat(r.Value, 64)
		if err != nil {
			return 0, fmt.Errorf("解析数值失败: %w", err)
		}
		w := methodWeights[r.Metric["method"]]
		if w == 0 {
			w = 1
		}
		total += v * w
	}
	return total, nil
}

func queryDailyP95Latency(promAddr, es string) (float64, error) {
	// 先查询当前 P95 延迟
	query := fmt.Sprintf(`histogram_quantile(0.95,sum by(le)(rate(traefik_service_request_duration_seconds_bucket{namespace="traefik",exported_service="%s"}[5m]))) * 1000`, es)

	result, err := queryProm(promAddr, query)
	if err != nil {
		return 0, fmt.Errorf("查询失败: %w", err)
	}
	if len(result) == 0 {
		return 0, nil // 没有数据是正常情况
	}

	v, err := strconv.ParseFloat(result[0].Value, 64)
	if err != nil {
		return 0, fmt.Errorf("解析延迟数值失败: %w", err)
	}
	return v, nil
}

/*
=====================
Prometheus HTTP Client
=====================
*/

type promResult struct {
	Metric map[string]string
	Value  string
}

func queryProm(promAddr, q string) ([]promResult, error) {
	u := strings.TrimRight(promAddr, "/") + "/api/v1/query"
	params := url.Values{}
	params.Set("query", q)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(u + "?" + params.Encode())
	if err != nil {
		return nil, fmt.Errorf("HTTP请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Prometheus返回错误状态码: %d, 响应: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应体失败: %w", err)
	}

	var res struct {
		Status string `json:"status"`
		Data   struct {
			Result []struct {
				Metric map[string]string `json:"metric"`
				Value  []interface{}     `json:"value"`
			} `json:"result"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &res); err != nil {
		return nil, fmt.Errorf("解析JSON失败: %w", err)
	}

	if res.Status != "success" {
		return nil, fmt.Errorf("Prometheus查询失败: %s", res.Status)
	}

	var out []promResult
	for _, r := range res.Data.Result {
		if len(r.Value) < 2 {
			continue
		}
		if valueStr, ok := r.Value[1].(string); ok {
			out = append(out, promResult{
				Metric: r.Metric,
				Value:  valueStr,
			})
		}
	}
	return out, nil
}

/*
=====================
exported_service 发现
=====================
*/

func getExportedServices(promAddr string) ([]string, error) {
	u := strings.TrimRight(promAddr, "/") + "/api/v1/label/exported_service/values"

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(u)
	if err != nil {
		return nil, fmt.Errorf("HTTP请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Prometheus返回错误状态码: %d, 响应: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应体失败: %w", err)
	}

	var res struct {
		Status string   `json:"status"`
		Data   []string `json:"data"`
	}

	if err := json.Unmarshal(body, &res); err != nil {
		return nil, fmt.Errorf("解析JSON失败: %w", err)
	}

	if res.Status != "success" {
		return nil, fmt.Errorf("Prometheus标签查询失败: %s", res.Status)
	}

	return res.Data, nil
}
