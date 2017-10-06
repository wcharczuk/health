package health

import "fmt"

// ANSI contains a bunch of ansi commands.
var ANSI = ansi{
	SOH: byte(1),
	STX: byte(2),
	ETX: byte(3),
	EOT: byte(4),
	ENQ: byte(5),
	ACK: byte(6),
	BS:  byte(8),
	TAB: byte(9),
	LF:  byte(10),
	VT:  byte(11),
	CR:  byte(13),
	SO:  byte(14),
	DLE: byte(16),
	ESC: byte(27),
	DEL: byte(127),

	EscSequenceStart: []byte{byte(27), byte('[')},
	Clear:            []byte{byte(27), byte('['), byte('2'), byte('J')},
	ClearLine:        []byte{byte(27), byte('['), byte('2'), byte('K')},
	HideCursor:       []byte{byte(27), byte('['), byte('?'), byte('2'), byte('5'), byte('l')},
	ShowCursor:       []byte{byte(27), byte('['), byte('?'), byte('2'), byte('5'), byte('h')},

	ColorReset:        []byte{byte(27), byte('['), byte('0'), byte('m')},
	ColorBold:         []byte{byte(27), byte('['), byte('1'), byte('m')},
	ColorItalics:      []byte{byte(27), byte('['), byte('1'), byte('m')},
	ColorUnderline:    []byte{byte(27), byte('['), byte('1'), byte('m')},
	ColorBoldOff:      []byte{byte(27), byte('['), byte('2'), byte('2'), byte('m')},
	ColorItalicsOff:   []byte{byte(27), byte('['), byte('2'), byte('3'), byte('m')},
	ColorUnderlineOff: []byte{byte(27), byte('['), byte('2'), byte('4'), byte('m')},
}

type ansi struct {
	SOH byte
	STX byte
	ETX byte
	EOT byte
	ENQ byte
	ACK byte
	BS  byte
	TAB byte
	LF  byte
	VT  byte
	CR  byte
	ESC byte
	SO  byte
	DLE byte
	DEL byte

	Left  byte
	Right byte
	Up    byte
	Down  byte

	EscSequenceStart []byte
	Clear            []byte
	ClearLine        []byte
	HideCursor       []byte
	ShowCursor       []byte

	ColorReset        []byte
	ColorBold         []byte
	ColorBoldOff      []byte
	ColorItalics      []byte
	ColorItalicsOff   []byte
	ColorUnderline    []byte
	ColorUnderlineOff []byte
}

func (a ansi) Escape(sequence []byte) []byte {
	return append(a.EscSequenceStart, sequence...)
}

func (a ansi) MoveCursor(row, col int) []byte {
	if row != 0 && col != 0 {
		return append(a.EscSequenceStart, []byte(fmt.Sprintf("%d;%dH", row, col))...)
	} else if row != 0 {
		return append(a.EscSequenceStart, []byte(fmt.Sprintf("%d;H", row))...)
	}
	return append(a.EscSequenceStart, []byte(fmt.Sprintf(";%dH", col))...)
}

func (a ansi) Spaces(count int) []byte {
	switch count {
	case 0:
		return []byte{}
	case 1:
		return []byte{' '}
	case 2:
		return []byte{' ', ' '}
	case 3:
		return []byte{' ', ' ', ' '}
	case 4:
		return []byte{' ', ' ', ' ', ' '}
	case 5:
		return []byte{' ', ' ', ' ', ' ', ' '}
	case 6:
		return []byte{' ', ' ', ' ', ' ', ' ', ' '}
	case 7:
		return []byte{' ', ' ', ' ', ' ', ' ', ' '}
	case 8:
		return []byte{' ', ' ', ' ', ' ', ' ', ' ', ' '}
	case 9:
		return []byte{' ', ' ', ' ', ' ', ' ', ' ', ' ', ' '}
	case 10:
		return []byte{' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' '}
	}
	var bytes []byte

	for x := 0; x < count; x++ {
		bytes = append(bytes, byte(' '))
	}
	return bytes
}
