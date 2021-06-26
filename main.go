package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"

	ui "github.com/jroimartin/gocui"
)

var (
	methods       = []string{"GET", "POST", "DELETE", "PUT"}
	views         = []string{"input-bar", "method", "history", "req-body", "res-output"}
	active_view   = len(views) - 1
	active_method = 0
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

func (w *MethodWidget) Layout(g *ui.Gui) error {
	v, err := g.SetView(w.name, w.x, w.y, w.x+w.w, w.y+w.h)
	if err != nil {
		if err != ui.ErrUnknownView {
			return err
		}
		fmt.Fprintf(v, "%s", methods[active_method])
	}
	return nil
}

type InputWidget struct {
	name  string
	title string
	x, y  int
	w, h  int
}

func NewInputWidget(name, title string, x, y, w, h int) *InputWidget {
	return &InputWidget{name: name, title: title, x: x, y: y, w: w, h: h}
}

func (w *InputWidget) Layout(g *ui.Gui) error {
	v, err := g.SetView(w.name, w.x, w.y, w.x+w.w, w.y+w.h)
	if err != nil {
		if err != ui.ErrUnknownView {
			return err
		}
		if _, err := g.SetCurrentView(v.Name()); err != nil {
			log.Panicln(err)
		}

	}
	v.Title = w.title
	v.Editable = true
	return nil
}

type HelperMenuWidget struct {
	name  string
	title string
	x, y  int
	w, h  int
}

func NewHelperMenuWidget(name, title string, x, y, w, h int) *HelperMenuWidget {
	return &HelperMenuWidget{name: name, title: title, x: x, y: y, w: w, h: h}
}

func (w *HelperMenuWidget) Layout(g *ui.Gui) error {
	v, err := g.SetView(w.name, w.x, w.y, w.x+w.w, w.y+w.h)
	if err != nil {
		if err != ui.ErrUnknownView {
			return err
		}
		if _, err := g.SetCurrentView(v.Name()); err != nil {
			log.Panicln(err)
		}
	}
	v.Title = w.title
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

func (w *RequestBodyWidget) Layout(g *ui.Gui) error {
	v, err := g.SetView(w.name, w.x, w.y, w.x+w.w, w.y+w.h)
	if err != nil {
		if err != ui.ErrUnknownView {
			return err
		}
	}
	v.Title = w.title
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

func (w *ResultsWidget) Layout(g *ui.Gui) error {
	v, err := g.SetView(w.name, w.x, w.y, w.x+w.w, w.y+w.h)
	if err != nil {
		if err != ui.ErrUnknownView {
			return err
		}
	}
	v.Title = w.title
	return nil
}

func main() {
	g, err := ui.NewGui(ui.OutputNormal)
	if err != nil {
		log.Panicln(err)
	}
	defer g.Close()

	g.Highlight = true
	g.SelFgColor = ui.ColorGreen

	max_x, max_y := g.Size()

	input := NewInputWidget("input-bar", "Api URL", 0, 0, max_x-12, 2)
	method := NewMethodWidget("method", methods[active_method], max_x-11, 0, max_x-2-(max_x-11), 2)
	history := NewHelperMenuWidget("history", "History", 0, 3, max_x/3, max_y/3)
	res_body := NewRequestBodyWidget("req-body", "Response Body", 0, max_y/3+4, max_x/3, max_y-2-(max_y/2-3))
	res_output := NewResultsWidget("res-output", "Results", max_x/3+1, 3, max_x-2-(max_x/3), max_y-4)

	g.SetManager(input, method, history, res_body, res_output)
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
	if err := g.SetKeybinding("", ui.KeyCtrlS, ui.ModNone, processRequest); err != nil {
		return err
	}
	if err := g.SetKeybinding("", ui.KeyCtrlL, ui.ModNone, clear); err != nil {
		return err
	}
	if err := g.SetKeybinding("method", ui.KeySpace, ui.ModNone, nextMethod); err != nil {
		return err
	}
	if err := g.SetKeybinding("res-output", ui.KeyArrowUp, ui.ModNone, cursorUp); err != nil {
		return err
	}
	if err := g.SetKeybinding("res-output", ui.KeyArrowDown, ui.ModNone, cursorDown); err != nil {
		return err
	}
	return nil
}

func cursorDown(g *ui.Gui, v *ui.View) error {
	if v != nil {
		cx, cy := v.Cursor()
		if err := v.SetCursor(cx, cy+1); err != nil {
			ox, oy := v.Origin()
			if err := v.SetOrigin(ox, oy+1); err != nil {
				return err
			}
		}
	}
	return nil
}

func cursorUp(g *ui.Gui, v *ui.View) error {
	if v != nil {
		ox, oy := v.Origin()
		cx, cy := v.Cursor()
		if err := v.SetCursor(cx, cy-1); err != nil && oy > 0 {
			if err := v.SetOrigin(ox, oy-1); err != nil {
				return err
			}
		}
	}
	return nil
}

func nextView(g *ui.Gui, v *ui.View) error {
	next_index := (active_view + 1) % len(views)
	name := views[next_index]

	if _, err := g.SetCurrentView(name); err != nil {
		return err
	}

	if next_index != 1 {
		g.Cursor = true
	} else {
		g.Cursor = false
	}

	active_view = next_index
	return nil
}

func nextMethod(g *ui.Gui, v *ui.View) error {
	next_index := (active_method + 1) % len(methods)
	label := methods[next_index]

	v.Clear()
	if _, err := v.Write([]byte(label)); err != nil {
		return err
	}

	active_method = next_index
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
		return nil
	}

	request_view, err := g.View("req-body")
	if err != nil {
		return err
	}
	req_buf := request_view.Buffer()

	var request_body_data []byte
	if active_method == 1 || active_method == 3 {
		valid := json.Valid([]byte(req_buf))
		if !valid {
			return nil
		}
		request_body_data = []byte(req_buf)
	}

	resultsChan := make(chan string)

	switch active_method {
	case 0:
		out, err := g.View("res-output")
		if err != nil {
			return err
		}
		go httpRequest(api_url, "GET", nil, out, resultsChan)
		fmt.Fprintln(out, <-resultsChan)
	case 1:
		out, err := g.View("res-output")
		if err != nil {
			return err
		}
		go httpRequest(api_url, "POST", request_body_data, out, resultsChan)
		fmt.Fprintln(out, <-resultsChan)
	case 2:
		out, err := g.View("res-output")
		if err != nil {
			return err
		}
		go httpRequest(api_url, "DELETE", request_body_data, out, resultsChan)
		fmt.Fprintln(out, <-resultsChan)
	case 3:
		out, err := g.View("res-output")
		if err != nil {
			return err
		}
		go httpRequest(api_url, "PUT", request_body_data, out, resultsChan)
		fmt.Fprintln(out, <-resultsChan)
	}

	return nil
}

func httpRequest(url, method string, body []byte, v *ui.View, ch chan string) error {
	start := time.Now()
	duration := time.Since(start).Seconds()
	fmt.Fprintf(v, "Time: %f s\n", duration)

	client := &http.Client{}
	req, err := http.NewRequest(method, url, bytes.NewBuffer(body))
	req.Header.Add("content-type", "application/json")
	if err != nil {
		ch <- fmt.Sprintf("Error: %s\n", err.Error())
		return nil
	}

	res, err := client.Do(req)
	if err != nil {
		ch <- fmt.Sprintf("Error: %s\n", err.Error())
		return nil
	}

	result, err := formatResponse(res)
	if err != nil {
		ch <- fmt.Sprintf("Error: %s\n", err.Error())
		return nil
	}
	defer res.Body.Close()

	ch <- result

	return nil
}

type Result struct {
	method string
	url    string
	path   string
	proto  string
	header http.Header
	status string
	body   string
}

func formatResponse(res *http.Response) (string, error) {
	var resultStr string

	r := Result{
		method: res.Request.Method,
		url:    res.Request.URL.String(),
		path:   res.Request.URL.Path,
		proto:  res.Proto,
		status: res.Status,
		header: res.Header,
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	r.body = string(body)

	resultStr += fmt.Sprintf("%s %s %s\n", r.method, r.path, r.proto)
	resultStr += fmt.Sprintf("Status %s \n", r.status)
	resultStr += "\n"

	for name, headers := range r.header {
		for _, h := range headers {
			resultStr += fmt.Sprintf("%s => %s\n", name, h)
		}
	}

	resultStr += "\n"
	resultStr += fmt.Sprintf("body  \n %s", r.body)

	history = append(history, resultStr)

	return resultStr, nil
}

// func saveRequest(result Result) error {
// 	content := fmt.Sprintf("%s %s/%s %s\n", result.method, result.url, result.path, result.status)

// 	home, err := homedir.Dir()
// 	if err != nil {
// 		return err
// 	}

// 	file_path := fmt.Sprintf("%s/hai.txt", home)

// 	f, err := os.Create(file_path)
// 	if err != nil {
// 		return err
// 	}

// 	w := bufio.NewWriter(f)
// 	_, err = w.WriteString(content)
// 	if err != nil {
// 		return err
// 	}

// 	return nil
// }

func quit(g *ui.Gui, v *ui.View) error {
	return ui.ErrQuit
}

func clear(g *ui.Gui, v *ui.View) error {
	return clearView(v)
}

func clearView(v *ui.View) error {
	v.Clear()
	if err := v.SetCursor(0, 0); err != nil {
		return err
	}
	return nil
}
