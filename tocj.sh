#!/bin/bash

curPath=$(pwd)
echo "curPath=" $curPath

function checkIsOk() {
    # $1 操作名

    if [[ $? > 0 ]]; then
        echo "[${1}] is failed"
        exit 1
    fi
    echo "[${1}] is success"
}

function updateCjGoFile() {
    # $1 getsqlstr.go
    # $2 getsqlstr_test.go

    # 更换包名 > 注释 log > 取消 glog 注释 > 注释 log. > 取消注释 glog.
    cd $curPath && checkIsOk "cd ${curPath}"

    # 处理 getsqlstr.go

    sed -e "s/package spellsql/package mysql/g" \
        -e "s/\"log\"/\/\/ \"log\"/g" \
        -e "s/\/\/ \"github.com/\"github.com/g" \
        -e "s/log.P/\/\/ log.P/g" \
        -e "s/\/\/ glog.I/glog.I/g" \
        spellsql.go >$1 && checkIsOk "update getsqlstr.go"

    # 处理 getsqlstr_test.go
    sed -e "s/package spellsql/package mysql/g" spellsql_test.go >$2 && checkIsOk "update getsqlstr_test.go"
}

function gitHandle() {
    # $1 项目路径

    projectDir=$1
    cd $projectDir && checkIsOk "cd ${projectDir}"
    git pull && checkIsOk "git pull"
    git add . && checkIsOk "git add"
    git commit -m "update getsqlstr" && checkIsOk "git commit"
    git push && checkIsOk "git push"
}

function startHandle() {
    # $1 gofile

    goFile=$1
    goFilePathDir="/app/model/mysql"
    # 解析项目路径
    projectDirIndex=$(echo $goFile | awk 'BEGIN {print index("'$goFile'", "'$goFilePathDir'")}')
    if [[ $projectDirIndex < 1 ]]; then
        echo "parse projectDirIndex is failed"
        return
    fi

    projectDir=${goFile:0:$projectDirIndex}
    printf "======= 开始处理: %s ===========\n" $projectDir
    updateCjGoFile "${projectDir}/${goFilePathDir}/getsqlstr.go" "${projectDir}/${goFilePathDir}/getsqlstr_test.go"
    gitHandle $projectDir
    printf "======= 处理成功: %s ===========\n" $projectDir
}

function main() {
    # $1 gofile 有值只处理此项目中的内容 getsqlstr.go 和 getsqlstr_test.go

    if [[ $1 != "" ]]; then
        startHandle $1
        return
    fi

    for goFile in $(find "/Users/xuesongtao/goProject/src/workGo" -name "getsqlstr.go"); do
        sleep 1s
        startHandle $goFile
    done
}

main
# main "/Users/xuesongtao/goProject/src/workGo/aesm/appside_server/app/model/mysql/getsqlstr.go"
# main "/Users/xuesongtao/goProject/src/workGo/aesm/appside_server/app/model/mysql/getsqlstr_test.go"
