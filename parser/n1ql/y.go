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
const LSM = 57488
const WHEN = 57489
const WHERE = 57490
const WHILE = 57491
const WITH = 57492
const WITHIN = 57493
const WORK = 57494
const XOR = 57495
const INT = 57496
const NUMBER = 57497
const STRING = 57498
const IDENTIFIER = 57499
const IDENTIFIER_ICASE = 57500
const LPAREN = 57501
const RPAREN = 57502
const LBRACE = 57503
const RBRACE = 57504
const LBRACKET = 57505
const RBRACKET = 57506
const RBRACKET_ICASE = 57507
const COMMA = 57508
const COLON = 57509
const INTERESECT = 57510
const EQ = 57511
const DEQ = 57512
const NE = 57513
const LT = 57514
const GT = 57515
const LE = 57516
const GE = 57517
const CONCAT = 57518
const PLUS = 57519
const STAR = 57520
const DIV = 57521
const MOD = 57522
const UMINUS = 57523
const DOT = 57524

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
	"LSM",
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
	159, 277,
	-2, 228,
	-1, 104,
	167, 63,
	-2, 64,
	-1, 140,
	50, 72,
	67, 72,
	85, 72,
	135, 72,
	-2, 50,
	-1, 167,
	169, 0,
	170, 0,
	171, 0,
	-2, 205,
	-1, 168,
	169, 0,
	170, 0,
	171, 0,
	-2, 206,
	-1, 169,
	169, 0,
	170, 0,
	171, 0,
	-2, 207,
	-1, 170,
	172, 0,
	173, 0,
	174, 0,
	175, 0,
	-2, 208,
	-1, 171,
	172, 0,
	173, 0,
	174, 0,
	175, 0,
	-2, 209,
	-1, 172,
	172, 0,
	173, 0,
	174, 0,
	175, 0,
	-2, 210,
	-1, 173,
	172, 0,
	173, 0,
	174, 0,
	175, 0,
	-2, 211,
	-1, 180,
	75, 0,
	-2, 214,
	-1, 181,
	58, 0,
	151, 0,
	-2, 216,
	-1, 182,
	58, 0,
	151, 0,
	-2, 218,
	-1, 275,
	75, 0,
	-2, 215,
	-1, 276,
	58, 0,
	151, 0,
	-2, 217,
	-1, 277,
	58, 0,
	151, 0,
	-2, 219,
}

const yyNprod = 293
const yyPrivate = 57344

var yyTokenNames []string
var yyStates []string

const yyLast = 2661

