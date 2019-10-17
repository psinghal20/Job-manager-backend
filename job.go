package main

import (
	"errors"
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
				return
			case stop:
				return
			}
		default:
			switch job.status {
			case Running:
				log.Println("Doing Job")
				time.Sleep(time.Second)
			}
		}
	}
}

func (job *Job) start() error {
	switch job.status {
	case Running:
		return errors.New("Failed to start the Job : Job already running")
	case Halted:
		return errors.New("Failed to start the Job : Job is halted. Try to resume the job")
	}
	job.status = Running
	go job.run()
	return nil
}

func (job *Job) halt() error {
	switch job.status {
	case Submitted:
		return errors.New("Failed to halt the Job : Job is not running")
	case Halted:
		return errors.New("Failed to halt the Job : Job is already halted")
	}
	job.status = Halted
	job.sigChan <- halt
	return nil
}

func (job *Job) stop() error {
	if job.status == Submitted {
		return errors.New("Failed to stop the Job : Job not running")
	}
	job.sigChan <- stop
	return nil
}

func (job *Job) resume() error {
	if job.status != Halted {
		return errors.New("Failed to resume the Job : Job not halted")
	}
	job.status = Running
	go job.run()
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

// ExportJob represents a Export data job having fromDate and toDate as arguments
type ExportJob struct {
	status  string
	jobID   uuid.UUID
	sigChan chan Signal

	fromDate time.Time
	toDate   time.Time
	curDate  time.Time
}

func (job *ExportJob) run() {
	for {
		select {
		case sig := <-job.sigChan:
			switch sig {
			case halt:
				job.status = Halted
				return
			case stop:
				return
			}
		default:
			switch job.status {
			case Running:
				if job.curDate.After(job.fromDate) && job.curDate.Before(job.toDate) {
					// Assuming we have access to some database from which we need to
					// export the data. Assuming each export operation to take a second
					log.Println("Exporting data: ", job.curDate.Format(timeLayout))
					job.curDate = job.curDate.Add(time.Hour * 24)
					time.Sleep(time.Second)
				}
			}
		}
	}
}

func (job *ExportJob) start() error {
	switch job.status {
	case Running:
		return errors.New("Failed to start the Job : Job already running")
	case Halted:
		return errors.New("Failed to start the Job : Job is halted. Try to resume the job")
	}
	job.status = Running
	go job.run()
	return nil
}

func (job *ExportJob) halt() error {
	switch job.status {
	case Submitted:
		return errors.New("Failed to halt the Job : Job is not running")
	case Halted:
		return errors.New("Failed to halt the Job : Job is already halted")
	}
	job.status = Halted
	job.sigChan <- halt
	return nil
}

func (job *ExportJob) stop() error {
	switch job.status {
	case Submitted:
		return errors.New("Failed to stop the Job : Job not running")
	case Halted:
		return nil
	}
	job.sigChan <- stop
	return nil
}

func (job *ExportJob) resume() error {
	if job.status != Halted {
		return errors.New("Failed to resume the Job : Job not halted")
	}
	job.status = Running
	go job.run()
	return nil
}

func (job *ExportJob) clean() error {
	return nil
}

func (job *ExportJob) details() map[string]interface{} {
	details := make(map[string]interface{})
	details["jobID"] = job.jobID
	details["status"] = job.status
	details["from_date"] = job.fromDate.Format(timeLayout)
	details["to_date"] = job.toDate.Format(timeLayout)
	return details
}
