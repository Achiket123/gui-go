package gl

/*
#cgo linux LDFLAGS: -lGL
#cgo darwin LDFLAGS: -framework OpenGL
#cgo windows LDFLAGS: -lopengl32

#define GL_GLEXT_PROTOTYPES 1
#if defined(__APPLE__)
#include <OpenGL/gl.h>
#include <OpenGL/glext.h>
#else
#include <GL/gl.h>
#include <GL/glext.h>
#endif
#include <stdlib.h>
#include <string.h>

extern void* getGLProcAddress(char* name);


// --- Extension function pointers ---
// We load them at runtime via glXGetProcAddress (done from Go side).
// Declare function pointer types for GLSL-era extensions:
typedef void  (*PFNGLGENBUFFERS)(GLsizei, GLuint*);
typedef void  (*PFNGLBINDBUFFER)(GLenum, GLuint);
typedef void  (*PFNGLBUFFERDATA)(GLenum, GLsizeiptr, const void*, GLenum);
typedef void  (*PFNGLDELETEBUFFERS)(GLsizei, const GLuint*);
typedef void  (*PFNGLGENVERTEXARRAYS)(GLsizei, GLuint*);
typedef void  (*PFNGLBINDVERTEXARRAY)(GLuint);
typedef void  (*PFNGLDELETEVERTEXARRAYS)(GLsizei, const GLuint*);
typedef void  (*PFNGLVERTEXATTRIBPOINTER)(GLuint, GLint, GLenum, GLboolean, GLsizei, const void*);
typedef void  (*PFNGLENABLEVERTEXATTRIBARRAY)(GLuint);
typedef GLuint (*PFNGLCREATESHADER)(GLenum);
typedef void  (*PFNGLSHADERSOURCE)(GLuint, GLsizei, const GLchar**, const GLint*);
typedef void  (*PFNGLCOMPILESHADER)(GLuint);
typedef void  (*PFNGLGETSHADERIV)(GLuint, GLenum, GLint*);
typedef void  (*PFNGLGETSHADERINFOLOG)(GLuint, GLsizei, GLsizei*, GLchar*);
typedef void  (*PFNGLDELETESHADER)(GLuint);
typedef GLuint (*PFNGLCREATEPROGRAM)(void);
typedef void  (*PFNGLATTACHSHADER)(GLuint, GLuint);
typedef void  (*PFNGLLINKPROGRAM)(GLuint);
typedef void  (*PFNGLGETPROGRAMIV)(GLuint, GLenum, GLint*);
typedef void  (*PFNGLGETPROGRAMINFOLOG)(GLuint, GLsizei, GLsizei*, GLchar*);
typedef void  (*PFNGLUSEPROGRAM)(GLuint);
typedef void  (*PFNGLDELETEPROGRAM)(GLuint);
typedef GLint (*PFNGLGETUNIFORMLOCATION)(GLuint, const GLchar*);
typedef void  (*PFNGLUNIFORM1I)(GLint, GLint);
typedef void  (*PFNGLUNIFORM1F)(GLint, GLfloat);
typedef void  (*PFNGLUNIFORM2F)(GLint, GLfloat, GLfloat);
typedef void  (*PFNGLUNIFORM4F)(GLint, GLfloat, GLfloat, GLfloat, GLfloat);
typedef void  (*PFNGLUNIFORMMATRIX3FV)(GLint, GLsizei, GLboolean, const GLfloat*);
typedef void  (*PFNGLUNIFORMMATRIX4FV)(GLint, GLsizei, GLboolean, const GLfloat*);
typedef void  (*PFNGLACTIVETEXTURE_FN)(GLenum);
typedef void  (*PFNGLGENERATEFRAGMENTSAMPLEMASKSNV)(GLuint, GLbitfield);
typedef void  (*PFNGLDRAWELEMENTSBASEVERTEX)(GLenum, GLsizei, GLenum, const void*, GLint);
typedef void  (*PFNGLBLENDFUNCSEPARATE)(GLenum, GLenum, GLenum, GLenum);

// Global function pointers filled in by goGLInit().
static PFNGLGENBUFFERS           pGenBuffers;
static PFNGLBINDBUFFER           pBindBuffer;
static PFNGLBUFFERDATA           pBufferData;
static PFNGLDELETEBUFFERS        pDeleteBuffers;
static PFNGLGENVERTEXARRAYS      pGenVertexArrays;
static PFNGLBINDVERTEXARRAY      pBindVertexArray;
static PFNGLDELETEVERTEXARRAYS   pDeleteVertexArrays;
static PFNGLVERTEXATTRIBPOINTER  pVertexAttribPointer;
static PFNGLENABLEVERTEXATTRIBARRAY pEnableVertexAttribArray;
static PFNGLCREATESHADER         pCreateShader;
static PFNGLSHADERSOURCE         pShaderSource;
static PFNGLCOMPILESHADER        pCompileShader;
static PFNGLGETSHADERIV          pGetShaderiv;
static PFNGLGETSHADERINFOLOG     pGetShaderInfoLog;
static PFNGLDELETESHADER         pDeleteShader;
static PFNGLCREATEPROGRAM        pCreateProgram;
static PFNGLATTACHSHADER         pAttachShader;
static PFNGLLINKPROGRAM          pLinkProgram;
static PFNGLGETPROGRAMIV         pGetProgramiv;
static PFNGLGETPROGRAMINFOLOG    pGetProgramInfoLog;
static PFNGLUSEPROGRAM           pUseProgram;
static PFNGLDELETEPROGRAM        pDeleteProgram;
static PFNGLGETUNIFORMLOCATION   pGetUniformLocation;
static PFNGLUNIFORM1I            pUniform1i;
static PFNGLUNIFORM1F            pUniform1f;
static PFNGLUNIFORM2F            pUniform2f;
static PFNGLUNIFORM4F            pUniform4f;
static PFNGLUNIFORMMATRIX3FV     pUniformMatrix3fv;
static PFNGLUNIFORMMATRIX4FV     pUniformMatrix4fv;
static PFNGLACTIVETEXTURE_FN     pActiveTexture;
static PFNGLBLENDFUNCSEPARATE    pBlendFuncSeparate;

#define LOAD(name, type, sym) name = (type)(uintptr_t)getGLProcAddress(sym)

static void goGLInit(void) {
    LOAD(pGenBuffers,              PFNGLGENBUFFERS,              "glGenBuffers");
    LOAD(pBindBuffer,              PFNGLBINDBUFFER,              "glBindBuffer");
    LOAD(pBufferData,              PFNGLBUFFERDATA,              "glBufferData");
    LOAD(pDeleteBuffers,           PFNGLDELETEBUFFERS,           "glDeleteBuffers");
    LOAD(pGenVertexArrays,         PFNGLGENVERTEXARRAYS,         "glGenVertexArrays");
    LOAD(pBindVertexArray,         PFNGLBINDVERTEXARRAY,         "glBindVertexArray");
    LOAD(pDeleteVertexArrays,      PFNGLDELETEVERTEXARRAYS,      "glDeleteVertexArrays");
    LOAD(pVertexAttribPointer,     PFNGLVERTEXATTRIBPOINTER,     "glVertexAttribPointer");
    LOAD(pEnableVertexAttribArray, PFNGLENABLEVERTEXATTRIBARRAY, "glEnableVertexAttribArray");
    LOAD(pCreateShader,            PFNGLCREATESHADER,            "glCreateShader");
    LOAD(pShaderSource,            PFNGLSHADERSOURCE,            "glShaderSource");
    LOAD(pCompileShader,           PFNGLCOMPILESHADER,           "glCompileShader");
    LOAD(pGetShaderiv,             PFNGLGETSHADERIV,             "glGetShaderiv");
    LOAD(pGetShaderInfoLog,        PFNGLGETSHADERINFOLOG,        "glGetShaderInfoLog");
    LOAD(pDeleteShader,            PFNGLDELETESHADER,            "glDeleteShader");
    LOAD(pCreateProgram,           PFNGLCREATEPROGRAM,           "glCreateProgram");
    LOAD(pAttachShader,            PFNGLATTACHSHADER,            "glAttachShader");
    LOAD(pLinkProgram,             PFNGLLINKPROGRAM,             "glLinkProgram");
    LOAD(pGetProgramiv,            PFNGLGETPROGRAMIV,            "glGetProgramiv");
    LOAD(pGetProgramInfoLog,       PFNGLGETPROGRAMINFOLOG,       "glGetProgramInfoLog");
    LOAD(pUseProgram,              PFNGLUSEPROGRAM,              "glUseProgram");
    LOAD(pDeleteProgram,           PFNGLDELETEPROGRAM,           "glDeleteProgram");
    LOAD(pGetUniformLocation,      PFNGLGETUNIFORMLOCATION,      "glGetUniformLocation");
    LOAD(pUniform1i,               PFNGLUNIFORM1I,               "glUniform1i");
    LOAD(pUniform1f,               PFNGLUNIFORM1F,               "glUniform1f");
    LOAD(pUniform2f,               PFNGLUNIFORM2F,               "glUniform2f");
    LOAD(pUniform4f,               PFNGLUNIFORM4F,               "glUniform4f");
    LOAD(pUniformMatrix3fv,        PFNGLUNIFORMMATRIX3FV,        "glUniformMatrix3fv");
    LOAD(pUniformMatrix4fv,        PFNGLUNIFORMMATRIX4FV,        "glUniformMatrix4fv");
    LOAD(pActiveTexture,           PFNGLACTIVETEXTURE_FN,        "glActiveTexture");
    LOAD(pBlendFuncSeparate,       PFNGLBLENDFUNCSEPARATE,       "glBlendFuncSeparate");
}

// --- Thin wrappers (called from Go) ---

static GLuint compileShader(GLenum type, const char* src) {
    GLuint s = pCreateShader(type);
    pShaderSource(s, 1, &src, NULL);
    pCompileShader(s);
    return s;
}
static int shaderOK(GLuint s) {
    GLint ok; pGetShaderiv(s, GL_COMPILE_STATUS, &ok); return ok;
}
static void shaderLog(GLuint s, char* buf, int len) {
    pGetShaderInfoLog(s, len, NULL, buf);
}
static GLuint linkProgram(GLuint vs, GLuint fs) {
    GLuint p = pCreateProgram();
    pAttachShader(p, vs); pAttachShader(p, fs);
    pLinkProgram(p);
    return p;
}
static int programOK(GLuint p) {
    GLint ok; pGetProgramiv(p, GL_LINK_STATUS, &ok); return ok;
}
static void programLog(GLuint p, char* buf, int len) {
    pGetProgramInfoLog(p, len, NULL, buf);
}
static void useProgram(GLuint p) { pUseProgram(p); }
static void deleteShader(GLuint s) { pDeleteShader(s); }
static void deleteProgram(GLuint p) { pDeleteProgram(p); }
static GLint uniformLoc(GLuint p, const char* name) { return pGetUniformLocation(p, name); }
static void uniform1i(GLint loc, GLint v) { pUniform1i(loc, v); }
static void uniform1f(GLint loc, GLfloat v) { pUniform1f(loc, v); }
static void uniform2f(GLint loc, GLfloat x, GLfloat y) { pUniform2f(loc, x, y); }
static void uniform4f(GLint loc, GLfloat r, GLfloat g, GLfloat b, GLfloat a) { pUniform4f(loc, r, g, b, a); }
static void uniformMat3(GLint loc, const GLfloat* m) { pUniformMatrix3fv(loc, 1, GL_FALSE, m); }
static void uniformMat4(GLint loc, const GLfloat* m) { pUniformMatrix4fv(loc, 1, GL_FALSE, m); }

// --- VBO/VAO ---
static GLuint genBuffer(void) { GLuint b; pGenBuffers(1, &b); return b; }
static void bindArrayBuffer(GLuint b) { pBindBuffer(GL_ARRAY_BUFFER, b); }
static void bindElementBuffer(GLuint b) { pBindBuffer(GL_ELEMENT_ARRAY_BUFFER, b); }
static void bufferDataArraysDynamic(GLsizeiptr sz, const void* data) {
    pBufferData(GL_ARRAY_BUFFER, sz, data, GL_DYNAMIC_DRAW);
}
static void bufferDataElementsDynamic(GLsizeiptr sz, const void* data) {
    pBufferData(GL_ELEMENT_ARRAY_BUFFER, sz, data, GL_DYNAMIC_DRAW);
}
static void deleteBuffer(GLuint b) { pDeleteBuffers(1, &b); }

static GLuint genVAO(void) { GLuint v; pGenVertexArrays(1, &v); return v; }
static void bindVAO(GLuint v) { pBindVertexArray(v); }
static void deleteVAO(GLuint v) { pDeleteVertexArrays(1, &v); }

// stride/offset in bytes
static void attribPointerF(GLuint idx, GLint size, GLsizei stride, GLsizeiptr offset) {
    pVertexAttribPointer(idx, size, GL_FLOAT, GL_FALSE, stride, (const void*)offset);
    pEnableVertexAttribArray(idx);
}

// --- Texture ---
static GLuint genTex(void) { GLuint t; glGenTextures(1, &t); return t; }
static void deleteTex(GLuint t) { glDeleteTextures(1, &t); }
static void bindTex2D(GLuint t) { glBindTexture(GL_TEXTURE_2D, t); }
static void uploadTex(int w, int h, const void* pix) {
    glTexImage2D(GL_TEXTURE_2D, 0, GL_RGBA8, w, h, 0, GL_RGBA, GL_UNSIGNED_BYTE, pix);
    glTexParameteri(GL_TEXTURE_2D, GL_TEXTURE_MIN_FILTER, GL_LINEAR);
    glTexParameteri(GL_TEXTURE_2D, GL_TEXTURE_MAG_FILTER, GL_LINEAR);
    glTexParameteri(GL_TEXTURE_2D, GL_TEXTURE_WRAP_S, GL_CLAMP_TO_EDGE);
    glTexParameteri(GL_TEXTURE_2D, GL_TEXTURE_WRAP_T, GL_CLAMP_TO_EDGE);
}
static void activeTex0(void) { pActiveTexture(GL_TEXTURE0); }
static void drawElements(GLsizei count) {
    glDrawElements(GL_TRIANGLES, count, GL_UNSIGNED_INT, 0);
}
static void setViewport(int w, int h) { glViewport(0, 0, w, h); }
static void clearColor(float r, float g, float b, float a) { glClearColor(r,g,b,a); glClear(GL_COLOR_BUFFER_BIT); }
static void enableBlend(void) {
    glEnable(GL_BLEND);
    pBlendFuncSeparate(GL_SRC_ALPHA, GL_ONE_MINUS_SRC_ALPHA, GL_ONE, GL_ONE_MINUS_SRC_ALPHA);
}
static void scissorOn(int x, int y, int w, int h) {
    glEnable(GL_SCISSOR_TEST);
    glScissor(x, y, w, h);
}
static void scissorOff(void) { glDisable(GL_SCISSOR_TEST); }
*/
import "C"

