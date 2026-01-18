<script setup lang="ts">
import toolchainsData from '../../../../internal/cli/toolchains_template.json'

const props = defineProps<{
  toolchains: string[]
}>()

function getCommand(name: string, cmd: string): string {
  const toolchain = toolchainsData.toolchains[name]
  if (!toolchain?.commands?.[cmd]) return '—'
  const value = toolchain.commands[cmd]
  if (Array.isArray(value)) return value.join(' + ')
  return value
}
</script>

<template>
  <table class="toolchain-overview-table">
    <thead>
      <tr>
        <th>Toolchain</th>
        <th>Build Command</th>
        <th>Test Command</th>
      </tr>
    </thead>
    <tbody>
      <tr v-for="name in toolchains" :key="name">
        <td>{{ name }}</td>
        <td><code>{{ getCommand(name, 'build') }}</code></td>
        <td><code>{{ getCommand(name, 'test') }}</code></td>
      </tr>
    </tbody>
  </table>
</template>

<style scoped>
.toolchain-overview-table {
  width: 100%;
  margin: 1rem 0;
}

.toolchain-overview-table th {
  text-align: left;
}
</style>
