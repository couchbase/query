package expression

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"

	"github.com/couchbase/query/util"
)

// IsUrlAllowedInCluster checks whether urlObj is permitted by the cluster
// allowlist stored in the request context.  It delegates evaluation to
// util.ValidateURLInAllowlist, which enforces restricted paths (/diag/eval),
// the all_access flag, allowed/disallowed URL lists, and wildcard host patterns.
// All errors are prefixed with "cluster allowlist:" so callers can distinguish
// them from credential-allowlist or URL-parse errors.
func IsUrlAllowedInCluster(urlObj *url.URL, context Context) error {
	var urlList map[string]interface{}

	if _curlContext, ok := context.(CurlContext); ok {
		urlList = _curlContext.GetAllowlist()
	}

	if len(urlList) == 0 {
		return fmt.Errorf("cluster allowlist: allowed URL list is empty")
	}

	if err := util.ValidateURLInAllowlist(urlObj.String(), urlList, nil); err != nil {
		return fmt.Errorf("cluster allowlist: %v", err)
	}
	return nil
}

// GetDefaultHttpClient returns an http.Client configured for CURL() requests:
//   - Redirects are disabled (returns the last response instead of following).
//   - HTTP/2 is preferred.
//   - TLS certificate verification is enabled by default (mirrors libcurl behaviour).
//   - Timeout is inherited from the query context.
func GetDefaultHttpClient(context Context) (http.Client, error) {
	client := http.Client{

		// Override the default CheckRedirect method
		// Now, if url redirection is attempted - call finishes after the first request
		// no error is returned ( since CheckRedirect returns the ErrUseLastResponse error )
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},

		Transport: &http.Transport{
			ForceAttemptHTTP2: true,
			TLSClientConfig: &tls.Config{
				// By default libcurl performs SSL certificate validation
				InsecureSkipVerify: false,
			},
		},

		// Default value of max-time be the request timeout
		Timeout: context.GetTimeout(),
	}

	return client, nil
}
