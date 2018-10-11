package main

import (
	"fmt"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	"github.com/rkoesters/xkcd"
	"github.com/skratchdot/open-golang/open"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"sync"
)

// Window is the main application window.
type Window struct {
	window *gtk.ApplicationWindow
	state  WindowState

	comic      *xkcd.Comic
	comicMutex *sync.Mutex

	actions map[string]*glib.SimpleAction
	accels  *gtk.AccelGroup

	header *gtk.HeaderBar
	image  *gtk.Image

	previous *gtk.Button
	next     *gtk.Button
	random   *gtk.Button

	search        *gtk.MenuButton
	searchEntry   *gtk.SearchEntry
	searchResults *gtk.Box

	menu *gtk.MenuButton

	properties *PropertiesDialog
}

// NewWindow creates a new xkcd viewer window.
func NewWindow(app *Application) (*Window, error) {
	var err error

	win := new(Window)

	win.comic = &xkcd.Comic{Title: appName}
	win.comicMutex = new(sync.Mutex)

	win.window, err = gtk.ApplicationWindowNew(app.application)
	if err != nil {
		return nil, err
	}

	// Initialize our window actions.
	actionFuncs := map[string]interface{}{
		"explain":         win.Explain,
		"goto-newest":     win.GotoNewest,
		"next-comic":      win.NextComic,
		"open-link":       win.OpenLink,
		"previous-comic":  win.PreviousComic,
		"random-comic":    win.RandomComic,
		"show-properties": win.ShowProperties,
	}

	win.actions = make(map[string]*glib.SimpleAction)
	for name, function := range actionFuncs {
		action := glib.SimpleActionNew(name, nil)
		action.Connect("activate", function)

		win.actions[name] = action
		win.window.AddAction(action)
	}

	// Initialize our window accelerators.
	win.accels, err = gtk.AccelGroupNew()
	if err != nil {
		return nil, err
	}
	win.window.AddAccelGroup(win.accels)

	addAccel := func(widget *gtk.Button, accel string) {
		key, mods := gtk.AcceleratorParse(accel)
		if key == 0 || mods == 0 {
			panic("AddAccel bad accelerator")
		}

		widget.AddAccelerator("activate", win.accels, key, mods, gtk.ACCEL_VISIBLE)
	}

	// If the gtk theme changes, we might want to adjust our styling.
	win.window.Window.Connect("style-updated", win.StyleUpdated)

	// Create HeaderBar
	win.header, err = gtk.HeaderBarNew()
	if err != nil {
		return nil, err
	}
	win.header.SetTitle(appName)
	win.header.SetShowCloseButton(true)

	navBox, err := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	if err != nil {
		return nil, err
	}
	navBoxStyleContext, err := navBox.GetStyleContext()
	if err != nil {
		return nil, err
	}
	navBoxStyleContext.AddClass("linked")

	win.previous, err = gtk.ButtonNew()
	if err != nil {
		return nil, err
	}
	win.previous.SetTooltipText("Go to the previous comic")
	win.previous.SetProperty("action-name", "win.previous-comic")
	addAccel(win.previous, "<Control>p")
	navBox.Add(win.previous)

	win.next, err = gtk.ButtonNew()
	if err != nil {
		return nil, err
	}
	win.next.SetTooltipText("Go to the next comic")
	win.next.SetProperty("action-name", "win.next-comic")
	addAccel(win.next, "<Control>n")
	navBox.Add(win.next)

	win.header.PackStart(navBox)

	win.random, err = gtk.ButtonNewWithLabel("Random")
	if err != nil {
		return nil, err
	}
	win.random.SetTooltipText("Go to a random comic")
	win.random.SetProperty("action-name", "win.random-comic")
	addAccel(win.random, "<Control>r")
	win.header.PackStart(win.random)

	// Create the menu
	win.menu, err = gtk.MenuButtonNew()
	if err != nil {
		return nil, err
	}
	win.menu.SetTooltipText("Menu")

	menu := glib.MenuNew()

	menuSection1 := glib.MenuNew()
	menuSection1.Append("Go to Newest Comic", "win.goto-newest")
	menuSection1.Append("Open Link", "win.open-link")
	menuSection1.Append("Explain", "win.explain")
	menuSection1.Append("Properties", "win.show-properties")
	menu.AppendSectionWithoutLabel(&menuSection1.MenuModel)

	if !app.application.PrefersAppMenu() {
		menuSection2 := glib.MenuNew()
		menuSection2.Append("New Window", "app.new-window")
		menu.AppendSectionWithoutLabel(&menuSection2.MenuModel)

		menuSection3 := glib.MenuNew()
		menuSection3.Append("what if?", "app.open-what-if")
		menuSection3.Append("xkcd blog", "app.open-blog")
		menuSection3.Append("xkcd store", "app.open-store")
		menu.AppendSectionWithoutLabel(&menuSection3.MenuModel)

		menuSection4 := glib.MenuNew()
		menuSection4.Append("About "+appName, "app.show-about")
		menu.AppendSectionWithoutLabel(&menuSection4.MenuModel)
	}

	win.menu.SetMenuModel(&menu.MenuModel)
	win.header.PackEnd(win.menu)

	// Create the search menu
	win.search, err = gtk.MenuButtonNew()
	if err != nil {
		return nil, err
	}
	win.search.SetTooltipText("Search")
	addAccel(&win.search.Button, "<Control>f")
	win.header.PackEnd(win.search)

	searchPopover, err := gtk.PopoverNew(win.search)
	if err != nil {
		return nil, err
	}
	win.search.SetPopover(searchPopover)

	box, err := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 10)
	if err != nil {
		return nil, err
	}
	box.SetMarginTop(12)
	box.SetMarginBottom(12)
	box.SetMarginStart(12)
	box.SetMarginEnd(12)
	win.searchEntry, err = gtk.SearchEntryNew()
	if err != nil {
		return nil, err
	}
	win.searchEntry.Connect("search-changed", win.Search)
	box.Add(win.searchEntry)
	scwin, err := gtk.ScrolledWindowNew(nil, nil)
	if err != nil {
		return nil, err
	}
	box.Add(scwin)
	win.searchResults, err = gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	if err != nil {
		return nil, err
	}
	scwin.Add(win.searchResults)
	scwin.SetSizeRequest(375, 250)
	win.loadSearchResults(nil)
	box.ShowAll()
	searchPopover.Add(box)

	win.header.ShowAll()
	win.window.SetTitlebar(win.header)

	// Create main part of window.
	imageScroller, err := gtk.ScrolledWindowNew(nil, nil)
	if err != nil {
		return nil, err
	}
	imageScroller.SetSizeRequest(400, 300)

	imageContext, err := imageScroller.GetStyleContext()
	if err != nil {
		return nil, err
	}
	imageContext.AddClass("comic-container")

	win.image, err = gtk.ImageNew()
	if err != nil {
		return nil, err
	}
	imageScroller.Add(win.image)
	imageScroller.ShowAll()
	win.window.Add(imageScroller)

	// Recall our window state.
	win.state.ReadFile(filepath.Join(CacheDir(), "state"))
	if win.state.Maximized {
		win.window.Maximize()
	} else {
		win.window.Resize(win.state.Width, win.state.Height)
		if win.state.PositionX != 0 && win.state.PositionY != 0 {
			win.window.Move(win.state.PositionX, win.state.PositionY)
		}
	}
	if win.state.PropertiesVisible {
		if win.properties == nil {
			win.properties, err = NewPropertiesDialog(win)
			if err != nil {
				return nil, err
			}
		}
		win.properties.Present()
	}
	win.SetComic(win.state.ComicNumber)

	// If the gtk window state changes, we want to update our internal
	// window state.
	win.window.Window.Connect("size-allocate", win.StateChanged)
	win.window.Window.Connect("window-state-event", win.StateChanged)

	// If the window is closed, we want to write our state to disk.
	win.window.Window.Connect("delete-event", win.SaveState)

	return win, nil
}

