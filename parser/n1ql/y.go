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
	-1, 25,
	163, 292,
	-2, 238,
	-1, 116,
	171, 67,
	-2, 68,
	-1, 153,
	51, 76,
	69, 76,
	88, 76,
	138, 76,
	-2, 54,
	-1, 180,
	173, 0,
	174, 0,
	175, 0,
	-2, 214,
	-1, 181,
	173, 0,
	174, 0,
	175, 0,
	-2, 215,
	-1, 182,
	173, 0,
	174, 0,
	175, 0,
	-2, 216,
	-1, 183,
	176, 0,
	177, 0,
	178, 0,
	179, 0,
	-2, 217,
	-1, 184,
	176, 0,
	177, 0,
	178, 0,
	179, 0,
	-2, 218,
	-1, 185,
	176, 0,
	177, 0,
	178, 0,
	179, 0,
	-2, 219,
	-1, 186,
	176, 0,
	177, 0,
	178, 0,
	179, 0,
	-2, 220,
	-1, 193,
	77, 0,
	-2, 223,
	-1, 194,
	59, 0,
	153, 0,
	-2, 225,
	-1, 195,
	59, 0,
	153, 0,
	-2, 227,
	-1, 289,
	77, 0,
	-2, 224,
	-1, 290,
	59, 0,
	153, 0,
	-2, 226,
	-1, 291,
	59, 0,
	153, 0,
	-2, 228,
}

const yyNprod = 308
const yyPrivate = 57344

var yyTokenNames []string
var yyStates []string

const yyLast = 2668

