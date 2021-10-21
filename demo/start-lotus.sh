#!/bin/bash

(./lotus daemon --genesis=devgen.car --bootstrap=false &) > /dev/null 2>&1
sleep 5
./lotus-miner run --nosync > /dev/null 2>&1
