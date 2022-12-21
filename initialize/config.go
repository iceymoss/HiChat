package initialize

import (
	"HiChat/global"

	"github.com/spf13/viper"
	"go.uber.org/zap"
)

func InitConfig() {
	//实例化对象
	v := viper.New()

	configFile := "../HiChat/config-debug.yaml"

	//读取配置文件
	v.SetConfigFile(configFile)

	//读入文件
	if err := v.ReadInConfig(); err != nil {
		panic(err)
	}

	//将数据放入global.ServerConfig 这个对象如何在其他文件中使用--全局变量
	if err := v.Unmarshal(&global.ServiceConfig); err != nil {
		panic(err)
	}

	zap.S().Info("配置信息", global.ServiceConfig)

}
