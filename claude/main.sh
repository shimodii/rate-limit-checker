#!/bin/bash


for i in $(seq 1 3600)
do
  ./main > log/log-$(date +%H:%M:%S)
done
