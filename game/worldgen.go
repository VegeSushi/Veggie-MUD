package game

import (
	"database/sql"
	"fmt"
	"math/rand"
	"strings"
	"time"
)

const LevelWidth = 25
const LevelHeight = 25

func init() {
	rand.Seed(time.Now().UnixNano())
}

func GetOrGenerateLevel(w *World, z int) {
	if _, exists := w.Levels[z]; exists {
		return
	}

	w.Levels[z] = make(map[string]rune)
	var mapData string
	err := w.DB.QueryRow("SELECT map_data FROM levels WHERE z=?", z).Scan(&mapData)

	if err == sql.ErrNoRows {
		mapData = generateCavern()
		w.DB.Exec("INSERT INTO levels (z, map_data) VALUES (?, ?)", z, mapData)

		// Load into memory first
		loadMapIntoMemory(w, z, mapData)

		// NEW: Populate the level with initial Goblins and Chests
		PopulateLevel(w, z)
	} else {
		loadMapIntoMemory(w, z, mapData)
	}
}

func loadMapIntoMemory(w *World, z int, data string) {
	lines := strings.Split(data, "\n")
	for y, row := range lines {
		for x, char := range row {
			if char != '.' {
				w.Levels[z][fmt.Sprintf("%d,%d", x, y)] = char
			}
		}
	}
}

func PopulateLevel(w *World, z int) {
	// Spawn 3 initial Chests
	for i := 0; i < 3; i++ {
		x, y := FindSafeSpawn(w, z)
		entity := w.CreateEntity()
		w.Positions[entity] = &Position{X: x, Y: y, Z: z}
		w.Renderables[entity] = &Renderable{Char: 'C'}
		w.Loot[entity] = &Loot{Items: []string{RandomLoot()}, Coins: rand.Intn(15) + 5}
	}
	// Goblins will be handled by the SpawnSystem in the main loop
}

func RandomLoot() string {
	items := []string{"Healing Potion", "Dagger", "Shortsword", "Leather Armor"}
	return items[rand.Intn(len(items))]
}

func generateCavern() string {
	grid := make([][]rune, LevelHeight)
	for y := 0; y < LevelHeight; y++ {
		grid[y] = make([]rune, LevelWidth)
		for x := 0; x < LevelWidth; x++ {
			if rand.Float32() < 0.45 {
				grid[y][x] = '#'
			} else {
				grid[y][x] = '.'
			}
		}
	}

	for i := 0; i < 4; i++ {
		newGrid := make([][]rune, LevelHeight)
		for y := 0; y < LevelHeight; y++ {
			newGrid[y] = make([]rune, LevelWidth)
			for x := 0; x < LevelWidth; x++ {
				if countWalls(grid, x, y) >= 5 {
					newGrid[y][x] = '#'
				} else {
					newGrid[y][x] = '.'
				}
			}
		}
		grid = newGrid
	}

	for y := 0; y < LevelHeight; y++ {
		grid[y][0] = '#'
		grid[y][LevelWidth-1] = '#'
	}
	for x := 0; x < LevelWidth; x++ {
		grid[0][x] = '#'
		grid[LevelHeight-1][x] = '#'
	}

	for {
		sx, sy := rand.Intn(LevelWidth-2)+1, rand.Intn(LevelHeight-2)+1
		if grid[sy][sx] == '.' {
			grid[sy][sx] = '>'
			break
		}
	}

	var sb strings.Builder
	for y := 0; y < LevelHeight; y++ {
		sb.WriteString(string(grid[y]))
		if y < LevelHeight-1 {
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

func countWalls(grid [][]rune, cx, cy int) int {
	count := 0
	for y := cy - 1; y <= cy+1; y++ {
		for x := cx - 1; x <= cx+1; x++ {
			if x < 0 || x >= LevelWidth || y < 0 || y >= LevelHeight {
				count++
			} else if grid[y][x] == '#' {
				count++
			}
		}
	}
	return count
}
