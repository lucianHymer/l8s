# 📚 L8s Knowledge Map

## 🏗️ Architecture

- @architecture/command_factory.md - Dual factory pattern for fast CLI startup
- @architecture/cobra_command_grouping.md - Built-in Cobra support for organized help output
- @architecture/git_native_design.md - Git extension architecture with deterministic container naming
- @architecture/ssh_container_operations.md - Remote-only container management via SSH
- @architecture/zsh_completion.md - Sophisticated tab completion architecture
- @architecture/ssh_certificate_timing.md - Pre-startup certificate configuration strategy
- @architecture/ssh_default_directory.md - Application-level workspace navigation
- @architecture/host_integration_embedding.md - Binary embedding system for host tools
- @architecture/zsh_plugin_duplication.md - Dual plugin locations requiring sync
- @architecture/team_session_management.md - Persistent sessions with dtach

## 📐 Patterns

- @patterns/cobra_command_structure.md - Consistent command implementation pattern
- @patterns/container_manager_interface.md - Adding new container operations
- @patterns/standalone_scripts_for_ssh.md - Executable scripts vs shell functions

## ✨ Features

- @features/command_grouping.md - Organized help output with command categories
- @features/paste_command.md - Clipboard sharing between host and containers
- @features/ssh_certificate_authority.md - Secure container connections with CA
- @features/github_cli_origin_remote.md - Automatic origin remote replication for gh CLI
- @features/slash_commands.md - Custom Claude Code commands for development workflows
- @features/web_port_forwarding.md - Automatic port 3000 forwarding for web apps

## 🔒 Security

- @security/ssh_certificate_authority.md - Complete CA implementation for MITM protection

## 🧪 Testing

- @testing/git_native_test_updates.md - Test changes for git-native architecture
- @testing/make_ci_requirements.md - Comprehensive CI validation process
- @testing/zsh_completion_framework.md - Custom framework with 91 tests

## 🔧 Configuration

- @config/github_token.md - Fine-grained personal access tokens for development
- @config/ssh_connection_stability.md - Enhanced keepalive settings

## 📦 Dependencies

- @dependencies/github_cli.md - GitHub CLI installation for development workflows

## ⚠️ Gotchas

- @gotchas/branch_checkout_on_ssh.md - Container branch synchronization issue
- @gotchas/remote_command_breaks_git.md - SSH RemoteCommand conflicts with git operations
- @gotchas/ssh_certificates_rebuild.md - Certificate setup missing in rebuild command
- @gotchas/testing_unexported_methods.md - Handler testing limitations and solutions
- @gotchas/zsh_completion_flags.md - Nonexistent list command flags issue
- @gotchas/mcp_server_protocol_error.md - Mim MCP server validation errors
- @gotchas/zsh_plugin_missing_from_embedded.md - Container tab completion broken
- @gotchas/claude_code_at_syntax.md - @ syntax limitations in slash commands
- @gotchas/containerfiles_location.md - Files in pkg/embed/containers/ not pkg/containerfiles/
- @gotchas/ssh_path_dependency.md - Non-interactive SSH PATH issues
- @gotchas/dotfiles_missing_on_rebuild.md - Dotfiles not redeployed during container rebuild