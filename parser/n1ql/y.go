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
const LSM = 57419
const MAP = 57420
const MAPPING = 57421
const MATCHED = 57422
const MATERIALIZED = 57423
const MERGE = 57424
const MINUS = 57425
const MISSING = 57426
const NAMESPACE = 57427
const NEST = 57428
const NOT = 57429
const NULL = 57430
const OBJECT = 57431
const OFFSET = 57432
const ON = 57433
const OPTION = 57434
const OR = 57435
const ORDER = 57436
const OUTER = 57437
const OVER = 57438
const PARTITION = 57439
const PASSWORD = 57440
const PATH = 57441
const POOL = 57442
const PREPARE = 57443
const PRIMARY = 57444
const PRIVATE = 57445
const PRIVILEGE = 57446
const PROCEDURE = 57447
const PUBLIC = 57448
const RAW = 57449
const REALM = 57450
const REDUCE = 57451
const RENAME = 57452
const RETURN = 57453
const RETURNING = 57454
const REVOKE = 57455
const RIGHT = 57456
const ROLE = 57457
const ROLLBACK = 57458
const SATISFIES = 57459
const SCHEMA = 57460
const SELECT = 57461
const SET = 57462
const SHOW = 57463
const SOME = 57464
const START = 57465
const STATISTICS = 57466
const SYSTEM = 57467
const THEN = 57468
const TO = 57469
const TRANSACTION = 57470
const TRIGGER = 57471
const TRUE = 57472
const TRUNCATE = 57473
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
const NAMED_PARAM = 57500
const POSITIONAL_PARAM = 57501
const LPAREN = 57502
const RPAREN = 57503
const LBRACE = 57504
const RBRACE = 57505
const LBRACKET = 57506
const RBRACKET = 57507
const RBRACKET_ICASE = 57508
const COMMA = 57509
const COLON = 57510
const INTERESECT = 57511
const EQ = 57512
const DEQ = 57513
const NE = 57514
const LT = 57515
const GT = 57516
const LE = 57517
const GE = 57518
const CONCAT = 57519
const PLUS = 57520
const STAR = 57521
const DIV = 57522
const MOD = 57523
const UMINUS = 57524
const DOT = 57525

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
	"LSM",
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
	"NAMED_PARAM",
	"POSITIONAL_PARAM",
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
	-1, 21,
	160, 288,
	-2, 235,
	-1, 109,
	168, 63,
	-2, 64,
	-1, 146,
	50, 72,
	67, 72,
	86, 72,
	135, 72,
	-2, 50,
	-1, 173,
	170, 0,
	171, 0,
	172, 0,
	-2, 211,
	-1, 174,
	170, 0,
	171, 0,
	172, 0,
	-2, 212,
	-1, 175,
	170, 0,
	171, 0,
	172, 0,
	-2, 213,
	-1, 176,
	173, 0,
	174, 0,
	175, 0,
	176, 0,
	-2, 214,
	-1, 177,
	173, 0,
	174, 0,
	175, 0,
	176, 0,
	-2, 215,
	-1, 178,
	173, 0,
	174, 0,
	175, 0,
	176, 0,
	-2, 216,
	-1, 179,
	173, 0,
	174, 0,
	175, 0,
	176, 0,
	-2, 217,
	-1, 186,
	75, 0,
	-2, 220,
	-1, 187,
	58, 0,
	150, 0,
	-2, 222,
	-1, 188,
	58, 0,
	150, 0,
	-2, 224,
	-1, 282,
	75, 0,
	-2, 221,
	-1, 283,
	58, 0,
	150, 0,
	-2, 223,
	-1, 284,
	58, 0,
	150, 0,
	-2, 225,
}

const yyNprod = 304
const yyPrivate = 57344

var yyTokenNames []string
var yyStates []string

const yyLast = 2816

