package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	colFocused  = lipgloss.Color("9")
	colTitle    = lipgloss.Color("9")

	colUnfocus  = lipgloss.Color("8")

	colPlaying  = lipgloss.Color("6")
	colSelected = lipgloss.Color("12")

	colReady    = lipgloss.Color("2")
	colPending  = lipgloss.Color("3")
	colError    = lipgloss.Color("1")
)

func (ps *PlayerState) View() string {
	if ps.windowWidth == 0 || ps.windowHeight == 0 {
		return "Loading..."
	}
	switch ps.mode {
	case ModeHelp:
		return ps.renderHelp()
	case ModeSearch, ModeSearching:
		return ps.renderMain()
	}
	return ps.renderMain()
}

//  Main view

func (ps *PlayerState) renderMain() string {
	h := ps.windowHeight - 2

	var abW int
	if ps.windowWidth > WideThreshold {
		abW = SideWidth
	} else {
		abW = 2 * ps.windowWidth / 5
	}

	var panels string
	if ps.showLibrary {
		libW := ps.windowWidth * 30 / 100
		chW := ps.windowWidth - libW - abW

		lib := ps.renderLibraryPanel(libW, h)
		ab := ps.renderAudiobooksPanel(abW, h)
		ch := ps.renderChaptersPanel(chW, h)
		panels = lipgloss.JoinHorizontal(lipgloss.Top, lib, ab, ch)
	} else {
		chW := ps.windowWidth - abW

		ab := ps.renderAudiobooksPanel(abW, h)
		ch := ps.renderChaptersPanel(chW, h)
		panels = lipgloss.JoinHorizontal(lipgloss.Top, ab, ch)
	}

	return lipgloss.JoinVertical(lipgloss.Left, panels, ps.renderPlayerBar())
}

//  Panel helper

// renderPanel builds a bordered panel with a title in the top border.
// titleX=-1 centers the title; titleX>=0 places it that many chars from the left.
// Total rendered size is exactly w×h characters.
func renderPanel(title string, w, h int, content string, focused bool, titleX int) string {
	borderCol := colUnfocus
	if focused {
		borderCol = colFocused
	}

	t := " " + title + " "
	inner := w - 2
	if inner < 0 {
		inner = 0
	}
	rem := inner - len(t)
	if rem < 0 {
		rem = 0
		if len(t) > inner {
			t = t[:inner]
		}
	}
	var lp, rp int
	if titleX < 0 {
		lp = rem / 2
		rp = rem - lp
	} else {
		lp = titleX
		if lp > rem {
			lp = rem
		}
		rp = rem - lp
	}
	borderStyle := lipgloss.NewStyle().Foreground(borderCol)
	if focused {
		borderStyle = borderStyle.Bold(true)
	}

	var topRendered string
	if focused {
		titleStyle := lipgloss.NewStyle().Foreground(colTitle).Bold(true)
		topRendered = borderStyle.Render("┌"+strings.Repeat("─", lp)) +
			titleStyle.Render(t) +
			borderStyle.Render(strings.Repeat("─", rp)+"┐")
	} else {
		topRendered = borderStyle.Render("┌" + strings.Repeat("─", lp) + t + strings.Repeat("─", rp) + "┐")
	}

	contentH := h - 2
	if contentH < 0 {
		contentH = 0
	}
	contentStyle := lipgloss.NewStyle().
		Width(inner).
		Height(contentH).
		Border(lipgloss.NormalBorder()).
		BorderTop(false).
		BorderForeground(borderCol)

	return topRendered + "\n" + contentStyle.Render(content)
}

// padLine returns s padded with spaces to exactly n visual characters.
func padLine(s string, n int) string {
	vis := lipgloss.Width(s)
	if vis >= n {
		return s
	}
	return s + strings.Repeat(" ", n-vis)
}

//  Audiobooks panel 

