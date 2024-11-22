package main

import (
	"database/sql"
	"embed"
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed frontend/dist
var assets embed.FS

type TodoItem struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Completed   bool   `json:"completed"`
	DueDate     string `json:"dueDate"`
}

func GetTodoItems(db *sql.DB) (string, error) {
	rows, err := db.Query("SELECT id, title, description, completed, dueDate FROM todo_items")
	if err != nil {
		return "", fmt.Errorf("query todo items failed: %w", err)
	}
	defer rows.Close()

	var todoItems []TodoItem
	for rows.Next() {
		var item TodoItem
		if err := rows.Scan(&item.ID, &item.Title, &item.Description, &item.Completed, &item.DueDate); err != nil {
			return "", fmt.Errorf("scan todo item failed: %w", err)
		}
		todoItems = append(todoItems, item)
	}

	if err := rows.Err(); err != nil {
		return "", fmt.Errorf("rows iteration error: %w", err)
	}

	todoItemsJSON, err := json.Marshal(todoItems)
	if err != nil {
		return "", fmt.Errorf("marshal todo items failed: %w", err)
	}

	return string(todoItemsJSON), nil
}

func GetSingleItem(db *sql.DB, itemID int) (string, error) {
	var item TodoItem
	err := db.QueryRow("SELECT id, title, description, completed, dueDate FROM todo_items WHERE id = ?", itemID).
		Scan(&item.ID, &item.Title, &item.Description, &item.Completed, &item.DueDate)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("no item found with ID %d", itemID)
		}
		return "", fmt.Errorf("fetch todo item failed: %w", err)
	}

	itemJSON, err := json.Marshal(item)
	if err != nil {
		return "", fmt.Errorf("marshal todo item failed: %w", err)
	}

	return string(itemJSON), nil
}

func CreateTodoItem(db *sql.DB, data any) (string, error) {
	var newItem TodoItem
	if err := json.Unmarshal([]byte(fmt.Sprintf("%v", data)), &newItem); err != nil {
		return "", fmt.Errorf("unmarshal todo item failed: %w", err)
	}

	stmt, err := db.Prepare("INSERT INTO todo_items (title, description, completed, dueDate) VALUES (?, ?, ?, ?)")
	if err != nil {
		return "", fmt.Errorf("prepare insert statement failed: %w", err)
	}
	defer stmt.Close()

	_, err = stmt.Exec(newItem.Title, newItem.Description, newItem.Completed, newItem.DueDate)
	if err != nil {
		return "", fmt.Errorf("execute insert failed: %w", err)
	}

	return GetTodoItems(db)
}

func ToggleTodoItem(db *sql.DB, itemID int) (string, error) {
	var item TodoItem
	err := db.QueryRow("SELECT id, title, description, completed, dueDate FROM todo_items WHERE id = ?", itemID).
		Scan(&item.ID, &item.Title, &item.Description, &item.Completed, &item.DueDate)
	if err != nil {
		return "", fmt.Errorf("fetch todo item failed: %w", err)
	}

	item.Completed = !item.Completed

	stmt, err := db.Prepare("UPDATE todo_items SET completed = ? WHERE id = ?")
	if err != nil {
		return "", fmt.Errorf("prepare update statement failed: %w", err)
	}
	defer stmt.Close()

	_, err = stmt.Exec(item.Completed, item.ID)
	if err != nil {
		return "", fmt.Errorf("execute update failed: %w", err)
	}

	return GetTodoItems(db)
}

func DeleteTodo(db *sql.DB, itemID int) (string, error) {
	stmt, err := db.Prepare("DELETE FROM todo_items WHERE id = ?")
	if err != nil {
		return "", fmt.Errorf("prepare delete statement failed: %w", err)
	}
	defer stmt.Close()

	_, err = stmt.Exec(itemID)
	if err != nil {
		return "", fmt.Errorf("execute delete failed: %w", err)
	}

	return GetTodoItems(db)
}

