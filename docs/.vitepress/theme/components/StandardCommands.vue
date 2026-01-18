<script setup lang="ts">
import { computed } from 'vue'
import toolchainsData from '../../../../internal/cli/toolchains_template.json'

const props = defineProps<{
  variant?: 'full' | 'brief'
}>()

const commands = computed(() => {
  return Object.entries(toolchainsData.commands).map(([name, cmd]) => ({
    name,
    description: (cmd as { description: string }).description
  }))
})

const showMutates = computed(() => props.variant !== 'brief')
</script>

<template>
  <table class="standard-commands-table">
    <thead>
      <tr>
        <th>Command</th>
        <th>Purpose</th>
        <th v-if="showMutates">Mutates</th>
      </tr>
    </thead>
    <tbody>
      <tr v-for="cmd in commands" :key="cmd.name">
        <td><code>{{ cmd.name }}</code></td>
        <td>{{ cmd.description }}</td>
        <td v-if="showMutates">{{ ['clean', 'restore', 'build', 'build:release', 'check:fix', 'pack', 'doc', 'publish'].includes(cmd.name) ? 'Yes' : 'No' }}</td>
      </tr>
    </tbody>
  </table>
</template>

<style scoped>
.standard-commands-table {
  width: 100%;
  margin: 1rem 0;
}

.standard-commands-table th {
  text-align: left;
}
</style>
