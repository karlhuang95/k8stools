/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"k8stools/internal/poderrors"
	"k8stools/pkg/config"
)

// poderrorsCmd represents the poderrors command
var poderrorsCmd = &cobra.Command{
	Use:   "poderrors",
	Short: "异常检查",
	Long:  `检测异常 Pod 状态（CrashLoop、ImagePull 等）`,
	Run: func(cmd *cobra.Command, args []string) {
		c, err := config.ReadYaml(path)
		if err != nil {
			fmt.Println(err)
		}
		poderrors.GetPodError(c)

	},
}

func init() {
	rootCmd.AddCommand(poderrorsCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// poderrorsCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// poderrorsCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	poderrorsCmd.Flags().StringVarP(&path, "file", "f", "config.yaml", "指定配置文件")
}
