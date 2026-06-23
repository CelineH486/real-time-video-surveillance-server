package services

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type StreamTokenClaims struct {
	TruckID  string
	CameraID string
	Quality  string
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
	encodedPayload := base64.RawURLEncoding.EncodeToString([]byte(payload))
	mac := hmac.New(sha256.New, []byte(s.signingKey))
	_, _ = mac.Write([]byte(encodedPayload))
	signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return encodedPayload + "." + signature
}

func (s *StreamService) ValidateAccess(token string, now time.Time) (StreamTokenClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return StreamTokenClaims{}, fmt.Errorf("invalid token format")
	}
	mac := hmac.New(sha256.New, []byte(s.signingKey))
	_, _ = mac.Write([]byte(parts[0]))
	actual, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || !hmac.Equal(actual, mac.Sum(nil)) {
		return StreamTokenClaims{}, fmt.Errorf("invalid token signature")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return StreamTokenClaims{}, fmt.Errorf("invalid token payload")
	}
	values := strings.Split(string(payload), "|")
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
