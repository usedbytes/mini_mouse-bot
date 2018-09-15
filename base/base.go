// Copyright 2018 Brian Starkey <stark3y@gmail.com>
package base

import (
	"math"
	"net"
	"log"

	"github.com/usedbytes/mini_mouse/bot/base/dev"
	"github.com/usedbytes/mini_mouse/bot/base/motor"
	"github.com/usedbytes/bot_matrix/datalink/netconn"
)

type Platform struct {
	dev *dev.Dev
	mmPerRev float32
	wheelbase float32

	Motors *motor.Motors
}

func (p *Platform) SetVelocity(a, b float32) {
	aRps := a / p.mmPerRev
	bRps := b / p.mmPerRev

	p.Motors.SetRPS(aRps, bRps)
}

func (p *Platform) SetOmega(w float32) {

	rps := w * (p.wheelbase / 2) / p.mmPerRev
	log.Printf("SetOmega w: %v, rps: %v\n", w, rps)

	p.Motors.SetRPS(-rps, rps)
}

func (p *Platform) GetMaxVelocity() float32 {
	max := p.Motors.GetMaxRPS()
	return max * p.mmPerRev
}

func (p *Platform) GetVelocity() (float32, float32) {
	a, b := p.Motors.GetRPS()
	return a * p.mmPerRev, b * p.mmPerRev
}

func (p *Platform) GetDistance() (float32, float32) {
	a, b := p.Motors.GetRevolutions()
	return a * p.mmPerRev, b * p.mmPerRev
}

func (p *Platform) Wheelbase() float32 {
	return p.wheelbase
}

func NewPlatform(/* Some config */) (*Platform, error) {
	c, err := net.Dial("unix", "/tmp/sock")
	if err != nil {
		log.Fatal(err)
	}
	t := netconn.NewNetconn(c)
	dev := dev.NewDev(t)

	p := &Platform{
		dev: dev,
		mmPerRev: (30.5 * math.Pi),
		wheelbase: 76,
	}
	p.Motors = motor.NewMotors(dev)

	return p, nil
}

func (p *Platform) Update() error {
	pkts, err := p.dev.Poll()
	if err != nil {
		return err
	}

	for _, pkt := range pkts {
		switch t := pkt.(type) {
		case (*motor.StepReport):
			p.Motors.AddSteps(t)
		default:
			if pkt != nil {
				log.Printf("%v\n", pkt)
			}
		}
	}

	return nil
}
