import { useState, useCallback, useEffect } from 'react'
import { SlidesService } from '../../services/wails-bridge'
import { EditorToolbar } from './EditorToolbar'
import './SlidesEditor.css'

interface SlidesEditorProps {
  tabId: string
  filePath?: string
  onTitleChange: (title: string) => void
  onDirtyChange: (dirty: boolean) => void
}

interface Slide {
  id: string
  elements: SlideElement[]
}

interface SlideElement {
  id: string
  kind: 'text' | 'image' | 'shape'
  x: number
  y: number
  w: number
  h: number
  text?: string
  src?: string
}

let slideCounter = 0

function createBlankSlide(): Slide {
  slideCounter++
  return {
    id: `slide-${slideCounter}`,
    elements: [
      {
        id: `el-${slideCounter}-title`,
        kind: 'text',
        x: 60, y: 40, w: 840, h: 80,
        text: 'Click to add title',
      },
      {
        id: `el-${slideCounter}-body`,
        kind: 'text',
        x: 60, y: 160, w: 840, h: 360,
        text: 'Click to add content',
      },
    ],
  }
}

export function SlidesEditor({ tabId, filePath, onTitleChange, onDirtyChange }: SlidesEditorProps) {
  const [slides, setSlides] = useState<Slide[]>([createBlankSlide()])
  const [currentSlide, setCurrentSlide] = useState(0)
  const [selectedElement, setSelectedElement] = useState<string | null>(null)

  // Open file or create blank via Go backend
  useEffect(() => {
    const init = async () => {
      try {
        let result: Record<string, unknown>
        if (filePath) {
          result = await SlidesService.openFile(tabId, filePath)
        } else {
          result = await SlidesService.newBlank(tabId)
        }
        if (result?.title) onTitleChange(result.title as string)
        else onTitleChange(filePath?.split(/[/\\]/).pop() || 'Untitled Presentation')

        // Load slides from result if available
        if (result?.slides && Array.isArray(result.slides)) {
          setSlides(result.slides as Slide[])
        }
      } catch {
        onTitleChange(filePath?.split(/[/\\]/).pop() || 'Untitled Presentation')
      }
    }
    init()
    return () => { SlidesService.close(tabId) }
  }, [filePath, tabId, onTitleChange])

  const addSlide = useCallback(async () => {
    const newSlide = createBlankSlide()
    setSlides((prev) => [...prev, newSlide])
    setCurrentSlide(slides.length)
    onDirtyChange(true)
    try {
      await SlidesService.addSlide(tabId, currentSlide, 'blank')
    } catch { /* fallback: local only */ }
  }, [slides.length, currentSlide, tabId, onDirtyChange])

  const deleteSlide = useCallback(async () => {
    if (slides.length <= 1) return
    const idx = currentSlide
    setSlides((prev) => prev.filter((_, i) => i !== idx))
    setCurrentSlide(Math.max(0, idx - 1))
    onDirtyChange(true)
    try {
      await SlidesService.deleteSlide(tabId, idx)
    } catch { /* fallback: local only */ }
  }, [slides.length, currentSlide, tabId, onDirtyChange])

  const handleSave = useCallback(async () => {
    try {
      const result = await SlidesService.save(tabId)
      if (result?.success) onDirtyChange(false)
    } catch (err) {
      console.error('Save failed:', err)
    }
  }, [tabId, onDirtyChange])

  const slide = slides[currentSlide]

  const toolbarGroups = [
    {
      id: 'file',
      actions: [{ id: 'save', label: 'Save', icon: '💾', onClick: handleSave }],
    },
    {
      id: 'slides',
      actions: [
        { id: 'add', label: 'Add Slide', icon: '➕', onClick: addSlide },
        { id: 'delete', label: 'Delete Slide', icon: '🗑', onClick: deleteSlide, disabled: slides.length <= 1 },
      ],
    },
    {
      id: 'insert',
      actions: [
        {
          id: 'text',
          label: 'Add Text',
          icon: 'T',
          onClick: async () => {
            const elem: SlideElement = {
              id: `el-${Date.now()}`, kind: 'text',
              x: 100, y: 200, w: 300, h: 100, text: 'New text box',
            }
            setSlides((prev) => prev.map((s, i) =>
              i === currentSlide ? { ...s, elements: [...s.elements, elem] } : s,
            ))
            onDirtyChange(true)
            try { await SlidesService.addElement(tabId, currentSlide, elem) } catch {}
          },
        },
      ],
    },
  ]

  return (
    <div className="slides-editor">
      <EditorToolbar groups={toolbarGroups} title="Presentation" />
      <div className="slides-body">
        <div className="slides-sidebar">
          {slides.map((s, i) => (
            <div
              key={s.id}
              className={`slides-thumb ${i === currentSlide ? 'active' : ''}`}
              onClick={() => { setCurrentSlide(i); setSelectedElement(null) }}
            >
              <span className="slides-thumb-num">{i + 1}</span>
              <div className="slides-thumb-preview">
                {s.elements.filter((el) => el.kind === 'text').map((el) => (
                  <div key={el.id} className="slides-thumb-text">
                    {el.text?.substring(0, 20)}
                  </div>
                ))}
              </div>
            </div>
          ))}
        </div>
        <div className="slides-canvas-area">
          <div
            className="slides-canvas"
            onClick={() => setSelectedElement(null)}
          >
            {slide?.elements.map((el) => (
              <div
                key={el.id}
                className={`slides-element ${selectedElement === el.id ? 'selected' : ''}`}
                style={{
                  left: `${(el.x / 960) * 100}%`,
                  top: `${(el.y / 540) * 100}%`,
                  width: `${(el.w / 960) * 100}%`,
                  height: `${(el.h / 540) * 100}%`,
                }}
                onClick={(e) => {
                  e.stopPropagation()
                  setSelectedElement(el.id)
                }}
              >
                {el.kind === 'text' && (
                  <div
                    className="slides-text-content"
                    contentEditable
                    suppressContentEditableWarning
                    onInput={() => onDirtyChange(true)}
                  >
                    {el.text}
                  </div>
                )}
                {el.kind === 'image' && el.src && (
                  <img src={el.src} alt="" className="slides-image-content" />
                )}
              </div>
            ))}
          </div>
        </div>
      </div>
      <div className="slides-footer">
        <span>Slide {currentSlide + 1} of {slides.length}</span>
      </div>
    </div>
  )
}
