//line n1ql.y:2
package n1ql

import __yyfmt__ "fmt"

//line n1ql.y:2
import "fmt"
import "github.com/couchbaselabs/clog"
import "github.com/couchbaselabs/query/algebra"
import "github.com/couchbaselabs/query/datastore"
import "github.com/couchbaselabs/query/expression"
import "github.com/couchbaselabs/query/value"

func logDebugGrammar(format string, v ...interface{}) {
	clog.To("PARSER", format, v...)
}

//line n1ql.y:16
type yySymType struct {
	yys int
	s   string
	n   int
	f   float64
	b   bool

	expr      expression.Expression
	exprs     expression.Expressions
	whenTerm  *expression.WhenTerm
	whenTerms expression.WhenTerms
	binding   *expression.Binding
	bindings  expression.Bindings

	node      algebra.Node
	statement algebra.Statement

	fullselect   *algebra.Select
	subresult    algebra.Subresult
	subselect    *algebra.Subselect
	fromTerm     algebra.FromTerm
	keyspaceTerm *algebra.KeyspaceTerm
	path         expression.Path
	group        *algebra.Group
	resultTerm   *algebra.ResultTerm
	resultTerms  algebra.ResultTerms
	projection   *algebra.Projection
	order        *algebra.Order
	sortTerm     *algebra.SortTerm
	sortTerms    algebra.SortTerms

	keyspaceRef *algebra.KeyspaceRef

	set          *algebra.Set
	unset        *algebra.Unset
	setTerm      *algebra.SetTerm
	setTerms     algebra.SetTerms
	unsetTerm    *algebra.UnsetTerm
	unsetTerms   algebra.UnsetTerms
	updateFor    *algebra.UpdateFor
	mergeActions *algebra.MergeActions
	mergeUpdate  *algebra.MergeUpdate
	mergeDelete  *algebra.MergeDelete
	mergeInsert  *algebra.MergeInsert

	createIndex *algebra.CreateIndex
	dropIndex   *algebra.DropIndex
	alterIndex  *algebra.AlterIndex
	indexType   datastore.IndexType
}

const ALL = 57346
const ALTER = 57347
const ANALYZE = 57348
const AND = 57349
const ANY = 57350
const ARRAY = 57351
const AS = 57352
const ASC = 57353
const BEGIN = 57354
const BETWEEN = 57355
const BREAK = 57356
const BUCKET = 57357
const BY = 57358
const CALL = 57359
const CASE = 57360
const CAST = 57361
const CLUSTER = 57362
const COLLATE = 57363
const COLLECTION = 57364
const COMMIT = 57365
const CONNECT = 57366
const CONTINUE = 57367
const CREATE = 57368
const DATABASE = 57369
const DATASET = 57370
const DATASTORE = 57371
const DECLARE = 57372
const DELETE = 57373
const DERIVED = 57374
const DESC = 57375
const DESCRIBE = 57376
const DISTINCT = 57377
const DO = 57378
const DROP = 57379
const EACH = 57380
const ELEMENT = 57381
const ELSE = 57382
const END = 57383
const EVERY = 57384
const EXCEPT = 57385
const EXCLUDE = 57386
const EXECUTE = 57387
const EXISTS = 57388
const EXPLAIN = 57389
const FALSE = 57390
const FIRST = 57391
const FLATTEN = 57392
const FOR = 57393
const FROM = 57394
const FUNCTION = 57395
const GRANT = 57396
const GROUP = 57397
const HAVING = 57398
const IF = 57399
const IN = 57400
const INCLUDE = 57401
const INDEX = 57402
const INLINE = 57403
const INNER = 57404
const INSERT = 57405
const INTERSECT = 57406
const INTO = 57407
const IS = 57408
const JOIN = 57409
const KEY = 57410
const KEYS = 57411
const KEYSPACE = 57412
const LAST = 57413
const LEFT = 57414
const LET = 57415
const LETTING = 57416
const LIKE = 57417
const LIMIT = 57418
const MAP = 57419
const MAPPING = 57420
const MATCHED = 57421
const MATERIALIZED = 57422
const MERGE = 57423
const MINUS = 57424
const MISSING = 57425
const NAMESPACE = 57426
const NEST = 57427
const NOT = 57428
const NULL = 57429
const OBJECT = 57430
const OFFSET = 57431
const ON = 57432
const OPTION = 57433
const OR = 57434
const ORDER = 57435
const OUTER = 57436
const OVER = 57437
const PARTITION = 57438
const PASSWORD = 57439
const PATH = 57440
const POOL = 57441
const PREPARE = 57442
const PRIMARY = 57443
const PRIVATE = 57444
const PRIVILEGE = 57445
const PROCEDURE = 57446
const PUBLIC = 57447
const RAW = 57448
const REALM = 57449
const REDUCE = 57450
const RENAME = 57451
const RETURN = 57452
const RETURNING = 57453
const REVOKE = 57454
const RIGHT = 57455
const ROLE = 57456
const ROLLBACK = 57457
const SATISFIES = 57458
const SCHEMA = 57459
const SELECT = 57460
const SET = 57461
const SHOW = 57462
const SOME = 57463
const START = 57464
const STATISTICS = 57465
const SYSTEM = 57466
const THEN = 57467
const TO = 57468
const TRANSACTION = 57469
const TRIGGER = 57470
const TRUE = 57471
const TRUNCATE = 57472
const TYPE = 57473
const UNDER = 57474
const UNION = 57475
const UNIQUE = 57476
const UNNEST = 57477
const UNSET = 57478
const UPDATE = 57479
const UPSERT = 57480
const USE = 57481
const USER = 57482
const USING = 57483
const VALUE = 57484
const VALUED = 57485
const VALUES = 57486
const VIEW = 57487
const WHEN = 57488
const WHERE = 57489
const WHILE = 57490
const WITH = 57491
const WITHIN = 57492
const WORK = 57493
const XOR = 57494
const INT = 57495
const NUMBER = 57496
const STRING = 57497
const IDENTIFIER = 57498
const IDENTIFIER_ICASE = 57499
const LPAREN = 57500
const RPAREN = 57501
const LBRACE = 57502
const RBRACE = 57503
const LBRACKET = 57504
const RBRACKET = 57505
const RBRACKET_ICASE = 57506
const COMMA = 57507
const COLON = 57508
const INTERESECT = 57509
const EQ = 57510
const DEQ = 57511
const NE = 57512
const LT = 57513
const GT = 57514
const LE = 57515
const GE = 57516
const CONCAT = 57517
const PLUS = 57518
const STAR = 57519
const DIV = 57520
const MOD = 57521
const UMINUS = 57522
const DOT = 57523

var yyToknames = []string{
	"ALL",
	"ALTER",
	"ANALYZE",
	"AND",
	"ANY",
	"ARRAY",
	"AS",
	"ASC",
	"BEGIN",
	"BETWEEN",
	"BREAK",
	"BUCKET",
	"BY",
	"CALL",
	"CASE",
	"CAST",
	"CLUSTER",
	"COLLATE",
	"COLLECTION",
	"COMMIT",
	"CONNECT",
	"CONTINUE",
	"CREATE",
	"DATABASE",
	"DATASET",
	"DATASTORE",
	"DECLARE",
	"DELETE",
	"DERIVED",
	"DESC",
	"DESCRIBE",
	"DISTINCT",
	"DO",
	"DROP",
	"EACH",
	"ELEMENT",
	"ELSE",
	"END",
	"EVERY",
	"EXCEPT",
	"EXCLUDE",
	"EXECUTE",
	"EXISTS",
	"EXPLAIN",
	"FALSE",
	"FIRST",
	"FLATTEN",
	"FOR",
	"FROM",
	"FUNCTION",
	"GRANT",
	"GROUP",
	"HAVING",
	"IF",
	"IN",
	"INCLUDE",
	"INDEX",
	"INLINE",
	"INNER",
	"INSERT",
	"INTERSECT",
	"INTO",
	"IS",
	"JOIN",
	"KEY",
	"KEYS",
	"KEYSPACE",
	"LAST",
	"LEFT",
	"LET",
	"LETTING",
	"LIKE",
	"LIMIT",
	"MAP",
	"MAPPING",
	"MATCHED",
	"MATERIALIZED",
	"MERGE",
	"MINUS",
	"MISSING",
	"NAMESPACE",
	"NEST",
	"NOT",
	"NULL",
	"OBJECT",
	"OFFSET",
	"ON",
	"OPTION",
	"OR",
	"ORDER",
	"OUTER",
	"OVER",
	"PARTITION",
	"PASSWORD",
	"PATH",
	"POOL",
	"PREPARE",
	"PRIMARY",
	"PRIVATE",
	"PRIVILEGE",
	"PROCEDURE",
	"PUBLIC",
	"RAW",
	"REALM",
	"REDUCE",
	"RENAME",
	"RETURN",
	"RETURNING",
	"REVOKE",
	"RIGHT",
	"ROLE",
	"ROLLBACK",
	"SATISFIES",
	"SCHEMA",
	"SELECT",
	"SET",
	"SHOW",
	"SOME",
	"START",
	"STATISTICS",
	"SYSTEM",
	"THEN",
	"TO",
	"TRANSACTION",
	"TRIGGER",
	"TRUE",
	"TRUNCATE",
	"TYPE",
	"UNDER",
	"UNION",
	"UNIQUE",
	"UNNEST",
	"UNSET",
	"UPDATE",
	"UPSERT",
	"USE",
	"USER",
	"USING",
	"VALUE",
	"VALUED",
	"VALUES",
	"VIEW",
	"WHEN",
	"WHERE",
	"WHILE",
	"WITH",
	"WITHIN",
	"WORK",
	"XOR",
	"INT",
	"NUMBER",
	"STRING",
	"IDENTIFIER",
	"IDENTIFIER_ICASE",
	"LPAREN",
	"RPAREN",
	"LBRACE",
	"RBRACE",
	"LBRACKET",
	"RBRACKET",
	"RBRACKET_ICASE",
	"COMMA",
	"COLON",
	"INTERESECT",
	"EQ",
	"DEQ",
	"NE",
	"LT",
	"GT",
	"LE",
	"GE",
	"CONCAT",
	"PLUS",
	"STAR",
	"DIV",
	"MOD",
	"UMINUS",
	"DOT",
}
var yyStatenames = []string{}

const yyEofCode = 1
const yyErrCode = 2
const yyMaxDepth = 200

//line yacctab:1
var yyExca = []int{
	-1, 1,
	1, -1,
	-2, 0,
	-1, 20,
	158, 276,
	-2, 227,
	-1, 104,
	166, 63,
	-2, 64,
	-1, 140,
	50, 72,
	67, 72,
	85, 72,
	135, 72,
	-2, 50,
	-1, 167,
	168, 0,
	169, 0,
	170, 0,
	-2, 204,
	-1, 168,
	168, 0,
	169, 0,
	170, 0,
	-2, 205,
	-1, 169,
	168, 0,
	169, 0,
	170, 0,
	-2, 206,
	-1, 170,
	171, 0,
	172, 0,
	173, 0,
	174, 0,
	-2, 207,
	-1, 171,
	171, 0,
	172, 0,
	173, 0,
	174, 0,
	-2, 208,
	-1, 172,
	171, 0,
	172, 0,
	173, 0,
	174, 0,
	-2, 209,
	-1, 173,
	171, 0,
	172, 0,
	173, 0,
	174, 0,
	-2, 210,
	-1, 180,
	75, 0,
	-2, 213,
	-1, 181,
	58, 0,
	150, 0,
	-2, 215,
	-1, 182,
	58, 0,
	150, 0,
	-2, 217,
	-1, 275,
	75, 0,
	-2, 214,
	-1, 276,
	58, 0,
	150, 0,
	-2, 216,
	-1, 277,
	58, 0,
	150, 0,
	-2, 218,
}

const yyNprod = 292
const yyPrivate = 57344

var yyTokenNames []string
var yyStates []string

const yyLast = 2625

