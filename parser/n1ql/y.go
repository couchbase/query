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
const OFFSET = 57430
const ON = 57431
const OPTION = 57432
const OR = 57433
const ORDER = 57434
const OUTER = 57435
const OVER = 57436
const PARTITION = 57437
const PASSWORD = 57438
const PATH = 57439
const POOL = 57440
const PREPARE = 57441
const PRIMARY = 57442
const PRIVATE = 57443
const PRIVILEGE = 57444
const PROCEDURE = 57445
const PUBLIC = 57446
const RAW = 57447
const REALM = 57448
const REDUCE = 57449
const RENAME = 57450
const RETURN = 57451
const RETURNING = 57452
const REVOKE = 57453
const RIGHT = 57454
const ROLE = 57455
const ROLLBACK = 57456
const SATISFIES = 57457
const SCHEMA = 57458
const SELECT = 57459
const SET = 57460
const SHOW = 57461
const SOME = 57462
const START = 57463
const STATISTICS = 57464
const SYSTEM = 57465
const THEN = 57466
const TO = 57467
const TRANSACTION = 57468
const TRIGGER = 57469
const TRUE = 57470
const TRUNCATE = 57471
const TYPE = 57472
const UNDER = 57473
const UNION = 57474
const UNIQUE = 57475
const UNNEST = 57476
const UNSET = 57477
const UPDATE = 57478
const UPSERT = 57479
const USE = 57480
const USER = 57481
const USING = 57482
const VALUE = 57483
const VALUED = 57484
const VALUES = 57485
const VIEW = 57486
const WHEN = 57487
const WHERE = 57488
const WHILE = 57489
const WITH = 57490
const WITHIN = 57491
const WORK = 57492
const XOR = 57493
const INT = 57494
const NUMBER = 57495
const IDENTIFIER = 57496
const STRING = 57497
const LPAREN = 57498
const RPAREN = 57499
const LBRACE = 57500
const RBRACE = 57501
const LBRACKET = 57502
const RBRACKET = 57503
const COMMA = 57504
const COLON = 57505
const EQ = 57506
const DEQ = 57507
const NE = 57508
const LT = 57509
const GT = 57510
const LE = 57511
const GE = 57512
const CONCAT = 57513
const PLUS = 57514
const STAR = 57515
const DIV = 57516
const MOD = 57517
const UMINUS = 57518
const DOT = 57519

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
	156, 270,
	-2, 223,
	-1, 104,
	163, 63,
	-2, 64,
	-1, 140,
	50, 72,
	67, 72,
	85, 72,
	134, 72,
	-2, 50,
	-1, 166,
	164, 0,
	165, 0,
	166, 0,
	-2, 200,
	-1, 167,
	164, 0,
	165, 0,
	166, 0,
	-2, 201,
	-1, 168,
	164, 0,
	165, 0,
	166, 0,
	-2, 202,
	-1, 169,
	167, 0,
	168, 0,
	169, 0,
	170, 0,
	-2, 203,
	-1, 170,
	167, 0,
	168, 0,
	169, 0,
	170, 0,
	-2, 204,
	-1, 171,
	167, 0,
	168, 0,
	169, 0,
	170, 0,
	-2, 205,
	-1, 172,
	167, 0,
	168, 0,
	169, 0,
	170, 0,
	-2, 206,
	-1, 179,
	75, 0,
	-2, 209,
	-1, 180,
	58, 0,
	149, 0,
	-2, 211,
	-1, 181,
	58, 0,
	149, 0,
	-2, 213,
	-1, 274,
	75, 0,
	-2, 210,
	-1, 275,
	58, 0,
	149, 0,
	-2, 212,
	-1, 276,
	58, 0,
	149, 0,
	-2, 214,
}

const yyNprod = 286
const yyPrivate = 57344

var yyTokenNames []string
var yyStates []string

const yyLast = 2560

