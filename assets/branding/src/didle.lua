-- d.idle pixel-art brand set, original artwork. Run through Aseprite in batch mode:
--   aseprite -b --script assets/branding/src/didle.lua --script-param out=OUTDIR --script-param src=SRCDIR --script-param pal=PALDIR
-- Everything is drawn here from the 15-colour palette below (index 0 is transparent); no font, no clip art, no other artwork.

local OUT = app.params.out or "out"
local SRC = app.params.src or "src"
local PAL = app.params.pal or "palettes"

-- Palette: the project's own tokens (internal/watch/web/tokens.css): stone ramp and gold ramp.
local P = {
  {0x0d,0x0b,0x09}, -- 1 ink      (--d-stone-1)
  {0x1c,0x18,0x13}, -- 2 plate    (--d-stone-3)
  {0x24,0x1f,0x19}, -- 3 plate hi (--d-stone-4)
  {0x3a,0x32,0x2a}, -- 4 border   (--d-stone-6)
  {0x6a,0x5c,0x4d}, -- 5 dim      (--d-stone-8)
  {0x8c,0x7b,0x67}, -- 6 mid      (--d-stone-9)
  {0xb9,0xaa,0x94}, -- 7 light    (--d-stone-11)
  {0xec,0xe2,0xcf}, -- 8 pale     (--d-stone-12)
  {0x5a,0x47,0x18}, -- 9 gold 0   (--d-gold-6)
  {0x8a,0x6d,0x22}, -- 10 gold 1  (--d-gold-8)
  {0xc7,0xa4,0x4a}, -- 11 gold 2  (--d-gold-9)
  {0xdc,0xb8,0x5e}, -- 12 gold 3  (--d-gold-10)
  {0xf3,0xe2,0xa8}, -- 13 gold 4  (--d-gold-12)
  {0x14,0x11,0x0e}, -- 14 deep    (--d-stone-2)
  {0x2d,0x27,0x20}, -- 15 plate 2 (--d-stone-5)
}
local INK, PLATE, PLATE2, BORDER, DIM, MID, LIGHT, PALE = 1, 2, 3, 4, 5, 6, 7, 8
local G0, G1, G2, G3, G4, DEEP, PLATE3 = 9, 10, 11, 12, 13, 14, 15

local function floor(x) return math.floor(x) end
local function abs(x) return math.abs(x) end

-- Canvas: plain table of palette indexes.
local function canvas(w, h)
  local c = { w = w, h = h }
  for i = 1, w * h do c[i] = 0 end
  return c
end
local function put(c, x, y, i)
  if x >= 0 and y >= 0 and x < c.w and y < c.h then c[y * c.w + x + 1] = i end
end
local function get(c, x, y)
  if x < 0 or y < 0 or x >= c.w or y >= c.h then return 0 end
  return c[y * c.w + x + 1]
end
local function rect(c, x, y, w, h, i)
  for yy = y, y + h - 1 do for xx = x, x + w - 1 do put(c, xx, yy, i) end end
end

