// d2d's get-hit DRAWN FRAME before the patch (d2lod-bot, card rendergh): the
// frame d2d's renderer picks at each tick of the hold, line for line from the
// clone at PR #136's head eb89646e (the hold's length is #136's; the frame is
// upstream main's), on the cases tools/emu/hit_render.py writes. Prints
// `id rate ticks f0 f1 ...`, one line a case, as hit_render.py's `game` does.
//
//   g++ -O2 -std=c++20 -o hit_render_harness hit_render_harness.cpp
//   ./hit_render_harness < cases.tsv > before.tsv
//
// - the rate and the hold: components/rules/combat.hpp:211..230 (#136's
//   get_hit_rate / get_hit_ticks; a tick 40 ms, ai.hpp:379 get_hit_ms).
// - a monster or the merc (kind 1): apps/d2d/town.cpp:195..203 pushes the
//   unit with its mode's anim and mode_ms, rate 1.f (world_view.hpp:72's
//   default), hold false, frames 0; world_view.cpp:440 elapsed =
//   uint32(float(elapsed_ms - mode_ms) * rate); object_elapsed (world_view.cpp:38)
//   returns it unchanged (frames 0); composite_frames (world_view.hpp:164..165)
//   frame = (elapsed / ms_per_frame) % frames_per_direction with ms_per_frame
//   = 40 x 256 / speed (gamedata.hpp:391, integer) and the COF's frames taken
//   as AnimData's.
// - the player (kind 0, not claimed, unchanged by the patch): fight.cpp:465
//   prate = float(length_ms) / float(len), length_ms = ms_per_frame x frames
//   (gamedata.hpp:394), len = ticks x 40; town.cpp:3422 rate = prate;
//   ingame.cpp:162; then world_view.cpp:440 and composite_frames as above.
#include <algorithm>
#include <cstdint>
#include <cstdio>

static int effective_speed(int speed, int cap = 120) { return speed > 0 ? cap * speed / (cap + speed) : speed; }

static int get_hit_rate(int speed, int fhr) { return std::clamp(speed * (50 + effective_speed(fhr)) / 100, 0, 0x7fff); }

static int get_hit_ticks(int frames, int speed, int fhr) {
    const int rate = get_hit_rate(speed, fhr);
    if (rate <= 0) return 1;
    const int held = frames > 0 ? (frames * 256 - 1) / rate : 0;
    return std::max(held, 1) + 1;
}

static std::uint32_t ms_per_frame(std::uint32_t speed) { return 40u * 256u / std::max<std::uint32_t>(speed, 1); }

int main() {
    long id, kind, speed, frames, fhr;
    char line[512];
    std::printf("# id rate ticks frames...\n");
    while (std::fgets(line, sizeof line, stdin)) {
        if (line[0] == '#' || std::sscanf(line, "%ld %ld %ld %ld %ld", &id, &kind, &speed, &frames, &fhr) != 5) continue;
        const int rate = get_hit_rate(int(speed), int(fhr));
        const int ticks = get_hit_ticks(int(frames), int(speed), int(fhr));
        const std::uint32_t mpf = ms_per_frame(std::uint32_t(speed));
        const std::uint32_t fpd = std::uint32_t(frames);
        float unit_rate = 1.f;
        if (kind == 0) {
            const std::uint32_t length_ms = mpf * std::max<std::uint32_t>(fpd, 1);
            const std::uint32_t len = std::uint32_t(ticks) * 40u;
            unit_rate = len ? float(length_ms) / float(len) : 1.f;
        }
        std::printf("%ld %d %d", id, rate, ticks);
        for (int tick = 0; tick < ticks; ++tick) {
            const std::uint32_t elapsed_ms = std::uint32_t(tick) * 40u;
            const auto elapsed = std::uint32_t(float(elapsed_ms) * unit_rate);
            const int frame = fpd ? int((elapsed / mpf) % fpd) : 0;
            std::printf(" %d", frame);
        }
        std::printf("\n");
    }
    return 0;
}
