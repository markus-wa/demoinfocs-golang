# Nade trajectory overview

This example shows how to create a overview of grenade trajectories of a match.

:information_source: Uses radar images from `../_assets/radar` directory.

## Running the example

`go run nade_trajectories.go -demo /path/to/demo > out.jpg`

This will create a JPEG with grenade trajectories of the first five rounds. The reason it doesn't do more trajectories is because the image would look quite cluttered otherwise.

![Resulting map overview with trajectories](https://raw.githubusercontent.com/markus-wa/demoinfocs-golang/master/examples/nade-trajectories/nade_trajectories.jpg)

## Tracking individual grenades

`GrenadeProjectile.UniqueID()` identifies an individual thrown grenade, whereas `Equipment.UniqueID2()` identifies the weapon it was thrown from (several throws can share it).
Grenade events like `HeExplode`, `FlashExplode` or `SmokeStart` reference the projectile they're about via `GrenadeEvent.Projectile`, so you can link a throw to its effect:

```go
p.RegisterEventHandler(func(e events.GrenadeProjectileThrow) {
	fmt.Println("thrown", e.Projectile.UniqueID())
})

p.RegisterEventHandler(func(e events.GrenadeEventIf) {
	// nil if the projectile couldn't be resolved, see GrenadeEvent.Projectile
	if proj := e.Base().Projectile; proj != nil {
		fmt.Printf("%T %d\n", e, proj.UniqueID())
	}
})
```
