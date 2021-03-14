package models

import (
	"bytes"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"strconv"
)

var (
	// ErrMissingClientID when a clientID is empty
	ErrMissingClientID = errors.New("missing clientID")
)

// Imgur encloses http operations of imgur API
type Imgur struct {
	ClientID   string
	httpClient *http.Client
	BaseURL    string
}

// NewImgur returns a imgur struct
func NewImgur(clientID string) (*Imgur, error) {
	if clientID == "" {
		return nil, nil
	}

	return &Imgur{
		ClientID:   clientID,
		httpClient: &http.Client{},
		BaseURL:    "https://api.imgur.com/3/",
	}, nil
}

func (i Imgur) Upload(image []byte, albumHash string) (string, error) {
	var body bytes.Buffer

	writer := multipart.NewWriter(&body)

	fileWriter, err := writer.CreateFormFile("image", "img")
	if err != nil {
		return "", err
	}

	_, err = fileWriter.Write(image)
	if err != nil {
		return "", err
	}

	if albumHash != "" {
		writer.WriteField("album", albumHash)
	}

	err = writer.Close()
	if err != nil {
		return "", err
	}

	request, err := http.NewRequest("POST", i.BaseURL+"image", &body)
	if err != nil {
		return "", err
	}

	request.Header.Add("Authorization", "Client-ID "+i.ClientID)
	request.Header.Add("Content-Type", writer.FormDataContentType())

	type Data struct {
		ID         string `json:"id"`
		Link       string `json:"link"`
		DeleteHash string `json:"deleteHash"`
		Error      string `json:"error,omitempty"`
	}

	type Response struct {
		Success bool `json:"success"`
		Status  int  `jsom:"status"`
		Data    Data `json:"data"`
	}

	response, err := i.httpClient.Do(request)
	if err != nil {
		return "", err
	}

	defer response.Body.Close()

	var uploadResponse Response

	err = json.NewDecoder(response.Body).Decode(&uploadResponse)
	if err != nil {
		return "", err
	}

	status := strconv.Itoa(uploadResponse.Status)

	if !uploadResponse.Success {
		return "", errors.New("error uploading image, status code: " + status + ", error: " + uploadResponse.Data.Error)
	}

	return uploadResponse.Data.Link, nil
}
