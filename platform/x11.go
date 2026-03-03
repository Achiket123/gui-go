// Package platform provides low-level CGo bindings to the X11 (Xlib) library.
// All exported functions are thin wrappers that convert Go types to C types
// and call the corresponding Xlib functions.
//
// CGo notes:
//   - All X11 calls MUST happen on the same OS thread.
//     The caller must ensure runtime.LockOSThread() is called.
//   - C pointers (Display*, Window, GC, etc.) are passed as unsafe.Pointer
//     so that the rest of the library stays free of CGo imports.
package platform

/*
#cgo LDFLAGS: -lX11

#include <X11/Xlib.h>
#include <X11/Xutil.h>
#include <X11/keysym.h>
#include <X11/XKBlib.h>
#include <stdlib.h>
#include <string.h>

// Helper: allocate an XColor and call XAllocColor, return the pixel value.
static unsigned long allocColor(Display* dpy, Colormap cmap,
                                unsigned short r, unsigned short g, unsigned short b) {
    XColor c;
    c.red   = r;
    c.green = g;
    c.blue  = b;
    c.flags = DoRed | DoGreen | DoBlue;
    XAllocColor(dpy, cmap, &c);
    return c.pixel;
}

// Helper: set line attributes (width, solid style, cap, join).
static void setLineWidth(Display* dpy, GC gc, unsigned int width) {
    XSetLineAttributes(dpy, gc, width, LineSolid, CapRound, JoinRound);
}

// Helper: draw polygon outline using XDrawLines.
static void drawPolygon(Display* dpy, Drawable win, GC gc,
                        XPoint* pts, int n) {
    XDrawLines(dpy, win, gc, pts, n, CoordModeOrigin);
}

// Helper: create an XImage from a raw byte buffer (BGRA / 32bpp).
// The caller owns the data buffer; XDestroyImage must be called to free the XImage
// (it will NOT free the data because we pass a pre-allocated buffer).
static XImage* createXImage(Display* dpy, int screen,
                             int width, int height, char* data) {
    Visual* vis = DefaultVisual(dpy, screen);
    XImage* img = XCreateImage(dpy, vis, DefaultDepth(dpy, screen),
                                ZPixmap, 0, data, width, height, 32, 0);
    return img;
}

// Helper: draw an XImage at (x,y) on a drawable.
static void putXImage(Display* dpy, Drawable win, GC gc, XImage* img,
                      int x, int y, int w, int h) {
    XPutImage(dpy, win, gc, img, 0, 0, x, y, (unsigned int)w, (unsigned int)h);
}

// Helper: look up a keysym name string for a keycode.
// Returns a static string managed by Xlib (do not free).
static const char* keySymName(Display* dpy, unsigned int keycode, unsigned int state) {
    // XKeycodeToKeysym is deprecated; use XkbKeycodeToKeysym instead.
    KeySym ks = XkbKeycodeToKeysym(dpy, keycode, 0, (state & ShiftMask) ? 1 : 0);
    return XKeysymToString(ks);
}

// Helper: retrieve modifier state bits from an XEvent.
// Works for ButtonPress/Release, MotionNotify, KeyPress/Release.
static unsigned int eventState(XEvent* e) {
    // All these event types have state at the same offset.
    return e->xkey.state;
}
// Helper: extract the first long from a ClientMessage data union.
static long clientMessageAtom(XEvent* e) {
    return e->xclient.data.l[0];
}
// Helper: destroy an XImage without freeing its data buffer.
// XDestroyImage is a macro so we wrap it in a static inline.
static void destroyXImage(XImage* img) {
    img->data = NULL; // do not free Go-owned buffer
    XDestroyImage(img);
}
*/
import "C"

import (
	"unsafe"
)

// --- Display / Window lifecycle ---

// OpenDisplay connects to the X server. Returns nil on failure.
func OpenDisplay() unsafe.Pointer {
	return unsafe.Pointer(C.XOpenDisplay(nil))
}

// CloseDisplay disconnects from the X server.
func CloseDisplay(display unsafe.Pointer) {
	C.XCloseDisplay((*C.Display)(display))
}

// DefaultScreen returns the default screen number.
func DefaultScreen(display unsafe.Pointer) int {
	return int(C.XDefaultScreen((*C.Display)(display)))
}

// DefaultRootWindow returns the root window ID for the default screen.
func DefaultRootWindow(display unsafe.Pointer) uintptr {
	return uintptr(C.XDefaultRootWindow((*C.Display)(display)))
}

// DefaultColormap returns the default colormap for a screen.
func DefaultColormap(display unsafe.Pointer, screen int) unsafe.Pointer {
	cmap := C.XDefaultColormap((*C.Display)(display), C.int(screen))
	return unsafe.Pointer(uintptr(cmap))
}

