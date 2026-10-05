package main

import (
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewHTTPServer_SetsTimeouts: the server main starts carries all four
// timeouts (C015); a zero would mean no limit.
func TestNewHTTPServer_SetsTimeouts(t *testing.T) {
	handler := http.NewServeMux()
	server := newHTTPServer("127.0.0.1:8080", handler)

	assert.Equal(t, "127.0.0.1:8080", server.Addr)
	assert.Same(t, handler, server.Handler)
	assert.Equal(t, 10*time.Second, server.ReadHeaderTimeout)
	assert.Equal(t, 2*time.Minute, server.ReadTimeout)
	assert.Equal(t, 5*time.Minute, server.WriteTimeout)
	assert.Equal(t, 2*time.Minute, server.IdleTimeout)
}

// TestNewHTTPServer_WriteTimeoutMatchesNginx: WriteTimeout is nginx's
// proxy_read_timeout on every host, so neither side cuts a slow response
// first. Raise both together.
func TestNewHTTPServer_WriteTimeoutMatchesNginx(t *testing.T) {
	nginxFileList, err := filepath.Glob("../../../../infra/*/etc/nginx_*")
	require.NoError(t, err)
	require.NotEmpty(t, nginxFileList)

	pattern := regexp.MustCompile(`proxy_read_timeout\s+(\d+)s;`)
	server := newHTTPServer("127.0.0.1:8080", http.NewServeMux())
	for _, nginxFile := range nginxFileList {
		source, err := os.ReadFile(nginxFile)
		require.NoError(t, err)
		matchList := pattern.FindAllStringSubmatch(string(source), -1)
		require.NotEmpty(t, matchList, "%s: no proxy_read_timeout", nginxFile)
		for _, match := range matchList {
			seconds, err := strconv.Atoi(match[1])
			require.NoError(t, err)
			assert.Equal(t, server.WriteTimeout,
				time.Duration(seconds)*time.Second,
				"%s: proxy_read_timeout", nginxFile)
		}
	}
}
