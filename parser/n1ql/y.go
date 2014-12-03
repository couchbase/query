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
const DECREMENT = 57373
const DELETE = 57374
const DERIVED = 57375
const DESC = 57376
const DESCRIBE = 57377
const DISTINCT = 57378
const DO = 57379
const DROP = 57380
const EACH = 57381
const ELEMENT = 57382
const ELSE = 57383
const END = 57384
const EVERY = 57385
const EXCEPT = 57386
const EXCLUDE = 57387
const EXECUTE = 57388
const EXISTS = 57389
const EXPLAIN = 57390
const FALSE = 57391
const FIRST = 57392
const FLATTEN = 57393
const FOR = 57394
const FROM = 57395
const FUNCTION = 57396
const GRANT = 57397
const GROUP = 57398
const HAVING = 57399
const IF = 57400
const IN = 57401
const INCLUDE = 57402
const INCREMENT = 57403
const INDEX = 57404
const INLINE = 57405
const INNER = 57406
const INSERT = 57407
const INTERSECT = 57408
const INTO = 57409
const IS = 57410
const JOIN = 57411
const KEY = 57412
const KEYS = 57413
const KEYSPACE = 57414
const LAST = 57415
const LEFT = 57416
const LET = 57417
const LETTING = 57418
const LIKE = 57419
const LIMIT = 57420
const LSM = 57421
const MAP = 57422
const MAPPING = 57423
const MATCHED = 57424
const MATERIALIZED = 57425
const MERGE = 57426
const MINUS = 57427
const MISSING = 57428
const NAMESPACE = 57429
const NEST = 57430
const NOT = 57431
const NULL = 57432
const OBJECT = 57433
const OFFSET = 57434
const ON = 57435
const OPTION = 57436
const OR = 57437
const ORDER = 57438
const OUTER = 57439
const OVER = 57440
const PARTITION = 57441
const PASSWORD = 57442
const PATH = 57443
const POOL = 57444
const PREPARE = 57445
const PRIMARY = 57446
const PRIVATE = 57447
const PRIVILEGE = 57448
const PROCEDURE = 57449
const PUBLIC = 57450
const RAW = 57451
const REALM = 57452
const REDUCE = 57453
const RENAME = 57454
const RETURN = 57455
const RETURNING = 57456
const REVOKE = 57457
const RIGHT = 57458
const ROLE = 57459
const ROLLBACK = 57460
const SATISFIES = 57461
const SCHEMA = 57462
const SELECT = 57463
const SELF = 57464
const SET = 57465
const SHOW = 57466
const SOME = 57467
const START = 57468
const STATISTICS = 57469
const SYSTEM = 57470
const THEN = 57471
const TO = 57472
const TRANSACTION = 57473
const TRIGGER = 57474
const TRUE = 57475
const TRUNCATE = 57476
const UNDER = 57477
const UNION = 57478
const UNIQUE = 57479
const UNNEST = 57480
const UNSET = 57481
const UPDATE = 57482
const UPSERT = 57483
const USE = 57484
const USER = 57485
const USING = 57486
const VALUE = 57487
const VALUED = 57488
const VALUES = 57489
const VIEW = 57490
const WHEN = 57491
const WHERE = 57492
const WHILE = 57493
const WITH = 57494
const WITHIN = 57495
const WORK = 57496
const XOR = 57497
const INT = 57498
const NUMBER = 57499
const STRING = 57500
const IDENTIFIER = 57501
const IDENTIFIER_ICASE = 57502
const NAMED_PARAM = 57503
const POSITIONAL_PARAM = 57504
const LPAREN = 57505
const RPAREN = 57506
const LBRACE = 57507
const RBRACE = 57508
const LBRACKET = 57509
const RBRACKET = 57510
const RBRACKET_ICASE = 57511
const COMMA = 57512
const COLON = 57513
const INTERESECT = 57514
const EQ = 57515
const DEQ = 57516
const NE = 57517
const LT = 57518
const GT = 57519
const LE = 57520
const GE = 57521
const CONCAT = 57522
const PLUS = 57523
const STAR = 57524
const DIV = 57525
const MOD = 57526
const UMINUS = 57527
const DOT = 57528

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
	"DECREMENT",
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
	"INCREMENT",
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
	"SELF",
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
	-1, 23,
	163, 290,
	-2, 236,
	-1, 113,
	171, 65,
	-2, 66,
	-1, 150,
	51, 74,
	69, 74,
	88, 74,
	138, 74,
	-2, 52,
	-1, 177,
	173, 0,
	174, 0,
	175, 0,
	-2, 212,
	-1, 178,
	173, 0,
	174, 0,
	175, 0,
	-2, 213,
	-1, 179,
	173, 0,
	174, 0,
	175, 0,
	-2, 214,
	-1, 180,
	176, 0,
	177, 0,
	178, 0,
	179, 0,
	-2, 215,
	-1, 181,
	176, 0,
	177, 0,
	178, 0,
	179, 0,
	-2, 216,
	-1, 182,
	176, 0,
	177, 0,
	178, 0,
	179, 0,
	-2, 217,
	-1, 183,
	176, 0,
	177, 0,
	178, 0,
	179, 0,
	-2, 218,
	-1, 190,
	77, 0,
	-2, 221,
	-1, 191,
	59, 0,
	153, 0,
	-2, 223,
	-1, 192,
	59, 0,
	153, 0,
	-2, 225,
	-1, 286,
	77, 0,
	-2, 222,
	-1, 287,
	59, 0,
	153, 0,
	-2, 224,
	-1, 288,
	59, 0,
	153, 0,
	-2, 226,
}

const yyNprod = 306
const yyPrivate = 57344

var yyTokenNames []string
var yyStates []string

const yyLast = 2699

