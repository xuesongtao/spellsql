#!/bin/bash

curPath=$(pwd)

# 执行所有
go test

# 执行 test 下
cd test
go test