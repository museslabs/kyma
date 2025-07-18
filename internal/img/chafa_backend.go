package img

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/ploMP4/chafa-go"
)

type chafaCache struct {
	cache map[string]string
}

func (c *chafaCache) key(path string, width, height int32, symbols bool) string {
	return strings.Join(
		[]string{
			path,
			strconv.Itoa(int(width)),
			strconv.Itoa(int(height)),
			strconv.FormatBool(symbols),
		},
		"|",
	)
}

func (c *chafaCache) Get(path string, width, height int32, symbols bool) string {
	img, ok := c.cache[c.key(path, width, height, symbols)]
	if !ok {
		return ""
	}
	return img
}

func (c *chafaCache) Save(img, path string, width, height int32, symbols bool) {
	c.cache[c.key(path, width, height, symbols)] = img
}

type chafaBackend struct {
	cache *chafaCache
}

const nChannels = 4

func NewChafaBackend() *chafaBackend {
	return &chafaBackend{
		cache: &chafaCache{
			cache: map[string]string{},
		},
	}
}

func (b *chafaBackend) Render(path string, width, height int32, symbols bool) (string, error) {
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

	c := b.cache.Get(path, width, height, symbols)
	if c != "" {
		return c, nil
	}

	out, err := b.render(path, width, height, symbols)
	b.cache.Save(out, path, width, height, symbols)

	return out, err
}

func (b chafaBackend) render(path string, width, height int32, symbols bool) (string, error) {
	pixels, pixelWidth, pixelHeight, err := chafa.Load(path)
	if err != nil {
		return "", err
	}

	chafa.CalcCanvasGeometry(width, height, &width, &height, 1, true, false)

	capabilities := b.detectTerminal()
	defer chafa.SymbolMapUnref(capabilities.symbolMap)
	defer chafa.TermInfoUnref(capabilities.termInfo)

	config := chafa.CanvasConfigNew()
	defer chafa.CanvasConfigUnref(config)

	chafa.CanvasConfigSetCanvasMode(config, capabilities.canvasMode)
	chafa.CanvasConfigSetGeometry(config, width, height)
	chafa.CanvasConfigSetPassthrough(config, capabilities.passthrough)
	chafa.CanvasConfigSetSymbolMap(config, capabilities.symbolMap)
	chafa.CanvasConfigSetCellGeometry(config, 18, 36)

	if symbols {
		chafa.CanvasConfigSetPixelMode(config, chafa.CHAFA_PIXEL_MODE_SYMBOLS)
	} else {
		chafa.CanvasConfigSetPixelMode(config, capabilities.pixelMode)
	}

	canvas := chafa.CanvasNew(config)
	defer chafa.CanvasUnRef(canvas)

	chafa.CanvasDrawAllPixels(
		canvas,
		chafa.CHAFA_PIXEL_RGBA8_UNASSOCIATED,
		pixels,
		pixelWidth,
		pixelHeight,
		pixelWidth*nChannels,
	)
	printable := chafa.CanvasPrint(canvas, nil)

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
