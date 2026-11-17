package ui

import (
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/yinrg/trustanchor/internal/truststore"
)

func TestValidityText(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name       string
		notAfter   time.Time
		wantText   string
		wantImport widget.Importance
	}{
		{
			name:       "未设置有效期",
			wantText:   "",
			wantImport: widget.LowImportance,
		},
		{
			name:       "正常有效期",
			notAfter:   now.AddDate(1, 0, 0),
			wantText:   "证书有效期至 2027-01-01",
			wantImport: widget.LowImportance,
		},
		{
			name:       "剩余 31 天",
			notAfter:   now.Add(31 * 24 * time.Hour),
			wantText:   "证书有效期至 2026-02-01",
			wantImport: widget.LowImportance,
		},
		{
			name:       "剩余 30 天进入预警",
			notAfter:   now.Add(30 * 24 * time.Hour),
			wantText:   "证书有效期至 2026-01-31（剩余 30 天，即将过期）",
			wantImport: widget.WarningImportance,
		},
		{
			name:       "已过期",
			notAfter:   now.AddDate(0, 0, -31),
			wantText:   "证书已于 2025-12-01 过期",
			wantImport: widget.DangerImportance,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st := truststore.State{Info: truststore.Info{NotAfter: tt.notAfter}}
			text, importance := validityText(st, now)
			if text != tt.wantText {
				t.Fatalf("文案 = %q，期望 %q", text, tt.wantText)
			}
			if importance != tt.wantImport {
				t.Fatalf("importance = %v，期望 %v", importance, tt.wantImport)
			}
		})
	}
}

func TestColorForImportance(t *testing.T) {
	tests := []struct {
		importance widget.Importance
		want       fyne.ThemeColorName
	}{
		{widget.SuccessImportance, theme.ColorNameSuccess},
		{widget.WarningImportance, theme.ColorNameWarning},
		{widget.DangerImportance, theme.ColorNameError},
		{widget.MediumImportance, ""},
	}
	for _, tt := range tests {
		if got := colorForImportance(tt.importance); got != tt.want {
			t.Fatalf("colorForImportance(%v) = %q，期望 %q", tt.importance, got, tt.want)
		}
	}
}
