#!/bin/bash

function checkIsOk() {
    # $1 操作名

    if [[ $? > 0 ]]; then
        echo "${1} is failed"
        exit 1
    fi
    echo "${1} is success"
}

function updateSqlStr() {
    # $1 目标文件

    # 更换包名 > 注释 log > 取消 glog 注释 > 注释 log. > 取消注释 glog.
    sed -e "s/package spellsql/package mysql/g" \
        -e "s/\"log\"/\/\/ \"log\"/g" \
        -e "s/\/\/ \"github.com/\"github.com/g" \
        -e "s/log.P/\/\/ log.P/g" \
        -e "s/\/\/ glog.I/glog.I/g" \
        spellsql.go >$1
}

function updateTestSqlStr() {
    # $1 目标文件

    # 更换包名
    sed -e "s/package spellsql/package mysql/g" spellsql_test.go >$1
}

function gitHandle() {
    # $1 项目路径

    cd $projectDir && checkIsOk "cd"
    git pull && checkIsOk "git pull"
    git add . && checkIsOk "git add"
    git commit -m "update getsqlstr" && checkIsOk "git commit"
    git push && checkIsOk "git push"
}

function handle() {
    # $1 gofile

    goFile=$1
    # 解析项目路径
    projectDirIndex=$(echo $goFile | awk 'BEGIN {print index("'$goFile'", "/app/model/mysql")}')
    projectDir=${goFile:0:$projectDirIndex-1}
    printf "======= 开始处理: %s ===========" $projectDir
    isTestFile=$(echo $goFile | awk 'BEGIN {print index("'$goFile'", "_test.go")}')
    if [[ $isTestFile > 0 ]]; then
        updateTestSqlStr $goFile && checkIsOk "updateTestSqlStr"
        continue
    fi
    updateSqlStr $goFile && checkIsOk "updateSqlStr"
    gitHandle $projectDir
    printf "======= 处理成功: %s ===========" $projectDir
}

function main() {
    # $1 gofile

    if [[ $1 != "" ]]; then
        handle $1
        return
    fi
    for goFile in $(find "/Users/xuesongtao/goProject/src/workGo" -name "getsqlstr*"); do
        handle $goFile
    done
}

# main
# updateSqlStr "/Users/xuesongtao/goProject/src/workGo/aesm/bside_server/app/model/mysql/getsqlstr.go"
main "/Users/xuesongtao/goProject/src/workGo/aesm/app_server/app/model/mysql/getsqlstr.go"
# updateTestSqlStr "/Users/xuesongtao/goProject/src/workGo/aesm/bside_server/app/model/mysql/getsqlstr_test.go"
# gitHandle "/Users/xuesongtao/goProject/src/workGo/aesm/bside_server/app/model/mysql/getsqlstr.go"
