// Copyright 2026-Present Couchbase, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL-Couchbase.txt.  As of the Change Date specified in
// that file, in accordance with the Business Source License, use of this
// software will be governed by the Apache License, Version 2.0, included in
// the file licenses/APL2.txt.

// cred_handler.go is the single entry point for resolving a credstore
// credential into a ready-to-use http.Client and http.Header.  One handler
// exists per credential type supported by the credstore:
//
//	HTTP        – basic / bearer / mTLS
//	Couchbase   – username+password (basic) or mTLS, with optional TLS config
//	AWS         – SigV4-signing RoundTripper
//	AzureShared – HMAC-SHA256 Azure Shared Key RoundTripper
//	AzureAD     – OAuth2 client-credentials token fetch
//	AzureSAS    – query-string SAS token injection RoundTripper
//	AzureManaged– Azure IMDS token fetch
//	GCP         – service-account JWT exchange or HMAC-mode SigV4

package expression

import (
	"bytes"
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/couchbase/cbauth"
	"github.com/couchbase/query/util"
	"github.com/youmark/pkcs8"
)

// getCbAuthTLSConfig is a variable so unit tests can substitute a stub without
// requiring a live cbauth daemon.  Production code always uses cbauth.GetTLSConfig.
var getCbAuthTLSConfig = cbauth.GetTLSConfig

// _tokenFetchTimeout is the maximum time allowed for an external OAuth2 / IMDS
// token fetch.  Individual calls use min(context.GetTimeout(), _tokenFetchTimeout)
// so a short-running query does not block longer than its own deadline.
const _tokenFetchTimeout = 30 * time.Second

// HandleCred is the single public entry point.  It fetches the credential
// identified by credId from the credstore, enforces URL guardrails, and
// returns a configured http.Client and http.Header appropriate for the
// credential's type.  Callers may layer additional headers on top of the
// returned header before issuing the request.
func HandleCred(urlObj *url.URL, credId string, context Context) (http.Client, http.Header, error) {
	cred, err := fetchCred(credId, context)
	if err != nil {
		return http.Client{}, http.Header{}, err
	}

	if err = isUrlAllowedForCred(urlObj, cred); err != nil {
		return http.Client{}, http.Header{}, err
	}

	client, header, err := applyCredential(cred, urlObj, context)
	if err != nil {
		return client, header, err
	}

	// Disable redirects so an allowed host cannot 3xx-redirect to an internal
	// target and bypass the allowlist (parity with GetDefaultHttpClient / CURL()).
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}
	return client, header, nil
}

// ─── Credential fetch & URL guardrail ────────────────────────────────────────

// fetchCred retrieves any credential type from the credstore.
func fetchCred(credId string, context Context) (*cbauth.Credential, error) {
	cred, err := context.ExternalCredential(credId)
	if err != nil {
		return nil, err
	}
	if cred == nil {
		return nil, fmt.Errorf("credential %q not found", credId)
	}
	return cred, nil
}

// isUrlAllowedForCred enforces the URL guardrails stored on the credential,
// delegating evaluation to util.ValidateURLInAllowlist which covers restricted
// paths (/diag/eval), all_access, allow/disallow lists, and wildcard hosts.
func isUrlAllowedForCred(urlObj *url.URL, cred *cbauth.Credential) error {
	gl := cred.Meta.Guardrails.URLWhitelist
	if gl == nil {
		return fmt.Errorf("credential allowlist: no URL guardrails found for credential %q", cred.ID)
	}
	allowlist := map[string]any{
		util.AllowlistKeyAllAccess:      gl.AllAccess,
		util.AllowlistKeyAllowedURLs:    gl.AllowedURLs,
		util.AllowlistKeyDisallowedURLs: gl.DisallowedURLs,
	}
	if err := util.ValidateURLInAllowlist(urlObj.String(), nil, allowlist); err != nil {
		return fmt.Errorf("credential allowlist: %v", err)
	}
	return nil
}

// ─── Dispatcher ───────────────────────────────────────────────────────────────

// applyCredential dispatches to the handler for the credential's concrete
// payload type.  Exactly one payload field will be non-nil per credential.
func applyCredential(cred *cbauth.Credential, urlObj *url.URL, context Context) (http.Client, http.Header, error) {
	switch {
	case cred.HTTP != nil:
		return applyHTTPPayload(cred, context)
	case cred.Couchbase != nil:
		return applyCouchbasePayload(cred, context)
	case cred.AWS != nil:
		return applyAWSPayload(cred, context)
	case cred.AzureShared != nil:
		return applyAzureSharedPayload(cred, context)
	case cred.AzureAD != nil:
		return applyAzureADPayload(cred, urlObj, context)
	case cred.AzureSAS != nil:
		return applyAzureSASPayload(cred, context)
	case cred.AzureManaged != nil:
		return applyAzureManagedPayload(cred, urlObj, context)
	case cred.GCP != nil:
		return applyGCPPayload(cred, context)
	default:
		return http.Client{}, http.Header{}, fmt.Errorf("credential %q has no recognized payload type", cred.ID)
	}
}

// ─── mTLS key-pair helper ────────────────────────────────────────────────────

// pemKeyPair builds a tls.Certificate from PEM-encoded cert and key content
// (as stored in the credstore), bypassing the file-path-based LoadX509KeyPair.
//
// When passphrase is empty the key is assumed to be unencrypted and the stdlib
// tls.X509KeyPair is used directly (handles PKCS#1, PKCS#8, and EC keys).
// When a passphrase is provided the key PEM block is parsed with full key-type
// coverage (PKCS#1 RSA → encrypted PKCS#8 via youmark → EC) and the cert/key
// pair is verified to match before being returned — mirroring the behaviour of
// goutils/tls.x509KeyPair.
func pemKeyPair(certPEM, keyPEM string, passphrase []byte) (tls.Certificate, error) {
	if len(passphrase) == 0 {
		return tls.X509KeyPair([]byte(certPEM), []byte(keyPEM))
	}

	// Passphrase supplied.  Decode the certificate chain.
	var certDERs [][]byte
	certRest := []byte(certPEM)
	for {
		var block *pem.Block
		block, certRest = pem.Decode(certRest)
		if block == nil {
			break
		}
		if block.Type == "CERTIFICATE" {
			certDERs = append(certDERs, block.Bytes)
		}
	}
	if len(certDERs) == 0 {
		return tls.Certificate{}, fmt.Errorf("pemKeyPair: no CERTIFICATE block found in certificate PEM")
	}

	// Decode the private key PEM block.
	keyBlock, _ := pem.Decode([]byte(keyPEM))
	if keyBlock == nil {
		return tls.Certificate{}, fmt.Errorf("pemKeyPair: no PEM block found in privateKey")
	}

	// Parse the key with passphrase: PKCS#1 RSA → encrypted PKCS#8 → EC.
	privKey, err := pemParsePrivateKey(keyBlock.Bytes, passphrase)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("pemKeyPair: %v", err)
	}

	// Verify the cert and key are a matching pair.
	x509Cert, err := x509.ParseCertificate(certDERs[0])
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("pemKeyPair: failed to parse certificate: %v", err)
	}
	if err := pemVerifyKeyPair(x509Cert.PublicKey, privKey); err != nil {
		return tls.Certificate{}, fmt.Errorf("pemKeyPair: %v", err)
	}

	return tls.Certificate{
		Certificate: certDERs,
		PrivateKey:  privKey,
	}, nil
}

