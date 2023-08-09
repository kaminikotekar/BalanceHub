package main
import ("os"
		"log"
		"bufio"
        "fmt"
        "net"
		"net/http"
		"net/url"
		"bytes"
		"github.com/kaminikotekar/BalanceHub/pkg/Config"
		"github.com/kaminikotekar/BalanceHub/pkg/Connection"
)


func main() {
	log.Println("Starting HTTP server...")
	config, err := Config.GetConfiguration("config.yaml")
	
	loadBalancer := config.LoadBalancer
	remoteServers := config.OriginalServers
	connectionLoad := &Connection.Connections{
		ActiveConnections: make(map[Config.Server]int),
	}
	connectionLoad.InitializeLoadServers(remoteServers)

	fmt.Println("loaad: ", connectionLoad)
	fmt.Println("Server List ", remoteServers)

	reverseProxy, err := net.Listen("tcp", loadBalancer.Ipaddress+":"+loadBalancer.Port)
	if err != nil {
		log.Printf("Error listening: %s", err.Error())
		os.Exit(1)
	}
	
	defer reverseProxy.Close()
	log.Printf("Listening on %s:%s \n",loadBalancer.Ipaddress, loadBalancer.Port)
	for {
		conn, err := reverseProxy.Accept()
		if err != nil {
			log.Printf("Error accepting: %s \n", err.Error())
			continue
		}
		fmt.Println("Received connection from ", conn)
		// read data from connection
		go handleConnection(conn, connectionLoad)
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

func handleConnection(conn net.Conn, connectionLoad *Connection.Connections) {
	
	defer conn.Close()
	fmt.Println("Load : ", connectionLoad.ActiveConnections)

	// Create an HTTP request reader
	reader := bufio.NewReader(conn)

	// Read the HTTP request
	request, err := http.ReadRequest(reader)

	fmt.Println("request: ", request)
	fmt.Printf("type: %T \n", request)
	if err != nil {
		fmt.Println("Error reading HTTP request:", err)
		return
	}

	// Find Server with least active connection
	fmt.Println("printing optimal server: ......")
	server := connectionLoad.GetOptimalServer()
	fmt.Println("Optimal Server: ", server)

	connectionLoad.AddConnection(server)
	defer connectionLoad.RemoveConnection(server)

	fmt.Println("Load : ", connectionLoad.ActiveConnections)

	response, err := performReverseProxy(request, server.Ipaddress + ":" + server.Port)

	if err != nil {
		fmt.Println(err)
		return
	}
	conn.Write(response)

}



