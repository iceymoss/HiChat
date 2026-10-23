package models

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
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
	MessageID string `json:"messageId" gorm:"-"`
	FormId    int64  `json:"userId"`   //信息发送者
	TargetId  int64  `json:"targetId"` //信息接收者
	Type      int    //聊天类型：群聊 私聊 广播
	Media     int    //信息类型：文字 图片 音频
	Content   string //消息内容
	Pic       string `json:"url"` //图片相关
	Url       string //文件相关
	Desc      string //文件描述
	Amount    int    //其他数据大小
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
	Done      chan struct{}
	closeOnce sync.Once
}

// 映射关系
var clientMap map[int64]*Node = make(map[int64]*Node, 0)

// 读写锁
var rwLocker sync.RWMutex

// Chat	需要 ：发送者ID ，接受者ID ，消息类型，发送的内容，发送类型
func Chat(w http.ResponseWriter, r *http.Request, userID int64) {
	//升级为socket
	conn, err := (&websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			if origin == "" {
				return true
			}
			parsedOrigin, err := url.Parse(origin)
			return err == nil && parsedOrigin.Host == r.Host
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
		Done:      make(chan struct{}),
	}

	// 每个用户只保留一个连接，避免旧连接继续接收私信。
	rwLocker.Lock()
	oldNode := clientMap[userID]
	clientMap[userID] = node
	rwLocker.Unlock()
	if oldNode != nil {
		_ = oldNode.Conn.Close()
	}

	//发送消息
	go sendProc(node)
	//接收消息
	go recProc(node, userID)
	sendMsg(userID, []byte("欢迎进入聊天系统"))
}

func sendProc(node *Node) {
	for {
		select {
		case data := <-node.DataQueue:
			err := node.Conn.WriteMessage(websocket.TextMessage, data)
			if err != nil {
				zap.S().Info("写入消息失败", err)
				_ = node.Conn.Close()
				return
			}
			fmt.Println("数据发送socket成功")
		case <-node.Done:
			return
		}
	}
}

