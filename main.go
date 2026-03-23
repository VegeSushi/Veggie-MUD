package main

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"

	_ "veggie-mud/content" // Triggers mod registry
	"veggie-mud/game"
)

func main() {
	// 1. Load Environment Variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, relying on system environment variables.")
	}

	// 2. Connect to MariaDB
	dsn := fmt.Sprintf("%s:%s@tcp(127.0.0.1:%s)/%s",
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_NAME"),
	)
	
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Ensure DB applies schema updates natively
	db.Exec("ALTER TABLE players ADD COLUMN inventory TEXT DEFAULT '[]'")
	db.Exec("ALTER TABLE players ADD COLUMN weapon VARCHAR(255) DEFAULT ''")
	db.Exec("ALTER TABLE players ADD COLUMN armor VARCHAR(255) DEFAULT ''")
	db.Exec("ALTER TABLE players ADD COLUMN coins INT DEFAULT 0")

	world := game.NewWorld(db)
	var mu sync.Mutex

	// 3. Generate the starting level (Z=0) right when the server boots
	game.GetOrGenerateLevel(world, 0)

	// 4. Spawn a test Goblin from the mod registry at 2, 2, 0
	if builder, ok := game.NPCRegistry["Goblin"]; ok {
		builder(world, 2, 2, 0)
	}

	// 5. Start the 600ms OSRS Tick Loop
	go func() {
		ticker := time.NewTicker(600 * time.Millisecond)
		for range ticker.C {
			mu.Lock()
			game.ProcessCommands(world)
			game.SpawnSystem(world)  // <--- ADD THIS LINE
			game.AISystem(world)
			game.ProcessCombat(world)
			game.RenderViewport(world)
			mu.Unlock()
		}
	}()

	// 6. Start listening for Telnet connections
	listener, err := net.Listen("tcp", ":1337")
	if err != nil {
		log.Fatalf("Fatal error: could not listen on port 1337. Is the server already running? Error: %v", err)
	}
	log.Println("VeggieMUD Secure Server running on port 1337...")

	for {
		conn, _ := listener.Accept()
		go handleConnection(conn, world, &mu)
	}
}

