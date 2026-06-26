//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package ai_gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
)

// gateway.go is the vendor-agnostic engine. It owns the Provider contract every
// vendor implements, the provider registry, the request orchestration
// (DoChatCompletion) and the shared HTTP transport (doProviderRequest: auth +
// static headers + retry). None of this is provider-specific.
//
// The common case is an HTTP vendor: the orchestrator builds the vendor-native
// payload, posts it through the shared transport and parses the response. A
// vendor whose protocol is not plain HTTP-POST (e.g. an SDK-based one) instead
// implements the optional DirectCompleter interface and owns its whole
// round-trip; the orchestrator dispatches to it before the HTTP path.
//
// Adding an HTTP vendor means adding a Provider implementation plus a
// registerProvider call; the orchestration and transport do not change.

// Context is what the gateway needs from the caller's query execution context.
// It is satisfied by the natural package's NaturalContext and is asserted to
// expression.Context inside the transport for credstore credential resolution.
// Declaring it here keeps the gateway free of any dependency on the natural
// package.
type Context interface {
	datastore.Context
	datastore.QueryContext
}

// Transport tunables shared by all HTTP providers. The backoff is a var, not a
// const, so unit tests can shrink it; production code never mutates it.
var _COMPLETIONS_REQ_BACKOFF_INIT = 1 * time.Second

const _COMPLETIONS_REQ_RETRY = 5
const httpClientTimeout = 2 * time.Minute

// httpClient is the long-lived client used for inline-auth requests. The
// credstore path supplies its own client via expression.HandleCred.
var httpClient = &http.Client{
	Timeout:       httpClientTimeout,
	CheckRedirect: noRedirect,
}

// noRedirect stops the HTTP client from following redirects. Every outbound URL
// must pass the cluster allowlist (see IsUrlAllowedInCluster in
// doProviderRequest); a redirect's Location is never vetted by that gate, so
// following it would bypass the allowlist (and leak provider auth headers to an
// unvetted host). Returning ErrUseLastResponse makes Do hand back the 3xx
// response without following it, and doProviderRequest rejects it.
func noRedirect(req *http.Request, via []*http.Request) error {
	return http.ErrUseLastResponse
}

// Provider is the contract a single LLM vendor implements. Everything that
// differs between HTTP vendors is expressed here; the orchestration in
// DoChatCompletion and the transport in doProviderRequest are vendor-neutral.
type Provider interface {
	// ID is the lowercase provider identifier used in natural_config.provider and the
	// registry (e.g. "openai").
	ID() string

	// DefaultModel is used when natural_config does not name a model.
	DefaultModel() string

	// ChatEndpoint returns the URL to POST a completion to, honouring an
	// optional caller-supplied override in cfg.Endpoint.
	ChatEndpoint(cfg *Config) string

	// BuildChatRequest renders the neutral request into the vendor-native body.
	BuildChatRequest(req *Request, cfg *Config) ([]byte, errors.Error)

	// ParseChatResponse turns the raw HTTP status and body into the normalized
	// response, or an error describing the failure.
	ParseChatResponse(status int, body []byte) (*Response, errors.Error)

	// Authorize applies the vendor's auth scheme to a request when an inline
	// api_key is used (the credstore path is handled by the transport). It may
	// set headers, sign the request, or add a query parameter.
	Authorize(httpReq *http.Request, cfg *Config) errors.Error

	// StaticHeaders returns non-auth headers that must be present on every
	// request regardless of the auth source (e.g. Anthropic's anthropic-version).
	// The transport applies them on both the credstore and inline paths. May
	// return nil.
	StaticHeaders() map[string]string

	// AllowsAmbientAuth reports whether the provider can authenticate without an
	// inline api_key or a credstore cred_id, i.e. from the ambient environment
	// (e.g. Bedrock via the AWS default credential chain: env / shared config /
	// IAM role). Providers that require an explicit key return false, and the
	// config layer then rejects a request that supplies neither credential -
	// unless a custom endpoint is configured, in which case the endpoint's owner
	// decides the auth policy and a credential is accepted but not required.
	AllowsAmbientAuth() bool

	// Moderate optionally screens user content before the completion request.
	// Providers without a moderation facility return nil.
	Moderate(req *Request, cfg *Config, ctx Context) errors.Error
}

// DirectCompleter is the optional interface a provider implements when its
// protocol is not the shared HTTP-POST path (e.g. an SDK-based vendor such as
// AWS Bedrock). The orchestrator dispatches to Complete before the HTTP path,
// so such a provider owns its entire request/response cycle and leaves the
// HTTP-shaped Provider methods as no-ops.
type DirectCompleter interface {
	Complete(ctx Context, req *Request, cfg *Config) (*Response, errors.Error)
}

