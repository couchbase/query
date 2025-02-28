//  Copyright 2024-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/finderr/platform"
)

func title(t string) {
	fmt.Printf("\n\033[1m%s\033[0m\n", t)
}

func fitWidth(s string, width int) string {
	n := 0
	b := 0
	p := 0
	for i, r := range s {
		if r == '\n' {
			fmt.Printf(s[:i])
			if i+1 < len(s) {
				return s[i+1:]
			}
			return ""
		}
		p = i + utf8.RuneLen(r)
		if unicode.IsSpace(r) {
			b = p
		}
		n++
		if n == width {
			break
		}
	}
	if p >= len(s) {
		b = len(s)
	} else if b == 0 {
		b = p
	}
	var builder strings.Builder
	builder.WriteString(s[:b])
	fmt.Print(builder.String())
	if len(s) > b {
		return s[b:]
	}
	return ""
}

func printWidth(width int, margin int, what string) {
	if margin < 0 {
		what = fitWidth(what, width)
		margin *= -1
		fmt.Printf("\n")
	}
	marginSpaces := strings.Repeat(" ", margin)
	for what != "" {
		if margin > 0 {
			fmt.Print(marginSpaces)
		}
		what = fitWidth(what, width-margin)
		fmt.Printf("\n")
	}
}

func main() {
	if len(os.Args) != 2 {
		fmt.Printf("USAGE: %s <code-or-pattern>\n\n"+
			"Display Couchbase Query service SQL++ related error information.\n\n"+
			"<code-or-pattern>\n"+
			"  If numeric, then the error number to display information for.\n"+
			"  If non-numeric, then the pattern to match against error information.\n\n",
			os.Args[0])
		return
	}

	width := platform.InitTerm()

	var errData *errors.ErrData
	code, err := strconv.Atoi(os.Args[1])
	if err != nil {
		errs := errors.SearchErrors(os.Args[1])
		if len(errs) == 0 {
			fmt.Println("No matching errors.")
			return
		} else if len(errs) == 1 {
			errData = errs[0]
		} else {
			title("Matching errors")
			for i := range errs {
				printWidth(width, -7, fmt.Sprintf("%6d %s", errs[i].Code, errs[i].Description))
			}
			fmt.Printf("\n")
			return
		}
	} else {
		errData = errors.DescribeError(errors.ErrorCode(code))
	}
	if errData == nil {
		fmt.Printf("Unable to find information for error code %d.\n", code)
		return
	}

	margin := 4

	title("CODE")
	s := fmt.Sprintf("%v ", errData.Code)
	if errData.IsWarning {
		s += "(warning)"
	} else {
		s += "(error)"
	}
	printWidth(width, margin, s)

	title("DESCRIPTION")
	printWidth(width, margin, fmt.Sprintf("%v", errData.Description))

	if len(errData.Reason) > 0 {
		title("REASON")
		for i := range errData.Reason {
			if i > 0 {
				fmt.Printf("\n")
			}
			printWidth(width, margin, errData.Reason[i])
		}
	}

	if len(errData.Action) > 0 {
		title("USER ACTION")
		for i := range errData.Action {
			if i > 0 {
				fmt.Printf("\n")
			}
			printWidth(width, margin, errData.Action[i])
		}
	}

	title("USER ERROR")
	switch errData.IsUser {
	case errors.YES:
		printWidth(width, margin, "Yes")
	case errors.MAYBE:
		printWidth(width, margin, "Possibly")
	default:
		printWidth(width, margin, "No")
	}

	title("APPLIES TO")
	printWidth(width, margin, strings.Join(errData.AppliesTo, ", "))

	fmt.Printf("\n")
}
