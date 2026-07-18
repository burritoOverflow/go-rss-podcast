package main

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/hajimehoshi/go-mp3"
	"github.com/hajimehoshi/oto/v2"
	"github.com/mmcdole/gofeed"
)

const (
	audioChannels = 2
)

// audioController owns the process-wide oto context. oto supports only one
// context, so the model keeps one controller and creates players from it.
type audioController struct {
	context *oto.Context
	ready   chan struct{}
	rate    int
	err     error
}

// audioPlayback represents one streamed episode and its player. Pause keeps
// the oto player and decoder alive, allowing playback to resume at the same
// position instead of restarting the HTTP stream.
type audioPlayback struct {
	player   oto.Player
	response io.ReadCloser
	title    string
	elapsed  time.Duration
	duration time.Duration
	started  time.Time
	paused   bool
}

type audioStartedMsg struct {
	playback *audioPlayback
	index    int
	err      error
}

type audioTickMsg struct{}

func newAudioController() *audioController {
	return &audioController{}
}

// startAudio streams an MP3 enclosure and hands its decoded PCM to oto.
// go-mp3 decodes incrementally from the HTTP response; the complete episode
// is not downloaded before playback begins.
func (c *audioController) startAudio(item *gofeed.Item) (*audioPlayback, error) {
	if c == nil || c.err != nil {
		if c == nil {
			return nil, fmt.Errorf("audio is unavailable")
		}
		return nil, c.err
	}
	if len(item.Enclosures) == 0 {
		return nil, fmt.Errorf("no audio enclosure")
	}

	response, err := http.Get(item.Enclosures[0].URL)
	if err != nil {
		return nil, err
	}
	if response.StatusCode != http.StatusOK {
		response.Body.Close()
		return nil, fmt.Errorf("bad status %s", response.Status)
	}

	decoder, err := mp3.NewDecoder(response.Body)
	if err != nil {
		response.Body.Close()
		return nil, fmt.Errorf("decode audio: %w", err)
	}
	if c.context == nil {
		context, ready, contextErr := oto.NewContext(decoder.SampleRate(), audioChannels, oto.FormatSignedInt16LE)
		if contextErr != nil {
			response.Body.Close()
			return nil, fmt.Errorf("initialize audio: %w", contextErr)
		}
		c.context = context
		c.ready = ready
		c.rate = decoder.SampleRate()
	} else if decoder.SampleRate() != c.rate {
		response.Body.Close()
		return nil, fmt.Errorf("audio sample rate changed from %d Hz to %d Hz", c.rate, decoder.SampleRate())
	}
	<-c.ready

	player := c.context.NewPlayer(decoder)
	player.Play()

	duration := episodeDuration(item)
	if duration == 0 && decoder.Length() > 0 {
		duration = time.Duration(decoder.Length()/(audioChannels*2)) * time.Second / time.Duration(decoder.SampleRate())
	}

	return &audioPlayback{
		player:   player,
		response: response.Body,
		title:    item.Title,
		duration: duration,
		started:  time.Now(),
	}, nil
}

func episodeDuration(item *gofeed.Item) time.Duration {
	if item == nil || item.ITunesExt == nil {
		return 0
	}
	seconds, err := time.ParseDuration(item.ITunesExt.Duration + "s")
	if err != nil || seconds < 0 {
		return 0
	}
	return seconds
}

func (p *audioPlayback) close() {
	if p == nil {
		return
	}
	p.player.Close()
	p.response.Close()
}

func (p *audioPlayback) pause(now time.Time) {
	if p == nil || p.paused {
		return
	}
	p.elapsed += now.Sub(p.started)
	p.player.Pause()
	p.paused = true
}

func (p *audioPlayback) play(now time.Time) {
	if p == nil || !p.paused {
		return
	}
	p.started = now
	p.player.Play()
	p.paused = false
}

func (p *audioPlayback) position(now time.Time) time.Duration {
	if p == nil {
		return 0
	}
	position := p.elapsed
	if !p.paused {
		position += now.Sub(p.started)
	}
	if p.duration > 0 && position > p.duration {
		return p.duration
	}
	return position
}
