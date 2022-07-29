package storage

import (
	"fmt"
	"log"
	"math"
	"math/big"
	"sort"
	"strconv"
	"strings"
	"time"

	"gopkg.in/redis.v3"

	"github.com/etclabscore/open-etc-pool/util"
)

type Config struct {
	Endpoint string `json:"endpoint"`
	Password string `json:"password"`
	Database int64  `json:"database"`
	PoolSize int    `json:"poolSize"`
}

type RedisClient struct {
	client   *redis.Client
	prefix   string
	pplns    int64
	CoinName string
}

type PoolCharts struct {
	Timestamp  int64  `json:"x"`
	TimeFormat string `json:"timeFormat"`
	PoolHash   int64  `json:"y"`
}

type MinerCharts struct {
	Timestamp      int64  `json:"x"`
	TimeFormat     string `json:"timeFormat"`
	MinerHash      int64  `json:"minerHash"`
	MinerLargeHash int64  `json:"minerLargeHash"`
	WorkerOnline   string `json:"workerOnline"`
}

type PaymentCharts struct {
	Timestamp  int64  `json:"x"`
	TimeFormat string `json:"timeFormat"`
	Amount     int64  `json:"amount"`
}

type LuckCharts struct {
	Timestamp  int64   `json:"x"`
	Height     int64   `json:"height"`
	Difficulty int64   `json:"difficulty"`
	Shares     int64   `json:"shares"`
	SharesDiff float64 `json:"sharesDiff"`
	Reward     string  `json:"reward"`
}

type SumRewardData struct {
	Interval int64  `json:"inverval"`
	Reward   int64  `json:"reward"`
	Name     string `json:"name"`
	Offset   int64  `json:"offset"`
	Blocks   int64  `json:"blocks"`
}

type RewardData struct {
	Height    int64   `json:"blockheight"`
	Timestamp int64   `json:"timestamp"`
	BlockHash string  `json:"blockhash"`
	Reward    int64   `json:"reward"`
	Percent   float64 `json:"percent"`
	Immature  bool    `json:"immature"`
}

type BlockData struct {
	Login          string   `json:"login"`
	Height         int64    `json:"height"`
	Timestamp      int64    `json:"timestamp"`
	Difficulty     int64    `json:"difficulty"`
	TotalShares    int64    `json:"shares"`
	Uncle          bool     `json:"uncle"`
	UncleHeight    int64    `json:"uncleHeight"`
	Orphan         bool     `json:"orphan"`
	Hash           string   `json:"hash"`
	Nonce          string   `json:"-"`
	PowHash        string   `json:"-"`
	MixDigest      string   `json:"-"`
	Reward         *big.Int `json:"-"`
	ExtraReward    *big.Int `json:"-"`
	ImmatureReward string   `json:"-"`
	RewardString   string   `json:"reward"`
	RoundHeight    int64    `json:"-"`
	candidateKey   string
	immatureKey    string
}

type NetCharts struct {
	Timestamp  int64  `json:"x"`
	TimeFormat string `json:"timeFormat"`
	NetHash    int64  `json:"y"`
}

type ShareCharts struct {
	Timestamp    int64  `json:"x"`
	TimeFormat   string `json:"timeFormat"`
	Valid        int64  `json:"valid"`
	Stale        int64  `json:"stale"`
	WorkerOnline string `json:"workerOnline"`
}

func (b *BlockData) RewardInShannon() int64 {
	reward := new(big.Int).Div(b.Reward, util.Shannon)
	return reward.Int64()
}

func (b *BlockData) serializeHash() string {
	if len(b.Hash) > 0 {
		return b.Hash
	} else {
		return "0x0"
	}
}

func (b *BlockData) RoundKey() string {
	return join(b.RoundHeight, b.Hash)
}

func (b *BlockData) key() string {
	return join(b.UncleHeight, b.Orphan, b.Nonce, b.serializeHash(), b.Timestamp, b.Difficulty, b.TotalShares, b.Reward, b.Login)
}

type Miner struct {
	LastBeat  int64 `json:"lastBeat"`
	HR        int64 `json:"hr"`
	Offline   bool  `json:"offline"`
	startedAt int64
	Blocks    int64 `json:"blocks"`
}

type Worker struct {
	Miner
	TotalHR         int64   `json:"hr2"`
	ValidShares     int64   `json:"valid"`
	StaleShares     int64   `json:"stale"`
	InvalidShares   int64   `json:"invalid"`
	ValidPercent    float64 `json:"v_per"`
	StalePercent    float64 `json:"s_per"`
	InvalidPercent  float64 `json:"i_per"`
	WorkerStatus    int64   `json:"w_stat"`
	WorkerStatushas int64   `json:"w_stat_s"`
}

func NewRedisClient(cfg *Config, prefix string, pplns int64, CoinName string) *RedisClient {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Endpoint,
		Password: cfg.Password,
		DB:       cfg.Database,
		PoolSize: cfg.PoolSize,
	})
	return &RedisClient{client: client, prefix: prefix, pplns: pplns, CoinName: CoinName}
}

func (r *RedisClient) Client() *redis.Client {
	return r.client
}

func (r *RedisClient) Check() (string, error) {
	return r.client.Ping().Result()
}

func (r *RedisClient) BgSave() (string, error) {
	return r.client.BgSave().Result()
}

// Always returns list of addresses. If Redis fails it will return empty list.
func (r *RedisClient) GetBlacklist() ([]string, error) {
	cmd := r.client.SMembers(r.formatKey("blacklist"))
	if cmd.Err() != nil {
		return []string{}, cmd.Err()
	}
	return cmd.Val(), nil
}

// Always returns list of IPs. If Redis fails it will return empty list.
func (r *RedisClient) GetWhitelist() ([]string, error) {
	cmd := r.client.SMembers(r.formatKey("whitelist"))
	if cmd.Err() != nil {
		return []string{}, cmd.Err()
	}
	return cmd.Val(), nil
}

func (r *RedisClient) WritePoolCharts(time1 int64, time2 string, poolHash string) error {
	s := join(time1, time2, poolHash)
	cmd := r.client.ZAdd(r.formatKey("charts", "pool"), redis.Z{Score: float64(time1), Member: s})
	return cmd.Err()
}

func (r *RedisClient) WriteDiffCharts(time1 int64, time2 string, netHash string) error {
	s := join(time1, time2, netHash)
	cmd := r.client.ZAdd(r.formatKey("charts", "difficulty"), redis.Z{Score: float64(time1), Member: s})
	return cmd.Err()
}

func (r *RedisClient) WriteMinerCharts(time1 int64, time2, k string, hash, largeHash, workerOnline int64) error {
	s := join(time1, time2, hash, largeHash, workerOnline)
	cmd := r.client.ZAdd(r.formatKey("charts", "miner", k), redis.Z{Score: float64(time1), Member: s})
	return cmd.Err()
}

