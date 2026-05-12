//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of the
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package external

import (
	"bytes"
	go_context "context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/storage"
	"database/sql"
	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/ipc"
	"github.com/apache/arrow-go/v18/arrow/memory"
	pqfile "github.com/apache/arrow-go/v18/parquet/file"
	"github.com/apache/arrow-go/v18/parquet/pqarrow"
	"github.com/apache/iceberg-go"

	"github.com/apache/iceberg-go/catalog"
	"github.com/apache/iceberg-go/catalog/glue"
	"github.com/apache/iceberg-go/catalog/rest"
	icesql "github.com/apache/iceberg-go/catalog/sql"
	iceio "github.com/apache/iceberg-go/io"
	"github.com/apache/iceberg-go/table"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/couchbase/cbauth"
	"github.com/couchbase/query/extparams"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/util"
	"github.com/hamba/avro/v2/ocf"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	"github.com/scritchley/orc"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	gOption "google.golang.org/api/option"
)

const (
	_ICEBERG_RESULT_CHAN_PER_WORKER   = 100
	_ICEBERG_IDLE_CONNS_PER_HOST_CPUS = 4
)

var (
	_icebergTransportOnce sync.Once
	_icebergHTTPTransport *http.Transport
)

// icebergTransport returns a shared *http.Transport for all iceberg HTTP calls
// (REST catalogs, S3, GCS). Cloned from http.DefaultTransport with MaxIdleConnsPerHost
// set to 4*NumCPU so concurrent file downloads share idle sockets per endpoint.
func icebergTransport() *http.Transport {
	_icebergTransportOnce.Do(func() {
		t := http.DefaultTransport.(*http.Transport).Clone()
		t.MaxIdleConnsPerHost = _ICEBERG_IDLE_CONNS_PER_HOST_CPUS * util.NumCPU()
		t.IdleConnTimeout = IcebergScanTimeout
		_icebergHTTPTransport = t
	})
	return _icebergHTTPTransport
}

type Scanner struct {
	catalog         catalog.Catalog
	table           *table.Table
	tableIdent      []string
	tableName       string
	databaseName    string
	snapshotID      *int64
	snapshotAsOf    *int64
	selectedFields  []string
	limit           int64
	filterPushdown  *FilterPushdown
	awsConfig       *aws.Config
	sourceType      string             // Track which catalog source type we're using
	parallelScans   int                // Scan parallelism override (0 = use default 1)
	collectionCred  *cbauth.Credential // Credential for reading data files
	decimalToDouble bool               // When true, decimal columns are converted to float64
}

type ScanOptions struct {
	Database           string
	Table              string
	SnapshotID         *int64
	SnapshotAsOf       *int64
	SelectedFields     []string
	CaseSensitive      bool
	Limit              int64
	AwsConfig          *aws.Config
	Filters            []IcebergFilter
	SourceType         string
	URI                string
	Warehouse          string
	SigV4SigningRegion string
	SigV4SigningName   string
	Credential         string
	CatalogCred        *cbauth.Credential
	CollectionCred     *cbauth.Credential
	QuotaProjectID     string
	ParallelScans      int  // Scan parallelism override (defaults to 1 if not set)
	DecimalToDouble    bool // When true, Decimal128/256 columns are converted to float64 instead of string
	SQLDialect         string
}

// NewAWSConfig creates an AWS config from credentials and region
// If region is empty, AWS SDK will auto-detect the region (works for AWS_GLUE)
func NewAWSConfig(accessKeyID, secretAccessKey, sessionToken, region string) (*aws.Config, error) {
	if accessKeyID == "" || secretAccessKey == "" {
		return nil, fmt.Errorf("access key ID and secret access key are required")
	}

	cfgProvider := credentials.NewStaticCredentialsProvider(accessKeyID, secretAccessKey, sessionToken)

	var cfg aws.Config
	var err error

	sharedHTTPClient := &http.Client{Transport: icebergTransport()}
	if region != "" {
		// Load config with explicit region
		cfg, err = awsconfig.LoadDefaultConfig(go_context.Background(),
			awsconfig.WithCredentialsProvider(cfgProvider),
			awsconfig.WithRegion(region),
			awsconfig.WithHTTPClient(sharedHTTPClient),
		)
	} else {
		// Load config without region (AWS SDK will auto-detect for AWS_GLUE)
		cfg, err = awsconfig.LoadDefaultConfig(go_context.Background(),
			awsconfig.WithCredentialsProvider(cfgProvider),
			awsconfig.WithHTTPClient(sharedHTTPClient),
		)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	return &cfg, nil
}

// IsAWSSource checks if the source type is AWS-based
func IsAWSSource(source string) bool {
	return source == extparams.CatalogSourceAWSGlue ||
		source == extparams.CatalogSourceAWSGlueRest ||
		source == extparams.CatalogSourceS3Tables
}

// GetAWSConfig creates AWS config from credential for AWS-based sources
// Returns nil config for non-AWS sources
// For AWS_GLUE_REST and S3_TABLES, region is required
// For AWS_GLUE, region is optional (AWS SDK auto-detects)
func GetAWSConfig(source string, cred *cbauth.Credential, sigV4SigningRegion string) (*aws.Config, error) {
	if !IsAWSSource(source) {
		return nil, nil
	}

	if cred == nil || cred.Type != "aws" || cred.AWS == nil {
		return nil, fmt.Errorf("AWS credential not found for source: %s", source)
	}

	awsCreds := cred.AWS
	region := sigV4SigningRegion
	if region == "" {
		region = awsCreds.Region
	}

	// For REST-based sources, region is required
	if (source == extparams.CatalogSourceAWSGlueRest || source == extparams.CatalogSourceS3Tables) && region == "" {
		return nil, fmt.Errorf("AWS region not found in catalog metadata or credential for source: %s", source)
	}

	return NewAWSConfig(awsCreds.AccessKeyID, awsCreds.SecretAccessKey, awsCreds.SessionToken, region)
}

// biglakeTransport handles BigLake Metastore specifics:
//  1. Uses Google ADC or a provided token source for proper OAuth2 authentication
//  2. Injects X-Goog-User-Project header for GCP quota billing
type biglakeTransport struct {
	base        http.RoundTripper
	projectID   string
	tokenSource oauth2.TokenSource
}

func (t *biglakeTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	if t.projectID != "" {
		req.Header.Set("X-Goog-User-Project", t.projectID)
	}
	if t.tokenSource != nil {
		token, err := t.tokenSource.Token()
		if err != nil {
			return nil, fmt.Errorf("biglakeTransport: failed to get GCP access token: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	}
	return t.base.RoundTrip(req)
}

// basicAuthTransport injects HTTP Basic Authentication on every request.
type basicAuthTransport struct {
	base     http.RoundTripper
	username string
	password string
}

func (t *basicAuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.SetBasicAuth(t.username, t.password)
	return t.base.RoundTrip(req)
}

// iceberg-go FileIO property keys for each storage backend.
const (
	_adlsSharedKeyAccountName = "adls.auth.shared-key.account.name"
	_adlsSharedKeyAccountKey  = "adls.auth.shared-key.account.key"
	_adlsSasTokenPrefix       = "adls.sas-token."
	_s3AccessKeyID            = "s3.access-key-id"
	_s3SecretAccessKey        = "s3.secret-access-key"
	_s3SessionToken           = "s3.session-token"
	_s3Region                 = "s3.region"
	_s3EndpointURL            = "s3.endpoint"
	_gcsJSONKey               = "gcs.jsonkey"
	_gcsEndpoint              = "gcs.endpoint"
	_azureTokenFetchTimeout   = 30 * time.Second
	_azureADAuthority         = "https://login.microsoftonline.com"
	_azureIMDSEndpoint        = "http://169.254.169.254/metadata/identity/oauth2/token"
)

// restAuthExtras carries source-type-specific overrides that influence catalog auth.
// SigV4SigningRegion / SigV4SigningName let an AWS-credentialed REST catalog target
// a specific AWS service (e.g. "glue" or "s3tables"); QuotaProjectID adds the
// X-Goog-User-Project header on GCP-authenticated catalogs (required by BigLake).
type restAuthExtras struct {
	SigV4SigningRegion string
	SigV4SigningName   string
	QuotaProjectID     string
}

// buildRESTCatalogAuthOpts builds REST catalog options from any supported credential
// type. Used by REST/NESSIE_REST/UNITY_CATALOG so a REST catalog can be fronted by
// any auth scheme — bearer/basic/mTLS (http), Azure AD / Managed Identity OAuth
// bearer, AWS SigV4, or GCP service-account bearer. azureShared and azureSas are
// storage-protocol credentials and are not valid for REST catalog authentication.
func buildRESTCatalogAuthOpts(ctx go_context.Context, cred *cbauth.Credential, uri string, extra restAuthExtras) ([]rest.Option, error) {
	if cred == nil {
		return nil, nil
	}
	switch cred.Type {
	case "http":
		return buildNessieHTTPAuthOpts(cred)

	case "azureAd":
		if cred.AzureAD == nil {
			return nil, fmt.Errorf("azureAd credential has no payload")
		}
		p := cred.AzureAD
		if p.TenantID == "" || p.ClientID == "" {
			return nil, fmt.Errorf("azureAd credential requires tenantId and clientId")
		}
		u, err := url.Parse(uri)
		if err != nil || u.Hostname() == "" {
			return nil, fmt.Errorf("invalid catalog URI %q for azureAd scope derivation", uri)
		}
		scope := "https://" + u.Hostname() + "/.default"
		token, err := fetchAzureADTokenForCatalog(p.TenantID, p.ClientID, p.ClientSecret, p.Endpoint, scope)
		if err != nil {
			return nil, err
		}
		logging.Debugf("buildRESTCatalogAuthOpts: Azure AD OAuth bearer (scope=%s)", scope)
		return []rest.Option{rest.WithOAuthToken(token)}, nil

	case "azureManaged":
		if cred.AzureManaged == nil {
			return nil, fmt.Errorf("azureManaged credential has no payload")
		}
		p := cred.AzureManaged
		u, err := url.Parse(uri)
		if err != nil || u.Hostname() == "" {
			return nil, fmt.Errorf("invalid catalog URI %q for azureManaged resource derivation", uri)
		}
		resource := "https://" + u.Hostname() + "/"
		token, err := fetchAzureManagedTokenForCatalog(p.ManagedIdentityID, p.Endpoint, resource)
		if err != nil {
			return nil, err
		}
		logging.Debugf("buildRESTCatalogAuthOpts: Azure Managed Identity bearer (resource=%s)", resource)
		return []rest.Option{rest.WithOAuthToken(token)}, nil

	case "aws":
		if cred.AWS == nil {
			return nil, fmt.Errorf("aws catalog credential has no payload")
		}
		c := cred.AWS
		region := extra.SigV4SigningRegion
		if region == "" {
			region = c.Region
		}
		cfg, err := NewAWSConfig(c.AccessKeyID, c.SecretAccessKey, c.SessionToken, region)
		if err != nil {
			return nil, fmt.Errorf("failed to build AWS config: %w", err)
		}
		// Default SigV4 lets iceberg-go infer the service from the URL host.
		// A caller can override by specifying SigV4SigningName (e.g. "glue",
		// "s3tables") so the same generic REST type can target service-specific
		// signing endpoints.
		if extra.SigV4SigningName != "" {
			logging.Debugf("buildRESTCatalogAuthOpts: AWS SigV4 (region=%s, service=%s)", region, extra.SigV4SigningName)
			return []rest.Option{rest.WithSigV4RegionSvc(region, extra.SigV4SigningName), rest.WithAwsConfig(*cfg)}, nil
		}
		logging.Debugf("buildRESTCatalogAuthOpts: AWS SigV4 (region=%s)", region)
		return []rest.Option{rest.WithSigV4(), rest.WithAwsConfig(*cfg)}, nil

	case "gcp":
		if cred.GCP == nil {
			return nil, fmt.Errorf("gcp catalog credential has no payload")
		}
		scopes := []string{"https://www.googleapis.com/auth/cloud-platform"}
		var ts oauth2.TokenSource
		if cred.GCP.AccessKeyID != "" && cred.GCP.SecretAccessKey != "" {
			ts = (&oauth2.Config{
				ClientID:     cred.GCP.AccessKeyID,
				ClientSecret: cred.GCP.SecretAccessKey,
				Endpoint:     google.Endpoint,
				Scopes:       scopes,
			}).TokenSource(ctx, &oauth2.Token{})
		} else {
			creds, err := google.CredentialsFromJSON(ctx, []byte(cred.GCP.JSONCredentials), scopes...)
			if err != nil {
				return nil, fmt.Errorf("failed to parse GCP service-account credential: %w", err)
			}
			if _, err := creds.TokenSource.Token(); err != nil {
				return nil, fmt.Errorf("failed to obtain GCP access token: %w", err)
			}
			ts = creds.TokenSource
		}
		ts = oauth2.ReuseTokenSource(nil, ts)
		logging.Debugf("buildRESTCatalogAuthOpts: GCP OAuth bearer (quotaProjectID=%q)", extra.QuotaProjectID)
		// Use a transport that refreshes the bearer token on each request rather
		// than capturing a single static token (which would expire mid-scan).
		// QuotaProjectID injects X-Goog-User-Project for BigLake-style quota routing.
		return []rest.Option{rest.WithCustomTransport(&biglakeTransport{
			base:        icebergTransport(),
			projectID:   extra.QuotaProjectID,
			tokenSource: ts,
		})}, nil

	default:
		return nil, fmt.Errorf("credential type %q is not valid for REST catalog authentication", cred.Type)
	}
}

// buildRESTStorageProps converts a collection credential into iceberg-go FileIO
// property keys (adls.*, s3.*, gcs.*). The returned map is forwarded to the REST
// catalog via rest.WithAdditionalProps so iceberg-go's LoadFSFunc picks up the
// right auth when opening data files. Catalogs that vend credentials (e.g. Unity
// Catalog, Polaris) supply these automatically in LoadTableResponse — these props
// are the fallback for catalogs that do not vend credentials.
func buildRESTStorageProps(cred *cbauth.Credential) (iceberg.Properties, error) {
	if cred == nil {
		return nil, nil
	}
	switch cred.Type {
	case "azureShared":
		if cred.AzureShared == nil {
			return nil, fmt.Errorf("azureShared credential has no payload")
		}
		p := cred.AzureShared
		if p.AccountName == "" || p.AccountKey == "" {
			return nil, fmt.Errorf("azureShared credential requires accountName and accountKey")
		}
		return iceberg.Properties{
			_adlsSharedKeyAccountName: p.AccountName,
			_adlsSharedKeyAccountKey:  p.AccountKey,
		}, nil
	case "azureSas":
		if cred.AzureSAS == nil {
			return nil, fmt.Errorf("azureSas credential has no payload")
		}
		p := cred.AzureSAS
		if p.AccountName == "" || p.SharedAccessSignature == "" {
			return nil, fmt.Errorf("azureSas credential requires accountName and sharedAccessSignature")
		}
		return iceberg.Properties{
			_adlsSasTokenPrefix + p.AccountName: p.SharedAccessSignature,
		}, nil
	case "azureAd", "azureManaged":
		// iceberg-go's createAzureBucket falls through to azidentity.NewDefaultAzureCredential
		// when no shared-key / sas-token / connection-string props are present.
		return nil, nil
	case "aws":
		if cred.AWS == nil {
			return nil, fmt.Errorf("aws credential has no payload")
		}
		c := cred.AWS
		props := iceberg.Properties{
			_s3AccessKeyID:     c.AccessKeyID,
			_s3SecretAccessKey: c.SecretAccessKey,
		}
		if c.SessionToken != "" {
			props[_s3SessionToken] = c.SessionToken
		}
		if c.Region != "" {
			props[_s3Region] = c.Region
		}
		if c.Endpoint != "" {
			props[_s3EndpointURL] = c.Endpoint
		}
		return props, nil
	case "gcp":
		if cred.GCP == nil {
			return nil, fmt.Errorf("gcp credential has no payload")
		}
		props := iceberg.Properties{}
		if cred.GCP.JSONCredentials != "" {
			props[_gcsJSONKey] = cred.GCP.JSONCredentials
		}
		if cred.GCP.Endpoint != "" {
			props[_gcsEndpoint] = cred.GCP.Endpoint
		}
		return props, nil
	default:
		return nil, fmt.Errorf("credential type %q is not a recognized storage credential", cred.Type)
	}
}

// fetchAzureADTokenForCatalog performs the OAuth2 client-credentials flow against
// Azure AD and returns an access token suitable for a Bearer Authorization header.
func fetchAzureADTokenForCatalog(tenantID, clientID, clientSecret, endpoint, scope string) (string, error) {
	authority := _azureADAuthority
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

	resp, err := (&http.Client{Timeout: _azureTokenFetchTimeout}).PostForm(tokenURL, data)
	if err != nil {
		return "", fmt.Errorf("Azure AD token fetch failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Azure AD token endpoint returned HTTP %d: %s", resp.StatusCode, string(body))
	}
	var result struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
		ErrorDesc   string `json:"error_description"`
	}
	if err = json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("Azure AD token response parse failed: %w", err)
	}
	if result.Error != "" {
		return "", fmt.Errorf("Azure AD token error %q: %s", result.Error, result.ErrorDesc)
	}
	return result.AccessToken, nil
}

// fetchAzureManagedTokenForCatalog acquires an access token from the Azure Instance
// Metadata Service (IMDS) using managed identity, optionally scoped to a specific client ID.
func fetchAzureManagedTokenForCatalog(managedIdentityID, customEndpoint, resource string) (string, error) {
	imdsBase := _azureIMDSEndpoint
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
		return "", fmt.Errorf("Azure Managed Identity IMDS request build failed: %w", err)
	}
	req.Header.Set("Metadata", "true")
	resp, err := (&http.Client{Timeout: _azureTokenFetchTimeout}).Do(req)
	if err != nil {
		return "", fmt.Errorf("Azure Managed Identity IMDS token fetch failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Azure Managed Identity IMDS returned HTTP %d: %s", resp.StatusCode, string(body))
	}
	var result struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
		ErrorDesc   string `json:"error_description"`
	}
	if err = json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("Azure Managed Identity IMDS response parse failed: %w", err)
	}
	if result.Error != "" {
		return "", fmt.Errorf("Azure Managed Identity IMDS error %q: %s", result.Error, result.ErrorDesc)
	}
	return result.AccessToken, nil
}

