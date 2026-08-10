package shortcuts

import "testing"

func TestNewRegistry(t *testing.T) {
	r := NewRegistry()
	if r == nil {
		t.Fatal("NewRegistry() returned nil")
	}
}

func TestRegisterAndGet(t *testing.T) {
	r := NewRegistry()

	err := r.Register(Shortcut{
		ID:         "test.unique1",
		Label:      "Unique Test",
		Category:   "Test",
		Modifiers:  []Modifier{ModCtrl, ModAlt},
		Key:        "9",
		MacDisplay: "⌥⌘9",
		WinDisplay: "Ctrl+Alt+9",
	})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	sc := r.Get("test.unique1")
	if sc == nil {
		t.Fatal("Get(test.unique1) returned nil")
	}
	if sc.Label != "Unique Test" {
		t.Errorf("expected label='Unique Test', got %q", sc.Label)
	}
}

func TestAll(t *testing.T) {
	r := NewRegistry()
	all := r.All()
	if len(all) == 0 {
		t.Error("expected default shortcuts, got 0")
	}
	t.Logf("registered %d default shortcuts", len(all))
}

func TestByCategory(t *testing.T) {
	r := NewRegistry()
	cats := r.ByCategory()
	if cats == nil {
		t.Error("ByCategory returned nil")
	}
	if len(cats) == 0 {
		t.Error("expected categories from defaults")
	}
}

func TestOverride(t *testing.T) {
	r := NewRegistry()

	// Override an existing default shortcut
	sc := r.Get("file.save")
	if sc == nil {
		t.Skip("file.save not in defaults")
	}

	err := r.Override("file.save", []Modifier{ModCtrl, ModAlt, ModShift}, "9")
	if err != nil {
		t.Fatalf("Override failed: %v", err)
	}

	got := r.Get("file.save")
	if len(got.Modifiers) != 3 {
		t.Errorf("expected 3 modifiers, got %d", len(got.Modifiers))
	}
}

func TestLookup(t *testing.T) {
	r := NewRegistry()

	// "file.save" default is Ctrl+S → accelerator "ctrl+s" (or "super+s" on mac)
	sc := r.Get("file.save")
	if sc == nil {
		t.Skip("file.save not in defaults")
	}
	accel := sc.Accelerator()

	id, found := r.Lookup(accel)
	if !found {
		t.Errorf("Lookup(%q) not found", accel)
	}
	if id != "file.save" {
		t.Errorf("expected id=file.save, got %q", id)
	}
}

func TestDisplayString(t *testing.T) {
	sc := Shortcut{
		MacDisplay: "⌘S",
		WinDisplay: "Ctrl+S",
	}
	display := sc.DisplayString()
	if display == "" {
		t.Error("DisplayString returned empty")
	}
	t.Logf("DisplayString = %q", display)
}

func TestAccelerator(t *testing.T) {
	sc := Shortcut{
		Modifiers: []Modifier{ModCtrl, ModShift},
		Key:       "s",
	}
	accel := sc.Accelerator()
	if accel != "ctrl+shift+s" {
		t.Errorf("expected ctrl+shift+s, got %q", accel)
	}
}

func TestPlatformShortcuts(t *testing.T) {
	result := PlatformShortcuts("⌘S")
	if result == "" {
		t.Error("PlatformShortcuts returned empty")
	}
	t.Logf("PlatformShortcuts(⌘S) = %q", result)
}

func TestResetAll(t *testing.T) {
	r := NewRegistry()
	sc := r.Get("file.save")
	if sc == nil {
		t.Skip("file.save not in defaults")
	}

	origMods := len(sc.Modifiers)
	r.Override("file.save", []Modifier{ModCtrl, ModShift, ModAlt}, "8")
	r.ResetAll()

	got := r.Get("file.save")
	if len(got.Modifiers) != origMods {
		t.Errorf("after reset: expected %d modifiers, got %d", origMods, len(got.Modifiers))
	}
}

func TestConflictDetection(t *testing.T) {
	r := NewRegistry()

	err := r.Register(Shortcut{
		ID:         "test.a",
		Label:      "A",
		Category:   "Test",
		Modifiers:  []Modifier{ModCtrl},
		Key:        "q",
		MacDisplay: "⌘Q",
		WinDisplay: "Ctrl+Q",
	})
	if err != nil {
		t.Fatalf("first register failed: %v", err)
	}

	err = r.Register(Shortcut{
		ID:         "test.b",
		Label:      "B",
		Category:   "Test",
		Modifiers:  []Modifier{ModCtrl},
		Key:        "q",
		MacDisplay: "⌘Q",
		WinDisplay: "Ctrl+Q",
	})
	if err == nil {
		t.Error("expected conflict error, got nil")
	}
}
