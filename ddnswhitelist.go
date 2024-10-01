// Package ddnswhitelist dynamic DNS whitelist
package ddnswhitelist

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
)

const (
	typeName      = "ddns-whitelist"
	xForwardedFor = "X-Forwarded-For"
)

// Define static error variable.
var (
	errNoHostListProvided = errors.New("no host list provided")
	errEmptyIPAddress     = errors.New("empty IP address")
	errParseIPAddress     = errors.New("could not parse IP address")
)

// Config the plugin configuration.
type Config struct {
	DdnsHostList []string `json:"ddnsHostList,omitempty"` // Add hosts to whitelist
}

type allowedIps []*net.IP

// CreateConfig creates the default plugin configuration.
func CreateConfig() *Config {
	return &Config{
		DdnsHostList: []string{},
	}
}

// DdnsWhitelist plugin.
type DdnsWhitelist struct {
	config *Config
	name   string
	next   http.Handler
}

// New created a new DDNSwhitelist plugin.
func New(_ context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	logger := newLogger("info", name, typeName)
	logger.Debug("Creating middleware")

	if len(config.DdnsHostList) == 0 {
		logger.Error("no host list provided")
		return nil, errNoHostListProvided
	}

	return &DdnsWhitelist{
		name:   name,
		next:   next,
		config: config,
	}, nil
}

// ServeHTTP DDNSwhitelist.
func (a *DdnsWhitelist) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	logger := newLogger("info", a.name, typeName)

	// TODO: this might be scheduled and not requested on every request
	// get list of allowed IPs
	aIps, err := newAllowedIps(a.config.DdnsHostList)
	if err != nil {
		logger.Errorf("could not look up ip address: %v", err)
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	reqIPAddr := a.GetRemoteIP(req)
	reqIPAddrLenOffset := len(reqIPAddr) - 1

	for i := reqIPAddrLenOffset; i >= 0; i-- {
		isAllowed, err := aIps.contains(reqIPAddr[i])
		if err != nil {
			logger.Errorf("%v", err)
		}

		if !isAllowed {
			logger.Infof("request denied [%s]", reqIPAddr[i])
			rw.WriteHeader(http.StatusForbidden)
			return
		}
	}

	a.next.ServeHTTP(rw, req)
}

func (a *allowedIps) contains(ipString string) (bool, error) {
	if len(ipString) == 0 {
		return false, errEmptyIPAddress
	}

	ipAddr := net.ParseIP(ipString)
	if ipAddr == nil {
		return false, fmt.Errorf("%w: %s", errParseIPAddress, ipAddr.String())
	}

	for _, ip := range *a {
		if ip.Equal(ipAddr) {
			return true, nil
		}
	}
	return false, nil
}

// GetRemoteIP returns a list of IPs that are associated with this request
// from https://github.com/kevtainer/denyip/blob/28930e800ff2b37b692c80d72c883cfde00bde1f/denyip.go#L76-L105
func (a *DdnsWhitelist) GetRemoteIP(req *http.Request) []string {
	var ipList []string

	xff := req.Header.Get(xForwardedFor)
	xffs := strings.Split(xff, ",")

	for i := len(xffs) - 1; i >= 0; i-- {
		xffsTrim := strings.TrimSpace(xffs[i])

		if len(xffsTrim) > 0 {
			ipList = append(ipList, xffsTrim)
		}
	}

	ip, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		remoteAddrTrim := strings.TrimSpace(req.RemoteAddr)
		if len(remoteAddrTrim) > 0 {
			ipList = append(ipList, remoteAddrTrim)
		}
	} else {
		ipTrim := strings.TrimSpace(ip)
		if len(ipTrim) > 0 {
			ipList = append(ipList, ipTrim)
		}
	}

	return ipList
}

func newAllowedIps(hosts []string) (*allowedIps, error) {
	aIps := &allowedIps{}

	for _, host := range hosts {
		ip, err := net.LookupIP(host)
		if err != nil {
			return nil, err
		}

		for _, i := range ip {
			iCopy := i
			*aIps = append(*aIps, &iCopy)
		}
	}

	return aIps, nil
}
