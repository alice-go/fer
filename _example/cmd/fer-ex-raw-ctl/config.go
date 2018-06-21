// Copyright 2017 The fer Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"net"
	"os"
	"strconv"

	"github.com/sbinet-alice/fer/config"
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
			return "", "", fmt.Errorf("error getting free TCP port: %v\n", err)
		}
		port2, err := getTCPPort()
		if err != nil {
			return "", "", fmt.Errorf("error getting free TCP port: %v\n", err)
		}
		return "tcp://localhost:" + port1, "tcp://localhost:" + port2, nil

	case "ipc":
		os.Remove("raw-ctl-p1")
		os.Remove("raw-ctl-p2")
		return "ipc://raw-ctl-p1", "ipc://raw-ctl-p2", nil
	}
	return "", "", fmt.Errorf("invalid protocol %q", *protocol)
}

func getSPSConfig(transport string) (config.Config, error) {
	var cfg config.Config

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
