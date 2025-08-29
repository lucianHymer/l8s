### [14:50] [security] SSH host key verification issue
**Details**: Currently L8s disables SSH host key checking completely by setting:
- StrictHostKeyChecking no
- UserKnownHostsFile /dev/null

This is done in pkg/ssh/keys.go when creating SSH config entries. While convenient, this makes the system vulnerable to MITM attacks. The TODO suggests implementing proper host key management using either:
1. A CA (Certificate Authority) approach
2. Persistent host keys that are trusted/verified

The current approach throws away all host key information and never verifies the container's SSH host identity.
**Files**: pkg/ssh/keys.go, pkg/cli/handlers.go, README.md
---

