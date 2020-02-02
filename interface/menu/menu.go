package menu

import (
	"image/color"

	"github.com/usedbytes/linux-led"

	"github.com/usedbytes/thunk-bot/base"
	"github.com/usedbytes/thunk-bot/interface/input"
)

type Direction int
const (
	None Direction = -1

	North Direction = iota
	East
	South
	West
)

type Item struct {
	color color.Color
	pick func()
}

type Menu struct {
	platform *base.Platform
	items map[Direction]Item

	dir Direction
}

func NewMenu(p *base.Platform) *Menu {
	return &Menu{
		platform: p,
		items: make(map[Direction]Item),
	}
}

func (m *Menu) AddItem(dir Direction, c color.Color, pick func()) {
	m.items[dir] = Item{ color: c, pick: pick }
}

func (m *Menu) Tick(buttons input.ButtonState) {
	dir := None
	if buttons[input.Up] == input.Held || buttons[input.Up] == input.LongPress {
		dir = North
	} else if buttons[input.Right] == input.Held || buttons[input.Right] == input.LongPress  {
		dir = East
	} else if buttons[input.Down] == input.Held || buttons[input.Down] == input.LongPress  {
		dir = South
	} else if buttons[input.Left] == input.Held || buttons[input.Left] == input.LongPress  {
		dir = West
	}

	switch dir {
	case None:
		if m.dir == None {
			return
		}
		m.dir = dir
		m.platform.ResetLEDColor()
		m.platform.SetLEDTrigger(led.TriggerHeartbeat)
	case North, East, South, West:
		item, ok := m.items[dir]
		if !ok {
			return
		}

		if m.dir == None {
			m.platform.SetLEDTrigger(led.TriggerNone)
		}

		if m.dir != dir {
			m.platform.SetLEDColor(item.color)

			m.dir = dir
		}

		if buttons[input.Cross] == input.Pressed {
			m.platform.SetLEDTrigger(led.TriggerHeartbeat)
			m.dir = None
			item.pick()
		}
	}

	buttons[input.Up] = input.None
	buttons[input.Right] = input.None
	buttons[input.Down] = input.None
	buttons[input.Right] = input.None
	buttons[input.Cross] = input.None
}
