package image

import (
	"bytes"
	"fmt"
	"hash/fnv"
	"image"
	"image/color"
	"io"
	"log/slog"
	"strings"
	"sync"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/ansi/kitty"
	"github.com/disintegration/imaging"
	paintbrush "github.com/jordanella/go-ansi-paintbrush"
	"github.com/purpose168/crush-cn/internal/ui/util"
)

// TransmittedMsg 是一条消息，指示图像已传输到终端。
type TransmittedMsg struct {
	ID string
}

// Encoding 表示图像的编码格式。
type Encoding byte

// 图像编码。
const (
	EncodingBlocks Encoding = iota
	EncodingKitty
)

type imageKey struct {
	id   string
	cols int
	rows int
}

// Hash 返回图像键的哈希值。
// 这使用FNV-32a以实现简单和快速。
func (k imageKey) Hash() uint32 {
	h := fnv.New32a()
	_, _ = io.WriteString(h, k.ID())
	return h.Sum32()
}

// ID 返回图像键的唯一字符串表示。
func (k imageKey) ID() string {
	return fmt.Sprintf("%s-%dx%d", k.id, k.cols, k.rows)
}

// CellSize 表示单个终端单元格的像素大小。
type CellSize struct {
	Width, Height int
}

type cachedImage struct {
	img        image.Image
	cols, rows int
}

var (
	cachedImages = map[imageKey]cachedImage{}
	cachedMutex  sync.RWMutex
)

// ResetCache 清除图像缓存，释放所有缓存的解码图像。
func ResetCache() {
	cachedMutex.Lock()
	clear(cachedImages)
	cachedMutex.Unlock()
}

// fitImage 调整图像大小以适应指定的终端单元格尺寸，
// 同时保持纵横比。
func fitImage(id string, img image.Image, cs CellSize, cols, rows int) image.Image {
	if img == nil {
		return nil
	}

	key := imageKey{id: id, cols: cols, rows: rows}

	cachedMutex.RLock()
	cached, ok := cachedImages[key]
	cachedMutex.RUnlock()
	if ok {
		return cached.img
	}

	if cs.Width == 0 || cs.Height == 0 {
		return img
	}

	maxWidth := cols * cs.Width
	maxHeight := rows * cs.Height

	img = imaging.Fit(img, maxWidth, maxHeight, imaging.Lanczos)

	cachedMutex.Lock()
	cachedImages[key] = cachedImage{
		img:  img,
		cols: cols,
		rows: rows,
	}
	cachedMutex.Unlock()

	return img
}

// HasTransmitted 检查具有给定ID的图像是否已经传输到终端。
func HasTransmitted(id string, cols, rows int) bool {
	key := imageKey{id: id, cols: cols, rows: rows}

	cachedMutex.RLock()
	_, ok := cachedImages[key]
	cachedMutex.RUnlock()
	return ok
}

// Transmit 如果需要则将图像数据传输到终端。这用于
// 在终端上缓存图像以供后续渲染。
func (e Encoding) Transmit(id string, img image.Image, cs CellSize, cols, rows int, tmux bool) tea.Cmd {
	if img == nil {
		return nil
	}

	key := imageKey{id: id, cols: cols, rows: rows}

	cachedMutex.RLock()
	_, ok := cachedImages[key]
	cachedMutex.RUnlock()
	if ok {
		return nil
	}

	cmd := func() tea.Msg {
		if e != EncodingKitty {
			cachedMutex.Lock()
			cachedImages[key] = cachedImage{
				img:  img,
				cols: cols,
				rows: rows,
			}
			cachedMutex.Unlock()
			return TransmittedMsg{ID: key.ID()}
		}

		var buf bytes.Buffer
		img := fitImage(id, img, cs, cols, rows)
		bounds := img.Bounds()
		imgWidth := bounds.Dx()
		imgHeight := bounds.Dy()
		imgID := int(key.Hash())
		if err := kitty.EncodeGraphics(&buf, img, &kitty.Options{
			ID:               imgID,
			Action:           kitty.TransmitAndPut,
			Transmission:     kitty.Direct,
			Format:           kitty.RGBA,
			ImageWidth:       imgWidth,
			ImageHeight:      imgHeight,
			Columns:          cols,
			Rows:             rows,
			VirtualPlacement: true,
			Quite:            1,
			Chunk:            true,
			ChunkFormatter: func(chunk string) string {
				if tmux {
					return ansi.TmuxPassthrough(chunk)
				}
				return chunk
			},
		}); err != nil {
			slog.Error("Failed to encode image for kitty graphics", "err", err)
			return util.InfoMsg{
				Type: util.InfoTypeError,
				Msg:  "failed to encode image",
			}
		}

		return tea.RawMsg{Msg: buf.String()}
	}

	return cmd
}

// Render 使用指定的编码在指定尺寸内渲染给定的图像。
func (e Encoding) Render(id string, cols, rows int) string {
	key := imageKey{id: id, cols: cols, rows: rows}
	cachedMutex.RLock()
	cached, ok := cachedImages[key]
	cachedMutex.RUnlock()
	if !ok {
		return ""
	}

	img := cached.img

	switch e {
	case EncodingBlocks:
		canvas := paintbrush.New()
		canvas.SetImage(img)
		canvas.SetWidth(cols)
		canvas.SetHeight(rows)
		canvas.Weights = map[rune]float64{
			'': .95,
			'': .95,
			'▁': .9,
			'▂': .9,
			'▃': .9,
			'▄': .9,
			'▅': .9,
			'▆': .85,
			'█': .85,
			'▊': .95,
			'▋': .95,
			'▌': .95,
			'▍': .95,
			'▎': .95,
			'▏': .95,
			'●': .95,
			'◀': .95,
			'▲': .95,
			'▶': .95,
			'▼': .9,
			'○': .8,
			'◉': .95,
			'◧': .9,
			'◨': .9,
			'◩': .9,
			'◪': .9,
		}
		canvas.Paint()
		return strings.TrimSpace(canvas.GetResult())
	case EncodingKitty:
		// Build Kitty graphics unicode place holders
		var fg color.Color
		var extra int
		var r, g, b int
		hashedID := key.Hash()
		id := int(hashedID)
		extra, r, g, b = id>>24&0xff, id>>16&0xff, id>>8&0xff, id&0xff

		if id <= 255 {
			fg = ansi.IndexedColor(b)
		} else {
			fg = color.RGBA{
				R: uint8(r), //nolint:gosec
				G: uint8(g), //nolint:gosec
				B: uint8(b), //nolint:gosec
				A: 0xff,
			}
		}

		fgStyle := ansi.NewStyle().ForegroundColor(fg).String()

		var buf bytes.Buffer
		for y := range rows {
			// 作为优化，我们只在第一个单元格上写入前景色序列ID和
			// 列-行数据一次。终端将处理其余部分。
			buf.WriteString(fgStyle)
			buf.WriteRune(kitty.Placeholder)
			buf.WriteRune(kitty.Diacritic(y))
			buf.WriteRune(kitty.Diacritic(0))
			if extra > 0 {
				buf.WriteRune(kitty.Diacritic(extra))
			}
			for x := 1; x < cols; x++ {
				buf.WriteString(fgStyle)
				buf.WriteRune(kitty.Placeholder)
			}
			if y < rows-1 {
				buf.WriteByte('\n')
			}
		}

		return buf.String()

	default:
		return ""
	}
}