func (ps *PlayerState) renderAudiobooksPanel(w, h int) string {
	focused := ps.onAlbum && !ps.libActive
	inner := w - 2
	contentH := h - 2

	books := ps.sortedLocalBooks()

	listH := contentH
	if ps.albumInfoOpen {
		listH = contentH*6/10 + 4
	}

	var listLines []string
	for i := ps.albumOffset; i < len(books) && len(listLines) < listH; i++ {
		b := books[i]
		isPlaying := ps.albumPlaying != nil && *ps.albumPlaying == i
		isSelected := i == ps.albumSelected

		pct := ps.bookProgress(b)
		var pctStr string
		switch {
		case pct >= 1.0:
			pctStr = "(end)"
		case pct <= 0:
			pctStr = "     "
		default:
			pctStr = fmt.Sprintf("(%02d%%)", int(pct*100))
		}
		titleW := inner - 1 - 5 - 2
		if titleW < 0 {
			titleW = 0
		}

		title := truncate(b.Title+" - "+b.Author, titleW)
		title = fmt.Sprintf("%-*s", titleW, title)

		plain := " " + title + " " + pctStr

		var style lipgloss.Style
		switch {
		case isPlaying && isSelected && !ps.albumInfoFocused:
			style = lipgloss.NewStyle().Reverse(true).Foreground(colPlaying)
		case isPlaying:
			style = lipgloss.NewStyle().Reverse(true).Foreground(colReady)
		case isSelected && !ps.albumInfoFocused:
			style = lipgloss.NewStyle().Reverse(true).Foreground(colSelected)
		default:
			style = lipgloss.NewStyle().Foreground(colUnfocus)
		}

		listLines = append(listLines, style.Render(padLine(plain, inner)))
	}
	for len(listLines) < listH {
		listLines = append(listLines, strings.Repeat(" ", inner))
	}

	if !ps.albumInfoOpen {
		return renderPanel("Audiobooks", w, h, strings.Join(listLines, "\n"), focused, -1)
	}

	borderCol := colUnfocus
	if focused {
		borderCol = colFocused
	}
	dividerCol := borderCol
	if ps.albumInfoFocused {
		dividerCol = colFocused
	}
	divider := lipgloss.NewStyle().Foreground(dividerCol).Render(strings.Repeat("─",inner))
	detailH := contentH - listH - 1
	b := ps.selectedLocalBook()
	var detailLines []string
	if b != nil {
		detailLines = ps.buildBookDetail(b, inner, detailH, false, ps.albumInfoOff, ps.albumInfoFocused)
	} else {
		for len(detailLines) < detailH {
			detailLines = append(detailLines, strings.Repeat(" ", inner))
		}
	}

	content := strings.Join(listLines, "\n") + "\n" + divider + "\n" + strings.Join(detailLines, "\n")
	return renderPanel("Audiobooks", w, h, content, focused, -1)
}

//  Chapters panel 

func (ps *PlayerState) renderChaptersPanel(w, h int) string {
	focused := !ps.onAlbum && !ps.libActive
	inner := w - 2
	contentH := h - 2

	book := ps.selectedLocalBook()
	if book == nil {
		return renderPanel("Chapters", w, h, "", focused, 4)
	}

	curCh := -1
	if book.Position != nil {
		curCh = book.Position.ChapterIndex
	}

	durW := 5
	for _, ch := range book.Chapters {
		if w := len(fmtMMSS(ch.Duration)); w > durW {
			durW = w
		}
	}
	// " "(1) + title + " "(1) + dur(durW) + " "(1) = inner
	titleW := inner - durW - 3
	if titleW < 0 {
		titleW = 0
	}

	var lines []string
	for i := ps.trackOffset; i < len(book.Chapters) && len(lines) < contentH; i++ {
		ch := book.Chapters[i]
		isCurrent := i == curCh
		isSelected := i == ps.trackSelected

		title := truncate(ch.Title, titleW)
		title = fmt.Sprintf("%-*s", titleW, title)
		dur := fmt.Sprintf("%*s", durW, fmtMMSS(ch.Duration))

		plain := " " + title + " " + dur

		var style lipgloss.Style
		switch {
		case isCurrent && isSelected:
			style = lipgloss.NewStyle().Reverse(true).Foreground(colPlaying)
		case isCurrent:
			style = lipgloss.NewStyle().Reverse(true).Foreground(colReady)
		case isSelected:
			style = lipgloss.NewStyle().Reverse(true).Foreground(colSelected)
		default:
			style = lipgloss.NewStyle().Foreground(colUnfocus)
		}

		lines = append(lines, style.Render(padLine(plain, inner)))
	}

	for len(lines) < contentH {
		lines = append(lines, strings.Repeat(" ", inner))
	}

	return renderPanel("Chapters", w, h, strings.Join(lines, "\n"), focused, 4)
}

