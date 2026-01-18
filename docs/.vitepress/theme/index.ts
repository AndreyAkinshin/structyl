import DefaultTheme from 'vitepress/theme'
import type { Theme } from 'vitepress'

import './custom.css'

import InstallTabs from './components/InstallTabs.vue'
import CodeBlock from './components/CodeBlock.vue'
import ToolchainCommands from './components/ToolchainCommands.vue'
import ToolchainOverview from './components/ToolchainOverview.vue'

export default {
  extends: DefaultTheme,
  enhanceApp({ app }) {
    app.component('InstallTabs', InstallTabs)
    app.component('CodeBlock', CodeBlock)
    app.component('ToolchainCommands', ToolchainCommands)
    app.component('ToolchainOverview', ToolchainOverview)
  }
} satisfies Theme
