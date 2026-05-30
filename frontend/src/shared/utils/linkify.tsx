import { type ReactNode } from 'react'

import { isNative } from '../platform'

// Match http(s) URLs. Deliberately greedy on the path but stops at whitespace,
// angle brackets and quotes so it never swallows surrounding markup.
const URL_REGEX = /(https?:\/\/[^\s<>"]+)/g

// Sentence punctuation that is far more likely to terminate a sentence than to
// be part of the URL. Trimmed off the link and pushed back into the text run.
const TRAILING_PUNCT = /[.,!?;:]+$/

function openExternal(url: string): void {
  // Capacitor WebView swallows `target="_blank"` navigations; open the system
  // browser explicitly. No @capacitor/browser dependency, so window.open it is.
  window.open(url, '_blank', 'noopener,noreferrer')
}

/**
 * Split plain text into an array of text runs and `<a>` elements for every URL.
 * Returns React nodes (never HTML strings) so React escapes the text runs — no
 * XSS surface even though message bodies are user-controlled.
 */
export function linkify(text: string): ReactNode[] {
  const nodes: ReactNode[] = []
  let lastIndex = 0
  let key = 0
  URL_REGEX.lastIndex = 0

  let match: RegExpExecArray | null
  while ((match = URL_REGEX.exec(text)) !== null) {
    const raw = match[0]
    const start = match.index

    let url = raw
    let trailing = ''
    const punct = TRAILING_PUNCT.exec(url)
    if (punct) {
      trailing = punct[0]
      url = url.slice(0, url.length - punct[0].length)
    }
    // A lone trailing ")" with no matching "(" is almost always "(see http://x)".
    if (url.endsWith(')') && !url.includes('(')) {
      trailing = ')' + trailing
      url = url.slice(0, -1)
    }

    if (start > lastIndex) {
      nodes.push(text.slice(lastIndex, start))
    }

    nodes.push(
      <a
        key={`lnk-${key++}`}
        href={url}
        target="_blank"
        rel="noopener noreferrer"
        onClick={(e) => {
          if (isNative()) {
            e.preventDefault()
            openExternal(url)
          }
        }}
      >
        {url}
      </a>,
    )

    if (trailing) nodes.push(trailing)
    lastIndex = start + raw.length
  }

  if (lastIndex < text.length) {
    nodes.push(text.slice(lastIndex))
  }

  return nodes
}
