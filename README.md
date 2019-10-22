# Job Manager Backend

This project is a simple Job manager backend to submit different kinds of jobs to a processing pipeline. It supports basic operations of submitting, pausing, resuming, stopping and checking details of a job. It is written in go with [Gin](https://github.com/gin-gonic/gin) framework. 

This project is done as part of an internship task for [Atlan](https://atlan.com/).

### Installing Go
You need to have installed Go to run this project. You can find the instructions and binaries to install Go [here](https://golang.org/doc/install).

## Quick start
- Clone the repo
- Install the dependecies by running: `go mod download`
- Run `go run .` inside the cloned directory

You can build the binary file by running:

## Running via docker container
You can build the docker image by running:
    make docker

Start the docker container by running:
    docker run -d -p 8080:8080 job-manager

*You might need to run docker commands as root user*

## Routes
    POST /submit
    GET /halt/:jobID
    GET /stop/:jobID
    GET /resume/:jobID
    GET /details/:jobID
    GET /swagger/

The API takes `jobID` as path argument for `GET` routes and the `POST` routes take a JSON body of format:
```json5
    {
        "Type": "Simple", // There are two types of jobs currently supported: Simple and Export
        "args": {
            "key": "value",
        }
    }
```

You can find detailed API documentation on [swagger](http://localhost:8080/swagger/index.html) after running the server. Instructions to start the server are mentioned above.

## Adding different jobs
Job manager provides a simple go interface for different types of jobs to be processed by the pipeline.
```go
// JobInterface is the common interface that every
// different job should implement
type JobInterface interface {
	start() error // Start the job processing
	halt() error // Halt or pause the job processing
	resume() error // Resume any halted/paused job
	stop() error // Stop processing of any running or halted job
	clean() error // Clean method can be used to rollback any changes when job is stopped
	details() map[string]interface{} // Return details about the Job as a Map
}
```
Different jobs can implement these methods to provide the similar interface to the API. 

Two sample implementations are provided as examples and can be found in [job.go](./job.go). These implementations provide 2 simple scenarios:
- One is a simple job, which just runs a loop and prints a statement
- Another is a Simple Export job, which take two arguments: `from_date` and `to_date`. Current implementation doesn't do anything and just runs a loop similar to above case but can be extended to intergrate any database to export database.

## License
This project is under MIT License. See the [LICENSE](./LICENSE) for details.