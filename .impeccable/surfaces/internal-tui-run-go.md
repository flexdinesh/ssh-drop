---
version: 1
slug: "internal-tui-run-go"
primary_target: "internal/tui/run.go"
related_targets: ["internal/tui/view.go"]
---

# SSH Drop TUI

Mode: Operate. Full terminal flow for local screenshot handoff to a configured SSH remote. Code-first, approved Route Line. Host list remains ssh-drop.conf; no SSH config discovery.

## Direction contract

THESIS: One dominant file action and one route show the handoff; metadata stays compact. The selected host remains visible across repeated drops.

OWN-WORLD: Terminal-native monospace, lime focus, neutral text, restrained borders, amber warnings and red errors. Respect the terminal background and support light themes.

STORY: Choose a host, authenticate, drop a file, see the remote path copied, and drop again. Password input is masked and occurs during connection setup. Clipboard failure never masquerades as upload failure.

FIRST VIEWPORT: Brand and remote above the prominent path field; FILE → SSH → CLIPBOARD below it; last result and keyboard help follow. Compact terminals collapse spacing rather than shrinking text. Signature interaction: the path field clears and invites another image while the previous copied path remains visible.

FORM: Route Line, chosen explicitly by the user. Direction seed db93e774; reproduced roll evidence in .impeccable/review/seed-roll.txt. The user's choice governs composition. No comps govern implementation; the approved comparison mockup is a direction reference.

FINISH: unreviewed and undocumented is unfinished; this build ends with the finish review, the verdict, DESIGN.md, and every shipping raster carrying its provenance

Constraints: preserve regular-file support, CLI flags, config and transfer summary. Key/agent authentication precedes password fallback. Reuse an authenticated SSH master within the TUI session; reconnect after disconnection. No unresolved decisions.
