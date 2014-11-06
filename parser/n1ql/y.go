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
	160, 282,
	-2, 230,
	-1, 108,
	168, 63,
	-2, 64,
	-1, 145,
	50, 72,
	67, 72,
	86, 72,
	135, 72,
	-2, 50,
	-1, 172,
	170, 0,
	171, 0,
	172, 0,
	-2, 206,
	-1, 173,
	170, 0,
	171, 0,
	172, 0,
	-2, 207,
	-1, 174,
	170, 0,
	171, 0,
	172, 0,
	-2, 208,
	-1, 175,
	173, 0,
	174, 0,
	175, 0,
	176, 0,
	-2, 209,
	-1, 176,
	173, 0,
	174, 0,
	175, 0,
	176, 0,
	-2, 210,
	-1, 177,
	173, 0,
	174, 0,
	175, 0,
	176, 0,
	-2, 211,
	-1, 178,
	173, 0,
	174, 0,
	175, 0,
	176, 0,
	-2, 212,
	-1, 185,
	75, 0,
	-2, 215,
	-1, 186,
	58, 0,
	150, 0,
	-2, 217,
	-1, 187,
	58, 0,
	150, 0,
	-2, 219,
	-1, 281,
	75, 0,
	-2, 216,
	-1, 282,
	58, 0,
	150, 0,
	-2, 218,
	-1, 283,
	58, 0,
	150, 0,
	-2, 220,
}

const yyNprod = 298
const yyPrivate = 57344

var yyTokenNames []string
var yyStates []string

const yyLast = 2927

