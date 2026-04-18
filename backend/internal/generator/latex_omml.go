package generator

import (
	"fmt"
	"strings"
)

// latexToOMML converts a LaTeX math expression to an OOXML <m:oMath> fragment
// that Word renders as native Office Math. Covers the subset we actually see
// in LLM output: variables, numbers, operators, Greek letters, sub/superscripts,
// fractions, roots, matrices, auto-sized delimiters, common functions, and
// n-ary operators with limits.
//
// Unrecognized macros fall through as plain text so output never fails — the
// expression degrades gracefully rather than aborting the render.
func latexToOMML(tex string) string {
	p := &texParser{src: strings.TrimSpace(tex)}
	body := strings.Join(p.parseUntil(""), "")
	if body == "" {
		return ""
	}
	return "<m:oMath>" + body + "</m:oMath>"
}

// latexToOMMLPara wraps latexToOMML in the paragraph-level element used for
// display math — rendered centered on its own line by Word.
func latexToOMMLPara(tex string) string {
	inner := latexToOMML(tex)
	if inner == "" {
		return ""
	}
	return "<m:oMathPara>" + inner + "</m:oMathPara>"
}

type texParser struct {
	src string
	pos int
}

// parseUntil consumes source until it hits one of the stop tokens (e.g. "}",
// "\\end", "&", "\\\\") and returns a slice of OMML fragments.
func (p *texParser) parseUntil(stops ...string) []string {
	var out []string
	for p.pos < len(p.src) {
		if p.matchAny(stops) {
			break
		}
		c := p.src[p.pos]
		switch {
		case c == ' ' || c == '\t' || c == '\n' || c == '\r':
			p.pos++
		case c == '\\':
			// try a matrix/environment shortcut first
			if tok := p.peekMacroName(); tok == "begin" {
				out = append(out, p.parseEnvironment())
				continue
			}
			out = append(out, p.parseMacro())
		case c == '{':
			p.pos++
			inner := p.parseUntil("}")
			p.consume("}")
			out = append(out, strings.Join(inner, ""))
		case c == '^':
			p.pos++
			sup := p.parseAtom()
			out = attachSupSub(out, sup, "", "sup")
		case c == '_':
			p.pos++
			sub := p.parseAtom()
			out = attachSupSub(out, "", sub, "sub")
		case isLetter(c):
			start := p.pos
			for p.pos < len(p.src) && isLetter(p.src[p.pos]) {
				p.pos++
			}
			out = append(out, mathRun(p.src[start:p.pos], true))
		case isDigit(c):
			start := p.pos
			for p.pos < len(p.src) && (isDigit(p.src[p.pos]) || p.src[p.pos] == '.') {
				p.pos++
			}
			out = append(out, mathRun(p.src[start:p.pos], false))
		case c == '(' || c == ')' || c == '[' || c == ']':
			out = append(out, mathRun(string(c), false))
			p.pos++
		default:
			out = append(out, mathRun(string(c), false))
			p.pos++
		}
	}
	return out
}

// parseAtom returns a single OMML fragment: either a braced group, a macro's
// output, or a single character/number. Used by ^, _, \sqrt, \frac arguments.
func (p *texParser) parseAtom() string {
	p.skipSpaces()
	if p.pos >= len(p.src) {
		return ""
	}
	c := p.src[p.pos]
	switch {
	case c == '{':
		p.pos++
		inner := p.parseUntil("}")
		p.consume("}")
		return strings.Join(inner, "")
	case c == '\\':
		return p.parseMacro()
	case isLetter(c):
		p.pos++
		return mathRun(string(c), true)
	case isDigit(c):
		start := p.pos
		for p.pos < len(p.src) && (isDigit(p.src[p.pos]) || p.src[p.pos] == '.') {
			p.pos++
		}
		return mathRun(p.src[start:p.pos], false)
	default:
		p.pos++
		return mathRun(string(c), false)
	}
}

// parseMacro handles a \name token and its arguments. Unknown macros return
// their name as plain text so the output keeps something readable.
func (p *texParser) parseMacro() string {
	if p.pos >= len(p.src) || p.src[p.pos] != '\\' {
		return ""
	}
	p.pos++
	if p.pos >= len(p.src) {
		return ""
	}
	// single-char macros like \\, \,, \; , \!, \(, \)
	c := p.src[p.pos]
	if !isLetter(c) {
		p.pos++
		return singleCharMacro(c)
	}
	start := p.pos
	for p.pos < len(p.src) && isLetter(p.src[p.pos]) {
		p.pos++
	}
	name := p.src[start:p.pos]
	return p.applyMacro(name)
}

