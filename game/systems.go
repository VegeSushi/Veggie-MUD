package game

import (
	"fmt"
	"strings"
)

// ProcessCommands handles movement and quitting
func ProcessCommands(w *World) {
	for entity, player := range w.Players {
		cmd := strings.ToLower(strings.TrimSpace(player.NextCmd))
		player.NextCmd = ""

		if cmd == "" { continue }

		pos, hasPos := w.Positions[entity]
		if !hasPos { continue }

		newX, newY := pos.X, pos.Y

		switch cmd {
		case "w": newY--
		case "s": newY++
		case "d": newX++
		case "a": newX--
		case ">", "down":
			// Check if standing on stairs
			if w.Levels[pos.Z][fmt.Sprintf("%d,%d", pos.X, pos.Y)] == '>' {
				pos.Z++
				GetOrGenerateLevel(w, pos.Z)
				
				// THIS IS THE LINE TO FIX: Change findSafeSpawn to FindSafeSpawn
				pos.X, pos.Y = FindSafeSpawn(w, pos.Z) 
				
				player.Conn.Write([]byte("\r\nYou descend deeper into the darkness...\r\n"))
			} else {
				player.Conn.Write([]byte("\r\nThere is no way down here.\r\n"))
			}
			continue
		case "quit":
			player.Conn.Close() // The defer in main.go handles saving
			continue
		default:
			player.Conn.Write([]byte("\r\nUnknown command.\r\n"))
			continue
		}

		// Collision Detection
		if w.Levels[pos.Z][fmt.Sprintf("%d,%d", newX, newY)] == '#' {
			player.Conn.Write([]byte("\r\nYou bump into a wall.\r\n"))
		} else {
			pos.X, pos.Y = newX, newY // Apply movement
		}
	}
}

// findSafeSpawn looks for a floor tile on a new Z-level
func FindSafeSpawn(w *World, z int) (int, int) {
	for y := 1; y < LevelHeight-1; y++ {
		for x := 1; x < LevelWidth-1; x++ {
			if _, isWall := w.Levels[z][fmt.Sprintf("%d,%d", x, y)]; !isWall {
				return x, y
			}
		}
	}
	return 2, 2 // Fallback
}

func RenderViewport(w *World) {
	for entity, player := range w.Players {
		pPos, hasPos := w.Positions[entity]
		if !hasPos { continue }

		var sb strings.Builder
		
		// \0337 is the highly-compatible VT100 Save Cursor.
		// \033[H moves the server's drawing cursor to the top-left.
		sb.WriteString("\0337\033[H--- The Depths ---\033[K\r\n")

		// Draw 11x11 grid
		for y := pPos.Y - 5; y <= pPos.Y + 5; y++ {
			for x := pPos.X - 5; x <= pPos.X + 5; x++ {
				key := fmt.Sprintf("%d,%d", x, y)
				
				entityFound := false
				for e, ePos := range w.Positions {
					if ePos.X == x && ePos.Y == y && ePos.Z == pPos.Z {
						if rnd, ok := w.Renderables[e]; ok {
							sb.WriteRune(rnd.Char)
							entityFound = true
							break
						}
					}
				}

				if !entityFound {
					if char, exists := w.Levels[pPos.Z][key]; exists {
						sb.WriteRune(char)
					} else {
						if x < 0 || x >= LevelWidth || y < 0 || y >= LevelHeight {
							sb.WriteRune(' ') // Void
						} else {
							sb.WriteRune('.') // Floor
						}
					}
				}
			}
			sb.WriteString("\033[K\r\n") 
		}
		
		sb.WriteString(fmt.Sprintf("Pos: %d,%d, Depth: %d \033[K\r\n", pPos.X, pPos.Y, pPos.Z))
		
		// \0338 is the highly-compatible VT100 Restore Cursor.
		sb.WriteString("\0338")
		
		player.Conn.Write([]byte(sb.String()))
	}
}