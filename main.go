package main

import (
	"fmt"
	redisapi "redis-terminal/redis-api"
	"strings"

	"github.com/gdamore/tcell"
	"github.com/renstrom/fuzzysearch/fuzzy"
	"github.com/rivo/tview"
)

var (
	app            *tview.Application
	keyList        *tview.List
	keyFilter      *tview.InputField
	previewText    *tview.TextView
	keyFlexBox     *tview.Flex
	previewFlexBox *tview.Flex
	viewFlexBox    *tview.Flex
	editForm       *tview.Form
)

func main() {
	r, err := redisapi.NewRedis("localhost:6379")
	if err != nil {
		panic(err)
	}

	app = tview.NewApplication()

	pages := tview.NewPages()
	// Main page primitives
	// Components
	keyList = tview.NewList()
	keyFilter = tview.NewInputField()
	previewText = tview.NewTextView()

	// Flex boxes
	keyFlexBox = tview.NewFlex()
	previewFlexBox = tview.NewFlex()

	keyList.
		ShowSecondaryText(false).
		SetTitle("KEYS").
		SetBorder(true).
		SetBorderColor(tcell.ColorSteelBlue)

	keyFilter.
		SetFieldBackgroundColor(tcell.ColorGhostWhite).
		SetFieldTextColor(tcell.ColorBlack)

	previewText.
		SetDynamicColors(true).
		SetRegions(true).
		SetScrollable(true).
		SetTitle("PREVIEW").
		SetBorder(true).
		SetBorderColor(tcell.ColorSteelBlue)

	keyFlexBox.
		SetDirection(tview.FlexRow)
	previewFlexBox.
		SetDirection(tview.FlexRow)

	viewFlexBox := tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(keyFlexBox, 0, 1, true).
		AddItem(previewFlexBox, 0, 4, false)

	pages.AddPage("main", viewFlexBox, true, true)

	// Edit Page primitives
	// Components
	editForm = tview.NewForm()
	editForm.
		SetFieldBackgroundColor(tcell.ColorGray).
		SetFieldTextColor(tcell.ColorWhiteSmoke).
		SetTitle("EDIT").
		SetBorder(true).
		SetBorderColor(tcell.ColorSteelBlue)

	pages.AddPage("edit", editForm, true, false)

	keyFlexBox.AddItem(keyList, 0, 40, true)
	previewFlexBox.AddItem(previewText, 0, 10, false)

	populateKeys := func(pattern string, keys []string) {
		rKeys := []string{}
		keyList.Clear()
		if pattern != "" {
			rKeys, err = r.GetKeys(pattern)
			if err != nil {
				return
			}
		} else {
			rKeys = keys
		}
		for _, k := range rKeys {
			keyList.AddItem(k, "", 0, nil)
		}
	}

	populateEditForm := func(key string) {
		v, err := r.GetValue(key)
		if err != nil {
			return
		}

		switch res := v.(type) {
		case map[string]string:
			for key, value := range res {
				editForm.AddInputField(key, value, len(value)+1, nil, nil)
			}
			break
		case []string:
			break
		case string:
			editForm.AddInputField("", res, len(res)+1, nil, nil)
			break
		default:
			fmt.Printf("Type not supported")
			break
		}
		editForm.
			AddButton("Save", func() {
				pages.SwitchToPage("main")
			}).
			AddButton("Quit", func() {
				pages.SwitchToPage("main")
			}).
			SetButtonBackgroundColor(tcell.ColorSteelBlue).
			SetButtonTextColor(tcell.ColorWhiteSmoke).
			SetButtonsAlign(tview.AlignCenter)
	}

	keyList.SetChangedFunc(func(i int, main string, sec string, sc rune) {
		v, err := redisapi.PrintKey(r, main)
		if err != nil {
			return
		}

		t, err := r.GetType(main)
		if err != nil {
			return
		}

		keyTextBox := fmt.Sprintf("[red]KEY[yellow]: %s\n", main)
		typeTextBox := fmt.Sprintf("[red]TYPE[yellow]: %s\n", strings.ToUpper(t))
		valueTextBox := fmt.Sprintf("%s", v)
		preview := fmt.Sprintf("%s%s%s", keyTextBox, typeTextBox, valueTextBox)
		previewText.SetText(preview)
	})

	keyList.SetSelectedFunc(func(i int, main string, sec string, r rune) {
		editForm.Clear(true)
		populateEditForm(main)
		pages.SwitchToPage("edit")
	})

	keyFilter.SetFinishedFunc(func(key tcell.Key) {
		keys, err := r.GetKeys("*")
		if err != nil {
			return
		}
		res := fuzzy.Find(keyFilter.GetText(), keys)

		populateKeys("", res)
		keyFlexBox.RemoveItem(keyFilter)
		app.SetFocus(keyList)
	})

	editForm.SetCancelFunc(func() {
		populateKeys("*", nil)
		pages.SwitchToPage("main")
	})

	populateKeys("*", nil)

	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlF:
			if viewFlexBox.HasFocus() {
				if !keyFilter.HasFocus() {
					keyFlexBox.AddItem(keyFilter, 0, 1, false)
					keyFilter.SetText("")
					app.SetFocus(keyFilter)
				}
			}
		case tcell.KeyCtrlQ:
			app.Stop()
		}
		return event
	})

	if err := app.SetRoot(pages, true).Run(); err != nil {
		panic(err)
	}
}
