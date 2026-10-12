package services

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"math"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const recordingPlaybackBuffer = 10 * time.Minute

type StreamTokenClaims struct {
	TruckID  string
	CameraID string
	Quality  string
	Expires  time.Time
}

type RecordingTokenClaims struct {
	TruckID  string
	CameraID string
	Start    time.Time
	Duration float64
	Expires  time.Time
}

type StreamService struct {
	publicBaseURL   string
	signingKey      string
	publishPassword string
}

func NewStreamService(publicBaseURL, signingKey, publishPassword string) *StreamService {
	return &StreamService{
		publicBaseURL:   strings.TrimRight(publicBaseURL, "/"),
		signingKey:      signingKey,
		publishPassword: publishPassword,
	}
}

func (s *StreamService) SignAccess(truckID, cameraID, quality string, expires time.Time) string {
	payload := strings.Join([]string{truckID, cameraID, quality, strconv.FormatInt(expires.Unix(), 10)}, "|")
	return s.signPayload(payload)
}

func (s *StreamService) SignRecordingAccess(truckID, cameraID string, start time.Time, duration float64, expires time.Time) string {
	payload := strings.Join([]string{
		"recording",
		truckID,
		cameraID,
		strconv.FormatInt(start.UnixNano(), 10),
		strconv.FormatInt(int64(math.Round(duration*float64(time.Second))), 10),
		strconv.FormatInt(expires.Unix(), 10),
	}, "|")
	return s.signPayload(payload)
}

func (s *StreamService) signPayload(payload string) string {
	encodedPayload := base64.RawURLEncoding.EncodeToString([]byte(payload))
	mac := hmac.New(sha256.New, []byte(s.signingKey))
	_, _ = mac.Write([]byte(encodedPayload))
	signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return encodedPayload + "." + signature
}

func (s *StreamService) ValidateAccess(token string, now time.Time) (StreamTokenClaims, error) {
	payload, err := s.validatePayload(token)
	if err != nil {
		return StreamTokenClaims{}, err
	}
	values := strings.Split(payload, "|")
	if len(values) != 4 {
		return StreamTokenClaims{}, fmt.Errorf("invalid token claims")
	}
	expiresUnix, err := strconv.ParseInt(values[3], 10, 64)
	if err != nil {
		return StreamTokenClaims{}, fmt.Errorf("invalid token expiry")
	}
	claims := StreamTokenClaims{
		TruckID: values[0], CameraID: values[1], Quality: values[2], Expires: time.Unix(expiresUnix, 0),
	}
	if !claims.Expires.After(now) {
		return StreamTokenClaims{}, fmt.Errorf("token expired")
	}
	return claims, nil
}

func (s *StreamService) ValidateRecordingAccess(token string, now time.Time) (RecordingTokenClaims, error) {
	payload, err := s.validatePayload(token)
	if err != nil {
		return RecordingTokenClaims{}, err
	}
	values := strings.Split(payload, "|")
	if len(values) != 6 || values[0] != "recording" {
		return RecordingTokenClaims{}, fmt.Errorf("invalid recording token claims")
	}
	startUnixNano, err := strconv.ParseInt(values[3], 10, 64)
	if err != nil {
		return RecordingTokenClaims{}, fmt.Errorf("invalid recording token start")
	}
	durationNanoseconds, err := strconv.ParseInt(values[4], 10, 64)
	if err != nil || durationNanoseconds < 0 {
		return RecordingTokenClaims{}, fmt.Errorf("invalid recording token duration")
	}
	expiresUnix, err := strconv.ParseInt(values[5], 10, 64)
	if err != nil {
		return RecordingTokenClaims{}, fmt.Errorf("invalid recording token expiry")
	}
	claims := RecordingTokenClaims{
		TruckID:  values[1],
		CameraID: values[2],
		Start:    time.Unix(0, startUnixNano),
		Duration: float64(durationNanoseconds) / float64(time.Second),
		Expires:  time.Unix(expiresUnix, 0),
	}
	if !claims.Expires.After(now) {
		return RecordingTokenClaims{}, fmt.Errorf("recording token expired")
	}
	return claims, nil
}

func (s *StreamService) validatePayload(token string) (string, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid token format")
	}
	mac := hmac.New(sha256.New, []byte(s.signingKey))
	_, _ = mac.Write([]byte(parts[0]))
	actual, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || !hmac.Equal(actual, mac.Sum(nil)) {
		return "", fmt.Errorf("invalid token signature")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return "", fmt.Errorf("invalid token payload")
	}
	return string(payload), nil
}

func (c RecordingTokenClaims) Allows(truckID, cameraID string, start time.Time, duration float64) bool {
	if c.TruckID != truckID || c.CameraID != cameraID || c.Duration <= 0 || duration <= 0 {
		return false
	}
	requestDuration := time.Duration(math.Round(duration * float64(time.Second)))
	claimDuration := time.Duration(math.Round(c.Duration * float64(time.Second)))
	requestStart := start.UTC()
	claimStart := c.Start.UTC()
	return !requestStart.Before(claimStart) && !requestStart.Add(requestDuration).After(claimStart.Add(claimDuration))
}

func RecordingAccessExpiry(now time.Time, duration float64) time.Time {
	return now.Add(time.Duration(math.Ceil(duration))*time.Second + recordingPlaybackBuffer)
}

func (s *StreamService) CameraURL(truckID, cameraID, quality string) string {
	path := url.PathEscape(truckID) + "/" + url.PathEscape(cameraID) + "/" + quality
	return s.publicBaseURL + "/" + path + "/whep"
}

func (s *StreamService) PublishCredentialsValid(user, password, truckID string) bool {
	return user == truckID && hmac.Equal([]byte(password), []byte(s.publishPassword))
}

func StreamPathParts(path string) (truckID, cameraID, quality string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 3 || (parts[2] != "main" && parts[2] != "sub") {
		return "", "", "", false
	}
	return parts[0], parts[1], parts[2], true
}
