package task

import "context"

// WeaponUser is the optional capability for quest weapons (Khalim's Will on the Compelling Orb, the Hell Forge
// Hammer): swap the pack item named by base code or display name into the weapon slot, run fn, swap the old weapon
// back (town.Town.UseWeapon). Steps type-assert Actions to it and fail with KindPrereq when it is missing.
type WeaponUser interface {
	UseWeapon(ctx context.Context, item string, fn func(ctx context.Context) error) error
}
