package cpu

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"strings"

	autoscalingv1 "k8s.io/api/autoscaling/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	metrics "k8s.io/metrics/pkg/client/clientset/versioned"
	"k8stools/pkg/config"
)

func GetDeyloymentCpu(c *config.Config) {
	cfg, err := clientcmd.BuildConfigFromFlags("", c.KubeConfig)
	if err != nil {
		panic(err)
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		panic(err)
	}

	metricsClient, err := metrics.NewForConfig(cfg)
	if err != nil {
		panic(err)
	}

	file, err := os.Create("deployment_cpu_info.csv")
	if err != nil {
		panic(err)
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	defer writer.Flush()

	writer.Write([]string{
		"Namespace", "Deployment",
		"Main CPU Usage (m)", "Sidecar CPU Usage (m)",
		"Main CPU Requests (m)", "Sidecar CPU Requests (m)",
		"Main CPU Limits (m)", "Sidecar CPU Limits (m)",
		"Pod Min Replicas", "Pod Max Replicas",
	})

	for _, ns := range c.NameSpace {
		collectDeploymentStats(ns, clientset, metricsClient, writer)
	}

	fmt.Println("✅ Deployment CPU 统计完成，输出文件：deployment_cpu_info.csv")
}

func collectDeploymentStats(ns string, clientset *kubernetes.Clientset, metricsClient *metrics.Clientset, writer *csv.Writer) {
	ctx := context.Background()

	deployments, err := clientset.AppsV1().Deployments(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		fmt.Printf("❌ 获取 Deployment 失败: %v\n", err)
		return
	}

	podMetricsList, _ := metricsClient.MetricsV1beta1().PodMetricses(ns).List(ctx, metav1.ListOptions{})
	podMetricsMap := make(map[string]map[string]int64)
	for _, podMetrics := range podMetricsList.Items {
		metrics := make(map[string]int64)
		for _, c := range podMetrics.Containers {
			metrics[c.Name] = c.Usage.Cpu().MilliValue()
		}
		podMetricsMap[podMetrics.Name] = metrics
	}

	hpaList, _ := clientset.AutoscalingV1().HorizontalPodAutoscalers(ns).List(ctx, metav1.ListOptions{})
	hpaMap := make(map[string]*autoscalingv1.HorizontalPodAutoscaler)
	for _, hpa := range hpaList.Items {
		hpaMap[hpa.Spec.ScaleTargetRef.Name] = &hpa
	}

	for _, deploy := range deployments.Items {
		selector := deploy.Spec.Selector.MatchLabels
		selectorStr := []string{}
		for k, v := range selector {
			selectorStr = append(selectorStr, fmt.Sprintf("%s=%s", k, v))
		}

		pods, _ := clientset.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{
			LabelSelector: strings.Join(selectorStr, ","),
		})

		var mainUsage, sidecarUsage, mainRequest, sidecarRequest, mainLimit, sidecarLimit int64
		for _, pod := range pods.Items {
			metrics := podMetricsMap[pod.Name]
			for i, c := range pod.Spec.Containers {
				cpuReq := c.Resources.Requests.Cpu().MilliValue()
				cpuLim := c.Resources.Limits.Cpu().MilliValue()
				cpuUse := metrics[c.Name]
				if i == 0 {
					mainUsage += cpuUse
					mainRequest += cpuReq
					mainLimit += cpuLim
				} else {
					sidecarUsage += cpuUse
					sidecarRequest += cpuReq
					sidecarLimit += cpuLim
				}
			}
		}

		minReplicas := int32(0)
		maxReplicas := *deploy.Spec.Replicas
		if hpa, ok := hpaMap[deploy.Name]; ok {
			minReplicas = *hpa.Spec.MinReplicas
			maxReplicas = hpa.Spec.MaxReplicas
		} else {
			minReplicas = *deploy.Spec.Replicas
		}

		writer.Write([]string{
			ns,
			deploy.Name,
			fmt.Sprintf("%d", mainUsage),
			fmt.Sprintf("%d", sidecarUsage),
			fmt.Sprintf("%d", mainRequest),
			fmt.Sprintf("%d", sidecarRequest),
			fmt.Sprintf("%d", mainLimit),
			fmt.Sprintf("%d", sidecarLimit),
			fmt.Sprintf("%d", minReplicas),
			fmt.Sprintf("%d", maxReplicas),
		})
	}
}