//  Library panel 

func (ps *PlayerState) renderLibraryPanel(w, h int) string {
	focused := ps.libActive
	borderCol := colUnfocus
	if focused {
		borderCol = colFocused
	}

	inner := w - 2
	contentH := h - 2

	var content string
	if ps.libInfoOpen {
		listH := contentH*6/10 + 4
		detailH := contentH - listH - 1
		listLines := ps.buildLibList(inner, listH)
		dividerCol := borderCol
		if ps.libInfoFocused {
			dividerCol = colFocused
		}
		divider := lipgloss.NewStyle().Foreground(dividerCol).Render(strings.Repeat("─",inner))
		detailLines := ps.buildLibDetail(inner, detailH)
		content = strings.Join(listLines, "\n") + "\n" + divider + "\n" + strings.Join(detailLines, "\n")
	} else {
		listLines := ps.buildLibList(inner, contentH)
		content = strings.Join(listLines, "\n")
	}

	// build top border manually
	t := " Library "
	rem := inner - len(t)
	if rem < 0 {
		rem = 0
	}
	lp := rem / 2
	rp := rem - lp
	borderStyle := lipgloss.NewStyle().Foreground(borderCol)
	if focused {
		borderStyle = borderStyle.Bold(true)
	}

	var topRendered string
	if focused {
		titleStyle := lipgloss.NewStyle().Foreground(colTitle).Bold(true)
		topRendered = borderStyle.Render("┌"+strings.Repeat("─", lp)) +
			titleStyle.Render(t) +
			borderStyle.Render(strings.Repeat("─", rp)+"┐")
	} else {
		topRendered = borderStyle.Render("┌" + strings.Repeat("─", lp) + t + strings.Repeat("─", rp) + "┐")
	}

	contentStyle := lipgloss.NewStyle().
		Width(inner).
		Height(contentH).
		Border(lipgloss.NormalBorder()).
		BorderTop(false).
		BorderForeground(borderCol)

	return topRendered + "\n" + contentStyle.Render(content)
}

func (ps *PlayerState) buildLibList(inner, h int) []string {
	books := ps.lib.Books

	var lines []string
	for i := ps.libOffset; i < len(books) && len(lines) < h; i++ {
		b := books[i]
		isSelected := i == ps.libSel

		var stateCol lipgloss.Color
		switch b.State {
		case DownloadInProgress, DownloadPreparing:
			stateCol = colPending
		case DownloadReady:
			stateCol = colReady
		default:
			stateCol = colUnfocus
		}

		label := b.Title + " - " + b.Author
		var plain string
		if b.State == DownloadInProgress {
			pct := fmt.Sprintf("%3d%%", int(b.DownloadProgress*100))
			titleW := inner - 1 - 1 - len(pct) - 1
			if titleW < 0 {
				titleW = 0
			}
			plain = " " + fmt.Sprintf("%-*s", titleW, truncate(label, titleW)) + " " + pct + " "
		} else {
			titleW := inner - 1
			plain = " " + fmt.Sprintf("%-*s", titleW, truncate(label, titleW))
		}

		var style lipgloss.Style
		if isSelected && !ps.libInfoFocused {
			style = lipgloss.NewStyle().Reverse(true).Foreground(colSelected)
		} else {
			style = lipgloss.NewStyle().Foreground(stateCol)
		}

		lines = append(lines, style.Render(padLine(plain, inner)))
	}

	for len(lines) < h {
		lines = append(lines, strings.Repeat(" ", inner))
	}
	return lines
}

