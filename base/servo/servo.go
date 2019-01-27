package servo

import (
	"fmt"

	"periph.io/x/periph/conn"
	"periph.io/x/periph/conn/i2c"
)

type Id int

const (
	ServoA Id = iota
	ServoB

	SERVO_REG_CONTROL uint8 = 0
	SERVO_REG_SERVO_A = 1
	SERVO_REG_SERVO_B = 2
	SERVO_REG_SERVO_A_MIN = 3
	SERVO_REG_SERVO_A_MAX = 4
	SERVO_REG_SERVO_B_MIN = 5
	SERVO_REG_SERVO_B_MAX = 6
)

func NewI2C(b i2c.Bus, addr uint8) (*Dev, error) {
	d := &Dev{d: &i2c.Dev{Bus: b, Addr: uint16(addr)}, name: "Servo", }

	return d, nil
}

func toPos(x float32) byte {
	if x < 0.0 {
		return 0
	} else if x > 1.0 {
		return 255
	}

	return byte(255.0 * x)
}

func (d *Dev) Enable(a, b bool) error {
	val := []byte{0}

	err := d.readReg(0, val)
	if err != nil {
		return err
	}

	val[0] &= ^byte(0x7);
	if a {
		val[0] |= 0x1
	}
	if b {
		val[0] |= 0x2
	}

	if a || b {
		val[0] |= (1 << 2);
	}

	return d.writeReg(0, val)
}

func (d *Dev) SetPos(a, b float32) error {
	return d.writeReg(1, []byte{toPos(a), toPos(b)})
}

func (d *Dev) SetSingle(servo Id, pos float32) error {
	addr := uint8(1)
	if servo == ServoB {
		addr = 2
	}

	return d.writeReg(addr, []byte{toPos(pos)})
}

func (d *Dev) Ping() bool {
	tmp := []byte{0}
	err := d.readReg(0, tmp)
	if err != nil {
		return false
	}

	return true
}

func (d *Dev) Dump(reg, num uint8) []byte {
	data := make([]byte, num)

	d.readReg(reg, data)

	return data
}

type Dev struct {
	d    conn.Conn
	name string
}

func (d *Dev) String() string {
	return fmt.Sprintf("%s{%s}", d.name, d.d)
}

func (d *Dev) Halt() error {
	return d.Enable(false, false)
}

func (d *Dev) readReg(reg uint8, data []byte) error {
	err := d.d.Tx([]byte{reg}, data)
	return err
}

func (d *Dev) writeReg(reg uint8, data []byte) error {
	write := make([]byte, 1, len(data) + 1)
	write[0] = reg
	write = append(write, data...)

	return d.d.Tx(write, nil)
}

var _ conn.Resource = &Dev{}