// peekMacroName returns the macro name at the current position without
// advancing, or "" if the cursor isn't at a macro.
func (p *texParser) peekMacroName() string {
	if p.pos >= len(p.src) || p.src[p.pos] != '\\' {
		return ""
	}
	i := p.pos + 1
	start := i
	for i < len(p.src) && isLetter(p.src[i]) {
		i++
	}
	return p.src[start:i]
}

// parseEnvironment handles \begin{name}...\end{name}. Supports the common
// matrix environments (matrix, pmatrix, bmatrix, vmatrix, Bmatrix, Vmatrix).
// Unknown environments degrade to their literal inner content.
func (p *texParser) parseEnvironment() string {
	// consume "\\begin"
	p.consumeMacro("begin")
	p.skipSpaces()
	p.consume("{")
	name := p.readUntilChar('}')
	p.consume("}")

	switch name {
	case "matrix", "pmatrix", "bmatrix", "vmatrix", "Bmatrix", "Vmatrix", "cases":
		return p.parseMatrixBody(name)
	default:
		// read body verbatim up to \end{name}
		endTok := "\\end{" + name + "}"
		idx := strings.Index(p.src[p.pos:], endTok)
		if idx < 0 {
			return ""
		}
		body := p.src[p.pos : p.pos+idx]
		p.pos += idx + len(endTok)
		return mathRun(body, false)
	}
}

// parseMatrixBody walks a matrix environment body, splitting rows on "\\\\"
// and cells on "&", and emits an <m:m> block wrapped in the environment's
// conventional delimiters (parens for pmatrix, brackets for bmatrix, etc.).
func (p *texParser) parseMatrixBody(name string) string {
	var rows [][]string
	endTok := "\\end{" + name + "}"
	for p.pos < len(p.src) {
		if strings.HasPrefix(p.src[p.pos:], endTok) {
			p.pos += len(endTok)
			break
		}
		cells := []string{}
		for {
			cellFrags := p.parseUntil("&", "\\\\", endTok)
			cells = append(cells, strings.Join(cellFrags, ""))
			if p.matchAny([]string{endTok}) {
				break
			}
			if p.matchAny([]string{"&"}) {
				p.consume("&")
				continue
			}
			if p.matchAny([]string{"\\\\"}) {
				p.consume("\\\\")
				break
			}
			break
		}
		if len(cells) > 0 {
			rows = append(rows, cells)
		}
	}
	if len(rows) == 0 {
		return ""
	}
	cols := 0
	for _, r := range rows {
		if len(r) > cols {
			cols = len(r)
		}
	}

	var b strings.Builder
	b.WriteString(`<m:m><m:mPr><m:mcs>`)
	for i := 0; i < cols; i++ {
		b.WriteString(`<m:mc><m:mcPr><m:count m:val="1"/><m:mcJc m:val="center"/></m:mcPr></m:mc>`)
	}
	b.WriteString(`</m:mcs></m:mPr>`)
	for _, r := range rows {
		b.WriteString(`<m:mr>`)
		for i := 0; i < cols; i++ {
			cell := ""
			if i < len(r) {
				cell = r[i]
			}
			b.WriteString("<m:e>" + cell + "</m:e>")
		}
		b.WriteString(`</m:mr>`)
	}
	b.WriteString(`</m:m>`)

	beg, end := matrixDelims(name)
	if beg == "" {
		return b.String()
	}
	return fmt.Sprintf(`<m:d><m:dPr><m:begChr m:val="%s"/><m:endChr m:val="%s"/></m:dPr><m:e>%s</m:e></m:d>`,
		beg, end, b.String())
}

func matrixDelims(env string) (string, string) {
	switch env {
	case "pmatrix":
		return "(", ")"
	case "bmatrix":
		return "[", "]"
	case "Bmatrix":
		return "{", "}"
	case "vmatrix":
		return "|", "|"
	case "Vmatrix":
		return "‖", "‖"
	case "cases":
		return "{", ""
	}
	return "", ""
}