// sqlDialectFromURI derives the bun SQL dialect and Go driver name from the DSN URI
// scheme. An explicit override dialect (from the sqlDialect catalog parameter) takes
// precedence. Supported: postgres/postgresql → postgres (lib/pq), all else → sqlite
// (mattn/go-sqlite3).
func sqlDialectFromURI(uri, override string) (icesql.SupportedDialect, string, error) {
	if override != "" {
		d := icesql.SupportedDialect(strings.ToLower(override))
		switch d {
		case icesql.Postgres:
			return d, "postgres", nil
		case icesql.SQLite:
			return d, "sqlite3", nil
		default:
			return "", "", fmt.Errorf("unsupported sqlDialect '%s': valid values are postgres, sqlite", override)
		}
	}
	lower := strings.ToLower(uri)
	if strings.HasPrefix(lower, "postgres://") || strings.HasPrefix(lower, "postgresql://") {
		return icesql.Postgres, "postgres", nil
	}
	// sqlite: file path, sqlite://, or any non-postgres URI
	return icesql.SQLite, "sqlite3", nil
}

// injectSQLCredentials takes a DSN URI and an optional http/basic credential and
// returns a DSN with username and password injected.  If the credential is nil or
// the URI already carries user-info, the original DSN is returned unchanged.
func injectSQLCredentials(dsn string, cred *cbauth.Credential) (string, error) {
	if cred == nil {
		return dsn, nil
	}
	if cred.Type != "http" || cred.HTTP == nil {
		return "", fmt.Errorf("SQL catalog credential must be type 'http', got '%s'", cred.Type)
	}
	switch strings.ToLower(cred.HTTP.AuthScheme) {
	case "basic":
		// handled below
	case "bearer":
		return "", fmt.Errorf("SQL catalog does not support bearer token auth; use authScheme 'basic'")
	case "mtls":
		return "", fmt.Errorf("SQL catalog mTLS is not yet supported; use authScheme 'basic'")
	default:
		return "", fmt.Errorf("SQL catalog unsupported authScheme '%s'; use 'basic'", cred.HTTP.AuthScheme)
	}
	u, err := url.Parse(dsn)
	if err != nil {
		return "", fmt.Errorf("failed to parse SQL DSN as URL: %w", err)
	}
	// Only inject if the URI doesn't already carry credentials
	if u.User == nil || u.User.Username() == "" {
		u.User = url.UserPassword(cred.HTTP.Username, cred.HTTP.Password)
	}
	return u.String(), nil
}

// buildNessieHTTPAuthOpts converts an HTTP credential into REST catalog options for Nessie.
// Supported auth schemes: "bearer", "basic", "mtls".
func buildNessieHTTPAuthOpts(cred *cbauth.Credential) ([]rest.Option, error) {
	if cred == nil || cred.Type != "http" || cred.HTTP == nil {
		return nil, nil
	}
	h := cred.HTTP
	switch strings.ToLower(h.AuthScheme) {
	case "bearer":
		if h.Token == "" {
			return nil, fmt.Errorf("bearer auth scheme requires a non-empty token")
		}
		logging.Debugf("buildNessieHTTPAuthOpts: using bearer token authentication")
		return []rest.Option{rest.WithOAuthToken(h.Token)}, nil

	case "basic":
		if h.Username == "" {
			return nil, fmt.Errorf("basic auth scheme requires a non-empty username")
		}
		logging.Debugf("buildNessieHTTPAuthOpts: using basic authentication (user=%s)", h.Username)
		return []rest.Option{rest.WithCustomTransport(&basicAuthTransport{
			base:     icebergTransport(),
			username: h.Username,
			password: h.Password,
		})}, nil

	case "mtls":
		tlsCfg, err := buildMTLSTLSConfig(h.Certificate, h.PrivateKey, h.Passphrase, h.RootCertificate, h.SkipVerify)
		if err != nil {
			return nil, fmt.Errorf("failed to build mTLS config: %w", err)
		}
		logging.Debugf("buildNessieHTTPAuthOpts: using mTLS authentication")
		return []rest.Option{rest.WithTLSConfig(tlsCfg)}, nil

	default:
		logging.Warnf("buildNessieHTTPAuthOpts: unknown auth scheme '%s', using no authentication", h.AuthScheme)
		return nil, nil
	}
}

// buildMTLSTLSConfig constructs a *tls.Config from PEM-encoded content strings.
// Certificate, PrivateKey, and RootCertificate are PEM content (not file paths).
func buildMTLSTLSConfig(certPEM, keyPEM, passphrase, rootCertPEM string, skipVerify bool) (*tls.Config, error) {
	tlsCfg := &tls.Config{InsecureSkipVerify: skipVerify}

	if rootCertPEM != "" {
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM([]byte(rootCertPEM)) {
			return nil, fmt.Errorf("failed to parse root certificate PEM")
		}
		tlsCfg.RootCAs = pool
	}

	if certPEM != "" && keyPEM != "" {
		keyBytes := []byte(keyPEM)
		if passphrase != "" {
			var err error
			keyBytes, err = decryptPEMKey(keyBytes, []byte(passphrase))
			if err != nil {
				return nil, fmt.Errorf("failed to decrypt private key: %w", err)
			}
		}
		cert, err := tls.X509KeyPair([]byte(certPEM), keyBytes)
		if err != nil {
			return nil, fmt.Errorf("failed to load mTLS key pair: %w", err)
		}
		tlsCfg.Certificates = []tls.Certificate{cert}
	}

	return tlsCfg, nil
}

// decryptPEMKey decrypts a passphrase-protected PEM private key block.
func decryptPEMKey(keyPEM, passphrase []byte) ([]byte, error) {
	block, _ := pem.Decode(keyPEM)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block from private key")
	}
	//nolint:staticcheck // x509.DecryptPEMBlock is deprecated but still the standard approach
	if !x509.IsEncryptedPEMBlock(block) {
		return keyPEM, nil
	}
	decrypted, err := x509.DecryptPEMBlock(block, passphrase)
	if err != nil {
		return nil, fmt.Errorf("decryption failed: %w", err)
	}
	block.Bytes = decrypted
	block.Headers = nil
	return pem.EncodeToMemory(block), nil
}

