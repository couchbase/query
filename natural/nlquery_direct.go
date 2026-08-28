//  Copyright 2024-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

// This file holds the direct ai_gateway natural-language path: the query engine
// talks to the LLM provider directly (no Capella iQ hop). All package-level
// symbols are direct-prefixed per the natural package naming convention; shared
// machinery lives in nlquery.go, and the Capella (iQ) path is in
// nlquery_capella.go. Routing between the two is decided by IsCapellaPath.

package natural

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/execution"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/natural/ai_gateway"
	"github.com/couchbase/query/parser/n1ql"
	"github.com/couchbase/query/primitives/couchbase"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

// slm provider prompt templates. The self-hosted (slm) provider uses these
// variant templates verbatim instead of the inline prompts built for the hosted
// providers. User/feedback templates use single-brace {tokens} filled by fillTemplate.

const slmSystemTmpl = "You are a Couchbase SQL++ query expert. Given a database schema and a natural language question, generate a syntactically valid SQL++ query that precisely answers the question.\n\nTask Instructions:\n" + _SQLPP_TASK_INSTRUCTIONS + "\nOutput Format:\nIn your answer, please enclose the generated SQL++ query in a code block:\n```\n-- Your SQL query\n```\n"

const slmUserTmpl = "{summary}Database Schema:\n{schema}\n\nThis schema describes the structure of the data in the specified bucket and scope. It includes information about the collections, fields, and their data types. Each top-level key in the schema is a fully-qualified `bucket`.`scope`.`collection` path; use those exact paths when referencing a collection. A field's nested object fields appear under \"properties\" and an array's element type appears under \"items\" -- a field can have both if it can be more than one shape.\n{knowledge}\nQuestion:\n{nl}\n"

const slmFeedbackTmpl = "The SQL++ query you generated:\n\n```\n{prevsqlpp}\n```\n\nreturned an error:\n\n<error>\n{error}\n</error>\n\nPlease analyze the error and provide a refined SQL++ query."

// fillTemplate replaces each {token} in tmpl with its value from vars, leaving
// tokens absent from vars untouched. It uses a Replacer rather than
// text/template because the templates embed SQL++ examples with literal braces.
func directFillTemplate(tmpl string, vars map[string]string) string {
	if len(vars) == 0 {
		return tmpl
	}
	oldnew := make([]string, 0, len(vars)*2)
	for k, v := range vars {
		oldnew = append(oldnew, k, v)
	}
	return strings.NewReplacer(oldnew...).Replace(tmpl)
}