var yyAct = []int{

	160, 3, 585, 574, 438, 583, 575, 303, 304, 193,
	93, 94, 459, 531, 500, 210, 299, 521, 307, 135,
	539, 212, 251, 493, 395, 96, 258, 211, 412, 227,
	397, 452, 206, 394, 296, 12, 152, 131, 155, 338,
	420, 67, 156, 147, 383, 252, 133, 134, 222, 230,
	115, 128, 53, 119, 259, 454, 325, 323, 343, 132,
	524, 87, 8, 139, 140, 505, 525, 237, 470, 469,
	428, 324, 164, 165, 166, 167, 168, 169, 170, 171,
	172, 173, 174, 175, 176, 177, 178, 179, 120, 427,
	186, 187, 188, 261, 274, 260, 241, 236, 234, 209,
	71, 342, 274, 71, 262, 496, 90, 137, 138, 277,
	278, 279, 132, 273, 92, 74, 75, 76, 224, 70,
	428, 273, 70, 89, 161, 162, 450, 451, 449, 378,
	238, 73, 163, 413, 413, 315, 235, 472, 473, 427,
	354, 313, 108, 180, 225, 474, 240, 360, 248, 87,
	181, 196, 198, 200, 518, 240, 266, 461, 238, 111,
	276, 366, 367, 136, 269, 253, 92, 214, 428, 368,
	161, 162, 213, 228, 150, 233, 268, 423, 163, 150,
	310, 109, 129, 73, 282, 283, 284, 427, 250, 558,
	263, 265, 264, 306, 90, 498, 291, 519, 91, 584,
	242, 223, 92, 297, 250, 109, 579, 522, 109, 402,
	109, 148, 71, 460, 182, 141, 312, 208, 314, 73,
	502, 301, 317, 239, 318, 77, 72, 74, 75, 76,
	68, 70, 305, 598, 597, 311, 286, 327, 302, 328,
	285, 274, 331, 332, 333, 181, 593, 565, 306, 554,
	292, 341, 293, 564, 294, 275, 277, 278, 279, 184,
	273, 344, 68, 497, 71, 358, 69, 231, 231, 316,
	440, 308, 364, 463, 352, 369, 183, 77, 72, 74,
	75, 76, 359, 70, 353, 107, 91, 326, 330, 520,
	124, 377, 243, 336, 337, 287, 116, 69, 351, 191,
	71, 386, 190, 189, 547, 357, 100, 221, 122, 389,
	391, 392, 390, 77, 72, 74, 75, 76, 214, 70,
	405, 290, 385, 130, 532, 400, 281, 99, 201, 69,
	376, 398, 123, 545, 384, 181, 456, 388, 181, 181,
	181, 181, 181, 181, 418, 322, 387, 321, 425, 320,
	121, 185, 309, 409, 149, 411, 401, 102, 192, 563,
	254, 244, 245, 591, 253, 595, 414, 594, 588, 433,
	406, 407, 408, 555, 431, 589, 68, 195, 143, 297,
	415, 560, 429, 430, 419, 426, 442, 424, 417, 441,
	399, 543, 443, 444, 300, 110, 98, 446, 544, 445,
	455, 447, 448, 355, 356, 458, 276, 104, 349, 103,
	232, 232, 220, 437, 465, 601, 365, 132, 276, 370,
	371, 372, 373, 374, 375, 345, 66, 256, 340, 475,
	600, 576, 229, 199, 226, 216, 481, 257, 457, 181,
	125, 471, 197, 69, 346, 476, 477, 529, 468, 68,
	485, 490, 487, 488, 467, 148, 486, 105, 537, 466,
	464, 335, 501, 231, 231, 231, 334, 410, 329, 219,
	596, 559, 416, 495, 494, 509, 398, 483, 202, 484,
	272, 68, 491, 489, 506, 514, 2, 274, 421, 421,
	68, 515, 350, 348, 145, 347, 1, 499, 95, 274,
	280, 275, 277, 278, 279, 557, 273, 462, 546, 511,
	512, 571, 280, 275, 277, 278, 279, 578, 273, 453,
	436, 517, 516, 396, 523, 393, 501, 253, 530, 492,
	549, 542, 526, 439, 533, 534, 482, 298, 494, 36,
	548, 541, 538, 35, 540, 540, 34, 553, 69, 551,
	552, 550, 18, 17, 16, 15, 276, 69, 78, 501,
	569, 570, 556, 14, 87, 561, 562, 73, 13, 7,
	567, 572, 573, 568, 566, 6, 577, 586, 106, 580,
	582, 581, 587, 5, 4, 379, 380, 288, 590, 289,
	78, 194, 295, 592, 97, 101, 87, 151, 149, 528,
	599, 586, 586, 603, 604, 602, 232, 232, 232, 90,
	527, 504, 507, 508, 503, 339, 249, 92, 142, 207,
	255, 144, 78, 146, 64, 213, 89, 65, 87, 28,
	118, 422, 422, 27, 73, 48, 23, 274, 88, 51,
	50, 90, 26, 114, 79, 113, 112, 25, 71, 92,
	280, 275, 277, 278, 279, 126, 273, 127, 89, 22,
	45, 44, 72, 74, 75, 76, 73, 70, 20, 19,
	88, 0, 0, 90, 0, 0, 79, 0, 0, 0,
	0, 92, 203, 204, 205, 0, 0, 0, 0, 215,
	89, 0, 0, 0, 0, 0, 0, 0, 73, 0,
	0, 91, 88, 0, 0, 0, 0, 0, 79, 0,
	0, 0, 0, 0, 0, 71, 535, 536, 0, 0,
	0, 80, 81, 82, 83, 84, 85, 86, 77, 72,
	74, 75, 76, 91, 70, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 78, 71, 478, 479,
	0, 0, 87, 80, 81, 82, 83, 84, 85, 86,
	77, 72, 74, 75, 76, 91, 70, 0, 0, 0,
	0, 214, 0, 0, 0, 0, 0, 0, 78, 71,
	0, 0, 381, 0, 87, 80, 81, 82, 83, 84,
	85, 86, 77, 72, 74, 75, 76, 90, 70, 0,
	0, 0, 0, 0, 382, 92, 0, 0, 0, 0,
	78, 0, 0, 0, 89, 0, 87, 0, 0, 0,
	0, 0, 73, 56, 0, 0, 88, 0, 0, 90,
	0, 0, 79, 0, 0, 0, 0, 92, 0, 0,
	0, 0, 0, 0, 54, 0, 89, 0, 0, 31,
	0, 0, 0, 0, 73, 55, 0, 0, 88, 0,
	0, 90, 0, 0, 79, 11, 0, 0, 0, 92,
	68, 0, 0, 0, 0, 0, 0, 0, 89, 0,
	0, 29, 0, 0, 0, 0, 73, 0, 0, 91,
	88, 0, 0, 0, 0, 0, 79, 0, 0, 0,
	33, 0, 0, 71, 434, 0, 0, 435, 0, 80,
	81, 82, 83, 84, 85, 86, 77, 72, 74, 75,
	76, 91, 70, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 78, 71, 0, 69, 0, 0,
	87, 80, 81, 82, 83, 84, 85, 86, 77, 72,
	74, 75, 76, 91, 70, 32, 30, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 71, 361, 362,
	0, 0, 0, 80, 81, 82, 83, 84, 85, 86,
	77, 72, 74, 75, 76, 90, 70, 0, 78, 0,
	0, 213, 0, 92, 87, 0, 0, 0, 0, 0,
	0, 0, 89, 0, 0, 0, 0, 0, 0, 0,
	73, 0, 0, 0, 88, 0, 0, 0, 0, 0,
	79, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 90,
	0, 0, 0, 0, 0, 0, 0, 92, 0, 0,
	0, 0, 0, 0, 0, 0, 89, 0, 0, 0,
	78, 0, 0, 0, 73, 0, 87, 0, 88, 0,
	0, 0, 0, 0, 79, 0, 0, 91, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 71, 270, 0, 0, 271, 0, 80, 81, 82,
	83, 84, 85, 86, 77, 72, 74, 75, 76, 0,
	70, 90, 0, 0, 0, 0, 0, 0, 0, 92,
	0, 78, 0, 0, 0, 0, 0, 87, 89, 0,
	0, 91, 0, 0, 0, 0, 73, 214, 0, 0,
	88, 0, 0, 0, 0, 71, 79, 0, 0, 0,
	0, 80, 81, 82, 83, 84, 85, 86, 77, 72,
	74, 75, 76, 0, 267, 454, 0, 0, 0, 0,
	0, 0, 90, 0, 0, 0, 0, 0, 0, 0,
	92, 0, 0, 0, 0, 0, 78, 0, 0, 89,
	0, 0, 87, 0, 0, 0, 0, 73, 0, 0,
	250, 88, 0, 91, 0, 0, 0, 79, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 71, 0, 0,
	0, 0, 0, 80, 81, 82, 83, 84, 85, 86,
	77, 72, 74, 75, 76, 0, 70, 90, 0, 0,
	0, 0, 0, 0, 0, 92, 0, 78, 0, 0,
	0, 0, 0, 87, 89, 0, 0, 0, 0, 0,
	0, 0, 73, 0, 91, 0, 88, 0, 0, 0,
	0, 0, 79, 0, 0, 0, 0, 0, 71, 0,
	0, 0, 0, 0, 80, 81, 82, 83, 84, 85,
	86, 77, 72, 74, 75, 76, 0, 70, 90, 0,
	0, 0, 0, 0, 0, 0, 92, 0, 0, 0,
	0, 0, 78, 0, 0, 89, 0, 0, 87, 0,
	0, 0, 0, 73, 0, 0, 0, 88, 0, 91,
	0, 0, 0, 79, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 71, 513, 0, 0, 0, 0, 80,
	81, 82, 83, 84, 85, 86, 77, 72, 74, 75,
	76, 0, 70, 90, 0, 0, 0, 0, 0, 0,
	0, 92, 0, 78, 0, 0, 0, 0, 0, 87,
	89, 0, 0, 0, 0, 0, 0, 0, 73, 0,
	91, 0, 88, 0, 0, 0, 0, 0, 79, 0,
	0, 0, 0, 0, 71, 510, 0, 0, 0, 0,
	80, 81, 82, 83, 84, 85, 86, 77, 72, 74,
	75, 76, 0, 70, 90, 0, 0, 0, 0, 0,
	0, 0, 92, 0, 0, 0, 0, 0, 78, 0,
	0, 89, 0, 0, 87, 0, 0, 0, 0, 73,
	0, 0, 0, 88, 0, 91, 0, 0, 0, 79,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 71,
	432, 0, 0, 0, 0, 80, 81, 82, 83, 84,
	85, 86, 77, 72, 74, 75, 76, 0, 70, 90,
	0, 0, 404, 0, 0, 0, 0, 92, 0, 78,
	0, 0, 0, 0, 0, 87, 89, 0, 0, 0,
	0, 0, 0, 0, 73, 0, 91, 0, 88, 0,
	0, 0, 0, 0, 79, 0, 0, 0, 0, 0,
	71, 0, 0, 0, 0, 0, 80, 81, 82, 83,
	84, 85, 86, 77, 72, 74, 75, 76, 0, 70,
	90, 0, 0, 0, 0, 0, 0, 0, 92, 0,
	0, 0, 0, 0, 0, 0, 0, 89, 0, 0,
	0, 78, 0, 0, 0, 73, 0, 87, 0, 88,
	0, 91, 0, 0, 0, 79, 0, 0, 0, 0,
	0, 0, 403, 0, 0, 71, 0, 0, 0, 0,
	0, 80, 81, 82, 83, 84, 85, 86, 77, 72,
	74, 75, 76, 0, 70, 247, 0, 0, 319, 0,
	0, 0, 90, 0, 0, 0, 0, 0, 0, 0,
	92, 0, 78, 0, 0, 0, 0, 0, 87, 89,
	0, 0, 91, 0, 0, 0, 0, 73, 0, 0,
	0, 88, 0, 0, 0, 0, 71, 79, 0, 0,
	0, 0, 80, 81, 82, 83, 84, 85, 86, 77,
	72, 74, 75, 76, 0, 70, 246, 0, 0, 0,
	0, 0, 0, 90, 0, 0, 0, 0, 0, 0,
	0, 92, 0, 0, 0, 0, 0, 78, 0, 0,
	89, 0, 0, 87, 0, 0, 0, 0, 73, 0,
	0, 0, 88, 0, 91, 0, 0, 0, 79, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 71, 0,
	0, 0, 0, 0, 80, 81, 82, 83, 84, 85,
	86, 77, 72, 74, 75, 76, 0, 70, 90, 0,
	0, 0, 0, 0, 0, 0, 92, 0, 78, 0,
	0, 0, 0, 0, 87, 89, 0, 0, 0, 0,
	0, 0, 0, 73, 0, 91, 0, 88, 0, 0,
	0, 0, 0, 79, 0, 0, 0, 0, 0, 71,
	0, 0, 0, 0, 0, 80, 81, 82, 83, 84,
	85, 86, 77, 72, 74, 75, 76, 0, 70, 90,
	0, 0, 0, 0, 0, 0, 0, 92, 0, 0,
	0, 0, 0, 0, 0, 0, 89, 0, 0, 0,
	0, 0, 0, 0, 73, 0, 117, 0, 88, 0,
	91, 0, 0, 0, 79, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 71, 0, 0, 0, 0, 0,
	80, 81, 82, 83, 84, 85, 86, 77, 72, 74,
	75, 76, 154, 70, 0, 0, 59, 62, 0, 0,
	0, 0, 0, 0, 0, 0, 49, 0, 0, 0,
	0, 78, 0, 0, 0, 0, 0, 87, 0, 0,
	0, 91, 0, 153, 0, 0, 0, 158, 0, 0,
	61, 0, 0, 0, 10, 71, 39, 63, 0, 0,
	0, 80, 81, 82, 83, 84, 85, 86, 77, 72,
	74, 75, 76, 0, 70, 0, 0, 0, 0, 0,
	0, 0, 90, 0, 0, 0, 0, 0, 0, 0,
	92, 24, 38, 0, 0, 9, 37, 0, 0, 89,
	0, 0, 0, 0, 0, 0, 0, 73, 0, 0,
	0, 88, 0, 0, 0, 157, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	60, 0, 0, 0, 0, 0, 0, 0, 40, 0,
	0, 0, 59, 62, 0, 0, 0, 0, 0, 0,
	0, 0, 49, 0, 0, 0, 0, 0, 0, 0,
	0, 42, 41, 43, 21, 0, 46, 47, 52, 0,
	57, 0, 58, 158, 91, 87, 61, 0, 0, 0,
	10, 0, 39, 63, 0, 0, 0, 159, 71, 0,
	0, 0, 0, 0, 80, 81, 82, 83, 84, 85,
	86, 77, 72, 74, 75, 76, 0, 70, 0, 0,
	0, 0, 0, 0, 0, 87, 0, 24, 38, 0,
	90, 9, 37, 0, 0, 0, 0, 0, 92, 0,
	0, 0, 0, 0, 0, 0, 0, 89, 0, 0,
	0, 157, 59, 62, 0, 73, 0, 0, 0, 88,
	0, 0, 49, 0, 0, 0, 60, 0, 0, 0,
	90, 0, 0, 0, 40, 0, 0, 0, 92, 217,
	0, 0, 0, 0, 0, 0, 61, 89, 0, 0,
	10, 0, 39, 63, 0, 73, 0, 42, 41, 43,
	21, 0, 46, 47, 52, 0, 57, 0, 58, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 91, 159, 0, 0, 0, 24, 38, 0,
	0, 9, 37, 0, 0, 0, 71, 0, 0, 0,
	0, 0, 80, 81, 82, 83, 84, 85, 86, 77,
	72, 74, 75, 76, 0, 70, 0, 0, 0, 0,
	0, 0, 91, 0, 0, 0, 60, 0, 0, 0,
	0, 59, 62, 0, 40, 0, 71, 0, 0, 0,
	0, 49, 0, 0, 0, 83, 84, 85, 86, 77,
	72, 74, 75, 76, 0, 70, 0, 42, 41, 43,
	21, 0, 46, 47, 52, 61, 57, 0, 58, 10,
	0, 39, 63, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 56, 218, 0, 59, 62, 0, 0, 0,
	0, 0, 0, 0, 0, 49, 0, 0, 0, 0,
	0, 0, 0, 54, 0, 0, 24, 38, 31, 0,
	9, 37, 0, 0, 55, 0, 0, 0, 0, 61,
	0, 0, 0, 10, 11, 39, 63, 0, 0, 68,
	0, 0, 0, 59, 62, 0, 0, 0, 0, 0,
	29, 0, 0, 49, 0, 60, 0, 0, 0, 0,
	0, 0, 0, 40, 0, 0, 0, 0, 0, 33,
	24, 38, 0, 0, 9, 37, 0, 61, 0, 0,
	0, 10, 0, 39, 63, 0, 42, 41, 43, 21,
	0, 46, 47, 52, 0, 57, 0, 58, 59, 62,
	0, 0, 0, 0, 0, 0, 69, 0, 49, 60,
	0, 0, 159, 0, 0, 0, 0, 40, 24, 38,
	0, 0, 9, 37, 32, 30, 0, 0, 0, 0,
	0, 0, 61, 0, 0, 0, 10, 0, 39, 63,
	42, 41, 43, 21, 0, 46, 47, 52, 0, 57,
	0, 58, 0, 0, 0, 0, 0, 60, 0, 0,
	0, 0, 0, 0, 0, 40, 0, 0, 0, 0,
	0, 0, 0, 24, 38, 0, 0, 9, 37, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 42, 41,
	43, 21, 0, 46, 47, 52, 0, 57, 0, 58,
	480, 59, 62, 0, 0, 0, 0, 0, 0, 0,
	0, 49, 60, 0, 0, 0, 0, 0, 0, 0,
	40, 0, 59, 62, 0, 0, 0, 0, 0, 0,
	0, 0, 49, 0, 0, 61, 0, 0, 0, 10,
	0, 39, 63, 42, 41, 43, 21, 0, 46, 47,
	52, 0, 57, 0, 58, 363, 61, 0, 0, 0,
	10, 0, 39, 63, 0, 0, 68, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 24, 38, 0, 0,
	9, 37, 0, 0, 59, 62, 0, 0, 0, 0,
	0, 0, 0, 0, 49, 0, 0, 24, 38, 0,
	0, 9, 37, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 60, 0, 0, 61, 0,
	0, 0, 10, 40, 39, 63, 0, 0, 0, 0,
	0, 0, 0, 69, 0, 0, 60, 0, 0, 117,
	0, 0, 0, 0, 40, 0, 42, 41, 43, 21,
	0, 46, 47, 52, 0, 57, 0, 58, 0, 24,
	38, 0, 0, 9, 37, 0, 0, 42, 41, 43,
	21, 0, 46, 47, 52, 0, 57, 0, 58, 59,
	62, 0, 0, 0, 0, 0, 0, 0, 0, 49,
	0, 0, 0, 0, 0, 0, 0, 0, 60, 0,
	0, 0, 0, 0, 0, 0, 40, 0, 0, 0,
	0, 0, 0, 61, 0, 0, 0, 0, 0, 39,
	63, 0, 0, 0, 0, 0, 0, 0, 0, 42,
	41, 43, 21, 0, 46, 47, 52, 0, 57, 0,
	58, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 24, 38, 0, 0, 0, 37,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 60, 0, 0, 0, 0, 0, 0,
	0, 40, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 42, 41, 43, 21, 0, 46,
	47, 52, 0, 57, 0, 58,
}
var yyPact = []int{

	2267, -1000, -1000, 1751, -1000, -1000, -1000, -1000, -1000, 2556,
	2556, 818, -1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000,
	-1000, -1000, -1000, -1000, 2556, -1000, -1000, -1000, 263, 344,
	342, 405, 25, 330, -1000, -1000, -1000, -1000, -1000, -1000,
	-1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000, -1, 2473,
	-1000, -1000, 2494, -1000, 248, 230, 380, 27, 2556, 7,
	7, 7, 2556, 2556, -1000, -1000, 305, 397, 54, 1868,
	14, 2556, 2556, 2556, 2556, 2556, 2556, 2556, 2556, 2556,
	2556, 2556, 2556, 2556, 2556, 2556, 2556, 2651, 201, 2556,
	2556, 2556, 215, 2022, 100, -1000, -61, 301, 438, 429,
	324, -1000, 462, 25, 25, 25, 78, -69, 162, -1000,
	25, 2094, 428, -1000, -1000, 1690, 161, 2556, -17, 1751,
	-1000, 374, 17, 372, 25, 25, -65, -31, -1000, -71,
	-98, -37, 1751, -21, -1000, 142, -1000, -21, -21, 1625,
	1564, 41, -1000, 11, 305, -1000, 365, -1000, -129, -73,
	-75, -1000, -63, 1994, 2213, 2556, -1000, -1000, -1000, -1000,
	981, -1000, -1000, 2556, 927, -64, -64, -61, -61, -61,
	484, 2022, 1884, 2062, 2062, 2062, 48, 48, 48, 48,
	473, -1000, 2651, 2556, 2556, 2556, 136, 100, 100, -1000,
	152, -1000, -1000, 231, -1000, 2556, -1000, 210, -1000, 210,
	-1000, 210, 2556, 326, 326, 78, 112, -1000, 169, 24,
	-1000, -1000, -1000, 11, -1000, 75, -20, 2556, -26, -1000,
	161, 2556, -1000, 2556, 1492, -1000, 258, 256, -1000, 254,
	-126, -1000, -97, -127, -1000, 27, 2556, -1000, 2556, 427,
	7, 2556, 2556, 2556, 425, 420, 7, 7, 373, -1000,
	2556, -66, -1000, -112, 41, 358, -1000, 203, 162, -16,
	24, 24, 2213, -63, 2556, -63, 615, -32, -1000, 803,
	-1000, 2370, 2651, 5, 2556, 2651, 2651, 2651, 2651, 2651,
	2651, 323, 136, 100, 100, -1000, -1000, -1000, -1000, -1000,
	2556, 1751, -1000, -1000, -1000, -38, -1000, 771, 178, -1000,
	2556, 178, 41, 57, 41, -16, -16, 321, -1000, 162,
	-1000, -1000, 49, -1000, 1431, -1000, -1000, 1366, 1751, 2556,
	25, 25, 25, 17, 24, 17, -1000, 1751, 1751, -1000,
	-1000, 1751, 1751, 1751, -1000, -1000, -12, -12, 147, -1000,
	456, 1751, 11, 2556, 373, 52, 52, 2556, -1000, -1000,
	-1000, -1000, 78, -94, -1000, -129, -129, -1000, 615, -1000,
	-1000, -1000, -1000, -1000, 1305, 335, -1000, -1000, 2556, 739,
	-70, -70, -62, -62, -62, 77, 2651, 1751, 2556, -1000,
	-1000, -1000, -1000, 158, 158, 2556, 1751, 158, 158, 301,
	41, 301, 301, -39, -1000, -44, -40, -1000, 4, 2556,
	-1000, 245, 210, -1000, 2556, 1751, 72, -3, -1000, -1000,
	-1000, 163, 419, 2556, 418, -1000, 2556, -1000, 1751, -1000,
	-1000, -129, -99, -100, -1000, 615, -1000, -19, 2556, 162,
	162, -1000, -1000, 583, -1000, 2315, 335, -1000, -1000, -1000,
	1994, -1000, 1751, -1000, -1000, 158, 301, 158, 158, -16,
	2556, -16, -1000, -1000, 7, 1751, 326, -56, 1751, -1000,
	118, 2556, -1000, 93, -1000, 1751, -1000, -9, 162, 24,
	24, -1000, -1000, -1000, 2556, 1240, 78, 78, -1000, -1000,
	-1000, 1179, -1000, -63, 2556, -1000, 158, -1000, -1000, -1000,
	1114, -1000, -13, -1000, 139, 61, 162, -1000, -1000, -101,
	-1000, 1751, 17, 391, -1000, 11, 233, -129, -129, 551,
	-1000, -1000, -1000, -1000, 1751, -1000, -1000, 417, 7, -16,
	-16, 301, 311, 242, 207, 2556, -1000, -1000, -1000, 2556,
	-66, -1000, 169, 162, 162, -1000, -1000, -1000, -1000, -1000,
	-94, -1000, 158, 123, 293, 326, 42, 455, -1000, 1751,
	312, 233, 233, -1000, 222, 121, 61, 72, 2556, 2556,
	2556, -1000, -1000, 112, 41, 368, 301, -1000, -1000, 1751,
	1751, 60, 57, 41, 53, -1000, 2556, 158, -1000, 288,
	-1000, 41, -1000, -1000, 276, -1000, 1053, -1000, 120, 287,
	-1000, 285, -1000, 439, 108, 107, 41, 367, 352, 53,
	2556, 2556, -1000, -1000, -1000,
}
var yyPgo = []int{

	0, 669, 668, 661, 660, 659, 51, 657, 655, 0,
	62, 143, 37, 323, 45, 22, 21, 27, 15, 19,
	647, 646, 645, 643, 48, 296, 642, 640, 639, 47,
	46, 223, 28, 636, 635, 633, 630, 35, 629, 52,
	627, 624, 623, 426, 621, 43, 40, 620, 24, 26,
	285, 142, 619, 32, 13, 215, 618, 6, 616, 39,
	615, 614, 611, 610, 599, 42, 36, 597, 41, 595,
	594, 34, 592, 591, 9, 589, 587, 586, 585, 486,
	584, 583, 575, 569, 568, 563, 555, 554, 553, 552,
	546, 543, 539, 578, 44, 16, 537, 536, 533, 4,
	23, 529, 20, 7, 33, 525, 8, 30, 523, 519,
	31, 17, 517, 511, 3, 2, 5, 29, 49, 508,
	12, 507, 14, 505, 497, 496, 38, 495, 18, 492,
}
var yyR1 = []int{

	0, 125, 125, 79, 79, 79, 79, 80, 81, 82,
	82, 82, 82, 82, 83, 89, 89, 89, 37, 38,
	38, 38, 38, 38, 38, 38, 39, 39, 41, 40,
	68, 67, 67, 67, 67, 67, 126, 126, 66, 66,
	65, 65, 65, 18, 18, 17, 17, 16, 44, 44,
	43, 42, 42, 42, 42, 127, 127, 45, 45, 45,
	46, 46, 46, 50, 51, 49, 49, 53, 53, 52,
	128, 128, 47, 47, 47, 129, 129, 54, 55, 55,
	56, 15, 15, 14, 57, 57, 58, 59, 59, 60,
	12, 12, 61, 61, 62, 63, 63, 64, 70, 70,
	69, 72, 72, 71, 78, 78, 77, 77, 74, 74,
	73, 76, 76, 75, 84, 84, 93, 93, 96, 96,
	95, 94, 99, 99, 98, 97, 97, 85, 85, 86,
	87, 87, 87, 103, 105, 105, 104, 110, 110, 109,
	101, 101, 100, 100, 19, 102, 32, 32, 106, 108,
	108, 107, 88, 88, 111, 111, 111, 111, 112, 112,
	112, 116, 116, 113, 113, 113, 114, 115, 90, 90,
	117, 118, 118, 119, 119, 120, 120, 120, 124, 124,
	122, 123, 123, 91, 91, 92, 121, 121, 48, 48,
	48, 48, 48, 48, 9, 9, 9, 9, 9, 9,
	9, 9, 9, 9, 9, 9, 9, 9, 9, 9,
	9, 9, 9, 9, 9, 9, 9, 9, 9, 9,
	9, 9, 9, 9, 9, 9, 9, 9, 9, 9,
	9, 9, 9, 10, 10, 10, 10, 10, 10, 10,
	10, 10, 11, 11, 11, 11, 11, 11, 11, 11,
	11, 11, 11, 11, 11, 11, 1, 1, 1, 1,
	1, 1, 1, 2, 2, 3, 8, 8, 7, 7,
	6, 4, 13, 13, 5, 5, 20, 21, 21, 22,
	25, 25, 23, 24, 24, 33, 33, 33, 34, 26,
	26, 27, 27, 27, 30, 30, 29, 29, 31, 28,
	28, 35, 36, 36,
}
var yyR2 = []int{

	0, 1, 1, 1, 1, 1, 1, 2, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1, 4, 1,
	3, 4, 3, 4, 3, 4, 1, 1, 5, 5,
	2, 1, 2, 2, 3, 4, 1, 1, 1, 3,
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
	6, 0, 6, 2, 3, 2, 1, 2, 6, 11,
	1, 1, 3, 0, 3, 0, 2, 2, 1, 3,
	1, 0, 2, 5, 5, 6, 0, 3, 1, 3,
	3, 5, 5, 4, 1, 3, 3, 5, 5, 4,
	5, 6, 3, 3, 3, 3, 3, 3, 3, 3,
	2, 3, 3, 3, 3, 3, 3, 3, 5, 6,
	3, 4, 3, 4, 3, 4, 3, 4, 3, 4,
	3, 4, 2, 1, 1, 1, 1, 1, 2, 1,
	1, 1, 1, 3, 3, 5, 5, 4, 5, 6,
	3, 3, 3, 3, 3, 3, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 3, 0, 1, 1, 3,
	3, 3, 0, 1, 1, 1, 3, 1, 1, 3,
	4, 5, 2, 0, 2, 4, 5, 4, 1, 1,
	1, 4, 4, 4, 1, 3, 3, 3, 2, 6,
	6, 3, 1, 1,
}
var yyChk = []int{

	-1000, -125, -79, -9, -80, -81, -82, -83, -10, 87,
	46, 47, -37, -84, -85, -86, -87, -88, -89, -1,
	-2, 156, -5, -33, 83, -20, -26, -35, -38, 63,
	138, 31, 137, 82, -90, -91, -92, 88, 84, 48,
	130, 154, 153, 155, -3, -4, 158, 159, -34, 18,
	-27, -28, 160, -39, 26, 37, 5, 162, 164, 8,
	122, 42, 9, 49, -41, -40, -43, -68, 52, 119,
	183, 164, 178, 83, 179, 180, 181, 177, 7, 93,
	170, 171, 172, 173, 174, 175, 176, 13, 87, 75,
	58, 150, 66, -9, -9, -79, -9, -70, 133, 64,
	43, -69, 94, 65, 65, 52, -93, -50, -51, 156,
	65, 160, -21, -22, -23, -9, -25, 146, -36, -9,
	-37, 102, 60, 102, 60, 60, -8, -7, -6, 155,
	-13, -12, -9, -30, -29, -19, 156, -30, -30, -9,
	-9, -55, -56, 73, -44, -43, -42, -45, -51, -50,
	125, -67, -66, 35, 4, -126, -65, 107, 39, 179,
	-9, 156, 157, 164, -9, -9, -9, -9, -9, -9,
	-9, -9, -9, -9, -9, -9, -9, -9, -9, -9,
	-11, -10, 13, 75, 58, 150, -9, -9, -9, 88,
	87, 84, 143, -74, -73, 76, -39, 4, -39, 4,
	-39, 4, 16, -93, -93, -93, -53, -52, 139, 168,
	-18, -17, -16, 10, 156, -93, -13, 35, 179, 41,
	-25, 146, -24, 40, -9, 161, 60, -117, 156, 60,
	-118, -51, -50, -118, 163, 167, 168, 165, 167, -31,
	167, 117, 58, 150, -31, -31, 51, 51, -57, -58,
	147, -15, -14, -16, -55, -47, 62, 72, -49, 183,
	168, 168, 167, -66, -126, -66, -9, 183, -18, -9,
	165, 168, 7, 183, 164, 178, 83, 179, 180, 181,
	177, -11, -9, -9, -9, 88, 84, 143, -76, -75,
	90, -9, -39, -39, -39, -72, -71, -9, -96, -95,
	68, -95, -53, -103, -106, 120, 136, -128, 102, -51,
	156, -16, 141, 161, -9, 161, -24, -9, -9, 126,
	91, 91, 91, 183, 168, 183, -6, -9, -9, 41,
	-29, -9, -9, -9, 41, 41, -30, -30, -59, -60,
	55, -9, 167, 170, -57, 67, 86, -127, 135, 50,
	-129, 95, -18, -48, 156, -51, -51, -65, -9, -18,
	179, 165, 166, 165, -9, -11, 156, 157, 164, -9,
	-11, -11, -11, -11, -11, -11, 7, -9, 167, -78,
	-77, 11, 33, -94, -37, 144, -9, -94, -37, -57,
	-106, -57, -57, -105, -104, -48, -108, -107, -48, 69,
	-18, -45, 160, 161, 126, -9, -118, -118, -118, -117,
	-51, -117, -32, 146, -32, -68, 16, -14, -9, -59,
	-46, -51, -50, 125, -46, -9, -53, 183, 164, -49,
	-49, -18, 165, -9, 165, 168, -11, -71, -99, -98,
	112, -99, -9, -99, -99, -74, -57, -74, -74, 167,
	170, 167, -110, -109, 51, -9, 91, -37, -9, -120,
	141, 160, -121, 110, 41, -9, 41, -12, -49, 168,
	168, -18, 156, 157, 164, -9, -18, -18, 165, 166,
	165, -9, -97, -66, -126, -99, -74, -99, -99, -104,
	-9, -107, -101, -100, -19, -95, 161, 145, 77, -124,
	-122, -9, 127, -61, -62, 74, -18, -51, -51, -9,
	165, -53, -53, 165, -9, -99, -110, -32, 167, 58,
	150, -111, 146, -17, 161, 167, -117, -63, -64, 56,
	-15, -54, 91, -49, -49, 165, 166, 41, -100, -102,
	-48, -102, -74, 80, 87, 91, -119, 97, -122, -9,
	-128, -18, -18, -99, 126, 80, -95, -123, 147, 16,
	69, -54, -54, 137, 31, 126, -111, -120, -122, -9,
	-9, -113, -103, -106, -114, -57, 63, -74, -112, 146,
	-57, -106, -57, -116, 146, -115, -9, -99, 80, 87,
	-57, 87, -57, 126, 80, 80, 31, 126, 126, -114,
	63, 63, -116, -115, -115,
}
var yyDef = []int{

	0, -2, 1, 2, 3, 4, 5, 6, 194, 0,
	0, 0, 8, 9, 10, 11, 12, 13, 14, 233,
	234, -2, 236, 237, 0, 239, 240, 241, 98, 0,
	0, 0, 0, 0, 15, 16, 17, 256, 257, 258,
	259, 260, 261, 262, 263, 264, 274, 275, 0, 0,
	289, 290, 0, 19, 0, 0, 0, 266, 272, 0,
	0, 0, 0, 0, 26, 27, 78, 48, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 210, 232, 7, 238, 108, 0, 0,
	0, 99, 0, 0, 0, 0, 67, 0, 43, -2,
	0, 272, 0, 277, 278, 0, 283, 0, 0, 302,
	303, 0, 0, 0, 0, 0, 0, 267, 268, 0,
	0, 273, 90, 0, 294, 0, 144, 0, 0, 0,
	0, 84, 79, 0, 78, 49, -2, 51, 65, 0,
	0, 30, 31, 0, 0, 0, 38, 36, 37, 40,
	43, 195, 196, 0, 0, 202, 203, 204, 205, 206,
	207, 208, 209, -2, -2, -2, -2, -2, -2, -2,
	0, 242, 0, 0, 0, 0, -2, -2, -2, 226,
	0, 228, 230, 111, 109, 0, 20, 0, 22, 0,
	24, 0, 0, 118, 0, 67, 0, 68, 70, 0,
	117, 44, 45, 0, 47, 0, 0, 0, 0, 276,
	283, 0, 282, 0, 0, 301, 0, 0, 170, 0,
	0, 171, 0, 0, 265, 0, 0, 271, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 87, 85,
	0, 80, 81, 0, 84, 0, 73, 75, 43, 0,
	0, 0, 0, 32, 0, 33, 43, 0, 42, 0,
	199, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, -2, -2, -2, 227, 229, 231, 18, 112,
	0, 110, 21, 23, 25, 100, 101, 104, 0, 119,
	0, 0, 84, 84, 84, 0, 0, 0, 71, 43,
	64, 46, 0, 285, 0, 287, 279, 0, 284, 0,
	0, 0, 0, 0, 0, 0, 269, 270, 91, 291,
	295, 298, 296, 297, 292, 293, 146, 146, 0, 88,
	0, 86, 0, 0, 87, 0, 0, 0, 55, 56,
	74, 76, 67, 66, 188, 65, 65, 39, 43, 34,
	41, 197, 198, 200, 0, 218, 243, 244, 0, 0,
	250, 251, 252, 253, 254, 255, 0, 113, 0, 103,
	105, 106, 107, 122, 122, 0, 120, 122, 122, 108,
	84, 108, 108, 133, 134, 0, 148, 149, 137, 0,
	116, 0, 0, 286, 0, 280, 175, 0, 183, 184,
	172, 186, 0, 0, 0, 28, 0, 82, 83, 29,
	52, 65, 0, 0, 53, 43, 57, 0, 0, 43,
	43, 35, 201, 0, 247, 0, 219, 102, 114, 123,
	0, 115, 121, 127, 128, 122, 108, 122, 122, 0,
	0, 0, 151, 138, 0, 69, 0, 0, 281, 168,
	0, 0, 185, 0, 299, 147, 300, 92, 43, 0,
	0, 54, 189, 190, 0, 0, 67, 67, 245, 246,
	248, 0, 124, 125, 0, 129, 122, 131, 132, 135,
	137, 150, 146, 140, 0, 154, 0, 176, 177, 0,
	178, 180, 0, 95, 93, 0, 0, 65, 65, 0,
	193, 58, 59, 249, 126, 130, 136, 0, 0, 0,
	0, 108, 0, 0, 173, 0, 187, 89, 96, 0,
	94, 60, 70, 43, 43, 191, 192, 139, 141, 142,
	145, 143, 122, 0, 0, 0, 181, 0, 179, 97,
	0, 0, 0, 152, 0, 0, 154, 175, 0, 0,
	0, 61, 62, 0, 84, 0, 108, 169, 182, 174,
	77, 158, 84, 84, 161, 166, 0, 122, 155, 0,
	163, 84, 165, 156, 0, 157, 84, 153, 0, 0,
	164, 0, 167, 0, 0, 0, 84, 0, 0, 161,
	0, 0, 159, 160, 162,
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
	182, 183,
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
		//line n1ql.y:344
		{
			yylex.(*lexer).setStatement(yyS[yypt-0].statement)
		}
	case 2:
		//line n1ql.y:349
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
		//line n1ql.y:366
		{
			yyVAL.statement = algebra.NewExplain(yyS[yypt-0].statement)
		}
	case 8:
		//line n1ql.y:373
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
		//line n1ql.y:404
		{
			yyVAL.fullselect = algebra.NewSelect(yyS[yypt-3].subresult, yyS[yypt-2].order, yyS[yypt-0].expr, yyS[yypt-1].expr) /* OFFSET precedes LIMIT */
		}
	case 19:
		//line n1ql.y:410
		{
			yyVAL.subresult = yyS[yypt-0].subselect
		}
	case 20:
		//line n1ql.y:415
		{
			yyVAL.subresult = algebra.NewUnion(yyS[yypt-2].subresult, yyS[yypt-0].subselect)
		}
	case 21:
		//line n1ql.y:420
		{
			yyVAL.subresult = algebra.NewUnionAll(yyS[yypt-3].subresult, yyS[yypt-0].subselect)
		}
	case 22:
		//line n1ql.y:425
		{
			yyVAL.subresult = algebra.NewIntersect(yyS[yypt-2].subresult, yyS[yypt-0].subselect)
		}
	case 23:
		//line n1ql.y:430
		{
			yyVAL.subresult = algebra.NewIntersectAll(yyS[yypt-3].subresult, yyS[yypt-0].subselect)
		}
	case 24:
		//line n1ql.y:435
		{
			yyVAL.subresult = algebra.NewExcept(yyS[yypt-2].subresult, yyS[yypt-0].subselect)
		}
	case 25:
		//line n1ql.y:440
		{
			yyVAL.subresult = algebra.NewExceptAll(yyS[yypt-3].subresult, yyS[yypt-0].subselect)
		}
	case 26:
		yyVAL.subselect = yyS[yypt-0].subselect
	case 27:
		yyVAL.subselect = yyS[yypt-0].subselect
	case 28:
		//line n1ql.y:453
		{
			yyVAL.subselect = algebra.NewSubselect(yyS[yypt-4].fromTerm, yyS[yypt-3].bindings, yyS[yypt-2].expr, yyS[yypt-1].group, yyS[yypt-0].projection)
		}
	case 29:
		//line n1ql.y:460
		{
			yyVAL.subselect = algebra.NewSubselect(yyS[yypt-3].fromTerm, yyS[yypt-2].bindings, yyS[yypt-1].expr, yyS[yypt-0].group, yyS[yypt-4].projection)
		}
	case 30:
		//line n1ql.y:475
		{
			yyVAL.projection = yyS[yypt-0].projection
		}
	case 31:
		//line n1ql.y:482
		{
			yyVAL.projection = algebra.NewProjection(false, yyS[yypt-0].resultTerms)
		}
	case 32:
		//line n1ql.y:487
		{
			yyVAL.projection = algebra.NewProjection(true, yyS[yypt-0].resultTerms)
		}
	case 33:
		//line n1ql.y:492
		{
			yyVAL.projection = algebra.NewProjection(false, yyS[yypt-0].resultTerms)
		}
	case 34:
		//line n1ql.y:497
		{
			yyVAL.projection = algebra.NewRawProjection(false, yyS[yypt-1].expr, yyS[yypt-0].s)
		}
	case 35:
		//line n1ql.y:502
		{
			yyVAL.projection = algebra.NewRawProjection(true, yyS[yypt-1].expr, yyS[yypt-0].s)
		}
	case 38:
		//line n1ql.y:515
		{
			yyVAL.resultTerms = algebra.ResultTerms{yyS[yypt-0].resultTerm}
		}
	case 39:
		//line n1ql.y:520
		{
			yyVAL.resultTerms = append(yyS[yypt-2].resultTerms, yyS[yypt-0].resultTerm)
		}
	case 40:
		//line n1ql.y:527
		{
			yyVAL.resultTerm = algebra.NewResultTerm(nil, true, "")
		}
	case 41:
		//line n1ql.y:532
		{
			yyVAL.resultTerm = algebra.NewResultTerm(yyS[yypt-2].expr, true, "")
		}
	case 42:
		//line n1ql.y:537
		{
			yyVAL.resultTerm = algebra.NewResultTerm(yyS[yypt-1].expr, false, yyS[yypt-0].s)
		}
	case 43:
		//line n1ql.y:544
		{
			yyVAL.s = ""
		}
	case 44:
		yyVAL.s = yyS[yypt-0].s
	case 45:
		yyVAL.s = yyS[yypt-0].s
	case 46:
		//line n1ql.y:555
		{
			yyVAL.s = yyS[yypt-0].s
		}
	case 47:
		yyVAL.s = yyS[yypt-0].s
	case 48:
		//line n1ql.y:573
		{
			yyVAL.fromTerm = nil
		}
	case 49:
		yyVAL.fromTerm = yyS[yypt-0].fromTerm
	case 50:
		//line n1ql.y:582
		{
			yyVAL.fromTerm = yyS[yypt-0].fromTerm
		}
	case 51:
		//line n1ql.y:589
		{
			yyVAL.fromTerm = yyS[yypt-0].keyspaceTerm
		}
	case 52:
		//line n1ql.y:594
		{
			yyVAL.fromTerm = algebra.NewJoin(yyS[yypt-3].fromTerm, yyS[yypt-2].b, yyS[yypt-0].keyspaceTerm)
		}
	case 53:
		//line n1ql.y:599
		{
			yyVAL.fromTerm = algebra.NewNest(yyS[yypt-3].fromTerm, yyS[yypt-2].b, yyS[yypt-0].keyspaceTerm)
		}
	case 54:
		//line n1ql.y:604
		{
			yyVAL.fromTerm = algebra.NewUnnest(yyS[yypt-4].fromTerm, yyS[yypt-3].b, yyS[yypt-1].expr, yyS[yypt-0].s)
		}
	case 57:
		//line n1ql.y:617
		{
			yyVAL.keyspaceTerm = algebra.NewKeyspaceTerm("", yyS[yypt-3].s, yyS[yypt-2].path, yyS[yypt-1].s, yyS[yypt-0].expr)
		}
	case 58:
		//line n1ql.y:622
		{
			yyVAL.keyspaceTerm = algebra.NewKeyspaceTerm(yyS[yypt-5].s, yyS[yypt-3].s, yyS[yypt-2].path, yyS[yypt-1].s, yyS[yypt-0].expr)
		}
	case 59:
		//line n1ql.y:627
		{
			yyVAL.keyspaceTerm = algebra.NewKeyspaceTerm("#system", yyS[yypt-3].s, yyS[yypt-2].path, yyS[yypt-1].s, yyS[yypt-0].expr)
		}
	case 60:
		//line n1ql.y:634
		{
			yyVAL.keyspaceTerm = algebra.NewKeyspaceTerm("", yyS[yypt-3].s, yyS[yypt-2].path, yyS[yypt-1].s, yyS[yypt-0].expr)
		}
	case 61:
		//line n1ql.y:639
		{
			yyVAL.keyspaceTerm = algebra.NewKeyspaceTerm(yyS[yypt-5].s, yyS[yypt-3].s, yyS[yypt-2].path, yyS[yypt-1].s, yyS[yypt-0].expr)
		}
	case 62:
		//line n1ql.y:644
		{
			yyVAL.keyspaceTerm = algebra.NewKeyspaceTerm("#system", yyS[yypt-3].s, yyS[yypt-2].path, yyS[yypt-1].s, yyS[yypt-0].expr)
		}
	case 63:
		yyVAL.s = yyS[yypt-0].s
	case 64:
		yyVAL.s = yyS[yypt-0].s
	case 65:
		//line n1ql.y:659
		{
			yyVAL.path = nil
		}
	case 66:
		//line n1ql.y:664
		{
			yyVAL.path = yyS[yypt-0].path
		}
	case 67:
		//line n1ql.y:671
		{
			yyVAL.expr = nil
		}
	case 68:
		yyVAL.expr = yyS[yypt-0].expr
	case 69:
		//line n1ql.y:680
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 70:
		//line n1ql.y:687
		{
		}
	case 72:
		//line n1ql.y:695
		{
			yyVAL.b = false
		}
	case 73:
		//line n1ql.y:700
		{
			yyVAL.b = false
		}
	case 74:
		//line n1ql.y:705
		{
			yyVAL.b = true
		}
	case 77:
		//line n1ql.y:718
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 78:
		//line n1ql.y:732
		{
			yyVAL.bindings = nil
		}
	case 79:
		yyVAL.bindings = yyS[yypt-0].bindings
	case 80:
		//line n1ql.y:741
		{
			yyVAL.bindings = yyS[yypt-0].bindings
		}
	case 81:
		//line n1ql.y:748
		{
			yyVAL.bindings = expression.Bindings{yyS[yypt-0].binding}
		}
	case 82:
		//line n1ql.y:753
		{
			yyVAL.bindings = append(yyS[yypt-2].bindings, yyS[yypt-0].binding)
		}
	case 83:
		//line n1ql.y:760
		{
			yyVAL.binding = expression.NewBinding(yyS[yypt-2].s, yyS[yypt-0].expr)
		}
	case 84:
		//line n1ql.y:774
		{
			yyVAL.expr = nil
		}
	case 85:
		yyVAL.expr = yyS[yypt-0].expr
	case 86:
		//line n1ql.y:783
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 87:
		//line n1ql.y:797
		{
			yyVAL.group = nil
		}
	case 88:
		yyVAL.group = yyS[yypt-0].group
	case 89:
		//line n1ql.y:806
		{
			yyVAL.group = algebra.NewGroup(yyS[yypt-2].exprs, yyS[yypt-1].bindings, yyS[yypt-0].expr)
		}
	case 90:
		//line n1ql.y:813
		{
			yyVAL.exprs = expression.Expressions{yyS[yypt-0].expr}
		}
	case 91:
		//line n1ql.y:818
		{
			yyVAL.exprs = append(yyS[yypt-2].exprs, yyS[yypt-0].expr)
		}
	case 92:
		//line n1ql.y:825
		{
			yyVAL.bindings = nil
		}
	case 93:
		yyVAL.bindings = yyS[yypt-0].bindings
	case 94:
		//line n1ql.y:834
		{
			yyVAL.bindings = yyS[yypt-0].bindings
		}
	case 95:
		//line n1ql.y:841
		{
			yyVAL.expr = nil
		}
	case 96:
		yyVAL.expr = yyS[yypt-0].expr
	case 97:
		//line n1ql.y:850
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 98:
		//line n1ql.y:864
		{
			yyVAL.order = nil
		}
	case 99:
		yyVAL.order = yyS[yypt-0].order
	case 100:
		//line n1ql.y:873
		{
			yyVAL.order = algebra.NewOrder(yyS[yypt-0].sortTerms)
		}
	case 101:
		//line n1ql.y:880
		{
			yyVAL.sortTerms = algebra.SortTerms{yyS[yypt-0].sortTerm}
		}
	case 102:
		//line n1ql.y:885
		{
			yyVAL.sortTerms = append(yyS[yypt-2].sortTerms, yyS[yypt-0].sortTerm)
		}
	case 103:
		//line n1ql.y:892
		{
			yyVAL.sortTerm = algebra.NewSortTerm(yyS[yypt-1].expr, yyS[yypt-0].b)
		}
	case 104:
		//line n1ql.y:899
		{
			yyVAL.b = false
		}
	case 105:
		yyVAL.b = yyS[yypt-0].b
	case 106:
		//line n1ql.y:908
		{
			yyVAL.b = false
		}
	case 107:
		//line n1ql.y:913
		{
			yyVAL.b = true
		}
	case 108:
		//line n1ql.y:927
		{
			yyVAL.expr = nil
		}
	case 109:
		yyVAL.expr = yyS[yypt-0].expr
	case 110:
		//line n1ql.y:936
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 111:
		//line n1ql.y:950
		{
			yyVAL.expr = nil
		}
	case 112:
		yyVAL.expr = yyS[yypt-0].expr
	case 113:
		//line n1ql.y:959
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 114:
		//line n1ql.y:973
		{
			yyVAL.statement = algebra.NewInsertValues(yyS[yypt-3].keyspaceRef, yyS[yypt-2].expr, yyS[yypt-1].expr, yyS[yypt-0].projection)
		}
	case 115:
		//line n1ql.y:978
		{
			yyVAL.statement = algebra.NewInsertSelect(yyS[yypt-3].keyspaceRef, yyS[yypt-2].expr, yyS[yypt-1].fullselect, yyS[yypt-0].projection)
		}
	case 116:
		//line n1ql.y:985
		{
			yyVAL.keyspaceRef = algebra.NewKeyspaceRef(yyS[yypt-3].s, yyS[yypt-1].s, yyS[yypt-0].s)
		}
	case 117:
		//line n1ql.y:990
		{
			yyVAL.keyspaceRef = algebra.NewKeyspaceRef("", yyS[yypt-1].s, yyS[yypt-0].s)
		}
	case 118:
		//line n1ql.y:997
		{
			yyVAL.expr = nil
		}
	case 119:
		yyVAL.expr = yyS[yypt-0].expr
	case 120:
		//line n1ql.y:1006
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 121:
		//line n1ql.y:1013
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 122:
		//line n1ql.y:1020
		{
			yyVAL.projection = nil
		}
	case 123:
		yyVAL.projection = yyS[yypt-0].projection
	case 124:
		//line n1ql.y:1029
		{
			yyVAL.projection = yyS[yypt-0].projection
		}
	case 125:
		//line n1ql.y:1036
		{
			yyVAL.projection = algebra.NewProjection(false, yyS[yypt-0].resultTerms)
		}
	case 126:
		//line n1ql.y:1041
		{
			yyVAL.projection = algebra.NewRawProjection(false, yyS[yypt-0].expr, "")
		}
	case 127:
		//line n1ql.y:1055
		{
			yyVAL.statement = algebra.NewUpsertValues(yyS[yypt-3].keyspaceRef, yyS[yypt-2].expr, yyS[yypt-1].expr, yyS[yypt-0].projection)
		}
	case 128:
		//line n1ql.y:1060
		{
			yyVAL.statement = algebra.NewUpsertSelect(yyS[yypt-3].keyspaceRef, yyS[yypt-2].expr, yyS[yypt-1].fullselect, yyS[yypt-0].projection)
		}
	case 129:
		//line n1ql.y:1074
		{
			yyVAL.statement = algebra.NewDelete(yyS[yypt-4].keyspaceRef, yyS[yypt-3].expr, yyS[yypt-2].expr, yyS[yypt-1].expr, yyS[yypt-0].projection)
		}
	case 130:
		//line n1ql.y:1088
		{
			yyVAL.statement = algebra.NewUpdate(yyS[yypt-6].keyspaceRef, yyS[yypt-5].expr, yyS[yypt-4].set, yyS[yypt-3].unset, yyS[yypt-2].expr, yyS[yypt-1].expr, yyS[yypt-0].projection)
		}
	case 131:
		//line n1ql.y:1093
		{
			yyVAL.statement = algebra.NewUpdate(yyS[yypt-5].keyspaceRef, yyS[yypt-4].expr, yyS[yypt-3].set, nil, yyS[yypt-2].expr, yyS[yypt-1].expr, yyS[yypt-0].projection)
		}
	case 132:
		//line n1ql.y:1098
		{
			yyVAL.statement = algebra.NewUpdate(yyS[yypt-5].keyspaceRef, yyS[yypt-4].expr, nil, yyS[yypt-3].unset, yyS[yypt-2].expr, yyS[yypt-1].expr, yyS[yypt-0].projection)
		}
	case 133:
		//line n1ql.y:1105
		{
			yyVAL.set = algebra.NewSet(yyS[yypt-0].setTerms)
		}
	case 134:
		//line n1ql.y:1112
		{
			yyVAL.setTerms = algebra.SetTerms{yyS[yypt-0].setTerm}
		}
	case 135:
		//line n1ql.y:1117
		{
			yyVAL.setTerms = append(yyS[yypt-2].setTerms, yyS[yypt-0].setTerm)
		}
	case 136:
		//line n1ql.y:1124
		{
			yyVAL.setTerm = algebra.NewSetTerm(yyS[yypt-3].path, yyS[yypt-1].expr, yyS[yypt-0].updateFor)
		}
	case 137:
		//line n1ql.y:1131
		{
			yyVAL.updateFor = nil
		}
	case 138:
		yyVAL.updateFor = yyS[yypt-0].updateFor
	case 139:
		//line n1ql.y:1140
		{
			yyVAL.updateFor = algebra.NewUpdateFor(yyS[yypt-2].bindings, yyS[yypt-1].expr)
		}
	case 140:
		//line n1ql.y:1147
		{
			yyVAL.bindings = expression.Bindings{yyS[yypt-0].binding}
		}
	case 141:
		//line n1ql.y:1152
		{
			yyVAL.bindings = append(yyS[yypt-2].bindings, yyS[yypt-0].binding)
		}
	case 142:
		//line n1ql.y:1159
		{
			yyVAL.binding = expression.NewBinding(yyS[yypt-2].s, yyS[yypt-0].expr)
		}
	case 143:
		//line n1ql.y:1164
		{
			yyVAL.binding = expression.NewDescendantBinding(yyS[yypt-2].s, yyS[yypt-0].expr)
		}
	case 144:
		yyVAL.s = yyS[yypt-0].s
	case 145:
		//line n1ql.y:1175
		{
			yyVAL.expr = yyS[yypt-0].path
		}
	case 146:
		//line n1ql.y:1182
		{
			yyVAL.expr = nil
		}
	case 147:
		//line n1ql.y:1187
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 148:
		//line n1ql.y:1194
		{
			yyVAL.unset = algebra.NewUnset(yyS[yypt-0].unsetTerms)
		}
	case 149:
		//line n1ql.y:1201
		{
			yyVAL.unsetTerms = algebra.UnsetTerms{yyS[yypt-0].unsetTerm}
		}
	case 150:
		//line n1ql.y:1206
		{
			yyVAL.unsetTerms = append(yyS[yypt-2].unsetTerms, yyS[yypt-0].unsetTerm)
		}
	case 151:
		//line n1ql.y:1213
		{
			yyVAL.unsetTerm = algebra.NewUnsetTerm(yyS[yypt-1].path, yyS[yypt-0].updateFor)
		}
	case 152:
		//line n1ql.y:1227
		{
			source := algebra.NewMergeSourceFrom(yyS[yypt-5].keyspaceTerm, "")
			yyVAL.statement = algebra.NewMerge(yyS[yypt-7].keyspaceRef, source, yyS[yypt-3].expr, yyS[yypt-2].mergeActions, yyS[yypt-1].expr, yyS[yypt-0].projection)
		}
	case 153:
		//line n1ql.y:1233
		{
			source := algebra.NewMergeSourceSelect(yyS[yypt-7].fullselect, yyS[yypt-5].s)
			yyVAL.statement = algebra.NewMerge(yyS[yypt-10].keyspaceRef, source, yyS[yypt-3].expr, yyS[yypt-2].mergeActions, yyS[yypt-1].expr, yyS[yypt-0].projection)
		}
	case 154:
		//line n1ql.y:1241
		{
			yyVAL.mergeActions = algebra.NewMergeActions(nil, nil, nil)
		}
	case 155:
		//line n1ql.y:1246
		{
			yyVAL.mergeActions = algebra.NewMergeActions(yyS[yypt-1].mergeUpdate, yyS[yypt-0].mergeActions.Delete(), yyS[yypt-0].mergeActions.Insert())
		}
	case 156:
		//line n1ql.y:1251
		{
			yyVAL.mergeActions = algebra.NewMergeActions(nil, yyS[yypt-1].mergeDelete, yyS[yypt-0].mergeInsert)
		}
	case 157:
		//line n1ql.y:1256
		{
			yyVAL.mergeActions = algebra.NewMergeActions(nil, nil, yyS[yypt-0].mergeInsert)
		}
	case 158:
		//line n1ql.y:1263
		{
			yyVAL.mergeActions = algebra.NewMergeActions(nil, nil, nil)
		}
	case 159:
		//line n1ql.y:1268
		{
			yyVAL.mergeActions = algebra.NewMergeActions(nil, yyS[yypt-1].mergeDelete, yyS[yypt-0].mergeInsert)
		}
	case 160:
		//line n1ql.y:1273
		{
			yyVAL.mergeActions = algebra.NewMergeActions(nil, nil, yyS[yypt-0].mergeInsert)
		}
	case 161:
		//line n1ql.y:1280
		{
			yyVAL.mergeInsert = nil
		}
	case 162:
		//line n1ql.y:1285
		{
			yyVAL.mergeInsert = yyS[yypt-0].mergeInsert
		}
	case 163:
		//line n1ql.y:1292
		{
			yyVAL.mergeUpdate = algebra.NewMergeUpdate(yyS[yypt-1].set, nil, yyS[yypt-0].expr)
		}
	case 164:
		//line n1ql.y:1297
		{
			yyVAL.mergeUpdate = algebra.NewMergeUpdate(yyS[yypt-2].set, yyS[yypt-1].unset, yyS[yypt-0].expr)
		}
	case 165:
		//line n1ql.y:1302
		{
			yyVAL.mergeUpdate = algebra.NewMergeUpdate(nil, yyS[yypt-1].unset, yyS[yypt-0].expr)
		}
	case 166:
		//line n1ql.y:1309
		{
			yyVAL.mergeDelete = algebra.NewMergeDelete(yyS[yypt-0].expr)
		}
	case 167:
		//line n1ql.y:1316
		{
			yyVAL.mergeInsert = algebra.NewMergeInsert(yyS[yypt-1].expr, yyS[yypt-0].expr)
		}
	case 168:
		//line n1ql.y:1330
		{
			yyVAL.statement = algebra.NewCreatePrimaryIndex(yyS[yypt-1].keyspaceRef, yyS[yypt-0].indexType)
		}
	case 169:
		//line n1ql.y:1335
		{
			yyVAL.statement = algebra.NewCreateIndex(yyS[yypt-8].s, yyS[yypt-6].keyspaceRef, yyS[yypt-4].exprs, yyS[yypt-2].expr, yyS[yypt-1].expr, yyS[yypt-0].indexType)
		}
	case 170:
		yyVAL.s = yyS[yypt-0].s
	case 171:
		//line n1ql.y:1346
		{
			yyVAL.keyspaceRef = algebra.NewKeyspaceRef("", yyS[yypt-0].s, "")
		}
	case 172:
		//line n1ql.y:1351
		{
			yyVAL.keyspaceRef = algebra.NewKeyspaceRef(yyS[yypt-2].s, yyS[yypt-0].s, "")
		}
	case 173:
		//line n1ql.y:1358
		{
			yyVAL.expr = nil
		}
	case 174:
		//line n1ql.y:1363
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 175:
		//line n1ql.y:1370
		{
			yyVAL.indexType = datastore.VIEW
		}
	case 176:
		//line n1ql.y:1375
		{
			yyVAL.indexType = datastore.VIEW
		}
	case 177:
		//line n1ql.y:1380
		{
			yyVAL.indexType = datastore.LSM
		}
	case 178:
		//line n1ql.y:1387
		{
			yyVAL.exprs = expression.Expressions{yyS[yypt-0].expr}
		}
	case 179:
		//line n1ql.y:1392
		{
			yyVAL.exprs = append(yyS[yypt-2].exprs, yyS[yypt-0].expr)
		}
	case 180:
		//line n1ql.y:1399
		{
			e := yyS[yypt-0].expr
			if !e.Indexable() {
				yylex.Error(fmt.Sprintf("Expression not indexable."))
			}

			yyVAL.expr = e
		}
	case 181:
		//line n1ql.y:1410
		{
			yyVAL.expr = nil
		}
	case 182:
		//line n1ql.y:1415
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 183:
		//line n1ql.y:1429
		{
			yyVAL.statement = algebra.NewDropIndex(yyS[yypt-0].keyspaceRef, "#primary")
		}
	case 184:
		//line n1ql.y:1434
		{
			yyVAL.statement = algebra.NewDropIndex(yyS[yypt-2].keyspaceRef, yyS[yypt-0].s)
		}
	case 185:
		//line n1ql.y:1447
		{
			yyVAL.statement = algebra.NewAlterIndex(yyS[yypt-3].keyspaceRef, yyS[yypt-1].s, yyS[yypt-0].s)
		}
	case 186:
		//line n1ql.y:1453
		{
			yyVAL.s = ""
		}
	case 187:
		//line n1ql.y:1458
		{
			yyVAL.s = yyS[yypt-0].s
		}
	case 188:
		//line n1ql.y:1472
		{
			yyVAL.path = expression.NewIdentifier(yyS[yypt-0].s)
		}
	case 189:
		//line n1ql.y:1477
		{
			yyVAL.path = expression.NewField(yyS[yypt-2].path, expression.NewFieldName(yyS[yypt-0].s))
		}
	case 190:
		//line n1ql.y:1482
		{
			field := expression.NewField(yyS[yypt-2].path, expression.NewFieldName(yyS[yypt-0].s))
			field.SetCaseInsensitive(true)
			yyVAL.path = field
		}
	case 191:
		//line n1ql.y:1489
		{
			yyVAL.path = expression.NewField(yyS[yypt-4].path, yyS[yypt-1].expr)
		}
	case 192:
		//line n1ql.y:1494
		{
			field := expression.NewField(yyS[yypt-4].path, yyS[yypt-1].expr)
			field.SetCaseInsensitive(true)
			yyVAL.path = field
		}
	case 193:
		//line n1ql.y:1501
		{
			yyVAL.path = expression.NewElement(yyS[yypt-3].path, yyS[yypt-1].expr)
		}
	case 194:
		yyVAL.expr = yyS[yypt-0].expr
	case 195:
		//line n1ql.y:1518
		{
			yyVAL.expr = expression.NewField(yyS[yypt-2].expr, expression.NewFieldName(yyS[yypt-0].s))
		}
	case 196:
		//line n1ql.y:1523
		{
			field := expression.NewField(yyS[yypt-2].expr, expression.NewFieldName(yyS[yypt-0].s))
			field.SetCaseInsensitive(true)
			yyVAL.expr = field
		}
	case 197:
		//line n1ql.y:1530
		{
			yyVAL.expr = expression.NewField(yyS[yypt-4].expr, yyS[yypt-1].expr)
		}
	case 198:
		//line n1ql.y:1535
		{
			field := expression.NewField(yyS[yypt-4].expr, yyS[yypt-1].expr)
			field.SetCaseInsensitive(true)
			yyVAL.expr = field
		}
	case 199:
		//line n1ql.y:1542
		{
			yyVAL.expr = expression.NewElement(yyS[yypt-3].expr, yyS[yypt-1].expr)
		}
	case 200:
		//line n1ql.y:1547
		{
			yyVAL.expr = expression.NewSlice(yyS[yypt-4].expr, yyS[yypt-2].expr)
		}
	case 201:
		//line n1ql.y:1552
		{
			yyVAL.expr = expression.NewSlice(yyS[yypt-5].expr, yyS[yypt-3].expr, yyS[yypt-1].expr)
		}
	case 202:
		//line n1ql.y:1558
		{
			yyVAL.expr = expression.NewAdd(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 203:
		//line n1ql.y:1563
		{
			yyVAL.expr = expression.NewSub(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 204:
		//line n1ql.y:1568
		{
			yyVAL.expr = expression.NewMult(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 205:
		//line n1ql.y:1573
		{
			yyVAL.expr = expression.NewDiv(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 206:
		//line n1ql.y:1578
		{
			yyVAL.expr = expression.NewMod(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 207:
		//line n1ql.y:1584
		{
			yyVAL.expr = expression.NewConcat(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 208:
		//line n1ql.y:1590
		{
			yyVAL.expr = expression.NewAnd(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 209:
		//line n1ql.y:1595
		{
			yyVAL.expr = expression.NewOr(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 210:
		//line n1ql.y:1600
		{
			yyVAL.expr = expression.NewNot(yyS[yypt-0].expr)
		}
	case 211:
		//line n1ql.y:1606
		{
			yyVAL.expr = expression.NewEq(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 212:
		//line n1ql.y:1611
		{
			yyVAL.expr = expression.NewEq(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 213:
		//line n1ql.y:1616
		{
			yyVAL.expr = expression.NewNE(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 214:
		//line n1ql.y:1621
		{
			yyVAL.expr = expression.NewLT(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 215:
		//line n1ql.y:1626
		{
			yyVAL.expr = expression.NewGT(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 216:
		//line n1ql.y:1631
		{
			yyVAL.expr = expression.NewLE(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 217:
		//line n1ql.y:1636
		{
			yyVAL.expr = expression.NewGE(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 218:
		//line n1ql.y:1641
		{
			yyVAL.expr = expression.NewBetween(yyS[yypt-4].expr, yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 219:
		//line n1ql.y:1646
		{
			yyVAL.expr = expression.NewNotBetween(yyS[yypt-5].expr, yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 220:
		//line n1ql.y:1651
		{
			yyVAL.expr = expression.NewLike(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 221:
		//line n1ql.y:1656
		{
			yyVAL.expr = expression.NewNotLike(yyS[yypt-3].expr, yyS[yypt-0].expr)
		}
	case 222:
		//line n1ql.y:1661
		{
			yyVAL.expr = expression.NewIn(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 223:
		//line n1ql.y:1666
		{
			yyVAL.expr = expression.NewNotIn(yyS[yypt-3].expr, yyS[yypt-0].expr)
		}
	case 224:
		//line n1ql.y:1671
		{
			yyVAL.expr = expression.NewWithin(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 225:
		//line n1ql.y:1676
		{
			yyVAL.expr = expression.NewNotWithin(yyS[yypt-3].expr, yyS[yypt-0].expr)
		}
	case 226:
		//line n1ql.y:1681
		{
			yyVAL.expr = expression.NewIsNull(yyS[yypt-2].expr)
		}
	case 227:
		//line n1ql.y:1686
		{
			yyVAL.expr = expression.NewIsNotNull(yyS[yypt-3].expr)
		}
	case 228:
		//line n1ql.y:1691
		{
			yyVAL.expr = expression.NewIsMissing(yyS[yypt-2].expr)
		}
	case 229:
		//line n1ql.y:1696
		{
			yyVAL.expr = expression.NewIsNotMissing(yyS[yypt-3].expr)
		}
	case 230:
		//line n1ql.y:1701
		{
			yyVAL.expr = expression.NewIsValued(yyS[yypt-2].expr)
		}
	case 231:
		//line n1ql.y:1706
		{
			yyVAL.expr = expression.NewIsNotValued(yyS[yypt-3].expr)
		}
	case 232:
		//line n1ql.y:1711
		{
			yyVAL.expr = expression.NewExists(yyS[yypt-0].expr)
		}
	case 233:
		yyVAL.expr = yyS[yypt-0].expr
	case 234:
		yyVAL.expr = yyS[yypt-0].expr
	case 235:
		//line n1ql.y:1725
		{
			yyVAL.expr = expression.NewIdentifier(yyS[yypt-0].s)
		}
	case 236:
		yyVAL.expr = yyS[yypt-0].expr
	case 237:
		yyVAL.expr = yyS[yypt-0].expr
	case 238:
		//line n1ql.y:1737
		{
			yyVAL.expr = expression.NewNeg(yyS[yypt-0].expr)
		}
	case 239:
		yyVAL.expr = yyS[yypt-0].expr
	case 240:
		yyVAL.expr = yyS[yypt-0].expr
	case 241:
		yyVAL.expr = yyS[yypt-0].expr
	case 242:
		yyVAL.expr = yyS[yypt-0].expr
	case 243:
		//line n1ql.y:1756
		{
			yyVAL.expr = expression.NewField(yyS[yypt-2].expr, expression.NewFieldName(yyS[yypt-0].s))
		}
	case 244:
		//line n1ql.y:1761
		{
			field := expression.NewField(yyS[yypt-2].expr, expression.NewFieldName(yyS[yypt-0].s))
			field.SetCaseInsensitive(true)
			yyVAL.expr = field
		}
	case 245:
		//line n1ql.y:1768
		{
			yyVAL.expr = expression.NewField(yyS[yypt-4].expr, yyS[yypt-1].expr)
		}
	case 246:
		//line n1ql.y:1773
		{
			field := expression.NewField(yyS[yypt-4].expr, yyS[yypt-1].expr)
			field.SetCaseInsensitive(true)
			yyVAL.expr = field
		}
	case 247:
		//line n1ql.y:1780
		{
			yyVAL.expr = expression.NewElement(yyS[yypt-3].expr, yyS[yypt-1].expr)
		}
	case 248:
		//line n1ql.y:1785
		{
			yyVAL.expr = expression.NewSlice(yyS[yypt-4].expr, yyS[yypt-2].expr)
		}
	case 249:
		//line n1ql.y:1790
		{
			yyVAL.expr = expression.NewSlice(yyS[yypt-5].expr, yyS[yypt-3].expr, yyS[yypt-1].expr)
		}
	case 250:
		//line n1ql.y:1796
		{
			yyVAL.expr = expression.NewAdd(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 251:
		//line n1ql.y:1801
		{
			yyVAL.expr = expression.NewSub(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 252:
		//line n1ql.y:1806
		{
			yyVAL.expr = expression.NewMult(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 253:
		//line n1ql.y:1811
		{
			yyVAL.expr = expression.NewDiv(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 254:
		//line n1ql.y:1816
		{
			yyVAL.expr = expression.NewMod(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 255:
		//line n1ql.y:1822
		{
			yyVAL.expr = expression.NewConcat(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 256:
		//line n1ql.y:1836
		{
			yyVAL.expr = expression.NULL_EXPR
		}
	case 257:
		//line n1ql.y:1841
		{
			yyVAL.expr = expression.MISSING_EXPR
		}
	case 258:
		//line n1ql.y:1846
		{
			yyVAL.expr = expression.FALSE_EXPR
		}
	case 259:
		//line n1ql.y:1851
		{
			yyVAL.expr = expression.TRUE_EXPR
		}
	case 260:
		//line n1ql.y:1856
		{
			yyVAL.expr = expression.NewConstant(value.NewValue(yyS[yypt-0].f))
		}
	case 261:
		//line n1ql.y:1861
		{
			yyVAL.expr = expression.NewConstant(value.NewValue(yyS[yypt-0].n))
		}
	case 262:
		//line n1ql.y:1866
		{
			yyVAL.expr = expression.NewConstant(value.NewValue(yyS[yypt-0].s))
		}
	case 263:
		yyVAL.expr = yyS[yypt-0].expr
	case 264:
		yyVAL.expr = yyS[yypt-0].expr
	case 265:
		//line n1ql.y:1886
		{
			yyVAL.expr = expression.NewObjectConstruct(yyS[yypt-1].bindings)
		}
	case 266:
		//line n1ql.y:1893
		{
			yyVAL.bindings = nil
		}
	case 267:
		yyVAL.bindings = yyS[yypt-0].bindings
	case 268:
		//line n1ql.y:1902
		{
			yyVAL.bindings = expression.Bindings{yyS[yypt-0].binding}
		}
	case 269:
		//line n1ql.y:1907
		{
			yyVAL.bindings = append(yyS[yypt-2].bindings, yyS[yypt-0].binding)
		}
	case 270:
		//line n1ql.y:1914
		{
			yyVAL.binding = expression.NewBinding(yyS[yypt-2].s, yyS[yypt-0].expr)
		}
	case 271:
		//line n1ql.y:1921
		{
			yyVAL.expr = expression.NewArrayConstruct(yyS[yypt-1].exprs...)
		}
	case 272:
		//line n1ql.y:1928
		{
			yyVAL.exprs = nil
		}
	case 273:
		yyVAL.exprs = yyS[yypt-0].exprs
	case 274:
		//line n1ql.y:1944
		{
			yyVAL.expr = algebra.NewNamedParameter(yyS[yypt-0].s)
		}
	case 275:
		//line n1ql.y:1949
		{
			yyVAL.expr = algebra.NewPositionalParameter(yyS[yypt-0].n)
		}
	case 276:
		//line n1ql.y:1963
		{
			yyVAL.expr = yyS[yypt-1].expr
		}
	case 277:
		yyVAL.expr = yyS[yypt-0].expr
	case 278:
		yyVAL.expr = yyS[yypt-0].expr
	case 279:
		//line n1ql.y:1976
		{
			yyVAL.expr = expression.NewSimpleCase(yyS[yypt-2].expr, yyS[yypt-1].whenTerms, yyS[yypt-0].expr)
		}
	case 280:
		//line n1ql.y:1983
		{
			yyVAL.whenTerms = expression.WhenTerms{&expression.WhenTerm{yyS[yypt-2].expr, yyS[yypt-0].expr}}
		}
	case 281:
		//line n1ql.y:1988
		{
			yyVAL.whenTerms = append(yyS[yypt-4].whenTerms, &expression.WhenTerm{yyS[yypt-2].expr, yyS[yypt-0].expr})
		}
	case 282:
		//line n1ql.y:1996
		{
			yyVAL.expr = expression.NewSearchedCase(yyS[yypt-1].whenTerms, yyS[yypt-0].expr)
		}
	case 283:
		//line n1ql.y:2003
		{
			yyVAL.expr = nil
		}
	case 284:
		//line n1ql.y:2008
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 285:
		//line n1ql.y:2022
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
	case 286:
		//line n1ql.y:2041
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
	case 287:
		//line n1ql.y:2056
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
	case 288:
		yyVAL.s = yyS[yypt-0].s
	case 289:
		yyVAL.expr = yyS[yypt-0].expr
	case 290:
		yyVAL.expr = yyS[yypt-0].expr
	case 291:
		//line n1ql.y:2090
		{
			yyVAL.expr = expression.NewAny(yyS[yypt-2].bindings, yyS[yypt-1].expr)
		}
	case 292:
		//line n1ql.y:2095
		{
			yyVAL.expr = expression.NewAny(yyS[yypt-2].bindings, yyS[yypt-1].expr)
		}
	case 293:
		//line n1ql.y:2100
		{
			yyVAL.expr = expression.NewEvery(yyS[yypt-2].bindings, yyS[yypt-1].expr)
		}
	case 294:
		//line n1ql.y:2107
		{
			yyVAL.bindings = expression.Bindings{yyS[yypt-0].binding}
		}
	case 295:
		//line n1ql.y:2112
		{
			yyVAL.bindings = append(yyS[yypt-2].bindings, yyS[yypt-0].binding)
		}
	case 296:
		//line n1ql.y:2119
		{
			yyVAL.binding = expression.NewBinding(yyS[yypt-2].s, yyS[yypt-0].expr)
		}
	case 297:
		//line n1ql.y:2124
		{
			yyVAL.binding = expression.NewDescendantBinding(yyS[yypt-2].s, yyS[yypt-0].expr)
		}
	case 298:
		//line n1ql.y:2131
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 299:
		//line n1ql.y:2138
		{
			yyVAL.expr = expression.NewArray(yyS[yypt-4].expr, yyS[yypt-2].bindings, yyS[yypt-1].expr)
		}
	case 300:
		//line n1ql.y:2143
		{
			yyVAL.expr = expression.NewFirst(yyS[yypt-4].expr, yyS[yypt-2].bindings, yyS[yypt-1].expr)
		}
	case 301:
		//line n1ql.y:2157
		{
			yyVAL.expr = yyS[yypt-1].expr
		}
	case 302:
		yyVAL.expr = yyS[yypt-0].expr
	case 303:
		//line n1ql.y:2166
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
