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
const IDENTIFIER = 57497
const STRING = 57498
const LPAREN = 57499
const RPAREN = 57500
const LBRACE = 57501
const RBRACE = 57502
const LBRACKET = 57503
const RBRACKET = 57504
const COMMA = 57505
const COLON = 57506
const EQ = 57507
const DEQ = 57508
const NE = 57509
const LT = 57510
const GT = 57511
const LE = 57512
const GE = 57513
const CONCAT = 57514
const PLUS = 57515
const STAR = 57516
const DIV = 57517
const MOD = 57518
const UMINUS = 57519
const DOT = 57520

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
	"IDENTIFIER",
	"STRING",
	"LPAREN",
	"RPAREN",
	"LBRACE",
	"RBRACE",
	"LBRACKET",
	"RBRACKET",
	"COMMA",
	"COLON",
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
	157, 270,
	-2, 223,
	-1, 104,
	164, 63,
	-2, 64,
	-1, 140,
	50, 72,
	67, 72,
	85, 72,
	135, 72,
	-2, 50,
	-1, 166,
	165, 0,
	166, 0,
	167, 0,
	-2, 200,
	-1, 167,
	165, 0,
	166, 0,
	167, 0,
	-2, 201,
	-1, 168,
	165, 0,
	166, 0,
	167, 0,
	-2, 202,
	-1, 169,
	168, 0,
	169, 0,
	170, 0,
	171, 0,
	-2, 203,
	-1, 170,
	168, 0,
	169, 0,
	170, 0,
	171, 0,
	-2, 204,
	-1, 171,
	168, 0,
	169, 0,
	170, 0,
	171, 0,
	-2, 205,
	-1, 172,
	168, 0,
	169, 0,
	170, 0,
	171, 0,
	-2, 206,
	-1, 179,
	75, 0,
	-2, 209,
	-1, 180,
	58, 0,
	150, 0,
	-2, 211,
	-1, 181,
	58, 0,
	150, 0,
	-2, 213,
	-1, 274,
	75, 0,
	-2, 210,
	-1, 275,
	58, 0,
	150, 0,
	-2, 212,
	-1, 276,
	58, 0,
	150, 0,
	-2, 214,
}

const yyNprod = 286
const yyPrivate = 57344

var yyTokenNames []string
var yyStates []string

const yyLast = 2502

