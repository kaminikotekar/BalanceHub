package Connection

import (
	"fmt"
	"sync"
	"github.com/kaminikotekar/BalanceHub/pkg/Models/RemoteServer"
)

type Connections struct {
	mu                sync.Mutex
	activeConnections map[int]int
}

var ConnectionMap *Connections

func InitConnection(serverIds []int) {
	connect := Connections{
		activeConnections: make(map[int]int),
	}
	for _, sid := range serverIds {
		connect.activeConnections[sid] = 0
	}
	ConnectionMap = &connect
}

func (c *Connections) AddConnection(serverId int) {
	c.mu.Lock()
	c.activeConnections[serverId]++
	c.mu.Unlock()
}

func (c *Connections) RemoveConnection(serverId int) {
	c.mu.Lock()
	c.activeConnections[serverId]--
	c.mu.Unlock()
}

func (c *Connections) GetOptimalServer(servers []int) int {
	c.mu.Lock()
	leastConnection := -1
	var optimalServer int
	for _, sid := range servers {
		connections := c.activeConnections[sid]
		if leastConnection == -1 {
			leastConnection = connections
			optimalServer = sid
			continue
		}
		if connections < leastConnection {
			leastConnection = connections
			optimalServer = sid
		}
	}
	c.mu.Unlock()
	return optimalServer
}

func (c *Connections) ActiveConnections() string {
	str := "{"
	for sid, value := range c.activeConnections {
		remoteServer := RemoteServer.RemoteServerMap.GetServerFromId(sid)
		server := remoteServer.Ipaddress + ":" + remoteServer.Port
		str += fmt.Sprintf("%s : %d, \n", server, value)
	}
	str += "}"
	return str
}
