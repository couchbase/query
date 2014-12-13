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
	165, 306,
	-2, 252,
	-1, 116,
	173, 69,
	-2, 70,
	-1, 153,
	53, 78,
	71, 78,
	90, 78,
	142, 78,
	-2, 56,
	-1, 180,
	175, 0,
	176, 0,
	177, 0,
	-2, 216,
	-1, 181,
	175, 0,
	176, 0,
	177, 0,
	-2, 217,
	-1, 182,
	175, 0,
	176, 0,
	177, 0,
	-2, 218,
	-1, 183,
	178, 0,
	179, 0,
	180, 0,
	181, 0,
	-2, 219,
	-1, 184,
	178, 0,
	179, 0,
	180, 0,
	181, 0,
	-2, 220,
	-1, 185,
	178, 0,
	179, 0,
	180, 0,
	181, 0,
	-2, 221,
	-1, 186,
	178, 0,
	179, 0,
	180, 0,
	181, 0,
	-2, 222,
	-1, 193,
	79, 0,
	-2, 225,
	-1, 194,
	61, 0,
	157, 0,
	-2, 227,
	-1, 195,
	61, 0,
	157, 0,
	-2, 229,
	-1, 296,
	79, 0,
	-2, 226,
	-1, 297,
	61, 0,
	157, 0,
	-2, 228,
	-1, 298,
	61, 0,
	157, 0,
	-2, 230,
}

const yyNprod = 322
const yyPrivate = 57344

var yyTokenNames []string
var yyStates []string

const yyLast = 2611

