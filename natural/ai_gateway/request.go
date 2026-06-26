//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package ai_gateway

// request.go defines the neutral, vendor-agnostic request/response schema the
// gateway exposes to the query layer. Callers (the natural language path today,
// AI functions later) populate these types once; each provider's
// BuildChatRequest/ParseChatResponse translates to and from its vendor-native
// shape. Nothing here is provider-specific.

// Message is the vendor-agnostic unit of a chat conversation. Role is one of
// "system", "user" or "assistant"; each provider maps the role to its own
// placement rules (e.g. inline for OpenAI, a dedicated system field elsewhere).
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Request is the provider-agnostic chat-completion request. InitMessages
// (system/preamble) are kept separate from Messages so providers that carry the
// system prompt in a dedicated field can route it correctly rather than guess
// which leading turns are "system".
type Request struct {
	// Model is the resolved model identifier.
	Model string

	// InitMessages carry the system/preamble turns.
	InitMessages []Message

	// Messages carry the conversation turns.
	Messages []Message

	// Temperature is the sampling temperature.
	Temperature float64

	// Seed requests deterministic sampling. Only some vendors honour it;
	// providers that do not simply omit it.
	Seed int

	// MaxTokens caps the completion length. Zero means "omit" so the provider
	// imposes no engine-side cap (vendors that require the field substitute
	// their own default).
	MaxTokens int
}

// TokenUsage is the normalized token accounting extracted from a provider
// response. Vendors report these counts under different names; the gateway maps
// them into this single shape for cost observability.
type TokenUsage struct {
	Prompt     int
	Completion int
	Total      int
}

// Add accumulates another usage into this one, so a caller can total the token
// cost across the several completions a single request or conversation makes.
func (t *TokenUsage) Add(o TokenUsage) {
	t.Prompt += o.Prompt
	t.Completion += o.Completion
	t.Total += o.Total
}

// Response is the normalized completion returned to the caller: the essential
// text output and token usage, with vendor-proprietary metadata stripped.
type Response struct {
	Content string
	Usage   TokenUsage
}
