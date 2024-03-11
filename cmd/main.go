package main

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/kaminikotekar/BalanceHub/pkg/Config"
	"github.com/kaminikotekar/BalanceHub/pkg/Connection"
	"github.com/kaminikotekar/BalanceHub/pkg/LBProtocol"
	"github.com/kaminikotekar/BalanceHub/pkg/Models/RemoteServer"
	"github.com/kaminikotekar/BalanceHub/pkg/Redis"
	"github.com/kaminikotekar/BalanceHub/pkg/Redis/Cache"
	"github.com/kaminikotekar/BalanceHub/pkg/Redis/LBLog"
	"github.com/gofor-little/env"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strings"
	"strconv"
	"time"
)

var DEFAULT_WORKERS = "2"

type Packet struct {
	ClientHost string
	ClientPort string
	PacType    string
	Data       []byte
}

func configTLS(config Config.LoadBalancer) *tls.Config {

	if config.Protocol == "HTTPS" {
		cert, err := tls.LoadX509KeyPair(config.SSLCert, config.SSLKey)
		if err != nil {
			LBLog.Log(LBLog.WARNING, fmt.Sprintf("Error loading TLS certificate and key: %s", err))
			return nil
		}
		tlsConfig := &tls.Config{
			Certificates:       []tls.Certificate{cert},
			InsecureSkipVerify: true,
		}
		return tlsConfig
	}
	return nil
}

func reWriteConfig(){
	if err := env.Write("DB_PASSWORD", "***", ".env", false); err != nil {
		panic(err)
	}
	if err := env.Write("REDIS_PASSWORD", "***", ".env", false); err != nil {
		panic(err)
	}
}

func init_server() bool {

	if err := env.Load(".env"); err != nil {
		panic(err)
	}
	workers, err := strconv.Atoi(env.Get("numCPU", DEFAULT_WORKERS))
	if err == nil {
		runtime.GOMAXPROCS(workers)
	}
	err = Config.LoadConfiguration()
	if err != nil {
		log.Fatal("Error Loading config ", err)
		return false
	}
	Redis.InitServer()
	LBLog.InitLogger()
	go LBLog.ProcessLogs()
	if Connection.LoadDB() {
		log.Fatal("Error Loading DB ")
		return false
	}
	Connection.InitConnection(RemoteServer.RemoteServerMap.GetServerIds())
	reWriteConfig()
	return true
}

func getReverProxyServer(config Config.LoadBalancer) (bool, net.Listener) {

	tlsConfig := configTLS(config)

	// Initialize remote server active connection pool
	Connection.InitConnection(RemoteServer.RemoteServerMap.GetServerIds())

	// Initialize server
	reverseProxy, err := net.Listen("tcp", Config.Configuration.GetLBServer())
	if err != nil {
		LBLog.Log(LBLog.WARNING, fmt.Sprintf("Error listening: %s", err.Error()))
		return false, nil
	}
	LBLog.Log(LBLog.INFO, fmt.Sprintf("Listening TCP requests on %s:%s ", Config.Configuration.GetLBIP(), Config.Configuration.GetLBPort()))
	if tlsConfig != nil {
		tlsListener := tls.NewListener(reverseProxy, tlsConfig)
		return true, tlsListener
	}
	return true, reverseProxy

}

func getTCPServer(config Config.LoadBalancer) (bool, net.Listener) {

	tcpListener, err := net.Listen("tcp", "localhost:"+config.TcpPort)
	if err != nil {
		LBLog.Log(LBLog.WARNING, fmt.Sprintf("Error listening: %s", err.Error()))
		return false, nil
	}
	LBLog.Log(LBLog.INFO, fmt.Sprintf("Listening TCP requests on %s:%s ", Config.Configuration.GetLBIP(), Config.Configuration.GetTcpPort()))
	return true, tcpListener
}

