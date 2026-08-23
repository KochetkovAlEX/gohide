package vpn

import (
	"encoding/json"
	"os"
	"strconv"
)

func buildProxySettings(configName string, configMap map[string]RawConfig) Outbound {
	var chosenConfig RawConfig = configMap[configName]
	portInt, _ := strconv.Atoi(chosenConfig.Port)

	myProxy := Outbound{
		Type:           chosenConfig.Protocol,
		Tag:            "proxy",
		Server:         chosenConfig.Address,
		ServerPort:     portInt,
		UUID:           chosenConfig.UUID,
		Flow:           chosenConfig.Flow,
		Network:        "tcp",
		PacketEncoding: "xudp",
		TLS: &TLSConfig{
			Enabled:    true,
			ServerName: chosenConfig.SNI,
			UTLS: &UTLSConfig{
				Enabled:     true,
				Fingerprint: "chrome",
			},
			Reality: &RealityConfig{
				Enabled:   true,
				PublicKey: chosenConfig.PBK,
				ShortID:   chosenConfig.SID,
			},
		},
	}
	return myProxy
}

func BuildConfig(configName string, configMap map[string]RawConfig) {
	myProxy := buildProxySettings(configName, configMap)

	fullConfig := Config{
		Log: &LogConfig{Level: "info", Timestamp: true},
		DNS: &DNSConfig{
			Servers: []DNSServer{
				{Type: "https", Tag: "dns_proxy", Server: "1.1.1.1"},
				{Type: "udp", Tag: "dns_direct", Server: "223.5.5.5"},
			},
			Rules: []DNSRule{
				{Protocol: "dns", Action: "route", Server: "dns_direct"},
			},
		},
		Inbounds: []Inbound{
			{Type: "socks", Tag: "socks-in", Listen: "127.0.0.1", ListenPort: 10808},
			{Type: "http", Tag: "http-in", Listen: "127.0.0.1", ListenPort: 10809},
		},

		Outbounds: []Outbound{
			myProxy,
			{Type: "direct", Tag: "direct"},
			{Type: "block", Tag: "block"},
		},

		Route: &RouteConfig{
			Rules: []RouteRule{
				{Action: "sniff"},
				{Protocol: "dns", Action: "hijack-dns"},
			},
			Final:                 "proxy", // Указывает на tag "proxy" нашего сервера
			DefaultDomainResolver: "dns_proxy",
			AutoDetectInterface:   true,
		},
	}
	jsonBytes, _ := json.MarshalIndent(fullConfig, "", "  ")
	os.WriteFile(configName+"_config.json", jsonBytes, 0644)
}
