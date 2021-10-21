#!/bin/bash

set -m
./lotus daemon --lotus-make-genesis=devgen.car --genesis-template=localnet.json --bootstrap=false &
sleep 30
./lotus wallet import --as-default ~/.genesis-sectors/pre-seal-t01000.key
./lotus-miner init --genesis-miner --actor=t01000 --sector-size=2KiB --pre-sealed-sectors=~/.genesis-sectors --pre-sealed-metadata=~/.genesis-sectors/pre-seal-t01000.json
./lotus-miner run --nosync &
height=$(./lotus chain getblock $(./lotus chain head) | jq -r ".Height")
while [ $height -lt 50 ]
do
	height=$(./lotus chain getblock $(./lotus chain head) | jq -r ".Height")
    echo $height
	sleep 5
done
./lotus-miner stop
./lotus daemon stop
