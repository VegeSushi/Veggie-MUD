package game

type ItemDef struct {
	Name         string
	Type         string // "consumable", "weapon", "armor"
	HealAmount   int
	AttackBonus  int
	DefenseBonus int
}

var ItemRegistry = map[string]ItemDef{
	"Healing Potion": {
		Name:       "Healing Potion",
		Type:       "consumable",
		HealAmount: 5,
	},
	"Dagger": {
		Name:        "Dagger",
		Type:        "weapon",
		AttackBonus: 2,
	},
	"Shortsword": {
		Name:        "Shortsword",
		Type:        "weapon",
		AttackBonus: 4,
	},
	"Leather Armor": {
		Name:         "Leather Armor",
		Type:         "armor",
		DefenseBonus: 1,
	},
}
