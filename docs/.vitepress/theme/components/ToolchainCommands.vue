<script setup lang="ts">
import { computed } from 'vue'
import toolchainsData from '../../../../internal/cli/toolchains_template.json'

const props = defineProps<{
  name: string
  variant?: 'guide' | 'spec'
}>()

const toolchain = computed(() => toolchainsData.toolchains[props.name])

// Derive command order from the source of truth
const commandOrder = Object.keys(toolchainsData.commands)

const guideCommands = [
  'build', 'build:release', 'test', 'check', 'check:fix',
  'restore', 'bench', 'pack', 'doc', 'demo'
]

function formatCommand(value: string | string[] | null): string {
  if (value === null) return '—'
  if (Array.isArray(value)) return value.join(' + ')
  return `\`${value}\``
}

const commands = computed(() => {
  if (!toolchain.value?.commands) return []

  const cmds = toolchain.value.commands
  const filterList = props.variant === 'spec' ? commandOrder : guideCommands

  return filterList
    .filter(cmd => cmd in cmds && (props.variant === 'spec' || cmds[cmd] !== null))
    .map(cmd => ({
      name: cmd,
      value: cmds[cmd],
      formatted: formatCommand(cmds[cmd])
    }))
})

const headerLabel = computed(() => props.variant === 'spec' ? 'Implementation' : 'Runs')
</script>

<template>
  <table v-if="toolchain" class="toolchain-commands-table">
    <thead>
      <tr>
        <th>Command</th>
        <th>{{ headerLabel }}</th>
      </tr>
    </thead>
    <tbody>
      <tr v-for="cmd in commands" :key="cmd.name">
        <td><code>{{ cmd.name }}</code></td>
        <td v-if="cmd.value === null">—</td>
        <td v-else-if="Array.isArray(cmd.value)">{{ cmd.value.join(' + ') }}</td>
        <td v-else><code>{{ cmd.value }}</code></td>
      </tr>
    </tbody>
  </table>
  <p v-else class="toolchain-error">Toolchain "{{ name }}" not found.</p>
</template>

<style scoped>
.toolchain-commands-table {
  width: 100%;
  margin: 1rem 0;
}

.toolchain-commands-table th {
  text-align: left;
}

.toolchain-error {
  color: var(--vp-c-danger-1);
}
</style>