var yyAct = []int{

	154, 3, 566, 554, 427, 564, 555, 296, 297, 187,
	88, 89, 504, 513, 292, 204, 386, 521, 300, 206,
	129, 480, 244, 91, 441, 251, 205, 221, 388, 402,
	289, 200, 410, 146, 385, 149, 125, 245, 330, 374,
	12, 223, 150, 141, 128, 110, 127, 102, 114, 252,
	62, 122, 317, 48, 126, 216, 8, 269, 133, 134,
	315, 335, 443, 457, 418, 267, 456, 158, 159, 160,
	161, 162, 163, 164, 165, 166, 167, 168, 169, 170,
	171, 172, 173, 417, 266, 180, 181, 182, 115, 507,
	418, 66, 87, 403, 488, 231, 439, 316, 403, 267,
	254, 253, 131, 132, 66, 229, 234, 126, 68, 417,
	65, 143, 501, 218, 270, 271, 272, 233, 266, 69,
	70, 71, 203, 65, 155, 156, 334, 255, 440, 269,
	157, 438, 369, 227, 231, 228, 230, 267, 483, 175,
	144, 308, 241, 306, 103, 351, 82, 190, 192, 194,
	259, 268, 270, 271, 272, 233, 266, 246, 262, 459,
	460, 226, 357, 358, 219, 461, 225, 225, 359, 123,
	261, 448, 104, 418, 393, 106, 243, 68, 275, 276,
	277, 256, 258, 257, 346, 231, 130, 207, 66, 413,
	284, 85, 417, 208, 155, 156, 222, 290, 303, 87,
	157, 72, 67, 69, 70, 71, 144, 65, 142, 267,
	104, 558, 307, 294, 502, 68, 310, 565, 311, 560,
	174, 104, 273, 268, 270, 271, 272, 304, 266, 235,
	319, 295, 320, 175, 505, 323, 324, 325, 104, 299,
	367, 217, 548, 305, 333, 285, 202, 286, 135, 287,
	243, 544, 485, 579, 336, 578, 574, 66, 350, 545,
	535, 64, 429, 224, 224, 355, 450, 344, 360, 345,
	309, 67, 69, 70, 71, 301, 65, 279, 322, 176,
	318, 278, 341, 86, 368, 529, 328, 329, 514, 343,
	232, 195, 63, 527, 377, 66, 124, 63, 349, 337,
	445, 117, 380, 382, 383, 381, 503, 111, 72, 67,
	69, 70, 71, 396, 65, 269, 389, 338, 391, 314,
	313, 236, 175, 283, 178, 175, 175, 175, 175, 175,
	175, 569, 375, 208, 378, 379, 408, 280, 570, 63,
	415, 177, 116, 399, 525, 401, 572, 215, 302, 392,
	576, 526, 575, 143, 246, 397, 398, 543, 64, 404,
	422, 225, 225, 64, 536, 298, 189, 340, 249, 137,
	290, 414, 407, 419, 420, 409, 416, 431, 250, 540,
	430, 405, 299, 432, 433, 412, 412, 247, 435, 376,
	434, 444, 436, 437, 95, 267, 447, 274, 347, 348,
	426, 390, 105, 210, 452, 64, 293, 126, 273, 268,
	270, 271, 272, 99, 266, 94, 179, 185, 214, 462,
	184, 183, 237, 238, 175, 468, 98, 582, 581, 556,
	220, 458, 119, 118, 446, 463, 464, 455, 101, 472,
	477, 474, 475, 454, 97, 473, 511, 332, 63, 126,
	142, 265, 193, 191, 100, 61, 519, 389, 224, 224,
	482, 400, 492, 470, 481, 471, 453, 451, 327, 478,
	326, 489, 497, 476, 321, 213, 577, 186, 498, 539,
	406, 196, 411, 411, 93, 484, 356, 2, 342, 361,
	362, 363, 364, 365, 366, 494, 495, 339, 1, 90,
	63, 63, 499, 449, 547, 528, 551, 559, 246, 500,
	506, 512, 530, 508, 524, 442, 515, 516, 139, 522,
	522, 523, 481, 520, 387, 384, 269, 479, 73, 534,
	428, 532, 533, 531, 82, 538, 469, 197, 198, 199,
	549, 550, 537, 291, 209, 34, 541, 542, 33, 32,
	546, 552, 553, 18, 17, 16, 557, 567, 73, 561,
	563, 562, 568, 15, 82, 14, 64, 64, 13, 571,
	7, 6, 5, 4, 573, 370, 371, 281, 282, 85,
	188, 580, 567, 567, 584, 585, 583, 87, 425, 288,
	92, 73, 96, 145, 207, 510, 84, 82, 509, 487,
	486, 490, 491, 68, 331, 242, 267, 83, 136, 85,
	201, 248, 138, 74, 140, 59, 60, 87, 26, 273,
	268, 270, 271, 272, 113, 266, 84, 25, 43, 21,
	46, 45, 24, 68, 109, 108, 107, 83, 23, 120,
	121, 42, 85, 74, 41, 19, 0, 0, 0, 0,
	87, 0, 0, 0, 0, 0, 0, 0, 0, 84,
	0, 0, 0, 0, 0, 0, 68, 0, 0, 0,
	83, 86, 0, 0, 0, 0, 74, 0, 0, 0,
	0, 0, 0, 66, 517, 518, 0, 0, 0, 75,
	76, 77, 78, 79, 80, 81, 72, 67, 69, 70,
	71, 86, 65, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 66, 465, 466, 0, 0, 0, 75,
	76, 77, 78, 79, 80, 81, 72, 67, 69, 70,
	71, 73, 65, 0, 86, 0, 0, 82, 0, 0,
	208, 0, 0, 0, 0, 0, 66, 51, 0, 0,
	0, 0, 75, 76, 77, 78, 79, 80, 81, 72,
	67, 69, 70, 71, 0, 65, 73, 0, 49, 0,
	372, 0, 82, 29, 0, 0, 0, 0, 0, 50,
	0, 0, 85, 0, 0, 0, 0, 0, 0, 11,
	87, 0, 373, 0, 63, 0, 73, 0, 0, 84,
	0, 0, 82, 0, 0, 27, 68, 0, 0, 0,
	83, 0, 0, 0, 0, 0, 74, 85, 0, 0,
	0, 0, 0, 31, 0, 87, 0, 0, 0, 0,
	0, 0, 0, 0, 84, 0, 0, 0, 0, 0,
	0, 68, 0, 0, 0, 83, 0, 85, 0, 0,
	0, 74, 0, 0, 0, 87, 0, 0, 0, 0,
	64, 0, 0, 0, 84, 0, 0, 0, 0, 0,
	0, 68, 0, 0, 86, 83, 0, 0, 0, 30,
	28, 74, 0, 0, 0, 0, 66, 423, 0, 0,
	424, 0, 75, 76, 77, 78, 79, 80, 81, 72,
	67, 69, 70, 71, 0, 65, 0, 0, 0, 86,
	0, 0, 0, 0, 0, 0, 0, 0, 73, 0,
	0, 66, 0, 0, 82, 0, 0, 75, 76, 77,
	78, 79, 80, 81, 72, 67, 69, 70, 71, 86,
	65, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 66, 352, 353, 0, 0, 0, 75, 76, 77,
	78, 79, 80, 81, 72, 67, 69, 70, 71, 85,
	65, 0, 73, 0, 0, 207, 0, 87, 82, 0,
	0, 0, 0, 0, 0, 0, 84, 0, 0, 0,
	0, 0, 0, 68, 0, 0, 0, 83, 0, 0,
	0, 0, 0, 74, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 85, 0, 0, 0, 0, 0, 0,
	0, 87, 0, 0, 0, 0, 0, 0, 0, 0,
	84, 0, 0, 73, 0, 0, 0, 68, 0, 82,
	0, 83, 0, 0, 0, 0, 0, 74, 0, 0,
	0, 86, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 66, 263, 0, 0, 264, 0, 75,
	76, 77, 78, 79, 80, 81, 72, 67, 69, 70,
	71, 0, 65, 0, 85, 0, 0, 0, 0, 0,
	0, 0, 87, 73, 0, 0, 0, 0, 0, 82,
	0, 84, 0, 0, 0, 86, 0, 0, 68, 0,
	0, 208, 83, 0, 0, 0, 0, 66, 74, 0,
	0, 0, 0, 75, 76, 77, 78, 79, 80, 81,
	72, 67, 69, 70, 71, 0, 260, 443, 0, 0,
	0, 0, 0, 0, 85, 0, 0, 0, 0, 0,
	0, 0, 87, 0, 0, 0, 0, 0, 73, 0,
	0, 84, 0, 0, 82, 0, 0, 0, 68, 0,
	0, 0, 83, 243, 0, 0, 86, 0, 74, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 66, 0,
	0, 0, 0, 0, 75, 76, 77, 78, 79, 80,
	81, 72, 67, 69, 70, 71, 0, 65, 0, 85,
	0, 0, 0, 0, 0, 73, 0, 87, 0, 0,
	0, 82, 0, 0, 0, 0, 84, 0, 0, 0,
	0, 0, 0, 68, 0, 0, 86, 83, 0, 0,
	0, 0, 0, 74, 0, 0, 0, 0, 66, 0,
	0, 0, 0, 0, 75, 76, 77, 78, 79, 80,
	81, 72, 67, 69, 70, 71, 85, 65, 0, 0,
	0, 0, 0, 0, 87, 0, 0, 0, 0, 0,
	73, 0, 0, 84, 0, 0, 82, 0, 0, 0,
	68, 0, 0, 0, 83, 0, 0, 0, 0, 0,
	74, 86, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 66, 496, 0, 0, 0, 0, 75,
	76, 77, 78, 79, 80, 81, 72, 67, 69, 70,
	71, 85, 65, 0, 0, 0, 0, 73, 0, 87,
	0, 0, 0, 82, 0, 0, 0, 0, 84, 0,
	0, 0, 0, 0, 0, 68, 0, 0, 86, 83,
	0, 0, 0, 0, 0, 74, 0, 0, 0, 0,
	66, 493, 0, 0, 0, 0, 75, 76, 77, 78,
	79, 80, 81, 72, 67, 69, 70, 71, 85, 65,
	0, 0, 0, 0, 0, 0, 87, 0, 0, 0,
	0, 0, 73, 0, 0, 84, 0, 0, 82, 0,
	0, 0, 68, 0, 0, 0, 83, 0, 0, 0,
	0, 0, 74, 86, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 66, 421, 0, 0, 0,
	0, 75, 76, 77, 78, 79, 80, 81, 72, 67,
	69, 70, 71, 85, 65, 395, 0, 0, 0, 73,
	0, 87, 0, 0, 0, 82, 0, 0, 0, 0,
	84, 0, 0, 0, 0, 0, 0, 68, 0, 0,
	86, 83, 0, 0, 0, 0, 0, 74, 0, 0,
	0, 0, 66, 0, 0, 0, 0, 0, 75, 76,
	77, 78, 79, 80, 81, 72, 67, 69, 70, 71,
	85, 65, 0, 0, 0, 0, 0, 0, 87, 0,
	0, 0, 0, 0, 0, 0, 0, 84, 0, 0,
	0, 73, 0, 0, 68, 0, 0, 82, 83, 0,
	0, 0, 0, 0, 74, 86, 0, 0, 0, 0,
	0, 0, 0, 0, 394, 0, 0, 66, 0, 0,
	0, 0, 0, 75, 76, 77, 78, 79, 80, 81,
	72, 67, 69, 70, 71, 240, 65, 312, 0, 0,
	0, 0, 85, 0, 0, 0, 0, 0, 73, 0,
	87, 0, 0, 0, 82, 0, 0, 0, 0, 84,
	0, 0, 86, 0, 0, 0, 68, 0, 0, 0,
	83, 0, 0, 0, 66, 0, 74, 0, 0, 0,
	75, 76, 77, 78, 79, 80, 81, 72, 67, 69,
	70, 71, 239, 65, 0, 0, 0, 0, 0, 85,
	0, 0, 0, 0, 0, 0, 0, 87, 0, 0,
	0, 0, 0, 73, 0, 0, 84, 0, 0, 82,
	0, 0, 0, 68, 0, 0, 0, 83, 0, 0,
	0, 0, 0, 74, 86, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 66, 0, 0, 0,
	0, 0, 75, 76, 77, 78, 79, 80, 81, 72,
	67, 69, 70, 71, 85, 65, 0, 0, 0, 0,
	73, 0, 87, 0, 0, 0, 82, 0, 0, 0,
	0, 84, 0, 0, 0, 0, 0, 0, 68, 0,
	0, 86, 83, 0, 0, 0, 0, 0, 74, 0,
	0, 0, 0, 66, 0, 0, 0, 0, 0, 75,
	76, 77, 78, 79, 80, 81, 72, 67, 69, 70,
	71, 85, 65, 0, 0, 0, 0, 0, 0, 87,
	0, 0, 0, 0, 0, 73, 0, 0, 84, 0,
	0, 82, 0, 0, 0, 68, 0, 0, 0, 83,
	0, 0, 112, 0, 0, 74, 86, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 66, 0,
	0, 0, 0, 0, 75, 76, 77, 78, 79, 80,
	81, 72, 67, 69, 70, 71, 85, 65, 0, 0,
	0, 0, 0, 0, 87, 0, 0, 0, 82, 0,
	0, 0, 0, 84, 0, 0, 0, 0, 0, 0,
	68, 0, 0, 86, 83, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 66, 0, 0, 0, 0,
	0, 75, 76, 77, 78, 79, 80, 81, 72, 67,
	69, 70, 71, 85, 65, 0, 0, 0, 0, 0,
	148, 87, 0, 0, 54, 57, 0, 0, 0, 0,
	84, 0, 0, 0, 44, 0, 0, 68, 0, 0,
	0, 83, 0, 0, 0, 0, 0, 0, 86, 0,
	0, 147, 0, 0, 0, 152, 0, 0, 56, 0,
	66, 0, 10, 0, 36, 58, 75, 76, 77, 78,
	79, 80, 81, 72, 67, 69, 70, 71, 0, 65,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 22, 0,
	0, 0, 9, 35, 0, 86, 0, 0, 54, 57,
	0, 0, 0, 0, 0, 0, 0, 66, 44, 0,
	0, 0, 151, 75, 76, 77, 78, 79, 80, 81,
	72, 67, 69, 70, 71, 82, 65, 55, 0, 152,
	0, 0, 56, 0, 0, 37, 10, 0, 36, 58,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 54, 57, 39,
	38, 40, 20, 0, 47, 0, 52, 44, 53, 0,
	85, 0, 22, 0, 0, 0, 9, 35, 87, 0,
	82, 0, 0, 153, 211, 0, 0, 84, 0, 0,
	0, 56, 0, 0, 68, 10, 151, 36, 58, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 55, 0, 0, 0, 0, 0, 0, 0, 37,
	54, 57, 0, 0, 0, 85, 0, 0, 0, 0,
	44, 22, 0, 87, 0, 9, 35, 0, 0, 0,
	0, 0, 84, 39, 38, 40, 20, 0, 47, 68,
	52, 0, 53, 0, 56, 0, 0, 0, 10, 0,
	36, 58, 86, 0, 0, 0, 0, 153, 0, 0,
	55, 0, 0, 0, 66, 0, 0, 0, 37, 0,
	0, 0, 0, 78, 79, 80, 81, 72, 67, 69,
	70, 71, 0, 65, 22, 0, 0, 0, 9, 35,
	0, 0, 39, 38, 40, 20, 0, 47, 0, 52,
	0, 53, 0, 0, 0, 0, 0, 86, 0, 0,
	0, 0, 0, 0, 0, 0, 212, 0, 0, 66,
	0, 0, 0, 55, 0, 0, 0, 0, 0, 54,
	57, 37, 72, 67, 69, 70, 71, 0, 65, 44,
	51, 0, 0, 54, 57, 0, 0, 0, 0, 0,
	0, 0, 0, 44, 0, 39, 38, 40, 20, 0,
	47, 49, 52, 56, 53, 0, 29, 10, 0, 36,
	58, 0, 50, 0, 0, 0, 0, 56, 0, 153,
	0, 10, 11, 36, 58, 0, 0, 63, 0, 54,
	57, 0, 0, 0, 0, 0, 0, 0, 27, 44,
	0, 0, 0, 22, 0, 0, 0, 9, 35, 0,
	0, 0, 0, 0, 0, 0, 31, 22, 0, 0,
	0, 9, 35, 56, 0, 0, 0, 10, 0, 36,
	58, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 55, 0, 0, 0, 0, 0, 0, 0,
	37, 0, 0, 64, 0, 0, 55, 0, 0, 0,
	0, 0, 0, 22, 37, 0, 0, 9, 35, 0,
	0, 0, 30, 28, 39, 38, 40, 20, 0, 47,
	0, 52, 0, 53, 467, 0, 54, 57, 39, 38,
	40, 20, 0, 47, 0, 52, 44, 53, 0, 0,
	0, 0, 55, 0, 0, 0, 0, 0, 54, 57,
	37, 0, 0, 0, 0, 0, 0, 0, 44, 0,
	56, 0, 0, 0, 10, 0, 36, 58, 0, 0,
	63, 0, 0, 0, 39, 38, 40, 20, 0, 47,
	0, 52, 56, 53, 354, 0, 10, 0, 36, 58,
	0, 54, 57, 0, 0, 0, 0, 0, 0, 0,
	22, 44, 0, 0, 9, 35, 0, 0, 0, 0,
	54, 57, 0, 0, 0, 0, 0, 0, 0, 0,
	44, 0, 22, 0, 0, 56, 9, 35, 0, 0,
	0, 36, 58, 0, 0, 0, 64, 0, 0, 55,
	0, 0, 0, 0, 56, 0, 0, 37, 10, 0,
	36, 58, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 55, 0, 0, 0, 22, 0, 0, 0, 37,
	35, 39, 38, 40, 20, 0, 47, 0, 52, 0,
	53, 0, 0, 0, 22, 0, 112, 0, 9, 35,
	0, 0, 0, 39, 38, 40, 20, 0, 47, 0,
	52, 0, 53, 0, 55, 0, 0, 0, 0, 0,
	0, 0, 37, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 55, 0, 0, 0, 0, 0, 0,
	0, 37, 0, 0, 0, 0, 39, 38, 40, 20,
	0, 47, 0, 52, 0, 53, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 39, 38, 40, 20, 0,
	47, 0, 52, 0, 53,
}
var yyPact = []int{

	2235, -1000, -1000, 1713, -1000, -1000, -1000, -1000, -1000, 2462,
	2462, 742, -1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000,
	-1000, -1000, 2462, -1000, -1000, -1000, 351, 361, 348, 402,
	54, 337, -1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000,
	-1000, -1000, -1000, 17, 2400, -1000, -1000, 2378, -1000, 241,
	373, 372, 14, 2462, 30, 30, 30, 2462, 2462, -1000,
	-1000, 296, 396, 82, 1896, 38, 2462, 2462, 2462, 2462,
	2462, 2462, 2462, 2462, 2462, 2462, 2462, 2462, 2462, 2462,
	2462, 2462, 2443, 266, 2462, 2462, 2462, 334, 1835, 26,
	-1000, -71, 290, 449, 448, 287, -1000, 465, 54, 54,
	54, 107, -44, 177, -1000, 54, 2039, 434, -1000, -1000,
	1656, 201, 2462, 5, 1713, -1000, 370, 40, 54, 54,
	-28, -30, -1000, -61, -27, -31, 1713, -10, -1000, 171,
	-1000, -10, -10, 1591, 1534, 29, -1000, 37, 296, -1000,
	306, -1000, -132, -65, -66, -1000, -38, 1980, 2102, 2462,
	-1000, -1000, -1000, -1000, 965, -1000, -1000, 2462, 911, -58,
	-58, -71, -71, -71, 95, 1835, 1778, 2002, 2002, 2002,
	2057, 2057, 2057, 2057, 444, -1000, 2443, 2462, 2462, 2462,
	133, 26, 26, -1000, 194, -1000, -1000, 234, -1000, 2462,
	-1000, 240, -1000, 240, -1000, 240, 2462, 338, 338, 107,
	246, -1000, 174, 42, -1000, -1000, -1000, 37, -1000, 102,
	-16, 2462, -18, -1000, 201, 2462, -1000, 2462, 1462, -1000,
	230, 229, -1000, -121, -1000, -69, -129, -1000, 14, 2462,
	-1000, 2462, 433, 30, 2462, 2462, 2462, 429, 427, 30,
	30, 392, -1000, 2462, -39, -1000, -107, 29, 232, -1000,
	195, 177, 28, 42, 42, 2102, -38, 2462, -38, 1713,
	-32, -1000, 789, -1000, 2281, 2443, 6, 2462, 2443, 2443,
	2443, 2443, 2443, 2443, 233, 133, 26, 26, -1000, -1000,
	-1000, -1000, -1000, 2462, 1713, -1000, -1000, -1000, -33, -1000,
	759, 245, -1000, 2462, 245, 29, 103, 29, 28, 28,
	332, -1000, 177, -1000, -1000, 16, -1000, 1405, -1000, -1000,
	1340, 1713, 2462, 54, 54, 40, 42, 40, -1000, 1713,
	1713, -1000, -1000, 1713, 1713, 1713, -1000, -1000, -48, -48,
	143, -1000, 464, 1713, 37, 2462, 392, 65, 65, 2462,
	-1000, -1000, -1000, -1000, 107, -98, -1000, -132, -132, -1000,
	1713, -1000, -1000, -1000, -1000, 1283, 47, -1000, -1000, 2462,
	724, -63, -63, -97, -97, -97, -25, 2443, 1713, 2462,
	-1000, -1000, -1000, -1000, 151, 151, 2462, 1713, 151, 151,
	290, 29, 290, 290, -34, -1000, -72, -37, -1000, 11,
	2462, -1000, 210, 240, -1000, 2462, 1713, -1000, 13, -1000,
	-1000, 157, 426, 2462, 425, -1000, 2462, -1000, 1713, -1000,
	-1000, -132, -100, -103, -1000, 584, -1000, 3, 2462, 177,
	177, -1000, 551, -1000, 2221, 47, -1000, -1000, -1000, 1980,
	-1000, 1713, -1000, -1000, 151, 290, 151, 151, 28, 2462,
	28, -1000, -1000, 30, 1713, 338, -21, 1713, 2462, -1000,
	126, -1000, 1713, -1000, 20, 177, 42, 42, -1000, -1000,
	-1000, 2462, 1218, 107, 107, -1000, -1000, -1000, 1161, -1000,
	-38, 2462, -1000, 151, -1000, -1000, -1000, 1096, -1000, -53,
	-1000, 156, 88, 177, -70, 40, 390, -1000, 37, 198,
	-132, -132, 521, -1000, -1000, -1000, -1000, 1713, -1000, -1000,
	415, 30, 28, 28, 290, 265, 203, 189, -1000, -1000,
	-1000, 2462, -39, -1000, 174, 177, 177, -1000, -1000, -1000,
	-1000, -1000, -98, -1000, 151, 135, 285, 338, 29, 463,
	1713, 310, 198, 198, -1000, 220, 134, 88, 101, 2462,
	2462, -1000, -1000, 246, 29, 366, 290, -1000, 66, 1713,
	1713, 73, 103, 29, 71, -1000, 2462, 151, -1000, -1000,
	252, -1000, 29, -1000, -1000, 260, -1000, 1036, -1000, 131,
	273, -1000, 271, -1000, 445, 130, 128, 29, 365, 364,
	71, 2462, 2462, -1000, -1000, -1000,
}
var yyPgo = []int{

	0, 645, 644, 641, 51, 640, 639, 0, 56, 220,
	36, 296, 37, 22, 19, 26, 15, 20, 638, 636,
	635, 634, 55, 307, 632, 631, 630, 44, 46, 290,
	29, 629, 628, 627, 624, 40, 618, 53, 616, 615,
	614, 455, 612, 43, 32, 611, 16, 25, 47, 144,
	610, 31, 13, 248, 608, 6, 605, 38, 604, 600,
	599, 598, 595, 42, 33, 593, 50, 592, 590, 30,
	589, 580, 9, 578, 577, 576, 575, 487, 573, 572,
	571, 570, 568, 565, 563, 555, 554, 553, 549, 548,
	545, 438, 39, 14, 543, 536, 530, 4, 21, 527,
	17, 7, 34, 525, 8, 28, 524, 515, 24, 12,
	507, 506, 3, 2, 5, 27, 41, 505, 504, 503,
	498, 35, 497, 18, 488,
}
var yyR1 = []int{

	0, 120, 120, 77, 77, 77, 77, 78, 79, 80,
	80, 80, 80, 80, 81, 87, 87, 87, 35, 36,
	36, 36, 36, 36, 36, 36, 37, 37, 39, 38,
	66, 65, 65, 65, 65, 65, 121, 121, 64, 64,
	63, 63, 63, 16, 16, 15, 15, 14, 42, 42,
	41, 40, 40, 40, 40, 122, 122, 43, 43, 43,
	44, 44, 44, 48, 49, 47, 47, 51, 51, 50,
	123, 123, 45, 45, 45, 124, 124, 52, 53, 53,
	54, 13, 13, 12, 55, 55, 56, 57, 57, 58,
	10, 10, 59, 59, 60, 61, 61, 62, 68, 68,
	67, 70, 70, 69, 76, 76, 75, 75, 72, 72,
	71, 74, 74, 73, 82, 82, 91, 91, 94, 94,
	93, 92, 97, 97, 96, 95, 95, 83, 83, 84,
	85, 85, 85, 101, 103, 103, 102, 108, 108, 107,
	99, 99, 98, 98, 17, 100, 30, 30, 104, 106,
	106, 105, 86, 86, 109, 109, 109, 109, 110, 110,
	110, 114, 114, 111, 111, 111, 112, 113, 88, 88,
	115, 116, 116, 117, 117, 118, 118, 89, 90, 119,
	119, 46, 46, 46, 46, 46, 46, 7, 7, 7,
	7, 7, 7, 7, 7, 7, 7, 7, 7, 7,
	7, 7, 7, 7, 7, 7, 7, 7, 7, 7,
	7, 7, 7, 7, 7, 7, 7, 7, 7, 7,
	7, 7, 7, 7, 7, 7, 8, 8, 8, 8,
	8, 8, 8, 9, 9, 9, 9, 9, 9, 9,
	9, 9, 9, 9, 9, 9, 9, 1, 1, 1,
	1, 1, 1, 1, 1, 2, 6, 6, 5, 5,
	4, 3, 11, 11, 18, 19, 19, 20, 23, 23,
	21, 22, 22, 31, 31, 31, 32, 24, 24, 25,
	25, 25, 28, 28, 27, 27, 29, 26, 26, 33,
	34, 34,
}
var yyR2 = []int{

	0, 1, 1, 1, 1, 1, 1, 2, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1, 4, 1,
	3, 4, 3, 4, 3, 4, 1, 1, 5, 5,
	2, 1, 2, 2, 2, 3, 1, 1, 1, 3,
	1, 3, 2, 0, 1, 1, 2, 1, 0, 1,
	2, 1, 4, 4, 5, 1, 1, 4, 6, 6,
	4, 6, 6, 1, 1, 0, 2, 0, 1, 4,
	0, 1, 0, 1, 2, 0, 1, 4, 0, 1,
	2, 1, 3, 3, 0, 1, 2, 0, 1, 5,
	1, 3, 0, 1, 2, 0, 1, 2, 0, 1,
	3, 1, 3, 2, 0, 1, 1, 1, 0, 1,
	2, 0, 1, 2, 6, 6, 4, 2, 0, 1,
	2, 2, 0, 1, 2, 1, 2, 6, 6, 7,
	8, 7, 7, 2, 1, 3, 4, 0, 1, 4,
	1, 3, 3, 3, 1, 1, 0, 2, 2, 1,
	3, 2, 10, 13, 0, 6, 6, 6, 0, 6,
	6, 0, 6, 2, 3, 2, 1, 2, 5, 11,
	1, 1, 3, 0, 3, 0, 2, 5, 6, 0,
	3, 1, 3, 3, 5, 5, 4, 1, 3, 3,
	5, 5, 4, 5, 6, 3, 3, 3, 3, 3,
	3, 3, 3, 2, 3, 3, 3, 3, 3, 3,
	3, 5, 6, 3, 4, 3, 4, 3, 4, 3,
	4, 3, 4, 3, 4, 2, 1, 1, 1, 2,
	1, 1, 1, 1, 3, 3, 5, 5, 4, 5,
	6, 3, 3, 3, 3, 3, 3, 1, 1, 1,
	1, 1, 1, 1, 1, 3, 0, 1, 1, 3,
	3, 3, 0, 1, 3, 1, 1, 3, 4, 5,
	2, 0, 2, 4, 5, 4, 1, 1, 1, 4,
	4, 4, 1, 3, 3, 3, 2, 6, 6, 3,
	1, 1,
}
var yyChk = []int{

	-1000, -120, -77, -7, -78, -79, -80, -81, -8, 86,
	46, 47, -35, -82, -83, -84, -85, -86, -87, -1,
	156, -31, 82, -18, -24, -33, -36, 63, 138, 31,
	137, 81, -88, -89, -90, 87, 48, 129, 154, 153,
	155, -2, -3, -32, 18, -25, -26, 158, -37, 26,
	37, 5, 160, 162, 8, 121, 42, 9, 49, -39,
	-38, -41, -66, 52, 118, 181, 162, 176, 82, 177,
	178, 179, 175, 7, 92, 168, 169, 170, 171, 172,
	173, 174, 13, 86, 75, 58, 150, 66, -7, -7,
	-77, -7, -68, 133, 64, 43, -67, 93, 65, 65,
	52, -91, -48, -49, 156, 65, 158, -19, -20, -21,
	-7, -23, 146, -34, -7, -35, 101, 60, 60, 60,
	-6, -5, -4, 155, -11, -10, -7, -28, -27, -17,
	156, -28, -28, -7, -7, -53, -54, 73, -42, -41,
	-40, -43, -49, -48, 124, -65, -64, 35, 4, -121,
	-63, 106, 39, 177, -7, 156, 157, 162, -7, -7,
	-7, -7, -7, -7, -7, -7, -7, -7, -7, -7,
	-7, -7, -7, -7, -9, -8, 13, 75, 58, 150,
	-7, -7, -7, 87, 86, 83, 143, -72, -71, 76,
	-37, 4, -37, 4, -37, 4, 16, -91, -91, -91,
	-51, -50, 139, 166, -16, -15, -14, 10, 156, -91,
	-11, 35, 177, 41, -23, 146, -22, 40, -7, 159,
	60, -115, 156, -116, -49, -48, -116, 161, 165, 166,
	163, 165, -29, 165, 116, 58, 150, -29, -29, 51,
	51, -55, -56, 147, -13, -12, -14, -53, -45, 62,
	72, -47, 181, 166, 166, 165, -64, -121, -64, -7,
	181, -16, -7, 163, 166, 7, 181, 162, 176, 82,
	177, 178, 179, 175, -9, -7, -7, -7, 87, 83,
	143, -74, -73, 89, -7, -37, -37, -37, -70, -69,
	-7, -94, -93, 68, -93, -51, -101, -104, 119, 136,
	-123, 101, -49, 156, -14, 141, 159, -7, 159, -22,
	-7, -7, 125, 90, 90, 181, 166, 181, -4, -7,
	-7, 41, -27, -7, -7, -7, 41, 41, -28, -28,
	-57, -58, 55, -7, 165, 168, -55, 67, 85, -122,
	135, 50, -124, 94, -16, -46, 156, -49, -49, -63,
	-7, 177, 163, 164, 163, -7, -9, 156, 157, 162,
	-7, -9, -9, -9, -9, -9, -9, 7, -7, 165,
	-76, -75, 11, 33, -92, -35, 144, -7, -92, -35,
	-55, -104, -55, -55, -103, -102, -46, -106, -105, -46,
	69, -16, -43, 158, 159, 125, -7, -116, -116, -115,
	-49, -115, -30, 146, -30, -66, 16, -12, -7, -57,
	-44, -49, -48, 124, -44, -7, -51, 181, 162, -47,
	-47, 163, -7, 163, 166, -9, -69, -97, -96, 111,
	-97, -7, -97, -97, -72, -55, -72, -72, 165, 168,
	165, -108, -107, 51, -7, 90, -35, -7, 158, -119,
	109, 41, -7, 41, -10, -47, 166, 166, -16, 156,
	157, 162, -7, -16, -16, 163, 164, 163, -7, -95,
	-64, -121, -97, -72, -97, -97, -102, -7, -105, -99,
	-98, -17, -93, 159, -10, 126, -59, -60, 74, -16,
	-49, -49, -7, 163, -51, -51, 163, -7, -97, -108,
	-30, 165, 58, 150, -109, 146, -15, 159, -115, -61,
	-62, 56, -13, -52, 90, -47, -47, 163, 164, 41,
	-98, -100, -46, -100, -72, 79, 86, 90, -117, 96,
	-7, -123, -16, -16, -97, 125, 79, -93, -55, 16,
	69, -52, -52, 137, 31, 125, -109, -118, 141, -7,
	-7, -111, -101, -104, -112, -55, 63, -72, 145, -110,
	146, -55, -104, -55, -114, 146, -113, -7, -97, 79,
	86, -55, 86, -55, 125, 79, 79, 31, 125, 125,
	-112, 63, 63, -114, -113, -113,
}
var yyDef = []int{

	0, -2, 1, 2, 3, 4, 5, 6, 187, 0,
	0, 0, 8, 9, 10, 11, 12, 13, 14, 226,
	-2, 228, 0, 230, 231, 232, 98, 0, 0, 0,
	0, 0, 15, 16, 17, 247, 248, 249, 250, 251,
	252, 253, 254, 0, 0, 277, 278, 0, 19, 0,
	0, 0, 256, 262, 0, 0, 0, 0, 0, 26,
	27, 78, 48, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 203, 225,
	7, 229, 108, 0, 0, 0, 99, 0, 0, 0,
	0, 67, 0, 43, -2, 0, 262, 0, 265, 266,
	0, 271, 0, 0, 290, 291, 0, 0, 0, 0,
	0, 257, 258, 0, 0, 263, 90, 0, 282, 0,
	144, 0, 0, 0, 0, 84, 79, 0, 78, 49,
	-2, 51, 65, 0, 0, 30, 31, 0, 0, 0,
	38, 36, 37, 40, 43, 188, 189, 0, 0, 195,
	196, 197, 198, 199, 200, 201, 202, -2, -2, -2,
	-2, -2, -2, -2, 0, 233, 0, 0, 0, 0,
	-2, -2, -2, 219, 0, 221, 223, 111, 109, 0,
	20, 0, 22, 0, 24, 0, 0, 118, 0, 67,
	0, 68, 70, 0, 117, 44, 45, 0, 47, 0,
	0, 0, 0, 264, 271, 0, 270, 0, 0, 289,
	0, 0, 170, 0, 171, 0, 0, 255, 0, 0,
	261, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 87, 85, 0, 80, 81, 0, 84, 0, 73,
	75, 43, 0, 0, 0, 0, 32, 0, 33, 34,
	0, 42, 0, 192, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, -2, -2, -2, 220, 222,
	224, 18, 112, 0, 110, 21, 23, 25, 100, 101,
	104, 0, 119, 0, 0, 84, 84, 84, 0, 0,
	0, 71, 43, 64, 46, 0, 273, 0, 275, 267,
	0, 272, 0, 0, 0, 0, 0, 0, 259, 260,
	91, 279, 283, 286, 284, 285, 280, 281, 146, 146,
	0, 88, 0, 86, 0, 0, 87, 0, 0, 0,
	55, 56, 74, 76, 67, 66, 181, 65, 65, 39,
	35, 41, 190, 191, 193, 0, 211, 234, 235, 0,
	0, 241, 242, 243, 244, 245, 246, 0, 113, 0,
	103, 105, 106, 107, 122, 122, 0, 120, 122, 122,
	108, 84, 108, 108, 133, 134, 0, 148, 149, 137,
	0, 116, 0, 0, 274, 0, 268, 168, 0, 177,
	172, 179, 0, 0, 0, 28, 0, 82, 83, 29,
	52, 65, 0, 0, 53, 43, 57, 0, 0, 43,
	43, 194, 0, 238, 0, 212, 102, 114, 123, 0,
	115, 121, 127, 128, 122, 108, 122, 122, 0, 0,
	0, 151, 138, 0, 69, 0, 0, 269, 0, 178,
	0, 287, 147, 288, 92, 43, 0, 0, 54, 182,
	183, 0, 0, 67, 67, 236, 237, 239, 0, 124,
	125, 0, 129, 122, 131, 132, 135, 137, 150, 146,
	140, 0, 154, 0, 0, 0, 95, 93, 0, 0,
	65, 65, 0, 186, 58, 59, 240, 126, 130, 136,
	0, 0, 0, 0, 108, 0, 0, 173, 180, 89,
	96, 0, 94, 60, 70, 43, 43, 184, 185, 139,
	141, 142, 145, 143, 122, 0, 0, 0, 84, 0,
	97, 0, 0, 0, 152, 0, 0, 154, 175, 0,
	0, 61, 62, 0, 84, 0, 108, 169, 0, 174,
	77, 158, 84, 84, 161, 166, 0, 122, 176, 155,
	0, 163, 84, 165, 156, 0, 157, 84, 153, 0,
	0, 164, 0, 167, 0, 0, 0, 84, 0, 0,
	161, 0, 0, 159, 160, 162,
}
var yyTok1 = []int{

	1,
}
var yyTok2 = []int{

	2, 3, 4, 5, 6, 7, 8, 9, 10, 11,
	12, 13, 14, 15, 16, 17, 18, 19, 20, 21,
	22, 23, 24, 25, 26, 27, 28, 29, 30, 31,
	32, 33, 34, 35, 36, 37, 38, 39, 40, 41,
	42, 43, 44, 45, 46, 47, 48, 49, 50, 51,
	52, 53, 54, 55, 56, 57, 58, 59, 60, 61,
	62, 63, 64, 65, 66, 67, 68, 69, 70, 71,
	72, 73, 74, 75, 76, 77, 78, 79, 80, 81,
	82, 83, 84, 85, 86, 87, 88, 89, 90, 91,
	92, 93, 94, 95, 96, 97, 98, 99, 100, 101,
	102, 103, 104, 105, 106, 107, 108, 109, 110, 111,
	112, 113, 114, 115, 116, 117, 118, 119, 120, 121,
	122, 123, 124, 125, 126, 127, 128, 129, 130, 131,
	132, 133, 134, 135, 136, 137, 138, 139, 140, 141,
	142, 143, 144, 145, 146, 147, 148, 149, 150, 151,
	152, 153, 154, 155, 156, 157, 158, 159, 160, 161,
	162, 163, 164, 165, 166, 167, 168, 169, 170, 171,
	172, 173, 174, 175, 176, 177, 178, 179, 180, 181,
}
var yyTok3 = []int{
	0,
}

