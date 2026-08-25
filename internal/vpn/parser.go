package vpn

import (
	"fmt"
	"net/url"
	"strings"
)

func ParseLine(urlString string) (RawConfig, error) {
	u, err := url.Parse(urlString)
	if err != nil {
		return RawConfig{}, err
	}
	if u.Scheme != "vless" {
		return RawConfig{}, fmt.Errorf("[ERROR] Protocol is not allowed")
	}

	query := u.Query()

	cfg := RawConfig{
		Protocol: u.Scheme,
		UUID:     u.User.Username(),
		Address:  u.Hostname(),
		Port:     u.Port(),
		Flow:     query.Get("flow"),
		SNI:      query.Get("sni"),
		PBK:      query.Get("pbk"),
		SID:      query.Get("sid"),
		FP:       query.Get("fp"),
		Name:     strings.TrimSpace(u.Fragment),
	}
	return cfg, nil
}

func GetMapByCountryNames(configs []RawConfig) (map[string]RawConfig, error) {
	if len(configs) < 1 {
		return nil, fmt.Errorf("[ERROR] Config list is empty")
	}
	names := make(map[string]RawConfig)
	for _, cfg := range configs {
		names[cfg.Name] = cfg
	}
	return names, nil
}
