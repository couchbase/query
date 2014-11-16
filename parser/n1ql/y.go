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
	160, 287,
	-2, 235,
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
	-2, 211,
	-1, 173,
	170, 0,
	171, 0,
	172, 0,
	-2, 212,
	-1, 174,
	170, 0,
	171, 0,
	172, 0,
	-2, 213,
	-1, 175,
	173, 0,
	174, 0,
	175, 0,
	176, 0,
	-2, 214,
	-1, 176,
	173, 0,
	174, 0,
	175, 0,
	176, 0,
	-2, 215,
	-1, 177,
	173, 0,
	174, 0,
	175, 0,
	176, 0,
	-2, 216,
	-1, 178,
	173, 0,
	174, 0,
	175, 0,
	176, 0,
	-2, 217,
	-1, 185,
	75, 0,
	-2, 220,
	-1, 186,
	58, 0,
	150, 0,
	-2, 222,
	-1, 187,
	58, 0,
	150, 0,
	-2, 224,
	-1, 281,
	75, 0,
	-2, 221,
	-1, 282,
	58, 0,
	150, 0,
	-2, 223,
	-1, 283,
	58, 0,
	150, 0,
	-2, 225,
}

const yyNprod = 303
const yyPrivate = 57344

var yyTokenNames []string
var yyStates []string

const yyLast = 2820

