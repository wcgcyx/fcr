.PHONY: build itest

build:
	go build -o ./build/fcr cmd/fcr/* 

docker:
	docker build -t wcgcyx/fcr .

utest:
	go test --timeout 5m -v --count=1 $(shell go list ./... | grep -v itest)

itest:
	cd itest; \
	docker compose up --force-recreate -d; \
	sleep 10; \
	go test --timeout 10m -v --count=1; \
	docker compose down

clean:
	rm -rf ./build/*
