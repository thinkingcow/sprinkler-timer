// Go version of i2c relay driver (reverse engineered)
package i2clib

import (
	"fmt"
	"os"

	"golang.org/x/sys/unix"
)

const (
	baseAddress  = 0x27   // base address of relay board ic slave
	slaveAddress = 0x0703 // from /usr/include/linux/i2c-dev.h
	readAddress  = 0x00   // read from the i2c device
	writeAddress = 0x01	  // write to the i2c device
)

var relay2Addr = []byte{0, 2, 1, 3, 6, 4, 5, 7} // Map relay number to i2c address
var addr2Relay []byte                           // Map i2c address to relay number

// Generate inverse mapping table
func init() {
	addr2Relay = make([]byte, len(relay2Addr))
	for i, v := range relay2Addr {
		addr2Relay[v] = byte(i)
	}
}

// convert relay bit mask to i2c bit mask
func relay2ic(in int) byte {
	var v byte
	for i := 0; i < 8; i++ {
		if (in & (1 << i)) != 0 {
			v |= 1 << relay2Addr[i]
		}
	}
	return v
}

// i2c bit mask to relay bit mask
func ic2relay(in byte) int {
	var v int
	for i := 0; i < 8; i++ {
		if (in & (1 << i)) != 0 {
			v |= 1 << addr2Relay[i]
		}
	}
	return v
}

type Relay struct {
	file  *os.File // file descriptor of I2C device
	board int
}

func NewRelay(bus int, board int) (*Relay, error) {
	if board < 1 || board > 8 {
		return nil, fmt.Errorf("Invalid board number")
	}
	file, err := os.OpenFile(fmt.Sprintf("/dev/i2c-%d", bus), os.O_RDWR, 0600)
	if err != nil {
		return nil, fmt.Errorf("Can't open i2c bus %d: %w", bus, err)
	}
	r := &Relay{file: file}
	if err:=r.Board(board); err != nil {
		return r, err
	}
	return r, nil
}

// boards are numbered 1-8
func (r *Relay) Board(n int) error {
	r.board = n
	if err := unix.IoctlSetInt(int(r.file.Fd()), slaveAddress, baseAddress-((n-1)&7)); err != nil {
		r.board = 0
		return fmt.Errorf("Can't set slave: %x, %x: %w", slaveAddress, baseAddress-((n-1)&7), err)
	}
	return nil
}

func (r *Relay) Set(state int) error {
	buf := []byte{writeAddress, relay2ic(state)}
	n, err := r.file.Write(buf)
	if err != nil {
		return fmt.Errorf("Set error %w", err)
	}
	if n != len(buf) {
		return fmt.Errorf("Write error: %d of %d bytes written", n, len(buf))
	}
	return nil
}

func (r *Relay) Get() (int, error) {
	buf := []byte{readAddress, 0}
	n, err := r.file.Read(buf)
	if err != nil {
		return 0, fmt.Errorf("Read error: %w", err)
	}
	if n != len(buf) {
		return 0, fmt.Errorf("Read error: %d of %d bytes read", n, len(buf))
	}
	return ic2relay(buf[1]), nil
}

func (r *Relay) Close() error {
	return r.file.Close()
}
