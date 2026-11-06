package tray

import "testing"

func TestPauseLabelNamesTheAction(t *testing.T) {
	t.Parallel()
	if pauseLabel(false) != labelPause || pauseLabel(true) != labelResume {
		t.Error("pause label wrong")
	}
	called := false
	invoke(func() { called = true })
	invoke(nil)
	if !called {
		t.Error("invoke did not call")
	}
}