func (ps *PlayerState) buildLibDetail(inner, h int) []string {
	if ps.libSel >= len(ps.lib.Books) {
		var lines []string
		for len(lines) < h {
			lines = append(lines, strings.Repeat(" ", inner))
		}
		return lines
	}
	return ps.buildBookDetail(&ps.lib.Books[ps.libSel], inner, h, true, ps.libInfoOff, ps.libInfoFocused)
}

func (ps *PlayerState) buildBookDetail(b *Audiobook, inner, h int, showChip bool, offset int, focused bool) []string {
	var lines []string

	dim := lipgloss.NewStyle().Foreground(colUnfocus)
	accent := lipgloss.NewStyle().Foreground(colReady)

	titleStyle := lipgloss.NewStyle()
	if focused {
		titleStyle = titleStyle.Foreground(colFocused).Bold(true)
	}
	lines = append(lines, padLine(titleStyle.Render(truncate(b.Title, inner)), inner))

	meta := fmt.Sprintf("%s · %d · %s", truncate(b.Author, 30), b.Date, fmtHM(b.Duration))
	lines = append(lines, padLine(dim.Render(truncate(meta, inner)), inner))

	genres := strings.Join(b.Genres, ", ")
	lines = append(lines, padLine(dim.Render(truncate(genres, inner)), inner))

	if showChip {
		chip, chipCol := bookChip(b)
		sizeStr := fmt.Sprintf("%s  %s", fmtSize(b.Size), lipgloss.NewStyle().Foreground(chipCol).Render(chip))
		lines = append(lines, padLine(sizeStr, inner))
	} else {
		lines = append(lines, padLine(dim.Render(fmtSize(b.Size)), inner))
	}

	if h > 4 {
		lines = append(lines, strings.Repeat(" ", inner))
	}

	if h > 5 && b.Description != "" {
		descLines := wrapText(b.Description, inner)
		remaining := h - len(lines)
		shown := 0
		for _, dl := range descLines[min(offset, len(descLines)):] {
			if shown >= remaining {
				break
			}
			lines = append(lines, padLine(accent.Render(truncate(dl, inner)), inner))
			shown++
		}
	}

	for len(lines) < h {
		lines = append(lines, strings.Repeat(" ", inner))
	}
	if len(lines) > h {
		lines = lines[:h]
	}
	return lines
}

func bookChip(b *Audiobook) (string, lipgloss.Color) {
	switch b.State {
	case DownloadReady:
		return "[downloaded]", colReady
	case DownloadInProgress:
		return fmt.Sprintf("[downloading %3d%%]", int(b.DownloadProgress*100)), colPending
	case DownloadPreparing:
		return "[queue]", colPending
	default:
		if !b.ArchiveReady {
			return "[building]", colPending
		}
		return "[download]", colUnfocus
	}
}

func fmtSize(bytes int64) string {
	switch {
	case bytes >= 1<<30:
		return fmt.Sprintf("%.1f GB", float64(bytes)/float64(1<<30))
	case bytes >= 1<<20:
		return fmt.Sprintf("%.0f MB", float64(bytes)/float64(1<<20))
	default:
		return fmt.Sprintf("%d KB", bytes>>10)
	}
}

func wrapText(text string, width int) []string {
	if width <= 0 {
		return nil
	}
	words := strings.Fields(text)
	var lines []string
	line := ""
	for _, w := range words {
		if line == "" {
			line = w
		} else if len(line)+1+len(w) <= width {
			line += " " + w
		} else {
			lines = append(lines, line)
			line = w
		}
	}
	if line != "" {
		lines = append(lines, line)
	}
	return lines
}

//  Player bar (2 lines)

func (ps *PlayerState) findPlayingBook() *Audiobook {
	for i := range ps.lib.Books {
		if ps.lib.Books[i].Hash == ps.lib.PlayingHash {
			return &ps.lib.Books[i]
		}
	}
	return nil
}