// PreviousComic sets the current comic to the previous comic.
func (win *Window) PreviousComic() {
	win.SetComic(win.comic.Num - 1)
}

// NextComic sets the current comic to the next comic.
func (win *Window) NextComic() {
	win.SetComic(win.comic.Num + 1)
}

// RandomComic sets the current comic to a random comic.
func (win *Window) RandomComic() {
	newestComic, _ := GetNewestComicInfo()
	if newestComic.Num <= 0 {
		win.SetComic(newestComic.Num)
	} else {
		win.SetComic(rand.Intn(newestComic.Num) + 1)
	}
}

// SetComic sets the current comic to the given comic.
func (win *Window) SetComic(n int) {
	// Make it clear that we are loading a comic.
	win.header.SetTitle("Loading comic...")
	win.header.SetSubtitle(strconv.Itoa(n))
	win.updateNextPreviousButtonStatus()
	win.state.ComicNumber = n

	go func() {
		var err error

		// Make sure we are the only ones changing win.comic.
		win.comicMutex.Lock()
		defer win.comicMutex.Unlock()

		win.comic, err = GetComicInfo(n)
		if err != nil {
			log.Printf("error downloading comic info: %v", n)
		} else {
			_, err = os.Stat(getComicImagePath(n))
			if os.IsNotExist(err) {
				err = DownloadComicImage(n)
				if err != nil {
					// We can be sneaky, we use SafeTitle for window
					// title, but we can leave Title alone so the
					// properties dialog can still be correct.
					win.comic.SafeTitle = "Connect to the internet to download comic image"
				}
			} else if err != nil {
				log.Print(err)
			}
		}

		// Add the DisplayComic function to the event loop so our UI
		// gets updated with the new comic.
		glib.IdleAdd(win.DisplayComic)
	}()
}

