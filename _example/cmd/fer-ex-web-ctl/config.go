// Copyright 2017 The fer Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"net"
	"strconv"

	"github.com/alice-go/fer/config"
	"golang.org/x/xerrors"
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

func getSPSConfig(transport string) (config.Config, error) {
	var cfg config.Config

	port1, err := getTCPPort()
	if err != nil {
		return cfg, xerrors.Errorf("error getting free TCP port: %w", err)
	}
	port2, err := getTCPPort()
	if err != nil {
		return cfg, xerrors.Errorf("error getting free TCP port: %w", err)
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
