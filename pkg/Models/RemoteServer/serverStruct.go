package RemoteServer

import (
	"fmt"
)

type Server struct {
	Ipaddress string
	Port string
	// AllowedIPs []string
	// Paths []string
}

type Paths struct {
	pathmap map[string]map[int]*Server
}

type WhiteList struct {
	ipmap map[string]map[int]*Server
}

func GeneratePaths() *Paths {
	var paths Paths
	paths.pathmap = make(map[string]map[int]*Server)
	return &paths
}

func (p *Paths) hasPath(path string) bool {
	_, err := p.pathmap[path]

	if len(p.pathmap) > 0{
		if err {
			return false
		}
	}
	return true
}

func(p *Paths) UpdatePath(path string, serverid int, server *Server) {

	// p.pathmap[path] = append(p.pathmap[path],server)
	fmt.Println("Inside update path")
	val, ok := p.pathmap[path]
	fmt.Println("Value:", val, "Err ", ok)

	if !ok{
		p.pathmap[path] = make(map[int]*Server)
	}

	p.pathmap[path][serverid] = server
}

func (w *WhiteList) isAllowedIP(ipaddress string) bool {

	_, err := w.ipmap[ipaddress]

	if len(w.ipmap) > 0{
		if err {
			return false
		}
	}
	return true
}






