package main

import "github.com/go-redis/redis"

func main() {
	redisDb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379", // redis地址
		Password: "",               // redis密码，没有则留空
		DB:       0,                // 默认数据库，默认是0
	})

	//通过 *redis.Client.Ping() 来检查是否成功连接到了redis服务器
	_, err := redisDb.Ping().Result()
	if err != nil {
		panic(err)
	}
}
