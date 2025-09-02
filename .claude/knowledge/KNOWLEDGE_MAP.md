# 📚 L8s Knowledge Map

*Last updated: 2025-09-02*

## 🏗️ Architecture

- [Command Factory Architecture](architecture/command_factory.md) - Dual factory pattern for fast CLI startup
- [SSH and Container Operations](architecture/ssh_container_operations.md) - Remote-only container management via SSH
- [ZSH Completion System](architecture/zsh_completion.md) - Sophisticated tab completion architecture
- [SSH Certificate Timing](architecture/ssh_certificate_timing.md) - Pre-startup certificate configuration strategy

## 📐 Patterns

- [Cobra Command Structure](patterns/cobra_command_structure.md) - Consistent command implementation pattern
- [ContainerManager Interface](patterns/container_manager_interface.md) - Adding new container operations

## ✨ Features

- [Paste Command](features/paste_command.md) - Clipboard sharing between host and containers
- [SSH Certificate Authority](features/ssh_certificate_authority.md) - Secure container connections with CA

## 🔒 Security

- [SSH Certificate Authority](security/ssh_certificate_authority.md) - Complete CA implementation for MITM protection

## 🧪 Testing

- [Make CI Requirements](testing/make_ci_requirements.md) - Comprehensive CI validation process
- [ZSH Completion Test Framework](testing/zsh_completion_framework.md) - Custom framework with 91 tests

## ⚠️ Gotchas

- [Testing Unexported Methods](gotchas/testing_unexported_methods.md) - Handler testing limitations and solutions
- [ZSH Completion Flags](gotchas/zsh_completion_flags.md) - Nonexistent list command flags issue

---

*This knowledge base is automatically maintained by the Mim system.*