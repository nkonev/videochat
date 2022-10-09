package main

import (
	"fmt"
	"github.com/montag451/go-eventbus"
	"time"
)

type ProcessStarted struct {
	Pid int
}

func (ProcessStarted) Name() eventbus.EventName {
	return "process.started"
}

type ProcessExited struct {
	Pid      int
	ExitCode int
}

func (ProcessExited) Name() eventbus.EventName {
	return "process.exited"
}

func main() {
	b := eventbus.New()
	b.Subscribe("process.*", func(e eventbus.Event, t time.Time) {
		switch e := e.(type) {
		case ProcessStarted:
			fmt.Printf("[%s] Process %d started\n", t, e.Pid)
		case ProcessExited:
			fmt.Printf("[%s] Process %d exited with code %d\n", t, e.Pid, e.ExitCode)
		}
	})
	b.PublishSync(ProcessStarted{12000})
	time.Sleep(1 * time.Second)
	b.PublishAsync(ProcessExited{12000, 10})
	b.Close()

	time.Sleep(10 * time.Second)
}
