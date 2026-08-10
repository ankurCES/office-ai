import { useState, useCallback, useEffect, useRef } from 'react'
import { MarkdownService, WailsRuntime } from '../../services/wails-bridge'
import { EditorToolbar } from './EditorToolbar'
import './MarkdownEditor.css'

interface MarkdownEditorProps {
  tabId: string
  filePath?: string
  onTitleChange: (title: string) => void
  onDirtyChange: (dirty: boolean) => void
}

/** Minimal markdown-to-HTML converter for preview */
function markdownToHtml(md: string): string {
  return md
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/^### (.+)$/gm, '<h3>$1</h3>')
    .replace(/^## (.+)$/gm, '<h2>$1</h2>')
    .replace(/^# (.+)$/gm, '<h1>$1</h1>')
    .replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>')
    .replace(/\*(.+?)\*/g, '<em>$1</em>')
    .replace(/```(\w*)\n([\s\S]*?)```/g, '<pre><code class="lang-$1">$2</code></pre>')
    .replace(/`(.+?)`/g, '<code>$1</code>')
    .replace(/\[(.+?)\]\((.+?)\)/g, '<a href="$2">$1</a>')
    .replace(/^\- (.+)$/gm, '<li>$1</li>')
    .replace(/\n\n/g, '</p><p>')
    .replace(/^(.+)$/gm, (_, line) =>
      line.startsWith('<') ? line : `<p>${line}</p>`,
    )
}

export function MarkdownEditor({ tabId, filePath, onTitleChange, onDirtyChange }: MarkdownEditorProps) {
  const [content, setContent] = useState('')
  const [showPreview, setShowPreview] = useState(true)
  const [wordCount, setWordCount] = useState(0)
  const textareaRef = useRef<HTMLTextAreaElement>(null)

  // Open file or create blank via Go backend
  useEffect(() => {
    const init = async () => {
      try {
        let result: Record<string, unknown>
        if (filePath) {
          result = await MarkdownService.openFile(tabId, filePath)
        } else {
          result = await MarkdownService.newBlank(tabId)
        }
        if (result?.title) onTitleChange(result.title as string)
        else onTitleChange(filePath?.split(/[/\\]/).pop() || 'Untitled.md')
        if (result?.content) {
          setContent(result.content as string)
          const words = (result.content as string).trim().split(/\s+/).filter(Boolean).length
          setWordCount(words)
        }
      } catch {
        onTitleChange(filePath?.split(/[/\\]/).pop() || 'Untitled.md')
      }
    }
    init()
    return () => { MarkdownService.close(tabId) }
  }, [filePath, tabId, onTitleChange])

  const handleChange = useCallback(
    (value: string) => {
      setContent(value)
      const words = value.trim().split(/\s+/).filter(Boolean).length
      setWordCount(words)
      onDirtyChange(true)
      // Sync content to Go backend (debounced in production)
      MarkdownService.updateContent(tabId, value).catch(() => {})
    },
    [tabId, onDirtyChange],
  )

  const insertAtCursor = useCallback(
    (before: string, after: string = '') => {
      const ta = textareaRef.current
      if (!ta) return
      const start = ta.selectionStart
      const end = ta.selectionEnd
      const selected = content.substring(start, end)
      const newContent =
        content.substring(0, start) + before + selected + after + content.substring(end)
      setContent(newContent)
      onDirtyChange(true)
      MarkdownService.updateContent(tabId, newContent).catch(() => {})
      requestAnimationFrame(() => {
        ta.focus()
        ta.setSelectionRange(start + before.length, start + before.length + selected.length)
      })
    },
    [content, tabId, onDirtyChange],
  )

  const handleSave = useCallback(async () => {
    try {
      const result = await MarkdownService.save(tabId)
      if (result?.success) onDirtyChange(false)
    } catch (err) {
      console.error('Save failed:', err)
    }
  }, [tabId, onDirtyChange])

  const handleSaveAs = useCallback(async () => {
    const path = await WailsRuntime.saveFileDialog()
    if (!path) return
    try {
      const result = await MarkdownService.saveAs(tabId, path)
      if (result?.success) {
        onDirtyChange(false)
        onTitleChange(path.split(/[/\\]/).pop() || 'Untitled.md')
      }
    } catch (err) {
      console.error('Save As failed:', err)
    }
  }, [tabId, onDirtyChange, onTitleChange])

  const toolbarGroups = [
    {
      id: 'file',
      actions: [
        { id: 'save', label: 'Save', icon: '💾', onClick: handleSave },
        { id: 'save-as', label: 'Save As', icon: '📄', onClick: handleSaveAs },
      ],
    },
    {
      id: 'format',
      actions: [
        { id: 'bold', label: 'Bold', icon: 'B', onClick: () => insertAtCursor('**', '**') },
        { id: 'italic', label: 'Italic', icon: 'I', onClick: () => insertAtCursor('*', '*') },
        { id: 'code', label: 'Code', icon: '⟨⟩', onClick: () => insertAtCursor('`', '`') },
        { id: 'link', label: 'Link', icon: '🔗', onClick: () => insertAtCursor('[', '](url)') },
      ],
    },
    {
      id: 'heading',
      actions: [
        { id: 'h1', label: 'H1', icon: 'H1', onClick: () => insertAtCursor('# ') },
        { id: 'h2', label: 'H2', icon: 'H2', onClick: () => insertAtCursor('## ') },
        { id: 'h3', label: 'H3', icon: 'H3', onClick: () => insertAtCursor('### ') },
      ],
    },
    {
      id: 'insert',
      actions: [
        { id: 'list', label: 'List', icon: '•', onClick: () => insertAtCursor('- ') },
        { id: 'codeblock', label: 'Code Block', icon: '```', onClick: () => insertAtCursor('```\n', '\n```') },
      ],
    },
    {
      id: 'view',
      actions: [
        {
          id: 'preview', label: 'Toggle Preview', icon: '👁',
          onClick: () => setShowPreview((v) => !v),
          active: showPreview,
        },
      ],
    },
  ]

  return (
    <div className="md-editor">
      <EditorToolbar groups={toolbarGroups} title="Markdown" />
      <div className="md-body">
        <div className={`md-source ${showPreview ? 'split' : 'full'}`}>
          <textarea
            ref={textareaRef}
            className="md-textarea"
            value={content}
            onChange={(e) => handleChange(e.target.value)}
            onKeyDown={(e) => {
              if ((e.metaKey || e.ctrlKey) && e.key === 's') {
                e.preventDefault()
                handleSave()
              }
            }}
            placeholder="# Start writing markdown..."
            spellCheck={false}
          />
        </div>
        {showPreview && (
          <div className="md-preview">
            <div
              className="md-preview-content"
              dangerouslySetInnerHTML={{ __html: markdownToHtml(content) }}
            />
          </div>
        )}
      </div>
      <div className="md-footer">
        <span>{wordCount} words</span>
        <span>{content.split('\n').length} lines</span>
      </div>
    </div>
  )
}