// newDirectSLMSQLPrompt builds a SQL/FTS prompt for the slm provider from the
// variant templates. It supports any number of keyspaces: the schema object is
// keyed by each keyspace's fully-qualified `bucket`.`scope`.`collection` path, so
// the bucket/scope/collection relationship travels with the schema itself.
func newDirectSLMSQLPrompt(keyspaceInfo map[string]interface{}, paths []*algebra.Path,
	naturalPrompt, summary, hint string, forfts bool, provider, model string) (*prompt, errors.Error) {

	// The slm was trained on a schema object keyed by the fully-qualified
	// `bucket`.`scope`.`collection` path, with the field map under "properties".
	// Re-key the shared keyspaceInfo into that shape here; this is slm-specific, so
	// the shared structure keeps its {schema,fullpath} form for the hosted
	// providers. Each path component is backtick-quoted so a name that itself
	// contains dots (e.g. `my.bucket`) stays unambiguous.
	slmSchema := make(map[string]interface{}, len(paths))
	slmKnowledge := make(map[string]interface{})
	for _, p := range paths {
		parts := p.Parts()[1:] // drop the namespace component
		quoted := make([]string, len(parts))
		for i, part := range parts {
			quoted[i] = "`" + part + "`"
		}
		fqpath := strings.Join(quoted, ".")
		info, ok := keyspaceInfo[p.Keyspace()].(map[string]interface{})
		if !ok {
			return nil, errors.NewNaturalLanguageRequestError(errors.E_NL_PROMPT_SCHEMA_MARSHAL,
				fmt.Errorf("missing schema for keyspace %q", p.Keyspace()))
		}
		schemaEntry := map[string]interface{}{"properties": info["schema"]}
		if vi, ok := info["vectorIndexes"]; ok {
			schemaEntry["vectorIndexes"] = vi
		}
		slmSchema[fqpath] = schemaEntry
		if k, ok := info["knowledge"]; ok {
			slmKnowledge[fqpath] = k
		}
	}
	binKeyspacesInfo, err := json.Marshal(slmSchema)
	if err != nil {
		return nil, errors.NewNaturalLanguageRequestError(errors.E_NL_PROMPT_SCHEMA_MARSHAL, err)
	}

	// Only emit the summary section when a summary exists (e.g. after a paused
	// chat is summarized); otherwise the {summary} slot is dropped entirely so
	// no empty header is sent.
	summarySection := ""
	if summary != "" {
		summarySection = "Summary of the conversation so far:\n" + summary + "\n\n"
	}

	// Only emit the knowledge section when at least one keyspace has knowledge entries;
	// otherwise the {knowledge} slot is dropped entirely, same as {summary} above.
	knowledgeSection := ""
	if len(slmKnowledge) > 0 {
		binKnowledge, err := json.Marshal(slmKnowledge)
		if err != nil {
			return nil, errors.NewNaturalLanguageRequestError(errors.E_NL_PROMPT_SCHEMA_MARSHAL, err)
		}
		knowledgeSection = "\nKnowledge:\n" + string(binKnowledge) +
			"\n\nAdmin-supplied context, keyed by the same path as the schema. Apply it when relevant, and " +
			"prefer it over your own assumptions if it conflicts with them.\n"
	}

	userMsg := directFillTemplate(slmUserTmpl, map[string]string{
		"{summary}":   summarySection,
		"{schema}":    string(binKeyspacesInfo),
		"{knowledge}": knowledgeSection,
		"{nl}":        naturalPrompt,
	})
	if hint != "" {
		userMsg += "\n\nHint: \"" + hint + "\""
	}
	userMsg += vectorSearchInstructions(false)
	// TODO: _AMBIGUOUS_TERM_INSTRUCTION/_AMBIGUOUS_VECTOR_FIELD_INSTRUCTION rely on "#ERR:" to
	// ask the user a clarifying question, but the SLM path is single-shot with no way to surface
	// that back to the user.
	if forfts {
		userMsg += "\n\nAlways add the USE Clause in the query to use the FTS index, i.e. USE INDEX (USING FTS)."
	}

	rv := &prompt{
		InitMessages: []message{{Role: "system", Content: slmSystemTmpl}},
		Provider:     provider,
		CompletionSettings: completionSettings{
			Model:       model,
			Temperature: directGetTemperatureForModel(provider, model),
			Seed:        1,
		},
		Size: _INIT_SIZE,
	}
	rv.Messages = []message{{Role: "user", Content: userMsg}}
	rv.Size += len(slmSystemTmpl) + len(userMsg)
	return rv, nil
}

func newDirectSQLPrompt(keyspaceInfo map[string]interface{}, paths []*algebra.Path,
	naturalPrompt, summary, hint string, forfts bool,
	provider string, model string) (*prompt, errors.Error) {
	if provider == ai_gateway.ProviderSLM {
		return newDirectSLMSQLPrompt(keyspaceInfo, paths, naturalPrompt, summary, hint, forfts, provider, model)
	}
	rv := &prompt{
		InitMessages: []message{
			message{
				Role: "system",
				Content: "You are a Couchbase Server expert. Your task is to create valid queries to retrieve" +
					" or create data based on the provided Information." +
					"\n\nApproach this task step-by-step and take your time.",
			},
		},
		Provider: provider,
		CompletionSettings: completionSettings{
			Model:       model,
			Temperature: directGetTemperatureForModel(provider, model),
			Seed:        1,
		},
		Size: _INIT_SIZE,
	}

	if err := appendSQLUserMessage(rv, keyspaceInfo, naturalPrompt, summary, hint, forfts); err != nil {
		return nil, err
	}

	return rv, nil
}

func newDirectJSUDFPrompt(keyspaceInfo map[string]interface{}, naturalPrompt, summary, hint string, provider string, model string) (*prompt, errors.Error) {
	systemMsg := "You are a Couchbase Server expert. Your task is to write valid Javascript user defined functions" +
		" based on the provided information." +
		"\n\nApproach this task step-by-step and take your time."
	// The slm finetune was trained on the slm system template, so send it that
	// template as well, followed by the JSUDF task instructions. Both go in a
	// single system message because self-hosted chat templates commonly accept
	// only one system turn.
	if provider == ai_gateway.ProviderSLM {
		systemMsg = slmSystemTmpl + "\n\n" + systemMsg
	}
	rv := &prompt{
		InitMessages: []message{
			message{
				Role:    "system",
				Content: systemMsg,
			},
		},
		Provider: provider,
		CompletionSettings: completionSettings{
			Model:       model,
			Temperature: directGetTemperatureForModel(provider, model),
			Seed:        1,
		},
		Size: _INIT_SIZE,
	}
	if provider == ai_gateway.ProviderSLM {
		rv.Size += len(slmSystemTmpl)
	}

	if err := appendJSUDFUserMessage(rv, keyspaceInfo, naturalPrompt, summary, hint); err != nil {
		return nil, err
	}

	return rv, nil
}

