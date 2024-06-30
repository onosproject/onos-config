#!/bin/bash
# SPDX-License-Identifier: Apache-2.0
# Copyright 2024 Intel Corporation

set +x

# input should be all, is_valid_format, is_dev, and is_unique
INPUT=$1

function is_valid_format() {
    # check if version format is matched to SemVer
    VER_REGEX='^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$'
    if [[ ! $(cat VERSION | tr -d '\n' | sed s/-dev//) =~ $VER_REGEX ]]
    then
      return 1
    fi
    return 0
}

function is_dev_version() {
    # check if version has '-dev'
    # if there is, no need to check version
    if [[ $(cat VERSION | tr -d '\n' | tail -c 4) =~ "-dev" ]]
    then
      return 0
    fi
    return 1
}

function is_unique_version() {
    # check if the version is already tagged in GitHub repository
    for t in $(git tag | cat)
    do
      if [[ $t == $(echo v$(cat VERSION | tr -d '\n')) ]]
      then
        return 1
      fi
    done
    return 0
}

case $INPUT in
  all)
    is_valid_format
    f_valid=$?
    if [[ $f_valid == 1 ]]
    then
        echo "ERROR: Version $(cat VERSION) is not in SemVer format"
        exit 2
    fi

    is_dev_version
    f_dev=$?
    if [[ $f_dev == 0 ]]
    then
        echo "This is dev version"
        exit 0
    fi

    is_unique_version
    f_unique=$?
    if [[ $f_unique == 1 ]]
    then
        echo "ERROR: duplicated tag $(cat VERSION)"
        exit 2
      fi
    ;;

  is_valid_format)
    is_valid_format
    ;;

  is_dev)
    is_dev_version
    f_dev=$?
    if [[ $f_dev == 0 ]]
    then
        echo "true"
        exit 0
    fi
    echo "false"
    ;;

  is_unique)
    is_unique_version
    ;;

  *)
    echo -n "unknown input"
    exit 2
    ;;

esac
