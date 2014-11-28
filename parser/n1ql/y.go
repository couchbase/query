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
	-1, 21,
	163, 288,
	-2, 234,
	-1, 110,
	171, 63,
	-2, 64,
	-1, 147,
	51, 72,
	69, 72,
	88, 72,
	138, 72,
	-2, 50,
	-1, 174,
	173, 0,
	174, 0,
	175, 0,
	-2, 210,
	-1, 175,
	173, 0,
	174, 0,
	175, 0,
	-2, 211,
	-1, 176,
	173, 0,
	174, 0,
	175, 0,
	-2, 212,
	-1, 177,
	176, 0,
	177, 0,
	178, 0,
	179, 0,
	-2, 213,
	-1, 178,
	176, 0,
	177, 0,
	178, 0,
	179, 0,
	-2, 214,
	-1, 179,
	176, 0,
	177, 0,
	178, 0,
	179, 0,
	-2, 215,
	-1, 180,
	176, 0,
	177, 0,
	178, 0,
	179, 0,
	-2, 216,
	-1, 187,
	77, 0,
	-2, 219,
	-1, 188,
	59, 0,
	153, 0,
	-2, 221,
	-1, 189,
	59, 0,
	153, 0,
	-2, 223,
	-1, 283,
	77, 0,
	-2, 220,
	-1, 284,
	59, 0,
	153, 0,
	-2, 222,
	-1, 285,
	59, 0,
	153, 0,
	-2, 224,
}

const yyNprod = 304
const yyPrivate = 57344

var yyTokenNames []string
var yyStates []string

const yyLast = 2774

