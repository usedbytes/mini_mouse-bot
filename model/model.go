// Copyright 2018 Brian Starkey <stark3y@gmail.com>
package model

import (
	"math"

	"github.com/usedbytes/mini_mouse/bot/base"
)

type Coord struct {
	X, Y float32
}

func (c Coord) Sub(b Coord) Coord {
	return Coord{ c.X - b.X, c.Y - b.Y }
}

func (c Coord) Add(b Coord) Coord {
	return Coord{ c.X + b.X, c.Y + b.Y }
}

type Model struct {
	platform *base.Platform

	pos Coord
	ori float32

	prevDist Coord
}

func (m *Model) GetPose() ( Coord, float32 ) {
	return m.pos, m.ori
}

func (c Coord) IsNaN() bool {
	return math.IsNaN(float64(c.X)) || math.IsNaN(float64(c.Y))
}

func sin32(x float32) float32 {
	return float32(math.Sin(float64(x)))
}

func cos32(x float32) float32 {
	return float32(math.Cos(float64(x)))
}

func (m *Model) ResetOrientation() {
	m.pos = Coord{ 0.0, 0.0 }
	m.ori = 0.0
}

func (m *Model) Tick() {
	/*
	wb := m.platform.Wheelbase()

	a, b := m.platform.GetDistance()
	newDist := Coord{ a, b }
	delta := newDist.Sub(m.prevDist)

	w := (delta.Y - delta.X) / (wb)
	r := (wb / 2) * (delta.X + delta.Y) / (delta.Y - delta.X)

	if w == 0.0 {
		m.pos = m.pos.Add(Coord{delta.X * cos32(m.ori), delta.Y * sin32(m.ori) })
	} else {
		rotCentre := Coord{m.pos.X - r * sin32(m.ori), m.pos.Y + r * cos32(m.ori)}
		if rotCentre.IsNaN() {
			log.Printf("rotCentre.IsNaN() - w: %v, rotCentre: %v, r: %v\n", w, rotCentre, r)
			return
		}
		tmp := m.pos.Sub(rotCentre)
		tmp.X = tmp.X * cos32(w) - tmp.Y * sin32(w)
		tmp.Y = tmp.X * sin32(w) + tmp.Y * cos32(w)
		if tmp.IsNaN() {
			log.Printf("tmp.IsNaN() - w: %v, rotCentre: %v, r: %v\n", w, rotCentre, r)
			return
		}
		m.pos = tmp.Add(rotCentre)
	}

	m.ori += w
	if m.ori > math.Pi || m.ori < -math.Pi {
		m.ori = float32(math.Atan2(math.Sin(float64(m.ori)), math.Cos(float64(m.ori))))
	}
	m.prevDist = newDist
	*/

	m.ori = m.platform.GetRot()
}

func NewModel(p *base.Platform) *Model {
	m := &Model{
		platform: p,
	}

	m.ResetOrientation()

	return m
}
