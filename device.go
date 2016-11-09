// Copyright 2016 The fer Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package fer

import (
	"fmt"
	"log"
	"strings"

	"github.com/sbinet-alice/fer/config"
	"github.com/sbinet-alice/fer/mq"
	_ "github.com/sbinet-alice/fer/mq/nanomsg" // load nanomsg plugin
	_ "github.com/sbinet-alice/fer/mq/zeromq"  // load zeromq plugin
)

type channel struct {
	cfg config.Channel
	sck mq.Socket
	cmd chan Cmd
	msg chan Msg
}

func (ch *channel) Name() string {
	return ch.cfg.Name
}

func (ch *channel) Send(data []byte) (int, error) {
	err := ch.sck.Send(data)
	return len(data), err
}

func (ch *channel) Recv() ([]byte, error) {
	return ch.sck.Recv()
}

func (ch *channel) run() {
	for {
		select {
		case msg := <-ch.msg:
			_, err := ch.Send(msg.Data)
			if err != nil {
				log.Fatal(err)
			}
		case ch.msg <- ch.recv():
		case cmd := <-ch.cmd:
			switch cmd {
			case CmdEnd:
				return
			}
		}
	}
}

func (ch *channel) recv() Msg {
	data, err := ch.Recv()
	return Msg{
		Data: data,
		Err:  err,
	}
}

func newChannel(drv mq.Driver, cfg config.Channel) (channel, error) {
	ch := channel{
		cmd: make(chan Cmd),
		cfg: cfg,
	}
	// FIXME(sbinet) support multiple sockets to send/recv to/from
	if len(cfg.Sockets) != 1 {
		panic("fer: not implemented")
	}
	typ := mq.SocketTypeFrom(cfg.Sockets[0].Type)
	sck, err := drv.NewSocket(typ)
	if err != nil {
		return ch, err
	}
	ch.sck = sck
	return ch, nil
}

type device struct {
	name  string
	chans map[string][]channel
	cmds  chan Cmd
	msgs  map[msgAddr]chan Msg
}

func newDevice(drv mq.Driver, cfg config.Device) (*device, error) {
	log.Printf("--- new device: %v\n", cfg)
	dev := device{
		chans: make(map[string][]channel),
		cmds:  make(chan Cmd),
		msgs:  make(map[msgAddr]chan Msg),
	}

	for _, opt := range cfg.Channels {
		log.Printf("--- new channel: %v\n", opt)
		ch, err := newChannel(drv, opt)
		if err != nil {
			return nil, err
		}
		ch.msg = make(chan Msg)
		dev.chans[opt.Name] = []channel{ch}
		dev.msgs[msgAddr{name: opt.Name, id: 0}] = ch.msg
	}
	return &dev, nil
}

func (dev *device) Chan(name string, i int) (chan Msg, error) {
	msg, ok := dev.msgs[msgAddr{name, i}]
	if !ok {
		return nil, fmt.Errorf("fer: no such channel (name=%q index=%d)", name, i)
	}
	return msg, nil
}

func (dev *device) Done() chan Cmd {
	return nil
}

func (dev *device) isControler() {}

func (dev *device) run() {
	for n, chans := range dev.chans {
		log.Printf("--- init channels [%s]...\n", n)
		for i, ch := range chans {
			log.Printf("--- init channel[%s][%d]...\n", n, i)
			sck := ch.cfg.Sockets[0]
			switch strings.ToLower(sck.Method) {
			case "bind":
				go func() {
					err := ch.sck.Listen(sck.Address)
					if err != nil {
						log.Fatal(err)
					}
				}()
			case "connect":
				go func() {
					err := ch.sck.Dial(sck.Address)
					if err != nil {
						log.Fatal(err)
					}
				}()
			default:
				log.Fatalf("fer: invalid socket method (value=%q)", sck.Method)
			}
		}
	}

	for n, chans := range dev.chans {
		log.Printf("--- start channels [%s]...\n", n)
		for i := range chans {
			go chans[i].run()
		}
	}

}

type Device interface {
	Configure(cfg config.Device) error
	Init(ctl Controler) error
	Run(ctl Controler) error
	Pause(ctl Controler) error
	Reset(ctl Controler) error
}

type Controler interface {
	Chan(name string, i int) (chan Msg, error)
	Done() chan Cmd

	isControler()
}

type msgAddr struct {
	name string
	id   int
}

type Msg struct {
	Data []byte
	Err  error
}

func Main(dev Device) error {
	cfg, err := config.Parse()
	if err != nil {
		return err
	}

	drvName := cfg.Transport
	if drvName == "" {
		drvName = "zeromq"
	}

	drv, err := mq.Open(drvName)
	if err != nil {
		return err
	}

	devName := cfg.ID
	devCfg, ok := cfg.Options.Device(devName)
	if !ok {
		return fmt.Errorf("fer: no such device %q", devName)
	}

	sys, err := newDevice(drv, devCfg)
	if err != nil {
		return err
	}

	err = dev.Configure(devCfg)
	if err != nil {
		return err
	}

	go sys.run()

	err = dev.Init(sys)
	if err != nil {
		return err
	}

	err = dev.Run(sys)
	if err != nil {
		return err
	}

	return nil
}

func deviceConfig(name string, cfg config.Config) (config.Device, error) {
	var (
		dev config.Device
		err error
	)

	return dev, err
}
