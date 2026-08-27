package vpn

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func buildProxySettings(chosenConfig RawConfig) Outbound {
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
			ServerName: "rutube.ru",
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

func BuildConfig(chosenConfig RawConfig) (string, error) {
	myProxy := buildProxySettings(chosenConfig)

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
				{Action: "hijack-dns", Protocol: "dns"},
			},
			Final:                 "proxy", // Указывает на tag "proxy" нашего сервера
			DefaultDomainResolver: "dns_proxy",
			AutoDetectInterface:   true,
		},
	}
	jsonBytes, err := json.MarshalIndent(fullConfig, "", "  ")
	if err != nil {
		return "", fmt.Errorf("[ERROR] Json Error: %v", err)
	}

	var baseDir string
	exePath, err := os.Executable()

	if err == nil && !strings.Contains(exePath, "go-build") {
		baseDir = filepath.Dir(exePath)
	} else {
		baseDir, err = os.Getwd()
		if err != nil {
			baseDir = "."
		}
	}

	configDir := filepath.Join(baseDir, "cfg")

	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", fmt.Errorf("[ERROR] Can`t create folder: %v", err)
	}

	finalFilePath := filepath.Join(configDir, chosenConfig.Name+"_config.json")
	if err := os.WriteFile(finalFilePath, jsonBytes, 0644); err != nil {
		return "", fmt.Errorf("[ERROR] Can`t write json file: %v", err)
	}

	return finalFilePath, nil
}

// Сборщик конфига для TUN-подключения
func BuildTunConfig(chosenConfig RawConfig) (string, error) {
	myProxy := buildProxySettings(chosenConfig) // Используем существующую функцию

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
		// Вот главное отличие: вместо socks/http создаем tun-интерфейс
		Inbounds: []Inbound{
			{
				Type:        "tun",
				Tag:         "tun-in",
				Address:     []string{"172.19.0.1/30"},
				AutoRoute:   true,
				StrictRoute: true,
				Stack:       "system", // В Windows/Linux работает отлично
			},
		},
		Outbounds: []Outbound{
			myProxy,
			{Type: "direct", Tag: "direct"},
			{Type: "block", Tag: "block"},
		},
		Route: &RouteConfig{
			Rules: []RouteRule{
				{Action: "sniff"},
				{Action: "hijack-dns", Protocol: "dns"},
			},
			Final:                 "proxy",
			DefaultDomainResolver: "dns_proxy",
			AutoDetectInterface:   true,
		},
	}

	jsonBytes, err := json.MarshalIndent(fullConfig, "", "  ")
	if err != nil {
		return "", fmt.Errorf("[ERROR] Json Error: %v", err)
	}

	var baseDir string
	exePath, err := os.Executable()

	if err == nil && !strings.Contains(exePath, "go-build") {
		baseDir = filepath.Dir(exePath)
	} else {
		baseDir, err = os.Getwd()
		if err != nil {
			baseDir = "."
		}
	}

	configDir := filepath.Join(baseDir, "cfg")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", fmt.Errorf("[ERROR] Can`t create folder: %v", err)
	}

	finalFilePath := filepath.Join(configDir, chosenConfig.Name+"_tun_config.json")
	if err := os.WriteFile(finalFilePath, jsonBytes, 0644); err != nil {
		return "", fmt.Errorf("[ERROR] Can`t write json file: %v", err)
	}

	return finalFilePath, nil
}
