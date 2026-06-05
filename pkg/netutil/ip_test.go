package netutil_test

import (
	"testing"

	"github.com/supuwoerc/gapi-server/pkg/netutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOutboundIP(t *testing.T) {
	ip, err := netutil.OutboundIP()
	t.Logf("outbound ip is: %v", ip)
	require.NoError(t, err)
	assert.NotNil(t, ip)
	assert.False(t, ip.IsLoopback(), "outbound IP should not be loopback")
	assert.False(t, ip.IsUnspecified(), "outbound IP should not be 0.0.0.0")
}
