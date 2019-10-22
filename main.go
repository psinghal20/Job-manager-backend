package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	_ "github.com/psinghal20/atlan-assignment/docs"
	swaggerFiles "github.com/swaggo/files"     // swagger embed files
	ginSwagger "github.com/swaggo/gin-swagger" // gin-swagger middleware
)

// JobRequest represents the job submission request
// Type is from a list of types of jobs found in const.go
// Args are the additional arguments to the job
type JobRequest struct {
	Type string                 `json:"Type"`
	Args map[string]interface{} `json:"args"`
}

type httpResponse struct {
	JobID   uuid.UUID              `json:"jobID" example:"55e75f6c-24f8-49b5-9e62-a268db7370e9"`
	Message string                 `json:"message" example:"Success"`
	Details map[string]interface{} `json:"details"`
}

type httpError struct {
	JobID string `json:"jobID"`
	Error error  `json:"error"`
}

var jobs map[uuid.UUID]JobInterface

func marshalError(err error, jobID string) []byte {
	errMap := make(map[string]string)
	errMap["jobID"] = jobID
	errMap["error"] = err.Error()
	buf, err := json.Marshal(errMap)
	return buf
}

func parseJobRequest(c *gin.Context) (*JobRequest, error) {
	jobRequest := &JobRequest{
		"",
		make(map[string]interface{}),
	}

	err := c.BindJSON(&jobRequest)
	if err != nil {
		return nil, err
	}
	return jobRequest, nil
}

// submitJob godoc
// @Summary Submit a job for processing
// @Description Job processing backend API for Atlan Collect
// @ID submit-job
// @Accept  json
// @Produce  json
// @Success 200 {object} httpResponse
// @Router /submit [post]
func submitJob(c *gin.Context) {
	newJobID := uuid.New()

	jobRequest, err := parseJobRequest(c)
	if err != nil {
		log.Println("Couldn't parse the job request")
		// w.Write([]byte(marshalError(errors.New("Invalid Job request format"), "")))
		c.JSON(http.StatusBadRequest, httpError{
			"",
			errors.New("Invalid Job request format"),
		})
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
			c.JSON(http.StatusBadRequest, httpError{
				"",
				errors.New("Invalid from_date format"),
			})
			// w.Write([]byte(marshalError(errors.New("Invalid from_date format"), "")))
			return
		}
		toDate, err := time.Parse(timeLayout, jobRequest.Args["to_date"].(string))
		if err != nil {
			c.JSON(http.StatusBadRequest, httpError{
				"",
				errors.New("Invalid to_date format"),
			})
			// w.Write([]byte(marshalError(errors.New("Invalid to_date format"), "")))
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
		c.JSON(http.StatusBadRequest, httpError{
			"",
			errors.New("Invalid Job Type"),
		})
		// w.Write([]byte(marshalError(errors.New("Invalid Job Type"), "")))
		return
	}

	jobs[newJobID] = newJob
	if err = newJob.start(); err != nil {
		log.Printf("Failed to start the job: %s\nError: %s", newJobID.String(), err.Error())
		// w.Write([]byte(marshalError(err, "")))
		c.JSON(http.StatusInternalServerError, httpError{
			"",
			err,
		})
		delete(jobs, newJobID)
		return
	}

	res, err := json.Marshal(httpResponse{
		JobID:   newJobID,
		Message: "Success",
		Details: make(map[string]interface{}),
	})
	c.JSON(http.StatusOK, res)
}

// haltJob godoc
// @Summary Halt a running job
// @Description Job processing backend API for Atlan Collect
// @ID halt-job
// @Accept  json
// @Produce  json
// @Param jobID path string true "Job ID"
// @Success 200 {object} httpResponse
// @Router /halt/{jobID} [get]
func haltJob(c *gin.Context) {
	jobID := c.Param("jobID")
	jobUUID, err := uuid.Parse(jobID)
	if err != nil {
		log.Println("Error while parsing UUID from string: ", jobID)
		c.JSON(http.StatusNotFound, httpError{
			jobID,
			errors.New("Invalid JobID"),
		})
		return
	}
	job, ok := jobs[jobUUID]
	if !ok {
		// w.Write([]byte(marshalError(errors.New("Invalid JobID"), jobID)))
		c.JSON(http.StatusNotFound, httpError{
			jobID,
			errors.New("Invalid JobID"),
		})
		return
	}

	if err = job.halt(); err != nil {
		log.Printf("Failed to stop the job: %s\nError: %s", jobID, err.Error())
		c.JSON(http.StatusInternalServerError, httpError{
			jobID,
			err,
		})
		// w.Write([]byte(marshalError(err, jobID)))
		return
	}
	res, err := json.Marshal(httpResponse{
		JobID:   jobUUID,
		Message: "Success",
		Details: make(map[string]interface{}),
	})
	log.Println("Halted job:", jobID)
	// w.Write([]byte(res))
	c.JSON(http.StatusOK, res)
}