//line yaccpar:1

/*	parser for yacc output	*/

var yyDebug = 0

type yyLexer interface {
	Lex(lval *yySymType) int
	Error(s string)
}

const yyFlag = -1000

func yyTokname(c int) string {
	// 4 is TOKSTART above
	if c >= 4 && c-4 < len(yyToknames) {
		if yyToknames[c-4] != "" {
			return yyToknames[c-4]
		}
	}
	return __yyfmt__.Sprintf("tok-%v", c)
}

func yyStatname(s int) string {
	if s >= 0 && s < len(yyStatenames) {
		if yyStatenames[s] != "" {
			return yyStatenames[s]
		}
	}
	return __yyfmt__.Sprintf("state-%v", s)
}

func yylex1(lex yyLexer, lval *yySymType) int {
	c := 0
	char := lex.Lex(lval)
	if char <= 0 {
		c = yyTok1[0]
		goto out
	}
	if char < len(yyTok1) {
		c = yyTok1[char]
		goto out
	}
	if char >= yyPrivate {
		if char < yyPrivate+len(yyTok2) {
			c = yyTok2[char-yyPrivate]
			goto out
		}
	}
	for i := 0; i < len(yyTok3); i += 2 {
		c = yyTok3[i+0]
		if c == char {
			c = yyTok3[i+1]
			goto out
		}
	}

out:
	if c == 0 {
		c = yyTok2[1] /* unknown char */
	}
	if yyDebug >= 3 {
		__yyfmt__.Printf("lex %s(%d)\n", yyTokname(c), uint(char))
	}
	return c
}

