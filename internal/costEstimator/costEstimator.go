package costEstimator

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
)

func GetCostEstimate(c *config.Config) {
	cpu := c.Cost.TotalCpu
	price := c.Cost.CpuPrice

	// 配置 kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", c.KubeConfig)
	if err != nil {
		panic(err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	namespaces := c.NameSpace

	// 创建 CSV 文件
	file, err := os.Create("cost_estimate.csv")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// 写入表头
	writer.Write([]string{
		"Namespace", "Pod", "Container",
		"CPU Request (m)", "CPU Cost ($)",
	})

	// 每个 CPU 核心的费用
	cpuCostPerUnit := float64(price) / float64(cpu)

	ctx := context.Background()

	for _, ns := range namespaces {
		pods, err := clientset.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{})
		if err != nil {
			fmt.Printf("无法获取命名空间 %s 的 Pods: %v\n", ns, err)
			continue
		}

		for _, pod := range pods.Items {
			for _, container := range pod.Spec.Containers {
				// 获取容器资源请求
				cpuRequest := container.Resources.Requests[corev1.ResourceCPU]
				cpuMilli := cpuRequest.MilliValue() // 毫核心

				// 计算 CPU 费用
				totalCost := float64(cpuMilli) * cpuCostPerUnit / 1000 // 计算 CPU 请求的费用

				writer.Write([]string{
					ns,
					pod.Name,
					container.Name,
					strconv.FormatInt(cpuMilli, 10),
					strconv.FormatFloat(totalCost, 'f', 4, 64),
				})
			}
		}
	}

	fmt.Println("✅ 已生成 cost_estimate.csv 文件")
}
