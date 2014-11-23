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
const SET = 57464
const SHOW = 57465
const SOME = 57466
const START = 57467
const STATISTICS = 57468
const SYSTEM = 57469
const THEN = 57470
const TO = 57471
const TRANSACTION = 57472
const TRIGGER = 57473
const TRUE = 57474
const TRUNCATE = 57475
const UNDER = 57476
const UNION = 57477
const UNIQUE = 57478
const UNNEST = 57479
const UNSET = 57480
const UPDATE = 57481
const UPSERT = 57482
const USE = 57483
const USER = 57484
const USING = 57485
const VALUE = 57486
const VALUED = 57487
const VALUES = 57488
const VIEW = 57489
const WHEN = 57490
const WHERE = 57491
const WHILE = 57492
const WITH = 57493
const WITHIN = 57494
const WORK = 57495
const XOR = 57496
const INT = 57497
const NUMBER = 57498
const STRING = 57499
const IDENTIFIER = 57500
const IDENTIFIER_ICASE = 57501
const NAMED_PARAM = 57502
const POSITIONAL_PARAM = 57503
const LPAREN = 57504
const RPAREN = 57505
const LBRACE = 57506
const RBRACE = 57507
const LBRACKET = 57508
const RBRACKET = 57509
const RBRACKET_ICASE = 57510
const COMMA = 57511
const COLON = 57512
const INTERESECT = 57513
const EQ = 57514
const DEQ = 57515
const NE = 57516
const LT = 57517
const GT = 57518
const LE = 57519
const GE = 57520
const CONCAT = 57521
const PLUS = 57522
const STAR = 57523
const DIV = 57524
const MOD = 57525
const UMINUS = 57526
const DOT = 57527

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
	162, 289,
	-2, 236,
	-1, 109,
	170, 63,
	-2, 64,
	-1, 146,
	51, 72,
	69, 72,
	88, 72,
	137, 72,
	-2, 50,
	-1, 173,
	172, 0,
	173, 0,
	174, 0,
	-2, 212,
	-1, 174,
	172, 0,
	173, 0,
	174, 0,
	-2, 213,
	-1, 175,
	172, 0,
	173, 0,
	174, 0,
	-2, 214,
	-1, 176,
	175, 0,
	176, 0,
	177, 0,
	178, 0,
	-2, 215,
	-1, 177,
	175, 0,
	176, 0,
	177, 0,
	178, 0,
	-2, 216,
	-1, 178,
	175, 0,
	176, 0,
	177, 0,
	178, 0,
	-2, 217,
	-1, 179,
	175, 0,
	176, 0,
	177, 0,
	178, 0,
	-2, 218,
	-1, 186,
	77, 0,
	-2, 221,
	-1, 187,
	59, 0,
	152, 0,
	-2, 223,
	-1, 188,
	59, 0,
	152, 0,
	-2, 225,
	-1, 282,
	77, 0,
	-2, 222,
	-1, 283,
	59, 0,
	152, 0,
	-2, 224,
	-1, 284,
	59, 0,
	152, 0,
	-2, 226,
}

const yyNprod = 305
const yyPrivate = 57344

var yyTokenNames []string
var yyStates []string

const yyLast = 2717

