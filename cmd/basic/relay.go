// Go version of i2c relay driver (reverse engineered)
package main

import (
	"flag"
	"fmt"
	"github.com/thinkingcow/sprinkler-timer/i2clib"
	"os"
	"strconv"
)

// Basic testing
func main() {
	var board int
	var bus int
	flag.IntVar(&board, "board", 1, "relay board number (1-8)")
	flag.IntVar(&bus, "i2c-bus", 1, "i2c bus number")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] get | set n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  set n (0 <= n <= 0xFF) activate relays\n")
		fmt.Fprintf(os.Stderr, "  get  get currently activated relay(s)\n")
		flag.PrintDefaults()
	}
	flag.Parse()
	argv := flag.Args()

	if len(argv) < 1 {
		flag.Usage()
		return
	}
	r, err := i2clib.NewRelay(bus, board)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't find board %d on bus %d: %s\n", board, bus, err)
		return
	}
	defer r.Close()
	switch argv[0] {
	case "set":
		if len(argv) < 2 {
			fmt.Fprintf(os.Stderr, "set: missing value\n", err)
			return
		}
		i, err := strconv.Atoi(argv[1])
		if err != nil {
			fmt.Fprintf(os.Stderr, "set: invalid integer: %s\n", err)
			return
		}
		if err = r.Set(i); err != nil {
			fmt.Fprintf(os.Stderr, "set error: %s\n", i, err)
			return
		}
	case "get":
		v, err := r.Get()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Can't get value: %s\n", err)
			return
		}
		fmt.Printf("%d\n", v)
	default:
		fmt.Fprintf(os.Stderr, "Invalid command: %q\n", argv[0])
		flag.Usage()
		return
	}
}
