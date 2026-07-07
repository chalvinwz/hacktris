package engine

// framesPerRowTable is the classic NES (NTSC) gravity curve: the number of
// 60 Hz frames a piece waits before dropping one row, indexed by level. Levels
// at or beyond the last entry use the final value (1 frame per row).
var framesPerRowTable = []int{
	48, 43, 38, 33, 28, 23, 18, 13, 8, 6, // 0-9
	5, 5, 5, // 10-12
	4, 4, 4, // 13-15
	3, 3, 3, // 16-18
	2, 2, 2, 2, 2, 2, 2, 2, 2, 2, // 19-28
	1, // 29+
}

// FramesPerRow returns the gravity delay (in frames) for a level.
func FramesPerRow(level int) int {
	if level < 0 {
		level = 0
	}
	if level >= len(framesPerRowTable) {
		return framesPerRowTable[len(framesPerRowTable)-1]
	}
	return framesPerRowTable[level]
}

// lineScores are the NES base points for clearing 1-4 rows at once, before the
// per-level multiplier.
var lineScores = [5]int{0, 40, 100, 300, 1200}

// LineScore returns the points for clearing n rows at the given level.
func LineScore(n, level int) int {
	if n < 0 || n >= len(lineScores) {
		return 0
	}
	return lineScores[n] * (level + 1)
}

// LevelFor returns the level for a total of cleared lines (NES advances one
// level every ten lines).
func LevelFor(totalLines int) int { return totalLines / 10 }

// garbageSent returns how many garbage rows a clear of n lines sends under the
// simple "lines minus one" rule.
func garbageSent(n int) int {
	if n <= 1 {
		return 0
	}
	return n - 1
}