// applyMacro dispatches on the macro name. Falls back to plain text for
// unrecognized macros.
func (p *texParser) applyMacro(name string) string {
	// Greek letters & standalone symbols
	if sym, ok := symbolMacros[name]; ok {
		return mathRun(sym, false)
	}
	if sym, ok := greekMacros[name]; ok {
		return mathRun(sym, false)
	}
	// Operators-with-limits (rendered as n-ary with optional _ and ^)
	if op, ok := naryMacros[name]; ok {
		return p.parseNary(op)
	}
	// Operator-names rendered upright
	if fn, ok := functionMacros[name]; ok {
		return mathRun(fn, false)
	}

	switch name {
	case "frac", "tfrac", "dfrac":
		num := p.parseAtom()
		den := p.parseAtom()
		return fmt.Sprintf(`<m:f><m:num>%s</m:num><m:den>%s</m:den></m:f>`, num, den)
	case "sqrt":
		p.skipSpaces()
		if p.pos < len(p.src) && p.src[p.pos] == '[' {
			p.pos++
			deg := strings.Join(p.parseUntil("]"), "")
			p.consume("]")
			arg := p.parseAtom()
			return fmt.Sprintf(`<m:rad><m:deg>%s</m:deg><m:e>%s</m:e></m:rad>`, deg, arg)
		}
		arg := p.parseAtom()
		return fmt.Sprintf(`<m:rad><m:radPr><m:degHide m:val="1"/></m:radPr><m:deg/><m:e>%s</m:e></m:rad>`, arg)
	case "left":
		return p.parseAutoDelim()
	case "right":
		// consumed inside parseAutoDelim; a stray \right falls through
		if p.pos < len(p.src) {
			p.pos++
		}
		return ""
	case "text", "mathrm", "operatorname":
		arg := p.readBracedRaw()
		return `<m:r><m:rPr><m:sty m:val="p"/></m:rPr><m:t xml:space="preserve">` + xmlEscape(arg) + `</m:t></m:r>`
	case "mathbf":
		arg := p.parseAtom()
		return fmt.Sprintf(`<m:r><m:rPr><m:sty m:val="b"/></m:rPr>%s</m:r>`, stripOuterRunTags(arg))
	case "mathbb", "mathcal":
		arg := p.readBracedRaw()
		return mathRun(arg, true)
	case "hat", "bar", "vec", "tilde", "dot", "ddot", "overline":
		arg := p.parseAtom()
		acc, isBar := accentMacros[name]
		if isBar {
			return fmt.Sprintf(`<m:bar><m:barPr><m:pos m:val="top"/></m:barPr><m:e>%s</m:e></m:bar>`, arg)
		}
		return fmt.Sprintf(`<m:acc><m:accPr><m:chr m:val="%s"/></m:accPr><m:e>%s</m:e></m:acc>`, acc, arg)
	case "cdot":
		return mathRun("⋅", false)
	case "ldots", "dots", "cdots":
		return mathRun("⋯", false)
	case "quad", "qquad":
		return mathRun(" ", false)
	}
	// Unknown — degrade to plain text of the macro name
	return mathRun("\\"+name, false)
}

// parseAutoDelim handles \left<X>...\right<Y>.
func (p *texParser) parseAutoDelim() string {
	p.skipSpaces()
	if p.pos >= len(p.src) {
		return ""
	}
	beg := string(p.src[p.pos])
	p.pos++
	inner := p.parseUntil("\\right")
	// consume \right and its delimiter
	if strings.HasPrefix(p.src[p.pos:], "\\right") {
		p.pos += len("\\right")
	}
	p.skipSpaces()
	end := ""
	if p.pos < len(p.src) {
		end = string(p.src[p.pos])
		p.pos++
	}
	if beg == "." {
		beg = ""
	}
	if end == "." {
		end = ""
	}
	return fmt.Sprintf(`<m:d><m:dPr><m:begChr m:val="%s"/><m:endChr m:val="%s"/></m:dPr><m:e>%s</m:e></m:d>`,
		beg, end, strings.Join(inner, ""))
}

// parseNary handles \sum, \prod, \int etc. Picks up optional _ and ^ limits
// that immediately follow the macro.
func (p *texParser) parseNary(op string) string {
	var sub, sup string
	for i := 0; i < 2; i++ {
		p.skipSpaces()
		if p.pos >= len(p.src) {
			break
		}
		switch p.src[p.pos] {
		case '_':
			p.pos++
			sub = p.parseAtom()
		case '^':
			p.pos++
			sup = p.parseAtom()
		default:
			i = 3 // break outer
		}
	}
	return fmt.Sprintf(`<m:nary><m:naryPr><m:chr m:val="%s"/></m:naryPr><m:sub>%s</m:sub><m:sup>%s</m:sup><m:e/></m:nary>`,
		op, sub, sup)
}

