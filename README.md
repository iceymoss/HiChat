# HiChat
[toc]

### 概况(HiChat)
基于go开发的一款在线聊天web应用

#### 介绍

在学习完go的基础后，现在从0到1来搭建一个简单的即时聊天项目(IM)的api，项目名称为HiChat。

#### 主要功能

* 登录、注册、用户信息更新、账号注销。
* 单聊、群聊。
* 发送文字、表情包、图片、语音。
* 加好友、好友列表、建群、加入群、群列表。

#### 技术栈

在该项目中使用的技术栈：Go、Gin、Websocket、UDP、Mysql、Redis、Viper、Gorm、Zap、Md5、Jwt

#### 前置知识

在学习本项目开发前，您应该有go基础、mysql基础、计算机网络基础。

您在学习中可能需要，补充学习的知识：

* [「Golang成长之路」系列文章](https://learnku.com/articles/61599)
*    [GoWeb框架Gin学习总结](https://learnku.com/articles/69259)
*    [MD5加密](https://learnku.com/articles/69126)
*    [GORM学习入门](https://learnku.com/articles/68943)
*   [基于viper的配置读取](https://learnku.com/articles/73184)

#### 系统架构

系统架构如下：

<img src="https://blogfiles-iceymoss.oss-cn-hangzhou.aliyuncs.com/blogs/%E6%88%AA%E5%B1%8F2022-12-28%20%E4%B8%8B%E5%8D%883.01.10.png" style="zoom:50%;" />

#### 通信流程

通信流程如下：

<img src="https://blogfiles-iceymoss.oss-cn-hangzhou.aliyuncs.com/blogs/%E6%88%AA%E5%B1%8F2022-12-28%20%E4%B8%8B%E5%8D%883.25.31.png" style="zoom:50%;" />

###  开发环境

#### IDE: Goland

#### 数据库工具：VScode

环境搭建可参考这一篇文章：[web项目部署](https://learnku.com/articles/74054)



### 项目初始化

这里将项目放置目录：

```
/Users/feng/go/src
```

使用命令初始化：

```shell
go mod init HiChat
```

当然也可是直接使用goland新建项目

构建项目目录：

```
HiChat   
    ├── common    //放置公共文件
    │  
    ├── config    //做配置文件
    │  
    ├── dao       //数据库crud
    │  
    ├── global    //放置各种连接池，配置等
    │   
    ├── initialize  //项目初始化文件
    │  
    ├── middlewear  //放置web中间件
    │ 
    ├── models      //数据库表设计
    │   
    ├── router   		//路由
    │   
    ├── service     //对外api
    │   
    ├── test        //测试文件
    │  
    ├── main.go     //项目入口
    ├── go.mod			//项目依赖管理
    ├── go.sum			//项目依赖管理
```



### 配置mysql连接池

#### 新建数据库

使用SQL语句新建数据库hi_chat

#### 声明全局mysql连接池变量

在global目录下新建一个global.go文件

```go
package global

import (
	"gorm.io/gorm"
)

var (
	DB            *gorm.DB
)
```

#### 建立连接池

拉取mysql驱动和gorm

```
go get gorm.io/driver/mysql
go get gorm.io/gorm
go get gorm.io/gorm/logger
```

在initialize项目下创建db.go文件

```go
package initialize

import (
	"fmt"
	"log"
	"os"
	"time"
  
  "HiChat/global"

	"gorm.io/gorm/logger"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func InitDB() {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local", User,
		Password, Host, Port, DBName)
	//注意：User和Password为MySQL数据库的管理员密码，Host和Port为数据库连接ip端口，DBname为要连接的数据库

	//sql语句配置
	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer（日志输出的目标，前缀和日志包含的内容——译者注）
		logger.Config{
			SlowThreshold:             time.Second, // 慢 SQL 阈值
			LogLevel:                  logger.Info, // 日志级别
			IgnoreRecordNotFoundError: true,        // 忽略ErrRecordNotFound（记录未找到）错误
			Colorful:                  true,        // 禁用彩色打印
		},
	)

	var err error
  
  //将获取到的连接赋值到global.DB
	global.DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: newLogger, //打印sql日志
	})
	if err != nil {
		panic(err)
	}
}
```



在main中调用：

```go
package main

import (
	"HiChat/initialize"
)


func main() {
	//初始化数据库
	initialize.InitDB()
}
```



### 初始化日志Zap配置

#### 拉取日志依赖

```
go get go.uber.org/zap
```



#### 日志初始化

在initialize目录中新建一个logger.go文件

```go
package initialize

import (
	"log"

	"go.uber.org/zap"
)

func InitLogger() {
	//初始化日志
	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatal("日志初始化失败", err.Error())
	}
	//使用全局logger
	zap.ReplaceGlobals(logger)

}
```

最后需要在main中调用

```go
func main() {
	//初始化日志
	initialize.InitLogger()
	//初始化数据库
	initialize.InitDB()
}
```



### 总结

到这里整个项目的初始化基本上完成了，主要就是项目目录文件结构的合理配置以及mysql连接池和日志的初始化，后续开始功能模块的开发请参考我的博客：[《从0到1搭建一个IM项目》](https://learnku.com/articles/74274)，感谢您的阅读。



