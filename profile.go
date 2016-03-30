package steam

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
)

const (
	PrivacyStatePrivate     = 1
	PrivacyStateFriendsOnly = 2
	PrivacyStatePublic      = 3
)

const (
	CommentSettingSelf    = "commentselfonly"
	CommentSettingFriends = "commentfriendsonly"
	CommentSettingPublic  = "commentanyone"
)

// NOTE / FIXME: We cannot use this to compare it against err in GetProfileURL()!
var ErrDoNotRedirect = errors.New("do not redirect")

func (session *Session) GetProfileURL() (string, error) {
	/* We do not follow redirect, we want to know where it'd redirect us.  */
	session.client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return errors.New("do not redirect")
	}

	/* Query normal, this will redirect us.  */
	resp, err := session.client.Get("https://steamcommunity.com/my")

	/* We restore redirect policy to default.  */
	session.client.CheckRedirect = nil

	if resp == nil {
		return "", err
	}

	io.Copy(ioutil.Discard, resp.Body)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusFound {
		return "", fmt.Errorf("http error: %d", resp.StatusCode)
	}

	/* We now have a few useful variables in header, for now, we will just grap "Location".  */
	return resp.Header.Get("Location"), nil
}

func (session *Session) SetupProfile(profileURL string) error {
	resp, err := session.client.Get(profileURL + "/edit?welcomed=1")
	if resp != nil {
		io.Copy(ioutil.Discard, resp.Body)
		defer resp.Body.Close()
	}

	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("http error: %d", resp.StatusCode)
	}

	return nil
}

func (session *Session) SetProfileInfo(profileURL string, values *map[string][]string) error {
	(*values)["sessionID"] = []string{session.sessionID}
	(*values)["type"] = []string{"profileSave"}

	resp, err := session.client.PostForm(profileURL+"/edit", *values)
	if resp != nil {
		io.Copy(ioutil.Discard, resp.Body)
		defer resp.Body.Close()
	}

	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("http error: %d", resp.StatusCode)
	}

	return nil
}

func (session *Session) SetProfilePrivacy(profileURL string, commentPrivacy string, privacy uint8) error {
	resp, err := session.client.PostForm(profileURL+"/edit/settings", url.Values{
		"sessionID":               {session.sessionID},
		"type":                    {"profileSettings"},
		"commentSetting":          {commentPrivacy},
		"privacySetting":          {strconv.FormatUint(uint64(privacy&0x3), 10)},
		"inventoryPrivacySetting": {strconv.FormatUint(uint64((privacy>>2)&0x3), 10)},
		"inventoryGiftPrivacy":    {strconv.FormatUint(uint64((privacy>>4)&0x3), 10)},
	})
	if resp != nil {
		io.Copy(ioutil.Discard, resp.Body)
		defer resp.Body.Close()
	}

	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("http error: %d", resp.StatusCode)
	}

	return nil
}