func (r *RedisClient) WriteShareCharts(time1 int64, time2, login string, valid, stale, workerOnline int64) error {
	valid_s := r.client.HGet(r.formatKey("chartsNum", "share", login), "valid")
	stale_s := r.client.HGet(r.formatKey("chartsNum", "share", login), "stale")

	if valid_s.Err() == redis.Nil || stale_s.Err() == redis.Nil {
		r.client.HSet(r.formatKey("chartsNum", "share", login), "valid", strconv.FormatInt(0, 10))
		r.client.HSet(r.formatKey("chartsNum", "share", login), "stale", strconv.FormatInt(0, 10))
		//return nil, nil
	} else if valid_s.Err() != nil || stale_s.Err() != nil {
		r.client.HSet(r.formatKey("chartsNum", "share", login), "valid", strconv.FormatInt(0, 10))
		r.client.HSet(r.formatKey("chartsNum", "share", login), "stale", strconv.FormatInt(0, 10))
		//return nil, valid_s.Err()
	}

	v_s, _ := valid_s.Int64()
	s_s, _ := stale_s.Int64()

	l_valid := r.client.HGet(r.formatKey("chartsNum", "share", login), "lastvalid")
	l_stale := r.client.HGet(r.formatKey("chartsNum", "share", login), "laststale")

	if l_valid.Err() == redis.Nil || l_stale.Err() == redis.Nil {
		r.client.HSet(r.formatKey("chartsNum", "share", login), "lastvalid", strconv.FormatInt(0, 10))
		r.client.HSet(r.formatKey("chartsNum", "share", login), "laststale", strconv.FormatInt(0, 10))
		//return nil, nil
	} else if l_valid.Err() != nil || l_stale.Err() != nil {
		r.client.HSet(r.formatKey("chartsNum", "share", login), "lastvalid", strconv.FormatInt(0, 10))
		r.client.HSet(r.formatKey("chartsNum", "share", login), "laststale", strconv.FormatInt(0, 10))
		//return nil, l_valid.Err()
	}
	l_v, _ := l_valid.Int64()
	l_s, _ := l_stale.Int64()

	valid_c := v_s - l_v
	stale_c := s_s - l_s
	s := join(time1, time2, valid_c, stale_c, workerOnline)
	cmd := r.client.ZAdd(r.formatKey("charts", "share", login), redis.Z{Score: float64(time1), Member: s})

	tx := r.client.Multi()
	defer tx.Close()
	tx.Exec(func() error {
		tx.HSet(r.formatKey("chartsNum", "share", login), "lastvalid", strconv.FormatInt(v_s, 10))
		tx.HSet(r.formatKey("chartsNum", "share", login), "laststale", strconv.FormatInt(s_s, 10))
		return nil
	})
	return cmd.Err()
}

func (r *RedisClient) GetPoolCharts(poolHashLen int64) (stats []*PoolCharts, err error) {

	tx := r.client.Multi()
	defer tx.Close()

	now := util.MakeTimestamp() / 1000

	cmds, err := tx.Exec(func() error {
		tx.ZRemRangeByScore(r.formatKey("charts", "pool"), "-inf", fmt.Sprint("(", now-172800))
		tx.ZRevRangeWithScores(r.formatKey("charts", "pool"), 0, poolHashLen)
		return nil
	})

	if err != nil {
		return nil, err
	}

	stats = convertPoolChartsResults(cmds[1].(*redis.ZSliceCmd))
	return stats, nil
}

func convertPoolChartsResults(raw *redis.ZSliceCmd) []*PoolCharts {
	var result []*PoolCharts
	for _, v := range raw.Val() {
		// "Timestamp:TimeFormat:Hash"
		pc := PoolCharts{}
		pc.Timestamp = int64(v.Score)
		str := v.Member.(string)
		pc.TimeFormat = str[strings.Index(str, ":")+1 : strings.LastIndex(str, ":")]
		pc.PoolHash, _ = strconv.ParseInt(str[strings.LastIndex(str, ":")+1:], 10, 64)
		result = append(result, &pc)
	}
	var reverse []*PoolCharts
	for i := len(result) - 1; i >= 0; i-- {
		reverse = append(reverse, result[i])
	}
	return reverse
}

func (r *RedisClient) GetNetCharts(netHashLen int64) (stats []*NetCharts, err error) {

	tx := r.client.Multi()
	defer tx.Close()

	now := util.MakeTimestamp() / 1000

	cmds, err := tx.Exec(func() error {
		tx.ZRemRangeByScore(r.formatKey("charts", "difficulty"), "-inf", fmt.Sprint("(", now-172800))
		tx.ZRevRangeWithScores(r.formatKey("charts", "difficulty"), 0, netHashLen)
		return nil
	})

	if err != nil {
		return nil, err
	}

	stats = convertNetChartsResults(cmds[1].(*redis.ZSliceCmd))
	return stats, nil
}

func convertNetChartsResults(raw *redis.ZSliceCmd) []*NetCharts {
	var result []*NetCharts
	for _, v := range raw.Val() {
		// "Timestamp:TimeFormat:Hash"
		pc := NetCharts{}
		pc.Timestamp = int64(v.Score)
		str := v.Member.(string)
		pc.TimeFormat = str[strings.Index(str, ":")+1 : strings.LastIndex(str, ":")]
		pc.NetHash, _ = strconv.ParseInt(str[strings.LastIndex(str, ":")+1:], 10, 64)
		result = append(result, &pc)
	}

	var reverse []*NetCharts
	for i := len(result) - 1; i >= 0; i-- {
		reverse = append(reverse, result[i])
	}
	return reverse
}

func convertMinerChartsResults(raw *redis.ZSliceCmd) []*MinerCharts {
	var result []*MinerCharts
	for _, v := range raw.Val() {
		// "Timestamp:TimeFormat:Hash:largeHash:workerOnline"
		mc := MinerCharts{}
		mc.Timestamp = int64(v.Score)
		str := v.Member.(string)
		mc.TimeFormat = strings.Split(str, ":")[1]
		mc.MinerHash, _ = strconv.ParseInt(strings.Split(str, ":")[2], 10, 64)
		mc.MinerLargeHash, _ = strconv.ParseInt(strings.Split(str, ":")[3], 10, 64)
		mc.WorkerOnline = strings.Split(str, ":")[4]
		result = append(result, &mc)
	}
	var reverse []*MinerCharts
	for i := len(result) - 1; i >= 0; i-- {
		reverse = append(reverse, result[i])
	}
	return reverse
}

func (r *RedisClient) GetAllMinerAccount() (account []string, err error) {
	var c int64
	for {
		now := util.MakeTimestamp() / 1000
		c, keys, err := r.client.Scan(c, r.formatKey("miners", "*"), now).Result()

		if err != nil {
			return account, err
		}
		for _, key := range keys {
			m := strings.Split(key, ":")
			//if ( len(m) >= 2 && strings.Index(strings.ToLower(m[2]), "0x") == 0) {
			if len(m) >= 2 {
				account = append(account, m[2])
			}
		}
		if c == 0 {
			break
		}
	}
	return account, nil
}

func (r *RedisClient) GetMinerCharts(hashNum int64, login string) (stats []*MinerCharts, err error) {

	tx := r.client.Multi()
	defer tx.Close()
	now := util.MakeTimestamp() / 1000
	cmds, err := tx.Exec(func() error {
		tx.ZRemRangeByScore(r.formatKey("charts", "miner", login), "-inf", fmt.Sprint("(", now-172800))
		tx.ZRevRangeWithScores(r.formatKey("charts", "miner", login), 0, hashNum)
		return nil
	})
	if err != nil {
		return nil, err
	}
	stats = convertMinerChartsResults(cmds[1].(*redis.ZSliceCmd))
	return stats, nil
}

func (r *RedisClient) GetShareCharts(shareNum int64, login string) (stats []*ShareCharts, err error) {

	tx := r.client.Multi()
	defer tx.Close()
	now := util.MakeTimestamp() / 1000
	cmds, err := tx.Exec(func() error {
		tx.ZRemRangeByScore(r.formatKey("charts", "share", login), "-inf", fmt.Sprint("(", now-172800))
		tx.ZRevRangeWithScores(r.formatKey("charts", "share", login), 0, shareNum)
		return nil
	})
	if err != nil {
		return nil, err
	}
	stats = convertShareChartsResults(cmds[1].(*redis.ZSliceCmd))
	return stats, nil
}

func convertShareChartsResults(raw *redis.ZSliceCmd) []*ShareCharts {
	var result []*ShareCharts
	for _, v := range raw.Val() {

		mc := ShareCharts{}
		mc.Timestamp = int64(v.Score)
		str := v.Member.(string)
		mc.TimeFormat = strings.Split(str, ":")[1]
		mc.Valid, _ = strconv.ParseInt(strings.Split(str, ":")[2], 10, 64)
		mc.Stale, _ = strconv.ParseInt(strings.Split(str, ":")[3], 10, 64)
		mc.WorkerOnline = strings.Split(str, ":")[4]
		result = append(result, &mc)
	}
	var reverse []*ShareCharts
	for i := len(result) - 1; i >= 0; i-- {
		reverse = append(reverse, result[i])
	}
	return reverse
}

