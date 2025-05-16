/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"k8stools/internal/paradise"
	"k8stools/pkg/config"
)

// paradiseCmd represents the paradise command
var paradiseCmd = &cobra.Command{
	Use:   "paradise",
	Short: "k8s理想情况分配",
	Long:  `根据特定规则，对k8s的pod资源进行调整`,
	Run: func(cmd *cobra.Command, args []string) {
		c, err := config.ReadYaml(path)
		if err != nil {
			fmt.Println(err)
		}
		paradise.GetParadise(c)
	},
}

func init() {
	rootCmd.AddCommand(paradiseCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// paradiseCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// paradiseCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	paradiseCmd.Flags().StringVarP(&path, "file", "f", "config.yaml", "指定配置文件")
}
