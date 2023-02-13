package config

//MysqlConfig mysql信息配置
type MysqlConfig struct {
	Host     string `mapstructure:"host" json:"host"`
	Port     int    `mapstructure:"port" json:"port"`
	Name     string `mapstructure:"name" json:"Name"`
	User     string `mapstructure:"user" json:"user"`
	Password string `mapstructure:"password" json:"password"`
}

type RedisConfig struct {
	Host string `mapstructure:"host" json:"host"`
	Port int    `mapstructure:"port" json:"port"`
}

type ServiceConfig struct {
	Port    int         `mapstructure:"port" json:"port"`
	DB      MysqlConfig `mapstructure:"mysql" json:"mysql"`
	RedisDB RedisConfig `mapstructure:"redis" json:"redis"`
}
