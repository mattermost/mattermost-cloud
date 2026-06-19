// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package utility

import (
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mattermost/mattermost-cloud/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// withGitlabConfig temporarily sets the package-level GitLab token and
// gitops repo URL used by fetchFromGitlabIfNecessary, restoring previous
// values on cleanup.
func withGitlabConfig(t *testing.T, token, repoURL string) {
	t.Helper()
	prevToken := model.GetGitlabToken()
	prevURL := model.GetGitopsRepoURL()
	model.SetGitlabToken(token)
	model.SetGitopsRepoURL(repoURL)
	t.Cleanup(func() {
		model.SetGitlabToken(prevToken)
		model.SetGitopsRepoURL(prevURL)
	})
}

// withFetchClient swaps the package-level gitlabFetchClient for a test
// client that trusts httptest's TLS server and counts outbound calls.
func withFetchClient(t *testing.T) *atomic.Int64 {
	t.Helper()
	prev := gitlabFetchClient
	var hits atomic.Int64
	gitlabFetchClient = &http.Client{
		Timeout: 5 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Transport: &countingTransport{
			counter: &hits,
			next: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		},
	}
	t.Cleanup(func() { gitlabFetchClient = prev })
	return &hits
}

type countingTransport struct {
	counter *atomic.Int64
	next    http.RoundTripper
}

func (c *countingTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	c.counter.Add(1)
	return c.next.RoundTrip(r)
}

func TestFetchFromGitlabIfNecessary_NoTokenReturnsPathUnchanged(t *testing.T) {
	withGitlabConfig(t, "", "https://gitlab.example.com")
	hits := withFetchClient(t)

	in := "https://gitlab.example.com/api/v4/projects/1/repository/files/values.yaml?ref=main"
	out, cleanup, err := fetchFromGitlabIfNecessary(in)
	require.NoError(t, err)
	assert.Equal(t, in, out)
	assert.Nil(t, cleanup)
	assert.Zero(t, hits.Load())
}

func TestFetchFromGitlabIfNecessary_NonConfiguredHostReturnsPathUnchanged(t *testing.T) {
	withGitlabConfig(t, "secret-token", "https://gitlab.example.com")
	hits := withFetchClient(t)

	in := "https://example.org/values.yaml"
	out, cleanup, err := fetchFromGitlabIfNecessary(in)
	require.NoError(t, err)
	assert.Equal(t, in, out)
	assert.Nil(t, cleanup)
	assert.Zero(t, hits.Load())
}

// TestFetchFromGitlabIfNecessary_HostMatchIsExact covers a range of URL
// shapes whose host is not equal to the configured GitLab host. All of
// them must be treated as non-GitLab URLs and returned unchanged.
func TestFetchFromGitlabIfNecessary_HostMatchIsExact(t *testing.T) {
	withGitlabConfig(t, "secret-token", "https://gitlab.example.com")
	hits := withFetchClient(t)

	nonConfiguredURLs := []string{
		"https://git.example.com/values.yaml?ref=main",
		"https://github.com/x?a=1",
		"https://gitea.example.org/y?z=1",
		"https://gitlab.example.com.other.org/values.yaml?ref=main",
		"https://other.org/gitlab.example.com/values.yaml",
		"https://gitlab.example.com@other.org/values.yaml",
	}

	for _, u := range nonConfiguredURLs {
		out, cleanup, err := fetchFromGitlabIfNecessary(u)
		require.NoError(t, err, "url=%s", u)
		assert.Equal(t, u, out, "url=%s should be returned unchanged", u)
		assert.Nil(t, cleanup, "url=%s should not produce a cleanup", u)
	}
	assert.Zero(t, hits.Load())
}

func TestFetchFromGitlabIfNecessary_RejectsHTTPForConfiguredHost(t *testing.T) {
	withGitlabConfig(t, "secret-token", "https://gitlab.example.com")
	hits := withFetchClient(t)

	in := "http://gitlab.example.com/api/v4/projects/1/repository/files/values.yaml?ref=main"
	_, _, err := fetchFromGitlabIfNecessary(in)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "HTTPS")
	assert.Zero(t, hits.Load())
}

