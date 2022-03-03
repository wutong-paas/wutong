#!/bin/sh
pull_number=$(jq --raw-output .pull_request.number "$GITHUB_EVENT_PATH")

URL="https://api.github.com/repos/wutong-paas/wutong/pulls/${pull_number}/files"

# 请求 GitHub api 接口，解析变更的文件
# 这里用 jq 过滤了部分文件
CHANGED_MARKDOWN_FILES=$(curl -s -X GET -G $URL | jq -r '.[] | select(.status != "removed") | select(.filename | endswith(".go")) | .filename')
for file in ${CHANGED_MARKDOWN_FILES}; do
  echo "golint ${file}"
  golint -set_exit_status=true ${file} || exit 1
done

echo "code golint check success"