import (
	"fmt"
	"unsafe"
)

// InitExtensions loads all OpenGL extension function pointers.
// Must be called after MakeCurrent().
func InitExtensions(loader func(string) unsafe.Pointer) {
	getProc = loader
	C.goGLInit()
}

// --- Shader ---

// CompileProgram compiles vertex + fragment GLSL source and links them.
func CompileProgram(vertSrc, fragSrc string) (uint32, error) {
	cvs := C.CString(vertSrc)
	defer C.free(unsafe.Pointer(cvs))
	cfs := C.CString(fragSrc)
	defer C.free(unsafe.Pointer(cfs))

	vs := C.compileShader(C.GL_VERTEX_SHADER, cvs)
	if C.shaderOK(vs) == 0 {
		var buf [512]C.char
		C.shaderLog(vs, &buf[0], 512)
		C.deleteShader(vs)
		return 0, fmt.Errorf("vertex shader: %s", C.GoString(&buf[0]))
	}
	fs := C.compileShader(C.GL_FRAGMENT_SHADER, cfs)
	if C.shaderOK(fs) == 0 {
		var buf [512]C.char
		C.shaderLog(fs, &buf[0], 512)
		C.deleteShader(vs)
		C.deleteShader(fs)
		return 0, fmt.Errorf("fragment shader: %s", C.GoString(&buf[0]))
	}
	prog := C.linkProgram(vs, fs)
	C.deleteShader(vs)
	C.deleteShader(fs)
	if C.programOK(prog) == 0 {
		var buf [512]C.char
		C.programLog(prog, &buf[0], 512)
		C.deleteProgram(prog)
		return 0, fmt.Errorf("link: %s", C.GoString(&buf[0]))
	}
	return uint32(prog), nil
}

