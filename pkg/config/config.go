package config

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"fmt"
)

type Config struct {
	Device `yaml:",omitempty,inline"`
	Peers  []Peer `yaml:"Peers"`
}

type Device struct {
	PrivateKey string `yaml:"PrivateKey" json:"PrivateKey"`
	ListenPort uint16 `yaml:"ListenPort,omitempty" json:"ListenPort,omitempty"`
	Address    string `yaml:"Address,omitempty" json:"Address,omitempty"`

	Addrs []string `yaml:"Addrs" json:"Addrs,omitempty"`
	Link  string   `yaml:"Link,omitempty" json:"Link,omitempty"`
}

type Peer struct {
	PublicKey    string   `yaml:"PublicKey" json:"PublicKey"`
	AllowedIPs   []string `yaml:"AllowedIPs" json:"AllowedIPs"`
	PresharedKey string   `yaml:"PresharedKey,omitempty" json:"PresharedKey,omitempty"`
	Endpoint     string   `yaml:"Endpoint,omitempty" json:"Endpoint,omitempty"`

	PersistentKeepalive string `yaml:"PersistentKeepalive,omitempty" json:"PersistentKeepalive,omitempty"`
}

func (d Device) String() string {
	var b bytes.Buffer
	fmt.Fprintf(&b, "private_key=%s\n", key2hex(d.PrivateKey))
	fmt.Fprintf(&b, "listen_port=%d\n", d.ListenPort)
	return b.String()
}

func (p Peer) String() string {
	var b bytes.Buffer
	fmt.Fprintf(&b, "public_key=%s\n", key2hex(p.PublicKey))
	if p.PresharedKey != "" {
		fmt.Fprintf(&b, "preshared_key=%s\n", key2hex(p.PresharedKey))
	}
	if len(p.AllowedIPs) > 0 {
		for _, ip := range p.AllowedIPs {
			fmt.Fprintf(&b, "allowed_ip=%s\n", ip)
		}
	}
	if p.Endpoint != "" {
		fmt.Fprintf(&b, "endpoint=%s\n", p.Endpoint)
	}
	if p.PersistentKeepalive != "" {
		fmt.Fprintf(&b, "persistent_keepalive_interval=%s\n", p.PersistentKeepalive)
	}
	return b.String()
}

func (c Config) String() string {
	var s = ""
	s += c.Device.String()
	for _, peer := range c.Peers {
		s += peer.String()
	}
	return s
}

func key2hex(s string) string {
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return s
	}
	return hex.EncodeToString(b)
}
