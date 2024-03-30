package RemoteServer

import (
	"log"
	"net"
	"strings"
	"sync"
)

var RemoteServerMap *Map

type Server struct {
	Ipaddress  string
	Port       string
	PathConsrt int
	IpConsrt   int
	// AllowedIPs []string
	// Paths []string
}

type Map struct {
	mu        sync.Mutex
	serverMap map[int]*Server
	pathMap   map[string]map[int]*Server
	ipMap     map[string]map[int]*Server
}

func GenerateMap() {
	var localMap Map
	localMap.serverMap = make(map[int]*Server)
	localMap.pathMap = make(map[string]map[int]*Server)
	localMap.ipMap = make(map[string]map[int]*Server)
	RemoteServerMap = &localMap
	// return &localMap
}

func (m *Map) AddServer(serverId int, ipaddress string, port string) {
	m.mu.Lock()
	_, ok := m.serverMap[serverId]
	if !ok {
		m.serverMap[serverId] = &Server{
			Ipaddress: ipaddress,
			Port:      port,
		}
	}
	m.mu.Unlock()
}

func (m *Map) RemoveServer(serverId int) {
	m.mu.Lock()
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
	m.mu.Unlock()
}

func (m *Map) GetServerFromId(id int) *Server {
	return m.serverMap[id]
}

func (m *Map) hasPath(path string) bool {
	_, err := m.pathMap[path]

	if len(m.pathMap) > 0 {
		if err {
			return false
		}
	}
	return true
}

func (m *Map) DeletePath(path string, serverId int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.pathMap[path][serverId]
	if !ok {
		return
	}
	m.serverMap[serverId].PathConsrt -= 1
	delete(m.pathMap[path], serverId)
}

func (m *Map) DeleteClient(clientIp string, serverId int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.ipMap[clientIp][serverId]
	if !ok {
		return
	}
	m.serverMap[serverId].IpConsrt -= 1
	delete(m.ipMap[clientIp], serverId)
}

func (m *Map) UpdatePath(path string, serverid int) {

	// p.pathmap[path] = append(p.pathmap[path],server)
	m.mu.Lock()
	server := m.serverMap[serverid]
	_, ok := m.pathMap[path]

	if !ok {
		m.pathMap[path] = make(map[int]*Server)
	}
	_, ok = m.pathMap[path][serverid]
	if !ok {
		server.PathConsrt += 1
	}

	m.pathMap[path][serverid] = server
	m.mu.Unlock()
}

func (m *Map) isAllowedIP(ipaddress string) bool {

	_, err := m.ipMap[ipaddress]

	if len(m.ipMap) > 0 {
		if err {
			return false
		}
	}
	return true
}

func (m *Map) UpdateClientIP(clientIp string, serverid int) {
	m.mu.Lock()
	server := m.serverMap[serverid]
	_, ok := m.ipMap[clientIp]

	if !ok {
		m.ipMap[clientIp] = make(map[int]*Server)
	}

	_, ok = m.ipMap[clientIp][serverid]
	if !ok {
		server.IpConsrt += 1
	}
	m.ipMap[clientIp][serverid] = server
	m.mu.Unlock()
}

func IpInSubnet(ip, subnet string) bool {
	ipAddr := net.ParseIP(ip)
	_, subnetIPNet, err := net.ParseCIDR(subnet)
	if err != nil {
		log.Println("Error parsing subnet:", err)
		return false
	}

	return subnetIPNet.Contains(ipAddr)
}

func (m *Map) GetPossibleServers(clientIp string, path string) []int {
	servers := make([]int, 0, len(m.serverMap))
	subnetList := make([]string, 0, len(m.ipMap))

	for key, _ := range m.ipMap {
		if strings.Contains(key, "/") && IpInSubnet(clientIp, key) {
			subnetList = append(subnetList, key) // works
			// subnetKey = key
		} else if key == clientIp {
			subnetList = append(subnetList, key)
			// subnetKey = key
		}

	}
	for k, server := range m.serverMap {
		_, res1 := m.pathMap[path][k]
		var res2 bool
		for _, subnetKey := range subnetList {
			_, res2 = m.ipMap[subnetKey][k]
			if res2 {
				break
			}
		}
		// _, res2 := m.ipMap[subnetKey][k]
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