var yyAct = []int{

	154, 3, 567, 554, 427, 565, 555, 296, 297, 187,
	88, 89, 504, 513, 292, 204, 386, 521, 300, 206,
	129, 480, 244, 91, 441, 251, 205, 221, 388, 402,
	289, 200, 410, 146, 385, 149, 125, 245, 330, 374,
	12, 223, 150, 141, 128, 110, 127, 102, 114, 252,
	62, 122, 317, 48, 126, 216, 8, 443, 133, 134,
	315, 335, 488, 457, 418, 267, 456, 158, 159, 160,
	161, 162, 163, 164, 165, 166, 167, 168, 169, 170,
	171, 172, 173, 417, 266, 180, 181, 182, 115, 87,
	316, 507, 254, 267, 253, 229, 269, 231, 418, 66,
	66, 234, 131, 132, 439, 68, 203, 126, 270, 271,
	272, 143, 266, 218, 69, 70, 71, 417, 65, 65,
	367, 403, 403, 334, 155, 156, 255, 440, 269, 438,
	157, 369, 231, 230, 228, 227, 483, 459, 460, 175,
	501, 233, 241, 461, 103, 351, 308, 190, 192, 194,
	259, 233, 144, 306, 231, 219, 448, 246, 262, 357,
	358, 226, 155, 156, 207, 359, 225, 225, 157, 418,
	261, 106, 346, 413, 144, 130, 208, 267, 275, 276,
	277, 256, 258, 257, 222, 104, 66, 393, 417, 303,
	284, 268, 270, 271, 272, 269, 266, 290, 104, 72,
	67, 69, 70, 71, 123, 65, 104, 104, 142, 267,
	243, 566, 307, 294, 502, 235, 310, 299, 311, 561,
	174, 505, 273, 268, 270, 271, 272, 304, 266, 243,
	319, 295, 320, 175, 217, 323, 324, 325, 548, 265,
	558, 559, 305, 202, 333, 285, 279, 286, 135, 287,
	278, 544, 485, 580, 336, 63, 298, 185, 350, 579,
	184, 183, 64, 224, 224, 355, 341, 344, 360, 345,
	309, 575, 545, 299, 535, 195, 267, 429, 322, 450,
	318, 301, 529, 337, 368, 343, 328, 329, 176, 273,
	268, 270, 271, 272, 377, 266, 232, 63, 349, 124,
	514, 338, 380, 382, 383, 381, 280, 503, 236, 527,
	111, 208, 445, 396, 269, 314, 389, 186, 391, 193,
	68, 64, 175, 63, 313, 175, 175, 175, 175, 175,
	175, 283, 375, 178, 378, 379, 408, 117, 573, 570,
	415, 215, 577, 399, 576, 401, 571, 376, 302, 392,
	177, 340, 536, 143, 246, 397, 398, 543, 189, 404,
	422, 225, 225, 64, 137, 191, 540, 63, 390, 249,
	290, 414, 407, 419, 420, 409, 416, 431, 116, 250,
	430, 405, 293, 432, 433, 412, 412, 247, 435, 64,
	434, 444, 436, 437, 105, 267, 447, 274, 347, 348,
	426, 66, 99, 98, 452, 583, 210, 126, 273, 268,
	270, 271, 272, 63, 266, 67, 69, 70, 71, 462,
	65, 214, 582, 525, 175, 468, 179, 556, 237, 238,
	526, 458, 220, 64, 446, 463, 464, 455, 61, 472,
	477, 474, 475, 454, 119, 473, 118, 95, 511, 126,
	142, 332, 63, 100, 519, 453, 451, 389, 224, 224,
	482, 400, 492, 470, 481, 471, 101, 327, 94, 478,
	326, 489, 497, 476, 321, 213, 578, 539, 498, 64,
	406, 196, 411, 411, 342, 484, 356, 2, 339, 361,
	362, 363, 364, 365, 366, 494, 495, 97, 1, 90,
	449, 139, 499, 547, 528, 551, 560, 442, 246, 500,
	506, 512, 530, 508, 524, 387, 515, 516, 384, 522,
	522, 523, 481, 520, 479, 428, 469, 291, 34, 534,
	73, 532, 533, 531, 33, 538, 82, 93, 32, 18,
	549, 550, 537, 17, 16, 15, 541, 542, 14, 13,
	546, 552, 553, 7, 6, 5, 557, 568, 4, 562,
	564, 563, 569, 73, 370, 197, 198, 199, 371, 82,
	572, 281, 209, 282, 188, 574, 288, 92, 96, 145,
	510, 85, 581, 568, 568, 585, 586, 584, 425, 87,
	509, 487, 486, 73, 331, 242, 207, 136, 84, 82,
	201, 490, 491, 248, 138, 68, 140, 59, 60, 83,
	26, 113, 25, 43, 85, 74, 21, 46, 45, 24,
	109, 108, 87, 107, 23, 120, 121, 42, 41, 19,
	0, 84, 0, 0, 0, 0, 0, 0, 68, 0,
	0, 0, 83, 0, 85, 0, 0, 0, 74, 0,
	0, 0, 87, 0, 0, 0, 0, 0, 0, 0,
	0, 84, 0, 0, 0, 0, 0, 0, 68, 0,
	0, 0, 83, 0, 86, 0, 0, 0, 74, 0,
	0, 0, 0, 0, 0, 0, 66, 517, 518, 0,
	0, 0, 75, 76, 77, 78, 79, 80, 81, 72,
	67, 69, 70, 71, 0, 65, 0, 86, 0, 0,
	0, 0, 0, 0, 0, 0, 73, 0, 0, 66,
	465, 466, 82, 0, 0, 75, 76, 77, 78, 79,
	80, 81, 72, 67, 69, 70, 71, 86, 65, 0,
	0, 0, 0, 208, 0, 0, 73, 0, 0, 66,
	372, 0, 82, 0, 0, 75, 76, 77, 78, 79,
	80, 81, 72, 67, 69, 70, 71, 85, 65, 0,
	0, 0, 373, 0, 0, 87, 73, 0, 0, 0,
	0, 0, 82, 0, 84, 0, 0, 0, 0, 0,
	0, 68, 0, 0, 51, 83, 0, 85, 0, 0,
	0, 74, 0, 0, 0, 87, 0, 0, 0, 0,
	0, 0, 0, 0, 84, 49, 0, 0, 0, 0,
	29, 68, 0, 0, 0, 83, 50, 85, 0, 0,
	0, 74, 0, 0, 0, 87, 11, 0, 0, 0,
	0, 63, 0, 0, 84, 0, 0, 0, 0, 0,
	0, 68, 27, 82, 0, 83, 0, 0, 0, 0,
	86, 74, 0, 0, 0, 0, 0, 0, 0, 0,
	31, 0, 66, 423, 0, 0, 424, 0, 75, 76,
	77, 78, 79, 80, 81, 72, 67, 69, 70, 71,
	86, 65, 0, 0, 0, 0, 0, 0, 85, 73,
	0, 0, 66, 0, 0, 82, 87, 64, 75, 76,
	77, 78, 79, 80, 81, 72, 67, 69, 70, 71,
	86, 65, 68, 0, 0, 0, 30, 28, 0, 0,
	0, 0, 66, 352, 353, 0, 0, 0, 75, 76,
	77, 78, 79, 80, 81, 72, 67, 69, 70, 71,
	85, 65, 0, 73, 0, 0, 207, 0, 87, 82,
	0, 0, 0, 0, 0, 0, 0, 84, 0, 0,
	0, 0, 0, 0, 68, 0, 0, 0, 83, 0,
	0, 0, 0, 0, 74, 0, 0, 0, 0, 0,
	0, 86, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 66, 85, 0, 0, 0, 0, 0,
	0, 0, 87, 0, 0, 0, 72, 67, 69, 70,
	71, 84, 65, 0, 0, 73, 0, 0, 68, 0,
	0, 82, 83, 0, 0, 0, 0, 0, 74, 0,
	0, 0, 0, 86, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 66, 263, 0, 0, 264,
	0, 75, 76, 77, 78, 79, 80, 81, 72, 67,
	69, 70, 71, 0, 65, 0, 85, 0, 0, 0,
	0, 0, 0, 73, 87, 0, 0, 0, 0, 82,
	0, 0, 0, 84, 0, 0, 0, 86, 0, 0,
	68, 0, 0, 208, 83, 0, 0, 0, 0, 66,
	74, 0, 0, 0, 0, 75, 76, 77, 78, 79,
	80, 81, 72, 67, 69, 70, 71, 443, 260, 0,
	0, 0, 0, 0, 85, 0, 0, 0, 0, 0,
	0, 0, 87, 0, 0, 0, 0, 0, 0, 0,
	73, 84, 0, 0, 0, 0, 82, 0, 68, 0,
	0, 0, 83, 0, 0, 0, 243, 0, 74, 86,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 66, 0, 0, 0, 0, 0, 75, 76, 77,
	78, 79, 80, 81, 72, 67, 69, 70, 71, 0,
	65, 85, 0, 0, 0, 0, 73, 0, 0, 87,
	0, 0, 82, 0, 0, 0, 0, 0, 84, 0,
	0, 0, 0, 0, 0, 68, 0, 86, 0, 83,
	0, 0, 0, 0, 0, 74, 0, 0, 0, 66,
	0, 0, 0, 0, 0, 75, 76, 77, 78, 79,
	80, 81, 72, 67, 69, 70, 71, 85, 65, 0,
	0, 0, 0, 0, 0, 87, 0, 0, 0, 0,
	0, 0, 0, 73, 84, 0, 0, 0, 0, 82,
	0, 68, 0, 0, 0, 83, 0, 0, 0, 0,
	0, 74, 0, 0, 86, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 66, 496, 0, 0,
	0, 0, 75, 76, 77, 78, 79, 80, 81, 72,
	67, 69, 70, 71, 85, 65, 0, 0, 0, 73,
	0, 0, 87, 0, 0, 82, 0, 0, 0, 0,
	0, 84, 0, 0, 0, 0, 0, 0, 68, 0,
	86, 0, 83, 0, 0, 0, 0, 0, 74, 0,
	0, 0, 66, 493, 0, 0, 0, 0, 75, 76,
	77, 78, 79, 80, 81, 72, 67, 69, 70, 71,
	85, 65, 0, 0, 0, 0, 0, 0, 87, 0,
	0, 0, 0, 0, 0, 0, 0, 84, 0, 73,
	0, 0, 0, 0, 68, 82, 0, 0, 83, 0,
	0, 0, 0, 0, 74, 0, 0, 86, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 66,
	421, 0, 0, 0, 0, 75, 76, 77, 78, 79,
	80, 81, 72, 67, 69, 70, 71, 395, 65, 0,
	85, 0, 0, 0, 0, 73, 0, 0, 87, 0,
	0, 82, 0, 0, 0, 0, 0, 84, 0, 0,
	0, 0, 0, 86, 68, 0, 0, 0, 83, 0,
	0, 0, 0, 0, 74, 66, 0, 0, 0, 0,
	0, 75, 76, 77, 78, 79, 80, 81, 72, 67,
	69, 70, 71, 0, 65, 0, 85, 0, 0, 0,
	0, 0, 0, 0, 87, 0, 0, 0, 0, 0,
	0, 0, 0, 84, 0, 0, 0, 0, 0, 0,
	68, 82, 0, 0, 83, 0, 0, 0, 0, 0,
	74, 0, 0, 86, 0, 0, 0, 0, 0, 0,
	0, 0, 394, 0, 0, 66, 0, 0, 0, 0,
	0, 75, 76, 77, 78, 79, 80, 81, 72, 67,
	69, 70, 71, 312, 65, 0, 85, 0, 0, 0,
	0, 0, 0, 0, 87, 73, 0, 0, 0, 0,
	0, 82, 0, 84, 0, 0, 0, 0, 0, 86,
	68, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 66, 0, 0, 0, 0, 0, 75, 76, 77,
	78, 79, 80, 81, 72, 67, 69, 70, 71, 240,
	65, 73, 0, 0, 0, 0, 85, 82, 0, 0,
	0, 0, 0, 0, 87, 0, 0, 0, 0, 0,
	0, 0, 0, 84, 0, 0, 0, 0, 0, 0,
	68, 0, 0, 0, 83, 0, 0, 0, 0, 86,
	74, 0, 0, 0, 0, 239, 0, 0, 0, 0,
	0, 66, 85, 0, 0, 0, 0, 0, 0, 0,
	87, 0, 0, 0, 72, 67, 69, 70, 71, 84,
	65, 0, 0, 0, 54, 57, 68, 0, 0, 0,
	83, 0, 0, 0, 44, 0, 74, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 86,
	0, 211, 0, 0, 0, 0, 0, 0, 56, 0,
	0, 66, 10, 0, 36, 58, 0, 75, 76, 77,
	78, 79, 80, 81, 72, 67, 69, 70, 71, 73,
	65, 0, 0, 0, 0, 82, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 86, 0, 0, 22, 0,
	0, 0, 9, 35, 0, 0, 0, 66, 0, 0,
	0, 0, 0, 75, 76, 77, 78, 79, 80, 81,
	72, 67, 69, 70, 71, 73, 65, 0, 0, 0,
	85, 82, 0, 0, 0, 0, 0, 55, 87, 0,
	0, 0, 0, 0, 0, 37, 0, 84, 0, 0,
	0, 0, 0, 0, 68, 0, 0, 73, 83, 0,
	0, 0, 0, 82, 74, 0, 0, 0, 0, 0,
	39, 38, 40, 20, 0, 47, 85, 52, 0, 53,
	0, 0, 0, 0, 87, 0, 0, 0, 0, 0,
	0, 0, 0, 84, 212, 0, 0, 0, 0, 0,
	68, 0, 0, 0, 83, 0, 0, 0, 85, 0,
	74, 82, 0, 0, 0, 0, 87, 0, 0, 112,
	0, 0, 0, 86, 0, 84, 0, 0, 0, 0,
	0, 0, 68, 0, 0, 66, 83, 0, 0, 0,
	0, 75, 76, 77, 78, 79, 80, 81, 72, 67,
	69, 70, 71, 0, 65, 0, 85, 0, 0, 0,
	0, 0, 0, 0, 87, 0, 0, 0, 0, 86,
	0, 0, 0, 84, 0, 0, 0, 0, 0, 0,
	68, 66, 0, 0, 83, 0, 0, 75, 76, 77,
	78, 79, 80, 81, 72, 67, 69, 70, 71, 0,
	65, 86, 0, 148, 0, 0, 0, 54, 57, 0,
	0, 0, 0, 66, 0, 0, 0, 44, 0, 75,
	76, 77, 78, 79, 80, 81, 72, 67, 69, 70,
	71, 0, 65, 0, 147, 0, 0, 0, 152, 0,
	0, 56, 0, 0, 0, 10, 0, 36, 58, 86,
	0, 0, 0, 54, 57, 82, 0, 0, 0, 0,
	0, 66, 0, 44, 0, 0, 0, 75, 76, 77,
	78, 79, 80, 81, 72, 67, 69, 70, 71, 0,
	65, 22, 0, 0, 152, 9, 35, 56, 0, 0,
	0, 10, 0, 36, 58, 0, 0, 0, 0, 0,
	85, 0, 0, 0, 0, 151, 0, 0, 87, 0,
	0, 0, 0, 0, 0, 0, 0, 84, 0, 0,
	55, 0, 0, 0, 68, 0, 0, 22, 37, 0,
	0, 9, 35, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 151, 0, 39, 38, 40, 20, 0, 47, 0,
	52, 0, 53, 0, 0, 0, 55, 0, 0, 54,
	57, 0, 0, 0, 37, 0, 0, 153, 0, 44,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 86, 0, 0, 0, 0, 0, 39,
	38, 40, 20, 56, 47, 66, 52, 10, 53, 36,
	58, 0, 0, 0, 78, 79, 80, 81, 72, 67,
	69, 70, 71, 153, 65, 51, 0, 0, 54, 57,
	0, 0, 0, 0, 0, 0, 0, 0, 44, 0,
	0, 0, 0, 22, 0, 0, 49, 9, 35, 0,
	0, 29, 0, 0, 0, 0, 0, 50, 0, 0,
	0, 0, 56, 0, 0, 0, 10, 11, 36, 58,
	0, 0, 63, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 55, 27, 0, 54, 57, 0, 0, 0,
	37, 0, 0, 0, 0, 44, 0, 0, 0, 0,
	0, 31, 22, 0, 0, 0, 9, 35, 0, 0,
	0, 0, 0, 0, 0, 39, 38, 40, 20, 56,
	47, 0, 52, 10, 53, 36, 58, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 64, 153,
	0, 55, 0, 0, 54, 57, 0, 0, 0, 37,
	0, 0, 0, 0, 44, 0, 0, 30, 28, 22,
	0, 0, 0, 9, 35, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 39, 38, 40, 20, 56, 47,
	0, 52, 10, 53, 36, 58, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 55, 0,
	0, 0, 0, 0, 0, 0, 37, 0, 0, 54,
	57, 0, 0, 0, 0, 0, 0, 0, 22, 44,
	0, 0, 9, 35, 0, 0, 0, 0, 0, 54,
	57, 39, 38, 40, 20, 0, 47, 0, 52, 44,
	53, 467, 0, 56, 0, 0, 0, 10, 0, 36,
	58, 0, 0, 63, 0, 0, 0, 55, 0, 0,
	0, 0, 0, 56, 0, 37, 0, 10, 0, 36,
	58, 0, 0, 0, 0, 54, 57, 0, 0, 0,
	0, 0, 0, 22, 0, 44, 0, 9, 35, 0,
	39, 38, 40, 20, 0, 47, 0, 52, 0, 53,
	354, 0, 0, 22, 0, 0, 0, 9, 35, 56,
	0, 0, 0, 10, 0, 36, 58, 0, 0, 64,
	0, 0, 55, 0, 0, 54, 57, 0, 0, 0,
	37, 0, 0, 0, 0, 44, 0, 0, 0, 0,
	0, 0, 55, 0, 0, 0, 0, 0, 0, 22,
	37, 0, 0, 9, 35, 39, 38, 40, 20, 56,
	47, 0, 52, 0, 53, 36, 58, 0, 112, 0,
	0, 0, 0, 0, 0, 39, 38, 40, 20, 0,
	47, 0, 52, 0, 53, 0, 0, 0, 55, 0,
	0, 0, 0, 0, 0, 0, 37, 0, 0, 22,
	0, 0, 0, 0, 35, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 39, 38, 40, 20, 0, 47, 0, 52, 0,
	53, 0, 0, 0, 0, 0, 0, 0, 55, 0,
	0, 0, 0, 0, 0, 0, 37, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 39, 38, 40, 20, 0, 47, 0, 52, 0,
	53,
}
var yyPact = []int{

	2200, -1000, -1000, 1798, -1000, -1000, -1000, -1000, -1000, 2447,
	2447, 789, -1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000,
	-1000, -1000, 2447, -1000, -1000, -1000, 404, 338, 337, 401,
	41, 329, -1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000,
	-1000, -1000, -1000, 12, 2401, -1000, -1000, 2381, -1000, 277,
	386, 384, 48, 2447, 18, 18, 18, 2447, 2447, -1000,
	-1000, 291, 400, 50, 1979, 5, 2447, 2447, 2447, 2447,
	2447, 2447, 2447, 2447, 2447, 2447, 2447, 2447, 2447, 2447,
	2447, 2447, 2497, 275, 2447, 2447, 2447, 174, 1878, 23,
	-1000, -63, 282, 361, 315, 271, -1000, 465, 41, 41,
	41, 104, -61, 154, -1000, 41, 1696, 434, -1000, -1000,
	1752, 194, 2447, -5, 1798, -1000, 372, 27, 41, 41,
	-27, -32, -1000, -72, -31, -34, 1798, -15, -1000, 157,
	-1000, -15, -15, 1624, 1578, 62, -1000, 19, 291, -1000,
	307, -1000, -133, -73, -75, -1000, -40, 2025, 2141, 2447,
	-1000, -1000, -1000, -1000, 946, -1000, -1000, 2447, 892, -64,
	-64, -63, -63, -63, 238, 1878, 1830, 2022, 2022, 2022,
	1518, 1518, 1518, 1518, 232, -1000, 2497, 2447, 2447, 2447,
	840, 23, 23, -1000, 163, -1000, -1000, 242, -1000, 2447,
	-1000, 245, -1000, 245, -1000, 245, 2447, 314, 314, 104,
	137, -1000, 180, 32, -1000, -1000, -1000, 19, -1000, 101,
	-7, 2447, -14, -1000, 194, 2447, -1000, 2447, 1448, -1000,
	234, 225, -1000, -122, -1000, -77, -130, -1000, 48, 2447,
	-1000, 2447, 433, 18, 2447, 2447, 2447, 429, 426, 18,
	18, 396, -1000, 2447, -43, -1000, -108, 62, 216, -1000,
	191, 154, 15, 32, 32, 2141, -40, 2447, -40, 1798,
	-33, -1000, 769, -1000, 2316, 2497, 2, 2447, 2497, 2497,
	2497, 2497, 2497, 2497, 113, 840, 23, 23, -1000, -1000,
	-1000, -1000, -1000, 2447, 1798, -1000, -1000, -1000, -35, -1000,
	739, 203, -1000, 2447, 203, 62, 81, 62, 15, 15,
	299, -1000, 154, -1000, -1000, 28, -1000, 1392, -1000, -1000,
	1322, 1798, 2447, 41, 41, 27, 32, 27, -1000, 1798,
	1798, -1000, -1000, 1798, 1798, 1798, -1000, -1000, -25, -25,
	144, -1000, 464, 1798, 19, 2447, 396, 49, 49, 2447,
	-1000, -1000, -1000, -1000, 104, -99, -1000, -133, -133, -1000,
	1798, -1000, -1000, -1000, -1000, 1266, 46, -1000, -1000, 2447,
	709, -70, -70, -98, -98, -98, 14, 2497, 1798, 2447,
	-1000, -1000, -1000, -1000, 166, 166, 2447, 1798, 166, 166,
	282, 62, 282, 282, -37, -1000, -65, -39, -1000, 6,
	2447, -1000, 222, 245, -1000, 2447, 1798, -1000, -3, -1000,
	-1000, 170, 415, 2447, 414, -1000, 2447, -1000, 1798, -1000,
	-1000, -133, -101, -104, -1000, 586, -1000, -20, 2447, 154,
	154, -1000, 556, -1000, 2257, 46, -1000, -1000, -1000, 2025,
	-1000, 1798, -1000, -1000, 166, 282, 166, 166, 15, 2447,
	15, -1000, -1000, 18, 1798, 314, -24, 1798, 2447, -1000,
	126, -1000, 1798, -1000, -12, 154, 32, 32, -1000, -1000,
	-1000, 2447, 1199, 104, 104, -1000, -1000, -1000, 1143, -1000,
	-40, 2447, -1000, 166, -1000, -1000, -1000, 1076, -1000, -26,
	-1000, 156, 74, 154, -69, 27, 392, -1000, 19, 210,
	-133, -133, 523, -1000, -1000, -1000, -1000, 1798, -1000, -1000,
	413, 18, 15, 15, 282, 344, 219, 186, -1000, -1000,
	-1000, 2447, -43, -1000, 180, 154, 154, -1000, -1000, -1000,
	-1000, -1000, -99, -1000, 166, 149, 273, 314, 62, 461,
	1798, 297, 210, 210, -1000, 220, 147, 74, 97, 2447,
	2447, -1000, -1000, 137, 62, 364, 282, -1000, 95, 1798,
	1798, 72, 81, 62, 64, -1000, 2447, 166, -1000, -1000,
	-1000, 260, -1000, 62, -1000, -1000, 252, -1000, 1018, -1000,
	146, 265, -1000, 263, -1000, 445, 134, 128, 62, 359,
	342, 64, 2447, 2447, -1000, -1000, -1000,
}
var yyPgo = []int{

	0, 629, 628, 627, 51, 626, 625, 0, 56, 220,
	36, 299, 37, 22, 19, 26, 15, 20, 624, 623,
	621, 620, 55, 310, 619, 618, 617, 44, 46, 296,
	29, 616, 613, 612, 611, 40, 610, 53, 608, 607,
	606, 438, 604, 43, 32, 603, 16, 25, 47, 144,
	600, 31, 13, 248, 597, 6, 595, 38, 594, 592,
	591, 590, 580, 42, 33, 579, 50, 578, 577, 30,
	576, 574, 9, 573, 571, 568, 564, 487, 558, 555,
	554, 553, 549, 548, 545, 544, 543, 539, 538, 534,
	528, 466, 39, 14, 527, 526, 525, 4, 21, 524,
	17, 7, 34, 518, 8, 28, 515, 507, 24, 12,
	506, 505, 3, 2, 5, 27, 41, 504, 503, 500,
	498, 35, 488, 18, 484,
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
	115, 116, 116, 117, 117, 118, 118, 118, 89, 90,
	119, 119, 46, 46, 46, 46, 46, 46, 7, 7,
	7, 7, 7, 7, 7, 7, 7, 7, 7, 7,
	7, 7, 7, 7, 7, 7, 7, 7, 7, 7,
	7, 7, 7, 7, 7, 7, 7, 7, 7, 7,
	7, 7, 7, 7, 7, 7, 7, 8, 8, 8,
	8, 8, 8, 8, 9, 9, 9, 9, 9, 9,
	9, 9, 9, 9, 9, 9, 9, 9, 1, 1,
	1, 1, 1, 1, 1, 1, 2, 6, 6, 5,
	5, 4, 3, 11, 11, 18, 19, 19, 20, 23,
	23, 21, 22, 22, 31, 31, 31, 32, 24, 24,
	25, 25, 25, 28, 28, 27, 27, 29, 26, 26,
	33, 34, 34,
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
	1, 1, 3, 0, 3, 0, 2, 2, 5, 6,
	0, 3, 1, 3, 3, 5, 5, 4, 1, 3,
	3, 5, 5, 4, 5, 6, 3, 3, 3, 3,
	3, 3, 3, 3, 2, 3, 3, 3, 3, 3,
	3, 3, 5, 6, 3, 4, 3, 4, 3, 4,
	3, 4, 3, 4, 3, 4, 2, 1, 1, 1,
	2, 1, 1, 1, 1, 3, 3, 5, 5, 4,
	5, 6, 3, 3, 3, 3, 3, 3, 1, 1,
	1, 1, 1, 1, 1, 1, 3, 0, 1, 1,
	3, 3, 3, 0, 1, 3, 1, 1, 3, 4,
	5, 2, 0, 2, 4, 5, 4, 1, 1, 1,
	4, 4, 4, 1, 3, 3, 3, 2, 6, 6,
	3, 1, 1,
}
var yyChk = []int{

	-1000, -120, -77, -7, -78, -79, -80, -81, -8, 86,
	46, 47, -35, -82, -83, -84, -85, -86, -87, -1,
	157, -31, 82, -18, -24, -33, -36, 63, 138, 31,
	137, 81, -88, -89, -90, 87, 48, 129, 155, 154,
	156, -2, -3, -32, 18, -25, -26, 159, -37, 26,
	37, 5, 161, 163, 8, 121, 42, 9, 49, -39,
	-38, -41, -66, 52, 118, 182, 163, 177, 82, 178,
	179, 180, 176, 7, 92, 169, 170, 171, 172, 173,
	174, 175, 13, 86, 75, 58, 151, 66, -7, -7,
	-77, -7, -68, 133, 64, 43, -67, 93, 65, 65,
	52, -91, -48, -49, 157, 65, 159, -19, -20, -21,
	-7, -23, 147, -34, -7, -35, 101, 60, 60, 60,
	-6, -5, -4, 156, -11, -10, -7, -28, -27, -17,
	157, -28, -28, -7, -7, -53, -54, 73, -42, -41,
	-40, -43, -49, -48, 124, -65, -64, 35, 4, -121,
	-63, 106, 39, 178, -7, 157, 158, 163, -7, -7,
	-7, -7, -7, -7, -7, -7, -7, -7, -7, -7,
	-7, -7, -7, -7, -9, -8, 13, 75, 58, 151,
	-7, -7, -7, 87, 86, 83, 143, -72, -71, 76,
	-37, 4, -37, 4, -37, 4, 16, -91, -91, -91,
	-51, -50, 139, 167, -16, -15, -14, 10, 157, -91,
	-11, 35, 178, 41, -23, 147, -22, 40, -7, 160,
	60, -115, 157, -116, -49, -48, -116, 162, 166, 167,
	164, 166, -29, 166, 116, 58, 151, -29, -29, 51,
	51, -55, -56, 148, -13, -12, -14, -53, -45, 62,
	72, -47, 182, 167, 167, 166, -64, -121, -64, -7,
	182, -16, -7, 164, 167, 7, 182, 163, 177, 82,
	178, 179, 180, 176, -9, -7, -7, -7, 87, 83,
	143, -74, -73, 89, -7, -37, -37, -37, -70, -69,
	-7, -94, -93, 68, -93, -51, -101, -104, 119, 136,
	-123, 101, -49, 157, -14, 141, 160, -7, 160, -22,
	-7, -7, 125, 90, 90, 182, 167, 182, -4, -7,
	-7, 41, -27, -7, -7, -7, 41, 41, -28, -28,
	-57, -58, 55, -7, 166, 169, -55, 67, 85, -122,
	135, 50, -124, 94, -16, -46, 157, -49, -49, -63,
	-7, 178, 164, 165, 164, -7, -9, 157, 158, 163,
	-7, -9, -9, -9, -9, -9, -9, 7, -7, 166,
	-76, -75, 11, 33, -92, -35, 144, -7, -92, -35,
	-55, -104, -55, -55, -103, -102, -46, -106, -105, -46,
	69, -16, -43, 159, 160, 125, -7, -116, -116, -115,
	-49, -115, -30, 147, -30, -66, 16, -12, -7, -57,
	-44, -49, -48, 124, -44, -7, -51, 182, 163, -47,
	-47, 164, -7, 164, 167, -9, -69, -97, -96, 111,
	-97, -7, -97, -97, -72, -55, -72, -72, 166, 169,
	166, -108, -107, 51, -7, 90, -35, -7, 159, -119,
	109, 41, -7, 41, -10, -47, 167, 167, -16, 157,
	158, 163, -7, -16, -16, 164, 165, 164, -7, -95,
	-64, -121, -97, -72, -97, -97, -102, -7, -105, -99,
	-98, -17, -93, 160, -10, 126, -59, -60, 74, -16,
	-49, -49, -7, 164, -51, -51, 164, -7, -97, -108,
	-30, 166, 58, 151, -109, 147, -15, 160, -115, -61,
	-62, 56, -13, -52, 90, -47, -47, 164, 165, 41,
	-98, -100, -46, -100, -72, 79, 86, 90, -117, 96,
	-7, -123, -16, -16, -97, 125, 79, -93, -55, 16,
	69, -52, -52, 137, 31, 125, -109, -118, 141, -7,
	-7, -111, -101, -104, -112, -55, 63, -72, 145, 146,
	-110, 147, -55, -104, -55, -114, 147, -113, -7, -97,
	79, 86, -55, 86, -55, 125, 79, 79, 31, 125,
	125, -112, 63, 63, -114, -113, -113,
}
var yyDef = []int{

	0, -2, 1, 2, 3, 4, 5, 6, 188, 0,
	0, 0, 8, 9, 10, 11, 12, 13, 14, 227,
	-2, 229, 0, 231, 232, 233, 98, 0, 0, 0,
	0, 0, 15, 16, 17, 248, 249, 250, 251, 252,
	253, 254, 255, 0, 0, 278, 279, 0, 19, 0,
	0, 0, 257, 263, 0, 0, 0, 0, 0, 26,
	27, 78, 48, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 204, 226,
	7, 230, 108, 0, 0, 0, 99, 0, 0, 0,
	0, 67, 0, 43, -2, 0, 263, 0, 266, 267,
	0, 272, 0, 0, 291, 292, 0, 0, 0, 0,
	0, 258, 259, 0, 0, 264, 90, 0, 283, 0,
	144, 0, 0, 0, 0, 84, 79, 0, 78, 49,
	-2, 51, 65, 0, 0, 30, 31, 0, 0, 0,
	38, 36, 37, 40, 43, 189, 190, 0, 0, 196,
	197, 198, 199, 200, 201, 202, 203, -2, -2, -2,
	-2, -2, -2, -2, 0, 234, 0, 0, 0, 0,
	-2, -2, -2, 220, 0, 222, 224, 111, 109, 0,
	20, 0, 22, 0, 24, 0, 0, 118, 0, 67,
	0, 68, 70, 0, 117, 44, 45, 0, 47, 0,
	0, 0, 0, 265, 272, 0, 271, 0, 0, 290,
	0, 0, 170, 0, 171, 0, 0, 256, 0, 0,
	262, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 87, 85, 0, 80, 81, 0, 84, 0, 73,
	75, 43, 0, 0, 0, 0, 32, 0, 33, 34,
	0, 42, 0, 193, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, -2, -2, -2, 221, 223,
	225, 18, 112, 0, 110, 21, 23, 25, 100, 101,
	104, 0, 119, 0, 0, 84, 84, 84, 0, 0,
	0, 71, 43, 64, 46, 0, 274, 0, 276, 268,
	0, 273, 0, 0, 0, 0, 0, 0, 260, 261,
	91, 280, 284, 287, 285, 286, 281, 282, 146, 146,
	0, 88, 0, 86, 0, 0, 87, 0, 0, 0,
	55, 56, 74, 76, 67, 66, 182, 65, 65, 39,
	35, 41, 191, 192, 194, 0, 212, 235, 236, 0,
	0, 242, 243, 244, 245, 246, 247, 0, 113, 0,
	103, 105, 106, 107, 122, 122, 0, 120, 122, 122,
	108, 84, 108, 108, 133, 134, 0, 148, 149, 137,
	0, 116, 0, 0, 275, 0, 269, 168, 0, 178,
	172, 180, 0, 0, 0, 28, 0, 82, 83, 29,
	52, 65, 0, 0, 53, 43, 57, 0, 0, 43,
	43, 195, 0, 239, 0, 213, 102, 114, 123, 0,
	115, 121, 127, 128, 122, 108, 122, 122, 0, 0,
	0, 151, 138, 0, 69, 0, 0, 270, 0, 179,
	0, 288, 147, 289, 92, 43, 0, 0, 54, 183,
	184, 0, 0, 67, 67, 237, 238, 240, 0, 124,
	125, 0, 129, 122, 131, 132, 135, 137, 150, 146,
	140, 0, 154, 0, 0, 0, 95, 93, 0, 0,
	65, 65, 0, 187, 58, 59, 241, 126, 130, 136,
	0, 0, 0, 0, 108, 0, 0, 173, 181, 89,
	96, 0, 94, 60, 70, 43, 43, 185, 186, 139,
	141, 142, 145, 143, 122, 0, 0, 0, 84, 0,
	97, 0, 0, 0, 152, 0, 0, 154, 175, 0,
	0, 61, 62, 0, 84, 0, 108, 169, 0, 174,
	77, 158, 84, 84, 161, 166, 0, 122, 176, 177,
	155, 0, 163, 84, 165, 156, 0, 157, 84, 153,
	0, 0, 164, 0, 167, 0, 0, 0, 84, 0,
	0, 161, 0, 0, 159, 160, 162,
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
	182,
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
		//line n1ql.y:340
		{
			yylex.(*lexer).setStatement(yyS[yypt-0].statement)
		}
	case 2:
		//line n1ql.y:345
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
		//line n1ql.y:362
		{
			yyVAL.statement = algebra.NewExplain(yyS[yypt-0].statement)
		}
	case 8:
		//line n1ql.y:369
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
		//line n1ql.y:400
		{
			yyVAL.fullselect = algebra.NewSelect(yyS[yypt-3].subresult, yyS[yypt-2].order, yyS[yypt-0].expr, yyS[yypt-1].expr) /* OFFSET precedes LIMIT */
		}
	case 19:
		//line n1ql.y:406
		{
			yyVAL.subresult = yyS[yypt-0].subselect
		}
	case 20:
		//line n1ql.y:411
		{
			yyVAL.subresult = algebra.NewUnion(yyS[yypt-2].subresult, yyS[yypt-0].subselect)
		}
	case 21:
		//line n1ql.y:416
		{
			yyVAL.subresult = algebra.NewUnionAll(yyS[yypt-3].subresult, yyS[yypt-0].subselect)
		}
	case 22:
		//line n1ql.y:421
		{
			yyVAL.subresult = algebra.NewIntersect(yyS[yypt-2].subresult, yyS[yypt-0].subselect)
		}
	case 23:
		//line n1ql.y:426
		{
			yyVAL.subresult = algebra.NewIntersectAll(yyS[yypt-3].subresult, yyS[yypt-0].subselect)
		}
	case 24:
		//line n1ql.y:431
		{
			yyVAL.subresult = algebra.NewExcept(yyS[yypt-2].subresult, yyS[yypt-0].subselect)
		}
	case 25:
		//line n1ql.y:436
		{
			yyVAL.subresult = algebra.NewExceptAll(yyS[yypt-3].subresult, yyS[yypt-0].subselect)
		}
	case 26:
		yyVAL.subselect = yyS[yypt-0].subselect
	case 27:
		yyVAL.subselect = yyS[yypt-0].subselect
	case 28:
		//line n1ql.y:449
		{
			yyVAL.subselect = algebra.NewSubselect(yyS[yypt-4].fromTerm, yyS[yypt-3].bindings, yyS[yypt-2].expr, yyS[yypt-1].group, yyS[yypt-0].projection)
		}
	case 29:
		//line n1ql.y:456
		{
			yyVAL.subselect = algebra.NewSubselect(yyS[yypt-3].fromTerm, yyS[yypt-2].bindings, yyS[yypt-1].expr, yyS[yypt-0].group, yyS[yypt-4].projection)
		}
	case 30:
		//line n1ql.y:471
		{
			yyVAL.projection = yyS[yypt-0].projection
		}
	case 31:
		//line n1ql.y:478
		{
			yyVAL.projection = algebra.NewProjection(false, yyS[yypt-0].resultTerms)
		}
	case 32:
		//line n1ql.y:483
		{
			yyVAL.projection = algebra.NewProjection(true, yyS[yypt-0].resultTerms)
		}
	case 33:
		//line n1ql.y:488
		{
			yyVAL.projection = algebra.NewProjection(false, yyS[yypt-0].resultTerms)
		}
	case 34:
		//line n1ql.y:493
		{
			yyVAL.projection = algebra.NewRawProjection(false, yyS[yypt-0].expr)
		}
	case 35:
		//line n1ql.y:498
		{
			yyVAL.projection = algebra.NewRawProjection(true, yyS[yypt-0].expr)
		}
	case 38:
		//line n1ql.y:511
		{
			yyVAL.resultTerms = algebra.ResultTerms{yyS[yypt-0].resultTerm}
		}
	case 39:
		//line n1ql.y:516
		{
			yyVAL.resultTerms = append(yyS[yypt-2].resultTerms, yyS[yypt-0].resultTerm)
		}
	case 40:
		//line n1ql.y:523
		{
			yyVAL.resultTerm = algebra.NewResultTerm(nil, true, "")
		}
	case 41:
		//line n1ql.y:528
		{
			yyVAL.resultTerm = algebra.NewResultTerm(yyS[yypt-2].expr, true, "")
		}
	case 42:
		//line n1ql.y:533
		{
			yyVAL.resultTerm = algebra.NewResultTerm(yyS[yypt-1].expr, false, yyS[yypt-0].s)
		}
	case 43:
		//line n1ql.y:540
		{
			yyVAL.s = ""
		}
	case 44:
		yyVAL.s = yyS[yypt-0].s
	case 45:
		yyVAL.s = yyS[yypt-0].s
	case 46:
		//line n1ql.y:551
		{
			yyVAL.s = yyS[yypt-0].s
		}
	case 47:
		yyVAL.s = yyS[yypt-0].s
	case 48:
		//line n1ql.y:569
		{
			yyVAL.fromTerm = nil
		}
	case 49:
		yyVAL.fromTerm = yyS[yypt-0].fromTerm
	case 50:
		//line n1ql.y:578
		{
			yyVAL.fromTerm = yyS[yypt-0].fromTerm
		}
	case 51:
		//line n1ql.y:585
		{
			yyVAL.fromTerm = yyS[yypt-0].keyspaceTerm
		}
	case 52:
		//line n1ql.y:590
		{
			yyVAL.fromTerm = algebra.NewJoin(yyS[yypt-3].fromTerm, yyS[yypt-2].b, yyS[yypt-0].keyspaceTerm)
		}
	case 53:
		//line n1ql.y:595
		{
			yyVAL.fromTerm = algebra.NewNest(yyS[yypt-3].fromTerm, yyS[yypt-2].b, yyS[yypt-0].keyspaceTerm)
		}
	case 54:
		//line n1ql.y:600
		{
			yyVAL.fromTerm = algebra.NewUnnest(yyS[yypt-4].fromTerm, yyS[yypt-3].b, yyS[yypt-1].expr, yyS[yypt-0].s)
		}
	case 57:
		//line n1ql.y:613
		{
			yyVAL.keyspaceTerm = algebra.NewKeyspaceTerm("", yyS[yypt-3].s, yyS[yypt-2].path, yyS[yypt-1].s, yyS[yypt-0].expr)
		}
	case 58:
		//line n1ql.y:618
		{
			yyVAL.keyspaceTerm = algebra.NewKeyspaceTerm(yyS[yypt-5].s, yyS[yypt-3].s, yyS[yypt-2].path, yyS[yypt-1].s, yyS[yypt-0].expr)
		}
	case 59:
		//line n1ql.y:623
		{
			yyVAL.keyspaceTerm = algebra.NewKeyspaceTerm("#system", yyS[yypt-3].s, yyS[yypt-2].path, yyS[yypt-1].s, yyS[yypt-0].expr)
		}
	case 60:
		//line n1ql.y:630
		{
			yyVAL.keyspaceTerm = algebra.NewKeyspaceTerm("", yyS[yypt-3].s, yyS[yypt-2].path, yyS[yypt-1].s, yyS[yypt-0].expr)
		}
	case 61:
		//line n1ql.y:635
		{
			yyVAL.keyspaceTerm = algebra.NewKeyspaceTerm(yyS[yypt-5].s, yyS[yypt-3].s, yyS[yypt-2].path, yyS[yypt-1].s, yyS[yypt-0].expr)
		}
	case 62:
		//line n1ql.y:640
		{
			yyVAL.keyspaceTerm = algebra.NewKeyspaceTerm("#system", yyS[yypt-3].s, yyS[yypt-2].path, yyS[yypt-1].s, yyS[yypt-0].expr)
		}
	case 63:
		yyVAL.s = yyS[yypt-0].s
	case 64:
		yyVAL.s = yyS[yypt-0].s
	case 65:
		//line n1ql.y:655
		{
			yyVAL.path = nil
		}
	case 66:
		//line n1ql.y:660
		{
			yyVAL.path = yyS[yypt-0].path
		}
	case 67:
		//line n1ql.y:667
		{
			yyVAL.expr = nil
		}
	case 68:
		yyVAL.expr = yyS[yypt-0].expr
	case 69:
		//line n1ql.y:676
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 70:
		//line n1ql.y:683
		{
		}
	case 72:
		//line n1ql.y:691
		{
			yyVAL.b = false
		}
	case 73:
		//line n1ql.y:696
		{
			yyVAL.b = false
		}
	case 74:
		//line n1ql.y:701
		{
			yyVAL.b = true
		}
	case 77:
		//line n1ql.y:714
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 78:
		//line n1ql.y:728
		{
			yyVAL.bindings = nil
		}
	case 79:
		yyVAL.bindings = yyS[yypt-0].bindings
	case 80:
		//line n1ql.y:737
		{
			yyVAL.bindings = yyS[yypt-0].bindings
		}
	case 81:
		//line n1ql.y:744
		{
			yyVAL.bindings = expression.Bindings{yyS[yypt-0].binding}
		}
	case 82:
		//line n1ql.y:749
		{
			yyVAL.bindings = append(yyS[yypt-2].bindings, yyS[yypt-0].binding)
		}
	case 83:
		//line n1ql.y:756
		{
			yyVAL.binding = expression.NewBinding(yyS[yypt-2].s, yyS[yypt-0].expr)
		}
	case 84:
		//line n1ql.y:770
		{
			yyVAL.expr = nil
		}
	case 85:
		yyVAL.expr = yyS[yypt-0].expr
	case 86:
		//line n1ql.y:779
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 87:
		//line n1ql.y:793
		{
			yyVAL.group = nil
		}
	case 88:
		yyVAL.group = yyS[yypt-0].group
	case 89:
		//line n1ql.y:802
		{
			yyVAL.group = algebra.NewGroup(yyS[yypt-2].exprs, yyS[yypt-1].bindings, yyS[yypt-0].expr)
		}
	case 90:
		//line n1ql.y:809
		{
			yyVAL.exprs = expression.Expressions{yyS[yypt-0].expr}
		}
	case 91:
		//line n1ql.y:814
		{
			yyVAL.exprs = append(yyS[yypt-2].exprs, yyS[yypt-0].expr)
		}
	case 92:
		//line n1ql.y:821
		{
			yyVAL.bindings = nil
		}
	case 93:
		yyVAL.bindings = yyS[yypt-0].bindings
	case 94:
		//line n1ql.y:830
		{
			yyVAL.bindings = yyS[yypt-0].bindings
		}
	case 95:
		//line n1ql.y:837
		{
			yyVAL.expr = nil
		}
	case 96:
		yyVAL.expr = yyS[yypt-0].expr
	case 97:
		//line n1ql.y:846
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 98:
		//line n1ql.y:860
		{
			yyVAL.order = nil
		}
	case 99:
		yyVAL.order = yyS[yypt-0].order
	case 100:
		//line n1ql.y:869
		{
			yyVAL.order = algebra.NewOrder(yyS[yypt-0].sortTerms)
		}
	case 101:
		//line n1ql.y:876
		{
			yyVAL.sortTerms = algebra.SortTerms{yyS[yypt-0].sortTerm}
		}
	case 102:
		//line n1ql.y:881
		{
			yyVAL.sortTerms = append(yyS[yypt-2].sortTerms, yyS[yypt-0].sortTerm)
		}
	case 103:
		//line n1ql.y:888
		{
			yyVAL.sortTerm = algebra.NewSortTerm(yyS[yypt-1].expr, yyS[yypt-0].b)
		}
	case 104:
		//line n1ql.y:895
		{
			yyVAL.b = false
		}
	case 105:
		yyVAL.b = yyS[yypt-0].b
	case 106:
		//line n1ql.y:904
		{
			yyVAL.b = false
		}
	case 107:
		//line n1ql.y:909
		{
			yyVAL.b = true
		}
	case 108:
		//line n1ql.y:923
		{
			yyVAL.expr = nil
		}
	case 109:
		yyVAL.expr = yyS[yypt-0].expr
	case 110:
		//line n1ql.y:932
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 111:
		//line n1ql.y:946
		{
			yyVAL.expr = nil
		}
	case 112:
		yyVAL.expr = yyS[yypt-0].expr
	case 113:
		//line n1ql.y:955
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 114:
		//line n1ql.y:969
		{
			yyVAL.statement = algebra.NewInsertValues(yyS[yypt-3].keyspaceRef, yyS[yypt-2].expr, yyS[yypt-1].expr, yyS[yypt-0].projection)
		}
	case 115:
		//line n1ql.y:974
		{
			yyVAL.statement = algebra.NewInsertSelect(yyS[yypt-3].keyspaceRef, yyS[yypt-2].expr, yyS[yypt-1].fullselect, yyS[yypt-0].projection)
		}
	case 116:
		//line n1ql.y:981
		{
			yyVAL.keyspaceRef = algebra.NewKeyspaceRef(yyS[yypt-3].s, yyS[yypt-1].s, yyS[yypt-0].s)
		}
	case 117:
		//line n1ql.y:986
		{
			yyVAL.keyspaceRef = algebra.NewKeyspaceRef("", yyS[yypt-1].s, yyS[yypt-0].s)
		}
	case 118:
		//line n1ql.y:993
		{
			yyVAL.expr = nil
		}
	case 119:
		yyVAL.expr = yyS[yypt-0].expr
	case 120:
		//line n1ql.y:1002
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 121:
		//line n1ql.y:1009
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 122:
		//line n1ql.y:1016
		{
			yyVAL.projection = nil
		}
	case 123:
		yyVAL.projection = yyS[yypt-0].projection
	case 124:
		//line n1ql.y:1025
		{
			yyVAL.projection = yyS[yypt-0].projection
		}
	case 125:
		//line n1ql.y:1032
		{
			yyVAL.projection = algebra.NewProjection(false, yyS[yypt-0].resultTerms)
		}
	case 126:
		//line n1ql.y:1037
		{
			yyVAL.projection = algebra.NewRawProjection(false, yyS[yypt-0].expr)
		}
	case 127:
		//line n1ql.y:1051
		{
			yyVAL.statement = algebra.NewUpsertValues(yyS[yypt-3].keyspaceRef, yyS[yypt-2].expr, yyS[yypt-1].expr, yyS[yypt-0].projection)
		}
	case 128:
		//line n1ql.y:1056
		{
			yyVAL.statement = algebra.NewUpsertSelect(yyS[yypt-3].keyspaceRef, yyS[yypt-2].expr, yyS[yypt-1].fullselect, yyS[yypt-0].projection)
		}
	case 129:
		//line n1ql.y:1070
		{
			yyVAL.statement = algebra.NewDelete(yyS[yypt-4].keyspaceRef, yyS[yypt-3].expr, yyS[yypt-2].expr, yyS[yypt-1].expr, yyS[yypt-0].projection)
		}
	case 130:
		//line n1ql.y:1084
		{
			yyVAL.statement = algebra.NewUpdate(yyS[yypt-6].keyspaceRef, yyS[yypt-5].expr, yyS[yypt-4].set, yyS[yypt-3].unset, yyS[yypt-2].expr, yyS[yypt-1].expr, yyS[yypt-0].projection)
		}
	case 131:
		//line n1ql.y:1089
		{
			yyVAL.statement = algebra.NewUpdate(yyS[yypt-5].keyspaceRef, yyS[yypt-4].expr, yyS[yypt-3].set, nil, yyS[yypt-2].expr, yyS[yypt-1].expr, yyS[yypt-0].projection)
		}
	case 132:
		//line n1ql.y:1094
		{
			yyVAL.statement = algebra.NewUpdate(yyS[yypt-5].keyspaceRef, yyS[yypt-4].expr, nil, yyS[yypt-3].unset, yyS[yypt-2].expr, yyS[yypt-1].expr, yyS[yypt-0].projection)
		}
	case 133:
		//line n1ql.y:1101
		{
			yyVAL.set = algebra.NewSet(yyS[yypt-0].setTerms)
		}
	case 134:
		//line n1ql.y:1108
		{
			yyVAL.setTerms = algebra.SetTerms{yyS[yypt-0].setTerm}
		}
	case 135:
		//line n1ql.y:1113
		{
			yyVAL.setTerms = append(yyS[yypt-2].setTerms, yyS[yypt-0].setTerm)
		}
	case 136:
		//line n1ql.y:1120
		{
			yyVAL.setTerm = algebra.NewSetTerm(yyS[yypt-3].path, yyS[yypt-1].expr, yyS[yypt-0].updateFor)
		}
	case 137:
		//line n1ql.y:1127
		{
			yyVAL.updateFor = nil
		}
	case 138:
		yyVAL.updateFor = yyS[yypt-0].updateFor
	case 139:
		//line n1ql.y:1136
		{
			yyVAL.updateFor = algebra.NewUpdateFor(yyS[yypt-2].bindings, yyS[yypt-1].expr)
		}
	case 140:
		//line n1ql.y:1143
		{
			yyVAL.bindings = expression.Bindings{yyS[yypt-0].binding}
		}
	case 141:
		//line n1ql.y:1148
		{
			yyVAL.bindings = append(yyS[yypt-2].bindings, yyS[yypt-0].binding)
		}
	case 142:
		//line n1ql.y:1155
		{
			yyVAL.binding = expression.NewBinding(yyS[yypt-2].s, yyS[yypt-0].expr)
		}
	case 143:
		//line n1ql.y:1160
		{
			yyVAL.binding = expression.NewDescendantBinding(yyS[yypt-2].s, yyS[yypt-0].expr)
		}
	case 144:
		yyVAL.s = yyS[yypt-0].s
	case 145:
		//line n1ql.y:1171
		{
			yyVAL.expr = yyS[yypt-0].path
		}
	case 146:
		//line n1ql.y:1178
		{
			yyVAL.expr = nil
		}
	case 147:
		//line n1ql.y:1183
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 148:
		//line n1ql.y:1190
		{
			yyVAL.unset = algebra.NewUnset(yyS[yypt-0].unsetTerms)
		}
	case 149:
		//line n1ql.y:1197
		{
			yyVAL.unsetTerms = algebra.UnsetTerms{yyS[yypt-0].unsetTerm}
		}
	case 150:
		//line n1ql.y:1202
		{
			yyVAL.unsetTerms = append(yyS[yypt-2].unsetTerms, yyS[yypt-0].unsetTerm)
		}
	case 151:
		//line n1ql.y:1209
		{
			yyVAL.unsetTerm = algebra.NewUnsetTerm(yyS[yypt-1].path, yyS[yypt-0].updateFor)
		}
	case 152:
		//line n1ql.y:1223
		{
			source := algebra.NewMergeSourceFrom(yyS[yypt-5].keyspaceTerm, "")
			yyVAL.statement = algebra.NewMerge(yyS[yypt-7].keyspaceRef, source, yyS[yypt-3].expr, yyS[yypt-2].mergeActions, yyS[yypt-1].expr, yyS[yypt-0].projection)
		}
	case 153:
		//line n1ql.y:1229
		{
			source := algebra.NewMergeSourceSelect(yyS[yypt-7].fullselect, yyS[yypt-5].s)
			yyVAL.statement = algebra.NewMerge(yyS[yypt-10].keyspaceRef, source, yyS[yypt-3].expr, yyS[yypt-2].mergeActions, yyS[yypt-1].expr, yyS[yypt-0].projection)
		}
	case 154:
		//line n1ql.y:1237
		{
			yyVAL.mergeActions = algebra.NewMergeActions(nil, nil, nil)
		}
	case 155:
		//line n1ql.y:1242
		{
			yyVAL.mergeActions = algebra.NewMergeActions(yyS[yypt-1].mergeUpdate, yyS[yypt-0].mergeActions.Delete(), yyS[yypt-0].mergeActions.Insert())
		}
	case 156:
		//line n1ql.y:1247
		{
			yyVAL.mergeActions = algebra.NewMergeActions(nil, yyS[yypt-1].mergeDelete, yyS[yypt-0].mergeInsert)
		}
	case 157:
		//line n1ql.y:1252
		{
			yyVAL.mergeActions = algebra.NewMergeActions(nil, nil, yyS[yypt-0].mergeInsert)
		}
	case 158:
		//line n1ql.y:1259
		{
			yyVAL.mergeActions = algebra.NewMergeActions(nil, nil, nil)
		}
	case 159:
		//line n1ql.y:1264
		{
			yyVAL.mergeActions = algebra.NewMergeActions(nil, yyS[yypt-1].mergeDelete, yyS[yypt-0].mergeInsert)
		}
	case 160:
		//line n1ql.y:1269
		{
			yyVAL.mergeActions = algebra.NewMergeActions(nil, nil, yyS[yypt-0].mergeInsert)
		}
	case 161:
		//line n1ql.y:1276
		{
			yyVAL.mergeInsert = nil
		}
	case 162:
		//line n1ql.y:1281
		{
			yyVAL.mergeInsert = yyS[yypt-0].mergeInsert
		}
	case 163:
		//line n1ql.y:1288
		{
			yyVAL.mergeUpdate = algebra.NewMergeUpdate(yyS[yypt-1].set, nil, yyS[yypt-0].expr)
		}
	case 164:
		//line n1ql.y:1293
		{
			yyVAL.mergeUpdate = algebra.NewMergeUpdate(yyS[yypt-2].set, yyS[yypt-1].unset, yyS[yypt-0].expr)
		}
	case 165:
		//line n1ql.y:1298
		{
			yyVAL.mergeUpdate = algebra.NewMergeUpdate(nil, yyS[yypt-1].unset, yyS[yypt-0].expr)
		}
	case 166:
		//line n1ql.y:1305
		{
			yyVAL.mergeDelete = algebra.NewMergeDelete(yyS[yypt-0].expr)
		}
	case 167:
		//line n1ql.y:1312
		{
			yyVAL.mergeInsert = algebra.NewMergeInsert(yyS[yypt-1].expr, yyS[yypt-0].expr)
		}
	case 168:
		//line n1ql.y:1326
		{
			yyVAL.statement = algebra.NewCreatePrimaryIndex(yyS[yypt-0].keyspaceRef)
		}
	case 169:
		//line n1ql.y:1331
		{
			yyVAL.statement = algebra.NewCreateIndex(yyS[yypt-8].s, yyS[yypt-6].keyspaceRef, yyS[yypt-4].exprs, yyS[yypt-2].expr, yyS[yypt-1].expr, yyS[yypt-0].indexType)
		}
	case 170:
		yyVAL.s = yyS[yypt-0].s
	case 171:
		//line n1ql.y:1342
		{
			yyVAL.keyspaceRef = algebra.NewKeyspaceRef("", yyS[yypt-0].s, "")
		}
	case 172:
		//line n1ql.y:1347
		{
			yyVAL.keyspaceRef = algebra.NewKeyspaceRef(yyS[yypt-2].s, yyS[yypt-0].s, "")
		}
	case 173:
		//line n1ql.y:1354
		{
			yyVAL.expr = nil
		}
	case 174:
		//line n1ql.y:1359
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 175:
		//line n1ql.y:1366
		{
			yyVAL.indexType = datastore.VIEW
		}
	case 176:
		//line n1ql.y:1371
		{
			yyVAL.indexType = datastore.VIEW
		}
	case 177:
		//line n1ql.y:1376
		{
			yyVAL.indexType = datastore.LSM
		}
	case 178:
		//line n1ql.y:1390
		{
			yyVAL.statement = algebra.NewDropIndex(yyS[yypt-2].keyspaceRef, yyS[yypt-0].s)
		}
	case 179:
		//line n1ql.y:1403
		{
			yyVAL.statement = algebra.NewAlterIndex(yyS[yypt-3].keyspaceRef, yyS[yypt-1].s, yyS[yypt-0].s)
		}
	case 180:
		//line n1ql.y:1409
		{
			yyVAL.s = ""
		}
	case 181:
		//line n1ql.y:1414
		{
			yyVAL.s = yyS[yypt-0].s
		}
	case 182:
		//line n1ql.y:1428
		{
			yyVAL.path = expression.NewIdentifier(yyS[yypt-0].s)
		}
	case 183:
		//line n1ql.y:1433
		{
			yyVAL.path = expression.NewField(yyS[yypt-2].path, expression.NewFieldName(yyS[yypt-0].s))
		}
	case 184:
		//line n1ql.y:1438
		{
			field := expression.NewField(yyS[yypt-2].path, expression.NewFieldName(yyS[yypt-0].s))
			field.SetCaseInsensitive(true)
			yyVAL.path = field
		}
	case 185:
		//line n1ql.y:1445
		{
			yyVAL.path = expression.NewField(yyS[yypt-4].path, yyS[yypt-1].expr)
		}
	case 186:
		//line n1ql.y:1450
		{
			field := expression.NewField(yyS[yypt-4].path, yyS[yypt-1].expr)
			field.SetCaseInsensitive(true)
			yyVAL.path = field
		}
	case 187:
		//line n1ql.y:1457
		{
			yyVAL.path = expression.NewElement(yyS[yypt-3].path, yyS[yypt-1].expr)
		}
	case 188:
		yyVAL.expr = yyS[yypt-0].expr
	case 189:
		//line n1ql.y:1474
		{
			yyVAL.expr = expression.NewField(yyS[yypt-2].expr, expression.NewFieldName(yyS[yypt-0].s))
		}
	case 190:
		//line n1ql.y:1479
		{
			field := expression.NewField(yyS[yypt-2].expr, expression.NewFieldName(yyS[yypt-0].s))
			field.SetCaseInsensitive(true)
			yyVAL.expr = field
		}
	case 191:
		//line n1ql.y:1486
		{
			yyVAL.expr = expression.NewField(yyS[yypt-4].expr, yyS[yypt-1].expr)
		}
	case 192:
		//line n1ql.y:1491
		{
			field := expression.NewField(yyS[yypt-4].expr, yyS[yypt-1].expr)
			field.SetCaseInsensitive(true)
			yyVAL.expr = field
		}
	case 193:
		//line n1ql.y:1498
		{
			yyVAL.expr = expression.NewElement(yyS[yypt-3].expr, yyS[yypt-1].expr)
		}
	case 194:
		//line n1ql.y:1503
		{
			yyVAL.expr = expression.NewSlice(yyS[yypt-4].expr, yyS[yypt-2].expr)
		}
	case 195:
		//line n1ql.y:1508
		{
			yyVAL.expr = expression.NewSlice(yyS[yypt-5].expr, yyS[yypt-3].expr, yyS[yypt-1].expr)
		}
	case 196:
		//line n1ql.y:1514
		{
			yyVAL.expr = expression.NewAdd(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 197:
		//line n1ql.y:1519
		{
			yyVAL.expr = expression.NewSub(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 198:
		//line n1ql.y:1524
		{
			yyVAL.expr = expression.NewMult(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 199:
		//line n1ql.y:1529
		{
			yyVAL.expr = expression.NewDiv(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 200:
		//line n1ql.y:1534
		{
			yyVAL.expr = expression.NewMod(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 201:
		//line n1ql.y:1540
		{
			yyVAL.expr = expression.NewConcat(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 202:
		//line n1ql.y:1546
		{
			yyVAL.expr = expression.NewAnd(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 203:
		//line n1ql.y:1551
		{
			yyVAL.expr = expression.NewOr(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 204:
		//line n1ql.y:1556
		{
			yyVAL.expr = expression.NewNot(yyS[yypt-0].expr)
		}
	case 205:
		//line n1ql.y:1562
		{
			yyVAL.expr = expression.NewEq(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 206:
		//line n1ql.y:1567
		{
			yyVAL.expr = expression.NewEq(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 207:
		//line n1ql.y:1572
		{
			yyVAL.expr = expression.NewNE(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 208:
		//line n1ql.y:1577
		{
			yyVAL.expr = expression.NewLT(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 209:
		//line n1ql.y:1582
		{
			yyVAL.expr = expression.NewGT(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 210:
		//line n1ql.y:1587
		{
			yyVAL.expr = expression.NewLE(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 211:
		//line n1ql.y:1592
		{
			yyVAL.expr = expression.NewGE(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 212:
		//line n1ql.y:1597
		{
			yyVAL.expr = expression.NewBetween(yyS[yypt-4].expr, yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 213:
		//line n1ql.y:1602
		{
			yyVAL.expr = expression.NewNotBetween(yyS[yypt-5].expr, yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 214:
		//line n1ql.y:1607
		{
			yyVAL.expr = expression.NewLike(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 215:
		//line n1ql.y:1612
		{
			yyVAL.expr = expression.NewNotLike(yyS[yypt-3].expr, yyS[yypt-0].expr)
		}
	case 216:
		//line n1ql.y:1617
		{
			yyVAL.expr = expression.NewIn(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 217:
		//line n1ql.y:1622
		{
			yyVAL.expr = expression.NewNotIn(yyS[yypt-3].expr, yyS[yypt-0].expr)
		}
	case 218:
		//line n1ql.y:1627
		{
			yyVAL.expr = expression.NewWithin(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 219:
		//line n1ql.y:1632
		{
			yyVAL.expr = expression.NewNotWithin(yyS[yypt-3].expr, yyS[yypt-0].expr)
		}
	case 220:
		//line n1ql.y:1637
		{
			yyVAL.expr = expression.NewIsNull(yyS[yypt-2].expr)
		}
	case 221:
		//line n1ql.y:1642
		{
			yyVAL.expr = expression.NewIsNotNull(yyS[yypt-3].expr)
		}
	case 222:
		//line n1ql.y:1647
		{
			yyVAL.expr = expression.NewIsMissing(yyS[yypt-2].expr)
		}
	case 223:
		//line n1ql.y:1652
		{
			yyVAL.expr = expression.NewIsNotMissing(yyS[yypt-3].expr)
		}
	case 224:
		//line n1ql.y:1657
		{
			yyVAL.expr = expression.NewIsValued(yyS[yypt-2].expr)
		}
	case 225:
		//line n1ql.y:1662
		{
			yyVAL.expr = expression.NewIsNotValued(yyS[yypt-3].expr)
		}
	case 226:
		//line n1ql.y:1667
		{
			yyVAL.expr = expression.NewExists(yyS[yypt-0].expr)
		}
	case 227:
		yyVAL.expr = yyS[yypt-0].expr
	case 228:
		//line n1ql.y:1678
		{
			yyVAL.expr = expression.NewIdentifier(yyS[yypt-0].s)
		}
	case 229:
		yyVAL.expr = yyS[yypt-0].expr
	case 230:
		//line n1ql.y:1687
		{
			yyVAL.expr = expression.NewNeg(yyS[yypt-0].expr)
		}
	case 231:
		yyVAL.expr = yyS[yypt-0].expr
	case 232:
		yyVAL.expr = yyS[yypt-0].expr
	case 233:
		yyVAL.expr = yyS[yypt-0].expr
	case 234:
		yyVAL.expr = yyS[yypt-0].expr
	case 235:
		//line n1ql.y:1706
		{
			yyVAL.expr = expression.NewField(yyS[yypt-2].expr, expression.NewFieldName(yyS[yypt-0].s))
		}
	case 236:
		//line n1ql.y:1711
		{
			field := expression.NewField(yyS[yypt-2].expr, expression.NewFieldName(yyS[yypt-0].s))
			field.SetCaseInsensitive(true)
			yyVAL.expr = field
		}
	case 237:
		//line n1ql.y:1718
		{
			yyVAL.expr = expression.NewField(yyS[yypt-4].expr, yyS[yypt-1].expr)
		}
	case 238:
		//line n1ql.y:1723
		{
			field := expression.NewField(yyS[yypt-4].expr, yyS[yypt-1].expr)
			field.SetCaseInsensitive(true)
			yyVAL.expr = field
		}
	case 239:
		//line n1ql.y:1730
		{
			yyVAL.expr = expression.NewElement(yyS[yypt-3].expr, yyS[yypt-1].expr)
		}
	case 240:
		//line n1ql.y:1735
		{
			yyVAL.expr = expression.NewSlice(yyS[yypt-4].expr, yyS[yypt-2].expr)
		}
	case 241:
		//line n1ql.y:1740
		{
			yyVAL.expr = expression.NewSlice(yyS[yypt-5].expr, yyS[yypt-3].expr, yyS[yypt-1].expr)
		}
	case 242:
		//line n1ql.y:1746
		{
			yyVAL.expr = expression.NewAdd(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 243:
		//line n1ql.y:1751
		{
			yyVAL.expr = expression.NewSub(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 244:
		//line n1ql.y:1756
		{
			yyVAL.expr = expression.NewMult(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 245:
		//line n1ql.y:1761
		{
			yyVAL.expr = expression.NewDiv(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 246:
		//line n1ql.y:1766
		{
			yyVAL.expr = expression.NewMod(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 247:
		//line n1ql.y:1772
		{
			yyVAL.expr = expression.NewConcat(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 248:
		//line n1ql.y:1786
		{
			yyVAL.expr = expression.NULL_EXPR
		}
	case 249:
		//line n1ql.y:1791
		{
			yyVAL.expr = expression.FALSE_EXPR
		}
	case 250:
		//line n1ql.y:1796
		{
			yyVAL.expr = expression.TRUE_EXPR
		}
	case 251:
		//line n1ql.y:1801
		{
			yyVAL.expr = expression.NewConstant(value.NewValue(yyS[yypt-0].f))
		}
	case 252:
		//line n1ql.y:1806
		{
			yyVAL.expr = expression.NewConstant(value.NewValue(yyS[yypt-0].n))
		}
	case 253:
		//line n1ql.y:1811
		{
			yyVAL.expr = expression.NewConstant(value.NewValue(yyS[yypt-0].s))
		}
	case 254:
		yyVAL.expr = yyS[yypt-0].expr
	case 255:
		yyVAL.expr = yyS[yypt-0].expr
	case 256:
		//line n1ql.y:1822
		{
			yyVAL.expr = expression.NewObjectConstruct(yyS[yypt-1].bindings)
		}
	case 257:
		//line n1ql.y:1829
		{
			yyVAL.bindings = nil
		}
	case 258:
		yyVAL.bindings = yyS[yypt-0].bindings
	case 259:
		//line n1ql.y:1838
		{
			yyVAL.bindings = expression.Bindings{yyS[yypt-0].binding}
		}
	case 260:
		//line n1ql.y:1843
		{
			yyVAL.bindings = append(yyS[yypt-2].bindings, yyS[yypt-0].binding)
		}
	case 261:
		//line n1ql.y:1850
		{
			yyVAL.binding = expression.NewBinding(yyS[yypt-2].s, yyS[yypt-0].expr)
		}
	case 262:
		//line n1ql.y:1857
		{
			yyVAL.expr = expression.NewArrayConstruct(yyS[yypt-1].exprs...)
		}
	case 263:
		//line n1ql.y:1864
		{
			yyVAL.exprs = nil
		}
	case 264:
		yyVAL.exprs = yyS[yypt-0].exprs
	case 265:
		//line n1ql.y:1880
		{
			yyVAL.expr = yyS[yypt-1].expr
		}
	case 266:
		yyVAL.expr = yyS[yypt-0].expr
	case 267:
		yyVAL.expr = yyS[yypt-0].expr
	case 268:
		//line n1ql.y:1893
		{
			yyVAL.expr = expression.NewSimpleCase(yyS[yypt-2].expr, yyS[yypt-1].whenTerms, yyS[yypt-0].expr)
		}
	case 269:
		//line n1ql.y:1900
		{
			yyVAL.whenTerms = expression.WhenTerms{&expression.WhenTerm{yyS[yypt-2].expr, yyS[yypt-0].expr}}
		}
	case 270:
		//line n1ql.y:1905
		{
			yyVAL.whenTerms = append(yyS[yypt-4].whenTerms, &expression.WhenTerm{yyS[yypt-2].expr, yyS[yypt-0].expr})
		}
	case 271:
		//line n1ql.y:1913
		{
			yyVAL.expr = expression.NewSearchedCase(yyS[yypt-1].whenTerms, yyS[yypt-0].expr)
		}
	case 272:
		//line n1ql.y:1920
		{
			yyVAL.expr = nil
		}
	case 273:
		//line n1ql.y:1925
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 274:
		//line n1ql.y:1939
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
	case 275:
		//line n1ql.y:1958
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
	case 276:
		//line n1ql.y:1973
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
	case 277:
		yyVAL.s = yyS[yypt-0].s
	case 278:
		yyVAL.expr = yyS[yypt-0].expr
	case 279:
		yyVAL.expr = yyS[yypt-0].expr
	case 280:
		//line n1ql.y:2007
		{
			yyVAL.expr = expression.NewAny(yyS[yypt-2].bindings, yyS[yypt-1].expr)
		}
	case 281:
		//line n1ql.y:2012
		{
			yyVAL.expr = expression.NewAny(yyS[yypt-2].bindings, yyS[yypt-1].expr)
		}
	case 282:
		//line n1ql.y:2017
		{
			yyVAL.expr = expression.NewEvery(yyS[yypt-2].bindings, yyS[yypt-1].expr)
		}
	case 283:
		//line n1ql.y:2024
		{
			yyVAL.bindings = expression.Bindings{yyS[yypt-0].binding}
		}
	case 284:
		//line n1ql.y:2029
		{
			yyVAL.bindings = append(yyS[yypt-2].bindings, yyS[yypt-0].binding)
		}
	case 285:
		//line n1ql.y:2036
		{
			yyVAL.binding = expression.NewBinding(yyS[yypt-2].s, yyS[yypt-0].expr)
		}
	case 286:
		//line n1ql.y:2041
		{
			yyVAL.binding = expression.NewDescendantBinding(yyS[yypt-2].s, yyS[yypt-0].expr)
		}
	case 287:
		//line n1ql.y:2048
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 288:
		//line n1ql.y:2055
		{
			yyVAL.expr = expression.NewArray(yyS[yypt-4].expr, yyS[yypt-2].bindings, yyS[yypt-1].expr)
		}
	case 289:
		//line n1ql.y:2060
		{
			yyVAL.expr = expression.NewFirst(yyS[yypt-4].expr, yyS[yypt-2].bindings, yyS[yypt-1].expr)
		}
	case 290:
		//line n1ql.y:2074
		{
			yyVAL.expr = yyS[yypt-1].expr
		}
	case 291:
		yyVAL.expr = yyS[yypt-0].expr
	case 292:
		//line n1ql.y:2083
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
