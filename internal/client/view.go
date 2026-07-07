package client

import (
	"fmt"
	"image/color"
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/chalvinwz/hacktris/internal/engine"
)

// nextQueue is how many upcoming pieces the NEXT column previews.
const nextQueue = 5

// pieceHex is the source of truth for piece colours; the beveled block shades
// (highlight/shadow) and the ghost outline are all derived from it.
var pieceHex = map[engine.PieceType]string{
	engine.PieceI: "#22C7D6",
	engine.PieceO: "#E8C41F",
	engine.PieceT: "#9A3CD1",
	engine.PieceS: "#3FB84F",
	engine.PieceZ: "#E5484D",
	engine.PieceJ: "#3D63DD",
	engine.PieceL: "#E8912B",
}

const garbageHex = "#8B949E"

var (
	colDim    = lipgloss.Color("#7D8590")
	colFaint  = lipgloss.Color("#454C56")
	colAccent = lipgloss.Color("#FF9900")
	colText   = lipgloss.Color("#E6EDF3")
	colWin    = lipgloss.Color("#3FB950")
	colLose   = lipgloss.Color("#F85149")
	colInk    = lipgloss.Color("#0D1117")

	// Board: black playfield with faint grid lines; empty cells are near-black.
	bgHex      = "#0A0D12"
	gridHex    = "#20262E"
	frameColor = lipgloss.Color("#C9D1D9") // white-ish frame like tetr.io
)

var (
	wellStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(frameColor)
	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(frameColor).
			Background(lipgloss.Color(bgHex)).
			Padding(0, 1)
	oppBoardStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(colFaint).
			Padding(0, 0)

	headerChip = lipgloss.NewStyle().Bold(true).Foreground(colInk).Background(colText).Padding(0, 1)
	titleChip  = lipgloss.NewStyle().Bold(true).Foreground(colInk).Background(colAccent).Padding(0, 1)

	metaStyle  = lipgloss.NewStyle().Foreground(colDim)
	barStyle   = lipgloss.NewStyle().Foreground(colDim)
	labelStyle = lipgloss.NewStyle().Foreground(colDim)
	valueStyle = lipgloss.NewStyle().Bold(true).Foreground(colText)
	bigStyle   = lipgloss.NewStyle().Bold(true).Foreground(colText)
	keyStyle   = lipgloss.NewStyle().Foreground(colAccent)
	overStyle  = lipgloss.NewStyle().Bold(true).Foreground(colLose)
	winStyle   = lipgloss.NewStyle().Bold(true).Foreground(colWin)

	leftColStyle  = lipgloss.NewStyle().MarginRight(2)
	rightColStyle = lipgloss.NewStyle().MarginLeft(2)
)

// View renders the current phase inside a full-screen frame: a title bar across
// the top, the body centred in the space between, and a hint bar across the
// bottom.
func (m Model) View() tea.View {
	var body string
	switch m.phase {
	case phaseConnecting:
		body = metaStyle.Render("Connecting to " + m.serverAddr + " …")
	case phaseLobby:
		body = fmt.Sprintf("%s\n%s",
			valueStyle.Render("Waiting for players …"),
			metaStyle.Render(fmt.Sprintf("joined as %q", m.name)))
	case phaseError:
		body = fmt.Sprintf("%s\n%s",
			overStyle.Render("Connection error"),
			metaStyle.Render(m.errMsg))
	default: // phasePlaying, phaseOver
		body = m.gameBody()
	}

	v := tea.NewView(m.renderFrame(body))
	v.AltScreen = true
	// Request key repeat/release reporting (Kitty keyboard protocol). Terminals
	// that support it let us track held arrows for smooth auto-shift and move
	// while rotating; others simply ignore it and fall back to per-press moves.
	v.KeyboardEnhancements.ReportEventTypes = true
	return v
}

// viewport returns the terminal size, falling back to sane defaults before the
// first resize message arrives.
func (m Model) viewport() (int, int) {
	w, h := m.width, m.height
	if w <= 0 {
		w = 80
	}
	if h <= 0 {
		h = 24
	}
	return w, h
}

