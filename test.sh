#!/bin/bash

# 执行所有
go test -v -run ^Test > out.log

# 执行 test 下
pushd test
go test -v -run ^Test >> ../out.log
popd

pushd builder
go test -v -run ^Test >> ../out.log
popd

go test -coverprofile=cover.out ./...
go tool cover -html=cover.out
