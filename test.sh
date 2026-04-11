#!/bin/bash

# 执行所有
go test -v -run ^Test > out.log

# 执行 test 下
cd test
go test
