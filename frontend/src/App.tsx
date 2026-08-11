import { useState, useCallback } from 'react'
import { TabBar } from './components/TabBar'
import { Home } from './components/Home'
import { DocsEditor } from './components/editors/DocsEditor'
import { SheetsEditor } from './components/editors/SheetsEditor'
import { SlidesEditor } from './components/editors/SlidesEditor'
import { PdfViewer } from './components/editors/PdfViewer'
import { MarkdownEditor } from './components/editors/MarkdownEditor'
import { AiPanel } from './components/AiPanel'
import { SettingsModal } from './components/SettingsModal'
import type { TabInfo } from './wails'
import './styles/app.css'

let nextTabId = 1
function genTabId(): string {
  return `tab-${nextTabId++}`
}

export function App() {
  const [tabs, setTabs] = useState<TabInfo[]>([
    { id: 'home', kind: 'home', title: 'Home', active: true },
  ])
  const [showSettings, setShowSettings] = useState(false)
  const [showAiPanel, setShowAiPanel] = useState(false)

  const activeTab = tabs.find((t) => t.active)

  const activateTab = useCallback((id: string) => {
    setTabs((prev) => prev.map((t) => ({ ...t, active: t.id === id })))
  }, [])

  const closeTab = useCallback((id: string) => {
    if (id === 'home') return
    setTabs((prev) => {
      const idx = prev.findIndex((t) => t.id === id)
      const next = prev.filter((t) => t.id !== id)
      if (prev[idx]?.active && next.length > 0) {
        const newActive = Math.min(idx, next.length - 1)
        next[newActive] = { ...next[newActive], active: true }
      }
      return next
    })
  }, [])

  const openNewTab = useCallback(
    (kind: TabInfo['kind'], title: string, filePath?: string) => {
      const id = genTabId()
      setTabs((prev) => [
        ...prev.map((t) => ({ ...t, active: false })),
        { id, kind, title, filePath, active: true },
      ])
      return id
    },
    [],
  )

  const updateTabTitle = useCallback((id: string, title: string) => {
    setTabs((prev) => prev.map((t) => (t.id === id ? { ...t, title } : t)))
  }, [])

  const setTabDirty = useCallback((id: string, isDirty: boolean) => {
    setTabs((prev) => prev.map((t) => (t.id === id ? { ...t, isDirty } : t)))
  }, [])

  const renderContent = () => {
    if (!activeTab) return null
    switch (activeTab.kind) {
      case 'home':
        return <Home onOpenTab={openNewTab} />
      case 'docs':
        return (
          <DocsEditor
            tabId={activeTab.id}
            filePath={activeTab.filePath}
            onTitleChange={(t) => updateTabTitle(activeTab.id, t)}
            onDirtyChange={(d) => setTabDirty(activeTab.id, d)}
          />
        )
      case 'sheets':
        return (
          <SheetsEditor
            tabId={activeTab.id}
            filePath={activeTab.filePath}
            onTitleChange={(t) => updateTabTitle(activeTab.id, t)}
            onDirtyChange={(d) => setTabDirty(activeTab.id, d)}
          />
        )
      case 'slides':
        return (
          <SlidesEditor
            tabId={activeTab.id}
            filePath={activeTab.filePath}
            onTitleChange={(t) => updateTabTitle(activeTab.id, t)}
            onDirtyChange={(d) => setTabDirty(activeTab.id, d)}
          />
        )
      case 'pdf':
        return <PdfViewer tabId={activeTab.id} filePath={activeTab.filePath} />
      case 'markdown':
        return (
          <MarkdownEditor
            tabId={activeTab.id}
            filePath={activeTab.filePath}
            onTitleChange={(t) => updateTabTitle(activeTab.id, t)}
            onDirtyChange={(d) => setTabDirty(activeTab.id, d)}
          />
        )
    }
  }

  return (
    <div className="app-root">
      <TabBar
        tabs={tabs}
        onActivate={activateTab}
        onClose={closeTab}
        onNewTab={() => openNewTab('docs', 'Untitled')}
        onSettings={() => setShowSettings(true)}
      />
      <div className="app-body">
        <div className="app-content">{renderContent()}</div>
        {showAiPanel && activeTab && activeTab.kind !== 'home' && (
          <AiPanel
            tabId={activeTab.id}
            tabKind={activeTab.kind}
            onClose={() => setShowAiPanel(false)}
          />
        )}
      </div>
      {activeTab && activeTab.kind !== 'home' && (
        <div className="app-statusbar">
          <button className="ai-toggle" onClick={() => setShowAiPanel((v) => !v)}>
            🪶 Quill AI
          </button>
          <span className="statusbar-text">
            {activeTab.isDirty ? '● Modified' : 'Saved'}
          </span>
        </div>
      )}
      {showSettings && <SettingsModal onClose={() => setShowSettings(false)} />}
    </div>
  )
}
