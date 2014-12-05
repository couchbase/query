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
const BINARY = 57356
const BOOLEAN = 57357
const BREAK = 57358
const BUCKET = 57359
const BY = 57360
const CALL = 57361
const CASE = 57362
const CAST = 57363
const CLUSTER = 57364
const COLLATE = 57365
const COLLECTION = 57366
const COMMIT = 57367
const CONNECT = 57368
const CONTINUE = 57369
const CREATE = 57370
const DATABASE = 57371
const DATASET = 57372
const DATASTORE = 57373
const DECLARE = 57374
const DECREMENT = 57375
const DELETE = 57376
const DERIVED = 57377
const DESC = 57378
const DESCRIBE = 57379
const DISTINCT = 57380
const DO = 57381
const DROP = 57382
const EACH = 57383
const ELEMENT = 57384
const ELSE = 57385
const END = 57386
const EVERY = 57387
const EXCEPT = 57388
const EXCLUDE = 57389
const EXECUTE = 57390
const EXISTS = 57391
const EXPLAIN = 57392
const FALSE = 57393
const FIRST = 57394
const FLATTEN = 57395
const FOR = 57396
const FROM = 57397
const FUNCTION = 57398
const GRANT = 57399
const GROUP = 57400
const HAVING = 57401
const IF = 57402
const IN = 57403
const INCLUDE = 57404
const INCREMENT = 57405
const INDEX = 57406
const INLINE = 57407
const INNER = 57408
const INSERT = 57409
const INTERSECT = 57410
const INTO = 57411
const IS = 57412
const JOIN = 57413
const KEY = 57414
const KEYS = 57415
const KEYSPACE = 57416
const LAST = 57417
const LEFT = 57418
const LET = 57419
const LETTING = 57420
const LIKE = 57421
const LIMIT = 57422
const LSM = 57423
const MAP = 57424
const MAPPING = 57425
const MATCHED = 57426
const MATERIALIZED = 57427
const MERGE = 57428
const MINUS = 57429
const MISSING = 57430
const NAMESPACE = 57431
const NEST = 57432
const NOT = 57433
const NULL = 57434
const NUMBER = 57435
const OBJECT = 57436
const OFFSET = 57437
const ON = 57438
const OPTION = 57439
const OR = 57440
const ORDER = 57441
const OUTER = 57442
const OVER = 57443
const PARTITION = 57444
const PASSWORD = 57445
const PATH = 57446
const POOL = 57447
const PREPARE = 57448
const PRIMARY = 57449
const PRIVATE = 57450
const PRIVILEGE = 57451
const PROCEDURE = 57452
const PUBLIC = 57453
const RAW = 57454
const REALM = 57455
const REDUCE = 57456
const RENAME = 57457
const RETURN = 57458
const RETURNING = 57459
const REVOKE = 57460
const RIGHT = 57461
const ROLE = 57462
const ROLLBACK = 57463
const SATISFIES = 57464
const SCHEMA = 57465
const SELECT = 57466
const SELF = 57467
const SET = 57468
const SHOW = 57469
const SOME = 57470
const START = 57471
const STATISTICS = 57472
const STRING = 57473
const SYSTEM = 57474
const THEN = 57475
const TO = 57476
const TRANSACTION = 57477
const TRIGGER = 57478
const TRUE = 57479
const TRUNCATE = 57480
const UNDER = 57481
const UNION = 57482
const UNIQUE = 57483
const UNNEST = 57484
const UNSET = 57485
const UPDATE = 57486
const UPSERT = 57487
const USE = 57488
const USER = 57489
const USING = 57490
const VALUE = 57491
const VALUED = 57492
const VALUES = 57493
const VIEW = 57494
const WHEN = 57495
const WHERE = 57496
const WHILE = 57497
const WITH = 57498
const WITHIN = 57499
const WORK = 57500
const XOR = 57501
const INT = 57502
const IDENTIFIER = 57503
const IDENTIFIER_ICASE = 57504
const NAMED_PARAM = 57505
const POSITIONAL_PARAM = 57506
const LPAREN = 57507
const RPAREN = 57508
const LBRACE = 57509
const RBRACE = 57510
const LBRACKET = 57511
const RBRACKET = 57512
const RBRACKET_ICASE = 57513
const COMMA = 57514
const COLON = 57515
const INTERESECT = 57516
const EQ = 57517
const DEQ = 57518
const NE = 57519
const LT = 57520
const GT = 57521
const LE = 57522
const GE = 57523
const CONCAT = 57524
const PLUS = 57525
const STAR = 57526
const DIV = 57527
const MOD = 57528
const UMINUS = 57529
const DOT = 57530

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
	"BINARY",
	"BOOLEAN",
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
	"NUMBER",
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
	"STRING",
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
	165, 292,
	-2, 238,
	-1, 116,
	173, 67,
	-2, 68,
	-1, 153,
	53, 76,
	71, 76,
	90, 76,
	142, 76,
	-2, 54,
	-1, 180,
	175, 0,
	176, 0,
	177, 0,
	-2, 214,
	-1, 181,
	175, 0,
	176, 0,
	177, 0,
	-2, 215,
	-1, 182,
	175, 0,
	176, 0,
	177, 0,
	-2, 216,
	-1, 183,
	178, 0,
	179, 0,
	180, 0,
	181, 0,
	-2, 217,
	-1, 184,
	178, 0,
	179, 0,
	180, 0,
	181, 0,
	-2, 218,
	-1, 185,
	178, 0,
	179, 0,
	180, 0,
	181, 0,
	-2, 219,
	-1, 186,
	178, 0,
	179, 0,
	180, 0,
	181, 0,
	-2, 220,
	-1, 193,
	79, 0,
	-2, 223,
	-1, 194,
	61, 0,
	157, 0,
	-2, 225,
	-1, 195,
	61, 0,
	157, 0,
	-2, 227,
	-1, 289,
	79, 0,
	-2, 224,
	-1, 290,
	61, 0,
	157, 0,
	-2, 226,
	-1, 291,
	61, 0,
	157, 0,
	-2, 228,
}

const yyNprod = 308
const yyPrivate = 57344

var yyTokenNames []string
var yyStates []string

const yyLast = 2639

