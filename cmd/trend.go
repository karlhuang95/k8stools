/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"k8stools/internal/trend"
	"k8stools/pkg/config"
)

// trendCmd represents the trend command
var trendCmd = &cobra.Command{
	Use:   "trend",
	Short: "基于 Prometheus 的资源使用趋势分析与建议",
	Long:  `根据prometheus一周的策略，分析出流量趋势`,
	Run: func(cmd *cobra.Command, args []string) {
		c, err := config.ReadYaml(path)
		if err != nil {
			fmt.Println(err)
		}
		if err := trend.GetTrend(c); err != nil {
			fmt.Printf("❌ 趋势分析失败: %v\n", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(trendCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// trendCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// trendCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	trendCmd.Flags().StringVarP(&path, "file", "f", "config.yaml", "指定配置文件")
}