func main() {

	flag := init_server()
	if !flag {
		os.Exit(1)
	}
	
	workers, err := strconv.Atoi(env.Get("numCPU", DEFAULT_WORKERS))
	if err != nil {
		workers = 2
	}
	flag, rProxy := getReverProxyServer(Config.Configuration.LoadBalancer)
	flag, tcpServer := getTCPServer(Config.Configuration.LoadBalancer)
	if !flag {
		os.Exit(1)
	}

	defer rProxy.Close()
	defer tcpServer.Close()
	// Handle HTTP connections
	httpsConnBuffer := make(chan net.Conn)
	go func() {
		for {
			conn, err := rProxy.Accept()
			httpsConnBuffer <- conn
			if err != nil {
				LBLog.Log(LBLog.WARNING, fmt.Sprintf("Error accepting: %s \n", err.Error()))
				continue
			}

			LBLog.Log(LBLog.INFO, fmt.Sprintf("Received connection from %s", conn))

		}
	}()

	for i := 1; i <= workers; i++{
		go handleHttpConnection(httpsConnBuffer)
	}
	
	// Handle TCP Connections
	go func() {
		for {
			conn, err := tcpServer.Accept()
			if err != nil {
				LBLog.Log(LBLog.WARNING, fmt.Sprintf("Error accepting: %s \n", err.Error()))
				continue
			}

			LBLog.Log(LBLog.INFO, fmt.Sprintf("Received connection from %s", conn))
			go handleConnection(conn)
		}
	}()
	select {}

}

func updateRequestParms(req *http.Request, client string, uri string) error {
	newURL, err := url.Parse("http://" + uri)
	if err != nil {
		return err
	}
	req.Host = newURL.Host
	req.URL.Host = newURL.Host
	req.URL.Scheme = newURL.Scheme
	req.RequestURI = ""
	req.Header.Set("X-Forwarded-For", client)
	return nil
}

func writeRequestToBytes(req *http.Request) ([]byte, error) {
	var buf bytes.Buffer
	bufPointer := &buf
	err := req.Write(bufPointer)
	if err != nil {
		return nil, err
	}
	bytes := bufPointer.Bytes()
	return bytes, nil
}

func writeResponseToBytes(res *http.Response) ([]byte, error) {
	var buf bytes.Buffer
	bufPointer := &buf
	err := res.Write(bufPointer)
	if err != nil {
		return nil, err
	}
	bytes := bufPointer.Bytes()
	return bytes, nil
}

func performReverseProxy(client string, req *http.Request, uri string) ([]byte, error) {

	remoteServerConnect, err := net.Dial("tcp", uri)
	if err != nil {
		LBLog.Log(LBLog.WARNING, fmt.Sprintf("Error connecting to original server %s", uri))

		return nil, err
	}

	updateRequestParms(req, client, uri)
	bytesToSend, err := writeRequestToBytes(req)
	if err != nil {
		return nil, err
	}
	_, err = remoteServerConnect.Write(bytesToSend)

	// Reader 2
	reader := bufio.NewReader(remoteServerConnect)
	response, err := http.ReadResponse(reader, nil)
	defer response.Body.Close()
	if err != nil {
		// fmt.Println("Error reading response:", err2)
		return nil, err
	}

	bytesToSend, err = writeResponseToBytes(response)
	remoteServerConnect.Close()

	// Cache response if GET request
	if req.Method == "GET" {
		Cache.CacheResponse(req, bytesToSend)
	}
	return bytesToSend, nil
}

func parseHTTPRequest(req []byte) (*http.Request, error) {

	reqString := string(req)
	bufioReader := bufio.NewReader(strings.NewReader(reqString))
	request, err := http.ReadRequest(bufioReader)
	return request, err

}

func isEndOfHttpRequest(buffer []byte) bool {
	buffLength := len(buffer)
	if buffLength >= 4 {
		return string(buffer[buffLength-4:]) == "\r\n\r\n"
	}
	return false
}

