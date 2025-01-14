package vt10x

import (
	"strconv"
	"strings"
)

// CSI (Control Sequence Introducer)
// ESC+[
type csiEscape struct {
	buf  []byte
	args []int
	mode byte
	priv bool
}

func (c *csiEscape) reset() {
	c.buf = c.buf[:0]
	c.args = c.args[:0]
	c.mode = 0
	c.priv = false
}

func (c *csiEscape) put(b byte) bool {
	c.buf = append(c.buf, b)
	if b >= 0x40 && b <= 0x7E || len(c.buf) >= 256 {
		c.parse()
		return true
	}
	return false
}

func (c *csiEscape) parse() {
	c.mode = c.buf[len(c.buf)-1]
	if len(c.buf) == 1 {
		return
	}
	s := string(c.buf)
	c.args = c.args[:0]
	if s[0] == '?' {
		c.priv = true
		s = s[1:]
	}
	s = s[:len(s)-1]
	ss := strings.Split(s, ";")
	for _, p := range ss {
		i, err := strconv.Atoi(p)
		if err != nil {
			//t.logf("invalid CSI arg '%s'\n", p)
			break
		}
		c.args = append(c.args, i)
	}
}

func (c *csiEscape) arg(i, def int) int {
	if i >= len(c.args) || i < 0 {
		return def
	}
	return c.args[i]
}

// maxarg takes the maximum of arg(i, def) and def
func (c *csiEscape) maxarg(i, def int) int {
	return max(c.arg(i, def), def)
}

func (t *State) handleCSI() {
	c := &t.csi
	switch c.mode {
	default:
		goto unknown
	case '@': // ICH - insert <n> blank char
		t.insertBlanks(c.arg(0, 1))
	case 'A': // CUU - Cursor <n> up
		t.moveTo(t.Cur.x, t.Cur.y-c.maxarg(0, 1))
	case 'B', 'e': // CUD, VPR - Cursor <n> down
		t.moveTo(t.Cur.x, t.Cur.y+c.maxarg(0, 1))
	case 'c': // DA - device attributes
		if c.arg(0, 0) == 0 {
			// TODO: write vt102 id
		}
	case 'C', 'a': // CUF, HPR - Cursor <n> forward
		t.moveTo(t.Cur.x+c.maxarg(0, 1), t.Cur.y)
	case 'D': // CUB - Cursor <n> backward
		t.moveTo(t.Cur.x-c.maxarg(0, 1), t.Cur.y)
	case 'E': // CNL - Cursor <n> down and first col
		t.moveTo(0, t.Cur.y+c.arg(0, 1))
	case 'F': // CPL - Cursor <n> up and first col
		t.moveTo(0, t.Cur.y-c.arg(0, 1))
	case 'g': // TBC - tabulation clear
		switch c.arg(0, 0) {
		// clear current tab stop
		case 0:
			t.tabs[t.Cur.x] = false
		// clear all tabs
		case 3:
			for i := range t.tabs {
				t.tabs[i] = false
			}
		default:
			goto unknown
		}
	case 'G', '`': // CHA, HPA - Move to <col>
		t.moveTo(c.arg(0, 1)-1, t.Cur.y)
	case 'H', 'f': // CUP, HVP - move to <row> <col>
		t.moveAbsTo(c.arg(1, 1)-1, c.arg(0, 1)-1)
	case 'I': // CHT - Cursor forward tabulation <n> tab stops
		n := c.arg(0, 1)
		for i := 0; i < n; i++ {
			t.putTab(true)
		}
	case 'J': // ED - clear screen
		// TODO: sel.ob.x = -1
		switch c.arg(0, 0) {
		case 0: // below
			t.clear(t.Cur.x, t.Cur.y, t.cols-1, t.Cur.y)
			if t.Cur.y < t.rows-1 {
				t.clear(0, t.Cur.y+1, t.cols-1, t.rows-1)
			}
		case 1: // above
			if t.Cur.y > 1 {
				t.clear(0, 0, t.cols-1, t.Cur.y-1)
			}
			t.clear(0, t.Cur.y, t.Cur.x, t.Cur.y)
		case 2: // all
			t.clear(0, 0, t.cols-1, t.rows-1)
		default:
			goto unknown
		}
	case 'K': // EL - clear line
		switch c.arg(0, 0) {
		case 0: // right
			t.clear(t.Cur.x, t.Cur.y, t.cols-1, t.Cur.y)
		case 1: // left
			t.clear(0, t.Cur.y, t.Cur.x, t.Cur.y)
		case 2: // all
			t.clear(0, t.Cur.y, t.cols-1, t.Cur.y)
		}
	case 'S': // SU - scroll <n> lines up
		t.ScrollUp(t.top, c.arg(0, 1))
	case 'T': // SD - scroll <n> lines down
		t.ScrollDown(t.top, c.arg(0, 1))
	case 'L': // IL - insert <n> blank lines
		t.insertBlankLines(c.arg(0, 1))
	case 'l': // RM - reset Mode
		t.setMode(c.priv, false, c.args)
	case 'M': // DL - delete <n> lines
		t.deleteLines(c.arg(0, 1))
	case 'X': // ECH - erase <n> chars
		t.clear(t.Cur.x, t.Cur.y, t.Cur.x+c.arg(0, 1)-1, t.Cur.y)
	case 'P': // DCH - delete <n> chars
		t.deleteChars(c.arg(0, 1))
	case 'Z': // CBT - Cursor backward tabulation <n> tab stops
		n := c.arg(0, 1)
		for i := 0; i < n; i++ {
			t.putTab(false)
		}
	case 'd': // VPA - move to <row>
		t.moveAbsTo(t.Cur.x, c.arg(0, 1)-1)
	case 'h': // SM - set terminal Mode
		t.setMode(c.priv, true, c.args)
	case 'm': // SGR - terminal attribute (color)
		t.setAttr(c.args)
	case 'r': // DECSTBM - set scrolling region
		if c.priv {
			goto unknown
		} else {
			t.setScroll(c.arg(0, 1)-1, c.arg(1, t.rows)-1)
			t.moveAbsTo(0, 0)
		}
	case 's': // DECSC - save Cursor position (ANSI.SYS)
		t.saveCursor()
	case 'u': // DECRC - restore Cursor position (ANSI.SYS)
		t.restoreCursor()
	case 'q': /*
	TODO: Handle cursor block shape
	Ps = 0,1    Block cursor / Blink
	   = 2      Block cursor / Steady
	   = 3      Underline cursor / Blink
	   = 4      Underline cursor / Steady
	   = 5      Vertical line cursor / Blink
	   = 6      Vertical line cursor / Steady
	*/
	case 't': // Ignoring IME state set change
	}

	return
unknown: // TODO: get rid of this goto
	t.logf("unknown CSI sequence '%s'\n", string(c.mode))

	// TODO: Char.dump()
}
