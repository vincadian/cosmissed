package missed

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"
)

const (
	commitPath        = `/commit?height=`
	validatorsetsPath = `/validatorsets/`
	historicalPath    = `/cosmos/staking/v1beta1/historical_info/`
	timeout           = 2 * time.Second
)

type statusResp struct {
	Result struct {
		NodeInfo struct {
			Network string `json:"network"`
		} `json:"node_info"`
		SyncInfo struct {
			LatestBlockHeight string `json:"latest_block_height"`
			CatchingUp        bool   `json:"catching_up"`
		} `json:"sync_info"`
	} `json:"result"`
}

func CurrentHeight() (curHeight int, networkName string, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", TUrl+"/status", nil)
	if err != nil {
		return 0, "", err
	}
	resp, err := TClient.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, "", err
	}
	sr := &statusResp{}
	err = json.Unmarshal(body, sr)
	if err != nil {
		return 0, "", err
	}
	if sr.Result.SyncInfo.CatchingUp {
		return 0, "", errors.New("node is catching up")
	}
	curHeight, err = strconv.Atoi(sr.Result.SyncInfo.LatestBlockHeight)
	networkName = sr.Result.NodeInfo.Network
	return
}

func fetch(height int, client *http.Client, baseUrl, path string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", baseUrl+path+strconv.Itoa(height), nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func FetchSummary(height int) (*Summary, error) {
	m := minSignatures{}
	b, err := fetch(height, TClient, TUrl, commitPath)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(b, &m)
	proposer, ts, signers := m.parse()
	v := minValidatorSet{}
	b, err = fetch(height, CClient, CUrl, validatorsetsPath)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(b, &v)
	if err != nil {
		return nil, err
	}
	addrs, cons := v.parse()
	b, err = fetch(height, CClient, CUrl, historicalPath)
	if err != nil {
		return nil, err
	}
	vals, err := ParseValidatorsResp(b)
	if err != nil {
		return nil, err
	}
	return summarize(height, ts, proposer, signers, addrs, cons, vals), nil
}
