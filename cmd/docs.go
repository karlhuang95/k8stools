package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

var docsCmd = &cobra.Command{
	Use:   "docs",
	Short: "生成 Markdown CLI 使用文档",
	Long:  "生成项目的 CLI 使用文档，支持 Hugo/VuePress 格式，可用于内部 Wiki 或开发文档。",
	Run: func(cmd *cobra.Command, args []string) {
		outDir := "./docs"
		if err := os.MkdirAll(outDir, 0755); err != nil {
			fmt.Printf("❌ 创建输出目录失败: %v\n", err)
			return
		}

		err := doc.GenMarkdownTreeCustom(rootCmd, outDir,
			func(filename string) string {
				// 提取命令名作为 title
				base := strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename))
				return fmt.Sprintf(`---
title: "%s"
description: "CLI 文档 - %s"
---

`, base, base)
			},
			func(s string) string {
				// 保持原样链接处理
				return s
			},
		)

		if err != nil {
			fmt.Printf("❌ 生成文档失败: %v\n", err)
		} else {
			fmt.Printf("✅ CLI 使用文档已导出到: %s\n", outDir)
		}
	},
}

func init() {
	rootCmd.AddCommand(docsCmd)
}
