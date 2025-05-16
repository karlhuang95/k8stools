package config

type Config struct {
	KubeConfig string   `json:"kubeconfig"`
	NameSpace  []string `json:"namespace"`
	Prometheus string   `json:"prometheus"`
	Cost       Cost     `json:"cost"`
}
type Cost struct {
	CpuPrice int `json:"cpuPrice"`
	TotalCpu int `json:"totalCpu"`
}
