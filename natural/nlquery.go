//  Copyright 2024-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package natural

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/distributed"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/natural/ai_gateway"
	"github.com/couchbase/query/primitives/couchbase"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

const MAX_KEYSPACES = 4

var _CHAT_LIMIT int

const (
	maxconcurrency       = 4
	maxWaiters           = 16
	waitTimeout          = 20 * time.Second
	maxCorrectionRetries = 4
)

const _CHAT_LOG_PREFIX = "NLCHAT:"

var naturalchatHistory *util.GenCache

type naturalReqThrottler struct {
	gate       chan bool
	waiters    int32
	maxwaiters int32
}

type naturalOutput int

const (
	SQL naturalOutput = iota
	JSUDF
	FTSSQL
	UNDEFINED_NATURAL_OUTPUT
)

func NewNaturalOutput(s string) naturalOutput {
	switch strings.ToUpper(s) {
	case "SQL":
		return SQL
	case "JSUDF":
		return JSUDF
	case "FTSSQL":
		return FTSSQL
	default:
		return UNDEFINED_NATURAL_OUTPUT
	}
}

func (n naturalOutput) String() string {
	var s string
	switch n {
	case SQL:
		s = "SQL"
	case JSUDF:
		s = "JSUDF"
	case FTSSQL:
		s = "FTSSQL"
	default:
		s = "UNDEFINED_NATURAL_OUTPUT"
	}
	return s
}

var nlreqThrottler naturalReqThrottler

func (this *naturalReqThrottler) getWaiter() errors.Error {
	if atomic.AddInt32(&this.waiters, 1) >= this.maxwaiters {
		atomic.AddInt32(&this.waiters, -1)
		return errors.NewNaturalLanguageRequestError(errors.E_NL_TOO_MANY_WAITERS)
	}
	return nil
}

func (this *naturalReqThrottler) releaseWaiter() {
	atomic.AddInt32(&this.waiters, -1)
}

func (this *naturalReqThrottler) nlgate() chan bool {
	return this.gate
}

func init() {

	nlreqThrottler = naturalReqThrottler{
		gate:       make(chan bool, maxconcurrency),
		maxwaiters: maxWaiters,
	}

	for i := 0; i < maxconcurrency; i++ {
		nlreqThrottler.nlgate() <- true
	}

	_CHAT_LIMIT = util.NumCPU() * 2

	naturalchatHistory = util.NewGenCache(_CHAT_LIMIT)
}

const _CHAT_INACTIVITY_TIMEOUT = 60 * time.Minute

type ChatEntry struct {
	Id        string
	prompt    *prompt
	Keyspaces []*algebra.Path
	Removed   bool
	users     []string
	Paused    bool
	Summary   string
	// Tokens accumulates LLM token usage across every request/conversation
	Tokens LLMTokenUsage
	// samples caches representative per-field sample values for the conversation
	// while it is on the slm provider. Held in memory only, excluded from
	// MarshalJSON, populated only for slm and cleared on a switch to a non-slm
	// provider, so raw sample values never reach a non-slm provider nor the
	// persisted chat document.
	samples           map[string]map[string][]interface{}
	timer             *time.Timer
	timerGen          int
	inactivityTimeout time.Duration
	sync.Mutex
}

// checkUser returns E_NL_CHAT_WRONG_USER if datastorecreds doesn't include any of the chat users.
// NOTE: nil datastorecreds represents admin access
func (ce *ChatEntry) CheckUser(datastorecreds []string) errors.Error {
	if datastorecreds != nil {
		for _, user := range datastorecreds {
			if slices.Contains(ce.users, user) {
				return nil
			}
		}
		return errors.NewNaturalLanguageRequestError(errors.E_NL_CHAT_WRONG_USER)
	}
	return nil
}

func (ce *ChatEntry) AlterTimeout(datastorecreds []string, timeout time.Duration) errors.Error {
	ce.Lock()
	defer ce.Unlock()
	if ce.Removed || ce.Paused {
		return nil
	}
	if err := ce.CheckUser(datastorecreds); err != nil {
		return err
	}
	ce.inactivityTimeout = timeout
	ce.resetInactivityTimerLocked()
	return nil
}

// resetInactivityTimerLocked restarts the inactivity timer using the per-chat timeout (or the default).
// Must be called while holding ce.Lock() or before the entry is added to the cache.
func (ce *ChatEntry) resetInactivityTimerLocked() {
	if ce.timer != nil {
		ce.timer.Stop()
	}
	timeout := ce.effectiveInactivityTimeout()
	ce.timerGen++
	gen := ce.timerGen
	ce.timer = time.AfterFunc(timeout, func() {
		ce.Lock()
		defer ce.Unlock()
		// was the fired expiry go routine from the most recent timer, last minute resets?
		// if yes, is the entry not already removed or paused
		if gen == ce.timerGen && !ce.Removed && !ce.Paused {
			DeleteConversation(ce.Id)
			ce.Removed = true
			logging.Infof("%s ChatEntry with id %s removed due to inactivity", _CHAT_LOG_PREFIX, ce.Id)
		}
	})
}

func (ce *ChatEntry) stopInactivityTimer() {
	if ce.timer != nil {
		ce.timer.Stop()
		ce.timer = nil
	}
}

func (ce *ChatEntry) effectiveInactivityTimeout() time.Duration {
	if ce.inactivityTimeout > 0 {
		return ce.inactivityTimeout
	}
	return _CHAT_INACTIVITY_TIMEOUT
}

