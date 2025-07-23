package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/golang/leveldb/table"
)

const (
	concurrentWorkers  = 4
	majorVersionNumber = 2
)

var re = regexp.MustCompile(`^\d{16}$`)

type Config struct {
	FolderPath     string
	DataFolderPath string
	StartDate      int
	EndDate        int
	Workers        int
}

type MetricsData struct {
	mu        sync.RWMutex
	Protocols map[int]int
	Ports     map[int]int
	IPv4      map[string]int
	IPv6      map[string]int
	TotalSize int64
}

func NewMetricsData() *MetricsData {
	return &MetricsData{
		Protocols: make(map[int]int),
		Ports:     make(map[int]int),
		IPv4:      make(map[string]int),
		IPv6:      make(map[string]int),
	}
}

func (m *MetricsData) AddProtocol(proto int, count int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Protocols[proto] += count
}

func (m *MetricsData) AddPort(port int, count int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Ports[port] += count
}

func (m *MetricsData) AddIPv4(ip string, count int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.IPv4[ip] += count
}

func (m *MetricsData) AddIPv6(ip string, count int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.IPv6[ip] += count
}

func (m *MetricsData) AddSize(size int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.TotalSize += size
}

func parseProtocolKey(key []byte) (int, int) {
	proto := int(key[1])
	count := len(key[4:]) / 4
	return proto, count
}

func parsePortKey(key []byte) (int, int) {
	port := int(binary.BigEndian.Uint16([]byte{key[1], key[2]}))
	count := len(key[4:]) / 4
	return port, count
}

func parseIPv4Key(key []byte) (string, int) {
	ipv4 := net.IP{key[1], key[2], key[3], key[4]}
	count := len(key[4:]) / 4
	return ipv4.String(), count
}

func parseIPv6Key(key []byte) (string, int) {
	ipv6 := net.IP{
		key[1], key[2], key[3], key[4],
		key[5], key[6], key[7], key[8],
		key[9], key[10], key[11], key[12],
		key[13], key[14], key[15], key[16],
	}
	count := len(key[16:]) / 4
	return ipv6.String(), count
}

func processKeyValue(keyType byte, key []byte, valueLen int, metrics *MetricsData) {
	switch keyType {
	case 1:
		proto := int(key[1])
		count := valueLen / 4
		metrics.AddProtocol(proto, count)
	case 2:
		port := int(binary.BigEndian.Uint16([]byte{key[1], key[2]}))
		count := valueLen / 4
		metrics.AddPort(port, count)
	case 4:
		ipv4 := net.IP{key[1], key[2], key[3], key[4]}
		count := valueLen / 4
		metrics.AddIPv4(ipv4.String(), count)
	case 6:
		ipv6 := net.IP{
			key[1], key[2], key[3], key[4],
			key[5], key[6], key[7], key[8],
			key[9], key[10], key[11], key[12],
			key[13], key[14], key[15], key[16],
		}
		count := valueLen / 4
		metrics.AddIPv6(ipv6.String(), count)
	}
}

func worker(id int, jobs <-chan string, config *Config, metrics *MetricsData, wg *sync.WaitGroup) {
	for filename := range jobs {
		if re.MatchString(filename) {
			if config.StartDate != -1 && config.EndDate != -1 {
				fileNameShort, err := strconv.Atoi(filename[:10])
				if err == nil && fileNameShort >= config.StartDate-60 && config.EndDate+60 >= fileNameShort {
					readIndexFile(filename, config, metrics)
				}
			} else {
				readIndexFile(filename, config, metrics)
			}
		}
		wg.Done()
	}
}

