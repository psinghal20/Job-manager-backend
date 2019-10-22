run:
	go run .

docker:
	docker build -t job-manager .

fmt:
	go fmt

build:
	go build
