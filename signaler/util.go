package signaler

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/shynome/go-doh-client"
)

func getDomain(ep string) (string, error) {
	ep = fmt.Sprintf("xhe://%s", ep)
	u, err := url.Parse(ep)
	if err != nil {
		return "", err
	}
	return u.Hostname(), nil
}

func getEndpoint(r *doh.Resolver, domain string) (ep string, err error) {
	rr, _, err := r.LookupURI(domain)
	if err != nil {
		return "", err
	}
	if len(rr) == 0 {
		return "", ErrNoEndpoint
	}
	return rr[0].Target, nil
}

var ErrNoEndpoint = errors.New("no endpoint")

// newReq 手动将 user:pass 设置 header 里, 兼容浏览器
func newReq(method string, link string, body io.Reader) (req *http.Request, err error) {
	u, err := url.Parse(link)
	if err != nil {
		return
	}
	userinfo := u.User
	u.User = nil
	link = u.String()
	if req, err = http.NewRequest(method, link, body); err != nil {
		return
	}
	if user := userinfo.Username(); user != "" {
		pass, _ := userinfo.Password()
		req.SetBasicAuth(user, pass)
	}
	return
}
