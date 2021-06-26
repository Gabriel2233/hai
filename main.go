package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"

	ui "github.com/jroimartin/gocui"
)

var (
	methods      = []string{"GET", "POST", "DELETE", "PATCH"}
	views        = []string{"input-bar", "method", "req-body", "res-output"}
	activeView   = len(views) - 1
	activeMethod = 0
)

type MethodWidget struct {
	name  string
	label string
	x, y  int
	w, h  int
}

func NewMethodWidget(name, title string, x, y, w, h int) *MethodWidget {
	return &MethodWidget{name: name, x: x, y: y, w: w, h: h}
}

func (iw *MethodWidget) Layout(g *ui.Gui) error {
	v, err := g.SetView(iw.name, iw.x, iw.y, iw.x+iw.w, iw.y+iw.h)
	if err != nil {
		if err != ui.ErrUnknownView {
			return err
		}
		fmt.Fprintf(v, "%s", methods[activeMethod])
	}
	return nil
}

type InputWidget struct {
	name  string
	title string
	x, y  int
	w, h  int
}

func NewInputWidget(name, title string, x, y, w, h int, handlers ...func(g *ui.Gui, v *ui.View) error) *InputWidget {
	return &InputWidget{name: name, title: title, x: x, y: y, w: w, h: h}
}

func (iw *InputWidget) Layout(g *ui.Gui) error {
	v, err := g.SetView(iw.name, iw.x, iw.y, iw.x+iw.w, iw.y+iw.h)
	if err != nil {
		if err != ui.ErrUnknownView {
			return err
		}
		if _, err := g.SetCurrentView(v.Name()); err != nil {
			log.Panicln(err)
		}

	}
	v.Title = iw.title
	v.Editable = true
	return nil
}

type RequestBodyWidget struct {
	name  string
	title string
	x, y  int
	w, h  int
}

func NewRequestBodyWidget(name, title string, x, y, w, h int) *RequestBodyWidget {
	return &RequestBodyWidget{name: name, title: title, x: x, y: y, w: w, h: h}
}

func (iw *RequestBodyWidget) Layout(g *ui.Gui) error {
	v, err := g.SetView(iw.name, iw.x, iw.y, iw.x+iw.w, iw.y+iw.h)
	if err != nil {
		if err != ui.ErrUnknownView {
			return err
		}
	}
	v.Title = iw.title
	v.Editable = true
	v.Wrap = true
	return nil
}

type ResultsWidget struct {
	name  string
	title string
	x, y  int
	w, h  int
}

func NewResultsWidget(name, title string, x, y, w, h int) *ResultsWidget {
	return &ResultsWidget{name: name, title: title, x: x, y: y, w: w, h: h}
}

func (iw *ResultsWidget) Layout(g *ui.Gui) error {
	v, err := g.SetView(iw.name, iw.x, iw.y, iw.x+iw.w, iw.y+iw.h)
	if err != nil {
		if err != ui.ErrUnknownView {
			return err
		}
	}
	v.Title = iw.title
	return nil
}

func main() {
	g, err := ui.NewGui(ui.OutputNormal)
	if err != nil {
		log.Panicln(err)
	}
	defer g.Close()

	g.Highlight = true
	g.SelFgColor = ui.ColorRed

	maxX, maxY := g.Size()

	input := NewInputWidget("input-bar", "Api URL", 0, 0, maxX-12, 2)
	method := NewMethodWidget("method", methods[activeMethod], maxX-11, 0, maxX-2-(maxX-11), 2)
	res_body := NewRequestBodyWidget("req-body", "Response Body", 0, 3, maxX/3, maxY-4)
	res_output := NewResultsWidget("res-output", "Results", maxX/3+1, 3, maxX-2-(maxX/3), maxY-4)

	g.SetManager(input, method, res_body, res_output)
	if err := initBindings(g); err != nil {
		log.Panicln(err)
	}

	if err := g.MainLoop(); err != nil && err != ui.ErrQuit {
		log.Panicln(err)
	}
}

func initBindings(g *ui.Gui) error {
	if err := g.SetKeybinding("", ui.KeyCtrlC, ui.ModNone, quit); err != nil {
		return err
	}
	if err := g.SetKeybinding("", ui.KeyTab, ui.ModNone, nextView); err != nil {
		return err
	}
	if err := g.SetKeybinding("method", ui.KeySpace, ui.ModNone, nextMethod); err != nil {
		return err
	}
	if err := g.SetKeybinding("", ui.KeyCtrlS, ui.ModNone, processRequest); err != nil {
		return err
	}
	return nil
}

func nextView(g *ui.Gui, v *ui.View) error {
	nextIndex := (activeView + 1) % len(views)
	name := views[nextIndex]

	if _, err := g.SetCurrentView(name); err != nil {
		return err
	}

	if nextIndex == 0 || nextIndex == 2 {
		g.Cursor = true
	} else {
		g.Cursor = false
	}

	activeView = nextIndex
	return nil
}

func nextMethod(g *ui.Gui, v *ui.View) error {
	nextIndex := (activeMethod + 1) % len(views)
	label := methods[nextIndex]

	v.Clear()
	if _, err := v.Write([]byte(label)); err != nil {
		return err
	}

	activeMethod = nextIndex
	return nil
}

func processRequest(g *ui.Gui, v *ui.View) error {
	input_view, err := g.View("input-bar")
	if err != nil {
		return err
	}
	i_buf := input_view.ViewBufferLines()

	if len(i_buf) <= 0 {
		return nil
	}

	api_url := i_buf[0]

	_, err = url.ParseRequestURI(api_url)
	if err != nil {
		input_view.Clear()
		input_view.SetCursor(0, 0)
	}

	request_view, err := g.View("req-body")
	if err != nil {
		return err
	}
	req_buf := request_view.Buffer()

	if req_buf == "" && activeMethod == 1 || activeMethod == 3 {
		return errors.New("cannot do this request without a body")
	}

	if activeMethod == 1 || activeMethod == 3 {
		var data map[string]interface{}
		if err := json.Unmarshal([]byte(req_buf), &data); err != nil {
			request_view.Clear()
			request_view.SetCursor(0, 0)
			return err
		}
	}

	switch activeMethod {
	case 0:
		start := time.Now()
		res, err := http.Get(api_url)
		if err != nil {
			return err
		}
		defer res.Body.Close()

		elapsed := time.Since(start).Seconds()

		out, err := g.View("res-output")
		if err != nil {
			return err
		}

		fmt.Fprintln(out, "\n-------- HEADERS")
		for name, headers := range res.Header {
			for _, h := range headers {
				fmt.Fprintf(out, "%v: %v\n", name, h)
			}
		}

		fmt.Fprintln(out, "\n--------")

		body, err := io.ReadAll(res.Body)
		if err != nil {
			return err
		}

		var dst bytes.Buffer
		err = json.Indent(&dst, body, "", "    ")
		if err != nil {
			return err
		}

		fmt.Fprintf(out, "Body: \n%s\n", dst.String())
		fmt.Fprintf(out, "Status: %d\n", res.StatusCode)
		fmt.Fprintf(out, "Time: %f s\n", elapsed)
	}

	return nil
}

func quit(g *ui.Gui, v *ui.View) error {
	return ui.ErrQuit
}
