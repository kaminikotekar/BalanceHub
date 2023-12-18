package main
import (
	"os"
	"io"
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

func main() {
	log.Println("Starting HTTP server...")
	lmap, error := Connection.LoadDB("RemoteServer.db")
	fmt.Println(lmap)
	if error {
		return
	}

	config, err := Config.GetConfiguration("config2.yaml")
	fmt.Println("config2.yaml ", config,  "err ", err)
	fmt.Println("LB server: ", config.GetLBServer())

	connections := Connection.InitConnection(lmap.GetServerIds())
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
		go handleConnection(conn, connections, lmap)
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

func readBytes(c chan []byte, reader *bufio.Reader){
	buffer := make([]byte,4)
	// Read 4 bytes of header 
	reader.Read(buffer[:4])
	remainingLength := 0

	// check if LB request
	if buffer[0] == 76 && buffer[1] == 66 {
		remainingLength = int(buffer[3])
		fmt.Println("remaining length ", remainingLength)
	}

	payloadLength := 1
	newPacket := false
	for {
		_byte, err := reader.ReadByte()
		fmt.Println("error reading byte ", err)
		fmt.Println("byte: ", _byte)
		if err != nil {
			c <- buffer
			break
		}
		buffer = append(buffer, _byte)
		if isEndOfHttpRequest(buffer){
			// Reached end of packet, allow receiving new packet
			newPacket = true
		}
		if remainingLength == payloadLength{
			// Reached end of packet, allow receiving new packet
			newPacket = true
		}
		if newPacket == true {
			// Push to the channel and continue
			c <- buffer
			buffer = buffer[:0]
			_, err = reader.Read(buffer[:4])
			if err != nil {
				break
			} else {
				newPacket = false
			}
		}
		payloadLength += 1
		
	}
}

func handleConnection(conn net.Conn, connectionLoad *Connection.Connections, lmap *RemoteServer.Map) {
	
	defer conn.Close()
	fmt.Println("Load : ")
	connectionLoad.PrintConnections()

	// Create an HTTP request reader
   	reader := bufio.NewReader(conn)

	fmt.Println("after creating reader ")

	// Loop throught the buffer to read all bytes

	c := make(chan []byte)
	go readBytes(c, reader)
	fmt.Println("after reading bytes")

	// fmt.Println("**************Reading from  connection" , buffer)
	buffer := <-c
	if buffer[0] == 76 && buffer[1] == 66{
		fmt.Println("Handle LB request")
		// conn.Write(CustomPacket.Test())
		// TODO: handle LB request
		res := decodeLBPacket(buffer)
		fmt.Println("sending res " ,res)
		conn.Write(res)
		return
	}

	fmt.Println("HTTP request : ", string(buffer))

	req, err := parseHTTPRequest(buffer)

	// request, err := http.ReadRequest(reader)

	fmt.Println("request: ", req)
	fmt.Printf("type: %T \n", req)
	fmt.Println("req body: ", req.Body)
	if err != nil {
		fmt.Println("Error reading HTTP request:", err)
		return
	}

	// Find Server with least active connection
	url := req.URL.Path

	fmt.Println("url: ", url)
	fmt.Println("RemoteAddr: ")
	fmt.Println("RemoteAddr: ", req.RemoteAddr)
	// TODO: check if the client is allowed to connect

	allowedServers := lmap.GetPossibleServers("127.0.0.1", url)

	fmt.Println("allowedServers: ", allowedServers)

	fmt.Println("printing optimal server: ......")
	remoteServerId := connectionLoad.GetOptimalServer(allowedServers)

	if remoteServerId == 0 {
		resBytes , err := getHttpRequestInBytes("401 Unauthorized", 
												http.StatusUnauthorized,
												"Unauthorized access",
												map[string]string{
													"Content-Type": "text/plain",
												})
		if err != nil {	
			conn.Write([]byte("Something went wrong"))
			return
		}
		conn.Write(resBytes)
		return
	}
	fmt.Printf("server returned: %v\n", remoteServerId)
	remoteServer := lmap.GetServerFromId(remoteServerId)
	fmt.Println("Optimal Server: ", remoteServer)

	connectionLoad.AddConnection(remoteServerId)
	defer connectionLoad.RemoveConnection(remoteServerId)

	dataToBereturned, err := performReverseProxy(req, remoteServer.Ipaddress + ":" + remoteServer.Port)
	if err != nil {
		conn.Write([]byte("Server Error: " + err.Error()))
	}
	conn.Write(dataToBereturned)
}

func decodeLBPacket(buffer []byte) []byte{
	
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



