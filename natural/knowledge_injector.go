//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package natural

import (
	"strings"

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/natural/knowledge"
)

// KnowledgeInjector supplies extra natural-language context for a keyspace, gathered from
// CREATE KNOWLEDGE entries, to fold into the USING AI prompt. Inject is called once per
// keyspace referenced by a USING AI AND KNOWLEDGE request (or a request carrying
// "knowledge":true in its WITH clause).
//
// The naturalPrompt is passed through so a future, more selective implementation (e.g. one
// backed by vector search) can rank or filter entries by relevance to the prompt; the basic
// implementation below ignores it and returns every entry for the keyspace.
type KnowledgeInjector interface {
	// Inject returns the knowledge text for keyspace p, or "" if none is found.
	Inject(context NaturalContext, p *algebra.Path, naturalPrompt string) (string, errors.Error)
}

// Injector is the active KnowledgeInjector, swappable at init time by builds that want a
// more selective implementation. Defaults to basicInjector.
var Injector KnowledgeInjector = basicInjector{}

// basicInjector gathers every knowledge entry stored for the keyspace, unfiltered, by calling
// directly into the natural/knowledge package - the caller (keyspacesInfoForPrompt) has already
// authorized the keyspace itself, so no further privilege check is needed here.
type basicInjector struct{}

func (basicInjector) Inject(context NaturalContext, p *algebra.Path, naturalPrompt string) (string, errors.Error) {
	entries, err := knowledge.FetchAll(p)
	if err != nil {
		return "", errors.NewNaturalLanguageRequestError(errors.E_NL_CONTEXT, err)
	}
	if len(entries) == 0 {
		return "", nil
	}

	var sb strings.Builder
	first := true
	for _, val := range entries {
		if !first {
			sb.WriteString("\n")
		}
		first = false
		sb.WriteString(val)
	}
	return sb.String(), nil
}
