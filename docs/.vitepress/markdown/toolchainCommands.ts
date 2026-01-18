/**
 * VitePress markdown plugin that transforms <ToolchainCommands> and <StandardCommands>
 * tags into markdown tables at build time.
 *
 * Source of truth: internal/cli/toolchains_template.json
 */
import type MarkdownIt from 'markdown-it'
import { readFileSync } from 'fs'
import { resolve } from 'path'

// Load toolchains data from the single source of truth
// Use process.cwd() as VitePress runs from the docs directory
const toolchainsPath = resolve(process.cwd(), '../internal/cli/toolchains_template.json')
const toolchainsData = JSON.parse(readFileSync(toolchainsPath, 'utf-8'))

// Derive command order from the source of truth
const specCommandOrder = Object.keys(toolchainsData.commands)

const guideCommandOrder = [
  'build', 'build:release', 'test', 'check', 'check:fix',
  'restore', 'bench', 'pack', 'doc', 'demo'
]

// Commands that mutate state
const mutatingCommands = ['clean', 'restore', 'build', 'build:release', 'check:fix', 'pack', 'doc', 'publish']

function formatCommand(value: string | string[] | null): string {
  if (value === null) return '—'
  if (Array.isArray(value)) return value.map(v => `\`${v}\``).join(' + ')
  return `\`${value}\``
}

function generateToolchainTable(name: string, variant: 'spec' | 'guide'): string {
  const toolchain = toolchainsData.toolchains[name]
  if (!toolchain) {
    return `> ⚠️ Toolchain "${name}" not found.\n`
  }

  const cmds = toolchain.commands
  const commandOrder = variant === 'spec' ? specCommandOrder : guideCommandOrder
  const headerLabel = variant === 'spec' ? 'Implementation' : 'Runs'

  const rows = commandOrder
    .filter(cmd => cmd in cmds && (variant === 'spec' || cmds[cmd] !== null))
    .map(cmd => `| \`${cmd}\` | ${formatCommand(cmds[cmd])} |`)

  if (rows.length === 0) {
    return `> No commands defined for toolchain "${name}".\n`
  }

  return [
    `| Command | ${headerLabel} |`,
    '|---------|----------------|',
    ...rows,
    ''
  ].join('\n')
}

function generateStandardCommandsTable(variant: 'full' | 'brief'): string {
  const commands = toolchainsData.commands as Record<string, { description: string }>

  if (variant === 'brief') {
    const rows = Object.entries(commands)
      .map(([name, cmd]) => `| \`${name}\` | ${cmd.description} |`)

    return [
      '| Command | Purpose |',
      '|---------|---------|',
      ...rows,
      ''
    ].join('\n')
  }

  // Full variant with Mutates column
  const rows = Object.entries(commands)
    .map(([name, cmd]) => {
      const mutates = mutatingCommands.includes(name) ? 'Yes' : 'No'
      return `| \`${name}\` | ${cmd.description} | ${mutates} |`
    })

  return [
    '| Command | Purpose | Mutates |',
    '|---------|---------|---------|',
    ...rows,
    ''
  ].join('\n')
}

function processContent(md: MarkdownIt, content: string): string | null {
  // Match <ToolchainCommands name="..." variant="..." />
  const toolchainMatch = content.match(/<ToolchainCommands\s+name="([^"]+)"(?:\s+variant="([^"]+)")?\s*\/>/)
  if (toolchainMatch) {
    const name = toolchainMatch[1]
    const variant = (toolchainMatch[2] || 'spec') as 'spec' | 'guide'
    const table = generateToolchainTable(name, variant)
    return md.render(table)
  }

  // Match <StandardCommands variant="..." /> or <StandardCommands />
  const standardMatch = content.match(/<StandardCommands(?:\s+variant="([^"]+)")?\s*\/>/)
  if (standardMatch) {
    const variant = (standardMatch[1] || 'full') as 'full' | 'brief'
    const table = generateStandardCommandsTable(variant)
    return md.render(table)
  }

  return null
}

export function toolchainCommandsPlugin(md: MarkdownIt) {
  // Store the original render function
  const defaultRender = md.renderer.rules.html_block || function(tokens, idx) {
    return tokens[idx].content
  }

  md.renderer.rules.html_block = (tokens, idx, options, env, self) => {
    const content = tokens[idx].content
    const result = processContent(md, content)
    if (result !== null) {
      return result
    }
    return defaultRender(tokens, idx, options, env, self)
  }

  // Also handle inline HTML
  const defaultInlineRender = md.renderer.rules.html_inline || function(tokens, idx) {
    return tokens[idx].content
  }

  md.renderer.rules.html_inline = (tokens, idx, options, env, self) => {
    const content = tokens[idx].content
    const result = processContent(md, content)
    if (result !== null) {
      return result
    }
    return defaultInlineRender(tokens, idx, options, env, self)
  }
}
