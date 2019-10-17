package main

import (
	"log"
	"time"

	"github.com/google/uuid"
)

type JobInterface interface {
	run()
	halt() error
	resume() error
	stop() error
	clean() error
}

type Job struct {
	status  Status
	jobID   uuid.UUID
	sigChan chan Signal
}

func (job *Job) run() {
	for {
		select {
		case sig := <-job.sigChan:
			switch sig {
			case Halt:
				job.status = Halted
			case Stop:
				job.status = Stopped
			}
		default:
			switch job.status {
			case Running:
				log.Println("Doing Job")
				time.Sleep(time.Second)
			case Halted:
			case Stopped:
				return
			}
		}
	}
}

func (job *Job) halt() error {
	job.sigChan <- Halt
	return nil
}

func (job *Job) stop() error {
	job.sigChan <- Stop
	return nil
}

func (job *Job) resume() error {
	job.status = Running
	return nil
}

func (job *Job) clean() error {
	return nil
}