var yyAct = []int{

	154, 3, 560, 548, 424, 558, 549, 295, 296, 186,
	88, 89, 499, 508, 291, 203, 383, 515, 299, 243,
	129, 475, 204, 91, 399, 250, 205, 220, 385, 438,
	288, 199, 146, 382, 329, 149, 125, 244, 407, 62,
	12, 102, 141, 371, 150, 110, 127, 128, 114, 222,
	251, 122, 316, 48, 126, 215, 8, 440, 133, 134,
	314, 334, 483, 454, 453, 315, 253, 157, 158, 159,
	160, 161, 162, 163, 164, 165, 166, 167, 168, 169,
	170, 171, 172, 266, 415, 179, 180, 181, 115, 415,
	252, 233, 87, 436, 266, 66, 269, 270, 271, 400,
	265, 414, 131, 132, 66, 143, 414, 126, 68, 155,
	400, 265, 65, 217, 268, 156, 496, 69, 70, 71,
	502, 65, 228, 202, 333, 230, 254, 232, 350, 437,
	435, 366, 230, 227, 355, 229, 226, 155, 232, 174,
	356, 144, 240, 156, 478, 307, 68, 189, 191, 193,
	258, 230, 456, 305, 457, 206, 218, 261, 103, 445,
	224, 224, 106, 410, 245, 123, 144, 415, 345, 225,
	260, 130, 104, 207, 390, 221, 302, 274, 275, 276,
	255, 257, 104, 256, 414, 242, 216, 66, 552, 283,
	497, 559, 234, 266, 104, 554, 289, 104, 72, 67,
	69, 70, 71, 298, 65, 267, 269, 270, 271, 500,
	265, 306, 293, 175, 242, 309, 173, 310, 542, 304,
	201, 63, 142, 480, 538, 66, 135, 573, 572, 318,
	294, 319, 174, 303, 322, 323, 324, 67, 69, 70,
	71, 568, 65, 332, 284, 278, 285, 184, 286, 277,
	183, 182, 539, 335, 297, 340, 529, 349, 177, 64,
	426, 447, 300, 523, 353, 342, 343, 357, 344, 308,
	231, 298, 336, 111, 509, 176, 124, 223, 223, 317,
	321, 117, 498, 365, 235, 327, 328, 64, 521, 442,
	337, 313, 214, 374, 312, 194, 282, 566, 63, 348,
	207, 377, 379, 380, 378, 279, 570, 185, 569, 563,
	192, 190, 393, 373, 101, 386, 564, 388, 530, 188,
	137, 174, 116, 248, 174, 174, 174, 174, 174, 174,
	537, 372, 534, 249, 376, 405, 387, 375, 292, 412,
	339, 519, 396, 63, 398, 95, 143, 389, 520, 105,
	178, 99, 98, 401, 224, 224, 576, 419, 63, 63,
	245, 301, 394, 395, 64, 246, 94, 289, 575, 402,
	406, 404, 416, 417, 428, 413, 411, 427, 409, 409,
	429, 430, 550, 209, 213, 432, 61, 431, 441, 433,
	434, 219, 273, 444, 119, 97, 118, 423, 506, 331,
	63, 449, 236, 237, 126, 100, 513, 450, 448, 64,
	326, 346, 347, 196, 197, 198, 458, 325, 320, 212,
	208, 174, 463, 571, 64, 64, 533, 403, 455, 364,
	195, 443, 459, 460, 452, 93, 467, 472, 469, 470,
	451, 2, 468, 341, 338, 1, 126, 446, 541, 139,
	522, 545, 553, 90, 386, 439, 384, 477, 487, 465,
	381, 476, 466, 142, 474, 425, 473, 492, 484, 471,
	464, 223, 223, 493, 397, 290, 34, 33, 32, 18,
	17, 354, 479, 16, 358, 359, 360, 361, 362, 363,
	15, 489, 490, 14, 13, 408, 408, 7, 6, 495,
	5, 501, 494, 507, 268, 4, 367, 524, 503, 518,
	245, 510, 511, 264, 516, 516, 517, 476, 514, 368,
	280, 281, 187, 528, 287, 92, 526, 527, 525, 532,
	96, 73, 145, 505, 543, 544, 531, 82, 504, 482,
	535, 536, 481, 330, 540, 546, 547, 241, 136, 200,
	551, 561, 247, 555, 557, 556, 562, 138, 73, 140,
	59, 206, 60, 565, 82, 26, 113, 25, 567, 43,
	21, 46, 45, 24, 109, 574, 561, 561, 578, 579,
	577, 422, 85, 266, 108, 73, 107, 23, 268, 369,
	87, 82, 120, 121, 272, 267, 269, 270, 271, 84,
	265, 42, 41, 19, 0, 0, 68, 0, 0, 85,
	83, 370, 485, 486, 0, 0, 74, 87, 0, 0,
	0, 0, 0, 0, 0, 0, 84, 0, 0, 0,
	0, 0, 0, 68, 0, 0, 85, 83, 0, 0,
	0, 0, 0, 74, 87, 0, 0, 0, 0, 0,
	0, 0, 0, 84, 0, 0, 0, 0, 0, 0,
	68, 0, 0, 0, 83, 0, 0, 266, 0, 0,
	74, 0, 0, 0, 86, 0, 0, 0, 272, 267,
	269, 270, 271, 0, 265, 66, 420, 0, 421, 75,
	76, 77, 78, 79, 80, 81, 72, 67, 69, 70,
	71, 86, 65, 0, 73, 0, 207, 0, 0, 0,
	82, 0, 66, 0, 0, 0, 75, 76, 77, 78,
	79, 80, 81, 72, 67, 69, 70, 71, 86, 65,
	0, 73, 268, 0, 206, 0, 0, 82, 0, 66,
	0, 0, 0, 75, 76, 77, 78, 79, 80, 81,
	72, 67, 69, 70, 71, 85, 65, 0, 0, 0,
	0, 73, 0, 87, 0, 0, 0, 82, 0, 0,
	0, 0, 84, 0, 0, 0, 0, 0, 0, 68,
	0, 0, 85, 83, 0, 0, 0, 0, 0, 74,
	87, 0, 0, 0, 0, 0, 0, 0, 0, 84,
	0, 0, 0, 0, 0, 0, 68, 0, 0, 0,
	83, 266, 85, 0, 0, 0, 74, 0, 0, 0,
	87, 0, 272, 267, 269, 270, 271, 0, 265, 84,
	0, 0, 0, 0, 0, 0, 68, 0, 0, 0,
	83, 0, 0, 0, 0, 0, 74, 86, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 66, 262,
	0, 263, 75, 76, 77, 78, 79, 80, 81, 72,
	67, 69, 70, 71, 86, 65, 0, 0, 0, 207,
	73, 0, 0, 0, 0, 66, 82, 0, 0, 75,
	76, 77, 78, 79, 80, 81, 72, 67, 69, 70,
	71, 242, 259, 0, 86, 0, 0, 73, 0, 0,
	0, 0, 0, 82, 0, 66, 0, 0, 0, 75,
	76, 77, 78, 79, 80, 81, 72, 67, 69, 70,
	71, 85, 65, 0, 73, 0, 0, 0, 0, 87,
	82, 0, 0, 0, 0, 0, 0, 0, 84, 0,
	0, 440, 0, 0, 0, 68, 0, 0, 85, 83,
	0, 0, 0, 0, 0, 74, 87, 0, 0, 0,
	0, 0, 0, 0, 0, 84, 0, 0, 51, 0,
	0, 0, 68, 0, 0, 85, 83, 0, 0, 0,
	0, 0, 74, 87, 0, 0, 0, 0, 0, 49,
	0, 0, 84, 0, 29, 0, 0, 0, 0, 68,
	50, 0, 0, 83, 0, 0, 0, 0, 0, 74,
	11, 0, 0, 86, 0, 63, 0, 0, 0, 0,
	0, 512, 0, 0, 66, 0, 27, 0, 75, 76,
	77, 78, 79, 80, 81, 72, 67, 69, 70, 71,
	86, 65, 0, 73, 31, 0, 0, 0, 0, 82,
	0, 66, 0, 0, 0, 75, 76, 77, 78, 79,
	80, 81, 72, 67, 69, 70, 71, 86, 65, 0,
	73, 0, 0, 0, 0, 0, 82, 0, 66, 491,
	0, 64, 75, 76, 77, 78, 79, 80, 81, 72,
	67, 69, 70, 71, 85, 65, 0, 73, 0, 0,
	30, 28, 87, 82, 0, 0, 0, 0, 0, 0,
	0, 84, 0, 0, 0, 0, 0, 0, 68, 0,
	0, 85, 83, 0, 0, 0, 0, 0, 74, 87,
	0, 0, 0, 0, 0, 0, 0, 0, 84, 0,
	0, 0, 0, 0, 0, 68, 0, 0, 85, 83,
	0, 0, 0, 0, 0, 74, 87, 0, 0, 0,
	0, 0, 0, 0, 0, 84, 0, 0, 0, 0,
	0, 0, 68, 0, 0, 0, 83, 0, 0, 0,
	0, 0, 74, 0, 0, 0, 86, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 66, 488, 0,
	0, 75, 76, 77, 78, 79, 80, 81, 72, 67,
	69, 70, 71, 86, 65, 0, 73, 0, 0, 0,
	0, 0, 82, 0, 66, 461, 0, 0, 75, 76,
	77, 78, 79, 80, 81, 72, 67, 69, 70, 71,
	86, 65, 0, 73, 0, 0, 0, 0, 0, 82,
	0, 66, 418, 0, 0, 75, 76, 77, 78, 79,
	80, 81, 72, 67, 69, 70, 71, 85, 65, 0,
	73, 0, 0, 0, 0, 87, 82, 0, 0, 0,
	0, 0, 0, 0, 84, 0, 0, 0, 0, 0,
	0, 68, 0, 0, 85, 83, 0, 0, 0, 0,
	0, 74, 87, 0, 0, 0, 0, 0, 0, 0,
	0, 84, 0, 0, 0, 0, 0, 0, 68, 0,
	0, 85, 83, 0, 0, 0, 0, 0, 74, 87,
	0, 0, 0, 0, 392, 0, 0, 0, 84, 0,
	0, 0, 0, 0, 0, 68, 0, 0, 0, 83,
	0, 0, 0, 0, 0, 74, 0, 0, 0, 86,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	66, 0, 0, 0, 75, 76, 77, 78, 79, 80,
	81, 72, 67, 69, 70, 71, 86, 65, 0, 73,
	0, 0, 0, 0, 391, 82, 0, 66, 0, 0,
	0, 75, 76, 77, 78, 79, 80, 81, 72, 67,
	69, 70, 71, 86, 65, 0, 73, 0, 0, 0,
	0, 0, 82, 0, 66, 351, 0, 0, 75, 76,
	77, 78, 79, 80, 81, 72, 67, 69, 70, 71,
	85, 65, 0, 73, 0, 0, 0, 0, 87, 82,
	0, 0, 0, 0, 0, 0, 0, 84, 0, 0,
	239, 0, 0, 0, 68, 0, 0, 85, 83, 0,
	0, 0, 0, 0, 74, 87, 0, 0, 0, 0,
	0, 0, 0, 0, 84, 0, 0, 238, 0, 0,
	0, 68, 0, 0, 85, 83, 0, 0, 0, 0,
	0, 74, 87, 0, 0, 0, 0, 311, 0, 0,
	0, 84, 0, 0, 0, 0, 0, 0, 68, 0,
	0, 0, 83, 0, 0, 0, 0, 0, 74, 0,
	0, 0, 86, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 66, 0, 0, 0, 75, 76, 77,
	78, 79, 80, 81, 72, 67, 69, 70, 71, 86,
	65, 0, 73, 0, 0, 0, 0, 0, 82, 0,
	66, 0, 0, 0, 75, 76, 77, 78, 79, 80,
	81, 72, 67, 69, 70, 71, 86, 65, 0, 73,
	0, 0, 0, 0, 0, 82, 0, 66, 0, 0,
	0, 75, 76, 77, 78, 79, 80, 81, 72, 67,
	69, 70, 71, 85, 65, 0, 73, 0, 0, 0,
	0, 87, 82, 0, 0, 0, 0, 0, 0, 0,
	84, 0, 0, 0, 0, 0, 0, 68, 0, 0,
	85, 83, 0, 0, 0, 0, 0, 74, 87, 0,
	0, 0, 0, 0, 0, 0, 0, 84, 0, 0,
	0, 0, 0, 0, 68, 0, 0, 85, 83, 0,
	0, 0, 0, 0, 74, 87, 0, 0, 0, 0,
	0, 0, 0, 0, 84, 0, 0, 0, 0, 0,
	0, 68, 82, 0, 0, 83, 0, 0, 0, 0,
	0, 112, 0, 0, 0, 86, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 66, 0, 0, 0,
	75, 76, 77, 78, 79, 80, 81, 72, 67, 69,
	70, 71, 86, 65, 0, 0, 0, 85, 0, 0,
	0, 82, 0, 66, 0, 87, 0, 75, 76, 77,
	78, 79, 80, 81, 72, 67, 69, 70, 71, 86,
	65, 68, 0, 0, 0, 0, 0, 0, 0, 0,
	66, 0, 0, 0, 75, 76, 77, 78, 79, 80,
	81, 72, 67, 69, 70, 71, 85, 65, 148, 0,
	0, 0, 54, 57, 87, 0, 0, 0, 0, 0,
	0, 0, 44, 84, 0, 0, 0, 0, 0, 0,
	68, 0, 0, 0, 83, 0, 0, 0, 0, 147,
	0, 0, 0, 152, 0, 0, 56, 0, 0, 86,
	10, 0, 36, 58, 0, 0, 0, 0, 0, 0,
	66, 0, 0, 0, 0, 0, 54, 57, 0, 0,
	0, 72, 67, 69, 70, 71, 44, 65, 0, 0,
	0, 0, 0, 0, 0, 82, 22, 0, 0, 0,
	9, 35, 0, 0, 0, 0, 0, 152, 86, 0,
	56, 0, 0, 0, 10, 0, 36, 58, 0, 66,
	151, 0, 0, 75, 76, 77, 78, 79, 80, 81,
	72, 67, 69, 70, 71, 55, 65, 0, 0, 0,
	85, 0, 0, 37, 0, 0, 0, 0, 87, 0,
	22, 0, 54, 57, 9, 35, 0, 84, 0, 0,
	0, 0, 44, 0, 68, 0, 0, 39, 38, 20,
	40, 47, 0, 52, 151, 53, 0, 0, 0, 210,
	0, 0, 0, 0, 0, 0, 56, 0, 153, 55,
	10, 0, 36, 58, 0, 0, 0, 37, 0, 0,
	54, 57, 0, 0, 0, 0, 0, 0, 0, 0,
	44, 0, 0, 0, 0, 0, 0, 0, 0, 82,
	0, 39, 38, 20, 40, 47, 22, 52, 0, 53,
	9, 35, 86, 0, 56, 0, 0, 0, 10, 0,
	36, 58, 153, 66, 0, 0, 0, 0, 0, 0,
	78, 79, 80, 81, 72, 67, 69, 70, 71, 0,
	65, 0, 0, 0, 85, 55, 0, 0, 0, 0,
	0, 0, 87, 37, 22, 0, 0, 0, 9, 35,
	0, 84, 0, 51, 0, 0, 54, 57, 68, 0,
	0, 0, 0, 0, 0, 0, 44, 39, 38, 20,
	40, 47, 0, 52, 49, 53, 0, 0, 0, 29,
	0, 0, 0, 55, 0, 50, 0, 0, 211, 0,
	56, 37, 0, 0, 10, 11, 36, 58, 0, 0,
	63, 0, 0, 0, 54, 57, 0, 0, 0, 0,
	0, 27, 0, 0, 44, 39, 38, 20, 40, 47,
	0, 52, 0, 53, 0, 0, 86, 0, 0, 31,
	22, 0, 0, 0, 9, 35, 153, 66, 56, 0,
	0, 0, 10, 0, 36, 58, 0, 0, 72, 67,
	69, 70, 71, 0, 65, 54, 57, 0, 0, 0,
	0, 0, 0, 0, 0, 44, 64, 0, 0, 55,
	0, 0, 0, 0, 0, 0, 0, 37, 22, 0,
	0, 0, 9, 35, 0, 30, 28, 0, 0, 56,
	0, 0, 0, 10, 0, 36, 58, 0, 0, 0,
	0, 39, 38, 20, 40, 47, 0, 52, 0, 53,
	0, 0, 0, 0, 0, 0, 0, 55, 0, 0,
	0, 0, 0, 0, 0, 37, 0, 0, 0, 22,
	0, 0, 0, 9, 35, 0, 0, 54, 57, 0,
	0, 0, 0, 0, 0, 0, 0, 44, 0, 39,
	38, 20, 40, 47, 0, 52, 0, 53, 462, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 55, 0,
	0, 56, 0, 0, 0, 10, 37, 36, 58, 0,
	0, 63, 0, 54, 57, 0, 0, 0, 0, 0,
	0, 0, 0, 44, 0, 54, 57, 0, 0, 0,
	39, 38, 20, 40, 47, 44, 52, 0, 53, 352,
	0, 22, 0, 0, 0, 9, 35, 56, 0, 0,
	0, 10, 0, 36, 58, 0, 0, 0, 0, 56,
	0, 0, 0, 10, 0, 36, 58, 0, 54, 57,
	0, 0, 0, 0, 0, 0, 0, 64, 44, 0,
	55, 0, 0, 0, 0, 0, 0, 22, 37, 0,
	0, 9, 35, 0, 0, 0, 0, 0, 0, 22,
	0, 0, 56, 9, 35, 0, 0, 0, 36, 58,
	0, 0, 39, 38, 20, 40, 47, 0, 52, 0,
	53, 0, 0, 0, 0, 0, 55, 0, 0, 0,
	0, 0, 0, 0, 37, 0, 0, 0, 55, 0,
	0, 0, 22, 0, 0, 0, 37, 35, 0, 0,
	0, 112, 0, 0, 0, 0, 0, 0, 39, 38,
	20, 40, 47, 0, 52, 0, 53, 0, 0, 0,
	39, 38, 20, 40, 47, 0, 52, 0, 53, 0,
	0, 55, 0, 0, 0, 0, 0, 0, 0, 37,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 39, 38, 20, 40, 47, 0, 52,
	0, 53,
}
var yyPact = []int{

	2058, -1000, -1000, 1592, -1000, -1000, -1000, -1000, -1000, 2297,
	2297, 973, -1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000,
	-1000, -1000, 2297, -1000, -1000, -1000, 302, 287, 286, 353,
	27, 284, -1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000,
	-1000, -1000, -1000, 5, 2285, -1000, -1000, 2239, -1000, 221,
	336, 334, 9, 2297, 16, 16, 16, 2297, 2297, -1000,
	-1000, 247, 348, 42, 1794, -18, 2297, 2297, 2297, 2297,
	2297, 2297, 2297, 2297, 2297, 2297, 2297, 2297, 2297, 2297,
	2297, 2297, 2340, 200, 2297, 2297, 2297, 164, 1738, 26,
	-1000, -66, 243, 307, 306, 291, -1000, 414, 27, 27,
	27, 81, -41, 145, -1000, 27, 1924, 378, -1000, -1000,
	1565, 146, 2297, -2, 1592, -1000, 331, 20, 27, 27,
	-24, -30, -1000, -42, -27, -31, 1592, -25, -1000, 134,
	-1000, -25, -25, 1446, 1419, 38, -1000, 18, 247, -1000,
	261, -1000, -128, -74, -98, -1000, -37, 1848, 1972, 2297,
	-1000, -1000, -1000, -1000, 724, -1000, 2297, 697, -57, -57,
	-66, -66, -66, 64, 1738, 1619, 1862, 1862, 1862, 1986,
	1986, 1986, 1986, 506, -1000, 2340, 2297, 2297, 2297, 1689,
	26, 26, -1000, 162, -1000, -1000, 207, -1000, 2297, -1000,
	246, -1000, 246, -1000, 246, 2297, 270, 270, 81, 135,
	-1000, 161, 21, -1000, -1000, -1000, 18, -1000, 78, -5,
	2297, -13, -1000, 146, 2297, -1000, 2297, 1392, -1000, 204,
	201, -1000, -118, -1000, -99, -126, -1000, 9, 2297, -1000,
	2297, 377, 16, 2297, 2297, 2297, 376, 369, 16, 16,
	344, -1000, 2297, -39, -1000, -104, 38, 205, -1000, 171,
	145, 13, 21, 21, 1972, -37, 2297, -37, 1592, -46,
	-1000, 1273, -1000, 2157, 2340, -21, 2297, 2340, 2340, 2340,
	2340, 2340, 2340, 422, 1689, 26, 26, -1000, -1000, -1000,
	-1000, -1000, 2297, 1592, -1000, -1000, -1000, -32, -1000, 578,
	169, -1000, 2297, 169, 38, 67, 38, 13, 13, 267,
	-1000, 145, -1000, -1000, 17, -1000, 1246, -1000, -1000, 1219,
	1592, 2297, 27, 27, 20, 21, 20, -1000, 1592, 1592,
	-1000, -1000, 1592, 1592, 1592, -1000, -1000, -36, -36, 141,
	-1000, 411, 1592, 18, 2297, 344, 39, 39, 2297, -1000,
	-1000, -1000, -1000, 81, -77, -1000, -128, -128, -1000, 1592,
	-1000, -1000, -1000, 1100, 650, -1000, 2297, 524, -78, -78,
	-67, -67, -67, 32, 2340, 1592, 2297, -1000, -1000, -1000,
	-1000, 149, 149, 2297, 1592, 149, 149, 243, 38, 243,
	243, -33, -1000, -72, -34, -1000, 6, 2297, -1000, 199,
	246, -1000, 2297, 1592, -1000, 2, -1000, -1000, 152, 367,
	2297, 366, -1000, 2297, -1000, 1592, -1000, -1000, -128, -100,
	-101, -1000, 551, -1000, -3, 2297, 145, 145, -1000, 1073,
	-1000, 2106, 650, -1000, -1000, -1000, 1848, -1000, 1592, -1000,
	-1000, 149, 243, 149, 149, 13, 2297, 13, -1000, -1000,
	16, 1592, 270, -14, 1592, 2297, -1000, 97, -1000, 1592,
	-1000, -12, 145, 21, 21, -1000, -1000, 2297, 1046, 81,
	81, -1000, -1000, 927, -1000, -37, 2297, -1000, 149, -1000,
	-1000, -1000, 900, -1000, -47, -1000, 132, 63, 145, -38,
	20, 342, -1000, 18, 184, -128, -128, 873, -1000, -1000,
	-1000, -1000, 1592, -1000, -1000, 365, 16, 13, 13, 243,
	262, 198, 167, -1000, -1000, -1000, 2297, -39, -1000, 161,
	145, 145, -1000, -1000, -1000, -1000, -77, -1000, 149, 131,
	239, 270, 38, 410, 1592, 263, 184, 184, -1000, 193,
	127, 63, 77, 2297, 2297, -1000, -1000, 135, 38, 319,
	243, -1000, 43, 1592, 1592, 49, 67, 38, 45, -1000,
	2297, 149, -1000, -1000, 230, -1000, 38, -1000, -1000, 211,
	-1000, 754, -1000, 116, 229, -1000, 227, -1000, 392, 103,
	102, 38, 305, 293, 45, 2297, 2297, -1000, -1000, -1000,
}
var yyPgo = []int{

	0, 603, 602, 601, 51, 593, 592, 0, 56, 216,
	36, 276, 37, 19, 26, 22, 15, 20, 587, 586,
	584, 574, 55, 273, 573, 572, 571, 47, 46, 270,
	24, 570, 569, 567, 566, 40, 565, 53, 562, 560,
	559, 386, 557, 42, 38, 552, 16, 25, 41, 158,
	549, 31, 13, 226, 548, 6, 547, 34, 543, 542,
	539, 538, 533, 44, 32, 532, 39, 530, 525, 30,
	524, 522, 9, 521, 520, 519, 506, 441, 505, 500,
	498, 497, 494, 493, 490, 483, 480, 479, 478, 477,
	476, 314, 43, 14, 475, 470, 465, 4, 21, 464,
	17, 7, 33, 460, 8, 28, 456, 455, 29, 12,
	452, 451, 3, 2, 5, 27, 49, 450, 448, 447,
	445, 35, 444, 18, 443,
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
	119, 46, 46, 46, 46, 7, 7, 7, 7, 7,
	7, 7, 7, 7, 7, 7, 7, 7, 7, 7,
	7, 7, 7, 7, 7, 7, 7, 7, 7, 7,
	7, 7, 7, 7, 7, 7, 7, 7, 7, 7,
	7, 7, 8, 8, 8, 8, 8, 8, 8, 9,
	9, 9, 9, 9, 9, 9, 9, 9, 9, 9,
	9, 1, 1, 1, 1, 1, 1, 1, 1, 2,
	6, 6, 5, 5, 4, 3, 11, 11, 18, 19,
	19, 20, 23, 23, 21, 22, 22, 31, 31, 31,
	32, 24, 24, 25, 25, 25, 28, 28, 27, 27,
	29, 26, 26, 33, 34, 34,
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
	3, 1, 3, 5, 4, 1, 3, 5, 4, 5,
	6, 3, 3, 3, 3, 3, 3, 3, 3, 2,
	3, 3, 3, 3, 3, 3, 3, 5, 6, 3,
	4, 3, 4, 3, 4, 3, 4, 3, 4, 3,
	4, 2, 1, 1, 1, 2, 1, 1, 1, 1,
	3, 5, 4, 5, 6, 3, 3, 3, 3, 3,
	3, 1, 1, 1, 1, 1, 1, 1, 1, 3,
	0, 1, 1, 3, 3, 3, 0, 1, 3, 1,
	1, 3, 4, 5, 2, 0, 2, 4, 5, 4,
	1, 1, 1, 4, 4, 4, 1, 3, 3, 3,
	2, 6, 6, 3, 1, 1,
}
var yyChk = []int{

	-1000, -120, -77, -7, -78, -79, -80, -81, -8, 86,
	46, 47, -35, -82, -83, -84, -85, -86, -87, -1,
	155, -31, 82, -18, -24, -33, -36, 63, 138, 31,
	137, 81, -88, -89, -90, 87, 48, 129, 154, 153,
	156, -2, -3, -32, 18, -25, -26, 157, -37, 26,
	37, 5, 159, 161, 8, 121, 42, 9, 49, -39,
	-38, -41, -66, 52, 118, 178, 161, 173, 82, 174,
	175, 176, 172, 7, 92, 165, 166, 167, 168, 169,
	170, 171, 13, 86, 75, 58, 150, 66, -7, -7,
	-77, -7, -68, 133, 64, 43, -67, 93, 65, 65,
	52, -91, -48, -49, 155, 65, 157, -19, -20, -21,
	-7, -23, 146, -34, -7, -35, 101, 60, 60, 60,
	-6, -5, -4, 156, -11, -10, -7, -28, -27, -17,
	155, -28, -28, -7, -7, -53, -54, 73, -42, -41,
	-40, -43, -49, -48, 124, -65, -64, 35, 4, -121,
	-63, 106, 39, 174, -7, 155, 161, -7, -7, -7,
	-7, -7, -7, -7, -7, -7, -7, -7, -7, -7,
	-7, -7, -7, -9, -8, 13, 75, 58, 150, -7,
	-7, -7, 87, 86, 83, 143, -72, -71, 76, -37,
	4, -37, 4, -37, 4, 16, -91, -91, -91, -51,
	-50, 139, 164, -16, -15, -14, 10, 155, -91, -11,
	35, 174, 41, -23, 146, -22, 40, -7, 158, 60,
	-115, 155, -116, -49, -48, -116, 160, 163, 164, 162,
	163, -29, 163, 116, 58, 150, -29, -29, 51, 51,
	-55, -56, 147, -13, -12, -14, -53, -45, 62, 72,
	-47, 178, 164, 164, 163, -64, -121, -64, -7, 178,
	-16, -7, 162, 164, 7, 178, 161, 173, 82, 174,
	175, 176, 172, -9, -7, -7, -7, 87, 83, 143,
	-74, -73, 89, -7, -37, -37, -37, -70, -69, -7,
	-94, -93, 68, -93, -51, -101, -104, 119, 136, -123,
	101, -49, 155, -14, 141, 158, -7, 158, -22, -7,
	-7, 125, 90, 90, 178, 164, 178, -4, -7, -7,
	41, -27, -7, -7, -7, 41, 41, -28, -28, -57,
	-58, 55, -7, 163, 165, -55, 67, 85, -122, 135,
	50, -124, 94, -16, -46, 155, -49, -49, -63, -7,
	174, 162, 162, -7, -9, 155, 161, -7, -9, -9,
	-9, -9, -9, -9, 7, -7, 163, -76, -75, 11,
	33, -92, -35, 144, -7, -92, -35, -55, -104, -55,
	-55, -103, -102, -46, -106, -105, -46, 69, -16, -43,
	157, 158, 125, -7, -116, -116, -115, -49, -115, -30,
	146, -30, -66, 16, -12, -7, -57, -44, -49, -48,
	124, -44, -7, -51, 178, 161, -47, -47, 162, -7,
	162, 164, -9, -69, -97, -96, 111, -97, -7, -97,
	-97, -72, -55, -72, -72, 163, 165, 163, -108, -107,
	51, -7, 90, -35, -7, 157, -119, 109, 41, -7,
	41, -10, -47, 164, 164, -16, 155, 157, -7, -16,
	-16, 162, 162, -7, -95, -64, -121, -97, -72, -97,
	-97, -102, -7, -105, -99, -98, -17, -93, 158, -10,
	126, -59, -60, 74, -16, -49, -49, -7, 162, -51,
	-51, 162, -7, -97, -108, -30, 163, 58, 150, -109,
	146, -15, 158, -115, -61, -62, 56, -13, -52, 90,
	-47, -47, 158, 41, -98, -100, -46, -100, -72, 79,
	86, 90, -117, 96, -7, -123, -16, -16, -97, 125,
	79, -93, -55, 16, 69, -52, -52, 137, 31, 125,
	-109, -118, 141, -7, -7, -111, -101, -104, -112, -55,
	63, -72, 145, -110, 146, -55, -104, -55, -114, 146,
	-113, -7, -97, 79, 86, -55, 86, -55, 125, 79,
	79, 31, 125, 125, -112, 63, 63, -114, -113, -113,
}
var yyDef = []int{

	0, -2, 1, 2, 3, 4, 5, 6, 185, 0,
	0, 0, 8, 9, 10, 11, 12, 13, 14, 222,
	-2, 224, 0, 226, 227, 228, 98, 0, 0, 0,
	0, 0, 15, 16, 17, 241, 242, 243, 244, 245,
	246, 247, 248, 0, 0, 271, 272, 0, 19, 0,
	0, 0, 250, 256, 0, 0, 0, 0, 0, 26,
	27, 78, 48, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 199, 221,
	7, 225, 108, 0, 0, 0, 99, 0, 0, 0,
	0, 67, 0, 43, -2, 0, 256, 0, 259, 260,
	0, 265, 0, 0, 284, 285, 0, 0, 0, 0,
	0, 251, 252, 0, 0, 257, 90, 0, 276, 0,
	144, 0, 0, 0, 0, 84, 79, 0, 78, 49,
	-2, 51, 65, 0, 0, 30, 31, 0, 0, 0,
	38, 36, 37, 40, 43, 186, 0, 0, 191, 192,
	193, 194, 195, 196, 197, 198, -2, -2, -2, -2,
	-2, -2, -2, 0, 229, 0, 0, 0, 0, -2,
	-2, -2, 215, 0, 217, 219, 111, 109, 0, 20,
	0, 22, 0, 24, 0, 0, 118, 0, 67, 0,
	68, 70, 0, 117, 44, 45, 0, 47, 0, 0,
	0, 0, 258, 265, 0, 264, 0, 0, 283, 0,
	0, 170, 0, 171, 0, 0, 249, 0, 0, 255,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	87, 85, 0, 80, 81, 0, 84, 0, 73, 75,
	43, 0, 0, 0, 0, 32, 0, 33, 34, 0,
	42, 0, 188, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, -2, -2, -2, 216, 218, 220,
	18, 112, 0, 110, 21, 23, 25, 100, 101, 104,
	0, 119, 0, 0, 84, 84, 84, 0, 0, 0,
	71, 43, 64, 46, 0, 267, 0, 269, 261, 0,
	266, 0, 0, 0, 0, 0, 0, 253, 254, 91,
	273, 277, 280, 278, 279, 274, 275, 146, 146, 0,
	88, 0, 86, 0, 0, 87, 0, 0, 0, 55,
	56, 74, 76, 67, 66, 181, 65, 65, 39, 35,
	41, 187, 189, 0, 207, 230, 0, 0, 235, 236,
	237, 238, 239, 240, 0, 113, 0, 103, 105, 106,
	107, 122, 122, 0, 120, 122, 122, 108, 84, 108,
	108, 133, 134, 0, 148, 149, 137, 0, 116, 0,
	0, 268, 0, 262, 168, 0, 177, 172, 179, 0,
	0, 0, 28, 0, 82, 83, 29, 52, 65, 0,
	0, 53, 43, 57, 0, 0, 43, 43, 190, 0,
	232, 0, 208, 102, 114, 123, 0, 115, 121, 127,
	128, 122, 108, 122, 122, 0, 0, 0, 151, 138,
	0, 69, 0, 0, 263, 0, 178, 0, 281, 147,
	282, 92, 43, 0, 0, 54, 182, 0, 0, 67,
	67, 231, 233, 0, 124, 125, 0, 129, 122, 131,
	132, 135, 137, 150, 146, 140, 0, 154, 0, 0,
	0, 95, 93, 0, 0, 65, 65, 0, 184, 58,
	59, 234, 126, 130, 136, 0, 0, 0, 0, 108,
	0, 0, 173, 180, 89, 96, 0, 94, 60, 70,
	43, 43, 183, 139, 141, 142, 145, 143, 122, 0,
	0, 0, 84, 0, 97, 0, 0, 0, 152, 0,
	0, 154, 175, 0, 0, 61, 62, 0, 84, 0,
	108, 169, 0, 174, 77, 158, 84, 84, 161, 166,
	0, 122, 176, 155, 0, 163, 84, 165, 156, 0,
	157, 84, 153, 0, 0, 164, 0, 167, 0, 0,
	0, 84, 0, 0, 161, 0, 0, 159, 160, 162,
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
	172, 173, 174, 175, 176, 177, 178,
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
			yyVAL.path = expression.NewField(yyS[yypt-4].path, yyS[yypt-1].expr)
		}
	case 184:
		//line n1ql.y:1437
		{
			yyVAL.path = expression.NewElement(yyS[yypt-3].path, yyS[yypt-1].expr)
		}
	case 185:
		yyVAL.expr = yyS[yypt-0].expr
	case 186:
		//line n1ql.y:1454
		{
			yyVAL.expr = expression.NewField(yyS[yypt-2].expr, expression.NewFieldName(yyS[yypt-0].s))
		}
	case 187:
		//line n1ql.y:1459
		{
			yyVAL.expr = expression.NewField(yyS[yypt-4].expr, yyS[yypt-1].expr)
		}
	case 188:
		//line n1ql.y:1464
		{
			yyVAL.expr = expression.NewElement(yyS[yypt-3].expr, yyS[yypt-1].expr)
		}
	case 189:
		//line n1ql.y:1469
		{
			yyVAL.expr = expression.NewSlice(yyS[yypt-4].expr, yyS[yypt-2].expr)
		}
	case 190:
		//line n1ql.y:1474
		{
			yyVAL.expr = expression.NewSlice(yyS[yypt-5].expr, yyS[yypt-3].expr, yyS[yypt-1].expr)
		}
	case 191:
		//line n1ql.y:1480
		{
			yyVAL.expr = expression.NewAdd(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 192:
		//line n1ql.y:1485
		{
			yyVAL.expr = expression.NewSub(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 193:
		//line n1ql.y:1490
		{
			yyVAL.expr = expression.NewMult(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 194:
		//line n1ql.y:1495
		{
			yyVAL.expr = expression.NewDiv(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 195:
		//line n1ql.y:1500
		{
			yyVAL.expr = expression.NewMod(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 196:
		//line n1ql.y:1506
		{
			yyVAL.expr = expression.NewConcat(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 197:
		//line n1ql.y:1512
		{
			yyVAL.expr = expression.NewAnd(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 198:
		//line n1ql.y:1517
		{
			yyVAL.expr = expression.NewOr(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 199:
		//line n1ql.y:1522
		{
			yyVAL.expr = expression.NewNot(yyS[yypt-0].expr)
		}
	case 200:
		//line n1ql.y:1528
		{
			yyVAL.expr = expression.NewEq(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 201:
		//line n1ql.y:1533
		{
			yyVAL.expr = expression.NewEq(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 202:
		//line n1ql.y:1538
		{
			yyVAL.expr = expression.NewNE(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 203:
		//line n1ql.y:1543
		{
			yyVAL.expr = expression.NewLT(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 204:
		//line n1ql.y:1548
		{
			yyVAL.expr = expression.NewGT(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 205:
		//line n1ql.y:1553
		{
			yyVAL.expr = expression.NewLE(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 206:
		//line n1ql.y:1558
		{
			yyVAL.expr = expression.NewGE(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 207:
		//line n1ql.y:1563
		{
			yyVAL.expr = expression.NewBetween(yyS[yypt-4].expr, yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 208:
		//line n1ql.y:1568
		{
			yyVAL.expr = expression.NewNotBetween(yyS[yypt-5].expr, yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 209:
		//line n1ql.y:1573
		{
			yyVAL.expr = expression.NewLike(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 210:
		//line n1ql.y:1578
		{
			yyVAL.expr = expression.NewNotLike(yyS[yypt-3].expr, yyS[yypt-0].expr)
		}
	case 211:
		//line n1ql.y:1583
		{
			yyVAL.expr = expression.NewIn(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 212:
		//line n1ql.y:1588
		{
			yyVAL.expr = expression.NewNotIn(yyS[yypt-3].expr, yyS[yypt-0].expr)
		}
	case 213:
		//line n1ql.y:1593
		{
			yyVAL.expr = expression.NewWithin(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 214:
		//line n1ql.y:1598
		{
			yyVAL.expr = expression.NewNotWithin(yyS[yypt-3].expr, yyS[yypt-0].expr)
		}
	case 215:
		//line n1ql.y:1603
		{
			yyVAL.expr = expression.NewIsNull(yyS[yypt-2].expr)
		}
	case 216:
		//line n1ql.y:1608
		{
			yyVAL.expr = expression.NewIsNotNull(yyS[yypt-3].expr)
		}
	case 217:
		//line n1ql.y:1613
		{
			yyVAL.expr = expression.NewIsMissing(yyS[yypt-2].expr)
		}
	case 218:
		//line n1ql.y:1618
		{
			yyVAL.expr = expression.NewIsNotMissing(yyS[yypt-3].expr)
		}
	case 219:
		//line n1ql.y:1623
		{
			yyVAL.expr = expression.NewIsValued(yyS[yypt-2].expr)
		}
	case 220:
		//line n1ql.y:1628
		{
			yyVAL.expr = expression.NewIsNotValued(yyS[yypt-3].expr)
		}
	case 221:
		//line n1ql.y:1633
		{
			yyVAL.expr = expression.NewExists(yyS[yypt-0].expr)
		}
	case 222:
		yyVAL.expr = yyS[yypt-0].expr
	case 223:
		//line n1ql.y:1644
		{
			yyVAL.expr = expression.NewIdentifier(yyS[yypt-0].s)
		}
	case 224:
		yyVAL.expr = yyS[yypt-0].expr
	case 225:
		//line n1ql.y:1653
		{
			yyVAL.expr = expression.NewNeg(yyS[yypt-0].expr)
		}
	case 226:
		yyVAL.expr = yyS[yypt-0].expr
	case 227:
		yyVAL.expr = yyS[yypt-0].expr
	case 228:
		yyVAL.expr = yyS[yypt-0].expr
	case 229:
		yyVAL.expr = yyS[yypt-0].expr
	case 230:
		//line n1ql.y:1672
		{
			yyVAL.expr = expression.NewField(yyS[yypt-2].expr, expression.NewFieldName(yyS[yypt-0].s))
		}
	case 231:
		//line n1ql.y:1677
		{
			yyVAL.expr = expression.NewField(yyS[yypt-4].expr, yyS[yypt-1].expr)
		}
	case 232:
		//line n1ql.y:1682
		{
			yyVAL.expr = expression.NewElement(yyS[yypt-3].expr, yyS[yypt-1].expr)
		}
	case 233:
		//line n1ql.y:1687
		{
			yyVAL.expr = expression.NewSlice(yyS[yypt-4].expr, yyS[yypt-2].expr)
		}
	case 234:
		//line n1ql.y:1692
		{
			yyVAL.expr = expression.NewSlice(yyS[yypt-5].expr, yyS[yypt-3].expr, yyS[yypt-1].expr)
		}
	case 235:
		//line n1ql.y:1698
		{
			yyVAL.expr = expression.NewAdd(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 236:
		//line n1ql.y:1703
		{
			yyVAL.expr = expression.NewSub(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 237:
		//line n1ql.y:1708
		{
			yyVAL.expr = expression.NewMult(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 238:
		//line n1ql.y:1713
		{
			yyVAL.expr = expression.NewDiv(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 239:
		//line n1ql.y:1718
		{
			yyVAL.expr = expression.NewMod(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 240:
		//line n1ql.y:1724
		{
			yyVAL.expr = expression.NewConcat(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 241:
		//line n1ql.y:1738
		{
			yyVAL.expr = expression.NULL_EXPR
		}
	case 242:
		//line n1ql.y:1743
		{
			yyVAL.expr = expression.FALSE_EXPR
		}
	case 243:
		//line n1ql.y:1748
		{
			yyVAL.expr = expression.TRUE_EXPR
		}
	case 244:
		//line n1ql.y:1753
		{
			yyVAL.expr = expression.NewConstant(value.NewValue(yyS[yypt-0].f))
		}
	case 245:
		//line n1ql.y:1758
		{
			yyVAL.expr = expression.NewConstant(value.NewValue(yyS[yypt-0].n))
		}
	case 246:
		//line n1ql.y:1763
		{
			yyVAL.expr = expression.NewConstant(value.NewValue(yyS[yypt-0].s))
		}
	case 247:
		yyVAL.expr = yyS[yypt-0].expr
	case 248:
		yyVAL.expr = yyS[yypt-0].expr
	case 249:
		//line n1ql.y:1774
		{
			yyVAL.expr = expression.NewObjectConstruct(yyS[yypt-1].bindings)
		}
	case 250:
		//line n1ql.y:1781
		{
			yyVAL.bindings = nil
		}
	case 251:
		yyVAL.bindings = yyS[yypt-0].bindings
	case 252:
		//line n1ql.y:1790
		{
			yyVAL.bindings = expression.Bindings{yyS[yypt-0].binding}
		}
	case 253:
		//line n1ql.y:1795
		{
			yyVAL.bindings = append(yyS[yypt-2].bindings, yyS[yypt-0].binding)
		}
	case 254:
		//line n1ql.y:1802
		{
			yyVAL.binding = expression.NewBinding(yyS[yypt-2].s, yyS[yypt-0].expr)
		}
	case 255:
		//line n1ql.y:1809
		{
			yyVAL.expr = expression.NewArrayConstruct(yyS[yypt-1].exprs...)
		}
	case 256:
		//line n1ql.y:1816
		{
			yyVAL.exprs = nil
		}
	case 257:
		yyVAL.exprs = yyS[yypt-0].exprs
	case 258:
		//line n1ql.y:1832
		{
			yyVAL.expr = yyS[yypt-1].expr
		}
	case 259:
		yyVAL.expr = yyS[yypt-0].expr
	case 260:
		yyVAL.expr = yyS[yypt-0].expr
	case 261:
		//line n1ql.y:1845
		{
			yyVAL.expr = expression.NewSimpleCase(yyS[yypt-2].expr, yyS[yypt-1].whenTerms, yyS[yypt-0].expr)
		}
	case 262:
		//line n1ql.y:1852
		{
			yyVAL.whenTerms = expression.WhenTerms{&expression.WhenTerm{yyS[yypt-2].expr, yyS[yypt-0].expr}}
		}
	case 263:
		//line n1ql.y:1857
		{
			yyVAL.whenTerms = append(yyS[yypt-4].whenTerms, &expression.WhenTerm{yyS[yypt-2].expr, yyS[yypt-0].expr})
		}
	case 264:
		//line n1ql.y:1865
		{
			yyVAL.expr = expression.NewSearchedCase(yyS[yypt-1].whenTerms, yyS[yypt-0].expr)
		}
	case 265:
		//line n1ql.y:1872
		{
			yyVAL.expr = nil
		}
	case 266:
		//line n1ql.y:1877
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 267:
		//line n1ql.y:1891
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
	case 268:
		//line n1ql.y:1910
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
	case 269:
		//line n1ql.y:1925
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
	case 270:
		yyVAL.s = yyS[yypt-0].s
	case 271:
		yyVAL.expr = yyS[yypt-0].expr
	case 272:
		yyVAL.expr = yyS[yypt-0].expr
	case 273:
		//line n1ql.y:1959
		{
			yyVAL.expr = expression.NewAny(yyS[yypt-2].bindings, yyS[yypt-1].expr)
		}
	case 274:
		//line n1ql.y:1964
		{
			yyVAL.expr = expression.NewAny(yyS[yypt-2].bindings, yyS[yypt-1].expr)
		}
	case 275:
		//line n1ql.y:1969
		{
			yyVAL.expr = expression.NewEvery(yyS[yypt-2].bindings, yyS[yypt-1].expr)
		}
	case 276:
		//line n1ql.y:1976
		{
			yyVAL.bindings = expression.Bindings{yyS[yypt-0].binding}
		}
	case 277:
		//line n1ql.y:1981
		{
			yyVAL.bindings = append(yyS[yypt-2].bindings, yyS[yypt-0].binding)
		}
	case 278:
		//line n1ql.y:1988
		{
			yyVAL.binding = expression.NewBinding(yyS[yypt-2].s, yyS[yypt-0].expr)
		}
	case 279:
		//line n1ql.y:1993
		{
			yyVAL.binding = expression.NewDescendantBinding(yyS[yypt-2].s, yyS[yypt-0].expr)
		}
	case 280:
		//line n1ql.y:2000
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 281:
		//line n1ql.y:2007
		{
			yyVAL.expr = expression.NewArray(yyS[yypt-4].expr, yyS[yypt-2].bindings, yyS[yypt-1].expr)
		}
	case 282:
		//line n1ql.y:2012
		{
			yyVAL.expr = expression.NewFirst(yyS[yypt-4].expr, yyS[yypt-2].bindings, yyS[yypt-1].expr)
		}
	case 283:
		//line n1ql.y:2026
		{
			yyVAL.expr = yyS[yypt-1].expr
		}
	case 284:
		yyVAL.expr = yyS[yypt-0].expr
	case 285:
		//line n1ql.y:2035
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
