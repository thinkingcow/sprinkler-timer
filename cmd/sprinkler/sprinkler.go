// Control a sprinkler program (version 1)
// Theory of operation:
//   A "program" is a set of sprinkler zones that run in a sequence, each for a specified duration.
//   This cli defines and runs a "program"
//   One or more instances are intended to be started via cron.
//   Only one relay (sprinkler zone) should be activated at a time.
package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/thinkingcow/sprinkler-timer/i2clib"
)

//  Wait a bit between sprinkler zone activations
var offTime time.Duration = time.Second * 3

// An event is a sprinker zone number and watering duration
type event struct {
	id  int           // sprinker number, 0 for none
	dur time.Duration // time duration
}

// Implement the flag.Value interface

func (e *event) String() string {
	return fmt.Sprintf("%d:%s", e.id, e.dur.String())
}

func (e *event) Set(s string) error {
	parts := strings.Split(s, ":")
	if len(parts) != 2 {
		return fmt.Errorf("Invalid event %q: expecting n:t", s)
	}
	dur, err := time.ParseDuration(parts[1])
	if err != nil {
		return fmt.Errorf("Invalid duration %q: %w", parts[1], err)
	}
	if dur < offTime {
		return fmt.Errorf("Invalid duration %s: Must be at least %v", dur, offTime)
	}
	id, err := strconv.Atoi(parts[0])
	if err != nil || id < 0 {
		return fmt.Errorf("Invalid id %q: %w", parts[0], err)
	}
	e.dur = dur
	e.id = id
	return nil
}

// Run activates the sprinkler, scaling the run time by pct.
func (e *event) run(r *i2clib.Relay, pct int) error {
	var mask int
	if e.id > 0 {
		mask = 1 << (e.id - 1)
	}
	if err := r.Set(mask); err != nil {
		return err
	}
	dur := scale(e.dur, pct)
	fmt.Fprintf(os.Stderr, "set %d for %s\n", e.id, dur)
	time.Sleep(dur)
	fmt.Fprintf(os.Stderr, "set %d off\n", e.id)
	r.Set(0)
	time.Sleep(offTime)
	return nil
}

// A program is a sequence of timed sprinkler zone activations.
type program []event

func (p *program) String() string {
	var s = make([]string, len(*p))
	for i, e := range *p {
		s[i] = e.String()
	}
	return strings.Join(s, ",")
}

func (p *program) Set(s string) error {
	for _, v := range strings.Split(s, ",") {
		var e event
		if err := e.Set(v); err != nil {
			return err
		}
		*p = append(*p, e)
	}
	return nil
}

// duration computes the total run time of a program sequence.
func (p *program) duration(pct int) time.Duration {
	var total time.Duration
	for _, e := range *p {
		total += scale(e.dur, pct)
	}
	return total + time.Duration(int64(len(*p))*int64(offTime))
}

// run the sequence.
func (p *program) run(r *i2clib.Relay, pct int) error {
	for _, e := range *p {
		if err := e.run(r, pct); err != nil {
			return err
		}
	}
	return nil
}

// scale is a convenience function to scale a duration
func scale(dur time.Duration, percent int) time.Duration {
	scaled := time.Duration(int64(dur) * int64(percent) / 100)
	if scaled < offTime {
		return offTime
	}
	return scaled
}

// Ensure all sprinklers are off if the program is terminated.
// signal USR1 can be used to query the existing sprinkler state.
func cleanup(r *i2clib.Relay) {
	c := make(chan os.Signal, 1)
	signal.Notify(c)
	go func() {
		for {
			s := <-c
			if s == syscall.SIGURG {
				continue // Go uses this internally
			}
			fmt.Fprintf(os.Stderr, "\nGot signal %s\n", s)
			if s == syscall.SIGUSR1 {
				i, _ := r.Get()
				fmt.Fprintf(os.Stderr, "state=0x%x\n", i)
				continue
			}
			r.Set(0)
			r.Close()
			os.Exit(0)
		}
	}()
}

func main() {
	var board int
	var bus int
	var scale int
	var isDuration bool
	var prog program
	flag.Var(&prog, "program", "comma-separated list of zone_number:duration")
	flag.IntVar(&board, "board", 1, "relay board number (1-8)")
	flag.IntVar(&bus, "i2c-bus", 1, "i2c bus number")
	flag.IntVar(&scale, "scale", 100, "scale all times by this value (use 0 for testing)")
	flag.BoolVar(&isDuration, "total-time", false, "Compute total duration of entire program")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Use 'kill -USR1 $pid' to see currently active zone, if any\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if len(prog) < 1 {
		fmt.Fprintln(os.Stderr, "No program specified")
		flag.Usage()
		return
	}
	if isDuration {
		fmt.Println(prog.duration(scale).String())
		return
	}
	r, err := i2clib.NewRelay(bus, board)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't find board %d on bus %d: %s\n", board, bus, err)
		return
	}
	i, err := r.Get()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't talk to relays: %s\n", err)
		return
	}
	if i != 0 {
		fmt.Fprintf(os.Stderr, "Relays already in use! (mask=0x%02x)\n", i)
		return
	}
	cleanup(r)
	defer r.Close()
	if err := prog.run(r, scale); err != nil {
		fmt.Fprintf(os.Stderr, "Failed: %s\n", err)
	}
}
