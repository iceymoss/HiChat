package models

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"sync"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	"gopkg.in/fatih/set.v0"
)

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

//Node 构造连接
type Node struct {
	Conn      *websocket.Conn //连接
	Addr      string          //客户端地址
	DataQueue chan []byte     //消息
	GroupSets set.Interface   //好友 / 群
}

//映射关系
var clientMap map[int64]*Node = make(map[int64]*Node, 0)

//读写锁
var rwLocker sync.RWMutex

//Chat	需要 ：发送者ID ，接受者ID ，消息类型，发送的内容，发送类型
func Chat(w http.ResponseWriter, r *http.Request) {
	//1.  获取参数 并 检验 token 等合法性
	query := r.URL.Query()
	fmt.Println("handle:", query)
	Id := query.Get("userId")
	//token := query.Get("token")

	//targeId := query.Get("targetId")

	fmt.Println("uID", Id)

	//tarID, err := strconv.ParseInt(targeId, 10, 64)
	//if err != nil {
	//	zap.S().Info("类型转换失败", err)
	//	return
	//}

	//content := query.Get("content")
	//chatTyep := query.Get("chatType")
	//msgType := query.Get("type")

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
	//todo

	//将userId和Node绑定
	rwLocker.Lock()
	clientMap[userId] = node
	rwLocker.Unlock()

	fmt.Println("uid", userId)

	//发送接收消息
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

		dispatch(data)

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

//UdpSendProc 完成upd数据发送
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

//UpdRecProc 完成udp数据的接收
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

	fmt.Println("解析数据:", msg, "msg.FormId", msg.FormId, "targetId:", msg.TargetId, "type:", msg.Type)

	//判断消息类型
	switch msg.Type {
	case 1: //私聊
		sendMsg(msg.TargetId, data)
	case 2: //群发
		sendGroupMsg()
	}
}

//sendMs 向用户发送消息
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

func sendGroupMsg() {}
