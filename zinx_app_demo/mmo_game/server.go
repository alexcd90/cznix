package main

import (
	"fmt"
	"github.com/alexcd90/czinx/ziface"
	"github.com/alexcd90/czinx/zinx_app_demo/mmo_game/core"
	"github.com/alexcd90/czinx/znet"
)

//当客户端建立连接的时候的hook函数
func OnConnectionAdd(conn ziface.IConnection) {
	//创建一个玩家
	player := core.NewPlayer(conn)
	//同步当前的PlayerID给客户端，走MsgID:1消息
	player.SyncPID()
	//同步当前玩家的初始化坐标信息给客户端，走MsgID:200消息
	player.BroadCastStartPosition()
	//将当前上线玩家添加到world manager中
	core.WorldMgrObj.AddPlayer(player)

	fmt.Println("=====> Player pidId = ", player.PID, " arrived ====")
}

func main() {
	//创建服务器句柄
	s := znet.NewServer()

	//注册客户端连接建立和丢失函数
	s.SetOnConnStart(OnConnectionAdd)

	//启动服务
	s.Serve()
}
