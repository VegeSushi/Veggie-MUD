package game

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
)

// ProcessCommands handles WASD movement, stairs (>), and picking up items (g)
func ProcessCommands(w *World) {
	for entity, player := range w.Players {
		cmd := strings.ToLower(strings.TrimSpace(player.NextCmd))
		player.NextCmd = ""

		if cmd == "" { continue }

		pos, hasPos := w.Positions[entity]
		if !hasPos { continue }

		newX, newY := pos.X, pos.Y

		if strings.HasPrefix(cmd, "d ") || strings.HasPrefix(cmd, "drop ") {
			parts := strings.SplitN(cmd, " ", 2)
			idx, err := strconv.Atoi(parts[1])
			inv := w.Inventories[entity]
			if err != nil || idx < 0 || idx >= len(inv.Items) {
				player.LogMsg = "Invalid item index."
			} else {
				item := inv.Items[idx]
				inv.Items = append(inv.Items[:idx], inv.Items[idx+1:]...)
				bag := w.CreateEntity()
				w.Positions[bag] = &Position{X: pos.X, Y: pos.Y, Z: pos.Z}
				w.Renderables[bag] = &Renderable{Char: 'b'}
				w.Loot[bag] = &Loot{Items: []string{item}}
				player.LogMsg = fmt.Sprintf("Dropped: %s", item)
			}
			continue
		}

		switch cmd {
		case "w": newY--
		case "s": newY++
		case "d": newX++
		case "a": newX--
		case "g": // GET ITEM / OPEN CHEST
			itemFound := false
			for e, loot := range w.Loot {
				lPos := w.Positions[e]
				if lPos.X == pos.X && lPos.Y == pos.Y && lPos.Z == pos.Z {
					w.Inventories[entity].Items = append(w.Inventories[entity].Items, loot.Items...)
					source := "bag"
					if rnd, ok := w.Renderables[e]; ok && rnd.Char == 'C' { source = "chest" }
					player.LogMsg = fmt.Sprintf("Looted %s: %s", source, strings.Join(loot.Items, ", "))
					// Clean up the chest/bag entity
					delete(w.Positions, e)
					delete(w.Renderables, e)
					delete(w.Loot, e)
					itemFound = true
					break
				}
			}
			if !itemFound { player.LogMsg = "Nothing here to pick up." }
			continue
		case ">":
			if w.Levels[pos.Z][fmt.Sprintf("%d,%d", pos.X, pos.Y)] == '>' {
				pos.Z++
				GetOrGenerateLevel(w, pos.Z)
				pos.X, pos.Y = FindSafeSpawn(w, pos.Z)
				player.LogMsg = "You descend deeper..."
				w.Combat[entity].Target = 0 
			} else {
				player.LogMsg = "There is no way down here."
			}
			continue
		case "quit":
			player.Conn.Close()
			continue
		default:
			player.LogMsg = "Unknown command."
			continue
		}

		// Collision & Combat Bump Check
		bumpedEntity := Entity(0)
		for e, ePos := range w.Positions {
			if ePos.X == newX && ePos.Y == newY && ePos.Z == pos.Z && e != entity {
				bumpedEntity = e
				break
			}
		}

		if bumpedEntity != 0 {
			if _, attackable := w.Stats[bumpedEntity]; attackable {
				_, isAttackerPlayer := w.Players[entity]
				_, isVictimPlayer := w.Players[bumpedEntity]
				if isAttackerPlayer && isVictimPlayer {
					player.LogMsg = "PVP is disabled."
				} else {
					w.Combat[entity].Target = bumpedEntity
					player.LogMsg = "Attacking..."
				}
			}
		} else if w.Levels[pos.Z][fmt.Sprintf("%d,%d", newX, newY)] == '#' {
			player.LogMsg = "Ouch! A wall."
		} else {
			pos.X, pos.Y = newX, newY 
			w.Combat[entity].Target = 0 
		}
	}
}

// SpawnSystem ensures each active level has at least 5 goblins
func SpawnSystem(w *World) {
	// Only run check every ~10 seconds (16 ticks) to save CPU
	if w.TickCount % 16 != 0 { return }

	// Count goblins per level
	counts := make(map[int]int)
	for e, rnd := range w.Renderables {
		if rnd.Char == 'g' {
			if pos, ok := w.Positions[e]; ok {
				counts[pos.Z]++
			}
		}
	}

	// For every level currently loaded in memory
	for z := range w.Levels {
		for counts[z] < 5 {
			x, y := FindSafeSpawn(w, z)
			if builder, ok := NPCRegistry["Goblin"]; ok {
				builder(w, x, y, z)
				counts[z]++
			}
		}
	}
}

// ProcessCombat handles the 4-tick OSRS hit cycle
func ProcessCombat(w *World) {
	w.TickCount++
	if w.TickCount % 4 != 0 { return }

	for attacker, state := range w.Combat {
		if state.Target == 0 { continue }

		targetStats, ok := w.Stats[state.Target]
		if !ok || targetStats.HP <= 0 {
			state.Target = 0
			continue
		}

		attStats := w.Stats[attacker]
		damage := rand.Intn(attStats.Attack + 1)
		targetStats.HP -= damage

		// Combat Logs
		if p, ok := w.Players[attacker]; ok {
			p.LogMsg = fmt.Sprintf("You hit %d! (Enemy HP: %d)", damage, targetStats.HP)
		}
		if p, ok := w.Players[state.Target]; ok {
			p.LogMsg = fmt.Sprintf("Ouch! Hit for %d! (Your HP: %d)", damage, targetStats.HP)
		}

		// AI Aggro: Non-player entities attack back
		if _, isPlayer := w.Players[state.Target]; !isPlayer {
			if tState, ok := w.Combat[state.Target]; ok && tState.Target == 0 {
				tState.Target = attacker
			}
		}

		// Death Check
		if targetStats.HP <= 0 {
			handleDeath(w, state.Target)
			state.Target = 0
		}
	}
}

