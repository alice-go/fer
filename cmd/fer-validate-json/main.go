// Copyright 2016 The fer Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// fer-validate-json validates JSON configuration files honor FairMQ's JSON schema.
package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"
)

type Config struct {
	Options Options `json:"fairMQOptions"`
}

type Options struct {
	Devices []Device `json:"devices"`
	Device  Device   `json:"device"`
}

type Device struct {
	ID       string    `json:"id"`
	Key      string    `json:"key"`
	Channels []Channel `json:"channels"`
	Channel  Channel   `json:"channel"`
}

func (dev *Device) isZero() bool {
	return dev.ID == "" && dev.Key == ""
}

func (dev *Device) name() string {
	if dev.ID != "" {
		return dev.ID
	}
	return dev.Key
}

type Channel struct {
	Name    string   `json:"name"`
	Sockets []Socket `json:"sockets"`
	Socket  Socket   `json:"socket"`
}

func (ch *Channel) isZero() bool {
	return ch.Name == ""
}

type Socket struct {
	Type        string `json:"type"`
	Method      string `json:"method"`
	Address     string `json:"address"`
	SendBufSize string `json:"sndBufSize"`
	RecvBufSize string `json:"rcvBufSize"`
	RateLogging string `json:"rateLogging"`
}

func (sck *Socket) isZero() bool {
	return *sck == Socket{}
}

func main() {
	verbose := flag.Bool("v", false, "enable verbose mode")

	flag.Parse()

	if len(flag.Args()) == 0 {
		flag.Usage()
		os.Exit(1)
	}

	run(*verbose)
}

func run(verbose bool) {
	fname := flag.Arg(0)
	f, err := os.Open(fname)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	var cfg Config
	err = json.NewDecoder(f).Decode(&cfg)
	if err != nil {
		log.Fatalf("error parsing [%s]: %v\n", fname, err)
	}

	allgood := true
	devices := append([]Device(nil), cfg.Options.Devices...)
	if !cfg.Options.Device.isZero() {
		allgood = false
		if verbose {
			log.Printf("%s: use of \"device\" keyword\n", fname)
		}
		devices = append(devices, cfg.Options.Device)
	}

	for _, dev := range devices {
		chans := append([]Channel(nil), dev.Channels...)
		if !dev.Channel.isZero() {
			allgood = false
			if verbose {
				log.Printf("%s: use of \"channel\" keyword (device=%q)\n", fname, dev.name())
			}
			chans = append(chans, dev.Channel)
		}

		for _, ch := range chans {
			//sockets := append([]Socket(nil), dev.Sockets...)
			if !ch.Socket.isZero() {
				allgood = false
				if verbose {
					log.Printf("%s: use of \"socket\" keyword (device=%q)\n", fname, dev.name()+"."+ch.Name)
				}
				//sockets = append(sockets, ch.Socket)
			}
		}
	}

	if !allgood {
		log.Fatalf("[%s] validation FAILED\n", fname)
	}
}
