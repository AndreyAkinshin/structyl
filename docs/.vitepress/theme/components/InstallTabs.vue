<script setup lang="ts">
import { ref } from 'vue'
import CodeBlock from './CodeBlock.vue'

const activeTab = ref('unix')

const tabs = [
  { id: 'unix', label: 'macOS / Linux' },
  { id: 'windows', label: 'Windows' },
  { id: 'go', label: 'Go Install' }
]

function setActiveTab(tabId: string) {
  activeTab.value = tabId
}
</script>

<template>
  <div class="install-tabs-container">
    <div class="install-tabs">
      <button
        v-for="tab in tabs"
        :key="tab.id"
        class="install-tab"
        :class="{ active: activeTab === tab.id }"
        @click="setActiveTab(tab.id)"
      >
        {{ tab.label }}
      </button>
    </div>

    <!-- macOS / Linux Tab -->
    <div class="install-tab-content" :class="{ active: activeTab === 'unix' }">
      <h2>Quick Install</h2>
      <p>Run this command in your terminal:</p>
      <CodeBlock
        label="Terminal"
        code="curl -fsSL https://structyl.akinshin.dev/install.sh | sh"
      />

      <h2>Install Specific Version</h2>
      <CodeBlock
        label="Terminal"
        code="curl -fsSL https://structyl.akinshin.dev/install.sh | sh -s -- --version 0.1.0"
      />

      <h2>What Gets Installed</h2>
      <p>
        The installer creates <code class="inline-code">~/.structyl/</code> with:
      </p>
      <ul>
        <li><code class="inline-code">bin/structyl</code> - Version manager shim</li>
        <li><code class="inline-code">versions/X.Y.Z/</code> - Installed binaries</li>
        <li><code class="inline-code">default-version</code> - Default version file</li>
      </ul>
    </div>

    <!-- Windows Tab -->
    <div class="install-tab-content" :class="{ active: activeTab === 'windows' }">
      <h2>Quick Install</h2>
      <p>Run this command in PowerShell:</p>
      <CodeBlock
        label="PowerShell"
        code="irm https://structyl.akinshin.dev/install.ps1 | iex"
      />

      <h2>Install Specific Version</h2>
      <CodeBlock
        label="PowerShell"
        code="irm https://structyl.akinshin.dev/install.ps1 -OutFile install.ps1
.\install.ps1 -Version 0.1.0"
      />

      <h2>What Gets Installed</h2>
      <p>
        The installer creates <code class="inline-code">%USERPROFILE%\.structyl\</code> with:
      </p>
      <ul>
        <li><code class="inline-code">bin\structyl.cmd</code> - Version manager shim</li>
        <li><code class="inline-code">versions\X.Y.Z\</code> - Installed binaries</li>
        <li><code class="inline-code">default-version</code> - Default version file</li>
      </ul>
    </div>

    <!-- Go Install Tab -->
    <div class="install-tab-content" :class="{ active: activeTab === 'go' }">
      <h2>Install with Go</h2>
      <p>If you have Go 1.22+ installed:</p>
      <CodeBlock
        label="Terminal"
        code="go install github.com/AndreyAkinshin/structyl/cmd/structyl@latest"
      />

      <h2>Install Specific Version</h2>
      <CodeBlock
        label="Terminal"
        code="go install github.com/AndreyAkinshin/structyl/cmd/structyl@v0.1.0"
      />

      <p style="margin-top: 1.5rem;">
        <strong>Note:</strong> Go install doesn't support version pinning. For multi-version management, use the binary installer instead.
      </p>
    </div>
  </div>

  <!-- Version Pinning Section -->
  <div class="install-section">
    <h2>Version Pinning</h2>
    <p>
      Pin a project to a specific Structyl version by creating a <code class="inline-code">.structyl/version</code> file:
    </p>
    <CodeBlock
      label="Terminal"
      code="mkdir -p .structyl && echo '0.1.0' > .structyl/version"
    />
    <p>
      The shim automatically detects this file and uses the specified version. This works from any subdirectory in your project. For new projects, running <code class="inline-code">structyl init</code> creates this file automatically.
    </p>

    <h2>Version Resolution</h2>
    <p>When you run <code class="inline-code">structyl</code>, the version is resolved in this order:</p>
    <ol>
      <li><code class="inline-code">STRUCTYL_VERSION</code> environment variable</li>
      <li><code class="inline-code">.structyl/version</code> file (searches up to root)</li>
      <li><code class="inline-code">~/.structyl/default-version</code> file</li>
      <li>Latest installed version</li>
    </ol>
  </div>

  <!-- Nightly Builds Section -->
  <div class="install-section">
    <h2>Nightly Builds</h2>
    <p>
      Install the latest development build from the main branch:
    </p>
    <CodeBlock
      label="Terminal"
      code="curl -fsSL https://structyl.akinshin.dev/install.sh | sh -s -- --version nightly"
    />
    <p>
      Pin a project to nightly builds:
    </p>
    <CodeBlock
      label="Terminal"
      code="mkdir -p .structyl && echo 'nightly' > .structyl/version"
    />
    <p style="font-size: 0.875rem; margin-top: 1rem;">
      Nightly builds are automatically updated on every push to main. Re-run the install command to update.
    </p>
  </div>

  <!-- Managing Versions Section -->
  <div class="install-section">
    <h2>Managing Versions</h2>

    <h3>Install Additional Versions</h3>
    <CodeBlock
      label="Terminal"
      code="curl -fsSL https://structyl.akinshin.dev/install.sh | sh -s -- --version 0.2.0"
    />

    <h3>Set Default Version</h3>
    <CodeBlock
      label="Terminal"
      code="echo '0.2.0' > ~/.structyl/default-version"
    />

    <h3>List Installed Versions</h3>
    <CodeBlock
      label="Terminal"
      code="ls ~/.structyl/versions/"
    />

    <h3>Remove a Version</h3>
    <CodeBlock
      label="Terminal"
      code="rm -rf ~/.structyl/versions/0.1.0"
    />
  </div>

  <!-- Verify Installation Section -->
  <div class="install-section">
    <h2>Verify Installation</h2>
    <CodeBlock
      label="Terminal"
      code="structyl version"
    />
  </div>
</template>
