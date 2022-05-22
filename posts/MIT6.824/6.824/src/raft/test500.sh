#!/usr/bin/env bash

rm -f ./2*.log
touch ./2A.log ./2B.log ./2C.log ./2D.log
for value in {1..500}
do
        echo $value
        #go test -run 2A | grep FAIL -B 1
        #go test -run 2A >> ./2A.log
        go test -run 2B >> ./2B.log
        #go test -run 2C >> ./2C.log
        #go test -run 2D >> ./2D.log
done