// pemParsePrivateKey mirrors goutils/tls.parsePrivateKey: tries PKCS#1 RSA,
// then encrypted PKCS#8 (youmark), then EC, accumulating errors on each miss.
func pemParsePrivateKey(der, passphrase []byte) (crypto.PrivateKey, error) {
	var errs []string
	if key, err := x509.ParsePKCS1PrivateKey(der); err == nil {
		return key, nil
	} else {
		errs = append(errs, err.Error())
	}
	if key, err := pkcs8.ParsePKCS8PrivateKey(der, passphrase); err == nil {
		switch key.(type) {
		case *rsa.PrivateKey, *ecdsa.PrivateKey, ed25519.PrivateKey:
			return key.(crypto.PrivateKey), nil
		default:
			return nil, fmt.Errorf("unknown private key type returned by PKCS#8 parse")
		}
	} else {
		errs = append(errs, err.Error())
	}
	if key, err := x509.ParseECPrivateKey(der); err == nil {
		return key, nil
	} else {
		errs = append(errs, err.Error())
	}
	return nil, fmt.Errorf("failed to parse private key: %s", strings.Join(errs, "; "))
}

// pemVerifyKeyPair confirms that the public key from the certificate and the
// private key are a matching pair, mirroring goutils/tls.x509KeyPair's check.
func pemVerifyKeyPair(pub interface{}, priv crypto.PrivateKey) error {
	switch pub := pub.(type) {
	case *rsa.PublicKey:
		pk, ok := priv.(*rsa.PrivateKey)
		if !ok {
			return fmt.Errorf("private key type does not match public key type")
		}
		if pub.N.Cmp(pk.N) != 0 {
			return fmt.Errorf("private key does not match public key")
		}
	case *ecdsa.PublicKey:
		pk, ok := priv.(*ecdsa.PrivateKey)
		if !ok {
			return fmt.Errorf("private key type does not match public key type")
		}
		if pub.X.Cmp(pk.X) != 0 || pub.Y.Cmp(pk.Y) != 0 {
			return fmt.Errorf("private key does not match public key")
		}
	case ed25519.PublicKey:
		pk, ok := priv.(ed25519.PrivateKey)
		if !ok {
			return fmt.Errorf("private key type does not match public key type")
		}
		if !bytes.Equal(pk.Public().(ed25519.PublicKey), pub) {
			return fmt.Errorf("private key does not match public key")
		}
	default:
		return fmt.Errorf("unknown public key algorithm")
	}
	return nil
}

// ─── HTTP ─────────────────────────────────────────────────────────────────────

// applyHTTPPayload configures an http.Client and http.Header from an HTTP
// credential payload (basic, bearer, or mTLS auth schemes).
//
// Spec references:
//   - Basic auth:   RFC 7617  https://datatracker.ietf.org/doc/html/rfc7617
//   - Bearer token: RFC 6750  https://datatracker.ietf.org/doc/html/rfc6750
//   - mTLS:         RFC 8446  https://datatracker.ietf.org/doc/html/rfc8446 §4.3.2
func applyHTTPPayload(cred *cbauth.Credential, context Context) (http.Client, http.Header, error) {
	p := cred.HTTP

	client, err := GetDefaultHttpClient(context)
	if err != nil {
		return http.Client{}, http.Header{}, err
	}
	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		return http.Client{}, http.Header{}, fmt.Errorf("HTTP credential: client's Transport is not an *http.Transport")
	}

	if p.SkipVerify {
		transport.TLSClientConfig.InsecureSkipVerify = true
	}

	if p.RootCertificate != "" {
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM([]byte(p.RootCertificate)) {
			return http.Client{}, http.Header{}, fmt.Errorf("HTTP credential: invalid rootCertificate: no valid PEM certificates found")
		}
		transport.TLSClientConfig.RootCAs = pool
	}

	if p.Certificate != "" && p.PrivateKey != "" {
		tlsCert, err := pemKeyPair(p.Certificate, p.PrivateKey, []byte(p.Passphrase))
		if err != nil {
			return http.Client{}, http.Header{}, fmt.Errorf("HTTP credential: %v", err)
		}
		transport.TLSClientConfig.Certificates = []tls.Certificate{tlsCert}
	} else if p.Certificate != "" || p.PrivateKey != "" {
		return http.Client{}, http.Header{}, fmt.Errorf("HTTP credential: both certificate and privateKey are required")
	}

	tlsConfig, err := getCbAuthTLSConfig()
	if err != nil {
		return http.Client{}, http.Header{}, fmt.Errorf("HTTP credential: failed to get cbauth tls config: %v", err)
	}
	transport.TLSClientConfig.CipherSuites = tlsConfig.CipherSuites

	header := http.Header{}
	switch p.AuthScheme {
	case "basic":
		if p.Username == "" {
			return http.Client{}, http.Header{}, fmt.Errorf("HTTP credential: username required for basic authScheme")
		}
		encoded := base64.StdEncoding.EncodeToString([]byte(p.Username + ":" + p.Password))
		header.Set("Authorization", "Basic "+encoded)
	case "bearer":
		if p.Token == "" {
			return http.Client{}, http.Header{}, fmt.Errorf("HTTP credential: token required for bearer authScheme")
		}
		if p.HeaderName == "" || strings.ToLower(p.HeaderName) == "authorization" {
			header.Set("Authorization", "Bearer "+p.Token)
		} else {
			header.Set(p.HeaderName, p.Token)
		}
	case "mtls":
		if p.Certificate == "" || p.PrivateKey == "" {
			return http.Client{}, http.Header{}, fmt.Errorf("HTTP credential: both certificate and privateKey are required for mtls authScheme")
		}
		// Auth is transport-level; no Authorization header needed.
	default:
		return http.Client{}, http.Header{}, fmt.Errorf("HTTP credential: unsupported authScheme %q", p.AuthScheme)
	}

	return client, header, nil
}

// ─── Couchbase ────────────────────────────────────────────────────────────────

