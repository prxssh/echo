package peer

import (
	"context"
	"net"
	"strings"

	"github.com/prxssh/echo/internal/utils"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type peerMetadata struct {
	Addr        string `json:"addr"`
	CountryCode string `json:"isoCode"`
	CountryName string `json:"country"`
	Flag        string `json:"flag"`
}

type peerMessageEvent struct {
	peerMetadata
	Type string `json:"type"`
}

func (p *Peer) metadata() peerMetadata {
	host, _, err := net.SplitHostPort(p.Addr())
	if err != nil {
		host = p.Addr()
	}
	code, name, _ := utils.IP2Country.CountryCode(host)

	return peerMetadata{
		Addr:        p.Addr(),
		CountryCode: code,
		CountryName: name,
		Flag:        countryFlag(code),
	}
}

func (p *Peer) emitStarted(ctx context.Context) {
	runtime.EventsEmit(ctx, "peers:started", p.metadata())
}

func (p *Peer) emitStopped(ctx context.Context) {
	runtime.EventsEmit(ctx, "peers:stopped", p.metadata())
}

func (p *Peer) emitMessage(ctx context.Context, typ string) {
	payload := peerMessageEvent{
		peerMetadata: p.metadata(),
		Type:         typ,
	}

	runtime.EventsEmit(ctx, "peer:msg", payload)
}

func countryFlag(code string) string {
	if len(code) != 2 {
		return ""
	}

	code = strings.ToUpper(code)
	r1 := rune(code[0]) - 'A' + 0x1F1E6
	r2 := rune(code[1]) - 'A' + 0x1F1E6

	return string([]rune{r1, r2})
}
