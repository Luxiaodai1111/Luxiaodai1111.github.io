#!/usr/bin/env bash

for idx in {1..100}
do
  cd raft${idx}
  #grep FAIL 2*
  grep PASS 2*
  cd ..
done

