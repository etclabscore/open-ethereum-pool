package proxy

import (
	"log"
	"math/big"
	"strconv"
	"strings"

	"github.com/etclabscore/go-etchash"
	"github.com/ethereum/go-ethereum/common"
)

var ecip1099FBlockClassic uint64 = 11700000 // classic mainnet
var ecip1099FBlockMordor uint64 = 2520000   // mordor

var hasher *etchash.Etchash = nil

func (s *ProxyServer) processShare(login, id, ip string, t *BlockTemplate, params []string) (bool, bool) {
	if hasher == nil {
		if s.config.Network == "classic" {
			hasher = etchash.New(&ecip1099FBlockClassic, nil)
		} else if s.config.Network == "mordor" {
			hasher = etchash.New(&ecip1099FBlockMordor, nil)
		} else {
			// unknown network
			log.Printf("Unknown network configuration %s", s.config.Network)
			return false, false
		}
	}
	nonceHex := params[0]
	hashNoNonce := params[1]
	mixDigest := params[2]
	nonce, _ := strconv.ParseUint(strings.Replace(nonceHex, "0x", "", -1), 16, 64)
	shareDiff := s.config.Proxy.Difficulty
	stratumHostname := s.config.Proxy.StratumHostname

	h, ok := t.headers[hashNoNonce]
	if !ok {
		log.Printf("Stale share from %v@%v", login, ip)
		// Here we have a stale share, we need to create a redis function as follows
		// CASE1: stale Share
		// s.backend.WriteWorkerShareStatus(login, id, valid bool, stale bool, invalid bool)
                        s.backend.WriteStaleShare(login, nonceHex)
		return false, false
	}

	share := Block{
		number:      h.height,
		hashNoNonce: common.HexToHash(hashNoNonce),
		difficulty:  big.NewInt(shareDiff),
		nonce:       nonce,
		mixDigest:   common.HexToHash(mixDigest),
	}

	block := Block{
		number:      h.height,
		hashNoNonce: common.HexToHash(hashNoNonce),
		difficulty:  h.diff,
		nonce:       nonce,
		mixDigest:   common.HexToHash(mixDigest),
	}

	if !hasher.Verify(share) {
		// THis is an invalid block, record it
		// CASE2: invalid Share
		// s.backend.WriteWorkerShareStatus(login, id, valid bool, stale bool, invalid bool)
        s.backend.WriteInvalidShare(login, nonceHex)

		return false, false
	}

	if hasher.Verify(block) {
		ok, err := s.rpc().SubmitBlock(params)
		if err != nil {
			log.Printf("Block submission failure at height %v for %v: %v", h.height, t.Header, err)
		} else if !ok {
			log.Printf("Block rejected at height %v for %v", h.height, t.Header)
			return false, false
		} else {
			s.fetchBlockTemplate()
			exist, err := s.backend.WriteBlock(login, id, params, shareDiff, h.diff.Int64(), h.height, s.hashrateExpiration, stratumHostname)
			if exist {
				return true, false
			}
			if err != nil {
				log.Println("Failed to insert block candidate into backend:", err)
			} else {
				log.Printf("Inserted block %v to backend", h.height)
			}
			// Here we have a valid share, which is in-fact a block and it is written to db
			log.Printf("Block found by miner %v@%v at height %d", login, ip, h.height)
		}
	} else {
		exist, err := s.backend.WriteShare(login, id, params, shareDiff, h.height, s.hashrateExpiration, stratumHostname)
		if exist {
			return true, false
		}
		if err != nil {
			log.Println("Failed to insert share data into backend:", err)
		}
	}
	return false, true
}
