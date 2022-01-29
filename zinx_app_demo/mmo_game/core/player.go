package core

import (
	"fmt"
	"github.com/alexcd90/czinx/ziface"
	"github.com/alexcd90/czinx/zinx_app_demo/mmo_game/pb"
	"github.com/golang/protobuf/proto"
	"math/rand"
	"sync"
	"time"
)

//玩家对象
type Player struct {
	PID  int32              //玩家ID
	Conn ziface.IConnection //当前玩家的连接
	X    float32            //平面x坐标
	Y    float32            //高度
	Z    float32            //平面y坐标 (注意不是Y)
	V    float32            //旋转0-360度
}

/*
	Player ID生成器
*/
var PidGen int32 = 1  //用来生成玩家ID的计数器
var IdLock sync.Mutex //保护PidGen的互斥机制

//创建一个玩家对象
func NewPlayer(conn ziface.IConnection) *Player {
	//生成一个PID
	IdLock.Lock()
	id := PidGen
	PidGen++
	IdLock.Unlock()

	p := &Player{
		PID:  id,
		Conn: conn,
		X:    float32(160 + rand.Intn(10)), //随机在160坐标点 基于X轴偏移若干坐标
		Y:    0,
		Z:    float32(134 + rand.Intn(17)), //随机在134坐标点 基于Y轴偏移若干坐标
		V:    0,
	}

	return p
}

/*
	发送消息给客户端，
	主要是将pb的protobuf数据序列化之后发送
*/
func (p *Player) SendMsg(msgId uint32, data proto.Message) {
	fmt.Printf("before Marshal data = %+v\n", data)
	//将proto Message结构体序列化
	msg, err := proto.Marshal(data)
	if err != nil {
		fmt.Println("marshal msg err: ", err)
		return
	}
	fmt.Printf("after Marshal data = %+v\n", msg)

	if p.Conn == nil {
		fmt.Println("connection in player is nil")
		return
	}

	//调用Zinx框架的SendMsg发包
	if err := p.Conn.SendMsg(msgId, msg); err != nil {
		fmt.Println("Player SendMsg error !")
		return
	}

}

//告知客户端pid,同步已经生成的玩家ID给客户端
func (p *Player) SyncPID() {
	//组建MsgId0 proto数据
	data := &pb.SyncPID{
		PID: p.PID,
	}
	//发送数据给客户端
	p.SendMsg(1, data)
}

//广播玩家自己的出生地点
func (p *Player) BroadCastStartPosition() {
	//组建MsgID200 proto数据
	msg := &pb.BroadCast{
		PID: p.PID,
		Tp:  2,
		Data: &pb.BroadCast_P{
			P: &pb.Position{
				X: p.X,
				Y: p.Y,
				Z: p.Z,
				V: p.V,
			},
		},
	}

	p.SendMsg(200, msg)
}

//广播玩家聊天
func (p *Player) Talk(content string) {
	//1. 组建MsgId200 proto数据
	msg := &pb.BroadCast{
		PID: p.PID,
		Tp:  1, //TP 1 代表聊天广播
		Data: &pb.BroadCast_Content{
			Content: content,
		},
	}

	//2. 得到当前世界所有的在线玩家
	playes := WorldMgrObj.GetAllPlayers()

	//3. 向所有玩家发送MsgId:200消息
	for _, player := range playes {
		player.SendMsg(200, msg)
	}

}

//给当前玩家周边的(九宫格内)玩家广播自己的位置，让他们显示自己
func (p *Player) SyncSurrounding() {
	//1 获取当前玩家周边全部玩家
	players := p.GetSurroundingPlayers()

	//2.1 组建MsgId200 proto数据
	msg := &pb.BroadCast{
		PID: p.PID,
		Tp:  2, //TP2 代表广播坐标
		Data: &pb.BroadCast_P{
			P: &pb.Position{
				X: p.X,
				Y: p.Y,
				Z: p.Z,
				V: p.V,
			},
		},
	}
	//2.2 每个玩家分别给对应的客户端发送200消息，显示人物
	for _, player := range players {
		player.SendMsg(200, msg)
	}

	//3 让周围九宫格内的玩家出现在自己的视野中
	//3.1 制作Message SyncPlayers 数据
	playerData := make([]*pb.Player, 0, len(players))
	for _, player := range players {
		p := &pb.Player{
			PID: p.PID,
			P: &pb.Position{
				X: player.X,
				Y: player.Y,
				Z: player.Z,
				V: player.V,
			},
		}
		playerData = append(playerData, p)
	}

	//3.2 封装SyncPlayer protobuf数据
	SyncPlayerMsg := &pb.SyncPlayers{
		Ps: playerData[:],
	}

	//3.3 给当前玩家发送需要显示周围的全部玩家数据
	p.SendMsg(202, SyncPlayerMsg)
}

