//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package ai_gateway

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
	"github.com/aws/smithy-go"

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/logging"
)

// provider.go holds every vendor-specific implementation. Each vendor is a
// small struct implementing Provider (and, for non-HTTP vendors, DirectCompleter)
// together with its native request/response shapes, all registered in the single
// init() below. The vendor-agnostic machinery lives in gateway.go.

// Provider identifiers.
const (
	ProviderOpenAI = "openai"
	ProviderGemini = "gemini"
	ProviderSLM    = "slm"
)

// Models.
const (
	GPT4o_2024_05_13 = "gpt-4o-2024-05-13"
	Gemini25Pro      = "gemini-2.5-pro"

	// SLMDefaultModel is the model used for the slm provider when natural_config
	// does not name one. The slm provider targets a self-hosted, OpenAI-compatible
	// server (e.g. vLLM), so this is the default served-model identifier.
	SLMDefaultModel = "jastorj/couchmind-v5.7.8.1_arctic_stage_3-cw-24K-16bit"
)

func init() {
	registerProvider(&openAIProvider{})
	registerProvider(&bedrockProvider{})
	registerProvider(&geminiProvider{})
	registerProvider(&slmProvider{})
}

// ---------------------------------------------------------------------------
// OpenAI provider
// ---------------------------------------------------------------------------

const (
	_OPENAI_CHAT_ENDPOINT        = "https://api.openai.com/v1/chat/completions"
	_OPENAI_MODERATIONS_ENDPOINT = "https://api.openai.com/v1/moderations"
)

// openAIProvider implements Provider for OpenAI's chat completions API.
//
// Registered once and shared by every concurrent request (see the registry in
// gateway.go), so it must stay effectively immutable: any field added here must
// be safe for concurrent use (guard it with a lock or a sync-safe type, as
// bedrockProvider does with its sync.Map client cache).
type openAIProvider struct{}

