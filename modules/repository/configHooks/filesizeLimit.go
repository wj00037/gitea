/*
Copyright (c) Huawei Technologies Co., Ltd. 2024. All rights reserved
*/

// Package configHooks for file size limit

package configHooks

import (
	"fmt"
	"time"

	"code.gitea.io/gitea/modules/setting"
)

// const
const (
	prefix     = "MF_OPERATION_LOG"
	timeLayout = "2006-01-02 15:04:05"
	user       = "git-client"
	ip         = "local"
	method     = "commit"
)

// FileSizeLimit for file size limit
type FileSizeLimit struct {
	Name    string
	Content string
}

// GetHookName for file size limit name
func (c FileSizeLimit) GetHookName() string {
	return c.Name
}

// GetHookContent for file size limit content
func (c FileSizeLimit) GetHookContent() string {
	if setting.CommonMaxFileSize > 0 {
		c.Content = fmt.Sprintf(`
max_size=%d

log_error() {
  echo "%s [ERROR] $*" > /proc/1/fd/1
}

log_operation() {
  echo "%s | %s | %s | %s | %s | $*" > /proc/1/fd/1
}

while read oldrev newrev _; do
  if [[ "$oldrev" == "0000000000000000000000000000000000000000" ]]; then
    files=$(git ls-tree --name-only ${newrev})
    for file in $files; do
      size=$(git cat-file -s ${newrev}:${file})
      if [[ ${size} -gt ${max_size} ]]; then
		    echo "The size of each file should be within $((max_size / 1048576))MB."
				log_error "The size of each file should be within $((max_size / 1048576))MB."
				log_operation "filesize check | failed"
		    exit 1
	    fi
	  done
  else
    changes=$(git rev-list ${oldrev}..${newrev})

    for commit in ${changes}; do
      files=$(git diff-tree --no-commit-id --name-only -r ${commit})

      for file in $files; do
				if echo $(git ls-tree --name-only -r $newrev) | grep -q ${file}; then
					size=$(git cat-file -s ${commit}:${file})
					if [[ ${size} -gt ${max_size} ]]; then
						echo "The size of each file should be within $((max_size / 1048576))MB."
						log_error "The size of each file should be within $((max_size / 1048576))MB."
						log_operation "filesize check | failed"
						exit 1
					fi
				fi
      done
    done
  fi
done

log_operation "filesize check | success"
`, setting.CommonMaxFileSize,
			time.Now().Format(timeLayout),
			prefix,
			time.Now().Format(timeLayout),
			user,
			ip,
			method,
		)
	}
	return c.Content
}