func (r *RedisClient) GetPaymentCharts(login string) (stats []*PaymentCharts, err error) {

	tx := r.client.Multi()
	defer tx.Close()
	cmds, err := tx.Exec(func() error {
		tx.ZRevRangeWithScores(r.formatKey("payments", login), 0, 360)
		return nil
	})
	if err != nil {
		return nil, err
	}
	stats = convertPaymentChartsResults(cmds[0].(*redis.ZSliceCmd))
	//fmt.Println(stats)
	return stats, nil
}

func (r *RedisClient) WriteNodeState(id string, height uint64, diff *big.Int) error {
	tx := r.client.Multi()
	defer tx.Close()

	now := util.MakeTimestamp() / 1000

	_, err := tx.Exec(func() error {
		tx.HSet(r.formatKey("nodes"), join(id, "name"), id)
		tx.HSet(r.formatKey("nodes"), join(id, "height"), strconv.FormatUint(height, 10))
		tx.HSet(r.formatKey("nodes"), join(id, "difficulty"), diff.String())
		tx.HSet(r.formatKey("nodes"), join(id, "lastBeat"), strconv.FormatInt(now, 10))
		return nil
	})
	return err
}

func (r *RedisClient) GetNodeStates() ([]map[string]interface{}, error) {
	cmd := r.client.HGetAllMap(r.formatKey("nodes"))
	if cmd.Err() != nil {
		return nil, cmd.Err()
	}
	m := make(map[string]map[string]interface{})
	for key, value := range cmd.Val() {
		parts := strings.Split(key, ":")
		if val, ok := m[parts[0]]; ok {
			val[parts[1]] = value
		} else {
			node := make(map[string]interface{})
			node[parts[1]] = value
			m[parts[0]] = node
		}
	}
	v := make([]map[string]interface{}, len(m), len(m))
	i := 0
	for _, value := range m {
		v[i] = value
		i++
	}
	return v, nil
}

func (r *RedisClient) checkPoWExist(height uint64, params []string) (bool, error) {
	// Sweep PoW backlog for previous blocks, we have 3 templates back in RAM
	r.client.ZRemRangeByScore(r.formatKey("pow"), "-inf", fmt.Sprint("(", height-8))
	val, err := r.client.ZAdd(r.formatKey("pow"), redis.Z{Score: float64(height), Member: strings.Join(params, ":")}).Result()
	return val == 0, err
}

func (r *RedisClient) WriteShare(login, id string, params []string, diff int64, height uint64, window time.Duration) (bool, error) {
	exist, err := r.checkPoWExist(height, params)
	if err != nil {
		return false, err
	}
	// Duplicate share, (nonce, powHash, mixDigest) pair exist
	if exist {
		return true, nil
	}
	tx := r.client.Multi()
	defer tx.Close()

	ms := util.MakeTimestamp()
	ts := ms / 1000

	_, err = tx.Exec(func() error {
		r.writeShare(tx, ms, ts, login, id, diff, window)
		tx.HIncrBy(r.formatKey("stats"), "roundShares", diff)
		return nil
	})
	return false, err
}

func (r *RedisClient) WriteBlock(login, id string, params []string, diff, roundDiff int64, height uint64, window time.Duration) (bool, error) {
	exist, err := r.checkPoWExist(height, params)
	if err != nil {
		return false, err
	}
	// Duplicate share, (nonce, powHash, mixDigest) pair exist
	if exist {
		return true, nil
	}
	tx := r.client.Multi()
	defer tx.Close()

	ms := util.MakeTimestamp()
	ts := ms / 1000

	cmds, err := tx.Exec(func() error {
		r.writeShare(tx, ms, ts, login, id, diff, window)
		tx.HSet(r.formatKey("stats"), "lastBlockFound", strconv.FormatInt(ts, 10))
		tx.HDel(r.formatKey("stats"), "roundShares")
		tx.ZIncrBy(r.formatKey("finders"), 1, login)
		tx.HIncrBy(r.formatKey("miners", login), "blocksFound", 1)
		tx.HGetAllMap(r.formatKey("shares", "roundCurrent"))
		tx.Del(r.formatKey("shares", "roundCurrent"))
		tx.LRange(r.formatKey("lastshares"), 0, r.pplns)
		return nil
	})
	r.WriteBlocksFound(ms, ts, login, id, params[0], diff)
	if err != nil {
		return false, err
	} else {

		shares := cmds[len(cmds)-1].(*redis.StringSliceCmd).Val()

		tx2 := r.client.Multi()
		defer tx2.Close()

		totalshares := make(map[string]int64)
		for _, val := range shares {
			totalshares[val] += 1
		}

		_, err := tx2.Exec(func() error {
			for k, v := range totalshares {
				tx2.HIncrBy(r.formatRound(int64(height), params[0]), k, v)
			}
			return nil
		})
		if err != nil {
			return false, err
		}

		sharesMap, _ := cmds[len(cmds)-3].(*redis.StringStringMapCmd).Result()
		totalShares := int64(0)
		for _, v := range sharesMap {
			n, _ := strconv.ParseInt(v, 10, 64)
			totalShares += n
		}
		hashHex := strings.Join(params, ":")
		s := join(hashHex, ts, roundDiff, totalShares, login)
		cmd := r.client.ZAdd(r.formatKey("blocks", "candidates"), redis.Z{Score: float64(height), Member: s})
		return false, cmd.Err()
	}
}

func (r *RedisClient) writeShare(tx *redis.Multi, ms, ts int64, login, id string, diff int64, expire time.Duration) {
	times := int(diff / 1000000000)
	for i := 0; i < times; i++ {
		tx.LPush(r.formatKey("lastshares"), login)
	}
	tx.LTrim(r.formatKey("lastshares"), 0, r.pplns)
	tx.HIncrBy(r.formatKey("shares", "roundCurrent"), login, diff)
	tx.ZAdd(r.formatKey("hashrate"), redis.Z{Score: float64(ts), Member: join(diff, login, id, ms)})
	tx.ZAdd(r.formatKey("hashrate", login), redis.Z{Score: float64(ts), Member: join(diff, id, ms)})
	tx.Expire(r.formatKey("hashrate", login), expire) // Will delete hashrates for miners that gone
	tx.HSet(r.formatKey("miners", login), "lastShare", strconv.FormatInt(ts, 10))
}

func (r *RedisClient) WriteBlocksFound(ms, ts int64, login, id, share string, diff int64) {
	r.client.ZAdd(r.formatKey("worker", "blocks", login), redis.Z{Score: float64(ts), Member: join(diff, share, id, ms)})
}

func (r *RedisClient) formatKey(args ...interface{}) string {
	return join(r.prefix, join(args...))
}

func (r *RedisClient) formatRound(height int64, nonce string) string {
	return r.formatKey("shares", "round"+strconv.FormatInt(height, 10), nonce)
}

func join(args ...interface{}) string {
	s := make([]string, len(args))
	for i, v := range args {
		switch v.(type) {
		case string:
			s[i] = v.(string)
		case int64:
			s[i] = strconv.FormatInt(v.(int64), 10)
		case uint64:
			s[i] = strconv.FormatUint(v.(uint64), 10)
		case float64:
			s[i] = strconv.FormatFloat(v.(float64), 'f', 0, 64)
		case bool:
			if v.(bool) {
				s[i] = "1"
			} else {
				s[i] = "0"
			}
		case *big.Rat:
			x := v.(*big.Rat)
			if x != nil {
				s[i] = x.FloatString(9)
			} else {
				s[i] = "0"
			}
		case *big.Int:
			n := v.(*big.Int)
			if n != nil {
				s[i] = n.String()
			} else {
				s[i] = "0"
			}
		default:
			panic("Invalid type specified for conversion")
		}
	}
	return strings.Join(s, ":")
}

func (r *RedisClient) GetCandidates(maxHeight int64) ([]*BlockData, error) {
	option := redis.ZRangeByScore{Min: "0", Max: strconv.FormatInt(maxHeight, 10)}
	cmd := r.client.ZRangeByScoreWithScores(r.formatKey("blocks", "candidates"), option)
	if cmd.Err() != nil {
		return nil, cmd.Err()
	}
	return convertCandidateResults(cmd), nil
}

