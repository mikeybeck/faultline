package sourcemap

import (
	"encoding/json"
	"strings"
)

// Map is a decoded v3 source map.
type Map struct {
	File    string
	Sources []string
	lines   [][]segment
}

type segment struct {
	genCol    int
	source    int
	origLine  int
	origCol   int
	hasSource bool
}

type rawMap struct {
	File     string   `json:"file"`
	Sources  []string `json:"sources"`
	Mappings string   `json:"mappings"`
}

// Parse decodes a source map JSON document.
func Parse(data []byte) (*Map, error) {
	var raw rawMap
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	m := &Map{File: raw.File, Sources: raw.Sources}
	m.lines = parseMappings(raw.Mappings)
	return m, nil
}

// Lookup maps a 1-based generated line/column onto original source location.
func (m *Map) Lookup(line, col int) (file string, origLine, origCol int, ok bool) {
	if m == nil || line <= 0 {
		return "", 0, 0, false
	}
	idx := line - 1
	if idx >= len(m.lines) {
		return "", 0, 0, false
	}
	segs := m.lines[idx]
	if len(segs) == 0 {
		return "", 0, 0, false
	}
	if col < 0 {
		col = 0
	}
	best := -1
	for i, s := range segs {
		if !s.hasSource {
			continue
		}
		if s.genCol <= col {
			best = i
		} else {
			break
		}
	}
	if best < 0 {
		for i, s := range segs {
			if s.hasSource {
				best = i
				break
			}
		}
	}
	if best < 0 {
		return "", 0, 0, false
	}
	s := segs[best]
	if s.source < 0 || s.source >= len(m.Sources) {
		return "", 0, 0, false
	}
	file = strings.TrimSpace(m.Sources[s.source])
	if file == "" {
		return "", 0, 0, false
	}
	return file, s.origLine + 1, s.origCol, true
}

func parseMappings(mappings string) [][]segment {
	var lines [][]segment
	src, ol, oc, gen := 0, 0, 0, 0
	i := 0
	var segs []segment
	flush := func() {
		out := make([]segment, len(segs))
		copy(out, segs)
		lines = append(lines, out)
		segs = segs[:0]
		gen = 0
	}
	for i < len(mappings) {
		if mappings[i] == ';' {
			flush()
			i++
			continue
		}
		if mappings[i] == ',' {
			i++
			continue
		}
		var ok bool
		var n int
		n, i, ok = decodeVLQ(mappings, i)
		if !ok {
			break
		}
		gen += n
		seg := segment{genCol: gen}
		if i < len(mappings) && mappings[i] != ',' && mappings[i] != ';' {
			n, i, ok = decodeVLQ(mappings, i)
			if !ok {
				break
			}
			src += n
			n, i, ok = decodeVLQ(mappings, i)
			if !ok {
				break
			}
			ol += n
			n, i, ok = decodeVLQ(mappings, i)
			if !ok {
				break
			}
			oc += n
			if i < len(mappings) && mappings[i] != ',' && mappings[i] != ';' {
				_, i, ok = decodeVLQ(mappings, i)
				if !ok {
					break
				}
			}
			seg.source = src
			seg.origLine = ol
			seg.origCol = oc
			seg.hasSource = true
		}
		segs = append(segs, seg)
	}
	flush()
	return lines
}
