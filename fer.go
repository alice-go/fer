// Copyright 2016 The fer Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package fer provides a basic framework to run FairMQ-like tasks.
package fer

import "fmt"

// Cmd describes commands to be sent to a device, via a channel.
type Cmd byte

const (
	CmdInitDevice Cmd = iota
	CmdInitTask
	CmdRun
	CmdPause
	CmdStop
	CmdResetTask
	CmdResetDevice
	CmdEnd
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
