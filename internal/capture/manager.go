package capture

import (
	"fmt"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"demodesk/neko/internal/config"
	"demodesk/neko/internal/types"
	"demodesk/neko/internal/types/codec"
)

type CaptureManagerCtx struct {
	logger     zerolog.Logger
	desktop    types.DesktopManager
	streaming  bool
	broadcast  *BroacastManagerCtx
	screencast *ScreencastManagerCtx
	audio      *StreamManagerCtx
	videos     map[string]*StreamManagerCtx
	videoIDs   []string
}

func New(desktop types.DesktopManager, config *config.Capture) *CaptureManagerCtx {
	logger := log.With().Str("module", "capture").Logger()

	broadcastPipeline := config.BroadcastPipeline
	if broadcastPipeline == "" {
		broadcastPipeline = fmt.Sprintf(
			"flvmux name=mux ! rtmpsink location='{url} live=1' "+
				"pulsesrc device=%s "+
				"! audio/x-raw,channels=2 "+
				"! audioconvert "+
				"! queue "+
				"! voaacenc bitrate=%d "+
				"! mux. "+
				"ximagesrc display-name=%s show-pointer=true use-damage=false "+
				"! video/x-raw "+
				"! videoconvert "+
				"! queue "+
				"! x264enc threads=4 bitrate=%d key-int-max=15 byte-stream=true tune=zerolatency speed-preset=%s "+
				"! mux.", config.AudioDevice, config.BroadcastAudioBitrate*1000, config.Display, config.BroadcastVideoBitrate, config.BroadcastPreset,
		)
	}

	screencastPipeline := config.ScreencastPipeline
	if screencastPipeline == "" {
		screencastPipeline = fmt.Sprintf(
			"ximagesrc display-name=%s show-pointer=true use-damage=false "+
				"! video/x-raw,framerate=%s "+
				"! videoconvert "+
				"! queue "+
				"! jpegenc quality=%s "+
				"! appsink name=appsink", config.Display, config.ScreencastRate, config.ScreencastQuality,
		)
	}

	return &CaptureManagerCtx{
		logger:     logger,
		desktop:    desktop,
		streaming:  false,
		broadcast:  broadcastNew(broadcastPipeline),
		screencast: screencastNew(config.ScreencastEnabled, screencastPipeline),
		audio: streamNew(config.AudioCodec, func() string {
			if config.AudioPipeline != "" {
				return config.AudioPipeline
			}

			return fmt.Sprintf(
				"pulsesrc device=%s "+
					"! audio/x-raw,channels=2 "+
					"! audioconvert "+
					"! queue "+
					"! %s "+
					"! appsink name=appsink", config.AudioDevice, config.AudioCodec.Pipeline,
			)
		}),
		videos: map[string]*StreamManagerCtx{
			"hd": streamNew(codec.VP8(), func() string {
				screen := desktop.GetScreenSize()
				bitrate := int((screen.Width * screen.Height * 6) / 4)
				buffer := bitrate / 1000

				return fmt.Sprintf(
					"ximagesrc display-name=%s show-pointer=false use-damage=false "+
						"! video/x-raw,framerate=25/1 "+
						"! videoconvert "+
						"! queue "+
						"! vp8enc end-usage=cbr target-bitrate=%d cpu-used=4 threads=4 deadline=1 undershoot=95 keyframe-max-dist=25 min-quantizer=3 max-quantizer=32 buffer-size=%d buffer-initial-size=%d buffer-optimal-size=%d "+
						"! appsink name=appsink", config.Display, bitrate, buffer*6, buffer*4, buffer*5,
				)
			}),
			"hq": streamNew(codec.VP8(), func() string {
				screen := desktop.GetScreenSize()
				bitrate := int((screen.Width * screen.Height * 6) / 4) / 2
				buffer := bitrate / 1000

				return fmt.Sprintf(
					"ximagesrc display-name=%s show-pointer=false use-damage=false "+
						"! video/x-raw,framerate=15/1 "+
						"! videoconvert "+
						"! queue "+
						"! vp8enc end-usage=cbr target-bitrate=%d cpu-used=4 threads=4 deadline=1 undershoot=95 keyframe-max-dist=25 min-quantizer=3 max-quantizer=32 buffer-size=%d buffer-initial-size=%d buffer-optimal-size=%d "+
						"! appsink name=appsink", config.Display, bitrate, buffer*6, buffer*4, buffer*5,
				)
			}),
			"mq": streamNew(codec.VP8(), func() string {
				screen := desktop.GetScreenSize()
				bitrate := int((screen.Width * screen.Height * 6) / 4) / 3
				buffer := bitrate / 1000

				return fmt.Sprintf(
					"ximagesrc display-name=%s show-pointer=false use-damage=false "+
						"! video/x-raw,framerate=10/1 "+
						"! videoconvert "+
						"! queue "+
						"! vp8enc end-usage=cbr target-bitrate=%d cpu-used=4 threads=4 deadline=1 undershoot=95 keyframe-max-dist=25 min-quantizer=3 max-quantizer=32 buffer-size=%d buffer-initial-size=%d buffer-optimal-size=%d "+
						"! appsink name=appsink", config.Display, bitrate, buffer*6, buffer*4, buffer*5,
				)
			}),
			"lq": streamNew(codec.VP8(), func() string {
				screen := desktop.GetScreenSize()
				bitrate := int((screen.Width * screen.Height * 6) / 4) / 4
				buffer := bitrate / 1000
	
				return fmt.Sprintf(
					"ximagesrc display-name=%s show-pointer=false use-damage=false "+
						"! video/x-raw,framerate=5/1 "+
						"! videoconvert "+
						"! queue "+
						"! vp8enc end-usage=cbr target-bitrate=%d cpu-used=4 threads=4 deadline=1 undershoot=95 keyframe-max-dist=25 min-quantizer=3 max-quantizer=32 buffer-size=%d buffer-initial-size=%d buffer-optimal-size=%d "+
						"! appsink name=appsink", config.Display, bitrate, buffer*6, buffer*4, buffer*5,
				)
			}),
		},
		videoIDs: []string{"hd", "hq", "mq", "lq"},
	}
}

