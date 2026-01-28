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
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"
)

func GetRuntimeInspect(c *config.Config) error {
	cfg, err := clientcmd.BuildConfigFromFlags("", c.KubeConfig)
	if err != nil {
		return fmt.Errorf("构建kubeconfig失败: %w", err)
	}
	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return fmt.Errorf("创建Kubernetes客户端失败: %w", err)
	}

	// 创建带时间戳的输出文件
	timestamp := time.Now().Format("2006-01-02_150405")
	filename := fmt.Sprintf("runtime_snapshot_%s.csv", timestamp)
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("创建输出文件失败: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	writer.Write([]string{
		"Namespace", "Pod", "Container", "Command", "Ports", "Processes", "Envs",
	})

	ctx := context.Background()

	// 限制并发执行，避免对集群造成压力
	semaphore := make(chan struct{}, 5) // 最多同时执行5个容器检查
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, ns := range c.NameSpace {
		pods, err := clientset.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{})
		if err != nil {
			fmt.Printf("❌ 获取命名空间 %s 的 Pod 失败: %v\n", ns, err)
			continue
		}

		for _, pod := range pods.Items {
			// 跳过非运行中的 Pod
			if pod.Status.Phase != corev1.PodRunning {
				continue
			}

			for _, container := range pod.Spec.Containers {
				wg.Add(1)
				go func(ns, podName, containerName string, cmd []string) {
					defer wg.Done()
					semaphore <- struct{}{}
					defer func() { <-semaphore }()

				// 执行命令采集运行时信息（带超时控制）
				processes, err1 := execInPod(cfg, clientset, ns, podName, containerName, []string{"ps", "aux"})
				envs, err2 := execInPod(cfg, clientset, ns, podName, containerName, []string{"printenv"})
				ports, err3 := execInPod(cfg, clientset, ns, podName, containerName, []string{"sh", "-c", "ss -tulnp || netstat -tulnp"})

				// 处理错误
				if err1 != nil {
					processes = fmt.Sprintf("执行失败: %v", err1)
				}
				if err2 != nil {
					envs = fmt.Sprintf("执行失败: %v", err2)
				}
				if err3 != nil {
					ports = fmt.Sprintf("执行失败: %v", err3)
				}

				mu.Lock()
				writer.Write([]string{
					ns,
					podName,
					containerName,
					strings.Join(cmd, " "),
					sanitize(ports),
					sanitize(processes),
					sanitize(envs),
				})
				mu.Unlock()
				}(ns, pod.Name, container.Name, container.Command)
			}
		}
	}

	wg.Wait()
	fmt.Printf("✅ 已生成 %s\n", filename)
	return nil
}

func execInPod(config *rest.Config, clientset *kubernetes.Clientset, namespace, pod, container string, cmd []string) (string, error) {
	// 安全检查：只允许执行白名单命令
	if !isSafeCommand(cmd) {
		return "", fmt.Errorf("命令不安全: %v", cmd)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

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
		return "", fmt.Errorf("Exec创建失败: %w", err)
	}

	var stdout, stderr strings.Builder
	err = exec.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdout: &stdout,
		Stderr: &stderr,
	})
	if err != nil {
		return "", fmt.Errorf("执行失败: %w (stderr: %s)", err, stderr.String())
	}

	return stdout.String(), nil
}

func isSafeCommand(cmd []string) bool {
	safeCommands := map[string]bool{
		"ps":     true,
		"aux":    true,
		"printenv": true,
		"ss":     true,
		"-tulnp": true,
		"netstat": true,
		"sh":     true,
		"-c":     true,
	}

	for _, arg := range cmd {
		if !safeCommands[arg] {
			// 检查是否包含危险字符
			if strings.ContainsAny(arg, "&|;`$()<>\"") {
				return false
			}
		}
	}
	return true
}

func sanitize(s string) string {
	s = strings.ReplaceAll(s, "\n", " | ")
	s = strings.ReplaceAll(s, "\r", "")
	return strings.TrimSpace(s)
}