func (r *RedisClient) GetImmatureBlocks(maxHeight int64) ([]*BlockData, error) {
	option := redis.ZRangeByScore{Min: "0", Max: strconv.FormatInt(maxHeight, 10)}
	cmd := r.client.ZRangeByScoreWithScores(r.formatKey("blocks", "immature"), option)
	if cmd.Err() != nil {
		return nil, cmd.Err()
	}
	return convertBlockResults(cmd), nil
}

func (r *RedisClient) GetRewards(login string) ([]*RewardData, error) {
	option := redis.ZRangeByScore{Min: "0", Max: strconv.FormatInt(10, 10)}
	cmd := r.client.ZRangeByScoreWithScores(r.formatKey("rewards", login), option)
	if cmd.Err() != nil {
		return nil, cmd.Err()
	}
	return convertRewardResults(cmd), nil
}

func (r *RedisClient) GetRoundShares(height int64, nonce string) (map[string]int64, error) {
	result := make(map[string]int64)
	cmd := r.client.HGetAllMap(r.formatRound(height, nonce))
	if cmd.Err() != nil {
		return nil, cmd.Err()
	}
	sharesMap, _ := cmd.Result()
	for login, v := range sharesMap {
		n, _ := strconv.ParseInt(v, 10, 64)
		result[login] = n
	}
	return result, nil
}

func (r *RedisClient) GetPayees() ([]string, error) {
	payees := make(map[string]struct{})
	var result []string
	var c int64

	for {
		var keys []string
		var err error
		c, keys, err = r.client.Scan(c, r.formatKey("miners", "*"), 100).Result()
		if err != nil {
			return nil, err
		}
		for _, row := range keys {
			login := strings.Split(row, ":")[2]
			payees[login] = struct{}{}
		}
		if c == 0 {
			break
		}
	}
	for login, _ := range payees {
		result = append(result, login)
	}
	return result, nil
}

func (r *RedisClient) GetTotalShares() (int64, error) {
	cmd := r.client.LLen(r.formatKey("lastshares"))
	if cmd.Err() == redis.Nil {
		return 0, nil
	} else if cmd.Err() != nil {
		return 0, cmd.Err()
	}
	return cmd.Val(), nil
}

func (r *RedisClient) GetBalance(login string) (int64, error) {
	cmd := r.client.HGet(r.formatKey("miners", login), "balance")
	if cmd.Err() == redis.Nil {
		return 0, nil
	} else if cmd.Err() != nil {
		return 0, cmd.Err()
	}
	return cmd.Int64()
}

func (r *RedisClient) LockPayouts(login string, amount int64) error {
	key := r.formatKey("payments", "lock")
	result := r.client.SetNX(key, join(login, amount), 0).Val()
	if !result {
		return fmt.Errorf("Unable to acquire lock '%s'", key)
	}
	return nil
}

func (r *RedisClient) UnlockPayouts() error {
	key := r.formatKey("payments", "lock")
	_, err := r.client.Del(key).Result()
	return err
}

func (r *RedisClient) IsPayoutsLocked() (bool, error) {
	_, err := r.client.Get(r.formatKey("payments", "lock")).Result()
	if err == redis.Nil {
		return false, nil
	} else if err != nil {
		return false, err
	} else {
		return true, nil
	}
}

type PendingPayment struct {
	Timestamp int64  `json:"timestamp"`
	Amount    int64  `json:"amount"`
	Address   string `json:"login"`
}

func (r *RedisClient) GetPendingPayments() []*PendingPayment {
	raw := r.client.ZRevRangeWithScores(r.formatKey("payments", "pending"), 0, -1)
	var result []*PendingPayment
	for _, v := range raw.Val() {
		// timestamp -> "address:amount"
		payment := PendingPayment{}
		payment.Timestamp = int64(v.Score)
		fields := strings.Split(v.Member.(string), ":")
		payment.Address = fields[0]
		payment.Amount, _ = strconv.ParseInt(fields[1], 10, 64)
		result = append(result, &payment)
	}
	return result
}

// Deduct miner's balance for payment
func (r *RedisClient) UpdateBalance(login string, amount int64) error {
	tx := r.client.Multi()
	defer tx.Close()

	ts := util.MakeTimestamp() / 1000

	_, err := tx.Exec(func() error {
		tx.HIncrBy(r.formatKey("miners", login), "balance", (amount * -1))
		tx.HIncrBy(r.formatKey("miners", login), "pending", amount)
		tx.HIncrBy(r.formatKey("finances"), "balance", (amount * -1))
		tx.HIncrBy(r.formatKey("finances"), "pending", amount)
		tx.ZAdd(r.formatKey("payments", "pending"), redis.Z{Score: float64(ts), Member: join(login, amount)})
		return nil
	})
	return err
}

func (r *RedisClient) RollbackBalance(login string, amount int64) error {
	tx := r.client.Multi()
	defer tx.Close()

	_, err := tx.Exec(func() error {
		tx.HIncrBy(r.formatKey("miners", login), "balance", amount)
		tx.HIncrBy(r.formatKey("miners", login), "pending", (amount * -1))
		tx.HIncrBy(r.formatKey("finances"), "balance", amount)
		tx.HIncrBy(r.formatKey("finances"), "pending", (amount * -1))
		tx.ZRem(r.formatKey("payments", "pending"), join(login, amount))
		return nil
	})
	return err
}

func (r *RedisClient) WritePayment(login, txHash string, amount int64) error {
	tx := r.client.Multi()
	defer tx.Close()

	ts := util.MakeTimestamp() / 1000

	_, err := tx.Exec(func() error {
		tx.HIncrBy(r.formatKey("miners", login), "pending", (amount * -1))
		tx.HIncrBy(r.formatKey("miners", login), "paid", amount)
		tx.HIncrBy(r.formatKey("finances"), "pending", (amount * -1))
		tx.HIncrBy(r.formatKey("finances"), "paid", amount)
		tx.ZAdd(r.formatKey("payments", "all"), redis.Z{Score: float64(ts), Member: join(txHash, login, amount)})
		tx.ZRemRangeByRank(r.formatKey("payments", "all"), 0, -10000)
		tx.ZAdd(r.formatKey("payments", login), redis.Z{Score: float64(ts), Member: join(txHash, amount)})
		tx.ZRemRangeByRank(r.formatKey("payments", login), 0, -100)
		tx.ZRem(r.formatKey("payments", "pending"), join(login, amount))
		tx.Del(r.formatKey("payments", "lock"))
		tx.HIncrBy(r.formatKey("paymentsTotal"), "all", 1)
		tx.HIncrBy(r.formatKey("paymentsTotal"), login, 1)
		return nil
	})
	return err
}

func (r *RedisClient) WriteReward(login string, amount int64, percent *big.Rat, immature bool, block *BlockData) error {
	if amount <= 0 {
		return nil
	}
	tx := r.client.Multi()
	defer tx.Close()

	addStr := join(amount, percent, immature, block.Hash, block.Height, block.Timestamp)
	remStr := join(amount, percent, !immature, block.Hash, block.Height, block.Timestamp)
	remscore := block.Timestamp - 3600*24*40 // Store the last 40 Days

	_, err := tx.Exec(func() error {
		tx.ZAdd(r.formatKey("rewards", login), redis.Z{Score: float64(block.Timestamp), Member: addStr})
		tx.ZRem(r.formatKey("rewards", login), remStr)
		tx.ZRemRangeByScore(r.formatKey("rewards", login), "-inf", "("+strconv.FormatInt(remscore, 10))

		return nil
	})
	return err
}

func (r *RedisClient) WriteImmatureBlock(block *BlockData, roundRewards map[string]int64) error {
	tx := r.client.Multi()
	defer tx.Close()

	_, err := tx.Exec(func() error {
		r.writeImmatureBlock(tx, block)
		total := int64(0)
		for login, amount := range roundRewards {
			total += amount
			tx.HIncrBy(r.formatKey("miners", login), "immature", amount)
			tx.HSetNX(r.formatKey("credits", "immature", block.Height, block.Hash), login, strconv.FormatInt(amount, 10))
		}
		tx.HIncrBy(r.formatKey("finances"), "immature", total)
		return nil
	})
	return err
}

