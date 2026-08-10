// Package shortcuts provides cross-platform keyboard shortcut management,
// mirroring GenOffice's shortcut system: Mac notation (⌘⌥⇧) auto-converted
// to Ctrl/Alt/Shift on non-Mac platforms, shortcut registration, and
// conflict detection.
package shortcuts

import (
	"fmt"
	"runtime"
	"sort"
	"strings"
	"sync"
)

// Modifier represents a keyboard modifier key.
type Modifier string

const (
	ModCtrl  Modifier = "Ctrl"
	ModAlt   Modifier = "Alt"
	ModShift Modifier = "Shift"
	ModMeta  Modifier = "Meta" // Cmd on Mac, Win on Windows
	ModSuper Modifier = "Super"
)

// Shortcut represents a keyboard shortcut binding.
type Shortcut struct {
	ID          string     `json:"id"`          // unique identifier (e.g. "file.save")
	Label       string     `json:"label"`       // display label (e.g. "Save")
	Category    string     `json:"category"`    // grouping (e.g. "File", "Edit")
	Modifiers   []Modifier `json:"modifiers"`
	Key         string     `json:"key"`         // the main key (e.g. "S", "Enter")
	MacDisplay  string     `json:"mac_display"` // e.g. "⌘S"
	WinDisplay  string     `json:"win_display"` // e.g. "Ctrl+S"
	Description string     `json:"description,omitempty"`
	Global      bool       `json:"global"`      // app-wide vs editor-specific
}

// DisplayString returns the platform-appropriate shortcut display string.
func (s Shortcut) DisplayString() string {
	if runtime.GOOS == "darwin" {
		return s.MacDisplay
	}
	return s.WinDisplay
}

// Accelerator returns the Wails-compatible accelerator string.
func (s Shortcut) Accelerator() string {
	parts := make([]string, 0, len(s.Modifiers)+1)
	for _, m := range s.Modifiers {
		switch m {
		case ModMeta:
			if runtime.GOOS == "darwin" {
				parts = append(parts, "super")
			} else {
				parts = append(parts, "ctrl")
			}
		default:
			parts = append(parts, strings.ToLower(string(m)))
		}
	}
	parts = append(parts, strings.ToLower(s.Key))
	return strings.Join(parts, "+")
}

// Registry manages keyboard shortcuts with conflict detection.
type Registry struct {
	mu        sync.RWMutex
	shortcuts map[string]*Shortcut  // id → shortcut
	keyMap    map[string]string     // accelerator → id (for conflict detection)
	custom    map[string]*Shortcut  // user overrides
}

// NewRegistry creates a new shortcut Registry with default bindings.
func NewRegistry() *Registry {
	r := &Registry{
		shortcuts: make(map[string]*Shortcut),
		keyMap:    make(map[string]string),
		custom:    make(map[string]*Shortcut),
	}
	r.registerDefaults()
	return r
}

// Get returns a shortcut by ID.
func (r *Registry) Get(id string) *Shortcut {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if s, ok := r.custom[id]; ok {
		return s
	}
	return r.shortcuts[id]
}

// All returns all registered shortcuts, sorted by category then label.
func (r *Registry) All() []Shortcut {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]Shortcut, 0, len(r.shortcuts))
	for _, s := range r.shortcuts {
		if custom, ok := r.custom[s.ID]; ok {
			result = append(result, *custom)
		} else {
			result = append(result, *s)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Category != result[j].Category {
			return result[i].Category < result[j].Category
		}
		return result[i].Label < result[j].Label
	})
	return result
}

// ByCategory returns shortcuts grouped by category.
func (r *Registry) ByCategory() map[string][]Shortcut {
	all := r.All()
	groups := make(map[string][]Shortcut)
	for _, s := range all {
		groups[s.Category] = append(groups[s.Category], s)
	}
	return groups
}

// Register adds or updates a shortcut binding.
func (r *Registry) Register(s Shortcut) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	accel := s.Accelerator()

	// Check for conflicts
	if existingID, exists := r.keyMap[accel]; exists && existingID != s.ID {
		return fmt.Errorf("shortcut conflict: %s already bound to %s", accel, existingID)
	}

	// Remove old key mapping if updating
	if old, ok := r.shortcuts[s.ID]; ok {
		delete(r.keyMap, old.Accelerator())
	}

	r.shortcuts[s.ID] = &s
	r.keyMap[accel] = s.ID
	return nil
}

