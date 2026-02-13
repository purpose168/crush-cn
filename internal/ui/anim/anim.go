// 包 anim 提供动画旋转器功能。
package anim

import (
	"fmt"
	"image/color"
	"math/rand/v2"
	"strings"
	"sync/atomic"
	"time"

	"github.com/zeebo/xxh3"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/lucasb-eyer/go-colorful"

	"github.com/purpose168/crush-cn/internal/csync"
)

const (
	fps           = 20
	initialChar   = '.'
	labelGap      = " "
	labelGapWidth = 1

	// 省略号动画速度的周期（以步数计）。
	//
	// 如果 FPS 为 20（50 毫秒），这意味着省略号将每 8 帧（400 毫秒）变化一次。
	ellipsisAnimSpeed = 8

	// 字符出现前允许的最大时间延迟。
	// 用于创建交错入场效果。
	maxBirthOffset = time.Second

	// 动画预渲染的帧数。达到此帧数后，动画将循环播放。
	// 仅在禁用颜色循环时适用。
	prerenderedFrames = 10

	// 默认的循环字符数量。
	defaultNumCyclingChars = 10
)

// 渐变的默认颜色。
var (
	defaultGradColorA = color.RGBA{R: 0xff, G: 0, B: 0, A: 0xff}
	defaultGradColorB = color.RGBA{R: 0, G: 0, B: 0xff, A: 0xff}
	defaultLabelColor = color.RGBA{R: 0xcc, G: 0xcc, B: 0xcc, A: 0xff}
)

var (
	availableRunes = []rune("0123456789abcdefABCDEF~!@#$£€%^&*()+=_")
	ellipsisFrames = []string{".", "..", "...", ""}
)

// 内部 ID 管理。用于动画期间确保帧消息仅被发送它们的旋转器组件接收。
var lastID int64

func nextID() int {
	return int(atomic.AddInt64(&lastID, 1))
}

// 动画计算的缓存结构
type animCache struct {
	initialFrames  [][]string
	cyclingFrames  [][]string
	width          int
	labelWidth     int
	label          []string
	ellipsisFrames []string
}

var animCacheMap = csync.NewMap[string, *animCache]()

// settingsHash 为设置创建哈希键用于缓存
func settingsHash(opts Settings) string {
	h := xxh3.New()
	fmt.Fprintf(h, "%d-%s-%v-%v-%v-%t",
		opts.Size, opts.Label, opts.LabelColor, opts.GradColorA, opts.GradColorB, opts.CycleColors)
	return fmt.Sprintf("%x", h.Sum(nil))
}

// StepMsg 是用于触发动画下一步的消息类型。
type StepMsg struct{ ID string }

// Settings 定义动画的设置。
type Settings struct {
	ID          string
	Size        int
	Label       string
	LabelColor  color.Color
	GradColorA  color.Color
	GradColorB  color.Color
	CycleColors bool
}

// 默认设置。
const ()

// Anim 是动画旋转器的 Bubble 组件。
type Anim struct {
	width            int
	cyclingCharWidth int
	label            *csync.Slice[string]
	labelWidth       int
	labelColor       color.Color
	startTime        time.Time
	birthOffsets     []time.Duration
	initialFrames    [][]string // 初始字符的帧
	initialized      atomic.Bool
	cyclingFrames    [][]string           // 循环字符的帧
	step             atomic.Int64         // 当前主帧步数
	ellipsisStep     atomic.Int64         // 当前省略号帧步数
	ellipsisFrames   *csync.Slice[string] // 省略号动画帧
	id               string
}