func (r *RedisClient) WriteMaturedBlock(block *BlockData, roundRewards map[string]int64) error {
	creditKey := r.formatKey("credits", "immature", block.RoundHeight, block.Hash)
	tx, err := r.client.Watch(creditKey)
	// Must decrement immatures using existing log entry
	immatureCredits := tx.HGetAllMap(creditKey)
	if err != nil {
		return err
	}
	defer tx.Close()

	ts := util.MakeTimestamp() / 1000
	value := join(block.Hash, ts, block.Reward)

	_, err = tx.Exec(func() error {
		r.writeMaturedBlock(tx, block)
		tx.ZAdd(r.formatKey("credits", "all"), redis.Z{Score: float64(block.Height), Member: value})

		// Decrement immature balances
		totalImmature := int64(0)
		for login, amountString := range immatureCredits.Val() {
			amount, _ := strconv.ParseInt(amountString, 10, 64)
			totalImmature += amount
			tx.HIncrBy(r.formatKey("miners", login), "immature", (amount * -1))
		}

		// Increment balances
		total := int64(0)
		for login, amount := range roundRewards {
			total += amount
			// NOTICE: Maybe expire round reward entry in 604800 (a week)?
			tx.HIncrBy(r.formatKey("miners", login), "balance", amount)
			tx.HSetNX(r.formatKey("credits", block.Height, block.Hash), login, strconv.FormatInt(amount, 10))
		}
		tx.Del(creditKey)
		tx.HIncrBy(r.formatKey("finances"), "balance", total)
		tx.HIncrBy(r.formatKey("finances"), "immature", (totalImmature * -1))
		tx.HSet(r.formatKey("finances"), "lastCreditHeight", strconv.FormatInt(block.Height, 10))
		tx.HSet(r.formatKey("finances"), "lastCreditHash", block.Hash)
		tx.HIncrBy(r.formatKey("finances"), "totalMined", block.RewardInShannon())
		tx.Expire(r.formatKey("credits", block.Height, block.Hash), 604800*time.Second)
		return nil
	})
	return err
}

func (r *RedisClient) WriteOrphan(block *BlockData) error {
	creditKey := r.formatKey("credits", "immature", block.RoundHeight, block.Hash)
	tx, err := r.client.Watch(creditKey)
	// Must decrement immatures using existing log entry
	immatureCredits := tx.HGetAllMap(creditKey)
	if err != nil {
		return err
	}
	defer tx.Close()

	_, err = tx.Exec(func() error {
		r.writeMaturedBlock(tx, block)

		// Decrement immature balances
		totalImmature := int64(0)
		for login, amountString := range immatureCredits.Val() {
			amount, _ := strconv.ParseInt(amountString, 10, 64)
			totalImmature += amount
			tx.HIncrBy(r.formatKey("miners", login), "immature", (amount * -1))
		}
		tx.Del(creditKey)
		tx.HIncrBy(r.formatKey("finances"), "immature", (totalImmature * -1))
		return nil
	})
	return err
}

func (r *RedisClient) WritePendingOrphans(blocks []*BlockData) error {
	tx := r.client.Multi()
	defer tx.Close()

	_, err := tx.Exec(func() error {
		for _, block := range blocks {
			r.writeImmatureBlock(tx, block)
		}
		return nil
	})
	return err
}

func (r *RedisClient) writeImmatureBlock(tx *redis.Multi, block *BlockData) {
	// Redis 2.8.x returns "ERR source and destination objects are the same"
	if block.Height != block.RoundHeight {
		tx.Rename(r.formatRound(block.RoundHeight, block.Nonce), r.formatRound(block.Height, block.Nonce))
	}
	tx.ZRem(r.formatKey("blocks", "candidates"), block.candidateKey)
	tx.ZAdd(r.formatKey("blocks", "immature"), redis.Z{Score: float64(block.Height), Member: block.key()})
}

func (r *RedisClient) writeMaturedBlock(tx *redis.Multi, block *BlockData) {
	tx.Del(r.formatRound(block.RoundHeight, block.Nonce))
	tx.ZRem(r.formatKey("blocks", "immature"), block.immatureKey)
	tx.ZAdd(r.formatKey("blocks", "matured"), redis.Z{Score: float64(block.Height), Member: block.key()})
}

func (r *RedisClient) IsMinerExists(login string) (bool, error) {
	return r.client.Exists(r.formatKey("miners", login)).Result()
}

func (r *RedisClient) GetMinerStats(login string, maxPayments int64) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	tx := r.client.Multi()
	defer tx.Close()

	cmds, err := tx.Exec(func() error {
		tx.HGetAllMap(r.formatKey("miners", login))
		tx.ZRevRangeWithScores(r.formatKey("payments", login), 0, maxPayments-1)
		tx.HGet(r.formatKey("paymentsTotal"), login)
		tx.HGet(r.formatKey("shares", "currentShares"), login)
		tx.LRange(r.formatKey("lastshares"), 0, r.pplns)
		tx.ZRevRangeWithScores(r.formatKey("rewards", login), 0, 39)
		tx.ZRevRangeWithScores(r.formatKey("rewards", login), 0, -1)
		return nil
	})

	if err != nil && err != redis.Nil {
		return nil, err
	} else {
		result, _ := cmds[0].(*redis.StringStringMapCmd).Result()
		stats["stats"] = convertStringMap(result)
		payments := convertPaymentsResults(cmds[1].(*redis.ZSliceCmd))
		stats["payments"] = payments
		stats["paymentsTotal"], _ = cmds[2].(*redis.StringCmd).Int64()
		shares := cmds[4].(*redis.StringSliceCmd).Val()
		csh := 0
		for _, val := range shares {
			if val == login {
				csh++
			}
		}
		stats["roundShares"] = csh
	}

	return stats, nil
}

// Try to convert all numeric strings to int64
func convertStringMap(m map[string]string) map[string]interface{} {
	result := make(map[string]interface{})
	var err error
	for k, v := range m {
		result[k], err = strconv.ParseInt(v, 10, 64)
		if err != nil {
			result[k] = v
		}
	}
	return result
}

// WARNING: Must run it periodically to flush out of window hashrate entries
func (r *RedisClient) FlushStaleStats(window, largeWindow time.Duration) (int64, error) {
	now := util.MakeTimestamp() / 1000
	max := fmt.Sprint("(", now-int64(window/time.Second))
	total, err := r.client.ZRemRangeByScore(r.formatKey("hashrate"), "-inf", max).Result()
	if err != nil {
		return total, err
	}

	var c int64
	miners := make(map[string]struct{})
	max = fmt.Sprint("(", now-int64(largeWindow/time.Second))

	for {
		var keys []string
		var err error
		c, keys, err = r.client.Scan(c, r.formatKey("hashrate", "*"), 100).Result()
		if err != nil {
			return total, err
		}
		for _, row := range keys {
			login := strings.Split(row, ":")[2]
			if _, ok := miners[login]; !ok {
				n, err := r.client.ZRemRangeByScore(r.formatKey("hashrate", login), "-inf", max).Result()
				if err != nil {
					return total, err
				}
				miners[login] = struct{}{}
				total += n
			}
		}
		if c == 0 {
			break
		}
	}
	return total, nil
}

