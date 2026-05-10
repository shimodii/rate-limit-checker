#!/bin/bash


URL="https://divar.ir"
TRESHHOLD=200

for i in {1..$TRESHHOLD}
do
  echo "===================================" >> output
  echo "attempt $i" >> output
  echo "" >> output
  curl $URL >> output
  echo ""
  echo "===================================" >> output
  echo ""

done
