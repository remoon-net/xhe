package signaler

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/lainio/err2/assert"
	"github.com/lainio/err2/try"
	"github.com/shynome/go-doh-client"
)

var testEndpoint = "device1.shynome.hhsdd.com"

func TestGetDomain(t *testing.T) {
	d := try.To1(getDomain(testEndpoint))
	assert.Equal(d, testEndpoint)
	d2 := try.To1(getDomain(fmt.Sprintf("%s:4222", testEndpoint)))
	assert.Equal(d2, testEndpoint)
}

func TestGetEndpoint(t *testing.T) {
	r := &doh.Resolver{
		Host:  "cloudflare-dns.com",
		Class: doh.IN,
	}
	ep := try.To1(getEndpoint(r, testEndpoint))
	t.Log(ep)
}

func TestNewReq(t *testing.T) {
	req := try.To1(newReq(http.MethodGet, "http://hello:world@127.0.0.1/", http.NoBody))
	t.Log(req)
}
