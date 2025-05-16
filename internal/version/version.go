package version

import (
	"fmt"
	"runtime"
)

var (
	Version   = "v0.1.0"
	BuildTime = "2025-4-24"
	Commit    = "none"
)

func PrintVersion() {
	fmt.Println("k8stools 版本信息:")
	fmt.Println("  Version:    ", Version)
	fmt.Println("  Commit:     ", Commit)
	fmt.Println("  Build Time: ", BuildTime)
	fmt.Println("  Go Version: ", runtime.Version())
}
