package poderrors

import (
	"context"
	"encoding/csv"
	"fmt"
	"k8stools/pkg/config"
	"os"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func GetPodError(c *config.Config) {
	config, err := clientcmd.BuildConfigFromFlags("", c.KubeConfig)
	if err != nil {
		panic(err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	file, err := os.Create("pod_error_report.csv")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	writer.Write([]string{"Namespace", "Pod", "Container", "状态原因", "错误信息"})

	ctx := context.Background()

	for _, ns := range c.NameSpace {
		pods, err := clientset.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{})
		if err != nil {
			fmt.Printf("❌ 获取命名空间 %s 的 Pod 失败: %v\n", ns, err)
			continue
		}

		for _, pod := range pods.Items {
			for _, cs := range pod.Status.ContainerStatuses {
				// 检查是否为 Waiting 且有常见错误状态
				if cs.State.Waiting != nil {
					reason := cs.State.Waiting.Reason
					if reason == "CrashLoopBackOff" || reason == "Error" || reason == "ImagePullBackOff" {
						message := cs.State.Waiting.Message
						writer.Write([]string{
							ns,
							pod.Name,
							cs.Name,
							reason,
							message,
						})
					}
				}
				// 检查 Terminated 状态是否是失败
				if cs.State.Terminated != nil && cs.State.Terminated.ExitCode != 0 {
					reason := cs.State.Terminated.Reason
					message := cs.State.Terminated.Message
					writer.Write([]string{
						ns,
						pod.Name,
						cs.Name,
						reason,
						message,
					})
				}
			}
		}
	}

	fmt.Println("✅ 已生成 pod_error_report.csv 文件")
}
