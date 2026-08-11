import { useState, useCallback, useRef, useEffect } from 'react'
import { AIService } from '../services/wails-bridge'
import type { AIMessage } from '../services/wails-bridge'
import './AiPanel.css'

interface AiPanelProps {
  tabId: string
  tabKind: string
  onClose: () => void
}

interface ChatMessage {
  role: 'user' | 'assistant'
  text: string
  timestamp: number
}

export function AiPanel({ tabId, tabKind, onClose }: AiPanelProps) {
  const [messages, setMessages] = useState<ChatMessage[]>([])
  const [input, setInput] = useState('')
  const [isLoading, setIsLoading] = useState(false)
  const messagesEndRef = useRef<HTMLDivElement>(null)
  const inputRef = useRef<HTMLTextAreaElement>(null)
  // Keep conversation history for multi-turn chat
  const historyRef = useRef<AIMessage[]>([])

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages])

  const sendMessage = useCallback(async () => {
    const text = input.trim()
    if (!text || isLoading) return

    const userMsg: ChatMessage = { role: 'user', text, timestamp: Date.now() }
    setMessages((prev) => [...prev, userMsg])
    setInput('')
    setIsLoading(true)

    // Build message history for the AI
    historyRef.current.push({ role: 'user', text })

    try {
      const result = await AIService.chat({
        messages: historyRef.current,
        system: `You are an AI assistant embedded in an office suite. The user is working on a ${tabKind} document. Help them with their request.`,
        max_tokens: 2048,
      })
      const responseText = result?.text || `Processed your request for ${tabKind}.`
      historyRef.current.push({ role: 'assistant', text: responseText })
      setMessages((prev) => [
        ...prev,
        { role: 'assistant', text: responseText, timestamp: Date.now() },
      ])
    } catch (err) {
      const errText = `Error: ${err instanceof Error ? err.message : err}`
      setMessages((prev) => [
        ...prev,
        { role: 'assistant', text: errText, timestamp: Date.now() },
      ])
    } finally {
      setIsLoading(false)
    }
  }, [input, isLoading, tabKind])

  const handleClear = useCallback(() => {
    setMessages([])
    historyRef.current = []
  }, [])

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (e.key === 'Enter' && !e.shiftKey) {
        e.preventDefault()
        sendMessage()
      }
    },
    [sendMessage],
  )

  const suggestions = getSuggestions(tabKind)

  return (
    <div className="ai-panel">
      <div className="ai-panel-header">
        <span className="ai-panel-title">✨ AI Assistant</span>
        <div className="ai-panel-actions">
          <button className="ai-panel-clear" onClick={handleClear} title="Clear chat">🗑</button>
          <button className="ai-panel-close" onClick={onClose}>×</button>
        </div>
      </div>

      <div className="ai-panel-messages">
        {messages.length === 0 && (
          <div className="ai-panel-empty">
            <p className="ai-panel-greeting">How can I help with your {tabKind}?</p>
            <div className="ai-panel-suggestions">
              {suggestions.map((s, i) => (
                <button
                  key={i}
                  className="ai-panel-suggestion"
                  onClick={() => {
                    setInput(s)
                    inputRef.current?.focus()
                  }}
                >
                  {s}
                </button>
              ))}
            </div>
          </div>
        )}
        {messages.map((msg, i) => (
          <div key={i} className={`ai-msg ai-msg-${msg.role}`}>
            <div className="ai-msg-avatar">{msg.role === 'user' ? '👤' : '✨'}</div>
            <div className="ai-msg-content">
              <p>{msg.text}</p>
            </div>
          </div>
        ))}
        {isLoading && (
          <div className="ai-msg ai-msg-assistant">
            <div className="ai-msg-avatar">✨</div>
            <div className="ai-msg-content ai-typing">
              <span /><span /><span />
            </div>
          </div>
        )}
        <div ref={messagesEndRef} />
      </div>

      <div className="ai-panel-input">
        <textarea
          ref={inputRef}
          className="ai-input-textarea"
          value={input}
          onChange={(e) => setInput(e.target.value)}
          onKeyDown={handleKeyDown}
          placeholder="Ask AI to help..."
          rows={1}
        />
        <button
          className="ai-input-send"
          onClick={sendMessage}
          disabled={!input.trim() || isLoading}
        >
          ➤
        </button>
      </div>
    </div>
  )
}

function getSuggestions(kind: string): string[] {
  switch (kind) {
    case 'docs':
      return ['Help me write an introduction', 'Summarize this document', 'Fix grammar and spelling', 'Make this more concise']
    case 'sheets':
      return ['Create a formula for column totals', 'Generate sample data', 'Explain this formula', 'Create a chart from this data']
    case 'slides':
      return ['Generate slide content for a topic', 'Improve slide layout', 'Add speaker notes', 'Create an outline for a presentation']
    case 'pdf':
      return ['Summarize this PDF', 'Extract key points', 'Find specific information']
    case 'markdown':
      return ['Format this as a README', 'Add a table of contents', 'Convert to a different style', 'Generate documentation from code']
    default:
      return ['How can I help?']
  }
}