// providers is the in-engine catalog of supported vendors. Each provider
// registers itself in an init(); adding a vendor is a new Provider
// implementation plus a registerProvider call, with no changes to the
// orchestration or transport.
var providers = map[string]Provider{}

func registerProvider(p Provider) {
	providers[p.ID()] = p
}

// providerFor returns the registered provider for a (lowercased) provider id.
func providerFor(provider string) (Provider, errors.Error) {
	p, ok := providers[provider]
	if !ok {
		return nil, errors.NewNaturalLanguageRequestError(errors.E_NL_VENDOR_NOT_SUPPORTED, provider)
	}
	return p, nil
}

// DoChatCompletion sends the request to the configured provider and returns the
// normalized completion. A provider implementing DirectCompleter owns its whole
// round-trip; otherwise the request flows through the shared HTTP path: run
// content moderation, build the provider-native payload, post it and parse the
// response. No output token limit is imposed unless the caller set
// "output_token_limit" in natural_config.
func DoChatCompletion(req *Request, cfg *Config, ctx Context) (*Response, errors.Error) {
	prov, err := providerFor(cfg.Provider)
	if err != nil {
		return nil, err
	}

	// SDK-based vendors bypass the HTTP transport entirely.
	if dc, ok := prov.(DirectCompleter); ok {
		return dc.Complete(ctx, req, cfg)
	}

	// Moderate user-supplied content before sending the completion request,
	// unless the caller opted out with "moderation":false in natural_config. Opt-
	// out is honored for every provider/endpoint so a caller pointing at an
	// OpenAI-compatible endpoint that does not implement the /moderations API
	// (e.g. AWS Bedrock's OpenAI-compat surface) can proceed.
	//
	// By design, moderation and the completion below are each bounded by their
	// own full-length deadline derived from the query timeout (GetTimeout), not a
	// single shared budget: Moderate calls requestContext internally, and the
	// completion derives a fresh one at reqCtx below. They run serially, so an
	// OpenAI request that moderates can consume up to ~2x the configured request
	// timeout in wall-clock. This is an accepted trade-off for the basic drop;
	// give the two phases a shared deadline if that total needs to be capped.
	if cfg.Moderation == nil || *cfg.Moderation {
		if err := prov.Moderate(req, cfg, ctx); err != nil {
			return nil, err
		}
	}

	payload, err := prov.BuildChatRequest(req, cfg)
	if err != nil {
		return nil, err
	}

	// Bound the completion round-trip - the request send below and the body read
	// that follows it - by the query deadline, so a cancelled or timed-out query
	// aborts an in-flight LLM call instead of blocking the query thread for the
	// full HTTP client timeout. cancel must outlive the body read, so it is held
	// here rather than inside doProviderRequest.
	reqCtx, cancel := requestContext(ctx)
	defer cancel()

	resp, err := doProviderRequest(reqCtx, prov.ChatEndpoint(cfg), payload, prov, cfg, ctx)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, rerr := io.ReadAll(resp.Body)
	if rerr != nil {
		return nil, errors.NewNaturalLanguageRequestError(errors.E_NL_ERR_CHATCOMPLETIONS_RESP, rerr)
	}
	return prov.ParseChatResponse(resp.StatusCode, body)
}