// DisplayComic updates the UI to show the contents of win.comic
func (win *Window) DisplayComic() {
	win.header.SetTitle(win.comic.SafeTitle)
	win.header.SetSubtitle(strconv.Itoa(win.comic.Num))
	win.image.SetFromFile(getComicImagePath(win.comic.Num))
	win.image.SetTooltipText(win.comic.Alt)
	win.updateNextPreviousButtonStatus()

	// If the comic has a link, lets give the option of visiting it.
	if win.comic.Link == "" {
		win.actions["open-link"].SetEnabled(false)
	} else {
		win.actions["open-link"].SetEnabled(true)
	}

	if win.properties != nil {
		win.properties.Update()
	}
}

func (win *Window) updateNextPreviousButtonStatus() {
	// Enable/disable previous button.
	if win.comic.Num > 1 {
		win.actions["previous-comic"].SetEnabled(true)
	} else {
		win.actions["previous-comic"].SetEnabled(false)
	}

	// Enable/disable next button.
	newest, _ := GetNewestComicInfo()
	if win.comic.Num < newest.Num {
		win.actions["next-comic"].SetEnabled(true)
	} else {
		win.actions["next-comic"].SetEnabled(false)
	}
}

// ShowProperties presents the properties dialog to the user. If the
// dialog doesn't exist yet, we create it.
func (win *Window) ShowProperties() {
	var err error
	if win.properties == nil {
		win.properties, err = NewPropertiesDialog(win)
		if err != nil {
			log.Print(err)
			return
		}
	}
	win.properties.Present()
}

// GotoNewest checks for a new comic and then shows the newest comic to
// the user.
func (win *Window) GotoNewest() {
	// Make it clear that we are checking for a new comic.
	win.header.SetTitle("Checking for new comic...")

	// Force GetNewestComicInfo to check for a new comic.
	cachedNewestComic = nil
	newestComic, err := GetNewestComicInfo()
	if err != nil {
		log.Print(err)
	}
	win.SetComic(newestComic.Num)
}

// Explain opens a link to explainxkcd.com in the user's web browser.
func (win *Window) Explain() {
	err := open.Start(fmt.Sprintf("https://www.explainxkcd.com/%v/", win.comic.Num))
	if err != nil {
		log.Print(err)
	}
}

// OpenLink opens the comic's Link in the user's web browser.
func (win *Window) OpenLink() {
	err := open.Start(win.comic.Link)
	if err != nil {
		log.Print(err)
	}
}
