/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"k8stools/internal/cpu"
	"k8stools/pkg/config"
)

// cpuCmd represents the cpu command
var cpuCmd = &cobra.Command{
	Use:   "cpu",
	Short: "获取k8s的cpu使用情况",
	Long:  `获取 k8s 当前使用情况，获取的是瞬时值可以配合监控参考。`,
	Run: func(cmd *cobra.Command, args []string) {
		c, err := config.ReadYaml(path)
		if err != nil {
			fmt.Println(err)
		}
		cpu.GetDeploymentCpu(c)
	},
}

func init() {
	rootCmd.AddCommand(cpuCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// cpuCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// cpuCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	cpuCmd.Flags().StringVarP(&path, "file", "f", "config.yaml", "指定配置文件")
}