func NewScanner(ctx go_context.Context, opts ScanOptions, cat catalog.Catalog) (*Scanner, error) {
	if opts.Database == "" {
		return nil, fmt.Errorf("database name is required")
	}
	if opts.Table == "" {
		return nil, fmt.Errorf("table name is required")
	}

	scanner := &Scanner{
		databaseName:    opts.Database,
		tableName:       opts.Table,
		tableIdent:      catalog.ToIdentifier(opts.Database + "." + opts.Table),
		snapshotID:      opts.SnapshotID,
		snapshotAsOf:    opts.SnapshotAsOf,
		selectedFields:  opts.SelectedFields,
		limit:           opts.Limit,
		awsConfig:       opts.AwsConfig,
		sourceType:      strings.ToUpper(opts.SourceType),
		parallelScans:   opts.ParallelScans,
		collectionCred:  opts.CollectionCred,
		decimalToDouble: opts.DecimalToDouble,
	}

	if cat == nil {
		awsCfg := opts.AwsConfig
		sourceType := strings.ToUpper(opts.SourceType)
		if awsCfg == nil {
			switch sourceType {
			case "BIGLAKE_METASTORE", "NESSIE", "REST", "NESSIE_REST", "UNITY_CATALOG", "SQL":
			default:
				return nil, fmt.Errorf("AWS config is required for source type '%s'", opts.SourceType)
			}
		}
		var awsCfgVal aws.Config
		if awsCfg != nil {
			awsCfgVal = *awsCfg
		}
		var err error
		cat, err = createCatalog(ctx, opts, awsCfgVal)
		if err != nil {
			return nil, fmt.Errorf("failed to create catalog for source type '%s': %w", opts.SourceType, err)
		}
	}
	scanner.catalog = cat

	if len(opts.Filters) > 0 {
		scanner.filterPushdown = &FilterPushdown{
			icebergFilters: make([]iceberg.BooleanExpression, 0),
			caseSensitive:  opts.CaseSensitive,
		}
		logging.Debugf("Iceberg Scanner: applying %d filter(s) for pushdown", len(opts.Filters))
		if err := scanner.filterPushdown.ApplyFilters(opts.Filters); err != nil {
			logging.Warnf("Iceberg Scanner: failed to apply filters: %v. Filters will be ignored.", err)
			scanner.filterPushdown = nil
		} else {
			logging.Debugf("Iceberg Scanner: filters applied successfully to pushdown")
		}
	}

	return scanner, nil
}

// createCatalog creates the appropriate iceberg catalog based on source type.
func createCatalog(ctx go_context.Context, opts ScanOptions, awsCfg aws.Config) (catalog.Catalog, error) {
	sourceType := strings.ToUpper(opts.SourceType)

	switch sourceType {
	case "AWS_GLUE", "":
		// Default: use native AWS Glue catalog
		logging.Debugf("createCatalog: creating AWS_GLUE catalog")
		return glue.NewCatalog(glue.WithAwsConfig(awsCfg)), nil

	case "AWS_GLUE_REST":
		// AWS Glue via Iceberg REST protocol with SigV4 signing (service=glue)
		if opts.URI == "" {
			return nil, fmt.Errorf("URI is required for AWS_GLUE_REST source type")
		}
		region := opts.SigV4SigningRegion
		if region == "" {
			region = awsCfg.Region
		}
		logging.Debugf("createCatalog: creating AWS_GLUE_REST catalog, uri=%s, region=%s", opts.URI, region)
		cat, err := rest.NewCatalog(ctx, "glue-rest", opts.URI,
			rest.WithAwsConfig(awsCfg),
			rest.WithSigV4RegionSvc(region, "glue"),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create AWS_GLUE_REST catalog: %w", err)
		}
		return cat, nil

	case "S3_TABLES":
		// S3 Tables via Iceberg REST protocol with SigV4 signing (service=s3tables)
		if opts.URI == "" {
			return nil, fmt.Errorf("URI is required for S3_TABLES source type")
		}
		region := opts.SigV4SigningRegion
		if region == "" {
			region = awsCfg.Region
		}
		signingName := opts.SigV4SigningName
		if signingName == "" {
			signingName = "s3tables"
		}

		var restOpts []rest.Option
		restOpts = append(restOpts,
			rest.WithAwsConfig(awsCfg),
			rest.WithSigV4RegionSvc(region, signingName),
		)
		if opts.Warehouse != "" {
			restOpts = append(restOpts, rest.WithWarehouseLocation(opts.Warehouse))
		}

		logging.Debugf("createCatalog: creating S3_TABLES catalog, uri=%s, region=%s, signingName=%s, warehouse=%s",
			opts.URI, region, signingName, opts.Warehouse)
		cat, err := rest.NewCatalog(ctx, "s3-tables", opts.URI, restOpts...)
		if err != nil {
			return nil, fmt.Errorf("failed to create S3_TABLES catalog: %w", err)
		}
		return cat, nil

	case "BIGLAKE_METASTORE":
		if opts.URI == "" {
			return nil, fmt.Errorf("URI is required for BIGLAKE_METASTORE source type")
		}
		cred := opts.CatalogCred
		if cred == nil || cred.Type != "gcp" || cred.GCP == nil {
			return nil, fmt.Errorf("GCP credential not found in collection")
		}
		// Google BigLake Metastore via Iceberg REST protocol.
		// Authenticates using a service-account JSON credential (from the catalog's
		// attached GCP credential).  A custom transport injects a fresh OAuth2 bearer
		// token and the X-Goog-User-Project quota header on every request.
		var ts oauth2.TokenSource
		scopes := []string{"https://www.googleapis.com/auth/cloud-platform"}

		if cred.GCP.AccessKeyID != "" && cred.GCP.SecretAccessKey != "" {
			ts = (&oauth2.Config{
				ClientID:     cred.GCP.AccessKeyID,
				ClientSecret: cred.GCP.SecretAccessKey,
				Endpoint:     google.Endpoint,
				Scopes:       scopes,
			}).TokenSource(go_context.Background(), &oauth2.Token{})
		} else {
			credential, err := google.CredentialsFromJSON(ctx, []byte(cred.GCP.JSONCredentials), scopes...)
			if err != nil {
				return nil, fmt.Errorf("failed to parse GCP service-account credential: %w", err)
			}

			// Eagerly obtain a token to catch configuration errors before the first
			// catalog request is made (e.g. wrong scopes, invalid JSON key).
			if _, err := credential.TokenSource.Token(); err != nil {
				return nil, fmt.Errorf("failed to obtain GCP access token for BIGLAKE_METASTORE: %w", err)
			}
			ts = credential.TokenSource
		}

		// Wrap the token source so successive calls reuse a cached token until
		// it is close to expiry.
		ts = oauth2.ReuseTokenSource(nil, ts)

		var restOpts []rest.Option
		// biglakeTransport overrides Authorization on each request with a fresh
		// bearer token; do NOT also pass rest.WithOAuthToken because
		// sessionTransport.RoundTrip adds its default headers *before* calling our
		// transport, which would result in two Authorization headers being sent.
		restOpts = append(restOpts, rest.WithCustomTransport(&biglakeTransport{
			base:        icebergTransport(),
			projectID:   opts.QuotaProjectID,
			tokenSource: ts,
		}))
		if opts.Warehouse != "" {
			restOpts = append(restOpts, rest.WithWarehouseLocation(opts.Warehouse))
		}

		logging.Debugf("createCatalog: creating BIGLAKE_METASTORE catalog, uri=%s, warehouse=%s, quotaProjectID=%s",
			opts.URI, opts.Warehouse, opts.QuotaProjectID)
		cat, err := rest.NewCatalog(ctx, "biglake-metastore", opts.URI, restOpts...)
		if err != nil {
			return nil, fmt.Errorf("failed to create BIGLAKE_METASTORE catalog: %w", err)
		}
		return cat, nil

	case "NESSIE":
		// Nessie catalog via Iceberg REST protocol
		if opts.URI == "" {
			return nil, fmt.Errorf("URI is required for NESSIE source type")
		}

		// Validate URI format
		if !strings.HasPrefix(opts.URI, "http://") && !strings.HasPrefix(opts.URI, "https://") {
			return nil, fmt.Errorf("NESSIE URI must start with http:// or https://, got: %s", opts.URI)
		}

		var restOpts []rest.Option
		authOpts, authErr := buildNessieHTTPAuthOpts(opts.CatalogCred)
		if authErr != nil {
			return nil, fmt.Errorf("NESSIE catalog credential error: %w", authErr)
		}
		if authOpts != nil {
			restOpts = append(restOpts, authOpts...)
		} else {
			logging.Debugf("createCatalog: NESSIE using no authentication (public server)")
		}

		if opts.Warehouse != "" {
			restOpts = append(restOpts, rest.WithWarehouseLocation(opts.Warehouse))
		}

		logging.Debugf("createCatalog: creating NESSIE catalog, uri=%s, warehouse=%s, hasCredential=%v", opts.URI, opts.Warehouse, opts.Credential != "")
		logging.Debugf("createCatalog: NESSIE restOpts count=%d", len(restOpts))
		for i, opt := range restOpts {
			logging.Debugf("createCatalog: NESSIE restOpt[%d]: %T", i, opt)
		}

		// Add panic recovery in case there's a panic in rest.NewCatalog
		var cat catalog.Catalog
		var err error
		func() {
			defer func() {
				if r := recover(); r != nil {
					logging.Errorf("createCatalog: NESSIE catalog creation panicked: %v", r)
					err = fmt.Errorf("catalog creation panicked: %v", r)
				}
			}()
			cat, err = rest.NewCatalog(ctx, "nessie", opts.URI, restOpts...)
		}()

		if err != nil {
			logging.Errorf("createCatalog: NESSIE catalog creation failed - error type: %T, error value: %+v, error string: '%s'", err, err, err.Error())

			// Try to extract more info from REST error
			if restErr, ok := err.(interface{ Error() string }); ok {
				logging.Errorf("createCatalog: NESSIE REST error details: %s", restErr.Error())
			}

			// Check if this might be a wrong endpoint error
			if err.Error() == ": " || err.Error() == "" {
				logging.Errorf("createCatalog: Empty error suggests wrong API endpoint. Nessie Iceberg REST API is usually at '/iceberg' not '/api/v2'")
				logging.Errorf("createCatalog: Try changing URI from '%s' to 'http://localhost:19120/iceberg'", opts.URI)
			}

			return nil, fmt.Errorf("failed to create NESSIE catalog: %w", err)
		}
		logging.Debugf("createCatalog: NESSIE catalog created successfully")
		return cat, nil

	case "REST", "NESSIE_REST", "UNITY_CATALOG":
		// Generic Iceberg REST catalog. REST is the canonical name; NESSIE_REST
		// and UNITY_CATALOG are accepted aliases that share this implementation.
		// Catalog auth supports http (bearer/basic/mTLS), Azure AD, Azure
		// Managed Identity, AWS SigV4, and GCP OAuth. Optional sigv4SigningName
		// and quotaProjectId let a generic REST source substitute for the more
		// specialized AWS_GLUE_REST / S3_TABLES / BIGLAKE_METASTORE presets.
		// Storage credentials (collection cred) are forwarded as iceberg-go
		// FileIO props (adls.* / s3.* / gcs.*) as a fallback for catalogs that
		// don't vend credentials in LoadTableResponse.
		if opts.URI == "" {
			return nil, fmt.Errorf("URI is required for %s source type", opts.SourceType)
		}

		var restOpts []rest.Option
		authOpts, err := buildRESTCatalogAuthOpts(ctx, opts.CatalogCred, opts.URI, restAuthExtras{
			SigV4SigningRegion: opts.SigV4SigningRegion,
			SigV4SigningName:   opts.SigV4SigningName,
			QuotaProjectID:     opts.QuotaProjectID,
		})
		if err != nil {
			return nil, fmt.Errorf("%s catalog credential error: %w", opts.SourceType, err)
		}
		if authOpts != nil {
			restOpts = append(restOpts, authOpts...)
		}

		if opts.CollectionCred != nil {
			props, err := buildRESTStorageProps(opts.CollectionCred)
			if err != nil {
				return nil, fmt.Errorf("%s storage credential error: %w", opts.SourceType, err)
			}
			if len(props) > 0 {
				restOpts = append(restOpts, rest.WithAdditionalProps(props))
			}
		}

		if opts.Warehouse != "" {
			restOpts = append(restOpts, rest.WithWarehouseLocation(opts.Warehouse))
		}

		// The catalog name is just a label iceberg-go uses for logging; preserve
		// the user's chosen alias so logs reflect intent.
		catName := "rest"
		switch sourceType {
		case "NESSIE_REST":
			catName = "nessie-rest"
		case "UNITY_CATALOG":
			catName = "unity-catalog"
		}
		logging.Debugf("createCatalog: creating %s catalog, uri=%s", opts.SourceType, opts.URI)
		cat, err := rest.NewCatalog(ctx, catName, opts.URI, restOpts...)
		if err != nil {
			return nil, fmt.Errorf("failed to create %s catalog: %w", opts.SourceType, err)
		}
		return cat, nil

	case "SQL":
		if opts.URI == "" {
			return nil, fmt.Errorf("URI (DSN connection string) is required for SQL source type")
		}
		dialect, driver, err := sqlDialectFromURI(opts.URI, opts.SQLDialect)
		if err != nil {
			return nil, err
		}
		dsn, err := injectSQLCredentials(opts.URI, opts.CatalogCred)
		if err != nil {
			return nil, fmt.Errorf("SQL catalog credential error: %w", err)
		}
		logging.Debugf("createCatalog: creating SQL catalog, dialect=%s", dialect)
		db, err := sql.Open(driver, dsn)
		if err != nil {
			return nil, fmt.Errorf("failed to open SQL connection for SQL catalog: %w", err)
		}
		cat, err := icesql.NewCatalog("sql", db, dialect, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create SQL catalog: %w", err)
		}
		return cat, nil

	default:
		return nil, fmt.Errorf("unsupported source type: '%s'", opts.SourceType)
	}
}

