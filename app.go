package main

import (
	"context"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct — bound to frontend, provides file dialogs and app-level actions.
type App struct {
	ctx context.Context
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// OpenFileDialog opens a native file picker and returns the selected path.
func (a *App) OpenFileDialog() (string, error) {
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Open File",
		Filters: []runtime.FileFilter{
			{DisplayName: "Documents (*.docx)", Pattern: "*.docx;*.doc"},
			{DisplayName: "Spreadsheets (*.xlsx)", Pattern: "*.xlsx;*.xls;*.csv"},
			{DisplayName: "Presentations (*.pptx)", Pattern: "*.pptx;*.ppt"},
			{DisplayName: "PDF (*.pdf)", Pattern: "*.pdf"},
			{DisplayName: "Markdown (*.md)", Pattern: "*.md;*.markdown"},
			{DisplayName: "All Files (*.*)", Pattern: "*.*"},
		},
	})
}

// SaveFileDialog opens a native save dialog and returns the chosen path.
func (a *App) SaveFileDialog() (string, error) {
	return runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title: "Save File",
		Filters: []runtime.FileFilter{
			{DisplayName: "Documents (*.docx)", Pattern: "*.docx"},
			{DisplayName: "Spreadsheets (*.xlsx)", Pattern: "*.xlsx"},
			{DisplayName: "Presentations (*.pptx)", Pattern: "*.pptx"},
			{DisplayName: "PDF (*.pdf)", Pattern: "*.pdf"},
			{DisplayName: "Markdown (*.md)", Pattern: "*.md"},
			{DisplayName: "All Files (*.*)", Pattern: "*.*"},
		},
	})
}
