package config

import (
	"fmt"
	"github.com/spf13/viper"
)

func ReadYaml(path string) (c *Config, err error) {
	// todo: 设置配置文件
	viper.SetConfigFile(path)
	// todo:  读取配置文件
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			fmt.Printf("未找到配置文件: %v", path)
		} else {
			fmt.Printf("读取配置文件报错： %v", err)
		}
		return nil, nil
	}
	if err := viper.Unmarshal(&c); err != nil {
		return nil, fmt.Errorf("解析配置文件时出错: %w", err)
	}
	return c, nil
}
