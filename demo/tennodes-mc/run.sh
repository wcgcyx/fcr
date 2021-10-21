#!/bin/bash

# Get cid imported by every node
cid0=$(docker exec node0 fcr cidnet piece list | tail -1 | xargs | tr -dc '[[:print:]]')
cid1=$(docker exec node1 fcr cidnet piece list | tail -1 | xargs | tr -dc '[[:print:]]')
cid2=$(docker exec node2 fcr cidnet piece list | tail -1 | xargs | tr -dc '[[:print:]]')
cid3=$(docker exec node3 fcr cidnet piece list | tail -1 | xargs | tr -dc '[[:print:]]')
cid4=$(docker exec node4 fcr cidnet piece list | tail -1 | xargs | tr -dc '[[:print:]]')
cid5=$(docker exec node5 fcr cidnet piece list | tail -1 | xargs | tr -dc '[[:print:]]')
cid6=$(docker exec node6 fcr cidnet piece list | tail -1 | xargs | tr -dc '[[:print:]]')
cid7=$(docker exec node7 fcr cidnet piece list | tail -1 | xargs | tr -dc '[[:print:]]')
cid8=$(docker exec node8 fcr cidnet piece list | tail -1 | xargs | tr -dc '[[:print:]]')
cid9=$(docker exec node9 fcr cidnet piece list | tail -1 | xargs | tr -dc '[[:print:]]')
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

# 90 concurrent retrievals
# docker exec node0 sh -c "fcr fast-retrieve 0 $cid0 . 0" &
docker exec node0 sh -c "fcr fast-retrieve 0 $cid1 . 0" &
docker exec node0 sh -c "fcr fast-retrieve 0 $cid2 . 0" &
docker exec node0 sh -c "fcr fast-retrieve 0 $cid3 . 0" &
docker exec node0 sh -c "fcr fast-retrieve 0 $cid4 . 0" &
docker exec node0 sh -c "fcr fast-retrieve 0 $cid5 . 0" &
docker exec node0 sh -c "fcr fast-retrieve 0 $cid6 . 0" &
docker exec node0 sh -c "fcr fast-retrieve 0 $cid7 . 0" &
docker exec node0 sh -c "fcr fast-retrieve 0 $cid8 . 0" &
docker exec node0 sh -c "fcr fast-retrieve 0 $cid9 . 0" &

docker exec node1 sh -c "fcr fast-retrieve 0 $cid0 . 0" &
# docker exec node1 sh -c "fcr fast-retrieve 0 $cid1 . 0" &
docker exec node1 sh -c "fcr fast-retrieve 0 $cid2 . 0" &
docker exec node1 sh -c "fcr fast-retrieve 0 $cid3 . 0" &
docker exec node1 sh -c "fcr fast-retrieve 0 $cid4 . 0" &
docker exec node1 sh -c "fcr fast-retrieve 0 $cid5 . 0" &
docker exec node1 sh -c "fcr fast-retrieve 0 $cid6 . 0" &
docker exec node1 sh -c "fcr fast-retrieve 0 $cid7 . 0" &
docker exec node1 sh -c "fcr fast-retrieve 0 $cid8 . 0" &
docker exec node1 sh -c "fcr fast-retrieve 0 $cid9 . 0" &

docker exec node2 sh -c "fcr fast-retrieve 0 $cid0 . 0" &
docker exec node2 sh -c "fcr fast-retrieve 0 $cid1 . 0" &
# docker exec node2 sh -c "fcr fast-retrieve 0 $cid2 . 0" &
docker exec node2 sh -c "fcr fast-retrieve 0 $cid3 . 0" &
docker exec node2 sh -c "fcr fast-retrieve 0 $cid4 . 0" &
docker exec node2 sh -c "fcr fast-retrieve 0 $cid5 . 0" &
docker exec node2 sh -c "fcr fast-retrieve 0 $cid6 . 0" &
docker exec node2 sh -c "fcr fast-retrieve 0 $cid7 . 0" &
docker exec node2 sh -c "fcr fast-retrieve 0 $cid8 . 0" &
docker exec node2 sh -c "fcr fast-retrieve 0 $cid9 . 0" &

docker exec node3 sh -c "fcr fast-retrieve 0 $cid0 . 0" &
docker exec node3 sh -c "fcr fast-retrieve 0 $cid1 . 0" &
docker exec node3 sh -c "fcr fast-retrieve 0 $cid2 . 0" &
# docker exec node3 sh -c "fcr fast-retrieve 0 $cid3 . 0" &
docker exec node3 sh -c "fcr fast-retrieve 0 $cid4 . 0" &
docker exec node3 sh -c "fcr fast-retrieve 0 $cid5 . 0" &
docker exec node3 sh -c "fcr fast-retrieve 0 $cid6 . 0" &
docker exec node3 sh -c "fcr fast-retrieve 0 $cid7 . 0" &
docker exec node3 sh -c "fcr fast-retrieve 0 $cid8 . 0" &
docker exec node3 sh -c "fcr fast-retrieve 0 $cid9 . 0" &