// New 创建一个新的 Anim 实例，使用指定的宽度和标签。
func New(opts Settings) *Anim {
	a := &Anim{}
	// 验证设置。
	if opts.Size < 1 {
		opts.Size = defaultNumCyclingChars
	}
	if colorIsUnset(opts.GradColorA) {
		opts.GradColorA = defaultGradColorA
	}
	if colorIsUnset(opts.GradColorB) {
		opts.GradColorB = defaultGradColorB
	}
	if colorIsUnset(opts.LabelColor) {
		opts.LabelColor = defaultLabelColor
	}

	if opts.ID != "" {
		a.id = opts.ID
	} else {
		a.id = fmt.Sprintf("%d", nextID())
	}
	a.startTime = time.Now()
	a.cyclingCharWidth = opts.Size
	a.labelColor = opts.LabelColor

	// 首先检查缓存
	cacheKey := settingsHash(opts)
	cached, exists := animCacheMap.Get(cacheKey)

	if exists {
		// 使用缓存的值
		a.width = cached.width
		a.labelWidth = cached.labelWidth
		a.label = csync.NewSliceFrom(cached.label)
		a.ellipsisFrames = csync.NewSliceFrom(cached.ellipsisFrames)
		a.initialFrames = cached.initialFrames
		a.cyclingFrames = cached.cyclingFrames
	} else {
		// 生成新值并缓存它们
		a.labelWidth = lipgloss.Width(opts.Label)

		// 动画的总宽度（以单元格计）。
		a.width = opts.Size
		if opts.Label != "" {
			a.width += labelGapWidth + lipgloss.Width(opts.Label)
		}

		// 渲染标签
		a.renderLabel(opts.Label)

		// 预生成渐变。
		var ramp []color.Color
		numFrames := prerenderedFrames
		if opts.CycleColors {
			ramp = makeGradientRamp(a.width*3, opts.GradColorA, opts.GradColorB, opts.GradColorA, opts.GradColorB)
			numFrames = a.width * 2
		} else {
			ramp = makeGradientRamp(a.width, opts.GradColorA, opts.GradColorB)
		}

		// 预渲染初始字符。
		a.initialFrames = make([][]string, numFrames)
		offset := 0
		for i := range a.initialFrames {
			a.initialFrames[i] = make([]string, a.width+labelGapWidth+a.labelWidth)
			for j := range a.initialFrames[i] {
				if j+offset >= len(ramp) {
					continue // 如果颜色用完则跳过
				}

				var c color.Color
				if j <= a.cyclingCharWidth {
					c = ramp[j+offset]
				} else {
					c = opts.LabelColor
				}

				// 同时使用 Lip Gloss 预渲染初始字符，以避免在渲染循环中处理。
				a.initialFrames[i][j] = lipgloss.NewStyle().
					Foreground(c).
					Render(string(initialChar))
			}
			if opts.CycleColors {
				offset++
			}
		}

		// 预渲染动画的打乱符文帧。
		a.cyclingFrames = make([][]string, numFrames)
		offset = 0
		for i := range a.cyclingFrames {
			a.cyclingFrames[i] = make([]string, a.width)
			for j := range a.cyclingFrames[i] {
				if j+offset >= len(ramp) {
					continue // 如果颜色用完则跳过
				}

				// 同时在此处使用 Lip Gloss 预渲染颜色，以避免在渲染循环中处理。
				r := availableRunes[rand.IntN(len(availableRunes))]
				a.cyclingFrames[i][j] = lipgloss.NewStyle().
					Foreground(ramp[j+offset]).
					Render(string(r))
			}
			if opts.CycleColors {
				offset++
			}
		}

		// 缓存结果
		labelSlice := make([]string, a.label.Len())
		for i, v := range a.label.Seq2() {
			labelSlice[i] = v
		}
		ellipsisSlice := make([]string, a.ellipsisFrames.Len())
		for i, v := range a.ellipsisFrames.Seq2() {
			ellipsisSlice[i] = v
		}
		cached = &animCache{
			initialFrames:  a.initialFrames,
			cyclingFrames:  a.cyclingFrames,
			width:          a.width,
			labelWidth:     a.labelWidth,
			label:          labelSlice,
			ellipsisFrames: ellipsisSlice,
		}
		animCacheMap.Set(cacheKey, cached)
	}

	// 随机为每个字符分配出生时间，以实现交错入场效果。
	a.birthOffsets = make([]time.Duration, a.width)
	for i := range a.birthOffsets {
		a.birthOffsets[i] = time.Duration(rand.N(int64(maxBirthOffset))) * time.Nanosecond
	}

	return a
}

// SetLabel 更新标签文本并重新渲染它。
func (a *Anim) SetLabel(newLabel string) {
	a.labelWidth = lipgloss.Width(newLabel)

	// 更新总宽度
	a.width = a.cyclingCharWidth
	if newLabel != "" {
		a.width += labelGapWidth + a.labelWidth
	}

	// 重新渲染标签
	a.renderLabel(newLabel)
}

// renderLabel 使用当前标签颜色渲染标签。
func (a *Anim) renderLabel(label string) {
	if a.labelWidth > 0 {
		// 预渲染标签。
		labelRunes := []rune(label)
		a.label = csync.NewSlice[string]()
		for i := range labelRunes {
			rendered := lipgloss.NewStyle().
				Foreground(a.labelColor).
				Render(string(labelRunes[i]))
			a.label.Append(rendered)
		}

		// 预渲染标签后的省略号帧。
		a.ellipsisFrames = csync.NewSlice[string]()
		for _, frame := range ellipsisFrames {
			rendered := lipgloss.NewStyle().
				Foreground(a.labelColor).
				Render(frame)
			a.ellipsisFrames.Append(rendered)
		}
	} else {
		a.label = csync.NewSlice[string]()
		a.ellipsisFrames = csync.NewSlice[string]()
	}
}

