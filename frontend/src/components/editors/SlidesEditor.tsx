import { useState, useCallback, useEffect } from 'react'
import { SlidesService } from '../../services/wails-bridge'
import type { SlideInfo, SlideElement } from '../../services/wails-bridge'
import { EditorToolbar } from './EditorToolbar'
import './SlidesEditor.css'

interface SlidesEditorProps {
  tabId: string
  filePath?: string
  onTitleChange: (title: string) => void
  onDirtyChange: (dirty: boolean) => void
}

export function SlidesEditor({ tabId, filePath, onTitleChange, onDirtyChange }: SlidesEditorProps) {
  const [slides, setSlides] = useState<SlideInfo[]>([])
  const [currentSlide, setCurrentSlide] = useState(0)
  const [selectedElement, setSelectedElement] = useState<string | null>(null)
  const [editText, setEditText] = useState('')

  useEffect(() => {
    const init = async () => {
      try {
        let result
        if (filePath) {
          result = await SlidesService.openFile(tabId, filePath)
        } else {
          result = await SlidesService.newBlank(tabId)
        }
        if (result?.slides) setSlides(result.slides)
        onTitleChange(result?.title || filePath?.split(/[/\\]/).pop() || 'Untitled Presentation')
      } catch {
        onTitleChange(filePath?.split(/[/\\]/).pop() || 'Untitled Presentation')
        // Start with one blank slide
        setSlides([{ index: 0, title: 'Slide 1', elements: [] }])
      }
    }
    init()
    return () => { SlidesService.close(tabId) }
  }, [filePath, tabId, onTitleChange])

  const refreshSlides = useCallback(async () => {
    try {
      const s = await SlidesService.getSlides(tabId)
      if (s?.length) setSlides(s)
    } catch {}
  }, [tabId])

  const handleAddSlide = useCallback(async () => {
    try {
      const newSlide = await SlidesService.addSlide(tabId)
      if (newSlide) {
        await refreshSlides()
        setCurrentSlide(slides.length)
        onDirtyChange(true)
      }
    } catch {
      // Fallback: add locally
      setSlides(prev => [...prev, {
        index: prev.length,
        title: `Slide ${prev.length + 1}`,
        elements: [],
      }])
      setCurrentSlide(slides.length)
    }
  }, [tabId, slides.length, onDirtyChange, refreshSlides])

  const handleDeleteSlide = useCallback(async () => {
    if (slides.length <= 1) return
    try {
      await SlidesService.deleteSlide(tabId, currentSlide)
      await refreshSlides()
      setCurrentSlide(Math.max(0, currentSlide - 1))
      onDirtyChange(true)
    } catch {
      setSlides(prev => prev.filter((_, i) => i !== currentSlide))
      setCurrentSlide(Math.max(0, currentSlide - 1))
    }
  }, [tabId, currentSlide, slides.length, onDirtyChange, refreshSlides])

  const handleDuplicateSlide = useCallback(async () => {
    try {
      await SlidesService.duplicateSlide(tabId, currentSlide)
      await refreshSlides()
      setCurrentSlide(currentSlide + 1)
      onDirtyChange(true)
    } catch {}
  }, [tabId, currentSlide, onDirtyChange, refreshSlides])

  const handleMoveSlide = useCallback(async (direction: 'up' | 'down') => {
    const target = direction === 'up' ? currentSlide - 1 : currentSlide + 1
    if (target < 0 || target >= slides.length) return
    try {
      await SlidesService.moveSlide(tabId, currentSlide, target)
      await refreshSlides()
      setCurrentSlide(target)
      onDirtyChange(true)
    } catch {}
  }, [tabId, currentSlide, slides.length, onDirtyChange, refreshSlides])

  const handleSave = useCallback(async () => {
    try {
      const result = await SlidesService.save(tabId)
      if (result?.success) onDirtyChange(false)
    } catch (err) {
      console.error('Save failed:', err)
    }
  }, [tabId, onDirtyChange])

  const handleUndo = useCallback(async () => {
    await SlidesService.undo(tabId)
    await refreshSlides()
  }, [tabId, refreshSlides])

  const handleRedo = useCallback(async () => {
    await SlidesService.redo(tabId)
    await refreshSlides()
  }, [tabId, refreshSlides])

  const handleElementClick = useCallback((elem: SlideElement) => {
    setSelectedElement(elem.id)
    setEditText(elem.text || '')
  }, [])

  const handleElementUpdate = useCallback(async () => {
    if (!selectedElement) return
    try {
      await SlidesService.updateElement(tabId, currentSlide, selectedElement, editText)
      await refreshSlides()
      onDirtyChange(true)
    } catch {}
    setSelectedElement(null)
    setEditText('')
  }, [tabId, currentSlide, selectedElement, editText, onDirtyChange, refreshSlides])

  const currentSlideData = slides[currentSlide]

  const toolbarGroups = [
    {
      id: 'file',
      actions: [{ id: 'save', label: 'Save', icon: '💾', onClick: handleSave }],
    },
    {
      id: 'slides',
      actions: [
        { id: 'add', label: 'Add Slide', icon: '➕', onClick: handleAddSlide },
        { id: 'duplicate', label: 'Duplicate', icon: '📋', onClick: handleDuplicateSlide },
        { id: 'delete', label: 'Delete', icon: '🗑️', onClick: handleDeleteSlide, disabled: slides.length <= 1 },
        { id: 'move-up', label: '↑', icon: '↑', onClick: () => handleMoveSlide('up'), disabled: currentSlide === 0 },
        { id: 'move-down', label: '↓', icon: '↓', onClick: () => handleMoveSlide('down'), disabled: currentSlide >= slides.length - 1 },
      ],
    },
    {
      id: 'edit',
      actions: [
        { id: 'undo', label: 'Undo', icon: '↩', onClick: handleUndo },
        { id: 'redo', label: 'Redo', icon: '↪', onClick: handleRedo },
      ],
    },
  ]

  return (
    <div className="slides-editor">
      <EditorToolbar groups={toolbarGroups} title="Presentation" />
      <div className="slides-workspace">
        <div className="slides-sidebar">
          {slides.map((slide, i) => (
            <div
              key={i}
              className={`slides-thumb ${i === currentSlide ? 'active' : ''}`}
              onClick={() => { setCurrentSlide(i); setSelectedElement(null) }}
            >
              <div className="slides-thumb-number">{i + 1}</div>
              <div className="slides-thumb-preview">
                <div className="slides-thumb-title">{slide.title || `Slide ${i + 1}`}</div>
                {slide.elements?.slice(0, 2).map((el, j) => (
                  <div key={j} className="slides-thumb-element">{el.text?.substring(0, 30)}</div>
                ))}
              </div>
            </div>
          ))}
        </div>
        <div className="slides-canvas-area">
          <div className="slides-canvas">
            <div className="slides-slide">
              {currentSlideData?.title && (
                <h2 className="slides-slide-title">{currentSlideData.title}</h2>
              )}
              {currentSlideData?.elements?.map((elem) => (
                <div
                  key={elem.id}
                  className={`slides-element ${selectedElement === elem.id ? 'selected' : ''}`}
                  style={{
                    position: 'absolute',
                    left: `${(elem.x / 9144000) * 100}%`,
                    top: `${(elem.y / 6858000) * 100}%`,
                    width: elem.w ? `${(elem.w / 9144000) * 100}%` : 'auto',
                  }}
                  onClick={() => handleElementClick(elem)}
                  onDoubleClick={() => handleElementClick(elem)}
                >
                  {selectedElement === elem.id ? (
                    <textarea
                      className="slides-element-edit"
                      value={editText}
                      onChange={(e) => setEditText(e.target.value)}
                      onBlur={handleElementUpdate}
                      onKeyDown={(e) => {
                        if (e.key === 'Escape') { setSelectedElement(null) }
                        if (e.key === 'Enter' && (e.ctrlKey || e.metaKey)) { handleElementUpdate() }
                      }}
                      autoFocus
                    />
                  ) : (
                    <div className="slides-element-text">{elem.text}</div>
                  )}
                </div>
              ))}
              {(!currentSlideData?.elements || currentSlideData.elements.length === 0) && (
                <div className="slides-placeholder">
                  <div className="slides-placeholder-text">Click to add content</div>
                </div>
              )}
            </div>
          </div>
          <div className="slides-status">
            Slide {currentSlide + 1} of {slides.length}
          </div>
        </div>
      </div>
    </div>
  )
}
