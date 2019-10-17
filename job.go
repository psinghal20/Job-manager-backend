package main

import (
	"log"
	"time"

	"github.com/google/uuid"
)

// JobInterface is the common interface that every
// different job should implement
type JobInterface interface {
	start() error
	halt() error
	resume() error
	stop() error
	clean() error
	details() map[string]interface{}
}

// Job is a generic simple job
type Job struct {
	status  string
	jobID   uuid.UUID
	sigChan chan Signal
}

// Signal is the representation of different signals
// that can be sent to Job
type Signal int

// Different types of signals sent to Job.run() method
const (
	halt Signal = iota
	stop
)

func (job *Job) run() {
	for {
		select {
		case sig := <-job.sigChan:
			switch sig {
			case halt:
				job.status = Halted
			case stop:
				return
			}
		default:
			switch job.status {
			case Running:
				log.Println("Doing Job")
				time.Sleep(time.Second)
			case Halted:
			}
		}
	}
}

func (job *Job) start() error {
	go job.run()
	return nil
}

func (job *Job) halt() error {
	job.sigChan <- halt
	return nil
}

func (job *Job) stop() error {
	job.sigChan <- stop
	return nil
}

func (job *Job) resume() error {
	job.status = Running
	return nil
}

func (job *Job) clean() error {
	return nil
}

func (job *Job) details() map[string]interface{} {
	details := make(map[string]interface{})
	details["jobID"] = job.jobID
	details["status"] = job.status
	return details
}
