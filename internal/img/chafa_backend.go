package img

import (
	"fmt"
	"os"

	"github.com/ploMP4/chafa-go"
)

type chafaBackend struct{}

const nChannels = 4

func (b chafaBackend) Render(path string, width, height int32) (string, error) {
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

	out, err := b.render(path, width, height)
	return out, err
}

func (b chafaBackend) render(path string, width, height int32) (string, error) {
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
	chafa.CanvasConfigSetPixelMode(config, capabilities.pixelMode)
	chafa.CanvasConfigSetPassthrough(config, capabilities.passthrough)
	chafa.CanvasConfigSetSymbolMap(config, capabilities.symbolMap)

	widthNew := config.Width
	heightNew := config.Height

	chafa.CalcCanvasGeometry(
		width,
		height,
		&widthNew,
		&heightNew,
		0.5,
		true,
		false,
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