var yyAct = []int{

	154, 3, 560, 548, 424, 558, 549, 295, 296, 186,
	88, 89, 499, 508, 291, 203, 383, 515, 299, 243,
	129, 475, 204, 91, 399, 250, 205, 220, 385, 438,
	288, 199, 146, 382, 329, 149, 125, 244, 407, 62,
	12, 102, 141, 371, 150, 110, 127, 128, 114, 222,
	251, 122, 316, 48, 126, 215, 8, 314, 133, 134,
	334, 502, 440, 483, 454, 453, 230, 157, 158, 159,
	160, 161, 162, 163, 164, 165, 166, 167, 168, 169,
	170, 171, 172, 266, 415, 179, 180, 181, 115, 415,
	315, 253, 87, 436, 266, 66, 269, 270, 271, 252,
	265, 414, 131, 132, 66, 143, 414, 126, 68, 400,
	400, 265, 65, 217, 155, 268, 156, 69, 70, 71,
	233, 65, 228, 229, 202, 333, 496, 232, 254, 437,
	435, 366, 230, 350, 227, 226, 478, 307, 456, 174,
	457, 144, 240, 364, 355, 305, 356, 189, 191, 193,
	258, 230, 218, 155, 445, 156, 106, 261, 103, 268,
	224, 224, 206, 410, 245, 123, 345, 232, 130, 225,
	260, 415, 104, 207, 390, 221, 302, 274, 275, 276,
	255, 257, 144, 256, 104, 242, 66, 216, 414, 283,
	497, 234, 559, 266, 104, 554, 289, 72, 67, 69,
	70, 71, 500, 65, 272, 267, 269, 270, 271, 298,
	265, 306, 293, 104, 175, 309, 173, 310, 268, 552,
	242, 542, 142, 63, 304, 135, 201, 538, 297, 318,
	294, 319, 174, 303, 322, 323, 324, 266, 184, 340,
	480, 183, 182, 332, 284, 298, 285, 573, 286, 267,
	269, 270, 271, 335, 265, 572, 336, 349, 568, 177,
	64, 278, 231, 539, 353, 277, 343, 357, 344, 308,
	95, 63, 529, 111, 337, 426, 176, 223, 223, 317,
	321, 498, 235, 365, 447, 327, 328, 300, 64, 124,
	194, 94, 214, 374, 523, 192, 266, 185, 342, 348,
	190, 377, 379, 380, 378, 117, 207, 272, 267, 269,
	270, 271, 393, 265, 373, 386, 282, 388, 101, 97,
	279, 174, 509, 339, 174, 174, 174, 174, 174, 174,
	521, 372, 537, 442, 376, 405, 64, 375, 63, 412,
	313, 312, 396, 63, 398, 116, 143, 389, 63, 563,
	178, 519, 566, 401, 224, 224, 564, 419, 520, 93,
	245, 301, 394, 395, 246, 137, 570, 289, 569, 402,
	406, 404, 416, 417, 428, 413, 411, 427, 409, 409,
	429, 430, 530, 188, 213, 432, 534, 431, 441, 433,
	434, 387, 273, 444, 236, 237, 209, 423, 292, 248,
	105, 449, 576, 64, 126, 99, 98, 575, 64, 249,
	550, 346, 347, 64, 61, 219, 458, 196, 197, 198,
	119, 174, 463, 118, 208, 506, 331, 63, 455, 100,
	264, 443, 459, 460, 452, 513, 467, 472, 469, 470,
	451, 450, 468, 448, 326, 325, 126, 320, 212, 571,
	533, 403, 195, 2, 386, 341, 338, 477, 487, 465,
	1, 476, 466, 142, 446, 90, 473, 492, 484, 471,
	541, 223, 223, 493, 397, 522, 545, 139, 553, 68,
	439, 354, 479, 384, 358, 359, 360, 361, 362, 363,
	381, 489, 490, 474, 425, 408, 408, 464, 290, 495,
	34, 501, 494, 507, 33, 268, 32, 524, 503, 518,
	245, 510, 511, 18, 516, 516, 517, 476, 514, 17,
	16, 15, 14, 528, 13, 7, 526, 527, 525, 532,
	6, 73, 5, 4, 543, 544, 531, 82, 367, 368,
	535, 536, 280, 281, 540, 546, 547, 187, 287, 92,
	551, 561, 96, 555, 557, 556, 562, 66, 73, 145,
	505, 206, 504, 565, 82, 482, 481, 330, 567, 67,
	69, 70, 71, 241, 65, 574, 561, 561, 578, 579,
	577, 422, 85, 266, 136, 73, 200, 247, 138, 369,
	87, 82, 140, 59, 272, 267, 269, 270, 271, 84,
	265, 60, 26, 113, 25, 43, 68, 21, 46, 85,
	83, 370, 485, 486, 45, 74, 24, 87, 109, 108,
	107, 23, 120, 121, 42, 41, 84, 19, 0, 0,
	0, 0, 0, 68, 0, 0, 85, 83, 0, 0,
	0, 0, 74, 0, 87, 0, 0, 0, 0, 0,
	0, 0, 0, 84, 0, 0, 0, 0, 0, 0,
	68, 0, 0, 0, 83, 0, 0, 0, 0, 74,
	0, 0, 0, 86, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 66, 420, 0, 421, 75, 76,
	77, 78, 79, 80, 81, 72, 67, 69, 70, 71,
	86, 65, 0, 73, 0, 207, 0, 0, 0, 82,
	0, 66, 0, 0, 0, 75, 76, 77, 78, 79,
	80, 81, 72, 67, 69, 70, 71, 86, 65, 0,
	73, 0, 0, 206, 0, 0, 82, 0, 66, 0,
	0, 0, 75, 76, 77, 78, 79, 80, 81, 72,
	67, 69, 70, 71, 85, 65, 0, 0, 0, 0,
	73, 0, 87, 0, 0, 0, 82, 0, 0, 0,
	0, 84, 0, 0, 0, 0, 0, 0, 68, 0,
	0, 85, 83, 0, 0, 0, 0, 74, 0, 87,
	0, 0, 0, 0, 0, 0, 0, 0, 84, 0,
	0, 0, 0, 0, 0, 68, 0, 0, 0, 83,
	0, 85, 0, 0, 74, 0, 0, 0, 0, 87,
	0, 0, 0, 0, 0, 0, 0, 0, 84, 0,
	0, 0, 0, 0, 0, 68, 0, 0, 0, 83,
	0, 0, 0, 0, 74, 86, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 66, 262, 0, 263,
	75, 76, 77, 78, 79, 80, 81, 72, 67, 69,
	70, 71, 86, 65, 0, 0, 0, 207, 73, 0,
	0, 0, 0, 66, 82, 0, 0, 75, 76, 77,
	78, 79, 80, 81, 72, 67, 69, 70, 71, 242,
	259, 0, 86, 0, 0, 73, 0, 0, 0, 0,
	0, 82, 0, 66, 0, 0, 0, 75, 76, 77,
	78, 79, 80, 81, 72, 67, 69, 70, 71, 85,
	65, 0, 73, 0, 0, 0, 0, 87, 82, 0,
	0, 0, 0, 0, 0, 0, 84, 0, 0, 440,
	0, 0, 0, 68, 0, 0, 85, 83, 0, 0,
	0, 0, 74, 0, 87, 0, 0, 0, 0, 0,
	0, 0, 0, 84, 0, 0, 51, 0, 0, 0,
	68, 0, 0, 85, 83, 0, 0, 0, 0, 74,
	0, 87, 0, 0, 0, 0, 0, 49, 0, 0,
	84, 0, 29, 0, 0, 0, 0, 68, 50, 0,
	0, 83, 0, 0, 0, 0, 74, 0, 11, 0,
	86, 0, 0, 63, 0, 0, 0, 0, 512, 0,
	0, 66, 0, 0, 27, 75, 76, 77, 78, 79,
	80, 81, 72, 67, 69, 70, 71, 86, 65, 0,
	73, 0, 31, 0, 0, 0, 82, 0, 66, 0,
	0, 0, 75, 76, 77, 78, 79, 80, 81, 72,
	67, 69, 70, 71, 86, 65, 0, 73, 0, 0,
	0, 0, 0, 82, 0, 66, 491, 0, 64, 75,
	76, 77, 78, 79, 80, 81, 72, 67, 69, 70,
	71, 85, 65, 0, 73, 0, 0, 30, 28, 87,
	82, 0, 0, 0, 0, 0, 0, 0, 84, 0,
	0, 0, 0, 0, 0, 68, 0, 0, 85, 83,
	0, 0, 0, 0, 74, 0, 87, 0, 0, 0,
	0, 0, 0, 0, 0, 84, 0, 0, 0, 0,
	0, 0, 68, 0, 0, 85, 83, 0, 0, 0,
	0, 74, 0, 87, 0, 0, 0, 0, 0, 0,
	0, 0, 84, 0, 0, 0, 0, 0, 0, 68,
	0, 0, 0, 83, 0, 0, 0, 0, 74, 0,
	0, 0, 86, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 66, 488, 0, 0, 75, 76, 77,
	78, 79, 80, 81, 72, 67, 69, 70, 71, 86,
	65, 0, 73, 0, 0, 0, 0, 461, 82, 0,
	66, 0, 0, 0, 75, 76, 77, 78, 79, 80,
	81, 72, 67, 69, 70, 71, 86, 65, 0, 73,
	0, 0, 0, 0, 0, 82, 0, 66, 418, 0,
	0, 75, 76, 77, 78, 79, 80, 81, 72, 67,
	69, 70, 71, 85, 65, 0, 73, 0, 0, 0,
	0, 87, 82, 0, 0, 0, 0, 0, 0, 0,
	84, 0, 0, 0, 0, 0, 0, 68, 0, 0,
	85, 83, 0, 0, 0, 0, 74, 0, 87, 0,
	0, 0, 0, 0, 0, 0, 0, 84, 0, 0,
	0, 0, 0, 0, 68, 0, 0, 85, 83, 0,
	0, 0, 0, 74, 0, 87, 0, 0, 0, 392,
	0, 0, 0, 0, 84, 0, 0, 0, 0, 0,
	0, 68, 0, 0, 0, 83, 0, 0, 0, 0,
	74, 0, 0, 0, 86, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 66, 0, 0, 0, 75,
	76, 77, 78, 79, 80, 81, 72, 67, 69, 70,
	71, 86, 65, 0, 73, 0, 0, 0, 0, 391,
	82, 0, 66, 0, 0, 0, 75, 76, 77, 78,
	79, 80, 81, 72, 67, 69, 70, 71, 86, 65,
	0, 73, 0, 0, 0, 0, 351, 82, 0, 66,
	0, 0, 0, 75, 76, 77, 78, 79, 80, 81,
	72, 67, 69, 70, 71, 85, 65, 0, 73, 0,
	0, 0, 0, 87, 82, 0, 0, 0, 0, 0,
	0, 0, 84, 0, 0, 239, 0, 0, 0, 68,
	0, 0, 85, 83, 0, 0, 0, 0, 74, 0,
	87, 0, 0, 0, 0, 0, 0, 0, 0, 84,
	0, 0, 238, 0, 0, 0, 68, 0, 0, 85,
	83, 0, 0, 0, 0, 74, 0, 87, 0, 0,
	0, 311, 0, 0, 0, 0, 84, 0, 0, 0,
	0, 0, 0, 68, 0, 0, 0, 83, 0, 0,
	0, 0, 74, 0, 0, 0, 86, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 66, 0, 0,
	0, 75, 76, 77, 78, 79, 80, 81, 72, 67,
	69, 70, 71, 86, 65, 0, 73, 0, 0, 0,
	0, 0, 82, 0, 66, 0, 0, 0, 75, 76,
	77, 78, 79, 80, 81, 72, 67, 69, 70, 71,
	86, 65, 0, 73, 0, 0, 0, 0, 0, 82,
	0, 66, 0, 0, 0, 75, 76, 77, 78, 79,
	80, 81, 72, 67, 69, 70, 71, 85, 65, 0,
	73, 0, 0, 0, 0, 87, 82, 0, 0, 0,
	0, 0, 0, 0, 84, 0, 0, 0, 0, 0,
	0, 68, 0, 0, 85, 83, 0, 0, 0, 0,
	74, 0, 87, 0, 0, 0, 0, 0, 0, 0,
	0, 84, 0, 0, 0, 0, 0, 0, 68, 0,
	0, 85, 83, 0, 0, 0, 0, 74, 0, 87,
	0, 0, 0, 0, 0, 0, 0, 0, 84, 0,
	0, 0, 0, 0, 0, 68, 0, 0, 0, 83,
	0, 0, 0, 0, 112, 0, 0, 0, 86, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 66,
	0, 0, 0, 75, 76, 77, 78, 79, 80, 81,
	72, 67, 69, 70, 71, 86, 65, 0, 0, 0,
	0, 0, 0, 0, 82, 0, 66, 0, 0, 0,
	75, 76, 77, 78, 79, 80, 81, 72, 67, 69,
	70, 71, 86, 65, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 66, 0, 0, 0, 75, 76, 77,
	78, 79, 80, 81, 72, 67, 69, 70, 71, 85,
	65, 148, 0, 0, 0, 54, 57, 87, 0, 0,
	0, 0, 0, 0, 0, 44, 84, 0, 0, 0,
	0, 0, 0, 68, 82, 0, 0, 83, 0, 0,
	0, 0, 147, 0, 0, 0, 152, 0, 0, 56,
	0, 0, 0, 10, 0, 36, 58, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 54, 57, 0,
	0, 0, 0, 0, 0, 0, 0, 44, 0, 85,
	0, 0, 0, 0, 0, 0, 0, 87, 0, 22,
	0, 0, 0, 9, 35, 0, 84, 0, 152, 0,
	86, 56, 0, 68, 0, 10, 0, 36, 58, 0,
	0, 66, 151, 0, 0, 75, 76, 77, 78, 79,
	80, 81, 72, 67, 69, 70, 71, 55, 65, 0,
	0, 0, 0, 0, 0, 37, 0, 0, 0, 0,
	0, 22, 0, 0, 0, 9, 35, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 39,
	38, 20, 40, 47, 151, 52, 0, 53, 0, 0,
	86, 0, 0, 0, 0, 0, 0, 0, 0, 55,
	153, 66, 0, 0, 0, 0, 0, 37, 78, 79,
	80, 81, 72, 67, 69, 70, 71, 0, 65, 54,
	57, 0, 0, 0, 0, 0, 0, 0, 0, 44,
	0, 39, 38, 20, 40, 47, 0, 52, 0, 53,
	0, 54, 57, 82, 0, 0, 210, 0, 0, 0,
	0, 44, 153, 56, 0, 0, 0, 10, 0, 36,
	58, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 56, 0, 82, 0, 10,
	0, 36, 58, 0, 0, 0, 0, 0, 85, 0,
	0, 0, 0, 22, 0, 0, 87, 9, 35, 0,
	51, 0, 0, 54, 57, 84, 0, 0, 0, 0,
	0, 0, 68, 44, 0, 22, 0, 0, 0, 9,
	35, 49, 85, 0, 0, 0, 29, 0, 0, 0,
	87, 55, 50, 0, 0, 0, 0, 56, 0, 37,
	0, 10, 11, 36, 58, 0, 68, 63, 0, 0,
	0, 0, 0, 55, 0, 0, 0, 0, 27, 0,
	0, 37, 0, 39, 38, 20, 40, 47, 0, 52,
	0, 53, 0, 0, 0, 0, 31, 22, 0, 86,
	0, 9, 35, 0, 211, 39, 38, 20, 40, 47,
	66, 52, 0, 53, 0, 0, 0, 0, 0, 0,
	0, 72, 67, 69, 70, 71, 153, 65, 0, 0,
	0, 0, 64, 86, 0, 55, 0, 0, 54, 57,
	0, 0, 0, 37, 66, 0, 0, 0, 44, 0,
	0, 30, 28, 54, 57, 72, 67, 69, 70, 71,
	0, 65, 0, 44, 0, 0, 0, 39, 38, 20,
	40, 47, 56, 52, 0, 53, 10, 0, 36, 58,
	0, 0, 0, 0, 0, 0, 0, 56, 0, 0,
	0, 10, 0, 36, 58, 0, 0, 0, 0, 0,
	54, 57, 0, 0, 0, 0, 0, 0, 0, 0,
	44, 0, 22, 0, 0, 0, 9, 35, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 22, 0, 0,
	0, 9, 35, 0, 56, 0, 0, 0, 10, 0,
	36, 58, 0, 0, 63, 0, 0, 0, 0, 0,
	55, 0, 0, 0, 0, 0, 0, 0, 37, 0,
	0, 0, 0, 0, 0, 55, 0, 0, 0, 0,
	0, 0, 0, 37, 22, 0, 0, 0, 9, 35,
	0, 0, 39, 38, 20, 40, 47, 0, 52, 0,
	53, 462, 54, 57, 0, 0, 0, 39, 38, 20,
	40, 47, 44, 52, 0, 53, 352, 0, 0, 64,
	0, 0, 55, 0, 0, 54, 57, 0, 0, 0,
	37, 0, 0, 0, 0, 44, 56, 0, 0, 0,
	10, 0, 36, 58, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 39, 38, 20, 40, 47, 56,
	52, 0, 53, 10, 0, 36, 58, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 22, 54, 57, 0,
	9, 35, 0, 0, 0, 0, 0, 44, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 22,
	0, 0, 0, 9, 35, 0, 0, 0, 0, 0,
	0, 56, 0, 0, 55, 0, 0, 36, 58, 0,
	0, 0, 37, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 55, 0, 112,
	0, 0, 0, 0, 0, 37, 39, 38, 20, 40,
	47, 22, 52, 0, 53, 0, 35, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 39,
	38, 20, 40, 47, 0, 52, 0, 53, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 55,
	0, 0, 0, 0, 0, 0, 0, 37, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 39, 38, 20, 40, 47, 0, 52, 0, 53,
}
var yyPact = []int{

	2055, -1000, -1000, 1586, -1000, -1000, -1000, -1000, -1000, 2347,
	2347, 971, -1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000,
	-1000, -1000, 2347, -1000, -1000, -1000, 227, 341, 340, 377,
	30, 335, -1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000,
	-1000, -1000, -1000, 0, 2324, -1000, -1000, 2232, -1000, 245,
	363, 360, 10, 2347, 14, 14, 14, 2347, 2347, -1000,
	-1000, 292, 375, 59, 1787, -1, 2347, 2347, 2347, 2347,
	2347, 2347, 2347, 2347, 2347, 2347, 2347, 2347, 2347, 2347,
	2347, 2347, 2399, 201, 2347, 2347, 2347, 155, 1731, 26,
	-1000, -65, 307, 296, 291, 286, -1000, 436, 30, 30,
	30, 88, -39, 152, -1000, 30, 1971, 407, -1000, -1000,
	1559, 147, 2347, -5, 1586, -1000, 355, 21, 30, 30,
	-24, -28, -1000, -41, -38, -30, 1586, 5, -1000, 133,
	-1000, 5, 5, 1441, 1414, 39, -1000, 19, 292, -1000,
	337, -1000, -127, -64, -72, -1000, -34, 1839, 1993, 2347,
	-1000, -1000, -1000, -1000, 723, -1000, 2347, 696, -56, -56,
	-65, -65, -65, 397, 1731, 1613, 1801, 1801, 1801, 1990,
	1990, 1990, 1990, 423, -1000, 2399, 2347, 2347, 2347, 2024,
	26, 26, -1000, 178, -1000, -1000, 228, -1000, 2347, -1000,
	219, -1000, 219, -1000, 219, 2347, 330, 330, 88, 110,
	-1000, 187, 22, -1000, -1000, -1000, 19, -1000, 84, -12,
	2347, -20, -1000, 147, 2347, -1000, 2347, 1387, -1000, 252,
	251, -1000, -120, -1000, -73, -125, -1000, 10, 2347, -1000,
	2347, 406, 14, 2347, 2347, 2347, 404, 403, 14, 14,
	371, -1000, 2347, -37, -1000, -104, 39, 189, -1000, 205,
	152, 12, 22, 22, 1993, -34, 2347, -34, 1586, -40,
	-1000, 1269, -1000, 2185, 2399, -10, 2347, 2399, 2399, 2399,
	2399, 2399, 2399, 136, 2024, 26, 26, -1000, -1000, -1000,
	-1000, -1000, 2347, 1586, -1000, -1000, -1000, -31, -1000, 578,
	171, -1000, 2347, 171, 39, 74, 39, 12, 12, 322,
	-1000, 152, -1000, -1000, 18, -1000, 1242, -1000, -1000, 1215,
	1586, 2347, 30, 30, 21, 22, 21, -1000, 1586, 1586,
	-1000, -1000, 1586, 1586, 1586, -1000, -1000, -35, -35, 143,
	-1000, 435, 1586, 19, 2347, 371, 40, 40, 2347, -1000,
	-1000, -1000, -1000, 88, -76, -1000, -127, -127, -1000, 1586,
	-1000, -1000, -1000, 1097, 33, -1000, 2347, 524, -77, -77,
	-66, -66, -66, 77, 2399, 1586, 2347, -1000, -1000, -1000,
	-1000, 165, 165, 2347, 1586, 165, 165, 307, 39, 307,
	307, -32, -1000, -71, -33, -1000, 11, 2347, -1000, 244,
	219, -1000, 2347, 1586, -1000, -2, -1000, -1000, 176, 402,
	2347, 400, -1000, 2347, -1000, 1586, -1000, -1000, -127, -98,
	-99, -1000, 551, -1000, -16, 2347, 152, 152, -1000, 1070,
	-1000, 2170, 33, -1000, -1000, -1000, 1839, -1000, 1586, -1000,
	-1000, 165, 307, 165, 165, 12, 2347, 12, -1000, -1000,
	14, 1586, 330, -21, 1586, 2347, -1000, 115, -1000, 1586,
	-1000, -11, 152, 22, 22, -1000, -1000, 2347, 1043, 88,
	88, -1000, -1000, 925, -1000, -34, 2347, -1000, 165, -1000,
	-1000, -1000, 898, -1000, -36, -1000, 132, 57, 152, -96,
	21, 369, -1000, 19, 233, -127, -127, 871, -1000, -1000,
	-1000, -1000, 1586, -1000, -1000, 394, 14, 12, 12, 307,
	272, 241, 199, -1000, -1000, -1000, 2347, -37, -1000, 187,
	152, 152, -1000, -1000, -1000, -1000, -76, -1000, 165, 148,
	303, 330, 39, 434, 1586, 317, 233, 233, -1000, 196,
	139, 57, 81, 2347, 2347, -1000, -1000, 110, 39, 347,
	307, -1000, 75, 1586, 1586, 50, 74, 39, 47, -1000,
	2347, 165, -1000, -1000, 270, -1000, 39, -1000, -1000, 266,
	-1000, 753, -1000, 134, 289, -1000, 287, -1000, 418, 131,
	123, 39, 344, 339, 47, 2347, 2347, -1000, -1000, -1000,
}
var yyPgo = []int{

	0, 627, 625, 624, 51, 623, 622, 0, 56, 216,
	36, 289, 37, 19, 26, 22, 15, 20, 621, 620,
	619, 618, 55, 273, 616, 614, 608, 47, 46, 262,
	24, 607, 605, 604, 603, 40, 602, 53, 601, 593,
	592, 414, 588, 42, 38, 587, 16, 25, 41, 158,
	586, 31, 13, 225, 584, 6, 573, 34, 567, 566,
	565, 562, 560, 44, 32, 559, 39, 552, 549, 30,
	548, 547, 9, 543, 542, 539, 538, 453, 533, 532,
	530, 525, 524, 522, 521, 520, 519, 513, 506, 504,
	500, 318, 43, 14, 498, 497, 494, 4, 21, 493,
	17, 7, 33, 490, 8, 28, 483, 480, 29, 12,
	478, 476, 3, 2, 5, 27, 49, 475, 470, 464,
	460, 35, 456, 18, 455,
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
	154, -31, 82, -18, -24, -33, -36, 63, 137, 31,
	136, 81, -88, -89, -90, 87, 48, 128, 153, 152,
	155, -2, -3, -32, 18, -25, -26, 156, -37, 26,
	37, 5, 158, 160, 8, 120, 42, 9, 49, -39,
	-38, -41, -66, 52, 117, 177, 160, 172, 82, 173,
	174, 175, 171, 7, 91, 164, 165, 166, 167, 168,
	169, 170, 13, 86, 75, 58, 149, 66, -7, -7,
	-77, -7, -68, 132, 64, 43, -67, 92, 65, 65,
	52, -91, -48, -49, 154, 65, 156, -19, -20, -21,
	-7, -23, 145, -34, -7, -35, 100, 60, 60, 60,
	-6, -5, -4, 155, -11, -10, -7, -28, -27, -17,
	154, -28, -28, -7, -7, -53, -54, 73, -42, -41,
	-40, -43, -49, -48, 123, -65, -64, 35, 4, -121,
	-63, 105, 39, 173, -7, 154, 156, -7, -7, -7,
	-7, -7, -7, -7, -7, -7, -7, -7, -7, -7,
	-7, -7, -7, -9, -8, 13, 75, 58, 149, -7,
	-7, -7, 87, 86, 83, 142, -72, -71, 76, -37,
	4, -37, 4, -37, 4, 16, -91, -91, -91, -51,
	-50, 138, 163, -16, -15, -14, 10, 154, -91, -11,
	35, 173, 41, -23, 145, -22, 40, -7, 157, 60,
	-115, 154, -116, -49, -48, -116, 159, 162, 163, 161,
	162, -29, 162, 115, 58, 149, -29, -29, 51, 51,
	-55, -56, 146, -13, -12, -14, -53, -45, 62, 72,
	-47, 177, 163, 163, 162, -64, -121, -64, -7, 177,
	-16, -7, 161, 163, 7, 177, 160, 172, 82, 173,
	174, 175, 171, -9, -7, -7, -7, 87, 83, 142,
	-74, -73, 88, -7, -37, -37, -37, -70, -69, -7,
	-94, -93, 68, -93, -51, -101, -104, 118, 135, -123,
	100, -49, 154, -14, 140, 157, -7, 157, -22, -7,
	-7, 124, 89, 89, 177, 163, 177, -4, -7, -7,
	41, -27, -7, -7, -7, 41, 41, -28, -28, -57,
	-58, 55, -7, 162, 164, -55, 67, 85, -122, 134,
	50, -124, 93, -16, -46, 154, -49, -49, -63, -7,
	173, 157, 161, -7, -9, 154, 156, -7, -9, -9,
	-9, -9, -9, -9, 7, -7, 162, -76, -75, 11,
	33, -92, -35, 143, -7, -92, -35, -55, -104, -55,
	-55, -103, -102, -46, -106, -105, -46, 69, -16, -43,
	156, 157, 124, -7, -116, -116, -115, -49, -115, -30,
	145, -30, -66, 16, -12, -7, -57, -44, -49, -48,
	123, -44, -7, -51, 177, 160, -47, -47, 161, -7,
	161, 163, -9, -69, -97, -96, 110, -97, -7, -97,
	-97, -72, -55, -72, -72, 162, 164, 162, -108, -107,
	51, -7, 89, -35, -7, 156, -119, 108, 41, -7,
	41, -10, -47, 163, 163, -16, 154, 156, -7, -16,
	-16, 157, 161, -7, -95, -64, -121, -97, -72, -97,
	-97, -102, -7, -105, -99, -98, -17, -93, 157, -10,
	125, -59, -60, 74, -16, -49, -49, -7, 161, -51,
	-51, 161, -7, -97, -108, -30, 162, 58, 149, -109,
	145, -15, 157, -115, -61, -62, 56, -13, -52, 89,
	-47, -47, 157, 41, -98, -100, -46, -100, -72, 79,
	86, 89, -117, 95, -7, -123, -16, -16, -97, 124,
	79, -93, -55, 16, 69, -52, -52, 136, 31, 124,
	-109, -118, 140, -7, -7, -111, -101, -104, -112, -55,
	63, -72, 144, -110, 145, -55, -104, -55, -114, 145,
	-113, -7, -97, 79, 86, -55, 86, -55, 124, 79,
	79, 31, 124, 124, -112, 63, 63, -114, -113, -113,
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
	172, 173, 174, 175, 176, 177,
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
		//line n1ql.y:338
		{
			yylex.(*lexer).setStatement(yyS[yypt-0].statement)
		}
	case 2:
		//line n1ql.y:343
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
		//line n1ql.y:360
		{
			yyVAL.statement = algebra.NewExplain(yyS[yypt-0].statement)
		}
	case 8:
		//line n1ql.y:367
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
		//line n1ql.y:398
		{
			yyVAL.fullselect = algebra.NewSelect(yyS[yypt-3].subresult, yyS[yypt-2].order, yyS[yypt-0].expr, yyS[yypt-1].expr) /* OFFSET precedes LIMIT */
		}
	case 19:
		//line n1ql.y:404
		{
			yyVAL.subresult = yyS[yypt-0].subselect
		}
	case 20:
		//line n1ql.y:409
		{
			yyVAL.subresult = algebra.NewUnion(yyS[yypt-2].subresult, yyS[yypt-0].subselect)
		}
	case 21:
		//line n1ql.y:414
		{
			yyVAL.subresult = algebra.NewUnionAll(yyS[yypt-3].subresult, yyS[yypt-0].subselect)
		}
	case 22:
		//line n1ql.y:419
		{
			yyVAL.subresult = algebra.NewIntersect(yyS[yypt-2].subresult, yyS[yypt-0].subselect)
		}
	case 23:
		//line n1ql.y:424
		{
			yyVAL.subresult = algebra.NewIntersectAll(yyS[yypt-3].subresult, yyS[yypt-0].subselect)
		}
	case 24:
		//line n1ql.y:429
		{
			yyVAL.subresult = algebra.NewExcept(yyS[yypt-2].subresult, yyS[yypt-0].subselect)
		}
	case 25:
		//line n1ql.y:434
		{
			yyVAL.subresult = algebra.NewExceptAll(yyS[yypt-3].subresult, yyS[yypt-0].subselect)
		}
	case 26:
		yyVAL.subselect = yyS[yypt-0].subselect
	case 27:
		yyVAL.subselect = yyS[yypt-0].subselect
	case 28:
		//line n1ql.y:447
		{
			yyVAL.subselect = algebra.NewSubselect(yyS[yypt-4].fromTerm, yyS[yypt-3].bindings, yyS[yypt-2].expr, yyS[yypt-1].group, yyS[yypt-0].projection)
		}
	case 29:
		//line n1ql.y:454
		{
			yyVAL.subselect = algebra.NewSubselect(yyS[yypt-3].fromTerm, yyS[yypt-2].bindings, yyS[yypt-1].expr, yyS[yypt-0].group, yyS[yypt-4].projection)
		}
	case 30:
		//line n1ql.y:469
		{
			yyVAL.projection = yyS[yypt-0].projection
		}
	case 31:
		//line n1ql.y:476
		{
			yyVAL.projection = algebra.NewProjection(false, yyS[yypt-0].resultTerms)
		}
	case 32:
		//line n1ql.y:481
		{
			yyVAL.projection = algebra.NewProjection(true, yyS[yypt-0].resultTerms)
		}
	case 33:
		//line n1ql.y:486
		{
			yyVAL.projection = algebra.NewProjection(false, yyS[yypt-0].resultTerms)
		}
	case 34:
		//line n1ql.y:491
		{
			yyVAL.projection = algebra.NewRawProjection(false, yyS[yypt-0].expr)
		}
	case 35:
		//line n1ql.y:496
		{
			yyVAL.projection = algebra.NewRawProjection(true, yyS[yypt-0].expr)
		}
	case 38:
		//line n1ql.y:509
		{
			yyVAL.resultTerms = algebra.ResultTerms{yyS[yypt-0].resultTerm}
		}
	case 39:
		//line n1ql.y:514
		{
			yyVAL.resultTerms = append(yyS[yypt-2].resultTerms, yyS[yypt-0].resultTerm)
		}
	case 40:
		//line n1ql.y:521
		{
			yyVAL.resultTerm = algebra.NewResultTerm(nil, true, "")
		}
	case 41:
		//line n1ql.y:526
		{
			yyVAL.resultTerm = algebra.NewResultTerm(yyS[yypt-2].expr, true, "")
		}
	case 42:
		//line n1ql.y:531
		{
			yyVAL.resultTerm = algebra.NewResultTerm(yyS[yypt-1].expr, false, yyS[yypt-0].s)
		}
	case 43:
		//line n1ql.y:538
		{
			yyVAL.s = ""
		}
	case 44:
		yyVAL.s = yyS[yypt-0].s
	case 45:
		yyVAL.s = yyS[yypt-0].s
	case 46:
		//line n1ql.y:549
		{
			yyVAL.s = yyS[yypt-0].s
		}
	case 47:
		yyVAL.s = yyS[yypt-0].s
	case 48:
		//line n1ql.y:567
		{
			yyVAL.fromTerm = nil
		}
	case 49:
		yyVAL.fromTerm = yyS[yypt-0].fromTerm
	case 50:
		//line n1ql.y:576
		{
			yyVAL.fromTerm = yyS[yypt-0].fromTerm
		}
	case 51:
		//line n1ql.y:583
		{
			yyVAL.fromTerm = yyS[yypt-0].keyspaceTerm
		}
	case 52:
		//line n1ql.y:588
		{
			yyVAL.fromTerm = algebra.NewJoin(yyS[yypt-3].fromTerm, yyS[yypt-2].b, yyS[yypt-0].keyspaceTerm)
		}
	case 53:
		//line n1ql.y:593
		{
			yyVAL.fromTerm = algebra.NewNest(yyS[yypt-3].fromTerm, yyS[yypt-2].b, yyS[yypt-0].keyspaceTerm)
		}
	case 54:
		//line n1ql.y:598
		{
			yyVAL.fromTerm = algebra.NewUnnest(yyS[yypt-4].fromTerm, yyS[yypt-3].b, yyS[yypt-1].expr, yyS[yypt-0].s)
		}
	case 57:
		//line n1ql.y:611
		{
			yyVAL.keyspaceTerm = algebra.NewKeyspaceTerm("", yyS[yypt-3].s, yyS[yypt-2].path, yyS[yypt-1].s, yyS[yypt-0].expr)
		}
	case 58:
		//line n1ql.y:616
		{
			yyVAL.keyspaceTerm = algebra.NewKeyspaceTerm(yyS[yypt-5].s, yyS[yypt-3].s, yyS[yypt-2].path, yyS[yypt-1].s, yyS[yypt-0].expr)
		}
	case 59:
		//line n1ql.y:621
		{
			yyVAL.keyspaceTerm = algebra.NewKeyspaceTerm("#system", yyS[yypt-3].s, yyS[yypt-2].path, yyS[yypt-1].s, yyS[yypt-0].expr)
		}
	case 60:
		//line n1ql.y:628
		{
			yyVAL.keyspaceTerm = algebra.NewKeyspaceTerm("", yyS[yypt-3].s, yyS[yypt-2].path, yyS[yypt-1].s, yyS[yypt-0].expr)
		}
	case 61:
		//line n1ql.y:633
		{
			yyVAL.keyspaceTerm = algebra.NewKeyspaceTerm(yyS[yypt-5].s, yyS[yypt-3].s, yyS[yypt-2].path, yyS[yypt-1].s, yyS[yypt-0].expr)
		}
	case 62:
		//line n1ql.y:638
		{
			yyVAL.keyspaceTerm = algebra.NewKeyspaceTerm("#system", yyS[yypt-3].s, yyS[yypt-2].path, yyS[yypt-1].s, yyS[yypt-0].expr)
		}
	case 63:
		yyVAL.s = yyS[yypt-0].s
	case 64:
		yyVAL.s = yyS[yypt-0].s
	case 65:
		//line n1ql.y:653
		{
			yyVAL.path = nil
		}
	case 66:
		//line n1ql.y:658
		{
			yyVAL.path = yyS[yypt-0].path
		}
	case 67:
		//line n1ql.y:665
		{
			yyVAL.expr = nil
		}
	case 68:
		yyVAL.expr = yyS[yypt-0].expr
	case 69:
		//line n1ql.y:674
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 70:
		//line n1ql.y:681
		{
		}
	case 72:
		//line n1ql.y:689
		{
			yyVAL.b = false
		}
	case 73:
		//line n1ql.y:694
		{
			yyVAL.b = false
		}
	case 74:
		//line n1ql.y:699
		{
			yyVAL.b = true
		}
	case 77:
		//line n1ql.y:712
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 78:
		//line n1ql.y:726
		{
			yyVAL.bindings = nil
		}
	case 79:
		yyVAL.bindings = yyS[yypt-0].bindings
	case 80:
		//line n1ql.y:735
		{
			yyVAL.bindings = yyS[yypt-0].bindings
		}
	case 81:
		//line n1ql.y:742
		{
			yyVAL.bindings = expression.Bindings{yyS[yypt-0].binding}
		}
	case 82:
		//line n1ql.y:747
		{
			yyVAL.bindings = append(yyS[yypt-2].bindings, yyS[yypt-0].binding)
		}
	case 83:
		//line n1ql.y:754
		{
			yyVAL.binding = expression.NewBinding(yyS[yypt-2].s, yyS[yypt-0].expr)
		}
	case 84:
		//line n1ql.y:768
		{
			yyVAL.expr = nil
		}
	case 85:
		yyVAL.expr = yyS[yypt-0].expr
	case 86:
		//line n1ql.y:777
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 87:
		//line n1ql.y:791
		{
			yyVAL.group = nil
		}
	case 88:
		yyVAL.group = yyS[yypt-0].group
	case 89:
		//line n1ql.y:800
		{
			yyVAL.group = algebra.NewGroup(yyS[yypt-2].exprs, yyS[yypt-1].bindings, yyS[yypt-0].expr)
		}
	case 90:
		//line n1ql.y:807
		{
			yyVAL.exprs = expression.Expressions{yyS[yypt-0].expr}
		}
	case 91:
		//line n1ql.y:812
		{
			yyVAL.exprs = append(yyS[yypt-2].exprs, yyS[yypt-0].expr)
		}
	case 92:
		//line n1ql.y:819
		{
			yyVAL.bindings = nil
		}
	case 93:
		yyVAL.bindings = yyS[yypt-0].bindings
	case 94:
		//line n1ql.y:828
		{
			yyVAL.bindings = yyS[yypt-0].bindings
		}
	case 95:
		//line n1ql.y:835
		{
			yyVAL.expr = nil
		}
	case 96:
		yyVAL.expr = yyS[yypt-0].expr
	case 97:
		//line n1ql.y:844
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 98:
		//line n1ql.y:858
		{
			yyVAL.order = nil
		}
	case 99:
		yyVAL.order = yyS[yypt-0].order
	case 100:
		//line n1ql.y:867
		{
			yyVAL.order = algebra.NewOrder(yyS[yypt-0].sortTerms)
		}
	case 101:
		//line n1ql.y:874
		{
			yyVAL.sortTerms = algebra.SortTerms{yyS[yypt-0].sortTerm}
		}
	case 102:
		//line n1ql.y:879
		{
			yyVAL.sortTerms = append(yyS[yypt-2].sortTerms, yyS[yypt-0].sortTerm)
		}
	case 103:
		//line n1ql.y:886
		{
			yyVAL.sortTerm = algebra.NewSortTerm(yyS[yypt-1].expr, yyS[yypt-0].b)
		}
	case 104:
		//line n1ql.y:893
		{
			yyVAL.b = false
		}
	case 105:
		yyVAL.b = yyS[yypt-0].b
	case 106:
		//line n1ql.y:902
		{
			yyVAL.b = false
		}
	case 107:
		//line n1ql.y:907
		{
			yyVAL.b = true
		}
	case 108:
		//line n1ql.y:921
		{
			yyVAL.expr = nil
		}
	case 109:
		yyVAL.expr = yyS[yypt-0].expr
	case 110:
		//line n1ql.y:930
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 111:
		//line n1ql.y:944
		{
			yyVAL.expr = nil
		}
	case 112:
		yyVAL.expr = yyS[yypt-0].expr
	case 113:
		//line n1ql.y:953
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 114:
		//line n1ql.y:967
		{
			yyVAL.statement = algebra.NewInsertValues(yyS[yypt-3].keyspaceRef, yyS[yypt-2].expr, yyS[yypt-1].expr, yyS[yypt-0].projection)
		}
	case 115:
		//line n1ql.y:972
		{
			yyVAL.statement = algebra.NewInsertSelect(yyS[yypt-3].keyspaceRef, yyS[yypt-2].expr, yyS[yypt-1].fullselect, yyS[yypt-0].projection)
		}
	case 116:
		//line n1ql.y:979
		{
			yyVAL.keyspaceRef = algebra.NewKeyspaceRef(yyS[yypt-3].s, yyS[yypt-1].s, yyS[yypt-0].s)
		}
	case 117:
		//line n1ql.y:984
		{
			yyVAL.keyspaceRef = algebra.NewKeyspaceRef("", yyS[yypt-1].s, yyS[yypt-0].s)
		}
	case 118:
		//line n1ql.y:991
		{
			yyVAL.expr = nil
		}
	case 119:
		yyVAL.expr = yyS[yypt-0].expr
	case 120:
		//line n1ql.y:1000
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 121:
		//line n1ql.y:1007
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 122:
		//line n1ql.y:1014
		{
			yyVAL.projection = nil
		}
	case 123:
		yyVAL.projection = yyS[yypt-0].projection
	case 124:
		//line n1ql.y:1023
		{
			yyVAL.projection = yyS[yypt-0].projection
		}
	case 125:
		//line n1ql.y:1030
		{
			yyVAL.projection = algebra.NewProjection(false, yyS[yypt-0].resultTerms)
		}
	case 126:
		//line n1ql.y:1035
		{
			yyVAL.projection = algebra.NewRawProjection(false, yyS[yypt-0].expr)
		}
	case 127:
		//line n1ql.y:1049
		{
			yyVAL.statement = algebra.NewUpsertValues(yyS[yypt-3].keyspaceRef, yyS[yypt-2].expr, yyS[yypt-1].expr, yyS[yypt-0].projection)
		}
	case 128:
		//line n1ql.y:1054
		{
			yyVAL.statement = algebra.NewUpsertSelect(yyS[yypt-3].keyspaceRef, yyS[yypt-2].expr, yyS[yypt-1].fullselect, yyS[yypt-0].projection)
		}
	case 129:
		//line n1ql.y:1068
		{
			yyVAL.statement = algebra.NewDelete(yyS[yypt-4].keyspaceRef, yyS[yypt-3].expr, yyS[yypt-2].expr, yyS[yypt-1].expr, yyS[yypt-0].projection)
		}
	case 130:
		//line n1ql.y:1082
		{
			yyVAL.statement = algebra.NewUpdate(yyS[yypt-6].keyspaceRef, yyS[yypt-5].expr, yyS[yypt-4].set, yyS[yypt-3].unset, yyS[yypt-2].expr, yyS[yypt-1].expr, yyS[yypt-0].projection)
		}
	case 131:
		//line n1ql.y:1087
		{
			yyVAL.statement = algebra.NewUpdate(yyS[yypt-5].keyspaceRef, yyS[yypt-4].expr, yyS[yypt-3].set, nil, yyS[yypt-2].expr, yyS[yypt-1].expr, yyS[yypt-0].projection)
		}
	case 132:
		//line n1ql.y:1092
		{
			yyVAL.statement = algebra.NewUpdate(yyS[yypt-5].keyspaceRef, yyS[yypt-4].expr, nil, yyS[yypt-3].unset, yyS[yypt-2].expr, yyS[yypt-1].expr, yyS[yypt-0].projection)
		}
	case 133:
		//line n1ql.y:1099
		{
			yyVAL.set = algebra.NewSet(yyS[yypt-0].setTerms)
		}
	case 134:
		//line n1ql.y:1106
		{
			yyVAL.setTerms = algebra.SetTerms{yyS[yypt-0].setTerm}
		}
	case 135:
		//line n1ql.y:1111
		{
			yyVAL.setTerms = append(yyS[yypt-2].setTerms, yyS[yypt-0].setTerm)
		}
	case 136:
		//line n1ql.y:1118
		{
			yyVAL.setTerm = algebra.NewSetTerm(yyS[yypt-3].path, yyS[yypt-1].expr, yyS[yypt-0].updateFor)
		}
	case 137:
		//line n1ql.y:1125
		{
			yyVAL.updateFor = nil
		}
	case 138:
		yyVAL.updateFor = yyS[yypt-0].updateFor
	case 139:
		//line n1ql.y:1134
		{
			yyVAL.updateFor = algebra.NewUpdateFor(yyS[yypt-2].bindings, yyS[yypt-1].expr)
		}
	case 140:
		//line n1ql.y:1141
		{
			yyVAL.bindings = expression.Bindings{yyS[yypt-0].binding}
		}
	case 141:
		//line n1ql.y:1146
		{
			yyVAL.bindings = append(yyS[yypt-2].bindings, yyS[yypt-0].binding)
		}
	case 142:
		//line n1ql.y:1153
		{
			yyVAL.binding = expression.NewBinding(yyS[yypt-2].s, yyS[yypt-0].expr)
		}
	case 143:
		//line n1ql.y:1158
		{
			yyVAL.binding = expression.NewDescendantBinding(yyS[yypt-2].s, yyS[yypt-0].expr)
		}
	case 144:
		yyVAL.s = yyS[yypt-0].s
	case 145:
		//line n1ql.y:1169
		{
			yyVAL.expr = yyS[yypt-0].path
		}
	case 146:
		//line n1ql.y:1176
		{
			yyVAL.expr = nil
		}
	case 147:
		//line n1ql.y:1181
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 148:
		//line n1ql.y:1188
		{
			yyVAL.unset = algebra.NewUnset(yyS[yypt-0].unsetTerms)
		}
	case 149:
		//line n1ql.y:1195
		{
			yyVAL.unsetTerms = algebra.UnsetTerms{yyS[yypt-0].unsetTerm}
		}
	case 150:
		//line n1ql.y:1200
		{
			yyVAL.unsetTerms = append(yyS[yypt-2].unsetTerms, yyS[yypt-0].unsetTerm)
		}
	case 151:
		//line n1ql.y:1207
		{
			yyVAL.unsetTerm = algebra.NewUnsetTerm(yyS[yypt-1].path, yyS[yypt-0].updateFor)
		}
	case 152:
		//line n1ql.y:1221
		{
			source := algebra.NewMergeSourceFrom(yyS[yypt-5].keyspaceTerm, "")
			yyVAL.statement = algebra.NewMerge(yyS[yypt-7].keyspaceRef, source, yyS[yypt-3].expr, yyS[yypt-2].mergeActions, yyS[yypt-1].expr, yyS[yypt-0].projection)
		}
	case 153:
		//line n1ql.y:1227
		{
			source := algebra.NewMergeSourceSelect(yyS[yypt-7].fullselect, yyS[yypt-5].s)
			yyVAL.statement = algebra.NewMerge(yyS[yypt-10].keyspaceRef, source, yyS[yypt-3].expr, yyS[yypt-2].mergeActions, yyS[yypt-1].expr, yyS[yypt-0].projection)
		}
	case 154:
		//line n1ql.y:1235
		{
			yyVAL.mergeActions = algebra.NewMergeActions(nil, nil, nil)
		}
	case 155:
		//line n1ql.y:1240
		{
			yyVAL.mergeActions = algebra.NewMergeActions(yyS[yypt-1].mergeUpdate, yyS[yypt-0].mergeActions.Delete(), yyS[yypt-0].mergeActions.Insert())
		}
	case 156:
		//line n1ql.y:1245
		{
			yyVAL.mergeActions = algebra.NewMergeActions(nil, yyS[yypt-1].mergeDelete, yyS[yypt-0].mergeInsert)
		}
	case 157:
		//line n1ql.y:1250
		{
			yyVAL.mergeActions = algebra.NewMergeActions(nil, nil, yyS[yypt-0].mergeInsert)
		}
	case 158:
		//line n1ql.y:1257
		{
			yyVAL.mergeActions = algebra.NewMergeActions(nil, nil, nil)
		}
	case 159:
		//line n1ql.y:1262
		{
			yyVAL.mergeActions = algebra.NewMergeActions(nil, yyS[yypt-1].mergeDelete, yyS[yypt-0].mergeInsert)
		}
	case 160:
		//line n1ql.y:1267
		{
			yyVAL.mergeActions = algebra.NewMergeActions(nil, nil, yyS[yypt-0].mergeInsert)
		}
	case 161:
		//line n1ql.y:1274
		{
			yyVAL.mergeInsert = nil
		}
	case 162:
		//line n1ql.y:1279
		{
			yyVAL.mergeInsert = yyS[yypt-0].mergeInsert
		}
	case 163:
		//line n1ql.y:1286
		{
			yyVAL.mergeUpdate = algebra.NewMergeUpdate(yyS[yypt-1].set, nil, yyS[yypt-0].expr)
		}
	case 164:
		//line n1ql.y:1291
		{
			yyVAL.mergeUpdate = algebra.NewMergeUpdate(yyS[yypt-2].set, yyS[yypt-1].unset, yyS[yypt-0].expr)
		}
	case 165:
		//line n1ql.y:1296
		{
			yyVAL.mergeUpdate = algebra.NewMergeUpdate(nil, yyS[yypt-1].unset, yyS[yypt-0].expr)
		}
	case 166:
		//line n1ql.y:1303
		{
			yyVAL.mergeDelete = algebra.NewMergeDelete(yyS[yypt-0].expr)
		}
	case 167:
		//line n1ql.y:1310
		{
			yyVAL.mergeInsert = algebra.NewMergeInsert(yyS[yypt-1].expr, yyS[yypt-0].expr)
		}
	case 168:
		//line n1ql.y:1324
		{
			yyVAL.statement = algebra.NewCreatePrimaryIndex(yyS[yypt-0].keyspaceRef)
		}
	case 169:
		//line n1ql.y:1329
		{
			yyVAL.statement = algebra.NewCreateIndex(yyS[yypt-8].s, yyS[yypt-6].keyspaceRef, yyS[yypt-4].exprs, yyS[yypt-2].expr, yyS[yypt-1].expr, yyS[yypt-0].indexType)
		}
	case 170:
		yyVAL.s = yyS[yypt-0].s
	case 171:
		//line n1ql.y:1340
		{
			yyVAL.keyspaceRef = algebra.NewKeyspaceRef("", yyS[yypt-0].s, "")
		}
	case 172:
		//line n1ql.y:1345
		{
			yyVAL.keyspaceRef = algebra.NewKeyspaceRef(yyS[yypt-2].s, yyS[yypt-0].s, "")
		}
	case 173:
		//line n1ql.y:1352
		{
			yyVAL.expr = nil
		}
	case 174:
		//line n1ql.y:1357
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 175:
		//line n1ql.y:1364
		{
			yyVAL.indexType = datastore.VIEW
		}
	case 176:
		//line n1ql.y:1369
		{
			yyVAL.indexType = datastore.VIEW
		}
	case 177:
		//line n1ql.y:1383
		{
			yyVAL.statement = algebra.NewDropIndex(yyS[yypt-2].keyspaceRef, yyS[yypt-0].s)
		}
	case 178:
		//line n1ql.y:1396
		{
			yyVAL.statement = algebra.NewAlterIndex(yyS[yypt-3].keyspaceRef, yyS[yypt-1].s, yyS[yypt-0].s)
		}
	case 179:
		//line n1ql.y:1402
		{
			yyVAL.s = ""
		}
	case 180:
		//line n1ql.y:1407
		{
			yyVAL.s = yyS[yypt-0].s
		}
	case 181:
		//line n1ql.y:1421
		{
			yyVAL.path = expression.NewIdentifier(yyS[yypt-0].s)
		}
	case 182:
		//line n1ql.y:1426
		{
			yyVAL.path = expression.NewField(yyS[yypt-2].path, expression.NewFieldName(yyS[yypt-0].s))
		}
	case 183:
		//line n1ql.y:1431
		{
			yyVAL.path = expression.NewField(yyS[yypt-4].path, yyS[yypt-1].expr)
		}
	case 184:
		//line n1ql.y:1436
		{
			yyVAL.path = expression.NewElement(yyS[yypt-3].path, yyS[yypt-1].expr)
		}
	case 185:
		yyVAL.expr = yyS[yypt-0].expr
	case 186:
		//line n1ql.y:1453
		{
			yyVAL.expr = expression.NewField(yyS[yypt-2].expr, expression.NewFieldName(yyS[yypt-0].s))
		}
	case 187:
		//line n1ql.y:1458
		{
			yyVAL.expr = expression.NewField(yyS[yypt-4].expr, yyS[yypt-1].expr)
		}
	case 188:
		//line n1ql.y:1463
		{
			yyVAL.expr = expression.NewElement(yyS[yypt-3].expr, yyS[yypt-1].expr)
		}
	case 189:
		//line n1ql.y:1468
		{
			yyVAL.expr = expression.NewSlice(yyS[yypt-4].expr, yyS[yypt-2].expr)
		}
	case 190:
		//line n1ql.y:1473
		{
			yyVAL.expr = expression.NewSlice(yyS[yypt-5].expr, yyS[yypt-3].expr, yyS[yypt-1].expr)
		}
	case 191:
		//line n1ql.y:1479
		{
			yyVAL.expr = expression.NewAdd(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 192:
		//line n1ql.y:1484
		{
			yyVAL.expr = expression.NewSub(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 193:
		//line n1ql.y:1489
		{
			yyVAL.expr = expression.NewMult(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 194:
		//line n1ql.y:1494
		{
			yyVAL.expr = expression.NewDiv(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 195:
		//line n1ql.y:1499
		{
			yyVAL.expr = expression.NewMod(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 196:
		//line n1ql.y:1505
		{
			yyVAL.expr = expression.NewConcat(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 197:
		//line n1ql.y:1511
		{
			yyVAL.expr = expression.NewAnd(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 198:
		//line n1ql.y:1516
		{
			yyVAL.expr = expression.NewOr(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 199:
		//line n1ql.y:1521
		{
			yyVAL.expr = expression.NewNot(yyS[yypt-0].expr)
		}
	case 200:
		//line n1ql.y:1527
		{
			yyVAL.expr = expression.NewEq(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 201:
		//line n1ql.y:1532
		{
			yyVAL.expr = expression.NewEq(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 202:
		//line n1ql.y:1537
		{
			yyVAL.expr = expression.NewNE(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 203:
		//line n1ql.y:1542
		{
			yyVAL.expr = expression.NewLT(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 204:
		//line n1ql.y:1547
		{
			yyVAL.expr = expression.NewGT(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 205:
		//line n1ql.y:1552
		{
			yyVAL.expr = expression.NewLE(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 206:
		//line n1ql.y:1557
		{
			yyVAL.expr = expression.NewGE(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 207:
		//line n1ql.y:1562
		{
			yyVAL.expr = expression.NewBetween(yyS[yypt-4].expr, yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 208:
		//line n1ql.y:1567
		{
			yyVAL.expr = expression.NewNotBetween(yyS[yypt-5].expr, yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 209:
		//line n1ql.y:1572
		{
			yyVAL.expr = expression.NewLike(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 210:
		//line n1ql.y:1577
		{
			yyVAL.expr = expression.NewNotLike(yyS[yypt-3].expr, yyS[yypt-0].expr)
		}
	case 211:
		//line n1ql.y:1582
		{
			yyVAL.expr = expression.NewIn(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 212:
		//line n1ql.y:1587
		{
			yyVAL.expr = expression.NewNotIn(yyS[yypt-3].expr, yyS[yypt-0].expr)
		}
	case 213:
		//line n1ql.y:1592
		{
			yyVAL.expr = expression.NewWithin(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 214:
		//line n1ql.y:1597
		{
			yyVAL.expr = expression.NewNotWithin(yyS[yypt-3].expr, yyS[yypt-0].expr)
		}
	case 215:
		//line n1ql.y:1602
		{
			yyVAL.expr = expression.NewIsNull(yyS[yypt-2].expr)
		}
	case 216:
		//line n1ql.y:1607
		{
			yyVAL.expr = expression.NewIsNotNull(yyS[yypt-3].expr)
		}
	case 217:
		//line n1ql.y:1612
		{
			yyVAL.expr = expression.NewIsMissing(yyS[yypt-2].expr)
		}
	case 218:
		//line n1ql.y:1617
		{
			yyVAL.expr = expression.NewIsNotMissing(yyS[yypt-3].expr)
		}
	case 219:
		//line n1ql.y:1622
		{
			yyVAL.expr = expression.NewIsValued(yyS[yypt-2].expr)
		}
	case 220:
		//line n1ql.y:1627
		{
			yyVAL.expr = expression.NewIsNotValued(yyS[yypt-3].expr)
		}
	case 221:
		//line n1ql.y:1632
		{
			yyVAL.expr = expression.NewExists(yyS[yypt-0].expr)
		}
	case 222:
		yyVAL.expr = yyS[yypt-0].expr
	case 223:
		//line n1ql.y:1643
		{
			yyVAL.expr = expression.NewIdentifier(yyS[yypt-0].s)
		}
	case 224:
		yyVAL.expr = yyS[yypt-0].expr
	case 225:
		//line n1ql.y:1652
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
		//line n1ql.y:1671
		{
			yyVAL.expr = expression.NewField(yyS[yypt-2].expr, expression.NewFieldName(yyS[yypt-0].s))
		}
	case 231:
		//line n1ql.y:1676
		{
			yyVAL.expr = expression.NewField(yyS[yypt-4].expr, yyS[yypt-1].expr)
		}
	case 232:
		//line n1ql.y:1681
		{
			yyVAL.expr = expression.NewElement(yyS[yypt-3].expr, yyS[yypt-1].expr)
		}
	case 233:
		//line n1ql.y:1686
		{
			yyVAL.expr = expression.NewSlice(yyS[yypt-4].expr, yyS[yypt-2].expr)
		}
	case 234:
		//line n1ql.y:1691
		{
			yyVAL.expr = expression.NewSlice(yyS[yypt-5].expr, yyS[yypt-3].expr, yyS[yypt-1].expr)
		}
	case 235:
		//line n1ql.y:1697
		{
			yyVAL.expr = expression.NewAdd(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 236:
		//line n1ql.y:1702
		{
			yyVAL.expr = expression.NewSub(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 237:
		//line n1ql.y:1707
		{
			yyVAL.expr = expression.NewMult(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 238:
		//line n1ql.y:1712
		{
			yyVAL.expr = expression.NewDiv(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 239:
		//line n1ql.y:1717
		{
			yyVAL.expr = expression.NewMod(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 240:
		//line n1ql.y:1723
		{
			yyVAL.expr = expression.NewConcat(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 241:
		//line n1ql.y:1737
		{
			yyVAL.expr = expression.NULL_EXPR
		}
	case 242:
		//line n1ql.y:1742
		{
			yyVAL.expr = expression.FALSE_EXPR
		}
	case 243:
		//line n1ql.y:1747
		{
			yyVAL.expr = expression.TRUE_EXPR
		}
	case 244:
		//line n1ql.y:1752
		{
			yyVAL.expr = expression.NewConstant(value.NewValue(yyS[yypt-0].f))
		}
	case 245:
		//line n1ql.y:1757
		{
			yyVAL.expr = expression.NewConstant(value.NewValue(yyS[yypt-0].n))
		}
	case 246:
		//line n1ql.y:1762
		{
			yyVAL.expr = expression.NewConstant(value.NewValue(yyS[yypt-0].s))
		}
	case 247:
		yyVAL.expr = yyS[yypt-0].expr
	case 248:
		yyVAL.expr = yyS[yypt-0].expr
	case 249:
		//line n1ql.y:1773
		{
			yyVAL.expr = expression.NewObjectConstruct(yyS[yypt-1].bindings)
		}
	case 250:
		//line n1ql.y:1780
		{
			yyVAL.bindings = nil
		}
	case 251:
		yyVAL.bindings = yyS[yypt-0].bindings
	case 252:
		//line n1ql.y:1789
		{
			yyVAL.bindings = expression.Bindings{yyS[yypt-0].binding}
		}
	case 253:
		//line n1ql.y:1794
		{
			yyVAL.bindings = append(yyS[yypt-2].bindings, yyS[yypt-0].binding)
		}
	case 254:
		//line n1ql.y:1801
		{
			yyVAL.binding = expression.NewBinding(yyS[yypt-2].s, yyS[yypt-0].expr)
		}
	case 255:
		//line n1ql.y:1808
		{
			yyVAL.expr = expression.NewArrayConstruct(yyS[yypt-1].exprs...)
		}
	case 256:
		//line n1ql.y:1815
		{
			yyVAL.exprs = nil
		}
	case 257:
		yyVAL.exprs = yyS[yypt-0].exprs
	case 258:
		//line n1ql.y:1831
		{
			yyVAL.expr = yyS[yypt-1].expr
		}
	case 259:
		yyVAL.expr = yyS[yypt-0].expr
	case 260:
		yyVAL.expr = yyS[yypt-0].expr
	case 261:
		//line n1ql.y:1844
		{
			yyVAL.expr = expression.NewSimpleCase(yyS[yypt-2].expr, yyS[yypt-1].whenTerms, yyS[yypt-0].expr)
		}
	case 262:
		//line n1ql.y:1851
		{
			yyVAL.whenTerms = expression.WhenTerms{&expression.WhenTerm{yyS[yypt-2].expr, yyS[yypt-0].expr}}
		}
	case 263:
		//line n1ql.y:1856
		{
			yyVAL.whenTerms = append(yyS[yypt-4].whenTerms, &expression.WhenTerm{yyS[yypt-2].expr, yyS[yypt-0].expr})
		}
	case 264:
		//line n1ql.y:1864
		{
			yyVAL.expr = expression.NewSearchedCase(yyS[yypt-1].whenTerms, yyS[yypt-0].expr)
		}
	case 265:
		//line n1ql.y:1871
		{
			yyVAL.expr = nil
		}
	case 266:
		//line n1ql.y:1876
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 267:
		//line n1ql.y:1890
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
		//line n1ql.y:1909
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
		//line n1ql.y:1924
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
		//line n1ql.y:1958
		{
			yyVAL.expr = expression.NewAny(yyS[yypt-2].bindings, yyS[yypt-1].expr)
		}
	case 274:
		//line n1ql.y:1963
		{
			yyVAL.expr = expression.NewAny(yyS[yypt-2].bindings, yyS[yypt-1].expr)
		}
	case 275:
		//line n1ql.y:1968
		{
			yyVAL.expr = expression.NewEvery(yyS[yypt-2].bindings, yyS[yypt-1].expr)
		}
	case 276:
		//line n1ql.y:1975
		{
			yyVAL.bindings = expression.Bindings{yyS[yypt-0].binding}
		}
	case 277:
		//line n1ql.y:1980
		{
			yyVAL.bindings = append(yyS[yypt-2].bindings, yyS[yypt-0].binding)
		}
	case 278:
		//line n1ql.y:1987
		{
			yyVAL.binding = expression.NewBinding(yyS[yypt-2].s, yyS[yypt-0].expr)
		}
	case 279:
		//line n1ql.y:1992
		{
			yyVAL.binding = expression.NewDescendantBinding(yyS[yypt-2].s, yyS[yypt-0].expr)
		}
	case 280:
		//line n1ql.y:1999
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 281:
		//line n1ql.y:2006
		{
			yyVAL.expr = expression.NewArray(yyS[yypt-4].expr, yyS[yypt-2].bindings, yyS[yypt-1].expr)
		}
	case 282:
		//line n1ql.y:2011
		{
			yyVAL.expr = expression.NewFirst(yyS[yypt-4].expr, yyS[yypt-2].bindings, yyS[yypt-1].expr)
		}
	case 283:
		//line n1ql.y:2025
		{
			yyVAL.expr = yyS[yypt-1].expr
		}
	case 284:
		yyVAL.expr = yyS[yypt-0].expr
	case 285:
		//line n1ql.y:2034
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
