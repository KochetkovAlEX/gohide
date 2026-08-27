package vpn

// struct for url
type RawConfig struct {
	Protocol string
	UUID     string
	Address  string
	Port     string
	Flow     string
	SNI      string
	PBK      string
	SID      string
	FP       string
	Name     string
}

// structs for config
type Config struct {
	Log       *LogConfig   `json:"log,omitempty"`
	DNS       *DNSConfig   `json:"dns,omitempty"`
	Inbounds  []Inbound    `json:"inbounds"`
	Outbounds []Outbound   `json:"outbounds"`
	Route     *RouteConfig `json:"route,omitempty"`
}

type LogConfig struct {
	Level     string `json:"level"`
	Timestamp bool   `json:"timestamp"`
}

type DNSConfig struct {
	Servers []DNSServer `json:"servers"`
	Rules   []DNSRule   `json:"rules,omitempty"`
}

type DNSServer struct {
	Type   string `json:"type"`
	Tag    string `json:"tag"`
	Server string `json:"server"`
}

type DNSRule struct {
	Protocol string `json:"protocol,omitempty"`
	Action   string `json:"action"`
	Server   string `json:"server"`
}

type Inbound struct {
	Type       string `json:"type"` // socks, http, tun
	Tag        string `json:"tag"`
	Listen     string `json:"listen,omitempty"`      // Для прокси: "127.0.0.1"
	ListenPort int    `json:"listen_port,omitempty"` // Для прокси: 10808

	Address     []string `json:"address,omitempty"`
	AutoRoute   bool     `json:"auto_route,omitempty"`   // Для TUN: true
	StrictRoute bool     `json:"strict_route,omitempty"` // Для TUN: true (улучшает перехват трафика)
	Stack       string   `json:"stack,omitempty"`        // Для TUN: "system" или "gvisor"
}

type Outbound struct {
	Type           string     `json:"type"`
	Tag            string     `json:"tag"`
	Server         string     `json:"server,omitempty"`
	ServerPort     int        `json:"server_port,omitempty"`
	UUID           string     `json:"uuid,omitempty"`
	Flow           string     `json:"flow,omitempty"`
	Network        string     `json:"network,omitempty"`         // tcp
	PacketEncoding string     `json:"packet_encoding,omitempty"` // xudp
	TLS            *TLSConfig `json:"tls,omitempty"`
}

type TLSConfig struct {
	Enabled    bool           `json:"enabled"`
	ServerName string         `json:"server_name"`
	UTLS       *UTLSConfig    `json:"utls,omitempty"`
	Reality    *RealityConfig `json:"reality,omitempty"`
}

type UTLSConfig struct {
	Enabled     bool   `json:"enabled"`
	Fingerprint string `json:"fingerprint"` // chrome
}

type RealityConfig struct {
	Enabled   bool   `json:"enabled"`
	PublicKey string `json:"public_key"`
	ShortID   string `json:"short_id"`
}

type RouteConfig struct {
	Rules                 []RouteRule `json:"rules"`
	Final                 string      `json:"final"`                   // proxy
	DefaultDomainResolver string      `json:"default_domain_resolver"` // dns_proxy
	AutoDetectInterface   bool        `json:"auto_detect_interface"`   // true
}

type RouteRule struct {
	Action   string `json:"action,omitempty"`   // sniff
	Protocol string `json:"protocol,omitempty"` // dns
	Outbound string `json:"outbound,omitempty"` // dns-out, direct
}
