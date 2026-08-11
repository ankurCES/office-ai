import { useState, useEffect, useCallback } from 'react'
import { PdfService, WailsRuntime } from '../../services/wails-bridge'
import type { PDFState } from '../../services/wails-bridge'
import { EditorToolbar } from './EditorToolbar'
import './PdfViewer.css'

interface PdfViewerProps {
  tabId: string
  filePath?: string
}

export function PdfViewer({ tabId, filePath }: PdfViewerProps) {
  const [pdfState, setPdfState] = useState<PDFState | null>(null)
  const [currentPage, setCurrentPage] = useState(1)
  const [extractedText, setExtractedText] = useState('')
  const [showText, setShowText] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!filePath) return
    const init = async () => {
      try {
        const result = await PdfService.openFile(tabId, filePath)
        if (result?.error) {
          setError(result.error as string)
          return
        }
        // Fetch full state
        const state = await PdfService.getState(tabId)
        if (state) setPdfState(state)
      } catch (err) {
        setError(`Failed to open PDF: ${err}`)
      }
    }
    init()
    return () => { PdfService.close(tabId) }
  }, [filePath, tabId])

  const handleExtractText = useCallback(async () => {
    try {
      const text = await PdfService.extractText(tabId)
      setExtractedText(text)
      setShowText(true)
    } catch (err) {
      setError(`Failed to extract text: ${err}`)
    }
  }, [tabId])

  const handleSplitPages = useCallback(async () => {
    try {
      const dir = await WailsRuntime.saveFileDialog()
      if (!dir) return
      const result = await PdfService.splitFile(tabId, dir)
      if (result?.error) setError(result.error)
    } catch (err) {
      setError(`Split failed: ${err}`)
    }
  }, [tabId])

  const handleRotate = useCallback(async (degrees: number) => {
    try {
      const result = await PdfService.rotatePages(tabId, degrees, [currentPage])
      if (result?.success) {
        const state = await PdfService.getState(tabId)
        if (state) setPdfState(state)
      }
    } catch (err) {
      setError(`Rotate failed: ${err}`)
    }
  }, [tabId, currentPage])

  const handleOptimize = useCallback(async () => {
    try {
      const path = await WailsRuntime.saveFileDialog()
      if (!path) return
      const result = await PdfService.optimize(tabId, path)
      if (result?.error) setError(result.error)
    } catch (err) {
      setError(`Optimize failed: ${err}`)
    }
  }, [tabId])

  const handleSave = useCallback(async () => {
    try {
      const result = await PdfService.save(tabId)
      if (result?.error) setError(result.error)
    } catch (err) {
      setError(`Save failed: ${err}`)
    }
  }, [tabId])

  const handleMerge = useCallback(async () => {
    try {
      const path = await WailsRuntime.openFileDialog()
      if (!path) return
      const outputPath = await WailsRuntime.saveFileDialog()
      if (!outputPath) return
      const result = await PdfService.mergeFiles([filePath || '', path], outputPath)
      if (result?.error) setError(result.error)
    } catch (err) {
      setError(`Merge failed: ${err}`)
    }
  }, [filePath])

  const handleWatermark = useCallback(async () => {
    const text = prompt('Enter watermark text:')
    if (!text) return
    try {
      const outputPath = await WailsRuntime.saveFileDialog()
      if (!outputPath) return
      const result = await PdfService.addWatermark(tabId, text, outputPath)
      if (result?.error) setError(result.error)
    } catch (err) {
      setError(`Watermark failed: ${err}`)
    }
  }, [tabId])

  const pageCount = pdfState?.page_count || 0

  const toolbarGroups = [
    {
      id: 'file',
      actions: [
        { id: 'save', label: 'Save', icon: '💾', onClick: handleSave },
      ],
    },
    {
      id: 'navigate',
      actions: [
        { id: 'prev', label: '◀', icon: '◀', onClick: () => setCurrentPage(p => Math.max(1, p - 1)), disabled: currentPage <= 1 },
        { id: 'next', label: '▶', icon: '▶', onClick: () => setCurrentPage(p => Math.min(pageCount, p + 1)), disabled: currentPage >= pageCount },
      ],
    },
    {
      id: 'tools',
      actions: [
        { id: 'extract', label: 'Extract Text', icon: '📝', onClick: handleExtractText },
        { id: 'split', label: 'Split', icon: '✂️', onClick: handleSplitPages },
        { id: 'merge', label: 'Merge', icon: '🔗', onClick: handleMerge },
        { id: 'rotate-cw', label: 'Rotate CW', icon: '🔄', onClick: () => handleRotate(90) },
        { id: 'rotate-ccw', label: 'Rotate CCW', icon: '🔃', onClick: () => handleRotate(270) },
        { id: 'watermark', label: 'Watermark', icon: '💧', onClick: handleWatermark },
        { id: 'optimize', label: 'Optimize', icon: '⚡', onClick: handleOptimize },
      ],
    },
  ]

  return (
    <div className="pdf-viewer">
      <EditorToolbar groups={toolbarGroups} title="PDF Viewer" />

      {error && (
        <div className="pdf-error" onClick={() => setError(null)}>
          ⚠️ {error}
        </div>
      )}

      <div className="pdf-content">
        {showText ? (
          <div className="pdf-text-view">
            <div className="pdf-text-header">
              <span>Extracted Text</span>
              <button onClick={() => setShowText(false)}>Back to PDF</button>
            </div>
            <pre className="pdf-text-content">{extractedText || 'No text extracted'}</pre>
          </div>
        ) : (
          <div className="pdf-page-view">
            {pdfState ? (
              <div className="pdf-page-container">
                <div className="pdf-page">
                  <div className="pdf-page-placeholder">
                    <div className="pdf-page-icon">📄</div>
                    <div className="pdf-page-label">Page {currentPage} of {pageCount}</div>
                    <div className="pdf-file-info">
                      {pdfState.title && <div>Title: {pdfState.title}</div>}
                      {pdfState.metadata && Object.entries(pdfState.metadata).map(([k, v]) => (
                        <div key={k} className="pdf-meta-item">{k}: {v}</div>
                      ))}
                    </div>
                  </div>
                </div>
              </div>
            ) : (
              <div className="pdf-empty">
                <div className="pdf-empty-icon">📄</div>
                <div>No PDF loaded</div>
              </div>
            )}
          </div>
        )}
      </div>

      <div className="pdf-status">
        {pdfState ? `${pdfState.file_path} — Page ${currentPage}/${pageCount}` : 'No file loaded'}
      </div>
    </div>
  )
}
