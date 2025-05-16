/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"k8stools/internal/costEstimator"
	"k8stools/pkg/config"

	"github.com/spf13/cobra"
)

// costEstimatorCmd represents the costEstimator command
var costEstimatorCmd = &cobra.Command{
	Use:   "costEstimator",
	Short: "成本估算",
	Long:  `成本估算`,
	Run: func(cmd *cobra.Command, args []string) {
		c, err := config.ReadYaml(path)
		if err != nil {
			fmt.Println(err)
		}
		costEstimator.GetCostEstimate(c)
	},
}

func init() {
	rootCmd.AddCommand(costEstimatorCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// costEstimatorCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// costEstimatorCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	costEstimatorCmd.Flags().StringVarP(&path, "file", "f", "config.yaml", "指定配置文件")
}