var yyAct = []int{

	167, 3, 603, 592, 462, 601, 593, 325, 326, 310,
	523, 542, 98, 99, 551, 483, 321, 224, 418, 557,
	329, 225, 142, 516, 272, 476, 435, 241, 363, 420,
	103, 226, 417, 159, 220, 162, 115, 318, 138, 444,
	16, 266, 72, 154, 360, 406, 140, 114, 163, 265,
	141, 135, 236, 290, 58, 122, 207, 452, 126, 273,
	347, 345, 367, 474, 139, 545, 494, 493, 146, 147,
	364, 546, 276, 478, 92, 452, 451, 171, 172, 173,
	174, 175, 176, 177, 178, 179, 180, 181, 182, 183,
	184, 185, 186, 346, 451, 193, 194, 195, 127, 275,
	244, 288, 274, 255, 290, 62, 250, 97, 223, 366,
	155, 288, 144, 145, 248, 78, 475, 76, 400, 139,
	287, 156, 95, 76, 78, 238, 291, 292, 293, 473,
	287, 97, 79, 80, 81, 288, 75, 436, 436, 187,
	94, 401, 75, 252, 249, 251, 519, 485, 78, 289,
	291, 292, 293, 254, 287, 262, 539, 254, 168, 169,
	210, 212, 214, 280, 252, 337, 170, 335, 245, 245,
	239, 283, 118, 496, 497, 286, 390, 391, 540, 246,
	246, 384, 267, 378, 392, 282, 288, 227, 452, 256,
	143, 296, 297, 298, 277, 279, 278, 76, 290, 294,
	289, 291, 292, 293, 242, 287, 76, 451, 447, 312,
	313, 77, 79, 80, 81, 332, 75, 319, 96, 82,
	77, 79, 80, 81, 228, 75, 168, 169, 116, 264,
	76, 10, 336, 247, 170, 323, 339, 116, 340, 88,
	89, 90, 91, 82, 77, 79, 80, 81, 157, 75,
	328, 349, 576, 350, 324, 290, 353, 354, 355, 333,
	331, 264, 157, 309, 237, 365, 314, 602, 315, 521,
	316, 597, 543, 484, 541, 368, 525, 116, 222, 382,
	288, 425, 334, 148, 582, 257, 388, 338, 73, 393,
	376, 116, 377, 294, 289, 291, 292, 293, 383, 287,
	327, 348, 253, 616, 615, 352, 611, 358, 359, 583,
	189, 379, 380, 572, 136, 74, 73, 328, 123, 464,
	487, 330, 131, 409, 188, 381, 215, 129, 565, 295,
	373, 412, 414, 415, 413, 375, 213, 288, 228, 552,
	520, 563, 428, 137, 107, 480, 113, 421, 369, 423,
	294, 289, 291, 292, 293, 344, 287, 74, 191, 209,
	311, 407, 343, 342, 411, 130, 106, 370, 442, 410,
	128, 155, 449, 432, 235, 434, 190, 73, 424, 245,
	245, 245, 156, 433, 408, 74, 437, 73, 606, 613,
	246, 246, 246, 457, 581, 607, 267, 109, 267, 211,
	455, 609, 319, 438, 453, 454, 445, 445, 441, 466,
	448, 450, 465, 443, 440, 467, 468, 446, 446, 372,
	470, 188, 469, 479, 471, 472, 389, 208, 482, 394,
	395, 396, 397, 398, 399, 268, 612, 489, 105, 461,
	139, 234, 209, 429, 430, 431, 74, 258, 259, 573,
	73, 208, 150, 498, 192, 578, 74, 217, 218, 219,
	504, 561, 230, 362, 229, 206, 481, 495, 562, 270,
	492, 499, 500, 422, 508, 513, 510, 511, 491, 271,
	509, 322, 117, 364, 111, 71, 524, 110, 619, 618,
	594, 243, 240, 132, 421, 550, 73, 518, 506, 112,
	507, 517, 555, 490, 488, 514, 512, 357, 535, 356,
	528, 351, 233, 614, 536, 577, 439, 216, 188, 74,
	527, 188, 188, 188, 188, 188, 188, 374, 371, 2,
	529, 530, 49, 1, 532, 533, 522, 575, 486, 537,
	460, 544, 538, 100, 101, 564, 589, 524, 102, 596,
	477, 567, 560, 547, 553, 554, 419, 566, 152, 558,
	558, 559, 517, 556, 416, 571, 515, 463, 505, 320,
	83, 569, 570, 568, 41, 40, 92, 524, 587, 588,
	574, 39, 22, 21, 579, 580, 584, 586, 20, 590,
	591, 585, 19, 18, 595, 604, 17, 598, 600, 599,
	605, 9, 83, 8, 7, 227, 608, 6, 92, 5,
	4, 610, 402, 403, 308, 317, 104, 108, 617, 604,
	604, 621, 622, 620, 95, 158, 549, 548, 526, 361,
	263, 149, 188, 97, 83, 221, 269, 151, 153, 69,
	92, 70, 94, 33, 125, 32, 53, 28, 56, 55,
	78, 31, 121, 120, 93, 119, 95, 30, 133, 134,
	27, 84, 50, 24, 23, 97, 0, 0, 0, 0,
	0, 0, 0, 0, 94, 0, 0, 0, 0, 0,
	0, 0, 78, 0, 0, 0, 93, 0, 95, 0,
	0, 0, 0, 84, 0, 0, 0, 97, 0, 0,
	0, 0, 0, 0, 0, 0, 94, 0, 0, 0,
	0, 0, 0, 0, 78, 0, 0, 0, 93, 0,
	96, 0, 0, 0, 0, 84, 0, 0, 0, 0,
	0, 0, 76, 501, 502, 0, 0, 0, 85, 86,
	87, 88, 89, 90, 91, 82, 77, 79, 80, 81,
	0, 75, 96, 0, 0, 0, 228, 0, 0, 0,
	0, 0, 83, 0, 76, 0, 404, 0, 92, 0,
	85, 86, 87, 88, 89, 90, 91, 82, 77, 79,
	80, 81, 0, 75, 96, 0, 0, 203, 0, 0,
	0, 405, 205, 200, 0, 83, 76, 458, 0, 0,
	459, 92, 85, 86, 87, 88, 89, 90, 91, 82,
	77, 79, 80, 81, 0, 75, 95, 0, 0, 0,
	0, 0, 0, 0, 0, 97, 0, 83, 0, 0,
	0, 0, 0, 92, 94, 0, 0, 0, 0, 0,
	0, 0, 78, 0, 0, 0, 93, 0, 0, 95,
	0, 0, 0, 84, 0, 0, 0, 0, 97, 0,
	0, 0, 0, 0, 0, 0, 198, 94, 0, 197,
	196, 201, 204, 0, 0, 78, 0, 0, 0, 93,
	0, 95, 0, 0, 0, 0, 84, 0, 0, 0,
	97, 0, 0, 0, 0, 0, 0, 0, 0, 94,
	0, 0, 0, 0, 0, 92, 0, 78, 0, 202,
	0, 93, 96, 0, 0, 0, 0, 0, 84, 0,
	0, 0, 0, 0, 76, 0, 0, 0, 199, 0,
	85, 86, 87, 88, 89, 90, 91, 82, 77, 79,
	80, 81, 0, 75, 0, 96, 0, 0, 0, 0,
	0, 0, 0, 95, 0, 0, 0, 76, 385, 386,
	0, 0, 97, 85, 86, 87, 88, 89, 90, 91,
	82, 77, 79, 80, 81, 83, 75, 96, 227, 78,
	0, 92, 0, 0, 0, 0, 305, 0, 0, 76,
	284, 307, 302, 285, 0, 85, 86, 87, 88, 89,
	90, 91, 82, 77, 79, 80, 81, 0, 75, 0,
	83, 0, 0, 0, 0, 0, 92, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 95,
	0, 0, 0, 0, 0, 0, 0, 0, 97, 0,
	0, 0, 83, 0, 0, 0, 0, 94, 92, 96,
	0, 0, 0, 0, 0, 78, 0, 0, 0, 93,
	0, 76, 0, 0, 95, 300, 84, 0, 0, 299,
	303, 306, 0, 97, 82, 77, 79, 80, 81, 0,
	75, 0, 94, 0, 0, 0, 0, 0, 0, 478,
	78, 0, 0, 0, 93, 0, 95, 0, 0, 0,
	0, 84, 0, 0, 0, 97, 0, 0, 304, 0,
	0, 0, 0, 0, 94, 0, 0, 0, 0, 0,
	0, 0, 78, 0, 0, 96, 93, 301, 0, 228,
	0, 0, 0, 84, 0, 0, 0, 76, 0, 0,
	0, 0, 0, 85, 86, 87, 88, 89, 90, 91,
	82, 77, 79, 80, 81, 0, 281, 264, 0, 0,
	96, 0, 0, 0, 0, 0, 0, 0, 83, 0,
	0, 0, 76, 0, 92, 0, 0, 0, 85, 86,
	87, 88, 89, 90, 91, 82, 77, 79, 80, 81,
	0, 75, 96, 0, 0, 0, 0, 0, 0, 0,
	83, 0, 0, 0, 76, 0, 92, 0, 0, 0,
	85, 86, 87, 88, 89, 90, 91, 82, 77, 79,
	80, 81, 95, 75, 0, 0, 0, 0, 0, 0,
	0, 97, 83, 0, 0, 0, 0, 0, 92, 0,
	94, 0, 0, 0, 0, 0, 0, 61, 78, 0,
	0, 0, 93, 0, 95, 0, 0, 0, 0, 84,
	0, 0, 0, 97, 0, 0, 0, 0, 0, 0,
	59, 0, 94, 0, 0, 0, 36, 0, 0, 0,
	78, 0, 60, 0, 93, 0, 95, 0, 0, 0,
	15, 84, 13, 0, 0, 97, 0, 73, 0, 0,
	0, 0, 0, 0, 94, 0, 0, 0, 0, 34,
	0, 0, 78, 0, 0, 0, 93, 0, 96, 0,
	0, 0, 0, 84, 0, 0, 0, 0, 38, 0,
	76, 534, 0, 0, 0, 0, 85, 86, 87, 88,
	89, 90, 91, 82, 77, 79, 80, 81, 14, 75,
	96, 0, 0, 0, 0, 0, 0, 0, 83, 0,
	0, 0, 76, 531, 92, 0, 74, 0, 85, 86,
	87, 88, 89, 90, 91, 82, 77, 79, 80, 81,
	0, 75, 96, 0, 0, 0, 37, 35, 0, 0,
	83, 0, 0, 0, 76, 456, 92, 0, 0, 0,
	85, 86, 87, 88, 89, 90, 91, 82, 77, 79,
	80, 81, 95, 75, 0, 0, 0, 0, 0, 0,
	0, 97, 83, 0, 0, 0, 0, 0, 92, 0,
	94, 0, 0, 0, 0, 0, 0, 0, 78, 0,
	0, 0, 93, 0, 95, 0, 0, 0, 0, 84,
	0, 0, 0, 97, 0, 0, 0, 0, 0, 0,
	0, 0, 94, 0, 0, 0, 0, 0, 0, 0,
	78, 0, 0, 0, 93, 0, 95, 0, 0, 0,
	0, 84, 0, 0, 427, 97, 0, 0, 0, 0,
	0, 0, 0, 0, 94, 0, 0, 0, 0, 0,
	0, 0, 78, 0, 0, 0, 93, 0, 96, 0,
	0, 0, 0, 84, 0, 0, 0, 0, 0, 0,
	76, 0, 0, 0, 0, 0, 85, 86, 87, 88,
	89, 90, 91, 82, 77, 79, 80, 81, 0, 75,
	96, 0, 0, 64, 67, 0, 0, 0, 341, 426,
	0, 0, 76, 0, 0, 54, 0, 0, 85, 86,
	87, 88, 89, 90, 91, 82, 77, 79, 80, 81,
	83, 75, 96, 0, 0, 0, 92, 0, 0, 0,
	66, 0, 0, 0, 76, 0, 44, 68, 0, 0,
	85, 86, 87, 88, 89, 90, 91, 82, 77, 79,
	80, 81, 83, 75, 64, 67, 0, 0, 92, 0,
	0, 0, 0, 0, 0, 0, 54, 261, 0, 0,
	0, 0, 29, 43, 95, 0, 0, 42, 46, 0,
	0, 0, 0, 97, 0, 0, 0, 0, 83, 0,
	0, 66, 94, 0, 92, 12, 0, 44, 68, 260,
	78, 0, 0, 0, 93, 0, 95, 0, 0, 0,
	26, 84, 0, 65, 0, 97, 48, 0, 0, 0,
	0, 0, 45, 0, 94, 0, 0, 0, 0, 0,
	0, 0, 78, 29, 43, 0, 93, 11, 42, 46,
	0, 0, 95, 84, 0, 47, 25, 0, 51, 52,
	57, 97, 62, 0, 63, 0, 0, 0, 0, 0,
	94, 0, 0, 0, 0, 0, 0, 0, 78, 0,
	96, 26, 93, 0, 65, 0, 0, 48, 0, 84,
	0, 0, 76, 45, 0, 0, 0, 0, 85, 86,
	87, 88, 89, 90, 91, 82, 77, 79, 80, 81,
	0, 75, 96, 0, 0, 0, 47, 25, 0, 51,
	52, 57, 0, 62, 76, 63, 503, 0, 0, 0,
	85, 86, 87, 88, 89, 90, 91, 82, 77, 79,
	80, 81, 0, 75, 124, 0, 161, 0, 96, 0,
	64, 67, 0, 0, 0, 0, 0, 0, 0, 0,
	76, 0, 54, 0, 0, 0, 85, 86, 87, 88,
	89, 90, 91, 82, 77, 79, 80, 81, 0, 75,
	160, 0, 0, 0, 165, 0, 0, 66, 0, 0,
	0, 12, 83, 44, 68, 0, 0, 0, 92, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 29,
	43, 0, 0, 11, 42, 46, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 95, 0, 0, 0,
	0, 0, 0, 0, 164, 97, 0, 0, 0, 0,
	0, 0, 0, 0, 94, 0, 0, 26, 0, 0,
	65, 0, 78, 48, 0, 0, 93, 0, 0, 45,
	0, 0, 0, 84, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 64, 67, 0, 0,
	0, 0, 47, 25, 0, 51, 52, 57, 54, 62,
	0, 63, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 83, 0, 0, 166, 0, 0, 92,
	165, 0, 0, 66, 0, 0, 0, 12, 0, 44,
	68, 0, 96, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 76, 0, 0, 0, 0, 0,
	85, 86, 87, 88, 89, 90, 91, 82, 77, 79,
	80, 81, 92, 75, 0, 29, 43, 95, 0, 11,
	42, 46, 0, 0, 0, 0, 97, 0, 0, 0,
	0, 0, 0, 0, 0, 94, 0, 0, 64, 67,
	164, 0, 0, 78, 0, 0, 0, 93, 0, 0,
	54, 0, 0, 26, 92, 0, 65, 0, 0, 48,
	95, 0, 0, 0, 0, 45, 0, 0, 231, 97,
	0, 0, 0, 0, 0, 66, 0, 0, 94, 12,
	0, 44, 68, 0, 0, 0, 78, 0, 47, 25,
	93, 51, 52, 57, 0, 62, 0, 63, 0, 0,
	0, 0, 95, 0, 0, 0, 0, 0, 0, 0,
	0, 97, 166, 96, 0, 0, 0, 29, 43, 0,
	94, 11, 42, 46, 0, 76, 0, 0, 78, 0,
	0, 85, 86, 87, 88, 89, 90, 91, 82, 77,
	79, 80, 81, 0, 75, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 26, 96, 0, 65, 64,
	67, 48, 0, 0, 0, 0, 0, 45, 76, 0,
	0, 54, 0, 0, 85, 86, 87, 88, 89, 90,
	91, 82, 77, 79, 80, 81, 0, 75, 0, 0,
	47, 25, 0, 51, 52, 57, 66, 62, 96, 63,
	12, 0, 44, 68, 0, 0, 0, 0, 0, 0,
	76, 0, 0, 0, 232, 0, 0, 0, 61, 0,
	0, 64, 67, 82, 77, 79, 80, 81, 0, 75,
	0, 0, 0, 54, 0, 0, 0, 0, 29, 43,
	0, 59, 11, 42, 46, 0, 0, 36, 0, 0,
	0, 0, 0, 60, 0, 0, 0, 0, 66, 0,
	0, 15, 12, 13, 44, 68, 0, 0, 73, 0,
	0, 0, 0, 0, 0, 0, 26, 0, 0, 65,
	34, 0, 48, 0, 64, 67, 0, 0, 45, 0,
	0, 0, 0, 0, 0, 0, 54, 0, 0, 38,
	29, 43, 0, 0, 11, 42, 46, 0, 0, 0,
	0, 47, 25, 0, 51, 52, 57, 0, 62, 14,
	63, 66, 0, 0, 0, 12, 0, 44, 68, 0,
	0, 0, 0, 0, 0, 166, 0, 74, 26, 0,
	0, 65, 64, 67, 48, 0, 0, 0, 0, 0,
	45, 0, 0, 0, 54, 0, 0, 37, 35, 0,
	0, 0, 0, 29, 43, 0, 0, 11, 42, 46,
	0, 0, 0, 47, 25, 0, 51, 52, 57, 66,
	62, 0, 63, 12, 0, 44, 68, 0, 0, 73,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 26, 0, 0, 65, 0, 0, 48, 0, 0,
	0, 0, 0, 45, 0, 0, 0, 0, 0, 0,
	0, 29, 43, 0, 0, 11, 42, 46, 0, 0,
	0, 0, 64, 67, 0, 0, 47, 25, 0, 51,
	52, 57, 0, 62, 54, 63, 387, 0, 0, 64,
	67, 0, 0, 0, 0, 0, 0, 0, 74, 26,
	0, 54, 65, 0, 0, 48, 0, 0, 0, 66,
	0, 45, 0, 12, 0, 44, 68, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 66, 0, 0, 0,
	12, 0, 44, 68, 47, 25, 0, 51, 52, 57,
	0, 62, 0, 63, 0, 0, 0, 0, 0, 0,
	0, 29, 43, 0, 0, 11, 42, 46, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 29, 43,
	0, 0, 11, 42, 46, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 26,
	0, 0, 65, 0, 0, 48, 0, 0, 0, 0,
	0, 45, 0, 0, 0, 0, 26, 0, 0, 65,
	0, 0, 48, 0, 0, 0, 0, 124, 45, 0,
	0, 0, 0, 0, 47, 25, 0, 51, 52, 57,
	0, 62, 0, 63, 0, 0, 0, 0, 0, 0,
	0, 47, 25, 0, 51, 52, 57, 0, 62, 0,
	63,
}
var yyPact = []int{

	2213, -1000, -1000, 1825, -1000, -1000, -1000, -1000, -1000, -1000,
	-1000, 2441, 2441, 1242, 1242, -62, -1000, -1000, -1000, -1000,
	-1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000, 2441,
	-1000, -1000, -1000, 298, 418, 415, 444, 67, 413, -1000,
	-1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000,
	-1000, -1000, -1000, 7, 2424, -1000, -1000, 2334, -1000, 263,
	258, 429, 183, 2441, 29, 29, 29, 2441, 2441, -1000,
	-1000, 375, 441, 130, 1782, 65, 2441, 2441, 2441, 2441,
	2441, 2441, 2441, 2441, 2441, 2441, 2441, 2441, 2441, 2441,
	2441, 2441, 1535, 297, 2441, 2441, 2441, 778, 1999, 37,
	-1000, -1000, -1000, -46, 347, 395, 332, 322, -1000, 499,
	67, 67, 67, 132, -65, 177, -1000, 67, 2030, 468,
	-1000, -1000, 1631, 221, 2441, 4, 1825, -1000, 428, 43,
	427, 67, 67, -54, -28, -1000, -67, -25, -29, 1825,
	-19, -1000, 128, -1000, -19, -19, 1595, 1563, 75, -1000,
	63, 375, -1000, 403, -1000, -129, -71, -74, -1000, -100,
	1928, 2151, 2441, -1000, -1000, -1000, -1000, 968, -1000, -1000,
	2441, 820, -52, -52, -46, -46, -46, 28, 1999, 1956,
	61, 61, 61, 2041, 2041, 2041, 2041, 168, -1000, 1535,
	2441, 2441, 2441, 892, 37, 37, -1000, 977, -1000, -1000,
	-1000, -1000, -1000, -1000, -1000, -1000, 264, 371, 2441, 2441,
	-1000, 261, -1000, 261, -1000, 261, 2441, 409, 409, 132,
	174, -1000, 214, 54, -1000, -1000, -1000, 63, -1000, 134,
	1, 2441, -1, -1000, 221, 2441, -1000, 2441, 1415, -1000,
	267, 266, -1000, 259, -127, -1000, -80, -128, -1000, 183,
	2441, -1000, 2441, 467, 29, 2441, 2441, 2441, 465, 463,
	29, 29, 405, -1000, 2441, -63, -1000, -113, 75, 277,
	-1000, 235, 177, 22, 54, 54, 2151, -100, 2441, -100,
	595, -3, -1000, 788, -1000, 2276, 1535, 15, 2441, 1535,
	1535, 1535, 1535, 1535, 1535, 111, 892, 37, 37, -1000,
	-1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000,
	-1000, -1000, 1825, 1825, -1000, -1000, -1000, -31, -1000, 755,
	233, -1000, 2441, 233, 75, 107, 75, 22, 22, 400,
	-1000, 177, -1000, -1000, 116, -1000, 1383, -1000, -1000, 1351,
	1825, 2441, 67, 67, 67, 43, 54, 43, -1000, 1825,
	1825, -1000, -1000, 1825, 1825, 1825, -1000, -1000, -15, -15,
	191, -1000, 498, -1000, 63, 1825, 63, 2441, 405, 76,
	76, 2441, -1000, -1000, -1000, -1000, 132, -94, -1000, -129,
	-129, -1000, 595, -1000, -1000, -1000, -1000, -1000, 1225, 17,
	-1000, -1000, 2441, 627, -58, -58, -68, -68, -68, -34,
	1535, 2441, -1000, -1000, -1000, -1000, 202, 202, 2441, 1825,
	202, 202, 371, 75, 371, 371, -43, -1000, -112, -56,
	-1000, 19, 2441, -1000, 249, 261, -1000, 2441, 1825, 125,
	-18, -1000, -1000, -1000, 205, 460, 2441, 459, -1000, 2441,
	-63, -1000, 1825, -1000, -1000, -129, -106, -107, -1000, 595,
	-1000, 12, 2441, 177, 177, -1000, -1000, 563, -1000, 1596,
	17, -1000, -1000, -1000, 1928, -1000, 1825, -1000, -1000, 202,
	371, 202, 202, 22, 2441, 22, -1000, -1000, 29, 1825,
	409, -20, 1825, -1000, 188, 2441, -1000, 142, -1000, 1825,
	-1000, -8, 177, 54, 54, -1000, -1000, -1000, 1193, 132,
	132, -1000, -1000, -1000, 1161, -1000, -100, 2441, -1000, 202,
	-1000, -1000, -1000, 1035, -1000, -16, -1000, 117, 119, 177,
	-1000, -1000, -101, -1000, 1825, 43, 436, -1000, 243, -129,
	-129, -1000, -1000, -1000, -1000, 1825, -1000, -1000, 458, 29,
	22, 22, 371, 377, 245, 226, 2441, -1000, -1000, -1000,
	2441, -1000, 214, 177, 177, -1000, -1000, -1000, -94, -1000,
	202, 180, 365, 409, 98, 497, -1000, 1825, 382, 243,
	243, -1000, 250, 176, 119, 125, 2441, 2441, 2441, -1000,
	-1000, 174, 75, 423, 371, -1000, -1000, 1825, 1825, 118,
	107, 75, 114, -1000, 2441, 202, -1000, 304, -1000, 75,
	-1000, -1000, 310, -1000, 1003, -1000, 173, 352, -1000, 305,
	-1000, 479, 171, 170, 75, 422, 421, 114, 2441, 2441,
	-1000, -1000, -1000,
}
var yyPgo = []int{

	0, 664, 663, 532, 662, 660, 51, 659, 658, 0,
	231, 139, 38, 343, 41, 49, 31, 21, 17, 22,
	657, 655, 653, 652, 52, 318, 651, 649, 648, 50,
	46, 302, 26, 647, 646, 645, 644, 40, 643, 54,
	641, 639, 638, 485, 637, 43, 39, 636, 18, 24,
	47, 36, 635, 34, 14, 283, 631, 6, 630, 44,
	629, 628, 28, 627, 626, 48, 33, 625, 42, 617,
	616, 37, 615, 360, 9, 56, 614, 613, 612, 529,
	610, 609, 607, 604, 603, 601, 596, 593, 592, 588,
	583, 582, 581, 575, 574, 346, 45, 16, 569, 568,
	567, 4, 23, 566, 19, 7, 32, 564, 8, 29,
	556, 550, 25, 11, 549, 546, 3, 2, 5, 27,
	100, 545, 15, 538, 10, 537, 536, 533, 35, 528,
	20, 527,
}
var yyR1 = []int{

	0, 127, 127, 79, 79, 79, 79, 79, 79, 80,
	81, 82, 83, 84, 84, 84, 84, 84, 85, 91,
	91, 91, 37, 37, 37, 38, 38, 38, 38, 38,
	38, 38, 39, 39, 41, 40, 68, 67, 67, 67,
	67, 67, 128, 128, 66, 66, 65, 65, 65, 18,
	18, 17, 17, 16, 44, 44, 43, 42, 42, 42,
	42, 129, 129, 45, 45, 45, 46, 46, 46, 50,
	51, 49, 49, 53, 53, 52, 130, 130, 47, 47,
	47, 131, 131, 54, 55, 55, 56, 15, 15, 14,
	57, 57, 58, 59, 59, 60, 60, 12, 12, 61,
	61, 62, 63, 63, 64, 70, 70, 69, 72, 72,
	71, 78, 78, 77, 77, 74, 74, 73, 76, 76,
	75, 86, 86, 95, 95, 98, 98, 97, 96, 101,
	101, 100, 99, 99, 87, 87, 88, 89, 89, 89,
	105, 107, 107, 106, 112, 112, 111, 103, 103, 102,
	102, 19, 104, 32, 32, 108, 110, 110, 109, 90,
	90, 113, 113, 113, 113, 114, 114, 114, 118, 118,
	115, 115, 115, 116, 117, 92, 92, 119, 120, 120,
	121, 121, 122, 122, 122, 126, 126, 124, 125, 125,
	93, 93, 94, 123, 123, 48, 48, 48, 48, 9,
	9, 9, 9, 9, 9, 9, 9, 9, 9, 9,
	9, 9, 9, 9, 9, 9, 9, 9, 9, 9,
	9, 9, 9, 9, 9, 9, 9, 9, 9, 9,
	9, 9, 9, 9, 9, 9, 9, 9, 9, 9,
	9, 9, 9, 9, 9, 9, 9, 9, 9, 9,
	10, 10, 10, 10, 10, 10, 10, 10, 10, 10,
	11, 11, 11, 11, 11, 11, 11, 11, 11, 11,
	11, 11, 11, 11, 1, 1, 1, 1, 1, 1,
	1, 2, 2, 3, 8, 8, 7, 7, 6, 4,
	13, 13, 5, 5, 20, 21, 21, 22, 25, 25,
	23, 24, 24, 33, 33, 33, 34, 26, 26, 27,
	27, 27, 30, 30, 29, 29, 31, 28, 28, 35,
	36, 36,
}
var yyR2 = []int{

	0, 1, 1, 1, 1, 1, 1, 1, 1, 2,
	2, 2, 1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 2, 4, 4, 1, 3, 4, 3, 4,
	3, 4, 1, 1, 5, 5, 2, 1, 2, 2,
	3, 4, 1, 1, 1, 3, 1, 3, 2, 0,
	1, 1, 2, 1, 0, 1, 2, 1, 4, 4,
	5, 1, 1, 4, 6, 6, 4, 6, 6, 1,
	1, 0, 2, 0, 1, 4, 0, 1, 0, 1,
	2, 0, 1, 4, 0, 1, 2, 1, 3, 3,
	0, 1, 2, 0, 1, 5, 1, 1, 3, 0,
	1, 2, 0, 1, 2, 0, 1, 3, 1, 3,
	2, 0, 1, 1, 1, 0, 1, 2, 0, 1,
	2, 6, 6, 4, 2, 0, 1, 2, 2, 0,
	1, 2, 1, 2, 6, 6, 7, 8, 7, 7,
	2, 1, 3, 4, 0, 1, 4, 1, 3, 3,
	3, 1, 1, 0, 2, 2, 1, 3, 2, 10,
	13, 0, 6, 6, 6, 0, 6, 6, 0, 6,
	2, 3, 2, 1, 2, 6, 11, 1, 1, 3,
	0, 3, 0, 2, 2, 1, 3, 1, 0, 2,
	5, 5, 6, 0, 3, 1, 3, 3, 4, 1,
	3, 3, 5, 5, 4, 5, 6, 3, 3, 3,
	3, 3, 3, 3, 3, 2, 3, 3, 3, 3,
	3, 3, 3, 5, 6, 3, 4, 3, 4, 3,
	4, 3, 4, 3, 4, 3, 4, 3, 4, 3,
	4, 3, 4, 3, 4, 3, 4, 3, 4, 2,
	1, 1, 1, 1, 1, 1, 2, 1, 1, 1,
	1, 3, 3, 5, 5, 4, 5, 6, 3, 3,
	3, 3, 3, 3, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 3, 0, 1, 1, 3, 3, 3,
	0, 1, 1, 1, 3, 1, 1, 3, 4, 5,
	2, 0, 2, 4, 5, 4, 1, 1, 1, 4,
	4, 4, 1, 3, 3, 3, 2, 6, 6, 3,
	1, 1,
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
	15, 93, 131, 9, 94, 14, -73, -75, 80, 95,
	-39, 4, -39, 4, -39, 4, 18, -95, -95, -95,
	-53, -52, 146, 173, -18, -17, -16, 10, 161, -95,
	-13, 38, 184, 44, -25, 153, -24, 43, -9, 166,
	64, -119, 161, 64, -120, -51, -50, -120, 168, 172,
	173, 170, 172, -31, 172, 122, 61, 157, -31, -31,
	54, 54, -57, -58, 154, -15, -14, -16, -55, -47,
	66, 76, -49, 188, 173, 173, 172, -66, -128, -66,
	-9, 188, -18, -9, 170, 173, 7, 188, 169, 183,
	87, 184, 185, 186, 182, -11, -9, -9, -9, 92,
	88, 150, 15, 93, 131, 9, 94, 14, -76, -75,
	-74, -73, -9, -9, -39, -39, -39, -72, -71, -9,
	-98, -97, 72, -97, -53, -105, -108, 126, 143, -130,
	107, -51, 161, -16, 148, 166, -9, 166, -24, -9,
	-9, 133, 96, 96, 96, 188, 173, 188, -6, -9,
	-9, 44, -29, -9, -9, -9, 44, 44, -30, -30,
	-59, -60, 58, -62, 78, -9, 172, 175, -57, 71,
	90, -129, 142, 53, -131, 100, -18, -48, 161, -51,
	-51, -65, -9, -18, 184, 170, 171, 170, -9, -11,
	161, 162, 169, -9, -11, -11, -11, -11, -11, -11,
	7, 172, -78, -77, 11, 36, -96, -37, 151, -9,
	-96, -37, -57, -108, -57, -57, -107, -106, -48, -110,
	-109, -48, 73, -18, -45, 165, 166, 133, -9, -120,
	-120, -120, -119, -51, -119, -32, 153, -32, -68, 18,
	-15, -14, -9, -59, -46, -51, -50, 132, -46, -9,
	-53, 188, 169, -49, -49, -18, 170, -9, 170, 173,
	-11, -71, -101, -100, 117, -101, -9, -101, -101, -74,
	-57, -74, -74, 172, 175, 172, -112, -111, 54, -9,
	96, -37, -9, -122, 148, 165, -123, 115, 44, -9,
	44, -12, -49, 173, 173, -18, 161, 162, -9, -18,
	-18, 170, 171, 170, -9, -99, -66, -128, -101, -74,
	-101, -101, -106, -9, -109, -103, -102, -19, -97, 166,
	152, 81, -126, -124, -9, 134, -61, -62, -18, -51,
	-51, 170, -53, -53, 170, -9, -101, -112, -32, 172,
	61, 157, -113, 153, -17, 166, 172, -119, -63, -64,
	59, -54, 96, -49, -49, 44, -102, -104, -48, -104,
	-74, 84, 91, 96, -121, 102, -124, -9, -130, -18,
	-18, -101, 133, 84, -97, -125, 154, 18, 73, -54,
	-54, 144, 34, 133, -113, -122, -124, -9, -9, -115,
	-105, -108, -116, -57, 67, -74, -114, 153, -57, -108,
	-57, -118, 153, -117, -9, -101, 84, 91, -57, 91,
	-57, 133, 84, 84, 34, 133, 133, -116, 67, 67,
	-118, -117, -117,
}
var yyDef = []int{

	0, -2, 1, 2, 3, 4, 5, 6, 7, 8,
	199, 0, 0, 0, 0, 0, 12, 13, 14, 15,
	16, 17, 18, 250, 251, -2, 253, 254, 255, 0,
	257, 258, 259, 105, 0, 0, 0, 0, 0, 19,
	20, 21, 274, 275, 276, 277, 278, 279, 280, 281,
	282, 292, 293, 0, 0, 307, 308, 0, 25, 0,
	0, 0, 284, 290, 0, 0, 0, 0, 0, 32,
	33, 84, 54, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 215, 249,
	9, 10, 11, 256, 22, 0, 0, 0, 106, 0,
	0, 0, 0, 73, 0, 49, -2, 0, 290, 0,
	295, 296, 0, 301, 0, 0, 320, 321, 0, 0,
	0, 0, 0, 0, 285, 286, 0, 0, 291, 97,
	0, 312, 0, 151, 0, 0, 0, 0, 90, 85,
	0, 84, 55, -2, 57, 71, 0, 0, 36, 37,
	0, 0, 0, 44, 42, 43, 46, 49, 200, 201,
	0, 0, 207, 208, 209, 210, 211, 212, 213, 214,
	-2, -2, -2, -2, -2, -2, -2, 0, 260, 0,
	0, 0, 0, -2, -2, -2, 231, 0, 233, 235,
	237, 239, 241, 243, 245, 247, 118, 115, 0, 0,
	26, 0, 28, 0, 30, 0, 0, 125, 0, 73,
	0, 74, 76, 0, 124, 50, 51, 0, 53, 0,
	0, 0, 0, 294, 301, 0, 300, 0, 0, 319,
	0, 0, 177, 0, 0, 178, 0, 0, 283, 0,
	0, 289, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 93, 91, 0, 86, 87, 0, 90, 0,
	79, 81, 49, 0, 0, 0, 0, 38, 0, 39,
	49, 0, 48, 0, 204, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, -2, -2, -2, 232,
	234, 236, 238, 240, 242, 244, 246, 248, 23, 119,
	24, 116, 117, 120, 27, 29, 31, 107, 108, 111,
	0, 126, 0, 0, 90, 90, 90, 0, 0, 0,
	77, 49, 70, 52, 0, 303, 0, 305, 297, 0,
	302, 0, 0, 0, 0, 0, 0, 0, 287, 288,
	98, 309, 313, 316, 314, 315, 310, 311, 153, 153,
	0, 94, 0, 96, 0, 92, 0, 0, 93, 0,
	0, 0, 61, 62, 80, 82, 73, 72, 195, 71,
	71, 45, 49, 40, 47, 202, 203, 205, 0, 223,
	261, 262, 0, 0, 268, 269, 270, 271, 272, 273,
	0, 0, 110, 112, 113, 114, 129, 129, 0, 127,
	129, 129, 115, 90, 115, 115, 140, 141, 0, 155,
	156, 144, 0, 123, 0, 0, 304, 0, 298, 182,
	0, 190, 191, 179, 193, 0, 0, 0, 34, 0,
	101, 88, 89, 35, 58, 71, 0, 0, 59, 49,
	63, 0, 0, 49, 49, 41, 206, 0, 265, 0,
	224, 109, 121, 130, 0, 122, 128, 134, 135, 129,
	115, 129, 129, 0, 0, 0, 158, 145, 0, 75,
	0, 0, 299, 175, 0, 0, 192, 0, 317, 154,
	318, 99, 49, 0, 0, 60, 196, 197, 0, 73,
	73, 263, 264, 266, 0, 131, 132, 0, 136, 129,
	138, 139, 142, 144, 157, 153, 147, 0, 161, 0,
	183, 184, 0, 185, 187, 0, 102, 100, 0, 71,
	71, 198, 64, 65, 267, 133, 137, 143, 0, 0,
	0, 0, 115, 0, 0, 180, 0, 194, 95, 103,
	0, 66, 76, 49, 49, 146, 148, 149, 152, 150,
	129, 0, 0, 0, 188, 0, 186, 104, 0, 0,
	0, 159, 0, 0, 161, 182, 0, 0, 0, 67,
	68, 0, 90, 0, 115, 176, 189, 181, 83, 165,
	90, 90, 168, 173, 0, 129, 162, 0, 170, 90,
	172, 163, 0, 164, 90, 160, 0, 0, 171, 0,
	174, 0, 0, 0, 90, 0, 0, 168, 0, 0,
	166, 167, 169,
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
			yyVAL.fullselect = algebra.NewSelect(yyS[yypt-1].subresult, yyS[yypt-0].order, nil, nil) /* OFFSET precedes LIMIT */
		}
	case 23:
		//line n1ql.y:433
		{
			yyVAL.fullselect = algebra.NewSelect(yyS[yypt-3].subresult, yyS[yypt-2].order, yyS[yypt-0].expr, yyS[yypt-1].expr) /* OFFSET precedes LIMIT */
		}
	case 24:
		//line n1ql.y:437
		{
			yyVAL.fullselect = algebra.NewSelect(yyS[yypt-3].subresult, yyS[yypt-2].order, yyS[yypt-1].expr, yyS[yypt-0].expr) /* OFFSET precedes LIMIT */
		}
	case 25:
		//line n1ql.y:443
		{
			yyVAL.subresult = yyS[yypt-0].subselect
		}
	case 26:
		//line n1ql.y:448
		{
			yyVAL.subresult = algebra.NewUnion(yyS[yypt-2].subresult, yyS[yypt-0].subselect)
		}
	case 27:
		//line n1ql.y:453
		{
			yyVAL.subresult = algebra.NewUnionAll(yyS[yypt-3].subresult, yyS[yypt-0].subselect)
		}
	case 28:
		//line n1ql.y:458
		{
			yyVAL.subresult = algebra.NewIntersect(yyS[yypt-2].subresult, yyS[yypt-0].subselect)
		}
	case 29:
		//line n1ql.y:463
		{
			yyVAL.subresult = algebra.NewIntersectAll(yyS[yypt-3].subresult, yyS[yypt-0].subselect)
		}
	case 30:
		//line n1ql.y:468
		{
			yyVAL.subresult = algebra.NewExcept(yyS[yypt-2].subresult, yyS[yypt-0].subselect)
		}
	case 31:
		//line n1ql.y:473
		{
			yyVAL.subresult = algebra.NewExceptAll(yyS[yypt-3].subresult, yyS[yypt-0].subselect)
		}
	case 32:
		yyVAL.subselect = yyS[yypt-0].subselect
	case 33:
		yyVAL.subselect = yyS[yypt-0].subselect
	case 34:
		//line n1ql.y:486
		{
			yyVAL.subselect = algebra.NewSubselect(yyS[yypt-4].fromTerm, yyS[yypt-3].bindings, yyS[yypt-2].expr, yyS[yypt-1].group, yyS[yypt-0].projection)
		}
	case 35:
		//line n1ql.y:493
		{
			yyVAL.subselect = algebra.NewSubselect(yyS[yypt-3].fromTerm, yyS[yypt-2].bindings, yyS[yypt-1].expr, yyS[yypt-0].group, yyS[yypt-4].projection)
		}
	case 36:
		//line n1ql.y:508
		{
			yyVAL.projection = yyS[yypt-0].projection
		}
	case 37:
		//line n1ql.y:515
		{
			yyVAL.projection = algebra.NewProjection(false, yyS[yypt-0].resultTerms)
		}
	case 38:
		//line n1ql.y:520
		{
			yyVAL.projection = algebra.NewProjection(true, yyS[yypt-0].resultTerms)
		}
	case 39:
		//line n1ql.y:525
		{
			yyVAL.projection = algebra.NewProjection(false, yyS[yypt-0].resultTerms)
		}
	case 40:
		//line n1ql.y:530
		{
			yyVAL.projection = algebra.NewRawProjection(false, yyS[yypt-1].expr, yyS[yypt-0].s)
		}
	case 41:
		//line n1ql.y:535
		{
			yyVAL.projection = algebra.NewRawProjection(true, yyS[yypt-1].expr, yyS[yypt-0].s)
		}
	case 44:
		//line n1ql.y:548
		{
			yyVAL.resultTerms = algebra.ResultTerms{yyS[yypt-0].resultTerm}
		}
	case 45:
		//line n1ql.y:553
		{
			yyVAL.resultTerms = append(yyS[yypt-2].resultTerms, yyS[yypt-0].resultTerm)
		}
	case 46:
		//line n1ql.y:560
		{
			yyVAL.resultTerm = algebra.NewResultTerm(nil, true, "")
		}
	case 47:
		//line n1ql.y:565
		{
			yyVAL.resultTerm = algebra.NewResultTerm(yyS[yypt-2].expr, true, "")
		}
	case 48:
		//line n1ql.y:570
		{
			yyVAL.resultTerm = algebra.NewResultTerm(yyS[yypt-1].expr, false, yyS[yypt-0].s)
		}
	case 49:
		//line n1ql.y:577
		{
			yyVAL.s = ""
		}
	case 50:
		yyVAL.s = yyS[yypt-0].s
	case 51:
		yyVAL.s = yyS[yypt-0].s
	case 52:
		//line n1ql.y:588
		{
			yyVAL.s = yyS[yypt-0].s
		}
	case 53:
		yyVAL.s = yyS[yypt-0].s
	case 54:
		//line n1ql.y:606
		{
			yyVAL.fromTerm = nil
		}
	case 55:
		yyVAL.fromTerm = yyS[yypt-0].fromTerm
	case 56:
		//line n1ql.y:615
		{
			yyVAL.fromTerm = yyS[yypt-0].fromTerm
		}
	case 57:
		//line n1ql.y:622
		{
			yyVAL.fromTerm = yyS[yypt-0].keyspaceTerm
		}
	case 58:
		//line n1ql.y:627
		{
			yyVAL.fromTerm = algebra.NewJoin(yyS[yypt-3].fromTerm, yyS[yypt-2].b, yyS[yypt-0].keyspaceTerm)
		}
	case 59:
		//line n1ql.y:632
		{
			yyVAL.fromTerm = algebra.NewNest(yyS[yypt-3].fromTerm, yyS[yypt-2].b, yyS[yypt-0].keyspaceTerm)
		}
	case 60:
		//line n1ql.y:637
		{
			yyVAL.fromTerm = algebra.NewUnnest(yyS[yypt-4].fromTerm, yyS[yypt-3].b, yyS[yypt-1].expr, yyS[yypt-0].s)
		}
	case 63:
		//line n1ql.y:650
		{
			yyVAL.keyspaceTerm = algebra.NewKeyspaceTerm("", yyS[yypt-3].s, yyS[yypt-2].path, yyS[yypt-1].s, yyS[yypt-0].expr)
		}
	case 64:
		//line n1ql.y:655
		{
			yyVAL.keyspaceTerm = algebra.NewKeyspaceTerm(yyS[yypt-5].s, yyS[yypt-3].s, yyS[yypt-2].path, yyS[yypt-1].s, yyS[yypt-0].expr)
		}
	case 65:
		//line n1ql.y:660
		{
			yyVAL.keyspaceTerm = algebra.NewKeyspaceTerm("#system", yyS[yypt-3].s, yyS[yypt-2].path, yyS[yypt-1].s, yyS[yypt-0].expr)
		}
	case 66:
		//line n1ql.y:667
		{
			yyVAL.keyspaceTerm = algebra.NewKeyspaceTerm("", yyS[yypt-3].s, yyS[yypt-2].path, yyS[yypt-1].s, yyS[yypt-0].expr)
		}
	case 67:
		//line n1ql.y:672
		{
			yyVAL.keyspaceTerm = algebra.NewKeyspaceTerm(yyS[yypt-5].s, yyS[yypt-3].s, yyS[yypt-2].path, yyS[yypt-1].s, yyS[yypt-0].expr)
		}
	case 68:
		//line n1ql.y:677
		{
			yyVAL.keyspaceTerm = algebra.NewKeyspaceTerm("#system", yyS[yypt-3].s, yyS[yypt-2].path, yyS[yypt-1].s, yyS[yypt-0].expr)
		}
	case 69:
		yyVAL.s = yyS[yypt-0].s
	case 70:
		yyVAL.s = yyS[yypt-0].s
	case 71:
		//line n1ql.y:692
		{
			yyVAL.path = nil
		}
	case 72:
		//line n1ql.y:697
		{
			yyVAL.path = yyS[yypt-0].path
		}
	case 73:
		//line n1ql.y:704
		{
			yyVAL.expr = nil
		}
	case 74:
		yyVAL.expr = yyS[yypt-0].expr
	case 75:
		//line n1ql.y:713
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 76:
		//line n1ql.y:720
		{
		}
	case 78:
		//line n1ql.y:728
		{
			yyVAL.b = false
		}
	case 79:
		//line n1ql.y:733
		{
			yyVAL.b = false
		}
	case 80:
		//line n1ql.y:738
		{
			yyVAL.b = true
		}
	case 83:
		//line n1ql.y:751
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 84:
		//line n1ql.y:765
		{
			yyVAL.bindings = nil
		}
	case 85:
		yyVAL.bindings = yyS[yypt-0].bindings
	case 86:
		//line n1ql.y:774
		{
			yyVAL.bindings = yyS[yypt-0].bindings
		}
	case 87:
		//line n1ql.y:781
		{
			yyVAL.bindings = expression.Bindings{yyS[yypt-0].binding}
		}
	case 88:
		//line n1ql.y:786
		{
			yyVAL.bindings = append(yyS[yypt-2].bindings, yyS[yypt-0].binding)
		}
	case 89:
		//line n1ql.y:793
		{
			yyVAL.binding = expression.NewBinding(yyS[yypt-2].s, yyS[yypt-0].expr)
		}
	case 90:
		//line n1ql.y:807
		{
			yyVAL.expr = nil
		}
	case 91:
		yyVAL.expr = yyS[yypt-0].expr
	case 92:
		//line n1ql.y:816
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 93:
		//line n1ql.y:830
		{
			yyVAL.group = nil
		}
	case 94:
		yyVAL.group = yyS[yypt-0].group
	case 95:
		//line n1ql.y:839
		{
			yyVAL.group = algebra.NewGroup(yyS[yypt-2].exprs, yyS[yypt-1].bindings, yyS[yypt-0].expr)
		}
	case 96:
		//line n1ql.y:844
		{
			yyVAL.group = algebra.NewGroup(nil, yyS[yypt-0].bindings, nil)
		}
	case 97:
		//line n1ql.y:851
		{
			yyVAL.exprs = expression.Expressions{yyS[yypt-0].expr}
		}
	case 98:
		//line n1ql.y:856
		{
			yyVAL.exprs = append(yyS[yypt-2].exprs, yyS[yypt-0].expr)
		}
	case 99:
		//line n1ql.y:863
		{
			yyVAL.bindings = nil
		}
	case 100:
		yyVAL.bindings = yyS[yypt-0].bindings
	case 101:
		//line n1ql.y:872
		{
			yyVAL.bindings = yyS[yypt-0].bindings
		}
	case 102:
		//line n1ql.y:879
		{
			yyVAL.expr = nil
		}
	case 103:
		yyVAL.expr = yyS[yypt-0].expr
	case 104:
		//line n1ql.y:888
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 105:
		//line n1ql.y:902
		{
			yyVAL.order = nil
		}
	case 106:
		yyVAL.order = yyS[yypt-0].order
	case 107:
		//line n1ql.y:911
		{
			yyVAL.order = algebra.NewOrder(yyS[yypt-0].sortTerms)
		}
	case 108:
		//line n1ql.y:918
		{
			yyVAL.sortTerms = algebra.SortTerms{yyS[yypt-0].sortTerm}
		}
	case 109:
		//line n1ql.y:923
		{
			yyVAL.sortTerms = append(yyS[yypt-2].sortTerms, yyS[yypt-0].sortTerm)
		}
	case 110:
		//line n1ql.y:930
		{
			yyVAL.sortTerm = algebra.NewSortTerm(yyS[yypt-1].expr, yyS[yypt-0].b)
		}
	case 111:
		//line n1ql.y:937
		{
			yyVAL.b = false
		}
	case 112:
		yyVAL.b = yyS[yypt-0].b
	case 113:
		//line n1ql.y:946
		{
			yyVAL.b = false
		}
	case 114:
		//line n1ql.y:951
		{
			yyVAL.b = true
		}
	case 115:
		//line n1ql.y:965
		{
			yyVAL.expr = nil
		}
	case 116:
		yyVAL.expr = yyS[yypt-0].expr
	case 117:
		//line n1ql.y:974
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 118:
		//line n1ql.y:988
		{
			yyVAL.expr = nil
		}
	case 119:
		yyVAL.expr = yyS[yypt-0].expr
	case 120:
		//line n1ql.y:997
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 121:
		//line n1ql.y:1011
		{
			yyVAL.statement = algebra.NewInsertValues(yyS[yypt-3].keyspaceRef, yyS[yypt-2].expr, yyS[yypt-1].expr, yyS[yypt-0].projection)
		}
	case 122:
		//line n1ql.y:1016
		{
			yyVAL.statement = algebra.NewInsertSelect(yyS[yypt-3].keyspaceRef, yyS[yypt-2].expr, yyS[yypt-1].fullselect, yyS[yypt-0].projection)
		}
	case 123:
		//line n1ql.y:1023
		{
			yyVAL.keyspaceRef = algebra.NewKeyspaceRef(yyS[yypt-3].s, yyS[yypt-1].s, yyS[yypt-0].s)
		}
	case 124:
		//line n1ql.y:1028
		{
			yyVAL.keyspaceRef = algebra.NewKeyspaceRef("", yyS[yypt-1].s, yyS[yypt-0].s)
		}
	case 125:
		//line n1ql.y:1035
		{
			yyVAL.expr = nil
		}
	case 126:
		yyVAL.expr = yyS[yypt-0].expr
	case 127:
		//line n1ql.y:1044
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 128:
		//line n1ql.y:1051
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 129:
		//line n1ql.y:1058
		{
			yyVAL.projection = nil
		}
	case 130:
		yyVAL.projection = yyS[yypt-0].projection
	case 131:
		//line n1ql.y:1067
		{
			yyVAL.projection = yyS[yypt-0].projection
		}
	case 132:
		//line n1ql.y:1074
		{
			yyVAL.projection = algebra.NewProjection(false, yyS[yypt-0].resultTerms)
		}
	case 133:
		//line n1ql.y:1079
		{
			yyVAL.projection = algebra.NewRawProjection(false, yyS[yypt-0].expr, "")
		}
	case 134:
		//line n1ql.y:1093
		{
			yyVAL.statement = algebra.NewUpsertValues(yyS[yypt-3].keyspaceRef, yyS[yypt-2].expr, yyS[yypt-1].expr, yyS[yypt-0].projection)
		}
	case 135:
		//line n1ql.y:1098
		{
			yyVAL.statement = algebra.NewUpsertSelect(yyS[yypt-3].keyspaceRef, yyS[yypt-2].expr, yyS[yypt-1].fullselect, yyS[yypt-0].projection)
		}
	case 136:
		//line n1ql.y:1112
		{
			yyVAL.statement = algebra.NewDelete(yyS[yypt-4].keyspaceRef, yyS[yypt-3].expr, yyS[yypt-2].expr, yyS[yypt-1].expr, yyS[yypt-0].projection)
		}
	case 137:
		//line n1ql.y:1126
		{
			yyVAL.statement = algebra.NewUpdate(yyS[yypt-6].keyspaceRef, yyS[yypt-5].expr, yyS[yypt-4].set, yyS[yypt-3].unset, yyS[yypt-2].expr, yyS[yypt-1].expr, yyS[yypt-0].projection)
		}
	case 138:
		//line n1ql.y:1131
		{
			yyVAL.statement = algebra.NewUpdate(yyS[yypt-5].keyspaceRef, yyS[yypt-4].expr, yyS[yypt-3].set, nil, yyS[yypt-2].expr, yyS[yypt-1].expr, yyS[yypt-0].projection)
		}
	case 139:
		//line n1ql.y:1136
		{
			yyVAL.statement = algebra.NewUpdate(yyS[yypt-5].keyspaceRef, yyS[yypt-4].expr, nil, yyS[yypt-3].unset, yyS[yypt-2].expr, yyS[yypt-1].expr, yyS[yypt-0].projection)
		}
	case 140:
		//line n1ql.y:1143
		{
			yyVAL.set = algebra.NewSet(yyS[yypt-0].setTerms)
		}
	case 141:
		//line n1ql.y:1150
		{
			yyVAL.setTerms = algebra.SetTerms{yyS[yypt-0].setTerm}
		}
	case 142:
		//line n1ql.y:1155
		{
			yyVAL.setTerms = append(yyS[yypt-2].setTerms, yyS[yypt-0].setTerm)
		}
	case 143:
		//line n1ql.y:1162
		{
			yyVAL.setTerm = algebra.NewSetTerm(yyS[yypt-3].path, yyS[yypt-1].expr, yyS[yypt-0].updateFor)
		}
	case 144:
		//line n1ql.y:1169
		{
			yyVAL.updateFor = nil
		}
	case 145:
		yyVAL.updateFor = yyS[yypt-0].updateFor
	case 146:
		//line n1ql.y:1178
		{
			yyVAL.updateFor = algebra.NewUpdateFor(yyS[yypt-2].bindings, yyS[yypt-1].expr)
		}
	case 147:
		//line n1ql.y:1185
		{
			yyVAL.bindings = expression.Bindings{yyS[yypt-0].binding}
		}
	case 148:
		//line n1ql.y:1190
		{
			yyVAL.bindings = append(yyS[yypt-2].bindings, yyS[yypt-0].binding)
		}
	case 149:
		//line n1ql.y:1197
		{
			yyVAL.binding = expression.NewBinding(yyS[yypt-2].s, yyS[yypt-0].expr)
		}
	case 150:
		//line n1ql.y:1202
		{
			yyVAL.binding = expression.NewDescendantBinding(yyS[yypt-2].s, yyS[yypt-0].expr)
		}
	case 151:
		yyVAL.s = yyS[yypt-0].s
	case 152:
		//line n1ql.y:1213
		{
			yyVAL.expr = yyS[yypt-0].path
		}
	case 153:
		//line n1ql.y:1220
		{
			yyVAL.expr = nil
		}
	case 154:
		//line n1ql.y:1225
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 155:
		//line n1ql.y:1232
		{
			yyVAL.unset = algebra.NewUnset(yyS[yypt-0].unsetTerms)
		}
	case 156:
		//line n1ql.y:1239
		{
			yyVAL.unsetTerms = algebra.UnsetTerms{yyS[yypt-0].unsetTerm}
		}
	case 157:
		//line n1ql.y:1244
		{
			yyVAL.unsetTerms = append(yyS[yypt-2].unsetTerms, yyS[yypt-0].unsetTerm)
		}
	case 158:
		//line n1ql.y:1251
		{
			yyVAL.unsetTerm = algebra.NewUnsetTerm(yyS[yypt-1].path, yyS[yypt-0].updateFor)
		}
	case 159:
		//line n1ql.y:1265
		{
			source := algebra.NewMergeSourceFrom(yyS[yypt-5].keyspaceTerm, "")
			yyVAL.statement = algebra.NewMerge(yyS[yypt-7].keyspaceRef, source, yyS[yypt-3].expr, yyS[yypt-2].mergeActions, yyS[yypt-1].expr, yyS[yypt-0].projection)
		}
	case 160:
		//line n1ql.y:1271
		{
			source := algebra.NewMergeSourceSelect(yyS[yypt-7].fullselect, yyS[yypt-5].s)
			yyVAL.statement = algebra.NewMerge(yyS[yypt-10].keyspaceRef, source, yyS[yypt-3].expr, yyS[yypt-2].mergeActions, yyS[yypt-1].expr, yyS[yypt-0].projection)
		}
	case 161:
		//line n1ql.y:1279
		{
			yyVAL.mergeActions = algebra.NewMergeActions(nil, nil, nil)
		}
	case 162:
		//line n1ql.y:1284
		{
			yyVAL.mergeActions = algebra.NewMergeActions(yyS[yypt-1].mergeUpdate, yyS[yypt-0].mergeActions.Delete(), yyS[yypt-0].mergeActions.Insert())
		}
	case 163:
		//line n1ql.y:1289
		{
			yyVAL.mergeActions = algebra.NewMergeActions(nil, yyS[yypt-1].mergeDelete, yyS[yypt-0].mergeInsert)
		}
	case 164:
		//line n1ql.y:1294
		{
			yyVAL.mergeActions = algebra.NewMergeActions(nil, nil, yyS[yypt-0].mergeInsert)
		}
	case 165:
		//line n1ql.y:1301
		{
			yyVAL.mergeActions = algebra.NewMergeActions(nil, nil, nil)
		}
	case 166:
		//line n1ql.y:1306
		{
			yyVAL.mergeActions = algebra.NewMergeActions(nil, yyS[yypt-1].mergeDelete, yyS[yypt-0].mergeInsert)
		}
	case 167:
		//line n1ql.y:1311
		{
			yyVAL.mergeActions = algebra.NewMergeActions(nil, nil, yyS[yypt-0].mergeInsert)
		}
	case 168:
		//line n1ql.y:1318
		{
			yyVAL.mergeInsert = nil
		}
	case 169:
		//line n1ql.y:1323
		{
			yyVAL.mergeInsert = yyS[yypt-0].mergeInsert
		}
	case 170:
		//line n1ql.y:1330
		{
			yyVAL.mergeUpdate = algebra.NewMergeUpdate(yyS[yypt-1].set, nil, yyS[yypt-0].expr)
		}
	case 171:
		//line n1ql.y:1335
		{
			yyVAL.mergeUpdate = algebra.NewMergeUpdate(yyS[yypt-2].set, yyS[yypt-1].unset, yyS[yypt-0].expr)
		}
	case 172:
		//line n1ql.y:1340
		{
			yyVAL.mergeUpdate = algebra.NewMergeUpdate(nil, yyS[yypt-1].unset, yyS[yypt-0].expr)
		}
	case 173:
		//line n1ql.y:1347
		{
			yyVAL.mergeDelete = algebra.NewMergeDelete(yyS[yypt-0].expr)
		}
	case 174:
		//line n1ql.y:1354
		{
			yyVAL.mergeInsert = algebra.NewMergeInsert(yyS[yypt-1].expr, yyS[yypt-0].expr)
		}
	case 175:
		//line n1ql.y:1368
		{
			yyVAL.statement = algebra.NewCreatePrimaryIndex(yyS[yypt-1].keyspaceRef, yyS[yypt-0].indexType)
		}
	case 176:
		//line n1ql.y:1373
		{
			yyVAL.statement = algebra.NewCreateIndex(yyS[yypt-8].s, yyS[yypt-6].keyspaceRef, yyS[yypt-4].exprs, yyS[yypt-2].expr, yyS[yypt-1].expr, yyS[yypt-0].indexType)
		}
	case 177:
		yyVAL.s = yyS[yypt-0].s
	case 178:
		//line n1ql.y:1384
		{
			yyVAL.keyspaceRef = algebra.NewKeyspaceRef("", yyS[yypt-0].s, "")
		}
	case 179:
		//line n1ql.y:1389
		{
			yyVAL.keyspaceRef = algebra.NewKeyspaceRef(yyS[yypt-2].s, yyS[yypt-0].s, "")
		}
	case 180:
		//line n1ql.y:1396
		{
			yyVAL.expr = nil
		}
	case 181:
		//line n1ql.y:1401
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 182:
		//line n1ql.y:1408
		{
			yyVAL.indexType = datastore.VIEW
		}
	case 183:
		//line n1ql.y:1413
		{
			yyVAL.indexType = datastore.VIEW
		}
	case 184:
		//line n1ql.y:1418
		{
			yyVAL.indexType = datastore.LSM
		}
	case 185:
		//line n1ql.y:1425
		{
			yyVAL.exprs = expression.Expressions{yyS[yypt-0].expr}
		}
	case 186:
		//line n1ql.y:1430
		{
			yyVAL.exprs = append(yyS[yypt-2].exprs, yyS[yypt-0].expr)
		}
	case 187:
		//line n1ql.y:1437
		{
			exp := yyS[yypt-0].expr
			if !exp.Indexable() || exp.Value() != nil {
				yylex.Error(fmt.Sprintf("Expression not indexable."))
			}

			yyVAL.expr = exp
		}
	case 188:
		//line n1ql.y:1448
		{
			yyVAL.expr = nil
		}
	case 189:
		//line n1ql.y:1453
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 190:
		//line n1ql.y:1467
		{
			yyVAL.statement = algebra.NewDropIndex(yyS[yypt-0].keyspaceRef, "#primary")
		}
	case 191:
		//line n1ql.y:1472
		{
			yyVAL.statement = algebra.NewDropIndex(yyS[yypt-2].keyspaceRef, yyS[yypt-0].s)
		}
	case 192:
		//line n1ql.y:1485
		{
			yyVAL.statement = algebra.NewAlterIndex(yyS[yypt-3].keyspaceRef, yyS[yypt-1].s, yyS[yypt-0].s)
		}
	case 193:
		//line n1ql.y:1491
		{
			yyVAL.s = ""
		}
	case 194:
		//line n1ql.y:1496
		{
			yyVAL.s = yyS[yypt-0].s
		}
	case 195:
		//line n1ql.y:1510
		{
			yyVAL.path = expression.NewIdentifier(yyS[yypt-0].s)
		}
	case 196:
		//line n1ql.y:1515
		{
			yyVAL.path = expression.NewField(yyS[yypt-2].path, expression.NewFieldName(yyS[yypt-0].s))
		}
	case 197:
		//line n1ql.y:1520
		{
			field := expression.NewField(yyS[yypt-2].path, expression.NewFieldName(yyS[yypt-0].s))
			field.SetCaseInsensitive(true)
			yyVAL.path = field
		}
	case 198:
		//line n1ql.y:1527
		{
			yyVAL.path = expression.NewElement(yyS[yypt-3].path, yyS[yypt-1].expr)
		}
	case 199:
		yyVAL.expr = yyS[yypt-0].expr
	case 200:
		//line n1ql.y:1544
		{
			yyVAL.expr = expression.NewField(yyS[yypt-2].expr, expression.NewFieldName(yyS[yypt-0].s))
		}
	case 201:
		//line n1ql.y:1549
		{
			field := expression.NewField(yyS[yypt-2].expr, expression.NewFieldName(yyS[yypt-0].s))
			field.SetCaseInsensitive(true)
			yyVAL.expr = field
		}
	case 202:
		//line n1ql.y:1556
		{
			yyVAL.expr = expression.NewField(yyS[yypt-4].expr, yyS[yypt-1].expr)
		}
	case 203:
		//line n1ql.y:1561
		{
			field := expression.NewField(yyS[yypt-4].expr, yyS[yypt-1].expr)
			field.SetCaseInsensitive(true)
			yyVAL.expr = field
		}
	case 204:
		//line n1ql.y:1568
		{
			yyVAL.expr = expression.NewElement(yyS[yypt-3].expr, yyS[yypt-1].expr)
		}
	case 205:
		//line n1ql.y:1573
		{
			yyVAL.expr = expression.NewSlice(yyS[yypt-4].expr, yyS[yypt-2].expr)
		}
	case 206:
		//line n1ql.y:1578
		{
			yyVAL.expr = expression.NewSlice(yyS[yypt-5].expr, yyS[yypt-3].expr, yyS[yypt-1].expr)
		}
	case 207:
		//line n1ql.y:1584
		{
			yyVAL.expr = expression.NewAdd(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 208:
		//line n1ql.y:1589
		{
			yyVAL.expr = expression.NewSub(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 209:
		//line n1ql.y:1594
		{
			yyVAL.expr = expression.NewMult(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 210:
		//line n1ql.y:1599
		{
			yyVAL.expr = expression.NewDiv(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 211:
		//line n1ql.y:1604
		{
			yyVAL.expr = expression.NewMod(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 212:
		//line n1ql.y:1610
		{
			yyVAL.expr = expression.NewConcat(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 213:
		//line n1ql.y:1616
		{
			yyVAL.expr = expression.NewAnd(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 214:
		//line n1ql.y:1621
		{
			yyVAL.expr = expression.NewOr(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 215:
		//line n1ql.y:1626
		{
			yyVAL.expr = expression.NewNot(yyS[yypt-0].expr)
		}
	case 216:
		//line n1ql.y:1632
		{
			yyVAL.expr = expression.NewEq(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 217:
		//line n1ql.y:1637
		{
			yyVAL.expr = expression.NewEq(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 218:
		//line n1ql.y:1642
		{
			yyVAL.expr = expression.NewNE(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 219:
		//line n1ql.y:1647
		{
			yyVAL.expr = expression.NewLT(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 220:
		//line n1ql.y:1652
		{
			yyVAL.expr = expression.NewGT(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 221:
		//line n1ql.y:1657
		{
			yyVAL.expr = expression.NewLE(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 222:
		//line n1ql.y:1662
		{
			yyVAL.expr = expression.NewGE(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 223:
		//line n1ql.y:1667
		{
			yyVAL.expr = expression.NewBetween(yyS[yypt-4].expr, yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 224:
		//line n1ql.y:1672
		{
			yyVAL.expr = expression.NewNotBetween(yyS[yypt-5].expr, yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 225:
		//line n1ql.y:1677
		{
			yyVAL.expr = expression.NewLike(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 226:
		//line n1ql.y:1682
		{
			yyVAL.expr = expression.NewNotLike(yyS[yypt-3].expr, yyS[yypt-0].expr)
		}
	case 227:
		//line n1ql.y:1687
		{
			yyVAL.expr = expression.NewIn(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 228:
		//line n1ql.y:1692
		{
			yyVAL.expr = expression.NewNotIn(yyS[yypt-3].expr, yyS[yypt-0].expr)
		}
	case 229:
		//line n1ql.y:1697
		{
			yyVAL.expr = expression.NewWithin(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 230:
		//line n1ql.y:1702
		{
			yyVAL.expr = expression.NewNotWithin(yyS[yypt-3].expr, yyS[yypt-0].expr)
		}
	case 231:
		//line n1ql.y:1707
		{
			yyVAL.expr = expression.NewIsNull(yyS[yypt-2].expr)
		}
	case 232:
		//line n1ql.y:1712
		{
			yyVAL.expr = expression.NewIsNotNull(yyS[yypt-3].expr)
		}
	case 233:
		//line n1ql.y:1717
		{
			yyVAL.expr = expression.NewIsMissing(yyS[yypt-2].expr)
		}
	case 234:
		//line n1ql.y:1722
		{
			yyVAL.expr = expression.NewIsNotMissing(yyS[yypt-3].expr)
		}
	case 235:
		//line n1ql.y:1727
		{
			yyVAL.expr = expression.NewIsValued(yyS[yypt-2].expr)
		}
	case 236:
		//line n1ql.y:1732
		{
			yyVAL.expr = expression.NewIsNotValued(yyS[yypt-3].expr)
		}
	case 237:
		//line n1ql.y:1737
		{
			yyVAL.expr = expression.NewIsBoolean(yyS[yypt-2].expr)
		}
	case 238:
		//line n1ql.y:1742
		{
			yyVAL.expr = expression.NewNot(expression.NewIsBoolean(yyS[yypt-3].expr))
		}
	case 239:
		//line n1ql.y:1747
		{
			yyVAL.expr = expression.NewIsNumber(yyS[yypt-2].expr)
		}
	case 240:
		//line n1ql.y:1752
		{
			yyVAL.expr = expression.NewNot(expression.NewIsNumber(yyS[yypt-3].expr))
		}
	case 241:
		//line n1ql.y:1757
		{
			yyVAL.expr = expression.NewIsString(yyS[yypt-2].expr)
		}
	case 242:
		//line n1ql.y:1762
		{
			yyVAL.expr = expression.NewNot(expression.NewIsString(yyS[yypt-3].expr))
		}
	case 243:
		//line n1ql.y:1767
		{
			yyVAL.expr = expression.NewIsArray(yyS[yypt-2].expr)
		}
	case 244:
		//line n1ql.y:1772
		{
			yyVAL.expr = expression.NewNot(expression.NewIsArray(yyS[yypt-3].expr))
		}
	case 245:
		//line n1ql.y:1777
		{
			yyVAL.expr = expression.NewIsObject(yyS[yypt-2].expr)
		}
	case 246:
		//line n1ql.y:1782
		{
			yyVAL.expr = expression.NewNot(expression.NewIsObject(yyS[yypt-3].expr))
		}
	case 247:
		//line n1ql.y:1787
		{
			yyVAL.expr = expression.NewIsBinary(yyS[yypt-2].expr)
		}
	case 248:
		//line n1ql.y:1792
		{
			yyVAL.expr = expression.NewNot(expression.NewIsBinary(yyS[yypt-3].expr))
		}
	case 249:
		//line n1ql.y:1797
		{
			yyVAL.expr = expression.NewExists(yyS[yypt-0].expr)
		}
	case 250:
		yyVAL.expr = yyS[yypt-0].expr
	case 251:
		yyVAL.expr = yyS[yypt-0].expr
	case 252:
		//line n1ql.y:1811
		{
			yyVAL.expr = expression.NewIdentifier(yyS[yypt-0].s)
		}
	case 253:
		//line n1ql.y:1817
		{
			yyVAL.expr = expression.NewSelf()
		}
	case 254:
		yyVAL.expr = yyS[yypt-0].expr
	case 255:
		yyVAL.expr = yyS[yypt-0].expr
	case 256:
		//line n1ql.y:1829
		{
			yyVAL.expr = expression.NewNeg(yyS[yypt-0].expr)
		}
	case 257:
		yyVAL.expr = yyS[yypt-0].expr
	case 258:
		yyVAL.expr = yyS[yypt-0].expr
	case 259:
		yyVAL.expr = yyS[yypt-0].expr
	case 260:
		yyVAL.expr = yyS[yypt-0].expr
	case 261:
		//line n1ql.y:1848
		{
			yyVAL.expr = expression.NewField(yyS[yypt-2].expr, expression.NewFieldName(yyS[yypt-0].s))
		}
	case 262:
		//line n1ql.y:1853
		{
			field := expression.NewField(yyS[yypt-2].expr, expression.NewFieldName(yyS[yypt-0].s))
			field.SetCaseInsensitive(true)
			yyVAL.expr = field
		}
	case 263:
		//line n1ql.y:1860
		{
			yyVAL.expr = expression.NewField(yyS[yypt-4].expr, yyS[yypt-1].expr)
		}
	case 264:
		//line n1ql.y:1865
		{
			field := expression.NewField(yyS[yypt-4].expr, yyS[yypt-1].expr)
			field.SetCaseInsensitive(true)
			yyVAL.expr = field
		}
	case 265:
		//line n1ql.y:1872
		{
			yyVAL.expr = expression.NewElement(yyS[yypt-3].expr, yyS[yypt-1].expr)
		}
	case 266:
		//line n1ql.y:1877
		{
			yyVAL.expr = expression.NewSlice(yyS[yypt-4].expr, yyS[yypt-2].expr)
		}
	case 267:
		//line n1ql.y:1882
		{
			yyVAL.expr = expression.NewSlice(yyS[yypt-5].expr, yyS[yypt-3].expr, yyS[yypt-1].expr)
		}
	case 268:
		//line n1ql.y:1888
		{
			yyVAL.expr = expression.NewAdd(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 269:
		//line n1ql.y:1893
		{
			yyVAL.expr = expression.NewSub(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 270:
		//line n1ql.y:1898
		{
			yyVAL.expr = expression.NewMult(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 271:
		//line n1ql.y:1903
		{
			yyVAL.expr = expression.NewDiv(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 272:
		//line n1ql.y:1908
		{
			yyVAL.expr = expression.NewMod(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 273:
		//line n1ql.y:1914
		{
			yyVAL.expr = expression.NewConcat(yyS[yypt-2].expr, yyS[yypt-0].expr)
		}
	case 274:
		//line n1ql.y:1928
		{
			yyVAL.expr = expression.NULL_EXPR
		}
	case 275:
		//line n1ql.y:1933
		{
			yyVAL.expr = expression.MISSING_EXPR
		}
	case 276:
		//line n1ql.y:1938
		{
			yyVAL.expr = expression.FALSE_EXPR
		}
	case 277:
		//line n1ql.y:1943
		{
			yyVAL.expr = expression.TRUE_EXPR
		}
	case 278:
		//line n1ql.y:1948
		{
			yyVAL.expr = expression.NewConstant(value.NewValue(yyS[yypt-0].f))
		}
	case 279:
		//line n1ql.y:1953
		{
			yyVAL.expr = expression.NewConstant(value.NewValue(yyS[yypt-0].n))
		}
	case 280:
		//line n1ql.y:1958
		{
			yyVAL.expr = expression.NewConstant(value.NewValue(yyS[yypt-0].s))
		}
	case 281:
		yyVAL.expr = yyS[yypt-0].expr
	case 282:
		yyVAL.expr = yyS[yypt-0].expr
	case 283:
		//line n1ql.y:1978
		{
			yyVAL.expr = expression.NewObjectConstruct(yyS[yypt-1].bindings)
		}
	case 284:
		//line n1ql.y:1985
		{
			yyVAL.bindings = nil
		}
	case 285:
		yyVAL.bindings = yyS[yypt-0].bindings
	case 286:
		//line n1ql.y:1994
		{
			yyVAL.bindings = expression.Bindings{yyS[yypt-0].binding}
		}
	case 287:
		//line n1ql.y:1999
		{
			yyVAL.bindings = append(yyS[yypt-2].bindings, yyS[yypt-0].binding)
		}
	case 288:
		//line n1ql.y:2006
		{
			yyVAL.binding = expression.NewBinding(yyS[yypt-2].s, yyS[yypt-0].expr)
		}
	case 289:
		//line n1ql.y:2013
		{
			yyVAL.expr = expression.NewArrayConstruct(yyS[yypt-1].exprs...)
		}
	case 290:
		//line n1ql.y:2020
		{
			yyVAL.exprs = nil
		}
	case 291:
		yyVAL.exprs = yyS[yypt-0].exprs
	case 292:
		//line n1ql.y:2036
		{
			yyVAL.expr = algebra.NewNamedParameter(yyS[yypt-0].s)
		}
	case 293:
		//line n1ql.y:2041
		{
			yyVAL.expr = algebra.NewPositionalParameter(yyS[yypt-0].n)
		}
	case 294:
		//line n1ql.y:2055
		{
			yyVAL.expr = yyS[yypt-1].expr
		}
	case 295:
		yyVAL.expr = yyS[yypt-0].expr
	case 296:
		yyVAL.expr = yyS[yypt-0].expr
	case 297:
		//line n1ql.y:2068
		{
			yyVAL.expr = expression.NewSimpleCase(yyS[yypt-2].expr, yyS[yypt-1].whenTerms, yyS[yypt-0].expr)
		}
	case 298:
		//line n1ql.y:2075
		{
			yyVAL.whenTerms = expression.WhenTerms{&expression.WhenTerm{yyS[yypt-2].expr, yyS[yypt-0].expr}}
		}
	case 299:
		//line n1ql.y:2080
		{
			yyVAL.whenTerms = append(yyS[yypt-4].whenTerms, &expression.WhenTerm{yyS[yypt-2].expr, yyS[yypt-0].expr})
		}
	case 300:
		//line n1ql.y:2088
		{
			yyVAL.expr = expression.NewSearchedCase(yyS[yypt-1].whenTerms, yyS[yypt-0].expr)
		}
	case 301:
		//line n1ql.y:2095
		{
			yyVAL.expr = nil
		}
	case 302:
		//line n1ql.y:2100
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 303:
		//line n1ql.y:2114
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
	case 304:
		//line n1ql.y:2133
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
	case 305:
		//line n1ql.y:2148
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
	case 306:
		yyVAL.s = yyS[yypt-0].s
	case 307:
		yyVAL.expr = yyS[yypt-0].expr
	case 308:
		yyVAL.expr = yyS[yypt-0].expr
	case 309:
		//line n1ql.y:2182
		{
			yyVAL.expr = expression.NewAny(yyS[yypt-2].bindings, yyS[yypt-1].expr)
		}
	case 310:
		//line n1ql.y:2187
		{
			yyVAL.expr = expression.NewAny(yyS[yypt-2].bindings, yyS[yypt-1].expr)
		}
	case 311:
		//line n1ql.y:2192
		{
			yyVAL.expr = expression.NewEvery(yyS[yypt-2].bindings, yyS[yypt-1].expr)
		}
	case 312:
		//line n1ql.y:2199
		{
			yyVAL.bindings = expression.Bindings{yyS[yypt-0].binding}
		}
	case 313:
		//line n1ql.y:2204
		{
			yyVAL.bindings = append(yyS[yypt-2].bindings, yyS[yypt-0].binding)
		}
	case 314:
		//line n1ql.y:2211
		{
			yyVAL.binding = expression.NewBinding(yyS[yypt-2].s, yyS[yypt-0].expr)
		}
	case 315:
		//line n1ql.y:2216
		{
			yyVAL.binding = expression.NewDescendantBinding(yyS[yypt-2].s, yyS[yypt-0].expr)
		}
	case 316:
		//line n1ql.y:2223
		{
			yyVAL.expr = yyS[yypt-0].expr
		}
	case 317:
		//line n1ql.y:2230
		{
			yyVAL.expr = expression.NewArray(yyS[yypt-4].expr, yyS[yypt-2].bindings, yyS[yypt-1].expr)
		}
	case 318:
		//line n1ql.y:2235
		{
			yyVAL.expr = expression.NewFirst(yyS[yypt-4].expr, yyS[yypt-2].bindings, yyS[yypt-1].expr)
		}
	case 319:
		//line n1ql.y:2249
		{
			yyVAL.expr = yyS[yypt-1].expr
		}
	case 320:
		yyVAL.expr = yyS[yypt-0].expr
	case 321:
		//line n1ql.y:2258
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