// applyCouchbasePayload builds an http.Client from a Couchbase credential.
// mTLS is used when certificate+privateKey are present; otherwise a Basic
// Authorization header is constructed from username and password.
//
// EncryptionType controls TLS enforcement:
//   - "none"       plain-text; TLS negotiated only if the caller uses https://
//   - "half"       TLS enabled, peer certificate NOT verified (CURLOPT_SSL_VERIFYPEER=0)
//   - "full" / ""  TLS enabled, peer certificate verified against RootCertificate
//
// Spec references:
//   - Basic auth: RFC 7617  https://datatracker.ietf.org/doc/html/rfc7617
//   - mTLS:       RFC 8446  https://datatracker.ietf.org/doc/html/rfc8446 §4.3.2
func applyCouchbasePayload(cred *cbauth.Credential, context Context) (http.Client, http.Header, error) {
	p := cred.Couchbase

	client, err := GetDefaultHttpClient(context)
	if err != nil {
		return http.Client{}, http.Header{}, err
	}
	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		return http.Client{}, http.Header{}, fmt.Errorf("Couchbase credential: client's Transport is not an *http.Transport")
	}

	switch p.EncryptionType {
	case "none":
		// Plain-text connection; no TLS config changes required.
		// TLS is negotiated only when the caller uses an https:// URL, in
		// which case Go's default TLS settings apply unchanged.
	case "half":
		// TLS encrypts the connection, but the peer's certificate is NOT
		// verified — equivalent to libcurl's CURLOPT_SSL_VERIFYPEER=0.
		transport.TLSClientConfig.InsecureSkipVerify = true
		tlsConfig, err := getCbAuthTLSConfig()
		if err != nil {
			return http.Client{}, http.Header{}, fmt.Errorf("Couchbase credential: failed to get cbauth tls config: %v", err)
		}
		transport.TLSClientConfig.CipherSuites = tlsConfig.CipherSuites
	case "full", "":
		// Full TLS: peer certificate is verified.
		// An empty EncryptionType defaults to full verification for safety.
		if p.RootCertificate != "" {
			pool := x509.NewCertPool()
			if !pool.AppendCertsFromPEM([]byte(p.RootCertificate)) {
				return http.Client{}, http.Header{}, fmt.Errorf("Couchbase credential: invalid rootCertificate")
			}
			transport.TLSClientConfig.RootCAs = pool
		}
		tlsConfig, err := getCbAuthTLSConfig()
		if err != nil {
			return http.Client{}, http.Header{}, fmt.Errorf("Couchbase credential: failed to get cbauth tls config: %v", err)
		}
		transport.TLSClientConfig.CipherSuites = tlsConfig.CipherSuites
	default:
		return http.Client{}, http.Header{}, fmt.Errorf("Couchbase credential: unsupported encryptionType %q", p.EncryptionType)
	}

	header := http.Header{}
	switch {
	case p.Certificate != "" && p.PrivateKey != "":
		tlsCert, err := pemKeyPair(p.Certificate, p.PrivateKey, []byte(p.Passphrase))
		if err != nil {
			return http.Client{}, http.Header{}, fmt.Errorf("Couchbase credential: %v", err)
		}
		transport.TLSClientConfig.Certificates = []tls.Certificate{tlsCert}
	case p.Certificate != "" || p.PrivateKey != "":
		return http.Client{}, http.Header{}, fmt.Errorf("Couchbase credential: both certificate and privateKey are required for mTLS")
	case p.Username != "":
		encoded := base64.StdEncoding.EncodeToString([]byte(p.Username + ":" + p.Password))
		header.Set("Authorization", "Basic "+encoded)
	default:
		return http.Client{}, http.Header{}, fmt.Errorf("Couchbase credential: no auth material configured (username or certificate+privateKey required)")
	}

	return client, header, nil
}

// ─── AWS SigV4 ────────────────────────────────────────────────────────────────

// applyAWSPayload wraps the default client with a SigV4-signing transport.
// The AWS service name is inferred from the request URL host at signing time
// (e.g. "bedrock-runtime.us-east-1.amazonaws.com" → service "bedrock-runtime").
// When Endpoint is set the transport rewrites the request's scheme+host to
// that base URL before signing, enabling S3-compatible services (MinIO, etc.).
//
// Spec references:
//   - AWS Signature Version 4: https://docs.aws.amazon.com/IAM/latest/UserGuide/reference_aws-signing.html
//   - Canonical request format: https://docs.aws.amazon.com/IAM/latest/UserGuide/create-signed-request.html
func applyAWSPayload(cred *cbauth.Credential, context Context) (http.Client, http.Header, error) {
	p := cred.AWS

	if p.AccessKeyID == "" {
		return http.Client{}, http.Header{}, fmt.Errorf("AWS credential: accessKeyId is required")
	}
	if p.SecretAccessKey == "" {
		return http.Client{}, http.Header{}, fmt.Errorf("AWS credential: secretAccessKey is required")
	}

	var endpointURL *url.URL
	if p.Endpoint != "" {
		var err error
		endpointURL, err = util.ParseAndValidateURL(p.Endpoint)
		if err != nil {
			return http.Client{}, http.Header{}, fmt.Errorf("AWS credential: invalid endpoint %q: %v", p.Endpoint, err)
		}
		if util.IsRestrictedURL(endpointURL) {
			return http.Client{}, http.Header{}, fmt.Errorf("AWS credential: service endpoint blocked - access restricted: %v", endpointURL.String())
		}
	}

	client, err := GetDefaultHttpClient(context)
	if err != nil {
		return http.Client{}, http.Header{}, err
	}
	client.Transport = &awsSigV4Transport{
		base:            client.Transport,
		accessKeyID:     p.AccessKeyID,
		secretAccessKey: p.SecretAccessKey,
		sessionToken:    strings.TrimSpace(p.SessionToken),
		region:          p.Region,
		endpoint:        endpointURL,
	}
	return client, http.Header{}, nil
}

// awsSigV4Transport signs outgoing requests with AWS Signature Version 4.
type awsSigV4Transport struct {
	base            http.RoundTripper
	accessKeyID     string
	secretAccessKey string
	sessionToken    string
	region          string
	// service overrides the AWS service name used in the SigV4 credential scope.
	// When empty, it is inferred from the request URL host (e.g.
	// "bedrock-runtime.us-east-1.amazonaws.com" → "bedrock-runtime").
	// Set explicitly only for GCP HMAC mode, where GCS hosts do not follow
	// the AWS naming convention and the service must be hardcoded to "s3".
	service string
	// endpoint overrides the request's scheme+host when targeting an
	// S3-compatible service (MinIO, LocalStack, Couchbase Capella, etc.).
	// nil means use the URL supplied by the caller unchanged.
	endpoint *url.URL
}

