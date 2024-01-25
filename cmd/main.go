package main
import (
	"os"
	"io"
	"time"
	"errors"
	"log"
	"bufio"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"bytes"
	"strings"
	"runtime"
	"crypto/tls"
	"github.com/kaminikotekar/BalanceHub/pkg/Config"
	"github.com/kaminikotekar/BalanceHub/pkg/Connection"
	"github.com/kaminikotekar/BalanceHub/pkg/Models/RemoteServer"
	"github.com/kaminikotekar/BalanceHub/pkg/LBProtocol"
	"github.com/kaminikotekar/BalanceHub/pkg/Redis"
	"github.com/kaminikotekar/BalanceHub/pkg/Redis/LBLog"
	"github.com/kaminikotekar/BalanceHub/pkg/Redis/Cache"
)

type Packet struct {
	Remote net.Addr
	PacType string
	Data []byte
}


func configTLS(config Config.LoadBalancer) *tls.Config {
	
	if config.Protocol == "HTTPS" {
		cert, err := tls.LoadX509KeyPair(config.SSLCert, config.SSLKey)
		if err != nil {
			LBLog.Log(LBLog.WARNING, fmt.Sprintf("Error loading TLS certificate and key: %s", err))
			return nil
		}
		tlsConfig := &tls.Config{
			Certificates: []tls.Certificate{cert},
			InsecureSkipVerify: true,
		}
		return tlsConfig
	}
	return nil
}

func init_server() (bool, net.Listener){

	numCPU := runtime.NumCPU()
	fmt.Println("Number of cores : ", numCPU)
	runtime.GOMAXPROCS(2)
	err := Config.LoadConfiguration()
	LBConfig := Config.Configuration.LoadBalancer
	if err != nil {
		log.Fatal("Error Loading config ", err)
		fmt.Println("Error Loading config ", err)
		return false, nil
	}
	Redis.InitServer()
	LBLog.InitLogger()
	go LBLog.ProcessLogs()
	if Connection.LoadDB() {
		fmt.Println(" Error Loading DB ")
		return false, nil
	}
	// Load cert
	tlsConfig := configTLS(LBConfig)

	// Initialize remote server active connection pool
	Connection.InitConnection(RemoteServer.RemoteServerMap.GetServerIds())

	// Initialize server
	reverseProxy, err := net.Listen("tcp", Config.Configuration.GetLBServer())
	if err != nil {
		LBLog.Log(LBLog.WARNING, fmt.Sprintf("Error listening: %s", err.Error()))
		return false, nil
	}
	LBLog.Log(LBLog.INFO, fmt.Sprintf("Listening on %s:%s ",Config.Configuration.GetLBIP(), Config.Configuration.GetLBPort()))
	if tlsConfig != nil {
		tlsListener := tls.NewListener(reverseProxy, tlsConfig)
		return true, tlsListener
	}
	return true, reverseProxy
}

func main() {

	flag, rProxy := init_server()
	if !flag {
		os.Exit(1)
	}
	defer rProxy.Close()
	for {
		conn, err := rProxy.Accept()
		if err != nil {
			LBLog.Log(LBLog.WARNING, fmt.Sprintf("Error accepting: %s \n", err.Error()))
			continue
		}

		LBLog.Log(LBLog.INFO, fmt.Sprintf("Received connection from %s", conn))
		// read data from connection
		go handleConnection(conn)
	}
}

func updateRequestParms(req *http.Request, uri string) error{
	newURL, err := url.Parse("http://"+uri)
	if err != nil {
		fmt.Println("Error parsing new URL:", err)
		return  err
	}
	req.Host = newURL.Host
	req.URL.Host = newURL.Host
	req.URL.Scheme = newURL.Scheme
	req.RequestURI = ""

	return nil
}

func writeRequestToBytes(req *http.Request) ([]byte, error){
	var buf bytes.Buffer
	bufPointer := &buf
	err := req.Write(bufPointer)
	if err!=nil {
		return nil, err
	}
	bytes := bufPointer.Bytes()
	return bytes, nil
}

