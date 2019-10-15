// Copyright 2017 The fer Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"net"
	"os"
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

func getProtPorts() (string, string, error) {
	switch *protocol {
	case "tcp":
		port1, err := getTCPPort()
		if err != nil {
			return "", "", xerrors.Errorf("error getting free TCP port: %w", err)
		}
		port2, err := getTCPPort()
		if err != nil {
			return "", "", xerrors.Errorf("error getting free TCP port: %w", err)
		}
		return "tcp://localhost:" + port1, "tcp://localhost:" + port2, nil

	case "ipc":
		os.Remove("raw-ctl-p1-" + *transport)
		os.Remove("raw-ctl-p2-" + *transport)
		return "ipc://raw-ctl-p1-" + *transport, "ipc://raw-ctl-p2-" + *transport, nil
	case "inproc":
		return "inproc://raw-ctl-p1", "inproc://raw-ctl-p2", nil
	}
	return "", "", xerrors.Errorf("invalid protocol %q", *protocol)
}

func getSPSConfig(transport string) (config.Config, error) {
	var cfg config.Config

	if transport == "czmq" && *protocol == "tcp" {
		return config.Config{
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
										Address: "tcp://*:5555",
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
										Address: "tcp://localhost:5555",
									},
								},
							},
							{
								Name: "data2",
								Sockets: []config.Socket{
									{
										Type:    "push",
										Method:  "connect",
										Address: "tcp://localhost:5556",
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
										Address: "tcp://*:5556",
									},
								},
							},
						},
					},
				},
			},
		}, nil
	}

	port1, port2, err := getProtPorts()
	if err != nil {
		return cfg, err
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
									Address: port1,
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
									Address: port1,
								},
							},
						},
						{
							Name: "data2",
							Sockets: []config.Socket{
								{
									Type:    "push",
									Method:  "connect",
									Address: port2,
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
									Address: port2,
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
