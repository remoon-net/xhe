package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"

	"github.com/lainio/err2/try"
	"github.com/remoon-net/xhe/pkg/config"
	"github.com/remoon-net/xhe/signaler"
	"github.com/shynome/wgortc"
	"golang.org/x/sys/unix"
	"golang.zx2c4.com/wireguard/device"
	"golang.zx2c4.com/wireguard/ipc"
	"golang.zx2c4.com/wireguard/tun"
	"gopkg.in/yaml.v3"
)

var args struct {
	link   string
	config string

	tdev     string
	logLevel int
	doh      string
	ices     string
}

var name = fmt.Sprintf("xhe %s", Version)
var f = flag.NewFlagSet(name, flag.ExitOnError)

func initFlags() {
	f.StringVar(&args.link, "link", os.Getenv("WG_XHE_LINK"), "server addr")
	f.StringVar(&args.tdev, "tdev", "xhe", "tun dev filename")
	f.StringVar(&args.doh, "doh", "dns.alidns.com", "custom doh server")
	f.StringVar(&args.ices, "ices", "", "ices server")
	f.StringVar(&args.config, "config", "", "yaml config")
}

func main() {
	initFlags()
	f.Parse(os.Args[1:])

	logLevel := func() int {
		switch os.Getenv("LOG_LEVEL") {
		case "verbose", "debug":
			return device.LogLevelVerbose
		case "error":
			return device.LogLevelError
		case "silent":
			return device.LogLevelSilent
		}
		return device.LogLevelError
	}()

	opts := []signaler.OptionApply{}
	if args.doh != "" {
		opts = append(opts, signaler.WithDOHServer(args.doh))
	}
	server := signaler.New(args.link, opts...)
	bind := wgortc.NewBind(server)
	if ices := args.ices; ices != "" {
		try.To(json.Unmarshal([]byte(ices), &bind.ICEServers))
	}
	tdev := try.To1(tun.CreateTUN(args.tdev, device.DefaultMTU))
	dev := device.NewDevice(tdev, bind, device.NewLogger(logLevel, "xhe "))
	defer dev.Close()

	if args.config != "" {
		b := try.To1(os.ReadFile(args.config))
		var c config.Config
		try.To(yaml.Unmarshal(b, &c))
		try.To(dev.IpcSet(c.String()))
		try.To(dev.Up())
	}

	errs := make(chan error)
	term := make(chan os.Signal, 1)

	fileUAPI := try.To1(ipc.UAPIOpen(args.tdev))
	uapi := try.To1(ipc.UAPIListen(args.tdev, fileUAPI))
	defer uapi.Close()
	go func() {
		for {
			conn, err := uapi.Accept()
			if err != nil {
				errs <- err
				return
			}
			go dev.IpcHandle(conn)
		}
	}()

	signal.Notify(term, unix.SIGTERM)
	signal.Notify(term, os.Interrupt)

	fmt.Printf("%s: uapi %s start\n", f.Name(), args.tdev)
	select {
	case <-term:
	case <-errs:
	case <-dev.Wait():
	}

}
