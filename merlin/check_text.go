package merlin

import (
	"errors"
	"strings"
	"unicode/utf8"

	moderationsdk "code.gitea.io/gitea/modules/setting"
	"gitee.com/modelers/moderation-server-sdk/moderation"
	moderationapi "gitee.com/modelers/moderation-server-sdk/moderation/api"
)

const templateText = "一一一一一一一一一一"

func CheckText(filenames []string, commitMessage string) (err error) {
	if checkTextLen(commitMessage) {
		return errors.New("text length should be less than 1500")
	}
	var file strings.Builder
	file.WriteString(commitMessage)
	for _, filename := range filenames {
		if checkTextLen(file.String() + filename + templateText) {
			if err = moderationText(file.String()); err != nil {
				return
			}
			file.Reset()
			file.WriteString(filename)
		} else {
			file.WriteString(templateText)
			file.WriteString(filename)
		}
	}
	return moderationText(file.String())
}

func moderationText(content string) (err error) {
	req := moderation.ReqToModerationText{
		Text:    content,
		Type:    "comment",
		BizType: "test",
	}
	var res moderation.RespModeration
	res, _, err = moderationapi.ModerationText(&req)
	if err != nil {
		return
	}
	if res.Result != "pass" {
		return errors.New("Sensitive data found in submission, Please remove it and try again.")
	}
	return
}

func checkTextLen(text string) bool {
	return utf8.RuneCountInString(text) > moderationsdk.MaxTextLen
}