var yyAct = []int{

	159, 3, 584, 573, 437, 582, 574, 302, 303, 192,
	92, 93, 458, 530, 499, 209, 298, 520, 306, 134,
	538, 211, 250, 492, 394, 95, 257, 210, 411, 226,
	396, 451, 205, 393, 295, 12, 151, 130, 154, 337,
	419, 66, 155, 146, 382, 251, 132, 133, 221, 114,
	127, 273, 118, 258, 229, 453, 275, 324, 131, 322,
	52, 342, 138, 139, 469, 468, 276, 277, 278, 427,
	272, 163, 164, 165, 166, 167, 168, 169, 170, 171,
	172, 173, 174, 175, 176, 177, 178, 119, 426, 185,
	186, 187, 323, 504, 427, 273, 70, 260, 240, 523,
	449, 91, 160, 161, 70, 524, 136, 137, 412, 412,
	162, 131, 86, 426, 272, 69, 259, 223, 72, 73,
	74, 75, 235, 69, 208, 359, 341, 261, 450, 517,
	239, 448, 377, 237, 234, 236, 495, 273, 471, 472,
	233, 149, 107, 8, 314, 312, 473, 247, 239, 224,
	279, 274, 276, 277, 278, 265, 272, 89, 195, 197,
	199, 365, 366, 268, 252, 91, 460, 110, 427, 367,
	160, 161, 108, 422, 88, 267, 401, 212, 162, 232,
	149, 353, 72, 281, 282, 283, 237, 426, 135, 262,
	264, 263, 213, 227, 309, 290, 108, 128, 305, 70,
	181, 249, 296, 518, 108, 241, 557, 583, 222, 249,
	147, 108, 76, 71, 73, 74, 75, 313, 69, 578,
	300, 316, 521, 317, 67, 190, 497, 140, 189, 188,
	180, 275, 459, 311, 310, 207, 326, 301, 327, 304,
	285, 330, 331, 332, 284, 183, 597, 563, 501, 90,
	340, 596, 238, 348, 99, 305, 592, 291, 564, 292,
	343, 293, 182, 70, 357, 553, 230, 230, 315, 68,
	344, 363, 67, 351, 368, 98, 76, 71, 73, 74,
	75, 358, 69, 352, 191, 325, 439, 329, 462, 345,
	376, 68, 335, 336, 496, 519, 115, 242, 123, 286,
	385, 106, 307, 129, 356, 101, 121, 546, 388, 390,
	391, 389, 273, 350, 220, 531, 384, 544, 179, 404,
	455, 321, 320, 213, 399, 180, 274, 276, 277, 278,
	397, 272, 319, 383, 289, 587, 387, 184, 347, 68,
	122, 375, 588, 417, 97, 386, 590, 424, 120, 542,
	594, 308, 408, 562, 410, 400, 543, 593, 554, 194,
	255, 142, 200, 252, 559, 413, 398, 299, 432, 148,
	256, 253, 65, 430, 405, 406, 407, 109, 296, 414,
	103, 428, 429, 418, 425, 441, 423, 416, 440, 243,
	244, 442, 443, 102, 600, 198, 445, 228, 444, 454,
	446, 447, 354, 355, 457, 599, 575, 225, 124, 196,
	67, 219, 436, 464, 215, 180, 131, 275, 180, 180,
	180, 180, 180, 180, 528, 231, 231, 339, 474, 67,
	104, 536, 558, 465, 463, 480, 334, 456, 333, 144,
	470, 328, 218, 67, 475, 476, 595, 467, 415, 484,
	489, 486, 487, 466, 147, 485, 201, 67, 2, 349,
	346, 500, 230, 230, 230, 1, 409, 498, 556, 461,
	94, 545, 494, 493, 508, 397, 482, 68, 483, 570,
	577, 490, 488, 505, 513, 452, 395, 420, 420, 392,
	514, 491, 271, 438, 481, 297, 36, 35, 273, 34,
	280, 18, 17, 16, 15, 14, 13, 7, 510, 511,
	68, 279, 274, 276, 277, 278, 6, 272, 5, 180,
	516, 515, 4, 522, 68, 500, 252, 529, 378, 548,
	541, 525, 379, 532, 533, 287, 288, 493, 86, 547,
	540, 537, 193, 539, 539, 294, 552, 96, 550, 551,
	549, 100, 150, 527, 526, 503, 77, 72, 500, 568,
	569, 555, 86, 502, 560, 561, 338, 105, 275, 566,
	571, 572, 567, 565, 248, 576, 585, 141, 579, 581,
	580, 586, 206, 89, 254, 143, 145, 589, 63, 64,
	364, 91, 591, 369, 370, 371, 372, 373, 374, 598,
	585, 585, 602, 603, 601, 28, 117, 89, 72, 27,
	77, 506, 507, 148, 47, 91, 86, 23, 50, 49,
	26, 231, 231, 231, 88, 113, 112, 111, 25, 125,
	126, 22, 72, 44, 43, 20, 87, 19, 70, 0,
	0, 0, 78, 0, 0, 0, 421, 421, 0, 273,
	0, 0, 71, 73, 74, 75, 0, 69, 0, 0,
	0, 89, 279, 274, 276, 277, 278, 0, 272, 91,
	202, 203, 204, 0, 0, 90, 0, 214, 88, 0,
	0, 0, 0, 0, 0, 0, 72, 0, 0, 70,
	87, 0, 0, 0, 435, 0, 78, 0, 0, 90,
	0, 0, 76, 71, 73, 74, 75, 0, 69, 0,
	0, 0, 0, 70, 534, 535, 0, 0, 0, 79,
	80, 81, 82, 83, 84, 85, 76, 71, 73, 74,
	75, 0, 69, 0, 77, 0, 0, 212, 0, 0,
	86, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 90, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 77, 70, 477, 478,
	0, 0, 86, 79, 80, 81, 82, 83, 84, 85,
	76, 71, 73, 74, 75, 89, 69, 0, 0, 0,
	0, 0, 0, 91, 0, 0, 0, 0, 0, 0,
	77, 0, 88, 0, 380, 0, 86, 0, 0, 0,
	72, 0, 0, 0, 87, 0, 0, 89, 0, 0,
	78, 0, 0, 0, 0, 91, 381, 0, 0, 0,
	0, 0, 0, 0, 88, 0, 0, 0, 0, 0,
	0, 0, 72, 0, 0, 0, 87, 0, 0, 0,
	0, 89, 78, 0, 0, 0, 0, 0, 0, 91,
	0, 0, 0, 0, 0, 0, 0, 0, 88, 0,
	0, 0, 0, 0, 0, 0, 72, 90, 0, 0,
	87, 0, 0, 213, 0, 0, 78, 0, 0, 0,
	0, 70, 0, 0, 0, 0, 0, 79, 80, 81,
	82, 83, 84, 85, 76, 71, 73, 74, 75, 90,
	69, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 70, 433, 0, 0, 434, 0, 79,
	80, 81, 82, 83, 84, 85, 76, 71, 73, 74,
	75, 77, 69, 90, 0, 0, 0, 86, 0, 0,
	0, 0, 0, 0, 0, 0, 55, 70, 0, 0,
	0, 0, 0, 79, 80, 81, 82, 83, 84, 85,
	76, 71, 73, 74, 75, 77, 69, 53, 0, 0,
	0, 86, 31, 0, 0, 0, 0, 0, 54, 0,
	0, 0, 89, 0, 0, 0, 0, 0, 11, 0,
	91, 0, 0, 67, 0, 0, 0, 77, 0, 88,
	212, 0, 0, 86, 29, 0, 0, 72, 0, 0,
	0, 87, 0, 0, 0, 0, 89, 78, 0, 0,
	0, 0, 0, 33, 91, 0, 0, 0, 0, 0,
	0, 0, 0, 88, 0, 0, 0, 0, 0, 0,
	0, 72, 0, 0, 0, 87, 0, 0, 89, 0,
	0, 78, 0, 0, 0, 0, 91, 0, 0, 0,
	68, 0, 0, 0, 0, 88, 0, 0, 0, 0,
	0, 0, 0, 72, 90, 0, 0, 87, 32, 30,
	0, 0, 0, 78, 0, 0, 0, 0, 70, 360,
	361, 0, 0, 0, 79, 80, 81, 82, 83, 84,
	85, 76, 71, 73, 74, 75, 0, 69, 90, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 77, 70, 269, 0, 0, 270, 86, 79, 80,
	81, 82, 83, 84, 85, 76, 71, 73, 74, 75,
	90, 69, 0, 0, 0, 0, 213, 0, 0, 0,
	0, 0, 0, 0, 70, 0, 0, 0, 0, 0,
	79, 80, 81, 82, 83, 84, 85, 76, 71, 73,
	74, 75, 89, 266, 0, 0, 77, 0, 0, 0,
	91, 0, 86, 0, 0, 0, 0, 0, 0, 88,
	0, 0, 0, 0, 0, 0, 0, 72, 0, 0,
	0, 87, 0, 0, 0, 0, 0, 78, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	453, 0, 0, 0, 0, 0, 0, 89, 0, 0,
	0, 0, 0, 0, 0, 91, 0, 0, 0, 0,
	0, 0, 0, 0, 88, 77, 0, 0, 0, 0,
	0, 86, 72, 0, 0, 0, 87, 0, 0, 0,
	0, 249, 78, 0, 90, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 70, 0,
	0, 0, 0, 0, 79, 80, 81, 82, 83, 84,
	85, 76, 71, 73, 74, 75, 89, 69, 0, 0,
	77, 0, 0, 0, 91, 0, 86, 0, 0, 0,
	0, 0, 0, 88, 0, 0, 0, 0, 0, 90,
	0, 72, 0, 0, 0, 87, 0, 0, 0, 0,
	0, 78, 0, 70, 0, 0, 0, 0, 0, 79,
	80, 81, 82, 83, 84, 85, 76, 71, 73, 74,
	75, 89, 69, 0, 0, 0, 0, 0, 0, 91,
	0, 0, 0, 0, 0, 0, 0, 0, 88, 77,
	0, 0, 0, 0, 0, 86, 72, 0, 0, 0,
	87, 0, 0, 0, 0, 0, 78, 0, 90, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 70, 512, 0, 0, 0, 0, 79, 80,
	81, 82, 83, 84, 85, 76, 71, 73, 74, 75,
	89, 69, 0, 0, 0, 0, 0, 77, 91, 0,
	0, 0, 0, 86, 0, 0, 0, 88, 0, 0,
	0, 0, 0, 90, 0, 72, 0, 0, 0, 87,
	0, 0, 0, 0, 0, 78, 0, 70, 509, 0,
	0, 0, 0, 79, 80, 81, 82, 83, 84, 85,
	76, 71, 73, 74, 75, 0, 69, 0, 89, 0,
	0, 0, 0, 0, 0, 0, 91, 0, 0, 0,
	0, 0, 0, 77, 0, 88, 0, 0, 0, 86,
	0, 0, 0, 72, 0, 0, 0, 87, 0, 0,
	0, 0, 90, 78, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 70, 431, 0, 0,
	0, 0, 79, 80, 81, 82, 83, 84, 85, 76,
	71, 73, 74, 75, 89, 69, 403, 0, 0, 0,
	0, 77, 91, 0, 0, 0, 0, 86, 0, 0,
	0, 88, 0, 0, 0, 0, 0, 0, 0, 72,
	90, 0, 0, 87, 0, 0, 0, 0, 0, 78,
	0, 0, 0, 0, 70, 0, 0, 0, 0, 0,
	79, 80, 81, 82, 83, 84, 85, 76, 71, 73,
	74, 75, 89, 69, 0, 0, 0, 0, 0, 0,
	91, 0, 0, 0, 0, 0, 0, 0, 0, 88,
	0, 0, 0, 0, 77, 0, 0, 72, 0, 0,
	86, 87, 0, 0, 0, 0, 90, 78, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 402, 0, 0,
	70, 0, 0, 0, 0, 0, 79, 80, 81, 82,
	83, 84, 85, 76, 71, 73, 74, 75, 246, 69,
	318, 0, 0, 0, 0, 89, 0, 0, 0, 0,
	0, 0, 77, 91, 0, 0, 0, 0, 86, 0,
	0, 0, 88, 0, 90, 0, 0, 0, 0, 0,
	72, 0, 0, 0, 87, 0, 0, 0, 70, 0,
	78, 0, 0, 0, 79, 80, 81, 82, 83, 84,
	85, 76, 71, 73, 74, 75, 245, 69, 0, 0,
	0, 0, 0, 89, 0, 0, 0, 0, 0, 0,
	0, 91, 0, 0, 0, 0, 0, 0, 77, 0,
	88, 0, 0, 0, 86, 0, 0, 0, 72, 0,
	0, 0, 87, 0, 0, 0, 0, 90, 78, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 70, 0, 0, 0, 0, 0, 79, 80, 81,
	82, 83, 84, 85, 76, 71, 73, 74, 75, 89,
	69, 0, 0, 0, 0, 0, 77, 91, 0, 0,
	0, 0, 86, 0, 0, 0, 88, 0, 0, 0,
	0, 0, 0, 0, 72, 90, 0, 0, 87, 0,
	0, 0, 0, 0, 78, 0, 0, 0, 0, 70,
	0, 0, 0, 0, 0, 79, 80, 81, 82, 83,
	84, 85, 76, 71, 73, 74, 75, 89, 69, 0,
	0, 0, 0, 0, 0, 91, 0, 0, 0, 0,
	0, 0, 77, 0, 88, 0, 0, 0, 86, 0,
	0, 0, 72, 0, 0, 0, 87, 116, 0, 0,
	0, 90, 78, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 70, 0, 0, 0, 0,
	0, 79, 80, 81, 82, 83, 84, 85, 76, 71,
	73, 74, 75, 89, 69, 0, 0, 0, 0, 0,
	0, 91, 0, 0, 0, 0, 0, 0, 0, 0,
	88, 0, 0, 0, 0, 0, 0, 0, 72, 90,
	0, 0, 87, 0, 0, 0, 0, 0, 86, 0,
	0, 0, 0, 70, 0, 0, 0, 0, 0, 79,
	80, 81, 82, 83, 84, 85, 76, 71, 73, 74,
	75, 153, 69, 0, 0, 58, 61, 0, 0, 0,
	0, 0, 0, 0, 0, 48, 0, 0, 0, 0,
	0, 0, 0, 89, 0, 0, 0, 0, 0, 0,
	0, 91, 152, 0, 0, 90, 157, 0, 0, 60,
	88, 0, 0, 10, 0, 38, 62, 0, 72, 70,
	0, 0, 87, 0, 0, 79, 80, 81, 82, 83,
	84, 85, 76, 71, 73, 74, 75, 0, 69, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	24, 0, 0, 0, 9, 37, 0, 0, 0, 0,
	58, 61, 0, 0, 0, 0, 0, 0, 0, 0,
	48, 0, 0, 0, 156, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 90, 0, 0, 86, 59,
	0, 157, 0, 0, 60, 0, 0, 39, 10, 70,
	38, 62, 0, 0, 0, 79, 80, 81, 82, 83,
	84, 85, 76, 71, 73, 74, 75, 0, 69, 0,
	41, 40, 42, 21, 0, 45, 46, 51, 0, 56,
	0, 57, 0, 89, 0, 24, 0, 0, 0, 9,
	37, 91, 0, 58, 61, 0, 158, 0, 0, 0,
	88, 0, 0, 48, 0, 58, 61, 0, 72, 156,
	0, 0, 0, 0, 0, 48, 0, 0, 0, 0,
	216, 0, 0, 0, 59, 0, 0, 60, 0, 0,
	0, 10, 39, 38, 62, 0, 0, 0, 0, 60,
	0, 0, 0, 10, 0, 38, 62, 0, 0, 0,
	0, 0, 0, 0, 0, 41, 40, 42, 21, 0,
	45, 46, 51, 0, 56, 0, 57, 0, 24, 0,
	0, 0, 9, 37, 0, 90, 0, 0, 0, 0,
	24, 158, 0, 0, 9, 37, 0, 0, 0, 70,
	0, 0, 0, 0, 0, 0, 0, 0, 82, 83,
	84, 85, 76, 71, 73, 74, 75, 59, 69, 0,
	0, 0, 0, 0, 0, 39, 0, 0, 0, 59,
	0, 0, 0, 0, 0, 0, 0, 39, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 41, 40,
	42, 21, 0, 45, 46, 51, 0, 56, 0, 57,
	41, 40, 42, 21, 0, 45, 46, 51, 0, 56,
	0, 57, 55, 0, 217, 58, 61, 0, 0, 0,
	0, 0, 0, 0, 0, 48, 158, 58, 61, 0,
	0, 0, 0, 53, 0, 0, 0, 48, 31, 0,
	0, 0, 58, 61, 54, 0, 0, 0, 0, 60,
	0, 0, 48, 10, 11, 38, 62, 0, 0, 67,
	0, 60, 0, 0, 0, 10, 0, 38, 62, 0,
	29, 0, 0, 0, 0, 0, 60, 0, 0, 0,
	10, 0, 38, 62, 0, 0, 0, 0, 0, 33,
	24, 0, 0, 0, 9, 37, 0, 0, 0, 0,
	0, 0, 24, 0, 0, 0, 9, 37, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 24, 0, 0,
	0, 9, 37, 0, 0, 0, 68, 0, 0, 59,
	0, 0, 0, 0, 0, 0, 0, 39, 0, 0,
	0, 59, 0, 0, 32, 30, 0, 0, 0, 39,
	0, 0, 0, 0, 0, 0, 59, 0, 0, 0,
	41, 40, 42, 21, 39, 45, 46, 51, 0, 56,
	0, 57, 41, 40, 42, 21, 0, 45, 46, 51,
	0, 56, 0, 57, 479, 58, 61, 41, 40, 42,
	21, 0, 45, 46, 51, 48, 56, 0, 57, 362,
	58, 61, 0, 0, 0, 0, 0, 0, 0, 0,
	48, 0, 58, 61, 0, 0, 0, 0, 0, 60,
	0, 0, 48, 10, 0, 38, 62, 0, 0, 67,
	0, 0, 0, 0, 60, 0, 0, 0, 10, 0,
	38, 62, 0, 0, 0, 0, 60, 0, 0, 0,
	10, 0, 38, 62, 0, 0, 0, 0, 0, 0,
	24, 0, 0, 0, 9, 37, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 24, 0, 0, 0, 9,
	37, 0, 0, 0, 0, 0, 0, 24, 0, 0,
	0, 9, 37, 0, 0, 0, 68, 0, 0, 59,
	0, 0, 0, 0, 0, 0, 0, 39, 0, 0,
	0, 0, 0, 0, 59, 0, 0, 0, 0, 0,
	0, 0, 39, 0, 0, 0, 59, 0, 0, 0,
	41, 40, 42, 21, 39, 45, 46, 51, 116, 56,
	0, 57, 0, 58, 61, 41, 40, 42, 21, 0,
	45, 46, 51, 48, 56, 0, 57, 41, 40, 42,
	21, 0, 45, 46, 51, 0, 56, 0, 57, 0,
	0, 0, 0, 0, 0, 0, 0, 60, 0, 0,
	0, 0, 0, 38, 62, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 24, 0,
	0, 0, 0, 37, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 59, 0, 0,
	0, 0, 0, 0, 0, 39, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 41, 40,
	42, 21, 0, 45, 46, 51, 0, 56, 0, 57,
}
var yyPact = []int{

	2327, -1000, -1000, 1809, -1000, -1000, -1000, -1000, -1000, 2524,
	2524, 951, -1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000,
	-1000, -1000, -1000, -1000, 2524, -1000, -1000, -1000, 211, 328,
	315, 378, 40, 312, -1000, -1000, -1000, -1000, -1000, -1000,
	-1000, -1000, -1000, -1000, -1000, -1000, -1000, 7, 2512, -1000,
	-1000, 2497, -1000, 246, 238, 348, 42, 2524, 32, 32,
	32, 2524, 2524, -1000, -1000, 288, 377, 55, 1987, 14,
	2524, 2524, 2524, 2524, 2524, 2524, 2524, 2524, 2524, 2524,
	2524, 2524, 2524, 2524, 2524, 2524, 2655, 187, 2524, 2524,
	2524, 141, 1955, 35, -1000, -68, 283, 405, 391, 358,
	-1000, 440, 40, 40, 40, 96, -44, 167, -1000, 40,
	2155, 401, -1000, -1000, 1751, 168, 2524, -12, 1809, -1000,
	347, 37, 337, 40, 40, -23, -33, -1000, -46, -30,
	-34, 1809, -19, -1000, 147, -1000, -19, -19, 1685, 1627,
	54, -1000, 36, 288, -1000, 298, -1000, -130, -52, -71,
	-1000, -40, 2072, 2167, 2524, -1000, -1000, -1000, -1000, 1000,
	-1000, -1000, 2524, 968, -60, -60, -68, -68, -68, 474,
	1955, 1875, 2095, 2095, 2095, 99, 99, 99, 99, 485,
	-1000, 2655, 2524, 2524, 2524, 525, 35, 35, -1000, 156,
	-1000, -1000, 244, -1000, 2524, -1000, 220, -1000, 220, -1000,
	220, 2524, 299, 299, 96, 119, -1000, 200, 38, -1000,
	-1000, -1000, 36, -1000, 92, -16, 2524, -17, -1000, 168,
	2524, -1000, 2524, 1554, -1000, 241, 231, -1000, 230, -124,
	-1000, -76, -126, -1000, 42, 2524, -1000, 2524, 400, 32,
	2524, 2524, 2524, 397, 395, 32, 32, 372, -1000, 2524,
	-41, -1000, -109, 54, 203, -1000, 218, 167, 25, 38,
	38, 2167, -40, 2524, -40, 727, -54, -1000, 934, -1000,
	2354, 2655, 5, 2524, 2655, 2655, 2655, 2655, 2655, 2655,
	334, 525, 35, 35, -1000, -1000, -1000, -1000, -1000, 2524,
	1809, -1000, -1000, -1000, -35, -1000, 793, 172, -1000, 2524,
	172, 54, 62, 54, 25, 25, 297, -1000, 167, -1000,
	-1000, 16, -1000, 1496, -1000, -1000, 1430, 1809, 2524, 40,
	40, 40, 37, 38, 37, -1000, 1809, 1809, -1000, -1000,
	1809, 1809, 1809, -1000, -1000, -37, -37, 150, -1000, 432,
	1809, 36, 2524, 372, 48, 48, 2524, -1000, -1000, -1000,
	-1000, 96, -95, -1000, -130, -130, -1000, 727, -1000, -1000,
	-1000, -1000, -1000, 1372, -27, -1000, -1000, 2524, 759, -113,
	-113, -69, -69, -69, 148, 2655, 1809, 2524, -1000, -1000,
	-1000, -1000, 174, 174, 2524, 1809, 174, 174, 283, 54,
	283, 283, -36, -1000, -70, -39, -1000, 4, 2524, -1000,
	229, 220, -1000, 2524, 1809, 91, 6, -1000, -1000, -1000,
	178, 393, 2524, 392, -1000, 2524, -1000, 1809, -1000, -1000,
	-130, -103, -104, -1000, 727, -1000, -18, 2524, 167, 167,
	-1000, -1000, 603, -1000, 2339, -27, -1000, -1000, -1000, 2072,
	-1000, 1809, -1000, -1000, 174, 283, 174, 174, 25, 2524,
	25, -1000, -1000, 32, 1809, 299, -25, 1809, -1000, 149,
	2524, -1000, 121, -1000, 1809, -1000, 19, 167, 38, 38,
	-1000, -1000, -1000, 2524, 1303, 96, 96, -1000, -1000, -1000,
	1248, -1000, -40, 2524, -1000, 174, -1000, -1000, -1000, 1179,
	-1000, -38, -1000, 145, 76, 167, -1000, -1000, -62, -1000,
	1809, 37, 368, -1000, 36, 224, -130, -130, 549, -1000,
	-1000, -1000, -1000, 1809, -1000, -1000, 390, 32, 25, 25,
	283, 269, 226, 210, 2524, -1000, -1000, -1000, 2524, -41,
	-1000, 200, 167, 167, -1000, -1000, -1000, -1000, -1000, -95,
	-1000, 174, 139, 278, 299, 59, 416, -1000, 1809, 295,
	224, 224, -1000, 216, 132, 76, 91, 2524, 2524, 2524,
	-1000, -1000, 119, 54, 343, 283, -1000, -1000, 1809, 1809,
	73, 62, 54, 61, -1000, 2524, 174, -1000, 255, -1000,
	54, -1000, -1000, 259, -1000, 1124, -1000, 130, 277, -1000,
	270, -1000, 415, 125, 120, 54, 342, 331, 61, 2524,
	2524, -1000, -1000, -1000,
}
var yyPgo = []int{

	0, 637, 635, 634, 633, 631, 50, 630, 629, 0,
	143, 318, 37, 303, 45, 22, 21, 27, 15, 19,
	628, 627, 626, 625, 48, 296, 620, 619, 618, 47,
	46, 252, 28, 617, 614, 609, 606, 35, 605, 60,
	589, 588, 586, 372, 585, 43, 40, 584, 24, 26,
	301, 142, 582, 32, 13, 227, 577, 6, 574, 39,
	566, 563, 555, 554, 553, 42, 36, 552, 41, 551,
	547, 34, 545, 542, 9, 536, 535, 532, 528, 458,
	522, 518, 516, 507, 506, 505, 504, 503, 502, 501,
	499, 497, 496, 567, 44, 16, 495, 494, 493, 4,
	23, 491, 20, 7, 33, 489, 8, 30, 486, 485,
	31, 17, 480, 479, 3, 2, 5, 29, 54, 471,
	12, 469, 14, 468, 467, 465, 38, 460, 18, 459,
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
	1, 1, 2, 2, 3, 8, 8, 7, 7, 6,
	4, 13, 13, 5, 5, 20, 21, 21, 22, 25,
	25, 23, 24, 24, 33, 33, 33, 34, 26, 26,
	27, 27, 27, 30, 30, 29, 29, 31, 28, 28,
	35, 36, 36,
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
	1, 1, 1, 1, 3, 0, 1, 1, 3, 3,
	3, 0, 1, 1, 1, 3, 1, 1, 3, 4,
	5, 2, 0, 2, 4, 5, 4, 1, 1, 1,
	4, 4, 4, 1, 3, 3, 3, 2, 6, 6,
	3, 1, 1,
}
var yyChk = []int{

	-1000, -125, -79, -9, -80, -81, -82, -83, -10, 87,
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
	-67, -66, 35, 4, -126, -65, 107, 39, 179, -9,
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
	168, 167, -66, -126, -66, -9, 183, -18, -9, 165,
	168, 7, 183, 164, 178, 83, 179, 180, 181, 177,
	-11, -9, -9, -9, 88, 84, 143, -76, -75, 90,
	-9, -39, -39, -39, -72, -71, -9, -96, -95, 68,
	-95, -53, -103, -106, 120, 136, -128, 102, -51, 156,
	-16, 141, 161, -9, 161, -24, -9, -9, 126, 91,
	91, 91, 183, 168, 183, -6, -9, -9, 41, -29,
	-9, -9, -9, 41, 41, -30, -30, -59, -60, 55,
	-9, 167, 170, -57, 67, 86, -127, 135, 50, -129,
	95, -18, -48, 156, -51, -51, -65, -9, -18, 179,
	165, 166, 165, -9, -11, 156, 157, 164, -9, -11,
	-11, -11, -11, -11, -11, 7, -9, 167, -78, -77,
	11, 33, -94, -37, 144, -9, -94, -37, -57, -106,
	-57, -57, -105, -104, -48, -108, -107, -48, 69, -18,
	-45, 160, 161, 126, -9, -118, -118, -118, -117, -51,
	-117, -32, 146, -32, -68, 16, -14, -9, -59, -46,
	-51, -50, 125, -46, -9, -53, 183, 164, -49, -49,
	-18, 165, -9, 165, 168, -11, -71, -99, -98, 112,
	-99, -9, -99, -99, -74, -57, -74, -74, 167, 170,
	167, -110, -109, 51, -9, 91, -37, -9, -120, 141,
	160, -121, 110, 41, -9, 41, -12, -49, 168, 168,
	-18, 156, 157, 164, -9, -18, -18, 165, 166, 165,
	-9, -97, -66, -126, -99, -74, -99, -99, -104, -9,
	-107, -101, -100, -19, -95, 161, 145, 77, -124, -122,
	-9, 127, -61, -62, 74, -18, -51, -51, -9, 165,
	-53, -53, 165, -9, -99, -110, -32, 167, 58, 150,
	-111, 146, -17, 161, 167, -117, -63, -64, 56, -15,
	-54, 91, -49, -49, 165, 166, 41, -100, -102, -48,
	-102, -74, 80, 87, 91, -119, 97, -122, -9, -128,
	-18, -18, -99, 126, 80, -95, -123, 147, 16, 69,
	-54, -54, 137, 31, 126, -111, -120, -122, -9, -9,
	-113, -103, -106, -114, -57, 63, -74, -112, 146, -57,
	-106, -57, -116, 146, -115, -9, -99, 80, 87, -57,
	87, -57, 126, 80, 80, 31, 126, 126, -114, 63,
	63, -116, -115, -115,
}
var yyDef = []int{

	0, -2, 1, 2, 3, 4, 5, 6, 194, 0,
	0, 0, 8, 9, 10, 11, 12, 13, 14, 233,
	234, -2, 236, 237, 0, 239, 240, 241, 98, 0,
	0, 0, 0, 0, 15, 16, 17, 256, 257, 258,
	259, 260, 261, 262, 263, 273, 274, 0, 0, 288,
	289, 0, 19, 0, 0, 0, 265, 271, 0, 0,
	0, 0, 0, 26, 27, 78, 48, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 210, 232, 7, 238, 108, 0, 0, 0,
	99, 0, 0, 0, 0, 67, 0, 43, -2, 0,
	271, 0, 276, 277, 0, 282, 0, 0, 301, 302,
	0, 0, 0, 0, 0, 0, 266, 267, 0, 0,
	272, 90, 0, 293, 0, 144, 0, 0, 0, 0,
	84, 79, 0, 78, 49, -2, 51, 65, 0, 0,
	30, 31, 0, 0, 0, 38, 36, 37, 40, 43,
	195, 196, 0, 0, 202, 203, 204, 205, 206, 207,
	208, 209, -2, -2, -2, -2, -2, -2, -2, 0,
	242, 0, 0, 0, 0, -2, -2, -2, 226, 0,
	228, 230, 111, 109, 0, 20, 0, 22, 0, 24,
	0, 0, 118, 0, 67, 0, 68, 70, 0, 117,
	44, 45, 0, 47, 0, 0, 0, 0, 275, 282,
	0, 281, 0, 0, 300, 0, 0, 170, 0, 0,
	171, 0, 0, 264, 0, 0, 270, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 87, 85, 0,
	80, 81, 0, 84, 0, 73, 75, 43, 0, 0,
	0, 0, 32, 0, 33, 43, 0, 42, 0, 199,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, -2, -2, -2, 227, 229, 231, 18, 112, 0,
	110, 21, 23, 25, 100, 101, 104, 0, 119, 0,
	0, 84, 84, 84, 0, 0, 0, 71, 43, 64,
	46, 0, 284, 0, 286, 278, 0, 283, 0, 0,
	0, 0, 0, 0, 0, 268, 269, 91, 290, 294,
	297, 295, 296, 291, 292, 146, 146, 0, 88, 0,
	86, 0, 0, 87, 0, 0, 0, 55, 56, 74,
	76, 67, 66, 188, 65, 65, 39, 43, 34, 41,
	197, 198, 200, 0, 218, 243, 244, 0, 0, 250,
	251, 252, 253, 254, 255, 0, 113, 0, 103, 105,
	106, 107, 122, 122, 0, 120, 122, 122, 108, 84,
	108, 108, 133, 134, 0, 148, 149, 137, 0, 116,
	0, 0, 285, 0, 279, 175, 0, 183, 184, 172,
	186, 0, 0, 0, 28, 0, 82, 83, 29, 52,
	65, 0, 0, 53, 43, 57, 0, 0, 43, 43,
	35, 201, 0, 247, 0, 219, 102, 114, 123, 0,
	115, 121, 127, 128, 122, 108, 122, 122, 0, 0,
	0, 151, 138, 0, 69, 0, 0, 280, 168, 0,
	0, 185, 0, 298, 147, 299, 92, 43, 0, 0,
	54, 189, 190, 0, 0, 67, 67, 245, 246, 248,
	0, 124, 125, 0, 129, 122, 131, 132, 135, 137,
	150, 146, 140, 0, 154, 0, 176, 177, 0, 178,
	180, 0, 95, 93, 0, 0, 65, 65, 0, 193,
	58, 59, 249, 126, 130, 136, 0, 0, 0, 0,
	108, 0, 0, 173, 0, 187, 89, 96, 0, 94,
	60, 70, 43, 43, 191, 192, 139, 141, 142, 145,
	143, 122, 0, 0, 0, 181, 0, 179, 97, 0,
	0, 0, 152, 0, 0, 154, 175, 0, 0, 0,
	61, 62, 0, 84, 0, 108, 169, 182, 174, 77,
	158, 84, 84, 161, 166, 0, 122, 155, 0, 163,
	84, 165, 156, 0, 157, 84, 153, 0, 0, 164,
	0, 167, 0, 0, 0, 84, 0, 0, 161, 0,
	0, 159, 160, 162,
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
			yyVAL.expr = expression.FALSE_EXPR
		}
	case 258:
		//line n1ql.y:1846
		{
			yyVAL.expr = expression.TRUE_EXPR
		}
	case 259:
		//line n1ql.y:1851
		{
			yyVAL.expr = expression.NewConstant(value.NewValue(yyS[yypt-0].f))
		}
	case 260:
		//line n1ql.y:1856
		{
			yyVAL.expr = expression.NewConstant(value.NewValue(yyS[yypt-0].n))
		}
	case 261:
		//line n1ql.y:1861
		{
			yyVAL.expr = expression.NewConstant(value.NewValue(yyS[yypt-0].s))
		}
	case 262:
		yyVAL.expr = yyS[yypt-0].expr
	case 263:
		yyVAL.expr = yyS[yypt-0].expr
	case 264:
		//line n1ql.y:1881
		{
			yyVAL.expr = expression.NewObjectConstruct(yyS[yypt-1].bindings)
		}
	case 265:
		//line n1ql.y:1888
		{
			yyVAL.bindings = nil
		}
	case 266:
		yyVAL.bindings = yyS[yypt-0].bindings
	case 267:
		//line n1ql.y:1897
		{
			yyVAL.bindings = expression.Bindings{yyS[yypt-0].binding}
		}
	case 268:
		//line n1ql.y:1902
		{
			yyVAL.bindings = append(yyS[yypt-2].bindings, yyS[yypt-0].binding)
		}
	case 269:
		//line n1ql.y:1909
		{
			yyVAL.binding = expression.NewBinding(yyS[yypt-2].s, yyS[yypt-0].expr)
		}
	case 270:
		//line n1ql.y:1916
		{
			yyVAL.expr = expression.NewArrayConstruct(yyS[yypt-1].exprs...)
		}
	case 271:
		//line n1ql.y:1923
		{
			yyVAL.exprs = nil
		}
	case 272:
		yyVAL.exprs = yyS[yypt-0].exprs
	case 273:
		//line n1ql.y:1939
		{
			yyVAL.expr = algebra.NewNamedParameter(yyS[yypt-0].s)
		}
	case 274:
		//line n1ql.y:1944
		{
			yyVAL.expr = algebra.NewPositionalParameter(yyS[yypt-0].n)
		}
	case 275:
		//line n1ql.y:1958
		{
			yyVAL.expr = yyS[yypt-1].expr
		}
	case 276:
		yyVAL.expr = yyS[yypt-0].expr
	case 277:
		yyVAL.expr = yyS[yypt-0].expr
	case 278:
		//line n1ql.y:1971
		{
			yyVAL.expr = expression.NewSimpleCase(yyS[yypt-2].expr, yyS[yypt-1].whenTerms, yyS[yypt-0].expr)
		}
	case 279:
		//line n1ql.y:1978
		{
			yyVAL.whenTerms = expression.WhenTerms{&expression.WhenTerm{yyS[yypt-2].expr, yyS[yypt-0].expr}}
		}
	case 280:
		//line n1ql.y:1983
		{
			yyVAL.whenTerms = append(yyS[yypt-4].whenTerms, &expression.WhenTerm{yyS[yypt-2].expr, yyS[yypt-0].expr})
		}
	case 281:
		//line n1ql.y:1991
		{
			yyVAL.expr = expression.NewSearchedCase(yyS[yypt-1].whenTerms, yyS[yypt-0].expr)
		}
	case 282:
		//line n1ql.y:1998
		{
			yyVAL.expr = nil
		}
	case 283:
		//line n1ql.y:2003
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 284:
		//line n1ql.y:2017
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
	case 285:
		//line n1ql.y:2036
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
	case 286:
		//line n1ql.y:2051
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
	case 287:
		yyVAL.s = yyS[yypt-0].s
	case 288:
		yyVAL.expr = yyS[yypt-0].expr
	case 289:
		yyVAL.expr = yyS[yypt-0].expr
	case 290:
		//line n1ql.y:2085
		{
			yyVAL.expr = expression.NewAny(yyS[yypt-2].bindings, yyS[yypt-1].expr)
		}
	case 291:
		//line n1ql.y:2090
		{
			yyVAL.expr = expression.NewAny(yyS[yypt-2].bindings, yyS[yypt-1].expr)
		}
	case 292:
		//line n1ql.y:2095
		{
			yyVAL.expr = expression.NewEvery(yyS[yypt-2].bindings, yyS[yypt-1].expr)
		}
	case 293:
		//line n1ql.y:2102
		{
			yyVAL.bindings = expression.Bindings{yyS[yypt-0].binding}
		}
	case 294:
		//line n1ql.y:2107
		{
			yyVAL.bindings = append(yyS[yypt-2].bindings, yyS[yypt-0].binding)
		}
	case 295:
		//line n1ql.y:2114
		{
			yyVAL.binding = expression.NewBinding(yyS[yypt-2].s, yyS[yypt-0].expr)
		}
	case 296:
		//line n1ql.y:2119
		{
			yyVAL.binding = expression.NewDescendantBinding(yyS[yypt-2].s, yyS[yypt-0].expr)
		}
	case 297:
		//line n1ql.y:2126
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 298:
		//line n1ql.y:2133
		{
			yyVAL.expr = expression.NewArray(yyS[yypt-4].expr, yyS[yypt-2].bindings, yyS[yypt-1].expr)
		}
	case 299:
		//line n1ql.y:2138
		{
			yyVAL.expr = expression.NewFirst(yyS[yypt-4].expr, yyS[yypt-2].bindings, yyS[yypt-1].expr)
		}
	case 300:
		//line n1ql.y:2152
		{
			yyVAL.expr = yyS[yypt-1].expr
		}
	case 301:
		yyVAL.expr = yyS[yypt-0].expr
	case 302:
		//line n1ql.y:2161
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
