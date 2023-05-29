package signaler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/donovanhide/eventsource"
	"github.com/lainio/err2"
	"github.com/lainio/err2/try"
	"github.com/shynome/go-doh-client"
	"github.com/shynome/wgortc/signaler"
)

type Signaler struct {
	authLink string

	doh    *doh.Resolver
	client *http.Client

	stream *eventsource.Stream
}

var _ signaler.Channel = (*Signaler)(nil)

func New(authLink string, options ...OptionApply) *Signaler {
	s := &Signaler{
		authLink: authLink,
		doh: &doh.Resolver{
			Host:  "dns.alidns.com",
			Class: doh.IN,
		},
	}
	for _, apply := range options {
		apply(s)
	}
	return s
}

func (s *Signaler) getClient() *http.Client {
	if s.client != nil {
		return s.client
	}
	return http.DefaultClient
}

func (s *Signaler) request(req *http.Request) (resp *http.Response, err error) {
	client := s.getClient()
	if resp, err = client.Do(req); err != nil {
		return
	}
	if strings.HasPrefix(resp.Status, "2") {
		return
	}
	var errText []byte
	if errText, err = io.ReadAll(resp.Body); err != nil {
		return
	}
	err = fmt.Errorf("server err. status: %s. content: %s", resp.Status, errText)
	return
}

func (s *Signaler) Handshake(ep string, offer signaler.SDP) (roffer *signaler.SDP, err error) {
	defer err2.Handle(&err)

	if !strings.HasPrefix(ep, "http") {
		ep = try.To1(getDomain(ep))
		ep = try.To1(getEndpoint(s.doh, ep))
	}

	var body bytes.Buffer
	try.To(json.NewEncoder(&body).Encode(offer))

	req := try.To1(newReq(http.MethodPost, ep, &body))
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req = req.WithContext(ctx)

	resp := try.To1(s.request(req))
	var sdp signaler.SDP
	try.To(json.NewDecoder(resp.Body).Decode(&sdp))

	return &sdp, nil
}

func (s *Signaler) Accept() (ch <-chan signaler.Session, err error) {
	defer err2.Handle(&err)
	if s.authLink == "" {
		cch := make(chan signaler.Session)
		close(cch)
		return cch, nil
	}
	req := try.To1(newReq(http.MethodGet, s.authLink, http.NoBody))
	stream := try.To1(eventsource.SubscribeWith("", s.getClient(), req))
	s.stream = stream
	offerCh := make(chan signaler.Session)
	go func() {
		defer close(offerCh)
		for ev := range stream.Events {
			go func(ev eventsource.Event) {
				var sdp signaler.SDP
				if err := json.Unmarshal([]byte(ev.Data()), &sdp); err != nil {
					return
				}
				offerCh <- &Session{
					Signaler: s,

					sdp: sdp,
					id:  ev.Id(),
				}
			}(ev)
		}
	}()
	go func() {
		for err := range stream.Errors {
			log.Println("eventsource err:", err)
		}
	}()
	return offerCh, nil
}

func (s *Signaler) Close() error {
	if stream := s.stream; stream != nil {
		s.stream = nil
		stream.Close()
	}
	return nil
}

type Session struct {
	*Signaler
	sdp signaler.SDP
	id  string
}

var _ signaler.Session = (*Session)(nil)

func (s *Session) Description() (offer signaler.SDP) { return s.sdp }
func (s *Session) Resolve(answer *signaler.SDP) (err error) {
	defer err2.Handle(&err)
	body := try.To1(json.Marshal(answer))

	req := try.To1(newReq(http.MethodDelete, s.authLink, bytes.NewReader(body)))
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req = req.WithContext(ctx)

	req.Header.Set("X-Event-Id", s.id)
	try.To1(s.request(req))
	return
}
func (*Session) Reject(err error) { return }
