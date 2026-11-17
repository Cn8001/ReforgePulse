import { createApp } from 'vue'
import App from './App.vue'
import { followAccent } from './lib/theme'
import { Frontend } from '../bindings/github.com/cn8001/reforgepulse/internal/app/logservice'
import './style.css'

// The webview's console is out of reach in a release, so errors go to the
// app log, where a crash report can be read.
const report = (e: unknown, where: string) =>
  Frontend(e instanceof Error ? `${e.message}\n${e.stack ?? ''}` : String(e), where).catch(() => {})
window.addEventListener('error', (e) => report(e.error ?? e.message, 'window'))
window.addEventListener('unhandledrejection', (e) => report(e.reason, 'promise'))

followAccent()
const app = createApp(App)
app.config.errorHandler = (e, _vm, info) => report(e, `vue: ${info}`)
app.mount('#app')