var yyAct = []int{

	160, 3, 586, 575, 441, 584, 576, 303, 304, 193,
	93, 94, 462, 532, 503, 210, 299, 523, 307, 135,
	540, 227, 211, 496, 397, 96, 258, 414, 206, 341,
	455, 399, 396, 131, 296, 12, 152, 423, 155, 147,
	338, 107, 385, 156, 134, 230, 133, 252, 92, 222,
	115, 128, 53, 119, 251, 212, 457, 108, 259, 132,
	325, 323, 8, 139, 140, 73, 345, 473, 472, 67,
	431, 324, 164, 165, 166, 167, 168, 169, 170, 171,
	172, 173, 174, 175, 176, 177, 178, 179, 120, 430,
	186, 187, 188, 261, 342, 431, 274, 71, 526, 234,
	499, 453, 260, 236, 527, 241, 274, 137, 138, 415,
	149, 209, 132, 276, 430, 273, 70, 71, 224, 276,
	415, 277, 278, 279, 378, 273, 148, 262, 344, 454,
	520, 73, 74, 75, 76, 452, 70, 380, 238, 235,
	237, 240, 315, 150, 313, 225, 71, 180, 248, 464,
	181, 196, 198, 200, 111, 240, 266, 213, 356, 77,
	72, 74, 75, 76, 269, 70, 232, 232, 136, 228,
	431, 233, 161, 162, 109, 426, 268, 310, 404, 214,
	163, 109, 231, 231, 282, 283, 284, 238, 129, 430,
	263, 265, 264, 521, 274, 362, 291, 475, 476, 253,
	274, 250, 276, 297, 150, 477, 109, 280, 275, 277,
	278, 279, 71, 273, 275, 277, 278, 279, 314, 273,
	242, 301, 317, 585, 318, 306, 72, 74, 75, 76,
	559, 70, 368, 369, 302, 109, 250, 327, 580, 328,
	370, 223, 331, 332, 333, 181, 524, 161, 162, 501,
	292, 343, 293, 463, 294, 163, 68, 182, 312, 208,
	505, 346, 565, 599, 286, 360, 141, 309, 285, 311,
	316, 351, 366, 191, 354, 371, 190, 189, 598, 239,
	594, 566, 361, 274, 355, 330, 522, 326, 305, 347,
	555, 379, 68, 336, 337, 69, 280, 275, 277, 278,
	279, 388, 273, 184, 306, 214, 359, 443, 348, 391,
	393, 394, 392, 243, 116, 308, 466, 500, 357, 358,
	407, 183, 548, 287, 69, 402, 201, 533, 199, 353,
	281, 400, 192, 546, 386, 181, 459, 390, 181, 181,
	181, 181, 181, 181, 389, 411, 421, 413, 221, 387,
	428, 124, 403, 322, 149, 321, 106, 350, 130, 320,
	69, 290, 232, 232, 232, 416, 408, 409, 410, 564,
	148, 436, 122, 197, 592, 68, 434, 68, 231, 231,
	231, 297, 412, 429, 432, 433, 427, 422, 445, 425,
	425, 444, 420, 123, 446, 447, 185, 419, 253, 449,
	253, 448, 458, 450, 451, 424, 424, 461, 417, 596,
	595, 254, 100, 556, 121, 440, 468, 244, 245, 132,
	367, 195, 68, 372, 373, 374, 375, 376, 377, 589,
	220, 544, 478, 143, 99, 561, 590, 340, 545, 484,
	460, 181, 401, 69, 474, 69, 300, 66, 479, 480,
	110, 471, 470, 488, 493, 490, 491, 342, 256, 489,
	203, 204, 205, 272, 102, 504, 104, 215, 257, 103,
	216, 602, 601, 577, 229, 226, 498, 497, 511, 400,
	486, 125, 487, 531, 68, 492, 494, 508, 516, 105,
	69, 538, 469, 467, 517, 335, 334, 329, 597, 219,
	507, 560, 418, 98, 202, 2, 352, 349, 513, 514,
	1, 502, 558, 465, 547, 145, 572, 95, 579, 456,
	398, 395, 525, 519, 518, 495, 439, 528, 504, 442,
	509, 510, 550, 543, 485, 298, 534, 535, 36, 78,
	497, 276, 549, 542, 539, 87, 541, 541, 554, 35,
	552, 553, 551, 34, 18, 17, 16, 15, 14, 13,
	504, 570, 571, 557, 7, 6, 562, 563, 5, 4,
	381, 568, 573, 574, 569, 567, 382, 578, 587, 78,
	581, 583, 582, 588, 288, 87, 289, 194, 295, 591,
	97, 90, 101, 151, 593, 530, 529, 506, 339, 249,
	92, 600, 587, 587, 604, 605, 603, 142, 207, 89,
	255, 144, 146, 78, 64, 65, 213, 73, 28, 87,
	118, 88, 274, 27, 48, 23, 51, 79, 50, 26,
	114, 90, 113, 112, 25, 280, 275, 277, 278, 279,
	92, 273, 126, 127, 22, 45, 44, 20, 19, 89,
	0, 0, 0, 0, 0, 0, 0, 73, 0, 0,
	0, 88, 0, 0, 0, 90, 0, 79, 0, 0,
	0, 0, 0, 0, 92, 0, 0, 0, 0, 0,
	0, 0, 0, 89, 91, 0, 0, 0, 0, 0,
	0, 73, 0, 0, 0, 88, 0, 0, 71, 536,
	537, 79, 0, 0, 80, 81, 82, 83, 84, 85,
	86, 77, 72, 74, 75, 76, 0, 70, 0, 0,
	0, 0, 0, 0, 91, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 71, 481,
	482, 0, 0, 0, 80, 81, 82, 83, 84, 85,
	86, 77, 72, 74, 75, 76, 78, 70, 91, 0,
	0, 0, 87, 0, 214, 0, 0, 0, 0, 0,
	0, 0, 71, 0, 0, 0, 0, 0, 80, 81,
	82, 83, 84, 85, 86, 77, 72, 74, 75, 76,
	0, 70, 0, 78, 0, 0, 0, 383, 0, 87,
	0, 0, 0, 0, 0, 0, 0, 0, 90, 0,
	0, 0, 0, 0, 0, 0, 0, 92, 0, 56,
	384, 0, 0, 0, 0, 78, 89, 0, 0, 0,
	0, 87, 0, 0, 73, 0, 0, 0, 88, 0,
	54, 0, 0, 0, 79, 90, 31, 0, 0, 0,
	0, 0, 55, 0, 92, 0, 0, 0, 0, 0,
	0, 0, 11, 89, 0, 0, 0, 68, 0, 0,
	0, 73, 0, 0, 0, 88, 0, 90, 0, 29,
	0, 79, 0, 0, 0, 0, 92, 0, 0, 0,
	0, 0, 0, 0, 0, 89, 0, 0, 33, 0,
	87, 91, 0, 73, 0, 0, 0, 88, 0, 0,
	0, 0, 0, 79, 0, 71, 437, 0, 0, 438,
	0, 80, 81, 82, 83, 84, 85, 86, 77, 72,
	74, 75, 76, 0, 70, 69, 0, 0, 91, 0,
	0, 0, 0, 0, 0, 0, 90, 0, 0, 0,
	78, 0, 71, 32, 30, 92, 87, 0, 80, 81,
	82, 83, 84, 85, 86, 77, 72, 74, 75, 76,
	91, 70, 73, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 71, 363, 364, 0, 0, 0,
	80, 81, 82, 83, 84, 85, 86, 77, 72, 74,
	75, 76, 90, 70, 78, 0, 0, 213, 0, 0,
	87, 92, 0, 0, 0, 0, 0, 0, 0, 0,
	89, 0, 0, 0, 0, 0, 0, 0, 73, 0,
	0, 0, 88, 0, 0, 0, 0, 0, 79, 91,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 71, 0, 0, 90, 0, 0, 0,
	0, 0, 0, 0, 0, 92, 77, 72, 74, 75,
	76, 0, 70, 0, 89, 0, 0, 0, 78, 0,
	0, 0, 73, 0, 87, 0, 88, 0, 0, 0,
	0, 0, 79, 0, 0, 91, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 71,
	270, 0, 0, 271, 0, 80, 81, 82, 83, 84,
	85, 86, 77, 72, 74, 75, 76, 0, 70, 0,
	90, 0, 0, 0, 0, 0, 78, 0, 0, 92,
	0, 0, 87, 0, 0, 0, 0, 0, 89, 91,
	0, 0, 0, 0, 0, 214, 73, 0, 0, 0,
	88, 0, 0, 71, 0, 0, 79, 0, 0, 80,
	81, 82, 83, 84, 85, 86, 77, 72, 74, 75,
	76, 457, 267, 0, 0, 0, 0, 0, 90, 0,
	0, 0, 0, 0, 0, 0, 0, 92, 0, 0,
	0, 0, 0, 0, 0, 78, 89, 0, 0, 0,
	0, 87, 0, 0, 73, 0, 0, 0, 88, 0,
	250, 0, 0, 91, 79, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 71, 0, 0,
	0, 0, 0, 80, 81, 82, 83, 84, 85, 86,
	77, 72, 74, 75, 76, 0, 70, 90, 0, 0,
	0, 78, 0, 0, 0, 0, 92, 87, 0, 0,
	0, 0, 0, 0, 0, 89, 0, 0, 0, 0,
	0, 91, 0, 73, 0, 0, 0, 88, 0, 0,
	0, 0, 0, 79, 0, 71, 0, 0, 0, 0,
	0, 80, 81, 82, 83, 84, 85, 86, 77, 72,
	74, 75, 76, 90, 70, 0, 0, 0, 0, 0,
	0, 0, 92, 0, 0, 0, 0, 0, 0, 0,
	78, 89, 0, 0, 0, 0, 87, 0, 0, 73,
	0, 0, 0, 88, 0, 0, 0, 0, 0, 79,
	91, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 71, 515, 0, 0, 0, 0,
	80, 81, 82, 83, 84, 85, 86, 77, 72, 74,
	75, 76, 90, 70, 0, 0, 78, 0, 0, 0,
	0, 92, 87, 0, 0, 0, 0, 0, 0, 0,
	89, 0, 0, 0, 0, 0, 91, 0, 73, 0,
	0, 0, 88, 0, 0, 0, 0, 0, 79, 0,
	71, 512, 0, 0, 0, 0, 80, 81, 82, 83,
	84, 85, 86, 77, 72, 74, 75, 76, 90, 70,
	0, 0, 0, 0, 0, 0, 0, 92, 0, 0,
	0, 0, 0, 0, 0, 0, 89, 0, 0, 78,
	0, 0, 0, 0, 73, 87, 0, 0, 88, 0,
	0, 0, 0, 0, 79, 91, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 71,
	435, 0, 0, 0, 0, 80, 81, 82, 83, 84,
	85, 86, 77, 72, 74, 75, 76, 406, 70, 0,
	0, 90, 0, 0, 0, 78, 0, 0, 0, 0,
	92, 87, 0, 0, 0, 0, 0, 0, 0, 89,
	0, 91, 0, 0, 0, 0, 0, 73, 0, 0,
	0, 88, 0, 0, 0, 71, 0, 79, 0, 0,
	0, 80, 81, 82, 83, 84, 85, 86, 77, 72,
	74, 75, 76, 0, 70, 0, 0, 90, 0, 0,
	0, 0, 0, 0, 0, 0, 92, 0, 0, 0,
	0, 0, 0, 0, 0, 89, 0, 0, 0, 0,
	0, 0, 0, 73, 0, 0, 0, 88, 0, 87,
	0, 0, 0, 79, 91, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 405, 0, 0, 71, 0,
	0, 0, 0, 0, 80, 81, 82, 83, 84, 85,
	86, 77, 72, 74, 75, 76, 319, 70, 0, 0,
	0, 0, 0, 0, 0, 90, 0, 78, 0, 0,
	0, 0, 0, 87, 92, 0, 0, 0, 0, 0,
	91, 0, 0, 89, 0, 0, 0, 0, 0, 0,
	0, 73, 0, 0, 71, 0, 0, 0, 0, 0,
	80, 81, 82, 83, 84, 85, 86, 77, 72, 74,
	75, 76, 247, 70, 78, 0, 0, 0, 0, 90,
	87, 0, 0, 0, 0, 0, 0, 0, 92, 0,
	0, 0, 0, 0, 0, 0, 0, 89, 0, 0,
	0, 0, 0, 0, 0, 73, 0, 0, 0, 88,
	0, 0, 0, 0, 0, 79, 0, 0, 91, 246,
	0, 0, 0, 0, 0, 0, 90, 0, 0, 0,
	0, 0, 71, 0, 0, 92, 0, 0, 0, 0,
	0, 0, 0, 0, 89, 77, 72, 74, 75, 76,
	0, 70, 73, 0, 0, 0, 88, 0, 0, 0,
	0, 0, 79, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 91, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 71, 0, 0, 0,
	0, 0, 80, 81, 82, 83, 84, 85, 86, 77,
	72, 74, 75, 76, 78, 70, 0, 0, 0, 0,
	87, 0, 0, 0, 0, 0, 0, 0, 0, 91,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 71, 0, 0, 0, 0, 0, 80,
	81, 82, 83, 84, 85, 86, 77, 72, 74, 75,
	76, 78, 70, 0, 0, 0, 90, 87, 0, 0,
	0, 0, 0, 0, 0, 92, 0, 0, 0, 0,
	0, 0, 0, 0, 89, 0, 0, 0, 0, 154,
	0, 0, 73, 59, 62, 0, 88, 0, 0, 0,
	0, 0, 79, 49, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 90, 0, 0, 0, 0, 0, 0,
	0, 153, 92, 0, 0, 158, 0, 0, 61, 0,
	0, 89, 10, 0, 39, 63, 0, 0, 0, 73,
	0, 78, 0, 88, 0, 0, 0, 87, 0, 79,
	0, 0, 0, 0, 0, 117, 0, 0, 0, 91,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	24, 38, 0, 71, 9, 37, 0, 0, 0, 80,
	81, 82, 83, 84, 85, 86, 77, 72, 74, 75,
	76, 0, 70, 90, 157, 0, 0, 0, 0, 0,
	0, 0, 92, 0, 0, 0, 91, 0, 0, 60,
	0, 89, 0, 0, 0, 0, 0, 40, 0, 73,
	71, 0, 0, 88, 0, 87, 80, 81, 82, 83,
	84, 85, 86, 77, 72, 74, 75, 76, 0, 70,
	42, 41, 43, 21, 0, 46, 47, 52, 0, 57,
	0, 58, 59, 62, 0, 0, 0, 0, 0, 0,
	0, 0, 49, 0, 0, 0, 159, 0, 0, 0,
	0, 90, 0, 0, 0, 0, 0, 0, 0, 0,
	92, 0, 0, 0, 158, 0, 91, 61, 0, 89,
	0, 10, 0, 39, 63, 0, 0, 73, 0, 0,
	71, 88, 0, 87, 0, 0, 80, 81, 82, 83,
	84, 85, 86, 77, 72, 74, 75, 76, 0, 70,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 24,
	38, 0, 0, 9, 37, 0, 0, 0, 59, 62,
	0, 0, 0, 0, 0, 0, 0, 0, 49, 90,
	0, 0, 0, 157, 0, 0, 0, 0, 92, 0,
	0, 0, 0, 0, 91, 0, 217, 89, 60, 0,
	0, 0, 0, 61, 0, 73, 40, 10, 71, 39,
	63, 0, 0, 0, 80, 81, 82, 83, 84, 85,
	86, 77, 72, 74, 75, 76, 0, 70, 0, 42,
	41, 43, 21, 0, 46, 47, 52, 0, 57, 0,
	58, 0, 0, 0, 0, 24, 38, 0, 0, 9,
	37, 0, 59, 62, 0, 159, 0, 0, 0, 0,
	0, 0, 49, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 91, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 60, 0, 71, 61, 0, 0,
	0, 10, 40, 39, 63, 83, 84, 85, 86, 77,
	72, 74, 75, 76, 0, 70, 0, 0, 0, 0,
	56, 0, 0, 59, 62, 42, 41, 43, 21, 0,
	46, 47, 52, 49, 57, 0, 58, 0, 0, 24,
	38, 54, 0, 9, 37, 0, 0, 31, 0, 0,
	0, 218, 0, 55, 0, 0, 0, 0, 61, 0,
	0, 0, 10, 11, 39, 63, 0, 0, 68, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 60, 0,
	29, 0, 59, 62, 0, 0, 40, 0, 0, 0,
	0, 0, 49, 0, 0, 0, 0, 0, 0, 33,
	24, 38, 0, 0, 9, 37, 0, 0, 0, 42,
	41, 43, 21, 0, 46, 47, 52, 61, 57, 0,
	58, 10, 0, 39, 63, 0, 0, 59, 62, 0,
	0, 0, 0, 0, 0, 159, 69, 49, 0, 60,
	0, 0, 0, 59, 62, 0, 0, 40, 0, 0,
	0, 0, 0, 49, 32, 30, 0, 0, 0, 24,
	38, 0, 61, 9, 37, 0, 10, 0, 39, 63,
	42, 41, 43, 21, 0, 46, 47, 52, 61, 57,
	0, 58, 10, 0, 39, 63, 0, 0, 68, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 60, 0,
	0, 0, 0, 0, 24, 38, 40, 0, 9, 37,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	24, 38, 0, 0, 9, 37, 0, 59, 62, 42,
	41, 43, 21, 0, 46, 47, 52, 49, 57, 0,
	58, 483, 0, 60, 0, 0, 0, 0, 0, 0,
	0, 40, 0, 0, 0, 0, 69, 0, 0, 60,
	0, 0, 61, 0, 0, 0, 10, 40, 39, 63,
	0, 0, 59, 62, 42, 41, 43, 21, 0, 46,
	47, 52, 49, 57, 0, 58, 365, 0, 59, 62,
	42, 41, 43, 21, 0, 46, 47, 52, 49, 57,
	0, 58, 0, 0, 24, 38, 0, 61, 9, 37,
	0, 10, 0, 39, 63, 0, 0, 0, 0, 0,
	0, 0, 0, 61, 0, 0, 0, 0, 0, 39,
	63, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 60, 0, 0, 0, 0, 0, 24,
	38, 40, 0, 9, 37, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 24, 38, 117, 0, 0,
	37, 0, 0, 0, 42, 41, 43, 21, 0, 46,
	47, 52, 0, 57, 0, 58, 0, 0, 60, 0,
	0, 0, 0, 0, 0, 0, 40, 0, 0, 0,
	0, 0, 0, 0, 60, 0, 0, 0, 0, 0,
	0, 0, 40, 0, 0, 0, 0, 0, 0, 42,
	41, 43, 21, 0, 46, 47, 52, 0, 57, 0,
	58, 0, 0, 0, 0, 42, 41, 43, 21, 0,
	46, 47, 52, 0, 57, 0, 58,
}
var yyPact = []int{

	2285, -1000, -1000, 1864, -1000, -1000, -1000, -1000, -1000, 2534,
	2534, 814, -1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000,
	-1000, -1000, -1000, -1000, 2534, -1000, -1000, -1000, 368, 402,
	399, 436, 23, 383, -1000, -1000, -1000, -1000, -1000, -1000,
	-1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000, -8, 2489,
	-1000, -1000, 2405, -1000, 310, 289, 419, 31, 2534, 10,
	10, 10, 2534, 2534, -1000, -1000, 358, 431, 77, 1895,
	89, 2534, 2534, 2534, 2534, 2534, 2534, 2534, 2534, 2534,
	2534, 2534, 2534, 2534, 2534, 2534, 2534, 2550, 244, 2534,
	2534, 2534, 187, 2022, -20, -1000, -69, 343, 369, 324,
	322, -1000, 488, 23, 23, 23, 118, -59, 147, -1000,
	23, 2140, 457, -1000, -1000, 1817, 200, 2534, -18, 1864,
	-1000, 413, 11, 412, 23, 23, -66, -30, -1000, -67,
	-27, -31, 1864, -14, -1000, 161, -1000, -14, -14, 1687,
	1640, 52, -1000, 21, 358, -1000, 394, -1000, -127, -68,
	-77, -1000, -42, 2054, 2224, 2534, -1000, -1000, -1000, -1000,
	997, -1000, -1000, 2534, 943, -49, -49, -69, -69, -69,
	46, 2022, 1944, 2100, 2100, 2100, 1586, 1586, 1586, 1586,
	456, -1000, 2550, 2534, 2534, 2534, 887, -20, -20, -1000,
	178, -1000, -1000, 269, -1000, 2534, -1000, 239, -1000, 239,
	-1000, 239, 2534, 376, 376, 118, 166, -1000, 211, 19,
	-1000, -1000, -1000, 21, -1000, 115, -19, 2534, -21, -1000,
	200, 2534, -1000, 2534, 1508, -1000, 266, 262, -1000, 260,
	-124, -1000, -99, -125, -1000, 31, 2534, -1000, 2534, 455,
	10, 2534, 2534, 2534, 454, 453, 10, 10, 381, -1000,
	2534, -41, -1000, -106, 52, 220, -1000, 232, 147, 0,
	19, 19, 2224, -42, 2534, -42, 606, 14, -1000, 818,
	-1000, 2389, 2550, 74, 2534, 2550, 2550, 2550, 2550, 2550,
	2550, 117, 887, -20, -20, -1000, -1000, -1000, -1000, -1000,
	2534, 1864, -1000, -1000, -1000, -32, -1000, 786, 203, -1000,
	2534, 203, 52, 87, 52, 0, 0, 371, -1000, 147,
	-1000, -1000, 16, -1000, 1452, -1000, -1000, 1379, 1864, 2534,
	23, 23, 23, 11, 19, 11, -1000, 1864, 1864, -1000,
	-1000, 1864, 1864, 1864, -1000, -1000, -28, -28, 174, -1000,
	486, -1000, 21, 1864, 21, 2534, 381, 48, 48, 2534,
	-1000, -1000, -1000, -1000, 118, -96, -1000, -127, -127, -1000,
	606, -1000, -1000, -1000, -1000, -1000, 1323, 28, -1000, -1000,
	2534, 749, -60, -60, -70, -70, -70, 34, 2550, 1864,
	2534, -1000, -1000, -1000, -1000, 193, 193, 2534, 1864, 193,
	193, 343, 52, 343, 343, -34, -1000, -71, -40, -1000,
	4, 2534, -1000, 243, 239, -1000, 2534, 1864, 110, -13,
	-1000, -1000, -1000, 204, 451, 2534, 450, -1000, 2534, -41,
	-1000, 1864, -1000, -1000, -127, -102, -103, -1000, 606, -1000,
	39, 2534, 147, 147, -1000, -1000, 572, -1000, 2344, 28,
	-1000, -1000, -1000, 2054, -1000, 1864, -1000, -1000, 193, 343,
	193, 193, 0, 2534, 0, -1000, -1000, 10, 1864, 376,
	-63, 1864, -1000, 170, 2534, -1000, 131, -1000, 1864, -1000,
	18, 147, 19, 19, -1000, -1000, -1000, 2534, 1254, 118,
	118, -1000, -1000, -1000, 1198, -1000, -42, 2534, -1000, 193,
	-1000, -1000, -1000, 1129, -1000, -39, -1000, 134, 98, 147,
	-1000, -1000, -65, -1000, 1864, 11, 426, -1000, 234, -127,
	-127, 532, -1000, -1000, -1000, -1000, 1864, -1000, -1000, 449,
	10, 0, 0, 343, 349, 240, 223, 2534, -1000, -1000,
	-1000, 2534, -1000, 211, 147, 147, -1000, -1000, -1000, -1000,
	-1000, -96, -1000, 193, 162, 331, 376, 81, 485, -1000,
	1864, 364, 234, 234, -1000, 230, 153, 98, 110, 2534,
	2534, 2534, -1000, -1000, 166, 52, 408, 343, -1000, -1000,
	1864, 1864, 90, 87, 52, 75, -1000, 2534, 193, -1000,
	347, -1000, 52, -1000, -1000, 285, -1000, 1071, -1000, 152,
	328, -1000, 327, -1000, 466, 150, 135, 52, 407, 406,
	75, 2534, 2534, -1000, -1000, -1000,
}
var yyPgo = []int{

	0, 648, 647, 646, 645, 644, 51, 643, 642, 0,
	62, 147, 33, 358, 47, 54, 55, 22, 15, 19,
	634, 633, 632, 630, 49, 314, 629, 628, 626, 44,
	46, 279, 27, 625, 624, 623, 620, 35, 618, 52,
	615, 614, 612, 447, 611, 39, 37, 610, 24, 26,
	41, 57, 608, 28, 13, 266, 607, 6, 599, 40,
	598, 597, 29, 596, 595, 43, 36, 593, 69, 592,
	590, 34, 588, 587, 9, 586, 584, 576, 570, 505,
	569, 568, 565, 564, 559, 558, 557, 556, 555, 554,
	553, 549, 538, 356, 42, 16, 535, 534, 529, 4,
	23, 525, 20, 7, 32, 521, 8, 31, 520, 519,
	30, 17, 518, 516, 3, 2, 5, 21, 45, 514,
	12, 513, 14, 512, 511, 510, 38, 507, 18, 506,
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
	60, 12, 12, 61, 61, 62, 63, 63, 64, 70,
	70, 69, 72, 72, 71, 78, 78, 77, 77, 74,
	74, 73, 76, 76, 75, 84, 84, 93, 93, 96,
	96, 95, 94, 99, 99, 98, 97, 97, 85, 85,
	86, 87, 87, 87, 103, 105, 105, 104, 110, 110,
	109, 101, 101, 100, 100, 19, 102, 32, 32, 106,
	108, 108, 107, 88, 88, 111, 111, 111, 111, 112,
	112, 112, 116, 116, 113, 113, 113, 114, 115, 90,
	90, 117, 118, 118, 119, 119, 120, 120, 120, 124,
	124, 122, 123, 123, 91, 91, 92, 121, 121, 48,
	48, 48, 48, 48, 48, 9, 9, 9, 9, 9,
	9, 9, 9, 9, 9, 9, 9, 9, 9, 9,
	9, 9, 9, 9, 9, 9, 9, 9, 9, 9,
	9, 9, 9, 9, 9, 9, 9, 9, 9, 9,
	9, 9, 9, 9, 10, 10, 10, 10, 10, 10,
	10, 10, 10, 11, 11, 11, 11, 11, 11, 11,
	11, 11, 11, 11, 11, 11, 11, 1, 1, 1,
	1, 1, 1, 1, 2, 2, 3, 8, 8, 7,
	7, 6, 4, 13, 13, 5, 5, 20, 21, 21,
	22, 25, 25, 23, 24, 24, 33, 33, 33, 34,
	26, 26, 27, 27, 27, 30, 30, 29, 29, 31,
	28, 28, 35, 36, 36,
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
	1, 1, 3, 0, 1, 2, 0, 1, 2, 0,
	1, 3, 1, 3, 2, 0, 1, 1, 1, 0,
	1, 2, 0, 1, 2, 6, 6, 4, 2, 0,
	1, 2, 2, 0, 1, 2, 1, 2, 6, 6,
	7, 8, 7, 7, 2, 1, 3, 4, 0, 1,
	4, 1, 3, 3, 3, 1, 1, 0, 2, 2,
	1, 3, 2, 10, 13, 0, 6, 6, 6, 0,
	6, 6, 0, 6, 2, 3, 2, 1, 2, 6,
	11, 1, 1, 3, 0, 3, 0, 2, 2, 1,
	3, 1, 0, 2, 5, 5, 6, 0, 3, 1,
	3, 3, 5, 5, 4, 1, 3, 3, 5, 5,
	4, 5, 6, 3, 3, 3, 3, 3, 3, 3,
	3, 2, 3, 3, 3, 3, 3, 3, 3, 5,
	6, 3, 4, 3, 4, 3, 4, 3, 4, 3,
	4, 3, 4, 2, 1, 1, 1, 1, 1, 2,
	1, 1, 1, 1, 3, 3, 5, 5, 4, 5,
	6, 3, 3, 3, 3, 3, 3, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 3, 0, 1, 1,
	3, 3, 3, 0, 1, 1, 1, 3, 1, 1,
	3, 4, 5, 2, 0, 2, 4, 5, 4, 1,
	1, 1, 4, 4, 4, 1, 3, 3, 3, 2,
	6, 6, 3, 1, 1,
}
var yyChk = []int{

	-1000, -125, -79, -9, -80, -81, -82, -83, -10, 89,
	47, 48, -37, -84, -85, -86, -87, -88, -89, -1,
	-2, 158, -5, -33, 85, -20, -26, -35, -38, 65,
	140, 32, 139, 84, -90, -91, -92, 90, 86, 49,
	132, 156, 155, 157, -3, -4, 160, 161, -34, 18,
	-27, -28, 162, -39, 26, 38, 5, 164, 166, 8,
	124, 43, 9, 50, -41, -40, -43, -68, 53, 121,
	185, 166, 180, 85, 181, 182, 183, 179, 7, 95,
	172, 173, 174, 175, 176, 177, 178, 13, 89, 77,
	59, 152, 68, -9, -9, -79, -9, -70, 135, 66,
	44, -69, 96, 67, 67, 53, -93, -50, -51, 158,
	67, 162, -21, -22, -23, -9, -25, 148, -36, -9,
	-37, 104, 62, 104, 62, 62, -8, -7, -6, 157,
	-13, -12, -9, -30, -29, -19, 158, -30, -30, -9,
	-9, -55, -56, 75, -44, -43, -42, -45, -51, -50,
	127, -67, -66, 36, 4, -126, -65, 109, 40, 181,
	-9, 158, 159, 166, -9, -9, -9, -9, -9, -9,
	-9, -9, -9, -9, -9, -9, -9, -9, -9, -9,
	-11, -10, 13, 77, 59, 152, -9, -9, -9, 90,
	89, 86, 145, -74, -73, 78, -39, 4, -39, 4,
	-39, 4, 16, -93, -93, -93, -53, -52, 141, 170,
	-18, -17, -16, 10, 158, -93, -13, 36, 181, 42,
	-25, 148, -24, 41, -9, 163, 62, -117, 158, 62,
	-118, -51, -50, -118, 165, 169, 170, 167, 169, -31,
	169, 119, 59, 152, -31, -31, 52, 52, -57, -58,
	149, -15, -14, -16, -55, -47, 64, 74, -49, 185,
	170, 170, 169, -66, -126, -66, -9, 185, -18, -9,
	167, 170, 7, 185, 166, 180, 85, 181, 182, 183,
	179, -11, -9, -9, -9, 90, 86, 145, -76, -75,
	92, -9, -39, -39, -39, -72, -71, -9, -96, -95,
	70, -95, -53, -103, -106, 122, 138, -128, 104, -51,
	158, -16, 143, 163, -9, 163, -24, -9, -9, 128,
	93, 93, 93, 185, 170, 185, -6, -9, -9, 42,
	-29, -9, -9, -9, 42, 42, -30, -30, -59, -60,
	56, -62, 76, -9, 169, 172, -57, 69, 88, -127,
	137, 51, -129, 97, -18, -48, 158, -51, -51, -65,
	-9, -18, 181, 167, 168, 167, -9, -11, 158, 159,
	166, -9, -11, -11, -11, -11, -11, -11, 7, -9,
	169, -78, -77, 11, 34, -94, -37, 146, -9, -94,
	-37, -57, -106, -57, -57, -105, -104, -48, -108, -107,
	-48, 71, -18, -45, 162, 163, 128, -9, -118, -118,
	-118, -117, -51, -117, -32, 148, -32, -68, 16, -15,
	-14, -9, -59, -46, -51, -50, 127, -46, -9, -53,
	185, 166, -49, -49, -18, 167, -9, 167, 170, -11,
	-71, -99, -98, 114, -99, -9, -99, -99, -74, -57,
	-74, -74, 169, 172, 169, -110, -109, 52, -9, 93,
	-37, -9, -120, 143, 162, -121, 112, 42, -9, 42,
	-12, -49, 170, 170, -18, 158, 159, 166, -9, -18,
	-18, 167, 168, 167, -9, -97, -66, -126, -99, -74,
	-99, -99, -104, -9, -107, -101, -100, -19, -95, 163,
	147, 79, -124, -122, -9, 129, -61, -62, -18, -51,
	-51, -9, 167, -53, -53, 167, -9, -99, -110, -32,
	169, 59, 152, -111, 148, -17, 163, 169, -117, -63,
	-64, 57, -54, 93, -49, -49, 167, 168, 42, -100,
	-102, -48, -102, -74, 82, 89, 93, -119, 99, -122,
	-9, -128, -18, -18, -99, 128, 82, -95, -123, 149,
	16, 71, -54, -54, 139, 32, 128, -111, -120, -122,
	-9, -9, -113, -103, -106, -114, -57, 65, -74, -112,
	148, -57, -106, -57, -116, 148, -115, -9, -99, 82,
	89, -57, 89, -57, 128, 82, 82, 32, 128, 128,
	-114, 65, 65, -116, -115, -115,
}
var yyDef = []int{

	0, -2, 1, 2, 3, 4, 5, 6, 195, 0,
	0, 0, 8, 9, 10, 11, 12, 13, 14, 234,
	235, -2, 237, 238, 0, 240, 241, 242, 99, 0,
	0, 0, 0, 0, 15, 16, 17, 257, 258, 259,
	260, 261, 262, 263, 264, 265, 275, 276, 0, 0,
	290, 291, 0, 19, 0, 0, 0, 267, 273, 0,
	0, 0, 0, 0, 26, 27, 78, 48, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 211, 233, 7, 239, 109, 0, 0,
	0, 100, 0, 0, 0, 0, 67, 0, 43, -2,
	0, 273, 0, 278, 279, 0, 284, 0, 0, 303,
	304, 0, 0, 0, 0, 0, 0, 268, 269, 0,
	0, 274, 91, 0, 295, 0, 145, 0, 0, 0,
	0, 84, 79, 0, 78, 49, -2, 51, 65, 0,
	0, 30, 31, 0, 0, 0, 38, 36, 37, 40,
	43, 196, 197, 0, 0, 203, 204, 205, 206, 207,
	208, 209, 210, -2, -2, -2, -2, -2, -2, -2,
	0, 243, 0, 0, 0, 0, -2, -2, -2, 227,
	0, 229, 231, 112, 110, 0, 20, 0, 22, 0,
	24, 0, 0, 119, 0, 67, 0, 68, 70, 0,
	118, 44, 45, 0, 47, 0, 0, 0, 0, 277,
	284, 0, 283, 0, 0, 302, 0, 0, 171, 0,
	0, 172, 0, 0, 266, 0, 0, 272, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 87, 85,
	0, 80, 81, 0, 84, 0, 73, 75, 43, 0,
	0, 0, 0, 32, 0, 33, 43, 0, 42, 0,
	200, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, -2, -2, -2, 228, 230, 232, 18, 113,
	0, 111, 21, 23, 25, 101, 102, 105, 0, 120,
	0, 0, 84, 84, 84, 0, 0, 0, 71, 43,
	64, 46, 0, 286, 0, 288, 280, 0, 285, 0,
	0, 0, 0, 0, 0, 0, 270, 271, 92, 292,
	296, 299, 297, 298, 293, 294, 147, 147, 0, 88,
	0, 90, 0, 86, 0, 0, 87, 0, 0, 0,
	55, 56, 74, 76, 67, 66, 189, 65, 65, 39,
	43, 34, 41, 198, 199, 201, 0, 219, 244, 245,
	0, 0, 251, 252, 253, 254, 255, 256, 0, 114,
	0, 104, 106, 107, 108, 123, 123, 0, 121, 123,
	123, 109, 84, 109, 109, 134, 135, 0, 149, 150,
	138, 0, 117, 0, 0, 287, 0, 281, 176, 0,
	184, 185, 173, 187, 0, 0, 0, 28, 0, 95,
	82, 83, 29, 52, 65, 0, 0, 53, 43, 57,
	0, 0, 43, 43, 35, 202, 0, 248, 0, 220,
	103, 115, 124, 0, 116, 122, 128, 129, 123, 109,
	123, 123, 0, 0, 0, 152, 139, 0, 69, 0,
	0, 282, 169, 0, 0, 186, 0, 300, 148, 301,
	93, 43, 0, 0, 54, 190, 191, 0, 0, 67,
	67, 246, 247, 249, 0, 125, 126, 0, 130, 123,
	132, 133, 136, 138, 151, 147, 141, 0, 155, 0,
	177, 178, 0, 179, 181, 0, 96, 94, 0, 65,
	65, 0, 194, 58, 59, 250, 127, 131, 137, 0,
	0, 0, 0, 109, 0, 0, 174, 0, 188, 89,
	97, 0, 60, 70, 43, 43, 192, 193, 140, 142,
	143, 146, 144, 123, 0, 0, 0, 182, 0, 180,
	98, 0, 0, 0, 153, 0, 0, 155, 176, 0,
	0, 0, 61, 62, 0, 84, 0, 109, 170, 183,
	175, 77, 159, 84, 84, 162, 167, 0, 123, 156,
	0, 164, 84, 166, 157, 0, 158, 84, 154, 0,
	0, 165, 0, 168, 0, 0, 0, 84, 0, 0,
	162, 0, 0, 160, 161, 163,
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
	182, 183, 184, 185,
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
		//line n1ql.y:346
		{
			yylex.(*lexer).setStatement(yyS[yypt-0].statement)
		}
	case 2:
		//line n1ql.y:351
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
		//line n1ql.y:368
		{
			yyVAL.statement = algebra.NewExplain(yyS[yypt-0].statement)
		}
	case 8:
		//line n1ql.y:375
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
		//line n1ql.y:406
		{
			yyVAL.fullselect = algebra.NewSelect(yyS[yypt-3].subresult, yyS[yypt-2].order, yyS[yypt-0].expr, yyS[yypt-1].expr) /* OFFSET precedes LIMIT */
		}
	case 19:
		//line n1ql.y:412
		{
			yyVAL.subresult = yyS[yypt-0].subselect
		}
	case 20:
		//line n1ql.y:417
		{
			yyVAL.subresult = algebra.NewUnion(yyS[yypt-2].subresult, yyS[yypt-0].subselect)
		}
	case 21:
		//line n1ql.y:422
		{
			yyVAL.subresult = algebra.NewUnionAll(yyS[yypt-3].subresult, yyS[yypt-0].subselect)
		}
	case 22:
		//line n1ql.y:427
		{
			yyVAL.subresult = algebra.NewIntersect(yyS[yypt-2].subresult, yyS[yypt-0].subselect)
		}
	case 23:
		//line n1ql.y:432
		{
			yyVAL.subresult = algebra.NewIntersectAll(yyS[yypt-3].subresult, yyS[yypt-0].subselect)
		}
	case 24:
		//line n1ql.y:437
		{
			yyVAL.subresult = algebra.NewExcept(yyS[yypt-2].subresult, yyS[yypt-0].subselect)
		}
	case 25:
		//line n1ql.y:442
		{
			yyVAL.subresult = algebra.NewExceptAll(yyS[yypt-3].subresult, yyS[yypt-0].subselect)
		}
	case 26:
		yyVAL.subselect = yyS[yypt-0].subselect
	case 27:
		yyVAL.subselect = yyS[yypt-0].subselect
	case 28:
		//line n1ql.y:455
		{
			yyVAL.subselect = algebra.NewSubselect(yyS[yypt-4].fromTerm, yyS[yypt-3].bindings, yyS[yypt-2].expr, yyS[yypt-1].group, yyS[yypt-0].projection)
		}
	case 29:
		//line n1ql.y:462
		{
			yyVAL.subselect = algebra.NewSubselect(yyS[yypt-3].fromTerm, yyS[yypt-2].bindings, yyS[yypt-1].expr, yyS[yypt-0].group, yyS[yypt-4].projection)
		}
	case 30:
		//line n1ql.y:477
		{
			yyVAL.projection = yyS[yypt-0].projection
		}
	case 31:
		//line n1ql.y:484
		{
			yyVAL.projection = algebra.NewProjection(false, yyS[yypt-0].resultTerms)
		}
	case 32:
		//line n1ql.y:489
		{
			yyVAL.projection = algebra.NewProjection(true, yyS[yypt-0].resultTerms)
		}
	case 33:
		//line n1ql.y:494
		{
			yyVAL.projection = algebra.NewProjection(false, yyS[yypt-0].resultTerms)
		}
	case 34:
		//line n1ql.y:499
		{
			yyVAL.projection = algebra.NewRawProjection(false, yyS[yypt-1].expr, yyS[yypt-0].s)
		}
	case 35:
		//line n1ql.y:504
		{
			yyVAL.projection = algebra.NewRawProjection(true, yyS[yypt-1].expr, yyS[yypt-0].s)
		}
	case 38:
		//line n1ql.y:517
		{
			yyVAL.resultTerms = algebra.ResultTerms{yyS[yypt-0].resultTerm}
		}
	case 39:
		//line n1ql.y:522
		{
			yyVAL.resultTerms = append(yyS[yypt-2].resultTerms, yyS[yypt-0].resultTerm)
		}
	case 40:
		//line n1ql.y:529
		{
			yyVAL.resultTerm = algebra.NewResultTerm(nil, true, "")
		}
	case 41:
		//line n1ql.y:534
		{
			yyVAL.resultTerm = algebra.NewResultTerm(yyS[yypt-2].expr, true, "")
		}
	case 42:
		//line n1ql.y:539
		{
			yyVAL.resultTerm = algebra.NewResultTerm(yyS[yypt-1].expr, false, yyS[yypt-0].s)
		}
	case 43:
		//line n1ql.y:546
		{
			yyVAL.s = ""
		}
	case 44:
		yyVAL.s = yyS[yypt-0].s
	case 45:
		yyVAL.s = yyS[yypt-0].s
	case 46:
		//line n1ql.y:557
		{
			yyVAL.s = yyS[yypt-0].s
		}
	case 47:
		yyVAL.s = yyS[yypt-0].s
	case 48:
		//line n1ql.y:575
		{
			yyVAL.fromTerm = nil
		}
	case 49:
		yyVAL.fromTerm = yyS[yypt-0].fromTerm
	case 50:
		//line n1ql.y:584
		{
			yyVAL.fromTerm = yyS[yypt-0].fromTerm
		}
	case 51:
		//line n1ql.y:591
		{
			yyVAL.fromTerm = yyS[yypt-0].keyspaceTerm
		}
	case 52:
		//line n1ql.y:596
		{
			yyVAL.fromTerm = algebra.NewJoin(yyS[yypt-3].fromTerm, yyS[yypt-2].b, yyS[yypt-0].keyspaceTerm)
		}
	case 53:
		//line n1ql.y:601
		{
			yyVAL.fromTerm = algebra.NewNest(yyS[yypt-3].fromTerm, yyS[yypt-2].b, yyS[yypt-0].keyspaceTerm)
		}
	case 54:
		//line n1ql.y:606
		{
			yyVAL.fromTerm = algebra.NewUnnest(yyS[yypt-4].fromTerm, yyS[yypt-3].b, yyS[yypt-1].expr, yyS[yypt-0].s)
		}
	case 57:
		//line n1ql.y:619
		{
			yyVAL.keyspaceTerm = algebra.NewKeyspaceTerm("", yyS[yypt-3].s, yyS[yypt-2].path, yyS[yypt-1].s, yyS[yypt-0].expr)
		}
	case 58:
		//line n1ql.y:624
		{
			yyVAL.keyspaceTerm = algebra.NewKeyspaceTerm(yyS[yypt-5].s, yyS[yypt-3].s, yyS[yypt-2].path, yyS[yypt-1].s, yyS[yypt-0].expr)
		}
	case 59:
		//line n1ql.y:629
		{
			yyVAL.keyspaceTerm = algebra.NewKeyspaceTerm("#system", yyS[yypt-3].s, yyS[yypt-2].path, yyS[yypt-1].s, yyS[yypt-0].expr)
		}
	case 60:
		//line n1ql.y:636
		{
			yyVAL.keyspaceTerm = algebra.NewKeyspaceTerm("", yyS[yypt-3].s, yyS[yypt-2].path, yyS[yypt-1].s, yyS[yypt-0].expr)
		}
	case 61:
		//line n1ql.y:641
		{
			yyVAL.keyspaceTerm = algebra.NewKeyspaceTerm(yyS[yypt-5].s, yyS[yypt-3].s, yyS[yypt-2].path, yyS[yypt-1].s, yyS[yypt-0].expr)
		}
	case 62:
		//line n1ql.y:646
		{
			yyVAL.keyspaceTerm = algebra.NewKeyspaceTerm("#system", yyS[yypt-3].s, yyS[yypt-2].path, yyS[yypt-1].s, yyS[yypt-0].expr)
		}
	case 63:
		yyVAL.s = yyS[yypt-0].s
	case 64:
		yyVAL.s = yyS[yypt-0].s
	case 65:
		//line n1ql.y:661
		{
			yyVAL.path = nil
		}
	case 66:
		//line n1ql.y:666
		{
			yyVAL.path = yyS[yypt-0].path
		}
	case 67:
		//line n1ql.y:673
		{
			yyVAL.expr = nil
		}
	case 68:
		yyVAL.expr = yyS[yypt-0].expr
	case 69:
		//line n1ql.y:682
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 70:
		//line n1ql.y:689
		{
		}
	case 72:
		//line n1ql.y:697
		{
			yyVAL.b = false
		}
	case 73:
		//line n1ql.y:702
		{
			yyVAL.b = false
		}
	case 74:
		//line n1ql.y:707
		{
			yyVAL.b = true
		}
	case 77:
		//line n1ql.y:720
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 78:
		//line n1ql.y:734
		{
			yyVAL.bindings = nil
		}
	case 79:
		yyVAL.bindings = yyS[yypt-0].bindings
	case 80:
		//line n1ql.y:743
		{
			yyVAL.bindings = yyS[yypt-0].bindings
		}
	case 81:
		//line n1ql.y:750
		{
			yyVAL.bindings = expression.Bindings{yyS[yypt-0].binding}
		}
	case 82:
		//line n1ql.y:755
		{
			yyVAL.bindings = append(yyS[yypt-2].bindings, yyS[yypt-0].binding)
		}
	case 83:
		//line n1ql.y:762
		{
			yyVAL.binding = expression.NewBinding(yyS[yypt-2].s, yyS[yypt-0].expr)
		}
	case 84:
		//line n1ql.y:776
		{
			yyVAL.expr = nil
		}
	case 85:
		yyVAL.expr = yyS[yypt-0].expr
	case 86:
		//line n1ql.y:785
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 87:
		//line n1ql.y:799
		{
			yyVAL.group = nil
		}
	case 88:
		yyVAL.group = yyS[yypt-0].group
	case 89:
		//line n1ql.y:808
		{
			yyVAL.group = algebra.NewGroup(yyS[yypt-2].exprs, yyS[yypt-1].bindings, yyS[yypt-0].expr)
		}
	case 90:
		//line n1ql.y:813
		{
			yyVAL.group = algebra.NewGroup(nil, yyS[yypt-0].bindings, nil)
		}
	case 91:
		//line n1ql.y:820
		{
			yyVAL.exprs = expression.Expressions{yyS[yypt-0].expr}
		}
	case 92:
		//line n1ql.y:825
		{
			yyVAL.exprs = append(yyS[yypt-2].exprs, yyS[yypt-0].expr)
		}
	case 93:
		//line n1ql.y:832
		{
			yyVAL.bindings = nil
		}
	case 94:
		yyVAL.bindings = yyS[yypt-0].bindings
	case 95:
		//line n1ql.y:841
		{
			yyVAL.bindings = yyS[yypt-0].bindings
		}
	case 96:
		//line n1ql.y:848
		{
			yyVAL.expr = nil
		}
	case 97:
		yyVAL.expr = yyS[yypt-0].expr
	case 98:
		//line n1ql.y:857
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 99:
		//line n1ql.y:871
		{
			yyVAL.order = nil
		}
	case 100:
		yyVAL.order = yyS[yypt-0].order
	case 101:
		//line n1ql.y:880
		{
			yyVAL.order = algebra.NewOrder(yyS[yypt-0].sortTerms)
		}
	case 102:
		//line n1ql.y:887
		{
			yyVAL.sortTerms = algebra.SortTerms{yyS[yypt-0].sortTerm}
		}
	case 103:
		//line n1ql.y:892
		{
			yyVAL.sortTerms = append(yyS[yypt-2].sortTerms, yyS[yypt-0].sortTerm)
		}
	case 104:
		//line n1ql.y:899
		{
			yyVAL.sortTerm = algebra.NewSortTerm(yyS[yypt-1].expr, yyS[yypt-0].b)
		}
	case 105:
		//line n1ql.y:906
		{
			yyVAL.b = false
		}
	case 106:
		yyVAL.b = yyS[yypt-0].b
	case 107:
		//line n1ql.y:915
		{
			yyVAL.b = false
		}
	case 108:
		//line n1ql.y:920
		{
			yyVAL.b = true
		}
	case 109:
		//line n1ql.y:934
		{
			yyVAL.expr = nil
		}
	case 110:
		yyVAL.expr = yyS[yypt-0].expr
	case 111:
		//line n1ql.y:943
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 112:
		//line n1ql.y:957
		{
			yyVAL.expr = nil
		}
	case 113:
		yyVAL.expr = yyS[yypt-0].expr
	case 114:
		//line n1ql.y:966
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 115:
		//line n1ql.y:980
		{
			yyVAL.statement = algebra.NewInsertValues(yyS[yypt-3].keyspaceRef, yyS[yypt-2].expr, yyS[yypt-1].expr, yyS[yypt-0].projection)
		}
	case 116:
		//line n1ql.y:985
		{
			yyVAL.statement = algebra.NewInsertSelect(yyS[yypt-3].keyspaceRef, yyS[yypt-2].expr, yyS[yypt-1].fullselect, yyS[yypt-0].projection)
		}
	case 117:
		//line n1ql.y:992
		{
			yyVAL.keyspaceRef = algebra.NewKeyspaceRef(yyS[yypt-3].s, yyS[yypt-1].s, yyS[yypt-0].s)
		}
	case 118:
		//line n1ql.y:997
		{
			yyVAL.keyspaceRef = algebra.NewKeyspaceRef("", yyS[yypt-1].s, yyS[yypt-0].s)
		}
	case 119:
		//line n1ql.y:1004
		{
			yyVAL.expr = nil
		}
	case 120:
		yyVAL.expr = yyS[yypt-0].expr
	case 121:
		//line n1ql.y:1013
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 122:
		//line n1ql.y:1020
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 123:
		//line n1ql.y:1027
		{
			yyVAL.projection = nil
		}
	case 124:
		yyVAL.projection = yyS[yypt-0].projection
	case 125:
		//line n1ql.y:1036
		{
			yyVAL.projection = yyS[yypt-0].projection
		}
	case 126:
		//line n1ql.y:1043
		{
			yyVAL.projection = algebra.NewProjection(false, yyS[yypt-0].resultTerms)
		}
	case 127:
		//line n1ql.y:1048
		{
			yyVAL.projection = algebra.NewRawProjection(false, yyS[yypt-0].expr, "")
		}
	case 128:
		//line n1ql.y:1062
		{
			yyVAL.statement = algebra.NewUpsertValues(yyS[yypt-3].keyspaceRef, yyS[yypt-2].expr, yyS[yypt-1].expr, yyS[yypt-0].projection)
		}
	case 129:
		//line n1ql.y:1067
		{
			yyVAL.statement = algebra.NewUpsertSelect(yyS[yypt-3].keyspaceRef, yyS[yypt-2].expr, yyS[yypt-1].fullselect, yyS[yypt-0].projection)
		}
	case 130:
		//line n1ql.y:1081
		{
			yyVAL.statement = algebra.NewDelete(yyS[yypt-4].keyspaceRef, yyS[yypt-3].expr, yyS[yypt-2].expr, yyS[yypt-1].expr, yyS[yypt-0].projection)
		}
	case 131:
		//line n1ql.y:1095
		{
			yyVAL.statement = algebra.NewUpdate(yyS[yypt-6].keyspaceRef, yyS[yypt-5].expr, yyS[yypt-4].set, yyS[yypt-3].unset, yyS[yypt-2].expr, yyS[yypt-1].expr, yyS[yypt-0].projection)
		}
	case 132:
		//line n1ql.y:1100
		{
			yyVAL.statement = algebra.NewUpdate(yyS[yypt-5].keyspaceRef, yyS[yypt-4].expr, yyS[yypt-3].set, nil, yyS[yypt-2].expr, yyS[yypt-1].expr, yyS[yypt-0].projection)
		}
	case 133:
		//line n1ql.y:1105
		{
			yyVAL.statement = algebra.NewUpdate(yyS[yypt-5].keyspaceRef, yyS[yypt-4].expr, nil, yyS[yypt-3].unset, yyS[yypt-2].expr, yyS[yypt-1].expr, yyS[yypt-0].projection)
		}
	case 134:
		//line n1ql.y:1112
		{
			yyVAL.set = algebra.NewSet(yyS[yypt-0].setTerms)
		}
	case 135:
		//line n1ql.y:1119
		{
			yyVAL.setTerms = algebra.SetTerms{yyS[yypt-0].setTerm}
		}
	case 136:
		//line n1ql.y:1124
		{
			yyVAL.setTerms = append(yyS[yypt-2].setTerms, yyS[yypt-0].setTerm)
		}
	case 137:
		//line n1ql.y:1131
		{
			yyVAL.setTerm = algebra.NewSetTerm(yyS[yypt-3].path, yyS[yypt-1].expr, yyS[yypt-0].updateFor)
		}
	case 138:
		//line n1ql.y:1138
		{
			yyVAL.updateFor = nil
		}
	case 139:
		yyVAL.updateFor = yyS[yypt-0].updateFor
	case 140:
		//line n1ql.y:1147
		{
			yyVAL.updateFor = algebra.NewUpdateFor(yyS[yypt-2].bindings, yyS[yypt-1].expr)
		}
	case 141:
		//line n1ql.y:1154
		{
			yyVAL.bindings = expression.Bindings{yyS[yypt-0].binding}
		}
	case 142:
		//line n1ql.y:1159
		{
			yyVAL.bindings = append(yyS[yypt-2].bindings, yyS[yypt-0].binding)
		}
	case 143:
		//line n1ql.y:1166
		{
			yyVAL.binding = expression.NewBinding(yyS[yypt-2].s, yyS[yypt-0].expr)
		}
	case 144:
		//line n1ql.y:1171
		{
			yyVAL.binding = expression.NewDescendantBinding(yyS[yypt-2].s, yyS[yypt-0].expr)
		}
	case 145:
		yyVAL.s = yyS[yypt-0].s
	case 146:
		//line n1ql.y:1182
		{
			yyVAL.expr = yyS[yypt-0].path
		}
	case 147:
		//line n1ql.y:1189
		{
			yyVAL.expr = nil
		}
	case 148:
		//line n1ql.y:1194
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 149:
		//line n1ql.y:1201
		{
			yyVAL.unset = algebra.NewUnset(yyS[yypt-0].unsetTerms)
		}
	case 150:
		//line n1ql.y:1208
		{
			yyVAL.unsetTerms = algebra.UnsetTerms{yyS[yypt-0].unsetTerm}
		}
	case 151:
		//line n1ql.y:1213
		{
			yyVAL.unsetTerms = append(yyS[yypt-2].unsetTerms, yyS[yypt-0].unsetTerm)
		}
	case 152:
		//line n1ql.y:1220
		{
			yyVAL.unsetTerm = algebra.NewUnsetTerm(yyS[yypt-1].path, yyS[yypt-0].updateFor)
		}
	case 153:
		//line n1ql.y:1234
		{
			source := algebra.NewMergeSourceFrom(yyS[yypt-5].keyspaceTerm, "")
			yyVAL.statement = algebra.NewMerge(yyS[yypt-7].keyspaceRef, source, yyS[yypt-3].expr, yyS[yypt-2].mergeActions, yyS[yypt-1].expr, yyS[yypt-0].projection)
		}
	case 154:
		//line n1ql.y:1240
		{
			source := algebra.NewMergeSourceSelect(yyS[yypt-7].fullselect, yyS[yypt-5].s)
			yyVAL.statement = algebra.NewMerge(yyS[yypt-10].keyspaceRef, source, yyS[yypt-3].expr, yyS[yypt-2].mergeActions, yyS[yypt-1].expr, yyS[yypt-0].projection)
		}
	case 155:
		//line n1ql.y:1248
		{
			yyVAL.mergeActions = algebra.NewMergeActions(nil, nil, nil)
		}
	case 156:
		//line n1ql.y:1253
		{
			yyVAL.mergeActions = algebra.NewMergeActions(yyS[yypt-1].mergeUpdate, yyS[yypt-0].mergeActions.Delete(), yyS[yypt-0].mergeActions.Insert())
		}
	case 157:
		//line n1ql.y:1258
		{
			yyVAL.mergeActions = algebra.NewMergeActions(nil, yyS[yypt-1].mergeDelete, yyS[yypt-0].mergeInsert)
		}
	case 158:
		//line n1ql.y:1263
		{
			yyVAL.mergeActions = algebra.NewMergeActions(nil, nil, yyS[yypt-0].mergeInsert)
		}
	case 159:
		//line n1ql.y:1270
		{
			yyVAL.mergeActions = algebra.NewMergeActions(nil, nil, nil)
		}
	case 160:
		//line n1ql.y:1275
		{
			yyVAL.mergeActions = algebra.NewMergeActions(nil, yyS[yypt-1].mergeDelete, yyS[yypt-0].mergeInsert)
		}
	case 161:
		//line n1ql.y:1280
		{
			yyVAL.mergeActions = algebra.NewMergeActions(nil, nil, yyS[yypt-0].mergeInsert)
		}
	case 162:
		//line n1ql.y:1287
		{
			yyVAL.mergeInsert = nil
		}
	case 163:
		//line n1ql.y:1292
		{
			yyVAL.mergeInsert = yyS[yypt-0].mergeInsert
		}
	case 164:
		//line n1ql.y:1299
		{
			yyVAL.mergeUpdate = algebra.NewMergeUpdate(yyS[yypt-1].set, nil, yyS[yypt-0].expr)
		}
	case 165:
		//line n1ql.y:1304
		{
			yyVAL.mergeUpdate = algebra.NewMergeUpdate(yyS[yypt-2].set, yyS[yypt-1].unset, yyS[yypt-0].expr)
		}
	case 166:
		//line n1ql.y:1309
		{
			yyVAL.mergeUpdate = algebra.NewMergeUpdate(nil, yyS[yypt-1].unset, yyS[yypt-0].expr)
		}
	case 167:
		//line n1ql.y:1316
		{
			yyVAL.mergeDelete = algebra.NewMergeDelete(yyS[yypt-0].expr)
		}
	case 168:
		//line n1ql.y:1323
		{
			yyVAL.mergeInsert = algebra.NewMergeInsert(yyS[yypt-1].expr, yyS[yypt-0].expr)
		}
	case 169:
		//line n1ql.y:1337
		{
			yyVAL.statement = algebra.NewCreatePrimaryIndex(yyS[yypt-1].keyspaceRef, yyS[yypt-0].indexType)
		}
	case 170:
		//line n1ql.y:1342
		{
			yyVAL.statement = algebra.NewCreateIndex(yyS[yypt-8].s, yyS[yypt-6].keyspaceRef, yyS[yypt-4].exprs, yyS[yypt-2].expr, yyS[yypt-1].expr, yyS[yypt-0].indexType)
		}
	case 171:
		yyVAL.s = yyS[yypt-0].s
	case 172:
		//line n1ql.y:1353
		{
			yyVAL.keyspaceRef = algebra.NewKeyspaceRef("", yyS[yypt-0].s, "")
		}
	case 173:
		//line n1ql.y:1358
		{
			yyVAL.keyspaceRef = algebra.NewKeyspaceRef(yyS[yypt-2].s, yyS[yypt-0].s, "")
		}
	case 174:
		//line n1ql.y:1365
		{
			yyVAL.expr = nil
		}
	case 175:
		//line n1ql.y:1370
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 176:
		//line n1ql.y:1377
		{
			yyVAL.indexType = datastore.VIEW
		}
	case 177:
		//line n1ql.y:1382
		{
			yyVAL.indexType = datastore.VIEW
		}
	case 178:
		//line n1ql.y:1387
		{
			yyVAL.indexType = datastore.LSM
		}
	case 179:
		//line n1ql.y:1394
		{
			yyVAL.exprs = expression.Expressions{yyS[yypt-0].expr}
		}
	case 180:
		//line n1ql.y:1399
		{
			yyVAL.exprs = append(yyS[yypt-2].exprs, yyS[yypt-0].expr)
		}
	case 181:
		//line n1ql.y:1406
		{
			e := yyS[yypt-0].expr
			if !e.Indexable() {
				yylex.Error(fmt.Sprintf("Expression not indexable."))
			}

			yyVAL.expr = e
		}
	case 182:
		//line n1ql.y:1417
		{
			yyVAL.expr = nil
		}
	case 183:
		//line n1ql.y:1422
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 184:
		//line n1ql.y:1436
		{
			yyVAL.statement = algebra.NewDropIndex(yyS[yypt-0].keyspaceRef, "#primary")
		}
	case 185:
		//line n1ql.y:1441
		{
			yyVAL.statement = algebra.NewDropIndex(yyS[yypt-2].keyspaceRef, yyS[yypt-0].s)
		}
	case 186:
		//line n1ql.y:1454
		{
			yyVAL.statement = algebra.NewAlterIndex(yyS[yypt-3].keyspaceRef, yyS[yypt-1].s, yyS[yypt-0].s)
		}
	case 187:
		//line n1ql.y:1460
		{
			yyVAL.s = ""
		}
	case 188:
		//line n1ql.y:1465
		{
			yyVAL.s = yyS[yypt-0].s
		}
	case 189:
		//line n1ql.y:1479
		{
			yyVAL.path = expression.NewIdentifier(yyS[yypt-0].s)
		}
	case 190:
		//line n1ql.y:1484
		{
			yyVAL.path = expression.NewField(yyS[yypt-2].path, expression.NewFieldName(yyS[yypt-0].s))
		}
	case 191:
		//line n1ql.y:1489
		{
			field := expression.NewField(yyS[yypt-2].path, expression.NewFieldName(yyS[yypt-0].s))
			field.SetCaseInsensitive(true)
			yyVAL.path = field
		}
	case 192:
		//line n1ql.y:1496
		{
			yyVAL.path = expression.NewField(yyS[yypt-4].path, yyS[yypt-1].expr)
		}
	case 193:
		//line n1ql.y:1501
		{
			field := expression.NewField(yyS[yypt-4].path, yyS[yypt-1].expr)
			field.SetCaseInsensitive(true)
			yyVAL.path = field
		}
	case 194:
		//line n1ql.y:1508
		{
			yyVAL.path = expression.NewElement(yyS[yypt-3].path, yyS[yypt-1].expr)
		}
	case 195:
		yyVAL.expr = yyS[yypt-0].expr
	case 196:
		//line n1ql.y:1525
		{
			yyVAL.expr = expression.NewField(yyS[yypt-2].expr, expression.NewFieldName(yyS[yypt-0].s))
		}
	case 197:
		//line n1ql.y:1530
		{
			field := expression.NewField(yyS[yypt-2].expr, expression.NewFieldName(yyS[yypt-0].s))
			field.SetCaseInsensitive(true)
			yyVAL.expr = field
		}
	case 198:
		//line n1ql.y:1537
		{
			yyVAL.expr = expression.NewField(yyS[yypt-4].expr, yyS[yypt-1].expr)
		}
	case 199:
		//line n1ql.y:1542
		{
			field := expression.NewField(yyS[yypt-4].expr, yyS[yypt-1].expr)
			field.SetCaseInsensitive(true)
			yyVAL.expr = field
		}
	case 200:
		//line n1ql.y:1549
		{
			yyVAL.expr = expression.NewElement(yyS[yypt-3].expr, yyS[yypt-1].expr)
		}
	case 201:
		//line n1ql.y:1554
		{
			yyVAL.expr = expression.NewSlice(yyS[yypt-4].expr, yyS[yypt-2].expr)
		}
	case 202:
		//line n1ql.y:1559
		{
			yyVAL.expr = expression.NewSlice(yyS[yypt-5].expr, yyS[yypt-3].expr, yyS[yypt-1].expr)
		}
	case 203:
		//line n1ql.y:1565
		{
			yyVAL.expr = expression.NewAdd(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 204:
		//line n1ql.y:1570
		{
			yyVAL.expr = expression.NewSub(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 205:
		//line n1ql.y:1575
		{
			yyVAL.expr = expression.NewMult(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 206:
		//line n1ql.y:1580
		{
			yyVAL.expr = expression.NewDiv(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 207:
		//line n1ql.y:1585
		{
			yyVAL.expr = expression.NewMod(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 208:
		//line n1ql.y:1591
		{
			yyVAL.expr = expression.NewConcat(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 209:
		//line n1ql.y:1597
		{
			yyVAL.expr = expression.NewAnd(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 210:
		//line n1ql.y:1602
		{
			yyVAL.expr = expression.NewOr(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 211:
		//line n1ql.y:1607
		{
			yyVAL.expr = expression.NewNot(yyS[yypt-0].expr)
		}
	case 212:
		//line n1ql.y:1613
		{
			yyVAL.expr = expression.NewEq(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 213:
		//line n1ql.y:1618
		{
			yyVAL.expr = expression.NewEq(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 214:
		//line n1ql.y:1623
		{
			yyVAL.expr = expression.NewNE(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 215:
		//line n1ql.y:1628
		{
			yyVAL.expr = expression.NewLT(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 216:
		//line n1ql.y:1633
		{
			yyVAL.expr = expression.NewGT(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 217:
		//line n1ql.y:1638
		{
			yyVAL.expr = expression.NewLE(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 218:
		//line n1ql.y:1643
		{
			yyVAL.expr = expression.NewGE(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 219:
		//line n1ql.y:1648
		{
			yyVAL.expr = expression.NewBetween(yyS[yypt-4].expr, yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 220:
		//line n1ql.y:1653
		{
			yyVAL.expr = expression.NewNotBetween(yyS[yypt-5].expr, yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 221:
		//line n1ql.y:1658
		{
			yyVAL.expr = expression.NewLike(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 222:
		//line n1ql.y:1663
		{
			yyVAL.expr = expression.NewNotLike(yyS[yypt-3].expr, yyS[yypt-0].expr)
		}
	case 223:
		//line n1ql.y:1668
		{
			yyVAL.expr = expression.NewIn(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 224:
		//line n1ql.y:1673
		{
			yyVAL.expr = expression.NewNotIn(yyS[yypt-3].expr, yyS[yypt-0].expr)
		}
	case 225:
		//line n1ql.y:1678
		{
			yyVAL.expr = expression.NewWithin(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 226:
		//line n1ql.y:1683
		{
			yyVAL.expr = expression.NewNotWithin(yyS[yypt-3].expr, yyS[yypt-0].expr)
		}
	case 227:
		//line n1ql.y:1688
		{
			yyVAL.expr = expression.NewIsNull(yyS[yypt-2].expr)
		}
	case 228:
		//line n1ql.y:1693
		{
			yyVAL.expr = expression.NewIsNotNull(yyS[yypt-3].expr)
		}
	case 229:
		//line n1ql.y:1698
		{
			yyVAL.expr = expression.NewIsMissing(yyS[yypt-2].expr)
		}
	case 230:
		//line n1ql.y:1703
		{
			yyVAL.expr = expression.NewIsNotMissing(yyS[yypt-3].expr)
		}
	case 231:
		//line n1ql.y:1708
		{
			yyVAL.expr = expression.NewIsValued(yyS[yypt-2].expr)
		}
	case 232:
		//line n1ql.y:1713
		{
			yyVAL.expr = expression.NewIsNotValued(yyS[yypt-3].expr)
		}
	case 233:
		//line n1ql.y:1718
		{
			yyVAL.expr = expression.NewExists(yyS[yypt-0].expr)
		}
	case 234:
		yyVAL.expr = yyS[yypt-0].expr
	case 235:
		yyVAL.expr = yyS[yypt-0].expr
	case 236:
		//line n1ql.y:1732
		{
			yyVAL.expr = expression.NewIdentifier(yyS[yypt-0].s)
		}
	case 237:
		yyVAL.expr = yyS[yypt-0].expr
	case 238:
		yyVAL.expr = yyS[yypt-0].expr
	case 239:
		//line n1ql.y:1744
		{
			yyVAL.expr = expression.NewNeg(yyS[yypt-0].expr)
		}
	case 240:
		yyVAL.expr = yyS[yypt-0].expr
	case 241:
		yyVAL.expr = yyS[yypt-0].expr
	case 242:
		yyVAL.expr = yyS[yypt-0].expr
	case 243:
		yyVAL.expr = yyS[yypt-0].expr
	case 244:
		//line n1ql.y:1763
		{
			yyVAL.expr = expression.NewField(yyS[yypt-2].expr, expression.NewFieldName(yyS[yypt-0].s))
		}
	case 245:
		//line n1ql.y:1768
		{
			field := expression.NewField(yyS[yypt-2].expr, expression.NewFieldName(yyS[yypt-0].s))
			field.SetCaseInsensitive(true)
			yyVAL.expr = field
		}
	case 246:
		//line n1ql.y:1775
		{
			yyVAL.expr = expression.NewField(yyS[yypt-4].expr, yyS[yypt-1].expr)
		}
	case 247:
		//line n1ql.y:1780
		{
			field := expression.NewField(yyS[yypt-4].expr, yyS[yypt-1].expr)
			field.SetCaseInsensitive(true)
			yyVAL.expr = field
		}
	case 248:
		//line n1ql.y:1787
		{
			yyVAL.expr = expression.NewElement(yyS[yypt-3].expr, yyS[yypt-1].expr)
		}
	case 249:
		//line n1ql.y:1792
		{
			yyVAL.expr = expression.NewSlice(yyS[yypt-4].expr, yyS[yypt-2].expr)
		}
	case 250:
		//line n1ql.y:1797
		{
			yyVAL.expr = expression.NewSlice(yyS[yypt-5].expr, yyS[yypt-3].expr, yyS[yypt-1].expr)
		}
	case 251:
		//line n1ql.y:1803
		{
			yyVAL.expr = expression.NewAdd(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 252:
		//line n1ql.y:1808
		{
			yyVAL.expr = expression.NewSub(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 253:
		//line n1ql.y:1813
		{
			yyVAL.expr = expression.NewMult(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 254:
		//line n1ql.y:1818
		{
			yyVAL.expr = expression.NewDiv(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 255:
		//line n1ql.y:1823
		{
			yyVAL.expr = expression.NewMod(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 256:
		//line n1ql.y:1829
		{
			yyVAL.expr = expression.NewConcat(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 257:
		//line n1ql.y:1843
		{
			yyVAL.expr = expression.NULL_EXPR
		}
	case 258:
		//line n1ql.y:1848
		{
			yyVAL.expr = expression.MISSING_EXPR
		}
	case 259:
		//line n1ql.y:1853
		{
			yyVAL.expr = expression.FALSE_EXPR
		}
	case 260:
		//line n1ql.y:1858
		{
			yyVAL.expr = expression.TRUE_EXPR
		}
	case 261:
		//line n1ql.y:1863
		{
			yyVAL.expr = expression.NewConstant(value.NewValue(yyS[yypt-0].f))
		}
	case 262:
		//line n1ql.y:1868
		{
			yyVAL.expr = expression.NewConstant(value.NewValue(yyS[yypt-0].n))
		}
	case 263:
		//line n1ql.y:1873
		{
			yyVAL.expr = expression.NewConstant(value.NewValue(yyS[yypt-0].s))
		}
	case 264:
		yyVAL.expr = yyS[yypt-0].expr
	case 265:
		yyVAL.expr = yyS[yypt-0].expr
	case 266:
		//line n1ql.y:1893
		{
			yyVAL.expr = expression.NewObjectConstruct(yyS[yypt-1].bindings)
		}
	case 267:
		//line n1ql.y:1900
		{
			yyVAL.bindings = nil
		}
	case 268:
		yyVAL.bindings = yyS[yypt-0].bindings
	case 269:
		//line n1ql.y:1909
		{
			yyVAL.bindings = expression.Bindings{yyS[yypt-0].binding}
		}
	case 270:
		//line n1ql.y:1914
		{
			yyVAL.bindings = append(yyS[yypt-2].bindings, yyS[yypt-0].binding)
		}
	case 271:
		//line n1ql.y:1921
		{
			yyVAL.binding = expression.NewBinding(yyS[yypt-2].s, yyS[yypt-0].expr)
		}
	case 272:
		//line n1ql.y:1928
		{
			yyVAL.expr = expression.NewArrayConstruct(yyS[yypt-1].exprs...)
		}
	case 273:
		//line n1ql.y:1935
		{
			yyVAL.exprs = nil
		}
	case 274:
		yyVAL.exprs = yyS[yypt-0].exprs
	case 275:
		//line n1ql.y:1951
		{
			yyVAL.expr = algebra.NewNamedParameter(yyS[yypt-0].s)
		}
	case 276:
		//line n1ql.y:1956
		{
			yyVAL.expr = algebra.NewPositionalParameter(yyS[yypt-0].n)
		}
	case 277:
		//line n1ql.y:1970
		{
			yyVAL.expr = yyS[yypt-1].expr
		}
	case 278:
		yyVAL.expr = yyS[yypt-0].expr
	case 279:
		yyVAL.expr = yyS[yypt-0].expr
	case 280:
		//line n1ql.y:1983
		{
			yyVAL.expr = expression.NewSimpleCase(yyS[yypt-2].expr, yyS[yypt-1].whenTerms, yyS[yypt-0].expr)
		}
	case 281:
		//line n1ql.y:1990
		{
			yyVAL.whenTerms = expression.WhenTerms{&expression.WhenTerm{yyS[yypt-2].expr, yyS[yypt-0].expr}}
		}
	case 282:
		//line n1ql.y:1995
		{
			yyVAL.whenTerms = append(yyS[yypt-4].whenTerms, &expression.WhenTerm{yyS[yypt-2].expr, yyS[yypt-0].expr})
		}
	case 283:
		//line n1ql.y:2003
		{
			yyVAL.expr = expression.NewSearchedCase(yyS[yypt-1].whenTerms, yyS[yypt-0].expr)
		}
	case 284:
		//line n1ql.y:2010
		{
			yyVAL.expr = nil
		}
	case 285:
		//line n1ql.y:2015
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 286:
		//line n1ql.y:2029
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
	case 287:
		//line n1ql.y:2048
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
	case 288:
		//line n1ql.y:2063
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
	case 289:
		yyVAL.s = yyS[yypt-0].s
	case 290:
		yyVAL.expr = yyS[yypt-0].expr
	case 291:
		yyVAL.expr = yyS[yypt-0].expr
	case 292:
		//line n1ql.y:2097
		{
			yyVAL.expr = expression.NewAny(yyS[yypt-2].bindings, yyS[yypt-1].expr)
		}
	case 293:
		//line n1ql.y:2102
		{
			yyVAL.expr = expression.NewAny(yyS[yypt-2].bindings, yyS[yypt-1].expr)
		}
	case 294:
		//line n1ql.y:2107
		{
			yyVAL.expr = expression.NewEvery(yyS[yypt-2].bindings, yyS[yypt-1].expr)
		}
	case 295:
		//line n1ql.y:2114
		{
			yyVAL.bindings = expression.Bindings{yyS[yypt-0].binding}
		}
	case 296:
		//line n1ql.y:2119
		{
			yyVAL.bindings = append(yyS[yypt-2].bindings, yyS[yypt-0].binding)
		}
	case 297:
		//line n1ql.y:2126
		{
			yyVAL.binding = expression.NewBinding(yyS[yypt-2].s, yyS[yypt-0].expr)
		}
	case 298:
		//line n1ql.y:2131
		{
			yyVAL.binding = expression.NewDescendantBinding(yyS[yypt-2].s, yyS[yypt-0].expr)
		}
	case 299:
		//line n1ql.y:2138
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 300:
		//line n1ql.y:2145
		{
			yyVAL.expr = expression.NewArray(yyS[yypt-4].expr, yyS[yypt-2].bindings, yyS[yypt-1].expr)
		}
	case 301:
		//line n1ql.y:2150
		{
			yyVAL.expr = expression.NewFirst(yyS[yypt-4].expr, yyS[yypt-2].bindings, yyS[yypt-1].expr)
		}
	case 302:
		//line n1ql.y:2164
		{
			yyVAL.expr = yyS[yypt-1].expr
		}
	case 303:
		yyVAL.expr = yyS[yypt-0].expr
	case 304:
		//line n1ql.y:2173
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