func IsChatCacheFull() bool {
	return naturalchatHistory.Size() >= _CHAT_LIMIT
}

func AddConversation(ce *ChatEntry, id string) {
	naturalchatHistory.Add(ce, id, nil)
}

func GetConversation(id string) interface{} {
	return naturalchatHistory.Get(id, nil)
}

func DeleteConversation(id string) {
	naturalchatHistory.Delete(id, nil)
}

func ForEachConversation(nonBlocking func(chatId string, entry *ChatEntry) bool, blocking func() bool) {
	dummyF := func(chatId string, entry interface{}) bool {
		ce := entry.(*ChatEntry)
		return nonBlocking(chatId, ce)
	}
	naturalchatHistory.ForEach(dummyF, blocking)
}

func CountCoversations() int {
	return naturalchatHistory.Size()
}

func FormatChatEntry(ce *ChatEntry) map[string]interface{} {
	item := map[string]interface{}{}

	if ceId := ce.Id; ceId != "" {
		item["chatId"] = ceId
	}
	if cekeyspaces := ce.Keyspaces; len(cekeyspaces) > 0 {
		keyspaces := make([]interface{}, len(ce.Keyspaces))
		for i, p := range ce.Keyspaces {
			keyspaces[i] = p.ProtectedString()
		}
		item["keyspaces"] = keyspaces
	}
	if pmpt := ce.prompt; pmpt != nil {
		item["prompt"] = value.NewMarshalledValue(pmpt)
	}
	if users := ce.users; users != nil {
		users := make([]interface{}, len(ce.users))
		for i, u := range ce.users {
			users[i] = u
		}
		item["users"] = users
	}
	if summary := ce.Summary; summary != "" {
		item["summary"] = summary
	}
	item["inactivityTimeout"] = ce.effectiveInactivityTimeout().String()
	return item
}

type NaturalContext interface {
	datastore.Context
	datastore.QueryContext
}

// Prompt
type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type completionSettings struct {
	Model string `json:"model"`
	// Stream and MaxTokens are used by the Capella (iQ) path; the direct
	// path leaves them zero and carries max-tokens on the gateway request. Both
	// are omitempty so they never appear in direct-path persisted prompts.
	Stream bool `json:"stream,omitempty"`
	// Optional fields
	MaxTokens   int     `json:"max_tokens,omitempty"`
	Temperature float64 `json:"temperature,omitempty"`
	Seed        int     `json:"seed,omitempty"`
}

type prompt struct {
	InitMessages       []message          `json:"initMessages"`
	CompletionSettings completionSettings `json:"completionSettings"`
	// Provider is the direct ai_gateway field; omitempty so it never appears in
	// Capella (iQ) payloads, which carry only "vendor" (iQ is sensitive to an
	// unexpected empty "provider").
	Provider string `json:"provider,omitempty"`
	// Vendor is the Capella-path counterpart of Provider; omitempty so it
	// never appears in direct-path persisted prompts.
	Vendor   string    `json:"vendor,omitempty"`
	Messages []message `json:"messages"`
	Size     int       `json:"size"`
	// samples holds representative per-field sample values (keyspace -> field ->
	// values) from INFER. It is unexported so it is never marshaled into the
	// persisted conversation, and it is injected into the outbound request only
	// for the slm provider (see doChatCompletion) -- so raw sample values never
	// enter Messages nor reach a non-slm provider.
	samples map[string]map[string][]interface{}
}

const _INIT_SIZE = 250
const _MAX_PROMPT_SIZE = util.MiB

// CheckAndReturnErrorResponse scans the LLM-generated content for a #ERR marker.
// When present, it extracts and returns the error message after the marker.
// Returns nil when no error marker is found.
func CheckAndReturnErrorResponse(content string) error {
	if n := strings.Index(content, "#ERR"); n != -1 {
		return fmt.Errorf("%s", strings.TrimRight(content[n+6:], "\n `"))
	}
	return nil
}

func throttleRequest() errors.Error {
	if err := nlreqThrottler.getWaiter(); err != nil {
		return err
	}
	defer nlreqThrottler.releaseWaiter()

	select {
	case <-nlreqThrottler.nlgate():
		defer func() {
			nlreqThrottler.nlgate() <- true
		}()
		return nil
	case <-time.After(waitTimeout):
		return errors.NewNaturalLanguageRequestError(errors.E_NL_TIMEOUT)
	}
}

func completeConversationPromptLocked(content string, ce *ChatEntry, prompt *prompt) {
	if content != "" {
		assistantmessage := message{
			Role:    "assistant",
			Content: content,
		}
		prompt.Messages = append(prompt.Messages, assistantmessage)
		prompt.Size += len(content)
		ce.prompt = prompt
		ce.resetInactivityTimerLocked()
		naturalchatHistory.Add(ce, ce.Id, nil)
	}
}

func ProcessBeginChat(naturalcontext string, datastorecreds []string, keyspaces []*algebra.Path, timeout time.Duration) (string, errors.Error) {

	if IsChatCacheFull() {
		return "", errors.NewNaturalLanguageRequestError(errors.E_NL_CHAT_CACHE_FULL)
	}

	chatId, err := util.UUIDV4()
	if err != nil {
		return "", errors.NewNaturalLanguageRequestError(errors.E_NL_CHAT_FAIL, err.Error())
	}

	ce := &ChatEntry{
		Id:                chatId,
		Keyspaces:         keyspaces,
		users:             datastorecreds,
		inactivityTimeout: timeout,
	}
	ce.resetInactivityTimerLocked()
	AddConversation(ce, chatId)
	return chatId, nil
}

