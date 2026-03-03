package canvas

import "math"

// Mat3 is a 3×3 column-major affine transform matrix.
// Layout: [col0, col1, col2], each column is 3 floats.
type Mat3 [9]float32

// Identity returns the identity matrix.
func Identity() Mat3 {
	return Mat3{1, 0, 0, 0, 1, 0, 0, 0, 1}
}

// Mul multiplies two Mat3 matrices (a * b).
func (a Mat3) Mul(b Mat3) Mat3 {
	return Mat3{
		a[0]*b[0] + a[3]*b[1] + a[6]*b[2],
		a[1]*b[0] + a[4]*b[1] + a[7]*b[2],
		a[2]*b[0] + a[5]*b[1] + a[8]*b[2],

		a[0]*b[3] + a[3]*b[4] + a[6]*b[5],
		a[1]*b[3] + a[4]*b[4] + a[7]*b[5],
		a[2]*b[3] + a[5]*b[4] + a[8]*b[5],

		a[0]*b[6] + a[3]*b[7] + a[6]*b[8],
		a[1]*b[6] + a[4]*b[7] + a[7]*b[8],
		a[2]*b[6] + a[5]*b[7] + a[8]*b[8],
	}
}

// TranslateMatrix returns a translation matrix.
func TranslateMatrix(tx, ty float32) Mat3 {
	return Mat3{1, 0, 0, 0, 1, 0, tx, ty, 1}
}

// RotateMatrix returns a rotation matrix (angle in radians).
func RotateMatrix(angle float32) Mat3 {
	c := float32(math.Cos(float64(angle)))
	s := float32(math.Sin(float64(angle)))
	return Mat3{c, s, 0, -s, c, 0, 0, 0, 1}
}

// ScaleMatrix returns a scale matrix.
func ScaleMatrix(sx, sy float32) Mat3 {
	return Mat3{sx, 0, 0, 0, sy, 0, 0, 0, 1}
}

// transformState holds the save/restore stack entry.
type transformState struct {
	matrix  Mat3
	clipSet bool
	clipX   float32
	clipY   float32
	clipW   float32
	clipH   float32
}

// TransformStack manages a push/pop stack of affine transform + clip states.
type TransformStack struct {
	stack  []transformState
	matrix Mat3
}

// NewTransformStack creates a stack starting at identity.
func NewTransformStack() *TransformStack {
	return &TransformStack{matrix: Identity()}
}

// Current returns the current accumulated transform matrix.
func (ts *TransformStack) Current() Mat3 { return ts.matrix }

// Save pushes the current transform + clip onto the stack.
func (ts *TransformStack) Save(clipSet bool, cx, cy, cw, ch float32) {
	ts.stack = append(ts.stack, transformState{
		matrix:  ts.matrix,
		clipSet: clipSet,
		clipX:   cx, clipY: cy, clipW: cw, clipH: ch,
	})
}

// Restore pops the last saved transform. Returns the restored clip state.
func (ts *TransformStack) Restore() (clipSet bool, cx, cy, cw, ch float32) {
	if len(ts.stack) == 0 {
		ts.matrix = Identity()
		return false, 0, 0, 0, 0
	}
	top := ts.stack[len(ts.stack)-1]
	ts.stack = ts.stack[:len(ts.stack)-1]
	ts.matrix = top.matrix
	return top.clipSet, top.clipX, top.clipY, top.clipW, top.clipH
}

// Translate applies a translation.
func (ts *TransformStack) Translate(x, y float32) {
	ts.matrix = ts.matrix.Mul(TranslateMatrix(x, y))
}

// Rotate applies a rotation around the current origin.
func (ts *TransformStack) Rotate(angle float32) {
	ts.matrix = ts.matrix.Mul(RotateMatrix(angle))
}

// RotateAround rotates around a specific point.
func (ts *TransformStack) RotateAround(cx, cy, angle float32) {
	ts.Translate(cx, cy)
	ts.Rotate(angle)
	ts.Translate(-cx, -cy)
}

// Scale applies independent X/Y scaling.
func (ts *TransformStack) Scale(sx, sy float32) {
	ts.matrix = ts.matrix.Mul(ScaleMatrix(sx, sy))
}

// ScaleUniform applies uniform scaling.
func (ts *TransformStack) ScaleUniform(s float32) {
	ts.Scale(s, s)
}

// ApplyToPoint transforms a point through the current matrix.
func (ts *TransformStack) ApplyToPoint(x, y float32) (float32, float32) {
	m := ts.matrix
	nx := m[0]*x + m[3]*y + m[6]
	ny := m[1]*x + m[4]*y + m[7]
	return nx, ny
}

// ResetTransform resets to identity.
func (ts *TransformStack) ResetTransform() {
	ts.matrix = Identity()
}
