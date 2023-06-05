package main

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"io/ioutil"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/andybrewer/mack"
	"github.com/asaskevich/govalidator"
	"github.com/fstanis/screenresolution"

	"github.com/deitrix/go-plist"
	"github.com/sqweek/dialog"
)

var root treeNode
var plistType string
var filename string
var r = treeNode{entry: Entry{key: "Root"}}

func ParsePlist(filename string, w fyne.Window, tree *widget.Tree) {
	file, err := os.Open(filename)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer file.Close()

	content, err := ioutil.ReadAll(file)
	if err != nil {
		fmt.Println("Error reading file:", err)
		return
	}
	plistData = plist.OrderedDict{}
	entries = []Entry{}
	root = treeNode{}
	r = treeNode{entry: Entry{key: "Root"}}
	_, err = plist.Unmarshal(content, &plistData)
	if err != nil {
		_, e := plist.Unmarshal(content, &arrayPlist)
		if e != nil {
			fmt.Println("Error parsing dict plist data:", err)
			fmt.Println("Error parsing array plist data:", e)
			mack.Alert("Error", "Failed to parse plist", "critical")
			w.Close()
		} else {
			fmt.Printf("[INFO] Parsed array plist %s\n", filename)
			plistType = "Array"
			for index, value := range arrayPlist {
				key := fmt.Sprintf("%v", index)
				entry := Parse(key, value, []string{key})
				entries = append(entries, entry)
			}
		}
	} else {
		fmt.Printf("[INFO] Parsed dict plist %s\n", filename)
		plistType = "Dictionary"
		for index, key := range plistData.Keys {
			entry := Parse(key, plistData.Values[index], []string{key})
			entries = append(entries, entry)
		}
	}
	root.children = append(root.children, &r)
	for _, e := range entries {
		t := r.AddChild(e)
		if len(e.children) != 0 {
			for _, c := range e.children {
				node := t.AddChild(c)
				if len(c.children) != 0 {
					for _, m := range c.children {
						ParseChildren(node, m)
					}
				}
			}
		}
	}
	tree.Refresh()
	tree.OpenAllBranches()
	w.SetTitle(filename)
	w.SetContent(tree)
}

func main() {

	a := app.New()
	w := a.NewWindow("Qlist Plist Editor")
	for _, a := range os.Args {
		if b, _ := govalidator.IsFilePath(a); b {
			if strings.HasSuffix(a, ".plist") {
				filename = a
			}
		}
	}
	w.SetCloseIntercept(func() {
		if runtime.GOOS == "darwin" {
			if len(plistData.Keys) > 0 {
				response, _ := mack.AlertBox(mack.AlertOptions{
					Title:   "Do you want to save this file?",
					Style:   "critical",
					Buttons: "No, Yes, Cancel",
				})
				if response.Clicked == "No" {
					w.Close()
				}
			} else {
				w.Close()
			}
		} else {
			w.Close()
		}
	})
	tree := widget.NewTree(
		func(tni widget.TreeNodeID) (nodes []widget.TreeNodeID) {
			if tni == "" {
				nodes = root.ChildrenKeys()
			} else {
				node := root.PathToNode(tni)
				if node != nil {
					for _, label := range node.ChildrenKeys() {
						nodes = append(nodes, tni+"\\-\\"+label)
					}
				}
			}
			return
		},
		func(tni widget.TreeNodeID) bool {
			if node := root.PathToNode(tni); node != nil && node.CountChildren() > 0 {
				return true
			}
			return false
		},
		func(b bool) fyne.CanvasObject {
			key := canvas.NewText("Key", theme.TextColor())
			typ := canvas.NewText("Type", theme.TextColor())
			value := canvas.NewText("Value", theme.TextColor())
			return container.New(layout.NewGridLayout(3), key, typ, value)
		},
		func(tni widget.TreeNodeID, b bool, co fyne.CanvasObject) {
			node := root.PathToNode(tni)
			container, _ := co.(*fyne.Container)
			key := container.Objects[0].(*canvas.Text)
			typ := container.Objects[1].(*canvas.Text)
			value := container.Objects[2].(*canvas.Text)
			if node == nil || &node.entry == nil {
				key.Text = "N/A"
				typ.Text = "N/A"
				value.Text = "N/A"

			} else {
				if node.entry.key == "Root" {
					key.Text = "Root"
					typ.Text = plistType
					if len(entries) == 1 {
						value.Text = "1 key/value entries"
					} else {
						value.Text = fmt.Sprintf("%v key/value entries", len(entries))
					}

				} else {
					t, v := GetType(node.entry)
					key.Text = node.entry.key
					typ.Text = t
					value.Text = v.display
				}
			}
		},
	)
	text := canvas.NewText("Please upload a plist file", theme.TextColor())
	text.Alignment = fyne.TextAlignCenter
	text.TextSize = 25
	text.Refresh()

	fileitem := fyne.NewMenuItem("Open", func() {
		filename, err := dialog.File().Filter("Property-List File", "plist").Load()
		if err != nil {
			fmt.Println("Error opening file:", err)
			return
		}
		ParsePlist(filename, w, tree)

	})

	filemenu := fyne.NewMenu("File", fileitem)
	mainmenu := fyne.NewMainMenu(filemenu)
	w.SetMainMenu(mainmenu)
	resolution := screenresolution.GetPrimary()
	w.Resize(fyne.Size{Width: float32(resolution.Width), Height: float32(resolution.Height)})

	if filename == "" {
		w.SetContent(text)
	} else {
		ParsePlist(filename, w, tree)
	}

	w.ShowAndRun()
}