func readIndexFile(filename string, config *Config, metrics *MetricsData) {
	filePath := fmt.Sprintf("%s/%s", config.FolderPath, filename)
	fh, fhErr := os.Open(filePath)
	if fhErr != nil {
		return
	}
	defer fh.Close()

	ss := table.NewReader(fh, nil)
	versions, err := ss.Get([]byte{0}, nil)
	if err != nil {
		return
	}
	if len(versions) != 8 {
		return
	}
	if major := binary.BigEndian.Uint32(versions[:4]); major != majorVersionNumber {
		return
	}

	iter := ss.Find([]byte{}, nil)
	defer iter.Close()

	for iter.Next() {
		foundKey := iter.Key()
		if len(foundKey) == 0 {
			continue
		}
		keyType := foundKey[0]
		processKeyValue(keyType, foundKey, len(iter.Value()), metrics)
	}

	dataFileStat, err := os.Stat(fmt.Sprintf("%s/%s", config.DataFolderPath, filename))
	if err == nil {
		metrics.AddSize(dataFileStat.Size())
	}
}

func parseArgs() (*Config, error) {
	if len(os.Args) != 2 && len(os.Args) != 4 {
		return nil, fmt.Errorf("usage: %s <folder_path> [start_timestamp end_timestamp]", os.Args[0])
	}

	config := &Config{
		FolderPath:     os.Args[1],
		DataFolderPath: strings.Replace(os.Args[1], "IDX0", "PKT0", 1),
		StartDate:      -1,
		EndDate:        -1,
		Workers:        concurrentWorkers,
	}

	if len(os.Args) == 4 {
		inputTimestampArgs := regexp.MustCompile(`^\d{10}$`)
		if !inputTimestampArgs.MatchString(os.Args[2]) || !inputTimestampArgs.MatchString(os.Args[3]) {
			return nil, fmt.Errorf("invalid timestamp format")
		}

		startDate, err := strconv.Atoi(os.Args[2])
		if err != nil {
			return nil, fmt.Errorf("invalid start timestamp: %v", err)
		}

		endDate, err := strconv.Atoi(os.Args[3])
		if err != nil {
			return nil, fmt.Errorf("invalid end timestamp: %v", err)
		}

		if endDate <= startDate {
			return nil, fmt.Errorf("start timestamp must be smaller than end timestamp")
		}

		config.StartDate = startDate
		config.EndDate = endDate
	}

	return config, nil
}

func processFiles(config *Config, metrics *MetricsData) error {
	files, err := ioutil.ReadDir(config.FolderPath)
	if err != nil {
		return fmt.Errorf("failed to read directory: %v", err)
	}

	jobs := make(chan string, 100)
	var wg sync.WaitGroup

	for w := 1; w <= config.Workers; w++ {
		go worker(w, jobs, config, metrics, &wg)
	}

	for _, file := range files {
		wg.Add(1)
		jobs <- file.Name()
	}

	close(jobs)
	wg.Wait()

	return nil
}

func outputResults(metrics *MetricsData) error {
	metrics.mu.RLock()
	defer metrics.mu.RUnlock()

	protocolsOut, err := json.Marshal(metrics.Protocols)
	if err != nil {
		return fmt.Errorf("failed to marshal protocols: %v", err)
	}

	portsOut, err := json.Marshal(metrics.Ports)
	if err != nil {
		return fmt.Errorf("failed to marshal ports: %v", err)
	}

	ipv4Out, err := json.Marshal(metrics.IPv4)
	if err != nil {
		return fmt.Errorf("failed to marshal IPv4: %v", err)
	}

	ipv6Out, err := json.Marshal(metrics.IPv6)
	if err != nil {
		return fmt.Errorf("failed to marshal IPv6: %v", err)
	}

	out := fmt.Sprintf(`{"totalSize": %d, "protocols":%s,"ports":%s,"ipv4":%s,"ipv6":%s}`,
		metrics.TotalSize,
		string(protocolsOut),
		string(portsOut),
		string(ipv4Out),
		string(ipv6Out))

	fmt.Println(out)
	return nil
}

func main() {
	config, err := parseArgs()
	if err != nil {
		log.Fatal(err)
	}

	metrics := NewMetricsData()

	if err := processFiles(config, metrics); err != nil {
		log.Fatal(err)
	}

	if err := outputResults(metrics); err != nil {
		log.Fatal(err)
	}
}