func ProcessEndChat(chatId string, datastorecreds []string, chatTokens *LLMTokenUsage) errors.Error {

	rv := GetConversation(chatId)
	if rv == nil {
		return errors.NewNaturalLanguageRequestError(errors.E_NL_NO_SUCH_CHAT, chatId)
	}
	ce, ok := rv.(*ChatEntry)
	if !ok {
		return errors.NewNaturalLanguageRequestError(errors.E_NL_CHAT_FAIL, "failed to cast cache entry")
	}
	ce.Lock()
	if err := ce.CheckUser(datastorecreds); err != nil {
		ce.Unlock()
		return err
	}
	ce.stopInactivityTimer()
	DeleteConversation(chatId)
	ce.Removed = true
	// Surface the conversation's running token total on the end response. Read
	// under the entry lock. Only the direct path accumulates ce.Tokens; on the
	// Capella path this stays zero and is suppressed by FmtNaturalChatTokens.
	if chatTokens != nil {
		*chatTokens = ce.Tokens
	}
	ce.Unlock()
	logging.Infof("%s Chat with id %s ended", _CHAT_LOG_PREFIX, chatId)
	return nil
}

func ParseChatTimeout(v interface{}) (time.Duration, error) {
	var timeout time.Duration
	switch val := v.(type) {
	case float64:
		timeout = time.Duration(val) * time.Second
	case string:
		if d, err := time.ParseDuration(val); err == nil {
			timeout = d
		} else if n, err := strconv.ParseFloat(val, 64); err == nil {
			timeout = time.Duration(n) * time.Second
		} else {
			return 0, fmt.Errorf("invalid timeout string: %v", val)
		}
	default:
		return 0, fmt.Errorf("invalid timeout type: %T", v)
	}

	if timeout < _CHAT_INACTIVITY_TIMEOUT {
		return 0, fmt.Errorf("inactivity timeout must be at least %v", _CHAT_INACTIVITY_TIMEOUT)
	}

	return timeout, nil
}

// use nil datastorecreds for admin access
func ProcessAlterChat(chatId string, datastorecreds []string, timeout time.Duration) errors.Error {
	if chatId == "" {
		return errors.NewNaturalLanguageRequestError(errors.E_NL_MISSING_CHAT_ID)
	}
	if timeout <= 0 {
		return errors.NewNaturalLanguageRequestError(errors.E_NL_INVALID_CHAT_TIMEOUT, "timeout must be a positive number of seconds")
	}

	rv := GetConversation(chatId)
	if rv == nil {
		return errors.NewNaturalLanguageRequestError(errors.E_NL_NO_SUCH_CHAT, chatId)
	}
	ce, ok := rv.(*ChatEntry)
	if !ok {
		return errors.NewNaturalLanguageRequestError(errors.E_NL_CHAT_FAIL, "failed to cast cache entry")
	}
	return ce.AlterTimeout(datastorecreds, timeout)
}

const CHAT_DOC_TTL_DURATION = 7 * 24 * time.Hour
const summarizeThreshold = 1024 * 10
const summarizeMessageLen = 8

const (
	maxRetry = 6
	interval = 100 * time.Millisecond
)

const _BATCH_SIZE = 64

var _STRING_ANNOTATED_POOL = value.NewStringAnnotatedPool(_BATCH_SIZE)