func (r *RedisClient) CollectStats(smallWindow time.Duration, maxBlocks, maxPayments int64) (map[string]interface{}, error) {
	window := int64(smallWindow / time.Second)
	stats := make(map[string]interface{})

	tx := r.client.Multi()
	defer tx.Close()

	now := util.MakeTimestamp() / 1000

	cmds, err := tx.Exec(func() error {
		tx.ZRemRangeByScore(r.formatKey("hashrate"), "-inf", fmt.Sprint("(", now-window))
		tx.ZRangeWithScores(r.formatKey("hashrate"), 0, -1)
		tx.HGetAllMap(r.formatKey("stats"))
		tx.ZRevRangeWithScores(r.formatKey("blocks", "candidates"), 0, -1)
		tx.ZRevRangeWithScores(r.formatKey("blocks", "immature"), 0, -1)
		tx.ZRevRangeWithScores(r.formatKey("blocks", "matured"), 0, maxBlocks-1)
		tx.ZCard(r.formatKey("blocks", "candidates"))
		tx.ZCard(r.formatKey("blocks", "immature"))
		tx.ZCard(r.formatKey("blocks", "matured"))
		tx.HGet(r.formatKey("paymentsTotal"), "all")
		tx.ZRevRangeWithScores(r.formatKey("payments", "all"), 0, maxPayments-1)
		tx.LLen(r.formatKey("lastshares"))
		tx.ZRevRangeWithScores(r.formatKey("finders"), 0, -1)
		tx.HGetAllMap(r.formatKey("exchange", r.CoinName))
		return nil
	})

	if (err != nil) && (err != redis.Nil) {
		return nil, err
	}

	result, _ := cmds[2].(*redis.StringStringMapCmd).Result()
	result["nShares"] = strconv.FormatInt(cmds[11].(*redis.IntCmd).Val(), 10)
	stats["stats"] = convertStringMap(result)
	candidates := convertCandidateResults(cmds[3].(*redis.ZSliceCmd))
	stats["candidates"] = candidates
	stats["candidatesTotal"] = cmds[6].(*redis.IntCmd).Val()

	immature := convertBlockResults(cmds[4].(*redis.ZSliceCmd))
	stats["immature"] = immature
	stats["immatureTotal"] = cmds[7].(*redis.IntCmd).Val()

	matured := convertBlockResults(cmds[5].(*redis.ZSliceCmd))
	stats["matured"] = matured
	stats["maturedTotal"] = cmds[8].(*redis.IntCmd).Val()

	payments := convertPaymentsResults(cmds[10].(*redis.ZSliceCmd))
	stats["payments"] = payments
	stats["paymentsTotal"], _ = cmds[9].(*redis.StringCmd).Int64()

	finders := convertFindersResults(cmds[12].(*redis.ZSliceCmd))
	stats["finders"] = finders

	totalHashrate, miners := convertMinersStats(window, cmds[1].(*redis.ZSliceCmd))
	stats["miners"] = miners
	stats["minersTotal"] = len(miners)
	stats["hashrate"] = totalHashrate

	exchangedata, _ := cmds[13].(*redis.StringStringMapCmd).Result()
	stats["exchangedata"] = exchangedata

	return stats, nil
}

func (r *RedisClient) CollectWorkersStats(sWindow, lWindow time.Duration, login string) (map[string]interface{}, error) {
	smallWindow := int64(sWindow / time.Second)
	largeWindow := int64(lWindow / time.Second)
	stats := make(map[string]interface{})

	tx := r.client.Multi()
	defer tx.Close()

	now := util.MakeTimestamp() / 1000

	cmds, err := tx.Exec(func() error {
		tx.ZRemRangeByScore(r.formatKey("hashrate", login), "-inf", fmt.Sprint("(", now-largeWindow))
		tx.ZRangeWithScores(r.formatKey("hashrate", login), 0, -1)
		tx.ZRevRangeWithScores(r.formatKey("rewards", login), 0, 39)
		tx.ZRevRangeWithScores(r.formatKey("rewards", login), 0, -1)
		tx.ZRangeWithScores(r.formatKey("worker", "blocks", login), 0, -1)
		return nil
	})

	if err != nil {
		return nil, err
	}

	totalHashrate := int64(0)
	currentHashrate := int64(0)
	online := int64(0)
	offline := int64(0)
	workers := convertWorkersStats(smallWindow, cmds[1].(*redis.ZSliceCmd), cmds[4].(*redis.ZSliceCmd), login, r)

	for id, worker := range workers {
		timeOnline := now - worker.startedAt
		if timeOnline < 600 {
			timeOnline = 600
		}

		boundary := timeOnline
		if timeOnline >= smallWindow {
			boundary = smallWindow
		}
		worker.HR = worker.HR / boundary

		boundary = timeOnline
		if timeOnline >= largeWindow {
			boundary = largeWindow
		}
		worker.TotalHR = worker.TotalHR / boundary

		if worker.LastBeat < (now - smallWindow/2) {
			worker.Offline = true
			offline++
		} else {
			online++
		}

		blocks := cmds[4].(*redis.ZSliceCmd).Val()

		for _, val := range blocks {
			parts := strings.Split(val.Member.(string), ":")
			rig := parts[2]
			if id == rig {
				str := fmt.Sprint(val.Member.(string))
				if worker.LastBeat < (now - largeWindow) {
					tx.ZRem(r.formatKey("worker", "blocks", login), str)
				}
			}
		}

		currentHashrate += worker.HR
		totalHashrate += worker.TotalHR
		valid_share, stale_share, invalid_share, _ := r.getSharesStatus(login, id)
		worker.ValidShares = int64(5)
		worker.StaleShares = int64(5)
		worker.InvalidShares = int64(5)
		worker.ValidShares = valid_share
		worker.StaleShares = stale_share
		worker.InvalidShares = invalid_share
		//test percentage
		worker.ValidPercent = float64(0)
		worker.StalePercent = float64(0)
		worker.InvalidPercent = float64(0)
		tot_share := int64(0)
		tot_share += valid_share
		tot_share += stale_share
		tot_share += invalid_share
		if tot_share > 0 {
			d := float64(100)
			//tot_share += ////error
			cost_per := float64(tot_share) / d
			v_per := float64(valid_share) / cost_per
			worker.ValidPercent = toFixed(v_per, 1)
			s_per := float64(stale_share) / cost_per
			worker.StalePercent = toFixed(s_per, 1)
			i_per := float64(invalid_share) / cost_per
			worker.InvalidPercent = toFixed(i_per, 1)
		} else {
			worker.ValidPercent = toFixed(0, 1)
			worker.StalePercent = toFixed(0, 1)
			worker.InvalidPercent = toFixed(0, 1)
		}
		w_stat := int64(0) //test worker large hashrate indicator
		if worker.HR >= worker.TotalHR {
			w_stat = 1
			worker.WorkerStatus = w_stat
		} else if worker.HR < worker.TotalHR {
			w_stat = 0
			worker.WorkerStatus = w_stat
		}
		///test small hr
		tot_w := r.client.HGet(r.formatKey("minerShare", login, id), "hashrate")

		if tot_w.Err() == redis.Nil {
			tx.HSet(r.formatKey("minerShare", login, id), "hashrate", strconv.FormatInt(0, 10))
			//return nil, nil
		} else if tot_w.Err() != nil {
			tx.HSet(r.formatKey("minerShare", login, id), "hashrate", strconv.FormatInt(0, 10))
			//return nil, tot_w.Err()
		}

		last_hr, _ := tot_w.Int64()
		w_stat_s := int64(0) //test worker hashrate indicator
		if worker.HR > last_hr {
			w_stat_s = 1
			worker.WorkerStatushas = w_stat_s
		} else if worker.HR <= last_hr {
			w_stat_s = 0
			worker.WorkerStatushas = w_stat_s
		}
		tx.HSet(r.formatKey("minerShare", login, id), "hashrate", strconv.FormatInt(worker.HR, 10))
		workers[id] = worker
	}

	stats["workers"] = workers
	stats["workersTotal"] = len(workers)
	stats["workersOnline"] = online
	stats["workersOffline"] = offline
	stats["hashrate"] = totalHashrate
	stats["currentHashrate"] = currentHashrate

	stats["rewards"] = convertRewardResults(cmds[2].(*redis.ZSliceCmd)) // last 40
	rewards := convertRewardResults(cmds[3].(*redis.ZSliceCmd))         // all

	var dorew []*SumRewardData
	dorew = append(dorew, &SumRewardData{Name: "Last 60 minutes", Interval: 3600, Offset: 0})
	dorew = append(dorew, &SumRewardData{Name: "Last 12 hours", Interval: 3600 * 12, Offset: 0})
	dorew = append(dorew, &SumRewardData{Name: "Last 24 hours", Interval: 3600 * 24, Offset: 0})
	dorew = append(dorew, &SumRewardData{Name: "Last 7 days", Interval: 3600 * 24 * 7, Offset: 0})
	dorew = append(dorew, &SumRewardData{Name: "Last 30 days", Interval: 3600 * 24 * 30, Offset: 0})

	for _, reward := range rewards {

		for _, dore := range dorew {
			dore.Reward += 0
			dore.Blocks += 0
			if reward.Timestamp > now-dore.Interval {
				dore.Reward += reward.Reward
				dore.Blocks++
			}
		}
	}
	stats["sumrewards"] = dorew
	stats["24hreward"] = dorew[2].Reward
	return stats, nil
}