// UseProgram binds a shader program.
func UseProgram(prog uint32) { C.useProgram(C.GLuint(prog)) }

// DeleteProgram frees a shader program.
func DeleteProgram(prog uint32) { C.deleteProgram(C.GLuint(prog)) }

// UniformLoc returns the location of a uniform variable.
func UniformLoc(prog uint32, name string) int32 {
	cs := C.CString(name)
	defer C.free(unsafe.Pointer(cs))
	return int32(C.uniformLoc(C.GLuint(prog), cs))
}

func Uniform1i(loc int32, v int32)      { C.uniform1i(C.GLint(loc), C.GLint(v)) }
func Uniform1f(loc int32, v float32)    { C.uniform1f(C.GLint(loc), C.GLfloat(v)) }
func Uniform2f(loc int32, x, y float32) { C.uniform2f(C.GLint(loc), C.GLfloat(x), C.GLfloat(y)) }
func Uniform4f(loc int32, r, g, b, a float32) {
	C.uniform4f(C.GLint(loc), C.GLfloat(r), C.GLfloat(g), C.GLfloat(b), C.GLfloat(a))
}
func UniformMat3(loc int32, m *[9]float32) {
	C.uniformMat3(C.GLint(loc), (*C.GLfloat)(unsafe.Pointer(m)))
}
func UniformMat4(loc int32, m *[16]float32) {
	C.uniformMat4(C.GLint(loc), (*C.GLfloat)(unsafe.Pointer(m)))
}

