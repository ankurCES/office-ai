import { useState, useCallback, useEffect } from 'react'
import type { TabInfo } from '../wails'
import { ShellService, WailsRuntime } from '../services/wails-bridge'
import type { RecentFile } from '../services/wails-bridge'
import { DocIcon, SheetIcon, SlideIcon, PdfIcon, MarkdownIcon } from './icons/FileIcons'
import './Home.css'

interface HomeProps {
  onOpenTab: (kind: TabInfo['kind'], title: string, filePath?: string) => string
}

const NEW_ITEMS = [
  { kind: 'docs' as const, label: 'Document', icon: DocIcon, color: 'var(--color-docs)' },
  { kind: 'sheets' as const, label: 'Spreadsheet', icon: SheetIcon, color: 'var(--color-sheets)' },
  { kind: 'slides' as const, label: 'Presentation', icon: SlideIcon, color: 'var(--color-slides)' },
  { kind: 'markdown' as const, label: 'Markdown', icon: MarkdownIcon, color: 'var(--color-markdown)' },
]

const EXT_TO_KIND: Record<string, TabInfo['kind']> = {
  docx: 'docs', doc: 'docs',
  xlsx: 'sheets', xls: 'sheets', csv: 'sheets',
  pptx: 'slides', ppt: 'slides',
  pdf: 'pdf',
  md: 'markdown', markdown: 'markdown',
}

export function Home({ onOpenTab }: HomeProps) {
  const [recentFiles, setRecentFiles] = useState<RecentFile[]>([])
  const [searchQuery, setSearchQuery] = useState('')

  // Load recent files from Go backend on mount
  useEffect(() => {
    ShellService.getRecentFiles().then(setRecentFiles).catch(() => {})
  }, [])

  const handleNew = useCallback(
    (kind: TabInfo['kind'], label: string) => {
      onOpenTab(kind, `Untitled ${label}`)
    },
    [onOpenTab],
  )

  const handleOpenFile = useCallback(async () => {
    try {
      const path = await WailsRuntime.openFileDialog()
      if (path) {
        const ext = path.split('.').pop()?.toLowerCase() || ''
        const kind = EXT_TO_KIND[ext] || 'docs'
        const name = path.split(/[/\\]/).pop() || 'Untitled'
        onOpenTab(kind, name, path)
      }
    } catch {
      // Dialog cancelled or not available
    }
  }, [onOpenTab])

  const handleRemoveRecent = useCallback(async (path: string) => {
    await ShellService.removeRecent(path)
    setRecentFiles((prev) => prev.filter((f) => f.path !== path))
  }, [])

  const handleToggleStar = useCallback(async (path: string) => {
    await ShellService.toggleStarred(path)
    setRecentFiles((prev) =>
      prev.map((f) => (f.path === path ? { ...f, starred: !f.starred } : f)),
    )
  }, [])

  const filteredRecent = searchQuery
    ? recentFiles.filter((f) =>
        f.title.toLowerCase().includes(searchQuery.toLowerCase()),
      )
    : recentFiles

  return (
    <div className="home">
      <div className="home-hero">
        <h1 className="home-title">
          <span className="home-logo">✨</span> Office AI
        </h1>
        <p className="home-subtitle">Create, edit, and collaborate with AI assistance</p>
      </div>

      <div className="home-section">
        <h2 className="home-section-title">Create New</h2>
        <div className="home-cards">
          {NEW_ITEMS.map(({ kind, label, icon: Icon, color }) => (
            <button
              key={kind}
              className="home-card"
              onClick={() => handleNew(kind, label)}
            >
              <div className="home-card-icon" style={{ background: color }}>
                <Icon size={32} />
              </div>
              <span className="home-card-label">{label}</span>
            </button>
          ))}
        </div>
      </div>

      <div className="home-section">
        <div className="home-section-header">
          <h2 className="home-section-title">Recent Files</h2>
          <button className="home-open-btn" onClick={handleOpenFile}>
            📂 Open File
          </button>
        </div>
        <div className="home-search">
          <input
            type="text"
            placeholder="Search recent files..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="home-search-input"
          />
        </div>
        {filteredRecent.length === 0 ? (
          <div className="home-empty">
            <p>No recent files yet. Create a new document or open an existing one.</p>
          </div>
        ) : (
          <div className="home-recent-list">
            {filteredRecent.map((file) => (
              <div key={file.path} className="home-recent-item">
                <button
                  className="home-recent-open"
                  onClick={() => {
                    const kind = (EXT_TO_KIND[file.path.split('.').pop()?.toLowerCase() || ''] || 'docs') as TabInfo['kind']
                    onOpenTab(kind, file.title, file.path)
                  }}
                >
                  <span className="home-recent-name">{file.title}</span>
                  <span className="home-recent-path">{file.path}</span>
                </button>
                <button
                  className="home-recent-star"
                  onClick={() => handleToggleStar(file.path)}
                  title={file.starred ? 'Unstar' : 'Star'}
                >
                  {file.starred ? '⭐' : '☆'}
                </button>
                <button
                  className="home-recent-remove"
                  onClick={() => handleRemoveRecent(file.path)}
                  title="Remove from recents"
                >
                  ×
                </button>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}
