#!/bin/bash

# Setup
echo "Start setup..."

# Bootstrap
addr1=$(docker exec node1 fcr system addr | grep "/ip4/" | head -1 | xargs | tr -dc '[[:print:]]')
addr2=$(docker exec node2 fcr system addr | grep "/ip4/" | head -1 | xargs | tr -dc '[[:print:]]')
addr3=$(docker exec node3 fcr system addr | grep "/ip4/" | head -1 | xargs | tr -dc '[[:print:]]')
addr4=$(docker exec node4 fcr system addr | grep "/ip4/" | head -1 | xargs | tr -dc '[[:print:]]')
addr5=$(docker exec node5 fcr system addr | grep "/ip4/" | head -1 | xargs | tr -dc '[[:print:]]')
echo "node 1 has addr: $addr1"
echo "node 2 has addr: $addr2"
echo "node 3 has addr: $addr3"
echo "node 4 has addr: $addr4"
echo "node 5 has addr: $addr5"
docker exec node2 fcr system bootstrap $addr1
docker exec node3 fcr system bootstrap $addr1
docker exec node4 fcr system bootstrap $addr1
docker exec node5 fcr system bootstrap $addr1

# Create one random file to import by every node
docker exec node1 sh -c "tr -dc A-Za-z0-9 </dev/urandom | head -c 4096 > /app/testfile"
docker exec node2 sh -c "tr -dc A-Za-z0-9 </dev/urandom | head -c 4096 > /app/testfile1"
docker exec node2 sh -c "tr -dc A-Za-z0-9 </dev/urandom | head -c 4096 > /app/testfile2"
docker exec node3 sh -c "tr -dc A-Za-z0-9 </dev/urandom | head -c 4096 > /app/testfile"
docker exec node4 sh -c "tr -dc A-Za-z0-9 </dev/urandom | head -c 4096 > /app/testfile"
docker exec node5 sh -c "tr -dc A-Za-z0-9 </dev/urandom | head -c 4096 > /app/testfile"
# Import files
cid1=$(docker exec node1 sh -c "fcr cidnet piece import /app/testfile" | cut -d ":" -f2 | xargs | tr -dc '[[:print:]]')
cid2_1=$(docker exec node2 sh -c "fcr cidnet piece import /app/testfile1" | cut -d ":" -f2 | xargs | tr -dc '[[:print:]]')
cid2_2=$(docker exec node2 sh -c "fcr cidnet piece import /app/testfile2" | cut -d ":" -f2 | xargs | tr -dc '[[:print:]]')
cid3=$(docker exec node3 sh -c "fcr cidnet piece import /app/testfile" | cut -d ":" -f2 | xargs | tr -dc '[[:print:]]')
cid4=$(docker exec node4 sh -c "fcr cidnet piece import /app/testfile" | cut -d ":" -f2 | xargs | tr -dc '[[:print:]]')
cid5=$(docker exec node5 sh -c "fcr cidnet piece import /app/testfile" | cut -d ":" -f2 | xargs | tr -dc '[[:print:]]')
echo "node 1 import cid: $cid1"
echo "node 2 import cid: $cid2_1 $cid2_2"
echo "node 3 import cid: $cid3"
echo "node 4 import cid: $cid4"
echo "node 5 import cid: $cid5"
# Start serving cids
docker exec node1 sh -c "fcr cidnet serving serve 1 $cid1 1"
docker exec node2 sh -c "fcr cidnet serving serve 1 $cid2_1 1"
docker exec node2 sh -c "fcr cidnet serving serve 1 $cid2_2 1"
docker exec node3 sh -c "fcr cidnet serving serve 1 $cid3 1"
docker exec node4 sh -c "fcr cidnet serving serve 1 $cid4 1"
docker exec node5 sh -c "fcr cidnet serving serve 1 $cid5 1"
# Force node to publish record immediately
docker exec node1 sh -c "fcr cidnet serving force-publish"
docker exec node2 sh -c "fcr cidnet serving force-publish"
docker exec node3 sh -c "fcr cidnet serving force-publish"
docker exec node4 sh -c "fcr cidnet serving force-publish"
docker exec node5 sh -c "fcr cidnet serving force-publish"

# Setup initial payment network connection.
# node 2 -> node 3 -> node 4 -> node 5
wallet1=$(docker exec node1 sh -c "fcr paynet root 1" | cut -d ":" -f2 | xargs | tr -dc '[[:print:]]')
wallet2=$(docker exec node2 sh -c "fcr paynet root 1" | cut -d ":" -f2 | xargs | tr -dc '[[:print:]]')
wallet3=$(docker exec node3 sh -c "fcr paynet root 1" | cut -d ":" -f2 | xargs | tr -dc '[[:print:]]')
wallet4=$(docker exec node4 sh -c "fcr paynet root 1" | cut -d ":" -f2 | xargs | tr -dc '[[:print:]]')
wallet5=$(docker exec node5 sh -c "fcr paynet root 1" | cut -d ":" -f2 | xargs | tr -dc '[[:print:]]')
echo "node 1 wallet address is $wallet1"
echo "node 2 wallet address is $wallet2"
echo "node 3 wallet address is $wallet3"
echo "node 4 wallet address is $wallet4"
echo "node 5 wallet address is $wallet5"
# Topup account
echo "Topup accounts..."
receipt1=$(docker exec lotus sh -c "./lotus send $wallet1 10")
receipt2=$(docker exec lotus sh -c "./lotus send $wallet2 10")
receipt3=$(docker exec lotus sh -c "./lotus send $wallet3 10")
receipt4=$(docker exec lotus sh -c "./lotus send $wallet4 10")
receipt5=$(docker exec lotus sh -c "./lotus send $wallet5 10")
echo "Wait for transactions to take effect..."
echo "wait for $receipt1"
docker exec lotus sh -c "./lotus state wait-msg $receipt1"
echo "wait for $receipt2"
docker exec lotus sh -c "./lotus state wait-msg $receipt2"
echo "wait for $receipt3"
docker exec lotus sh -c "./lotus state wait-msg $receipt3"
echo "wait for $receipt4"
docker exec lotus sh -c "./lotus state wait-msg $receipt4"
echo "wait for $receipt5"
docker exec lotus sh -c "./lotus state wait-msg $receipt5"
# Create payment channels
docker exec node2 sh -c "(echo yes) | fcr paynet outbound create 1 $wallet3 10000000000000 $addr3"
docker exec node3 sh -c "(echo yes) | fcr paynet outbound create 1 $wallet4 10000000000000 $addr4"
docker exec node4 sh -c "(echo yes) | fcr paynet outbound create 1 $wallet5 10000000000000 $addr5"
# Start serving channels
docker exec node4 sh -c "fcr paynet serving serve 1 $wallet5 5 100"
docker exec node3 sh -c "fcr paynet serving serve 1 $wallet4 5 100"
docker exec node2 sh -c "fcr paynet serving serve 1 $wallet3 5 100"
# Force node to publish record immediately
docker exec node4 sh -c "fcr paynet serving force-publish"
docker exec node3 sh -c "fcr paynet serving force-publish"
docker exec node2 sh -c "fcr paynet serving force-publish"

# Done setup
echo "node 1 serving: $cid1"
echo "node 2 serving: $cid2_1 $cid2_2"
echo "node 3 serving: $cid3"
echo "node 4 serving: $cid4"
echo "node 5 serving: $cid5"
echo "Done setup"
