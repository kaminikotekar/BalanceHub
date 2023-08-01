package Connection

import (
	"fmt"
	"sync"
	"github.com/kaminikotekar/BalanceHub/Server"
)

type Connections struct {
	mu sync.Mutex
	ActiveConnections map[Server] int
}

func (c *Connections) AddConnection(server Server) {
	c.mu.Lock()
	c.ActiveConnections[server]++
	c.mu.Unlock()
}

func (c *Connections) RemoveConnection(server Server) {
	c.mu.Unlock()
	c.ActiveConnections[server]--
	c.mu.Unlock()
}

func (c *Connections) PrintConnections() {
	fmt.Println("Connections ", c.ActiveConnections)
}