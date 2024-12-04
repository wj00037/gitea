package setting

// Otel settings
type OtelConfig struct {
	Enabled   bool
	Endpoint  string
	Name      string
	Fractions float64
}

var Otel = OtelConfig{}

func loadOtelFrom(rootCfg ConfigProvider) {
	sec := rootCfg.Section("otel")
	Otel.Enabled = sec.Key("ENABLED").MustBool(false)
	if Otel.Enabled {
		Otel.Endpoint = sec.Key("ENDPOINT").String()
		Otel.Name = sec.Key("NAME").MustString("gitea")
		Otel.Fractions = sec.Key("FRACTIONS").MustFloat64()
	}
}
