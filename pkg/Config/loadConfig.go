package Config
import (
	"fmt"
	"io/ioutil"
	"gopkg.in/yaml.v3"
)


type ServerRestrictions struct {
	AllowSubnet []string `yaml:"allow"`
	DenySubnet []string `yaml:"deny"`
}

type LoadBalancer struct {
	// Ipaddress string `yaml:"address"`
	Port string `yaml:"listen"`
	Algorithm string `yaml:"algorithm"`
	AccessLogs string `yaml:"access-logs"`
	AccessLogsPath string `yaml:"access-logs-path"`
	Caching bool `yaml:"caching"`
	CacheDuration int `yaml:"cache-duration"`
}

type Config struct {
	OrigServer ServerRestrictions `yaml:"Original-Server"`
	LoadBalancer LoadBalancer `yaml:"Server"`

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
	fmt.Println("Configuration ", config)
	return config, nil
}

func (config Config) GetLBServer() string {
	return "localhost:" + config.LoadBalancer.Port
}

func (config Config) GetLBIP() string {
	return "localhost:"
}

func (config Config) GetLBPort() string {
	return config.LoadBalancer.Port
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


