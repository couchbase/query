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
const TYPE = 57474
const UNDER = 57475
const UNION = 57476
const UNIQUE = 57477
const UNNEST = 57478
const UNSET = 57479
const UPDATE = 57480
const UPSERT = 57481
const USE = 57482
const USER = 57483
const USING = 57484
const VALUE = 57485
const VALUED = 57486
const VALUES = 57487
const VIEW = 57488
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
const NAMED_PARAM = 57501
const POSITIONAL_PARAM = 57502
const LPAREN = 57503
const RPAREN = 57504
const LBRACE = 57505
const RBRACE = 57506
const LBRACKET = 57507
const RBRACKET = 57508
const RBRACKET_ICASE = 57509
const COMMA = 57510
const COLON = 57511
const INTERESECT = 57512
const EQ = 57513
const DEQ = 57514
const NE = 57515
const LT = 57516
const GT = 57517
const LE = 57518
const GE = 57519
const CONCAT = 57520
const PLUS = 57521
const STAR = 57522
const DIV = 57523
const MOD = 57524
const UMINUS = 57525
const DOT = 57526

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
	161, 281,
	-2, 229,
	-1, 108,
	169, 63,
	-2, 64,
	-1, 144,
	50, 72,
	67, 72,
	86, 72,
	136, 72,
	-2, 50,
	-1, 171,
	171, 0,
	172, 0,
	173, 0,
	-2, 205,
	-1, 172,
	171, 0,
	172, 0,
	173, 0,
	-2, 206,
	-1, 173,
	171, 0,
	172, 0,
	173, 0,
	-2, 207,
	-1, 174,
	174, 0,
	175, 0,
	176, 0,
	177, 0,
	-2, 208,
	-1, 175,
	174, 0,
	175, 0,
	176, 0,
	177, 0,
	-2, 209,
	-1, 176,
	174, 0,
	175, 0,
	176, 0,
	177, 0,
	-2, 210,
	-1, 177,
	174, 0,
	175, 0,
	176, 0,
	177, 0,
	-2, 211,
	-1, 184,
	75, 0,
	-2, 214,
	-1, 185,
	58, 0,
	151, 0,
	-2, 216,
	-1, 186,
	58, 0,
	151, 0,
	-2, 218,
	-1, 279,
	75, 0,
	-2, 215,
	-1, 280,
	58, 0,
	151, 0,
	-2, 217,
	-1, 281,
	58, 0,
	151, 0,
	-2, 219,
}

const yyNprod = 297
const yyPrivate = 57344

var yyTokenNames []string
var yyStates []string

const yyLast = 2715

