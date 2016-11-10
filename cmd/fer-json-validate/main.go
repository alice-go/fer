// Copyright 2016 The fer Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// fer-json-validate validates JSON configuration files honor FairMQ's JSON schema.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"reflect"
)

type config struct {
	Options options `json:"fairMQOptions"`
}

type options struct {
	Devices []device `json:"devices,omitempty"`
	Device  device   `json:"device"`
}

func (opts options) MarshalJSON() ([]byte, error) {
	if opts.Device.isZero() {
		return json.Marshal(struct {
			Devices []device `json:"devices"`
		}{
			Devices: opts.Devices,
		})
	}
	return json.Marshal(struct {
		Devices []device `json:"devices,omitempty"`
		Device  device   `json:"device"`
	}{
		Devices: opts.Devices,
		Device:  opts.Device,
	})
}

type device struct {
	ID       string    `json:"id,omitempty"`
	Key      string    `json:"key,omitempty"`
	Channels []channel `json:"channels,omitempty"`
	Channel  channel   `json:"channel"`
}

func (dev *device) isZero() bool {
	return dev.ID == "" && dev.Key == "" && len(dev.Channels) == 0
}

func (dev *device) name() string {
	if dev.ID != "" {
		return dev.ID
	}
	return dev.Key
}

func (dev device) MarshalJSON() ([]byte, error) {
	if dev.Channel.isZero() {
		return json.Marshal(struct {
			ID       string    `json:"id,omitempty"`
			Key      string    `json:"key,omitempty"`
			Channels []channel `json:"channels,omitempty"`
		}{
			ID:       dev.ID,
			Key:      dev.Key,
			Channels: dev.Channels,
		})
	}
	return json.Marshal(struct {
		ID       string    `json:"id,omitempty"`
		Key      string    `json:"key,omitempty"`
		Channels []channel `json:"channels,omitempty"`
		Channel  channel   `json:"channel"`
	}{
		ID:       dev.ID,
		Key:      dev.Key,
		Channels: dev.Channels,
		Channel:  dev.Channel,
	})
}

type channel struct {
	Name    string   `json:"name"`
	Sockets []socket `json:"sockets,omitempty"`
	Socket  socket   `json:"socket"`
}

func (ch *channel) MarshalJSON() ([]byte, error) {
	if ch.Socket.isZero() {
		return json.Marshal(struct {
			Name    string   `json:"name"`
			Sockets []socket `json:"sockets,omitempty"`
		}{
			Name:    ch.Name,
			Sockets: ch.Sockets,
		})
	}

	return json.Marshal(struct {
		Name    string   `json:"name"`
		Sockets []socket `json:"sockets,omitempty"`
		Socket  socket   `json:"socket"`
	}{
		Name:    ch.Name,
		Sockets: ch.Sockets,
		Socket:  ch.Socket,
	})
}

func (ch *channel) isZero() bool {
	return ch.Name == ""
}

type socket struct {
	Type        string `json:"type,omitempty"`
	Method      string `json:"method,omitempty"`
	Address     string `json:"address,omitempty"`
	SendBufSize string `json:"sndBufSize,omitempty"`
	RecvBufSize string `json:"rcvBufSize,omitempty"`
	RateLogging string `json:"rateLogging,omitempty"`
}

func (sck *socket) isZero() bool {
	return *sck == socket{}
}

func main() {
	verbose := flag.Bool("v", false, "enable verbose mode")

	flag.Parse()

	if len(flag.Args()) == 0 {
		flag.Usage()
		os.Exit(1)
	}

	log.SetPrefix("fer-json-validate: ")
	log.SetFlags(0)

	validate(*verbose)
	// run(*verbose)
}

func run(verbose bool) {
	fname := flag.Arg(0)
	f, err := os.Open(fname)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	var cfg config
	err = json.NewDecoder(f).Decode(&cfg)
	if err != nil {
		log.Fatalf("error parsing [%s]: %v\n", fname, err)
	}

	allgood := true
	devices := append([]device(nil), cfg.Options.Devices...)
	if !cfg.Options.Device.isZero() {
		allgood = false
		if verbose {
			log.Printf("%s: use of \"device\" keyword\n", fname)
		}
		devices = append(devices, cfg.Options.Device)
	}

	for _, dev := range devices {
		chans := append([]channel(nil), dev.Channels...)
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

func validate(verbose bool) {
	fname := flag.Arg(0)
	f, err := os.Open(fname)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	w := new(bytes.Buffer)
	var cfg config
	err = json.NewDecoder(f).Decode(&cfg)
	if err != nil {
		log.Fatal(err)
	}
	err = json.NewEncoder(w).Encode(cfg)
	if err != nil {
		log.Fatal(err)
	}

	chk := linearize(w)

	_, err = f.Seek(0, 0)
	if err != nil {
		log.Fatal(err)
	}
	ref := linearize(f)

	if !reflect.DeepEqual(ref, chk) {
		log.Fatalf("[%s] validation FAILED:\nref=%v\nchk=%v\n", fname, string(ref), string(chk))
	}
}

func linearize(r io.Reader) []byte {
	str := new(bytes.Buffer)
	dec := json.NewDecoder(r)
	for {
		t, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		switch t := t.(type) {
		case json.Delim:
			fmt.Fprintf(str, "%v", t)
		default:
			s := ""
			if dec.More() {
				s = " "
			}
			fmt.Fprintf(str, "%v%s", t, s)
		}
	}
	return str.Bytes()
}