func newDirectSQLIterativePrompt(chat *prompt, naturalPrompt string, hint string, forfts bool, provider, model string) *prompt {
	if provider != "" {
		chat.Provider = provider
	}
	if model != "" {
		chat.CompletionSettings.Model = model
		chat.CompletionSettings.Temperature = directGetTemperatureForModel(provider, model)
	}

	return appendSQLIterativeUserMessage(chat, naturalPrompt, hint, forfts)
}

func newDirectJSUDFIterativePrompt(chat *prompt, naturalPrompt string, hint string, provider, model string) *prompt {
	if provider != "" {
		chat.Provider = provider
	}
	if model != "" {
		chat.CompletionSettings.Model = model
		chat.CompletionSettings.Temperature = directGetTemperatureForModel(provider, model)
	}
	chat = appendJSUDFIterativeUserMessage(chat, naturalPrompt, hint)
	return chat
}

func directGetTemperatureForModel(provider, model string) float64 {
	switch provider {
	case ai_gateway.ProviderOpenAI:
		// Model identifiers are passed to the provider verbatim, so compare
		// case-insensitively here.
		if strings.HasPrefix(strings.ToLower(model), "gpt-5") {
			return 1
		}
		return 0
	default:
		return 0
	}
}

func ProcessDirectRequest(cfg *NaturalConfig, nlquery, nlHint string, elems []*algebra.Path, nloutputOpt naturalOutput,
	explain, advise, useKnowledge bool,
	context NaturalContext, record func(execution.Phases, time.Duration),
	tokens *LLMTokenUsage) (string, algebra.Statement, errors.Error) {

	waitTime := util.Now()
	err := throttleRequest()
	record(execution.NLWAIT, util.Since(waitTime))
	if err != nil {
		return "", nil, err
	}

	if err = cfg.ResolveProviderAndModel(); err != nil {
		return "", nil, err
	}
	provider, model := cfg.Provider, cfg.Model

	keyspaceInfo := make(map[string]interface{}, len(elems))
	var samples map[string]map[string]*sampleField
	inferschema := util.Now()
	keyspaceInfo, samples, err = keyspacesInfoForPrompt(keyspaceInfo, elems, context, provider == ai_gateway.ProviderSLM,
		useKnowledge, nlquery)
	record(execution.INFERSCHEMA, util.Since(inferschema))
	if err != nil {
		return "", nil, err
	}

	var prompt *prompt
	switch nloutputOpt {
	case SQL:
		prompt, err = newDirectSQLPrompt(keyspaceInfo, elems, nlquery, "", nlHint, false, provider, model)
	case JSUDF:
		prompt, err = newDirectJSUDFPrompt(keyspaceInfo, nlquery, "", nlHint, provider, model)
	case FTSSQL:
		prompt, err = newDirectSQLPrompt(keyspaceInfo, elems, nlquery, "", nlHint, true, provider, model)

	default:
		err = errors.NewServiceErrorUnrecognizedValue("natural_output", nloutputOpt.String())
	}
	if err != nil {
		return "", nil, err
	}
	prompt.samples = samples

	chatcompletionreq := util.Now()
	content, err := doDirectChatCompletion(prompt, cfg, context, tokens)
	record(execution.CHATCOMPLETIONSREQ, util.Since(chatcompletionreq))
	if err != nil {
		return "", nil, err
	}
	if err := CheckAndReturnErrorResponse(content); err != nil {
		return "", nil, errors.NewNaturalLanguageRequestError(errors.E_NL_ERR_CHATCOMPLETIONS_RESP, err)
	}

	parse := util.Now()
	stmt, err := getStatement(content, nloutputOpt)
	if err != nil {
		return "", nil, err
	}

	if advise || explain {
		prefix := "advise "
		if explain {
			prefix = "explain "
		}
		stmt = prefix + stmt
	}

	var parseErr error
	var nlAlgebraStmt algebra.Statement
	nlAlgebraStmt, parseErr = n1ql.ParseStatement2(stmt, "default", "")
	record(execution.NLPARSE, util.Since(parse))
	if parseErr == nil {
		parseErr = validateGeneratedStatement(nlAlgebraStmt, context)
	}
	if parseErr != nil {
		retrytime := util.Now()
		prompt = directBuildRetryPrompt(prompt, content, parseErr, true)
		var retryErr error
		for i := 0; i < maxCorrectionRetries; i++ {
			var fatalErr errors.Error
			content, stmt, nlAlgebraStmt, fatalErr, retryErr = directRetryRequest(cfg, prompt, context, record, nloutputOpt, explain, advise, tokens)
			if fatalErr != nil {
				// Request-level failure (rate limit, gateway/transport error, model
				// refusal): not a correctable statement, so surface it immediately
				// instead of feeding it back as correction feedback and re-sending.
				return "", nil, fatalErr
			}
			if retryErr == nil {
				record(execution.NLRETRY, util.Since(retrytime))
				return stmt, nlAlgebraStmt, nil
			}
			// Only build the next correction prompt if another attempt follows;
			// on the final iteration the prompt would be discarded.
			if i < maxCorrectionRetries-1 {
				prompt = directBuildRetryPrompt(prompt, content, retryErr, false)
			}
		}
		return "", nil, errors.NewNaturalLanguageRequestError(errors.E_NL_FAIL_GENERATED_STMT,
			maxCorrectionRetries, content, retryErr)
	}

	return stmt, nlAlgebraStmt, nil
}