func yyParse(yylex yyLexer) int {
	var yyn int
	var yylval yySymType
	var yyVAL yySymType
	yyS := make([]yySymType, yyMaxDepth)

	Nerrs := 0   /* number of errors */
	Errflag := 0 /* error recovery flag */
	yystate := 0
	yychar := -1
	yyp := -1
	goto yystack

ret0:
	return 0

ret1:
	return 1

yystack:
	/* put a state and value onto the stack */
	if yyDebug >= 4 {
		__yyfmt__.Printf("char %v in %v\n", yyTokname(yychar), yyStatname(yystate))
	}

	yyp++
	if yyp >= len(yyS) {
		nyys := make([]yySymType, len(yyS)*2)
		copy(nyys, yyS)
		yyS = nyys
	}
	yyS[yyp] = yyVAL
	yyS[yyp].yys = yystate

yynewstate:
	yyn = yyPact[yystate]
	if yyn <= yyFlag {
		goto yydefault /* simple state */
	}
	if yychar < 0 {
		yychar = yylex1(yylex, &yylval)
	}
	yyn += yychar
	if yyn < 0 || yyn >= yyLast {
		goto yydefault
	}
	yyn = yyAct[yyn]
	if yyChk[yyn] == yychar { /* valid shift */
		yychar = -1
		yyVAL = yylval
		yystate = yyn
		if Errflag > 0 {
			Errflag--
		}
		goto yystack
	}

yydefault:
	/* default state action */
	yyn = yyDef[yystate]
	if yyn == -2 {
		if yychar < 0 {
			yychar = yylex1(yylex, &yylval)
		}

		/* look through exception table */
		xi := 0
		for {
			if yyExca[xi+0] == -1 && yyExca[xi+1] == yystate {
				break
			}
			xi += 2
		}
		for xi += 2; ; xi += 2 {
			yyn = yyExca[xi+0]
			if yyn < 0 || yyn == yychar {
				break
			}
		}
		yyn = yyExca[xi+1]
		if yyn < 0 {
			goto ret0
		}
	}
	if yyn == 0 {
		/* error ... attempt to resume parsing */
		switch Errflag {
		case 0: /* brand new error */
			yylex.Error("syntax error")
			Nerrs++
			if yyDebug >= 1 {
				__yyfmt__.Printf("%s", yyStatname(yystate))
				__yyfmt__.Printf(" saw %s\n", yyTokname(yychar))
			}
			fallthrough

		case 1, 2: /* incompletely recovered error ... try again */
			Errflag = 3

			/* find a state where "error" is a legal shift action */
			for yyp >= 0 {
				yyn = yyPact[yyS[yyp].yys] + yyErrCode
				if yyn >= 0 && yyn < yyLast {
					yystate = yyAct[yyn] /* simulate a shift of "error" */
					if yyChk[yystate] == yyErrCode {
						goto yystack
					}
				}

				/* the current p has no shift on "error", pop stack */
				if yyDebug >= 2 {
					__yyfmt__.Printf("error recovery pops state %d\n", yyS[yyp].yys)
				}
				yyp--
			}
			/* there is no state on the stack with an error shift ... abort */
			goto ret1

		case 3: /* no shift yet; clobber input char */
			if yyDebug >= 2 {
				__yyfmt__.Printf("error recovery discards %s\n", yyTokname(yychar))
			}
			if yychar == yyEofCode {
				goto ret1
			}
			yychar = -1
			goto yynewstate /* try again in the same state */
		}
	}

	/* reduction by production yyn */
	if yyDebug >= 2 {
		__yyfmt__.Printf("reduce %v in:\n\t%v\n", yyn, yyStatname(yystate))
	}

	yynt := yyn
	yypt := yyp
	_ = yypt // guard against "declared and not used"

	yyp -= yyR2[yyn]
	yyVAL = yyS[yyp+1]

	/* consult goto table to find next state */
	yyn = yyR1[yyn]
	yyg := yyPgo[yyn]
	yyj := yyg + yyS[yyp].yys + 1

	if yyj >= yyLast {
		yystate = yyAct[yyg]
	} else {
		yystate = yyAct[yyj]
		if yyChk[yystate] != -yyn {
			yystate = yyAct[yyg]
		}
	}
	// dummy call; replaced with literal code
	switch yynt {

	case 1:
		//line n1ql.y:339
		{
			yylex.(*lexer).setStatement(yyS[yypt-0].statement)
		}
	case 2:
		//line n1ql.y:344
		{
			yylex.(*lexer).setExpression(yyS[yypt-0].expr)
		}
	case 3:
		yyVAL.statement = yyS[yypt-0].statement
	case 4:
		yyVAL.statement = yyS[yypt-0].statement
	case 5:
		yyVAL.statement = yyS[yypt-0].statement
	case 6:
		yyVAL.statement = yyS[yypt-0].statement
	case 7:
		//line n1ql.y:361
		{
			yyVAL.statement = algebra.NewExplain(yyS[yypt-0].statement)
		}
	case 8:
		//line n1ql.y:368
		{
			yyVAL.statement = yyS[yypt-0].fullselect
		}
	case 9:
		yyVAL.statement = yyS[yypt-0].statement
	case 10:
		yyVAL.statement = yyS[yypt-0].statement
	case 11:
		yyVAL.statement = yyS[yypt-0].statement
	case 12:
		yyVAL.statement = yyS[yypt-0].statement
	case 13:
		yyVAL.statement = yyS[yypt-0].statement
	case 14:
		yyVAL.statement = yyS[yypt-0].statement
	case 15:
		yyVAL.statement = yyS[yypt-0].statement
	case 16:
		yyVAL.statement = yyS[yypt-0].statement
	case 17:
		yyVAL.statement = yyS[yypt-0].statement
	case 18:
		//line n1ql.y:399
		{
			yyVAL.fullselect = algebra.NewSelect(yyS[yypt-3].subresult, yyS[yypt-2].order, yyS[yypt-0].expr, yyS[yypt-1].expr) /* OFFSET precedes LIMIT */
		}
	case 19:
		//line n1ql.y:405
		{
			yyVAL.subresult = yyS[yypt-0].subselect
		}
	case 20:
		//line n1ql.y:410
		{
			yyVAL.subresult = algebra.NewUnion(yyS[yypt-2].subresult, yyS[yypt-0].subselect)
		}
	case 21:
		//line n1ql.y:415
		{
			yyVAL.subresult = algebra.NewUnionAll(yyS[yypt-3].subresult, yyS[yypt-0].subselect)
		}
	case 22:
		//line n1ql.y:420
		{
			yyVAL.subresult = algebra.NewIntersect(yyS[yypt-2].subresult, yyS[yypt-0].subselect)
		}
	case 23:
		//line n1ql.y:425
		{
			yyVAL.subresult = algebra.NewIntersectAll(yyS[yypt-3].subresult, yyS[yypt-0].subselect)
		}
	case 24:
		//line n1ql.y:430
		{
			yyVAL.subresult = algebra.NewExcept(yyS[yypt-2].subresult, yyS[yypt-0].subselect)
		}
	case 25:
		//line n1ql.y:435
		{
			yyVAL.subresult = algebra.NewExceptAll(yyS[yypt-3].subresult, yyS[yypt-0].subselect)
		}
	case 26:
		yyVAL.subselect = yyS[yypt-0].subselect
	case 27:
		yyVAL.subselect = yyS[yypt-0].subselect
	case 28:
		//line n1ql.y:448
		{
			yyVAL.subselect = algebra.NewSubselect(yyS[yypt-4].fromTerm, yyS[yypt-3].bindings, yyS[yypt-2].expr, yyS[yypt-1].group, yyS[yypt-0].projection)
		}
	case 29:
		//line n1ql.y:455
		{
			yyVAL.subselect = algebra.NewSubselect(yyS[yypt-3].fromTerm, yyS[yypt-2].bindings, yyS[yypt-1].expr, yyS[yypt-0].group, yyS[yypt-4].projection)
		}
	case 30:
		//line n1ql.y:470
		{
			yyVAL.projection = yyS[yypt-0].projection
		}
	case 31:
		//line n1ql.y:477
		{
			yyVAL.projection = algebra.NewProjection(false, yyS[yypt-0].resultTerms)
		}
	case 32:
		//line n1ql.y:482
		{
			yyVAL.projection = algebra.NewProjection(true, yyS[yypt-0].resultTerms)
		}
	case 33:
		//line n1ql.y:487
		{
			yyVAL.projection = algebra.NewProjection(false, yyS[yypt-0].resultTerms)
		}
	case 34:
		//line n1ql.y:492
		{
			yyVAL.projection = algebra.NewRawProjection(false, yyS[yypt-0].expr)
		}
	case 35:
		//line n1ql.y:497
		{
			yyVAL.projection = algebra.NewRawProjection(true, yyS[yypt-0].expr)
		}
	case 38:
		//line n1ql.y:510
		{
			yyVAL.resultTerms = algebra.ResultTerms{yyS[yypt-0].resultTerm}
		}
	case 39:
		//line n1ql.y:515
		{
			yyVAL.resultTerms = append(yyS[yypt-2].resultTerms, yyS[yypt-0].resultTerm)
		}
	case 40:
		//line n1ql.y:522
		{
			yyVAL.resultTerm = algebra.NewResultTerm(nil, true, "")
		}
	case 41:
		//line n1ql.y:527
		{
			yyVAL.resultTerm = algebra.NewResultTerm(yyS[yypt-2].expr, true, "")
		}
	case 42:
		//line n1ql.y:532
		{
			yyVAL.resultTerm = algebra.NewResultTerm(yyS[yypt-1].expr, false, yyS[yypt-0].s)
		}
	case 43:
		//line n1ql.y:539
		{
			yyVAL.s = ""
		}
	case 44:
		yyVAL.s = yyS[yypt-0].s
	case 45:
		yyVAL.s = yyS[yypt-0].s
	case 46:
		//line n1ql.y:550
		{
			yyVAL.s = yyS[yypt-0].s
		}
	case 47:
		yyVAL.s = yyS[yypt-0].s
	case 48:
		//line n1ql.y:568
		{
			yyVAL.fromTerm = nil
		}
	case 49:
		yyVAL.fromTerm = yyS[yypt-0].fromTerm
	case 50:
		//line n1ql.y:577
		{
			yyVAL.fromTerm = yyS[yypt-0].fromTerm
		}
	case 51:
		//line n1ql.y:584
		{
			yyVAL.fromTerm = yyS[yypt-0].keyspaceTerm
		}
	case 52:
		//line n1ql.y:589
		{
			yyVAL.fromTerm = algebra.NewJoin(yyS[yypt-3].fromTerm, yyS[yypt-2].b, yyS[yypt-0].keyspaceTerm)
		}
	case 53:
		//line n1ql.y:594
		{
			yyVAL.fromTerm = algebra.NewNest(yyS[yypt-3].fromTerm, yyS[yypt-2].b, yyS[yypt-0].keyspaceTerm)
		}
	case 54:
		//line n1ql.y:599
		{
			yyVAL.fromTerm = algebra.NewUnnest(yyS[yypt-4].fromTerm, yyS[yypt-3].b, yyS[yypt-1].expr, yyS[yypt-0].s)
		}
	case 57:
		//line n1ql.y:612
		{
			yyVAL.keyspaceTerm = algebra.NewKeyspaceTerm("", yyS[yypt-3].s, yyS[yypt-2].path, yyS[yypt-1].s, yyS[yypt-0].expr)
		}
	case 58:
		//line n1ql.y:617
		{
			yyVAL.keyspaceTerm = algebra.NewKeyspaceTerm(yyS[yypt-5].s, yyS[yypt-3].s, yyS[yypt-2].path, yyS[yypt-1].s, yyS[yypt-0].expr)
		}
	case 59:
		//line n1ql.y:622
		{
			yyVAL.keyspaceTerm = algebra.NewKeyspaceTerm("#system", yyS[yypt-3].s, yyS[yypt-2].path, yyS[yypt-1].s, yyS[yypt-0].expr)
		}
	case 60:
		//line n1ql.y:629
		{
			yyVAL.keyspaceTerm = algebra.NewKeyspaceTerm("", yyS[yypt-3].s, yyS[yypt-2].path, yyS[yypt-1].s, yyS[yypt-0].expr)
		}
	case 61:
		//line n1ql.y:634
		{
			yyVAL.keyspaceTerm = algebra.NewKeyspaceTerm(yyS[yypt-5].s, yyS[yypt-3].s, yyS[yypt-2].path, yyS[yypt-1].s, yyS[yypt-0].expr)
		}
	case 62:
		//line n1ql.y:639
		{
			yyVAL.keyspaceTerm = algebra.NewKeyspaceTerm("#system", yyS[yypt-3].s, yyS[yypt-2].path, yyS[yypt-1].s, yyS[yypt-0].expr)
		}
	case 63:
		yyVAL.s = yyS[yypt-0].s
	case 64:
		yyVAL.s = yyS[yypt-0].s
	case 65:
		//line n1ql.y:654
		{
			yyVAL.path = nil
		}
	case 66:
		//line n1ql.y:659
		{
			yyVAL.path = yyS[yypt-0].path
		}
	case 67:
		//line n1ql.y:666
		{
			yyVAL.expr = nil
		}
	case 68:
		yyVAL.expr = yyS[yypt-0].expr
	case 69:
		//line n1ql.y:675
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 70:
		//line n1ql.y:682
		{
		}
	case 72:
		//line n1ql.y:690
		{
			yyVAL.b = false
		}
	case 73:
		//line n1ql.y:695
		{
			yyVAL.b = false
		}
	case 74:
		//line n1ql.y:700
		{
			yyVAL.b = true
		}
	case 77:
		//line n1ql.y:713
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 78:
		//line n1ql.y:727
		{
			yyVAL.bindings = nil
		}
	case 79:
		yyVAL.bindings = yyS[yypt-0].bindings
	case 80:
		//line n1ql.y:736
		{
			yyVAL.bindings = yyS[yypt-0].bindings
		}
	case 81:
		//line n1ql.y:743
		{
			yyVAL.bindings = expression.Bindings{yyS[yypt-0].binding}
		}
	case 82:
		//line n1ql.y:748
		{
			yyVAL.bindings = append(yyS[yypt-2].bindings, yyS[yypt-0].binding)
		}
	case 83:
		//line n1ql.y:755
		{
			yyVAL.binding = expression.NewBinding(yyS[yypt-2].s, yyS[yypt-0].expr)
		}
	case 84:
		//line n1ql.y:769
		{
			yyVAL.expr = nil
		}
	case 85:
		yyVAL.expr = yyS[yypt-0].expr
	case 86:
		//line n1ql.y:778
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 87:
		//line n1ql.y:792
		{
			yyVAL.group = nil
		}
	case 88:
		yyVAL.group = yyS[yypt-0].group
	case 89:
		//line n1ql.y:801
		{
			yyVAL.group = algebra.NewGroup(yyS[yypt-2].exprs, yyS[yypt-1].bindings, yyS[yypt-0].expr)
		}
	case 90:
		//line n1ql.y:808
		{
			yyVAL.exprs = expression.Expressions{yyS[yypt-0].expr}
		}
	case 91:
		//line n1ql.y:813
		{
			yyVAL.exprs = append(yyS[yypt-2].exprs, yyS[yypt-0].expr)
		}
	case 92:
		//line n1ql.y:820
		{
			yyVAL.bindings = nil
		}
	case 93:
		yyVAL.bindings = yyS[yypt-0].bindings
	case 94:
		//line n1ql.y:829
		{
			yyVAL.bindings = yyS[yypt-0].bindings
		}
	case 95:
		//line n1ql.y:836
		{
			yyVAL.expr = nil
		}
	case 96:
		yyVAL.expr = yyS[yypt-0].expr
	case 97:
		//line n1ql.y:845
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 98:
		//line n1ql.y:859
		{
			yyVAL.order = nil
		}
	case 99:
		yyVAL.order = yyS[yypt-0].order
	case 100:
		//line n1ql.y:868
		{
			yyVAL.order = algebra.NewOrder(yyS[yypt-0].sortTerms)
		}
	case 101:
		//line n1ql.y:875
		{
			yyVAL.sortTerms = algebra.SortTerms{yyS[yypt-0].sortTerm}
		}
	case 102:
		//line n1ql.y:880
		{
			yyVAL.sortTerms = append(yyS[yypt-2].sortTerms, yyS[yypt-0].sortTerm)
		}
	case 103:
		//line n1ql.y:887
		{
			yyVAL.sortTerm = algebra.NewSortTerm(yyS[yypt-1].expr, yyS[yypt-0].b)
		}
	case 104:
		//line n1ql.y:894
		{
			yyVAL.b = false
		}
	case 105:
		yyVAL.b = yyS[yypt-0].b
	case 106:
		//line n1ql.y:903
		{
			yyVAL.b = false
		}
	case 107:
		//line n1ql.y:908
		{
			yyVAL.b = true
		}
	case 108:
		//line n1ql.y:922
		{
			yyVAL.expr = nil
		}
	case 109:
		yyVAL.expr = yyS[yypt-0].expr
	case 110:
		//line n1ql.y:931
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 111:
		//line n1ql.y:945
		{
			yyVAL.expr = nil
		}
	case 112:
		yyVAL.expr = yyS[yypt-0].expr
	case 113:
		//line n1ql.y:954
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 114:
		//line n1ql.y:968
		{
			yyVAL.statement = algebra.NewInsertValues(yyS[yypt-3].keyspaceRef, yyS[yypt-2].expr, yyS[yypt-1].expr, yyS[yypt-0].projection)
		}
	case 115:
		//line n1ql.y:973
		{
			yyVAL.statement = algebra.NewInsertSelect(yyS[yypt-3].keyspaceRef, yyS[yypt-2].expr, yyS[yypt-1].fullselect, yyS[yypt-0].projection)
		}
	case 116:
		//line n1ql.y:980
		{
			yyVAL.keyspaceRef = algebra.NewKeyspaceRef(yyS[yypt-3].s, yyS[yypt-1].s, yyS[yypt-0].s)
		}
	case 117:
		//line n1ql.y:985
		{
			yyVAL.keyspaceRef = algebra.NewKeyspaceRef("", yyS[yypt-1].s, yyS[yypt-0].s)
		}
	case 118:
		//line n1ql.y:992
		{
			yyVAL.expr = nil
		}
	case 119:
		yyVAL.expr = yyS[yypt-0].expr
	case 120:
		//line n1ql.y:1001
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 121:
		//line n1ql.y:1008
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 122:
		//line n1ql.y:1015
		{
			yyVAL.projection = nil
		}
	case 123:
		yyVAL.projection = yyS[yypt-0].projection
	case 124:
		//line n1ql.y:1024
		{
			yyVAL.projection = yyS[yypt-0].projection
		}
	case 125:
		//line n1ql.y:1031
		{
			yyVAL.projection = algebra.NewProjection(false, yyS[yypt-0].resultTerms)
		}
	case 126:
		//line n1ql.y:1036
		{
			yyVAL.projection = algebra.NewRawProjection(false, yyS[yypt-0].expr)
		}
	case 127:
		//line n1ql.y:1050
		{
			yyVAL.statement = algebra.NewUpsertValues(yyS[yypt-3].keyspaceRef, yyS[yypt-2].expr, yyS[yypt-1].expr, yyS[yypt-0].projection)
		}
	case 128:
		//line n1ql.y:1055
		{
			yyVAL.statement = algebra.NewUpsertSelect(yyS[yypt-3].keyspaceRef, yyS[yypt-2].expr, yyS[yypt-1].fullselect, yyS[yypt-0].projection)
		}
	case 129:
		//line n1ql.y:1069
		{
			yyVAL.statement = algebra.NewDelete(yyS[yypt-4].keyspaceRef, yyS[yypt-3].expr, yyS[yypt-2].expr, yyS[yypt-1].expr, yyS[yypt-0].projection)
		}
	case 130:
		//line n1ql.y:1083
		{
			yyVAL.statement = algebra.NewUpdate(yyS[yypt-6].keyspaceRef, yyS[yypt-5].expr, yyS[yypt-4].set, yyS[yypt-3].unset, yyS[yypt-2].expr, yyS[yypt-1].expr, yyS[yypt-0].projection)
		}
	case 131:
		//line n1ql.y:1088
		{
			yyVAL.statement = algebra.NewUpdate(yyS[yypt-5].keyspaceRef, yyS[yypt-4].expr, yyS[yypt-3].set, nil, yyS[yypt-2].expr, yyS[yypt-1].expr, yyS[yypt-0].projection)
		}
	case 132:
		//line n1ql.y:1093
		{
			yyVAL.statement = algebra.NewUpdate(yyS[yypt-5].keyspaceRef, yyS[yypt-4].expr, nil, yyS[yypt-3].unset, yyS[yypt-2].expr, yyS[yypt-1].expr, yyS[yypt-0].projection)
		}
	case 133:
		//line n1ql.y:1100
		{
			yyVAL.set = algebra.NewSet(yyS[yypt-0].setTerms)
		}
	case 134:
		//line n1ql.y:1107
		{
			yyVAL.setTerms = algebra.SetTerms{yyS[yypt-0].setTerm}
		}
	case 135:
		//line n1ql.y:1112
		{
			yyVAL.setTerms = append(yyS[yypt-2].setTerms, yyS[yypt-0].setTerm)
		}
	case 136:
		//line n1ql.y:1119
		{
			yyVAL.setTerm = algebra.NewSetTerm(yyS[yypt-3].path, yyS[yypt-1].expr, yyS[yypt-0].updateFor)
		}
	case 137:
		//line n1ql.y:1126
		{
			yyVAL.updateFor = nil
		}
	case 138:
		yyVAL.updateFor = yyS[yypt-0].updateFor
	case 139:
		//line n1ql.y:1135
		{
			yyVAL.updateFor = algebra.NewUpdateFor(yyS[yypt-2].bindings, yyS[yypt-1].expr)
		}
	case 140:
		//line n1ql.y:1142
		{
			yyVAL.bindings = expression.Bindings{yyS[yypt-0].binding}
		}
	case 141:
		//line n1ql.y:1147
		{
			yyVAL.bindings = append(yyS[yypt-2].bindings, yyS[yypt-0].binding)
		}
	case 142:
		//line n1ql.y:1154
		{
			yyVAL.binding = expression.NewBinding(yyS[yypt-2].s, yyS[yypt-0].expr)
		}
	case 143:
		//line n1ql.y:1159
		{
			yyVAL.binding = expression.NewDescendantBinding(yyS[yypt-2].s, yyS[yypt-0].expr)
		}
	case 144:
		yyVAL.s = yyS[yypt-0].s
	case 145:
		//line n1ql.y:1170
		{
			yyVAL.expr = yyS[yypt-0].path
		}
	case 146:
		//line n1ql.y:1177
		{
			yyVAL.expr = nil
		}
	case 147:
		//line n1ql.y:1182
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 148:
		//line n1ql.y:1189
		{
			yyVAL.unset = algebra.NewUnset(yyS[yypt-0].unsetTerms)
		}
	case 149:
		//line n1ql.y:1196
		{
			yyVAL.unsetTerms = algebra.UnsetTerms{yyS[yypt-0].unsetTerm}
		}
	case 150:
		//line n1ql.y:1201
		{
			yyVAL.unsetTerms = append(yyS[yypt-2].unsetTerms, yyS[yypt-0].unsetTerm)
		}
	case 151:
		//line n1ql.y:1208
		{
			yyVAL.unsetTerm = algebra.NewUnsetTerm(yyS[yypt-1].path, yyS[yypt-0].updateFor)
		}
	case 152:
		//line n1ql.y:1222
		{
			source := algebra.NewMergeSourceFrom(yyS[yypt-5].keyspaceTerm, "")
			yyVAL.statement = algebra.NewMerge(yyS[yypt-7].keyspaceRef, source, yyS[yypt-3].expr, yyS[yypt-2].mergeActions, yyS[yypt-1].expr, yyS[yypt-0].projection)
		}
	case 153:
		//line n1ql.y:1228
		{
			source := algebra.NewMergeSourceSelect(yyS[yypt-7].fullselect, yyS[yypt-5].s)
			yyVAL.statement = algebra.NewMerge(yyS[yypt-10].keyspaceRef, source, yyS[yypt-3].expr, yyS[yypt-2].mergeActions, yyS[yypt-1].expr, yyS[yypt-0].projection)
		}
	case 154:
		//line n1ql.y:1236
		{
			yyVAL.mergeActions = algebra.NewMergeActions(nil, nil, nil)
		}
	case 155:
		//line n1ql.y:1241
		{
			yyVAL.mergeActions = algebra.NewMergeActions(yyS[yypt-1].mergeUpdate, yyS[yypt-0].mergeActions.Delete(), yyS[yypt-0].mergeActions.Insert())
		}
	case 156:
		//line n1ql.y:1246
		{
			yyVAL.mergeActions = algebra.NewMergeActions(nil, yyS[yypt-1].mergeDelete, yyS[yypt-0].mergeInsert)
		}
	case 157:
		//line n1ql.y:1251
		{
			yyVAL.mergeActions = algebra.NewMergeActions(nil, nil, yyS[yypt-0].mergeInsert)
		}
	case 158:
		//line n1ql.y:1258
		{
			yyVAL.mergeActions = algebra.NewMergeActions(nil, nil, nil)
		}
	case 159:
		//line n1ql.y:1263
		{
			yyVAL.mergeActions = algebra.NewMergeActions(nil, yyS[yypt-1].mergeDelete, yyS[yypt-0].mergeInsert)
		}
	case 160:
		//line n1ql.y:1268
		{
			yyVAL.mergeActions = algebra.NewMergeActions(nil, nil, yyS[yypt-0].mergeInsert)
		}
	case 161:
		//line n1ql.y:1275
		{
			yyVAL.mergeInsert = nil
		}
	case 162:
		//line n1ql.y:1280
		{
			yyVAL.mergeInsert = yyS[yypt-0].mergeInsert
		}
	case 163:
		//line n1ql.y:1287
		{
			yyVAL.mergeUpdate = algebra.NewMergeUpdate(yyS[yypt-1].set, nil, yyS[yypt-0].expr)
		}
	case 164:
		//line n1ql.y:1292
		{
			yyVAL.mergeUpdate = algebra.NewMergeUpdate(yyS[yypt-2].set, yyS[yypt-1].unset, yyS[yypt-0].expr)
		}
	case 165:
		//line n1ql.y:1297
		{
			yyVAL.mergeUpdate = algebra.NewMergeUpdate(nil, yyS[yypt-1].unset, yyS[yypt-0].expr)
		}
	case 166:
		//line n1ql.y:1304
		{
			yyVAL.mergeDelete = algebra.NewMergeDelete(yyS[yypt-0].expr)
		}
	case 167:
		//line n1ql.y:1311
		{
			yyVAL.mergeInsert = algebra.NewMergeInsert(yyS[yypt-1].expr, yyS[yypt-0].expr)
		}
	case 168:
		//line n1ql.y:1325
		{
			yyVAL.statement = algebra.NewCreatePrimaryIndex(yyS[yypt-0].keyspaceRef)
		}
	case 169:
		//line n1ql.y:1330
		{
			yyVAL.statement = algebra.NewCreateIndex(yyS[yypt-8].s, yyS[yypt-6].keyspaceRef, yyS[yypt-4].exprs, yyS[yypt-2].expr, yyS[yypt-1].expr, yyS[yypt-0].indexType)
		}
	case 170:
		yyVAL.s = yyS[yypt-0].s
	case 171:
		//line n1ql.y:1341
		{
			yyVAL.keyspaceRef = algebra.NewKeyspaceRef("", yyS[yypt-0].s, "")
		}
	case 172:
		//line n1ql.y:1346
		{
			yyVAL.keyspaceRef = algebra.NewKeyspaceRef(yyS[yypt-2].s, yyS[yypt-0].s, "")
		}
	case 173:
		//line n1ql.y:1353
		{
			yyVAL.expr = nil
		}
	case 174:
		//line n1ql.y:1358
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 175:
		//line n1ql.y:1365
		{
			yyVAL.indexType = datastore.VIEW
		}
	case 176:
		//line n1ql.y:1370
		{
			yyVAL.indexType = datastore.VIEW
		}
	case 177:
		//line n1ql.y:1384
		{
			yyVAL.statement = algebra.NewDropIndex(yyS[yypt-2].keyspaceRef, yyS[yypt-0].s)
		}
	case 178:
		//line n1ql.y:1397
		{
			yyVAL.statement = algebra.NewAlterIndex(yyS[yypt-3].keyspaceRef, yyS[yypt-1].s, yyS[yypt-0].s)
		}
	case 179:
		//line n1ql.y:1403
		{
			yyVAL.s = ""
		}
	case 180:
		//line n1ql.y:1408
		{
			yyVAL.s = yyS[yypt-0].s
		}
	case 181:
		//line n1ql.y:1422
		{
			yyVAL.path = expression.NewIdentifier(yyS[yypt-0].s)
		}
	case 182:
		//line n1ql.y:1427
		{
			yyVAL.path = expression.NewField(yyS[yypt-2].path, expression.NewFieldName(yyS[yypt-0].s))
		}
	case 183:
		//line n1ql.y:1432
		{
			field := expression.NewField(yyS[yypt-2].path, expression.NewFieldName(yyS[yypt-0].s))
			field.SetCaseInsensitive(true)
			yyVAL.path = field
		}
	case 184:
		//line n1ql.y:1439
		{
			yyVAL.path = expression.NewField(yyS[yypt-4].path, yyS[yypt-1].expr)
		}
	case 185:
		//line n1ql.y:1444
		{
			field := expression.NewField(yyS[yypt-4].path, yyS[yypt-1].expr)
			field.SetCaseInsensitive(true)
			yyVAL.path = field
		}
	case 186:
		//line n1ql.y:1451
		{
			yyVAL.path = expression.NewElement(yyS[yypt-3].path, yyS[yypt-1].expr)
		}
	case 187:
		yyVAL.expr = yyS[yypt-0].expr
	case 188:
		//line n1ql.y:1468
		{
			yyVAL.expr = expression.NewField(yyS[yypt-2].expr, expression.NewFieldName(yyS[yypt-0].s))
		}
	case 189:
		//line n1ql.y:1473
		{
			field := expression.NewField(yyS[yypt-2].expr, expression.NewFieldName(yyS[yypt-0].s))
			field.SetCaseInsensitive(true)
			yyVAL.expr = field
		}
	case 190:
		//line n1ql.y:1480
		{
			yyVAL.expr = expression.NewField(yyS[yypt-4].expr, yyS[yypt-1].expr)
		}
	case 191:
		//line n1ql.y:1485
		{
			field := expression.NewField(yyS[yypt-4].expr, yyS[yypt-1].expr)
			field.SetCaseInsensitive(true)
			yyVAL.expr = field
		}
	case 192:
		//line n1ql.y:1492
		{
			yyVAL.expr = expression.NewElement(yyS[yypt-3].expr, yyS[yypt-1].expr)
		}
	case 193:
		//line n1ql.y:1497
		{
			yyVAL.expr = expression.NewSlice(yyS[yypt-4].expr, yyS[yypt-2].expr)
		}
	case 194:
		//line n1ql.y:1502
		{
			yyVAL.expr = expression.NewSlice(yyS[yypt-5].expr, yyS[yypt-3].expr, yyS[yypt-1].expr)
		}
	case 195:
		//line n1ql.y:1508
		{
			yyVAL.expr = expression.NewAdd(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 196:
		//line n1ql.y:1513
		{
			yyVAL.expr = expression.NewSub(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 197:
		//line n1ql.y:1518
		{
			yyVAL.expr = expression.NewMult(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 198:
		//line n1ql.y:1523
		{
			yyVAL.expr = expression.NewDiv(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 199:
		//line n1ql.y:1528
		{
			yyVAL.expr = expression.NewMod(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 200:
		//line n1ql.y:1534
		{
			yyVAL.expr = expression.NewConcat(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 201:
		//line n1ql.y:1540
		{
			yyVAL.expr = expression.NewAnd(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 202:
		//line n1ql.y:1545
		{
			yyVAL.expr = expression.NewOr(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 203:
		//line n1ql.y:1550
		{
			yyVAL.expr = expression.NewNot(yyS[yypt-0].expr)
		}
	case 204:
		//line n1ql.y:1556
		{
			yyVAL.expr = expression.NewEq(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 205:
		//line n1ql.y:1561
		{
			yyVAL.expr = expression.NewEq(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 206:
		//line n1ql.y:1566
		{
			yyVAL.expr = expression.NewNE(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 207:
		//line n1ql.y:1571
		{
			yyVAL.expr = expression.NewLT(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 208:
		//line n1ql.y:1576
		{
			yyVAL.expr = expression.NewGT(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 209:
		//line n1ql.y:1581
		{
			yyVAL.expr = expression.NewLE(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 210:
		//line n1ql.y:1586
		{
			yyVAL.expr = expression.NewGE(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 211:
		//line n1ql.y:1591
		{
			yyVAL.expr = expression.NewBetween(yyS[yypt-4].expr, yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 212:
		//line n1ql.y:1596
		{
			yyVAL.expr = expression.NewNotBetween(yyS[yypt-5].expr, yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 213:
		//line n1ql.y:1601
		{
			yyVAL.expr = expression.NewLike(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 214:
		//line n1ql.y:1606
		{
			yyVAL.expr = expression.NewNotLike(yyS[yypt-3].expr, yyS[yypt-0].expr)
		}
	case 215:
		//line n1ql.y:1611
		{
			yyVAL.expr = expression.NewIn(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 216:
		//line n1ql.y:1616
		{
			yyVAL.expr = expression.NewNotIn(yyS[yypt-3].expr, yyS[yypt-0].expr)
		}
	case 217:
		//line n1ql.y:1621
		{
			yyVAL.expr = expression.NewWithin(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 218:
		//line n1ql.y:1626
		{
			yyVAL.expr = expression.NewNotWithin(yyS[yypt-3].expr, yyS[yypt-0].expr)
		}
	case 219:
		//line n1ql.y:1631
		{
			yyVAL.expr = expression.NewIsNull(yyS[yypt-2].expr)
		}
	case 220:
		//line n1ql.y:1636
		{
			yyVAL.expr = expression.NewIsNotNull(yyS[yypt-3].expr)
		}
	case 221:
		//line n1ql.y:1641
		{
			yyVAL.expr = expression.NewIsMissing(yyS[yypt-2].expr)
		}
	case 222:
		//line n1ql.y:1646
		{
			yyVAL.expr = expression.NewIsNotMissing(yyS[yypt-3].expr)
		}
	case 223:
		//line n1ql.y:1651
		{
			yyVAL.expr = expression.NewIsValued(yyS[yypt-2].expr)
		}
	case 224:
		//line n1ql.y:1656
		{
			yyVAL.expr = expression.NewIsNotValued(yyS[yypt-3].expr)
		}
	case 225:
		//line n1ql.y:1661
		{
			yyVAL.expr = expression.NewExists(yyS[yypt-0].expr)
		}
	case 226:
		yyVAL.expr = yyS[yypt-0].expr
	case 227:
		//line n1ql.y:1672
		{
			yyVAL.expr = expression.NewIdentifier(yyS[yypt-0].s)
		}
	case 228:
		yyVAL.expr = yyS[yypt-0].expr
	case 229:
		//line n1ql.y:1681
		{
			yyVAL.expr = expression.NewNeg(yyS[yypt-0].expr)
		}
	case 230:
		yyVAL.expr = yyS[yypt-0].expr
	case 231:
		yyVAL.expr = yyS[yypt-0].expr
	case 232:
		yyVAL.expr = yyS[yypt-0].expr
	case 233:
		yyVAL.expr = yyS[yypt-0].expr
	case 234:
		//line n1ql.y:1700
		{
			yyVAL.expr = expression.NewField(yyS[yypt-2].expr, expression.NewFieldName(yyS[yypt-0].s))
		}
	case 235:
		//line n1ql.y:1705
		{
			field := expression.NewField(yyS[yypt-2].expr, expression.NewFieldName(yyS[yypt-0].s))
			field.SetCaseInsensitive(true)
			yyVAL.expr = field
		}
	case 236:
		//line n1ql.y:1712
		{
			yyVAL.expr = expression.NewField(yyS[yypt-4].expr, yyS[yypt-1].expr)
		}
	case 237:
		//line n1ql.y:1717
		{
			field := expression.NewField(yyS[yypt-4].expr, yyS[yypt-1].expr)
			field.SetCaseInsensitive(true)
			yyVAL.expr = field
		}
	case 238:
		//line n1ql.y:1724
		{
			yyVAL.expr = expression.NewElement(yyS[yypt-3].expr, yyS[yypt-1].expr)
		}
	case 239:
		//line n1ql.y:1729
		{
			yyVAL.expr = expression.NewSlice(yyS[yypt-4].expr, yyS[yypt-2].expr)
		}
	case 240:
		//line n1ql.y:1734
		{
			yyVAL.expr = expression.NewSlice(yyS[yypt-5].expr, yyS[yypt-3].expr, yyS[yypt-1].expr)
		}
	case 241:
		//line n1ql.y:1740
		{
			yyVAL.expr = expression.NewAdd(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 242:
		//line n1ql.y:1745
		{
			yyVAL.expr = expression.NewSub(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 243:
		//line n1ql.y:1750
		{
			yyVAL.expr = expression.NewMult(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 244:
		//line n1ql.y:1755
		{
			yyVAL.expr = expression.NewDiv(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 245:
		//line n1ql.y:1760
		{
			yyVAL.expr = expression.NewMod(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 246:
		//line n1ql.y:1766
		{
			yyVAL.expr = expression.NewConcat(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 247:
		//line n1ql.y:1780
		{
			yyVAL.expr = expression.NULL_EXPR
		}
	case 248:
		//line n1ql.y:1785
		{
			yyVAL.expr = expression.FALSE_EXPR
		}
	case 249:
		//line n1ql.y:1790
		{
			yyVAL.expr = expression.TRUE_EXPR
		}
	case 250:
		//line n1ql.y:1795
		{
			yyVAL.expr = expression.NewConstant(value.NewValue(yyS[yypt-0].f))
		}
	case 251:
		//line n1ql.y:1800
		{
			yyVAL.expr = expression.NewConstant(value.NewValue(yyS[yypt-0].n))
		}
	case 252:
		//line n1ql.y:1805
		{
			yyVAL.expr = expression.NewConstant(value.NewValue(yyS[yypt-0].s))
		}
	case 253:
		yyVAL.expr = yyS[yypt-0].expr
	case 254:
		yyVAL.expr = yyS[yypt-0].expr
	case 255:
		//line n1ql.y:1816
		{
			yyVAL.expr = expression.NewObjectConstruct(yyS[yypt-1].bindings)
		}
	case 256:
		//line n1ql.y:1823
		{
			yyVAL.bindings = nil
		}
	case 257:
		yyVAL.bindings = yyS[yypt-0].bindings
	case 258:
		//line n1ql.y:1832
		{
			yyVAL.bindings = expression.Bindings{yyS[yypt-0].binding}
		}
	case 259:
		//line n1ql.y:1837
		{
			yyVAL.bindings = append(yyS[yypt-2].bindings, yyS[yypt-0].binding)
		}
	case 260:
		//line n1ql.y:1844
		{
			yyVAL.binding = expression.NewBinding(yyS[yypt-2].s, yyS[yypt-0].expr)
		}
	case 261:
		//line n1ql.y:1851
		{
			yyVAL.expr = expression.NewArrayConstruct(yyS[yypt-1].exprs...)
		}
	case 262:
		//line n1ql.y:1858
		{
			yyVAL.exprs = nil
		}
	case 263:
		yyVAL.exprs = yyS[yypt-0].exprs
	case 264:
		//line n1ql.y:1874
		{
			yyVAL.expr = yyS[yypt-1].expr
		}
	case 265:
		yyVAL.expr = yyS[yypt-0].expr
	case 266:
		yyVAL.expr = yyS[yypt-0].expr
	case 267:
		//line n1ql.y:1887
		{
			yyVAL.expr = expression.NewSimpleCase(yyS[yypt-2].expr, yyS[yypt-1].whenTerms, yyS[yypt-0].expr)
		}
	case 268:
		//line n1ql.y:1894
		{
			yyVAL.whenTerms = expression.WhenTerms{&expression.WhenTerm{yyS[yypt-2].expr, yyS[yypt-0].expr}}
		}
	case 269:
		//line n1ql.y:1899
		{
			yyVAL.whenTerms = append(yyS[yypt-4].whenTerms, &expression.WhenTerm{yyS[yypt-2].expr, yyS[yypt-0].expr})
		}
	case 270:
		//line n1ql.y:1907
		{
			yyVAL.expr = expression.NewSearchedCase(yyS[yypt-1].whenTerms, yyS[yypt-0].expr)
		}
	case 271:
		//line n1ql.y:1914
		{
			yyVAL.expr = nil
		}
	case 272:
		//line n1ql.y:1919
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 273:
		//line n1ql.y:1933
		{
			yyVAL.expr = nil
			f, ok := expression.GetFunction(yyS[yypt-3].s)
			if !ok && yylex.(*lexer).parsingStatement() {
				f, ok = algebra.GetAggregate(yyS[yypt-3].s, false)
			}

			if ok {
				if len(yyS[yypt-1].exprs) < f.MinArgs() || len(yyS[yypt-1].exprs) > f.MaxArgs() {
					yylex.Error(fmt.Sprintf("Wrong number of arguments to function %s.", yyS[yypt-3].s))
				} else {
					yyVAL.expr = f.Constructor()(yyS[yypt-1].exprs...)
				}
			} else {
				yylex.Error(fmt.Sprintf("Invalid function %s.", yyS[yypt-3].s))
			}
		}
	case 274:
		//line n1ql.y:1952
		{
			yyVAL.expr = nil
			if !yylex.(*lexer).parsingStatement() {
				yylex.Error("Cannot use aggregate as an inline expression.")
			} else {
				agg, ok := algebra.GetAggregate(yyS[yypt-4].s, true)
				if ok {
					yyVAL.expr = agg.Constructor()(yyS[yypt-1].expr)
				} else {
					yylex.Error(fmt.Sprintf("Invalid aggregate function %s.", yyS[yypt-4].s))
				}
			}
		}
	case 275:
		//line n1ql.y:1967
		{
			yyVAL.expr = nil
			if !yylex.(*lexer).parsingStatement() {
				yylex.Error("Cannot use aggregate as an inline expression.")
			} else {
				agg, ok := algebra.GetAggregate(yyS[yypt-3].s, false)
				if ok {
					yyVAL.expr = agg.Constructor()(nil)
				} else {
					yylex.Error(fmt.Sprintf("Invalid aggregate function %s.", yyS[yypt-3].s))
				}
			}
		}
	case 276:
		yyVAL.s = yyS[yypt-0].s
	case 277:
		yyVAL.expr = yyS[yypt-0].expr
	case 278:
		yyVAL.expr = yyS[yypt-0].expr
	case 279:
		//line n1ql.y:2001
		{
			yyVAL.expr = expression.NewAny(yyS[yypt-2].bindings, yyS[yypt-1].expr)
		}
	case 280:
		//line n1ql.y:2006
		{
			yyVAL.expr = expression.NewAny(yyS[yypt-2].bindings, yyS[yypt-1].expr)
		}
	case 281:
		//line n1ql.y:2011
		{
			yyVAL.expr = expression.NewEvery(yyS[yypt-2].bindings, yyS[yypt-1].expr)
		}
	case 282:
		//line n1ql.y:2018
		{
			yyVAL.bindings = expression.Bindings{yyS[yypt-0].binding}
		}
	case 283:
		//line n1ql.y:2023
		{
			yyVAL.bindings = append(yyS[yypt-2].bindings, yyS[yypt-0].binding)
		}
	case 284:
		//line n1ql.y:2030
		{
			yyVAL.binding = expression.NewBinding(yyS[yypt-2].s, yyS[yypt-0].expr)
		}
	case 285:
		//line n1ql.y:2035
		{
			yyVAL.binding = expression.NewDescendantBinding(yyS[yypt-2].s, yyS[yypt-0].expr)
		}
	case 286:
		//line n1ql.y:2042
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 287:
		//line n1ql.y:2049
		{
			yyVAL.expr = expression.NewArray(yyS[yypt-4].expr, yyS[yypt-2].bindings, yyS[yypt-1].expr)
		}
	case 288:
		//line n1ql.y:2054
		{
			yyVAL.expr = expression.NewFirst(yyS[yypt-4].expr, yyS[yypt-2].bindings, yyS[yypt-1].expr)
		}
	case 289:
		//line n1ql.y:2068
		{
			yyVAL.expr = yyS[yypt-1].expr
		}
	case 290:
		yyVAL.expr = yyS[yypt-0].expr
	case 291:
		//line n1ql.y:2077
		{
			yyVAL.expr = nil
			if yylex.(*lexer).parsingStatement() {
				yyVAL.expr = algebra.NewSubquery(yyS[yypt-0].fullselect)
			} else {
				yylex.Error("Cannot use subquery as an inline expression.")
			}
		}
	}
	goto yystack /* stack new state and value */
}
