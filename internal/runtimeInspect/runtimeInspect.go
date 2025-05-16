package runtimeInspect

import (
	"context"
	"encoding/csv"
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8stools/pkg/config"
	"os"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"
)

func GetRuntimeInspect(c *config.Config) {
	cfg, err := clientcmd.BuildConfigFromFlags("", c.KubeConfig)
	if err != nil {
		panic(err)
	}
	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		panic(err)
	}

	file, err := os.Create("runtime_snapshot.csv")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	writer.Write([]string{
		"Namespace", "Pod", "Container", "Command", "Ports", "Processes", "Envs",
	})

	ctx := context.Background()

	for _, ns := range c.NameSpace {
		pods, err := clientset.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{})
		if err != nil {
			fmt.Printf("❌ 获取 Pod 失败: %v\n", err)
			continue
		}

		for _, pod := range pods.Items {
			for _, container := range pod.Spec.Containers {
				// 跳过非运行中的 Pod
				if pod.Status.Phase != corev1.PodRunning {
					continue
				}

				// 执行命令采集运行时信息
				processes := execInPod(cfg, clientset, ns, pod.Name, container.Name, []string{"ps", "aux"})
				envs := execInPod(cfg, clientset, ns, pod.Name, container.Name, []string{"printenv"})
				ports := execInPod(cfg, clientset, ns, pod.Name, container.Name, []string{"sh", "-c", "ss -tulnp || netstat -tulnp"})

				writer.Write([]string{
					ns,
					pod.Name,
					container.Name,
					strings.Join(container.Command, " "),
					sanitize(ports),
					sanitize(processes),
					sanitize(envs),
				})
			}
		}
	}

	fmt.Println("✅ 已生成 runtime_snapshot.csv")
}

func execInPod(config *rest.Config, clientset *kubernetes.Clientset, namespace, pod, container string, cmd []string) string {
	req := clientset.CoreV1().RESTClient().
		Post().
		Resource("pods").
		Name(pod).
		Namespace(namespace).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: container,
			Command:   cmd,
			Stdin:     false,
			Stdout:    true,
			Stderr:    true,
			TTY:       false,
		}, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(config, "POST", req.URL())
	if err != nil {
		return fmt.Sprintf("⚠️ Exec 创建失败: %v", err)
	}

	var stdout, stderr strings.Builder
	err = exec.Stream(remotecommand.StreamOptions{
		Stdout: &stdout,
		Stderr: &stderr,
	})
	if err != nil {
		return fmt.Sprintf("⚠️ 执行失败: %v", err)
	}

	return stdout.String()
}

func sanitize(s string) string {
	s = strings.ReplaceAll(s, "\n", " | ")
	s = strings.ReplaceAll(s, "\r", "")
	return strings.TrimSpace(s)
}
