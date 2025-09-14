package utils

import (
	"errors"
	"net"
	"net/netip"

	"github.com/oschwald/maxminddb-golang"
)

type IP2CountryResolver struct {
	v4 *maxminddb.Reader
	v6 *maxminddb.Reader
}

var IP2Country *IP2CountryResolver

// NewIP2CountryResolver opens separate MMDBs for IPv4 and IPv6.
// Pass "" for a family you don't have.
func NewIP2CountryResolver(v4Path, v6Path string) error {
	if v4Path == "" && v6Path == "" {
		return errors.New("must provide at least one mmdb path")
	}
	var (
		v4, v6 *maxminddb.Reader
		err    error
	)
	if v4Path != "" {
		if v4, err = maxminddb.Open(v4Path); err != nil {
			return err
		}
	}
	if v6Path != "" {
		if v6, err = maxminddb.Open(v6Path); err != nil {
			if v4 != nil {
				_ = v4.Close()
			}
			return err
		}
	}
	IP2Country = &IP2CountryResolver{v4: v4, v6: v6}
	return nil
}

func (r *IP2CountryResolver) Close() error {
	var e1, e2 error
	if r.v4 != nil {
		e1 = r.v4.Close()
	}
	if r.v6 != nil {
		e2 = r.v6.Close()
	}
	if e1 != nil {
		return e1
	}
	return e2
}

// record shapes for different vendors
type mmCountry struct {
	Country struct {
		ISOCode string            `maxminddb:"iso_code"`
		Names   map[string]string `maxminddb:"names"`
	} `maxminddb:"country"`
}
type sapicsCountry struct {
	// Sapics often exposes a flat code as well
	CountryCode string `maxminddb:"country_code"`
	// but sometimes also nested
	Country struct {
		ISOCode string            `maxminddb:"iso_code"`
		Names   map[string]string `maxminddb:"names"`
	} `maxminddb:"country"`
}

// CountryCode returns ISO alpha-2 (e.g. "US") and English name (if present).
// Returns ("","",nil) for private/loopback/link-local/multicast/unspecified or
// not found.
func (r *IP2CountryResolver) CountryCode(ipstr string) (string, string, error) {
	if r == nil {
		return "", "", errors.New("resolver is nil")
	}
	addr, err := netip.ParseAddr(ipstr)
	if err != nil {
		return "", "", err
	}
	if addr.IsPrivate() || addr.IsLoopback() || addr.IsLinkLocalUnicast() ||
		addr.IsMulticast() || addr.IsUnspecified() {
		return "", "", nil
	}

	var reader *maxminddb.Reader
	switch {
	case addr.Is4():
		reader = r.v4
	case addr.Is6():
		reader = r.v6
	default:
		return "", "", errors.New("unknown IP family")
	}
	if reader == nil {
		return "", "", nil
	}

	ip := net.IP(addr.AsSlice())

	// Try MaxMind/DB-IP style first
	var mm mmCountry
	if err := reader.Lookup(ip, &mm); err == nil &&
		mm.Country.ISOCode != "" {
		return mm.Country.ISOCode, mm.Country.Names["en"], nil
	}

	// Fallback to Sapics style
	var sp sapicsCountry
	if err := reader.Lookup(ip, &sp); err == nil {
		if sp.Country.ISOCode != "" {
			return sp.Country.ISOCode, sp.Country.Names["en"], nil
		}
		if sp.CountryCode != "" {
			return sp.CountryCode, "", nil
		}
	}

	// Not found
	return "", "", nil
}
