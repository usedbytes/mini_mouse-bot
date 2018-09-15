// Copyright 2018 Brian Starkey <stark3y@gmail.com>
package motor

import (
	"bytes"
	"encoding/binary"
	"math"

	"github.com/usedbytes/bot_matrix/datalink"
	"github.com/usedbytes/mini_mouse/bot/base/dev"
)

type motor struct {
	alpha float32
}

type Motors struct {
	dev *dev.Dev
	maxRPS float32

	aRPS, bRPS float32
	aRevs, bRevs float32

	motors []motor
}

type StepReport struct {
	Id uint32
	Steps int32
}

func rxStepReport(p *datalink.Packet) interface{} {
	if p.Endpoint != 0x12 {
		return nil
	}

	rep := &StepReport{}
	buf := bytes.NewBuffer(p.Data)
	binary.Read(buf, binary.LittleEndian, &rep.Id)
	binary.Read(buf, binary.LittleEndian, &rep.Steps)

	return rep
}

func (m *Motors) setRadss(id int32, speed int32) {
	p := datalink.Packet{ Endpoint: 1 }

	buf := &bytes.Buffer{}
	binary.Write(buf, binary.LittleEndian, byte(id))
	binary.Write(buf, binary.LittleEndian, byte(speed))
	p.Data = buf.Bytes()

	m.dev.Queue(&p)
}

func (m *Motors) SetRPS(a, b float32) {
	pa := float64(m.rpsToRadss(a))
	pb := float64(m.rpsToRadss(b))

	m.setRadss(0, int32(math.Round(-pa)))
	m.setRadss(1, int32(math.Round(pb)))
}

func (m *Motors) GetRPS() (float32, float32) {
	return m.aRPS, m.bRPS
}

func (m *Motors) GetMaxRPS() float32 {
	return m.maxRPS
}

func (m *Motors) GetRevolutions() (float32, float32) {
	return m.aRevs, m.bRevs
}

func (m *motor) stepsToRevs(steps int32) float32 {
	return float32(steps) * float32(m.alpha) / (2 * math.Pi)
}

func (m *motor) stepsToRps(steps int32) float32 {
	revs := m.stepsToRevs(steps)
	// TODO: Need to not hardcode this
	rps := revs / 0.016
	return rps
}

func (m *Motors) AddSteps(steps *StepReport) {
	if (steps.Id == 0) {
		m.aRevs -= m.motors[0].stepsToRevs(steps.Steps)
		m.aRPS = -m.motors[0].stepsToRps(steps.Steps)
	} else if (steps.Id == 1) {
		m.bRevs += m.motors[1].stepsToRevs(steps.Steps)
		m.bRPS = m.motors[1].stepsToRps(steps.Steps)
	}
}

func (m *Motors) rpsToRadss(rps float32) float32 {
	if rps == 0 {
		return 0
	}
	radss := float32(rps * math.Pi * 2)
	return radss
}

func (m *Motors) Receive(pkt *datalink.Packet) interface{} {
	return rxStepReport(pkt)
}

func NewMotors(dev *dev.Dev) *Motors {
	m := &Motors{
		dev: dev,
		maxRPS: 4.13,

		motors: []motor {
			{ alpha: 2 * math.Pi / 600 },
			{ alpha: 2 * math.Pi / 600 },
		},
	}

	dev.Add(0x12, m.Receive)

	return m
}
