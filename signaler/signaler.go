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
	"sync"
	"time"

	"github.com/donovanhide/eventsource"
	"github.com/lainio/err2"
	"github.com/lainio/err2/try"
	"github.com/shynome/go-doh-client"
	"github.com/shynome/wgortc/signaler"
)

type Signaler struct {
	links []string
	items []*StreamItem

	doh    *doh.Resolver
	client *http.Client
}

var _ signaler.Channel = (*Signaler)(nil)

type StreamItem struct {
	*eventsource.Stream
	link string
}

func New(links []string, options ...OptionApply) *Signaler {
	s := &Signaler{
		links: links,
		doh: &doh.Resolver{
			Host:  "1.1.1.1",
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
	if len(s.links) == 0 {
		cch := make(chan signaler.Session)
		close(cch)
		return cch, nil
	}
	var wg sync.WaitGroup
	wg.Add(len(s.links))
	for _, link := range s.links {
		go func(link string) {
			defer wg.Done()
			defer err2.Catch()
			req := try.To1(newReq(http.MethodGet, link, http.NoBody))
			stream := try.To1(eventsource.SubscribeWith("", s.getClient(), req))
			s.items = append(s.items, &StreamItem{Stream: stream, link: link})
		}(link)
	}
	wg.Wait()

	offerCh := make(chan signaler.Session)
	go func() {
		defer close(offerCh)
		var wg sync.WaitGroup
		wg.Add(len(s.items))
		for _, item := range s.items {
			go func(stream *eventsource.Stream, link string) {
				defer wg.Done()
				for ev := range stream.Events {
					go func(ev eventsource.Event) {
						var sdp signaler.SDP
						if err := json.Unmarshal([]byte(ev.Data()), &sdp); err != nil {
							return
						}
						offerCh <- &Session{
							Signaler: s,

							sdp:  sdp,
							id:   ev.Id(),
							link: link,
						}
					}(ev)
				}
			}(item.Stream, item.link)
		}
		wg.Wait()
	}()
	go func() {
		var wg sync.WaitGroup
		wg.Add(len(s.items))
		for _, item := range s.items {
			go func(stream *eventsource.Stream) {
				defer wg.Done()
				for err := range stream.Errors {
					log.Println("eventsource err:", err)
				}
			}(item.Stream)
		}
		wg.Wait()
	}()
	return offerCh, nil
}

func (s *Signaler) Close() error {
	for _, stream := range s.items {
		stream.Close()
	}
	return nil
}

type Session struct {
	*Signaler
	sdp signaler.SDP
	id  string

	link string
}

var _ signaler.Session = (*Session)(nil)

func (s *Session) Description() (offer signaler.SDP) { return s.sdp }
func (s *Session) Resolve(answer *signaler.SDP) (err error) {
	defer err2.Handle(&err)
	body := try.To1(json.Marshal(answer))

	req := try.To1(newReq(http.MethodDelete, s.link, bytes.NewReader(body)))
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req = req.WithContext(ctx)

	req.Header.Set("X-Event-Id", s.id)
	try.To1(s.request(req))
	return
}
func (*Session) Reject(err error) { return }
