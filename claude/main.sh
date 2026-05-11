#!/bin/bash


for i in $(seq 1 3600)
do
  ./main > status$(date +%H:%M:%S)
done