func (s *Scanner) LoadTable(ctx go_context.Context) error {
	if s.catalog == nil {
		return fmt.Errorf("catalog not initialized")
	}

	// Use identifier slice as expected by iceberg-go
	tbl, err := s.catalog.LoadTable(ctx, s.tableIdent)
	if err != nil {
		return fmt.Errorf("failed to load table %s.%s: %w", s.databaseName, s.tableName, err)
	}

	s.table = tbl

	// Log table info for debugging
	if s.table != nil {
		snapshot := s.table.CurrentSnapshot()
		if snapshot != nil {
			logging.Debugf("Iceberg table %s.%s loaded - current snapshot: %d, location: %s",
				s.databaseName, s.tableName, snapshot.SnapshotID, snapshot.ManifestList)
		} else {
			logging.Warnf("Iceberg table %s.%s loaded but has no current snapshot",
				s.databaseName, s.tableName)
		}
	}

	return nil
}

func (s *Scanner) Scan(ctx go_context.Context) (*table.Scan, error) {
	if s.table == nil {
		return nil, fmt.Errorf("table not loaded, call LoadTable first")
	}

	// Log table metadata to understand the structure
	metadata := s.table.Metadata()
	logging.Debugf("Iceberg Table Metadata: format_version=%d, table_uuid=%s",
		metadata.Version(), metadata.TableUUID())

	if snapshot := metadata.CurrentSnapshot(); snapshot != nil {
		logging.Debugf("Iceberg Current Snapshot: id=%d, manifest_list=%s",
			snapshot.SnapshotID, snapshot.ManifestList)

		if summary := snapshot.Summary; summary != nil {
			logging.Debugf("Iceberg Snapshot Summary: operation=%s, properties=%v",
				summary.Operation, summary.Properties)
		}
	} else {
		logging.Debugf("Iceberg Table: no current snapshot")
	}

	// Create scan object from table with appropriate options
	var opts []table.ScanOption

	opts = append(opts, table.WitMaxConcurrency(s.parallelScans))

	logging.Debugf("Iceberg scan options - Database: %s, Table: %s", s.databaseName, s.tableName)

	if s.filterPushdown != nil {
		// Get the combined filter expression for pushdown
		filterExpr := s.filterPushdown.GetExpression()
		if filterExpr != nil {
			logging.Debugf("Iceberg scan: applying row filter pushdown: %T", filterExpr)
			opts = append(opts, table.WithRowFilter(filterExpr))
		} else {
			logging.Warnf("Iceberg scan: filterPushdown defined but no filter expression found")
		}
	}

	if s.snapshotID != nil {
		logging.Debugf("Iceberg scan with snapshot ID: %d", *s.snapshotID)
		opts = append(opts, table.WithSnapshotID(*s.snapshotID))
	}

	if s.snapshotAsOf != nil {
		// Convert timestamp to readable format
		timestamp := *s.snapshotAsOf
		logging.Debugf("Iceberg scan with snapshot as of timestamp: %d", timestamp)
		opts = append(opts, table.WithSnapshotAsOf(timestamp))
	}

	if s.limit > 0 {
		logging.Debugf("Iceberg scan with limit: %d", s.limit)
		opts = append(opts, table.WithLimit(s.limit))
	}

	scanObj := s.table.Scan(opts...)
	logging.Debugf("Iceberg Scan object created successfully for table %s.%s", s.databaseName, s.tableName)

	return scanObj, nil
}

// extractTopLevelParents extracts the top-level parent field names from projection paths.
// For example: ["x.y.z", "y1.z", "a.b", "c", "`address`.`city`"] -> {"x", "y1", "a", "c", "address"}
// This allows Arrow to filter at the parent column level while ObjectPopulate handles
// fine-grained nested field filtering.
//
// Note: This function is defined in the iceberg package, but it uses parseDottedPath
// from the value package. To avoid circular dependencies, we duplicate the parsing logic here.
func extractTopLevelParents(fields []string) []string {
	parents := make(map[string]bool)

	for _, field := range fields {
		if field == "" {
			continue
		}

		// Parse the dotted path to extract the first segment (top-level parent)
		// We need to handle backtick-quoted identifiers like `address`.`city`
		segments := parseDottedPathForExtract(field)
		if len(segments) == 0 {
			continue
		}

		// The first segment is the top-level parent column
		parents[segments[0]] = true
	}

	result := make([]string, 0, len(parents))
	for parent := range parents {
		result = append(result, parent)
	}

	return result
}

// parseDottedPathForExtract parses a dotted notation string into field segments,
// properly handling backticks. This is a simplified version of parseDottedPath
// to avoid circular dependency with the value package.
func parseDottedPathForExtract(path string) []string {
	if path == "" {
		return []string{}
	}

	var segments []string
	var currentSegment strings.Builder
	inBacktick := false
	escapeNext := false

	for i := 0; i < len(path); i++ {
		c := path[i]

		if escapeNext {
			currentSegment.WriteByte(c)
			escapeNext = false
			continue
		}

		if c == '\\' && inBacktick {
			escapeNext = true
			continue
		}

		if c == '`' {
			if inBacktick {
				// Closing backtick
				if currentSegment.Len() > 0 || (i > 0 && path[i-1] == '`') {
					segments = append(segments, currentSegment.String())
					currentSegment.Reset()
				}
				inBacktick = false
			} else {
				// Opening backtick
				if currentSegment.Len() > 0 {
					segments = append(segments, currentSegment.String())
					currentSegment.Reset()
				}
				inBacktick = true
			}
		} else if c == '.' && !inBacktick {
			// Dot outside of backticks is a separator
			if currentSegment.Len() > 0 {
				segments = append(segments, currentSegment.String())
				currentSegment.Reset()
			}
		} else {
			currentSegment.WriteByte(c)
		}
	}

	// Handle any remaining content
	if inBacktick {
		segments = append(segments, currentSegment.String())
	} else if currentSegment.Len() > 0 {
		segments = append(segments, currentSegment.String())
	}

	return segments
}

// CreateReader creates a reader for iterating over scan results
func (s *Scanner) CreateReader(ctx go_context.Context) (*Reader, error) {
	if s.table == nil {
		return nil, fmt.Errorf("table not loaded, call LoadTable first")
	}

	scan, err := s.Scan(ctx)
	if err != nil {
		return nil, err
	}

	reader, err := NewReader(ctx, scan)
	if err != nil {
		return nil, err
	}

	if s.decimalToDouble {
		reader.SetDecimalToDouble(true)
	}

	// Set column filter if selectedFields is specified
	if len(s.selectedFields) > 0 {
		// Extract top-level parent columns from projection paths
		parents := extractTopLevelParents(s.selectedFields)

		if len(parents) > 0 {
			// Build a field set for fast lookup
			fieldSet := make(map[string]bool, len(parents))
			for _, parent := range parents {
				fieldSet[parent] = true
			}

			reader.SetColumnFilter(func(fieldName string) bool {
				return fieldSet[fieldName]
			})

			logging.Debugf("Iceberg CreateReader: set column filter for %d parent columns from %d projection fields: parents=%v, projections=%v",
				len(parents), len(s.selectedFields), parents, s.selectedFields)
		} else {
			logging.Warnf("Iceberg CreateReader: selectedFields specified but no parent columns extracted, reading all columns")
		}
	}

	return reader, nil
}

// ScanAndConvertWithPlanFiles reads data using PlanFiles approach to avoid NESSIE context cancellation
func (s *Scanner) ScanAndConvertWithPlanFiles(ctx go_context.Context) (<-chan map[string]interface{}, <-chan error) {
	logging.Debugf("Iceberg ScanAndConvertWithPlanFiles: starting NESSIE multi-file scan for %s.%s", s.databaseName, s.tableName)

	scan, err := s.Scan(ctx)
	if err != nil {
		logging.Errorf("Iceberg ScanAndConvertWithPlanFiles: failed to create scan: %v", err)
		resultChan := make(chan map[string]interface{})
		errorChan := make(chan error, 1)
		errorChan <- err
		close(resultChan)
		close(errorChan)
		return resultChan, errorChan
	}

	// Get all data files for this table
	fileTasks, err := scan.PlanFiles(ctx)
	if err != nil {
		logging.Errorf("Iceberg ScanAndConvertWithPlanFiles: failed to plan files: %v", err)
		resultChan := make(chan map[string]interface{})
		errorChan := make(chan error, 1)
		errorChan <- err
		close(resultChan)
		close(errorChan)
		return resultChan, errorChan
	}

	logging.Debugf("Iceberg ScanAndConvertWithPlanFiles: NESSIE catalog planned %d data files", len(fileTasks))

	resultChan := make(chan map[string]interface{}, 100)
	errorChan := make(chan error, 1)

	go func() {
		defer close(errorChan)
		defer close(resultChan)

		totalRowsSent := 0
		totalExpectedRows := int64(0)
		for _, task := range fileTasks {
			totalExpectedRows += task.File.Count()
		}

		logging.Debugf("Iceberg ScanAndConvertWithPlanFiles: NESSIE catalog expecting %d total rows from %d files",
			totalExpectedRows, len(fileTasks))

		// Use the original ToArrowRecords but with our own file processing
		schema, recordIter, err := scan.ToArrowRecords(ctx)
		if err != nil {
			logging.Errorf("Iceberg ScanAndConvertWithPlanFiles: failed to create arrow records iterator: %v", err)
			errorChan <- err
			return
		}

		logging.Debugf("Iceberg ScanAndConvertWithPlanFiles: created arrow records iterator, schema has %d fields", len(schema.Fields()))

		recordBatchCount := 0
		for recordBatch, err := range recordIter {
			if err != nil {
				// Check if this is the context cancellation bug
				if err.Error() == "context canceled" {
					logging.Warnf("Iceberg ScanAndConvertWithPlanFiles: NESSIE context canceled (iceberg-go v0.4.0 limitation), sent %d/%d rows",
						totalRowsSent, totalExpectedRows)
					// Don't treat this as an error since we got partial data
					break
				} else {
					logging.Errorf("Iceberg ScanAndConvertWithPlanFiles: error reading record batch: %v", err)
					errorChan <- err
					return
				}
			}

			recordBatchCount++

			// Convert each row in this batch
			for rowIdx := 0; rowIdx < int(recordBatch.NumRows()); rowIdx++ {
				// Extract row data
				rowData := make(map[string]interface{})
				for colIdx, field := range schema.Fields() {
					column := recordBatch.Column(colIdx)
					if rowIdx < int(column.Len()) {
						if !column.IsNull(rowIdx) {
							// Convert Arrow value to Go interface{}
							val, convErr := s.convertArrowValue(column, rowIdx, field.Type)
							if convErr != nil {
								logging.Warnf("Iceberg ScanAndConvertWithPlanFiles: failed to convert field %s: %v",
									field.Name, convErr)
								continue
							}
							rowData[field.Name] = val
						}
					}
				}

				totalRowsSent++
				select {
				case resultChan <- rowData:
					// Row sent successfully
				case <-ctx.Done():
					logging.Debugf("Iceberg ScanAndConvertWithPlanFiles: context canceled by caller after %d rows", totalRowsSent)
					return
				}
			}

			recordBatch.Release()
		}

		logging.Debugf("Iceberg ScanAndConvertWithPlanFiles: NESSIE completed, sent %d rows from %d record batches", totalRowsSent, recordBatchCount)
	}()

	return resultChan, errorChan
}

// convertArrowValue converts an Arrow array value at index to Go interface{} (used by NESSIE multi-file workaround)
func (s *Scanner) convertArrowValue(column arrow.Array, rowIdx int, fieldType arrow.DataType) (interface{}, error) {
	switch arr := column.(type) {
	case *array.String:
		return arr.Value(rowIdx), nil
	case *array.Int64:
		return arr.Value(rowIdx), nil
	case *array.Int32:
		return arr.Value(rowIdx), nil
	case *array.Float64:
		return arr.Value(rowIdx), nil
	case *array.Float32:
		return arr.Value(rowIdx), nil
	case *array.Boolean:
		return arr.Value(rowIdx), nil
	case *array.Timestamp:
		return arr.Value(rowIdx), nil
	default:
		// For complex types, try to get string representation
		return fmt.Sprintf("%v", column.GetOneForMarshal(rowIdx)), nil
	}
}

