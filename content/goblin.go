package content

import (
	"veggie-mud/game"
)

func init() {
	game.RegisterNPC("Goblin", func(w *game.World, x, y, z int) game.Entity {
		entity := w.CreateEntity()
		w.Positions[entity] = &game.Position{X: x, Y: y, Z: z}
		w.Renderables[entity] = &game.Renderable{Char: 'g'}
		return entity
	})
}