func round(num float64) int {
	return int(num + math.Copysign(0.5, num))
}

func toFixed(num float64, precision int) float64 {
	output := math.Pow(10, float64(precision))
	return float64(round(num*output)) / output
}

func (r *RedisClient) CollectLuckStats(windows []int) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	tx := r.client.Multi()
	defer tx.Close()

	max := int64(windows[len(windows)-1])

	cmds, err := tx.Exec(func() error {
		tx.ZRevRangeWithScores(r.formatKey("blocks", "immature"), 0, -1)
		tx.ZRevRangeWithScores(r.formatKey("blocks", "matured"), 0, max-1)
		return nil
	})
	if err != nil {
		return stats, err
	}
	blocks := convertBlockResults(cmds[0].(*redis.ZSliceCmd), cmds[1].(*redis.ZSliceCmd))

	calcLuck := func(max int) (int, float64, float64, float64) {
		var total int
		var sharesDiff, uncles, orphans float64
		for i, block := range blocks {
			if i > (max - 1) {
				break
			}
			if block.Uncle {
				uncles++
			}
			if block.Orphan {
				orphans++
			}
			sharesDiff += float64(block.TotalShares) / float64(block.Difficulty)
			total++
		}
		if total > 0 {
			sharesDiff /= float64(total)
			uncles /= float64(total)
			orphans /= float64(total)
		}
		return total, sharesDiff, uncles, orphans
	}
	for _, max := range windows {
		total, sharesDiff, uncleRate, orphanRate := calcLuck(max)
		row := map[string]float64{
			"luck": sharesDiff, "uncleRate": uncleRate, "orphanRate": orphanRate,
		}
		stats[strconv.Itoa(total)] = row
		if total < max {
			break
		}
	}
	return stats, nil
}

func (r *RedisClient) CollectLuckCharts(max int) (stats []*LuckCharts, err error) {
	var result []*LuckCharts
	tx := r.client.Multi()
	defer tx.Close()

	cmds, err := tx.Exec(func() error {
		tx.ZRevRangeWithScores(r.formatKey("blocks", "matured"), 0, int64(max-1))
		return nil
	})
	if err != nil {
		return result, err
	}
	blocks := convertBlockResults(cmds[0].(*redis.ZSliceCmd))

	for i, block := range blocks {
		if i > (max - 1) {
			break
		}
		lc := LuckCharts{}
		var sharesDiff = float64(block.TotalShares) / float64(block.Difficulty)
		lc.Timestamp = block.Timestamp
		lc.Height = block.RoundHeight
		lc.Difficulty = block.Difficulty
		lc.Shares = block.TotalShares
		lc.SharesDiff = sharesDiff
		lc.Reward = block.RewardString
		result = append(result, &lc)
	}
	sort.Sort(TimestampSorter(result))
	return result, nil
}

type TimestampSorter []*LuckCharts

func (a TimestampSorter) Len() int           { return len(a) }
func (a TimestampSorter) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a TimestampSorter) Less(i, j int) bool { return a[i].Timestamp < a[j].Timestamp }

func convertCandidateResults(raw *redis.ZSliceCmd) []*BlockData {
	var result []*BlockData
	for _, v := range raw.Val() {
		// "nonce:powHash:mixDigest:timestamp:diff:totalShares"
		block := BlockData{}
		block.Height = int64(v.Score)
		block.RoundHeight = block.Height
		fields := strings.Split(v.Member.(string), ":")
		block.Nonce = fields[0]
		block.PowHash = fields[1]
		block.MixDigest = fields[2]
		block.Timestamp, _ = strconv.ParseInt(fields[3], 10, 64)
		block.Difficulty, _ = strconv.ParseInt(fields[4], 10, 64)
		block.TotalShares, _ = strconv.ParseInt(fields[5], 10, 64)
		block.Login = fields[6]
		block.candidateKey = v.Member.(string)
		result = append(result, &block)
	}
	return result
}

func convertRewardResults(rows ...*redis.ZSliceCmd) []*RewardData {
	var result []*RewardData
	for _, row := range rows {
		for _, v := range row.Val() {
			// "amount:percent:immature:block.Hash:block.height"
			reward := RewardData{}
			reward.Timestamp = int64(v.Score)
			fields := strings.Split(v.Member.(string), ":")
			//block.UncleHeight, _ = strconv.ParseInt(fields[0], 10, 64)
			reward.BlockHash = fields[3]
			reward.Reward, _ = strconv.ParseInt(fields[0], 10, 64)
			reward.Percent, _ = strconv.ParseFloat(fields[1], 64)
			reward.Immature, _ = strconv.ParseBool(fields[2])
			reward.Height, _ = strconv.ParseInt(fields[4], 10, 64)
			result = append(result, &reward)
		}
	}
	return result
}

func convertBlockResults(rows ...*redis.ZSliceCmd) []*BlockData {
	var result []*BlockData
	for _, row := range rows {
		for _, v := range row.Val() {
			// "uncleHeight:orphan:nonce:blockHash:timestamp:diff:totalShares:rewardInWei"
			block := BlockData{}
			block.Height = int64(v.Score)
			block.RoundHeight = block.Height
			fields := strings.Split(v.Member.(string), ":")
			block.UncleHeight, _ = strconv.ParseInt(fields[0], 10, 64)
			block.Uncle = block.UncleHeight > 0
			block.Orphan, _ = strconv.ParseBool(fields[1])
			block.Nonce = fields[2]
			block.Hash = fields[3]
			block.Timestamp, _ = strconv.ParseInt(fields[4], 10, 64)
			block.Difficulty, _ = strconv.ParseInt(fields[5], 10, 64)
			block.TotalShares, _ = strconv.ParseInt(fields[6], 10, 64)
			block.RewardString = fields[7]
			block.ImmatureReward = fields[7]
			block.Login = fields[8]
			block.immatureKey = v.Member.(string)
			result = append(result, &block)
		}
	}
	return result
}

// Build per login workers's total shares map {'rig-1': 12345, 'rig-2': 6789, ...}
// TS => diff, id, ms
func convertWorkersStats(window int64, raw *redis.ZSliceCmd, blocks *redis.ZSliceCmd, login string, r *RedisClient) map[string]Worker {
	now := util.MakeTimestamp() / 1000
	workers := make(map[string]Worker)

	for _, v := range blocks.Val() {
		parts := strings.Split(v.Member.(string), ":")
		id := parts[2]
		worker := workers[id]
		worker.Blocks++
		workers[id] = worker
	}

	for _, v := range raw.Val() {
		parts := strings.Split(v.Member.(string), ":")
		share, _ := strconv.ParseInt(parts[0], 10, 64)
		id := parts[1]
		score := int64(v.Score)
		worker := workers[id]

		// Add for large window
		worker.TotalHR += share
		worker.ValidShares = int64(4)
		worker.ValidPercent = float64(0)
		worker.StalePercent = float64(0)
		worker.InvalidPercent = float64(0)
		worker.WorkerStatus = int64(0)
		worker.WorkerStatushas = int64(0)
		//worker.StatleShares = int64(4)
		//worker.InvalidShares = int64(4)

		// Add for small window if matches
		if score >= now-window {
			worker.HR += share
		}

		if worker.LastBeat < score {
			worker.LastBeat = score
		}
		if worker.startedAt > score || worker.startedAt == 0 {
			worker.startedAt = score
		}
		workers[id] = worker
	}
	return workers
}

