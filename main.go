// Copyright 2018 Brian Starkey <stark3y@gmail.com>
package main

import (
	"log"
	"net"
	"net/rpc"
	"net/http"
	"sync"
	"time"

	"github.com/usedbytes/mini_mouse/bot/interface/input"
	"github.com/usedbytes/mini_mouse/bot/base"
)

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

	input := input.NewCollector()

	telem := Telem{Euler: make([]float64, 3)}

	rpc.Register(&telem)
	rpc.HandleHTTP()
	l, err := net.Listen("tcp", ":1234")
	if err != nil {
		log.Fatal(err)
	}
	go http.Serve(l, nil)

	platform, err := base.NewPlatform()
	if (err != nil) {
		log.Fatalf(err.Error())
	}

	tick := time.NewTicker(16 * time.Millisecond)

	maxSpeed := platform.GetMaxVelocity()

	for _ = range tick.C {
		err = platform.Update()
		if err != nil {
			log.Println(err.Error())
		}

		a, b := input.GetSticks()
		platform.SetVelocity(float64(a) * maxSpeed, float64(b) * maxSpeed)
	}
}
