#!/usr/bin/env bash

for idx in {1..50}
do
  result=`bash test.sh`
  echo $result
  if [[ $result == 300 ]]
  then
    echo "success"
  else
    echo "failed"
    exit 1
  fi
done
