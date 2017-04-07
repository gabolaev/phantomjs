package phantomjs

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"time"
)

// Default settings.
const (
	DefaultPort    = 20202
	DefaultBinPath = "phantomjs"
)

// Process represents a PhantomJS process.
type Process struct {
	path string
	cmd  *exec.Cmd

	// Path to the 'phantomjs' binary.
	BinPath string

	// HTTP port used to communicate with phantomjs.
	Port int

	// Output from the process.
	Stdout io.Writer
	Stderr io.Writer
}

// NewProcess returns a new instance of Process.
func NewProcess() *Process {
	return &Process{
		BinPath: DefaultBinPath,
		Port:    DefaultPort,
		Stdout:  os.Stdout,
		Stderr:  os.Stderr,
	}
}

// Close stops the process.
func (p *Process) Close() (err error) {
	// Kill process.
	if p.cmd != nil {
		if e := p.cmd.Process.Kill(); e != nil && err == nil {
			err = e
		}
		p.cmd.Wait()
	}

	// Remove shim file.
	if p.path != "" {
		if e := os.Remove(p.path); e != nil && err == nil {
			err = e
		}
	}

	return err
}

// URL returns the process' API URL.
func (p *Process) URL() string {
	return fmt.Sprintf("http://localhost:%d", p.Port)
}

// wait continually checks the process until it gets a response or times out.
func (p *Process) wait() error {
	ticker := time.NewTicker(1000 * time.Millisecond)
	defer ticker.Stop()

	timer := time.NewTimer(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timer.C:
			return errors.New("timeout")
		case <-ticker.C:
			if err := p.ping(); err == nil {
				return nil
			}
		}
	}
}

// ping checks the process to see if it is up.
func (p *Process) ping() error {
	// Send request.
	resp, err := http.Get(p.URL() + "/ping")
	if err != nil {
		return err
	}
	resp.Body.Close()

	// Verify successful status code.
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}
	return nil
}

// CreateWebPage returns a new instance of a "webpage".
func (p *Process) CreateWebPage() *WebPage {
	var resp struct {
		Ref refJSON `json:"ref"`
	}
	p.mustDoJSON("POST", "/webpage/create", nil, &resp)
	return &WebPage{ref: newRef(p, resp.Ref.ID)}
}

// mustDoJSON sends an HTTP request to url and encodes and decodes the req/resp as JSON.
// This function will panic if it cannot communicate with the phantomjs API.
func (p *Process) mustDoJSON(method, path string, req, resp interface{}) {
	// Encode request.
	var r io.Reader
	if req != nil {
		buf, err := json.Marshal(req)
		if err != nil {
			panic(err)
		}
		r = bytes.NewReader(buf)
	}

	// Create request.
	httpRequest, err := http.NewRequest(method, p.URL()+path, r)
	if err != nil {
		panic(err)
	}

	// Send request.
	httpResponse, err := http.DefaultClient.Do(httpRequest)
	if err != nil {
		panic(err)
	}
	defer httpResponse.Body.Close()

	// Check response code.
	if httpResponse.StatusCode == http.StatusNotFound {
		panic(errors.New("not found"))
	} else if httpResponse.StatusCode == http.StatusInternalServerError {
		body, _ := ioutil.ReadAll(httpResponse.Body)
		panic(errors.New(string(body)))
	}

	// Decode response if reference passed in.
	if resp != nil {
		if err := json.NewDecoder(httpResponse.Body).Decode(resp); err != nil {
			panic(err)
		}
	}
}

// WebPage represents an object returned from "webpage.create()".
type WebPage struct {
	ref *Ref
}

// Open opens a URL.
func (p *WebPage) Open(url string) error {
	req := map[string]interface{}{
		"ref": p.ref.id,
		"url": url,
	}
	var resp struct {
		Status string `json:"status"`
	}
	p.ref.process.mustDoJSON("POST", "/webpage/open", req, &resp)

	if resp.Status != "success" {
		return errors.New("failed")
	}
	return nil
}

