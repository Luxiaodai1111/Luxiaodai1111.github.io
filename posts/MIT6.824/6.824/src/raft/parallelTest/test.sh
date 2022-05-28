#!/usr/bin/env bash

rm -rf ./raft*
mkdir raft
cp -rf ../*.go raft/
for idx in {1..9}
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

cp -rf raft raft10
cd raft10
go test -run 2A > 2A
go test -run 2B > 2B
go test -run 2C > 2C
#go test -run 2D > 2D
#go test -run TestBackup2B > 2B
#go test -run TestFigure8Unreliable2C > 2C
while :
do
  checkFinished=`ps -ef | grep raft | wc -l`
  if [[ $checkFinished == 1 ]]
  then
    break
  else
    sleep 1
  fi
done
cd ..
bash check.sh | grep PASS | wc -l
