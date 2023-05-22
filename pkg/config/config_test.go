package config

import (
	"testing"

	"github.com/lainio/err2/assert"
)

func TestConfig(t *testing.T) {
	c := Config{
		Device: Device{
			PrivateKey: "aCykG/rNYDq6h8elhUBgxdnxhU9JZcWt+tXxQlzMCWU=",
		},
		Peers: []Peer{
			{
				PublicKey:  "WPyGz58E5tju7DH1CdPz2bQKMtiT3gBwOof+KnVlHmw=",
				AllowedIPs: []string{"fdd9:f800::2"},
			},
			{
				PublicKey:    "FsojecwzyHD9HQr+Mknl2Klg8RRsuP5c+RRkIjADTAM=",
				PresharedKey: "kNuohsA/3ziSyKJGzJdSCtcS9KJI1QRcbARcpCyVp2Q=",
				AllowedIPs:   []string{"fdd9:f800::3", "fdd9:f800::4"},
			},
		},
	}
	s := c.String()
	expectCfg := `private_key=682ca41bfacd603aba87c7a5854060c5d9f1854f4965c5adfad5f1425ccc0965
listen_port=0
public_key=58fc86cf9f04e6d8eeec31f509d3f3d9b40a32d893de00703a87fe2a75651e6c
allowed_ip=fdd9:f800::2
public_key=16ca2379cc33c870fd1d0afe3249e5d8a960f1146cb8fe5cf914642230034c03
preshared_key=90dba886c03fdf3892c8a246cc97520ad712f4a248d5045c6c045ca42c95a764
allowed_ip=fdd9:f800::3
allowed_ip=fdd9:f800::4
`
	assert.Equal(s, expectCfg)
	t.Log(s)
}