func ProcessResumeChat(chatId, requestId string, datastorecreds []string, chatTokens *LLMTokenUsage) errors.Error {
	if chatId == "" {
		return errors.NewNaturalLanguageRequestError(errors.E_NL_MISSING_CHAT_ID)
	}

	if IsChatCacheFull() {
		return errors.NewNaturalLanguageRequestError(errors.E_NL_CHAT_RESUME_FAILED,
			errors.NewNaturalLanguageRequestError(errors.E_NL_CHAT_CACHE_FULL))
	}

	hasquerymetadata, err := hasQueryMetadataForNLChat(false, requestId, "", false)
	if err != nil {
		return errors.NewNaturalLanguageRequestError(errors.E_NL_CHAT_RESUME_FAILED,
			fmt.Sprintf("failed to get query metadata: %v", err))
	} else if !hasquerymetadata {
		return errors.NewMissingQueryMetadataError("RESUME CHAT")
	}

	store := datastore.GetDatastore()
	if store == nil {
		err := errors.NewNoDatastoreError()
		return errors.NewNaturalLanguageRequestError(errors.E_NL_CHAT_RESUME_FAILED, "failed to get datastore", err)
	}

	queryMetadata, err := store.GetQueryMetadata()
	if err != nil {
		return errors.NewNaturalLanguageRequestError(errors.E_NL_CHAT_RESUME_FAILED, "failed to get query metadata", err)
	}

	fetchMap := _STRING_ANNOTATED_POOL.Get()
	defer _STRING_ANNOTATED_POOL.Put(fetchMap)
	key := fmt.Sprintf("%s%s", CHAT_DOC_PREFIX, chatId)

	queryContext := datastore.GetDurableQueryContextFor(queryMetadata)
	ce := &ChatEntry{}
	var chatdoc value.AnnotatedValue
	var ok bool
	claimed := false

	claimInterval := interval
	for claimFetch := 0; claimFetch < maxRetry; claimFetch++ {

		errs := queryMetadata.Fetch([]string{key}, fetchMap, queryContext, nil, nil, false)
		if errs != nil {
			return errors.NewNaturalLanguageRequestError(errors.E_NL_CHAT_RESUME_FAILED,
				fmt.Sprintf("errs in fetching the chat document: %v", errs))
		}

		if chatdoc, ok = fetchMap[key]; !ok || chatdoc == nil {
			return errors.NewNaturalLanguageRequestError(errors.E_NL_CHAT_RESUME_FAILED,
				fmt.Sprintf("chat with id:%s is not found in QUERY_METADATA", chatId))
		}

		val := chatdoc.GetValue()
		if vt := val.Type(); vt != value.OBJECT {
			return errors.NewNaturalLanguageRequestError(errors.E_NL_UNEXPECTED_CHAT_DOC,
				fmt.Sprintf("value type for chat document: %s expected object type %s", val, vt))
		}

		claimer, ok := val.Field("claimer")
		if ok && claimer.ToString() != distributed.RemoteAccess().WhoAmI() {
			claimtime, ok := val.Field("claim_time")
			if !ok {
				return errors.NewNaturalLanguageRequestError(errors.E_NL_UNEXPECTED_CHAT_DOC,
					"\"claim_time\" field is not found in the chat document")
			}
			if ct := claimtime.Type(); ct != value.STRING {
				return errors.NewNaturalLanguageRequestError(errors.E_NL_UNEXPECTED_CHAT_DOC,
					fmt.Sprintf("unexpected value type for \"claim_time\" field in the chat document: %s expected string", ct), err)
			}
			ct, perr := time.Parse(util.DEFAULT_FORMAT, claimtime.ToString())
			if perr != nil {
				return errors.NewNaturalLanguageRequestError(errors.E_NL_CHAT_RESUME_FAILED,
					"failed to parse claim_time field in the chat document", perr)
			}
			if time.Since(ct) < 2*time.Minute {
				return errors.NewNaturalLanguageRequestError(errors.E_NL_CHAT_RESUME_FAILED,
					fmt.Sprintf("chat is currently claimed by %s", claimer.ToString()))
			}
			// orphaned claim, can be claimed
		}

		b, err := GetChatDataFromObjectValue(val)
		if err != nil {
			return err
		}

		uerr := ce.UnmarshalJSON(b)
		if uerr != nil {
			return errors.NewNaturalLanguageRequestError(errors.E_NL_CHAT_RESUME_FAILED, "unmarshalling decoded chat failed", uerr)
		}

		if ce.users == nil || len(ce.users) == 0 {
			return errors.NewNaturalLanguageRequestError(errors.E_NL_UNEXPECTED_CHAT_DOC,
				"\"users\" field is not found in the chat document")
		}

		if err := ce.CheckUser(datastorecreds); err != nil {
			return err
		}

		udpairs := make([]value.Pair, 1)
		udpairs[0].Name = key
		chatdoc.SetField("claimer", value.NewValue(distributed.RemoteAccess().WhoAmI()))
		chatdoc.SetField("claim_time", value.NewValue(time.Now().Format(util.DEFAULT_FORMAT)))
		udpairs[0].Value = chatdoc

		retryClaim := false
		claimUpdateInterval := interval
		for claimUpdate := 0; claimUpdate < maxRetry; claimUpdate++ {
			_, _, errs = queryMetadata.Update(udpairs, queryContext, false)
			if len(errs) > 0 {
				if couchbase.CanRetryWithRefresh(errs[0]) {
					time.Sleep(claimUpdateInterval)
					claimUpdateInterval *= 2
				} else if errs[0].HasCause(errors.E_CAS_MISMATCH) || errs[0].ContainsText("SYNC_WRITE_IN_PROGRESS") {
					// some else tried to resume concurrently
					chatdoc.Recycle()
					chatdoc = nil
					fetchMap[key] = nil
					ce.Reset()
					retryClaim = true
					break
				} else {
					logging.Errorf("%s Chat claim failed: error updating QUERY_METADATA bucket (key %s): %v",
						_CHAT_LOG_PREFIX, key, errs)
					return errors.NewNaturalLanguageRequestError(errors.E_NL_CHAT_RESUME_FAILED,
						fmt.Sprintf("err updating the chat document: %v", errs))
				}
			} else {
				claimed = true
				break
			}
		}

		if retryClaim {
			claimInterval *= 2
			time.Sleep(claimInterval)
			continue
		}

		if claimed {
			logging.Infof("%s Chat claimed successfully for chat id: %s", _CHAT_LOG_PREFIX, chatId)
			break
		}
	}

	if !claimed {
		logging.Errorf("%s Chat claim failed after %d retries: failed to update the chat document for chat id: %s", _CHAT_LOG_PREFIX, maxRetry, chatId)
		return errors.NewNaturalLanguageRequestError(errors.E_NL_CHAT_RESUME_FAILED,
			fmt.Sprintf("failed to claim chat document for chat id: %s after retries: %d", chatId, maxRetry))
	}

	dpairs := make([]value.Pair, 1)
	dpairs[0].Name = key
	completeClaimInterval := interval
	claimcompleted := true
	for claimComplete := 0; claimComplete < maxRetry; claimComplete++ {
		claimcompleted = false
		_, _, errs := queryMetadata.Delete(dpairs, queryContext, false)
		if len(errs) > 0 {
			if couchbase.CanRetryWithRefresh(errs[0]) {
				time.Sleep(completeClaimInterval)
				completeClaimInterval *= 2
			} else {
				logging.Errorf("%s Chat claim completion failed: error deleting from QUERY_METADATA bucket (key %s): %v",
					_CHAT_LOG_PREFIX, key, errs)
				return errors.NewNaturalLanguageRequestError(errors.E_NL_CHAT_RESUME_FAILED,
					fmt.Sprintf("err deleting the chat document: %v", errs))
			}
		} else {
			logging.Infof("%s Chat claim completed for chat id: %s", _CHAT_LOG_PREFIX, chatId)
			claimcompleted = true
			break
		}
	}

	if !claimcompleted {
		logging.Errorf("%s Chat claim completion failed after %d retries:"+
			" error in deleting the chat document for chat id: %s", _CHAT_LOG_PREFIX, maxRetry, chatId)
		return errors.NewNaturalLanguageRequestError(errors.E_NL_CHAT_RESUME_FAILED,
			fmt.Sprintf("failed to complete the claim for chat document for chat id: %s after retries: %d", chatId, maxRetry))
	}

	ce.Id = chatId
	// Surface the conversation's running token total (restored from the persisted
	// chat document) on the resume response. Read while ce is still private to this
	// goroutine, before AddConversation publishes it. Only the direct path
	// accumulates ce.Tokens; on the Capella path this is zero and is suppressed by
	// FmtNaturalChatTokens.
	if chatTokens != nil {
		*chatTokens = ce.Tokens
	}
	ce.resetInactivityTimerLocked()
	AddConversation(ce, ce.Id)
	logging.Infof("%s Chat with id %s resumed", _CHAT_LOG_PREFIX, chatId)
	return nil
}

