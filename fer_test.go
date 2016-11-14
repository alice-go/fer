// Copyright 2016 The fer Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package fer

import (
	"context"
	"io"
	"net"
	"os"
	"strconv"
	"testing"

	"github.com/sbinet-alice/fer/config"
	"golang.org/x/sync/errgroup"
)

func getTCPPort() (string, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return "", err
	}
	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return "", err
	}
	defer l.Close()
	return strconv.Itoa(l.Addr().(*net.TCPAddr).Port), nil
}

func runSamplerProcessorSink(t *testing.T, transport string) {

	stdin := os.Stdin

	port1, err := getTCPPort()
	if err != nil {
		t.Fatalf("error getting free TCP port: %v\n", err)
	}
	port2, err := getTCPPort()
	if err != nil {
		t.Fatalf("error getting free TCP port: %v\n", err)
	}

	cfg := config.Config{
		Transport: transport,
		Options: config.Options{
			Devices: []config.Device{
				{
					ID: "sampler1",
					Channels: []config.Channel{
						{
							Name: "data1",
							Sockets: []config.Socket{
								{
									Type:    "push",
									Method:  "bind",
									Address: "tcp://*:" + port1,
								},
							},
						},
					},
				},
				{
					Key: "processor",
					Channels: []config.Channel{
						{
							Name: "data1",
							Sockets: []config.Socket{
								{
									Type:    "pull",
									Method:  "connect",
									Address: "tcp://localhost:" + port1,
								},
							},
						},
						{
							Name: "data2",
							Sockets: []config.Socket{
								{
									Type:    "push",
									Method:  "connect",
									Address: "tcp://localhost:" + port2,
								},
							},
						},
					},
				},
				{
					ID: "sink1",
					Channels: []config.Channel{
						{
							Name: "data2",
							Sockets: []config.Socket{
								{
									Type:    "pull",
									Method:  "bind",
									Address: "tcp://*:" + port2,
								},
							},
						},
					},
				},
			},
		},
	}

	grp, ctx := errgroup.WithContext(context.Background())
	newTestDevice := func(id string, dev Device) *device {
		cfg := cfg
		cfg.ID = id
		sys, err := newDevice(ctx, cfg, dev, stdin)
		if err != nil {
			t.Fatalf("error creating device %q: %v\n", id, err)
		}
		return sys
	}

	done := make(chan int)
	dev1 := newTestDevice("sampler1", &sampler{done: done})
	dev2 := newTestDevice("processor", &processor{})
	dev3 := newTestDevice("sink1", &sink{})

	grp.Go(func() error { return dev1.run(ctx) })
	grp.Go(func() error { return dev2.run(ctx) })
	grp.Go(func() error { return dev3.run(ctx) })

	broadcast(CmdInitDevice, dev1, dev2, dev3)
	broadcast(CmdRun, dev1, dev2, dev3)

	go func() {
		<-done
		broadcast(CmdEnd, dev1, dev2, dev3)
	}()

	err = grp.Wait()
	if err != nil {
		if o, ok := io.Writer(stdout).(interface {
			Flush() error
		}); ok {
			o.Flush()
		}
		t.Fatalf("unexpected error value: %v\n", err)
	}
}

func TestSamplerProcessorSinkZMQ(t *testing.T) { runSamplerProcessorSink(t, "zeromq") }
func TestSamplerProcessorSinkNN(t *testing.T)  { runSamplerProcessorSink(t, "nanomsg") }

type sampler struct {
	cfg   config.Device
	datac chan Msg
	done  chan int
}

func (dev *sampler) Configure(cfg config.Device) error {
	dev.cfg = cfg
	return nil
}

func (dev *sampler) Init(ctl Controler) error {
	datac, err := ctl.Chan("data1", 0)
	if err != nil {
		return err
	}

	dev.datac = datac
	return nil
}

func (dev *sampler) Run(ctl Controler) error {
	i := 0
	for {
		select {
		case dev.datac <- Msg{Data: []byte("HELLO")}:
			i++
		case <-ctl.Done():
			return nil
		}
		if i >= 10 {
			dev.datac = nil
			dev.done <- 1
		}
	}
	return nil
}

func (dev *sampler) Pause(ctl Controler) error {
	return nil
}

func (dev *sampler) Reset(ctl Controler) error {
	return nil
}

type processor struct {
	cfg    config.Device
	idatac chan Msg
	odatac chan Msg
}

func (dev *processor) Configure(cfg config.Device) error {
	dev.cfg = cfg
	return nil
}

func (dev *processor) Init(ctl Controler) error {
	idatac, err := ctl.Chan("data1", 0)
	if err != nil {
		return err
	}

	odatac, err := ctl.Chan("data2", 0)
	if err != nil {
		return err
	}

	dev.idatac = idatac
	dev.odatac = odatac
	return nil
}

func (dev *processor) Run(ctl Controler) error {
	for {
		select {
		case data := <-dev.idatac:
			ctl.Printf("received: %q\n", string(data.Data))
			out := append([]byte(nil), data.Data...)
			out = append(out, []byte(" (modified by "+dev.cfg.ID+")")...)
			dev.odatac <- Msg{Data: out}
		case <-ctl.Done():
			return nil
		}
	}
}

func (dev *processor) Pause(ctl Controler) error {
	return nil
}

func (dev *processor) Reset(ctl Controler) error {
	return nil
}

type sink struct {
	cfg   config.Device
	datac chan Msg
}

func (dev *sink) Configure(cfg config.Device) error {
	dev.cfg = cfg
	return nil
}

func (dev *sink) Init(ctl Controler) error {
	datac, err := ctl.Chan("data2", 0)
	if err != nil {
		return err
	}

	dev.datac = datac
	return nil
}

func (dev *sink) Run(ctl Controler) error {
	for {
		select {
		case data := <-dev.datac:
			ctl.Printf("received: %q\n", string(data.Data))
		case <-ctl.Done():
			return nil
		}
	}
}

func (dev *sink) Pause(ctl Controler) error {
	return nil
}

func (dev *sink) Reset(ctl Controler) error {
	return nil
}