// ─── Utilities ──────────────────────────────────────────────────────────────

func (p *texParser) matchAny(stops []string) bool {
	if p.pos >= len(p.src) {
		return false
	}
	for _, s := range stops {
		if s == "" {
			continue
		}
		if strings.HasPrefix(p.src[p.pos:], s) {
			return true
		}
	}
	return false
}

func (p *texParser) consume(s string) {
	if p.pos+len(s) <= len(p.src) && p.src[p.pos:p.pos+len(s)] == s {
		p.pos += len(s)
	}
}

func (p *texParser) consumeMacro(name string) {
	tok := "\\" + name
	if strings.HasPrefix(p.src[p.pos:], tok) {
		p.pos += len(tok)
	}
}

func (p *texParser) skipSpaces() {
	for p.pos < len(p.src) {
		c := p.src[p.pos]
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			p.pos++
			continue
		}
		return
	}
}

func (p *texParser) readUntilChar(c byte) string {
	start := p.pos
	for p.pos < len(p.src) && p.src[p.pos] != c {
		p.pos++
	}
	return p.src[start:p.pos]
}

// readBracedRaw reads a {…} group as raw text, without recursing into LaTeX
// parsing — used by \text, \mathrm, \operatorname.
func (p *texParser) readBracedRaw() string {
	p.skipSpaces()
	if p.pos >= len(p.src) || p.src[p.pos] != '{' {
		// single char
		if p.pos < len(p.src) {
			c := string(p.src[p.pos])
			p.pos++
			return c
		}
		return ""
	}
	p.pos++
	depth := 1
	start := p.pos
	for p.pos < len(p.src) && depth > 0 {
		switch p.src[p.pos] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				s := p.src[start:p.pos]
				p.pos++
				return s
			}
		}
		p.pos++
	}
	return p.src[start:p.pos]
}