// includeInstructions folds _SQLPP_TASK_INSTRUCTIONS into the non-slm feedback
// message (the slm feedback template never needs it, since slmSystemTmpl already
// carries it in the system turn). Callers pass true only for the first correction
// round of a request: the rules then stay in the message history for every
// subsequent round without needing to be resent.
func directBuildRetryPrompt(pmt *prompt, assistantContent string, retryErr error, includeInstructions bool) *prompt {
	var feedback string
	if pmt.Provider == ai_gateway.ProviderSLM {
		feedback = directFillTemplate(slmFeedbackTmpl, map[string]string{
			"{prevsqlpp}": assistantContent,
			"{error}":     retryErr.Error(),
		})
	} else {
		feedback = correctionFeedback(retryErr, includeInstructions)
	}
	return buildRetryPrompt(pmt, assistantContent, feedback)
}

// retryRequest runs one correction round: it re-sends the prompt, extracts the
// statement and parses it. It separates the two failure modes so the caller can
// react correctly:
//   - fatalErr is a request-level failure (rate-limit/throttle, a gateway or
//     transport error from the completion call, or a model refusal via #ERR). It
//     is not correctable by feeding it back to the model, so the caller must
//     surface it immediately.
//   - parseErr is a correctable failure: the model produced a statement that did
//     not parse, so the caller can append it as feedback and retry.
//
// At most one of fatalErr / parseErr is non-nil.
func directRetryRequest(cfg *NaturalConfig, prompt *prompt, context NaturalContext,
	record func(execution.Phases, time.Duration), nloutputOpt naturalOutput,
	explain, advise bool, tokens *LLMTokenUsage) (content, stmt string,
	nlAlgebraStmt algebra.Statement, fatalErr errors.Error, parseErr error) {

	waitTime := util.Now()
	if err := throttleRequest(); err != nil {
		record(execution.NLWAIT, util.Since(waitTime))
		return "", "", nil, err, nil
	}
	record(execution.NLWAIT, util.Since(waitTime))

	chatcompletionreq := util.Now()
	content, cerr := doDirectChatCompletion(prompt, cfg, context, tokens)
	record(execution.CHATCOMPLETIONSREQ, util.Since(chatcompletionreq))
	if cerr != nil {
		return "", "", nil, cerr, nil
	}
	if err := CheckAndReturnErrorResponse(content); err != nil {
		return content, "", nil, errors.NewNaturalLanguageRequestError(errors.E_NL_ERR_CHATCOMPLETIONS_RESP, err), nil
	}

	parse := util.Now()
	stmt, serr := getStatement(content, nloutputOpt)
	if serr != nil {
		return content, "", nil, nil, serr
	}

	if advise || explain {
		prefix := "advise "
		if explain {
			prefix = "explain "
		}
		stmt = prefix + stmt
	}

	nlAlgebraStmt, parseErr = n1ql.ParseStatement2(stmt, "default", "")
	record(execution.NLPARSE, util.Since(parse))
	if parseErr == nil {
		parseErr = validateGeneratedStatement(nlAlgebraStmt, context)
	}

	return content, stmt, nlAlgebraStmt, nil, parseErr
}