func EditDescription(db *sql.DB, itemID int, newDescription string) (string, error) {
	var item TodoItem
	err := db.QueryRow("SELECT id, title, description, completed, dueDate FROM todo_items WHERE id = ?", itemID).
		Scan(&item.ID, &item.Title, &item.Description, &item.Completed, &item.DueDate)
	if err != nil {
		return "", fmt.Errorf("fetch todo item failed: %w", err)
	}

	item.Description = newDescription

	stmt, err := db.Prepare("UPDATE todo_items SET description = ? WHERE id = ?")
	if err != nil {
		return "", fmt.Errorf("prepare update statement failed: %w", err)
	}
	defer stmt.Close()

	_, err = stmt.Exec(item.Description, item.ID)
	if err != nil {
		return "", fmt.Errorf("execute update failed: %w", err)
	}

	updatedItemJSON, err := json.Marshal(item)
	if err != nil {
		return "", fmt.Errorf("marshal updated todo item failed: %w", err)
	}

	return string(updatedItemJSON), nil
}

func main() {

	db, err := sql.Open("sqlite3", "todo.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS todo_items (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			title TEXT,
			description TEXT,
			completed BOOLEAN,
			dueDate TEXT
		)
	`)
	if err != nil {
		log.Fatal(err)
	}

	app := application.New(application.Options{
		Name:        "todo",
		Description: "A demo todo app",
		Services:    []application.Service{},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	app.NewWebviewWindowWithOptions(application.WebviewWindowOptions{
		Title: "todo list",
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 50,
			Backdrop:                application.MacBackdropTranslucent,
			TitleBar:                application.MacTitleBarHiddenInset,
		},
		BackgroundColour: application.NewRGB(27, 38, 54),
		URL:              "/",
		Width:            700,
		Height:           700,
	})

	app.OnEvent("requestTodos", func(e *application.CustomEvent) {
		app.Logger.Info("[Go] CustomEvent received", "name", e.Name, "data", e.Data, "sender", e.Sender, "cancelled", e.Cancelled)
		todoItemsJSON, err := GetTodoItems(db)
		if err != nil {
			app.Logger.Error(fmt.Sprintf("Error fetching todos: %v", err))
			return
		}
		app.EmitEvent("responseTodos", todoItemsJSON)
	})

	app.OnEvent("createTodo", func(e *application.CustomEvent) {
		app.Logger.Info("[Go] CustomEvent received", "name", e.Name, "data", e.Data, "sender", e.Sender, "cancelled", e.Cancelled)
		todoItemsJSON, err := CreateTodoItem(db, e.Data)
		if err != nil {
			app.Logger.Error(fmt.Sprintf("Error creating todo: %v", err))
			return
		}
		app.EmitEvent("responseTodos", todoItemsJSON)
		app.EmitEvent("feedbackCreated", "Created!!!")
	})

	app.OnEvent("deleteTodo", func(e *application.CustomEvent) {
		app.Logger.Info("[Go] CustomEvent received", "name", e.Name, "data", e.Data, "sender", e.Sender, "cancelled", e.Cancelled)
		i, _ := strconv.Atoi(fmt.Sprintf("%.f", e.Data))
		todoItemsJSON, err := DeleteTodo(db, i)
		if err != nil {
			app.Logger.Error(fmt.Sprintf("Error deleting todo: %v", err))
			return
		}
		app.EmitEvent("responseTodos", todoItemsJSON)
	})

	app.OnEvent("requestSingleItem", func(e *application.CustomEvent) {
		app.Logger.Info("[Go] CustomEvent received", "name", e.Name, "data", e.Data, "sender", e.Sender, "cancelled", e.Cancelled)
		itemID, err := strconv.Atoi(fmt.Sprintf("%v", e.Data))
		if err != nil {
			app.Logger.Error(fmt.Sprintf("Error converting event data to integer: %v", err))
			return
		}

		singleTodoJSON, err := GetSingleItem(db, itemID)
		if err != nil {
			app.Logger.Error(fmt.Sprintf("Error fetching single item: %v", err))
			return
		}
		app.EmitEvent("responseSingleItem", singleTodoJSON)
	})

	app.OnEvent("toggleTodoCompleted", func(e *application.CustomEvent) {
		app.Logger.Info("[Go] CustomEvent received", "name", e.Name, "data", e.Data, "sender", e.Sender, "cancelled", e.Cancelled)
		i, _ := strconv.Atoi(fmt.Sprintf("%.f", e.Data))
		todoItemsJSON, err := ToggleTodoItem(db, i)
		if err != nil {
			app.Logger.Error(fmt.Sprintf("Error toggling todo: %v", err))
			return
		}
		app.EmitEvent("responseTodos", todoItemsJSON)
	})

	app.OnEvent("editDescription", func(e *application.CustomEvent) {
		app.Logger.Info("[Go] CustomEvent received", "name", e.Name, "data", e.Data, "sender", e.Sender, "cancelled", e.Cancelled)
		data := e.Data.([]interface{})
		itemID, err := strconv.Atoi(fmt.Sprintf("%v", data[0]))
		if err != nil {
			app.Logger.Error(fmt.Sprintf("Error converting item ID to integer: %v", err))
			return
		}
		newDescription := fmt.Sprintf("%v", data[1])

		updatedTodoJSON, err := EditDescription(db, itemID, newDescription)
		if err != nil {
			app.Logger.Error(fmt.Sprintf("Error editing description: %v", err))
			return
		}
		app.EmitEvent("responseSingleItem", updatedTodoJSON)
		app.EmitEvent("feedbackSaved", itemID)

	})

	// to open a new window for the required item
	//	var (
	//		windowPosition int = 1 // Current window position (1-4)
	//	)

	// Define the window positions
	//	var windowPositions = []struct {
	//		x, y int
	//	}{
	//		{350, 50},   // Position 1 (top-left)
	//		{1150, 50},  // Position 2 (top-right)
	//		{350, 550},  // Position 3 (bottom-left)
	//		{1150, 550}, // Position 4 (bottom-right)
	//	}

	app.OnEvent("openItemWindow", func(e *application.CustomEvent) {
		app.Logger.Info("[Go] CustomEvent received", "name", e.Name, "data", e.Data, "sender", e.Sender, "cancelled", e.Cancelled)
		// Convert the event data (item ID) to a string
		itemID := fmt.Sprintf("%.f", e.Data)

		// Construct the URL with the item ID in the path
		url := fmt.Sprintf("/#%s", itemID)

		// Get the current window position
		//		windowX, windowY := windowPositions[windowPosition-1].x, windowPositions[windowPosition-1].y

		app.NewWebviewWindowWithOptions(application.WebviewWindowOptions{
			Title: itemID,
			Mac: application.MacWindow{
				InvisibleTitleBarHeight: 50,
				Backdrop:                application.MacBackdropTranslucent,
				TitleBar:                application.MacTitleBarHiddenInset,
			},
			Width:  500,
			Height: 450,
			//			X:                windowX,
			//			Y:                windowY,
			//Frameless:        true,
			BackgroundColour: application.NewRGB(27, 38, 54),
			URL:              url,
		})

		// Increment the window position state
		//		windowPosition++
		//		if windowPosition > 4 {
		//			windowPosition = 1
		//		}
	})

	app.OnEvent("close-window", func(e *application.CustomEvent) {
		app.Logger.Info("[Go] CustomEvent received", "name", e.Name, "data", e.Data, "sender", e.Sender, "cancelled", e.Cancelled)
		window := app.CurrentWindow()
		window.Close()
	})

	go func() {
		for {
			now := time.Now().Format(time.RFC1123)
			app.EmitEvent("time", now)
			time.Sleep(time.Second)
		}
	}()

	// Run the application. This blocks until the application has been exited.
	err = app.Run()

	// If an error occurred while running the application, log it and exit.
	if err != nil {
		log.Fatal(err)
	}
}
