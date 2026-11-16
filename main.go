package main

import (
	"os"

	"fyne.io/fyne/v2/app"

	"github.com/yinrg/trustanchor/internal/truststore"
	"github.com/yinrg/trustanchor/internal/ui"
)

func main() {
	if handled, code := truststore.RunCLI(os.Args[1:]); handled {
		os.Exit(code)
	}
	ui.Run(app.NewWithID(ui.AppID))
}
