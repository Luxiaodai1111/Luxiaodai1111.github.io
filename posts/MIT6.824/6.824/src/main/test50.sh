#!/usr/bin/env bash

for value in {1..50}
do
	echo $value
	bash test-mr.sh | grep FAIL -B 1
done
