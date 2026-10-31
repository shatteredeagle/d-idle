package leveler

import "d2lod-bot/internal/mem"

func mem32(src mem.Source, addr uint32) (uint32, error) { return mem.U32(src, addr) }
