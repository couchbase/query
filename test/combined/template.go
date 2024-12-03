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
	"io"
	"math/rand"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
	"text/template"

	"github.com/couchbase/query/logging"
)

type Template struct {
	tpl        *template.Template
	iterations int
}

type TemplateData struct {
	Keyspaces       []string
	RandomKeyspaces []string
	Iteration       int
}

// loads all .tpl files found under the directory and recurses into any sub-directories
func LoadTemplates(dir string) ([]*Template, error) {
	logging.Debugf("%s", dir)
	d, err := os.Open(dir)
	if err != nil {
		return nil, fmt.Errorf("Failed to open directory: %s - %v", dir, err)
	}
	var res []*Template
	for {
		ents, err := d.ReadDir(10)
		if err == nil {
			for i := range ents {
				if ents[i].IsDir() {
					if ents[i].Name() != "." && ents[i].Name() != ".." {
						if qrys, err := LoadTemplates(path.Join(dir, ents[i].Name())); err != nil {
							return nil, err
						} else {
							res = append(res, qrys...)
						}
					}
				} else if strings.HasSuffix(ents[i].Name(), ".tpl") {
					t, err := LoadTemplate(path.Join(dir, ents[i].Name()))
					if err != nil {
						return nil, fmt.Errorf("Failed to load template from %s: %v", path.Join(dir, ents[i].Name()), err)
					} else if t != nil {
						res = append(res, t)
					}
				}
			}
		}
		if err != nil || len(ents) < 10 {
			break
		}
	}
	return res, nil
}

// functions available in the template must be defined before parsing
var templateFuncMap = template.FuncMap{
	"JoinStrings":  strings.Join,
	"GetJoinOn":    func(ks1 string, a1 string, ks2 string, a2 string) string { return "stub" },
	"RandomFields": func(ks string, num int) []string { return []string{"stub"} },
	"RandomFilter": func(ks string) string { return "stub" },
	"GetValue":     func(ks string, field string) string { return "stub" },
}
var iterationsPattern = regexp.MustCompile("{{-* */\\* *iterations=([0-9]+) *\\*/ *-*}}")

func LoadTemplate(file string) (*Template, error) {
	logging.Debugf("%s", file)
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	b, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}
	f.Close()
	tpl := &Template{iterations: 1}
	tpl.tpl, err = template.New(file).Funcs(templateFuncMap).Parse(string(b))
	if err != nil {
		return nil, err
	}
	if m := iterationsPattern.FindStringSubmatch(string(b)); len(m) == 2 {
		tpl.iterations, err = strconv.Atoi(m[1])
		if err != nil {
			return nil, fmt.Errorf("Invalid iterations comment: %v", err)
		}
	}
	if tpl.iterations < 0 {
		tpl.iterations = rand.Intn(tpl.iterations * -1)
	}
	if tpl.iterations == 0 {
		return nil, nil
	}
	return tpl, nil
}

// ---------------------------------------------------------------------------------------------------------------------------------
