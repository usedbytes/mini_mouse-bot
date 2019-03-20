// Copyright 2018 Brian Starkey <stark3y@gmail.com>
package music

import (
	"bytes"
	"encoding/binary"
	"log"
	"time"
	"image/color"

	"github.com/BurntSushi/toml"
	"gitlab.com/gomidi/midi/mid"
	"gitlab.com/gomidi/midi/smf"

	"github.com/usedbytes/mini_mouse/bot/base"
	"github.com/usedbytes/mini_mouse/bot/base/dev"
	"github.com/usedbytes/mini_mouse/bot/interface/input"
	"github.com/usedbytes/bot_matrix/datalink"

)

const TaskName = "music"

type Task struct {
	p *base.Platform
	d *dev.Dev
	fileIdx int

	Player *Player
	Files []*File
}

type Channel struct {
	Track int16
	Channel uint8
}

type Note struct {
	Key uint8
	Channel Channel
	Start uint64
	Duration uint64
}

type Track struct {
	Number int
	Start uint64
	End uint64
	Notes []*Note

	current *Note
	prevKey uint8
}

type File struct {
	Tracks []*Track
	TicksToDuration func(uint32) time.Duration
	DurationToTicks func(time.Duration) uint32
	MetricTicks smf.MetricTicks

	ChannelMap map[Channel][]int
}

type Cursor struct {
	Idx int
	Notes []*Note
	NextTs uint64
	Note *Note
}

func (c *Cursor) Advance() bool {
	if c.Idx == len(c.Notes) {
		return false
	}

	note := c.Notes[c.Idx]
	if c.Note == nil {
		c.NextTs = c.Notes[c.Idx].Start
		c.Note = c.Notes[c.Idx]
	} else {
		c.NextTs = c.NextTs + note.Duration
		c.Note = nil
		c.Idx++
	}

	return true
}

type Player struct {
	File *File
	Started time.Time
	Cursors []*Cursor
	Notes []uint8
	Timestamp uint64
	NextTs uint64
	End uint64
	ch chan bool
	dev *dev.Dev
}

func NewPlayer(f *File, d *dev.Dev) *Player {
	p := &Player{
		File: f,
		Cursors: make([]*Cursor, 0, len(f.Tracks)),
		Notes: make([]uint8, len(f.Tracks)),
		ch: make(chan bool, 10),
		dev: d,
	}

	for _, t := range f.Tracks {
		if t.Notes == nil || len(t.Notes) == 0 {
			continue
		}

		if t.End > p.End {
			p.End = t.End
		}

		c := &Cursor {
			Notes: t.Notes,
		}
		c.Advance()
		p.Cursors = append(p.Cursors, c)
	}

	return p
}

func (p *Player) emit(note *Note) {
	outputs, ok := p.File.ChannelMap[note.Channel]
	if !ok {
		return
	}

	for _, out := range outputs {
		buf := new(bytes.Buffer)

		tsUs := p.File.TicksToDuration(uint32(p.Timestamp)).Nanoseconds() / 1000
		durUs := p.File.TicksToDuration(uint32(note.Duration)).Nanoseconds() / 1000
		binary.Write(buf, binary.LittleEndian, uint32(out));
		binary.Write(buf, binary.LittleEndian, uint32(tsUs));
		binary.Write(buf, binary.LittleEndian, uint32(note.Key));
		binary.Write(buf, binary.LittleEndian, uint32(durUs));

		pkt := &datalink.Packet{
			Endpoint: 3,
			Data: buf.Bytes(),
		}

		p.dev.Queue(pkt)
	}
}

func (p *Player) Reset() {
	p.Started = time.Time{}

	buf := new(bytes.Buffer)

	binary.Write(buf, binary.LittleEndian, uint32(0));
	binary.Write(buf, binary.LittleEndian, uint32(1));

	pkt := &datalink.Packet{
		Endpoint: 4,
		Data: buf.Bytes(),
	}

	p.dev.Queue(pkt)
}

func (p *Player) PlayPause(play bool) {
	buf := new(bytes.Buffer)

	if play {
		binary.Write(buf, binary.LittleEndian, uint32(1));
	} else {
		binary.Write(buf, binary.LittleEndian, uint32(0));
	}
	binary.Write(buf, binary.LittleEndian, uint32(0));

	pkt := &datalink.Packet{
		Endpoint: 4,
		Data: buf.Bytes(),
	}

	p.dev.Queue(pkt)
}

