package gl

/*
#cgo LDFLAGS: -lGL -lX11

#include <GL/gl.h>
#include <GL/glx.h>
#include <X11/Xlib.h>
#include <stdlib.h>

// glXChooseFBConfig wrapper returning first valid config.
static GLXFBConfig chooseFBConfig(Display* dpy, int screen) {
    int attribs[] = {
        GLX_X_RENDERABLE,  True,
        GLX_DRAWABLE_TYPE, GLX_WINDOW_BIT,
        GLX_RENDER_TYPE,   GLX_RGBA_BIT,
        GLX_DOUBLEBUFFER,  True,
        GLX_RED_SIZE,      8,
        GLX_GREEN_SIZE,    8,
        GLX_BLUE_SIZE,     8,
        GLX_ALPHA_SIZE,    8,
        GLX_DEPTH_SIZE,    0,
        None
    };
    int count = 0;
    GLXFBConfig* cfgs = glXChooseFBConfig(dpy, screen, attribs, &count);
    if (!cfgs || count == 0) return NULL;
    GLXFBConfig best = cfgs[0];
    XFree(cfgs);
    return best;
}

static GLXContext createContext(Display* dpy, GLXFBConfig cfg) {
    return glXCreateNewContext(dpy, cfg, GLX_RGBA_TYPE, NULL, True);
}

static int makeCurrent(Display* dpy, GLXDrawable drawable, GLXContext ctx) {
    return glXMakeCurrent(dpy, drawable, ctx);
}

static void swapBuffers(Display* dpy, GLXDrawable drawable) {
    glXSwapBuffers(dpy, drawable);
}

static void destroyCtx(Display* dpy, GLXContext ctx) {
    glXDestroyContext(dpy, ctx);
}
*/
import "C"
import (
	"fmt"
	"unsafe"
)

// GLContext holds a GLX rendering context.
type GLContext struct {
	display  *C.Display
	drawable C.GLXDrawable
	ctx      C.GLXContext
	cfg      C.GLXFBConfig
}

// CreateContext initialises an OpenGL context on an X11 window.
// display and xwin are the unsafe.Pointer values from platform/x11.go.
func CreateContext(display unsafe.Pointer, xwin uintptr, screenNum int) (*GLContext, error) {
	dpy := (*C.Display)(display)
	screen := C.int(screenNum)

	cfg := C.chooseFBConfig(dpy, screen)
	if cfg == nil {
		return nil, fmt.Errorf("glx: no suitable FBConfig found")
	}

	ctx := C.createContext(dpy, cfg)
	if ctx == nil {
		return nil, fmt.Errorf("glx: glXCreateNewContext failed")
	}

	drawable := C.GLXDrawable(xwin)
	if C.makeCurrent(dpy, drawable, ctx) == 0 {
		return nil, fmt.Errorf("glx: glXMakeCurrent failed")
	}

	return &GLContext{
		display:  dpy,
		drawable: drawable,
		ctx:      ctx,
		cfg:      cfg,
	}, nil
}

// MakeCurrent binds this context to the current OS thread.
func (g *GLContext) MakeCurrent() {
	C.makeCurrent(g.display, g.drawable, g.ctx)
}

// SwapBuffers presents the rendered frame.
func (g *GLContext) SwapBuffers() {
	C.swapBuffers(g.display, g.drawable)
}

// Destroy frees the GL context.
func (g *GLContext) Destroy() {
	C.destroyCtx(g.display, g.ctx)
}