// wellScale is the cell magnification. It picks the largest size (up to 3) that
// still fits the board, panels and frame bars, so the well grows to fill big
// terminals while staying safe on small ones. At scale s each cell is 2s columns
// by s rows, keeping blocks square.
func (m Model) wellScale() int {
	w, h := m.viewport()
	best := 1
	for s := 2; s <= 3; s++ {
		needH := engine.Height*s + 7   // board + borders + title/footer
		needW := engine.Width*2*s + 44 // board + HOLD/NEXT columns
		if h >= needH && w >= needW {
			best = s
		}
	}
	return best
}

// renderFrame wraps body with the title and hint bars, filling the whole
// terminal and centring body in the remaining vertical space.
func (m Model) renderFrame(body string) string {
	w, h := m.viewport()
	title := m.titleBar(w)
	footer := m.footerBar(w)
	mid := h - lipgloss.Height(title) - lipgloss.Height(footer)
	if mid < 1 {
		mid = 1
	}
	centered := lipgloss.Place(w, mid, lipgloss.Center, lipgloss.Center, body)
	return strings.Join([]string{title, centered, footer}, "\n")
}

// titleBar is the full-width header: the HACKTRIS chip plus a status summary.
func (m Model) titleBar(w int) string {
	bar := titleChip.Render("HACKTRIS") + metaStyle.Render("  "+m.statusMeta())
	return barStyle.Width(w).MaxWidth(w).Render(bar)
}

// footerBar is the full-width key-hint line.
func (m Model) footerBar(w int) string {
	return barStyle.Width(w).MaxWidth(w).Render(m.hints())
}

// statusMeta is the short phase/mode summary shown in the title bar.
func (m Model) statusMeta() string {
	switch m.phase {
	case phaseConnecting:
		return "connecting…"
	case phaseLobby:
		return "lobby · waiting for players"
	case phaseError:
		return "disconnected"
	case phaseOver:
		if m.winnerID == m.myID && m.myID != 0 {
			return "you win!"
		}
		return "match over"
	default:
		if m.solo() {
			return "solo"
		}
		return fmt.Sprintf("vs · %d players · %d alive", len(m.players), m.aliveCount())
	}
}

// aliveCount counts living players (self plus opponents).
func (m Model) aliveCount() int {
	n := 0
	if !m.eliminated {
		n++
	}
	for _, o := range m.opponents {
		if o.alive {
			n++
		}
	}
	return n
}

// hints builds the key-hint line for the footer.
func (m Model) hints() string {
	pairs := [][2]string{
		{"← →", "move"}, {"↑/x", "rotate"}, {"z", "ccw"},
		{"c", "hold"}, {"↓", "soft"}, {"space", "hard drop"},
	}
	if m.solo() {
		pairs = append(pairs, [2]string{"r", "restart"})
	}
	pairs = append(pairs, [2]string{"q", "quit"})

	parts := make([]string, len(pairs))
	for i, p := range pairs {
		parts[i] = keyStyle.Render(p[0]) + " " + metaStyle.Render(p[1])
	}
	return strings.Join(parts, metaStyle.Render("   ·   "))
}

// gameBody lays out the tetr.io-style board: HOLD column on the left, the well
// with the score beneath it in the centre, and the NEXT queue on the right.
// Opponent boards sit below in multiplayer.
func (m Model) gameBody() string {
	well := wellStyle.Render(m.renderWell())
	center := lipgloss.JoinVertical(lipgloss.Center, well, "", m.scoreBlock())
	left := leftColStyle.Render(m.leftColumn())
	right := rightColStyle.Render(m.rightColumn())
	row := lipgloss.JoinHorizontal(lipgloss.Top, left, center, right)

	if opp := m.renderOpponents(); opp != "" {
		row = lipgloss.JoinVertical(lipgloss.Center, row, "", opp)
	}
	if banner := m.resultBanner(); banner != "" {
		return lipgloss.JoinVertical(lipgloss.Center, banner, "", row)
	}
	return row
}