func handleDeath(w *World, victim Entity) {
	pos := w.Positions[victim]

	if p, isPlayer := w.Players[victim]; isPlayer {
		p.LogMsg = "Oh dear, you are dead!"
		
		// 50% Drop Logic
		inv := w.Inventories[victim]
		if len(inv.Items) > 0 {
			rand.Shuffle(len(inv.Items), func(i, j int) { inv.Items[i], inv.Items[j] = inv.Items[j], inv.Items[i] })
			dropCount := (len(inv.Items) + 1) / 2
			dropped := inv.Items[:dropCount]
			w.Inventories[victim].Items = inv.Items[dropCount:]

			bag := w.CreateEntity()
			w.Positions[bag] = &Position{X: pos.X, Y: pos.Y, Z: pos.Z}
			w.Renderables[bag] = &Renderable{Char: 'b'}
			w.Loot[bag] = &Loot{Items: dropped}
		}

		// Respawn
		w.Stats[victim].HP = w.Stats[victim].MaxHP
		pos.X, pos.Y = FindSafeSpawn(w, 0)
		pos.Z = 0
	} else {
		// NPC Death
		if inv, hasInv := w.Inventories[victim]; hasInv && len(inv.Items) > 0 {
			bag := w.CreateEntity()
			w.Positions[bag] = &Position{X: pos.X, Y: pos.Y, Z: pos.Z}
			w.Renderables[bag] = &Renderable{Char: 'b'}
			w.Loot[bag] = &Loot{Items: inv.Items}
		}

		delete(w.Positions, victim)
		delete(w.Renderables, victim)
		delete(w.Stats, victim)
		delete(w.Combat, victim)
		delete(w.Inventories, victim)
	}
}

func RenderViewport(w *World) {
	for entity, player := range w.Players {
		pPos, hasPos := w.Positions[entity]
		if !hasPos { continue }

		var sb strings.Builder
		sb.WriteString("\0337\033[H--- VeggieMUD ---\033[K\r\n")

		for y := pPos.Y - 5; y <= pPos.Y + 5; y++ {
			for x := pPos.X - 5; x <= pPos.X + 5; x++ {
				key := fmt.Sprintf("%d,%d", x, y)
				found := false
				for e, ePos := range w.Positions {
					if ePos.X == x && ePos.Y == y && ePos.Z == pPos.Z {
						if rnd, ok := w.Renderables[e]; ok {
							sb.WriteRune(rnd.Char)
							found = true
							break
						}
					}
				}
				if !found {
					if char, exists := w.Levels[pPos.Z][key]; exists {
						sb.WriteRune(char)
					} else {
						if x < 0 || x >= LevelWidth || y < 0 || y >= LevelHeight { sb.WriteRune(' ') } else { sb.WriteRune('.') }
					}
				}
			}
			sb.WriteString("\033[K\r\n")
		}
		
		stats := w.Stats[entity]
		inv := w.Inventories[entity]
		var invStr strings.Builder
		for i, item := range inv.Items {
			invStr.WriteString(fmt.Sprintf("[%d]%s ", i, item))
		}
		sb.WriteString(fmt.Sprintf("HP: %d/%d | Depth: %d | Inv: %s\033[K\r\n", stats.HP, stats.MaxHP, pPos.Z, strings.TrimSpace(invStr.String())))
		sb.WriteString(fmt.Sprintf("Log: %s\033[K\r\n", player.LogMsg))
		sb.WriteString("\0338")
		player.Conn.Write([]byte(sb.String()))
	}
}

func FindSafeSpawn(w *World, z int) (int, int) {
	for i := 0; i < 100; i++ { // Try 100 times to find a random spot
		x, y := rand.Intn(LevelWidth-2)+1, rand.Intn(LevelHeight-2)+1
		if w.Levels[z][fmt.Sprintf("%d,%d", x, y)] != '#' {
			return x, y
		}
	}
	return 2, 2
}

// AISystem manages basic NPC movement toward their combat targets
func AISystem(w *World) {
	if w.TickCount % 2 != 0 { return }

	for entity, state := range w.Combat {
		if _, isPlayer := w.Players[entity]; isPlayer { continue }
		
		pos, hasPos := w.Positions[entity]
		if !hasPos || state.Target == 0 { continue }
		
		targetPos, targetHasPos := w.Positions[state.Target]
		if !targetHasPos || targetPos.Z != pos.Z {
			state.Target = 0
			continue
		}
		
		dx := targetPos.X - pos.X
		dy := targetPos.Y - pos.Y
		distSq := dx*dx + dy*dy
		
		if distSq > 100 { 
			state.Target = 0
			continue
		}
		
		if distSq <= 2 { continue }
		
		stepX, stepY := 0, 0
		if dx > 0 { stepX = 1 } else if dx < 0 { stepX = -1 }
		if dy > 0 { stepY = 1 } else if dy < 0 { stepY = -1 }
		
		newX, newY := pos.X + stepX, pos.Y + stepY
		
		canMove := true
		if newX < 0 || newX >= LevelWidth || newY < 0 || newY >= LevelHeight {
			canMove = false
		} else if w.Levels[pos.Z][fmt.Sprintf("%d,%d", newX, newY)] == '#' {
			canMove = false
		} else {
			for e, ePos := range w.Positions {
				if ePos.X == newX && ePos.Y == newY && ePos.Z == pos.Z && e != entity {
					canMove = false
					break
				}
			}
		}
		
		if canMove {
			pos.X = newX
			pos.Y = newY
		}
	}
}