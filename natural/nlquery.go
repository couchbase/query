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
	// samples caches a representative sample tree per field for the conversation
	// while it is on the slm provider. Held in memory only, excluded from
	// MarshalJSON, populated only for slm and cleared on a switch to a non-slm
	// provider, so raw sample values never reach a non-slm provider nor the
	// persisted chat document.
	samples           map[string]map[string]*sampleField
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
	// samples holds a representative sample tree per keyspace field (keyspace ->
	// field -> tree) from INFER, mirroring the nesting of the schema tree but
	// carrying Samples instead. It is unexported so it is never marshaled into
	// the persisted conversation, and it is injected into the outbound request
	// only for the slm provider (see doChatCompletion) -- so raw sample values
	// never enter Messages nor reach a non-slm provider.
	samples map[string]map[string]*sampleField
}

const _INIT_SIZE = 250
const _MAX_PROMPT_SIZE = util.MiB

// CheckAndReturnErrorResponse scans the LLM-generated content for a #ERR marker.
// When present, it extracts and returns the error message after the marker.
// Returns nil when no error marker is found.
func CheckAndReturnErrorResponse(content string) error {
	if n := strings.Index(content, "#ERR"); n != -1 {
		if len(content) > n+6 {
			return fmt.Errorf("%s", strings.TrimRight(content[n+6:], "\n `"))
		} else {
			return fmt.Errorf("unexpected empty error response from LLM")
		}
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

	// Resume is a one-time operation: a successful resume consumes (deletes) the
	// chat document from QUERY_METADATA and republishes the chat in the local
	// active-chat cache. If the chat is already active here, it has already been
	// resumed -- report that plainly instead of the misleading "not found in
	// QUERY_METADATA" that the metadata fetch below would otherwise produce.
	if rv := GetConversation(chatId); rv != nil {
		if ce, ok := rv.(*ChatEntry); ok {
			ce.Lock()
			// Re-check under the entry lock. A concurrent pause/end (or an
			// inactivity expiry) may have removed this entry from the cache and
			// set Paused/Removed after our GetConversation read above -- all of
			// those writers hold this lock while doing so. If that happened the
			// chat is no longer active, so fall through to the normal resume path:
			// a just-paused chat is picked up from QUERY_METADATA (the pause
			// writes the document under this same lock, before setting Paused),
			// and an ended chat is correctly reported as not found.
			if !ce.Removed && !ce.Paused {
				if err := ce.CheckUser(datastorecreds); err != nil {
					ce.Unlock()
					return err
				}
				ce.Unlock()
				return errors.NewNaturalLanguageRequestError(errors.E_NL_CHAT_ALREADY_ACTIVE, chatId)
			}
			ce.Unlock()
		}
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
			// No paused chat document exists for this id. This is not necessarily a
			// bad id: a prior resume consumes the document, so a chat that was
			// previously active and already resumed lands here too. The document
			// may also have expired (CHAT_DOC_TTL_DURATION). Do not claim the chat
			// never existed.
			return errors.NewNaturalLanguageRequestError(errors.E_NL_CHAT_RESUME_FAILED,
				fmt.Sprintf("no paused chat with id:%s found in QUERY_METADATA; it may have already been"+
					" resumed (resume is a one-time operation), expired, or never existed", chatId))
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

// _MAX_SAMPLE_ARRAY_LEN bounds how many elements of a nested array sample
// value are kept. INFER's own per-field sample count already bounds how many
// whole sample values a field carries, but not how large any one of those
// values is: an array-typed field's sample is a whole array value copied
// verbatim from a real document (see leafSamples), so without this a single
// large array field (e.g. hundreds of tags) would dump unabridged into the
// prompt.
const _MAX_SAMPLE_ARRAY_LEN = 5

// capSampleValue recursively bounds the size of a single raw sample value
// before it is embedded in the prompt: strings are truncated to
// _MAX_SAMPLE_STRING_LEN bytes, snapped back to a UTF-8 rune boundary so a
// multi-byte character is never split into invalid UTF-8; arrays are
// truncated to their first _MAX_SAMPLE_ARRAY_LEN elements; and both array
// elements and object field values are capped by recursing, so nesting can't
// smuggle an unbounded value past the top-level cap. Other scalars pass
// through unchanged.
func capSampleValue(v interface{}) interface{} {
	switch t := v.(type) {
	case string:
		if len(t) <= _MAX_SAMPLE_STRING_LEN {
			return t
		}
		end := _MAX_SAMPLE_STRING_LEN
		for end > 0 && !utf8.RuneStart(t[end]) {
			end--
		}
		return t[:end]
	case []interface{}:
		n := len(t)
		if n > _MAX_SAMPLE_ARRAY_LEN {
			n = _MAX_SAMPLE_ARRAY_LEN
		}
		capped := make([]interface{}, n)
		for i := 0; i < n; i++ {
			capped[i] = capSampleValue(t[i])
		}
		return capped
	case map[string]interface{}:
		capped := make(map[string]interface{}, len(t))
		for k, fv := range t {
			capped[k] = capSampleValue(fv)
		}
		return capped
	default:
		return v
	}
}

// capSampleBucket applies capSampleValue to every value in a field's
// sample-value slice. Runs only on the AI path (includeSamples), after
// samples are received from INFER.
func capSampleBucket(arr []interface{}) []interface{} {
	capped := make([]interface{}, len(arr))
	for i, e := range arr {
		capped[i] = capSampleValue(value.NewValue(e).Actual())
	}
	return capped
}

// schemaField is the recursive type tree collectFields builds from a single
// field's INFER schema, nesting object fields under Properties and array
// fields under Items. It never carries sample values, so it is always safe
// to persist (see prompt.samples).
type schemaField struct {
	Type       string                  `json:"type"`
	Properties map[string]*schemaField `json:"properties,omitempty"`
	Items      []*schemaField          `json:"items,omitempty"`
}

// sampleField is the samples counterpart of schemaField, built in lockstep by
// collectFields when includeSamples is true. It mirrors the same
// Properties/Items nesting so a samples tree can be read alongside the type
// tree. Samples is keyed by shape name (e.g. "string", "number") rather than
// a single flat list -- when a field has more than one shape (e.g. "string or
// number"), INFER keeps a separate list of example values per shape, and
// flattening them together would blur which examples apply when the field
// takes on which shape.
type sampleField struct {
	Samples    map[string][]interface{} `json:"samples,omitempty"`
	Properties map[string]*sampleField  `json:"properties,omitempty"`
	Items      []*sampleField           `json:"items,omitempty"`
}

// empty reports whether s carries no sample data anywhere in its subtree, so
// callers can omit a node entirely rather than including it as a content-free
// placeholder.
func (s *sampleField) empty() bool {
	return s == nil || (len(s.Samples) == 0 && len(s.Properties) == 0 && len(s.Items) == 0)
}

// collectProperties builds a type map and (when includeSamples is true) a
// samples map from a properties container's fields, by calling collectFields
// on each one. skip, if non-nil, excludes field names it returns true for --
// used to drop INFER's synthetic "~meta" field at the keyspace root; nested
// object properties pass nil. A field is only added to the samples map when
// its subtree actually carries sample data, so the returned map is nil when
// none of the fields do.
func collectProperties(props value.Value, includeSamples bool,
	skip func(name string) bool) (map[string]*schemaField, map[string]*sampleField) {

	types := map[string]*schemaField{}
	var samples map[string]*sampleField
	for _, name := range props.FieldNames(nil) {
		if skip != nil && skip(name) {
			continue
		}
		if fieldVal, ok := props.Field(name); ok {
			childType, childSamples := collectFields(fieldVal, includeSamples)
			types[name] = childType
			if includeSamples && !childSamples.empty() {
				if samples == nil {
					samples = map[string]*sampleField{}
				}
				samples[name] = childSamples
			}
		}
	}
	return types, samples
}

// sampleBucket returns the raw sample bucket for the named shape (e.g.
// "array", "string", "number") from a field's INFER schema value. INFER
// reports a field with a single shape as one flat "samples" list; a field
// with multiple shapes (e.g. "string or number or null") instead reports
// "samples" as one bucket per shape, in the same order as "type" -- so
// wantType's bucket is found at the matching index in typeNames.
func sampleBucket(schema value.Value, typeNames []string, wantType string) []interface{} {
	s, ok := schema.Field("samples")
	if !ok || s.Type() != value.ARRAY {
		return nil
	}
	arr, ok := s.Actual().([]interface{})
	if !ok {
		return nil
	}

	if len(typeNames) <= 1 {
		if len(typeNames) == 0 || typeNames[0] == wantType {
			return arr
		}
		return nil
	}
	for i, name := range typeNames {
		if name == wantType && i < len(arr) {
			bucket, _ := arr[i].([]interface{})
			return bucket
		}
	}
	return nil
}

// leafSamples collects a field's own example values, grouped by shape name,
// from typeNames' shapes. "null" is skipped (no useful literal value) and
// "object" is skipped because an object's own fields already carry their own
// samples via Properties (see collectProperties). "array" is included: INFER
// never attaches samples inside "items" itself -- only whole example array
// values on the array field -- so those whole instances become this field's
// own "array" sample bucket rather than being decomposed into item fields.
// capSampleBucket bounds their size since, unlike a scalar sample, a
// whole-instance array sample is copied verbatim from a real document and can
// be arbitrarily large.
func leafSamples(schema value.Value, typeNames []string) map[string][]interface{} {
	var result map[string][]interface{}
	for _, name := range typeNames {
		if name == "null" || name == "object" {
			continue
		}
		if bucket := sampleBucket(schema, typeNames, name); len(bucket) > 0 {
			if result == nil {
				result = map[string][]interface{}{}
			}
			result[name] = capSampleBucket(bucket)
		}
	}
	return result
}

// collectFields recursively builds a field's type tree and, when
// includeSamples is true, its samples tree, from a single field's INFER
// schema value. Both trees come from the same traversal of schema -- each
// node is visited once, regardless of whether one or two trees are being
// built from it. sampleNode is nil when includeSamples is false.
//
// A field's type may be more than one shape at once (e.g. "object or array"),
// so object handling, array handling, and leaf-sample collection all run
// independently below rather than as mutually exclusive switch cases --
// otherwise a field that is e.g. both string- and array-shaped would lose
// one description or the other.
func collectFields(schema value.Value, includeSamples bool) (typeNode *schemaField, sampleNode *sampleField) {
	typeNode = &schemaField{}
	if includeSamples {
		sampleNode = &sampleField{}
	}
	typeinfo, ok := schema.Field("type")
	if !ok {
		return typeNode, sampleNode
	}

	var typeNames []string
	if typeinfo.Type() == value.ARRAY {
		var typestring strings.Builder
		var typestr string

		if typestrslice, ok := typeinfo.Actual().([]interface{}); ok {
			for _, s := range typestrslice {
				if typestr, ok = s.(string); ok {
					typeNames = append(typeNames, typestr)
				}
			}
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

		typeNode.Type = typestring.String()
	} else if t, ok := typeinfo.Actual().(string); ok {
		typeNode.Type = t
		typeNames = []string{t}
	}

	if slices.Contains(typeNames, "object") {
		if props, ok := schema.Field("properties"); ok {
			typeProps, sampleProps := collectProperties(props, includeSamples, nil)
			typeNode.Properties = typeProps
			if includeSamples {
				sampleNode.Properties = sampleProps
			}
		}
	}

	if slices.Contains(typeNames, "array") {
		if items, ok := schema.Field("items"); ok {
			if items.Type() == value.ARRAY {
				itemArr, _ := items.Actual().([]interface{})
				for _, item := range itemArr {
					childType, childSamples := collectFields(value.NewValue(item), includeSamples)
					typeNode.Items = append(typeNode.Items, childType)
					if includeSamples {
						sampleNode.Items = append(sampleNode.Items, childSamples)
					}
				}
			} else {
				childType, childSamples := collectFields(items, includeSamples)
				typeNode.Items = append(typeNode.Items, childType)
				if includeSamples {
					sampleNode.Items = append(sampleNode.Items, childSamples)
				}
			}
			if includeSamples {
				allEmpty := true
				for _, it := range sampleNode.Items {
					if !it.empty() {
						allEmpty = false
						break
					}
				}
				if allEmpty {
					sampleNode.Items = nil
				}
			}
		}
	}

	if includeSamples {
		if scalar := leafSamples(schema, typeNames); scalar != nil {
			if sampleNode.Samples == nil {
				sampleNode.Samples = scalar
			} else {
				for k, v := range scalar {
					sampleNode.Samples[k] = v
				}
			}
		}
	}

	return typeNode, sampleNode
}

// collectSchemaFromInfer extracts, from a single keyspace's INFER result, a
// per-field type tree (rendered into the prompt schema and the persisted
// conversation) and, when includeSamples is set, a separate per-field samples
// tree. Both come from a single collectFields traversal per field, which
// builds the two trees in lockstep -- the type tree is a distinct type from
// the samples tree, so it can never carry sample values, and is always safe
// to persist. The returned samples map is nil when includeSamples is false or
// INFER reported no samples.
func collectSchemaFromInfer(schema map[string]*schemaField, inferSchema value.Value,
	includeSamples bool) (map[string]*schemaField, map[string]*sampleField) {

	v, ok := inferSchema.Index(0)
	if !ok {
		return schema, nil
	}
	prop, ok := v.Field("properties")
	if !ok {
		return schema, nil
	}

	types, samples := collectProperties(prop, includeSamples, func(name string) bool { return name == "~meta" })
	for name, node := range types {
		schema[name] = node
	}

	return schema, samples
}

func inferSchema(schema map[string]*schemaField, p *algebra.Path, context NaturalContext,
	includeSamples bool) (map[string]*schemaField, map[string]*sampleField, errors.Error) {

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

	var samples map[string]*sampleField
	if inferSchema != nil && ok {
		schema, samples = collectSchemaFromInfer(schema, inferSchema, includeSamples)
	}

	return schema, samples, nil
}

// vectorIndex is the minimal structural interface needed to read vector-index
// metadata off a datastore.Index. Kept narrower than datastore.Index6 so tests
// can fake it without implementing the full Index2-Index6 method stack.
type vectorIndex interface {
	Name() string
	RangeKey2() datastore.IndexKeys
	IsVector() bool
	VectorDistanceType() datastore.IndexDistanceType
	VectorDimension() int
	VectorProbes() int
}

func vectorFieldInfo(vi vectorIndex) map[string]interface{} {
	field := ""
	vtype := ""
	for _, k := range vi.RangeKey2() {
		if vtype = k.VectorType(); vtype != "" {
			if expr := k.Expression(); expr != nil {
				field = expr.String()
				break
			}
		}
	}
	if field == "" {
		return nil
	}
	info := map[string]interface{}{
		"field":      field,
		"type":       vtype,
		"similarity": string(vi.VectorDistanceType()),
		"probes":     vi.VectorProbes(),
		"indexName":  vi.Name(),
	}
	// Sparse vector indexes have no fixed dimension: each document only carries
	// whichever indices are non-zero for it, so the field is meaningless here.
	if vtype != datastore.IK_SPARSE_VECTOR_NAME {
		info["dimension"] = vi.VectorDimension()
	}
	return info
}

func collectVectorIndexes(indexes []datastore.Index) []map[string]interface{} {
	var rv []map[string]interface{}
	for _, idx := range indexes {
		vi, ok := idx.(vectorIndex)
		if !ok || !vi.IsVector() {
			continue
		}
		if info := vectorFieldInfo(vi); info != nil {
			rv = append(rv, info)
		}
	}
	return rv
}

func vectorIndexesForPath(p *algebra.Path) []map[string]interface{} {
	keyspace, err := datastore.GetKeyspace(p.Parts()...)
	if err != nil || keyspace == nil {
		return nil
	}
	indexer, err := keyspace.Indexer(datastore.GSI)
	if err != nil || indexer == nil {
		return nil
	}
	indexes, err := indexer.Indexes()
	if err != nil {
		return nil
	}
	return collectVectorIndexes(indexes)
}

func keyspacesInfoForPrompt(keyspaceInfo map[string]interface{}, elems []*algebra.Path,
	context NaturalContext, includeSamples bool,
	useKnowledge bool, naturalPrompt string) (map[string]interface{},
	map[string]map[string]*sampleField, errors.Error) {

	var err errors.Error
	var samplesByKeyspace map[string]map[string]*sampleField
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
		schema := map[string]*schemaField{}
		var ksSamples map[string]*sampleField
		schema, ksSamples, err = inferSchema(schema, p, context, includeSamples)

		if err != nil {
			return nil, nil, errors.NewNaturalLanguageRequestError(errors.E_NL_PROMPT_INFER, p.ProtectedString(), err)
		}
		info := map[string]interface{}{}
		info["schema"] = schema
		fullpath := p.ProtectedString()
		info["fullpath"] = fullpath[strings.Index(fullpath, ":"):]

		if vecIdx := vectorIndexesForPath(p); len(vecIdx) > 0 {
			info["vectorIndexes"] = vecIdx
		}
		if useKnowledge {
			// Knowledge is an enrichment, not a requirement for building a valid prompt (unlike
			// schema, above): a failure here shouldn't fail an otherwise-successful NL query, so
			// log and continue without knowledge for this keyspace rather than aborting.
			knowledge, kerr := Injector.Inject(context, p, naturalPrompt)
			if kerr != nil {
				logging.Errorf("USING AI AND KNOWLEDGE [%s]: failed to gather knowledge for %s: %v",
					context.RequestId(), p.ProtectedString(), kerr)
			} else if knowledge != "" {
				info["knowledge"] = knowledge
			}
		}

		keyspaceInfo[p.Keyspace()] = info
		if len(ksSamples) > 0 {
			if samplesByKeyspace == nil {
				samplesByKeyspace = map[string]map[string]*sampleField{}
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
	userMessageBuf.WriteString("Information about keyspaces: a field's nested object fields appear under " +
		"\"properties\" and an array's element type appears under \"items\" -- a field can have both if it " +
		"can be more than one shape.\n\n")
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
	userMessageBuf.WriteString(_AMBIGUOUS_TERM_INSTRUCTION)
	userMessageBuf.WriteString(vectorSearchInstructions(false))
	userMessageBuf.WriteString(_KNOWLEDGE_INSTRUCTION)
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

func appendJSUDFUserMessage(rv *prompt, keyspaceInfo map[string]interface{},
	naturalPrompt, summary, hint string) errors.Error {

	var userMessage string
	var userMessageBuf strings.Builder
	binKeyspacesInfo, err := json.Marshal(keyspaceInfo)
	if err != nil {
		return errors.NewNaturalLanguageRequestError(errors.E_NL_PROMPT_SCHEMA_MARSHAL, err)
	}
	if summary != "" {
		userMessageBuf.WriteString("Summary of the conversation so far: " + summary + "\n\n")
	}
	userMessageBuf.WriteString("Information about keyspaces: a field's nested object fields appear under " +
		"\"properties\" and an array's element type appears under \"items\" -- a field can have both if it " +
		"can be more than one shape.\n\n")
	userMessageBuf.WriteString(string(binKeyspacesInfo))
	userMessageBuf.WriteString("\n\nPrompt: \"")
	userMessageBuf.WriteString(naturalPrompt)
	userMessageBuf.WriteString("\"")
	if hint != "" {
		userMessageBuf.WriteString("\n\nHint: \"")
		userMessageBuf.WriteString(hint)
		userMessageBuf.WriteString("\"")
	}
	userMessageBuf.WriteString("\n\nBased on the above Information, write a valid Javascript User Defined Function with" +
		" no explanation that implements the request in the Prompt." +
		"\n\nComment the code liberally to explain what each piece does and why it's written that way." +
		"\nAlways comment using multiline comment syntax,i.e /* ... */ as you have construct a SQL++ managed user defined function" +
		" in a single line." +
		"\n\nPrompt may request for something trivial such as string manipulation, mathematical calculations, etc." +
		" Which are not related to the information provided but must always instruct to generate a function." +
		" \n\n Your task is to return CREATE FUNCTION statement that follows construct for SQL++ managed user defined function." +
		"As Capella does not currently support a way to create or manage an external library" +
		"\n\nExamples:" +
		"\n\nExample1) shows an example for a request that doesn't use the information provided, example prompt:" +
		"add 2 numbers. Statement to create a function for the request would be: CREATE FUNCTION add(a,b) LANGUAGE JAVASCRIPT AS" +
		" 'function add(a,b) { return(a+b);}'" +
		"\n\nExample2) shows an example for a request that uses the information provided, example prompt:" +
		"select airlines given country as an argument. Statement to create a function for the request would be: CREATE FUNCTION" +
		" selectAirline(country) LANGUAGE JAVASCRIPT AS 'function selectAirline(country)" +
		" {var q = SELECT name as airline_name, callsign as airline_callsign FROM `travel-sample`.`inventory`.`airline` " +
		"WHERE country = $country; var res = []; for (const doc of q) { var airline = {}; airline.name = doc.airline_name;" +
		"airline.callsign = doc.airline_callsign; res.push(airline);} return res;}" +
		"\n\nNote query context is unset." +
		"\n\nUse the fullpath from the information about keyspaces for retrieval along with an alias." +
		"\n\nAlias is for ease of use." +
		"\n\nQuote aliases with grave accent characters.")
	userMessageBuf.WriteString(_AMBIGUOUS_TERM_INSTRUCTION)
	userMessageBuf.WriteString(vectorSearchInstructions(true))
	userMessageBuf.WriteString(_KNOWLEDGE_INSTRUCTION)
	userMessageBuf.WriteString("\n\nReturn only a single CREATE FUNCTION statement on a single line." +
		"\n\nIf you're sure the Prompt can't be used to generate a function, say " +
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

func appendJSUDFIterativeUserMessage(chat *prompt, naturalPrompt string, hint string) *prompt {
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
	userMessageBuf.WriteString("\"\n\nBased on the above Information, write a valid Javascript User Defined Function with" +
		" no explanation that implements the request in the Prompt." +
		"\n\nComment the code liberally to explain what each piece does and why it's written that way." +
		"\nAlways comment using multiline comment syntax,i.e /* ... */ as you have construct a SQL++ managed user defined function" +
		" in a single line." +
		"\n\nPrompt may request for something trivial such as string manipulation, mathematical calculations, etc." +
		" Which are not related to the information provided but must always instruct to generate a function." +
		" \n\n Your task is to return CREATE FUNCTION statement that follows construct for SQL++ managed user defined function." +
		"As Capella does not currently support a way to create or manage an external library" +
		"\n\nExamples:" +
		"\n\nExample1) shows an example for a request that doesn't use the information provided, example prompt:" +
		"add 2 numbers. Statement to create a function for the request would be: CREATE FUNCTION add(a,b) LANGUAGE JAVASCRIPT AS" +
		" 'function add(a,b) { return(a+b);}'" +
		"\n\nExample2) shows an example for a request that uses the information provided, example prompt:" +
		"select airlines given country as an argument. Statement to create a function for the request would be: CREATE FUNCTION" +
		" selectAirline(country) LANGUAGE JAVASCRIPT AS 'function selectAirline(country)" +
		" {var q = SELECT name as airline_name, callsign as airline_callsign FROM `travel-sample`.`inventory`.`airline` " +
		"WHERE country = $country; var res = []; for (const doc of q) { var airline = {}; airline.name = doc.airline_name;" +
		"airline.callsign = doc.airline_callsign; res.push(airline);} return res;}" +
		"\n\nNote query context is unset." +
		"\n\nUse the fullpath from the information about keyspaces for retrieval along with an alias." +
		"\n\nAlias is for ease of use." +
		"\n\nQuote aliases with grave accent characters." +
		"\n\nIf the previous message was not a CREATE FUNCTION statement, use the previous messages to for a CREATE FUNCTION statement." +
		"\nReturn only a single CREATE FUNCTION statement on a single line." +
		"\n\nIf you're sure the Prompt can't be used to generate a function, say " +
		"\n#ERR:\" and then explain why not without prefix.\n\n")

	chat.Size += userMessageBuf.Len()
	userMessage = userMessageBuf.String()
	chat.Messages = append(chat.Messages, message{
		Content: userMessage,
		Role:    "user",
	})

	return chat
}

// Reusable instruction snippets, shared across prompt builders.

const _KNOWLEDGE_INSTRUCTION = "\n\nIf a keyspace's information above includes a \"knowledge\" entry, apply " +
	"it the same way as a Hint, except that it is persistent, keyspace-level context rather than something " +
	"supplied with this particular request. Prefer it over your own assumptions if it conflicts with them."

const _AMBIGUOUS_TERM_INSTRUCTION = "\n\nIf a value needed to complete the query " +
	"(for example a numeric threshold, date, or category) is not stated in the Prompt and " +
	"cannot be inferred from the provided schema or the Hint, do not invent or guess a value." +
	"\n\nInstead, prefer writing that value as a named parameter placeholder in the statement, " +
	"for example `price < $price` or `created_at > $cutoff_date`, using a short, descriptive " +
	"name for the placeholder." +
	"\n\nIf that isn't possible and you have to say #ERR because of this, clearly state what " +
	"could not be inferred or what is ambiguous, and suggest the user either rephrase their " +
	"request with the missing detail or supply it using the natural_hint option."

const _AMBIGUOUS_VECTOR_FIELD_INSTRUCTION = "\n\nIf more than one candidate vector field exists -- more than " +
	"one \"vectorIndexes\" entry, or, when there is no vector index, more than one schema field matching " +
	"the dense or sparse shapes described above -- and the Prompt does not identify which field to search " +
	"by name, do not guess which one to use. Instead, say #ERR: and ask the user which field to search, " +
	"listing the candidate field names."

// vectorSearchInstructions returns the instruction paragraph teaching the LLM to use
// APPROX_VECTOR_DISTANCE/SPARSE_VECTOR_DISTANCE for similarity/nearest-neighbor requests.
// Shared by newSQLPrompt and newJSUDFPrompt; not used by the iterative variants, since a
// conversation's first message already carries these instructions and the full history
// is resent every turn.
//
// Sparse vector search is suggested either off an explicit vectorIndexes entry, or, absent
// one, off the schema's own nested "items" shape (see schemaField/collectFields, MB-72779):
// a field typed "array" whose "items" are themselves array-shaped with numeric leaves is the
// [indices, values] pair shape SPARSE_VECTOR_DISTANCE expects, distinguishable from a plain
// dense vector field, which is a flat array of numbers (no nested array items).
func vectorSearchInstructions(forJSUDF bool) string {
	var b strings.Builder
	if forJSUDF {
		b.WriteString("\n\nIf the Prompt is asking for documents similar to something (nearest neighbor" +
			" or semantic/similarity search), the SQL++ query inside the generated function should use" +
			" ORDER BY APPROX_VECTOR_DISTANCE(field, $qvec, metric, nprobes) LIMIT k, where qvec is a" +
			" parameter of the generated function, referenced as $qvec inside the query the same way" +
			" other function arguments are referenced (see the country/$country example above).")
	} else {
		b.WriteString("\n\nIf the Prompt is asking for documents similar to something (nearest neighbor" +
			" or semantic/similarity search), express it using" +
			" ORDER BY APPROX_VECTOR_DISTANCE(field, $qvec, metric, nprobes) LIMIT k.")
	}
	b.WriteString("\n\nIf the keyspace information above includes a \"vectorIndexes\" entry, use that entry's \"field\"," +
		" \"similarity\", and \"probes\" values exactly when the vector \"field\" is relevant to the question." +
		"\n\nIf there is no relevant \"vectorIndexes\" entry, use the schema's nested \"items\" shape to pick" +
		" between a dense and a sparse vector field. A schema field with \"type\": \"array\" whose \"items\"" +
		" is directly \"number\" (a flat array of numbers) is a candidate dense vector field: default to the" +
		" 'cosine' similarity metric and 1 probe. A schema field with \"type\": \"array\" whose \"items\" are" +
		" themselves array-shaped with numeric leaves (an array of arrays, e.g. [[2,9,14],[0.8,0.6,0.3]]) is" +
		" a candidate sparse vector field.")
	b.WriteString(_AMBIGUOUS_VECTOR_FIELD_INSTRUCTION)
	b.WriteString("\n\nIf the relevant vector field is sparse -- either because its \"vectorIndexes\" entry has" +
		" \"type\": \"sparse\", or, when there is no vector index, because the schema shows the array-of-arrays" +
		" shape above -- APPROX_VECTOR_DISTANCE does not apply to it. Use SPARSE_VECTOR_DISTANCE(field, $qvec, nprobes)" +
		" instead: it takes no similarity metric argument (it always computes a negated dot product, so" +
		" ORDER BY ascending still returns the closest matches first) and $qvec must be a pair of parallel" +
		" arrays [indices, values], e.g. [[2,9,14],[0.85,0.4,0.3]], never a flat array of numbers.")
	if forJSUDF {
		b.WriteString("\n\nThe query vector must always come from a function parameter, never a literal array" +
			" hard-coded in the query, unless the Prompt itself supplies literal numbers to search for.")
	} else {
		b.WriteString("\n\nThe query vector argument to APPROX_VECTOR_DISTANCE or SPARSE_VECTOR_DISTANCE must" +
			" always be a query parameter such as $qvec or $1, never a literal array, unless the Prompt itself" +
			" supplies literal numbers to search for. This also applies when the Prompt asks for documents" +
			" similar to a previous result: reference a query parameter for the vector, never attempt to look" +
			" up or invent a previous document's data.")
	}
	return b.String()
}

// End of reusable instruction snippets.

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

func CanServerExecuteGeneratedStatement(nlAlgebraStmt algebra.Statement) bool {

	switch stmttype := nlAlgebraStmt.Type(); stmttype {
	case "ADVISE", "EXPLAIN":
		return true
	case "SELECT":
		if nlAlgebraStmt.ParamsCount() > 0 {
			return false
		}
		return true
	default:
		return false
	}
}
