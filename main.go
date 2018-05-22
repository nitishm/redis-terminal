package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/gdamore/tcell"
	"github.com/gomodule/redigo/redis"
	"github.com/rivo/tview"
)

const (
	finderPage = "*finder*" // The name of the Finder page.
)

var (
	app         *tview.Application // The tview application.
	pages       *tview.Pages       // The application pages.
	finderFocus tview.Primitive    // The primitive in the Finder that last had focus.
)

// Main entry point.
func main() {
	conn, err := redis.Dial("tcp", "localhost:6379")
	if err != nil {
		log.Fatalf("Failed to connect to redis-server @ %s", "localhost:6379")
	}

	// Start the application.
	app = tview.NewApplication()

	scan(conn)

	if err := app.Run(); err != nil {
		fmt.Printf("Error running application: %s\n", err)
	}
}

// Sets up a "Finder" used to navigate the databases, tables, and columns.
func scan(conn redis.Conn) {
	// Create the basic objects.
	keys := tview.NewList().ShowSecondaryText(false)
	keys.SetBorder(true).SetTitle("Keys")
	columns := tview.NewTable().SetBorders(true)
	columns.SetBorder(true).SetTitle("Columns").SetBorderColor(tcell.ColorPurple)

	// Create the layout.
	flex := tview.NewFlex().
		AddItem(keys, 0, 1, true).
		AddItem(columns, 0, 1, false)

	list, err := redis.Strings(conn.Do("KEYS", "*"))
	if err != nil {
		return
	}

	for _, item := range list {
		keys.AddItem(item, "", 0, nil)
	}

	keys.SetChangedFunc(func(i int, pname string, sname string, s rune) {
		columns.Clear()
		columns.SetCell(0, 0, &tview.TableCell{Text: "Key", Align: tview.AlignCenter, Color: tcell.ColorYellow})
		columns.SetCell(0, 1, &tview.TableCell{Text: "Value", Align: tview.AlignCenter, Color: tcell.ColorYellow})
		columns.SetCell(0, 2, &tview.TableCell{Text: "Type", Align: tview.AlignCenter, Color: tcell.ColorYellow})
		tp, err := redis.String(conn.Do("TYPE", pname))
		if err != nil {
			return
		}

		v, err := getValueJSON(conn, tp, pname)
		if err != nil {
			return
		}

		columns.SetCell(1, 0, &tview.TableCell{Text: pname, Align: tview.AlignCenter, Color: tcell.ColorWhite})
		columns.SetCell(1, 1, &tview.TableCell{Text: *v, Align: tview.AlignLeft, Color: tcell.ColorBlue, Expansion: 0})
		columns.SetCell(1, 2, &tview.TableCell{Text: strings.ToUpper(tp), Align: tview.AlignCenter, Color: tcell.ColorRed})
	})

	keys.SetCurrentItem(0)

	keys.SetSelectedFunc(func(i int, pname string, sname string, s rune) {
		details(conn, pname)
	})

	keys.SetDoneFunc(func() {
		app.Stop()
	})

	pages = tview.NewPages().
		AddPage(finderPage, flex, true, true)
	app.SetRoot(pages, true)
}

func details(conn redis.Conn, key string) {
	finderFocus = app.GetFocus()

	table := tview.NewTable().
		SetFixed(1, 0).
		SetBorders(true).
		SetBordersColor(tcell.ColorYellow)
	frame := tview.NewFrame(table).
		SetBorders(0, 0, 0, 0, 0, 0)
	frame.SetBorder(true).
		SetTitle(fmt.Sprintf(`["%s"]`, key))

	tp, err := redis.String(conn.Do("TYPE", key))
	if err != nil {
		return
	}

	switch tp {
	case "string", "list", "set":
		v, err := getValueJSON(conn, tp, key)
		if err != nil {
			return
		}
		frame.AddText(*v, true, tview.AlignLeft, tcell.ColorYellow)
	case "hash":
		v, err := getValue(conn, tp, key)
		if err != nil {
			return
		}

		m := v.(map[string]string)
		i := 0
		for mk, mv := range m {
			table.SetCell(0, i, &tview.TableCell{Text: strings.ToUpper(mk), Align: tview.AlignCenter, Color: tcell.ColorRed})
			table.SetCell(1, i, &tview.TableCell{Text: mv, Align: tview.AlignCenter, Color: tcell.ColorWhite})
			i = i + 1
		}
	}

	table.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEscape, tcell.KeyEnter, tcell.KeyBackspace2:
			// Go back to Finder.
			pages.SwitchToPage(finderPage)
			if finderFocus != nil {
				app.SetFocus(finderFocus)
			}
		}
	})

	pages.AddPage(key, frame, true, true)
}

func getValue(conn redis.Conn, t string, k string) (v interface{}, err error) {
	switch t {
	case "hash":
		return redis.StringMap(conn.Do("HGETALL", k))
	case "list":
		return redis.Strings(conn.Do("LRANGE", k, 0, -1))
	case "string":
		return redis.String(conn.Do("GET", k))
	}
	return
}

func getValueJSON(conn redis.Conn, t string, k string) (v *string, err error) {
	v = new(string)

	value, err := getValue(conn, t, k)
	if err != nil {
		return
	}

	b, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	*v = string(b)

	return
}