func (t *awsSigV4Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	var bodyBytes []byte
	if req.Body != nil {
		var err error
		bodyBytes, err = io.ReadAll(req.Body)
		req.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("AWS SigV4: failed to read request body: %v", err)
		}
	}

	clone := req.Clone(req.Context())
	if len(bodyBytes) > 0 {
		clone.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	} else {
		clone.Body = nil
	}

	// Derive the AWS service name from the original request host before any
	// endpoint rewrite so that custom endpoints (e.g. localhost:9000 for
	// MinIO) don't produce a wrong service name like "localhost".
	service := t.service
	if service == "" {
		service = awsServiceFromHost(clone.URL.Host)
	}

	// If an endpoint override is configured, rewrite scheme+host before signing
	// so both the canonical request and the actual TCP connection use it.
	if t.endpoint != nil {
		clone.URL.Scheme = t.endpoint.Scheme
		clone.URL.Host = t.endpoint.Host
	}

	now := time.Now().UTC()
	dateISO := now.Format("20060102T150405Z")
	dateShort := now.Format("20060102")
	payloadHash := sha256HexBytes(bodyBytes)

	clone.Header.Set("X-Amz-Date", dateISO)
	clone.Header.Set("X-Amz-Content-Sha256", payloadHash)
	if t.sessionToken != "" {
		clone.Header.Set("X-Amz-Security-Token", t.sessionToken)
	}

	signedHdrs, canonicalHdrs := awsCanonicalSignedHeaders(clone.Header, clone.URL.Host)
	canonicalReq := strings.Join([]string{
		clone.Method,
		awsCanonicalURI(clone.URL),
		awsCanonicalQueryString(clone.URL),
		canonicalHdrs,
		signedHdrs,
		payloadHash,
	}, "\n")

	credScope := strings.Join([]string{dateShort, t.region, service, "aws4_request"}, "/")
	stringToSign := strings.Join([]string{"AWS4-HMAC-SHA256", dateISO, credScope, sha256HexString(canonicalReq)}, "\n")
	signingKey := awsDeriveSigningKey(t.secretAccessKey, dateShort, t.region, service)
	signature := hex.EncodeToString(hmacSHA256(signingKey, []byte(stringToSign)))

	clone.Header.Set("Authorization", fmt.Sprintf(
		"AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		t.accessKeyID, credScope, signedHdrs, signature,
	))
	return t.base.RoundTrip(clone)
}

func awsServiceFromHost(host string) string {
	if idx := strings.LastIndex(host, ":"); idx > strings.LastIndex(host, "]") {
		host = host[:idx]
	}
	if parts := strings.SplitN(host, ".", 2); len(parts) > 0 && parts[0] != "" {
		return parts[0]
	}
	return "execute-api"
}

func awsCanonicalURI(u *url.URL) string {
	if u.EscapedPath() == "" {
		return "/"
	}
	return u.EscapedPath()
}

func awsCanonicalQueryString(u *url.URL) string {
	q := u.Query()
	keys := make([]string, 0, len(q))
	for k := range q {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var parts []string
	for _, k := range keys {
		vals := q[k]
		sort.Strings(vals)
		for _, v := range vals {
			parts = append(parts, url.QueryEscape(k)+"="+url.QueryEscape(v))
		}
	}
	return strings.Join(parts, "&")
}

func awsCanonicalSignedHeaders(h http.Header, host string) (signedHeaders, canonicalHeaders string) {
	lower := map[string]string{"host": host}
	for k, vals := range h {
		lower[strings.ToLower(k)] = strings.TrimSpace(strings.Join(vals, ","))
	}
	names := make([]string, 0, len(lower))
	for k := range lower {
		names = append(names, k)
	}
	sort.Strings(names)
	var hdr strings.Builder
	for _, n := range names {
		hdr.WriteString(n)
		hdr.WriteString(":")
		hdr.WriteString(lower[n])
		hdr.WriteString("\n")
	}
	return strings.Join(names, ";"), hdr.String()
}

func awsDeriveSigningKey(secret, date, region, service string) []byte {
	kDate := hmacSHA256([]byte("AWS4"+secret), []byte(date))
	kRegion := hmacSHA256(kDate, []byte(region))
	kService := hmacSHA256(kRegion, []byte(service))
	return hmacSHA256(kService, []byte("aws4_request"))
}

// ─── Azure Shared Key ─────────────────────────────────────────────────────────

// applyAzureSharedPayload wraps the client with a transport that signs every
// request using Azure Blob Storage Shared Key (HMAC-SHA256).
// When Endpoint is set the transport rewrites the request's scheme+host to
// that base URL before signing, enabling Azure Stack or emulator targets.
//
// Spec references:
//   - Azure Blob Storage Shared Key authorization:
//     https://learn.microsoft.com/en-us/rest/api/storageservices/authorize-with-shared-key
//   - Azure Storage service versioning (x-ms-version header):
//     https://learn.microsoft.com/en-us/rest/api/storageservices/versioning-for-the-azure-storage-services
func applyAzureSharedPayload(cred *cbauth.Credential, context Context) (http.Client, http.Header, error) {
	p := cred.AzureShared

	if p.AccountName == "" {
		return http.Client{}, http.Header{}, fmt.Errorf("Azure Shared Key credential: accountName is required")
	}
	if p.AccountKey == "" {
		return http.Client{}, http.Header{}, fmt.Errorf("Azure Shared Key credential: accountKey is required")
	}

	keyBytes, err := base64.StdEncoding.DecodeString(p.AccountKey)
	if err != nil {
		return http.Client{}, http.Header{}, fmt.Errorf("Azure Shared Key credential: invalid accountKey base64: %v", err)
	}

	var endpointURL *url.URL
	if p.Endpoint != "" {
		endpointURL, err = util.ParseAndValidateURL(p.Endpoint)
		if err != nil {
			return http.Client{}, http.Header{}, fmt.Errorf("Azure Shared Key credential: invalid endpoint %q: %v", p.Endpoint, err)
		}
		if util.IsRestrictedURL(endpointURL) {
			return http.Client{}, http.Header{}, fmt.Errorf("Azure Shared Key credential: service endpoint blocked - access restricted: %v", endpointURL.String())
		}
	}

	client, err := GetDefaultHttpClient(context)
	if err != nil {
		return http.Client{}, http.Header{}, err
	}
	client.Transport = &azureSharedKeyTransport{
		base:        client.Transport,
		accountName: p.AccountName,
		accountKey:  keyBytes,
		endpoint:    endpointURL,
	}
	return client, http.Header{}, nil
}

type azureSharedKeyTransport struct {
	base        http.RoundTripper
	accountName string
	accountKey  []byte
	// endpoint overrides the request's scheme+host when targeting Azure Stack
	// or the Azurite emulator.  nil means use the caller's URL unchanged.
	endpoint *url.URL
}

func (t *azureSharedKeyTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	clone := req.Clone(req.Context())
	if t.endpoint != nil {
		clone.URL.Scheme = t.endpoint.Scheme
		clone.URL.Host = t.endpoint.Host
	}
	clone.Header.Set("x-ms-date", time.Now().UTC().Format(http.TimeFormat))
	// x-ms-version pins the Azure Storage REST API version used for all requests.
	// "2024-11-04" is the latest stable GA version at time of writing; pinning
	// ensures consistent behaviour and available feature set regardless of future
	// service defaults.  Update when newer API features are required.
	clone.Header.Set("x-ms-version", "2024-11-04")

	sig, err := azureSharedKeySignature(t.accountKey, t.accountName, clone)
	if err != nil {
		return nil, err
	}
	clone.Header.Set("Authorization", "SharedKey "+t.accountName+":"+sig)
	return t.base.RoundTrip(clone)
}

func azureSharedKeySignature(key []byte, accountName string, req *http.Request) (string, error) {
	contentLength := req.Header.Get("Content-Length")
	if contentLength == "0" {
		contentLength = ""
	}

	var msHeaders []string
	for k, v := range req.Header {
		lk := strings.ToLower(k)
		if strings.HasPrefix(lk, "x-ms-") {
			msHeaders = append(msHeaders, lk+":"+strings.TrimSpace(strings.Join(v, ",")))
		}
	}
	sort.Strings(msHeaders)

	var canonicalResourceBuilder strings.Builder
	canonicalResourceBuilder.WriteString("/")
	canonicalResourceBuilder.WriteString(accountName)
	canonicalResourceBuilder.WriteString(req.URL.EscapedPath())
	q := req.URL.Query()
	if len(q) > 0 {
		qKeys := make([]string, 0, len(q))
		for k := range q {
			qKeys = append(qKeys, k)
		}
		sort.Strings(qKeys)
		for _, k := range qKeys {
			vals := q[k]
			sort.Strings(vals)
			canonicalResourceBuilder.WriteString("\n")
			canonicalResourceBuilder.WriteString(strings.ToLower(k))
			canonicalResourceBuilder.WriteString(":")
			canonicalResourceBuilder.WriteString(strings.Join(vals, ","))
		}
	}
	canonicalResource := canonicalResourceBuilder.String()

	stringToSign := strings.Join([]string{
		req.Method,
		req.Header.Get("Content-Encoding"),
		req.Header.Get("Content-Language"),
		contentLength,
		req.Header.Get("Content-MD5"),
		req.Header.Get("Content-Type"),
		req.Header.Get("Date"),
		req.Header.Get("If-Modified-Since"),
		req.Header.Get("If-Match"),
		req.Header.Get("If-None-Match"),
		req.Header.Get("If-Unmodified-Since"),
		req.Header.Get("Range"),
		strings.Join(msHeaders, "\n"),
		canonicalResource,
	}, "\n")

	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(stringToSign))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil)), nil
}

