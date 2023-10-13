package models

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"HiChat/global"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	"gopkg.in/fatih/set.v0"
)

// 记录用户聊天key
var saveKeyMutex sync.Mutex
var saveKey []string

type Message struct {
	Model
	FormId   int64  `json:"userId"`   //信息发送者
	TargetId int64  `json:"targetId"` //信息接收者
	Type     int    //聊天类型：群聊 私聊 广播
	Media    int    //信息类型：文字 图片 音频
	Content  string //消息内容
	Pic      string `json:"url"` //图片相关
	Url      string //文件相关
	Desc     string //文件描述
	Amount   int    //其他数据大小
}

func (m *Message) MsgTableName() string {
	return "message"
}

// Node 构造连接
type Node struct {
	Conn      *websocket.Conn //连接
	Addr      string          //客户端地址
	DataQueue chan []byte     //消息
	GroupSets set.Interface   //好友 / 群
}

// 映射关系
var clientMap map[int64]*Node = make(map[int64]*Node, 0)

// 读写锁
var rwLocker sync.RWMutex

// Chat	需要 ：发送者ID ，接受者ID ，消息类型，发送的内容，发送类型
func Chat(w http.ResponseWriter, r *http.Request) {
	//1.  获取参数 并 检验 token 等合法性
	query := r.URL.Query()
	fmt.Println("handle:", query)
	Id := query.Get("userId")
	//token := query.Get("token")

	userId, err := strconv.ParseInt(Id, 10, 64)
	if err != nil {
		zap.S().Info("类型转换失败", err)
		return
	}

	//升级为socket
	var isvalida = true
	conn, err := (&websocket.Upgrader{
		//token 校验
		CheckOrigin: func(r *http.Request) bool {
			return isvalida
		},
	}).Upgrade(w, r, nil)
	if err != nil {
		fmt.Println(err)
		return
	}

	//获取socket连接,构造消息节点
	node := &Node{
		Conn:      conn,
		DataQueue: make(chan []byte, 50),
		GroupSets: set.New(set.ThreadSafe),
	}

	//用户关系
	//将userId和Node绑定
	rwLocker.Lock()
	clientMap[userId] = node
	rwLocker.Unlock()

	fmt.Println("uid", userId)

	//发送消息
	go sendProc(node)
	//接收消息
	go recProc(node)
	sendMsg(userId, []byte("欢迎进入聊天系统"))
}

func sendProc(node *Node) {
	for {
		select {
		case data := <-node.DataQueue:
			err := node.Conn.WriteMessage(websocket.TextMessage, data)
			if err != nil {
				zap.S().Info("写入消息失败", err)
				return
			}
			fmt.Println("数据发送socket成功")
		}
	}
}

func recProc(node *Node) {
	for {
		//获取信息
		_, data, err := node.Conn.ReadMessage()
		if err != nil {
			zap.S().Info("读取消息失败", err)
			return
		}

		brodMsg(data)

		//这里是简单实现的一种方法
		//msg := Message{}
		//err = json.Unmarshal(data, &msg)
		//if err != nil {
		//	zap.S().Info("json解析失败", err)
		//	return
		//}
		//
		//if msg.Type == 1 {
		//	zap.S().Info("这是一条私信:", msg.Content)
		//	tarNode, ok := clientMap[msg.TargetId]
		//	if !ok {
		//		zap.S().Info("不存在对应的node", msg.TargetId)
		//		return
		//	}
		//
		//	tarNode.DataQueue <- data
		//	fmt.Println("发送成功：", string(data))
		//}
	}
}

var upSendChan chan []byte = make(chan []byte, 1024)

func brodMsg(data []byte) {
	upSendChan <- data
}

func init() {
	go UdpSendProc()
	go UpdRecProc()
}

// UdpSendProc 完成upd数据发送
func UdpSendProc() {
	udpConn, err := net.DialUDP("udp", nil, &net.UDPAddr{
		//192.168.31.147
		IP:   net.IPv4(127, 0, 0, 1),
		Port: 3000,
		Zone: "",
	})
	if err != nil {
		zap.S().Info("拨号udp端口失败", err)
		return
	}

	defer udpConn.Close()

	for {
		select {
		case data := <-upSendChan:
			_, err := udpConn.Write(data)
			if err != nil {
				zap.S().Info("写入udp消息失败", err)
				return
			}
			fmt.Println("数据成功发送到udp服务端:", string(data))
		}
	}

}

