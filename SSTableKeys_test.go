package main

import (
	"os"
	"sync"
	"testing"
)

func TestNewMetricsData(t *testing.T) {
	metrics := NewMetricsData()
	
	if metrics == nil {
		t.Fatal("NewMetricsData returned nil")
	}
	
	if metrics.Protocols == nil {
		t.Error("Protocols map not initialized")
	}
	
	if metrics.Ports == nil {
		t.Error("Ports map not initialized")
	}
	
	if metrics.IPv4 == nil {
		t.Error("IPv4 map not initialized")
	}
	
	if metrics.IPv6 == nil {
		t.Error("IPv6 map not initialized")
	}
	
	if metrics.TotalSize != 0 {
		t.Errorf("Expected TotalSize to be 0, got %d", metrics.TotalSize)
	}
}

func TestMetricsDataAddProtocol(t *testing.T) {
	metrics := NewMetricsData()
	
	metrics.AddProtocol(6, 10)
	metrics.AddProtocol(17, 5)
	metrics.AddProtocol(6, 15)
	
	metrics.mu.RLock()
	defer metrics.mu.RUnlock()
	
	if metrics.Protocols[6] != 25 {
		t.Errorf("Expected protocol 6 count to be 25, got %d", metrics.Protocols[6])
	}
	
	if metrics.Protocols[17] != 5 {
		t.Errorf("Expected protocol 17 count to be 5, got %d", metrics.Protocols[17])
	}
}

func TestMetricsDataAddPort(t *testing.T) {
	metrics := NewMetricsData()
	
	metrics.AddPort(80, 100)
	metrics.AddPort(443, 50)
	metrics.AddPort(80, 25)
	
	metrics.mu.RLock()
	defer metrics.mu.RUnlock()
	
	if metrics.Ports[80] != 125 {
		t.Errorf("Expected port 80 count to be 125, got %d", metrics.Ports[80])
	}
	
	if metrics.Ports[443] != 50 {
		t.Errorf("Expected port 443 count to be 50, got %d", metrics.Ports[443])
	}
}

func TestMetricsDataAddIPv4(t *testing.T) {
	metrics := NewMetricsData()
	
	metrics.AddIPv4("192.168.1.1", 10)
	metrics.AddIPv4("10.0.0.1", 5)
	metrics.AddIPv4("192.168.1.1", 15)
	
	metrics.mu.RLock()
	defer metrics.mu.RUnlock()
	
	if metrics.IPv4["192.168.1.1"] != 25 {
		t.Errorf("Expected 192.168.1.1 count to be 25, got %d", metrics.IPv4["192.168.1.1"])
	}
	
	if metrics.IPv4["10.0.0.1"] != 5 {
		t.Errorf("Expected 10.0.0.1 count to be 5, got %d", metrics.IPv4["10.0.0.1"])
	}
}

func TestMetricsDataAddIPv6(t *testing.T) {
	metrics := NewMetricsData()
	
	metrics.AddIPv6("::1", 20)
	metrics.AddIPv6("2001:db8::1", 10)
	metrics.AddIPv6("::1", 30)
	
	metrics.mu.RLock()
	defer metrics.mu.RUnlock()
	
	if metrics.IPv6["::1"] != 50 {
		t.Errorf("Expected ::1 count to be 50, got %d", metrics.IPv6["::1"])
	}
	
	if metrics.IPv6["2001:db8::1"] != 10 {
		t.Errorf("Expected 2001:db8::1 count to be 10, got %d", metrics.IPv6["2001:db8::1"])
	}
}

func TestMetricsDataAddSize(t *testing.T) {
	metrics := NewMetricsData()
	
	metrics.AddSize(1024)
	metrics.AddSize(2048)
	metrics.AddSize(512)
	
	metrics.mu.RLock()
	defer metrics.mu.RUnlock()
	
	expected := int64(1024 + 2048 + 512)
	if metrics.TotalSize != expected {
		t.Errorf("Expected total size to be %d, got %d", expected, metrics.TotalSize)
	}
}

func TestMetricsDataConcurrency(t *testing.T) {
	metrics := NewMetricsData()
	var wg sync.WaitGroup
	
	numGoroutines := 100
	incrementsPerGoroutine := 100
	
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < incrementsPerGoroutine; j++ {
				metrics.AddProtocol(6, 1)
				metrics.AddPort(80, 1)
				metrics.AddIPv4("192.168.1.1", 1)
				metrics.AddIPv6("::1", 1)
				metrics.AddSize(1)
			}
		}()
	}
	
	wg.Wait()
	
	expected := numGoroutines * incrementsPerGoroutine
	
	metrics.mu.RLock()
	defer metrics.mu.RUnlock()
	
	if metrics.Protocols[6] != expected {
		t.Errorf("Expected protocol count %d, got %d", expected, metrics.Protocols[6])
	}
	
	if metrics.Ports[80] != expected {
		t.Errorf("Expected port count %d, got %d", expected, metrics.Ports[80])
	}
	
	if metrics.IPv4["192.168.1.1"] != expected {
		t.Errorf("Expected IPv4 count %d, got %d", expected, metrics.IPv4["192.168.1.1"])
	}
	
	if metrics.IPv6["::1"] != expected {
		t.Errorf("Expected IPv6 count %d, got %d", expected, metrics.IPv6["::1"])
	}
	
	if metrics.TotalSize != int64(expected) {
		t.Errorf("Expected total size %d, got %d", expected, metrics.TotalSize)
	}
}

