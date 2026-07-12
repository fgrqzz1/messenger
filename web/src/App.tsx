import { useState } from 'react'
import { Sidebar } from './components/Sidebar/Sidebar'
import { ConnectedChatWindow } from './components/ChatWindow/ChatWindow'
import { ActiveChatProvider } from './context/ActiveChatContext'
import { AuthProvider, useAuth } from './context/AuthContext'
import { SidebarProvider, useSidebar } from './context/SidebarContext'
import { ChatsProvider, useChats } from './hooks/useChats'
import { WebSocketProvider } from './hooks/useWebSocket'
import { Login } from './screens/Login/Login'
import { Profile } from './screens/Profile/Profile'
import styles from './App.module.css'

type AppScreen = 'chats' | 'profile'

function BackgroundBlobs() {
  return (
    <div className={styles.blobs} aria-hidden="true">
      <div className={`${styles.blob} ${styles.blobA}`} />
      <div className={`${styles.blob} ${styles.blobB}`} />
      <div className={`${styles.blob} ${styles.blobC}`} />
    </div>
  )
}

function MessengerLayout({ onOpenProfile }: { onOpenProfile: () => void }) {
  const { updateChatPreview, advanceMyReadCursor, ensureChatFromMessage } = useChats()
  const { isNarrow, sidebarOpen, closeSidebar } = useSidebar()

  return (
    <WebSocketProvider
      updateChatPreview={updateChatPreview}
      advanceMyReadCursor={advanceMyReadCursor}
      ensureChatFromMessage={ensureChatFromMessage}
    >
      <div className={styles.layout}>
        {isNarrow && sidebarOpen && (
          <button
            type="button"
            className={styles.backdrop}
            aria-label="Закрыть список чатов"
            onClick={closeSidebar}
          />
        )}
        <Sidebar onOpenProfile={onOpenProfile} />
        <main className={styles.main}>
          <ConnectedChatWindow />
        </main>
      </div>
    </WebSocketProvider>
  )
}

function AuthenticatedApp() {
  const [screen, setScreen] = useState<AppScreen>('chats')

  if (screen === 'profile') {
    return <Profile onBack={() => setScreen('chats')} />
  }

  return (
    <ActiveChatProvider>
      <ChatsProvider>
        <SidebarProvider>
          <MessengerLayout onOpenProfile={() => setScreen('profile')} />
        </SidebarProvider>
      </ChatsProvider>
    </ActiveChatProvider>
  )
}

function AppContent() {
  const { isAuthenticated } = useAuth()

  if (!isAuthenticated) {
    return <Login />
  }

  return <AuthenticatedApp />
}

export default function App() {
  return (
    <AuthProvider>
      <div className={styles.appShell}>
        <BackgroundBlobs />
        <div className={styles.appContent}>
          <AppContent />
        </div>
      </div>
    </AuthProvider>
  )
}