// Override sets a user custom binding for an existing shortcut.
func (r *Registry) Override(id string, modifiers []Modifier, key string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	orig, ok := r.shortcuts[id]
	if !ok {
		return fmt.Errorf("unknown shortcut: %s", id)
	}

	custom := *orig
	custom.Modifiers = modifiers
	custom.Key = key
	custom.MacDisplay = buildMacDisplay(modifiers, key)
	custom.WinDisplay = buildWinDisplay(modifiers, key)

	accel := custom.Accelerator()
	if existingID, exists := r.keyMap[accel]; exists && existingID != id {
		return fmt.Errorf("shortcut conflict: %s already bound to %s", accel, existingID)
	}

	// Remove old mapping
	if old, ok := r.custom[id]; ok {
		delete(r.keyMap, old.Accelerator())
	} else {
		delete(r.keyMap, orig.Accelerator())
	}

	r.custom[id] = &custom
	r.keyMap[accel] = id
	return nil
}

// ResetAll removes all user overrides.
func (r *Registry) ResetAll() {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Rebuild keyMap from defaults only
	r.keyMap = make(map[string]string)
	for id, s := range r.shortcuts {
		r.keyMap[s.Accelerator()] = id
	}
	r.custom = make(map[string]*Shortcut)
}

// Lookup finds which shortcut ID an accelerator maps to.
func (r *Registry) Lookup(accel string) (string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	id, ok := r.keyMap[accel]
	return id, ok
}

// PlatformShortcuts converts Mac notation to platform-appropriate display.
// Mirrors GenOffice's platformShortcuts() function.
func PlatformShortcuts(macNotation string) string {
	if runtime.GOOS == "darwin" {
		return macNotation
	}

	replacer := strings.NewReplacer(
		"⌘", "Ctrl+",
		"⌥", "Alt+",
		"⇧", "Shift+",
		"⌃", "Ctrl+",
	)
	result := replacer.Replace(macNotation)
	// Clean up double-plus from consecutive modifiers
	for strings.Contains(result, "++") {
		result = strings.ReplaceAll(result, "++", "+")
	}
	return strings.TrimSuffix(result, "+")
}

func buildMacDisplay(mods []Modifier, key string) string {
	var b strings.Builder
	for _, m := range mods {
		switch m {
		case ModMeta:
			b.WriteString("⌘")
		case ModAlt:
			b.WriteString("⌥")
		case ModShift:
			b.WriteString("⇧")
		case ModCtrl:
			b.WriteString("⌃")
		}
	}
	b.WriteString(strings.ToUpper(key))
	return b.String()
}

func buildWinDisplay(mods []Modifier, key string) string {
	parts := make([]string, 0, len(mods)+1)
	for _, m := range mods {
		switch m {
		case ModMeta:
			parts = append(parts, "Ctrl")
		default:
			parts = append(parts, string(m))
		}
	}
	parts = append(parts, strings.ToUpper(key))
	return strings.Join(parts, "+")
}

