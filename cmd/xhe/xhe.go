package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"

	"github.com/lainio/err2/try"
	"github.com/shynome/wgortc"
	"golang.org/x/sys/unix"
	"golang.zx2c4.com/wireguard/device"
	"golang.zx2c4.com/wireguard/ipc"
	"golang.zx2c4.com/wireguard/tun"
	"gopkg.in/yaml.v3"
	"remoon.net/xhe/pkg/config"
	"remoon.net/xhe/signaler"
)

var args struct {
	config string

	tdev     string
	mtu      int
	logLevel int
	doh      string
	ices     string
}

var name = fmt.Sprintf("xhe %s", Version)
var f = flag.NewFlagSet(name, flag.ExitOnError)

func initFlags() {
	f.StringVar(&args.tdev, "tdev", "xhe", "tun dev filename")
	f.IntVar(&args.mtu, "mtu", defaultMTU, "dev mtu")
	f.StringVar(&args.doh, "doh", "1.1.1.1", "custom doh server")
	f.StringVar(&args.ices, "ices", "", "ices server")
	f.StringVar(&args.config, "config", "xhe.yaml", "yaml config")
}

const defaultMTU = 1200 - (1500 - device.DefaultMTU)

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

	var c config.Config
	{ // read config file
		b := try.To1(os.ReadFile(args.config))
		try.To(yaml.Unmarshal(b, &c))
	}

	if c.Link != "" {
		c.Links = append(c.Links, c.Link)
	}
	server := signaler.New(c.Links, opts...)
	bind := wgortc.NewBind(server)
	if ices := args.ices; ices != "" {
		try.To(json.Unmarshal([]byte(ices), &bind.ICEServers))
	}
	tdev := try.To1(tun.CreateTUN(args.tdev, args.mtu))
	dev := device.NewDevice(tdev, bind, device.NewLogger(logLevel, "xhe "))
	defer dev.Close()

	try.To(dev.IpcSet(c.String()))
	try.To(dev.Up())

	addrs := c.Addrs
	if c.Address != "" {
		addrs = append(addrs, c.Address)
	}
	if len(addrs) > 0 {
		for _, addr := range addrs {
			try.To(exec.Command("ip", "addr", "add", addr, "dev", args.tdev).Run())
		}
		try.To(exec.Command("ip", "link", "set", "dev", args.tdev, "up").Run())
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

	log.Printf("%s: uapi %s start\n", f.Name(), args.tdev)
	select {
	case <-term:
	case <-errs:
	case <-dev.Wait():
	}

}
