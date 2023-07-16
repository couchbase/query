//  Copyright 2023-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package command

var statement_syntax = map[string][][]string{
	"ident_or_default": [][]string{
		[]string{"IDENT"},
		[]string{"DEFAULT"},
	},
	"opt_trailer": [][]string{
		[]string{"%empty"},
		[]string{"opt_trailer", "SEMI"},
	},
	"statements": [][]string{
		[]string{"advise"},
		[]string{"explain"},
		[]string{"prepare"},
		[]string{"execute"},
		[]string{"explain_function"},
		[]string{"statement"},
	},
	"statement": [][]string{
		[]string{"select_statement"},
		[]string{"dml_statement"},
		[]string{"ddl_statement"},
		[]string{"infer"},
		[]string{"update_statistics"},
		[]string{"role_statement"},
		[]string{"function_statement"},
		[]string{"transaction_statement"},
	},
	"advise": [][]string{
		[]string{"ADVISE", "opt_index", "statement"},
	},
	"opt_index": [][]string{
		[]string{"%empty"},
		[]string{"INDEX"},
	},
	"explain": [][]string{
		[]string{"EXPLAIN", "statement"},
	},
	"explain_function": [][]string{
		[]string{"EXPLAIN", "FUNCTION", "func_name"},
	},
	"prepare": [][]string{
		[]string{"PREPARE", "opt_force", "opt_name", "statement"},
	},
	"opt_force": [][]string{
		[]string{"%empty"},
		[]string{"FORCE"},
	},
	"opt_name": [][]string{
		[]string{"%empty"},
		[]string{"ident_or_default", "from_or_as"},
		[]string{"IDENT_ICASE", "from_or_as"},
		[]string{"STR", "from_or_as"},
	},
	"from_or_as": [][]string{
		[]string{"FROM"},
		[]string{"AS"},
	},
	"execute": [][]string{
		[]string{"EXECUTE", "expr", "execute_using"},
	},
	"execute_using": [][]string{
		[]string{"%empty"},
		[]string{"USING", "construction_expr"},
	},
	"infer": [][]string{
		[]string{"INFER", "keyspace_collection", "simple_keyspace_ref", "opt_infer_using", "opt_with_clause"},
		[]string{"INFER", "keyspace_path", "opt_as_alias", "opt_infer_using", "opt_with_clause"},
		[]string{"INFER", "expr", "opt_infer_using", "opt_with_clause"},
	},
	"keyspace_collection": [][]string{
		[]string{"KEYSPACE"},
		[]string{"COLLECTION"},
	},
	"opt_keyspace_collection": [][]string{
		[]string{"%empty"},
		[]string{"keyspace_collection"},
	},
	"opt_infer_using": [][]string{
		[]string{"%empty"},
	},
	"select_statement": [][]string{
		[]string{"fullselect"},
	},
	"dml_statement": [][]string{
		[]string{"insert"},
		[]string{"upsert"},
		[]string{"delete"},
		[]string{"update"},
		[]string{"merge"},
	},
	"ddl_statement": [][]string{
		[]string{"index_statement"},
		[]string{"scope_statement"},
		[]string{"collection_statement"},
	},
	"role_statement": [][]string{
		[]string{"grant_role"},
		[]string{"revoke_role"},
	},
	"index_statement": [][]string{
		[]string{"create_index"},
		[]string{"drop_index"},
		[]string{"alter_index"},
		[]string{"build_index"},
	},
	"scope_statement": [][]string{
		[]string{"create_scope"},
		[]string{"drop_scope"},
	},
	"collection_statement": [][]string{
		[]string{"create_collection"},
		[]string{"drop_collection"},
		[]string{"flush_collection"},
	},
	"function_statement": [][]string{
		[]string{"create_function"},
		[]string{"drop_function"},
		[]string{"execute_function"},
	},
	"transaction_statement": [][]string{
		[]string{"start_transaction"},
		[]string{"commit_transaction"},
		[]string{"rollback_transaction"},
		[]string{"savepoint"},
		[]string{"set_transaction_isolation"},
	},
	"fullselect": [][]string{
		[]string{"select_terms", "opt_order_by"},
		[]string{"select_terms", "opt_order_by", "limit", "opt_offset"},
		[]string{"select_terms", "opt_order_by", "offset", "opt_limit"},
		[]string{"with", "select_terms", "opt_order_by"},
		[]string{"with", "select_terms", "opt_order_by", "limit", "opt_offset"},
		[]string{"with", "select_terms", "opt_order_by", "offset", "opt_limit"},
	},
	"select_terms": [][]string{
		[]string{"subselect"},
		[]string{"select_terms", "setop", "select_term"},
		[]string{"subquery_expr", "setop", "select_term"},
	},
	"select_term": [][]string{
		[]string{"subselect"},
		[]string{"subquery_expr"},
	},
	"subselect": [][]string{
		[]string{"from_select"},
		[]string{"select_from"},
	},
	"from_select": [][]string{
		[]string{"from", "opt_let", "opt_where", "opt_group", "opt_window_clause", "SELECT", "opt_optim_hints", "projection"},
	},
	"select_from": [][]string{
		[]string{"SELECT", "opt_optim_hints", "projection", "opt_from", "opt_let", "opt_where", "opt_group", "opt_window_clause"},
	},
	"setop": [][]string{
		[]string{"UNION"},
		[]string{"UNION", "ALL"},
		[]string{"INTERSECT"},
		[]string{"INTERSECT", "ALL"},
		[]string{"EXCEPT"},
		[]string{"EXCEPT", "ALL"},
	},
	"opt_optim_hints": [][]string{
		[]string{"%empty"},
		[]string{"OPTIM_HINTS"},
		[]string{"PLUS", "object"},
	},
	"optim_hints": [][]string{
		[]string{"optim_hint"},
		[]string{"optim_hints", "optim_hint"},
	},
	"optim_hint": [][]string{
		[]string{"ident_or_default"},
		[]string{"ident_or_default", "LPAREN", "opt_hint_args", "RPAREN"},
		[]string{"INDEX", "LPAREN", "opt_hint_args", "RPAREN"},
	},
	"opt_hint_args": [][]string{
		[]string{"%empty"},
		[]string{"hint_args"},
	},
	"hint_args": [][]string{
		[]string{"ident_or_default"},
		[]string{"ident_or_default", "DIV", "BUILD"},
		[]string{"ident_or_default", "DIV", "PROBE"},
		[]string{"hint_args", "ident_or_default"},
	},
	"projection": [][]string{
		[]string{"opt_quantifier", "projects", "opt_exclude"},
		[]string{"opt_quantifier", "raw", "expr", "opt_as_alias"},
	},
	"opt_quantifier": [][]string{
		[]string{"%empty"},
		[]string{"ALL"},
		[]string{"DISTINCT"},
	},
	"opt_exclude": [][]string{
		[]string{"%empty"},
		[]string{"EXCLUDE", "exprs"},
	},
	"raw": [][]string{
		[]string{"RAW"},
		[]string{"ELEMENT"},
		[]string{"VALUE"},
	},
	"projects": [][]string{
		[]string{"project"},
		[]string{"projects", "COMMA", "project"},
	},
	"project": [][]string{
		[]string{"STAR"},
		[]string{"expr", "DOT", "STAR"},
		[]string{"expr", "opt_as_alias"},
	},
	"opt_as_alias": [][]string{
		[]string{"%empty"},
		[]string{"as_alias"},
	},
	"as_alias": [][]string{
		[]string{"alias"},
		[]string{"AS", "alias"},
	},
	"alias": [][]string{
		[]string{"ident_or_default"},
	},
	"opt_from": [][]string{
		[]string{"%empty"},
		[]string{"from"},
	},
	"from": [][]string{
		[]string{"FROM", "from_terms"},
	},
	"from_terms": [][]string{
		[]string{"from_term"},
		[]string{"from_terms", "COMMA", "from_term"},
		[]string{"from_terms", "COMMA", "LATERAL", "from_term"},
	},
	"from_term": [][]string{
		[]string{"simple_from_term"},
		[]string{"from_term", "opt_join_type", "JOIN", "simple_from_term", "on_keys"},
		[]string{"from_term", "opt_join_type", "JOIN", "LATERAL", "simple_from_term", "on_keys"},
		[]string{"from_term", "opt_join_type", "JOIN", "simple_from_term", "on_key", "FOR", "ident_or_default"},
		[]string{"from_term", "opt_join_type", "JOIN", "LATERAL", "simple_from_term", "on_key", "FOR", "ident_or_default"},
		[]string{"from_term", "opt_join_type", "NEST", "simple_from_term", "on_keys"},
		[]string{"from_term", "opt_join_type", "NEST", "LATERAL", "simple_from_term", "on_keys"},
		[]string{"from_term", "opt_join_type", "NEST", "simple_from_term", "on_key", "FOR", "ident_or_default"},
		[]string{"from_term", "opt_join_type", "NEST", "LATERAL", "simple_from_term", "on_key", "FOR", "ident_or_default"},
		[]string{"from_term", "opt_join_type", "unnest", "expr", "opt_as_alias"},
		[]string{"from_term", "opt_join_type", "JOIN", "simple_from_term", "ON", "expr"},
		[]string{"from_term", "opt_join_type", "JOIN", "LATERAL", "simple_from_term", "ON", "expr"},
		[]string{"from_term", "opt_join_type", "NEST", "simple_from_term", "ON", "expr"},
		[]string{"from_term", "opt_join_type", "NEST", "LATERAL", "simple_from_term", "ON", "expr"},
		[]string{"simple_from_term", "RIGHT", "opt_outer", "JOIN", "simple_from_term", "ON", "expr"},
		[]string{"simple_from_term", "RIGHT", "opt_outer", "JOIN", "LATERAL", "simple_from_term", "ON", "expr"},
	},
	"simple_from_term": [][]string{
		[]string{"keyspace_term"},
		[]string{"expr", "opt_as_alias", "opt_use"},
	},
	"unnest": [][]string{
		[]string{"UNNEST"},
		[]string{"FLATTEN"},
	},
	"keyspace_term": [][]string{
		[]string{"keyspace_path", "opt_as_alias", "opt_use"},
	},
	"keyspace_path": [][]string{
		[]string{"namespace_term", "keyspace_name"},
		[]string{"namespace_term", "path_part", "DOT", "path_part", "DOT", "keyspace_name"},
	},
	"namespace_term": [][]string{
		[]string{"namespace_name"},
		[]string{"SYSTEM", "COLON"},
	},
	"namespace_name": [][]string{
		[]string{"NAMESPACE_ID", "COLON"},
	},
	"path_part": [][]string{
		[]string{"ident_or_default"},
	},
	"keyspace_name": [][]string{
		[]string{"ident_or_default"},
		[]string{"IDENT_ICASE"},
	},
	"opt_use": [][]string{
		[]string{"%empty"},
		[]string{"USE", "use_options"},
	},
	"use_options": [][]string{
		[]string{"use_keys"},
		[]string{"use_index"},
		[]string{"join_hint"},
		[]string{"use_index", "join_hint"},
		[]string{"join_hint", "use_index"},
		[]string{"use_keys", "join_hint"},
		[]string{"join_hint", "use_keys"},
	},
	"use_keys": [][]string{
		[]string{"opt_primary", "KEYS", "expr"},
		[]string{"opt_primary", "KEYS", "VALIDATE", "expr"},
	},
	"use_index": [][]string{
		[]string{"INDEX", "LPAREN", "index_refs", "RPAREN"},
	},
	"join_hint": [][]string{
		[]string{"HASH", "LPAREN", "use_hash_option", "RPAREN"},
		[]string{"NL"},
	},
	"opt_primary": [][]string{
		[]string{"%empty"},
		[]string{"PRIMARY"},
	},
	"index_refs": [][]string{
		[]string{"index_ref"},
		[]string{"index_refs", "COMMA", "index_ref"},
	},
	"index_ref": [][]string{
		[]string{"opt_index_name", "opt_index_using"},
	},
	"use_hash_option": [][]string{
		[]string{"BUILD"},
		[]string{"PROBE"},
	},
	"opt_use_del_upd": [][]string{
		[]string{"opt_use"},
	},
	"opt_join_type": [][]string{
		[]string{"%empty"},
		[]string{"INNER"},
		[]string{"LEFT", "opt_outer"},
	},
	"opt_outer": [][]string{
		[]string{"%empty"},
		[]string{"OUTER"},
	},
	"on_keys": [][]string{
		[]string{"ON", "opt_primary", "KEYS", "expr"},
		[]string{"ON", "opt_primary", "KEYS", "VALIDATE", "expr"},
	},
	"on_key": [][]string{
		[]string{"ON", "opt_primary", "KEY", "expr"},
		[]string{"ON", "opt_primary", "KEY", "VALIDATE", "expr"},
	},
	"opt_let": [][]string{
		[]string{"%empty"},
		[]string{"let"},
	},
	"let": [][]string{
		[]string{"LET", "bindings"},
	},
	"bindings": [][]string{
		[]string{"binding"},
		[]string{"bindings", "COMMA", "binding"},
	},
	"binding": [][]string{
		[]string{"alias", "EQ", "expr"},
	},
	"with": [][]string{
		[]string{"WITH", "with_list"},
	},
	"with_list": [][]string{
		[]string{"with_term"},
		[]string{"with_list", "COMMA", "with_term"},
	},
	"with_term": [][]string{
		[]string{"alias", "AS", "paren_expr"},
	},
	"opt_where": [][]string{
		[]string{"%empty"},
		[]string{"where"},
	},
	"where": [][]string{
		[]string{"WHERE", "expr"},
	},
	"opt_group": [][]string{
		[]string{"%empty"},
		[]string{"group"},
	},
	"group": [][]string{
		[]string{"GROUP", "BY", "group_terms", "opt_group_as", "opt_letting", "opt_having"},
		[]string{"letting"},
	},
	"group_terms": [][]string{
		[]string{"group_term"},
		[]string{"group_terms", "COMMA", "group_term"},
	},
	"group_term": [][]string{
		[]string{"expr", "opt_as_alias"},
	},
	"opt_letting": [][]string{
		[]string{"%empty"},
		[]string{"letting"},
	},
	"letting": [][]string{
		[]string{"LETTING", "bindings"},
	},
	"opt_having": [][]string{
		[]string{"%empty"},
		[]string{"having"},
	},
	"having": [][]string{
		[]string{"HAVING", "expr"},
	},
	"opt_group_as": [][]string{
		[]string{"%empty"},
		[]string{"GROUP", "AS", "ident_or_default"},
	},
	"opt_order_by": [][]string{
		[]string{"%empty"},
		[]string{"order_by"},
	},
	"order_by": [][]string{
		[]string{"ORDER", "BY", "sort_terms"},
	},
	"sort_terms": [][]string{
		[]string{"sort_term"},
		[]string{"sort_terms", "COMMA", "sort_term"},
	},
	"sort_term": [][]string{
		[]string{"expr", "opt_dir", "opt_order_nulls"},
	},
	"opt_dir": [][]string{
		[]string{"%empty"},
		[]string{"dir"},
	},
	"dir": [][]string{
		[]string{"param_expr"},
		[]string{"ASC"},
		[]string{"DESC"},
	},
	"opt_order_nulls": [][]string{
		[]string{"%empty"},
		[]string{"NULLS", "FIRST"},
		[]string{"NULLS", "LAST"},
		[]string{"NULLS", "param_expr"},
	},
	"first_last": [][]string{
		[]string{"FIRST"},
		[]string{"LAST"},
	},
	"opt_limit": [][]string{
		[]string{"%empty"},
		[]string{"limit"},
	},
	"limit": [][]string{
		[]string{"LIMIT", "expr"},
	},
	"opt_offset": [][]string{
		[]string{"%empty"},
		[]string{"offset"},
	},
	"offset": [][]string{
		[]string{"OFFSET", "expr"},
	},
	"insert": [][]string{
		[]string{"INSERT", "INTO", "keyspace_ref", "opt_values_header", "values_list", "opt_returning"},
		[]string{"INSERT", "INTO", "keyspace_ref", "LPAREN", "key_val_options_expr_header", "RPAREN", "fullselect", "opt_returning"},
	},
	"simple_keyspace_ref": [][]string{
		[]string{"keyspace_name", "opt_as_alias"},
		[]string{"path_part", "DOT", "path_part", "opt_as_alias"},
		[]string{"keyspace_path", "opt_as_alias"},
		[]string{"path_part", "DOT", "path_part", "DOT", "keyspace_name", "opt_as_alias"},
	},
	"keyspace_ref": [][]string{
		[]string{"simple_keyspace_ref"},
		[]string{"param_expr", "opt_as_alias"},
	},
	"opt_values_header": [][]string{
		[]string{"%empty"},
		[]string{"LPAREN", "opt_primary", "KEY", "COMMA", "VALUE", "RPAREN"},
		[]string{"LPAREN", "opt_primary", "KEY", "COMMA", "VALUE", "COMMA", "OPTIONS", "RPAREN"},
	},
	"key": [][]string{
		[]string{"opt_primary", "KEY"},
	},
	"values_list": [][]string{
		[]string{"values"},
		[]string{"values_list", "COMMA", "next_values"},
	},
	"values": [][]string{
		[]string{"VALUES", "key_val_expr"},
		[]string{"VALUES", "key_val_options_expr"},
	},
	"next_values": [][]string{
		[]string{"values"},
		[]string{"key_val_expr"},
		[]string{"key_val_options_expr"},
	},
	"key_val_expr": [][]string{
		[]string{"LPAREN", "expr", "COMMA", "expr", "RPAREN"},
	},
	"key_val_options_expr": [][]string{
		[]string{"LPAREN", "expr", "COMMA", "expr", "COMMA", "expr", "RPAREN"},
	},
	"opt_returning": [][]string{
		[]string{"%empty"},
		[]string{"returning"},
	},
	"returning": [][]string{
		[]string{"RETURNING", "returns"},
	},
	"returns": [][]string{
		[]string{"projects"},
		[]string{"raw", "expr"},
	},
	"key_expr_header": [][]string{
		[]string{"key", "expr"},
	},
	"value_expr_header": [][]string{
		[]string{"VALUE", "expr"},
	},
	"options_expr_header": [][]string{
		[]string{"OPTIONS", "expr"},
	},
	"key_val_options_expr_header": [][]string{
		[]string{"key_expr_header"},
		[]string{"key_expr_header", "COMMA", "value_expr_header"},
		[]string{"key_expr_header", "COMMA", "value_expr_header", "COMMA", "options_expr_header"},
		[]string{"key_expr_header", "COMMA", "options_expr_header"},
	},
	"upsert": [][]string{
		[]string{"UPSERT", "INTO", "keyspace_ref", "opt_values_header", "values_list", "opt_returning"},
		[]string{"UPSERT", "INTO", "keyspace_ref", "LPAREN", "key_val_options_expr_header", "RPAREN", "fullselect", "opt_returning"},
	},
	"delete": [][]string{
		[]string{"DELETE", "opt_optim_hints", "FROM", "keyspace_ref", "opt_use_del_upd", "opt_where", "limit", "opt_offset", "opt_returning"},
		[]string{"DELETE", "opt_optim_hints", "FROM", "keyspace_ref", "opt_use_del_upd", "opt_where", "offset", "opt_limit", "opt_returning"},
		[]string{"DELETE", "opt_optim_hints", "FROM", "keyspace_ref", "opt_use_del_upd", "opt_where", "opt_returning"},
	},
	"update": [][]string{
		[]string{"UPDATE", "opt_optim_hints", "keyspace_ref", "opt_use_del_upd", "set", "unset", "opt_where", "opt_limit", "opt_returning"},
		[]string{"UPDATE", "opt_optim_hints", "keyspace_ref", "opt_use_del_upd", "set", "opt_where", "opt_limit", "opt_returning"},
		[]string{"UPDATE", "opt_optim_hints", "keyspace_ref", "opt_use_del_upd", "unset", "opt_where", "opt_limit", "opt_returning"},
	},
	"set": [][]string{
		[]string{"SET", "set_terms"},
	},
	"set_terms": [][]string{
		[]string{"set_term"},
		[]string{"set_terms", "COMMA", "set_term"},
	},
	"set_term": [][]string{
		[]string{"path", "EQ", "expr", "opt_update_for"},
		[]string{"function_meta_expr", "DOT", "path", "EQ", "expr"},
	},
	"function_meta_expr": [][]string{
		[]string{"function_name", "LPAREN", "opt_exprs", "RPAREN"},
	},
	"opt_update_for": [][]string{
		[]string{"%empty"},
		[]string{"update_for"},
	},
	"update_for": [][]string{
		[]string{"update_dimensions", "opt_when", "END"},
	},
	"update_dimensions": [][]string{
		[]string{"FOR", "update_dimension"},
		[]string{"update_dimensions", "FOR", "update_dimension"},
	},
	"update_dimension": [][]string{
		[]string{"update_binding"},
		[]string{"update_dimension", "COMMA", "update_binding"},
	},
	"update_binding": [][]string{
		[]string{"variable", "IN", "expr"},
		[]string{"variable", "WITHIN", "expr"},
		[]string{"variable", "COLON", "variable", "IN", "expr"},
		[]string{"variable", "COLON", "variable", "WITHIN", "expr"},
	},
	"variable": [][]string{
		[]string{"ident_or_default"},
	},
	"opt_when": [][]string{
		[]string{"%empty"},
		[]string{"WHEN", "expr"},
	},
	"unset": [][]string{
		[]string{"UNSET", "unset_terms"},
	},
	"unset_terms": [][]string{
		[]string{"unset_term"},
		[]string{"unset_terms", "COMMA", "unset_term"},
	},
	"unset_term": [][]string{
		[]string{"path", "opt_update_for"},
	},
	"merge": [][]string{
		[]string{"MERGE", "opt_optim_hints", "INTO", "simple_keyspace_ref", "opt_use_merge", "USING", "simple_from_term", "ON", "opt_key", "expr", "merge_actions", "opt_limit", "opt_returning"},
	},
	"opt_use_merge": [][]string{
		[]string{"opt_use"},
	},
	"opt_key": [][]string{
		[]string{"%empty"},
		[]string{"key"},
	},
	"merge_actions": [][]string{
		[]string{"%empty"},
		[]string{"WHEN", "MATCHED", "THEN", "UPDATE", "merge_update", "opt_merge_delete_insert"},
		[]string{"WHEN", "MATCHED", "THEN", "DELETE", "merge_delete", "opt_merge_insert"},
		[]string{"WHEN", "NOT", "MATCHED", "THEN", "INSERT", "merge_insert"},
	},
	"opt_merge_delete_insert": [][]string{
		[]string{"%empty"},
		[]string{"WHEN", "MATCHED", "THEN", "DELETE", "merge_delete", "opt_merge_insert"},
		[]string{"WHEN", "NOT", "MATCHED", "THEN", "INSERT", "merge_insert"},
	},
	"opt_merge_insert": [][]string{
		[]string{"%empty"},
		[]string{"WHEN", "NOT", "MATCHED", "THEN", "INSERT", "merge_insert"},
	},
	"merge_update": [][]string{
		[]string{"set", "opt_where"},
		[]string{"set", "unset", "opt_where"},
		[]string{"unset", "opt_where"},
	},
	"merge_delete": [][]string{
		[]string{"opt_where"},
	},
	"merge_insert": [][]string{
		[]string{"expr", "opt_where"},
		[]string{"key_val_expr", "opt_where"},
		[]string{"key_val_options_expr", "opt_where"},
		[]string{"LPAREN", "key_val_options_expr_header", "RPAREN", "opt_where"},
	},
	"grant_role": [][]string{
		[]string{"GRANT", "role_list", "TO", "user_list"},
		[]string{"GRANT", "role_list", "ON", "keyspace_scope_list", "TO", "user_list"},
	},
	"role_list": [][]string{
		[]string{"role_name"},
		[]string{"role_list", "COMMA", "role_name"},
	},
	"role_name": [][]string{
		[]string{"ident_or_default"},
		[]string{"SELECT"},
		[]string{"INSERT"},
		[]string{"UPDATE"},
		[]string{"DELETE"},
	},
	"keyspace_scope_list": [][]string{
		[]string{"keyspace_scope"},
		[]string{"keyspace_scope_list", "COMMA", "keyspace_scope"},
	},
	"keyspace_scope": [][]string{
		[]string{"keyspace_name"},
		[]string{"path_part", "DOT", "path_part"},
		[]string{"namespace_name", "keyspace_name"},
		[]string{"namespace_name", "path_part", "DOT", "path_part", "DOT", "keyspace_name"},
		[]string{"path_part", "DOT", "path_part", "DOT", "keyspace_name"},
		[]string{"namespace_name", "path_part", "DOT", "path_part"},
	},
	"user_list": [][]string{
		[]string{"user"},
		[]string{"user_list", "COMMA", "user"},
	},
	"user": [][]string{
		[]string{"ident_or_default"},
		[]string{"ident_or_default", "COLON", "ident_or_default"},
	},
	"revoke_role": [][]string{
		[]string{"REVOKE", "role_list", "FROM", "user_list"},
		[]string{"REVOKE", "role_list", "ON", "keyspace_scope_list", "FROM", "user_list"},
	},
	"create_scope": [][]string{
		[]string{"CREATE", "SCOPE", "named_scope_ref", "opt_if_not_exists"},
	},
	"drop_scope": [][]string{
		[]string{"DROP", "SCOPE", "named_scope_ref", "opt_if_exists"},
	},
	"create_collection": [][]string{
		[]string{"CREATE", "COLLECTION", "named_keyspace_ref", "opt_if_not_exists", "opt_with_clause"},
	},
	"drop_collection": [][]string{
		[]string{"DROP", "COLLECTION", "named_keyspace_ref", "opt_if_exists"},
	},
	"flush_collection": [][]string{
		[]string{"flush_or_truncate", "COLLECTION", "named_keyspace_ref"},
	},
	"flush_or_truncate": [][]string{
		[]string{"FLUSH"},
		[]string{"TRUNCATE"},
	},
	"create_index": [][]string{
		[]string{"CREATE", "PRIMARY", "INDEX", "opt_primary_name", "opt_if_not_exists", "ON", "named_keyspace_ref", "index_partition", "opt_index_using", "opt_with_clause"},
		[]string{"CREATE", "INDEX", "index_name", "opt_if_not_exists", "ON", "named_keyspace_ref", "LPAREN", "index_terms", "RPAREN", "index_partition", "index_where", "opt_index_using", "opt_with_clause"},
	},
	"opt_primary_name": [][]string{
		[]string{"%empty"},
		[]string{"index_name"},
	},
	"index_name": [][]string{
		[]string{"ident_or_default"},
	},
	"opt_index_name": [][]string{
		[]string{"%empty"},
		[]string{"index_name"},
	},
	"opt_if_not_exists": [][]string{
		[]string{"%empty"},
		[]string{"IF", "NOT", "EXISTS"},
	},
	"named_keyspace_ref": [][]string{
		[]string{"simple_named_keyspace_ref"},
		[]string{"namespace_name", "path_part"},
		[]string{"path_part", "DOT", "path_part", "DOT", "keyspace_name"},
		[]string{"path_part", "DOT", "keyspace_name"},
	},
	"simple_named_keyspace_ref": [][]string{
		[]string{"keyspace_name"},
		[]string{"namespace_name", "path_part", "DOT", "path_part", "DOT", "keyspace_name"},
	},
	"named_scope_ref": [][]string{
		[]string{"namespace_name", "path_part", "DOT", "path_part"},
		[]string{"path_part", "DOT", "path_part"},
		[]string{"path_part"},
	},
	"index_partition": [][]string{
		[]string{"%empty"},
		[]string{"PARTITION", "BY", "HASH", "LPAREN", "exprs", "RPAREN"},
	},
	"opt_index_using": [][]string{
		[]string{"%empty"},
		[]string{"index_using"},
	},
	"index_using": [][]string{
		[]string{"USING", "VIEW"},
		[]string{"USING", "GSI"},
		[]string{"USING", "FTS"},
	},
	"index_terms": [][]string{
		[]string{"index_term"},
		[]string{"index_terms", "COMMA", "index_term"},
	},
	"index_term": [][]string{
		[]string{"index_term_expr", "opt_ikattr"},
	},
	"index_term_expr": [][]string{
		[]string{"expr"},
		[]string{"all_expr"},
	},
	"all_expr": [][]string{
		[]string{"all", "expr"},
		[]string{"all", "DISTINCT", "expr"},
		[]string{"DISTINCT", "expr"},
	},
	"all": [][]string{
		[]string{"ALL"},
		[]string{"EACH"},
	},
	"flatten_keys_expr": [][]string{
		[]string{"expr", "opt_ikattr"},
	},
	"flatten_keys_exprs": [][]string{
		[]string{"flatten_keys_expr"},
		[]string{"flatten_keys_exprs", "COMMA", "flatten_keys_expr"},
	},
	"opt_flatten_keys_exprs": [][]string{
		[]string{"%empty"},
		[]string{"flatten_keys_exprs"},
	},
	"index_where": [][]string{
		[]string{"%empty"},
		[]string{"WHERE", "expr"},
	},
	"opt_ikattr": [][]string{
		[]string{"%empty"},
		[]string{"ikattr"},
		[]string{"ikattr", "ikattr"},
	},
	"ikattr": [][]string{
		[]string{"ASC"},
		[]string{"DESC"},
		[]string{"INCLUDE", "MISSING"},
	},
	"drop_index": [][]string{
		[]string{"DROP", "PRIMARY", "INDEX", "opt_primary_name", "opt_if_exists", "ON", "named_keyspace_ref", "opt_index_using"},
		[]string{"DROP", "INDEX", "simple_named_keyspace_ref", "DOT", "index_name", "opt_if_exists", "opt_index_using"},
		[]string{"DROP", "INDEX", "index_name", "opt_if_exists", "ON", "named_keyspace_ref", "opt_index_using"},
	},
	"opt_if_exists": [][]string{
		[]string{"%empty"},
		[]string{"IF", "EXISTS"},
	},
	"alter_index": [][]string{
		[]string{"ALTER", "INDEX", "simple_named_keyspace_ref", "DOT", "index_name", "opt_index_using", "with_clause"},
		[]string{"ALTER", "INDEX", "index_name", "ON", "named_keyspace_ref", "opt_index_using", "with_clause"},
	},
	"build_index": [][]string{
		[]string{"BUILD", "INDEX", "ON", "named_keyspace_ref", "LPAREN", "exprs", "RPAREN", "opt_index_using"},
	},
	"create_function": [][]string{
		[]string{"CREATE", "opt_replace", "FUNCTION", "func_name", "LPAREN", "parm_list", "RPAREN", "opt_if_not_exists", "func_body"},
	},
	"opt_replace": [][]string{
		[]string{"%empty"},
		[]string{"OR", "REPLACE"},
	},
	"func_name": [][]string{
		[]string{"short_func_name"},
		[]string{"long_func_name"},
	},
	"short_func_name": [][]string{
		[]string{"keyspace_name"},
		[]string{"path_part", "DOT", "path_part"},
		[]string{"path_part", "DOT", "path_part", "DOT", "path_part"},
	},
	"long_func_name": [][]string{
		[]string{"namespace_term", "keyspace_name"},
		[]string{"namespace_term", "path_part", "DOT", "path_part", "DOT", "keyspace_name"},
	},
	"parm_list": [][]string{
		[]string{"%empty"},
		[]string{"DOT", "DOT", "DOT"},
		[]string{"parameter_terms"},
	},
	"parameter_terms": [][]string{
		[]string{"ident_or_default"},
		[]string{"parameter_terms", "COMMA", "ident_or_default"},
	},
	"func_body": [][]string{
		[]string{"LBRACE", "expr", "RBRACE"},
		[]string{"LANGUAGE", "INLINE", "AS", "expr"},
		[]string{"LANGUAGE", "JAVASCRIPT", "AS", "STR"},
		[]string{"LANGUAGE", "JAVASCRIPT", "AS", "STR", "AT", "STR"},
		[]string{"LANGUAGE", "GOLANG", "AS", "STR", "AT", "STR"},
	},
	"drop_function": [][]string{
		[]string{"DROP", "FUNCTION", "func_name", "opt_if_exists"},
	},
	"execute_function": [][]string{
		[]string{"EXECUTE", "FUNCTION", "func_name", "LPAREN", "opt_exprs", "RPAREN"},
	},
	"update_statistics": [][]string{
		[]string{"UPDATE", "STATISTICS", "opt_for", "named_keyspace_ref", "LPAREN", "update_stat_terms", "RPAREN", "opt_with_clause"},
		[]string{"UPDATE", "STATISTICS", "opt_for", "named_keyspace_ref", "DELETE", "LPAREN", "update_stat_terms", "RPAREN"},
		[]string{"UPDATE", "STATISTICS", "opt_for", "named_keyspace_ref", "DELETE", "ALL"},
		[]string{"UPDATE", "STATISTICS", "opt_for", "named_keyspace_ref", "INDEX", "LPAREN", "exprs", "RPAREN", "opt_index_using", "opt_with_clause"},
		[]string{"UPDATE", "STATISTICS", "opt_for", "named_keyspace_ref", "INDEX", "ALL", "opt_index_using", "opt_with_clause"},
		[]string{"UPDATE", "STATISTICS", "FOR", "INDEX", "simple_named_keyspace_ref", "DOT", "index_name", "opt_index_using", "opt_with_clause"},
		[]string{"UPDATE", "STATISTICS", "FOR", "INDEX", "index_name", "ON", "named_keyspace_ref", "opt_index_using", "opt_with_clause"},
		[]string{"ANALYZE", "opt_keyspace_collection", "named_keyspace_ref", "LPAREN", "update_stat_terms", "RPAREN", "opt_with_clause"},
		[]string{"ANALYZE", "opt_keyspace_collection", "named_keyspace_ref", "DELETE", "STATISTICS", "LPAREN", "update_stat_terms", "RPAREN"},
		[]string{"ANALYZE", "opt_keyspace_collection", "named_keyspace_ref", "DELETE", "STATISTICS"},
		[]string{"ANALYZE", "opt_keyspace_collection", "named_keyspace_ref", "INDEX", "LPAREN", "exprs", "RPAREN", "opt_index_using", "opt_with_clause"},
		[]string{"ANALYZE", "opt_keyspace_collection", "named_keyspace_ref", "INDEX", "ALL", "opt_index_using", "opt_with_clause"},
		[]string{"ANALYZE", "INDEX", "simple_named_keyspace_ref", "DOT", "index_name", "opt_index_using", "opt_with_clause"},
		[]string{"ANALYZE", "INDEX", "index_name", "ON", "named_keyspace_ref", "opt_index_using", "opt_with_clause"},
	},
	"opt_for": [][]string{
		[]string{"%empty"},
		[]string{"FOR"},
	},
	"update_stat_terms": [][]string{
		[]string{"update_stat_term"},
		[]string{"update_stat_terms", "COMMA", "update_stat_term"},
	},
	"update_stat_term": [][]string{
		[]string{"index_term_expr"},
	},
	"path": [][]string{
		[]string{"ident_or_default"},
		[]string{"path", "DOT", "ident_or_default"},
		[]string{"path", "DOT", "ident_icase"},
		[]string{"path", "DOT", "LBRACKET", "expr", "RBRACKET"},
		[]string{"path", "DOT", "LBRACKET", "expr", "RBRACKET_ICASE"},
		[]string{"path", "LBRACKET", "expr", "RBRACKET"},
	},
	"ident": [][]string{
		[]string{"ident_or_default"},
	},
	"ident_icase": [][]string{
		[]string{"IDENT_ICASE"},
	},
	"expr": [][]string{
		[]string{"c_expr"},
		[]string{"expr", "DOT", "ident", "LPAREN", "opt_exprs", "RPAREN"},
		[]string{"expr", "DOT", "ident"},
		[]string{"expr", "DOT", "ident_icase"},
		[]string{"expr", "DOT", "LBRACKET", "expr", "RBRACKET"},
		[]string{"expr", "DOT", "LBRACKET", "expr", "RBRACKET_ICASE"},
		[]string{"expr", "LBRACKET", "RANDOM_ELEMENT", "RBRACKET"},
		[]string{"expr", "LBRACKET", "expr", "RBRACKET"},
		[]string{"expr", "LBRACKET", "expr", "COLON", "RBRACKET"},
		[]string{"expr", "LBRACKET", "expr", "COLON", "expr", "RBRACKET"},
		[]string{"expr", "LBRACKET", "COLON", "expr", "RBRACKET"},
		[]string{"expr", "LBRACKET", "COLON", "RBRACKET"},
		[]string{"expr", "LBRACKET", "RBRACKET"},
		[]string{"expr", "LBRACKET", "STAR", "RBRACKET"},
		[]string{"expr", "PLUS", "expr"},
		[]string{"expr", "MINUS", "expr"},
		[]string{"expr", "STAR", "expr"},
		[]string{"expr", "DIV", "expr"},
		[]string{"expr", "MOD", "expr"},
		[]string{"expr", "POW", "expr"},
		[]string{"expr", "CONCAT", "expr"},
		[]string{"expr", "AND", "expr"},
		[]string{"expr", "OR", "expr"},
		[]string{"NOT", "expr"},
		[]string{"expr", "EQ", "expr"},
		[]string{"expr", "DEQ", "expr"},
		[]string{"expr", "NE", "expr"},
		[]string{"expr", "LT", "expr"},
		[]string{"expr", "GT", "expr"},
		[]string{"expr", "LE", "expr"},
		[]string{"expr", "GE", "expr"},
		[]string{"expr", "BETWEEN", "b_expr", "AND", "b_expr"},
		[]string{"expr", "NOT", "BETWEEN", "b_expr", "AND", "b_expr"},
		[]string{"expr", "LIKE", "expr", "ESCAPE", "expr"},
		[]string{"expr", "LIKE", "expr"},
		[]string{"expr", "NOT", "LIKE", "expr", "ESCAPE", "expr"},
		[]string{"expr", "NOT", "LIKE", "expr"},
		[]string{"expr", "IN", "expr"},
		[]string{"expr", "NOT", "IN", "expr"},
		[]string{"expr", "WITHIN", "expr"},
		[]string{"expr", "NOT", "WITHIN", "expr"},
		[]string{"expr", "IS", "NULL"},
		[]string{"expr", "IS", "NOT", "NULL"},
		[]string{"expr", "IS", "MISSING"},
		[]string{"expr", "IS", "NOT", "MISSING"},
		[]string{"expr", "IS", "valued"},
		[]string{"expr", "IS", "NOT", "UNKNOWN"},
		[]string{"expr", "IS", "NOT", "valued"},
		[]string{"expr", "IS", "UNKNOWN"},
		[]string{"expr", "IS", "DISTINCT", "FROM", "expr"},
		[]string{"expr", "IS", "NOT", "DISTINCT", "FROM", "expr"},
		[]string{"EXISTS", "expr"},
	},
	"valued": [][]string{
		[]string{"VALUED"},
		[]string{"KNOWN"},
	},
	"c_expr": [][]string{
		[]string{"literal"},
		[]string{"construction_expr"},
		[]string{"ident_or_default"},
		[]string{"IDENT_ICASE"},
		[]string{"SELF"},
		[]string{"param_expr"},
		[]string{"function_expr"},
		[]string{"MINUS", "expr"},
		[]string{"case_expr"},
		[]string{"collection_expr"},
		[]string{"paren_expr"},
	},
	"b_expr": [][]string{
		[]string{"c_expr"},
		[]string{"b_expr", "DOT", "ident_or_default", "LPAREN", "opt_exprs", "RPAREN"},
		[]string{"b_expr", "DOT", "ident_or_default"},
		[]string{"b_expr", "DOT", "ident_icase"},
		[]string{"b_expr", "DOT", "LBRACKET", "expr", "RBRACKET"},
		[]string{"b_expr", "DOT", "LBRACKET", "expr", "RBRACKET_ICASE"},
		[]string{"b_expr", "LBRACKET", "expr", "RBRACKET"},
		[]string{"b_expr", "LBRACKET", "expr", "COLON", "RBRACKET"},
		[]string{"b_expr", "LBRACKET", "COLON", "expr", "RBRACKET"},
		[]string{"b_expr", "LBRACKET", "expr", "COLON", "expr", "RBRACKET"},
		[]string{"b_expr", "LBRACKET", "COLON", "RBRACKET"},
		[]string{"b_expr", "LBRACKET", "STAR", "RBRACKET"},
		[]string{"b_expr", "PLUS", "b_expr"},
		[]string{"b_expr", "MINUS", "b_expr"},
		[]string{"b_expr", "STAR", "b_expr"},
		[]string{"b_expr", "DIV", "b_expr"},
		[]string{"b_expr", "MOD", "b_expr"},
		[]string{"b_expr", "POW", "b_expr"},
		[]string{"b_expr", "CONCAT", "b_expr"},
	},
	"literal": [][]string{
		[]string{"NULL"},
		[]string{"MISSING"},
		[]string{"FALSE"},
		[]string{"TRUE"},
		[]string{"NUM"},
		[]string{"INT"},
		[]string{"STR"},
	},
	"construction_expr": [][]string{
		[]string{"object"},
		[]string{"array"},
	},
	"object": [][]string{
		[]string{"LBRACE", "opt_members", "RBRACE"},
	},
	"opt_members": [][]string{
		[]string{"%empty"},
		[]string{"members"},
	},
	"members": [][]string{
		[]string{"member"},
		[]string{"members", "COMMA", "member"},
	},
	"member": [][]string{
		[]string{"expr", "COLON", "expr"},
		[]string{"expr", "opt_as_alias"},
	},
	"array": [][]string{
		[]string{"LBRACKET", "opt_exprs", "RBRACKET"},
	},
	"opt_exprs": [][]string{
		[]string{"%empty"},
		[]string{"exprs"},
	},
	"exprs": [][]string{
		[]string{"expr"},
		[]string{"exprs", "COMMA", "expr"},
	},
	"param_expr": [][]string{
		[]string{"NAMED_PARAM"},
		[]string{"POSITIONAL_PARAM"},
		[]string{"NEXT_PARAM"},
	},
	"case_expr": [][]string{
		[]string{"CASE", "simple_or_searched_case", "END"},
	},
	"simple_or_searched_case": [][]string{
		[]string{"simple_case"},
		[]string{"searched_case"},
	},
	"simple_case": [][]string{
		[]string{"expr", "when_thens", "opt_else"},
	},
	"when_thens": [][]string{
		[]string{"WHEN", "expr", "THEN", "expr"},
		[]string{"when_thens", "WHEN", "expr", "THEN", "expr"},
	},
	"searched_case": [][]string{
		[]string{"when_thens", "opt_else"},
	},
	"opt_else": [][]string{
		[]string{"%empty"},
		[]string{"ELSE", "expr"},
	},
	"function_expr": [][]string{
		[]string{"FLATTEN_KEYS", "LPAREN", "opt_flatten_keys_exprs", "RPAREN"},
		[]string{"NTH_VALUE", "LPAREN", "exprs", "RPAREN", "opt_from_first_last", "opt_nulls_treatment", "window_function_details"},
		[]string{"function_name", "LPAREN", "opt_exprs", "RPAREN", "opt_filter", "opt_nulls_treatment", "opt_window_function"},
		[]string{"function_name", "LPAREN", "agg_quantifier", "expr", "RPAREN", "opt_filter", "opt_window_function"},
		[]string{"function_name", "LPAREN", "STAR", "RPAREN", "opt_filter", "opt_window_function"},
		[]string{"long_func_name", "LPAREN", "opt_exprs", "RPAREN"},
	},
	"function_name": [][]string{
		[]string{"ident"},
		[]string{"REPLACE"},
	},
	"collection_expr": [][]string{
		[]string{"collection_cond"},
		[]string{"collection_xform"},
	},
	"collection_cond": [][]string{
		[]string{"ANY", "coll_bindings", "satisfies", "END"},
		[]string{"SOME", "coll_bindings", "satisfies", "END"},
		[]string{"EVERY", "coll_bindings", "satisfies", "END"},
		[]string{"ANY", "AND", "EVERY", "coll_bindings", "satisfies", "END"},
		[]string{"SOME", "AND", "EVERY", "coll_bindings", "satisfies", "END"},
	},
	"coll_bindings": [][]string{
		[]string{"coll_binding"},
		[]string{"coll_bindings", "COMMA", "coll_binding"},
	},
	"coll_binding": [][]string{
		[]string{"variable", "IN", "expr"},
		[]string{"variable", "WITHIN", "expr"},
		[]string{"variable", "COLON", "variable", "IN", "expr"},
		[]string{"variable", "COLON", "variable", "WITHIN", "expr"},
	},
	"satisfies": [][]string{
		[]string{"SATISFIES", "expr"},
	},
	"collection_xform": [][]string{
		[]string{"ARRAY", "expr", "FOR", "coll_bindings", "opt_when", "END"},
		[]string{"FIRST", "expr", "FOR", "coll_bindings", "opt_when", "END"},
		[]string{"OBJECT", "expr", "COLON", "expr", "FOR", "coll_bindings", "opt_when", "END"},
	},
	"paren_expr": [][]string{
		[]string{"LPAREN", "expr", "RPAREN"},
		[]string{"LPAREN", "all_expr", "RPAREN"},
		[]string{"subquery_expr"},
	},
	"subquery_expr": [][]string{
		[]string{"CORRELATED", "LPAREN", "fullselect", "RPAREN"},
		[]string{"LPAREN", "fullselect", "RPAREN"},
		[]string{"all_expr"},
	},
	"opt_window_clause": [][]string{
		[]string{"%empty"},
		[]string{"WINDOW", "window_list"},
	},
	"window_list": [][]string{
		[]string{"window_term"},
		[]string{"window_list", "COMMA", "window_term"},
	},
	"window_term": [][]string{
		[]string{"ident_or_default", "AS", "window_specification"},
	},
	"window_specification": [][]string{
		[]string{"LPAREN", "opt_window_name", "opt_window_partition", "opt_order_by", "opt_window_frame", "RPAREN"},
	},
	"opt_window_name": [][]string{
		[]string{"%empty"},
		[]string{"ident_or_default"},
	},
	"opt_window_partition": [][]string{
		[]string{"%empty"},
		[]string{"PARTITION", "BY", "exprs"},
	},
	"opt_window_frame": [][]string{
		[]string{"%empty"},
		[]string{"window_frame_modifier", "window_frame_extents", "opt_window_frame_exclusion"},
	},
	"window_frame_modifier": [][]string{
		[]string{"ROWS"},
		[]string{"RANGE"},
		[]string{"GROUPS"},
	},
	"opt_window_frame_exclusion": [][]string{
		[]string{"%empty"},
		[]string{"EXCLUDE", "NO", "OTHERS"},
		[]string{"EXCLUDE", "CURRENT", "ROW"},
		[]string{"EXCLUDE", "TIES"},
		[]string{"EXCLUDE", "GROUP"},
	},
	"window_frame_extents": [][]string{
		[]string{"window_frame_extent"},
		[]string{"BETWEEN", "window_frame_extent", "AND", "window_frame_extent"},
	},
	"window_frame_extent": [][]string{
		[]string{"UNBOUNDED", "PRECEDING"},
		[]string{"UNBOUNDED", "FOLLOWING"},
		[]string{"CURRENT", "ROW"},
		[]string{"expr", "window_frame_valexpr_modifier"},
	},
	"window_frame_valexpr_modifier": [][]string{
		[]string{"PRECEDING"},
		[]string{"FOLLOWING"},
	},
	"opt_nulls_treatment": [][]string{
		[]string{"%empty"},
		[]string{"nulls_treatment"},
	},
	"nulls_treatment": [][]string{
		[]string{"RESPECT", "NULLS"},
		[]string{"IGNORE", "NULLS"},
	},
	"opt_from_first_last": [][]string{
		[]string{"%empty"},
		[]string{"FROM", "first_last"},
	},
	"agg_quantifier": [][]string{
		[]string{"ALL"},
		[]string{"DISTINCT"},
	},
	"opt_filter": [][]string{
		[]string{"%empty"},
		[]string{"FILTER", "LPAREN", "where", "RPAREN"},
	},
	"opt_window_function": [][]string{
		[]string{"%empty"},
		[]string{"window_function_details"},
	},
	"window_function_details": [][]string{
		[]string{"OVER", "ident_or_default"},
		[]string{"OVER", "window_specification"},
	},
	"start_transaction": [][]string{
		[]string{"start_or_begin", "transaction", "opt_isolation_level"},
	},
	"commit_transaction": [][]string{
		[]string{"COMMIT", "opt_transaction"},
	},
	"rollback_transaction": [][]string{
		[]string{"ROLLBACK", "opt_transaction", "opt_savepoint"},
	},
	"start_or_begin": [][]string{
		[]string{"START"},
		[]string{"BEGIN"},
	},
	"opt_transaction": [][]string{
		[]string{"%empty"},
		[]string{"transaction"},
	},
	"transaction": [][]string{
		[]string{"TRAN"},
		[]string{"TRANSACTION"},
		[]string{"WORK"},
	},
	"opt_savepoint": [][]string{
		[]string{"%empty"},
		[]string{"TO", "SAVEPOINT", "savepoint_name"},
	},
	"savepoint_name": [][]string{
		[]string{"ident_or_default"},
	},
	"opt_isolation_level": [][]string{
		[]string{"%empty"},
		[]string{"isolation_level"},
	},
	"isolation_level": [][]string{
		[]string{"ISOLATION", "LEVEL", "isolation_val"},
	},
	"isolation_val": [][]string{
		[]string{"READ", "COMMITTED"},
	},
	"set_transaction_isolation": [][]string{
		[]string{"SET", "TRANSACTION", "isolation_level"},
	},
	"savepoint": [][]string{
		[]string{"SAVEPOINT", "savepoint_name"},
	},
	"opt_with_clause": [][]string{
		[]string{"%empty"},
		[]string{"with_clause"},
	},
	"with_clause": [][]string{
		[]string{"WITH", "expr"},
	},
}
