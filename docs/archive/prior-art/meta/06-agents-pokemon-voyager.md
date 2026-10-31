# 6. LLM agents that play: Claude Plays Pokemon, Voyager, and the rest (only as the "direction" layer)

## Claude Plays Pokemon (Anthropic, Pokemon Red, emulator)
Sources: https://michaelyliu6.github.io/posts/claude-plays-pokemon (V, fetched; early-stream snapshot, harness changed later), search summaries (S). Not open source as far as found (U).
- Harness loop (V): assemble prompt (tools + system prompt + knowledge base + summary + history) -> model -> run tools -> if history > max turns, summarize and clear -> save state.
- Three tools (V): `use_emulator` (button list; returns screenshot plus a RAM-derived overlay: location, coordinates, money, badges, inventory, party HP), `navigator` (target tile -> A* on walkable tiles -> buttons; clear error when no path), `update_knowledge_base`.
- Memory (V): knowledge base dictionary with `current_status`, `game_progress`, `current_objectives`, `inventory`; survives summaries; a second LLM reviews it for inconsistency after each summary.
- Numbers (V from blog): ~35,000 actions / ~140 hours for three badges; 78 hours in Mt. Moon in one run, 24+ in another.
- Failure modes (V): walking into walls, choosing trap routes, re-fighting beaten trainers, declaring goals complete prematurely, forgetting recorded facts, confusing NPCs with self, not noticing inputs failed.
- Takeaway: memory read (RAM overlay) plus deterministic navigation tool made the LLM viable; pure vision was worse. This is our architecture inverted: we have the deterministic executor, an LLM could be an optional goal planner. Goal set by human = objectives text in the knowledge base.

## Voyager (Minecraft, NVIDIA/Caltech et al., 2023)
- https://arxiv.org/abs/2305.16291 (V via search), repo https://github.com/MineDojo/Voyager (MIT, ~7.3k stars, last push 2024-04) (V `gh api`).
- Three parts (V): automatic curriculum (what to do next), skill library of executable code (retrieved by embedding), iterative prompting with environment feedback and self-verification. Action space is code over the Mineflayer API.
- Takeaways: the curriculum is a goal manager; the skill library is "recorded routes", but written and verified by the agent; code-as-action gives composability. Reuse: MIT, but needs GPT-4-class API; stale since 2024.

## Others (S unless noted)
- SIMA 2 (DeepMind, Nov 2025): Gemini-based, plays 3D games from pixels and keyboard/mouse, takes natural-language goals, self-improves with a task generator and reward model; limited research preview; struggles with long-horizon steps and limited memory (https://deepmind.google/sima, https://techcrunch.com/2025/11/13/googles-sima-2-agent-uses-gemini-to-reason-and-act-in-virtual-world).
- BALROG (MIT, 276 stars, V `gh api`; https://arxiv.org/abs/2411.13543 V): benchmark over NetHack Learning Environment and other games; finding: LLMs/VLMs weak on long-horizon, vision often worse than text. NLE (facebookresearch/nle, ~1k stars) is the text-observation interface (U details).
- Factorio Learning Environment (https://arxiv.org/abs/2503.09617 V, repo 1.2k stars, licence "other"): agent writes Python against a game API; models plan short-horizon well but fail at complex automation.
- Mineflayer (MIT, 7.7k stars) and Baritone (LGPL-3.0, 9.3k stars) (V `gh api`): headless-protocol bot API and a pathfinder; community fleet managers are small experiments (S).

## What this means for us
- LLMs are goal-setters and debuggers, not frame-by-frame players: expensive (140 h for 3 badges) and loop-prone. Keep our deterministic bot; optionally let an LLM draft goal queues or read error bundles. Needs no new layer now.
- State bits to steal: periodic summaries, a "current objective" text the user can edit, an overlay of verified game state (we already read memory).