func (ps *PlayerState) renderPlayerBar() string {
	volumeW := 30

	remStr := ps.remainingString()
	remW := len(remStr)
	infoW := ps.windowWidth - remW

	speedStr := fmt.Sprintf("x%.2f ", ps.playerSpeed)
	speedW := len(speedStr)
	progressW := ps.windowWidth - volumeW - speedW

	gray := lipgloss.NewStyle().Foreground(colUnfocus)
	var speedRendered string
	if ps.playerSpeed != 1.0 {
		speedRendered = lipgloss.NewStyle().Foreground(colReady).Bold(true).Render(speedStr)
	} else {
		speedRendered = gray.Render(speedStr)
	}
	line1 := ps.buildInfoSection(infoW) + gray.Render(remStr)
	line2 := ps.buildVolumeSection(volumeW) + ps.buildProgressSection(progressW) + speedRendered
	return line1 + "\n" + line2
}

func (ps *PlayerState) buildInfoSection(w int) string {
	gray := lipgloss.NewStyle().Foreground(colUnfocus)
	accent := lipgloss.NewStyle().Foreground(colReady)

	if ps.conflictHash != "" {
		prompt := " Conflict: [y] keep local  [n] take server"
		return padLine(lipgloss.NewStyle().Foreground(colPending).Render(truncate(prompt, w)), w)
	}

	if ps.statusMsg != "" {
		col := colUnfocus
		if ps.statusErr {
			col = colError
		}
		return padLine(lipgloss.NewStyle().Foreground(col).Render(" "+truncate(ps.statusMsg, w-1)), w)
	}

	if ps.mode == ModeSearch {
		cursor := lipgloss.NewStyle().Foreground(colReady).Render("█")
		query := truncate(ps.searchQuery, w-3)
		return padLine(" "+gray.Render("/ "+query)+cursor, w)
	}

	if ps.mode == ModeSearching {
		counter := "[0/0] "
		if len(ps.searchMatches) > 0 {
			counter = fmt.Sprintf("[%d/%d] ", ps.searchMatchIdx+1, len(ps.searchMatches))
		}
		query := truncate(ps.searchQuery, w-3-len(counter))
		return padLine(" "+accent.Render(counter)+gray.Render("/ "+query), w)
	}

	b := ps.findPlayingBook()
	if b == nil || b.Position == nil {
		return padLine(gray.Render(" Not playing"), w)
	}

	playingLabel := "Playing"
	if ps.playerPaused {
		playingLabel = "Paused"
	}
	chTitle := ""
	if b.Position.ChapterIndex < len(b.Chapters) {
		chTitle = b.Chapters[b.Position.ChapterIndex].Title
	}

	// overhead: " " + label + " " + " from " + " by " = len(label) + 12
	overhead := len(playingLabel) + 12
	avail := w - overhead
	if avail < 6 {
		return padLine(gray.Render(" "+playingLabel), w)
	}
	chW := avail / 3
	titleW := avail / 3
	authW := avail - chW - titleW

	line := " " + gray.Render(playingLabel+" ") +
		accent.Render(truncate(chTitle, chW)) +
		gray.Render(" from ") +
		accent.Render(truncate(b.Title, titleW)) +
		gray.Render(" by ") +
		accent.Render(truncate(b.Author, authW))
	return padLine(line, w)
}


func (ps *PlayerState) buildVolumeSection(w int) string {
	gray := lipgloss.NewStyle().Foreground(colUnfocus)
	prefix := " Vol "
	suffix := fmt.Sprintf(" %d%%", ps.playerVolume)
	barW := w - len(prefix) - len(suffix)
	if barW < 1 {
		return gray.Render(strings.Repeat(" ", w))
	}
	pos := barW * ps.playerVolume / 100
	if pos >= barW {
		pos = barW - 1
	}
	bar := strings.Repeat("─",pos) + "█" + strings.Repeat("─",barW-pos-1)
	return gray.Render(padLine(prefix+bar+suffix, w))
}