// registerDefaults sets up all default keyboard shortcuts
// mirroring GenOffice's shortcut definitions.
func (r *Registry) registerDefaults() {
	defaults := []Shortcut{
		// File operations
		{ID: "file.new", Label: "New Document", Category: "File", Modifiers: []Modifier{ModMeta}, Key: "n", MacDisplay: "⌘N", WinDisplay: "Ctrl+N", Global: true},
		{ID: "file.open", Label: "Open File", Category: "File", Modifiers: []Modifier{ModMeta}, Key: "o", MacDisplay: "⌘O", WinDisplay: "Ctrl+O", Global: true},
		{ID: "file.save", Label: "Save", Category: "File", Modifiers: []Modifier{ModMeta}, Key: "s", MacDisplay: "⌘S", WinDisplay: "Ctrl+S", Global: true},
		{ID: "file.save_as", Label: "Save As", Category: "File", Modifiers: []Modifier{ModMeta, ModShift}, Key: "s", MacDisplay: "⌘⇧S", WinDisplay: "Ctrl+Shift+S", Global: true},
		{ID: "file.close_tab", Label: "Close Tab", Category: "File", Modifiers: []Modifier{ModMeta}, Key: "w", MacDisplay: "⌘W", WinDisplay: "Ctrl+W", Global: true},

		// Edit operations
		{ID: "edit.undo", Label: "Undo", Category: "Edit", Modifiers: []Modifier{ModMeta}, Key: "z", MacDisplay: "⌘Z", WinDisplay: "Ctrl+Z"},
		{ID: "edit.redo", Label: "Redo", Category: "Edit", Modifiers: []Modifier{ModMeta, ModShift}, Key: "z", MacDisplay: "⌘⇧Z", WinDisplay: "Ctrl+Shift+Z"},
		{ID: "edit.cut", Label: "Cut", Category: "Edit", Modifiers: []Modifier{ModMeta}, Key: "x", MacDisplay: "⌘X", WinDisplay: "Ctrl+X"},
		{ID: "edit.copy", Label: "Copy", Category: "Edit", Modifiers: []Modifier{ModMeta}, Key: "c", MacDisplay: "⌘C", WinDisplay: "Ctrl+C"},
		{ID: "edit.paste", Label: "Paste", Category: "Edit", Modifiers: []Modifier{ModMeta}, Key: "v", MacDisplay: "⌘V", WinDisplay: "Ctrl+V"},
		{ID: "edit.select_all", Label: "Select All", Category: "Edit", Modifiers: []Modifier{ModMeta}, Key: "a", MacDisplay: "⌘A", WinDisplay: "Ctrl+A"},
		{ID: "edit.find", Label: "Find", Category: "Edit", Modifiers: []Modifier{ModMeta}, Key: "f", MacDisplay: "⌘F", WinDisplay: "Ctrl+F"},
		{ID: "edit.replace", Label: "Find & Replace", Category: "Edit", Modifiers: []Modifier{ModMeta}, Key: "h", MacDisplay: "⌘H", WinDisplay: "Ctrl+H"},

		// Format operations
		{ID: "format.bold", Label: "Bold", Category: "Format", Modifiers: []Modifier{ModMeta}, Key: "b", MacDisplay: "⌘B", WinDisplay: "Ctrl+B"},
		{ID: "format.italic", Label: "Italic", Category: "Format", Modifiers: []Modifier{ModMeta}, Key: "i", MacDisplay: "⌘I", WinDisplay: "Ctrl+I"},
		{ID: "format.underline", Label: "Underline", Category: "Format", Modifiers: []Modifier{ModMeta}, Key: "u", MacDisplay: "⌘U", WinDisplay: "Ctrl+U"},

		// View operations
		{ID: "view.zoom_in", Label: "Zoom In", Category: "View", Modifiers: []Modifier{ModMeta}, Key: "=", MacDisplay: "⌘+", WinDisplay: "Ctrl++"},
		{ID: "view.zoom_out", Label: "Zoom Out", Category: "View", Modifiers: []Modifier{ModMeta}, Key: "-", MacDisplay: "⌘-", WinDisplay: "Ctrl+-"},
		{ID: "view.zoom_reset", Label: "Reset Zoom", Category: "View", Modifiers: []Modifier{ModMeta}, Key: "0", MacDisplay: "⌘0", WinDisplay: "Ctrl+0"},
		{ID: "view.fullscreen", Label: "Full Screen", Category: "View", Modifiers: []Modifier{ModMeta, ModCtrl}, Key: "f", MacDisplay: "⌃⌘F", WinDisplay: "F11", Global: true},

		// AI operations
		{ID: "ai.toggle_panel", Label: "Toggle AI Panel", Category: "AI", Modifiers: []Modifier{ModMeta}, Key: "j", MacDisplay: "⌘J", WinDisplay: "Ctrl+J", Global: true},
		{ID: "ai.quick_action", Label: "AI Quick Action", Category: "AI", Modifiers: []Modifier{ModMeta, ModShift}, Key: "k", MacDisplay: "⌘⇧K", WinDisplay: "Ctrl+Shift+K"},

		// Navigation
		{ID: "nav.next_tab", Label: "Next Tab", Category: "Navigation", Modifiers: []Modifier{ModCtrl}, Key: "Tab", MacDisplay: "⌃Tab", WinDisplay: "Ctrl+Tab", Global: true},
		{ID: "nav.prev_tab", Label: "Previous Tab", Category: "Navigation", Modifiers: []Modifier{ModCtrl, ModShift}, Key: "Tab", MacDisplay: "⌃⇧Tab", WinDisplay: "Ctrl+Shift+Tab", Global: true},
		{ID: "nav.settings", Label: "Settings", Category: "Navigation", Modifiers: []Modifier{ModMeta}, Key: ",", MacDisplay: "⌘,", WinDisplay: "Ctrl+,", Global: true},
	}

	for _, s := range defaults {
		r.shortcuts[s.ID] = &Shortcut{}
		*r.shortcuts[s.ID] = s
		r.keyMap[s.Accelerator()] = s.ID
	}
}
