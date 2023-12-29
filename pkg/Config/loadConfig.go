package Config
import (
	"fmt"
	"io/ioutil"
	"gopkg.in/yaml.v3"
)

const configFile = "config2.yaml"
var Configuration *Config

type ServerRestrictions struct {
	AllowSubnet []string `yaml:"allow"`
}

type LoadBalancer struct {
	// Ipaddress string `yaml:"address"`
	Port string `yaml:"listen"`
	Algorithm string `yaml:"algorithm"`
	AccessLogs string `yaml:"access-logs"`
	AccessLogsPath string `yaml:"access-logs-path"`
	Caching bool `yaml:"caching"`
	CacheDuration int `yaml:"cache-duration"`
	DBPath string `yaml:"db-path"`
}

type Config struct {
	OrigServer ServerRestrictions `yaml:"Original-Server"`
	LoadBalancer LoadBalancer `yaml:"BalanceHub"`

}

func LoadConfiguration() (error) {

	var c Config
	yamlFile, err := ioutil.ReadFile(configFile)
	if err != nil {
		fmt.Printf("Error reading YAML file: %v", err)
		return err
	}
	err = yaml.Unmarshal(yamlFile, &c)
	if err != nil {
		fmt.Println("Error unmarsh ", err)
		return err
	}	
	fmt.Println("Configuration ", Configuration)
	Configuration = &c
	return nil
}

func (c *Config) GetLBServer() string {
	return "localhost:" + c.LoadBalancer.Port
}

func (c *Config) GetLBIP() string {
	return "localhost:"
}

func (c *Config) GetLBPort() string {
	return c.LoadBalancer.Port
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