// ─── Azure Active Directory (OAuth2 client credentials) ───────────────────────

// applyAzureADPayload fetches an OAuth2 access token via the client-credentials
// flow and returns it as a Bearer Authorization header.  The OAuth2 scope is
// derived from the target URL host: "https://{host}/.default".
//
// Two client-authentication methods are supported:
//   - Client secret (default): a static shared secret sent as client_secret.
//   - Client certificate:      a signed JWT assertion (client_assertion) built
//     from the certificate's private key; the x5t header carries the SHA-1
//     thumbprint so Azure AD can look up the registered public key.
//
// Spec references:
//   - OAuth2 client credentials grant: RFC 6749 §4.4  https://datatracker.ietf.org/doc/html/rfc6749#section-4.4
//   - JWT client assertions:           RFC 7521       https://datatracker.ietf.org/doc/html/rfc7521
//   - JSON Web Signatures (JWS):       RFC 7515       https://datatracker.ietf.org/doc/html/rfc7515
//   - Microsoft identity client assertion format:
//     https://learn.microsoft.com/en-us/entra/identity-platform/certificate-credentials
func applyAzureADPayload(cred *cbauth.Credential, urlObj *url.URL, context Context) (http.Client, http.Header, error) {
	p := cred.AzureAD

	if p.TenantID == "" {
		return http.Client{}, http.Header{}, fmt.Errorf("Azure AD credential: tenantId is required")
	}
	if p.ClientID == "" {
		return http.Client{}, http.Header{}, fmt.Errorf("Azure AD credential: clientId is required")
	}

	if p.Endpoint != "" {
		endpointURL, err := util.ParseAndValidateURL(p.Endpoint)
		if err != nil {
			return http.Client{}, http.Header{}, fmt.Errorf("Azure AD credential: invalid endpoint %q: %v", p.Endpoint, err)
		}
		if util.IsRestrictedURL(endpointURL) {
			return http.Client{}, http.Header{}, fmt.Errorf("Azure AD credential: service endpoint blocked - access restricted: %v", endpointURL.String())
		}
	}

	scope := "https://" + urlObj.Hostname() + "/.default"

	tokenTimeout := _tokenFetchTimeout
	if qt := context.GetTimeout(); qt > 0 && qt < tokenTimeout {
		tokenTimeout = qt
	}

	var (
		token string
		err   error
	)
	switch {
	case p.Certificate != "":
		token, err = fetchAzureADTokenWithCert(p.TenantID, p.ClientID, p.Certificate, p.Endpoint, scope, tokenTimeout)
	default:
		token, err = fetchAzureADToken(p.TenantID, p.ClientID, p.ClientSecret, p.Endpoint, scope, tokenTimeout)
	}
	if err != nil {
		return http.Client{}, http.Header{}, err
	}

	client, err := GetDefaultHttpClient(context)
	if err != nil {
		return http.Client{}, http.Header{}, err
	}
	header := http.Header{}
	header.Set("Authorization", "Bearer "+token)
	return client, header, nil
}

func fetchAzureADToken(tenantID, clientID, clientSecret, endpoint, scope string, timeout time.Duration) (string, error) {
	authority := "https://login.microsoftonline.com"
	if endpoint != "" {
		authority = strings.TrimRight(endpoint, "/")
	}
	tokenURL := authority + "/" + tenantID + "/oauth2/v2.0/token"

	data := url.Values{
		"grant_type": {"client_credentials"},
		"client_id":  {clientID},
		"scope":      {scope},
	}
	if clientSecret != "" {
		data.Set("client_secret", clientSecret)
	}

	resp, err := (&http.Client{Timeout: timeout}).PostForm(tokenURL, data)
	if err != nil {
		return "", fmt.Errorf("Azure AD credential: token fetch failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		errBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Azure AD credential: token endpoint returned HTTP %d: %s", resp.StatusCode, string(errBody))
	}

	var result struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
		ErrorDesc   string `json:"error_description"`
	}
	if err = json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("Azure AD credential: token response parse failed: %v", err)
	}
	if result.Error != "" {
		return "", fmt.Errorf("Azure AD credential: token error %q: %s", result.Error, result.ErrorDesc)
	}
	return result.AccessToken, nil
}

