package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"

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

type JobRequest struct {
	Type string
	Args map[string]interface{}
}

var jobs map[uuid.UUID]JobInterface

func parseJobRequest(body io.ReadCloser) (*JobRequest, error) {
	decoder := json.NewDecoder(body)
	jobRequest := &JobRequest{
		"",
		make(map[string]interface{}),
	}
	err := decoder.Decode(&jobRequest)
	if err != nil {
		log.Println("Couldn't parse the job request")
		return nil, err
	}
	return jobRequest, nil
}

func submitJob(w http.ResponseWriter, r *http.Request) {
	newJobID := uuid.New()

	jobRequest, err := parseJobRequest(r.Body)
	if err != nil {
		log.Println("Couldn't parse the job request")
		w.Write([]byte("Invalid Request Format"))
		return
	}

	var newJob JobInterface
	switch jobRequest.Type {
	case "Simple":
		newJob = &Job{
			status:  Running,
			jobID:   newJobID,
			sigChan: make(chan Signal),
		}
	default:
		log.Println("Invalid Job Type")
		w.Write([]byte("Invalid Job Type"))
		return
	}

	jobs[newJobID] = newJob
	res, err := json.Marshal(newJobID)
	if err != nil {
		log.Println("Couldn't marshal JobID")
		w.Write([]byte("Error: Couldn't submit the job"))
		delete(jobs, newJobID)
		return
	}

	go newJob.run()
	w.Write(res)
}

func haltJob(w http.ResponseWriter, r *http.Request) {
	jobID := chi.URLParam(r, "jobID")
	log.Println("Halting job: ", jobID)
	jobUUID, err := uuid.Parse(jobID)
	if err != nil {
		log.Println("Error while parsing UUID from string: ", jobID)
		w.Write([]byte("Invalid JobID: " + jobID))
		return
	}
	job, ok := jobs[jobUUID]
	if !ok {
		w.Write([]byte("Invalid JobID: " + jobID))
		return
	}
	job.halt()
	res := "Halted job: " + jobID
	log.Println(res)
	w.Write([]byte(res))
}

func stopJob(w http.ResponseWriter, r *http.Request) {
	jobID := chi.URLParam(r, "jobID")
	log.Println("Stopping job: ", jobID)
	jobUUID, err := uuid.Parse(jobID)
	if err != nil {
		log.Println("Error while parsing UUID from string: ", jobID)
		w.Write([]byte("Invalid JobID: " + jobID))
		return
	}
	job, ok := jobs[jobUUID]
	if !ok {
		w.Write([]byte("Invalid JobID: " + jobID))
		return
	}
	job.stop()
	job.clean()
	delete(jobs, jobUUID)
	res := "Stopped job: " + jobID
	log.Println(res)
	w.Write([]byte(res))
}

func resumeJob(w http.ResponseWriter, r *http.Request) {
	jobID := chi.URLParam(r, "jobID")
	log.Println("Resuming job: ", jobID)
	jobUUID, err := uuid.Parse(jobID)
	if err != nil {
		log.Println("Error while parsing UUID from string: ", jobID)
		w.Write([]byte("Invalid JobID: " + jobID))
		return
	}
	job, ok := jobs[jobUUID]
	if !ok {
		w.Write([]byte("Invalid JobID: " + jobID))
		return
	}
	job.resume()
	res := "Resumed job: " + jobID
	log.Println(res)
	w.Write([]byte(res))
}

func main() {
	jobs = make(map[uuid.UUID]JobInterface)
	r := initRouter()
	http.ListenAndServe(":3333", r)
}

func initRouter() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Post("/submit", submitJob)
	r.Get("/halt/{jobID}", haltJob)
	r.Get("/stop/{jobID}", stopJob)
	r.Get("/resume/{jobID}", resumeJob)
	return r
}
