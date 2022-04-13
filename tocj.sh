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
    # $3 orm.go
    # $4 orm_test.go

    # 更换包名 > 取消 glog 包注释 > 注释 cjLog. > 取消注释 glog.
    cd $curPath
    checkIsOk "cd ${curPath}"

    # 处理 getsqlstr.go
    sed -e "s/package spellsql/package mysql/g" \
        -e "s/\/\/ \"github.com/\"github.com/g" \
        -e "s/cjLog./\/\/ cjLog./g" \
        -e "s/\/\/ glog./glog./g" \
        spellsql.go >$1
    checkIsOk "update getsqlstr.go"
    # 处理 getsqlstr_test.go
    sed -e "s/package spellsql/package mysql/g" spellsql_test.go >$2
    checkIsOk "update getsqlstr_test.go"

    # 处理 orm.go
    sed -e "s/package spellsql/package mysql/g" \
        -e "s/\/\/ \"github.com/\"github.com/g" \
        -e "s/cjLog./\/\/ cjLog./g" \
        -e "s/\/\/ glog./glog./g" \
        orm.go >$3
    checkIsOk "update orm.go"
    # 处理 orm_test.go
    sed -e "s/package spellsql/package mysql/g" \
        -e "s/_ \"github.com/\/\/ _ \"github.com/g" \
        -e "s/\/\/ db=Db/db = Db/g" \
        -e "s/InitMyDb(1)/\/\/ InitMyDb(1)/g" \
        orm_test.go >$4
    checkIsOk "update orm_test.go"
}

function gitHandle() {
    # $1 项目路径

    projectDir=$1
    cd $projectDir
    checkIsOk "cd ${projectDir}"

    git pull
    checkIsOk "git pull"

    targetDir="app/model/mysql"
    waitGitAdd="${targetDir}/getsqlstr.go ${targetDir}/getsqlstr_test.go ${targetDir}/orm.go ${targetDir}/orm_test.go"
    git add $waitGitAdd
    checkIsOk "git add ${waitGitAdd}"

    git commit -m "update getsqlstr"
    checkIsOk "git commit"

    git push
    checkIsOk "git push"
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
    goFilePathDir="${projectDir}/${goFilePathDir}"
    updateCjGoFile \
        "${goFilePathDir}/getsqlstr.go" \
        "${goFilePathDir}/getsqlstr_test.go" \
        "${goFilePathDir}/orm.go" \
        "${goFilePathDir}/orm_test.go"
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
        sleep 1
        startHandle $goFile
    done
}

main
# main "/Users/xuesongtao/goProject/src/workGo/aesm/appside_server/app/model/mysql/getsqlstr.go"
# main "/Users/xuesongtao/goProject/src/workGo/aesm/appside_server/app/model/mysql/getsqlstr_test.go"
# main "/Users/xuesongtao/goProject/src/workGo/aesm/appside_server/app/model/mysql/orm.go"
# main "/Users/xuesongtao/goProject/src/workGo/aesm/appside_server/app/model/mysql/orm_test.go"