// UpdRecProc 完成udp数据的接收
func UpdRecProc() {
	udpConn, err := net.ListenUDP("udp", &net.UDPAddr{
		IP:   net.IPv4(127, 0, 0, 1),
		Port: 3000,
	})
	if err != nil {
		zap.S().Info("监听udp端口失败", err)
		return
	}

	defer udpConn.Close()

	for {
		var buf [1024]byte
		n, err := udpConn.Read(buf[0:])
		if err != nil {
			zap.S().Info("读取udp数据失败", err)
			return
		}

		fmt.Println("udp服务端接收udp数据", buf[0:n])

		//处理发送逻辑
		dispatch(buf[0:n])
	}
}

func dispatch(data []byte) {
	//解析消息
	msg := Message{}
	err := json.Unmarshal(data, &msg)
	if err != nil {
		zap.S().Info("消息解析失败", err)
		return
	}

	fmt.Println("解析数据:", "msg.FormId", msg.FormId, "targetId:", msg.TargetId, "type:", msg.Type)
	//fmt.Println("消息类型：", msg.Type, msg.Content)
	//判断消息类型
	switch msg.Type {
	case 1: //私聊
		sendMsgAndSave(msg.TargetId, data)
	case 2: //群发
		sendGroupMsg(uint(msg.FormId), uint(msg.TargetId), data)
	}
}

// sendGroupMsg 群发
func sendGroupMsg(formId, target uint, data []byte) (int, error) {
	//群发的逻辑：1获取到群里所有用户，然后向除开自己的每一位用户发送消息
	userIDs, err := FindUsers(target)
	if err != nil {
		return -1, err
	}

	for _, userId := range *userIDs {
		if formId != userId {
			sendMsgAndSave(int64(userId), data)
		}
	}
	return 0, nil
}

// sendMs 向用户发送消息
func sendMsg(id int64, msg []byte) {
	rwLocker.Lock()
	node, ok := clientMap[id]
	rwLocker.Unlock()

	if !ok {
		zap.S().Info("userID没有对应的node")
		return
	}

	zap.S().Info("targetID:", id, "node:", node)
	if ok {
		node.DataQueue <- msg
	}
}

// sendMsgTest 发送消息 并存储聊天记录到redis
func sendMsgAndSave(userId int64, msg []byte) {

	rwLocker.RLock()
	node, ok := clientMap[userId] //对方是否在线
	rwLocker.RUnlock()

	jsonMsg := Message{}
	json.Unmarshal(msg, &jsonMsg)
	ctx := context.Background()
	targetIdStr := strconv.Itoa(int(userId))
	userIdStr := strconv.Itoa(int(jsonMsg.FormId))

	//如果在线，需要即时推送
	if ok {
		node.DataQueue <- msg
	}

	//拼接记录名称
	var key string
	if userId > jsonMsg.FormId {
		key = "msg_" + userIdStr + "_" + targetIdStr
	} else {
		key = "msg_" + targetIdStr + "_" + userIdStr
	}

	//创建记录
	res, err := global.RedisDB.ZRevRange(ctx, key, 0, -1).Result()
	if err != nil {
		fmt.Println(err)
		return
	}

	//将聊天记录写入数据库
	score := float64(cap(res)) + 1
	ress, e := global.RedisDB.ZAdd(ctx, key, &redis.Z{score, msg}).Result() //jsonMsg
	//res, e := utils.Red.Do(ctx, "zadd", key, 1, jsonMsg).Result() //备用 后续拓展 记录完整msg
	if e != nil {
		fmt.Println(e)
		return
	}

	// 设置ZSET的过期时间为10天
	expirationTime := 24 * time.Hour * 7
	_, expireErr := global.RedisDB.Expire(ctx, key, expirationTime).Result()
	if expireErr != nil {
		fmt.Println(expireErr)
		return
	}

	fmt.Println("ZSET的过期时间已设置为10天")
	fmt.Println(ress)

	//将key放入全局并
	addKeyToSaveKey(key)
}

