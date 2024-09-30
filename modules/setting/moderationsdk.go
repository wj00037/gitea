package setting

import (
	"gitee.com/modelers/moderation-server-sdk/httpclient"
)

var MaxTextLen int

// Init is for http client init
func loadModerationFrom(rootCfg ConfigProvider) {
	sec := rootCfg.Section("moderation_server")

	if sec.HasKey("TOKEN_SERVER") {
		cfg := &moderationConfig{
			Token:       sec.Key("TOKEN_SERVER").String(),
			Endpoint:    sec.Key("ENDPOINT_SERVER").String(),
			TokenHeader: sec.Key("TOKEN_HEADER_SERVER").String(),
		}
		httpclient.Init(cfg)
	}
	MaxTextLen = rootCfg.Section("merlin").Key("MAX_TEXT_LEN").MustInt()
}

// Config is for http client config
type moderationConfig = httpclient.Config