// CreateSimpleWindow creates a basic window and returns its ID.
func CreateSimpleWindow(display unsafe.Pointer, parent uintptr,
	x, y, width, height, borderWidth int,
	border, background uint64) uintptr {

	win := C.XCreateSimpleWindow(
		(*C.Display)(display),
		C.Window(parent),
		C.int(x), C.int(y),
		C.uint(width), C.uint(height),
		C.uint(borderWidth),
		C.ulong(border),
		C.ulong(background),
	)
	return uintptr(win)
}

// StoreName sets the window title.
func StoreName(display unsafe.Pointer, win uintptr, title string) {
	cs := C.CString(title)
	defer C.free(unsafe.Pointer(cs))
	C.XStoreName((*C.Display)(display), C.Window(win), cs)
}

// MapWindow makes a window visible.
func MapWindow(display unsafe.Pointer, win uintptr) {
	C.XMapWindow((*C.Display)(display), C.Window(win))
}

// DestroyWindow destroys a window.
func DestroyWindow(display unsafe.Pointer, win uintptr) {
	C.XDestroyWindow((*C.Display)(display), C.Window(win))
}

// ResizeWindow resizes a window.
func ResizeWindow(display unsafe.Pointer, win uintptr, width, height int) {
	C.XResizeWindow((*C.Display)(display), C.Window(win), C.uint(width), C.uint(height))
}

// --- Event handling ---

// SelectInput registers the event mask for a window.
// mask is the ORed combination of X11 event mask constants.
func SelectInput(display unsafe.Pointer, win uintptr, mask int64) {
	C.XSelectInput((*C.Display)(display), C.Window(win), C.long(mask))
}

// EventMask constants matching X11 definitions.
const (
	ExposureMask        int64 = 1 << 15
	KeyPressMask        int64 = 1 << 0
	KeyReleaseMask      int64 = 1 << 1
	ButtonPressMask     int64 = 1 << 2
	ButtonReleaseMask   int64 = 1 << 3
	PointerMotionMask   int64 = 1 << 6
	StructureNotifyMask int64 = 1 << 17
)

// EventType constants.
const (
	KeyPress        = 2
	KeyRelease      = 3
	ButtonPress     = 4
	ButtonRelease   = 5
	MotionNotify    = 6
	Expose          = 12
	DestroyNotify   = 17
	ConfigureNotify = 22
	ClientMessage   = 33
	ShiftMask       = 1
	ControlMask     = 4
	Mod1Mask        = 8 // Alt
)

// XEventData carries the decoded fields from an XEvent union.
type XEventData struct {
	Type    int
	X, Y    int
	Button  int
	KeyCode int
	KeySym  string
	Width   int
	Height  int
	State   int
	Atom    uint64 // for ClientMessage l[0]
}

// Pending returns the number of events waiting in the queue (non-blocking).
func Pending(display unsafe.Pointer) int {
	return int(C.XPending((*C.Display)(display)))
}

// NextEvent retrieves and removes the next event from the queue.
// Returns the decoded event data.
func NextEvent(display unsafe.Pointer) XEventData {
	var ev C.XEvent
	C.XNextEvent((*C.Display)(display), &ev)

	// The type is the first int in every XEvent union member.
	evType := int(*(*C.int)(unsafe.Pointer(&ev)))

	d := XEventData{Type: evType}
	state := int(C.eventState(&ev))
	d.State = state

	switch evType {
	case KeyPress, KeyRelease:
		ke := (*C.XKeyEvent)(unsafe.Pointer(&ev))
		d.KeyCode = int(ke.keycode)
		name := C.keySymName((*C.Display)(display), ke.keycode, ke.state)
		if name != nil {
			d.KeySym = C.GoString(name)
		}
	case ButtonPress, ButtonRelease:
		be := (*C.XButtonEvent)(unsafe.Pointer(&ev))
		d.X = int(be.x)
		d.Y = int(be.y)
		d.Button = int(be.button)
	case MotionNotify:
		me := (*C.XMotionEvent)(unsafe.Pointer(&ev))
		d.X = int(me.x)
		d.Y = int(me.y)
	case ConfigureNotify:
		ce := (*C.XConfigureEvent)(unsafe.Pointer(&ev))
		d.Width = int(ce.width)
		d.Height = int(ce.height)
	case ClientMessage:
		d.Atom = uint64(C.clientMessageAtom(&ev))
	}
	return d
}

// InternAtom returns the atom ID for a named property.
func InternAtom(display unsafe.Pointer, name string, onlyIfExists bool) uint64 {
	cs := C.CString(name)
	defer C.free(unsafe.Pointer(cs))
	exists := C.Bool(0)
	if onlyIfExists {
		exists = 1
	}
	atom := C.XInternAtom((*C.Display)(display), cs, exists)
	return uint64(atom)
}

