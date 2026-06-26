//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package ai_gateway

import (
	"strings"

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/value"
)

// config.go owns the "natural_config" request-parameter contract: parsing the
// supplied configuration object into a Config and resolving the provider/model
// against the provider registry.

// defaultProvider is used when natural_config does not name a provider.
const defaultProvider = ProviderOpenAI

// Config is the parsed form of the natural_config request parameter.
type Config struct {
	Provider         string
	Model            string
	CredId           string
	APIKey           string
	Endpoint         string
	OutputTokenLimit int
	// Region is the AWS region for SDK-based providers (Bedrock). It is unused by
	// HTTP providers whose endpoint is fixed or supplied via Endpoint.
	Region string
	// Moderation opts content moderation in or out for the request. A nil pointer
	// means the caller did not specify it and moderation runs by default; an
	// explicit false skips it (for OpenAI-compatible endpoints that do not
	// implement the /moderations API, e.g. AWS Bedrock's OpenAI-compat surface).
	Moderation *bool
}

// ParseConfig reads and validates the shape of the natural_config object into a
// Config. The value is expected to be an object carrying the provider, model
// and credentials. The credential policy (whether a cred_id or api_key is
// required) is provider-specific and enforced later in ResolveProviderAndModel,
// once the provider is known.
func ParseConfig(v value.Value) (*Config, errors.Error) {
	if v == nil || v.Type() == value.NULL || v.Type() == value.MISSING {
		return nil, errors.NewNaturalLanguageRequestError(errors.E_NL_MISSING_NL_PARAM, "\"natural_config\"")
	}
	if v.Type() != value.OBJECT {
		return nil, errors.NewNaturalLanguageRequestError(errors.E_NL_INVALID_NATURAL_CONFIG,
			"\"natural_config\" must be an object")
	}

	cfg := &Config{}
	var err errors.Error
	if cfg.Provider, err = stringField(v, "provider"); err != nil {
		return nil, err
	}
	if cfg.Model, err = stringField(v, "model"); err != nil {
		return nil, err
	}
	if cfg.CredId, err = stringField(v, "cred_id"); err != nil {
		return nil, err
	}
	if cfg.APIKey, err = stringField(v, "api_key"); err != nil {
		return nil, err
	}
	if cfg.Endpoint, err = stringField(v, "endpoint"); err != nil {
		return nil, err
	}
	if cfg.Region, err = stringField(v, "region"); err != nil {
		return nil, err
	}
	if cfg.OutputTokenLimit, err = intField(v, "output_token_limit"); err != nil {
		return nil, err
	}
	if cfg.Moderation, err = boolField(v, "moderation"); err != nil {
		return nil, err
	}

	return cfg, nil
}

// fieldPresent reports whether name is set to a usable value. A missing field or
// an explicit null is treated as "not provided" so the field falls back to its
// default; only a present, non-null value is validated for type.
func fieldPresent(v value.Value, name string) (value.Value, bool) {
	f, ok := v.Field(name)
	if !ok || f.Type() == value.MISSING || f.Type() == value.NULL {
		return nil, false
	}
	return f, true
}

// stringField reads a string-typed field. A wrong type is a caller error rather
// than a silent fallback to the zero value.
func stringField(v value.Value, name string) (string, errors.Error) {
	f, ok := fieldPresent(v, name)
	if !ok {
		return "", nil
	}
	if f.Type() != value.STRING {
		return "", errors.NewNaturalLanguageRequestError(errors.E_NL_INVALID_NATURAL_CONFIG,
			"\""+name+"\" must be a string")
	}
	return f.ToString(), nil
}

// intField reads an integer-valued field. A non-number or a non-integral number
// is a caller error. IsIntValue does the integrality check against int64, so the
// result is architecture-independent and both float64 and int64 actuals are
// accepted.
func intField(v value.Value, name string) (int, errors.Error) {
	f, ok := fieldPresent(v, name)
	if !ok {
		return 0, nil
	}
	i, ok := value.IsIntValue(f)
	if !ok {
		return 0, errors.NewNaturalLanguageRequestError(errors.E_NL_INVALID_NATURAL_CONFIG,
			"\""+name+"\" must be an integer")
	}
	return int(i), nil
}

// boolField reads a boolean field, returning a pointer so an unset field is
// distinguishable from an explicit false. A wrong type is a caller error.
func boolField(v value.Value, name string) (*bool, errors.Error) {
	f, ok := fieldPresent(v, name)
	if !ok {
		return nil, nil
	}
	b, ok := f.Actual().(bool)
	if f.Type() != value.BOOLEAN || !ok {
		return nil, errors.NewNaturalLanguageRequestError(errors.E_NL_INVALID_NATURAL_CONFIG,
			"\""+name+"\" must be a boolean")
	}
	return &b, nil
}

// ResolveProviderAndModel validates the configured provider against the registry
// and fills in the provider's default model when one is not specified. An empty
// provider defaults to defaultProvider.
func (c *Config) ResolveProviderAndModel() errors.Error {
	if c.Provider == "" {
		c.Provider = defaultProvider
	} else {
		c.Provider = strings.ToLower(c.Provider)
	}

	prov, err := providerFor(c.Provider)
	if err != nil {
		return err
	}

	// Model identifiers are passed to the provider verbatim: several backends
	// treat them as case-sensitive (self-hosted served-model names, Bedrock
	// ARNs), and normalizing would corrupt those. A wrongly-cased id surfaces
	// as the provider's own "model not found" error.
	if c.Model == "" {
		c.Model = prov.DefaultModel()
	}

	// The slm provider has no built-in host, so an endpoint is always required.
	// Checked here so the caller gets the real problem (missing endpoint) rather
	// than the credential error below.
	if c.Provider == ProviderSLM && c.Endpoint == "" {
		return errors.NewNaturalLanguageRequestError(errors.E_NL_INVALID_NATURAL_CONFIG,
			"\"endpoint\" is required for the slm provider")
	}

	// Credential policy. A caller-supplied endpoint is a self-hosted or proxied
	// deployment whose owner decides the auth policy, so a credential is accepted
	// but not required. On a provider's built-in host a credential is required
	// unless the provider can authenticate from the ambient environment (e.g.
	// Bedrock via the AWS default credential chain).
	if c.CredId == "" && c.APIKey == "" && c.Endpoint == "" && !prov.AllowsAmbientAuth() {
		return errors.NewNaturalLanguageRequestError(errors.E_NL_INVALID_NATURAL_CONFIG,
			"\"natural_config\" must specify either \"cred_id\" or \"api_key\"")
	}

	return nil
}
