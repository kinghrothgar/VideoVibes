package media

import (
	"errors"
	"fmt"
	"image"

	"github.com/zergon321/reisen"
)

type Media struct {
	m *reisen.Media
	v *reisen.VideoStream
	a *reisen.AudioStream
}

func NewMedia(mediapath string) (*Media, error) {
	m, err := reisen.NewMedia(mediapath)
	if err != nil {
		return nil, fmt.Errorf("failed to create NewMedia: %w", err)
	}
	vstreams := m.VideoStreams()
	if len(vstreams) > 1 {
		return nil, errors.New("more than 1 video stream")
	}
	v := vstreams[0]
	return &Media{
		m: m,
		v: v,
	}, nil
}

func (m *Media) Open() error {
	if err := m.m.OpenDecode(); err != nil {
		return fmt.Errorf("failed to open media: %w", err)
	}
	if err := m.v.Open(); err != nil {
		return fmt.Errorf("failed to open video: %w", err)
	}
	return nil
}

func (m *Media) Close() {
	if m.m != nil {
		m.m.Close()
	}
	if m.v != nil {
		m.v.Close()
	}
	if m.a != nil {
		m.a.Close()
	}
}

func (m *Media) Stream(frames chan *image.RGBA, e chan error) chan bool {
	done := make(chan bool)
	go func() {
		for {
			packet, gotPacket, err := m.m.ReadPacket()

			if err != nil {
				e <- fmt.Errorf("failed to read packet: %w", err)
			}

			if !gotPacket {
				e <- errors.New("got no packet")
				break
			}

			switch packet.Type() {
			case reisen.StreamVideo:
				s := m.m.Streams()[packet.StreamIndex()].(*reisen.VideoStream)
				videoFrame, gotFrame, err := s.ReadVideoFrame()

				if err != nil {
					e <- fmt.Errorf("failed to read video frame: %w", err)
				}

				if !gotFrame {
					e <- errors.New("failed to get frame")
					break
				}

				if videoFrame == nil {
					continue
				}

				frames <- videoFrame.Image()
			case reisen.StreamAudio:
			}
		}
		done <- true
	}()
	return done
}
