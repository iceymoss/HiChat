package initialize

import (
	"HiChat/global"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/viper"
	"go.uber.org/zap"
)

func InitConfig() {
	//实例化对象
	v := viper.New()

	// 设置默认值
	v.SetDefault("port", 8000)
	v.SetDefault("mysql.host", "127.0.0.1")
	v.SetDefault("mysql.port", 3306)
	v.SetDefault("mysql.name", "hi_chat")
	v.SetDefault("mysql.user", "iceymoss")
	v.SetDefault("mysql.password", "Yk?123456")
	v.SetDefault("redis.host", "127.0.0.1")
	v.SetDefault("redis.port", 6379)

	// 支持环境变量（使用HICHAT作为前缀）
	v.SetEnvPrefix("HICHAT")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// 尝试读取配置文件（如果存在）
	configFile := "config-debug.yaml"
	if _, err := os.Stat(configFile); err == nil {
		v.SetConfigFile(configFile)
		if err := v.ReadInConfig(); err == nil {
			zap.S().Info("使用配置文件:", configFile)
		}
	} else {
		// 如果当前目录没有，尝试上级目录
		configFile = "../HiChat/config-debug.yaml"
		if _, err := os.Stat(configFile); err == nil {
			v.SetConfigFile(configFile)
			if err := v.ReadInConfig(); err == nil {
				zap.S().Info("使用配置文件:", configFile)
			}
		}
	}

	// 环境变量优先级更高，直接覆盖配置
	if os.Getenv("PORT") != "" {
		if port, err := strconv.Atoi(os.Getenv("PORT")); err == nil {
			v.Set("port", port)
		}
	}
	if os.Getenv("MYSQL_HOST") != "" {
		v.Set("mysql.host", os.Getenv("MYSQL_HOST"))
	}
	if os.Getenv("MYSQL_PORT") != "" {
		if port, err := strconv.Atoi(os.Getenv("MYSQL_PORT")); err == nil {
			v.Set("mysql.port", port)
		}
	}
	if os.Getenv("MYSQL_NAME") != "" {
		v.Set("mysql.name", os.Getenv("MYSQL_NAME"))
	}
	if os.Getenv("MYSQL_USER") != "" {
		v.Set("mysql.user", os.Getenv("MYSQL_USER"))
	}
	if os.Getenv("MYSQL_PASSWORD") != "" {
		v.Set("mysql.password", os.Getenv("MYSQL_PASSWORD"))
	}
	if os.Getenv("REDIS_HOST") != "" {
		v.Set("redis.host", os.Getenv("REDIS_HOST"))
	}
	if os.Getenv("REDIS_PORT") != "" {
		if port, err := strconv.Atoi(os.Getenv("REDIS_PORT")); err == nil {
			v.Set("redis.port", port)
		}
	}

	// 将数据放入global.ServiceConfig
	if err := v.Unmarshal(&global.ServiceConfig); err != nil {
		panic(err)
	}

	zap.S().Info("配置信息", global.ServiceConfig)
}