// fetchAzureADTokenWithCert obtains an OAuth2 access token via the Azure AD
// client-credentials flow authenticated with a client certificate rather than
// a shared secret.  The caller supplies a PEM bundle containing a certificate
// and an unencrypted RSA private key.  A signed JWT assertion is constructed
// and sent as the client_assertion.
//
// Note: encrypted (password-protected) private keys are intentionally not
// supported.  The credential store's own encryption should be relied on to
// protect the key material; keys must be provided as unencrypted PEM.
func fetchAzureADTokenWithCert(tenantID, clientID, certPEM, endpoint, scope string, timeout time.Duration) (string, error) {
	authority := "https://login.microsoftonline.com"
	if endpoint != "" {
		authority = strings.TrimRight(endpoint, "/")
	}
	tokenURL := authority + "/" + tenantID + "/oauth2/v2.0/token"

	cert, privateKey, err := parseAzureCertAndKey(certPEM)
	if err != nil {
		return "", fmt.Errorf("Azure AD credential: %v", err)
	}

	assertion, err := buildAzureADClientAssertion(clientID, tokenURL, cert, privateKey)
	if err != nil {
		return "", fmt.Errorf("Azure AD credential: %v", err)
	}

	data := url.Values{
		"grant_type":            {"client_credentials"},
		"client_id":             {clientID},
		"scope":                 {scope},
		"client_assertion_type": {"urn:ietf:params:oauth:client-assertion-type:jwt-bearer"},
		"client_assertion":      {assertion},
	}

	resp, err := (&http.Client{Timeout: timeout}).PostForm(tokenURL, data)
	if err != nil {
		return "", fmt.Errorf("Azure AD credential: token fetch failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		errBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Azure AD credential: token endpoint returned HTTP %d: %s", resp.StatusCode, string(errBody))
	}

	var result struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
		ErrorDesc   string `json:"error_description"`
	}
	if err = json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("Azure AD credential: token response parse failed: %v", err)
	}
	if result.Error != "" {
		return "", fmt.Errorf("Azure AD credential: token error %q: %s", result.Error, result.ErrorDesc)
	}
	return result.AccessToken, nil
}

// buildAzureADClientAssertion creates a signed JWT for the client_assertion
// parameter.  The x5t header field carries the SHA-1 thumbprint of the
// certificate so Azure AD can look up the registered public key.
func buildAzureADClientAssertion(clientID, audience string, cert *x509.Certificate, key *rsa.PrivateKey) (string, error) {
	// x5t: base64url(SHA-1(DER-encoded certificate))
	thumbprint := sha1.Sum(cert.Raw)
	x5t := base64.RawURLEncoding.EncodeToString(thumbprint[:])

	jwtHeader := base64.RawURLEncoding.EncodeToString([]byte(
		fmt.Sprintf(`{"alg":"RS256","typ":"JWT","x5t":%q}`, x5t),
	))

	now := time.Now().Unix()
	jtiBytes := make([]byte, 16)
	if _, err := rand.Read(jtiBytes); err != nil {
		return "", fmt.Errorf("failed to generate jti: %v", err)
	}
	jwtClaims := base64.RawURLEncoding.EncodeToString([]byte(fmt.Sprintf(
		`{"aud":%q,"exp":%d,"iss":%q,"jti":%q,"nbf":%d,"sub":%q}`,
		audience, now+600, clientID, hex.EncodeToString(jtiBytes), now, clientID,
	)))

	signingInput := jwtHeader + "." + jwtClaims
	h := sha256.New()
	h.Write([]byte(signingInput))
	sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, h.Sum(nil))
	if err != nil {
		return "", fmt.Errorf("JWT signing failed: %v", err)
	}
	return signingInput + "." + base64.RawURLEncoding.EncodeToString(sig), nil
}

// parseAzureCertAndKey decodes a PEM bundle containing exactly one CERTIFICATE
// block and one private key block.  Two PEM key types are recognised:
//
//   - "RSA PRIVATE KEY" — unencrypted PKCS#1 RSA key
//   - "PRIVATE KEY"     — unencrypted PKCS#8 RSA key
//
// PEM-level key encryption ("ENCRYPTED PRIVATE KEY" / legacy Proc-Type headers)
// is not supported: the deprecated x509.IsEncryptedPEMBlock / DecryptPEMBlock
// APIs were removed in Go 1.16 as cryptographically insecure, and there is no
// standard-library replacement.  Keys that must be password-protected should be
// stored as PKCS#12 and converted to unencrypted PEM before use, or the
// credential store's own encryption should be relied on.
func parseAzureCertAndKey(certPEM string) (*x509.Certificate, *rsa.PrivateKey, error) {

	var cert *x509.Certificate
	var privateKey *rsa.PrivateKey

	data := []byte(certPEM)
	for {
		var block *pem.Block
		block, data = pem.Decode(data)
		if block == nil {
			break
		}
		switch block.Type {
		case "CERTIFICATE":
			if cert != nil {
				continue // use the first certificate in the bundle
			}
			c, err := x509.ParseCertificate(block.Bytes)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to parse certificate: %v", err)
			}
			cert = c
		case "RSA PRIVATE KEY": // unencrypted PKCS#1
			if privateKey != nil {
				continue // use the first key in the bundle
			}
			k, err := x509.ParsePKCS1PrivateKey(block.Bytes)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to parse RSA private key: %v", err)
			}
			privateKey = k
		case "PRIVATE KEY": // unencrypted PKCS#8
			if privateKey != nil {
				continue // use the first key in the bundle
			}
			raw, err := x509.ParsePKCS8PrivateKey(block.Bytes)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to parse PKCS8 private key: %v", err)
			}
			k, ok := raw.(*rsa.PrivateKey)
			if !ok {
				return nil, nil, fmt.Errorf("PKCS8 private key is not RSA")
			}
			privateKey = k
		}
	}

	if cert == nil {
		return nil, nil, fmt.Errorf("no certificate found in PEM data")
	}
	if privateKey == nil {
		return nil, nil, fmt.Errorf("no private key found in PEM data")
	}
	return cert, privateKey, nil
}

// ─── Azure SAS ────────────────────────────────────────────────────────────────

// applyAzureSASPayload wraps the client with a transport that appends the
// Shared Access Signature parameters to every request URL's query string.
// When Endpoint is set the transport also rewrites the request's scheme+host
// to that base URL, enabling Azurite or Azure Stack targets.
//
// The sharedAccessSignature value is a pre-built SAS query string (e.g.
// "sv=2021-06-08&ss=b&srt=o&sp=r&...&sig=...").  Its key=value pairs are
// merged into the outgoing request URL so each request carries the SAS token.
//
// Spec references:
//   - Azure Blob Storage service SAS:
//     https://learn.microsoft.com/en-us/rest/api/storageservices/create-service-sas
//   - Azure account SAS:
//     https://learn.microsoft.com/en-us/rest/api/storageservices/create-account-sas
func applyAzureSASPayload(cred *cbauth.Credential, context Context) (http.Client, http.Header, error) {
	p := cred.AzureSAS

	if p.SharedAccessSignature == "" {
		return http.Client{}, http.Header{}, fmt.Errorf("Azure SAS credential: sharedAccessSignature is required")
	}

	var endpointURL *url.URL
	if p.Endpoint != "" {
		var err error
		endpointURL, err = util.ParseAndValidateURL(p.Endpoint)
		if err != nil {
			return http.Client{}, http.Header{}, fmt.Errorf("Azure SAS credential: invalid endpoint %q: %v", p.Endpoint, err)
		}
		if util.IsRestrictedURL(endpointURL) {
			return http.Client{}, http.Header{}, fmt.Errorf("Azure SAS credential: service endpoint blocked - access restricted: %v", endpointURL.String())
		}
	}

	client, err := GetDefaultHttpClient(context)
	if err != nil {
		return http.Client{}, http.Header{}, err
	}
	client.Transport = &azureSASTransport{
		base:     client.Transport,
		sas:      p.SharedAccessSignature,
		endpoint: endpointURL,
	}
	return client, http.Header{}, nil
}

