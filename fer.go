// Copyright 2016 The fer Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package fer provides a basic framework to run FairMQ-like tasks.
package fer

import "fmt"

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
