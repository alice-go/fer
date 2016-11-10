// Copyright 2016 The fer Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package config

import (
	"bytes"
	"encoding/json"
	"testing"
)

var data = map[string][]byte{
	"examples/MQ/1-sampler-sink/ex1-sampler-sink.json": []byte(`{
    "fairMQOptions":
    {
        "devices":
        [{
            "id": "sampler1",
            "channels":
            [{
                "name": "data1",
                "sockets":
                [{
                    "type": "push",
                    "method": "bind",
                    "address": "tcp://*:5555",
                    "sndBufSize": 1000,
                    "rcvBufSize": 1000,
                    "rateLogging": 0
                }]
            }]
        },
        {
            "key": "processor",
            "channels":
            [{
                "name": "data1",
                "sockets":
                [{
                    "type": "pull",
                    "method": "connect",
                    "address": "tcp://localhost:5555",
                    "sndBufSize": 1000,
                    "rcvBufSize": 1000,
                    "rateLogging": 0
                }]
            },
            {
                "name": "data2",
                "sockets":
                [{
                    "type": "push",
                    "method": "connect",
                    "address": "tcp://localhost:5556",
                    "sndBufSize": 1000,
                    "rcvBufSize": 1000,
                    "rateLogging": 0
                }]
            }]
        },
        {
            "id": "sink1",
            "channels":
            [{
                "name": "data2",
                "sockets":
                [{
                    "type": "pull",
                    "method": "bind",
                    "address": "tcp://*:5556",
                    "sndBufSize": 1000,
                    "rcvBufSize": 1000,
                    "rateLogging": 0
                }]
            }]
        }]
    }
}
`),
	"examples/MQ/6-multiple-channels/ex6-multiple-channels.json": []byte(`{
    "fairMQOptions":
    {
        "devices":
        [{
            "id": "sampler1",
            "channels":
            [{
                "name": "data",
                "sockets":
                [{
                    "type": "push",
                    "method": "bind",
                    "address": "tcp://*:5555",
                    "sndBufSize": 1000,
                    "rcvBufSize": 1000,
                    "rateLogging": 0
                }]
            },
            {
                "name": "broadcast",
                "sockets":
                [{
                    "type": "sub",
                    "method": "connect",
                    "address": "tcp://localhost:5005",
                    "sndBufSize": 1000,
                    "rcvBufSize": 1000,
                    "rateLogging": 0
                }]
            }]
        },
        {
            "id": "sink1",
            "channels":
            [{
                "name": "data",
                "sockets":
                [{
                    "type": "pull",
                    "method": "connect",
                    "address": "tcp://localhost:5555",
                    "sndBufSize": 1000,
                    "rcvBufSize": 1000,
                    "rateLogging": 0
                }]
            },
            {
                "name": "broadcast",
                "sockets":
                [{
                    "type": "sub",
                    "method": "connect",
                    "address": "tcp://localhost:5005",
                    "sndBufSize": 1000,
                    "rcvBufSize": 1000,
                    "rateLogging": 0
                }]
            }]
        },
        {
            "id": "broadcaster1",
            "channels":
            [{
                "name": "broadcast",
                "sockets":
                [{
                    "type": "pub",
                    "method": "bind",
                    "address": "tcp://*:5005",
                    "sndBufSize": 1000,
                    "rcvBufSize": 1000,
                    "rateLogging": 0
                }]
            }]
        }]
    }
}`),
	"multi-sockets-with-defaults.json": []byte(`{
    "fairMQOptions":
    {
        "devices":
        [{
            "id": "device1",
            "channels":
            [{
                "name": "data",
                "type": "push",
                "method": "connect",
                "sockets":
                [
                    { "address": "tcp://127.0.0.1:5555" },
                    { "address": "tcp://127.0.0.1:5556" },
                    { "address": "tcp://127.0.0.1:5557" }
                ]
            }]
        }]
    }
}
`),
}

func TestConfigParser(t *testing.T) {
	for n, v := range data {
		r := bytes.NewReader(v)
		var cfg Config
		err := json.NewDecoder(r).Decode(&cfg)
		if err != nil {
			t.Fatalf("error decoding [%s]: %v\n", n, err)
		}
		//fmt.Printf("cfg[%s]=%#v\n", n, cfg)
	}
}
