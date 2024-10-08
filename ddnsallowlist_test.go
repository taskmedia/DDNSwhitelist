// Package ddns_allowlist dynamic DNS allowlist
//
//revive:disable-next-line:var-naming
//nolint:stylecheck
package ddns_allowlist

import (
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCreateConfig(t *testing.T) {
	t.Run("create config", func(t *testing.T) {
		config := CreateConfig()
		assert.NotNil(t, config)
	})
}

func TestNew(t *testing.T) {
	testCases := []struct {
		desc   string
		config *DdnsAllowListConfig
		err    error
	}{
		{
			desc:   "empty sourceRangeHosts",
			config: &DdnsAllowListConfig{},
			err:    errEmptySourceRangeHosts,
		},
		{
			desc: "modified rejectStatus",
			config: &DdnsAllowListConfig{
				SourceRangeHosts: []string{"example.com"},
				RejectStatusCode: 200,
			},
		},
		{
			desc: "invalid rejectStatus",
			config: &DdnsAllowListConfig{
				SourceRangeHosts: []string{"example.com"},
				RejectStatusCode: 999,
			},
			err: fmt.Errorf("%w: %d", errInvalidHTTPStatuscode, 999),
		},
		{
			desc: "lookupInterval",
			config: &DdnsAllowListConfig{
				SourceRangeHosts: []string{"example.com"},
				LookupInterval:   10,
			},
		},
	}

	for _, tc := range testCases {
		// t.Parallel()
		t.Run(tc.desc, func(t *testing.T) {
			dal, err := New(nil, nil, tc.config, "test")

			if err != nil {
				assert.Equal(t, tc.err, err)
				return
			}

			if tc.config.RejectStatusCode != 0 {
				assert.Equal(t, tc.config.RejectStatusCode, dal.(*ddnsAllowLister).rejectStatusCode)
			}

			if tc.config.LookupInterval != 0 {
				expectedInterval := time.Duration(tc.config.LookupInterval) * time.Second
				assert.Equal(t, expectedInterval, dal.(*ddnsAllowLister).lookupInterval)
			}
		})
	}
}

func TestResolveHosts(t *testing.T) {
	testCases := []struct {
		desc            string
		hosts           []string
		expectedHostIPs []string
	}{
		{
			desc:            "no hosts",
			hosts:           []string{},
			expectedHostIPs: []string{},
			// TODO: might check if empty was logged?
		},
		{
			desc:            "localhost",
			hosts:           []string{"localhost"},
			expectedHostIPs: []string{"127.0.0.1"},
		},
		{
			desc:            "single host",
			hosts:           []string{"dns.google"},
			expectedHostIPs: []string{"8.8.4.4", "8.8.8.8"},
		},
		{
			desc:            "multiple hosts",
			hosts:           []string{"dns.google", "cloudflare-dns.com"},
			expectedHostIPs: []string{"104.16.248.249", "104.16.249.249", "8.8.4.4", "8.8.8.8"},
		},
	}

	logger := &Logger{}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			hostIPs := resolveHosts(*logger, tC.hosts)
			sort.Strings(hostIPs)
			assert.Equal(t, tC.expectedHostIPs, hostIPs)
		})
	}
}
