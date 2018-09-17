// Copyright 2018 Brian Starkey <stark3y@gmail.com>
package dev

import (
	"fmt"

	"github.com/usedbytes/bot_matrix/datalink"
)

type Component struct {
	d *Dev
	ep uint8
}

type Receiver func(*datalink.Packet) interface{}

type Dev struct {
	transactor datalink.Transactor
	cmps map[uint8]Receiver
	toSend []datalink.Packet
	minNum int
	allocNum int
}

func (d *Dev) receive(p *datalink.Packet) interface{} {
	if p.Endpoint == 0 {
		return nil
	}

	r, ok := d.cmps[p.Endpoint]
	if !ok {
		return fmt.Errorf("Received unknown datalink Packet (EP %d)", p.Endpoint)
	}

	return r(p)
}

func (d *Dev) Add(ep uint8, r Receiver) (Component, error) {
	_, ok := d.cmps[ep]
	if ok {
		return Component{}, fmt.Errorf("Duplicate endpoint '%u'", ep)
	}

	d.cmps[ep] = r

	return Component{d, ep}, nil
}

func (d *Dev) remove(ep uint8) error {
	_, ok := d.cmps[ep]
	if !ok {
		return fmt.Errorf("No endpoint '%u'", ep)
	}

	delete(d.cmps, ep)

	return nil
}

func (c Component) Remove() error {
	return c.d.remove(c.ep)
}

func (d *Dev) Queue(p *datalink.Packet) {
	d.toSend = append(d.toSend, *p)
}

func (d *Dev) Poll() ([]interface{}, error) {
	// XXX: try and adjust minNum by heuristics to get the necessary
	// throughput based on actual utilisation
	toSend := d.toSend
	if len(toSend) > 0 {
		d.toSend = make([]datalink.Packet, 0, d.allocNum)

		if len(toSend) < d.minNum {
			toSend = append(toSend, make([]datalink.Packet, d.minNum - len(toSend))...)
		}
	}

	pkts, err := d.transactor.Transact(toSend)
	if err != nil {
		return nil, err
	}

	ret := make([]interface{}, 0, len(pkts))
	for _, p := range pkts {
		ret = append(ret, d.receive(&p))
	}

	return ret, nil
}

func NewDev(transactor datalink.Transactor) *Dev {
	minNum := 0
	allocNum := 4
	dev := &Dev{
		transactor: transactor,
		cmps: make(map[uint8]Receiver),
		toSend: make([]datalink.Packet, allocNum),
		minNum: minNum,
		allocNum: allocNum,
	}

	return dev
}
