#!/bin/bash

# 只执行 spellsql 部分
go test -timeout 30s -run ^TestNewCacheSql gitee.com/xuesongtao/spellsql -v -count=1
