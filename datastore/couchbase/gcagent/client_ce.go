//  Copyright (c) 2020 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

// +build !enterprise

package gcagent

import (
	"time"
)

// Call this method with a TLS certificate file name to make communication
type Client struct {
}

func NewClient(url, certFile string, defExpirationTime time.Duration) (rv *Client, err error) {
	return nil, ErrCENotSupported
}

func (c *Client) InitTLS(certFile string) error {
	return ErrCENotSupported
}

func (c *Client) ClearTLS() {
}

type AgentProvider struct {
}

func (ap *AgentProvider) Close() error {
	return nil
}

func (ap *AgentProvider) Refresh() {
}