func getStatement(content string, nloutputOpt naturalOutput) (string, errors.Error) {
	if content == "" {
		return "", errors.NewNaturalLanguageRequestError(errors.E_NL_FAIL_GENERATED_STMT, "empty response")
	}
	switch nloutputOpt {
	case SQL, FTSSQL:
		return getSQLContent(content), nil
	case JSUDF:
		return getJsContent(content), nil
	default:
		return "", errors.NewServiceErrorUnrecognizedValue("natural_output", nloutputOpt.String())
	}
}

func getSQLContent(content string) string {
	sqlstmt := strings.TrimSpace(content)
	// Strip an optional markdown code fence. Hosted providers are told to emit
	// ```sql ... ```; self-hosted (slm) models often emit a plain ``` ... ```
	// fence, so handle both (and no fence).
	if strings.HasPrefix(sqlstmt, "```") {
		sqlstmt = strings.TrimPrefix(sqlstmt, "```sql")
		sqlstmt = strings.TrimPrefix(sqlstmt, "```")
		sqlstmt = strings.TrimSuffix(sqlstmt, "```")
		sqlstmt = strings.TrimSpace(sqlstmt)
	}
	if end := len(sqlstmt) - 1; end >= 0 && sqlstmt[end] == ';' {
		sqlstmt = sqlstmt[:end]
	}
	return strings.TrimSpace(sqlstmt)
}

func getJsContent(content string) string {
	stmt := strings.TrimSpace(content)
	// Strip an optional markdown code fence. Hosted providers are told to emit
	// ```javascript ... ```; the slm system template instructs a plain
	// ``` ... ``` fence, so handle both (and no fence). Language tags must be
	// trimmed before the bare fence so "```javascript" is not left as
	// "javascript".
	if strings.HasPrefix(stmt, "```") {
		stmt = strings.TrimPrefix(stmt, "```javascript")
		stmt = strings.TrimPrefix(stmt, "```js")
		stmt = strings.TrimPrefix(stmt, "```sql")
		stmt = strings.TrimPrefix(stmt, "```")
		stmt = strings.TrimSuffix(stmt, "```")
		stmt = strings.TrimSpace(stmt)
	}
	return stmt
}

// ---- shared prompt-input construction (path-agnostic) ----
// The following helpers are shared by both the Capella and direct paths. They
// build the keyspace schema info and the SQL user-message body that are
// identical across paths. Anything that differs per path (the wire vendor/
// provider field, the temperature function, the system message, and the slm
// prompt) stays in the per-path builders.

// _MAX_SAMPLE_STRING_LEN bounds the length, in bytes, of an individual string
// sample value sent to the provider. Long free-text values add prompt tokens
// without helping the model pick literals, so each string sample is truncated to
// this many bytes in the natural (AI) layer.
const _MAX_SAMPLE_STRING_LEN = 50

// capSampleStrings truncates each string sample in a field's sample-value slice
// to at most _MAX_SAMPLE_STRING_LEN bytes. The cut is snapped back to a UTF-8
// rune boundary so a multi-byte character is never split into invalid UTF-8.
// Non-string samples are left untouched. Runs only on the AI path
// (includeSamples), after samples are received from INFER.
func capSampleStrings(arr []interface{}) []interface{} {
	for i, e := range arr {
		if s, ok := value.NewValue(e).Actual().(string); ok && len(s) > _MAX_SAMPLE_STRING_LEN {
			end := _MAX_SAMPLE_STRING_LEN
			for end > 0 && !utf8.RuneStart(s[end]) {
				end--
			}
			arr[i] = value.NewValue(s[:end])
		}
	}
	return arr
}

