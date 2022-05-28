#!/usr/bin/env bash

for idx in {1..10}
do
  cd raft${idx}
  grep -w PASS 2*
  cd ..
done

