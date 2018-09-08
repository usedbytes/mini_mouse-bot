// Copyright 2018 Brian Starkey <stark3y@gmail.com>
package main

import (
	"bytes"
	"encoding/binary"
	"log"
	"net"
	"time"

	"github.com/usedbytes/mini_mouse/bot/interface/input"
)


func sendSpeed(c net.Conn, channel byte, speed int8) {

	buf := new(bytes.Buffer)

	binary.Write(buf, binary.LittleEndian, uint32(1));
	binary.Write(buf, binary.LittleEndian, uint32(2));
	binary.Write(buf, binary.LittleEndian, channel);
	binary.Write(buf, binary.LittleEndian, speed);

	_, err := c.Write(buf.Bytes())
	if err != nil {
		println(err.Error())
	}
}

func main() {
	log.Println("Mini Mouse")

	input := input.NewCollector()

	c,err := net.Dial("unix", "/tmp/sock")
	if err != nil {
		panic(err.Error())
	}

	tick := time.NewTicker(16 * time.Millisecond)

	aspeed := int8(0)
	bspeed := int8(0)

	for _ = range tick.C {

		a, b := input.GetSticks()

		newa := -int8(a * float32(26.0))
		if (aspeed != newa) {
			sendSpeed(c, 0, newa)
			aspeed = newa
		}

		newb := int8(b * float32(26.0))
		if (bspeed != newb) {
			sendSpeed(c, 1, newb)
			bspeed = newb
		}
	}
}
