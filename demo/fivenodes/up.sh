#!/bin/bash

docker compose rm -f
docker volume fivenodes_lotus-volume
docker compose up