// doProviderRequest authenticates and POSTs the payload to the provider URL,
// retrying transient failures (429 / 5xx) with exponential backoff. Auth comes
// from the credstore when cred_id is set, otherwise the provider's Authorize
// applies the inline key. The provider's static headers are applied on both
// paths. The returned response (when non-nil) is the caller's to read and close.
func doProviderRequest(reqCtx context.Context, endpoint string, payload []byte, prov Provider, cfg *Config,
	ctx Context) (*http.Response, errors.Error) {

	u, e := url.Parse(endpoint)
	if e != nil {
		return nil, errors.NewNaturalLanguageRequestError(errors.E_NL_DIRECT_CREATE_REQ, endpoint, e)
	}

	// The execution context carries the cluster allowlist and is also required
	// for credstore credential resolution.
	exprCtx, ok := ctx.(expression.Context)
	if !ok {
		return nil, errors.NewNaturalLanguageRequestError(errors.E_NL_CRED_RESOLUTION_FAILED,
			fmt.Errorf("no execution context available"))
	}

	// Cluster allowlist gate: every outbound endpoint must be permitted by the
	// cluster allowlist, on both the inline-key and credstore paths. This also
	// covers the moderation endpoint, which flows through this same function.
	if ae := expression.IsUrlAllowedInCluster(u, exprCtx); ae != nil {
		return nil, errors.NewNaturalLanguageRequestError(errors.E_NL_URL_NOT_ALLOWED, ae.Error())
	}

	var client *http.Client
	baseHeader := http.Header{}
	useInlineAuth := false

	if cfg.CredId != "" {
		c, h, ce := expression.HandleCred(u, cfg.CredId, exprCtx)
		if ce != nil {
			return nil, errors.NewNaturalLanguageRequestError(errors.E_NL_CRED_RESOLUTION_FAILED, ce)
		}
		client = &c
		client.CheckRedirect = noRedirect
		if h != nil {
			baseHeader = h
		}
	} else {
		client = httpClient
		useInlineAuth = true
	}
	baseHeader.Set("Content-Type", "application/json")

	// Non-auth headers required regardless of the auth source.
	for k, v := range prov.StaticHeaders() {
		baseHeader.Set(k, v)
	}

	backoff := _COMPLETIONS_REQ_BACKOFF_INIT
	for attempt := 0; attempt < _COMPLETIONS_REQ_RETRY; attempt++ {
		req, e := http.NewRequestWithContext(reqCtx, "POST", endpoint, bytes.NewBuffer(payload))
		if e != nil {
			return nil, errors.NewNaturalLanguageRequestError(errors.E_NL_DIRECT_CREATE_REQ, endpoint, e)
		}
		req.Header = baseHeader.Clone()
		if useInlineAuth {
			if ae := prov.Authorize(req, cfg); ae != nil {
				return nil, ae
			}
		}

		r, e := client.Do(req)
		if e != nil {
			return nil, errors.NewNaturalLanguageRequestError(errors.E_NL_DIRECT_SEND_REQ, endpoint, e)
		}

		// Redirects are not followed (see noRedirect): the Location has not passed
		// the cluster allowlist, so following it would bypass the SSRF gate above.
		// Reject the 3xx rather than handing it back as a chat response.
		if r.StatusCode >= 300 && r.StatusCode < 400 {
			loc := r.Header.Get("Location")
			r.Body.Close()
			return nil, errors.NewNaturalLanguageRequestError(errors.E_NL_URL_NOT_ALLOWED,
				fmt.Sprintf("endpoint returned redirect (%d) to %q; redirects are not permitted", r.StatusCode, loc))
		}

		// Success or a non-transient status: hand the response back as-is.
		if r.StatusCode != http.StatusTooManyRequests && r.StatusCode < 500 {
			return r, nil
		}

		// Transient failure (429 / 5xx). On the final attempt, hand the response
		// back with its body intact so the caller can surface the provider's
		// error detail; otherwise discard it and back off before retrying.
		if attempt == _COMPLETIONS_REQ_RETRY-1 {
			return r, nil
		}
		r.Body.Close()

		// Back off before retrying, but abort the wait if the query is cancelled
		// or times out. A bare time.Sleep cannot be interrupted, so it would keep
		// the query thread blocked for the full backoff (up to ~15s across the
		// schedule) even after reqCtx is done - defeating the deadline the caller
		// bounded this request with.
		timer := time.NewTimer(backoff)
		select {
		case <-reqCtx.Done():
			timer.Stop()
			return nil, errors.NewNaturalLanguageRequestError(errors.E_NL_DIRECT_SEND_REQ, endpoint, reqCtx.Err())
		case <-timer.C:
		}
		backoff *= 2
	}

	// Unreachable: _COMPLETIONS_REQ_RETRY is a positive constant, so the loop
	// always returns. Present to satisfy the compiler.
	return nil, errors.NewNaturalLanguageRequestError(errors.E_NL_DIRECT_SEND_REQ, endpoint,
		fmt.Errorf("no request attempts were made"))
}

// requestContext derives a stdlib context bounded by the query deadline (when
// the execution context exposes one via GetTimeout) so an outbound provider
// call is cancelled when the query is cancelled or times out, rather than
// blocking for the full HTTP client timeout. Mirrors the Bedrock path, which
// derives the same bound for its SDK calls. The returned cancel must be invoked
// only after the response body has been read, so callers hold it at the scope
// that owns the body. A nil or non-expression context (e.g. in unit tests)
// yields an uncancelled background context.
func requestContext(ctx Context) (context.Context, context.CancelFunc) {
	if exprCtx, ok := ctx.(expression.Context); ok {
		if d := exprCtx.GetTimeout(); d > 0 {
			return context.WithTimeout(context.Background(), d)
		}
	}
	return context.Background(), func() {}
}

// llmErrCause turns a provider error body into a cause error: a JSON body is
// formatted as a map, otherwise the raw text is used. Returns nil when empty.
func llmErrCause(body []byte) error {
	if len(body) == 0 {
		return nil
	}
	var errRes map[string]interface{}
	if json.Unmarshal(body, &errRes) == nil {
		return fmt.Errorf("%v", errRes)
	}
	return fmt.Errorf("%s", body)
}
