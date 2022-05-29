#!/usr/bin/env bash

for idx in {1..500}
do
  result=`bash test.sh`
  echo "test term:" $idx, "result is:" $result
  if [[ $result == 10 ]]
  then
    echo "success"
  else
    echo "failed"
    exit 1
  fi
done
