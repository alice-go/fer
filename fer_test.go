// Copyright 2016 The fer Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package fer

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/alice-go/fer/config"
	"golang.org/x/sync/errgroup"
)

var testDrivers = []string{"zeromq", "nanomsg"}

func TestSamplerProcessorSink(t *testing.T) {
	for _, n := range testDrivers {
		transport := n
		t.Run("transport="+transport, func(t *testing.T) {

			t.Parallel()

			cfg, err := getSPSConfig(transport)
			if err != nil {
				t.Fatal(err)
			}

			stdin := os.Stdin
			stdout := new(bytes.Buffer)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			grp, ctx := errgroup.WithContext(ctx)
			newTestDevice := func(id string, dev Device) *device {
				cfg := cfg
				cfg.ID = id
				sys, err := newDevice(ctx, cfg, dev, stdin, stdout)
				if err != nil {
					t.Fatalf("error creating device %q: %v\n", id, err)
				}
				return sys
			}

			const N = 1024
			sumc := make(chan string)
			dev1 := newTestDevice("sampler1", &sampler{n: N})
			dev2 := newTestDevice("processor", &processor{})
			dev3 := newTestDevice("sink1", &sink{sum: sumc, n: N})

			grp.Go(func() error { return dev1.run(ctx) })
			grp.Go(func() error { return dev2.run(ctx) })
			grp.Go(func() error { return dev3.run(ctx) })

			broadcast(CmdInitDevice, dev1, dev2, dev3)
			broadcast(CmdRun, dev1, dev2, dev3)

			sum := make([]string, 0, N)
			go func() {
				for s := range sumc {
					sum = append(sum, s)
				}
				broadcast(CmdEnd, dev1, dev2, dev3)
			}()

			err = grp.Wait()
			if err != nil {
				t.Fatalf("unexpected error value: %v\n", err)
			}

			if len(sum) != N {
				t.Fatalf("got %d. want %d\n", len(sum), N)
			}

			want := make([]string, 0, N)
			for i := 0; i < N; i++ {
				want = append(want, fmt.Sprintf("HELLO-%02[1]d (modified by %[2]s - %02[1]d) - %02[1]d", i, dev2.name))
			}
			if !reflect.DeepEqual(sum, want) {

				scan := bufio.NewScanner(stdout)
				for scan.Scan() {
					err = scan.Err()
					if err != nil {
						break
					}
					t.Logf("%v\n", scan.Text())
				}

				t.Fatalf("error comparing outputs\ngot:\n%s\n\nwant:\n%s\n",
					strings.Join(sum, "\n"),
					strings.Join(want, "\n"),
				)
				if err == io.EOF {
					err = nil
				}

				if err != nil {
					t.Fatalf("error scanning stdout: %v\n", err)
				}
			}
		})
	}
}

type sampler struct {
	cfg   config.Device
	datac chan Msg
	n     int
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
		case dev.datac <- Msg{Data: []byte(fmt.Sprintf("HELLO-%02d", i))}:
			i++
		case <-ctl.Done():
			return nil
		}
		if i >= dev.n {
			dev.datac = nil
		}
	}
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
	i := 0
	for {
		select {
		case data := <-dev.idatac:
			out := append([]byte(nil), data.Data...)
			out = append(out, []byte(fmt.Sprintf(" (modified by %s - %02d)", dev.cfg.Name(), i))...)
			dev.odatac <- Msg{Data: out}
			i++
		case <-ctl.Done():
			return nil
		}
	}
}

type sink struct {
	cfg   config.Device
	datac chan Msg
	n     int
	sum   chan string
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
	i := 0
	for {
		select {
		case data := <-dev.datac:
			//ctl.Printf("received: %q (%d)\n", string(data.Data), i)
			dev.sum <- fmt.Sprintf("%s - %02d", string(data.Data), i)
			i++
		case <-ctl.Done():
			return nil
		}
		if i >= dev.n {
			close(dev.sum)
		}
	}
}

func TestDeviceFSM(t *testing.T) {
	for _, n := range testDrivers {
		transport := n
		for _, cmds := range [][]Cmd{
			{CmdInitDevice, CmdRun, CmdPause, CmdStop, CmdEnd},
			{CmdInitDevice, CmdRun, CmdStop, CmdEnd},
			{CmdInitDevice, CmdRun, CmdEnd},
			{CmdInitDevice, CmdEnd},
			{CmdEnd},
		} {
			cmds := cmds
			list := make([]string, len(cmds))
			for i := range cmds {
				list[i] = cmds[i].String()
			}

			t.Run("transport="+transport+";cmds="+strings.Join(list, "|"), func(t *testing.T) {

				t.Parallel()

				const N = 1024
				cfg, err := getSPSConfig(transport)
				if err != nil {
					t.Fatal(err)
				}
				cfg.ID = "sampler1"

				stdin := new(bytes.Buffer)
				stdout := new(bytes.Buffer)

				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				grp, ctx := errgroup.WithContext(ctx)
				dev1, err := newDevice(ctx, cfg, &sampler{n: N}, stdin, stdout)
				if err != nil {
					t.Fatalf("error creating device %q: %v\n", cfg.ID, err)
				}
				grp.Go(func() error { return dev1.run(ctx) })

				for _, cmd := range cmds {
					dev1.cmds <- cmd
				}

				err = grp.Wait()
				if err != nil {
					t.Fatal(err)
				}
			})
		}
	}
}

func TestDeviceFSMFromStdin(t *testing.T) {
	for _, n := range testDrivers {
		transport := n
		for _, cmds := range [][]byte{
			{'i', 'r', 'p', 's', 'q'},
			{'i', 'r', 's', 'q'},
			{'i', 'r', 'q'},
			{'i', 'q'},
			{'q'},
		} {
			cmds := cmds
			list := make([]string, len(cmds))
			for i := range cmds {
				list[i] = string(cmds[i])
			}

			t.Run("transport="+transport+";cmds="+strings.Join(list, "|"), func(t *testing.T) {

				t.Parallel()

				const N = 1024
				cfg, err := getSPSConfig(transport)
				if err != nil {
					t.Fatal(err)
				}
				cfg.ID = "sampler1"

				pr, pw, err := os.Pipe()
				if err != nil {
					t.Fatalf("could not create pipe: %v", err)
				}
				defer pr.Close()
				defer pw.Close()
				stdout := new(bytes.Buffer)

				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				grp, ctx := errgroup.WithContext(ctx)
				dev1, err := newDevice(ctx, cfg, &sampler{n: N}, pr, stdout)
				if err != nil {
					t.Fatalf("error creating device %q: %v\n", cfg.ID, err)
				}
				grp.Go(func() error { return dev1.run(ctx) })

				for _, cmd := range cmds {
					pw.Write([]byte{cmd, '\n'})
				}

				err = grp.Wait()
				if err != nil {
					t.Fatal(err)
				}
			})
		}
	}
}

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

func getSPSConfig(transport string) (config.Config, error) {
	var cfg config.Config

	port1, err := getTCPPort()
	if err != nil {
		return cfg, fmt.Errorf("error getting free TCP port: %v\n", err)
	}
	port2, err := getTCPPort()
	if err != nil {
		return cfg, fmt.Errorf("error getting free TCP port: %v\n", err)
	}

	cfg = config.Config{
		Control:   "interactive",
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

	return cfg, nil
}
