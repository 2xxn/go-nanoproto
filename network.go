package nanoproto

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"golang.org/x/crypto/blake2b"
)

type AccountHistory struct {
	Account string               `json:"account"`
	History []AccountHistoryItem `json:"history"`
}

type AccountHistoryItem struct {
	Type    string `json:"type"`
	Account string `json:"account"`
	Amount  string `json:"amount"`
}

type AccountHistoryRepresentatives struct {
	Account string                    `json:"account"`
	History []AccountHistoryRepChange `json:"history"`
}

type AccountHistoryRepChange struct {
	Type           string `json:"type"`
	Subtype        string `json:"subtype"`
	Representative string `json:"representative"`
}

type AccountInfo struct {
	Frontier                   string `json:"frontier"`
	OpenBlock                  string `json:"open_block"`
	RepresentativeBlock        string `json:"representative_block"`
	Balance                    string `json:"balance"`
	ModifiedTimestamp          string `json:"modified_timestamp"`
	BlockCount                 string `json:"block_count"`
	AccountVersion             string `json:"account_version"`
	ConfirmationHeight         string `json:"confirmation_height"`
	ConfirmationHeightFrontier string `json:"confirmation_height_frontier"`
}

type RPC struct {
	url    string
	client *http.Client
}

func NewRPC(uri string) *RPC {
	// proxyURL, _ := url.Parse("http://127.0.0.1:8888")
	// proxy := http.ProxyURL(proxyURL)
	// transport := &http.Transport{Proxy: proxy}

	client := &http.Client{
		// Transport: transport,
	}

	return &RPC{uri, client}
}

func (r *RPC) Call(data map[string]interface{}) ([]byte, error) {
	obj, err := json.Marshal(data)
	if err != nil {
		log.Fatal(err)
	}

	body := string(obj)
	contentType := "application/json"

	req, err := http.NewRequest("POST", r.url, bytes.NewBuffer([]byte(body)))

	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Content-Length", fmt.Sprintf("%d", len(body)))
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64; rv:78.0) Gecko/20100101 Firefox/78.0")

	resp, err := r.client.Do(req)

	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bytes, err := io.ReadAll(resp.Body)

	if err != nil {
		return nil, err
	}

	return bytes, nil
}

func (r *RPC) WorkGenerate(hash string) (string, error) {
	data := map[string]interface{}{
		"action": "work_generate",
		"hash":   hash,
	}

	resp, err := r.Call(data)

	if err != nil {
		return "", err
	}

	var x map[string]string

	json.Unmarshal(resp, &x)

	return x["work"], nil
}

func (r *RPC) AccountInfo(address string) (AccountInfo, error) {
	data := map[string]interface{}{
		"action":  "account_info",
		"account": address,
	}

	resp, err := r.Call(data)

	if err != nil {
		return AccountInfo{}, err
	}

	var x AccountInfo

	json.Unmarshal(resp, &x)

	return x, nil
}

func (r *RPC) ChangeRepresentativeBlock(privateKey, address, representative, work, previous, balance string) (map[string]interface{}, error) {
	accountPubHex, err := nanoAddressToPublicKey(address)
	repPubHex, err := nanoAddressToPublicKey(representative)
	if err != nil {
		return nil, fmt.Errorf("failed to convert address to public key: %v", err)
	}

	accountPubBytes, err := hex.DecodeString(accountPubHex)
	repPubBytes, err := hex.DecodeString(repPubHex)
	prevBytes, err := hex.DecodeString(previous)
	if err != nil {
		return nil, fmt.Errorf("invalid previous block hash hex: %v", err)
	}

	preamble := make([]byte, 32)
	preamble[31] = 0x6

	zeroLink := make([]byte, 32) // Link field is all zeroes for a change block

	balanceBytes, err := convertBalanceToBytes(balance)
	if err != nil {
		return nil, fmt.Errorf("failed to get account info: %v", err)
	}

	// Create the block
	hashData := append(preamble, accountPubBytes...) // Account public key
	hashData = append(hashData, prevBytes...)        // Previous block hash
	hashData = append(hashData, repPubBytes...)      // New representative
	hashData = append(hashData, balanceBytes...)     // Balance
	hashData = append(hashData, zeroLink...)         // Link (zero for change block)

	// Hash the block data using BLAKE2b (32-byte digest)
	hasher, _ := blake2b.New(32, nil)
	hasher.Write(hashData)
	blockHash := hasher.Sum(nil)

	// Decode private key from hex and sign the block hash
	ed := NewEd25519()

	privKeyBytes, _ := hex.DecodeString(privateKey)
	signature, _ := ed.Sign(blockHash, privKeyBytes)

	block := map[string]interface{}{
		"type":           "state",
		"account":        address,
		"previous":       previous,
		"representative": representative,
		"balance":        balance, // Balance as string
		"link":           "0000000000000000000000000000000000000000000000000000000000000000",
		"signature":      hex.EncodeToString(signature),
		"work":           work,
	}

	return block, nil
}

func (r *RPC) ProcessChangeRepBlock(block map[string]interface{}) (string, error) {
	data := map[string]interface{}{
		"action":     "process",
		"json_block": "true",
		"subtype":    "change",
		"block":      block,
	}

	resp, err := r.Call(data)

	// log.Println("Processing change rep block")
	// log.Println(string(resp))
	if err != nil {
		return "", err
	}

	var x map[string]string

	json.Unmarshal(resp, &x)

	return x["hash"], nil
}

func (r *RPC) GetAccountInfo(address string) (AccountInfo, error) {
	data := map[string]interface{}{
		"action":  "account_info",
		"account": address,
	}

	resp, err := r.Call(data)

	if err != nil {
		return AccountInfo{}, err
	}

	var x AccountInfo

	json.Unmarshal(resp, &x)

	return x, nil
}

func (r *RPC) History(address string) ([]AccountHistoryRepChange, error) {
	data := map[string]interface{}{
		"action":  "account_history",
		"account": address,
		"count":   200,
		"raw":     true,
	}

	resp, err := r.Call(data)

	if err != nil {
		return []AccountHistoryRepChange{}, err
	}

	var x AccountHistoryRepresentatives

	json.Unmarshal(resp, &x)

	var received []AccountHistoryRepChange

	for _, item := range x.History {
		if item.Type == "change" || (item.Type == "state" && item.Subtype == "change") {
			received = append(received, item)
		}
	}

	for i, j := 0, len(received)-1; i < j; i, j = i+1, j-1 {
		received[i], received[j] = received[j], received[i]
	}

	return received, nil
}

func (r *RPC) Received(address string) ([]AccountHistoryItem, error) {
	data := map[string]interface{}{
		"action":  "account_history",
		"account": address,
		"count":   200,
	}

	resp, err := r.Call(data)

	if err != nil {
		return []AccountHistoryItem{}, err
	}

	var x AccountHistory

	json.Unmarshal(resp, &x)

	var received []AccountHistoryItem

	for _, item := range x.History {
		if item.Type == "receive" {
			received = append(received, item)
		}
	}

	for i, j := 0, len(received)-1; i < j; i, j = i+1, j-1 {
		received[i], received[j] = received[j], received[i]
	}

	return received, nil
}
