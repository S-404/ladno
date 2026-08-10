package logs

import "image/color"

var (
	colorBright = color.NRGBA{R: 0xEE, G: 0xF0, B: 0xF4, A: 0xFF}
	colorMuted  = color.NRGBA{R: 0xB0, G: 0xB8, B: 0xC4, A: 0xFF}

	colorStatus1xx = color.NRGBA{R: 0x9A, G: 0xA0, B: 0xA6, A: 0xFF} // серый
	colorStatus2xx = color.NRGBA{R: 0x34, G: 0xA8, B: 0x53, A: 0xFF} // зелёный
	colorStatus3xx = color.NRGBA{R: 0xFB, G: 0xBC, B: 0x04, A: 0xFF} // жёлтый
	colorStatus4xx = color.NRGBA{R: 0xEA, G: 0x43, B: 0x35, A: 0xFF} // красный
	colorStatus5xx = color.NRGBA{R: 0xC2, G: 0x18, B: 0x5B, A: 0xFF} // бордовый
	colorStatusErr = color.NRGBA{R: 0xEA, G: 0x43, B: 0x35, A: 0xFF}
)

func StatusColor(statusCode int, isError bool) color.Color {
	if isError && statusCode == 0 {
		return colorStatusErr
	}
	switch statusCode / 100 {
	case 1:
		return colorStatus1xx
	case 2:
		return colorStatus2xx
	case 3:
		return colorStatus3xx
	case 4:
		return colorStatus4xx
	case 5:
		return colorStatus5xx
	default:
		if isError {
			return colorStatusErr
		}
		return colorBright
	}
}