var yyAct = []int{

	159, 3, 576, 565, 435, 574, 566, 302, 303, 456,
	92, 93, 516, 525, 298, 209, 192, 490, 306, 211,
	393, 533, 250, 226, 134, 95, 257, 410, 210, 154,
	449, 395, 295, 151, 205, 12, 130, 392, 251, 418,
	66, 146, 155, 381, 337, 132, 133, 127, 229, 114,
	221, 426, 118, 258, 106, 324, 275, 451, 131, 322,
	342, 519, 138, 139, 500, 467, 52, 237, 236, 466,
	425, 163, 164, 165, 166, 167, 168, 169, 170, 171,
	172, 173, 174, 175, 176, 177, 178, 119, 426, 185,
	186, 187, 91, 323, 447, 260, 275, 273, 259, 273,
	70, 341, 160, 161, 261, 136, 137, 425, 240, 72,
	162, 131, 276, 277, 278, 70, 272, 223, 272, 69,
	235, 411, 148, 374, 411, 358, 208, 448, 446, 72,
	73, 74, 75, 107, 69, 376, 237, 273, 234, 233,
	493, 8, 513, 314, 271, 239, 312, 247, 224, 458,
	279, 274, 276, 277, 278, 265, 272, 237, 239, 469,
	470, 110, 252, 268, 195, 197, 199, 471, 364, 365,
	426, 353, 135, 232, 213, 267, 366, 273, 231, 231,
	160, 161, 263, 281, 282, 283, 262, 264, 162, 425,
	70, 274, 276, 277, 278, 290, 272, 227, 309, 275,
	149, 147, 296, 76, 71, 73, 74, 75, 212, 69,
	70, 108, 128, 421, 179, 149, 249, 313, 300, 575,
	275, 316, 222, 317, 71, 73, 74, 75, 180, 69,
	514, 108, 310, 241, 305, 400, 326, 570, 327, 301,
	517, 330, 331, 332, 108, 249, 108, 457, 190, 311,
	340, 189, 188, 495, 207, 497, 556, 230, 230, 304,
	343, 589, 588, 291, 357, 292, 140, 293, 181, 584,
	315, 362, 285, 351, 367, 305, 284, 557, 547, 352,
	273, 99, 325, 67, 68, 238, 329, 200, 460, 437,
	375, 335, 336, 279, 274, 276, 277, 278, 307, 272,
	384, 273, 98, 129, 356, 67, 541, 191, 387, 389,
	390, 388, 350, 183, 279, 274, 276, 277, 278, 403,
	272, 494, 515, 180, 398, 242, 396, 115, 220, 289,
	182, 286, 101, 382, 526, 67, 386, 123, 539, 453,
	321, 198, 308, 416, 385, 320, 407, 423, 409, 319,
	68, 579, 537, 399, 213, 582, 348, 194, 580, 538,
	196, 252, 555, 142, 412, 586, 148, 430, 404, 405,
	406, 97, 68, 344, 231, 231, 231, 296, 413, 122,
	415, 427, 428, 121, 439, 422, 424, 438, 417, 67,
	440, 441, 345, 354, 355, 443, 280, 383, 452, 420,
	420, 105, 68, 455, 442, 184, 444, 445, 67, 434,
	253, 585, 462, 180, 215, 131, 180, 180, 180, 180,
	180, 180, 243, 244, 548, 120, 552, 472, 255, 397,
	299, 65, 109, 478, 103, 102, 454, 592, 256, 468,
	591, 347, 219, 473, 474, 147, 465, 482, 487, 484,
	485, 464, 567, 230, 230, 230, 68, 408, 228, 131,
	483, 225, 124, 523, 339, 67, 104, 481, 492, 396,
	531, 480, 504, 463, 461, 68, 491, 334, 419, 419,
	488, 501, 509, 333, 486, 328, 363, 218, 510, 368,
	369, 370, 371, 372, 373, 496, 587, 551, 144, 414,
	201, 2, 349, 346, 202, 203, 204, 1, 506, 507,
	459, 214, 540, 94, 562, 569, 180, 512, 511, 450,
	252, 520, 518, 524, 542, 394, 391, 489, 436, 527,
	528, 532, 479, 536, 297, 534, 534, 535, 491, 36,
	35, 546, 34, 544, 545, 543, 18, 550, 17, 16,
	15, 77, 560, 561, 549, 14, 13, 86, 553, 554,
	559, 7, 558, 563, 564, 6, 5, 4, 577, 377,
	571, 573, 572, 578, 378, 568, 287, 288, 193, 581,
	294, 96, 100, 150, 583, 522, 521, 499, 498, 433,
	338, 590, 577, 577, 594, 595, 593, 77, 248, 141,
	502, 503, 89, 86, 206, 254, 143, 145, 63, 64,
	91, 28, 117, 27, 47, 23, 50, 49, 26, 88,
	55, 113, 112, 111, 25, 125, 126, 72, 22, 44,
	43, 87, 20, 19, 0, 0, 0, 78, 0, 0,
	0, 53, 0, 0, 0, 0, 31, 0, 89, 0,
	0, 0, 54, 0, 0, 0, 91, 0, 0, 0,
	86, 0, 11, 0, 0, 88, 0, 67, 0, 0,
	0, 0, 0, 72, 0, 0, 0, 87, 29, 0,
	0, 0, 0, 78, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 90, 0, 0, 33, 0, 0,
	0, 0, 0, 0, 0, 89, 0, 0, 70, 529,
	530, 0, 0, 91, 79, 80, 81, 82, 83, 84,
	85, 76, 71, 73, 74, 75, 77, 69, 0, 212,
	72, 0, 86, 0, 68, 0, 0, 0, 0, 0,
	90, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 32, 30, 70, 475, 476, 0, 0, 0,
	79, 80, 81, 82, 83, 84, 85, 76, 71, 73,
	74, 75, 77, 69, 0, 0, 0, 89, 86, 0,
	0, 0, 0, 0, 0, 91, 0, 0, 0, 0,
	0, 0, 0, 0, 88, 0, 0, 90, 0, 0,
	0, 0, 72, 0, 0, 0, 87, 0, 0, 0,
	0, 70, 78, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 89, 76, 71, 73, 74, 75, 0,
	69, 91, 0, 0, 0, 0, 0, 0, 0, 0,
	88, 0, 0, 0, 0, 58, 61, 0, 72, 0,
	0, 0, 87, 0, 0, 48, 0, 0, 78, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 90,
	0, 0, 0, 0, 0, 213, 0, 0, 0, 60,
	0, 0, 0, 70, 0, 38, 62, 0, 0, 79,
	80, 81, 82, 83, 84, 85, 76, 71, 73, 74,
	75, 77, 69, 0, 0, 379, 0, 86, 0, 0,
	0, 0, 0, 0, 0, 90, 0, 0, 0, 0,
	24, 0, 0, 0, 0, 37, 0, 380, 0, 70,
	431, 0, 0, 432, 0, 79, 80, 81, 82, 83,
	84, 85, 76, 71, 73, 74, 75, 77, 69, 0,
	0, 0, 89, 86, 0, 0, 0, 0, 0, 59,
	91, 0, 0, 0, 0, 0, 0, 39, 0, 88,
	0, 0, 0, 0, 0, 0, 0, 72, 0, 0,
	0, 87, 0, 0, 0, 0, 0, 78, 0, 0,
	41, 40, 42, 21, 0, 45, 46, 51, 89, 56,
	0, 57, 0, 0, 0, 0, 91, 0, 0, 0,
	0, 0, 0, 0, 0, 88, 0, 0, 0, 0,
	0, 0, 0, 72, 0, 0, 0, 87, 0, 0,
	0, 0, 0, 78, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 90, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 70, 0,
	0, 0, 0, 0, 79, 80, 81, 82, 83, 84,
	85, 76, 71, 73, 74, 75, 77, 69, 0, 0,
	0, 0, 86, 0, 0, 0, 0, 0, 0, 0,
	90, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 70, 359, 360, 0, 0, 0,
	79, 80, 81, 82, 83, 84, 85, 76, 71, 73,
	74, 75, 77, 69, 0, 212, 0, 89, 86, 0,
	0, 0, 0, 0, 0, 91, 0, 0, 0, 0,
	0, 0, 0, 0, 88, 0, 0, 0, 0, 0,
	0, 0, 72, 0, 0, 0, 87, 0, 0, 0,
	0, 0, 78, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 89, 0, 0, 0, 0, 0, 0,
	0, 91, 0, 0, 0, 0, 0, 0, 0, 0,
	88, 0, 0, 0, 0, 0, 0, 0, 72, 0,
	0, 0, 87, 0, 0, 0, 0, 0, 78, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 90,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 70, 269, 0, 0, 270, 0, 79,
	80, 81, 82, 83, 84, 85, 76, 71, 73, 74,
	75, 77, 69, 0, 0, 0, 0, 86, 0, 0,
	0, 0, 0, 0, 0, 90, 0, 0, 0, 0,
	0, 213, 0, 0, 0, 0, 0, 0, 0, 70,
	0, 0, 0, 0, 0, 79, 80, 81, 82, 83,
	84, 85, 76, 71, 73, 74, 75, 77, 266, 0,
	0, 0, 89, 86, 0, 0, 0, 0, 0, 0,
	91, 0, 0, 0, 0, 0, 0, 0, 0, 88,
	0, 0, 0, 0, 0, 0, 0, 72, 0, 0,
	0, 87, 0, 0, 0, 0, 0, 78, 0, 0,
	0, 451, 0, 0, 0, 0, 0, 0, 89, 0,
	0, 0, 0, 0, 0, 0, 91, 0, 0, 0,
	0, 0, 0, 0, 0, 88, 0, 0, 0, 0,
	0, 0, 0, 72, 0, 0, 0, 87, 0, 0,
	0, 0, 0, 78, 0, 0, 0, 0, 0, 0,
	0, 249, 0, 0, 90, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 70, 0,
	0, 0, 0, 0, 79, 80, 81, 82, 83, 84,
	85, 76, 71, 73, 74, 75, 77, 69, 0, 0,
	0, 0, 86, 0, 0, 0, 0, 0, 0, 0,
	90, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 70, 0, 0, 0, 0, 0,
	79, 80, 81, 82, 83, 84, 85, 76, 71, 73,
	74, 75, 77, 69, 0, 0, 0, 89, 86, 0,
	0, 0, 0, 0, 0, 91, 0, 0, 0, 0,
	0, 0, 0, 0, 88, 0, 0, 0, 0, 0,
	0, 0, 72, 0, 0, 0, 87, 0, 0, 0,
	0, 0, 78, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 89, 0, 0, 0, 0, 0, 0,
	0, 91, 0, 0, 0, 0, 0, 0, 0, 0,
	88, 0, 0, 0, 0, 0, 0, 0, 72, 0,
	0, 0, 87, 0, 0, 0, 0, 0, 78, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 90,
	0, 0, 0, 0, 0, 0, 58, 61, 0, 0,
	0, 0, 0, 70, 508, 0, 48, 0, 0, 79,
	80, 81, 82, 83, 84, 85, 76, 71, 73, 74,
	75, 77, 69, 216, 0, 0, 0, 86, 0, 0,
	60, 0, 0, 0, 10, 90, 38, 62, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 70,
	505, 0, 0, 0, 0, 79, 80, 81, 82, 83,
	84, 85, 76, 71, 73, 74, 75, 0, 69, 0,
	0, 24, 89, 0, 0, 9, 37, 77, 0, 0,
	91, 0, 0, 86, 0, 0, 0, 0, 0, 88,
	0, 0, 0, 0, 0, 0, 0, 72, 0, 0,
	0, 87, 0, 0, 0, 0, 0, 78, 0, 0,
	59, 0, 0, 0, 0, 0, 0, 0, 39, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 89, 0,
	0, 0, 0, 0, 0, 0, 91, 0, 0, 0,
	0, 41, 40, 42, 21, 88, 45, 46, 51, 77,
	56, 0, 57, 72, 0, 86, 0, 87, 0, 0,
	0, 0, 0, 78, 90, 0, 0, 217, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 70, 429,
	0, 0, 0, 0, 79, 80, 81, 82, 83, 84,
	85, 76, 71, 73, 74, 75, 402, 69, 0, 0,
	89, 0, 0, 0, 0, 77, 0, 0, 91, 0,
	0, 86, 0, 0, 0, 0, 0, 88, 0, 0,
	90, 0, 0, 0, 0, 72, 0, 0, 0, 87,
	0, 0, 0, 0, 70, 78, 0, 0, 0, 0,
	79, 80, 81, 82, 83, 84, 85, 76, 71, 73,
	74, 75, 0, 69, 0, 0, 89, 0, 0, 0,
	0, 0, 0, 0, 91, 0, 0, 0, 0, 0,
	0, 0, 0, 88, 0, 0, 0, 0, 0, 0,
	0, 72, 0, 0, 0, 87, 0, 86, 0, 0,
	0, 78, 90, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 401, 0, 0, 70, 0, 0, 0,
	0, 0, 79, 80, 81, 82, 83, 84, 85, 76,
	71, 73, 74, 75, 318, 69, 0, 0, 0, 0,
	0, 0, 89, 0, 0, 0, 77, 0, 0, 0,
	91, 0, 86, 0, 0, 0, 0, 0, 90, 88,
	0, 0, 0, 0, 0, 0, 0, 72, 0, 0,
	0, 0, 70, 0, 0, 0, 0, 0, 79, 80,
	81, 82, 83, 84, 85, 76, 71, 73, 74, 75,
	246, 69, 77, 0, 0, 0, 0, 89, 86, 0,
	0, 0, 0, 0, 0, 91, 0, 0, 0, 0,
	0, 0, 0, 0, 88, 0, 0, 0, 0, 0,
	0, 0, 72, 0, 0, 0, 87, 0, 0, 0,
	0, 0, 78, 0, 90, 0, 245, 0, 0, 0,
	0, 0, 0, 89, 0, 0, 0, 0, 70, 0,
	0, 91, 0, 0, 0, 0, 0, 0, 0, 0,
	88, 76, 71, 73, 74, 75, 0, 69, 72, 0,
	0, 0, 87, 0, 0, 0, 0, 0, 78, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 90,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 70, 0, 0, 0, 0, 0, 79,
	80, 81, 82, 83, 84, 85, 76, 71, 73, 74,
	75, 77, 69, 0, 0, 0, 0, 86, 0, 0,
	0, 0, 0, 0, 0, 90, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 70,
	0, 0, 0, 0, 0, 79, 80, 81, 82, 83,
	84, 85, 76, 71, 73, 74, 75, 77, 69, 0,
	0, 0, 89, 86, 0, 0, 0, 0, 0, 0,
	91, 0, 0, 0, 0, 0, 0, 0, 0, 88,
	0, 0, 0, 0, 0, 0, 0, 72, 0, 77,
	0, 87, 0, 0, 0, 86, 0, 78, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 89, 0,
	0, 0, 0, 0, 0, 0, 91, 0, 0, 0,
	0, 0, 58, 61, 0, 88, 0, 0, 0, 0,
	0, 0, 48, 72, 0, 0, 0, 87, 0, 0,
	89, 0, 0, 78, 0, 0, 0, 0, 91, 0,
	116, 0, 0, 0, 90, 0, 60, 88, 0, 0,
	10, 0, 38, 62, 0, 72, 0, 0, 70, 87,
	0, 0, 0, 0, 79, 80, 81, 82, 83, 84,
	85, 76, 71, 73, 74, 75, 0, 69, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 24, 0, 0,
	90, 9, 37, 153, 0, 0, 0, 58, 61, 0,
	0, 0, 0, 0, 70, 0, 0, 48, 0, 0,
	79, 80, 81, 82, 83, 84, 85, 76, 71, 73,
	74, 75, 90, 69, 152, 86, 59, 0, 157, 0,
	0, 60, 0, 0, 39, 10, 70, 38, 62, 0,
	0, 0, 79, 80, 81, 82, 83, 84, 85, 76,
	71, 73, 74, 75, 0, 69, 0, 41, 40, 42,
	21, 0, 45, 46, 51, 0, 56, 86, 57, 0,
	89, 0, 24, 0, 0, 0, 9, 37, 91, 0,
	0, 0, 0, 158, 0, 0, 0, 88, 0, 0,
	0, 0, 0, 0, 0, 72, 156, 0, 0, 87,
	58, 61, 0, 0, 0, 0, 0, 0, 0, 0,
	48, 59, 89, 0, 0, 0, 0, 0, 0, 39,
	91, 0, 0, 0, 0, 0, 0, 0, 0, 88,
	0, 157, 0, 0, 60, 0, 0, 72, 10, 0,
	38, 62, 41, 40, 42, 21, 0, 45, 46, 51,
	0, 56, 0, 57, 0, 0, 0, 0, 0, 0,
	0, 0, 90, 0, 0, 0, 0, 0, 158, 0,
	0, 0, 0, 0, 0, 24, 70, 0, 0, 9,
	37, 0, 79, 80, 81, 82, 83, 84, 85, 76,
	71, 73, 74, 75, 0, 69, 0, 0, 0, 156,
	0, 0, 0, 0, 90, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 59, 0, 0, 0, 70, 0,
	0, 0, 39, 0, 0, 0, 0, 82, 83, 84,
	85, 76, 71, 73, 74, 75, 55, 69, 0, 58,
	61, 0, 0, 0, 0, 41, 40, 42, 21, 48,
	45, 46, 51, 0, 56, 0, 57, 53, 0, 58,
	61, 0, 31, 0, 0, 0, 0, 0, 54, 48,
	0, 158, 0, 60, 0, 0, 0, 10, 11, 38,
	62, 0, 0, 67, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 60, 29, 0, 0, 10, 0, 38,
	62, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 33, 24, 0, 0, 0, 9, 37,
	0, 0, 58, 61, 0, 0, 0, 0, 0, 0,
	0, 0, 48, 0, 24, 0, 0, 0, 9, 37,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	68, 0, 0, 59, 0, 0, 60, 0, 0, 0,
	10, 39, 38, 62, 0, 0, 0, 0, 32, 30,
	0, 0, 0, 59, 0, 0, 0, 0, 0, 0,
	0, 39, 0, 0, 41, 40, 42, 21, 0, 45,
	46, 51, 0, 56, 0, 57, 0, 24, 0, 0,
	0, 9, 37, 0, 41, 40, 42, 21, 0, 45,
	46, 51, 0, 56, 0, 57, 477, 58, 61, 0,
	0, 0, 0, 0, 0, 0, 0, 48, 0, 0,
	0, 0, 0, 0, 0, 0, 59, 0, 0, 0,
	0, 0, 0, 0, 39, 0, 0, 0, 0, 0,
	0, 60, 0, 0, 0, 10, 0, 38, 62, 0,
	0, 67, 0, 0, 0, 58, 61, 41, 40, 42,
	21, 0, 45, 46, 51, 48, 56, 0, 57, 361,
	58, 61, 0, 0, 0, 0, 0, 0, 0, 0,
	48, 0, 24, 0, 0, 0, 9, 37, 0, 60,
	0, 0, 0, 10, 0, 38, 62, 0, 0, 0,
	0, 0, 0, 0, 60, 0, 0, 0, 10, 0,
	38, 62, 0, 0, 0, 0, 0, 0, 68, 0,
	0, 59, 0, 0, 0, 0, 0, 0, 0, 39,
	24, 0, 0, 0, 9, 37, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 24, 0, 0, 0, 9,
	37, 0, 41, 40, 42, 21, 0, 45, 46, 51,
	0, 56, 0, 57, 0, 0, 0, 0, 0, 59,
	0, 0, 0, 0, 0, 0, 0, 39, 0, 0,
	0, 0, 0, 0, 59, 0, 0, 0, 0, 0,
	0, 0, 39, 116, 0, 0, 0, 0, 0, 0,
	41, 40, 42, 21, 0, 45, 46, 51, 0, 56,
	0, 57, 0, 0, 0, 41, 40, 42, 21, 0,
	45, 46, 51, 0, 56, 0, 57,
}
var yyPact = []int{

	2521, -1000, -1000, 2130, -1000, -1000, -1000, -1000, -1000, 2762,
	2762, 615, -1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000,
	-1000, -1000, -1000, -1000, 2762, -1000, -1000, -1000, 238, 370,
	369, 414, 55, 367, -1000, -1000, -1000, -1000, -1000, -1000,
	-1000, -1000, -1000, -1000, -1000, -1000, -1000, 1, 2747, -1000,
	-1000, 2699, -1000, 323, 277, 402, 57, 2762, 16, 16,
	16, 2762, 2762, -1000, -1000, 290, 413, 90, 2279, 24,
	2762, 2762, 2762, 2762, 2762, 2762, 2762, 2762, 2762, 2762,
	2762, 2762, 2762, 2762, 2762, 2762, 837, 255, 2762, 2762,
	2762, 164, 2302, 26, -1000, -64, 281, 356, 337, 283,
	-1000, 484, 55, 55, 55, 115, -42, 198, -1000, 55,
	1568, 446, -1000, -1000, 2084, 182, 2762, -13, 2130, -1000,
	401, 41, 398, 55, 55, -24, -29, -1000, -48, -97,
	-31, 2130, -9, -1000, 175, -1000, -9, -9, 1955, 1909,
	69, -1000, 18, 290, -1000, 366, -1000, -130, -70, -73,
	-1000, -63, 2382, 2194, 2762, -1000, -1000, -1000, -1000, 1115,
	-1000, -1000, 2762, 1069, -49, -49, -64, -64, -64, 46,
	2302, 2162, 2344, 2344, 2344, 1854, 1854, 1854, 1854, 137,
	-1000, 837, 2762, 2762, 2762, 647, 26, 26, -1000, 188,
	-1000, -1000, 239, -1000, 2762, -1000, 231, -1000, 231, -1000,
	231, 2762, 362, 362, 115, 139, -1000, 196, 42, -1000,
	-1000, -1000, 18, -1000, 108, -15, 2762, -18, -1000, 182,
	2762, -1000, 2762, 1778, -1000, 258, 254, -1000, 249, -124,
	-1000, -75, -128, -1000, 57, 2762, -1000, 2762, 444, 16,
	2762, 2762, 2762, 442, 436, 16, 16, 409, -1000, 2762,
	-66, -1000, -110, 69, 306, -1000, 217, 198, 15, 42,
	42, 2194, -63, 2762, -63, 2130, -54, -1000, 940, -1000,
	2604, 837, 12, 2762, 837, 837, 837, 837, 837, 837,
	116, 647, 26, 26, -1000, -1000, -1000, -1000, -1000, 2762,
	2130, -1000, -1000, -1000, -32, -1000, 894, 253, -1000, 2762,
	253, 69, 98, 69, 15, 15, 360, -1000, 198, -1000,
	-1000, 75, -1000, 1722, -1000, -1000, 1650, 2130, 2762, 55,
	55, 55, 41, 42, 41, -1000, 2130, 2130, -1000, -1000,
	2130, 2130, 2130, -1000, -1000, -22, -22, 165, -1000, 483,
	2130, 18, 2762, 409, 88, 88, 2762, -1000, -1000, -1000,
	-1000, 115, -113, -1000, -130, -130, -1000, 2130, -1000, -1000,
	-1000, -1000, 1594, -27, -1000, -1000, 2762, 765, -67, -67,
	-65, -65, -65, 13, 837, 2130, 2762, -1000, -1000, -1000,
	-1000, 177, 177, 2762, 2130, 177, 177, 281, 69, 281,
	281, -39, -1000, -76, -40, -1000, 6, 2762, -1000, 248,
	231, -1000, 2762, 2130, 106, -11, -1000, -1000, -1000, 178,
	433, 2762, 432, -1000, 2762, -1000, 2130, -1000, -1000, -130,
	-99, -103, -1000, 719, -1000, 3, 2762, 198, 198, -1000,
	590, -1000, 2541, -27, -1000, -1000, -1000, 2382, -1000, 2130,
	-1000, -1000, 177, 281, 177, 177, 15, 2762, 15, -1000,
	-1000, 16, 2130, 362, -21, 2130, -1000, 176, 2762, -1000,
	128, -1000, 2130, -1000, -10, 198, 42, 42, -1000, -1000,
	-1000, 2762, 1465, 115, 115, -1000, -1000, -1000, 1419, -1000,
	-63, 2762, -1000, 177, -1000, -1000, -1000, 1290, -1000, -25,
	-1000, 172, 94, 198, -1000, -1000, -100, 41, 407, -1000,
	18, 243, -130, -130, 544, -1000, -1000, -1000, -1000, 2130,
	-1000, -1000, 429, 16, 15, 15, 281, 272, 247, 209,
	-1000, -1000, -1000, 2762, -66, -1000, 196, 198, 198, -1000,
	-1000, -1000, -1000, -1000, -113, -1000, 177, 152, 344, 362,
	69, 481, 2130, 357, 243, 243, -1000, 225, 151, 94,
	106, 2762, 2762, -1000, -1000, 139, 69, 389, 281, -1000,
	2130, 2130, 91, 98, 69, 73, -1000, 2762, 177, -1000,
	271, -1000, 69, -1000, -1000, 268, -1000, 1244, -1000, 143,
	331, -1000, 285, -1000, 465, 136, 135, 69, 377, 374,
	73, 2762, 2762, -1000, -1000, -1000,
}
var yyPgo = []int{

	0, 633, 632, 630, 629, 628, 47, 626, 625, 0,
	141, 214, 36, 303, 38, 22, 19, 28, 15, 24,
	624, 623, 622, 621, 50, 327, 618, 617, 616, 46,
	45, 285, 27, 615, 614, 613, 612, 35, 611, 66,
	609, 608, 607, 431, 606, 41, 39, 605, 20, 26,
	54, 133, 604, 34, 13, 266, 599, 6, 598, 44,
	590, 588, 587, 586, 585, 42, 33, 583, 40, 582,
	581, 32, 580, 578, 16, 577, 576, 574, 569, 501,
	567, 566, 565, 561, 556, 555, 550, 549, 548, 546,
	542, 540, 539, 401, 43, 14, 534, 532, 528, 4,
	17, 527, 21, 7, 37, 526, 8, 31, 525, 519,
	30, 12, 515, 514, 3, 2, 5, 23, 48, 512,
	9, 510, 507, 29, 503, 18, 502,
}
var yyR1 = []int{

	0, 122, 122, 79, 79, 79, 79, 80, 81, 82,
	82, 82, 82, 82, 83, 89, 89, 89, 37, 38,
	38, 38, 38, 38, 38, 38, 39, 39, 41, 40,
	68, 67, 67, 67, 67, 67, 123, 123, 66, 66,
	65, 65, 65, 18, 18, 17, 17, 16, 44, 44,
	43, 42, 42, 42, 42, 124, 124, 45, 45, 45,
	46, 46, 46, 50, 51, 49, 49, 53, 53, 52,
	125, 125, 47, 47, 47, 126, 126, 54, 55, 55,
	56, 15, 15, 14, 57, 57, 58, 59, 59, 60,
	12, 12, 61, 61, 62, 63, 63, 64, 70, 70,
	69, 72, 72, 71, 78, 78, 77, 77, 74, 74,
	73, 76, 76, 75, 84, 84, 93, 93, 96, 96,
	95, 94, 99, 99, 98, 97, 97, 85, 85, 86,
	87, 87, 87, 103, 105, 105, 104, 110, 110, 109,
	101, 101, 100, 100, 19, 102, 32, 32, 106, 108,
	108, 107, 88, 88, 111, 111, 111, 111, 112, 112,
	112, 116, 116, 113, 113, 113, 114, 115, 90, 90,
	117, 118, 118, 119, 119, 120, 120, 120, 91, 91,
	92, 121, 121, 48, 48, 48, 48, 48, 48, 9,
	9, 9, 9, 9, 9, 9, 9, 9, 9, 9,
	9, 9, 9, 9, 9, 9, 9, 9, 9, 9,
	9, 9, 9, 9, 9, 9, 9, 9, 9, 9,
	9, 9, 9, 9, 9, 9, 9, 9, 10, 10,
	10, 10, 10, 10, 10, 10, 10, 11, 11, 11,
	11, 11, 11, 11, 11, 11, 11, 11, 11, 11,
	11, 1, 1, 1, 1, 1, 1, 2, 2, 3,
	8, 8, 7, 7, 6, 4, 13, 13, 5, 5,
	20, 21, 21, 22, 25, 25, 23, 24, 24, 33,
	33, 33, 34, 26, 26, 27, 27, 27, 30, 30,
	29, 29, 31, 28, 28, 35, 36, 36,
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
	6, 0, 6, 2, 3, 2, 1, 2, 6, 11,
	1, 1, 3, 0, 3, 0, 2, 2, 5, 5,
	6, 0, 3, 1, 3, 3, 5, 5, 4, 1,
	3, 3, 5, 5, 4, 5, 6, 3, 3, 3,
	3, 3, 3, 3, 3, 2, 3, 3, 3, 3,
	3, 3, 3, 5, 6, 3, 4, 3, 4, 3,
	4, 3, 4, 3, 4, 3, 4, 2, 1, 1,
	1, 1, 1, 2, 1, 1, 1, 1, 3, 3,
	5, 5, 4, 5, 6, 3, 3, 3, 3, 3,
	3, 1, 1, 1, 1, 1, 1, 1, 1, 3,
	0, 1, 1, 3, 3, 3, 0, 1, 1, 1,
	3, 1, 1, 3, 4, 5, 2, 0, 2, 4,
	5, 4, 1, 1, 1, 4, 4, 4, 1, 3,
	3, 3, 2, 6, 6, 3, 1, 1,
}
var yyChk = []int{

	-1000, -122, -79, -9, -80, -81, -82, -83, -10, 87,
	46, 47, -37, -84, -85, -86, -87, -88, -89, -1,
	-2, 156, -5, -33, 83, -20, -26, -35, -38, 63,
	138, 31, 137, 82, -90, -91, -92, 88, 48, 130,
	154, 153, 155, -3, -4, 158, 159, -34, 18, -27,
	-28, 160, -39, 26, 37, 5, 162, 164, 8, 122,
	42, 9, 49, -41, -40, -43, -68, 52, 119, 183,
	164, 178, 83, 179, 180, 181, 177, 7, 93, 170,
	171, 172, 173, 174, 175, 176, 13, 87, 75, 58,
	150, 66, -9, -9, -79, -9, -70, 133, 64, 43,
	-69, 94, 65, 65, 52, -93, -50, -51, 156, 65,
	160, -21, -22, -23, -9, -25, 146, -36, -9, -37,
	102, 60, 102, 60, 60, -8, -7, -6, 155, -13,
	-12, -9, -30, -29, -19, 156, -30, -30, -9, -9,
	-55, -56, 73, -44, -43, -42, -45, -51, -50, 125,
	-67, -66, 35, 4, -123, -65, 107, 39, 179, -9,
	156, 157, 164, -9, -9, -9, -9, -9, -9, -9,
	-9, -9, -9, -9, -9, -9, -9, -9, -9, -11,
	-10, 13, 75, 58, 150, -9, -9, -9, 88, 87,
	84, 143, -74, -73, 76, -39, 4, -39, 4, -39,
	4, 16, -93, -93, -93, -53, -52, 139, 168, -18,
	-17, -16, 10, 156, -93, -13, 35, 179, 41, -25,
	146, -24, 40, -9, 161, 60, -117, 156, 60, -118,
	-51, -50, -118, 163, 167, 168, 165, 167, -31, 167,
	117, 58, 150, -31, -31, 51, 51, -57, -58, 147,
	-15, -14, -16, -55, -47, 62, 72, -49, 183, 168,
	168, 167, -66, -123, -66, -9, 183, -18, -9, 165,
	168, 7, 183, 164, 178, 83, 179, 180, 181, 177,
	-11, -9, -9, -9, 88, 84, 143, -76, -75, 90,
	-9, -39, -39, -39, -72, -71, -9, -96, -95, 68,
	-95, -53, -103, -106, 120, 136, -125, 102, -51, 156,
	-16, 141, 161, -9, 161, -24, -9, -9, 126, 91,
	91, 91, 183, 168, 183, -6, -9, -9, 41, -29,
	-9, -9, -9, 41, 41, -30, -30, -59, -60, 55,
	-9, 167, 170, -57, 67, 86, -124, 135, 50, -126,
	95, -18, -48, 156, -51, -51, -65, -9, 179, 165,
	166, 165, -9, -11, 156, 157, 164, -9, -11, -11,
	-11, -11, -11, -11, 7, -9, 167, -78, -77, 11,
	33, -94, -37, 144, -9, -94, -37, -57, -106, -57,
	-57, -105, -104, -48, -108, -107, -48, 69, -18, -45,
	160, 161, 126, -9, -118, -118, -118, -117, -51, -117,
	-32, 146, -32, -68, 16, -14, -9, -59, -46, -51,
	-50, 125, -46, -9, -53, 183, 164, -49, -49, 165,
	-9, 165, 168, -11, -71, -99, -98, 112, -99, -9,
	-99, -99, -74, -57, -74, -74, 167, 170, 167, -110,
	-109, 51, -9, 91, -37, -9, -120, 141, 160, -121,
	110, 41, -9, 41, -12, -49, 168, 168, -18, 156,
	157, 164, -9, -18, -18, 165, 166, 165, -9, -97,
	-66, -123, -99, -74, -99, -99, -104, -9, -107, -101,
	-100, -19, -95, 161, 145, 77, -12, 127, -61, -62,
	74, -18, -51, -51, -9, 165, -53, -53, 165, -9,
	-99, -110, -32, 167, 58, 150, -111, 146, -17, 161,
	-117, -63, -64, 56, -15, -54, 91, -49, -49, 165,
	166, 41, -100, -102, -48, -102, -74, 80, 87, 91,
	-119, 97, -9, -125, -18, -18, -99, 126, 80, -95,
	-57, 16, 69, -54, -54, 137, 31, 126, -111, -120,
	-9, -9, -113, -103, -106, -114, -57, 63, -74, -112,
	146, -57, -106, -57, -116, 146, -115, -9, -99, 80,
	87, -57, 87, -57, 126, 80, 80, 31, 126, 126,
	-114, 63, 63, -116, -115, -115,
}
var yyDef = []int{

	0, -2, 1, 2, 3, 4, 5, 6, 189, 0,
	0, 0, 8, 9, 10, 11, 12, 13, 14, 228,
	229, -2, 231, 232, 0, 234, 235, 236, 98, 0,
	0, 0, 0, 0, 15, 16, 17, 251, 252, 253,
	254, 255, 256, 257, 258, 268, 269, 0, 0, 283,
	284, 0, 19, 0, 0, 0, 260, 266, 0, 0,
	0, 0, 0, 26, 27, 78, 48, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 205, 227, 7, 233, 108, 0, 0, 0,
	99, 0, 0, 0, 0, 67, 0, 43, -2, 0,
	266, 0, 271, 272, 0, 277, 0, 0, 296, 297,
	0, 0, 0, 0, 0, 0, 261, 262, 0, 0,
	267, 90, 0, 288, 0, 144, 0, 0, 0, 0,
	84, 79, 0, 78, 49, -2, 51, 65, 0, 0,
	30, 31, 0, 0, 0, 38, 36, 37, 40, 43,
	190, 191, 0, 0, 197, 198, 199, 200, 201, 202,
	203, 204, -2, -2, -2, -2, -2, -2, -2, 0,
	237, 0, 0, 0, 0, -2, -2, -2, 221, 0,
	223, 225, 111, 109, 0, 20, 0, 22, 0, 24,
	0, 0, 118, 0, 67, 0, 68, 70, 0, 117,
	44, 45, 0, 47, 0, 0, 0, 0, 270, 277,
	0, 276, 0, 0, 295, 0, 0, 170, 0, 0,
	171, 0, 0, 259, 0, 0, 265, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 87, 85, 0,
	80, 81, 0, 84, 0, 73, 75, 43, 0, 0,
	0, 0, 32, 0, 33, 34, 0, 42, 0, 194,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, -2, -2, -2, 222, 224, 226, 18, 112, 0,
	110, 21, 23, 25, 100, 101, 104, 0, 119, 0,
	0, 84, 84, 84, 0, 0, 0, 71, 43, 64,
	46, 0, 279, 0, 281, 273, 0, 278, 0, 0,
	0, 0, 0, 0, 0, 263, 264, 91, 285, 289,
	292, 290, 291, 286, 287, 146, 146, 0, 88, 0,
	86, 0, 0, 87, 0, 0, 0, 55, 56, 74,
	76, 67, 66, 183, 65, 65, 39, 35, 41, 192,
	193, 195, 0, 213, 238, 239, 0, 0, 245, 246,
	247, 248, 249, 250, 0, 113, 0, 103, 105, 106,
	107, 122, 122, 0, 120, 122, 122, 108, 84, 108,
	108, 133, 134, 0, 148, 149, 137, 0, 116, 0,
	0, 280, 0, 274, 175, 0, 178, 179, 172, 181,
	0, 0, 0, 28, 0, 82, 83, 29, 52, 65,
	0, 0, 53, 43, 57, 0, 0, 43, 43, 196,
	0, 242, 0, 214, 102, 114, 123, 0, 115, 121,
	127, 128, 122, 108, 122, 122, 0, 0, 0, 151,
	138, 0, 69, 0, 0, 275, 168, 0, 0, 180,
	0, 293, 147, 294, 92, 43, 0, 0, 54, 184,
	185, 0, 0, 67, 67, 240, 241, 243, 0, 124,
	125, 0, 129, 122, 131, 132, 135, 137, 150, 146,
	140, 0, 154, 0, 176, 177, 0, 0, 95, 93,
	0, 0, 65, 65, 0, 188, 58, 59, 244, 126,
	130, 136, 0, 0, 0, 0, 108, 0, 0, 173,
	182, 89, 96, 0, 94, 60, 70, 43, 43, 186,
	187, 139, 141, 142, 145, 143, 122, 0, 0, 0,
	84, 0, 97, 0, 0, 0, 152, 0, 0, 154,
	175, 0, 0, 61, 62, 0, 84, 0, 108, 169,
	174, 77, 158, 84, 84, 161, 166, 0, 122, 155,
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
		//line n1ql.y:342
		{
			yylex.(*lexer).setStatement(yyS[yypt-0].statement)
		}
	case 2:
		//line n1ql.y:347
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
		//line n1ql.y:364
		{
			yyVAL.statement = algebra.NewExplain(yyS[yypt-0].statement)
		}
	case 8:
		//line n1ql.y:371
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
		//line n1ql.y:402
		{
			yyVAL.fullselect = algebra.NewSelect(yyS[yypt-3].subresult, yyS[yypt-2].order, yyS[yypt-0].expr, yyS[yypt-1].expr) /* OFFSET precedes LIMIT */
		}
	case 19:
		//line n1ql.y:408
		{
			yyVAL.subresult = yyS[yypt-0].subselect
		}
	case 20:
		//line n1ql.y:413
		{
			yyVAL.subresult = algebra.NewUnion(yyS[yypt-2].subresult, yyS[yypt-0].subselect)
		}
	case 21:
		//line n1ql.y:418
		{
			yyVAL.subresult = algebra.NewUnionAll(yyS[yypt-3].subresult, yyS[yypt-0].subselect)
		}
	case 22:
		//line n1ql.y:423
		{
			yyVAL.subresult = algebra.NewIntersect(yyS[yypt-2].subresult, yyS[yypt-0].subselect)
		}
	case 23:
		//line n1ql.y:428
		{
			yyVAL.subresult = algebra.NewIntersectAll(yyS[yypt-3].subresult, yyS[yypt-0].subselect)
		}
	case 24:
		//line n1ql.y:433
		{
			yyVAL.subresult = algebra.NewExcept(yyS[yypt-2].subresult, yyS[yypt-0].subselect)
		}
	case 25:
		//line n1ql.y:438
		{
			yyVAL.subresult = algebra.NewExceptAll(yyS[yypt-3].subresult, yyS[yypt-0].subselect)
		}
	case 26:
		yyVAL.subselect = yyS[yypt-0].subselect
	case 27:
		yyVAL.subselect = yyS[yypt-0].subselect
	case 28:
		//line n1ql.y:451
		{
			yyVAL.subselect = algebra.NewSubselect(yyS[yypt-4].fromTerm, yyS[yypt-3].bindings, yyS[yypt-2].expr, yyS[yypt-1].group, yyS[yypt-0].projection)
		}
	case 29:
		//line n1ql.y:458
		{
			yyVAL.subselect = algebra.NewSubselect(yyS[yypt-3].fromTerm, yyS[yypt-2].bindings, yyS[yypt-1].expr, yyS[yypt-0].group, yyS[yypt-4].projection)
		}
	case 30:
		//line n1ql.y:473
		{
			yyVAL.projection = yyS[yypt-0].projection
		}
	case 31:
		//line n1ql.y:480
		{
			yyVAL.projection = algebra.NewProjection(false, yyS[yypt-0].resultTerms)
		}
	case 32:
		//line n1ql.y:485
		{
			yyVAL.projection = algebra.NewProjection(true, yyS[yypt-0].resultTerms)
		}
	case 33:
		//line n1ql.y:490
		{
			yyVAL.projection = algebra.NewProjection(false, yyS[yypt-0].resultTerms)
		}
	case 34:
		//line n1ql.y:495
		{
			yyVAL.projection = algebra.NewRawProjection(false, yyS[yypt-0].expr)
		}
	case 35:
		//line n1ql.y:500
		{
			yyVAL.projection = algebra.NewRawProjection(true, yyS[yypt-0].expr)
		}
	case 38:
		//line n1ql.y:513
		{
			yyVAL.resultTerms = algebra.ResultTerms{yyS[yypt-0].resultTerm}
		}
	case 39:
		//line n1ql.y:518
		{
			yyVAL.resultTerms = append(yyS[yypt-2].resultTerms, yyS[yypt-0].resultTerm)
		}
	case 40:
		//line n1ql.y:525
		{
			yyVAL.resultTerm = algebra.NewResultTerm(nil, true, "")
		}
	case 41:
		//line n1ql.y:530
		{
			yyVAL.resultTerm = algebra.NewResultTerm(yyS[yypt-2].expr, true, "")
		}
	case 42:
		//line n1ql.y:535
		{
			yyVAL.resultTerm = algebra.NewResultTerm(yyS[yypt-1].expr, false, yyS[yypt-0].s)
		}
	case 43:
		//line n1ql.y:542
		{
			yyVAL.s = ""
		}
	case 44:
		yyVAL.s = yyS[yypt-0].s
	case 45:
		yyVAL.s = yyS[yypt-0].s
	case 46:
		//line n1ql.y:553
		{
			yyVAL.s = yyS[yypt-0].s
		}
	case 47:
		yyVAL.s = yyS[yypt-0].s
	case 48:
		//line n1ql.y:571
		{
			yyVAL.fromTerm = nil
		}
	case 49:
		yyVAL.fromTerm = yyS[yypt-0].fromTerm
	case 50:
		//line n1ql.y:580
		{
			yyVAL.fromTerm = yyS[yypt-0].fromTerm
		}
	case 51:
		//line n1ql.y:587
		{
			yyVAL.fromTerm = yyS[yypt-0].keyspaceTerm
		}
	case 52:
		//line n1ql.y:592
		{
			yyVAL.fromTerm = algebra.NewJoin(yyS[yypt-3].fromTerm, yyS[yypt-2].b, yyS[yypt-0].keyspaceTerm)
		}
	case 53:
		//line n1ql.y:597
		{
			yyVAL.fromTerm = algebra.NewNest(yyS[yypt-3].fromTerm, yyS[yypt-2].b, yyS[yypt-0].keyspaceTerm)
		}
	case 54:
		//line n1ql.y:602
		{
			yyVAL.fromTerm = algebra.NewUnnest(yyS[yypt-4].fromTerm, yyS[yypt-3].b, yyS[yypt-1].expr, yyS[yypt-0].s)
		}
	case 57:
		//line n1ql.y:615
		{
			yyVAL.keyspaceTerm = algebra.NewKeyspaceTerm("", yyS[yypt-3].s, yyS[yypt-2].path, yyS[yypt-1].s, yyS[yypt-0].expr)
		}
	case 58:
		//line n1ql.y:620
		{
			yyVAL.keyspaceTerm = algebra.NewKeyspaceTerm(yyS[yypt-5].s, yyS[yypt-3].s, yyS[yypt-2].path, yyS[yypt-1].s, yyS[yypt-0].expr)
		}
	case 59:
		//line n1ql.y:625
		{
			yyVAL.keyspaceTerm = algebra.NewKeyspaceTerm("#system", yyS[yypt-3].s, yyS[yypt-2].path, yyS[yypt-1].s, yyS[yypt-0].expr)
		}
	case 60:
		//line n1ql.y:632
		{
			yyVAL.keyspaceTerm = algebra.NewKeyspaceTerm("", yyS[yypt-3].s, yyS[yypt-2].path, yyS[yypt-1].s, yyS[yypt-0].expr)
		}
	case 61:
		//line n1ql.y:637
		{
			yyVAL.keyspaceTerm = algebra.NewKeyspaceTerm(yyS[yypt-5].s, yyS[yypt-3].s, yyS[yypt-2].path, yyS[yypt-1].s, yyS[yypt-0].expr)
		}
	case 62:
		//line n1ql.y:642
		{
			yyVAL.keyspaceTerm = algebra.NewKeyspaceTerm("#system", yyS[yypt-3].s, yyS[yypt-2].path, yyS[yypt-1].s, yyS[yypt-0].expr)
		}
	case 63:
		yyVAL.s = yyS[yypt-0].s
	case 64:
		yyVAL.s = yyS[yypt-0].s
	case 65:
		//line n1ql.y:657
		{
			yyVAL.path = nil
		}
	case 66:
		//line n1ql.y:662
		{
			yyVAL.path = yyS[yypt-0].path
		}
	case 67:
		//line n1ql.y:669
		{
			yyVAL.expr = nil
		}
	case 68:
		yyVAL.expr = yyS[yypt-0].expr
	case 69:
		//line n1ql.y:678
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 70:
		//line n1ql.y:685
		{
		}
	case 72:
		//line n1ql.y:693
		{
			yyVAL.b = false
		}
	case 73:
		//line n1ql.y:698
		{
			yyVAL.b = false
		}
	case 74:
		//line n1ql.y:703
		{
			yyVAL.b = true
		}
	case 77:
		//line n1ql.y:716
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 78:
		//line n1ql.y:730
		{
			yyVAL.bindings = nil
		}
	case 79:
		yyVAL.bindings = yyS[yypt-0].bindings
	case 80:
		//line n1ql.y:739
		{
			yyVAL.bindings = yyS[yypt-0].bindings
		}
	case 81:
		//line n1ql.y:746
		{
			yyVAL.bindings = expression.Bindings{yyS[yypt-0].binding}
		}
	case 82:
		//line n1ql.y:751
		{
			yyVAL.bindings = append(yyS[yypt-2].bindings, yyS[yypt-0].binding)
		}
	case 83:
		//line n1ql.y:758
		{
			yyVAL.binding = expression.NewBinding(yyS[yypt-2].s, yyS[yypt-0].expr)
		}
	case 84:
		//line n1ql.y:772
		{
			yyVAL.expr = nil
		}
	case 85:
		yyVAL.expr = yyS[yypt-0].expr
	case 86:
		//line n1ql.y:781
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 87:
		//line n1ql.y:795
		{
			yyVAL.group = nil
		}
	case 88:
		yyVAL.group = yyS[yypt-0].group
	case 89:
		//line n1ql.y:804
		{
			yyVAL.group = algebra.NewGroup(yyS[yypt-2].exprs, yyS[yypt-1].bindings, yyS[yypt-0].expr)
		}
	case 90:
		//line n1ql.y:811
		{
			yyVAL.exprs = expression.Expressions{yyS[yypt-0].expr}
		}
	case 91:
		//line n1ql.y:816
		{
			yyVAL.exprs = append(yyS[yypt-2].exprs, yyS[yypt-0].expr)
		}
	case 92:
		//line n1ql.y:823
		{
			yyVAL.bindings = nil
		}
	case 93:
		yyVAL.bindings = yyS[yypt-0].bindings
	case 94:
		//line n1ql.y:832
		{
			yyVAL.bindings = yyS[yypt-0].bindings
		}
	case 95:
		//line n1ql.y:839
		{
			yyVAL.expr = nil
		}
	case 96:
		yyVAL.expr = yyS[yypt-0].expr
	case 97:
		//line n1ql.y:848
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 98:
		//line n1ql.y:862
		{
			yyVAL.order = nil
		}
	case 99:
		yyVAL.order = yyS[yypt-0].order
	case 100:
		//line n1ql.y:871
		{
			yyVAL.order = algebra.NewOrder(yyS[yypt-0].sortTerms)
		}
	case 101:
		//line n1ql.y:878
		{
			yyVAL.sortTerms = algebra.SortTerms{yyS[yypt-0].sortTerm}
		}
	case 102:
		//line n1ql.y:883
		{
			yyVAL.sortTerms = append(yyS[yypt-2].sortTerms, yyS[yypt-0].sortTerm)
		}
	case 103:
		//line n1ql.y:890
		{
			yyVAL.sortTerm = algebra.NewSortTerm(yyS[yypt-1].expr, yyS[yypt-0].b)
		}
	case 104:
		//line n1ql.y:897
		{
			yyVAL.b = false
		}
	case 105:
		yyVAL.b = yyS[yypt-0].b
	case 106:
		//line n1ql.y:906
		{
			yyVAL.b = false
		}
	case 107:
		//line n1ql.y:911
		{
			yyVAL.b = true
		}
	case 108:
		//line n1ql.y:925
		{
			yyVAL.expr = nil
		}
	case 109:
		yyVAL.expr = yyS[yypt-0].expr
	case 110:
		//line n1ql.y:934
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 111:
		//line n1ql.y:948
		{
			yyVAL.expr = nil
		}
	case 112:
		yyVAL.expr = yyS[yypt-0].expr
	case 113:
		//line n1ql.y:957
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 114:
		//line n1ql.y:971
		{
			yyVAL.statement = algebra.NewInsertValues(yyS[yypt-3].keyspaceRef, yyS[yypt-2].expr, yyS[yypt-1].expr, yyS[yypt-0].projection)
		}
	case 115:
		//line n1ql.y:976
		{
			yyVAL.statement = algebra.NewInsertSelect(yyS[yypt-3].keyspaceRef, yyS[yypt-2].expr, yyS[yypt-1].fullselect, yyS[yypt-0].projection)
		}
	case 116:
		//line n1ql.y:983
		{
			yyVAL.keyspaceRef = algebra.NewKeyspaceRef(yyS[yypt-3].s, yyS[yypt-1].s, yyS[yypt-0].s)
		}
	case 117:
		//line n1ql.y:988
		{
			yyVAL.keyspaceRef = algebra.NewKeyspaceRef("", yyS[yypt-1].s, yyS[yypt-0].s)
		}
	case 118:
		//line n1ql.y:995
		{
			yyVAL.expr = nil
		}
	case 119:
		yyVAL.expr = yyS[yypt-0].expr
	case 120:
		//line n1ql.y:1004
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 121:
		//line n1ql.y:1011
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 122:
		//line n1ql.y:1018
		{
			yyVAL.projection = nil
		}
	case 123:
		yyVAL.projection = yyS[yypt-0].projection
	case 124:
		//line n1ql.y:1027
		{
			yyVAL.projection = yyS[yypt-0].projection
		}
	case 125:
		//line n1ql.y:1034
		{
			yyVAL.projection = algebra.NewProjection(false, yyS[yypt-0].resultTerms)
		}
	case 126:
		//line n1ql.y:1039
		{
			yyVAL.projection = algebra.NewRawProjection(false, yyS[yypt-0].expr)
		}
	case 127:
		//line n1ql.y:1053
		{
			yyVAL.statement = algebra.NewUpsertValues(yyS[yypt-3].keyspaceRef, yyS[yypt-2].expr, yyS[yypt-1].expr, yyS[yypt-0].projection)
		}
	case 128:
		//line n1ql.y:1058
		{
			yyVAL.statement = algebra.NewUpsertSelect(yyS[yypt-3].keyspaceRef, yyS[yypt-2].expr, yyS[yypt-1].fullselect, yyS[yypt-0].projection)
		}
	case 129:
		//line n1ql.y:1072
		{
			yyVAL.statement = algebra.NewDelete(yyS[yypt-4].keyspaceRef, yyS[yypt-3].expr, yyS[yypt-2].expr, yyS[yypt-1].expr, yyS[yypt-0].projection)
		}
	case 130:
		//line n1ql.y:1086
		{
			yyVAL.statement = algebra.NewUpdate(yyS[yypt-6].keyspaceRef, yyS[yypt-5].expr, yyS[yypt-4].set, yyS[yypt-3].unset, yyS[yypt-2].expr, yyS[yypt-1].expr, yyS[yypt-0].projection)
		}
	case 131:
		//line n1ql.y:1091
		{
			yyVAL.statement = algebra.NewUpdate(yyS[yypt-5].keyspaceRef, yyS[yypt-4].expr, yyS[yypt-3].set, nil, yyS[yypt-2].expr, yyS[yypt-1].expr, yyS[yypt-0].projection)
		}
	case 132:
		//line n1ql.y:1096
		{
			yyVAL.statement = algebra.NewUpdate(yyS[yypt-5].keyspaceRef, yyS[yypt-4].expr, nil, yyS[yypt-3].unset, yyS[yypt-2].expr, yyS[yypt-1].expr, yyS[yypt-0].projection)
		}
	case 133:
		//line n1ql.y:1103
		{
			yyVAL.set = algebra.NewSet(yyS[yypt-0].setTerms)
		}
	case 134:
		//line n1ql.y:1110
		{
			yyVAL.setTerms = algebra.SetTerms{yyS[yypt-0].setTerm}
		}
	case 135:
		//line n1ql.y:1115
		{
			yyVAL.setTerms = append(yyS[yypt-2].setTerms, yyS[yypt-0].setTerm)
		}
	case 136:
		//line n1ql.y:1122
		{
			yyVAL.setTerm = algebra.NewSetTerm(yyS[yypt-3].path, yyS[yypt-1].expr, yyS[yypt-0].updateFor)
		}
	case 137:
		//line n1ql.y:1129
		{
			yyVAL.updateFor = nil
		}
	case 138:
		yyVAL.updateFor = yyS[yypt-0].updateFor
	case 139:
		//line n1ql.y:1138
		{
			yyVAL.updateFor = algebra.NewUpdateFor(yyS[yypt-2].bindings, yyS[yypt-1].expr)
		}
	case 140:
		//line n1ql.y:1145
		{
			yyVAL.bindings = expression.Bindings{yyS[yypt-0].binding}
		}
	case 141:
		//line n1ql.y:1150
		{
			yyVAL.bindings = append(yyS[yypt-2].bindings, yyS[yypt-0].binding)
		}
	case 142:
		//line n1ql.y:1157
		{
			yyVAL.binding = expression.NewBinding(yyS[yypt-2].s, yyS[yypt-0].expr)
		}
	case 143:
		//line n1ql.y:1162
		{
			yyVAL.binding = expression.NewDescendantBinding(yyS[yypt-2].s, yyS[yypt-0].expr)
		}
	case 144:
		yyVAL.s = yyS[yypt-0].s
	case 145:
		//line n1ql.y:1173
		{
			yyVAL.expr = yyS[yypt-0].path
		}
	case 146:
		//line n1ql.y:1180
		{
			yyVAL.expr = nil
		}
	case 147:
		//line n1ql.y:1185
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 148:
		//line n1ql.y:1192
		{
			yyVAL.unset = algebra.NewUnset(yyS[yypt-0].unsetTerms)
		}
	case 149:
		//line n1ql.y:1199
		{
			yyVAL.unsetTerms = algebra.UnsetTerms{yyS[yypt-0].unsetTerm}
		}
	case 150:
		//line n1ql.y:1204
		{
			yyVAL.unsetTerms = append(yyS[yypt-2].unsetTerms, yyS[yypt-0].unsetTerm)
		}
	case 151:
		//line n1ql.y:1211
		{
			yyVAL.unsetTerm = algebra.NewUnsetTerm(yyS[yypt-1].path, yyS[yypt-0].updateFor)
		}
	case 152:
		//line n1ql.y:1225
		{
			source := algebra.NewMergeSourceFrom(yyS[yypt-5].keyspaceTerm, "")
			yyVAL.statement = algebra.NewMerge(yyS[yypt-7].keyspaceRef, source, yyS[yypt-3].expr, yyS[yypt-2].mergeActions, yyS[yypt-1].expr, yyS[yypt-0].projection)
		}
	case 153:
		//line n1ql.y:1231
		{
			source := algebra.NewMergeSourceSelect(yyS[yypt-7].fullselect, yyS[yypt-5].s)
			yyVAL.statement = algebra.NewMerge(yyS[yypt-10].keyspaceRef, source, yyS[yypt-3].expr, yyS[yypt-2].mergeActions, yyS[yypt-1].expr, yyS[yypt-0].projection)
		}
	case 154:
		//line n1ql.y:1239
		{
			yyVAL.mergeActions = algebra.NewMergeActions(nil, nil, nil)
		}
	case 155:
		//line n1ql.y:1244
		{
			yyVAL.mergeActions = algebra.NewMergeActions(yyS[yypt-1].mergeUpdate, yyS[yypt-0].mergeActions.Delete(), yyS[yypt-0].mergeActions.Insert())
		}
	case 156:
		//line n1ql.y:1249
		{
			yyVAL.mergeActions = algebra.NewMergeActions(nil, yyS[yypt-1].mergeDelete, yyS[yypt-0].mergeInsert)
		}
	case 157:
		//line n1ql.y:1254
		{
			yyVAL.mergeActions = algebra.NewMergeActions(nil, nil, yyS[yypt-0].mergeInsert)
		}
	case 158:
		//line n1ql.y:1261
		{
			yyVAL.mergeActions = algebra.NewMergeActions(nil, nil, nil)
		}
	case 159:
		//line n1ql.y:1266
		{
			yyVAL.mergeActions = algebra.NewMergeActions(nil, yyS[yypt-1].mergeDelete, yyS[yypt-0].mergeInsert)
		}
	case 160:
		//line n1ql.y:1271
		{
			yyVAL.mergeActions = algebra.NewMergeActions(nil, nil, yyS[yypt-0].mergeInsert)
		}
	case 161:
		//line n1ql.y:1278
		{
			yyVAL.mergeInsert = nil
		}
	case 162:
		//line n1ql.y:1283
		{
			yyVAL.mergeInsert = yyS[yypt-0].mergeInsert
		}
	case 163:
		//line n1ql.y:1290
		{
			yyVAL.mergeUpdate = algebra.NewMergeUpdate(yyS[yypt-1].set, nil, yyS[yypt-0].expr)
		}
	case 164:
		//line n1ql.y:1295
		{
			yyVAL.mergeUpdate = algebra.NewMergeUpdate(yyS[yypt-2].set, yyS[yypt-1].unset, yyS[yypt-0].expr)
		}
	case 165:
		//line n1ql.y:1300
		{
			yyVAL.mergeUpdate = algebra.NewMergeUpdate(nil, yyS[yypt-1].unset, yyS[yypt-0].expr)
		}
	case 166:
		//line n1ql.y:1307
		{
			yyVAL.mergeDelete = algebra.NewMergeDelete(yyS[yypt-0].expr)
		}
	case 167:
		//line n1ql.y:1314
		{
			yyVAL.mergeInsert = algebra.NewMergeInsert(yyS[yypt-1].expr, yyS[yypt-0].expr)
		}
	case 168:
		//line n1ql.y:1328
		{
			yyVAL.statement = algebra.NewCreatePrimaryIndex(yyS[yypt-1].keyspaceRef, yyS[yypt-0].indexType)
		}
	case 169:
		//line n1ql.y:1333
		{
			yyVAL.statement = algebra.NewCreateIndex(yyS[yypt-8].s, yyS[yypt-6].keyspaceRef, yyS[yypt-4].exprs, yyS[yypt-2].expr, yyS[yypt-1].expr, yyS[yypt-0].indexType)
		}
	case 170:
		yyVAL.s = yyS[yypt-0].s
	case 171:
		//line n1ql.y:1344
		{
			yyVAL.keyspaceRef = algebra.NewKeyspaceRef("", yyS[yypt-0].s, "")
		}
	case 172:
		//line n1ql.y:1349
		{
			yyVAL.keyspaceRef = algebra.NewKeyspaceRef(yyS[yypt-2].s, yyS[yypt-0].s, "")
		}
	case 173:
		//line n1ql.y:1356
		{
			yyVAL.expr = nil
		}
	case 174:
		//line n1ql.y:1361
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 175:
		//line n1ql.y:1368
		{
			yyVAL.indexType = datastore.VIEW
		}
	case 176:
		//line n1ql.y:1373
		{
			yyVAL.indexType = datastore.VIEW
		}
	case 177:
		//line n1ql.y:1378
		{
			yyVAL.indexType = datastore.LSM
		}
	case 178:
		//line n1ql.y:1392
		{
			yyVAL.statement = algebra.NewDropIndex(yyS[yypt-0].keyspaceRef, "#primary")
		}
	case 179:
		//line n1ql.y:1397
		{
			yyVAL.statement = algebra.NewDropIndex(yyS[yypt-2].keyspaceRef, yyS[yypt-0].s)
		}
	case 180:
		//line n1ql.y:1410
		{
			yyVAL.statement = algebra.NewAlterIndex(yyS[yypt-3].keyspaceRef, yyS[yypt-1].s, yyS[yypt-0].s)
		}
	case 181:
		//line n1ql.y:1416
		{
			yyVAL.s = ""
		}
	case 182:
		//line n1ql.y:1421
		{
			yyVAL.s = yyS[yypt-0].s
		}
	case 183:
		//line n1ql.y:1435
		{
			yyVAL.path = expression.NewIdentifier(yyS[yypt-0].s)
		}
	case 184:
		//line n1ql.y:1440
		{
			yyVAL.path = expression.NewField(yyS[yypt-2].path, expression.NewFieldName(yyS[yypt-0].s))
		}
	case 185:
		//line n1ql.y:1445
		{
			field := expression.NewField(yyS[yypt-2].path, expression.NewFieldName(yyS[yypt-0].s))
			field.SetCaseInsensitive(true)
			yyVAL.path = field
		}
	case 186:
		//line n1ql.y:1452
		{
			yyVAL.path = expression.NewField(yyS[yypt-4].path, yyS[yypt-1].expr)
		}
	case 187:
		//line n1ql.y:1457
		{
			field := expression.NewField(yyS[yypt-4].path, yyS[yypt-1].expr)
			field.SetCaseInsensitive(true)
			yyVAL.path = field
		}
	case 188:
		//line n1ql.y:1464
		{
			yyVAL.path = expression.NewElement(yyS[yypt-3].path, yyS[yypt-1].expr)
		}
	case 189:
		yyVAL.expr = yyS[yypt-0].expr
	case 190:
		//line n1ql.y:1481
		{
			yyVAL.expr = expression.NewField(yyS[yypt-2].expr, expression.NewFieldName(yyS[yypt-0].s))
		}
	case 191:
		//line n1ql.y:1486
		{
			field := expression.NewField(yyS[yypt-2].expr, expression.NewFieldName(yyS[yypt-0].s))
			field.SetCaseInsensitive(true)
			yyVAL.expr = field
		}
	case 192:
		//line n1ql.y:1493
		{
			yyVAL.expr = expression.NewField(yyS[yypt-4].expr, yyS[yypt-1].expr)
		}
	case 193:
		//line n1ql.y:1498
		{
			field := expression.NewField(yyS[yypt-4].expr, yyS[yypt-1].expr)
			field.SetCaseInsensitive(true)
			yyVAL.expr = field
		}
	case 194:
		//line n1ql.y:1505
		{
			yyVAL.expr = expression.NewElement(yyS[yypt-3].expr, yyS[yypt-1].expr)
		}
	case 195:
		//line n1ql.y:1510
		{
			yyVAL.expr = expression.NewSlice(yyS[yypt-4].expr, yyS[yypt-2].expr)
		}
	case 196:
		//line n1ql.y:1515
		{
			yyVAL.expr = expression.NewSlice(yyS[yypt-5].expr, yyS[yypt-3].expr, yyS[yypt-1].expr)
		}
	case 197:
		//line n1ql.y:1521
		{
			yyVAL.expr = expression.NewAdd(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 198:
		//line n1ql.y:1526
		{
			yyVAL.expr = expression.NewSub(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 199:
		//line n1ql.y:1531
		{
			yyVAL.expr = expression.NewMult(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 200:
		//line n1ql.y:1536
		{
			yyVAL.expr = expression.NewDiv(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 201:
		//line n1ql.y:1541
		{
			yyVAL.expr = expression.NewMod(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 202:
		//line n1ql.y:1547
		{
			yyVAL.expr = expression.NewConcat(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 203:
		//line n1ql.y:1553
		{
			yyVAL.expr = expression.NewAnd(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 204:
		//line n1ql.y:1558
		{
			yyVAL.expr = expression.NewOr(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 205:
		//line n1ql.y:1563
		{
			yyVAL.expr = expression.NewNot(yyS[yypt-0].expr)
		}
	case 206:
		//line n1ql.y:1569
		{
			yyVAL.expr = expression.NewEq(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 207:
		//line n1ql.y:1574
		{
			yyVAL.expr = expression.NewEq(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 208:
		//line n1ql.y:1579
		{
			yyVAL.expr = expression.NewNE(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 209:
		//line n1ql.y:1584
		{
			yyVAL.expr = expression.NewLT(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 210:
		//line n1ql.y:1589
		{
			yyVAL.expr = expression.NewGT(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 211:
		//line n1ql.y:1594
		{
			yyVAL.expr = expression.NewLE(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 212:
		//line n1ql.y:1599
		{
			yyVAL.expr = expression.NewGE(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 213:
		//line n1ql.y:1604
		{
			yyVAL.expr = expression.NewBetween(yyS[yypt-4].expr, yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 214:
		//line n1ql.y:1609
		{
			yyVAL.expr = expression.NewNotBetween(yyS[yypt-5].expr, yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 215:
		//line n1ql.y:1614
		{
			yyVAL.expr = expression.NewLike(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 216:
		//line n1ql.y:1619
		{
			yyVAL.expr = expression.NewNotLike(yyS[yypt-3].expr, yyS[yypt-0].expr)
		}
	case 217:
		//line n1ql.y:1624
		{
			yyVAL.expr = expression.NewIn(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 218:
		//line n1ql.y:1629
		{
			yyVAL.expr = expression.NewNotIn(yyS[yypt-3].expr, yyS[yypt-0].expr)
		}
	case 219:
		//line n1ql.y:1634
		{
			yyVAL.expr = expression.NewWithin(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 220:
		//line n1ql.y:1639
		{
			yyVAL.expr = expression.NewNotWithin(yyS[yypt-3].expr, yyS[yypt-0].expr)
		}
	case 221:
		//line n1ql.y:1644
		{
			yyVAL.expr = expression.NewIsNull(yyS[yypt-2].expr)
		}
	case 222:
		//line n1ql.y:1649
		{
			yyVAL.expr = expression.NewIsNotNull(yyS[yypt-3].expr)
		}
	case 223:
		//line n1ql.y:1654
		{
			yyVAL.expr = expression.NewIsMissing(yyS[yypt-2].expr)
		}
	case 224:
		//line n1ql.y:1659
		{
			yyVAL.expr = expression.NewIsNotMissing(yyS[yypt-3].expr)
		}
	case 225:
		//line n1ql.y:1664
		{
			yyVAL.expr = expression.NewIsValued(yyS[yypt-2].expr)
		}
	case 226:
		//line n1ql.y:1669
		{
			yyVAL.expr = expression.NewIsNotValued(yyS[yypt-3].expr)
		}
	case 227:
		//line n1ql.y:1674
		{
			yyVAL.expr = expression.NewExists(yyS[yypt-0].expr)
		}
	case 228:
		yyVAL.expr = yyS[yypt-0].expr
	case 229:
		yyVAL.expr = yyS[yypt-0].expr
	case 230:
		//line n1ql.y:1688
		{
			yyVAL.expr = expression.NewIdentifier(yyS[yypt-0].s)
		}
	case 231:
		yyVAL.expr = yyS[yypt-0].expr
	case 232:
		yyVAL.expr = yyS[yypt-0].expr
	case 233:
		//line n1ql.y:1700
		{
			yyVAL.expr = expression.NewNeg(yyS[yypt-0].expr)
		}
	case 234:
		yyVAL.expr = yyS[yypt-0].expr
	case 235:
		yyVAL.expr = yyS[yypt-0].expr
	case 236:
		yyVAL.expr = yyS[yypt-0].expr
	case 237:
		yyVAL.expr = yyS[yypt-0].expr
	case 238:
		//line n1ql.y:1719
		{
			yyVAL.expr = expression.NewField(yyS[yypt-2].expr, expression.NewFieldName(yyS[yypt-0].s))
		}
	case 239:
		//line n1ql.y:1724
		{
			field := expression.NewField(yyS[yypt-2].expr, expression.NewFieldName(yyS[yypt-0].s))
			field.SetCaseInsensitive(true)
			yyVAL.expr = field
		}
	case 240:
		//line n1ql.y:1731
		{
			yyVAL.expr = expression.NewField(yyS[yypt-4].expr, yyS[yypt-1].expr)
		}
	case 241:
		//line n1ql.y:1736
		{
			field := expression.NewField(yyS[yypt-4].expr, yyS[yypt-1].expr)
			field.SetCaseInsensitive(true)
			yyVAL.expr = field
		}
	case 242:
		//line n1ql.y:1743
		{
			yyVAL.expr = expression.NewElement(yyS[yypt-3].expr, yyS[yypt-1].expr)
		}
	case 243:
		//line n1ql.y:1748
		{
			yyVAL.expr = expression.NewSlice(yyS[yypt-4].expr, yyS[yypt-2].expr)
		}
	case 244:
		//line n1ql.y:1753
		{
			yyVAL.expr = expression.NewSlice(yyS[yypt-5].expr, yyS[yypt-3].expr, yyS[yypt-1].expr)
		}
	case 245:
		//line n1ql.y:1759
		{
			yyVAL.expr = expression.NewAdd(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 246:
		//line n1ql.y:1764
		{
			yyVAL.expr = expression.NewSub(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 247:
		//line n1ql.y:1769
		{
			yyVAL.expr = expression.NewMult(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 248:
		//line n1ql.y:1774
		{
			yyVAL.expr = expression.NewDiv(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 249:
		//line n1ql.y:1779
		{
			yyVAL.expr = expression.NewMod(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 250:
		//line n1ql.y:1785
		{
			yyVAL.expr = expression.NewConcat(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 251:
		//line n1ql.y:1799
		{
			yyVAL.expr = expression.NULL_EXPR
		}
	case 252:
		//line n1ql.y:1804
		{
			yyVAL.expr = expression.FALSE_EXPR
		}
	case 253:
		//line n1ql.y:1809
		{
			yyVAL.expr = expression.TRUE_EXPR
		}
	case 254:
		//line n1ql.y:1814
		{
			yyVAL.expr = expression.NewConstant(value.NewValue(yyS[yypt-0].f))
		}
	case 255:
		//line n1ql.y:1819
		{
			yyVAL.expr = expression.NewConstant(value.NewValue(yyS[yypt-0].n))
		}
	case 256:
		//line n1ql.y:1824
		{
			yyVAL.expr = expression.NewConstant(value.NewValue(yyS[yypt-0].s))
		}
	case 257:
		yyVAL.expr = yyS[yypt-0].expr
	case 258:
		yyVAL.expr = yyS[yypt-0].expr
	case 259:
		//line n1ql.y:1844
		{
			yyVAL.expr = expression.NewObjectConstruct(yyS[yypt-1].bindings)
		}
	case 260:
		//line n1ql.y:1851
		{
			yyVAL.bindings = nil
		}
	case 261:
		yyVAL.bindings = yyS[yypt-0].bindings
	case 262:
		//line n1ql.y:1860
		{
			yyVAL.bindings = expression.Bindings{yyS[yypt-0].binding}
		}
	case 263:
		//line n1ql.y:1865
		{
			yyVAL.bindings = append(yyS[yypt-2].bindings, yyS[yypt-0].binding)
		}
	case 264:
		//line n1ql.y:1872
		{
			yyVAL.binding = expression.NewBinding(yyS[yypt-2].s, yyS[yypt-0].expr)
		}
	case 265:
		//line n1ql.y:1879
		{
			yyVAL.expr = expression.NewArrayConstruct(yyS[yypt-1].exprs...)
		}
	case 266:
		//line n1ql.y:1886
		{
			yyVAL.exprs = nil
		}
	case 267:
		yyVAL.exprs = yyS[yypt-0].exprs
	case 268:
		//line n1ql.y:1902
		{
			yyVAL.expr = algebra.NewNamedParameter(yyS[yypt-0].s)
		}
	case 269:
		//line n1ql.y:1907
		{
			yyVAL.expr = algebra.NewPositionalParameter(yyS[yypt-0].n)
		}
	case 270:
		//line n1ql.y:1921
		{
			yyVAL.expr = yyS[yypt-1].expr
		}
	case 271:
		yyVAL.expr = yyS[yypt-0].expr
	case 272:
		yyVAL.expr = yyS[yypt-0].expr
	case 273:
		//line n1ql.y:1934
		{
			yyVAL.expr = expression.NewSimpleCase(yyS[yypt-2].expr, yyS[yypt-1].whenTerms, yyS[yypt-0].expr)
		}
	case 274:
		//line n1ql.y:1941
		{
			yyVAL.whenTerms = expression.WhenTerms{&expression.WhenTerm{yyS[yypt-2].expr, yyS[yypt-0].expr}}
		}
	case 275:
		//line n1ql.y:1946
		{
			yyVAL.whenTerms = append(yyS[yypt-4].whenTerms, &expression.WhenTerm{yyS[yypt-2].expr, yyS[yypt-0].expr})
		}
	case 276:
		//line n1ql.y:1954
		{
			yyVAL.expr = expression.NewSearchedCase(yyS[yypt-1].whenTerms, yyS[yypt-0].expr)
		}
	case 277:
		//line n1ql.y:1961
		{
			yyVAL.expr = nil
		}
	case 278:
		//line n1ql.y:1966
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 279:
		//line n1ql.y:1980
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
	case 280:
		//line n1ql.y:1999
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
	case 281:
		//line n1ql.y:2014
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
	case 282:
		yyVAL.s = yyS[yypt-0].s
	case 283:
		yyVAL.expr = yyS[yypt-0].expr
	case 284:
		yyVAL.expr = yyS[yypt-0].expr
	case 285:
		//line n1ql.y:2048
		{
			yyVAL.expr = expression.NewAny(yyS[yypt-2].bindings, yyS[yypt-1].expr)
		}
	case 286:
		//line n1ql.y:2053
		{
			yyVAL.expr = expression.NewAny(yyS[yypt-2].bindings, yyS[yypt-1].expr)
		}
	case 287:
		//line n1ql.y:2058
		{
			yyVAL.expr = expression.NewEvery(yyS[yypt-2].bindings, yyS[yypt-1].expr)
		}
	case 288:
		//line n1ql.y:2065
		{
			yyVAL.bindings = expression.Bindings{yyS[yypt-0].binding}
		}
	case 289:
		//line n1ql.y:2070
		{
			yyVAL.bindings = append(yyS[yypt-2].bindings, yyS[yypt-0].binding)
		}
	case 290:
		//line n1ql.y:2077
		{
			yyVAL.binding = expression.NewBinding(yyS[yypt-2].s, yyS[yypt-0].expr)
		}
	case 291:
		//line n1ql.y:2082
		{
			yyVAL.binding = expression.NewDescendantBinding(yyS[yypt-2].s, yyS[yypt-0].expr)
		}
	case 292:
		//line n1ql.y:2089
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 293:
		//line n1ql.y:2096
		{
			yyVAL.expr = expression.NewArray(yyS[yypt-4].expr, yyS[yypt-2].bindings, yyS[yypt-1].expr)
		}
	case 294:
		//line n1ql.y:2101
		{
			yyVAL.expr = expression.NewFirst(yyS[yypt-4].expr, yyS[yypt-2].bindings, yyS[yypt-1].expr)
		}
	case 295:
		//line n1ql.y:2115
		{
			yyVAL.expr = yyS[yypt-1].expr
		}
	case 296:
		yyVAL.expr = yyS[yypt-0].expr
	case 297:
		//line n1ql.y:2124
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
