#!/usr/bin/env bash

rm -f ./result.log
touch ./result.log
for value in {1..500}
do
	echo $value
	#go test -run 2A | grep FAIL -B 1
	go test -run 2A >> result.log
done
