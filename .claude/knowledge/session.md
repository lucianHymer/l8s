### [21:00] [architecture] SSH initial directory behavior
**Details**: L8s SSH command currently logs users into the default home directory (/home/dev). The SSH connection is established using exec.Command("ssh", containerName) which just runs a standard SSH command. There's no mechanism to set the initial working directory to /workspace/project automatically.
**Files**: pkg/container/manager.go, pkg/cli/handlers.go, pkg/ssh/keys.go
---

### [21:04] [architecture] SSH RemoteCommand impact analysis
**Details**: Adding RemoteCommand to SSH config would NOT interfere with L8s operations:

1. **l8s exec**: Uses Manager.ExecContainer which calls client.ExecContainer - this uses Podman exec directly, NOT SSH. So RemoteCommand won't affect it.

2. **scp/file transfers**: SCP operations like "scp file.txt dev-myproject:" would still work because RemoteCommand only runs for interactive SSH sessions (when RequestTTY is yes). SCP doesn't request a TTY, so RemoteCommand is skipped.

3. **VS Code Remote SSH**: Would benefit from RemoteCommand - VS Code would open directly in /workspace/project.

4. **git operations**: Git push/pull to container remotes use SSH for transport but don't execute commands interactively, so RemoteCommand won't interfere.

5. **Non-interactive SSH commands**: Commands like "ssh dev-myproject ls" would need special handling - RemoteCommand would try to cd first. Could be solved by making RemoteCommand check if it's interactive: "cd /workspace/project 2>/dev/null || true; exec $SHELL -l"
**Files**: pkg/container/manager.go, pkg/ssh/keys.go, pkg/cli/handlers.go
---

### [21:18] [feature] SSH default directory to /workspace/project
**Details**: Added RemoteCommand to SSH config generation to automatically change to /workspace/project when SSHing into containers. 

Implementation:
- Modified GenerateSSHConfigEntry in pkg/ssh/keys.go
- Added: RemoteCommand cd /workspace/project && exec $SHELL -l
- This only affects interactive SSH sessions (l8s ssh or ssh dev-container)
- Non-interactive commands (ssh dev-container ls) ignore RemoteCommand by SSH design
- SCP and SFTP operations are unaffected
- RequestTTY remains at default (auto) which is perfect for our needs

The RemoteCommand ensures users land directly in their project workspace instead of the home directory, improving developer experience.
**Files**: pkg/ssh/keys.go, pkg/ssh/keys_test.go
---