func writeResponseToBytes(res *http.Response) ([]byte, error){
	var buf bytes.Buffer
	bufPointer := &buf
	err := res.Write(bufPointer)
	if err!=nil {
		return nil, err
	}
	bytes := bufPointer.Bytes()
	return bytes, nil
}

func performReverseProxy(req *http.Request, uri string) ([]byte, error){

	remoteServerConnect, err := net.Dial("tcp", uri)
	fmt.Printf("connect type : %T \n", remoteServerConnect)
	if err != nil {
		fmt.Println("Error connecting to original server: ", err)
		return nil, err
	}

	updateRequestParms(req, uri)
	bytesToSend , err := writeRequestToBytes(req) 
	if err != nil {
		return nil, err
	}
	_, err = remoteServerConnect.Write(bytesToSend)

	// Reader 2
	reader := bufio.NewReader(remoteServerConnect)
	response, err := http.ReadResponse(reader, nil)
	fmt.Printf("type: %T \n", response)
	fmt.Println("response: ", response.Body)
	defer response.Body.Close()
	if err != nil {
		// fmt.Println("Error reading response:", err2)
		return nil, err
	}
	
	fmt.Println("response to send back to client", response)

	bytesToSend, err = writeResponseToBytes(response)
	remoteServerConnect.Close()

	// Cache response if GET request
	if req.Method == "GET" {
		Cache.CacheResponse(req, bytesToSend)
	}
	return  bytesToSend, nil
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

func getHttpRequestInBytes (status string, statusCode int, body string, headers map[string]string) ([]byte, error) {
	response := http.Response{
		Status: status,
		StatusCode: statusCode,
		Proto: "HTTP/1.1",
		ProtoMajor:    1,
		  ProtoMinor:    1,
		Body: io.NopCloser(bytes.NewBufferString(body)),
		Header: make(http.Header, 0),
	}
	for key,value := range headers {
		response.Header.Set(key, value)
	}
	
	buf := bytes.NewBuffer(nil)
	err := response.Write(buf)
	if err != nil {	
		fmt.Println("err", err)
		return nil, err
	}
	return buf.Bytes(), nil
}

func readHeader(reader *bufio.Reader) (*Packet, int, error) {
	p := Packet {
		Data : make([]byte, 4),
	}
	_, err := reader.Read(p.Data[:4])
	if err != nil {
		return nil, 0 , err
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

func readBytes(c chan *Packet, reader *bufio.Reader){

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
		if isEndOfHttpRequest(packet.Data){
			// Reached end of packet, allow receiving new packet
			newPacket = true
		}
		if remainingLength == payloadLength{
			// Reached end of packet, allow receiving new packet
			newPacket = true
		}
		if newPacket == true {
			// Push to the channel and continue
			fmt.Println("reached packet end : ", packet.Data)
			c <- packet
			payloadLength = 0
			packet, remainingLength, err = readHeader(reader)
			if err != nil { break
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

func isAllowedRemote(remote net.Addr) bool {
	allowedRemotes := Config.Configuration.OrigServer.AllowSubnet
	remoteIP, _, _ := net.SplitHostPort(remote.String())
	for _, subnet := range allowedRemotes {
		if RemoteServer.IpInSubnet(remoteIP, subnet){
			return true
		}
	}
	return false
}

func (packet *Packet) handlePacket() ([]byte, error) {

	fmt.Println("buffer received : ", packet.Data)
	if packet.PacType == "LB" {
		fmt.Println("Handle LB request")
		if !isAllowedRemote(packet.Remote){
			LBLog.Log(LBLog.WARNING, "LB packet requested unfulfilled as not allowed")
			return nil, errors.New("Not Allowed")
		}
		res := decodeLBPacket(packet.Remote, packet.Data)
		fmt.Println("sending res " ,res)
		return res, nil
	}

	fmt.Println("HTTP request : ", string(packet.Data))

	req, err := parseHTTPRequest(packet.Data)

	if req == nil {
		return nil, errors.New("Invalid request")
	}
	fmt.Println("request: ", req)
	fmt.Printf("type: %T \n", req)
	fmt.Println("req body: ", req.Body)
	if err != nil {
		fmt.Println("Error reading HTTP request:", err)
		return nil, err
	}

	// Find Server with least active connection
	url := req.URL.Path

	fmt.Println("url: ", url)
	fmt.Println("RemoteAddr: ", packet.Remote)
	remoteIP, _, err := net.SplitHostPort(packet.Remote.String())
	allowedServers := RemoteServer.RemoteServerMap.GetPossibleServers(remoteIP, url)

	fmt.Println("allowedServers: ", allowedServers)

	fmt.Println("printing optimal server: ......")
	remoteServerId := Connection.ConnectionMap.GetOptimalServer(allowedServers)

	if remoteServerId == 0 {
		resBytes , err := getHttpRequestInBytes("401 Unauthorized", 
												http.StatusUnauthorized,
												"Unauthorized access",
												map[string]string{
													"Content-Type": "text/plain",
												})
		if err != nil {	
			return []byte("Something went wrong"),nil
		}
		return resBytes,nil
	}

	// Find if request is cached
	res, err := Cache.GetFromCache(req)
	if err == nil {
		fmt.Println("Got response from cahce : ", res)
		return res, nil
	}
	fmt.Println("Request does not contain in cache: ", err)

	fmt.Printf("server returned: %v\n", remoteServerId)
	remoteServer := RemoteServer.RemoteServerMap.GetServerFromId(remoteServerId)
	fmt.Println("Optimal Server: ", remoteServer)

	Connection.ConnectionMap.AddConnection(remoteServerId)
	defer Connection.ConnectionMap.RemoveConnection(remoteServerId)

	dataToBereturned, err := performReverseProxy(req, remoteServer.Ipaddress + ":" + remoteServer.Port)
	if err != nil {
		return []byte("Server Error: " + err.Error()), nil
	}
	fmt.Println("Data to Bereturned: ", dataToBereturned)
	return dataToBereturned, nil
}

func wrapTLS (conn net.Conn) net.Conn{

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

func handleConnection(conn net.Conn) {
	defer conn.Close()
	tlsConn := wrapTLS(conn)
	if tlsConn == nil {
		tlsConn = conn
	}

	LBLog.Log(LBLog.INFO, fmt.Sprintf("Connections : ", Connection.ConnectionMap.ActiveConnections()))
	// conn.SetReadDeadline(time.Now().Add(5 * time.Millisecond))
	tlsConn.SetReadDeadline(time.Now().Add(5 * time.Millisecond))

	// Create an request reader
   	reader := bufio.NewReader(conn)
	reader = bufio.NewReader(tlsConn)

	c := make(chan *Packet)
	// client := conn.RemoteAddr()
	client := tlsConn.RemoteAddr()
	LBLog.Log(LBLog.INFO, fmt.Sprintf("client addr  %s", client))
	go readBytes(c, reader)

	for {
		packet, ok := <-c
		if !ok {
			break
		}
		packet.Remote = client
		rdata, err := packet.handlePacket()
		if err != nil {
			break
		}
		// conn.Write(rdata)
		tlsConn.Write(rdata)
	}
}

func decodeLBPacket(remote net.Addr, buffer []byte) []byte{

	decodedPacket, err := LBProtocol.DecodeToPacket(buffer)
	fmt.Println("decoded packet: ", decodedPacket)	
	if err != nil {
		fmt.Println("error : ", err)
		return LBProtocol.GenerateResponse(true, "Packet error: Unable to decode packet")

	}

	remoteIP, remotePort, err := net.SplitHostPort(remote.String())
	err = decodedPacket.HandleRemoteRequest(remoteIP, remotePort)

	if err != nil {
		return LBProtocol.GenerateResponse(true, err.Error())
	} else {
		return LBProtocol.GenerateResponse(true, "Successful")
	}
}



