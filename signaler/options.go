package signaler

import "net/http"

type OptionApply func(*Signaler)

func WithDOHServer(server string) OptionApply {
	return func(s *Signaler) {
		s.doh.Host = server
	}
}

func WithClient(client *http.Client) OptionApply {
	return func(s *Signaler) {
		s.client = client
		s.doh.HTTPClient = client
	}
}

func WithHTTPClient(client *http.Client) OptionApply {
	return func(s *Signaler) {
		s.client = client
	}
}

func WithDOHClient(client *http.Client) OptionApply {
	return func(s *Signaler) {
		s.doh.HTTPClient = client
	}
}