// leftColumn is the HOLD box plus line/level stats and any status note.
func (m Model) leftColumn() string {
	hold := lipgloss.JoinVertical(lipgloss.Left, headerChip.Render("HOLD"), boxStyle.Render(m.renderHold()))
	parts := []string{
		hold, "",
		stat("LINES", fmt.Sprintf("%d", m.game.Lines)),
		stat("LEVEL", fmt.Sprintf("%d", m.game.Level)),
	}
	switch {
	case m.eliminated && m.phase != phaseOver:
		parts = append(parts, "", overStyle.Render("ELIMINATED"))
	case m.solo() && m.game.GameOver:
		parts = append(parts, "", overStyle.Render("GAME OVER"))
	}
	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

// rightColumn is the NEXT header over a vertical queue of upcoming pieces, plus
// an alive count in multiplayer.
func (m Model) rightColumn() string {
	q := m.game.Queue(nextQueue)
	rows := make([]string, 0, 2*len(q))
	for i, p := range q {
		if i > 0 {
			rows = append(rows, "")
		}
		rows = append(rows, previewPiece(p, false))
	}
	queue := lipgloss.JoinVertical(lipgloss.Center, rows...)
	body := lipgloss.JoinVertical(lipgloss.Left, headerChip.Render("NEXT"), boxStyle.Render(queue))
	if !m.solo() {
		body = lipgloss.JoinVertical(lipgloss.Left, body, "", stat("ALIVE", fmt.Sprintf("%d", m.aliveCount())))
	}
	return body
}

// scoreBlock is the large centred score shown under the well.
func (m Model) scoreBlock() string {
	return lipgloss.JoinVertical(lipgloss.Center,
		labelStyle.Render("SCORE"),
		bigStyle.Render(commafmt(m.game.Score)))
}

// resultBanner returns the headline shown over the board once play ends: the
// multiplayer win/lose result, or — in solo — a game-over card with a restart
// hint (solo never leaves phasePlaying, so the well simply freezes behind it).
func (m Model) resultBanner() string {
	if m.phase == phaseOver {
		if m.winnerID == m.myID && m.myID != 0 {
			return bannerBox(colWin).Render(winStyle.Render("★  YOU WIN!  ★"))
		}
		name := "nobody"
		for _, p := range m.players {
			if p.ID == m.winnerID {
				name = p.Name
			}
		}
		card := lipgloss.JoinVertical(lipgloss.Center, overStyle.Render(fmt.Sprintf("%s wins", name)), metaStyle.Render("press q to quit"))
		return bannerBox(colLose).Render(card)
	}
	if m.solo() && m.game != nil && m.game.GameOver {
		card := lipgloss.JoinVertical(lipgloss.Center,
			overStyle.Render("GAME OVER"),
			metaStyle.Render("press r to restart · q to quit"))
		return bannerBox(colLose).Render(card)
	}
	return ""
}

// bannerBox is a rounded, padded frame in the given accent colour for the
// win/lose/game-over headline.
func bannerBox(c color.Color) lipgloss.Style {
	return lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(c).Padding(0, 3)
}

// composeGrid overlays the active piece on the locked board, for both rendering
// and the network snapshot.
func (m Model) composeGrid() [engine.Height][engine.Width]engine.Cell {
	var grid [engine.Height][engine.Width]engine.Cell
	b := m.game.Board()
	for y := 0; y < engine.Height; y++ {
		for x := 0; x < engine.Width; x++ {
			grid[y][x] = b.At(x, y)
		}
	}
	kind, cells := m.game.Current()
	for _, c := range cells {
		x, y := c[0], c[1]
		if y >= 0 && y < engine.Height && x >= 0 && x < engine.Width {
			grid[y][x] = kind.CellValue()
		}
	}
	return grid
}

// renderWell draws the playfield: beveled "candy" tiles for blocks, faint grid
// lines on empty cells, and a hollow ghost showing where the active piece lands.
// The ghost is display-only — it never enters composeGrid or the network
// snapshot.
func (m Model) renderWell() string {
	scale := m.wellScale()
	w := 2 * scale
	grid := m.composeGrid()

	var ghost [engine.Height][engine.Width]bool
	kind, _ := m.game.Current()
	if !m.game.GameOver {
		for _, c := range m.game.GhostCells() {
			if c[1] >= 0 && c[1] < engine.Height && c[0] >= 0 && c[0] < engine.Width {
				ghost[c[1]][c[0]] = true
			}
		}
	}

	var sb strings.Builder
	for y := 0; y < engine.Height; y++ {
		for sub := 0; sub < scale; sub++ {
			for x := 0; x < engine.Width; x++ {
				switch c := grid[y][x]; {
				case c == engine.Garbage:
					sb.WriteString(blockSubRow(garbageHex, w, sub, scale))
				case c != engine.Empty:
					sb.WriteString(blockSubRow(pieceHex[engine.PieceType(c-1)], w, sub, scale))
				case ghost[y][x]:
					sb.WriteString(ghostSubRow(pieceHex[kind], w, sub, scale))
				default:
					sb.WriteString(emptySubRow(w, sub))
				}
			}
			if !(y == engine.Height-1 && sub == scale-1) {
				sb.WriteByte('\n')
			}
		}
	}
	return sb.String()
}

// blockSubRow renders one terminal row of a block cell as a solid colour square.
func blockSubRow(baseHex string, w, sub, scale int) string {
	return fg(lipgloss.Color(baseHex)).Render(strings.Repeat("█", w))
}

// emptySubRow renders one terminal row of an empty cell as a near-black square
// with faint grid lines: a vertical tick at the left edge and, on the top
// sub-row, a horizontal tick along the top.
func emptySubRow(w, sub int) string {
	bg := lipgloss.Color(bgHex)
	grid := lipgloss.Color(gridHex)
	var b strings.Builder
	for i := 0; i < w; i++ {
		switch {
		case i == 0:
			b.WriteString(lipgloss.NewStyle().Foreground(grid).Background(bg).Render("▏"))
		case sub == 0:
			b.WriteString(lipgloss.NewStyle().Foreground(grid).Background(bg).Render("▔"))
		default:
			b.WriteString(lipgloss.NewStyle().Background(bg).Render(" "))
		}
	}
	return b.String()
}

// ghostSubRow renders one terminal row of the ghost as a hollow outline in a
// dimmed piece hue (a faint translucent tile at scale 1).
func ghostSubRow(baseHex string, w, sub, scale int) string {
	c := lerpHex(baseHex, bgHex, 0.50)
	bg := lipgloss.Color(bgHex)
	st := lipgloss.NewStyle().Foreground(c).Background(bg)
	if scale == 1 {
		return st.Render(strings.Repeat("░", w))
	}
	top, bottom := sub == 0, sub == scale-1
	var b strings.Builder
	for i := 0; i < w; i++ {
		ch := " "
		switch {
		case top:
			ch = "▔"
		case bottom:
			ch = "▁"
		case i == 0:
			ch = "▏"
		case i == w-1:
			ch = "▕"
		}
		b.WriteString(st.Render(ch))
	}
	return b.String()
}

// renderHold draws the hold-slot piece, greyed while hold is on cooldown, or an
// empty box before anything has been held.
func (m Model) renderHold() string {
	held, ok := m.game.Held()
	if !ok {
		return "        \n        " // two blank preview rows (4 cells wide)
	}
	return previewPiece(held, !m.game.CanHold())
}

// previewPiece draws a piece in a 4x2 grid of beveled tiles (greyed when dim).
func previewPiece(p engine.PieceType, dim bool) string {
	baseHex := pieceHex[p]
	if dim {
		baseHex = "#3A3F46"
	}
	var box [2][4]bool
	for _, c := range engine.PreviewCells(p) {
		x, y := c[0], c[1]
		if y >= 0 && y < 2 && x >= 0 && x < 4 {
			box[y][x] = true
		}
	}
	var sb strings.Builder
	for y := 0; y < 2; y++ {
		for x := 0; x < 4; x++ {
			if box[y][x] {
				sb.WriteString(blockSubRow(baseHex, 2, 0, 1))
			} else {
				sb.WriteString("  ")
			}
		}
		if y == 0 {
			sb.WriteByte('\n')
		}
	}
	return sb.String()
}

// renderOpponents draws a header plus a wrapping grid of opponent cards, living
// players first so the active boards stay grouped.
func (m Model) renderOpponents() string {
	if len(m.opponents) == 0 {
		return ""
	}
	ids := m.oppOrder()
	cards := make([]string, 0, len(ids))
	for _, id := range ids {
		cards = append(cards, renderOppCard(m.opponents[id]))
	}
	header := labelStyle.Render("OPPONENTS  ") + valueStyle.Render(fmt.Sprintf("%d alive", m.aliveCount()))
	return lipgloss.JoinVertical(lipgloss.Left, header, "", wrapCards(cards, m.oppPerRow()))
}

// oppOrder returns opponent ids with living players first, each group sorted by
// id for a stable layout.
func (m Model) oppOrder() []int {
	ids := make([]int, 0, len(m.opponents))
	for id := range m.opponents {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool {
		a, b := m.opponents[ids[i]], m.opponents[ids[j]]
		if a.alive != b.alive {
			return a.alive
		}
		return ids[i] < ids[j]
	})
	return ids
}

// oppPerRow estimates how many opponent cards fit across the terminal.
func (m Model) oppPerRow() int {
	w, _ := m.viewport()
	cardW := engine.Width + 4 // board + border + gap
	n := (w - 4) / cardW
	if n < 1 {
		n = 1
	}
	if n > len(m.opponents) {
		n = len(m.opponents)
	}
	return n
}

// renderOppCard draws a single opponent: name header, compact board, and a
// score line (or a KO tag once eliminated).
func renderOppCard(o *oppView) string {
	name := o.name
	if len(name) > engine.Width {
		name = name[:engine.Width]
	}
	head := valueStyle.Render(name)
	if !o.alive {
		head = overStyle.Render(name + " ✗")
	}

	var sb strings.Builder
	for y := 0; y < engine.Height; y++ {
		for x := 0; x < engine.Width; x++ {
			c := engine.Empty
			if o.cells != nil && y*engine.Width+x < len(o.cells) {
				c = engine.Cell(o.cells[y*engine.Width+x])
			}
			sb.WriteString(miniGlyph(c, o.alive))
		}
		if y < engine.Height-1 {
			sb.WriteByte('\n')
		}
	}
	board := oppBoardStyle.Render(sb.String())

	foot := labelStyle.Render(fmt.Sprintf("%d", o.score))
	if !o.alive {
		foot = overStyle.Render("KO")
	}
	return lipgloss.JoinVertical(lipgloss.Left, head, board, foot)
}

// wrapCards lays cards out in rows of at most perRow, then stacks the rows.
func wrapCards(cards []string, perRow int) string {
	if perRow < 1 {
		perRow = 1
	}
	gap := lipgloss.NewStyle().MarginLeft(1)
	var rows []string
	for i := 0; i < len(cards); i += perRow {
		end := i + perRow
		if end > len(cards) {
			end = len(cards)
		}
		row := cards[i]
		for _, c := range cards[i+1 : end] {
			row = lipgloss.JoinHorizontal(lipgloss.Top, row, gap.Render(c))
		}
		rows = append(rows, row)
	}
	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

func stat(label, value string) string {
	return labelStyle.Render(fmt.Sprintf("%-6s", label)) + valueStyle.Render(value)
}

// miniGlyph styles a single mini-board cell (one column wide) as a solid pixel.
// Eliminated boards render their blocks dimmed.
func miniGlyph(c engine.Cell, alive bool) string {
	switch {
	case c == engine.Empty:
		return lipgloss.NewStyle().Foreground(lipgloss.Color(gridHex)).Render("█")
	case !alive:
		return fg(colFaint).Render("█")
	case c == engine.Garbage:
		return fg(lipgloss.Color(garbageHex)).Render("█")
	default:
		return fg(lipgloss.Color(pieceHex[engine.PieceType(c-1)])).Render("█")
	}
}

// fg is shorthand for a foreground-only style.
func fg(c color.Color) lipgloss.Style { return lipgloss.NewStyle().Foreground(c) }

// commafmt formats an integer with thousands separators (e.g. 40029 -> 40,029).
func commafmt(n int) string {
	s := fmt.Sprintf("%d", n)
	neg := strings.HasPrefix(s, "-")
	if neg {
		s = s[1:]
	}
	var out []byte
	for i := 0; i < len(s); i++ {
		if i > 0 && (len(s)-i)%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, s[i])
	}
	if neg {
		return "-" + string(out)
	}
	return string(out)
}

// lerpHex linearly interpolates between two "#RRGGBB" colours.
func lerpHex(a, b string, t float64) color.Color {
	ar, ag, ab := hexRGB(a)
	br, bg, bb := hexRGB(b)
	lerp := func(x, y int) int { return x + int(float64(y-x)*t+0.5) }
	return lipgloss.Color(fmt.Sprintf("#%02X%02X%02X", lerp(ar, br), lerp(ag, bg), lerp(ab, bb)))
}

func hexRGB(s string) (r, g, b int) {
	fmt.Sscanf(s, "#%02x%02x%02x", &r, &g, &b)
	return
}
