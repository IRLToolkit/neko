package webrtc

import (
	"github.com/pion/webrtc/v3"
)

type WebRTCPeerCtx struct {
	api           *webrtc.API
	engine        *webrtc.MediaEngine
	settings      *webrtc.SettingEngine
	connection    *webrtc.PeerConnection
	configuration *webrtc.Configuration
}

func (webrtc_peer *WebRTCPeerCtx) SignalAnswer(sdp string) error {
	return webrtc_peer.connection.SetRemoteDescription(webrtc.SessionDescription{
		SDP: sdp,
		Type: webrtc.SDPTypeAnswer,
	})
}

func (webrtc_peer *WebRTCPeerCtx) Destroy() error {
	if webrtc_peer.connection == nil || webrtc_peer.connection.ConnectionState() != webrtc.PeerConnectionStateConnected {
		return nil
	}
	
	return webrtc_peer.connection.Close()
}