// stopJob godoc
// @Summary Stop a  job
// @Description Job processing backend API for Atlan Collect
// @ID stop-job
// @Accept  json
// @Produce  json
// @Param jobID path string true "Job ID"
// @Success 200 {object} httpResponse
// @Router /stop/{jobID} [get]
func stopJob(c *gin.Context) {
	jobID := c.Param("jobID")
	jobUUID, err := uuid.Parse(jobID)
	if err != nil {
		log.Println("Error while parsing UUID from string: ", jobID)
		c.JSON(http.StatusNotFound, httpError{
			jobID,
			errors.New("Invalid JobID"),
		})
		return
	}
	job, ok := jobs[jobUUID]
	if !ok {
		c.JSON(http.StatusNotFound, httpError{
			jobID,
			errors.New("Invalid JobID"),
		})
		return
	}
	if err = job.stop(); err != nil {
		log.Printf("Failed to stop the job: %s\nError: %s\n", jobID, err.Error())
		c.JSON(http.StatusInternalServerError, httpError{
			jobID,
			err,
		})
		return
	}
	job.clean()
	delete(jobs, jobUUID)
	res, err := json.Marshal(httpResponse{
		JobID:   jobUUID,
		Message: "Success",
		Details: make(map[string]interface{}),
	})
	log.Println("Stopped job: ", jobID)
	c.JSON(http.StatusOK, res)
}

// resumeJob godoc
// @Summary Resume a pause/halted job
// @Description Job processing backend API for Atlan Collect
// @ID resume-job
// @Accept  json
// @Produce  json
// @Param jobID path string true "Job ID"
// @Success 200 {object} httpResponse
// @Router /resume/{jobID} [get]
func resumeJob(c *gin.Context) {
	jobID := c.Param("jobID")
	jobUUID, err := uuid.Parse(jobID)
	if err != nil {
		log.Println("Error while parsing UUID from string: ", jobID)
		c.JSON(http.StatusNotFound, httpError{
			jobID,
			errors.New("Invalid JobID"),
		})
		return
	}
	job, ok := jobs[jobUUID]
	if !ok {
		c.JSON(http.StatusNotFound, httpError{
			jobID,
			errors.New("Invalid JobID"),
		})
		return
	}
	if err = job.resume(); err != nil {
		log.Printf("Failed to stop the job: %s\nError: %s\n", jobID, err.Error())
		c.JSON(http.StatusInternalServerError, httpError{
			jobID,
			err,
		})
		return
	}
	res, err := json.Marshal(httpResponse{
		JobID:   jobUUID,
		Message: "Success",
		Details: make(map[string]interface{}),
	})
	log.Println("Resumed Job:", jobID)
	c.JSON(http.StatusOK, res)
}

// detailsJob godoc
// @Summary Fetch details about a submitted job
// @Description Job processing backend API for Atlan Collect
// @ID details-job
// @Accept  json
// @Produce  json
// @Param jobID path string true "Job ID"
// @Success 200 {object} httpResponse
// @Router /details/{jobID} [get]
func detailsJob(c *gin.Context) {
	jobID := c.Param("jobID")
	jobUUID, err := uuid.Parse(jobID)
	if err != nil {
		log.Println("Error while parsing UUID from string: ", jobID)
		c.JSON(http.StatusNotFound, httpError{
			jobID,
			errors.New("Invalid JobID"),
		})
		return
	}
	job, ok := jobs[jobUUID]
	if !ok {
		c.JSON(http.StatusNotFound, httpError{
			jobID,
			errors.New("Invalid JobID"),
		})
		return
	}
	details := job.details()
	res, err := json.Marshal(httpResponse{
		JobID:   jobUUID,
		Message: "Success",
		Details: details,
	})
	if err != nil {
		log.Println("Couldn't marshal job details")
		c.JSON(http.StatusInternalServerError, httpError{
			"",
			errors.New("Failed to fetch the details"),
		})
		return
	}
	c.JSON(http.StatusOK, res)
}

// @title Job submitting backend
// @version 0.1
// @description Job processing backend API for Atlan Collect
func main() {
	// Setup jobs queue
	jobs = make(map[uuid.UUID]JobInterface)
	r := initRouter()

	r.Run(":8080")
}

func initRouter() *gin.Engine {
	r := gin.Default()
	r.Use(gin.Logger())
	r.Use(gin.Recovery())
	r.POST("/submit", submitJob)
	r.GET("/halt/:jobID", haltJob)
	r.GET("/stop/:jobID", stopJob)
	r.GET("/resume/:jobID", resumeJob)
	r.GET("/details/:jobID", detailsJob)
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	return r
}