// Width 返回动画的总宽度。
func (a *Anim) Width() (w int) {
	w = a.width
	if a.labelWidth > 0 {
		w += labelGapWidth + a.labelWidth

		var widestEllipsisFrame int
		for _, f := range ellipsisFrames {
			fw := lipgloss.Width(f)
			if fw > widestEllipsisFrame {
				widestEllipsisFrame = fw
			}
		}
		w += widestEllipsisFrame
	}
	return w
}

// Start 启动动画。
func (a *Anim) Start() tea.Cmd {
	return a.Step()
}

// Animate 将动画推进到下一步。
func (a *Anim) Animate(msg StepMsg) tea.Cmd {
	if msg.ID != a.id {
		return nil
	}

	step := a.step.Add(1)
	if int(step) >= len(a.cyclingFrames) {
		a.step.Store(0)
	}

	if a.initialized.Load() && a.labelWidth > 0 {
		// 管理省略号动画。
		ellipsisStep := a.ellipsisStep.Add(1)
		if int(ellipsisStep) >= ellipsisAnimSpeed*len(ellipsisFrames) {
			a.ellipsisStep.Store(0)
		}
	} else if !a.initialized.Load() && time.Since(a.startTime) >= maxBirthOffset {
		a.initialized.Store(true)
	}
	return a.Step()
}

// Render 渲染动画的当前状态。
func (a *Anim) Render() string {
	var b strings.Builder
	step := int(a.step.Load())
	for i := range a.width {
		switch {
		case !a.initialized.Load() && i < len(a.birthOffsets) && time.Since(a.startTime) < a.birthOffsets[i]:
			// 出生偏移未达到：渲染初始字符。
			b.WriteString(a.initialFrames[step][i])
		case i < a.cyclingCharWidth:
			// 渲染循环字符。
			b.WriteString(a.cyclingFrames[step][i])
		case i == a.cyclingCharWidth:
			// 渲染标签间隙。
			b.WriteString(labelGap)
		case i > a.cyclingCharWidth:
			// 标签。
			if labelChar, ok := a.label.Get(i - a.cyclingCharWidth - labelGapWidth); ok {
				b.WriteString(labelChar)
			}
		}
	}
	// 如果所有字符都已初始化，在标签末尾渲染动画省略号。
	if a.initialized.Load() && a.labelWidth > 0 {
		ellipsisStep := int(a.ellipsisStep.Load())
		if ellipsisFrame, ok := a.ellipsisFrames.Get(ellipsisStep / ellipsisAnimSpeed); ok {
			b.WriteString(ellipsisFrame)
		}
	}

	return b.String()
}

// Step 是一个命令，用于触发动画的下一步。
func (a *Anim) Step() tea.Cmd {
	return tea.Tick(time.Second/time.Duration(fps), func(t time.Time) tea.Msg {
		return StepMsg{ID: a.id}
	})
}

// makeGradientRamp() 返回在给定关键点之间混合的颜色切片。
// 混合使用 Hcl 方式以保持在色域内。
func makeGradientRamp(size int, stops ...color.Color) []color.Color {
	if len(stops) < 2 {
		return nil
	}

	points := make([]colorful.Color, len(stops))
	for i, k := range stops {
		points[i], _ = colorful.MakeColor(k)
	}

	numSegments := len(stops) - 1
	if numSegments == 0 {
		return nil
	}
	blended := make([]color.Color, 0, size)

	// 计算每个段应该有多少种颜色。
	segmentSizes := make([]int, numSegments)
	baseSize := size / numSegments
	remainder := size % numSegments

	// 将余数分配到各个段中。
	for i := range numSegments {
		segmentSizes[i] = baseSize
		if i < remainder {
			segmentSizes[i]++
		}
	}

	// 为每个段生成颜色。
	for i := range numSegments {
		c1 := points[i]
		c2 := points[i+1]
		segmentSize := segmentSizes[i]

		for j := range segmentSize {
			if segmentSize == 0 {
				continue
			}
			t := float64(j) / float64(segmentSize)
			c := c1.BlendHcl(c2, t)
			blended = append(blended, c)
		}
	}

	return blended
}

func colorIsUnset(c color.Color) bool {
	if c == nil {
		return true
	}
	_, _, _, a := c.RGBA()
	return a == 0
}
