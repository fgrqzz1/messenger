import { Sidebar } from './components/Sidebar/Sidebar'
import { ConnectedChatWindow } from './components/ChatWindow/ChatWindow'
import { ActiveChatProvider } from './context/ActiveChatContext'
import { AuthProvider, useAuth } from './context/AuthContext'
import { SidebarProvider, useSidebar } from './context/SidebarContext'
import { ChatsProvider, useChats } from './hooks/useChats'
import { WebSocketProvider } from './hooks/useWebSocket'
import { Login } from './screens/Login/Login'
import styles from './App.module.css'

function MessengerLayout() {
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
        <Sidebar />
        <main className={styles.main}>
          <ConnectedChatWindow />
        </main>
      </div>
    </WebSocketProvider>
  )
}

function AppContent() {
  const { isAuthenticated } = useAuth()

  if (!isAuthenticated) {
    return <Login />
  }

  return (
    <ActiveChatProvider>
      <ChatsProvider>
        <SidebarProvider>
          <MessengerLayout />
        </SidebarProvider>
      </ChatsProvider>
    </ActiveChatProvider>
  )
}

export default function App() {
  return (
    <AuthProvider>
      <AppContent />
    </AuthProvider>
  )
}
