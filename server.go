package main

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/google/uuid"
)

// JobRequest represents the job submission request
// Type is from a list of types of jobs found in const.go
// Args are the additional arguments to the job
type JobRequest struct {
	Type string                 `json:"Type"`
	Args map[string]interface{} `json:"args"`
}

var jobs map[uuid.UUID]JobInterface

func marshalError(err error, jobID string) []byte {
	errMap := make(map[string]string)
	errMap["jobID"] = jobID
	errMap["error"] = err.Error()
	buf, err := json.Marshal(errMap)
	return buf
}

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
		w.Write([]byte(marshalError(errors.New("Invalid Job request format"), "")))
		return
	}

	var newJob JobInterface
	switch jobRequest.Type {
	case Simple:
		newJob = &Job{
			status:  Submitted,
			jobID:   newJobID,
			sigChan: make(chan Signal),
		}
	case Export:
		fromDate, err := time.Parse(timeLayout, jobRequest.Args["from_date"].(string))
		if err != nil {
			w.Write([]byte(marshalError(errors.New("Invalid from_date format"), "")))
			return
		}
		toDate, err := time.Parse(timeLayout, jobRequest.Args["to_date"].(string))
		if err != nil {
			w.Write([]byte(marshalError(errors.New("Invalid to_date format"), "")))
			return
		}
		newJob = &ExportJob{
			status:   Submitted,
			jobID:    newJobID,
			sigChan:  make(chan Signal),
			fromDate: fromDate,
			toDate:   toDate,
			curDate:  fromDate.Add(time.Hour * 24),
		}
	default:
		log.Println("Invalid Job Type")
		w.Write([]byte(marshalError(errors.New("Invalid Job Type"), "")))
		return
	}

	jobs[newJobID] = newJob
	if err = newJob.start(); err != nil {
		log.Printf("Failed to start the job: %s\nError: %s", newJobID.String(), err.Error())
		w.Write([]byte(marshalError(err, "")))
		delete(jobs, newJobID)
		return
	}

	res, err := json.Marshal(map[string]string{
		"result": "Success",
		"jobID":  newJobID.String(),
	})
	w.Write(res)
}

func haltJob(w http.ResponseWriter, r *http.Request) {
	jobID := chi.URLParam(r, "jobID")
	jobUUID, err := uuid.Parse(jobID)
	if err != nil {
		log.Println("Error while parsing UUID from string: ", jobID)
		w.Write([]byte(marshalError(errors.New("Invalid JobID"), jobID)))
		return
	}
	job, ok := jobs[jobUUID]
	if !ok {
		w.Write([]byte(marshalError(errors.New("Invalid JobID"), jobID)))
		return
	}

	if err = job.halt(); err != nil {
		log.Printf("Failed to stop the job: %s\nError: %s", jobID, err.Error())
		w.Write([]byte(marshalError(err, jobID)))
		return
	}
	res, err := json.Marshal(map[string]string{
		"result": "Success",
		"jobID":  jobID,
	})
	log.Println("Halted job:", jobID)
	w.Write([]byte(res))
}

func stopJob(w http.ResponseWriter, r *http.Request) {
	jobID := chi.URLParam(r, "jobID")
	jobUUID, err := uuid.Parse(jobID)
	if err != nil {
		log.Println("Error while parsing UUID from string: ", jobID)
		w.Write([]byte(marshalError(errors.New("Invalid JobID"), jobID)))
		return
	}
	job, ok := jobs[jobUUID]
	if !ok {
		w.Write([]byte(marshalError(errors.New("Invalid JobID"), jobID)))
		return
	}
	if err = job.stop(); err != nil {
		log.Printf("Failed to stop the job: %s\nError: %s\n", jobID, err.Error())
		w.Write([]byte(marshalError(err, jobID)))
		return
	}
	job.clean()
	delete(jobs, jobUUID)
	res, err := json.Marshal(map[string]string{
		"result": "Success",
		"jobID":  jobID,
	})
	log.Println("Stopped job: ", jobID)
	w.Write([]byte(res))
}

func resumeJob(w http.ResponseWriter, r *http.Request) {
	jobID := chi.URLParam(r, "jobID")
	jobUUID, err := uuid.Parse(jobID)
	if err != nil {
		log.Println("Error while parsing UUID from string: ", jobID)
		w.Write([]byte(marshalError(errors.New("Invalid JobID"), jobID)))
		return
	}
	job, ok := jobs[jobUUID]
	if !ok {
		w.Write([]byte(marshalError(errors.New("Invalid JobID"), jobID)))
		return
	}
	if err = job.resume(); err != nil {
		log.Printf("Failed to stop the job: %s\nError: %s\n", jobID, err.Error())
		w.Write([]byte(marshalError(err, jobID)))
		return
	}
	res, err := json.Marshal(map[string]string{
		"result": "Success",
		"jobID":  jobID,
	})
	log.Println("Resumed Job:", jobID)
	w.Write([]byte(res))
}

func detailsJob(w http.ResponseWriter, r *http.Request) {
	jobID := chi.URLParam(r, "jobID")
	jobUUID, err := uuid.Parse(jobID)
	if err != nil {
		log.Println("Error while parsing UUID from string: ", jobID)
		w.Write([]byte(marshalError(errors.New("Invalid JobID"), jobID)))
		return
	}
	job, ok := jobs[jobUUID]
	if !ok {
		w.Write([]byte(marshalError(errors.New("Invalid JobID"), jobID)))
		return
	}
	details := job.details()
	res, err := json.Marshal(details)
	if err != nil {
		log.Println("Couldn't marshal job details")
		w.Write([]byte(marshalError(errors.New("Failed to fetch the details"), jobID)))
		return
	}
	w.Write([]byte(res))
}

func main() {
	// Setup jobs queue
	jobs = make(map[uuid.UUID]JobInterface)
	r := initRouter()
	http.ListenAndServe(":8080", r)
}

func initRouter() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Post("/submit", submitJob)
	r.Get("/halt/{jobID}", haltJob)
	r.Get("/stop/{jobID}", stopJob)
	r.Get("/resume/{jobID}", resumeJob)
	r.Get("/details/{jobID}", detailsJob)
	return r
}
