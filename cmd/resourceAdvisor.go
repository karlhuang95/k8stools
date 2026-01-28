/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"k8stools/internal/resourceAdvisor"
	"k8stools/pkg/config"
)

// resourceAdvisorCmd represents the resourceAdvisor command
var resourceAdvisorCmd = &cobra.Command{
	Use:   "resourceAdvisor",
	Short: "资源顾问",
	Long:  `根据监控信息，估算项目资源分配情况`,
	Run: func(cmd *cobra.Command, args []string) {
		c, err := config.ReadYaml(path)
		if err != nil {
			fmt.Println(err)
		}
		if err := resourceAdvisor.ResourceAdvisor(c); err != nil {
			fmt.Printf("❌ 资源顾问分析失败: %v\n", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(resourceAdvisorCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// resourceAdvisorCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// resourceAdvisorCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
