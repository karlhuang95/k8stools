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

func ResourceAdvisor(c *config.Config) {
	file, err := os.Create("resource_advice.csv")
	if err != nil {
		panic(err)
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

	for _, ns := range c.NameSpace {
		records := runAdvisorForNamespace(c, ns)
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
		}
	}

	fmt.Println("✅ resourceAdvisor（日级）分析完成，已生成 resource_advice.csv")
}

/*
=====================
Namespace 维度
=====================
*/

func runAdvisorForNamespace(c *config.Config, ns string) []AdviceRecord {
	var records []AdviceRecord

	services := getExportedServices(c.Prometheus)
	for _, es := range services {
		if !strings.HasSuffix(es, "@kubernetescrd") {
			continue
		}
		r := runAdvisorForService(c, ns, es)
		records = append(records, r)
	}

	return records
}

/*
=====================
Service 维度
=====================
*/

func runAdvisorForService(c *config.Config, ns, es string) AdviceRecord {
	rps := queryDailyWeightedRPS(c.Prometheus, es)
	lat := queryDailyP95Latency(c.Prometheus, es)

	rec := AdviceRecord{
		Namespace:     ns,
		Service:       es,
		WeightedRPS:   rps,
		P95LatencyMs:  lat,
		CPURequest:    cpuRequestBase,
		CPULimit:      cpuLimitBase,
		MemRequest:    memRequestBase,
		MemLimit:      memLimitBase,
		MinReplicas:   replicaMin,
		MetricsWindow: "1d",
		GeneratedAt:   time.Now().Format(time.RFC3339),
	}

	// 决策逻辑（日级）
	switch {
	case rps == 0:
		rec.Decision = "NO_TRAFFIC"
		rec.RecommendedReplicas = replicaMin
		rec.Risk = "LOW"
		rec.Confidence = "HIGH"
		rec.Reason = "连续24小时未观测到业务流量"

	case rps > 50 || lat > 500:
		rec.Decision = "SCALE_OUT"
		rec.RecommendedReplicas = int(math.Ceil(rps / 20.0 * 1.5))
		rec.Risk = "MEDIUM"
		rec.Confidence = "HIGH"
		rec.Reason = "日峰值RPS或P95延迟偏高"

	default:
		rec.Decision = "KEEP"
		rec.RecommendedReplicas = replicaMin
		rec.Risk = "LOW"
		rec.Confidence = "MEDIUM"
		rec.Reason = "日级指标稳定，资源利用合理"
	}

	if rec.RecommendedReplicas < replicaMin {
		rec.RecommendedReplicas = replicaMin
	}

	return rec
}

/*
=====================
Prometheus 查询
=====================
*/

func queryDailyWeightedRPS(promAddr, es string) float64 {
	query := fmt.Sprintf(`
avg_over_time(
  sum by(method)(
    rate(
      traefik_service_requests_total{
        namespace="traefik",
        exported_service="%s"
      }[5m]
    )
  )[1d]
)
`, es)

	result := queryProm(promAddr, query)

	var total float64
	for _, r := range result {
		v, _ := strconv.ParseFloat(r.Value, 64)
		w := methodWeights[r.Metric["method"]]
		if w == 0 {
			w = 1
		}
		total += v * w
	}
	return total
}

func queryDailyP95Latency(promAddr, es string) float64 {
	query := fmt.Sprintf(`
histogram_quantile(
  0.95,
  sum by(le)(
    rate(
      traefik_service_request_duration_seconds_bucket{
        namespace="traefik",
        exported_service="%s"
      }[5m]
    )
  )
) * 1000
`, es)

	result := queryProm(promAddr, query)
	if len(result) == 0 {
		return 0
	}
	v, _ := strconv.ParseFloat(result[0].Value, 64)
	return v
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

func queryProm(promAddr, q string) []promResult {
	u := strings.TrimRight(promAddr, "/") + "/api/v1/query"
	params := url.Values{}
	params.Set("query", q)

	resp, err := http.Get(u + "?" + params.Encode())
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var res struct {
		Data struct {
			Result []struct {
				Metric map[string]string `json:"metric"`
				Value  []interface{}     `json:"value"`
			} `json:"result"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &res); err != nil {
		return nil
	}

	var out []promResult
	for _, r := range res.Data.Result {
		out = append(out, promResult{
			Metric: r.Metric,
			Value:  r.Value[1].(string),
		})
	}
	return out
}

/*
=====================
exported_service 发现
=====================
*/

func getExportedServices(promAddr string) []string {
	u := strings.TrimRight(promAddr, "/") + "/api/v1/label/exported_service/values"

	resp, err := http.Get(u)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var res struct {
		Data []string `json:"data"`
	}

	if err := json.Unmarshal(body, &res); err != nil {
		return nil
	}

	return res.Data
}
