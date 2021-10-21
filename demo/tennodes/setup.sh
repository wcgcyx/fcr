#!/bin/bash

# Setup
echo "Start setup..."

# Bootstrap
addr0=$(docker exec node0 fcr system addr | grep "/ip4/" | head -1 | xargs | tr -dc '[[:print:]]')
addr1=$(docker exec node1 fcr system addr | grep "/ip4/" | head -1 | xargs | tr -dc '[[:print:]]')
addr2=$(docker exec node2 fcr system addr | grep "/ip4/" | head -1 | xargs | tr -dc '[[:print:]]')
addr3=$(docker exec node3 fcr system addr | grep "/ip4/" | head -1 | xargs | tr -dc '[[:print:]]')
addr4=$(docker exec node4 fcr system addr | grep "/ip4/" | head -1 | xargs | tr -dc '[[:print:]]')
addr5=$(docker exec node5 fcr system addr | grep "/ip4/" | head -1 | xargs | tr -dc '[[:print:]]')
addr6=$(docker exec node6 fcr system addr | grep "/ip4/" | head -1 | xargs | tr -dc '[[:print:]]')
addr7=$(docker exec node7 fcr system addr | grep "/ip4/" | head -1 | xargs | tr -dc '[[:print:]]')
addr8=$(docker exec node8 fcr system addr | grep "/ip4/" | head -1 | xargs | tr -dc '[[:print:]]')
addr9=$(docker exec node9 fcr system addr | grep "/ip4/" | head -1 | xargs | tr -dc '[[:print:]]')
echo "node 0 has addr: $addr0"
echo "node 1 has addr: $addr1"
echo "node 2 has addr: $addr2"
echo "node 3 has addr: $addr3"
echo "node 4 has addr: $addr4"
echo "node 5 has addr: $addr5"
echo "node 6 has addr: $addr6"
echo "node 7 has addr: $addr7"
echo "node 8 has addr: $addr8"
echo "node 9 has addr: $addr9"
docker exec node1 fcr system bootstrap $addr0
docker exec node2 fcr system bootstrap $addr0
docker exec node3 fcr system bootstrap $addr0
docker exec node4 fcr system bootstrap $addr0
docker exec node5 fcr system bootstrap $addr0
docker exec node6 fcr system bootstrap $addr0
docker exec node7 fcr system bootstrap $addr0
docker exec node8 fcr system bootstrap $addr0
docker exec node9 fcr system bootstrap $addr0

# Create one random file to import by every node
docker exec node0 sh -c "tr -dc A-Za-z0-9 </dev/urandom | head -c 4096 > /app/testfile"
docker exec node1 sh -c "tr -dc A-Za-z0-9 </dev/urandom | head -c 4096 > /app/testfile"
docker exec node2 sh -c "tr -dc A-Za-z0-9 </dev/urandom | head -c 4096 > /app/testfile"
docker exec node3 sh -c "tr -dc A-Za-z0-9 </dev/urandom | head -c 4096 > /app/testfile"
docker exec node4 sh -c "tr -dc A-Za-z0-9 </dev/urandom | head -c 4096 > /app/testfile"
docker exec node5 sh -c "tr -dc A-Za-z0-9 </dev/urandom | head -c 4096 > /app/testfile"
docker exec node6 sh -c "tr -dc A-Za-z0-9 </dev/urandom | head -c 4096 > /app/testfile"
docker exec node7 sh -c "tr -dc A-Za-z0-9 </dev/urandom | head -c 4096 > /app/testfile"
docker exec node8 sh -c "tr -dc A-Za-z0-9 </dev/urandom | head -c 4096 > /app/testfile"
docker exec node9 sh -c "tr -dc A-Za-z0-9 </dev/urandom | head -c 4096 > /app/testfile"
# Import files
cid0=$(docker exec node0 sh -c "fcr cidnet piece import /app/testfile" | cut -d ":" -f2 | xargs | tr -dc '[[:print:]]')
cid1=$(docker exec node1 sh -c "fcr cidnet piece import /app/testfile" | cut -d ":" -f2 | xargs | tr -dc '[[:print:]]')
cid2=$(docker exec node2 sh -c "fcr cidnet piece import /app/testfile" | cut -d ":" -f2 | xargs | tr -dc '[[:print:]]')
cid3=$(docker exec node3 sh -c "fcr cidnet piece import /app/testfile" | cut -d ":" -f2 | xargs | tr -dc '[[:print:]]')
cid4=$(docker exec node4 sh -c "fcr cidnet piece import /app/testfile" | cut -d ":" -f2 | xargs | tr -dc '[[:print:]]')
cid5=$(docker exec node5 sh -c "fcr cidnet piece import /app/testfile" | cut -d ":" -f2 | xargs | tr -dc '[[:print:]]')
cid6=$(docker exec node6 sh -c "fcr cidnet piece import /app/testfile" | cut -d ":" -f2 | xargs | tr -dc '[[:print:]]')
cid7=$(docker exec node7 sh -c "fcr cidnet piece import /app/testfile" | cut -d ":" -f2 | xargs | tr -dc '[[:print:]]')
cid8=$(docker exec node8 sh -c "fcr cidnet piece import /app/testfile" | cut -d ":" -f2 | xargs | tr -dc '[[:print:]]')
cid9=$(docker exec node9 sh -c "fcr cidnet piece import /app/testfile" | cut -d ":" -f2 | xargs | tr -dc '[[:print:]]')
echo "node 0 import cid: $cid0"
echo "node 1 import cid: $cid1"
echo "node 2 import cid: $cid2"
echo "node 3 import cid: $cid3"
echo "node 4 import cid: $cid4"
echo "node 5 import cid: $cid5"
echo "node 6 import cid: $cid6"
echo "node 7 import cid: $cid7"
echo "node 8 import cid: $cid8"
echo "node 9 import cid: $cid9"
# Start serving cids
docker exec node0 sh -c "fcr cidnet serving serve 1 $cid0 1"
docker exec node1 sh -c "fcr cidnet serving serve 1 $cid1 1"
docker exec node2 sh -c "fcr cidnet serving serve 1 $cid2 1"
docker exec node3 sh -c "fcr cidnet serving serve 1 $cid3 1"
docker exec node4 sh -c "fcr cidnet serving serve 1 $cid4 1"
docker exec node5 sh -c "fcr cidnet serving serve 1 $cid5 1"
docker exec node6 sh -c "fcr cidnet serving serve 1 $cid6 1"
docker exec node7 sh -c "fcr cidnet serving serve 1 $cid7 1"
docker exec node8 sh -c "fcr cidnet serving serve 1 $cid8 1"
docker exec node9 sh -c "fcr cidnet serving serve 1 $cid9 1"
# Force node to publish record immediately
docker exec node0 sh -c "fcr cidnet serving force-publish"
docker exec node1 sh -c "fcr cidnet serving force-publish"
docker exec node2 sh -c "fcr cidnet serving force-publish"
docker exec node3 sh -c "fcr cidnet serving force-publish"
docker exec node4 sh -c "fcr cidnet serving force-publish"
docker exec node5 sh -c "fcr cidnet serving force-publish"
docker exec node6 sh -c "fcr cidnet serving force-publish"
docker exec node7 sh -c "fcr cidnet serving force-publish"
docker exec node8 sh -c "fcr cidnet serving force-publish"
docker exec node9 sh -c "fcr cidnet serving force-publish"

