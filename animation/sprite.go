package animation

// Framer is implemented by anything that can provide an image for a given frame.
// goui.Image satisfies this interface.
type Framer interface {
	Width() int
	Height() int
}

// DrawFunc is a function that draws a single sprite frame onto a canvas.
// The canvas parameter is typed as interface{} to avoid an import cycle;
// callers should type-assert to *goui.Canvas.
type DrawFunc func(canvas interface{}, frame Framer, x, y int)

// Sprite cycles through a list of frames at a fixed FPS.
//
// Because the animation package cannot import the goui package (cycle),
// drawing is delegated to a DrawFunc provided by the user.
//
// Example:
//
//	frames := []*goui.Image{img1, img2, img3, img4}
//	frameItems := make([]animation.Framer, len(frames))
//	for i, f := range frames { frameItems[i] = f }
//
//	sp := animation.NewSprite(frameItems, 12)
//	sp.SetDrawFunc(func(cv interface{}, fr animation.Framer, x, y int) {
//	    cv.(*goui.Canvas).DrawImage(fr.(*goui.Image), x, y)
//	})
//	sp.Play()
//	w.AddAnimation(sp)
type Sprite struct {
	frames       []Framer
	fps          float64
	currentFrame int
	elapsed      float64
	looping      bool
	playing      bool
	finished     bool

	drawFn DrawFunc
}

// NewSprite creates a Sprite from a frame list and playback speed (frames/sec).
func NewSprite(frames []Framer, fps float64) *Sprite {
	if fps <= 0 {
		fps = 12
	}
	return &Sprite{
		frames:  frames,
		fps:     fps,
		looping: true,
	}
}

// SetDrawFunc sets the function used to render each frame.
func (s *Sprite) SetDrawFunc(fn DrawFunc) {
	s.drawFn = fn
}

// Play starts or resumes playback.
func (s *Sprite) Play() {
	s.playing = true
	s.finished = false
}

// Stop pauses playback (current frame is preserved).
func (s *Sprite) Stop() {
	s.playing = false
}

// Reset goes back to frame 0.
func (s *Sprite) Reset() {
	s.currentFrame = 0
	s.elapsed = 0
	s.finished = false
}

// SetFPS changes the playback speed.
func (s *Sprite) SetFPS(fps float64) {
	s.fps = fps
}

// SetLoop enables or disables looping.
func (s *Sprite) SetLoop(loop bool) {
	s.looping = loop
}

// CurrentFrame returns the index of the frame currently displayed.
func (s *Sprite) CurrentFrame() int {
	return s.currentFrame
}

// IsFinished returns true when a non-looping sprite has played its last frame.
func (s *Sprite) IsFinished() bool {
	return s.finished
}

// Tick advances the frame timer by delta seconds.
func (s *Sprite) Tick(delta float64) {
	if !s.playing || s.finished || len(s.frames) == 0 {
		return
	}
	s.elapsed += delta
	frameDuration := 1.0 / s.fps
	for s.elapsed >= frameDuration {
		s.elapsed -= frameDuration
		s.currentFrame++
		if s.currentFrame >= len(s.frames) {
			if s.looping {
				s.currentFrame = 0
			} else {
				s.currentFrame = len(s.frames) - 1
				s.finished = true
				s.playing = false
				return
			}
		}
	}
}

// Draw renders the current frame at (x, y) using the registered DrawFunc.
// canvas should be *goui.Canvas.
func (s *Sprite) Draw(canvas interface{}, x, y int) {
	if s.drawFn == nil || len(s.frames) == 0 {
		return
	}
	s.drawFn(canvas, s.frames[s.currentFrame], x, y)
}
