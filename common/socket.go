package common

import (
	"HiChat/global"
	"context"
	"fmt"

	"go.uber.org/zap"
)

const (
	PublishKey = "websocket"
)

//Publish 发布消息到Redis
func Publish(ctx context.Context, channel string, msg string) error {
	var err error
	fmt.Println("Publish 。。。。", msg)
	err = global.RedisDB.Publish(ctx, channel, msg).Err()
	if err != nil {
		fmt.Println(err)
	}
	return err
}

//Subscribe 订阅Redis消息
func Subscribe(ctx context.Context, channel string) (string, error) {
	//获取订阅
	sub := global.RedisDB.Subscribe(ctx, channel)
	fmt.Println("Subscribe 。。。。", ctx)
	//获取订阅中的消息
	msg, err := sub.ReceiveMessage(ctx)
	if err != nil {
		zap.S().Info("获取订阅数据失败", err)
		return "", err
	}
	fmt.Println("Subscribe 。。。。", msg.Payload)
	return msg.Payload, err
}