# Setup initial payment network connection.
wallet0=$(docker exec node0 sh -c "fcr paynet root 1" | cut -d ":" -f2 | xargs | tr -dc '[[:print:]]')
wallet1=$(docker exec node1 sh -c "fcr paynet root 1" | cut -d ":" -f2 | xargs | tr -dc '[[:print:]]')
wallet2=$(docker exec node2 sh -c "fcr paynet root 1" | cut -d ":" -f2 | xargs | tr -dc '[[:print:]]')
wallet3=$(docker exec node3 sh -c "fcr paynet root 1" | cut -d ":" -f2 | xargs | tr -dc '[[:print:]]')
wallet4=$(docker exec node4 sh -c "fcr paynet root 1" | cut -d ":" -f2 | xargs | tr -dc '[[:print:]]')
wallet5=$(docker exec node5 sh -c "fcr paynet root 1" | cut -d ":" -f2 | xargs | tr -dc '[[:print:]]')
wallet6=$(docker exec node6 sh -c "fcr paynet root 1" | cut -d ":" -f2 | xargs | tr -dc '[[:print:]]')
wallet7=$(docker exec node7 sh -c "fcr paynet root 1" | cut -d ":" -f2 | xargs | tr -dc '[[:print:]]')
wallet8=$(docker exec node8 sh -c "fcr paynet root 1" | cut -d ":" -f2 | xargs | tr -dc '[[:print:]]')
wallet9=$(docker exec node9 sh -c "fcr paynet root 1" | cut -d ":" -f2 | xargs | tr -dc '[[:print:]]')
echo "node 0 wallet address is $wallet0"
echo "node 1 wallet address is $wallet1"
echo "node 2 wallet address is $wallet2"
echo "node 3 wallet address is $wallet3"
echo "node 4 wallet address is $wallet4"
echo "node 5 wallet address is $wallet5"
echo "node 6 wallet address is $wallet6"
echo "node 7 wallet address is $wallet7"
echo "node 8 wallet address is $wallet8"
echo "node 9 wallet address is $wallet9"
# Topup account
echo "Topup accounts..."
receipt0=$(docker exec lotus sh -c "./lotus send $wallet0 10")
receipt1=$(docker exec lotus sh -c "./lotus send $wallet1 10")
receipt2=$(docker exec lotus sh -c "./lotus send $wallet2 10")
receipt3=$(docker exec lotus sh -c "./lotus send $wallet3 10")
receipt4=$(docker exec lotus sh -c "./lotus send $wallet4 10")
receipt5=$(docker exec lotus sh -c "./lotus send $wallet5 10")
receipt6=$(docker exec lotus sh -c "./lotus send $wallet6 10")
receipt7=$(docker exec lotus sh -c "./lotus send $wallet7 10")
receipt8=$(docker exec lotus sh -c "./lotus send $wallet8 10")
receipt9=$(docker exec lotus sh -c "./lotus send $wallet9 10")
echo "Wait for transactions to take effect..."
echo "wait for $receipt0"
docker exec lotus sh -c "./lotus state wait-msg $receipt0"
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
echo "wait for $receipt6"
docker exec lotus sh -c "./lotus state wait-msg $receipt6"
echo "wait for $receipt7"
docker exec lotus sh -c "./lotus state wait-msg $receipt7"
echo "wait for $receipt8"
docker exec lotus sh -c "./lotus state wait-msg $receipt8"
echo "wait for $receipt9"
docker exec lotus sh -c "./lotus state wait-msg $receipt9"
# Create payment channels
docker exec node0 sh -c "(echo yes) | fcr paynet outbound create 1 $wallet1 10000000000000 $addr1"
docker exec node1 sh -c "(echo yes) | fcr paynet outbound create 1 $wallet2 10000000000000 $addr2"
docker exec node1 sh -c "(echo yes) | fcr paynet outbound create 1 $wallet4 10000000000000 $addr4"
docker exec node1 sh -c "(echo yes) | fcr paynet outbound create 1 $wallet6 10000000000000 $addr6"
docker exec node2 sh -c "(echo yes) | fcr paynet outbound create 1 $wallet3 10000000000000 $addr3"
docker exec node3 sh -c "(echo yes) | fcr paynet outbound create 1 $wallet0 10000000000000 $addr0"
docker exec node4 sh -c "(echo yes) | fcr paynet outbound create 1 $wallet5 10000000000000 $addr5" 
docker exec node4 sh -c "(echo yes) | fcr paynet outbound create 1 $wallet7 10000000000000 $addr7"
docker exec node5 sh -c "(echo yes) | fcr paynet outbound create 1 $wallet2 10000000000000 $addr2"
docker exec node6 sh -c "(echo yes) | fcr paynet outbound create 1 $wallet7 10000000000000 $addr7"
docker exec node7 sh -c "(echo yes) | fcr paynet outbound create 1 $wallet8 10000000000000 $addr8"
docker exec node8 sh -c "(echo yes) | fcr paynet outbound create 1 $wallet5 10000000000000 $addr5"
docker exec node8 sh -c "(echo yes) | fcr paynet outbound create 1 $wallet9 10000000000000 $addr9"
docker exec node9 sh -c "(echo yes) | fcr paynet outbound create 1 $wallet8 10000000000000 $addr8"
wait
# Start serving channels
docker exec node0 sh -c "fcr paynet serving serve 1 $wallet1 5 100"
docker exec node1 sh -c "fcr paynet serving serve 1 $wallet2 5 100"
docker exec node1 sh -c "fcr paynet serving serve 1 $wallet4 5 100"
docker exec node1 sh -c "fcr paynet serving serve 1 $wallet6 5 100"
docker exec node2 sh -c "fcr paynet serving serve 1 $wallet3 5 100"
docker exec node3 sh -c "fcr paynet serving serve 1 $wallet0 5 100"
docker exec node4 sh -c "fcr paynet serving serve 1 $wallet5 5 100"
docker exec node4 sh -c "fcr paynet serving serve 1 $wallet7 5 100"
docker exec node5 sh -c "fcr paynet serving serve 1 $wallet2 5 100"
docker exec node6 sh -c "fcr paynet serving serve 1 $wallet7 5 100"
docker exec node7 sh -c "fcr paynet serving serve 1 $wallet8 5 100"
docker exec node8 sh -c "fcr paynet serving serve 1 $wallet5 5 100"
docker exec node8 sh -c "fcr paynet serving serve 1 $wallet9 5 100"
docker exec node9 sh -c "fcr paynet serving serve 1 $wallet8 5 100"
# Force node to publish record four times (make sure every payment route is propagated to the network) immediately
for i in {1..4}
do
    docker exec node9 sh -c "fcr paynet serving force-publish"
    sleep .5
    docker exec node8 sh -c "fcr paynet serving force-publish"
    sleep .5
    docker exec node7 sh -c "fcr paynet serving force-publish"
    sleep .5
    docker exec node6 sh -c "fcr paynet serving force-publish"
    sleep .5
    docker exec node5 sh -c "fcr paynet serving force-publish"
    sleep .5
    docker exec node4 sh -c "fcr paynet serving force-publish"
    sleep .5
    docker exec node3 sh -c "fcr paynet serving force-publish"
    sleep .5
    docker exec node2 sh -c "fcr paynet serving force-publish"
    sleep .5
    docker exec node1 sh -c "fcr paynet serving force-publish"
    sleep .5
    docker exec node0 sh -c "fcr paynet serving force-publish"
    sleep .5
    docker exec node9 sh -c "fcr paynet serving force-publish"
    sleep .5
done
# Done setup
echo "Done setup"
