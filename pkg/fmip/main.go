package fmip

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strconv"
)

type Credentials struct {
	Username string
	Password string
}

type ISession struct {
	*http.Client
}

var defaultHeaders = map[string]string{
	"Content-Type":          "text/plain",
	"Accept":                "application/json, text/javascript, */*; q=0.01",
	"Connection":            "keep-alive",
	"Accept-Language":       "en-US,en;q=0.9,cs;q=0.8",
	"Origin":                "https://www.icloud.com",
	"X-Apple-Realm-Support": "1.0",
	"X-Apple-Find-API-Ver":  "3.0",
	"User-Agent":            "FindMyiPhone/500 CFNetwork/758.4.3 Darwin/15.5.0",
}

const fmipServer = "fmipmobile.icloud.com"

type FmipResponse struct {
	ServerContext struct {
		AuthToken string `json:"authToken"`
		PrsID     int    `json:"prsId"`
	}
	Content []struct {
		ID                string  `json:"id"`
		Name              string  `json:"name"`
		DeviceDisplayName string  `json:"deviceDisplayName"`
		BatteryLevel      float32 `json:"batteryLevel"`
		BatteryStatus     string  `json:"batteryStatus"`
		Location          struct {
			VerticalAccuracy   float32 `json:"verticalAccuracy"`
			HorizontalAccuracy float32 `json:"horizontalAccuracy"`
			Altitude           float32 `json:"altitude"`
			Longitude          float64 `json:"longitude"`
			Latitude           float64 `json:"latitude"`
		} `json:"location"`
	} `json:"content"`
}

func NewISession() (*ISession, error) {
	conn, err := net.Dial("tcp", fmipServer+":443")
	if err != nil {
		return nil, err
	}

	client := tls.Client(conn, &tls.Config{
		ServerName:         fmipServer,
		InsecureSkipVerify: true,
	})
	defer client.Close()

	if err := client.Handshake(); err != nil {
		return nil, fmt.Errorf("SSL handshake failed: %v", err)
	}

	rootCAs, _ := x509.SystemCertPool()
	if rootCAs == nil {
		rootCAs = x509.NewCertPool()
	}

	rootCAs.AddCert(client.ConnectionState().PeerCertificates[0])

	return &ISession{
		Client: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{RootCAs: rootCAs},
			},
		},
	}, nil
}

func (s *ISession) actionURI(accountName string, action string) string {
	return fmt.Sprintf("https://%s/fmipservice/device/%s/%s", fmipServer, accountName, action)
}

func (s *ISession) makeRequest(accountName string, password string, prsID int) (*FmipResponse, error) {
	action := "initClient"
	authScheme := "UserIDGuest"
	login := accountName

	data := map[string]interface{}{}

	if prsID > 0 {
		login = strconv.Itoa(prsID)
		action = "refreshClient"
		authScheme = "Forever"
	} else {
		data["accountName"] = accountName
	}

	accJSON, _ := json.Marshal(data)

	req, err := http.NewRequest("POST", s.actionURI(login, action), bytes.NewBuffer(accJSON))

	req.SetBasicAuth(login, password)
	req.Header.Set("X-Apple-AuthScheme", authScheme)
	for k, v := range defaultHeaders {
		req.Header.Set(k, v)
	}

	resp, err := s.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("access denied: %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	response := &FmipResponse{}
	if err = json.Unmarshal(body, response); err != nil {
		return nil, err
	}

	return response, nil
}

func (s *ISession) Init(accountName string, password string) (*FmipResponse, error) {
	return s.makeRequest(accountName, password, 0)
}

func (s *ISession) Refresh(accountName string, prsID int, authToken string) (*FmipResponse, error) {
	return s.makeRequest(accountName, authToken, prsID)
}