func ProcessDirectConversationalRequest(cfg *NaturalConfig, nlquery, nlHint string, chatId string,
	nloutputOpt naturalOutput, explain, advise, useKnowledge bool,
	users []string,
	context NaturalContext, record func(execution.Phases, time.Duration),
	tokens, chatTokens *LLMTokenUsage) (string, algebra.Statement, errors.Error) {

	waitTime := util.Now()
	err := throttleRequest()
	record(execution.NLWAIT, util.Since(waitTime))
	if err != nil {
		return "", nil, err
	}

	var ce *ChatEntry
	rv := naturalchatHistory.Get(chatId, nil)
	if rv != nil {
		ce = rv.(*ChatEntry)
	} else {
		return "", nil, errors.NewNaturalLanguageRequestError(errors.E_NL_NO_SUCH_CHAT, chatId)
	}

	ce.Lock()
	defer ce.Unlock()
	// Fold this turn's token usage into the conversation's running total (under the
	// entry lock, before it unlocks). "tokens" keeps this turn's own usage
	// (surfaced as requestTokens); "chatTokens" receives the cumulative
	// conversation total. Deferred so tokens already spent are still counted if the
	// turn later errors. "tokens" is a required out-param (every caller supplies
	// one), so it is dereferenced unconditionally; "chatTokens" is optional.
	defer func() {
		ce.Tokens.Add(*tokens)
		if chatTokens != nil {
			*chatTokens = ce.Tokens
		}
	}()
	if ce.Removed {
		return "", nil, errors.NewNaturalLanguageRequestError(errors.E_NL_CHAT_FAIL,
			fmt.Sprintf("conversation with \"natural_chatid\":%s was deleted", chatId))
	}
	if ce.Paused {
		return "", nil, errors.NewNaturalLanguageRequestError(errors.E_NL_CHAT_FAIL,
			fmt.Sprintf("conversation with \"natural_chatid\":%s was paused", chatId))
	}

	if err := ce.CheckUser(users); err != nil {
		return "", nil, err
	}

	if err = cfg.ResolveProviderAndModel(); err != nil {
		return "", nil, err
	}
	provider, model := cfg.Provider, cfg.Model

	var prompt *prompt
	if ce.prompt == nil {
		keyspaceInfo := make(map[string]interface{}, len(ce.Keyspaces))
		var samples map[string]map[string]*sampleField
		inferschema := util.Now()
		keyspaceInfo, samples, err = keyspacesInfoForPrompt(keyspaceInfo, ce.Keyspaces, context, provider == ai_gateway.ProviderSLM,
			useKnowledge, nlquery)
		record(execution.INFERSCHEMA, util.Since(inferschema))
		if err != nil {
			return "", nil, err
		}
		ce.samples = samples

		switch nloutputOpt {
		case SQL:
			prompt, err = newDirectSQLPrompt(keyspaceInfo, ce.Keyspaces, nlquery, ce.Summary, nlHint, false, provider, model)
		case JSUDF:
			prompt, err = newDirectJSUDFPrompt(keyspaceInfo, nlquery, ce.Summary, nlHint, provider, model)
		case FTSSQL:
			prompt, err = newDirectSQLPrompt(keyspaceInfo, ce.Keyspaces, nlquery, ce.Summary, nlHint, true, provider, model)
		default:
			err = errors.NewServiceErrorUnrecognizedValue("natural_output", nloutputOpt.String())
		}
		ce.Summary = ""
		if err != nil {
			return "", nil, err
		}
		prompt.samples = ce.samples
	} else {
		// The provider may have changed since the cached prompt (and ce.samples)
		// were built -- including a resume under a different provider, since
		// ce.prompt.Provider is persisted. Keep sample values confined to slm:
		// drop the cache when off slm, and lazily (re)collect it when on slm but
		// the cache is empty. ce.samples is read only here (copied onto the
		// transient prompt for injection in doChatCompletion) and never enters
		// the persisted conversation.
		if provider != ai_gateway.ProviderSLM {
			ce.samples = nil
		} else if ce.samples == nil {
			tmp := make(map[string]interface{}, len(ce.Keyspaces))
			inferschema := util.Now()
			_, samples, serr := keyspacesInfoForPrompt(tmp, ce.Keyspaces, context, true, false, nlquery)
			record(execution.INFERSCHEMA, util.Since(inferschema))
			if serr != nil {
				return "", nil, serr
			}
			ce.samples = samples
		}

		switch nloutputOpt {
		case SQL:
			prompt = newDirectSQLIterativePrompt(ce.prompt, nlquery, nlHint, false, provider, model)
		case JSUDF:
			prompt = newDirectJSUDFIterativePrompt(ce.prompt, nlquery, nlHint, provider, model)
		case FTSSQL:
			prompt = newDirectSQLIterativePrompt(ce.prompt, nlquery, nlHint, true, provider, model)
		default:
			err = errors.NewServiceErrorUnrecognizedValue("natural_output", nloutputOpt.String())
		}
		if err != nil {
			return "", nil, err
		}
		prompt.samples = ce.samples
	}

	if prompt.Size >= _MAX_PROMPT_SIZE {
		return "", nil, errors.NewNaturalLanguageRequestError(errors.E_NL_CHAT_PROMPT_TOO_LARGE,
			logging.HumanReadableSize(int64(prompt.Size), false), logging.HumanReadableSize(_MAX_PROMPT_SIZE, false))
	}

	chatcompletionreq := util.Now()
	content, err := doDirectChatCompletion(prompt, cfg, context, tokens)
	record(execution.CHATCOMPLETIONSREQ, util.Since(chatcompletionreq))
	if err != nil {
		return "", nil, err
	}
	if err := CheckAndReturnErrorResponse(content); err != nil {
		completeConversationPromptLocked(content, ce, prompt)
		return "", nil, errors.NewNaturalLanguageRequestError(errors.E_NL_ERR_CHATCOMPLETIONS_RESP, err)
	}

	parse := util.Now()
	stmt, err := getStatement(content, nloutputOpt)
	if err != nil {
		return "", nil, err
	}

	if advise || explain {
		prefix := "advise "
		if explain {
			prefix = "explain "
		}
		stmt = prefix + stmt
	}

	var parseErr error
	var nlAlgebraStmt algebra.Statement
	nlAlgebraStmt, parseErr = n1ql.ParseStatement2(stmt, "default", "")
	record(execution.NLPARSE, util.Since(parse))
	if parseErr == nil {
		parseErr = validateGeneratedStatement(nlAlgebraStmt, context)
	}
	if parseErr != nil {
		retrytime := util.Now()
		prompt = directBuildRetryPrompt(prompt, content, parseErr, true)
		var retryErr error
		for i := 0; i < maxCorrectionRetries; i++ {
			var fatalErr errors.Error
			content, stmt, nlAlgebraStmt, fatalErr, retryErr = directRetryRequest(cfg, prompt, context, record, nloutputOpt, explain, advise, tokens)
			if fatalErr != nil {
				// Request-level failure (rate limit, gateway/transport error, model
				// refusal): not a correctable statement, so surface it immediately
				// instead of feeding it back as correction feedback and re-sending.
				return "", nil, fatalErr
			}
			if retryErr == nil {
				completeConversationPromptLocked(content, ce, prompt)
				record(execution.NLRETRY, util.Since(retrytime))
				return stmt, nlAlgebraStmt, nil
			}
			if i < maxCorrectionRetries-1 {
				prompt = directBuildRetryPrompt(prompt, content, retryErr, false)
			}
		}
		completeConversationPromptLocked(content, ce, prompt)
		return "", nil, errors.NewNaturalLanguageRequestError(errors.E_NL_FAIL_GENERATED_STMT,
			maxCorrectionRetries, content, retryErr)
	}

	completeConversationPromptLocked(content, ce, prompt)
	return stmt, nlAlgebraStmt, err
}

