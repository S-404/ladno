package rest

import (
	"fmt"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/s-404/ladno/internal/app/components/ui"
	"github.com/s-404/ladno/internal/app/entity"
)

type ResponseView struct {
	root         fyne.CanvasObject
	statusLabel  *widget.Label
	metaLabel    *widget.Label
	bodyEntry    *widget.Entry
	headersEntry *widget.Entry
}

func NewResponseView() *ResponseView {
	v := &ResponseView{
		statusLabel:  widget.NewLabel("Ready"),
		metaLabel:    widget.NewLabel(""),
		bodyEntry:    widget.NewMultiLineEntry(),
		headersEntry: widget.NewMultiLineEntry(),
	}
	v.statusLabel.TextStyle = fyne.TextStyle{Bold: true}
	v.bodyEntry.TextStyle = fyne.TextStyle{Monospace: true}
	v.bodyEntry.Wrapping = fyne.TextWrapOff
	v.bodyEntry.SetPlaceHolder("Response body")
	v.headersEntry.TextStyle = fyne.TextStyle{Monospace: true}
	v.headersEntry.Wrapping = fyne.TextWrapOff
	v.headersEntry.SetPlaceHolder("Response headers")

	tabs := container.NewAppTabs(
		container.NewTabItem("Body", container.NewScroll(v.bodyEntry)),
		container.NewTabItem("Headers", container.NewScroll(v.headersEntry)),
	)

	header := container.NewHBox(v.statusLabel, v.metaLabel)
	panel := container.NewBorder(header, nil, nil, nil, tabs)
	v.root = ui.NewPanelBackground(panel)
	return v
}

func (v *ResponseView) Object() fyne.CanvasObject {
	return v.root
}

func (v *ResponseView) SetLoading() {
	v.statusLabel.SetText("Sending…")
	v.metaLabel.SetText("")
	v.bodyEntry.SetText("")
	v.headersEntry.SetText("")
}

func (v *ResponseView) SetIdle() {
	v.statusLabel.SetText("Ready")
	v.metaLabel.SetText("")
}

func (v *ResponseView) SetResponse(resp *entity.RestResponse) {
	if resp == nil {
		v.SetIdle()
		return
	}
	if resp.Error != "" && resp.StatusCode == 0 {
		v.statusLabel.SetText("Error")
		v.metaLabel.SetText(formatDuration(resp.Duration))
		v.bodyEntry.SetText(resp.Error)
		v.headersEntry.SetText("")
		return
	}

	v.statusLabel.SetText(fmt.Sprintf("%d %s", resp.StatusCode, statusText(resp.Status)))
	meta := formatDuration(resp.Duration)
	if resp.Error != "" {
		meta += " · " + resp.Error
	}
	v.metaLabel.SetText(meta)
	v.bodyEntry.SetText(resp.Body)
	v.headersEntry.SetText(formatHeaders(resp.Headers))
}

func statusText(status string) string {
	if status == "" {
		return ""
	}
	parts := strings.SplitN(status, " ", 2)
	if len(parts) == 2 {
		return parts[1]
	}
	return status
}

func formatDuration(d time.Duration) string {
	if d < time.Millisecond {
		return d.String()
	}
	return fmt.Sprintf("%d ms", d.Milliseconds())
}

func formatHeaders(headers map[string][]string) string {
	if len(headers) == 0 {
		return ""
	}
	var b strings.Builder
	for k, vals := range headers {
		for _, val := range vals {
			b.WriteString(k)
			b.WriteString(": ")
			b.WriteString(val)
			b.WriteByte('\n')
		}
	}
	return b.String()
}
