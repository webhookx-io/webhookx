import { create } from 'zustand'
import { persist } from 'zustand/middleware'

type Theme = 'dark' | 'light'

interface AppState {
  theme: Theme
  activeWorkspaceName: string
  sidebarOpen: boolean
  setTheme: (theme: Theme) => void
  setActiveWorkspaceName: (workspaceName: string) => void
  setSidebarOpen: (open: boolean) => void
}

export const useAppStore = create<AppState>()(
  persist(
    (set) => ({
      theme: 'dark',
      activeWorkspaceName: 'default',
      sidebarOpen: false,
      setTheme: (theme) => set({ theme }),
      setActiveWorkspaceName: (activeWorkspaceName) => set({ activeWorkspaceName }),
      setSidebarOpen: (sidebarOpen) => set({ sidebarOpen }),
    }),
    {
      name: 'webhookx',
      partialize: (state) => ({
        theme: state.theme,
        activeWorkspaceName: state.activeWorkspaceName,
      }),
    },
  ),
)
