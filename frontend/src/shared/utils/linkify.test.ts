import { isValidElement, type ReactNode } from 'react'
import { describe, expect, it } from 'vitest'

import { linkify } from './linkify'

type AnchorProps = { href: string; children: ReactNode }

function hrefs(nodes: ReactNode[]): string[] {
  const out: string[] = []
  for (const n of nodes) {
    if (isValidElement<AnchorProps>(n)) out.push(n.props.href)
  }
  return out
}

function flatText(nodes: ReactNode[]): string {
  return nodes
    .map((n) => (isValidElement<AnchorProps>(n) ? String(n.props.children) : String(n)))
    .join('')
}

describe('linkify', () => {
  it('leaves plain text untouched', () => {
    const nodes = linkify('just some text')
    expect(hrefs(nodes)).toEqual([])
    expect(flatText(nodes)).toBe('just some text')
  })

  it('links a single bare URL', () => {
    const nodes = linkify('https://example.com')
    expect(hrefs(nodes)).toEqual(['https://example.com'])
  })

  it('links a URL in the middle of a sentence and preserves surrounding text', () => {
    const nodes = linkify('see https://example.com/page now')
    expect(hrefs(nodes)).toEqual(['https://example.com/page'])
    expect(flatText(nodes)).toBe('see https://example.com/page now')
  })

  it('trims trailing sentence punctuation out of the link', () => {
    const nodes = linkify('go to https://example.com.')
    expect(hrefs(nodes)).toEqual(['https://example.com'])
    expect(flatText(nodes)).toBe('go to https://example.com.')
  })

  it('keeps a dangling close-paren as text', () => {
    const nodes = linkify('(https://example.com)')
    expect(hrefs(nodes)).toEqual(['https://example.com'])
    expect(flatText(nodes)).toBe('(https://example.com)')
  })

  it('links multiple URLs', () => {
    const nodes = linkify('a http://a.test b https://b.test c')
    expect(hrefs(nodes)).toEqual(['http://a.test', 'https://b.test'])
    expect(flatText(nodes)).toBe('a http://a.test b https://b.test c')
  })

  it('does not link non-http schemes', () => {
    const nodes = linkify('email me at me@example.com or ftp://x')
    expect(hrefs(nodes)).toEqual([])
  })
})