func (manager *CaptureManagerCtx) Start() {
	if manager.broadcast.Started() {
		if err := manager.broadcast.createPipeline(); err != nil {
			manager.logger.Panic().Err(err).Msg("unable to create broadcast pipeline")
		}
	}

	manager.desktop.OnBeforeScreenSizeChange(func() {
		for _, video := range manager.videos {
			if video.Started() {
				video.destroyPipeline()
			}
		}

		if manager.broadcast.Started() {
			manager.broadcast.destroyPipeline()
		}

		if manager.screencast.Started() {
			manager.screencast.destroyPipeline()
		}
	})

	manager.desktop.OnAfterScreenSizeChange(func() {
		for _, video := range manager.videos {
			if video.Started() {
				if err := video.createPipeline(); err != nil {
					manager.logger.Panic().Err(err).Msg("unable to recreate video pipeline")
				}
			}
		}

		if manager.broadcast.Started() {
			if err := manager.broadcast.createPipeline(); err != nil {
				manager.logger.Panic().Err(err).Msg("unable to recreate broadcast pipeline")
			}
		}

		if manager.screencast.Started() {
			if err := manager.screencast.createPipeline(); err != nil {
				manager.logger.Panic().Err(err).Msg("unable to recreate screencast pipeline")
			}
		}
	})
}

func (manager *CaptureManagerCtx) Shutdown() error {
	manager.logger.Info().Msgf("capture shutting down")

	manager.broadcast.shutdown()
	manager.screencast.shutdown()

	manager.audio.shutdown()

	for _, video := range manager.videos {
		video.shutdown()
	}

	return nil
}

func (manager *CaptureManagerCtx) Broadcast() types.BroadcastManager {
	return manager.broadcast
}

func (manager *CaptureManagerCtx) Screencast() types.ScreencastManager {
	return manager.screencast
}

func (manager *CaptureManagerCtx) Audio() types.StreamManager {
	return manager.audio
}

func (manager *CaptureManagerCtx) Video(videoID string) (types.StreamManager, bool) {
	video, ok := manager.videos[videoID]
	return video, ok
}

func (manager *CaptureManagerCtx) VideoIDs() []string {
	return manager.videoIDs
}
