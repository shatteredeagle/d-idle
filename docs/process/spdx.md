# SPDX header policy (ADR 0010)

The repo is GPL-3.0-or-later (LICENSE). New source files (.go, .sh, .py, .js) SHOULD carry `// SPDX-License-Identifier: GPL-3.0-or-later` (`#` for shell and Python) as the first line (after a shebang). Existing files are not mass-edited; add the line when you touch a file anyway. Not a gate yet: a warn-only changecheck rule for new files without the line is proposed, not built.
