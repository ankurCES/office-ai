package main

import (
	"context"
	"embed"

	"github.com/ankurCES/office-ai/internal/docs"
	"github.com/ankurCES/office-ai/internal/markdown"
	"github.com/ankurCES/office-ai/internal/pdf"
	"github.com/ankurCES/office-ai/internal/sheets"
	"github.com/ankurCES/office-ai/internal/shell"
	"github.com/ankurCES/office-ai/internal/slides"
	"github.com/ankurCES/office-ai/pkg/agentcore"
	"github.com/ankurCES/office-ai/pkg/aiprovider"
	"github.com/ankurCES/office-ai/pkg/i18n"
	"github.com/ankurCES/office-ai/pkg/projectstore"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Shared services
	i18nSvc := i18n.New()
	store := projectstore.New()
	aiProvider := aiprovider.New()
	agentLoop := agentcore.New(aiProvider)

	// App-level service (file dialogs, etc.)
	app := NewApp()

	// Module services (mirror GenOffice app architecture)
	shellSvc := shell.New(i18nSvc)
	docsSvc := docs.New(i18nSvc, store, agentLoop)
	sheetsSvc := sheets.New(i18nSvc, store, agentLoop)
	slidesSvc := slides.New(i18nSvc, store, agentLoop)
	pdfSvc := pdf.New(i18nSvc, agentLoop)
	markdownSvc := markdown.New(i18nSvc, store, agentLoop)

	err := wails.Run(&options.App{
		Title:            "Quill",
		Width:            1440,
		Height:           900,
		MinWidth:         800,
		MinHeight:        600,
		DisableResize:    false,
		Fullscreen:       false,
		WindowStartState: options.Normal,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 255, G: 255, B: 255, A: 1},
		OnStartup: func(ctx context.Context) {
			app.startup(ctx)
			shellSvc.OnStartup(ctx)
		},
		OnShutdown:       shellSvc.OnShutdown,
		Bind: []interface{}{
			app,
			shellSvc,
			docsSvc,
			sheetsSvc,
			slidesSvc,
			pdfSvc,
			markdownSvc,
			i18nSvc,
			store,
			aiProvider,
			agentLoop,
		},
	})
	if err != nil {
		println("Error:", err.Error())
	}
}