// caller should have already acquired lock on ce
func ProcessDirectPauseChat(chatId, requestId string,
	datastorecreds []string,
	summarize value.Tristate, cfg *NaturalConfig,
	context NaturalContext, record func(execution.Phases, time.Duration),
	reqTokens, chatTokens *LLMTokenUsage) errors.Error {
	if chatId == "" {
		return errors.NewNaturalLanguageRequestError(errors.E_NL_MISSING_CHAT_ID)
	}

	rv := GetConversation(chatId)
	if rv == nil {
		return errors.NewNaturalLanguageRequestError(errors.E_NL_NO_SUCH_CHAT, chatId)
	}
	ce, ok := rv.(*ChatEntry)
	if !ok {
		return errors.NewNaturalLanguageRequestError(errors.E_NL_CHAT_FAIL, "failed to cast cache entry")
	}
	ce.Lock()
	defer ce.Unlock()
	if err := ce.CheckUser(datastorecreds); err != nil {
		return err
	}

	shouldSummarize := ce.prompt != nil &&
		(summarize == value.TRUE ||
			(summarize == value.NONE &&
				(ce.prompt.Size >= summarizeThreshold || len(ce.prompt.Messages) >= summarizeMessageLen)))
	if shouldSummarize {
		if cfg == nil {
			return errors.NewNaturalLanguageRequestError(errors.E_NL_MISSING_NL_PARAM, "\"natural_config\"")
		}
		if err := cfg.ResolveProviderAndModel(); err != nil {
			return errors.NewNaturalLanguageRequestError(errors.E_NL_CHAT_SUMMARIZE_FAILED, "failed to resolve provider and model", err)
		}

		// Fold the summarization completion's tokens into the conversation total so
		// they are persisted with the chat document and reflected in the running
		// total surfaced on the pause response below.
		var sumTokens LLMTokenUsage
		if err := directSummarizePrompt(ce, cfg, context, record, &sumTokens); err != nil {
			return errors.NewNaturalLanguageRequestError(errors.E_NL_CHAT_SUMMARIZE_FAILED, err)
		}
		ce.Tokens.Add(sumTokens)
		// The summarization completion is this pause request's own LLM cost, surfaced
		// as requestTokens. Left zero (and suppressed) when no summarization ran.
		if reqTokens != nil {
			*reqTokens = sumTokens
		}
	}

	// Surface the conversation's running token total (including any summarization
	// tokens folded in above) as the chat total on the pause response. Read under
	// the entry lock.
	if chatTokens != nil {
		*chatTokens = ce.Tokens
	}

	hasquerymetadata, err := hasQueryMetadataForNLChat(true, requestId, "Natural Language chat PAUSE", true)
	if err != nil {
		return errors.NewNaturalLanguageRequestError(errors.E_NL_CHAT_PAUSE_FAILED,
			fmt.Sprintf("failed to get query metadata: %v", err))
	} else if !hasquerymetadata {
		return errors.NewMissingQueryMetadataError("PAUSE CHAT")
	}

	store := datastore.GetDatastore()
	if store == nil {
		err := errors.NewNoDatastoreError()
		return errors.NewNaturalLanguageRequestError(errors.E_NL_CHAT_PAUSE_FAILED, "failed to get datastore", err)
	}

	queryMetadata, err := store.GetQueryMetadata()
	if err != nil {
		return errors.NewNaturalLanguageRequestError(errors.E_NL_CHAT_PAUSE_FAILED, "failed to get query metadata: %v", err)
	}

	dpairs := make([]value.Pair, 1)
	queryContext := datastore.GetDurableQueryContextFor(queryMetadata)

	marshalledchat, merr := ce.MarshalJSON()
	if merr != nil {
		return errors.NewNaturalLanguageRequestError(errors.E_NL_CHAT_PAUSE_FAILED, "failed to marshal chat entry", merr)
	}
	key := fmt.Sprintf("%s%s", CHAT_DOC_PREFIX, chatId)
	dpairs[0].Name = key
	dpairs[0].Value = value.NewValue(map[string]interface{}{"chat": base64.StdEncoding.EncodeToString(marshalledchat)})
	ttltime := time.Now().Add(CHAT_DOC_TTL_DURATION)
	opt := value.NewValue(map[string]interface{}{})
	opt.SetField("expiration", ttltime.Unix())
	dpairs[0].Options = opt
	insertInterval := interval
	for i := 0; i < maxRetry; i++ {
		_, _, errs := queryMetadata.Insert(dpairs, queryContext, false)
		if len(errs) > 0 {
			if couchbase.CanRetryWithRefresh(errs[0]) {
				time.Sleep(insertInterval)
				insertInterval *= 2
			} else {
				logging.Errorf("%s Error inserting into QUERY_METADATA bucket: %v (key %s)", _CHAT_LOG_PREFIX, errs, key)
				return errors.NewNaturalLanguageRequestError(errors.E_NL_CHAT_PAUSE_FAILED,
					fmt.Sprintf("err inserting the chat document: %v", errs))
			}
		} else {
			break
		}
	}
	ce.stopInactivityTimer()
	DeleteConversation(chatId)
	ce.Paused = true
	logging.Infof("%s Chat with id %s paused", _CHAT_LOG_PREFIX, chatId)
	return nil
}

