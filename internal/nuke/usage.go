package nuke

const usageText = `nuke kills processes by port, pid, or process name.

Usage:
  nuke port <port> [port...] [--force] [--dry-run] [--yes]
  nuke port -r <start-end> [--force] [--dry-run] [--yes]
  nuke port --range <start-end> [--force] [--dry-run] [--yes]

  nuke pid <pid> [pid...] [--force] [--dry-run] [--yes]
  nuke pid -n <name> [--force] [--dry-run] [--yes]
  nuke pid --name <name> [--force] [--dry-run] [--yes]

Flags:
  --force     use hard kill instead of graceful terminate
  --dry-run   show what would be killed without killing anything
  --yes       skip confirmation prompts

Rules:
  target modes cannot be combined
  nuke port 3000 -r 4000-5000 is invalid
  nuke pid 1234 -n main is invalid
`

func Usage() string {
	return usageText
}