// collectSchemaFromInfer extracts, from a single keyspace's INFER result, the
// per-field type strings (rendered into the prompt schema and the persisted
// conversation) and, when includeSamples is set, the per-field representative
// sample values. Types and samples are returned separately on purpose: types
// are safe to persist, whereas samples are provider-gated and are kept out of
// the persisted messages (see prompt.samples). The returned samples map is nil
// when includeSamples is false or INFER reported no samples.
func collectSchemaFromInfer(schema map[string]string, inferSchema value.Value,
	includeSamples bool) (map[string]string, map[string][]interface{}) {

	var samples map[string][]interface{}
	if v, ok := inferSchema.Index(0); ok {
		if prop, ok := v.Field("properties"); ok {
			schemaFieldNames := []string{}
			schemaFieldNames = prop.FieldNames(schemaFieldNames)
			for _, fieldname := range schemaFieldNames {
				if fieldname == "~meta" {
					continue
				}

				if fieldSpecific, ok := prop.Field(fieldname); ok {
					if typeinfo, ok := fieldSpecific.Field("type"); ok {
						if typeinfo.Type() == value.ARRAY {
							var typestring strings.Builder
							var typestr string
							if typestrslice, ok := typeinfo.Actual().([]interface{}); ok {
								if typestr, ok = typestrslice[0].(string); ok {
									typestring.WriteString(typestr)
									typestring.WriteRune(' ')
								}
								for _, s := range typestrslice[1:] {
									if typestr, ok = s.(string); ok {
										typestring.WriteString("or ")
										typestring.WriteString(typestr)
									}
								}
							}
							schema[fieldname] = typestring.String()
						} else {
							schema[fieldname] = typeinfo.String()
						}
					}

					if includeSamples {
						if s, ok := fieldSpecific.Field("samples"); ok && s.Type() == value.ARRAY {
							if arr, ok := s.Actual().([]interface{}); ok && len(arr) > 0 {
								if samples == nil {
									samples = map[string][]interface{}{}
								}
								samples[fieldname] = capSampleStrings(arr)
							}
						}
					}
				}
			}
		}
	}

	return schema, samples
}

func inferSchema(schema map[string]string, p *algebra.Path, context NaturalContext,
	includeSamples bool) (map[string]string, map[string][]interface{}, errors.Error) {

	keyspace, err := datastore.GetKeyspace(p.Parts()...)
	if err != nil {
		return nil, nil, err
	}

	conn := datastore.NewValueConnection(context)
	infer, err := context.Datastore().Inferencer(datastore.INF_DEFAULT)
	if err != nil {
		return nil, nil, err
	}
	infer.InferKeyspace(context, keyspace, nil, conn)

	inferSchema, ok := <-conn.ValueChannel()

	var samples map[string][]interface{}
	if inferSchema != nil && ok {
		schema, samples = collectSchemaFromInfer(schema, inferSchema, includeSamples)
	}

	return schema, samples, nil
}

func keyspacesInfoForPrompt(keyspaceInfo map[string]interface{}, elems []*algebra.Path,
	context NaturalContext, includeSamples bool) (map[string]interface{},
	map[string]map[string][]interface{}, errors.Error) {

	var err errors.Error
	var samplesByKeyspace map[string]map[string][]interface{}
	priv := auth.NewPrivileges()

	var ds datastore.Datastore
	if context != nil {
		ds = context.Datastore()
		if ds == nil {
			return nil, nil, errors.NewNaturalLanguageRequestError(errors.E_NL_CONTEXT, fmt.Errorf("no datastore"))
		}
	} else {
		return nil, nil, errors.NewNaturalLanguageRequestError(errors.E_NL_CONTEXT, fmt.Errorf("no context"))
	}
	for _, p := range elems {
		ps := p.SimpleString()
		if p.IsSystem() || (strings.Contains(ps, ":") && algebra.IsSystemName(ps)) {
			return nil, nil, errors.NewNaturalLanguageRequestError(errors.E_NL_CONTEXT,
				fmt.Errorf("system keyspace is not allowed: %s", ps))
		}
		priv.List = priv.List[:0]
		priv.Add(ps, auth.PRIV_QUERY_SELECT, auth.PRIV_PROPS_NONE)
		err = ds.Authorize(priv, context.Credentials())
		if err != nil {
			return nil, nil, errors.NewNaturalLanguageRequestError(errors.E_NL_CONTEXT, err)
		}
		schema := map[string]string{}
		var ksSamples map[string][]interface{}
		schema, ksSamples, err = inferSchema(schema, p, context, includeSamples)

		if err != nil {
			return nil, nil, errors.NewNaturalLanguageRequestError(errors.E_NL_PROMPT_INFER, p.ProtectedString(), err)
		}
		info := map[string]interface{}{}
		info["schema"] = schema
		fullpath := p.ProtectedString()
		info["fullpath"] = fullpath[strings.Index(fullpath, ":"):]

		keyspaceInfo[p.Keyspace()] = info
		if len(ksSamples) > 0 {
			if samplesByKeyspace == nil {
				samplesByKeyspace = map[string]map[string][]interface{}{}
			}
			samplesByKeyspace[p.Keyspace()] = ksSamples
		}
	}

	return keyspaceInfo, samplesByKeyspace, nil
}

