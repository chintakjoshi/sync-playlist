package handlers

import (
	"time"

	"server/internal/database"
)

type ConnectedServiceResponse struct {
	ID              uint      `json:"id"`
	UserID          uint      `json:"user_id"`
	ServiceType     string    `json:"service_type"`
	ServiceUserID   string    `json:"service_user_id"`
	ServiceUserName string    `json:"service_user_name"`
	TokenExpiry     int64     `json:"token_expiry"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type StoredPlaylistResponse struct {
	ID           uint   `json:"id"`
	ServiceType  string `json:"service_type"`
	ServiceID    string `json:"service_id"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	TrackCount   int    `json:"track_count"`
	ImageURL     string `json:"image_url"`
	IsPublic     bool   `json:"is_public"`
	LastSyncedAt int64  `json:"last_synced_at"`
}

type TransferResponse struct {
	ID                 uint      `json:"id"`
	SourceService      string    `json:"source_service"`
	SourcePlaylistID   string    `json:"source_playlist_id"`
	SourcePlaylistName string    `json:"source_playlist_name"`
	TargetService      string    `json:"target_service"`
	TargetPlaylistID   string    `json:"target_playlist_id"`
	TargetPlaylistName string    `json:"target_playlist_name"`
	Status             string    `json:"status"`
	TracksTotal        int       `json:"tracks_total"`
	TracksMatched      int       `json:"tracks_matched"`
	TracksFailed       int       `json:"tracks_failed"`
	ErrorMessage       string    `json:"error_message"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type TransferTrackResponse struct {
	ID              uint    `json:"id"`
	TransferID      uint    `json:"transfer_id"`
	SourceTrackID   string  `json:"source_track_id"`
	SourceTrackName string  `json:"source_track_name"`
	SourceArtist    string  `json:"source_artist"`
	TargetTrackID   string  `json:"target_track_id"`
	TargetTrackName string  `json:"target_track_name"`
	TargetArtist    string  `json:"target_artist"`
	Status          string  `json:"status"`
	MatchConfidence float64 `json:"match_confidence"`
}

func newConnectedServiceResponse(service database.UserService) ConnectedServiceResponse {
	return ConnectedServiceResponse{
		ID:              service.ID,
		UserID:          service.UserID,
		ServiceType:     service.ServiceType,
		ServiceUserID:   service.ServiceUserID,
		ServiceUserName: service.ServiceUserName,
		TokenExpiry:     service.TokenExpiry,
		CreatedAt:       service.CreatedAt,
		UpdatedAt:       service.UpdatedAt,
	}
}

func newStoredPlaylistResponse(playlist database.Playlist) StoredPlaylistResponse {
	return StoredPlaylistResponse{
		ID:           playlist.ID,
		ServiceType:  playlist.ServiceType,
		ServiceID:    playlist.ServiceID,
		Name:         playlist.Name,
		Description:  playlist.Description,
		TrackCount:   playlist.TrackCount,
		ImageURL:     playlist.ImageURL,
		IsPublic:     playlist.IsPublic,
		LastSyncedAt: playlist.LastSyncedAt,
	}
}

func newTransferResponse(transfer database.Transfer) TransferResponse {
	return TransferResponse{
		ID:                 transfer.ID,
		SourceService:      transfer.SourceService,
		SourcePlaylistID:   transfer.SourcePlaylistID,
		SourcePlaylistName: transfer.SourcePlaylistName,
		TargetService:      transfer.TargetService,
		TargetPlaylistID:   transfer.TargetPlaylistID,
		TargetPlaylistName: transfer.TargetPlaylistName,
		Status:             transfer.Status,
		TracksTotal:        transfer.TracksTotal,
		TracksMatched:      transfer.TracksMatched,
		TracksFailed:       transfer.TracksFailed,
		ErrorMessage:       transfer.ErrorMessage,
		CreatedAt:          transfer.CreatedAt,
		UpdatedAt:          transfer.UpdatedAt,
	}
}

func newTransferTrackResponse(track database.TransferTrack) TransferTrackResponse {
	return TransferTrackResponse{
		ID:              track.ID,
		TransferID:      track.TransferID,
		SourceTrackID:   track.SourceTrackID,
		SourceTrackName: track.SourceTrackName,
		SourceArtist:    track.SourceArtist,
		TargetTrackID:   track.TargetTrackID,
		TargetTrackName: track.TargetTrackName,
		TargetArtist:    track.TargetArtist,
		Status:          track.Status,
		MatchConfidence: track.MatchConfidence,
	}
}
