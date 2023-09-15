package errors

type ErrData struct {
	Code        int
	ErrorCode   string
	Description string
	Causes      []string
	Actions     []string
	IsUser      bool
}

var errmap = map[ErrorCode]ErrData{
	E_SERVICE_READONLY: {
		Code:        1000,
		ErrorCode:   "E_SERVICE_READONLY",
		Description: "The server or request is read-only and cannot accept this write statement.",
		Causes: []string{
			"When a request is read-only statements that write data are not permitted.  A statement may be read-only by being submitted to the REST enpoint using the GET method or by setting the \"readonly\" request parameter.  Read-only requests may PREPARE statements, including write statements when the \"auto_execute\" request parameter is not true.",
		},
		Actions: []string{
			"Use POST to submit write statements and ensure the \"readonly\" request parameter is not set or is set to false.",
		},
		IsUser: true,
	},
	E_SERVICE_HTTP_UNSUPPORTED_METHOD: {
		Code:        1010,
		ErrorCode:   "E_SERVICE_HTTP_UNSUPPORTED_METHOD",
		Description: "Unsupported http method:[METHOD]",
		Causes: []string{
			"The service endpoint supports only GET & POST HTTP methods.  All other HTTP methods are not supported.",
		},
		Actions: []string{
			"Use a supported method to submit requests.",
		},
		IsUser: true,
	},
	E_SERVICE_NOT_IMPLEMENTED: {
		Code:        1020,
		ErrorCode:   "E_SERVICE_NOT_IMPLEMENTED",
		Description: "[feature] [value] not implemented",
		Causes: []string{
			"The noted feature/value combination is reserved but is not implemented.",
		},
		Actions: []string{
			"Use only supported feature/value combinations.",
		},
		IsUser: true,
	},
	E_SERVICE_UNRECOGNIZED_VALUE: {
		Code:        1030,
		ErrorCode:   "E_SERVICE_UNRECOGNIZED_VALUE",
		Description: "Unknown [parameter] value: [value]",
		Causes: []string{
			"The value supplied for the noted parameter is not recognised.",
		},
		Actions: []string{
			"Ensure the value supplied is  a supported value in the required format for the request parameter noted.",
		},
		IsUser: true,
	},
	E_SERVICE_BAD_VALUE: {
		Code:        1040,
		ErrorCode:   "E_SERVICE_BAD_VALUE",
		Description: "Error processing [message]",
		Causes: []string{
			"There was an error in processing as detailed in the message.  For example, a non-numeric string value passed as the value for a request parameter that is expected to be numeric.",
		},
		Actions: []string{
			"Where there error is derived from user-controlled data, correct the data.",
		},
	},
	E_SERVICE_MISSING_VALUE: {
		Code:        1050,
		ErrorCode:   "E_SERVICE_MISSING_VALUE",
		Description: "No [parameter] value",
		Causes: []string{
			"A value was not supplied for the required parameter.",
		},
		Actions: []string{
			"Provide valid values for all required parameters.  For example, ensure a user and password are supplied for all requests and a scan_vector is supplied for requests using AT_PLUS consistency level.",
		},
		IsUser: true,
	},
	E_SERVICE_MULTIPLE_VALUES: {
		Code:        1060,
		ErrorCode:   "E_SERVICE_MULTIPLE_VALUES",
		Description: "Multiple values for [feature]",
		Causes: []string{
			"1) namedArgument: each named argument passed in the request string must be unique, example bad request: /query/service?statement=SELECT $name;&$name=\"a\"&$name=\"b\",\n\"errors\":[\n{\n\"code\": 1060,\n\"msg\": \"Multiple values for 'name'.\"\n}\n]",
			"2) multiple statements( PREPARE/DML statements/ DDL statements) , example bad request: query/service?statement=SELECT $name;&$name=\"a\"&statement=CREATE INDEX defix ON default(a);",
			"3) user has passed both scan_vector & scan_vectors parameter.",
			"4) user has both auto_execute(request_level setting) and auto_prepare(service_level or request_level) set to true.",
			"5) when passing namedargument through a POST request body, user has a repeat for a particular namedArgumet\n    example: \nPOST /query/service?statement=SELECT%2520%2540key%253B&%2524name=%2522a%2522 HTTP/1.1\nAuthorization: Basic QWRtaW5pc3RyYXRvcjpwYXNzd29yZA==\nContent-Length: 38\nContent-Type: application/json\nHost: 127.0.0.1:9499\n{\n  \"@key\":\"secret\",\n  \"@key\":\"NOPE\"\n}\n\n\"errors\":[\n{\n\"code\": 1060,\n\"msg\": \"Multiple values for @key.\"\n}\n]",
			"6) URL form: multiple values for a request header field",
			"7) request query string has multiple values for a request level setting, example: /query/service?statement=SELECT 1;&auto_execute=true&auto_execute=2",
		},
		Actions: []string{
			"1) ensure all namedargument parameters ($[identifier] or @[identifier]) are unique",
			"2) have only one statement request parameter, execute the other statement in a new request.",
			"3) when using scan_vector cannot have scan_vectors parameter, either scan_vector(for a single keyspace query) or scan_vectors(query with multiple keyspaces). In other words cannot have both. Typical usage is with scan_consistency set to AT_PLUS to insure index is upto date with datastore.",
			"4) doesn't make sense to have both auto_execute and auto_prepare.",
			"5) whenever passing namedArgs through a json request body, ensure they are all unique. Unlikely case as only form value we take is authorization? ( that is caught by another error)",
			"6) ensure all request parameters  https://docs.couchbase.com/server/current/n1ql/n1ql-rest-api/index.html#_request_parameters passed are unique ",
		},
		IsUser: true,
	},
	E_SERVICE_UNRECOGNIZED_PARAMETER: {
		Code:        1065,
		ErrorCode:   "E_SERVICE_UNRECOGNIZED_PARAMETER",
		Description: "Unrecognized parameter in request: [parameter]",
		Causes: []string{
			"1) request URL string has an unrecognized request parameter.",
			"2) request body has an unrecognized parameter.(again the request level parameters)\n\nexample: query/service?statement=SELECT 1;&notvalidreqparam=3\n\"errors\":[\n{\n\"code\": 1065,\n\"msg\": \"Unrecognized parameter in request: notvalidreqparam\"\n}\n]",
		},
		Actions: []string{
			"ensure when passing request parameters either through URL string or request body, ensure it is from the listed setting here https://docs.couchbase.com/server/current/n1ql/n1ql-rest-api/index.html#_request_parameters \n",
		},
		IsUser: true,
	},
	E_SERVICE_TYPE_MISMATCH: {
		Code:        1070,
		ErrorCode:   "E_SERVICE_TYPE_MISMATCH",
		Description: "[feature] has to be of type [expected]",
		Causes: []string{
			"1) when parsing input for scan_vector as full_scan vector: a) Expected array of 1024 entries( entry [vbseqno[NUMBER], vbucketuuid[string]) but got fewer entries, b) entry expected is an array of length 2, but got something else.",
			"2) scan_vector pararmeter is expected to be array(full_scan vector) or map[string][entry] (sparse vector)",
			"3) \"args\" request parameter is expected to be an array.",
			"4) \"creds\" request parameter is expected to be array of {user, pass}",
			"5) following request parameters are expected to have a string value: durability_level, atrcollection, format, compression, endcoding, query_context, txid, statement, timeout, kvtimeout, namespace, loglevel, usereplica, duration_style, client_context_id, txtimeout",
			"6) following request parameters are expected to have tristate value: true/false/none\nusefts, sort_projection, controls, signature, auto_prepare, auto_execute, usecbo, tximplicit, preserve_expiry, readonly, metrics, pretty.",
		},
		Actions: []string{
			"1) use cbstat for this getting vbucket_uuid and vbucket_seqno-> https://docs.couchbase.com/server/7.1/cli/cbstats/cbstats-vbucket-seqno.html#example",
			"2) Scan vectors have two forms: Full scan vector: an array of [value, guard] entries, giving an entry for every vBucket in the system.\nSparse scan vectors: an object providing entries for specific vBuckets, mapping a vBucket number (a string) to each [value, guard] entry.",
			"3) \"args\" request parameter must be something like [\"abc\", 31].",
			"4) \"creds\" request parameter must be something like [ { \"user\" : \"local:bucket-name\", \"pass\" : \"password\" }, { \"user\" : \"admin:admin-name\", \"pass\" : \"password\" } ].",
			"5) change value to allowed string , link to documentation https://docs.couchbase.com/server/current/n1ql/n1ql-rest-api/index.html#_request_parameters",
			"6) change value to a tristate value, link to documentation https://docs.couchbase.com/server/current/n1ql/n1ql-rest-api/index.html#_request_parameters ",
		},
		IsUser: true,
	},
	E_SERVICE_TIMEOUT: {
		Code:        1080,
		ErrorCode:   "E_SERVICE_TIMEOUT",
		Description: "Timeout [duration] exceeded",
		Causes: []string{
			"request level timeout has been hit, hence the request is timedout",
		},
		Actions: []string{
			"change \"timeout\" request parameter to a higher value",
		},
		IsUser: true,
	},
	E_SERVICE_INVALID_VALUE: {
		Code:        1090,
		ErrorCode:   "E_SERVICE_INVALID_VALUE",
		Description: "[parameter] = [value] is invalid. [message]",
		Causes: []string{
			"The named paramer's value is invalid for the reason noted in the message.",
		},
		Actions: []string{
			"Set the parameter to a valid value.  For example, don't set or set \"readonly\" to true when submitting a request using the GET method.",
		},
		IsUser: true,
	},
	E_SERVICE_INVALID_JSON: {
		Code:        1100,
		ErrorCode:   "E_SERVICE_INVALID_JSON",
		Description: "Invalid JSON in results",
		Causes: []string{
			"error in logic to write the error to the output buffer(usually stdout), actual query processing has completed only errored out while try to write out the results.",
		},
		Actions: []string{
			"please contact support",
		},
		IsUser: false,
	},
	E_SERVICE_CLIENTID: {
		Code:        1110,
		ErrorCode:   "E_SERVICE_CLIENTID",
		Description: "forbidden character (\\\\ or \\\") in client_context_id",
		Causes: []string{
			"request has client_context_id set to a value that has \" or \\ which is disallowed as it is not vaild json character",
		},
		Actions: []string{
			"change your client_context_id to something that doesn't have the characters \" or \\",
		},
		IsUser: true,
	},
	E_SERVICE_MEDIA_TYPE: {
		Code:        1120,
		ErrorCode:   "E_SERVICE_MEDIA_TYPE",
		Description: "Unsupported media type:[mediaType]",
		Causes: []string{
			"allowed value for Accept: request header is not */* or \"application/json\", we only support response of json / xml for now",
		},
		Actions: []string{
			"change \"accept\" header field to \"application/json\"",
		},
		IsUser: true,
	},
	E_SERVICE_HTTP_REQ: {
		Code:        1130,
		ErrorCode:   "E_SERVICE_HTTP_REQ",
		Description: "",
		Causes:      []string{},
		Actions:     []string{},
	},
	E_SERVICE_SCAN_VECTOR_BAD_LENGTH: {
		Code:        1140,
		ErrorCode:   "E_SERVICE_SCAN_VECTOR_BAD_LENGTH",
		Description: "Array [scan_entry] should be of length 2",
		Causes: []string{
			"when passing entries in the scan_vector or scan_vectors parameter-> entry must be [value:vbucket_seqno, guard:vbucket_uuid]. This applies for both fullscan vector and sparse scan vector.",
		},
		Actions: []string{
			"correct the scan entry in your scan vector to [vbucket_seqno:JSON-number, vbucket_uuid:string]",
		},
		IsUser: true,
	},
	E_SERVICE_SCAN_VECTOR_BAD_SEQUENCE_NUMBER: {
		Code:        1150,
		ErrorCode:   "E_SERVICE_SCAN_VECTOR_BAD_SEQUENCE_NUMBER",
		Description: "Bad sequence number [vbucket_seqno input]. Expected an unsigned 64-bit integer.",
		Causes: []string{
			"entry in the scan_vector parameter is a not an unsigned-integer.",
		},
		Actions: []string{
			"correct vbsequence number , maybe use cbstat tool for this https://docs.couchbase.com/server/7.1/cli/cbstats/cbstats-vbucket-seqno.html#example ",
		},
		IsUser: true,
	},
	E_SERVICE_SCAN_VECTOR_BADUUID: {
		Code:        1155,
		ErrorCode:   "E_SERVICE_SCAN_VECTOR_BADUUID",
		Description: "Bad UUID [vbucket_uuid]. Expected a string.",
		Causes: []string{
			"scan_vector has an entry with [vbucket_seqno, vbucket_uuid]-> uuid as non-string value",
		},
		Actions: []string{
			"correct the vbuuid, maybe use cbstat tool for this https://docs.couchbase.com/server/7.1/cli/cbstats/cbstats-vbucket-seqno.html#example ",
		},
		IsUser: true,
	},
	E_SERVICE_DECODE_NIL: {
		Code:        1160,
		ErrorCode:   "E_SERVICE_DECODE_NIL",
		Description: "Failed to decode nil value.",
		Causes: []string{
			"server got a nil request body. 1) for POST /admin/settings 2) POST /admin/clusters 3) POST /admin/clusters/{clusters}/nodes",
		},
		Actions: []string{
			"retry request with a non-nil request body, ",
		},
		IsUser: true,
	},
	E_SERVICE_HTTP_METHOD: {
		Code:        1170,
		ErrorCode:   "E_SERVICE_HTTP_METHOD",
		Description: "Unsupported method [request-method]",
		Causes: []string{
			"endpoint you are trying to send your request to doesn't support the request method specified",
		},
		Actions: []string{
			"documentation link for N1QL API support https://docs.couchbase.com/server/current/n1ql/n1ql-rest-api/index.html ",
		},
		IsUser: true,
	},
	E_SERVICE_SHUTTING_DOWN: {
		Code:        1180,
		ErrorCode:   "E_SERVICE_SHUTTING_DOWN",
		Description: "INTERNAL USE",
		Causes: []string{
			"1) redundant shutdown request from orchestrator during removal rebalance during service shutdown,",
			"2) if partial graceful shutdown feature is not supported any request to service during shutdown is errored out with this error during service shutdown",
		},
		Actions: []string{},
	},
	E_SERVICE_SHUT_DOWN: {
		Code:        1181,
		ErrorCode:   "E_SERVICE_SHUT_DOWN",
		Description: "INTERNAL USE",
		Causes: []string{
			"1) redundant shutdown request from orchestrator during removal rebalabce after service has shutdown,",
			"2) if partial graceful shutdown is not supported any request after shutdown is errored out with this error during service shutdown.",
		},
		Actions: []string{},
	},
	E_SERVICE_UNAVAILABLE: {
		Code:        1182,
		ErrorCode:   "E_SERVICE_UNAVAILABLE",
		Description: "Service cannot handle requests",
		Causes: []string{
			"/admin/ping endpoint to determine if server is healthy. That is neither unbounded queue(this is for request with scan_consistency = not_bounded) nor plus queue(request scan_consistency is at_plus/request_plus/statement_plus) is full and cannot respond to anymore requests, this is because the queueCount(number of request in the runninung queue) is greater the fullqueue(running queue capacity).",
		},
		Actions: []string{
			"Try checking in active_requests(SELECT * FROM system:active_requests) and optionally check for scan_consistency, for eg:  WHERE scan_consistency=\"unbounded\". \nWait till the number of documents in active requests comedown.",
		},
		IsUser: false,
	},
	E_SERVICE_USER_REQUEST_EXCEEDED: {
		Code:        1191,
		ErrorCode:   "E_SERVICE_USER_REQUEST_EXCEEDED",
		Description: "UNUSED- pending changes from ns_server for free-tier(from commit message)",
		Causes:      []string{},
		Actions:     []string{},
	},
	E_SERVICE_USER_REQUEST_RATE_EXCEEDED: {
		Code:        1192,
		ErrorCode:   "E_SERVICE_USER_REQUEST_RATE_EXCEEDED",
		Description: "UNUSED- pending changes from ns_server for free-tier(from commit message)",
		Causes:      []string{},
		Actions:     []string{},
	},
	E_SERVICE_USER_REQUEST_SIZE_EXCEEDED: {
		Code:        1193,
		ErrorCode:   "E_SERVICE_USER_REQUEST_SIZE_EXCEEDED",
		Description: "UNUSED- pending changes from ns_server for free-tier(from commit message)",
		Causes:      []string{},
		Actions:     []string{},
	},
	E_SERVICE_USER_RESULT_SIZE_EXCEEDED: {
		Code:        1194,
		ErrorCode:   "E_SERVICE_USER_RESULT_SIZE_EXCEEDED",
		Description: "UNUSED- pending changes from ns_server for free-tier(from commit message)",
		Causes:      []string{},
		Actions:     []string{},
	},
	E_REQUEST_ERROR_LIMIT: {
		Code:        1195,
		ErrorCode:   "E_REQUEST_ERROR_LIMIT",
		Description: "Request execution aborted as the number of errors raised has reached the maximum permitted.",
		Causes: []string{
			"Possible occurence indicates errors in DML statements(DELETE /INSERT /UPDATE/ UPSERT) on a keyspace. This maybe due to CASmismatch at the document level due to concurrent modification request.",
		},
		Actions: []string{
			"The error limit is configurable per request level by using the \"error_limit\" request parameter. Change it to a higher value.",
		},
		IsUser: false,
	},
	E_SERVICE_TENANT_THROTTLED: {
		Code:        1196,
		ErrorCode:   "E_SERVICE_TENANT_THROTTLED",
		Description: "",
		Causes:      []string{},
		Actions:     []string{},
	},
	E_SERVICE_TENANT_MISSING: {
		Code:        1197,
		ErrorCode:   "E_SERVICE_TENANT_MISSING",
		Description: "",
		Causes:      []string{},
		Actions:     []string{},
	},
	E_SERVICE_TENANT_NOT_AUTHORIZED: {
		Code:        1198,
		ErrorCode:   "E_SERVICE_TENANT_NOT_AUTHORIZED",
		Description: "",
		Causes:      []string{},
		Actions:     []string{},
	},
	E_SERVICE_TENANT_REJECTED: {
		Code:        1199,
		ErrorCode:   "E_SERVICE_TENANT_REJECTED",
		Description: "",
		Causes:      []string{},
		Actions:     []string{},
	},
	E_SERVICE_TENANT_NOT_FOUND: {
		Code:        1200,
		ErrorCode:   "E_SERVICE_TENANT_NOT_FOUND",
		Description: "",
		Causes:      []string{},
		Actions:     []string{},
	},
	E_SERVICE_REQUEST_QUEUE_FULL: {
		Code:        1201,
		ErrorCode:   "E_SERVICE_REQUEST_QUEUE_FULL",
		Description: "Request queue full",
		Causes: []string{
			"the request is stopped as the request queue is full, note that we maintain 2 separate queues for unbounded requests and plus requests.",
		},
		Actions: []string{
			"Try checking in active_requests(SELECT * FROM system:active_requests) and optionally check for scan_consistency, for eg:  WHERE scan_consistency=\"unbounded\". \nWait till the number of documents in active requests comedown.",
		},
		IsUser: false,
	},
	E_SERVICE_NO_CLIENT: {
		Code:        1202,
		ErrorCode:   "E_SERVICE_NO_CLIENT",
		Description: "Client disconnected",
		Causes: []string{
			"Server aborts servicing a request if a client has aborted the request. This is done when client closes its connection(disconnects / cancel request)",
		},
		Actions: []string{
			"Contact support",
		},
		IsUser: true,
	},
	E_ADMIN_CONNECTION: {
		Code:        2000,
		ErrorCode:   "E_ADMIN_CONNECTION",
		Description: "Error connecting to  [msg]",
		Causes: []string{
			"1) When failing to connect to configstore, this error is logged(not returned).",
			"2) GET /admin/config internally tries to get poolServices from /pool/default/nodeServices",
			"3) GET /admin/clusters/{clusters}/nodes/{node} & GET /admin/clusters/{clusters}/nodes internally tries to get handle of the couchbase configstore connection but fails, that is when we raise this error.",
		},
		Actions: []string{
			"",
		},
	},
	E_ADMIN_START: {
		Code:        2001,
		ErrorCode:   "E_ADMIN_START",
		Description: "Error accounting manager: Fail to open sigar.",
		Causes: []string{
			"1) while initailizing Accounting store on query-service startup we have encountered an error while opening handle to account for system stats",
		},
		Actions: []string{
			"1) Contact Support",
		},
		IsUser: false,
	},
	E_ADMIN_INVALIDURL: {
		Code:        2010,
		ErrorCode:   "E_ADMIN_INVALIDURL",
		Description: "Invalid [component] URL: [URL]",
		Causes: []string{
			"1) during startup Logger URL is invalid, 2) accounting store URL is invalid, 3) configstore URL is invalid",
		},
		Actions: []string{
			"1) URL prefix for logger-> golog / builtin / file / null , 2) URL scheme/protocol for accounting store-> gometrics: , 3) URL scheme/protocol for configstore-> http: , if encountered on source build https://github.com/couchbase/tlm trying changing URL to supported format, else if contact support.",
		},
		IsUser: false,
	},
	E_ADMIN_DECODING: {
		Code:        2020,
		ErrorCode:   "E_ADMIN_DECODING",
		Description: "Error in JSON decoding",
		Causes: []string{
			"failed to decode Basic authorization 1) when trying to profile using /debug/pprof/ or /debug/pprof/profile endpoint. 2) Similarly for /debug/pprof/block , /debug/pprof/goroutine , /debug/pprof/threadcreate, /debug/pprof/heap, /debug/pprof/mutex 3) when trying to request /admin/settings endpoint.\n\nFailed to write api's returned object to the http response stream , for all admin -api's https://docs.couchbase.com/server/current/n1ql/n1ql-rest-api/admin.html",
		},
		Actions: []string{
			"Contact Support",
		},
	},
	E_ADMIN_ENCODING: {
		Code:        2030,
		ErrorCode:   "E_ADMIN_ENCODING",
		Description: "Error in JSON encoding - Only raised under zookeeper clustering",
		Causes:      []string{},
		Actions:     []string{},
	},
	E_ADMIN_UNKNOWN_SETTING: {
		Code:        2031,
		ErrorCode:   "E_ADMIN_UNKNOWN_SETTING",
		Description: "Unknown setting: [setting]",
		Causes: []string{
			"POST request to /admin/settings to change node level settings https://docs.couchbase.com/server/current/settings/query-settings.html has a setting in the request body which is not recognized.\n\nExample: request: POST /admin/settings HTTP/1.1\nAuthorization: Basic QWRtaW5pc3RyYXRvcjpwYXNzd29yZA==\nContent-Length: 18\nContent-Type: application/json\nHost: 127.0.0.1:9499\n{\n  \"cleanup\": 1\n}\n\nresponse:\n{\n\"_level\": \"exception\",\n\"caller\": \"set_params:405\",\n\"code\": 2031,\n\"key\": \"admin.unknown_setting\",\n\"message\": \"Unknown setting: cleanup\"\n}",
		},
		Actions: []string{
			"Find the allowed settings here https://docs.couchbase.com/server/current/settings/query-settings.html , and correct the unrecognized setting ",
		},
		IsUser: true,
	},
	E_ADMIN_SETTING_TYPE: {
		Code:        2032,
		ErrorCode:   "E_ADMIN_SETTING_TYPE",
		Description: "Incorrect value for setting",
		Causes: []string{
			"1) POST request to /admin/settings endpoint, request body has valid settings as key but value for one of them is not the allowed type. For example loglevel setting expects string but we have passed a number\nrequest:\nPOST /admin/settings HTTP/1.1\nAuthorization: Basic QWRtaW5pc3RyYXRvcjpwYXNzd29yZA==\nContent-Length: 18\nContent-Type: application/json\nHost: 127.0.0.1:9499\n{\n  \"loglevel\":1\n}\n\n\nresponse:\nHTTP/1.1 500 Internal Server Error\nContent-Type: application/json\nDate: Tue, 21 Nov 2023 09:23:44 GMT\nContent-Length: 155\n{\"_level\":\"exception\",\"caller\":\"set_params:409\",\"code\":2032,\"key\":\"admin.setting_type_error\",\"message\":\"Incorrect value '1' (int64) for setting: loglevel\"}",
			"2) \"completed\" setting allows tagged set https://docs.couchbase.com/server/current/manage/monitor/monitoring-n1ql-query.html#tagged-sets which expects a string as value but user has passed a non-string value.",
			"3) \"completed\" setting expects an object",
			"4) \"atrcollection\" setting expects a string that is a valid n1ql path( bucket / bucket.scope.collection / namespace:bucket.scope.collection )",
		},
		Actions: []string{
			"Find the allowed settings here https://docs.couchbase.com/server/current/settings/query-settings.html  , and correct the setting the mismatched the value. ",
		},
		IsUser: true,
	},
	E_ADMIN_GET_CLUSTER: {
		Code:        2040,
		ErrorCode:   "E_ADMIN_GET_CLUSTER",
		Description: "Error retrieving cluster ",
		Causes: []string{
			"1) for the enpoints /admin/clusters/{cluster}  , /admin/clusters/{cluster}/nodes & /admin/clusters/{cluster}/nodes/{node}, no cluster by the name provided by the path parameter exists in the config store.",
			"2) when user tries to GET admin/config , invalid response from /pools/{poolname} to get pool data or /pools/{poolname}/nodeServices to get services data, usually the poolName passed is \"default\".",
		},
		Actions: []string{
			"1) provide a valid cluster name",
			"2) please contact support, or try GET /pools/default endpoint at 8091 port(orchestrator process) to confirm if it is a query issue and not a cluster-wide issue.",
		},
		IsUser: true,
	},
	E_ADMIN_ADD_CLUSTER: {
		Code:        2050,
		ErrorCode:   "E_ADMIN_ADD_CLUSTER",
		Description: "Error adding cluster - Only raised under zookeeper clustering",
		Causes:      []string{},
		Actions:     []string{},
	},
	E_ADMIN_REMOVE_CLUSTER: {
		Code:        2060,
		ErrorCode:   "E_ADMIN_REMOVE_CLUSTER",
		Description: "Error removing cluster - Only raised under zookeeper clustering",
		Causes:      []string{},
		Actions:     []string{},
	},
	E_ADMIN_GET_NODE: {
		Code:        2070,
		ErrorCode:   "E_ADMIN_GET_NODE",
		Description: "Error retrieving node ",
		Causes: []string{
			"1) user has tried to query( SELECT * / SELECT COUNT(*) ) system:nodes keyspace which internally gets node/topology( information from /pools/default/nodeServices endpoint, but response doesn't include mgmt port(management port). Which is typically 8091.",
			"2) user has set GET /admin/config , and we try to return SQLclustering topolgy information such as service ip-address and \"queryEndpoint\" , \"adminEndpoint\", \"querySecure\", \"adminSecure\" URIs. But again /pools/default or /pools/default/nodeServices endpoint return erroreous response maybe adminConnection error, etc.",
			"3) /admin/clusters/{clusters}/nodes endpoint failed to gather nodes information for the pool {cluster} from configstore. Similar scenario may occur for /admin/clusters/{clusters}/nodes?uuid=<uuid>",
		},
		Actions: []string{
			"the orchestrator(ns_server) process's endpoint /pools/default & /pools/default/nodeServices",
		},
		IsUser: true,
	},
	E_ADMIN_NO_NODE: {
		Code:        2080,
		ErrorCode:   "E_ADMIN_NO_NODE",
		Description: "No such node <message>",
		Causes: []string{
			"1) admin/config , admin/clusters/{cluster}/nodes , admin/clusters/{cluster}/nodes/{node} endpoint internally lookup node information from /pool/default/nodeServices (orchestrator endpoint) and use that to get config from the sql clustering manager. But node name from /pools/default/nodeServices doesn't match one from the config store.",
		},
		Actions: []string{
			"Contact support",
		},
		IsUser: true,
	},
	E_ADMIN_ADD_NODE: {
		Code:        2090,
		ErrorCode:   "E_ADMIN_ADD_NODE",
		Description: "Error adding node  - Only raised under zookeeper clustering",
		Causes:      []string{},
		Actions:     []string{},
	},
	E_ADMIN_REMOVE_NODE: {
		Code:        2100,
		ErrorCode:   "E_ADMIN_REMOVE_NODE",
		Description: "Error removing node -  Only raised under zookeeper clustering",
		Causes:      []string{},
		Actions:     []string{},
	},
	E_ADMIN_MAKE_METRIC: {
		Code:        2110,
		ErrorCode:   "E_ADMIN_MAKE_METRIC",
		Description: "Error creating metric -> internal for testing purpose, as we use GetOrRegister() instead of Register() for metrics in the accounting package",
		Causes:      []string{},
		Actions:     []string{},
	},
	E_ADMIN_AUTH: {
		Code:        2120,
		ErrorCode:   "E_ADMIN_AUTH",
		Description: "Error authorizing against cluster",
		Causes: []string{
			"1) authentication (not authorization) for a request->  user provided credentials( either 1. Basic Auth, 2. Auth Header/Token, 3. Certificates, 4. Creds query parameter), provided credentials don't match authenticated user credentials on the datastore auth handler(usually cbauth).",
			"2) request with no auth credentials",
			"3) when trying to read or write settins to /admin/settings endpoint, user must have cluster.settings!read privilege if GET , else POST then cluster.settings!write.",
		},
		Actions: []string{
			"Please ensure your users credential(username:password is authenticated) , if not contact support.\nIf an admin-authorization give the user the respective persmission. doc reference here https://docs.couchbase.com/server/7.1/manage/manage-security/manage-users-and-roles.html , ideally a full-admin / cluster-admin would be sufficient for most admin related authorization",
		},
		IsUser: true,
	},
	E_ADMIN_ENDPOINT: {
		Code:        2130,
		ErrorCode:   "E_ADMIN_ENDPOINT",
		Description: "",
		Causes:      []string{},
		Actions:     []string{},
	},
	E_ADMIN_SSL_NOT_ENABLED: {
		Code:        2140,
		ErrorCode:   "E_ADMIN_SSL_NOT_ENABLED",
		Description: "server is not ssl enabled",
		Causes: []string{
			"user has sent a  POST /admin/ssl_cert but the service endpoint doesn't support https.",
		},
		Actions: []string{},
	},
	E_ADMIN_CREDS: {
		Code:        2150,
		ErrorCode:   "E_ADMIN_CREDS",
		Description: "UNUSED",
		Causes:      []string{},
		Actions:     []string{},
	},
	E_COMPLETED_QUALIFIER_EXISTS: {
		Code:        2160,
		ErrorCode:   "E_COMPLETED_QUALIFIER_EXISTS",
		Description: "Completed requests qualifier already set: [qualifier name]",
		Causes: []string{
			"user has tried to add a qualifier using the completed object in POST request to /admin/settings but the qualifier is already been set before.",
		},
		Actions: []string{
			"to update a qualifier, don't use '+' prefix -> for eg: curl http://localhost:8093/admin/settings -u Administrator:password \\\n  -H 'Content-Type: application/json' \\\n  -d '{\"completed\": {\"user\": \"marco\"}\n\ndocumentation link to logging qualifiers https://docs.couchbase.com/server/current/manage/monitor/monitoring-n1ql-query.html#logging-qualifiers ",
		},
		IsUser: true,
	},
	E_COMPLETED_QUALIFIER_UNKNOWN: {
		Code:        2170,
		ErrorCode:   "E_COMPLETED_QUALIFIER_UNKNOWN",
		Description: "Completed requests qualifier unknown: [qualifier name]",
		Causes: []string{
			"user has tried to add a qualifier that is not recognized by the server in the completed object as a part of the POST request to /admin/settings. \n\nfor eg:\nPOST /admin/settings HTTP/1.1\nAuthorization: Basic QWRtaW5pc3RyYXRvcjpwYXNzd29yZA==\nContent-Length: 16\nHost: 127.0.0.1:9499\n{\n\"users\":\"GA\"\n}\n\nHTTP/1.1 500 Internal Server Error\nContent-Type: application/json\nDate: Wed, 22 Nov 2023 11:17:08 GMT\nContent-Length: 125\n{\"_level\":\"exception\",\"caller\":\"set_params:405\",\"code\":2031,\"key\":\"admin.unknown_setting\",\"message\":\"Unknown setting: users\"}",
		},
		Actions: []string{
			"list of valid logging qualifiers https://docs.couchbase.com/server/current/manage/monitor/monitoring-n1ql-query.html#logging-qualifiers ",
		},
		IsUser: true,
	},
	E_COMPLETED_QUALIFIER_NOT_FOUND: {
		Code:        2180,
		ErrorCode:   "E_COMPLETED_QUALIFIER_NOT_FOUND",
		Description: "Completed requests qualifier not set: [qualifier name]",
		Causes: []string{
			"1) removing a previously unset logging qualifier, eg:\nPOST /admin/settings HTTP/1.1\nAuthorization: Basic QWRtaW5pc3RyYXRvcjpwYXNzd29yZA==\nContent-Length: 35\nHost: 127.0.0.1:9499\n{\n\"completed\": {\"-user\": \"marco\"}\n}\n\nHTTP/1.1 500 Internal Server Error\nContent-Type: application/json\nDate: Wed, 22 Nov 2023 11:39:30 GMT\nContent-Length: 174\n{\"_level\":\"exception\",\"caller\":\"completed_requests:304\",\"code\":2180,\"key\":\"admin.accounting.completed.not_found\",\"message\":\"Completed requests qualifier not set: user marco\"}",
		},
		Actions: []string{
			"GET /admin/settings, \nlook at the completed field for existing qualifiers, when removing the value must match.\n\nfor eg: \nfrom the response\n\"completed\":[\n{\n\"client\": \"172.1.2.3\",\n\"tag\": \"both_user_and_error\"\n},\n{\n\"aborted\": null,\n\"client\": \"172.1.2.2\",\n\"seqscan_keys\": 10000,\n\"threshold\": 1000,\n\"user\": \"parco\"\n}\n\nto remove client:\n{\"completed\": {\"-client\":\"172.1.2.2\"}}\n",
		},
		IsUser: true,
	},
	E_COMPLETED_QUALIFIER_NOT_UNIQUE: {
		Code:        2190,
		ErrorCode:   "E_COMPLETED_QUALIFIER_NOT_UNIQUE",
		Description: "Non-unique completed requests qualifier [qualifier] cannot be updated",
		Causes: []string{
			"user request to /admin/settings includes a completed object that is trying to update non-unique qualifier( error, client, user, context)\nfor eg:\nPOST /admin/settings HTTP/1.1\nAuthorization: Basic QWRtaW5pc3RyYXRvcjpwYXNzd29yZA==\nContent-Length: 31\nContent-Type: application/json\nHost: 127.0.0.1:9499\n{\"completed\": {\"user\":\"parco\"}}\n\nHTTP/1.1 500 Internal Server Error\nContent-Type: application/json\nDate: Wed, 22 Nov 2023 11:49:53 GMT\nContent-Length: 192\n{\"_level\":\"exception\",\"caller\":\"completed_requests:294\",\"code\":2190,\"key\":\"admin.accounting.completed.not_unique\",\"message\":\"Non-unique completed requests qualifier 'user' cannot be updated.\"}",
		},
		Actions: []string{
			"can only update unique qualifiers-> threshold , aborted\n\nfor others remove the qualifier with '-' prefix \nand add with a new value using '+' prefix\n\nsomething like, \n/admin/settings :{\"completed\": {\"-user\":\"marco\"}} \n/admin/settings :{\"completed\": {\"+user\":\"donald\"}}",
		},
		IsUser: true,
	},
	E_COMPLETED_QUALIFIER_INVALID_ARGUMENT: {
		Code:        2200,
		ErrorCode:   "E_COMPLETED_QUALIFIER_INVALID_ARGUMENT",
		Description: "Completed requests qualifier [qualifier name] cannot accept argument  [user-input]",
		Causes: []string{
			"The scenario here is that when adding a request logging qualifier in a request to /admin/settings , the expected type for the logging qualifier doesn't match the input type.\n\nfor eg: user expects string but you have passed a number\nPOST /admin/settings HTTP/1.1\nAuthorization: Basic QWRtaW5pc3RyYXRvcjpwYXNzd29yZA==\nContent-Length: 26\nContent-Type: application/json\nHost: 127.0.0.1:9499\n{\"completed\": {\"+user\":1}}\n\nHTTP/1.1 500 Internal Server Error\nContent-Type: application/json\nDate: Thu, 23 Nov 2023 06:47:53 GMT\nContent-Length: 184\n{\"_level\":\"exception\",\"caller\":\"completed_requests:861\",\"code\":2200,\"key\":\"admin.accounting.completed.invalid\",\"message\":\"Completed requests qualifier error cannot accept argument  1\"}",
		},
		Actions: []string{
			"expected input for the request qualifiers \nuser-> string\nthreshold -> number\ncontext -> string\nclient -> string",
		},
		IsUser: true,
	},
	E_COMPLETED_BAD_MAX_SIZE: {
		Code:        2201,
		ErrorCode:   "E_COMPLETED_BAD_MAX_SIZE",
		Description: "UNUSED- but logic is to avoid adding in the plan by limiting the max plan size allowed to be logged. https://review.couchbase.org/c/query/+/194204 ",
		Causes:      []string{},
		Actions:     []string{},
	},
	E_ADMIN_BAD_SERVICE_PORT: {
		Code:        2210,
		ErrorCode:   "E_ADMIN_BAD_SERVICE_PORT",
		Description: "Invalid service port: [portnumber]",
		Causes: []string{
			"on startup we set config store options( http_address, https_address) but invalid portno is passed for the configstore address, that is a number<0.",
		},
		Actions: []string{
			"contact support if running cbserver from a binary, if you have a src build, for the -configstore flag has a valid port number(typically http-8093, https-18093) for query service.",
		},
		IsUser: false,
	},
	E_ADMIN_BODY: {
		Code:        2220,
		ErrorCode:   "E_ADMIN_BODY",
		Description: "LEGACY-  Error getting request body",
		Causes: []string{
			"/admin/prepareds/{name} endpoint allows PUT request with request body as encoded plan to setup a correspnding prepared statement for the plan passed. But something went wrong in reading the request body sent.",
		},
		Actions: []string{
			"contact support",
		},
		IsUser: true,
	},
	E_ADMIN_FFDC: {
		Code:        2230,
		ErrorCode:   "E_ADMIN_FFDC",
		Description: "FFDC invocation failed.",
		Causes: []string{
			"POST /admin/ffdc to start \"manual\" ffdc(first fault data capture) is done before the allowed interval between consequtive attempts to start the invoation. The default interval time is 10sec.\nFor eg: consequtive request before 10seconds pass from the first request\nPOST /admin/ffdc HTTP/1.1\nAuthorization: Basic QWRtaW5pc3RyYXRvcjpwYXNzd29yZA==\nContent-Length: 0\nContent-Type: application/json\nHost: 127.0.0.1:9499\n\nHTTP/1.1 500 Internal Server Error\nContent-Type: application/json\nDate: Thu, 23 Nov 2023 09:05:47 GMT\nContent-Length: 225\n{\"_level\":\"exception\",\"caller\":\"admin_accounting_endpoint:2475\",\"cause\":{\"message\":\"Ensure sufficient interval between invocations.\",\"seconds_before_next\":6},\"code\":2230,\"key\":\"admin.ffdc\",\"message\":\"FFDC invocation failed.\"}",
		},
		Actions: []string{
			"wait for 10sec before starting ffdc ",
		},
		IsUser: true,
	},
	E_ADMIN_LOG: {
		Code:        2240,
		ErrorCode:   "E_ADMIN_LOG",
		Description: "Error accessing log",
		Causes: []string{
			"GET request to /admin/log/{file} failed as 1) {file} doesn't exist in log-path 2) {file} was deleted whilst being read.",
		},
		Actions: []string{
			"Ensure the log file being accessed exists for the duration of the request.",
		},
		IsUser: true,
	},
	E_PARSE_SYNTAX: {
		Code:        3000,
		ErrorCode:   "E_PARSE_SYNTAX",
		Description: "Indicates a syntax error occurred during parsing.  The error details will be contained in the cause field.",
		Causes: []string{
			"A syntax error is present in a statement being parsed.",
		},
		Actions: []string{},
	},
	E_ERROR_CONTEXT: {
		Code:        3005,
		ErrorCode:   "E_ERROR_CONTEXT",
		Description: "Generic error-> that lets the user know where(line and column cursor) his query is syntactically wrong",
		Causes: []string{
			"During the Parsing phase, the tokens from the scanner are reduced to a particluar parser rule(Grammar) but for the particular use case of",
		},
		Actions: []string{
			"some examples: 1) query= SELECT 1 FROM `default`, 1 USE KEYS \"a\"; error:\"FROM Expression cannot have USE KEYS or USE INDEX (near line 1, column 26).\", what is message is saying is that if FROM term is not a collection/bucket cannot have USE KEYS or USE INDEX CLAUSE(s).  The safe bet would be to ask on https://www.couchbase.com/forums/ under \"n1ql\" or \"query\" tag",
		},
		IsUser: true,
	},
	E_PARSE_INVALID_ESCAPE_SEQUENCE: {
		Code:        3006,
		ErrorCode:   "E_PARSE_INVALID_ESCAPE_SEQUENCE",
		Description: "invalid escape sequence",
		Causes: []string{
			"An invalid escape sequence was encountered whilst processing a string value.",
		},
		Actions: []string{
			"Refer to the documentation for valid string escape sequences",
		},
		IsUser: true,
	},
	E_PARSE_INVALID_STRING: {
		Code:        3007,
		ErrorCode:   "E_PARSE_INVALID_STRING",
		Description: "An invalid string was encounterd.",
		Causes: []string{
			"An opening quotation mark defining a string was encountered without any further characters.  All strings must be properly quoted.",
		},
		Actions: []string{},
	},
	E_AMBIGUOUS_REFERENCE: {
		Code:        3080,
		ErrorCode:   "E_AMBIGUOUS_REFERENCE",
		Description: "Ambiguous reference to field",
		Causes: []string{
			"Query's project field is ambiguous. Note we are schemaless so this scenario comes to play only in 2 cases. 1) no keyspace but projection field: SELECT k; error:[\n  {\n    \"code\": 3080,\n    \"column\": 8,\n    \"line\": 1,\n    \"msg\": \"Ambiguous reference to field 'k' (near line 1, column 8).\",\n    \"query\": \"SELECT k;\"\n  }\n],  2) join query but projection does't indicate which keyspace to look at SELECT k FROM {\"a\":1} a, {\"b\":1} b WHERE a.a=b.b; error:[\n  {\n    \"code\": 3080,\n    \"column\": 8,\n    \"line\": 1,\n    \"msg\": \"Ambiguous reference to field 'k' (near line 1, column 8).\",\n    \"query\": \"SELECT k FROM {\\\"a\\\":1} a, {\\\"b\\\":1} b WHERE a.a=b.b;\"\n  }\n]. Just to note the schemaless point SELECT a.k FROM {\"a\":1} a, {\"b\":1} b WHERE a.a=b.b; returns empty result without error as keyspace a doesn't have a document with `k` field after join.",
		},
		Actions: []string{
			"if 1) please recheck if the query without keyspace is what you meant. 2) the projection field must have the source keyspace attached (Formalized by hand) in case Join",
		},
		IsUser: true,
	},
	E_DUPLICATE_VARIABLE: {
		Code:        3081,
		ErrorCode:   "E_DUPLICATE_VARIABLE",
		Description: "Duplicate variable, already in the scope as allowed identifier",
		Causes: []string{
			"1) When using LET CLAUSE and have variables with same name-> SELECT t1.airportname, t1.geo.lat, t1.geo.lon, t1.city, t1.type\nFROM `travel-sample`.inventory.airport t1\nLET min_lat = 71, min_lat = ABS(t1.geo.lon)*4+1;\nWHERE WHERE t1.geo.lat > min_lat\nAND t1.geo.lat < max_lat; error: msg\": \"Duplicate variable: 'min_lat' already in scope (near line 3, column 19).\",  2) When using WITH Clause, 2 or more ctes have the same name WITH a AS (SELECT 1), a AS (SELECT 2) SELECT 1; error: \"msg\": \"Duplicate WITH clause alias 'a' (near line 1, column 23)\"",
		},
		Actions: []string{
			"Rename your duplicate LET variable or duplicate CTE",
		},
		IsUser: true,
	},
	E_FORMALIZER_INTERNAL: {
		Code:        3082,
		ErrorCode:   "E_FORMALIZER_INTERNAL",
		Description: "Formalizer internal error",
		Causes: []string{
			"This is raised in particular for a case where encoded_plan is used for subquery and subquery is marked as correlated, for eg: PREPARE test AS SELECT d1.a FROM `default` d1 WHERE d1.a in (SELECT RAW d2.a FROM default d2 WHERE d2.a=d1.a)\n;  Note in the plan:-> {\n                      \"#operator\": \"Filter\",\n                      \"condition\": \"(cover ((`d1`.`a`)) in correlated (select raw cover ((`d2`.`a`)) from `default`:`default` as `d2` where (cover ((`d2`.`a`)) = cover ((`d1`.`a`)))))\",\n                      \"optimizer_estimates\": {\n                        \"cardinality\": 1000.9999999999998,\n                        \"cost\": 751.1415658002394,\n                        \"fr_cost\": 12.738403162637601,\n                        \"size\": 105\n                      }\n                    }. By some magic someone has gone into your node and changed  where (cover ((`d2`.`a`)) = cover ((`d1`.`a`) to  where (cover ((`d2`.`a`)) = cover ((`d1`.`b`). You expect the error that unexpected correlated reference d2.b is not allowed.",
		},
		Actions: []string{
			"Your solution to is reprepare your original query and use the new prepare",
		},
		IsUser: true,
	},
	E_PARSE_INVALID_INPUT: {
		Code:        3083,
		ErrorCode:   "E_PARSE_INVALID_INPUT",
		Description: "missing closing quote",
		Causes: []string{
			"1) when trying to parse a string expression, didn't find closing quote \" eg: SELECT a.* FROM {\"a\":\"} a;  , eg: SELECT a.* FROM {\"a\":\"\\\"} a; (Misuse of escaping \\)",
		},
		Actions: []string{
			"Correct the usage by adding closing quote at the appropriate place, or ask on ask on https://www.couchbase.com/forums/ under \"n1ql\" or \"query\" tag ",
		},
		IsUser: true,
	},
	E_SEMANTICS: {
		Code:        3100,
		ErrorCode:   "E_SEMANTICS",
		Description: "Wrapper error-> please check cause for actual error message",
		Causes:      []string{},
		Actions:     []string{},
	},
	E_JOIN_NEST_NO_JOIN_HINT: {
		Code:        3110,
		ErrorCode:   "E_JOIN_NEST_NO_JOIN_HINT",
		Description: "cannot have join hint (USE HASH or USE NL)",
		Causes: []string{
			"This is raised when user uses Lookup Join https://docs.couchbase.com/server/current/n1ql/n1ql-language-reference/join.html#lookup-join-clause , as the working of lookup join solely depends on ON KEYS Clause, so at the semantic layer we disallow join hint (USE HASH/ USE NL) , USE KEYS, USE INDEX. Also the case for LOOKUP NEST",
		},
		Actions: []string{
			"Prefer ANSI or INDEX JOIN for join HINTS/ Use KEYS / USE INDEX HINTS",
		},
		IsUser: true,
	},
	E_JOIN_NEST_NO_USE_KEYS: {
		Code:        3120,
		ErrorCode:   "E_JOIN_NEST_NO_USE_KEYS",
		Description: "cannot have USE KEYS",
		Causes: []string{
			",,",
		},
		Actions: []string{},
		IsUser:  true,
	},
	E_JOIN_NEST_NO_USE_INDEX: {
		Code:        3130,
		ErrorCode:   "E_JOIN_NEST_NO_USE_INDEX",
		Description: "cannot have USE INDEX",
		Causes: []string{
			",,",
		},
		Actions: []string{},
		IsUser:  true,
	},
	E_MERGE_INSERT_NO_KEY: {
		Code:        3150,
		ErrorCode:   "E_MERGE_INSERT_NO_KEY",
		Description: "MERGE with ON KEY clause cannot have document key specification in INSERT action.",
		Causes: []string{
			"User has used LOOKUP MERGE https://docs.couchbase.com/server/current/n1ql/n1ql-language-reference/merge.html#lookup-merge , WHEN USING INSERT as LOOKUP-MERGE-ACTION, You cannoy pass key specification. Example of a wrong query MERGE INTO hotel t\nUSING [\n  {\"id\":\"2172211\", \"vacancy\": true},\n  {\"id\":\"2173111\", \"vacancy\": true}\n] source\nON KEY \"hotel_\"|| source.id\nWHEN NOT MATCHED THEN\n  INSERT(id, {\"id\":source.id, \"vacancy\":source.vacancy, \"new\":true});    The correct query would be to change the insert clause->  INSERT {\"id\":source.id, \"vacancy\":source.vacancy, \"new\":true};",
		},
		Actions: []string{
			"Change your INSERT Clause as per LOOKUP-MERGE-INSERT https://docs.couchbase.com/server/current/n1ql/n1ql-language-reference/merge.html#lookup-merge-insert , take a look at this example for inspiration https://docs.couchbase.com/server/current/n1ql/n1ql-language-reference/merge.html#examples EXAMPLE 7",
		},
		IsUser: true,
	},
	E_MERGE_INSERT_MISSING_KEY: {
		Code:        3160,
		ErrorCode:   "E_MERGE_INSERT_MISSING_KEY",
		Description: "MERGE with ON clause must have document key specification in INSERT action",
		Causes: []string{
			"USer has used ANSI MERGE https://docs.couchbase.com/server/current/n1ql/n1ql-language-reference/merge.html#ansi-merge , WHEN USING INSERT as  ANSI-MERGE-ACTION , Need to pass key in INSERT Clause. Example of a wrong query MERGE INTO airport AS target\nUSING [\n  {\"iata\":\"DSA\", \"name\": \"Doncaster Sheffield Airport\"},\n  {\"iata\":\"VLY\", \"name\": \"Anglesey Airport / Maes Awyr M\u00f4n\"}\n] AS source\nON target.faa = source.iata\nWHEN MATCHED THEN\n  UPDATE SET target.old_name = target.airportname,\n             target.airportname = source.name,\n             target.updated = true\nWHEN NOT MATCHED THEN\n  INSERT {\"faa\": source.iata,\n                 \"airportname\": source.name,\n                 \"type\": \"airport\",\n                 \"inserted\": true} \nRETURNING *;  The correct query would be to change the insert clause-> INSERT (KEY UUID(),\n          VALUE {\"faa\": source.iata,\n                 \"airportname\": source.name,\n                 \"type\": \"airport\",\n                 \"inserted\": true} )",
		},
		Actions: []string{
			"Change your INSERT CLause as per ANSI-MERGE-INSERT https://docs.couchbase.com/server/current/n1ql/n1ql-language-reference/merge.html#ansi-merge , take a look at this example for inspiration https://docs.couchbase.com/server/current/n1ql/n1ql-language-reference/merge.html#examples EXAMPLE 4 ",
		},
		IsUser: true,
	},
	E_MERGE_MISSING_SOURCE: {
		Code:        3170,
		ErrorCode:   "E_MERGE_MISSING_SOURCE",
		Description: "MERGE is missing source.- DEAD CODE as caught by parser ??",
		Causes:      []string{},
		Actions:     []string{},
	},
	E_MERGE_NO_INDEX_HINT: {
		Code:        3180,
		ErrorCode:   "E_MERGE_NO_INDEX_HINT",
		Description: "MERGE with ON KEY clause cannot have USE INDEX hint specified on target.",
		Causes: []string{
			"Cannot pass index hints on target keyspace for LOOKUP-MERGE.  Example bad query MERGE INTO hotel t USE INDEX (def_inventory_hotel_city  USING GSI)\nUSING [\n  {\"id\":\"2172211\", \"vacancy\": true},\n  {\"id\":\"2173111\", \"vacancy\": true}\n] source\nON KEY \"hotel_\"|| source.id\nWHEN NOT MATCHED THEN\n  INSERT {\"id\":source.id, \"vacancy\":source.vacancy, \"new\":true}; -> remove USE INDEX CLAUSE",
		},
		Actions: []string{
			"Switch to ANSI MERGE if you want to use USE INDEX CLAUSE https://docs.couchbase.com/server/current/n1ql/n1ql-language-reference/merge.html#ansi-merge ",
		},
	},
	E_MERGE_NO_JOIN_HINT: {
		Code:        3190,
		ErrorCode:   "E_MERGE_NO_JOIN_HINT",
		Description: "MERGE with ON KEY clause cannot have join hint specified on source.",
		Causes: []string{
			"Cannot pass JOIN HINTS( USE HASH(BUILD) / USE HASH(PROBE) / USE NL) for LOOKUP-MERGE, as underlying JOIN is LOOKUP-JOIN. Example bad query MERGE INTO hotel t \nUSING [\n  {\"id\":\"2172211\", \"vacancy\": true},\n  {\"id\":\"2173111\", \"vacancy\": true}\n] source USE HASH(PROBE)\nON KEY \"hotel_\"|| source.id\nWHEN NOT MATCHED THEN\n  INSERT {\"id\":source.id, \"vacancy\":source.vacancy, \"new\":true}; -> if you want to use Nested loop join or Hash join shift to ANSI MERGE (NOTE: for NL need to have appropriate index)",
		},
		Actions: []string{
			"Switch to ANSI MERGE if you want to use JOIN HINTS(USE NL/USE HASH) https://docs.couchbase.com/server/current/n1ql/n1ql-language-reference/merge.html#ansi-merge ",
		},
	},
	E_MIXED_JOIN: {
		Code:        3200,
		ErrorCode:   "E_MIXED_JOIN",
		Description: "When you have more than 1 JOIN Clause-> 1) Cannot mix NON-ANSI syntax with ANSI syntax, i.e left of ANSI JOIN/NEST is NON-ANSI(LOOKUP/INDEX ) JOIN/NEST, 2) CANNOT mix ANSI syntax with NON-ANSI syntax, i.e left of NON-ANSI(LOOKUP/INDEX) JOIN/NEST is ANSI JOIN/NEST. To put simply if starting with INDEX JOIN/NEST any use of JOINS after this must also be NON-ANSI (LOOKUP/INDEX), if starting with ANSI JOIN any use of JOINS after this MUST also be ANSI. Same goes for NEST",
		Causes: []string{
			"examples of bad query: SELECT e.employee_name, d.department_name, p.project_name\nFROM `employees` AS e\nJOIN `departments` AS d ON e.department_id = META(d).id\nJOIN `projects` AS p ON KEYS META(d).id. -> We are combining ANSI JOIN and LOOKUP JOIN here",
		},
		Actions: []string{
			"JOIN documentation for ypur reference to reform your query https://docs.couchbase.com/server/current/n1ql/n1ql-language-reference/join.html#section_ek1_jnx_1db or use couchbaseiq in capella",
		},
	},
	E_WINDOW_SEMANTIC: {
		Code:        3220,
		ErrorCode:   "E_WINDOW_SEMANTIC",
		Description: "For Specific Aggregates we don't support usage of 1.Aggregate Quantifier DISTINCT , 2. NULL modifier  https://docs.couchbase.com/server/current/n1ql/n1ql-language-reference/windowfun.html#nulls-treatment is not allowed, 3. FROM modifier https://docs.couchbase.com/server/current/n1ql/n1ql-language-reference/windowfun.html#nthval-from  is not allowed, 4. for some window function FILTER CLAUSE is not allowed https://docs.couchbase.com/server/current/n1ql/n1ql-language-reference/aggregatefun.html#filter-clause , 5. window functions require OVER CLAUSE, regular usage is not allowed",
		Causes: []string{
			"1) NOT SURE OF ANY AGGREGATE THAT DISALLOWS DISTINCT as is or over a WINDOW??",
			"2) NULLS treatment is allowed only for Window functions https://docs.couchbase.com/server/current/n1ql/n1ql-language-reference/windowfun.html , and not for aggegrate functions https://docs.couchbase.com/server/current/n1ql/n1ql-language-reference/aggregatefun.html used as window function.",
			"3) FROm modifier is only allowed in case of NTH_VALUE function https://docs.couchbase.com/server/current/n1ql/n1ql-language-reference/windowfun.html#fn-window-nth-value . Reason behind this is \"semantically\" nulls checking is not possible in aggregate functions and order-for-rank([FROM LAST|FROM FIRST]) doesn't make sense for any function except NTH_VALUE()",
		},
		Actions: []string{
			"Please correct the usage by removing the Clause that is breaking the semantic check. Or ask at couchbase forums for further help",
		},
	},
	E_ENTERPRISE_FEATURE: {
		Code:        3230,
		ErrorCode:   "E_ENTERPRISE_FEATURE",
		Description: "The feature accessed by the STATEMENT user has tried to execute is an enterprise level feature.",
		Causes: []string{
			"1) Window Functions supported only in EE",
			"2) ADVISE statement and ADVISOR() function is only supported in EE",
			"3) UPDATE STATISTICS STATEMENT is not supported in EE as cost-based-optimizer is not available",
		},
		Actions: []string{},
	},
	E_ADVISE_UNSUPPORTED_STMT: {
		Code:        3250,
		ErrorCode:   "E_ADVISE_UNSUPPORTED_STMT",
		Description: "Advise supports SELECT, MERGE, UPDATE and DELETE statements only.",
		Causes: []string{
			"Pretty self-sufficient error message but incase you wanted to know-> ADVISE DOESN'T SUPPORT 1. UPSERT, 2. UPDATE STATISTICS , 3. START TRANSACTION, 4. COMMIT, 5. ROLLBACK [SAVE_POINT] , 5. SET TRANSACTION ISOLATION, 6. DROP SCOPE, 7. CREATE SCOPE, 8. SAVEPOINT , 9. REVOKE ROLE, 10. GRANT ROLE, 11. INSERT, 12. INFER, 13. CREATE PRIMARY INDEX, 13. DROP INDEX, 14. CREATE INDEX, 15. BUILD INDEX, 16. ALTER INDEX, 17. EXECUTE FUNCTION, 18. DROP FUNCTION, 19. CREATE FUNCTION, 20. EXPLAIN, 21. EXPLAIN FUNCTION, 22. EXECUTE, 23. DROP COLLECTION, 24. CREATE COLLECTION, 25. ADVISE",
		},
		Actions: []string{
			"ADVISE documentation for your reference https://docs.couchbase.com/server/current/n1ql/n1ql-language-reference/advise.html ",
		},
	},
	E_ADVISOR_PROJ_ONLY: {
		Code:        3255,
		ErrorCode:   "E_ADVISOR_PROJ_ONLY",
		Description: "Advisor function is only allowed in projection clause",
		Causes: []string{
			"Cannot USE ADIVISOR(..) function in FROM/WHERE CLAUSE. As doesn't make sense to use semantically.",
		},
		Actions: []string{
			"Typical usage SELECT AVISOR([statement/array]) https://docs.couchbase.com/server/current/n1ql/n1ql-language-reference/advisor.html  ",
		},
	},
	E_ADVISOR_NO_FROM: {
		Code:        3256,
		ErrorCode:   "E_ADVISOR_NO_FROM",
		Description: "FROM clause is not allowed when Advisor function is present in projection clause.",
		Causes: []string{
			"Cannot have FROM CLAUSE when using ADVISOR() in projection. Again for semantic reasons.",
		},
		Actions: []string{
			"Typical usage SELECT AVISOR([statement/array]) https://docs.couchbase.com/server/current/n1ql/n1ql-language-reference/advisor.html   ",
		},
	},
	E_MHDP_ONLY_FEATURE: {
		Code:        3260,
		ErrorCode:   "E_MHDP_ONLY_FEATURE",
		Description: "UNUSED",
		Causes:      []string{},
		Actions:     []string{},
	},
	E_MISSING_USE_KEYS: {
		Code:        3261,
		ErrorCode:   "E_MISSING_USE_KEYS",
		Description: "term must have USE KEYS",
		Causes: []string{
			"user has passed keyspace using named/positional parameter for UPDATE/DELETE statement. Example: \\set -$d `default`;  DELETE FROM $d as d USE KEYS [\"key::00d9be61-0905-41a2-9c9d-cb67dd310d04\"]; , but without USE KEYS this is a bad query eg: DELETE FROM $d as d;  In a similar fashion same holds true for UPDATE when passing keyspace as named/positional param something like UPDATE [name-param] as [alias] USE KEYS [..] SET/UNSET CLAUSE-WHERE CLAUSE-LIMIT CLAUSE-RETURNING CLAUSE.",
		},
		Actions: []string{
			"If the reason behind hiding the keyspace term using named/positional is not significant replace with the keyspace-term[bucket.scope.collection] to avoid USE KEYS clause.",
		},
	},
	E_HAS_USE_INDEXES: {
		Code:        3262,
		ErrorCode:   "E_HAS_USE_INDEXES",
		Description: "term should not have USE INDEX",
		Causes: []string{
			"user has passed keyspace term using named/positional parameter for UPDATE/DELETE statement. And has USE INDEX subclause, this is the root of this is error. Semantically we disallow USE INDEX clause as positional/named parameters are dynamic.",
		},
		Actions: []string{
			"If the reason behind hiding the keyspace term using named/positional is not significant replace with the keyspace-term[bucket.scope.collection] and have USE INDEX clause.",
		},
	},
	E_UPDATE_STAT_INVALID_INDEX_TYPE: {
		Code:        3270,
		ErrorCode:   "E_UPDATE_STAT_INVALID_INDEX_TYPE",
		Description: "UPDATE STATISTICS (ANALYZE) supports GSI indexes only for INDEX option.",
		Causes: []string{
			"user may have tried to run UPDATE STATISTICS for index other than GSI as the provider. Example bad query: UPDATE STATISTICS FOR INDEX test ON `travel-sample`.`inventory`.`landmark` USING FTS;",
		},
		Actions: []string{
			"Currently CBO is only supported for GSI indexes.",
		},
	},
	E_UPDATE_STAT_INDEX_ALL_COLLECTION_ONLY: {
		Code:        3271,
		ErrorCode:   "E_UPDATE_STAT_INDEX_ALL_COLLECTION_ONLY",
		Description: "INDEX ALL option for UPDATE STATISTICS (ANALYZE) can only be used for a collection.",
		Causes: []string{
			"INDEX ALL clause expects keyspace_ref specified to be a collection. Example bad query: UPDATE STATISTICS FOR `travel-sample` INDEX ALL; cannot update statistics for all indexes on travel-sample bucket, but can collect statistics for all indexes of a collection, example: UPDATE STATISTICS FOR `travel-sample`.inventory.airport INDEX ALL;",
		},
		Actions: []string{
			"Change keyspace_ref term to a path that points to a collection.",
		},
	},
	E_UPDATE_STAT_SELF_NOTALLOWED: {
		Code:        3272,
		ErrorCode:   "E_UPDATE_STAT_SELF_NOTALLOWED",
		Description: "\"UPDATE STATISTICS of 'self' is not allowed\"",
		Causes: []string{
			"user tried a UPDATE STATISTICS for index expression https://docs.couchbase.com/server/current/n1ql/n1ql-language-reference/statistics-expressions.html query, with one of the index expression as \"SELF\", which is semantically ruled out as it is not a valid index expression. Example bad query: UPDATE STATISTICS FOR hotel(city, country, free_breakfast, SELF); Correct usage of SELF is in SELECT SELF / RETURN SELF constructs.",
		},
		Actions: []string{
			"remove SELF keyword from the index expressions list. Reference for ypur help https://docs.couchbase.com/server/current/n1ql/n1ql-language-reference/statistics-expressions.html#index-expr ",
		},
	},
	E_CREATE_INDEX_NOT_INDEXABLE: {
		Code:        3280,
		ErrorCode:   "E_CREATE_INDEX_NOT_INDEXABLE",
		Description: "index key expression is not indexable",
		Causes: []string{
			"CREATE INDEX statement , index keys expression cannot be a constant or the passed expression is flagged as not indexable. For a run through of this scenario assume we have `default` collection with documents like {\"a\":[number]} , some bad index definitions: 1) CREATE INDEX def_idx ON default(SIN(60)); Expression SIN(60)'s value is predetermined. 2) CREATE INDEX def_idx ON default(INFER_VALUE(a)); the function INFER_VALUE() is not indexable. Some good examples 1) CREATE INDEX def_idx ON default(a); 2) CREATE INDEX def_idx(SIN(a));",
		},
		Actions: []string{
			"change the index key definition to an expression that fits your original purpose and is indexable. SQL++ expressions for your reference https://docs.couchbase.com/server/current/n1ql/n1ql-language-reference/index.html#N1QL_Expressions , or array expression for array indexing https://docs.couchbase.com/server/current/n1ql/n1ql-language-reference/indexing-arrays.html#array-expr ",
		},
	},
	E_CREATE_INDEX_ATTRIBUTE_MISSING: {
		Code:        3281,
		ErrorCode:   "E_CREATE_INDEX_ATTRIBUTE_MISSING",
		Description: "MISSING attribute not allowed (Only allowed with gsi leading key).",
		Causes: []string{
			"1) if using FLATTEN_KEYS construct only 1st argument can have INCLUDE MISSING",
			"2) if not a gsi INDEX , definition allows INCLUDE MISSING only for leading key.",
		},
		Actions: []string{},
	},
	E_CREATE_INDEX_ATTRIBUTE: {
		Code:        3282,
		ErrorCode:   "E_CREATE_INDEX_ATTRIBUTE",
		Description: "index key attributes are not allowed for array indexing when using FLATTEN_KEYS()",
		Causes: []string{
			"[ INCLUDE MISSING ]/ [ ASC/DESC ] index key attributes are disallowed when using ARRAY index keys using FLATTEN_KEYS(..), example bad query CREATE INDEX ixf_sched_missing\nON route\n(DISTINCT ARRAY FLATTEN_KEYS(v.utc INCLUDE MISSING, v.day) FOR v IN schedule END INCLUDE MISSING);",
		},
		Actions: []string{
			"NOTE: expressions passed to FLATTEN_KEYS() can use index-key attributes but not the actual array-index expression.",
		},
	},
	E_FLATTEN_KEYS: {
		Code:        3283,
		ErrorCode:   "E_FLATTEN_KEYS",
		Description: "flatten_keys(...) is not allowed in this context",
		Causes: []string{
			"FLATTEN_KEYS function is only allowed in 1) CREATE INDEX 2) UPDATE STATISTICS 3) Not surrounded by a function , example bad query: CREATE INDEX ixf_sched\n  ON route\n  (ALL ARRAY GREATEST(FLATTEN_KEYS(s.day DESC, s.flight), 100) FOR s IN schedule END,\n  sourceairport, destinationairport, stops);  Here GREATEST function wraps FLATTEN_KEYS which is semantically disallowed. 4) No recursive calls , example bad query: CREATE INDEX ixf_sched\n  ON route\n  (ALL ARRAY GREATEST(FLATTEN_KEYS(s.day DESC, FLATTEN_KEYS(s.flight))) FOR s IN schedule END,\n  sourceairport, destinationairport, stops); Recursive usage is semanticantically incorrect.",
		},
		Actions: []string{
			"rethink if your query needs FLATTEN_KEYS function, also to summarize semantic checks -> 1. not surrounded by any other function,  2. no recursive calls ",
		},
	},
	E_ALL_DISTINCT_NOT_ALLOWED: {
		Code:        3284,
		ErrorCode:   "E_ALL_DISTINCT_NOT_ALLOWED",
		Description: "ALL/DISTINCT is only allowed in CREATE INDEX & UPDATE STATISTICS statements. -> NOT CALLED as parser rules never reduce to expression.all for any other case except CREATE INDEX & UPDATE STATISTICS using all_expr rule",
		Causes:      []string{},
		Actions:     []string{},
	},
	E_CREATE_INDEX_SELF_NOTALLOWED: {
		Code:        3285,
		ErrorCode:   "E_CREATE_INDEX_SELF_NOTALLOWED",
		Description: "Index of SELF is not allowed as a index key",
		Causes: []string{
			"SELF keyword cannot be used as a index key. Example bad query CREATE INDEX temp_def_idx ON default(SELF, a);",
		},
		Actions: []string{
			"Remove SELF as index key, if you want an index on each document in the collection you could use the meta function https://docs.couchbase.com/server/current/n1ql/n1ql-language-reference/metafun.html#meta  , the meta().id key is covered by any index on the keyspace which covers the document key. This approach can be used if this is the angle you where thinking of.",
		},
	},
	E_INDEX_NOT_ALLOWED: {
		Code:        3286,
		ErrorCode:   "E_INDEX_NOT_ALLOWED",
		Description: "PRIMARY INDEX is not allowed using FTS",
		Causes: []string{
			"CREATE PRIMARY INDEX ON [keyspace] USING FTS, is semantically not allowed.",
		},
		Actions: []string{
			"Change to gsi as your provider.(USING GSI)",
		},
	},
	E_JOIN_HINT_FIRST_FROM_TERM: {
		Code:        3290,
		ErrorCode:   "E_JOIN_HINT_FIRST_FROM_TERM",
		Description: "Join hint (USE HASH or USE NL) cannot be specified on the first from term",
		Causes: []string{
			"Legacy USE HASH: https://docs.couchbase.com/server/current/n1ql/n1ql-language-reference/join.html#use-hash-hint  / USE NL hints: https://docs.couchbase.com/server/current/n1ql/n1ql-language-reference/join.html#use-nl-hint  are not allowed for the first from term in the join tree. Example bad query: SELECT a.airportname, r.airline\nFROM airport a USE HASH(probe)\nJOIN route r \nON a.faa = r.sourceairport\nWHERE a.city = \"San Francisco\";",
		},
		Actions: []string{
			"Assuming you have cbo(cost-based-optimizer) on, we allow join enumeration. Now new relation style join hints can be used to provide join hints on any term in the join tree including first keyspace term. Example usage of this construct: SELECT /*+ USE_HASH(a) */\n       a.airportname, r.airline\nFROM airport a\nJOIN route r\nON a.faa = r.sourceairport\nWHERE a.city = \"San Francisco\";  documentation link for your reference: https://docs.couchbase.com/server/current/n1ql/n1ql-language-reference/optimizer-hints.html ",
		},
	},
	E_RECURSIVE_WITH_SEMANTIC: {
		Code:        3300,
		ErrorCode:   "E_RECURSIVE_WITH_SEMANTIC",
		Description: "recursive_with semantics: [cause]",
		Causes: []string{
			"1) We semantically disallow certain constructs when using Recursive CTEs. To be in accordance to the sql standard and also follow linearly recursive definition approach.\nLinear recursive, to put simply just keep joining newly generated Ancestor documents with Parent.",
			"2) disallow order by/ limit / offset Clause in subquery definition for the CTE.\nOrder By/ LIMIT / OFFSET all are applied on the entire result set thus making it unclear how to apply the during each iteration. Order/LIMIT/OFFSET are part to the entire select(both anchor & recursive) which if seen during formalization we error out.\n WITH RECURSIVE empHierar AS (SELECT e.*, 0 as lvl FROM employees e WHERE e.manager_id IS MISSING UNION SELECT e1.*, empHierar.lvl+1 as lvl FROM employees e1 JOIN empHierar ON empHierar.employee_id = e1.manager_id ORDER BY employee_name) SELECT * FROM empHierar;\n{\n    \"requestID\": \"46d6646d-e12f-4ab6-8950-1905626ecace\",\n    \"errors\": [\n        {\n            \"code\": 3300,\n            \"msg\": \"recursive_with semantics: Order/Limit/Offset not allowed\"\n        }\n    ],\n\nWITH RECURSIVE empHierar AS (SELECT e.*, 0 as lvl FROM employees e WHERE e.manager_id IS MISSING UNION SELECT e1.*, empHierar.lvl+1 as lvl FROM employees e1 JOIN empHierar ON empHierar.employee_id = e1.manager_id LIMIT 3) SELECT * FROM empHierar;\n{\n    \"requestID\": \"cd899dce-66d3-4f81-a09d-de9806dce1fc\",\n    \"errors\": [\n        {\n            \"code\": 3300,\n            \"msg\": \"recursive_with semantics: Order/Limit/Offset not allowed\"\n        }\n    ],\n \nWITH RECURSIVE empHierar AS (SELECT e.*, 0 as lvl FROM employees e WHERE e.manager_id IS MISSING UNION SELECT e1.*, empHierar.lvl+1 as lvl FROM employees e1 JOIN empHierar ON empHierar.employee_id = e1.manager_id OFFSET 3) SELECT * FROM empHierar;\n{\n    \"requestID\": \"ad2b5cec-5831-4846-af19-96390fb01ced\",\n    \"errors\": [\n        {\n            \"code\": 3300,\n            \"msg\": \"recursive_with semantics: Order/Limit/Offset not allowed\"\n        }\n    ],",
			"3) Aggregates/Window functions are not allowed, -> both anchor as well as recursive clause\nThe reason for this restriction is related to the way recursive CTEs are processed. The recursive term is essentially executed repeatedly until no more rows are returned. If you introduce an aggregate within the recursive term, it becomes unclear how the aggregation should be applied during each iteration. Aggregations are typically applied after all rows have been retrieved, making their usage within a recursive context less straightforward.\n WITH RECURSIVE cte AS (SELECT COUNT(*) AS CountAll FROM landmark) SELECT * FROM cte;\n{\n    \"requestID\": \"4385c979-6062-4f85-8298-176a12093fbd\",\n    \"errors\": [\n        {\n            \"code\": 3300,\n            \"msg\": \"recursive_with semantics: Aggregates/Window functions are not allowed\"\n        }\n    ],",
			"4) Groupby is not allowed -> both in anchor as well as recursive clause\nWhen you introduce GROUP BY in the recursive term, it creates ambiguity in terms of how to group the results at each iteration. The grouping operation is typically applied to the entire result set, and it's not clear how the grouping should be handled during each step of the recursion.\nWITH RECURSIVE empHierar AS (SELECT e.*, 0 as lvl FROM employees e WHERE e.manager_id IS MISSING UNION SELECT e1.*, empHierar.lvl+1 as lvl FROM employees e1 JOIN empHierar ON empHierar.employee_id = e1.manager_id GROUP BY e1.manager_id) SELECT * FROM empHierar;\n{\n    \"requestID\": \"7909e4fa-6c0f-402c-b60f-bf24c38ef1bb\",\n    \"errors\": [\n        {\n            \"code\": 3300,\n            \"msg\": \"recursive_with semantics: Grouping is not allowed\"\n        }\n    ],",
			"5) DISTINCT is not allowed in PROJECTION. As it doesn't make sense on how to account for DISTINCT Projection terms across iteration, \n WITH RECURSIVE empHierar AS (SELECT DISTINCT e.employee_id, 0 as lvl FROM employees e WHERE e.manager_id IS MISSING UNION SELECT e1.employee_id, empHierar.lvl+1 as lvl FROM employees e1 JOIN empHierar ON empHierar.employee_id = e1.manager_id) SELECT * FROM empHierar;\n{\n    \"requestID\": \"7df8a033-876c-4567-8fd9-0fc4f971571c\",\n    \"errors\": [\n        {\n            \"code\": 3300,\n            \"msg\": \"recursive_with semantics: Distinct not allowed\"\n        }\n    ],",
		},
		Actions: []string{
			"Don't make use of\n1) Order/Limit/Offset on the CTE definition\n2) group by neither permitted in anchor nor recursive\n3) distinct clause on projection terms is not allowed in both anchor and recursive clause.\n4) Aggregates/window functions are not allowed as well again for both anchor and recursive clause.",
		},
		IsUser: true,
	},
	E_ANCHOR_RECURSIVE_REF: {
		Code:        3301,
		ErrorCode:   "E_ANCHOR_RECURSIVE_REF",
		Description: "Anchor Clause cannot have recursive reference in FROM Expression : [recursive_alias]",
		Causes: []string{
			"During formalization of WITH Clause , if recursive and also subquery-SELECT has both UNION/UNION-ALL.\nWe split the SELECT on UNION/UNION-ALL -> to produce anchor and recursive clause.\n\nSemantically disallow recursive reference in Anchor's FROM Clause.\n WITH RECURSIVE empHierar AS (SELECT * FROM empHierar UNION SELECT 1) SELECT * FROM empHierar;\n{\n    \"requestID\": \"322b79a1-85b6-4522-a420-09129bdb9c4b\",\n    \"errors\": [\n        {\n            \"code\": 3301,\n            \"msg\": \"Anchor Clause cannot have recursive reference in FROM expression: empHierar\"\n        }\n    ],",
		},
		Actions: []string{
			"Can't have recursive refernce in anchor as it isn't a defined expression yet, surely this not what you were trying todo.",
		},
		IsUser: true,
	},
	E_MORE_THAN_ONE_RECURSIVE_REF: {
		Code:        3302,
		ErrorCode:   "E_MORE_THAN_ONE_RECURSIVE_REF",
		Description: "recursive ref:[recursive cte-alias] must not appear more than once in the FROM clause",
		Causes: []string{
			"As we follow linear recursive sql standard for Recursive withs, \n1) We semantically disallow SELF JOIN for the recursive alias, as uing self-join with the recursive alias directly within the recursive clause could lead to ambiguity in terms of which instance of the CTE is being referred to during each iteration\n\n> WITH RECURSIVE empHierar AS (SELECT e.employee_id, 0 as lvl FROM employees e WHERE e.manager_id IS MISSING UNION SELECT e1.employee_id, empHierar.lvl+1 as lvl FROM employees e1 JOIN empHierar h1 ON eh1.employee_id = e1.manager_id JOIN empHierar h2 ON h2.employee_name=h1.employee_name ) SELECT * FROM empHierar;\n{\n    \"requestID\": \"0626fda3-c8a7-4682-94b0-1f6502eab134\",\n    \"errors\": [\n        {\n            \"code\": 3302,\n            \"msg\": \"recursive ref:empHierar must not appear more than once in the FROM clause\"\n        }\n    ]",
		},
		Actions: []string{
			"Multiple usage of recursive reference in the FROM clause is not supported, please rethink the query or ask on forums",
		},
		IsUser: true,
	},
	E_CONFIG_INVALID_OPTION: {
		Code:        3303,
		ErrorCode:   "E_CONFIG_INVALID_OPTION",
		Description: "Invalid config option ",
		Causes: []string{
			"Only \"levels\", \"documents\" are valid config options for the options clause\n\n WITH RECURSIVE empHierar AS (SELECT e.employee_id, 0 as lvl FROM employees e WHERE e.manager_id IS MISSING UNION SELECT e1.employee_id, h1.lvl+1 as lvl FROM employees e1 JOIN empHierar h1 ON h1.employee_id = e1.manager_id ) OPTIONS {\"not-valid\":1} SELECT * FROM empHierar;\n{\n    \"requestID\": \"750d744b-70c6-4395-851a-2ca9c4644807\",\n    \"signature\": {\n        \"*\": \"*\"\n    },\n    \"results\": [\n    ],\n    \"errors\": [\n        {\n            \"code\": 3303,\n            \"msg\": \"Invalid config option not-valid\",\n            \"reason\": {\n                \"invalid_option\": \"not-valid\"\n            }\n        }\n    ],",
		},
		Actions: []string{
			"Allowed config options are:\n\"levels\"-   Exit after level N.\n\"documents\" -  Exit after accumulating N documents.",
		},
		IsUser: true,
	},
	E_RECURSION_UNSUPPORTED: {
		Code:        3304,
		ErrorCode:   "E_RECURSION_UNSUPPORTED",
		Description: "recursive_with_unsupported: [reason]",
		Causes: []string{
			"1) OUTER JOIN not supported, as may lead \n WITH RECURSIVE empHierar AS (SELECT e.employee_id, 0 as lvl FROM employees e WHERE e.manager_id IS MISSING UNION SELECT e1.employee_id, h1.lvl+1 as lvl FROM employees e1 LEFT OUTER JOIN empHierar h1 ON h1.employee_id = e1.manager_id ) OPTIONS {\"not-valid\":1} SELECT * FROM empHierar;\n{\n    \"requestID\": \"bd4f39a2-42a2-4005-abae-f67aae3be504\",\n    \"errors\": [\n        {\n            \"code\": 3304,\n            \"msg\": \"recursive_with_unsupported: OUTER JOIN\",\n            \"reason\": \"may lead to potential infinite recursion\"\n        }\n    ],\nreason for this lies in how the outer join condition affects the way rows are matched during each iteration of the recursion",
			"2) recursive NEST is unsupported for now\n WITH RECURSIVE empHierar AS (SELECT e.employee_id, 0 as lvl FROM employees e WHERE e.manager_id IS MISSING UNION SELECT e1.employee_id, h1.lvl+1 as lvl FROM employees e1 NEST travellers t ON t.name = e1.manager_id ) OPTIONS {\"not-valid\":1} SELECT * FROM empHierar;\n{\n    \"requestID\": \"73dc1eaf-62bd-4419-b53a-aa5ee510b25c\",\n    \"errors\": [\n        {\n            \"code\": 3304,\n            \"msg\": \"recursive_with_unsupported: NEST\",\n            \"reason\": \"`default`:`employees` as `e1` nest `default`:`travellers` as `t` on ((`t`.`name`) = (`e1`.`manager_id`))\"\n        }\n    ],",
			"3) recursive UNNEST is not supported for now\n WITH RECURSIVE empHierar AS (SELECT e.employee_id, 0 as lvl FROM employees e WHERE e.manager_id IS MISSING UNION SELECT e1.employee_id, h1.lvl+1 as lvl, dept FROM employees e1 UNNEST e1.dept as dept ) OPTIONS {\"not-valid\":1} SELECT * FROM empHierar;\n{\n    \"requestID\": \"0494a309-b0d4-41d2-9a32-56b32862a8ed\",\n    \"errors\": [\n        {\n            \"code\": 3304,\n            \"msg\": \"recursive_with_unsupported: UNNEST\",\n            \"reason\": \"`default`:`employees` as `e1` unnest (`e1`.`dept`) as `dept`\"\n        }\n    ],",
		},
		Actions: []string{
			"1) only use INNER JOIN (which is the default for Ansi, Lookup & Index join)",
			"2) recursive NESTing is unsupported for now, reach out to support if you think it is a good feature to have.",
			"3) recursive UNNESTing is unsupported for now, reach out to support if you think it is a good feature to have.",
		},
		IsUser: true,
	},
	E_PLAN: {
		Code:        4000,
		ErrorCode:   "E_PLAN",
		Description: "",
		Causes:      []string{},
		Actions:     []string{},
	},
	E_REPREPARE: {
		Code:        4001,
		ErrorCode:   "E_REPREPARE",
		Description: "reprepare_error",
		Causes: []string{
			"When trying to get prepared from the preparedcache if the entry is not found on the local machine, the prepared statement cache is primed from another query node. We do not use the plan as is straight away as we cannot trust anything received over the network. We verify the plan, we check for the uid for each keyspace involved in the plan operator.  Also, check if metadata for the indexers & keyspaces involved is same as the one one in the encoded plan. If the before mentioned verify & metadata check return false. We reprepare the text-statement. And while repreparing either 1) parsing the statement failed,  2) building the plan failed  3) encoding the plan to store in the preparedcache entry failed",
		},
		Actions: []string{
			"The prepared entry is likely to be corrupt. One way to go about this would be to prepare the statement under a new name.",
		},
		IsUser: false,
	},
	E_NO_TERM_NAME: {
		Code:        4010,
		ErrorCode:   "E_NO_TERM_NAME",
		Description: "From Term must have a name or alias.",
		Causes: []string{
			"root of the issue here is when passing terms in FROM clause, the formalizer runs through the terms for the easy of projection operator. When constructing from_term using a keyspace/collection, without an alias the path is taken for eg: `travel-sample`inventory.airport -> path=airport. Thus projection airport.id now knows how to pull out the value for field id from the scope. But when using a from expression term, eg: [{\n  \"id\": 1254}] without an explicit alias, The formalizer would fail to annotate the projection terms thus we error out during formalization of the expression term if no alias is passed. This applies for joins/nest as well. Example bad query: SELECT id\nFROM [{\n  \"id\": 1254,\n  \"type\": \"airport\",\n}];",
		},
		Actions: []string{
			"add an alias for each expression term used in FROM Clause. Don't forget to annotate your projection terms with the same alias. The correct query for the bad example in causes: SELECT a.id\nFROM [{\n  \"id\": 1254,\n  \"type\": \"airport\"\n}] a;",
		},
	},
	E_DUPLICATE_ALIAS: {
		Code:        4020,
		ErrorCode:   "E_DUPLICATE_ALIAS",
		Description: "Duplicate alias ",
		Causes: []string{
			"user is expected alias from terms in a distinguishable manner,i.e all of them are unique. Possible causes 1) same alias in JOIN-> SELECT r.airportname, r.airline\nFROM airport r\nJOIN route r\nON r.faa = r.sourceairport\nWHERE r.city = \"San Francisco\"; 2) Similarly for NEST https://docs.couchbase.com/server/current/n1ql/n1ql-language-reference/nest.html  3) UNNEST https://docs.couchbase.com/server/current/n1ql/n1ql-language-reference/unnest.html 4) FROM term has same alias as WITH term, example bad query: WITH a AS (SELECT 1 as r) SELECT a.r as r1, a.r as r2 FROM a, [{\"r\":2}] a; NOTE: from term can keyspace/expression/subquery, 4) MERGE source and target have same alias, example bad query: MERGE INTO hotel t\nUSING [\n  {\"id\":\"21728\", \"vacancy\": true},\n  {\"id\":\"21730\", \"vacancy\": true}\n] t\nON meta(t).id = \"hotel_\" || t.id\nWHEN MATCHED THEN ...",
		},
		Actions: []string{
			"Rename the alias used so it is unique.",
		},
	},
	E_DUPLICATE_WITH_ALIAS: {
		Code:        4021,
		ErrorCode:   "E_DUPLICATE_WITH_ALIAS",
		Description: "Duplicate WITH alias reference",
		Causes: []string{
			"WITH clause terms or cte(common table expression) have the same alias. This is is disallowed during formalization as it creates ambiguity , example bad query: WITH a AS (SELECT 1 as r), a AS (SELECT 2 as r) SELECT a.r as r1 FROM a;",
		},
		Actions: []string{
			"Rename the alis used for the cte that is a duplicate",
		},
	},
	E_UNKNOWN_FOR: {
		Code:        4025,
		ErrorCode:   "E_UNKNOWN_FOR",
		Description: "Unknow alias in : ON KEY [expr] FOR [alias]. ",
		Causes: []string{
			"This error is raised when using IndexJoins where the ON KEY ... FOR ... Clause has an alias that was not previously seen by the formalizer. Typically the FOR includes the left side of join and user has made a mistake in refering to the same alias, example bad query: SELECT * FROM airline\n  JOIN route\n  ON KEY route.airlineid FOR air\nWHERE airline.icao = \"SWA\";  Here inplace of \"air\" the formalizer expects \"airline\"",
		},
		Actions: []string{
			"Use the correct alias for ON KEY ... FOR ... Clause. Find the documentation here https://docs.couchbase.com/server/current/n1ql/n1ql-language-reference/join.html#index-join-clause for your reference.",
		},
	},
	E_SUBQUERY_MISSING_KEYS: {
		Code:        4030,
		ErrorCode:   "E_SUBQUERY_MISSING_KEYS",
		Description: "UNUSED",
		Causes:      []string{},
		Actions:     []string{},
	},
	E_SUBQUERY_MISSING_INDEX: {
		Code:        4035,
		ErrorCode:   "E_SUBQUERY_MISSING_INDEX",
		Description: "No secondary index available for keyspace, in correlated subquery",
		Causes: []string{
			"The keyspace in from clause of a correlated subquery: has more than 1000documents so at planning time we raise error as in production without a secondary index the query would be slow due to the underlying cartesian product. Example bad query:  SELECT airportname FROM `travel-sample`.inventory.airport AS outerAirport WHERE ( SELECT COUNT(*) FROM `travel-sample`.inventory.route AS innerRoute WHERE innerRoute.sourceairportid = outerAirport.id ) > 50;\\n error:  {\n            \"code\": 5370,\n            \"msg\": \"Unable to run subquery - cause: Correlated subquery's keyspace (innerRoute) cannot have more than 1000 documents without appropriate secondary index\"\n        },\n        {\n            \"code\": 5010,\n            \"msg\": \"Error evaluating filter\",\n            \"reason\": {\n                \"_level\": \"exception\",\n                \"caller\": \"context:1132\",\n                \"code\": 5370,\n                \"icause\": \"Correlated subquery's keyspace (innerRoute) cannot have more than 1000 documents without appropriate secondary index\",\n                \"key\": \"execution.subquery.build\",\n                \"message\": \"Unable to run subquery\"\n            }\n        }",
		},
		Actions: []string{
			"Create relevant index for the predicate(key) that makes the subquery correlated. For our bad example: CREATE INDEX adv_sourceairportid ON `travel-sample`.`inventory`.`route`(`sourceairportid` INCLUDE MISSING)",
		},
		IsUser: true,
	},
	E_SUBQUERY_PRIMARY_DOCS_EXCEEDED: {
		Code:        4036,
		ErrorCode:   "E_SUBQUERY_PRIMARY_DOCS_EXCEEDED",
		Description: "Correlated subquery's keyspace [keyspace] cannot have more than 100 documents without appropriate secondary index",
		Causes: []string{
			"Similar to joins correlated subquery needed secondary index for performance reasons,\n\nSELECT a FROM default d1 WHERE EXISTS (SELECT a FROM default d2 WHERE d1.a=d2.a);\n[\n  {\n    \"code\": 5370,\n    \"msg\": \"Unable to run subquery - cause: Correlated subquery's keyspace (d2) cannot have more than 1000 documents without appropriate secondary index\"\n  },\n  {\n    \"code\": 5010,\n    \"msg\": \"Error evaluating filter\",\n    \"reason\": {\n      \"_level\": \"exception\",\n      \"caller\": \"context:1154\",\n      \"code\": 5370,\n      \"icause\": \"Correlated subquery's keyspace (d2) cannot have more than 1000 documents without appropriate secondary index\",\n      \"key\": \"execution.subquery.build\",\n      \"message\": \"Unable to run subquery\"\n    }\n  }\n]",
		},
		Actions: []string{
			"CREATE INDEX for the keyspace in the correlated subquery",
		},
		IsUser: true,
	},
	E_NO_SUCH_PREPARED: {
		Code:        4040,
		ErrorCode:   "E_NO_SUCH_PREPARED",
		Description: "No such prepared statement:",
		Causes: []string{
			"1) User has requested sent a DELETE request to /query/service/admin/prepareds/\\{pname\\} to delete the pname entry from the prepared cache, but got a bad response as no such prepared entry with name \\{pname\\} is found. 2) query run : EXECUTE {pname} ; but pname doesn't exist in the prepared cache. 3) user may have prepared a statement with a different query context, and execute with another query context -> this fails the getPrepared logic.",
		},
		Actions: []string{
			"run SELECT * FROM system:prepareds; this will tell you all the prepareds you have(across nodes) , notice name and queryContext fields.",
		},
		IsUser: true,
	},
	E_UNRECOGNIZED_PREPARED: {
		Code:        4050,
		ErrorCode:   "E_UNRECOGNIZED_PREPARED",
		Description: "JSON unmarshalling error",
		Causes: []string{
			"On startup cbq-engine/query service tries to prime prepared entries from other nodes that have query service running, after received the entry(which contains encoded plan), we decode the encoded plan. But logic to decode fails on the data(encoded plan ) received.",
		},
		Actions: []string{
			"likely corrupt encoded plan/ prepared statement, start fresh prepare the statement under a new name",
		},
		IsUser: false,
	},
	E_PREPARED_NAME: {
		Code:        4060,
		ErrorCode:   "E_PREPARED_NAME",
		Description: "Unable to add name:[preparedname], duplicate name: [preparedname]",
		Causes: []string{
			"User has ran PREPARE [pname] as [stmt], but entry for [pname] already exists with a different text i.e statement, so cannot add.",
		},
		Actions: []string{
			"use SELECT  statement FROM system:prepareds WHERE name=[pname], only if that statement matches you are allowed to manually  try for a reprepare. Else the simple solution here would be to just use a different name",
		},
		IsUser: true,
	},
	E_PREPARED_DECODING: {
		Code:        4070,
		ErrorCode:   "E_PREPARED_DECODING",
		Description: "Unable to decode prepared statement",
		Causes: []string{
			"decode has 3 steps: 1) going from encoded prepared statement first decode 2) decompress 3) unmarshal prepared bytes to prepared algebra struct. If any of the steps mentioned earlier fail while priming the cache we raise this error",
		},
		Actions: []string{
			"likely corrupt encoded plan/ prepared statement, start fresh prepare the statement under a new name",
		},
		IsUser: false,
	},
	E_PREPARED_ENCODING_MISMATCH: {
		Code:        4080,
		ErrorCode:   "E_PREPARED_ENCODING_MISMATCH",
		Description: "",
		Causes:      []string{},
		Actions:     []string{},
	},
	E_ENCODING_NAME_MISMATCH: {
		Code:        4090,
		ErrorCode:   "E_ENCODING_NAME_MISMATCH",
		Description: "Mismatching name in encoded plan",
		Causes: []string{
			"When trying to prime prepared entires from other nodes, encoding plan and name mapping doesn't match on the current node when comparing with the remote entry that is when this error is raised",
		},
		Actions: []string{
			"DELETE the entry and prepare again.",
		},
		IsUser: false,
	},
	E_ENCODING_CONTEXT_MISMATCH: {
		Code:        4091,
		ErrorCode:   "E_ENCODING_CONTEXT_MISMATCH",
		Description: "Mismatching query_context in encoded plan",
		Causes:      []string{},
		Actions:     []string{},
	},
	E_PREDEFINED_PREPARED_NAME: {
		Code:        4092,
		ErrorCode:   "E_PREDEFINED_PREPARED_NAME",
		Description: "Prepared name [predefinedname] is predefined (reserved).",
		Causes: []string{
			" __get, __insert, __upsert, __update, __delete names are not allowed as this are used for cache warmup and not allowed to be used by the user.",
		},
		Actions: []string{
			"pick a different name other than  __get, __insert, __upsert, __update, __delete",
		},
		IsUser: true,
	},
	E_NO_INDEX_JOIN: {
		Code:        4100,
		ErrorCode:   "E_NO_INDEX_JOIN",
		Description: "No index available for right hand side join term, in index join construct",
		Causes: []string{
			"INDEX JOIN , specifies usage of ON KEY [right-hand-side term's key] FOR [left-hand-side term] . This matches the index key(foriegn key) of right hand side term with left-hand-side terms document key(primary key). With this mentioned, this error is raised when there is not index available for right-hand-side term.  Example bad query:  cbq> SELECT * FROM `travel-sample`.inventory.airline JOIN `travel-sample`.inventory.route ON KEY route.airlineid FOR airline WHERE airline.icao = \"SWA\";\n{\n    \"requestID\": \"a50f012b-691c-4a66-8be3-02d0fa84f0eb\",\n    \"errors\": [\n        {\n            \"code\": 4100,\n            \"msg\": \"No index available for join term route\"\n        }\n    ],\n    \"status\": \"fatal\",\n    \"metrics\": {\n        \"elapsedTime\": \"794.458\u00b5s\",\n        \"executionTime\": \"726.583\u00b5s\",\n        \"resultCount\": 0,\n        \"resultSize\": 0,\n        \"serviceLoad\": 2,\n        \"errorCount\": 1\n    }\n}\nPoint to note: Sequential Scan feature is not enabled.",
		},
		Actions: []string{
			"CREATE suitable index for right-hand-side term when using IndexJoin construct. Index Avisor link https://docs.couchbase.com/server/current/guides/index-advisor.html#advice-single for your reference. To correct our bad example we can create index like so, CREATE INDEX route_airlineid ON route(airlineid);",
		},
		IsUser: true,
	},
	E_USE_KEYS_USE_INDEXES: {
		Code:        4110,
		ErrorCode:   "E_USE_KEYS_USE_INDEXES",
		Description: "From Expression Term cannot have USE KEYS or USE INDEX Clause",
		Causes: []string{
			"As the error message suggests it doesn't make sense for USE KEYS/USE INDEX so we error out during formalization. Example bad queries: SELECT a.id FROM {\"id\":1} a USE KEYS [\"a_1\"];  SELECT a.id FROM {\"id\":1} a USE INDEX(def_idx USING GSI);",
		},
		Actions: []string{
			"Usage of from expression may have been for testing purpose on your part, replace with actual keyspaceterm(collection). Or use cbimport to transfer the expression's documents to a bucket(USE KEYS) and create suitable index(for USE INDEX)",
		},
	},
	E_NO_PRIMARY_INDEX: {
		Code:        4120,
		ErrorCode:   "E_NO_PRIMARY_INDEX",
		Description: "No index available on keyspace, use CREATE PRIMARY INDEX on [keyspace]",
		Causes: []string{
			"When running SELECT/UPDATE/DELETE queries, during planning we first build the scan operator, except when using USE KEYS CLAUSE for UPDATE/DELETE statements. For which we look at the keyspace term and predicates(if any) to pick the most suitable index. At worst case we resort to primary index if no secondary index is present. But in your case you neither have a secondary index nor a primary index on your keyspace to build the scan operator for the query. Example bad query (no gsi index on airline collection) : SELECT * FROM `travel-sample`.inventory.airline;\n{\n    \"requestID\": \"99bec664-b1c6-4818-a5e7-f6b61c2e491e\",\n    \"errors\": [\n        {\n            \"code\": 4000,\n            \"msg\": \"No index available on keyspace `default`:`travel-sample`.`inventory`.`airline` that matches your query. Use CREATE PRIMARY INDEX ON `default`:`travel-sample`.`inventory`.`airline` to create a primary index, or check that your expected index is online.\"\n        }\n    ],\n    \"status\": \"fatal\",\n    \"metrics\": {\n        \"elapsedTime\": \"829\u00b5s\",\n        \"executionTime\": \"690.5\u00b5s\",\n        \"resultCount\": 0,\n        \"resultSize\": 0,\n        \"serviceLoad\": 2,\n        \"errorCount\": 1\n    }\n} , Note: in production using PRIMARY INDEX is not advisable but while developing/figuring out your query for your problem it is ok to use. Also this error wouldn't be raised if sequential scan feature is on.",
		},
		Actions: []string{
			"to mitigate the error, simply create the primary index: CREATE PRIMARY INDEX ON [keyspace]; ",
		},
	},
	E_PRIMARY_INDEX_OFFLINE: {
		Code:        4125,
		ErrorCode:   "E_PRIMARY_INDEX_OFFLINE",
		Description: "Primary index [indexname] not online.",
		Causes: []string{
			"1) When running SELECT/UPDATE/DELETE queries, during planning we first build the scan operator, except when using USE KEYS CLAUSE for UPDATE/DELETE statements. For which we look at the keyspace term and predicates(if any) to pick the most suitable index. At worst case we resort to primary index if no secondary index is present. But in your case you have a primary index which hasn't been built yet. In other words the index is offline. \n\nExample : \n1) CREATE PRIMARY INDEX idx_landmark_primary\n  ON landmark\n  USING GSI\n  WITH {\"defer_build\":true};",
			"2) SELECT * FROM landmark LIMIT 1;\n{\n    \"requestID\": \"dfd6ba86-1c16-48b9-9a87-529d160e2d35\",\n    \"errors\": [\n        {\n            \"code\" :4000,        \n           \"msg\":\"Primary index idx_landmark_primary not online.\"        \n           \"query\":\"SELECT * FROM landmark LIMIT 1;\"\n        }\n    ],",
		},
		Actions: []string{
			"Solution: Build the index-> BUILD INDEX ON landmark(idx_landmark_primary) USING GSI;",
		},
		IsUser: true,
	},
	E_LIST_SUBQUERIES: {
		Code:        4130,
		ErrorCode:   "E_LIST_SUBQUERIES",
		Description: "Error listing subqueries. NEVER RAISED as the expression subquery lister logic never returns error while traversing any of the expression types",
		Causes: []string{
			"code paths involved:-> subquery Privileges & building subquery",
		},
		Actions: []string{},
	},
	E_NOT_GROUP_KEY_OR_AGG: {
		Code:        4210,
		ErrorCode:   "E_NOT_GROUP_KEY_OR_AGG",
		Description: "Expression in Projection Clause must depend only on group keys or aggregates.",
		Causes: []string{
			"The projection term is not an equivalent to any of the group keys , i.e MAX(country) is an equivalent of `country` but `city` is not an equivalent of `country`. Example bad query: \nSELECT country, name FROM landmark GROUP BY country;\n{\n    \"requestID\": \"8a043403-a0ce-4e66-8515-c529f0717d62\",\n    \"errors\": [\n        {\n            \"code\": 4210,\n            \"column\": 17,\n            \"line\": 1,\n            \"msg\": \"Expression (`landmark`.`name`) (near line 1, column 17) must depend only on group keys or aggregates.\"\n        }\n    ],",
		},
		Actions: []string{
			"Reframe your query to use only projection that are \"equivalentTo\" a group key, Or modify query to use GROUP AS clause to make all other fields available, but doing so will lose the index pushdown benefit you might have got earlier.",
		},
		IsUser: true,
	},
	E_INDEX_ALREADY_EXISTS: {
		Code:        4300,
		ErrorCode:   "E_INDEX_ALREADY_EXISTS",
		Description: "The index already exists. ",
		Causes: []string{
			"Index with same name already exists. That is when we raise this error. Example bad query: (same index already created) CREATE PRIMARY INDEX idx_landmark_primary ON landmark;",
		},
		Actions: []string{
			"A way to avoid the error, is to add the IF NOT EXISTS Clause, this doesn't make the request to create index fatal. example: CREATE PRIMARY INDEX IF NOT EXISTS idx_landmark_primary ON landmark;",
		},
		IsUser: true,
	},
	E_AMBIGUOUS_META: {
		Code:        4310,
		ErrorCode:   "E_AMBIGUOUS_META",
		Description: "meta() / search_meta() / search_score() in query with multiple FROM terms requires an argument",
		Causes: []string{
			"functions meta() / search_meta() / search_score require argument of keyspace alias when query involves more than 1 keyspace term, as during formalization of the function expression, is ambiguous as we are not sure which keyspace do we want to evaluate the function on. Example bad query: SELECT meta() FROM landmark, route r WHERE r.sourceairport = \"TLV\" LIMIT 100;",
		},
		Actions: []string{
			"change the call to accept the keyspace alias as an argument, so for the example bad query to get metadata for route keyspace -> SELECT meta(r) FROM landmark, route r WHERE r.sourceairport = \"TLV\" LIMIT 100;",
		},
		IsUser: true,
	},
	E_INDEXER_DESC_COLLATION: {
		Code:        4320,
		ErrorCode:   "E_INDEXER_DESC_COLLATION",
		Description: "DESC option is not supported by the indexer.",
		Causes: []string{
			"Currently only gsi's indexAPI 1 doesn't support DESC option on index keys",
		},
		Actions: []string{
			"upgrade to higher version of indexing service that supports DESC options on index keys",
		},
	},
	E_PLAN_INTERNAL: {
		Code:        4321,
		ErrorCode:   "E_PLAN_INTERNAL",
		Description: "error raised during planning when things are not going as planned by the builder( NEED TO GO THROUGH EVERY code path)",
		Causes:      []string{},
		Actions:     []string{},
	},
	E_ALTER_INDEX: {
		Code:        4322,
		ErrorCode:   "E_ALTER_INDEX",
		Description: "ALTER INDEX not supported",
		Causes: []string{
			"ALTER INDEX statement is supported only from (gsi)indexer API3 onwards",
		},
		Actions: []string{
			"upgrade to a higher version of indexing service ",
		},
	},
	E_PLAN_NO_PLACEHOLDER: {
		Code:        4323,
		ErrorCode:   "E_PLAN_NO_PLACEHOLDER",
		Description: "Unable to reproduce -> use named/pos param as keyspace term ",
		Causes:      []string{},
		Actions:     []string{},
	},
	E_NO_ANSI_JOIN: {
		Code:        4330,
		ErrorCode:   "E_NO_ANSI_JOIN",
		Description: "No index available for ANSI [JOIN/NEST] term [keyspace-alias]",
		Causes: []string{
			"ANSI JOIN/NEST require suitable secondary index for the keyspace terms involved. Unless join is 1) primaryJoin (Joining on primary key meta().id) , 2) nestedloop primary scan( only for keyspaces with less than 1000 documents) , 3) if hash join is being considered if ON CLAUSE or JOIN predicate is an equality predicate, 4) keyspace used is from system scope",
		},
		Actions: []string{
			"CREATE a suitable index secondary index, link to index advisor https://docs.couchbase.com/server/current/guides/index-advisor.html#advice-single ",
		},
		IsUser: true,
	},
	E_PARTITION_INDEX_NOT_SUPPORTED: {
		Code:        4340,
		ErrorCode:   "E_PARTITION_INDEX_NOT_SUPPORTED",
		Description: "PARTITION index is not supported by indexer.",
		Causes: []string{
			"Index Partitioning https://docs.couchbase.com/server/current/n1ql/n1ql-language-reference/index-partitioning.html clause in CREATE INDEX statement ( PARTITION BY HASH ([exprs] ) )is supported from Indexer API3 onwards",
		},
		Actions: []string{
			"upgrade to a higher version of indexing service ",
		},
	},
	E_ENCODED_PLAN_NOT_ALLOWED: {
		Code:        4400,
		ErrorCode:   "E_ENCODED_PLAN_NOT_ALLOWED",
		Description: "Encoded plan use is not allowed in serverless mode.",
		Causes:      []string{},
		Actions:     []string{},
	},
	E_CBO: {
		Code:        4600,
		ErrorCode:   "E_CBO",
		Description: "UNUSED",
		Causes:      []string{},
		Actions:     []string{},
	},
	E_INDEX_STAT: {
		Code:        4610,
		ErrorCode:   "E_INDEX_STAT",
		Description: "Invalid index statistics for index [index_name] : [state_name]",
		Causes: []string{
			"internal error in ready index stats from index storage api for cbo logic",
		},
		Actions: []string{
			"contact support",
		},
		IsUser: false,
	},
	E_EXECUTION_PANIC: {
		Code:        5001,
		ErrorCode:   "E_EXECUTION_PANIC",
		Description: "golang panic in source",
		Causes: []string{
			"when there is a panic in runtime we recover and abort the request while logging the panic and also return the same as a response.\n2 points where we might panic-> within the execution of an operator or when servicing the request.\n\nReasons:\n1. nil pointer dereference\n2. index out of range\n3. closing a closed channel\n4. send on a closed channel, etc",
		},
		Actions: []string{
			"Contact support",
		},
		IsUser: false,
	},
	E_EXECUTION_INTERNAL: {
		Code:        5002,
		ErrorCode:   "E_EXECUTION_INTERNAL",
		Description: "Execution internal error: ",
		Causes:      []string{},
		Actions:     []string{},
	},
	E_EXECUTION_PARAMETER: {
		Code:        5003,
		ErrorCode:   "E_EXECUTION_PARAMETER",
		Description: "Execution parameter error: [reason]",
		Causes: []string{
			"1) cannot have USING Clause and named parameters when using EXECUTE statement.\n    for eg:\n    GET /query/service?statement=EXECUTE p1 USING {\"name\":\"me\"};&$name=1\n\n{\n\"requestID\": \"74cb2d68-55d9-416c-b049-0a6287cf59c0\",\n\"errors\": [{\"code\":5003,\"msg\":\"Execution parameter error: cannot have both USING clause and request parameters\"}],\n\"status\": \"fatal\",\n\"metrics\": {\"elapsedTime\": \"854.667\u00b5s\",\"executionTime\": \"800.5\u00b5s\",\"resultCount\": 0,\"resultSize\": 0,\"serviceLoad\": 2,\"errorCount\": 1}\n}",
			"2) EXECUTE <prepare_name> USING <expr>;\n  USING Clause is expected to be static, i.e either an array or an object. But not any other expression types\n for eg: a subquery expression is not static\n EXECUTE p1 USING {\"name\":(SELECT * FROM default)};\n{\n\"requestID\": \"8a1863d4-23c9-4a6c-b755-8fde09abe0cc\",\n\"errors\": [{\"code\":5003,\"msg\":\"Execution parameter error: USING clause does not evaluate to static values\"}],\n\"status\": \"fatal\",\n\"metrics\": {\"elapsedTime\": \"340.333\u00b5s\",\"executionTime\": \"310.5\u00b5s\",\"resultCount\": 0,\"resultSize\": 0,\"serviceLoad\": 2,\"errorCount\": 1}\n}",
		},
		Actions: []string{
			"1) either USING or request paremeter as named_arguments , not both.",
			"2) USING Clause -> expression typically expects array construct (positional parameters: [1,2,3]) or object construct ( with fieldnames as namedparameter and field value as it's value, {\"name\":\"nobody\"})",
		},
		IsUser: true,
	},
	E_PARSING: {
		Code:        5004,
		ErrorCode:   "E_PARSING",
		Description: "Expression parsing: [expression] failed",
		Causes: []string{
			"User has EXCLUDE clause in the projection.\n\nEXCLUDE clause passed uses string based referencing, but when trying to parse raw string(or comma split string term) we get an error as the string is not a valid expression.",
		},
		Actions: []string{
			"Please change the string passed to a valid expression.\n\nAs a way to debug use\nSELECT <expr> FROM <any-keyspace>; // if it errors here it will error in Exclude Clause too.",
		},
		IsUser: true,
	},
	E_EXECUTION_KEY_VALIDATION: {
		Code:        5006,
		ErrorCode:   "E_EXECUTION_KEY_VALIDATION",
		Description: "Out of key validation space.",
		Causes: []string{
			"When upserting/inserting to ensure we aren't updating the same key more than once. There is a skip keys mechanism in place to ensure we avoid halloween problems, but if the available free system memory doesn't satisfy minimun requirement(134217728 bytes)  to track incoming keys for skip mechanism we error out with this error regardless of if the insert/upsert is Value based or Select based.",
		},
		Actions: []string{
			"simplest solution: reduce insert original insert into 2 statements.\n\nor: one could play around with the pipeline_batch https://docs.couchbase.com/server/current/settings/query-settings.html#pipeline_batch_req  & max_parallelism settings https://docs.couchbase.com/server/current/settings/query-settings.html#max_parallelism_req  so the operator can take advantage of that to reduce number of documents batched together.",
		},
		IsUser: false,
	},
	E_EXECUTION_CURL: {
		Code:        5007,
		ErrorCode:   "E_EXECUTION_CURL",
		Description: "Error executing CURL function",
		Causes: []string{
			"1) No host in request URL.\n[\n  {\n    \"code\": 5010,\n    \"msg\": \"Error evaluating projection\",\n    \"reason\": {\n      \"_level\": \"exception\",\n      \"caller\": \"func_curl:190\",\n      \"cause\": {\n        \"error\": \"No host in request URL.\"\n      },\n      \"code\": 5007,\n      \"key\": \"execution.curl\",\n      \"message\": \"Error executing CURL function\"\n    }\n  }\n]",
			"2) /diag/eval prefix path is restricted\n SELECT CURL(\"http://127.0.0.1:9000/diag/eval\", {\"user\":\"Administrator:password\"});\n{\n    \"requestID\": \"b78e9958-8b61-469c-94a9-b81435834198\",\n    \"signature\": {\n        \"$1\": \"object\"\n    },\n    \"results\": [\n    ],\n    \"errors\": [\n        {\n            \"code\": 5010,\n            \"msg\": \"Error evaluating projection\",\n            \"reason\": {\n                \"_level\": \"exception\",\n                \"caller\": \"func_curl:191\",\n                \"cause\": {\n                    \"error\": \"Access restricted - http://127.0.0.1:9000/diag/eval.\"\n                },\n                \"code\": 5007,\n                \"key\": \"execution.curl\",\n                \"message\": \"Error executing CURL function\"\n            }\n        }\n    ],",
			"3) options header-> expects string array.\nSELECT CURL(\"http://127.0.0.1:9000/pools\", {\"header\":[1,\"Authorization:Basic QWRtaW5pc3RyYXRvcjpwYXNzd29yZA==\"]});\n\"errors\": [\n        {\n            \"code\": 5010,\n            \"msg\": \"Error evaluating projection\",\n            \"reason\": {\n                \"_level\": \"exception\",\n                \"caller\": \"func_curl:191\",\n                \"cause\": {\n                    \"error\": \"Incorrect type for header option 1 in CURL. Header option should be a string value or an array of strings.  \"\n                },\n                \"code\": 5007,\n                \"key\": \"execution.curl\",\n                \"message\": \"Error executing CURL function\"\n            }\n        }\n    ],",
			"4) expect get options value to be boolean\ncbq> SELECT CURL(\"http://127.0.0.1:9000/pools\", {\"get\":\"YES\"});\n{\n    \"requestID\": \"af3dc78e-3d52-4386-b0f5-bd8ac80457ff\",\n    \"signature\": {\n        \"$1\": \"object\"\n    },\n    \"results\": [\n    ],\n    \"errors\": [\n        {\n            \"code\": 5010,\n            \"msg\": \"Error evaluating projection\",\n            \"reason\": {\n                \"_level\": \"exception\",\n                \"caller\": \"func_curl:191\",\n                \"cause\": {\n                    \"error\": \"Incorrect type for get option in CURL \"\n                },\n                \"code\": 5007,\n                \"key\": \"execution.curl\",\n                \"message\": \"Error executing CURL function\"\n            }\n        }\n    ],",
			"5) Only GET and POST requests are supported, option request must be \"GET\" or \"POST\"\ncbq> SELECT CURL(\"http://127.0.0.1:9000/pools\", {\"request\":\"PUT\"});\n{\n    \"requestID\": \"e35e9486-7a58-4f2b-b7f0-d9c27fb0c1f1\",\n    \"signature\": {\n        \"$1\": \"object\"\n    },\n    \"results\": [\n    ],\n    \"errors\": [\n        {\n            \"code\": 5010,\n            \"msg\": \"Error evaluating projection\",\n            \"reason\": {\n                \"_level\": \"exception\",\n                \"caller\": \"func_curl:191\",\n                \"cause\": {\n                    \"error\": \"CURL only supports GET and POST requests. \"\n                },\n                \"code\": 5007,\n                \"key\": \"execution.curl\",\n                \"message\": \"Error executing CURL function\"\n            }\n        }\n    ]",
			"6) Timeout error\ncbq> SELECT CURL(\"https://httpbin.org/delay/10\", {\"max-time\":5});\n{\n    \"requestID\": \"86a41638-a017-466a-b752-65c11ed97486\",\n    \"signature\": {\n        \"$1\": \"object\"\n    },\n    \"results\": [\n    ],\n    \"errors\": [\n        {\n            \"code\": 5010,\n            \"msg\": \"Error evaluating projection\",\n            \"reason\": {\n                \"_level\": \"exception\",\n                \"caller\": \"func_curl:191\",\n                \"cause\": {\n                    \"error\": \"curl: Timeout was reached\"\n                },\n                \"code\": 5007,\n                \"key\": \"execution.curl\",\n                \"message\": \"Error executing CURL function\"\n            }\n        }\n    ]",
			"7) hostname is not in allowed URLs (or) is in the disallowed URLs\ncbq> SELECT CURL(\"http://127.0.0.1:9000/pools\");\n{\n    \"requestID\": \"9fc94bba-7385-443b-9fd6-18bf404e2dbf\",\n    \"signature\": {\n        \"$1\": \"object\"\n    },\n    \"results\": [\n    ],\n    \"errors\": [\n        {\n            \"code\": 5010,\n            \"msg\": \"Error evaluating projection\",\n            \"reason\": {\n                \"_level\": \"exception\",\n                \"caller\": \"func_curl:191\",\n                \"cause\": {\n                    \"error\": \"The end point http://127.0.0.1:9000/pools is not permitted.  List allowed end points in the configuration.\"\n                },\n                \"code\": 5007,\n                \"key\": \"execution.curl\",\n                \"message\": \"Error executing CURL function\"\n            }\n        }\n    ],",
		},
		Actions: []string{
			"1) only allow http or https scheme",
			"2) expect hostname in the URL:\nExample: www.example.com\nThe host identifies the domain name or IP address of the server where the resource is located",
			"3) diag/eval is restricted as we don't want to allow users execute arbitarary code on the server.",
			"4) headers option: must be a string(\"[header]:[value\") or array of strings",
			"5) set max-time and connect-timeout to a value that avoids the timeout:)",
			"6) add URL in the allowed list https://docs.couchbase.com/server/current/n1ql/n1ql-language-reference/curl.html#curl-access-list",
		},
		IsUser: true,
	},
	E_EXECUTION_STATEMENT_STOPPED: {
		Code:        5008,
		ErrorCode:   "E_EXECUTION_STATEMENT_STOPPED",
		Description: "Execution of statement has been stopped.",
		Causes: []string{
			"When the outer query is stopped any query's started by the udf call in the outer query must also be stopped this error is raised.\n\nFor eg:\nudf library->\nfunction dummy(a,b) {\n  var q = SELECT * FROM `test`;\n  var res = [];\n  for(const doc of q) {\n      res.push(doc);\n  }\n  \n  return res;\n}\n\n\nfunction-def->\nCREATE FUNCTION dummy(lat, lon)\n  LANGUAGE JAVASCRIPT AS \"dummy\" AT \"dummy\";\n\nouter query:\nSELECT * FROM `travel-sample`.inventory.route r WHERE r.airport in dummy(1,2);,\n\ncancel the query via ui.",
		},
		Actions: []string{
			"Letting the outer query finish would result in this error never being raised.",
		},
		IsUser: true,
	},
	E_EVALUATION_ABORT: {
		Code:        5010,
		ErrorCode:   "E_EVALUATION_ABORT",
		Description: "",
		Causes:      []string{},
		Actions:     []string{},
	},
	E_EVALUATION: {
		Code:        5011,
		ErrorCode:   "E_EVALUATION",
		Description: "",
		Causes: []string{
			"for",
		},
		Actions: []string{},
	},
	E_EXPLAIN: {
		Code:        5015,
		ErrorCode:   "E_EXPLAIN",
		Description: "EXPLAIN: Error marshaling JSON.",
		Causes: []string{
			"Error when in marshalling plan. MarshalJson() method for a particular operator might be the cause.",
		},
		Actions: []string{
			"Contact support",
		},
		IsUser: false,
	},
	E_EXPLAIN_FUNCTION: {
		Code:        5017,
		ErrorCode:   "E_EXPLAIN_FUNCTION",
		Description: "EXPLAIN FUNCTION: [reason]",
		Causes: []string{
			"1) When running EXPLAIN FUNCTION <func_name>",
			"2) after getting query plans used in inline udfs / javascript udfs. There was an error when marshalling the plans.\n2) after getting statement strings from the evaluator , we error out during when we are building the plan for the statement strings received, this may be due to i) failing to parse to a valid algebra node ii) if we have a transaction started, i.e we are executing in the context of a transaction , error is raised for all statements not supported during a transaction. iii) incorrect semantics in the statement. iv) failed building plan from the algebra statement we got from parsing.",
		},
		Actions: []string{
			"Amend your usage by understanding from the causes where EXPLAIN FUNCTION can't be used.",
		},
		IsUser: true,
	},
	E_GROUP_UPDATE: {
		Code:        5020,
		ErrorCode:   "E_GROUP_UPDATE",
		Description: "Error updating initial GROUP value/ intermediate GROUP value/ final GROUP value",
		Causes: []string{
			"User has a query with GROUP BY Clause and aggregates ,\n\nWhile computing aggregates the execution logic is dividied into 3 operators, 1) GroupInit , 2) GroupIntermediate, 3) GroupFinal\neg: query: SELECT city, COUNT(name)\nFROM `grouptestsmall`\nGROUP BY city;\nGroupInit\n* Collect group keys (city)\n* seed default value for the aggregate ( for count()-> 0)\n* cumulate initial per item in the group for the aggregate ( for count(name) where city=\"A\" add 1 for every item that has city=\"A\" to the aggregate value)\n\nGroupIntermediate\n* Typically used to merge the partial results in a multinode setup\n\nGroupFinal\n* throw out final cumulated value per group. ({\"city\": \"A\", \"count\":100} , if 100 docs with city as \"A\")\n\nthe error is raised when\n1) incorrect seed ( default value for the aggregate)\n2) incorrect partial value ( Count() aggregate expects Number but got string)",
		},
		Actions: []string{
			"Contact support, error in the logic of group aggregation updates. This is a bug!",
		},
		IsUser: false,
	},
	E_INVALID_VALUE: {
		Code:        5030,
		ErrorCode:   "E_INVALID_VALUE",
		Description: "",
		Causes:      []string{},
		Actions:     []string{},
	},
	E_INVALID_EXPRESSION: {
		Code:        5031,
		ErrorCode:   "E_INVALID_EXPRESSION",
		Description: "",
		Causes:      []string{},
		Actions:     []string{},
	},
	E_UNSUPPORTED_EXPRESSION: {
		Code:        5032,
		ErrorCode:   "E_UNSUPPORTED_EXPRESSION",
		Description: "",
		Causes:      []string{},
		Actions:     []string{},
	},
	E_RANGE: {
		Code:        5035,
		ErrorCode:   "E_RANGE",
		Description: "Out of range evaluating [term]",
		Causes: []string{
			"1) Builtin functions that support range operations have their capacity ceiling as maxint32(2147483647).",
			"2) SELECT DATE_RANGE_STR(\"0001-01-01\", \"9999-12-31\", \"minute\");\n[\n  {\n    \"code\": 5010,\n    \"msg\": \"Error evaluating projection\",\n    \"reason\": {\n      \"_level\": \"exception\",\n      \"caller\": \"func_date:1213\",\n      \"code\": 5035,\n      \"key\": \"execution.range_error\",\n      \"message\": \"Out of range evaluating DATE_RANGE_STR().\"\n    }\n  }\n]",
			"3) SELECT ARRAY_REPEAT(\"a\",2147483648);\n[\n  {\n    \"code\": 5010,\n    \"msg\": \"Error evaluating projection\",\n    \"reason\": {\n      \"_level\": \"exception\",\n      \"caller\": \"func_array:1813\",\n      \"code\": 5035,\n      \"key\": \"execution.range_error\",\n      \"message\": \"Out of range evaluating ARRAY_REPEAT().\"\n    }\n  }\n]",
			"4) SELECT ARRAY_RANGE(0, 2147483648) AS gen_array_range;\n[\n  {\n    \"code\": 5010,\n    \"msg\": \"Error evaluating projection\",\n    \"reason\": {\n      \"_level\": \"exception\",\n      \"caller\": \"func_array:1634\",\n      \"code\": 5035,\n      \"key\": \"execution.range_error\",\n      \"message\": \"Out of range evaluating ARRAY_RANGE().\"\n    }\n  }\n]",
			"5) SELECT DATE_RANGE_MILLIS(1672531200, 5967734400, \"millisecond\"); // 2023-01-01 to 2159-02-10 milliseconds\n[\n  {\n    \"code\": 5010,\n    \"msg\": \"Error evaluating projection\",\n    \"reason\": {\n      \"_level\": \"exception\",\n      \"caller\": \"func_date:1405\",\n      \"code\": 5035,\n      \"key\": \"execution.range_error\",\n      \"message\": \"Out of range evaluating DATE_RANGE_MILLIS().\"\n    }\n  }\n]",
			"6) SELECT REPEAT(\"A\", 2147483648);\n[\n  {\n    \"code\": 5010,\n    \"msg\": \"Error evaluating projection\",\n    \"reason\": {\n      \"_level\": \"exception\",\n      \"caller\": \"func_str:482\",\n      \"code\": 5035,\n      \"key\": \"execution.range_error\",\n      \"message\": \"Out of range evaluating REPEAT().\"\n    }\n  }\n]",
		},
		Actions: []string{
			"TRUE",
		},
	},
	W_DIVIDE_BY_ZERO: {
		Code:        5036,
		ErrorCode:   "W_DIVIDE_BY_ZERO",
		Description: "",
		Causes: []string{
			"1) division by zero leads to a warning and \"null\" as the result, where underlying finction are \"DIV\" or \"IDIV\" and the 2nd operand in 0.",
			"2) cbq> SELECT 1/0;\n{\n    \"requestID\": \"c746a7d9-1d4d-46d2-a4ae-a2bc64fc7ea1\",\n    \"signature\": {\n        \"$1\": \"number\"\n    },\n    \"results\": [\n    {\n        \"$1\": null\n    }\n    ],\n    \"warnings\": [\n        {\n            \"code\": 5036,\n            \"msg\": \"Division by 0.\"\n        }\n    ],",
			"3) cbq> SELECT DIV(1,0);\n{\n    \"requestID\": \"825dad4b-46c7-45c7-8407-fd686fef8a93\",\n    \"signature\": {\n        \"$1\": \"number\"\n    },\n    \"results\": [\n    {\n        \"$1\": null\n    }\n    ],\n    \"warnings\": [\n        {\n            \"code\": 5036,\n            \"msg\": \"Division by 0.\"\n        }\n    ],",
			"4) cbq> SELECT IDIV(1,0);\n{\n    \"requestID\": \"1cf852c0-838c-4cc1-9bc6-11a40679ec9d\",\n    \"signature\": {\n        \"$1\": \"number\"\n    },\n    \"results\": [\n    {\n        \"$1\": null\n    }\n    ],\n    \"warnings\": [\n        {\n            \"code\": 5036,\n            \"msg\": \"Division by 0.\"\n        }\n    ],",
		},
		Actions: []string{
			"2nd operand can be anythong expect 0 , this way we avoid the warning.",
		},
		IsUser: true,
	},
	E_DUPLICATE_FINAL_GROUP: {
		Code:        5040,
		ErrorCode:   "E_DUPLICATE_FINAL_GROUP",
		Description: "Duplicate Final Group.",
		Causes: []string{
			"Group by is done by 3 operators\n1. group initial\ncumulate aggregate per group key\n\n2. group intermediate\nmerge(from  partial cumulatives that have the same key)\n\n3. group final\nsend the final cumulative values to the next operator.\n\nBut here something went wrong in groupIntermediate \nas in the groupfinal operator we receive the same group key again, i.e they haven't been merged in the intermediate group operator",
		},
		Actions: []string{
			"Contact support , this might be a bug!",
		},
		IsUser: false,
	},
	E_INSERT_KEY: {
		Code:        5050,
		ErrorCode:   "E_INSERT_KEY",
		Description: "No INSERT key for [document]",
		Causes: []string{
			"User has run INSERT ... VALUES statement https://docs.couchbase.com/server/current/n1ql/n1ql-language-reference/insert.html#insert-values \n\nBut the annotated value sent to sendInsertoperator doesn't have a \"key\" attachment that is set in the ValueScan operator.\nThis means something has gone wrong when sending the annotated value(with the new \"key\" & \"value\" attachment) from valueScan operator over the operator's valueexchange queue.",
		},
		Actions: []string{
			"Contact support , this might be a bug, as code is present for fail-safe purpose!",
		},
		IsUser: false,
	},
	E_INSERT_VALUE: {
		Code:        5060,
		ErrorCode:   "E_INSERT_VALUE",
		Description: "No INSERT value for [document]",
		Causes: []string{
			"User has run INSERT ... SELECT statement https://docs.couchbase.com/server/current/n1ql/n1ql-language-reference/insert.html#insert-select \n\nBut the annotated value sent to sendInsertoperator doesn't have a \"value\" attachment that is set in the ValueScan operator.\nThis means something has gone wrong when sending the annotated value(with the new \"key\" & \"value\" attachment) from valueScan operator over the the operator's valueexchange queue.",
		},
		Actions: []string{
			"Contact support , this might be a bug, as code is present for fail-safe purpose!",
		},
		IsUser: false,
	},
	E_INSERT_KEY_TYPE: {
		Code:        5070,
		ErrorCode:   "E_INSERT_KEY_TYPE",
		Description: "Cannot INSERT non-string key [key-passed-value] of type [key-passed-type]",
		Causes: []string{
			"1) The key for a document must always be a string, this applies to\n1) INSERT ...VALUE statement https://docs.couchbase.com/server/current/n1ql/n1ql-language-reference/insert.html#insert-values :\n \nINSERT INTO `travel-sample`.inventory.airline ( KEY, VALUE ) VALUES ( 1, { \"id\": \"01\", \"type\": \"airline\"} ) RETURNING META().id as docid, *;\n\n{\n    \"requestID\": \"d5c66327-63a8-431e-a6d7-f49e0da46341\",\n    \"signature\": {\n        \"docid\": \"json\",\n        \"*\": \"*\"\n    },\n    \"results\": [\n    ],\n    \"errors\": [\n        {\n            \"code\": 5070,\n            \"msg\": \"Cannot INSERT non-string key 1 of type value.intValue.\"\n        }\n    ]",
			"2) MERGE ... WHEN NOT MATCHED INSERT\ncbq> MERGE INTO `travel-sample`.inventory.airport AS target USING [ {\"iata\":\"DSA\", \"name\": \"Doncaster Sheffield Airport\"}, {\"iata\":\"VLY\", \"name\": \"Anglesey Airport / Maes Awyr M\u00f4n\"} ] AS source ON target.faa = source.iata WHEN NOT MATCHED THEN INSERT (KEY 1+to_number(UUID()), VALUE {\"faa\": source.iata, \"airportname\": source.name, \"type\": \"airport\", \"inserted\": true}, OPTIONS {\"expiration\": 7*24*60*60} );\n{\n    \"requestID\": \"f74adb33-bdec-401c-8bd5-bee2b6071cb2\",\n    \"signature\": null,\n    \"results\": [\n    ],\n    \"errors\": [\n        {\n            \"code\": 5070,\n            \"msg\": \"Cannot INSERT non-string key null of type *value.nullValue.\"\n        }\n    ],",
		},
		Actions: []string{
			"Ensure the exression for KEY in INSERT ... VALUE is always producing a string( that is unique obviously ).",
		},
		IsUser: true,
	},
	E_INSERT_OPTIONS_TYPE: {
		Code:        5071,
		ErrorCode:   "E_INSERT_OPTIONS_TYPE",
		Description: "Cannot INSERT non-OBJECT options [options-object-input] of type [options-object-input-Type]",
		Causes: []string{
			"Both for INSERT .. VALUES https://docs.couchbase.com/server/current/n1ql/n1ql-language-reference/insert.html#values-clause \nand INSERT SELECT https://docs.couchbase.com/server/current/n1ql/n1ql-language-reference/insert.html#insert-select \n\nOptions passed is expected to be a object with \"expiration\" being the only recognized options field that takes document expiration time as a number (seconds)\n\nExample bad query:\n INSERT INTO `travel-sample`.inventory.airline (KEY, VALUE, OPTIONS) VALUES ( \"airline::ttl\", { \"callsign\": \"Temporary\", \"country\" : \"USA\", \"type\" : \"airline\" }, \"HEELE\");\n\n{\n    \"requestID\": \"925ce38b-1576-4d96-800e-227d6ed9e60f\",\n    \"signature\": null,\n    \"results\": [\n    ],\n    \"errors\": [\n        {\n            \"code\": 5071,\n            \"msg\": \"Cannot INSERT non-OBJECT options \\\"HEELE\\\" of type value.stringValue.\"\n        }\n    ]",
		},
		Actions: []string{},
		IsUser:  true,
	},
	E_UPSERT_KEY: {
		Code:        5072,
		ErrorCode:   "E_UPSERT_KEY",
		Description: " No UPSERT key for [annotated value ]",
		Causes: []string{
			"User has run UPSERT ... VALUES statement https://docs.couchbase.com/server/current/n1ql/n1ql-language-reference/upsert.html#insert-values  \nBut the annotated value sent to sendUpsert operator doesn't have a \"key\" attachment that is set in the ValueScan operator. \nThis means something has gone wrong when sending the annotated value(with the new \"key\" & \"value\" attachment) from valueScan operator over the operator's valueexchange queue.",
		},
		Actions: []string{
			"Contact support, this is a bug as code here is for fail-safe purpose ",
		},
		IsUser: false,
	},
	E_UPSERT_KEY_ALREADY_MUTATED: {
		Code:        5073,
		ErrorCode:   "E_UPSERT_KEY_ALREADY_MUTATED",
		Description: "Cannot act on the same key multiple times in an UPSERT statement",
		Causes: []string{
			"we track keys to not mutate the same keys more than once under skip key mechanism.\n\nexample bad query:\nUPSERT INTO landmark (KEY, VALUE)\nVALUES (\"upsert-1\", { \"name\": \"The Minster Inn\", \"type\": \"landmark-pub\"}),\n(\"upsert-1\", {\"name\": \"The Black Swan\", \"type\": \"landmark-pub\"})\nRETURNING VALUE name;\n[\n  {\n    \"code\": 5073,\n    \"msg\": \"Cannot act on the same key multiple times in an UPSERT statement.\",\n    \"reason\": {\n      \"key\": \"upsert-1\",\n      \"keyspace\": \"default:travel-sample.inventory.landmark\"\n    }\n  }\n]",
		},
		Actions: []string{
			"Ensure upsert pairs have key as a unique string",
		},
		IsUser: true,
	},
	E_UPSERT_VALUE: {
		Code:        5075,
		ErrorCode:   "E_UPSERT_VALUE",
		Description: "No UPSERT value for [annotated value]",
		Causes: []string{
			"User has run UPSERT ... SELECT statement https://docs.couchbase.com/server/current/n1ql/n1ql-language-reference/upsert.html#insert-select \nBut the annotated value sent to sendUpsert operator doesn't have a \"key\" attachment that is set in the ValueScan operator. \nThis means something has gone wrong when sending the annotated value(with the new \"key\" & \"value\" attachment) from valueScan operator over the operator's valueexchange queue.",
		},
		Actions: []string{
			"Contact support, this is a bug as code here is for fail-safe purpose ",
		},
		IsUser: false,
	},
	E_UPSERT_KEY_TYPE: {
		Code:        5078,
		ErrorCode:   "E_UPSERT_KEY_TYPE",
		Description: "Cannot UPSERT non-string key [key-value-passed] of type [key-passed-type].",
		Causes: []string{
			"Both for UPSERT .. VALUES https://docs.couchbase.com/server/current/n1ql/n1ql-language-reference/upsert.html#insert-values \nand UPSERT SELECT https://docs.couchbase.com/server/current/n1ql/n1ql-language-reference/upsert.html#insert-select  \n\nOptions passed is expected to be a object with \"expiration\" being the only recognized options field that takes document expiration time as a number (seconds)\n\nExample bad query:\n INSERT INTO `travel-sample`.inventory.airline (KEY, VALUE, OPTIONS) VALUES ( \"airline::ttl\", { \"callsign\": \"Temporary\", \"country\" : \"USA\", \"type\" : \"airline\" }, \"HEELE\");\n\n{\n    \"requestID\": \"925ce38b-1576-4d96-800e-227d6ed9e60f\",\n    \"signature\": null,\n    \"results\": [\n    ],\n    \"errors\": [\n        {\n            \"code\": 5071,\n            \"msg\": \"Cannot INSERT non-OBJECT options \\\"HEELE\\\" of type value.stringValue.\"\n        }\n    ]",
		},
		Actions: []string{
			"As couchbase only supports document-keys of string type, always use KEY expression in upsert statement to produce a string string value( which is unique).\nfor eg: builtin uuid() function.",
		},
		IsUser: true,
	},
	E_UPSERT_OPTIONS_TYPE: {
		Code:        5079,
		ErrorCode:   "E_UPSERT_OPTIONS_TYPE",
		Description: "UNUSED never raised",
		Causes:      []string{},
		Actions:     []string{},
	},
	E_DELETE_ALIAS_MISSING: {
		Code:        5080,
		ErrorCode:   "E_DELETE_ALIAS_MISSING",
		Description: "DELETE alias [keyspace-alias] not found in item [annotated value]",
		Causes: []string{
			"fail-safe code , error should ideally never occur\n\nThis is to catch the case where the previous operator( Scan op) has not setfield for the item being passed with the keyspace_ref's alias.",
		},
		Actions: []string{
			"Contact Support! this is a bug",
		},
		IsUser: false,
	},
	E_DELETE_ALIAS_METADATA: {
		Code:        5090,
		ErrorCode:   "E_DELETE_ALIAS_METADATA",
		Description: "DELETE alias [keyspace-alias] has no metadata in item.",
		Causes: []string{
			"fail-safe code , error should ideally never occur\n\nThis is to catch the case where the previous operator( Scan op), has passed an item whose value for the fieldname as keyspace-alias has no metadata, i.e not an annotated value(couchbase document abstraction for the query service).",
		},
		Actions: []string{
			"Contact Support! this is a bug",
		},
		IsUser: false,
	},
	E_UPDATE_ALIAS_MISSING: {
		Code:        5100,
		ErrorCode:   "E_UPDATE_ALIAS_MISSING",
		Description: "UPDATE alias [keyspace-alias] not found in item",
		Causes: []string{
			"fail-safe code,  error should ideally never occur\n\nThis is to catch the case where the previous operator( Scan op/ keyscan) has not setfield for the item being passed with the keyspace_ref's alias.",
		},
		Actions: []string{
			"Contact Support! this is a bug",
		},
		IsUser: false,
	},
	E_UPDATE_ALIAS_METADATA: {
		Code:        5110,
		ErrorCode:   "E_UPDATE_ALIAS_METADATA",
		Description: "UPDATE alias [keyspace-alias] has no metadata in item.",
		Causes: []string{
			"fail-safe code , error should ideally never occur\n\nThis is to catch the case where the previous operator( Scan op), has passed an item whose value for the fieldname as keyspace-alias has no metadata, i.e not an annotated value(couchbase document abstraction for the query service).",
		},
		Actions: []string{
			"Contact Support! this is a bug",
		},
		IsUser: false,
	},
	E_UPDATE_MISSING_CLONE: {
		Code:        5120,
		ErrorCode:   "E_UPDATE_MISSING_CLONE",
		Description: "Missing UPDATE clone.",
		Causes: []string{
			"fail-safe code, error should ideally never occur\n\nThis is to catch the case where the clone operator has not set the \"clone\" attachment in the item so set operator can modify the clone-annotated value.",
		},
		Actions: []string{
			"Contact Support! this is a bug",
		},
		IsUser: false,
	},
	E_UNNEST_INVALID_POSITION: {
		Code:        5180,
		ErrorCode:   "E_UNNEST_INVALID_POSITION",
		Description: "Invalid UNNEST position of type [pos_type]",
		Causes: []string{
			"fail-safe code, error should ideally never occur\n\nUser has used UNNEST_POS function https://docs.couchbase.com/server/current/n1ql/n1ql-language-reference/metafun.html#unnest-pos  ,\nfor eg: SELECT UNNEST_POS(u) AS upos, u FROM [{\"a1\":[10,9,4]}] AS d UNNEST d.a1 AS u; \n\nThis is to catch invalid 'unnest_position' attachment in the item set by unnest operator. Expected to be of integer type.",
		},
		Actions: []string{
			"Contact Support! this is a bug\n",
		},
		IsUser: false,
	},
	E_SCAN_VECTOR_TOO_MANY_SCANNED_BUCKETS: {
		Code:        5190,
		ErrorCode:   "E_SCAN_VECTOR_TOO_MANY_SCANNED_BUCKETS",
		Description: "The scan_vector parameter should not be used for queries accessing more than one keyspace.  Use scan_vectors instead. Keyspaces: [buckets...]",
		Causes: []string{
			"The client has specified scan_vector as a part of the request parameter https://docs.couchbase.com/server/current/n1ql/n1ql-rest-api/index.html \nbut query has more than one keyspace, hence we error out.\n\nexample bad request:\nGET query/service?statement=SELECT default.*, grouptestsmall.* FROM default, grouptestsmall ;&scan_vector=[[0, \"abc\"],[0, \"abc\"],[0, \"abc\"],[0, \"abc\"],[0, \"abc\"],[0, \"abc\"],[0, \"abc\"],[0, \"abc\"],[0, \"abc\"],[0, \"abc\"],[0, \"abc\"],[0, \"abc\"],[0, \"abc\"],[0, \"abc\"],[0, \"abc\"],[0, \"abc\"],[0, \"abc\"],[0, \"abc\"],[0, \"abc\"],[0, \"abc\"],[0, \"abc\"],[0, \"abc\"],[0, \"abc\"],[0, \"abc\"],[0, \"abc\"],.....]\n\n{\n\"requestID\": \"5bbcee19-ce52-422c-bfbf-f6bd8327e95f\",\n\"errors\":[\n{\n\"code\": 5190,\n\"msg\": \"The scan_vector parameter should not be used for queries accessing more than one keyspace. Use scan_vectors instead. Keyspaces: [  default:default default:grouptestsmall]\"\n},\n{\n\"code\": 5001,\n\"msg\": \"Panic: runtime error: invalid memory address or nil pointer dereference\"\n}\n],",
		},
		Actions: []string{
			"scan_vectors map must be supplied by client when quering on more than one keyspace, doc reference https://docs.couchbase.com/server/current/n1ql/n1ql-rest-api/index.html#test ",
		},
		IsUser: true,
	},
	E_DYNAMIC_AUTH: {
		Code:        5201,
		ErrorCode:   "E_DYNAMIC_AUTH",
		Description: "Dynamic auth error",
		Causes: []string{
			"Particular case when the FROM Clause has positional/named parameter with USE KEYS clause\n\nBut there was an error in doing a privilege check on the expression passed as a named/positional parameter\n\nFor eg: \nhttp://127.0.0.1:9499/query/service?statement=SELECT d.a FROM $a d USE KEYS [\"key::002a1616-aca9-4c2f-ad87-277d5845bb5a\"]&$p=\"default\"\n{\n\"requestID\": \"3eee8a00-f1b1-488e-a119-a4d0b8853b0a\",\n\"signature\": {\"a\":\"json\"},\n\"results\": [\n],\n\"errors\": [{\"code\":5201,\"msg\":\"Dynamic auth error\",\"reason\":{}}],\n\"status\": \"fatal\",\n\"metrics\": {\"elapsedTime\": \"23.718125ms\",\"executionTime\": \"2.075125ms\",\"resultCount\": 0,\"resultSize\": 0,\"serviceLoad\": 2,\"errorCount\": 1}\n}\n\nPossible causes:\n1) positional / named parameter is not defined as part of the request parameter.",
		},
		Actions: []string{
			"1) make sure the from clause used defined positional/named parameter",
		},
		IsUser: true,
	},
	E_TRANSACTIONAL_AUTH: {
		Code:        5202,
		ErrorCode:   "E_TRANSACTIONAL_AUTH",
		Description: "",
		Causes:      []string{},
		Actions:     []string{},
	},
	E_USER_NOT_FOUND: {
		Code:        5210,
		ErrorCode:   "E_USER_NOT_FOUND",
		Description: "Unable to find user",
		Causes: []string{
			"Internally the execution of GRANT ROLE, gets current users information from /settings/rbac/users endpoint https://docs.couchbase.com/server/current/manage/manage-security/manage-users-and-roles.html#get-user-information-with-the-rest-api \nbut the user received from query doesn't match any user in the usersinformation given by the datastore.\n\nFor eg:\nGET /settings/rbac/users  \n[\n    {\n        \"id\": \"jan\",\n        \"domain\": \"local\",\n        \"roles\": [\n        ],\n        \"groups\": [\n        ],\n        \"external_groups\": [\n        ],\n        \"name\": \"\",\n        \"uuid\": \"f3f8e9ac-94be-4a0c-aa7c-02fac6c51721\",\n        \"password_change_date\": \"2023-11-22T15:13:03.000Z\"\n    }\n]\n\nOnly user is Jan\n\nBut we ran query-> GRANT Replication Admin, Query External Access TO cchaplan, jgleason;\n[\n  {\n    \"code\": 5210,\n    \"msg\": \"Unable to find user local:cchaplan.\",\n    \"reason\": {\n      \"user\": \"local:cchaplan\"\n    }\n  },\n  {\n    \"code\": 5210,\n    \"msg\": \"Unable to find user local:jgleason.\",\n    \"reason\": {\n      \"user\": \"local:jgleason\"\n    }\n  }\n]\n\n\nThe same applies to REVOKE ROLE statement as well.",
		},
		Actions: []string{
			"Use https://docs.couchbase.com/server/current/manage/manage-security/manage-users-and-roles.html#get-user-information-with-the-rest-api \nto check the list of users present.",
		},
		IsUser: true,
	},
	E_ROLE_REQUIRES_KEYSPACE: {
		Code:        5220,
		ErrorCode:   "E_ROLE_REQUIRES_KEYSPACE",
		Description: "Role [role-requested] requires a keyspace. ",
		Causes: []string{
			"Role passed as a part of GRANT/REVOKE ROLE statement, is not an unparameterized role, but a parameterized requires keyspace as these roles are defined on a keyspace\n\nexample bad query:\nGRANT query_select TO jan;\n[\n  {\n    \"code\": 5220,\n    \"msg\": \"Role query_select requires a keyspace.\",\n    \"reason\": {\n      \"role\": \"query_select\"\n    }\n  }\n]",
		},
		Actions: []string{
			"following list of roles require parameterized usage as shown in the docs https://docs.couchbase.com/server/current/n1ql/n1ql-language-reference/grant.html#usage \n\n[\n    {\n        \"role\": \"bucket_admin\",\n        \"bucket_name\": \"*\",\n        \"name\": \"Bucket Admin\",\n        \"desc\": \"Can manage ALL bucket features for a given bucket (including start/stop XDCR). This user can access the web console. This user cannot read data.\"\n    },\n    {\n        \"role\": \"scope_admin\",\n        \"bucket_name\": \"*\",\n        \"scope_name\": \"*\",\n        \"name\": \"Manage Scopes\",\n        \"desc\": \"Can create/delete scopes and collections within a given bucket. This user cannot access the web console.\"\n    },\n    {\n        \"role\": \"bucket_full_access\",\n        \"bucket_name\": \"*\",\n        \"name\": \"Application Access\",\n        \"desc\": \"Full access to bucket data. This user cannot access the web console and is intended only for application access. This user can read and write data except for the _system scope which can only be read.\"\n    },\n    {\n        \"role\": \"views_admin\",\n        \"bucket_name\": \"*\",\n        \"name\": \"Views Admin\",\n        \"desc\": \"Can create and manage views of a given bucket. This user can access the web console. This user can read some data.\"\n    },\n    {\n        \"role\": \"views_reader\",\n        \"bucket_name\": \"*\",\n        \"name\": \"Views Reader\",\n        \"desc\": \"Can read data from the views of a given bucket. This user cannot access the web console and is intended only for application access. This user can read some data.\"\n    },\n    {\n        \"role\": \"data_reader\",\n        \"bucket_name\": \"*\",\n        \"scope_name\": \"*\",\n        \"collection_name\": \"*\",\n        \"name\": \"Data Reader\",\n        \"desc\": \"Can read data from a given bucket, scope or collection. This user cannot access the web console and is intended only for application access. This user can read data, but cannot write it.\"\n    },\n    {\n        \"role\": \"data_writer\",\n        \"bucket_name\": \"*\",\n        \"scope_name\": \"*\",\n        \"collection_name\": \"*\",\n        \"name\": \"Data Writer\",\n        \"desc\": \"Can write data to a given bucket, scope or collection. This user cannot access the web console and is intended only for application access. This user can write data, but cannot read it.\"\n    },\n    {\n        \"role\": \"data_dcp_reader\",\n        \"bucket_name\": \"*\",\n        \"scope_name\": \"*\",\n        \"collection_name\": \"*\",\n        \"name\": \"Data DCP Reader\",\n        \"desc\": \"Can initiate DCP streams for a given bucket, scope or collection. This user cannot access the web console and is intended only for application access. This user can read data.\"\n    },\n    {\n        \"role\": \"data_backup\",\n        \"bucket_name\": \"*\",\n        \"name\": \"Data Backup & Restore\",\n        \"desc\": \"Can backup and restore a given bucket's data. This user cannot access the web console and is intended only for application access. This user can read data.\"\n    },\n    {\n        \"role\": \"data_monitoring\",\n        \"bucket_name\": \"*\",\n        \"scope_name\": \"*\",\n        \"collection_name\": \"*\",\n        \"name\": \"Data Monitor\",\n        \"desc\": \"Can read statistics for a given bucket, scope or collection. This user cannot access the web console and is intended only for application access. This user cannot read data.\"\n    },\n    {\n        \"role\": \"fts_admin\",\n        \"bucket_name\": \"*\",\n        \"name\": \"Search Admin\",\n        \"desc\": \"Can administer all Full Text Search features. This user can access the web console. This user can read some data.\"\n    },\n    {\n        \"role\": \"fts_searcher\",\n        \"bucket_name\": \"*\",\n        \"scope_name\": \"*\",\n        \"collection_name\": \"*\",\n        \"name\": \"Search Reader\",\n        \"desc\": \"Can query Full Text Search indexes for a given bucket, scope or collection. This user can access the web console. This user can read some data.\"\n    }\n]",
		},
		IsUser: true,
	},
	E_ROLE_TAKES_NO_KEYSPACE: {
		Code:        5230,
		ErrorCode:   "E_ROLE_TAKES_NO_KEYSPACE",
		Description: "Role [role-requested] does not take a keyspace.",
		Causes: []string{
			"GET /settings/rbac/roles, roles that don't have bucketname field are unparameterized roles.\n\nSo, we semantically disallow parameterization on such roles.\nExample bad query:\nGRANT query_execute_global_external_functions ON `default`._default._default TO jan;\n[\n  {\n    \"code\": 5230,\n    \"msg\": \"Role query_execute_global_external_functions does not take a keyspace.\",\n    \"reason\": {\n      \"role\": \"query_execute_global_external_functions\"\n    },\n    \"query\": \"GRANT query_execute_global_external_functions ON `default`._default._default TO jan;\"\n  }\n]",
		},
		Actions: []string{
			"roles that expect unparameterized usage\n[\n    {\n        \"role\": \"admin\",\n        \"name\": \"Full Admin\",\n        \"desc\": \"Can manage all cluster features (including security). This user can access the web console. This user can read and write all data.\"\n    },\n    {\n        \"role\": \"ro_admin\",\n        \"name\": \"Read-Only Admin\",\n        \"desc\": \"Can view all cluster statistics. This user can access the web console.\"\n    },\n    {\n        \"role\": \"security_admin_local\",\n        \"name\": \"Local User Security Admin\",\n        \"desc\": \"Can view all cluster statistics and manage local user roles, but not grant Full Admin or Security Admin roles to other users or alter their own role. This user can access the web console. This user cannot read data.\"\n    },\n    {\n        \"role\": \"security_admin_external\",\n        \"name\": \"External User Security Admin\",\n        \"desc\": \"Can view all cluster statistics and manage external user roles, but not grant Full Admin or Security Admin roles to other users or alter their own role. This user can access the web console. This user cannot read data.\"\n    },\n    {\n        \"role\": \"cluster_admin\",\n        \"name\": \"Cluster Admin\",\n        \"desc\": \"Can manage all cluster features except security. This user can access the web console. This user cannot read data.\"\n    },\n    {\n        \"role\": \"eventing_admin\",\n        \"name\": \"Eventing Full Admin\",\n        \"desc\": \"Can create/manage eventing functions. This user can access the web console\"\n    },\n    {\n        \"role\": \"backup_admin\",\n        \"name\": \"Backup Full Admin\",\n        \"desc\": \"Can perform backup related tasks. This user can access the web console\"\n    },\n    {\n        \"role\": \"scope_admin\",\n        \"name\": \"Manage Scopes\",\n        \"desc\": \"Can create/delete scopes and collections within a given bucket. This user cannot access the web console.\"\n    },\n    {\n        \"role\": \"bucket_full_access\",\n        \"name\": \"Application Access\",\n        \"desc\": \"Full access to bucket data. This user cannot access the web console and is intended only for application access. This user can read and write data except for the _system scope which can only be read.\"\n    },\n    {\n        \"role\": \"views_admin\",\n        \"name\": \"Views Admin\",\n        \"desc\": \"Can create and manage views of a given bucket. This user can access the web console. This user can read some data.\"\n    },\n    {\n        \"role\": \"views_reader\",\n        \"name\": \"Views Reader\",\n        \"desc\": \"Can read data from the views of a given bucket. This user cannot access the web console and is intended only for application access. This user can read some data.\"\n    },\n    {\n        \"role\": \"replication_admin\",\n        \"name\": \"XDCR Admin\",\n        \"desc\": \"Can administer XDCR features to create cluster references and replication streams out of this cluster. This user can access the web console. This user can read some data.\"\n    },\n    {\n        \"role\": \"data_reader\",\n        \"name\": \"Data Reader\",\n        \"desc\": \"Can read data from a given bucket, scope or collection. This user cannot access the web console and is intended only for application access. This user can read data, but cannot write it.\"\n    },\n    {\n        \"role\": \"data_writer\",\n        \"name\": \"Data Writer\",\n        \"desc\": \"Can write data to a given bucket, scope or collection. This user cannot access the web console and is intended only for application access. This user can write data, but cannot read it.\"\n    },\n    {\n        \"role\": \"data_dcp_reader\",\n        \"name\": \"Data DCP Reader\",\n        \"desc\": \"Can initiate DCP streams for a given bucket, scope or collection. This user cannot access the web console and is intended only for application access. This user can read data.\"\n    },\n    {\n        \"role\": \"data_backup\",\n        \"name\": \"Data Backup & Restore\",\n        \"desc\": \"Can backup and restore a given bucket's data. This user cannot access the web console and is intended only for application access. This user can read data.\"\n    },\n    {\n        \"role\": \"data_monitoring\",\n        \"name\": \"Data Monitor\",\n        \"desc\": \"Can read statistics for a given bucket, scope or collection. This user cannot access the web console and is intended only for application access. This user cannot read data.\"\n    },\n    {\n        \"role\": \"fts_admin\",\n        \"name\": \"Search Admin\",\n        \"desc\": \"Can administer all Full Text Search features. This user can access the web console. This user can read some data.\"\n    },\n    {\n        \"role\": \"fts_searcher\",\n        \"name\": \"Search Reader\",\n        \"desc\": \"Can query Full Text Search indexes for a given bucket, scope or collection. This user can access the web console. This user can read some data.\"\n    },\n    {\n        \"role\": \"query_select\",\n        \"name\": \"Query Select\",\n        \"desc\": \"Can execute a SELECT statement on a given bucket, scope or collection to retrieve data. This user can access the web console and can read data, but not write it.\"\n    },\n    {\n        \"role\": \"query_update\",\n        \"name\": \"Query Update\",\n        \"desc\": \"Can execute an UPDATE statement on a given bucket, scope or collection to update data. This user can access the web console and write data, but cannot read it.\"\n    },\n    {\n        \"role\": \"query_insert\",\n        \"name\": \"Query Insert\",\n        \"desc\": \"Can execute an INSERT statement on a given bucket, scope or collection to add data. This user can access the web console and insert data, but cannot read it.\"\n    },\n    {\n        \"role\": \"query_delete\",\n        \"name\": \"Query Delete\",\n        \"desc\": \"Can execute a DELETE statement on a given bucket, scope or collection to delete data. This user can access the web console, but cannot read data. This user can delete data.\"\n    },\n    {\n        \"role\": \"query_manage_index\",\n        \"name\": \"Query Manage Index\",\n        \"desc\": \"Can manage indexes for a given bucket, scope or collection. This user can access the web console, can read statistics for a given bucket, scope or collection. This user cannot read data.\"\n    },\n    {\n        \"role\": \"query_system_catalog\",\n        \"name\": \"Query System Catalog\",\n        \"desc\": \"Can look up system catalog information via N1QL. This user can access the web console, but cannot read user data.\"\n    },\n    {\n        \"role\": \"query_external_access\",\n        \"name\": \"Query CURL Access\",\n        \"desc\": \"Can execute the CURL statement from within N1QL. This user can access the web console, but cannot read data (within Couchbase).\"\n    },\n    {\n        \"role\": \"query_manage_global_functions\",\n        \"name\": \"Manage Global Functions\",\n        \"desc\": \"Can manage global n1ql functions\"\n    }\n]",
		},
		IsUser: true,
	},
	E_NO_SUCH_KEYSPACE: {
		Code:        5240,
		ErrorCode:   "E_NO_SUCH_KEYSPACE",
		Description: "Keyspace [keyspace] is not valid. (in context of a parameterized role request)",
		Causes: []string{
			"The keyspace attached to the role is not present,\nwe don't have the bucket-> 'trial'. But we are trying to grant query_select on it.\n\nGRANT  query_select ON `trial` TO jan;\n[\n  {\n    \"code\": 5240,\n    \"msg\": \"Keyspace default:travel-sample.inventory.trial is not valid.\"\n  }\n]\n\nsame applies to REVOKE as well.",
		},
		Actions: []string{
			"GET /pools/default/bucket endpoint will tell you about valid buckets in you cluster.",
		},
		IsUser: true,
	},
	E_NO_SUCH_SCOPE: {
		Code:        5241,
		ErrorCode:   "E_NO_SUCH_SCOPE",
		Description: "Scope [scope] is not valid. (in context of a parameterized role request)",
		Causes: []string{
			"scope_admin is the only role that requires scope parameterization.\n\nBut the scope path passed user doesn't exist. For example _myscope doesn't exist in the default bucket.\nGRANT scope_admin ON default:`default`._myscope TO jan;\n[\n  {\n    \"code\": 5241,\n    \"msg\": \"Scope default:default._myscope is not valid.\"\n  }\n]\n\nSame applies to REVOKE as well.",
		},
		Actions: []string{
			"GET http://127.0.0.1:9000/pools/default/buckets/{bucketname}/scopes will tell you all valid scopes in your bucket.",
		},
		IsUser: true,
	},
	E_NO_SUCH_BUCKET: {
		Code:        5242,
		ErrorCode:   "E_NO_SUCH_BUCKET",
		Description: "Bucket [bucket] is not valid.",
		Causes: []string{
			"parameterized role request with keyspace having only 2 parts (namespace.bucket)\n1) has _default scope and _default collection, role request is for that keyspace\n2) but if no _default collection, role request for entire bucket\n\nand the bucket information can't be got from the datastore.\nTypically would not occur.",
		},
		Actions: []string{},
	},
	E_ROLE_NOT_FOUND: {
		Code:        5250,
		ErrorCode:   "E_ROLE_NOT_FOUND",
		Description: "Role [role-requested] is not valid.",
		Causes: []string{
			"As a part of the logic for validating roles requested as a part of GRANT/REVOKE role statement. \nWe compare roles received with defined roles on the datastore https://docs.couchbase.com/server/current/rest-api/rbac.html#list-roles /settings/rbac/roles\nif requested role doesn't match any of the defined roles we error out with this error. \nfor eg: \nGRANT replication_supremo TO jan; \n[ { \n       \"code\": 5250, \n       \"msg\": \"Role replication_supremo is not valid.\", \n       \"query\": \"GRANT replication_supremo\\n TO jan;\" \n} ]",
		},
		Actions: []string{
			"use roles defined at GET /settings/rbac/roles endpoint.",
		},
		IsUser: true,
	},
	W_ROLE_ALREADY_PRESENT: {
		Code:        5260,
		ErrorCode:   "W_ROLE_ALREADY_PRESENT",
		Description: "User %s already has role [role]([bucket])",
		Causes: []string{
			"GET settings/rbac/users\nalready lists the requested role for a particular user.\n\nFor eg:\nGET settings/rbac/users\n[\n    {\n        \"id\": \"jan\",\n        \"domain\": \"local\",\n        \"roles\": [\n            {\n                \"role\": \"scope_admin\",\n                \"bucket_name\": \"default\",\n                \"scope_name\": \"_default\",\n                \"origins\": [\n                    {\n                        \"type\": \"user\"\n                    }\n                ]\n            },\n            {\n                \"role\": \"replication_admin\",\n                \"origins\": [\n                    {\n                        \"type\": \"user\"\n                    }\n                ]\n            },\n            {\n                \"role\": \"query_external_access\",\n                \"origins\": [\n                    {\n                        \"type\": \"user\"\n                    }\n                ]\n            },\n            {\n                \"role\": \"query_execute_global_functions\",\n                \"origins\": [\n                    {\n                        \"type\": \"user\"\n                    }\n                ]\n            }\n        ],\n        \"groups\": [\n        ],\n        \"external_groups\": [\n        ],\n        \"name\": \"\",\n        \"uuid\": \"f3f8e9ac-94be-4a0c-aa7c-02fac6c51721\",\n        \"password_change_date\": \"2023-11-22T15:13:03.000Z\"\n    }\n]\n\nGRANT query_execute_global_functions TO jan;\ncode        msg\n5260        \"User local:jan already has role query_execute_global_functions.\"\n\nSimilar for REVOKE statement.",
		},
		Actions: []string{
			"You can check assigned roles for a user from ui-> https://docs.couchbase.com/server/current/manage/manage-security/manage-users-and-roles.html#manage-users-with-the-ui \n\nor Rest-api: GET settings/rbac/users",
		},
		IsUser: true,
	},
	W_USER_WITH_NO_ROLES: {
		Code:        5280,
		ErrorCode:   "W_USER_WITH_NO_ROLES",
		Description: "User [user-name] has no roles. Connecting with this user may not be possible",
		Causes: []string{
			"REVOKE query has successfully matched roles requested to be revoked and those the user has. But after removing those roles this particular user has no roles hence we issue this warning\n\nfor eg: \nGET /settings/rbac/users\n{\n\"id\": \"tom\",\n\"domain\": \"local\",\n\"roles\":[\n{\n\"role\": \"query_execute_global_functions\",\n\"origins\":[{\"type\": \"user\" }]\n}\n],\n\"groups\":[],\n\"external_groups\":[],\n\"name\": \"tommy\",\n\"uuid\": \"add6b21e-87b7-4683-a9fd-b229097925b8\",\n\"password_change_date\": \"2023-12-01T14:25:02.000Z\"\n}\n\n> REVOKE query_execute_global_functions FROM tom;\n[\n  {\n    \"code\": 5280,\n    \"msg\": \"User local:tom has no roles. Connecting with this user may not be possible\"\n  }\n]",
		},
		Actions: []string{
			"No actions to take here error just signals that REVOKE has left user roleless.",
		},
		IsUser: true,
	},
	E_HASH_TABLE_PUT: {
		Code:        5300,
		ErrorCode:   "E_HASH_TABLE_PUT",
		Description: "",
		Causes:      []string{},
		Actions:     []string{},
	},
	E_HASH_TABLE_GET: {
		Code:        5310,
		ErrorCode:   "E_HASH_TABLE_GET",
		Description: "",
		Causes:      []string{},
		Actions:     []string{},
	},
	E_MERGE_MULTI_UPDATE: {
		Code:        5320,
		ErrorCode:   "E_MERGE_MULTI_UPDATE",
		Description: "Multiple UPDATE/DELETE of the same document (document key [key-passed]) in a MERGE statement",
		Causes: []string{
			"fail-safe code, ideally would never occur\n\nWHEN MATCHED\nmerge-update , merge-delete actions must not see the same document again.",
		},
		Actions: []string{},
		IsUser:  false,
	},
	E_MERGE_MULTI_INSERT: {
		Code:        5330,
		ErrorCode:   "E_MERGE_MULTI_INSERT",
		Description: "Multiple INSERT of the same document (document key [key-passed]) in a MERGE statement",
		Causes: []string{
			"fail-safe code for LOOKUP merge insert as doesn't expect key expression for insert, look at example 7: https://docs.couchbase.com/server/current/n1ql/n1ql-language-reference/merge.html#examples \n\nfor ANSI_MERGE insert , ensure key expression generates unique key for the send-insert operator.\nbad example:\nMERGE INTO route\nUSING airport\nON route.sourceairport = airport.faa\nWHEN NOT MATCHED THEN \n    INSERT (KEY \"p1\", VALUE { \"faa\": airport.faa });\n\n[\n  {\n    \"code\": 5330,\n    \"msg\": \"Multiple INSERT of the same document (document key 'p1') in a MERGE statement\"\n  }\n]",
		},
		Actions: []string{
			"Safe bet to avoid this is to have the KEY expression set as uuid() or something concatenated to uuid() to ensure unique key generation.",
		},
		IsUser: true,
	},
	E_WINDOW_EVALUATION: {
		Code:        5340,
		ErrorCode:   "E_WINDOW_EVALUATION",
		Description: "1) Error initial setup\n\n2) Error during evaluating duplicate oby value.\n\n3) Error evaluating Window partition value.\n\n4) Error evaluating Window function.",
		Causes: []string{
			"1) intial setup failed due to voilation in Window Frame Clause's Extents received:\n\nWindow frame extents that result in an explicit violation are:\n( ROWS | RANGE | GROUPS ) BETWEEN CURRENT ROW AND valexpr PRECEDING\n( ROWS | RANGE | GROUPS ) BETWEEN valexpr FOLLOWING AND valexpr PRECEDING\n( ROWS | RANGE | GROUPS ) BETWEEN valexpr FOLLOWING AND CURRENT ROW",
			"2) window terms orderby clause evaluation failed",
			"3) i) fail-safe code, attachment for partition by clause's filed is not set\n    ii) partition by clause's expression( maybe a field/path or any n1ql expression ) this expression's evaluation failed on an item.",
			"4) actual aggregate/window function evaluation on the item failed.",
		},
		Actions: []string{
			"Please ask on forums",
		},
		IsUser: true,
	},
	E_ADVISE_INDEX: {
		Code:        5350,
		ErrorCode:   "E_ADVISE_INDEX",
		Description: "AdviseIndex: Error marshaling JSON.",
		Causes: []string{
			"During execution of Advice operator, something wen wrong in marshalling the planOperator(Advise).",
		},
		Actions: []string{
			"Maybe a bug, contact support.",
		},
		IsUser: false,
	},
	E_ADVISE_INVALID_RESULTS: {
		Code:        5351,
		ErrorCode:   "E_ADVISE_INVALID_RESULTS",
		Description: "Invalid advise results",
		Causes: []string{
			"fail-safe code, to ensure task-entry has \"state\" field.\nRequired for processing the logic for purgeResults of a \"completed\"/\"cancelled\"/\"deleting\"",
		},
		Actions: []string{
			"Not a bug",
		},
		IsUser: false,
	},
	E_UPDATE_STATISTICS: {
		Code:        5360,
		ErrorCode:   "E_UPDATE_STATISTICS",
		Description: "UNUSED",
		Causes:      []string{},
		Actions:     []string{},
	},
	E_SUBQUERY_BUILD: {
		Code:        5370,
		ErrorCode:   "E_SUBQUERY_BUILD",
		Description: "Unable to run subquery",
		Causes: []string{
			"When trying to evaluate a subquery, we first check if we have the subqeryExecutionTree in saved, then we reopen it and start the execution for it.\nIf the execution tree is not found, we have to rebuild from the plan. If something goes wrong during the build we error out and wrap the causing error with this error.",
		},
		Actions: []string{
			"Contact support.",
		},
		IsUser: false,
	},
	E_INDEX_LEADING_KEY_MISSING_NOT_SUPPORTED: {
		Code:        5380,
		ErrorCode:   "E_INDEX_LEADING_KEY_MISSING_NOT_SUPPORTED",
		Description: "Indexing leading key MISSING values are not supported by indexer.",
		Causes: []string{
			"INCLUDE MISSING (lead-key-attribs https://docs.couchbase.com/server/current/n1ql/n1ql-language-reference/createindex.html#index-key-attrib ) is only supported for indexers that support index API 3, API 5.",
		},
		Actions: []string{
			"Upgrade index service to a version that supports https://docs.couchbase.com/server/current/n1ql/n1ql-language-reference/createindex.html#index-key-attrib ",
		},
		IsUser: true,
	},
	E_INDEX_NOT_IN_MEMORY: {
		Code:        5390,
		ErrorCode:   "E_INDEX_NOT_IN_MEMORY",
		Description: "UNUSED",
		Causes:      []string{},
		Actions:     []string{},
	},
	E_MISSING_SYSTEMCBO_STATS: {
		Code:        5400,
		ErrorCode:   "E_MISSING_SYSTEMCBO_STATS",
		Description: "UNUSED. N1QL_SYSTEM_BUCKET logic has been replaced.",
		Causes:      []string{},
		Actions:     []string{},
	},
	E_INVALID_INDEX_NAME: {
		Code:        5410,
		ErrorCode:   "E_INVALID_INDEX_NAME",
		Description: "index name([index_name_received]) must be a string",
		Causes: []string{
			"1) index names passed for build index , index name https://docs.couchbase.com/server/current/n1ql/n1ql-language-reference/build-index.html#index-name must be a string\nBUILD INDEX ON default(3);\n[\n  {\n    \"code\": 5410,\n    \"msg\": \"index name(3) must be a string\"\n  }\n]",
			"2) update statistics, a) for single index expects index name to be a string https://docs.couchbase.com/server/current/n1ql/n1ql-language-reference/statistics-index.html \n                                b) for multiple indexes, expects all index names to be a string https://docs.couchbase.com/server/current/n1ql/n1ql-language-reference/statistics-indexes.html",
		},
		Actions: []string{
			"1) use SELECT * FROM system:indexes WHERE state=\"deferred\"; to find out the indexes that are waiting for build to kick-off.",
			"2) SELECT * FROM system:indexes; to find out all existing indexes.",
		},
		IsUser: true,
	},
	E_INDEX_NOT_FOUND: {
		Code:        5411,
		ErrorCode:   "E_INDEX_NOT_FOUND",
		Description: "index [index-name] is not found",
		Causes: []string{
			"1) index name doesn't exist in the indexer that we are trying to build for\nBUILD INDEX ON default(temp);\n[\n  {\n    \"code\": 5411,\n    \"msg\": \"index temp is not found - cause: Index Not Found - cause: GSI index temp not found.\"\n  }\n]",
			"2) update statistics, a) for single index, index name doesn't exist https://docs.couchbase.com/server/current/n1ql/n1ql-language-reference/statistics-index.html \n                                b) for multiple indexes, one of the index names don't exist https://docs.couchbase.com/server/current/n1ql/n1ql-language-reference/statistics-indexes.html",
		},
		Actions: []string{
			"1) use SELECT * FROM system:indexes WHERE state=\"deferred\"; to find out the indexes that are waiting for build to kick-off.",
			"2) SELECT * FROM system:indexes; to find out all existing indexes.",
		},
		IsUser: true,
	},
	E_INDEX_UPD_STATS: {
		Code:        5415,
		ErrorCode:   "E_INDEX_UPD_STATS",
		Description: "Error with UPDATE STATISTICS for indexes ([index-names])",
		Causes: []string{
			"BUILD INDEX / CREATE INDEX / CREATE PRIMARY INDEX statements , schedule updatestats task for cbo\nThe logic for scheduling, 1) task-id generation failed, 2) actual task entry was created but couldn't get added to the schedule cache.",
		},
		Actions: []string{
			"Contact support",
		},
		IsUser: false,
	},
	E_TIME_PARSE: {
		Code:        5416,
		ErrorCode:   "E_TIME_PARSE",
		Description: "UNUSED",
		Causes:      []string{},
		Actions:     []string{},
	},
	E_JOIN_ON_PRIMARY_DOCS_EXCEEDED: {
		Code:        5420,
		ErrorCode:   "E_JOIN_ON_PRIMARY_DOCS_EXCEEDED",
		Description: "Inner of nested-loop join ([keyspace]) cannot have more than 1000 documents without appropriate secondary index",
		Causes: []string{
			"Before joins needed secondary index to be defined on the right-hand side for perfomance reasons, now inorder to allow new developers to play with join queries that use NestedLoop Join(non-equality join predicates) without secondary index (using primary or sequential-scan), but we set a max limit of 1000docs on the right side of the join for this case.\n\n\nRight of join:\ncbq> SELECT COUNT(1) FROM defaul1;\n{\n    \"requestID\": \"88ce984c-bc1d-4eb1-870a-b666268659cf\",\n    \"signature\": {\n        \"$1\": \"number\"\n    },\n    \"results\": [\n    {\n        \"$1\": 1001\n    }\n    ], \n\n\nSELECT d1.a as a1, d2.a as a2 FROM default d1 JOIN defaul1 d2 ON d1.a>d2.a;\n...\n...\n...\n     \"a1\": 93,\n        \"a2\": 76\n    },\n    {\n        \"a1\": 93,\n        \"a2\": 61\n    },\n    {\n        \"a1\": 93,\n        \"a2\": 30\n    },\n    {\n        \"a1\": 93,\n        \"a2\": 39\n    }\n    ],\n    \"errors\": [\n        {\n            \"code\": 5420,\n            \"msg\": \"Inner of nested-loop join (d2) cannot have more than 1000 documents without appropriate secondary index\",\n            \"reason\": {\n                \"keyspace_alias\": \"d2\",\n                \"limit\": 1000\n            }\n        }\n    ],",
		},
		Actions: []string{
			"CREATE INDEX for the right hand side-> for the example CREATE INDEX idx ON defaul1(a);",
		},
		IsUser: true,
	},
	E_MEMORY_QUOTA_EXCEEDED: {
		Code:        5500,
		ErrorCode:   "E_MEMORY_QUOTA_EXCEEDED",
		Description: "Request has exceeded memory quota.",
		Causes: []string{
			"Enforced at request level by setting memory_quota request parameter to a non-zero value https://docs.couchbase.com/server/current/n1ql/n1ql-rest-api/index.html \n(This parameter enforces a ceiling on the memory used for the tracked documents required for processing a request. It does not take into account any other memory that might be used to process a request, such as the stack, the operators, or some intermediate values.)\n\nThe logic is that when an  item is processed by an operator in the execution tree we account for memory quota on the item's size, that is item's size is not greater than the memory quota.\nThe size over here for example if a number-> 8bytes , for a string-> 1byte per character.\n\nIn conclusion the item with new attachments and other information we pass in the execution phase has exceeded the set memory quota.",
		},
		Actions: []string{
			"Increase the request-level memory quota to allow the execution to continue",
		},
		IsUser: true,
	},
	E_NIL_EVALUATE_PARAM: {
		Code:        5501,
		ErrorCode:   "E_NIL_EVALUATE_PARAM",
		Description: "nil [param] parameter for evaluation",
		Causes: []string{
			"1) all aggregate functions https://docs.couchbase.com/server/current/n1ql/n1ql-language-reference/aggregatefun.html , windowfunctions https://docs.couchbase.com/server/current/n1ql/n1ql-language-reference/windowfun.html \n    fail-safe code to ensure annotedvalue with the aggregate computed by the group operators(initial , intermediate and final) has finally been passed to the actual expression call (either in PROJECTION/ LET / WHERE clause), so the expression can evaluate to the computed aggregate value by the group logic. This happens regardless of if we have GROUP BY clause in query or not, \ni.e \nEXPLAIN SELECT sourceairport,COUNTN(stops) FROM `travel-sample`.inventory.route GROUP BY sourceairport ;\n {\n          \"#operator\": \"Parallel\",\n          \"~child\": {\n            \"#operator\": \"Sequence\",\n            \"~children\": [\n              {\n                \"#operator\": \"InitialGroup\",\n                \"aggregates\": [\n                  \"countn((`route`.`stops`))\"\n                ],\n                \"flags\": 4,\n                \"group_keys\": [\n                  \"(`route`.`sourceairport`)\"\n                ],\n                \"optimizer_estimates\": {\n                  \"cardinality\": 1367.999967695319,\n                  \"cost\": 38744.559677066005,\n                  \"fr_cost\": 38744.559677066005,\n                  \"size\": 569\n                }\n              }\n            ]\n          }\n        },\n        {\n          \"#operator\": \"IntermediateGroup\",\n          \"aggregates\": [\n            \"countn((`route`.`stops`))\"\n          ],\n          \"flags\": 4,\n          \"group_keys\": [\n            \"(`route`.`sourceairport`)\"\n          ],\n          \"optimizer_estimates\": {\n            \"cardinality\": 1367.999967695319,\n            \"cost\": 38895.03967351249,\n            \"fr_cost\": 38895.03967351249,\n            \"size\": 569\n          }\n        },\n        {\n          \"#operator\": \"FinalGroup\",\n          \"aggregates\": [\n            \"countn((`route`.`stops`))\"\n          ],\n          \"flags\": 4,\n          \"group_keys\": [\n            \"(`route`.`sourceairport`)\"\n          ],\n          \"optimizer_estimates\": {\n            \"cardinality\": 1367.999967695319,\n            \"cost\": 38922.39967286639,\n            \"fr_cost\": 38922.39967286639,\n            \"size\": 569\n          }\n        },\n        {\n          \"#operator\": \"Parallel\",\n          \"~child\": {\n            \"#operator\": \"Sequence\",\n            \"~children\": [\n              {\n                \"#operator\": \"InitialProject\",\n                \"discard_original\": true,\n                \"optimizer_estimates\": {\n                  \"cardinality\": 1367.999967695319,\n                  \"cost\": 38987.66345166317,\n                  \"fr_cost\": 38922.44738030816,\n                  \"size\": 569\n                },\n                \"preserve_order\": true,\n                \"result_terms\": [\n                  {\n                    \"expr\": \"(`route`.`sourceairport`)\"\n                  },\n                  {\n                    \"expr\": \"countn((`route`.`stops`))\"\n                  }\n                ]\n              }\n            ]\n          }\n\nEXPLAIN SELECT COUNTN(stops) FROM `travel-sample`.inventory.route ;\n{\n          \"#operator\": \"Parallel\",\n          \"~child\": {\n            \"#operator\": \"Sequence\",\n            \"~children\": [\n              {\n                \"#operator\": \"InitialGroup\",\n                \"aggregates\": [\n                  \"countn((`route`.`stops`))\"\n                ],\n                \"flags\": 4,\n                \"group_keys\": [],\n                \"optimizer_estimates\": {\n                  \"cardinality\": 1,\n                  \"cost\": 33013.941771953156,\n                  \"fr_cost\": 33013.941771953156,\n                  \"size\": 569\n                }\n              }\n            ]\n          }\n        },\n        {\n          \"#operator\": \"IntermediateGroup\",\n          \"aggregates\": [\n            \"countn((`route`.`stops`))\"\n          ],\n          \"flags\": 4,\n          \"group_keys\": [],\n          \"optimizer_estimates\": {\n            \"cardinality\": 1,\n            \"cost\": 33013.95177195316,\n            \"fr_cost\": 33013.95177195316,\n            \"size\": 569\n          }\n        },\n        {\n          \"#operator\": \"FinalGroup\",\n          \"aggregates\": [\n            \"countn((`route`.`stops`))\"\n          ],\n          \"flags\": 4,\n          \"group_keys\": [],\n          \"optimizer_estimates\": {\n            \"cardinality\": 1,\n            \"cost\": 33013.96177195316,\n            \"fr_cost\": 33013.96177195316,\n            \"size\": 569\n          }\n        },\n        {\n          \"#operator\": \"Parallel\",\n          \"~child\": {\n            \"#operator\": \"Sequence\",\n            \"~children\": [\n              {\n                \"#operator\": \"InitialProject\",\n                \"discard_original\": true,\n                \"optimizer_estimates\": {\n                  \"cardinality\": 1,\n                  \"cost\": 33013.985625674046,\n                  \"fr_cost\": 33013.985625674046,\n                  \"size\": 569\n                },\n                \"preserve_order\": true,\n                \"result_terms\": [\n                  {\n                    \"expr\": \"countn((`route`.`stops`))\"\n                  }\n                ]\n              }\n            ]\n          }",
			"2) fail-safe code-> for covered-expression , after undergoing covering transformation(expression part of PROJECTION/ WHERE/ LET Clause) during planning, failsafe code to ensure the scan operator has added attachment for covers to be used in the evaluation of the covered expression.",
			"3) fail-safe-code-> for expressions that are a call to NowMillis, NowTZ, NowStr, NowUtc builtin-functions, rely on the request context to know the current time when request was issued.",
			"4) fail-safe-code-> for expressions that refer to positional/named parameter, rely on the request context to hold the parameters got in as a part of the request context.",
			"5) fail-safe-code-> for expressions that are a call to ds_version function, rely on request context, to get the information from the datastore.",
			"6) fail-safe-code-> for call to advisor() , to mark the context as advisor context to account to advisor sessions, etc.",
			"7) fail-safe-code-> for identifier expression, to ensure the item passed from scan operator has a field with same name as the identifier so we can evaluate the identifier to that value.",
			"8) ail-safe-code-> for expression that are a call to object_remove builtin function, expects the have a non-nil context to parse the string passed to identifier/fieldname expression that has to be deleted in the object passed.",
		},
		Actions: []string{
			"Contact support.",
		},
		IsUser: false,
	},
	E_BUCKET_ACTION: {
		Code:        5502,
		ErrorCode:   "E_BUCKET_ACTION",
		Description: "Unable to complete action after say N attempts",
		Causes: []string{
			"this is just a wrapper error for cases where DML statements fail, what is happening is that command/op is not completed due to hitting maxretries or a connection timeout",
		},
		Actions: []string{
			"try again",
		},
		IsUser: false,
	},
	W_MISSING_KEY: {
		Code:        5503,
		ErrorCode:   "W_MISSING_KEY",
		Description: "Key(s) in USE KEYS hint not found",
		Causes: []string{
			"When user has passed VALIDATE keyword as a part of USE KEYS/ ON KEY/ ON KEYS  clause\nservice returns the missing keys as a warning in the response to the request.\n\nPossible usage:\n1) Lookup Joins https://docs.couchbase.com/server/current/n1ql/n1ql-language-reference/join.html#lookup-join-clause , ON KEYS VALIDATE\n2) USE keys Clause https://docs.couchbase.com/server/current/n1ql/n1ql-language-reference/hints.html#use-keys-clause , USE KEYS VALIDATE\n3) Index Nest: https://docs.couchbase.com/server/current/n1ql/n1ql-language-reference/nest.html#section_rgr_rnx_1db , ON KEY VALIDATE\n4) Lookup Nest: https://docs.couchbase.com/server/current/n1ql/n1ql-language-reference/nest.html#nest , ON KEY VALIDATE\n5) Index Join: https://docs.couchbase.com/server/current/n1ql/n1ql-language-reference/join.html#index-join-clause  ,ON KEY VALIDATE\n5) Delete: delete hint: https://docs.couchbase.com/server/current/n1ql/n1ql-language-reference/delete.html#delete-hint  , USE KEYS VALIDATE",
		},
		Actions: []string{
			"Not an error, just a warning so user can know which documents didn't get included as a part of their query.",
		},
		IsUser: false,
	},
	E_NODE_QUOTA_EXCEEDED: {
		Code:        5600,
		ErrorCode:   "E_NODE_QUOTA_EXCEEDED",
		Description: "Query node has run out of memory",
		Causes: []string{
			"node-level memory quota: set by \"node-quota\" , and \"node-quota-val-percent\" setting",
		},
		Actions: []string{},
	},
	E_TENANT_QUOTA_EXCEEDED: {
		Code:        5601,
		ErrorCode:   "E_TENANT_QUOTA_EXCEEDED",
		Description: "UNUSED.",
		Causes:      []string{},
		Actions:     []string{},
	},
	E_VALUE_RECONSTRUCT: {
		Code:        5700,
		ErrorCode:   "E_VALUE_RECONSTRUCT",
		Description: "Failed to reconstruct value",
		Causes: []string{
			"Spill to disk feature is enabled\n\nwhen no suitable index for ORDER BY/ GROUP BY pushdown\nfor memory-concerns we resort to spill to disk ( SpillingArray (for order by) / SpillingMap (for group by)).\n\nreading next value failed as bytes on the wire didn't match any of the spill_type(something like book-keeping), hence we error out.",
		},
		Actions: []string{
			"Contact support",
		},
	},
	E_VALUE_INVALID: {
		Code:        5701,
		ErrorCode:   "E_VALUE_INVALID",
		Description: "Invalid reconstructed value",
		Causes: []string{
			"Spill to disk feature is enabled\n\nwhen no suitable index for ORDER BY/ GROUP BY pushdown\nfor memory-concerns we resort to spill to disk ( SpillingArray (for order by) / SpillingMap (for group by)).\n\nwhen reading spillvalues from spillfile , couldn't decode the spillvalue to a recognized annotated value to send to the next execution operator.",
		},
		Actions: []string{
			"Contact support",
		},
	},
	E_VALUE_SPILL_CREATE: {
		Code:        5702,
		ErrorCode:   "E_VALUE_SPILL_CREATE",
		Description: "Failed to create spill file",
		Causes: []string{
			"Spill to disk feature is enabled\n\nwhen no suitable index for ORDER BY/ GROUP BY pushdown\nfor memory-concerns we resort to spill to disk ( SpillingArray (for order by) / SpillingMap (for group by)).\n\nThis error is raised when the something goes wrong in call to create a tmp file to spill( stored in /ns_server/tmp ) in the documents appended soo far.",
		},
		Actions: []string{
			"Contact support",
		},
		IsUser: false,
	},
	E_VALUE_SPILL_READ: {
		Code:        5703,
		ErrorCode:   "E_VALUE_SPILL_READ",
		Description: "Failed to read from spill file",
		Causes: []string{
			"Spill to disk feature is enabled\n\nwhen no suitable index for ORDER BY/ GROUP BY pushdown\nwith regards memory-concerns we resort to spill to disk ( SpillingArray (for order by) / SpillingMap (for group by)).\n\nThis error is raised post the spill and sort , when we read\ni) rewind each file: seek back to the start of each spillfile\nii) read next sortedvalue(SPILL_VALUE) from the spillheap(incase of OrderBy), internally reads a value from the current spillfile, but the format received from the wire doesn't match the values we recognise\niii) similarly read next unsortedvalue",
		},
		Actions: []string{
			"Contact support",
		},
		IsUser: false,
	},
	E_VALUE_SPILL_WRITE: {
		Code:        5704,
		ErrorCode:   "E_VALUE_SPILL_WRITE",
		Description: "Failed to write to spill file",
		Causes: []string{
			"Spill to disk feature is enabled\n\nwhen no suitable index for ORDER BY/ GROUP BY pushdown\nwith regards memory-concerns we resort to spill to disk ( SpillingArray (for order by) / SpillingMap (for group by)).\n\nThis error is raised when something goes wrong in io (write the encoded spillvalue( SPILL_TYPE+SPILL_VAL from actual value)) to the spill file.",
		},
		Actions: []string{
			"Contact support",
		},
		IsUser: false,
	},
	E_VALUE_SPILL_SIZE: {
		Code:        5705,
		ErrorCode:   "E_VALUE_SPILL_SIZE",
		Description: "Failed to determine spill file size",
		Causes: []string{
			"Spill to disk feature is enabled\n\nwhen no suitable index for ORDER BY/ GROUP BY pushdown\nwith regards memory-concerns we resort to spill to disk ( SpillingArray (for order by) / SpillingMap (for group by)).\n\nPost the write to the spill file, we seek to the end to account for the file size. If something goes wrong here this error is raised.",
		},
		Actions: []string{
			"Contact support",
		},
		IsUser: false,
	},
	E_VALUE_SPILL_SEEK: {
		Code:        5706,
		ErrorCode:   "E_VALUE_SPILL_SEEK",
		Description: "UNUSED.",
		Causes:      []string{},
		Actions:     []string{},
	},
	E_SCHEDULER: {
		Code:        6001,
		ErrorCode:   "E_SCHEDULER",
		Description: "The scheduler encountered an error in generating uuid for the task entry",
		Causes: []string{
			"1) user has issued : CREATE INDEX / BUILD INDEX / CREATE PRIMARY INDEX , updatestats task is scheduled by the service for the index being created",
			"2) user has started an advisor session, with start action\n\nlogic to generate uuid has returned an error.",
		},
		Actions: []string{
			"ideally would never occur",
		},
		IsUser: true,
	},
	E_DUPLICATE_TASK: {
		Code:        6002,
		ErrorCode:   "E_DUPLICATE_TASK",
		Description: "Task already exists [task_id]",
		Causes: []string{
			"1) user has issued : CREATE INDEX / BUILD INDEX /CREATE PRIMARY INDEX, updatestats task is scheduled by the service for the index being created.",
			"2) user has started an advisor session, with start action\n\nlogic for adding a task entry to the scheduled task cache, fails as entry with same taskid is already present.",
		},
		Actions: []string{
			"ideally would never occur as we clean the scheduled task when a task entry's state is completed/cancelled. But if this happens can be signs of a new timing issue and we have a bug on our hands. Contact support.",
		},
		IsUser: true,
	},
	E_TASK_RUNNING: {
		Code:        6003,
		ErrorCode:   "E_TASK_RUNNING",
		Description: "Task [id] is currently executing and cannot be deleted",
		Causes: []string{
			"user has just created \nfor eg:\nGET /admin/tasks_cache HTTP/1.1\nAuthorization: Basic QWRtaW5pc3RyYXRvcjpwYXNzd29yZA==\nHost: 127.0.0.1:9499\n\nHTTP/1.1 200 OK\nContent-Type: application/json\nDate: Fri, 24 Nov 2023 06:09:03 GMT\nContent-Length: 345\n[{\"class\":\"update_statistics\",\"delay\":\"1s\",\"description\":\"default:default._default._default(defidx2)\",\"id\":\"136eedf0-ac5b-5980-a0fb-08ffa35b3c34\",\"name\":\"2465ca51-5d7d-4ef3-9dfa-a37ed7807a0e\",\"queryContext\":\"\",\"startTime\":\"2023-11-24T11:38:30.026+05:30\",\"state\":\"running\",\"subClass\":\"create_index\",\"submitTime\":\"2023-11-24T11:38:29.026+05:30\"}]\n\nand tried to delete the task entry for update statistics using 1) DELETE /admin/tasks_cache/{id} , 2) DELETE * FROM system:tasks_cache;\nbut the task is under running state, hence this error is raised",
		},
		Actions: []string{
			"GET /admin/tasks_cache/{id}\nand check if the state field is \"completed\", \n\nonly then can the user issue a DELETE /admin/task_cache/{id} or DELETE statment on system:task_cache keyspace",
		},
	},
	E_TASK_NOT_FOUND: {
		Code:        6004,
		ErrorCode:   "E_TASK_NOT_FOUND",
		Description: "the task [id] was not found - IMPOSSIBLE TO REPRODUCE",
		Causes:      []string{},
		Actions:     []string{},
	},
	E_INFER_INVALID_OPTION: {
		Code:        7000,
		ErrorCode:   "E_INFER_INVALID_OPTION",
		Description: "Invalid WITH clause usage in INFER statement",
		Causes: []string{
			"WITH CLAUSE expects:-> options to be passed as an object.",
		},
		Actions: []string{
			" something like this https://docs.couchbase.com/server/current/n1ql/n1ql-language-reference/infer.html#examples   ",
		},
		IsUser: true,
	},
	E_INFER_OPTION_MUST_BE_NUMERIC: {
		Code:        7001,
		ErrorCode:   "E_INFER_OPTION_MUST_BE_NUMERIC",
		Description: "passed option expects numeric value",
		Causes: []string{
			"all infer options are expected to be Numeric, except flag which can be numeric/string/array",
		},
		Actions: []string{
			"reframe your options from here https://docs.couchbase.com/server/current/n1ql/n1ql-language-reference/infer.html#infer-parameters ",
		},
		IsUser: true,
	},
	E_INFER_READING_NUMBER: {
		Code:        7002,
		ErrorCode:   "E_INFER_READING_NUMBER",
		Description: "particular invalid format for flags argument in INFER keyspace statement",
		Causes: []string{
			"This error is raised when user passes a string as input to flags to set appropriate bits in the flags option.",
		},
		Actions: []string{
			"user is expected to pass a string whose value is between \"1\" and \"4294967296\" which is later parsed to set the appropriate flag bits",
		},
		IsUser: true,
	},
	E_INFER_NO_KEYSPACE_DOCUMENTS: {
		Code:        7003,
		ErrorCode:   "E_INFER_NO_KEYSPACE_DOCUMENTS",
		Description: "UNUSED",
		Causes:      []string{},
		Actions:     []string{},
	},
	E_INFER_CREATE_RETRIEVER: {
		Code:        7004,
		ErrorCode:   "E_INFER_CREATE_RETRIEVER",
		Description: "Error creating document retriever.",
		Causes: []string{
			"user has used INFER EXPRESSION query,  the process of setting up a DocumentRetriver to start describing the expression passed: 1. Context passed is not an expression context, 2. user has passed subquery expression but failed to get execution handle to allow describing the subquery results, 3. error evaluating expression passed by user.",
		},
		Actions: []string{
			"for case of 2. check if subquery passed evaluates by running the SELECT statement directly, 3. check expression passed evaluates directly using SELECT {expression}",
		},
		IsUser: true,
	},
	E_INFER_NO_RANDOM_ENTRY: {
		Code:        7005,
		ErrorCode:   "E_INFER_NO_RANDOM_ENTRY",
		Description: "Keyspace does not support random document retrieval.",
		Causes: []string{
			"error is raised if no way to do randomScan, no sequential scan, no indexes -> only way to build the retriever is with random_entry. user has not passed \"no_random_entry\" flag, datastore for the keyspace fails to perform GET_RANDOM_KEY operation hence failing to get a random entry.",
		},
		Actions: []string{
			"the GetRandomDoc request may have failed due to timeout , or keyspace doesn't support PrimaryIndex3 (for eg: system:indexes and all other keyspaces under systems namespace)",
		},
		IsUser: false,
	},
	E_INFER_NO_RANDOM_DOCS: {
		Code:        7006,
		ErrorCode:   "E_INFER_NO_RANDOM_DOCS",
		Description: "SAME AS E_INFER_NO_RANDOM_ENTRY",
		Causes:      []string{},
		Actions:     []string{},
		IsUser:      false,
	},
	E_INFER_MISSING_CONTEXT: {
		Code:        7007,
		ErrorCode:   "E_INFER_MISSING_CONTEXT",
		Description: "Missing expression context.",
		Causes: []string{
			"INFER {expression}, expects expression context.",
		},
		Actions: []string{},
		IsUser:  false,
	},
	E_INFER_EXPRESSION_EVAL: {
		Code:        7008,
		ErrorCode:   "E_INFER_EXPRESSION_EVAL",
		Description: "Expression evaluation failed.",
		Causes: []string{
			"this is a wrapper error that signals that users expression couldn't be processed to start the Describe phase. 1. Subquery expression cannot be opened to get the execution handle, 2. expression cannot be evaluated",
		},
		Actions: []string{
			"user needs to rethink the subquery or expression he is using, try running the expression directly as SELECT {expr} to debug where things are going wrong step-by-step.",
		},
		IsUser: true,
	},
	E_INFER_KEYSPACE_ERROR: {
		Code:        7009,
		ErrorCode:   "E_INFER_KEYSPACE_ERROR",
		Description: "Keyspace error.",
		Causes: []string{
			"While building the Document Retriever, we try to fetch the document Count. The gomemcached.STAT op fails and hence we error out from building the document retriever",
		},
		Actions: []string{},
		IsUser:  false,
	},
	E_INFER_NO_SUITABLE_PRIMARY_INDEX: {
		Code:        7010,
		ErrorCode:   "E_INFER_NO_SUITABLE_PRIMARY_INDEX",
		Description: "No suitable primary index found.",
		Causes: []string{
			"error is raised if no way to get randomEntry, no way to do randomScan, no sequential scan -> only way to build the retriever is with primary index but it isn't built.",
		},
		Actions: []string{
			"CREATE PRIMARY INDEX ON {keyspace}",
		},
		IsUser: true,
	},
	E_INFER_NO_SUITABLE_SECONDARY_INDEX: {
		Code:        7011,
		ErrorCode:   "E_INFER_NO_SUITABLE_SECONDARY_INDEX",
		Description: "No suitable secondary index found.",
		Causes: []string{
			"error is raised when no randomEntry, no randomScan, no sequential scan, no primary indexes and user hasn't passed \"no_secondary_index\" flag. But no secondary index is available then this error is raised",
		},
		Actions: []string{
			"CREATE INDEX ON {keyspace}([indexkeys])",
		},
		IsUser: true,
	},
	W_INFER_TIMEOUT: {
		Code:        7012,
		ErrorCode:   "W_INFER_TIMEOUT",
		Description: "Stopped after exceeding infer_timeout. Schema may be incomplete.",
		Causes: []string{
			"Stopped after exceeding infer_timeout set by user as a part of options.  Schema may be incomplete. NOTE: default is 60seconds(Context Deadline)",
		},
		Actions: []string{
			"If describing your keyspace takes more time than default time. Retry the INFER statement with {\"infer_timeout\": [something greater than 60sec]}",
		},
		IsUser: true,
	},
	W_INFER_SIZE_LIMIT: {
		Code:        7013,
		ErrorCode:   "W_INFER_SIZE_LIMIT",
		Description: "Stopped after exceeding max_schema_MB. Schema may be incomplete.",
		Causes: []string{
			"exceeded schema size of \"max_schema_MB\" option, hence finishing the inferencing at this point. NOTE: default is 10MB.",
		},
		Actions: []string{
			"If your keyspace takes more than 10MB, Retry the INFER statement with {\"max_schema_MB\":[something greater than 10]}",
		},
		IsUser: true,
	},
	E_INFER_NO_DOCUMENTS: {
		Code:        7014,
		ErrorCode:   "E_INFER_NO_DOCUMENTS",
		Description: "No documents found, unable to infer schema.",
		Causes: []string{
			"The keyspace you are trying to infer schema on has no documents",
		},
		Actions: []string{
			"Insert documents using cbimport https://docs.couchbase.com/server/current/tools/cbimport.html or INSERT statement or SDK https://docs.couchbase.com/go-sdk/current/howtos/kv-operations.html ",
		},
		IsUser: true,
	},
	E_INFER_CONNECT: {
		Code:        7015,
		ErrorCode:   "E_INFER_CONNECT",
		Description: "FOR INFERENCER TOOL NOT cbq-engine",
		Causes:      []string{},
		Actions:     []string{},
	},
	E_INFER_GET_POOL: {
		Code:        7016,
		ErrorCode:   "E_INFER_GET_POOL",
		Description: "FOR INFERENCER TOOL NOT cbq-engine",
		Causes:      []string{},
		Actions:     []string{},
	},
	E_INFER_GET_BUCKET: {
		Code:        7017,
		ErrorCode:   "E_INFER_GET_BUCKET",
		Description: "FOR INFERENCER TOOL NOT cbq-engine",
		Causes:      []string{},
		Actions:     []string{},
	},
	E_INFER_GET_RANDOM: {
		Code:        7019,
		ErrorCode:   "E_INFER_GET_RANDOM",
		Description: "Failed to get random document.",
		Causes: []string{
			"Infer Statement is using Random Entry retriver on the keyspace. This requires logic of performing getRandomDoc on datastore using GET_RANDOM_KEY op",
		},
		Actions: []string{
			"Failed KV op, try again after sometime ",
		},
		IsUser: false,
	},
	E_INFER_NO_RANDOM_SCAN: {
		Code:        7020,
		ErrorCode:   "E_INFER_NO_RANDOM_SCAN",
		Description: "Keyspace does not support random key scans",
		Causes: []string{
			"error is raised if no way to get randomEntry, no sequential scan, no indexes -> only way to build the retriever is with randomScan. user has not passed \"no_random_scan\" flag, something went wrong when doing sequentialScan( KV-Range Scan)",
		},
		Actions: []string{},
		IsUser:  false,
	},
	E_INFER_NO_SEQUENTIAL_SCAN: {
		Code:        7021,
		ErrorCode:   "E_INFER_NO_SEQUENTIAL_SCAN",
		Description: "Sequential scan not available.",
		Causes: []string{
			"error is raised when no randomEntry, no randomScan, no primary indexes or secondary indexes. \"full_scan\" flag is on. But seqscan is disabled or not available for the keyspace being used",
		},
		Actions: []string{
			"Turn of n1ql-feat-ctrl flag and turn on sequential scan https://docs.couchbase.com/server/current/settings/query-settings.html#n1ql-feat-ctrl , NOTE: not recommended in production",
		},
		IsUser: true,
	},
	E_INFER_NO_RETRIEVERS: {
		Code:        7022,
		ErrorCode:   "E_INFER_NO_RETRIEVERS",
		Description: "No document retrievers available.",
		Causes: []string{
			"Wrapper error for scenarios of E_INFER_NO_SEQUENTIAL_SCAN, E_INFER_NO_SUITABLE_PRIMARY_INDEX, E_INFER_NO_RANDOM_DOCS, E_INFER_NO_RANDOM_SCAN, E_INFER_NO_RANDOM_ENTRY",
		},
		Actions: []string{
			"Actions follow the actions recomended for the listed errors being wrapped",
		},
		IsUser: true,
	},
	E_INFER_OPTIONS: {
		Code:        7023,
		ErrorCode:   "E_INFER_OPTIONS",
		Description: "UNUSED-> for INFER keyspace/expression/tool all pass default options -> dead code https://github.com/couchbase/query/blob/master/inferencer/describe_keyspace.go#L48 ",
		Causes:      []string{},
		Actions:     []string{},
	},
	E_INFER_NEXT_DOCUMENT: {
		Code:        7024,
		ErrorCode:   "E_INFER_NEXT_DOCUMENT",
		Description: "NextDocument failed",
		Causes: []string{
			"User has run INFER {subquery expression}. But execution handle created on subquery for Describing purpose returns error on getNextDoc().",
		},
		Actions: []string{},
		IsUser:  false,
	},
	E_MIGRATION: {
		Code:        7200,
		ErrorCode:   "E_MIGRATION",
		Description: "Error occurred during [UDF/CBO_Stat] migration",
		Causes: []string{
			"1) You have upgraded a node(with query-service) from 7.2 to 7.6, \nThis triggers Migration of UDF and CBOStats.",
			"2) UDF Migration\n7.2: scope functions are moved from MetaKV to system scope(_system) for the bucket.\nreason: to reduce the load on KV\n\nMigration is done by all query nodes, and synchronization is done document-level atomicity(i.e for UDF the function entry). \n\nCertain errors are raised as a part of the Migration logic( that is other than the logic needed to maintain consensus among the query nodes( retry logic, etc)\n1) error in parsing function name-> parts (namespace:bucket.scope.collection) to create a system_entry\n2) error out in writing the body to system storage\n3) error out when deleting the old metaKv entry\n4) error out when checking if migration is completed but entries still exist in metastorage.",
			"3) CBOStats migration \nCBOStats are moved from N1QL_SYSTEM_BUCKET to system storage.",
		},
		Actions: []string{
			"The easy way out is to just recreate your library and create the function refresh.",
		},
		IsUser: false,
	},
	E_MIGRATION_INTERNAL: {
		Code:        7201,
		ErrorCode:   "E_MIGRATION_INTERNAL",
		Description: "Unexpected error occurred during [UDF/CBO] migration: Unexpected error the datastore is not available",
		Causes: []string{
			"During Migration retry while waiting only the system scope to be created, but something went wrong and we don't have an handle on the datastore",
		},
		Actions: []string{
			"Contact support",
		},
		IsUser: false,
	},
	E_DATASTORE_AUTHORIZATION: {
		Code:        10000,
		ErrorCode:   "E_DATASTORE_AUTHORIZATION",
		Description: "Unable to authorize user.",
		Causes: []string{
			"for all serverless, free-tier or on-prem, when user credentials don't match the required privileges on datastore this error is thrown",
		},
		Actions: []string{
			"admin has to grant user required RBAC using security tab in web console or follow https://docs.couchbase.com/server/current/rest-api/rbac.html",
		},
		IsUser: true,
	},
	E_FTS_MISSING_PORT_ERR: {
		Code:        10003,
		ErrorCode:   "E_FTS_MISSING_PORT_ERR",
		Description: "UNUSED.",
		Causes:      []string{},
		Actions:     []string{},
	},
	E_NODE_INFO_ACCESS_ERR: {
		Code:        10004,
		ErrorCode:   "E_NODE_INFO_ACCESS_ERR",
		Description: "UNUSED.",
		Causes:      []string{},
		Actions:     []string{},
	},
	E_NODE_SERVICE_ERR: {
		Code:        10005,
		ErrorCode:   "E_NODE_SERVICE_ERR",
		Description: "UNUSED.",
		Causes:      []string{},
		Actions:     []string{},
	},
	E_FUNCTIONS_NOT_SUPPORTED: {
		Code:        10100,
		ErrorCode:   "E_FUNCTIONS_NOT_SUPPORTED",
		Description: "Functions of type Javascript are only supported in Enterprise Edition.",
		Causes: []string{
			"Disallow javascript user defined functions in community edition",
		},
		Actions: []string{
			"switch to enterprise https://www.couchbase.com/downloads/?family=couchbase-server ",
		},
		IsUser: true,
	},
	E_MISSING_FUNCTION: {
		Code:        10101,
		ErrorCode:   "E_MISSING_FUNCTION",
		Description: "Function [func-name] not found",
		Causes: []string{
			"1) DROP FUNCTION [func_name]; attempt to drop udf fails as function was never created.",
		},
		Actions: []string{
			"run SELECT * FROM system:functions;  to view available functions. You can add these as expressions in your query.",
		},
		IsUser: true,
	},
	E_DUPLICATE_FUNCTION: {
		Code:        10102,
		ErrorCode:   "E_DUPLICATE_FUNCTION",
		Description: "Function [func-name] already exists",
		Causes: []string{
			"CREATE FUNCTION [func_name(arg_list) LANGUAGE INLINE AS [definition];  fails as func_name already exists( systemEntry in _system scope already exits)",
		},
		Actions: []string{
			"if you are trying to redefine the function, add REPLACE Clause in query , something like: CREATE OR REPLACE [func_name] LANGUAGE INLINE AS [definition]",
		},
		IsUser: true,
	},
	E_INTERNAL_FUNCTION: {
		Code:        10103,
		ErrorCode:   "E_INTERNAL_FUNCTION",
		Description: "operation on function encountered unexpected error ",
		Causes: []string{
			"the unexpected errors may come from 1) when calling a scope function, there is a context switch to execute the function for eg : SELECT default._default.add3(1); here we switch from current context to default._default but for some unknown reason this fails. 2) When ready function body either from cache or systementry(storage) the retrieved body is not of the expected type( say function is inline but body is from javascript library), 3) error when formalizing variable names in INLINE function's expression(i.e the body) 4)",
		},
		Actions: []string{
			"Please contact support",
		},
		IsUser: false,
	},
	E_ARGUMENTS_MISMATCH: {
		Code:        10104,
		ErrorCode:   "E_ARGUMENTS_MISMATCH",
		Description: "Incorrect number of arguments supplied to function( the function is non-variadic)",
		Causes: []string{
			"either lesser or more number of arguments are passed for eg: CREATE OR REPLACE FUNCTION default:`default`.`_default`.`add4` (a,b) LANGUAGE inline AS ((a+b));  \nSELECT default._default.add4(1);  \nSELECT default._default.add4(1,2,3);",
		},
		Actions: []string{
			"To solve this issue you can look up number of parameters in the definition-> either from ui \nOR using functions keyspace , taking example in the causes: SELECT len(definition.parameters) FROM system:functions WHERE identity.name=\"add4\";\n \"results\": [\n    {\n        \"$1\": 2\n    }\n    ]",
		},
		IsUser: true,
	},
	E_INVALID_FUNCTION_NAME: {
		Code:        10105,
		ErrorCode:   "E_INVALID_FUNCTION_NAME",
		Description: "Invalid function name ",
		Causes: []string{
			"As of now only default nampespace is supported,  function name in CREATE FUNCTION statement must be 1) `[func-name`] -> Global function ( expanded as default:[func_name]), 2) `[bucket]`.`[scope]`.`[func_name]` -> Scope function( explanded as default:[bucket]`.`[scope]`.`[func_name]`",
		},
		Actions: []string{
			"Change the function name to follow the scheme specified in the cause",
		},
		IsUser: true,
	},
	E_FUNCTIONS_STORAGE: {
		Code:        10106,
		ErrorCode:   "E_FUNCTIONS_STORAGE",
		Description: "couldn't access function definition / function change counter",
		Causes: []string{
			"Could not access function definition for 1) LOAD(during execution): as get operation in metaKv or storage failed. 2) SAVE(Replace): as get or set operation in metaKv or storage failed. 3) DELETE(drop): as delete operation failed. \nCould not access changecounter: change counter is maintained in metakv for functionscache(system:functioncache-> information of recently used udfs) monitoring purpose. Immediately after LOAD/SAVE/DELETE for a function entry we update its change counter, but while doing so a metakv error(bad response) has occured",
		},
		Actions: []string{
			"Please contact support",
		},
		IsUser: false,
	},
	E_FUNCTION_ENCODING: {
		Code:        10107,
		ErrorCode:   "E_FUNCTION_ENCODING",
		Description: "Could not [encode/decode] function definition for [func_name]",
		Causes: []string{
			"1) encode: during a SAVE, the functions body is marshalled to be stored in metakv, 2) decode: during LOAD, the function body is unmarshalled from the response body from meta/system storage. 3) the LOAD and SAVE mechanism for functions also happen during bucket backups",
		},
		Actions: []string{
			"Please contact support",
		},
		IsUser: false,
	},
	E_FUNCTIONS_DISABLED: {
		Code:        10108,
		ErrorCode:   "E_FUNCTIONS_DISABLED",
		Description: "internal/external javascript functions are disabled.",
		Causes: []string{
			"1) if either internal or external or tenant evaluator(for serverless offering) are not initialized the respective javascript function support is disabled.",
		},
		Actions: []string{},
		IsUser:  false,
	},
	E_FUNCTION_EXECUTION: {
		Code:        10109,
		ErrorCode:   "E_FUNCTION_EXECUTION",
		Description: "Error executing function [inline function/javascript function]",
		Causes: []string{
			"when evaluating the definition of the inline function, the definition expression evalution raises error. (N1QL philosophy is if an expression doesn't make sense you don't error out but rather return null, so errors from inline function are likely to be system errors). Same goes for javascript errors like nil deference( eg: let l; let c = l.a;) , wrong type assertion are errored out.",
		},
		Actions: []string{
			"try debugging your definition step by step from the error message returned(which line in the definition fails). For javascript udfs look at \"details\" field. Or ask on couchbase forums",
		},
	},
	E_TOO_MANY_NESTED_FUNCTIONS: {
		Code:        10112,
		ErrorCode:   "E_TOO_MANY_NESTED_FUNCTIONS",
		Description: "Error executing function: [func-name]: [max-level] nested calls",
		Causes: []string{
			"The problem here is essentially: hitting the max-level depth,i.e maximum calls from a udf . To better explain lets take this example (I call you and you call me). \n/* libA*/\nfunction funcA(a,b) {\n  var p = EXECUTE FUNCTION funcB();\n  var res = []\n  for(const doc of p) {\n      res.push(doc)\n  }\n  return res;\n}\n\n/* libB */\nfunction funcB() {\n  \n  var q = EXECUTE FUNCTION funcA(1,3);\n  var res = []\n  for(const doc of q) {\n      res.push(doc);\n  }\n  return res;\n}\n\n//\nCREATE FUNCTION funcB() LANGUAGE JAVASCRIPT AS \"funcB\" AT \"libB\";\n\n//\nCREATE FUNCTION funcA() LANGUAGE JAVASCRIPT AS \"funcB\" AT \"libB\";\n\n//\nEXECUTE FUNCTION funcA(1,2);\n[\n  {\n    \"code\": 10109,\n    \"msg\": \"Error executing function 'funcA' (libA:funcA)\",\n    \"reason\": {\n      \"details\": {\n        \"Code\": \"  var p = N1QL('EXECUTE FUNCTION funcB();', {}, true);\",\n        \"Exception\": {\n          \"_level\": \"exception\",\n          \"caller\": \"javascript:386\",\n          \"code\": 10112,\n          \"key\": \"function.nested.error\",\n          \"message\": \"Error executing function 'funcA': 129 nested javascript calls\"\n        },\n        \"Location\": \"functions/libA.js:3\",\n        \"Stack\": \"   at funcA (functions/libA.js:3:11)\"\n      },\n      \"type\": \"Exceptions from JS code\"\n    }\n  }\n]",
		},
		Actions: []string{
			"in your definition ensure you aren't running into a circular callback loop",
		},
	},
	E_INNER_FUNCTION_EXECUTION: {
		Code:        10113,
		ErrorCode:   "E_INNER_FUNCTION_EXECUTION",
		Description: "UNUSED: not show to end user, internal to js-evaluator. (Need: func A-> func B-> func C (where a query fails) ,this is propagated up by unwinding the stack and propagated to funcA level. So never actually raised to end user.",
		Causes:      []string{},
		Actions:     []string{},
	},
	E_LIBRARY_PATH_ERROR: {
		Code:        10114,
		ErrorCode:   "E_LIBRARY_PATH_ERROR",
		Description: "Invalid javascript library path: [lib-path] ",
		Causes: []string{
			"CREATE FUNCTION [func-name](...) LANGUAGE JAVASCRIPT AS [func-in-lib] AT [lib-path].  Expected libpath to be 1) for global libraries the library name as is (for eg: \"Add\" or relative path \"./Add\") 2) for scope libraries , need to pass [bucket]/[scope]/[libraryname](eg: default/_default/[lib-name] or relative-path ./default/_default/[lib-name] )",
		},
		Actions: []string{
			"Please change the path to library to expected format explained in the cause",
		},
		IsUser: true,
	},
	E_FUNCTION_LOADING: {
		Code:        10115,
		ErrorCode:   "E_FUNCTION_LOADING",
		Description: "Error loading function",
		Causes: []string{
			"Function Loading error\n1) when using internalJS CREATE FUNCTION [func-name](..) lANGUAGE AS JAVASCRIPT AS \"[js-code]\"; , couldn't transpile code. 2) using external evaluator-> either failed to get the evaluator or load the library function from library ( the function exists but something went wrong in evaluator logic in eventing service.\n\nEvaluator Loading error\n1) \n\nEvaluator Inflating error\n1)",
		},
		Actions: []string{
			"// find out where to look for eventing logs etc",
		},
		IsUser: false,
	},
	E_FUNCTIONS_UNSUPPORTED_ACTION: {
		Code:        10118,
		ErrorCode:   "E_FUNCTIONS_UNSUPPORTED_ACTION",
		Description: "NEVER RAISED outside , only for dummy runner",
		Causes:      []string{},
		Actions:     []string{},
	},
	E_FUNCTION_STATEMENTS: {
		Code:        10119,
		ErrorCode:   "E_FUNCTION_STATEMENTS",
		Description: "",
		Causes:      []string{},
		Actions:     []string{},
	},
	E_DATASTORE_INVALID_BUCKET_PARTS: {
		Code:        10200,
		ErrorCode:   "E_DATASTORE_INVALID_BUCKET_PARTS",
		Description: "",
		Causes:      []string{},
		Actions:     []string{},
	},
	E_QUERY_CONTEXT: {
		Code:        10201,
		ErrorCode:   "E_QUERY_CONTEXT",
		Description: "invalid argument to query-context in request",
		Causes: []string{
			"perhaps missing backtick/ invalid namespace(currently only support default)",
		},
		Actions: []string{
			"you can look over here for more clarity https://docs.couchbase.com/server/current/n1ql/n1ql-rest-api/index.html#_request_parameters ",
		},
		IsUser: true,
	},
	E_BUCKET_NO_DEFAULT_COLLECTION: {
		Code:        10202,
		ErrorCode:   "E_BUCKET_NO_DEFAULT_COLLECTION",
		Description: "Bucket [name] does not have a default collection",
		Causes: []string{
			"1) if selecting on the bucket, we are actually selecting _default scope, _default collection, but this may be missing.",
			"2) GRANT/REVOKE role on the bucket, but no _default scope, _default collection.",
			"3) Similarly if INSERT/UPDATE on bucket.",
		},
		Actions: []string{
			"look for specific collection and specify path as bucket.scope.collection.",
		},
		IsUser: false,
	},
	E_NO_DATASTORE: {
		Code:        10203,
		ErrorCode:   "E_NO_DATASTORE",
		Description: "No datastore is available",
		Causes: []string{
			"1) when doing bucket backup using POST req to api/v1/bucket/{bucket}/backup and have cbo=on, and try to get _system collection, but datastore instance is not set we raise this error.",
			"2) In a similar fashion when JS udf library are stored under bucket.scope When try to \"get\" function on editing body, or \"save\" on adding a new udf , or \"Load\" i.e if storage counter is changed we re-\"get\" in case of execution of the function/explain function, or \"delete\"  when we delete the function from storage or during migration of functions to n1ql_system_bucket from _system_collection. So for \"get\",\"save\",\"load\", \"delete\" scenarios but datastore is not set.",
		},
		Actions: []string{
			"Possible that the node where memory/storage is, is down but another node where query is up.",
		},
		IsUser: true,
	},
	E_BUCKET_UPDATER_MAX_ERRORS: {
		Code:        10300,
		ErrorCode:   "E_BUCKET_UPDATER_MAX_ERRORS",
		Description: "Max failures reached.",
		Causes: []string{
			"During Query Planning when, we go from Algebra node of the keyspace (for eg: a simpleFromTerm(`default`) in SELECT * FROM default;) to the actual datastore keyspace abstraction. This involves GET request from streamingUrl-> /pools/default/bucketsStreaming/{:bucket} to load in bucket/keyspace information and stored in namespace's keyspace cache. But when the bucket is updated for eg: adding a KV node and rebalance-> the updater's streaming callback is called to reset the bucket's updater. During this process we hit MAX_RETRIES LIMIT for request to streamingUrl( /pools/default/bucketsStreaming/{:bucket} ).",
		},
		Actions: []string{
			"This is problem with NsServer possibly, contact support.",
		},
		IsUser: false,
	},
	E_BUCKET_UPDATER_NO_HEALTHY_NODES: {
		Code:        10301,
		ErrorCode:   "E_BUCKET_UPDATER_NO_HEALTHY_NODES",
		Description: "No healthy nodes found.",
		Causes: []string{
			"When running the updater to get latest bucket info, no nodes where the bucket resides where found to be healthy",
		},
		Actions: []string{},
		IsUser:  false,
	},
	E_BUCKET_UPDATER_STREAM_ERROR: {
		Code:        10302,
		ErrorCode:   "E_BUCKET_UPDATER_STREAM_ERROR",
		Description: "Streaming error",
		Causes: []string{
			"Error while creating http request using go's http package to the streamUrl-> /pools/default/bucketsStreaming/{:bucket}",
		},
		Actions: []string{
			"Contact Support",
		},
		IsUser: false,
	},
	E_BUCKET_UPDATER_AUTH_ERROR: {
		Code:        10303,
		ErrorCode:   "E_BUCKET_UPDATER_AUTH_ERROR",
		Description: "Authentication error",
		Causes: []string{
			"Error while getting auth user & password from the authentication handler(either from cbauth or from auth header in the client request)",
		},
		Actions: []string{
			"Contact Support",
		},
		IsUser: false,
	},
	E_BUCKET_UPDATER_CONNECTION_FAILED: {
		Code:        10304,
		ErrorCode:   "E_BUCKET_UPDATER_CONNECTION_FAILED",
		Description: "Failed to connect to host.",
		Causes: []string{
			"GET request to /pools/default/bucketsStreaming/{:bucket} endpoint got a response status of 5XX or 408 Request Timeout",
		},
		Actions: []string{
			"Contact Support",
		},
		IsUser: false,
	},
	E_BUCKET_UPDATER_ERROR_MAPPING: {
		Code:        10305,
		ErrorCode:   "E_BUCKET_UPDATER_ERROR_MAPPING",
		Description: "Mapping error: from Kv-TCP to Kv-TLS",
		Causes: []string{
			"While running doing bucket updater logic and client expects tls, if KV service is distributed across nodes. We take host:tcp-port and map to host:kv-ssl port",
		},
		Actions: []string{},
		IsUser:  false,
	},
	E_BUCKET_UPDATER_EP_NOT_FOUND: {
		Code:        10306,
		ErrorCode:   "E_BUCKET_UPDATER_EP_NOT_FOUND",
		Description: "Streaming endpoint not found",
		Causes: []string{
			"GET request to /pools/default/bucketsStreaming/{:bucket} endpoint got a response status of 404 Not Found",
		},
		Actions: []string{
			"Some problem with NsServer(Orchestrator process), Contact Support",
		},
	},
	E_ADVISOR_SESSION_NOT_FOUND: {
		Code:        10500,
		ErrorCode:   "E_ADVISOR_SESSION_NOT_FOUND",
		Description: "Advisor: Session not found",
		Causes: []string{
			"user has run Advisor function with stop_object https://docs.couchbase.com/server/current/n1ql/n1ql-language-reference/advisor.html#advisor-session-stop .  But session id passed does not exist in the task_cache.",
		},
		Actions: []string{
			"run SELECT * FROM system:tasks_cache;",
		},
		IsUser: true,
	},
	E_ADVISOR_INVALID_ACTION: {
		Code:        10501,
		ErrorCode:   "E_ADVISOR_INVALID_ACTION",
		Description: "Advisor: Invalid value for 'action",
		Causes: []string{
			"invalid value for \"actions\" field.",
		},
		Actions: []string{
			"Allowed values for \"actions\"-> \"get\", \"purge\", \"abort\", \"list\", \"stop\", link to advisor function documentation : https://docs.couchbase.com/server/current/n1ql/n1ql-language-reference/advisor.html ",
		},
		IsUser: true,
	},
	E_ADVISOR_ACTION_MISSING: {
		Code:        10502,
		ErrorCode:   "E_ADVISOR_ACTION_MISSING",
		Description: "Advisor: missing argument for 'action",
		Causes: []string{
			"input object to advisor must always have an \"action\" field.",
		},
		Actions: []string{
			"Allowed values for \"actions\"-> \"get\", \"purge\", \"abort\", \"list\", \"stop\", link to advisor function documentation : https://docs.couchbase.com/server/current/n1ql/n1ql-language-reference/advisor.html   ",
		},
		IsUser: true,
	},
	E_ADVISOR_INVALID_ARGS: {
		Code:        10503,
		ErrorCode:   "E_ADVISOR_INVALID_ARGS",
		Description: "Advisor: Invalid arguments.",
		Causes: []string{
			"some fields in the advisor object are not valid. Example bad query:  SELECT ADVISOR({\"act\": \"start\", \"novalid\": \"cd040b30-59a2-4b3c-81fa-6ab748\"});\n{\n    \"requestID\": \"acdf5d13-fad2-461f-b817-f26c13b41dd6\",\n    \"signature\": {\n        \"$1\": \"object\"\n    },\n    \"results\": [\n    ],\n    \"errors\": [\n        {\n            \"code\": 5010,\n            \"msg\": \"Error evaluating projection\",\n            \"reason\": {\n                \"_level\": \"exception\",\n                \"caller\": \"func_advisor:399\",\n                \"cause\": {\n                    \"args\": [\n                        \"act\",\n                        \"novalid\"\n                    ]\n                },\n                \"code\": 10503,\n                \"key\": \"function.advisor.invalid_arguments\",\n                \"message\": \"Advisor: Invalid arguments.\"\n            }\n        }\n    ],",
		},
		Actions: []string{
			"allowed fieldnames-> for startobject https://docs.couchbase.com/server/current/n1ql/n1ql-language-reference/advisor.html#arguments-3 \nfor listobject https://docs.couchbase.com/server/current/n1ql/n1ql-language-reference/advisor.html#arguments-4 \nfor stopobject https://docs.couchbase.com/server/current/n1ql/n1ql-language-reference/advisor.html#advisor-session-stop \nfor abortobject https://docs.couchbase.com/server/current/n1ql/n1ql-language-reference/advisor.html#arguments-6 \nfor getobject https://docs.couchbase.com/server/current/n1ql/n1ql-language-reference/advisor.html#arguments-7 \nfor purgeoject https://docs.couchbase.com/server/current/n1ql/n1ql-language-reference/advisor.html#arguments-8 ",
		},
		IsUser: true,
	},
	E_SYSTEM_DATASTORE: {
		Code:        11000,
		ErrorCode:   "E_SYSTEM_DATASTORE",
		Description: "",
		Causes:      []string{},
		Actions:     []string{},
	},
	E_SYSTEM_KEYSPACE_NOT_FOUND: {
		Code:        11002,
		ErrorCode:   "E_SYSTEM_KEYSPACE_NOT_FOUND",
		Description: "Keyspace not found in system namespace",
		Causes: []string{
			"keyspace provided by user doesn't exist in system namespace",
		},
		Actions: []string{
			"look over here https://docs.couchbase.com/server/current/n1ql/n1ql-intro/sysinfo.html , to see keyspaces in system namespace",
		},
		IsUser: true,
	},
	E_SYSTEM_NOT_IMPLEMENTED: {
		Code:        11003,
		ErrorCode:   "E_SYSTEM_NOT_IMPLEMENTED",
		Description: "UNUSED",
		Causes:      []string{},
		Actions:     []string{},
	},
	E_SYSTEM_NOT_SUPPORTED: {
		Code:        11004,
		ErrorCode:   "E_SYSTEM_NOT_SUPPORTED",
		Description: "System datastore : Not supported",
		Causes: []string{
			"Cannot CREATE PRIMARY INDEX / CREATE INDEX / BUILD INDEX on system catalog keyspaces",
		},
		Actions: []string{
			"Can query on system datastore without creating index.",
		},
		IsUser: true,
	},
	E_SYSTEM_IDX_NOT_FOUND: {
		Code:        11005,
		ErrorCode:   "E_SYSTEM_IDX_NOT_FOUND",
		Description: "",
		Causes:      []string{},
		Actions:     []string{},
	},
	E_SYSTEM_IDX_NO_DROP: {
		Code:        11006,
		ErrorCode:   "E_SYSTEM_IDX_NO_DROP",
		Description: "System datastore : This index cannot be dropped ",
		Causes:      []string{},
		Actions:     []string{},
	},
	E_SYSTEM_STMT_NOT_FOUND: {
		Code:        11007,
		ErrorCode:   "E_SYSTEM_STMT_NOT_FOUND",
		Description: "System datastore : Statement not found",
		Causes:      []string{},
		Actions:     []string{},
	},
	W_SYSTEM_REMOTE: {
		Code:        11008,
		ErrorCode:   "W_SYSTEM_REMOTE",
		Description: "",
		Causes:      []string{},
		Actions:     []string{},
	},
	E_SYSTEM_UNABLE_TO_RETRIEVE: {
		Code:        11009,
		ErrorCode:   "E_SYSTEM_UNABLE_TO_RETRIEVE",
		Description: "",
		Causes:      []string{},
		Actions:     []string{},
	},
	E_SYSTEM_UNABLE_TO_UPDATE: {
		Code:        11010,
		ErrorCode:   "E_SYSTEM_UNABLE_TO_UPDATE",
		Description: "System datastore : unable to update user information in server",
		Causes: []string{
			"Your query to GRANT or REVOKE set roles using GRANT/REVOKE has not been handled by \"/settings/rbac/users/[local]/{user}\" and returns bad response",
		},
		Actions: []string{
			"try again",
		},
		IsUser: false,
	},
	W_SYSTEM_FILTERED_ROWS: {
		Code:        11011,
		ErrorCode:   "W_SYSTEM_FILTERED_ROWS",
		Description: "One or more documents were excluded from the system bucket because of insufficient user permissions,",
		Causes: []string{
			"User is COUNTING or querying on system collections but doesn't have query_system_catalog role, eg: 1) SELECT COUNT(*) FROM system:indexes -> but user doesn't have query_system_catalog role.",
		},
		Actions: []string{
			"Not an error just a warning, ask Admin to grant query _system_catalog role if required by the user",
		},
		IsUser: true,
	},
	E_SYSTEM_MALFORMED_KEY: {
		Code:        11012,
		ErrorCode:   "E_SYSTEM_MALFORMED_KEY",
		Description: "System datastore : key is not of the correct format for keyspace",
		Causes: []string{
			"Expected key format from index:-> 1) {namespace}/{bucket}/{index_id} or 2) {namespace}/{bucket}/{scope}/{collection}/{index_id} for Fetch Call to dataservice",
		},
		Actions: []string{
			"DROP INDEX and recreate as document keys are malformed, possibly contact the support team",
		},
		IsUser: false,
	},
	E_SYSTEM_NO_BUCKETS: {
		Code:        11013,
		ErrorCode:   "E_SYSTEM_NO_BUCKETS",
		Description: "The system namespace contains no buckets that contain scopes.",
		Causes: []string{
			"the system namespace buckets don't have bucket.scope.collection format, i.e no scopes or collection only bucket. For eg: system:indexes.",
		},
		Actions: []string{
			"Please figure out which system bucket you want to query on from https://docs.couchbase.com/server/current/n1ql/n1ql-intro/sysinfo.html ",
		},
	},
	W_SYSTEM_REMOTE_NODE_NOT_FOUND: {
		Code:        11015,
		ErrorCode:   "W_SYSTEM_REMOTE_NODE_NOT_FOUND",
		Description: "",
		Causes:      []string{},
		Actions:     []string{},
	},
	E_CB_CONNECTION: {
		Code:        12000,
		ErrorCode:   "E_CB_CONNECTION",
		Description: "cbq-engine cannot connect to the cluster on for the provided URL (default- http://127.0.0.1:8091)",
		Causes: []string{
			"on startup cbq-engine creates a datastore instance which gives access to various information about all buckets & services across the cluster. This involves creating couchbase client which when cbauth database is not initialized proceeds with a basic URL based authorization, on failure here this error is thrown",
		},
		Actions: []string{},
	},
	E_CB_NAMESPACE_NOT_FOUND: {
		Code:        12002,
		ErrorCode:   "E_CB_NAMESPACE_NOT_FOUND",
		Description: "Namespace not found in CB datastore",
		Causes: []string{
			"typically this means server failed to get the default pool",
		},
		Actions: []string{},
		IsUser:  false,
	},
	E_CB_KEYSPACE_NOT_FOUND: {
		Code:        12003,
		ErrorCode:   "E_CB_KEYSPACE_NOT_FOUND",
		Description: "Keyspace not found in CB datastore",
		Causes: []string{
			"couldn't find a bucket or collection of the provided name by the user",
		},
		Actions: []string{
			"use \"SELECT * FROM system:keyspaces\" and check the fields bucket & name to see all existing keyspaces in the cluster",
		},
		IsUser: true,
	},
	E_CB_PRIMARY_INDEX_NOT_FOUND: {
		Code:        12004,
		ErrorCode:   "E_CB_PRIMARY_INDEX_NOT_FOUND",
		Description: "NOT in 7.6",
		Causes:      []string{},
		Actions:     []string{},
	},
	E_CB_INDEXER_NOT_IMPLEMENTED: {
		Code:        12005,
		ErrorCode:   "E_CB_INDEXER_NOT_IMPLEMENTED",
		Description: "Indexer not implemented ",
		Causes: []string{
			"when indexing service(GSI) is not enabled or similarly if using FTS index in a query but FTS service is not enabled",
		},
		Actions: []string{},
		IsUser:  true,
	},
	E_CB_KEYSPACE_COUNT: {
		Code:        12006,
		ErrorCode:   "E_CB_KEYSPACE_COUNT",
		Description: "Failed to get count for keyspace ",
		Causes: []string{
			"when selecting on system:keyspaces_info to to get count of number of documents from bucket API, but we get an error while doing so",
		},
		Actions: []string{
			"try again after sometime",
		},
		IsUser: false,
	},
	E_CB_BULK_GET: {
		Code:        12008,
		ErrorCode:   "E_CB_BULK_GET",
		Description: "Error performing bulk get operation ",
		Causes: []string{
			"when fetching documents from a bucket if has more than one document we issue bulkget operation, this error is raised when max number of retries is hit or default timeout is hit",
		},
		Actions: []string{
			"might be the kv is overwhelmed and your request was throttled you would have to just retry",
		},
		IsUser: false,
	},
	E_CB_DML: {
		Code:        12009,
		ErrorCode:   "E_CB_DML",
		Description: "",
		Causes:      []string{},
		Actions:     []string{},
	},
	E_CB_DELETE_FAILED: {
		Code:        12011,
		ErrorCode:   "E_CB_DELETE_FAILED",
		Description: "delete request failed",
		Causes: []string{
			"regardless of parallelised mututaion ops or single mutation op, for a particular key the delete operation has got a bad response, thats when this error is raised",
		},
		Actions: []string{},
		IsUser:  false,
	},
	E_CB_LOAD_INDEXES: {
		Code:        12012,
		ErrorCode:   "E_CB_LOAD_INDEXES",
		Description: "UNUSED",
		Causes:      []string{},
		Actions:     []string{},
	},
	E_CB_BUCKET_TYPE_NOT_SUPPORTED: {
		Code:        12013,
		ErrorCode:   "E_CB_BUCKET_TYPE_NOT_SUPPORTED",
		Description: "This bucket type is not supported",
		Causes: []string{
			"as 7.6 only memcached buckets are deprecated and are not supported to query on, couchbase and ephimeral buckets are allowed",
		},
		Actions: []string{
			"migrate all your documents to a couchbase bucket ",
		},
		IsUser: true,
	},
	E_CB_INDEX_SCAN_TIMEOUT: {
		Code:        12015,
		ErrorCode:   "E_CB_INDEX_SCAN_TIMEOUT",
		Description: "Index scan timed out",
		Causes: []string{
			"particular for primaryscan, if the indexer cannot find a snapshot that satisfies the consistency guarantee of the query within the timeout limit, it will timeout without returning any primary keys. Query service will resort to chunk based scanning,i.e successive scans until all primary keys are returned. But when query has Offset/Aggregates/Order is when this error is raised, as for the earlier mentioned clauses we require results to be exact.",
		},
		Actions: []string{
			"please create appropriate secondary index.",
		},
		IsUser: false,
	},
	E_CB_INDEX_NOT_FOUND: {
		Code:        12016,
		ErrorCode:   "E_CB_INDEX_NOT_FOUND",
		Description: "Index Not Found",
		Causes: []string{
			"while using ALTER or DROP INDEX statements the name of index in query doesn't exist",
		},
		Actions: []string{
			"check available indexes on the cluster using system:indexes ",
		},
		IsUser: true,
	},
	E_CB_GET_RANDOM_ENTRY: {
		Code:        12017,
		ErrorCode:   "E_CB_GET_RANDOM_ENTRY",
		Description: "Error getting random entry from keyspace",
		Causes: []string{
			"when using INFER statement ,underhood request to kv is getrandomdoc op for which on receiving a bad response we raise this error",
		},
		Actions: []string{},
		IsUser:  false,
	},
	E_UNABLE_TO_INIT_CB_AUTH: {
		Code:        12018,
		ErrorCode:   "E_UNABLE_TO_INIT_CB_AUTH",
		Description: "",
		Causes:      []string{},
		Actions:     []string{},
	},
	E_AUDIT_STREAM_HANDLER_FAILED: {
		Code:        12019,
		ErrorCode:   "E_AUDIT_STREAM_HANDLER_FAILED",
		Description: "",
		Causes:      []string{},
		Actions:     []string{},
	},
	E_CB_BUCKET_NOT_FOUND: {
		Code:        12020,
		ErrorCode:   "E_CB_BUCKET_NOT_FOUND",
		Description: "Scope not found in CB datastore",
		Causes: []string{
			"Only raised when dropping a non-existing scope,",
		},
		Actions: []string{
			"verify existing scopes using \"SELECT `scope` FROM system:keyspaces WHERE `bucket`=[BUCKETNAME];",
		},
		IsUser: true,
	},
	E_CB_KEYSPACE_SIZE: {
		Code:        12022,
		ErrorCode:   "E_CB_KEYSPACE_SIZE",
		Description: "Failed to get size for keyspace",
		Causes: []string{
			"when querying on system:keyspace_info, we get a bad response from the bucket stats api",
		},
		Actions: []string{
			"try again after sometime",
		},
		IsUser: false,
	},
	E_CB_SECURITY_CONFIG_NOT_PROVIDED: {
		Code:        12023,
		ErrorCode:   "E_CB_SECURITY_CONFIG_NOT_PROVIDED",
		Description: "Connection security config not provided. Unable to load bucket",
		Causes: []string{
			"TLS settings and node to node security settings if not passed to datastore, it refuses to startup buckets",
		},
		Actions: []string{
			"check if cbauth is initialized",
		},
		IsUser: false,
	},
	E_CB_CREATE_SYSTEM_BUCKET: {
		Code:        12024,
		ErrorCode:   "E_CB_CREATE_SYSTEM_BUCKET",
		Description: "Error while creating system bucket",
		Causes: []string{
			"onprem - N1QL_SYSTEM_BUCKET is created for cbo purposes. This errors is raised when the bucket is not created.",
		},
		Actions: []string{},
		IsUser:  false,
	},
	E_CB_BUCKET_CREATE_SCOPE: {
		Code:        12025,
		ErrorCode:   "E_CB_BUCKET_CREATE_SCOPE",
		Description: "Error while creating scope",
		Causes: []string{
			"1. CREATE SCOPE [bucket].[scope] may result in error being raised if [scope] already exists",
			"2. failed to created _N1QL_SYSTEM_SCOPE on _N1QL_SYSTEM_BUCKET for update statistics",
		},
		Actions: []string{},
	},
	E_CB_BUCKET_DROP_SCOPE: {
		Code:        12026,
		ErrorCode:   "E_CB_BUCKET_DROP_SCOPE",
		Description: "Error while dropping scope",
		Causes: []string{
			"DROP SCOPE [bucket].[scope] , scope doesn't exist",
		},
		Actions: []string{},
		IsUser:  true,
	},
	E_CB_BUCKET_CREATE_COLLECTION: {
		Code:        12027,
		ErrorCode:   "E_CB_BUCKET_CREATE_COLLECTION",
		Description: "Error while creating collection",
		Causes: []string{
			"1. option is not \"maxTTL\", 2. maxTTL option value is not a number, 3. post request to the api for create collection returned error",
		},
		Actions: []string{},
	},
	E_CB_BUCKET_DROP_COLLECTION: {
		Code:        12028,
		ErrorCode:   "E_CB_BUCKET_DROP_COLLECTION",
		Description: "Error while dropping collection ",
		Causes: []string{
			"DROP COLLECTION [bucket].[scope].[collection] , post request to the api for drop collection returns error as it is not successful.",
		},
		Actions: []string{
			"Maybe check if collection/scope/bucket exists",
		},
		IsUser: false,
	},
	E_CB_BUCKET_FLUSH_COLLECTION: {
		Code:        12029,
		ErrorCode:   "E_CB_BUCKET_FLUSH_COLLECTION",
		Description: "Error while flushing collection",
		Causes: []string{
			"FLUSH COLLECT [keyspace], post request to bucket api /pools/default/buckets/default/controller/doFlush is not successful, as 1. bucket may not have flushEnabled, 2. keyspace doesn't exist",
		},
		Actions: []string{
			"follow https://docs.couchbase.com/server/current/rest-api/rest-bucket-create.html#flushenabled to create flush enabled bucket or edit existing bucket to allow flushing",
		},
		IsUser: true,
	},
	E_BINARY_DOCUMENT_MUTATION: {
		Code:        12030,
		ErrorCode:   "E_BINARY_DOCUMENT_MUTATION",
		Description: "Mutation of binary document is not supported",
		Causes: []string{
			"non-JSON serialization will prevent the document from being accessible via Query, for INSERT/UPDATE/UPSERT, but is allowed for DELETE.",
		},
		Actions: []string{},
		IsUser:  true,
	},
	E_DURABILITY_NOT_SUPPORTED: {
		Code:        12031,
		ErrorCode:   "E_DURABILITY_NOT_SUPPORTED",
		Description: "Durability is not supported in the SDK being used",
		Causes: []string{
			"to query on buckets that have durability enforced https://docs.couchbase.com/server/current/learn/data/durability.html , sdk must pass a client context without which the DML statement will not passthrough",
		},
		Actions: []string{
			"upgrade to a newer version of SDK ",
		},
		IsUser: true,
	},
	E_PRESERVE_EXPIRY_NOT_SUPPORTED: {
		Code:        12032,
		ErrorCode:   "E_PRESERVE_EXPIRY_NOT_SUPPORTED",
		Description: "Preserve expiration is not supported.",
		Causes: []string{
			"SDK version doesn't support preservexpiry",
		},
		Actions: []string{
			"upgrade to a newer version of SDK ",
		},
		IsUser: true,
	},
	E_CAS_MISMATCH: {
		Code:        12033,
		ErrorCode:   "E_CAS_MISMATCH",
		Description: "Error performing bulk get operation, CAS mismatch-> only emebedded in E_CB_DML",
		Causes:      []string{},
		Actions:     []string{},
	},
	E_DML_MC: {
		Code:        12034,
		ErrorCode:   "E_DML_MC",
		Description: "MC(data service) error, usually wrapped in DMLerror / BulkGetError",
		Causes: []string{
			"Data service operation failed, fatal memcached response is received maybe during fetch or when performing mutation",
		},
		Actions: []string{
			"try again",
		},
		IsUser: false,
	},
	E_CB_NOT_PRIMARY_INDEX: {
		Code:        12035,
		ErrorCode:   "E_CB_NOT_PRIMARY_INDEX",
		Description: "Index you are trying to drop is not a primary index",
		Causes: []string{
			"DROP PRIMARY INDEX statement but index specified is not a primary index",
		},
		Actions: []string{
			"look up provider for the index using system:indexes keyspace. If gsi: use DROP INDEX statement instead, incase of fts please use ui to drop the index",
		},
		IsUser: true,
	},
	E_DML_INSERT: {
		Code:        12036,
		ErrorCode:   "E_DML_INSERT",
		Description: "Error in INSERT of a particular key-value pair",
		Causes: []string{
			"INSERT statement issues Add op per document to dataservice, if receiving a fatal response this error is raised",
		},
		Actions: []string{
			"try again",
		},
		IsUser: false,
	},
	E_ACCESS_DENIED: {
		Code:        12037,
		ErrorCode:   "E_ACCESS_DENIED",
		Description: "User doesn't have access to the particular keyspace in serverless",
		Causes: []string{
			"similar to insufficient datastore credentials but for security reasons in serverless a more generic error is raised",
		},
		Actions: []string{},
		IsUser:  true,
	},
	E_WITH_INVALID_OPTION: {
		Code:        12038,
		ErrorCode:   "E_WITH_INVALID_OPTION",
		Description: "Invalid option is passed by user",
		Causes: []string{
			"When passing options using WITH construct for CREATE COLLECTION/ CREATE SEQUENCE/ ALTER SEQUENCE",
		},
		Actions: []string{},
		IsUser:  true,
	},
	E_WITH_INVALID_TYPE: {
		Code:        12039,
		ErrorCode:   "E_WITH_INVALID_TYPE",
		Description: "Invalid value for option that expects a particular type",
		Causes: []string{
			"for example: in CREATE SEQUENCE option \"cycle\" expects bool(true/false) but user has passed non-bool",
		},
		Actions: []string{},
		IsUser:  true,
	},
	E_INVALID_COMPRESSED_VALUE: {
		Code:        12040,
		ErrorCode:   "E_INVALID_COMPRESSED_VALUE",
		Description: "Invalid compressed document received from datastore",
		Causes: []string{
			"Expects snappy compression encoded data",
		},
		Actions: []string{},
		IsUser:  false,
	},
	E_CB_BUCKET_CLOSED: {
		Code:        12041,
		ErrorCode:   "E_CB_BUCKET_CLOSED",
		Description: "Bucket is closed",
		Causes:      []string{},
		Actions:     []string{},
	},
	E_CB_SUBDOC_GET: {
		Code:        12042,
		ErrorCode:   "E_CB_SUBDOC_GET",
		Description: "UNUSED",
		Causes:      []string{},
		Actions:     []string{},
	},
	E_CB_SUBDOC_SET: {
		Code:        12043,
		ErrorCode:   "E_CB_SUBDOC_SET",
		Description: "Sub-doc set operation failed, when using sequences",
		Causes:      []string{},
		Actions:     []string{},
	},
	E_CB_DROP_SYSTEM_BUCKET: {
		Code:        12044,
		ErrorCode:   "E_CB_DROP_SYSTEM_BUCKET",
		Description: "Error while dropping system bucket ",
		Causes: []string{
			"onprem - at the end of migration N1QL_SYSTEM_BUCKET is dropped, but if request to drop is erronrous we raise this error",
		},
		Actions: []string{},
		IsUser:  false,
	},
	E_DATASTORE_CLUSTER: {
		Code:        13012,
		ErrorCode:   "E_DATASTORE_CLUSTER",
		Description: "Error retrieving cluster information ",
		Causes: []string{
			"user has run query on system:nodes keyspace, which internally checks the response from the nsserver endpoints: /pools/default/ , /pools/default/nodeServices, but has got a bad response. Or query service is interested in topology of the cluster.",
		},
		Actions: []string{},
		IsUser:  false,
	},
	E_DATASTORE_UNABLE_TO_RETRIEVE_ROLES: {
		Code:        13013,
		ErrorCode:   "E_DATASTORE_UNABLE_TO_RETRIEVE_ROLES",
		Description: "Unable to retrieve roles from server.",
		Causes: []string{
			"request to http://{ip-address-or-domain-name}:8091/settings/rbac/roles endpoint fails",
		},
		Actions: []string{},
		IsUser:  false,
	},
	E_DATASTORE_INSUFFICIENT_CREDENTIALS: {
		Code:        13014,
		ErrorCode:   "E_DATASTORE_INSUFFICIENT_CREDENTIALS",
		Description: "\"User doesn't have either QuerySelect/QueryUpdate/QueryInsert/QueryDelete",
		Causes: []string{
			"User doesn't have either QuerySelect/QueryUpdate/QueryInsert/QueryDelete",
		},
		Actions: []string{
			"admin has to grant user required RBAC using security tab in web console or follow https://docs.couchbase.com/server/current/rest-api/rbac.html",
		},
		IsUser: true,
	},
	E_DATASTORE_UNABLE_TO_RETRIEVE_BUCKETS: {
		Code:        13015,
		ErrorCode:   "E_DATASTORE_UNABLE_TO_RETRIEVE_BUCKETS",
		Description: "is not directly returned and is usually the cause for SystemDatastoreError:E_SYSTEM_DATASTORE",
		Causes: []string{
			"Occurs when doing COUNT on buckets in system catalog, when cbauth is stale or empty crendential list",
		},
		Actions: []string{},
		IsUser:  false,
	},
	E_DATASTORE_NO_ADMIN: {
		Code:        13016,
		ErrorCode:   "E_DATASTORE_NO_ADMIN",
		Description: "Unable to determine admin credentials",
		Causes: []string{
			"When running index advisor on serverless, we get Admin Credentials to use Admin context for Advisor session queries. But we fail to get back Admin credentials for that particular datastore",
		},
		Actions: []string{},
		IsUser:  false,
	},
	E_DATASTORE_NOT_SET: {
		Code:        13017,
		ErrorCode:   "E_DATASTORE_NOT_SET",
		Description: "Datastore not set",
		Causes: []string{
			"trying to get bucket / keyspace / scope information in the execution flow but datastore is not set",
		},
		Actions: []string{},
	},
	E_DATASTORE_INVALID_URI: {
		Code:        13018,
		ErrorCode:   "E_DATASTORE_INVALID_URI",
		Description: "Invalid datastore URI",
		Causes: []string{
			"if you are trying to build cbserver on your own, cbq-engine requires -> URI to have either http, dir, file, mock as protocol or file path but none of the following was received",
		},
		Actions: []string{
			"standard --datastore argument is -datastore=http://127.0.0.1:8091",
		},
		IsUser: true,
	},
	E_INDEX_SCAN_SIZE: {
		Code:        14000,
		ErrorCode:   "E_INDEX_SCAN_SIZE",
		Description: "when index connection buffer size is requested is less than 0 - NOT SURE IF THIS EVER HAPPENS",
		Causes:      []string{},
		Actions:     []string{},
	},
	E_SS_IDX_NOT_FOUND: {
		Code:        16050,
		ErrorCode:   "E_SS_IDX_NOT_FOUND",
		Description: "Index not found",
		Causes:      []string{},
		Actions:     []string{},
	},
	E_SS_NOT_SUPPORTED: {
		Code:        16051,
		ErrorCode:   "E_SS_NOT_SUPPORTED",
		Description: "[Operation/Statement] not supported for scan",
		Causes: []string{
			"1) CREATE PRIMARY - Fail Safe code, algebra tree for CREATE PRIMARY doesn't allow sequential scan as input for USING Clause",
			"2) CREATE INDEX  - Fail Safe code, again algebra tree for CREATE INDEX doesn't allow sequential scan as input for USING Clause",
			"3) BUILD INDEX - Fail Safe code, again algebra tree for BUILD INDEX doesn't allow sequential scan as input for USING Clause",
			"4) DROP INDEX - Fail Safe code, again algebra tree for DROP INDEX doesn't allow sequential scan as input for USING Clause",
			"5) ALTER INDEX - Fail Safe code, again algebra tree for ALTER INDEX doesn't allow sequential scan as input for USING Clause",
			"6) Datastore doesn't implement the interface required for KV-range scans ( used by Sequential Scan under the hood).",
		},
		Actions: []string{
			"For 6) upgrade dataservice to a version that supports KV-range scans",
		},
		IsUser: false,
	},
	E_SS_INACTIVE: {
		Code:        16052,
		ErrorCode:   "E_SS_INACTIVE",
		Description: "Inactive scan in Fetch",
		Causes: []string{
			"The scan coordinator go routine has left the scan object inactive possibly due to a cancel on timeout or error, thus the fetch procedure cannot use the scan identifier to get keys.",
		},
		Actions: []string{
			"Contact Support.",
		},
		IsUser: false,
	},
	E_SS_INVALID: {
		Code:        16053,
		ErrorCode:   "E_SS_INVALID",
		Description: "Invalid scan in [stop/fetch]",
		Causes: []string{
			"1) Fetch-> fail-safe code, to ensure scan received is of sequential-scan",
			"2) Stop-> fail-safe code, to ensure scan received is of sequential-scan",
		},
		Actions: []string{
			"Contact Support.",
		},
		IsUser: false,
	},
	E_SS_CONTINUE: {
		Code:        16054,
		ErrorCode:   "E_SS_CONTINUE",
		Description: "Scan continuation failed",
		Causes: []string{
			"1) kv doc https://github.com/couchbase/kv_engine/blob/abbf3412cad1fa8399b6e99bcb68bbfdad67ef4d/docs/range_scans/range_scan_continue.md \n1) data service responds with WOULD_THROTTLE / NOT_MY_VBUCKET / KEY_ENOENT response status on Range-scan-continue command, so we requeue the vbucket-scan handle, but the scan state was not the expected _WORKING, or queue assigned to the scan is higher than that of the number of queues predefined in the rangescanworkerController.",
			"2) Range-scan-continue command request returned with a response with status other than WOULD_THROTTLE / NOT_MY_VBUCKET/ KEY_ENOENT , we report the error",
			"3) On WOULD_THROTTLE response, we try to requeue the vbucket-scan handle so a worker can pick it up, after a brief suspend. But the requeue failed.",
			"4) Similarly if we receive  NOT_MY_VBUCKET/ KEY_ENOENT response status  , we requeue the vbucket-scan handle but the requeue failed.",
			"5) error from Range-scan-continue command has neither WOULD_THROTTLE / NOT_MY_VBUCKET/ KEY_ENOENT response status , hence we report the error and proceed to cancel the scan",
		},
		Actions: []string{
			"Contact Support.",
		},
		IsUser: false,
	},
	E_SS_CREATE: {
		Code:        16055,
		ErrorCode:   "E_SS_CREATE",
		Description: "Scan creation failed",
		Causes: []string{
			"1) kv doc https://github.com/couchbase/kv_engine/blob/abbf3412cad1fa8399b6e99bcb68bbfdad67ef4d/docs/range_scans/range_scan_create.md \n1) Create Range Scan failed with response status of WOULD_THROTTLE , so we try to requeue the scan after a suspend, but the requeue failed. Hence we report the error.",
			"2) Response for Range-scan-create is not WOULD_THROTTLE/KEY_ENOENT, we report the error and setup a retry and close the connection.",
		},
		Actions: []string{},
	},
	E_SS_CANCEL: {
		Code:        16056,
		ErrorCode:   "E_SS_CANCEL",
		Description: "UNUSED.",
		Causes:      []string{},
		Actions:     []string{},
	},
	E_SS_TIMEOUT: {
		Code:        16057,
		ErrorCode:   "E_SS_TIMEOUT",
		Description: "Scan exceeded permitted duration",
		Causes: []string{
			"1) Hit timeout when trying to send the keys(for ordered or unordered scan request) got by the coordinator goroutine to the fetchkeys goroutine, hence we error out and cancel all the scheduled vbucket-scans",
			"2) When trying queue a vbscan across the servers, we timedout.",
		},
		Actions: []string{
			"Increase KVTimeout",
		},
	},
	E_SS_CID_GET: {
		Code:        16058,
		ErrorCode:   "E_SS_CID_GET",
		Description: "Failed to get collection ID for scan",
		Causes: []string{
			"Memcached Op: COLLECTIONS_GET_CID, failed , but collection id is needed for CREATE_RANGE_SCAN op as a key.",
		},
		Actions: []string{
			"Contact support",
		},
		IsUser: false,
	},
	E_SS_CONN: {
		Code:        16059,
		ErrorCode:   "E_SS_CONN",
		Description: "Failed to get connection for scan",
		Causes: []string{
			"1) Worker failed to get a vbucket connection-> vbmap smaller than vbucket list / invalid vbmap entry for vb [vb-id] /  No master for vbucket / failed to get connection for the pool\n2) vbucket range scan connection failed, so we try to  requeue as we still have retries allowed for it so another worker can pick it up",
		},
		Actions: []string{},
	},
	E_SS_FETCH_WAIT_TIMEOUT: {
		Code:        16060,
		ErrorCode:   "E_SS_FETCH_WAIT_TIMEOUT",
		Description: "Timed out polling scan for data",
		Causes: []string{
			"Hit scanpoll timeout before we could get a key from the scan-coordinator goroutine on the channel.",
		},
		Actions: []string{
			"Increase KVTimeout",
		},
		IsUser: false,
	},
	E_SS_WORKER_ABORT: {
		Code:        16061,
		ErrorCode:   "E_SS_WORKER_ABORT",
		Description: "A fatal error occurred in scan processing",
		Causes: []string{
			"When a worker goes through the runScan logic (Create then Continue for a Vbucket) if we panic during the steps the scan is not completed and thus we add all the scans we have to cancel queue to ensure we abort the request for sequential scan and not take up anymore resources.",
		},
		Actions: []string{
			"Contact support",
		},
		IsUser: false,
	},
	E_SS_FAILED: {
		Code:        16062,
		ErrorCode:   "E_SS_FAILED",
		Description: "Scan failed",
		Causes: []string{
			"1) When request timeout is set in /admin/settings , we ensure the timeout hasn't expired before scan using Sequential Scan indexer.",
			"2) When tring to start key scan on a collection-> if something goes wrong in getting the collectionid using COLLECTIONS_GET_CID memcached Command, we error out.",
			"3) After starting the scan coordinator (i.e perform RANGE_SCAN_CREATE (& RANGE_SCAN_CONTINUE) ), but we hit the request timeout so, error out on the deadline and signal that the connection timed out.",
			"4) Fetching keys got from the range_scan_continue command, has internally returned a error response, thus the worker reports error over the channel instead of keys. This renders the scan as failed",
			"5) Timeout when sending the keys got from range_scan downstream",
		},
		Actions: []string{
			"Contact Support, possible bug.",
		},
		IsUser: false,
	},
	E_SS_SPILL: {
		Code:        16063,
		ErrorCode:   "E_SS_SPILL",
		Description: "Operation failed on scan spill file",
		Causes: []string{
			"1) The threshold before spilling is currently 10 KiB of keys, meaning for a full scan of all v-buckets, the key memory is limited to 10 MiB per sequential scan.",
			"2) in the processing of CREATE_RANGE_SCAN op , if reponse status returned is KEY_ENOENT (key doesn't exist). We truncate any resources used for the vbscan keys slice, buffer slice and spill file. To trucate the spill file we call Ftrucate syscall but if this fails we raise this error.",
			"3) similar release procedure is followed once all the vbscan are processed and nothing remains in the queue, again if Ftruncate syscall fails we raise this error.",
			"4) When reading in keys from countine_range_scan op, we add keys to in memory buffer first but if the buffer capacity is exhaused, we spill\n  i. spill file creation failed.\n  ii. when flushing buffer to spill file something went wrong in the write",
		},
		Actions: []string{
			"Contact support",
		},
		IsUser: false,
	},
	E_SS_VALIDATE: {
		Code:        16064,
		ErrorCode:   "E_SS_VALIDATE",
		Description: "Failed to validate document key",
		Causes: []string{
			"If the scan range is detected to be for a single key value then a single key-validation operation - instead of a scan - for one v-bucket is all that is generated.\n\nwe issue a REPLACE op for the key , and if response status is KEY_ENOENT. We raise this error and the kv-connection is discarded.",
		},
		Actions: []string{
			"Contact Support",
		},
		IsUser: false,
	},
	E_TRAN_DATASTORE_NOT_SUPPORTED: {
		Code:        17001,
		ErrorCode:   "E_TRAN_DATASTORE_NOT_SUPPORTED",
		Description: "UNUSED- only for mock/file based datastore",
		Causes:      []string{},
		Actions:     []string{},
	},
	E_TRAN_STATEMENT_NOT_SUPPORTED: {
		Code:        17002,
		ErrorCode:   "E_TRAN_STATEMENT_NOT_SUPPORTED",
		Description: "statement is not supported",
		Causes: []string{
			"1) START_TRANSACTION statement is not supported within the transaction, i.e cannot run START TRANSACTION statement when we have already issued the same statement before within the transaction timeout",
			"2) COMMIT / ROLLBACK / ROLLBACK_SAVEPOINT / SET_TRANSACTION_ISOLATION / SAVEPOINT are not allowed outside a transaction(i.e need to issue START TRANSACTION before)",
		},
		Actions: []string{
			"documentation reference for transactions https://docs.couchbase.com/server/current/n1ql/n1ql-language-reference/transactions.html , also flow this blog if you don't like documentation:) https://www.couchbase.com/blog/transactions-n1ql-couchbase-distributed-nosql/ ",
		},
		IsUser: true,
	},
	E_TRAN_FUNCTION_NOT_SUPPORTED: {
		Code:        17003,
		ErrorCode:   "E_TRAN_FUNCTION_NOT_SUPPORTED",
		Description: "advisor function is not supported within the transaction",
		Causes: []string{
			"Advisor Function is not allowed within a transaction, as it doesn't make sense semantically.",
		},
		Actions: []string{
			"Call Advisor outside of a transaction.",
		},
		IsUser: true,
	},
	E_TRANSACTION_CONTEXT: {
		Code:        17004,
		ErrorCode:   "E_TRANSACTION_CONTEXT",
		Description: "Transaction context error",
		Causes: []string{
			"The transcation id passed by client(either workbench/cbqshell/rest-api) is not present in the transaction cache on the node you are querying on. That is, failed to get the transaction context for that particular transaction id. Maybe, transaction timedout and there was a cleanup, or memory quota was hit. Other reasons tximplicit parameter is used and under the hood \"START TRANSACTION\" was issued before serving any requests,which failed for some reason, hence couldn't retrieve the transaction context.",
		},
		Actions: []string{
			"Contact support.",
		},
		IsUser: false,
	},
	E_TRAN_STATEMENT_OUT_OF_ORDER: {
		Code:        17005,
		ErrorCode:   "E_TRAN_STATEMENT_OUT_OF_ORDER",
		Description: "Transaction statement is out of order",
		Causes: []string{
			"User is not using tximplicit parameter , has started a transaction by issuing BEGIN WORK/START TRANSACTION. But has passed txstmtnum parameter in the request while executing following statements for the duration of the transaction. The scenario here is that previous request had a stmtnum> current request which is not allowed so we error out. Example-> cbq> START TRANSACTION;\n{\n    \"requestID\": \"9741b6f4-0e7a-49fa-900a-74840cc522e3\",\n    \"signature\": \"json\",\n    \"results\": [\n    {\n        \"nodeUUID\": \"444892295e0eb3a1cf180dad6104955a\",\n        \"txid\": \"cc55d22d-25ec-477f-9101-3c757f020f3e\"\n    }\n    ],\n    \"status\": \"success\",\n    \"metrics\": {\n        \"elapsedTime\": \"896.459\u00b5s\",\n        \"executionTime\": \"789.167\u00b5s\",\n        \"resultCount\": 1,\n        \"resultSize\": 118,\n        \"serviceLoad\": 2,\n        \"transactionElapsedTime\": \"458.792\u00b5s\",\n        \"transactionRemainingTime\": \"1m59.999537208s\"\n    }\n}\ncbq> \\set -txstmtnum 3;\ncbq> UPDATE customer SET balance = balance - 100 WHERE cid = 1924;\n{\n    \"requestID\": \"b7d8e634-7fe1-4a6c-9dbf-c4005881b2d3\",\n    \"signature\": null,\n    \"results\": [\n    ],\n    \"status\": \"success\",\n    \"metrics\": {\n        \"elapsedTime\": \"42.013958ms\",\n        \"executionTime\": \"41.685375ms\",\n        \"resultCount\": 0,\n        \"resultSize\": 0,\n        \"serviceLoad\": 0,\n        \"mutationCount\": 1,\n        \"transactionElapsedTime\": \"10.865749833s\",\n        \"transactionRemainingTime\": \"1m49.134244917s\"\n    }\n}\ncbq> \\set -txstmtnum 1;\ncbq> UPDATE customer SET balance = balance - 100 WHERE cid = 1924;\n{\n    \"requestID\": \"567b67d2-5c4d-4594-8d70-6b670cb90f80\",\n    \"errors\": [\n        {\n            \"code\": 17005,\n            \"msg\": \"Transaction statement is out of order (3, 1) \"\n        }\n    ],\n    \"status\": \"fatal\",\n    \"metrics\": {\n        \"elapsedTime\": \"291.459\u00b5s\",\n        \"executionTime\": \"74.709\u00b5s\",\n        \"resultCount\": 0,\n        \"resultSize\": 0,\n        \"serviceLoad\": 0,\n        \"errorCount\": 1\n    }\n}",
		},
		Actions: []string{
			"Correct your stmtnum parameter to always be incremental",
		},
		IsUser: true,
	},
	E_START_TRANSACTION: {
		Code:        17006,
		ErrorCode:   "E_START_TRANSACTION",
		Description: "Wrapper error for all error paths during START TRANSACTION / BEGIN WORK statement",
		Causes: []string{
			"1) failed to create transcation context, 2)",
		},
		Actions: []string{},
	},
	E_COMMIT_TRANSACTION: {
		Code:        17007,
		ErrorCode:   "E_COMMIT_TRANSACTION",
		Description: "",
		Causes:      []string{},
		Actions:     []string{},
	},
	E_ROLLBACK_TRANSACTION: {
		Code:        17008,
		ErrorCode:   "E_ROLLBACK_TRANSACTION",
		Description: "",
		Causes:      []string{},
		Actions:     []string{},
	},
	E_NO_SAVEPOINT: {
		Code:        17009,
		ErrorCode:   "E_NO_SAVEPOINT",
		Description: "savepoint is not defined",
		Causes: []string{
			"User has run ROLLBACK TRAN TO SAVEPOINT [savepoint-name];  But the savepoint [savepoint-name] was not defined",
		},
		Actions: []string{
			"Rerun with the correct savepoint-name",
		},
		IsUser: true,
	},
	E_TRANSACTION_EXPIRED: {
		Code:        17010,
		ErrorCode:   "E_TRANSACTION_EXPIRED",
		Description: "Transaction timeout",
		Causes: []string{
			"When setting transaction info for the request under transaction. We check if the transaction context for the transactionid has not expired, if it has we raised this error. All mutations done under this transaction are cleaned without commiting. Another place for raising this error user has issued DELETE FROM system:transactions; which renders all ongoing transactions as expired, the same for a DELETE request to admin/transactions/{transaction_id}",
		},
		Actions: []string{
			"default timeout cbq file = 15sec, for cbqshell interactive session/work-bench/rest-api = 2min. You can increase the timeout by setting txtimeout parameter https://docs.couchbase.com/server/current/settings/query-settings.html#txtimeout_req ",
		},
		IsUser: true,
	},
	E_TRANSACTION_RELEASED: {
		Code:        17011,
		ErrorCode:   "E_TRANSACTION_RELEASED",
		Description: "Transaction is released",
		Causes: []string{
			"During the transaction , user ran START TRANSACTION / COMMIT / ROLLBACK / SET_TRANSACTION_ISOLATION / SAVEPOINT / ROLLBACK_SAVEPOINT statement but the statement completes with errors, then we Delete the transaction entry in system:transactions also mark the transaction status as RELEASED so any further queries coming in with the same transaction id don't go through.",
		},
		Actions: []string{
			"Look at why the  START TRANSACTION / COMMIT / ROLLBACK / SET_TRANSACTION_ISOLATION / SAVEPOINT / ROLLBACK_SAVEPOINT statement failed.",
		},
	},
	E_DUPLICATE_KEY: {
		Code:        17012,
		ErrorCode:   "E_DUPLICATE_KEY",
		Description: "Duplicate Key",
		Causes: []string{
			"Particular key passed in the INSERT statement is already present in the transaction mutations(delta keyspace) or at commit time we found that same key is already present in datastore and commit is unsucessful, and transaction is released.",
		},
		Actions: []string{
			"Ensure you are inserting documents with unique keys",
		},
		IsUser: true,
	},
	E_TRANSACTION_INUSE: {
		Code:        17013,
		ErrorCode:   "E_TRANSACTION_INUSE",
		Description: "Parallel execution of the statements are not allowed within the transaction",
		Causes: []string{
			"Parallel requests are not allowed when using transactions",
		},
		Actions: []string{
			"ensure you are sending your requests in a blocking way(i.e new request only after receiving the response for the previous request).",
		},
		IsUser: true,
	},
	E_KEY_NOT_FOUND: {
		Code:        17014,
		ErrorCode:   "E_KEY_NOT_FOUND",
		Description: "",
		Causes:      []string{},
		Actions:     []string{},
	},
	E_SCAS_MISMATCH: {
		Code:        17015,
		ErrorCode:   "E_SCAS_MISMATCH",
		Description: "",
		Causes:      []string{},
		Actions:     []string{},
	},
	E_TRANSACTION_MEMORY_QUOTA_EXCEEDED: {
		Code:        17016,
		ErrorCode:   "E_TRANSACTION_MEMORY_QUOTA_EXCEEDED",
		Description: "",
		Causes:      []string{},
		Actions:     []string{},
	},
	E_TRANSACTION_FETCH: {
		Code:        17017,
		ErrorCode:   "E_TRANSACTION_FETCH",
		Description: "",
		Causes:      []string{},
		Actions:     []string{},
	},
	E_POST_COMMIT_TRANSACTION: {
		Code:        17018,
		ErrorCode:   "E_POST_COMMIT_TRANSACTION",
		Description: "",
		Causes:      []string{},
		Actions:     []string{},
	},
	E_AMBIGUOUS_COMMIT_TRANSACTION: {
		Code:        17019,
		ErrorCode:   "E_AMBIGUOUS_COMMIT_TRANSACTION",
		Description: "",
		Causes:      []string{},
		Actions:     []string{},
	},
	E_TRANSACTION_STAGING: {
		Code:        17020,
		ErrorCode:   "E_TRANSACTION_STAGING",
		Description: "",
		Causes:      []string{},
		Actions:     []string{},
	},
	E_TRANSACTION_QUEUE_FULL: {
		Code:        17021,
		ErrorCode:   "E_TRANSACTION_QUEUE_FULL",
		Description: "Transaction queue is full",
		Causes: []string{
			"The number of on going requests for a particular transactionid in the queue hit the max limit allowed which is 16.",
		},
		Actions: []string{},
		IsUser:  true,
	},
	W_POST_COMMIT_TRANSACTION: {
		Code:        17022,
		ErrorCode:   "W_POST_COMMIT_TRANSACTION",
		Description: "",
		Causes:      []string{},
		Actions:     []string{},
	},
	E_GC_AGENT: {
		Code:        17096,
		ErrorCode:   "E_GC_AGENT",
		Description: "",
		Causes:      []string{},
		Actions:     []string{},
	},
	E_TRANCE_NOTSUPPORTED: {
		Code:        17097,
		ErrorCode:   "E_TRANCE_NOTSUPPORTED",
		Description: "Transactions are not supported in Community Edition",
		Causes: []string{
			"transcation requires enterprise edition",
		},
		Actions: []string{},
		IsUser:  true,
	},
	E_MEMORY_ALLOCATION: {
		Code:        17098,
		ErrorCode:   "E_MEMORY_ALLOCATION",
		Description: "Memory allocation error",
		Causes: []string{
			"query service uses pools to reduce the load on garbage collector by reusing structures that are accessed by multiple goroutines, but when we go out of memory, or when memory allocation fails this error is raised. The pools maintained are for 1) transaction_mutation, 2) mutation_value_map, 3) savepoints_map_pool, 4) deltakeyspace_map_pool, 5) transactionlogvalue_pool",
		},
		Actions: []string{
			"Consider scaling up the node on which you have query service running. Or contact support.",
		},
		IsUser: false,
	},
	E_TRANSACTION: {
		Code:        17099,
		ErrorCode:   "E_TRANSACTION",
		Description: "",
		Causes:      []string{},
		Actions:     []string{},
	},
	E_SEQUENCE_NOT_ENABLED: {
		Code:        19100,
		ErrorCode:   "E_SEQUENCE_NOT_ENABLED",
		Description: "Sequence support is not enabled for [bucket]",
		Causes: []string{
			"1) Sequences are only enabled for a bucket once the _system scope and _query collection are available for it.",
			"2) CREATE SEQUENCE statement failed, as we failed to get/create _system scope , _query collection.\n2) backup restore endpoint: /api/v1/bucket/{bucket}/backup is when request is to restore Sequences again we look for _system scope but couldn't find it.\n3) DROP SEQUENCE statement failed, as we failed to get _system scope.\n4) ALTER SEQUENCE statement failed, as we failed to get _system scope.\n5) NEXT VALUE FOR <sequence>/ PREV VALUE FOR <sequence> failed as we could not get _system scope.\n6) On exhausing current cache block incase of NEXT VALUE FOR / PREV VALUE FOR, to know the next block we try to read \"block\" from system storage but fail to do so.\n7) When trying to query system:all_sequences keyspace, if a sequence key is not present in cache we go to storage to read it in, but we fail to do so as getting _system scope failed.",
		},
		Actions: []string{
			"Contact Support",
		},
		IsUser: false,
	},
	E_SEQUENCE_CREATE: {
		Code:        19101,
		ErrorCode:   "E_SEQUENCE_CREATE",
		Description: "Create failed for sequence [sequnce name]",
		Causes: []string{
			"1) pre-creation, we validate the scope path for the sequence given by the user ( namespace:bucket.scope.sequencename ) \n    i) system namespace, is not permitted\n    ii) datastore is unset, hence scope validation fails\n    iii) bucket not found, i.e bucket not found in default namespace\n    iv) scope not found, i.e scope is not created",
			"2) Insert operation failed for the storage of the sequence being created, i) could be dml memcached error( cas-mismatch) (unlikely here though), ii) Duplicate Key (unlikely here)",
			"3) CREATE SEQUENCE [IF NOT EXISTS] <name> [IF NOT EXISTS] WITH <options>, \n    We validate options passed to see if they are one of [start cache increment min max cycle], if not we error out on invalid option for create sequence\n     \nCREATE SEQUENCE `default`.`_default`.test3 WITH {\"notmelel\":3};\n{\n    \"requestID\": \"c053a0b2-f9fa-4d04-b096-7920e551ce9e\",\n    \"signature\": null,\n    \"results\": [\n    ],\n    \"errors\": [\n        {\n            \"code\": 19101,\n            \"msg\": \"Create failed for sequence 'default:default._default.test3'\",\n            \"reason\": {\n                \"_level\": \"exception\",\n                \"caller\": \"sequences:1090\",\n                \"cause\": {\n                    \"option\": \"notmelel\"\n                },\n                \"code\": 12038,\n                \"key\": \"datastore.with.invalid_option\",\n                \"message\": \"Invalid option 'notmelel'\"\n            }\n        }\n    ]",
			"4) when reading [start cache increment min max ] options , expected value is an integer but didn't get an integer\n    TBNt: we catch this right from the parser , so technically dead code. Also error doesn't indicate which option has received invalid input which would be more helpful than line,col \n\n     CREATE SEQUENCE `default`.`_default`.test6 START WITH \"6\";\n{\n    \"requestID\": \"e6b06c7e-2c43-4dd4-9f55-2b9097e72a9b\",\n    \"errors\": [\n        {\n            \"code\": 3000,\n            \"column\": 44,\n            \"line\": 1,\n            \"msg\": \"syntax error - invalid option value (near line 1, column 44)\"\n        }\n    ],\n\n   Similarly, cycle option expects boolean, but got non-boolean value.",
			"5) cache options ,expects a positive integer value\n\n CREATE SEQUENCE `default`.`_default`.test6 CACHE -6;\n{\n    \"requestID\": \"83e273ad-418c-473b-bb01-dc31009c5390\",\n    \"signature\": null,\n    \"results\": [\n    ],\n    \"errors\": [\n        {\n            \"code\": 19101,\n            \"msg\": \"Create failed for sequence 'default:default._default.test6'\",\n            \"reason\": {\n                \"_level\": \"exception\",\n                \"caller\": \"sequences:199\",\n                \"code\": 19105,\n                \"key\": \"datastore.sequence.cache\",\n                \"message\": \"Invalid cache value -6\"\n            }\n        }\n    ],",
			"6) When specifying the sequence range, always have max>=min. Else we error out\n\n CREATE SEQUENCE `default`.`_default`.test7 MINVALUE 10 MAXVALUE 1;\n{\n    \"requestID\": \"9dcc43e1-eb3c-4556-a194-950fe82ea51f\",\n    \"signature\": null,\n    \"results\": [\n    ],\n    \"errors\": [\n        {\n            \"code\": 19101,\n            \"msg\": \"Create failed for sequence 'default:default._default.test7'\",\n            \"reason\": {\n                \"_level\": \"exception\",\n                \"caller\": \"sequences:224\",\n                \"code\": 19104,\n                \"key\": \"datastore.sequence.range\",\n                \"message\": \"Invalid range 10 to 1\"\n            }\n        }\n    ]",
		},
		Actions: []string{},
		IsUser:  true,
	},
	E_SEQUENCE_ALTER: {
		Code:        19102,
		ErrorCode:   "E_SEQUENCE_ALTER",
		Description: "Alter failed for sequence [seq-name]",
		Causes: []string{
			"1) ALTER SEQUENCE <name> WITH <options>\n   Validating with options failed, unexpected option other than [restart cache increment min max cycle]. Hence we error out",
			"2) cycle expects boolean, restart- boolean or an integer, cache increment min max - an integer, we raise error if we don't receive expected value type for the option.\n\n ALTER SEQUENCE `default`._default.testseq WITH {\"restart\":\"s\"};\n{\n    \"requestID\": \"ab992c28-3ded-4dab-8faf-047e8094e1ea\",\n    \"signature\": null,\n    \"results\": [\n    ],\n    \"errors\": [\n        {\n            \"code\": 19102,\n            \"msg\": \"Alter failed for sequence 'default:default._default.testseq'\",\n            \"reason\": {\n                \"_level\": \"exception\",\n                \"caller\": \"sequences:1102\",\n                \"cause\": {\n                    \"option\": \"restart\"\n                },\n                \"code\": 12039,\n                \"key\": \"datastore.with.invalid_value\",\n                \"message\": \"Invalid value for 'restart'\"\n            }\n        }\n    ],",
			"3) cache expects a positive integer, else we error out on INVALID_CACHE",
			"4) newly set min and max options, but min>max , error out on INVALID_RANGE",
			"5) newly set min, but now min> max(old) , error out on INVALID_RANGE",
			"6) newly set max , but now min(old) > max, error out on INVALID_RANGE",
			"7) Before going to stoarge to update the sequence document, we check if _system scope is accessible on the bucket path given in the sequence name.\n i) system namespace, is not permitted\n    ii) datastore is unset, hence scope validation fails\n    iii) bucket not found, i.e bucket not found in default namespace\n    iv) scope not found, i.e scope is not created",
			"8) alter logic,\n      fetch the document using sequence key-> but GET op has returned a bad response,\n\n      then write in new option fields too the annotated value,\n      update , store (set with CAS) op failed  , if due to CAS mismatch or SYNC_WRITE_IN_PROGRESS we retry else we error out.",
		},
		Actions: []string{},
		IsUser:  true,
	},
	E_SEQUENCE_DROP: {
		Code:        19103,
		ErrorCode:   "E_SEQUENCE_DROP",
		Description: "Drop failed for sequence [seq-name]",
		Causes: []string{
			"1) DELETE memcached command failed for the sequence document stored under system scope",
		},
		Actions: []string{
			"Possibly external activity has already deleted the sequence document. ",
		},
		IsUser: false,
	},
	E_SEQUENCE_INVALID_RANGE: {
		Code:        19104,
		ErrorCode:   "E_SEQUENCE_INVALID_RANGE",
		Description: "Invalid range [min]-[max]",
		Causes: []string{
			"CREATE / ALTER SEQUENCE has received min and max options , but min>max. Hence we raised this error.",
		},
		Actions: []string{
			"When using with options, \"min\" field & \"max\" field must be passed such that min<=max.\nSimilar case for when using MINVALUE <num> and MAXVALUE <num> construct.",
		},
		IsUser: true,
	},
	E_SEQUENCE_INVALID_CACHE: {
		Code:        19105,
		ErrorCode:   "E_SEQUENCE_INVALID_CACHE",
		Description: "Invalid cache value [cache-option-argument]",
		Causes: []string{
			"CREATE / ALTER SEQUENCE statement has received, negative integer so we terminate the logic for create/alter with the cause as invalid cache.",
		},
		Actions: []string{
			"When using with options, \"cache\" field must have value as a positive integer.\nSimilar case for when using CACHE <num> construct.",
		},
		IsUser: true,
	},
	E_SEQUENCE_NOT_FOUND: {
		Code:        19106,
		ErrorCode:   "E_SEQUENCE_NOT_FOUND",
		Description: "Sequence [sequence name] not found",
		Causes:      []string{},
		Actions:     []string{},
	},
	E_SEQUENCE: {
		Code:        19107,
		ErrorCode:   "E_SEQUENCE",
		Description: "Error accessing sequence",
		Causes: []string{
			"Fail-safe code( as we disallow sequence name with less than 3 elements at the parser)\nWhen we a sequence instance is not found in cache, we load it in from system scope. But there was an error in parsing the path from sequence name(namespace:bucket.scope.seq_name). Ideally would never occur",
		},
		Actions: []string{},
	},
	E_SEQUENCE_ALREADY_EXISTS: {
		Code:        19108,
		ErrorCode:   "E_SEQUENCE_ALREADY_EXISTS",
		Description: "Sequence [seq-name] already exists ",
		Causes: []string{
			"1) CREATE SEQUENCE ( without IF NOT EXISTS ) , if sequence with same name(namespace:bucket.scope.name) already exists we raise this error.",
			"2) when doing backup restore of a bucket, if the sequence from the restoration index already exists, we raise this error.",
		},
		Actions: []string{
			"SELECT * FROM system:all_sequences;\n\nwill tell you all the defined sequence on the cluster, when creating name your new sequence something else:)",
		},
		IsUser: true,
	},
	E_SEQUENCE_METAKV: {
		Code:        19109,
		ErrorCode:   "E_SEQUENCE_METAKV",
		Description: "Error accessing sequences cache monitor data",
		Causes: []string{
			"When an ALTER is issued, the cache revision is updated and monitored by all query nodes using MetaKV.\nSo, on startup we initialize cacherevision kvpair using key as cache-revison path: /query/sequences_cache/, and value as [{node-ip-addr}]+cacherevision\nBut the add failed due to a revmismatch hence log this error as a warning, indicating the sequence cache monitoring rountine couldn't be started as the kv-entry failed to be added.",
		},
		Actions: []string{
			"Find out why Add failed, something is wrong with cbauth?\n1) possible revmismatch, same key(/query/sequences_cache/revision/) already exists",
		},
		IsUser: false,
	},
	E_SEQUENCE_INVALID_DATA: {
		Code:        19110,
		ErrorCode:   "E_SEQUENCE_INVALID_DATA",
		Description: "Invalid sequence data",
		Causes: []string{
			"1) missing data,\n     ALTER SEQUENCE [sequence_name] WITH {\"restart\":4};\n     But [sequence_name] is not defined, i.e no sequence document in _system._query collection has its key as [sequence_name]. Hence the alter fails.\n\n    ALTER SEQUENCE `default`._default.testseq WITH {\"restart\":4};\n{\n    \"requestID\": \"952b80e2-433f-44a9-895f-cead92426846\",\n    \"signature\": null,\n    \"results\": [\n    ],\n    \"errors\": [\n        {\n            \"code\": 19110,\n            \"msg\": \"Invalid sequence data\",\n            \"reason\": \"missing data\"\n        }\n    ],",
			"2) ALTER SEQUENCE `default`._default.test5 WITH {\"restart\":true};\n\nBut externally someone has changed the string value assigned to \"initial\" to something else\nFor eg:\nWhat we expected KV return\n{\n  \"base\": \"4\",\n  \"block\": 2,\n  \"cache\": \"50\",\n  \"cycle\": false,\n  \"increment\": \"1\",\n  \"initial\": \"4\",\n  \"max\": \"9223372036854775807\",\n  \"min\": \"-9223372036854775808\",\n  \"version\": 1\n}\n\nBut we got\n{\n  \"base\": \"4\",\n  \"block\": 2,\n  \"cache\": \"50\",\n  \"cycle\": false,\n  \"increment\": \"1\",\n  \"initial\": 4,\n  \"max\": \"9223372036854775807\",\n  \"min\": \"-9223372036854775808\",\n  \"version\": 1\n}\n\nNotice that \"initial\" field has a non - string value\nHence the parsing logic for restart failed.\n ALTER SEQUENCE `default`._default.test5 WITH {\"restart\":true};\n{\n    \"requestID\": \"1c601dcf-1507-4e16-b903-d8156fcc2da9\",\n    \"signature\": null,\n    \"results\": [\n    ],\n    \"errors\": [\n        {\n            \"code\": 19110,\n            \"msg\": \"Invalid sequence data\",\n            \"reason\": \"{\\\"base\\\":\\\"4\\\",\\\"block\\\":2,\\\"cache\\\":\\\"50\\\",\\\"cycle\\\":false,\\\"increment\\\":\\\"1\\\",\\\"initial\\\":4,\\\"max\\\":\\\"9223372036854775807\\\",\\\"min\\\":\\\"-9223372036854775808\\\",\\\"version\\\":1}\"\n        }\n    ],\n\n\nSimilar error would be raised if the string value assigned to \"initial\" in the document is a string but can't be parsed as an integer.",
			"3) When user wants to ALTER a sequences' increment or cache value.\n    We reseed the sequence with current value as the new base, and set block to 0. After this update either cache or increment  or both to the new user passed value.\n    But will running logic for the new base value ( block*cache*incr + base_old) but we may have failed to read \"block\", \"cache\", \"increment\", \"base\" as the value got from KV is of unexpected type(string) or couldn't be parsed as an integer.\n   \n    For eg:\n    ALTER SEQUENCE `default`._default.test5 WITH {\"cache\":100};\n{\n    \"requestID\": \"39cd1355-efea-48b4-915e-879e8b733009\",\n    \"signature\": null,\n    \"results\": [\n    ],\n    \"errors\": [\n        {\n            \"code\": 19110,\n            \"msg\": \"Invalid sequence data\",\n            \"reason\": \"{\\\"base\\\":\\\"4\\\",\\\"block\\\":2,\\\"cache\\\":\\\"50\\\",\\\"cycle\\\":false,\\\"increment\\\":1,\\\"initial\\\":4,\\\"max\\\":\\\"9223372036854775807\\\",\\\"min\\\":\\\"-9223372036854775808\\\",\\\"version\\\":1}\"\n        }\n    ],\n \nHere we see that \\\"increment\\\":1 is a non-string value. Hence the ALTER fails",
			"4) On ALTER we increment the \"version\" of a sequence\n  But sequence document got from KV has a \"version\" as a non-integer value.\n\nALTER SEQUENCE `default`._default.test5 WITH {\"cache\":100};\n{\n    \"requestID\": \"27541f94-cce4-4336-8b83-b81bab10ce96\",\n    \"signature\": null,\n    \"results\": [\n    ],\n    \"errors\": [\n        {\n            \"code\": 19110,\n            \"msg\": \"Invalid sequence data\",\n            \"reason\": \"{\\\"base\\\":\\\"104\\\",\\\"block\\\":0,\\\"cache\\\":\\\"100\\\",\\\"cycle\\\":false,\\\"increment\\\":\\\"1\\\",\\\"initial\\\":4,\\\"max\\\":\\\"9223372036854775807\\\",\\\"min\\\":\\\"-9223372036854775808\\\",\\\"version\\\":\\\"1\\\"}\"\n        }\n    ],",
			"5) For NEXT VALUE FOR [seq] or PREV VALUE FOR [seq] expression evaluation\n\nif sequence is not available in cache we load it in from KV (_system scope, _query collection)\n\nIn the logic for loading we expect\n\"version\" -> to be a number\n\"cache\" -> to be a string that can be parsed as an integer\n\"base\" -> to be a string that can be parsed as an integer\n\"min\" -> to be a string that can be parsed as an integer\n\"max\" ->  to be a string that can be parsed as an integer \n\"increment\" -> to be a string that can be parsed as an integer \n\"cycle\" -> to be a boolean\n\nBut we didn't get the desired value type for field(s) from KV, hence we raise this error.",
			"6) When issuing NEXT VALUE FOR [seq] expression, (sequence doesn't have a valid cycle range)\n\nfailsafe code\nwe use subdocAPI for getting a next-block when current block is completed for a node, to do so we issue SUBDOC_MULTI_MUTATION command,\nparticularly here we increment the path=\"block\" for the sequence document , if the path('block\") doesn't match the results field name-> something is wrong. That is someone external has changed the sequence document.",
			"7) When issuing NEXT VALUE FOR [seq] expression, (sequence has valid cycle range, i.e min and max are defined with cycle set as true)\n\nfailsafe code\nOn the last remaining increment for a block, we check if we have to cycle or goto the next block\nif we have to cycle, we increment the version signalling to all other nodes that their cache revision is invalid and also set block back to 0. To do this we use  subdocAPI using SUBDOC_MULTI_MUTATION command.\nwe check if path passed \"version\" is same as results field name, if not something is wrong. And we error out.",
		},
		Actions: []string{
			"1) SELECT * FROM system:all_sequences;\nwill tell you all the defined sequence on the cluster, only these can be altered.\n\nInvalid data may come from mismatch type at the document.\nEnsure the sequence document's \n\"base\":  is a string (that can be parsed as an integer)\n\"block\": integer\n\"cache\": is a string (that can be parsed as an integer) \n\"cycle\": boolean\n\"increment\": is a string (that can be parsed as an integer) \n\"initial\": is a string (that can be parsed as an integer) \n\"max\": is a string (that can be parsed as an integer) \n\"min\": is a string (that can be parsed as an integer) \n\"version\": integer",
		},
		IsUser: true,
	},
	E_SEQUENCE_EXHAUSTED: {
		Code:        19111,
		ErrorCode:   "E_SEQUENCE_EXHAUSTED",
		Description: "Sequence [seq-name] has reached its limit",
		Causes: []string{
			"1) non cycle sequence with min and max set , NEXT VALUE FOR [seq] expression evaluation fails when.\n   i) increment<0, currvalue <min : range exhausted\n   ii) increment>0, currvalue > max : range exhausted\n    \n  For eg:\n  CREATE SEQUENCE default._default.testseq5 WITH {\"cache\":10,\"min\":0, \"max\":10};\n   \n   run following statement 10 times:\n   SELECT NEXT VALUE FOR default._default.testseq5;\n   \n  next run:\n   to history file for the shell : /Users/gauravj/.cbq_history \ncbq> SELECT NEXT VALUE FOR default._default.testseq5;\n{\n    \"requestID\": \"c29e7399-02e1-4033-84c8-4840566081a2\",\n    \"signature\": {\n        \"$1\": \"number\"\n    },\n    \"results\": [\n    ],\n    \"errors\": [\n        {\n            \"code\": 5010,\n            \"msg\": \"Error evaluating projection\",\n            \"reason\": {\n                \"_level\": \"exception\",\n                \"caller\": \"sequences:1015\",\n                \"code\": 19111,\n                \"key\": \"datastore.sequence.exhausted\",\n                \"message\": \"Sequence 'default:default._default.testseq5' has reached its limit\"\n            }\n        }\n    ],",
			"2) non cycle sequence , reaches int64 limit\n\nFor eg:\nCREATE SEQUENCE default._default.testseq7 START WITH 9223372036854775808;\n\nNow running next val would exhausts the sequence, as the value will under go int64 wrapping.\nSELECT NEXT VALUE FOR default._default.testseq7;\n{\n    \"requestID\": \"d6f8a9d4-2b16-48db-a840-a860f9f2da3f\",\n    \"signature\": {\n        \"$1\": \"number\"\n    },\n    \"results\": [\n    ],\n    \"errors\": [\n        {\n            \"code\": 5010,\n            \"msg\": \"Error evaluating projection\",\n            \"reason\": {\n                \"_level\": \"exception\",\n                \"caller\": \"sequences:1015\",\n                \"code\": 19111,\n                \"key\": \"datastore.sequence.exhausted\",\n                \"message\": \"Sequence 'default:default._default.testseq7' has reached its limit\"\n            }\n        }\n    ]",
		},
		Actions: []string{
			"1) Alter the sequence , if increment>0, ",
		},
	},
	E_SEQUENCE_CYCLE: {
		Code:        19112,
		ErrorCode:   "E_SEQUENCE_CYCLE",
		Description: "UNUSED - couchbase trinty , Cycle failed for sequence [seq-name]",
		Causes:      []string{},
		Actions:     []string{},
	},
	E_SEQUENCE_INVALID_NAME: {
		Code:        19113,
		ErrorCode:   "E_SEQUENCE_INVALID_NAME",
		Description: "UNUSED - couchbase trinty ( all invalid naming are caught by the parser) , Invalid sequence name [seq-name]",
		Causes: []string{
			"1) when using sequence operations, sequence name must be - namespace:bucket.scope.sequence_name ( 3 part name-> bucket, scope, sequencename)\n    sample incorrect usage:\n   \n     SELECT NEXT VALUE FOR default._default.test2.notallowed;\n{\n    \"requestID\": \"c643f531-42de-4f19-8638-85ce3d8e2a98\",\n    \"errors\": [\n        {\n            \"code\": 3000,\n            \"column\": 8,\n            \"line\": 1,\n            \"msg\": \"Invalid sequence name (near line 1, column 8)\"\n        }\n    ],",
			"2) CREATE SEQUENCE [sequence-name]\n    expects scope in the path for sequence-name\n    sample incorrect usage:\n   \n      CREATE SEQUENCE test;\n{\n    \"requestID\": \"2ba7979b-f887-4752-b8bc-0a0fa1ae66a6\",\n    \"errors\": [\n        {\n            \"code\": 3000,\n            \"column\": 17,\n            \"line\": 1,\n            \"msg\": \"Invalid sequence name (near line 1, column 17)\"\n        }\n    ],",
		},
		Actions: []string{},
	},
	E_SEQUENCE_READ_ONLY_REQ: {
		Code:        19114,
		ErrorCode:   "E_SEQUENCE_READ_ONLY_REQ",
		Description: "Sequences cannot be used in read-only requests",
		Causes: []string{
			"Cannot do sequence operations on GET request\n\nGET /query/service?statement=SELECT%20NEXT%20VALUE%20FOR%20default._default.test4 HTTP/1.1\nAuthorization: Basic QWRtaW5pc3RyYXRvcjpwYXNzd29yZA==\nHost: 127.0.0.1:9499\n\nHTTP/1.1 200 OK\nContent-Length: 488\nContent-Type: application/json; version=7.6.0-N1QL\nDate: Mon, 08 Jan 2024 06:40:13 GMT\n{\n\"requestID\": \"ca3790ea-f2e1-4ae8-a1d5-bb7512addf4b\",\n\"signature\": {\"$1\":\"number\"},\n\"results\": [\n],\n\"errors\": [{\"code\":5010,\"msg\":\"Error evaluating projection\",\"reason\":{\"_level\":\"exception\",\"caller\":\"sequence:111\",\"code\":19114,\"key\":\"datastore.sequence.read_only\",\"message\":\"Sequences cannot be used in read-only requests\"}}],\n\"status\": \"fatal\",\n\"metrics\": {\"elapsedTime\": \"29.807792ms\",\"executionTime\": \"501.542\u00b5s\",\"resultCount\": 0,\"resultSize\": 0,\"serviceLoad\": 2,\"errorCount\": 1}\n}",
		},
		Actions: []string{
			"switch request method to POST \nthe request is no longer readonly",
		},
		IsUser: true,
	},
	W_SEQUENCE_CACHE_SIZE: {
		Code:        19115,
		ErrorCode:   "W_SEQUENCE_CACHE_SIZE",
		Description: "Cache size (CACHE-size passed) below recommended minimum",
		Causes: []string{
			"When setting cache option using CREATE SEQUENCE / ALTER SEQUENCE\n\nThe recommended cache size is above 10, for performance reasons. To avoid cache block validation and allocation frequently.\nNOTE: this error is only a warning.",
		},
		Actions: []string{},
		IsUser:  true,
	},
	E_SEQUENCE_NAME_PARTS: {
		Code:        19116,
		ErrorCode:   "E_SEQUENCE_NAME_PARTS",
		Description: "Sequence name resolves to [sequence-full-name] - check query_context?( [line] [column])",
		Causes: []string{
			"failsafe code\nTo semantically disallow a sequence name not having scope, from being evaluated for it's NEXT VAL/PREV VAL expressions.",
		},
		Actions: []string{},
	},
	E_SEQUENCE_DROP_ALL: {
		Code:        19117,
		ErrorCode:   "E_SEQUENCE_DROP_ALL",
		Description: "Drop failed for sequences [list of sequence names]",
		Causes: []string{
			"When Dropping a scope that has sequences\n\nWe cleanup by doing a refresh(bucket update) to clear old sequence cache ( just as for dictionaries and functions) \nIn the logic to delete the sequence document from the system scope, we get back error response on DELETE command, that is when we raise this error.",
		},
		Actions: []string{},
		IsUser:  false,
	},
	W_SEQUENCE_NO_PREV_VALUE: {
		Code:        19118,
		ErrorCode:   "W_SEQUENCE_NO_PREV_VALUE",
		Description: "Sequence previous value cannot be accessed before next value generation.",
		Causes: []string{
			"A newly CREATED sequence's previous value cannot be accessed without running NEXT VALUE FOR / NEXTVAL FOR expression atleast once.\n\n CREATE SEQUENCE `default`._default.trial2;\n{\n    \"requestID\": \"dfaa93a0-5b20-41b2-9476-8e7cc04f7368\",\n    \"signature\": null,\n    \"results\": [\n    ],\n    \"status\": \"success\",\n    \"metrics\": {\n        \"elapsedTime\": \"19.984166ms\",\n        \"executionTime\": \"19.831208ms\",\n        \"resultCount\": 0,\n        \"resultSize\": 0,\n        \"serviceLoad\": 2\n    }\n}\n\n\nSELECT PREVVAL FOR default._default.trial2;\n{\n    \"requestID\": \"f4b61573-8751-44ca-b24d-d510b5b1adb4\",\n    \"signature\": {\n        \"$1\": \"number\"\n    },\n    \"results\": [\n    {}\n    ],\n    \"warnings\": [\n        {\n            \"code\": 19118,\n            \"msg\": \"Sequence previous value cannot be accessed before next value generation.\"\n        }\n    ],",
		},
		Actions: []string{},
		IsUser:  true,
	},
}
