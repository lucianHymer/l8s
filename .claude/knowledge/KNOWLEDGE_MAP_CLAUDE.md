# 📚 L8s Knowledge Map

## 🏗️ Architecture

- @architecture/command_factory.md - Dual factory pattern for fast CLI startup
- @architecture/git_native_design.md - Git extension architecture with deterministic container naming
- @architecture/ssh_container_operations.md - Remote-only container management via SSH
- @architecture/zsh_completion.md - Sophisticated tab completion architecture
- @architecture/ssh_certificate_timing.md - Pre-startup certificate configuration strategy

## 📐 Patterns

- @patterns/cobra_command_structure.md - Consistent command implementation pattern
- @patterns/container_manager_interface.md - Adding new container operations

## ✨ Features

- @features/paste_command.md - Clipboard sharing between host and containers
- @features/ssh_certificate_authority.md - Secure container connections with CA

## 🔒 Security

- @security/ssh_certificate_authority.md - Complete CA implementation for MITM protection

## 🧪 Testing

- @testing/git_native_test_updates.md - Test changes for git-native architecture
- @testing/make_ci_requirements.md - Comprehensive CI validation process
- @testing/zsh_completion_framework.md - Custom framework with 91 tests

## 🔧 Configuration

- @config/github_token.md - Fine-grained personal access tokens for development

## 📦 Dependencies

- @dependencies/github_cli.md - GitHub CLI installation for development workflows

## ⚠️ Gotchas

- @gotchas/branch_checkout_on_ssh.md - Container branch synchronization issue
- @gotchas/ssh_certificates_rebuild.md - Certificate setup missing in rebuild command
- @gotchas/testing_unexported_methods.md - Handler testing limitations and solutions
- @gotchas/zsh_completion_flags.md - Nonexistent list command flags issue