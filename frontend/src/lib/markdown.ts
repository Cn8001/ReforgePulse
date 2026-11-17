// A small Markdown reader for the terms, the licence and release notes. It
// returns plain data that Markdown.vue renders as elements, never as HTML,
// so text from a release page cannot run anything in an elevated app.

export type Inline =
  | { t: 'text' | 'b' | 'i' | 'code'; text: string }
  | { t: 'link'; text: string; href: string }

export type Block =
  | { t: 'h'; level: 1 | 2 | 3; inline: Inline[] }
  | { t: 'p'; inline: Inline[] }
  | { t: 'ul' | 'ol'; items: Inline[][] }
  | { t: 'code'; text: string }

const bullet = /^\s*[-*+]\s+(.*)$/
const numbered = /^\s*\d+[.)]\s+(.*)$/
const heading = /^(#{1,6})\s+(.*?)\s*#*\s*$/

export function parse(src: string): Block[] {
  const lines = src.replace(/\r\n?/g, '\n').split('\n')
  const out: Block[] = []
  let para: string[] = []
  const flush = () => {
    if (para.length) out.push({ t: 'p', inline: inline(para.join(' ')) })
    para = []
  }
  for (let i = 0; i < lines.length; i++) {
    const line = lines[i]
    if (line.trim().startsWith('```')) {
      flush()
      const code: string[] = []
      while (++i < lines.length && !lines[i].trim().startsWith('```')) code.push(lines[i])
      out.push({ t: 'code', text: code.join('\n') })
      continue
    }
    const h = heading.exec(line)
    if (h) {
      flush()
      out.push({ t: 'h', level: Math.min(h[1].length, 3) as 1 | 2 | 3, inline: inline(h[2]) })
      continue
    }
    const kind = bullet.test(line) ? 'ul' : numbered.test(line) ? 'ol' : null
    if (kind) {
      flush()
      const re = kind === 'ul' ? bullet : numbered
      const items: string[] = []
      for (; i < lines.length; i++) {
        const m = re.exec(lines[i])
        if (m) items.push(m[1])
        else if (lines[i].trim() && /^\s{2,}/.test(lines[i]) && items.length) items[items.length - 1] += ' ' + lines[i].trim()
        else break
      }
      i--
      out.push({ t: kind, items: items.map(inline) })
      continue
    }
    if (!line.trim()) flush()
    else para.push(line.trim())
  }
  flush()
  return out
}

// One level of emphasis, code and links is all these texts use.
const token = /(\*\*[^*]+\*\*|__[^_]+__|`[^`]+`|\[[^\]]+\]\([^)\s]+\)|\*[^*\s][^*]*\*|_[^_\s][^_]*_)/

export function inline(text: string): Inline[] {
  const out: Inline[] = []
  for (const part of text.split(token)) {
    if (!part) continue
    if (/^(\*\*|__)/.test(part) && part.length > 4) out.push({ t: 'b', text: part.slice(2, -2) })
    else if (part.startsWith('`')) out.push({ t: 'code', text: part.slice(1, -1) })
    else if (part.startsWith('[')) {
      const m = /^\[([^\]]+)\]\(([^)\s]+)\)$/.exec(part)!
      out.push({ t: 'link', text: m[1], href: m[2] })
    } else if (/^[*_]/.test(part) && part.length > 2 && part.endsWith(part[0])) out.push({ t: 'i', text: part.slice(1, -1) })
    else out.push({ t: 'text', text: part })
  }
  return out
}
