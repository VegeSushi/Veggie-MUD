package game

import (
	"database/sql"
	"fmt"
	"math/rand"
	"strings"
	"time"
)

const LevelWidth = 50
const LevelHeight = 50

func init() {
	rand.Seed(time.Now().UnixNano())
}

// GetOrGenerateLevel checks if a Z-level is in memory, then DB, or generates it.
func GetOrGenerateLevel(w *World, z int) {
	if _, exists := w.Levels[z]; exists {
		return // Already loaded
	}

	w.Levels[z] = make(map[string]rune)
	var mapData string
	err := w.DB.QueryRow("SELECT map_data FROM levels WHERE z=?", z).Scan(&mapData)

	if err == sql.ErrNoRows {
		mapData = generateCavern()
		w.DB.Exec("INSERT INTO levels (z, map_data) VALUES (?, ?)", z, mapData)
	}

	// Load map data into memory (only store non-floor tiles to save RAM)
	lines := strings.Split(mapData, "\n")
	for y, row := range lines {
		for x, char := range row {
			if char != '.' {
				w.Levels[z][fmt.Sprintf("%d,%d", x, y)] = char
			}
		}
	}
}

// generateCavern uses Cellular Automata to make organic caves
func generateCavern() string {
	grid := make([][]rune, LevelHeight)
	
	// 1. Scatter random noise
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

	// 2. Smooth the noise (Cellular Automata)
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

	// 3. Add solid borders
	for y := 0; y < LevelHeight; y++ { grid[y][0] = '#'; grid[y][LevelWidth-1] = '#' }
	for x := 0; x < LevelWidth; x++ { grid[0][x] = '#'; grid[LevelHeight-1][x] = '#' }
	
	// 4. Place one Down-Staircase (>)
	for {
		sx, sy := rand.Intn(LevelWidth-2)+1, rand.Intn(LevelHeight-2)+1
		if grid[sy][sx] == '.' {
			grid[sy][sx] = '>'
			break
		}
	}

	// Convert grid to a flat string for database storage
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
	for y := cy - 1; y <= cy + 1; y++ {
		for x := cx - 1; x <= cx + 1; x++ {
			if x < 0 || x >= LevelWidth || y < 0 || y >= LevelHeight {
				count++ // Out of bounds counts as a wall
			} else if grid[y][x] == '#' {
				count++
			}
		}
	}
	return count
}