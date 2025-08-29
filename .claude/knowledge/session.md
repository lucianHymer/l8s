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

### [17:28] [architecture] SSH CA Implementation Requirements
**Details**: L8s currently disables SSH host key verification (StrictHostKeyChecking no, UserKnownHostsFile /dev/null) for container connections, leaving them vulnerable to MITM attacks. The SSH CA implementation will:
1. Generate CA keypair during init (stored in ~/.config/l8s/ca/)
2. Sign each container's SSH host key during creation 
3. Configure SSH to trust the CA via known_hosts entry
4. Enable StrictHostKeyChecking for secure connections
Key integration points:
- pkg/ssh/ca.go - new CA management package
- runInit handler - generate CA during initialization
- CreateContainer - sign host keys during container creation
- GenerateSSHConfigEntry - enable strict checking with CA trust
- Config struct - add CA path fields
**Files**: pkg/ssh/keys.go, pkg/cli/handlers.go, pkg/container/manager.go, pkg/config/config.go
---

### [17:45] [features] SSH Certificate Authority Implementation
**Details**: Successfully implemented SSH Certificate Authority system for L8s to replace insecure host key handling:

Key Components:
1. CA package (pkg/ssh/ca.go) - Manages CA keypair generation, host key signing, and known_hosts entries
2. Config updates - Added CA paths to both config.Config and container.Config structs
3. Init command - Generates CA during initialization and creates known_hosts file
4. Container creation - Generates and signs host keys, copies certificates to containers
5. SSH config - Uses StrictHostKeyChecking with CA-trusted known_hosts

The implementation provides:
- Cryptographic verification of container identities
- Protection against MITM attacks
- Seamless user experience (no SSH warnings)
- 10-year certificate validity for dev environments
- Automatic trust for all L8s containers

All tests pass including new CA unit tests.
**Files**: pkg/ssh/ca.go, pkg/ssh/ca_test.go, pkg/config/config.go, pkg/container/types.go, pkg/container/manager.go, pkg/cli/handlers.go, pkg/ssh/keys.go
---

