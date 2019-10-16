package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/google/uuid"
)

type Status int

const (
	Running Status = iota
	Halted
	Stopped
)

type Signal int

const (
	Halt Signal = iota
	Stop
	Resume
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

type JobRequest struct {
	Type string
}

func (job *Job) run() {
	for {
		select {
		case sig := <-job.sigChan:
			fmt.Println("GOT A SIG: ", sig)
			switch sig {
			case Halt:
				job.status = Halted
			case Stop:
				job.status = Stopped
			}
		default:
			switch job.status {
			case Running:
				fmt.Println("Doing Job")
				time.Sleep(time.Second)
			case Halted:
				fmt.Println("Job is halted")
			case Stopped:
				fmt.Println("Quiting job: ", job.jobID)
				return
			}
		}
	}
}

func (job *Job) halt() error {
	fmt.Println("Inside halt")
	job.sigChan <- Halt
	fmt.Println("Done with halt")
	return nil
}

func (job *Job) stop() error {
	job.sigChan <- Stop
	job.clean()
	return nil
}

func (job *Job) resume() error {
	job.status = Running
	return nil
}

func (job *Job) clean() error {
	return nil
}

var jobs map[uuid.UUID]JobInterface

func parseJobRequest(body io.ReadCloser) JobRequest {
	decoder := json.NewDecoder(body)
	fmt.Println("Body: ", body)
	var jobRequest JobRequest
	err := decoder.Decode(&jobRequest)
	if err != nil {
		fmt.Println("Couldn't parse the job request")
	}
	return jobRequest
}

func submitJob(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Getting a new job!")
	newJobID := uuid.New()
	jobRequest := parseJobRequest(r.Body)
	var newJob JobInterface
	switch jobRequest.Type {
	case "Simple":
		newJob = &Job{
			status:  Running,
			jobID:   newJobID,
			sigChan: make(chan Signal),
		}
	}
	jobs[newJobID] = newJob
	res, err := json.Marshal(newJobID)
	if err != nil {
		fmt.Println("Couldn't marshal JobID")
	}
	fmt.Println(newJobID)
	go newJob.run()
	w.Write(res)
}

func haltJob(w http.ResponseWriter, r *http.Request) {
	jobID := chi.URLParam(r, "jobID")
	fmt.Println("Halting job: ", jobID)
	jobUUID, err := uuid.Parse(jobID)
	if err != nil {
		fmt.Println("Error while parsing UUID string")
	}
	job, ok := jobs[jobUUID]
	if !ok {
		w.Write([]byte("Invalid JobID: " + jobID))
		return
	}
	job.halt()
	w.Write([]byte("Halted job: " + jobID))
}

func stopJob(w http.ResponseWriter, r *http.Request) {
	jobID := chi.URLParam(r, "jobID")
	fmt.Println("Stopping job: ", jobID)
	jobUUID, err := uuid.Parse(jobID)
	if err != nil {
		fmt.Println("Error while parsing UUID string")
	}
	job, ok := jobs[jobUUID]
	if !ok {
		w.Write([]byte("Invalid JobID: " + jobID))
		return
	}
	job.stop()
	delete(jobs, jobUUID)
	w.Write([]byte("Stopped job: " + jobID))
}

func resumeJob(w http.ResponseWriter, r *http.Request) {
	jobID := chi.URLParam(r, "jobID")
	fmt.Println("Resuming job: ", jobID)
	jobUUID, err := uuid.Parse(jobID)
	if err != nil {
		fmt.Println("Error while parsing UUID string")
	}
	job, ok := jobs[jobUUID]
	if !ok {
		w.Write([]byte("Invalid JobID: " + jobID))
		return
	}
	job.resume()
	w.Write([]byte("Resumed job: " + jobID))
}

func main() {
	jobs = make(map[uuid.UUID]JobInterface)
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Post("/submit", submitJob)
	r.Get("/halt/{jobID}", haltJob)
	r.Get("/stop/{jobID}", stopJob)
	r.Get("/resume/{jobID}", resumeJob)
	http.ListenAndServe(":3333", r)
}