func TestFetchFromGitlabIfNecessary_SendsTokenViaPrivateTokenHeader(t *testing.T) {
	var (
		seenPrivateTokenHeader string
		seenRawQuery           string
	)

	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenPrivateTokenHeader = r.Header.Get("PRIVATE-TOKEN")
		seenRawQuery = r.URL.RawQuery
		body, _ := json.Marshal(gitlabValuesFileResponse{
			Content: base64.StdEncoding.EncodeToString([]byte("key: value\n")),
		})
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	}))
	t.Cleanup(srv.Close)

	srvURL, err := url.Parse(srv.URL)
	require.NoError(t, err)

	withGitlabConfig(t, "secret-token", "https://"+srvURL.Host)
	_ = withFetchClient(t)

	in := "https://" + srvURL.Host + "/api/v4/projects/1/repository/files/values.yaml?ref=main"
	out, cleanup, err := fetchFromGitlabIfNecessary(in)
	require.NoError(t, err)
	require.NotNil(t, cleanup)
	t.Cleanup(func() { cleanup(out) })

	assert.Equal(t, "secret-token", seenPrivateTokenHeader)
	assert.NotContains(t, seenRawQuery, "private_token")
	assert.NotContains(t, seenRawQuery, "secret-token")

	assert.True(t, strings.HasPrefix(out, os.TempDir()))
	content, err := os.ReadFile(out)
	require.NoError(t, err)
	assert.Equal(t, "key: value\n", string(content))
}

// TestFetchFromGitlabIfNecessary_CleansUpTempFileOnDecodeError verifies
// that a malformed (non-base64) response body does not leave an orphan
// temp file behind.
func TestFetchFromGitlabIfNecessary_CleansUpTempFileOnDecodeError(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Content field is not valid base64.
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"content":"!!!not base64!!!"}`))
	}))
	t.Cleanup(srv.Close)

	srvURL, err := url.Parse(srv.URL)
	require.NoError(t, err)

	withGitlabConfig(t, "secret-token", "https://"+srvURL.Host)
	_ = withFetchClient(t)

	before := countHelmValuesTempFiles(t)

	in := "https://" + srvURL.Host + "/api/v4/projects/1/repository/files/values.yaml?ref=main"
	_, cleanup, err := fetchFromGitlabIfNecessary(in)
	require.Error(t, err)
	assert.Nil(t, cleanup)

	after := countHelmValuesTempFiles(t)
	assert.Equal(t, before, after, "no orphan temp file should remain after a decode error")
}

// TestFetchFromGitlabIfNecessary_LimitsResponseBodySize verifies that an
// oversized response is truncated by the limit reader and surfaces as a
// JSON parse error rather than being read into memory in full.
func TestFetchFromGitlabIfNecessary_LimitsResponseBodySize(t *testing.T) {
	// Build a payload larger than maxGitlabResponseSize so the limit
	// reader truncates mid-stream and JSON parsing fails.
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"content":"`))
		buf := make([]byte, 1<<20)
		for i := range buf {
			buf[i] = 'A'
		}
		for written := 0; written < maxGitlabResponseSize+(2<<20); written += len(buf) {
			_, _ = w.Write(buf)
		}
		w.Write([]byte(`"}`))
	}))
	t.Cleanup(srv.Close)

	srvURL, err := url.Parse(srv.URL)
	require.NoError(t, err)

	withGitlabConfig(t, "secret-token", "https://"+srvURL.Host)
	_ = withFetchClient(t)

	in := "https://" + srvURL.Host + "/api/v4/projects/1/repository/files/values.yaml?ref=main"
	_, _, err = fetchFromGitlabIfNecessary(in)
	require.Error(t, err)
}

// countHelmValuesTempFiles returns the number of files in os.TempDir()
// whose name matches the helm-values-file- prefix used by the function.
func countHelmValuesTempFiles(t *testing.T) int {
	t.Helper()
	entries, err := os.ReadDir(os.TempDir())
	require.NoError(t, err)
	count := 0
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "helm-values-file-") {
			count++
		}
	}
	return count
}

// TestFetchFromGitlabIfNecessary_DoesNotFollowRedirects ensures the
// fetch client surfaces a redirect response as an error rather than
// silently following it.
func TestFetchFromGitlabIfNecessary_DoesNotFollowRedirects(t *testing.T) {
	var downstreamHit atomic.Bool
	downstream := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		downstreamHit.Store(true)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(downstream.Close)

	gitlab := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, downstream.URL+"/v", http.StatusFound)
	}))
	t.Cleanup(gitlab.Close)

	gitlabURL, err := url.Parse(gitlab.URL)
	require.NoError(t, err)

	withGitlabConfig(t, "secret-token", "https://"+gitlabURL.Host)
	_ = withFetchClient(t)

	in := "https://" + gitlabURL.Host + "/api/v4/projects/1/repository/files/values.yaml?ref=main"
	_, _, err = fetchFromGitlabIfNecessary(in)
	require.Error(t, err)
	assert.False(t, downstreamHit.Load())
}
