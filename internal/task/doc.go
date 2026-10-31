// Package task is the task layer (docs/progression/tasklayer.md, docs/task.md): a step graph with preconditions that
// sits above bot.Runner's 8 states. The next step is a pure function of the World each tick, so there is no saved
// cursor; death, restart and a new game resume by re-deriving. The package imports only goals (for Progress) and the
// standard library; the bot supplies Env and Actions adapters over nav, combat, town and loot.
package task
