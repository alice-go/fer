// Copyright 2016 The fer Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package fer provides a basic framework to run FairMQ-like tasks.
package fer

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/sbinet-alice/fer/config"
)

// Main configures and runs a device's execution, managing its state.
func Main(dev Device) error {
	cfg, err := config.Parse()
	if err != nil {
		return err
	}

	return runDevice(context.Background(), cfg, dev, os.Stdin)
}

// Device is a handle to what users get to run via the Fer toolkit.
//
// Devices are configured according to command-line flags and a JSON
// configuration file.
// Clients should implement the Run method to receive and send data via
// the Controler data channels.
type Device interface {
	// Configure hands a device its configuration.
	Configure(cfg config.Device) error
	// Init gives a chance to the device to initialize internal
	// data structures, retrieve channels to input/output data.
	Init(ctl Controler) error
	// Run is where the device's main activity happens.
	// Run should loop forever, until the Controler.Done() channel says
	// otherwise.
	Run(ctl Controler) error
	// Pause pauses the device's execution.
	Pause(ctl Controler) error
	// Reset resets the device's internal state.
	Reset(ctl Controler) error
}

// Controler controls devices execution and gives a device access to input and
// output data channels.
type Controler interface {
	Logger
	Chan(name string, i int) (chan Msg, error)
	Done() chan Cmd

	isControler()
}

// Logger gives access to printf-like facilities
type Logger interface {
	Fatalf(format string, v ...interface{})
	Printf(format string, v ...interface{})
}

// Msg is a quantum of data being exchanged between devices.
type Msg struct {
	Data []byte // Data is the message payload.
	Err  error  // Err indicates whether an error occured.
}

// Cmd describes commands to be sent to a device, via a channel.
type Cmd byte

const (
	// CmdInitDevice is the command sent to initialize a device
	CmdInitDevice Cmd = iota
	// CmdInitTask is the command sent to initialize the tasks of a device
	CmdInitTask
	// CmdRun is the command sent to run a device
	CmdRun
	// CmdPause is the command sent to pause the execution of a device
	CmdPause
	// CmdStop is the command sent to stop the execution of a device
	CmdStop
	// CmdResetTask is the command sent to reset the state of the tasks of a device
	CmdResetTask
	// CmdResetDevice is the command sent to reset the state of a device
	CmdResetDevice
	// CmdEnd is the command sent to end a device
	CmdEnd
	// CmdError is the command sent to notify of an error
	CmdError
)

func (cmd Cmd) String() string {
	switch cmd {
	case CmdInitDevice:
		return "INIT_DEVICE"
	case CmdInitTask:
		return "INIT_TASK"
	case CmdRun:
		return "RUN"
	case CmdPause:
		return "PAUSE"
	case CmdStop:
		return "STOP"
	case CmdResetTask:
		return "RESET_TASK"
	case CmdResetDevice:
		return "RESET_DEVICE"
	case CmdEnd:
		return "END"
	case CmdError:
		return "ERROR_FOUND"
	}
	panic(fmt.Errorf("fer: invalid Cmd value (command=%d)", int(cmd)))
}

func runDevice(ctx context.Context, cfg config.Config, dev Device, r io.Reader) error {
	sys, err := newDevice(ctx, cfg, dev, r)
	if err != nil {
		return err
	}

	if true {
		sys.cmds <- CmdInitDevice
		sys.cmds <- CmdRun
	}

	return sys.run(ctx)
}

func broadcast(cmd Cmd, devs ...*device) {
	for _, dev := range devs {
		dev.cmds <- cmd
	}
}
