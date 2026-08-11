import { useState, useCallback, useEffect, useRef } from 'react'
import { DocsService, WailsRuntime } from '../../services/wails-bridge'
import { EditorToolbar } from './EditorToolbar'
import './DocsEditor.css'

interface DocsEditorProps {
  tabId: string
  filePath?: string
  onTitleChange: (title: string) => void
  onDirtyChange: (dirty: boolean) => void
}

export function DocsEditor({ tabId, filePath, onTitleChange, onDirtyChange }: DocsEditorProps) {
  const [wordCount, setWordCount] = useState(0)
  const [pageCount, setPageCount] = useState(1)
  const [isBold, setIsBold] = useState(false)
  const [isItalic, setIsItalic] = useState(false)
  const [saveError, setSaveError] = useState<string | null>(null)
  const editorRef = useRef<HTMLDivElement>(null)

  // Open file or create blank via Go backend
  useEffect(() => {
    const init = async () => {
      try {
        if (filePath) {
          const result = await DocsService.openFile(tabId, filePath)
          if (result?.title) onTitleChange(result.title)
          if (result?.error) setSaveError(result.error)
        } else {
          const result = await DocsService.newBlank(tabId)
          if (result?.title) onTitleChange(result.title)
          else onTitleChange('Untitled Document')
        }
        // Sync initial state
        const state = await DocsService.getState(tabId)
        if (state) {
          setWordCount(state.word_count)
          setPageCount(state.page_count)
        }
      } catch {
        // Fallback: set title from path
        onTitleChange(filePath?.split(/[/\\]/).pop() || 'Untitled Document')
      }
    }
    init()
    return () => { DocsService.close(tabId) }
  }, [filePath, tabId, onTitleChange])

  const handleInput = useCallback(() => {
    if (!editorRef.current) return
    const text = editorRef.current.innerText || ''
    const words = text.trim().split(/\s+/).filter(Boolean).length
    setWordCount(words)
    onDirtyChange(true)
  }, [onDirtyChange])

  const execCommand = useCallback((cmd: string, value?: string) => {
    document.execCommand(cmd, false, value)
    editorRef.current?.focus()
  }, [])

  const handleSave = useCallback(async () => {
    setSaveError(null)
    try {
      const result = await DocsService.save(tabId)
      if (result?.success) {
        onDirtyChange(false)
        WailsRuntime.windowSetTitle(`Quill — ${result.file_path?.split(/[/\\]/).pop() || 'Document'}`)
      } else if (result?.error) {
        setSaveError(result.error)
      }
    } catch (err) {
      setSaveError(`Save failed: ${err}`)
    }
  }, [tabId, onDirtyChange])

  const handleSaveAs = useCallback(async () => {
    const path = await WailsRuntime.saveFileDialog()
    if (!path) return
    setSaveError(null)
    try {
      const result = await DocsService.saveAs(tabId, path)
      if (result?.success) {
        onDirtyChange(false)
        onTitleChange(path.split(/[/\\]/).pop() || 'Document')
      } else if (result?.error) {
        setSaveError(result.error)
      }
    } catch (err) {
      setSaveError(`Save As failed: ${err}`)
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
        {
          id: 'bold', label: 'Bold', icon: 'B',
          onClick: () => { execCommand('bold'); setIsBold((v) => !v) },
          active: isBold,
        },
        {
          id: 'italic', label: 'Italic', icon: 'I',
          onClick: () => { execCommand('italic'); setIsItalic((v) => !v) },
          active: isItalic,
        },
        { id: 'underline', label: 'Underline', icon: 'U', onClick: () => execCommand('underline') },
        { id: 'strikethrough', label: 'Strikethrough', icon: 'S̶', onClick: () => execCommand('strikeThrough') },
      ],
    },
    {
      id: 'paragraph',
      actions: [
        { id: 'align-left', label: 'Align Left', icon: '⫷', onClick: () => execCommand('justifyLeft') },
        { id: 'align-center', label: 'Align Center', icon: '⫿', onClick: () => execCommand('justifyCenter') },
        { id: 'align-right', label: 'Align Right', icon: '⫸', onClick: () => execCommand('justifyRight') },
      ],
    },
    {
      id: 'insert',
      actions: [
        { id: 'ul', label: 'Bullet List', icon: '•', onClick: () => execCommand('insertUnorderedList') },
        { id: 'ol', label: 'Numbered List', icon: '1.', onClick: () => execCommand('insertOrderedList') },
      ],
    },
  ]

  return (
    <div className="docs-editor">
      <EditorToolbar groups={toolbarGroups} title="Document" />
      {saveError && (
        <div className="docs-editor-error" role="alert">
          ⚠️ {saveError}
          <button onClick={() => setSaveError(null)}>×</button>
        </div>
      )}
      <div className="docs-editor-page">
        <div
          ref={editorRef}
          className="docs-editor-content"
          contentEditable
          suppressContentEditableWarning
          onInput={handleInput}
          onKeyDown={(e) => {
            if ((e.metaKey || e.ctrlKey) && e.key === 's') {
              e.preventDefault()
              handleSave()
            }
          }}
          data-placeholder="Start typing your document..."
        />
      </div>
      <div className="docs-editor-footer">
        <span>{wordCount} words</span>
        <span>Page 1 of {pageCount}</span>
      </div>
    </div>
  )
}
