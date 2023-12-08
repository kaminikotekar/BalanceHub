package main
import (
	"os"
	// "io"
		"log"
		"bufio"
        "fmt"
        "net"
		"net/http"
		"net/url"
		"bytes"
		"strings"
		"github.com/kaminikotekar/BalanceHub/pkg/Config"
		"github.com/kaminikotekar/BalanceHub/pkg/Connection"
		"github.com/kaminikotekar/BalanceHub/pkg/Models/RemoteServer"
		// "github.com/kaminikotekar/BalanceHub/pkg/LBRequestProtocol"
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

func handleConnection(conn net.Conn, connectionLoad *Connection.Connections, lmap *RemoteServer.Map) {
	
	defer conn.Close()
	fmt.Println("Load : ")
	connectionLoad.PrintConnections()

	// Create an HTTP request reader
   	reader := bufio.NewReader(conn)


	// Loop throught the buffer to read all bytes
	buffer := make([]byte,0)
	for {
		
		_byte, err := reader.ReadByte()
		if err!=nil {
			break
		}
		buffer = append(buffer, _byte)
	}

	fmt.Println("**************Reading from  connection" , buffer)

	if buffer[0] == 76 && buffer[1] == 66{
		fmt.Println("Handle LB request")
		// conn.Write(CustomPacket.Test())
		// TODO: handle LB request
		return
	}

	fmt.Println("HTTP request : ", string(buffer))

	req, err := parseHTTPRequest(buffer)

	// request, err := http.ReadRequest(reader)

	fmt.Println("request: ", req)
	fmt.Printf("type: %T \n", req)
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
	server := connectionLoad.GetOptimalServer(allowedServers)
	fmt.Printf("server returned: %v\n", server)
	fmt.Println("Optimal Server: ", lmap.GetServerFromId(server))

	connectionLoad.AddConnection(server)
	defer connectionLoad.RemoveConnection(server)



}

func decodeTCPPacket(reader *bufio.Reader) {
	action, _ := reader.ReadByte()
	if (action == 0XF0){
		fmt.Println("Request for Regiist")
	}

}



