package config

import (
	"fmt"
	"strings"
)

// BootstrapNode represents a bootstrap node configuration
type BootstrapNode struct {
	NodeID      string
	IP          string
	StakingPort uint16
	HTTPPort    uint16
}

// MainnetBootstrapNodes are the official mainnet bootstrap nodes
var MainnetBootstrapNodes = []BootstrapNode{
	{
		NodeID:      "NodeID-Mp8JrhoLmrGznZoYsszM19W6dTdcR35NF",
		IP:          "52.53.185.222",
		StakingPort: 9651,
		HTTPPort:    9630,
	},
	{
		NodeID:      "NodeID-Nf5M5YoDN5CfR1wEmCPsf5zt2ojTZZj6j",
		IP:          "52.53.185.223",
		StakingPort: 9651,
		HTTPPort:    9630,
	},
	{
		NodeID:      "NodeID-JCBCEeyRZdeDxEhwoztS55fsWx9SwJDVL",
		IP:          "52.53.185.224",
		StakingPort: 9651,
		HTTPPort:    9630,
	},
	{
		NodeID:      "NodeID-JQvVo8DpzgyjhEDZKgqsFLVUPmN6JP3ig",
		IP:          "52.53.185.225",
		StakingPort: 9651,
		HTTPPort:    9630,
	},
	{
		NodeID:      "NodeID-PKTUGFE6jnQbnskSDM3zvmQjnHKV3fxy4",
		IP:          "52.53.185.226",
		StakingPort: 9651,
		HTTPPort:    9630,
	},
	{
		NodeID:      "NodeID-LtBrcgdgPW9Nj9JoU1AwGeCgi29R9JoQC",
		IP:          "52.53.185.227",
		StakingPort: 9651,
		HTTPPort:    9630,
	},
	{
		NodeID:      "NodeID-962omv3YgJsqbcPvVR4yDHU8RPtaKCLt",
		IP:          "52.53.185.228",
		StakingPort: 9651,
		HTTPPort:    9630,
	},
	{
		NodeID:      "NodeID-LPznW4BxjJaFYP5KEuJUenwVGTkH48XDe",
		IP:          "52.53.185.229",
		StakingPort: 9651,
		HTTPPort:    9630,
	},
	{
		NodeID:      "NodeID-4nDStCMacNr5aadavMZxAxk9m9bfFf69F",
		IP:          "52.53.185.230",
		StakingPort: 9651,
		HTTPPort:    9630,
	},
	{
		NodeID:      "NodeID-GGpbeWwfsZBaasex25ZPMkJFN713BXx7u",
		IP:          "52.53.185.231",
		StakingPort: 9651,
		HTTPPort:    9630,
	},
	{
		NodeID:      "NodeID-Fh7dFdzt1QYQDTKJfZTVBLMyPipP99AmH",
		IP:          "52.53.185.232",
		StakingPort: 9651,
		HTTPPort:    9630,
	},
}

// TestnetBootstrapNodes are the testnet bootstrap nodes
var TestnetBootstrapNodes = []BootstrapNode{
	{
		NodeID:      "NodeID-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
		IP:          "testnet1.lux.network",
		StakingPort: 9651,
		HTTPPort:    9630,
	},
	{
		NodeID:      "NodeID-yyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyy",
		IP:          "testnet2.lux.network",
		StakingPort: 9651,
		HTTPPort:    9630,
	},
}

// LocalBootstrapNodes for local development
var LocalBootstrapNodes = []BootstrapNode{
	// Empty for local - nodes discover each other
}

// GetBootstrapNodes returns bootstrap nodes for a network
func GetBootstrapNodes(networkName string) []BootstrapNode {
	switch networkName {
	case "mainnet":
		return MainnetBootstrapNodes
	case "testnet":
		return TestnetBootstrapNodes
	case "local":
		return LocalBootstrapNodes
	default:
		// L2 networks use parent bootstrap nodes
		if net, err := GetNetwork(networkName); err == nil && net.IsL2 {
			return GetBootstrapNodes(net.ParentNetwork)
		}
		return []BootstrapNode{}
	}
}

// FormatBootstrapIPs creates the bootstrap IPs string for luxd
func FormatBootstrapIPs(nodes []BootstrapNode) string {
	var ips []string
	for _, node := range nodes {
		ips = append(ips, fmt.Sprintf("%s:%d", node.IP, node.StakingPort))
	}
	return strings.Join(ips, ",")
}

// FormatBootstrapIDs creates the bootstrap IDs string for luxd
func FormatBootstrapIDs(nodes []BootstrapNode) string {
	var ids []string
	for _, node := range nodes {
		ids = append(ids, node.NodeID)
	}
	return strings.Join(ids, ",")
}