func directSummarizePrompt(ce *ChatEntry, cfg *NaturalConfig, context NaturalContext,
	record func(execution.Phases, time.Duration), tokens *LLMTokenUsage) errors.Error {
	if ce.prompt == nil || len(ce.prompt.Messages) <= 1 {
		return nil
	}
	provider, model := cfg.Provider, cfg.Model

	var promptBuf strings.Builder
	promptBuf.WriteString("The following is a conversation history between a user and an assistant. " +
		"The conversation history is being summarized to save space but important information might be lost in the process. " +
		"Summarize the conversation while keeping important details that can be useful for the continuation of the conversation. " +
		"Preserve all important details related to the assistant's sql++ suggestions :" +
		"Fields used in SELECT, WHERE, JOIN, GROUP BY, and ORDER BY clauses\n" +
		"Any predicates, filters, conditions, and their values\n" +
		"Join relationships, including keys and join types\n" +
		"Aggregations, functions, and computed expressions\n" +
		"Relevant bucket, scope, and collection names\n" +
		"Capture the user's intent and any constraints or preferences expressed\n" +
		"Retain important assumptions or clarifications made by the assistant\n" +
		"Trim redundant information\n\n")
	for _, msg := range ce.prompt.Messages {
		promptBuf.WriteString(fmt.Sprintf("%s: %s\n", msg.Role, msg.Content))
	}
	promptBuf.WriteString("Summarize the above conversation history as precisely as possible.\n\n")

	pmt := &prompt{
		InitMessages: []message{
			message{
				Role:    "system",
				Content: "You are a helpful assistant for summarizing conversation history.",
			},
		},
		Messages: []message{
			message{
				Role:    "user",
				Content: promptBuf.String(),
			},
		},
		Provider: provider,
		CompletionSettings: completionSettings{
			Model:       model,
			Temperature: directGetTemperatureForModel(provider, model),
			Seed:        1,
		},
		Size: len(promptBuf.String()),
	}

	chatcompletions := util.Now()
	content, err := doDirectChatCompletion(pmt, cfg, context, tokens)
	record(execution.CHATCOMPLETIONSREQ, util.Since(chatcompletions))
	if err != nil {
		return errors.NewNaturalLanguageRequestError(errors.E_NL_CHAT_SUMMARIZE_FAILED, "chat completions request failed", err)
	}
	ce.Summary = content
	ce.prompt = nil
	return nil
}

