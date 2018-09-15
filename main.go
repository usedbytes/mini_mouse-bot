// Copyright 2018 Brian Starkey <stark3y@gmail.com>
package main

import (
	"bytes"
	"encoding/binary"
	"log"
	"net"
	"net/rpc"
	"net/http"
	"sync"
	"time"

	"periph.io/x/periph/host"
	"periph.io/x/periph/conn/i2c/i2creg"
	"github.com/usedbytes/bno055"
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

type Telem struct {
	lock sync.Mutex
	Euler []float64
}

func (t *Telem) SetEuler(vec []float64) {
	t.lock.Lock()
	defer t.lock.Unlock()

	t.Euler = vec
}

func (t *Telem) GetEuler(ignored bool, vec *[]float64) error {
	t.lock.Lock()
	defer t.lock.Unlock()

	*vec = t.Euler

	return nil
}

func main() {
	log.Println("Mini Mouse")

	if _, err := host.Init(); err != nil {
		log.Fatal(err)
	}

	input := input.NewCollector()

	c, err := net.Dial("unix", "/tmp/sock")
	if err != nil {
		panic(err.Error())
	}

	b, err := i2creg.Open("")
	if err != nil {
		log.Fatal(err)
	}
	defer b.Close()

	telem := Telem{Euler: make([]float64, 3)}

	rpc.Register(&telem)
	rpc.HandleHTTP()
	l, err := net.Listen("tcp", ":1234")
	if err != nil {
		log.Fatal(err)
	}
	go http.Serve(l, nil)

	imu, err := bno055.NewI2C(b, 0x29)
	if err != nil {
		log.Fatal(err)
	}

	err = imu.SetUseExternalCrystal(true)
	if err != nil {
		log.Fatal(err)
	}

	tick := time.NewTicker(16 * time.Millisecond)

	aspeed := int8(0)
	bspeed := int8(0)

	for _ = range tick.C {

		vec, err := imu.GetVector(bno055.VECTOR_EULER)
		if err != nil {
			log.Println(err)
		} else {
			telem.SetEuler(vec)
		}

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