// appendSQLUserMessage builds the user-message turn shared by the Capella and
// direct (non-slm) SQL prompts and appends it to rv. The caller is responsible
// for the prompt shell (system message, vendor/provider field, temperature,
// stream). rv.Size is updated in place.
func appendSQLUserMessage(rv *prompt, keyspaceInfo map[string]interface{},
	naturalPrompt, summary, hint string, forfts bool) errors.Error {

	var userMessage string
	var userMessageBuf strings.Builder

	binKeyspacesInfo, err := json.Marshal(keyspaceInfo)
	if err != nil {
		return errors.NewNaturalLanguageRequestError(errors.E_NL_PROMPT_SCHEMA_MARSHAL, err)
	}
	if summary != "" {
		userMessageBuf.WriteString("Summary of the conversation so far: " + summary + "\n\n")
	}
	userMessageBuf.WriteString("Information about keyspaces:\n\n")
	userMessageBuf.WriteString(string(binKeyspacesInfo))
	userMessageBuf.WriteString("\n\nPrompt: \"")
	userMessageBuf.WriteString(naturalPrompt)
	userMessageBuf.WriteString("\"")
	if hint != "" {
		userMessageBuf.WriteString("\n\nHint: \"")
		userMessageBuf.WriteString(hint)
		userMessageBuf.WriteString("\"")
	}
	userMessageBuf.WriteString("\n\nBased on the above Information, write valid SQL++ only and with no explanation." +
		"\n\nNote query context is unset." +
		"\n\nUse the fullpath from the information about keyspaces for retrieval along with an alias." +
		"\n\nAlias is for ease of use." +
		"\n\nQuote aliases with grave accent characters." +
		"\nMake use of RAW keyword when you require a non-object result, for example when comparing a field with a subquery's result set.")
	if forfts {
		userMessageBuf.WriteString("\n\nAlways add the USE Clause in the query to use the FTS index." +
			"\n\nFor example, SELECT a.*, ap.* FROM `travel-sample`.`inventory`.`airline` AS a USE INDEX " +
			"(USING FTS) JOIN `travel-sample`.`inventory`.`airport` AS ap USE INDEX (USING FTS)" +
			" ON a.country = ap.country WHERE a.country = \"United Kingdom\"" +
			"\n\nIn other words, always use USE INDEX (USING FTS) in the query.")
	}
	userMessageBuf.WriteString("\n\nReturn only a single SQL++ statement on a single line." +
		"\n\nIf you're sure the Prompt can't be used to generate a query, say " +
		"\n#ERR:\" and then explain why not without prefix.\n\n")
	rv.Size += userMessageBuf.Len()
	userMessage = userMessageBuf.String()
	rv.Messages = []message{
		message{
			Role:    "user",
			Content: userMessage,
		},
	}

	return nil
}

// appendSQLIterativeUserMessage builds the iterative (follow-up) user-message
// turn shared by the Capella and direct SQL paths and appends it to chat. The
// caller sets the vendor/provider field, model, and temperature on chat before
// calling. chat.Size is updated in place.
func appendSQLIterativeUserMessage(chat *prompt, naturalPrompt string, hint string, forfts bool) *prompt {
	var userMessage string
	var userMessageBuf strings.Builder

	userMessageBuf.WriteString("Your goal is to iterate on the previouly generated query by modifying it's code,")
	userMessageBuf.WriteString(" based on this prompt:")
	userMessageBuf.WriteString("\"")
	userMessageBuf.WriteString(naturalPrompt)
	userMessageBuf.WriteString("\".")
	if hint != "" {
		userMessageBuf.WriteString("\n\nHint: \"")
		userMessageBuf.WriteString(hint)
		userMessageBuf.WriteString("\"")
	}
	userMessageBuf.WriteString("\n\nRespond only with code and no explanation." +
		"\n\nNote query context is unset." +
		"\n\nUse the fullpath from the information about keyspaces for retrieval along with an alias." +
		"\n\nAlias is for ease of use." +
		"\n\nQuote aliases with grave accent characters.")
	if forfts {
		userMessageBuf.WriteString("\n\nAlways add the USE Clause in the query to use the FTS index." +
			"\n\nFor example, SELECT a.*, ap.* FROM `travel-sample`.`inventory`.`airline` AS a USE INDEX " +
			"(USING FTS) JOIN `travel-sample`.`inventory`.`airport` AS ap USE INDEX (USING FTS)" +
			" ON a.country = ap.country WHERE a.country = \"United Kingdom\"" +
			"\n\nIn other words, always use USE INDEX (USING FTS) in the query.")
	}
	userMessageBuf.WriteString("\n\nIf the previous message was a CREATE FUNCTION statement you don't have to repeat a CREATE FUNCTION." +
		"\nReturn only a single SQL++ statement on a single line." +
		"\n\nIf you're sure the Prompt can't be used to generate a query, say " +
		"\n#ERR:\" and then explain why not without prefix.\n\n")

	chat.Size += userMessageBuf.Len()
	userMessage = userMessageBuf.String()
	chat.Messages = append(chat.Messages, message{
		Content: userMessage,
		Role:    "user",
	})

	return chat
}

const CHAT_DOC_PREFIX = "aichat::"

func (ce *ChatEntry) MarshalJSON() ([]byte, error) {
	rv := map[string]interface{}{}
	if users := ce.users; users != nil {
		rv["users"] = users
	}
	keyspaces := make([]string, len(ce.Keyspaces))
	for i, k := range ce.Keyspaces {
		keyspaces[i] = k.ProtectedString()
	}
	rv["keyspaces"] = keyspaces
	if pmt := ce.prompt; pmt != nil {
		rv["prompt"] = pmt
	}
	if summ := ce.Summary; summ != "" {
		rv["summary"] = summ
	}
	if t := ce.Tokens; t.Prompt != 0 || t.Completion != 0 || t.Total != 0 {
		rv["tokens"] = map[string]interface{}{
			"prompt":     t.Prompt,
			"completion": t.Completion,
			"total":      t.Total,
		}
	}
	if timeout := ce.inactivityTimeout; timeout > 0 {
		rv["inactivity_timeout"] = timeout.String()
	}
	return json.Marshal(rv)
}

