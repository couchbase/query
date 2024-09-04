//  Copyright 2024-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package errors

import "fmt"

func getSchemaHelp(setting string) string {
	var help string
	switch setting {
	case "change_percentage":
		help = "Integer between 0 and 100."
	case "schedule.start_time", "schedule.end_time":
		help = "Valid timestamp in HH:MM:SS format."
	case "schedule.timezone":
		help = "UTC or IANA timezone."
	case "schedule.days":
		help = "List of text string names of the days of the week."
	case "enable", "all_buckets":
		help = "boolean."
	default:
		return ""
	}

	return fmt.Sprintf("Setting '%s' must be: %s", setting, help)
}

func getSemanticsHelp(setting string) string {
	switch setting {
	case "schedule.start_time", "schedule.end_time":
		return "'schedule.start_time' must be earlier than 'schedule.end_time' by at least 30 minutes."
	case "schedule":
		return "A valid schedule must be set if Auto Update Statistics is enabled."
	default:
		return ""
	}
}

func NewAusNotSupportedError() Error {
	return &err{level: EXCEPTION, ICode: E_AUS_NOT_SUPPORTED, IKey: "aus.not_supported",
		InternalMsg:    "Auto Update Statistics is not supported in Community Edition. It is an enterprise level feature.",
		InternalCaller: CallerN(1)}
}

func NewAusNotInitialized() Error {
	return &err{level: EXCEPTION, ICode: E_AUS_NOT_INITIALIZED, IKey: "aus.not_initialized",
		InternalMsg: "Auto Update Statistics is not initialized for the node. It is only available on clusters migrated to" +
			" a supported version.",
		InternalCaller: CallerN(1)}
}

func NewAusStorageAccessError(cause error) Error {
	return &err{level: EXCEPTION, ICode: E_AUS_STORAGE, IKey: "aus.storage.error",
		InternalMsg: "Error accessing Auto Update Statistics information from storage.", cause: cause, InternalCaller: CallerN(1)}
}

func NewAusDocInvalidSettingsValue(setting string, value interface{}) Error {
	c := make(map[string]interface{})
	c["cause"] = fmt.Sprintf("Invalid value '%v' (%T) for setting '%s'", value, value, setting)

	if help := getSchemaHelp(setting); help != "" {
		c["help"] = help
	}

	return &err{level: EXCEPTION, ICode: E_AUS_INVALID_DOCUMENT_SCHEMA, IKey: "aus.settings.invalid_schema",
		InternalMsg: "Invalid schema or semantics detected in the Auto Update Statistics settings document.", cause: c,
		InternalCaller: CallerN(1)}
}

func NewAusDocMissingSetting(setting string, defaultVal interface{}) Error {
	c := make(map[string]interface{})
	c["cause"] = fmt.Sprintf("Setting '%s' cannot be missing in the Auto Update Statistics settings document.", setting)

	if help := getSchemaHelp(setting); help != "" {
		c["help"] = help
	}

	if defaultVal != nil {
		c["default_action_taken"] = fmt.Sprintf("'%s' set to default value of: %v", setting, defaultVal)
	}

	return &err{level: EXCEPTION, ICode: E_AUS_INVALID_DOCUMENT_SCHEMA, IKey: "aus.settings.invalid_schema",
		InternalMsg: "Invalid schema or semantics detected in the Auto Update Statistics settings document.", cause: c,
		InternalCaller: CallerN(1)}
}

func NewAusDocUnknownSetting(setting string) Error {
	return &err{level: EXCEPTION, ICode: E_AUS_INVALID_DOCUMENT_SCHEMA, IKey: "aus.settings.invalid_schema",
		InternalMsg:    "Invalid schema or semantics detected in the Auto Update Statistics settings document.",
		cause:          fmt.Sprintf("Unknown setting '%s' detected in the Auto Update Statistics settings document.", setting),
		InternalCaller: CallerN(1)}
}

func NewAusDocInvalidSemantics(setting string) Error {
	return &err{level: EXCEPTION, ICode: E_AUS_INVALID_DOCUMENT_SCHEMA, IKey: "aus.settings.invalid_schema",
		InternalMsg:    "Invalid schema or semantics detected in the Auto Update Statistics settings document.",
		cause:          getSemanticsHelp(setting),
		InternalCaller: CallerN(1)}
}

func NewAusDocEncodingError(isEncode bool, cause error) Error {
	var action string
	if isEncode {
		action = "encoding"
	} else {
		action = "decoding"
	}

	return &err{level: EXCEPTION, ICode: E_AUS_SETTINGS_ENCODING, IKey: "aus.settings.encoding_error",
		InternalMsg: fmt.Sprintf("Error %s Automatic Update Statistics settings document.", action), ICause: cause,
		InternalCaller: CallerN(1)}
}

func NewAusStorageInvalidKey(key string, cause error) Error {
	return &err{level: EXCEPTION, ICode: E_AUS_STORAGE_INVALID_KEY, IKey: "aus.storage.invalid_key",
		InternalMsg: fmt.Sprintf("Invalid document key '%s' for Auto Update Statistics document", key), cause: cause,
		InternalCaller: CallerN(1)}
}