func convertMinersStats(window int64, raw *redis.ZSliceCmd) (int64, map[string]Miner) {
	now := util.MakeTimestamp() / 1000
	miners := make(map[string]Miner)
	totalHashrate := int64(0)

	for _, v := range raw.Val() {
		parts := strings.Split(v.Member.(string), ":")
		share, _ := strconv.ParseInt(parts[0], 10, 64)
		id := parts[1]
		score := int64(v.Score)
		miner := miners[id]
		miner.HR += share

		if miner.LastBeat < score {
			miner.LastBeat = score
		}
		if miner.startedAt > score || miner.startedAt == 0 {
			miner.startedAt = score
		}
		miners[id] = miner
	}

	for id, miner := range miners {
		timeOnline := now - miner.startedAt
		if timeOnline < 600 {
			timeOnline = 600
		}

		boundary := timeOnline
		if timeOnline >= window {
			boundary = window
		}
		miner.HR = miner.HR / boundary

		if miner.LastBeat < (now - window/2) {
			miner.Offline = true
		}
		totalHashrate += miner.HR
		miners[id] = miner
	}
	return totalHashrate, miners
}

func convertPaymentsResults(raw *redis.ZSliceCmd) []map[string]interface{} {
	var result []map[string]interface{}
	for _, v := range raw.Val() {
		tx := make(map[string]interface{})
		tx["timestamp"] = int64(v.Score)
		fields := strings.Split(v.Member.(string), ":")
		tx["tx"] = fields[0]
		// Individual or whole payments row
		if len(fields) < 3 {
			tx["amount"], _ = strconv.ParseInt(fields[1], 10, 64)
		} else {
			tx["address"] = fields[1]
			tx["amount"], _ = strconv.ParseInt(fields[2], 10, 64)
		}
		result = append(result, tx)
	}
	var reverse []map[string]interface{}
	for i := len(result) - 1; i >= 0; i-- {
		reverse = append(reverse, result[i])
	}
	return result
}

func convertFindersResults(raw *redis.ZSliceCmd) []map[string]interface{} {
	var result []map[string]interface{}
	for _, v := range raw.Val() {
		miner := make(map[string]interface{})
		miner["blocks"] = int64(v.Score)
		miner["address"] = v.Member.(string)
		result = append(result, miner)
	}
	return result
}

/*
Timestamp  int64  `json:"x"`
TimeFormat string `json:"timeFormat"`
Amount     int64  `json:"amount"`
*/
func convertPaymentChartsResults(raw *redis.ZSliceCmd) []*PaymentCharts {
	var result []*PaymentCharts
	for _, v := range raw.Val() {
		pc := PaymentCharts{}
		pc.Timestamp = int64(v.Score)
		tm := time.Unix(pc.Timestamp, 0)
		pc.TimeFormat = tm.Format("2006-01-02") + " 00_00"
		fields := strings.Split(v.Member.(string), ":")
		pc.Amount, _ = strconv.ParseInt(fields[1], 10, 64)
		//fmt.Printf("%d : %s : %d \n", pc.Timestamp, pc.TimeFormat, pc.Amount)

		var chkAppend bool
		for _, pcc := range result {
			if pcc.TimeFormat == pc.TimeFormat {
				pcc.Amount += pc.Amount
				chkAppend = true
			}
		}
		if !chkAppend {
			pc.Timestamp -= int64(math.Mod(float64(v.Score), float64(86400)))
			result = append(result, &pc)
		}
	}
	var reverse []*PaymentCharts
	for i := len(result) - 1; i >= 0; i-- {
		reverse = append(reverse, result[i])
	}
	return reverse
}

func (r *RedisClient) GetCurrentHashrate(login string) (int64, error) {
	hashrate := r.client.HGet(r.formatKey("currenthashrate", login), "hashrate")
	if hashrate.Err() == redis.Nil {
		return 0, nil
	} else if hashrate.Err() != nil {
		return 0, hashrate.Err()
	}
	return hashrate.Int64()
}

// Need a function to delete on round end or whatever, and another function to get.
func (r *RedisClient) ResetWorkerShareStatus() {
	tx := r.client.Multi()
	defer tx.Close()

	tx.Exec(func() error {
		tx.HDel(r.formatKey("minerShare"))
		return nil
	})

	// THis should do it ay ?
	// fuck it
}

// Don't know if this will work, returning three values, but let's see

func (r *RedisClient) getSharesStatus(login string, id string) (int64, int64, int64, error) {
	valid_shares := r.client.HGet(r.formatKey("minerShare", login, id), "valid")
	stale_shares := r.client.HGet(r.formatKey("minerShare", login, id), "stale")
	invalid_shares := r.client.HGet(r.formatKey("minerShare", login, id), "invalid")

	if valid_shares.Err() == redis.Nil || stale_shares.Err() == redis.Nil || invalid_shares.Err() == redis.Nil {
		return 0, 0, 0, nil
	} else if valid_shares.Err() != nil || stale_shares.Err() != nil || invalid_shares.Err() != nil {
		return 0, 0, 0, valid_shares.Err()
	}

	v_c, _ := valid_shares.Int64()
	s_c, _ := stale_shares.Int64()
	i_c, _ := invalid_shares.Int64()
	return v_c, s_c, i_c, nil

}

//lets try to fuck without understanding and see if it works
func (r *RedisClient) WriteWorkerShareStatus(login string, id string, valid bool, stale bool, invalid bool) {

	valid_int := 0
	stale_int := 0
	invalid_int := 0
	if valid {
		valid_int = 1
	}
	if stale {
		stale_int = 1
	}
	if invalid {
		invalid_int = 1
	}

	// var after = time.Now().AddDate(0, 0, -1).Unix()
	//  var now = time.Now().Unix()
	// if(now >= after){
	//   tx.HDel(r.formatKey("minerShare", login, id))
	// }
	t := time.Now().Local()
	if t.Format("15:04:05") >= "23:59:00" {
		tx := r.client.Multi()
		defer tx.Close()
		tx.Exec(func() error {
			//tx.Del(r.formatKey("minerShare", login, id))
			tx.HSet(r.formatKey("minerShare", login, id), "valid", strconv.FormatInt(0, 10))
			tx.HSet(r.formatKey("minerShare", login, id), "stale", strconv.FormatInt(0, 10))
			tx.HSet(r.formatKey("minerShare", login, id), "invalid", strconv.FormatInt(0, 10))
			return nil
		})
	} else {
		// So, we need to initiate the tx object
		tx := r.client.Multi()
		defer tx.Close()

		tx.Exec(func() error {
			// OK, good, no need to read reset and add if i use Hset and HGet shit
			tx.HIncrBy(r.formatKey("minerShare", login, id), "valid", int64(valid_int))
			tx.HIncrBy(r.formatKey("minerShare", login, id), "stale", int64(stale_int))
			tx.HIncrBy(r.formatKey("minerShare", login, id), "invalid", int64(invalid_int))
			tx.HIncrBy(r.formatKey("chartsNum", "share", login), "valid", int64(valid_int))
			tx.HIncrBy(r.formatKey("chartsNum", "share", login), "stale", int64(stale_int)) // Would that work?

			return nil
		})
	} //end else
}

func (r *RedisClient) NumberStratumWorker(count int) {
	tx := r.client.Multi()
	defer tx.Close()

	tx.Exec(func() error {
		tx.Del(r.formatKey("WorkersTot"))
		tx.HIncrBy(r.formatKey("WorkersTot"), "workers", int64(count))
		//tx.HSet(r.formatKey("WorkersTotal"), "workers", int64(count))
		return nil
	})
}

func (r *RedisClient) StoreExchangeData(ExchangeData []map[string]interface{}) {

	tx := r.client.Multi()
	defer tx.Close()

	log.Printf("ExchangeData: %s", ExchangeData)

	for _, coindata := range ExchangeData {
		for key, value := range coindata {

			cmd := tx.HSet(r.formatKey("exchange", coindata["symbol"]), fmt.Sprintf("%v", key), fmt.Sprintf("%v", value))
			err := cmd.Err()
			if err != nil {
				log.Printf("Error while Storing %s : Key-%s , value-%s , Error : %v", coindata["symbol"], key, value, err)
			}

		}
	}
	log.Printf("Writing Exchange Data ")
	return
}

func (r *RedisClient) GetExchangeData(coinsymbol string) (map[string]string, error) {

	cmd := r.client.HGetAllMap(r.formatKey("exchange", coinsymbol))

	result, err := cmd.Result()

	if err != nil {
		return nil, err
	}

	return result, err
}
