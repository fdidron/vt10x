package vt10x

func isControlCode(c rune) bool {
	return c < 0x20 || c == 0177
}

func (t *State) parse(c rune) {
	if isControlCode(c) {
		if t.handleControlCodes(c) || t.Cur.Attr.Mode&attrGfx == 0 {
			return
		}
	}
	// TODO: update selection; see st.Char:2450

	if t.mode&ModeWrap != 0 && t.Cur.state&cursorWrapNext != 0 {
		t.lines[t.Cur.y][t.Cur.x].Mode |= attrWrap
		t.newline(true)
	}

	if t.mode&ModeInsert != 0 && t.Cur.x+1 < t.cols {
		// TODO: move shiz, look at st.Char:2458
		t.logln("insert Mode not implemented")
	}

	t.setChar(c, &t.Cur.Attr, t.Cur.x, t.Cur.y)
	if t.Cur.x+1 < t.cols {
		t.moveTo(t.Cur.x+1, t.Cur.y)
	} else {
		t.Cur.state |= cursorWrapNext
	}
}

func (t *State) parseEsc(c rune) {
	if t.handleControlCodes(c) {
		return
	}
	next := t.parse
	switch c {
	case '[':
		next = t.parseEscCSI
	case '#':
		next = t.parseEscTest
	case 'P', // DCS - Device Control String
		'_', // APC - Application Program Command
		'^', // PM - Privacy Message
		']', // OSC - Operating System Command
		'k': // old title set compatibility
		t.str.reset()
		t.str.typ = c
		next = t.parseEscStr
	case '(': // set primary charset G0
		next = t.parseEscAltCharset
	case ')', // set secondary charset G1 (ignored)
		'*', // set tertiary charset G2 (ignored)
		'+': // set quaternary charset G3 (ignored)
	case 'D': // IND - linefeed
		if t.Cur.y == t.bottom {
			t.ScrollUp(t.top, 1)
		} else {
			t.moveTo(t.Cur.x, t.Cur.y+1)
		}
	case 'E': // NEL - next line
		t.newline(true)
	case 'H': // HTS - horizontal tab stop
		t.tabs[t.Cur.x] = true
	case 'M': // RI - reverse index
		if t.Cur.y == t.top {
			t.ScrollDown(t.top, 1)
		} else {
			t.moveTo(t.Cur.x, t.Cur.y-1)
		}
	case 'Z': // DECID - identify terminal
		// TODO: write to our writer our id
	case 'c': // RIS - reset to initial state
		t.reset()
	case '=': // DECPAM - application keypad
		t.mode |= ModeAppKeypad
	case '>': // DECPNM - normal keypad
		t.mode &^= ModeAppKeypad
	case '7': // DECSC - save Cursor
		t.saveCursor()
	case '8': // DECRC - restore Cursor
		t.restoreCursor()
	case '\\': // ST - stop
	default:
		t.logf("unknown ESC sequence '%Char'\n", c)
	}
	t.state = next
}

func (t *State) parseEscCSI(c rune) {
	if t.handleControlCodes(c) {
		return
	}
	if t.csi.put(byte(c)) {
		t.state = t.parse
		t.handleCSI()
	}
}

func (t *State) parseEscStr(c rune) {
	switch c {
	case '\033':
		t.state = t.parseEscStrEnd
	case '\a': // backwards compatiblity to xterm
		t.state = t.parse
		t.handleSTR()
	default:
		t.str.put(c)
	}
}

func (t *State) parseEscStrEnd(c rune) {
	if t.handleControlCodes(c) {
		return
	}
	t.state = t.parse
	if c == '\\' {
		t.handleSTR()
	}
}

func (t *State) parseEscAltCharset(c rune) {
	if t.handleControlCodes(c) {
		return
	}
	switch c {
	case '0': // line drawing set
		t.Cur.Attr.Mode |= attrGfx
	case 'B': // USASCII
		t.Cur.Attr.Mode &^= attrGfx
	case 'A', // UK (ignored)
		'<', // multinational (ignored)
		'5', // Finnish (ignored)
		'C', // Finnish (ignored)
		'K': // German (ignored)
	default:
		t.logf("unknown alt. charset '%Char'\n", c)
	}
	t.state = t.parse
}

func (t *State) parseEscTest(c rune) {
	if t.handleControlCodes(c) {
		return
	}
	// DEC screen alignment test
	if c == '8' {
		for y := 0; y < t.rows; y++ {
			for x := 0; x < t.cols; x++ {
				t.setChar('E', &t.Cur.Attr, x, y)
			}
		}
	}
	t.state = t.parse
}

func (t *State) handleControlCodes(c rune) bool {
	if !isControlCode(c) {
		return false
	}
	switch c {
	// HT
	case '\t':
		t.putTab(true)
	// BS
	case '\b':
		t.moveTo(t.Cur.x-1, t.Cur.y)
	// CR
	case '\r':
		t.moveTo(0, t.Cur.y)
	// LF, VT, LF
	case '\f', '\v', '\n':
		// go to first col if Mode is set
		t.newline(t.mode&ModeCRLF != 0)
	// BEL
	case '\a':
		// TODO: emit sound
		// TODO: window alert if not focused
	// ESC
	case 033:
		t.csi.reset()
		t.state = t.parseEsc
	// SO, SI
	case 016, 017:
		// different charsets not supported. apps should use the correct
		// alt charset escapes, probably for line drawing
	// SUB, CAN
	case 032, 030:
		t.csi.reset()
	// ignore ENQ, NUL, XON, XOFF, DEL
	case 005, 000, 021, 023, 0177:
	default:
		return false
	}
	return true
}