func handleConnection(conn net.Conn, w *game.World, mu *sync.Mutex) {
	conn.Write([]byte("Welcome to VeggieMUD! Enter username (or type 'new' to register):\r\n> "))
	scanner := bufio.NewScanner(conn)
	
	state := "AWAITING_NAME"
	var username string
	var playerEntity game.Entity = 0

	// ----------------------------------------------------
	// BULLETPROOF AUTO-SAVE: Runs guaranteed on disconnect
	// ----------------------------------------------------
	defer func() {
		mu.Lock()
		if state == "PLAYING" {
			if pPos, ok := w.Positions[playerEntity]; ok {
				// Safely check for stats before saving
				if stats, hasStats := w.Stats[playerEntity]; hasStats {
					invJSON, _ := json.Marshal(w.Inventories[playerEntity].Items)
					eq := w.Equipment[playerEntity]
					w.DB.Exec("UPDATE players SET x=?, y=?, z=?, hp=?, max_hp=?, inventory=?, weapon=?, armor=?, coins=? WHERE id=?", pPos.X, pPos.Y, pPos.Z, stats.HP, stats.MaxHP, string(invJSON), eq.Weapon, eq.Armor, w.Inventories[playerEntity].Coins, w.Players[playerEntity].DB_ID)
				} else {
					// Fallback just in case stats didn't load
					w.DB.Exec("UPDATE players SET x=?, y=?, z=? WHERE id=?", pPos.X, pPos.Y, pPos.Z, w.Players[playerEntity].DB_ID)
				}
				log.Printf("Auto-saved player %s at %d,%d,%d", username, pPos.X, pPos.Y, pPos.Z)
			}
			
			// Clean up all ECS components so the server doesn't leak memory
			delete(w.Players, playerEntity)
			delete(w.Positions, playerEntity)
			delete(w.Renderables, playerEntity)
			delete(w.Stats, playerEntity)
			delete(w.Combat, playerEntity)
			delete(w.Inventories, playerEntity)
			delete(w.Equipment, playerEntity)
		}
		mu.Unlock() 
		conn.Close() // Close connection after unlocking
	}()
	// ----------------------------------------------------

	// The main input loop
	for scanner.Scan() {
		input := strings.TrimSpace(scanner.Text())

		mu.Lock()
		switch state {
		case "AWAITING_NAME":
			if input == "new" {
				state = "REGISTER_NAME"
				conn.Write([]byte("Choose a username:\r\n> "))
			} else {
				username = input
				state = "AWAITING_PASS"
				conn.Write([]byte("Enter password:\r\n> "))
			}

		case "REGISTER_NAME":
			username = input
			state = "REGISTER_PASS"
			conn.Write([]byte("Choose a password:\r\n> "))

		case "REGISTER_PASS":
			hashedBytes, err := bcrypt.GenerateFromPassword([]byte(input), bcrypt.DefaultCost)
			if err != nil {
				conn.Write([]byte("Server error securing password. Try again:\r\n> "))
				break
			}

			// We don't need to insert HP here because our database defaults new players to 10 HP
			_, err = w.DB.Exec("INSERT INTO players (username, password) VALUES (?, ?)", username, string(hashedBytes))
			if err != nil {
				conn.Write([]byte("Name taken or DB error. Try again:\r\n> "))
				state = "REGISTER_NAME"
			} else {
				conn.Write([]byte("Registered securely! Please log in. Enter username:\r\n> "))
				state = "AWAITING_NAME"
			}

		case "AWAITING_PASS":
			var id, x, y, z, hp, max_hp, coins int
			var storedHash, invJSON, weapon, armor string
			
			// Fetch the stored hash, position, and health
			err := w.DB.QueryRow("SELECT id, password, x, y, z, hp, max_hp, COALESCE(inventory, '[]'), COALESCE(weapon, ''), COALESCE(armor, ''), COALESCE(coins, 0) FROM players WHERE username=?", username).Scan(&id, &storedHash, &x, &y, &z, &hp, &max_hp, &invJSON, &weapon, &armor, &coins)
			
			if err != nil {
				conn.Write([]byte("User not found. Enter username:\r\n> "))
				state = "AWAITING_NAME"
			} else {
				// Compare the provided password against the stored bcrypt hash
				err = bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(input))
				if err != nil {
					conn.Write([]byte("Incorrect password. Enter username:\r\n> "))
					state = "AWAITING_NAME"
				} else {
					// The Wall Spawn Fix: If they are at 0,0, move them to safety
					if x == 0 && y == 0 {
						x, y = game.FindSafeSpawn(w, z)
						w.DB.Exec("UPDATE players SET x=?, y=?, hp=?, max_hp=? WHERE id=?", x, y, hp, max_hp, id)
					}

					// Login Success! Create all ECS Entities
					playerEntity = w.CreateEntity()
					w.Positions[playerEntity] = &game.Position{X: x, Y: y, Z: z}
					w.Renderables[playerEntity] = &game.Renderable{Char: '@'}
					w.Players[playerEntity] = &game.PlayerControl{Conn: conn, Name: username, DB_ID: id, LogMsg: "Welcome to the depths..."}
					
					// Assign Combat and Inventory components
					w.Stats[playerEntity] = &game.CombatStats{HP: hp, MaxHP: max_hp, Attack: 3, Defense: 1}
					w.Combat[playerEntity] = &game.CombatState{}
					w.Inventories[playerEntity] = &game.Inventory{Items: []string{}, Coins: coins}
					json.Unmarshal([]byte(invJSON), &w.Inventories[playerEntity].Items)
					w.Equipment[playerEntity] = &game.Equipment{Weapon: weapon, Armor: armor}
					
					state = "PLAYING"
					
					// Clear screen and lock prompt safely below the map
					welcome := fmt.Sprintf("\033[2J\033[15;1HWelcome back, %s!\r\n> ", username)
					conn.Write([]byte(welcome))
				}
			}

		case "PLAYING":
			// Route the command directly to the player's component
			if p, ok := w.Players[playerEntity]; ok {
				p.NextCmd = input
				
				// Reset the cursor below the map, clear old text, and reprint prompt
				response := fmt.Sprintf("\033[15;1H\033[JAction: %s\r\n> ", input)
				conn.Write([]byte(response))
			}
		}
		mu.Unlock()
	}
}