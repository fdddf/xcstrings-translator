package translator

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"

	"xcstrings-translator/internal/model"

	"github.com/go-resty/resty/v2"
)

// BaiduTranslator implements the TranslationProvider interface for Baidu Translate API
type BaiduTranslator struct {
	AppID     string
	AppSecret string
	Client    *resty.Client
}

// BaiduTranslateResponse represents the response from Baidu Translate API
type BaiduTranslateResponse struct {
	From        string `json:"from"`
	To          string `json:"to"`
	TransResult []struct {
		Src string `json:"src"`
		Dst string `json:"dst"`
	} `json:"trans_result"`
	ErrorCode string `json:"error_code,omitempty"`
	ErrorMsg  string `json:"error_msg,omitempty"`
}

// NewBaiduTranslator creates a new Baidu Translator instance
func NewBaiduTranslator(appID, appSecret string) *BaiduTranslator {
	client := resty.New()
	client.SetHeader("Content-Type", "application/x-www-form-urlencoded")

	return &BaiduTranslator{
		AppID:     appID,
		AppSecret: appSecret,
		Client:    client,
	}
}

// generateSign generates the signature for Baidu Translate API
func (b *BaiduTranslator) generateSign(q, salt string) string {
	str := fmt.Sprintf("%s%s%s%s", b.AppID, q, salt, b.AppSecret)
	h := md5.New()
	h.Write([]byte(str))
	return hex.EncodeToString(h.Sum(nil))
}

// Translate translates a string using Baidu Translate API
func (b *BaiduTranslator) Translate(ctx context.Context, req model.TranslationRequest) (model.TranslationResponse, error) {
	apiURL := "https://fanyi-api.baidu.com/api/trans/vip/translate"

	// Generate random salt
	salt := strconv.Itoa(rand.Intn(1000000000))

	// Generate sign
	sign := b.generateSign(req.Text, salt)

	// Convert language codes to Baidu format
	sourceLang := convertToBaiduLang(req.SourceLanguage)
	targetLang := convertToBaiduLang(req.TargetLanguage)

	// Prepare form data
	formData := url.Values{}
	formData.Set("q", req.Text)
	formData.Set("from", sourceLang)
	formData.Set("to", targetLang)
	formData.Set("appid", b.AppID)
	formData.Set("salt", salt)
	formData.Set("sign", sign)

	resp, err := b.Client.R().
		SetContext(ctx).
		SetBody(formData.Encode()).
		Post(apiURL)

	if err != nil {
		return model.TranslationResponse{
			Key:            req.Key,
			TargetLanguage: req.TargetLanguage,
			Error:          fmt.Errorf("request failed: %v", err),
		}, nil
	}

	if resp.StatusCode() != http.StatusOK {
		return model.TranslationResponse{
			Key:            req.Key,
			TargetLanguage: req.TargetLanguage,
			Error:          fmt.Errorf("API request failed with status code: %d, response: %s", resp.StatusCode(), resp.String()),
		}, nil
	}

	var translationResponse BaiduTranslateResponse
	err = json.Unmarshal(resp.Body(), &translationResponse)
	if err != nil {
		return model.TranslationResponse{
			Key:            req.Key,
			TargetLanguage: req.TargetLanguage,
			Error:          fmt.Errorf("failed to parse response: %v", err),
		}, nil
	}

	if translationResponse.ErrorCode != "" {
		return model.TranslationResponse{
			Key:            req.Key,
			TargetLanguage: req.TargetLanguage,
			Error:          fmt.Errorf("API error: %s - %s", translationResponse.ErrorCode, translationResponse.ErrorMsg),
		}, nil
	}

	if len(translationResponse.TransResult) == 0 {
		return model.TranslationResponse{
			Key:            req.Key,
			TargetLanguage: req.TargetLanguage,
			Error:          fmt.Errorf("no translation results"),
		}, nil
	}

	return model.TranslationResponse{
		Key:            req.Key,
		TargetLanguage: req.TargetLanguage,
		TranslatedText: translationResponse.TransResult[0].Dst,
	}, nil
}

// convertToBaiduLang converts standard language codes to Baidu format
func convertToBaiduLang(lang string) string {
	switch lang {
	case "zh-Hans":
		return "zh"
	case "zh-Hant":
		return "cht"
	case "en":
		return "en"
	case "ja":
		return "jp"
	case "ko":
		return "kor"
	case "fr":
		return "fra"
	case "de":
		return "de"
	case "es":
		return "spa"
	case "ru":
		return "ru"
	case "pt":
		return "pt"
	case "it":
		return "it"
	case "ar":
		return "ara"
	case "hi":
		return "hi"
	case "auto":
		return "auto"
	default:
		return lang
	}
}
