// Copyright (c) 2017 VMware, Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package clair

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	//	"path"

	"github.com/vmware/harbor/src/common/models"
	"github.com/vmware/harbor/src/common/utils/log"
)

// Client communicates with clair endpoint to scan image and get detailed scan result
type Client struct {
	endpoint string
	//need to customize the logger to write output to job log.
	logger *log.Logger
	client *http.Client
}

// NewClient creates a new instance of client, set the logger as the job's logger if it's used in a job handler.
func NewClient(endpoint string, logger *log.Logger) *Client {
	if logger == nil {
		logger = log.DefaultLogger()
	}
	return &Client{
		endpoint: strings.TrimSuffix(endpoint, "/"),
		logger:   logger,
		client:   &http.Client{},
	}
}

// ScanLayer calls Clair's API to scan a layer.
func (c *Client) ScanLayer(l models.ClairLayer) error {
	layer := models.ClairLayerEnvelope{
		Layer: &l,
		Error: nil,
	}
	data, err := json.Marshal(layer)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", c.endpoint+"/v1/layers", bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set(http.CanonicalHeaderKey("Content-Type"), "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	c.logger.Infof("response code: %d", resp.StatusCode)
	if resp.StatusCode != http.StatusCreated {
		c.logger.Warningf("Unexpected status code: %d", resp.StatusCode)
		return fmt.Errorf("Unexpected status code: %d, text: %s", resp.StatusCode, string(b))
	}
	c.logger.Infof("Returning.")
	return nil
}

// GetResult calls Clair's API to get layers with detailed vulnerability list
func (c *Client) GetResult(layerName string) (*models.ClairLayerEnvelope, error) {
	req, err := http.NewRequest("GET", c.endpoint+"/v1/layers/"+layerName+"?features&vulnerabilities", nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Unexpected status code: %d, text: %s", resp.StatusCode, string(b))
	}
	var res models.ClairLayerEnvelope
	err = json.Unmarshal(b, &res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

// GetNotification calls Clair's API to get details of notification
func (c *Client) GetNotification(id string) (*models.ClairNotification, error) {
	req, err := http.NewRequest("GET", c.endpoint+"/v1/notifications/"+id+"?limit=2", nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Unexpected status code: %d, text: %s", resp.StatusCode, string(b))
	}
	var ne models.ClairNotificationEnvelope
	err = json.Unmarshal(b, &ne)
	if err != nil {
		return nil, err
	}
	if ne.Error != nil {
		return nil, fmt.Errorf("Clair error: %s", ne.Error.Message)
	}
	log.Debugf("Retrived notification %s from Clair.", id)
	return ne.Notification, nil
}

// DeleteNotification deletes a notification record from Clair
func (c *Client) DeleteNotification(id string) error {
	req, err := http.NewRequest("DELETE", c.endpoint+"/v1/notifications/"+id, nil)
	if err != nil {
		return err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Unexpected status code: %d, text: %s", resp.StatusCode, string(b))
	}
	log.Debugf("Deleted notification %s from Clair.", id)
	return nil
}