var yyAct = []int{

	167, 3, 589, 578, 448, 587, 579, 310, 311, 200,
	509, 528, 98, 99, 537, 469, 306, 217, 404, 543,
	314, 218, 142, 502, 265, 462, 421, 234, 348, 406,
	103, 303, 213, 403, 159, 16, 115, 162, 138, 345,
	430, 72, 154, 392, 237, 259, 140, 141, 229, 258,
	163, 135, 58, 219, 266, 122, 332, 281, 126, 330,
	464, 352, 187, 349, 139, 64, 67, 480, 146, 147,
	479, 331, 284, 285, 286, 54, 280, 171, 172, 173,
	174, 175, 176, 177, 178, 179, 180, 181, 182, 183,
	184, 185, 186, 127, 268, 193, 194, 195, 267, 248,
	66, 76, 438, 438, 12, 281, 44, 68, 460, 76,
	155, 243, 144, 145, 422, 114, 79, 80, 81, 139,
	75, 437, 437, 422, 280, 231, 531, 216, 75, 168,
	169, 269, 532, 351, 461, 525, 459, 170, 387, 245,
	244, 242, 29, 43, 247, 10, 11, 42, 241, 385,
	247, 62, 369, 375, 376, 255, 505, 245, 203, 205,
	207, 377, 322, 273, 168, 169, 157, 320, 238, 238,
	232, 276, 170, 471, 118, 438, 433, 240, 157, 26,
	482, 483, 65, 363, 220, 275, 143, 235, 317, 156,
	45, 289, 290, 291, 437, 270, 272, 116, 271, 221,
	116, 411, 136, 298, 260, 526, 257, 116, 249, 116,
	304, 562, 588, 47, 46, 48, 25, 148, 51, 52,
	57, 313, 62, 189, 63, 321, 583, 283, 308, 324,
	230, 325, 257, 529, 507, 246, 470, 198, 188, 166,
	197, 196, 319, 73, 334, 309, 335, 239, 239, 338,
	339, 340, 288, 316, 215, 511, 602, 299, 350, 300,
	601, 301, 568, 597, 312, 569, 558, 450, 353, 191,
	473, 293, 367, 74, 318, 292, 323, 315, 551, 373,
	313, 123, 378, 361, 107, 362, 73, 190, 131, 360,
	208, 368, 358, 137, 333, 337, 538, 199, 386, 527,
	343, 344, 250, 506, 364, 365, 106, 549, 395, 281,
	354, 74, 466, 129, 329, 328, 398, 400, 401, 399,
	366, 327, 287, 282, 284, 285, 286, 414, 280, 355,
	130, 294, 407, 221, 409, 188, 109, 394, 228, 73,
	297, 393, 374, 595, 397, 379, 380, 381, 382, 383,
	384, 206, 396, 428, 74, 128, 155, 435, 418, 599,
	420, 598, 410, 192, 238, 238, 238, 204, 419, 261,
	567, 423, 415, 416, 417, 592, 105, 547, 443, 357,
	251, 252, 593, 559, 548, 441, 113, 424, 304, 439,
	440, 431, 431, 429, 436, 452, 434, 427, 451, 426,
	73, 453, 454, 260, 227, 260, 456, 74, 455, 465,
	457, 458, 223, 283, 468, 202, 73, 347, 150, 447,
	564, 408, 307, 475, 263, 188, 139, 605, 188, 188,
	188, 188, 188, 188, 264, 156, 117, 349, 111, 484,
	71, 110, 604, 239, 239, 239, 490, 467, 446, 580,
	236, 233, 132, 481, 536, 73, 478, 485, 486, 112,
	494, 499, 496, 497, 477, 541, 495, 476, 74, 474,
	432, 432, 510, 342, 341, 336, 279, 226, 600, 563,
	407, 425, 209, 504, 74, 492, 49, 503, 493, 359,
	356, 500, 1, 498, 521, 281, 514, 210, 211, 212,
	522, 508, 102, 561, 222, 472, 513, 2, 287, 282,
	284, 285, 286, 152, 280, 550, 515, 516, 518, 519,
	575, 100, 101, 582, 463, 523, 405, 530, 524, 402,
	501, 188, 449, 510, 491, 305, 41, 553, 546, 533,
	539, 540, 40, 552, 283, 544, 544, 545, 503, 542,
	83, 557, 39, 22, 283, 21, 92, 555, 556, 554,
	20, 19, 78, 510, 573, 574, 560, 18, 17, 9,
	565, 566, 570, 572, 8, 576, 577, 571, 7, 6,
	581, 590, 5, 584, 586, 585, 591, 83, 4, 388,
	220, 389, 594, 92, 295, 296, 201, 596, 302, 104,
	108, 158, 95, 535, 603, 590, 590, 607, 608, 606,
	534, 97, 512, 346, 256, 149, 214, 262, 151, 83,
	94, 153, 69, 70, 33, 92, 281, 125, 78, 32,
	53, 28, 93, 56, 55, 31, 281, 121, 84, 95,
	282, 284, 285, 286, 76, 280, 120, 119, 97, 287,
	282, 284, 285, 286, 30, 280, 133, 94, 77, 79,
	80, 81, 134, 75, 27, 78, 50, 24, 23, 93,
	0, 95, 0, 0, 0, 84, 0, 0, 0, 0,
	97, 0, 0, 0, 0, 0, 0, 0, 0, 94,
	0, 0, 0, 64, 67, 92, 96, 78, 0, 0,
	0, 93, 0, 54, 0, 0, 0, 84, 0, 0,
	76, 487, 488, 0, 0, 0, 85, 86, 87, 88,
	89, 90, 91, 82, 77, 79, 80, 81, 66, 75,
	0, 0, 12, 96, 44, 68, 0, 0, 0, 221,
	0, 95, 0, 0, 0, 83, 0, 76, 0, 390,
	97, 92, 0, 85, 86, 87, 88, 89, 90, 91,
	82, 77, 79, 80, 81, 96, 75, 78, 0, 0,
	29, 43, 391, 0, 11, 42, 0, 0, 0, 76,
	444, 0, 0, 445, 97, 85, 86, 87, 88, 89,
	90, 91, 82, 77, 79, 80, 81, 95, 75, 83,
	0, 78, 0, 0, 0, 92, 97, 26, 0, 0,
	65, 0, 0, 0, 0, 94, 0, 0, 45, 0,
	0, 0, 0, 78, 0, 0, 0, 93, 0, 0,
	0, 0, 0, 84, 0, 96, 0, 0, 0, 0,
	0, 47, 46, 48, 25, 0, 51, 52, 57, 76,
	62, 95, 63, 489, 0, 0, 0, 0, 0, 0,
	97, 0, 82, 77, 79, 80, 81, 0, 75, 94,
	0, 0, 0, 0, 83, 0, 0, 78, 0, 0,
	92, 93, 0, 76, 0, 0, 0, 84, 0, 0,
	0, 96, 0, 0, 0, 0, 82, 77, 79, 80,
	81, 0, 75, 0, 0, 76, 0, 0, 0, 0,
	0, 85, 86, 87, 88, 89, 90, 91, 82, 77,
	79, 80, 81, 0, 75, 83, 95, 0, 220, 0,
	0, 92, 0, 0, 0, 97, 0, 0, 0, 0,
	0, 0, 0, 0, 94, 96, 0, 0, 0, 0,
	0, 0, 78, 0, 0, 0, 93, 0, 0, 76,
	370, 371, 84, 0, 0, 85, 86, 87, 88, 89,
	90, 91, 82, 77, 79, 80, 81, 95, 75, 0,
	0, 0, 0, 0, 0, 0, 97, 0, 0, 0,
	0, 0, 0, 0, 0, 94, 0, 0, 0, 0,
	83, 0, 0, 78, 0, 0, 92, 93, 0, 0,
	0, 0, 0, 84, 0, 0, 0, 0, 0, 0,
	96, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 76, 277, 0, 0, 278, 0,
	85, 86, 87, 88, 89, 90, 91, 82, 77, 79,
	80, 81, 95, 75, 0, 0, 0, 0, 0, 0,
	83, 97, 0, 0, 0, 0, 92, 0, 0, 0,
	94, 96, 0, 0, 0, 0, 0, 221, 78, 0,
	0, 0, 93, 0, 0, 76, 0, 0, 84, 0,
	0, 85, 86, 87, 88, 89, 90, 91, 82, 77,
	79, 80, 81, 0, 274, 464, 0, 0, 0, 0,
	0, 0, 95, 0, 0, 0, 0, 0, 0, 0,
	0, 97, 0, 0, 0, 0, 83, 0, 0, 0,
	94, 0, 92, 0, 0, 0, 0, 0, 78, 0,
	0, 0, 93, 257, 0, 0, 96, 0, 84, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	76, 0, 0, 0, 0, 0, 85, 86, 87, 88,
	89, 90, 91, 82, 77, 79, 80, 81, 95, 75,
	0, 0, 0, 0, 0, 0, 83, 97, 0, 0,
	0, 0, 92, 0, 0, 0, 94, 0, 0, 0,
	0, 0, 0, 0, 78, 0, 96, 0, 93, 0,
	0, 0, 0, 0, 84, 0, 0, 0, 0, 0,
	76, 0, 0, 0, 0, 0, 85, 86, 87, 88,
	89, 90, 91, 82, 77, 79, 80, 81, 95, 75,
	0, 0, 0, 0, 0, 0, 0, 97, 0, 0,
	0, 0, 83, 0, 0, 0, 94, 0, 92, 0,
	0, 0, 0, 0, 78, 0, 0, 0, 93, 0,
	0, 0, 96, 0, 84, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 76, 520, 0, 0,
	0, 0, 85, 86, 87, 88, 89, 90, 91, 82,
	77, 79, 80, 81, 95, 75, 0, 0, 0, 0,
	0, 0, 83, 97, 0, 0, 0, 0, 92, 0,
	0, 0, 94, 0, 0, 0, 0, 0, 0, 0,
	78, 0, 96, 0, 93, 0, 0, 0, 0, 0,
	84, 0, 0, 0, 0, 0, 76, 517, 0, 0,
	0, 0, 85, 86, 87, 88, 89, 90, 91, 82,
	77, 79, 80, 81, 95, 75, 0, 0, 0, 0,
	0, 0, 0, 97, 0, 0, 0, 0, 83, 0,
	0, 0, 94, 0, 92, 0, 0, 0, 0, 0,
	78, 0, 0, 0, 93, 0, 0, 0, 96, 0,
	84, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 76, 442, 0, 0, 0, 0, 85, 86,
	87, 88, 89, 90, 91, 82, 77, 79, 80, 81,
	95, 75, 0, 0, 413, 0, 0, 0, 83, 97,
	0, 0, 0, 0, 92, 0, 0, 0, 94, 0,
	0, 0, 0, 0, 0, 0, 78, 0, 96, 0,
	93, 0, 0, 0, 0, 0, 84, 0, 0, 0,
	0, 0, 76, 0, 0, 0, 0, 0, 85, 86,
	87, 88, 89, 90, 91, 82, 77, 79, 80, 81,
	95, 75, 0, 0, 0, 0, 0, 0, 0, 97,
	0, 0, 0, 0, 0, 0, 0, 0, 94, 0,
	0, 83, 0, 0, 0, 0, 78, 92, 0, 0,
	93, 0, 0, 0, 96, 0, 84, 0, 0, 0,
	0, 0, 0, 0, 0, 412, 0, 0, 76, 0,
	0, 0, 0, 0, 85, 86, 87, 88, 89, 90,
	91, 82, 77, 79, 80, 81, 254, 75, 0, 0,
	326, 0, 0, 95, 0, 0, 0, 0, 0, 0,
	0, 83, 97, 0, 0, 0, 0, 92, 0, 0,
	0, 94, 0, 0, 96, 0, 0, 0, 0, 78,
	0, 0, 0, 93, 0, 0, 0, 0, 76, 84,
	0, 0, 0, 0, 85, 86, 87, 88, 89, 90,
	91, 82, 77, 79, 80, 81, 253, 75, 0, 0,
	0, 0, 0, 95, 0, 0, 0, 0, 0, 0,
	0, 0, 97, 0, 0, 0, 0, 83, 0, 0,
	0, 94, 0, 92, 0, 0, 0, 0, 0, 78,
	0, 0, 0, 93, 0, 0, 0, 96, 0, 84,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 76, 0, 0, 0, 0, 0, 85, 86, 87,
	88, 89, 90, 91, 82, 77, 79, 80, 81, 95,
	75, 0, 0, 0, 0, 0, 0, 0, 97, 0,
	0, 0, 0, 0, 0, 0, 0, 94, 0, 0,
	0, 0, 0, 0, 0, 78, 0, 96, 0, 93,
	0, 0, 0, 0, 0, 84, 0, 0, 0, 0,
	0, 76, 0, 0, 0, 0, 0, 85, 86, 87,
	88, 89, 90, 91, 82, 77, 79, 80, 81, 161,
	75, 0, 0, 64, 67, 0, 0, 0, 0, 0,
	0, 0, 0, 54, 0, 0, 0, 0, 83, 0,
	0, 0, 0, 0, 92, 0, 0, 0, 0, 124,
	0, 160, 0, 96, 0, 165, 0, 0, 66, 0,
	0, 0, 12, 0, 44, 68, 0, 76, 0, 0,
	0, 0, 0, 85, 86, 87, 88, 89, 90, 91,
	82, 77, 79, 80, 81, 0, 75, 0, 0, 0,
	95, 0, 0, 0, 0, 0, 0, 0, 0, 97,
	29, 43, 0, 0, 11, 42, 0, 0, 94, 0,
	0, 0, 0, 0, 0, 0, 78, 0, 64, 67,
	93, 0, 0, 0, 164, 0, 84, 0, 54, 0,
	0, 0, 0, 0, 0, 0, 0, 26, 0, 0,
	65, 0, 0, 0, 0, 0, 0, 0, 45, 0,
	165, 0, 0, 66, 0, 0, 0, 12, 0, 44,
	68, 0, 0, 0, 83, 0, 0, 0, 0, 0,
	92, 47, 46, 48, 25, 0, 51, 52, 57, 0,
	62, 0, 63, 0, 96, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 29, 43, 166, 76, 11,
	42, 0, 0, 0, 85, 86, 87, 88, 89, 90,
	91, 82, 77, 79, 80, 81, 95, 75, 0, 164,
	0, 0, 0, 0, 92, 97, 0, 0, 0, 0,
	0, 0, 26, 0, 94, 65, 0, 0, 0, 0,
	0, 0, 78, 45, 0, 0, 93, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 92, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 47, 46, 48, 25,
	95, 51, 52, 57, 0, 62, 0, 63, 0, 97,
	0, 0, 0, 0, 64, 67, 0, 0, 94, 0,
	0, 0, 166, 0, 54, 0, 78, 0, 0, 0,
	93, 0, 95, 0, 0, 0, 0, 0, 0, 0,
	96, 97, 224, 0, 0, 0, 0, 0, 0, 66,
	94, 0, 0, 12, 76, 44, 68, 0, 78, 0,
	85, 86, 87, 88, 89, 90, 91, 82, 77, 79,
	80, 81, 0, 75, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 29, 43, 0, 96, 11, 42, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 76, 0,
	0, 0, 0, 0, 85, 86, 87, 88, 89, 90,
	91, 82, 77, 79, 80, 81, 96, 75, 26, 0,
	0, 65, 0, 0, 0, 0, 0, 0, 0, 45,
	76, 0, 0, 0, 0, 0, 0, 0, 0, 88,
	89, 90, 91, 82, 77, 79, 80, 81, 0, 75,
	0, 0, 47, 46, 48, 25, 0, 51, 52, 57,
	0, 62, 61, 63, 0, 64, 67, 0, 0, 0,
	0, 0, 92, 0, 0, 54, 0, 0, 225, 0,
	0, 0, 0, 59, 0, 0, 0, 0, 0, 36,
	0, 0, 0, 0, 0, 60, 0, 0, 0, 0,
	66, 0, 0, 15, 12, 13, 44, 68, 0, 0,
	73, 0, 0, 0, 0, 0, 0, 0, 95, 0,
	0, 0, 34, 0, 0, 0, 0, 97, 0, 0,
	0, 0, 0, 0, 0, 0, 94, 0, 0, 0,
	0, 38, 29, 43, 78, 0, 11, 42, 0, 0,
	0, 0, 64, 67, 0, 0, 0, 0, 0, 0,
	14, 0, 54, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 74, 26,
	0, 0, 65, 0, 0, 0, 0, 66, 0, 0,
	45, 12, 0, 44, 68, 0, 0, 37, 35, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 96, 47, 46, 48, 25, 0, 51, 52,
	57, 0, 62, 0, 63, 0, 76, 0, 0, 29,
	43, 0, 0, 11, 42, 0, 0, 64, 67, 82,
	77, 79, 80, 81, 0, 75, 0, 54, 0, 0,
	0, 0, 64, 67, 0, 0, 0, 0, 0, 0,
	0, 0, 54, 0, 0, 0, 26, 0, 0, 65,
	0, 0, 66, 0, 0, 0, 12, 45, 44, 68,
	0, 0, 73, 0, 0, 0, 0, 66, 0, 0,
	0, 12, 0, 44, 68, 0, 0, 0, 0, 0,
	47, 46, 48, 25, 0, 51, 52, 57, 0, 62,
	0, 63, 372, 0, 29, 43, 0, 0, 11, 42,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 29,
	43, 0, 0, 11, 42, 0, 0, 64, 67, 0,
	0, 0, 0, 0, 0, 0, 0, 54, 0, 0,
	74, 26, 0, 0, 65, 0, 0, 0, 0, 0,
	0, 0, 45, 0, 0, 0, 26, 0, 0, 65,
	0, 0, 66, 0, 0, 0, 12, 45, 44, 68,
	0, 0, 0, 0, 0, 47, 46, 48, 25, 0,
	51, 52, 57, 124, 62, 0, 63, 0, 64, 67,
	47, 46, 48, 25, 61, 51, 52, 57, 54, 62,
	0, 63, 0, 0, 29, 43, 0, 0, 11, 42,
	0, 0, 0, 0, 0, 59, 0, 0, 0, 0,
	0, 36, 0, 66, 0, 0, 0, 60, 0, 44,
	68, 0, 0, 0, 0, 15, 0, 13, 0, 0,
	0, 26, 73, 0, 65, 0, 0, 0, 0, 0,
	0, 0, 45, 0, 34, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 29, 43, 0, 0, 0,
	42, 0, 0, 38, 0, 47, 46, 48, 25, 0,
	51, 52, 57, 0, 62, 0, 63, 0, 0, 0,
	0, 0, 14, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 26, 0, 0, 65, 0, 0, 0, 0,
	74, 0, 0, 45, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 37,
	35, 0, 0, 0, 0, 0, 47, 46, 48, 25,
	0, 51, 52, 57, 0, 62, 0, 63,
}
var yyPact = []int{

	2167, -1000, -1000, 1761, -1000, -1000, -1000, -1000, -1000, -1000,
	-1000, 2439, 2439, 2509, 2509, -14, -1000, -1000, -1000, -1000,
	-1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000, 2439,
	-1000, -1000, -1000, 240, 374, 371, 406, 41, 369, -1000,
	-1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000,
	-1000, -1000, -1000, 11, 2354, -1000, -1000, 2339, -1000, 251,
	226, 390, 44, 2439, 27, 27, 27, 2439, 2439, -1000,
	-1000, 343, 402, 50, 1745, 5, 2439, 2439, 2439, 2439,
	2439, 2439, 2439, 2439, 2439, 2439, 2439, 2439, 2439, 2439,
	2439, 2439, 2500, 210, 2439, 2439, 2439, 151, 1941, 716,
	-1000, -1000, -1000, -58, 337, 363, 347, 286, -1000, 466,
	41, 41, 41, 112, -44, 174, -1000, 41, 2006, 435,
	-1000, -1000, 1630, 189, 2439, 6, 1761, -1000, 389, 28,
	388, 41, 41, -18, -29, -1000, -60, -28, -31, 1761,
	-20, -1000, 149, -1000, -20, -20, 1564, 1504, 56, -1000,
	40, 343, -1000, 360, -1000, -132, -73, -77, -1000, -39,
	1840, 57, 2439, -1000, -1000, -1000, -1000, 918, -1000, -1000,
	2439, 867, -66, -66, -58, -58, -58, 477, 1941, 1887,
	1973, 1973, 1973, 2169, 2169, 2169, 2169, 469, -1000, 2500,
	2439, 2439, 2439, 682, 716, 716, -1000, 185, -1000, -1000,
	248, -1000, 2439, -1000, 233, -1000, 233, -1000, 233, 2439,
	352, 352, 112, 141, -1000, 173, 29, -1000, -1000, -1000,
	40, -1000, 98, 3, 2439, -2, -1000, 189, 2439, -1000,
	2439, 1431, -1000, 228, 222, -1000, 221, -127, -1000, -100,
	-130, -1000, 44, 2439, -1000, 2439, 433, 27, 2439, 2439,
	2439, 432, 431, 27, 27, 361, -1000, 2439, -37, -1000,
	-112, 56, 241, -1000, 192, 174, 24, 29, 29, 57,
	-39, 2439, -39, 580, -30, -1000, 792, -1000, 2254, 2500,
	-6, 2439, 2500, 2500, 2500, 2500, 2500, 2500, 142, 682,
	716, 716, -1000, -1000, -1000, -1000, -1000, 2439, 1761, -1000,
	-1000, -1000, -32, -1000, 738, 190, -1000, 2439, 190, 56,
	82, 56, 24, 24, 350, -1000, 174, -1000, -1000, 38,
	-1000, 1371, -1000, -1000, 1305, 1761, 2439, 41, 41, 41,
	28, 29, 28, -1000, 1761, 1761, -1000, -1000, 1761, 1761,
	1761, -1000, -1000, -26, -26, 152, -1000, 465, -1000, 40,
	1761, 40, 2439, 361, 48, 48, 2439, -1000, -1000, -1000,
	-1000, 112, -64, -1000, -132, -132, -1000, 580, -1000, -1000,
	-1000, -1000, -1000, 1245, 328, -1000, -1000, 2439, 612, -110,
	-110, -62, -62, -62, 459, 2500, 1761, 2439, -1000, -1000,
	-1000, -1000, 153, 153, 2439, 1761, 153, 153, 337, 56,
	337, 337, -34, -1000, -65, -36, -1000, 8, 2439, -1000,
	219, 233, -1000, 2439, 1761, 92, 10, -1000, -1000, -1000,
	158, 427, 2439, 425, -1000, 2439, -37, -1000, 1761, -1000,
	-1000, -132, -101, -104, -1000, 580, -1000, 21, 2439, 174,
	174, -1000, -1000, 543, -1000, 685, 328, -1000, -1000, -1000,
	1840, -1000, 1761, -1000, -1000, 153, 337, 153, 153, 24,
	2439, 24, -1000, -1000, 27, 1761, 352, -8, 1761, -1000,
	155, 2439, -1000, 125, -1000, 1761, -1000, -13, 174, 29,
	29, -1000, -1000, -1000, 1179, 112, 112, -1000, -1000, -1000,
	1119, -1000, -39, 2439, -1000, 153, -1000, -1000, -1000, 1053,
	-1000, -35, -1000, 146, 84, 174, -1000, -1000, -38, -1000,
	1761, 28, 397, -1000, 203, -132, -132, -1000, -1000, -1000,
	-1000, 1761, -1000, -1000, 423, 27, 24, 24, 337, 295,
	214, 179, 2439, -1000, -1000, -1000, 2439, -1000, 173, 174,
	174, -1000, -1000, -1000, -64, -1000, 153, 137, 301, 352,
	61, 463, -1000, 1761, 349, 203, 203, -1000, 230, 136,
	84, 92, 2439, 2439, 2439, -1000, -1000, 141, 56, 384,
	337, -1000, -1000, 1761, 1761, 77, 82, 56, 63, -1000,
	2439, 153, -1000, 293, -1000, 56, -1000, -1000, 254, -1000,
	993, -1000, 134, 279, -1000, 277, -1000, 446, 131, 127,
	56, 377, 362, 63, 2439, 2439, -1000, -1000, -1000,
}
var yyPgo = []int{

	0, 668, 667, 486, 666, 664, 51, 662, 656, 0,
	145, 62, 38, 293, 45, 49, 53, 21, 17, 22,
	654, 647, 646, 637, 48, 281, 635, 634, 633, 47,
	46, 235, 26, 631, 630, 629, 627, 35, 624, 52,
	623, 622, 621, 440, 618, 42, 40, 617, 18, 24,
	115, 36, 616, 32, 14, 217, 615, 6, 614, 39,
	613, 612, 28, 610, 603, 50, 34, 601, 41, 600,
	599, 31, 598, 596, 9, 595, 594, 591, 589, 507,
	588, 582, 579, 578, 574, 569, 568, 567, 561, 560,
	555, 553, 552, 542, 536, 386, 43, 16, 535, 534,
	532, 4, 23, 530, 19, 7, 33, 529, 8, 29,
	526, 524, 25, 11, 523, 520, 3, 2, 5, 27,
	44, 515, 15, 505, 10, 503, 501, 492, 37, 490,
	20, 489,
}
var yyR1 = []int{

	0, 127, 127, 79, 79, 79, 79, 79, 79, 80,
	81, 82, 83, 84, 84, 84, 84, 84, 85, 91,
	91, 91, 37, 38, 38, 38, 38, 38, 38, 38,
	39, 39, 41, 40, 68, 67, 67, 67, 67, 67,
	128, 128, 66, 66, 65, 65, 65, 18, 18, 17,
	17, 16, 44, 44, 43, 42, 42, 42, 42, 129,
	129, 45, 45, 45, 46, 46, 46, 50, 51, 49,
	49, 53, 53, 52, 130, 130, 47, 47, 47, 131,
	131, 54, 55, 55, 56, 15, 15, 14, 57, 57,
	58, 59, 59, 60, 60, 12, 12, 61, 61, 62,
	63, 63, 64, 70, 70, 69, 72, 72, 71, 78,
	78, 77, 77, 74, 74, 73, 76, 76, 75, 86,
	86, 95, 95, 98, 98, 97, 96, 101, 101, 100,
	99, 99, 87, 87, 88, 89, 89, 89, 105, 107,
	107, 106, 112, 112, 111, 103, 103, 102, 102, 19,
	104, 32, 32, 108, 110, 110, 109, 90, 90, 113,
	113, 113, 113, 114, 114, 114, 118, 118, 115, 115,
	115, 116, 117, 92, 92, 119, 120, 120, 121, 121,
	122, 122, 122, 126, 126, 124, 125, 125, 93, 93,
	94, 123, 123, 48, 48, 48, 48, 9, 9, 9,
	9, 9, 9, 9, 9, 9, 9, 9, 9, 9,
	9, 9, 9, 9, 9, 9, 9, 9, 9, 9,
	9, 9, 9, 9, 9, 9, 9, 9, 9, 9,
	9, 9, 9, 9, 9, 9, 10, 10, 10, 10,
	10, 10, 10, 10, 10, 10, 11, 11, 11, 11,
	11, 11, 11, 11, 11, 11, 11, 11, 11, 11,
	1, 1, 1, 1, 1, 1, 1, 2, 2, 3,
	8, 8, 7, 7, 6, 4, 13, 13, 5, 5,
	20, 21, 21, 22, 25, 25, 23, 24, 24, 33,
	33, 33, 34, 26, 26, 27, 27, 27, 30, 30,
	29, 29, 31, 28, 28, 35, 36, 36,
}
var yyR2 = []int{

	0, 1, 1, 1, 1, 1, 1, 1, 1, 2,
	2, 2, 1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 4, 1, 3, 4, 3, 4, 3, 4,
	1, 1, 5, 5, 2, 1, 2, 2, 3, 4,
	1, 1, 1, 3, 1, 3, 2, 0, 1, 1,
	2, 1, 0, 1, 2, 1, 4, 4, 5, 1,
	1, 4, 6, 6, 4, 6, 6, 1, 1, 0,
	2, 0, 1, 4, 0, 1, 0, 1, 2, 0,
	1, 4, 0, 1, 2, 1, 3, 3, 0, 1,
	2, 0, 1, 5, 1, 1, 3, 0, 1, 2,
	0, 1, 2, 0, 1, 3, 1, 3, 2, 0,
	1, 1, 1, 0, 1, 2, 0, 1, 2, 6,
	6, 4, 2, 0, 1, 2, 2, 0, 1, 2,
	1, 2, 6, 6, 7, 8, 7, 7, 2, 1,
	3, 4, 0, 1, 4, 1, 3, 3, 3, 1,
	1, 0, 2, 2, 1, 3, 2, 10, 13, 0,
	6, 6, 6, 0, 6, 6, 0, 6, 2, 3,
	2, 1, 2, 6, 11, 1, 1, 3, 0, 3,
	0, 2, 2, 1, 3, 1, 0, 2, 5, 5,
	6, 0, 3, 1, 3, 3, 4, 1, 3, 3,
	5, 5, 4, 5, 6, 3, 3, 3, 3, 3,
	3, 3, 3, 2, 3, 3, 3, 3, 3, 3,
	3, 5, 6, 3, 4, 3, 4, 3, 4, 3,
	4, 3, 4, 3, 4, 2, 1, 1, 1, 1,
	1, 1, 2, 1, 1, 1, 1, 3, 3, 5,
	5, 4, 5, 6, 3, 3, 3, 3, 3, 3,
	1, 1, 1, 1, 1, 1, 1, 1, 1, 3,
	0, 1, 1, 3, 3, 3, 0, 1, 1, 1,
	3, 1, 1, 3, 4, 5, 2, 0, 2, 4,
	5, 4, 1, 1, 1, 4, 4, 4, 1, 3,
	3, 3, 2, 6, 6, 3, 1, 1,
}
var yyChk = []int{

	-1000, -127, -79, -9, -80, -81, -82, -83, -84, -85,
	-10, 89, 47, 48, 103, 46, -37, -86, -87, -88,
	-89, -90, -91, -1, -2, 159, 122, -5, -33, 85,
	-20, -26, -35, -38, 65, 141, 32, 140, 84, -92,
	-93, -94, 90, 86, 49, 133, 157, 156, 158, -3,
	-4, 161, 162, -34, 18, -27, -28, 163, -39, 26,
	38, 5, 165, 167, 8, 125, 43, 9, 50, -41,
	-40, -43, -68, 53, 121, 186, 167, 181, 85, 182,
	183, 184, 180, 7, 95, 173, 174, 175, 176, 177,
	178, 179, 13, 89, 77, 59, 153, 68, -9, -9,
	-79, -79, -3, -9, -70, 136, 66, 44, -69, 96,
	67, 67, 53, -95, -50, -51, 159, 67, 163, -21,
	-22, -23, -9, -25, 149, -36, -9, -37, 104, 62,
	104, 62, 62, -8, -7, -6, 158, -13, -12, -9,
	-30, -29, -19, 159, -30, -30, -9, -9, -55, -56,
	75, -44, -43, -42, -45, -51, -50, 128, -67, -66,
	36, 4, -128, -65, 109, 40, 182, -9, 159, 160,
	167, -9, -9, -9, -9, -9, -9, -9, -9, -9,
	-9, -9, -9, -9, -9, -9, -9, -11, -10, 13,
	77, 59, 153, -9, -9, -9, 90, 89, 86, 146,
	-74, -73, 78, -39, 4, -39, 4, -39, 4, 16,
	-95, -95, -95, -53, -52, 142, 171, -18, -17, -16,
	10, 159, -95, -13, 36, 182, 42, -25, 149, -24,
	41, -9, 164, 62, -119, 159, 62, -120, -51, -50,
	-120, 166, 170, 171, 168, 170, -31, 170, 119, 59,
	153, -31, -31, 52, 52, -57, -58, 150, -15, -14,
	-16, -55, -47, 64, 74, -49, 186, 171, 171, 170,
	-66, -128, -66, -9, 186, -18, -9, 168, 171, 7,
	186, 167, 181, 85, 182, 183, 184, 180, -11, -9,
	-9, -9, 90, 86, 146, -76, -75, 92, -9, -39,
	-39, -39, -72, -71, -9, -98, -97, 70, -97, -53,
	-105, -108, 123, 139, -130, 104, -51, 159, -16, 144,
	164, -9, 164, -24, -9, -9, 129, 93, 93, 93,
	186, 171, 186, -6, -9, -9, 42, -29, -9, -9,
	-9, 42, 42, -30, -30, -59, -60, 56, -62, 76,
	-9, 170, 173, -57, 69, 88, -129, 138, 51, -131,
	97, -18, -48, 159, -51, -51, -65, -9, -18, 182,
	168, 169, 168, -9, -11, 159, 160, 167, -9, -11,
	-11, -11, -11, -11, -11, 7, -9, 170, -78, -77,
	11, 34, -96, -37, 147, -9, -96, -37, -57, -108,
	-57, -57, -107, -106, -48, -110, -109, -48, 71, -18,
	-45, 163, 164, 129, -9, -120, -120, -120, -119, -51,
	-119, -32, 149, -32, -68, 16, -15, -14, -9, -59,
	-46, -51, -50, 128, -46, -9, -53, 186, 167, -49,
	-49, -18, 168, -9, 168, 171, -11, -71, -101, -100,
	114, -101, -9, -101, -101, -74, -57, -74, -74, 170,
	173, 170, -112, -111, 52, -9, 93, -37, -9, -122,
	144, 163, -123, 112, 42, -9, 42, -12, -49, 171,
	171, -18, 159, 160, -9, -18, -18, 168, 169, 168,
	-9, -99, -66, -128, -101, -74, -101, -101, -106, -9,
	-109, -103, -102, -19, -97, 164, 148, 79, -126, -124,
	-9, 130, -61, -62, -18, -51, -51, 168, -53, -53,
	168, -9, -101, -112, -32, 170, 59, 153, -113, 149,
	-17, 164, 170, -119, -63, -64, 57, -54, 93, -49,
	-49, 42, -102, -104, -48, -104, -74, 82, 89, 93,
	-121, 99, -124, -9, -130, -18, -18, -101, 129, 82,
	-97, -125, 150, 16, 71, -54, -54, 140, 32, 129,
	-113, -122, -124, -9, -9, -115, -105, -108, -116, -57,
	65, -74, -114, 149, -57, -108, -57, -118, 149, -117,
	-9, -101, 82, 89, -57, 89, -57, 129, 82, 82,
	32, 129, 129, -116, 65, 65, -118, -117, -117,
}
var yyDef = []int{

	0, -2, 1, 2, 3, 4, 5, 6, 7, 8,
	197, 0, 0, 0, 0, 0, 12, 13, 14, 15,
	16, 17, 18, 236, 237, -2, 239, 240, 241, 0,
	243, 244, 245, 103, 0, 0, 0, 0, 0, 19,
	20, 21, 260, 261, 262, 263, 264, 265, 266, 267,
	268, 278, 279, 0, 0, 293, 294, 0, 23, 0,
	0, 0, 270, 276, 0, 0, 0, 0, 0, 30,
	31, 82, 52, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 213, 235,
	9, 10, 11, 242, 113, 0, 0, 0, 104, 0,
	0, 0, 0, 71, 0, 47, -2, 0, 276, 0,
	281, 282, 0, 287, 0, 0, 306, 307, 0, 0,
	0, 0, 0, 0, 271, 272, 0, 0, 277, 95,
	0, 298, 0, 149, 0, 0, 0, 0, 88, 83,
	0, 82, 53, -2, 55, 69, 0, 0, 34, 35,
	0, 0, 0, 42, 40, 41, 44, 47, 198, 199,
	0, 0, 205, 206, 207, 208, 209, 210, 211, 212,
	-2, -2, -2, -2, -2, -2, -2, 0, 246, 0,
	0, 0, 0, -2, -2, -2, 229, 0, 231, 233,
	116, 114, 0, 24, 0, 26, 0, 28, 0, 0,
	123, 0, 71, 0, 72, 74, 0, 122, 48, 49,
	0, 51, 0, 0, 0, 0, 280, 287, 0, 286,
	0, 0, 305, 0, 0, 175, 0, 0, 176, 0,
	0, 269, 0, 0, 275, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 91, 89, 0, 84, 85,
	0, 88, 0, 77, 79, 47, 0, 0, 0, 0,
	36, 0, 37, 47, 0, 46, 0, 202, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, -2,
	-2, -2, 230, 232, 234, 22, 117, 0, 115, 25,
	27, 29, 105, 106, 109, 0, 124, 0, 0, 88,
	88, 88, 0, 0, 0, 75, 47, 68, 50, 0,
	289, 0, 291, 283, 0, 288, 0, 0, 0, 0,
	0, 0, 0, 273, 274, 96, 295, 299, 302, 300,
	301, 296, 297, 151, 151, 0, 92, 0, 94, 0,
	90, 0, 0, 91, 0, 0, 0, 59, 60, 78,
	80, 71, 70, 193, 69, 69, 43, 47, 38, 45,
	200, 201, 203, 0, 221, 247, 248, 0, 0, 254,
	255, 256, 257, 258, 259, 0, 118, 0, 108, 110,
	111, 112, 127, 127, 0, 125, 127, 127, 113, 88,
	113, 113, 138, 139, 0, 153, 154, 142, 0, 121,
	0, 0, 290, 0, 284, 180, 0, 188, 189, 177,
	191, 0, 0, 0, 32, 0, 99, 86, 87, 33,
	56, 69, 0, 0, 57, 47, 61, 0, 0, 47,
	47, 39, 204, 0, 251, 0, 222, 107, 119, 128,
	0, 120, 126, 132, 133, 127, 113, 127, 127, 0,
	0, 0, 156, 143, 0, 73, 0, 0, 285, 173,
	0, 0, 190, 0, 303, 152, 304, 97, 47, 0,
	0, 58, 194, 195, 0, 71, 71, 249, 250, 252,
	0, 129, 130, 0, 134, 127, 136, 137, 140, 142,
	155, 151, 145, 0, 159, 0, 181, 182, 0, 183,
	185, 0, 100, 98, 0, 69, 69, 196, 62, 63,
	253, 131, 135, 141, 0, 0, 0, 0, 113, 0,
	0, 178, 0, 192, 93, 101, 0, 64, 74, 47,
	47, 144, 146, 147, 150, 148, 127, 0, 0, 0,
	186, 0, 184, 102, 0, 0, 0, 157, 0, 0,
	159, 180, 0, 0, 0, 65, 66, 0, 88, 0,
	113, 174, 187, 179, 81, 163, 88, 88, 166, 171,
	0, 127, 160, 0, 168, 88, 170, 161, 0, 162,
	88, 158, 0, 0, 169, 0, 172, 0, 0, 0,
	88, 0, 0, 166, 0, 0, 164, 165, 167,
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
		yyVAL.statement = yyS[yypt-0].statement
	case 9:
		//line n1ql.y:373
		{
			yyVAL.statement = algebra.NewExplain(yyS[yypt-0].statement)
		}
	case 10:
		//line n1ql.y:380
		{
			yyVAL.statement = algebra.NewPrepare(yyS[yypt-0].statement)
		}
	case 11:
		//line n1ql.y:387
		{
			yyVAL.statement = algebra.NewExecute(yyS[yypt-0].expr)
		}
	case 12:
		//line n1ql.y:394
		{
			yyVAL.statement = yyS[yypt-0].fullselect
		}
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
		yyVAL.statement = yyS[yypt-0].statement
	case 21:
		yyVAL.statement = yyS[yypt-0].statement
	case 22:
		//line n1ql.y:425
		{
			yyVAL.fullselect = algebra.NewSelect(yyS[yypt-3].subresult, yyS[yypt-2].order, yyS[yypt-0].expr, yyS[yypt-1].expr) /* OFFSET precedes LIMIT */
		}
	case 23:
		//line n1ql.y:431
		{
			yyVAL.subresult = yyS[yypt-0].subselect
		}
	case 24:
		//line n1ql.y:436
		{
			yyVAL.subresult = algebra.NewUnion(yyS[yypt-2].subresult, yyS[yypt-0].subselect)
		}
	case 25:
		//line n1ql.y:441
		{
			yyVAL.subresult = algebra.NewUnionAll(yyS[yypt-3].subresult, yyS[yypt-0].subselect)
		}
	case 26:
		//line n1ql.y:446
		{
			yyVAL.subresult = algebra.NewIntersect(yyS[yypt-2].subresult, yyS[yypt-0].subselect)
		}
	case 27:
		//line n1ql.y:451
		{
			yyVAL.subresult = algebra.NewIntersectAll(yyS[yypt-3].subresult, yyS[yypt-0].subselect)
		}
	case 28:
		//line n1ql.y:456
		{
			yyVAL.subresult = algebra.NewExcept(yyS[yypt-2].subresult, yyS[yypt-0].subselect)
		}
	case 29:
		//line n1ql.y:461
		{
			yyVAL.subresult = algebra.NewExceptAll(yyS[yypt-3].subresult, yyS[yypt-0].subselect)
		}
	case 30:
		yyVAL.subselect = yyS[yypt-0].subselect
	case 31:
		yyVAL.subselect = yyS[yypt-0].subselect
	case 32:
		//line n1ql.y:474
		{
			yyVAL.subselect = algebra.NewSubselect(yyS[yypt-4].fromTerm, yyS[yypt-3].bindings, yyS[yypt-2].expr, yyS[yypt-1].group, yyS[yypt-0].projection)
		}
	case 33:
		//line n1ql.y:481
		{
			yyVAL.subselect = algebra.NewSubselect(yyS[yypt-3].fromTerm, yyS[yypt-2].bindings, yyS[yypt-1].expr, yyS[yypt-0].group, yyS[yypt-4].projection)
		}
	case 34:
		//line n1ql.y:496
		{
			yyVAL.projection = yyS[yypt-0].projection
		}
	case 35:
		//line n1ql.y:503
		{
			yyVAL.projection = algebra.NewProjection(false, yyS[yypt-0].resultTerms)
		}
	case 36:
		//line n1ql.y:508
		{
			yyVAL.projection = algebra.NewProjection(true, yyS[yypt-0].resultTerms)
		}
	case 37:
		//line n1ql.y:513
		{
			yyVAL.projection = algebra.NewProjection(false, yyS[yypt-0].resultTerms)
		}
	case 38:
		//line n1ql.y:518
		{
			yyVAL.projection = algebra.NewRawProjection(false, yyS[yypt-1].expr, yyS[yypt-0].s)
		}
	case 39:
		//line n1ql.y:523
		{
			yyVAL.projection = algebra.NewRawProjection(true, yyS[yypt-1].expr, yyS[yypt-0].s)
		}
	case 42:
		//line n1ql.y:536
		{
			yyVAL.resultTerms = algebra.ResultTerms{yyS[yypt-0].resultTerm}
		}
	case 43:
		//line n1ql.y:541
		{
			yyVAL.resultTerms = append(yyS[yypt-2].resultTerms, yyS[yypt-0].resultTerm)
		}
	case 44:
		//line n1ql.y:548
		{
			yyVAL.resultTerm = algebra.NewResultTerm(nil, true, "")
		}
	case 45:
		//line n1ql.y:553
		{
			yyVAL.resultTerm = algebra.NewResultTerm(yyS[yypt-2].expr, true, "")
		}
	case 46:
		//line n1ql.y:558
		{
			yyVAL.resultTerm = algebra.NewResultTerm(yyS[yypt-1].expr, false, yyS[yypt-0].s)
		}
	case 47:
		//line n1ql.y:565
		{
			yyVAL.s = ""
		}
	case 48:
		yyVAL.s = yyS[yypt-0].s
	case 49:
		yyVAL.s = yyS[yypt-0].s
	case 50:
		//line n1ql.y:576
		{
			yyVAL.s = yyS[yypt-0].s
		}
	case 51:
		yyVAL.s = yyS[yypt-0].s
	case 52:
		//line n1ql.y:594
		{
			yyVAL.fromTerm = nil
		}
	case 53:
		yyVAL.fromTerm = yyS[yypt-0].fromTerm
	case 54:
		//line n1ql.y:603
		{
			yyVAL.fromTerm = yyS[yypt-0].fromTerm
		}
	case 55:
		//line n1ql.y:610
		{
			yyVAL.fromTerm = yyS[yypt-0].keyspaceTerm
		}
	case 56:
		//line n1ql.y:615
		{
			yyVAL.fromTerm = algebra.NewJoin(yyS[yypt-3].fromTerm, yyS[yypt-2].b, yyS[yypt-0].keyspaceTerm)
		}
	case 57:
		//line n1ql.y:620
		{
			yyVAL.fromTerm = algebra.NewNest(yyS[yypt-3].fromTerm, yyS[yypt-2].b, yyS[yypt-0].keyspaceTerm)
		}
	case 58:
		//line n1ql.y:625
		{
			yyVAL.fromTerm = algebra.NewUnnest(yyS[yypt-4].fromTerm, yyS[yypt-3].b, yyS[yypt-1].expr, yyS[yypt-0].s)
		}
	case 61:
		//line n1ql.y:638
		{
			yyVAL.keyspaceTerm = algebra.NewKeyspaceTerm("", yyS[yypt-3].s, yyS[yypt-2].path, yyS[yypt-1].s, yyS[yypt-0].expr)
		}
	case 62:
		//line n1ql.y:643
		{
			yyVAL.keyspaceTerm = algebra.NewKeyspaceTerm(yyS[yypt-5].s, yyS[yypt-3].s, yyS[yypt-2].path, yyS[yypt-1].s, yyS[yypt-0].expr)
		}
	case 63:
		//line n1ql.y:648
		{
			yyVAL.keyspaceTerm = algebra.NewKeyspaceTerm("#system", yyS[yypt-3].s, yyS[yypt-2].path, yyS[yypt-1].s, yyS[yypt-0].expr)
		}
	case 64:
		//line n1ql.y:655
		{
			yyVAL.keyspaceTerm = algebra.NewKeyspaceTerm("", yyS[yypt-3].s, yyS[yypt-2].path, yyS[yypt-1].s, yyS[yypt-0].expr)
		}
	case 65:
		//line n1ql.y:660
		{
			yyVAL.keyspaceTerm = algebra.NewKeyspaceTerm(yyS[yypt-5].s, yyS[yypt-3].s, yyS[yypt-2].path, yyS[yypt-1].s, yyS[yypt-0].expr)
		}
	case 66:
		//line n1ql.y:665
		{
			yyVAL.keyspaceTerm = algebra.NewKeyspaceTerm("#system", yyS[yypt-3].s, yyS[yypt-2].path, yyS[yypt-1].s, yyS[yypt-0].expr)
		}
	case 67:
		yyVAL.s = yyS[yypt-0].s
	case 68:
		yyVAL.s = yyS[yypt-0].s
	case 69:
		//line n1ql.y:680
		{
			yyVAL.path = nil
		}
	case 70:
		//line n1ql.y:685
		{
			yyVAL.path = yyS[yypt-0].path
		}
	case 71:
		//line n1ql.y:692
		{
			yyVAL.expr = nil
		}
	case 72:
		yyVAL.expr = yyS[yypt-0].expr
	case 73:
		//line n1ql.y:701
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 74:
		//line n1ql.y:708
		{
		}
	case 76:
		//line n1ql.y:716
		{
			yyVAL.b = false
		}
	case 77:
		//line n1ql.y:721
		{
			yyVAL.b = false
		}
	case 78:
		//line n1ql.y:726
		{
			yyVAL.b = true
		}
	case 81:
		//line n1ql.y:739
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 82:
		//line n1ql.y:753
		{
			yyVAL.bindings = nil
		}
	case 83:
		yyVAL.bindings = yyS[yypt-0].bindings
	case 84:
		//line n1ql.y:762
		{
			yyVAL.bindings = yyS[yypt-0].bindings
		}
	case 85:
		//line n1ql.y:769
		{
			yyVAL.bindings = expression.Bindings{yyS[yypt-0].binding}
		}
	case 86:
		//line n1ql.y:774
		{
			yyVAL.bindings = append(yyS[yypt-2].bindings, yyS[yypt-0].binding)
		}
	case 87:
		//line n1ql.y:781
		{
			yyVAL.binding = expression.NewBinding(yyS[yypt-2].s, yyS[yypt-0].expr)
		}
	case 88:
		//line n1ql.y:795
		{
			yyVAL.expr = nil
		}
	case 89:
		yyVAL.expr = yyS[yypt-0].expr
	case 90:
		//line n1ql.y:804
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 91:
		//line n1ql.y:818
		{
			yyVAL.group = nil
		}
	case 92:
		yyVAL.group = yyS[yypt-0].group
	case 93:
		//line n1ql.y:827
		{
			yyVAL.group = algebra.NewGroup(yyS[yypt-2].exprs, yyS[yypt-1].bindings, yyS[yypt-0].expr)
		}
	case 94:
		//line n1ql.y:832
		{
			yyVAL.group = algebra.NewGroup(nil, yyS[yypt-0].bindings, nil)
		}
	case 95:
		//line n1ql.y:839
		{
			yyVAL.exprs = expression.Expressions{yyS[yypt-0].expr}
		}
	case 96:
		//line n1ql.y:844
		{
			yyVAL.exprs = append(yyS[yypt-2].exprs, yyS[yypt-0].expr)
		}
	case 97:
		//line n1ql.y:851
		{
			yyVAL.bindings = nil
		}
	case 98:
		yyVAL.bindings = yyS[yypt-0].bindings
	case 99:
		//line n1ql.y:860
		{
			yyVAL.bindings = yyS[yypt-0].bindings
		}
	case 100:
		//line n1ql.y:867
		{
			yyVAL.expr = nil
		}
	case 101:
		yyVAL.expr = yyS[yypt-0].expr
	case 102:
		//line n1ql.y:876
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 103:
		//line n1ql.y:890
		{
			yyVAL.order = nil
		}
	case 104:
		yyVAL.order = yyS[yypt-0].order
	case 105:
		//line n1ql.y:899
		{
			yyVAL.order = algebra.NewOrder(yyS[yypt-0].sortTerms)
		}
	case 106:
		//line n1ql.y:906
		{
			yyVAL.sortTerms = algebra.SortTerms{yyS[yypt-0].sortTerm}
		}
	case 107:
		//line n1ql.y:911
		{
			yyVAL.sortTerms = append(yyS[yypt-2].sortTerms, yyS[yypt-0].sortTerm)
		}
	case 108:
		//line n1ql.y:918
		{
			yyVAL.sortTerm = algebra.NewSortTerm(yyS[yypt-1].expr, yyS[yypt-0].b)
		}
	case 109:
		//line n1ql.y:925
		{
			yyVAL.b = false
		}
	case 110:
		yyVAL.b = yyS[yypt-0].b
	case 111:
		//line n1ql.y:934
		{
			yyVAL.b = false
		}
	case 112:
		//line n1ql.y:939
		{
			yyVAL.b = true
		}
	case 113:
		//line n1ql.y:953
		{
			yyVAL.expr = nil
		}
	case 114:
		yyVAL.expr = yyS[yypt-0].expr
	case 115:
		//line n1ql.y:962
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 116:
		//line n1ql.y:976
		{
			yyVAL.expr = nil
		}
	case 117:
		yyVAL.expr = yyS[yypt-0].expr
	case 118:
		//line n1ql.y:985
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 119:
		//line n1ql.y:999
		{
			yyVAL.statement = algebra.NewInsertValues(yyS[yypt-3].keyspaceRef, yyS[yypt-2].expr, yyS[yypt-1].expr, yyS[yypt-0].projection)
		}
	case 120:
		//line n1ql.y:1004
		{
			yyVAL.statement = algebra.NewInsertSelect(yyS[yypt-3].keyspaceRef, yyS[yypt-2].expr, yyS[yypt-1].fullselect, yyS[yypt-0].projection)
		}
	case 121:
		//line n1ql.y:1011
		{
			yyVAL.keyspaceRef = algebra.NewKeyspaceRef(yyS[yypt-3].s, yyS[yypt-1].s, yyS[yypt-0].s)
		}
	case 122:
		//line n1ql.y:1016
		{
			yyVAL.keyspaceRef = algebra.NewKeyspaceRef("", yyS[yypt-1].s, yyS[yypt-0].s)
		}
	case 123:
		//line n1ql.y:1023
		{
			yyVAL.expr = nil
		}
	case 124:
		yyVAL.expr = yyS[yypt-0].expr
	case 125:
		//line n1ql.y:1032
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 126:
		//line n1ql.y:1039
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 127:
		//line n1ql.y:1046
		{
			yyVAL.projection = nil
		}
	case 128:
		yyVAL.projection = yyS[yypt-0].projection
	case 129:
		//line n1ql.y:1055
		{
			yyVAL.projection = yyS[yypt-0].projection
		}
	case 130:
		//line n1ql.y:1062
		{
			yyVAL.projection = algebra.NewProjection(false, yyS[yypt-0].resultTerms)
		}
	case 131:
		//line n1ql.y:1067
		{
			yyVAL.projection = algebra.NewRawProjection(false, yyS[yypt-0].expr, "")
		}
	case 132:
		//line n1ql.y:1081
		{
			yyVAL.statement = algebra.NewUpsertValues(yyS[yypt-3].keyspaceRef, yyS[yypt-2].expr, yyS[yypt-1].expr, yyS[yypt-0].projection)
		}
	case 133:
		//line n1ql.y:1086
		{
			yyVAL.statement = algebra.NewUpsertSelect(yyS[yypt-3].keyspaceRef, yyS[yypt-2].expr, yyS[yypt-1].fullselect, yyS[yypt-0].projection)
		}
	case 134:
		//line n1ql.y:1100
		{
			yyVAL.statement = algebra.NewDelete(yyS[yypt-4].keyspaceRef, yyS[yypt-3].expr, yyS[yypt-2].expr, yyS[yypt-1].expr, yyS[yypt-0].projection)
		}
	case 135:
		//line n1ql.y:1114
		{
			yyVAL.statement = algebra.NewUpdate(yyS[yypt-6].keyspaceRef, yyS[yypt-5].expr, yyS[yypt-4].set, yyS[yypt-3].unset, yyS[yypt-2].expr, yyS[yypt-1].expr, yyS[yypt-0].projection)
		}
	case 136:
		//line n1ql.y:1119
		{
			yyVAL.statement = algebra.NewUpdate(yyS[yypt-5].keyspaceRef, yyS[yypt-4].expr, yyS[yypt-3].set, nil, yyS[yypt-2].expr, yyS[yypt-1].expr, yyS[yypt-0].projection)
		}
	case 137:
		//line n1ql.y:1124
		{
			yyVAL.statement = algebra.NewUpdate(yyS[yypt-5].keyspaceRef, yyS[yypt-4].expr, nil, yyS[yypt-3].unset, yyS[yypt-2].expr, yyS[yypt-1].expr, yyS[yypt-0].projection)
		}
	case 138:
		//line n1ql.y:1131
		{
			yyVAL.set = algebra.NewSet(yyS[yypt-0].setTerms)
		}
	case 139:
		//line n1ql.y:1138
		{
			yyVAL.setTerms = algebra.SetTerms{yyS[yypt-0].setTerm}
		}
	case 140:
		//line n1ql.y:1143
		{
			yyVAL.setTerms = append(yyS[yypt-2].setTerms, yyS[yypt-0].setTerm)
		}
	case 141:
		//line n1ql.y:1150
		{
			yyVAL.setTerm = algebra.NewSetTerm(yyS[yypt-3].path, yyS[yypt-1].expr, yyS[yypt-0].updateFor)
		}
	case 142:
		//line n1ql.y:1157
		{
			yyVAL.updateFor = nil
		}
	case 143:
		yyVAL.updateFor = yyS[yypt-0].updateFor
	case 144:
		//line n1ql.y:1166
		{
			yyVAL.updateFor = algebra.NewUpdateFor(yyS[yypt-2].bindings, yyS[yypt-1].expr)
		}
	case 145:
		//line n1ql.y:1173
		{
			yyVAL.bindings = expression.Bindings{yyS[yypt-0].binding}
		}
	case 146:
		//line n1ql.y:1178
		{
			yyVAL.bindings = append(yyS[yypt-2].bindings, yyS[yypt-0].binding)
		}
	case 147:
		//line n1ql.y:1185
		{
			yyVAL.binding = expression.NewBinding(yyS[yypt-2].s, yyS[yypt-0].expr)
		}
	case 148:
		//line n1ql.y:1190
		{
			yyVAL.binding = expression.NewDescendantBinding(yyS[yypt-2].s, yyS[yypt-0].expr)
		}
	case 149:
		yyVAL.s = yyS[yypt-0].s
	case 150:
		//line n1ql.y:1201
		{
			yyVAL.expr = yyS[yypt-0].path
		}
	case 151:
		//line n1ql.y:1208
		{
			yyVAL.expr = nil
		}
	case 152:
		//line n1ql.y:1213
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 153:
		//line n1ql.y:1220
		{
			yyVAL.unset = algebra.NewUnset(yyS[yypt-0].unsetTerms)
		}
	case 154:
		//line n1ql.y:1227
		{
			yyVAL.unsetTerms = algebra.UnsetTerms{yyS[yypt-0].unsetTerm}
		}
	case 155:
		//line n1ql.y:1232
		{
			yyVAL.unsetTerms = append(yyS[yypt-2].unsetTerms, yyS[yypt-0].unsetTerm)
		}
	case 156:
		//line n1ql.y:1239
		{
			yyVAL.unsetTerm = algebra.NewUnsetTerm(yyS[yypt-1].path, yyS[yypt-0].updateFor)
		}
	case 157:
		//line n1ql.y:1253
		{
			source := algebra.NewMergeSourceFrom(yyS[yypt-5].keyspaceTerm, "")
			yyVAL.statement = algebra.NewMerge(yyS[yypt-7].keyspaceRef, source, yyS[yypt-3].expr, yyS[yypt-2].mergeActions, yyS[yypt-1].expr, yyS[yypt-0].projection)
		}
	case 158:
		//line n1ql.y:1259
		{
			source := algebra.NewMergeSourceSelect(yyS[yypt-7].fullselect, yyS[yypt-5].s)
			yyVAL.statement = algebra.NewMerge(yyS[yypt-10].keyspaceRef, source, yyS[yypt-3].expr, yyS[yypt-2].mergeActions, yyS[yypt-1].expr, yyS[yypt-0].projection)
		}
	case 159:
		//line n1ql.y:1267
		{
			yyVAL.mergeActions = algebra.NewMergeActions(nil, nil, nil)
		}
	case 160:
		//line n1ql.y:1272
		{
			yyVAL.mergeActions = algebra.NewMergeActions(yyS[yypt-1].mergeUpdate, yyS[yypt-0].mergeActions.Delete(), yyS[yypt-0].mergeActions.Insert())
		}
	case 161:
		//line n1ql.y:1277
		{
			yyVAL.mergeActions = algebra.NewMergeActions(nil, yyS[yypt-1].mergeDelete, yyS[yypt-0].mergeInsert)
		}
	case 162:
		//line n1ql.y:1282
		{
			yyVAL.mergeActions = algebra.NewMergeActions(nil, nil, yyS[yypt-0].mergeInsert)
		}
	case 163:
		//line n1ql.y:1289
		{
			yyVAL.mergeActions = algebra.NewMergeActions(nil, nil, nil)
		}
	case 164:
		//line n1ql.y:1294
		{
			yyVAL.mergeActions = algebra.NewMergeActions(nil, yyS[yypt-1].mergeDelete, yyS[yypt-0].mergeInsert)
		}
	case 165:
		//line n1ql.y:1299
		{
			yyVAL.mergeActions = algebra.NewMergeActions(nil, nil, yyS[yypt-0].mergeInsert)
		}
	case 166:
		//line n1ql.y:1306
		{
			yyVAL.mergeInsert = nil
		}
	case 167:
		//line n1ql.y:1311
		{
			yyVAL.mergeInsert = yyS[yypt-0].mergeInsert
		}
	case 168:
		//line n1ql.y:1318
		{
			yyVAL.mergeUpdate = algebra.NewMergeUpdate(yyS[yypt-1].set, nil, yyS[yypt-0].expr)
		}
	case 169:
		//line n1ql.y:1323
		{
			yyVAL.mergeUpdate = algebra.NewMergeUpdate(yyS[yypt-2].set, yyS[yypt-1].unset, yyS[yypt-0].expr)
		}
	case 170:
		//line n1ql.y:1328
		{
			yyVAL.mergeUpdate = algebra.NewMergeUpdate(nil, yyS[yypt-1].unset, yyS[yypt-0].expr)
		}
	case 171:
		//line n1ql.y:1335
		{
			yyVAL.mergeDelete = algebra.NewMergeDelete(yyS[yypt-0].expr)
		}
	case 172:
		//line n1ql.y:1342
		{
			yyVAL.mergeInsert = algebra.NewMergeInsert(yyS[yypt-1].expr, yyS[yypt-0].expr)
		}
	case 173:
		//line n1ql.y:1356
		{
			yyVAL.statement = algebra.NewCreatePrimaryIndex(yyS[yypt-1].keyspaceRef, yyS[yypt-0].indexType)
		}
	case 174:
		//line n1ql.y:1361
		{
			yyVAL.statement = algebra.NewCreateIndex(yyS[yypt-8].s, yyS[yypt-6].keyspaceRef, yyS[yypt-4].exprs, yyS[yypt-2].expr, yyS[yypt-1].expr, yyS[yypt-0].indexType)
		}
	case 175:
		yyVAL.s = yyS[yypt-0].s
	case 176:
		//line n1ql.y:1372
		{
			yyVAL.keyspaceRef = algebra.NewKeyspaceRef("", yyS[yypt-0].s, "")
		}
	case 177:
		//line n1ql.y:1377
		{
			yyVAL.keyspaceRef = algebra.NewKeyspaceRef(yyS[yypt-2].s, yyS[yypt-0].s, "")
		}
	case 178:
		//line n1ql.y:1384
		{
			yyVAL.expr = nil
		}
	case 179:
		//line n1ql.y:1389
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 180:
		//line n1ql.y:1396
		{
			yyVAL.indexType = datastore.VIEW
		}
	case 181:
		//line n1ql.y:1401
		{
			yyVAL.indexType = datastore.VIEW
		}
	case 182:
		//line n1ql.y:1406
		{
			yyVAL.indexType = datastore.LSM
		}
	case 183:
		//line n1ql.y:1413
		{
			yyVAL.exprs = expression.Expressions{yyS[yypt-0].expr}
		}
	case 184:
		//line n1ql.y:1418
		{
			yyVAL.exprs = append(yyS[yypt-2].exprs, yyS[yypt-0].expr)
		}
	case 185:
		//line n1ql.y:1425
		{
			e := yyS[yypt-0].expr
			if !e.Indexable() {
				yylex.Error(fmt.Sprintf("Expression not indexable."))
			}

			yyVAL.expr = e
		}
	case 186:
		//line n1ql.y:1436
		{
			yyVAL.expr = nil
		}
	case 187:
		//line n1ql.y:1441
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 188:
		//line n1ql.y:1455
		{
			yyVAL.statement = algebra.NewDropIndex(yyS[yypt-0].keyspaceRef, "#primary")
		}
	case 189:
		//line n1ql.y:1460
		{
			yyVAL.statement = algebra.NewDropIndex(yyS[yypt-2].keyspaceRef, yyS[yypt-0].s)
		}
	case 190:
		//line n1ql.y:1473
		{
			yyVAL.statement = algebra.NewAlterIndex(yyS[yypt-3].keyspaceRef, yyS[yypt-1].s, yyS[yypt-0].s)
		}
	case 191:
		//line n1ql.y:1479
		{
			yyVAL.s = ""
		}
	case 192:
		//line n1ql.y:1484
		{
			yyVAL.s = yyS[yypt-0].s
		}
	case 193:
		//line n1ql.y:1498
		{
			yyVAL.path = expression.NewIdentifier(yyS[yypt-0].s)
		}
	case 194:
		//line n1ql.y:1503
		{
			yyVAL.path = expression.NewField(yyS[yypt-2].path, expression.NewFieldName(yyS[yypt-0].s))
		}
	case 195:
		//line n1ql.y:1508
		{
			field := expression.NewField(yyS[yypt-2].path, expression.NewFieldName(yyS[yypt-0].s))
			field.SetCaseInsensitive(true)
			yyVAL.path = field
		}
	case 196:
		//line n1ql.y:1515
		{
			yyVAL.path = expression.NewElement(yyS[yypt-3].path, yyS[yypt-1].expr)
		}
	case 197:
		yyVAL.expr = yyS[yypt-0].expr
	case 198:
		//line n1ql.y:1532
		{
			yyVAL.expr = expression.NewField(yyS[yypt-2].expr, expression.NewFieldName(yyS[yypt-0].s))
		}
	case 199:
		//line n1ql.y:1537
		{
			field := expression.NewField(yyS[yypt-2].expr, expression.NewFieldName(yyS[yypt-0].s))
			field.SetCaseInsensitive(true)
			yyVAL.expr = field
		}
	case 200:
		//line n1ql.y:1544
		{
			yyVAL.expr = expression.NewField(yyS[yypt-4].expr, yyS[yypt-1].expr)
		}
	case 201:
		//line n1ql.y:1549
		{
			field := expression.NewField(yyS[yypt-4].expr, yyS[yypt-1].expr)
			field.SetCaseInsensitive(true)
			yyVAL.expr = field
		}
	case 202:
		//line n1ql.y:1556
		{
			yyVAL.expr = expression.NewElement(yyS[yypt-3].expr, yyS[yypt-1].expr)
		}
	case 203:
		//line n1ql.y:1561
		{
			yyVAL.expr = expression.NewSlice(yyS[yypt-4].expr, yyS[yypt-2].expr)
		}
	case 204:
		//line n1ql.y:1566
		{
			yyVAL.expr = expression.NewSlice(yyS[yypt-5].expr, yyS[yypt-3].expr, yyS[yypt-1].expr)
		}
	case 205:
		//line n1ql.y:1572
		{
			yyVAL.expr = expression.NewAdd(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 206:
		//line n1ql.y:1577
		{
			yyVAL.expr = expression.NewSub(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 207:
		//line n1ql.y:1582
		{
			yyVAL.expr = expression.NewMult(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 208:
		//line n1ql.y:1587
		{
			yyVAL.expr = expression.NewDiv(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 209:
		//line n1ql.y:1592
		{
			yyVAL.expr = expression.NewMod(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 210:
		//line n1ql.y:1598
		{
			yyVAL.expr = expression.NewConcat(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 211:
		//line n1ql.y:1604
		{
			yyVAL.expr = expression.NewAnd(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 212:
		//line n1ql.y:1609
		{
			yyVAL.expr = expression.NewOr(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 213:
		//line n1ql.y:1614
		{
			yyVAL.expr = expression.NewNot(yyS[yypt-0].expr)
		}
	case 214:
		//line n1ql.y:1620
		{
			yyVAL.expr = expression.NewEq(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 215:
		//line n1ql.y:1625
		{
			yyVAL.expr = expression.NewEq(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 216:
		//line n1ql.y:1630
		{
			yyVAL.expr = expression.NewNE(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 217:
		//line n1ql.y:1635
		{
			yyVAL.expr = expression.NewLT(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 218:
		//line n1ql.y:1640
		{
			yyVAL.expr = expression.NewGT(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 219:
		//line n1ql.y:1645
		{
			yyVAL.expr = expression.NewLE(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 220:
		//line n1ql.y:1650
		{
			yyVAL.expr = expression.NewGE(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 221:
		//line n1ql.y:1655
		{
			yyVAL.expr = expression.NewBetween(yyS[yypt-4].expr, yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 222:
		//line n1ql.y:1660
		{
			yyVAL.expr = expression.NewNotBetween(yyS[yypt-5].expr, yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 223:
		//line n1ql.y:1665
		{
			yyVAL.expr = expression.NewLike(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 224:
		//line n1ql.y:1670
		{
			yyVAL.expr = expression.NewNotLike(yyS[yypt-3].expr, yyS[yypt-0].expr)
		}
	case 225:
		//line n1ql.y:1675
		{
			yyVAL.expr = expression.NewIn(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 226:
		//line n1ql.y:1680
		{
			yyVAL.expr = expression.NewNotIn(yyS[yypt-3].expr, yyS[yypt-0].expr)
		}
	case 227:
		//line n1ql.y:1685
		{
			yyVAL.expr = expression.NewWithin(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 228:
		//line n1ql.y:1690
		{
			yyVAL.expr = expression.NewNotWithin(yyS[yypt-3].expr, yyS[yypt-0].expr)
		}
	case 229:
		//line n1ql.y:1695
		{
			yyVAL.expr = expression.NewIsNull(yyS[yypt-2].expr)
		}
	case 230:
		//line n1ql.y:1700
		{
			yyVAL.expr = expression.NewIsNotNull(yyS[yypt-3].expr)
		}
	case 231:
		//line n1ql.y:1705
		{
			yyVAL.expr = expression.NewIsMissing(yyS[yypt-2].expr)
		}
	case 232:
		//line n1ql.y:1710
		{
			yyVAL.expr = expression.NewIsNotMissing(yyS[yypt-3].expr)
		}
	case 233:
		//line n1ql.y:1715
		{
			yyVAL.expr = expression.NewIsValued(yyS[yypt-2].expr)
		}
	case 234:
		//line n1ql.y:1720
		{
			yyVAL.expr = expression.NewIsNotValued(yyS[yypt-3].expr)
		}
	case 235:
		//line n1ql.y:1725
		{
			yyVAL.expr = expression.NewExists(yyS[yypt-0].expr)
		}
	case 236:
		yyVAL.expr = yyS[yypt-0].expr
	case 237:
		yyVAL.expr = yyS[yypt-0].expr
	case 238:
		//line n1ql.y:1739
		{
			yyVAL.expr = expression.NewIdentifier(yyS[yypt-0].s)
		}
	case 239:
		//line n1ql.y:1745
		{
			yyVAL.expr = expression.NewSelf()
		}
	case 240:
		yyVAL.expr = yyS[yypt-0].expr
	case 241:
		yyVAL.expr = yyS[yypt-0].expr
	case 242:
		//line n1ql.y:1757
		{
			yyVAL.expr = expression.NewNeg(yyS[yypt-0].expr)
		}
	case 243:
		yyVAL.expr = yyS[yypt-0].expr
	case 244:
		yyVAL.expr = yyS[yypt-0].expr
	case 245:
		yyVAL.expr = yyS[yypt-0].expr
	case 246:
		yyVAL.expr = yyS[yypt-0].expr
	case 247:
		//line n1ql.y:1776
		{
			yyVAL.expr = expression.NewField(yyS[yypt-2].expr, expression.NewFieldName(yyS[yypt-0].s))
		}
	case 248:
		//line n1ql.y:1781
		{
			field := expression.NewField(yyS[yypt-2].expr, expression.NewFieldName(yyS[yypt-0].s))
			field.SetCaseInsensitive(true)
			yyVAL.expr = field
		}
	case 249:
		//line n1ql.y:1788
		{
			yyVAL.expr = expression.NewField(yyS[yypt-4].expr, yyS[yypt-1].expr)
		}
	case 250:
		//line n1ql.y:1793
		{
			field := expression.NewField(yyS[yypt-4].expr, yyS[yypt-1].expr)
			field.SetCaseInsensitive(true)
			yyVAL.expr = field
		}
	case 251:
		//line n1ql.y:1800
		{
			yyVAL.expr = expression.NewElement(yyS[yypt-3].expr, yyS[yypt-1].expr)
		}
	case 252:
		//line n1ql.y:1805
		{
			yyVAL.expr = expression.NewSlice(yyS[yypt-4].expr, yyS[yypt-2].expr)
		}
	case 253:
		//line n1ql.y:1810
		{
			yyVAL.expr = expression.NewSlice(yyS[yypt-5].expr, yyS[yypt-3].expr, yyS[yypt-1].expr)
		}
	case 254:
		//line n1ql.y:1816
		{
			yyVAL.expr = expression.NewAdd(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 255:
		//line n1ql.y:1821
		{
			yyVAL.expr = expression.NewSub(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 256:
		//line n1ql.y:1826
		{
			yyVAL.expr = expression.NewMult(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 257:
		//line n1ql.y:1831
		{
			yyVAL.expr = expression.NewDiv(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 258:
		//line n1ql.y:1836
		{
			yyVAL.expr = expression.NewMod(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 259:
		//line n1ql.y:1842
		{
			yyVAL.expr = expression.NewConcat(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 260:
		//line n1ql.y:1856
		{
			yyVAL.expr = expression.NULL_EXPR
		}
	case 261:
		//line n1ql.y:1861
		{
			yyVAL.expr = expression.MISSING_EXPR
		}
	case 262:
		//line n1ql.y:1866
		{
			yyVAL.expr = expression.FALSE_EXPR
		}
	case 263:
		//line n1ql.y:1871
		{
			yyVAL.expr = expression.TRUE_EXPR
		}
	case 264:
		//line n1ql.y:1876
		{
			yyVAL.expr = expression.NewConstant(value.NewValue(yyS[yypt-0].f))
		}
	case 265:
		//line n1ql.y:1881
		{
			yyVAL.expr = expression.NewConstant(value.NewValue(yyS[yypt-0].n))
		}
	case 266:
		//line n1ql.y:1886
		{
			yyVAL.expr = expression.NewConstant(value.NewValue(yyS[yypt-0].s))
		}
	case 267:
		yyVAL.expr = yyS[yypt-0].expr
	case 268:
		yyVAL.expr = yyS[yypt-0].expr
	case 269:
		//line n1ql.y:1906
		{
			yyVAL.expr = expression.NewObjectConstruct(yyS[yypt-1].bindings)
		}
	case 270:
		//line n1ql.y:1913
		{
			yyVAL.bindings = nil
		}
	case 271:
		yyVAL.bindings = yyS[yypt-0].bindings
	case 272:
		//line n1ql.y:1922
		{
			yyVAL.bindings = expression.Bindings{yyS[yypt-0].binding}
		}
	case 273:
		//line n1ql.y:1927
		{
			yyVAL.bindings = append(yyS[yypt-2].bindings, yyS[yypt-0].binding)
		}
	case 274:
		//line n1ql.y:1934
		{
			yyVAL.binding = expression.NewBinding(yyS[yypt-2].s, yyS[yypt-0].expr)
		}
	case 275:
		//line n1ql.y:1941
		{
			yyVAL.expr = expression.NewArrayConstruct(yyS[yypt-1].exprs...)
		}
	case 276:
		//line n1ql.y:1948
		{
			yyVAL.exprs = nil
		}
	case 277:
		yyVAL.exprs = yyS[yypt-0].exprs
	case 278:
		//line n1ql.y:1964
		{
			yyVAL.expr = algebra.NewNamedParameter(yyS[yypt-0].s)
		}
	case 279:
		//line n1ql.y:1969
		{
			yyVAL.expr = algebra.NewPositionalParameter(yyS[yypt-0].n)
		}
	case 280:
		//line n1ql.y:1983
		{
			yyVAL.expr = yyS[yypt-1].expr
		}
	case 281:
		yyVAL.expr = yyS[yypt-0].expr
	case 282:
		yyVAL.expr = yyS[yypt-0].expr
	case 283:
		//line n1ql.y:1996
		{
			yyVAL.expr = expression.NewSimpleCase(yyS[yypt-2].expr, yyS[yypt-1].whenTerms, yyS[yypt-0].expr)
		}
	case 284:
		//line n1ql.y:2003
		{
			yyVAL.whenTerms = expression.WhenTerms{&expression.WhenTerm{yyS[yypt-2].expr, yyS[yypt-0].expr}}
		}
	case 285:
		//line n1ql.y:2008
		{
			yyVAL.whenTerms = append(yyS[yypt-4].whenTerms, &expression.WhenTerm{yyS[yypt-2].expr, yyS[yypt-0].expr})
		}
	case 286:
		//line n1ql.y:2016
		{
			yyVAL.expr = expression.NewSearchedCase(yyS[yypt-1].whenTerms, yyS[yypt-0].expr)
		}
	case 287:
		//line n1ql.y:2023
		{
			yyVAL.expr = nil
		}
	case 288:
		//line n1ql.y:2028
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 289:
		//line n1ql.y:2042
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
	case 290:
		//line n1ql.y:2061
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
	case 291:
		//line n1ql.y:2076
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
	case 292:
		yyVAL.s = yyS[yypt-0].s
	case 293:
		yyVAL.expr = yyS[yypt-0].expr
	case 294:
		yyVAL.expr = yyS[yypt-0].expr
	case 295:
		//line n1ql.y:2110
		{
			yyVAL.expr = expression.NewAny(yyS[yypt-2].bindings, yyS[yypt-1].expr)
		}
	case 296:
		//line n1ql.y:2115
		{
			yyVAL.expr = expression.NewAny(yyS[yypt-2].bindings, yyS[yypt-1].expr)
		}
	case 297:
		//line n1ql.y:2120
		{
			yyVAL.expr = expression.NewEvery(yyS[yypt-2].bindings, yyS[yypt-1].expr)
		}
	case 298:
		//line n1ql.y:2127
		{
			yyVAL.bindings = expression.Bindings{yyS[yypt-0].binding}
		}
	case 299:
		//line n1ql.y:2132
		{
			yyVAL.bindings = append(yyS[yypt-2].bindings, yyS[yypt-0].binding)
		}
	case 300:
		//line n1ql.y:2139
		{
			yyVAL.binding = expression.NewBinding(yyS[yypt-2].s, yyS[yypt-0].expr)
		}
	case 301:
		//line n1ql.y:2144
		{
			yyVAL.binding = expression.NewDescendantBinding(yyS[yypt-2].s, yyS[yypt-0].expr)
		}
	case 302:
		//line n1ql.y:2151
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 303:
		//line n1ql.y:2158
		{
			yyVAL.expr = expression.NewArray(yyS[yypt-4].expr, yyS[yypt-2].bindings, yyS[yypt-1].expr)
		}
	case 304:
		//line n1ql.y:2163
		{
			yyVAL.expr = expression.NewFirst(yyS[yypt-4].expr, yyS[yypt-2].bindings, yyS[yypt-1].expr)
		}
	case 305:
		//line n1ql.y:2177
		{
			yyVAL.expr = yyS[yypt-1].expr
		}
	case 306:
		yyVAL.expr = yyS[yypt-0].expr
	case 307:
		//line n1ql.y:2186
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