// ScanAndConvertStream reads data and converts to JSON format with streaming
func (s *Scanner) ScanAndConvertStream(ctx go_context.Context) (<-chan map[string]interface{}, <-chan error) {
	logging.Debugf("Iceberg ScanAndConvertStream: starting scan for %s.%s", s.databaseName, s.tableName)

	// If collection credential is set (AWS or GCP), use direct parallel file reading path.
	// Catalog credential is used for PlanFiles; collection credential reads the actual data files.
	if s.collectionCred != nil && isStorageCredential(s.collectionCred) {
		logging.Debugf("Iceberg ScanAndConvertStream: using parallel file read path (type=%s, parallelScans=%d)",
			s.collectionCred.Type, s.parallelScans)
		return s.ScanAndConvertParallelFiles(ctx)
	}

	// Use individual file scan approach for REST-protocol catalogs (which have multi-file context cancellation issues)
	if s.sourceType == "NESSIE" || s.sourceType == "REST" || s.sourceType == "NESSIE_REST" || s.sourceType == "UNITY_CATALOG" {
		logging.Debugf("Iceberg ScanAndConvertStream: using individual file scan approach for %s catalog", s.sourceType)

		// Check if this is a multi-file table
		scan, err := s.Scan(ctx)
		if err != nil {
			logging.Errorf("Iceberg ScanAndConvertStream: failed to create scan: %v", err)
			resultChan := make(chan map[string]interface{})
			errorChan := make(chan error, 1)
			errorChan <- err
			close(resultChan)
			close(errorChan)
			return resultChan, errorChan
		}

		fileTasks, err := scan.PlanFiles(ctx)
		if err != nil {
			logging.Warnf("Iceberg ScanAndConvertStream: failed to plan files for %s, falling back to standard approach: %v",
				s.sourceType, err)
			return s.scanAndConvertStreamFallback(ctx)
		}

		if len(fileTasks) > 1 {
			logging.Debugf("Iceberg ScanAndConvertStream: detected multi-file table (%d files) for %s, using PlanFiles approach",
				len(fileTasks), s.sourceType)
			return s.ScanAndConvertWithPlanFiles(ctx)
		} else {
			logging.Debugf("Iceberg ScanAndConvertStream: single file table for %s, using standard approach", s.sourceType)
		}
	}

	// Use standard approach for all other catalog types and single-file NESSIE tables
	logging.Debugf("Iceberg ScanAndConvertStream: using standard streaming approach for %s catalog", s.sourceType)
	return s.scanAndConvertStreamFallback(ctx)
}

// scanAndConvertStreamFallback is the original streaming approach
func (s *Scanner) scanAndConvertStreamFallback(ctx go_context.Context) (<-chan map[string]interface{}, <-chan error) {

	reader, err := s.CreateReader(ctx)
	if err != nil {
		logging.Errorf("Iceberg ScanAndConvertStream: failed to create reader: %v", err)
		resultChan := make(chan map[string]interface{})
		errorChan := make(chan error, 1)
		errorChan <- err
		close(resultChan)
		close(errorChan)
		return resultChan, errorChan
	}

	logging.Debugf("Iceberg ScanAndConvertStream: reader created successfully, starting iteration")
	iterator := NewIterator(reader)

	resultChan := make(chan map[string]interface{}, s.parallelScans*_ICEBERG_RESULT_CHAN_PER_WORKER)
	errorChan := make(chan error, 1)

	go func() {
		defer close(errorChan)
		defer close(resultChan)
		defer iterator.Close()

		rowsSent := 0

		logging.Debugf("Iceberg ScanAndConvertStream: beginning row iteration")
		for iterator.Next() {
			if ctx.Err() != nil {
				return
			}

			row, err := iterator.Row()
			if err != nil {
				logging.Errorf("Iceberg ScanAndConvertStream: error getting row %d: %v", rowsSent+1, err)
				errorChan <- err
				return
			}

			select {
			case resultChan <- row:
			case <-ctx.Done():
				return
			}
			rowsSent++

			if rowsSent%100 == 0 {
				logging.Debugf("Iceberg scan progress: %d rows sent", rowsSent)
			}
		}

		logging.Debugf("Iceberg scan completed: %d total rows sent", rowsSent)

		if err := iterator.Err(); err != nil {
			logging.Errorf("Iceberg ScanAndConvertStream: iterator error after completion: %v", err)
			errorChan <- err
		}
	}()

	return resultChan, errorChan
}

// isStorageCredential returns true for credential types that can access cloud object storage.
func isStorageCredential(cred *cbauth.Credential) bool {
	if cred == nil {
		return false
	}
	switch cred.Type {
	case "aws":
		return cred.AWS != nil
	case "gcp":
		return cred.GCP != nil
	case "azureShared":
		return cred.AzureShared != nil
	case "azureSas":
		return cred.AzureSAS != nil
	case "azureAd":
		return cred.AzureAD != nil
	case "azureManaged":
		return cred.AzureManaged != nil
	}
	return false
}

// fileDownloader abstracts cloud object storage file access across providers.
type fileDownloader interface {
	download(ctx go_context.Context, uri string) ([]byte, error)
}

// s3Downloader downloads files from AWS S3 (or S3-compatible, e.g. s3a://).
type s3Downloader struct{ client *s3.Client }

func (d *s3Downloader) download(ctx go_context.Context, uri string) ([]byte, error) {
	// Normalise s3a:// → s3://
	uri = strings.Replace(uri, "s3a://", "s3://", 1)
	bucket, key, err := ParseS3URI(uri)
	if err != nil {
		return nil, fmt.Errorf("invalid S3 URI %s: %w", uri, err)
	}
	result, err := d.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("S3 GetObject s3://%s/%s: %w", bucket, key, err)
	}
	defer result.Body.Close()
	return io.ReadAll(result.Body)
}

// gcsDownloader downloads files from Google Cloud Storage (gs://).
type gcsDownloader struct{ client *storage.Client }

func (d *gcsDownloader) download(ctx go_context.Context, uri string) ([]byte, error) {
	bucket, object, err := ParseGCSURI(uri)
	if err != nil {
		return nil, fmt.Errorf("invalid GCS URI %s: %w", uri, err)
	}
	rc, err := d.client.Bucket(bucket).Object(object).NewReader(ctx)
	if err != nil {
		return nil, fmt.Errorf("GCS open gs://%s/%s: %w", bucket, object, err)
	}
	defer rc.Close()
	return io.ReadAll(rc)
}

// adlsDownloader downloads files from Azure Data Lake Storage / Blob Storage
// (abfs://, abfss://, wasb://, wasbs://) using iceberg-go's FileIO. The props
// map carries the same ADLS auth keys that the REST catalog passes to LoadFSFunc,
// so the same credential plumbing works for both metadata and data file access.
type adlsDownloader struct {
	props iceberg.Properties
}

func (d *adlsDownloader) download(ctx go_context.Context, uri string) ([]byte, error) {
	fs, err := iceio.LoadFS(ctx, d.props, uri)
	if err != nil {
		return nil, fmt.Errorf("ADLS LoadFS for %s: %w", uri, err)
	}
	if rf, ok := fs.(iceio.ReadFileIO); ok {
		return rf.ReadFile(uri)
	}
	f, err := fs.Open(uri)
	if err != nil {
		return nil, fmt.Errorf("ADLS open %s: %w", uri, err)
	}
	defer f.Close()
	return io.ReadAll(f)
}

// ParseGCSURI splits a gs://bucket/object URI into its parts.
func ParseGCSURI(uri string) (bucket, object string, err error) {
	const prefix = "gs://"
	if !strings.HasPrefix(uri, prefix) {
		return "", "", fmt.Errorf("not a GCS URI: %s", uri)
	}
	trimmed := strings.TrimPrefix(uri, prefix)
	parts := strings.SplitN(trimmed, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid GCS URI (bucket or object empty): %s", uri)
	}
	return parts[0], parts[1], nil
}

// buildFileDownloader constructs the appropriate downloader for the given collection/catalog credential.
// It also takes a sample file URI so it can do URI-scheme detection when the credential type
// does not unambiguously determine the provider (e.g. Nessie with no collection cred).
func (s *Scanner) buildFileDownloader(ctx go_context.Context, sampleURI string) (fileDownloader, error) {
	// Prefer collection credential when available.
	if s.collectionCred != nil {
		switch s.collectionCred.Type {
		case "aws":
			if s.collectionCred.AWS == nil {
				return nil, fmt.Errorf("aws collection credential has no AWS payload")
			}
			c := s.collectionCred.AWS
			region := c.Region
			if region == "" && s.awsConfig != nil {
				region = s.awsConfig.Region
			}
			cfg, err := NewAWSConfig(c.AccessKeyID, c.SecretAccessKey, c.SessionToken, region)
			if err != nil {
				return nil, fmt.Errorf("failed to build AWS config from collection credential: %w", err)
			}
			return &s3Downloader{client: s3.NewFromConfig(*cfg)}, nil

		case "gcp":
			if s.collectionCred.GCP == nil {
				return nil, fmt.Errorf("gcp collection credential has no GCP payload")
			}
			gcsClient, err := buildGCSClient(ctx, s.collectionCred)
			if err != nil {
				return nil, fmt.Errorf("failed to build GCS client from collection credential: %w", err)
			}
			return &gcsDownloader{client: gcsClient}, nil

		case "azureShared", "azureSas", "azureAd", "azureManaged":
			props, err := buildRESTStorageProps(s.collectionCred)
			if err != nil {
				return nil, fmt.Errorf("failed to build ADLS props from collection credential: %w", err)
			}
			return &adlsDownloader{props: props}, nil
		}
	}

	// No collection cred (e.g. Nessie): detect from the file URI scheme.
	lower := strings.ToLower(sampleURI)
	switch {
	case strings.HasPrefix(lower, "s3://") || strings.HasPrefix(lower, "s3a://"):
		if s.awsConfig == nil {
			return nil, fmt.Errorf("S3 URI detected but no AWS credential available")
		}
		return &s3Downloader{client: s3.NewFromConfig(*s.awsConfig)}, nil

	case strings.HasPrefix(lower, "gs://"):
		return nil, fmt.Errorf("GCS URI detected but no GCP collection credential provided")

	case strings.HasPrefix(lower, "abfs://"),
		strings.HasPrefix(lower, "abfss://"),
		strings.HasPrefix(lower, "wasb://"),
		strings.HasPrefix(lower, "wasbs://"):
		// No explicit Azure credential — let iceberg-go fall back to azidentity.NewDefaultAzureCredential.
		return &adlsDownloader{props: nil}, nil
	}

	return nil, fmt.Errorf("cannot determine storage provider for URI: %s", sampleURI)
}

// buildGCSClient creates a GCS storage.Client from a GCP credential.
// Supports service-account JSON (JSONCredentials) and HMAC (AccessKeyID/SecretAccessKey).
func buildGCSClient(ctx go_context.Context, cred *cbauth.Credential) (*storage.Client, error) {
	gcp := cred.GCP
	if gcp.JSONCredentials != "" {
		creds, err := google.CredentialsFromJSON(ctx, []byte(gcp.JSONCredentials),
			"https://www.googleapis.com/auth/cloud-platform")
		if err != nil {
			return nil, fmt.Errorf("invalid GCP service-account JSON: %w", err)
		}
		// Explicitly chain oauth2.Transport → shared transport so the service-account
		// bearer token is injected while connections are reused via icebergTransport.
		return storage.NewClient(ctx, gOption.WithHTTPClient(&http.Client{
			Transport: &oauth2.Transport{
				Source: creds.TokenSource,
				Base:   icebergTransport(),
			},
		}))
	}
	if gcp.AccessKeyID != "" && gcp.SecretAccessKey != "" {
		// HMAC key – use OAuth2 HMAC signing (GCS S3-compatible is separate;
		// for the native JSON API we need a token source that signs requests).
		// With only HMAC keys the native GCS client cannot be used directly.
		// Fall back to Application Default Credentials and warn.
		logging.Warnf("buildGCSClient: HMAC-mode GCP credential provided; " +
			"GCS native API requires service-account JSON. Attempting Application Default Credentials.")
	}
	// Fallback: Application Default Credentials manage their own transport chain.
	return storage.NewClient(ctx)
}

