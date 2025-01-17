package roblox

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/vinegarhq/vinegar/util"
)

const (
	DefaultChannel  = "live"
	VersionCheckURL = "https://clientsettingscdn.roblox.com/v2/client-version"
)

var (
	ErrNoCDNFound = errors.New("failed to find an accessible roblox deploy mirror")
	ErrNoVersion  = errors.New("no version found")
	CDNURLs       = []string{
		"https://setup.rbxcdn.com",
		"https://s3.amazonaws.com/setup.roblox.com",
		"https://setup-ak.rbxcdn.com",
		"https://setup-cfly.rbxcdn.com",
	}
)

type ClientVersion struct {
	Version                 string `json:"version"`
	ClientVersionUpload     string `json:"clientVersionUpload"`
	BootstrapperVersion     string `json:"bootstrapperVersion"`
	NextClientVersionUpload string `json:"nextClientVersionUpload"`
	NextClientVersion       string `json:"nextClientVersion"`
}

type Version struct {
	Type      BinaryType
	DeployURL string
	GUID      string
}

func FindCDN() (string, error) {
	log.Println("Finding an accessible Roblox deploy mirror")

	for _, cdn := range CDNURLs {
		resp, err := http.Head(cdn + "/" + "version")
		if err != nil {
			continue
		}
		resp.Body.Close()

		if resp.StatusCode == 200 {
			log.Printf("Found deploy mirror: %s", cdn)

			return cdn, nil
		}
	}

	return "", ErrNoCDNFound
}

func ChannelPath(channel string) string {
	// Ensure that the channel is lowercased, since internally in
	// ClientSettings it will be lowercased, but not on the deploy mirror.
	channel = strings.ToLower(channel)

	if channel == DefaultChannel {
		return "/"
	}

	return "/channel/" + channel + "/"
}

func NewVersion(bt BinaryType, channel string, GUID string) (Version, error) {
	if channel == "" {
		channel = DefaultChannel
	}

	if GUID == "" {
		return Version{}, ErrNoVersion
	}

	cdn, err := FindCDN()
	if err != nil {
		return Version{}, fmt.Errorf("failed to find deploy mirror: %w", err)
	}

	log.Printf("Found %s version %s", bt.String(), GUID)

	return Version{
		Type:      bt,
		DeployURL: cdn + ChannelPath(channel) + GUID,
		GUID:      GUID,
	}, nil
}

func LatestVersion(bt BinaryType, channel string) (Version, error) {
	var cv ClientVersion

	if channel == "" {
		channel = DefaultChannel
	}

	url := VersionCheckURL + "/" + bt.String() + ChannelPath(channel)

	log.Printf("Fetching latest version of %s for channel %s (%s)", bt.String(), channel, url)

	resp, err := util.Body(url)
	if err != nil {
		return Version{}, fmt.Errorf("failed to fetch version: %w", err)
	}

	err = json.Unmarshal([]byte(resp), &cv)
	if err != nil {
		return Version{}, fmt.Errorf("failed to unmarshal clientsettings: %w", err)
	}

	if cv.ClientVersionUpload == "" {
		return Version{}, ErrNoVersion
	}

	log.Printf("Fetched %s canonical version %s", bt.String(), cv.Version)

	return NewVersion(bt, channel, cv.ClientVersionUpload)
}
