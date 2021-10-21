.PHONY: demo build

build:
	go build -o ./build/fcr cmd/fcr/* 

mc:
	go build -o ./build/mc internal/paymgr/chainmgr/mockchain/*

clean:
	rm -rf ./build/*

demo:
	docker build -t wcgcyx/fcr/demo-fcr -f ./demo/Dockerfile.fcr .

demo-mc:
	docker build -t wcgcyx/fcr/demo-fcr-mc -f ./demo/Dockerfile.fcr-mc .
	docker build -t wcgcyx/fcr/demo-mc -f ./demo/Dockerfile.mc .

lotus:
	docker build -t wcgcyx/fcr/demo-lotus -f ./demo/Dockerfile.lotus .
