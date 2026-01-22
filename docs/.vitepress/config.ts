import { defineConfig } from 'vitepress'
import { toolchainCommandsPlugin } from './markdown/toolchainCommands'

export default defineConfig({
  title: 'Structyl',
  description: 'Build orchestration for multi-language projects',

  head: [
    ['link', { rel: 'icon', type: 'image/svg+xml', href: '/logo.svg' }],
  ],

  markdown: {
    config: (md) => {
      md.use(toolchainCommandsPlugin)
    }
  },

  themeConfig: {
    logo: '/logo.svg',

    nav: [
      { text: 'Getting Started', link: '/getting-started/' },
      { text: 'Guide', link: '/guide/configuration' },
      { text: 'Reference', link: '/reference/error-codes' },
      { text: 'Specs', link: '/specs/' },
    ],

    sidebar: {
      '/getting-started/': [
        {
          text: 'Getting Started',
          items: [
            { text: 'Introduction', link: '/getting-started/' },
            { text: 'Installation', link: '/getting-started/installation' },
            { text: 'Quick Start', link: '/getting-started/quick-start' },
          ],
        },
      ],

      '/guide/': [
        {
          text: 'Guide',
          items: [
            { text: 'Configuration', link: '/guide/configuration' },
            { text: 'Project Structure', link: '/guide/project-structure' },
            { text: 'Targets', link: '/guide/targets' },
            { text: 'Commands', link: '/guide/commands' },
            { text: 'Toolchains', link: '/guide/toolchains' },
            { text: 'Testing', link: '/guide/testing' },
            { text: 'Version Management', link: '/guide/version-management' },
            { text: 'Docker', link: '/guide/docker' },
            { text: 'CI Integration', link: '/guide/ci-integration' },
          ],
        },
      ],

      '/reference/': [
        {
          text: 'Reference',
          items: [
            { text: 'Error Codes', link: '/reference/error-codes' },
            { text: 'Glossary', link: '/specs/glossary' },
            { text: 'JSON Schema', link: '/reference/schema' },
          ],
        },
      ],

      '/specs/': [
        {
          text: 'Specifications',
          items: [
            { text: 'Overview', link: '/specs/' },
            { text: 'Configuration', link: '/specs/configuration' },
            { text: 'Project Structure', link: '/specs/project-structure' },
            { text: 'Targets', link: '/specs/targets' },
            { text: 'Commands', link: '/specs/commands' },
            { text: 'Toolchains', link: '/specs/toolchains' },
            { text: 'Test System', link: '/specs/test-system' },
            { text: 'Version Management', link: '/specs/version-management' },
            { text: 'Docker', link: '/specs/docker' },
            { text: 'CI Integration', link: '/specs/ci-integration' },
            { text: 'Cross-Platform', link: '/specs/cross-platform' },
            { text: 'Error Handling', link: '/specs/error-handling' },
            { text: 'Go Architecture', link: '/specs/go-architecture' },
            { text: 'Glossary', link: '/specs/glossary' },
            { text: 'Stability', link: '/specs/stability' },
          ],
        },
      ],
    },

    socialLinks: [
      { icon: 'github', link: 'https://github.com/AndreyAkinshin/structyl' },
    ],

    footer: {
      message: '© 2026 Andrey Akinshin <a href="https://opensource.org/licenses/MIT">MIT</a>',
    },

    search: {
      provider: 'local',
    },

    editLink: {
      pattern: 'https://github.com/AndreyAkinshin/structyl/edit/main/docs/:path',
      text: 'Edit this page on GitHub',
    },
  },
})
