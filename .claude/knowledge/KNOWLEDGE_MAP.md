# 📚 L8s Knowledge Map

*Last updated: 2025-10-29*

## 🏗️ Architecture

- [Command Factory Architecture](architecture/command_factory.md) - Dual factory pattern for fast CLI startup
- [Cobra Command Grouping](architecture/cobra_command_grouping.md) - Built-in Cobra support for organized help output
- [Git-Native Design](architecture/git_native_design.md) - Git extension architecture with deterministic container naming
- [SSH and Container Operations](architecture/ssh_container_operations.md) - Remote-only container management via SSH
- [ZSH Completion System](architecture/zsh_completion.md) - Sophisticated tab completion architecture
- [SSH Certificate Timing](architecture/ssh_certificate_timing.md) - Pre-startup certificate configuration strategy
- [SSH Default Directory](architecture/ssh_default_directory.md) - Application-level workspace navigation
- [Host Integration Embedding](architecture/host_integration_embedding.md) - Binary embedding system for host tools
- [ZSH Plugin Duplication](architecture/zsh_plugin_duplication.md) - Dual plugin locations requiring sync
- [Team Session Management](architecture/team_session_management.md) - Persistent sessions with dtach *(New: 2025-10-29)*

## 📐 Patterns

- [Cobra Command Structure](patterns/cobra_command_structure.md) - Consistent command implementation pattern
- [ContainerManager Interface](patterns/container_manager_interface.md) - Adding new container operations
- [Standalone Scripts for SSH](patterns/standalone_scripts_for_ssh.md) - Executable scripts vs shell functions *(New: 2025-10-29)*

## ✨ Features

- [Command Grouping](features/command_grouping.md) - Organized help output with command categories
- [Paste Command](features/paste_command.md) - Clipboard sharing between host and containers
- [SSH Certificate Authority](features/ssh_certificate_authority.md) - Secure container connections with CA
- [GitHub CLI Origin Remote](features/github_cli_origin_remote.md) - Automatic origin remote replication for gh CLI
- [Slash Commands](features/slash_commands.md) - Custom Claude Code commands for development workflows
- [Web Port Forwarding](features/web_port_forwarding.md) - Automatic port 3000 forwarding for web apps *(New: 2025-10-29)*

## 🔒 Security

- [SSH Certificate Authority](security/ssh_certificate_authority.md) - Complete CA implementation for MITM protection

## 🧪 Testing

- [Git-Native Test Updates](testing/git_native_test_updates.md) - Test changes for git-native architecture
- [Make CI Requirements](testing/make_ci_requirements.md) - Comprehensive CI validation process
- [ZSH Completion Test Framework](testing/zsh_completion_framework.md) - Custom framework with 91 tests

## 🔧 Configuration

- [GitHub Token Configuration](config/github_token.md) - Fine-grained personal access tokens for development
- [SSH Connection Stability](config/ssh_connection_stability.md) - Enhanced keepalive settings *(New: 2025-10-29)*

## 📦 Dependencies

- [GitHub CLI in Containers](dependencies/github_cli.md) - GitHub CLI installation for development workflows

## ⚠️ Gotchas

- [Branch Checkout on SSH](gotchas/branch_checkout_on_ssh.md) - Container branch synchronization issue
- [RemoteCommand Breaks Git Push](gotchas/remote_command_breaks_git.md) - SSH RemoteCommand conflicts with git operations
- [SSH Certificates Lost on Rebuild](gotchas/ssh_certificates_rebuild.md) - Certificate setup missing in rebuild command
- [Testing Unexported Methods](gotchas/testing_unexported_methods.md) - Handler testing limitations and solutions
- [ZSH Completion Flags](gotchas/zsh_completion_flags.md) - Nonexistent list command flags issue
- [MCP Server Protocol Error](gotchas/mcp_server_protocol_error.md) - Mim MCP server validation errors
- [ZSH Plugin Missing from Embedded](gotchas/zsh_plugin_missing_from_embedded.md) - Container tab completion broken
- [Claude Code @ Syntax](gotchas/claude_code_at_syntax.md) - @ syntax limitations in slash commands
- [Containerfiles Location](gotchas/containerfiles_location.md) - Files in pkg/embed/containers/ not pkg/containerfiles/ *(New: 2025-10-29)*
- [SSH PATH Dependency](gotchas/ssh_path_dependency.md) - Non-interactive SSH PATH issues *(New: 2025-10-29)*
- [Dotfiles Missing on Rebuild](gotchas/dotfiles_missing_on_rebuild.md) - Dotfiles not redeployed during container rebuild *(New: 2025-10-29)*

---

*This knowledge base is automatically maintained by the Mim system.*