// --- VBO/VAO ---

func GenBuffer() uint32          { return uint32(C.genBuffer()) }
func BindArrayBuffer(b uint32)   { C.bindArrayBuffer(C.GLuint(b)) }
func BindElementBuffer(b uint32) { C.bindElementBuffer(C.GLuint(b)) }
func BufferArraysDynamic(data []float32) {
	if len(data) == 0 {
		return
	}
	C.bufferDataArraysDynamic(C.GLsizeiptr(len(data)*4), unsafe.Pointer(&data[0]))
}
func BufferElementsDynamic(data []uint32) {
	if len(data) == 0 {
		return
	}
	C.bufferDataElementsDynamic(C.GLsizeiptr(len(data)*4), unsafe.Pointer(&data[0]))
}
func DeleteBuffer(b uint32) { C.deleteBuffer(C.GLuint(b)) }

func GenVAO() uint32     { return uint32(C.genVAO()) }
func BindVAO(v uint32)   { C.bindVAO(C.GLuint(v)) }
func DeleteVAO(v uint32) { C.deleteVAO(C.GLuint(v)) }

// AttribPointerF sets a float vertex attribute pointer.
// stride and offset are in bytes.
func AttribPointerF(idx, size int, stride, offset int) {
	C.attribPointerF(C.GLuint(idx), C.GLint(size), C.GLsizei(stride), C.GLsizeiptr(offset))
}

// --- Textures ---

func GenTexture() uint32     { return uint32(C.genTex()) }
func DeleteTexture(t uint32) { C.deleteTex(C.GLuint(t)) }
func BindTexture2D(t uint32) { C.bindTex2D(C.GLuint(t)) }
func ActiveTexture0()        { C.activeTex0() }
func UploadTextureRGBA(w, h int, pix []byte) {
	C.uploadTex(C.int(w), C.int(h), unsafe.Pointer(&pix[0]))
}

// --- Draw ---

func DrawTriangleElements(count int) { C.drawElements(C.GLsizei(count)) }
func SetViewport(w, h int)           { C.setViewport(C.int(w), C.int(h)) }
func ClearColor(r, g, b, a float32)  { C.clearColor(C.float(r), C.float(g), C.float(b), C.float(a)) }
func EnableBlend()                   { C.enableBlend() }
func ScissorOn(x, y, w, h int)       { C.scissorOn(C.int(x), C.int(y), C.int(w), C.int(h)) }
func ScissorOff()                    { C.scissorOff() }
