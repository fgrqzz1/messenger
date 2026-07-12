import { Search, X } from 'lucide-react'
import { useEffect, useState } from 'react'
import { searchMessages } from '../../api/search'
import type { Message } from '../../types/domain'
import { formatMessageTime } from '../../utils/formatMessageTime'
import styles from './SearchPanel.module.css'

const DEBOUNCE_MS = 300

type SearchPanelProps = {
  chatId: number
  memberNames: Record<number, string>
  onClose: () => void
  onSelectMessage: (messageId: number) => void | Promise<void>
}

export function SearchPanel({
  chatId,
  memberNames,
  onClose,
  onSelectMessage,
}: SearchPanelProps) {
  const [query, setQuery] = useState('')
  const [results, setResults] = useState<Message[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [navigatingId, setNavigatingId] = useState<number | null>(null)

  useEffect(() => {
    const trimmed = query.trim()
    if (!trimmed) {
      setResults([])
      setError(null)
      setLoading(false)
      return
    }

    let cancelled = false
    setLoading(true)
    setError(null)

    const timer = window.setTimeout(() => {
      searchMessages(chatId, trimmed)
        .then((messages) => {
          if (!cancelled) {
            setResults(messages)
          }
        })
        .catch((err: unknown) => {
          if (!cancelled) {
            setResults([])
            setError(err instanceof Error ? err.message : 'Ошибка поиска')
          }
        })
        .finally(() => {
          if (!cancelled) {
            setLoading(false)
          }
        })
    }, DEBOUNCE_MS)

    return () => {
      cancelled = true
      window.clearTimeout(timer)
    }
  }, [chatId, query])

  const trimmedQuery = query.trim()
  const showResults = trimmedQuery.length > 0

  const handleSelect = async (messageId: number) => {
    setNavigatingId(messageId)
    try {
      await onSelectMessage(messageId)
      onClose()
    } finally {
      setNavigatingId(null)
    }
  }

  return (
    <section className={styles.searchPanel} aria-label="Поиск по чату">
      <div className={styles.searchBar}>
        <div className={styles.inputWrap}>
          <Search className={styles.searchIcon} size={16} strokeWidth={1.75} aria-hidden />
          <input
            className={styles.searchInput}
            type="search"
            placeholder="Поиск в чате…"
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            autoFocus
          />
        </div>
        <button
          type="button"
          className={styles.closeBtn}
          aria-label="Закрыть поиск"
          onClick={onClose}
        >
          <X size={16} strokeWidth={1.75} aria-hidden />
        </button>
      </div>

      {error && <div className={styles.errorBanner}>{error}</div>}

      {showResults && (
        <ul className={styles.results}>
          {loading && <li className={styles.stateMessage}>Поиск…</li>}
          {!loading && results.length === 0 && !error && (
            <li className={styles.stateMessage}>Ничего не найдено</li>
          )}
          {!loading &&
            results.map((message) => {
              const senderName = memberNames[message.sender_id] ?? `id:${message.sender_id}`

              return (
                <li key={message.id}>
                  <button
                    type="button"
                    className={styles.resultItem}
                    disabled={navigatingId !== null}
                    onClick={() => void handleSelect(message.id)}
                  >
                    <span className={styles.resultBody}>{message.body}</span>
                    <span className={styles.resultMeta}>
                      {senderName} · {formatMessageTime(message.created_at)}
                    </span>
                  </button>
                </li>
              )
            })}
        </ul>
      )}
    </section>
  )
}