func TestProcessKeyValue(t *testing.T) {
	metrics := NewMetricsData()
	
	// Test protocol key
	protocolKey := []byte{1, 6}
	processKeyValue(1, protocolKey, 12, metrics)
	
	// Test port key
	portKey := []byte{2, 0, 80}
	processKeyValue(2, portKey, 8, metrics)
	
	// Test IPv4 key
	ipv4Key := []byte{4, 192, 168, 1, 1}
	processKeyValue(4, ipv4Key, 16, metrics)
	
	// Test IPv6 key
	ipv6Key := []byte{6, 0x20, 0x01, 0x0d, 0xb8, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}
	processKeyValue(6, ipv6Key, 20, metrics)
	
	// Test unknown key type (should be ignored)
	unknownKey := []byte{99, 1, 2, 3, 4}
	processKeyValue(99, unknownKey, 4, metrics)
	
	metrics.mu.RLock()
	defer metrics.mu.RUnlock()
	
	if metrics.Protocols[6] != 3 { // 12/4 = 3
		t.Errorf("Expected protocol count 3, got %d", metrics.Protocols[6])
	}
	
	if metrics.Ports[80] != 2 { // 8/4 = 2
		t.Errorf("Expected port count 2, got %d", metrics.Ports[80])
	}
	
	if metrics.IPv4["192.168.1.1"] != 4 { // 16/4 = 4
		t.Errorf("Expected IPv4 count 4, got %d", metrics.IPv4["192.168.1.1"])
	}
	
	if metrics.IPv6["2001:db8::1"] != 5 { // 20/4 = 5
		t.Errorf("Expected IPv6 count 5, got %d", metrics.IPv6["2001:db8::1"])
	}
}

func TestParseArgsValid(t *testing.T) {
	// Save original args
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	
	// Test basic folder path
	os.Args = []string{"SSTableKeys", "/path/to/IDX0"}
	config, err := parseArgs()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	if config.FolderPath != "/path/to/IDX0" {
		t.Errorf("Expected folder path '/path/to/IDX0', got '%s'", config.FolderPath)
	}
	
	if config.DataFolderPath != "/path/to/PKT0" {
		t.Errorf("Expected data folder path '/path/to/PKT0', got '%s'", config.DataFolderPath)
	}
	
	if config.StartDate != -1 {
		t.Errorf("Expected StartDate -1, got %d", config.StartDate)
	}
	
	if config.EndDate != -1 {
		t.Errorf("Expected EndDate -1, got %d", config.EndDate)
	}
	
	if config.Workers != concurrentWorkers {
		t.Errorf("Expected Workers %d, got %d", concurrentWorkers, config.Workers)
	}
}

func TestParseArgsWithTimestamps(t *testing.T) {
	// Save original args
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	
	// Test with valid timestamps
	os.Args = []string{"SSTableKeys", "/path/to/IDX0", "1234567890", "1234567900"}
	config, err := parseArgs()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	if config.StartDate != 1234567890 {
		t.Errorf("Expected StartDate 1234567890, got %d", config.StartDate)
	}
	
	if config.EndDate != 1234567900 {
		t.Errorf("Expected EndDate 1234567900, got %d", config.EndDate)
	}
}

func TestParseArgsInvalidArgs(t *testing.T) {
	// Save original args
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	
	tests := []struct {
		name string
		args []string
	}{
		{"no args", []string{"SSTableKeys"}},
		{"too many args", []string{"SSTableKeys", "path", "start", "end", "extra"}},
		{"three args", []string{"SSTableKeys", "path", "start"}},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Args = tt.args
			_, err := parseArgs()
			if err == nil {
				t.Error("Expected error for invalid arguments")
			}
		})
	}
}

func TestParseArgsInvalidTimestamps(t *testing.T) {
	// Save original args
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	
	tests := []struct {
		name  string
		start string
		end   string
	}{
		{"invalid start format", "123456789", "1234567890"},
		{"invalid end format", "1234567890", "123456789"},
		{"non-numeric start", "abcdefghij", "1234567890"},
		{"non-numeric end", "1234567890", "abcdefghij"},
		{"end before start", "1234567900", "1234567890"},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Args = []string{"SSTableKeys", "/path/to/IDX0", tt.start, tt.end}
			_, err := parseArgs()
			if err == nil {
				t.Error("Expected error for invalid timestamps")
			}
		})
	}
}

func TestRegexFilenameMatching(t *testing.T) {
	tests := []struct {
		filename string
		expected bool
	}{
		{"1234567890123456", true},
		{"0000000000000000", true},
		{"9999999999999999", true},
		{"123456789012345", false},   // too short
		{"12345678901234567", false}, // too long
		{"abcdefghijklmnop", false},  // non-numeric
		{"123456789012345a", false},  // contains letter
		{"", false},                  // empty
	}
	
	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			result := re.MatchString(tt.filename)
			if result != tt.expected {
				t.Errorf("Expected %v for filename '%s', got %v", tt.expected, tt.filename, result)
			}
		})
	}
}