package services

import (
	"fmt"
	"log"
	"net"
	"path/filepath"
	"strings"

	"github.com/oschwald/geoip2-golang"
)

type GeoLocation struct {
	Country   string  `json:"country,omitempty"`
	City      string  `json:"city,omitempty"`
	Region    string  `json:"region,omitempty"`
	Latitude  float64 `json:"latitude,omitempty"`
	Longitude float64 `json:"longitude,omitempty"`
}

var (
	cityReader *geoip2.Reader
	asnReader  *geoip2.Reader
)

func InitGeoIP(dbPath string) {
	var err error
	cityReader, err = geoip2.Open(filepath.Join(dbPath, "GeoLite2-City.mmdb"))
	if err != nil {
		log.Printf("GeoLite2-City.mmdb not loaded: %v", err)
	} else {
		log.Println("GeoLite2-City.mmdb loaded successfully")
	}

	asnReader, err = geoip2.Open(filepath.Join(dbPath, "GeoLite2-ASN.mmdb"))
	if err != nil {
		log.Printf("GeoLite2-ASN.mmdb not loaded: %v", err)
	} else {
		log.Println("GeoLite2-ASN.mmdb loaded successfully")
	}
}

func CloseGeoIP() {
	if cityReader != nil {
		cityReader.Close()
	}
	if asnReader != nil {
		asnReader.Close()
	}
}

func GetGeoLocation(ipStr string) GeoLocation {
	var geo GeoLocation
	if cityReader == nil || ipStr == "" {
		return geo
	}

	// Handle forwarded IPs: grab the first valid IP from a comma-separated list
	ips := strings.Split(ipStr, ",")
	clientIP := net.ParseIP(strings.TrimSpace(ips[0]))

	if clientIP == nil {
		return geo
	}

	record, err := cityReader.City(clientIP)
	if err != nil {
		return geo
	}

	if record.Country.IsoCode != "" {
		geo.Country = record.Country.IsoCode
	}
	if record.City.Names["en"] != "" {
		geo.City = record.City.Names["en"]
	}
	if len(record.Subdivisions) > 0 && record.Subdivisions[0].IsoCode != "" {
		geo.Region = record.Subdivisions[0].IsoCode
	}
	geo.Latitude = record.Location.Latitude
	geo.Longitude = record.Location.Longitude

	return geo
}

func GetASN(ipStr string) string {
	if asnReader == nil || ipStr == "" {
		return ""
	}

	ips := strings.Split(ipStr, ",")
	clientIP := net.ParseIP(strings.TrimSpace(ips[0]))
	if clientIP == nil {
		return ""
	}

	record, err := asnReader.ASN(clientIP)
	if err != nil || record.AutonomousSystemOrganization == "" {
		return ""
	}

	return fmt.Sprintf("AS%d %s", record.AutonomousSystemNumber, record.AutonomousSystemOrganization)
}