type azureSASTransport struct {
	base http.RoundTripper
	sas  string
	// endpoint overrides the request's scheme+host when targeting Azurite
	// or Azure Stack.  nil means use the caller's URL unchanged.
	endpoint *url.URL
}

func (t *azureSASTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	clone := req.Clone(req.Context())
	if t.endpoint != nil {
		clone.URL.Scheme = t.endpoint.Scheme
		clone.URL.Host = t.endpoint.Host
	}
	q := clone.URL.Query()

	// The sharedAccessSignature value is a complete query string
	// (e.g. "sv=2021-06-08&ss=b&...&sig=..."); merge its params into the URL.
	sasParams, err := url.ParseQuery(t.sas)
	if err != nil {
		q.Set("sig", t.sas)
	} else {
		for k, vals := range sasParams {
			q[k] = vals
		}
	}
	clone.URL.RawQuery = q.Encode()
	return t.base.RoundTrip(clone)
}

// ─── Azure Managed Identity ───────────────────────────────────────────────────

// applyAzureManagedPayload fetches an access token from the Azure Instance
// Metadata Service (IMDS) and returns it as a Bearer Authorization header.
// The resource is derived from the target URL host.
//
// IMDS is a non-routable HTTP endpoint (169.254.169.254) available only from
// within Azure VMs/containers.  No outbound credential material is required;
// access is granted by the VM's assigned managed identity.  An optional
// ManagedIdentityID selects a specific user-assigned identity when multiple
// identities are attached to the VM.
//
// Spec references:
//   - Azure IMDS token acquisition:
//     https://learn.microsoft.com/en-us/entra/identity/managed-identities-azure-resources/how-to-use-vm-token
//   - IMDS REST API reference:
//     https://learn.microsoft.com/en-us/azure/virtual-machines/instance-metadata-service
func applyAzureManagedPayload(cred *cbauth.Credential, urlObj *url.URL, context Context) (http.Client, http.Header, error) {
	p := cred.AzureManaged

	if p.Endpoint != "" {
		endpointURL, err := util.ParseAndValidateURL(p.Endpoint)
		if err != nil {
			return http.Client{}, http.Header{}, fmt.Errorf("Azure Managed Identity credential: invalid endpoint %q: %v", p.Endpoint, err)
		}
		if util.IsRestrictedURL(endpointURL) {
			return http.Client{}, http.Header{}, fmt.Errorf("Azure Managed Identity credential: service endpoint blocked - access restricted: %v", endpointURL.String())
		}
	}

	resource := "https://" + urlObj.Hostname()
	tokenTimeout := _tokenFetchTimeout
	if qt := context.GetTimeout(); qt > 0 && qt < tokenTimeout {
		tokenTimeout = qt
	}
	token, err := fetchAzureManagedToken(p.ManagedIdentityID, p.Endpoint, resource, tokenTimeout)
	if err != nil {
		return http.Client{}, http.Header{}, err
	}

	client, err := GetDefaultHttpClient(context)
	if err != nil {
		return http.Client{}, http.Header{}, err
	}
	header := http.Header{}
	header.Set("Authorization", "Bearer "+token)
	return client, header, nil
}

func fetchAzureManagedToken(managedIdentityID, customEndpoint, resource string, timeout time.Duration) (string, error) {
	imdsBase := "http://169.254.169.254/metadata/identity/oauth2/token"
	if customEndpoint != "" {
		imdsBase = strings.TrimRight(customEndpoint, "/")
	}

	params := url.Values{
		"api-version": {"2018-02-01"},
		"resource":    {resource},
	}
	if managedIdentityID != "" {
		params.Set("client_id", managedIdentityID)
	}

	req, err := http.NewRequest("GET", imdsBase+"?"+params.Encode(), nil)
	if err != nil {
		return "", fmt.Errorf("Azure Managed Identity credential: failed to build IMDS request: %v", err)
	}
	req.Header.Set("Metadata", "true")

	resp, err := (&http.Client{Timeout: timeout}).Do(req)
	if err != nil {
		return "", fmt.Errorf("Azure Managed Identity credential: IMDS token fetch failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		errBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Azure Managed Identity credential: IMDS returned HTTP %d: %s", resp.StatusCode, string(errBody))
	}

	var result struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
		ErrorDesc   string `json:"error_description"`
	}
	if err = json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("Azure Managed Identity credential: IMDS response parse failed: %v", err)
	}
	if result.Error != "" {
		return "", fmt.Errorf("Azure Managed Identity credential: IMDS error %q: %s", result.Error, result.ErrorDesc)
	}
	return result.AccessToken, nil
}

// ─── GCP ──────────────────────────────────────────────────────────────────────

// applyGCPPayload handles both GCP authentication modes:
//   - Service-account mode (jsonCredentials set): exchanges a signed JWT for
//     an OAuth2 access token, returned as a Bearer Authorization header.
//   - HMAC mode (accessKeyId + secretAccessKey set): wraps the client with
//     the same SigV4-signing transport used for AWS (GCS HMAC-compatible).
//
// HMAC mode reuses awsSigV4Transport with service="s3" because GCS exposes an
// S3-compatible XML API that accepts standard AWS Signature Version 4.  GCP
// HMAC keys (not service-account keys) are used as the access/secret pair.
//
// Spec references:
//   - GCP service account authentication (JWT exchange):
//     https://developers.google.com/identity/protocols/oauth2/service-account
//   - GCS HMAC keys for S3-compatible interoperability:
//     https://cloud.google.com/storage/docs/authentication/hmackeys
//   - GCS S3-compatible XML API:
//     https://cloud.google.com/storage/docs/interoperability
func applyGCPPayload(cred *cbauth.Credential, context Context) (http.Client, http.Header, error) {
	p := cred.GCP
	switch {
	case p.JSONCredentials != "":
		return applyGCPServiceAccount(p.JSONCredentials, context)
	case p.AccessKeyID != "":
		var endpointURL *url.URL
		if p.Endpoint != "" {
			var err error
			endpointURL, err = util.ParseAndValidateURL(p.Endpoint)
			if err != nil {
				return http.Client{}, http.Header{}, fmt.Errorf("GCP credential: invalid endpoint %q: %v", p.Endpoint, err)
			}
			if util.IsRestrictedURL(endpointURL) {
				return http.Client{}, http.Header{}, fmt.Errorf("GCP credential: service endpoint blocked - access restricted: %v", endpointURL.String())
			}
		}
		client, err := GetDefaultHttpClient(context)
		if err != nil {
			return http.Client{}, http.Header{}, err
		}
		client.Transport = &awsSigV4Transport{
			base:            client.Transport,
			accessKeyID:     p.AccessKeyID,
			secretAccessKey: p.SecretAccessKey,
			region:          p.Region,
			service:         "s3",
			endpoint:        endpointURL,
		}
		return client, http.Header{}, nil
	default:
		return http.Client{}, http.Header{}, fmt.Errorf("GCP credential: either jsonCredentials or accessKeyId must be provided")
	}
}

