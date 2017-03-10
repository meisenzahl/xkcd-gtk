package main

import (
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	"log"
)

const appId = "com.ryankoesters.xkcd-gtk"

type Application struct {
	GtkApp *gtk.Application
}

func NewApplication() (*Application, error) {
	var app Application
	var err error

	app.GtkApp, err = gtk.ApplicationNew(appId, glib.APPLICATION_FLAGS_NONE)
	if err != nil {
		return nil, err
	}

	app.GtkApp.Connect("activate", app.Activate)

	return &app, nil
}

func (a *Application) Run(args []string) int {
	return a.GtkApp.Run(args)
}

func (a *Application) Activate() {
	viewer, err := New()
	if err != nil {
		log.Fatal(err)
	}
	a.GtkApp.AddWindow(viewer.win)
	viewer.win.ShowAll()

	go func() {
		viewer.SetComic(*number)
	}()
}