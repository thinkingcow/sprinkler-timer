# sprinkler-timer
Sprinkler timer using the SequentMicrosystems stackable 8-relay board(s)
on a Raspberry PI

The relays are controlled by communicating with an I2C device, using the "i2c" Linux kernel driver.

This code is/was reverse engineered by running the 8relay binary from SequentMicro with
strace -e open,read,write,ioctl 8relay ...

Basic library usage:

  `r, err := i2clib.NewRelay(bus, board)`
Open a channel to relay "board" on i2c "bus"
- The usual bus for a Raspberry PI is "1"
- The board, with no address jumper pins installed is "1"
  `r.Set(i)`
Set the relays based on the bit mask i (0 - 0xff)
  `r.Get()`
Get the currently activated relays (e.g. r.Set(i);r.Get() should emit "i")
  `r.Close()`
Close the channel

Build "sample" program:
  `go build cmd/basic/relay.go`
