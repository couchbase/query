package n1ql

import "strconv"
import "github.com/couchbaselabs/clog"
import ("bufio";"io";"strings")
type dfa struct {
  acc []bool
  f []func(rune) int
  id int
}
type family struct {
  a []dfa
  endcase int
}
var a0 [187]dfa
var a []family
func init() {
a = make([]family, 1)
{
var acc [21]bool
var fun [21]func(rune) int
fun[6] = func(r rune) int {
  switch(r) {
  case 34: return 4
  case 47: return 3
  case 98: return 3
  case 114: return 3
  case 116: return 3
  case 52: return 3
  case 92: return 2
  case 102: return 3
  case 110: return 3
  case 117: return 3
  case 123: return 3
  case 125: return 3
  default:
    switch {
    case 48 <= r && r <= 57: return 3
    case 65 <= r && r <= 70: return 3
    case 97 <= r && r <= 102: return 3
    default: return 3
    }
  }
  panic("unreachable")
}
fun[11] = func(r rune) int {
  switch(r) {
  case 34: return 4
  case 47: return 3
  case 98: return 3
  case 114: return 3
  case 116: return 3
  case 52: return 3
  case 92: return 2
  case 102: return 3
  case 110: return 3
  case 117: return 3
  case 123: return 3
  case 125: return 3
  default:
    switch {
    case 48 <= r && r <= 57: return 3
    case 65 <= r && r <= 70: return 3
    case 97 <= r && r <= 102: return 3
    default: return 3
    }
  }
  panic("unreachable")
}
fun[13] = func(r rune) int {
  switch(r) {
  case 92: return 2
  case 102: return 3
  case 110: return 3
  case 117: return 3
  case 123: return 3
  case 125: return 3
  case 34: return 4
  case 47: return 3
  case 98: return 3
  case 114: return 3
  case 116: return 3
  case 52: return 3
  default:
    switch {
    case 48 <= r && r <= 57: return 3
    case 65 <= r && r <= 70: return 3
    case 97 <= r && r <= 102: return 3
    default: return 3
    }
  }
  panic("unreachable")
}
fun[15] = func(r rune) int {
  switch(r) {
  case 92: return -1
  case 102: return 16
  case 110: return -1
  case 117: return -1
  case 123: return -1
  case 125: return -1
  case 34: return -1
  case 47: return -1
  case 98: return 16
  case 114: return -1
  case 116: return -1
  case 52: return 16
  default:
    switch {
    case 48 <= r && r <= 57: return 16
    case 65 <= r && r <= 70: return 16
    case 97 <= r && r <= 102: return 16
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 92: return 5
  case 102: return 6
  case 110: return 7
  case 117: return 8
  case 123: return -1
  case 125: return -1
  case 34: return 9
  case 47: return 10
  case 98: return 11
  case 114: return 12
  case 116: return 13
  case 52: return -1
  default:
    switch {
    case 48 <= r && r <= 57: return -1
    case 65 <= r && r <= 70: return -1
    case 97 <= r && r <= 102: return -1
    default: return -1
    }
  }
  panic("unreachable")
}
fun[8] = func(r rune) int {
  switch(r) {
  case 34: return -1
  case 47: return -1
  case 98: return 14
  case 114: return -1
  case 116: return -1
  case 52: return 14
  case 92: return -1
  case 102: return 14
  case 110: return -1
  case 117: return -1
  case 123: return -1
  case 125: return -1
  default:
    switch {
    case 48 <= r && r <= 57: return 14
    case 65 <= r && r <= 70: return 14
    case 97 <= r && r <= 102: return 14
    default: return -1
    }
  }
  panic("unreachable")
}
fun[12] = func(r rune) int {
  switch(r) {
  case 92: return 2
  case 102: return 3
  case 110: return 3
  case 117: return 3
  case 123: return 3
  case 125: return 3
  case 34: return 4
  case 47: return 3
  case 98: return 3
  case 114: return 3
  case 116: return 3
  case 52: return 3
  default:
    switch {
    case 48 <= r && r <= 57: return 3
    case 65 <= r && r <= 70: return 3
    case 97 <= r && r <= 102: return 3
    default: return 3
    }
  }
  panic("unreachable")
}
fun[17] = func(r rune) int {
  switch(r) {
  case 92: return -1
  case 102: return -1
  case 110: return -1
  case 117: return -1
  case 123: return 18
  case 125: return -1
  case 34: return -1
  case 47: return -1
  case 98: return -1
  case 114: return -1
  case 116: return -1
  case 52: return -1
  default:
    switch {
    case 48 <= r && r <= 57: return -1
    case 65 <= r && r <= 70: return -1
    case 97 <= r && r <= 102: return -1
    default: return -1
    }
  }
  panic("unreachable")
}
fun[19] = func(r rune) int {
  switch(r) {
  case 34: return -1
  case 47: return -1
  case 98: return -1
  case 114: return -1
  case 116: return -1
  case 52: return -1
  case 92: return -1
  case 102: return -1
  case 110: return -1
  case 117: return -1
  case 123: return -1
  case 125: return 20
  default:
    switch {
    case 48 <= r && r <= 57: return -1
    case 65 <= r && r <= 70: return -1
    case 97 <= r && r <= 102: return -1
    default: return -1
    }
  }
  panic("unreachable")
}
fun[20] = func(r rune) int {
  switch(r) {
  case 34: return 4
  case 47: return 3
  case 98: return 3
  case 114: return 3
  case 116: return 3
  case 52: return 3
  case 92: return 2
  case 102: return 3
  case 110: return 3
  case 117: return 3
  case 123: return 3
  case 125: return 3
  default:
    switch {
    case 48 <= r && r <= 57: return 3
    case 65 <= r && r <= 70: return 3
    case 97 <= r && r <= 102: return 3
    default: return 3
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 34: return 4
  case 47: return 3
  case 98: return 3
  case 114: return 3
  case 116: return 3
  case 52: return 3
  case 92: return 2
  case 102: return 3
  case 110: return 3
  case 117: return 3
  case 123: return 3
  case 125: return 3
  default:
    switch {
    case 48 <= r && r <= 57: return 3
    case 65 <= r && r <= 70: return 3
    case 97 <= r && r <= 102: return 3
    default: return 3
    }
  }
  panic("unreachable")
}
fun[7] = func(r rune) int {
  switch(r) {
  case 92: return 2
  case 102: return 3
  case 110: return 3
  case 117: return 3
  case 123: return 3
  case 125: return 3
  case 34: return 4
  case 47: return 3
  case 98: return 3
  case 114: return 3
  case 116: return 3
  case 52: return 3
  default:
    switch {
    case 48 <= r && r <= 57: return 3
    case 65 <= r && r <= 70: return 3
    case 97 <= r && r <= 102: return 3
    default: return 3
    }
  }
  panic("unreachable")
}
fun[0] = func(r rune) int {
  switch(r) {
  case 34: return 1
  case 47: return -1
  case 98: return -1
  case 114: return -1
  case 116: return -1
  case 52: return -1
  case 92: return -1
  case 102: return -1
  case 110: return -1
  case 117: return -1
  case 123: return -1
  case 125: return -1
  default:
    switch {
    case 48 <= r && r <= 57: return -1
    case 65 <= r && r <= 70: return -1
    case 97 <= r && r <= 102: return -1
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 92: return 2
  case 102: return 3
  case 110: return 3
  case 117: return 3
  case 123: return 3
  case 125: return 3
  case 34: return 4
  case 47: return 3
  case 98: return 3
  case 114: return 3
  case 116: return 3
  case 52: return 3
  default:
    switch {
    case 48 <= r && r <= 57: return 3
    case 65 <= r && r <= 70: return 3
    case 97 <= r && r <= 102: return 3
    default: return 3
    }
  }
  panic("unreachable")
}
acc[4] = true
fun[4] = func(r rune) int {
  switch(r) {
  case 92: return -1
  case 102: return -1
  case 110: return -1
  case 117: return -1
  case 123: return -1
  case 125: return -1
  case 34: return -1
  case 47: return -1
  case 98: return -1
  case 114: return -1
  case 116: return -1
  case 52: return -1
  default:
    switch {
    case 48 <= r && r <= 57: return -1
    case 65 <= r && r <= 70: return -1
    case 97 <= r && r <= 102: return -1
    default: return -1
    }
  }
  panic("unreachable")
}
fun[5] = func(r rune) int {
  switch(r) {
  case 92: return 2
  case 102: return 3
  case 110: return 3
  case 117: return 3
  case 123: return 3
  case 125: return 3
  case 34: return 4
  case 47: return 3
  case 98: return 3
  case 114: return 3
  case 116: return 3
  case 52: return 3
  default:
    switch {
    case 48 <= r && r <= 57: return 3
    case 65 <= r && r <= 70: return 3
    case 97 <= r && r <= 102: return 3
    default: return 3
    }
  }
  panic("unreachable")
}
fun[9] = func(r rune) int {
  switch(r) {
  case 92: return 2
  case 102: return 3
  case 110: return 3
  case 117: return 3
  case 123: return 3
  case 125: return 3
  case 34: return 4
  case 47: return 3
  case 98: return 3
  case 114: return 3
  case 116: return 3
  case 52: return 3
  default:
    switch {
    case 48 <= r && r <= 57: return 3
    case 65 <= r && r <= 70: return 3
    case 97 <= r && r <= 102: return 3
    default: return 3
    }
  }
  panic("unreachable")
}
fun[10] = func(r rune) int {
  switch(r) {
  case 34: return 4
  case 47: return 3
  case 98: return 3
  case 114: return 3
  case 116: return 3
  case 52: return 3
  case 92: return 2
  case 102: return 3
  case 110: return 3
  case 117: return 3
  case 123: return 3
  case 125: return 3
  default:
    switch {
    case 48 <= r && r <= 57: return 3
    case 65 <= r && r <= 70: return 3
    case 97 <= r && r <= 102: return 3
    default: return 3
    }
  }
  panic("unreachable")
}
fun[14] = func(r rune) int {
  switch(r) {
  case 34: return -1
  case 47: return -1
  case 98: return 15
  case 114: return -1
  case 116: return -1
  case 52: return 15
  case 92: return -1
  case 102: return 15
  case 110: return -1
  case 117: return -1
  case 123: return -1
  case 125: return -1
  default:
    switch {
    case 48 <= r && r <= 57: return 15
    case 65 <= r && r <= 70: return 15
    case 97 <= r && r <= 102: return 15
    default: return -1
    }
  }
  panic("unreachable")
}
fun[16] = func(r rune) int {
  switch(r) {
  case 34: return -1
  case 47: return -1
  case 98: return 17
  case 114: return -1
  case 116: return -1
  case 52: return 17
  case 92: return -1
  case 102: return 17
  case 110: return -1
  case 117: return -1
  case 123: return -1
  case 125: return -1
  default:
    switch {
    case 48 <= r && r <= 57: return 17
    case 65 <= r && r <= 70: return 17
    case 97 <= r && r <= 102: return 17
    default: return -1
    }
  }
  panic("unreachable")
}
fun[18] = func(r rune) int {
  switch(r) {
  case 92: return -1
  case 102: return -1
  case 110: return -1
  case 117: return -1
  case 123: return -1
  case 125: return -1
  case 34: return -1
  case 47: return -1
  case 98: return -1
  case 114: return -1
  case 116: return -1
  case 52: return 19
  default:
    switch {
    case 48 <= r && r <= 57: return -1
    case 65 <= r && r <= 70: return -1
    case 97 <= r && r <= 102: return -1
    default: return -1
    }
  }
  panic("unreachable")
}
a0[0].acc = acc[:]
a0[0].f = fun[:]
a0[0].id = 0
}
{
var acc [22]bool
var fun [22]func(rune) int
fun[1] = func(r rune) int {
  switch(r) {
  case 47: return 2
  case 98: return 2
  case 125: return 2
  case 39: return 3
  case 92: return 4
  case 34: return -1
  case 102: return 2
  case 110: return 2
  case 114: return 2
  case 116: return 2
  case 117: return 2
  case 123: return 2
  case 52: return 2
  default:
    switch {
    case 48 <= r && r <= 57: return 2
    case 65 <= r && r <= 70: return 2
    case 97 <= r && r <= 102: return 2
    default: return 2
    }
  }
  panic("unreachable")
}
fun[7] = func(r rune) int {
  switch(r) {
  case 39: return 3
  case 92: return 4
  case 34: return -1
  case 102: return 2
  case 110: return 2
  case 114: return 2
  case 116: return 2
  case 117: return 2
  case 123: return 2
  case 52: return 2
  case 47: return 2
  case 98: return 2
  case 125: return 2
  default:
    switch {
    case 48 <= r && r <= 57: return 2
    case 65 <= r && r <= 70: return 2
    case 97 <= r && r <= 102: return 2
    default: return 2
    }
  }
  panic("unreachable")
}
fun[20] = func(r rune) int {
  switch(r) {
  case 39: return 3
  case 92: return 4
  case 34: return -1
  case 102: return 2
  case 110: return 2
  case 114: return 2
  case 116: return 2
  case 117: return 2
  case 123: return 2
  case 52: return 2
  case 47: return 2
  case 98: return 2
  case 125: return 2
  default:
    switch {
    case 48 <= r && r <= 57: return 2
    case 65 <= r && r <= 70: return 2
    case 97 <= r && r <= 102: return 2
    default: return 2
    }
  }
  panic("unreachable")
}
fun[0] = func(r rune) int {
  switch(r) {
  case 47: return -1
  case 98: return -1
  case 125: return -1
  case 39: return 1
  case 92: return -1
  case 34: return -1
  case 102: return -1
  case 110: return -1
  case 114: return -1
  case 116: return -1
  case 117: return -1
  case 123: return -1
  case 52: return -1
  default:
    switch {
    case 48 <= r && r <= 57: return -1
    case 65 <= r && r <= 70: return -1
    case 97 <= r && r <= 102: return -1
    default: return -1
    }
  }
  panic("unreachable")
}
fun[5] = func(r rune) int {
  switch(r) {
  case 39: return 3
  case 92: return 4
  case 34: return -1
  case 102: return 2
  case 110: return 2
  case 114: return 2
  case 116: return 2
  case 117: return 2
  case 123: return 2
  case 52: return 2
  case 47: return 2
  case 98: return 2
  case 125: return 2
  default:
    switch {
    case 48 <= r && r <= 57: return 2
    case 65 <= r && r <= 70: return 2
    case 97 <= r && r <= 102: return 2
    default: return 2
    }
  }
  panic("unreachable")
}
fun[6] = func(r rune) int {
  switch(r) {
  case 47: return 2
  case 98: return 2
  case 125: return 2
  case 39: return 3
  case 92: return 4
  case 34: return -1
  case 102: return 2
  case 110: return 2
  case 114: return 2
  case 116: return 2
  case 117: return 2
  case 123: return 2
  case 52: return 2
  default:
    switch {
    case 48 <= r && r <= 57: return 2
    case 65 <= r && r <= 70: return 2
    case 97 <= r && r <= 102: return 2
    default: return 2
    }
  }
  panic("unreachable")
}
fun[14] = func(r rune) int {
  switch(r) {
  case 47: return -1
  case 98: return 15
  case 125: return -1
  case 39: return -1
  case 92: return -1
  case 34: return -1
  case 102: return 15
  case 110: return -1
  case 114: return -1
  case 116: return -1
  case 117: return -1
  case 123: return -1
  case 52: return 15
  default:
    switch {
    case 48 <= r && r <= 57: return 15
    case 65 <= r && r <= 70: return 15
    case 97 <= r && r <= 102: return 15
    default: return -1
    }
  }
  panic("unreachable")
}
fun[16] = func(r rune) int {
  switch(r) {
  case 39: return -1
  case 92: return -1
  case 34: return -1
  case 102: return 17
  case 110: return -1
  case 114: return -1
  case 116: return -1
  case 117: return -1
  case 123: return -1
  case 52: return 17
  case 47: return -1
  case 98: return 17
  case 125: return -1
  default:
    switch {
    case 48 <= r && r <= 57: return 17
    case 65 <= r && r <= 70: return 17
    case 97 <= r && r <= 102: return 17
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 47: return 2
  case 98: return 2
  case 125: return 2
  case 39: return 3
  case 92: return 4
  case 34: return -1
  case 102: return 2
  case 110: return 2
  case 114: return 2
  case 116: return 2
  case 117: return 2
  case 123: return 2
  case 52: return 2
  default:
    switch {
    case 48 <= r && r <= 57: return 2
    case 65 <= r && r <= 70: return 2
    case 97 <= r && r <= 102: return 2
    default: return 2
    }
  }
  panic("unreachable")
}
acc[3] = true
fun[3] = func(r rune) int {
  switch(r) {
  case 47: return -1
  case 98: return -1
  case 125: return -1
  case 39: return 21
  case 92: return -1
  case 34: return -1
  case 102: return -1
  case 110: return -1
  case 114: return -1
  case 116: return -1
  case 117: return -1
  case 123: return -1
  case 52: return -1
  default:
    switch {
    case 48 <= r && r <= 57: return -1
    case 65 <= r && r <= 70: return -1
    case 97 <= r && r <= 102: return -1
    default: return -1
    }
  }
  panic("unreachable")
}
fun[4] = func(r rune) int {
  switch(r) {
  case 39: return -1
  case 92: return 5
  case 34: return 6
  case 102: return 7
  case 110: return 8
  case 114: return 9
  case 116: return 10
  case 117: return 11
  case 123: return -1
  case 52: return -1
  case 47: return 12
  case 98: return 13
  case 125: return -1
  default:
    switch {
    case 48 <= r && r <= 57: return -1
    case 65 <= r && r <= 70: return -1
    case 97 <= r && r <= 102: return -1
    default: return -1
    }
  }
  panic("unreachable")
}
fun[15] = func(r rune) int {
  switch(r) {
  case 47: return -1
  case 98: return 16
  case 125: return -1
  case 39: return -1
  case 92: return -1
  case 34: return -1
  case 102: return 16
  case 110: return -1
  case 114: return -1
  case 116: return -1
  case 117: return -1
  case 123: return -1
  case 52: return 16
  default:
    switch {
    case 48 <= r && r <= 57: return 16
    case 65 <= r && r <= 70: return 16
    case 97 <= r && r <= 102: return 16
    default: return -1
    }
  }
  panic("unreachable")
}
fun[17] = func(r rune) int {
  switch(r) {
  case 39: return -1
  case 92: return -1
  case 34: return -1
  case 102: return -1
  case 110: return -1
  case 114: return -1
  case 116: return -1
  case 117: return -1
  case 123: return 18
  case 52: return -1
  case 47: return -1
  case 98: return -1
  case 125: return -1
  default:
    switch {
    case 48 <= r && r <= 57: return -1
    case 65 <= r && r <= 70: return -1
    case 97 <= r && r <= 102: return -1
    default: return -1
    }
  }
  panic("unreachable")
}
fun[19] = func(r rune) int {
  switch(r) {
  case 39: return -1
  case 92: return -1
  case 34: return -1
  case 102: return -1
  case 110: return -1
  case 114: return -1
  case 116: return -1
  case 117: return -1
  case 123: return -1
  case 52: return -1
  case 47: return -1
  case 98: return -1
  case 125: return 20
  default:
    switch {
    case 48 <= r && r <= 57: return -1
    case 65 <= r && r <= 70: return -1
    case 97 <= r && r <= 102: return -1
    default: return -1
    }
  }
  panic("unreachable")
}
fun[21] = func(r rune) int {
  switch(r) {
  case 47: return 2
  case 98: return 2
  case 125: return 2
  case 39: return 3
  case 92: return 4
  case 34: return -1
  case 102: return 2
  case 110: return 2
  case 114: return 2
  case 116: return 2
  case 117: return 2
  case 123: return 2
  case 52: return 2
  default:
    switch {
    case 48 <= r && r <= 57: return 2
    case 65 <= r && r <= 70: return 2
    case 97 <= r && r <= 102: return 2
    default: return 2
    }
  }
  panic("unreachable")
}
fun[8] = func(r rune) int {
  switch(r) {
  case 39: return 3
  case 92: return 4
  case 34: return -1
  case 102: return 2
  case 110: return 2
  case 114: return 2
  case 116: return 2
  case 117: return 2
  case 123: return 2
  case 52: return 2
  case 47: return 2
  case 98: return 2
  case 125: return 2
  default:
    switch {
    case 48 <= r && r <= 57: return 2
    case 65 <= r && r <= 70: return 2
    case 97 <= r && r <= 102: return 2
    default: return 2
    }
  }
  panic("unreachable")
}
fun[9] = func(r rune) int {
  switch(r) {
  case 47: return 2
  case 98: return 2
  case 125: return 2
  case 39: return 3
  case 92: return 4
  case 34: return -1
  case 102: return 2
  case 110: return 2
  case 114: return 2
  case 116: return 2
  case 117: return 2
  case 123: return 2
  case 52: return 2
  default:
    switch {
    case 48 <= r && r <= 57: return 2
    case 65 <= r && r <= 70: return 2
    case 97 <= r && r <= 102: return 2
    default: return 2
    }
  }
  panic("unreachable")
}
fun[10] = func(r rune) int {
  switch(r) {
  case 47: return 2
  case 98: return 2
  case 125: return 2
  case 39: return 3
  case 92: return 4
  case 34: return -1
  case 102: return 2
  case 110: return 2
  case 114: return 2
  case 116: return 2
  case 117: return 2
  case 123: return 2
  case 52: return 2
  default:
    switch {
    case 48 <= r && r <= 57: return 2
    case 65 <= r && r <= 70: return 2
    case 97 <= r && r <= 102: return 2
    default: return 2
    }
  }
  panic("unreachable")
}
fun[11] = func(r rune) int {
  switch(r) {
  case 39: return -1
  case 92: return -1
  case 34: return -1
  case 102: return 14
  case 110: return -1
  case 114: return -1
  case 116: return -1
  case 117: return -1
  case 123: return -1
  case 52: return 14
  case 47: return -1
  case 98: return 14
  case 125: return -1
  default:
    switch {
    case 48 <= r && r <= 57: return 14
    case 65 <= r && r <= 70: return 14
    case 97 <= r && r <= 102: return 14
    default: return -1
    }
  }
  panic("unreachable")
}
fun[12] = func(r rune) int {
  switch(r) {
  case 47: return 2
  case 98: return 2
  case 125: return 2
  case 39: return 3
  case 92: return 4
  case 34: return -1
  case 102: return 2
  case 110: return 2
  case 114: return 2
  case 116: return 2
  case 117: return 2
  case 123: return 2
  case 52: return 2
  default:
    switch {
    case 48 <= r && r <= 57: return 2
    case 65 <= r && r <= 70: return 2
    case 97 <= r && r <= 102: return 2
    default: return 2
    }
  }
  panic("unreachable")
}
fun[13] = func(r rune) int {
  switch(r) {
  case 47: return 2
  case 98: return 2
  case 125: return 2
  case 39: return 3
  case 92: return 4
  case 34: return -1
  case 102: return 2
  case 110: return 2
  case 114: return 2
  case 116: return 2
  case 117: return 2
  case 123: return 2
  case 52: return 2
  default:
    switch {
    case 48 <= r && r <= 57: return 2
    case 65 <= r && r <= 70: return 2
    case 97 <= r && r <= 102: return 2
    default: return 2
    }
  }
  panic("unreachable")
}
fun[18] = func(r rune) int {
  switch(r) {
  case 47: return -1
  case 98: return -1
  case 125: return -1
  case 39: return -1
  case 92: return -1
  case 34: return -1
  case 102: return -1
  case 110: return -1
  case 114: return -1
  case 116: return -1
  case 117: return -1
  case 123: return -1
  case 52: return 19
  default:
    switch {
    case 48 <= r && r <= 57: return -1
    case 65 <= r && r <= 70: return -1
    case 97 <= r && r <= 102: return -1
    default: return -1
    }
  }
  panic("unreachable")
}
a0[1].acc = acc[:]
a0[1].f = fun[:]
a0[1].id = 1
}
{
var acc [24]bool
var fun [24]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 116: return -1
  case 117: return -1
  case 34: return -1
  case 102: return -1
  case 114: return -1
  case 123: return -1
  case 52: return -1
  case 125: return -1
  case 96: return 1
  case 47: return -1
  case 98: return -1
  case 92: return -1
  case 110: return -1
  case 105: return -1
  default:
    switch {
    case 48 <= r && r <= 57: return -1
    case 65 <= r && r <= 70: return -1
    case 97 <= r && r <= 102: return -1
    default: return -1
    }
  }
  panic("unreachable")
}
fun[4] = func(r rune) int {
  switch(r) {
  case 116: return 5
  case 117: return 6
  case 34: return 7
  case 102: return 8
  case 114: return 9
  case 123: return -1
  case 52: return -1
  case 125: return -1
  case 96: return -1
  case 47: return 10
  case 98: return 11
  case 92: return 12
  case 110: return 13
  case 105: return -1
  default:
    switch {
    case 48 <= r && r <= 57: return -1
    case 65 <= r && r <= 70: return -1
    case 97 <= r && r <= 102: return -1
    default: return -1
    }
  }
  panic("unreachable")
}
fun[12] = func(r rune) int {
  switch(r) {
  case 92: return 4
  case 110: return 2
  case 105: return 2
  case 116: return 2
  case 117: return 2
  case 34: return -1
  case 102: return 2
  case 114: return 2
  case 123: return 2
  case 52: return 2
  case 125: return 2
  case 96: return 14
  case 47: return 2
  case 98: return 2
  default:
    switch {
    case 48 <= r && r <= 57: return 2
    case 65 <= r && r <= 70: return 2
    case 97 <= r && r <= 102: return 2
    default: return 2
    }
  }
  panic("unreachable")
}
fun[14] = func(r rune) int {
  switch(r) {
  case 92: return -1
  case 110: return -1
  case 105: return 15
  case 116: return -1
  case 117: return -1
  case 34: return -1
  case 102: return -1
  case 114: return -1
  case 123: return -1
  case 52: return -1
  case 125: return -1
  case 96: return 16
  case 47: return -1
  case 98: return -1
  default:
    switch {
    case 48 <= r && r <= 57: return -1
    case 65 <= r && r <= 70: return -1
    case 97 <= r && r <= 102: return -1
    default: return -1
    }
  }
  panic("unreachable")
}
fun[16] = func(r rune) int {
  switch(r) {
  case 92: return 4
  case 110: return 2
  case 105: return 2
  case 116: return 2
  case 117: return 2
  case 34: return -1
  case 102: return 2
  case 114: return 2
  case 123: return 2
  case 52: return 2
  case 125: return 2
  case 96: return 14
  case 47: return 2
  case 98: return 2
  default:
    switch {
    case 48 <= r && r <= 57: return 2
    case 65 <= r && r <= 70: return 2
    case 97 <= r && r <= 102: return 2
    default: return 2
    }
  }
  panic("unreachable")
}
fun[19] = func(r rune) int {
  switch(r) {
  case 116: return -1
  case 117: return -1
  case 34: return -1
  case 102: return 20
  case 114: return -1
  case 123: return -1
  case 52: return 20
  case 125: return -1
  case 96: return -1
  case 47: return -1
  case 98: return 20
  case 92: return -1
  case 110: return -1
  case 105: return -1
  default:
    switch {
    case 48 <= r && r <= 57: return 20
    case 65 <= r && r <= 70: return 20
    case 97 <= r && r <= 102: return 20
    default: return -1
    }
  }
  panic("unreachable")
}
fun[11] = func(r rune) int {
  switch(r) {
  case 96: return 14
  case 47: return 2
  case 98: return 2
  case 92: return 4
  case 110: return 2
  case 105: return 2
  case 116: return 2
  case 117: return 2
  case 34: return -1
  case 102: return 2
  case 114: return 2
  case 123: return 2
  case 52: return 2
  case 125: return 2
  default:
    switch {
    case 48 <= r && r <= 57: return 2
    case 65 <= r && r <= 70: return 2
    case 97 <= r && r <= 102: return 2
    default: return 2
    }
  }
  panic("unreachable")
}
fun[17] = func(r rune) int {
  switch(r) {
  case 34: return -1
  case 102: return 18
  case 114: return -1
  case 123: return -1
  case 52: return 18
  case 125: return -1
  case 96: return -1
  case 47: return -1
  case 98: return 18
  case 92: return -1
  case 110: return -1
  case 105: return -1
  case 116: return -1
  case 117: return -1
  default:
    switch {
    case 48 <= r && r <= 57: return 18
    case 65 <= r && r <= 70: return 18
    case 97 <= r && r <= 102: return 18
    default: return -1
    }
  }
  panic("unreachable")
}
fun[22] = func(r rune) int {
  switch(r) {
  case 116: return -1
  case 117: return -1
  case 34: return -1
  case 102: return -1
  case 114: return -1
  case 123: return -1
  case 52: return -1
  case 125: return 23
  case 96: return -1
  case 47: return -1
  case 98: return -1
  case 92: return -1
  case 110: return -1
  case 105: return -1
  default:
    switch {
    case 48 <= r && r <= 57: return -1
    case 65 <= r && r <= 70: return -1
    case 97 <= r && r <= 102: return -1
    default: return -1
    }
  }
  panic("unreachable")
}
fun[5] = func(r rune) int {
  switch(r) {
  case 92: return 4
  case 110: return 2
  case 105: return 2
  case 116: return 2
  case 117: return 2
  case 34: return -1
  case 102: return 2
  case 114: return 2
  case 123: return 2
  case 52: return 2
  case 125: return 2
  case 96: return 14
  case 47: return 2
  case 98: return 2
  default:
    switch {
    case 48 <= r && r <= 57: return 2
    case 65 <= r && r <= 70: return 2
    case 97 <= r && r <= 102: return 2
    default: return 2
    }
  }
  panic("unreachable")
}
fun[10] = func(r rune) int {
  switch(r) {
  case 34: return -1
  case 102: return 2
  case 114: return 2
  case 123: return 2
  case 52: return 2
  case 125: return 2
  case 96: return 14
  case 47: return 2
  case 98: return 2
  case 92: return 4
  case 110: return 2
  case 105: return 2
  case 116: return 2
  case 117: return 2
  default:
    switch {
    case 48 <= r && r <= 57: return 2
    case 65 <= r && r <= 70: return 2
    case 97 <= r && r <= 102: return 2
    default: return 2
    }
  }
  panic("unreachable")
}
fun[13] = func(r rune) int {
  switch(r) {
  case 34: return -1
  case 102: return 2
  case 114: return 2
  case 123: return 2
  case 52: return 2
  case 125: return 2
  case 96: return 14
  case 47: return 2
  case 98: return 2
  case 92: return 4
  case 110: return 2
  case 105: return 2
  case 116: return 2
  case 117: return 2
  default:
    switch {
    case 48 <= r && r <= 57: return 2
    case 65 <= r && r <= 70: return 2
    case 97 <= r && r <= 102: return 2
    default: return 2
    }
  }
  panic("unreachable")
}
fun[21] = func(r rune) int {
  switch(r) {
  case 96: return -1
  case 47: return -1
  case 98: return -1
  case 92: return -1
  case 110: return -1
  case 105: return -1
  case 116: return -1
  case 117: return -1
  case 34: return -1
  case 102: return -1
  case 114: return -1
  case 123: return -1
  case 52: return 22
  case 125: return -1
  default:
    switch {
    case 48 <= r && r <= 57: return -1
    case 65 <= r && r <= 70: return -1
    case 97 <= r && r <= 102: return -1
    default: return -1
    }
  }
  panic("unreachable")
}
fun[23] = func(r rune) int {
  switch(r) {
  case 34: return -1
  case 102: return 2
  case 114: return 2
  case 123: return 2
  case 52: return 2
  case 125: return 2
  case 96: return 14
  case 47: return 2
  case 98: return 2
  case 92: return 4
  case 110: return 2
  case 105: return 2
  case 116: return 2
  case 117: return 2
  default:
    switch {
    case 48 <= r && r <= 57: return 2
    case 65 <= r && r <= 70: return 2
    case 97 <= r && r <= 102: return 2
    default: return 2
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 116: return 2
  case 117: return 2
  case 34: return -1
  case 102: return 2
  case 114: return 2
  case 123: return 2
  case 52: return 2
  case 125: return 2
  case 96: return 3
  case 47: return 2
  case 98: return 2
  case 92: return 4
  case 110: return 2
  case 105: return 2
  default:
    switch {
    case 48 <= r && r <= 57: return 2
    case 65 <= r && r <= 70: return 2
    case 97 <= r && r <= 102: return 2
    default: return 2
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 96: return 14
  case 47: return 2
  case 98: return 2
  case 92: return 4
  case 110: return 2
  case 105: return 2
  case 116: return 2
  case 117: return 2
  case 34: return -1
  case 102: return 2
  case 114: return 2
  case 123: return 2
  case 52: return 2
  case 125: return 2
  default:
    switch {
    case 48 <= r && r <= 57: return 2
    case 65 <= r && r <= 70: return 2
    case 97 <= r && r <= 102: return 2
    default: return 2
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 92: return -1
  case 110: return -1
  case 105: return -1
  case 116: return -1
  case 117: return -1
  case 34: return -1
  case 102: return -1
  case 114: return -1
  case 123: return -1
  case 52: return -1
  case 125: return -1
  case 96: return 16
  case 47: return -1
  case 98: return -1
  default:
    switch {
    case 48 <= r && r <= 57: return -1
    case 65 <= r && r <= 70: return -1
    case 97 <= r && r <= 102: return -1
    default: return -1
    }
  }
  panic("unreachable")
}
fun[6] = func(r rune) int {
  switch(r) {
  case 116: return -1
  case 117: return -1
  case 34: return -1
  case 102: return 17
  case 114: return -1
  case 123: return -1
  case 52: return 17
  case 125: return -1
  case 96: return -1
  case 47: return -1
  case 98: return 17
  case 92: return -1
  case 110: return -1
  case 105: return -1
  default:
    switch {
    case 48 <= r && r <= 57: return 17
    case 65 <= r && r <= 70: return 17
    case 97 <= r && r <= 102: return 17
    default: return -1
    }
  }
  panic("unreachable")
}
fun[7] = func(r rune) int {
  switch(r) {
  case 96: return 14
  case 47: return 2
  case 98: return 2
  case 92: return 4
  case 110: return 2
  case 105: return 2
  case 116: return 2
  case 117: return 2
  case 34: return -1
  case 102: return 2
  case 114: return 2
  case 123: return 2
  case 52: return 2
  case 125: return 2
  default:
    switch {
    case 48 <= r && r <= 57: return 2
    case 65 <= r && r <= 70: return 2
    case 97 <= r && r <= 102: return 2
    default: return 2
    }
  }
  panic("unreachable")
}
fun[8] = func(r rune) int {
  switch(r) {
  case 92: return 4
  case 110: return 2
  case 105: return 2
  case 116: return 2
  case 117: return 2
  case 34: return -1
  case 102: return 2
  case 114: return 2
  case 123: return 2
  case 52: return 2
  case 125: return 2
  case 96: return 14
  case 47: return 2
  case 98: return 2
  default:
    switch {
    case 48 <= r && r <= 57: return 2
    case 65 <= r && r <= 70: return 2
    case 97 <= r && r <= 102: return 2
    default: return 2
    }
  }
  panic("unreachable")
}
fun[9] = func(r rune) int {
  switch(r) {
  case 92: return 4
  case 110: return 2
  case 105: return 2
  case 116: return 2
  case 117: return 2
  case 34: return -1
  case 102: return 2
  case 114: return 2
  case 123: return 2
  case 52: return 2
  case 125: return 2
  case 96: return 14
  case 47: return 2
  case 98: return 2
  default:
    switch {
    case 48 <= r && r <= 57: return 2
    case 65 <= r && r <= 70: return 2
    case 97 <= r && r <= 102: return 2
    default: return 2
    }
  }
  panic("unreachable")
}
acc[15] = true
fun[15] = func(r rune) int {
  switch(r) {
  case 92: return -1
  case 110: return -1
  case 105: return -1
  case 116: return -1
  case 117: return -1
  case 34: return -1
  case 102: return -1
  case 114: return -1
  case 123: return -1
  case 52: return -1
  case 125: return -1
  case 96: return -1
  case 47: return -1
  case 98: return -1
  default:
    switch {
    case 48 <= r && r <= 57: return -1
    case 65 <= r && r <= 70: return -1
    case 97 <= r && r <= 102: return -1
    default: return -1
    }
  }
  panic("unreachable")
}
fun[18] = func(r rune) int {
  switch(r) {
  case 96: return -1
  case 47: return -1
  case 98: return 19
  case 92: return -1
  case 110: return -1
  case 105: return -1
  case 116: return -1
  case 117: return -1
  case 34: return -1
  case 102: return 19
  case 114: return -1
  case 123: return -1
  case 52: return 19
  case 125: return -1
  default:
    switch {
    case 48 <= r && r <= 57: return 19
    case 65 <= r && r <= 70: return 19
    case 97 <= r && r <= 102: return 19
    default: return -1
    }
  }
  panic("unreachable")
}
fun[20] = func(r rune) int {
  switch(r) {
  case 34: return -1
  case 102: return -1
  case 114: return -1
  case 123: return 21
  case 52: return -1
  case 125: return -1
  case 96: return -1
  case 47: return -1
  case 98: return -1
  case 92: return -1
  case 110: return -1
  case 105: return -1
  case 116: return -1
  case 117: return -1
  default:
    switch {
    case 48 <= r && r <= 57: return -1
    case 65 <= r && r <= 70: return -1
    case 97 <= r && r <= 102: return -1
    default: return -1
    }
  }
  panic("unreachable")
}
a0[2].acc = acc[:]
a0[2].f = fun[:]
a0[2].id = 2
}
{
var acc [23]bool
var fun [23]func(rune) int
fun[4] = func(r rune) int {
  switch(r) {
  case 34: return -1
  case 98: return 4
  case 110: return 4
  case 116: return 4
  case 52: return 4
  case 125: return 4
  case 96: return 5
  case 92: return 3
  case 47: return 4
  case 102: return 4
  case 114: return 4
  case 117: return 4
  case 123: return 4
  default:
    switch {
    case 48 <= r && r <= 57: return 4
    case 65 <= r && r <= 70: return 4
    case 97 <= r && r <= 102: return 4
    default: return 4
    }
  }
  panic("unreachable")
}
fun[8] = func(r rune) int {
  switch(r) {
  case 34: return -1
  case 98: return 4
  case 110: return 4
  case 116: return 4
  case 52: return 4
  case 125: return 4
  case 96: return 5
  case 92: return 3
  case 47: return 4
  case 102: return 4
  case 114: return 4
  case 117: return 4
  case 123: return 4
  default:
    switch {
    case 48 <= r && r <= 57: return 4
    case 65 <= r && r <= 70: return 4
    case 97 <= r && r <= 102: return 4
    default: return 4
    }
  }
  panic("unreachable")
}
fun[14] = func(r rune) int {
  switch(r) {
  case 96: return 5
  case 92: return 3
  case 47: return 4
  case 102: return 4
  case 114: return 4
  case 117: return 4
  case 123: return 4
  case 34: return -1
  case 98: return 4
  case 110: return 4
  case 116: return 4
  case 52: return 4
  case 125: return 4
  default:
    switch {
    case 48 <= r && r <= 57: return 4
    case 65 <= r && r <= 70: return 4
    case 97 <= r && r <= 102: return 4
    default: return 4
    }
  }
  panic("unreachable")
}
fun[15] = func(r rune) int {
  switch(r) {
  case 34: return -1
  case 98: return 4
  case 110: return 4
  case 116: return 4
  case 52: return 4
  case 125: return 4
  case 96: return 5
  case 92: return 3
  case 47: return 4
  case 102: return 4
  case 114: return 4
  case 117: return 4
  case 123: return 4
  default:
    switch {
    case 48 <= r && r <= 57: return 4
    case 65 <= r && r <= 70: return 4
    case 97 <= r && r <= 102: return 4
    default: return 4
    }
  }
  panic("unreachable")
}
fun[19] = func(r rune) int {
  switch(r) {
  case 96: return -1
  case 92: return -1
  case 47: return -1
  case 102: return -1
  case 114: return -1
  case 117: return -1
  case 123: return 20
  case 34: return -1
  case 98: return -1
  case 110: return -1
  case 116: return -1
  case 52: return -1
  case 125: return -1
  default:
    switch {
    case 48 <= r && r <= 57: return -1
    case 65 <= r && r <= 70: return -1
    case 97 <= r && r <= 102: return -1
    default: return -1
    }
  }
  panic("unreachable")
}
fun[20] = func(r rune) int {
  switch(r) {
  case 34: return -1
  case 98: return -1
  case 110: return -1
  case 116: return -1
  case 52: return 21
  case 125: return -1
  case 96: return -1
  case 92: return -1
  case 47: return -1
  case 102: return -1
  case 114: return -1
  case 117: return -1
  case 123: return -1
  default:
    switch {
    case 48 <= r && r <= 57: return -1
    case 65 <= r && r <= 70: return -1
    case 97 <= r && r <= 102: return -1
    default: return -1
    }
  }
  panic("unreachable")
}
fun[21] = func(r rune) int {
  switch(r) {
  case 96: return -1
  case 92: return -1
  case 47: return -1
  case 102: return -1
  case 114: return -1
  case 117: return -1
  case 123: return -1
  case 34: return -1
  case 98: return -1
  case 110: return -1
  case 116: return -1
  case 52: return -1
  case 125: return 22
  default:
    switch {
    case 48 <= r && r <= 57: return -1
    case 65 <= r && r <= 70: return -1
    case 97 <= r && r <= 102: return -1
    default: return -1
    }
  }
  panic("unreachable")
}
fun[22] = func(r rune) int {
  switch(r) {
  case 34: return -1
  case 98: return 4
  case 110: return 4
  case 116: return 4
  case 52: return 4
  case 125: return 4
  case 96: return 5
  case 92: return 3
  case 47: return 4
  case 102: return 4
  case 114: return 4
  case 117: return 4
  case 123: return 4
  default:
    switch {
    case 48 <= r && r <= 57: return 4
    case 65 <= r && r <= 70: return 4
    case 97 <= r && r <= 102: return 4
    default: return 4
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 96: return 2
  case 92: return 3
  case 47: return 4
  case 102: return 4
  case 114: return 4
  case 117: return 4
  case 123: return 4
  case 34: return -1
  case 98: return 4
  case 110: return 4
  case 116: return 4
  case 52: return 4
  case 125: return 4
  default:
    switch {
    case 48 <= r && r <= 57: return 4
    case 65 <= r && r <= 70: return 4
    case 97 <= r && r <= 102: return 4
    default: return 4
    }
  }
  panic("unreachable")
}
fun[9] = func(r rune) int {
  switch(r) {
  case 96: return 5
  case 92: return 3
  case 47: return 4
  case 102: return 4
  case 114: return 4
  case 117: return 4
  case 123: return 4
  case 34: return -1
  case 98: return 4
  case 110: return 4
  case 116: return 4
  case 52: return 4
  case 125: return 4
  default:
    switch {
    case 48 <= r && r <= 57: return 4
    case 65 <= r && r <= 70: return 4
    case 97 <= r && r <= 102: return 4
    default: return 4
    }
  }
  panic("unreachable")
}
fun[10] = func(r rune) int {
  switch(r) {
  case 96: return 5
  case 92: return 3
  case 47: return 4
  case 102: return 4
  case 114: return 4
  case 117: return 4
  case 123: return 4
  case 34: return -1
  case 98: return 4
  case 110: return 4
  case 116: return 4
  case 52: return 4
  case 125: return 4
  default:
    switch {
    case 48 <= r && r <= 57: return 4
    case 65 <= r && r <= 70: return 4
    case 97 <= r && r <= 102: return 4
    default: return 4
    }
  }
  panic("unreachable")
}
fun[12] = func(r rune) int {
  switch(r) {
  case 34: return -1
  case 98: return 4
  case 110: return 4
  case 116: return 4
  case 52: return 4
  case 125: return 4
  case 96: return 5
  case 92: return 3
  case 47: return 4
  case 102: return 4
  case 114: return 4
  case 117: return 4
  case 123: return 4
  default:
    switch {
    case 48 <= r && r <= 57: return 4
    case 65 <= r && r <= 70: return 4
    case 97 <= r && r <= 102: return 4
    default: return 4
    }
  }
  panic("unreachable")
}
fun[17] = func(r rune) int {
  switch(r) {
  case 96: return -1
  case 92: return -1
  case 47: return -1
  case 102: return 18
  case 114: return -1
  case 117: return -1
  case 123: return -1
  case 34: return -1
  case 98: return 18
  case 110: return -1
  case 116: return -1
  case 52: return 18
  case 125: return -1
  default:
    switch {
    case 48 <= r && r <= 57: return 18
    case 65 <= r && r <= 70: return 18
    case 97 <= r && r <= 102: return 18
    default: return -1
    }
  }
  panic("unreachable")
}
fun[0] = func(r rune) int {
  switch(r) {
  case 96: return 1
  case 92: return -1
  case 47: return -1
  case 102: return -1
  case 114: return -1
  case 117: return -1
  case 123: return -1
  case 34: return -1
  case 98: return -1
  case 110: return -1
  case 116: return -1
  case 52: return -1
  case 125: return -1
  default:
    switch {
    case 48 <= r && r <= 57: return -1
    case 65 <= r && r <= 70: return -1
    case 97 <= r && r <= 102: return -1
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 96: return -1
  case 92: return 7
  case 47: return 8
  case 102: return 9
  case 114: return 10
  case 117: return 11
  case 123: return -1
  case 34: return 12
  case 98: return 13
  case 110: return 14
  case 116: return 15
  case 52: return -1
  case 125: return -1
  default:
    switch {
    case 48 <= r && r <= 57: return -1
    case 65 <= r && r <= 70: return -1
    case 97 <= r && r <= 102: return -1
    default: return -1
    }
  }
  panic("unreachable")
}
acc[5] = true
fun[5] = func(r rune) int {
  switch(r) {
  case 34: return -1
  case 98: return -1
  case 110: return -1
  case 116: return -1
  case 52: return -1
  case 125: return -1
  case 96: return 6
  case 92: return -1
  case 47: return -1
  case 102: return -1
  case 114: return -1
  case 117: return -1
  case 123: return -1
  default:
    switch {
    case 48 <= r && r <= 57: return -1
    case 65 <= r && r <= 70: return -1
    case 97 <= r && r <= 102: return -1
    default: return -1
    }
  }
  panic("unreachable")
}
fun[6] = func(r rune) int {
  switch(r) {
  case 96: return 5
  case 92: return 3
  case 47: return 4
  case 102: return 4
  case 114: return 4
  case 117: return 4
  case 123: return 4
  case 34: return -1
  case 98: return 4
  case 110: return 4
  case 116: return 4
  case 52: return 4
  case 125: return 4
  default:
    switch {
    case 48 <= r && r <= 57: return 4
    case 65 <= r && r <= 70: return 4
    case 97 <= r && r <= 102: return 4
    default: return 4
    }
  }
  panic("unreachable")
}
fun[11] = func(r rune) int {
  switch(r) {
  case 34: return -1
  case 98: return 16
  case 110: return -1
  case 116: return -1
  case 52: return 16
  case 125: return -1
  case 96: return -1
  case 92: return -1
  case 47: return -1
  case 102: return 16
  case 114: return -1
  case 117: return -1
  case 123: return -1
  default:
    switch {
    case 48 <= r && r <= 57: return 16
    case 65 <= r && r <= 70: return 16
    case 97 <= r && r <= 102: return 16
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 34: return -1
  case 98: return -1
  case 110: return -1
  case 116: return -1
  case 52: return -1
  case 125: return -1
  case 96: return 6
  case 92: return -1
  case 47: return -1
  case 102: return -1
  case 114: return -1
  case 117: return -1
  case 123: return -1
  default:
    switch {
    case 48 <= r && r <= 57: return -1
    case 65 <= r && r <= 70: return -1
    case 97 <= r && r <= 102: return -1
    default: return -1
    }
  }
  panic("unreachable")
}
fun[7] = func(r rune) int {
  switch(r) {
  case 96: return 5
  case 92: return 3
  case 47: return 4
  case 102: return 4
  case 114: return 4
  case 117: return 4
  case 123: return 4
  case 34: return -1
  case 98: return 4
  case 110: return 4
  case 116: return 4
  case 52: return 4
  case 125: return 4
  default:
    switch {
    case 48 <= r && r <= 57: return 4
    case 65 <= r && r <= 70: return 4
    case 97 <= r && r <= 102: return 4
    default: return 4
    }
  }
  panic("unreachable")
}
fun[13] = func(r rune) int {
  switch(r) {
  case 96: return 5
  case 92: return 3
  case 47: return 4
  case 102: return 4
  case 114: return 4
  case 117: return 4
  case 123: return 4
  case 34: return -1
  case 98: return 4
  case 110: return 4
  case 116: return 4
  case 52: return 4
  case 125: return 4
  default:
    switch {
    case 48 <= r && r <= 57: return 4
    case 65 <= r && r <= 70: return 4
    case 97 <= r && r <= 102: return 4
    default: return 4
    }
  }
  panic("unreachable")
}
fun[16] = func(r rune) int {
  switch(r) {
  case 34: return -1
  case 98: return 17
  case 110: return -1
  case 116: return -1
  case 52: return 17
  case 125: return -1
  case 96: return -1
  case 92: return -1
  case 47: return -1
  case 102: return 17
  case 114: return -1
  case 117: return -1
  case 123: return -1
  default:
    switch {
    case 48 <= r && r <= 57: return 17
    case 65 <= r && r <= 70: return 17
    case 97 <= r && r <= 102: return 17
    default: return -1
    }
  }
  panic("unreachable")
}
fun[18] = func(r rune) int {
  switch(r) {
  case 34: return -1
  case 98: return 19
  case 110: return -1
  case 116: return -1
  case 52: return 19
  case 125: return -1
  case 96: return -1
  case 92: return -1
  case 47: return -1
  case 102: return 19
  case 114: return -1
  case 117: return -1
  case 123: return -1
  default:
    switch {
    case 48 <= r && r <= 57: return 19
    case 65 <= r && r <= 70: return 19
    case 97 <= r && r <= 102: return 19
    default: return -1
    }
  }
  panic("unreachable")
}
a0[3].acc = acc[:]
a0[3].f = fun[:]
a0[3].id = 3
}
{
var acc [9]bool
var fun [9]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 48: return 1
  case 46: return -1
  case 101: return -1
  case 69: return -1
  case 43: return -1
  case 45: return -1
  default:
    switch {
    case 48 <= r && r <= 48: return -1
    case 49 <= r && r <= 57: return 2
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 48: return 3
  case 46: return 4
  case 101: return -1
  case 69: return -1
  case 43: return -1
  case 45: return -1
  default:
    switch {
    case 48 <= r && r <= 48: return 3
    case 49 <= r && r <= 57: return 3
    default: return -1
    }
  }
  panic("unreachable")
}
fun[6] = func(r rune) int {
  switch(r) {
  case 48: return 7
  case 46: return -1
  case 101: return -1
  case 69: return -1
  case 43: return 8
  case 45: return 8
  default:
    switch {
    case 48 <= r && r <= 48: return 7
    case 49 <= r && r <= 57: return 7
    default: return -1
    }
  }
  panic("unreachable")
}
acc[7] = true
fun[7] = func(r rune) int {
  switch(r) {
  case 48: return 7
  case 46: return -1
  case 101: return -1
  case 69: return -1
  case 43: return -1
  case 45: return -1
  default:
    switch {
    case 48 <= r && r <= 48: return 7
    case 49 <= r && r <= 57: return 7
    default: return -1
    }
  }
  panic("unreachable")
}
fun[8] = func(r rune) int {
  switch(r) {
  case 48: return 7
  case 46: return -1
  case 101: return -1
  case 69: return -1
  case 43: return -1
  case 45: return -1
  default:
    switch {
    case 48 <= r && r <= 48: return 7
    case 49 <= r && r <= 57: return 7
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 48: return -1
  case 46: return 4
  case 101: return -1
  case 69: return -1
  case 43: return -1
  case 45: return -1
  default:
    switch {
    case 48 <= r && r <= 48: return -1
    case 49 <= r && r <= 57: return -1
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 48: return 3
  case 46: return 4
  case 101: return -1
  case 69: return -1
  case 43: return -1
  case 45: return -1
  default:
    switch {
    case 48 <= r && r <= 48: return 3
    case 49 <= r && r <= 57: return 3
    default: return -1
    }
  }
  panic("unreachable")
}
fun[4] = func(r rune) int {
  switch(r) {
  case 48: return 5
  case 46: return -1
  case 101: return -1
  case 69: return -1
  case 43: return -1
  case 45: return -1
  default:
    switch {
    case 48 <= r && r <= 48: return 5
    case 49 <= r && r <= 57: return 5
    default: return -1
    }
  }
  panic("unreachable")
}
acc[5] = true
fun[5] = func(r rune) int {
  switch(r) {
  case 48: return 5
  case 46: return -1
  case 101: return 6
  case 69: return 6
  case 43: return -1
  case 45: return -1
  default:
    switch {
    case 48 <= r && r <= 48: return 5
    case 49 <= r && r <= 57: return 5
    default: return -1
    }
  }
  panic("unreachable")
}
a0[4].acc = acc[:]
a0[4].f = fun[:]
a0[4].id = 4
}
{
var acc [7]bool
var fun [7]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 48: return 1
  case 101: return -1
  case 69: return -1
  case 43: return -1
  case 45: return -1
  default:
    switch {
    case 48 <= r && r <= 48: return -1
    case 49 <= r && r <= 57: return 2
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 48: return -1
  case 101: return 4
  case 69: return 4
  case 43: return -1
  case 45: return -1
  default:
    switch {
    case 48 <= r && r <= 48: return -1
    case 49 <= r && r <= 57: return -1
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 48: return 3
  case 101: return 4
  case 69: return 4
  case 43: return -1
  case 45: return -1
  default:
    switch {
    case 48 <= r && r <= 48: return 3
    case 49 <= r && r <= 57: return 3
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 48: return 3
  case 101: return 4
  case 69: return 4
  case 43: return -1
  case 45: return -1
  default:
    switch {
    case 48 <= r && r <= 48: return 3
    case 49 <= r && r <= 57: return 3
    default: return -1
    }
  }
  panic("unreachable")
}
fun[4] = func(r rune) int {
  switch(r) {
  case 48: return 5
  case 101: return -1
  case 69: return -1
  case 43: return 6
  case 45: return 6
  default:
    switch {
    case 48 <= r && r <= 48: return 5
    case 49 <= r && r <= 57: return 5
    default: return -1
    }
  }
  panic("unreachable")
}
acc[5] = true
fun[5] = func(r rune) int {
  switch(r) {
  case 48: return 5
  case 101: return -1
  case 69: return -1
  case 43: return -1
  case 45: return -1
  default:
    switch {
    case 48 <= r && r <= 48: return 5
    case 49 <= r && r <= 57: return 5
    default: return -1
    }
  }
  panic("unreachable")
}
fun[6] = func(r rune) int {
  switch(r) {
  case 48: return 5
  case 101: return -1
  case 69: return -1
  case 43: return -1
  case 45: return -1
  default:
    switch {
    case 48 <= r && r <= 48: return 5
    case 49 <= r && r <= 57: return 5
    default: return -1
    }
  }
  panic("unreachable")
}
a0[5].acc = acc[:]
a0[5].f = fun[:]
a0[5].id = 5
}
{
var acc [4]bool
var fun [4]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 48: return 1
  default:
    switch {
    case 48 <= r && r <= 48: return -1
    case 49 <= r && r <= 57: return 2
    default: return -1
    }
  }
  panic("unreachable")
}
acc[1] = true
fun[1] = func(r rune) int {
  switch(r) {
  case 48: return -1
  default:
    switch {
    case 48 <= r && r <= 48: return -1
    case 49 <= r && r <= 57: return -1
    default: return -1
    }
  }
  panic("unreachable")
}
acc[2] = true
fun[2] = func(r rune) int {
  switch(r) {
  case 48: return 3
  default:
    switch {
    case 48 <= r && r <= 48: return 3
    case 49 <= r && r <= 57: return 3
    default: return -1
    }
  }
  panic("unreachable")
}
acc[3] = true
fun[3] = func(r rune) int {
  switch(r) {
  case 48: return 3
  default:
    switch {
    case 48 <= r && r <= 48: return 3
    case 49 <= r && r <= 57: return 3
    default: return -1
    }
  }
  panic("unreachable")
}
a0[6].acc = acc[:]
a0[6].f = fun[:]
a0[6].id = 6
}
{
var acc [10]bool
var fun [10]func(rune) int
fun[1] = func(r rune) int {
  switch(r) {
  case 47: return -1
  case 42: return 2
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 47: return 3
  case 42: return 4
  default:
    switch {
    default: return 3
    }
  }
  panic("unreachable")
}
fun[4] = func(r rune) int {
  switch(r) {
  case 47: return 5
  case 42: return 6
  default:
    switch {
    default: return 7
    }
  }
  panic("unreachable")
}
fun[6] = func(r rune) int {
  switch(r) {
  case 47: return 8
  case 42: return 6
  default:
    switch {
    default: return 9
    }
  }
  panic("unreachable")
}
fun[7] = func(r rune) int {
  switch(r) {
  case 47: return 3
  case 42: return 4
  default:
    switch {
    default: return 3
    }
  }
  panic("unreachable")
}
fun[9] = func(r rune) int {
  switch(r) {
  case 47: return 3
  case 42: return 4
  default:
    switch {
    default: return 3
    }
  }
  panic("unreachable")
}
fun[0] = func(r rune) int {
  switch(r) {
  case 47: return 1
  case 42: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 47: return 3
  case 42: return 4
  default:
    switch {
    default: return 3
    }
  }
  panic("unreachable")
}
acc[5] = true
fun[5] = func(r rune) int {
  switch(r) {
  case 47: return -1
  case 42: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[8] = true
fun[8] = func(r rune) int {
  switch(r) {
  case 47: return 3
  case 42: return 4
  default:
    switch {
    default: return 3
    }
  }
  panic("unreachable")
}
a0[7].acc = acc[:]
a0[7].f = fun[:]
a0[7].id = 7
}
{
var acc [6]bool
var fun [6]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 34: return 1
  case 45: return -1
  case 10: return -1
  case 13: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 34: return -1
  case 45: return 2
  case 10: return -1
  case 13: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 34: return -1
  case 45: return 3
  case 10: return -1
  case 13: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 34: return 4
  case 45: return -1
  case 10: return -1
  case 13: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[4] = true
fun[4] = func(r rune) int {
  switch(r) {
  case 34: return 5
  case 45: return 5
  case 10: return -1
  case 13: return -1
  default:
    switch {
    default: return 5
    }
  }
  panic("unreachable")
}
acc[5] = true
fun[5] = func(r rune) int {
  switch(r) {
  case 34: return 5
  case 45: return 5
  case 10: return -1
  case 13: return -1
  default:
    switch {
    default: return 5
    }
  }
  panic("unreachable")
}
a0[8].acc = acc[:]
a0[8].f = fun[:]
a0[8].id = 8
}
{
var acc [2]bool
var fun [2]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 32: return 1
  case 9: return 1
  case 10: return 1
  case 13: return 1
  case 12: return 1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[1] = true
fun[1] = func(r rune) int {
  switch(r) {
  case 32: return 1
  case 9: return 1
  case 10: return 1
  case 13: return 1
  case 12: return 1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[9].acc = acc[:]
a0[9].f = fun[:]
a0[9].id = 9
}
{
var acc [2]bool
var fun [2]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 46: return 1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[1] = true
fun[1] = func(r rune) int {
  switch(r) {
  case 46: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[10].acc = acc[:]
a0[10].f = fun[:]
a0[10].id = 10
}
{
var acc [2]bool
var fun [2]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 43: return 1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[1] = true
fun[1] = func(r rune) int {
  switch(r) {
  case 43: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[11].acc = acc[:]
a0[11].f = fun[:]
a0[11].id = 11
}
{
var acc [2]bool
var fun [2]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 45: return 1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[1] = true
fun[1] = func(r rune) int {
  switch(r) {
  case 45: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[12].acc = acc[:]
a0[12].f = fun[:]
a0[12].id = 12
}
{
var acc [2]bool
var fun [2]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 42: return 1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[1] = true
fun[1] = func(r rune) int {
  switch(r) {
  case 42: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[13].acc = acc[:]
a0[13].f = fun[:]
a0[13].id = 13
}
{
var acc [2]bool
var fun [2]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 47: return 1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[1] = true
fun[1] = func(r rune) int {
  switch(r) {
  case 47: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[14].acc = acc[:]
a0[14].f = fun[:]
a0[14].id = 14
}
{
var acc [2]bool
var fun [2]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 37: return 1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[1] = true
fun[1] = func(r rune) int {
  switch(r) {
  case 37: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[15].acc = acc[:]
a0[15].f = fun[:]
a0[15].id = 15
}
{
var acc [3]bool
var fun [3]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 61: return 1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 61: return 2
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[2] = true
fun[2] = func(r rune) int {
  switch(r) {
  case 61: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[16].acc = acc[:]
a0[16].f = fun[:]
a0[16].id = 16
}
{
var acc [2]bool
var fun [2]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 61: return 1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[1] = true
fun[1] = func(r rune) int {
  switch(r) {
  case 61: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[17].acc = acc[:]
a0[17].f = fun[:]
a0[17].id = 17
}
{
var acc [3]bool
var fun [3]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 33: return 1
  case 61: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 33: return -1
  case 61: return 2
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[2] = true
fun[2] = func(r rune) int {
  switch(r) {
  case 33: return -1
  case 61: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[18].acc = acc[:]
a0[18].f = fun[:]
a0[18].id = 18
}
{
var acc [3]bool
var fun [3]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 60: return 1
  case 62: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 60: return -1
  case 62: return 2
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[2] = true
fun[2] = func(r rune) int {
  switch(r) {
  case 60: return -1
  case 62: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[19].acc = acc[:]
a0[19].f = fun[:]
a0[19].id = 19
}
{
var acc [2]bool
var fun [2]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 60: return 1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[1] = true
fun[1] = func(r rune) int {
  switch(r) {
  case 60: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[20].acc = acc[:]
a0[20].f = fun[:]
a0[20].id = 20
}
{
var acc [3]bool
var fun [3]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 60: return 1
  case 61: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 60: return -1
  case 61: return 2
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[2] = true
fun[2] = func(r rune) int {
  switch(r) {
  case 60: return -1
  case 61: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[21].acc = acc[:]
a0[21].f = fun[:]
a0[21].id = 21
}
{
var acc [2]bool
var fun [2]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 62: return 1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[1] = true
fun[1] = func(r rune) int {
  switch(r) {
  case 62: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[22].acc = acc[:]
a0[22].f = fun[:]
a0[22].id = 22
}
{
var acc [3]bool
var fun [3]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 62: return 1
  case 61: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 62: return -1
  case 61: return 2
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[2] = true
fun[2] = func(r rune) int {
  switch(r) {
  case 62: return -1
  case 61: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[23].acc = acc[:]
a0[23].f = fun[:]
a0[23].id = 23
}
{
var acc [3]bool
var fun [3]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 124: return 1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 124: return 2
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[2] = true
fun[2] = func(r rune) int {
  switch(r) {
  case 124: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[24].acc = acc[:]
a0[24].f = fun[:]
a0[24].id = 24
}
{
var acc [2]bool
var fun [2]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 40: return 1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[1] = true
fun[1] = func(r rune) int {
  switch(r) {
  case 40: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[25].acc = acc[:]
a0[25].f = fun[:]
a0[25].id = 25
}
{
var acc [2]bool
var fun [2]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 41: return 1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[1] = true
fun[1] = func(r rune) int {
  switch(r) {
  case 41: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[26].acc = acc[:]
a0[26].f = fun[:]
a0[26].id = 26
}
{
var acc [2]bool
var fun [2]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 123: return 1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[1] = true
fun[1] = func(r rune) int {
  switch(r) {
  case 123: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[27].acc = acc[:]
a0[27].f = fun[:]
a0[27].id = 27
}
{
var acc [2]bool
var fun [2]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 125: return 1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[1] = true
fun[1] = func(r rune) int {
  switch(r) {
  case 125: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[28].acc = acc[:]
a0[28].f = fun[:]
a0[28].id = 28
}
{
var acc [2]bool
var fun [2]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 44: return 1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[1] = true
fun[1] = func(r rune) int {
  switch(r) {
  case 44: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[29].acc = acc[:]
a0[29].f = fun[:]
a0[29].id = 29
}
{
var acc [2]bool
var fun [2]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 58: return 1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[1] = true
fun[1] = func(r rune) int {
  switch(r) {
  case 58: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[30].acc = acc[:]
a0[30].f = fun[:]
a0[30].id = 30
}
{
var acc [2]bool
var fun [2]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 91: return 1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[1] = true
fun[1] = func(r rune) int {
  switch(r) {
  case 91: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[31].acc = acc[:]
a0[31].f = fun[:]
a0[31].id = 31
}
{
var acc [2]bool
var fun [2]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 93: return 1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[1] = true
fun[1] = func(r rune) int {
  switch(r) {
  case 93: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[32].acc = acc[:]
a0[32].f = fun[:]
a0[32].id = 32
}
{
var acc [3]bool
var fun [3]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 93: return 1
  case 105: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 93: return -1
  case 105: return 2
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[2] = true
fun[2] = func(r rune) int {
  switch(r) {
  case 93: return -1
  case 105: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[33].acc = acc[:]
a0[33].f = fun[:]
a0[33].id = 33
}
{
var acc [4]bool
var fun [4]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 97: return 1
  case 65: return 1
  case 108: return -1
  case 76: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 97: return -1
  case 65: return -1
  case 108: return 2
  case 76: return 2
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 97: return -1
  case 65: return -1
  case 108: return 3
  case 76: return 3
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[3] = true
fun[3] = func(r rune) int {
  switch(r) {
  case 97: return -1
  case 65: return -1
  case 108: return -1
  case 76: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[34].acc = acc[:]
a0[34].f = fun[:]
a0[34].id = 34
}
{
var acc [6]bool
var fun [6]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 97: return 1
  case 76: return -1
  case 116: return -1
  case 84: return -1
  case 69: return -1
  case 65: return 1
  case 108: return -1
  case 101: return -1
  case 114: return -1
  case 82: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 97: return -1
  case 76: return 2
  case 116: return -1
  case 84: return -1
  case 69: return -1
  case 65: return -1
  case 108: return 2
  case 101: return -1
  case 114: return -1
  case 82: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 97: return -1
  case 76: return -1
  case 116: return 3
  case 84: return 3
  case 69: return -1
  case 65: return -1
  case 108: return -1
  case 101: return -1
  case 114: return -1
  case 82: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 97: return -1
  case 76: return -1
  case 116: return -1
  case 84: return -1
  case 69: return 4
  case 65: return -1
  case 108: return -1
  case 101: return 4
  case 114: return -1
  case 82: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[4] = func(r rune) int {
  switch(r) {
  case 65: return -1
  case 108: return -1
  case 101: return -1
  case 114: return 5
  case 82: return 5
  case 97: return -1
  case 76: return -1
  case 116: return -1
  case 84: return -1
  case 69: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[5] = true
fun[5] = func(r rune) int {
  switch(r) {
  case 65: return -1
  case 108: return -1
  case 101: return -1
  case 114: return -1
  case 82: return -1
  case 97: return -1
  case 76: return -1
  case 116: return -1
  case 84: return -1
  case 69: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[35].acc = acc[:]
a0[35].f = fun[:]
a0[35].id = 35
}
{
var acc [8]bool
var fun [8]func(rune) int
fun[1] = func(r rune) int {
  switch(r) {
  case 65: return -1
  case 78: return 2
  case 108: return -1
  case 89: return -1
  case 122: return -1
  case 101: return -1
  case 69: return -1
  case 97: return -1
  case 110: return 2
  case 76: return -1
  case 121: return -1
  case 90: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 97: return -1
  case 110: return -1
  case 76: return 4
  case 121: return -1
  case 90: return -1
  case 65: return -1
  case 78: return -1
  case 108: return 4
  case 89: return -1
  case 122: return -1
  case 101: return -1
  case 69: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[6] = func(r rune) int {
  switch(r) {
  case 65: return -1
  case 78: return -1
  case 108: return -1
  case 89: return -1
  case 122: return -1
  case 101: return 7
  case 69: return 7
  case 97: return -1
  case 110: return -1
  case 76: return -1
  case 121: return -1
  case 90: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[7] = true
fun[7] = func(r rune) int {
  switch(r) {
  case 97: return -1
  case 110: return -1
  case 76: return -1
  case 121: return -1
  case 90: return -1
  case 65: return -1
  case 78: return -1
  case 108: return -1
  case 89: return -1
  case 122: return -1
  case 101: return -1
  case 69: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[0] = func(r rune) int {
  switch(r) {
  case 65: return 1
  case 78: return -1
  case 108: return -1
  case 89: return -1
  case 122: return -1
  case 101: return -1
  case 69: return -1
  case 97: return 1
  case 110: return -1
  case 76: return -1
  case 121: return -1
  case 90: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 65: return 3
  case 78: return -1
  case 108: return -1
  case 89: return -1
  case 122: return -1
  case 101: return -1
  case 69: return -1
  case 97: return 3
  case 110: return -1
  case 76: return -1
  case 121: return -1
  case 90: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[4] = func(r rune) int {
  switch(r) {
  case 97: return -1
  case 110: return -1
  case 76: return -1
  case 121: return 5
  case 90: return -1
  case 65: return -1
  case 78: return -1
  case 108: return -1
  case 89: return 5
  case 122: return -1
  case 101: return -1
  case 69: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[5] = func(r rune) int {
  switch(r) {
  case 97: return -1
  case 110: return -1
  case 76: return -1
  case 121: return -1
  case 90: return 6
  case 65: return -1
  case 78: return -1
  case 108: return -1
  case 89: return -1
  case 122: return 6
  case 101: return -1
  case 69: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[36].acc = acc[:]
a0[36].f = fun[:]
a0[36].id = 36
}
{
var acc [4]bool
var fun [4]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 97: return 1
  case 65: return 1
  case 110: return -1
  case 78: return -1
  case 100: return -1
  case 68: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 97: return -1
  case 65: return -1
  case 110: return 2
  case 78: return 2
  case 100: return -1
  case 68: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 97: return -1
  case 65: return -1
  case 110: return -1
  case 78: return -1
  case 100: return 3
  case 68: return 3
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[3] = true
fun[3] = func(r rune) int {
  switch(r) {
  case 97: return -1
  case 65: return -1
  case 110: return -1
  case 78: return -1
  case 100: return -1
  case 68: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[37].acc = acc[:]
a0[37].f = fun[:]
a0[37].id = 37
}
{
var acc [4]bool
var fun [4]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 97: return 1
  case 65: return 1
  case 110: return -1
  case 78: return -1
  case 121: return -1
  case 89: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 97: return -1
  case 65: return -1
  case 110: return 2
  case 78: return 2
  case 121: return -1
  case 89: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 97: return -1
  case 65: return -1
  case 110: return -1
  case 78: return -1
  case 121: return 3
  case 89: return 3
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[3] = true
fun[3] = func(r rune) int {
  switch(r) {
  case 97: return -1
  case 65: return -1
  case 110: return -1
  case 78: return -1
  case 121: return -1
  case 89: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[38].acc = acc[:]
a0[38].f = fun[:]
a0[38].id = 38
}
{
var acc [6]bool
var fun [6]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 97: return 1
  case 65: return 1
  case 114: return -1
  case 82: return -1
  case 121: return -1
  case 89: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 97: return -1
  case 65: return -1
  case 114: return 2
  case 82: return 2
  case 121: return -1
  case 89: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 97: return -1
  case 65: return -1
  case 114: return 3
  case 82: return 3
  case 121: return -1
  case 89: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 97: return 4
  case 65: return 4
  case 114: return -1
  case 82: return -1
  case 121: return -1
  case 89: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[4] = func(r rune) int {
  switch(r) {
  case 97: return -1
  case 65: return -1
  case 114: return -1
  case 82: return -1
  case 121: return 5
  case 89: return 5
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[5] = true
fun[5] = func(r rune) int {
  switch(r) {
  case 97: return -1
  case 65: return -1
  case 114: return -1
  case 82: return -1
  case 121: return -1
  case 89: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[39].acc = acc[:]
a0[39].f = fun[:]
a0[39].id = 39
}
{
var acc [3]bool
var fun [3]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 97: return 1
  case 65: return 1
  case 115: return -1
  case 83: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 97: return -1
  case 65: return -1
  case 115: return 2
  case 83: return 2
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[2] = true
fun[2] = func(r rune) int {
  switch(r) {
  case 97: return -1
  case 65: return -1
  case 115: return -1
  case 83: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[40].acc = acc[:]
a0[40].f = fun[:]
a0[40].id = 40
}
{
var acc [4]bool
var fun [4]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 97: return 1
  case 65: return 1
  case 115: return -1
  case 83: return -1
  case 99: return -1
  case 67: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 97: return -1
  case 65: return -1
  case 115: return 2
  case 83: return 2
  case 99: return -1
  case 67: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 97: return -1
  case 65: return -1
  case 115: return -1
  case 83: return -1
  case 99: return 3
  case 67: return 3
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[3] = true
fun[3] = func(r rune) int {
  switch(r) {
  case 97: return -1
  case 65: return -1
  case 115: return -1
  case 83: return -1
  case 99: return -1
  case 67: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[41].acc = acc[:]
a0[41].f = fun[:]
a0[41].id = 41
}
{
var acc [6]bool
var fun [6]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 98: return 1
  case 66: return 1
  case 101: return -1
  case 69: return -1
  case 103: return -1
  case 73: return -1
  case 110: return -1
  case 71: return -1
  case 105: return -1
  case 78: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 71: return -1
  case 105: return -1
  case 78: return -1
  case 98: return -1
  case 66: return -1
  case 101: return 2
  case 69: return 2
  case 103: return -1
  case 73: return -1
  case 110: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 71: return 3
  case 105: return -1
  case 78: return -1
  case 98: return -1
  case 66: return -1
  case 101: return -1
  case 69: return -1
  case 103: return 3
  case 73: return -1
  case 110: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 71: return -1
  case 105: return 4
  case 78: return -1
  case 98: return -1
  case 66: return -1
  case 101: return -1
  case 69: return -1
  case 103: return -1
  case 73: return 4
  case 110: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[4] = func(r rune) int {
  switch(r) {
  case 98: return -1
  case 66: return -1
  case 101: return -1
  case 69: return -1
  case 103: return -1
  case 73: return -1
  case 110: return 5
  case 71: return -1
  case 105: return -1
  case 78: return 5
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[5] = true
fun[5] = func(r rune) int {
  switch(r) {
  case 98: return -1
  case 66: return -1
  case 101: return -1
  case 69: return -1
  case 103: return -1
  case 73: return -1
  case 110: return -1
  case 71: return -1
  case 105: return -1
  case 78: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[42].acc = acc[:]
a0[42].f = fun[:]
a0[42].id = 42
}
{
var acc [8]bool
var fun [8]func(rune) int
fun[2] = func(r rune) int {
  switch(r) {
  case 69: return -1
  case 116: return 3
  case 87: return -1
  case 110: return -1
  case 98: return -1
  case 66: return -1
  case 101: return -1
  case 84: return 3
  case 119: return -1
  case 78: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[5] = func(r rune) int {
  switch(r) {
  case 98: return -1
  case 66: return -1
  case 101: return 6
  case 84: return -1
  case 119: return -1
  case 78: return -1
  case 69: return 6
  case 116: return -1
  case 87: return -1
  case 110: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[7] = true
fun[7] = func(r rune) int {
  switch(r) {
  case 69: return -1
  case 116: return -1
  case 87: return -1
  case 110: return -1
  case 98: return -1
  case 66: return -1
  case 101: return -1
  case 84: return -1
  case 119: return -1
  case 78: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[0] = func(r rune) int {
  switch(r) {
  case 98: return 1
  case 66: return 1
  case 101: return -1
  case 84: return -1
  case 119: return -1
  case 78: return -1
  case 69: return -1
  case 116: return -1
  case 87: return -1
  case 110: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 69: return 2
  case 116: return -1
  case 87: return -1
  case 110: return -1
  case 98: return -1
  case 66: return -1
  case 101: return 2
  case 84: return -1
  case 119: return -1
  case 78: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 98: return -1
  case 66: return -1
  case 101: return -1
  case 84: return -1
  case 119: return 4
  case 78: return -1
  case 69: return -1
  case 116: return -1
  case 87: return 4
  case 110: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[4] = func(r rune) int {
  switch(r) {
  case 98: return -1
  case 66: return -1
  case 101: return 5
  case 84: return -1
  case 119: return -1
  case 78: return -1
  case 69: return 5
  case 116: return -1
  case 87: return -1
  case 110: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[6] = func(r rune) int {
  switch(r) {
  case 69: return -1
  case 116: return -1
  case 87: return -1
  case 110: return 7
  case 98: return -1
  case 66: return -1
  case 101: return -1
  case 84: return -1
  case 119: return -1
  case 78: return 7
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[43].acc = acc[:]
a0[43].f = fun[:]
a0[43].id = 43
}
{
var acc [6]bool
var fun [6]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 66: return 1
  case 114: return -1
  case 69: return -1
  case 107: return -1
  case 98: return 1
  case 82: return -1
  case 101: return -1
  case 97: return -1
  case 65: return -1
  case 75: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 98: return -1
  case 82: return 2
  case 101: return -1
  case 97: return -1
  case 65: return -1
  case 75: return -1
  case 66: return -1
  case 114: return 2
  case 69: return -1
  case 107: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 98: return -1
  case 82: return -1
  case 101: return 3
  case 97: return -1
  case 65: return -1
  case 75: return -1
  case 66: return -1
  case 114: return -1
  case 69: return 3
  case 107: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 66: return -1
  case 114: return -1
  case 69: return -1
  case 107: return -1
  case 98: return -1
  case 82: return -1
  case 101: return -1
  case 97: return 4
  case 65: return 4
  case 75: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[4] = func(r rune) int {
  switch(r) {
  case 66: return -1
  case 114: return -1
  case 69: return -1
  case 107: return 5
  case 98: return -1
  case 82: return -1
  case 101: return -1
  case 97: return -1
  case 65: return -1
  case 75: return 5
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[5] = true
fun[5] = func(r rune) int {
  switch(r) {
  case 98: return -1
  case 82: return -1
  case 101: return -1
  case 97: return -1
  case 65: return -1
  case 75: return -1
  case 66: return -1
  case 114: return -1
  case 69: return -1
  case 107: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[44].acc = acc[:]
a0[44].f = fun[:]
a0[44].id = 44
}
{
var acc [7]bool
var fun [7]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 85: return -1
  case 107: return -1
  case 75: return -1
  case 101: return -1
  case 116: return -1
  case 84: return -1
  case 98: return 1
  case 66: return 1
  case 117: return -1
  case 99: return -1
  case 67: return -1
  case 69: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 98: return -1
  case 66: return -1
  case 117: return 2
  case 99: return -1
  case 67: return -1
  case 69: return -1
  case 85: return 2
  case 107: return -1
  case 75: return -1
  case 101: return -1
  case 116: return -1
  case 84: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 98: return -1
  case 66: return -1
  case 117: return -1
  case 99: return 3
  case 67: return 3
  case 69: return -1
  case 85: return -1
  case 107: return -1
  case 75: return -1
  case 101: return -1
  case 116: return -1
  case 84: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 98: return -1
  case 66: return -1
  case 117: return -1
  case 99: return -1
  case 67: return -1
  case 69: return -1
  case 85: return -1
  case 107: return 4
  case 75: return 4
  case 101: return -1
  case 116: return -1
  case 84: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[4] = func(r rune) int {
  switch(r) {
  case 98: return -1
  case 66: return -1
  case 117: return -1
  case 99: return -1
  case 67: return -1
  case 69: return 5
  case 85: return -1
  case 107: return -1
  case 75: return -1
  case 101: return 5
  case 116: return -1
  case 84: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[5] = func(r rune) int {
  switch(r) {
  case 98: return -1
  case 66: return -1
  case 117: return -1
  case 99: return -1
  case 67: return -1
  case 69: return -1
  case 85: return -1
  case 107: return -1
  case 75: return -1
  case 101: return -1
  case 116: return 6
  case 84: return 6
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[6] = true
fun[6] = func(r rune) int {
  switch(r) {
  case 85: return -1
  case 107: return -1
  case 75: return -1
  case 101: return -1
  case 116: return -1
  case 84: return -1
  case 98: return -1
  case 66: return -1
  case 117: return -1
  case 99: return -1
  case 67: return -1
  case 69: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[45].acc = acc[:]
a0[45].f = fun[:]
a0[45].id = 45
}
{
var acc [3]bool
var fun [3]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 98: return 1
  case 66: return 1
  case 121: return -1
  case 89: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 98: return -1
  case 66: return -1
  case 121: return 2
  case 89: return 2
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[2] = true
fun[2] = func(r rune) int {
  switch(r) {
  case 98: return -1
  case 66: return -1
  case 121: return -1
  case 89: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[46].acc = acc[:]
a0[46].f = fun[:]
a0[46].id = 46
}
{
var acc [5]bool
var fun [5]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 99: return 1
  case 67: return 1
  case 97: return -1
  case 65: return -1
  case 108: return -1
  case 76: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 99: return -1
  case 67: return -1
  case 97: return 2
  case 65: return 2
  case 108: return -1
  case 76: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 99: return -1
  case 67: return -1
  case 97: return -1
  case 65: return -1
  case 108: return 3
  case 76: return 3
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 99: return -1
  case 67: return -1
  case 97: return -1
  case 65: return -1
  case 108: return 4
  case 76: return 4
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[4] = true
fun[4] = func(r rune) int {
  switch(r) {
  case 99: return -1
  case 67: return -1
  case 97: return -1
  case 65: return -1
  case 108: return -1
  case 76: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[47].acc = acc[:]
a0[47].f = fun[:]
a0[47].id = 47
}
{
var acc [5]bool
var fun [5]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 99: return 1
  case 67: return 1
  case 97: return -1
  case 65: return -1
  case 115: return -1
  case 83: return -1
  case 101: return -1
  case 69: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 99: return -1
  case 67: return -1
  case 97: return 2
  case 65: return 2
  case 115: return -1
  case 83: return -1
  case 101: return -1
  case 69: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 99: return -1
  case 67: return -1
  case 97: return -1
  case 65: return -1
  case 115: return 3
  case 83: return 3
  case 101: return -1
  case 69: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 99: return -1
  case 67: return -1
  case 97: return -1
  case 65: return -1
  case 115: return -1
  case 83: return -1
  case 101: return 4
  case 69: return 4
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[4] = true
fun[4] = func(r rune) int {
  switch(r) {
  case 99: return -1
  case 67: return -1
  case 97: return -1
  case 65: return -1
  case 115: return -1
  case 83: return -1
  case 101: return -1
  case 69: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[48].acc = acc[:]
a0[48].f = fun[:]
a0[48].id = 48
}
{
var acc [5]bool
var fun [5]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 99: return 1
  case 67: return 1
  case 97: return -1
  case 65: return -1
  case 115: return -1
  case 83: return -1
  case 116: return -1
  case 84: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 99: return -1
  case 67: return -1
  case 97: return 2
  case 65: return 2
  case 115: return -1
  case 83: return -1
  case 116: return -1
  case 84: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 99: return -1
  case 67: return -1
  case 97: return -1
  case 65: return -1
  case 115: return 3
  case 83: return 3
  case 116: return -1
  case 84: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 99: return -1
  case 67: return -1
  case 97: return -1
  case 65: return -1
  case 115: return -1
  case 83: return -1
  case 116: return 4
  case 84: return 4
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[4] = true
fun[4] = func(r rune) int {
  switch(r) {
  case 99: return -1
  case 67: return -1
  case 97: return -1
  case 65: return -1
  case 115: return -1
  case 83: return -1
  case 116: return -1
  case 84: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[49].acc = acc[:]
a0[49].f = fun[:]
a0[49].id = 49
}
{
var acc [8]bool
var fun [8]func(rune) int
fun[1] = func(r rune) int {
  switch(r) {
  case 85: return -1
  case 116: return -1
  case 84: return -1
  case 101: return -1
  case 82: return -1
  case 67: return -1
  case 76: return 2
  case 115: return -1
  case 83: return -1
  case 108: return 2
  case 69: return -1
  case 114: return -1
  case 99: return -1
  case 117: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 99: return -1
  case 117: return -1
  case 85: return -1
  case 116: return -1
  case 84: return -1
  case 101: return -1
  case 82: return -1
  case 67: return -1
  case 76: return -1
  case 115: return 4
  case 83: return 4
  case 108: return -1
  case 69: return -1
  case 114: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[6] = func(r rune) int {
  switch(r) {
  case 67: return -1
  case 76: return -1
  case 115: return -1
  case 83: return -1
  case 108: return -1
  case 69: return -1
  case 114: return 7
  case 99: return -1
  case 117: return -1
  case 85: return -1
  case 116: return -1
  case 84: return -1
  case 101: return -1
  case 82: return 7
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[7] = true
fun[7] = func(r rune) int {
  switch(r) {
  case 99: return -1
  case 117: return -1
  case 85: return -1
  case 116: return -1
  case 84: return -1
  case 101: return -1
  case 82: return -1
  case 67: return -1
  case 76: return -1
  case 115: return -1
  case 83: return -1
  case 108: return -1
  case 69: return -1
  case 114: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[0] = func(r rune) int {
  switch(r) {
  case 108: return -1
  case 69: return -1
  case 114: return -1
  case 99: return 1
  case 117: return -1
  case 85: return -1
  case 116: return -1
  case 84: return -1
  case 101: return -1
  case 82: return -1
  case 67: return 1
  case 76: return -1
  case 115: return -1
  case 83: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 67: return -1
  case 76: return -1
  case 115: return -1
  case 83: return -1
  case 108: return -1
  case 69: return -1
  case 114: return -1
  case 99: return -1
  case 117: return 3
  case 85: return 3
  case 116: return -1
  case 84: return -1
  case 101: return -1
  case 82: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[4] = func(r rune) int {
  switch(r) {
  case 108: return -1
  case 69: return -1
  case 114: return -1
  case 99: return -1
  case 117: return -1
  case 85: return -1
  case 116: return 5
  case 84: return 5
  case 101: return -1
  case 82: return -1
  case 67: return -1
  case 76: return -1
  case 115: return -1
  case 83: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[5] = func(r rune) int {
  switch(r) {
  case 85: return -1
  case 116: return -1
  case 84: return -1
  case 101: return 6
  case 82: return -1
  case 67: return -1
  case 76: return -1
  case 115: return -1
  case 83: return -1
  case 108: return -1
  case 69: return 6
  case 114: return -1
  case 99: return -1
  case 117: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[50].acc = acc[:]
a0[50].f = fun[:]
a0[50].id = 50
}
{
var acc [8]bool
var fun [8]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 99: return 1
  case 108: return -1
  case 76: return -1
  case 97: return -1
  case 65: return -1
  case 116: return -1
  case 84: return -1
  case 101: return -1
  case 69: return -1
  case 67: return 1
  case 111: return -1
  case 79: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 67: return -1
  case 111: return -1
  case 79: return -1
  case 99: return -1
  case 108: return 3
  case 76: return 3
  case 97: return -1
  case 65: return -1
  case 116: return -1
  case 84: return -1
  case 101: return -1
  case 69: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[4] = func(r rune) int {
  switch(r) {
  case 67: return -1
  case 111: return -1
  case 79: return -1
  case 99: return -1
  case 108: return -1
  case 76: return -1
  case 97: return 5
  case 65: return 5
  case 116: return -1
  case 84: return -1
  case 101: return -1
  case 69: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[6] = func(r rune) int {
  switch(r) {
  case 67: return -1
  case 111: return -1
  case 79: return -1
  case 99: return -1
  case 108: return -1
  case 76: return -1
  case 97: return -1
  case 65: return -1
  case 116: return -1
  case 84: return -1
  case 101: return 7
  case 69: return 7
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 67: return -1
  case 111: return 2
  case 79: return 2
  case 99: return -1
  case 108: return -1
  case 76: return -1
  case 97: return -1
  case 65: return -1
  case 116: return -1
  case 84: return -1
  case 101: return -1
  case 69: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 99: return -1
  case 108: return 4
  case 76: return 4
  case 97: return -1
  case 65: return -1
  case 116: return -1
  case 84: return -1
  case 101: return -1
  case 69: return -1
  case 67: return -1
  case 111: return -1
  case 79: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[5] = func(r rune) int {
  switch(r) {
  case 99: return -1
  case 108: return -1
  case 76: return -1
  case 97: return -1
  case 65: return -1
  case 116: return 6
  case 84: return 6
  case 101: return -1
  case 69: return -1
  case 67: return -1
  case 111: return -1
  case 79: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[7] = true
fun[7] = func(r rune) int {
  switch(r) {
  case 99: return -1
  case 108: return -1
  case 76: return -1
  case 97: return -1
  case 65: return -1
  case 116: return -1
  case 84: return -1
  case 101: return -1
  case 69: return -1
  case 67: return -1
  case 111: return -1
  case 79: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[51].acc = acc[:]
a0[51].f = fun[:]
a0[51].id = 51
}
{
var acc [11]bool
var fun [11]func(rune) int
fun[1] = func(r rune) int {
  switch(r) {
  case 111: return 2
  case 79: return 2
  case 108: return -1
  case 84: return -1
  case 73: return -1
  case 67: return -1
  case 116: return -1
  case 110: return -1
  case 99: return -1
  case 76: return -1
  case 101: return -1
  case 69: return -1
  case 105: return -1
  case 78: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 111: return -1
  case 79: return -1
  case 108: return 3
  case 84: return -1
  case 73: return -1
  case 67: return -1
  case 116: return -1
  case 110: return -1
  case 99: return -1
  case 76: return 3
  case 101: return -1
  case 69: return -1
  case 105: return -1
  case 78: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 69: return -1
  case 105: return -1
  case 78: return -1
  case 111: return -1
  case 79: return -1
  case 108: return 4
  case 84: return -1
  case 73: return -1
  case 67: return -1
  case 116: return -1
  case 110: return -1
  case 99: return -1
  case 76: return 4
  case 101: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[4] = func(r rune) int {
  switch(r) {
  case 99: return -1
  case 76: return -1
  case 101: return 5
  case 69: return 5
  case 105: return -1
  case 78: return -1
  case 111: return -1
  case 79: return -1
  case 108: return -1
  case 84: return -1
  case 73: return -1
  case 67: return -1
  case 116: return -1
  case 110: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[5] = func(r rune) int {
  switch(r) {
  case 69: return -1
  case 105: return -1
  case 78: return -1
  case 111: return -1
  case 79: return -1
  case 108: return -1
  case 84: return -1
  case 73: return -1
  case 67: return 6
  case 116: return -1
  case 110: return -1
  case 99: return 6
  case 76: return -1
  case 101: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[7] = func(r rune) int {
  switch(r) {
  case 111: return -1
  case 79: return -1
  case 108: return -1
  case 84: return -1
  case 73: return 8
  case 67: return -1
  case 116: return -1
  case 110: return -1
  case 99: return -1
  case 76: return -1
  case 101: return -1
  case 69: return -1
  case 105: return 8
  case 78: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[10] = true
fun[10] = func(r rune) int {
  switch(r) {
  case 99: return -1
  case 76: return -1
  case 101: return -1
  case 69: return -1
  case 105: return -1
  case 78: return -1
  case 111: return -1
  case 79: return -1
  case 108: return -1
  case 84: return -1
  case 73: return -1
  case 67: return -1
  case 116: return -1
  case 110: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[0] = func(r rune) int {
  switch(r) {
  case 99: return 1
  case 76: return -1
  case 101: return -1
  case 69: return -1
  case 105: return -1
  case 78: return -1
  case 111: return -1
  case 79: return -1
  case 108: return -1
  case 84: return -1
  case 73: return -1
  case 67: return 1
  case 116: return -1
  case 110: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[6] = func(r rune) int {
  switch(r) {
  case 99: return -1
  case 76: return -1
  case 101: return -1
  case 69: return -1
  case 105: return -1
  case 78: return -1
  case 111: return -1
  case 79: return -1
  case 108: return -1
  case 84: return 7
  case 73: return -1
  case 67: return -1
  case 116: return 7
  case 110: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[8] = func(r rune) int {
  switch(r) {
  case 111: return 9
  case 79: return 9
  case 108: return -1
  case 84: return -1
  case 73: return -1
  case 67: return -1
  case 116: return -1
  case 110: return -1
  case 99: return -1
  case 76: return -1
  case 101: return -1
  case 69: return -1
  case 105: return -1
  case 78: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[9] = func(r rune) int {
  switch(r) {
  case 69: return -1
  case 105: return -1
  case 78: return 10
  case 111: return -1
  case 79: return -1
  case 108: return -1
  case 84: return -1
  case 73: return -1
  case 67: return -1
  case 116: return -1
  case 110: return 10
  case 99: return -1
  case 76: return -1
  case 101: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[52].acc = acc[:]
a0[52].f = fun[:]
a0[52].id = 52
}
{
var acc [7]bool
var fun [7]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 111: return -1
  case 79: return -1
  case 77: return -1
  case 73: return -1
  case 116: return -1
  case 84: return -1
  case 99: return 1
  case 67: return 1
  case 109: return -1
  case 105: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 111: return 2
  case 79: return 2
  case 77: return -1
  case 73: return -1
  case 116: return -1
  case 84: return -1
  case 99: return -1
  case 67: return -1
  case 109: return -1
  case 105: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 99: return -1
  case 67: return -1
  case 109: return 3
  case 105: return -1
  case 111: return -1
  case 79: return -1
  case 77: return 3
  case 73: return -1
  case 116: return -1
  case 84: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 99: return -1
  case 67: return -1
  case 109: return 4
  case 105: return -1
  case 111: return -1
  case 79: return -1
  case 77: return 4
  case 73: return -1
  case 116: return -1
  case 84: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[4] = func(r rune) int {
  switch(r) {
  case 99: return -1
  case 67: return -1
  case 109: return -1
  case 105: return 5
  case 111: return -1
  case 79: return -1
  case 77: return -1
  case 73: return 5
  case 116: return -1
  case 84: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[5] = func(r rune) int {
  switch(r) {
  case 111: return -1
  case 79: return -1
  case 77: return -1
  case 73: return -1
  case 116: return 6
  case 84: return 6
  case 99: return -1
  case 67: return -1
  case 109: return -1
  case 105: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[6] = true
fun[6] = func(r rune) int {
  switch(r) {
  case 99: return -1
  case 67: return -1
  case 109: return -1
  case 105: return -1
  case 111: return -1
  case 79: return -1
  case 77: return -1
  case 73: return -1
  case 116: return -1
  case 84: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[53].acc = acc[:]
a0[53].f = fun[:]
a0[53].id = 53
}
{
var acc [8]bool
var fun [8]func(rune) int
fun[1] = func(r rune) int {
  switch(r) {
  case 110: return -1
  case 101: return -1
  case 69: return -1
  case 84: return -1
  case 99: return -1
  case 67: return -1
  case 111: return 2
  case 79: return 2
  case 78: return -1
  case 116: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 110: return 4
  case 101: return -1
  case 69: return -1
  case 84: return -1
  case 99: return -1
  case 67: return -1
  case 111: return -1
  case 79: return -1
  case 78: return 4
  case 116: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[5] = func(r rune) int {
  switch(r) {
  case 110: return -1
  case 101: return -1
  case 69: return -1
  case 84: return -1
  case 99: return 6
  case 67: return 6
  case 111: return -1
  case 79: return -1
  case 78: return -1
  case 116: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[7] = true
fun[7] = func(r rune) int {
  switch(r) {
  case 110: return -1
  case 101: return -1
  case 69: return -1
  case 84: return -1
  case 99: return -1
  case 67: return -1
  case 111: return -1
  case 79: return -1
  case 78: return -1
  case 116: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[0] = func(r rune) int {
  switch(r) {
  case 110: return -1
  case 101: return -1
  case 69: return -1
  case 84: return -1
  case 99: return 1
  case 67: return 1
  case 111: return -1
  case 79: return -1
  case 78: return -1
  case 116: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 99: return -1
  case 67: return -1
  case 111: return -1
  case 79: return -1
  case 78: return 3
  case 116: return -1
  case 110: return 3
  case 101: return -1
  case 69: return -1
  case 84: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[4] = func(r rune) int {
  switch(r) {
  case 99: return -1
  case 67: return -1
  case 111: return -1
  case 79: return -1
  case 78: return -1
  case 116: return -1
  case 110: return -1
  case 101: return 5
  case 69: return 5
  case 84: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[6] = func(r rune) int {
  switch(r) {
  case 110: return -1
  case 101: return -1
  case 69: return -1
  case 84: return 7
  case 99: return -1
  case 67: return -1
  case 111: return -1
  case 79: return -1
  case 78: return -1
  case 116: return 7
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[54].acc = acc[:]
a0[54].f = fun[:]
a0[54].id = 54
}
{
var acc [9]bool
var fun [9]func(rune) int
fun[1] = func(r rune) int {
  switch(r) {
  case 111: return 2
  case 116: return -1
  case 84: return -1
  case 105: return -1
  case 117: return -1
  case 101: return -1
  case 79: return 2
  case 85: return -1
  case 110: return -1
  case 69: return -1
  case 99: return -1
  case 67: return -1
  case 78: return -1
  case 73: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 79: return -1
  case 85: return -1
  case 110: return 3
  case 69: return -1
  case 99: return -1
  case 67: return -1
  case 78: return 3
  case 73: return -1
  case 111: return -1
  case 116: return -1
  case 84: return -1
  case 105: return -1
  case 117: return -1
  case 101: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 99: return -1
  case 67: return -1
  case 78: return -1
  case 73: return -1
  case 111: return -1
  case 116: return 4
  case 84: return 4
  case 105: return -1
  case 117: return -1
  case 101: return -1
  case 79: return -1
  case 85: return -1
  case 110: return -1
  case 69: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[6] = func(r rune) int {
  switch(r) {
  case 110: return -1
  case 69: return -1
  case 99: return -1
  case 67: return -1
  case 78: return -1
  case 73: return -1
  case 111: return -1
  case 116: return -1
  case 84: return -1
  case 105: return -1
  case 117: return 7
  case 101: return -1
  case 79: return -1
  case 85: return 7
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[7] = func(r rune) int {
  switch(r) {
  case 110: return -1
  case 69: return 8
  case 99: return -1
  case 67: return -1
  case 78: return -1
  case 73: return -1
  case 111: return -1
  case 116: return -1
  case 84: return -1
  case 105: return -1
  case 117: return -1
  case 101: return 8
  case 79: return -1
  case 85: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[0] = func(r rune) int {
  switch(r) {
  case 99: return 1
  case 67: return 1
  case 78: return -1
  case 73: return -1
  case 111: return -1
  case 116: return -1
  case 84: return -1
  case 105: return -1
  case 117: return -1
  case 101: return -1
  case 79: return -1
  case 85: return -1
  case 110: return -1
  case 69: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[4] = func(r rune) int {
  switch(r) {
  case 99: return -1
  case 67: return -1
  case 78: return -1
  case 73: return 5
  case 111: return -1
  case 116: return -1
  case 84: return -1
  case 105: return 5
  case 117: return -1
  case 101: return -1
  case 79: return -1
  case 85: return -1
  case 110: return -1
  case 69: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[5] = func(r rune) int {
  switch(r) {
  case 111: return -1
  case 116: return -1
  case 84: return -1
  case 105: return -1
  case 117: return -1
  case 101: return -1
  case 79: return -1
  case 85: return -1
  case 110: return 6
  case 69: return -1
  case 99: return -1
  case 67: return -1
  case 78: return 6
  case 73: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[8] = true
fun[8] = func(r rune) int {
  switch(r) {
  case 79: return -1
  case 85: return -1
  case 110: return -1
  case 69: return -1
  case 99: return -1
  case 67: return -1
  case 78: return -1
  case 73: return -1
  case 111: return -1
  case 116: return -1
  case 84: return -1
  case 105: return -1
  case 117: return -1
  case 101: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[55].acc = acc[:]
a0[55].f = fun[:]
a0[55].id = 55
}
{
var acc [7]bool
var fun [7]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 99: return 1
  case 114: return -1
  case 82: return -1
  case 101: return -1
  case 69: return -1
  case 97: return -1
  case 65: return -1
  case 84: return -1
  case 67: return 1
  case 116: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 99: return -1
  case 114: return 2
  case 82: return 2
  case 101: return -1
  case 69: return -1
  case 97: return -1
  case 65: return -1
  case 84: return -1
  case 67: return -1
  case 116: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 99: return -1
  case 114: return -1
  case 82: return -1
  case 101: return 3
  case 69: return 3
  case 97: return -1
  case 65: return -1
  case 84: return -1
  case 67: return -1
  case 116: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 99: return -1
  case 114: return -1
  case 82: return -1
  case 101: return -1
  case 69: return -1
  case 97: return 4
  case 65: return 4
  case 84: return -1
  case 67: return -1
  case 116: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[4] = func(r rune) int {
  switch(r) {
  case 99: return -1
  case 114: return -1
  case 82: return -1
  case 101: return -1
  case 69: return -1
  case 97: return -1
  case 65: return -1
  case 84: return 5
  case 67: return -1
  case 116: return 5
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[5] = func(r rune) int {
  switch(r) {
  case 67: return -1
  case 116: return -1
  case 99: return -1
  case 114: return -1
  case 82: return -1
  case 101: return 6
  case 69: return 6
  case 97: return -1
  case 65: return -1
  case 84: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[6] = true
fun[6] = func(r rune) int {
  switch(r) {
  case 67: return -1
  case 116: return -1
  case 99: return -1
  case 114: return -1
  case 82: return -1
  case 101: return -1
  case 69: return -1
  case 97: return -1
  case 65: return -1
  case 84: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[56].acc = acc[:]
a0[56].f = fun[:]
a0[56].id = 56
}
{
var acc [9]bool
var fun [9]func(rune) int
fun[2] = func(r rune) int {
  switch(r) {
  case 100: return -1
  case 68: return -1
  case 97: return -1
  case 65: return -1
  case 66: return -1
  case 101: return -1
  case 69: return -1
  case 116: return 3
  case 84: return 3
  case 98: return -1
  case 115: return -1
  case 83: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 100: return -1
  case 68: return -1
  case 97: return 4
  case 65: return 4
  case 66: return -1
  case 101: return -1
  case 69: return -1
  case 116: return -1
  case 84: return -1
  case 98: return -1
  case 115: return -1
  case 83: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[5] = func(r rune) int {
  switch(r) {
  case 116: return -1
  case 84: return -1
  case 98: return -1
  case 115: return -1
  case 83: return -1
  case 100: return -1
  case 68: return -1
  case 97: return 6
  case 65: return 6
  case 66: return -1
  case 101: return -1
  case 69: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[0] = func(r rune) int {
  switch(r) {
  case 116: return -1
  case 84: return -1
  case 98: return -1
  case 115: return -1
  case 83: return -1
  case 100: return 1
  case 68: return 1
  case 97: return -1
  case 65: return -1
  case 66: return -1
  case 101: return -1
  case 69: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 100: return -1
  case 68: return -1
  case 97: return 2
  case 65: return 2
  case 66: return -1
  case 101: return -1
  case 69: return -1
  case 116: return -1
  case 84: return -1
  case 98: return -1
  case 115: return -1
  case 83: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[4] = func(r rune) int {
  switch(r) {
  case 116: return -1
  case 84: return -1
  case 98: return 5
  case 115: return -1
  case 83: return -1
  case 100: return -1
  case 68: return -1
  case 97: return -1
  case 65: return -1
  case 66: return 5
  case 101: return -1
  case 69: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[6] = func(r rune) int {
  switch(r) {
  case 100: return -1
  case 68: return -1
  case 97: return -1
  case 65: return -1
  case 66: return -1
  case 101: return -1
  case 69: return -1
  case 116: return -1
  case 84: return -1
  case 98: return -1
  case 115: return 7
  case 83: return 7
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[7] = func(r rune) int {
  switch(r) {
  case 100: return -1
  case 68: return -1
  case 97: return -1
  case 65: return -1
  case 66: return -1
  case 101: return 8
  case 69: return 8
  case 116: return -1
  case 84: return -1
  case 98: return -1
  case 115: return -1
  case 83: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[8] = true
fun[8] = func(r rune) int {
  switch(r) {
  case 116: return -1
  case 84: return -1
  case 98: return -1
  case 115: return -1
  case 83: return -1
  case 100: return -1
  case 68: return -1
  case 97: return -1
  case 65: return -1
  case 66: return -1
  case 101: return -1
  case 69: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[57].acc = acc[:]
a0[57].f = fun[:]
a0[57].id = 57
}
{
var acc [8]bool
var fun [8]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 100: return 1
  case 65: return -1
  case 83: return -1
  case 68: return 1
  case 97: return -1
  case 116: return -1
  case 84: return -1
  case 115: return -1
  case 101: return -1
  case 69: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 68: return -1
  case 97: return 4
  case 116: return -1
  case 84: return -1
  case 115: return -1
  case 101: return -1
  case 69: return -1
  case 100: return -1
  case 65: return 4
  case 83: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[4] = func(r rune) int {
  switch(r) {
  case 68: return -1
  case 97: return -1
  case 116: return -1
  case 84: return -1
  case 115: return 5
  case 101: return -1
  case 69: return -1
  case 100: return -1
  case 65: return -1
  case 83: return 5
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[6] = func(r rune) int {
  switch(r) {
  case 68: return -1
  case 97: return -1
  case 116: return 7
  case 84: return 7
  case 115: return -1
  case 101: return -1
  case 69: return -1
  case 100: return -1
  case 65: return -1
  case 83: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[7] = true
fun[7] = func(r rune) int {
  switch(r) {
  case 68: return -1
  case 97: return -1
  case 116: return -1
  case 84: return -1
  case 115: return -1
  case 101: return -1
  case 69: return -1
  case 100: return -1
  case 65: return -1
  case 83: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 100: return -1
  case 65: return 2
  case 83: return -1
  case 68: return -1
  case 97: return 2
  case 116: return -1
  case 84: return -1
  case 115: return -1
  case 101: return -1
  case 69: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 68: return -1
  case 97: return -1
  case 116: return 3
  case 84: return 3
  case 115: return -1
  case 101: return -1
  case 69: return -1
  case 100: return -1
  case 65: return -1
  case 83: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[5] = func(r rune) int {
  switch(r) {
  case 100: return -1
  case 65: return -1
  case 83: return -1
  case 68: return -1
  case 97: return -1
  case 116: return -1
  case 84: return -1
  case 115: return -1
  case 101: return 6
  case 69: return 6
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[58].acc = acc[:]
a0[58].f = fun[:]
a0[58].id = 58
}
{
var acc [10]bool
var fun [10]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 84: return -1
  case 111: return -1
  case 79: return -1
  case 69: return -1
  case 100: return 1
  case 97: return -1
  case 65: return -1
  case 115: return -1
  case 83: return -1
  case 114: return -1
  case 116: return -1
  case 82: return -1
  case 68: return 1
  case 101: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 84: return -1
  case 111: return -1
  case 79: return -1
  case 69: return -1
  case 100: return -1
  case 97: return 2
  case 65: return 2
  case 115: return -1
  case 83: return -1
  case 114: return -1
  case 116: return -1
  case 82: return -1
  case 68: return -1
  case 101: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 84: return 3
  case 111: return -1
  case 79: return -1
  case 69: return -1
  case 100: return -1
  case 97: return -1
  case 65: return -1
  case 115: return -1
  case 83: return -1
  case 114: return -1
  case 116: return 3
  case 82: return -1
  case 68: return -1
  case 101: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[6] = func(r rune) int {
  switch(r) {
  case 68: return -1
  case 101: return -1
  case 84: return -1
  case 111: return 7
  case 79: return 7
  case 69: return -1
  case 100: return -1
  case 97: return -1
  case 65: return -1
  case 115: return -1
  case 83: return -1
  case 114: return -1
  case 116: return -1
  case 82: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 84: return -1
  case 111: return -1
  case 79: return -1
  case 69: return -1
  case 100: return -1
  case 97: return 4
  case 65: return 4
  case 115: return -1
  case 83: return -1
  case 114: return -1
  case 116: return -1
  case 82: return -1
  case 68: return -1
  case 101: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[4] = func(r rune) int {
  switch(r) {
  case 84: return -1
  case 111: return -1
  case 79: return -1
  case 69: return -1
  case 100: return -1
  case 97: return -1
  case 65: return -1
  case 115: return 5
  case 83: return 5
  case 114: return -1
  case 116: return -1
  case 82: return -1
  case 68: return -1
  case 101: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[5] = func(r rune) int {
  switch(r) {
  case 84: return 6
  case 111: return -1
  case 79: return -1
  case 69: return -1
  case 100: return -1
  case 97: return -1
  case 65: return -1
  case 115: return -1
  case 83: return -1
  case 114: return -1
  case 116: return 6
  case 82: return -1
  case 68: return -1
  case 101: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[7] = func(r rune) int {
  switch(r) {
  case 116: return -1
  case 82: return 8
  case 68: return -1
  case 101: return -1
  case 84: return -1
  case 111: return -1
  case 79: return -1
  case 69: return -1
  case 100: return -1
  case 97: return -1
  case 65: return -1
  case 115: return -1
  case 83: return -1
  case 114: return 8
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[8] = func(r rune) int {
  switch(r) {
  case 84: return -1
  case 111: return -1
  case 79: return -1
  case 69: return 9
  case 100: return -1
  case 97: return -1
  case 65: return -1
  case 115: return -1
  case 83: return -1
  case 114: return -1
  case 116: return -1
  case 82: return -1
  case 68: return -1
  case 101: return 9
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[9] = true
fun[9] = func(r rune) int {
  switch(r) {
  case 84: return -1
  case 111: return -1
  case 79: return -1
  case 69: return -1
  case 100: return -1
  case 97: return -1
  case 65: return -1
  case 115: return -1
  case 83: return -1
  case 114: return -1
  case 116: return -1
  case 82: return -1
  case 68: return -1
  case 101: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[59].acc = acc[:]
a0[59].f = fun[:]
a0[59].id = 59
}
{
var acc [8]bool
var fun [8]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 100: return 1
  case 68: return 1
  case 101: return -1
  case 69: return -1
  case 108: return -1
  case 97: return -1
  case 65: return -1
  case 99: return -1
  case 67: return -1
  case 76: return -1
  case 114: return -1
  case 82: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 99: return 3
  case 67: return 3
  case 76: return -1
  case 114: return -1
  case 82: return -1
  case 100: return -1
  case 68: return -1
  case 101: return -1
  case 69: return -1
  case 108: return -1
  case 97: return -1
  case 65: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 100: return -1
  case 68: return -1
  case 101: return -1
  case 69: return -1
  case 108: return 4
  case 97: return -1
  case 65: return -1
  case 99: return -1
  case 67: return -1
  case 76: return 4
  case 114: return -1
  case 82: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[5] = func(r rune) int {
  switch(r) {
  case 99: return -1
  case 67: return -1
  case 76: return -1
  case 114: return 6
  case 82: return 6
  case 100: return -1
  case 68: return -1
  case 101: return -1
  case 69: return -1
  case 108: return -1
  case 97: return -1
  case 65: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[6] = func(r rune) int {
  switch(r) {
  case 100: return -1
  case 68: return -1
  case 101: return 7
  case 69: return 7
  case 108: return -1
  case 97: return -1
  case 65: return -1
  case 99: return -1
  case 67: return -1
  case 76: return -1
  case 114: return -1
  case 82: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 100: return -1
  case 68: return -1
  case 101: return 2
  case 69: return 2
  case 108: return -1
  case 97: return -1
  case 65: return -1
  case 99: return -1
  case 67: return -1
  case 76: return -1
  case 114: return -1
  case 82: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[4] = func(r rune) int {
  switch(r) {
  case 100: return -1
  case 68: return -1
  case 101: return -1
  case 69: return -1
  case 108: return -1
  case 97: return 5
  case 65: return 5
  case 99: return -1
  case 67: return -1
  case 76: return -1
  case 114: return -1
  case 82: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[7] = true
fun[7] = func(r rune) int {
  switch(r) {
  case 99: return -1
  case 67: return -1
  case 76: return -1
  case 114: return -1
  case 82: return -1
  case 100: return -1
  case 68: return -1
  case 101: return -1
  case 69: return -1
  case 108: return -1
  case 97: return -1
  case 65: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[60].acc = acc[:]
a0[60].f = fun[:]
a0[60].id = 60
}
{
var acc [7]bool
var fun [7]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 100: return 1
  case 68: return 1
  case 101: return -1
  case 69: return -1
  case 108: return -1
  case 76: return -1
  case 116: return -1
  case 84: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 100: return -1
  case 68: return -1
  case 101: return 2
  case 69: return 2
  case 108: return -1
  case 76: return -1
  case 116: return -1
  case 84: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 100: return -1
  case 68: return -1
  case 101: return -1
  case 69: return -1
  case 108: return 3
  case 76: return 3
  case 116: return -1
  case 84: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 100: return -1
  case 68: return -1
  case 101: return 4
  case 69: return 4
  case 108: return -1
  case 76: return -1
  case 116: return -1
  case 84: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[4] = func(r rune) int {
  switch(r) {
  case 100: return -1
  case 68: return -1
  case 101: return -1
  case 69: return -1
  case 108: return -1
  case 76: return -1
  case 116: return 5
  case 84: return 5
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[5] = func(r rune) int {
  switch(r) {
  case 100: return -1
  case 68: return -1
  case 101: return 6
  case 69: return 6
  case 108: return -1
  case 76: return -1
  case 116: return -1
  case 84: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[6] = true
fun[6] = func(r rune) int {
  switch(r) {
  case 100: return -1
  case 68: return -1
  case 101: return -1
  case 69: return -1
  case 108: return -1
  case 76: return -1
  case 116: return -1
  case 84: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[61].acc = acc[:]
a0[61].f = fun[:]
a0[61].id = 61
}
{
var acc [8]bool
var fun [8]func(rune) int
fun[2] = func(r rune) int {
  switch(r) {
  case 100: return -1
  case 68: return -1
  case 114: return 3
  case 82: return 3
  case 118: return -1
  case 101: return -1
  case 69: return -1
  case 105: return -1
  case 73: return -1
  case 86: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[4] = func(r rune) int {
  switch(r) {
  case 100: return -1
  case 68: return -1
  case 114: return -1
  case 82: return -1
  case 118: return 5
  case 101: return -1
  case 69: return -1
  case 105: return -1
  case 73: return -1
  case 86: return 5
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[7] = true
fun[7] = func(r rune) int {
  switch(r) {
  case 101: return -1
  case 69: return -1
  case 105: return -1
  case 73: return -1
  case 86: return -1
  case 100: return -1
  case 68: return -1
  case 114: return -1
  case 82: return -1
  case 118: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[0] = func(r rune) int {
  switch(r) {
  case 100: return 1
  case 68: return 1
  case 114: return -1
  case 82: return -1
  case 118: return -1
  case 101: return -1
  case 69: return -1
  case 105: return -1
  case 73: return -1
  case 86: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 101: return 2
  case 69: return 2
  case 105: return -1
  case 73: return -1
  case 86: return -1
  case 100: return -1
  case 68: return -1
  case 114: return -1
  case 82: return -1
  case 118: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 101: return -1
  case 69: return -1
  case 105: return 4
  case 73: return 4
  case 86: return -1
  case 100: return -1
  case 68: return -1
  case 114: return -1
  case 82: return -1
  case 118: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[5] = func(r rune) int {
  switch(r) {
  case 101: return 6
  case 69: return 6
  case 105: return -1
  case 73: return -1
  case 86: return -1
  case 100: return -1
  case 68: return -1
  case 114: return -1
  case 82: return -1
  case 118: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[6] = func(r rune) int {
  switch(r) {
  case 101: return -1
  case 69: return -1
  case 105: return -1
  case 73: return -1
  case 86: return -1
  case 100: return 7
  case 68: return 7
  case 114: return -1
  case 82: return -1
  case 118: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[62].acc = acc[:]
a0[62].f = fun[:]
a0[62].id = 62
}
{
var acc [5]bool
var fun [5]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 100: return 1
  case 68: return 1
  case 101: return -1
  case 69: return -1
  case 115: return -1
  case 83: return -1
  case 99: return -1
  case 67: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 100: return -1
  case 68: return -1
  case 101: return 2
  case 69: return 2
  case 115: return -1
  case 83: return -1
  case 99: return -1
  case 67: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 100: return -1
  case 68: return -1
  case 101: return -1
  case 69: return -1
  case 115: return 3
  case 83: return 3
  case 99: return -1
  case 67: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 100: return -1
  case 68: return -1
  case 101: return -1
  case 69: return -1
  case 115: return -1
  case 83: return -1
  case 99: return 4
  case 67: return 4
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[4] = true
fun[4] = func(r rune) int {
  switch(r) {
  case 100: return -1
  case 68: return -1
  case 101: return -1
  case 69: return -1
  case 115: return -1
  case 83: return -1
  case 99: return -1
  case 67: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[63].acc = acc[:]
a0[63].f = fun[:]
a0[63].id = 63
}
{
var acc [9]bool
var fun [9]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 69: return -1
  case 115: return -1
  case 67: return -1
  case 82: return -1
  case 73: return -1
  case 100: return 1
  case 68: return 1
  case 83: return -1
  case 99: return -1
  case 114: return -1
  case 105: return -1
  case 66: return -1
  case 101: return -1
  case 98: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 100: return -1
  case 68: return -1
  case 83: return -1
  case 99: return -1
  case 114: return -1
  case 105: return -1
  case 66: return -1
  case 101: return 2
  case 98: return -1
  case 69: return 2
  case 115: return -1
  case 67: return -1
  case 82: return -1
  case 73: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 101: return -1
  case 98: return -1
  case 69: return -1
  case 115: return 3
  case 67: return -1
  case 82: return -1
  case 73: return -1
  case 100: return -1
  case 68: return -1
  case 83: return 3
  case 99: return -1
  case 114: return -1
  case 105: return -1
  case 66: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 101: return -1
  case 98: return -1
  case 69: return -1
  case 115: return -1
  case 67: return 4
  case 82: return -1
  case 73: return -1
  case 100: return -1
  case 68: return -1
  case 83: return -1
  case 99: return 4
  case 114: return -1
  case 105: return -1
  case 66: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[5] = func(r rune) int {
  switch(r) {
  case 101: return -1
  case 98: return -1
  case 69: return -1
  case 115: return -1
  case 67: return -1
  case 82: return -1
  case 73: return 6
  case 100: return -1
  case 68: return -1
  case 83: return -1
  case 99: return -1
  case 114: return -1
  case 105: return 6
  case 66: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[8] = true
fun[8] = func(r rune) int {
  switch(r) {
  case 100: return -1
  case 68: return -1
  case 83: return -1
  case 99: return -1
  case 114: return -1
  case 105: return -1
  case 66: return -1
  case 101: return -1
  case 98: return -1
  case 69: return -1
  case 115: return -1
  case 67: return -1
  case 82: return -1
  case 73: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[4] = func(r rune) int {
  switch(r) {
  case 101: return -1
  case 98: return -1
  case 69: return -1
  case 115: return -1
  case 67: return -1
  case 82: return 5
  case 73: return -1
  case 100: return -1
  case 68: return -1
  case 83: return -1
  case 99: return -1
  case 114: return 5
  case 105: return -1
  case 66: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[6] = func(r rune) int {
  switch(r) {
  case 101: return -1
  case 98: return 7
  case 69: return -1
  case 115: return -1
  case 67: return -1
  case 82: return -1
  case 73: return -1
  case 100: return -1
  case 68: return -1
  case 83: return -1
  case 99: return -1
  case 114: return -1
  case 105: return -1
  case 66: return 7
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[7] = func(r rune) int {
  switch(r) {
  case 69: return 8
  case 115: return -1
  case 67: return -1
  case 82: return -1
  case 73: return -1
  case 100: return -1
  case 68: return -1
  case 83: return -1
  case 99: return -1
  case 114: return -1
  case 105: return -1
  case 66: return -1
  case 101: return 8
  case 98: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[64].acc = acc[:]
a0[64].f = fun[:]
a0[64].id = 64
}
{
var acc [9]bool
var fun [9]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 100: return 1
  case 116: return -1
  case 84: return -1
  case 110: return -1
  case 78: return -1
  case 67: return -1
  case 68: return 1
  case 105: return -1
  case 73: return -1
  case 115: return -1
  case 83: return -1
  case 99: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 68: return -1
  case 105: return 2
  case 73: return 2
  case 115: return -1
  case 83: return -1
  case 99: return -1
  case 100: return -1
  case 116: return -1
  case 84: return -1
  case 110: return -1
  case 78: return -1
  case 67: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 100: return -1
  case 116: return -1
  case 84: return -1
  case 110: return -1
  case 78: return -1
  case 67: return -1
  case 68: return -1
  case 105: return -1
  case 73: return -1
  case 115: return 3
  case 83: return 3
  case 99: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 100: return -1
  case 116: return 4
  case 84: return 4
  case 110: return -1
  case 78: return -1
  case 67: return -1
  case 68: return -1
  case 105: return -1
  case 73: return -1
  case 115: return -1
  case 83: return -1
  case 99: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[6] = func(r rune) int {
  switch(r) {
  case 100: return -1
  case 116: return -1
  case 84: return -1
  case 110: return -1
  case 78: return -1
  case 67: return 7
  case 68: return -1
  case 105: return -1
  case 73: return -1
  case 115: return -1
  case 83: return -1
  case 99: return 7
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[7] = func(r rune) int {
  switch(r) {
  case 68: return -1
  case 105: return -1
  case 73: return -1
  case 115: return -1
  case 83: return -1
  case 99: return -1
  case 100: return -1
  case 116: return 8
  case 84: return 8
  case 110: return -1
  case 78: return -1
  case 67: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[8] = true
fun[8] = func(r rune) int {
  switch(r) {
  case 100: return -1
  case 116: return -1
  case 84: return -1
  case 110: return -1
  case 78: return -1
  case 67: return -1
  case 68: return -1
  case 105: return -1
  case 73: return -1
  case 115: return -1
  case 83: return -1
  case 99: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[4] = func(r rune) int {
  switch(r) {
  case 100: return -1
  case 116: return -1
  case 84: return -1
  case 110: return -1
  case 78: return -1
  case 67: return -1
  case 68: return -1
  case 105: return 5
  case 73: return 5
  case 115: return -1
  case 83: return -1
  case 99: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[5] = func(r rune) int {
  switch(r) {
  case 100: return -1
  case 116: return -1
  case 84: return -1
  case 110: return 6
  case 78: return 6
  case 67: return -1
  case 68: return -1
  case 105: return -1
  case 73: return -1
  case 115: return -1
  case 83: return -1
  case 99: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[65].acc = acc[:]
a0[65].f = fun[:]
a0[65].id = 65
}
{
var acc [3]bool
var fun [3]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 100: return 1
  case 68: return 1
  case 111: return -1
  case 79: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 100: return -1
  case 68: return -1
  case 111: return 2
  case 79: return 2
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[2] = true
fun[2] = func(r rune) int {
  switch(r) {
  case 100: return -1
  case 68: return -1
  case 111: return -1
  case 79: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[66].acc = acc[:]
a0[66].f = fun[:]
a0[66].id = 66
}
{
var acc [5]bool
var fun [5]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 100: return 1
  case 68: return 1
  case 114: return -1
  case 82: return -1
  case 111: return -1
  case 79: return -1
  case 112: return -1
  case 80: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 100: return -1
  case 68: return -1
  case 114: return 2
  case 82: return 2
  case 111: return -1
  case 79: return -1
  case 112: return -1
  case 80: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 100: return -1
  case 68: return -1
  case 114: return -1
  case 82: return -1
  case 111: return 3
  case 79: return 3
  case 112: return -1
  case 80: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 100: return -1
  case 68: return -1
  case 114: return -1
  case 82: return -1
  case 111: return -1
  case 79: return -1
  case 112: return 4
  case 80: return 4
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[4] = true
fun[4] = func(r rune) int {
  switch(r) {
  case 100: return -1
  case 68: return -1
  case 114: return -1
  case 82: return -1
  case 111: return -1
  case 79: return -1
  case 112: return -1
  case 80: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[67].acc = acc[:]
a0[67].f = fun[:]
a0[67].id = 67
}
{
var acc [5]bool
var fun [5]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 101: return 1
  case 69: return 1
  case 97: return -1
  case 65: return -1
  case 99: return -1
  case 67: return -1
  case 104: return -1
  case 72: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 101: return -1
  case 69: return -1
  case 97: return 2
  case 65: return 2
  case 99: return -1
  case 67: return -1
  case 104: return -1
  case 72: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 101: return -1
  case 69: return -1
  case 97: return -1
  case 65: return -1
  case 99: return 3
  case 67: return 3
  case 104: return -1
  case 72: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 101: return -1
  case 69: return -1
  case 97: return -1
  case 65: return -1
  case 99: return -1
  case 67: return -1
  case 104: return 4
  case 72: return 4
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[4] = true
fun[4] = func(r rune) int {
  switch(r) {
  case 101: return -1
  case 69: return -1
  case 97: return -1
  case 65: return -1
  case 99: return -1
  case 67: return -1
  case 104: return -1
  case 72: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[68].acc = acc[:]
a0[68].f = fun[:]
a0[68].id = 68
}
{
var acc [8]bool
var fun [8]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 108: return -1
  case 76: return -1
  case 109: return -1
  case 78: return -1
  case 101: return 1
  case 69: return 1
  case 77: return -1
  case 110: return -1
  case 116: return -1
  case 84: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 101: return -1
  case 69: return -1
  case 77: return 4
  case 110: return -1
  case 116: return -1
  case 84: return -1
  case 108: return -1
  case 76: return -1
  case 109: return 4
  case 78: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[5] = func(r rune) int {
  switch(r) {
  case 101: return -1
  case 69: return -1
  case 77: return -1
  case 110: return 6
  case 116: return -1
  case 84: return -1
  case 108: return -1
  case 76: return -1
  case 109: return -1
  case 78: return 6
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 101: return -1
  case 69: return -1
  case 77: return -1
  case 110: return -1
  case 116: return -1
  case 84: return -1
  case 108: return 2
  case 76: return 2
  case 109: return -1
  case 78: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 108: return -1
  case 76: return -1
  case 109: return -1
  case 78: return -1
  case 101: return 3
  case 69: return 3
  case 77: return -1
  case 110: return -1
  case 116: return -1
  case 84: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[4] = func(r rune) int {
  switch(r) {
  case 108: return -1
  case 76: return -1
  case 109: return -1
  case 78: return -1
  case 101: return 5
  case 69: return 5
  case 77: return -1
  case 110: return -1
  case 116: return -1
  case 84: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[6] = func(r rune) int {
  switch(r) {
  case 101: return -1
  case 69: return -1
  case 77: return -1
  case 110: return -1
  case 116: return 7
  case 84: return 7
  case 108: return -1
  case 76: return -1
  case 109: return -1
  case 78: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[7] = true
fun[7] = func(r rune) int {
  switch(r) {
  case 101: return -1
  case 69: return -1
  case 77: return -1
  case 110: return -1
  case 116: return -1
  case 84: return -1
  case 108: return -1
  case 76: return -1
  case 109: return -1
  case 78: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[69].acc = acc[:]
a0[69].f = fun[:]
a0[69].id = 69
}
{
var acc [5]bool
var fun [5]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 101: return 1
  case 69: return 1
  case 108: return -1
  case 76: return -1
  case 115: return -1
  case 83: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 101: return -1
  case 69: return -1
  case 108: return 2
  case 76: return 2
  case 115: return -1
  case 83: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 101: return -1
  case 69: return -1
  case 108: return -1
  case 76: return -1
  case 115: return 3
  case 83: return 3
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 101: return 4
  case 69: return 4
  case 108: return -1
  case 76: return -1
  case 115: return -1
  case 83: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[4] = true
fun[4] = func(r rune) int {
  switch(r) {
  case 101: return -1
  case 69: return -1
  case 108: return -1
  case 76: return -1
  case 115: return -1
  case 83: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[70].acc = acc[:]
a0[70].f = fun[:]
a0[70].id = 70
}
{
var acc [4]bool
var fun [4]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 101: return 1
  case 69: return 1
  case 110: return -1
  case 78: return -1
  case 100: return -1
  case 68: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 101: return -1
  case 69: return -1
  case 110: return 2
  case 78: return 2
  case 100: return -1
  case 68: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 101: return -1
  case 69: return -1
  case 110: return -1
  case 78: return -1
  case 100: return 3
  case 68: return 3
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[3] = true
fun[3] = func(r rune) int {
  switch(r) {
  case 101: return -1
  case 69: return -1
  case 110: return -1
  case 78: return -1
  case 100: return -1
  case 68: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[71].acc = acc[:]
a0[71].f = fun[:]
a0[71].id = 71
}
{
var acc [6]bool
var fun [6]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 101: return 1
  case 69: return 1
  case 118: return -1
  case 86: return -1
  case 114: return -1
  case 82: return -1
  case 121: return -1
  case 89: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 101: return -1
  case 69: return -1
  case 118: return 2
  case 86: return 2
  case 114: return -1
  case 82: return -1
  case 121: return -1
  case 89: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 101: return 3
  case 69: return 3
  case 118: return -1
  case 86: return -1
  case 114: return -1
  case 82: return -1
  case 121: return -1
  case 89: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 101: return -1
  case 69: return -1
  case 118: return -1
  case 86: return -1
  case 114: return 4
  case 82: return 4
  case 121: return -1
  case 89: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[4] = func(r rune) int {
  switch(r) {
  case 101: return -1
  case 69: return -1
  case 118: return -1
  case 86: return -1
  case 114: return -1
  case 82: return -1
  case 121: return 5
  case 89: return 5
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[5] = true
fun[5] = func(r rune) int {
  switch(r) {
  case 101: return -1
  case 69: return -1
  case 118: return -1
  case 86: return -1
  case 114: return -1
  case 82: return -1
  case 121: return -1
  case 89: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[72].acc = acc[:]
a0[72].f = fun[:]
a0[72].id = 72
}
{
var acc [7]bool
var fun [7]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 120: return -1
  case 99: return -1
  case 112: return -1
  case 80: return -1
  case 101: return 1
  case 69: return 1
  case 88: return -1
  case 67: return -1
  case 116: return -1
  case 84: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 101: return -1
  case 69: return -1
  case 88: return 2
  case 67: return -1
  case 116: return -1
  case 84: return -1
  case 120: return 2
  case 99: return -1
  case 112: return -1
  case 80: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 101: return -1
  case 69: return -1
  case 88: return -1
  case 67: return 3
  case 116: return -1
  case 84: return -1
  case 120: return -1
  case 99: return 3
  case 112: return -1
  case 80: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 120: return -1
  case 99: return -1
  case 112: return -1
  case 80: return -1
  case 101: return 4
  case 69: return 4
  case 88: return -1
  case 67: return -1
  case 116: return -1
  case 84: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[4] = func(r rune) int {
  switch(r) {
  case 101: return -1
  case 69: return -1
  case 88: return -1
  case 67: return -1
  case 116: return -1
  case 84: return -1
  case 120: return -1
  case 99: return -1
  case 112: return 5
  case 80: return 5
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[5] = func(r rune) int {
  switch(r) {
  case 120: return -1
  case 99: return -1
  case 112: return -1
  case 80: return -1
  case 101: return -1
  case 69: return -1
  case 88: return -1
  case 67: return -1
  case 116: return 6
  case 84: return 6
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[6] = true
fun[6] = func(r rune) int {
  switch(r) {
  case 120: return -1
  case 99: return -1
  case 112: return -1
  case 80: return -1
  case 101: return -1
  case 69: return -1
  case 88: return -1
  case 67: return -1
  case 116: return -1
  case 84: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[73].acc = acc[:]
a0[73].f = fun[:]
a0[73].id = 73
}
{
var acc [8]bool
var fun [8]func(rune) int
fun[1] = func(r rune) int {
  switch(r) {
  case 101: return -1
  case 69: return -1
  case 67: return -1
  case 76: return -1
  case 120: return 2
  case 88: return 2
  case 99: return -1
  case 108: return -1
  case 117: return -1
  case 85: return -1
  case 100: return -1
  case 68: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 101: return -1
  case 69: return -1
  case 67: return -1
  case 76: return 4
  case 120: return -1
  case 88: return -1
  case 99: return -1
  case 108: return 4
  case 117: return -1
  case 85: return -1
  case 100: return -1
  case 68: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[4] = func(r rune) int {
  switch(r) {
  case 120: return -1
  case 88: return -1
  case 99: return -1
  case 108: return -1
  case 117: return 5
  case 85: return 5
  case 100: return -1
  case 68: return -1
  case 101: return -1
  case 69: return -1
  case 67: return -1
  case 76: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[5] = func(r rune) int {
  switch(r) {
  case 120: return -1
  case 88: return -1
  case 99: return -1
  case 108: return -1
  case 117: return -1
  case 85: return -1
  case 100: return 6
  case 68: return 6
  case 101: return -1
  case 69: return -1
  case 67: return -1
  case 76: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[6] = func(r rune) int {
  switch(r) {
  case 120: return -1
  case 88: return -1
  case 99: return -1
  case 108: return -1
  case 117: return -1
  case 85: return -1
  case 100: return -1
  case 68: return -1
  case 101: return 7
  case 69: return 7
  case 67: return -1
  case 76: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[0] = func(r rune) int {
  switch(r) {
  case 120: return -1
  case 88: return -1
  case 99: return -1
  case 108: return -1
  case 117: return -1
  case 85: return -1
  case 100: return -1
  case 68: return -1
  case 101: return 1
  case 69: return 1
  case 67: return -1
  case 76: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 120: return -1
  case 88: return -1
  case 99: return 3
  case 108: return -1
  case 117: return -1
  case 85: return -1
  case 100: return -1
  case 68: return -1
  case 101: return -1
  case 69: return -1
  case 67: return 3
  case 76: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[7] = true
fun[7] = func(r rune) int {
  switch(r) {
  case 101: return -1
  case 69: return -1
  case 67: return -1
  case 76: return -1
  case 120: return -1
  case 88: return -1
  case 99: return -1
  case 108: return -1
  case 117: return -1
  case 85: return -1
  case 100: return -1
  case 68: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[74].acc = acc[:]
a0[74].f = fun[:]
a0[74].id = 74
}
{
var acc [8]bool
var fun [8]func(rune) int
fun[2] = func(r rune) int {
  switch(r) {
  case 120: return -1
  case 88: return -1
  case 117: return -1
  case 116: return -1
  case 101: return 3
  case 69: return 3
  case 99: return -1
  case 67: return -1
  case 85: return -1
  case 84: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 101: return -1
  case 69: return -1
  case 99: return 4
  case 67: return 4
  case 85: return -1
  case 84: return -1
  case 120: return -1
  case 88: return -1
  case 117: return -1
  case 116: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[6] = func(r rune) int {
  switch(r) {
  case 120: return -1
  case 88: return -1
  case 117: return -1
  case 116: return -1
  case 101: return 7
  case 69: return 7
  case 99: return -1
  case 67: return -1
  case 85: return -1
  case 84: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[7] = true
fun[7] = func(r rune) int {
  switch(r) {
  case 120: return -1
  case 88: return -1
  case 117: return -1
  case 116: return -1
  case 101: return -1
  case 69: return -1
  case 99: return -1
  case 67: return -1
  case 85: return -1
  case 84: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[0] = func(r rune) int {
  switch(r) {
  case 120: return -1
  case 88: return -1
  case 117: return -1
  case 116: return -1
  case 101: return 1
  case 69: return 1
  case 99: return -1
  case 67: return -1
  case 85: return -1
  case 84: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 120: return 2
  case 88: return 2
  case 117: return -1
  case 116: return -1
  case 101: return -1
  case 69: return -1
  case 99: return -1
  case 67: return -1
  case 85: return -1
  case 84: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[4] = func(r rune) int {
  switch(r) {
  case 120: return -1
  case 88: return -1
  case 117: return 5
  case 116: return -1
  case 101: return -1
  case 69: return -1
  case 99: return -1
  case 67: return -1
  case 85: return 5
  case 84: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[5] = func(r rune) int {
  switch(r) {
  case 120: return -1
  case 88: return -1
  case 117: return -1
  case 116: return 6
  case 101: return -1
  case 69: return -1
  case 99: return -1
  case 67: return -1
  case 85: return -1
  case 84: return 6
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[75].acc = acc[:]
a0[75].f = fun[:]
a0[75].id = 75
}
{
var acc [7]bool
var fun [7]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 101: return 1
  case 69: return 1
  case 120: return -1
  case 88: return -1
  case 105: return -1
  case 73: return -1
  case 115: return -1
  case 83: return -1
  case 116: return -1
  case 84: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 69: return -1
  case 120: return 2
  case 88: return 2
  case 105: return -1
  case 73: return -1
  case 115: return -1
  case 83: return -1
  case 116: return -1
  case 84: return -1
  case 101: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 101: return -1
  case 69: return -1
  case 120: return -1
  case 88: return -1
  case 105: return 3
  case 73: return 3
  case 115: return -1
  case 83: return -1
  case 116: return -1
  case 84: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 101: return -1
  case 69: return -1
  case 120: return -1
  case 88: return -1
  case 105: return -1
  case 73: return -1
  case 115: return 4
  case 83: return 4
  case 116: return -1
  case 84: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[4] = func(r rune) int {
  switch(r) {
  case 69: return -1
  case 120: return -1
  case 88: return -1
  case 105: return -1
  case 73: return -1
  case 115: return -1
  case 83: return -1
  case 116: return 5
  case 84: return 5
  case 101: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[5] = func(r rune) int {
  switch(r) {
  case 101: return -1
  case 69: return -1
  case 120: return -1
  case 88: return -1
  case 105: return -1
  case 73: return -1
  case 115: return 6
  case 83: return 6
  case 116: return -1
  case 84: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[6] = true
fun[6] = func(r rune) int {
  switch(r) {
  case 69: return -1
  case 120: return -1
  case 88: return -1
  case 105: return -1
  case 73: return -1
  case 115: return -1
  case 83: return -1
  case 116: return -1
  case 84: return -1
  case 101: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[76].acc = acc[:]
a0[76].f = fun[:]
a0[76].id = 76
}
{
var acc [8]bool
var fun [8]func(rune) int
fun[1] = func(r rune) int {
  switch(r) {
  case 65: return -1
  case 73: return -1
  case 78: return -1
  case 120: return 2
  case 97: return -1
  case 110: return -1
  case 69: return -1
  case 88: return 2
  case 112: return -1
  case 80: return -1
  case 105: return -1
  case 101: return -1
  case 108: return -1
  case 76: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 65: return -1
  case 73: return -1
  case 78: return -1
  case 120: return -1
  case 97: return -1
  case 110: return -1
  case 69: return -1
  case 88: return -1
  case 112: return 3
  case 80: return 3
  case 105: return -1
  case 101: return -1
  case 108: return -1
  case 76: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 120: return -1
  case 97: return -1
  case 110: return -1
  case 69: return -1
  case 88: return -1
  case 112: return -1
  case 80: return -1
  case 105: return -1
  case 101: return -1
  case 108: return 4
  case 76: return 4
  case 65: return -1
  case 73: return -1
  case 78: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[4] = func(r rune) int {
  switch(r) {
  case 101: return -1
  case 108: return -1
  case 76: return -1
  case 65: return 5
  case 73: return -1
  case 78: return -1
  case 120: return -1
  case 97: return 5
  case 110: return -1
  case 69: return -1
  case 88: return -1
  case 112: return -1
  case 80: return -1
  case 105: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[5] = func(r rune) int {
  switch(r) {
  case 101: return -1
  case 108: return -1
  case 76: return -1
  case 65: return -1
  case 73: return 6
  case 78: return -1
  case 120: return -1
  case 97: return -1
  case 110: return -1
  case 69: return -1
  case 88: return -1
  case 112: return -1
  case 80: return -1
  case 105: return 6
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[7] = true
fun[7] = func(r rune) int {
  switch(r) {
  case 120: return -1
  case 97: return -1
  case 110: return -1
  case 69: return -1
  case 88: return -1
  case 112: return -1
  case 80: return -1
  case 105: return -1
  case 101: return -1
  case 108: return -1
  case 76: return -1
  case 65: return -1
  case 73: return -1
  case 78: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[0] = func(r rune) int {
  switch(r) {
  case 65: return -1
  case 73: return -1
  case 78: return -1
  case 120: return -1
  case 97: return -1
  case 110: return -1
  case 69: return 1
  case 88: return -1
  case 112: return -1
  case 80: return -1
  case 105: return -1
  case 101: return 1
  case 108: return -1
  case 76: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[6] = func(r rune) int {
  switch(r) {
  case 69: return -1
  case 88: return -1
  case 112: return -1
  case 80: return -1
  case 105: return -1
  case 101: return -1
  case 108: return -1
  case 76: return -1
  case 65: return -1
  case 73: return -1
  case 78: return 7
  case 120: return -1
  case 97: return -1
  case 110: return 7
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[77].acc = acc[:]
a0[77].f = fun[:]
a0[77].id = 77
}
{
var acc [6]bool
var fun [6]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 102: return 1
  case 70: return 1
  case 97: return -1
  case 65: return -1
  case 108: return -1
  case 76: return -1
  case 101: return -1
  case 69: return -1
  case 115: return -1
  case 83: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 102: return -1
  case 70: return -1
  case 97: return 2
  case 65: return 2
  case 108: return -1
  case 76: return -1
  case 101: return -1
  case 69: return -1
  case 115: return -1
  case 83: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 115: return -1
  case 83: return -1
  case 102: return -1
  case 70: return -1
  case 97: return -1
  case 65: return -1
  case 108: return 3
  case 76: return 3
  case 101: return -1
  case 69: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 102: return -1
  case 70: return -1
  case 97: return -1
  case 65: return -1
  case 108: return -1
  case 76: return -1
  case 101: return -1
  case 69: return -1
  case 115: return 4
  case 83: return 4
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[4] = func(r rune) int {
  switch(r) {
  case 115: return -1
  case 83: return -1
  case 102: return -1
  case 70: return -1
  case 97: return -1
  case 65: return -1
  case 108: return -1
  case 76: return -1
  case 101: return 5
  case 69: return 5
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[5] = true
fun[5] = func(r rune) int {
  switch(r) {
  case 115: return -1
  case 83: return -1
  case 102: return -1
  case 70: return -1
  case 97: return -1
  case 65: return -1
  case 108: return -1
  case 76: return -1
  case 101: return -1
  case 69: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[78].acc = acc[:]
a0[78].f = fun[:]
a0[78].id = 78
}
{
var acc [6]bool
var fun [6]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 102: return 1
  case 114: return -1
  case 70: return 1
  case 105: return -1
  case 73: return -1
  case 82: return -1
  case 115: return -1
  case 83: return -1
  case 116: return -1
  case 84: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 102: return -1
  case 114: return -1
  case 70: return -1
  case 105: return 2
  case 73: return 2
  case 82: return -1
  case 115: return -1
  case 83: return -1
  case 116: return -1
  case 84: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 70: return -1
  case 105: return -1
  case 73: return -1
  case 82: return 3
  case 115: return -1
  case 83: return -1
  case 116: return -1
  case 84: return -1
  case 102: return -1
  case 114: return 3
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 70: return -1
  case 105: return -1
  case 73: return -1
  case 82: return -1
  case 115: return 4
  case 83: return 4
  case 116: return -1
  case 84: return -1
  case 102: return -1
  case 114: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[4] = func(r rune) int {
  switch(r) {
  case 102: return -1
  case 114: return -1
  case 70: return -1
  case 105: return -1
  case 73: return -1
  case 82: return -1
  case 115: return -1
  case 83: return -1
  case 116: return 5
  case 84: return 5
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[5] = true
fun[5] = func(r rune) int {
  switch(r) {
  case 102: return -1
  case 114: return -1
  case 70: return -1
  case 105: return -1
  case 73: return -1
  case 82: return -1
  case 115: return -1
  case 83: return -1
  case 116: return -1
  case 84: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[79].acc = acc[:]
a0[79].f = fun[:]
a0[79].id = 79
}
{
var acc [8]bool
var fun [8]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 70: return 1
  case 108: return -1
  case 97: return -1
  case 65: return -1
  case 116: return -1
  case 102: return 1
  case 76: return -1
  case 84: return -1
  case 101: return -1
  case 69: return -1
  case 110: return -1
  case 78: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 102: return -1
  case 76: return -1
  case 84: return -1
  case 101: return -1
  case 69: return -1
  case 110: return -1
  case 78: return -1
  case 70: return -1
  case 108: return -1
  case 97: return 3
  case 65: return 3
  case 116: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 70: return -1
  case 108: return -1
  case 97: return -1
  case 65: return -1
  case 116: return 4
  case 102: return -1
  case 76: return -1
  case 84: return 4
  case 101: return -1
  case 69: return -1
  case 110: return -1
  case 78: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[5] = func(r rune) int {
  switch(r) {
  case 70: return -1
  case 108: return -1
  case 97: return -1
  case 65: return -1
  case 116: return -1
  case 102: return -1
  case 76: return -1
  case 84: return -1
  case 101: return 6
  case 69: return 6
  case 110: return -1
  case 78: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 70: return -1
  case 108: return 2
  case 97: return -1
  case 65: return -1
  case 116: return -1
  case 102: return -1
  case 76: return 2
  case 84: return -1
  case 101: return -1
  case 69: return -1
  case 110: return -1
  case 78: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[4] = func(r rune) int {
  switch(r) {
  case 70: return -1
  case 108: return -1
  case 97: return -1
  case 65: return -1
  case 116: return 5
  case 102: return -1
  case 76: return -1
  case 84: return 5
  case 101: return -1
  case 69: return -1
  case 110: return -1
  case 78: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[6] = func(r rune) int {
  switch(r) {
  case 102: return -1
  case 76: return -1
  case 84: return -1
  case 101: return -1
  case 69: return -1
  case 110: return 7
  case 78: return 7
  case 70: return -1
  case 108: return -1
  case 97: return -1
  case 65: return -1
  case 116: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[7] = true
fun[7] = func(r rune) int {
  switch(r) {
  case 70: return -1
  case 108: return -1
  case 97: return -1
  case 65: return -1
  case 116: return -1
  case 102: return -1
  case 76: return -1
  case 84: return -1
  case 101: return -1
  case 69: return -1
  case 110: return -1
  case 78: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[80].acc = acc[:]
a0[80].f = fun[:]
a0[80].id = 80
}
{
var acc [4]bool
var fun [4]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 102: return 1
  case 70: return 1
  case 111: return -1
  case 79: return -1
  case 114: return -1
  case 82: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 102: return -1
  case 70: return -1
  case 111: return 2
  case 79: return 2
  case 114: return -1
  case 82: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 102: return -1
  case 70: return -1
  case 111: return -1
  case 79: return -1
  case 114: return 3
  case 82: return 3
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[3] = true
fun[3] = func(r rune) int {
  switch(r) {
  case 102: return -1
  case 70: return -1
  case 111: return -1
  case 79: return -1
  case 114: return -1
  case 82: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[81].acc = acc[:]
a0[81].f = fun[:]
a0[81].id = 81
}
{
var acc [5]bool
var fun [5]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 102: return 1
  case 70: return 1
  case 114: return -1
  case 82: return -1
  case 111: return -1
  case 79: return -1
  case 109: return -1
  case 77: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 102: return -1
  case 70: return -1
  case 114: return 2
  case 82: return 2
  case 111: return -1
  case 79: return -1
  case 109: return -1
  case 77: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 102: return -1
  case 70: return -1
  case 114: return -1
  case 82: return -1
  case 111: return 3
  case 79: return 3
  case 109: return -1
  case 77: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 102: return -1
  case 70: return -1
  case 114: return -1
  case 82: return -1
  case 111: return -1
  case 79: return -1
  case 109: return 4
  case 77: return 4
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[4] = true
fun[4] = func(r rune) int {
  switch(r) {
  case 102: return -1
  case 70: return -1
  case 114: return -1
  case 82: return -1
  case 111: return -1
  case 79: return -1
  case 109: return -1
  case 77: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[82].acc = acc[:]
a0[82].f = fun[:]
a0[82].id = 82
}
{
var acc [9]bool
var fun [9]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 117: return -1
  case 111: return -1
  case 99: return -1
  case 67: return -1
  case 79: return -1
  case 70: return 1
  case 85: return -1
  case 78: return -1
  case 116: return -1
  case 105: return -1
  case 102: return 1
  case 110: return -1
  case 84: return -1
  case 73: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 99: return 4
  case 67: return 4
  case 79: return -1
  case 70: return -1
  case 85: return -1
  case 78: return -1
  case 116: return -1
  case 105: return -1
  case 102: return -1
  case 110: return -1
  case 84: return -1
  case 73: return -1
  case 117: return -1
  case 111: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[4] = func(r rune) int {
  switch(r) {
  case 70: return -1
  case 85: return -1
  case 78: return -1
  case 116: return 5
  case 105: return -1
  case 102: return -1
  case 110: return -1
  case 84: return 5
  case 73: return -1
  case 117: return -1
  case 111: return -1
  case 99: return -1
  case 67: return -1
  case 79: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[5] = func(r rune) int {
  switch(r) {
  case 102: return -1
  case 110: return -1
  case 84: return -1
  case 73: return 6
  case 117: return -1
  case 111: return -1
  case 99: return -1
  case 67: return -1
  case 79: return -1
  case 70: return -1
  case 85: return -1
  case 78: return -1
  case 116: return -1
  case 105: return 6
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 102: return -1
  case 110: return -1
  case 84: return -1
  case 73: return -1
  case 117: return 2
  case 111: return -1
  case 99: return -1
  case 67: return -1
  case 79: return -1
  case 70: return -1
  case 85: return 2
  case 78: return -1
  case 116: return -1
  case 105: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 99: return -1
  case 67: return -1
  case 79: return -1
  case 70: return -1
  case 85: return -1
  case 78: return 3
  case 116: return -1
  case 105: return -1
  case 102: return -1
  case 110: return 3
  case 84: return -1
  case 73: return -1
  case 117: return -1
  case 111: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[6] = func(r rune) int {
  switch(r) {
  case 99: return -1
  case 67: return -1
  case 79: return 7
  case 70: return -1
  case 85: return -1
  case 78: return -1
  case 116: return -1
  case 105: return -1
  case 102: return -1
  case 110: return -1
  case 84: return -1
  case 73: return -1
  case 117: return -1
  case 111: return 7
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[7] = func(r rune) int {
  switch(r) {
  case 70: return -1
  case 85: return -1
  case 78: return 8
  case 116: return -1
  case 105: return -1
  case 102: return -1
  case 110: return 8
  case 84: return -1
  case 73: return -1
  case 117: return -1
  case 111: return -1
  case 99: return -1
  case 67: return -1
  case 79: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[8] = true
fun[8] = func(r rune) int {
  switch(r) {
  case 102: return -1
  case 110: return -1
  case 84: return -1
  case 73: return -1
  case 117: return -1
  case 111: return -1
  case 99: return -1
  case 67: return -1
  case 79: return -1
  case 70: return -1
  case 85: return -1
  case 78: return -1
  case 116: return -1
  case 105: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[83].acc = acc[:]
a0[83].f = fun[:]
a0[83].id = 83
}
{
var acc [6]bool
var fun [6]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 103: return 1
  case 82: return -1
  case 97: return -1
  case 110: return -1
  case 78: return -1
  case 71: return 1
  case 114: return -1
  case 65: return -1
  case 116: return -1
  case 84: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 103: return -1
  case 82: return 2
  case 97: return -1
  case 110: return -1
  case 78: return -1
  case 71: return -1
  case 114: return 2
  case 65: return -1
  case 116: return -1
  case 84: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 71: return -1
  case 114: return -1
  case 65: return 3
  case 116: return -1
  case 84: return -1
  case 103: return -1
  case 82: return -1
  case 97: return 3
  case 110: return -1
  case 78: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 103: return -1
  case 82: return -1
  case 97: return -1
  case 110: return 4
  case 78: return 4
  case 71: return -1
  case 114: return -1
  case 65: return -1
  case 116: return -1
  case 84: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[4] = func(r rune) int {
  switch(r) {
  case 71: return -1
  case 114: return -1
  case 65: return -1
  case 116: return 5
  case 84: return 5
  case 103: return -1
  case 82: return -1
  case 97: return -1
  case 110: return -1
  case 78: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[5] = true
fun[5] = func(r rune) int {
  switch(r) {
  case 71: return -1
  case 114: return -1
  case 65: return -1
  case 116: return -1
  case 84: return -1
  case 103: return -1
  case 82: return -1
  case 97: return -1
  case 110: return -1
  case 78: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[84].acc = acc[:]
a0[84].f = fun[:]
a0[84].id = 84
}
{
var acc [6]bool
var fun [6]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 103: return 1
  case 114: return -1
  case 82: return -1
  case 79: return -1
  case 117: return -1
  case 85: return -1
  case 112: return -1
  case 71: return 1
  case 111: return -1
  case 80: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 71: return -1
  case 111: return -1
  case 80: return -1
  case 103: return -1
  case 114: return 2
  case 82: return 2
  case 79: return -1
  case 117: return -1
  case 85: return -1
  case 112: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 71: return -1
  case 111: return 3
  case 80: return -1
  case 103: return -1
  case 114: return -1
  case 82: return -1
  case 79: return 3
  case 117: return -1
  case 85: return -1
  case 112: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 103: return -1
  case 114: return -1
  case 82: return -1
  case 79: return -1
  case 117: return 4
  case 85: return 4
  case 112: return -1
  case 71: return -1
  case 111: return -1
  case 80: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[4] = func(r rune) int {
  switch(r) {
  case 103: return -1
  case 114: return -1
  case 82: return -1
  case 79: return -1
  case 117: return -1
  case 85: return -1
  case 112: return 5
  case 71: return -1
  case 111: return -1
  case 80: return 5
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[5] = true
fun[5] = func(r rune) int {
  switch(r) {
  case 103: return -1
  case 114: return -1
  case 82: return -1
  case 79: return -1
  case 117: return -1
  case 85: return -1
  case 112: return -1
  case 71: return -1
  case 111: return -1
  case 80: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[85].acc = acc[:]
a0[85].f = fun[:]
a0[85].id = 85
}
{
var acc [7]bool
var fun [7]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 104: return 1
  case 97: return -1
  case 65: return -1
  case 86: return -1
  case 105: return -1
  case 110: return -1
  case 78: return -1
  case 103: return -1
  case 72: return 1
  case 118: return -1
  case 73: return -1
  case 71: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 104: return -1
  case 97: return 2
  case 65: return 2
  case 86: return -1
  case 105: return -1
  case 110: return -1
  case 78: return -1
  case 103: return -1
  case 72: return -1
  case 118: return -1
  case 73: return -1
  case 71: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 104: return -1
  case 97: return -1
  case 65: return -1
  case 86: return 3
  case 105: return -1
  case 110: return -1
  case 78: return -1
  case 103: return -1
  case 72: return -1
  case 118: return 3
  case 73: return -1
  case 71: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 104: return -1
  case 97: return -1
  case 65: return -1
  case 86: return -1
  case 105: return 4
  case 110: return -1
  case 78: return -1
  case 103: return -1
  case 72: return -1
  case 118: return -1
  case 73: return 4
  case 71: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[4] = func(r rune) int {
  switch(r) {
  case 104: return -1
  case 97: return -1
  case 65: return -1
  case 86: return -1
  case 105: return -1
  case 110: return 5
  case 78: return 5
  case 103: return -1
  case 72: return -1
  case 118: return -1
  case 73: return -1
  case 71: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[5] = func(r rune) int {
  switch(r) {
  case 72: return -1
  case 118: return -1
  case 73: return -1
  case 71: return 6
  case 104: return -1
  case 97: return -1
  case 65: return -1
  case 86: return -1
  case 105: return -1
  case 110: return -1
  case 78: return -1
  case 103: return 6
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[6] = true
fun[6] = func(r rune) int {
  switch(r) {
  case 104: return -1
  case 97: return -1
  case 65: return -1
  case 86: return -1
  case 105: return -1
  case 110: return -1
  case 78: return -1
  case 103: return -1
  case 72: return -1
  case 118: return -1
  case 73: return -1
  case 71: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[86].acc = acc[:]
a0[86].f = fun[:]
a0[86].id = 86
}
{
var acc [3]bool
var fun [3]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 105: return 1
  case 73: return 1
  case 102: return -1
  case 70: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 105: return -1
  case 73: return -1
  case 102: return 2
  case 70: return 2
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[2] = true
fun[2] = func(r rune) int {
  switch(r) {
  case 105: return -1
  case 73: return -1
  case 102: return -1
  case 70: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[87].acc = acc[:]
a0[87].f = fun[:]
a0[87].id = 87
}
{
var acc [3]bool
var fun [3]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 105: return 1
  case 73: return 1
  case 110: return -1
  case 78: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 105: return -1
  case 73: return -1
  case 110: return 2
  case 78: return 2
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[2] = true
fun[2] = func(r rune) int {
  switch(r) {
  case 105: return -1
  case 73: return -1
  case 110: return -1
  case 78: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[88].acc = acc[:]
a0[88].f = fun[:]
a0[88].id = 88
}
{
var acc [8]bool
var fun [8]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 67: return -1
  case 76: return -1
  case 85: return -1
  case 101: return -1
  case 105: return 1
  case 117: return -1
  case 73: return 1
  case 110: return -1
  case 99: return -1
  case 108: return -1
  case 100: return -1
  case 68: return -1
  case 78: return -1
  case 69: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 73: return -1
  case 110: return -1
  case 99: return -1
  case 108: return 4
  case 100: return -1
  case 68: return -1
  case 78: return -1
  case 69: return -1
  case 67: return -1
  case 76: return 4
  case 85: return -1
  case 101: return -1
  case 105: return -1
  case 117: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[7] = true
fun[7] = func(r rune) int {
  switch(r) {
  case 78: return -1
  case 69: return -1
  case 67: return -1
  case 76: return -1
  case 85: return -1
  case 101: return -1
  case 105: return -1
  case 117: return -1
  case 73: return -1
  case 110: return -1
  case 99: return -1
  case 108: return -1
  case 100: return -1
  case 68: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 78: return 2
  case 69: return -1
  case 67: return -1
  case 76: return -1
  case 85: return -1
  case 101: return -1
  case 105: return -1
  case 117: return -1
  case 73: return -1
  case 110: return 2
  case 99: return -1
  case 108: return -1
  case 100: return -1
  case 68: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 105: return -1
  case 117: return -1
  case 73: return -1
  case 110: return -1
  case 99: return 3
  case 108: return -1
  case 100: return -1
  case 68: return -1
  case 78: return -1
  case 69: return -1
  case 67: return 3
  case 76: return -1
  case 85: return -1
  case 101: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[4] = func(r rune) int {
  switch(r) {
  case 67: return -1
  case 76: return -1
  case 85: return 5
  case 101: return -1
  case 105: return -1
  case 117: return 5
  case 73: return -1
  case 110: return -1
  case 99: return -1
  case 108: return -1
  case 100: return -1
  case 68: return -1
  case 78: return -1
  case 69: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[5] = func(r rune) int {
  switch(r) {
  case 67: return -1
  case 76: return -1
  case 85: return -1
  case 101: return -1
  case 105: return -1
  case 117: return -1
  case 73: return -1
  case 110: return -1
  case 99: return -1
  case 108: return -1
  case 100: return 6
  case 68: return 6
  case 78: return -1
  case 69: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[6] = func(r rune) int {
  switch(r) {
  case 67: return -1
  case 76: return -1
  case 85: return -1
  case 101: return 7
  case 105: return -1
  case 117: return -1
  case 73: return -1
  case 110: return -1
  case 99: return -1
  case 108: return -1
  case 100: return -1
  case 68: return -1
  case 78: return -1
  case 69: return 7
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[89].acc = acc[:]
a0[89].f = fun[:]
a0[89].id = 89
}
{
var acc [6]bool
var fun [6]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 110: return -1
  case 100: return -1
  case 68: return -1
  case 101: return -1
  case 69: return -1
  case 88: return -1
  case 105: return 1
  case 73: return 1
  case 78: return -1
  case 120: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 105: return -1
  case 73: return -1
  case 78: return 2
  case 120: return -1
  case 110: return 2
  case 100: return -1
  case 68: return -1
  case 101: return -1
  case 69: return -1
  case 88: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 105: return -1
  case 73: return -1
  case 78: return -1
  case 120: return -1
  case 110: return -1
  case 100: return 3
  case 68: return 3
  case 101: return -1
  case 69: return -1
  case 88: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 110: return -1
  case 100: return -1
  case 68: return -1
  case 101: return 4
  case 69: return 4
  case 88: return -1
  case 105: return -1
  case 73: return -1
  case 78: return -1
  case 120: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[4] = func(r rune) int {
  switch(r) {
  case 105: return -1
  case 73: return -1
  case 78: return -1
  case 120: return 5
  case 110: return -1
  case 100: return -1
  case 68: return -1
  case 101: return -1
  case 69: return -1
  case 88: return 5
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[5] = true
fun[5] = func(r rune) int {
  switch(r) {
  case 110: return -1
  case 100: return -1
  case 68: return -1
  case 101: return -1
  case 69: return -1
  case 88: return -1
  case 105: return -1
  case 73: return -1
  case 78: return -1
  case 120: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[90].acc = acc[:]
a0[90].f = fun[:]
a0[90].id = 90
}
{
var acc [7]bool
var fun [7]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 105: return 1
  case 73: return 1
  case 110: return -1
  case 78: return -1
  case 108: return -1
  case 76: return -1
  case 101: return -1
  case 69: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 105: return -1
  case 73: return -1
  case 110: return 2
  case 78: return 2
  case 108: return -1
  case 76: return -1
  case 101: return -1
  case 69: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 105: return -1
  case 73: return -1
  case 110: return -1
  case 78: return -1
  case 108: return 3
  case 76: return 3
  case 101: return -1
  case 69: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 105: return 4
  case 73: return 4
  case 110: return -1
  case 78: return -1
  case 108: return -1
  case 76: return -1
  case 101: return -1
  case 69: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[4] = func(r rune) int {
  switch(r) {
  case 105: return -1
  case 73: return -1
  case 110: return 5
  case 78: return 5
  case 108: return -1
  case 76: return -1
  case 101: return -1
  case 69: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[5] = func(r rune) int {
  switch(r) {
  case 105: return -1
  case 73: return -1
  case 110: return -1
  case 78: return -1
  case 108: return -1
  case 76: return -1
  case 101: return 6
  case 69: return 6
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[6] = true
fun[6] = func(r rune) int {
  switch(r) {
  case 105: return -1
  case 73: return -1
  case 110: return -1
  case 78: return -1
  case 108: return -1
  case 76: return -1
  case 101: return -1
  case 69: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[91].acc = acc[:]
a0[91].f = fun[:]
a0[91].id = 91
}
{
var acc [6]bool
var fun [6]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 105: return 1
  case 73: return 1
  case 110: return -1
  case 78: return -1
  case 101: return -1
  case 69: return -1
  case 114: return -1
  case 82: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 105: return -1
  case 73: return -1
  case 110: return 2
  case 78: return 2
  case 101: return -1
  case 69: return -1
  case 114: return -1
  case 82: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 105: return -1
  case 73: return -1
  case 110: return 3
  case 78: return 3
  case 101: return -1
  case 69: return -1
  case 114: return -1
  case 82: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 105: return -1
  case 73: return -1
  case 110: return -1
  case 78: return -1
  case 101: return 4
  case 69: return 4
  case 114: return -1
  case 82: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[4] = func(r rune) int {
  switch(r) {
  case 105: return -1
  case 73: return -1
  case 110: return -1
  case 78: return -1
  case 101: return -1
  case 69: return -1
  case 114: return 5
  case 82: return 5
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[5] = true
fun[5] = func(r rune) int {
  switch(r) {
  case 105: return -1
  case 73: return -1
  case 110: return -1
  case 78: return -1
  case 101: return -1
  case 69: return -1
  case 114: return -1
  case 82: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[92].acc = acc[:]
a0[92].f = fun[:]
a0[92].id = 92
}
{
var acc [7]bool
var fun [7]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 78: return -1
  case 115: return -1
  case 82: return -1
  case 116: return -1
  case 84: return -1
  case 105: return 1
  case 73: return 1
  case 110: return -1
  case 83: return -1
  case 101: return -1
  case 69: return -1
  case 114: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 78: return 2
  case 115: return -1
  case 82: return -1
  case 116: return -1
  case 84: return -1
  case 105: return -1
  case 73: return -1
  case 110: return 2
  case 83: return -1
  case 101: return -1
  case 69: return -1
  case 114: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 78: return -1
  case 115: return 3
  case 82: return -1
  case 116: return -1
  case 84: return -1
  case 105: return -1
  case 73: return -1
  case 110: return -1
  case 83: return 3
  case 101: return -1
  case 69: return -1
  case 114: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 105: return -1
  case 73: return -1
  case 110: return -1
  case 83: return -1
  case 101: return 4
  case 69: return 4
  case 114: return -1
  case 78: return -1
  case 115: return -1
  case 82: return -1
  case 116: return -1
  case 84: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[4] = func(r rune) int {
  switch(r) {
  case 105: return -1
  case 73: return -1
  case 110: return -1
  case 83: return -1
  case 101: return -1
  case 69: return -1
  case 114: return 5
  case 78: return -1
  case 115: return -1
  case 82: return 5
  case 116: return -1
  case 84: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[5] = func(r rune) int {
  switch(r) {
  case 78: return -1
  case 115: return -1
  case 82: return -1
  case 116: return 6
  case 84: return 6
  case 105: return -1
  case 73: return -1
  case 110: return -1
  case 83: return -1
  case 101: return -1
  case 69: return -1
  case 114: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[6] = true
fun[6] = func(r rune) int {
  switch(r) {
  case 105: return -1
  case 73: return -1
  case 110: return -1
  case 83: return -1
  case 101: return -1
  case 69: return -1
  case 114: return -1
  case 78: return -1
  case 115: return -1
  case 82: return -1
  case 116: return -1
  case 84: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[93].acc = acc[:]
a0[93].f = fun[:]
a0[93].id = 93
}
{
var acc [10]bool
var fun [10]func(rune) int
fun[2] = func(r rune) int {
  switch(r) {
  case 73: return -1
  case 84: return 3
  case 69: return -1
  case 114: return -1
  case 105: return -1
  case 110: return -1
  case 116: return 3
  case 101: return -1
  case 115: return -1
  case 83: return -1
  case 99: return -1
  case 78: return -1
  case 82: return -1
  case 67: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[0] = func(r rune) int {
  switch(r) {
  case 105: return 1
  case 110: return -1
  case 116: return -1
  case 101: return -1
  case 115: return -1
  case 83: return -1
  case 99: return -1
  case 78: return -1
  case 82: return -1
  case 67: return -1
  case 73: return 1
  case 84: return -1
  case 69: return -1
  case 114: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 73: return -1
  case 84: return -1
  case 69: return -1
  case 114: return -1
  case 105: return -1
  case 110: return 2
  case 116: return -1
  case 101: return -1
  case 115: return -1
  case 83: return -1
  case 99: return -1
  case 78: return 2
  case 82: return -1
  case 67: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 105: return -1
  case 110: return -1
  case 116: return -1
  case 101: return 4
  case 115: return -1
  case 83: return -1
  case 99: return -1
  case 78: return -1
  case 82: return -1
  case 67: return -1
  case 73: return -1
  case 84: return -1
  case 69: return 4
  case 114: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[4] = func(r rune) int {
  switch(r) {
  case 78: return -1
  case 82: return 5
  case 67: return -1
  case 73: return -1
  case 84: return -1
  case 69: return -1
  case 114: return 5
  case 105: return -1
  case 110: return -1
  case 116: return -1
  case 101: return -1
  case 115: return -1
  case 83: return -1
  case 99: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[5] = func(r rune) int {
  switch(r) {
  case 78: return -1
  case 82: return -1
  case 67: return -1
  case 73: return -1
  case 84: return -1
  case 69: return -1
  case 114: return -1
  case 105: return -1
  case 110: return -1
  case 116: return -1
  case 101: return -1
  case 115: return 6
  case 83: return 6
  case 99: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[6] = func(r rune) int {
  switch(r) {
  case 105: return -1
  case 110: return -1
  case 116: return -1
  case 101: return 7
  case 115: return -1
  case 83: return -1
  case 99: return -1
  case 78: return -1
  case 82: return -1
  case 67: return -1
  case 73: return -1
  case 84: return -1
  case 69: return 7
  case 114: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[7] = func(r rune) int {
  switch(r) {
  case 105: return -1
  case 110: return -1
  case 116: return -1
  case 101: return -1
  case 115: return -1
  case 83: return -1
  case 99: return 8
  case 78: return -1
  case 82: return -1
  case 67: return 8
  case 73: return -1
  case 84: return -1
  case 69: return -1
  case 114: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[8] = func(r rune) int {
  switch(r) {
  case 78: return -1
  case 82: return -1
  case 67: return -1
  case 73: return -1
  case 84: return 9
  case 69: return -1
  case 114: return -1
  case 105: return -1
  case 110: return -1
  case 116: return 9
  case 101: return -1
  case 115: return -1
  case 83: return -1
  case 99: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[9] = true
fun[9] = func(r rune) int {
  switch(r) {
  case 105: return -1
  case 110: return -1
  case 116: return -1
  case 101: return -1
  case 115: return -1
  case 83: return -1
  case 99: return -1
  case 78: return -1
  case 82: return -1
  case 67: return -1
  case 73: return -1
  case 84: return -1
  case 69: return -1
  case 114: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[94].acc = acc[:]
a0[94].f = fun[:]
a0[94].id = 94
}
{
var acc [5]bool
var fun [5]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 105: return 1
  case 73: return 1
  case 110: return -1
  case 78: return -1
  case 116: return -1
  case 84: return -1
  case 111: return -1
  case 79: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 105: return -1
  case 73: return -1
  case 110: return 2
  case 78: return 2
  case 116: return -1
  case 84: return -1
  case 111: return -1
  case 79: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 105: return -1
  case 73: return -1
  case 110: return -1
  case 78: return -1
  case 116: return 3
  case 84: return 3
  case 111: return -1
  case 79: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 105: return -1
  case 73: return -1
  case 110: return -1
  case 78: return -1
  case 116: return -1
  case 84: return -1
  case 111: return 4
  case 79: return 4
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[4] = true
fun[4] = func(r rune) int {
  switch(r) {
  case 105: return -1
  case 73: return -1
  case 110: return -1
  case 78: return -1
  case 116: return -1
  case 84: return -1
  case 111: return -1
  case 79: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[95].acc = acc[:]
a0[95].f = fun[:]
a0[95].id = 95
}
{
var acc [3]bool
var fun [3]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 105: return 1
  case 73: return 1
  case 115: return -1
  case 83: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 105: return -1
  case 73: return -1
  case 115: return 2
  case 83: return 2
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[2] = true
fun[2] = func(r rune) int {
  switch(r) {
  case 105: return -1
  case 73: return -1
  case 115: return -1
  case 83: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[96].acc = acc[:]
a0[96].f = fun[:]
a0[96].id = 96
}
{
var acc [5]bool
var fun [5]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 106: return 1
  case 74: return 1
  case 111: return -1
  case 79: return -1
  case 105: return -1
  case 73: return -1
  case 110: return -1
  case 78: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 106: return -1
  case 74: return -1
  case 111: return 2
  case 79: return 2
  case 105: return -1
  case 73: return -1
  case 110: return -1
  case 78: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 106: return -1
  case 74: return -1
  case 111: return -1
  case 79: return -1
  case 105: return 3
  case 73: return 3
  case 110: return -1
  case 78: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 106: return -1
  case 74: return -1
  case 111: return -1
  case 79: return -1
  case 105: return -1
  case 73: return -1
  case 110: return 4
  case 78: return 4
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[4] = true
fun[4] = func(r rune) int {
  switch(r) {
  case 106: return -1
  case 74: return -1
  case 111: return -1
  case 79: return -1
  case 105: return -1
  case 73: return -1
  case 110: return -1
  case 78: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[97].acc = acc[:]
a0[97].f = fun[:]
a0[97].id = 97
}
{
var acc [4]bool
var fun [4]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 107: return 1
  case 75: return 1
  case 101: return -1
  case 69: return -1
  case 121: return -1
  case 89: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 107: return -1
  case 75: return -1
  case 101: return 2
  case 69: return 2
  case 121: return -1
  case 89: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 107: return -1
  case 75: return -1
  case 101: return -1
  case 69: return -1
  case 121: return 3
  case 89: return 3
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[3] = true
fun[3] = func(r rune) int {
  switch(r) {
  case 107: return -1
  case 75: return -1
  case 101: return -1
  case 69: return -1
  case 121: return -1
  case 89: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[98].acc = acc[:]
a0[98].f = fun[:]
a0[98].id = 98
}
{
var acc [5]bool
var fun [5]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 107: return 1
  case 75: return 1
  case 101: return -1
  case 69: return -1
  case 121: return -1
  case 89: return -1
  case 115: return -1
  case 83: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 107: return -1
  case 75: return -1
  case 101: return 2
  case 69: return 2
  case 121: return -1
  case 89: return -1
  case 115: return -1
  case 83: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 107: return -1
  case 75: return -1
  case 101: return -1
  case 69: return -1
  case 121: return 3
  case 89: return 3
  case 115: return -1
  case 83: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 107: return -1
  case 75: return -1
  case 101: return -1
  case 69: return -1
  case 121: return -1
  case 89: return -1
  case 115: return 4
  case 83: return 4
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[4] = true
fun[4] = func(r rune) int {
  switch(r) {
  case 107: return -1
  case 75: return -1
  case 101: return -1
  case 69: return -1
  case 121: return -1
  case 89: return -1
  case 115: return -1
  case 83: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[99].acc = acc[:]
a0[99].f = fun[:]
a0[99].id = 99
}
{
var acc [9]bool
var fun [9]func(rune) int
fun[2] = func(r rune) int {
  switch(r) {
  case 69: return -1
  case 83: return -1
  case 80: return -1
  case 75: return -1
  case 101: return -1
  case 65: return -1
  case 99: return -1
  case 67: return -1
  case 107: return -1
  case 121: return 3
  case 112: return -1
  case 97: return -1
  case 89: return 3
  case 115: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 69: return -1
  case 83: return 4
  case 80: return -1
  case 75: return -1
  case 101: return -1
  case 65: return -1
  case 99: return -1
  case 67: return -1
  case 107: return -1
  case 121: return -1
  case 112: return -1
  case 97: return -1
  case 89: return -1
  case 115: return 4
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[6] = func(r rune) int {
  switch(r) {
  case 107: return -1
  case 121: return -1
  case 112: return -1
  case 97: return -1
  case 89: return -1
  case 115: return -1
  case 69: return -1
  case 83: return -1
  case 80: return -1
  case 75: return -1
  case 101: return -1
  case 65: return -1
  case 99: return 7
  case 67: return 7
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[0] = func(r rune) int {
  switch(r) {
  case 107: return 1
  case 121: return -1
  case 112: return -1
  case 97: return -1
  case 89: return -1
  case 115: return -1
  case 69: return -1
  case 83: return -1
  case 80: return -1
  case 75: return 1
  case 101: return -1
  case 65: return -1
  case 99: return -1
  case 67: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 69: return 2
  case 83: return -1
  case 80: return -1
  case 75: return -1
  case 101: return 2
  case 65: return -1
  case 99: return -1
  case 67: return -1
  case 107: return -1
  case 121: return -1
  case 112: return -1
  case 97: return -1
  case 89: return -1
  case 115: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[4] = func(r rune) int {
  switch(r) {
  case 89: return -1
  case 115: return -1
  case 69: return -1
  case 83: return -1
  case 80: return 5
  case 75: return -1
  case 101: return -1
  case 65: return -1
  case 99: return -1
  case 67: return -1
  case 107: return -1
  case 121: return -1
  case 112: return 5
  case 97: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[5] = func(r rune) int {
  switch(r) {
  case 75: return -1
  case 101: return -1
  case 65: return 6
  case 99: return -1
  case 67: return -1
  case 107: return -1
  case 121: return -1
  case 112: return -1
  case 97: return 6
  case 89: return -1
  case 115: return -1
  case 69: return -1
  case 83: return -1
  case 80: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[7] = func(r rune) int {
  switch(r) {
  case 89: return -1
  case 115: return -1
  case 69: return 8
  case 83: return -1
  case 80: return -1
  case 75: return -1
  case 101: return 8
  case 65: return -1
  case 99: return -1
  case 67: return -1
  case 107: return -1
  case 121: return -1
  case 112: return -1
  case 97: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[8] = true
fun[8] = func(r rune) int {
  switch(r) {
  case 107: return -1
  case 121: return -1
  case 112: return -1
  case 97: return -1
  case 89: return -1
  case 115: return -1
  case 69: return -1
  case 83: return -1
  case 80: return -1
  case 75: return -1
  case 101: return -1
  case 65: return -1
  case 99: return -1
  case 67: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[100].acc = acc[:]
a0[100].f = fun[:]
a0[100].id = 100
}
{
var acc [5]bool
var fun [5]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 108: return 1
  case 76: return 1
  case 97: return -1
  case 65: return -1
  case 115: return -1
  case 83: return -1
  case 116: return -1
  case 84: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 108: return -1
  case 76: return -1
  case 97: return 2
  case 65: return 2
  case 115: return -1
  case 83: return -1
  case 116: return -1
  case 84: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 108: return -1
  case 76: return -1
  case 97: return -1
  case 65: return -1
  case 115: return 3
  case 83: return 3
  case 116: return -1
  case 84: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 108: return -1
  case 76: return -1
  case 97: return -1
  case 65: return -1
  case 115: return -1
  case 83: return -1
  case 116: return 4
  case 84: return 4
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[4] = true
fun[4] = func(r rune) int {
  switch(r) {
  case 108: return -1
  case 76: return -1
  case 97: return -1
  case 65: return -1
  case 115: return -1
  case 83: return -1
  case 116: return -1
  case 84: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[101].acc = acc[:]
a0[101].f = fun[:]
a0[101].id = 101
}
{
var acc [5]bool
var fun [5]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 108: return 1
  case 76: return 1
  case 101: return -1
  case 69: return -1
  case 102: return -1
  case 70: return -1
  case 116: return -1
  case 84: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 108: return -1
  case 76: return -1
  case 101: return 2
  case 69: return 2
  case 102: return -1
  case 70: return -1
  case 116: return -1
  case 84: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 108: return -1
  case 76: return -1
  case 101: return -1
  case 69: return -1
  case 102: return 3
  case 70: return 3
  case 116: return -1
  case 84: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 108: return -1
  case 76: return -1
  case 101: return -1
  case 69: return -1
  case 102: return -1
  case 70: return -1
  case 116: return 4
  case 84: return 4
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[4] = true
fun[4] = func(r rune) int {
  switch(r) {
  case 108: return -1
  case 76: return -1
  case 101: return -1
  case 69: return -1
  case 102: return -1
  case 70: return -1
  case 116: return -1
  case 84: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[102].acc = acc[:]
a0[102].f = fun[:]
a0[102].id = 102
}
{
var acc [4]bool
var fun [4]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 108: return 1
  case 76: return 1
  case 101: return -1
  case 69: return -1
  case 116: return -1
  case 84: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 108: return -1
  case 76: return -1
  case 101: return 2
  case 69: return 2
  case 116: return -1
  case 84: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 108: return -1
  case 76: return -1
  case 101: return -1
  case 69: return -1
  case 116: return 3
  case 84: return 3
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[3] = true
fun[3] = func(r rune) int {
  switch(r) {
  case 108: return -1
  case 76: return -1
  case 101: return -1
  case 69: return -1
  case 116: return -1
  case 84: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[103].acc = acc[:]
a0[103].f = fun[:]
a0[103].id = 103
}
{
var acc [8]bool
var fun [8]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 108: return 1
  case 76: return 1
  case 69: return -1
  case 116: return -1
  case 110: return -1
  case 78: return -1
  case 103: return -1
  case 101: return -1
  case 84: return -1
  case 105: return -1
  case 73: return -1
  case 71: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 108: return -1
  case 76: return -1
  case 69: return 2
  case 116: return -1
  case 110: return -1
  case 78: return -1
  case 103: return -1
  case 101: return 2
  case 84: return -1
  case 105: return -1
  case 73: return -1
  case 71: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 108: return -1
  case 76: return -1
  case 69: return -1
  case 116: return 4
  case 110: return -1
  case 78: return -1
  case 103: return -1
  case 101: return -1
  case 84: return 4
  case 105: return -1
  case 73: return -1
  case 71: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[4] = func(r rune) int {
  switch(r) {
  case 101: return -1
  case 84: return -1
  case 105: return 5
  case 73: return 5
  case 71: return -1
  case 108: return -1
  case 76: return -1
  case 69: return -1
  case 116: return -1
  case 110: return -1
  case 78: return -1
  case 103: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[5] = func(r rune) int {
  switch(r) {
  case 101: return -1
  case 84: return -1
  case 105: return -1
  case 73: return -1
  case 71: return -1
  case 108: return -1
  case 76: return -1
  case 69: return -1
  case 116: return -1
  case 110: return 6
  case 78: return 6
  case 103: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[6] = func(r rune) int {
  switch(r) {
  case 101: return -1
  case 84: return -1
  case 105: return -1
  case 73: return -1
  case 71: return 7
  case 108: return -1
  case 76: return -1
  case 69: return -1
  case 116: return -1
  case 110: return -1
  case 78: return -1
  case 103: return 7
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[7] = true
fun[7] = func(r rune) int {
  switch(r) {
  case 108: return -1
  case 76: return -1
  case 69: return -1
  case 116: return -1
  case 110: return -1
  case 78: return -1
  case 103: return -1
  case 101: return -1
  case 84: return -1
  case 105: return -1
  case 73: return -1
  case 71: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 101: return -1
  case 84: return 3
  case 105: return -1
  case 73: return -1
  case 71: return -1
  case 108: return -1
  case 76: return -1
  case 69: return -1
  case 116: return 3
  case 110: return -1
  case 78: return -1
  case 103: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[104].acc = acc[:]
a0[104].f = fun[:]
a0[104].id = 104
}
{
var acc [5]bool
var fun [5]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 108: return 1
  case 76: return 1
  case 105: return -1
  case 73: return -1
  case 107: return -1
  case 75: return -1
  case 101: return -1
  case 69: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 108: return -1
  case 76: return -1
  case 105: return 2
  case 73: return 2
  case 107: return -1
  case 75: return -1
  case 101: return -1
  case 69: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 108: return -1
  case 76: return -1
  case 105: return -1
  case 73: return -1
  case 107: return 3
  case 75: return 3
  case 101: return -1
  case 69: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 108: return -1
  case 76: return -1
  case 105: return -1
  case 73: return -1
  case 107: return -1
  case 75: return -1
  case 101: return 4
  case 69: return 4
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[4] = true
fun[4] = func(r rune) int {
  switch(r) {
  case 108: return -1
  case 76: return -1
  case 105: return -1
  case 73: return -1
  case 107: return -1
  case 75: return -1
  case 101: return -1
  case 69: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[105].acc = acc[:]
a0[105].f = fun[:]
a0[105].id = 105
}
{
var acc [6]bool
var fun [6]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 108: return 1
  case 76: return 1
  case 105: return -1
  case 73: return -1
  case 109: return -1
  case 77: return -1
  case 116: return -1
  case 84: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 108: return -1
  case 76: return -1
  case 105: return 2
  case 73: return 2
  case 109: return -1
  case 77: return -1
  case 116: return -1
  case 84: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 108: return -1
  case 76: return -1
  case 105: return -1
  case 73: return -1
  case 109: return 3
  case 77: return 3
  case 116: return -1
  case 84: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 108: return -1
  case 76: return -1
  case 105: return 4
  case 73: return 4
  case 109: return -1
  case 77: return -1
  case 116: return -1
  case 84: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[4] = func(r rune) int {
  switch(r) {
  case 108: return -1
  case 76: return -1
  case 105: return -1
  case 73: return -1
  case 109: return -1
  case 77: return -1
  case 116: return 5
  case 84: return 5
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[5] = true
fun[5] = func(r rune) int {
  switch(r) {
  case 108: return -1
  case 76: return -1
  case 105: return -1
  case 73: return -1
  case 109: return -1
  case 77: return -1
  case 116: return -1
  case 84: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[106].acc = acc[:]
a0[106].f = fun[:]
a0[106].id = 106
}
{
var acc [4]bool
var fun [4]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 108: return 1
  case 76: return 1
  case 115: return -1
  case 83: return -1
  case 109: return -1
  case 77: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 108: return -1
  case 76: return -1
  case 115: return 2
  case 83: return 2
  case 109: return -1
  case 77: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 108: return -1
  case 76: return -1
  case 115: return -1
  case 83: return -1
  case 109: return 3
  case 77: return 3
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[3] = true
fun[3] = func(r rune) int {
  switch(r) {
  case 108: return -1
  case 76: return -1
  case 115: return -1
  case 83: return -1
  case 109: return -1
  case 77: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[107].acc = acc[:]
a0[107].f = fun[:]
a0[107].id = 107
}
{
var acc [4]bool
var fun [4]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 109: return 1
  case 77: return 1
  case 97: return -1
  case 65: return -1
  case 112: return -1
  case 80: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 109: return -1
  case 77: return -1
  case 97: return 2
  case 65: return 2
  case 112: return -1
  case 80: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 109: return -1
  case 77: return -1
  case 97: return -1
  case 65: return -1
  case 112: return 3
  case 80: return 3
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[3] = true
fun[3] = func(r rune) int {
  switch(r) {
  case 109: return -1
  case 77: return -1
  case 97: return -1
  case 65: return -1
  case 112: return -1
  case 80: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[108].acc = acc[:]
a0[108].f = fun[:]
a0[108].id = 108
}
{
var acc [8]bool
var fun [8]func(rune) int
fun[2] = func(r rune) int {
  switch(r) {
  case 77: return -1
  case 80: return 3
  case 105: return -1
  case 110: return -1
  case 78: return -1
  case 103: return -1
  case 71: return -1
  case 109: return -1
  case 97: return -1
  case 65: return -1
  case 112: return 3
  case 73: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 109: return -1
  case 97: return -1
  case 65: return -1
  case 112: return 4
  case 73: return -1
  case 77: return -1
  case 80: return 4
  case 105: return -1
  case 110: return -1
  case 78: return -1
  case 103: return -1
  case 71: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[0] = func(r rune) int {
  switch(r) {
  case 77: return 1
  case 80: return -1
  case 105: return -1
  case 110: return -1
  case 78: return -1
  case 103: return -1
  case 71: return -1
  case 109: return 1
  case 97: return -1
  case 65: return -1
  case 112: return -1
  case 73: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 109: return -1
  case 97: return 2
  case 65: return 2
  case 112: return -1
  case 73: return -1
  case 77: return -1
  case 80: return -1
  case 105: return -1
  case 110: return -1
  case 78: return -1
  case 103: return -1
  case 71: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[4] = func(r rune) int {
  switch(r) {
  case 77: return -1
  case 80: return -1
  case 105: return 5
  case 110: return -1
  case 78: return -1
  case 103: return -1
  case 71: return -1
  case 109: return -1
  case 97: return -1
  case 65: return -1
  case 112: return -1
  case 73: return 5
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[5] = func(r rune) int {
  switch(r) {
  case 109: return -1
  case 97: return -1
  case 65: return -1
  case 112: return -1
  case 73: return -1
  case 77: return -1
  case 80: return -1
  case 105: return -1
  case 110: return 6
  case 78: return 6
  case 103: return -1
  case 71: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[6] = func(r rune) int {
  switch(r) {
  case 77: return -1
  case 80: return -1
  case 105: return -1
  case 110: return -1
  case 78: return -1
  case 103: return 7
  case 71: return 7
  case 109: return -1
  case 97: return -1
  case 65: return -1
  case 112: return -1
  case 73: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[7] = true
fun[7] = func(r rune) int {
  switch(r) {
  case 77: return -1
  case 80: return -1
  case 105: return -1
  case 110: return -1
  case 78: return -1
  case 103: return -1
  case 71: return -1
  case 109: return -1
  case 97: return -1
  case 65: return -1
  case 112: return -1
  case 73: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[109].acc = acc[:]
a0[109].f = fun[:]
a0[109].id = 109
}
{
var acc [8]bool
var fun [8]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 109: return 1
  case 116: return -1
  case 77: return 1
  case 101: return -1
  case 97: return -1
  case 65: return -1
  case 84: return -1
  case 72: return -1
  case 99: return -1
  case 67: return -1
  case 104: return -1
  case 69: return -1
  case 100: return -1
  case 68: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 97: return 2
  case 65: return 2
  case 84: return -1
  case 72: return -1
  case 99: return -1
  case 67: return -1
  case 104: return -1
  case 69: return -1
  case 100: return -1
  case 68: return -1
  case 109: return -1
  case 116: return -1
  case 77: return -1
  case 101: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[4] = func(r rune) int {
  switch(r) {
  case 99: return -1
  case 67: return -1
  case 104: return 5
  case 69: return -1
  case 100: return -1
  case 68: return -1
  case 109: return -1
  case 116: return -1
  case 77: return -1
  case 101: return -1
  case 97: return -1
  case 65: return -1
  case 84: return -1
  case 72: return 5
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[5] = func(r rune) int {
  switch(r) {
  case 77: return -1
  case 101: return 6
  case 97: return -1
  case 65: return -1
  case 84: return -1
  case 72: return -1
  case 99: return -1
  case 67: return -1
  case 104: return -1
  case 69: return 6
  case 100: return -1
  case 68: return -1
  case 109: return -1
  case 116: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[6] = func(r rune) int {
  switch(r) {
  case 77: return -1
  case 101: return -1
  case 97: return -1
  case 65: return -1
  case 84: return -1
  case 72: return -1
  case 99: return -1
  case 67: return -1
  case 104: return -1
  case 69: return -1
  case 100: return 7
  case 68: return 7
  case 109: return -1
  case 116: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 99: return -1
  case 67: return -1
  case 104: return -1
  case 69: return -1
  case 100: return -1
  case 68: return -1
  case 109: return -1
  case 116: return 3
  case 77: return -1
  case 101: return -1
  case 97: return -1
  case 65: return -1
  case 84: return 3
  case 72: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 77: return -1
  case 101: return -1
  case 97: return -1
  case 65: return -1
  case 84: return -1
  case 72: return -1
  case 99: return 4
  case 67: return 4
  case 104: return -1
  case 69: return -1
  case 100: return -1
  case 68: return -1
  case 109: return -1
  case 116: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[7] = true
fun[7] = func(r rune) int {
  switch(r) {
  case 77: return -1
  case 101: return -1
  case 97: return -1
  case 65: return -1
  case 84: return -1
  case 72: return -1
  case 99: return -1
  case 67: return -1
  case 104: return -1
  case 69: return -1
  case 100: return -1
  case 68: return -1
  case 109: return -1
  case 116: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[110].acc = acc[:]
a0[110].f = fun[:]
a0[110].id = 110
}
{
var acc [13]bool
var fun [13]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 84: return -1
  case 122: return -1
  case 77: return 1
  case 65: return -1
  case 114: return -1
  case 82: return -1
  case 100: return -1
  case 68: return -1
  case 116: return -1
  case 101: return -1
  case 69: return -1
  case 73: return -1
  case 76: return -1
  case 90: return -1
  case 109: return 1
  case 97: return -1
  case 105: return -1
  case 108: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[8] = func(r rune) int {
  switch(r) {
  case 77: return -1
  case 65: return -1
  case 114: return -1
  case 82: return -1
  case 100: return -1
  case 68: return -1
  case 116: return -1
  case 101: return -1
  case 69: return -1
  case 73: return 9
  case 76: return -1
  case 90: return -1
  case 109: return -1
  case 97: return -1
  case 105: return 9
  case 108: return -1
  case 84: return -1
  case 122: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[10] = func(r rune) int {
  switch(r) {
  case 77: return -1
  case 65: return -1
  case 114: return -1
  case 82: return -1
  case 100: return -1
  case 68: return -1
  case 116: return -1
  case 101: return 11
  case 69: return 11
  case 73: return -1
  case 76: return -1
  case 90: return -1
  case 109: return -1
  case 97: return -1
  case 105: return -1
  case 108: return -1
  case 84: return -1
  case 122: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[11] = func(r rune) int {
  switch(r) {
  case 84: return -1
  case 122: return -1
  case 77: return -1
  case 65: return -1
  case 114: return -1
  case 82: return -1
  case 100: return 12
  case 68: return 12
  case 116: return -1
  case 101: return -1
  case 69: return -1
  case 73: return -1
  case 76: return -1
  case 90: return -1
  case 109: return -1
  case 97: return -1
  case 105: return -1
  case 108: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 109: return -1
  case 97: return -1
  case 105: return -1
  case 108: return -1
  case 84: return 3
  case 122: return -1
  case 77: return -1
  case 65: return -1
  case 114: return -1
  case 82: return -1
  case 100: return -1
  case 68: return -1
  case 116: return 3
  case 101: return -1
  case 69: return -1
  case 73: return -1
  case 76: return -1
  case 90: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[5] = func(r rune) int {
  switch(r) {
  case 84: return -1
  case 122: return -1
  case 77: return -1
  case 65: return -1
  case 114: return -1
  case 82: return -1
  case 100: return -1
  case 68: return -1
  case 116: return -1
  case 101: return -1
  case 69: return -1
  case 73: return 6
  case 76: return -1
  case 90: return -1
  case 109: return -1
  case 97: return -1
  case 105: return 6
  case 108: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[6] = func(r rune) int {
  switch(r) {
  case 116: return -1
  case 101: return -1
  case 69: return -1
  case 73: return -1
  case 76: return -1
  case 90: return -1
  case 109: return -1
  case 97: return 7
  case 105: return -1
  case 108: return -1
  case 84: return -1
  case 122: return -1
  case 77: return -1
  case 65: return 7
  case 114: return -1
  case 82: return -1
  case 100: return -1
  case 68: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[7] = func(r rune) int {
  switch(r) {
  case 109: return -1
  case 97: return -1
  case 105: return -1
  case 108: return 8
  case 84: return -1
  case 122: return -1
  case 77: return -1
  case 65: return -1
  case 114: return -1
  case 82: return -1
  case 100: return -1
  case 68: return -1
  case 116: return -1
  case 101: return -1
  case 69: return -1
  case 73: return -1
  case 76: return 8
  case 90: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[9] = func(r rune) int {
  switch(r) {
  case 77: return -1
  case 65: return -1
  case 114: return -1
  case 82: return -1
  case 100: return -1
  case 68: return -1
  case 116: return -1
  case 101: return -1
  case 69: return -1
  case 73: return -1
  case 76: return -1
  case 90: return 10
  case 109: return -1
  case 97: return -1
  case 105: return -1
  case 108: return -1
  case 84: return -1
  case 122: return 10
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[12] = true
fun[12] = func(r rune) int {
  switch(r) {
  case 116: return -1
  case 101: return -1
  case 69: return -1
  case 73: return -1
  case 76: return -1
  case 90: return -1
  case 109: return -1
  case 97: return -1
  case 105: return -1
  case 108: return -1
  case 84: return -1
  case 122: return -1
  case 77: return -1
  case 65: return -1
  case 114: return -1
  case 82: return -1
  case 100: return -1
  case 68: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 84: return -1
  case 122: return -1
  case 77: return -1
  case 65: return -1
  case 114: return -1
  case 82: return -1
  case 100: return -1
  case 68: return -1
  case 116: return -1
  case 101: return 4
  case 69: return 4
  case 73: return -1
  case 76: return -1
  case 90: return -1
  case 109: return -1
  case 97: return -1
  case 105: return -1
  case 108: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 116: return -1
  case 101: return -1
  case 69: return -1
  case 73: return -1
  case 76: return -1
  case 90: return -1
  case 109: return -1
  case 97: return 2
  case 105: return -1
  case 108: return -1
  case 84: return -1
  case 122: return -1
  case 77: return -1
  case 65: return 2
  case 114: return -1
  case 82: return -1
  case 100: return -1
  case 68: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[4] = func(r rune) int {
  switch(r) {
  case 109: return -1
  case 97: return -1
  case 105: return -1
  case 108: return -1
  case 84: return -1
  case 122: return -1
  case 77: return -1
  case 65: return -1
  case 114: return 5
  case 82: return 5
  case 100: return -1
  case 68: return -1
  case 116: return -1
  case 101: return -1
  case 69: return -1
  case 73: return -1
  case 76: return -1
  case 90: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[111].acc = acc[:]
a0[111].f = fun[:]
a0[111].id = 111
}
{
var acc [6]bool
var fun [6]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 109: return 1
  case 77: return 1
  case 101: return -1
  case 69: return -1
  case 114: return -1
  case 82: return -1
  case 103: return -1
  case 71: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 109: return -1
  case 77: return -1
  case 101: return 2
  case 69: return 2
  case 114: return -1
  case 82: return -1
  case 103: return -1
  case 71: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 109: return -1
  case 77: return -1
  case 101: return -1
  case 69: return -1
  case 114: return 3
  case 82: return 3
  case 103: return -1
  case 71: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 109: return -1
  case 77: return -1
  case 101: return -1
  case 69: return -1
  case 114: return -1
  case 82: return -1
  case 103: return 4
  case 71: return 4
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[4] = func(r rune) int {
  switch(r) {
  case 109: return -1
  case 77: return -1
  case 101: return 5
  case 69: return 5
  case 114: return -1
  case 82: return -1
  case 103: return -1
  case 71: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[5] = true
fun[5] = func(r rune) int {
  switch(r) {
  case 109: return -1
  case 77: return -1
  case 101: return -1
  case 69: return -1
  case 114: return -1
  case 82: return -1
  case 103: return -1
  case 71: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[112].acc = acc[:]
a0[112].f = fun[:]
a0[112].id = 112
}
{
var acc [6]bool
var fun [6]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 77: return 1
  case 110: return -1
  case 85: return -1
  case 115: return -1
  case 83: return -1
  case 109: return 1
  case 105: return -1
  case 73: return -1
  case 78: return -1
  case 117: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 77: return -1
  case 110: return -1
  case 85: return -1
  case 115: return -1
  case 83: return -1
  case 109: return -1
  case 105: return 2
  case 73: return 2
  case 78: return -1
  case 117: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 109: return -1
  case 105: return -1
  case 73: return -1
  case 78: return 3
  case 117: return -1
  case 77: return -1
  case 110: return 3
  case 85: return -1
  case 115: return -1
  case 83: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 109: return -1
  case 105: return -1
  case 73: return -1
  case 78: return -1
  case 117: return 4
  case 77: return -1
  case 110: return -1
  case 85: return 4
  case 115: return -1
  case 83: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[4] = func(r rune) int {
  switch(r) {
  case 109: return -1
  case 105: return -1
  case 73: return -1
  case 78: return -1
  case 117: return -1
  case 77: return -1
  case 110: return -1
  case 85: return -1
  case 115: return 5
  case 83: return 5
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[5] = true
fun[5] = func(r rune) int {
  switch(r) {
  case 109: return -1
  case 105: return -1
  case 73: return -1
  case 78: return -1
  case 117: return -1
  case 77: return -1
  case 110: return -1
  case 85: return -1
  case 115: return -1
  case 83: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[113].acc = acc[:]
a0[113].f = fun[:]
a0[113].id = 113
}
{
var acc [8]bool
var fun [8]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 109: return 1
  case 77: return 1
  case 73: return -1
  case 115: return -1
  case 71: return -1
  case 105: return -1
  case 83: return -1
  case 110: return -1
  case 78: return -1
  case 103: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 105: return 2
  case 83: return -1
  case 110: return -1
  case 78: return -1
  case 103: return -1
  case 109: return -1
  case 77: return -1
  case 73: return 2
  case 115: return -1
  case 71: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 105: return -1
  case 83: return 3
  case 110: return -1
  case 78: return -1
  case 103: return -1
  case 109: return -1
  case 77: return -1
  case 73: return -1
  case 115: return 3
  case 71: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 109: return -1
  case 77: return -1
  case 73: return -1
  case 115: return 4
  case 71: return -1
  case 105: return -1
  case 83: return 4
  case 110: return -1
  case 78: return -1
  case 103: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[4] = func(r rune) int {
  switch(r) {
  case 109: return -1
  case 77: return -1
  case 73: return 5
  case 115: return -1
  case 71: return -1
  case 105: return 5
  case 83: return -1
  case 110: return -1
  case 78: return -1
  case 103: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[7] = true
fun[7] = func(r rune) int {
  switch(r) {
  case 109: return -1
  case 77: return -1
  case 73: return -1
  case 115: return -1
  case 71: return -1
  case 105: return -1
  case 83: return -1
  case 110: return -1
  case 78: return -1
  case 103: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[5] = func(r rune) int {
  switch(r) {
  case 105: return -1
  case 83: return -1
  case 110: return 6
  case 78: return 6
  case 103: return -1
  case 109: return -1
  case 77: return -1
  case 73: return -1
  case 115: return -1
  case 71: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[6] = func(r rune) int {
  switch(r) {
  case 109: return -1
  case 77: return -1
  case 73: return -1
  case 115: return -1
  case 71: return 7
  case 105: return -1
  case 83: return -1
  case 110: return -1
  case 78: return -1
  case 103: return 7
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[114].acc = acc[:]
a0[114].f = fun[:]
a0[114].id = 114
}
{
var acc [10]bool
var fun [10]func(rune) int
fun[2] = func(r rune) int {
  switch(r) {
  case 110: return -1
  case 97: return -1
  case 69: return -1
  case 80: return -1
  case 99: return -1
  case 77: return 3
  case 78: return -1
  case 83: return -1
  case 112: return -1
  case 67: return -1
  case 65: return -1
  case 109: return 3
  case 101: return -1
  case 115: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[6] = func(r rune) int {
  switch(r) {
  case 77: return -1
  case 78: return -1
  case 83: return -1
  case 112: return -1
  case 67: return -1
  case 65: return 7
  case 109: return -1
  case 101: return -1
  case 115: return -1
  case 110: return -1
  case 97: return 7
  case 69: return -1
  case 80: return -1
  case 99: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[7] = func(r rune) int {
  switch(r) {
  case 65: return -1
  case 109: return -1
  case 101: return -1
  case 115: return -1
  case 110: return -1
  case 97: return -1
  case 69: return -1
  case 80: return -1
  case 99: return 8
  case 77: return -1
  case 78: return -1
  case 83: return -1
  case 112: return -1
  case 67: return 8
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[8] = func(r rune) int {
  switch(r) {
  case 65: return -1
  case 109: return -1
  case 101: return 9
  case 115: return -1
  case 110: return -1
  case 97: return -1
  case 69: return 9
  case 80: return -1
  case 99: return -1
  case 77: return -1
  case 78: return -1
  case 83: return -1
  case 112: return -1
  case 67: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[9] = true
fun[9] = func(r rune) int {
  switch(r) {
  case 65: return -1
  case 109: return -1
  case 101: return -1
  case 115: return -1
  case 110: return -1
  case 97: return -1
  case 69: return -1
  case 80: return -1
  case 99: return -1
  case 77: return -1
  case 78: return -1
  case 83: return -1
  case 112: return -1
  case 67: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[0] = func(r rune) int {
  switch(r) {
  case 110: return 1
  case 97: return -1
  case 69: return -1
  case 80: return -1
  case 99: return -1
  case 77: return -1
  case 78: return 1
  case 83: return -1
  case 112: return -1
  case 67: return -1
  case 65: return -1
  case 109: return -1
  case 101: return -1
  case 115: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 77: return -1
  case 78: return -1
  case 83: return -1
  case 112: return -1
  case 67: return -1
  case 65: return 2
  case 109: return -1
  case 101: return -1
  case 115: return -1
  case 110: return -1
  case 97: return 2
  case 69: return -1
  case 80: return -1
  case 99: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 77: return -1
  case 78: return -1
  case 83: return -1
  case 112: return -1
  case 67: return -1
  case 65: return -1
  case 109: return -1
  case 101: return 4
  case 115: return -1
  case 110: return -1
  case 97: return -1
  case 69: return 4
  case 80: return -1
  case 99: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[4] = func(r rune) int {
  switch(r) {
  case 110: return -1
  case 97: return -1
  case 69: return -1
  case 80: return -1
  case 99: return -1
  case 77: return -1
  case 78: return -1
  case 83: return 5
  case 112: return -1
  case 67: return -1
  case 65: return -1
  case 109: return -1
  case 101: return -1
  case 115: return 5
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[5] = func(r rune) int {
  switch(r) {
  case 78: return -1
  case 83: return -1
  case 112: return 6
  case 67: return -1
  case 65: return -1
  case 109: return -1
  case 101: return -1
  case 115: return -1
  case 110: return -1
  case 97: return -1
  case 69: return -1
  case 80: return 6
  case 99: return -1
  case 77: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[115].acc = acc[:]
a0[115].f = fun[:]
a0[115].id = 115
}
{
var acc [5]bool
var fun [5]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 110: return 1
  case 78: return 1
  case 101: return -1
  case 69: return -1
  case 115: return -1
  case 83: return -1
  case 116: return -1
  case 84: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 110: return -1
  case 78: return -1
  case 101: return 2
  case 69: return 2
  case 115: return -1
  case 83: return -1
  case 116: return -1
  case 84: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 110: return -1
  case 78: return -1
  case 101: return -1
  case 69: return -1
  case 115: return 3
  case 83: return 3
  case 116: return -1
  case 84: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 110: return -1
  case 78: return -1
  case 101: return -1
  case 69: return -1
  case 115: return -1
  case 83: return -1
  case 116: return 4
  case 84: return 4
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[4] = true
fun[4] = func(r rune) int {
  switch(r) {
  case 110: return -1
  case 78: return -1
  case 101: return -1
  case 69: return -1
  case 115: return -1
  case 83: return -1
  case 116: return -1
  case 84: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[116].acc = acc[:]
a0[116].f = fun[:]
a0[116].id = 116
}
{
var acc [4]bool
var fun [4]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 110: return 1
  case 78: return 1
  case 111: return -1
  case 79: return -1
  case 116: return -1
  case 84: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 110: return -1
  case 78: return -1
  case 111: return 2
  case 79: return 2
  case 116: return -1
  case 84: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 110: return -1
  case 78: return -1
  case 111: return -1
  case 79: return -1
  case 116: return 3
  case 84: return 3
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[3] = true
fun[3] = func(r rune) int {
  switch(r) {
  case 110: return -1
  case 78: return -1
  case 111: return -1
  case 79: return -1
  case 116: return -1
  case 84: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[117].acc = acc[:]
a0[117].f = fun[:]
a0[117].id = 117
}
{
var acc [5]bool
var fun [5]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 110: return 1
  case 78: return 1
  case 117: return -1
  case 85: return -1
  case 108: return -1
  case 76: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 110: return -1
  case 78: return -1
  case 117: return 2
  case 85: return 2
  case 108: return -1
  case 76: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 110: return -1
  case 78: return -1
  case 117: return -1
  case 85: return -1
  case 108: return 3
  case 76: return 3
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 110: return -1
  case 78: return -1
  case 117: return -1
  case 85: return -1
  case 108: return 4
  case 76: return 4
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[4] = true
fun[4] = func(r rune) int {
  switch(r) {
  case 110: return -1
  case 78: return -1
  case 117: return -1
  case 85: return -1
  case 108: return -1
  case 76: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[118].acc = acc[:]
a0[118].f = fun[:]
a0[118].id = 118
}
{
var acc [7]bool
var fun [7]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 111: return 1
  case 79: return 1
  case 106: return -1
  case 101: return -1
  case 69: return -1
  case 99: return -1
  case 98: return -1
  case 66: return -1
  case 74: return -1
  case 67: return -1
  case 116: return -1
  case 84: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 111: return -1
  case 79: return -1
  case 106: return -1
  case 101: return -1
  case 69: return -1
  case 99: return -1
  case 98: return 2
  case 66: return 2
  case 74: return -1
  case 67: return -1
  case 116: return -1
  case 84: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 98: return -1
  case 66: return -1
  case 74: return 3
  case 67: return -1
  case 116: return -1
  case 84: return -1
  case 111: return -1
  case 79: return -1
  case 106: return 3
  case 101: return -1
  case 69: return -1
  case 99: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 98: return -1
  case 66: return -1
  case 74: return -1
  case 67: return -1
  case 116: return -1
  case 84: return -1
  case 111: return -1
  case 79: return -1
  case 106: return -1
  case 101: return 4
  case 69: return 4
  case 99: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[4] = func(r rune) int {
  switch(r) {
  case 111: return -1
  case 79: return -1
  case 106: return -1
  case 101: return -1
  case 69: return -1
  case 99: return 5
  case 98: return -1
  case 66: return -1
  case 74: return -1
  case 67: return 5
  case 116: return -1
  case 84: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[5] = func(r rune) int {
  switch(r) {
  case 111: return -1
  case 79: return -1
  case 106: return -1
  case 101: return -1
  case 69: return -1
  case 99: return -1
  case 98: return -1
  case 66: return -1
  case 74: return -1
  case 67: return -1
  case 116: return 6
  case 84: return 6
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[6] = true
fun[6] = func(r rune) int {
  switch(r) {
  case 111: return -1
  case 79: return -1
  case 106: return -1
  case 101: return -1
  case 69: return -1
  case 99: return -1
  case 98: return -1
  case 66: return -1
  case 74: return -1
  case 67: return -1
  case 116: return -1
  case 84: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[119].acc = acc[:]
a0[119].f = fun[:]
a0[119].id = 119
}
{
var acc [7]bool
var fun [7]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 111: return 1
  case 115: return -1
  case 69: return -1
  case 84: return -1
  case 79: return 1
  case 102: return -1
  case 70: return -1
  case 83: return -1
  case 101: return -1
  case 116: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 111: return -1
  case 115: return -1
  case 69: return -1
  case 84: return -1
  case 79: return -1
  case 102: return 2
  case 70: return 2
  case 83: return -1
  case 101: return -1
  case 116: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 79: return -1
  case 102: return 3
  case 70: return 3
  case 83: return -1
  case 101: return -1
  case 116: return -1
  case 111: return -1
  case 115: return -1
  case 69: return -1
  case 84: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 111: return -1
  case 115: return 4
  case 69: return -1
  case 84: return -1
  case 79: return -1
  case 102: return -1
  case 70: return -1
  case 83: return 4
  case 101: return -1
  case 116: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[4] = func(r rune) int {
  switch(r) {
  case 79: return -1
  case 102: return -1
  case 70: return -1
  case 83: return -1
  case 101: return 5
  case 116: return -1
  case 111: return -1
  case 115: return -1
  case 69: return 5
  case 84: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[5] = func(r rune) int {
  switch(r) {
  case 79: return -1
  case 102: return -1
  case 70: return -1
  case 83: return -1
  case 101: return -1
  case 116: return 6
  case 111: return -1
  case 115: return -1
  case 69: return -1
  case 84: return 6
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[6] = true
fun[6] = func(r rune) int {
  switch(r) {
  case 79: return -1
  case 102: return -1
  case 70: return -1
  case 83: return -1
  case 101: return -1
  case 116: return -1
  case 111: return -1
  case 115: return -1
  case 69: return -1
  case 84: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[120].acc = acc[:]
a0[120].f = fun[:]
a0[120].id = 120
}
{
var acc [3]bool
var fun [3]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 111: return 1
  case 79: return 1
  case 110: return -1
  case 78: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 111: return -1
  case 79: return -1
  case 110: return 2
  case 78: return 2
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[2] = true
fun[2] = func(r rune) int {
  switch(r) {
  case 111: return -1
  case 79: return -1
  case 110: return -1
  case 78: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[121].acc = acc[:]
a0[121].f = fun[:]
a0[121].id = 121
}
{
var acc [7]bool
var fun [7]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 112: return -1
  case 105: return -1
  case 73: return -1
  case 111: return 1
  case 79: return 1
  case 80: return -1
  case 116: return -1
  case 84: return -1
  case 110: return -1
  case 78: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 111: return -1
  case 79: return -1
  case 80: return 2
  case 116: return -1
  case 84: return -1
  case 110: return -1
  case 78: return -1
  case 112: return 2
  case 105: return -1
  case 73: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 111: return -1
  case 79: return -1
  case 80: return -1
  case 116: return 3
  case 84: return 3
  case 110: return -1
  case 78: return -1
  case 112: return -1
  case 105: return -1
  case 73: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 111: return -1
  case 79: return -1
  case 80: return -1
  case 116: return -1
  case 84: return -1
  case 110: return -1
  case 78: return -1
  case 112: return -1
  case 105: return 4
  case 73: return 4
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[4] = func(r rune) int {
  switch(r) {
  case 111: return 5
  case 79: return 5
  case 80: return -1
  case 116: return -1
  case 84: return -1
  case 110: return -1
  case 78: return -1
  case 112: return -1
  case 105: return -1
  case 73: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[5] = func(r rune) int {
  switch(r) {
  case 112: return -1
  case 105: return -1
  case 73: return -1
  case 111: return -1
  case 79: return -1
  case 80: return -1
  case 116: return -1
  case 84: return -1
  case 110: return 6
  case 78: return 6
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[6] = true
fun[6] = func(r rune) int {
  switch(r) {
  case 111: return -1
  case 79: return -1
  case 80: return -1
  case 116: return -1
  case 84: return -1
  case 110: return -1
  case 78: return -1
  case 112: return -1
  case 105: return -1
  case 73: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[122].acc = acc[:]
a0[122].f = fun[:]
a0[122].id = 122
}
{
var acc [3]bool
var fun [3]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 111: return 1
  case 79: return 1
  case 114: return -1
  case 82: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 111: return -1
  case 79: return -1
  case 114: return 2
  case 82: return 2
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[2] = true
fun[2] = func(r rune) int {
  switch(r) {
  case 111: return -1
  case 79: return -1
  case 114: return -1
  case 82: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[123].acc = acc[:]
a0[123].f = fun[:]
a0[123].id = 123
}
{
var acc [6]bool
var fun [6]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 111: return 1
  case 79: return 1
  case 114: return -1
  case 82: return -1
  case 100: return -1
  case 68: return -1
  case 101: return -1
  case 69: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 111: return -1
  case 79: return -1
  case 114: return 2
  case 82: return 2
  case 100: return -1
  case 68: return -1
  case 101: return -1
  case 69: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 111: return -1
  case 79: return -1
  case 114: return -1
  case 82: return -1
  case 100: return 3
  case 68: return 3
  case 101: return -1
  case 69: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 111: return -1
  case 79: return -1
  case 114: return -1
  case 82: return -1
  case 100: return -1
  case 68: return -1
  case 101: return 4
  case 69: return 4
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[4] = func(r rune) int {
  switch(r) {
  case 111: return -1
  case 79: return -1
  case 114: return 5
  case 82: return 5
  case 100: return -1
  case 68: return -1
  case 101: return -1
  case 69: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[5] = true
fun[5] = func(r rune) int {
  switch(r) {
  case 111: return -1
  case 79: return -1
  case 114: return -1
  case 82: return -1
  case 100: return -1
  case 68: return -1
  case 101: return -1
  case 69: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[124].acc = acc[:]
a0[124].f = fun[:]
a0[124].id = 124
}
{
var acc [6]bool
var fun [6]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 84: return -1
  case 114: return -1
  case 82: return -1
  case 111: return 1
  case 79: return 1
  case 117: return -1
  case 85: return -1
  case 116: return -1
  case 101: return -1
  case 69: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 84: return -1
  case 114: return -1
  case 82: return -1
  case 111: return -1
  case 79: return -1
  case 117: return 2
  case 85: return 2
  case 116: return -1
  case 101: return -1
  case 69: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 111: return -1
  case 79: return -1
  case 117: return -1
  case 85: return -1
  case 116: return 3
  case 101: return -1
  case 69: return -1
  case 84: return 3
  case 114: return -1
  case 82: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 111: return -1
  case 79: return -1
  case 117: return -1
  case 85: return -1
  case 116: return -1
  case 101: return 4
  case 69: return 4
  case 84: return -1
  case 114: return -1
  case 82: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[4] = func(r rune) int {
  switch(r) {
  case 84: return -1
  case 114: return 5
  case 82: return 5
  case 111: return -1
  case 79: return -1
  case 117: return -1
  case 85: return -1
  case 116: return -1
  case 101: return -1
  case 69: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[5] = true
fun[5] = func(r rune) int {
  switch(r) {
  case 84: return -1
  case 114: return -1
  case 82: return -1
  case 111: return -1
  case 79: return -1
  case 117: return -1
  case 85: return -1
  case 116: return -1
  case 101: return -1
  case 69: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[125].acc = acc[:]
a0[125].f = fun[:]
a0[125].id = 125
}
{
var acc [5]bool
var fun [5]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 111: return 1
  case 79: return 1
  case 118: return -1
  case 86: return -1
  case 101: return -1
  case 69: return -1
  case 114: return -1
  case 82: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 111: return -1
  case 79: return -1
  case 118: return 2
  case 86: return 2
  case 101: return -1
  case 69: return -1
  case 114: return -1
  case 82: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 111: return -1
  case 79: return -1
  case 118: return -1
  case 86: return -1
  case 101: return 3
  case 69: return 3
  case 114: return -1
  case 82: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 111: return -1
  case 79: return -1
  case 118: return -1
  case 86: return -1
  case 101: return -1
  case 69: return -1
  case 114: return 4
  case 82: return 4
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[4] = true
fun[4] = func(r rune) int {
  switch(r) {
  case 111: return -1
  case 79: return -1
  case 118: return -1
  case 86: return -1
  case 101: return -1
  case 69: return -1
  case 114: return -1
  case 82: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[126].acc = acc[:]
a0[126].f = fun[:]
a0[126].id = 126
}
{
var acc [10]bool
var fun [10]func(rune) int
fun[1] = func(r rune) int {
  switch(r) {
  case 73: return -1
  case 79: return -1
  case 112: return -1
  case 80: return -1
  case 97: return 2
  case 116: return -1
  case 105: return -1
  case 111: return -1
  case 78: return -1
  case 82: return -1
  case 84: return -1
  case 110: return -1
  case 65: return 2
  case 114: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 112: return -1
  case 80: return -1
  case 97: return -1
  case 116: return -1
  case 105: return -1
  case 111: return -1
  case 78: return -1
  case 82: return 3
  case 84: return -1
  case 110: return -1
  case 65: return -1
  case 114: return 3
  case 73: return -1
  case 79: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 73: return -1
  case 79: return -1
  case 112: return -1
  case 80: return -1
  case 97: return -1
  case 116: return 4
  case 105: return -1
  case 111: return -1
  case 78: return -1
  case 82: return -1
  case 84: return 4
  case 110: return -1
  case 65: return -1
  case 114: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[6] = func(r rune) int {
  switch(r) {
  case 82: return -1
  case 84: return -1
  case 110: return -1
  case 65: return -1
  case 114: return -1
  case 73: return 7
  case 79: return -1
  case 112: return -1
  case 80: return -1
  case 97: return -1
  case 116: return -1
  case 105: return 7
  case 111: return -1
  case 78: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[7] = func(r rune) int {
  switch(r) {
  case 82: return -1
  case 84: return -1
  case 110: return -1
  case 65: return -1
  case 114: return -1
  case 73: return -1
  case 79: return 8
  case 112: return -1
  case 80: return -1
  case 97: return -1
  case 116: return -1
  case 105: return -1
  case 111: return 8
  case 78: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[8] = func(r rune) int {
  switch(r) {
  case 112: return -1
  case 80: return -1
  case 97: return -1
  case 116: return -1
  case 105: return -1
  case 111: return -1
  case 78: return 9
  case 82: return -1
  case 84: return -1
  case 110: return 9
  case 65: return -1
  case 114: return -1
  case 73: return -1
  case 79: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[0] = func(r rune) int {
  switch(r) {
  case 65: return -1
  case 114: return -1
  case 73: return -1
  case 79: return -1
  case 112: return 1
  case 80: return 1
  case 97: return -1
  case 116: return -1
  case 105: return -1
  case 111: return -1
  case 78: return -1
  case 82: return -1
  case 84: return -1
  case 110: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[4] = func(r rune) int {
  switch(r) {
  case 112: return -1
  case 80: return -1
  case 97: return -1
  case 116: return -1
  case 105: return 5
  case 111: return -1
  case 78: return -1
  case 82: return -1
  case 84: return -1
  case 110: return -1
  case 65: return -1
  case 114: return -1
  case 73: return 5
  case 79: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[5] = func(r rune) int {
  switch(r) {
  case 73: return -1
  case 79: return -1
  case 112: return -1
  case 80: return -1
  case 97: return -1
  case 116: return 6
  case 105: return -1
  case 111: return -1
  case 78: return -1
  case 82: return -1
  case 84: return 6
  case 110: return -1
  case 65: return -1
  case 114: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[9] = true
fun[9] = func(r rune) int {
  switch(r) {
  case 65: return -1
  case 114: return -1
  case 73: return -1
  case 79: return -1
  case 112: return -1
  case 80: return -1
  case 97: return -1
  case 116: return -1
  case 105: return -1
  case 111: return -1
  case 78: return -1
  case 82: return -1
  case 84: return -1
  case 110: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[127].acc = acc[:]
a0[127].f = fun[:]
a0[127].id = 127
}
{
var acc [9]bool
var fun [9]func(rune) int
fun[1] = func(r rune) int {
  switch(r) {
  case 115: return -1
  case 119: return -1
  case 79: return -1
  case 114: return -1
  case 112: return -1
  case 97: return 2
  case 87: return -1
  case 82: return -1
  case 100: return -1
  case 68: return -1
  case 80: return -1
  case 65: return 2
  case 83: return -1
  case 111: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[7] = func(r rune) int {
  switch(r) {
  case 115: return -1
  case 119: return -1
  case 79: return -1
  case 114: return -1
  case 112: return -1
  case 97: return -1
  case 87: return -1
  case 82: return -1
  case 100: return 8
  case 68: return 8
  case 80: return -1
  case 65: return -1
  case 83: return -1
  case 111: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[8] = true
fun[8] = func(r rune) int {
  switch(r) {
  case 112: return -1
  case 97: return -1
  case 87: return -1
  case 82: return -1
  case 100: return -1
  case 68: return -1
  case 80: return -1
  case 65: return -1
  case 83: return -1
  case 111: return -1
  case 115: return -1
  case 119: return -1
  case 79: return -1
  case 114: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[0] = func(r rune) int {
  switch(r) {
  case 80: return 1
  case 65: return -1
  case 83: return -1
  case 111: return -1
  case 115: return -1
  case 119: return -1
  case 79: return -1
  case 114: return -1
  case 112: return 1
  case 97: return -1
  case 87: return -1
  case 82: return -1
  case 100: return -1
  case 68: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 100: return -1
  case 68: return -1
  case 80: return -1
  case 65: return -1
  case 83: return 3
  case 111: return -1
  case 115: return 3
  case 119: return -1
  case 79: return -1
  case 114: return -1
  case 112: return -1
  case 97: return -1
  case 87: return -1
  case 82: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 112: return -1
  case 97: return -1
  case 87: return -1
  case 82: return -1
  case 100: return -1
  case 68: return -1
  case 80: return -1
  case 65: return -1
  case 83: return 4
  case 111: return -1
  case 115: return 4
  case 119: return -1
  case 79: return -1
  case 114: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[4] = func(r rune) int {
  switch(r) {
  case 115: return -1
  case 119: return 5
  case 79: return -1
  case 114: return -1
  case 112: return -1
  case 97: return -1
  case 87: return 5
  case 82: return -1
  case 100: return -1
  case 68: return -1
  case 80: return -1
  case 65: return -1
  case 83: return -1
  case 111: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[5] = func(r rune) int {
  switch(r) {
  case 100: return -1
  case 68: return -1
  case 80: return -1
  case 65: return -1
  case 83: return -1
  case 111: return 6
  case 115: return -1
  case 119: return -1
  case 79: return 6
  case 114: return -1
  case 112: return -1
  case 97: return -1
  case 87: return -1
  case 82: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[6] = func(r rune) int {
  switch(r) {
  case 112: return -1
  case 97: return -1
  case 87: return -1
  case 82: return 7
  case 100: return -1
  case 68: return -1
  case 80: return -1
  case 65: return -1
  case 83: return -1
  case 111: return -1
  case 115: return -1
  case 119: return -1
  case 79: return -1
  case 114: return 7
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[128].acc = acc[:]
a0[128].f = fun[:]
a0[128].id = 128
}
{
var acc [5]bool
var fun [5]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 112: return 1
  case 80: return 1
  case 97: return -1
  case 65: return -1
  case 116: return -1
  case 84: return -1
  case 104: return -1
  case 72: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 112: return -1
  case 80: return -1
  case 97: return 2
  case 65: return 2
  case 116: return -1
  case 84: return -1
  case 104: return -1
  case 72: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 112: return -1
  case 80: return -1
  case 97: return -1
  case 65: return -1
  case 116: return 3
  case 84: return 3
  case 104: return -1
  case 72: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 112: return -1
  case 80: return -1
  case 97: return -1
  case 65: return -1
  case 116: return -1
  case 84: return -1
  case 104: return 4
  case 72: return 4
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[4] = true
fun[4] = func(r rune) int {
  switch(r) {
  case 112: return -1
  case 80: return -1
  case 97: return -1
  case 65: return -1
  case 116: return -1
  case 84: return -1
  case 104: return -1
  case 72: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[129].acc = acc[:]
a0[129].f = fun[:]
a0[129].id = 129
}
{
var acc [5]bool
var fun [5]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 112: return 1
  case 80: return 1
  case 111: return -1
  case 79: return -1
  case 108: return -1
  case 76: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 112: return -1
  case 80: return -1
  case 111: return 2
  case 79: return 2
  case 108: return -1
  case 76: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 112: return -1
  case 80: return -1
  case 111: return 3
  case 79: return 3
  case 108: return -1
  case 76: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 112: return -1
  case 80: return -1
  case 111: return -1
  case 79: return -1
  case 108: return 4
  case 76: return 4
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[4] = true
fun[4] = func(r rune) int {
  switch(r) {
  case 112: return -1
  case 80: return -1
  case 111: return -1
  case 79: return -1
  case 108: return -1
  case 76: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[130].acc = acc[:]
a0[130].f = fun[:]
a0[130].id = 130
}
{
var acc [8]bool
var fun [8]func(rune) int
fun[1] = func(r rune) int {
  switch(r) {
  case 112: return -1
  case 80: return -1
  case 114: return 2
  case 82: return 2
  case 101: return -1
  case 69: return -1
  case 97: return -1
  case 65: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 112: return 4
  case 80: return 4
  case 114: return -1
  case 82: return -1
  case 101: return -1
  case 69: return -1
  case 97: return -1
  case 65: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[4] = func(r rune) int {
  switch(r) {
  case 112: return -1
  case 80: return -1
  case 114: return -1
  case 82: return -1
  case 101: return -1
  case 69: return -1
  case 97: return 5
  case 65: return 5
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[5] = func(r rune) int {
  switch(r) {
  case 112: return -1
  case 80: return -1
  case 114: return 6
  case 82: return 6
  case 101: return -1
  case 69: return -1
  case 97: return -1
  case 65: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[6] = func(r rune) int {
  switch(r) {
  case 112: return -1
  case 80: return -1
  case 114: return -1
  case 82: return -1
  case 101: return 7
  case 69: return 7
  case 97: return -1
  case 65: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[0] = func(r rune) int {
  switch(r) {
  case 112: return 1
  case 80: return 1
  case 114: return -1
  case 82: return -1
  case 101: return -1
  case 69: return -1
  case 97: return -1
  case 65: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 112: return -1
  case 80: return -1
  case 114: return -1
  case 82: return -1
  case 101: return 3
  case 69: return 3
  case 97: return -1
  case 65: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[7] = true
fun[7] = func(r rune) int {
  switch(r) {
  case 112: return -1
  case 80: return -1
  case 114: return -1
  case 82: return -1
  case 101: return -1
  case 69: return -1
  case 97: return -1
  case 65: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[131].acc = acc[:]
a0[131].f = fun[:]
a0[131].id = 131
}
{
var acc [8]bool
var fun [8]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 105: return -1
  case 77: return -1
  case 65: return -1
  case 121: return -1
  case 112: return 1
  case 80: return 1
  case 114: return -1
  case 82: return -1
  case 73: return -1
  case 109: return -1
  case 97: return -1
  case 89: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 112: return -1
  case 80: return -1
  case 114: return 2
  case 82: return 2
  case 73: return -1
  case 109: return -1
  case 97: return -1
  case 89: return -1
  case 105: return -1
  case 77: return -1
  case 65: return -1
  case 121: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 112: return -1
  case 80: return -1
  case 114: return -1
  case 82: return -1
  case 73: return 3
  case 109: return -1
  case 97: return -1
  case 89: return -1
  case 105: return 3
  case 77: return -1
  case 65: return -1
  case 121: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[4] = func(r rune) int {
  switch(r) {
  case 105: return -1
  case 77: return -1
  case 65: return 5
  case 121: return -1
  case 112: return -1
  case 80: return -1
  case 114: return -1
  case 82: return -1
  case 73: return -1
  case 109: return -1
  case 97: return 5
  case 89: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[5] = func(r rune) int {
  switch(r) {
  case 105: return -1
  case 77: return -1
  case 65: return -1
  case 121: return -1
  case 112: return -1
  case 80: return -1
  case 114: return 6
  case 82: return 6
  case 73: return -1
  case 109: return -1
  case 97: return -1
  case 89: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[7] = true
fun[7] = func(r rune) int {
  switch(r) {
  case 105: return -1
  case 77: return -1
  case 65: return -1
  case 121: return -1
  case 112: return -1
  case 80: return -1
  case 114: return -1
  case 82: return -1
  case 73: return -1
  case 109: return -1
  case 97: return -1
  case 89: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 112: return -1
  case 80: return -1
  case 114: return -1
  case 82: return -1
  case 73: return -1
  case 109: return 4
  case 97: return -1
  case 89: return -1
  case 105: return -1
  case 77: return 4
  case 65: return -1
  case 121: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[6] = func(r rune) int {
  switch(r) {
  case 105: return -1
  case 77: return -1
  case 65: return -1
  case 121: return 7
  case 112: return -1
  case 80: return -1
  case 114: return -1
  case 82: return -1
  case 73: return -1
  case 109: return -1
  case 97: return -1
  case 89: return 7
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[132].acc = acc[:]
a0[132].f = fun[:]
a0[132].id = 132
}
{
var acc [8]bool
var fun [8]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 73: return -1
  case 65: return -1
  case 116: return -1
  case 69: return -1
  case 112: return 1
  case 80: return 1
  case 105: return -1
  case 118: return -1
  case 86: return -1
  case 101: return -1
  case 114: return -1
  case 82: return -1
  case 84: return -1
  case 97: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 112: return -1
  case 80: return -1
  case 105: return -1
  case 118: return -1
  case 86: return -1
  case 101: return -1
  case 114: return 2
  case 82: return 2
  case 84: return -1
  case 97: return -1
  case 73: return -1
  case 65: return -1
  case 116: return -1
  case 69: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 73: return 3
  case 65: return -1
  case 116: return -1
  case 69: return -1
  case 112: return -1
  case 80: return -1
  case 105: return 3
  case 118: return -1
  case 86: return -1
  case 101: return -1
  case 114: return -1
  case 82: return -1
  case 84: return -1
  case 97: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 114: return -1
  case 82: return -1
  case 84: return -1
  case 97: return -1
  case 73: return -1
  case 65: return -1
  case 116: return -1
  case 69: return -1
  case 112: return -1
  case 80: return -1
  case 105: return -1
  case 118: return 4
  case 86: return 4
  case 101: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[4] = func(r rune) int {
  switch(r) {
  case 114: return -1
  case 82: return -1
  case 84: return -1
  case 97: return 5
  case 73: return -1
  case 65: return 5
  case 116: return -1
  case 69: return -1
  case 112: return -1
  case 80: return -1
  case 105: return -1
  case 118: return -1
  case 86: return -1
  case 101: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[5] = func(r rune) int {
  switch(r) {
  case 114: return -1
  case 82: return -1
  case 84: return 6
  case 97: return -1
  case 73: return -1
  case 65: return -1
  case 116: return 6
  case 69: return -1
  case 112: return -1
  case 80: return -1
  case 105: return -1
  case 118: return -1
  case 86: return -1
  case 101: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[6] = func(r rune) int {
  switch(r) {
  case 114: return -1
  case 82: return -1
  case 84: return -1
  case 97: return -1
  case 73: return -1
  case 65: return -1
  case 116: return -1
  case 69: return 7
  case 112: return -1
  case 80: return -1
  case 105: return -1
  case 118: return -1
  case 86: return -1
  case 101: return 7
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[7] = true
fun[7] = func(r rune) int {
  switch(r) {
  case 112: return -1
  case 80: return -1
  case 105: return -1
  case 118: return -1
  case 86: return -1
  case 101: return -1
  case 114: return -1
  case 82: return -1
  case 84: return -1
  case 97: return -1
  case 73: return -1
  case 65: return -1
  case 116: return -1
  case 69: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[133].acc = acc[:]
a0[133].f = fun[:]
a0[133].id = 133
}
{
var acc [10]bool
var fun [10]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 112: return 1
  case 82: return -1
  case 101: return -1
  case 69: return -1
  case 114: return -1
  case 118: return -1
  case 80: return 1
  case 105: return -1
  case 73: return -1
  case 86: return -1
  case 108: return -1
  case 76: return -1
  case 103: return -1
  case 71: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 80: return -1
  case 105: return 3
  case 73: return 3
  case 86: return -1
  case 108: return -1
  case 76: return -1
  case 103: return -1
  case 71: return -1
  case 112: return -1
  case 82: return -1
  case 101: return -1
  case 69: return -1
  case 114: return -1
  case 118: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 112: return -1
  case 82: return -1
  case 101: return -1
  case 69: return -1
  case 114: return -1
  case 118: return 4
  case 80: return -1
  case 105: return -1
  case 73: return -1
  case 86: return 4
  case 108: return -1
  case 76: return -1
  case 103: return -1
  case 71: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[9] = true
fun[9] = func(r rune) int {
  switch(r) {
  case 80: return -1
  case 105: return -1
  case 73: return -1
  case 86: return -1
  case 108: return -1
  case 76: return -1
  case 103: return -1
  case 71: return -1
  case 112: return -1
  case 82: return -1
  case 101: return -1
  case 69: return -1
  case 114: return -1
  case 118: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 114: return 2
  case 118: return -1
  case 80: return -1
  case 105: return -1
  case 73: return -1
  case 86: return -1
  case 108: return -1
  case 76: return -1
  case 103: return -1
  case 71: return -1
  case 112: return -1
  case 82: return 2
  case 101: return -1
  case 69: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[4] = func(r rune) int {
  switch(r) {
  case 112: return -1
  case 82: return -1
  case 101: return -1
  case 69: return -1
  case 114: return -1
  case 118: return -1
  case 80: return -1
  case 105: return 5
  case 73: return 5
  case 86: return -1
  case 108: return -1
  case 76: return -1
  case 103: return -1
  case 71: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[5] = func(r rune) int {
  switch(r) {
  case 112: return -1
  case 82: return -1
  case 101: return -1
  case 69: return -1
  case 114: return -1
  case 118: return -1
  case 80: return -1
  case 105: return -1
  case 73: return -1
  case 86: return -1
  case 108: return 6
  case 76: return 6
  case 103: return -1
  case 71: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[6] = func(r rune) int {
  switch(r) {
  case 80: return -1
  case 105: return -1
  case 73: return -1
  case 86: return -1
  case 108: return -1
  case 76: return -1
  case 103: return -1
  case 71: return -1
  case 112: return -1
  case 82: return -1
  case 101: return 7
  case 69: return 7
  case 114: return -1
  case 118: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[7] = func(r rune) int {
  switch(r) {
  case 112: return -1
  case 82: return -1
  case 101: return -1
  case 69: return -1
  case 114: return -1
  case 118: return -1
  case 80: return -1
  case 105: return -1
  case 73: return -1
  case 86: return -1
  case 108: return -1
  case 76: return -1
  case 103: return 8
  case 71: return 8
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[8] = func(r rune) int {
  switch(r) {
  case 114: return -1
  case 118: return -1
  case 80: return -1
  case 105: return -1
  case 73: return -1
  case 86: return -1
  case 108: return -1
  case 76: return -1
  case 103: return -1
  case 71: return -1
  case 112: return -1
  case 82: return -1
  case 101: return 9
  case 69: return 9
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[134].acc = acc[:]
a0[134].f = fun[:]
a0[134].id = 134
}
{
var acc [10]bool
var fun [10]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 112: return 1
  case 80: return 1
  case 114: return -1
  case 82: return -1
  case 111: return -1
  case 99: return -1
  case 67: return -1
  case 101: return -1
  case 69: return -1
  case 117: return -1
  case 85: return -1
  case 79: return -1
  case 100: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 112: return -1
  case 80: return -1
  case 114: return 2
  case 82: return 2
  case 111: return -1
  case 99: return -1
  case 67: return -1
  case 101: return -1
  case 69: return -1
  case 117: return -1
  case 85: return -1
  case 79: return -1
  case 100: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 79: return 3
  case 100: return -1
  case 112: return -1
  case 80: return -1
  case 114: return -1
  case 82: return -1
  case 111: return 3
  case 99: return -1
  case 67: return -1
  case 101: return -1
  case 69: return -1
  case 117: return -1
  case 85: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[4] = func(r rune) int {
  switch(r) {
  case 112: return -1
  case 80: return -1
  case 114: return -1
  case 82: return -1
  case 111: return -1
  case 99: return -1
  case 67: return -1
  case 101: return 5
  case 69: return 5
  case 117: return -1
  case 85: return -1
  case 79: return -1
  case 100: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[5] = func(r rune) int {
  switch(r) {
  case 112: return -1
  case 80: return -1
  case 114: return -1
  case 82: return -1
  case 111: return -1
  case 99: return -1
  case 67: return -1
  case 101: return -1
  case 69: return 6
  case 117: return -1
  case 85: return -1
  case 79: return -1
  case 100: return 6
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[6] = func(r rune) int {
  switch(r) {
  case 79: return -1
  case 100: return -1
  case 112: return -1
  case 80: return -1
  case 114: return -1
  case 82: return -1
  case 111: return -1
  case 99: return -1
  case 67: return -1
  case 101: return -1
  case 69: return -1
  case 117: return 7
  case 85: return 7
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[7] = func(r rune) int {
  switch(r) {
  case 112: return -1
  case 80: return -1
  case 114: return 8
  case 82: return 8
  case 111: return -1
  case 99: return -1
  case 67: return -1
  case 101: return -1
  case 69: return -1
  case 117: return -1
  case 85: return -1
  case 79: return -1
  case 100: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[9] = true
fun[9] = func(r rune) int {
  switch(r) {
  case 112: return -1
  case 80: return -1
  case 114: return -1
  case 82: return -1
  case 111: return -1
  case 99: return -1
  case 67: return -1
  case 101: return -1
  case 69: return -1
  case 117: return -1
  case 85: return -1
  case 79: return -1
  case 100: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 112: return -1
  case 80: return -1
  case 114: return -1
  case 82: return -1
  case 111: return -1
  case 99: return 4
  case 67: return 4
  case 101: return -1
  case 69: return -1
  case 117: return -1
  case 85: return -1
  case 79: return -1
  case 100: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[8] = func(r rune) int {
  switch(r) {
  case 79: return -1
  case 100: return -1
  case 112: return -1
  case 80: return -1
  case 114: return -1
  case 82: return -1
  case 111: return -1
  case 99: return -1
  case 67: return -1
  case 101: return 9
  case 69: return 9
  case 117: return -1
  case 85: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[135].acc = acc[:]
a0[135].f = fun[:]
a0[135].id = 135
}
{
var acc [7]bool
var fun [7]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 80: return 1
  case 66: return -1
  case 108: return -1
  case 76: return -1
  case 73: return -1
  case 99: return -1
  case 67: return -1
  case 112: return 1
  case 117: return -1
  case 85: return -1
  case 98: return -1
  case 105: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 112: return -1
  case 117: return 2
  case 85: return 2
  case 98: return -1
  case 105: return -1
  case 80: return -1
  case 66: return -1
  case 108: return -1
  case 76: return -1
  case 73: return -1
  case 99: return -1
  case 67: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 80: return -1
  case 66: return 3
  case 108: return -1
  case 76: return -1
  case 73: return -1
  case 99: return -1
  case 67: return -1
  case 112: return -1
  case 117: return -1
  case 85: return -1
  case 98: return 3
  case 105: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 80: return -1
  case 66: return -1
  case 108: return 4
  case 76: return 4
  case 73: return -1
  case 99: return -1
  case 67: return -1
  case 112: return -1
  case 117: return -1
  case 85: return -1
  case 98: return -1
  case 105: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[4] = func(r rune) int {
  switch(r) {
  case 112: return -1
  case 117: return -1
  case 85: return -1
  case 98: return -1
  case 105: return 5
  case 80: return -1
  case 66: return -1
  case 108: return -1
  case 76: return -1
  case 73: return 5
  case 99: return -1
  case 67: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[5] = func(r rune) int {
  switch(r) {
  case 112: return -1
  case 117: return -1
  case 85: return -1
  case 98: return -1
  case 105: return -1
  case 80: return -1
  case 66: return -1
  case 108: return -1
  case 76: return -1
  case 73: return -1
  case 99: return 6
  case 67: return 6
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[6] = true
fun[6] = func(r rune) int {
  switch(r) {
  case 80: return -1
  case 66: return -1
  case 108: return -1
  case 76: return -1
  case 73: return -1
  case 99: return -1
  case 67: return -1
  case 112: return -1
  case 117: return -1
  case 85: return -1
  case 98: return -1
  case 105: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[136].acc = acc[:]
a0[136].f = fun[:]
a0[136].id = 136
}
{
var acc [4]bool
var fun [4]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 114: return 1
  case 82: return 1
  case 97: return -1
  case 65: return -1
  case 119: return -1
  case 87: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 114: return -1
  case 82: return -1
  case 97: return 2
  case 65: return 2
  case 119: return -1
  case 87: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 114: return -1
  case 82: return -1
  case 97: return -1
  case 65: return -1
  case 119: return 3
  case 87: return 3
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[3] = true
fun[3] = func(r rune) int {
  switch(r) {
  case 114: return -1
  case 82: return -1
  case 97: return -1
  case 65: return -1
  case 119: return -1
  case 87: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[137].acc = acc[:]
a0[137].f = fun[:]
a0[137].id = 137
}
{
var acc [6]bool
var fun [6]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 101: return -1
  case 69: return -1
  case 65: return -1
  case 76: return -1
  case 109: return -1
  case 77: return -1
  case 114: return 1
  case 82: return 1
  case 97: return -1
  case 108: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 114: return -1
  case 82: return -1
  case 97: return -1
  case 108: return -1
  case 101: return 2
  case 69: return 2
  case 65: return -1
  case 76: return -1
  case 109: return -1
  case 77: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 101: return -1
  case 69: return -1
  case 65: return 3
  case 76: return -1
  case 109: return -1
  case 77: return -1
  case 114: return -1
  case 82: return -1
  case 97: return 3
  case 108: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 101: return -1
  case 69: return -1
  case 65: return -1
  case 76: return 4
  case 109: return -1
  case 77: return -1
  case 114: return -1
  case 82: return -1
  case 97: return -1
  case 108: return 4
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[4] = func(r rune) int {
  switch(r) {
  case 101: return -1
  case 69: return -1
  case 65: return -1
  case 76: return -1
  case 109: return 5
  case 77: return 5
  case 114: return -1
  case 82: return -1
  case 97: return -1
  case 108: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[5] = true
fun[5] = func(r rune) int {
  switch(r) {
  case 114: return -1
  case 82: return -1
  case 97: return -1
  case 108: return -1
  case 101: return -1
  case 69: return -1
  case 65: return -1
  case 76: return -1
  case 109: return -1
  case 77: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[138].acc = acc[:]
a0[138].f = fun[:]
a0[138].id = 138
}
{
var acc [7]bool
var fun [7]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 114: return 1
  case 82: return 1
  case 69: return -1
  case 99: return -1
  case 101: return -1
  case 100: return -1
  case 68: return -1
  case 117: return -1
  case 85: return -1
  case 67: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 101: return 2
  case 100: return -1
  case 68: return -1
  case 117: return -1
  case 85: return -1
  case 67: return -1
  case 114: return -1
  case 82: return -1
  case 69: return 2
  case 99: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 114: return -1
  case 82: return -1
  case 69: return -1
  case 99: return -1
  case 101: return -1
  case 100: return 3
  case 68: return 3
  case 117: return -1
  case 85: return -1
  case 67: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 114: return -1
  case 82: return -1
  case 69: return -1
  case 99: return -1
  case 101: return -1
  case 100: return -1
  case 68: return -1
  case 117: return 4
  case 85: return 4
  case 67: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[4] = func(r rune) int {
  switch(r) {
  case 101: return -1
  case 100: return -1
  case 68: return -1
  case 117: return -1
  case 85: return -1
  case 67: return 5
  case 114: return -1
  case 82: return -1
  case 69: return -1
  case 99: return 5
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[5] = func(r rune) int {
  switch(r) {
  case 101: return 6
  case 100: return -1
  case 68: return -1
  case 117: return -1
  case 85: return -1
  case 67: return -1
  case 114: return -1
  case 82: return -1
  case 69: return 6
  case 99: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[6] = true
fun[6] = func(r rune) int {
  switch(r) {
  case 114: return -1
  case 82: return -1
  case 69: return -1
  case 99: return -1
  case 101: return -1
  case 100: return -1
  case 68: return -1
  case 117: return -1
  case 85: return -1
  case 67: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[139].acc = acc[:]
a0[139].f = fun[:]
a0[139].id = 139
}
{
var acc [7]bool
var fun [7]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 114: return 1
  case 69: return -1
  case 110: return -1
  case 65: return -1
  case 77: return -1
  case 82: return 1
  case 101: return -1
  case 78: return -1
  case 97: return -1
  case 109: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 82: return -1
  case 101: return 2
  case 78: return -1
  case 97: return -1
  case 109: return -1
  case 114: return -1
  case 69: return 2
  case 110: return -1
  case 65: return -1
  case 77: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 114: return -1
  case 69: return -1
  case 110: return 3
  case 65: return -1
  case 77: return -1
  case 82: return -1
  case 101: return -1
  case 78: return 3
  case 97: return -1
  case 109: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 114: return -1
  case 69: return -1
  case 110: return -1
  case 65: return 4
  case 77: return -1
  case 82: return -1
  case 101: return -1
  case 78: return -1
  case 97: return 4
  case 109: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[4] = func(r rune) int {
  switch(r) {
  case 114: return -1
  case 69: return -1
  case 110: return -1
  case 65: return -1
  case 77: return 5
  case 82: return -1
  case 101: return -1
  case 78: return -1
  case 97: return -1
  case 109: return 5
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[5] = func(r rune) int {
  switch(r) {
  case 114: return -1
  case 69: return 6
  case 110: return -1
  case 65: return -1
  case 77: return -1
  case 82: return -1
  case 101: return 6
  case 78: return -1
  case 97: return -1
  case 109: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[6] = true
fun[6] = func(r rune) int {
  switch(r) {
  case 82: return -1
  case 101: return -1
  case 78: return -1
  case 97: return -1
  case 109: return -1
  case 114: return -1
  case 69: return -1
  case 110: return -1
  case 65: return -1
  case 77: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[140].acc = acc[:]
a0[140].f = fun[:]
a0[140].id = 140
}
{
var acc [7]bool
var fun [7]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 114: return 1
  case 82: return 1
  case 69: return -1
  case 116: return -1
  case 117: return -1
  case 85: return -1
  case 110: return -1
  case 78: return -1
  case 101: return -1
  case 84: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 114: return -1
  case 82: return -1
  case 69: return 2
  case 116: return -1
  case 117: return -1
  case 85: return -1
  case 110: return -1
  case 78: return -1
  case 101: return 2
  case 84: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 101: return -1
  case 84: return 3
  case 114: return -1
  case 82: return -1
  case 69: return -1
  case 116: return 3
  case 117: return -1
  case 85: return -1
  case 110: return -1
  case 78: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 101: return -1
  case 84: return -1
  case 114: return -1
  case 82: return -1
  case 69: return -1
  case 116: return -1
  case 117: return 4
  case 85: return 4
  case 110: return -1
  case 78: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[4] = func(r rune) int {
  switch(r) {
  case 114: return 5
  case 82: return 5
  case 69: return -1
  case 116: return -1
  case 117: return -1
  case 85: return -1
  case 110: return -1
  case 78: return -1
  case 101: return -1
  case 84: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[5] = func(r rune) int {
  switch(r) {
  case 114: return -1
  case 82: return -1
  case 69: return -1
  case 116: return -1
  case 117: return -1
  case 85: return -1
  case 110: return 6
  case 78: return 6
  case 101: return -1
  case 84: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[6] = true
fun[6] = func(r rune) int {
  switch(r) {
  case 101: return -1
  case 84: return -1
  case 114: return -1
  case 82: return -1
  case 69: return -1
  case 116: return -1
  case 117: return -1
  case 85: return -1
  case 110: return -1
  case 78: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[141].acc = acc[:]
a0[141].f = fun[:]
a0[141].id = 141
}
{
var acc [10]bool
var fun [10]func(rune) int
fun[4] = func(r rune) int {
  switch(r) {
  case 114: return 5
  case 85: return -1
  case 110: return -1
  case 105: return -1
  case 82: return 5
  case 103: return -1
  case 101: return -1
  case 116: return -1
  case 84: return -1
  case 78: return -1
  case 71: return -1
  case 69: return -1
  case 117: return -1
  case 73: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[5] = func(r rune) int {
  switch(r) {
  case 82: return -1
  case 103: return -1
  case 101: return -1
  case 116: return -1
  case 84: return -1
  case 78: return 6
  case 71: return -1
  case 69: return -1
  case 117: return -1
  case 73: return -1
  case 114: return -1
  case 85: return -1
  case 110: return 6
  case 105: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[6] = func(r rune) int {
  switch(r) {
  case 114: return -1
  case 85: return -1
  case 110: return -1
  case 105: return 7
  case 82: return -1
  case 103: return -1
  case 101: return -1
  case 116: return -1
  case 84: return -1
  case 78: return -1
  case 71: return -1
  case 69: return -1
  case 117: return -1
  case 73: return 7
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[7] = func(r rune) int {
  switch(r) {
  case 101: return -1
  case 116: return -1
  case 84: return -1
  case 78: return 8
  case 71: return -1
  case 69: return -1
  case 117: return -1
  case 73: return -1
  case 114: return -1
  case 85: return -1
  case 110: return 8
  case 105: return -1
  case 82: return -1
  case 103: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[8] = func(r rune) int {
  switch(r) {
  case 101: return -1
  case 116: return -1
  case 84: return -1
  case 78: return -1
  case 71: return 9
  case 69: return -1
  case 117: return -1
  case 73: return -1
  case 114: return -1
  case 85: return -1
  case 110: return -1
  case 105: return -1
  case 82: return -1
  case 103: return 9
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[9] = true
fun[9] = func(r rune) int {
  switch(r) {
  case 101: return -1
  case 116: return -1
  case 84: return -1
  case 78: return -1
  case 71: return -1
  case 69: return -1
  case 117: return -1
  case 73: return -1
  case 114: return -1
  case 85: return -1
  case 110: return -1
  case 105: return -1
  case 82: return -1
  case 103: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[0] = func(r rune) int {
  switch(r) {
  case 82: return 1
  case 103: return -1
  case 101: return -1
  case 116: return -1
  case 84: return -1
  case 78: return -1
  case 71: return -1
  case 69: return -1
  case 117: return -1
  case 73: return -1
  case 114: return 1
  case 85: return -1
  case 110: return -1
  case 105: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 114: return -1
  case 85: return -1
  case 110: return -1
  case 105: return -1
  case 82: return -1
  case 103: return -1
  case 101: return 2
  case 116: return -1
  case 84: return -1
  case 78: return -1
  case 71: return -1
  case 69: return 2
  case 117: return -1
  case 73: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 82: return -1
  case 103: return -1
  case 101: return -1
  case 116: return 3
  case 84: return 3
  case 78: return -1
  case 71: return -1
  case 69: return -1
  case 117: return -1
  case 73: return -1
  case 114: return -1
  case 85: return -1
  case 110: return -1
  case 105: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 69: return -1
  case 117: return 4
  case 73: return -1
  case 114: return -1
  case 85: return 4
  case 110: return -1
  case 105: return -1
  case 82: return -1
  case 103: return -1
  case 101: return -1
  case 116: return -1
  case 84: return -1
  case 78: return -1
  case 71: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[142].acc = acc[:]
a0[142].f = fun[:]
a0[142].id = 142
}
{
var acc [7]bool
var fun [7]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 82: return 1
  case 101: return -1
  case 69: return -1
  case 86: return -1
  case 111: return -1
  case 79: return -1
  case 75: return -1
  case 114: return 1
  case 118: return -1
  case 107: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 114: return -1
  case 118: return -1
  case 107: return -1
  case 82: return -1
  case 101: return 2
  case 69: return 2
  case 86: return -1
  case 111: return -1
  case 79: return -1
  case 75: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 114: return -1
  case 118: return 3
  case 107: return -1
  case 82: return -1
  case 101: return -1
  case 69: return -1
  case 86: return 3
  case 111: return -1
  case 79: return -1
  case 75: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 82: return -1
  case 101: return -1
  case 69: return -1
  case 86: return -1
  case 111: return 4
  case 79: return 4
  case 75: return -1
  case 114: return -1
  case 118: return -1
  case 107: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[4] = func(r rune) int {
  switch(r) {
  case 114: return -1
  case 118: return -1
  case 107: return 5
  case 82: return -1
  case 101: return -1
  case 69: return -1
  case 86: return -1
  case 111: return -1
  case 79: return -1
  case 75: return 5
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[5] = func(r rune) int {
  switch(r) {
  case 114: return -1
  case 118: return -1
  case 107: return -1
  case 82: return -1
  case 101: return 6
  case 69: return 6
  case 86: return -1
  case 111: return -1
  case 79: return -1
  case 75: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[6] = true
fun[6] = func(r rune) int {
  switch(r) {
  case 82: return -1
  case 101: return -1
  case 69: return -1
  case 86: return -1
  case 111: return -1
  case 79: return -1
  case 75: return -1
  case 114: return -1
  case 118: return -1
  case 107: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[143].acc = acc[:]
a0[143].f = fun[:]
a0[143].id = 143
}
{
var acc [6]bool
var fun [6]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 73: return -1
  case 103: return -1
  case 71: return -1
  case 84: return -1
  case 114: return 1
  case 82: return 1
  case 105: return -1
  case 104: return -1
  case 72: return -1
  case 116: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 73: return 2
  case 103: return -1
  case 71: return -1
  case 84: return -1
  case 114: return -1
  case 82: return -1
  case 105: return 2
  case 104: return -1
  case 72: return -1
  case 116: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 114: return -1
  case 82: return -1
  case 105: return -1
  case 104: return -1
  case 72: return -1
  case 116: return -1
  case 73: return -1
  case 103: return 3
  case 71: return 3
  case 84: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 114: return -1
  case 82: return -1
  case 105: return -1
  case 104: return 4
  case 72: return 4
  case 116: return -1
  case 73: return -1
  case 103: return -1
  case 71: return -1
  case 84: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[4] = func(r rune) int {
  switch(r) {
  case 73: return -1
  case 103: return -1
  case 71: return -1
  case 84: return 5
  case 114: return -1
  case 82: return -1
  case 105: return -1
  case 104: return -1
  case 72: return -1
  case 116: return 5
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[5] = true
fun[5] = func(r rune) int {
  switch(r) {
  case 114: return -1
  case 82: return -1
  case 105: return -1
  case 104: return -1
  case 72: return -1
  case 116: return -1
  case 73: return -1
  case 103: return -1
  case 71: return -1
  case 84: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[144].acc = acc[:]
a0[144].f = fun[:]
a0[144].id = 144
}
{
var acc [5]bool
var fun [5]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 114: return 1
  case 82: return 1
  case 111: return -1
  case 79: return -1
  case 108: return -1
  case 76: return -1
  case 101: return -1
  case 69: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 114: return -1
  case 82: return -1
  case 111: return 2
  case 79: return 2
  case 108: return -1
  case 76: return -1
  case 101: return -1
  case 69: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 114: return -1
  case 82: return -1
  case 111: return -1
  case 79: return -1
  case 108: return 3
  case 76: return 3
  case 101: return -1
  case 69: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 114: return -1
  case 82: return -1
  case 111: return -1
  case 79: return -1
  case 108: return -1
  case 76: return -1
  case 101: return 4
  case 69: return 4
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[4] = true
fun[4] = func(r rune) int {
  switch(r) {
  case 114: return -1
  case 82: return -1
  case 111: return -1
  case 79: return -1
  case 108: return -1
  case 76: return -1
  case 101: return -1
  case 69: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[145].acc = acc[:]
a0[145].f = fun[:]
a0[145].id = 145
}
{
var acc [9]bool
var fun [9]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 114: return 1
  case 111: return -1
  case 65: return -1
  case 75: return -1
  case 108: return -1
  case 66: return -1
  case 107: return -1
  case 82: return 1
  case 79: return -1
  case 76: return -1
  case 98: return -1
  case 97: return -1
  case 99: return -1
  case 67: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 82: return -1
  case 79: return 2
  case 76: return -1
  case 98: return -1
  case 97: return -1
  case 99: return -1
  case 67: return -1
  case 114: return -1
  case 111: return 2
  case 65: return -1
  case 75: return -1
  case 108: return -1
  case 66: return -1
  case 107: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 82: return -1
  case 79: return -1
  case 76: return 3
  case 98: return -1
  case 97: return -1
  case 99: return -1
  case 67: return -1
  case 114: return -1
  case 111: return -1
  case 65: return -1
  case 75: return -1
  case 108: return 3
  case 66: return -1
  case 107: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[7] = func(r rune) int {
  switch(r) {
  case 82: return -1
  case 79: return -1
  case 76: return -1
  case 98: return -1
  case 97: return -1
  case 99: return -1
  case 67: return -1
  case 114: return -1
  case 111: return -1
  case 65: return -1
  case 75: return 8
  case 108: return -1
  case 66: return -1
  case 107: return 8
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 108: return 4
  case 66: return -1
  case 107: return -1
  case 82: return -1
  case 79: return -1
  case 76: return 4
  case 98: return -1
  case 97: return -1
  case 99: return -1
  case 67: return -1
  case 114: return -1
  case 111: return -1
  case 65: return -1
  case 75: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[4] = func(r rune) int {
  switch(r) {
  case 114: return -1
  case 111: return -1
  case 65: return -1
  case 75: return -1
  case 108: return -1
  case 66: return 5
  case 107: return -1
  case 82: return -1
  case 79: return -1
  case 76: return -1
  case 98: return 5
  case 97: return -1
  case 99: return -1
  case 67: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[5] = func(r rune) int {
  switch(r) {
  case 82: return -1
  case 79: return -1
  case 76: return -1
  case 98: return -1
  case 97: return 6
  case 99: return -1
  case 67: return -1
  case 114: return -1
  case 111: return -1
  case 65: return 6
  case 75: return -1
  case 108: return -1
  case 66: return -1
  case 107: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[6] = func(r rune) int {
  switch(r) {
  case 82: return -1
  case 79: return -1
  case 76: return -1
  case 98: return -1
  case 97: return -1
  case 99: return 7
  case 67: return 7
  case 114: return -1
  case 111: return -1
  case 65: return -1
  case 75: return -1
  case 108: return -1
  case 66: return -1
  case 107: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[8] = true
fun[8] = func(r rune) int {
  switch(r) {
  case 82: return -1
  case 79: return -1
  case 76: return -1
  case 98: return -1
  case 97: return -1
  case 99: return -1
  case 67: return -1
  case 114: return -1
  case 111: return -1
  case 65: return -1
  case 75: return -1
  case 108: return -1
  case 66: return -1
  case 107: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[146].acc = acc[:]
a0[146].f = fun[:]
a0[146].id = 146
}
{
var acc [10]bool
var fun [10]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 116: return -1
  case 84: return -1
  case 69: return -1
  case 115: return 1
  case 83: return 1
  case 97: return -1
  case 65: return -1
  case 105: return -1
  case 73: return -1
  case 102: return -1
  case 70: return -1
  case 101: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[4] = func(r rune) int {
  switch(r) {
  case 116: return -1
  case 84: return -1
  case 69: return -1
  case 115: return 5
  case 83: return 5
  case 97: return -1
  case 65: return -1
  case 105: return -1
  case 73: return -1
  case 102: return -1
  case 70: return -1
  case 101: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[5] = func(r rune) int {
  switch(r) {
  case 116: return -1
  case 84: return -1
  case 69: return -1
  case 115: return -1
  case 83: return -1
  case 97: return -1
  case 65: return -1
  case 105: return -1
  case 73: return -1
  case 102: return 6
  case 70: return 6
  case 101: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[7] = func(r rune) int {
  switch(r) {
  case 116: return -1
  case 84: return -1
  case 69: return 8
  case 115: return -1
  case 83: return -1
  case 97: return -1
  case 65: return -1
  case 105: return -1
  case 73: return -1
  case 102: return -1
  case 70: return -1
  case 101: return 8
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 115: return -1
  case 83: return -1
  case 97: return 2
  case 65: return 2
  case 105: return -1
  case 73: return -1
  case 102: return -1
  case 70: return -1
  case 101: return -1
  case 116: return -1
  case 84: return -1
  case 69: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 116: return 3
  case 84: return 3
  case 69: return -1
  case 115: return -1
  case 83: return -1
  case 97: return -1
  case 65: return -1
  case 105: return -1
  case 73: return -1
  case 102: return -1
  case 70: return -1
  case 101: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 115: return -1
  case 83: return -1
  case 97: return -1
  case 65: return -1
  case 105: return 4
  case 73: return 4
  case 102: return -1
  case 70: return -1
  case 101: return -1
  case 116: return -1
  case 84: return -1
  case 69: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[6] = func(r rune) int {
  switch(r) {
  case 115: return -1
  case 83: return -1
  case 97: return -1
  case 65: return -1
  case 105: return 7
  case 73: return 7
  case 102: return -1
  case 70: return -1
  case 101: return -1
  case 116: return -1
  case 84: return -1
  case 69: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[8] = func(r rune) int {
  switch(r) {
  case 116: return -1
  case 84: return -1
  case 69: return -1
  case 115: return 9
  case 83: return 9
  case 97: return -1
  case 65: return -1
  case 105: return -1
  case 73: return -1
  case 102: return -1
  case 70: return -1
  case 101: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[9] = true
fun[9] = func(r rune) int {
  switch(r) {
  case 115: return -1
  case 83: return -1
  case 97: return -1
  case 65: return -1
  case 105: return -1
  case 73: return -1
  case 102: return -1
  case 70: return -1
  case 101: return -1
  case 116: return -1
  case 84: return -1
  case 69: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[147].acc = acc[:]
a0[147].f = fun[:]
a0[147].id = 147
}
{
var acc [7]bool
var fun [7]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 99: return -1
  case 67: return -1
  case 104: return -1
  case 101: return -1
  case 69: return -1
  case 65: return -1
  case 115: return 1
  case 83: return 1
  case 72: return -1
  case 109: return -1
  case 77: return -1
  case 97: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 99: return 2
  case 67: return 2
  case 104: return -1
  case 101: return -1
  case 69: return -1
  case 65: return -1
  case 115: return -1
  case 83: return -1
  case 72: return -1
  case 109: return -1
  case 77: return -1
  case 97: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 115: return -1
  case 83: return -1
  case 72: return 3
  case 109: return -1
  case 77: return -1
  case 97: return -1
  case 99: return -1
  case 67: return -1
  case 104: return 3
  case 101: return -1
  case 69: return -1
  case 65: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 115: return -1
  case 83: return -1
  case 72: return -1
  case 109: return -1
  case 77: return -1
  case 97: return -1
  case 99: return -1
  case 67: return -1
  case 104: return -1
  case 101: return 4
  case 69: return 4
  case 65: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[4] = func(r rune) int {
  switch(r) {
  case 99: return -1
  case 67: return -1
  case 104: return -1
  case 101: return -1
  case 69: return -1
  case 65: return -1
  case 115: return -1
  case 83: return -1
  case 72: return -1
  case 109: return 5
  case 77: return 5
  case 97: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[5] = func(r rune) int {
  switch(r) {
  case 115: return -1
  case 83: return -1
  case 72: return -1
  case 109: return -1
  case 77: return -1
  case 97: return 6
  case 99: return -1
  case 67: return -1
  case 104: return -1
  case 101: return -1
  case 69: return -1
  case 65: return 6
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[6] = true
fun[6] = func(r rune) int {
  switch(r) {
  case 99: return -1
  case 67: return -1
  case 104: return -1
  case 101: return -1
  case 69: return -1
  case 65: return -1
  case 115: return -1
  case 83: return -1
  case 72: return -1
  case 109: return -1
  case 77: return -1
  case 97: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[148].acc = acc[:]
a0[148].f = fun[:]
a0[148].id = 148
}
{
var acc [7]bool
var fun [7]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 115: return 1
  case 108: return -1
  case 76: return -1
  case 99: return -1
  case 67: return -1
  case 84: return -1
  case 83: return 1
  case 101: return -1
  case 69: return -1
  case 116: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 115: return -1
  case 108: return -1
  case 76: return -1
  case 99: return -1
  case 67: return -1
  case 84: return -1
  case 83: return -1
  case 101: return 2
  case 69: return 2
  case 116: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 115: return -1
  case 108: return 3
  case 76: return 3
  case 99: return -1
  case 67: return -1
  case 84: return -1
  case 83: return -1
  case 101: return -1
  case 69: return -1
  case 116: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 83: return -1
  case 101: return 4
  case 69: return 4
  case 116: return -1
  case 115: return -1
  case 108: return -1
  case 76: return -1
  case 99: return -1
  case 67: return -1
  case 84: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[4] = func(r rune) int {
  switch(r) {
  case 83: return -1
  case 101: return -1
  case 69: return -1
  case 116: return -1
  case 115: return -1
  case 108: return -1
  case 76: return -1
  case 99: return 5
  case 67: return 5
  case 84: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[5] = func(r rune) int {
  switch(r) {
  case 115: return -1
  case 108: return -1
  case 76: return -1
  case 99: return -1
  case 67: return -1
  case 84: return 6
  case 83: return -1
  case 101: return -1
  case 69: return -1
  case 116: return 6
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[6] = true
fun[6] = func(r rune) int {
  switch(r) {
  case 115: return -1
  case 108: return -1
  case 76: return -1
  case 99: return -1
  case 67: return -1
  case 84: return -1
  case 83: return -1
  case 101: return -1
  case 69: return -1
  case 116: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[149].acc = acc[:]
a0[149].f = fun[:]
a0[149].id = 149
}
{
var acc [4]bool
var fun [4]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 115: return 1
  case 83: return 1
  case 101: return -1
  case 69: return -1
  case 116: return -1
  case 84: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 115: return -1
  case 83: return -1
  case 101: return 2
  case 69: return 2
  case 116: return -1
  case 84: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 115: return -1
  case 83: return -1
  case 101: return -1
  case 69: return -1
  case 116: return 3
  case 84: return 3
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[3] = true
fun[3] = func(r rune) int {
  switch(r) {
  case 115: return -1
  case 83: return -1
  case 101: return -1
  case 69: return -1
  case 116: return -1
  case 84: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[150].acc = acc[:]
a0[150].f = fun[:]
a0[150].id = 150
}
{
var acc [5]bool
var fun [5]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 115: return 1
  case 83: return 1
  case 104: return -1
  case 72: return -1
  case 111: return -1
  case 79: return -1
  case 119: return -1
  case 87: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 115: return -1
  case 83: return -1
  case 104: return 2
  case 72: return 2
  case 111: return -1
  case 79: return -1
  case 119: return -1
  case 87: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 115: return -1
  case 83: return -1
  case 104: return -1
  case 72: return -1
  case 111: return 3
  case 79: return 3
  case 119: return -1
  case 87: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 115: return -1
  case 83: return -1
  case 104: return -1
  case 72: return -1
  case 111: return -1
  case 79: return -1
  case 119: return 4
  case 87: return 4
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[4] = true
fun[4] = func(r rune) int {
  switch(r) {
  case 115: return -1
  case 83: return -1
  case 104: return -1
  case 72: return -1
  case 111: return -1
  case 79: return -1
  case 119: return -1
  case 87: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[151].acc = acc[:]
a0[151].f = fun[:]
a0[151].id = 151
}
{
var acc [5]bool
var fun [5]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 115: return 1
  case 83: return 1
  case 111: return -1
  case 79: return -1
  case 109: return -1
  case 77: return -1
  case 101: return -1
  case 69: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 115: return -1
  case 83: return -1
  case 111: return 2
  case 79: return 2
  case 109: return -1
  case 77: return -1
  case 101: return -1
  case 69: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 115: return -1
  case 83: return -1
  case 111: return -1
  case 79: return -1
  case 109: return 3
  case 77: return 3
  case 101: return -1
  case 69: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 115: return -1
  case 83: return -1
  case 111: return -1
  case 79: return -1
  case 109: return -1
  case 77: return -1
  case 101: return 4
  case 69: return 4
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[4] = true
fun[4] = func(r rune) int {
  switch(r) {
  case 115: return -1
  case 83: return -1
  case 111: return -1
  case 79: return -1
  case 109: return -1
  case 77: return -1
  case 101: return -1
  case 69: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[152].acc = acc[:]
a0[152].f = fun[:]
a0[152].id = 152
}
{
var acc [6]bool
var fun [6]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 115: return 1
  case 83: return 1
  case 116: return -1
  case 84: return -1
  case 97: return -1
  case 65: return -1
  case 114: return -1
  case 82: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 115: return -1
  case 83: return -1
  case 116: return 2
  case 84: return 2
  case 97: return -1
  case 65: return -1
  case 114: return -1
  case 82: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 115: return -1
  case 83: return -1
  case 116: return -1
  case 84: return -1
  case 97: return 3
  case 65: return 3
  case 114: return -1
  case 82: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 115: return -1
  case 83: return -1
  case 116: return -1
  case 84: return -1
  case 97: return -1
  case 65: return -1
  case 114: return 4
  case 82: return 4
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[4] = func(r rune) int {
  switch(r) {
  case 115: return -1
  case 83: return -1
  case 116: return 5
  case 84: return 5
  case 97: return -1
  case 65: return -1
  case 114: return -1
  case 82: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[5] = true
fun[5] = func(r rune) int {
  switch(r) {
  case 115: return -1
  case 83: return -1
  case 116: return -1
  case 84: return -1
  case 97: return -1
  case 65: return -1
  case 114: return -1
  case 82: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[153].acc = acc[:]
a0[153].f = fun[:]
a0[153].id = 153
}
{
var acc [11]bool
var fun [11]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 84: return -1
  case 97: return -1
  case 65: return -1
  case 105: return -1
  case 99: return -1
  case 115: return 1
  case 83: return 1
  case 116: return -1
  case 73: return -1
  case 67: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 84: return -1
  case 97: return 3
  case 65: return 3
  case 105: return -1
  case 99: return -1
  case 115: return -1
  case 83: return -1
  case 116: return -1
  case 73: return -1
  case 67: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[5] = func(r rune) int {
  switch(r) {
  case 115: return 6
  case 83: return 6
  case 116: return -1
  case 73: return -1
  case 67: return -1
  case 84: return -1
  case 97: return -1
  case 65: return -1
  case 105: return -1
  case 99: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[6] = func(r rune) int {
  switch(r) {
  case 115: return -1
  case 83: return -1
  case 116: return 7
  case 73: return -1
  case 67: return -1
  case 84: return 7
  case 97: return -1
  case 65: return -1
  case 105: return -1
  case 99: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[8] = func(r rune) int {
  switch(r) {
  case 84: return -1
  case 97: return -1
  case 65: return -1
  case 105: return -1
  case 99: return 9
  case 115: return -1
  case 83: return -1
  case 116: return -1
  case 73: return -1
  case 67: return 9
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[9] = func(r rune) int {
  switch(r) {
  case 84: return -1
  case 97: return -1
  case 65: return -1
  case 105: return -1
  case 99: return -1
  case 115: return 10
  case 83: return 10
  case 116: return -1
  case 73: return -1
  case 67: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[10] = true
fun[10] = func(r rune) int {
  switch(r) {
  case 84: return -1
  case 97: return -1
  case 65: return -1
  case 105: return -1
  case 99: return -1
  case 115: return -1
  case 83: return -1
  case 116: return -1
  case 73: return -1
  case 67: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 115: return -1
  case 83: return -1
  case 116: return 2
  case 73: return -1
  case 67: return -1
  case 84: return 2
  case 97: return -1
  case 65: return -1
  case 105: return -1
  case 99: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 84: return 4
  case 97: return -1
  case 65: return -1
  case 105: return -1
  case 99: return -1
  case 115: return -1
  case 83: return -1
  case 116: return 4
  case 73: return -1
  case 67: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[4] = func(r rune) int {
  switch(r) {
  case 84: return -1
  case 97: return -1
  case 65: return -1
  case 105: return 5
  case 99: return -1
  case 115: return -1
  case 83: return -1
  case 116: return -1
  case 73: return 5
  case 67: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[7] = func(r rune) int {
  switch(r) {
  case 84: return -1
  case 97: return -1
  case 65: return -1
  case 105: return 8
  case 99: return -1
  case 115: return -1
  case 83: return -1
  case 116: return -1
  case 73: return 8
  case 67: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[154].acc = acc[:]
a0[154].f = fun[:]
a0[154].id = 154
}
{
var acc [7]bool
var fun [7]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 83: return 1
  case 89: return -1
  case 84: return -1
  case 115: return 1
  case 121: return -1
  case 116: return -1
  case 101: return -1
  case 69: return -1
  case 109: return -1
  case 77: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 115: return -1
  case 121: return 2
  case 116: return -1
  case 101: return -1
  case 69: return -1
  case 109: return -1
  case 77: return -1
  case 83: return -1
  case 89: return 2
  case 84: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 83: return 3
  case 89: return -1
  case 84: return -1
  case 115: return 3
  case 121: return -1
  case 116: return -1
  case 101: return -1
  case 69: return -1
  case 109: return -1
  case 77: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 115: return -1
  case 121: return -1
  case 116: return 4
  case 101: return -1
  case 69: return -1
  case 109: return -1
  case 77: return -1
  case 83: return -1
  case 89: return -1
  case 84: return 4
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[4] = func(r rune) int {
  switch(r) {
  case 115: return -1
  case 121: return -1
  case 116: return -1
  case 101: return 5
  case 69: return 5
  case 109: return -1
  case 77: return -1
  case 83: return -1
  case 89: return -1
  case 84: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[5] = func(r rune) int {
  switch(r) {
  case 115: return -1
  case 121: return -1
  case 116: return -1
  case 101: return -1
  case 69: return -1
  case 109: return 6
  case 77: return 6
  case 83: return -1
  case 89: return -1
  case 84: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[6] = true
fun[6] = func(r rune) int {
  switch(r) {
  case 115: return -1
  case 121: return -1
  case 116: return -1
  case 101: return -1
  case 69: return -1
  case 109: return -1
  case 77: return -1
  case 83: return -1
  case 89: return -1
  case 84: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[155].acc = acc[:]
a0[155].f = fun[:]
a0[155].id = 155
}
{
var acc [5]bool
var fun [5]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 116: return 1
  case 84: return 1
  case 104: return -1
  case 72: return -1
  case 101: return -1
  case 69: return -1
  case 110: return -1
  case 78: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 116: return -1
  case 84: return -1
  case 104: return 2
  case 72: return 2
  case 101: return -1
  case 69: return -1
  case 110: return -1
  case 78: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 116: return -1
  case 84: return -1
  case 104: return -1
  case 72: return -1
  case 101: return 3
  case 69: return 3
  case 110: return -1
  case 78: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 116: return -1
  case 84: return -1
  case 104: return -1
  case 72: return -1
  case 101: return -1
  case 69: return -1
  case 110: return 4
  case 78: return 4
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[4] = true
fun[4] = func(r rune) int {
  switch(r) {
  case 116: return -1
  case 84: return -1
  case 104: return -1
  case 72: return -1
  case 101: return -1
  case 69: return -1
  case 110: return -1
  case 78: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[156].acc = acc[:]
a0[156].f = fun[:]
a0[156].id = 156
}
{
var acc [3]bool
var fun [3]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 116: return 1
  case 84: return 1
  case 111: return -1
  case 79: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 116: return -1
  case 84: return -1
  case 111: return 2
  case 79: return 2
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[2] = true
fun[2] = func(r rune) int {
  switch(r) {
  case 116: return -1
  case 84: return -1
  case 111: return -1
  case 79: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[157].acc = acc[:]
a0[157].f = fun[:]
a0[157].id = 157
}
{
var acc [12]bool
var fun [12]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 116: return 1
  case 82: return -1
  case 78: return -1
  case 73: return -1
  case 110: return -1
  case 67: return -1
  case 111: return -1
  case 79: return -1
  case 97: return -1
  case 115: return -1
  case 83: return -1
  case 84: return 1
  case 114: return -1
  case 65: return -1
  case 99: return -1
  case 105: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[6] = func(r rune) int {
  switch(r) {
  case 110: return -1
  case 67: return 7
  case 111: return -1
  case 79: return -1
  case 97: return -1
  case 115: return -1
  case 83: return -1
  case 84: return -1
  case 114: return -1
  case 65: return -1
  case 99: return 7
  case 105: return -1
  case 116: return -1
  case 82: return -1
  case 78: return -1
  case 73: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[7] = func(r rune) int {
  switch(r) {
  case 97: return -1
  case 115: return -1
  case 83: return -1
  case 84: return 8
  case 114: return -1
  case 65: return -1
  case 99: return -1
  case 105: return -1
  case 116: return 8
  case 82: return -1
  case 78: return -1
  case 73: return -1
  case 110: return -1
  case 67: return -1
  case 111: return -1
  case 79: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[9] = func(r rune) int {
  switch(r) {
  case 116: return -1
  case 82: return -1
  case 78: return -1
  case 73: return -1
  case 110: return -1
  case 67: return -1
  case 111: return 10
  case 79: return 10
  case 97: return -1
  case 115: return -1
  case 83: return -1
  case 84: return -1
  case 114: return -1
  case 65: return -1
  case 99: return -1
  case 105: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 84: return -1
  case 114: return 2
  case 65: return -1
  case 99: return -1
  case 105: return -1
  case 116: return -1
  case 82: return 2
  case 78: return -1
  case 73: return -1
  case 110: return -1
  case 67: return -1
  case 111: return -1
  case 79: return -1
  case 97: return -1
  case 115: return -1
  case 83: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 97: return 3
  case 115: return -1
  case 83: return -1
  case 84: return -1
  case 114: return -1
  case 65: return 3
  case 99: return -1
  case 105: return -1
  case 116: return -1
  case 82: return -1
  case 78: return -1
  case 73: return -1
  case 110: return -1
  case 67: return -1
  case 111: return -1
  case 79: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 84: return -1
  case 114: return -1
  case 65: return -1
  case 99: return -1
  case 105: return -1
  case 116: return -1
  case 82: return -1
  case 78: return 4
  case 73: return -1
  case 110: return 4
  case 67: return -1
  case 111: return -1
  case 79: return -1
  case 97: return -1
  case 115: return -1
  case 83: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[4] = func(r rune) int {
  switch(r) {
  case 110: return -1
  case 67: return -1
  case 111: return -1
  case 79: return -1
  case 97: return -1
  case 115: return 5
  case 83: return 5
  case 84: return -1
  case 114: return -1
  case 65: return -1
  case 99: return -1
  case 105: return -1
  case 116: return -1
  case 82: return -1
  case 78: return -1
  case 73: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[5] = func(r rune) int {
  switch(r) {
  case 110: return -1
  case 67: return -1
  case 111: return -1
  case 79: return -1
  case 97: return 6
  case 115: return -1
  case 83: return -1
  case 84: return -1
  case 114: return -1
  case 65: return 6
  case 99: return -1
  case 105: return -1
  case 116: return -1
  case 82: return -1
  case 78: return -1
  case 73: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[8] = func(r rune) int {
  switch(r) {
  case 116: return -1
  case 82: return -1
  case 78: return -1
  case 73: return 9
  case 110: return -1
  case 67: return -1
  case 111: return -1
  case 79: return -1
  case 97: return -1
  case 115: return -1
  case 83: return -1
  case 84: return -1
  case 114: return -1
  case 65: return -1
  case 99: return -1
  case 105: return 9
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[10] = func(r rune) int {
  switch(r) {
  case 116: return -1
  case 82: return -1
  case 78: return 11
  case 73: return -1
  case 110: return 11
  case 67: return -1
  case 111: return -1
  case 79: return -1
  case 97: return -1
  case 115: return -1
  case 83: return -1
  case 84: return -1
  case 114: return -1
  case 65: return -1
  case 99: return -1
  case 105: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[11] = true
fun[11] = func(r rune) int {
  switch(r) {
  case 84: return -1
  case 114: return -1
  case 65: return -1
  case 99: return -1
  case 105: return -1
  case 116: return -1
  case 82: return -1
  case 78: return -1
  case 73: return -1
  case 110: return -1
  case 67: return -1
  case 111: return -1
  case 79: return -1
  case 97: return -1
  case 115: return -1
  case 83: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[158].acc = acc[:]
a0[158].f = fun[:]
a0[158].id = 158
}
{
var acc [8]bool
var fun [8]func(rune) int
fun[1] = func(r rune) int {
  switch(r) {
  case 116: return -1
  case 105: return -1
  case 103: return -1
  case 101: return -1
  case 69: return -1
  case 84: return -1
  case 114: return 2
  case 82: return 2
  case 73: return -1
  case 71: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 84: return -1
  case 114: return -1
  case 82: return -1
  case 73: return -1
  case 71: return 4
  case 116: return -1
  case 105: return -1
  case 103: return 4
  case 101: return -1
  case 69: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[4] = func(r rune) int {
  switch(r) {
  case 84: return -1
  case 114: return -1
  case 82: return -1
  case 73: return -1
  case 71: return 5
  case 116: return -1
  case 105: return -1
  case 103: return 5
  case 101: return -1
  case 69: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[0] = func(r rune) int {
  switch(r) {
  case 84: return 1
  case 114: return -1
  case 82: return -1
  case 73: return -1
  case 71: return -1
  case 116: return 1
  case 105: return -1
  case 103: return -1
  case 101: return -1
  case 69: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 116: return -1
  case 105: return 3
  case 103: return -1
  case 101: return -1
  case 69: return -1
  case 84: return -1
  case 114: return -1
  case 82: return -1
  case 73: return 3
  case 71: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[5] = func(r rune) int {
  switch(r) {
  case 116: return -1
  case 105: return -1
  case 103: return -1
  case 101: return 6
  case 69: return 6
  case 84: return -1
  case 114: return -1
  case 82: return -1
  case 73: return -1
  case 71: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[6] = func(r rune) int {
  switch(r) {
  case 116: return -1
  case 105: return -1
  case 103: return -1
  case 101: return -1
  case 69: return -1
  case 84: return -1
  case 114: return 7
  case 82: return 7
  case 73: return -1
  case 71: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[7] = true
fun[7] = func(r rune) int {
  switch(r) {
  case 116: return -1
  case 105: return -1
  case 103: return -1
  case 101: return -1
  case 69: return -1
  case 84: return -1
  case 114: return -1
  case 82: return -1
  case 73: return -1
  case 71: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[159].acc = acc[:]
a0[159].f = fun[:]
a0[159].id = 159
}
{
var acc [5]bool
var fun [5]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 116: return 1
  case 84: return 1
  case 114: return -1
  case 82: return -1
  case 117: return -1
  case 85: return -1
  case 101: return -1
  case 69: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 116: return -1
  case 84: return -1
  case 114: return 2
  case 82: return 2
  case 117: return -1
  case 85: return -1
  case 101: return -1
  case 69: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 116: return -1
  case 84: return -1
  case 114: return -1
  case 82: return -1
  case 117: return 3
  case 85: return 3
  case 101: return -1
  case 69: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 116: return -1
  case 84: return -1
  case 114: return -1
  case 82: return -1
  case 117: return -1
  case 85: return -1
  case 101: return 4
  case 69: return 4
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[4] = true
fun[4] = func(r rune) int {
  switch(r) {
  case 116: return -1
  case 84: return -1
  case 114: return -1
  case 82: return -1
  case 117: return -1
  case 85: return -1
  case 101: return -1
  case 69: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[160].acc = acc[:]
a0[160].f = fun[:]
a0[160].id = 160
}
{
var acc [9]bool
var fun [9]func(rune) int
fun[3] = func(r rune) int {
  switch(r) {
  case 116: return -1
  case 84: return -1
  case 114: return -1
  case 67: return -1
  case 101: return -1
  case 117: return -1
  case 110: return 4
  case 97: return -1
  case 69: return -1
  case 78: return 4
  case 65: return -1
  case 82: return -1
  case 85: return -1
  case 99: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[4] = func(r rune) int {
  switch(r) {
  case 82: return -1
  case 85: return -1
  case 99: return 5
  case 116: return -1
  case 84: return -1
  case 114: return -1
  case 67: return 5
  case 101: return -1
  case 117: return -1
  case 110: return -1
  case 97: return -1
  case 69: return -1
  case 78: return -1
  case 65: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[8] = true
fun[8] = func(r rune) int {
  switch(r) {
  case 116: return -1
  case 84: return -1
  case 114: return -1
  case 67: return -1
  case 101: return -1
  case 117: return -1
  case 110: return -1
  case 97: return -1
  case 69: return -1
  case 78: return -1
  case 65: return -1
  case 82: return -1
  case 85: return -1
  case 99: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[0] = func(r rune) int {
  switch(r) {
  case 82: return -1
  case 85: return -1
  case 99: return -1
  case 116: return 1
  case 84: return 1
  case 114: return -1
  case 67: return -1
  case 101: return -1
  case 117: return -1
  case 110: return -1
  case 97: return -1
  case 69: return -1
  case 78: return -1
  case 65: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 117: return -1
  case 110: return -1
  case 97: return -1
  case 69: return -1
  case 78: return -1
  case 65: return -1
  case 82: return 2
  case 85: return -1
  case 99: return -1
  case 116: return -1
  case 84: return -1
  case 114: return 2
  case 67: return -1
  case 101: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 117: return 3
  case 110: return -1
  case 97: return -1
  case 69: return -1
  case 78: return -1
  case 65: return -1
  case 82: return -1
  case 85: return 3
  case 99: return -1
  case 116: return -1
  case 84: return -1
  case 114: return -1
  case 67: return -1
  case 101: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[5] = func(r rune) int {
  switch(r) {
  case 117: return -1
  case 110: return -1
  case 97: return 6
  case 69: return -1
  case 78: return -1
  case 65: return 6
  case 82: return -1
  case 85: return -1
  case 99: return -1
  case 116: return -1
  case 84: return -1
  case 114: return -1
  case 67: return -1
  case 101: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[6] = func(r rune) int {
  switch(r) {
  case 117: return -1
  case 110: return -1
  case 97: return -1
  case 69: return -1
  case 78: return -1
  case 65: return -1
  case 82: return -1
  case 85: return -1
  case 99: return -1
  case 116: return 7
  case 84: return 7
  case 114: return -1
  case 67: return -1
  case 101: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[7] = func(r rune) int {
  switch(r) {
  case 117: return -1
  case 110: return -1
  case 97: return -1
  case 69: return 8
  case 78: return -1
  case 65: return -1
  case 82: return -1
  case 85: return -1
  case 99: return -1
  case 116: return -1
  case 84: return -1
  case 114: return -1
  case 67: return -1
  case 101: return 8
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[161].acc = acc[:]
a0[161].f = fun[:]
a0[161].id = 161
}
{
var acc [5]bool
var fun [5]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 116: return 1
  case 84: return 1
  case 121: return -1
  case 89: return -1
  case 112: return -1
  case 80: return -1
  case 101: return -1
  case 69: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 116: return -1
  case 84: return -1
  case 121: return 2
  case 89: return 2
  case 112: return -1
  case 80: return -1
  case 101: return -1
  case 69: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 116: return -1
  case 84: return -1
  case 121: return -1
  case 89: return -1
  case 112: return 3
  case 80: return 3
  case 101: return -1
  case 69: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 116: return -1
  case 84: return -1
  case 121: return -1
  case 89: return -1
  case 112: return -1
  case 80: return -1
  case 101: return 4
  case 69: return 4
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[4] = true
fun[4] = func(r rune) int {
  switch(r) {
  case 116: return -1
  case 84: return -1
  case 121: return -1
  case 89: return -1
  case 112: return -1
  case 80: return -1
  case 101: return -1
  case 69: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[162].acc = acc[:]
a0[162].f = fun[:]
a0[162].id = 162
}
{
var acc [6]bool
var fun [6]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 117: return 1
  case 85: return 1
  case 100: return -1
  case 68: return -1
  case 114: return -1
  case 110: return -1
  case 78: return -1
  case 101: return -1
  case 69: return -1
  case 82: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 110: return 2
  case 78: return 2
  case 101: return -1
  case 69: return -1
  case 82: return -1
  case 117: return -1
  case 85: return -1
  case 100: return -1
  case 68: return -1
  case 114: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 110: return -1
  case 78: return -1
  case 101: return -1
  case 69: return -1
  case 82: return -1
  case 117: return -1
  case 85: return -1
  case 100: return 3
  case 68: return 3
  case 114: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 117: return -1
  case 85: return -1
  case 100: return -1
  case 68: return -1
  case 114: return -1
  case 110: return -1
  case 78: return -1
  case 101: return 4
  case 69: return 4
  case 82: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[4] = func(r rune) int {
  switch(r) {
  case 110: return -1
  case 78: return -1
  case 101: return -1
  case 69: return -1
  case 82: return 5
  case 117: return -1
  case 85: return -1
  case 100: return -1
  case 68: return -1
  case 114: return 5
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[5] = true
fun[5] = func(r rune) int {
  switch(r) {
  case 117: return -1
  case 85: return -1
  case 100: return -1
  case 68: return -1
  case 114: return -1
  case 110: return -1
  case 78: return -1
  case 101: return -1
  case 69: return -1
  case 82: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[163].acc = acc[:]
a0[163].f = fun[:]
a0[163].id = 163
}
{
var acc [6]bool
var fun [6]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 117: return 1
  case 85: return 1
  case 110: return -1
  case 78: return -1
  case 105: return -1
  case 73: return -1
  case 111: return -1
  case 79: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 117: return -1
  case 85: return -1
  case 110: return 2
  case 78: return 2
  case 105: return -1
  case 73: return -1
  case 111: return -1
  case 79: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 117: return -1
  case 85: return -1
  case 110: return -1
  case 78: return -1
  case 105: return 3
  case 73: return 3
  case 111: return -1
  case 79: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 117: return -1
  case 85: return -1
  case 110: return -1
  case 78: return -1
  case 105: return -1
  case 73: return -1
  case 111: return 4
  case 79: return 4
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[4] = func(r rune) int {
  switch(r) {
  case 117: return -1
  case 85: return -1
  case 110: return 5
  case 78: return 5
  case 105: return -1
  case 73: return -1
  case 111: return -1
  case 79: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[5] = true
fun[5] = func(r rune) int {
  switch(r) {
  case 117: return -1
  case 85: return -1
  case 110: return -1
  case 78: return -1
  case 105: return -1
  case 73: return -1
  case 111: return -1
  case 79: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[164].acc = acc[:]
a0[164].f = fun[:]
a0[164].id = 164
}
{
var acc [7]bool
var fun [7]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 117: return 1
  case 78: return -1
  case 73: return -1
  case 81: return -1
  case 101: return -1
  case 85: return 1
  case 110: return -1
  case 105: return -1
  case 113: return -1
  case 69: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 117: return -1
  case 78: return 2
  case 73: return -1
  case 81: return -1
  case 101: return -1
  case 85: return -1
  case 110: return 2
  case 105: return -1
  case 113: return -1
  case 69: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 117: return -1
  case 78: return -1
  case 73: return 3
  case 81: return -1
  case 101: return -1
  case 85: return -1
  case 110: return -1
  case 105: return 3
  case 113: return -1
  case 69: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 117: return -1
  case 78: return -1
  case 73: return -1
  case 81: return 4
  case 101: return -1
  case 85: return -1
  case 110: return -1
  case 105: return -1
  case 113: return 4
  case 69: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[4] = func(r rune) int {
  switch(r) {
  case 117: return 5
  case 78: return -1
  case 73: return -1
  case 81: return -1
  case 101: return -1
  case 85: return 5
  case 110: return -1
  case 105: return -1
  case 113: return -1
  case 69: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[5] = func(r rune) int {
  switch(r) {
  case 85: return -1
  case 110: return -1
  case 105: return -1
  case 113: return -1
  case 69: return 6
  case 117: return -1
  case 78: return -1
  case 73: return -1
  case 81: return -1
  case 101: return 6
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[6] = true
fun[6] = func(r rune) int {
  switch(r) {
  case 117: return -1
  case 78: return -1
  case 73: return -1
  case 81: return -1
  case 101: return -1
  case 85: return -1
  case 110: return -1
  case 105: return -1
  case 113: return -1
  case 69: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[165].acc = acc[:]
a0[165].f = fun[:]
a0[165].id = 165
}
{
var acc [7]bool
var fun [7]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 117: return 1
  case 85: return 1
  case 110: return -1
  case 115: return -1
  case 84: return -1
  case 78: return -1
  case 101: return -1
  case 69: return -1
  case 83: return -1
  case 116: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 117: return -1
  case 85: return -1
  case 110: return 2
  case 115: return -1
  case 84: return -1
  case 78: return 2
  case 101: return -1
  case 69: return -1
  case 83: return -1
  case 116: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 78: return 3
  case 101: return -1
  case 69: return -1
  case 83: return -1
  case 116: return -1
  case 117: return -1
  case 85: return -1
  case 110: return 3
  case 115: return -1
  case 84: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 78: return -1
  case 101: return 4
  case 69: return 4
  case 83: return -1
  case 116: return -1
  case 117: return -1
  case 85: return -1
  case 110: return -1
  case 115: return -1
  case 84: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[4] = func(r rune) int {
  switch(r) {
  case 117: return -1
  case 85: return -1
  case 110: return -1
  case 115: return 5
  case 84: return -1
  case 78: return -1
  case 101: return -1
  case 69: return -1
  case 83: return 5
  case 116: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[5] = func(r rune) int {
  switch(r) {
  case 78: return -1
  case 101: return -1
  case 69: return -1
  case 83: return -1
  case 116: return 6
  case 117: return -1
  case 85: return -1
  case 110: return -1
  case 115: return -1
  case 84: return 6
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[6] = true
fun[6] = func(r rune) int {
  switch(r) {
  case 117: return -1
  case 85: return -1
  case 110: return -1
  case 115: return -1
  case 84: return -1
  case 78: return -1
  case 101: return -1
  case 69: return -1
  case 83: return -1
  case 116: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[166].acc = acc[:]
a0[166].f = fun[:]
a0[166].id = 166
}
{
var acc [6]bool
var fun [6]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 85: return 1
  case 78: return -1
  case 115: return -1
  case 83: return -1
  case 116: return -1
  case 117: return 1
  case 110: return -1
  case 101: return -1
  case 69: return -1
  case 84: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 117: return -1
  case 110: return 2
  case 101: return -1
  case 69: return -1
  case 84: return -1
  case 85: return -1
  case 78: return 2
  case 115: return -1
  case 83: return -1
  case 116: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 117: return -1
  case 110: return -1
  case 101: return -1
  case 69: return -1
  case 84: return -1
  case 85: return -1
  case 78: return -1
  case 115: return 3
  case 83: return 3
  case 116: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 117: return -1
  case 110: return -1
  case 101: return 4
  case 69: return 4
  case 84: return -1
  case 85: return -1
  case 78: return -1
  case 115: return -1
  case 83: return -1
  case 116: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[4] = func(r rune) int {
  switch(r) {
  case 85: return -1
  case 78: return -1
  case 115: return -1
  case 83: return -1
  case 116: return 5
  case 117: return -1
  case 110: return -1
  case 101: return -1
  case 69: return -1
  case 84: return 5
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[5] = true
fun[5] = func(r rune) int {
  switch(r) {
  case 117: return -1
  case 110: return -1
  case 101: return -1
  case 69: return -1
  case 84: return -1
  case 85: return -1
  case 78: return -1
  case 115: return -1
  case 83: return -1
  case 116: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[167].acc = acc[:]
a0[167].f = fun[:]
a0[167].id = 167
}
{
var acc [7]bool
var fun [7]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 85: return 1
  case 80: return -1
  case 100: return -1
  case 68: return -1
  case 97: return -1
  case 65: return -1
  case 116: return -1
  case 69: return -1
  case 117: return 1
  case 112: return -1
  case 84: return -1
  case 101: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 85: return -1
  case 80: return 2
  case 100: return -1
  case 68: return -1
  case 97: return -1
  case 65: return -1
  case 116: return -1
  case 69: return -1
  case 117: return -1
  case 112: return 2
  case 84: return -1
  case 101: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 85: return -1
  case 80: return -1
  case 100: return 3
  case 68: return 3
  case 97: return -1
  case 65: return -1
  case 116: return -1
  case 69: return -1
  case 117: return -1
  case 112: return -1
  case 84: return -1
  case 101: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 85: return -1
  case 80: return -1
  case 100: return -1
  case 68: return -1
  case 97: return 4
  case 65: return 4
  case 116: return -1
  case 69: return -1
  case 117: return -1
  case 112: return -1
  case 84: return -1
  case 101: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[4] = func(r rune) int {
  switch(r) {
  case 117: return -1
  case 112: return -1
  case 84: return 5
  case 101: return -1
  case 85: return -1
  case 80: return -1
  case 100: return -1
  case 68: return -1
  case 97: return -1
  case 65: return -1
  case 116: return 5
  case 69: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[5] = func(r rune) int {
  switch(r) {
  case 117: return -1
  case 112: return -1
  case 84: return -1
  case 101: return 6
  case 85: return -1
  case 80: return -1
  case 100: return -1
  case 68: return -1
  case 97: return -1
  case 65: return -1
  case 116: return -1
  case 69: return 6
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[6] = true
fun[6] = func(r rune) int {
  switch(r) {
  case 117: return -1
  case 112: return -1
  case 84: return -1
  case 101: return -1
  case 85: return -1
  case 80: return -1
  case 100: return -1
  case 68: return -1
  case 97: return -1
  case 65: return -1
  case 116: return -1
  case 69: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[168].acc = acc[:]
a0[168].f = fun[:]
a0[168].id = 168
}
{
var acc [7]bool
var fun [7]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 117: return 1
  case 80: return -1
  case 115: return -1
  case 83: return -1
  case 116: return -1
  case 85: return 1
  case 112: return -1
  case 101: return -1
  case 69: return -1
  case 114: return -1
  case 82: return -1
  case 84: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 85: return -1
  case 112: return 2
  case 101: return -1
  case 69: return -1
  case 114: return -1
  case 82: return -1
  case 84: return -1
  case 117: return -1
  case 80: return 2
  case 115: return -1
  case 83: return -1
  case 116: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 85: return -1
  case 112: return -1
  case 101: return -1
  case 69: return -1
  case 114: return -1
  case 82: return -1
  case 84: return -1
  case 117: return -1
  case 80: return -1
  case 115: return 3
  case 83: return 3
  case 116: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 85: return -1
  case 112: return -1
  case 101: return 4
  case 69: return 4
  case 114: return -1
  case 82: return -1
  case 84: return -1
  case 117: return -1
  case 80: return -1
  case 115: return -1
  case 83: return -1
  case 116: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[4] = func(r rune) int {
  switch(r) {
  case 117: return -1
  case 80: return -1
  case 115: return -1
  case 83: return -1
  case 116: return -1
  case 85: return -1
  case 112: return -1
  case 101: return -1
  case 69: return -1
  case 114: return 5
  case 82: return 5
  case 84: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[5] = func(r rune) int {
  switch(r) {
  case 117: return -1
  case 80: return -1
  case 115: return -1
  case 83: return -1
  case 116: return 6
  case 85: return -1
  case 112: return -1
  case 101: return -1
  case 69: return -1
  case 114: return -1
  case 82: return -1
  case 84: return 6
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[6] = true
fun[6] = func(r rune) int {
  switch(r) {
  case 117: return -1
  case 80: return -1
  case 115: return -1
  case 83: return -1
  case 116: return -1
  case 85: return -1
  case 112: return -1
  case 101: return -1
  case 69: return -1
  case 114: return -1
  case 82: return -1
  case 84: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[169].acc = acc[:]
a0[169].f = fun[:]
a0[169].id = 169
}
{
var acc [4]bool
var fun [4]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 117: return 1
  case 85: return 1
  case 115: return -1
  case 83: return -1
  case 101: return -1
  case 69: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 117: return -1
  case 85: return -1
  case 115: return 2
  case 83: return 2
  case 101: return -1
  case 69: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 117: return -1
  case 85: return -1
  case 115: return -1
  case 83: return -1
  case 101: return 3
  case 69: return 3
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[3] = true
fun[3] = func(r rune) int {
  switch(r) {
  case 117: return -1
  case 85: return -1
  case 115: return -1
  case 83: return -1
  case 101: return -1
  case 69: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[170].acc = acc[:]
a0[170].f = fun[:]
a0[170].id = 170
}
{
var acc [5]bool
var fun [5]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 117: return 1
  case 85: return 1
  case 115: return -1
  case 83: return -1
  case 101: return -1
  case 69: return -1
  case 114: return -1
  case 82: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 117: return -1
  case 85: return -1
  case 115: return 2
  case 83: return 2
  case 101: return -1
  case 69: return -1
  case 114: return -1
  case 82: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 117: return -1
  case 85: return -1
  case 115: return -1
  case 83: return -1
  case 101: return 3
  case 69: return 3
  case 114: return -1
  case 82: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 117: return -1
  case 85: return -1
  case 115: return -1
  case 83: return -1
  case 101: return -1
  case 69: return -1
  case 114: return 4
  case 82: return 4
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[4] = true
fun[4] = func(r rune) int {
  switch(r) {
  case 117: return -1
  case 85: return -1
  case 115: return -1
  case 83: return -1
  case 101: return -1
  case 69: return -1
  case 114: return -1
  case 82: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[171].acc = acc[:]
a0[171].f = fun[:]
a0[171].id = 171
}
{
var acc [6]bool
var fun [6]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 85: return 1
  case 105: return -1
  case 110: return -1
  case 117: return 1
  case 115: return -1
  case 83: return -1
  case 73: return -1
  case 78: return -1
  case 103: return -1
  case 71: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 85: return -1
  case 105: return -1
  case 110: return -1
  case 117: return -1
  case 115: return 2
  case 83: return 2
  case 73: return -1
  case 78: return -1
  case 103: return -1
  case 71: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 117: return -1
  case 115: return -1
  case 83: return -1
  case 73: return 3
  case 78: return -1
  case 103: return -1
  case 71: return -1
  case 85: return -1
  case 105: return 3
  case 110: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 85: return -1
  case 105: return -1
  case 110: return 4
  case 117: return -1
  case 115: return -1
  case 83: return -1
  case 73: return -1
  case 78: return 4
  case 103: return -1
  case 71: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[4] = func(r rune) int {
  switch(r) {
  case 85: return -1
  case 105: return -1
  case 110: return -1
  case 117: return -1
  case 115: return -1
  case 83: return -1
  case 73: return -1
  case 78: return -1
  case 103: return 5
  case 71: return 5
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[5] = true
fun[5] = func(r rune) int {
  switch(r) {
  case 117: return -1
  case 115: return -1
  case 83: return -1
  case 73: return -1
  case 78: return -1
  case 103: return -1
  case 71: return -1
  case 85: return -1
  case 105: return -1
  case 110: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[172].acc = acc[:]
a0[172].f = fun[:]
a0[172].id = 172
}
{
var acc [6]bool
var fun [6]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 97: return -1
  case 117: return -1
  case 101: return -1
  case 69: return -1
  case 118: return 1
  case 86: return 1
  case 65: return -1
  case 108: return -1
  case 76: return -1
  case 85: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 118: return -1
  case 86: return -1
  case 65: return 2
  case 108: return -1
  case 76: return -1
  case 85: return -1
  case 97: return 2
  case 117: return -1
  case 101: return -1
  case 69: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 118: return -1
  case 86: return -1
  case 65: return -1
  case 108: return 3
  case 76: return 3
  case 85: return -1
  case 97: return -1
  case 117: return -1
  case 101: return -1
  case 69: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 118: return -1
  case 86: return -1
  case 65: return -1
  case 108: return -1
  case 76: return -1
  case 85: return 4
  case 97: return -1
  case 117: return 4
  case 101: return -1
  case 69: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[4] = func(r rune) int {
  switch(r) {
  case 118: return -1
  case 86: return -1
  case 65: return -1
  case 108: return -1
  case 76: return -1
  case 85: return -1
  case 97: return -1
  case 117: return -1
  case 101: return 5
  case 69: return 5
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[5] = true
fun[5] = func(r rune) int {
  switch(r) {
  case 97: return -1
  case 117: return -1
  case 101: return -1
  case 69: return -1
  case 118: return -1
  case 86: return -1
  case 65: return -1
  case 108: return -1
  case 76: return -1
  case 85: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[173].acc = acc[:]
a0[173].f = fun[:]
a0[173].id = 173
}
{
var acc [7]bool
var fun [7]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 118: return 1
  case 97: return -1
  case 65: return -1
  case 117: return -1
  case 101: return -1
  case 69: return -1
  case 86: return 1
  case 108: return -1
  case 76: return -1
  case 85: return -1
  case 100: return -1
  case 68: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 86: return -1
  case 108: return -1
  case 76: return -1
  case 85: return -1
  case 100: return -1
  case 68: return -1
  case 118: return -1
  case 97: return 2
  case 65: return 2
  case 117: return -1
  case 101: return -1
  case 69: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 86: return -1
  case 108: return 3
  case 76: return 3
  case 85: return -1
  case 100: return -1
  case 68: return -1
  case 118: return -1
  case 97: return -1
  case 65: return -1
  case 117: return -1
  case 101: return -1
  case 69: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 118: return -1
  case 97: return -1
  case 65: return -1
  case 117: return 4
  case 101: return -1
  case 69: return -1
  case 86: return -1
  case 108: return -1
  case 76: return -1
  case 85: return 4
  case 100: return -1
  case 68: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[4] = func(r rune) int {
  switch(r) {
  case 86: return -1
  case 108: return -1
  case 76: return -1
  case 85: return -1
  case 100: return -1
  case 68: return -1
  case 118: return -1
  case 97: return -1
  case 65: return -1
  case 117: return -1
  case 101: return 5
  case 69: return 5
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[5] = func(r rune) int {
  switch(r) {
  case 118: return -1
  case 97: return -1
  case 65: return -1
  case 117: return -1
  case 101: return -1
  case 69: return -1
  case 86: return -1
  case 108: return -1
  case 76: return -1
  case 85: return -1
  case 100: return 6
  case 68: return 6
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[6] = true
fun[6] = func(r rune) int {
  switch(r) {
  case 86: return -1
  case 108: return -1
  case 76: return -1
  case 85: return -1
  case 100: return -1
  case 68: return -1
  case 118: return -1
  case 97: return -1
  case 65: return -1
  case 117: return -1
  case 101: return -1
  case 69: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[174].acc = acc[:]
a0[174].f = fun[:]
a0[174].id = 174
}
{
var acc [7]bool
var fun [7]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 118: return 1
  case 86: return 1
  case 97: return -1
  case 65: return -1
  case 115: return -1
  case 83: return -1
  case 108: return -1
  case 76: return -1
  case 117: return -1
  case 85: return -1
  case 101: return -1
  case 69: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 118: return -1
  case 86: return -1
  case 97: return 2
  case 65: return 2
  case 115: return -1
  case 83: return -1
  case 108: return -1
  case 76: return -1
  case 117: return -1
  case 85: return -1
  case 101: return -1
  case 69: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 108: return 3
  case 76: return 3
  case 117: return -1
  case 85: return -1
  case 101: return -1
  case 69: return -1
  case 118: return -1
  case 86: return -1
  case 97: return -1
  case 65: return -1
  case 115: return -1
  case 83: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 118: return -1
  case 86: return -1
  case 97: return -1
  case 65: return -1
  case 115: return -1
  case 83: return -1
  case 108: return -1
  case 76: return -1
  case 117: return 4
  case 85: return 4
  case 101: return -1
  case 69: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[4] = func(r rune) int {
  switch(r) {
  case 108: return -1
  case 76: return -1
  case 117: return -1
  case 85: return -1
  case 101: return 5
  case 69: return 5
  case 118: return -1
  case 86: return -1
  case 97: return -1
  case 65: return -1
  case 115: return -1
  case 83: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[5] = func(r rune) int {
  switch(r) {
  case 108: return -1
  case 76: return -1
  case 117: return -1
  case 85: return -1
  case 101: return -1
  case 69: return -1
  case 118: return -1
  case 86: return -1
  case 97: return -1
  case 65: return -1
  case 115: return 6
  case 83: return 6
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[6] = true
fun[6] = func(r rune) int {
  switch(r) {
  case 118: return -1
  case 86: return -1
  case 97: return -1
  case 65: return -1
  case 115: return -1
  case 83: return -1
  case 108: return -1
  case 76: return -1
  case 117: return -1
  case 85: return -1
  case 101: return -1
  case 69: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[175].acc = acc[:]
a0[175].f = fun[:]
a0[175].id = 175
}
{
var acc [5]bool
var fun [5]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 118: return 1
  case 86: return 1
  case 105: return -1
  case 73: return -1
  case 101: return -1
  case 69: return -1
  case 119: return -1
  case 87: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 118: return -1
  case 86: return -1
  case 105: return 2
  case 73: return 2
  case 101: return -1
  case 69: return -1
  case 119: return -1
  case 87: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 118: return -1
  case 86: return -1
  case 105: return -1
  case 73: return -1
  case 101: return 3
  case 69: return 3
  case 119: return -1
  case 87: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 118: return -1
  case 86: return -1
  case 105: return -1
  case 73: return -1
  case 101: return -1
  case 69: return -1
  case 119: return 4
  case 87: return 4
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[4] = true
fun[4] = func(r rune) int {
  switch(r) {
  case 118: return -1
  case 86: return -1
  case 105: return -1
  case 73: return -1
  case 101: return -1
  case 69: return -1
  case 119: return -1
  case 87: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[176].acc = acc[:]
a0[176].f = fun[:]
a0[176].id = 176
}
{
var acc [5]bool
var fun [5]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 119: return 1
  case 87: return 1
  case 104: return -1
  case 72: return -1
  case 101: return -1
  case 69: return -1
  case 110: return -1
  case 78: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 119: return -1
  case 87: return -1
  case 104: return 2
  case 72: return 2
  case 101: return -1
  case 69: return -1
  case 110: return -1
  case 78: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 119: return -1
  case 87: return -1
  case 104: return -1
  case 72: return -1
  case 101: return 3
  case 69: return 3
  case 110: return -1
  case 78: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 119: return -1
  case 87: return -1
  case 104: return -1
  case 72: return -1
  case 101: return -1
  case 69: return -1
  case 110: return 4
  case 78: return 4
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[4] = true
fun[4] = func(r rune) int {
  switch(r) {
  case 119: return -1
  case 87: return -1
  case 104: return -1
  case 72: return -1
  case 101: return -1
  case 69: return -1
  case 110: return -1
  case 78: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[177].acc = acc[:]
a0[177].f = fun[:]
a0[177].id = 177
}
{
var acc [6]bool
var fun [6]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 119: return 1
  case 87: return 1
  case 104: return -1
  case 72: return -1
  case 101: return -1
  case 69: return -1
  case 114: return -1
  case 82: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 119: return -1
  case 87: return -1
  case 104: return 2
  case 72: return 2
  case 101: return -1
  case 69: return -1
  case 114: return -1
  case 82: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 119: return -1
  case 87: return -1
  case 104: return -1
  case 72: return -1
  case 101: return 3
  case 69: return 3
  case 114: return -1
  case 82: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 119: return -1
  case 87: return -1
  case 104: return -1
  case 72: return -1
  case 101: return -1
  case 69: return -1
  case 114: return 4
  case 82: return 4
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[4] = func(r rune) int {
  switch(r) {
  case 119: return -1
  case 87: return -1
  case 104: return -1
  case 72: return -1
  case 101: return 5
  case 69: return 5
  case 114: return -1
  case 82: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[5] = true
fun[5] = func(r rune) int {
  switch(r) {
  case 119: return -1
  case 87: return -1
  case 104: return -1
  case 72: return -1
  case 101: return -1
  case 69: return -1
  case 114: return -1
  case 82: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[178].acc = acc[:]
a0[178].f = fun[:]
a0[178].id = 178
}
{
var acc [6]bool
var fun [6]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 87: return 1
  case 104: return -1
  case 72: return -1
  case 105: return -1
  case 108: return -1
  case 101: return -1
  case 119: return 1
  case 73: return -1
  case 76: return -1
  case 69: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 119: return -1
  case 73: return -1
  case 76: return -1
  case 69: return -1
  case 87: return -1
  case 104: return 2
  case 72: return 2
  case 105: return -1
  case 108: return -1
  case 101: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 87: return -1
  case 104: return -1
  case 72: return -1
  case 105: return 3
  case 108: return -1
  case 101: return -1
  case 119: return -1
  case 73: return 3
  case 76: return -1
  case 69: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 87: return -1
  case 104: return -1
  case 72: return -1
  case 105: return -1
  case 108: return 4
  case 101: return -1
  case 119: return -1
  case 73: return -1
  case 76: return 4
  case 69: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[4] = func(r rune) int {
  switch(r) {
  case 87: return -1
  case 104: return -1
  case 72: return -1
  case 105: return -1
  case 108: return -1
  case 101: return 5
  case 119: return -1
  case 73: return -1
  case 76: return -1
  case 69: return 5
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[5] = true
fun[5] = func(r rune) int {
  switch(r) {
  case 119: return -1
  case 73: return -1
  case 76: return -1
  case 69: return -1
  case 87: return -1
  case 104: return -1
  case 72: return -1
  case 105: return -1
  case 108: return -1
  case 101: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[179].acc = acc[:]
a0[179].f = fun[:]
a0[179].id = 179
}
{
var acc [5]bool
var fun [5]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 119: return 1
  case 87: return 1
  case 105: return -1
  case 73: return -1
  case 116: return -1
  case 84: return -1
  case 104: return -1
  case 72: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 119: return -1
  case 87: return -1
  case 105: return 2
  case 73: return 2
  case 116: return -1
  case 84: return -1
  case 104: return -1
  case 72: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 119: return -1
  case 87: return -1
  case 105: return -1
  case 73: return -1
  case 116: return 3
  case 84: return 3
  case 104: return -1
  case 72: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 119: return -1
  case 87: return -1
  case 105: return -1
  case 73: return -1
  case 116: return -1
  case 84: return -1
  case 104: return 4
  case 72: return 4
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[4] = true
fun[4] = func(r rune) int {
  switch(r) {
  case 119: return -1
  case 87: return -1
  case 105: return -1
  case 73: return -1
  case 116: return -1
  case 84: return -1
  case 104: return -1
  case 72: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[180].acc = acc[:]
a0[180].f = fun[:]
a0[180].id = 180
}
{
var acc [7]bool
var fun [7]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 73: return -1
  case 84: return -1
  case 72: return -1
  case 110: return -1
  case 78: return -1
  case 119: return 1
  case 87: return 1
  case 105: return -1
  case 116: return -1
  case 104: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 73: return 2
  case 84: return -1
  case 72: return -1
  case 110: return -1
  case 78: return -1
  case 119: return -1
  case 87: return -1
  case 105: return 2
  case 116: return -1
  case 104: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 73: return -1
  case 84: return 3
  case 72: return -1
  case 110: return -1
  case 78: return -1
  case 119: return -1
  case 87: return -1
  case 105: return -1
  case 116: return 3
  case 104: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 73: return -1
  case 84: return -1
  case 72: return 4
  case 110: return -1
  case 78: return -1
  case 119: return -1
  case 87: return -1
  case 105: return -1
  case 116: return -1
  case 104: return 4
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[4] = func(r rune) int {
  switch(r) {
  case 73: return 5
  case 84: return -1
  case 72: return -1
  case 110: return -1
  case 78: return -1
  case 119: return -1
  case 87: return -1
  case 105: return 5
  case 116: return -1
  case 104: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[5] = func(r rune) int {
  switch(r) {
  case 119: return -1
  case 87: return -1
  case 105: return -1
  case 116: return -1
  case 104: return -1
  case 73: return -1
  case 84: return -1
  case 72: return -1
  case 110: return 6
  case 78: return 6
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[6] = true
fun[6] = func(r rune) int {
  switch(r) {
  case 119: return -1
  case 87: return -1
  case 105: return -1
  case 116: return -1
  case 104: return -1
  case 73: return -1
  case 84: return -1
  case 72: return -1
  case 110: return -1
  case 78: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[181].acc = acc[:]
a0[181].f = fun[:]
a0[181].id = 181
}
{
var acc [5]bool
var fun [5]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 119: return 1
  case 87: return 1
  case 111: return -1
  case 79: return -1
  case 114: return -1
  case 82: return -1
  case 107: return -1
  case 75: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 119: return -1
  case 87: return -1
  case 111: return 2
  case 79: return 2
  case 114: return -1
  case 82: return -1
  case 107: return -1
  case 75: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 119: return -1
  case 87: return -1
  case 111: return -1
  case 79: return -1
  case 114: return 3
  case 82: return 3
  case 107: return -1
  case 75: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[3] = func(r rune) int {
  switch(r) {
  case 119: return -1
  case 87: return -1
  case 111: return -1
  case 79: return -1
  case 114: return -1
  case 82: return -1
  case 107: return 4
  case 75: return 4
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[4] = true
fun[4] = func(r rune) int {
  switch(r) {
  case 119: return -1
  case 87: return -1
  case 111: return -1
  case 79: return -1
  case 114: return -1
  case 82: return -1
  case 107: return -1
  case 75: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[182].acc = acc[:]
a0[182].f = fun[:]
a0[182].id = 182
}
{
var acc [4]bool
var fun [4]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 120: return 1
  case 88: return 1
  case 111: return -1
  case 79: return -1
  case 114: return -1
  case 82: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 120: return -1
  case 88: return -1
  case 111: return 2
  case 79: return 2
  case 114: return -1
  case 82: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
fun[2] = func(r rune) int {
  switch(r) {
  case 120: return -1
  case 88: return -1
  case 111: return -1
  case 79: return -1
  case 114: return 3
  case 82: return 3
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
acc[3] = true
fun[3] = func(r rune) int {
  switch(r) {
  case 120: return -1
  case 88: return -1
  case 111: return -1
  case 79: return -1
  case 114: return -1
  case 82: return -1
  default:
    switch {
    default: return -1
    }
  }
  panic("unreachable")
}
a0[183].acc = acc[:]
a0[183].f = fun[:]
a0[183].id = 183
}
{
var acc [3]bool
var fun [3]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 95: return 1
  default:
    switch {
    case 48 <= r && r <= 57: return -1
    case 65 <= r && r <= 90: return 1
    case 97 <= r && r <= 122: return 1
    default: return -1
    }
  }
  panic("unreachable")
}
acc[1] = true
fun[1] = func(r rune) int {
  switch(r) {
  case 95: return 2
  default:
    switch {
    case 48 <= r && r <= 57: return 2
    case 65 <= r && r <= 90: return 2
    case 97 <= r && r <= 122: return 2
    default: return -1
    }
  }
  panic("unreachable")
}
acc[2] = true
fun[2] = func(r rune) int {
  switch(r) {
  case 95: return 2
  default:
    switch {
    case 48 <= r && r <= 57: return 2
    case 65 <= r && r <= 90: return 2
    case 97 <= r && r <= 122: return 2
    default: return -1
    }
  }
  panic("unreachable")
}
a0[184].acc = acc[:]
a0[184].f = fun[:]
a0[184].id = 184
}
{
var acc [4]bool
var fun [4]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 36: return 1
  case 95: return -1
  default:
    switch {
    case 48 <= r && r <= 57: return -1
    case 65 <= r && r <= 90: return -1
    case 97 <= r && r <= 122: return -1
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 36: return -1
  case 95: return 2
  default:
    switch {
    case 48 <= r && r <= 57: return -1
    case 65 <= r && r <= 90: return 2
    case 97 <= r && r <= 122: return 2
    default: return -1
    }
  }
  panic("unreachable")
}
acc[2] = true
fun[2] = func(r rune) int {
  switch(r) {
  case 36: return -1
  case 95: return 3
  default:
    switch {
    case 48 <= r && r <= 57: return 3
    case 65 <= r && r <= 90: return 3
    case 97 <= r && r <= 122: return 3
    default: return -1
    }
  }
  panic("unreachable")
}
acc[3] = true
fun[3] = func(r rune) int {
  switch(r) {
  case 36: return -1
  case 95: return 3
  default:
    switch {
    case 48 <= r && r <= 57: return 3
    case 65 <= r && r <= 90: return 3
    case 97 <= r && r <= 122: return 3
    default: return -1
    }
  }
  panic("unreachable")
}
a0[185].acc = acc[:]
a0[185].f = fun[:]
a0[185].id = 185
}
{
var acc [4]bool
var fun [4]func(rune) int
fun[0] = func(r rune) int {
  switch(r) {
  case 36: return 1
  default:
    switch {
    case 48 <= r && r <= 48: return -1
    case 49 <= r && r <= 57: return -1
    default: return -1
    }
  }
  panic("unreachable")
}
fun[1] = func(r rune) int {
  switch(r) {
  case 36: return -1
  default:
    switch {
    case 48 <= r && r <= 48: return -1
    case 49 <= r && r <= 57: return 2
    default: return -1
    }
  }
  panic("unreachable")
}
acc[2] = true
fun[2] = func(r rune) int {
  switch(r) {
  case 36: return -1
  default:
    switch {
    case 48 <= r && r <= 48: return 3
    case 49 <= r && r <= 57: return 3
    default: return -1
    }
  }
  panic("unreachable")
}
acc[3] = true
fun[3] = func(r rune) int {
  switch(r) {
  case 36: return -1
  default:
    switch {
    case 48 <= r && r <= 48: return 3
    case 49 <= r && r <= 57: return 3
    default: return -1
    }
  }
  panic("unreachable")
}
a0[186].acc = acc[:]
a0[186].f = fun[:]
a0[186].id = 186
}
a[0].endcase = 187
a[0].a = a0[:]
}
func getAction(c *frame) int {
  if -1 == c.match { return -1 }
  c.action = c.fam.a[c.match].id
  c.match = -1
  return c.action
}
type frame struct {
  atEOF bool
  action, match, matchn, n int
  buf []rune
  text string
  in *bufio.Reader
  state []int
  fam family
}
func newFrame(in *bufio.Reader, index int) *frame {
  f := new(frame)
  f.buf = make([]rune, 0, 128)
  f.in = in
  f.match = -1
  f.fam = a[index]
  f.state = make([]int, len(f.fam.a))
  return f
}
type Lexer []*frame
func NewLexer(in io.Reader) Lexer {
  stack := make([]*frame, 0, 4)
  stack = append(stack, newFrame(bufio.NewReader(in), 0))
  return stack
}
func (stack Lexer) isDone() bool {
  return 1 == len(stack) && stack[0].atEOF
}
func (stack Lexer) nextAction() int {
  c := stack[len(stack) - 1]
  for {
    if c.atEOF { return c.fam.endcase }
    if c.n == len(c.buf) {
      r,_,er := c.in.ReadRune()
      switch er {
      case nil: c.buf = append(c.buf, r)
      case io.EOF:
	c.atEOF = true
	if c.n > 0 {
	  c.text = string(c.buf)
	  return getAction(c)
	}
	return c.fam.endcase
      default: panic(er.Error())
      }
    }
    jammed := true
    r := c.buf[c.n]
    for i, x := range c.fam.a {
      if -1 == c.state[i] { continue }
      c.state[i] = x.f[c.state[i]](r)
      if -1 == c.state[i] { continue }
      jammed = false
      if x.acc[c.state[i]] {
	if -1 == c.match || c.matchn < c.n+1 || c.match > i {
	  c.match = i
	  c.matchn = c.n+1
	}
      }
    }
    if jammed {
      a := getAction(c)
      if -1 == a { c.matchn = c.n + 1 }
      c.n = 0
      for i, _ := range c.state { c.state[i] = 0 }
      c.text = string(c.buf[:c.matchn])
      copy(c.buf, c.buf[c.matchn:])
      c.buf = c.buf[:len(c.buf) - c.matchn]
      return a
    }
    c.n++
  }
  panic("unreachable")
}
func (stack Lexer) push(index int) Lexer {
  c := stack[len(stack) - 1]
  return append(stack,
      newFrame(bufio.NewReader(strings.NewReader(c.text)), index))
}
func (stack Lexer) pop() Lexer {
  return stack[:len(stack) - 1]
}
func (stack Lexer) Text() string {
  c := stack[len(stack) - 1]
  return c.text
}
func (yylex Lexer) Error(e string) {
  panic(e)
}
func (yylex Lexer) Lex(lval *yySymType) int {
  for !yylex.isDone() {
    switch yylex.nextAction() {
    case -1:
    case 0:  //\"((\\\\)|(\\\")|(\\\/)|(\\b)|(\\f)|(\\n)|(\\r)|(\\t)|(\\u([0-9a-fA-F][0-9a-fA-F][0-9a-fA-F][0-9a-fA-F]){4})|[^\\\"])*\"/
{
		    lval.s, _ = UnmarshalDoubleQuoted(yylex.Text())
		    logToken("STRING - %s", lval.s)
		    return STRING
		  }
    case 1:  //'((\\\\)|(\\\")|(\\\/)|(\\b)|(\\f)|(\\n)|(\\r)|(\\t)|(\\u([0-9a-fA-F][0-9a-fA-F][0-9a-fA-F][0-9a-fA-F]){4})|('')|[^\\\"'])*'/
{
		    lval.s, _ = UnmarshalSingleQuoted(yylex.Text())
		    logToken("STRING - %s", lval.s)
		    return STRING
		  }
    case 2:  //`((\\\\)|(\\\")|(\\\/)|(\\b)|(\\f)|(\\n)|(\\r)|(\\t)|(\\u([0-9a-fA-F][0-9a-fA-F][0-9a-fA-F][0-9a-fA-F]){4})|(``)|[^\\\"`])+`i/
{
		    // Case-insensitive identifier
		    text := yylex.Text()
		    text = text[0 : len(text)-1]
		    lval.s, _ = UnmarshalBackQuoted(text)
		    logToken("IDENTIFIER_ICASE - %s", lval.s)
		    return IDENTIFIER_ICASE
		  }
    case 3:  //`((\\\\)|(\\\")|(\\\/)|(\\b)|(\\f)|(\\n)|(\\r)|(\\t)|(\\u([0-9a-fA-F][0-9a-fA-F][0-9a-fA-F][0-9a-fA-F]){4})|(``)|[^\\\"`])+`/
{
		    // Escaped identifier
		    lval.s, _ = UnmarshalBackQuoted(yylex.Text())
		    logToken("IDENTIFIER - %s", lval.s)
		    return IDENTIFIER
		  }
    case 4:  //(0|[1-9][0-9]*)\.[0-9]+([eE][+\-]?[0-9]+)?/
{
		  // We differentiate NUMBER from INT
		    lval.f,_ = strconv.ParseFloat(yylex.Text(), 64)
		    logToken("NUMBER - %f", lval.f)
		    return NUMBER
		  }
    case 5:  //(0|[1-9][0-9]*)[eE][+\-]?[0-9]+/
{
		  // We differentiate NUMBER from INT
		    lval.f, _ = strconv.ParseFloat(yylex.Text(), 64)
		    logToken("NUMBER - %f", lval.f)
		    return NUMBER
		  }
    case 6:  //0|[1-9][0-9]*/
{
		  // We differentiate NUMBER from INT
		    lval.n, _ = strconv.Atoi(yylex.Text())
		    logToken("INT - %d", lval.n)
		    return INT
		  }
    case 7:  //(\/\*)([^\*]|(\*)+[^\/])*((\*)+\/)/
{
		    logToken("BLOCK_COMMENT (length=%d)", len(yylex.Text())) /* eat up block comment */
		  }
    case 8:  //"--"[^\n\r]*/
{ logToken("LINE_COMMENT (length=%d)", len(yylex.Text())) /* eat up line comment */ }
    case 9:  //[ \t\n\r\f]+/
{ logToken("WHITESPACE (count=%d)", len(yylex.Text())) /* eat up whitespace */ }
    case 10:  //\./
{ logToken("DOT"); return DOT }
    case 11:  //\+/
{ logToken("PLUS"); return PLUS }
    case 12:  //-/
{ logToken("MINUS"); return MINUS }
    case 13:  //\*/
{ logToken("MULT"); return STAR }
    case 14:  //\//
{ logToken("DIV"); return DIV }
    case 15:  //%/
{ logToken("MOD"); return MOD }
    case 16:  //\=\=/
{ logToken("DEQ"); return DEQ }
    case 17:  //\=/
{ logToken("EQ"); return EQ }
    case 18:  //\!\=/
{ logToken("NE"); return NE }
    case 19:  //\<\>/
{ logToken("NE"); return NE }
    case 20:  //\</
{ logToken("LT"); return LT }
    case 21:  //\<\=/
{ logToken("LTE"); return LE }
    case 22:  //\>/
{ logToken("GT"); return GT }
    case 23:  //\>\=/
{ logToken("GTE"); return GE }
    case 24:  //\|\|/
{ logToken("CONCAT"); return CONCAT }
    case 25:  //\(/
{ logToken("LPAREN"); return LPAREN }
    case 26:  //\)/
{ logToken("RPAREN"); return RPAREN }
    case 27:  //\{/
{ logToken("LBRACE"); return LBRACE }
    case 28:  //\}/
{ logToken("RBRACE"); return RBRACE }
    case 29:  //\,/
{ logToken("COMMA"); return COMMA }
    case 30:  //\:/
{ logToken("COLON"); return COLON }
    case 31:  //\[/
{ logToken("LBRACKET"); return LBRACKET }
    case 32:  //\]/
{ logToken("RBRACKET"); return RBRACKET }
    case 33:  //\]i/
{ logToken("RBRACKET_ICASE"); return RBRACKET_ICASE }
    case 34:  //[aA][lL][lL]/
{ logToken("ALL"); return ALL }
    case 35:  //[aA][lL][tT][eE][rR]/
{ logToken("ALTER"); return ALTER }
    case 36:  //[aA][nN][aA][lL][yY][zZ][eE]/
{ logToken("ANALYZE"); return ANALYZE }
    case 37:  //[aA][nN][dD]/
{ logToken("AND"); return AND }
    case 38:  //[aA][nN][yY]/
{ logToken("ANY"); return ANY }
    case 39:  //[aA][rR][rR][aA][yY]/
{ logToken("ARRAY"); return ARRAY }
    case 40:  //[aA][sS]/
{ logToken("AS"); return AS }
    case 41:  //[aA][sS][cC]/
{ logToken("ASC"); return ASC }
    case 42:  //[bB][eE][gG][iI][nN]/
{ logToken("BEGIN"); return BEGIN }
    case 43:  //[bB][eE][tT][wW][eE][eE][nN]/
{ logToken("BETWEEN"); return BETWEEN }
    case 44:  //[bB][rR][eE][aA][kK]/
{ logToken("BREAK"); return BREAK }
    case 45:  //[bB][uU][cC][kK][eE][tT]/
{ logToken("BUCKET"); return BUCKET }
    case 46:  //[bB][yY]/
{ logToken("BY"); return BY }
    case 47:  //[cC][aA][lL][lL]/
{ logToken("CALL"); return CALL }
    case 48:  //[cC][aA][sS][eE]/
{ logToken("CASE"); return CASE }
    case 49:  //[cC][aA][sS][tT]/
{ logToken("CAST"); return CAST }
    case 50:  //[cC][lL][uU][sS][tT][eE][rR]/
{ logToken("CLUSTER"); return CLUSTER }
    case 51:  //[cC][oO][lL][lL][aA][tT][eE]/
{ logToken("COLLATE"); return COLLATE }
    case 52:  //[cC][oO][lL][lL][eE][cC][tT][iI][oO][nN]/
{ logToken("COLLECTION"); return COLLECTION }
    case 53:  //[cC][oO][mM][mM][iI][tT]/
{ logToken("COMMIT"); return COMMIT }
    case 54:  //[cC][oO][nN][nN][eE][cC][tT]/
{ logToken("CONNECT"); return CONNECT }
    case 55:  //[cC][oO][nN][tT][iI][nN][uU][eE]/
{ logToken("CONTINUE"); return CONTINUE }
    case 56:  //[cC][rR][eE][aA][tT][eE]/
{ logToken("CREATE"); return CREATE }
    case 57:  //[dD][aA][tT][aA][bB][aA][sS][eE]/
{ logToken("DATABASE"); return DATABASE }
    case 58:  //[dD][aA][tT][aA][sS][eE][tT]/
{ logToken("DATASET"); return DATASET }
    case 59:  //[dD][aA][tT][aA][sS][tT][oO][rR][eE]/
{ logToken("DATASTORE"); return DATASTORE }
    case 60:  //[dD][eE][cC][lL][aA][rR][eE]/
{ logToken("DECLARE"); return DECLARE }
    case 61:  //[dD][eE][lL][eE][tT][eE]/
{ logToken("DELETE"); return DELETE }
    case 62:  //[dD][eE][rR][iI][vV][eE][dD]/
{ logToken("DERIVED"); return DERIVED }
    case 63:  //[dD][eE][sS][cC]/
{ logToken("DESC"); return DESC }
    case 64:  //[dD][eE][sS][cC][rR][iI][bB][eE]/
{ logToken("DESCRIBE"); return DESCRIBE }
    case 65:  //[dD][iI][sS][tT][iI][nN][cC][tT]/
{ logToken("DISTINCT"); return DISTINCT }
    case 66:  //[dD][oO]/
{ logToken("DO"); return DO }
    case 67:  //[dD][rR][oO][pP]/
{ logToken("DROP"); return DROP }
    case 68:  //[eE][aA][cC][hH]/
{ logToken("EACH"); return EACH }
    case 69:  //[eE][lL][eE][mM][eE][nN][tT]/
{ logToken("ELEMENT"); return ELEMENT }
    case 70:  //[eE][lL][sS][eE]/
{ logToken("ELSE"); return ELSE }
    case 71:  //[eE][nN][dD]/
{ logToken("END"); return END }
    case 72:  //[eE][vV][eE][rR][yY]/
{ logToken("EVERY"); return EVERY }
    case 73:  //[eE][xX][cC][eE][pP][tT]/
{ logToken("EXCEPT"); return EXCEPT }
    case 74:  //[eE][xX][cC][lL][uU][dD][eE]/
{ logToken("EXCLUDE"); return EXCLUDE }
    case 75:  //[eE][xX][eE][cC][uU][tT][eE]/
{ logToken("EXECUTE"); return EXECUTE }
    case 76:  //[eE][xX][iI][sS][tT][sS]/
{ logToken("EXISTS"); return EXISTS }
    case 77:  //[eE][xX][pP][lL][aA][iI][nN]/
{ logToken("EXPLAIN"); return EXPLAIN }
    case 78:  //[fF][aA][lL][sS][eE]/
{ logToken("FALSE"); return FALSE }
    case 79:  //[fF][iI][rR][sS][tT]/
{ logToken("FIRST"); return FIRST }
    case 80:  //[fF][lL][aA][tT][tT][eE][nN]/
{ logToken("FLATTEN"); return FLATTEN }
    case 81:  //[fF][oO][rR]/
{ logToken("FOR"); return FOR }
    case 82:  //[fF][rR][oO][mM]/
{ logToken("FROM"); return FROM }
    case 83:  //[fF][uU][nN][cC][tT][iI][oO][nN]/
{ logToken("FUNCTION"); return FUNCTION }
    case 84:  //[gG][rR][aA][nN][tT]/
{ logToken("GRANT"); return GRANT }
    case 85:  //[gG][rR][oO][uU][pP]/
{ logToken("GROUP"); return GROUP }
    case 86:  //[hH][aA][vV][iI][nN][gG]/
{ logToken("HAVING"); return HAVING }
    case 87:  //[iI][fF]/
{ logToken("IF"); return IF }
    case 88:  //[iI][nN]/
{ logToken("IN"); return IN }
    case 89:  //[iI][nN][cC][lL][uU][dD][eE]/
{ logToken("INCLUDE"); return INCLUDE }
    case 90:  //[iI][nN][dD][eE][xX]/
{ logToken("INDEX"); return INDEX }
    case 91:  //[iI][nN][lL][iI][nN][eE]/
{ logToken("INLINE"); return INLINE }
    case 92:  //[iI][nN][nN][eE][rR]/
{ logToken("INNER"); return INNER }
    case 93:  //[iI][nN][sS][eE][rR][tT]/
{ logToken("INSERT"); return INSERT }
    case 94:  //[iI][nN][tT][eE][rR][sS][eE][cC][tT]/
{ logToken("INTERSECT"); return INTERSECT }
    case 95:  //[iI][nN][tT][oO]/
{ logToken("INTO"); return INTO }
    case 96:  //[iI][sS]/
{ logToken("IS"); return IS }
    case 97:  //[jJ][oO][iI][nN]/
{ logToken("JOIN"); return JOIN }
    case 98:  //[kK][eE][yY]/
{ logToken("KEY"); return KEY }
    case 99:  //[kK][eE][yY][sS]/
{ logToken("KEYS"); return KEYS }
    case 100:  //[kK][eE][yY][sS][pP][aA][cC][eE]/
{ logToken("KEYSPACE"); return KEYSPACE }
    case 101:  //[lL][aA][sS][tT]/
{ logToken("LAST"); return LAST }
    case 102:  //[lL][eE][fF][tT]/
{ logToken("LEFT"); return LEFT }
    case 103:  //[lL][eE][tT]/
{ logToken("LET"); return LET }
    case 104:  //[lL][eE][tT][tT][iI][nN][gG]/
{ logToken("LETTING"); return LETTING }
    case 105:  //[lL][iI][kK][eE]/
{ logToken("LIKE"); return LIKE }
    case 106:  //[lL][iI][mM][iI][tT]/
{ logToken("LIMIT"); return LIMIT }
    case 107:  //[lL][sS][mM]/
{ logToken("LSM"); return LSM }
    case 108:  //[mM][aA][pP]/
{ logToken("MAP"); return MAP }
    case 109:  //[mM][aA][pP][pP][iI][nN][gG]/
{ logToken("MAPPING"); return MAPPING }
    case 110:  //[mM][aA][tT][cC][hH][eE][dD]/
{ logToken("MATCHED"); return MATCHED }
    case 111:  //[mM][aA][tT][eE][rR][iI][aA][lL][iI][zZ][eE][dD]/
{ logToken("MATERIALIZED"); return MATERIALIZED }
    case 112:  //[mM][eE][rR][gG][eE]/
{ logToken("MERGE"); return MERGE }
    case 113:  //[mM][iI][nN][uU][sS]/
{ logToken("MINUS"); return MINUS }
    case 114:  //[mM][iI][sS][sS][iI][nN][gG]/
{ logToken("MISSING"); return MISSING }
    case 115:  //[nN][aA][mM][eE][sS][pP][aA][cC][eE]/
{ logToken("NAMESPACE"); return NAMESPACE }
    case 116:  //[nN][eE][sS][tT]/
{ logToken("NEST"); return NEST }
    case 117:  //[nN][oO][tT]/
{ logToken("NOT"); return NOT }
    case 118:  //[nN][uU][lL][lL]/
{ logToken("NULL"); return NULL }
    case 119:  //[oO][bB][jJ][eE][cC][tT]/
{ logToken("OBJECT"); return OBJECT }
    case 120:  //[oO][fF][fF][sS][eE][tT]/
{ logToken("OFFSET"); return OFFSET }
    case 121:  //[oO][nN]/
{ logToken("ON"); return ON }
    case 122:  //[oO][pP][tT][iI][oO][nN]/
{ logToken("OPTION"); return OPTION }
    case 123:  //[oO][rR]/
{ logToken("OR"); return OR }
    case 124:  //[oO][rR][dD][eE][rR]/
{ logToken("ORDER"); return ORDER }
    case 125:  //[oO][uU][tT][eE][rR]/
{ logToken("OUTER"); return OUTER }
    case 126:  //[oO][vV][eE][rR]/
{ logToken("OVER"); return OVER }
    case 127:  //[pP][aA][rR][tT][iI][tT][iI][oO][nN]/
{ logToken("PARTITION"); return PARTITION }
    case 128:  //[pP][aA][sS][sS][wW][oO][rR][dD]/
{ logToken("PASSWORD"); return PASSWORD }
    case 129:  //[pP][aA][tT][hH]/
{ logToken("PATH"); return PATH }
    case 130:  //[pP][oO][oO][lL]/
{ logToken("POOL"); return POOL }
    case 131:  //[pP][rR][eE][pP][aA][rR][eE]/
{ logToken("PREPARE"); return PREPARE }
    case 132:  //[pP][rR][iI][mM][aA][rR][yY]/
{ logToken("PRIMARY"); return PRIMARY }
    case 133:  //[pP][rR][iI][vV][aA][tT][eE]/
{ logToken("PRIVATE"); return PRIVATE }
    case 134:  //[pP][rR][iI][vV][iI][lL][eE][gG][eE]/
{ logToken("PRIVILEGE"); return PRIVILEGE }
    case 135:  //[pP][rR][oO][cC][eE][dE][uU][rR][eE]/
{ logToken("PROCEDURE"); return PROCEDURE }
    case 136:  //[pP][uU][bB][lL][iI][cC]/
{ logToken("PUBLIC"); return PUBLIC }
    case 137:  //[rR][aA][wW]/
{ logToken("RAW"); return RAW }
    case 138:  //[rR][eE][aA][lL][mM]/
{ logToken("REALM"); return REALM }
    case 139:  //[rR][eE][dD][uU][cC][eE]/
{ logToken("REDUCE"); return REDUCE }
    case 140:  //[rR][eE][nN][aA][mM][eE]/
{ logToken("RENAME"); return RENAME }
    case 141:  //[rR][eE][tT][uU][rR][nN]/
{ logToken("RETURN"); return RETURN }
    case 142:  //[rR][eE][tT][uU][rR][nN][iI][nN][gG]/
{ logToken("RETURNING"); return RETURNING }
    case 143:  //[rR][eE][vV][oO][kK][eE]/
{ logToken("REVOKE"); return REVOKE }
    case 144:  //[rR][iI][gG][hH][tT]/
{ logToken("RIGHT"); return RIGHT }
    case 145:  //[rR][oO][lL][eE]/
{ logToken("ROLE"); return ROLE }
    case 146:  //[rR][oO][lL][lL][bB][aA][cC][kK]/
{ logToken("ROLLBACK"); return ROLLBACK }
    case 147:  //[sS][aA][tT][iI][sS][fF][iI][eE][sS]/
{ logToken("SATISFIES"); return SATISFIES }
    case 148:  //[sS][cC][hH][eE][mM][aA]/
{ logToken("SCHEMA"); return SCHEMA }
    case 149:  //[sS][eE][lL][eE][cC][tT]/
{ logToken("SELECT"); return SELECT }
    case 150:  //[sS][eE][tT]/
{ logToken("SET"); return SET }
    case 151:  //[sS][hH][oO][wW]/
{ logToken("SHOW"); return SHOW }
    case 152:  //[sS][oO][mM][eE]/
{ logToken("SOME"); return SOME }
    case 153:  //[sS][tT][aA][rR][tT]/
{ logToken("START"); return START }
    case 154:  //[sS][tT][aA][tT][iI][sS][tT][iI][cC][sS]/
{ logToken("STATISTICS"); return STATISTICS }
    case 155:  //[sS][yY][sS][tT][eE][mM]/
{ logToken("SYSTEM"); return SYSTEM }
    case 156:  //[tT][hH][eE][nN]/
{ logToken("THEN"); return THEN }
    case 157:  //[tT][oO]/
{ logToken("TO"); return TO }
    case 158:  //[tT][rR][aA][nN][sS][aA][cC][tT][iI][oO][nN]/
{ logToken("TRANSACTION"); return TRANSACTION }
    case 159:  //[tT][rR][iI][gG][gG][eE][rR]/
{ logToken("TRIGGER"); return TRIGGER }
    case 160:  //[tT][rR][uU][eE]/
{ logToken("TRUE"); return TRUE }
    case 161:  //[tT][rR][uU][nN][cC][aA][tT][eE]/
{ logToken("TRUNCATE"); return TRUNCATE }
    case 162:  //[tT][yY][pP][eE]/
{ logToken("TYPE"); return TYPE }
    case 163:  //[uU][nN][dD][eE][rR]/
{ logToken("UNDER"); return UNDER }
    case 164:  //[uU][nN][iI][oO][nN]/
{ logToken("UNION"); return UNION }
    case 165:  //[uU][nN][iI][qQ][uU][eE]/
{ logToken("UNIQUE"); return UNIQUE }
    case 166:  //[uU][nN][nN][eE][sS][tT]/
{ logToken("UNNEST"); return UNNEST }
    case 167:  //[uU][nN][sS][eE][tT]/
{ logToken("UNSET"); return UNSET }
    case 168:  //[uU][pP][dD][aA][tT][eE]/
{ logToken("UPDATE"); return UPDATE }
    case 169:  //[uU][pP][sS][eE][rR][tT]/
{ logToken("UPSERT"); return UPSERT }
    case 170:  //[uU][sS][eE]/
{ logToken("USE"); return USE }
    case 171:  //[uU][sS][eE][rR]/
{ logToken("USER"); return USER }
    case 172:  //[uU][sS][iI][nN][gG]/
{ logToken("USING"); return USING }
    case 173:  //[vV][aA][lL][uU][eE]/
{ logToken("VALUE"); return VALUE }
    case 174:  //[vV][aA][lL][uU][eE][dD]/
{ logToken("VALUED"); return VALUED }
    case 175:  //[vV][aA][lL][uU][eE][sS]/
{ logToken("VALUES"); return VALUES }
    case 176:  //[vV][iI][eE][wW]/
{ logToken("VIEW"); return VIEW }
    case 177:  //[wW][hH][eE][nN]/
{ logToken("WHEN"); return WHEN }
    case 178:  //[wW][hH][eE][rR][eE]/
{ logToken("WHERE"); return WHERE }
    case 179:  //[wW][hH][iI][lL][eE]/
{ logToken("WHILE"); return WHILE }
    case 180:  //[wW][iI][tT][hH]/
{ logToken("WITH"); return WITH }
    case 181:  //[wW][iI][tT][hH][iI][nN]/
{ logToken("WITHIN"); return WITHIN }
    case 182:  //[wW][oO][rR][kK]/
{ logToken("WORK"); return WORK }
    case 183:  //[xX][oO][rR]/
{ logToken("XOR"); return XOR }
    case 184:  //[a-zA-Z_][a-zA-Z0-9_]*/
{
		    lval.s = yylex.Text()
		    logToken("IDENTIFIER - %s", lval.s)
		    return IDENTIFIER
		  }
    case 185:  //\$[a-zA-Z_][a-zA-Z0-9_]*/
{
		    lval.s = yylex.Text()[1:]
		    logToken("NAMED_PARAM - %s", lval.s)
		    return NAMED_PARAM
		  }
    case 186:  //\$[1-9][0-9]*/
{
		    lval.n, _ = strconv.Atoi(yylex.Text()[1:])
		    logToken("POSITIONAL_PARAM - %d", lval.n)
		    return POSITIONAL_PARAM
		  }
    case 187:  ///
// [END]
    }
  }
  return 0
}
func logToken(format string, v ...interface{}) {
    clog.To("LEXER", format, v...)
}