var yyAct = []int{

	164, 3, 586, 575, 445, 584, 576, 307, 308, 197,
	506, 96, 97, 525, 534, 466, 303, 214, 401, 540,
	311, 215, 139, 499, 262, 459, 418, 231, 100, 345,
	403, 400, 210, 156, 300, 159, 112, 427, 135, 151,
	14, 70, 342, 389, 234, 256, 137, 138, 263, 255,
	160, 132, 226, 119, 56, 216, 123, 9, 382, 329,
	327, 461, 136, 349, 528, 477, 143, 144, 346, 476,
	529, 266, 435, 278, 280, 168, 169, 170, 171, 172,
	173, 174, 175, 176, 177, 178, 179, 180, 181, 182,
	183, 434, 277, 190, 191, 192, 124, 95, 328, 265,
	264, 435, 278, 245, 74, 238, 419, 457, 152, 74,
	141, 142, 419, 348, 76, 240, 136, 281, 282, 283,
	434, 277, 228, 73, 77, 78, 79, 522, 73, 213,
	165, 166, 458, 244, 456, 384, 280, 242, 167, 239,
	241, 372, 373, 502, 319, 184, 276, 317, 185, 374,
	229, 468, 252, 366, 244, 217, 278, 200, 202, 204,
	270, 115, 242, 154, 360, 235, 235, 140, 273, 284,
	279, 281, 282, 283, 237, 277, 435, 165, 166, 479,
	480, 232, 272, 430, 314, 167, 154, 218, 286, 287,
	288, 267, 269, 268, 113, 434, 74, 113, 408, 133,
	295, 227, 523, 257, 254, 246, 111, 301, 186, 80,
	75, 77, 78, 79, 113, 73, 310, 113, 278, 559,
	585, 580, 318, 526, 280, 305, 321, 254, 322, 145,
	467, 284, 279, 281, 282, 283, 316, 277, 504, 212,
	508, 331, 306, 332, 185, 309, 335, 336, 337, 599,
	313, 565, 598, 594, 188, 347, 296, 355, 297, 566,
	298, 310, 555, 205, 447, 350, 470, 195, 72, 364,
	194, 193, 187, 315, 243, 351, 370, 320, 153, 375,
	358, 104, 359, 71, 71, 290, 128, 312, 365, 289,
	548, 330, 334, 120, 352, 383, 524, 340, 341, 247,
	357, 361, 362, 103, 218, 392, 278, 503, 134, 225,
	535, 546, 71, 395, 397, 398, 396, 363, 203, 284,
	279, 281, 282, 283, 411, 277, 463, 196, 127, 404,
	201, 406, 285, 106, 185, 236, 236, 185, 185, 185,
	185, 185, 185, 390, 354, 291, 394, 326, 189, 393,
	425, 72, 72, 152, 432, 415, 407, 417, 126, 564,
	325, 235, 235, 235, 324, 416, 294, 71, 420, 412,
	413, 414, 592, 102, 280, 440, 589, 391, 258, 71,
	72, 596, 438, 590, 421, 301, 436, 437, 428, 428,
	431, 433, 449, 426, 424, 448, 423, 595, 450, 451,
	125, 556, 257, 453, 257, 452, 462, 454, 455, 199,
	76, 465, 344, 224, 544, 147, 248, 249, 561, 444,
	472, 545, 371, 136, 220, 376, 377, 378, 379, 380,
	381, 260, 346, 405, 69, 72, 481, 114, 304, 108,
	185, 261, 110, 487, 107, 602, 601, 72, 577, 464,
	478, 233, 230, 475, 482, 483, 278, 491, 496, 493,
	494, 474, 129, 492, 533, 71, 109, 538, 473, 507,
	279, 281, 282, 283, 471, 277, 339, 404, 338, 333,
	501, 489, 223, 490, 500, 597, 560, 422, 495, 497,
	206, 518, 74, 511, 356, 353, 1, 519, 505, 558,
	469, 547, 572, 579, 510, 149, 75, 77, 78, 79,
	460, 73, 402, 512, 513, 515, 516, 399, 498, 446,
	488, 302, 520, 153, 527, 521, 39, 38, 443, 37,
	507, 236, 236, 236, 550, 543, 530, 536, 537, 20,
	549, 19, 541, 541, 542, 500, 539, 81, 554, 18,
	207, 208, 209, 90, 552, 553, 551, 219, 429, 429,
	507, 570, 571, 557, 17, 16, 15, 562, 563, 8,
	569, 567, 573, 574, 568, 7, 6, 578, 587, 2,
	581, 583, 582, 588, 81, 5, 4, 217, 385, 591,
	90, 386, 98, 99, 593, 292, 293, 198, 299, 93,
	101, 600, 587, 587, 604, 605, 603, 105, 95, 155,
	532, 531, 509, 343, 253, 146, 81, 92, 211, 259,
	148, 150, 90, 67, 68, 76, 31, 122, 30, 91,
	51, 26, 54, 53, 29, 82, 93, 118, 117, 116,
	28, 130, 131, 25, 48, 95, 47, 22, 21, 0,
	0, 0, 0, 0, 92, 0, 0, 0, 0, 0,
	0, 0, 76, 0, 0, 0, 91, 0, 93, 0,
	0, 0, 82, 0, 0, 0, 0, 95, 0, 0,
	0, 0, 0, 0, 0, 0, 92, 0, 0, 0,
	62, 65, 90, 94, 76, 0, 0, 0, 91, 0,
	52, 0, 0, 0, 82, 0, 0, 74, 484, 485,
	0, 0, 0, 83, 84, 85, 86, 87, 88, 89,
	80, 75, 77, 78, 79, 64, 73, 0, 0, 11,
	94, 42, 66, 0, 0, 0, 218, 0, 93, 0,
	0, 0, 81, 0, 74, 0, 387, 95, 90, 0,
	83, 84, 85, 86, 87, 88, 89, 80, 75, 77,
	78, 79, 94, 73, 76, 0, 0, 27, 41, 388,
	0, 10, 40, 0, 0, 0, 74, 441, 0, 0,
	442, 0, 83, 84, 85, 86, 87, 88, 89, 80,
	75, 77, 78, 79, 93, 73, 81, 0, 0, 0,
	0, 0, 90, 95, 24, 0, 0, 63, 0, 0,
	0, 0, 92, 0, 0, 43, 0, 0, 0, 0,
	76, 0, 0, 0, 91, 0, 0, 0, 0, 0,
	82, 0, 94, 0, 0, 0, 0, 0, 45, 44,
	46, 23, 0, 49, 50, 55, 74, 60, 93, 61,
	0, 0, 0, 0, 0, 0, 0, 95, 0, 80,
	75, 77, 78, 79, 163, 73, 92, 0, 0, 0,
	0, 81, 0, 0, 76, 0, 0, 90, 91, 0,
	0, 0, 0, 0, 82, 0, 0, 0, 94, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 74, 0, 0, 0, 0, 0, 83, 84,
	85, 86, 87, 88, 89, 80, 75, 77, 78, 79,
	0, 73, 81, 93, 0, 217, 0, 0, 90, 0,
	0, 0, 95, 0, 0, 0, 0, 0, 0, 0,
	0, 92, 94, 0, 0, 0, 0, 0, 0, 76,
	0, 0, 0, 91, 0, 0, 74, 367, 368, 82,
	0, 0, 83, 84, 85, 86, 87, 88, 89, 80,
	75, 77, 78, 79, 93, 73, 0, 0, 0, 0,
	0, 0, 0, 95, 0, 0, 0, 0, 0, 0,
	0, 0, 92, 0, 0, 0, 0, 81, 0, 0,
	76, 0, 0, 90, 91, 0, 0, 0, 0, 0,
	82, 0, 0, 0, 0, 0, 0, 94, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 74, 274, 0, 0, 275, 0, 83, 84, 85,
	86, 87, 88, 89, 80, 75, 77, 78, 79, 93,
	73, 0, 0, 0, 0, 0, 0, 81, 95, 0,
	0, 0, 0, 90, 0, 0, 0, 92, 94, 0,
	0, 0, 0, 0, 218, 76, 0, 0, 0, 91,
	0, 0, 74, 0, 0, 82, 0, 0, 83, 84,
	85, 86, 87, 88, 89, 80, 75, 77, 78, 79,
	0, 271, 461, 0, 0, 0, 0, 0, 0, 93,
	0, 0, 0, 0, 0, 0, 0, 0, 95, 0,
	0, 0, 0, 81, 0, 0, 0, 92, 0, 90,
	0, 0, 0, 0, 0, 76, 0, 0, 0, 91,
	254, 0, 0, 94, 0, 82, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 74, 0, 0,
	0, 0, 0, 83, 84, 85, 86, 87, 88, 89,
	80, 75, 77, 78, 79, 93, 73, 0, 0, 0,
	0, 0, 0, 81, 95, 0, 0, 0, 0, 90,
	0, 0, 0, 92, 0, 0, 0, 0, 0, 0,
	0, 76, 0, 94, 0, 91, 0, 0, 0, 0,
	0, 82, 0, 0, 0, 0, 0, 74, 0, 0,
	0, 0, 0, 83, 84, 85, 86, 87, 88, 89,
	80, 75, 77, 78, 79, 93, 73, 0, 0, 0,
	0, 0, 0, 0, 95, 0, 0, 0, 0, 81,
	0, 0, 0, 92, 0, 90, 0, 0, 0, 0,
	0, 76, 0, 0, 0, 91, 0, 0, 0, 94,
	0, 82, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 74, 517, 0, 0, 0, 0, 83,
	84, 85, 86, 87, 88, 89, 80, 75, 77, 78,
	79, 93, 73, 0, 0, 0, 0, 0, 0, 81,
	95, 0, 0, 0, 0, 90, 0, 0, 0, 92,
	0, 0, 0, 0, 0, 0, 0, 76, 0, 94,
	0, 91, 0, 0, 0, 0, 0, 82, 0, 0,
	0, 0, 0, 74, 514, 0, 0, 0, 0, 83,
	84, 85, 86, 87, 88, 89, 80, 75, 77, 78,
	79, 93, 73, 0, 0, 0, 0, 0, 0, 0,
	95, 0, 0, 0, 0, 81, 0, 0, 0, 92,
	0, 90, 0, 0, 0, 0, 0, 76, 0, 0,
	0, 91, 0, 0, 0, 94, 0, 82, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 74,
	439, 0, 0, 0, 0, 83, 84, 85, 86, 87,
	88, 89, 80, 75, 77, 78, 79, 93, 73, 0,
	0, 410, 0, 0, 0, 81, 95, 0, 0, 0,
	0, 90, 0, 0, 0, 92, 0, 0, 0, 0,
	0, 0, 0, 76, 0, 94, 0, 91, 0, 0,
	0, 0, 0, 82, 0, 0, 0, 0, 0, 74,
	0, 0, 0, 0, 0, 83, 84, 85, 86, 87,
	88, 89, 80, 75, 77, 78, 79, 93, 73, 0,
	0, 0, 0, 0, 0, 0, 95, 0, 0, 0,
	0, 0, 0, 0, 0, 92, 0, 0, 81, 0,
	0, 0, 0, 76, 90, 0, 0, 91, 0, 0,
	0, 94, 0, 82, 0, 0, 0, 0, 0, 0,
	0, 0, 409, 0, 0, 74, 0, 59, 0, 0,
	0, 83, 84, 85, 86, 87, 88, 89, 80, 75,
	77, 78, 79, 251, 73, 0, 0, 323, 57, 0,
	93, 0, 0, 0, 34, 0, 0, 0, 81, 95,
	58, 0, 0, 0, 90, 0, 0, 0, 92, 0,
	12, 94, 0, 0, 0, 71, 76, 0, 0, 0,
	91, 0, 0, 0, 0, 74, 82, 32, 0, 0,
	0, 83, 84, 85, 86, 87, 88, 89, 80, 75,
	77, 78, 79, 250, 73, 0, 36, 0, 0, 0,
	93, 0, 0, 0, 0, 0, 0, 0, 0, 95,
	0, 0, 0, 0, 81, 13, 0, 0, 92, 0,
	90, 0, 0, 0, 0, 0, 76, 0, 0, 0,
	91, 0, 0, 72, 94, 0, 82, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 74, 0,
	0, 0, 35, 33, 83, 84, 85, 86, 87, 88,
	89, 80, 75, 77, 78, 79, 93, 73, 0, 0,
	0, 0, 0, 0, 0, 95, 0, 0, 0, 0,
	0, 0, 0, 0, 92, 0, 0, 0, 0, 0,
	0, 0, 76, 0, 94, 0, 91, 0, 0, 0,
	0, 0, 82, 0, 0, 0, 0, 0, 74, 0,
	0, 0, 0, 0, 83, 84, 85, 86, 87, 88,
	89, 80, 75, 77, 78, 79, 158, 73, 0, 0,
	62, 65, 0, 0, 0, 0, 0, 0, 0, 0,
	52, 0, 0, 0, 0, 81, 0, 0, 0, 0,
	0, 90, 0, 0, 0, 0, 121, 0, 157, 0,
	94, 0, 162, 0, 0, 64, 0, 0, 0, 11,
	0, 42, 66, 0, 74, 0, 0, 0, 0, 0,
	83, 84, 85, 86, 87, 88, 89, 80, 75, 77,
	78, 79, 0, 73, 0, 0, 0, 93, 0, 0,
	0, 0, 0, 0, 0, 0, 95, 27, 41, 0,
	0, 10, 40, 0, 0, 92, 0, 0, 0, 0,
	0, 0, 0, 76, 0, 62, 65, 91, 0, 0,
	0, 161, 0, 82, 0, 52, 0, 0, 0, 0,
	0, 0, 0, 0, 24, 0, 0, 63, 0, 0,
	0, 0, 0, 0, 0, 43, 0, 162, 0, 0,
	64, 0, 0, 0, 11, 0, 42, 66, 0, 0,
	0, 81, 0, 0, 0, 0, 0, 90, 45, 44,
	46, 23, 0, 49, 50, 55, 0, 60, 0, 61,
	0, 94, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 27, 41, 163, 74, 10, 40, 0, 0,
	0, 83, 84, 85, 86, 87, 88, 89, 80, 75,
	77, 78, 79, 93, 73, 0, 161, 0, 0, 0,
	0, 90, 95, 0, 0, 0, 0, 0, 0, 24,
	0, 92, 63, 0, 0, 0, 0, 0, 0, 76,
	43, 0, 0, 91, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 90, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 45, 44, 46, 23, 93, 49, 50,
	55, 0, 60, 0, 61, 0, 95, 0, 0, 0,
	0, 62, 65, 0, 0, 92, 0, 0, 0, 163,
	0, 52, 0, 76, 0, 0, 0, 91, 0, 93,
	0, 0, 0, 0, 0, 0, 0, 94, 95, 221,
	0, 0, 0, 0, 0, 0, 64, 92, 0, 0,
	11, 74, 42, 66, 0, 76, 0, 83, 84, 85,
	86, 87, 88, 89, 80, 75, 77, 78, 79, 0,
	73, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 27, 41,
	0, 94, 10, 40, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 74, 0, 0, 0, 0,
	0, 83, 84, 85, 86, 87, 88, 89, 80, 75,
	77, 78, 79, 94, 73, 24, 0, 0, 63, 0,
	0, 0, 0, 0, 0, 0, 43, 74, 0, 0,
	0, 0, 0, 0, 0, 0, 86, 87, 88, 89,
	80, 75, 77, 78, 79, 0, 73, 0, 0, 45,
	44, 46, 23, 0, 49, 50, 55, 0, 60, 59,
	61, 0, 62, 65, 0, 0, 0, 0, 0, 90,
	0, 0, 52, 0, 0, 222, 0, 0, 0, 0,
	57, 0, 0, 0, 0, 0, 34, 0, 0, 0,
	0, 0, 58, 0, 0, 0, 0, 64, 0, 0,
	0, 11, 12, 42, 66, 0, 0, 71, 0, 0,
	0, 0, 0, 0, 0, 93, 0, 0, 0, 32,
	0, 0, 0, 0, 95, 0, 0, 0, 0, 0,
	0, 0, 0, 92, 0, 0, 0, 0, 36, 27,
	41, 76, 0, 10, 40, 0, 0, 0, 0, 62,
	65, 0, 0, 0, 0, 0, 0, 13, 0, 52,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 72, 24, 0, 0, 63,
	0, 0, 0, 0, 64, 0, 0, 43, 11, 0,
	42, 66, 0, 0, 35, 33, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 94,
	45, 44, 46, 23, 0, 49, 50, 55, 0, 60,
	0, 61, 0, 74, 0, 0, 27, 41, 0, 0,
	10, 40, 0, 0, 62, 65, 80, 75, 77, 78,
	79, 0, 73, 0, 52, 0, 0, 62, 65, 0,
	0, 0, 0, 0, 0, 0, 0, 52, 0, 0,
	0, 0, 0, 24, 0, 0, 63, 0, 0, 64,
	0, 0, 0, 11, 43, 42, 66, 0, 0, 0,
	0, 0, 64, 0, 0, 0, 11, 0, 42, 66,
	0, 0, 71, 0, 0, 0, 0, 45, 44, 46,
	23, 0, 49, 50, 55, 0, 60, 0, 61, 486,
	0, 27, 41, 0, 0, 10, 40, 0, 0, 0,
	0, 0, 0, 0, 27, 41, 0, 0, 10, 40,
	0, 0, 62, 65, 0, 0, 0, 0, 0, 0,
	0, 0, 52, 0, 0, 0, 0, 0, 24, 0,
	0, 63, 0, 0, 0, 0, 0, 0, 0, 43,
	72, 24, 0, 0, 63, 0, 0, 64, 0, 0,
	0, 11, 43, 42, 66, 0, 0, 0, 0, 0,
	0, 0, 45, 44, 46, 23, 0, 49, 50, 55,
	0, 60, 0, 61, 369, 45, 44, 46, 23, 0,
	49, 50, 55, 0, 60, 0, 61, 0, 0, 27,
	41, 0, 0, 10, 40, 0, 0, 62, 65, 0,
	0, 0, 0, 0, 0, 0, 0, 52, 0, 62,
	65, 0, 0, 0, 0, 0, 0, 0, 0, 52,
	0, 0, 0, 0, 0, 0, 24, 0, 0, 63,
	0, 0, 64, 0, 0, 0, 11, 43, 42, 66,
	0, 0, 0, 0, 64, 0, 0, 0, 0, 0,
	42, 66, 0, 121, 0, 0, 0, 0, 0, 0,
	45, 44, 46, 23, 0, 49, 50, 55, 0, 60,
	0, 61, 0, 0, 27, 41, 0, 0, 10, 40,
	0, 0, 0, 0, 0, 0, 27, 41, 0, 0,
	0, 40, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 24, 0, 0, 63, 0, 0, 0, 0, 0,
	0, 0, 43, 24, 0, 0, 63, 0, 0, 0,
	0, 0, 0, 0, 43, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 45, 44, 46, 23, 0,
	49, 50, 55, 0, 60, 0, 61, 45, 44, 46,
	23, 0, 49, 50, 55, 0, 60, 0, 61,
}
var yyPact = []int{

	2164, -1000, -1000, 1758, -1000, -1000, -1000, -1000, -1000, -1000,
	2519, 2519, 1532, 1532, -1000, -1000, -1000, -1000, -1000, -1000,
	-1000, -1000, -1000, -1000, -1000, -1000, -1000, 2519, -1000, -1000,
	-1000, 237, 377, 372, 413, 38, 370, -1000, -1000, -1000,
	-1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000,
	-1000, -2, 2434, -1000, -1000, 2349, -1000, 296, 224, 400,
	41, 2519, 8, 8, 8, 2519, 2519, -1000, -1000, 340,
	412, 58, 1742, 18, 2519, 2519, 2519, 2519, 2519, 2519,
	2519, 2519, 2519, 2519, 2519, 2519, 2519, 2519, 2519, 2519,
	2531, 195, 2519, 2519, 2519, 181, 1938, 29, -1000, -1000,
	-63, 331, 326, 314, 259, -1000, 474, 38, 38, 38,
	97, -42, 145, -1000, 38, 2003, 440, -1000, -1000, 1627,
	160, 2519, -14, 1758, -1000, 390, 22, 389, 38, 38,
	-61, -31, -1000, -56, -28, -33, 1758, -16, -1000, 146,
	-1000, -16, -16, 1561, 1501, 54, -1000, 28, 340, -1000,
	367, -1000, -138, -71, -72, -1000, -99, 1837, 682, 2519,
	-1000, -1000, -1000, -1000, 915, -1000, -1000, 2519, 864, -58,
	-58, -63, -63, -63, 325, 1938, 1884, 1970, 1970, 1970,
	2166, 2166, 2166, 2166, 139, -1000, 2531, 2519, 2519, 2519,
	679, 29, 29, -1000, 199, -1000, -1000, 274, -1000, 2519,
	-1000, 231, -1000, 231, -1000, 231, 2519, 368, 368, 97,
	122, -1000, 183, 25, -1000, -1000, -1000, 28, -1000, 92,
	-17, 2519, -20, -1000, 160, 2519, -1000, 2519, 1428, -1000,
	271, 267, -1000, 254, -126, -1000, -73, -127, -1000, 41,
	2519, -1000, 2519, 437, 8, 2519, 2519, 2519, 436, 434,
	8, 8, 356, -1000, 2519, -57, -1000, -110, 54, 206,
	-1000, 203, 145, 5, 25, 25, 682, -99, 2519, -99,
	577, -29, -1000, 789, -1000, 2336, 2531, -18, 2519, 2531,
	2531, 2531, 2531, 2531, 2531, 51, 679, 29, 29, -1000,
	-1000, -1000, -1000, -1000, 2519, 1758, -1000, -1000, -1000, -35,
	-1000, 735, 230, -1000, 2519, 230, 54, 77, 54, 5,
	5, 362, -1000, 145, -1000, -1000, 35, -1000, 1368, -1000,
	-1000, 1302, 1758, 2519, 38, 38, 38, 22, 25, 22,
	-1000, 1758, 1758, -1000, -1000, 1758, 1758, 1758, -1000, -1000,
	-37, -37, 147, -1000, 471, -1000, 28, 1758, 28, 2519,
	356, 55, 55, 2519, -1000, -1000, -1000, -1000, 97, -95,
	-1000, -138, -138, -1000, 577, -1000, -1000, -1000, -1000, -1000,
	1242, -11, -1000, -1000, 2519, 609, -65, -65, -94, -94,
	-94, 289, 2531, 1758, 2519, -1000, -1000, -1000, -1000, 150,
	150, 2519, 1758, 150, 150, 331, 54, 331, 331, -36,
	-1000, -66, -38, -1000, 9, 2519, -1000, 233, 231, -1000,
	2519, 1758, 86, -12, -1000, -1000, -1000, 154, 432, 2519,
	426, -1000, 2519, -57, -1000, 1758, -1000, -1000, -138, -102,
	-106, -1000, 577, -1000, 20, 2519, 145, 145, -1000, -1000,
	540, -1000, 2251, -11, -1000, -1000, -1000, 1837, -1000, 1758,
	-1000, -1000, 150, 331, 150, 150, 5, 2519, 5, -1000,
	-1000, 8, 1758, 368, -21, 1758, -1000, 159, 2519, -1000,
	110, -1000, 1758, -1000, -8, 145, 25, 25, -1000, -1000,
	-1000, 1176, 97, 97, -1000, -1000, -1000, 1116, -1000, -99,
	2519, -1000, 150, -1000, -1000, -1000, 1050, -1000, -43, -1000,
	143, 74, 145, -1000, -1000, -100, -1000, 1758, 22, 407,
	-1000, 217, -138, -138, -1000, -1000, -1000, -1000, 1758, -1000,
	-1000, 425, 8, 5, 5, 331, 332, 218, 191, 2519,
	-1000, -1000, -1000, 2519, -1000, 183, 145, 145, -1000, -1000,
	-1000, -95, -1000, 150, 133, 319, 368, 69, 470, -1000,
	1758, 347, 217, 217, -1000, 219, 130, 74, 86, 2519,
	2519, 2519, -1000, -1000, 122, 54, 383, 331, -1000, -1000,
	1758, 1758, 72, 77, 54, 71, -1000, 2519, 150, -1000,
	294, -1000, 54, -1000, -1000, 283, -1000, 990, -1000, 124,
	315, -1000, 299, -1000, 453, 123, 120, 54, 381, 380,
	71, 2519, 2519, -1000, -1000, -1000,
}
var yyPgo = []int{

	0, 648, 647, 646, 644, 643, 51, 642, 641, 0,
	57, 145, 38, 308, 45, 49, 55, 21, 17, 22,
	640, 639, 638, 637, 52, 293, 634, 633, 632, 47,
	46, 274, 26, 631, 630, 628, 627, 40, 626, 54,
	624, 623, 621, 434, 620, 39, 37, 619, 18, 24,
	206, 36, 618, 32, 14, 229, 615, 6, 614, 42,
	613, 612, 29, 611, 610, 50, 33, 609, 41, 607,
	600, 34, 598, 597, 9, 596, 595, 591, 588, 579,
	586, 585, 576, 575, 569, 566, 565, 564, 549, 541,
	539, 529, 527, 526, 442, 43, 16, 521, 520, 519,
	4, 23, 518, 19, 7, 31, 517, 8, 30, 512,
	510, 25, 13, 503, 502, 3, 2, 5, 27, 44,
	501, 15, 500, 10, 499, 498, 496, 35, 495, 20,
	494,
}
var yyR1 = []int{

	0, 126, 126, 79, 79, 79, 79, 79, 80, 81,
	82, 83, 83, 83, 83, 83, 84, 90, 90, 90,
	37, 38, 38, 38, 38, 38, 38, 38, 39, 39,
	41, 40, 68, 67, 67, 67, 67, 67, 127, 127,
	66, 66, 65, 65, 65, 18, 18, 17, 17, 16,
	44, 44, 43, 42, 42, 42, 42, 128, 128, 45,
	45, 45, 46, 46, 46, 50, 51, 49, 49, 53,
	53, 52, 129, 129, 47, 47, 47, 130, 130, 54,
	55, 55, 56, 15, 15, 14, 57, 57, 58, 59,
	59, 60, 60, 12, 12, 61, 61, 62, 63, 63,
	64, 70, 70, 69, 72, 72, 71, 78, 78, 77,
	77, 74, 74, 73, 76, 76, 75, 85, 85, 94,
	94, 97, 97, 96, 95, 100, 100, 99, 98, 98,
	86, 86, 87, 88, 88, 88, 104, 106, 106, 105,
	111, 111, 110, 102, 102, 101, 101, 19, 103, 32,
	32, 107, 109, 109, 108, 89, 89, 112, 112, 112,
	112, 113, 113, 113, 117, 117, 114, 114, 114, 115,
	116, 91, 91, 118, 119, 119, 120, 120, 121, 121,
	121, 125, 125, 123, 124, 124, 92, 92, 93, 122,
	122, 48, 48, 48, 48, 9, 9, 9, 9, 9,
	9, 9, 9, 9, 9, 9, 9, 9, 9, 9,
	9, 9, 9, 9, 9, 9, 9, 9, 9, 9,
	9, 9, 9, 9, 9, 9, 9, 9, 9, 9,
	9, 9, 9, 9, 10, 10, 10, 10, 10, 10,
	10, 10, 10, 10, 11, 11, 11, 11, 11, 11,
	11, 11, 11, 11, 11, 11, 11, 11, 1, 1,
	1, 1, 1, 1, 1, 2, 2, 3, 8, 8,
	7, 7, 6, 4, 13, 13, 5, 5, 20, 21,
	21, 22, 25, 25, 23, 24, 24, 33, 33, 33,
	34, 26, 26, 27, 27, 27, 30, 30, 29, 29,
	31, 28, 28, 35, 36, 36,
}
var yyR2 = []int{

	0, 1, 1, 1, 1, 1, 1, 1, 2, 2,
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
	4, 1, 3, 4, 3, 4, 3, 4, 1, 1,
	5, 5, 2, 1, 2, 2, 3, 4, 1, 1,
	1, 3, 1, 3, 2, 0, 1, 1, 2, 1,
	0, 1, 2, 1, 4, 4, 5, 1, 1, 4,
	6, 6, 4, 6, 6, 1, 1, 0, 2, 0,
	1, 4, 0, 1, 0, 1, 2, 0, 1, 4,
	0, 1, 2, 1, 3, 3, 0, 1, 2, 0,
	1, 5, 1, 1, 3, 0, 1, 2, 0, 1,
	2, 0, 1, 3, 1, 3, 2, 0, 1, 1,
	1, 0, 1, 2, 0, 1, 2, 6, 6, 4,
	2, 0, 1, 2, 2, 0, 1, 2, 1, 2,
	6, 6, 7, 8, 7, 7, 2, 1, 3, 4,
	0, 1, 4, 1, 3, 3, 3, 1, 1, 0,
	2, 2, 1, 3, 2, 10, 13, 0, 6, 6,
	6, 0, 6, 6, 0, 6, 2, 3, 2, 1,
	2, 6, 11, 1, 1, 3, 0, 3, 0, 2,
	2, 1, 3, 1, 0, 2, 5, 5, 6, 0,
	3, 1, 3, 3, 4, 1, 3, 3, 5, 5,
	4, 5, 6, 3, 3, 3, 3, 3, 3, 3,
	3, 2, 3, 3, 3, 3, 3, 3, 3, 5,
	6, 3, 4, 3, 4, 3, 4, 3, 4, 3,
	4, 3, 4, 2, 1, 1, 1, 1, 1, 1,
	2, 1, 1, 1, 1, 3, 3, 5, 5, 4,
	5, 6, 3, 3, 3, 3, 3, 3, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 3, 0, 1,
	1, 3, 3, 3, 0, 1, 1, 1, 3, 1,
	1, 3, 4, 5, 2, 0, 2, 4, 5, 4,
	1, 1, 1, 4, 4, 4, 1, 3, 3, 3,
	2, 6, 6, 3, 1, 1,
}
var yyChk = []int{

	-1000, -126, -79, -9, -80, -81, -82, -83, -84, -10,
	89, 47, 48, 103, -37, -85, -86, -87, -88, -89,
	-90, -1, -2, 159, 122, -5, -33, 85, -20, -26,
	-35, -38, 65, 141, 32, 140, 84, -91, -92, -93,
	90, 86, 49, 133, 157, 156, 158, -3, -4, 161,
	162, -34, 18, -27, -28, 163, -39, 26, 38, 5,
	165, 167, 8, 125, 43, 9, 50, -41, -40, -43,
	-68, 53, 121, 186, 167, 181, 85, 182, 183, 184,
	180, 7, 95, 173, 174, 175, 176, 177, 178, 179,
	13, 89, 77, 59, 153, 68, -9, -9, -79, -79,
	-9, -70, 136, 66, 44, -69, 96, 67, 67, 53,
	-94, -50, -51, 159, 67, 163, -21, -22, -23, -9,
	-25, 149, -36, -9, -37, 104, 62, 104, 62, 62,
	-8, -7, -6, 158, -13, -12, -9, -30, -29, -19,
	159, -30, -30, -9, -9, -55, -56, 75, -44, -43,
	-42, -45, -51, -50, 128, -67, -66, 36, 4, -127,
	-65, 109, 40, 182, -9, 159, 160, 167, -9, -9,
	-9, -9, -9, -9, -9, -9, -9, -9, -9, -9,
	-9, -9, -9, -9, -11, -10, 13, 77, 59, 153,
	-9, -9, -9, 90, 89, 86, 146, -74, -73, 78,
	-39, 4, -39, 4, -39, 4, 16, -94, -94, -94,
	-53, -52, 142, 171, -18, -17, -16, 10, 159, -94,
	-13, 36, 182, 42, -25, 149, -24, 41, -9, 164,
	62, -118, 159, 62, -119, -51, -50, -119, 166, 170,
	171, 168, 170, -31, 170, 119, 59, 153, -31, -31,
	52, 52, -57, -58, 150, -15, -14, -16, -55, -47,
	64, 74, -49, 186, 171, 171, 170, -66, -127, -66,
	-9, 186, -18, -9, 168, 171, 7, 186, 167, 181,
	85, 182, 183, 184, 180, -11, -9, -9, -9, 90,
	86, 146, -76, -75, 92, -9, -39, -39, -39, -72,
	-71, -9, -97, -96, 70, -96, -53, -104, -107, 123,
	139, -129, 104, -51, 159, -16, 144, 164, -9, 164,
	-24, -9, -9, 129, 93, 93, 93, 186, 171, 186,
	-6, -9, -9, 42, -29, -9, -9, -9, 42, 42,
	-30, -30, -59, -60, 56, -62, 76, -9, 170, 173,
	-57, 69, 88, -128, 138, 51, -130, 97, -18, -48,
	159, -51, -51, -65, -9, -18, 182, 168, 169, 168,
	-9, -11, 159, 160, 167, -9, -11, -11, -11, -11,
	-11, -11, 7, -9, 170, -78, -77, 11, 34, -95,
	-37, 147, -9, -95, -37, -57, -107, -57, -57, -106,
	-105, -48, -109, -108, -48, 71, -18, -45, 163, 164,
	129, -9, -119, -119, -119, -118, -51, -118, -32, 149,
	-32, -68, 16, -15, -14, -9, -59, -46, -51, -50,
	128, -46, -9, -53, 186, 167, -49, -49, -18, 168,
	-9, 168, 171, -11, -71, -100, -99, 114, -100, -9,
	-100, -100, -74, -57, -74, -74, 170, 173, 170, -111,
	-110, 52, -9, 93, -37, -9, -121, 144, 163, -122,
	112, 42, -9, 42, -12, -49, 171, 171, -18, 159,
	160, -9, -18, -18, 168, 169, 168, -9, -98, -66,
	-127, -100, -74, -100, -100, -105, -9, -108, -102, -101,
	-19, -96, 164, 148, 79, -125, -123, -9, 130, -61,
	-62, -18, -51, -51, 168, -53, -53, 168, -9, -100,
	-111, -32, 170, 59, 153, -112, 149, -17, 164, 170,
	-118, -63, -64, 57, -54, 93, -49, -49, 42, -101,
	-103, -48, -103, -74, 82, 89, 93, -120, 99, -123,
	-9, -129, -18, -18, -100, 129, 82, -96, -124, 150,
	16, 71, -54, -54, 140, 32, 129, -112, -121, -123,
	-9, -9, -114, -104, -107, -115, -57, 65, -74, -113,
	149, -57, -107, -57, -117, 149, -116, -9, -100, 82,
	89, -57, 89, -57, 129, 82, 82, 32, 129, 129,
	-115, 65, 65, -117, -116, -116,
}
var yyDef = []int{

	0, -2, 1, 2, 3, 4, 5, 6, 7, 195,
	0, 0, 0, 0, 10, 11, 12, 13, 14, 15,
	16, 234, 235, -2, 237, 238, 239, 0, 241, 242,
	243, 101, 0, 0, 0, 0, 0, 17, 18, 19,
	258, 259, 260, 261, 262, 263, 264, 265, 266, 276,
	277, 0, 0, 291, 292, 0, 21, 0, 0, 0,
	268, 274, 0, 0, 0, 0, 0, 28, 29, 80,
	50, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 211, 233, 8, 9,
	240, 111, 0, 0, 0, 102, 0, 0, 0, 0,
	69, 0, 45, -2, 0, 274, 0, 279, 280, 0,
	285, 0, 0, 304, 305, 0, 0, 0, 0, 0,
	0, 269, 270, 0, 0, 275, 93, 0, 296, 0,
	147, 0, 0, 0, 0, 86, 81, 0, 80, 51,
	-2, 53, 67, 0, 0, 32, 33, 0, 0, 0,
	40, 38, 39, 42, 45, 196, 197, 0, 0, 203,
	204, 205, 206, 207, 208, 209, 210, -2, -2, -2,
	-2, -2, -2, -2, 0, 244, 0, 0, 0, 0,
	-2, -2, -2, 227, 0, 229, 231, 114, 112, 0,
	22, 0, 24, 0, 26, 0, 0, 121, 0, 69,
	0, 70, 72, 0, 120, 46, 47, 0, 49, 0,
	0, 0, 0, 278, 285, 0, 284, 0, 0, 303,
	0, 0, 173, 0, 0, 174, 0, 0, 267, 0,
	0, 273, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 89, 87, 0, 82, 83, 0, 86, 0,
	75, 77, 45, 0, 0, 0, 0, 34, 0, 35,
	45, 0, 44, 0, 200, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, -2, -2, -2, 228,
	230, 232, 20, 115, 0, 113, 23, 25, 27, 103,
	104, 107, 0, 122, 0, 0, 86, 86, 86, 0,
	0, 0, 73, 45, 66, 48, 0, 287, 0, 289,
	281, 0, 286, 0, 0, 0, 0, 0, 0, 0,
	271, 272, 94, 293, 297, 300, 298, 299, 294, 295,
	149, 149, 0, 90, 0, 92, 0, 88, 0, 0,
	89, 0, 0, 0, 57, 58, 76, 78, 69, 68,
	191, 67, 67, 41, 45, 36, 43, 198, 199, 201,
	0, 219, 245, 246, 0, 0, 252, 253, 254, 255,
	256, 257, 0, 116, 0, 106, 108, 109, 110, 125,
	125, 0, 123, 125, 125, 111, 86, 111, 111, 136,
	137, 0, 151, 152, 140, 0, 119, 0, 0, 288,
	0, 282, 178, 0, 186, 187, 175, 189, 0, 0,
	0, 30, 0, 97, 84, 85, 31, 54, 67, 0,
	0, 55, 45, 59, 0, 0, 45, 45, 37, 202,
	0, 249, 0, 220, 105, 117, 126, 0, 118, 124,
	130, 131, 125, 111, 125, 125, 0, 0, 0, 154,
	141, 0, 71, 0, 0, 283, 171, 0, 0, 188,
	0, 301, 150, 302, 95, 45, 0, 0, 56, 192,
	193, 0, 69, 69, 247, 248, 250, 0, 127, 128,
	0, 132, 125, 134, 135, 138, 140, 153, 149, 143,
	0, 157, 0, 179, 180, 0, 181, 183, 0, 98,
	96, 0, 67, 67, 194, 60, 61, 251, 129, 133,
	139, 0, 0, 0, 0, 111, 0, 0, 176, 0,
	190, 91, 99, 0, 62, 72, 45, 45, 142, 144,
	145, 148, 146, 125, 0, 0, 0, 184, 0, 182,
	100, 0, 0, 0, 155, 0, 0, 157, 178, 0,
	0, 0, 63, 64, 0, 86, 0, 111, 172, 185,
	177, 79, 161, 86, 86, 164, 169, 0, 125, 158,
	0, 166, 86, 168, 159, 0, 160, 86, 156, 0,
	0, 167, 0, 170, 0, 0, 0, 86, 0, 0,
	164, 0, 0, 162, 163, 165,
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
	182, 183, 184, 185, 186,
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
		//line n1ql.y:347
		{
			yylex.(*lexer).setStatement(yyS[yypt-0].statement)
		}
	case 2:
		//line n1ql.y:352
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
		yyVAL.statement = yyS[yypt-0].statement
	case 8:
		//line n1ql.y:371
		{
			yyVAL.statement = algebra.NewExplain(yyS[yypt-0].statement)
		}
	case 9:
		//line n1ql.y:378
		{
			yyVAL.statement = algebra.NewPrepare(yyS[yypt-0].statement)
		}
	case 10:
		//line n1ql.y:385
		{
			yyVAL.statement = yyS[yypt-0].fullselect
		}
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
		yyVAL.statement = yyS[yypt-0].statement
	case 19:
		yyVAL.statement = yyS[yypt-0].statement
	case 20:
		//line n1ql.y:416
		{
			yyVAL.fullselect = algebra.NewSelect(yyS[yypt-3].subresult, yyS[yypt-2].order, yyS[yypt-0].expr, yyS[yypt-1].expr) /* OFFSET precedes LIMIT */
		}
	case 21:
		//line n1ql.y:422
		{
			yyVAL.subresult = yyS[yypt-0].subselect
		}
	case 22:
		//line n1ql.y:427
		{
			yyVAL.subresult = algebra.NewUnion(yyS[yypt-2].subresult, yyS[yypt-0].subselect)
		}
	case 23:
		//line n1ql.y:432
		{
			yyVAL.subresult = algebra.NewUnionAll(yyS[yypt-3].subresult, yyS[yypt-0].subselect)
		}
	case 24:
		//line n1ql.y:437
		{
			yyVAL.subresult = algebra.NewIntersect(yyS[yypt-2].subresult, yyS[yypt-0].subselect)
		}
	case 25:
		//line n1ql.y:442
		{
			yyVAL.subresult = algebra.NewIntersectAll(yyS[yypt-3].subresult, yyS[yypt-0].subselect)
		}
	case 26:
		//line n1ql.y:447
		{
			yyVAL.subresult = algebra.NewExcept(yyS[yypt-2].subresult, yyS[yypt-0].subselect)
		}
	case 27:
		//line n1ql.y:452
		{
			yyVAL.subresult = algebra.NewExceptAll(yyS[yypt-3].subresult, yyS[yypt-0].subselect)
		}
	case 28:
		yyVAL.subselect = yyS[yypt-0].subselect
	case 29:
		yyVAL.subselect = yyS[yypt-0].subselect
	case 30:
		//line n1ql.y:465
		{
			yyVAL.subselect = algebra.NewSubselect(yyS[yypt-4].fromTerm, yyS[yypt-3].bindings, yyS[yypt-2].expr, yyS[yypt-1].group, yyS[yypt-0].projection)
		}
	case 31:
		//line n1ql.y:472
		{
			yyVAL.subselect = algebra.NewSubselect(yyS[yypt-3].fromTerm, yyS[yypt-2].bindings, yyS[yypt-1].expr, yyS[yypt-0].group, yyS[yypt-4].projection)
		}
	case 32:
		//line n1ql.y:487
		{
			yyVAL.projection = yyS[yypt-0].projection
		}
	case 33:
		//line n1ql.y:494
		{
			yyVAL.projection = algebra.NewProjection(false, yyS[yypt-0].resultTerms)
		}
	case 34:
		//line n1ql.y:499
		{
			yyVAL.projection = algebra.NewProjection(true, yyS[yypt-0].resultTerms)
		}
	case 35:
		//line n1ql.y:504
		{
			yyVAL.projection = algebra.NewProjection(false, yyS[yypt-0].resultTerms)
		}
	case 36:
		//line n1ql.y:509
		{
			yyVAL.projection = algebra.NewRawProjection(false, yyS[yypt-1].expr, yyS[yypt-0].s)
		}
	case 37:
		//line n1ql.y:514
		{
			yyVAL.projection = algebra.NewRawProjection(true, yyS[yypt-1].expr, yyS[yypt-0].s)
		}
	case 40:
		//line n1ql.y:527
		{
			yyVAL.resultTerms = algebra.ResultTerms{yyS[yypt-0].resultTerm}
		}
	case 41:
		//line n1ql.y:532
		{
			yyVAL.resultTerms = append(yyS[yypt-2].resultTerms, yyS[yypt-0].resultTerm)
		}
	case 42:
		//line n1ql.y:539
		{
			yyVAL.resultTerm = algebra.NewResultTerm(nil, true, "")
		}
	case 43:
		//line n1ql.y:544
		{
			yyVAL.resultTerm = algebra.NewResultTerm(yyS[yypt-2].expr, true, "")
		}
	case 44:
		//line n1ql.y:549
		{
			yyVAL.resultTerm = algebra.NewResultTerm(yyS[yypt-1].expr, false, yyS[yypt-0].s)
		}
	case 45:
		//line n1ql.y:556
		{
			yyVAL.s = ""
		}
	case 46:
		yyVAL.s = yyS[yypt-0].s
	case 47:
		yyVAL.s = yyS[yypt-0].s
	case 48:
		//line n1ql.y:567
		{
			yyVAL.s = yyS[yypt-0].s
		}
	case 49:
		yyVAL.s = yyS[yypt-0].s
	case 50:
		//line n1ql.y:585
		{
			yyVAL.fromTerm = nil
		}
	case 51:
		yyVAL.fromTerm = yyS[yypt-0].fromTerm
	case 52:
		//line n1ql.y:594
		{
			yyVAL.fromTerm = yyS[yypt-0].fromTerm
		}
	case 53:
		//line n1ql.y:601
		{
			yyVAL.fromTerm = yyS[yypt-0].keyspaceTerm
		}
	case 54:
		//line n1ql.y:606
		{
			yyVAL.fromTerm = algebra.NewJoin(yyS[yypt-3].fromTerm, yyS[yypt-2].b, yyS[yypt-0].keyspaceTerm)
		}
	case 55:
		//line n1ql.y:611
		{
			yyVAL.fromTerm = algebra.NewNest(yyS[yypt-3].fromTerm, yyS[yypt-2].b, yyS[yypt-0].keyspaceTerm)
		}
	case 56:
		//line n1ql.y:616
		{
			yyVAL.fromTerm = algebra.NewUnnest(yyS[yypt-4].fromTerm, yyS[yypt-3].b, yyS[yypt-1].expr, yyS[yypt-0].s)
		}
	case 59:
		//line n1ql.y:629
		{
			yyVAL.keyspaceTerm = algebra.NewKeyspaceTerm("", yyS[yypt-3].s, yyS[yypt-2].path, yyS[yypt-1].s, yyS[yypt-0].expr)
		}
	case 60:
		//line n1ql.y:634
		{
			yyVAL.keyspaceTerm = algebra.NewKeyspaceTerm(yyS[yypt-5].s, yyS[yypt-3].s, yyS[yypt-2].path, yyS[yypt-1].s, yyS[yypt-0].expr)
		}
	case 61:
		//line n1ql.y:639
		{
			yyVAL.keyspaceTerm = algebra.NewKeyspaceTerm("#system", yyS[yypt-3].s, yyS[yypt-2].path, yyS[yypt-1].s, yyS[yypt-0].expr)
		}
	case 62:
		//line n1ql.y:646
		{
			yyVAL.keyspaceTerm = algebra.NewKeyspaceTerm("", yyS[yypt-3].s, yyS[yypt-2].path, yyS[yypt-1].s, yyS[yypt-0].expr)
		}
	case 63:
		//line n1ql.y:651
		{
			yyVAL.keyspaceTerm = algebra.NewKeyspaceTerm(yyS[yypt-5].s, yyS[yypt-3].s, yyS[yypt-2].path, yyS[yypt-1].s, yyS[yypt-0].expr)
		}
	case 64:
		//line n1ql.y:656
		{
			yyVAL.keyspaceTerm = algebra.NewKeyspaceTerm("#system", yyS[yypt-3].s, yyS[yypt-2].path, yyS[yypt-1].s, yyS[yypt-0].expr)
		}
	case 65:
		yyVAL.s = yyS[yypt-0].s
	case 66:
		yyVAL.s = yyS[yypt-0].s
	case 67:
		//line n1ql.y:671
		{
			yyVAL.path = nil
		}
	case 68:
		//line n1ql.y:676
		{
			yyVAL.path = yyS[yypt-0].path
		}
	case 69:
		//line n1ql.y:683
		{
			yyVAL.expr = nil
		}
	case 70:
		yyVAL.expr = yyS[yypt-0].expr
	case 71:
		//line n1ql.y:692
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 72:
		//line n1ql.y:699
		{
		}
	case 74:
		//line n1ql.y:707
		{
			yyVAL.b = false
		}
	case 75:
		//line n1ql.y:712
		{
			yyVAL.b = false
		}
	case 76:
		//line n1ql.y:717
		{
			yyVAL.b = true
		}
	case 79:
		//line n1ql.y:730
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 80:
		//line n1ql.y:744
		{
			yyVAL.bindings = nil
		}
	case 81:
		yyVAL.bindings = yyS[yypt-0].bindings
	case 82:
		//line n1ql.y:753
		{
			yyVAL.bindings = yyS[yypt-0].bindings
		}
	case 83:
		//line n1ql.y:760
		{
			yyVAL.bindings = expression.Bindings{yyS[yypt-0].binding}
		}
	case 84:
		//line n1ql.y:765
		{
			yyVAL.bindings = append(yyS[yypt-2].bindings, yyS[yypt-0].binding)
		}
	case 85:
		//line n1ql.y:772
		{
			yyVAL.binding = expression.NewBinding(yyS[yypt-2].s, yyS[yypt-0].expr)
		}
	case 86:
		//line n1ql.y:786
		{
			yyVAL.expr = nil
		}
	case 87:
		yyVAL.expr = yyS[yypt-0].expr
	case 88:
		//line n1ql.y:795
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 89:
		//line n1ql.y:809
		{
			yyVAL.group = nil
		}
	case 90:
		yyVAL.group = yyS[yypt-0].group
	case 91:
		//line n1ql.y:818
		{
			yyVAL.group = algebra.NewGroup(yyS[yypt-2].exprs, yyS[yypt-1].bindings, yyS[yypt-0].expr)
		}
	case 92:
		//line n1ql.y:823
		{
			yyVAL.group = algebra.NewGroup(nil, yyS[yypt-0].bindings, nil)
		}
	case 93:
		//line n1ql.y:830
		{
			yyVAL.exprs = expression.Expressions{yyS[yypt-0].expr}
		}
	case 94:
		//line n1ql.y:835
		{
			yyVAL.exprs = append(yyS[yypt-2].exprs, yyS[yypt-0].expr)
		}
	case 95:
		//line n1ql.y:842
		{
			yyVAL.bindings = nil
		}
	case 96:
		yyVAL.bindings = yyS[yypt-0].bindings
	case 97:
		//line n1ql.y:851
		{
			yyVAL.bindings = yyS[yypt-0].bindings
		}
	case 98:
		//line n1ql.y:858
		{
			yyVAL.expr = nil
		}
	case 99:
		yyVAL.expr = yyS[yypt-0].expr
	case 100:
		//line n1ql.y:867
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 101:
		//line n1ql.y:881
		{
			yyVAL.order = nil
		}
	case 102:
		yyVAL.order = yyS[yypt-0].order
	case 103:
		//line n1ql.y:890
		{
			yyVAL.order = algebra.NewOrder(yyS[yypt-0].sortTerms)
		}
	case 104:
		//line n1ql.y:897
		{
			yyVAL.sortTerms = algebra.SortTerms{yyS[yypt-0].sortTerm}
		}
	case 105:
		//line n1ql.y:902
		{
			yyVAL.sortTerms = append(yyS[yypt-2].sortTerms, yyS[yypt-0].sortTerm)
		}
	case 106:
		//line n1ql.y:909
		{
			yyVAL.sortTerm = algebra.NewSortTerm(yyS[yypt-1].expr, yyS[yypt-0].b)
		}
	case 107:
		//line n1ql.y:916
		{
			yyVAL.b = false
		}
	case 108:
		yyVAL.b = yyS[yypt-0].b
	case 109:
		//line n1ql.y:925
		{
			yyVAL.b = false
		}
	case 110:
		//line n1ql.y:930
		{
			yyVAL.b = true
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
			yyVAL.expr = nil
		}
	case 115:
		yyVAL.expr = yyS[yypt-0].expr
	case 116:
		//line n1ql.y:976
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 117:
		//line n1ql.y:990
		{
			yyVAL.statement = algebra.NewInsertValues(yyS[yypt-3].keyspaceRef, yyS[yypt-2].expr, yyS[yypt-1].expr, yyS[yypt-0].projection)
		}
	case 118:
		//line n1ql.y:995
		{
			yyVAL.statement = algebra.NewInsertSelect(yyS[yypt-3].keyspaceRef, yyS[yypt-2].expr, yyS[yypt-1].fullselect, yyS[yypt-0].projection)
		}
	case 119:
		//line n1ql.y:1002
		{
			yyVAL.keyspaceRef = algebra.NewKeyspaceRef(yyS[yypt-3].s, yyS[yypt-1].s, yyS[yypt-0].s)
		}
	case 120:
		//line n1ql.y:1007
		{
			yyVAL.keyspaceRef = algebra.NewKeyspaceRef("", yyS[yypt-1].s, yyS[yypt-0].s)
		}
	case 121:
		//line n1ql.y:1014
		{
			yyVAL.expr = nil
		}
	case 122:
		yyVAL.expr = yyS[yypt-0].expr
	case 123:
		//line n1ql.y:1023
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 124:
		//line n1ql.y:1030
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 125:
		//line n1ql.y:1037
		{
			yyVAL.projection = nil
		}
	case 126:
		yyVAL.projection = yyS[yypt-0].projection
	case 127:
		//line n1ql.y:1046
		{
			yyVAL.projection = yyS[yypt-0].projection
		}
	case 128:
		//line n1ql.y:1053
		{
			yyVAL.projection = algebra.NewProjection(false, yyS[yypt-0].resultTerms)
		}
	case 129:
		//line n1ql.y:1058
		{
			yyVAL.projection = algebra.NewRawProjection(false, yyS[yypt-0].expr, "")
		}
	case 130:
		//line n1ql.y:1072
		{
			yyVAL.statement = algebra.NewUpsertValues(yyS[yypt-3].keyspaceRef, yyS[yypt-2].expr, yyS[yypt-1].expr, yyS[yypt-0].projection)
		}
	case 131:
		//line n1ql.y:1077
		{
			yyVAL.statement = algebra.NewUpsertSelect(yyS[yypt-3].keyspaceRef, yyS[yypt-2].expr, yyS[yypt-1].fullselect, yyS[yypt-0].projection)
		}
	case 132:
		//line n1ql.y:1091
		{
			yyVAL.statement = algebra.NewDelete(yyS[yypt-4].keyspaceRef, yyS[yypt-3].expr, yyS[yypt-2].expr, yyS[yypt-1].expr, yyS[yypt-0].projection)
		}
	case 133:
		//line n1ql.y:1105
		{
			yyVAL.statement = algebra.NewUpdate(yyS[yypt-6].keyspaceRef, yyS[yypt-5].expr, yyS[yypt-4].set, yyS[yypt-3].unset, yyS[yypt-2].expr, yyS[yypt-1].expr, yyS[yypt-0].projection)
		}
	case 134:
		//line n1ql.y:1110
		{
			yyVAL.statement = algebra.NewUpdate(yyS[yypt-5].keyspaceRef, yyS[yypt-4].expr, yyS[yypt-3].set, nil, yyS[yypt-2].expr, yyS[yypt-1].expr, yyS[yypt-0].projection)
		}
	case 135:
		//line n1ql.y:1115
		{
			yyVAL.statement = algebra.NewUpdate(yyS[yypt-5].keyspaceRef, yyS[yypt-4].expr, nil, yyS[yypt-3].unset, yyS[yypt-2].expr, yyS[yypt-1].expr, yyS[yypt-0].projection)
		}
	case 136:
		//line n1ql.y:1122
		{
			yyVAL.set = algebra.NewSet(yyS[yypt-0].setTerms)
		}
	case 137:
		//line n1ql.y:1129
		{
			yyVAL.setTerms = algebra.SetTerms{yyS[yypt-0].setTerm}
		}
	case 138:
		//line n1ql.y:1134
		{
			yyVAL.setTerms = append(yyS[yypt-2].setTerms, yyS[yypt-0].setTerm)
		}
	case 139:
		//line n1ql.y:1141
		{
			yyVAL.setTerm = algebra.NewSetTerm(yyS[yypt-3].path, yyS[yypt-1].expr, yyS[yypt-0].updateFor)
		}
	case 140:
		//line n1ql.y:1148
		{
			yyVAL.updateFor = nil
		}
	case 141:
		yyVAL.updateFor = yyS[yypt-0].updateFor
	case 142:
		//line n1ql.y:1157
		{
			yyVAL.updateFor = algebra.NewUpdateFor(yyS[yypt-2].bindings, yyS[yypt-1].expr)
		}
	case 143:
		//line n1ql.y:1164
		{
			yyVAL.bindings = expression.Bindings{yyS[yypt-0].binding}
		}
	case 144:
		//line n1ql.y:1169
		{
			yyVAL.bindings = append(yyS[yypt-2].bindings, yyS[yypt-0].binding)
		}
	case 145:
		//line n1ql.y:1176
		{
			yyVAL.binding = expression.NewBinding(yyS[yypt-2].s, yyS[yypt-0].expr)
		}
	case 146:
		//line n1ql.y:1181
		{
			yyVAL.binding = expression.NewDescendantBinding(yyS[yypt-2].s, yyS[yypt-0].expr)
		}
	case 147:
		yyVAL.s = yyS[yypt-0].s
	case 148:
		//line n1ql.y:1192
		{
			yyVAL.expr = yyS[yypt-0].path
		}
	case 149:
		//line n1ql.y:1199
		{
			yyVAL.expr = nil
		}
	case 150:
		//line n1ql.y:1204
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 151:
		//line n1ql.y:1211
		{
			yyVAL.unset = algebra.NewUnset(yyS[yypt-0].unsetTerms)
		}
	case 152:
		//line n1ql.y:1218
		{
			yyVAL.unsetTerms = algebra.UnsetTerms{yyS[yypt-0].unsetTerm}
		}
	case 153:
		//line n1ql.y:1223
		{
			yyVAL.unsetTerms = append(yyS[yypt-2].unsetTerms, yyS[yypt-0].unsetTerm)
		}
	case 154:
		//line n1ql.y:1230
		{
			yyVAL.unsetTerm = algebra.NewUnsetTerm(yyS[yypt-1].path, yyS[yypt-0].updateFor)
		}
	case 155:
		//line n1ql.y:1244
		{
			source := algebra.NewMergeSourceFrom(yyS[yypt-5].keyspaceTerm, "")
			yyVAL.statement = algebra.NewMerge(yyS[yypt-7].keyspaceRef, source, yyS[yypt-3].expr, yyS[yypt-2].mergeActions, yyS[yypt-1].expr, yyS[yypt-0].projection)
		}
	case 156:
		//line n1ql.y:1250
		{
			source := algebra.NewMergeSourceSelect(yyS[yypt-7].fullselect, yyS[yypt-5].s)
			yyVAL.statement = algebra.NewMerge(yyS[yypt-10].keyspaceRef, source, yyS[yypt-3].expr, yyS[yypt-2].mergeActions, yyS[yypt-1].expr, yyS[yypt-0].projection)
		}
	case 157:
		//line n1ql.y:1258
		{
			yyVAL.mergeActions = algebra.NewMergeActions(nil, nil, nil)
		}
	case 158:
		//line n1ql.y:1263
		{
			yyVAL.mergeActions = algebra.NewMergeActions(yyS[yypt-1].mergeUpdate, yyS[yypt-0].mergeActions.Delete(), yyS[yypt-0].mergeActions.Insert())
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
			yyVAL.mergeActions = algebra.NewMergeActions(nil, nil, nil)
		}
	case 162:
		//line n1ql.y:1285
		{
			yyVAL.mergeActions = algebra.NewMergeActions(nil, yyS[yypt-1].mergeDelete, yyS[yypt-0].mergeInsert)
		}
	case 163:
		//line n1ql.y:1290
		{
			yyVAL.mergeActions = algebra.NewMergeActions(nil, nil, yyS[yypt-0].mergeInsert)
		}
	case 164:
		//line n1ql.y:1297
		{
			yyVAL.mergeInsert = nil
		}
	case 165:
		//line n1ql.y:1302
		{
			yyVAL.mergeInsert = yyS[yypt-0].mergeInsert
		}
	case 166:
		//line n1ql.y:1309
		{
			yyVAL.mergeUpdate = algebra.NewMergeUpdate(yyS[yypt-1].set, nil, yyS[yypt-0].expr)
		}
	case 167:
		//line n1ql.y:1314
		{
			yyVAL.mergeUpdate = algebra.NewMergeUpdate(yyS[yypt-2].set, yyS[yypt-1].unset, yyS[yypt-0].expr)
		}
	case 168:
		//line n1ql.y:1319
		{
			yyVAL.mergeUpdate = algebra.NewMergeUpdate(nil, yyS[yypt-1].unset, yyS[yypt-0].expr)
		}
	case 169:
		//line n1ql.y:1326
		{
			yyVAL.mergeDelete = algebra.NewMergeDelete(yyS[yypt-0].expr)
		}
	case 170:
		//line n1ql.y:1333
		{
			yyVAL.mergeInsert = algebra.NewMergeInsert(yyS[yypt-1].expr, yyS[yypt-0].expr)
		}
	case 171:
		//line n1ql.y:1347
		{
			yyVAL.statement = algebra.NewCreatePrimaryIndex(yyS[yypt-1].keyspaceRef, yyS[yypt-0].indexType)
		}
	case 172:
		//line n1ql.y:1352
		{
			yyVAL.statement = algebra.NewCreateIndex(yyS[yypt-8].s, yyS[yypt-6].keyspaceRef, yyS[yypt-4].exprs, yyS[yypt-2].expr, yyS[yypt-1].expr, yyS[yypt-0].indexType)
		}
	case 173:
		yyVAL.s = yyS[yypt-0].s
	case 174:
		//line n1ql.y:1363
		{
			yyVAL.keyspaceRef = algebra.NewKeyspaceRef("", yyS[yypt-0].s, "")
		}
	case 175:
		//line n1ql.y:1368
		{
			yyVAL.keyspaceRef = algebra.NewKeyspaceRef(yyS[yypt-2].s, yyS[yypt-0].s, "")
		}
	case 176:
		//line n1ql.y:1375
		{
			yyVAL.expr = nil
		}
	case 177:
		//line n1ql.y:1380
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 178:
		//line n1ql.y:1387
		{
			yyVAL.indexType = datastore.VIEW
		}
	case 179:
		//line n1ql.y:1392
		{
			yyVAL.indexType = datastore.VIEW
		}
	case 180:
		//line n1ql.y:1397
		{
			yyVAL.indexType = datastore.LSM
		}
	case 181:
		//line n1ql.y:1404
		{
			yyVAL.exprs = expression.Expressions{yyS[yypt-0].expr}
		}
	case 182:
		//line n1ql.y:1409
		{
			yyVAL.exprs = append(yyS[yypt-2].exprs, yyS[yypt-0].expr)
		}
	case 183:
		//line n1ql.y:1416
		{
			e := yyS[yypt-0].expr
			if !e.Indexable() {
				yylex.Error(fmt.Sprintf("Expression not indexable."))
			}

			yyVAL.expr = e
		}
	case 184:
		//line n1ql.y:1427
		{
			yyVAL.expr = nil
		}
	case 185:
		//line n1ql.y:1432
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 186:
		//line n1ql.y:1446
		{
			yyVAL.statement = algebra.NewDropIndex(yyS[yypt-0].keyspaceRef, "#primary")
		}
	case 187:
		//line n1ql.y:1451
		{
			yyVAL.statement = algebra.NewDropIndex(yyS[yypt-2].keyspaceRef, yyS[yypt-0].s)
		}
	case 188:
		//line n1ql.y:1464
		{
			yyVAL.statement = algebra.NewAlterIndex(yyS[yypt-3].keyspaceRef, yyS[yypt-1].s, yyS[yypt-0].s)
		}
	case 189:
		//line n1ql.y:1470
		{
			yyVAL.s = ""
		}
	case 190:
		//line n1ql.y:1475
		{
			yyVAL.s = yyS[yypt-0].s
		}
	case 191:
		//line n1ql.y:1489
		{
			yyVAL.path = expression.NewIdentifier(yyS[yypt-0].s)
		}
	case 192:
		//line n1ql.y:1494
		{
			yyVAL.path = expression.NewField(yyS[yypt-2].path, expression.NewFieldName(yyS[yypt-0].s))
		}
	case 193:
		//line n1ql.y:1499
		{
			field := expression.NewField(yyS[yypt-2].path, expression.NewFieldName(yyS[yypt-0].s))
			field.SetCaseInsensitive(true)
			yyVAL.path = field
		}
	case 194:
		//line n1ql.y:1506
		{
			yyVAL.path = expression.NewElement(yyS[yypt-3].path, yyS[yypt-1].expr)
		}
	case 195:
		yyVAL.expr = yyS[yypt-0].expr
	case 196:
		//line n1ql.y:1523
		{
			yyVAL.expr = expression.NewField(yyS[yypt-2].expr, expression.NewFieldName(yyS[yypt-0].s))
		}
	case 197:
		//line n1ql.y:1528
		{
			field := expression.NewField(yyS[yypt-2].expr, expression.NewFieldName(yyS[yypt-0].s))
			field.SetCaseInsensitive(true)
			yyVAL.expr = field
		}
	case 198:
		//line n1ql.y:1535
		{
			yyVAL.expr = expression.NewField(yyS[yypt-4].expr, yyS[yypt-1].expr)
		}
	case 199:
		//line n1ql.y:1540
		{
			field := expression.NewField(yyS[yypt-4].expr, yyS[yypt-1].expr)
			field.SetCaseInsensitive(true)
			yyVAL.expr = field
		}
	case 200:
		//line n1ql.y:1547
		{
			yyVAL.expr = expression.NewElement(yyS[yypt-3].expr, yyS[yypt-1].expr)
		}
	case 201:
		//line n1ql.y:1552
		{
			yyVAL.expr = expression.NewSlice(yyS[yypt-4].expr, yyS[yypt-2].expr)
		}
	case 202:
		//line n1ql.y:1557
		{
			yyVAL.expr = expression.NewSlice(yyS[yypt-5].expr, yyS[yypt-3].expr, yyS[yypt-1].expr)
		}
	case 203:
		//line n1ql.y:1563
		{
			yyVAL.expr = expression.NewAdd(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 204:
		//line n1ql.y:1568
		{
			yyVAL.expr = expression.NewSub(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 205:
		//line n1ql.y:1573
		{
			yyVAL.expr = expression.NewMult(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 206:
		//line n1ql.y:1578
		{
			yyVAL.expr = expression.NewDiv(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 207:
		//line n1ql.y:1583
		{
			yyVAL.expr = expression.NewMod(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 208:
		//line n1ql.y:1589
		{
			yyVAL.expr = expression.NewConcat(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 209:
		//line n1ql.y:1595
		{
			yyVAL.expr = expression.NewAnd(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 210:
		//line n1ql.y:1600
		{
			yyVAL.expr = expression.NewOr(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 211:
		//line n1ql.y:1605
		{
			yyVAL.expr = expression.NewNot(yyS[yypt-0].expr)
		}
	case 212:
		//line n1ql.y:1611
		{
			yyVAL.expr = expression.NewEq(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 213:
		//line n1ql.y:1616
		{
			yyVAL.expr = expression.NewEq(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 214:
		//line n1ql.y:1621
		{
			yyVAL.expr = expression.NewNE(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 215:
		//line n1ql.y:1626
		{
			yyVAL.expr = expression.NewLT(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 216:
		//line n1ql.y:1631
		{
			yyVAL.expr = expression.NewGT(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 217:
		//line n1ql.y:1636
		{
			yyVAL.expr = expression.NewLE(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 218:
		//line n1ql.y:1641
		{
			yyVAL.expr = expression.NewGE(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 219:
		//line n1ql.y:1646
		{
			yyVAL.expr = expression.NewBetween(yyS[yypt-4].expr, yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 220:
		//line n1ql.y:1651
		{
			yyVAL.expr = expression.NewNotBetween(yyS[yypt-5].expr, yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 221:
		//line n1ql.y:1656
		{
			yyVAL.expr = expression.NewLike(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 222:
		//line n1ql.y:1661
		{
			yyVAL.expr = expression.NewNotLike(yyS[yypt-3].expr, yyS[yypt-0].expr)
		}
	case 223:
		//line n1ql.y:1666
		{
			yyVAL.expr = expression.NewIn(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 224:
		//line n1ql.y:1671
		{
			yyVAL.expr = expression.NewNotIn(yyS[yypt-3].expr, yyS[yypt-0].expr)
		}
	case 225:
		//line n1ql.y:1676
		{
			yyVAL.expr = expression.NewWithin(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 226:
		//line n1ql.y:1681
		{
			yyVAL.expr = expression.NewNotWithin(yyS[yypt-3].expr, yyS[yypt-0].expr)
		}
	case 227:
		//line n1ql.y:1686
		{
			yyVAL.expr = expression.NewIsNull(yyS[yypt-2].expr)
		}
	case 228:
		//line n1ql.y:1691
		{
			yyVAL.expr = expression.NewIsNotNull(yyS[yypt-3].expr)
		}
	case 229:
		//line n1ql.y:1696
		{
			yyVAL.expr = expression.NewIsMissing(yyS[yypt-2].expr)
		}
	case 230:
		//line n1ql.y:1701
		{
			yyVAL.expr = expression.NewIsNotMissing(yyS[yypt-3].expr)
		}
	case 231:
		//line n1ql.y:1706
		{
			yyVAL.expr = expression.NewIsValued(yyS[yypt-2].expr)
		}
	case 232:
		//line n1ql.y:1711
		{
			yyVAL.expr = expression.NewIsNotValued(yyS[yypt-3].expr)
		}
	case 233:
		//line n1ql.y:1716
		{
			yyVAL.expr = expression.NewExists(yyS[yypt-0].expr)
		}
	case 234:
		yyVAL.expr = yyS[yypt-0].expr
	case 235:
		yyVAL.expr = yyS[yypt-0].expr
	case 236:
		//line n1ql.y:1730
		{
			yyVAL.expr = expression.NewIdentifier(yyS[yypt-0].s)
		}
	case 237:
		//line n1ql.y:1736
		{
			yyVAL.expr = expression.NewSelf()
		}
	case 238:
		yyVAL.expr = yyS[yypt-0].expr
	case 239:
		yyVAL.expr = yyS[yypt-0].expr
	case 240:
		//line n1ql.y:1748
		{
			yyVAL.expr = expression.NewNeg(yyS[yypt-0].expr)
		}
	case 241:
		yyVAL.expr = yyS[yypt-0].expr
	case 242:
		yyVAL.expr = yyS[yypt-0].expr
	case 243:
		yyVAL.expr = yyS[yypt-0].expr
	case 244:
		yyVAL.expr = yyS[yypt-0].expr
	case 245:
		//line n1ql.y:1767
		{
			yyVAL.expr = expression.NewField(yyS[yypt-2].expr, expression.NewFieldName(yyS[yypt-0].s))
		}
	case 246:
		//line n1ql.y:1772
		{
			field := expression.NewField(yyS[yypt-2].expr, expression.NewFieldName(yyS[yypt-0].s))
			field.SetCaseInsensitive(true)
			yyVAL.expr = field
		}
	case 247:
		//line n1ql.y:1779
		{
			yyVAL.expr = expression.NewField(yyS[yypt-4].expr, yyS[yypt-1].expr)
		}
	case 248:
		//line n1ql.y:1784
		{
			field := expression.NewField(yyS[yypt-4].expr, yyS[yypt-1].expr)
			field.SetCaseInsensitive(true)
			yyVAL.expr = field
		}
	case 249:
		//line n1ql.y:1791
		{
			yyVAL.expr = expression.NewElement(yyS[yypt-3].expr, yyS[yypt-1].expr)
		}
	case 250:
		//line n1ql.y:1796
		{
			yyVAL.expr = expression.NewSlice(yyS[yypt-4].expr, yyS[yypt-2].expr)
		}
	case 251:
		//line n1ql.y:1801
		{
			yyVAL.expr = expression.NewSlice(yyS[yypt-5].expr, yyS[yypt-3].expr, yyS[yypt-1].expr)
		}
	case 252:
		//line n1ql.y:1807
		{
			yyVAL.expr = expression.NewAdd(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 253:
		//line n1ql.y:1812
		{
			yyVAL.expr = expression.NewSub(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 254:
		//line n1ql.y:1817
		{
			yyVAL.expr = expression.NewMult(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 255:
		//line n1ql.y:1822
		{
			yyVAL.expr = expression.NewDiv(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 256:
		//line n1ql.y:1827
		{
			yyVAL.expr = expression.NewMod(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 257:
		//line n1ql.y:1833
		{
			yyVAL.expr = expression.NewConcat(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 258:
		//line n1ql.y:1847
		{
			yyVAL.expr = expression.NULL_EXPR
		}
	case 259:
		//line n1ql.y:1852
		{
			yyVAL.expr = expression.MISSING_EXPR
		}
	case 260:
		//line n1ql.y:1857
		{
			yyVAL.expr = expression.FALSE_EXPR
		}
	case 261:
		//line n1ql.y:1862
		{
			yyVAL.expr = expression.TRUE_EXPR
		}
	case 262:
		//line n1ql.y:1867
		{
			yyVAL.expr = expression.NewConstant(value.NewValue(yyS[yypt-0].f))
		}
	case 263:
		//line n1ql.y:1872
		{
			yyVAL.expr = expression.NewConstant(value.NewValue(yyS[yypt-0].n))
		}
	case 264:
		//line n1ql.y:1877
		{
			yyVAL.expr = expression.NewConstant(value.NewValue(yyS[yypt-0].s))
		}
	case 265:
		yyVAL.expr = yyS[yypt-0].expr
	case 266:
		yyVAL.expr = yyS[yypt-0].expr
	case 267:
		//line n1ql.y:1897
		{
			yyVAL.expr = expression.NewObjectConstruct(yyS[yypt-1].bindings)
		}
	case 268:
		//line n1ql.y:1904
		{
			yyVAL.bindings = nil
		}
	case 269:
		yyVAL.bindings = yyS[yypt-0].bindings
	case 270:
		//line n1ql.y:1913
		{
			yyVAL.bindings = expression.Bindings{yyS[yypt-0].binding}
		}
	case 271:
		//line n1ql.y:1918
		{
			yyVAL.bindings = append(yyS[yypt-2].bindings, yyS[yypt-0].binding)
		}
	case 272:
		//line n1ql.y:1925
		{
			yyVAL.binding = expression.NewBinding(yyS[yypt-2].s, yyS[yypt-0].expr)
		}
	case 273:
		//line n1ql.y:1932
		{
			yyVAL.expr = expression.NewArrayConstruct(yyS[yypt-1].exprs...)
		}
	case 274:
		//line n1ql.y:1939
		{
			yyVAL.exprs = nil
		}
	case 275:
		yyVAL.exprs = yyS[yypt-0].exprs
	case 276:
		//line n1ql.y:1955
		{
			yyVAL.expr = algebra.NewNamedParameter(yyS[yypt-0].s)
		}
	case 277:
		//line n1ql.y:1960
		{
			yyVAL.expr = algebra.NewPositionalParameter(yyS[yypt-0].n)
		}
	case 278:
		//line n1ql.y:1974
		{
			yyVAL.expr = yyS[yypt-1].expr
		}
	case 279:
		yyVAL.expr = yyS[yypt-0].expr
	case 280:
		yyVAL.expr = yyS[yypt-0].expr
	case 281:
		//line n1ql.y:1987
		{
			yyVAL.expr = expression.NewSimpleCase(yyS[yypt-2].expr, yyS[yypt-1].whenTerms, yyS[yypt-0].expr)
		}
	case 282:
		//line n1ql.y:1994
		{
			yyVAL.whenTerms = expression.WhenTerms{&expression.WhenTerm{yyS[yypt-2].expr, yyS[yypt-0].expr}}
		}
	case 283:
		//line n1ql.y:1999
		{
			yyVAL.whenTerms = append(yyS[yypt-4].whenTerms, &expression.WhenTerm{yyS[yypt-2].expr, yyS[yypt-0].expr})
		}
	case 284:
		//line n1ql.y:2007
		{
			yyVAL.expr = expression.NewSearchedCase(yyS[yypt-1].whenTerms, yyS[yypt-0].expr)
		}
	case 285:
		//line n1ql.y:2014
		{
			yyVAL.expr = nil
		}
	case 286:
		//line n1ql.y:2019
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 287:
		//line n1ql.y:2033
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
	case 288:
		//line n1ql.y:2052
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
	case 289:
		//line n1ql.y:2067
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
	case 290:
		yyVAL.s = yyS[yypt-0].s
	case 291:
		yyVAL.expr = yyS[yypt-0].expr
	case 292:
		yyVAL.expr = yyS[yypt-0].expr
	case 293:
		//line n1ql.y:2101
		{
			yyVAL.expr = expression.NewAny(yyS[yypt-2].bindings, yyS[yypt-1].expr)
		}
	case 294:
		//line n1ql.y:2106
		{
			yyVAL.expr = expression.NewAny(yyS[yypt-2].bindings, yyS[yypt-1].expr)
		}
	case 295:
		//line n1ql.y:2111
		{
			yyVAL.expr = expression.NewEvery(yyS[yypt-2].bindings, yyS[yypt-1].expr)
		}
	case 296:
		//line n1ql.y:2118
		{
			yyVAL.bindings = expression.Bindings{yyS[yypt-0].binding}
		}
	case 297:
		//line n1ql.y:2123
		{
			yyVAL.bindings = append(yyS[yypt-2].bindings, yyS[yypt-0].binding)
		}
	case 298:
		//line n1ql.y:2130
		{
			yyVAL.binding = expression.NewBinding(yyS[yypt-2].s, yyS[yypt-0].expr)
		}
	case 299:
		//line n1ql.y:2135
		{
			yyVAL.binding = expression.NewDescendantBinding(yyS[yypt-2].s, yyS[yypt-0].expr)
		}
	case 300:
		//line n1ql.y:2142
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 301:
		//line n1ql.y:2149
		{
			yyVAL.expr = expression.NewArray(yyS[yypt-4].expr, yyS[yypt-2].bindings, yyS[yypt-1].expr)
		}
	case 302:
		//line n1ql.y:2154
		{
			yyVAL.expr = expression.NewFirst(yyS[yypt-4].expr, yyS[yypt-2].bindings, yyS[yypt-1].expr)
		}
	case 303:
		//line n1ql.y:2168
		{
			yyVAL.expr = yyS[yypt-1].expr
		}
	case 304:
		yyVAL.expr = yyS[yypt-0].expr
	case 305:
		//line n1ql.y:2177
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