func recProc(node *Node, userID int64) {
	defer func() {
		node.closeOnce.Do(func() { close(node.Done) })
		rwLocker.Lock()
		if clientMap[userID] == node {
			delete(clientMap, userID)
		}
		rwLocker.Unlock()
		_ = node.Conn.Close()
	}()

	for {
		//获取信息
		_, data, err := node.Conn.ReadMessage()
		if err != nil {
			zap.S().Info("读取消息失败", err)
			return
		}

		data, err = authenticatedMessage(data, userID)
		if err != nil {
			zap.S().Info("消息解析失败", err)
			continue
		}
		dispatch(data)

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

func authenticatedMessage(data []byte, userID int64) ([]byte, error) {
	msg := Message{}
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, err
	}
	if msg.Type == 0 {
		msg.FormId = userID
		return json.Marshal(msg)
	}
	if msg.TargetId <= 0 || (msg.Type != 1 && msg.Type != 2) {
		return nil, fmt.Errorf("消息参数不合法")
	}
	messageID := make([]byte, 16)
	if _, err := rand.Read(messageID); err != nil {
		return nil, err
	}
	msg.MessageID = fmt.Sprintf("%x", messageID)
	msg.FormId = userID
	return json.Marshal(msg)
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
	if !IsGroupMember(formId, target) {
		return -1, fmt.Errorf("用户不是群成员")
	}
	if err := saveRedisMessage(groupMessageKey(target), data); err != nil {
		return -1, err
	}

	//群发的逻辑：1获取到群里所有用户，然后向除开自己的每一位用户发送消息
	userIDs, err := FindUsers(target)
	if err != nil {
		return -1, err
	}

	for _, userId := range *userIDs {
		if formId != userId {
			sendMsg(int64(userId), data)
		}
	}
	return 0, nil
}

// sendMs 向用户发送消息
func sendMsg(id int64, msg []byte) {
	rwLocker.RLock()
	node, ok := clientMap[id]
	rwLocker.RUnlock()

	if !ok {
		zap.S().Info("userID没有对应的node")
		return
	}

	zap.S().Info("targetID:", id, "node:", node)
	select {
	case node.DataQueue <- msg:
	case <-node.Done:
	}
}

// sendMsgTest 发送消息 并存储聊天记录到redis
func sendMsgAndSave(userId int64, msg []byte) {

	rwLocker.RLock()
	node, ok := clientMap[userId] //对方是否在线
	rwLocker.RUnlock()

	jsonMsg := Message{}
	json.Unmarshal(msg, &jsonMsg)
	targetIdStr := strconv.Itoa(int(userId))
	userIdStr := strconv.Itoa(int(jsonMsg.FormId))

	//如果在线，需要即时推送
	if ok {
		select {
		case node.DataQueue <- msg:
		case <-node.Done:
		}
	}

	//拼接记录名称
	var key string
	if userId > jsonMsg.FormId {
		key = "msg_" + userIdStr + "_" + targetIdStr
	} else {
		key = "msg_" + targetIdStr + "_" + userIdStr
	}

	if err := saveRedisMessage(key, msg); err != nil {
		zap.S().Info("保存聊天记录失败", err)
	}
}

func saveRedisMessage(key string, msg []byte) error {
	ctx := context.Background()
	score, err := global.RedisDB.Incr(ctx, key+"_seq").Result()
	if err != nil {
		return err
	}
	if err := global.RedisDB.ZAdd(ctx, key, &redis.Z{Score: float64(score), Member: msg}).Err(); err != nil {
		return err
	}
	expiration := 7 * 24 * time.Hour
	pipe := global.RedisDB.Pipeline()
	pipe.Expire(ctx, key, expiration)
	pipe.Expire(ctx, key+"_seq", expiration)
	if _, err := pipe.Exec(ctx); err != nil {
		return err
	}
	addKeyToSaveKey(key)
	return nil
}

func groupMessageKey(groupID uint) string {
	return "group_msg_" + strconv.FormatUint(uint64(groupID), 10)
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

func GroupRedisMsg(groupID uint, start, end int64, isRev bool) []string {
	if start < 0 || end < start {
		return []string{}
	}

	ctx := context.Background()
	redisCount, err := global.RedisDB.ZCard(ctx, groupMessageKey(groupID)).Result()
	if err != nil {
		zap.S().Info("获取群聊天记录数量失败", err)
		redisCount = 0
	}

	if isRev {
		return groupMessagesAscending(groupID, start, end, redisCount)
	}
	return groupMessagesDescending(groupID, start, end, redisCount)
}

func groupMessagesDescending(groupID uint, start, end, redisCount int64) []string {
	ctx := context.Background()
	result := make([]string, 0, end-start+1)
	if start < redisCount {
		redisEnd := end
		if redisEnd >= redisCount {
			redisEnd = redisCount - 1
		}
		records, err := global.RedisDB.ZRevRange(ctx, groupMessageKey(groupID), start, redisEnd).Result()
		if err != nil {
			zap.S().Info("获取群聊天记录失败", err)
		} else {
			result = append(result, records...)
		}
	}
	if end >= redisCount {
		dbStart := start - redisCount
		if dbStart < 0 {
			dbStart = 0
		}
		result = append(result, databaseGroupMessages(groupID, dbStart, end-redisCount, true)...)
	}
	return result
}

func groupMessagesAscending(groupID uint, start, end, redisCount int64) []string {
	var databaseCount int64
	if err := global.DB.Table("messages").Where("type = ? and target_id = ?", 2, groupID).Count(&databaseCount).Error; err != nil {
		zap.S().Info("获取已持久化群聊天记录数量失败", err)
		return []string{}
	}

	result := make([]string, 0, end-start+1)
	if start < databaseCount {
		databaseEnd := end
		if databaseEnd >= databaseCount {
			databaseEnd = databaseCount - 1
		}
		result = append(result, databaseGroupMessages(groupID, start, databaseEnd, false)...)
	}
	if end >= databaseCount && redisCount > 0 {
		redisStart := start - databaseCount
		if redisStart < 0 {
			redisStart = 0
		}
		redisEnd := end - databaseCount
		if redisEnd >= redisCount {
			redisEnd = redisCount - 1
		}
		records, err := global.RedisDB.ZRange(context.Background(), groupMessageKey(groupID), redisStart, redisEnd).Result()
		if err != nil {
			zap.S().Info("获取群聊天记录失败", err)
		} else {
			result = append(result, records...)
		}
	}
	return result
}

func databaseGroupMessages(groupID uint, start, end int64, descending bool) []string {
	if end < start {
		return []string{}
	}
	order := "id ASC"
	if descending {
		order = "id DESC"
	}
	messages := make([]Message, 0, end-start+1)
	if err := global.DB.Table("messages").
		Where("type = ? and target_id = ?", 2, groupID).
		Order(order).
		Offset(int(start)).
		Limit(int(end - start + 1)).
		Find(&messages).Error; err != nil {
		zap.S().Info("获取已持久化群聊天记录失败", err)
		return []string{}
	}

	result := make([]string, 0, len(messages))
	for _, message := range messages {
		data, err := json.Marshal(message)
		if err != nil {
			zap.S().Info("群聊天记录编码失败", err)
			continue
		}
		result = append(result, string(data))
	}
	return result
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

					if len(msg) <= 0 {
						tx.Rollback()
						return
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
	for _, savedKey := range saveKey {
		if savedKey == key {
			return
		}
	}
	saveKey = append(saveKey, key)
}

// 获取当前的 saveKey 副本的函数，保护了 saveKey 的并发访问
func getSaveKey() []string {
	saveKeyMutex.Lock()
	defer saveKeyMutex.Unlock()
	return append([]string(nil), saveKey...)
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