-- Pixel bytes of a canvas, scaled by an integer factor with nearest neighbour.
local function bytesOf(c, f, mono)
  local px = { [0] = "\0\0\0\0" }
  for i, col in ipairs(P) do
    px[i] = mono and "\0\0\0\255" or string.char(col[1], col[2], col[3], 255)
  end
  local rows = {}
  for y = 0, c.h - 1 do
    local r = {}
    for x = 0, c.w - 1 do r[#r + 1] = string.rep(px[c[y * c.w + x + 1]], f) end
    local row = table.concat(r)
    rows[#rows + 1] = string.rep(row, f)
  end
  return table.concat(rows)
end

local function newPalette()
  local pal = Palette(#P + 1)
  pal:setColor(0, Color{ r = 0, g = 0, b = 0, a = 0 })
  for i, col in ipairs(P) do pal:setColor(i, Color{ r = col[1], g = col[2], b = col[3], a = 255 }) end
  return pal
end

local function sprite(c, f, mono)
  local spr = Sprite(c.w * f, c.h * f, ColorMode.RGB)
  spr:setPalette(newPalette())
  spr.cels[1].image.bytes = bytesOf(c, f, mono)
  return spr
end

local function savePng(c, f, name, mono)
  local spr = sprite(c, f, mono)
  spr:saveCopyAs(OUT .. "/" .. name)
  spr:close()
end

local function saveSource(c, name)
  local spr = sprite(c, 1)
  spr:saveAs(SRC .. "/" .. name)
  spr:close()
end

-- ---------------------------------------------------------------------------
-- The mark: a gold diamond with a dark hourglass cut into it (idle = waiting), on a rounded stone plate.
-- opts.plate (default true), opts.mono (diamond filled index 1, hourglass left transparent), opts.detail (dither, finer lines)
-- ---------------------------------------------------------------------------
local function mark(c, ox, oy, N, o)
  o = o or {}
  local plate = o.plate ~= false
  local cc = (N - 1) / 2
  local line = math.max(1, floor(N / 64 + 0.5))   -- outline thickness in pixels
  if plate then
    local rad = math.max(2, floor(N * 0.16))
    for y = 0, N - 1 do for x = 0, N - 1 do
      -- rounded square mask
      local dx = math.max(rad - x, x - (N - 1 - rad), 0)
      local dy = math.max(rad - y, y - (N - 1 - rad), 0)
      if dx * dx + dy * dy <= rad * rad + 1 then
        local edge = x < line or y < line or x >= N - line or y >= N - line
          or (dx > 0 and dx * dx + dy * dy > (rad - line) * (rad - line))
        local c2 = PLATE
        if edge then c2 = (N <= 20) and INK or BORDER
        elseif y < N * 0.08 + line then c2 = PLATE2 end
        put(c, ox + x, oy + y, c2)
      end
    end end
  end
  local D = N * (plate and 0.40 or 0.49)
  local H, HW = N * (plate and 0.20 or 0.21), N * (plate and 0.15 or 0.16)
  for y = 0, N - 1 do for x = 0, N - 1 do
    local dx, dy = x - cc, y - cc
    local r = abs(dx) + abs(dy)
    if r <= D then
      local depth = D - r                        -- 0 at the rim
      local col
      if o.mono then col = INK
      else
        if dx < 0 and dy < 0 then col = G3 elseif dx >= 0 and dy < 0 then col = G2
        elseif dx < 0 then col = G2 else col = G1 end
        if depth < line then col = G0                       -- dark rim
        elseif depth < line * 2 then                       -- bevel: light from the top left
          if dy < 0 then col = (dx < 0) and G4 or G3 else col = (dx < 0) and G2 or G1 end
          if dx >= 0 and dy >= 0 then col = G1 end
        end
        if o.detail and depth >= line * 2 and depth < line * 5 and ((x + y) % 2 == 0) and dy < 0 then
          col = (dx < 0) and G4 or G3                       -- dithered highlight, native large sizes only
        end
      end
      put(c, ox + x, oy + y, col)
    end
  end end
  -- hourglass: hand-drawn bitmap on small grids, computed profile on large ones
  if N <= 24 then
    local bm = { "######", ".####.", "..##..", "..##..", ".####.", "######" }
    local x0, y0 = floor(cc - 2.5 + 0.5), floor(cc - 2.5 + 0.5)
    for r, row in ipairs(bm) do for k = 1, #row do
      if row:sub(k, k) == "#" then
        local col = o.mono and 0 or INK
        if not o.mono and (r == 1 or r == 6) then col = G4 end
        put(c, ox + x0 + k - 1, oy + y0 + r - 1, col)
      end
    end end
  else
    for y = 0, N - 1 do for x = 0, N - 1 do
      local dx, dy = x - cc, y - cc
      if abs(dy) <= H then
        local hw = 0.5 + (HW - 0.5) * (abs(dy) / H)
        if abs(dx) <= hw then
          local col = o.mono and 0 or INK
          if not o.mono then
            local inner = abs(dx) <= hw - line
            if dy > 0 and abs(dy) > H * 0.55 and inner then col = (dx < 0) and G3 or G2 end
            if abs(dx) < 0.6 and abs(dy) < H * 0.4 then col = G3 end     -- the falling grain
            if abs(dy) > H - line then col = G4 end                      -- glass caps
          end
          put(c, ox + x, oy + y, col)
        end
      end
    end end
  end
end

-- ---------------------------------------------------------------------------
-- Hand-drawn pixel fonts (drawn here as strings, no licensed font): big (5x7) for the wordmark, small (3x5) for the tagline.
-- ---------------------------------------------------------------------------
local BIG = {
  d = { "....#", "....#", ".####", "#...#", "#...#", "#...#", ".####" },
  ["."] = { ".", ".", ".", ".", ".", ".", "#" },
  i = { "#", ".", "#", "#", "#", "#", "#" },
  l = { "#", "#", "#", "#", "#", "#", "#" },
  e = { ".....", ".....", ".###.", "#...#", "#####", "#....", ".####" },
}
local SMALL = {
  a = { ".#.", "#.#", "###", "#.#", "#.#" }, d = { "..#", "###", "#.#", "#.#", "###" },
  e = { "###", "#..", "##.", "#..", "###" }, f = { "###", "#..", "##.", "#..", "#.." },
  i = { ".#.", "...", ".#.", ".#.", ".#." }, l = { "#..", "#..", "#..", "#..", "###" },
  n = { "##.", "#.#", "#.#", "#.#", "#.#" }, o = { ".#.", "#.#", "#.#", "#.#", ".#." },
  t = { "###", ".#.", ".#.", ".#.", ".#." }, u = { "#.#", "#.#", "#.#", "#.#", "###" },
  ["."] = { ".", ".", ".", ".", "#" }, [" "] = { "..", "..", "..", "..", ".." },
}
local function textWidth(font, s, scale)
  local w = 0
  for ch in s:gmatch(".") do w = w + #font[ch][1] + 1 end
  return (w - 1) * scale
end
local function text(c, x, y, s, font, scale, col, shadow, dotcol)
  local pass = shadow and { { scale, scale, shadow }, { 0, 0, col } } or { { 0, 0, col } }
  for _, p in ipairs(pass) do
    local cx = x
    for ch in s:gmatch(".") do
      local g = font[ch]
      for ry, row in ipairs(g) do for rx = 1, #row do
        if row:sub(rx, rx) == "#" then
          local k = (p[3] == col and ch == "." and dotcol) or p[3]
          rect(c, cx + p[1] + (rx - 1) * scale, y + p[2] + (ry - 1) * scale, scale, scale, k)
        end
      end end
      cx = cx + (#g[1] + 1) * scale
    end
  end
end

-- Backdrop used by the banner and the card: deep stone, a faint 8px grid of dots, double frame, corner diamonds.
local function backdrop(c)
  rect(c, 0, 0, c.w, c.h, DEEP)
  for y = 4, c.h - 1, 8 do for x = 4, c.w - 1, 8 do put(c, x, y, PLATE) end end
  for x = 0, c.w - 1 do put(c, x, 0, INK); put(c, x, c.h - 1, INK); put(c, x, 1, BORDER); put(c, x, c.h - 2, BORDER) end
  for y = 0, c.h - 1 do put(c, 0, y, INK); put(c, c.w - 1, y, INK); put(c, 1, y, BORDER); put(c, c.w - 2, y, BORDER) end
  local function gem(x, y)
    put(c, x, y, G2); put(c, x - 1, y, G1); put(c, x + 1, y, G1); put(c, x, y - 1, G1); put(c, x, y + 1, G1)
  end
  gem(5, 5); gem(c.w - 6, 5); gem(5, c.h - 6); gem(c.w - 6, c.h - 6)
end

local TAG = "unattended. offline. idle."

-- ---- icons -----------------------------------------------------------------
local masters = {}
for _, n in ipairs({ 16, 32, 48, 64 }) do
  local c = canvas(n, n); mark(c, 0, 0, n); masters[n] = c
end
saveSource(masters[32], "mark32.aseprite")
saveSource(masters[64], "mark64.aseprite")
saveSource(masters[16], "mark16.aseprite")
for _, n in ipairs({ 16, 32, 48, 64 }) do savePng(masters[n], 1, "icon-" .. n .. ".png") end
-- nearest-neighbour upscales of the 64 master
for _, f in ipairs({ 2, 4, 8, 16 }) do savePng(masters[64], f, "icon-" .. 64 * f .. ".png") end
-- native 256: drawn on the 256 grid with finer outline and dithered facets
local big = canvas(256, 256); mark(big, 0, 0, 256, { detail = true })
saveSource(big, "mark256.aseprite")
savePng(big, 1, "icon-256-native.png")
-- macOS template images: black on transparent, no plate. 16 and 32 are the sizes electron's tray uses; 22/44 the menu bar @1x/@2x.
for _, n in ipairs({ 16, 22, 32, 44 }) do
  local c = canvas(n, n); mark(c, 0, 0, n, { plate = false, mono = true })
  savePng(c, 1, "template-" .. n .. ".png", true)
  if n == 16 then saveSource(c, "template16.aseprite") end
end

-- ---- banner 1280x320 (320x80 at 4x) --------------------------------------
do
  local c = canvas(320, 80); backdrop(c)
  mark(c, 14, 8, 64, { plate = true })
  local x0 = 96
  text(c, x0, 14, "d.idle", BIG, 5, PALE, INK, G3)
  text(c, x0, 58, TAG, SMALL, 2, MID, INK)
  saveSource(c, "banner.aseprite")
  savePng(c, 4, "banner-1280x320.png")
end
-- ---- social card 1200x630 (240x126 at 5x) -------------------------------
do
  local c = canvas(240, 126); backdrop(c)
  mark(c, 88, 9, 64, { plate = true })
  local w = textWidth(BIG, "d.idle", 4)
  text(c, floor((240 - w) / 2), 77, "d.idle", BIG, 4, PALE, INK, G3)
  local tw = textWidth(SMALL, TAG, 1)
  text(c, floor((240 - tw) / 2), 115, TAG, SMALL, 1, MID, INK)
  saveSource(c, "card.aseprite")
  savePng(c, 5, "card-1200x630.png")
end
-- ---- d2d launcher banner 1120x135 (224x27 at 5x), transparent ------------
do
  local c = canvas(224, 27)
  mark(c, 2, 1, 25, { plate = true })
  text(c, 34, 3, "d.idle", BIG, 3, PALE, INK, G3)
  saveSource(c, "hero.aseprite")
  savePng(c, 5, "hero-1120x135.png")
end

-- ---- palette (GIMP .gpl, readable by Aseprite) ---------------------------
do
  local f = assert(io.open(PAL .. "/didle.gpl", "w"))
  f:write("GIMP Palette\nName: didle\nColumns: 5\n# d.idle brand palette: 15 colours from internal/watch/web/tokens.css; original, CC0 1.0\n")
  local names = { "ink", "plate", "plate-hi", "border", "dim", "mid", "light", "pale", "gold-0", "gold-1", "gold-2", "gold-3", "gold-4", "deep", "plate-2" }
  for i, col in ipairs(P) do f:write(string.format("%3d %3d %3d  %s\n", col[1], col[2], col[3], names[i])) end
  f:close()
end