func (ps *PlayerState) buildProgressSection(w int) string {
	gray := lipgloss.NewStyle().Foreground(colUnfocus)
	b := ps.findPlayingBook()
	var elapsed, total int64
	if b != nil && b.Position != nil {
		chIdx := b.Position.ChapterIndex
		elapsed = b.Position.ChapterPosition
		if chIdx < len(b.Chapters) {
			total = b.Chapters[chIdx].Duration
		}
	}
	// use same width for both timestamps so the bar is stable
	tClock := fmtClock(total)
	eClock := fmtClock(elapsed)
	tsW := len(tClock)
	if len(eClock) > tsW {
		tsW = len(eClock)
	}
	eStr := fmt.Sprintf(" %*s ", tsW, eClock)
	tStr := fmt.Sprintf(" %*s ", tsW, tClock)
	barW := w - len(eStr) - len(tStr)
	if barW < 1 {
		return gray.Render(strings.Repeat(" ", w))
	}
	var pos int
	if total > 0 {
		pos = int(int64(barW) * elapsed / total)
	}
	if pos >= barW {
		pos = barW - 1
	}
	bar := strings.Repeat("─",pos) + "█" + strings.Repeat("─",barW-pos-1)
	return gray.Render(padLine(eStr+bar+tStr, w))
}

func (ps *PlayerState) remainingString() string {
	b := ps.findPlayingBook()
	if b == nil {
		return ""
	}
	elapsed := ps.bookElapsed(b)
	remaining := b.Duration - elapsed
	if remaining < 0 {
		remaining = 0
	}
	if ps.playerSpeed > 0 {
		remaining = int64(float64(remaining) / ps.playerSpeed)
	}
	secs := remaining / 1000
	h := secs / 3600
	m := (secs % 3600) / 60
	if h > 0 {
		return fmt.Sprintf("%dh%02dm remaining ", h, m)
	}
	return fmt.Sprintf("%dm remaining ", secs/60)
}

//  Help overlay 

type helpEntry struct{ key, desc string }

func (ps *PlayerState) helpEntries() []helpEntry {
	common := []helpEntry{
		{"?  q  Esc", "close help"},
		{"Ctrl+C", "force quit"},
	}
	switch ps.helpForMode {
	case ModeMain:
		return append([]helpEntry{
			{"j / k / ↑ / ↓", "scroll list"},
			{"h / l", "move panel focus"},
			{"d", "toggle library"},
			{"i", "toggle info pane"},
			{"I", "focus info pane"},
			{"Enter", "download / cancel"},
			{"p", "play selected"},
			{"Space", "play / pause"},
			{"s", "cycle speed"},
			{"x", "delete local book"},
			{"/", "search albums"},
			{"r", "refresh from server"},
			{"q", "quit"},
			{"Q", "logout + quit"},
		}, common...)
	case ModeSearch:
		return append([]helpEntry{
			{"printable", "append to query"},
			{"Backspace", "delete char"},
			{"Enter", "confirm query"},
			{"Esc", "cancel search"},
		}, common...)
	case ModeSearching:
		return append([]helpEntry{
			{"n / p", "next / prev match"},
			{"/", "re-edit query"},
			{"Enter", "confirm"},
			{"Esc", "cancel"},
		}, common...)
	}
	return common
}

func (ps *PlayerState) renderHelp() string {
	entries := ps.helpEntries()

	keyW := 12
	descW := 24
	boxW := keyW + descW + 5 // borders + spacing

	var sb strings.Builder
	sb.WriteString("  Key Bindings\n\n")
	for _, e := range entries {
		sb.WriteString(fmt.Sprintf("  %-*s  %s\n", keyW, e.key, e.desc))
	}

	content := sb.String()
	boxH := strings.Count(content, "\n") + 3

	// center in terminal
	leftPad := (ps.windowWidth - boxW) / 2
	topPad := (ps.windowHeight - boxH) / 2
	if leftPad < 0 {
		leftPad = 0
	}
	if topPad < 0 {
		topPad = 0
	}

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colFocused).
		Padding(0, 1).
		Render(content)

	var out strings.Builder
	empty := strings.Repeat(" ", ps.windowWidth)
	for i := 0; i < topPad; i++ {
		out.WriteString(empty + "\n")
	}
	for _, line := range strings.Split(box, "\n") {
		out.WriteString(strings.Repeat(" ", leftPad) + line + "\n")
	}
	return out.String()
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