// SetWMProtocols registers WM protocols (e.g. WM_DELETE_WINDOW).
func SetWMProtocols(display unsafe.Pointer, win uintptr, atoms []uint64) {
	if len(atoms) == 0 {
		return
	}
	cAtoms := make([]C.Atom, len(atoms))
	for i, a := range atoms {
		cAtoms[i] = C.Atom(a)
	}
	C.XSetWMProtocols((*C.Display)(display), C.Window(win), &cAtoms[0], C.int(len(cAtoms)))
}

// --- Graphics Context ---

// CreateGC creates a Graphics Context for a window.
func CreateGC(display unsafe.Pointer, win uintptr) unsafe.Pointer {
	gc := C.XCreateGC((*C.Display)(display), C.Drawable(win), 0, nil)
	return unsafe.Pointer(gc)
}

// FreeGC frees a Graphics Context.
func FreeGC(display unsafe.Pointer, gc unsafe.Pointer) {
	C.XFreeGC((*C.Display)(display), C.GC(gc))
}

// SetForeground sets the foreground (draw) color on a GC.
func SetForeground(display unsafe.Pointer, gc unsafe.Pointer, pixel uint64) {
	C.XSetForeground((*C.Display)(display), C.GC(gc), C.ulong(pixel))
}

// SetLineWidth sets the line width on a GC.
func SetLineWidth(display unsafe.Pointer, gc unsafe.Pointer, width int) {
	C.setLineWidth((*C.Display)(display), C.GC(gc), C.uint(width))
}

// --- Color allocation ---

// AllocColor allocates an RGB color and returns the pixel value.
// r, g, b are 0–255.
func AllocColor(display, colormap unsafe.Pointer, r, g, b uint8) uint64 {
	return uint64(C.allocColor(
		(*C.Display)(display),
		C.Colormap(uintptr(colormap)),
		C.ushort(uint16(r)<<8),
		C.ushort(uint16(g)<<8),
		C.ushort(uint16(b)<<8),
	))
}

// --- Drawing primitives ---

// FillRectangle draws a filled rectangle.
func FillRectangle(display unsafe.Pointer, win uintptr, gc unsafe.Pointer, x, y, w, h int) {
	C.XFillRectangle((*C.Display)(display), C.Drawable(win), C.GC(gc),
		C.int(x), C.int(y), C.uint(w), C.uint(h))
}

// DrawRectangle draws a rectangle outline.
func DrawRectangle(display unsafe.Pointer, win uintptr, gc unsafe.Pointer, x, y, w, h int) {
	C.XDrawRectangle((*C.Display)(display), C.Drawable(win), C.GC(gc),
		C.int(x), C.int(y), C.uint(w), C.uint(h))
}

// FillArc draws a filled arc/circle.
// angle1 and angle2 are in 64ths of a degree (360*64 = full circle).
func FillArc(display unsafe.Pointer, win uintptr, gc unsafe.Pointer, x, y, w, h, angle1, angle2 int) {
	C.XFillArc((*C.Display)(display), C.Drawable(win), C.GC(gc),
		C.int(x), C.int(y), C.uint(w), C.uint(h), C.int(angle1), C.int(angle2))
}

// DrawArc draws an arc/circle outline.
func DrawArc(display unsafe.Pointer, win uintptr, gc unsafe.Pointer, x, y, w, h, angle1, angle2 int) {
	C.XDrawArc((*C.Display)(display), C.Drawable(win), C.GC(gc),
		C.int(x), C.int(y), C.uint(w), C.uint(h), C.int(angle1), C.int(angle2))
}

// DrawLine draws a straight line.
func DrawLine(display unsafe.Pointer, win uintptr, gc unsafe.Pointer, x1, y1, x2, y2 int) {
	C.XDrawLine((*C.Display)(display), C.Drawable(win), C.GC(gc),
		C.int(x1), C.int(y1), C.int(x2), C.int(y2))
}

// Point is a 2D integer coordinate, compatible with XPoint.
type Point struct {
	X, Y int
}

// FillPolygon draws a filled polygon.
func FillPolygon(display unsafe.Pointer, win uintptr, gc unsafe.Pointer, points []Point) {
	if len(points) == 0 {
		return
	}
	cpts := make([]C.XPoint, len(points))
	for i, p := range points {
		cpts[i].x = C.short(p.X)
		cpts[i].y = C.short(p.Y)
	}
	C.XFillPolygon((*C.Display)(display), C.Drawable(win), C.GC(gc),
		&cpts[0], C.int(len(cpts)), C.Complex, C.CoordModeOrigin)
}

// DrawPolygon draws a polygon outline (closed).
func DrawPolygon(display unsafe.Pointer, win uintptr, gc unsafe.Pointer, points []Point) {
	if len(points) < 2 {
		return
	}
	// Close the polygon by appending the first point.
	closed := append(points, points[0])
	cpts := make([]C.XPoint, len(closed))
	for i, p := range closed {
		cpts[i].x = C.short(p.X)
		cpts[i].y = C.short(p.Y)
	}
	C.drawPolygon((*C.Display)(display), C.Drawable(win), C.GC(gc),
		&cpts[0], C.int(len(cpts)))
}

