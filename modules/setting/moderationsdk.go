package setting

import (
	"github.com/openmerlin/moderation-service-sdk/httpclient"
)

var MaxTextLen int

// Init is for http client init
func loadModerationFrom(rootCfg ConfigProvider) {
	sec := rootCfg.Section("moderation_service")

	if sec.HasKey("TOKEN") {
		cfg := &moderationConfig{
			Token:       sec.Key("TOKEN").String(),
			Endpoint:    sec.Key("ENDPOINT").String(),
			TokenHeader: sec.Key("TOKEN_HEADER").String(),
		}
		httpclient.Init(cfg)
	}
	MaxTextLen = rootCfg.Section("merlin").Key("MAX_TEXT_LEN").MustInt()
}

// Config is for http client config
type moderationConfig = httpclient.Config