func isLetter(c byte) bool { return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') }
func isDigit(c byte) bool  { return c >= '0' && c <= '9' }

// mathRun emits an OMML run. When italic is true, the run carries <m:sty
// m:val="i"/>, which is what Word uses for variable letters.
func mathRun(text string, italic bool) string {
	if text == "" {
		return ""
	}
	if italic {
		return `<m:r><m:rPr><m:sty m:val="i"/></m:rPr><m:t xml:space="preserve">` + xmlEscape(text) + `</m:t></m:r>`
	}
	return `<m:r><m:t xml:space="preserve">` + xmlEscape(text) + `</m:t></m:r>`
}

// attachSupSub wraps the last emitted fragment with a subscript, superscript,
// or combined sub/sup element. If there's no preceding fragment, wraps an
// empty base.
func attachSupSub(out []string, sup, sub, primary string) []string {
	base := ""
	if len(out) > 0 {
		base = out[len(out)-1]
		out = out[:len(out)-1]
	}
	var wrapped string
	switch {
	case primary == "sup" && sub == "":
		wrapped = fmt.Sprintf(`<m:sSup><m:e>%s</m:e><m:sup>%s</m:sup></m:sSup>`, base, sup)
	case primary == "sub" && sup == "":
		wrapped = fmt.Sprintf(`<m:sSub><m:e>%s</m:e><m:sub>%s</m:sub></m:sSub>`, base, sub)
	default:
		wrapped = fmt.Sprintf(`<m:sSubSup><m:e>%s</m:e><m:sub>%s</m:sub><m:sup>%s</m:sup></m:sSubSup>`, base, sub, sup)
	}
	out = append(out, wrapped)
	return out
}

// singleCharMacro handles macros like "\\" (newline-in-math, rarely used
// outside environments), "\\,", "\\;", "\\!", etc.
func singleCharMacro(c byte) string {
	switch c {
	case '\\':
		return "" // row break handled in matrix parsing
	case ',', ';', ':', '!', ' ':
		return mathRun(" ", false)
	case '{', '}', '%', '&', '$', '#':
		return mathRun(string(c), false)
	}
	return mathRun(string(c), false)
}

// stripOuterRunTags is a last-resort helper used by \mathbf to inject a bold
// style when the argument is a single plain run. For structured arguments the
// style is dropped (OMML has no document-wide bold across nested structures).
func stripOuterRunTags(frag string) string {
	if strings.HasPrefix(frag, "<m:r>") && strings.HasSuffix(frag, "</m:r>") {
		return frag[len("<m:r>") : len(frag)-len("</m:r>")]
	}
	return frag
}

// ─── Symbol and function tables ─────────────────────────────────────────────

var greekMacros = map[string]string{
	"alpha": "α", "beta": "β", "gamma": "γ", "delta": "δ", "epsilon": "ε",
	"varepsilon": "ɛ", "zeta": "ζ", "eta": "η", "theta": "θ", "vartheta": "ϑ",
	"iota": "ι", "kappa": "κ", "lambda": "λ", "mu": "μ", "nu": "ν", "xi": "ξ",
	"omicron": "ο", "pi": "π", "varpi": "ϖ", "rho": "ρ", "varrho": "ϱ",
	"sigma": "σ", "varsigma": "ς", "tau": "τ", "upsilon": "υ", "phi": "φ",
	"varphi": "ϕ", "chi": "χ", "psi": "ψ", "omega": "ω",
	"Gamma": "Γ", "Delta": "Δ", "Theta": "Θ", "Lambda": "Λ", "Xi": "Ξ",
	"Pi": "Π", "Sigma": "Σ", "Upsilon": "Υ", "Phi": "Φ", "Psi": "Ψ", "Omega": "Ω",
}

var symbolMacros = map[string]string{
	"times": "×", "div": "÷", "pm": "±", "mp": "∓", "ast": "∗",
	"star": "⋆", "circ": "∘", "bullet": "•", "oplus": "⊕", "otimes": "⊗",
	"odot": "⊙", "leq": "≤", "le": "≤", "geq": "≥", "ge": "≥",
	"neq": "≠", "ne": "≠", "approx": "≈", "equiv": "≡", "sim": "∼",
	"simeq": "≃", "cong": "≅", "propto": "∝", "to": "→", "rightarrow": "→",
	"leftarrow": "←", "leftrightarrow": "↔", "Rightarrow": "⇒",
	"Leftarrow": "⇐", "Leftrightarrow": "⇔", "mapsto": "↦",
	"infty": "∞", "partial": "∂", "nabla": "∇", "forall": "∀", "exists": "∃",
	"emptyset": "∅", "in": "∈", "notin": "∉", "subset": "⊂", "supset": "⊃",
	"subseteq": "⊆", "supseteq": "⊇", "cup": "∪", "cap": "∩", "setminus": "∖",
	"because": "∵", "therefore": "∴", "angle": "∠", "perp": "⊥", "parallel": "∥",
	"prime": "′", "degree": "°", "hbar": "ℏ", "ell": "ℓ", "Re": "ℜ", "Im": "ℑ",
	"wp": "℘", "aleph": "ℵ",
}

var functionMacros = map[string]string{
	"sin": "sin", "cos": "cos", "tan": "tan", "cot": "cot", "sec": "sec", "csc": "csc",
	"arcsin": "arcsin", "arccos": "arccos", "arctan": "arctan",
	"sinh": "sinh", "cosh": "cosh", "tanh": "tanh",
	"log": "log", "ln": "ln", "lg": "lg", "exp": "exp",
	"min": "min", "max": "max", "gcd": "gcd", "lcm": "lcm",
	"det": "det", "dim": "dim", "ker": "ker", "deg": "deg",
	"arg": "arg", "mod": "mod", "lim": "lim", "sup": "sup", "inf": "inf",
}

var naryMacros = map[string]string{
	"sum": "∑", "prod": "∏", "coprod": "∐",
	"int": "∫", "iint": "∬", "iiint": "∭", "oint": "∮",
	"bigcup": "⋃", "bigcap": "⋂", "bigvee": "⋁", "bigwedge": "⋀",
	"bigoplus": "⊕", "bigotimes": "⊗", "biguplus": "⊎",
}

var accentMacros = map[string]string{
	"hat":   "̂",
	"tilde": "̃",
	"vec":   "⃗",
	"dot":   "̇",
	"ddot":  "̈",
	// "bar" and "overline" use <m:bar> rather than <m:acc>
	"bar":      "",
	"overline": "",
}
