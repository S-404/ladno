package grpcui

import (
	"fmt"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/s-404/ladno/internal/app/components/ui"
	"github.com/s-404/ladno/internal/app/entity"
	"github.com/s-404/ladno/internal/app/utils"
)

type ResponseView struct {
	root        fyne.CanvasObject
	statusLabel *widget.Label
	metaLabel   *widget.Label
	bodyEntry   *widget.Entry
	metaEntry   *widget.Entry
}

func NewResponseView() *ResponseView {
	v := &ResponseView{
		statusLabel: widget.NewLabel("Ready"),
		metaLabel:   widget.NewLabel(""),
		bodyEntry:   widget.NewMultiLineEntry(),
		metaEntry:   widget.NewMultiLineEntry(),
	}
	v.statusLabel.TextStyle = fyne.TextStyle{Bold: true}
	v.bodyEntry.TextStyle = fyne.TextStyle{Monospace: true}
	v.bodyEntry.Wrapping = fyne.TextWrapOff
	v.bodyEntry.SetPlaceHolder("Response message")
	v.metaEntry.TextStyle = fyne.TextStyle{Monospace: true}
	v.metaEntry.Wrapping = fyne.TextWrapOff
	v.metaEntry.SetPlaceHolder("Response metadata")

	tabs := container.NewAppTabs(
		container.NewTabItem("Body", container.NewScroll(v.bodyEntry)),
		container.NewTabItem("Metadata", container.NewScroll(v.metaEntry)),
	)
	header := container.NewHBox(v.statusLabel, v.metaLabel)
	v.root = ui.NewPanelBackground(container.NewBorder(header, nil, nil, nil, tabs))
	return v
}

func (v *ResponseView) Object() fyne.CanvasObject {
	return v.root
}

func (v *ResponseView) SetLoading() {
	v.statusLabel.SetText("Sending…")
	v.metaLabel.SetText("")
	v.bodyEntry.SetText("")
	v.metaEntry.SetText("")
}

func (v *ResponseView) SetIdle() {
	v.statusLabel.SetText("Ready")
	v.metaLabel.SetText("")
}

func (v *ResponseView) SetResponse(resp *entity.GrpcResponse) {
	if resp == nil {
		v.SetIdle()
		return
	}
	if resp.Error != "" && resp.Status == "" {
		v.statusLabel.SetText("Error")
		v.metaLabel.SetText(grpcDuration(resp.Duration))
		v.bodyEntry.SetText(resp.Error)
		v.metaEntry.SetText("")
		return
	}
	st := resp.Status
	if st == "" {
		st = "OK"
	}
	v.statusLabel.SetText(st)
	meta := grpcDuration(resp.Duration)
	if resp.Error != "" {
		meta += " · " + resp.Error
	}
	v.metaLabel.SetText(meta)
	if resp.Body != "" {
		v.bodyEntry.SetText(utils.PrettyBody(resp.Body, "application/json"))
	} else {
		v.bodyEntry.SetText(resp.Error)
	}
	v.metaEntry.SetText(formatMetadata(resp.Metadata))
}

func grpcDuration(d time.Duration) string {
	if d < time.Millisecond {
		return d.String()
	}
	return fmt.Sprintf("%d ms", d.Milliseconds())
}

func formatMetadata(headers map[string][]string) string {
	if len(headers) == 0 {
		return ""
	}
	var b strings.Builder
	for k, vals := range headers {
		for _, val := range vals {
			b.WriteString(k)
			b.WriteString(": ")
			if strings.EqualFold(strings.TrimSpace(k), "authorization") {
				if val != "" {
					b.WriteString("••••••••")
				}
			} else {
				b.WriteString(val)
			}
			b.WriteByte('\n')
		}
	}
	return b.String()
}