// ScanAndConvertParallelFiles uses PlanFiles to obtain the data file list from the catalog,
// then reads each file independently and in parallel using the collection credential.
// Supported: parquet, arrow IPC, avro (OCF). ORC is skipped with a warning.
func (s *Scanner) ScanAndConvertParallelFiles(ctx go_context.Context) (<-chan map[string]interface{}, <-chan error) {
	resultChan := make(chan map[string]interface{}, s.parallelScans*_ICEBERG_RESULT_CHAN_PER_WORKER)
	errorChan := make(chan error, 1)

	go func() {
		defer close(errorChan)
		defer close(resultChan)

		scan, err := s.Scan(ctx)
		if err != nil {
			errorChan <- fmt.Errorf("failed to create scan: %w", err)
			return
		}

		fileTasks, err := scan.PlanFiles(ctx)
		if err != nil {
			errorChan <- fmt.Errorf("failed to plan files: %w", err)
			return
		}

		logging.Debugf("ScanAndConvertParallelFiles: planned %d data files, parallelism=%d",
			len(fileTasks), s.parallelScans)

		if len(fileTasks) == 0 {
			return
		}

		// Build the storage downloader using the first file's URI for scheme detection.
		downloader, err := s.buildFileDownloader(ctx, fileTasks[0].File.FilePath())
		if err != nil {
			errorChan <- fmt.Errorf("failed to build file downloader: %w", err)
			return
		}

		parallelism := s.parallelScans
		if parallelism <= 0 {
			parallelism = 1
		}

		taskChan := make(chan table.FileScanTask, len(fileTasks))
		for _, task := range fileTasks {
			taskChan <- task
		}
		close(taskChan)

		var wg sync.WaitGroup
		for i := 0; i < parallelism; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for task := range taskChan {
					if ctx.Err() != nil {
						return
					}
					if readErr := s.streamFileTask(ctx, downloader, task, resultChan); readErr != nil {
						logging.Errorf("ScanAndConvertParallelFiles: error reading %s: %v",
							task.File.FilePath(), readErr)
					}
				}
			}()
		}

		wg.Wait()
	}()

	return resultChan, errorChan
}

// streamFileTask downloads a single data file and streams its rows to resultChan.
func (s *Scanner) streamFileTask(ctx go_context.Context, dl fileDownloader, task table.FileScanTask,
	resultChan chan<- map[string]interface{}) error {

	filePath := task.File.FilePath()
	format := task.File.FileFormat()

	logging.Debugf("ScanAndConvertParallelFiles: reading file %s (format=%s, records=%d)",
		filePath, format, task.File.Count())

	fileData, err := dl.download(ctx, filePath)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}

	switch format {
	case iceberg.ParquetFile:
		return s.streamParquetFile(ctx, fileData, resultChan)
	case iceberg.AvroFile:
		return s.streamAvroFile(ctx, fileData, resultChan)
	case iceberg.OrcFile:
		return s.streamOrcFile(ctx, fileData, resultChan)
	default:
		// Format unknown from Iceberg metadata; detect from extension.
		return s.streamFileByExtension(ctx, fileData, filePath, resultChan)
	}
}

// streamParquetFile reads a parquet file and sends rows to resultChan.
func (s *Scanner) streamParquetFile(ctx go_context.Context, data []byte, resultChan chan<- map[string]interface{}) error {
	r := bytes.NewReader(data)

	pqReader, err := pqfile.NewParquetReader(r)
	if err != nil {
		return fmt.Errorf("failed to open parquet reader: %w", err)
	}
	defer pqReader.Close()

	arrowReader, err := pqarrow.NewFileReader(pqReader, pqarrow.ArrowReadProperties{BatchSize: 4096}, memory.DefaultAllocator)
	if err != nil {
		return fmt.Errorf("failed to create arrow/parquet reader: %w", err)
	}

	rr, err := arrowReader.GetRecordReader(ctx, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to get parquet record reader: %w", err)
	}
	defer rr.Release()

	helper := &Reader{decimalToDouble: s.decimalToDouble}
	fieldSet := s.projectionFieldSet()

	for rr.Next() {
		rec := rr.Record()
		schema := rec.Schema()
		for rowIdx := 0; rowIdx < int(rec.NumRows()); rowIdx++ {
			row := make(map[string]interface{}, schema.NumFields())
			for colIdx, field := range schema.Fields() {
				if fieldSet != nil && !fieldSet[field.Name] {
					continue
				}
				col := rec.Column(colIdx)
				if col.IsNull(rowIdx) {
					row[field.Name] = nil
				} else {
					row[field.Name] = helper.getColumnValue(col, rowIdx)
				}
			}
			select {
			case resultChan <- row:
			case <-ctx.Done():
				rec.Release()
				return ctx.Err()
			}
		}
		rec.Release()
	}
	return rr.Err()
}

