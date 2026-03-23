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
	DB_ID   int
}

// --- The World (ECS Manager) ---
type Entity uint64

type World struct {
	NextID      Entity
	Positions   map[Entity]*Position
	Renderables map[Entity]*Renderable
	Players     map[Entity]*PlayerControl
	Levels      map[int]map[string]rune // Z -> "X,Y" -> rune
	DB          *sql.DB
}

func NewWorld(db *sql.DB) *World {
	return &World{
		NextID:      1,
		Positions:   make(map[Entity]*Position),
		Renderables: make(map[Entity]*Renderable),
		Players:     make(map[Entity]*PlayerControl),
		Levels:      make(map[int]map[string]rune), // Initialize it
		DB:          db,
	}
}

func (w *World) CreateEntity() Entity {
	id := w.NextID
	w.NextID++
	return id
}

// --- Modding Registry ---
var NPCRegistry = make(map[string]func(w *World, x, y, z int) Entity)

func RegisterNPC(name string, builder func(w *World, x, y, z int) Entity) {
	NPCRegistry[name] = builder
}