var yyAct = []int{

	161, 3, 583, 572, 442, 581, 573, 304, 305, 194,
	94, 95, 463, 531, 503, 308, 300, 522, 211, 496,
	537, 212, 398, 228, 259, 456, 97, 136, 415, 207,
	342, 400, 297, 213, 153, 397, 156, 424, 132, 148,
	12, 108, 253, 386, 157, 8, 339, 231, 135, 252,
	129, 116, 134, 275, 120, 223, 260, 181, 68, 326,
	133, 458, 54, 324, 140, 141, 346, 343, 278, 279,
	280, 474, 274, 165, 166, 167, 168, 169, 170, 171,
	172, 173, 174, 175, 176, 177, 178, 179, 180, 72,
	432, 187, 188, 189, 121, 93, 454, 432, 275, 72,
	416, 525, 162, 163, 75, 76, 77, 526, 71, 431,
	164, 150, 74, 133, 138, 139, 431, 274, 71, 225,
	379, 519, 416, 473, 325, 363, 262, 261, 237, 210,
	242, 263, 345, 455, 182, 453, 381, 239, 236, 369,
	370, 235, 238, 241, 162, 163, 499, 371, 109, 249,
	316, 314, 164, 151, 226, 465, 112, 267, 476, 477,
	214, 239, 197, 199, 201, 270, 357, 233, 233, 427,
	151, 277, 137, 229, 234, 311, 432, 215, 254, 110,
	269, 241, 130, 251, 110, 283, 284, 285, 405, 264,
	266, 265, 307, 556, 72, 431, 224, 292, 277, 582,
	110, 110, 501, 251, 298, 577, 523, 78, 73, 75,
	76, 77, 520, 71, 142, 464, 313, 209, 149, 315,
	243, 69, 302, 318, 192, 319, 306, 191, 190, 182,
	505, 287, 202, 562, 240, 286, 303, 596, 328, 595,
	329, 282, 307, 332, 333, 334, 591, 563, 312, 552,
	69, 70, 344, 275, 444, 273, 467, 545, 309, 117,
	354, 293, 347, 294, 532, 295, 361, 276, 278, 279,
	280, 500, 274, 367, 232, 232, 372, 317, 355, 543,
	275, 69, 460, 356, 193, 323, 362, 327, 131, 70,
	331, 288, 380, 281, 276, 278, 279, 280, 183, 274,
	337, 338, 389, 125, 222, 200, 521, 198, 360, 215,
	392, 394, 395, 393, 244, 388, 123, 322, 70, 182,
	321, 408, 182, 182, 182, 182, 182, 182, 291, 403,
	401, 368, 589, 277, 373, 374, 375, 376, 377, 378,
	387, 561, 593, 391, 185, 124, 390, 422, 412, 70,
	414, 429, 592, 404, 69, 150, 69, 553, 122, 310,
	255, 586, 184, 233, 233, 233, 196, 417, 587, 409,
	410, 411, 437, 245, 246, 101, 221, 254, 107, 254,
	435, 352, 298, 433, 434, 430, 144, 428, 421, 446,
	426, 426, 445, 420, 423, 447, 448, 100, 418, 348,
	450, 217, 449, 459, 451, 452, 541, 558, 462, 402,
	358, 359, 301, 542, 441, 275, 341, 469, 349, 277,
	133, 111, 70, 67, 70, 182, 105, 103, 281, 276,
	278, 279, 280, 478, 274, 257, 343, 440, 186, 104,
	484, 599, 598, 574, 230, 258, 461, 227, 475, 126,
	472, 530, 479, 480, 488, 493, 490, 491, 471, 69,
	489, 106, 149, 535, 470, 468, 504, 99, 351, 594,
	232, 232, 232, 336, 413, 335, 330, 498, 401, 486,
	220, 487, 557, 204, 205, 206, 497, 494, 515, 492,
	216, 508, 146, 419, 516, 203, 2, 425, 425, 353,
	350, 275, 507, 1, 502, 555, 466, 544, 96, 512,
	513, 569, 74, 576, 281, 276, 278, 279, 280, 517,
	274, 524, 457, 399, 518, 396, 495, 504, 443, 527,
	485, 547, 540, 299, 533, 534, 37, 36, 35, 536,
	18, 546, 539, 538, 538, 551, 17, 497, 548, 16,
	15, 14, 549, 550, 13, 79, 7, 504, 567, 568,
	554, 88, 6, 559, 560, 5, 4, 382, 565, 570,
	571, 566, 564, 383, 575, 584, 289, 578, 580, 579,
	585, 290, 195, 296, 98, 102, 588, 79, 152, 529,
	214, 590, 528, 88, 72, 506, 340, 250, 597, 584,
	584, 601, 602, 600, 143, 208, 256, 91, 73, 75,
	76, 77, 145, 71, 147, 65, 93, 66, 29, 119,
	28, 79, 509, 510, 49, 90, 24, 88, 52, 51,
	27, 115, 114, 74, 113, 57, 26, 89, 127, 91,
	128, 23, 46, 80, 45, 20, 19, 0, 93, 0,
	0, 0, 0, 0, 0, 0, 55, 90, 0, 0,
	0, 0, 32, 0, 0, 74, 0, 0, 56, 89,
	0, 0, 0, 91, 0, 80, 0, 0, 11, 0,
	0, 0, 93, 69, 0, 0, 0, 0, 0, 0,
	0, 90, 0, 0, 0, 30, 0, 0, 0, 74,
	0, 92, 0, 89, 0, 0, 0, 0, 0, 80,
	0, 0, 0, 0, 34, 72, 481, 482, 0, 0,
	0, 81, 82, 83, 84, 85, 86, 87, 78, 73,
	75, 76, 77, 92, 71, 0, 0, 0, 0, 215,
	0, 0, 0, 0, 0, 0, 0, 72, 0, 0,
	0, 70, 0, 81, 82, 83, 84, 85, 86, 87,
	78, 73, 75, 76, 77, 0, 71, 92, 0, 0,
	33, 31, 79, 0, 0, 0, 384, 0, 88, 0,
	0, 72, 438, 0, 0, 439, 0, 81, 82, 83,
	84, 85, 86, 87, 78, 73, 75, 76, 77, 385,
	71, 0, 0, 0, 79, 0, 0, 0, 0, 0,
	88, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 91, 0, 0, 0, 0, 0,
	0, 0, 0, 93, 0, 0, 0, 0, 79, 0,
	0, 0, 90, 0, 88, 0, 0, 0, 0, 0,
	74, 0, 0, 0, 89, 0, 91, 0, 0, 0,
	80, 0, 0, 0, 0, 93, 0, 0, 0, 0,
	0, 0, 0, 0, 90, 0, 0, 0, 0, 0,
	0, 0, 74, 0, 0, 0, 89, 0, 0, 0,
	91, 0, 80, 0, 0, 0, 0, 0, 0, 93,
	0, 0, 0, 0, 0, 0, 0, 0, 90, 0,
	0, 0, 0, 0, 88, 0, 74, 0, 92, 0,
	89, 0, 0, 0, 0, 0, 80, 0, 0, 0,
	0, 0, 72, 0, 0, 0, 0, 0, 81, 82,
	83, 84, 85, 86, 87, 78, 73, 75, 76, 77,
	92, 71, 0, 0, 0, 0, 0, 0, 0, 0,
	91, 0, 0, 0, 72, 364, 365, 0, 0, 93,
	81, 82, 83, 84, 85, 86, 87, 78, 73, 75,
	76, 77, 79, 71, 92, 214, 74, 0, 88, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 72, 271,
	0, 0, 272, 0, 81, 82, 83, 84, 85, 86,
	87, 78, 73, 75, 76, 77, 0, 71, 0, 79,
	0, 0, 0, 0, 0, 88, 0, 0, 0, 0,
	0, 0, 0, 0, 91, 0, 0, 0, 0, 0,
	0, 0, 0, 93, 0, 0, 0, 0, 0, 0,
	0, 79, 90, 0, 92, 0, 0, 88, 0, 0,
	74, 0, 0, 0, 89, 0, 0, 0, 72, 0,
	80, 91, 0, 0, 0, 0, 0, 0, 0, 0,
	93, 78, 73, 75, 76, 77, 0, 71, 0, 90,
	0, 0, 0, 0, 0, 0, 458, 74, 0, 0,
	0, 89, 0, 91, 0, 0, 0, 80, 0, 0,
	0, 0, 93, 0, 0, 0, 0, 0, 0, 0,
	0, 90, 0, 0, 0, 0, 0, 0, 92, 74,
	0, 0, 0, 89, 215, 0, 0, 0, 0, 80,
	0, 0, 72, 0, 0, 0, 0, 0, 81, 82,
	83, 84, 85, 86, 87, 78, 73, 75, 76, 77,
	0, 268, 251, 0, 0, 92, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 79, 0, 72,
	0, 0, 0, 88, 0, 81, 82, 83, 84, 85,
	86, 87, 78, 73, 75, 76, 77, 92, 71, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 79,
	0, 72, 0, 0, 0, 88, 0, 81, 82, 83,
	84, 85, 86, 87, 78, 73, 75, 76, 77, 91,
	71, 0, 0, 0, 0, 0, 0, 0, 93, 0,
	0, 0, 0, 79, 0, 0, 0, 90, 0, 88,
	0, 0, 0, 0, 0, 74, 0, 0, 0, 89,
	0, 91, 0, 0, 0, 80, 0, 0, 0, 0,
	93, 0, 0, 0, 0, 0, 0, 0, 0, 90,
	0, 0, 0, 0, 0, 0, 0, 74, 0, 0,
	0, 89, 0, 0, 0, 91, 0, 80, 0, 0,
	0, 0, 0, 0, 93, 0, 0, 0, 0, 0,
	0, 0, 0, 90, 0, 0, 0, 0, 0, 0,
	0, 74, 0, 92, 0, 89, 0, 0, 0, 0,
	0, 80, 0, 0, 0, 0, 0, 72, 514, 0,
	0, 0, 0, 81, 82, 83, 84, 85, 86, 87,
	78, 73, 75, 76, 77, 92, 71, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 72,
	511, 0, 0, 0, 0, 81, 82, 83, 84, 85,
	86, 87, 78, 73, 75, 76, 77, 79, 71, 92,
	0, 0, 0, 88, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 72, 436, 0, 0, 0, 0, 81,
	82, 83, 84, 85, 86, 87, 78, 73, 75, 76,
	77, 0, 71, 0, 79, 0, 0, 0, 0, 0,
	88, 0, 0, 0, 0, 0, 0, 0, 0, 91,
	0, 0, 0, 0, 0, 0, 0, 0, 93, 0,
	0, 0, 0, 0, 0, 0, 79, 90, 60, 63,
	0, 0, 88, 0, 0, 74, 0, 0, 50, 89,
	0, 0, 0, 0, 0, 80, 91, 0, 0, 0,
	0, 0, 0, 0, 0, 93, 0, 0, 0, 0,
	0, 0, 0, 62, 90, 0, 0, 10, 0, 40,
	64, 0, 74, 0, 0, 0, 89, 0, 91, 407,
	0, 0, 80, 0, 0, 0, 0, 93, 0, 0,
	0, 0, 0, 0, 0, 0, 90, 0, 0, 0,
	0, 0, 0, 92, 74, 25, 39, 0, 89, 9,
	38, 0, 0, 0, 80, 0, 0, 72, 0, 0,
	0, 0, 0, 81, 82, 83, 84, 85, 86, 87,
	78, 73, 75, 76, 77, 0, 71, 0, 0, 0,
	92, 0, 22, 0, 0, 61, 0, 0, 320, 0,
	0, 406, 0, 41, 72, 0, 0, 0, 0, 0,
	81, 82, 83, 84, 85, 86, 87, 78, 73, 75,
	76, 77, 92, 71, 79, 0, 43, 42, 44, 21,
	88, 47, 48, 53, 0, 58, 72, 59, 483, 0,
	0, 0, 81, 82, 83, 84, 85, 86, 87, 78,
	73, 75, 76, 77, 0, 71, 79, 0, 0, 0,
	0, 0, 88, 0, 0, 0, 0, 0, 0, 248,
	0, 0, 0, 0, 0, 0, 91, 0, 0, 0,
	0, 0, 0, 0, 0, 93, 0, 0, 0, 0,
	0, 0, 0, 0, 90, 0, 0, 79, 0, 0,
	0, 247, 74, 88, 0, 0, 89, 0, 91, 0,
	0, 0, 80, 0, 0, 0, 0, 93, 0, 0,
	0, 0, 0, 0, 0, 0, 90, 0, 0, 0,
	0, 0, 0, 0, 74, 0, 0, 0, 89, 0,
	0, 0, 0, 0, 80, 0, 0, 0, 0, 91,
	0, 0, 0, 0, 0, 0, 0, 0, 93, 0,
	0, 0, 0, 0, 0, 0, 0, 90, 0, 0,
	92, 0, 0, 0, 0, 74, 0, 0, 0, 89,
	0, 0, 0, 0, 72, 80, 0, 0, 0, 0,
	81, 82, 83, 84, 85, 86, 87, 78, 73, 75,
	76, 77, 92, 71, 0, 155, 0, 0, 0, 60,
	63, 0, 0, 0, 0, 0, 72, 0, 0, 50,
	0, 0, 81, 82, 83, 84, 85, 86, 87, 78,
	73, 75, 76, 77, 0, 71, 79, 154, 0, 118,
	0, 159, 88, 92, 62, 0, 0, 0, 10, 0,
	40, 64, 0, 0, 0, 0, 0, 72, 0, 0,
	0, 0, 0, 81, 82, 83, 84, 85, 86, 87,
	78, 73, 75, 76, 77, 0, 71, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 25, 39, 91, 0,
	9, 38, 0, 0, 0, 0, 0, 93, 0, 0,
	0, 0, 0, 0, 0, 0, 90, 0, 0, 0,
	158, 0, 0, 0, 74, 0, 0, 0, 89, 0,
	0, 0, 0, 22, 80, 0, 61, 0, 0, 0,
	0, 0, 0, 0, 41, 0, 0, 0, 0, 60,
	63, 0, 0, 0, 0, 0, 0, 0, 0, 50,
	0, 0, 0, 0, 0, 0, 0, 43, 42, 44,
	21, 0, 47, 48, 53, 0, 58, 0, 59, 0,
	0, 159, 0, 79, 62, 0, 0, 0, 10, 88,
	40, 64, 92, 160, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 72, 0, 0, 0,
	0, 0, 81, 82, 83, 84, 85, 86, 87, 78,
	73, 75, 76, 77, 0, 71, 25, 39, 0, 88,
	9, 38, 0, 0, 0, 91, 0, 0, 0, 0,
	0, 0, 0, 0, 93, 0, 0, 0, 0, 0,
	158, 0, 0, 90, 0, 0, 0, 0, 0, 0,
	0, 74, 0, 22, 0, 89, 61, 0, 0, 88,
	0, 0, 0, 0, 41, 91, 0, 0, 0, 0,
	0, 0, 0, 0, 93, 0, 0, 0, 0, 0,
	0, 0, 0, 90, 0, 0, 0, 43, 42, 44,
	21, 74, 47, 48, 53, 89, 58, 0, 59, 0,
	0, 0, 0, 0, 0, 91, 0, 0, 0, 0,
	0, 0, 0, 160, 93, 0, 0, 0, 0, 92,
	0, 0, 0, 90, 0, 60, 63, 0, 0, 0,
	0, 74, 0, 72, 0, 50, 0, 0, 0, 81,
	82, 83, 84, 85, 86, 87, 78, 73, 75, 76,
	77, 0, 71, 218, 0, 0, 0, 0, 0, 92,
	62, 0, 0, 0, 10, 0, 40, 64, 0, 0,
	0, 0, 0, 72, 0, 0, 0, 0, 0, 81,
	82, 83, 84, 85, 86, 87, 78, 73, 75, 76,
	77, 0, 71, 0, 0, 0, 0, 0, 0, 92,
	0, 0, 25, 39, 0, 0, 9, 38, 0, 0,
	60, 63, 0, 72, 0, 0, 0, 0, 0, 0,
	50, 0, 84, 85, 86, 87, 78, 73, 75, 76,
	77, 88, 71, 0, 0, 0, 0, 0, 0, 22,
	0, 0, 61, 0, 0, 62, 0, 0, 0, 10,
	41, 40, 64, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 43, 42, 44, 21, 91, 47, 48,
	53, 0, 58, 0, 59, 0, 93, 25, 39, 0,
	0, 9, 38, 0, 0, 90, 0, 60, 63, 219,
	0, 0, 0, 74, 0, 0, 0, 50, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 22, 0, 0, 61, 0, 0,
	0, 0, 62, 0, 0, 41, 10, 0, 40, 64,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 57,
	0, 0, 60, 63, 0, 0, 0, 0, 43, 42,
	44, 21, 50, 47, 48, 53, 0, 58, 0, 59,
	55, 92, 0, 0, 25, 39, 32, 0, 9, 38,
	0, 0, 56, 0, 160, 72, 0, 62, 0, 0,
	0, 10, 11, 40, 64, 0, 0, 69, 78, 73,
	75, 76, 77, 0, 71, 0, 0, 0, 0, 30,
	0, 22, 0, 0, 61, 0, 0, 0, 0, 0,
	0, 0, 41, 0, 0, 0, 0, 0, 34, 25,
	39, 0, 0, 9, 38, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 43, 42, 44, 21, 0,
	47, 48, 53, 0, 58, 0, 59, 366, 0, 0,
	0, 0, 0, 0, 0, 70, 22, 0, 0, 61,
	0, 0, 0, 60, 63, 0, 0, 41, 0, 0,
	0, 0, 0, 50, 33, 31, 0, 0, 60, 63,
	0, 0, 0, 0, 0, 0, 0, 0, 50, 0,
	43, 42, 44, 21, 0, 47, 48, 53, 62, 58,
	0, 59, 10, 0, 40, 64, 0, 0, 69, 0,
	0, 0, 0, 62, 0, 0, 0, 10, 0, 40,
	64, 0, 0, 0, 0, 0, 0, 60, 63, 0,
	0, 0, 0, 0, 0, 0, 0, 50, 0, 0,
	25, 39, 0, 0, 9, 38, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 25, 39, 0, 0, 9,
	38, 0, 62, 0, 0, 0, 10, 0, 40, 64,
	0, 0, 0, 0, 0, 0, 70, 22, 0, 0,
	61, 0, 0, 0, 0, 0, 0, 0, 41, 0,
	0, 0, 22, 0, 0, 61, 0, 0, 0, 0,
	0, 0, 0, 41, 25, 39, 0, 0, 9, 38,
	0, 43, 42, 44, 21, 0, 47, 48, 53, 118,
	58, 0, 59, 0, 60, 63, 43, 42, 44, 21,
	0, 47, 48, 53, 50, 58, 0, 59, 0, 0,
	0, 22, 0, 0, 61, 0, 0, 0, 0, 0,
	0, 0, 41, 0, 0, 0, 0, 0, 0, 62,
	0, 0, 0, 0, 0, 40, 64, 0, 0, 0,
	0, 0, 0, 0, 0, 43, 42, 44, 21, 0,
	47, 48, 53, 0, 58, 0, 59, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 25, 39, 0, 0, 0, 38, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 22, 0,
	0, 61, 0, 0, 0, 0, 0, 0, 0, 41,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 43, 42, 44, 21, 0, 47, 48, 53,
	0, 58, 0, 59,
}
var yyPact = []int{

	2324, -1000, -1000, 1809, -1000, -1000, -1000, -1000, -1000, 2509,
	2509, 630, -1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000,
	-1000, -1000, -1000, -1000, -1000, 2509, -1000, -1000, -1000, 331,
	372, 359, 408, 20, 354, -1000, -1000, -1000, -1000, -1000,
	-1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000, -7,
	2460, -1000, -1000, 2445, -1000, 254, 241, 387, 24, 2509,
	13, 13, 13, 2509, 2509, -1000, -1000, 311, 406, 42,
	1781, -15, 2509, 2509, 2509, 2509, 2509, 2509, 2509, 2509,
	2509, 2509, 2509, 2509, 2509, 2509, 2509, 2509, 2606, 285,
	2509, 2509, 2509, 138, 1986, 27, -1000, -68, 288, 303,
	301, 228, -1000, 479, 20, 20, 20, 75, -42, 150,
	-1000, 20, 2097, 438, -1000, -1000, 1670, 155, 2509, -10,
	1809, -1000, 385, 14, 382, 20, 20, -25, -32, -1000,
	-43, -26, -33, 1809, 11, -1000, 161, -1000, 11, 11,
	1629, 1597, 33, -1000, 18, 311, -1000, 371, -1000, -130,
	-44, -45, -1000, -39, 1911, 2182, 2509, -1000, -1000, -1000,
	-1000, 975, -1000, -1000, 2509, 831, -78, -78, -68, -68,
	-68, 427, 1986, 1946, 2026, 2026, 2026, 2198, 2198, 2198,
	2198, 248, -1000, 2606, 2509, 2509, 2509, 901, 27, 27,
	-1000, 145, -1000, -1000, 236, -1000, 2509, -1000, 197, -1000,
	197, -1000, 197, 2509, 342, 342, 75, 103, -1000, 154,
	16, -1000, -1000, -1000, 18, -1000, 72, -13, 2509, -14,
	-1000, 155, 2509, -1000, 2509, 1449, -1000, 227, 224, -1000,
	192, -123, -1000, -47, -127, -1000, 24, 2509, -1000, 2509,
	434, 13, 2509, 2509, 2509, 433, 431, 13, 13, 360,
	-1000, 2509, -38, -1000, -107, 33, 330, -1000, 163, 150,
	7, 16, 16, 2182, -39, 2509, -39, 580, -57, -1000,
	797, -1000, 2269, 2606, -20, 2509, 2606, 2606, 2606, 2606,
	2606, 2606, 113, 901, 27, 27, -1000, -1000, -1000, -1000,
	-1000, 2509, 1809, -1000, -1000, -1000, -34, -1000, 765, 168,
	-1000, 2509, 168, 33, 53, 33, 7, 7, 338, -1000,
	150, -1000, -1000, 25, -1000, 1417, -1000, -1000, 1380, 1809,
	2509, 20, 20, 20, 14, 16, 14, -1000, 1809, 1809,
	-1000, -1000, 1809, 1809, 1809, -1000, -1000, -27, -27, 130,
	-1000, 477, -1000, 18, 1809, 18, 2509, 360, 41, 41,
	2509, -1000, -1000, -1000, -1000, 75, -70, -1000, -130, -130,
	-1000, 580, -1000, -1000, -1000, -1000, -1000, 1236, 334, -1000,
	-1000, 2509, 614, -114, -114, -69, -69, -69, 86, 2606,
	1809, 2509, -1000, -1000, -1000, -1000, 140, 140, 2509, 1809,
	140, 140, 288, 33, 288, 288, -35, -1000, -77, -37,
	-1000, 9, 2509, -1000, 189, 197, -1000, 2509, 1809, 71,
	-8, -1000, -1000, -1000, 144, 423, 2509, 422, -1000, 2509,
	-38, -1000, 1809, -1000, -1000, -130, -48, -100, -1000, 580,
	-1000, -1, 2509, 150, 150, -1000, -1000, 548, -1000, 1450,
	334, -1000, -1000, -1000, 1911, -1000, 1809, -1000, -1000, 140,
	288, 140, 140, 7, 2509, 7, -1000, -1000, 13, 1809,
	342, -18, 1809, -1000, 123, 2509, -1000, 100, -1000, 1809,
	-1000, -9, 150, 16, 16, -1000, -1000, -1000, 1202, 75,
	75, -1000, -1000, -1000, 1170, -1000, -39, 2509, -1000, 140,
	-1000, -1000, -1000, 1044, -1000, -49, -1000, 153, 57, 150,
	-1000, -1000, -63, -1000, 1809, 14, 394, -1000, 171, -130,
	-130, -1000, -1000, -1000, -1000, 1809, -1000, -1000, 421, 13,
	7, 7, 288, 324, 186, 158, 2509, -1000, -1000, -1000,
	2509, -1000, 154, 150, 150, -1000, -1000, -1000, -70, -1000,
	140, 120, 275, 342, 43, 466, -1000, 1809, 336, 171,
	171, -1000, 201, 118, 57, 71, 2509, 2509, 2509, -1000,
	-1000, 103, 33, 378, 288, -1000, -1000, 1809, 1809, 56,
	53, 33, 50, -1000, 2509, 140, -1000, 279, -1000, 33,
	-1000, -1000, 243, -1000, 1012, -1000, 117, 270, -1000, 260,
	-1000, 437, 110, 108, 33, 377, 376, 50, 2509, 2509,
	-1000, -1000, -1000,
}
var yyPgo = []int{

	0, 646, 645, 644, 642, 641, 50, 640, 638, 0,
	45, 57, 38, 288, 42, 49, 33, 21, 18, 27,
	636, 634, 632, 631, 55, 259, 630, 629, 628, 48,
	52, 234, 28, 626, 624, 620, 619, 40, 618, 62,
	617, 615, 614, 423, 612, 39, 37, 606, 22, 24,
	41, 148, 605, 29, 13, 214, 604, 6, 597, 46,
	596, 595, 30, 592, 589, 44, 34, 588, 58, 585,
	584, 32, 583, 582, 9, 581, 576, 573, 567, 496,
	566, 565, 562, 556, 554, 551, 550, 549, 546, 540,
	538, 537, 536, 378, 43, 16, 533, 530, 528, 4,
	19, 526, 20, 7, 35, 525, 8, 31, 523, 522,
	25, 17, 513, 511, 3, 2, 5, 23, 47, 507,
	12, 506, 14, 505, 504, 503, 36, 500, 15, 499,
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
	48, 48, 48, 9, 9, 9, 9, 9, 9, 9,
	9, 9, 9, 9, 9, 9, 9, 9, 9, 9,
	9, 9, 9, 9, 9, 9, 9, 9, 9, 9,
	9, 9, 9, 9, 9, 9, 9, 9, 9, 9,
	9, 9, 10, 10, 10, 10, 10, 10, 10, 10,
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
	3, 3, 4, 1, 3, 3, 5, 5, 4, 5,
	6, 3, 3, 3, 3, 3, 3, 3, 3, 2,
	3, 3, 3, 3, 3, 3, 3, 5, 6, 3,
	4, 3, 4, 3, 4, 3, 4, 3, 4, 3,
	4, 2, 1, 1, 1, 1, 1, 1, 2, 1,
	1, 1, 1, 3, 3, 5, 5, 4, 5, 6,
	3, 3, 3, 3, 3, 3, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 3, 0, 1, 1, 3,
	3, 3, 0, 1, 1, 1, 3, 1, 1, 3,
	4, 5, 2, 0, 2, 4, 5, 4, 1, 1,
	1, 4, 4, 4, 1, 3, 3, 3, 2, 6,
	6, 3, 1, 1,
}
var yyChk = []int{

	-1000, -125, -79, -9, -80, -81, -82, -83, -10, 89,
	47, 48, -37, -84, -85, -86, -87, -88, -89, -1,
	-2, 159, 122, -5, -33, 85, -20, -26, -35, -38,
	65, 141, 32, 140, 84, -90, -91, -92, 90, 86,
	49, 133, 157, 156, 158, -3, -4, 161, 162, -34,
	18, -27, -28, 163, -39, 26, 38, 5, 165, 167,
	8, 125, 43, 9, 50, -41, -40, -43, -68, 53,
	121, 186, 167, 181, 85, 182, 183, 184, 180, 7,
	95, 173, 174, 175, 176, 177, 178, 179, 13, 89,
	77, 59, 153, 68, -9, -9, -79, -9, -70, 136,
	66, 44, -69, 96, 67, 67, 53, -93, -50, -51,
	159, 67, 163, -21, -22, -23, -9, -25, 149, -36,
	-9, -37, 104, 62, 104, 62, 62, -8, -7, -6,
	158, -13, -12, -9, -30, -29, -19, 159, -30, -30,
	-9, -9, -55, -56, 75, -44, -43, -42, -45, -51,
	-50, 128, -67, -66, 36, 4, -126, -65, 109, 40,
	182, -9, 159, 160, 167, -9, -9, -9, -9, -9,
	-9, -9, -9, -9, -9, -9, -9, -9, -9, -9,
	-9, -11, -10, 13, 77, 59, 153, -9, -9, -9,
	90, 89, 86, 146, -74, -73, 78, -39, 4, -39,
	4, -39, 4, 16, -93, -93, -93, -53, -52, 142,
	171, -18, -17, -16, 10, 159, -93, -13, 36, 182,
	42, -25, 149, -24, 41, -9, 164, 62, -117, 159,
	62, -118, -51, -50, -118, 166, 170, 171, 168, 170,
	-31, 170, 119, 59, 153, -31, -31, 52, 52, -57,
	-58, 150, -15, -14, -16, -55, -47, 64, 74, -49,
	186, 171, 171, 170, -66, -126, -66, -9, 186, -18,
	-9, 168, 171, 7, 186, 167, 181, 85, 182, 183,
	184, 180, -11, -9, -9, -9, 90, 86, 146, -76,
	-75, 92, -9, -39, -39, -39, -72, -71, -9, -96,
	-95, 70, -95, -53, -103, -106, 123, 139, -128, 104,
	-51, 159, -16, 144, 164, -9, 164, -24, -9, -9,
	129, 93, 93, 93, 186, 171, 186, -6, -9, -9,
	42, -29, -9, -9, -9, 42, 42, -30, -30, -59,
	-60, 56, -62, 76, -9, 170, 173, -57, 69, 88,
	-127, 138, 51, -129, 97, -18, -48, 159, -51, -51,
	-65, -9, -18, 182, 168, 169, 168, -9, -11, 159,
	160, 167, -9, -11, -11, -11, -11, -11, -11, 7,
	-9, 170, -78, -77, 11, 34, -94, -37, 147, -9,
	-94, -37, -57, -106, -57, -57, -105, -104, -48, -108,
	-107, -48, 71, -18, -45, 163, 164, 129, -9, -118,
	-118, -118, -117, -51, -117, -32, 149, -32, -68, 16,
	-15, -14, -9, -59, -46, -51, -50, 128, -46, -9,
	-53, 186, 167, -49, -49, -18, 168, -9, 168, 171,
	-11, -71, -99, -98, 114, -99, -9, -99, -99, -74,
	-57, -74, -74, 170, 173, 170, -110, -109, 52, -9,
	93, -37, -9, -120, 144, 163, -121, 112, 42, -9,
	42, -12, -49, 171, 171, -18, 159, 160, -9, -18,
	-18, 168, 169, 168, -9, -97, -66, -126, -99, -74,
	-99, -99, -104, -9, -107, -101, -100, -19, -95, 164,
	148, 79, -124, -122, -9, 130, -61, -62, -18, -51,
	-51, 168, -53, -53, 168, -9, -99, -110, -32, 170,
	59, 153, -111, 149, -17, 164, 170, -117, -63, -64,
	57, -54, 93, -49, -49, 42, -100, -102, -48, -102,
	-74, 82, 89, 93, -119, 99, -122, -9, -128, -18,
	-18, -99, 129, 82, -95, -123, 150, 16, 71, -54,
	-54, 140, 32, 129, -111, -120, -122, -9, -9, -113,
	-103, -106, -114, -57, 65, -74, -112, 149, -57, -106,
	-57, -116, 149, -115, -9, -99, 82, 89, -57, 89,
	-57, 129, 82, 82, 32, 129, 129, -114, 65, 65,
	-116, -115, -115,
}
var yyDef = []int{

	0, -2, 1, 2, 3, 4, 5, 6, 193, 0,
	0, 0, 8, 9, 10, 11, 12, 13, 14, 232,
	233, -2, 235, 236, 237, 0, 239, 240, 241, 99,
	0, 0, 0, 0, 0, 15, 16, 17, 256, 257,
	258, 259, 260, 261, 262, 263, 264, 274, 275, 0,
	0, 289, 290, 0, 19, 0, 0, 0, 266, 272,
	0, 0, 0, 0, 0, 26, 27, 78, 48, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 209, 231, 7, 238, 109, 0,
	0, 0, 100, 0, 0, 0, 0, 67, 0, 43,
	-2, 0, 272, 0, 277, 278, 0, 283, 0, 0,
	302, 303, 0, 0, 0, 0, 0, 0, 267, 268,
	0, 0, 273, 91, 0, 294, 0, 145, 0, 0,
	0, 0, 84, 79, 0, 78, 49, -2, 51, 65,
	0, 0, 30, 31, 0, 0, 0, 38, 36, 37,
	40, 43, 194, 195, 0, 0, 201, 202, 203, 204,
	205, 206, 207, 208, -2, -2, -2, -2, -2, -2,
	-2, 0, 242, 0, 0, 0, 0, -2, -2, -2,
	225, 0, 227, 229, 112, 110, 0, 20, 0, 22,
	0, 24, 0, 0, 119, 0, 67, 0, 68, 70,
	0, 118, 44, 45, 0, 47, 0, 0, 0, 0,
	276, 283, 0, 282, 0, 0, 301, 0, 0, 171,
	0, 0, 172, 0, 0, 265, 0, 0, 271, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 87,
	85, 0, 80, 81, 0, 84, 0, 73, 75, 43,
	0, 0, 0, 0, 32, 0, 33, 43, 0, 42,
	0, 198, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, -2, -2, -2, 226, 228, 230, 18,
	113, 0, 111, 21, 23, 25, 101, 102, 105, 0,
	120, 0, 0, 84, 84, 84, 0, 0, 0, 71,
	43, 64, 46, 0, 285, 0, 287, 279, 0, 284,
	0, 0, 0, 0, 0, 0, 0, 269, 270, 92,
	291, 295, 298, 296, 297, 292, 293, 147, 147, 0,
	88, 0, 90, 0, 86, 0, 0, 87, 0, 0,
	0, 55, 56, 74, 76, 67, 66, 189, 65, 65,
	39, 43, 34, 41, 196, 197, 199, 0, 217, 243,
	244, 0, 0, 250, 251, 252, 253, 254, 255, 0,
	114, 0, 104, 106, 107, 108, 123, 123, 0, 121,
	123, 123, 109, 84, 109, 109, 134, 135, 0, 149,
	150, 138, 0, 117, 0, 0, 286, 0, 280, 176,
	0, 184, 185, 173, 187, 0, 0, 0, 28, 0,
	95, 82, 83, 29, 52, 65, 0, 0, 53, 43,
	57, 0, 0, 43, 43, 35, 200, 0, 247, 0,
	218, 103, 115, 124, 0, 116, 122, 128, 129, 123,
	109, 123, 123, 0, 0, 0, 152, 139, 0, 69,
	0, 0, 281, 169, 0, 0, 186, 0, 299, 148,
	300, 93, 43, 0, 0, 54, 190, 191, 0, 67,
	67, 245, 246, 248, 0, 125, 126, 0, 130, 123,
	132, 133, 136, 138, 151, 147, 141, 0, 155, 0,
	177, 178, 0, 179, 181, 0, 96, 94, 0, 65,
	65, 192, 58, 59, 249, 127, 131, 137, 0, 0,
	0, 0, 109, 0, 0, 174, 0, 188, 89, 97,
	0, 60, 70, 43, 43, 140, 142, 143, 146, 144,
	123, 0, 0, 0, 182, 0, 180, 98, 0, 0,
	0, 153, 0, 0, 155, 176, 0, 0, 0, 61,
	62, 0, 84, 0, 109, 170, 183, 175, 77, 159,
	84, 84, 162, 167, 0, 123, 156, 0, 164, 84,
	166, 157, 0, 158, 84, 154, 0, 0, 165, 0,
	168, 0, 0, 0, 84, 0, 0, 162, 0, 0,
	160, 161, 163,
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
		//line n1ql.y:369
		{
			yyVAL.statement = algebra.NewExplain(yyS[yypt-0].statement)
		}
	case 8:
		//line n1ql.y:376
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
		//line n1ql.y:407
		{
			yyVAL.fullselect = algebra.NewSelect(yyS[yypt-3].subresult, yyS[yypt-2].order, yyS[yypt-0].expr, yyS[yypt-1].expr) /* OFFSET precedes LIMIT */
		}
	case 19:
		//line n1ql.y:413
		{
			yyVAL.subresult = yyS[yypt-0].subselect
		}
	case 20:
		//line n1ql.y:418
		{
			yyVAL.subresult = algebra.NewUnion(yyS[yypt-2].subresult, yyS[yypt-0].subselect)
		}
	case 21:
		//line n1ql.y:423
		{
			yyVAL.subresult = algebra.NewUnionAll(yyS[yypt-3].subresult, yyS[yypt-0].subselect)
		}
	case 22:
		//line n1ql.y:428
		{
			yyVAL.subresult = algebra.NewIntersect(yyS[yypt-2].subresult, yyS[yypt-0].subselect)
		}
	case 23:
		//line n1ql.y:433
		{
			yyVAL.subresult = algebra.NewIntersectAll(yyS[yypt-3].subresult, yyS[yypt-0].subselect)
		}
	case 24:
		//line n1ql.y:438
		{
			yyVAL.subresult = algebra.NewExcept(yyS[yypt-2].subresult, yyS[yypt-0].subselect)
		}
	case 25:
		//line n1ql.y:443
		{
			yyVAL.subresult = algebra.NewExceptAll(yyS[yypt-3].subresult, yyS[yypt-0].subselect)
		}
	case 26:
		yyVAL.subselect = yyS[yypt-0].subselect
	case 27:
		yyVAL.subselect = yyS[yypt-0].subselect
	case 28:
		//line n1ql.y:456
		{
			yyVAL.subselect = algebra.NewSubselect(yyS[yypt-4].fromTerm, yyS[yypt-3].bindings, yyS[yypt-2].expr, yyS[yypt-1].group, yyS[yypt-0].projection)
		}
	case 29:
		//line n1ql.y:463
		{
			yyVAL.subselect = algebra.NewSubselect(yyS[yypt-3].fromTerm, yyS[yypt-2].bindings, yyS[yypt-1].expr, yyS[yypt-0].group, yyS[yypt-4].projection)
		}
	case 30:
		//line n1ql.y:478
		{
			yyVAL.projection = yyS[yypt-0].projection
		}
	case 31:
		//line n1ql.y:485
		{
			yyVAL.projection = algebra.NewProjection(false, yyS[yypt-0].resultTerms)
		}
	case 32:
		//line n1ql.y:490
		{
			yyVAL.projection = algebra.NewProjection(true, yyS[yypt-0].resultTerms)
		}
	case 33:
		//line n1ql.y:495
		{
			yyVAL.projection = algebra.NewProjection(false, yyS[yypt-0].resultTerms)
		}
	case 34:
		//line n1ql.y:500
		{
			yyVAL.projection = algebra.NewRawProjection(false, yyS[yypt-1].expr, yyS[yypt-0].s)
		}
	case 35:
		//line n1ql.y:505
		{
			yyVAL.projection = algebra.NewRawProjection(true, yyS[yypt-1].expr, yyS[yypt-0].s)
		}
	case 38:
		//line n1ql.y:518
		{
			yyVAL.resultTerms = algebra.ResultTerms{yyS[yypt-0].resultTerm}
		}
	case 39:
		//line n1ql.y:523
		{
			yyVAL.resultTerms = append(yyS[yypt-2].resultTerms, yyS[yypt-0].resultTerm)
		}
	case 40:
		//line n1ql.y:530
		{
			yyVAL.resultTerm = algebra.NewResultTerm(nil, true, "")
		}
	case 41:
		//line n1ql.y:535
		{
			yyVAL.resultTerm = algebra.NewResultTerm(yyS[yypt-2].expr, true, "")
		}
	case 42:
		//line n1ql.y:540
		{
			yyVAL.resultTerm = algebra.NewResultTerm(yyS[yypt-1].expr, false, yyS[yypt-0].s)
		}
	case 43:
		//line n1ql.y:547
		{
			yyVAL.s = ""
		}
	case 44:
		yyVAL.s = yyS[yypt-0].s
	case 45:
		yyVAL.s = yyS[yypt-0].s
	case 46:
		//line n1ql.y:558
		{
			yyVAL.s = yyS[yypt-0].s
		}
	case 47:
		yyVAL.s = yyS[yypt-0].s
	case 48:
		//line n1ql.y:576
		{
			yyVAL.fromTerm = nil
		}
	case 49:
		yyVAL.fromTerm = yyS[yypt-0].fromTerm
	case 50:
		//line n1ql.y:585
		{
			yyVAL.fromTerm = yyS[yypt-0].fromTerm
		}
	case 51:
		//line n1ql.y:592
		{
			yyVAL.fromTerm = yyS[yypt-0].keyspaceTerm
		}
	case 52:
		//line n1ql.y:597
		{
			yyVAL.fromTerm = algebra.NewJoin(yyS[yypt-3].fromTerm, yyS[yypt-2].b, yyS[yypt-0].keyspaceTerm)
		}
	case 53:
		//line n1ql.y:602
		{
			yyVAL.fromTerm = algebra.NewNest(yyS[yypt-3].fromTerm, yyS[yypt-2].b, yyS[yypt-0].keyspaceTerm)
		}
	case 54:
		//line n1ql.y:607
		{
			yyVAL.fromTerm = algebra.NewUnnest(yyS[yypt-4].fromTerm, yyS[yypt-3].b, yyS[yypt-1].expr, yyS[yypt-0].s)
		}
	case 57:
		//line n1ql.y:620
		{
			yyVAL.keyspaceTerm = algebra.NewKeyspaceTerm("", yyS[yypt-3].s, yyS[yypt-2].path, yyS[yypt-1].s, yyS[yypt-0].expr)
		}
	case 58:
		//line n1ql.y:625
		{
			yyVAL.keyspaceTerm = algebra.NewKeyspaceTerm(yyS[yypt-5].s, yyS[yypt-3].s, yyS[yypt-2].path, yyS[yypt-1].s, yyS[yypt-0].expr)
		}
	case 59:
		//line n1ql.y:630
		{
			yyVAL.keyspaceTerm = algebra.NewKeyspaceTerm("#system", yyS[yypt-3].s, yyS[yypt-2].path, yyS[yypt-1].s, yyS[yypt-0].expr)
		}
	case 60:
		//line n1ql.y:637
		{
			yyVAL.keyspaceTerm = algebra.NewKeyspaceTerm("", yyS[yypt-3].s, yyS[yypt-2].path, yyS[yypt-1].s, yyS[yypt-0].expr)
		}
	case 61:
		//line n1ql.y:642
		{
			yyVAL.keyspaceTerm = algebra.NewKeyspaceTerm(yyS[yypt-5].s, yyS[yypt-3].s, yyS[yypt-2].path, yyS[yypt-1].s, yyS[yypt-0].expr)
		}
	case 62:
		//line n1ql.y:647
		{
			yyVAL.keyspaceTerm = algebra.NewKeyspaceTerm("#system", yyS[yypt-3].s, yyS[yypt-2].path, yyS[yypt-1].s, yyS[yypt-0].expr)
		}
	case 63:
		yyVAL.s = yyS[yypt-0].s
	case 64:
		yyVAL.s = yyS[yypt-0].s
	case 65:
		//line n1ql.y:662
		{
			yyVAL.path = nil
		}
	case 66:
		//line n1ql.y:667
		{
			yyVAL.path = yyS[yypt-0].path
		}
	case 67:
		//line n1ql.y:674
		{
			yyVAL.expr = nil
		}
	case 68:
		yyVAL.expr = yyS[yypt-0].expr
	case 69:
		//line n1ql.y:683
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 70:
		//line n1ql.y:690
		{
		}
	case 72:
		//line n1ql.y:698
		{
			yyVAL.b = false
		}
	case 73:
		//line n1ql.y:703
		{
			yyVAL.b = false
		}
	case 74:
		//line n1ql.y:708
		{
			yyVAL.b = true
		}
	case 77:
		//line n1ql.y:721
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 78:
		//line n1ql.y:735
		{
			yyVAL.bindings = nil
		}
	case 79:
		yyVAL.bindings = yyS[yypt-0].bindings
	case 80:
		//line n1ql.y:744
		{
			yyVAL.bindings = yyS[yypt-0].bindings
		}
	case 81:
		//line n1ql.y:751
		{
			yyVAL.bindings = expression.Bindings{yyS[yypt-0].binding}
		}
	case 82:
		//line n1ql.y:756
		{
			yyVAL.bindings = append(yyS[yypt-2].bindings, yyS[yypt-0].binding)
		}
	case 83:
		//line n1ql.y:763
		{
			yyVAL.binding = expression.NewBinding(yyS[yypt-2].s, yyS[yypt-0].expr)
		}
	case 84:
		//line n1ql.y:777
		{
			yyVAL.expr = nil
		}
	case 85:
		yyVAL.expr = yyS[yypt-0].expr
	case 86:
		//line n1ql.y:786
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 87:
		//line n1ql.y:800
		{
			yyVAL.group = nil
		}
	case 88:
		yyVAL.group = yyS[yypt-0].group
	case 89:
		//line n1ql.y:809
		{
			yyVAL.group = algebra.NewGroup(yyS[yypt-2].exprs, yyS[yypt-1].bindings, yyS[yypt-0].expr)
		}
	case 90:
		//line n1ql.y:814
		{
			yyVAL.group = algebra.NewGroup(nil, yyS[yypt-0].bindings, nil)
		}
	case 91:
		//line n1ql.y:821
		{
			yyVAL.exprs = expression.Expressions{yyS[yypt-0].expr}
		}
	case 92:
		//line n1ql.y:826
		{
			yyVAL.exprs = append(yyS[yypt-2].exprs, yyS[yypt-0].expr)
		}
	case 93:
		//line n1ql.y:833
		{
			yyVAL.bindings = nil
		}
	case 94:
		yyVAL.bindings = yyS[yypt-0].bindings
	case 95:
		//line n1ql.y:842
		{
			yyVAL.bindings = yyS[yypt-0].bindings
		}
	case 96:
		//line n1ql.y:849
		{
			yyVAL.expr = nil
		}
	case 97:
		yyVAL.expr = yyS[yypt-0].expr
	case 98:
		//line n1ql.y:858
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 99:
		//line n1ql.y:872
		{
			yyVAL.order = nil
		}
	case 100:
		yyVAL.order = yyS[yypt-0].order
	case 101:
		//line n1ql.y:881
		{
			yyVAL.order = algebra.NewOrder(yyS[yypt-0].sortTerms)
		}
	case 102:
		//line n1ql.y:888
		{
			yyVAL.sortTerms = algebra.SortTerms{yyS[yypt-0].sortTerm}
		}
	case 103:
		//line n1ql.y:893
		{
			yyVAL.sortTerms = append(yyS[yypt-2].sortTerms, yyS[yypt-0].sortTerm)
		}
	case 104:
		//line n1ql.y:900
		{
			yyVAL.sortTerm = algebra.NewSortTerm(yyS[yypt-1].expr, yyS[yypt-0].b)
		}
	case 105:
		//line n1ql.y:907
		{
			yyVAL.b = false
		}
	case 106:
		yyVAL.b = yyS[yypt-0].b
	case 107:
		//line n1ql.y:916
		{
			yyVAL.b = false
		}
	case 108:
		//line n1ql.y:921
		{
			yyVAL.b = true
		}
	case 109:
		//line n1ql.y:935
		{
			yyVAL.expr = nil
		}
	case 110:
		yyVAL.expr = yyS[yypt-0].expr
	case 111:
		//line n1ql.y:944
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 112:
		//line n1ql.y:958
		{
			yyVAL.expr = nil
		}
	case 113:
		yyVAL.expr = yyS[yypt-0].expr
	case 114:
		//line n1ql.y:967
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 115:
		//line n1ql.y:981
		{
			yyVAL.statement = algebra.NewInsertValues(yyS[yypt-3].keyspaceRef, yyS[yypt-2].expr, yyS[yypt-1].expr, yyS[yypt-0].projection)
		}
	case 116:
		//line n1ql.y:986
		{
			yyVAL.statement = algebra.NewInsertSelect(yyS[yypt-3].keyspaceRef, yyS[yypt-2].expr, yyS[yypt-1].fullselect, yyS[yypt-0].projection)
		}
	case 117:
		//line n1ql.y:993
		{
			yyVAL.keyspaceRef = algebra.NewKeyspaceRef(yyS[yypt-3].s, yyS[yypt-1].s, yyS[yypt-0].s)
		}
	case 118:
		//line n1ql.y:998
		{
			yyVAL.keyspaceRef = algebra.NewKeyspaceRef("", yyS[yypt-1].s, yyS[yypt-0].s)
		}
	case 119:
		//line n1ql.y:1005
		{
			yyVAL.expr = nil
		}
	case 120:
		yyVAL.expr = yyS[yypt-0].expr
	case 121:
		//line n1ql.y:1014
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 122:
		//line n1ql.y:1021
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 123:
		//line n1ql.y:1028
		{
			yyVAL.projection = nil
		}
	case 124:
		yyVAL.projection = yyS[yypt-0].projection
	case 125:
		//line n1ql.y:1037
		{
			yyVAL.projection = yyS[yypt-0].projection
		}
	case 126:
		//line n1ql.y:1044
		{
			yyVAL.projection = algebra.NewProjection(false, yyS[yypt-0].resultTerms)
		}
	case 127:
		//line n1ql.y:1049
		{
			yyVAL.projection = algebra.NewRawProjection(false, yyS[yypt-0].expr, "")
		}
	case 128:
		//line n1ql.y:1063
		{
			yyVAL.statement = algebra.NewUpsertValues(yyS[yypt-3].keyspaceRef, yyS[yypt-2].expr, yyS[yypt-1].expr, yyS[yypt-0].projection)
		}
	case 129:
		//line n1ql.y:1068
		{
			yyVAL.statement = algebra.NewUpsertSelect(yyS[yypt-3].keyspaceRef, yyS[yypt-2].expr, yyS[yypt-1].fullselect, yyS[yypt-0].projection)
		}
	case 130:
		//line n1ql.y:1082
		{
			yyVAL.statement = algebra.NewDelete(yyS[yypt-4].keyspaceRef, yyS[yypt-3].expr, yyS[yypt-2].expr, yyS[yypt-1].expr, yyS[yypt-0].projection)
		}
	case 131:
		//line n1ql.y:1096
		{
			yyVAL.statement = algebra.NewUpdate(yyS[yypt-6].keyspaceRef, yyS[yypt-5].expr, yyS[yypt-4].set, yyS[yypt-3].unset, yyS[yypt-2].expr, yyS[yypt-1].expr, yyS[yypt-0].projection)
		}
	case 132:
		//line n1ql.y:1101
		{
			yyVAL.statement = algebra.NewUpdate(yyS[yypt-5].keyspaceRef, yyS[yypt-4].expr, yyS[yypt-3].set, nil, yyS[yypt-2].expr, yyS[yypt-1].expr, yyS[yypt-0].projection)
		}
	case 133:
		//line n1ql.y:1106
		{
			yyVAL.statement = algebra.NewUpdate(yyS[yypt-5].keyspaceRef, yyS[yypt-4].expr, nil, yyS[yypt-3].unset, yyS[yypt-2].expr, yyS[yypt-1].expr, yyS[yypt-0].projection)
		}
	case 134:
		//line n1ql.y:1113
		{
			yyVAL.set = algebra.NewSet(yyS[yypt-0].setTerms)
		}
	case 135:
		//line n1ql.y:1120
		{
			yyVAL.setTerms = algebra.SetTerms{yyS[yypt-0].setTerm}
		}
	case 136:
		//line n1ql.y:1125
		{
			yyVAL.setTerms = append(yyS[yypt-2].setTerms, yyS[yypt-0].setTerm)
		}
	case 137:
		//line n1ql.y:1132
		{
			yyVAL.setTerm = algebra.NewSetTerm(yyS[yypt-3].path, yyS[yypt-1].expr, yyS[yypt-0].updateFor)
		}
	case 138:
		//line n1ql.y:1139
		{
			yyVAL.updateFor = nil
		}
	case 139:
		yyVAL.updateFor = yyS[yypt-0].updateFor
	case 140:
		//line n1ql.y:1148
		{
			yyVAL.updateFor = algebra.NewUpdateFor(yyS[yypt-2].bindings, yyS[yypt-1].expr)
		}
	case 141:
		//line n1ql.y:1155
		{
			yyVAL.bindings = expression.Bindings{yyS[yypt-0].binding}
		}
	case 142:
		//line n1ql.y:1160
		{
			yyVAL.bindings = append(yyS[yypt-2].bindings, yyS[yypt-0].binding)
		}
	case 143:
		//line n1ql.y:1167
		{
			yyVAL.binding = expression.NewBinding(yyS[yypt-2].s, yyS[yypt-0].expr)
		}
	case 144:
		//line n1ql.y:1172
		{
			yyVAL.binding = expression.NewDescendantBinding(yyS[yypt-2].s, yyS[yypt-0].expr)
		}
	case 145:
		yyVAL.s = yyS[yypt-0].s
	case 146:
		//line n1ql.y:1183
		{
			yyVAL.expr = yyS[yypt-0].path
		}
	case 147:
		//line n1ql.y:1190
		{
			yyVAL.expr = nil
		}
	case 148:
		//line n1ql.y:1195
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 149:
		//line n1ql.y:1202
		{
			yyVAL.unset = algebra.NewUnset(yyS[yypt-0].unsetTerms)
		}
	case 150:
		//line n1ql.y:1209
		{
			yyVAL.unsetTerms = algebra.UnsetTerms{yyS[yypt-0].unsetTerm}
		}
	case 151:
		//line n1ql.y:1214
		{
			yyVAL.unsetTerms = append(yyS[yypt-2].unsetTerms, yyS[yypt-0].unsetTerm)
		}
	case 152:
		//line n1ql.y:1221
		{
			yyVAL.unsetTerm = algebra.NewUnsetTerm(yyS[yypt-1].path, yyS[yypt-0].updateFor)
		}
	case 153:
		//line n1ql.y:1235
		{
			source := algebra.NewMergeSourceFrom(yyS[yypt-5].keyspaceTerm, "")
			yyVAL.statement = algebra.NewMerge(yyS[yypt-7].keyspaceRef, source, yyS[yypt-3].expr, yyS[yypt-2].mergeActions, yyS[yypt-1].expr, yyS[yypt-0].projection)
		}
	case 154:
		//line n1ql.y:1241
		{
			source := algebra.NewMergeSourceSelect(yyS[yypt-7].fullselect, yyS[yypt-5].s)
			yyVAL.statement = algebra.NewMerge(yyS[yypt-10].keyspaceRef, source, yyS[yypt-3].expr, yyS[yypt-2].mergeActions, yyS[yypt-1].expr, yyS[yypt-0].projection)
		}
	case 155:
		//line n1ql.y:1249
		{
			yyVAL.mergeActions = algebra.NewMergeActions(nil, nil, nil)
		}
	case 156:
		//line n1ql.y:1254
		{
			yyVAL.mergeActions = algebra.NewMergeActions(yyS[yypt-1].mergeUpdate, yyS[yypt-0].mergeActions.Delete(), yyS[yypt-0].mergeActions.Insert())
		}
	case 157:
		//line n1ql.y:1259
		{
			yyVAL.mergeActions = algebra.NewMergeActions(nil, yyS[yypt-1].mergeDelete, yyS[yypt-0].mergeInsert)
		}
	case 158:
		//line n1ql.y:1264
		{
			yyVAL.mergeActions = algebra.NewMergeActions(nil, nil, yyS[yypt-0].mergeInsert)
		}
	case 159:
		//line n1ql.y:1271
		{
			yyVAL.mergeActions = algebra.NewMergeActions(nil, nil, nil)
		}
	case 160:
		//line n1ql.y:1276
		{
			yyVAL.mergeActions = algebra.NewMergeActions(nil, yyS[yypt-1].mergeDelete, yyS[yypt-0].mergeInsert)
		}
	case 161:
		//line n1ql.y:1281
		{
			yyVAL.mergeActions = algebra.NewMergeActions(nil, nil, yyS[yypt-0].mergeInsert)
		}
	case 162:
		//line n1ql.y:1288
		{
			yyVAL.mergeInsert = nil
		}
	case 163:
		//line n1ql.y:1293
		{
			yyVAL.mergeInsert = yyS[yypt-0].mergeInsert
		}
	case 164:
		//line n1ql.y:1300
		{
			yyVAL.mergeUpdate = algebra.NewMergeUpdate(yyS[yypt-1].set, nil, yyS[yypt-0].expr)
		}
	case 165:
		//line n1ql.y:1305
		{
			yyVAL.mergeUpdate = algebra.NewMergeUpdate(yyS[yypt-2].set, yyS[yypt-1].unset, yyS[yypt-0].expr)
		}
	case 166:
		//line n1ql.y:1310
		{
			yyVAL.mergeUpdate = algebra.NewMergeUpdate(nil, yyS[yypt-1].unset, yyS[yypt-0].expr)
		}
	case 167:
		//line n1ql.y:1317
		{
			yyVAL.mergeDelete = algebra.NewMergeDelete(yyS[yypt-0].expr)
		}
	case 168:
		//line n1ql.y:1324
		{
			yyVAL.mergeInsert = algebra.NewMergeInsert(yyS[yypt-1].expr, yyS[yypt-0].expr)
		}
	case 169:
		//line n1ql.y:1338
		{
			yyVAL.statement = algebra.NewCreatePrimaryIndex(yyS[yypt-1].keyspaceRef, yyS[yypt-0].indexType)
		}
	case 170:
		//line n1ql.y:1343
		{
			yyVAL.statement = algebra.NewCreateIndex(yyS[yypt-8].s, yyS[yypt-6].keyspaceRef, yyS[yypt-4].exprs, yyS[yypt-2].expr, yyS[yypt-1].expr, yyS[yypt-0].indexType)
		}
	case 171:
		yyVAL.s = yyS[yypt-0].s
	case 172:
		//line n1ql.y:1354
		{
			yyVAL.keyspaceRef = algebra.NewKeyspaceRef("", yyS[yypt-0].s, "")
		}
	case 173:
		//line n1ql.y:1359
		{
			yyVAL.keyspaceRef = algebra.NewKeyspaceRef(yyS[yypt-2].s, yyS[yypt-0].s, "")
		}
	case 174:
		//line n1ql.y:1366
		{
			yyVAL.expr = nil
		}
	case 175:
		//line n1ql.y:1371
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 176:
		//line n1ql.y:1378
		{
			yyVAL.indexType = datastore.VIEW
		}
	case 177:
		//line n1ql.y:1383
		{
			yyVAL.indexType = datastore.VIEW
		}
	case 178:
		//line n1ql.y:1388
		{
			yyVAL.indexType = datastore.LSM
		}
	case 179:
		//line n1ql.y:1395
		{
			yyVAL.exprs = expression.Expressions{yyS[yypt-0].expr}
		}
	case 180:
		//line n1ql.y:1400
		{
			yyVAL.exprs = append(yyS[yypt-2].exprs, yyS[yypt-0].expr)
		}
	case 181:
		//line n1ql.y:1407
		{
			e := yyS[yypt-0].expr
			if !e.Indexable() {
				yylex.Error(fmt.Sprintf("Expression not indexable."))
			}

			yyVAL.expr = e
		}
	case 182:
		//line n1ql.y:1418
		{
			yyVAL.expr = nil
		}
	case 183:
		//line n1ql.y:1423
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 184:
		//line n1ql.y:1437
		{
			yyVAL.statement = algebra.NewDropIndex(yyS[yypt-0].keyspaceRef, "#primary")
		}
	case 185:
		//line n1ql.y:1442
		{
			yyVAL.statement = algebra.NewDropIndex(yyS[yypt-2].keyspaceRef, yyS[yypt-0].s)
		}
	case 186:
		//line n1ql.y:1455
		{
			yyVAL.statement = algebra.NewAlterIndex(yyS[yypt-3].keyspaceRef, yyS[yypt-1].s, yyS[yypt-0].s)
		}
	case 187:
		//line n1ql.y:1461
		{
			yyVAL.s = ""
		}
	case 188:
		//line n1ql.y:1466
		{
			yyVAL.s = yyS[yypt-0].s
		}
	case 189:
		//line n1ql.y:1480
		{
			yyVAL.path = expression.NewIdentifier(yyS[yypt-0].s)
		}
	case 190:
		//line n1ql.y:1485
		{
			yyVAL.path = expression.NewField(yyS[yypt-2].path, expression.NewFieldName(yyS[yypt-0].s))
		}
	case 191:
		//line n1ql.y:1490
		{
			field := expression.NewField(yyS[yypt-2].path, expression.NewFieldName(yyS[yypt-0].s))
			field.SetCaseInsensitive(true)
			yyVAL.path = field
		}
	case 192:
		//line n1ql.y:1497
		{
			yyVAL.path = expression.NewElement(yyS[yypt-3].path, yyS[yypt-1].expr)
		}
	case 193:
		yyVAL.expr = yyS[yypt-0].expr
	case 194:
		//line n1ql.y:1514
		{
			yyVAL.expr = expression.NewField(yyS[yypt-2].expr, expression.NewFieldName(yyS[yypt-0].s))
		}
	case 195:
		//line n1ql.y:1519
		{
			field := expression.NewField(yyS[yypt-2].expr, expression.NewFieldName(yyS[yypt-0].s))
			field.SetCaseInsensitive(true)
			yyVAL.expr = field
		}
	case 196:
		//line n1ql.y:1526
		{
			yyVAL.expr = expression.NewField(yyS[yypt-4].expr, yyS[yypt-1].expr)
		}
	case 197:
		//line n1ql.y:1531
		{
			field := expression.NewField(yyS[yypt-4].expr, yyS[yypt-1].expr)
			field.SetCaseInsensitive(true)
			yyVAL.expr = field
		}
	case 198:
		//line n1ql.y:1538
		{
			yyVAL.expr = expression.NewElement(yyS[yypt-3].expr, yyS[yypt-1].expr)
		}
	case 199:
		//line n1ql.y:1543
		{
			yyVAL.expr = expression.NewSlice(yyS[yypt-4].expr, yyS[yypt-2].expr)
		}
	case 200:
		//line n1ql.y:1548
		{
			yyVAL.expr = expression.NewSlice(yyS[yypt-5].expr, yyS[yypt-3].expr, yyS[yypt-1].expr)
		}
	case 201:
		//line n1ql.y:1554
		{
			yyVAL.expr = expression.NewAdd(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 202:
		//line n1ql.y:1559
		{
			yyVAL.expr = expression.NewSub(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 203:
		//line n1ql.y:1564
		{
			yyVAL.expr = expression.NewMult(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 204:
		//line n1ql.y:1569
		{
			yyVAL.expr = expression.NewDiv(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 205:
		//line n1ql.y:1574
		{
			yyVAL.expr = expression.NewMod(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 206:
		//line n1ql.y:1580
		{
			yyVAL.expr = expression.NewConcat(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 207:
		//line n1ql.y:1586
		{
			yyVAL.expr = expression.NewAnd(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 208:
		//line n1ql.y:1591
		{
			yyVAL.expr = expression.NewOr(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 209:
		//line n1ql.y:1596
		{
			yyVAL.expr = expression.NewNot(yyS[yypt-0].expr)
		}
	case 210:
		//line n1ql.y:1602
		{
			yyVAL.expr = expression.NewEq(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 211:
		//line n1ql.y:1607
		{
			yyVAL.expr = expression.NewEq(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 212:
		//line n1ql.y:1612
		{
			yyVAL.expr = expression.NewNE(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 213:
		//line n1ql.y:1617
		{
			yyVAL.expr = expression.NewLT(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 214:
		//line n1ql.y:1622
		{
			yyVAL.expr = expression.NewGT(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 215:
		//line n1ql.y:1627
		{
			yyVAL.expr = expression.NewLE(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 216:
		//line n1ql.y:1632
		{
			yyVAL.expr = expression.NewGE(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 217:
		//line n1ql.y:1637
		{
			yyVAL.expr = expression.NewBetween(yyS[yypt-4].expr, yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 218:
		//line n1ql.y:1642
		{
			yyVAL.expr = expression.NewNotBetween(yyS[yypt-5].expr, yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 219:
		//line n1ql.y:1647
		{
			yyVAL.expr = expression.NewLike(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 220:
		//line n1ql.y:1652
		{
			yyVAL.expr = expression.NewNotLike(yyS[yypt-3].expr, yyS[yypt-0].expr)
		}
	case 221:
		//line n1ql.y:1657
		{
			yyVAL.expr = expression.NewIn(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 222:
		//line n1ql.y:1662
		{
			yyVAL.expr = expression.NewNotIn(yyS[yypt-3].expr, yyS[yypt-0].expr)
		}
	case 223:
		//line n1ql.y:1667
		{
			yyVAL.expr = expression.NewWithin(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 224:
		//line n1ql.y:1672
		{
			yyVAL.expr = expression.NewNotWithin(yyS[yypt-3].expr, yyS[yypt-0].expr)
		}
	case 225:
		//line n1ql.y:1677
		{
			yyVAL.expr = expression.NewIsNull(yyS[yypt-2].expr)
		}
	case 226:
		//line n1ql.y:1682
		{
			yyVAL.expr = expression.NewIsNotNull(yyS[yypt-3].expr)
		}
	case 227:
		//line n1ql.y:1687
		{
			yyVAL.expr = expression.NewIsMissing(yyS[yypt-2].expr)
		}
	case 228:
		//line n1ql.y:1692
		{
			yyVAL.expr = expression.NewIsNotMissing(yyS[yypt-3].expr)
		}
	case 229:
		//line n1ql.y:1697
		{
			yyVAL.expr = expression.NewIsValued(yyS[yypt-2].expr)
		}
	case 230:
		//line n1ql.y:1702
		{
			yyVAL.expr = expression.NewIsNotValued(yyS[yypt-3].expr)
		}
	case 231:
		//line n1ql.y:1707
		{
			yyVAL.expr = expression.NewExists(yyS[yypt-0].expr)
		}
	case 232:
		yyVAL.expr = yyS[yypt-0].expr
	case 233:
		yyVAL.expr = yyS[yypt-0].expr
	case 234:
		//line n1ql.y:1721
		{
			yyVAL.expr = expression.NewIdentifier(yyS[yypt-0].s)
		}
	case 235:
		//line n1ql.y:1727
		{
			yyVAL.expr = expression.NewSelf()
		}
	case 236:
		yyVAL.expr = yyS[yypt-0].expr
	case 237:
		yyVAL.expr = yyS[yypt-0].expr
	case 238:
		//line n1ql.y:1739
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
		//line n1ql.y:1758
		{
			yyVAL.expr = expression.NewField(yyS[yypt-2].expr, expression.NewFieldName(yyS[yypt-0].s))
		}
	case 244:
		//line n1ql.y:1763
		{
			field := expression.NewField(yyS[yypt-2].expr, expression.NewFieldName(yyS[yypt-0].s))
			field.SetCaseInsensitive(true)
			yyVAL.expr = field
		}
	case 245:
		//line n1ql.y:1770
		{
			yyVAL.expr = expression.NewField(yyS[yypt-4].expr, yyS[yypt-1].expr)
		}
	case 246:
		//line n1ql.y:1775
		{
			field := expression.NewField(yyS[yypt-4].expr, yyS[yypt-1].expr)
			field.SetCaseInsensitive(true)
			yyVAL.expr = field
		}
	case 247:
		//line n1ql.y:1782
		{
			yyVAL.expr = expression.NewElement(yyS[yypt-3].expr, yyS[yypt-1].expr)
		}
	case 248:
		//line n1ql.y:1787
		{
			yyVAL.expr = expression.NewSlice(yyS[yypt-4].expr, yyS[yypt-2].expr)
		}
	case 249:
		//line n1ql.y:1792
		{
			yyVAL.expr = expression.NewSlice(yyS[yypt-5].expr, yyS[yypt-3].expr, yyS[yypt-1].expr)
		}
	case 250:
		//line n1ql.y:1798
		{
			yyVAL.expr = expression.NewAdd(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 251:
		//line n1ql.y:1803
		{
			yyVAL.expr = expression.NewSub(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 252:
		//line n1ql.y:1808
		{
			yyVAL.expr = expression.NewMult(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 253:
		//line n1ql.y:1813
		{
			yyVAL.expr = expression.NewDiv(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 254:
		//line n1ql.y:1818
		{
			yyVAL.expr = expression.NewMod(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 255:
		//line n1ql.y:1824
		{
			yyVAL.expr = expression.NewConcat(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 256:
		//line n1ql.y:1838
		{
			yyVAL.expr = expression.NULL_EXPR
		}
	case 257:
		//line n1ql.y:1843
		{
			yyVAL.expr = expression.MISSING_EXPR
		}
	case 258:
		//line n1ql.y:1848
		{
			yyVAL.expr = expression.FALSE_EXPR
		}
	case 259:
		//line n1ql.y:1853
		{
			yyVAL.expr = expression.TRUE_EXPR
		}
	case 260:
		//line n1ql.y:1858
		{
			yyVAL.expr = expression.NewConstant(value.NewValue(yyS[yypt-0].f))
		}
	case 261:
		//line n1ql.y:1863
		{
			yyVAL.expr = expression.NewConstant(value.NewValue(yyS[yypt-0].n))
		}
	case 262:
		//line n1ql.y:1868
		{
			yyVAL.expr = expression.NewConstant(value.NewValue(yyS[yypt-0].s))
		}
	case 263:
		yyVAL.expr = yyS[yypt-0].expr
	case 264:
		yyVAL.expr = yyS[yypt-0].expr
	case 265:
		//line n1ql.y:1888
		{
			yyVAL.expr = expression.NewObjectConstruct(yyS[yypt-1].bindings)
		}
	case 266:
		//line n1ql.y:1895
		{
			yyVAL.bindings = nil
		}
	case 267:
		yyVAL.bindings = yyS[yypt-0].bindings
	case 268:
		//line n1ql.y:1904
		{
			yyVAL.bindings = expression.Bindings{yyS[yypt-0].binding}
		}
	case 269:
		//line n1ql.y:1909
		{
			yyVAL.bindings = append(yyS[yypt-2].bindings, yyS[yypt-0].binding)
		}
	case 270:
		//line n1ql.y:1916
		{
			yyVAL.binding = expression.NewBinding(yyS[yypt-2].s, yyS[yypt-0].expr)
		}
	case 271:
		//line n1ql.y:1923
		{
			yyVAL.expr = expression.NewArrayConstruct(yyS[yypt-1].exprs...)
		}
	case 272:
		//line n1ql.y:1930
		{
			yyVAL.exprs = nil
		}
	case 273:
		yyVAL.exprs = yyS[yypt-0].exprs
	case 274:
		//line n1ql.y:1946
		{
			yyVAL.expr = algebra.NewNamedParameter(yyS[yypt-0].s)
		}
	case 275:
		//line n1ql.y:1951
		{
			yyVAL.expr = algebra.NewPositionalParameter(yyS[yypt-0].n)
		}
	case 276:
		//line n1ql.y:1965
		{
			yyVAL.expr = yyS[yypt-1].expr
		}
	case 277:
		yyVAL.expr = yyS[yypt-0].expr
	case 278:
		yyVAL.expr = yyS[yypt-0].expr
	case 279:
		//line n1ql.y:1978
		{
			yyVAL.expr = expression.NewSimpleCase(yyS[yypt-2].expr, yyS[yypt-1].whenTerms, yyS[yypt-0].expr)
		}
	case 280:
		//line n1ql.y:1985
		{
			yyVAL.whenTerms = expression.WhenTerms{&expression.WhenTerm{yyS[yypt-2].expr, yyS[yypt-0].expr}}
		}
	case 281:
		//line n1ql.y:1990
		{
			yyVAL.whenTerms = append(yyS[yypt-4].whenTerms, &expression.WhenTerm{yyS[yypt-2].expr, yyS[yypt-0].expr})
		}
	case 282:
		//line n1ql.y:1998
		{
			yyVAL.expr = expression.NewSearchedCase(yyS[yypt-1].whenTerms, yyS[yypt-0].expr)
		}
	case 283:
		//line n1ql.y:2005
		{
			yyVAL.expr = nil
		}
	case 284:
		//line n1ql.y:2010
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 285:
		//line n1ql.y:2024
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
		//line n1ql.y:2043
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
		//line n1ql.y:2058
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
		//line n1ql.y:2092
		{
			yyVAL.expr = expression.NewAny(yyS[yypt-2].bindings, yyS[yypt-1].expr)
		}
	case 292:
		//line n1ql.y:2097
		{
			yyVAL.expr = expression.NewAny(yyS[yypt-2].bindings, yyS[yypt-1].expr)
		}
	case 293:
		//line n1ql.y:2102
		{
			yyVAL.expr = expression.NewEvery(yyS[yypt-2].bindings, yyS[yypt-1].expr)
		}
	case 294:
		//line n1ql.y:2109
		{
			yyVAL.bindings = expression.Bindings{yyS[yypt-0].binding}
		}
	case 295:
		//line n1ql.y:2114
		{
			yyVAL.bindings = append(yyS[yypt-2].bindings, yyS[yypt-0].binding)
		}
	case 296:
		//line n1ql.y:2121
		{
			yyVAL.binding = expression.NewBinding(yyS[yypt-2].s, yyS[yypt-0].expr)
		}
	case 297:
		//line n1ql.y:2126
		{
			yyVAL.binding = expression.NewDescendantBinding(yyS[yypt-2].s, yyS[yypt-0].expr)
		}
	case 298:
		//line n1ql.y:2133
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 299:
		//line n1ql.y:2140
		{
			yyVAL.expr = expression.NewArray(yyS[yypt-4].expr, yyS[yypt-2].bindings, yyS[yypt-1].expr)
		}
	case 300:
		//line n1ql.y:2145
		{
			yyVAL.expr = expression.NewFirst(yyS[yypt-4].expr, yyS[yypt-2].bindings, yyS[yypt-1].expr)
		}
	case 301:
		//line n1ql.y:2159
		{
			yyVAL.expr = yyS[yypt-1].expr
		}
	case 302:
		yyVAL.expr = yyS[yypt-0].expr
	case 303:
		//line n1ql.y:2168
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