// streamArrowIPCFile reads an Arrow IPC file and sends rows to resultChan.
func (s *Scanner) streamArrowIPCFile(ctx go_context.Context, data []byte, resultChan chan<- map[string]interface{}) error {
	ipcReader, err := ipc.NewReader(bytes.NewReader(data), ipc.WithAllocator(memory.DefaultAllocator))
	if err != nil {
		return fmt.Errorf("failed to create Arrow IPC reader: %w", err)
	}
	defer ipcReader.Release()

	helper := &Reader{decimalToDouble: s.decimalToDouble}
	fieldSet := s.projectionFieldSet()

	for ipcReader.Next() {
		rec := ipcReader.Record()
		schema := rec.Schema()
		for rowIdx := 0; rowIdx < int(rec.NumRows()); rowIdx++ {
			row := make(map[string]interface{}, schema.NumFields())
			for colIdx, field := range schema.Fields() {
				if fieldSet != nil && !fieldSet[field.Name] {
					continue
				}
				col := rec.Column(colIdx)
				if col.IsNull(rowIdx) {
					row[field.Name] = nil
				} else {
					row[field.Name] = helper.getColumnValue(col, rowIdx)
				}
			}
			select {
			case resultChan <- row:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
	return ipcReader.Err()
}

// streamAvroFile reads an Avro OCF file and sends rows to resultChan.
// Row values are decoded into map[string]interface{} via hamba/avro.
func (s *Scanner) streamAvroFile(ctx go_context.Context, data []byte, resultChan chan<- map[string]interface{}) error {
	dec, err := ocf.NewDecoder(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create Avro OCF decoder: %w", err)
	}
	defer dec.Close()

	fieldSet := s.projectionFieldSet()

	for dec.HasNext() {
		var raw any
		if err := dec.Decode(&raw); err != nil {
			return fmt.Errorf("avro decode error: %w", err)
		}

		var row map[string]interface{}
		switch v := raw.(type) {
		case map[string]interface{}:
			row = v
		default:
			// Wrap non-record types under a synthetic key.
			row = map[string]interface{}{"value": raw}
		}

		if fieldSet != nil {
			filtered := make(map[string]interface{}, len(fieldSet))
			for k, v := range row {
				if fieldSet[k] {
					filtered[k] = v
				}
			}
			row = filtered
		}

		select {
		case resultChan <- row:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return dec.Error()
}

// streamFileByExtension detects the file format from the extension and streams rows.
func (s *Scanner) streamFileByExtension(ctx go_context.Context, data []byte, filePath string,
	resultChan chan<- map[string]interface{}) error {

	lower := strings.ToLower(filePath)
	switch {
	case strings.HasSuffix(lower, ".parquet"):
		return s.streamParquetFile(ctx, data, resultChan)
	case strings.HasSuffix(lower, ".arrow") || strings.HasSuffix(lower, ".arrows"):
		return s.streamArrowIPCFile(ctx, data, resultChan)
	case strings.HasSuffix(lower, ".avro"):
		return s.streamAvroFile(ctx, data, resultChan)
	case strings.HasSuffix(lower, ".orc"):
		return s.streamOrcFile(ctx, data, resultChan)
	default:
		// Default assumption for Iceberg is parquet.
		logging.Warnf("ScanAndConvertParallelFiles: unknown extension in %s, attempting parquet read", filePath)
		return s.streamParquetFile(ctx, data, resultChan)
	}
}

// sizedBytesReader wraps bytes.Reader to satisfy orc.SizedReaderAt (ReadAt + Size).
// bytes.Reader.Len() returns remaining bytes, not total; we store the total separately.
type sizedBytesReader struct {
	r    *bytes.Reader
	size int64
}

func (s *sizedBytesReader) ReadAt(p []byte, off int64) (int, error) { return s.r.ReadAt(p, off) }
func (s *sizedBytesReader) Size() int64                             { return s.size }

// streamOrcFile reads an ORC file using scritchley/orc and sends rows to resultChan.
func (s *Scanner) streamOrcFile(ctx go_context.Context, data []byte, resultChan chan<- map[string]interface{}) error {
	// bytes.Reader satisfies io.ReaderAt; wrap it to also expose Size().
	sr := &sizedBytesReader{r: bytes.NewReader(data), size: int64(len(data))}

	r, err := orc.NewReader(sr)
	if err != nil {
		return fmt.Errorf("failed to open ORC reader: %w", err)
	}
	defer r.Close()

	schema := r.Schema()
	fields := schema.Columns()
	fieldSet := s.projectionFieldSet()

	// Filter to requested columns; orc.Select streams only those stripes.
	selected := fields
	if fieldSet != nil {
		selected = selected[:0]
		for _, f := range fields {
			if fieldSet[f] {
				selected = append(selected, f)
			}
		}
	}

	cursor := r.Select(selected...)
	for cursor.Next() {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		vals := cursor.Row()
		row := make(map[string]interface{}, len(selected))
		for i, f := range selected {
			if i < len(vals) {
				row[f] = vals[i]
			}
		}
		select {
		case resultChan <- row:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return cursor.Err()
}

// projectionFieldSet returns a set of top-level parent column names from selectedFields.
// Returns nil when no projection is specified (include all columns).
func (s *Scanner) projectionFieldSet() map[string]bool {
	if len(s.selectedFields) == 0 {
		return nil
	}
	parents := extractTopLevelParents(s.selectedFields)
	if len(parents) == 0 {
		return nil
	}
	fieldSet := make(map[string]bool, len(parents))
	for _, p := range parents {
		fieldSet[p] = true
	}
	return fieldSet
}

// GetSchema returns the table schema
func (s *Scanner) GetSchema() interface{} {
	if s.table == nil {
		return nil
	}

	return s.table.Schema()
}

// GetTableLocation returns the table location
func (s *Scanner) GetTableLocation() string {
	if s.table == nil {
		return ""
	}

	return s.table.Location()
}

// GetTable returns the table object
func (s *Scanner) GetTable() *table.Table {
	return s.table
}

func (s *Scanner) Close() error {
	s.table = nil
	return nil
}

// Catalog returns the underlying catalog client so callers can cache and reuse it.
func (s *Scanner) Catalog() catalog.Catalog {
	return s.catalog
}

func ParseS3URI(uri string) (bucket, key string, err error) {
	if !strings.HasPrefix(uri, "s3://") {
		return "", "", fmt.Errorf("invalid S3 URI: %s", uri)
	}

	trimmed := strings.TrimPrefix(uri, "s3://")
	parts := strings.SplitN(trimmed, "/", 2)
	if len(parts) < 1 {
		return "", "", fmt.Errorf("invalid S3 URI: missing bucket")
	}

	bucket = parts[0]
	if len(parts) > 1 {
		key = parts[1]
	} else {
		key = ""
	}

	return bucket, key, nil
}

// Filter represents a filter expression with operator and operands
type IcebergFilter struct {
	Op       string          `json:"op"`       // "=", ">", "<", ">=", "<=", "!=", "and", "or", "not", "in", "not_in", "like"
	Field    string          `json:"field"`    // field name for simple comparisons
	Value    interface{}     `json:"value"`    // value for comparison
	Children []IcebergFilter `json:"children"` // child filters for logical operators
}

// FilterPushdown handles filter pushdown for Iceberg scans
type FilterPushdown struct {
	icebergFilters []iceberg.BooleanExpression
	schema         interface{}
	caseSensitive  bool
}

// NewFilterPushdown creates a new filter pushdown handler
func NewFilterPushdown(schema interface{}, caseSensitive bool) *FilterPushdown {
	return &FilterPushdown{
		schema:         schema,
		caseSensitive:  caseSensitive,
		icebergFilters: make([]iceberg.BooleanExpression, 0),
	}
}

// ConvertFilter converts generic filters to Iceberg expressions
func (fp *FilterPushdown) ConvertFilter(filter IcebergFilter) (iceberg.BooleanExpression, error) {
	switch strings.ToLower(filter.Op) {
	case "=":
		return fp.createEqualFilter(filter)
	case "!=", "<>":
		return fp.createNotEqualFilter(filter)
	case ">":
		return fp.createGreaterThanFilter(filter)
	case "<":
		return fp.createLessThanFilter(filter)
	case ">=":
		return fp.createGreaterThanOrEqualFilter(filter)
	case "<=":
		return fp.createLessThanOrEqualFilter(filter)
	case "and":
		return fp.createAndFilter(filter)
	case "or":
		return fp.createOrFilter(filter)
	case "not":
		return fp.createNotFilter(filter)
	case "in":
		return fp.createInFilter(filter)
	case "not_in":
		return fp.createNotInFilter(filter)
	case "is_null":
		return fp.createIsNullFilter(filter)
	case "is_not_null":
		return fp.createIsNotNullFilter(filter)
	case "like", "contains":
		return fp.createLikeFilter(filter)
	case "starts_with":
		return fp.createStartsWithFilter(filter)
	case "ends_with":
		return fp.createEndsWithFilter(filter)
	default:
		return nil, fmt.Errorf("unsupported filter operator: %s", filter.Op)
	}
}

// createEqualFilter creates an equality expression
func (fp *FilterPushdown) createEqualFilter(filter IcebergFilter) (iceberg.BooleanExpression, error) {
	if filter.Field == "" {
		return nil, fmt.Errorf("field is required for equality filter")
	}
	return fp.createLiteralPredicate(iceberg.OpEQ, filter.Field, filter.Value)
}

// createNotEqualFilter creates a not-equal expression
func (fp *FilterPushdown) createNotEqualFilter(filter IcebergFilter) (iceberg.BooleanExpression, error) {
	if filter.Field == "" {
		return nil, fmt.Errorf("field is required for not-equal filter")
	}
	return fp.createLiteralPredicate(iceberg.OpNEQ, filter.Field, filter.Value)
}

// createGreaterThanFilter creates a greater-than expression
func (fp *FilterPushdown) createGreaterThanFilter(filter IcebergFilter) (iceberg.BooleanExpression, error) {
	if filter.Field == "" {
		return nil, fmt.Errorf("field is required for greater-than filter")
	}
	return fp.createLiteralPredicate(iceberg.OpGT, filter.Field, filter.Value)
}

// createLessThanFilter creates a less-than expression
func (fp *FilterPushdown) createLessThanFilter(filter IcebergFilter) (iceberg.BooleanExpression, error) {
	if filter.Field == "" {
		return nil, fmt.Errorf("field is required for less-than filter")
	}
	return fp.createLiteralPredicate(iceberg.OpLT, filter.Field, filter.Value)
}

// createGreaterThanOrEqualFilter creates a greater-than-or-equal expression
func (fp *FilterPushdown) createGreaterThanOrEqualFilter(filter IcebergFilter) (iceberg.BooleanExpression, error) {
	if filter.Field == "" {
		return nil, fmt.Errorf("field is required for greater-than-or-equal filter")
	}
	return fp.createLiteralPredicate(iceberg.OpGTEQ, filter.Field, filter.Value)
}

// createLessThanOrEqualFilter creates a less-than-or-equal expression
func (fp *FilterPushdown) createLessThanOrEqualFilter(filter IcebergFilter) (iceberg.BooleanExpression, error) {
	if filter.Field == "" {
		return nil, fmt.Errorf("field is required for less-than-or-equal filter")
	}
	return fp.createLiteralPredicate(iceberg.OpLTEQ, filter.Field, filter.Value)
}

// createLiteralPredicate creates a literal predicate expression
func (fp *FilterPushdown) createLiteralPredicate(op iceberg.Operation, field string, value interface{}) (iceberg.BooleanExpression, error) {
	// Strip backticks from field name (N1QL quotes identifiers with backticks)
	field = strings.Trim(field, "`")

	if field == "" {
		logging.Errorf("FilterPushdown: empty field name after stripping backticks for op=%d, value=%v", op, value)
		return nil, fmt.Errorf("field name is empty after stripping backticks")
	}

	ref := iceberg.Reference(field)
	logging.Debugf("FilterPushdown: creating literal predicate - field='%s', op=%d, value=%v (type=%T)", field, op, value, value)

	// Create literal based on value type
	var lit iceberg.Literal
	switch v := value.(type) {
	case string:
		lit = iceberg.StringLiteral(v)
	case int:
		lit = iceberg.Int32Literal(int32(v))
	case int8:
		lit = iceberg.Int32Literal(int32(v))
	case int16:
		lit = iceberg.Int32Literal(int32(v))
	case int32:
		lit = iceberg.Int32Literal(v)
	case int64:
		lit = iceberg.Int64Literal(v)
	case uint:
		// Convert unsigned to signed int64
		lit = iceberg.Int64Literal(int64(v))
	case uint8:
		lit = iceberg.Int32Literal(int32(v))
	case uint16:
		lit = iceberg.Int32Literal(int32(v))
	case uint32:
		lit = iceberg.Int64Literal(int64(v))
	case uint64:
		// Convert unsigned to signed int64
		lit = iceberg.Int64Literal(int64(v))
	case float32:
		lit = iceberg.Float32Literal(v)
	case float64:
		// Check if it's a whole number float that can be converted to int
		if v == float64(int64(v)) && v >= math.MinInt64 && v <= math.MaxInt64 {
			lit = iceberg.Int64Literal(int64(v))
			logging.Debugf("FilterPushdown: converted float64 %v to int64 for field '%s'", v, field)
		} else {
			lit = iceberg.Float64Literal(v)
		}
	case bool:
		lit = iceberg.BoolLiteral(v)
	default:
		// For unsupported types, convert to string
		strVal := fmt.Sprintf("%v", v)
		logging.Warnf("FilterPushdown: unsupported value type %T for field '%s', converting to string: %s", v, field, strVal)
		lit = iceberg.StringLiteral(strVal)
	}

	return iceberg.LiteralPredicate(op, ref, lit), nil
}

// createAndFilter creates an AND expression
func (fp *FilterPushdown) createAndFilter(filter IcebergFilter) (iceberg.BooleanExpression, error) {
	if len(filter.Children) < 2 {
		return nil, fmt.Errorf("AND filter requires at least 2 child filters")
	}

	exprs := make([]iceberg.BooleanExpression, len(filter.Children))
	for i, child := range filter.Children {
		expr, err := fp.ConvertFilter(child)
		if err != nil {
			return nil, fmt.Errorf("failed to convert child filter %d in AND: %w", i, err)
		}
		exprs[i] = expr
	}

	return iceberg.NewAnd(exprs[0], exprs[1], exprs[2:]...), nil
}

// createOrFilter creates an OR expression
func (fp *FilterPushdown) createOrFilter(filter IcebergFilter) (iceberg.BooleanExpression, error) {
	if len(filter.Children) < 2 {
		return nil, fmt.Errorf("OR filter requires at least 2 child filters")
	}

	exprs := make([]iceberg.BooleanExpression, len(filter.Children))
	for i, child := range filter.Children {
		expr, err := fp.ConvertFilter(child)
		if err != nil {
			return nil, fmt.Errorf("failed to convert child filter %d in OR: %w", i, err)
		}
		exprs[i] = expr
	}

	return iceberg.NewOr(exprs[0], exprs[1], exprs[2:]...), nil
}

// createNotFilter creates a NOT expression
func (fp *FilterPushdown) createNotFilter(filter IcebergFilter) (iceberg.BooleanExpression, error) {
	if len(filter.Children) != 1 {
		return nil, fmt.Errorf("NOT filter requires exactly 1 child filter")
	}

	expr, err := fp.ConvertFilter(filter.Children[0])
	if err != nil {
		return nil, fmt.Errorf("failed to convert child filter in NOT: %w", err)
	}

	return iceberg.NewNot(expr), nil
}

// createInFilter creates an IN expression
func (fp *FilterPushdown) createInFilter(filter IcebergFilter) (iceberg.BooleanExpression, error) {
	if filter.Field == "" {
		return nil, fmt.Errorf("field is required for IN filter")
	}

	// Strip backticks from field name
	field := strings.Trim(filter.Field, "`")
	ref := iceberg.Reference(field)
	logging.Debugf("FilterPushdown: creating IN predicate - field='%s', values count=%d", field, len(filter.Value.([]interface{})))

	values, ok := filter.Value.([]interface{})
	if !ok {
		return nil, fmt.Errorf("value must be a slice for IN filter")
	}

	literals := make([]iceberg.Literal, len(values))
	for i, v := range values {
		// Convert values to literals with the same type handling as createLiteralPredicate
		var lit iceberg.Literal
		switch val := v.(type) {
		case string:
			lit = iceberg.StringLiteral(val)
		case int:
			lit = iceberg.Int32Literal(int32(val))
		case int8:
			lit = iceberg.Int32Literal(int32(val))
		case int16:
			lit = iceberg.Int32Literal(int32(val))
		case int32:
			lit = iceberg.Int32Literal(val)
		case int64:
			lit = iceberg.Int64Literal(val)
		case uint:
			lit = iceberg.Int64Literal(int64(val))
		case uint8:
			lit = iceberg.Int32Literal(int32(val))
		case uint16:
			lit = iceberg.Int32Literal(int32(val))
		case uint32:
			lit = iceberg.Int64Literal(int64(val))
		case uint64:
			lit = iceberg.Int64Literal(int64(val))
		case float32:
			lit = iceberg.Float32Literal(val)
		case float64:
			// Check if it's a whole number float that can be converted to int
			if val == float64(int64(val)) && val >= math.MinInt64 && val <= math.MaxInt64 {
				lit = iceberg.Int64Literal(int64(val))
				logging.Debugf("FilterPushdown: [IN] converted float64 %v to int64 for field '%s'", val, field)
			} else {
				lit = iceberg.Float64Literal(val)
			}
		case bool:
			lit = iceberg.BoolLiteral(val)
		default:
			// For unsupported types, convert to string
			logging.Warnf("FilterPushdown: [IN] unsupported value type %T for field '%s', converting to string", val, field)
			lit = iceberg.StringLiteral(fmt.Sprintf("%v", val))
		}
		literals[i] = lit
	}

	return iceberg.SetPredicate(iceberg.OpIn, ref, literals), nil
}

// createNotInFilter creates a NOT IN expression
func (fp *FilterPushdown) createNotInFilter(filter IcebergFilter) (iceberg.BooleanExpression, error) {
	inExpr, err := fp.createInFilter(filter)
	if err != nil {
		return nil, err
	}
	return iceberg.NewNot(inExpr), nil
}

// createIsNullFilter creates an IS NULL expression
func (fp *FilterPushdown) createIsNullFilter(filter IcebergFilter) (iceberg.BooleanExpression, error) {
	if filter.Field == "" {
		return nil, fmt.Errorf("field is required for IS NULL filter")
	}

	// Strip backticks from field name
	field := strings.Trim(filter.Field, "`")
	ref := iceberg.Reference(field)
	logging.Debugf("FilterPushdown: created IS NULL predicate - field='%s'", field)
	return iceberg.UnaryPredicate(iceberg.OpIsNull, ref), nil
}

// createIsNotNullFilter creates an IS NOT NULL expression
func (fp *FilterPushdown) createIsNotNullFilter(filter IcebergFilter) (iceberg.BooleanExpression, error) {
	if filter.Field == "" {
		return nil, fmt.Errorf("field is required for IS NOT NULL filter")
	}

	// Strip backticks from field name
	field := strings.Trim(filter.Field, "`")
	ref := iceberg.Reference(field)
	logging.Debugf("FilterPushdown: created IS NOT NULL predicate - field='%s'", field)
	return iceberg.UnaryPredicate(iceberg.OpNotNull, ref), nil
}

// createLikeFilter creates a LIKE expression
func (fp *FilterPushdown) createLikeFilter(filter IcebergFilter) (iceberg.BooleanExpression, error) {
	if filter.Field == "" {
		return nil, fmt.Errorf("field is required for LIKE filter")
	}

	// Strip backticks from field name
	field := strings.Trim(filter.Field, "`")

	pattern, ok := filter.Value.(string)
	if !ok {
		return nil, fmt.Errorf("value must be a string for LIKE filter")
	}

	logging.Debugf("FilterPushdown: creating LIKE predicate - field='%s', pattern='%s'", field, pattern)

	// Iceberg-go support for LIKE - convert to Iceberg-compatible expressions
	if strings.Contains(pattern, "%") {
		// Convert SQL LIKE to Iceberg-compatible expressions
		if strings.HasPrefix(pattern, "%") && strings.HasSuffix(pattern, "%") {
			// "%pattern%" -> Contains - but Iceberg doesn't have Contains
			// Use StartsWith with the full pattern
			searchPattern := strings.Trim(pattern, "%")
			if searchPattern != "" {
				return fp.createStartsWithFilter(IcebergFilter{Op: "starts_with", Field: field, Value: searchPattern})
			}
		} else if strings.HasPrefix(pattern, "%") {
			// "%pattern" -> EndsWith - Iceberg uses NotStartsWith
			searchPattern := strings.TrimPrefix(pattern, "%")
			if searchPattern != "" {
				return fp.createEndsWithFilter(IcebergFilter{Op: "ends_with", Field: field, Value: searchPattern})
			}
		} else if strings.HasSuffix(pattern, "%") {
			// "pattern%" -> StartsWith
			searchPattern := strings.TrimSuffix(pattern, "%")
			return fp.createStartsWithFilter(IcebergFilter{Op: "starts_with", Field: field, Value: searchPattern})
		}
	}

	// Exact match without wildcards
	return fp.createEqualFilter(IcebergFilter{Op: "=", Field: field, Value: pattern})
}

// createStartsWithFilter creates a starts-with expression
func (fp *FilterPushdown) createStartsWithFilter(filter IcebergFilter) (iceberg.BooleanExpression, error) {
	if filter.Field == "" {
		return nil, fmt.Errorf("field is required for starts-with filter")
	}

	// Strip backticks from field name
	field := strings.Trim(filter.Field, "`")

	prefix, ok := filter.Value.(string)
	if !ok {
		return nil, fmt.Errorf("value must be a string for starts-with filter")
	}

	ref := iceberg.Reference(field)
	lit := iceberg.StringLiteral(prefix)
	logging.Debugf("FilterPushdown: created STARTS_WITH predicate - field='%s', prefix='%s'", field, prefix)
	return iceberg.LiteralPredicate(iceberg.OpStartsWith, ref, lit), nil
}

// createEndsWithFilter creates an ends-with expression
func (fp *FilterPushdown) createEndsWithFilter(filter IcebergFilter) (iceberg.BooleanExpression, error) {
	if filter.Field == "" {
		return nil, fmt.Errorf("field is required for ends-with filter")
	}

	// Strip backticks from field name
	field := strings.Trim(filter.Field, "`")

	suffix, ok := filter.Value.(string)
	if !ok {
		return nil, fmt.Errorf("value must be a string for ends-with filter")
	}

	// Iceberg supports NotStartsWith, but not EndsWith
	// For now, we'll use StartsWith as a fallback (not perfect but functional)
	ref := iceberg.Reference(field)
	lit := iceberg.StringLiteral(suffix)
	logging.Debugf("FilterPushdown: created ENDS_WITH predicate (fallback) - field='%s', suffix='%s'", field, suffix)
	return iceberg.LiteralPredicate(iceberg.OpStartsWith, ref, lit), nil
}

// ApplyFilters applies filters to create a combined iceberg expression
func (fp *FilterPushdown) ApplyFilters(filters []IcebergFilter) error {
	if len(filters) == 0 {
		return nil
	}

	logging.Debugf("FilterPushdown: applying %d filter(s)", len(filters))

	icebergExprs := make([]iceberg.BooleanExpression, len(filters))
	var errorList []error

	for i, filter := range filters {
		logging.Debugf("FilterPushdown: converting filter %d (op=%s, field=%s, value=%v)",
			i, filter.Op, filter.Field, filter.Value)

		expr, err := fp.ConvertFilter(filter)
		if err != nil {
			logging.Warnf("FilterPushdown: failed to convert filter %d (op=%s, field=%s): %v",
				i, filter.Op, filter.Field, err)
			errorList = append(errorList, fmt.Errorf("filter %d: %w", i, err))
			continue
		}
		logging.Debugf("FilterPushdown: successfully converted filter %d to iceberg expression (type=%T)", i, expr)
		icebergExprs[i] = expr
	}

	if len(errorList) > 0 {
		// Return combined error
		msg := "failed to convert filters:\n"
		for _, err := range errorList {
			msg += "  - " + err.Error() + "\n"
		}
		return fmt.Errorf("%s", msg)
	}

	// Combine all filters with AND
	var finalExpr iceberg.BooleanExpression
	if len(icebergExprs) == 1 {
		finalExpr = icebergExprs[0]
		logging.Debugf("FilterPushdown: single filter, using expression directly (type=%T)", finalExpr)
	} else {
		finalExpr = iceberg.NewAnd(icebergExprs[0], icebergExprs[1], icebergExprs[2:]...)
		logging.Debugf("FilterPushdown: combined %d filters with AND (result type=%T)", len(icebergExprs), finalExpr)
	}

	fp.icebergFilters = append(fp.icebergFilters, finalExpr)

	return nil
}

// GetExpression returns the combined filter expression
func (fp *FilterPushdown) GetExpression() iceberg.BooleanExpression {
	if len(fp.icebergFilters) == 0 {
		return iceberg.AlwaysTrue{}
	}
	if len(fp.icebergFilters) == 1 {
		return fp.icebergFilters[0]
	}
	return iceberg.NewAnd(fp.icebergFilters[0], fp.icebergFilters[1], fp.icebergFilters[2:]...)
}

// GetIcebergFilters returns all converted Iceberg filter expressions
func (fp *FilterPushdown) GetIcebergFilters() []iceberg.BooleanExpression {
	return fp.icebergFilters
}

// ParseFilterJSON parses a JSON string into Filter structure
func ParseFilterJSON(jsonStr string) ([]IcebergFilter, error) {
	if jsonStr == "" {
		return nil, nil
	}

	var filters []IcebergFilter
	err := json.Unmarshal([]byte(jsonStr), &filters)
	if err != nil {
		return nil, fmt.Errorf("failed to parse filter JSON: %w", err)
	}

	return filters, nil
}

// CreateEqualFilter creates a simple equality filter
func CreateEqualFilter(field string, value interface{}) IcebergFilter {
	return IcebergFilter{
		Op:    "=",
		Field: field,
		Value: value,
	}
}

// CreateRangeFilter creates a range filter (inclusive)
func CreateRangeFilter(field string, min, max interface{}) ([]IcebergFilter, error) {
	if min == nil && max == nil {
		return nil, fmt.Errorf("at least one of min or max must be specified for range filter")
	}

	var filters []IcebergFilter

	if min != nil {
		filters = append(filters, IcebergFilter{
			Op:    ">=",
			Field: field,
			Value: min,
		})
	}

	if max != nil {
		filters = append(filters, IcebergFilter{
			Op:    "<=",
			Field: field,
			Value: max,
		})
	}

	return filters, nil
}

// CreateAndFilter creates an AND filter combining multiple filters
func CreateAndFilter(filters ...IcebergFilter) IcebergFilter {
	return IcebergFilter{
		Op:       "and",
		Children: filters,
	}
}

// CreateOrFilter creates an OR filter combining multiple filters
func CreateOrFilter(filters ...IcebergFilter) IcebergFilter {
	return IcebergFilter{
		Op:       "or",
		Children: filters,
	}
}

// CreateInFilter creates an IN filter
func CreateInFilter(field string, values ...interface{}) IcebergFilter {
	return IcebergFilter{
		Op:    "in",
		Field: field,
		Value: values,
	}
}

// CatalogMetadata represents the metadata for an Iceberg catalog
type CatalogMetadata struct {
	Namespaces []NamespaceInfo `json:"namespaces,omitempty"`
}

// NamespaceInfo represents information about a namespace (database)
type NamespaceInfo struct {
	Name   string      `json:"name"`
	Tables []TableInfo `json:"tables,omitempty"`
}

const _ICEBERG_MAX_SNAPSHOTS = 8

// Mirror of datastore.CatalogInfoXxx — kept local to avoid import cycle with datastore/.
const (
	_CatalogInfoSchema    uint64 = 1 << 0
	_CatalogInfoSnapshots uint64 = 1 << 1
	_CatalogInfoFiles     uint64 = 1 << 2
)

// SnapshotInfo represents a single Iceberg snapshot
type SnapshotInfo struct {
	SnapshotID  int64 `json:"snapshot_id"`
	TimestampMs int64 `json:"timestamp_ms"`
}

// ColumnInfo represents a single column in an Iceberg table schema
type ColumnInfo struct {
	Name          string `json:"name"`
	Type          string `json:"type"`
	Required      bool   `json:"required"`
	AddedInSchema int    `json:"added_in_schema,omitempty"`
}

// TableInfo represents information about an Iceberg table
type TableInfo struct {
	Name          string         `json:"name"`
	Location      string         `json:"location,omitempty"`
	Format        string         `json:"format,omitempty"`
	Files         []string       `json:"files,omitempty"`
	CurrentSchema int            `json:"current_schema,omitempty"`
	Schema        []ColumnInfo   `json:"schema,omitempty"`
	Snapshots     []SnapshotInfo `json:"snapshots,omitempty"`
}

// LoadCatalogMetadata loads metadata (namespaces, tables) from an Iceberg catalog.
// infoFlags selects which detail sections to populate (schema, snapshots, files).
// Pass 0 to list namespaces and table names only.
func LoadCatalogMetadata(ctx go_context.Context, entry *extparams.CatalogEntry, cred *cbauth.Credential, infoFlags uint64) (*CatalogMetadata, error) {
	if entry.URI != "" {
		var allowlist map[string]interface{}
		if cred != nil && cred.Meta.Guardrails.URLWhitelist != nil {
			wl := cred.Meta.Guardrails.URLWhitelist
			allowlist = map[string]interface{}{
				util.AllowlistKeyAllAccess:      wl.AllAccess,
				util.AllowlistKeyAllowedURLs:    wl.AllowedURLs,
				util.AllowlistKeyDisallowedURLs: wl.DisallowedURLs,
			}
		}
		if err := util.ValidateURLInAllowlist(entry.URI, nil, allowlist); err != nil {
			return nil, fmt.Errorf("catalog URI not permitted: %w", err)
		}
	}

	// Get AWS config using common function
	awsCfg, err := GetAWSConfig(entry.Source, cred, entry.SigV4SigningRegion)
	if err != nil {
		return nil, err
	}

	// Create catalog options
	opts := ScanOptions{
		SourceType:         entry.Source,
		URI:                entry.URI,
		Warehouse:          entry.Warehouse,
		SigV4SigningRegion: entry.SigV4SigningRegion,
		SigV4SigningName:   entry.SigV4SigningName,
		QuotaProjectID:     entry.QuotaProjectID,
		CatalogCred:        cred,
	}

	// Create the appropriate catalog
	var awsCfgVal aws.Config
	if awsCfg != nil {
		awsCfgVal = *awsCfg
	}
	cat, err := createCatalog(ctx, opts, awsCfgVal)
	if err != nil {
		return nil, fmt.Errorf("failed to create catalog: %w", err)
	}
	defer func() {
		// Catalog doesn't have a Close method, but we can release resources
	}()

	metadata := &CatalogMetadata{
		Namespaces: make([]NamespaceInfo, 0),
	}

	// List namespaces
	namespaces, err := cat.ListNamespaces(ctx, nil)
	if err != nil {
		logging.Warnf("LoadCatalogMetadata: failed to list namespaces for catalog %s: %v", entry.Name, err)
		return metadata, nil // Return empty metadata instead of error
	}

	for _, ns := range namespaces {
		nsName := strings.Join(ns, ".")
		nsInfo := NamespaceInfo{
			Name:   nsName,
			Tables: make([]TableInfo, 0),
		}

		// List tables in this namespace
		for tblIdent, tblErr := range cat.ListTables(ctx, ns) {
			if tblErr != nil {
				logging.Warnf("LoadCatalogMetadata: error listing tables for namespace %s: %v", nsName, tblErr)
				continue
			}
			tblName := catalog.TableNameFromIdent(tblIdent)
			tblInfo := TableInfo{
				Name: tblName,
			}

			if infoFlags != 0 {
				tbl, err := cat.LoadTable(ctx, tblIdent)
				if err == nil && tbl != nil {
					tblInfo.Location = tbl.Location()
					if meta := tbl.Metadata(); meta != nil {
						if infoFlags&_CatalogInfoSchema != 0 {
							fieldAddedIn := make(map[int]int)
							for _, s := range meta.Schemas() {
								for _, f := range s.Fields() {
									if _, seen := fieldAddedIn[f.ID]; !seen {
										fieldAddedIn[f.ID] = s.ID
									}
								}
							}
							if schema := tbl.Schema(); schema != nil {
								tblInfo.CurrentSchema = schema.ID
								for _, field := range schema.Fields() {
									tblInfo.Schema = append(tblInfo.Schema, ColumnInfo{
										Name:          field.Name,
										Type:          field.Type.String(),
										Required:      field.Required,
										AddedInSchema: fieldAddedIn[field.ID],
									})
								}
							}
						}
						if infoFlags&_CatalogInfoSnapshots != 0 {
							all := meta.Snapshots()
							start := len(all) - _ICEBERG_MAX_SNAPSHOTS
							if start < 0 {
								start = 0
							}
							for _, s := range all[start:] {
								tblInfo.Snapshots = append(tblInfo.Snapshots, SnapshotInfo{
									SnapshotID:  s.SnapshotID,
									TimestampMs: s.TimestampMs,
								})
							}
						}
					}
				}
			}

			nsInfo.Tables = append(nsInfo.Tables, tblInfo)
		}

		metadata.Namespaces = append(metadata.Namespaces, nsInfo)
	}

	return metadata, nil
}