type gcpServiceAccountKey struct {
	PrivateKey  string `json:"private_key"`
	ClientEmail string `json:"client_email"`
	TokenURI    string `json:"token_uri"`
}

// applyGCPServiceAccount exchanges a GCP service-account key (from a JSON
// credentials file) for a short-lived OAuth2 access token using the JWT bearer
// grant.  The signed JWT is posted to the token_uri from the JSON file
// (default: https://oauth2.googleapis.com/token) and the returned access token
// is sent as a Bearer Authorization header.
//
// Spec references:
//   - JWT bearer grant type:  RFC 7523  https://datatracker.ietf.org/doc/html/rfc7523
//   - GCP service account flow: https://developers.google.com/identity/protocols/oauth2/service-account#jwt-auth
func applyGCPServiceAccount(jsonCreds string, context Context) (http.Client, http.Header, error) {
	var sa gcpServiceAccountKey
	if err := json.Unmarshal([]byte(jsonCreds), &sa); err != nil {
		return http.Client{}, http.Header{}, fmt.Errorf("GCP credential: invalid jsonCredentials: %v", err)
	}
	if sa.ClientEmail == "" {
		return http.Client{}, http.Header{}, fmt.Errorf("GCP credential: client_email is required in jsonCredentials")
	}
	// Validate the token URI from the service-account JSON before use.
	// A credential with a malicious token_uri (e.g. /diag/eval) must be blocked
	// the same way explicit endpoint overrides are blocked in HMAC/Azure modes.
	tokenURI := sa.TokenURI
	if tokenURI == "" {
		tokenURI = "https://oauth2.googleapis.com/token"
	}
	parsedTokenURI, err := util.ParseAndValidateURL(tokenURI)
	if err != nil {
		return http.Client{}, http.Header{}, fmt.Errorf("GCP credential: invalid token_uri %q: %v", tokenURI, err)
	}
	if util.IsRestrictedURL(parsedTokenURI) {
		return http.Client{}, http.Header{}, fmt.Errorf("GCP credential: service endpoint blocked - access restricted: %v", tokenURI)
	}
	tokenTimeout := _tokenFetchTimeout
	if qt := context.GetTimeout(); qt > 0 && qt < tokenTimeout {
		tokenTimeout = qt
	}
	token, err := fetchGCPServiceAccountToken(sa, tokenTimeout)
	if err != nil {
		return http.Client{}, http.Header{}, err
	}
	client, err := GetDefaultHttpClient(context)
	if err != nil {
		return http.Client{}, http.Header{}, err
	}
	header := http.Header{}
	header.Set("Authorization", "Bearer "+token)
	return client, header, nil
}

func fetchGCPServiceAccountToken(sa gcpServiceAccountKey, timeout time.Duration) (string, error) {
	block, _ := pem.Decode([]byte(sa.PrivateKey))
	if block == nil {
		return "", fmt.Errorf("GCP credential: failed to decode private key PEM")
	}
	raw, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return "", fmt.Errorf("GCP credential: failed to parse private key: %v", err)
	}
	rsaKey, ok := raw.(*rsa.PrivateKey)
	if !ok {
		return "", fmt.Errorf("GCP credential: private key is not RSA")
	}

	now := time.Now().Unix()
	scope := "https://www.googleapis.com/auth/cloud-platform"
	tokenURI := sa.TokenURI
	if tokenURI == "" {
		tokenURI = "https://oauth2.googleapis.com/token"
	}

	jwtHeader := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256","typ":"JWT"}`))
	claims := fmt.Sprintf(
		`{"iss":%q,"scope":%q,"aud":%q,"iat":%d,"exp":%d}`,
		sa.ClientEmail, scope, tokenURI, now, now+300, // 5 min: JWT is exchanged immediately; 1 hr max is allowed but unnecessarily long
	)
	jwtClaims := base64.RawURLEncoding.EncodeToString([]byte(claims))
	signingInput := jwtHeader + "." + jwtClaims

	h := sha256.New()
	h.Write([]byte(signingInput))
	sig, err := rsa.SignPKCS1v15(rand.Reader, rsaKey, crypto.SHA256, h.Sum(nil))
	if err != nil {
		return "", fmt.Errorf("GCP credential: JWT signing failed: %v", err)
	}
	jwt := signingInput + "." + base64.RawURLEncoding.EncodeToString(sig)

	data := url.Values{
		"grant_type": {"urn:ietf:params:oauth:grant-type:jwt-bearer"},
		"assertion":  {jwt},
	}
	resp, err := (&http.Client{Timeout: timeout}).PostForm(tokenURI, data)
	if err != nil {
		return "", fmt.Errorf("GCP credential: token fetch failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		errBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("GCP credential: token endpoint returned HTTP %d: %s", resp.StatusCode, string(errBody))
	}

	var result struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
		ErrorDesc   string `json:"error_description"`
	}
	if err = json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("GCP credential: token response parse failed: %v", err)
	}
	if result.Error != "" {
		return "", fmt.Errorf("GCP credential: token error %q: %s", result.Error, result.ErrorDesc)
	}
	return result.AccessToken, nil
}

// UnwrapTransport peels through credential-wrapping RoundTrippers
// (awsSigV4Transport, azureSharedKeyTransport, azureSASTransport)
// to find the underlying *http.Transport. Returns nil if no
// *http.Transport is found in the chain.
func UnwrapTransport(rt http.RoundTripper) *http.Transport {
	for {
		switch t := rt.(type) {
		case *http.Transport:
			return t
		case *awsSigV4Transport:
			rt = t.base
		case *azureSharedKeyTransport:
			rt = t.base
		case *azureSASTransport:
			rt = t.base
		default:
			return nil
		}
	}
}

// ─── Shared crypto helpers ────────────────────────────────────────────────────

func hmacSHA256(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}

func sha256HexBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func sha256HexString(s string) string {
	return sha256HexBytes([]byte(s))
}
