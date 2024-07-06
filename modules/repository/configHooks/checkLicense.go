/*
Copyright (c) Huawei Technologies Co., Ltd. 2024. All rights reserved
*/

// Package configHooks for check license

package configHooks

import (
	"fmt"
	"strings"
	"time"

	"code.gitea.io/gitea/modules/setting"
)

// CheckLicense for check license
type CheckLicense struct {
	Name    string
	Content string
}

// GetHookName for check license name
func (c CheckLicense) GetHookName() string {
	return c.Name
}

// GetHookContent for check license content
func (c CheckLicense) GetHookContent() string {
	license := setting.CfgProvider.Section("merlin").Key("LICENSE").Strings(",")
	if len(license) > 0 {
		shellLicense := strings.Join(license, " ")
		c.Content = fmt.Sprintf(`
valid_licenses=(%s)

log_error() {
  echo "%s [ERROR] $*" > /proc/1/fd/1
}

log_operation() {
  echo "%s | %s | %s | %s | %s | $*" > /proc/1/fd/1
}

while read oldrev newrev _; do
	files=$(git diff --name-only $oldrev $newrev)
  if echo "$files" | grep -q "README.md"; then
		readme_content=$(git show $newrev:README.md)
		license=$(echo "$readme_content" | grep -ozP -m 1 "license:\s*\K\S+" | tr -d '\0')
		if [[ ${license} = "-" ]]; then
			license=$(echo "$readme_content" | grep -ozP -m 1 "license:\s*\K(\s+-.+\n{1})+"| tr -d '\0')
			arr=($(echo "$license" | awk -F ' - ' '{a[NR]=$2}END{for(i in a) print a[i]}'))
			for i in "${!arr[@]}"; do
				if [[ ! " ${valid_licenses[@]} " =~ " ${arr[$i]} " ]]; then
					echo "Sorry, your push was rejected during YAML metadata verification:"
					echo " - Error: "license" must be one of (${valid_licenses[@]})"
					log_error "Sorry, your push was rejected during YAML metadata verification:"
					log_error " - Error: "license" must be one of (${valid_licenses[@]})"
					log_operation "license check | failed"
					exit 1
				fi
			done
			echo "License field is valid. Proceeding with the push."
			log_operation "license check | success"
		elif [[ "$license" =~ ^\[ ]]; then
			license=$(echo "$readme_content" | grep -ozP -m 1 "license:\s*\K\[.*?\]")
			license=$(echo "$license" | tr -d '[]')
			license=$(echo "$license" | tr -d ',')
			arr=($license)
            for item in "${arr[@]}"; do
				if [[ ! " ${valid_licenses[@]} " =~ " $item " ]]; then
			        echo "Sorry, your push was rejected during YAML metadata verification:"
                    echo " - Error: "license" must be one of (${valid_licenses[@]})"
                    log_error "Sorry, your push was rejected during YAML metadata verification:"
                    log_error " - Error: "license" must be one of (${valid_licenses[@]})"
                    log_operation "license check | failed"
                    exit 1
			    fi
			done
			echo "License field is valid. Proceeding with the push."
			log_operation "license check | success"
		elif [[ " ${valid_licenses[@]} " =~ " ${license} " ]]; then
				echo "License field is valid. Proceeding with the push."
				log_operation "license check | success"
		else
				echo "Sorry, your push was rejected during YAML metadata verification:"
				echo " - Error: "license" must be one of (${valid_licenses[@]})"
				log_error "Sorry, your push was rejected during YAML metadata verification:"
				log_error " - Error: "license" must be one of (${valid_licenses[@]})"
				log_operation "license check | failed"
				exit 1
		fi
  fi
done
`, shellLicense,
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
