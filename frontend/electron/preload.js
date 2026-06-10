// Runs in the isolated preload world but shares the DOM with the page —
// strips browser-feel keyboard/drag behaviors from the bundled web app
// without forking it (the same bundle runs on the website unchanged).

// Tab must not crawl focus across buttons and links like in a browser.
// Field-to-field traversal inside forms (login, register, settings) still
// works: there the keydown target is the input itself.
//
// Deliberate product decision (not an oversight): review suggested allowing
// Tab from any focusable element for keyboard-only users, but that re-enables
// exactly the browser-style focus crawl this exists to remove. Revisit if
// desktop keyboard navigation ever becomes a requirement.
window.addEventListener(
  'keydown',
  (event) => {
    if (event.key !== 'Tab') return
    const el = event.target
    const editable =
      el instanceof HTMLElement &&
      (el.isContentEditable || el.tagName === 'INPUT' || el.tagName === 'TEXTAREA' || el.tagName === 'SELECT')
    if (!editable) event.preventDefault()
  },
  true,
)

// A file dropped anywhere in the window must not make Chromium "navigate"
// to it, replacing the app. The web app has no drop targets (attachments go
// through the file picker), so swallowing the default is safe globally.
window.addEventListener('dragover', (event) => event.preventDefault())
window.addEventListener('drop', (event) => event.preventDefault())