// CanGoBack returns true if the page can be navigated back.
func (p *WebPage) CanGoBack() bool {
	var resp struct {
		Value bool `json:"value"`
	}
	p.ref.process.mustDoJSON("POST", "/webpage/can_go_back", map[string]interface{}{"ref": p.ref.id}, &resp)
	return resp.Value
}

// CanGoForward returns true if the page can be navigated forward.
func (p *WebPage) CanGoForward() bool {
	var resp struct {
		Value bool `json:"value"`
	}
	p.ref.process.mustDoJSON("POST", "/webpage/can_go_forward", map[string]interface{}{"ref": p.ref.id}, &resp)
	return resp.Value
}

// ClipRect returns the clipping rectangle used when rendering.
// Returns nil if no clipping rectangle is set.
func (p *WebPage) ClipRect() Rect {
	var resp struct {
		Value rectJSON `json:"value"`
	}
	p.ref.process.mustDoJSON("POST", "/webpage/clip_rect", map[string]interface{}{"ref": p.ref.id}, &resp)
	return Rect{
		Top:    resp.Value.Top,
		Left:   resp.Value.Left,
		Width:  resp.Value.Width,
		Height: resp.Value.Height,
	}
}

// SetClipRect sets the clipping rectangle used when rendering.
// Set to nil to render the entire webpage.
func (p *WebPage) SetClipRect(rect Rect) {
	req := map[string]interface{}{
		"ref": p.ref.id,
		"rect": rectJSON{
			Top:    rect.Top,
			Left:   rect.Left,
			Width:  rect.Width,
			Height: rect.Height,
		},
	}
	p.ref.process.mustDoJSON("POST", "/webpage/set_clip_rect", req, nil)
}

// Content returns content of the webpage enclosed in an HTML/XML element.
func (p *WebPage) Content() string {
	var resp struct {
		Value string `json:"value"`
	}
	p.ref.process.mustDoJSON("POST", "/webpage/content", map[string]interface{}{"ref": p.ref.id}, &resp)
	return resp.Value
}

func (p *WebPage) Cookies() string {
	panic("TODO")
}

func (p *WebPage) CustomHeaders() string {
	panic("TODO")
}

func (p *WebPage) Event() string {
	panic("TODO")
}

func (p *WebPage) FocusedFrameName() string {
	panic("TODO")
}

func (p *WebPage) FrameContent() string {
	panic("TODO")
}

func (p *WebPage) FrameName() string {
	panic("TODO")
}

func (p *WebPage) FramePlainText() string {
	panic("TODO")
}

func (p *WebPage) FrameTitle() string {
	panic("TODO")
}

func (p *WebPage) FrameUrl() string {
	panic("TODO")
}

func (p *WebPage) FramesCount() string {
	panic("TODO")
}

func (p *WebPage) FramesName() string {
	panic("TODO")
}

func (p *WebPage) LibraryPath() string {
	panic("TODO")
}

func (p *WebPage) NavigationLocked() string {
	panic("TODO")
}

func (p *WebPage) OfflineStoragePath() string {
	panic("TODO")
}

func (p *WebPage) OfflineStorageQuota() string {
	panic("TODO")
}

func (p *WebPage) OwnsPages() string {
	panic("TODO")
}

func (p *WebPage) PagesWindowName() string {
	panic("TODO")
}

func (p *WebPage) Pages() string {
	panic("TODO")
}

func (p *WebPage) PaperSize() string {
	panic("TODO")
}

func (p *WebPage) PlainText() string {
	panic("TODO")
}

func (p *WebPage) ScrollPosition() string {
	panic("TODO")
}

func (p *WebPage) Settings() string {
	panic("TODO")
}

func (p *WebPage) Title() string {
	panic("TODO")
}

func (p *WebPage) Url() string {
	panic("TODO")
}

