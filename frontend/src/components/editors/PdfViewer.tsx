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
        if ((result as any)?.error) {
          setError((result as any).error)
          return
        }
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

  const handleExtractPages = useCallback(async () => {
    try {
      const outPath = await WailsRuntime.saveFileDialog()
      if (!outPath) return
      const result = await PdfService.extractPages(tabId, [currentPage], outPath)
      if ((result as any)?.error) setError((result as any).error)
    } catch (err) {
      setError(`Extract pages failed: ${err}`)
    }
  }, [tabId, currentPage])

  const handleRotate = useCallback(async (degrees: number) => {
    try {
      const result = await PdfService.rotatePage(tabId, currentPage, degrees)
      if ((result as any)?.success) {
        const state = await PdfService.getState(tabId)
        if (state) setPdfState(state)
      }
    } catch (err) {
      setError(`Rotate failed: ${err}`)
    }
  }, [tabId, currentPage])

  const handleMerge = useCallback(async () => {
    try {
      const path = await WailsRuntime.openFileDialog()
      if (!path) return
      const outputPath = await WailsRuntime.saveFileDialog()
      if (!outputPath) return
      const result = await PdfService.mergeFiles([filePath || '', path], outputPath)
      if ((result as any)?.error) setError((result as any).error)
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
      if ((result as any)?.error) setError((result as any).error)
    } catch (err) {
      setError(`Watermark failed: ${err}`)
    }
  }, [tabId])

  const pageCount = pdfState?.page_count || 0

  const toolbarGroups = [
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
        { id: 'extract-pages', label: 'Extract Pages', icon: '📑', onClick: handleExtractPages },
        { id: 'merge', label: 'Merge', icon: '🔗', onClick: handleMerge },
        { id: 'rotate-cw', label: 'Rotate CW', icon: '🔄', onClick: () => handleRotate(90) },
        { id: 'rotate-ccw', label: 'Rotate CCW', icon: '🔃', onClick: () => handleRotate(270) },
        { id: 'watermark', label: 'Watermark', icon: '💧', onClick: handleWatermark },
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
                      {pdfState.meta && (
                        <div className="pdf-meta-item">
                          Author: {(pdfState.meta as any)?.author || 'Unknown'}
                        </div>
                      )}
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
