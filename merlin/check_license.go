package merlin

import (
	"bytes"
	"encoding/base64"
	"errors"
	"reflect"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/parser"
	"go.abhg.dev/goldmark/frontmatter"
)

type FrontMatter struct {
	License  interface{} `yaml:"license"`
	licenses []string
}

func (m *FrontMatter) setLicenses() error {
	switch reflect.TypeOf(m.License).Kind() {
	case reflect.Slice, reflect.Array:
		for _, v := range m.License.([]interface{}) {
			if str, ok := v.(string); ok {
				m.licenses = append(m.licenses, str)
			}
		}

	case reflect.String:
		m.licenses = []string{m.License.(string)}

	default:
		return errors.New("invalid license type")
	}
	return nil
}

func CheckLicense(content string) error {
	license, err := parseLicense(content)
	if err != nil {
		return err
	}

	if len(license) == 0 {
		return errors.New("invalid license")
	}

	if len(validLicense) == 0 {
		initConfig()
	}

	for _, v := range license {
		if !validLicense.Has(v) {
			return errors.New("invalid license")
		}

	}

	return nil
}

func parseLicense(content string) ([]string, error) {
	b, err := base64.StdEncoding.DecodeString(content)
	if err != nil {
		return nil, err
	}

	md := goldmark.New(
		goldmark.WithExtensions(&frontmatter.Extender{}),
	)

	ctx := parser.NewContext()
	var buf bytes.Buffer
	if err = md.Convert(b, &buf, parser.WithContext(ctx)); err != nil {
		return nil, err
	}

	data := frontmatter.Get(ctx)
	if data == nil {
		return nil, nil
	}

	var meta FrontMatter
	if err := data.Decode(&meta); err != nil {
		return nil, err
	}
	if err := meta.setLicenses(); err != nil {
		return nil, err
	}
	return meta.licenses, nil
}