//广播玩家位置移动
func (p *Player) UpdatePos(x float32, y float32, z float32, v float32) {

	//触发消失视野和添加视野业务
	//计算旧格子gID
	oldGID := WorldMgrObj.AoiMgr.GetGIDByPos(p.X, p.Z)
	//计算新格子gID
	newGID := WorldMgrObj.AoiMgr.GetGIDByPos(x, z)
	//更新玩家的位置信息
	p.X = x
	p.Y = y
	p.Z = z
	p.V = v

	if oldGID != newGID {
		//触发gird切换
		//把pID从就的aoi格子中删除
		WorldMgrObj.AoiMgr.RemovePIDFromGrID(int(p.PID), oldGID)
		//把pID添加到新的aoi格子中去
		WorldMgrObj.AoiMgr.AddPidToGrid(int(p.PID), newGID)

		_ = p.OnExchangeAoiGrID(oldGID, newGID)
	}

	//组装protobuf协议，发送位置给周围玩家
	msg := &pb.BroadCast{
		PID: p.PID,
		Tp:  4, //4 - 移动之后的坐标信息
		Data: &pb.BroadCast_P{P: &pb.Position{
			X: p.X,
			Y: p.Y,
			Z: p.Z,
			V: p.V,
		}},
	}

	//获取当前玩家周边全部玩家
	players := p.GetSurroundingPlayers()
	//向周边的每个玩家发送MsgID:200消息，移动位置更新消息
	for _, player := range players {
		player.SendMsg(200, msg)
	}
}

func (p *Player) OnExchangeAoiGrID(oldGID, newGID int) error {
	//获取旧的九宫格成员
	oldGriIDs := WorldMgrObj.AoiMgr.GetSurroundGridsByGid(oldGID)

	//为旧的九宫格成员建立哈希表，用来快速查找
	oldGriIDsMap := make(map[int]bool, len(oldGriIDs))
	for _, griID := range oldGriIDs {
		oldGriIDsMap[griID.GID] = true
	}

	//获取新九宫格成员
	newGriIDs := WorldMgrObj.AoiMgr.GetSurroundGridsByGid(newGID)
	//为新九宫格建立哈希表，用来快速查找
	newGriIDsMap := make(map[int]bool, len(newGriIDs))
	for _, griID := range newGriIDs {
		newGriIDsMap[griID.GID] = true
	}

	//------ > 处理视野消失 <-------
	offlineMsg := &pb.SyncPID{
		PID: p.PID,
	}

	//找到在旧的九宫格中出现，但是在新的九宫格中没有出现的格子
	leavingGrIDs := make([]*Grid, 0)
	for _, grID := range oldGriIDs {
		if _, ok := newGriIDsMap[grID.GID]; !ok {
			leavingGrIDs = append(leavingGrIDs, grID)
		}
	}

	//获取需要消失的格子中的全部玩家
	for _, grID := range leavingGrIDs {
		players := WorldMgrObj.GetPlayersByGID(grID.GID)
		for _, player := range players {
			//让自己在其他玩家的客户端消失
			player.SendMsg(201, offlineMsg)

			//将其他玩家信息 在自己的客户端中消失
			anotherOfflineMsg := &pb.SyncPID{
				PID: player.PID,
			}
			p.SendMsg(201, anotherOfflineMsg)
			time.Sleep(200 * time.Millisecond)
		}
	}

	//------ > 处理视野出现 <-------

	//找到在新的九宫格内出现,但是没有在就的九宫格内出现的格子
	enteringGrIDs := make([]*Grid, 0)
	for _, grID := range newGriIDs {
		if _, ok := oldGriIDsMap[grID.GID]; !ok {
			enteringGrIDs = append(enteringGrIDs, grID)
		}
	}

	onlineMsg := &pb.BroadCast{
		PID: p.PID,
		Tp:  2,
		Data: &pb.BroadCast_P{
			P: &pb.Position{
				X: p.X,
				Y: p.Y,
				Z: p.Z,
				V: p.V,
			}},
	}

	//获取需要显示格子的全部玩家
	for _, grID := range enteringGrIDs {
		players := WorldMgrObj.GetPlayersByGID(grID.GID)

		for _, player := range players {
			//让自己出现在其他人视野中
			player.SendMsg(200, onlineMsg)

			//让其他人出现在自己的视野中
			anotherOnlineMsg := &pb.BroadCast{
				PID: player.PID,
				Tp:  2,
				Data: &pb.BroadCast_P{
					P: &pb.Position{
						X: player.X,
						Y: player.Y,
						Z: player.Z,
						V: player.V,
					}},
			}

			time.Sleep(200 * time.Millisecond)
			p.SendMsg(200, anotherOnlineMsg)
		}
	}

	return nil
}

//获得当前玩家的AOI周边玩家信息
func (p *Player) GetSurroundingPlayers() []*Player {
	//得到当前AOI区域的所有pid
	pids := WorldMgrObj.AoiMgr.GetPIDsByPos(p.X, p.Z)

	//将所有pid对应的Player放到Player切片中
	players := make([]*Player, 0, len(pids))
	for _, pid := range pids {
		players = append(players, WorldMgrObj.Players[int32(pid)])
	}
	return players
}

func (p *Player) LostConnection() {
	//1 获取周围AOI九宫格内的玩机
	players := p.GetSurroundingPlayers()

	//2 封装MsgID:201消息
	msg := &pb.SyncPID{
		PID: p.PID,
	}

	//3 向周围玩家发送消息
	for _, player := range players {
		player.SendMsg(201, msg)
	}

	//4 世界管理器将当前玩家从AOI中摘除
	WorldMgrObj.AoiMgr.RemoveFromGridByPos(int(p.PID), p.X, p.Z)
	WorldMgrObj.RemovePlayerByPid(p.PID)
}
