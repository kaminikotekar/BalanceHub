package Config
import (
	"fmt"
	"io/ioutil"
	"gopkg.in/yaml.v3"
)

type Server struct{
	Ipaddress string `yaml:"address"`
	Port string `yaml:"port"`
}

type LoadBalancer struct {
	Ipaddress string `yaml:"address"`
	Port string `yaml:"port"`
}

type Config struct {
	OriginalServers []Server `yaml:"servers"`
	LoadBalancer LoadBalancer `yaml:"loadBalancer"`
}

func GetConfiguration(filename string) (Config, error) {

	var config Config
	yamlFile, err := ioutil.ReadFile(filename)
	if err != nil {
		fmt.Printf("Error reading YAML file: %v", err)
		return config, err
	}
	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		fmt.Println("Error unmarsh ", err)
		return config, err
	}	
	return config, nil
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


