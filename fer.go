// Copyright 2016 The fer Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package fer provides a basic framework to run FairMQ-like processes.
// Clients create fer Devices that exchange data via fer message queue sockets.
//
// A client device might look like so:
//
//   import "github.com/alice-go/fer"
//   import "github.com/alice-go/fer/config"
//   type myDevice struct {
//       cfg  config.Device
//       imsg chan fer.Msg
//       omsg chan fer.Msg
//   }
//
// A device needs to implement the fer.Device interface:
//  func (dev *myDevice) Run(ctl fer.Controller)   error { ... }
//
// Optionnally, the following methods may be also implemented:
//  func (dev *myDevice) Configure(cfg config.Device) error { ... }
//  func (dev *myDevice) Init(ctl fer.Controller)  error { ... }
//  func (dev *myDevice) Pause(ctl fer.Controller) error { ... }
//  func (dev *myDevice) Reset(ctl fer.Controller) error { ... }
//
// Typically, the Configure method is used to retrieve the configuration
// associated with the client's device.
// The Init method is used to retrieve the channels of input/output data messages.
// The Run method is an infinite for-loop, selecting on these input/output data
// messages.
// This infinite for-loop will also NEED to listen for the Controller.Done()
// channel to exit that for-loop.
//
// e.g.:
//
//  func (dev *myDevice) Init(ctl fer.Controller) error {
//      imsg, err := ctl.Chan("data-1", 0)
//      omsg, err := ctl.Chan("data-2", 0)
//      dev.imsg = imsg
//      dev.omsg = omsg
//      return nil
//  }
//
//  func (dev *myDevice) Run(ctl fer.Controller) error {
//      for {
//          select {
//          case data := <-dev.imsg:
//              dev.omsg <- bytes.Repeat(data, 2)
//          case <-ctl.Done():
//              return nil
//          }
//      }
//  }
//
// Then, to create an executable:
//
//  package main
//
//  func main() {
//      err := fer.Main(&myDevice{})
//      if err != nil {
//          log.Fatal(err)
//      }
//  }
//
// Build it as usual and run like so:
//
//  $> go build -o my-device
//  $> ./my-device --help
//  Usage of my-device:
//    -control string
//      	starts device in interactive/static mode (default "interactive")
//    -id string
//      	device ID
//    -mq-config string
//      	path to JSON file holding device configuration
//    -transport string
//      	transport mechanism to use (zeromq, nanomsg, go-chan, ...) (default "zeromq")
//  $> ./my-device --id my-id --mq-config ./path/to/config.json
package fer // import "github.com/alice-go/fer"

import (
	"context"
	"io"
	"os"

	"github.com/alice-go/fer/config"
	"golang.org/x/xerrors"
)

// Main configures and runs a device's execution, managing its state.
func Main(dev Device) error {
	cfg, err := config.Parse()
	if err != nil {
		return err
	}

	if cfg.Control == "" {
		cfg.Control = "static"
	}
	return runDevice(context.Background(), cfg, dev, os.Stdin, os.Stdout)
}

// RunDevice runs a device's execution, managing its state.
func RunDevice(ctx context.Context, cfg config.Config, dev Device, r io.Reader, w io.Writer) error {
	return runDevice(ctx, cfg, dev, r, w)
}

// Device is a handle to what users get to run via the Fer toolkit.
//
// Devices are configured according to command-line flags and a JSON
// configuration file.
// Clients need to implement the Run method to receive and send data via
// the Controller data channels.
type Device interface {
	// Run is where the device's main activity happens.
	// Run should loop forever, until the Controller.Done() channel says
	// otherwise.
	Run(ctl Controller) error
}

// DevConfigurer configures a fer device.
type DevConfigurer interface {
	// Configure hands a device its configuration.
	Configure(cfg config.Device) error
}

// DevIniter initializes a fer device.
type DevIniter interface {
	// Init gives a chance to the device to initialize internal
	// data structures, retrieve channels to input/output data.
	Init(ctl Controller) error
}

// DevPauser pauses the execution of a fer device.
type DevPauser interface {
	// Pause pauses the device's execution.
	Pause(ctl Controller) error
}

// DevReseter resets a fer device.
type DevReseter interface {
	// Reset resets the device's internal state.
	Reset(ctl Controller) error
}

// Controller controls devices execution and gives a device access to input and
// output data channels.
type Controller interface {
	Logger
	Chan(name string, i int) (chan Msg, error)
	Done() chan Cmd

	isController()
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
	panic(xerrors.Errorf("fer: invalid Cmd value (command=%d)", int(cmd)))
}

func runDevice(ctx context.Context, cfg config.Config, dev Device, r io.Reader, w io.Writer) error {
	sys, err := newDevice(ctx, cfg, dev, r, w)
	if err != nil {
		return err
	}

	sys.cmds <- CmdInitDevice
	sys.cmds <- CmdRun

	return sys.run(ctx)
}

func broadcast(cmd Cmd, devs ...*device) {
	for _, dev := range devs {
		dev.cmds <- cmd
	}
}