func (p *WebPage) ViewportSize() string {
	panic("TODO")
}

func (p *WebPage) WindowName() string {
	panic("TODO")
}

func (p *WebPage) ZoomFactor() string {
	panic("TODO")
}

func (p *WebPage) AddCookie() {
	panic("TODO")
}

func (p *WebPage) ChildFramesCount() {
	panic("TODO")
}

func (p *WebPage) ChildFramesName() {
	panic("TODO")
}

func (p *WebPage) ClearCookies() {
	panic("TODO")
}

// Close releases the web page and its resources.
func (p *WebPage) Close() {
	p.ref.process.mustDoJSON("POST", "/webpage/close", map[string]interface{}{"ref": p.ref.id}, nil)
}

func (p *WebPage) CurrentFrameName() {
	panic("TODO")
}

func (p *WebPage) DeleteCookie() {
	panic("TODO")
}

func (p *WebPage) EvaluateAsync() {
	panic("TODO")
}

func (p *WebPage) EvaluateJavaScript() {
	panic("TODO")
}

func (p *WebPage) Evaluate() {
	panic("TODO")
}

func (p *WebPage) GetPage() {
	panic("TODO")
}

func (p *WebPage) GoBack() {
	panic("TODO")
}

func (p *WebPage) GoForward() {
	panic("TODO")
}

func (p *WebPage) Go() {
	panic("TODO")
}

func (p *WebPage) IncludeJs() {
	panic("TODO")
}

func (p *WebPage) InjectJs() {
	panic("TODO")
}

func (p *WebPage) OpenUrl() {
	panic("TODO")
}

// Open start the phantomjs process with the shim script.
func (p *Process) Open() error {
	// Write shim to a temporary file.
	f, err := ioutil.TempFile("", "phantomjs-")
	if err != nil {
		return err
	} else if _, err := f.WriteString(shim); err != nil {
		f.Close()
		os.Remove(f.Name())
		return err
	} else if err := f.Close(); err != nil {
		os.Remove(f.Name())
		return err
	}
	p.path = f.Name()

	// Start external process.
	cmd := exec.Command(p.BinPath, p.path)
	cmd.Env = []string{fmt.Sprintf("PORT=%d", p.Port)}
	cmd.Stdout = p.Stdout
	cmd.Stderr = p.Stderr
	if err := cmd.Start(); err != nil {
		return err
	}
	p.cmd = cmd

	// Wait until process is available.
	if err := p.wait(); err != nil {
		return err
	}

	return nil
}

func (p *WebPage) Release() {
	panic("TODO")
}

func (p *WebPage) Reload() {
	panic("TODO")
}

func (p *WebPage) RenderBase64() {
	panic("TODO")
}

func (p *WebPage) RenderBuffer() {
	panic("TODO")
}

func (p *WebPage) Render() {
	panic("TODO")
}

func (p *WebPage) SendEvent() {
	panic("TODO")
}

func (p *WebPage) SetContent() {
	panic("TODO")
}

func (p *WebPage) Stop() {
	panic("TODO")
}

func (p *WebPage) SwitchToChildFrame() {
	panic("TODO")
}

func (p *WebPage) SwitchToFocusedFrame() {
	panic("TODO")
}

func (p *WebPage) SwitchToFrame() {
	panic("TODO")
}

func (p *WebPage) SwitchToMainFrame() {
	panic("TODO")
}

func (p *WebPage) SwitchToParentFrame() {
	panic("TODO")
}

func (p *WebPage) UploadFile() {
	panic("TODO")
}

// OpenWebPageSettings represents the settings object passed to WebPage.Open().
type OpenWebPageSettings struct {
	Method string `json:"method"`
}

// Ref represents a reference to an object in phantomjs.
type Ref struct {
	process *Process
	id      string
}

// newRef returns a new instance of a referenced object within the process.
func newRef(p *Process, id string) *Ref {
	return &Ref{process: p, id: id}
}

