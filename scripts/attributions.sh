#!/bin/bash

REPO_DIR="$( cd "$(dirname "$( dirname "${BASH_SOURCE[0]}" )")" &> /dev/null && pwd )"

GO_LICENCE_DETECTOR=''
NOTICE_FILE=''

while getopts c:b:n:g: flag
do
  case "${flag}" in
    c) components=${OPTARG};; #e.g. reciever/nopreceiver
    b) GO_LICENCE_DETECTOR=${OPTARG};;
    n) NOTICE_FILE=${OPTARG};;
    g) GO=${OPTARG};;
    *) exit 1;;
  esac
done

[[ -n "$NOTICE_FILE" ]] || NOTICE_FILE='THIRD_PARTY_NOTICES.md'

[[ -n "$GO_LICENCE_DETECTOR" ]] || GO_LICENCE_DETECTOR='go-licence-detector'

if [[ -z $components ]]; then
  echo "List of components to build not provided. Use '-c' to specify the names of the components to generate attributions for. Ex.:"
  echo "$0 -c reciever/nopreceiver"s
  exit 1
fi

for component in $(echo "$components" | tr "," "\n")
do
  pushd "${REPO_DIR}/${component}" > /dev/null || exit

  echo "📜 Building notice for ${distribution}..."

  ${GO} list -mod=mod -m -json all | ${GO_LICENCE_DETECTOR} \
    -rules "${REPO_DIR}/internal/assets/license/rules.json" \
    -noticeTemplate "${REPO_DIR}/internal/assets/license/THIRD_PARTY_NOTICES.md.tmpl" \
    -noticeOut "${REPO_DIR}/${component}/${NOTICE_FILE}"

  popd > /dev/null || exit
done
