package Server
import (
	"fmt"
	"io/ioutil"
	"gopkg.in/yaml.v3"
)

type Server struct{
	Ipaddress string `yaml:"address"`
	Port int `yaml:"port"`
}

type ServerList struct {
	OriginalServers []Server `yaml:"servers"` 
}

func GetServerList(filename string) (ServerList, error) {
	var serverList ServerList
	yamlFile, err := ioutil.ReadFile(filename)
	if err != nil {
		fmt.Printf("Error reading YAML file: %v", err)
		return serverList, err
	}

	err = yaml.Unmarshal(yamlFile, &serverList)
	if err != nil {
		fmt.Println("Error unmarsh ", err)
		return serverList, err
	}
	
	return serverList, nil
}




// func main() {
// 	yamlFile, err := ioutil.ReadFile("config.yaml")
// 	if err != nil {
// 		fmt.Printf("Error reading YAML file: %v", err)
// 		}

// 	var serverList ServerList
// 	// var m map[string]string
// 	err = yaml.Unmarshal(yamlFile, &serverList)

// 	if err != nil {
// 		fmt.Println("Error unmarsh ", err)
// 	}

// 	fmt.Println("Server List: ", serverList)

// 	for _,server := range(serverList.OriginalServers){
// 		fmt.Println("server: ", server)
// 		fmt.Println("IP address: ", server.Ipaddress)
// 		fmt.Println("Port: ", server.Port)
// 		fmt.Printf("Type : %T", server)
// 	}

// }