// ID returns the reference identifier.
func (r *Ref) ID() string {
	return r.id
}

// refJSON is a struct for encoding refs as JSON.
type refJSON struct {
	ID string `json:"id"`
}

// Rect represents a rectangle used by WebPage.ClipRect().
type Rect struct {
	Top    int
	Left   int
	Width  int
	Height int
}

// rectJSON is a struct for encoding rects as JSON.
type rectJSON struct {
	Top    int `json:"top"`
	Left   int `json:"left"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

// shim is the included javascript used to communicate with PhantomJS.
const shim = `
var system = require("system")
var webpage = require('webpage');
var webserver = require('webserver');

/*
 * HTTP API
 */

// Serves RPC API.
var server = webserver.create();
server.listen(system.env["PORT"], function(request, response) {
	try {
		switch (request.url) {
			case '/ping': return handlePing(request, response);
			case '/webpage/can_go_back': return handleWebpageCanGoBack(request, response);
			case '/webpage/can_go_forward': return handleWebpageCanGoForward(request, response);
			case '/webpage/clip_rect': return handleWebpageClipRect(request, response);
			case '/webpage/set_clip_rect': return handleWebpageSetClipRect(request, response);
			case '/webpage/create': return handleWebpageCreate(request, response);
			case '/webpage/content': return handleWebpageContent(request, response);
			case '/webpage/open': return handleWebpageOpen(request, response);
			case '/webpage/close': return handleWebpageClose(request, response);
			default: return handleNotFound(request, response);
		}
	} catch(e) {
		response.statusCode = 500;
		response.write(request.url + ": " + e.message);
		response.closeGracefully();
	}
});

function handlePing(request, response) {
	response.statusCode = 200;
	response.write('ok');
	response.closeGracefully();
}

function handleWebpageCanGoBack(request, response) {
	var page = ref(JSON.parse(request.post).ref);
	response.write(JSON.stringify({value: page.canGoBack}));
	response.closeGracefully();
}

function handleWebpageCanGoForward(request, response) {
	var page = ref(JSON.parse(request.post).ref);
	response.write(JSON.stringify({value: page.canGoForward}));
	response.closeGracefully();
}

function handleWebpageClipRect(request, response) {
	var page = ref(JSON.parse(request.post).ref);
	response.write(JSON.stringify({value: page.clipRect}));
	response.closeGracefully();
}

function handleWebpageSetClipRect(request, response) {
	var msg = JSON.parse(request.post);
	var page = ref(msg.ref);
	page.clipRect = msg.rect;
	response.closeGracefully();
}

function handleWebpageCreate(request, response) {
	var ref = createRef(webpage.create());
	response.statusCode = 200;
	response.write(JSON.stringify({ref: ref}));
	response.closeGracefully();
}

function handleWebpageOpen(request, response) {
	var msg = JSON.parse(request.post)
	var page = ref(msg.ref)
	page.open(msg.url, function(status) {
		response.write(JSON.stringify({status: status}));
		response.closeGracefully();
	})
}

function handleWebpageContent(request, response) {
	var page = ref(JSON.parse(request.post).ref)
	response.write(JSON.stringify({value: page.content}));
	response.closeGracefully();
}

function handleWebpageClose(request, response) {
	var msg = JSON.parse(request.post)
	var page = ref(msg.ref)
	page.close()
	delete(refs, msg.ref)
	response.statusCode = 200;
	response.closeGracefully();
}

function handleNotFound(request, response) {
	response.statusCode = 404;
	response.write('not found');
	response.closeGracefully();
}


/*
 * REFS
 */

// Holds references to remote objects.
var refID = 0;
var refs = {};

// Adds an object to the reference map and a ref object.
function createRef(value) {
	refID++;
	refs[refID] = value;
	return {id: refID.toString()};
}

// Returns a reference object by ID.
function ref(id) {
	return refs[id];
}
`
