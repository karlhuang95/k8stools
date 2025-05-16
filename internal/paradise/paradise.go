package paradise

import (
	"context"
	"encoding/csv"
	"fmt"
	"k8stools/pkg/config"
	"os"
	"strconv"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	metrics "k8s.io/metrics/pkg/client/clientset/versioned"
)

func GetParadise(c *config.Config) {
	config, err := clientcmd.BuildConfigFromFlags("", c.KubeConfig)
	if err != nil {
		panic(err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	metricsClient, err := metrics.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	namespaces := c.NameSpace

	file, err := os.Create("pod_resource_advice.csv")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	writer.Write([]string{
		"Namespace", "Deployment", "Container",
		"建议 CPU Requests (m)", "建议 CPU Limits (m)",
		"建议 Memory Requests (Mi)", "建议 Memory Limits (Mi)",
		"建议说明",
	})

	ctx := context.Background()

	for _, ns := range namespaces {
		deployments, err := clientset.AppsV1().Deployments(ns).List(ctx, metav1.ListOptions{})
		if err != nil {
			fmt.Printf("无法获取命名空间 %s 的 Deployments: %v\n", ns, err)
			continue
		}

		podList, err := clientset.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{})
		if err != nil {
			fmt.Printf("无法获取命名空间 %s 的 Pods: %v\n", ns, err)
			continue
		}

		metricsList, err := metricsClient.MetricsV1beta1().PodMetricses(ns).List(ctx, metav1.ListOptions{})
		if err != nil {
			fmt.Printf("无法获取命名空间 %s 的 metrics: %v\n", ns, err)
			continue
		}

		// Map: PodName -> Metrics
		podMetricsMap := make(map[string]map[string]corev1.ResourceList)
		for _, pm := range metricsList.Items {
			containerMap := make(map[string]corev1.ResourceList)
			for _, c := range pm.Containers {
				containerMap[c.Name] = c.Usage
			}
			podMetricsMap[pm.Name] = containerMap
		}

		for _, deploy := range deployments.Items {
			selector := deploy.Spec.Selector.MatchLabels
			matchPods := []corev1.Pod{}
			for _, pod := range podList.Items {
				matched := true
				for k, v := range selector {
					if pod.Labels[k] != v {
						matched = false
						break
					}
				}
				if matched {
					matchPods = append(matchPods, pod)
				}
			}

			containerUsageMap := make(map[string][]corev1.ResourceList)
			for _, pod := range matchPods {
				if cm, exists := podMetricsMap[pod.Name]; exists {
					for cname, usage := range cm {
						containerUsageMap[cname] = append(containerUsageMap[cname], usage)
					}
				}
			}

			for cname, usages := range containerUsageMap {
				var totalCPU, totalMem int64
				for _, u := range usages {
					totalCPU += u.Cpu().MilliValue()
					totalMem += u.Memory().Value() / (1024 * 1024)
				}

				count := int64(len(usages))
				if count == 0 {
					continue
				}
				avgCPU := totalCPU / count
				avgMem := totalMem / count

				var cpuRequest, cpuLimit int64
				var memRequest, memLimit int64
				var advice string

				switch {
				case avgCPU < 50:
					cpuRequest = 50
					cpuLimit = 100
					advice = "使用率较低，建议使用最小推荐值"
				case avgCPU > 1000:
					cpuRequest = avgCPU / 2
					cpuLimit = avgCPU
					advice = "使用率较高，建议设置严格限制"
				default:
					cpuRequest = avgCPU / 2
					cpuLimit = avgCPU
					advice = "正常使用，建议标准配置"
				}

				switch {
				case avgMem < 64:
					memRequest = 64
					memLimit = 128
				case avgMem > 1024:
					memRequest = int64(float64(avgMem) * 0.75)
					memLimit = int64(float64(avgMem) * 1.5)
				default:
					memRequest = int64(float64(avgMem) * 0.8)
					memLimit = int64(float64(avgMem) * 1.5)
				}

				writer.Write([]string{
					ns,
					deploy.Name,
					cname,
					strconv.FormatInt(cpuRequest, 10),
					strconv.FormatInt(cpuLimit, 10),
					strconv.FormatInt(memRequest, 10),
					strconv.FormatInt(memLimit, 10),
					advice,
				})
			}
		}
	}
	fmt.Println("✅ 已生成 pod_resource_advice.csv 文件（基于 Deployment）")
}
