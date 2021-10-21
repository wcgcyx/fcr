# fcr

![alt text][logo]

[logo]: logo.png "FCR"

## _Yet another filecoin secondary retrieval client_
> FCR is a filecoin secondary retrieval client featured with the ability to participate in an ipld  retrieval network and a payment proxy network.
> You can earn filecoin (FIL) by importing a regular file or a .car file or a lotus unsealed sector and serving the ipld graph to the network for paid retrieval.
> -OR-
> You can earn the surcharge by creating a payment channel and serving the payment channel to the network for others to do a proxy payment.

## Features
- Access to a CID Network based on Kademlia DHT.
- Access to an incentivized Payment Network.

## Install & Run
```sh
make build
./build/fcr --version
```

## Run demo
This repository contains two demo: a five nodes network demo and a ten nodes network demo.
To build the demo using mocked chain:
```sh
make demo-mc
```
To build the demo using local Filecoin devnet:
```sh
make demo
make lotus
```
For example, to run a five nodes network demo with mocked chain:
```sh
cd ./demo/fivenodes-mc
./up.sh
```
Note: When using mocked chain, the currency ID should be 0. When using lotus devnet, the currency ID should be 1. Also, don't forget to run `docker compose rm -f` to clean after done with the demo.

## Things to do
0. *__More document!__*
1. Currently, FCR does not support Filecoin Mainnet, because it currently only supports actor version 4, a little bit more work will be done to support actor version 6 so it supports Mainnet.
2. The overall system has a lot of things that can be improved/simplifed.
3. At the moment, the initialisation process needs to be done by asking for user prompt. There should be a way to initialise FCR with a configuration.
4. Need to test a lot of failure cases especially around disk storage operation.
5. Need to test import sector.

## One last thing
FCR is designed to support multicurrency as you can see the term `currency_ID` appears all around the repository. At the moment, only FIL is supported with the `currency_ID` to be 1.

## Warning
Please be warned that this project should be considered as a proof-of-concept ONLY. You should not use it for production purpose.

## Contributor
Zhenyang Shi

## License
Apache 2.0
