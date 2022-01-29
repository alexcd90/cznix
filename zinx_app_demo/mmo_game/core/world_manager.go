package core

import "sync"

/*
	当前游戏世界的总管理模块
*/
type WorldManager struct {
	AoiMgr  *AOIManager       //当前世界地图的AOI规划管理器
	Players map[int32]*Player //当前在线的玩家集合
	PLock   sync.RWMutex      //保护Players的互斥读写机制
}

//提供一个对外的世界管理模块句柄
var WorldMgrObj *WorldManager

//提供WorldManager初始化方法
func init() {
	WorldMgrObj = &WorldManager{
		AoiMgr:  NewAOIManager(AOI_MIN_X, AOI_MAX_X, AOI_CNTS_X, AOI_MIN_Y, AOI_MAX_Y, AOI_CNTS_Y),
		Players: make(map[int32]*Player),
	}
}

//提供添加一个玩家的的功能，将玩家添加进玩家信息表Players
func (wm *WorldManager) AddPlayer(player *Player) {
	//将player添加到 世界管理器中
	wm.PLock.Lock()
	wm.Players[player.PID] = player
	wm.PLock.Unlock()

	//将player 添加到AOI网络规划中
	wm.AoiMgr.AddToGridByPos(int(player.PID), player.X, player.Z)
}

//从玩家信息表中移除一个玩家
func (wm *WorldManager) RemovePlayerByPid(pid int32) {
	wm.PLock.Lock()
	delete(wm.Players, pid)
	wm.PLock.Unlock()
}

//通过玩家ID 获取对应玩家信息
func (wm *WorldManager) GetPlayerByPid(pid int32) *Player {
	wm.PLock.RLock()
	defer wm.PLock.RUnlock()

	return wm.Players[pid]
}

//获取所有玩家的信息
func (wm *WorldManager) GetAllPlayers() []*Player {
	wm.PLock.RLock()
	defer wm.PLock.RUnlock()

	//创建返回的player集合切片
	players := make([]*Player, 0)

	//添加切片
	for _, v := range wm.Players {
		players = append(players, v)
	}

	return players
}

//获取指定gID中的所有player信息
func (wm *WorldManager) GetPlayersByGID(gID int) []*Player {
	//通过gID获取 对应 格子中的所有pID
	pIDs := wm.AoiMgr.grids[gID].GetPlyerIDs()

	//通过pID找到对应的player对象
	players := make([]*Player, 0, len(pIDs))
	wm.PLock.RLock()
	for _, pID := range pIDs {
		players = append(players, wm.Players[int32(pID)])
	}
	wm.PLock.RUnlock()

	return players
}
