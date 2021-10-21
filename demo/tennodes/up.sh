#!/bin/bash

docker compose rm -f
docker volume rm tennodes_lotus-volume
docker compose up
