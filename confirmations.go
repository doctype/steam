package steam

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

type Confirmation struct {
	ID        uint64
	Key       uint64
	Title     string
	Receiving string
	Since     string
}

const offerIDPart = "tradeofferid_"

var (
	ErrConfirmationsUnknownError = errors.New("unknown error occurered finding confirmations")
	ErrCannotFindConfirmations   = errors.New("unable to find confirmations")
	ErrCannotFindDescriptions    = errors.New("unable to find confirmation descriptions")
	ErrConfiramtionsDescMismatch = errors.New("cannot match confirmations with their respective descriptions")
	ErrConfirmationOfferIDFail   = errors.New("unable to get confirmation offer id")
	ErrCannotFindTradeOffer      = errors.New("unable to find tradeoffer div to get offer id for confirmation")
	ErrCannotFindOfferIDAttr     = errors.New("unable to find offer ID attribute")
)

func (community *Community) execConfirmationRequest(request, key, tag string, current int64, values map[string]interface{}) (*http.Response, error) {
	params := url.Values{
		"p":   {community.deviceID},
		"a":   {community.oauth.SteamID.ToString()},
		"k":   {key},
		"t":   {strconv.FormatInt(current, 10)},
		"m":   {"android"},
		"tag": {tag},
	}

	if values != nil {
		for k, v := range values {
			switch v := v.(type) {
			case string:
				params.Add(k, v)
			case uint64:
				params.Add(k, strconv.FormatUint(v, 10))
			default:
				return nil, fmt.Errorf("execConfirmationRequest: missing implementation for type %v", v)
			}
		}
	}

	return community.client.Get("https://steamcommunity.com/mobileconf/" + request + "?" + params.Encode())
}

func (community *Community) GetConfirmations(identitySecret string, current int64) ([]*Confirmation, error) {
	key, err := GenerateConfirmationCode(identitySecret, "conf", current)
	if err != nil {
		return nil, err
	}

	resp, err := community.execConfirmationRequest("conf", key, "conf", current, nil)
	if resp != nil {
		defer resp.Body.Close()
	}

	if err != nil {
		return nil, err
	}

	doc, err := goquery.NewDocumentFromReader(io.Reader(resp.Body))
	if err != nil {
		return nil, err
	}

	/* FIXME: broken
	if empty := doc.Find(".mobileconf_empty"); empty != nil {
		if done := doc.Find(".mobileconf_done"); done != nil {
			return nil, nil
		}

		return nil, ErrConfirmationsUnknownError // FIXME
	}
	*/

	entries := doc.Find(".mobileconf_list_entry")
	if entries == nil {
		return nil, ErrCannotFindConfirmations
	}

	descriptions := doc.Find(".mobileconf_list_entry_description")
	if descriptions == nil {
		return nil, ErrCannotFindDescriptions
	}

	if len(entries.Nodes) != len(descriptions.Nodes) {
		return nil, ErrConfiramtionsDescMismatch
	}

	confirmations := []*Confirmation{}
	for k, sel := range entries.Nodes {
		confirmation := &Confirmation{}
		for _, attr := range sel.Attr {
			if attr.Key == "data-confid" {
				confirmation.ID, _ = strconv.ParseUint(attr.Val, 10, 32)
			} else if attr.Key == "data-key" {
				confirmation.Key, _ = strconv.ParseUint(attr.Val, 10, 64)
			}
		}

		descSel := descriptions.Nodes[k]
		depth := 0
		for child := descSel.FirstChild; child != nil; child = child.NextSibling {
			for n := child.FirstChild; n != nil; n = n.NextSibling {
				switch depth {
				case 0:
					confirmation.Title = n.Data
				case 1:
					confirmation.Receiving = n.Data
				case 2:
					confirmation.Since = n.Data
				}
				depth++
			}
		}

		confirmations = append(confirmations, confirmation)
	}

	return confirmations, nil
}

func (community *Community) GetConfirmationOfferID(identitySecret string, cid uint64, current int64) (uint64, error) {
	key, err := GenerateConfirmationCode(identitySecret, "details", current)
	if err != nil {
		return 0, err
	}

	resp, err := community.execConfirmationRequest(fmt.Sprintf("details/%d", cid), key, "details", current, nil)
	if resp != nil {
		defer resp.Body.Close()
	}

	if err != nil {
		return 0, err
	}

	type Response struct {
		Success bool   `json:"success"`
		HTML    string `json:"html"`
	}

	var response Response
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return 0, err
	}

	if !response.Success {
		return 0, ErrConfirmationOfferIDFail
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(response.HTML))
	if err != nil {
		return 0, err
	}

	offer := doc.Find(".tradeoffer")
	if offer == nil {
		return 0, ErrCannotFindTradeOffer
	}

	val, ok := offer.Attr("id")
	if !ok || len(val) <= len(offerIDPart) || val[:len(offerIDPart)] != offerIDPart {
		return 0, ErrCannotFindOfferIDAttr
	}

	id := val[len(offerIDPart):]
	raw, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		return 0, err
	}

	return raw, nil
}

func (community *Community) AnswerConfirmation(confirmation *Confirmation, identitySecret, answer string, current int64) error {
	key, err := GenerateConfirmationCode(identitySecret, answer, current)
	if err != nil {
		return err
	}

	op := map[string]interface{}{
		"op":  answer,
		"cid": uint64(confirmation.ID),
		"ck":  confirmation.Key,
	}

	resp, err := community.execConfirmationRequest("ajaxop", key, answer, current, op)
	if resp != nil {
		defer resp.Body.Close()
	}

	if err != nil {
		return err
	}

	type Response struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}

	var response Response
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return err
	}

	if !response.Success {
		return errors.New(response.Message)
	}

	return nil
}

func (confirmation *Confirmation) GetOfferID(community *Community, key string, current int64) (uint64, error) {
	return community.GetConfirmationOfferID(key, confirmation.ID, current)
}

func (confirmation *Confirmation) Answer(community *Community, key, answer string, current int64) error {
	return community.AnswerConfirmation(confirmation, key, answer, current)
}
