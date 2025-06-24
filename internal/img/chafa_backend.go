package img

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/ploMP4/chafa-go"
)

type chafaBackend struct{}

const nChannels = 4

func (b chafaBackend) Render(path string, width, height int32, animating bool) (string, error) {
	var err error

	defer func() {
		if r := recover(); r != nil {
			if recErr, ok := r.(error); ok {
				err = recErr
			} else {
				err = fmt.Errorf("panic: %v", r)
			}
		}
	}()

	out, err := b.render(path, width, height, animating)
	return out, err
}

func (b chafaBackend) render(path string, width, height int32, animating bool) (string, error) {
	pixels, pixelWidth, pixelHeight, err := chafa.Load(path)
	if err != nil {
		return "", err
	}

	capabilities := b.detectTerminal()
	defer chafa.SymbolMapUnref(capabilities.symbolMap)
	defer chafa.TermInfoUnref(capabilities.termInfo)

	config := chafa.CanvasConfigNew()
	defer chafa.CanvasConfigUnref(config)

	chafa.CanvasConfigSetGeometry(config, width, height)
	chafa.CanvasConfigSetCellGeometry(config, 18, 36)
	chafa.CanvasConfigSetCanvasMode(config, capabilities.canvasMode)
	chafa.CanvasConfigSetPassthrough(config, capabilities.passthrough)
	chafa.CanvasConfigSetSymbolMap(config, capabilities.symbolMap)

	var pixelModeUsed chafa.PixelMode
	if animating {
		pixelModeUsed = chafa.CHAFA_PIXEL_MODE_SYMBOLS
		chafa.CanvasConfigSetPixelMode(config, chafa.CHAFA_PIXEL_MODE_SYMBOLS)
	} else {
		pixelModeUsed = capabilities.pixelMode
		chafa.CanvasConfigSetPixelMode(config, capabilities.pixelMode)
	}

	widthNew := config.Width
	heightNew := config.Height

	slog.Info("Chafa render start",
		"path", path,
		"animating", animating,
		"requested_width", width,
		"requested_height", height,
		"initial_width", widthNew,
		"initial_height", heightNew,
		"pixel_mode", pixelModeUsed,
		"canvas_mode", capabilities.canvasMode,
		"image_dimensions", fmt.Sprintf("%dx%d", pixelWidth, pixelHeight),
	)

	chafa.CalcCanvasGeometry(
		width,
		height,
		&widthNew,
		&heightNew,
		0.5,
		true,
		true,
	)

	slog.Info("Chafa geometry calculated",
		"path", path,
		"animating", animating,
		"final_width", widthNew,
		"final_height", heightNew,
	)

	chafa.CanvasConfigSetGeometry(config, widthNew, heightNew)

	canvas := chafa.CanvasNew(config)
	defer chafa.CanvasUnRef(canvas)

	frame := chafa.FrameNew(
		pixels,
		chafa.CHAFA_PIXEL_RGBA8_UNASSOCIATED,
		pixelWidth,
		pixelHeight,
		pixelWidth*nChannels,
	)
	defer chafa.FrameUnref(frame)

	img := chafa.ImageNew()
	defer chafa.ImageUnref(img)

	chafa.ImageSetFrame(img, frame)

	placement := chafa.PlacementNew(img, 1)
	defer chafa.PlacementUnref(placement)

	chafa.PlacementSetTuck(placement, chafa.CHAFA_TUCK_FIT)

	chafa.CanvasSetPlacement(canvas, placement)
	printable := chafa.CanvasPrint(canvas, capabilities.termInfo)

	return printable.String(), nil
}

type terminalCapabilities struct {
	termInfo    *chafa.TermInfo
	canvasMode  chafa.CanvasMode
	pixelMode   chafa.PixelMode
	passthrough chafa.Passthrough
	symbolMap   *chafa.SymbolMap
}

func (b chafaBackend) detectTerminal() terminalCapabilities {
	termInfo := chafa.TermDbDetect(chafa.TermDbGetDefault(), os.Environ())

	mode := chafa.TermInfoGetBestCanvasMode(termInfo)
	pixelMode := chafa.TermInfoGetBestPixelMode(termInfo)

	passthrough := chafa.CHAFA_PASSTHROUGH_NONE
	if chafa.TermInfoGetIsPixelPassthroughNeeded(termInfo, pixelMode) {
		passthrough = chafa.TermInfoGetPassthroughType(termInfo)
	}

	symbolMap := chafa.SymbolMapNew()
	chafa.SymbolMapAddByTags(symbolMap, chafa.TermInfoGetSafeSymbolTags(termInfo))

	return terminalCapabilities{
		termInfo:    termInfo,
		canvasMode:  mode,
		pixelMode:   pixelMode,
		passthrough: passthrough,
		symbolMap:   symbolMap,
	}
}
