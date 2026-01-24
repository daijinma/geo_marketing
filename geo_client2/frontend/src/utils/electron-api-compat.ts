// Wails API types - compatible with electronAPI
import { wailsAPI } from './wails-api';

// Make wailsAPI available as window.electronAPI for compatibility
declare global {
  interface Window {
    electronAPI: typeof wailsAPI;
  }
}

// Set window.electronAPI when module loads
if (typeof window !== 'undefined') {
  window.electronAPI = wailsAPI;
}

export {};
