import DefaultTheme from 'vitepress/theme'
import type { Theme } from 'vitepress'

import './custom.css'

import InstallTabs from './components/InstallTabs.vue'
import CodeBlock from './components/CodeBlock.vue'

export default {
  extends: DefaultTheme,
  enhanceApp({ app }) {
    app.component('InstallTabs', InstallTabs)
    app.component('CodeBlock', CodeBlock)
  }
} satisfies Theme
