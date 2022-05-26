#!/usr/bin/env bash

rm -rf ./raft*
mkdir raft
cp -rf ../*.go raft/
for idx in {1..99}
do
  cp -rf raft raft${idx}
  cd raft${idx}
  go test -run 2A > 2A &
  go test -run 2B > 2B &
  go test -run 2C > 2C &
#  go test -run 2D > 2D &
#  go test -run TestBackup2B > 2B &
#  go test -run TestFigure8Unreliable2C > 2C &
  cd ..
done

cp -rf raft raft100
cd raft100
go test -run 2A > 2A
go test -run 2B > 2B
go test -run 2C > 2C
#go test -run 2D > 2D
#go test -run TestBackup2B > 2B
#go test -run TestFigure8Unreliable2C > 2C
sleep 120
cd ..
bash check.sh | grep PASS | wc -l
