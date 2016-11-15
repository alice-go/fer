// Copyright 2016 The fer Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mq_test

import (
	"fmt"
	"net"
	"strconv"
	"sync"
	"testing"

	"github.com/sbinet-alice/fer/mq"
	_ "github.com/sbinet-alice/fer/mq/nanomsg"
	_ "github.com/sbinet-alice/fer/mq/zeromq"
)

func TestOpen(t *testing.T) {
	_, err := mq.Open("no-such-driver")
	if err == nil {
		t.Fatalf("expected a no such-driver error")
	}

	drv1, err := mq.Open("nanomsg")
	if err != nil {
		t.Fatal(err)
	}

	drv2, err := mq.Open("nanomsg")
	if err != nil {
		t.Fatal(err)
	}

	if drv1 != drv2 {
		t.Fatalf("Open is not idem-potent")
	}
}

var drivers = []string{"zeromq", "nanomsg"}

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

func TestPushPull(t *testing.T) {
	for i := range drivers {
		transport := drivers[i]
		t.Run("transport="+transport, func(t *testing.T) {

			const (
				N    = 5
				tmpl = "data-%02d"
			)

			port, err := getTCPPort()
			if err != nil {
				t.Fatalf("error getting free TCP port: %v\n", err)
			}

			drv, err := mq.Open(transport)
			if err != nil {
				t.Fatal(err)
			}
			pull, err := drv.NewSocket(mq.Pull)
			if err != nil {
				t.Fatal(err)
			}
			defer pull.Close()

			push, err := drv.NewSocket(mq.Push)
			if err != nil {
				t.Fatal(err)
			}
			defer push.Close()

			var wg sync.WaitGroup
			wg.Add(1)
			go func() {
				err := push.Dial("tcp://localhost:" + port)
				if err != nil {
					t.Fatal(err)
				}
				for i := 0; i < N; i++ {
					err = push.Send([]byte(fmt.Sprintf(tmpl, i)))
					if err != nil {
						t.Fatalf("error sending data[%d]: %v\n", i, err)
					}
				}
				wg.Done()
			}()

			err = pull.Listen("tcp://*:" + port)
			if err != nil {
				t.Fatal(err)
			}
			for i := 0; i < N; i++ {
				msg, err := pull.Recv()
				if err != nil {
					t.Fatal(err)
				}
				if got, want := string(msg), fmt.Sprintf(tmpl, i); got != want {
					t.Errorf("push-pull[%d]: got=%q want=%q\n", i, got, want)
				}
			}
			wg.Wait()
			err = pull.Close()
			if err != nil {
				t.Fatal(err)
			}
			err = push.Close()
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}
