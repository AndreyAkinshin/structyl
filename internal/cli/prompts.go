package cli

import _ "embed"

// AgentsPromptFileName is the name of the LLM prompt file created during init.
const AgentsPromptFileName = "AGENTS.md"

// AgentsPromptContent contains LLM instructions for AI assistants working with Structyl projects.
//
//go:embed agents_template.md
var AgentsPromptContent string
