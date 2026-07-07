package client

import (
	"strings"
	"testing"
	"time"

	"github.com/chalvinwz/hacktris/internal/engine"
)

// Advancing the model through many frames must keep rendering a well-shaped
// playfield and panel without panicking.
func TestRenderSmoke(t *testing.T) {
	m := New()
	m.width, m.height = 80, 30

	for i := 0; i < 600; i++ {
		next, _ := m.Update(tickMsg(time.Time{}))
		m = next.(Model)
	}

	well := m.renderWell()
	if want := engine.Height*m.wellScale() - 1; strings.Count(well, "\n") != want {
		t.Fatalf("well has %d newlines, want %d", strings.Count(well, "\n"), want)
	}
	body := m.gameBody()
	for _, want := range []string{"HOLD", "NEXT", "SCORE"} {
		if !strings.Contains(body, want) {
			t.Fatalf("game body missing %q", want)
		}
	}
	if bar := m.titleBar(m.width); !strings.Contains(bar, "HACKTRIS") {
		t.Fatalf("title bar missing app name")
	}

	// View construction must not panic.
	_ = m.View()
}
