package Connection

import (
	"fmt"
	"sync"
	"github.com/kaminikotekar/BalanceHub/pkg/Config"
)

type Connections struct {
	mu sync.Mutex
	ActiveConnections map[Config.Server] int
}

func (c *Connections) InitializeLoadServers(remoteServers []Config.Server) {

	for _, s := range remoteServers {
		c.ActiveConnections[s] = 0
	}
}

func (c *Connections) AddConnection(server Config.Server) {
	c.mu.Lock()
	c.ActiveConnections[server]++
	c.mu.Unlock()
}

func (c *Connections) RemoveConnection(server Config.Server) {
	c.mu.Lock()
	c.ActiveConnections[server]--
	c.mu.Unlock()
}

func (c *Connections) GetOptimalServer() Config.Server {
	c.mu.Lock()
	leastConnection := -1
	// var optimalServer Config.Server
	optimalServer := Config.Server{
			Ipaddress: "None",  
			Port: "None"}
	for s, connections := range c.ActiveConnections{
		if leastConnection == -1{
			leastConnection = connections
			optimalServer.Ipaddress = s.Ipaddress
			optimalServer.Port = s.Port
			continue
		}
		if connections < leastConnection{
			leastConnection = connections
			optimalServer.Ipaddress = s.Ipaddress
			optimalServer.Port = s.Port
		}
	}
	fmt.Println("Least Connections :", leastConnection)
	c.mu.Unlock()
	return optimalServer
}

func (c *Connections) PrintConnections() {
	fmt.Println("Connections ", c.ActiveConnections)
}
