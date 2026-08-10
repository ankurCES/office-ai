package projectstore

import "testing"

func TestNew(t *testing.T) {
	store := New()
	if store == nil {
		t.Fatal("New() returned nil")
	}
}

func TestCreateAndGetProject(t *testing.T) {
	store := New()

	proj, err := store.CreateProject(ProjectInfo{
		Name:     "Test Project",
		Kind:     "docs",
		FilePath: "/tmp/test.docx",
	})
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}
	if proj == nil || proj.ID == "" {
		t.Fatal("CreateProject returned nil or empty ID")
	}

	got, err := store.GetProject(proj.ID)
	if err != nil {
		t.Fatalf("GetProject failed: %v", err)
	}
	if got.Name != "Test Project" {
		t.Errorf("expected name 'Test Project', got %q", got.Name)
	}
	if got.Kind != "docs" {
		t.Errorf("expected kind 'docs', got %q", got.Kind)
	}

	// Cleanup
	store.DeleteProject(proj.ID)
}

func TestListProjects(t *testing.T) {
	store := New()

	p1, _ := store.CreateProject(ProjectInfo{Name: "Doc 1", Kind: "docs", FilePath: "/tmp/a.docx"})
	p2, _ := store.CreateProject(ProjectInfo{Name: "Sheet 1", Kind: "sheets", FilePath: "/tmp/b.xlsx"})

	projects := store.ListProjects()
	if len(projects) < 2 {
		t.Errorf("expected at least 2 projects, got %d", len(projects))
	}

	// Cleanup
	store.DeleteProject(p1.ID)
	store.DeleteProject(p2.ID)
}

func TestDeleteProject(t *testing.T) {
	store := New()

	proj, _ := store.CreateProject(ProjectInfo{Name: "To Delete", Kind: "docs", FilePath: "/tmp/del.docx"})
	err := store.DeleteProject(proj.ID)
	if err != nil {
		t.Fatalf("DeleteProject failed: %v", err)
	}

	_, err = store.GetProject(proj.ID)
	if err == nil {
		t.Error("expected error after delete")
	}
}

func TestUpdateProject(t *testing.T) {
	store := New()

	proj, _ := store.CreateProject(ProjectInfo{Name: "Original", Kind: "docs", FilePath: "/tmp/orig.docx"})
	err := store.UpdateProject(proj.ID, ProjectInfo{Name: "Updated", Kind: "docs", FilePath: "/tmp/orig.docx"})
	if err != nil {
		t.Fatalf("UpdateProject failed: %v", err)
	}

	got, _ := store.GetProject(proj.ID)
	if got.Name != "Updated" {
		t.Errorf("expected name 'Updated', got %q", got.Name)
	}

	// Cleanup
	store.DeleteProject(proj.ID)
}

func TestSaveAndLoadChat(t *testing.T) {
	store := New()

	proj, _ := store.CreateProject(ProjectInfo{Name: "Chat Test", Kind: "docs", FilePath: "/tmp/chat.docx"})

	chat := ChatMeta{
		ID:        "chat-1",
		ProjectID: proj.ID,
		Messages: []ChatMessage{
			{Role: "user", Text: "Hello"},
			{Role: "assistant", Text: "Hi there"},
		},
	}
	err := store.SaveChat(chat)
	if err != nil {
		t.Fatalf("SaveChat failed: %v", err)
	}

	loaded, err := store.LoadChat(proj.ID, "chat-1")
	if err != nil {
		t.Fatalf("LoadChat failed: %v", err)
	}
	if len(loaded.Messages) != 2 {
		t.Errorf("expected 2 messages, got %d", len(loaded.Messages))
	}
	if loaded.Messages[0].Text != "Hello" {
		t.Errorf("expected first message 'Hello', got %q", loaded.Messages[0].Text)
	}

	// Cleanup
	store.DeleteProject(proj.ID)
}
