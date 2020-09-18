local:
	GOOS=linux GOARCH=amd64 go build -o greedy-scheduler ./cmd/scheduler 
build:
	docker build --no-cache . -t fusidic/greedy-scheduler:0.1
push:
	docker push fusidic/greedy-scheduler:0.1

format:
	sudo gofmt -l -w .
clean:
	sudo rm -f scheduler