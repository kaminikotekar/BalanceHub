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
		"github.com/google/gopacket"
		"github.com/kaminikotekar/BalanceHub/pkg/Config"
		"github.com/kaminikotekar/BalanceHub/pkg/Connection"
		"github.com/kaminikotekar/BalanceHub/pkg/Models/RemoteServer"
		"github.com/kaminikotekar/BalanceHub/pkg/LBProtocol"
)

type Packet struct {
	Remote net.Addr
	PacType string
	Data []byte
}

func main() {
	log.Println("Starting HTTP server...")
	error := Connection.LoadDB("RemoteServer.db")
	// fmt.Println(lmap)
	if error {
		return
	}

	config, err := Config.GetConfiguration("config2.yaml")
	fmt.Println("config2.yaml ", config,  "err ", err)
	fmt.Println("LB server: ", config.GetLBServer())

	Connection.InitConnection(RemoteServer.RemoteServerMap.GetServerIds())
	// loadBalancer := config.LoadBalancer
	// remoteServers := config.OriginalServers
	// connectionLoad := &Connection.Connections{
	// 	ActiveConnections: make(map[Config.Server]int),
	// }
	// connectionLoad.InitializeLoadServers(remoteServers)
	// LBRequestProtocol.Test()

	// fmt.Println("loaad: ", connectionLoad)
	// fmt.Println("Server List ", remoteServers)

	reverseProxy, err := net.Listen("tcp", config.GetLBServer())
	if err != nil {
		log.Printf("Error listening: %s", err.Error())
		os.Exit(1)
	}
	
	defer reverseProxy.Close()
	log.Printf("Listening on %s:%s \n",config.GetLBIP(), config.GetLBPort())
	for {
		conn, err := reverseProxy.Accept()
		if err != nil {
			log.Printf("Error accepting: %s \n", err.Error())
			continue
		}
		fmt.Println("Received connection from ", conn)
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
		fmt.Println("remaining length ", remainingLength)
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

func (packet *Packet) handlePacket() ([]byte, error) {

	fmt.Println("buffer received : ", packet.Data)
	if packet.PacType == "LB" {
		fmt.Println("Handle LB request")
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
	fmt.Println("RemoteAddr: ", req.RemoteAddr)

	allowedServers := RemoteServer.RemoteServerMap.GetPossibleServers("127.0.0.1", url)

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

func handleConnection(conn net.Conn) {
	
	defer conn.Close()
	Connection.ConnectionMap.PrintConnections()
	conn.SetReadDeadline(time.Now().Add(10 * time.Millisecond))

	// Create an request reader
   	reader := bufio.NewReader(conn)

	c := make(chan *Packet)
	client := conn.RemoteAddr()
	fmt.Println("client addr " ,client)
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
		conn.Write(rdata)
	}
}

func decodeLBPacket(remote net.Addr, buffer []byte) []byte{
	
	// action := buffer[2]
	remainingLength := buffer[3]
	// payload := buffer[4: remainingLength]

	packet := gopacket.NewPacket(buffer[:4+remainingLength], 
		LBProtocol.LBLayerType,
		gopacket.Default)

	fmt.Println("packet: ")

	customLayer := packet.Layer(LBProtocol.LBLayerType)
	customLayerContent, _ := customLayer.(*LBProtocol.LBRequestLayer)
	decodedPacket, err := customLayerContent.Deserialize()

	fmt.Println("decoded packet: ", decodedPacket)
	fmt.Println("error : ", err)
	
	remoteIP, remotePort, err := net.SplitHostPort(remote.String())
	decodedPacket.HandleRemoteRequest(remoteIP, remotePort)

	// Create res packet
	rawBytes := []byte("RE")
	var message []byte
	if err != nil {

		rawBytes = append(rawBytes, 0xFF)
		message = []byte("Could not decode packet")
		// fmt.Println("remaining length byte: ", byte(len(pydata)) )

	} else{
		rawBytes = append(rawBytes, 0x00)
		message = []byte("Success")
	}

	rawBytes = append(rawBytes, byte(len(message)))
	pData := append(rawBytes, message...)
	res := gopacket.NewPacket(
		pData,
		LBProtocol.ErrLayerType,
		gopacket.Default,
	)
	
	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{}

	err = gopacket.SerializePacket(buf, opts, res)

	return buf.Bytes()
}



