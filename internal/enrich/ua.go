package enrich

import (
	"sync"

	"github.com/openpanel-dev/openpanel-api/internal/models"
	"github.com/ua-parser/uap-go/uaparser"
)

var (
	parser *uaparser.Parser
	once   sync.Once
)

func InitUAParser() error {
	var err error
	once.Do(func() {
		// uap-go includes compiled definitions by default.
		// uaparser.New() uses these internal definitions.
		parser, err = uaparser.New()
	})
	return err
}

func GetUAInfo(ua string) models.UAInfo {
	if parser == nil {
		return models.UAInfo{}
	}

	client := parser.Parse(ua)

	return models.UAInfo{
		OS:             client.Os.Family,
		OSVersion:      formatVersion(client.Os.Major, client.Os.Minor, client.Os.Patch),
		Browser:        client.UserAgent.Family,
		BrowserVersion: formatVersion(client.UserAgent.Major, client.UserAgent.Minor, client.UserAgent.Patch),
		Device:         client.Device.Family,
		Brand:          client.Device.Brand,
		Model:          client.Device.Model,
	}
}

func formatVersion(major, minor, patch string) string {
	if major == "" {
		return ""
	}
	version := major
	if minor != "" {
		version += "." + minor
	}
	if patch != "" {
		version += "." + patch
	}
	return version
}
