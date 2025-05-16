/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"k8stools/internal/runtimeInspect"
	"k8stools/pkg/config"
)

// runtimeInspectCmd represents the runtimeInspect command
var runtimeInspectCmd = &cobra.Command{
	Use:   "runtimeInspect",
	Short: "采集运行中的 Pod 容器行为信息（进程、端口、环境变量）",
	Long:  `采集运行中的 Pod 容器行为信息（进程、端口、环境变量）`,
	Run: func(cmd *cobra.Command, args []string) {
		c, err := config.ReadYaml(path)
		if err != nil {
			fmt.Println(err)
		}
		runtimeInspect.GetRuntimeInspect(c)
	},
}

func init() {
	rootCmd.AddCommand(runtimeInspectCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// runtimeInspectCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// runtimeInspectCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	runtimeInspectCmd.Flags().StringVarP(&path, "file", "f", "config.yaml", "指定配置文件")
}
