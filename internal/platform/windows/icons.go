//go:build windows

package windows

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"runtime"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	shell32 = windows.NewLazySystemDLL("shell32.dll")
	user32  = windows.NewLazySystemDLL("user32.dll")
	gdi32   = windows.NewLazySystemDLL("gdi32.dll")

	procSHGetFileInfoW     = shell32.NewProc("SHGetFileInfoW")
	procGetIconInfo        = user32.NewProc("GetIconInfo")
	procDestroyIcon        = user32.NewProc("DestroyIcon")
	procCreateCompatibleDC = gdi32.NewProc("CreateCompatibleDC")
	procDeleteDC           = gdi32.NewProc("DeleteDC")
	procGetDIBits          = gdi32.NewProc("GetDIBits")
	procDeleteObject       = gdi32.NewProc("DeleteObject")
	procGetObjectW         = gdi32.NewProc("GetObjectW")
)

// bitmapHeader is BITMAP, read to learn the icon's real size: a "large"
// icon is 48 px on a screen scaled to 150%, not 32.
type bitmapHeader struct {
	typ        int32
	width      int32
	height     int32
	widthBytes int32
	planes     uint16
	bitsPixel  uint16
	bits       uintptr
}

const (
	shgfiIcon    = 0x100
	shgfiLarge   = 0x0 // 32 px or more: drawn at 16, it stays sharp on scaled screens
	maxIconSize  = 256
	dibRGBColors = 0
	biRGB        = 0
)

// shFileInfo is SHFILEINFOW.
type shFileInfo struct {
	icon        windows.Handle
	iconIndex   int32
	attributes  uint32
	displayName [260]uint16
	typeName    [80]uint16
}

// iconInfo is ICONINFO.
type iconInfo struct {
	isIcon   int32
	xHotspot uint32
	yHotspot uint32
	mask     windows.Handle
	color    windows.Handle
}

// bitmapInfoHeader is BITMAPINFOHEADER, followed by room for the palette
// GetDIBits may write for a 1-bit mask.
type bitmapInfo struct {
	size          uint32
	width         int32
	height        int32
	planes        uint16
	bitCount      uint16
	compression   uint32
	sizeImage     uint32
	xPelsPerMeter int32
	yPelsPerMeter int32
	clrUsed       uint32
	clrImportant  uint32
	colors        [256]uint32
}

// IconPNG is the icon Explorer shows for path, as a PNG at the size
// Windows keeps it (32 px, more on scaled screens).
func (System) IconPNG(path string) ([]byte, error) {
	// The shell wants COM on the thread that asks it.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if err := windows.CoInitializeEx(0, windows.COINIT_APARTMENTTHREADED); err == nil {
		defer windows.CoUninitialize()
	}

	p, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return nil, err
	}
	var fi shFileInfo
	if r, _, _ := procSHGetFileInfoW.Call(uintptr(unsafe.Pointer(p)), 0, uintptr(unsafe.Pointer(&fi)), unsafe.Sizeof(fi), shgfiIcon|shgfiLarge); r == 0 || fi.icon == 0 {
		return nil, fmt.Errorf("no icon for %s", path)
	}
	defer procDestroyIcon.Call(uintptr(fi.icon))

	var ii iconInfo
	if r, _, err := procGetIconInfo.Call(uintptr(fi.icon), uintptr(unsafe.Pointer(&ii))); r == 0 {
		return nil, fmt.Errorf("read icon: %w", err)
	}
	defer procDeleteObject.Call(uintptr(ii.mask))
	if ii.color != 0 {
		defer procDeleteObject.Call(uintptr(ii.color))
	}
	if ii.color == 0 {
		return nil, fmt.Errorf("%s has a monochrome icon", path)
	}

	dc, _, _ := procCreateCompatibleDC.Call(0)
	if dc == 0 {
		return nil, fmt.Errorf("create device context")
	}
	defer procDeleteDC.Call(dc)

	var bm bitmapHeader
	if r, _, _ := procGetObjectW.Call(uintptr(ii.color), unsafe.Sizeof(bm), uintptr(unsafe.Pointer(&bm))); r == 0 {
		return nil, fmt.Errorf("read icon size")
	}
	w, h := int(bm.width), int(bm.height)
	if w <= 0 || h <= 0 || w > maxIconSize || h > maxIconSize {
		return nil, fmt.Errorf("icon of %dx%d", w, h)
	}
	pixels, err := dibits(dc, ii.color, w, h)
	if err != nil {
		return nil, err
	}
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	alpha := false
	for i := 0; i < w*h; i++ {
		b, g, r, a := pixels[i*4], pixels[i*4+1], pixels[i*4+2], pixels[i*4+3]
		img.Pix[i*4], img.Pix[i*4+1], img.Pix[i*4+2], img.Pix[i*4+3] = r, g, b, a
		alpha = alpha || a != 0
	}
	// Old icons carry no alpha; their mask says which pixels are see-through.
	if !alpha {
		mask, err := dibits(dc, ii.mask, w, h)
		if err != nil {
			return nil, err
		}
		for i := 0; i < w*h; i++ {
			if mask[i*4] == 0 {
				img.Pix[i*4+3] = 255
			}
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("encode icon: %w", err)
	}
	return buf.Bytes(), nil
}

// dibits reads a w by h bitmap as top-down 32-bit BGRA rows. The buffer is
// sized from the bitmap's own size, so Windows never writes past it.
func dibits(dc uintptr, bmp windows.Handle, w, h int) ([]byte, error) {
	bi := bitmapInfo{width: int32(w), height: -int32(h), planes: 1, bitCount: 32, compression: biRGB}
	bi.size = uint32(unsafe.Sizeof(bi) - unsafe.Sizeof(bi.colors))
	buf := make([]byte, w*h*4)
	if r, _, err := procGetDIBits.Call(dc, uintptr(bmp), 0, uintptr(h), uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&bi)), dibRGBColors); r == 0 {
		return nil, fmt.Errorf("read icon pixels: %w", err)
	}
	return buf, nil
}
