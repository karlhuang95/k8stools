package config

type Config struct {
	KubeConfig      string                `json:"kubeconfig"`
	NameSpace       []string              `json:"namespace"`
	Prometheus      string                `json:"prometheus"`
	Cost            Cost                  `json:"cost"`
	ResourceAdvisor ResourceAdvisorConfig `json:"resourceAdvisor"`
}

type Cost struct {
	CpuPrice int `json:"cpuPrice"`
	TotalCpu int `json:"totalCpu"`
}

// ResourceAdvisorConfig 包含 ResourceAdvisor 所需参数
type ResourceAdvisorConfig struct {
	UserMaxConn int64    `json:"userMaxConn"`
	Business    []string `json:"business"`

	// 可调系数
	CPURequestFactor    float64 `json:"cpuRequestFactor"`    // request.cpu 系数
	CPULimitFactor      float64 `json:"cpuLimitFactor"`      // limit.cpu 系数
	MemRequestFactor    float64 `json:"memRequestFactor"`    // request.mem 系数
	MemLimitFactor      float64 `json:"memLimitFactor"`      // limit.mem 系数
	PodRedundancyFactor float64 `json:"podRedundancyFactor"` // pods冗余系数
}