docker exec node4 sh -c "fcr fast-retrieve 0 $cid0 . 0" &
docker exec node4 sh -c "fcr fast-retrieve 0 $cid1 . 0" &
docker exec node4 sh -c "fcr fast-retrieve 0 $cid2 . 0" &
docker exec node4 sh -c "fcr fast-retrieve 0 $cid3 . 0" &
# docker exec node4 sh -c "fcr fast-retrieve 0 $cid4 . 0" &
docker exec node4 sh -c "fcr fast-retrieve 0 $cid5 . 0" &
docker exec node4 sh -c "fcr fast-retrieve 0 $cid6 . 0" &
docker exec node4 sh -c "fcr fast-retrieve 0 $cid7 . 0" &
docker exec node4 sh -c "fcr fast-retrieve 0 $cid8 . 0" &
docker exec node4 sh -c "fcr fast-retrieve 0 $cid9 . 0" &

docker exec node5 sh -c "fcr fast-retrieve 0 $cid0 . 0" &
docker exec node5 sh -c "fcr fast-retrieve 0 $cid1 . 0" &
docker exec node5 sh -c "fcr fast-retrieve 0 $cid2 . 0" &
docker exec node5 sh -c "fcr fast-retrieve 0 $cid3 . 0" &
docker exec node5 sh -c "fcr fast-retrieve 0 $cid4 . 0" &
# docker exec node5 sh -c "fcr fast-retrieve 0 $cid5 . 0" &
docker exec node5 sh -c "fcr fast-retrieve 0 $cid6 . 0" &
docker exec node5 sh -c "fcr fast-retrieve 0 $cid7 . 0" &
docker exec node5 sh -c "fcr fast-retrieve 0 $cid8 . 0" &
docker exec node5 sh -c "fcr fast-retrieve 0 $cid9 . 0" &

docker exec node6 sh -c "fcr fast-retrieve 0 $cid0 . 0" &
docker exec node6 sh -c "fcr fast-retrieve 0 $cid1 . 0" &
docker exec node6 sh -c "fcr fast-retrieve 0 $cid2 . 0" &
docker exec node6 sh -c "fcr fast-retrieve 0 $cid3 . 0" &
docker exec node6 sh -c "fcr fast-retrieve 0 $cid4 . 0" &
docker exec node6 sh -c "fcr fast-retrieve 0 $cid5 . 0" &
# docker exec node6 sh -c "fcr fast-retrieve 0 $cid6 . 0" &
docker exec node6 sh -c "fcr fast-retrieve 0 $cid7 . 0" &
docker exec node6 sh -c "fcr fast-retrieve 0 $cid8 . 0" &
docker exec node6 sh -c "fcr fast-retrieve 0 $cid9 . 0" &

docker exec node7 sh -c "fcr fast-retrieve 0 $cid0 . 0" &
docker exec node7 sh -c "fcr fast-retrieve 0 $cid1 . 0" &
docker exec node7 sh -c "fcr fast-retrieve 0 $cid2 . 0" &
docker exec node7 sh -c "fcr fast-retrieve 0 $cid3 . 0" &
docker exec node7 sh -c "fcr fast-retrieve 0 $cid4 . 0" &
docker exec node7 sh -c "fcr fast-retrieve 0 $cid5 . 0" &
docker exec node7 sh -c "fcr fast-retrieve 0 $cid6 . 0" &
# docker exec node7 sh -c "fcr fast-retrieve 0 $cid7 . 0" &
docker exec node7 sh -c "fcr fast-retrieve 0 $cid8 . 0" &
docker exec node7 sh -c "fcr fast-retrieve 0 $cid9 . 0" &

docker exec node8 sh -c "fcr fast-retrieve 0 $cid0 . 0" &
docker exec node8 sh -c "fcr fast-retrieve 0 $cid1 . 0" &
docker exec node8 sh -c "fcr fast-retrieve 0 $cid2 . 0" &
docker exec node8 sh -c "fcr fast-retrieve 0 $cid3 . 0" &
docker exec node8 sh -c "fcr fast-retrieve 0 $cid4 . 0" &
docker exec node8 sh -c "fcr fast-retrieve 0 $cid5 . 0" &
docker exec node8 sh -c "fcr fast-retrieve 0 $cid6 . 0" &
docker exec node8 sh -c "fcr fast-retrieve 0 $cid7 . 0" &
# docker exec node8 sh -c "fcr fast-retrieve 0 $cid8 . 0" &
docker exec node8 sh -c "fcr fast-retrieve 0 $cid9 . 0" &

docker exec node9 sh -c "fcr fast-retrieve 0 $cid0 . 0" &
docker exec node9 sh -c "fcr fast-retrieve 0 $cid1 . 0" &
docker exec node9 sh -c "fcr fast-retrieve 0 $cid2 . 0" &
docker exec node9 sh -c "fcr fast-retrieve 0 $cid3 . 0" &
docker exec node9 sh -c "fcr fast-retrieve 0 $cid4 . 0" &
docker exec node9 sh -c "fcr fast-retrieve 0 $cid5 . 0" &
docker exec node9 sh -c "fcr fast-retrieve 0 $cid6 . 0" &
docker exec node9 sh -c "fcr fast-retrieve 0 $cid7 . 0" &
docker exec node9 sh -c "fcr fast-retrieve 0 $cid8 . 0" &
# docker exec node9 sh -c "fcr fast-retrieve 0 $cid9 . 0" &

# Wait for all retrievals to finish
wait
echo "Finish all retrievals"
