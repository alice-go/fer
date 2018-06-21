// Copyright 2017 The fer Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build ignore

package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
)

func main() {
	for _, v := range [][2]string{
		{"linux", "amd64"},
		{"linux", "386"},
		{"linux", "arm"},
		{"linux", "arm64"},
		{"darwin", "amd64"},
		{"darwin", "386"},
		{"windows", "amd64"},
		{"windows", "386"},
	} {
		log.Printf("building %s-%s...\n", v[0], v[1])
		cmd := exec.Command(
			"go", "build",
			//"-tags", "netgo",
			"-o", fmt.Sprintf("fer-ex-raw-ctl-%s-%s.exe", v[0], v[1]),
		)
		cmd.Env = []string{"GOOS=" + v[0], "GOARCH=" + v[1]}
		cmd.Env = append(cmd.Env, os.Environ()...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		err := cmd.Run()
		if err != nil {
			log.Printf("build error for GOOS=%s GOARCH=%s: %v\n",
				v[0], v[1],
				err,
			)
		}
	}
}