// OpenAI native request/response shapes.
type openAIChatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature"`
	Seed        int       `json:"seed,omitempty"`
	Stream      bool      `json:"stream"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
}

type openAIChatResponse struct {
	Error   map[string]interface{} `json:"error"`
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

type moderationResponse struct {
	Results []struct {
		Flagged bool `json:"flagged"`
	} `json:"results"`
}

func (*openAIProvider) ID() string { return ProviderOpenAI }

func (*openAIProvider) DefaultModel() string { return GPT4o_2024_05_13 }

func (*openAIProvider) ChatEndpoint(cfg *Config) string {
	if cfg.Endpoint != "" {
		return cfg.Endpoint
	}
	return _OPENAI_CHAT_ENDPOINT
}

func (*openAIProvider) BuildChatRequest(req *Request, cfg *Config) ([]byte, errors.Error) {
	return buildOpenAIBody(req)
}

func (*openAIProvider) ParseChatResponse(status int, body []byte) (*Response, errors.Error) {
	return parseOpenAIResponse(status, body)
}

// buildOpenAIBody renders the neutral Request into the OpenAI chat-completions
// body. It is shared by every provider that speaks the OpenAI wire format (the
// real OpenAI provider and self-hosted OpenAI-compatible servers via slm).
func buildOpenAIBody(req *Request) ([]byte, errors.Error) {
	msgs := make([]Message, 0, len(req.InitMessages)+len(req.Messages))
	msgs = append(msgs, req.InitMessages...)
	msgs = append(msgs, req.Messages...)

	reqBody := openAIChatRequest{
		Model:       req.Model,
		Messages:    msgs,
		Temperature: req.Temperature,
		Seed:        req.Seed,
		Stream:      false,
		// MaxTokens is sent only when the caller set it via natural_config; when zero it
		// is omitted (omitempty) so the provider imposes no engine-side cap on
		// the completion length.
		MaxTokens: req.MaxTokens,
	}

	payload, e := json.Marshal(reqBody)
	if e != nil {
		return nil, errors.NewNaturalLanguageRequestError(errors.E_NL_CHATCOMPLETIONS_PROMPT_MARSHAL, e)
	}
	return payload, nil
}

// parseOpenAIResponse turns the raw HTTP status and body of an OpenAI
// chat-completions response into the normalized Response. Shared by every
// provider that speaks the OpenAI wire format.
func parseOpenAIResponse(status int, body []byte) (*Response, errors.Error) {
	if status != http.StatusOK {
		return nil, errors.NewNaturalLanguageRequestError(errors.E_NL_CHATCOMPLETIONS_REQ_FAILED,
			status, llmErrCause(body))
	}

	var res openAIChatResponse
	if e := json.Unmarshal(body, &res); e != nil {
		return nil, errors.NewNaturalLanguageRequestError(errors.E_NL_CHATCOMPLETIONS_RESP_UNMARSHAL, e)
	}
	if res.Error != nil {
		return nil, errors.NewNaturalLanguageRequestError(errors.E_NL_ERR_CHATCOMPLETIONS_RESP, res.Error)
	}
	if len(res.Choices) == 0 {
		return nil, errors.NewNaturalLanguageRequestError(errors.E_NL_ERR_CHATCOMPLETIONS_RESP,
			fmt.Errorf("no message in response"))
	}

	return &Response{
		Content: res.Choices[0].Message.Content,
		Usage: TokenUsage{
			Prompt:     res.Usage.PromptTokens,
			Completion: res.Usage.CompletionTokens,
			Total:      res.Usage.TotalTokens,
		},
	}, nil
}

// Authorize applies the bearer key. An empty key (permitted on a custom
// endpoint) sends no Authorization header at all: servers treat a bare
// "Bearer " as malformed rather than anonymous.
func (*openAIProvider) Authorize(httpReq *http.Request, cfg *Config) errors.Error {
	if cfg.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	}
	return nil
}

// OpenAI carries no mandatory non-auth headers.
func (*openAIProvider) StaticHeaders() map[string]string { return nil }

// OpenAI requires an explicit bearer key; it cannot authenticate ambiently.
func (*openAIProvider) AllowsAmbientAuth() bool { return false }

// Moderate runs OpenAI's moderation endpoint over the user-role messages and
// fails the request if any content is flagged. All user turns are screened in
// a single call: the moderations API accepts an array input and returns one
// result per element, so a conversation history costs one round trip rather
// than one per message.
func (o *openAIProvider) Moderate(req *Request, cfg *Config, ctx Context) errors.Error {
	inputs := make([]string, 0, len(req.Messages))
	for _, m := range req.Messages {
		if m.Role == "user" && m.Content != "" {
			inputs = append(inputs, m.Content)
		}
	}
	if len(inputs) == 0 {
		return nil
	}

	// A custom chat endpoint we cannot derive a moderations URL from means the
	// caller is not on the standard OpenAI URL layout (e.g. a proxy). Skip
	// moderation rather than silently posting the content - and the caller's
	// key - to the default api.openai.com host.
	modEndpoint := o.moderationsEndpoint(cfg.Endpoint)
	if modEndpoint == "" {
		logging.Warnf("ai_gateway: skipping moderation: cannot derive a moderations URL from custom endpoint %q",
			cfg.Endpoint)
		return nil
	}

	payload, e := json.Marshal(map[string]interface{}{"input": inputs})
	if e != nil {
		return errors.NewNaturalLanguageRequestError(errors.E_NL_CHATCOMPLETIONS_PROMPT_MARSHAL, e)
	}

	// Bound the moderation round-trip (request + body read below) by the query
	// deadline, matching the completion path in DoChatCompletion.
	reqCtx, cancel := requestContext(ctx)
	defer cancel()

	resp, err := doProviderRequest(reqCtx, modEndpoint, payload, o, cfg, ctx)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		// Only a 404/405/501 from the derived /moderations URL proves the endpoint
		// has no moderation route (e.g. an OpenAI-compatible endpoint like AWS
		// Bedrock's compat surface); tell the caller to opt out via
		// "moderation":false. Any other status - transient 5xx, auth, rate limit -
		// is a real request failure and must keep its own error, so we never advise
		// disabling a safety check to work around a transient or partial problem.
		switch resp.StatusCode {
		case http.StatusNotFound, http.StatusMethodNotAllowed, http.StatusNotImplemented:
			return errors.NewNaturalLanguageRequestError(errors.E_NL_MODERATION_UNAVAILABLE,
				fmt.Sprintf("status %v: %v", resp.StatusCode, llmErrCause(body)))
		default:
			return errors.NewNaturalLanguageRequestError(errors.E_NL_CHATCOMPLETIONS_REQ_FAILED,
				resp.StatusCode, llmErrCause(body))
		}
	}

	var mr moderationResponse
	de := json.NewDecoder(resp.Body).Decode(&mr)
	resp.Body.Close()
	if de != nil {
		return errors.NewNaturalLanguageRequestError(errors.E_NL_CHATCOMPLETIONS_RESP_UNMARSHAL, de)
	}
	for _, r := range mr.Results {
		if r.Flagged {
			return errors.NewNaturalLanguageRequestError(errors.E_NL_MODERATION_FLAGGED)
		}
	}

	return nil
}

// moderationsEndpoint derives the moderations URL from the chat endpoint so a
// custom (e.g. proxied) chat endpoint moderates against the same host. Returns
// "" when a custom endpoint does not follow the standard ".../chat/completions"
// layout, in which case no moderations URL can be derived and the caller skips
// moderation instead of switching hosts.
func (*openAIProvider) moderationsEndpoint(chatEndpoint string) string {
	if chatEndpoint == "" {
		return _OPENAI_MODERATIONS_ENDPOINT
	}
	if strings.HasSuffix(chatEndpoint, "/chat/completions") {
		return strings.TrimSuffix(chatEndpoint, "/chat/completions") + "/moderations"
	}
	return ""
}

// ---------------------------------------------------------------------------
// AWS Bedrock provider
//
// Bedrock does not fit the shared HTTP path: auth is AWS SigV4, the request and
// response are typed SDK structs (not JSON bytes), and the regional endpoint is
// resolved by the SDK. So bedrockProvider implements the optional DirectCompleter
// interface and owns its whole round-trip via the Converse API; the orchestrator
// dispatches to Complete before the HTTP path, leaving the HTTP-shaped Provider
// methods on this type unused. The Converse mapping, retry policy and error
// classification mirror the Capella iQ implementation.
//
// Region failover needs no code here: the model is a geographic cross-region
// inference profile, so AWS routes inference across US regions transparently
// (CRIS). The caller hits a single source region and never sees the destinations.
// ---------------------------------------------------------------------------

const (
	ProviderBedrock = "bedrock"

	// Claude45InferenceProfileID is the geographic cross-region inference profile
	// for Claude Sonnet 4.5 (US). Same identifier Capella uses.
	Claude45InferenceProfileID = "us.anthropic.claude-sonnet-4-5-20250929-v1:0"

	// defaultBedrockRegion is the source region used when natural_config does not set one.
	defaultBedrockRegion = "us-east-1"
)

// bedrockConverseAPI is the subset of the bedrockruntime client used here. The
// real *bedrockruntime.Client satisfies it; defining it as an interface keeps
// the provider unit-testable with a mock (mirrors Capella's BedrockClientAPI).
type bedrockConverseAPI interface {
	Converse(ctx context.Context, params *bedrockruntime.ConverseInput,
		optFns ...func(*bedrockruntime.Options)) (*bedrockruntime.ConverseOutput, error)
}

// bedrockClient wraps the SDK client with a single same-endpoint retry on
// transient errors (mirrors Capella's bedrockClient).
type bedrockClient struct {
	client bedrockConverseAPI
}

func (c *bedrockClient) Converse(ctx context.Context,
	params *bedrockruntime.ConverseInput) (*bedrockruntime.ConverseOutput, error) {
	out, err := c.client.Converse(ctx, params)
	// Retry only while the deadline/cancellation has not already fired. Retrying
	// with a dead context would fail immediately with a context error and replace
	// the original (retriable) provider error with that uninformative one.
	if err != nil && isRetriableError(err) && ctx.Err() == nil {
		return c.client.Converse(ctx, params)
	}
	return out, err
}

// bedrockProvider implements Provider + DirectCompleter for AWS Bedrock.
type bedrockProvider struct {
	clients sync.Map // "region|endpoint" -> *bedrockClient
}

func (*bedrockProvider) ID() string { return ProviderBedrock }

func (*bedrockProvider) DefaultModel() string { return Claude45InferenceProfileID }

// The HTTP-path methods below are never invoked: the orchestrator routes Bedrock
// through Complete (DirectCompleter) before reaching the HTTP transport. They
// exist only to satisfy the Provider interface for registry/catalog purposes.
func (*bedrockProvider) ChatEndpoint(*Config) string { return "" }

func (*bedrockProvider) BuildChatRequest(*Request, *Config) ([]byte, errors.Error) {
	return nil, errors.NewNaturalLanguageRequestError(errors.E_NL_ERR_CHATCOMPLETIONS_RESP,
		fmt.Errorf("bedrock uses the Converse SDK path, not the HTTP transport"))
}

func (*bedrockProvider) ParseChatResponse(int, []byte) (*Response, errors.Error) {
	return nil, errors.NewNaturalLanguageRequestError(errors.E_NL_ERR_CHATCOMPLETIONS_RESP,
		fmt.Errorf("bedrock uses the Converse SDK path, not the HTTP transport"))
}

func (*bedrockProvider) Authorize(*http.Request, *Config) errors.Error { return nil }

func (*bedrockProvider) StaticHeaders() map[string]string { return nil }

func (*bedrockProvider) Moderate(*Request, *Config, Context) errors.Error { return nil }

// Bedrock can authenticate from the AWS default credential chain (env / shared
// config / IAM role), so a request may omit both cred_id and api_key.
func (*bedrockProvider) AllowsAmbientAuth() bool { return true }

// Complete owns the entire Bedrock round-trip: validate, map to ConverseInput,
// call Converse (with retry), and normalize the output. Mirrors Capella's
// validateBedrockMessages + getBedrockConverseInput + handleBedrockConverse.
func (p *bedrockProvider) Complete(ctx Context, req *Request, cfg *Config) (*Response, errors.Error) {
	if req.Model == "" {
		return nil, errors.NewNaturalLanguageRequestError(errors.E_NL_INVALID_NATURAL_CONFIG,
			"\"model\" is required for bedrock requests")
	}
	if len(req.Messages) == 0 {
		return nil, errors.NewNaturalLanguageRequestError(errors.E_NL_INVALID_NATURAL_CONFIG,
			"at least one message is required for bedrock requests")
	}

	input, err := buildBedrockConverseInput(req)
	if err != nil {
		return nil, err
	}

	exprCtx, ok := ctx.(expression.Context)
	if !ok {
		return nil, errors.NewNaturalLanguageRequestError(errors.E_NL_CRED_RESOLUTION_FAILED,
			fmt.Errorf("no execution context available"))
	}

	// When natural_config supplies a cred_id, AWS key material comes from the credstore;
	// otherwise the SDK's default credential chain (env / shared config / IAM
	// role) is used.
	var awsCred *expression.AWSCredential
	if cfg.CredId != "" {
		var ce error
		awsCred, ce = expression.ResolveAWSCredential(cfg.CredId, exprCtx)
		if ce != nil {
			return nil, errors.NewNaturalLanguageRequestError(errors.E_NL_CRED_RESOLUTION_FAILED, ce)
		}
	}

	// Resolve the effective region and endpoint. Precedence: the natural_config value,
	// then the credential's own field, then the built-in default region.
	region := cfg.Region
	if region == "" && awsCred != nil {
		region = awsCred.Region
	}
	if region == "" {
		region = defaultBedrockRegion
	}
	// The credstore credential's Endpoint is honored deliberately: a user who set
	// it is opting into routing Bedrock at that host (an explicit natural_config.endpoint
	// still wins). This is intentional, not an oversight - the cluster allowlist
	// gate below contains the blast radius, so the worst case of reusing a cred
	// whose Endpoint targets another service is a denied/failed request, not an
	// uncontrolled destination.
	endpoint := cfg.Endpoint
	if endpoint == "" && awsCred != nil {
		endpoint = awsCred.Endpoint
	}

	// Cluster allowlist gate: the effective Bedrock endpoint must be permitted
	// by the cluster allowlist, mirroring the HTTP path in doProviderRequest.
	// With no explicit endpoint the SDK targets the regional bedrock-runtime
	// host, so that is the URL we validate.
	//
	// This is a representative, source-region check, not an exact match of every
	// host actually contacted: the SDK may resolve a FIPS, dual-stack or non-
	// standard-partition host instead, and a geographic CRIS
	// inference profile fans the request out to other regions transparently.
	// Those variants and cross-region destinations are not individually gated;
	// the gate's purpose here is to govern the source-region endpoint.
	effectiveURL := endpoint
	if effectiveURL == "" {
		effectiveURL = fmt.Sprintf("https://bedrock-runtime.%s.amazonaws.com", region)
	}
	u, e := url.Parse(effectiveURL)
	if e != nil {
		return nil, errors.NewNaturalLanguageRequestError(errors.E_NL_DIRECT_CREATE_REQ, effectiveURL, e)
	}
	if ae := expression.IsUrlAllowedInCluster(u, exprCtx); ae != nil {
		return nil, errors.NewNaturalLanguageRequestError(errors.E_NL_URL_NOT_ALLOWED, ae.Error())
	}

	// Bound the round-trip by the request deadline so a stalled provider unblocks
	// the calling query thread (design doc: strict deadlines). Shares the same
	// derivation as the HTTP path via requestContext.
	stdctx, cancel := requestContext(ctx)
	defer cancel()

	client, ce := p.clientFor(stdctx, region, endpoint, awsCred)
	if ce != nil {
		return nil, errors.NewNaturalLanguageRequestError(errors.E_NL_CRED_RESOLUTION_FAILED, ce)
	}

	output, ce := client.Converse(stdctx, input)
	if ce != nil {
		return nil, mapBedrockError(ce)
	}

	resp := &Response{Content: extractBedrockResponseText(output)}
	if output.Usage != nil {
		if output.Usage.InputTokens != nil {
			resp.Usage.Prompt = int(*output.Usage.InputTokens)
		}
		if output.Usage.OutputTokens != nil {
			resp.Usage.Completion = int(*output.Usage.OutputTokens)
		}
		// Prefer the provider-reported total (authoritative, and accounts for
		// any cache tokens); fall back to the sum only if it is absent.
		if output.Usage.TotalTokens != nil {
			resp.Usage.Total = int(*output.Usage.TotalTokens)
		} else {
			resp.Usage.Total = resp.Usage.Prompt + resp.Usage.Completion
		}
	}
	return resp, nil
}

// clientFor returns a Bedrock client for the request.
//
// When awsCred is non-nil the client is built per request with static credstore
// credentials and is NOT cached, so credential rotation / expiry take effect
// immediately and distinct cred_ids never share a cached client. Otherwise it
// lazily builds and caches a client per (region, endpoint), backed by the AWS
// default credential chain, so connections are reused across requests.
func (p *bedrockProvider) clientFor(ctx context.Context, region, endpoint string,
	awsCred *expression.AWSCredential) (*bedrockClient, error) {
	if awsCred != nil {
		c, err := newBedrockClientStatic(ctx, region, endpoint, awsCred)
		if err != nil {
			return nil, err
		}
		return &bedrockClient{client: c}, nil
	}

	key := region + "|" + endpoint
	if c, ok := p.clients.Load(key); ok {
		return c.(*bedrockClient), nil
	}
	c, err := newBedrockClient(ctx, region, endpoint)
	if err != nil {
		return nil, err
	}
	bc := &bedrockClient{client: c}
	actual, _ := p.clients.LoadOrStore(key, bc)
	return actual.(*bedrockClient), nil
}

// newBedrockClient creates a bedrockruntime.Client for the given region, sourcing
// credentials from the AWS default chain (env / shared config / IAM role) and
// honouring an optional endpoint override. Mirrors Capella's newBedrockClient.
func newBedrockClient(ctx context.Context, region, endpointURL string) (*bedrockruntime.Client, error) {
	awscfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config for region %s: %w", region, err)
	}

	return bedrockClientFromConfig(awscfg, endpointURL), nil
}

// newBedrockClientStatic creates a bedrockruntime.Client using static AWS
// credentials resolved from the credstore, honouring an optional endpoint
// override. Used when natural_config supplies a cred_id, bypassing the AWS default
// credential chain so credentials never depend on ~/.aws or instance role.
func newBedrockClientStatic(ctx context.Context, region, endpointURL string,
	cred *expression.AWSCredential) (*bedrockruntime.Client, error) {
	awscfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion(region),
		awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				cred.AccessKeyID, cred.SecretAccessKey, cred.SessionToken)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config for region %s: %w", region, err)
	}

	return bedrockClientFromConfig(awscfg, endpointURL), nil
}

// bedrockClientFromConfig builds the runtime client from a resolved aws.Config,
// applying an optional endpoint override. Shared by the default-chain and
// credstore-backed construction paths.
func bedrockClientFromConfig(awscfg aws.Config, endpointURL string) *bedrockruntime.Client {
	var clientOpts []func(*bedrockruntime.Options)
	if endpointURL != "" {
		clientOpts = append(clientOpts, func(o *bedrockruntime.Options) {
			o.BaseEndpoint = aws.String(endpointURL)
		})
	}
	return bedrockruntime.NewFromConfig(awscfg, clientOpts...)
}

// buildBedrockConverseInput maps the neutral Request onto a ConverseInput.
// Mirrors Capella's getBedrockConverseInput: init messages and any system-role
// turns go to the System field (Converse rejects system entries in Messages),
// user/assistant turns become typed text content blocks, and MaxTokens /
// Temperature are applied only when set.
func buildBedrockConverseInput(req *Request) (*bedrockruntime.ConverseInput, errors.Error) {
	var systemPrompts []types.SystemContentBlock
	for _, m := range req.InitMessages {
		if m.Content == "" {
			continue
		}
		systemPrompts = append(systemPrompts, &types.SystemContentBlockMemberText{Value: m.Content})
	}

	var messages []types.Message
	for _, m := range req.Messages {
		role := types.ConversationRole(m.Role)
		switch {
		case role == types.ConversationRoleUser || role == types.ConversationRoleAssistant:
			messages = append(messages, types.Message{
				Role:    role,
				Content: []types.ContentBlock{&types.ContentBlockMemberText{Value: m.Content}},
			})
		case strings.EqualFold(m.Role, "system") && m.Content != "":
			// Bedrock Converse does not accept system-role entries in Messages;
			// promote them to System so embedded context is not silently dropped.
			systemPrompts = append(systemPrompts, &types.SystemContentBlockMemberText{Value: m.Content})
		}
	}

	if len(messages) == 0 {
		return nil, errors.NewNaturalLanguageRequestError(errors.E_NL_INVALID_NATURAL_CONFIG,
			"no valid conversation messages for bedrock request")
	}

	input := &bedrockruntime.ConverseInput{
		ModelId:  aws.String(req.Model),
		Messages: messages,
	}
	if len(systemPrompts) > 0 {
		input.System = systemPrompts
	}

	inferenceConfig := &types.InferenceConfiguration{}
	hasInferenceConfig := false

	if req.MaxTokens < 0 || req.MaxTokens > math.MaxInt32 {
		return nil, errors.NewNaturalLanguageRequestError(errors.E_NL_INVALID_NATURAL_CONFIG,
			fmt.Sprintf("max_tokens %d is out of range", req.MaxTokens))
	}
	if req.MaxTokens > 0 {
		mt := int32(req.MaxTokens)
		inferenceConfig.MaxTokens = &mt
		hasInferenceConfig = true
	}
	if req.Temperature > 0 {
		t := float32(req.Temperature)
		inferenceConfig.Temperature = &t
		hasInferenceConfig = true
	}
	if hasInferenceConfig {
		input.InferenceConfig = inferenceConfig
	}

	return input, nil
}

// extractBedrockResponseText pulls the assistant text from a ConverseOutput.
// Mirrors Capella's extractBedrockResponseText.
func extractBedrockResponseText(output *bedrockruntime.ConverseOutput) string {
	if output == nil || output.Output == nil {
		return ""
	}
	msg, ok := output.Output.(*types.ConverseOutputMemberMessage)
	if !ok {
		return ""
	}
	for _, content := range msg.Value.Content {
		if textContent, ok := content.(*types.ContentBlockMemberText); ok {
			return textContent.Value
		}
	}
	return ""
}

// isRetriableError reports whether a Bedrock error is a transient failure worth
// one retry. Mirrors Capella's isRetriableError.
func isRetriableError(err error) bool {
	if err == nil {
		return false
	}
	var apiErr smithy.APIError
	if stderrors.As(err, &apiErr) {
		switch apiErr.ErrorCode() {
		case "ThrottlingException", "ServiceUnavailableException",
			"InternalServerException", "ModelTimeoutException":
			return true
		}
	}
	return false
}

// mapBedrockError classifies an AWS Bedrock SDK error into a typed gateway error.
// Mirrors Capella's mapBedrockError categorisation, targeting E_NL_* codes.
func mapBedrockError(err error) errors.Error {
	var apiErr smithy.APIError
	if stderrors.As(err, &apiErr) {
		switch apiErr.ErrorCode() {
		case "ThrottlingException":
			return errors.NewNaturalLanguageRequestError(errors.E_NL_CHATCOMPLETIONS_REQ_FAILED,
				http.StatusTooManyRequests, fmt.Errorf("%s", apiErr.ErrorMessage()))
		case "AccessDeniedException":
			return errors.NewNaturalLanguageRequestError(errors.E_NL_CRED_RESOLUTION_FAILED,
				fmt.Errorf("%s", apiErr.ErrorMessage()))
		case "InvalidInputException", "ResourceNotFoundException", "ValidationException":
			return errors.NewNaturalLanguageRequestError(errors.E_NL_INVALID_NATURAL_CONFIG, apiErr.ErrorMessage())
		}
	}
	return errors.NewNaturalLanguageRequestError(errors.E_NL_ERR_CHATCOMPLETIONS_RESP, err)
}

// ---------------------------------------------------------------------------
// Google Gemini provider (public Generative Language API)
//
// geminiProvider speaks Google's Gemini generateContent protocol over the shared
// HTTP transport, hitting the public Generative Language API with an api-key
// query parameter. The codec below (buildGeminiBody / parseGeminiResponse and
// the native shapes) renders and parses that wire format.
// ---------------------------------------------------------------------------

// Gemini native request/response shapes.
type geminiPart struct {
	Text string `json:"text"`
}

type geminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []geminiPart `json:"parts"`
}

type geminiGenerationConfig struct {
	Temperature     *float64 `json:"temperature,omitempty"`
	MaxOutputTokens *int     `json:"maxOutputTokens,omitempty"`
}

type geminiRequest struct {
	Contents          []geminiContent         `json:"contents"`
	SystemInstruction *geminiContent          `json:"systemInstruction,omitempty"`
	GenerationConfig  *geminiGenerationConfig `json:"generationConfig,omitempty"`
}

type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []geminiPart `json:"parts"`
			Role  string       `json:"role"`
		} `json:"content"`
		FinishReason string `json:"finishReason"`
	} `json:"candidates"`
	UsageMetadata struct {
		PromptTokenCount     int `json:"promptTokenCount"`
		CandidatesTokenCount int `json:"candidatesTokenCount"`
		TotalTokenCount      int `json:"totalTokenCount"`
	} `json:"usageMetadata"`
	Error map[string]interface{} `json:"error"`
}

// buildGeminiBody renders the neutral Request into the Gemini generateContent
// body shared by both Google providers. InitMessages (and any system-role turns
// embedded in Messages) become the systemInstruction; user/assistant turns
// become contents with Gemini's role names ("model" for assistant). Temperature
// and MaxTokens are applied only when set.
func buildGeminiBody(req *Request) ([]byte, errors.Error) {
	var sysParts []geminiPart
	for _, m := range req.InitMessages {
		if m.Content == "" {
			continue
		}
		sysParts = append(sysParts, geminiPart{Text: m.Content})
	}

	var contents []geminiContent
	for _, m := range req.Messages {
		if strings.EqualFold(m.Role, "system") {
			// Gemini carries system context in systemInstruction, not contents;
			// promote system-role turns so embedded context is not dropped.
			if m.Content != "" {
				sysParts = append(sysParts, geminiPart{Text: m.Content})
			}
			continue
		}
		role := "user"
		if strings.EqualFold(m.Role, "assistant") || strings.EqualFold(m.Role, "model") {
			role = "model"
		}
		contents = append(contents, geminiContent{
			Role:  role,
			Parts: []geminiPart{{Text: m.Content}},
		})
	}

	body := geminiRequest{Contents: contents}
	if len(sysParts) > 0 {
		body.SystemInstruction = &geminiContent{Parts: sysParts}
	}

	// Temperature 0 is deliberately omitted so the model runs at its provider
	// default (1.0), matching Capella's Google path (temperature is only set
	// when non-zero there too). The Bedrock builder applies the same convention.
	gc := &geminiGenerationConfig{}
	hasGC := false
	if req.Temperature > 0 {
		t := req.Temperature
		gc.Temperature = &t
		hasGC = true
	}
	if req.MaxTokens > 0 {
		mt := req.MaxTokens
		gc.MaxOutputTokens = &mt
		hasGC = true
	}
	if hasGC {
		body.GenerationConfig = gc
	}

	payload, e := json.Marshal(body)
	if e != nil {
		return nil, errors.NewNaturalLanguageRequestError(errors.E_NL_CHATCOMPLETIONS_PROMPT_MARSHAL, e)
	}
	return payload, nil
}

// parseGeminiResponse turns the raw HTTP status and body of a Gemini
// generateContent response into the normalized Response, concatenating the
// candidate's text parts and mapping usageMetadata onto TokenUsage.
func parseGeminiResponse(status int, body []byte) (*Response, errors.Error) {
	if status != http.StatusOK {
		return nil, errors.NewNaturalLanguageRequestError(errors.E_NL_CHATCOMPLETIONS_REQ_FAILED,
			status, llmErrCause(body))
	}

	var res geminiResponse
	if e := json.Unmarshal(body, &res); e != nil {
		return nil, errors.NewNaturalLanguageRequestError(errors.E_NL_CHATCOMPLETIONS_RESP_UNMARSHAL, e)
	}
	if res.Error != nil {
		return nil, errors.NewNaturalLanguageRequestError(errors.E_NL_ERR_CHATCOMPLETIONS_RESP, res.Error)
	}
	if len(res.Candidates) == 0 || len(res.Candidates[0].Content.Parts) == 0 {
		return nil, errors.NewNaturalLanguageRequestError(errors.E_NL_ERR_CHATCOMPLETIONS_RESP,
			fmt.Errorf("no message in response"))
	}

	var sb strings.Builder
	for _, p := range res.Candidates[0].Content.Parts {
		sb.WriteString(p.Text)
	}

	return &Response{
		Content: sb.String(),
		Usage: TokenUsage{
			Prompt:     res.UsageMetadata.PromptTokenCount,
			Completion: res.UsageMetadata.CandidatesTokenCount,
			Total:      res.UsageMetadata.TotalTokenCount,
		},
	}, nil
}

// ---------------------------------------------------------------------------
// Gemini provider (public Generative Language API)
// ---------------------------------------------------------------------------

const _GEMINI_API_BASE = "https://generativelanguage.googleapis.com/v1beta/models/"

// geminiProvider implements Provider for the public Gemini generateContent API,
// authenticated with an api-key query parameter.
// Registered once and shared across concurrent requests: keep it immutable, and
// synchronize any field added later (see openAIProvider for the rationale).
type geminiProvider struct{}

func (*geminiProvider) ID() string { return ProviderGemini }

func (*geminiProvider) DefaultModel() string { return Gemini25Pro }

func (*geminiProvider) ChatEndpoint(cfg *Config) string {
	if cfg.Endpoint != "" {
		return cfg.Endpoint
	}
	return _GEMINI_API_BASE + cfg.Model + ":generateContent"
}

func (*geminiProvider) BuildChatRequest(req *Request, cfg *Config) ([]byte, errors.Error) {
	return buildGeminiBody(req)
}

func (*geminiProvider) ParseChatResponse(status int, body []byte) (*Response, errors.Error) {
	return parseGeminiResponse(status, body)
}

// Authorize sends the api key in the "x-goog-api-key" header, which the public
// Gemini API accepts as an alternative to the "?key=" query parameter. Keeping
// the key out of the URL ensures it can never leak through a transport-level
// *url.Error (which embeds the request URL) into an error cause or log. An empty
// key (permitted on a custom endpoint) sends no header.
func (*geminiProvider) Authorize(httpReq *http.Request, cfg *Config) errors.Error {
	if cfg.APIKey != "" {
		httpReq.Header.Set("x-goog-api-key", cfg.APIKey)
	}
	return nil
}

func (*geminiProvider) StaticHeaders() map[string]string { return nil }

// The public Gemini API requires an explicit api key; it cannot authenticate
// ambiently.
func (*geminiProvider) AllowsAmbientAuth() bool { return false }

func (*geminiProvider) Moderate(*Request, *Config, Context) errors.Error { return nil }

// ---------------------------------------------------------------------------
// SLM provider (self-hosted, OpenAI-compatible endpoint)
//
// slmProvider targets a self-hosted small language model served behind an
// OpenAI-compatible API (e.g. vLLM, TGI, Ollama's OpenAI shim). Because the wire
// format is identical to OpenAI's, it reuses the shared OpenAI codec
// (buildOpenAIBody / parseOpenAIResponse) verbatim; only the endpoint, auth and
// the absence of moderation differ. The endpoint is caller-supplied and
// mandatory - there is no built-in host - and moderation is a no-op since these
// servers do not implement OpenAI's /moderations API.
// ---------------------------------------------------------------------------

// Registered once and shared across concurrent requests: keep it immutable, and
// synchronize any field added later (see openAIProvider for the rationale).
type slmProvider struct{}

func (*slmProvider) ID() string { return ProviderSLM }

func (*slmProvider) DefaultModel() string { return SLMDefaultModel }

func (*slmProvider) ChatEndpoint(cfg *Config) string { return cfg.Endpoint }

func (*slmProvider) BuildChatRequest(req *Request, cfg *Config) ([]byte, errors.Error) {
	if cfg.Endpoint == "" {
		return nil, errors.NewNaturalLanguageRequestError(errors.E_NL_INVALID_NATURAL_CONFIG,
			"\"endpoint\" is required for slm requests")
	}
	return buildOpenAIBody(req)
}

func (*slmProvider) ParseChatResponse(status int, body []byte) (*Response, errors.Error) {
	return parseOpenAIResponse(status, body)
}

// Authorize applies a bearer token, matching the OpenAI-compatible servers'
// Authorization: Bearer <token> scheme. Self-hosted servers often run without
// auth; with no key configured, no Authorization header is sent.
func (*slmProvider) Authorize(httpReq *http.Request, cfg *Config) errors.Error {
	if cfg.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	}
	return nil
}

func (*slmProvider) StaticHeaders() map[string]string { return nil }

// slm cannot authenticate ambiently, but because its endpoint is always
// caller-supplied the config layer never requires a credential for it - a
// key is applied only when the caller provides one.
func (*slmProvider) AllowsAmbientAuth() bool { return false }

// Self-hosted OpenAI-compatible servers do not implement the /moderations API,
// so moderation is a no-op - it runs only for the real openai provider.
func (*slmProvider) Moderate(*Request, *Config, Context) errors.Error { return nil }