func directSLMSamplesBlock(samples map[string]map[string]*sampleField) string {
	if len(samples) == 0 {
		return ""
	}
	b, err := json.Marshal(samples)
	if err != nil {
		return ""
	}
	return "Representative sample values per field, grouped by keyspace, nested to mirror " +
		"the schema: \"properties\" holds an object's fields, \"items\" holds an array's " +
		"element type, and a field can have both if it can be more than one shape. " +
		"\"samples\" is grouped by shape (e.g. \"string\", \"number\") when a field can be " +
		"more than one type. Use them to choose correct literal values; they are examples, " +
		"not the complete set of values:\n" + string(b)
}

// doChatCompletion maps the natural prompt onto the gateway's neutral request
// schema, dispatches it, and returns the completion text. When acc is non-nil
// the gateway response's token usage is added to it, so a caller can accumulate
// the total across the (possibly multiple) completions a single request makes.
func doDirectChatCompletion(p *prompt, cfg *NaturalConfig, context NaturalContext, acc *LLMTokenUsage) (string, errors.Error) {
	req := p.toDirectGatewayRequest(cfg)
	// Inject cached sample values for the slm provider only. req.Messages is a
	// fresh copy from toGatewayRequest, so appending the block to the final user
	// turn's content mutates only this transient request -- p.Messages and the
	// persisted conversation are untouched. This keeps samples adjacent to the
	// schema/question the model reads, while raw sample values stay independent of
	// stored history and never reach a non-slm provider.
	if p.Provider == ai_gateway.ProviderSLM && len(p.samples) > 0 {
		if block := directSLMSamplesBlock(p.samples); block != "" {
			if n := len(req.Messages); n > 0 && req.Messages[n-1].Role == "user" {
				req.Messages[n-1].Content += "\n\n" + block
			} else {
				req.Messages = append(req.Messages, ai_gateway.Message{Role: "user", Content: block})
			}
		}
	}
	resp, err := ai_gateway.DoChatCompletion(req, cfg, context)
	if err != nil {
		return "", err
	}
	if acc != nil {
		acc.Prompt += resp.Usage.Prompt
		acc.Completion += resp.Usage.Completion
		acc.Total += resp.Usage.Total
	}
	return resp.Content, nil
}

// toGatewayRequest renders the natural prompt into the gateway's neutral
// request. The output token cap comes from natural_config (cfg.OutputTokenLimit); when
// unset it is zero and the provider imposes no engine-side cap.
func (p *prompt) toDirectGatewayRequest(cfg *NaturalConfig) *ai_gateway.Request {
	return &ai_gateway.Request{
		Model:        p.CompletionSettings.Model,
		InitMessages: directToGatewayMessages(p.InitMessages),
		Messages:     directToGatewayMessages(p.Messages),
		Temperature:  p.CompletionSettings.Temperature,
		Seed:         p.CompletionSettings.Seed,
		MaxTokens:    cfg.OutputTokenLimit,
	}
}

func directToGatewayMessages(msgs []message) []ai_gateway.Message {
	if msgs == nil {
		return nil
	}
	out := make([]ai_gateway.Message, len(msgs))
	for i, m := range msgs {
		out[i] = ai_gateway.Message{Role: m.Role, Content: m.Content}
	}
	return out
}
