# fcr

![alt text](logo.png)

## _Yet another filecoin secondary retrieval client_
> FCR is a filecoin secondary retrieval client featured with the ability to participate in an ipld  retrieval network and a payment proxy network.
> You can earn filecoin (FIL) by importing a regular file or a .car file or a lotus unsealed sector and serving the ipld graph to the network for paid retrieval.
>
> -OR-
>
> You can earn the surcharge by creating a payment channel and serving the payment channel to the network for others to do a proxy payment.

## Build and install
### Prerequisites
Building `fcr` currently requires a Go (version 1.17.9 or later). To obtain Go, visit [here](https://go.dev/doc/install).
### Instructions
1. Clone the repository and checkout to one of the releases [here](https://github.com/wcgcyx/fcr/releases). 
```
git clone https://github.com/wcgcyx/fcr.git
cd fcr/
git checkout <version>
```
2. Build and install the binary.
```
make build
sudo mv ./build/fcr /usr/local/bin/fcr
```
3. Test installation.
```
fcr version
```
4. Now you should have FCR installed. You can now start running an FCR node.
## Run FCR
At the moment, FCR network runs only on [filecoin calibration network](https://docs.filecoin.io/networks/overview/) with the currency ID be 1.
1. Add configuration.
```
mkdir ~/.fcr/
mv mv ./config/config.yaml ~/.fcr/
```
The configuration file by default uses the [testnet glif node endpoint](https://lotus.filecoin.io/developers/glif-nodes/#testnet-endpoint). If you wish to use local lotus node to access the network, set `TRANSACTOR_FILECOIN_AP` to be the local lotus API address (by default `http://localhost:1234/rpc/v0`) and set `TRANSACTOR_FILECOIN_AUTH_TOKEN` to be the API token.

2. Run daemon.
```
fcr daemon
```
3. Set a private key for sending/receiving payment.

It is recommended that you generate a random account to join FCR network at this stage.
```
fcr wallet generate 1 # This will generate a private key
fcr wallet set 1 1 ${private key}
```
Once you set the private key. You can inspect your filecoin address by `fcr wallet list -l`.

You can then obtain some testing FIL from the faucet [here](https://faucet.calibration.fildev.network/).

4. Bootstrapping

To connect the network, you need to do a bootstrapping. You can find a list of available peers here (TBD).
```
fcr system connect ${peer addr}
```
5. Now you should have joined the FCR network.
## Simple example
To try a simple video sharing website that builds on top of FCR, visit [here](https://github.com/wcgcyx/fcr-simple-example). 

Normally, it is expensive to host your own video sharing website because you need to at least have a large amount of storage to store videos and a large network bandwidth to serve the videos to users. 

However, using FCR, you can setup a website that only serves video IDs to the user. The actual data is stored in FCR nodes that operate as retrieval providers and user will be able to stream videos from the peer-to-peer network via its own FCR node as a retrieval client.
## Documentation
Detailed documentation and instructions on running FCR as retrieval provider/payment provider are WIP.
## Issues
Feel free to open an issue [here](https://github.com/wcgcyx/fcr/issues). Otherwise if you have questions or wish to discuss, you can join the [discord channel](https://discord.gg/GgK9eqrNtG).
## Contributor
Zhenyang Shi - wcgcyx@gmail.com
## License
Dual-licensed under MIT and Apache 2.0.