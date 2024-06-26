package Config

import (
	"log"
	"gopkg.in/yaml.v3"
	"io/ioutil"
)

const configFile = "config.yaml"

var Configuration *Config

type ServerRestrictions struct {
	AllowSubnet []string `yaml:"allow"`
}

type RedisServer struct {
	Ip            string `yaml:"ip"`
	Port          string `yaml:"port"`
	Dbindex       int    `yaml:"db"`
	Caching       bool   `yaml:"caching"`
	CacheDuration int    `yaml:"cache-duration"`
}

type LoadBalancer struct {
	Port           string      `yaml:"listen"`
	TcpPort        string      `yaml:"tcpListener"`
	Protocol       string      `yaml:"protocol"`
	SSLCert        string      `yaml:"ssl_certificate"`
	SSLKey         string      `yaml:"ssl_certificate_key"`
	Algorithm      string      `yaml:"algorithm"`
	RedisWorker    RedisServer `yaml:"redis-server"`
	AccessLogsPath string      `yaml:"access-logs-path"`
	DBPath         string      `yaml:"db-path"`
}

type Config struct {
	OrigServer   ServerRestrictions `yaml:"Original-Server"`
	LoadBalancer LoadBalancer       `yaml:"BalanceHub"`
}

func LoadConfiguration() error {

	var c Config
	yamlFile, err := ioutil.ReadFile(configFile)
	if err != nil {
		log.Printf("Error reading YAML file: %v", err)
		return err
	}
	err = yaml.Unmarshal(yamlFile, &c)
	if err != nil {
		log.Println("Error unmarsh ", err)
		return err
	}
	Configuration = &c
	return nil
}

func (c *Config) GetLBServer() string {
	return "0.0.0.0:" + c.LoadBalancer.Port
}

func (c *Config) GetLBIP() string {
	return "0.0.0.0:"
}

func (c *Config) GetLBPort() string {
	return c.LoadBalancer.Port
}

func (c *Config) GetTcpPort() string {
	return c.LoadBalancer.TcpPort
}

func (c *Config) GetRedisConfig() RedisServer {
	return c.LoadBalancer.RedisWorker
}
