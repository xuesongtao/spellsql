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

    curPath=$(pwd)
    # gitlab.cd.anpro/kb/module-kb/vxxx
    #替换为具体的版本, 支持 v3, v4
    read -p "输入需要替换的 gitlab.cd.anpro/kb/module-kb/xv$ 的版本 (e.g., 3, 4): " replaceVersion
    if [[ -z "$replaceVersion" ]]; then
        echo "未输入版本号，使用默认版本 4"
        replaceVersion="4"
    fi
    replaceVersion="v$replaceVersion"

    for goFile in $(find . -name "*.go"); do
        # skipFile=$(awk 'BEGIN {print index("'${goFile}'", "benchmark")}') # 不更新的
        # if [[ $skipFile > 0 ]]; then
        #     printf "${goFile} is skip\n"
        #     continue
        # fi
        goFile=${goFile##"./"}
        echo $goFile
        # continue
        # continue
        # gitee.com
        # gitlab.cd.anpro
        targetFile="${targetDir}/${goFile}"
        if [[ ! -d $(dirname $targetFile) ]]; then
            mkdir -p $(dirname $targetFile)
            checkIsOk "mkdir -p $(dirname $targetFile)"
        fi

        # 替换
        sed -e "s/\/\/ \"gitlab.cd.anpro/\"gitlab.cd.anpro/g" \
            -e "s/gitee.com\\/xuesongtao\\/spellsql\\/v2/gitlab.cd.anpro\\/kb\\/module-kb\\/${replaceVersion}\\/spellsql/g" \
            -e "s/logOs \"os\"/\/\/ logOs \"os\"/g" \
            -e "s/log: log.New/\/\/ log: log.New/g" \
            -e "s/d.log/\/\/ d.log/g" \
            -e "s/\/\/ slog/slog/g" \
            -e "s/\/\/ \"errors\"/\"errors\"/g" \
            $goFile >$targetFile
        checkIsOk "repalce: ${goFile} ${targetFile}"
    done

    # 将 readme.md 也移动过去
    cp "README.md" $targetDir
    checkIsOk "cp README.md ${targetDir}"

    rm -rf $tmpDir
    checkIsOk "rm -rf ${tmpDir}"
}

main
