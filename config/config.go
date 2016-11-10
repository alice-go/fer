// Copyright 2016 The fer Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package config

import (
	"encoding/json"
	"flag"
	"os"
	"strconv"
)

// Parse parses the command-line flags from os.Args[1:]. Must be called after
// all flags are defined and before flags are accessed by the program.
func Parse() (Config, error) {
	var (
		id    = flag.String("id", "", "device ID")
		trans = flag.String("transport", "zeromq", "transport mechanism to use (zeromq, nanomsg, go-chan, ...")
		mq    = flag.String("mq-config", "", "path to JSON file holding device configuration")
	)

	flag.Parse()

	cfg := Config{
		ID:        *id,
		Transport: *trans,
	}

	f, err := os.Open(*mq)
	if err != nil {
		return cfg, err
	}
	defer f.Close()

	err = json.NewDecoder(f).Decode(&cfg)
	if err != nil {
		return cfg, err
	}

	return cfg, err
}

// Config holds the configuration of a Fer program.
type Config struct {
	Options   Options `json:"fairMQOptions"`
	ID        string  `json:"fer_id,omitempty"`
	Transport string  `json:"fer_transport,omitempty"` // zeromq, nanomsg, chan
}

// Options holds the configuration of a Fer MQ program.
type Options struct {
	Devices []Device `json:"devices"`
}

// Device returns the configuration of a device by name.
func (opts Options) Device(name string) (Device, bool) {
	for _, dev := range opts.Devices {
		if dev.ID == name {
			return dev, true
		}
	}
	return Device{}, false
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (opts *Options) UnmarshalJSON(data []byte) error {
	var raw struct {
		Device  Device   `json:"device"`
		Devices []Device `json:"devices"`
	}
	err := json.Unmarshal(data, &raw)
	if err != nil {
		return err
	}
	opts.Devices = opts.Devices[:0]
	if !raw.Device.isZero() {
		opts.Devices = append(opts.Devices, raw.Device)
	}
	opts.Devices = append(opts.Devices, raw.Devices...)
	return nil
}

// Device holds the configuration of a device.
type Device struct {
	ID       string    `json:"id"`
	Channels []Channel `json:"channels"`
}

func (dev Device) isZero() bool {
	return dev.ID == "" && len(dev.Channels) == 0
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (dev *Device) UnmarshalJSON(data []byte) error {
	var raw struct {
		ID       string    `json:"id"`
		Key      string    `json:"key"`
		Channel  Channel   `json:"channel"`
		Channels []Channel `json:"channels"`
	}

	err := json.Unmarshal(data, &raw)
	if err != nil {
		return err
	}
	dev.ID = raw.ID
	if raw.ID == "" && raw.Key != "" {
		dev.ID = raw.Key
	}
	dev.Channels = dev.Channels[:0]
	if !raw.Channel.isZero() {
		dev.Channels = append(dev.Channels, raw.Channel)
	}
	dev.Channels = append(dev.Channels, raw.Channels...)
	return nil
}

// Channel holds the configuration of a channel.
type Channel struct {
	Name    string   `json:"name"`
	Sockets []Socket `json:"sockets"`
}

func (ch Channel) isZero() bool {
	return ch.Name == "" && len(ch.Sockets) == 0
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (ch *Channel) UnmarshalJSON(data []byte) error {
	var raw struct {
		Name    string   `json:"name"`
		Socket  Socket   `json:"socket"`
		Sockets []Socket `json:"sockets"`
	}

	err := json.Unmarshal(data, &raw)
	if err != nil {
		return err
	}
	ch.Name = raw.Name
	ch.Sockets = ch.Sockets[:0]
	if (raw.Socket != Socket{}) {
		ch.Sockets = append(ch.Sockets, raw.Socket)
	}
	ch.Sockets = append(ch.Sockets, raw.Sockets...)
	return nil
}

// Socket holds the configuration of a socket.
type Socket struct {
	Type        string `json:"type"`    // Type is the type of a Socket (PUB/SUB/PUSH/PULL/...)
	Method      string `json:"method"`  // Method to operate the socket (connect/bind)
	Address     string `json:"address"` // Address is the socket end-point
	SendBufSize int    `json:"sndBufSize"`
	RecvBufSize int    `json:"rcvBufSize"`
	RateLogging int    `json:"rateLogging"`
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (sck *Socket) UnmarshalJSON(data []byte) error {
	var raw struct {
		Type        string `json:"type"`
		Method      string `json:"method"`
		Address     string `json:"address"`
		SendBufSize string `json:"sndBufSize"`
		RecvBufSize string `json:"rcvBufSize"`
		RateLogging string `json:"rateLogging"`
	}

	err := json.Unmarshal(data, &raw)
	if err != nil {
		return err
	}

	sck.Type = raw.Type
	sck.Method = raw.Method
	sck.Address = raw.Address
	strFunc := func(v *string, def string) {
		if *v == "" {
			*v = def
		}
	}
	strFunc(&raw.SendBufSize, "1000")
	sck.SendBufSize, err = strconv.Atoi(raw.SendBufSize)
	if err != nil {
		return err
	}
	strFunc(&raw.RecvBufSize, "1000")
	sck.RecvBufSize, err = strconv.Atoi(raw.RecvBufSize)
	if err != nil {
		return err
	}
	strFunc(&raw.RateLogging, "0")
	sck.RateLogging, err = strconv.Atoi(raw.RateLogging)
	if err != nil {
		return err
	}
	return nil
}
