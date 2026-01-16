<script setup lang="ts">
import { ref } from 'vue'

const props = defineProps<{
  label: string
  code: string
}>()

const copied = ref(false)

async function copyToClipboard() {
  try {
    await navigator.clipboard.writeText(props.code)
    copied.value = true
    setTimeout(() => {
      copied.value = false
    }, 2000)
  } catch (err) {
    console.error('Failed to copy:', err)
  }
}
</script>

<template>
  <div class="install-code-block">
    <div class="install-code-header">
      <span class="install-code-label">{{ label }}</span>
      <button
        class="install-copy-btn"
        :class="{ copied }"
        @click="copyToClipboard"
      >
        {{ copied ? 'Copied!' : 'Copy' }}
      </button>
    </div>
    <div class="install-code-content">
      <code>{{ code }}</code>
    </div>
  </div>
</template>
