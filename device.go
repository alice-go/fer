// Copyright 2016 The fer Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package fer

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/sbinet-alice/fer/config"
	"github.com/sbinet-alice/fer/mq"
	_ "github.com/sbinet-alice/fer/mq/nanomsg" // load nanomsg plugin
	_ "github.com/sbinet-alice/fer/mq/zeromq"  // load zeromq plugin
)

// FIXME(sbinet) use a per-device stdout
//var stdout = bufio.NewWriter(os.Stdout)
var stdout = os.Stdout

type channel struct {
	cfg config.Channel
	sck mq.Socket
	cmd chan Cmd
	msg chan Msg
	log *log.Logger
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

func (ch *channel) run(ctx context.Context) {
	typ := mq.SocketTypeFrom(ch.cfg.Sockets[0].Type)
	// ch.log.Printf("--- run [%v]\n", typ)
	switch typ {
	case mq.Pub, mq.Push, mq.Sub:
		go func() {
			for msg := range ch.msg {
				if len(msg.Data) <= 0 {
					continue
				}
				_, err := ch.Send(msg.Data)
				if err != nil {
					ch.log.Fatalf("send error: %v\n", err)
				}
			}
		}()
	}

	switch typ {
	case mq.Pub, mq.Pull, mq.Sub:
		go func() {
			for {
				data, err := ch.Recv()
				ch.msg <- Msg{data, err}
				if err != nil {
					ch.log.Fatalf("recv error: %v\n", err)
				}
			}
		}()
	}

	for {
		select {
		case cmd := <-ch.cmd:
			switch cmd {
			case CmdEnd:
				return
			}
		case <-ctx.Done():
			return
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

func newChannel(drv mq.Driver, cfg config.Channel, dev *device) (channel, error) {
	ch := channel{
		cmd: make(chan Cmd),
		cfg: cfg,
		log: log.New(stdout, dev.name+"."+cfg.Name+": ", 0),
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

type msgAddr struct {
	name string
	id   int
}

type device struct {
	name  string
	cfg   config.Device
	chans map[string][]channel
	done  chan Cmd
	quit  chan error
	cmds  chan Cmd
	msgs  map[msgAddr]chan Msg
	msg   *log.Logger

	mu  sync.Mutex
	usr Device
}

func newDevice(ctx context.Context, cfg config.Config, udev Device, r io.Reader) (*device, error) {
	drv, err := mq.Open(cfg.Transport)
	if err != nil {
		return nil, err
	}
	dcfg, ok := cfg.Options.Device(cfg.ID)
	if !ok {
		return nil, fmt.Errorf("fer: no such device %q", cfg.ID)
	}

	msg := log.New(stdout, dcfg.Name()+": ", 0)
	msg.Printf("--- new device: %v\n", dcfg)
	dev := device{
		name:  dcfg.Name(),
		cfg:   dcfg,
		chans: make(map[string][]channel),
		done:  make(chan Cmd),
		quit:  make(chan error),
		cmds:  make(chan Cmd),
		msgs:  make(map[msgAddr]chan Msg),
		msg:   msg,
		usr:   udev,
	}

	for _, opt := range dcfg.Channels {
		// dev.msg.Printf("--- new channel: %v\n", opt)
		ch, err := newChannel(drv, opt, &dev)
		if err != nil {
			return nil, err
		}
		ch.msg = make(chan Msg)
		dev.chans[opt.Name] = []channel{ch}
		dev.msgs[msgAddr{name: opt.Name, id: 0}] = ch.msg
	}

	go dev.input(ctx, r)
	go dev.dispatch(ctx)

	return &dev, nil
}

func (dev *device) dispatch(ctx context.Context) {
	var err error
loop:
	for {
		select {
		case <-ctx.Done():
			err = ctx.Err()
			break loop
		case cmd := <-dev.cmds:
			// dev.msg.Printf("received command %d\n", int(cmd))
			switch cmd {
			case CmdInitDevice:
				dev.initDevice(ctx)
			case CmdInitTask:
			case CmdRun:
				go dev.runDevice(ctx)
			case CmdPause:
			case CmdStop:
			case CmdResetTask:
			case CmdResetDevice:
			case CmdEnd:
				dev.done <- cmd
				break loop
			case CmdError:
				dev.done <- cmd
				break loop
			default:
				panic(fmt.Errorf("fer: invalid cmd value (command=%d)", int(cmd)))
			}
		}
	}
	dev.quit <- err
}

func (dev *device) input(ctx context.Context, r io.Reader) {
	var err error
	scan := bufio.NewScanner(r)
	//scan.Split(bufio.ScanBytes)
	for scan.Scan() {
		err = scan.Err()
		if err != nil {
			break
		}
		buf := scan.Bytes()
		if len(buf) == 0 {
			continue
		}
		switch buf[0] {
		case 'i':
			dev.cmds <- CmdInitDevice
		case 'j':
			dev.cmds <- CmdInitTask
		case 'p':
			dev.cmds <- CmdPause
		case 'r':
			dev.cmds <- CmdRun
		case 's':
			dev.cmds <- CmdStop
		case 't':
			dev.cmds <- CmdResetTask
		case 'd':
			dev.cmds <- CmdResetDevice
		case 'h':
			// FIXME(sbinet): print interactive state loop help
		case 'q':
			dev.cmds <- CmdStop
			dev.cmds <- CmdResetTask
			dev.cmds <- CmdResetDevice
			dev.cmds <- CmdEnd
			return
		default:
			dev.msg.Printf("invalid input [%q]\n", string(buf))
		}
	}

	if err == io.EOF {
		err = nil
	}

	if err != nil {
		panic(err)
	}
}

func (dev *device) Chan(name string, i int) (chan Msg, error) {
	msg, ok := dev.msgs[msgAddr{name, i}]
	if !ok {
		return nil, fmt.Errorf("fer: no such channel (name=%q index=%d)", name, i)
	}
	return msg, nil
}

func (dev *device) Done() chan Cmd {
	return dev.done
}

func (dev *device) isControler() {}

func (dev *device) Fatalf(format string, v ...interface{}) {
	dev.msg.Fatalf(format, v...)
}

func (dev *device) Printf(format string, v ...interface{}) {
	dev.msg.Printf(format, v...)
}

func (dev *device) initDevice(ctx context.Context) {
	dev.mu.Lock()
	err := dev.usr.Init(dev)
	dev.mu.Unlock()
	if err != nil {
		dev.quit <- err
	}
}

func (dev *device) runDevice(ctx context.Context) {
	//dev.mu.Lock()
	err := dev.usr.Run(dev)
	//dev.mu.Unlock()
	if err != nil {
		dev.quit <- err
	}
}

func (dev *device) stopDevice(ctx context.Context) {
	for _, chans := range dev.chans {
		for _, ch := range chans {
			ch.cmd <- CmdEnd
		}
	}
}

func (dev *device) run(ctx context.Context) error {
	err := dev.usr.Configure(dev.cfg)
	if err != nil {
		return err
	}

	for _, chans := range dev.chans {
		// dev.msg.Printf("--- init channels [%s]...\n", n)
		for _, ch := range chans {
			// dev.msg.Printf("--- init channel[%s][%d]...\n", n, i)
			sck := ch.cfg.Sockets[0]
			switch strings.ToLower(sck.Method) {
			case "bind":
				go func() {
					err := ch.sck.Listen(sck.Address)
					if err != nil {
						dev.quit <- err
					}
				}()
			case "connect":
				go func() {
					err := ch.sck.Dial(sck.Address)
					if err != nil {
						dev.quit <- err
					}
				}()
			default:
				go func() {
					dev.quit <- fmt.Errorf("fer: invalid socket method (value=%q)", sck.Method)
				}()
			}
		}
	}

	for _, chans := range dev.chans {
		// dev.msg.Printf("--- start channels [%s]...\n", n)
		for i := range chans {
			go chans[i].run(ctx)
		}
	}

	defer dev.stopDevice(ctx)

	return <-dev.quit
}
