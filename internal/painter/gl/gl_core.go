// +build !gles,!arm,!arm64,!android,!ios,!mobile

package gl

import (
	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/mathgl/mgl64"
	glu "github.com/maxfish/gl_utils/gl_utils"
	"image"
	"image/draw"

	"fyne.io/fyne"
	"fyne.io/fyne/theme"
)

// Buffer represents a GL buffer
type Buffer uint32

// Program represents a compiled GL program
type Program uint32

var shader glu.ShaderProgram
var quad *glu.Primitive2D
var camera *glu.Camera2D

type Texture uint32

var NoTexture = Texture(0)

func newTexture() Texture {
	var texture uint32
	gl.GenTextures(1, &texture)
	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, texture)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)

	return Texture(texture)
}

func (p *glPainter) imgToTexture(img image.Image) Texture {

	switch i := img.(type) {
	case *image.Uniform:
		// TODO: Color!
		//t, _ := gl_utils.NewEmptyTexture(1,1, gl.RGBA)
		texture := newTexture()
		r, g, b, a := i.RGBA()
		r8, g8, b8, a8 := uint8(r>>8), uint8(g>>8), uint8(b>>8), uint8(a>>8)
		data := []uint8{r8, g8, b8, a8}
		gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA, 1, 1, 0, gl.RGBA,
			gl.UNSIGNED_BYTE, gl.Ptr(data))
		return texture
	case *image.RGBA:
		if len(i.Pix) == 0 { // image is empty
			return 0
		}
		//t, _ := gl_utils.NewEmptyTexture(i.Rect.Size().X, i.Rect.Size().Y, gl.RGBA)
		texture := newTexture()
		gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA, int32(i.Rect.Size().X), int32(i.Rect.Size().Y),
			0, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(i.Pix))
		return texture
	default:
		rgba := image.NewRGBA(image.Rect(0, 0, img.Bounds().Dx(), img.Bounds().Dy()))
		draw.Draw(rgba, rgba.Rect, img, image.ZP, draw.Over)
		return p.imgToTexture(rgba)
	}
}

func (p *glPainter) SetOutputSize(width, height int) {
	gl.Viewport(0, 0, int32(width), int32(height))
	camera = glu.NewCamera2D(width, height, 1)
	//camera.SetCentered(true)
}

func (p *glPainter) freeTexture(obj fyne.CanvasObject) {
	texture := textures[obj]
	if texture != 0 {
		tex := uint32(texture)
		gl.DeleteTextures(1, &tex)
		delete(textures, obj)
	}
}

func glInit() {
	err := gl.Init()
	if err != nil {
		fyne.LogError("failed to initialise OpenGL", err)
		return
	}

	gl.Disable(gl.DEPTH_TEST)
	gl.Enable(gl.BLEND)
}

func (p *glPainter) Init() {
	quad = glu.NewQuadPrimitive(mgl64.Vec3{0, 0, 0}, mgl64.Vec2{1, 1})
}

func (p *glPainter) glClearBuffer() {
	gl.UseProgram(uint32(p.program))

	r, g, b, a := theme.BackgroundColor().RGBA()
	max16bit := float32(255 * 255)
	gl.ClearColor(float32(r)/max16bit, float32(g)/max16bit, float32(b)/max16bit, float32(a)/max16bit)
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
}

func (p *glPainter) glScissorOpen(x, y, w, h int32) {
	gl.Scissor(x, y, w, h)
	gl.Enable(gl.SCISSOR_TEST)
}

func (p *glPainter) glScissorClose() {
	gl.Disable(gl.SCISSOR_TEST)
}

var position mgl64.Vec3
var size mgl64.Vec2
func (p *glPainter) glCreateBuffer(points []float32) Buffer {
	// The coordinates are now based on the camera and the transformations need to be reversed

	//size, pos, frame, fill, aspect, pad)
	//	return []float32{
	//		// coord x, y, z texture x, y
	//		x1, y2, 0, 0.0, 1.0, // top left
	//		x1, y1, 0, 0.0, 0.0, // bottom left
	//		x2, y2, 0, 1.0, 1.0, // top right
	//		x2, y1, 0, 1.0, 0.0, // bottom right
	//	}
	// Solutions
	// pos.X = ((x1 + 1) / 2) * frame.Width + pad
	// pos.Y = (1-y1)*frame.Height/2 + pad
	// width = (x2 + 1)*frame.Width/2 -pad - pos.X
	// height = (1-y2)*frame.Height/2 - pos.Y - pad

	posX := float64(points[0]+1)*camera.Width()/2 + 0
	posY := float64(1-points[6])*camera.Height()/2 + 0
	position = mgl64.Vec3{posX, posY, 0}
	sizeW := float64(points[10] + 1)*camera.Width()/2 - 0 - posX
	sizeH := float64(1-points[1])*camera.Height()/2 - 0 - posY
	size = mgl64.Vec2{sizeW, sizeH}

	return 0
}

func (p *glPainter) glFreeBuffer(vbo Buffer) {
	// Nothing to do here, the buffer is reused
}

func (p *glPainter) glDrawTexture(texture Texture, alpha float32) {
	// here we have to choose between blending the image alpha or fading it...
	// TODO find a way to support both
	if alpha != 1.0 {
		gl.BlendColor(0, 0, 0, alpha)
		gl.BlendFunc(gl.CONSTANT_ALPHA, gl.ONE_MINUS_CONSTANT_ALPHA)
	} else {
		gl.BlendFunc(gl.ONE, gl.ONE_MINUS_SRC_ALPHA)
	}

	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, uint32(texture))
	quad.SetPosition(position)
	quad.SetScale(size)
	quad.Draw(camera.ProjectionMatrix32())
}

func (p *glPainter) glCapture(width, height int32, pixels *[]uint8) {
	gl.ReadBuffer(gl.FRONT)
	gl.ReadPixels(0, 0, int32(width), int32(height), gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(*pixels))
}
