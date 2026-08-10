import { useState, useEffect } from 'react'
import { PdfService } from '../../services/wails-bridge'
import { EditorToolbar } from './EditorToolbar'
import './PdfViewer.css'

interface PdfViewerProps {
  tabId: string
  filePath?: string
}

export function PdfViewer({ tabId, filePath }: PdfViewerProps) {
  const [currentPage, setCurrentPage] = useState(1)
  const [totalPages, setTotalPages] = useState(1)
  const [zoom, setZoom] = useState(100)
  const [pageContent, setPageContent] = useState<string>('')

  // Open file via Go backend
  useEffect(() => {
    if (!filePath) return
    const init = async () => {
      try {
        const result = await PdfService.openFile(tabId, filePath)
        if (result?.page_count) setTotalPages(result.page_count as number)
        // Load first page preview
        const preview = await PdfService.getPagePreview(tabId, 0)
        if (preview) setPageContent(preview)
      } catch {
        // Fallback: show filename
      }
    }
    init()
    return () => { PdfService.close(tabId) }
  }, [tabId, filePath])

  // Load page preview when navigating
  useEffect(() => {
    if (!filePath) return
    PdfService.getPagePreview(tabId, currentPage - 1)
      .then((preview) => { if (preview) setPageContent(preview) })
      .catch(() => {})
  }, [tabId, filePath, currentPage])

  const toolbarGroups = [
    {
      id: 'nav',
      actions: [
        {
          id: 'prev', label: 'Previous Page', icon: '◀',
          onClick: () => setCurrentPage((p) => Math.max(1, p - 1)),
          disabled: currentPage <= 1,
        },
        {
          id: 'next', label: 'Next Page', icon: '▶',
          onClick: () => setCurrentPage((p) => Math.min(totalPages, p + 1)),
          disabled: currentPage >= totalPages,
        },
      ],
    },
    {
      id: 'zoom',
      actions: [
        { id: 'zoom-out', label: 'Zoom Out', icon: '−', onClick: () => setZoom((z) => Math.max(25, z - 25)) },
        { id: 'zoom-in', label: 'Zoom In', icon: '+', onClick: () => setZoom((z) => Math.min(400, z + 25)) },
        { id: 'zoom-fit', label: 'Fit to Width', icon: '⊟', onClick: () => setZoom(100) },
      ],
    },
    {
      id: 'actions',
      actions: [
        { id: 'print', label: 'Print', icon: '🖨', onClick: () => window.print() },
      ],
    },
  ]

  return (
    <div className="pdf-viewer">
      <EditorToolbar groups={toolbarGroups} title="PDF Viewer" />
      <div className="pdf-page-info">
        <span>Page {currentPage} of {totalPages}</span>
        <span>{zoom}%</span>
      </div>
      <div className="pdf-content">
        <div className="pdf-page" style={{ transform: `scale(${zoom / 100})` }}>
          {pageContent ? (
            <div className="pdf-page-rendered" dangerouslySetInnerHTML={{ __html: pageContent }} />
          ) : filePath ? (
            <div className="pdf-page-placeholder">
              <p>📄 {filePath.split(/[/\\]/).pop()}</p>
              <p className="pdf-page-hint">PDF rendering powered by Go backend</p>
            </div>
          ) : (
            <div className="pdf-page-placeholder">
              <p>No PDF loaded</p>
              <p className="pdf-page-hint">Open a PDF file to view it here</p>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