// DrawString draws a text string at (x, y).
func DrawString(display unsafe.Pointer, win uintptr, gc unsafe.Pointer, x, y int, text string) {
	cs := C.CString(text)
	defer C.free(unsafe.Pointer(cs))
	C.XDrawString((*C.Display)(display), C.Drawable(win), C.GC(gc),
		C.int(x), C.int(y), cs, C.int(len(text)))
}

// --- Fonts ---

// LoadFont loads an X11 font by XLFD name and returns its handle.
func LoadFont(display unsafe.Pointer, name string) unsafe.Pointer {
	cs := C.CString(name)
	defer C.free(unsafe.Pointer(cs))
	font := C.XLoadFont((*C.Display)(display), cs)
	return unsafe.Pointer(uintptr(font))
}

// SetFont sets a font on a GC.
func SetFont(display unsafe.Pointer, gc unsafe.Pointer, font unsafe.Pointer) {
	C.XSetFont((*C.Display)(display), C.GC(gc), C.Font(uintptr(font)))
}

// TextExtents returns the width and height (ascent+descent) of a string.
func TextExtents(display unsafe.Pointer, font unsafe.Pointer, text string) (w, h int) {
	cs := C.CString(text)
	defer C.free(unsafe.Pointer(cs))
	var dir, ascent, descent C.int
	var overall C.XCharStruct
	C.XQueryTextExtents((*C.Display)(display), C.XID(uintptr(font)),
		cs, C.int(len(text)), &dir, &ascent, &descent, &overall)
	return int(overall.width), int(ascent + descent)
}

// --- Pixmap (double buffering) ---

// CreatePixmap creates an offscreen pixmap.
func CreatePixmap(display unsafe.Pointer, win uintptr, width, height, depth int) uintptr {
	pm := C.XCreatePixmap((*C.Display)(display), C.Drawable(win),
		C.uint(width), C.uint(height), C.uint(depth))
	return uintptr(pm)
}

// FreePixmap frees a pixmap.
func FreePixmap(display unsafe.Pointer, pixmap uintptr) {
	C.XFreePixmap((*C.Display)(display), C.Pixmap(pixmap))
}

// CopyArea copies a rectangular region from src to dst drawable.
func CopyArea(display unsafe.Pointer, src, dst uintptr, gc unsafe.Pointer,
	srcX, srcY, width, height, dstX, dstY int) {
	C.XCopyArea((*C.Display)(display),
		C.Drawable(src), C.Drawable(dst), C.GC(gc),
		C.int(srcX), C.int(srcY),
		C.uint(width), C.uint(height),
		C.int(dstX), C.int(dstY))
}

// DefaultDepth returns the default visual depth for a screen.
func DefaultDepth(display unsafe.Pointer, screen int) int {
	return int(C.XDefaultDepth((*C.Display)(display), C.int(screen)))
}

// --- XImage (pixel buffer) ---

// XImageHandle is an opaque handle to a C XImage.
type XImageHandle struct {
	ptr  unsafe.Pointer
	data []byte // We own the backing pixel data
}

// CreateXImage creates an XImage backed by a Go byte slice (BGRX, 32bpp).
// data must be width*height*4 bytes, in BGRX order.
func CreateXImage(display unsafe.Pointer, screen, width, height int, data []byte) *XImageHandle {
	img := C.createXImage((*C.Display)(display), C.int(screen),
		C.int(width), C.int(height), (*C.char)(unsafe.Pointer(&data[0])))
	return &XImageHandle{ptr: unsafe.Pointer(img), data: data}
}

// PutXImage draws an XImage onto a drawable.
func PutXImage(display unsafe.Pointer, win uintptr, gc unsafe.Pointer,
	img *XImageHandle, x, y, w, h int) {
	C.putXImage((*C.Display)(display), C.Drawable(win), C.GC(gc),
		(*C.XImage)(img.ptr), C.int(x), C.int(y), C.int(w), C.int(h))
}

// DestroyXImage frees the XImage struct (but not the backing data — Go owns that).
func DestroyXImage(img *XImageHandle) {
	if img == nil || img.ptr == nil {
		return
	}
	C.destroyXImage((*C.XImage)(img.ptr))
	img.ptr = nil
}

// --- Flush ---

// Flush flushes pending draw commands to the X server.
func Flush(display unsafe.Pointer) {
	C.XFlush((*C.Display)(display))
}

// Sync flushes and waits for the X server to process all commands.
func Sync(display unsafe.Pointer) {
	C.XSync((*C.Display)(display), 0)
}
