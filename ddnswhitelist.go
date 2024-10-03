// Package ddns_whitelist dynamic DNS whitelist
//
//revive:disable-next-line:var-naming
//nolint:stylecheck
package ddns_whitelist

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
	cloudflareIP  = "Cf-Connecting-Ip"
)

// Define static error variable.
var (
	errNoHostListProvided = errors.New("no host list provided")
	errEmptyIPAddress     = errors.New("empty IP address")
	errParseIPAddress     = errors.New("could not parse IP address after DNS resolution")
	errParseIPListAddress = errors.New("could not parse IP address from ipList")
)

// Config the plugin configuration.
type Config struct {
	LogLevel string   `json:"logLevel,omitempty"` // Log level (DEBUG, INFO, ERROR)
	HostList []string `json:"hostList,omitempty"` // Add hosts to whitelist
	IPList   []string `json:"ipList,omitempty"`   // Add additional IP addresses to whitelist
}

type allowedIps []*net.IP

// CreateConfig creates the default plugin configuration.
func CreateConfig() *Config {
	return &Config{
		HostList: []string{},
		IPList:   []string{},
	}
}

// ddnswhitelist plugin.
type ddnswhitelist struct {
	config *Config
	name   string
	next   http.Handler
	logger *Logger
}

// New created a new DDNSwhitelist plugin.
func New(_ context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	log := newLogger(config.LogLevel, name, typeName)
	log.Debug("Creating middleware")

	if len(config.HostList) == 0 {
		return nil, errNoHostListProvided
	}

	return &ddnswhitelist{
		name:   name,
		next:   next,
		config: config,
		logger: log,
	}, nil
}

// ServeHTTP ddnswhitelist.
func (a *ddnswhitelist) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	log := a.logger

	// TODO: this might be scheduled and not requested on every request
	// get list of allowed IPs
	aIps, err := newAllowedIps(a.config.HostList, a.config.IPList)
	if err != nil {
		log.Errorf("could not look up ip address: %v", err)
		reject(http.StatusInternalServerError, rw, log)
		return
	}

	reqIPAddr := getRemoteIP(req)
	reqIPAddrLenOffset := len(reqIPAddr) - 1

	for i := reqIPAddrLenOffset; i >= 0; i-- {
		isAllowed, err := aIps.contains(reqIPAddr[i])
		if err != nil {
			log.Errorf("%v", err)
		}

		if !isAllowed {
			log.Infof("request denied [%s]", reqIPAddr[i])
			reject(http.StatusForbidden, rw, log)
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

// getRemoteIP returns a list of IPs that are associated with this request
// from https://github.com/kevtainer/denyip/blob/28930e800ff2b37b692c80d72c883cfde00bde1f/denyip.go#L76-L105
func getRemoteIP(req *http.Request) []string {
	var ipList []string
	var headerIPs []string

	xff := req.Header.Get(xForwardedFor)
	xffs := strings.Split(xff, ",")
	headerIPs = append(headerIPs, xffs...)

	ccip := req.Header.Get(cloudflareIP)
	ccips := strings.Split(ccip, ",")
	headerIPs = append(headerIPs, ccips...)

	for i := len(headerIPs) - 1; i >= 0; i-- {
		headerIPsTrim := strings.TrimSpace(headerIPs[i])

		if len(headerIPsTrim) > 0 {
			ipList = append(ipList, headerIPsTrim)
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

func newAllowedIps(hosts, ips []string) (*allowedIps, error) {
	aIps := &allowedIps{}

	for _, ip := range ips {
		ipAddr := net.ParseIP(ip)
		if ipAddr == nil {
			return nil, fmt.Errorf("%w: %s", errParseIPListAddress, ip)
		}

		*aIps = append(*aIps, &ipAddr)
	}

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

func reject(statusCode int, rw http.ResponseWriter, log *Logger) {
	rw.WriteHeader(statusCode)
	_, err := rw.Write([]byte(http.StatusText(statusCode)))
	if err != nil {
		log.Errorf("could not write response: %v", err)
	}
}