func getHttpRequestInBytes(status string, statusCode int, body string, headers map[string]string) ([]byte, error) {
	response := http.Response{
		Status:     status,
		StatusCode: statusCode,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Body:       io.NopCloser(bytes.NewBufferString(body)),
		Header:     make(http.Header, 0),
	}
	for key, value := range headers {
		response.Header.Set(key, value)
	}

	buf := bytes.NewBuffer(nil)
	err := response.Write(buf)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func readHeader(reader *bufio.Reader) (*Packet, int, error) {
	p := Packet{
		Data: make([]byte, 4),
	}
	_, err := reader.Read(p.Data[:4])
	if err != nil {
		return nil, 0, err
	}
	remainingLength := 0

	// check if LB request
	if p.Data[0] == 76 && p.Data[1] == 66 {
		remainingLength = int(p.Data[3])
		p.PacType = "LB"
		LBLog.Log(LBLog.INFO, fmt.Sprintf("remaining length %d", remainingLength))
	}
	return &p, remainingLength, nil
}

func readBytes(c chan *Packet, reader *bufio.Reader) {

	packet, remainingLength, err := readHeader(reader)
	if err != nil {
		close(c)
		return
	}

	payloadLength, newPacket := 1, false
	for {
		_byte, err := reader.ReadByte()
		if err != nil {
			c <- packet
			break
		}
		packet.Data = append(packet.Data, _byte)
		if remainingLength == payloadLength {
			// Reached end of packet, allow receiving new packet
			newPacket = true
		}
		if newPacket == true {
			// Push to the channel and continue
			c <- packet
			payloadLength = 0
			packet, remainingLength, err = readHeader(reader)
			if err != nil {
				break
			}
			newPacket = false
		}
		payloadLength += 1
	}

	for {
		if len(c) == 0 {
			close(c)
			break
		}
	}
}

func isAllowedRemote(remoteHost string) bool {
	allowedRemotes := Config.Configuration.OrigServer.AllowSubnet
	for _, subnet := range allowedRemotes {
		if RemoteServer.IpInSubnet(remoteHost, subnet) {
			return true
		}
	}
	return false
}

func (packet *Packet) handlePacket() ([]byte, error) {

	if packet.PacType == "LB" {
		if !isAllowedRemote(packet.ClientHost) {
			LBLog.Log(LBLog.WARNING, "LB packet requested unfulfilled as not allowed")
			return nil, errors.New("Not Allowed")
		}
		res := packet.decodeLBPacket()
		return res, nil
	}
	return nil, errors.New("Invalid Packet")
}

func tryRemoteConnection(clientHost string, clientUrl string, req *http.Request) ([]byte, error) {

	remoteServerMap := RemoteServer.RemoteServerMap
	connectionMap := Connection.ConnectionMap
	totalTries := 3
	for totalTries > 0 {
		allowedServers := remoteServerMap.GetPossibleServers(clientHost, clientUrl)
		if len(allowedServers) == 0 {
			break
		}
		remoteServerId := connectionMap.GetOptimalServer(allowedServers)
		connectionMap.AddConnection(remoteServerId)
		defer connectionMap.RemoveConnection(remoteServerId)
		remoteServer := remoteServerMap.GetServerFromId(remoteServerId)
		dataToBereturned, err := performReverseProxy(clientHost, req, remoteServer.Ipaddress+":"+remoteServer.Port)
		if err != nil {
			remoteServerMap.RemoveServer(remoteServerId)
			totalTries -= 1
			continue
		}
		// f, err := os.OpenFile("load.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)

		// if err != nil {
		// 	log.Fatal("failed to open file", err)
		// }
		// f.WriteString(fmt.Sprintf("remoteServer : %s\n", remoteServer.Ipaddress+":"+remoteServer.Port))
		// f.WriteString(connectionMap.ActiveConnections())
		// f.Close()
		return dataToBereturned, nil
	}

	return nil, errors.New("Could not fulfill request")
}

func wrapTLS(conn net.Conn) net.Conn {

	// Wrap the connection
	tlsConn, ok := conn.(*tls.Conn)
	if !ok {
		LBLog.Log(LBLog.WARNING, "Connection is not a TLS connection")
		return nil
	}

	// Perform the TLS handshake
	err := tlsConn.Handshake()
	if err != nil {
		LBLog.Log(LBLog.WARNING, fmt.Sprintf("TLS handshake error: %s", err.Error()))
		log.Printf("TLS handshake error: %s", err.Error())
	}
	return tlsConn
}

func handleHttpConnection(c chan net.Conn) {

	for {
		conn , ok := <- c
		if !ok {
			break
		}
		// defer conn.Close()
		tlsConn := wrapTLS(conn)
		if tlsConn == nil {
			tlsConn = conn
		}

		req, err := http.ReadRequest(bufio.NewReader(conn))
		client, _, _ := net.SplitHostPort(conn.RemoteAddr().String())
		if err != nil {
			LBLog.Log(LBLog.INFO, fmt.Sprintf("Error reading HTTP request from %s", client))
			conn.Close()
			continue
		}
		url := req.URL.Path
		LBLog.Log(LBLog.INFO, fmt.Sprintf("Request Method: %s, url: %s, client %s", req.Method, url, client))

		allowedServers := RemoteServer.RemoteServerMap.GetPossibleServers(client, url)
		LBLog.Log(LBLog.INFO, fmt.Sprintf("Allowed servers: %v", allowedServers))

		var resBytes []byte
		// if no allowed server is found, then no access is allowed
		if len(allowedServers) == 0 {

			resBytes, err = getHttpRequestInBytes("401 Unauthorized",
				http.StatusUnauthorized,
				"Unauthorized access",
				map[string]string{
					"Content-Type": "text/plain",
				})

		} else if resBytes, err = Cache.GetFromCache(req); len(resBytes) != 0 {
			LBLog.Log(LBLog.INFO, "Using Cache")
		} else {
			resBytes, err = tryRemoteConnection(client, url, req)
			if err != nil {
				resBytes, err = getHttpRequestInBytes("503 Unauthorized",
					http.StatusServiceUnavailable,
					"Service currently unavailable, Please try after sometime",
					map[string]string{
						"Content-Type": "text/plain",
					})
			}
		}

		if err != nil {
			LBLog.Log(LBLog.WARNING, fmt.Sprintf("Error getting response: %s", err.Error()))
			conn.Close()
			continue
		}
		conn.Write(resBytes)
		conn.Close()
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	// LBLog.Log(LBLog.INFO, fmt.Sprintf("Connections : ", Connection.ConnectionMap.ActiveConnections()))
	conn.SetReadDeadline(time.Now().Add(5 * time.Millisecond))

	// Create an request reader
	reader := bufio.NewReader(conn)

	c := make(chan *Packet)
	clientHost, clientPort, _ := net.SplitHostPort(conn.RemoteAddr().String())
	LBLog.Log(LBLog.INFO, fmt.Sprintf("client addr  %s:%s", clientHost, clientPort))
	go readBytes(c, reader)

	for {
		packet, ok := <-c
		if !ok {
			break
		}
		packet.ClientHost = clientHost
		packet.ClientPort = clientPort
		rdata, err := packet.handlePacket()
		if err != nil {
			break
		}
		conn.Write(rdata)
	}
}

func (packet *Packet) decodeLBPacket() []byte {

	LBLog.Log(LBLog.INFO, fmt.Sprintf("REVEIVED LB Packet from %s:%s", packet.ClientHost, packet.ClientPort))
	decodedPacket, err := LBProtocol.DecodeToPacket(packet.Data)
	if err != nil {
		return LBProtocol.GenerateResponse(true, "Packet error: Unable to decode packet")

	}
	err = decodedPacket.HandleRemoteRequest(packet.ClientHost, packet.ClientPort)
	if err != nil {
		return LBProtocol.GenerateResponse(true, err.Error())
	} else {
		return LBProtocol.GenerateResponse(true, "Successful")
	}
}
