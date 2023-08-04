#!/bin/bash

function checkIsOk() {
    # $1 操作名

    if [[ $? > 0 ]]; then
        echo -e "\"${1}\" is \033[1;31mFailed\033[0m"
        exit 1
    fi
    echo -e "\"${1}\" is \033[1;32mSuccess\033[0m"
}

function main() {
    targetDir="${XJ_COMMON_DIR}/spellsql"
    if [[ ! -d $targetDir ]]; then
        mkdir -p $targetDir
        checkIsOk "mkdir -p ${targetDir}"
    fi

    tmpDir="spellsql"
    if [[ ! -d $tmpDir ]]; then
        mkdir -p $tmpDir
    fi
    cp ./* $tmpDir

    for goFile in $(find $tmpDir -name "*.go"); do
        # skipFile=$(awk 'BEGIN {print index("'${goFile}'", "benchmark")}') # 不更新的
        # if [[ $skipFile > 0 ]]; then
        #     printf "${goFile} is skip\n"
        #     continue
        # fi

        # gitee.com
        # gitlab.cd.anpro
        sed -e "s/\/\/ \"gitlab.cd.anpro/\"gitlab.cd.anpro/g" \
            -e "s/\"gitee.com\\/xuesongtao\\/spellsql\\/test\"/\/\/ \"gitee.com\\/xuesongtao\\/spellsql\\/test\"/g" \
            $goFile >"${XJ_COMMON_DIR}/${goFile}"
        checkIsOk "repalce: $goFile ${targetDir}"
    done

    # 将 readme.md 也移动过去
    cp "README.md" $targetDir
    checkIsOk "cp README.md ${targetDir}"

    rm -rf $tmpDir
    checkIsOk "rm -rf ${tmpDir}"
}

main
