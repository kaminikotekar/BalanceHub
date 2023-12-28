package RemoteServer

import (
	"fmt"
)

var RemoteServerMap *Map

type Server struct {
	Ipaddress string
	Port string
	PathConsrt int
	IpConsrt int
	// AllowedIPs []string
	// Paths []string
}

type Map struct {

	serverMap map[int]*Server
	pathMap map[string]map[int]*Server
	ipMap map[string]map[int]*Server

}

func GenerateMap() {
	var localMap Map
	localMap.serverMap = make(map[int]*Server)
	localMap.pathMap = make(map[string]map[int]*Server)
	localMap.ipMap = make(map[string]map[int]*Server)
	RemoteServerMap = &localMap
	// return &localMap
}

func (m* Map) AddServer(serverId int, ipaddress string, port string) {
	_, ok := m.serverMap[serverId]
	if !ok {
		m.serverMap[serverId] = &Server{
			Ipaddress: ipaddress,
			Port: port,
		}
	}
}

func (m * Map) RemoveServer(serverId int) {
	delete(m.serverMap, serverId)
	// Delete serverID from pathmap
	for path, _ := range m.pathMap {
		_, ok := m.pathMap[path][serverId]
		if ok {
			delete(m.pathMap[path], serverId)
		}
	}
	// Delete serverID from clientMap
	for client, _ := range m.ipMap {
		_, ok := m.ipMap[client][serverId]
		if ok {
			delete(m.ipMap[client], serverId)
		}
	}
}

func (m* Map) GetServerFromId(id int) *Server {
	return m.serverMap[id]
}

func (m *Map) hasPath(path string) bool {
	_, err := m.pathMap[path]

	if len(m.pathMap) > 0{
		if err {
			return false
		}
	}
	return true
}

func (m *Map) DeletePath(path string, serverId int){
	_, ok := m.pathMap[path][serverId]
	if !ok {
		return 
	}
	m.serverMap[serverId].PathConsrt -= 1
	delete(m.pathMap[path], serverId)
}

func (m *Map) DeleteClient(clientIp string, serverId int){
	_, ok := m.ipMap[clientIp][serverId]
	if !ok {
		return 
	}
	m.serverMap[serverId].IpConsrt -= 1
	delete(m.ipMap[clientIp], serverId)
}

func(m *Map) UpdatePath(path string, serverid int) {

	// p.pathmap[path] = append(p.pathmap[path],server)
	fmt.Println("Inside update path")
	server := m.serverMap[serverid]
	val, ok := m.pathMap[path]
	fmt.Println("Value:", val, "Err ", ok)

	if !ok{
		m.pathMap[path] = make(map[int]*Server)
	}
	_, ok = m.pathMap[path][serverid]
	if !ok{
		server.PathConsrt+= 1
	}

	m.pathMap[path][serverid] = server
}

func (m *Map) isAllowedIP(ipaddress string) bool {

	_, err := m.ipMap[ipaddress]

	if len(m.ipMap) > 0{
		if err {
			return false
		}
	}
	return true
}

func (m *Map) UpdateClientIP(clientIp string, serverid int) {
	fmt.Println("Inside update client IP")
	server := m.serverMap[serverid]
	val, ok := m.ipMap[clientIp]
	fmt.Println("Value:", val, "Err ", ok)

	if !ok{
		m.ipMap[clientIp] = make(map[int]*Server)
	}

	_, ok = m.ipMap[clientIp][serverid]
	if !ok{
		server.IpConsrt+= 1
	}
	m.ipMap[clientIp][serverid] = server
}

func (m *Map) GetPossibleServers(clientIp string, path string) ([]int) {
	servers := make([]int, 0, len(m.serverMap))

	for k, server := range m.serverMap {
		_, res1 := m.pathMap[path][k]
		_, res2 := m.ipMap[clientIp][k]
		if server.PathConsrt == 0 && server.IpConsrt == 0 {
			servers = append(servers, k)
		} else if server.IpConsrt == 0 {
			if res1 {
				servers = append(servers, k)
			}
		} else if server.PathConsrt == 0 {
			if res2 {
				servers = append(servers, k)
			}
		} else {
			if res1 && res2 {
				servers = append(servers, k)
			}
		}
	}
	return servers
}

func (m *Map) GetServerIds() []int {
	keys := make([]int, 0, len(m.serverMap))

	for k, _ := range m.serverMap {
		keys = append(keys, k)
		// values = append(values, v)
	}
	return keys
}










