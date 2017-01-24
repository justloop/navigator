package discovery

import (
	"strconv"

	"net"

	"github.com/hashicorp/serf/cmd/serf/command/agent"
	"github.com/hashicorp/serf/serf"
	"github.com/justloop/navigator/utils"
)

// Config is the configuration related to discovery module
type Config struct {
	// Address is the address this node bind to
	Address string `json:"addr"`
	// Port is the discovery port of this node bind to
	Port int `json:"port"`
	// Seed is the seed nodes this discovery node can join to
	Seed []string `json:"seed"`

	// ClientAddr is the address to communicate with client
	ClientAddr string `json:"clientAddr"`
	// ClientPort is the port to communicate with the client
	ClientPort int `json:"clientPort"`
}

// GetClusterAddrStr will return the string representation of cluster address
func GetClusterAddrStr(config *Config) string {
	host := config.Address
	port := config.Port
	if len(host) == 0 {
		host = utils.GetHostname()

	}
	if port == 0 {
		port = 7946
	}
	return host + ":" + strconv.Itoa(port)
}

// GetClientAddrStr will return the string representation of client address
func GetClientAddrStr(config *Config) string {
	if len(config.ClientAddr) == 0 || config.ClientPort == 0 {
		return utils.GetHostname() + ":7373"
	}
	return config.ClientAddr + ":" + strconv.Itoa(config.ClientPort)
}

// DefaultConfig is getting the default configuration of Discovery
func DefaultConfig() *Config {
	aConfig := agent.DefaultConfig()
	sConfig := serf.DefaultConfig()
	clientAddr, clientPort, _ := net.SplitHostPort(aConfig.RPCAddr)
	portInt, _ := strconv.Atoi(clientPort)
	return &Config{
		Address:    sConfig.MemberlistConfig.BindAddr,
		Port:       sConfig.MemberlistConfig.BindPort,
		ClientAddr: clientAddr,
		ClientPort: portInt,
	}
}

var getAgentAndSerfConfig = func(config *Config) (*agent.Config, *serf.Config) {
	aConfig := agent.DefaultConfig()
	if len(config.ClientAddr) > 0 && config.ClientPort > 0 {
		aConfig.RPCAddr = GetClientAddrStr(config)
	}
	sConfig := serf.DefaultConfig()
	if len(config.Address) > 0 {
		sConfig.MemberlistConfig.BindAddr = config.Address
	}
	if config.Port > 0 {
		sConfig.MemberlistConfig.BindPort = config.Port
	}
	sConfig.NodeName = GetClusterAddrStr(config)
	sConfig.Init()

	return aConfig, sConfig
}