var yyAct = []int{

	167, 3, 589, 578, 448, 587, 579, 310, 311, 200,
	509, 528, 98, 99, 537, 469, 306, 217, 404, 543,
	314, 218, 142, 502, 265, 462, 421, 234, 348, 406,
	103, 303, 213, 403, 159, 16, 115, 162, 138, 345,
	430, 72, 154, 392, 237, 259, 140, 141, 229, 258,
	163, 135, 58, 219, 266, 122, 281, 438, 126, 332,
	464, 330, 187, 460, 139, 352, 480, 349, 146, 147,
	479, 284, 285, 286, 331, 280, 437, 171, 172, 173,
	174, 175, 176, 177, 178, 179, 180, 181, 182, 183,
	184, 185, 186, 127, 531, 193, 194, 195, 438, 281,
	532, 248, 76, 97, 268, 267, 269, 76, 168, 169,
	155, 422, 144, 145, 422, 114, 170, 437, 280, 139,
	78, 75, 79, 80, 81, 231, 75, 243, 216, 351,
	525, 369, 461, 247, 459, 387, 245, 242, 283, 244,
	92, 375, 376, 241, 62, 10, 168, 169, 505, 377,
	157, 247, 322, 320, 170, 255, 232, 471, 203, 205,
	207, 245, 118, 273, 482, 483, 363, 526, 238, 238,
	220, 276, 433, 143, 235, 438, 157, 240, 317, 116,
	221, 116, 249, 411, 257, 275, 562, 230, 95, 156,
	588, 289, 290, 291, 437, 270, 272, 97, 271, 583,
	313, 116, 76, 298, 260, 116, 94, 189, 529, 507,
	304, 257, 470, 319, 78, 82, 77, 79, 80, 81,
	281, 75, 215, 148, 246, 321, 568, 511, 308, 324,
	73, 325, 602, 287, 282, 284, 285, 286, 188, 280,
	601, 597, 569, 558, 334, 309, 335, 239, 239, 338,
	339, 340, 288, 316, 136, 191, 73, 299, 350, 300,
	293, 301, 198, 527, 292, 197, 196, 450, 353, 312,
	358, 74, 367, 190, 318, 315, 323, 123, 250, 373,
	506, 473, 378, 361, 96, 362, 313, 208, 354, 551,
	360, 368, 206, 538, 333, 337, 76, 228, 386, 74,
	343, 344, 137, 549, 364, 365, 466, 355, 395, 82,
	77, 79, 80, 81, 329, 75, 398, 400, 401, 399,
	366, 221, 294, 328, 199, 74, 394, 414, 327, 204,
	297, 595, 407, 385, 409, 188, 567, 599, 73, 131,
	129, 393, 374, 73, 397, 379, 380, 381, 382, 383,
	384, 192, 396, 428, 592, 598, 155, 435, 418, 357,
	420, 593, 410, 559, 238, 238, 238, 202, 419, 251,
	252, 423, 415, 416, 417, 261, 150, 283, 443, 564,
	73, 408, 130, 128, 307, 441, 113, 424, 304, 439,
	440, 431, 431, 429, 436, 452, 434, 427, 451, 426,
	227, 453, 454, 260, 347, 260, 456, 74, 455, 465,
	457, 458, 74, 283, 468, 547, 117, 263, 111, 447,
	605, 223, 548, 475, 349, 188, 139, 264, 188, 188,
	188, 188, 188, 188, 71, 156, 110, 604, 580, 484,
	236, 233, 132, 239, 239, 239, 490, 467, 446, 74,
	536, 73, 112, 481, 279, 541, 478, 485, 486, 281,
	494, 499, 496, 497, 477, 600, 495, 107, 476, 474,
	432, 432, 510, 282, 284, 285, 286, 342, 280, 341,
	407, 336, 226, 504, 563, 492, 425, 503, 493, 106,
	209, 500, 359, 498, 521, 281, 514, 210, 211, 212,
	522, 356, 1, 508, 222, 49, 513, 152, 287, 282,
	284, 285, 286, 561, 280, 2, 515, 516, 518, 519,
	109, 102, 472, 550, 575, 523, 582, 530, 524, 100,
	101, 188, 463, 510, 283, 405, 402, 553, 546, 533,
	539, 540, 501, 552, 449, 544, 544, 545, 503, 542,
	491, 557, 305, 41, 40, 39, 83, 555, 556, 554,
	22, 105, 92, 510, 573, 574, 560, 21, 78, 20,
	565, 566, 570, 572, 19, 576, 577, 571, 18, 17,
	581, 590, 9, 584, 586, 585, 591, 8, 83, 7,
	6, 220, 594, 5, 92, 4, 388, 596, 389, 295,
	296, 201, 302, 104, 603, 590, 590, 607, 608, 606,
	95, 108, 158, 535, 534, 512, 281, 346, 256, 97,
	83, 149, 214, 262, 151, 153, 92, 69, 94, 287,
	282, 284, 285, 286, 70, 280, 78, 33, 125, 32,
	93, 53, 95, 28, 56, 55, 31, 84, 121, 120,
	76, 97, 119, 30, 133, 134, 27, 50, 24, 23,
	94, 0, 0, 0, 77, 79, 80, 81, 78, 75,
	0, 0, 93, 0, 95, 0, 0, 0, 0, 84,
	0, 0, 0, 97, 0, 0, 0, 0, 0, 0,
	0, 0, 94, 0, 0, 0, 0, 0, 0, 0,
	78, 0, 0, 0, 93, 0, 96, 0, 0, 0,
	0, 84, 0, 0, 0, 0, 0, 0, 76, 487,
	488, 0, 0, 0, 85, 86, 87, 88, 89, 90,
	91, 82, 77, 79, 80, 81, 0, 75, 96, 0,
	0, 0, 221, 0, 0, 0, 0, 0, 83, 0,
	76, 0, 390, 0, 92, 0, 85, 86, 87, 88,
	89, 90, 91, 82, 77, 79, 80, 81, 0, 75,
	96, 0, 0, 0, 0, 0, 0, 391, 0, 0,
	0, 83, 76, 444, 0, 0, 445, 92, 85, 86,
	87, 88, 89, 90, 91, 82, 77, 79, 80, 81,
	0, 75, 95, 0, 0, 0, 0, 0, 0, 0,
	0, 97, 0, 83, 0, 0, 0, 0, 0, 92,
	94, 0, 0, 0, 0, 0, 0, 0, 78, 0,
	0, 0, 93, 0, 0, 95, 0, 0, 0, 84,
	0, 0, 0, 0, 97, 0, 0, 0, 0, 0,
	0, 0, 0, 94, 0, 0, 0, 0, 0, 0,
	0, 78, 0, 0, 0, 93, 0, 95, 0, 0,
	0, 0, 84, 0, 0, 0, 97, 0, 0, 0,
	0, 0, 0, 0, 0, 94, 0, 0, 0, 0,
	0, 92, 0, 78, 0, 0, 0, 93, 96, 0,
	0, 0, 0, 0, 84, 0, 0, 0, 0, 0,
	76, 0, 0, 0, 0, 0, 85, 86, 87, 88,
	89, 90, 91, 82, 77, 79, 80, 81, 0, 75,
	0, 96, 0, 0, 0, 0, 0, 0, 0, 95,
	0, 0, 0, 76, 370, 371, 0, 0, 97, 85,
	86, 87, 88, 89, 90, 91, 82, 77, 79, 80,
	81, 83, 75, 96, 220, 78, 0, 92, 0, 0,
	0, 0, 0, 0, 0, 76, 277, 61, 0, 278,
	0, 85, 86, 87, 88, 89, 90, 91, 82, 77,
	79, 80, 81, 0, 75, 0, 83, 0, 0, 0,
	59, 0, 92, 0, 0, 0, 36, 0, 0, 0,
	0, 0, 60, 0, 0, 95, 0, 0, 0, 0,
	15, 0, 13, 0, 97, 0, 0, 73, 83, 0,
	0, 0, 0, 94, 92, 96, 0, 0, 0, 34,
	0, 78, 0, 0, 0, 93, 0, 76, 0, 0,
	95, 0, 84, 0, 0, 0, 0, 0, 38, 97,
	82, 77, 79, 80, 81, 0, 75, 0, 94, 0,
	0, 0, 0, 0, 0, 464, 78, 0, 14, 0,
	93, 0, 95, 0, 0, 0, 0, 84, 0, 0,
	0, 97, 0, 0, 0, 0, 74, 0, 0, 0,
	94, 0, 0, 0, 0, 0, 0, 0, 78, 0,
	0, 96, 93, 0, 0, 221, 37, 35, 0, 84,
	0, 0, 0, 76, 0, 0, 0, 0, 0, 85,
	86, 87, 88, 89, 90, 91, 82, 77, 79, 80,
	81, 0, 274, 257, 0, 0, 96, 0, 0, 0,
	0, 0, 0, 0, 83, 0, 0, 0, 76, 0,
	92, 0, 0, 0, 85, 86, 87, 88, 89, 90,
	91, 82, 77, 79, 80, 81, 0, 75, 96, 0,
	0, 0, 0, 0, 0, 0, 83, 0, 0, 0,
	76, 0, 92, 0, 0, 0, 85, 86, 87, 88,
	89, 90, 91, 82, 77, 79, 80, 81, 95, 75,
	0, 0, 0, 0, 0, 0, 0, 97, 83, 0,
	0, 0, 0, 0, 92, 0, 94, 0, 0, 0,
	0, 0, 0, 0, 78, 0, 0, 0, 93, 0,
	95, 0, 0, 0, 0, 84, 0, 0, 0, 97,
	0, 0, 0, 0, 0, 0, 0, 0, 94, 0,
	0, 0, 0, 0, 0, 0, 78, 0, 0, 0,
	93, 0, 95, 0, 0, 0, 0, 84, 0, 0,
	0, 97, 0, 0, 0, 0, 0, 0, 0, 0,
	94, 0, 0, 0, 0, 0, 0, 0, 78, 0,
	0, 0, 93, 0, 96, 0, 0, 0, 0, 84,
	0, 0, 0, 0, 0, 0, 76, 520, 0, 0,
	0, 0, 85, 86, 87, 88, 89, 90, 91, 82,
	77, 79, 80, 81, 0, 75, 96, 0, 0, 0,
	0, 0, 0, 0, 83, 0, 0, 0, 76, 517,
	92, 0, 0, 0, 85, 86, 87, 88, 89, 90,
	91, 82, 77, 79, 80, 81, 0, 75, 96, 0,
	0, 0, 0, 0, 0, 0, 83, 0, 0, 0,
	76, 442, 92, 0, 0, 0, 85, 86, 87, 88,
	89, 90, 91, 82, 77, 79, 80, 81, 95, 75,
	0, 0, 0, 0, 0, 0, 0, 97, 83, 0,
	0, 0, 0, 0, 92, 0, 94, 0, 0, 0,
	0, 0, 0, 0, 78, 0, 0, 0, 93, 0,
	95, 0, 0, 0, 0, 84, 0, 0, 0, 97,
	0, 0, 0, 0, 0, 0, 0, 0, 94, 0,
	0, 0, 0, 0, 0, 0, 78, 0, 0, 0,
	93, 0, 95, 0, 0, 0, 0, 84, 0, 0,
	413, 97, 0, 0, 0, 0, 0, 0, 0, 0,
	94, 0, 0, 0, 0, 0, 0, 0, 78, 0,
	0, 0, 93, 0, 96, 0, 0, 0, 0, 84,
	0, 0, 0, 0, 0, 0, 76, 0, 0, 0,
	0, 0, 85, 86, 87, 88, 89, 90, 91, 82,
	77, 79, 80, 81, 0, 75, 96, 0, 0, 64,
	67, 0, 0, 0, 326, 412, 0, 0, 76, 0,
	0, 54, 0, 0, 85, 86, 87, 88, 89, 90,
	91, 82, 77, 79, 80, 81, 83, 75, 96, 0,
	0, 0, 92, 0, 0, 0, 66, 0, 0, 0,
	76, 0, 44, 68, 0, 0, 85, 86, 87, 88,
	89, 90, 91, 82, 77, 79, 80, 81, 83, 75,
	64, 67, 0, 0, 92, 0, 0, 0, 0, 0,
	0, 0, 54, 254, 0, 0, 0, 0, 29, 43,
	95, 0, 0, 42, 46, 0, 0, 0, 0, 97,
	0, 0, 0, 0, 83, 0, 0, 66, 94, 0,
	92, 12, 0, 44, 68, 253, 78, 0, 0, 0,
	93, 0, 95, 0, 0, 0, 26, 84, 0, 65,
	0, 97, 48, 0, 0, 0, 0, 0, 45, 0,
	94, 0, 0, 0, 0, 0, 0, 0, 78, 29,
	43, 0, 93, 11, 42, 46, 0, 0, 95, 84,
	0, 47, 25, 0, 51, 52, 57, 97, 62, 0,
	63, 0, 0, 0, 0, 0, 94, 0, 0, 0,
	0, 0, 0, 0, 78, 0, 96, 26, 93, 0,
	65, 0, 0, 48, 0, 84, 0, 0, 76, 45,
	0, 0, 0, 0, 85, 86, 87, 88, 89, 90,
	91, 82, 77, 79, 80, 81, 0, 75, 96, 0,
	0, 0, 47, 25, 0, 51, 52, 57, 0, 62,
	76, 63, 489, 0, 0, 0, 85, 86, 87, 88,
	89, 90, 91, 82, 77, 79, 80, 81, 0, 75,
	124, 0, 161, 0, 96, 0, 64, 67, 0, 0,
	0, 0, 0, 0, 0, 0, 76, 0, 54, 0,
	0, 0, 85, 86, 87, 88, 89, 90, 91, 82,
	77, 79, 80, 81, 0, 75, 160, 0, 0, 0,
	165, 0, 0, 66, 0, 0, 0, 12, 83, 44,
	68, 0, 0, 0, 92, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 29, 43, 0, 0, 11,
	42, 46, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 95, 0, 0, 0, 0, 0, 0, 0,
	164, 97, 0, 0, 0, 0, 0, 0, 0, 0,
	94, 0, 0, 26, 0, 0, 65, 0, 78, 48,
	0, 0, 93, 0, 0, 45, 0, 0, 0, 84,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 64, 67, 0, 0, 0, 0, 47, 25,
	0, 51, 52, 57, 54, 62, 0, 63, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 83,
	0, 0, 166, 0, 0, 92, 165, 0, 0, 66,
	0, 0, 0, 12, 0, 44, 68, 0, 96, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	76, 0, 0, 0, 0, 0, 85, 86, 87, 88,
	89, 90, 91, 82, 77, 79, 80, 81, 92, 75,
	0, 29, 43, 95, 0, 11, 42, 46, 0, 0,
	0, 0, 97, 0, 0, 0, 0, 0, 0, 0,
	0, 94, 0, 0, 64, 67, 164, 0, 0, 78,
	0, 0, 0, 93, 0, 0, 54, 0, 0, 26,
	92, 0, 65, 0, 0, 48, 95, 0, 0, 0,
	0, 45, 0, 0, 224, 97, 0, 0, 0, 0,
	0, 66, 0, 0, 94, 12, 0, 44, 68, 0,
	0, 0, 78, 0, 47, 25, 93, 51, 52, 57,
	0, 62, 0, 63, 0, 0, 0, 0, 95, 0,
	0, 0, 0, 0, 0, 0, 0, 97, 166, 96,
	0, 0, 0, 29, 43, 0, 94, 11, 42, 46,
	0, 76, 0, 0, 78, 0, 0, 85, 86, 87,
	88, 89, 90, 91, 82, 77, 79, 80, 81, 0,
	75, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 26, 96, 0, 65, 64, 67, 48, 0, 0,
	0, 0, 0, 45, 76, 0, 0, 54, 0, 0,
	85, 86, 87, 88, 89, 90, 91, 82, 77, 79,
	80, 81, 0, 75, 0, 0, 47, 25, 0, 51,
	52, 57, 66, 62, 96, 63, 12, 0, 44, 68,
	0, 0, 0, 0, 0, 0, 76, 0, 0, 0,
	225, 0, 0, 0, 0, 88, 89, 90, 91, 82,
	77, 79, 80, 81, 0, 75, 0, 0, 0, 0,
	0, 0, 0, 0, 29, 43, 0, 0, 11, 42,
	46, 0, 61, 0, 0, 64, 67, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 54, 0, 0,
	0, 0, 0, 0, 0, 59, 0, 0, 0, 0,
	0, 36, 26, 0, 0, 65, 0, 60, 48, 0,
	0, 0, 66, 0, 45, 15, 12, 13, 44, 68,
	0, 0, 73, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 34, 64, 67, 47, 25, 0,
	51, 52, 57, 0, 62, 0, 63, 54, 0, 0,
	0, 0, 0, 38, 29, 43, 0, 0, 11, 42,
	46, 166, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 66, 14, 0, 0, 12, 0, 44, 68,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 74, 26, 0, 0, 65, 64, 67, 48, 0,
	0, 0, 0, 0, 45, 0, 0, 0, 54, 0,
	0, 37, 35, 0, 29, 43, 0, 0, 11, 42,
	46, 0, 0, 0, 0, 0, 0, 47, 25, 0,
	51, 52, 57, 66, 62, 0, 63, 12, 0, 44,
	68, 0, 0, 73, 0, 0, 0, 0, 0, 0,
	0, 0, 26, 0, 0, 65, 0, 0, 48, 0,
	0, 64, 67, 0, 45, 0, 0, 0, 0, 0,
	0, 0, 0, 54, 0, 29, 43, 0, 0, 11,
	42, 46, 0, 0, 0, 0, 0, 47, 25, 0,
	51, 52, 57, 0, 62, 0, 63, 372, 66, 0,
	0, 0, 12, 0, 44, 68, 0, 0, 0, 0,
	0, 0, 74, 26, 0, 0, 65, 64, 67, 48,
	0, 0, 0, 0, 0, 45, 0, 0, 0, 54,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	29, 43, 0, 0, 11, 42, 46, 0, 47, 25,
	0, 51, 52, 57, 66, 62, 0, 63, 12, 0,
	44, 68, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 26, 0,
	0, 65, 0, 0, 48, 0, 0, 0, 0, 0,
	45, 0, 0, 0, 0, 0, 29, 43, 0, 0,
	11, 42, 46, 0, 0, 0, 124, 0, 0, 0,
	0, 0, 0, 47, 25, 0, 51, 52, 57, 0,
	62, 0, 63, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 26, 0, 0, 65, 0, 0,
	48, 0, 0, 0, 0, 0, 45, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 47,
	25, 0, 51, 52, 57, 0, 62, 0, 63,
}
var yyPact = []int{

	2227, -1000, -1000, 1811, -1000, -1000, -1000, -1000, -1000, -1000,
	-1000, 2469, 2469, 972, 972, -23, -1000, -1000, -1000, -1000,
	-1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000, 2469,
	-1000, -1000, -1000, 421, 367, 349, 397, 20, 347, -1000,
	-1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000,
	-1000, -1000, -1000, -3, 2413, -1000, -1000, 2348, -1000, 276,
	275, 378, 123, 2469, 12, 12, 12, 2469, 2469, -1000,
	-1000, 299, 396, 44, 1768, -15, 2469, 2469, 2469, 2469,
	2469, 2469, 2469, 2469, 2469, 2469, 2469, 2469, 2469, 2469,
	2469, 2469, 1521, 194, 2469, 2469, 2469, 174, 1985, 33,
	-1000, -1000, -1000, -67, 287, 325, 288, 283, -1000, 472,
	20, 20, 20, 76, -45, 160, -1000, 20, 2016, 438,
	-1000, -1000, 1617, 144, 2469, -10, 1811, -1000, 377, 13,
	376, 20, 20, -25, -35, -1000, -46, -31, -36, 1811,
	-21, -1000, 121, -1000, -21, -21, 1581, 1549, 30, -1000,
	19, 299, -1000, 351, -1000, -134, -68, -69, -1000, -66,
	1914, 2137, 2469, -1000, -1000, -1000, -1000, 954, -1000, -1000,
	2469, 806, -62, -62, -67, -67, -67, 481, 1985, 1942,
	2027, 2027, 2027, 127, 127, 127, 127, 447, -1000, 1521,
	2469, 2469, 2469, 878, 33, 33, -1000, 172, -1000, -1000,
	235, -1000, 2469, -1000, 201, -1000, 201, -1000, 201, 2469,
	312, 312, 76, 143, -1000, 168, 17, -1000, -1000, -1000,
	19, -1000, 65, -13, 2469, -14, -1000, 144, 2469, -1000,
	2469, 1401, -1000, 232, 227, -1000, 218, -127, -1000, -99,
	-129, -1000, 123, 2469, -1000, 2469, 437, 12, 2469, 2469,
	2469, 435, 433, 12, 12, 346, -1000, 2469, -43, -1000,
	-110, 30, 217, -1000, 190, 160, 5, 17, 17, 2137,
	-66, 2469, -66, 581, -53, -1000, 774, -1000, 2287, 1521,
	-20, 2469, 1521, 1521, 1521, 1521, 1521, 1521, 326, 878,
	33, 33, -1000, -1000, -1000, -1000, -1000, 2469, 1811, -1000,
	-1000, -1000, -37, -1000, 741, 175, -1000, 2469, 175, 30,
	57, 30, 5, 5, 308, -1000, 160, -1000, -1000, 18,
	-1000, 1369, -1000, -1000, 1337, 1811, 2469, 20, 20, 20,
	13, 17, 13, -1000, 1811, 1811, -1000, -1000, 1811, 1811,
	1811, -1000, -1000, -39, -39, 147, -1000, 468, -1000, 19,
	1811, 19, 2469, 346, 40, 40, 2469, -1000, -1000, -1000,
	-1000, 76, -71, -1000, -134, -134, -1000, 581, -1000, -1000,
	-1000, -1000, -1000, 1211, 51, -1000, -1000, 2469, 613, -113,
	-113, -70, -70, -70, 290, 1521, 1811, 2469, -1000, -1000,
	-1000, -1000, 150, 150, 2469, 1811, 150, 150, 287, 30,
	287, 287, -38, -1000, -112, -40, -1000, 6, 2469, -1000,
	210, 201, -1000, 2469, 1811, 64, -8, -1000, -1000, -1000,
	166, 425, 2469, 424, -1000, 2469, -43, -1000, 1811, -1000,
	-1000, -134, -103, -107, -1000, 581, -1000, 3, 2469, 160,
	160, -1000, -1000, 549, -1000, 1582, 51, -1000, -1000, -1000,
	1914, -1000, 1811, -1000, -1000, 150, 287, 150, 150, 5,
	2469, 5, -1000, -1000, 12, 1811, 312, -18, 1811, -1000,
	128, 2469, -1000, 93, -1000, 1811, -1000, -11, 160, 17,
	17, -1000, -1000, -1000, 1179, 76, 76, -1000, -1000, -1000,
	1147, -1000, -66, 2469, -1000, 150, -1000, -1000, -1000, 1021,
	-1000, -42, -1000, 106, 55, 160, -1000, -1000, -72, -1000,
	1811, 13, 391, -1000, 197, -134, -134, -1000, -1000, -1000,
	-1000, 1811, -1000, -1000, 411, 12, 5, 5, 287, 331,
	207, 187, 2469, -1000, -1000, -1000, 2469, -1000, 168, 160,
	160, -1000, -1000, -1000, -71, -1000, 150, 110, 279, 312,
	32, 466, -1000, 1811, 306, 197, 197, -1000, 192, 109,
	55, 64, 2469, 2469, 2469, -1000, -1000, 143, 30, 371,
	287, -1000, -1000, 1811, 1811, 46, 57, 30, 37, -1000,
	2469, 150, -1000, 270, -1000, 30, -1000, -1000, 240, -1000,
	989, -1000, 108, 271, -1000, 253, -1000, 431, 107, 99,
	30, 370, 353, 37, 2469, 2469, -1000, -1000, -1000,
}
var yyPgo = []int{

	0, 659, 658, 505, 657, 656, 51, 655, 654, 0,
	145, 62, 38, 302, 45, 49, 53, 21, 17, 22,
	653, 652, 649, 648, 48, 277, 646, 645, 644, 47,
	46, 224, 26, 643, 641, 639, 638, 35, 637, 52,
	634, 627, 625, 434, 624, 42, 40, 623, 18, 24,
	115, 36, 622, 32, 14, 223, 621, 6, 618, 39,
	617, 615, 28, 614, 613, 50, 34, 612, 41, 611,
	603, 31, 602, 601, 9, 600, 599, 598, 596, 515,
	595, 593, 590, 589, 587, 582, 579, 578, 574, 569,
	567, 560, 555, 554, 553, 386, 43, 16, 552, 550,
	544, 4, 23, 542, 19, 7, 33, 536, 8, 29,
	535, 532, 25, 11, 526, 524, 3, 2, 5, 27,
	44, 523, 15, 522, 10, 513, 503, 502, 37, 501,
	20, 492,
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
	-10, 91, 49, 50, 106, 48, -37, -86, -87, -88,
	-89, -90, -91, -1, -2, 161, 125, -5, -33, 87,
	-20, -26, -35, -38, 67, 145, 34, 144, 86, -92,
	-93, -94, 92, 88, 51, 137, 93, 160, 131, -3,
	-4, 163, 164, -34, 20, -27, -28, 165, -39, 28,
	40, 5, 167, 169, 8, 128, 45, 9, 52, -41,
	-40, -43, -68, 55, 124, 188, 169, 183, 87, 184,
	185, 186, 182, 7, 98, 175, 176, 177, 178, 179,
	180, 181, 13, 91, 79, 61, 157, 70, -9, -9,
	-79, -79, -3, -9, -70, 140, 68, 46, -69, 99,
	69, 69, 55, -95, -50, -51, 161, 69, 165, -21,
	-22, -23, -9, -25, 153, -36, -9, -37, 107, 64,
	107, 64, 64, -8, -7, -6, 131, -13, -12, -9,
	-30, -29, -19, 161, -30, -30, -9, -9, -55, -56,
	77, -44, -43, -42, -45, -51, -50, 132, -67, -66,
	38, 4, -128, -65, 112, 42, 184, -9, 161, 162,
	169, -9, -9, -9, -9, -9, -9, -9, -9, -9,
	-9, -9, -9, -9, -9, -9, -9, -11, -10, 13,
	79, 61, 157, -9, -9, -9, 92, 91, 88, 150,
	-74, -73, 80, -39, 4, -39, 4, -39, 4, 18,
	-95, -95, -95, -53, -52, 146, 173, -18, -17, -16,
	10, 161, -95, -13, 38, 184, 44, -25, 153, -24,
	43, -9, 166, 64, -119, 161, 64, -120, -51, -50,
	-120, 168, 172, 173, 170, 172, -31, 172, 122, 61,
	157, -31, -31, 54, 54, -57, -58, 154, -15, -14,
	-16, -55, -47, 66, 76, -49, 188, 173, 173, 172,
	-66, -128, -66, -9, 188, -18, -9, 170, 173, 7,
	188, 169, 183, 87, 184, 185, 186, 182, -11, -9,
	-9, -9, 92, 88, 150, -76, -75, 95, -9, -39,
	-39, -39, -72, -71, -9, -98, -97, 72, -97, -53,
	-105, -108, 126, 143, -130, 107, -51, 161, -16, 148,
	166, -9, 166, -24, -9, -9, 133, 96, 96, 96,
	188, 173, 188, -6, -9, -9, 44, -29, -9, -9,
	-9, 44, 44, -30, -30, -59, -60, 58, -62, 78,
	-9, 172, 175, -57, 71, 90, -129, 142, 53, -131,
	100, -18, -48, 161, -51, -51, -65, -9, -18, 184,
	170, 171, 170, -9, -11, 161, 162, 169, -9, -11,
	-11, -11, -11, -11, -11, 7, -9, 172, -78, -77,
	11, 36, -96, -37, 151, -9, -96, -37, -57, -108,
	-57, -57, -107, -106, -48, -110, -109, -48, 73, -18,
	-45, 165, 166, 133, -9, -120, -120, -120, -119, -51,
	-119, -32, 153, -32, -68, 18, -15, -14, -9, -59,
	-46, -51, -50, 132, -46, -9, -53, 188, 169, -49,
	-49, -18, 170, -9, 170, 173, -11, -71, -101, -100,
	117, -101, -9, -101, -101, -74, -57, -74, -74, 172,
	175, 172, -112, -111, 54, -9, 96, -37, -9, -122,
	148, 165, -123, 115, 44, -9, 44, -12, -49, 173,
	173, -18, 161, 162, -9, -18, -18, 170, 171, 170,
	-9, -99, -66, -128, -101, -74, -101, -101, -106, -9,
	-109, -103, -102, -19, -97, 166, 152, 81, -126, -124,
	-9, 134, -61, -62, -18, -51, -51, 170, -53, -53,
	170, -9, -101, -112, -32, 172, 61, 157, -113, 153,
	-17, 166, 172, -119, -63, -64, 59, -54, 96, -49,
	-49, 44, -102, -104, -48, -104, -74, 84, 91, 96,
	-121, 102, -124, -9, -130, -18, -18, -101, 133, 84,
	-97, -125, 154, 18, 73, -54, -54, 144, 34, 133,
	-113, -122, -124, -9, -9, -115, -105, -108, -116, -57,
	67, -74, -114, 153, -57, -108, -57, -118, 153, -117,
	-9, -101, 84, 91, -57, 91, -57, 133, 84, 84,
	34, 133, 133, -116, 67, 67, -118, -117, -117,
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
	182, 183, 184, 185, 186, 187, 188,
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
		//line n1ql.y:351
		{
			yylex.(*lexer).setStatement(yyS[yypt-0].statement)
		}
	case 2:
		//line n1ql.y:356
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
		//line n1ql.y:377
		{
			yyVAL.statement = algebra.NewExplain(yyS[yypt-0].statement)
		}
	case 10:
		//line n1ql.y:384
		{
			yyVAL.statement = algebra.NewPrepare(yyS[yypt-0].statement)
		}
	case 11:
		//line n1ql.y:391
		{
			yyVAL.statement = algebra.NewExecute(yyS[yypt-0].expr)
		}
	case 12:
		//line n1ql.y:398
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
		//line n1ql.y:429
		{
			yyVAL.fullselect = algebra.NewSelect(yyS[yypt-3].subresult, yyS[yypt-2].order, yyS[yypt-0].expr, yyS[yypt-1].expr) /* OFFSET precedes LIMIT */
		}
	case 23:
		//line n1ql.y:435
		{
			yyVAL.subresult = yyS[yypt-0].subselect
		}
	case 24:
		//line n1ql.y:440
		{
			yyVAL.subresult = algebra.NewUnion(yyS[yypt-2].subresult, yyS[yypt-0].subselect)
		}
	case 25:
		//line n1ql.y:445
		{
			yyVAL.subresult = algebra.NewUnionAll(yyS[yypt-3].subresult, yyS[yypt-0].subselect)
		}
	case 26:
		//line n1ql.y:450
		{
			yyVAL.subresult = algebra.NewIntersect(yyS[yypt-2].subresult, yyS[yypt-0].subselect)
		}
	case 27:
		//line n1ql.y:455
		{
			yyVAL.subresult = algebra.NewIntersectAll(yyS[yypt-3].subresult, yyS[yypt-0].subselect)
		}
	case 28:
		//line n1ql.y:460
		{
			yyVAL.subresult = algebra.NewExcept(yyS[yypt-2].subresult, yyS[yypt-0].subselect)
		}
	case 29:
		//line n1ql.y:465
		{
			yyVAL.subresult = algebra.NewExceptAll(yyS[yypt-3].subresult, yyS[yypt-0].subselect)
		}
	case 30:
		yyVAL.subselect = yyS[yypt-0].subselect
	case 31:
		yyVAL.subselect = yyS[yypt-0].subselect
	case 32:
		//line n1ql.y:478
		{
			yyVAL.subselect = algebra.NewSubselect(yyS[yypt-4].fromTerm, yyS[yypt-3].bindings, yyS[yypt-2].expr, yyS[yypt-1].group, yyS[yypt-0].projection)
		}
	case 33:
		//line n1ql.y:485
		{
			yyVAL.subselect = algebra.NewSubselect(yyS[yypt-3].fromTerm, yyS[yypt-2].bindings, yyS[yypt-1].expr, yyS[yypt-0].group, yyS[yypt-4].projection)
		}
	case 34:
		//line n1ql.y:500
		{
			yyVAL.projection = yyS[yypt-0].projection
		}
	case 35:
		//line n1ql.y:507
		{
			yyVAL.projection = algebra.NewProjection(false, yyS[yypt-0].resultTerms)
		}
	case 36:
		//line n1ql.y:512
		{
			yyVAL.projection = algebra.NewProjection(true, yyS[yypt-0].resultTerms)
		}
	case 37:
		//line n1ql.y:517
		{
			yyVAL.projection = algebra.NewProjection(false, yyS[yypt-0].resultTerms)
		}
	case 38:
		//line n1ql.y:522
		{
			yyVAL.projection = algebra.NewRawProjection(false, yyS[yypt-1].expr, yyS[yypt-0].s)
		}
	case 39:
		//line n1ql.y:527
		{
			yyVAL.projection = algebra.NewRawProjection(true, yyS[yypt-1].expr, yyS[yypt-0].s)
		}
	case 42:
		//line n1ql.y:540
		{
			yyVAL.resultTerms = algebra.ResultTerms{yyS[yypt-0].resultTerm}
		}
	case 43:
		//line n1ql.y:545
		{
			yyVAL.resultTerms = append(yyS[yypt-2].resultTerms, yyS[yypt-0].resultTerm)
		}
	case 44:
		//line n1ql.y:552
		{
			yyVAL.resultTerm = algebra.NewResultTerm(nil, true, "")
		}
	case 45:
		//line n1ql.y:557
		{
			yyVAL.resultTerm = algebra.NewResultTerm(yyS[yypt-2].expr, true, "")
		}
	case 46:
		//line n1ql.y:562
		{
			yyVAL.resultTerm = algebra.NewResultTerm(yyS[yypt-1].expr, false, yyS[yypt-0].s)
		}
	case 47:
		//line n1ql.y:569
		{
			yyVAL.s = ""
		}
	case 48:
		yyVAL.s = yyS[yypt-0].s
	case 49:
		yyVAL.s = yyS[yypt-0].s
	case 50:
		//line n1ql.y:580
		{
			yyVAL.s = yyS[yypt-0].s
		}
	case 51:
		yyVAL.s = yyS[yypt-0].s
	case 52:
		//line n1ql.y:598
		{
			yyVAL.fromTerm = nil
		}
	case 53:
		yyVAL.fromTerm = yyS[yypt-0].fromTerm
	case 54:
		//line n1ql.y:607
		{
			yyVAL.fromTerm = yyS[yypt-0].fromTerm
		}
	case 55:
		//line n1ql.y:614
		{
			yyVAL.fromTerm = yyS[yypt-0].keyspaceTerm
		}
	case 56:
		//line n1ql.y:619
		{
			yyVAL.fromTerm = algebra.NewJoin(yyS[yypt-3].fromTerm, yyS[yypt-2].b, yyS[yypt-0].keyspaceTerm)
		}
	case 57:
		//line n1ql.y:624
		{
			yyVAL.fromTerm = algebra.NewNest(yyS[yypt-3].fromTerm, yyS[yypt-2].b, yyS[yypt-0].keyspaceTerm)
		}
	case 58:
		//line n1ql.y:629
		{
			yyVAL.fromTerm = algebra.NewUnnest(yyS[yypt-4].fromTerm, yyS[yypt-3].b, yyS[yypt-1].expr, yyS[yypt-0].s)
		}
	case 61:
		//line n1ql.y:642
		{
			yyVAL.keyspaceTerm = algebra.NewKeyspaceTerm("", yyS[yypt-3].s, yyS[yypt-2].path, yyS[yypt-1].s, yyS[yypt-0].expr)
		}
	case 62:
		//line n1ql.y:647
		{
			yyVAL.keyspaceTerm = algebra.NewKeyspaceTerm(yyS[yypt-5].s, yyS[yypt-3].s, yyS[yypt-2].path, yyS[yypt-1].s, yyS[yypt-0].expr)
		}
	case 63:
		//line n1ql.y:652
		{
			yyVAL.keyspaceTerm = algebra.NewKeyspaceTerm("#system", yyS[yypt-3].s, yyS[yypt-2].path, yyS[yypt-1].s, yyS[yypt-0].expr)
		}
	case 64:
		//line n1ql.y:659
		{
			yyVAL.keyspaceTerm = algebra.NewKeyspaceTerm("", yyS[yypt-3].s, yyS[yypt-2].path, yyS[yypt-1].s, yyS[yypt-0].expr)
		}
	case 65:
		//line n1ql.y:664
		{
			yyVAL.keyspaceTerm = algebra.NewKeyspaceTerm(yyS[yypt-5].s, yyS[yypt-3].s, yyS[yypt-2].path, yyS[yypt-1].s, yyS[yypt-0].expr)
		}
	case 66:
		//line n1ql.y:669
		{
			yyVAL.keyspaceTerm = algebra.NewKeyspaceTerm("#system", yyS[yypt-3].s, yyS[yypt-2].path, yyS[yypt-1].s, yyS[yypt-0].expr)
		}
	case 67:
		yyVAL.s = yyS[yypt-0].s
	case 68:
		yyVAL.s = yyS[yypt-0].s
	case 69:
		//line n1ql.y:684
		{
			yyVAL.path = nil
		}
	case 70:
		//line n1ql.y:689
		{
			yyVAL.path = yyS[yypt-0].path
		}
	case 71:
		//line n1ql.y:696
		{
			yyVAL.expr = nil
		}
	case 72:
		yyVAL.expr = yyS[yypt-0].expr
	case 73:
		//line n1ql.y:705
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 74:
		//line n1ql.y:712
		{
		}
	case 76:
		//line n1ql.y:720
		{
			yyVAL.b = false
		}
	case 77:
		//line n1ql.y:725
		{
			yyVAL.b = false
		}
	case 78:
		//line n1ql.y:730
		{
			yyVAL.b = true
		}
	case 81:
		//line n1ql.y:743
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 82:
		//line n1ql.y:757
		{
			yyVAL.bindings = nil
		}
	case 83:
		yyVAL.bindings = yyS[yypt-0].bindings
	case 84:
		//line n1ql.y:766
		{
			yyVAL.bindings = yyS[yypt-0].bindings
		}
	case 85:
		//line n1ql.y:773
		{
			yyVAL.bindings = expression.Bindings{yyS[yypt-0].binding}
		}
	case 86:
		//line n1ql.y:778
		{
			yyVAL.bindings = append(yyS[yypt-2].bindings, yyS[yypt-0].binding)
		}
	case 87:
		//line n1ql.y:785
		{
			yyVAL.binding = expression.NewBinding(yyS[yypt-2].s, yyS[yypt-0].expr)
		}
	case 88:
		//line n1ql.y:799
		{
			yyVAL.expr = nil
		}
	case 89:
		yyVAL.expr = yyS[yypt-0].expr
	case 90:
		//line n1ql.y:808
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 91:
		//line n1ql.y:822
		{
			yyVAL.group = nil
		}
	case 92:
		yyVAL.group = yyS[yypt-0].group
	case 93:
		//line n1ql.y:831
		{
			yyVAL.group = algebra.NewGroup(yyS[yypt-2].exprs, yyS[yypt-1].bindings, yyS[yypt-0].expr)
		}
	case 94:
		//line n1ql.y:836
		{
			yyVAL.group = algebra.NewGroup(nil, yyS[yypt-0].bindings, nil)
		}
	case 95:
		//line n1ql.y:843
		{
			yyVAL.exprs = expression.Expressions{yyS[yypt-0].expr}
		}
	case 96:
		//line n1ql.y:848
		{
			yyVAL.exprs = append(yyS[yypt-2].exprs, yyS[yypt-0].expr)
		}
	case 97:
		//line n1ql.y:855
		{
			yyVAL.bindings = nil
		}
	case 98:
		yyVAL.bindings = yyS[yypt-0].bindings
	case 99:
		//line n1ql.y:864
		{
			yyVAL.bindings = yyS[yypt-0].bindings
		}
	case 100:
		//line n1ql.y:871
		{
			yyVAL.expr = nil
		}
	case 101:
		yyVAL.expr = yyS[yypt-0].expr
	case 102:
		//line n1ql.y:880
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 103:
		//line n1ql.y:894
		{
			yyVAL.order = nil
		}
	case 104:
		yyVAL.order = yyS[yypt-0].order
	case 105:
		//line n1ql.y:903
		{
			yyVAL.order = algebra.NewOrder(yyS[yypt-0].sortTerms)
		}
	case 106:
		//line n1ql.y:910
		{
			yyVAL.sortTerms = algebra.SortTerms{yyS[yypt-0].sortTerm}
		}
	case 107:
		//line n1ql.y:915
		{
			yyVAL.sortTerms = append(yyS[yypt-2].sortTerms, yyS[yypt-0].sortTerm)
		}
	case 108:
		//line n1ql.y:922
		{
			yyVAL.sortTerm = algebra.NewSortTerm(yyS[yypt-1].expr, yyS[yypt-0].b)
		}
	case 109:
		//line n1ql.y:929
		{
			yyVAL.b = false
		}
	case 110:
		yyVAL.b = yyS[yypt-0].b
	case 111:
		//line n1ql.y:938
		{
			yyVAL.b = false
		}
	case 112:
		//line n1ql.y:943
		{
			yyVAL.b = true
		}
	case 113:
		//line n1ql.y:957
		{
			yyVAL.expr = nil
		}
	case 114:
		yyVAL.expr = yyS[yypt-0].expr
	case 115:
		//line n1ql.y:966
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 116:
		//line n1ql.y:980
		{
			yyVAL.expr = nil
		}
	case 117:
		yyVAL.expr = yyS[yypt-0].expr
	case 118:
		//line n1ql.y:989
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 119:
		//line n1ql.y:1003
		{
			yyVAL.statement = algebra.NewInsertValues(yyS[yypt-3].keyspaceRef, yyS[yypt-2].expr, yyS[yypt-1].expr, yyS[yypt-0].projection)
		}
	case 120:
		//line n1ql.y:1008
		{
			yyVAL.statement = algebra.NewInsertSelect(yyS[yypt-3].keyspaceRef, yyS[yypt-2].expr, yyS[yypt-1].fullselect, yyS[yypt-0].projection)
		}
	case 121:
		//line n1ql.y:1015
		{
			yyVAL.keyspaceRef = algebra.NewKeyspaceRef(yyS[yypt-3].s, yyS[yypt-1].s, yyS[yypt-0].s)
		}
	case 122:
		//line n1ql.y:1020
		{
			yyVAL.keyspaceRef = algebra.NewKeyspaceRef("", yyS[yypt-1].s, yyS[yypt-0].s)
		}
	case 123:
		//line n1ql.y:1027
		{
			yyVAL.expr = nil
		}
	case 124:
		yyVAL.expr = yyS[yypt-0].expr
	case 125:
		//line n1ql.y:1036
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 126:
		//line n1ql.y:1043
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 127:
		//line n1ql.y:1050
		{
			yyVAL.projection = nil
		}
	case 128:
		yyVAL.projection = yyS[yypt-0].projection
	case 129:
		//line n1ql.y:1059
		{
			yyVAL.projection = yyS[yypt-0].projection
		}
	case 130:
		//line n1ql.y:1066
		{
			yyVAL.projection = algebra.NewProjection(false, yyS[yypt-0].resultTerms)
		}
	case 131:
		//line n1ql.y:1071
		{
			yyVAL.projection = algebra.NewRawProjection(false, yyS[yypt-0].expr, "")
		}
	case 132:
		//line n1ql.y:1085
		{
			yyVAL.statement = algebra.NewUpsertValues(yyS[yypt-3].keyspaceRef, yyS[yypt-2].expr, yyS[yypt-1].expr, yyS[yypt-0].projection)
		}
	case 133:
		//line n1ql.y:1090
		{
			yyVAL.statement = algebra.NewUpsertSelect(yyS[yypt-3].keyspaceRef, yyS[yypt-2].expr, yyS[yypt-1].fullselect, yyS[yypt-0].projection)
		}
	case 134:
		//line n1ql.y:1104
		{
			yyVAL.statement = algebra.NewDelete(yyS[yypt-4].keyspaceRef, yyS[yypt-3].expr, yyS[yypt-2].expr, yyS[yypt-1].expr, yyS[yypt-0].projection)
		}
	case 135:
		//line n1ql.y:1118
		{
			yyVAL.statement = algebra.NewUpdate(yyS[yypt-6].keyspaceRef, yyS[yypt-5].expr, yyS[yypt-4].set, yyS[yypt-3].unset, yyS[yypt-2].expr, yyS[yypt-1].expr, yyS[yypt-0].projection)
		}
	case 136:
		//line n1ql.y:1123
		{
			yyVAL.statement = algebra.NewUpdate(yyS[yypt-5].keyspaceRef, yyS[yypt-4].expr, yyS[yypt-3].set, nil, yyS[yypt-2].expr, yyS[yypt-1].expr, yyS[yypt-0].projection)
		}
	case 137:
		//line n1ql.y:1128
		{
			yyVAL.statement = algebra.NewUpdate(yyS[yypt-5].keyspaceRef, yyS[yypt-4].expr, nil, yyS[yypt-3].unset, yyS[yypt-2].expr, yyS[yypt-1].expr, yyS[yypt-0].projection)
		}
	case 138:
		//line n1ql.y:1135
		{
			yyVAL.set = algebra.NewSet(yyS[yypt-0].setTerms)
		}
	case 139:
		//line n1ql.y:1142
		{
			yyVAL.setTerms = algebra.SetTerms{yyS[yypt-0].setTerm}
		}
	case 140:
		//line n1ql.y:1147
		{
			yyVAL.setTerms = append(yyS[yypt-2].setTerms, yyS[yypt-0].setTerm)
		}
	case 141:
		//line n1ql.y:1154
		{
			yyVAL.setTerm = algebra.NewSetTerm(yyS[yypt-3].path, yyS[yypt-1].expr, yyS[yypt-0].updateFor)
		}
	case 142:
		//line n1ql.y:1161
		{
			yyVAL.updateFor = nil
		}
	case 143:
		yyVAL.updateFor = yyS[yypt-0].updateFor
	case 144:
		//line n1ql.y:1170
		{
			yyVAL.updateFor = algebra.NewUpdateFor(yyS[yypt-2].bindings, yyS[yypt-1].expr)
		}
	case 145:
		//line n1ql.y:1177
		{
			yyVAL.bindings = expression.Bindings{yyS[yypt-0].binding}
		}
	case 146:
		//line n1ql.y:1182
		{
			yyVAL.bindings = append(yyS[yypt-2].bindings, yyS[yypt-0].binding)
		}
	case 147:
		//line n1ql.y:1189
		{
			yyVAL.binding = expression.NewBinding(yyS[yypt-2].s, yyS[yypt-0].expr)
		}
	case 148:
		//line n1ql.y:1194
		{
			yyVAL.binding = expression.NewDescendantBinding(yyS[yypt-2].s, yyS[yypt-0].expr)
		}
	case 149:
		yyVAL.s = yyS[yypt-0].s
	case 150:
		//line n1ql.y:1205
		{
			yyVAL.expr = yyS[yypt-0].path
		}
	case 151:
		//line n1ql.y:1212
		{
			yyVAL.expr = nil
		}
	case 152:
		//line n1ql.y:1217
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 153:
		//line n1ql.y:1224
		{
			yyVAL.unset = algebra.NewUnset(yyS[yypt-0].unsetTerms)
		}
	case 154:
		//line n1ql.y:1231
		{
			yyVAL.unsetTerms = algebra.UnsetTerms{yyS[yypt-0].unsetTerm}
		}
	case 155:
		//line n1ql.y:1236
		{
			yyVAL.unsetTerms = append(yyS[yypt-2].unsetTerms, yyS[yypt-0].unsetTerm)
		}
	case 156:
		//line n1ql.y:1243
		{
			yyVAL.unsetTerm = algebra.NewUnsetTerm(yyS[yypt-1].path, yyS[yypt-0].updateFor)
		}
	case 157:
		//line n1ql.y:1257
		{
			source := algebra.NewMergeSourceFrom(yyS[yypt-5].keyspaceTerm, "")
			yyVAL.statement = algebra.NewMerge(yyS[yypt-7].keyspaceRef, source, yyS[yypt-3].expr, yyS[yypt-2].mergeActions, yyS[yypt-1].expr, yyS[yypt-0].projection)
		}
	case 158:
		//line n1ql.y:1263
		{
			source := algebra.NewMergeSourceSelect(yyS[yypt-7].fullselect, yyS[yypt-5].s)
			yyVAL.statement = algebra.NewMerge(yyS[yypt-10].keyspaceRef, source, yyS[yypt-3].expr, yyS[yypt-2].mergeActions, yyS[yypt-1].expr, yyS[yypt-0].projection)
		}
	case 159:
		//line n1ql.y:1271
		{
			yyVAL.mergeActions = algebra.NewMergeActions(nil, nil, nil)
		}
	case 160:
		//line n1ql.y:1276
		{
			yyVAL.mergeActions = algebra.NewMergeActions(yyS[yypt-1].mergeUpdate, yyS[yypt-0].mergeActions.Delete(), yyS[yypt-0].mergeActions.Insert())
		}
	case 161:
		//line n1ql.y:1281
		{
			yyVAL.mergeActions = algebra.NewMergeActions(nil, yyS[yypt-1].mergeDelete, yyS[yypt-0].mergeInsert)
		}
	case 162:
		//line n1ql.y:1286
		{
			yyVAL.mergeActions = algebra.NewMergeActions(nil, nil, yyS[yypt-0].mergeInsert)
		}
	case 163:
		//line n1ql.y:1293
		{
			yyVAL.mergeActions = algebra.NewMergeActions(nil, nil, nil)
		}
	case 164:
		//line n1ql.y:1298
		{
			yyVAL.mergeActions = algebra.NewMergeActions(nil, yyS[yypt-1].mergeDelete, yyS[yypt-0].mergeInsert)
		}
	case 165:
		//line n1ql.y:1303
		{
			yyVAL.mergeActions = algebra.NewMergeActions(nil, nil, yyS[yypt-0].mergeInsert)
		}
	case 166:
		//line n1ql.y:1310
		{
			yyVAL.mergeInsert = nil
		}
	case 167:
		//line n1ql.y:1315
		{
			yyVAL.mergeInsert = yyS[yypt-0].mergeInsert
		}
	case 168:
		//line n1ql.y:1322
		{
			yyVAL.mergeUpdate = algebra.NewMergeUpdate(yyS[yypt-1].set, nil, yyS[yypt-0].expr)
		}
	case 169:
		//line n1ql.y:1327
		{
			yyVAL.mergeUpdate = algebra.NewMergeUpdate(yyS[yypt-2].set, yyS[yypt-1].unset, yyS[yypt-0].expr)
		}
	case 170:
		//line n1ql.y:1332
		{
			yyVAL.mergeUpdate = algebra.NewMergeUpdate(nil, yyS[yypt-1].unset, yyS[yypt-0].expr)
		}
	case 171:
		//line n1ql.y:1339
		{
			yyVAL.mergeDelete = algebra.NewMergeDelete(yyS[yypt-0].expr)
		}
	case 172:
		//line n1ql.y:1346
		{
			yyVAL.mergeInsert = algebra.NewMergeInsert(yyS[yypt-1].expr, yyS[yypt-0].expr)
		}
	case 173:
		//line n1ql.y:1360
		{
			yyVAL.statement = algebra.NewCreatePrimaryIndex(yyS[yypt-1].keyspaceRef, yyS[yypt-0].indexType)
		}
	case 174:
		//line n1ql.y:1365
		{
			yyVAL.statement = algebra.NewCreateIndex(yyS[yypt-8].s, yyS[yypt-6].keyspaceRef, yyS[yypt-4].exprs, yyS[yypt-2].expr, yyS[yypt-1].expr, yyS[yypt-0].indexType)
		}
	case 175:
		yyVAL.s = yyS[yypt-0].s
	case 176:
		//line n1ql.y:1376
		{
			yyVAL.keyspaceRef = algebra.NewKeyspaceRef("", yyS[yypt-0].s, "")
		}
	case 177:
		//line n1ql.y:1381
		{
			yyVAL.keyspaceRef = algebra.NewKeyspaceRef(yyS[yypt-2].s, yyS[yypt-0].s, "")
		}
	case 178:
		//line n1ql.y:1388
		{
			yyVAL.expr = nil
		}
	case 179:
		//line n1ql.y:1393
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 180:
		//line n1ql.y:1400
		{
			yyVAL.indexType = datastore.VIEW
		}
	case 181:
		//line n1ql.y:1405
		{
			yyVAL.indexType = datastore.VIEW
		}
	case 182:
		//line n1ql.y:1410
		{
			yyVAL.indexType = datastore.LSM
		}
	case 183:
		//line n1ql.y:1417
		{
			yyVAL.exprs = expression.Expressions{yyS[yypt-0].expr}
		}
	case 184:
		//line n1ql.y:1422
		{
			yyVAL.exprs = append(yyS[yypt-2].exprs, yyS[yypt-0].expr)
		}
	case 185:
		//line n1ql.y:1429
		{
			e := yyS[yypt-0].expr
			if !e.Indexable() {
				yylex.Error(fmt.Sprintf("Expression not indexable."))
			}

			yyVAL.expr = e
		}
	case 186:
		//line n1ql.y:1440
		{
			yyVAL.expr = nil
		}
	case 187:
		//line n1ql.y:1445
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 188:
		//line n1ql.y:1459
		{
			yyVAL.statement = algebra.NewDropIndex(yyS[yypt-0].keyspaceRef, "#primary")
		}
	case 189:
		//line n1ql.y:1464
		{
			yyVAL.statement = algebra.NewDropIndex(yyS[yypt-2].keyspaceRef, yyS[yypt-0].s)
		}
	case 190:
		//line n1ql.y:1477
		{
			yyVAL.statement = algebra.NewAlterIndex(yyS[yypt-3].keyspaceRef, yyS[yypt-1].s, yyS[yypt-0].s)
		}
	case 191:
		//line n1ql.y:1483
		{
			yyVAL.s = ""
		}
	case 192:
		//line n1ql.y:1488
		{
			yyVAL.s = yyS[yypt-0].s
		}
	case 193:
		//line n1ql.y:1502
		{
			yyVAL.path = expression.NewIdentifier(yyS[yypt-0].s)
		}
	case 194:
		//line n1ql.y:1507
		{
			yyVAL.path = expression.NewField(yyS[yypt-2].path, expression.NewFieldName(yyS[yypt-0].s))
		}
	case 195:
		//line n1ql.y:1512
		{
			field := expression.NewField(yyS[yypt-2].path, expression.NewFieldName(yyS[yypt-0].s))
			field.SetCaseInsensitive(true)
			yyVAL.path = field
		}
	case 196:
		//line n1ql.y:1519
		{
			yyVAL.path = expression.NewElement(yyS[yypt-3].path, yyS[yypt-1].expr)
		}
	case 197:
		yyVAL.expr = yyS[yypt-0].expr
	case 198:
		//line n1ql.y:1536
		{
			yyVAL.expr = expression.NewField(yyS[yypt-2].expr, expression.NewFieldName(yyS[yypt-0].s))
		}
	case 199:
		//line n1ql.y:1541
		{
			field := expression.NewField(yyS[yypt-2].expr, expression.NewFieldName(yyS[yypt-0].s))
			field.SetCaseInsensitive(true)
			yyVAL.expr = field
		}
	case 200:
		//line n1ql.y:1548
		{
			yyVAL.expr = expression.NewField(yyS[yypt-4].expr, yyS[yypt-1].expr)
		}
	case 201:
		//line n1ql.y:1553
		{
			field := expression.NewField(yyS[yypt-4].expr, yyS[yypt-1].expr)
			field.SetCaseInsensitive(true)
			yyVAL.expr = field
		}
	case 202:
		//line n1ql.y:1560
		{
			yyVAL.expr = expression.NewElement(yyS[yypt-3].expr, yyS[yypt-1].expr)
		}
	case 203:
		//line n1ql.y:1565
		{
			yyVAL.expr = expression.NewSlice(yyS[yypt-4].expr, yyS[yypt-2].expr)
		}
	case 204:
		//line n1ql.y:1570
		{
			yyVAL.expr = expression.NewSlice(yyS[yypt-5].expr, yyS[yypt-3].expr, yyS[yypt-1].expr)
		}
	case 205:
		//line n1ql.y:1576
		{
			yyVAL.expr = expression.NewAdd(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 206:
		//line n1ql.y:1581
		{
			yyVAL.expr = expression.NewSub(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 207:
		//line n1ql.y:1586
		{
			yyVAL.expr = expression.NewMult(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 208:
		//line n1ql.y:1591
		{
			yyVAL.expr = expression.NewDiv(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 209:
		//line n1ql.y:1596
		{
			yyVAL.expr = expression.NewMod(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 210:
		//line n1ql.y:1602
		{
			yyVAL.expr = expression.NewConcat(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 211:
		//line n1ql.y:1608
		{
			yyVAL.expr = expression.NewAnd(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 212:
		//line n1ql.y:1613
		{
			yyVAL.expr = expression.NewOr(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 213:
		//line n1ql.y:1618
		{
			yyVAL.expr = expression.NewNot(yyS[yypt-0].expr)
		}
	case 214:
		//line n1ql.y:1624
		{
			yyVAL.expr = expression.NewEq(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 215:
		//line n1ql.y:1629
		{
			yyVAL.expr = expression.NewEq(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 216:
		//line n1ql.y:1634
		{
			yyVAL.expr = expression.NewNE(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 217:
		//line n1ql.y:1639
		{
			yyVAL.expr = expression.NewLT(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 218:
		//line n1ql.y:1644
		{
			yyVAL.expr = expression.NewGT(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 219:
		//line n1ql.y:1649
		{
			yyVAL.expr = expression.NewLE(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 220:
		//line n1ql.y:1654
		{
			yyVAL.expr = expression.NewGE(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 221:
		//line n1ql.y:1659
		{
			yyVAL.expr = expression.NewBetween(yyS[yypt-4].expr, yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 222:
		//line n1ql.y:1664
		{
			yyVAL.expr = expression.NewNotBetween(yyS[yypt-5].expr, yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 223:
		//line n1ql.y:1669
		{
			yyVAL.expr = expression.NewLike(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 224:
		//line n1ql.y:1674
		{
			yyVAL.expr = expression.NewNotLike(yyS[yypt-3].expr, yyS[yypt-0].expr)
		}
	case 225:
		//line n1ql.y:1679
		{
			yyVAL.expr = expression.NewIn(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 226:
		//line n1ql.y:1684
		{
			yyVAL.expr = expression.NewNotIn(yyS[yypt-3].expr, yyS[yypt-0].expr)
		}
	case 227:
		//line n1ql.y:1689
		{
			yyVAL.expr = expression.NewWithin(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 228:
		//line n1ql.y:1694
		{
			yyVAL.expr = expression.NewNotWithin(yyS[yypt-3].expr, yyS[yypt-0].expr)
		}
	case 229:
		//line n1ql.y:1699
		{
			yyVAL.expr = expression.NewIsNull(yyS[yypt-2].expr)
		}
	case 230:
		//line n1ql.y:1704
		{
			yyVAL.expr = expression.NewIsNotNull(yyS[yypt-3].expr)
		}
	case 231:
		//line n1ql.y:1709
		{
			yyVAL.expr = expression.NewIsMissing(yyS[yypt-2].expr)
		}
	case 232:
		//line n1ql.y:1714
		{
			yyVAL.expr = expression.NewIsNotMissing(yyS[yypt-3].expr)
		}
	case 233:
		//line n1ql.y:1719
		{
			yyVAL.expr = expression.NewIsValued(yyS[yypt-2].expr)
		}
	case 234:
		//line n1ql.y:1724
		{
			yyVAL.expr = expression.NewIsNotValued(yyS[yypt-3].expr)
		}
	case 235:
		//line n1ql.y:1729
		{
			yyVAL.expr = expression.NewExists(yyS[yypt-0].expr)
		}
	case 236:
		yyVAL.expr = yyS[yypt-0].expr
	case 237:
		yyVAL.expr = yyS[yypt-0].expr
	case 238:
		//line n1ql.y:1743
		{
			yyVAL.expr = expression.NewIdentifier(yyS[yypt-0].s)
		}
	case 239:
		//line n1ql.y:1749
		{
			yyVAL.expr = expression.NewSelf()
		}
	case 240:
		yyVAL.expr = yyS[yypt-0].expr
	case 241:
		yyVAL.expr = yyS[yypt-0].expr
	case 242:
		//line n1ql.y:1761
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
		//line n1ql.y:1780
		{
			yyVAL.expr = expression.NewField(yyS[yypt-2].expr, expression.NewFieldName(yyS[yypt-0].s))
		}
	case 248:
		//line n1ql.y:1785
		{
			field := expression.NewField(yyS[yypt-2].expr, expression.NewFieldName(yyS[yypt-0].s))
			field.SetCaseInsensitive(true)
			yyVAL.expr = field
		}
	case 249:
		//line n1ql.y:1792
		{
			yyVAL.expr = expression.NewField(yyS[yypt-4].expr, yyS[yypt-1].expr)
		}
	case 250:
		//line n1ql.y:1797
		{
			field := expression.NewField(yyS[yypt-4].expr, yyS[yypt-1].expr)
			field.SetCaseInsensitive(true)
			yyVAL.expr = field
		}
	case 251:
		//line n1ql.y:1804
		{
			yyVAL.expr = expression.NewElement(yyS[yypt-3].expr, yyS[yypt-1].expr)
		}
	case 252:
		//line n1ql.y:1809
		{
			yyVAL.expr = expression.NewSlice(yyS[yypt-4].expr, yyS[yypt-2].expr)
		}
	case 253:
		//line n1ql.y:1814
		{
			yyVAL.expr = expression.NewSlice(yyS[yypt-5].expr, yyS[yypt-3].expr, yyS[yypt-1].expr)
		}
	case 254:
		//line n1ql.y:1820
		{
			yyVAL.expr = expression.NewAdd(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 255:
		//line n1ql.y:1825
		{
			yyVAL.expr = expression.NewSub(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 256:
		//line n1ql.y:1830
		{
			yyVAL.expr = expression.NewMult(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 257:
		//line n1ql.y:1835
		{
			yyVAL.expr = expression.NewDiv(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 258:
		//line n1ql.y:1840
		{
			yyVAL.expr = expression.NewMod(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 259:
		//line n1ql.y:1846
		{
			yyVAL.expr = expression.NewConcat(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 260:
		//line n1ql.y:1860
		{
			yyVAL.expr = expression.NULL_EXPR
		}
	case 261:
		//line n1ql.y:1865
		{
			yyVAL.expr = expression.MISSING_EXPR
		}
	case 262:
		//line n1ql.y:1870
		{
			yyVAL.expr = expression.FALSE_EXPR
		}
	case 263:
		//line n1ql.y:1875
		{
			yyVAL.expr = expression.TRUE_EXPR
		}
	case 264:
		//line n1ql.y:1880
		{
			yyVAL.expr = expression.NewConstant(value.NewValue(yyS[yypt-0].f))
		}
	case 265:
		//line n1ql.y:1885
		{
			yyVAL.expr = expression.NewConstant(value.NewValue(yyS[yypt-0].n))
		}
	case 266:
		//line n1ql.y:1890
		{
			yyVAL.expr = expression.NewConstant(value.NewValue(yyS[yypt-0].s))
		}
	case 267:
		yyVAL.expr = yyS[yypt-0].expr
	case 268:
		yyVAL.expr = yyS[yypt-0].expr
	case 269:
		//line n1ql.y:1910
		{
			yyVAL.expr = expression.NewObjectConstruct(yyS[yypt-1].bindings)
		}
	case 270:
		//line n1ql.y:1917
		{
			yyVAL.bindings = nil
		}
	case 271:
		yyVAL.bindings = yyS[yypt-0].bindings
	case 272:
		//line n1ql.y:1926
		{
			yyVAL.bindings = expression.Bindings{yyS[yypt-0].binding}
		}
	case 273:
		//line n1ql.y:1931
		{
			yyVAL.bindings = append(yyS[yypt-2].bindings, yyS[yypt-0].binding)
		}
	case 274:
		//line n1ql.y:1938
		{
			yyVAL.binding = expression.NewBinding(yyS[yypt-2].s, yyS[yypt-0].expr)
		}
	case 275:
		//line n1ql.y:1945
		{
			yyVAL.expr = expression.NewArrayConstruct(yyS[yypt-1].exprs...)
		}
	case 276:
		//line n1ql.y:1952
		{
			yyVAL.exprs = nil
		}
	case 277:
		yyVAL.exprs = yyS[yypt-0].exprs
	case 278:
		//line n1ql.y:1968
		{
			yyVAL.expr = algebra.NewNamedParameter(yyS[yypt-0].s)
		}
	case 279:
		//line n1ql.y:1973
		{
			yyVAL.expr = algebra.NewPositionalParameter(yyS[yypt-0].n)
		}
	case 280:
		//line n1ql.y:1987
		{
			yyVAL.expr = yyS[yypt-1].expr
		}
	case 281:
		yyVAL.expr = yyS[yypt-0].expr
	case 282:
		yyVAL.expr = yyS[yypt-0].expr
	case 283:
		//line n1ql.y:2000
		{
			yyVAL.expr = expression.NewSimpleCase(yyS[yypt-2].expr, yyS[yypt-1].whenTerms, yyS[yypt-0].expr)
		}
	case 284:
		//line n1ql.y:2007
		{
			yyVAL.whenTerms = expression.WhenTerms{&expression.WhenTerm{yyS[yypt-2].expr, yyS[yypt-0].expr}}
		}
	case 285:
		//line n1ql.y:2012
		{
			yyVAL.whenTerms = append(yyS[yypt-4].whenTerms, &expression.WhenTerm{yyS[yypt-2].expr, yyS[yypt-0].expr})
		}
	case 286:
		//line n1ql.y:2020
		{
			yyVAL.expr = expression.NewSearchedCase(yyS[yypt-1].whenTerms, yyS[yypt-0].expr)
		}
	case 287:
		//line n1ql.y:2027
		{
			yyVAL.expr = nil
		}
	case 288:
		//line n1ql.y:2032
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 289:
		//line n1ql.y:2046
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
		//line n1ql.y:2065
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
		//line n1ql.y:2080
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
		//line n1ql.y:2114
		{
			yyVAL.expr = expression.NewAny(yyS[yypt-2].bindings, yyS[yypt-1].expr)
		}
	case 296:
		//line n1ql.y:2119
		{
			yyVAL.expr = expression.NewAny(yyS[yypt-2].bindings, yyS[yypt-1].expr)
		}
	case 297:
		//line n1ql.y:2124
		{
			yyVAL.expr = expression.NewEvery(yyS[yypt-2].bindings, yyS[yypt-1].expr)
		}
	case 298:
		//line n1ql.y:2131
		{
			yyVAL.bindings = expression.Bindings{yyS[yypt-0].binding}
		}
	case 299:
		//line n1ql.y:2136
		{
			yyVAL.bindings = append(yyS[yypt-2].bindings, yyS[yypt-0].binding)
		}
	case 300:
		//line n1ql.y:2143
		{
			yyVAL.binding = expression.NewBinding(yyS[yypt-2].s, yyS[yypt-0].expr)
		}
	case 301:
		//line n1ql.y:2148
		{
			yyVAL.binding = expression.NewDescendantBinding(yyS[yypt-2].s, yyS[yypt-0].expr)
		}
	case 302:
		//line n1ql.y:2155
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 303:
		//line n1ql.y:2162
		{
			yyVAL.expr = expression.NewArray(yyS[yypt-4].expr, yyS[yypt-2].bindings, yyS[yypt-1].expr)
		}
	case 304:
		//line n1ql.y:2167
		{
			yyVAL.expr = expression.NewFirst(yyS[yypt-4].expr, yyS[yypt-2].bindings, yyS[yypt-1].expr)
		}
	case 305:
		//line n1ql.y:2181
		{
			yyVAL.expr = yyS[yypt-1].expr
		}
	case 306:
		yyVAL.expr = yyS[yypt-0].expr
	case 307:
		//line n1ql.y:2190
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
