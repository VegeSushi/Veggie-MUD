package game

import (
	"database/sql"
	"net"
)

// --- Components ---
type Position struct{ X, Y, Z int }
type Renderable struct{ Char rune }
type PlayerControl struct {
	Conn    net.Conn
	Name    string
	NextCmd string
	LogMsg  string 
	DB_ID   int
}
type CombatStats struct {
	HP, MaxHP, Attack, Defense int
}
type CombatState struct {
	Target Entity 
}
type Inventory struct {
	Items []string 
}

// Loot represents items sitting on the ground (Chests or Bags)
type Loot struct {
	Items []string
}

// --- The World (ECS Manager) ---
type Entity uint64

type World struct {
	NextID      Entity
	TickCount   uint64 
	Positions   map[Entity]*Position
	Renderables map[Entity]*Renderable
	Players     map[Entity]*PlayerControl
	Stats       map[Entity]*CombatStats
	Combat      map[Entity]*CombatState
	Inventories map[Entity]*Inventory
	Loot        map[Entity]*Loot // Added this back
	Levels      map[int]map[string]rune
	DB          *sql.DB
}

func NewWorld(db *sql.DB) *World {
	return &World{
		NextID:      1,
		Positions:   make(map[Entity]*Position),
		Renderables: make(map[Entity]*Renderable),
		Players:     make(map[Entity]*PlayerControl),
		Stats:       make(map[Entity]*CombatStats),
		Combat:      make(map[Entity]*CombatState),
		Inventories: make(map[Entity]*Inventory),
		Loot:        make(map[Entity]*Loot), // Initialized
		Levels:      make(map[int]map[string]rune),
		DB:          db,
	}
}

func (w *World) CreateEntity() Entity {
	id := w.NextID
	w.NextID++
	return id
}

var NPCRegistry = make(map[string]func(w *World, x, y, z int) Entity)

func RegisterNPC(name string, builder func(w *World, x, y, z int) Entity) {
	NPCRegistry[name] = builder
}