func (p *Player) PlayUntil(until time.Time) {
	if p.Started.IsZero() {
		log.Println("Starting")
		p.Started = time.Now()
	}
	abs := until.Sub(p.Started)
	if abs < 0 {
		panic("Underflow")
	}

	end := p.File.DurationToTicks(abs)

	for p.Timestamp < uint64(end) {
		next := p.End
		nextTrack := -1
		for i, c := range p.Cursors {
			if c == nil {
				continue
			}

			if c.NextTs <= next {
				next = c.NextTs
				nextTrack = i
			}
		}
		p.Timestamp = next

		if nextTrack >= 0 {
			note := p.Cursors[nextTrack].Note
			if note != nil {
				p.emit(note)
				p.Notes[nextTrack] = p.Cursors[nextTrack].Note.Key
			}
			finished := !p.Cursors[nextTrack].Advance()
			if finished {
				p.Cursors[nextTrack] = nil
			}
		}
		//time.Sleep(p.File.TicksToDuration(uint32(next - p.Timestamp)))
	}
}

type Tune struct {
	Filename string
	Channels [][][]int
}

func LoadTune(tune *Tune) *File {
	f := &File{
		ChannelMap: make(map[Channel][]int),
	}

	for out, matches := range tune.Channels {
		for _, ch := range matches {
			channel := Channel{ int16(ch[0]), uint8(ch[1]) }
			current := f.ChannelMap[channel]
			current = append(current, out)
			f.ChannelMap[channel] = current
		}
	}

	rd := mid.NewReader()

	rd.SMFHeader = func(hdr smf.Header) {
		f.MetricTicks = hdr.TimeFormat.(smf.MetricTicks)
		f.TicksToDuration = func(ticks uint32) time.Duration {
			return f.MetricTicks.FractionalDuration(120.0, ticks)
		}
		f.DurationToTicks = func(dur time.Duration) uint32 {
			return f.MetricTicks.FractionalTicks(120.0, dur)
		}
		f.Tracks = make([]*Track, hdr.NumTracks)
		for i, _ := range f.Tracks {
			f.Tracks[i] = &Track{
				Number: i,
				Notes: make([]*Note, 0),
			}
		}
	}
	rd.Msg.Meta.TempoBPM = func(p mid.Position, bpm float64) {
		f.TicksToDuration = func(ticks uint32) time.Duration {
			return f.MetricTicks.FractionalDuration(bpm, ticks)
		}
		f.DurationToTicks = func(dur time.Duration) uint32 {
			return f.MetricTicks.FractionalTicks(bpm, dur)
		}
	}
	rd.Msg.Channel.NoteOn = func(p *mid.Position, channel, key, vel uint8) {
		track := f.Tracks[p.Track]
		if len(track.Notes) == 0 {
			track.Start = p.AbsoluteTicks
		}
		if track.current != nil {
			//fmt.Println("Ignoring Polyphonic? Already have current")
			track.current.Duration = p.AbsoluteTicks - track.current.Start
			if track.current.Duration != 0 {
				track.Notes = append(track.Notes, track.current)
			}
			track.current = nil
		}

		track.current = &Note{ Key: key, Channel: Channel{ Track: p.Track, Channel: channel}, Start: p.AbsoluteTicks }
	}
	rd.Msg.Channel.NoteOff = func(p *mid.Position, channel, key, vel uint8) {
		track := f.Tracks[p.Track]
		track.End = p.AbsoluteTicks
		if track.current == nil || key != track.current.Key || channel != track.current.Channel.Channel {
			//fmt.Println("Ignoring Polyphonic? Key doesn't match")
			return
		}

		track.current.Duration = p.AbsoluteTicks - track.current.Start
		track.Notes = append(track.Notes, track.current)
		track.current = nil
	}

	err := rd.ReadSMFFile(tune.Filename)
	if err != nil {
		panic(err.Error())
	}

	return f
}

var cfg struct {
	Files []Tune
}

func (t *Task) Enter() {
	t.p.SetVelocity(0, 0)

	t.Player = NewPlayer(t.Files[t.fileIdx], t.d)
	t.Player.Reset()
	t.Player.PlayPause(true)

	t.fileIdx++
	if t.fileIdx >= len(t.Files) {
		t.fileIdx = 0
	}
}

func (t *Task) Exit() {
	t.Player.PlayPause(false)
	t.p.SetVelocity(0, 0)

	t.p.SetBoost(base.BoostNone)
}

func (t *Task) Tick(buttons input.ButtonState) {
	if t.Player == nil {
		return
	}

	if t.Player.Timestamp >= t.Player.End {
		return
	}

	t.Player.PlayUntil(time.Now().Add(50 * time.Millisecond))
}

func (t *Task) Color() color.Color {
	return color.NRGBA{ 0x00, 0xff, 0xff, 0xff }
}

func NewTask(ip *input.Collector, pl *base.Platform) *Task {
	task := &Task{
		p: pl,
		d: pl.Dev(),
		Files: make([]*File, 0),
	}

	_, err := toml.DecodeFile("/home/pi/audio/midi/tunes.toml", &cfg)
	if err != nil {
		panic(err.Error())
	}

	for _, tune := range cfg.Files {
		task.Files = append(task.Files, LoadTune(&tune))
	}

	return task
}
