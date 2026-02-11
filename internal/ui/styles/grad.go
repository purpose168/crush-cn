package styles

import (
	"fmt"
	"image/color"
	"strings"

	"github.com/lucasb-eyer/go-colorful"
	"github.com/rivo/uniseg"
)

// ForegroundGrad 返回一个字符串切片，表示用从 color1 到 color2 的水平渐变前景色渲染的输入字符串
// 返回切片中的每个字符串对应输入字符串中的一个字形簇（grapheme cluster）
// 如果 bold 为 true，渲染的字符串将以粗体显示
func ForegroundGrad(t *Styles, input string, bold bool, color1, color2 color.Color) []string {
	if input == "" {
		return []string{""}
	}
	if len(input) == 1 {
		style := t.Base.Foreground(color1)
		if bold {
			style.Bold(true)
		}
		return []string{style.Render(input)}
	}
	var clusters []string
	gr := uniseg.NewGraphemes(input)
	for gr.Next() {
		clusters = append(clusters, string(gr.Runes()))
	}

	ramp := blendColors(len(clusters), color1, color2)
	for i, c := range ramp {
		style := t.Base.Foreground(c)
		if bold {
			style.Bold(true)
		}
		clusters[i] = style.Render(clusters[i])
	}
	return clusters
}

// ApplyForegroundGrad 使用水平渐变前景色渲染给定的字符串
func ApplyForegroundGrad(t *Styles, input string, color1, color2 color.Color) string {
	if input == "" {
		return ""
	}
	var o strings.Builder
	clusters := ForegroundGrad(t, input, false, color1, color2)
	for _, c := range clusters {
		fmt.Fprint(&o, c)
	}
	return o.String()
}

// ApplyBoldForegroundGrad 使用水平渐变前景色和粗体渲染给定的字符串
func ApplyBoldForegroundGrad(t *Styles, input string, color1, color2 color.Color) string {
	if input == "" {
		return ""
	}
	var o strings.Builder
	clusters := ForegroundGrad(t, input, true, color1, color2)
	for _, c := range clusters {
		fmt.Fprint(&o, c)
	}
	return o.String()
}

// blendColors 返回在给定颜色键之间混合的颜色切片
// 混合在 Hcl 颜色空间中进行，以保持在色域内
func blendColors(size int, stops ...color.Color) []color.Color {
	if len(stops) < 2 {
		return nil
	}

	stopsPrime := make([]colorful.Color, len(stops))
	for i, k := range stops {
		stopsPrime[i], _ = colorful.MakeColor(k)
	}

	numSegments := len(stopsPrime) - 1
	blended := make([]color.Color, 0, size)

	// 计算每个段应该有多少种颜色
	segmentSizes := make([]int, numSegments)
	baseSize := size / numSegments
	remainder := size % numSegments

	// 将余数分配到各个段
	for i := range numSegments {
		segmentSizes[i] = baseSize
		if i < remainder {
			segmentSizes[i]++
		}
	}

	// 为每个段生成颜色
	for i := range numSegments {
		c1 := stopsPrime[i]
		c2 := stopsPrime[i+1]
		segmentSize := segmentSizes[i]

		for j := range segmentSize {
			var t float64
			if segmentSize > 1 {
				t = float64(j) / float64(segmentSize-1)
			}
			c := c1.BlendHcl(c2, t)
			blended = append(blended, c)
		}
	}

	return blended
}
