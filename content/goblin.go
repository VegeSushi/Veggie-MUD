package content

import "veggie-mud/game"

func init() {
	game.RegisterNPC("Goblin", func(w *game.World, x, y, z int) game.Entity {
		entity := w.CreateEntity()
		w.Positions[entity] = &game.Position{X: x, Y: y, Z: z}
		w.Renderables[entity] = &game.Renderable{Char: 'g'}
		
		// Goblin Stats: 7 HP, hits up to 2
		w.Stats[entity] = &game.CombatStats{HP: 7, MaxHP: 7, Attack: 2, Defense: 1}
		w.Combat[entity] = &game.CombatState{}
		w.Inventories[entity] = &game.Inventory{Items: []string{game.RandomLoot()}}
		
		return entity
	})
}