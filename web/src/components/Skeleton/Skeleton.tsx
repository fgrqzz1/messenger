import styles from './Skeleton.module.css'

const CHAT_SKELETON_ROWS = [
  ['lineWide', 'lineMid'],
  ['lineMid', 'lineShort'],
  ['lineWide', 'lineMid'],
  ['lineShort', 'lineMid'],
  ['lineWide', 'lineShort'],
] as const

const MESSAGE_SKELETONS = [
  { own: false, size: 'bubbleMid' },
  { own: true, size: 'bubbleWide' },
  { own: false, size: 'bubbleShort' },
  { own: true, size: 'bubbleMid' },
] as const

export function ChatListSkeleton() {
  return (
    <ul className={styles.chatList} aria-hidden="true">
      {CHAT_SKELETON_ROWS.map((lines, index) => (
        <li key={index} className={styles.chatRow}>
          <div className={`${styles.shimmer} ${styles.avatar}`} />
          <div className={styles.chatLines}>
            <div className={`${styles.shimmer} ${styles.line} ${styles[lines[0]]}`} />
            <div className={`${styles.shimmer} ${styles.line} ${styles[lines[1]]}`} />
          </div>
        </li>
      ))}
    </ul>
  )
}

/** Skeleton rows for inside the messages `<ul>` (keeps listRef mounted). */
export function MessageListSkeletonItems() {
  return (
    <>
      {MESSAGE_SKELETONS.map((item, index) => (
        <li
          key={`skeleton-${index}`}
          className={`${styles.bubbleRow} ${item.own ? styles.bubbleRowOwn : styles.bubbleRowOther}`}
          aria-hidden="true"
        >
          <div className={`${styles.shimmer} ${styles.bubble} ${styles[item.size]}`} />
          <div className={`${styles.shimmer} ${styles.meta}`} />
        </li>
      ))}
    </>
  )
}
