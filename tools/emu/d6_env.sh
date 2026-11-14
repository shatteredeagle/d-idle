# decomp6 (worker host): environment for the emulator oracles on the scratch tree
export PATH=$HOME/.local/bin:$PATH D2_MPQ_DIR=${D2_EMU_DIR:-$HOME/d2d-lab/emu}/mpq D2_PATCH_INSTALLER=${D2_EMU_DIR:-$HOME/d2d-lab/emu}/patch/LODPatch_114d.exe
export T=$HOME/d2d-lab/wt-decomp6 B=$HOME/d2d-lab/build-decomp6 O=$HOME/d2d-lab/d6
mkdir -p $O