// MarshalBinary 需要重写此方法才能完整的msg转byte[]
func (msg Message) MarshalBinary() ([]byte, error) {
	return json.Marshal(msg)
}

// RedisMsg 获取缓存里面的消息
func RedisMsg(userIdA int64, userIdB int64, start int64, end int64, isRev bool) []string {
	ctx := context.Background()
	userIdStr := strconv.Itoa(int(userIdA))
	targetIdStr := strconv.Itoa(int(userIdB))

	//拼接key
	var key string
	if userIdA > userIdB {
		key = "msg_" + targetIdStr + "_" + userIdStr
	} else {
		key = "msg_" + userIdStr + "_" + targetIdStr
	}

	var rels []string
	var err error
	if isRev {
		rels, err = global.RedisDB.ZRange(ctx, key, start, end).Result()
	} else {
		rels, err = global.RedisDB.ZRevRange(ctx, key, start, end).Result()
	}
	if err != nil {
		fmt.Println(err) //没有找到
	}
	fmt.Println("聊天记录：", rels)
	return rels
}

// RecordPersistence 聊天记录持久化
func RecordPersistence() {
	ctx := context.Background()

	//启动定时任务，每隔24小时执行一次
	interval := 24 * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// 创建一个通道用于控制并发，控制10个并发
	concurrency := make(chan struct{}, 10)

	zap.S().Info("----------持久化开始------------")
	for {
		//多路复用，定时获取数据
		select {
		case <-ticker.C:
			zap.S().Info("持久化中")
			// 处理数据持久化
			msgKeys := getSaveKey()
			go func(keys []string) {

				// 控制并发，10并发缓冲区满，G陷入阻塞
				concurrency <- struct{}{}
				defer func() {

					//正常的G完成后，消费缓冲区
					<-concurrency
				}()

				for _, key := range keys {
					chatRecords, err := global.RedisDB.ZRange(ctx, key, 0, -1).Result()
					if err != nil {
						zap.S().Info("从Redis获取聊天记录失败:", err)
						return
					}

					zap.S().Info("----------持久化中------------")
					//存储需要持久化的记录
					msg := make([]Message, 0)

					//持久化失败的记录归还
					backMsg := make([]string, 0)

					//开启事务
					tx := global.DB.Begin()

					for _, record := range chatRecords {
						m := Message{}
						err := json.Unmarshal([]byte(record), &m)
						if err != nil {
							zap.S().Info("Unmarshal fail", err.Error())

							//回滚事务
							tx.Rollback()
							return
						}
						msg = append(msg, m)
						backMsg = append(backMsg, record)
					}

					if err := tx.Table("messages").Save(msg).Error; err != nil {
						zap.S().Info("持久化失败", err.Error())

						//回滚事务
						tx.Rollback()
						return
					}

					for _, bg := range backMsg {
						if err := global.RedisDB.ZRem(ctx, key, bg).Err(); err != nil {
							fmt.Println("从Redis删除聊天记录失败:", err)

							//回滚事务
							tx.Rollback()
							return
						}
					}
					//提交事务
					tx.Commit()

					//持久化成功后移除对应key
					removeKeyFromSaveKey(key)
					zap.S().Info("----------持久化成功------------")
				}
			}(msgKeys)
		}
	}
}

// 向 saveKey 中添加 key 的函数，保护了 saveKey 的并发访问
func addKeyToSaveKey(key string) {
	saveKeyMutex.Lock()
	defer saveKeyMutex.Unlock()
	saveKey = append(saveKey, key)
}

// 获取当前的 saveKey 副本的函数，保护了 saveKey 的并发访问
func getSaveKey() []string {
	saveKeyMutex.Lock()
	defer saveKeyMutex.Unlock()
	return saveKey
}

// 从 saveKey 中删除 key 的函数，保护了 saveKey 的并发访问
func removeKeyFromSaveKey(key string) {
	saveKeyMutex.Lock()
	defer saveKeyMutex.Unlock()
	for i, k := range saveKey {
		if k == key {
			saveKey = append(saveKey[:i], saveKey[i+1:]...)
			break
		}
	}
}