func (ce *ChatEntry) UnmarshalJSON(body []byte) error {
	var unmarshalledStruct struct {
		Keyspaces []string `json:"keyspaces"`
		Prompt    *prompt  `json:"prompt"`
		Users     []string `json:"users"`
		Summary   string   `json:"summary"`
		Timeout   string   `json:"inactivity_timeout"`
		Tokens    struct {
			Prompt     int `json:"prompt"`
			Completion int `json:"completion"`
			Total      int `json:"total"`
		} `json:"tokens"`
	}

	err := json.Unmarshal(body, &unmarshalledStruct)
	if err != nil {
		return err
	}

	if users := unmarshalledStruct.Users; users != nil {
		ce.users = users
	}
	if keyspaces := unmarshalledStruct.Keyspaces; keyspaces != nil {
		keyspacelist := strings.Join(keyspaces, ",")
		elems, err := algebra.ParseAndValidatePathList(keyspacelist, "default", "")
		if err != nil {
			return fmt.Errorf("error validating keyspaces: %s", err)
		}
		ce.Keyspaces = elems
	}
	if prompt := unmarshalledStruct.Prompt; prompt != nil {
		ce.prompt = prompt
	}
	if summary := unmarshalledStruct.Summary; summary != "" {
		ce.Summary = summary
	}
	if t := unmarshalledStruct.Tokens; t.Prompt != 0 || t.Completion != 0 || t.Total != 0 {
		ce.Tokens = LLMTokenUsage{Prompt: t.Prompt, Completion: t.Completion, Total: t.Total}
	}
	if timeout := unmarshalledStruct.Timeout; timeout != "" {
		if d, err := time.ParseDuration(timeout); err == nil && d > 0 {
			ce.inactivityTimeout = d
		} else {
			return fmt.Errorf("invalid inactivity timeout value: %s", timeout)
		}
	}
	return nil
}

func (ce *ChatEntry) Reset() {
	ce.stopInactivityTimer()
	ce.users = nil
	ce.Keyspaces = nil
	ce.prompt = nil
	ce.Id = ""
	ce.Summary = ""
	ce.Tokens = LLMTokenUsage{}
	ce.Removed = false
	ce.Paused = false
}

func GetChatDataFromObjectValue(val value.Value) ([]byte, errors.Error) {
	encodedchat, ok := val.Field("chat")
	if !ok {
		return nil, errors.NewNaturalLanguageRequestError(errors.E_NL_UNEXPECTED_CHAT_DOC,
			"\"chat\" field is not found in the chat document")
	}

	if et := encodedchat.Type(); et != value.STRING {
		return nil, errors.NewNaturalLanguageRequestError(errors.E_NL_UNEXPECTED_CHAT_DOC,
			fmt.Sprintf("value type for \"chat\" field in the chat document: %s expected string", et))
	}

	b, derr := base64.StdEncoding.DecodeString(encodedchat.ToString())
	if derr != nil {
		return nil, errors.NewNaturalLanguageRequestError(errors.E_NL_CHAT_RESUME_FAILED, "chat decoding failed", derr)
	}
	return b, nil
}

const (
	T_BEGINCHAT  = "BEGIN_CHAT"
	T_ENDCHAT    = "END_CHAT"
	T_PAUSECHAT  = "PAUSE_CHAT"
	T_RESUMECHAT = "RESUME_CHAT"
	T_ALTERCHAT  = "ALTER_CHAT"
)

func IsNaturalLanguageChatStatement(stmtType string) bool {
	switch stmtType {
	case T_BEGINCHAT, T_ENDCHAT, T_PAUSECHAT, T_RESUMECHAT, T_ALTERCHAT:
		return true
	default:
		return false
	}
}

// ---------------------------------------------------------------------------
// ai_gateway bridge
//
// The natural package owns prompt construction, the chat lifecycle and the
// parse/retry loop; the ai_gateway package owns all provider interaction. The
// functions below map the natural prompt onto the gateway's neutral request
// schema and preserve the API consumed by the server and by the rest of
// nlquery.go.
// ---------------------------------------------------------------------------

// NaturalConfig is the parsed natural_config configuration. It is an alias for
// ai_gateway.Config so existing callers keep their types while the gateway owns
// the definition.
type NaturalConfig = ai_gateway.Config

// LLMTokenUsage is the normalized token accounting from the gateway's common
// response. It is an alias for ai_gateway.TokenUsage so the server can capture
// the per-request total without importing the gateway package directly.
type LLMTokenUsage = ai_gateway.TokenUsage

// ParseNaturalConfig reads and validates the natural_config request parameter,
// delegating to the gateway.
func ParseNaturalConfig(naturalConfig value.Value) (*NaturalConfig, errors.Error) {
	return ai_gateway.ParseConfig(naturalConfig)
}

// IsCapellaPath reports whether a natural language request should be served by
// the Capella (iQ) path rather than the direct ai_gateway path. Capella
// is selected when the request carries any Capella credential (natural_cred
// and/or natural_orgid); otherwise the request falls through to the direct path
// keyed on natural_config. This is the single source of truth for the routing
// decision, used by both the HTTP layer and the server dispatch.
func IsCapellaPath(nlCred, nlOrgId string) bool {
	return nlCred != "" || nlOrgId != ""
}

// slmSamplesBlock renders cached per-field sample values (keyspace -> field ->
// values) into a compact context block appended to the final user turn for the
// slm provider. Returns "" when there is nothing to add so the caller can skip
// injection.
