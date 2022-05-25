#!/usr/bin/env bash

rm -rf ./result
mkdir -p ./result
for idx in {1..500}
do
        #echo 2A-${idx} ; go test -run 2A > ./result/2A-${idx}
        #echo 2B-${idx} ; go test -run 2B > ./result/2B-${idx}
        echo 2B-${idx} ; go test -run TestBackup2B > ./result/2B-${idx}
        echo 2C-${idx} ; go test -run TestFigure8Unreliable2C > ./result/2C-${idx}
        #echo 2C-${idx} ; go test -run 2C > ./result/2C-${idx}
        #echo 2D-${idx} ; go test -run 2D > ./result/2D-${idx}
done