var yyAct = []int{

	158, 3, 571, 558, 431, 569, 559, 300, 301, 191,
	92, 93, 508, 517, 296, 208, 390, 525, 304, 210,
	133, 484, 248, 225, 445, 95, 255, 406, 204, 209,
	129, 389, 392, 150, 334, 153, 12, 414, 66, 293,
	145, 227, 378, 249, 154, 131, 132, 106, 126, 114,
	271, 220, 118, 52, 256, 447, 321, 319, 130, 273,
	339, 461, 137, 138, 511, 274, 275, 276, 422, 270,
	235, 162, 163, 164, 165, 166, 167, 168, 169, 170,
	171, 172, 173, 174, 175, 176, 177, 421, 119, 184,
	185, 186, 91, 460, 320, 492, 258, 70, 257, 271,
	70, 422, 159, 160, 234, 135, 136, 443, 407, 72,
	161, 130, 73, 74, 75, 147, 69, 222, 270, 69,
	421, 233, 407, 207, 238, 355, 338, 259, 86, 505,
	444, 442, 373, 235, 232, 231, 148, 463, 464, 361,
	362, 271, 8, 237, 107, 465, 245, 363, 487, 312,
	310, 194, 196, 198, 263, 272, 274, 275, 276, 223,
	270, 250, 266, 452, 110, 230, 211, 417, 108, 422,
	229, 229, 397, 89, 265, 237, 350, 148, 72, 134,
	212, 91, 279, 280, 281, 260, 262, 261, 421, 235,
	88, 70, 159, 160, 288, 226, 307, 108, 72, 108,
	161, 294, 127, 303, 76, 71, 73, 74, 75, 108,
	69, 247, 146, 563, 247, 570, 311, 298, 506, 565,
	314, 239, 315, 221, 509, 178, 139, 552, 309, 179,
	273, 308, 299, 206, 323, 548, 324, 302, 180, 327,
	328, 329, 189, 67, 584, 188, 187, 489, 337, 289,
	583, 290, 579, 291, 303, 283, 549, 539, 340, 282,
	70, 68, 354, 433, 454, 99, 90, 228, 228, 359,
	313, 348, 364, 349, 71, 73, 74, 75, 115, 69,
	70, 322, 562, 182, 326, 533, 98, 67, 372, 332,
	333, 236, 121, 76, 71, 73, 74, 75, 381, 69,
	181, 305, 190, 128, 353, 347, 384, 386, 387, 385,
	68, 507, 271, 212, 240, 284, 101, 400, 518, 531,
	393, 105, 395, 179, 449, 277, 272, 274, 275, 276,
	219, 270, 379, 318, 120, 383, 380, 199, 317, 287,
	412, 382, 547, 403, 419, 405, 574, 577, 529, 581,
	396, 580, 306, 575, 68, 530, 97, 147, 250, 401,
	402, 408, 540, 193, 426, 229, 229, 141, 544, 251,
	394, 297, 109, 409, 294, 413, 183, 420, 423, 424,
	418, 435, 411, 65, 434, 67, 103, 436, 437, 416,
	416, 253, 439, 218, 438, 448, 440, 441, 587, 102,
	451, 254, 351, 352, 586, 560, 278, 224, 456, 123,
	122, 130, 179, 430, 214, 179, 179, 179, 179, 179,
	179, 345, 515, 466, 201, 202, 203, 241, 242, 472,
	336, 213, 197, 195, 450, 462, 67, 104, 341, 467,
	468, 458, 459, 476, 481, 478, 479, 523, 457, 477,
	143, 455, 68, 130, 146, 331, 330, 342, 325, 217,
	543, 393, 228, 228, 486, 404, 496, 474, 485, 475,
	582, 410, 200, 2, 480, 493, 501, 482, 346, 343,
	67, 67, 502, 488, 1, 94, 415, 415, 453, 551,
	532, 555, 564, 446, 391, 360, 498, 499, 365, 366,
	367, 368, 369, 370, 388, 483, 503, 344, 432, 473,
	295, 504, 250, 512, 179, 516, 534, 510, 528, 36,
	35, 519, 520, 526, 526, 527, 485, 524, 34, 18,
	17, 16, 15, 538, 77, 536, 537, 535, 14, 542,
	86, 13, 7, 6, 553, 554, 541, 68, 68, 5,
	545, 546, 4, 374, 550, 556, 557, 375, 285, 286,
	561, 572, 192, 566, 568, 567, 573, 292, 96, 100,
	77, 149, 514, 513, 576, 491, 86, 490, 335, 578,
	246, 140, 205, 252, 142, 89, 585, 572, 572, 589,
	590, 588, 144, 91, 63, 64, 28, 429, 117, 27,
	47, 23, 88, 50, 49, 494, 495, 26, 113, 77,
	72, 112, 211, 111, 87, 86, 25, 124, 125, 22,
	78, 89, 44, 43, 20, 19, 0, 0, 0, 91,
	0, 0, 0, 0, 0, 0, 0, 0, 88, 0,
	0, 0, 0, 0, 0, 0, 72, 58, 61, 0,
	87, 0, 0, 0, 0, 0, 78, 48, 0, 0,
	89, 0, 0, 0, 0, 0, 0, 0, 91, 0,
	0, 0, 0, 0, 0, 371, 0, 88, 90, 0,
	0, 60, 0, 0, 0, 72, 0, 38, 62, 87,
	0, 0, 70, 521, 522, 78, 0, 0, 79, 80,
	81, 82, 83, 84, 85, 76, 71, 73, 74, 75,
	0, 69, 0, 0, 90, 0, 0, 0, 0, 0,
	0, 0, 24, 0, 0, 0, 0, 37, 70, 469,
	470, 0, 0, 0, 79, 80, 81, 82, 83, 84,
	85, 76, 71, 73, 74, 75, 77, 69, 0, 0,
	0, 273, 86, 90, 0, 0, 0, 0, 0, 212,
	0, 59, 0, 0, 0, 0, 0, 70, 0, 39,
	0, 0, 0, 79, 80, 81, 82, 83, 84, 85,
	76, 71, 73, 74, 75, 77, 69, 0, 0, 376,
	0, 86, 0, 41, 40, 42, 21, 89, 45, 46,
	51, 0, 56, 0, 57, 91, 0, 0, 0, 0,
	0, 377, 0, 0, 88, 0, 0, 77, 0, 0,
	0, 0, 72, 86, 0, 0, 87, 0, 0, 0,
	0, 0, 78, 271, 0, 0, 89, 0, 0, 0,
	0, 0, 0, 0, 91, 0, 277, 272, 274, 275,
	276, 0, 270, 88, 0, 0, 0, 0, 0, 269,
	0, 72, 0, 0, 0, 87, 0, 0, 89, 0,
	0, 78, 0, 0, 0, 0, 91, 86, 0, 0,
	0, 0, 0, 0, 0, 88, 0, 0, 0, 0,
	90, 0, 0, 72, 0, 0, 0, 87, 0, 0,
	0, 0, 0, 78, 70, 427, 0, 0, 428, 0,
	79, 80, 81, 82, 83, 84, 85, 76, 71, 73,
	74, 75, 89, 69, 0, 86, 0, 0, 0, 90,
	91, 0, 0, 0, 0, 273, 0, 0, 0, 88,
	0, 0, 77, 70, 0, 0, 0, 72, 86, 79,
	80, 81, 82, 83, 84, 85, 76, 71, 73, 74,
	75, 90, 69, 0, 0, 0, 0, 0, 0, 0,
	89, 0, 0, 0, 0, 70, 356, 357, 91, 0,
	0, 79, 80, 81, 82, 83, 84, 85, 76, 71,
	73, 74, 75, 89, 69, 72, 77, 0, 0, 211,
	0, 91, 86, 0, 0, 0, 0, 0, 0, 0,
	88, 0, 0, 0, 0, 90, 0, 271, 72, 0,
	0, 0, 87, 0, 0, 0, 0, 0, 78, 70,
	277, 272, 274, 275, 276, 0, 270, 0, 82, 83,
	84, 85, 76, 71, 73, 74, 75, 89, 69, 0,
	0, 0, 0, 0, 0, 91, 0, 0, 0, 0,
	0, 0, 0, 90, 88, 0, 0, 77, 0, 0,
	0, 0, 72, 86, 0, 0, 87, 70, 0, 0,
	0, 0, 78, 0, 0, 0, 90, 0, 0, 0,
	76, 71, 73, 74, 75, 0, 69, 0, 0, 0,
	70, 267, 0, 0, 268, 0, 79, 80, 81, 82,
	83, 84, 85, 76, 71, 73, 74, 75, 89, 69,
	0, 0, 0, 0, 0, 0, 91, 0, 0, 0,
	77, 0, 0, 0, 0, 88, 86, 0, 0, 0,
	90, 0, 0, 72, 0, 0, 212, 87, 0, 0,
	0, 0, 0, 78, 70, 0, 0, 0, 0, 0,
	79, 80, 81, 82, 83, 84, 85, 76, 71, 73,
	74, 75, 0, 264, 447, 0, 0, 0, 0, 0,
	0, 89, 0, 0, 0, 0, 0, 0, 0, 91,
	0, 0, 0, 0, 0, 77, 0, 0, 88, 0,
	0, 86, 0, 0, 0, 0, 72, 0, 247, 0,
	87, 90, 0, 0, 0, 0, 78, 0, 0, 0,
	0, 0, 0, 0, 0, 70, 0, 0, 0, 0,
	0, 79, 80, 81, 82, 83, 84, 85, 76, 71,
	73, 74, 75, 0, 69, 0, 89, 0, 0, 0,
	0, 0, 0, 0, 91, 77, 0, 0, 0, 0,
	0, 86, 0, 88, 0, 0, 0, 0, 0, 0,
	0, 72, 0, 0, 90, 87, 0, 0, 0, 0,
	0, 78, 0, 0, 0, 0, 0, 0, 70, 0,
	0, 0, 0, 0, 79, 80, 81, 82, 83, 84,
	85, 76, 71, 73, 74, 75, 89, 69, 0, 0,
	0, 0, 0, 0, 91, 0, 0, 0, 0, 0,
	77, 0, 0, 88, 0, 0, 86, 0, 0, 0,
	0, 72, 0, 0, 0, 87, 0, 0, 0, 90,
	0, 78, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 70, 500, 0, 0, 0, 0, 79,
	80, 81, 82, 83, 84, 85, 76, 71, 73, 74,
	75, 89, 69, 0, 0, 0, 0, 0, 0, 91,
	77, 0, 0, 0, 0, 0, 86, 0, 88, 0,
	0, 0, 0, 0, 0, 0, 72, 0, 0, 90,
	87, 0, 0, 0, 0, 0, 78, 0, 0, 0,
	0, 0, 0, 70, 497, 0, 0, 0, 0, 79,
	80, 81, 82, 83, 84, 85, 76, 71, 73, 74,
	75, 89, 69, 0, 0, 0, 0, 0, 0, 91,
	0, 0, 0, 0, 0, 77, 0, 0, 88, 0,
	0, 86, 0, 0, 0, 0, 72, 0, 0, 0,
	87, 0, 0, 0, 90, 0, 78, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 70, 425,
	0, 0, 0, 0, 79, 80, 81, 82, 83, 84,
	85, 76, 71, 73, 74, 75, 89, 69, 0, 399,
	0, 0, 0, 0, 91, 77, 0, 0, 0, 0,
	0, 86, 0, 88, 0, 0, 0, 0, 0, 0,
	0, 72, 0, 0, 90, 87, 0, 0, 0, 0,
	0, 78, 0, 0, 0, 0, 0, 0, 70, 0,
	0, 0, 0, 0, 79, 80, 81, 82, 83, 84,
	85, 76, 71, 73, 74, 75, 89, 69, 0, 0,
	0, 0, 0, 0, 91, 0, 0, 0, 0, 0,
	0, 0, 0, 88, 0, 0, 0, 0, 0, 0,
	0, 72, 77, 0, 0, 87, 0, 0, 86, 90,
	0, 78, 0, 0, 0, 0, 0, 0, 0, 0,
	398, 0, 0, 70, 0, 0, 0, 0, 0, 79,
	80, 81, 82, 83, 84, 85, 76, 71, 73, 74,
	75, 0, 69, 0, 316, 0, 244, 0, 0, 0,
	0, 0, 0, 89, 0, 0, 0, 77, 0, 0,
	0, 91, 0, 86, 0, 0, 0, 0, 0, 90,
	88, 0, 0, 0, 0, 0, 0, 0, 72, 0,
	0, 0, 87, 70, 0, 0, 0, 0, 78, 79,
	80, 81, 82, 83, 84, 85, 76, 71, 73, 74,
	75, 243, 69, 55, 0, 0, 0, 0, 89, 0,
	0, 0, 0, 0, 0, 0, 91, 0, 0, 0,
	0, 0, 0, 0, 53, 88, 0, 0, 0, 31,
	77, 0, 0, 72, 0, 54, 86, 87, 0, 0,
	0, 0, 0, 78, 0, 11, 90, 0, 0, 0,
	67, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	70, 29, 0, 0, 0, 0, 79, 80, 81, 82,
	83, 84, 85, 76, 71, 73, 74, 75, 0, 69,
	33, 89, 77, 0, 0, 0, 0, 0, 86, 91,
	0, 0, 0, 0, 0, 0, 0, 0, 88, 0,
	0, 90, 0, 0, 0, 0, 72, 0, 0, 0,
	87, 0, 0, 0, 0, 70, 78, 68, 0, 0,
	0, 79, 80, 81, 82, 83, 84, 85, 76, 71,
	73, 74, 75, 89, 69, 0, 32, 30, 0, 0,
	0, 91, 0, 0, 0, 0, 0, 0, 0, 0,
	88, 0, 0, 0, 0, 77, 0, 0, 72, 0,
	0, 86, 87, 0, 0, 0, 0, 0, 78, 0,
	116, 0, 0, 0, 90, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 70, 0,
	0, 0, 0, 0, 79, 80, 81, 82, 83, 84,
	85, 76, 71, 73, 74, 75, 89, 69, 0, 0,
	0, 0, 0, 0, 91, 0, 0, 0, 0, 0,
	0, 0, 0, 88, 0, 0, 90, 0, 0, 0,
	0, 72, 0, 0, 0, 87, 0, 0, 0, 0,
	70, 0, 0, 0, 0, 0, 79, 80, 81, 82,
	83, 84, 85, 76, 71, 73, 74, 75, 152, 69,
	0, 0, 58, 61, 0, 0, 0, 0, 0, 0,
	0, 0, 48, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 86, 0, 0, 151,
	0, 0, 0, 156, 0, 0, 60, 0, 0, 90,
	10, 0, 38, 62, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 70, 0, 0, 0, 0, 0, 79,
	80, 81, 82, 83, 84, 85, 76, 71, 73, 74,
	75, 89, 69, 0, 0, 0, 0, 24, 0, 91,
	0, 9, 37, 0, 0, 0, 0, 0, 88, 0,
	0, 0, 0, 0, 0, 0, 72, 58, 61, 0,
	87, 155, 0, 0, 0, 0, 0, 48, 0, 0,
	0, 0, 0, 0, 0, 0, 59, 0, 0, 0,
	0, 0, 0, 0, 39, 0, 0, 0, 156, 0,
	0, 60, 0, 0, 0, 10, 0, 38, 62, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 41, 40,
	42, 21, 0, 45, 46, 51, 0, 56, 0, 57,
	0, 0, 0, 0, 90, 0, 0, 0, 0, 0,
	0, 0, 24, 0, 157, 0, 9, 37, 70, 0,
	0, 0, 0, 0, 79, 80, 81, 82, 83, 84,
	85, 76, 71, 73, 74, 75, 155, 69, 58, 61,
	0, 0, 0, 0, 0, 0, 0, 0, 48, 0,
	0, 59, 0, 0, 0, 0, 0, 0, 0, 39,
	0, 0, 0, 0, 0, 215, 0, 0, 0, 0,
	0, 0, 60, 0, 0, 0, 10, 0, 38, 62,
	0, 0, 0, 41, 40, 42, 21, 0, 45, 46,
	51, 0, 56, 0, 57, 58, 61, 0, 0, 0,
	0, 0, 0, 0, 0, 48, 0, 0, 0, 157,
	0, 0, 0, 24, 0, 0, 0, 9, 37, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 60,
	0, 0, 0, 10, 0, 38, 62, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 59, 0, 0, 0, 0, 0, 0, 0,
	39, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	24, 0, 0, 0, 9, 37, 0, 0, 0, 55,
	0, 0, 58, 61, 41, 40, 42, 21, 0, 45,
	46, 51, 48, 56, 0, 57, 0, 0, 0, 0,
	53, 0, 0, 0, 0, 31, 0, 0, 0, 59,
	216, 54, 0, 0, 0, 0, 60, 39, 0, 0,
	10, 11, 38, 62, 0, 0, 67, 0, 0, 0,
	58, 61, 0, 0, 0, 0, 0, 29, 0, 0,
	48, 41, 40, 42, 21, 0, 45, 46, 51, 0,
	56, 0, 57, 0, 0, 0, 33, 24, 0, 0,
	0, 9, 37, 0, 60, 0, 0, 157, 10, 0,
	38, 62, 0, 0, 0, 0, 0, 0, 0, 0,
	58, 61, 0, 0, 0, 0, 0, 0, 0, 0,
	48, 0, 0, 68, 0, 0, 59, 0, 0, 0,
	0, 0, 0, 0, 39, 24, 0, 0, 0, 9,
	37, 0, 32, 30, 60, 0, 0, 0, 10, 0,
	38, 62, 0, 0, 0, 0, 0, 0, 41, 40,
	42, 21, 0, 45, 46, 51, 0, 56, 0, 57,
	0, 0, 0, 0, 59, 0, 0, 0, 0, 0,
	0, 0, 39, 0, 0, 24, 0, 0, 0, 9,
	37, 0, 0, 0, 0, 58, 61, 0, 0, 0,
	0, 0, 0, 0, 0, 48, 41, 40, 42, 21,
	0, 45, 46, 51, 0, 56, 0, 57, 471, 0,
	0, 0, 0, 0, 59, 0, 0, 0, 0, 60,
	0, 0, 39, 10, 0, 38, 62, 0, 0, 67,
	0, 0, 0, 58, 61, 0, 0, 0, 0, 0,
	0, 0, 0, 48, 0, 0, 41, 40, 42, 21,
	0, 45, 46, 51, 0, 56, 0, 57, 358, 0,
	24, 0, 0, 0, 9, 37, 0, 60, 0, 0,
	0, 10, 0, 38, 62, 0, 0, 58, 61, 0,
	0, 0, 0, 0, 0, 0, 0, 48, 0, 0,
	0, 0, 0, 0, 0, 0, 68, 0, 0, 59,
	0, 0, 0, 0, 0, 0, 0, 39, 24, 0,
	0, 60, 9, 37, 0, 10, 0, 38, 62, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 41, 40, 42, 21, 0, 45, 46, 51, 0,
	56, 0, 57, 0, 0, 0, 0, 59, 0, 0,
	0, 0, 24, 0, 0, 39, 9, 37, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 116, 0, 0, 0, 0, 0, 0, 41,
	40, 42, 21, 0, 45, 46, 51, 0, 56, 0,
	57, 59, 0, 0, 0, 0, 0, 0, 0, 39,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 41, 40, 42, 21, 0, 45, 46,
	51, 0, 56, 0, 57,
}
var yyPact = []int{

	2274, -1000, -1000, 1755, -1000, -1000, -1000, -1000, -1000, 2549,
	2549, 1678, -1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000,
	-1000, -1000, -1000, -1000, 2549, -1000, -1000, -1000, 222, 334,
	321, 385, 40, 307, -1000, -1000, -1000, -1000, -1000, -1000,
	-1000, -1000, -1000, -1000, -1000, -1000, -1000, 3, 2505, -1000,
	-1000, 2457, -1000, 232, 350, 349, 46, 2549, 22, 22,
	22, 2549, 2549, -1000, -1000, 294, 384, 52, 1934, 35,
	2549, 2549, 2549, 2549, 2549, 2549, 2549, 2549, 2549, 2549,
	2549, 2549, 2549, 2549, 2549, 2549, 639, 225, 2549, 2549,
	2549, 158, 1953, 26, -1000, -65, 287, 429, 428, 333,
	-1000, 456, 40, 40, 40, 93, -46, 156, -1000, 40,
	2130, 418, -1000, -1000, 1703, 183, 2549, -3, 1755, -1000,
	347, 38, 40, 40, -29, -34, -1000, -48, -62, -35,
	1755, 7, -1000, 163, -1000, 7, 7, 1630, 1575, 63,
	-1000, 23, 294, -1000, 329, -1000, -130, -71, -73, -1000,
	-41, 2029, 2187, 2549, -1000, -1000, -1000, -1000, 989, -1000,
	-1000, 2549, 935, -68, -68, -65, -65, -65, 95, 1953,
	1828, 864, 864, 864, 115, 115, 115, 115, 852, -1000,
	639, 2549, 2549, 2549, 912, 26, 26, -1000, 171, -1000,
	-1000, 249, -1000, 2549, -1000, 235, -1000, 235, -1000, 235,
	2549, 303, 303, 93, 117, -1000, 199, 39, -1000, -1000,
	-1000, 23, -1000, 86, -12, 2549, -13, -1000, 183, 2549,
	-1000, 2549, 1498, -1000, 247, 242, -1000, -127, -1000, -75,
	-128, -1000, 46, 2549, -1000, 2549, 417, 22, 2549, 2549,
	2549, 415, 414, 22, 22, 375, -1000, 2549, -42, -1000,
	-111, 63, 371, -1000, 210, 156, 19, 39, 39, 2187,
	-41, 2549, -41, 1755, -55, -1000, 810, -1000, 2372, 639,
	-18, 2549, 639, 639, 639, 639, 639, 639, 668, 912,
	26, 26, -1000, -1000, -1000, -1000, -1000, 2549, 1755, -1000,
	-1000, -1000, -36, -1000, 778, 191, -1000, 2549, 191, 63,
	66, 63, 19, 19, 301, -1000, 156, -1000, -1000, 11,
	-1000, 1438, -1000, -1000, 1373, 1755, 2549, 40, 40, 38,
	39, 38, -1000, 1755, 1755, -1000, -1000, 1755, 1755, 1755,
	-1000, -1000, -25, -25, 142, -1000, 455, 1755, 23, 2549,
	375, 42, 42, 2549, -1000, -1000, -1000, -1000, 93, -97,
	-1000, -130, -130, -1000, 1755, -1000, -1000, -1000, -1000, 1313,
	147, -1000, -1000, 2549, 739, -115, -115, -66, -66, -66,
	-24, 639, 1755, 2549, -1000, -1000, -1000, -1000, 151, 151,
	2549, 1755, 151, 151, 287, 63, 287, 287, -37, -1000,
	-64, -38, -1000, 4, 2549, -1000, 233, 235, -1000, 2549,
	1755, -1000, 2, -1000, -1000, 154, 410, 2549, 407, -1000,
	2549, -1000, 1755, -1000, -1000, -130, -76, -108, -1000, 602,
	-1000, -20, 2549, 156, 156, -1000, 563, -1000, 2322, 147,
	-1000, -1000, -1000, 2029, -1000, 1755, -1000, -1000, 151, 287,
	151, 151, 19, 2549, 19, -1000, -1000, 22, 1755, 303,
	-14, 1755, 2549, -1000, 120, -1000, 1755, -1000, 21, 156,
	39, 39, -1000, -1000, -1000, 2549, 1248, 93, 93, -1000,
	-1000, -1000, 1188, -1000, -41, 2549, -1000, 151, -1000, -1000,
	-1000, 1123, -1000, -39, -1000, 160, 77, 156, -98, 38,
	366, -1000, 23, 227, -130, -130, 527, -1000, -1000, -1000,
	-1000, 1755, -1000, -1000, 406, 22, 19, 19, 287, 268,
	228, 188, -1000, -1000, -1000, 2549, -42, -1000, 199, 156,
	156, -1000, -1000, -1000, -1000, -1000, -97, -1000, 151, 131,
	282, 303, 63, 444, 1755, 299, 227, 227, -1000, 204,
	130, 77, 85, 2549, 2549, -1000, -1000, 117, 63, 342,
	287, -1000, 136, 1755, 1755, 72, 66, 63, 68, -1000,
	2549, 151, -1000, -1000, -1000, 266, -1000, 63, -1000, -1000,
	260, -1000, 1060, -1000, 126, 271, -1000, 269, -1000, 439,
	124, 118, 63, 341, 335, 68, 2549, 2549, -1000, -1000,
	-1000,
}
var yyPgo = []int{

	0, 625, 624, 623, 622, 619, 48, 618, 617, 0,
	142, 225, 30, 303, 43, 22, 19, 29, 15, 20,
	616, 613, 611, 608, 51, 278, 607, 604, 603, 46,
	45, 291, 27, 601, 600, 599, 598, 36, 596, 53,
	595, 594, 592, 383, 584, 40, 37, 583, 16, 26,
	47, 144, 582, 28, 13, 226, 581, 6, 580, 34,
	578, 577, 575, 573, 572, 44, 33, 571, 38, 569,
	568, 39, 567, 562, 9, 559, 558, 557, 553, 473,
	552, 549, 543, 542, 541, 538, 532, 531, 530, 529,
	528, 520, 519, 321, 42, 14, 510, 509, 508, 4,
	21, 505, 17, 7, 31, 504, 8, 32, 494, 493,
	24, 12, 492, 491, 3, 2, 5, 23, 41, 490,
	489, 488, 484, 35, 479, 18, 478,
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
	117, 118, 118, 119, 119, 120, 120, 120, 91, 92,
	121, 121, 48, 48, 48, 48, 48, 48, 9, 9,
	9, 9, 9, 9, 9, 9, 9, 9, 9, 9,
	9, 9, 9, 9, 9, 9, 9, 9, 9, 9,
	9, 9, 9, 9, 9, 9, 9, 9, 9, 9,
	9, 9, 9, 9, 9, 9, 9, 10, 10, 10,
	10, 10, 10, 10, 10, 10, 11, 11, 11, 11,
	11, 11, 11, 11, 11, 11, 11, 11, 11, 11,
	1, 1, 1, 1, 1, 1, 2, 2, 3, 8,
	8, 7, 7, 6, 4, 13, 13, 5, 5, 20,
	21, 21, 22, 25, 25, 23, 24, 24, 33, 33,
	33, 34, 26, 26, 27, 27, 27, 30, 30, 29,
	29, 31, 28, 28, 35, 36, 36,
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
	1, 1, 2, 1, 1, 1, 1, 3, 3, 5,
	5, 4, 5, 6, 3, 3, 3, 3, 3, 3,
	1, 1, 1, 1, 1, 1, 1, 1, 3, 0,
	1, 1, 3, 3, 3, 0, 1, 1, 1, 3,
	1, 1, 3, 4, 5, 2, 0, 2, 4, 5,
	4, 1, 1, 1, 4, 4, 4, 1, 3, 3,
	3, 2, 6, 6, 3, 1, 1,
}
var yyChk = []int{

	-1000, -122, -79, -9, -80, -81, -82, -83, -10, 87,
	46, 47, -37, -84, -85, -86, -87, -88, -89, -1,
	-2, 157, -5, -33, 83, -20, -26, -35, -38, 63,
	139, 31, 138, 82, -90, -91, -92, 88, 48, 130,
	155, 154, 156, -3, -4, 159, 160, -34, 18, -27,
	-28, 161, -39, 26, 37, 5, 163, 165, 8, 122,
	42, 9, 49, -41, -40, -43, -68, 52, 119, 184,
	165, 179, 83, 180, 181, 182, 178, 7, 93, 171,
	172, 173, 174, 175, 176, 177, 13, 87, 75, 58,
	151, 66, -9, -9, -79, -9, -70, 134, 64, 43,
	-69, 94, 65, 65, 52, -93, -50, -51, 157, 65,
	161, -21, -22, -23, -9, -25, 147, -36, -9, -37,
	102, 60, 60, 60, -8, -7, -6, 156, -13, -12,
	-9, -30, -29, -19, 157, -30, -30, -9, -9, -55,
	-56, 73, -44, -43, -42, -45, -51, -50, 125, -67,
	-66, 35, 4, -123, -65, 107, 39, 180, -9, 157,
	158, 165, -9, -9, -9, -9, -9, -9, -9, -9,
	-9, -9, -9, -9, -9, -9, -9, -9, -11, -10,
	13, 75, 58, 151, -9, -9, -9, 88, 87, 84,
	144, -74, -73, 76, -39, 4, -39, 4, -39, 4,
	16, -93, -93, -93, -53, -52, 140, 169, -18, -17,
	-16, 10, 157, -93, -13, 35, 180, 41, -25, 147,
	-24, 40, -9, 162, 60, -117, 157, -118, -51, -50,
	-118, 164, 168, 169, 166, 168, -31, 168, 117, 58,
	151, -31, -31, 51, 51, -57, -58, 148, -15, -14,
	-16, -55, -47, 62, 72, -49, 184, 169, 169, 168,
	-66, -123, -66, -9, 184, -18, -9, 166, 169, 7,
	184, 165, 179, 83, 180, 181, 182, 178, -11, -9,
	-9, -9, 88, 84, 144, -76, -75, 90, -9, -39,
	-39, -39, -72, -71, -9, -96, -95, 68, -95, -53,
	-103, -106, 120, 137, -125, 102, -51, 157, -16, 142,
	162, -9, 162, -24, -9, -9, 126, 91, 91, 184,
	169, 184, -6, -9, -9, 41, -29, -9, -9, -9,
	41, 41, -30, -30, -59, -60, 55, -9, 168, 171,
	-57, 67, 86, -124, 136, 50, -126, 95, -18, -48,
	157, -51, -51, -65, -9, 180, 166, 167, 166, -9,
	-11, 157, 158, 165, -9, -11, -11, -11, -11, -11,
	-11, 7, -9, 168, -78, -77, 11, 33, -94, -37,
	145, -9, -94, -37, -57, -106, -57, -57, -105, -104,
	-48, -108, -107, -48, 69, -18, -45, 161, 162, 126,
	-9, -118, -118, -117, -51, -117, -32, 147, -32, -68,
	16, -14, -9, -59, -46, -51, -50, 125, -46, -9,
	-53, 184, 165, -49, -49, 166, -9, 166, 169, -11,
	-71, -99, -98, 112, -99, -9, -99, -99, -74, -57,
	-74, -74, 168, 171, 168, -110, -109, 51, -9, 91,
	-37, -9, 161, -121, 110, 41, -9, 41, -12, -49,
	169, 169, -18, 157, 158, 165, -9, -18, -18, 166,
	167, 166, -9, -97, -66, -123, -99, -74, -99, -99,
	-104, -9, -107, -101, -100, -19, -95, 162, -12, 127,
	-61, -62, 74, -18, -51, -51, -9, 166, -53, -53,
	166, -9, -99, -110, -32, 168, 58, 151, -111, 147,
	-17, 162, -117, -63, -64, 56, -15, -54, 91, -49,
	-49, 166, 167, 41, -100, -102, -48, -102, -74, 80,
	87, 91, -119, 97, -9, -125, -18, -18, -99, 126,
	80, -95, -57, 16, 69, -54, -54, 138, 31, 126,
	-111, -120, 142, -9, -9, -113, -103, -106, -114, -57,
	63, -74, 146, 77, -112, 147, -57, -106, -57, -116,
	147, -115, -9, -99, 80, 87, -57, 87, -57, 126,
	80, 80, 31, 126, 126, -114, 63, 63, -116, -115,
	-115,
}
var yyDef = []int{

	0, -2, 1, 2, 3, 4, 5, 6, 188, 0,
	0, 0, 8, 9, 10, 11, 12, 13, 14, 227,
	228, -2, 230, 231, 0, 233, 234, 235, 98, 0,
	0, 0, 0, 0, 15, 16, 17, 250, 251, 252,
	253, 254, 255, 256, 257, 267, 268, 0, 0, 282,
	283, 0, 19, 0, 0, 0, 259, 265, 0, 0,
	0, 0, 0, 26, 27, 78, 48, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 204, 226, 7, 232, 108, 0, 0, 0,
	99, 0, 0, 0, 0, 67, 0, 43, -2, 0,
	265, 0, 270, 271, 0, 276, 0, 0, 295, 296,
	0, 0, 0, 0, 0, 260, 261, 0, 0, 266,
	90, 0, 287, 0, 144, 0, 0, 0, 0, 84,
	79, 0, 78, 49, -2, 51, 65, 0, 0, 30,
	31, 0, 0, 0, 38, 36, 37, 40, 43, 189,
	190, 0, 0, 196, 197, 198, 199, 200, 201, 202,
	203, -2, -2, -2, -2, -2, -2, -2, 0, 236,
	0, 0, 0, 0, -2, -2, -2, 220, 0, 222,
	224, 111, 109, 0, 20, 0, 22, 0, 24, 0,
	0, 118, 0, 67, 0, 68, 70, 0, 117, 44,
	45, 0, 47, 0, 0, 0, 0, 269, 276, 0,
	275, 0, 0, 294, 0, 0, 170, 0, 171, 0,
	0, 258, 0, 0, 264, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 87, 85, 0, 80, 81,
	0, 84, 0, 73, 75, 43, 0, 0, 0, 0,
	32, 0, 33, 34, 0, 42, 0, 193, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, -2,
	-2, -2, 221, 223, 225, 18, 112, 0, 110, 21,
	23, 25, 100, 101, 104, 0, 119, 0, 0, 84,
	84, 84, 0, 0, 0, 71, 43, 64, 46, 0,
	278, 0, 280, 272, 0, 277, 0, 0, 0, 0,
	0, 0, 262, 263, 91, 284, 288, 291, 289, 290,
	285, 286, 146, 146, 0, 88, 0, 86, 0, 0,
	87, 0, 0, 0, 55, 56, 74, 76, 67, 66,
	182, 65, 65, 39, 35, 41, 191, 192, 194, 0,
	212, 237, 238, 0, 0, 244, 245, 246, 247, 248,
	249, 0, 113, 0, 103, 105, 106, 107, 122, 122,
	0, 120, 122, 122, 108, 84, 108, 108, 133, 134,
	0, 148, 149, 137, 0, 116, 0, 0, 279, 0,
	273, 168, 0, 178, 172, 180, 0, 0, 0, 28,
	0, 82, 83, 29, 52, 65, 0, 0, 53, 43,
	57, 0, 0, 43, 43, 195, 0, 241, 0, 213,
	102, 114, 123, 0, 115, 121, 127, 128, 122, 108,
	122, 122, 0, 0, 0, 151, 138, 0, 69, 0,
	0, 274, 0, 179, 0, 292, 147, 293, 92, 43,
	0, 0, 54, 183, 184, 0, 0, 67, 67, 239,
	240, 242, 0, 124, 125, 0, 129, 122, 131, 132,
	135, 137, 150, 146, 140, 0, 154, 0, 0, 0,
	95, 93, 0, 0, 65, 65, 0, 187, 58, 59,
	243, 126, 130, 136, 0, 0, 0, 0, 108, 0,
	0, 173, 181, 89, 96, 0, 94, 60, 70, 43,
	43, 185, 186, 139, 141, 142, 145, 143, 122, 0,
	0, 0, 84, 0, 97, 0, 0, 0, 152, 0,
	0, 154, 175, 0, 0, 61, 62, 0, 84, 0,
	108, 169, 0, 174, 77, 158, 84, 84, 161, 166,
	0, 122, 176, 177, 155, 0, 163, 84, 165, 156,
	0, 157, 84, 153, 0, 0, 164, 0, 167, 0,
	0, 0, 84, 0, 0, 161, 0, 0, 159, 160,
	162,
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
	182, 183, 184,
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
		//line n1ql.y:343
		{
			yylex.(*lexer).setStatement(yyS[yypt-0].statement)
		}
	case 2:
		//line n1ql.y:348
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
		//line n1ql.y:365
		{
			yyVAL.statement = algebra.NewExplain(yyS[yypt-0].statement)
		}
	case 8:
		//line n1ql.y:372
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
		//line n1ql.y:403
		{
			yyVAL.fullselect = algebra.NewSelect(yyS[yypt-3].subresult, yyS[yypt-2].order, yyS[yypt-0].expr, yyS[yypt-1].expr) /* OFFSET precedes LIMIT */
		}
	case 19:
		//line n1ql.y:409
		{
			yyVAL.subresult = yyS[yypt-0].subselect
		}
	case 20:
		//line n1ql.y:414
		{
			yyVAL.subresult = algebra.NewUnion(yyS[yypt-2].subresult, yyS[yypt-0].subselect)
		}
	case 21:
		//line n1ql.y:419
		{
			yyVAL.subresult = algebra.NewUnionAll(yyS[yypt-3].subresult, yyS[yypt-0].subselect)
		}
	case 22:
		//line n1ql.y:424
		{
			yyVAL.subresult = algebra.NewIntersect(yyS[yypt-2].subresult, yyS[yypt-0].subselect)
		}
	case 23:
		//line n1ql.y:429
		{
			yyVAL.subresult = algebra.NewIntersectAll(yyS[yypt-3].subresult, yyS[yypt-0].subselect)
		}
	case 24:
		//line n1ql.y:434
		{
			yyVAL.subresult = algebra.NewExcept(yyS[yypt-2].subresult, yyS[yypt-0].subselect)
		}
	case 25:
		//line n1ql.y:439
		{
			yyVAL.subresult = algebra.NewExceptAll(yyS[yypt-3].subresult, yyS[yypt-0].subselect)
		}
	case 26:
		yyVAL.subselect = yyS[yypt-0].subselect
	case 27:
		yyVAL.subselect = yyS[yypt-0].subselect
	case 28:
		//line n1ql.y:452
		{
			yyVAL.subselect = algebra.NewSubselect(yyS[yypt-4].fromTerm, yyS[yypt-3].bindings, yyS[yypt-2].expr, yyS[yypt-1].group, yyS[yypt-0].projection)
		}
	case 29:
		//line n1ql.y:459
		{
			yyVAL.subselect = algebra.NewSubselect(yyS[yypt-3].fromTerm, yyS[yypt-2].bindings, yyS[yypt-1].expr, yyS[yypt-0].group, yyS[yypt-4].projection)
		}
	case 30:
		//line n1ql.y:474
		{
			yyVAL.projection = yyS[yypt-0].projection
		}
	case 31:
		//line n1ql.y:481
		{
			yyVAL.projection = algebra.NewProjection(false, yyS[yypt-0].resultTerms)
		}
	case 32:
		//line n1ql.y:486
		{
			yyVAL.projection = algebra.NewProjection(true, yyS[yypt-0].resultTerms)
		}
	case 33:
		//line n1ql.y:491
		{
			yyVAL.projection = algebra.NewProjection(false, yyS[yypt-0].resultTerms)
		}
	case 34:
		//line n1ql.y:496
		{
			yyVAL.projection = algebra.NewRawProjection(false, yyS[yypt-0].expr)
		}
	case 35:
		//line n1ql.y:501
		{
			yyVAL.projection = algebra.NewRawProjection(true, yyS[yypt-0].expr)
		}
	case 38:
		//line n1ql.y:514
		{
			yyVAL.resultTerms = algebra.ResultTerms{yyS[yypt-0].resultTerm}
		}
	case 39:
		//line n1ql.y:519
		{
			yyVAL.resultTerms = append(yyS[yypt-2].resultTerms, yyS[yypt-0].resultTerm)
		}
	case 40:
		//line n1ql.y:526
		{
			yyVAL.resultTerm = algebra.NewResultTerm(nil, true, "")
		}
	case 41:
		//line n1ql.y:531
		{
			yyVAL.resultTerm = algebra.NewResultTerm(yyS[yypt-2].expr, true, "")
		}
	case 42:
		//line n1ql.y:536
		{
			yyVAL.resultTerm = algebra.NewResultTerm(yyS[yypt-1].expr, false, yyS[yypt-0].s)
		}
	case 43:
		//line n1ql.y:543
		{
			yyVAL.s = ""
		}
	case 44:
		yyVAL.s = yyS[yypt-0].s
	case 45:
		yyVAL.s = yyS[yypt-0].s
	case 46:
		//line n1ql.y:554
		{
			yyVAL.s = yyS[yypt-0].s
		}
	case 47:
		yyVAL.s = yyS[yypt-0].s
	case 48:
		//line n1ql.y:572
		{
			yyVAL.fromTerm = nil
		}
	case 49:
		yyVAL.fromTerm = yyS[yypt-0].fromTerm
	case 50:
		//line n1ql.y:581
		{
			yyVAL.fromTerm = yyS[yypt-0].fromTerm
		}
	case 51:
		//line n1ql.y:588
		{
			yyVAL.fromTerm = yyS[yypt-0].keyspaceTerm
		}
	case 52:
		//line n1ql.y:593
		{
			yyVAL.fromTerm = algebra.NewJoin(yyS[yypt-3].fromTerm, yyS[yypt-2].b, yyS[yypt-0].keyspaceTerm)
		}
	case 53:
		//line n1ql.y:598
		{
			yyVAL.fromTerm = algebra.NewNest(yyS[yypt-3].fromTerm, yyS[yypt-2].b, yyS[yypt-0].keyspaceTerm)
		}
	case 54:
		//line n1ql.y:603
		{
			yyVAL.fromTerm = algebra.NewUnnest(yyS[yypt-4].fromTerm, yyS[yypt-3].b, yyS[yypt-1].expr, yyS[yypt-0].s)
		}
	case 57:
		//line n1ql.y:616
		{
			yyVAL.keyspaceTerm = algebra.NewKeyspaceTerm("", yyS[yypt-3].s, yyS[yypt-2].path, yyS[yypt-1].s, yyS[yypt-0].expr)
		}
	case 58:
		//line n1ql.y:621
		{
			yyVAL.keyspaceTerm = algebra.NewKeyspaceTerm(yyS[yypt-5].s, yyS[yypt-3].s, yyS[yypt-2].path, yyS[yypt-1].s, yyS[yypt-0].expr)
		}
	case 59:
		//line n1ql.y:626
		{
			yyVAL.keyspaceTerm = algebra.NewKeyspaceTerm("#system", yyS[yypt-3].s, yyS[yypt-2].path, yyS[yypt-1].s, yyS[yypt-0].expr)
		}
	case 60:
		//line n1ql.y:633
		{
			yyVAL.keyspaceTerm = algebra.NewKeyspaceTerm("", yyS[yypt-3].s, yyS[yypt-2].path, yyS[yypt-1].s, yyS[yypt-0].expr)
		}
	case 61:
		//line n1ql.y:638
		{
			yyVAL.keyspaceTerm = algebra.NewKeyspaceTerm(yyS[yypt-5].s, yyS[yypt-3].s, yyS[yypt-2].path, yyS[yypt-1].s, yyS[yypt-0].expr)
		}
	case 62:
		//line n1ql.y:643
		{
			yyVAL.keyspaceTerm = algebra.NewKeyspaceTerm("#system", yyS[yypt-3].s, yyS[yypt-2].path, yyS[yypt-1].s, yyS[yypt-0].expr)
		}
	case 63:
		yyVAL.s = yyS[yypt-0].s
	case 64:
		yyVAL.s = yyS[yypt-0].s
	case 65:
		//line n1ql.y:658
		{
			yyVAL.path = nil
		}
	case 66:
		//line n1ql.y:663
		{
			yyVAL.path = yyS[yypt-0].path
		}
	case 67:
		//line n1ql.y:670
		{
			yyVAL.expr = nil
		}
	case 68:
		yyVAL.expr = yyS[yypt-0].expr
	case 69:
		//line n1ql.y:679
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 70:
		//line n1ql.y:686
		{
		}
	case 72:
		//line n1ql.y:694
		{
			yyVAL.b = false
		}
	case 73:
		//line n1ql.y:699
		{
			yyVAL.b = false
		}
	case 74:
		//line n1ql.y:704
		{
			yyVAL.b = true
		}
	case 77:
		//line n1ql.y:717
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 78:
		//line n1ql.y:731
		{
			yyVAL.bindings = nil
		}
	case 79:
		yyVAL.bindings = yyS[yypt-0].bindings
	case 80:
		//line n1ql.y:740
		{
			yyVAL.bindings = yyS[yypt-0].bindings
		}
	case 81:
		//line n1ql.y:747
		{
			yyVAL.bindings = expression.Bindings{yyS[yypt-0].binding}
		}
	case 82:
		//line n1ql.y:752
		{
			yyVAL.bindings = append(yyS[yypt-2].bindings, yyS[yypt-0].binding)
		}
	case 83:
		//line n1ql.y:759
		{
			yyVAL.binding = expression.NewBinding(yyS[yypt-2].s, yyS[yypt-0].expr)
		}
	case 84:
		//line n1ql.y:773
		{
			yyVAL.expr = nil
		}
	case 85:
		yyVAL.expr = yyS[yypt-0].expr
	case 86:
		//line n1ql.y:782
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 87:
		//line n1ql.y:796
		{
			yyVAL.group = nil
		}
	case 88:
		yyVAL.group = yyS[yypt-0].group
	case 89:
		//line n1ql.y:805
		{
			yyVAL.group = algebra.NewGroup(yyS[yypt-2].exprs, yyS[yypt-1].bindings, yyS[yypt-0].expr)
		}
	case 90:
		//line n1ql.y:812
		{
			yyVAL.exprs = expression.Expressions{yyS[yypt-0].expr}
		}
	case 91:
		//line n1ql.y:817
		{
			yyVAL.exprs = append(yyS[yypt-2].exprs, yyS[yypt-0].expr)
		}
	case 92:
		//line n1ql.y:824
		{
			yyVAL.bindings = nil
		}
	case 93:
		yyVAL.bindings = yyS[yypt-0].bindings
	case 94:
		//line n1ql.y:833
		{
			yyVAL.bindings = yyS[yypt-0].bindings
		}
	case 95:
		//line n1ql.y:840
		{
			yyVAL.expr = nil
		}
	case 96:
		yyVAL.expr = yyS[yypt-0].expr
	case 97:
		//line n1ql.y:849
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 98:
		//line n1ql.y:863
		{
			yyVAL.order = nil
		}
	case 99:
		yyVAL.order = yyS[yypt-0].order
	case 100:
		//line n1ql.y:872
		{
			yyVAL.order = algebra.NewOrder(yyS[yypt-0].sortTerms)
		}
	case 101:
		//line n1ql.y:879
		{
			yyVAL.sortTerms = algebra.SortTerms{yyS[yypt-0].sortTerm}
		}
	case 102:
		//line n1ql.y:884
		{
			yyVAL.sortTerms = append(yyS[yypt-2].sortTerms, yyS[yypt-0].sortTerm)
		}
	case 103:
		//line n1ql.y:891
		{
			yyVAL.sortTerm = algebra.NewSortTerm(yyS[yypt-1].expr, yyS[yypt-0].b)
		}
	case 104:
		//line n1ql.y:898
		{
			yyVAL.b = false
		}
	case 105:
		yyVAL.b = yyS[yypt-0].b
	case 106:
		//line n1ql.y:907
		{
			yyVAL.b = false
		}
	case 107:
		//line n1ql.y:912
		{
			yyVAL.b = true
		}
	case 108:
		//line n1ql.y:926
		{
			yyVAL.expr = nil
		}
	case 109:
		yyVAL.expr = yyS[yypt-0].expr
	case 110:
		//line n1ql.y:935
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 111:
		//line n1ql.y:949
		{
			yyVAL.expr = nil
		}
	case 112:
		yyVAL.expr = yyS[yypt-0].expr
	case 113:
		//line n1ql.y:958
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 114:
		//line n1ql.y:972
		{
			yyVAL.statement = algebra.NewInsertValues(yyS[yypt-3].keyspaceRef, yyS[yypt-2].expr, yyS[yypt-1].expr, yyS[yypt-0].projection)
		}
	case 115:
		//line n1ql.y:977
		{
			yyVAL.statement = algebra.NewInsertSelect(yyS[yypt-3].keyspaceRef, yyS[yypt-2].expr, yyS[yypt-1].fullselect, yyS[yypt-0].projection)
		}
	case 116:
		//line n1ql.y:984
		{
			yyVAL.keyspaceRef = algebra.NewKeyspaceRef(yyS[yypt-3].s, yyS[yypt-1].s, yyS[yypt-0].s)
		}
	case 117:
		//line n1ql.y:989
		{
			yyVAL.keyspaceRef = algebra.NewKeyspaceRef("", yyS[yypt-1].s, yyS[yypt-0].s)
		}
	case 118:
		//line n1ql.y:996
		{
			yyVAL.expr = nil
		}
	case 119:
		yyVAL.expr = yyS[yypt-0].expr
	case 120:
		//line n1ql.y:1005
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 121:
		//line n1ql.y:1012
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 122:
		//line n1ql.y:1019
		{
			yyVAL.projection = nil
		}
	case 123:
		yyVAL.projection = yyS[yypt-0].projection
	case 124:
		//line n1ql.y:1028
		{
			yyVAL.projection = yyS[yypt-0].projection
		}
	case 125:
		//line n1ql.y:1035
		{
			yyVAL.projection = algebra.NewProjection(false, yyS[yypt-0].resultTerms)
		}
	case 126:
		//line n1ql.y:1040
		{
			yyVAL.projection = algebra.NewRawProjection(false, yyS[yypt-0].expr)
		}
	case 127:
		//line n1ql.y:1054
		{
			yyVAL.statement = algebra.NewUpsertValues(yyS[yypt-3].keyspaceRef, yyS[yypt-2].expr, yyS[yypt-1].expr, yyS[yypt-0].projection)
		}
	case 128:
		//line n1ql.y:1059
		{
			yyVAL.statement = algebra.NewUpsertSelect(yyS[yypt-3].keyspaceRef, yyS[yypt-2].expr, yyS[yypt-1].fullselect, yyS[yypt-0].projection)
		}
	case 129:
		//line n1ql.y:1073
		{
			yyVAL.statement = algebra.NewDelete(yyS[yypt-4].keyspaceRef, yyS[yypt-3].expr, yyS[yypt-2].expr, yyS[yypt-1].expr, yyS[yypt-0].projection)
		}
	case 130:
		//line n1ql.y:1087
		{
			yyVAL.statement = algebra.NewUpdate(yyS[yypt-6].keyspaceRef, yyS[yypt-5].expr, yyS[yypt-4].set, yyS[yypt-3].unset, yyS[yypt-2].expr, yyS[yypt-1].expr, yyS[yypt-0].projection)
		}
	case 131:
		//line n1ql.y:1092
		{
			yyVAL.statement = algebra.NewUpdate(yyS[yypt-5].keyspaceRef, yyS[yypt-4].expr, yyS[yypt-3].set, nil, yyS[yypt-2].expr, yyS[yypt-1].expr, yyS[yypt-0].projection)
		}
	case 132:
		//line n1ql.y:1097
		{
			yyVAL.statement = algebra.NewUpdate(yyS[yypt-5].keyspaceRef, yyS[yypt-4].expr, nil, yyS[yypt-3].unset, yyS[yypt-2].expr, yyS[yypt-1].expr, yyS[yypt-0].projection)
		}
	case 133:
		//line n1ql.y:1104
		{
			yyVAL.set = algebra.NewSet(yyS[yypt-0].setTerms)
		}
	case 134:
		//line n1ql.y:1111
		{
			yyVAL.setTerms = algebra.SetTerms{yyS[yypt-0].setTerm}
		}
	case 135:
		//line n1ql.y:1116
		{
			yyVAL.setTerms = append(yyS[yypt-2].setTerms, yyS[yypt-0].setTerm)
		}
	case 136:
		//line n1ql.y:1123
		{
			yyVAL.setTerm = algebra.NewSetTerm(yyS[yypt-3].path, yyS[yypt-1].expr, yyS[yypt-0].updateFor)
		}
	case 137:
		//line n1ql.y:1130
		{
			yyVAL.updateFor = nil
		}
	case 138:
		yyVAL.updateFor = yyS[yypt-0].updateFor
	case 139:
		//line n1ql.y:1139
		{
			yyVAL.updateFor = algebra.NewUpdateFor(yyS[yypt-2].bindings, yyS[yypt-1].expr)
		}
	case 140:
		//line n1ql.y:1146
		{
			yyVAL.bindings = expression.Bindings{yyS[yypt-0].binding}
		}
	case 141:
		//line n1ql.y:1151
		{
			yyVAL.bindings = append(yyS[yypt-2].bindings, yyS[yypt-0].binding)
		}
	case 142:
		//line n1ql.y:1158
		{
			yyVAL.binding = expression.NewBinding(yyS[yypt-2].s, yyS[yypt-0].expr)
		}
	case 143:
		//line n1ql.y:1163
		{
			yyVAL.binding = expression.NewDescendantBinding(yyS[yypt-2].s, yyS[yypt-0].expr)
		}
	case 144:
		yyVAL.s = yyS[yypt-0].s
	case 145:
		//line n1ql.y:1174
		{
			yyVAL.expr = yyS[yypt-0].path
		}
	case 146:
		//line n1ql.y:1181
		{
			yyVAL.expr = nil
		}
	case 147:
		//line n1ql.y:1186
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 148:
		//line n1ql.y:1193
		{
			yyVAL.unset = algebra.NewUnset(yyS[yypt-0].unsetTerms)
		}
	case 149:
		//line n1ql.y:1200
		{
			yyVAL.unsetTerms = algebra.UnsetTerms{yyS[yypt-0].unsetTerm}
		}
	case 150:
		//line n1ql.y:1205
		{
			yyVAL.unsetTerms = append(yyS[yypt-2].unsetTerms, yyS[yypt-0].unsetTerm)
		}
	case 151:
		//line n1ql.y:1212
		{
			yyVAL.unsetTerm = algebra.NewUnsetTerm(yyS[yypt-1].path, yyS[yypt-0].updateFor)
		}
	case 152:
		//line n1ql.y:1226
		{
			source := algebra.NewMergeSourceFrom(yyS[yypt-5].keyspaceTerm, "")
			yyVAL.statement = algebra.NewMerge(yyS[yypt-7].keyspaceRef, source, yyS[yypt-3].expr, yyS[yypt-2].mergeActions, yyS[yypt-1].expr, yyS[yypt-0].projection)
		}
	case 153:
		//line n1ql.y:1232
		{
			source := algebra.NewMergeSourceSelect(yyS[yypt-7].fullselect, yyS[yypt-5].s)
			yyVAL.statement = algebra.NewMerge(yyS[yypt-10].keyspaceRef, source, yyS[yypt-3].expr, yyS[yypt-2].mergeActions, yyS[yypt-1].expr, yyS[yypt-0].projection)
		}
	case 154:
		//line n1ql.y:1240
		{
			yyVAL.mergeActions = algebra.NewMergeActions(nil, nil, nil)
		}
	case 155:
		//line n1ql.y:1245
		{
			yyVAL.mergeActions = algebra.NewMergeActions(yyS[yypt-1].mergeUpdate, yyS[yypt-0].mergeActions.Delete(), yyS[yypt-0].mergeActions.Insert())
		}
	case 156:
		//line n1ql.y:1250
		{
			yyVAL.mergeActions = algebra.NewMergeActions(nil, yyS[yypt-1].mergeDelete, yyS[yypt-0].mergeInsert)
		}
	case 157:
		//line n1ql.y:1255
		{
			yyVAL.mergeActions = algebra.NewMergeActions(nil, nil, yyS[yypt-0].mergeInsert)
		}
	case 158:
		//line n1ql.y:1262
		{
			yyVAL.mergeActions = algebra.NewMergeActions(nil, nil, nil)
		}
	case 159:
		//line n1ql.y:1267
		{
			yyVAL.mergeActions = algebra.NewMergeActions(nil, yyS[yypt-1].mergeDelete, yyS[yypt-0].mergeInsert)
		}
	case 160:
		//line n1ql.y:1272
		{
			yyVAL.mergeActions = algebra.NewMergeActions(nil, nil, yyS[yypt-0].mergeInsert)
		}
	case 161:
		//line n1ql.y:1279
		{
			yyVAL.mergeInsert = nil
		}
	case 162:
		//line n1ql.y:1284
		{
			yyVAL.mergeInsert = yyS[yypt-0].mergeInsert
		}
	case 163:
		//line n1ql.y:1291
		{
			yyVAL.mergeUpdate = algebra.NewMergeUpdate(yyS[yypt-1].set, nil, yyS[yypt-0].expr)
		}
	case 164:
		//line n1ql.y:1296
		{
			yyVAL.mergeUpdate = algebra.NewMergeUpdate(yyS[yypt-2].set, yyS[yypt-1].unset, yyS[yypt-0].expr)
		}
	case 165:
		//line n1ql.y:1301
		{
			yyVAL.mergeUpdate = algebra.NewMergeUpdate(nil, yyS[yypt-1].unset, yyS[yypt-0].expr)
		}
	case 166:
		//line n1ql.y:1308
		{
			yyVAL.mergeDelete = algebra.NewMergeDelete(yyS[yypt-0].expr)
		}
	case 167:
		//line n1ql.y:1315
		{
			yyVAL.mergeInsert = algebra.NewMergeInsert(yyS[yypt-1].expr, yyS[yypt-0].expr)
		}
	case 168:
		//line n1ql.y:1329
		{
			yyVAL.statement = algebra.NewCreatePrimaryIndex(yyS[yypt-0].keyspaceRef)
		}
	case 169:
		//line n1ql.y:1334
		{
			yyVAL.statement = algebra.NewCreateIndex(yyS[yypt-8].s, yyS[yypt-6].keyspaceRef, yyS[yypt-4].exprs, yyS[yypt-2].expr, yyS[yypt-1].expr, yyS[yypt-0].indexType)
		}
	case 170:
		yyVAL.s = yyS[yypt-0].s
	case 171:
		//line n1ql.y:1345
		{
			yyVAL.keyspaceRef = algebra.NewKeyspaceRef("", yyS[yypt-0].s, "")
		}
	case 172:
		//line n1ql.y:1350
		{
			yyVAL.keyspaceRef = algebra.NewKeyspaceRef(yyS[yypt-2].s, yyS[yypt-0].s, "")
		}
	case 173:
		//line n1ql.y:1357
		{
			yyVAL.expr = nil
		}
	case 174:
		//line n1ql.y:1362
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 175:
		//line n1ql.y:1369
		{
			yyVAL.indexType = datastore.VIEW
		}
	case 176:
		//line n1ql.y:1374
		{
			yyVAL.indexType = datastore.VIEW
		}
	case 177:
		//line n1ql.y:1379
		{
			yyVAL.indexType = datastore.LSM
		}
	case 178:
		//line n1ql.y:1393
		{
			yyVAL.statement = algebra.NewDropIndex(yyS[yypt-2].keyspaceRef, yyS[yypt-0].s)
		}
	case 179:
		//line n1ql.y:1406
		{
			yyVAL.statement = algebra.NewAlterIndex(yyS[yypt-3].keyspaceRef, yyS[yypt-1].s, yyS[yypt-0].s)
		}
	case 180:
		//line n1ql.y:1412
		{
			yyVAL.s = ""
		}
	case 181:
		//line n1ql.y:1417
		{
			yyVAL.s = yyS[yypt-0].s
		}
	case 182:
		//line n1ql.y:1431
		{
			yyVAL.path = expression.NewIdentifier(yyS[yypt-0].s)
		}
	case 183:
		//line n1ql.y:1436
		{
			yyVAL.path = expression.NewField(yyS[yypt-2].path, expression.NewFieldName(yyS[yypt-0].s))
		}
	case 184:
		//line n1ql.y:1441
		{
			field := expression.NewField(yyS[yypt-2].path, expression.NewFieldName(yyS[yypt-0].s))
			field.SetCaseInsensitive(true)
			yyVAL.path = field
		}
	case 185:
		//line n1ql.y:1448
		{
			yyVAL.path = expression.NewField(yyS[yypt-4].path, yyS[yypt-1].expr)
		}
	case 186:
		//line n1ql.y:1453
		{
			field := expression.NewField(yyS[yypt-4].path, yyS[yypt-1].expr)
			field.SetCaseInsensitive(true)
			yyVAL.path = field
		}
	case 187:
		//line n1ql.y:1460
		{
			yyVAL.path = expression.NewElement(yyS[yypt-3].path, yyS[yypt-1].expr)
		}
	case 188:
		yyVAL.expr = yyS[yypt-0].expr
	case 189:
		//line n1ql.y:1477
		{
			yyVAL.expr = expression.NewField(yyS[yypt-2].expr, expression.NewFieldName(yyS[yypt-0].s))
		}
	case 190:
		//line n1ql.y:1482
		{
			field := expression.NewField(yyS[yypt-2].expr, expression.NewFieldName(yyS[yypt-0].s))
			field.SetCaseInsensitive(true)
			yyVAL.expr = field
		}
	case 191:
		//line n1ql.y:1489
		{
			yyVAL.expr = expression.NewField(yyS[yypt-4].expr, yyS[yypt-1].expr)
		}
	case 192:
		//line n1ql.y:1494
		{
			field := expression.NewField(yyS[yypt-4].expr, yyS[yypt-1].expr)
			field.SetCaseInsensitive(true)
			yyVAL.expr = field
		}
	case 193:
		//line n1ql.y:1501
		{
			yyVAL.expr = expression.NewElement(yyS[yypt-3].expr, yyS[yypt-1].expr)
		}
	case 194:
		//line n1ql.y:1506
		{
			yyVAL.expr = expression.NewSlice(yyS[yypt-4].expr, yyS[yypt-2].expr)
		}
	case 195:
		//line n1ql.y:1511
		{
			yyVAL.expr = expression.NewSlice(yyS[yypt-5].expr, yyS[yypt-3].expr, yyS[yypt-1].expr)
		}
	case 196:
		//line n1ql.y:1517
		{
			yyVAL.expr = expression.NewAdd(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 197:
		//line n1ql.y:1522
		{
			yyVAL.expr = expression.NewSub(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 198:
		//line n1ql.y:1527
		{
			yyVAL.expr = expression.NewMult(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 199:
		//line n1ql.y:1532
		{
			yyVAL.expr = expression.NewDiv(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 200:
		//line n1ql.y:1537
		{
			yyVAL.expr = expression.NewMod(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 201:
		//line n1ql.y:1543
		{
			yyVAL.expr = expression.NewConcat(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 202:
		//line n1ql.y:1549
		{
			yyVAL.expr = expression.NewAnd(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 203:
		//line n1ql.y:1554
		{
			yyVAL.expr = expression.NewOr(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 204:
		//line n1ql.y:1559
		{
			yyVAL.expr = expression.NewNot(yyS[yypt-0].expr)
		}
	case 205:
		//line n1ql.y:1565
		{
			yyVAL.expr = expression.NewEq(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 206:
		//line n1ql.y:1570
		{
			yyVAL.expr = expression.NewEq(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 207:
		//line n1ql.y:1575
		{
			yyVAL.expr = expression.NewNE(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 208:
		//line n1ql.y:1580
		{
			yyVAL.expr = expression.NewLT(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 209:
		//line n1ql.y:1585
		{
			yyVAL.expr = expression.NewGT(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 210:
		//line n1ql.y:1590
		{
			yyVAL.expr = expression.NewLE(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 211:
		//line n1ql.y:1595
		{
			yyVAL.expr = expression.NewGE(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 212:
		//line n1ql.y:1600
		{
			yyVAL.expr = expression.NewBetween(yyS[yypt-4].expr, yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 213:
		//line n1ql.y:1605
		{
			yyVAL.expr = expression.NewNotBetween(yyS[yypt-5].expr, yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 214:
		//line n1ql.y:1610
		{
			yyVAL.expr = expression.NewLike(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 215:
		//line n1ql.y:1615
		{
			yyVAL.expr = expression.NewNotLike(yyS[yypt-3].expr, yyS[yypt-0].expr)
		}
	case 216:
		//line n1ql.y:1620
		{
			yyVAL.expr = expression.NewIn(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 217:
		//line n1ql.y:1625
		{
			yyVAL.expr = expression.NewNotIn(yyS[yypt-3].expr, yyS[yypt-0].expr)
		}
	case 218:
		//line n1ql.y:1630
		{
			yyVAL.expr = expression.NewWithin(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 219:
		//line n1ql.y:1635
		{
			yyVAL.expr = expression.NewNotWithin(yyS[yypt-3].expr, yyS[yypt-0].expr)
		}
	case 220:
		//line n1ql.y:1640
		{
			yyVAL.expr = expression.NewIsNull(yyS[yypt-2].expr)
		}
	case 221:
		//line n1ql.y:1645
		{
			yyVAL.expr = expression.NewIsNotNull(yyS[yypt-3].expr)
		}
	case 222:
		//line n1ql.y:1650
		{
			yyVAL.expr = expression.NewIsMissing(yyS[yypt-2].expr)
		}
	case 223:
		//line n1ql.y:1655
		{
			yyVAL.expr = expression.NewIsNotMissing(yyS[yypt-3].expr)
		}
	case 224:
		//line n1ql.y:1660
		{
			yyVAL.expr = expression.NewIsValued(yyS[yypt-2].expr)
		}
	case 225:
		//line n1ql.y:1665
		{
			yyVAL.expr = expression.NewIsNotValued(yyS[yypt-3].expr)
		}
	case 226:
		//line n1ql.y:1670
		{
			yyVAL.expr = expression.NewExists(yyS[yypt-0].expr)
		}
	case 227:
		yyVAL.expr = yyS[yypt-0].expr
	case 228:
		yyVAL.expr = yyS[yypt-0].expr
	case 229:
		//line n1ql.y:1684
		{
			yyVAL.expr = expression.NewIdentifier(yyS[yypt-0].s)
		}
	case 230:
		yyVAL.expr = yyS[yypt-0].expr
	case 231:
		yyVAL.expr = yyS[yypt-0].expr
	case 232:
		//line n1ql.y:1696
		{
			yyVAL.expr = expression.NewNeg(yyS[yypt-0].expr)
		}
	case 233:
		yyVAL.expr = yyS[yypt-0].expr
	case 234:
		yyVAL.expr = yyS[yypt-0].expr
	case 235:
		yyVAL.expr = yyS[yypt-0].expr
	case 236:
		yyVAL.expr = yyS[yypt-0].expr
	case 237:
		//line n1ql.y:1715
		{
			yyVAL.expr = expression.NewField(yyS[yypt-2].expr, expression.NewFieldName(yyS[yypt-0].s))
		}
	case 238:
		//line n1ql.y:1720
		{
			field := expression.NewField(yyS[yypt-2].expr, expression.NewFieldName(yyS[yypt-0].s))
			field.SetCaseInsensitive(true)
			yyVAL.expr = field
		}
	case 239:
		//line n1ql.y:1727
		{
			yyVAL.expr = expression.NewField(yyS[yypt-4].expr, yyS[yypt-1].expr)
		}
	case 240:
		//line n1ql.y:1732
		{
			field := expression.NewField(yyS[yypt-4].expr, yyS[yypt-1].expr)
			field.SetCaseInsensitive(true)
			yyVAL.expr = field
		}
	case 241:
		//line n1ql.y:1739
		{
			yyVAL.expr = expression.NewElement(yyS[yypt-3].expr, yyS[yypt-1].expr)
		}
	case 242:
		//line n1ql.y:1744
		{
			yyVAL.expr = expression.NewSlice(yyS[yypt-4].expr, yyS[yypt-2].expr)
		}
	case 243:
		//line n1ql.y:1749
		{
			yyVAL.expr = expression.NewSlice(yyS[yypt-5].expr, yyS[yypt-3].expr, yyS[yypt-1].expr)
		}
	case 244:
		//line n1ql.y:1755
		{
			yyVAL.expr = expression.NewAdd(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 245:
		//line n1ql.y:1760
		{
			yyVAL.expr = expression.NewSub(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 246:
		//line n1ql.y:1765
		{
			yyVAL.expr = expression.NewMult(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 247:
		//line n1ql.y:1770
		{
			yyVAL.expr = expression.NewDiv(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 248:
		//line n1ql.y:1775
		{
			yyVAL.expr = expression.NewMod(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 249:
		//line n1ql.y:1781
		{
			yyVAL.expr = expression.NewConcat(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 250:
		//line n1ql.y:1795
		{
			yyVAL.expr = expression.NULL_EXPR
		}
	case 251:
		//line n1ql.y:1800
		{
			yyVAL.expr = expression.FALSE_EXPR
		}
	case 252:
		//line n1ql.y:1805
		{
			yyVAL.expr = expression.TRUE_EXPR
		}
	case 253:
		//line n1ql.y:1810
		{
			yyVAL.expr = expression.NewConstant(value.NewValue(yyS[yypt-0].f))
		}
	case 254:
		//line n1ql.y:1815
		{
			yyVAL.expr = expression.NewConstant(value.NewValue(yyS[yypt-0].n))
		}
	case 255:
		//line n1ql.y:1820
		{
			yyVAL.expr = expression.NewConstant(value.NewValue(yyS[yypt-0].s))
		}
	case 256:
		yyVAL.expr = yyS[yypt-0].expr
	case 257:
		yyVAL.expr = yyS[yypt-0].expr
	case 258:
		//line n1ql.y:1840
		{
			yyVAL.expr = expression.NewObjectConstruct(yyS[yypt-1].bindings)
		}
	case 259:
		//line n1ql.y:1847
		{
			yyVAL.bindings = nil
		}
	case 260:
		yyVAL.bindings = yyS[yypt-0].bindings
	case 261:
		//line n1ql.y:1856
		{
			yyVAL.bindings = expression.Bindings{yyS[yypt-0].binding}
		}
	case 262:
		//line n1ql.y:1861
		{
			yyVAL.bindings = append(yyS[yypt-2].bindings, yyS[yypt-0].binding)
		}
	case 263:
		//line n1ql.y:1868
		{
			yyVAL.binding = expression.NewBinding(yyS[yypt-2].s, yyS[yypt-0].expr)
		}
	case 264:
		//line n1ql.y:1875
		{
			yyVAL.expr = expression.NewArrayConstruct(yyS[yypt-1].exprs...)
		}
	case 265:
		//line n1ql.y:1882
		{
			yyVAL.exprs = nil
		}
	case 266:
		yyVAL.exprs = yyS[yypt-0].exprs
	case 267:
		//line n1ql.y:1898
		{
			yyVAL.expr = algebra.NewNamedParameter(yyS[yypt-0].s)
		}
	case 268:
		//line n1ql.y:1903
		{
			yyVAL.expr = algebra.NewPositionalParameter(yyS[yypt-0].n)
		}
	case 269:
		//line n1ql.y:1917
		{
			yyVAL.expr = yyS[yypt-1].expr
		}
	case 270:
		yyVAL.expr = yyS[yypt-0].expr
	case 271:
		yyVAL.expr = yyS[yypt-0].expr
	case 272:
		//line n1ql.y:1930
		{
			yyVAL.expr = expression.NewSimpleCase(yyS[yypt-2].expr, yyS[yypt-1].whenTerms, yyS[yypt-0].expr)
		}
	case 273:
		//line n1ql.y:1937
		{
			yyVAL.whenTerms = expression.WhenTerms{&expression.WhenTerm{yyS[yypt-2].expr, yyS[yypt-0].expr}}
		}
	case 274:
		//line n1ql.y:1942
		{
			yyVAL.whenTerms = append(yyS[yypt-4].whenTerms, &expression.WhenTerm{yyS[yypt-2].expr, yyS[yypt-0].expr})
		}
	case 275:
		//line n1ql.y:1950
		{
			yyVAL.expr = expression.NewSearchedCase(yyS[yypt-1].whenTerms, yyS[yypt-0].expr)
		}
	case 276:
		//line n1ql.y:1957
		{
			yyVAL.expr = nil
		}
	case 277:
		//line n1ql.y:1962
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 278:
		//line n1ql.y:1976
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
	case 279:
		//line n1ql.y:1995
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
	case 280:
		//line n1ql.y:2010
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
	case 281:
		yyVAL.s = yyS[yypt-0].s
	case 282:
		yyVAL.expr = yyS[yypt-0].expr
	case 283:
		yyVAL.expr = yyS[yypt-0].expr
	case 284:
		//line n1ql.y:2044
		{
			yyVAL.expr = expression.NewAny(yyS[yypt-2].bindings, yyS[yypt-1].expr)
		}
	case 285:
		//line n1ql.y:2049
		{
			yyVAL.expr = expression.NewAny(yyS[yypt-2].bindings, yyS[yypt-1].expr)
		}
	case 286:
		//line n1ql.y:2054
		{
			yyVAL.expr = expression.NewEvery(yyS[yypt-2].bindings, yyS[yypt-1].expr)
		}
	case 287:
		//line n1ql.y:2061
		{
			yyVAL.bindings = expression.Bindings{yyS[yypt-0].binding}
		}
	case 288:
		//line n1ql.y:2066
		{
			yyVAL.bindings = append(yyS[yypt-2].bindings, yyS[yypt-0].binding)
		}
	case 289:
		//line n1ql.y:2073
		{
			yyVAL.binding = expression.NewBinding(yyS[yypt-2].s, yyS[yypt-0].expr)
		}
	case 290:
		//line n1ql.y:2078
		{
			yyVAL.binding = expression.NewDescendantBinding(yyS[yypt-2].s, yyS[yypt-0].expr)
		}
	case 291:
		//line n1ql.y:2085
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 292:
		//line n1ql.y:2092
		{
			yyVAL.expr = expression.NewArray(yyS[yypt-4].expr, yyS[yypt-2].bindings, yyS[yypt-1].expr)
		}
	case 293:
		//line n1ql.y:2097
		{
			yyVAL.expr = expression.NewFirst(yyS[yypt-4].expr, yyS[yypt-2].bindings, yyS[yypt-1].expr)
		}
	case 294:
		//line n1ql.y:2111
		{
			yyVAL.expr = yyS[yypt-1].expr
		}
	case 295:
		yyVAL.expr = yyS[yypt-0].expr
	case 296:
		//line n1ql.y:2120
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
